// Package api exposes the HTTP surface of the lumemor server.
package api

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	apigen "github.com/welpher/lumemor/server/internal/api/gen"
	"github.com/welpher/lumemor/server/internal/api/static"
	"github.com/welpher/lumemor/server/internal/auth"
	"github.com/welpher/lumemor/server/internal/buildinfo"
	dbgen "github.com/welpher/lumemor/server/internal/db/gen"
	"github.com/welpher/lumemor/server/internal/events"
	"github.com/welpher/lumemor/server/internal/ml"
	"github.com/welpher/lumemor/server/internal/web"
)

// Deps carries the runtime dependencies of the HTTP handlers. It must be
// non-nil; Pool may only be nil when no /api/v1 route is served.
type Deps struct {
	// Pool is the PostgreSQL connection pool used by every API route.
	Pool *pgxpool.Pool
	// Logger receives handler-level diagnostics; nil falls back to
	// slog.Default().
	Logger *slog.Logger
	// Version is the server build version reported by GET /server/info;
	// the empty string falls back to buildinfo.Version.
	Version string
	// Initialized reports whether first-run setup has completed (an
	// administrator account exists); GET / routes on it. Production
	// wires it to the CountAdmins query, tests inject a stub; nil means
	// "not initialized", keeping the root on the wizard.
	Initialized func(ctx context.Context) bool
	// Events is the in-process bus backing the SSE activity stream
	// (GET /api/v1/events); nil keeps that endpoint answering 500
	// until the caller wires a bus.
	Events *events.Bus
	// DataDir is the photo data root: the upload endpoint stages
	// multipart file parts in its tmp/ directory and finalizes
	// accepted files into its content-addressed originals tree. It
	// must name a layout prepared by storage.EnsureLayout.
	DataDir string
	// UploadLimitBytes caps each uploaded file of POST /assets; the
	// streaming receiver aborts over-limit transfers mid-stream with
	// 413 ASSET_TOO_LARGE. Zero rejects every upload -- the caller
	// (production wiring, tests) is responsible for a usable value.
	UploadLimitBytes int64
	// Jobs enqueues pipeline jobs for uploaded assets: the upload
	// handler enqueues the metadata and thumbnail jobs (and, for a
	// video upload or a Live Photo's video-family component, the
	// transcode job) after its transaction commits.
	// Nil keeps the upload working without a queue (tests, wiring
	// order); the assets simply stay not-ready until one runs.
	Jobs AssetJobEnqueuer
	// ModelDownloads starts the default model group's download runs:
	// the download endpoint enqueues the model_download job through
	// this seam. Nil keeps the endpoint answering 500
	// MODEL_DOWNLOAD_FAILED (no queue wired -- tests, wiring order);
	// production passes the jobs queue, which implements it
	// structurally.
	ModelDownloads ModelDownloadEnqueuer
	// Models is the instance's SHARED model registry: the model
	// management endpoints list, activate and scan through it. Nil
	// keeps those endpoints answering 500 INTERNAL (no registry wired
	// -- tests, wiring order); production passes the registry the job
	// queue already shares.
	Models *ml.Registry
	// ModelsDir is the model store root the scan endpoint walks. It
	// must be the registry's own models dir: a model outside it is
	// refused by the registry's scan (the registered file_path must
	// resolve against the registry's root), so a divergence fails
	// loudly instead of registering unusable rows.
	ModelsDir string
	// MlReruns starts the ML rerun runs of a slot: the rerun endpoint
	// enqueues the rerun driver job through this seam. Nil keeps the
	// endpoint answering 500 INTERNAL (no queue wired -- tests,
	// wiring order); production passes the jobs queue, which
	// implements it structurally.
	MlReruns MlRerunEnqueuer
	// PlacesReruns starts the place re-resolution runs: the
	// re-resolution endpoint enqueues the places rerun job through
	// this seam. Nil keeps the endpoint answering 500 INTERNAL (no
	// queue wired -- tests, wiring order); production passes the jobs
	// queue, which implements it structurally.
	PlacesReruns PlacesRerunEnqueuer
}

// AssetJobEnqueuer enqueues the asset pipeline's jobs. It is the seam
// between the upload handler and the job queue: production passes the
// jobs package's Queue, which implements it structurally.
type AssetJobEnqueuer interface {
	// EnqueueMetadata inserts the metadata job of the asset. It must
	// only be called once the asset's rows are committed.
	EnqueueMetadata(ctx context.Context, assetID uuid.UUID) error
	// EnqueueThumbnail inserts the thumbnail job of the asset (the
	// three AVIF sizes of an image, the poster frame of a video). Like
	// the metadata job it must only be called once the asset's rows
	// are committed.
	EnqueueThumbnail(ctx context.Context, assetID uuid.UUID) error
	// EnqueueTranscode inserts the transcode job of a VIDEO asset
	// (the playable web-safe MP4) -- a standalone video, or a Live
	// Photo's video-family component (the one job its counter
	// counts). The upload handler gates the call on the asset's own
	// family, matching the pending_jobs initialization.
	EnqueueTranscode(ctx context.Context, assetID uuid.UUID) error
}

// ModelDownloadEnqueuer starts the default model group's download. It
// is the seam between the download endpoint and the job queue:
// production passes the jobs package's Queue, which implements it
// structurally.
type ModelDownloadEnqueuer interface {
	// EnqueueModelDownload creates the download's job_runs row and
	// enqueues the River job in one transaction, returning the run's
	// identifier.
	EnqueueModelDownload(ctx context.Context) (uuid.UUID, error)
}

// MlRerunEnqueuer starts one ML rerun of a slot. It is the seam
// between the rerun endpoint and the job queue: production passes the
// jobs package's Queue, which implements it structurally.
type MlRerunEnqueuer interface {
	// EnqueueMlRerun creates the rerun's job_runs row and enqueues the
	// River driver job in one transaction, returning the run's
	// identifier.
	EnqueueMlRerun(ctx context.Context, slot ml.Slot) (uuid.UUID, error)
}

// PlacesRerunEnqueuer starts one place re-resolution run. It is the
// seam between the re-resolution endpoint and the job queue:
// production passes the jobs package's Queue, which implements it
// structurally.
type PlacesRerunEnqueuer interface {
	// EnqueuePlacesRerun creates the run's job_runs row and enqueues
	// the River job in one transaction, returning the run's
	// identifier. The enqueue refuses a server without the place
	// reference data: there is no grid to re-resolve against.
	EnqueuePlacesRerun(ctx context.Context) (uuid.UUID, error)
}

// NewRouter builds the HTTP routes of the server: the operational
// endpoints (/healthz, /metrics), the first-run wizard (/setup), the
// embedded single-page web app (/ and, through the SPA catch-all, its
// hashed assets and the client-side-route fallback) and the generated
// /api/v1 API surface.
//
// The security headers (nosniff, Referrer-Policy) run as ROOT-level
// middleware: every response of the server carries them -- pages,
// static assets, the SPA shell, the API, the operational endpoints
// (the M2a handoff's HTML-surface item; the CSP arrives with the
// map milestone). The /api/v1 pipeline order is pinned and
// security-relevant: after the root middleware, the origin check
// guards every state-changing API request, then the route groups.
// The JSON body limit and the per-request handler timeout apply only
// to the JSON surface (nothing downstream of limitJSONBody may
// buffer an unbounded body, and http.TimeoutHandler -- a handler
// that outlives it answers 503 -- bounds every non-streaming JSON
// handler): the authed groups sit behind RequireSession, the SSE
// stream group runs outside the timeout, and the upload group runs
// outside both the JSON body limit and the timeout -- a multipart
// upload is not JSON, and a multi-gigabyte video legitimately
// outlives any timer; the streaming receiver's own size limit
// governs instead.
func NewRouter(deps *Deps) *chi.Mux {
	return newRouter(deps, handlerTimeout)
}

// newRouter builds the router with a configurable timeout for the
// non-streaming handler groups. NewRouter pins that bound to
// handlerTimeout; tests shorten it so the timeout behavior itself is
// exercisable in bounded time.
func newRouter(deps *Deps, timeout time.Duration) *chi.Mux {
	s := &server{
		pool:    deps.Pool,
		queries: dbgen.New(deps.Pool),
		version: deps.Version,
		logger:  deps.Logger,
		events:  deps.Events,
		dataDir: deps.DataDir,
		// The upload limit passes through as configured: zero is a
		// valid (reject-everything) configuration, not a request for
		// a default -- production wiring owns the value.
		uploadLimit: deps.UploadLimitBytes,
		// The job queue seam: nil is a valid (no pipeline) wiring.
		jobs: deps.Jobs,
		// The model download seam: nil is a valid (no queue) wiring;
		// the download endpoint answers its own failure code then.
		modelDownloads: deps.ModelDownloads,
		// The shared model registry and its models dir: nil is a
		// valid (no registry) wiring; the model management endpoints
		// answer the generic internal error then.
		models:    deps.Models,
		modelsDir: deps.ModelsDir,
		// The rerun enqueue seam: nil is a valid (no queue) wiring.
		mlReruns: deps.MlReruns,
		// The places re-resolution enqueue seam: nil is a valid (no
		// queue) wiring.
		placesReruns: deps.PlacesReruns,
		// In-memory per process: a restart clears the login locks, an
		// accepted trade for not growing a store into the auth path.
		limiter: auth.NewLimiter(nil),
		// The same trade for the pairing budget: a restart forgets the
		// day's count, the daily window re-arms at UTC midnight.
		pairingBudget: auth.NewDailyLimiter(pairingDailyLimit, nil),
	}
	if s.version == "" {
		s.version = buildinfo.Version
	}
	if s.logger == nil {
		s.logger = slog.Default()
	}
	// A nil Initialized keeps the root on the wizard: only production
	// wiring and tests that exercise / need the real predicate.
	initialized := deps.Initialized
	if initialized == nil {
		initialized = func(context.Context) bool { return false }
	}

	r := chi.NewRouter()
	// Root-level middleware: the security headers cover every response
	// the server emits -- pages, the SPA shell and its assets, the API,
	// and the operational endpoints (M2a handoff's HTML-surface item).
	r.Use(setSecurityHeaders)
	r.Get("/healthz", handleHealthz)
	r.Handle("/metrics", promhttp.Handler())
	r.Get("/setup", handleSetupPage)
	// The embedded single-page web app. The root serves its shell (the
	// app div mounts the client-side router) once setup is done; before
	// that the first-run redirect in the handler below keeps the wizard
	// as the face of a fresh instance.
	spa := web.Handler()
	r.Get("/", func(w http.ResponseWriter, req *http.Request) {
		if !initialized(req.Context()) {
			http.Redirect(w, req, "/setup", http.StatusFound)
			return
		}
		spa.ServeHTTP(w, req)
	})
	r.Route("/api/v1", func(r chi.Router) {
		// The origin check guards every state-changing API request;
		// the security headers already ran at the router root. The
		// JSON body limit moved into the JSON group below so multipart
		// uploads can stream past it (the receiver's own size limit
		// governs them).
		r.Use(checkOrigin)

		// The wrapper reuses the generated request binders (the path
		// uuid, the Last-Event-ID header) and routes their failures
		// into the shared error envelope through binderError. Routes
		// are mounted by hand into groups because the generated
		// HandlerFromMux registers every operation on one flat
		// router; TestAPIRoutesCoverContract guards the mounted set
		// against contract drift.
		wrapper := &apigen.ServerInterfaceWrapper{
			Handler:          s,
			ErrorHandlerFunc: s.binderError,
		}

		// The JSON surface: small structured request bodies, bounded
		// by the 1 MiB cap and (for everything but the SSE stream)
		// the handler timeout.
		r.Group(func(r chi.Router) {
			r.Use(limitJSONBody)

			// Public group: the unauthenticated entry points. Login and
			// the pairing exchange carry their own credentials; server
			// info and the one-time wizard precede any account. The
			// group runs inside the handler timeout like every
			// non-streaming group.
			r.Group(func(r chi.Router) {
				r.Use(timeoutMiddleware(timeout))
				r.Post("/auth/login", wrapper.Login)
				r.Post("/auth/pairing/consume", wrapper.ConsumePairingToken)
				r.Get("/server/info", wrapper.GetServerInfo)
				r.Post("/server/setup/admin", wrapper.CreateAdmin)
			})

			// Authed non-streaming group: session management, pairing
			// generation and the asset reads behind RequireSession, all
			// inside the handler timeout. The timeout wraps the session
			// middleware too, so even a wedged session lookup cannot pin a
			// connection. The asset detail, the aggregated timeline and
			// the thumbnail serving belong here: all answer from bounded
			// row reads plus bounded bytes -- the detail and the timeline
			// are small JSON documents (the timeline's page size is
			// capped by the contract), the derived files are re-encoded
			// copies of at most a few megabytes.
			r.Group(func(r chi.Router) {
				r.Use(timeoutMiddleware(timeout), s.requireSession)
				r.Post("/auth/logout", wrapper.Logout)
				r.Get("/auth/sessions", wrapper.ListSessions)
				r.Delete("/auth/sessions/{id}", wrapper.RevokeSession)
				r.Post("/auth/pairing/token", wrapper.CreatePairingToken)
				r.Get("/assets/by-device-id/{deviceAssetId}", wrapper.GetAssetByDeviceId)
				r.Get("/assets/timeline", wrapper.GetTimeline)
				// The batch and curation views (M2c Task 4): the
				// set-semantics batch write, and the favorites and
				// hidden listings. The static segments win over the
				// /assets/{id} parameter below, so mounting them in
				// this group keeps every asset read a bounded row
				// read plus a bounded JSON document.
				r.Post("/assets/batch", wrapper.BatchAssets)
				r.Get("/assets/favorites", wrapper.GetFavorites)
				r.Get("/assets/hidden", wrapper.GetHidden)
				r.Get("/assets/{id}", wrapper.GetAsset)
				r.Patch("/assets/{id}", wrapper.PatchAsset)
				r.Get("/assets/{id}/thumbnail", wrapper.GetAssetThumbnail)
				// The trash lifecycle endpoints (M2c Task 3): the
				// owner-scoped listing and restore, and the explicit
				// permanent delete. All three are bounded operations
				// (row reads and writes plus metadata-only file
				// removals), so they belong in this group like every
				// other JSON handler.
				r.Get("/trash", wrapper.GetTrash)
				r.Post("/trash/{id}/restore", wrapper.RestoreAsset)
				r.Delete("/assets/{id}", wrapper.DeleteAsset)
				// The album surface reads (M3c Task 1): the wall
				// (the caller's own and shared albums) and one
				// album's detail with its keyset-paged items. Both
				// are bounded row reads (the detail's page size is
				// capped by the contract); the wall's derived
				// covers are a second bounded read of at most four
				// candidates per album.
				r.Get("/albums", wrapper.ListAlbums)
				r.Get("/albums/{albumId}", wrapper.GetAlbum)
				// The album surface writes (M3c Task 2): virtual
				// album creation, the property patch, the delete
				// and the membership curation. All are bounded row
				// writes -- the membership lists are capped at 200
				// ids by the contract, and the cascades ride the
				// row deletes -- so they belong in this group like
				// every other bounded JSON handler. The static
				// /albums/{albumId}/items segment never contends
				// with the /albums/{albumId} routes (a longer path
				// shape).
				r.Post("/albums", wrapper.CreateAlbum)
				r.Patch("/albums/{albumId}", wrapper.UpdateAlbum)
				r.Delete("/albums/{albumId}", wrapper.DeleteAlbum)
				r.Post("/albums/{albumId}/items", wrapper.AddAlbumItems)
				r.Delete("/albums/{albumId}/items", wrapper.RemoveAlbumItems)
				// The album sharing face (M3c Task 3): the view-ACL
				// grant and revoke, and the instance's minimal
				// member list the share dialog picks its grantee
				// from. The grant is one bounded row write; the
				// revoke deletes one row; the member list is one
				// bounded read of the small users table. The
				// /albums/{albumId}/shares segments never contend
				// with the /albums/{albumId} routes (a longer path
				// shape), and /users never contends with the
				// /admin/users group (a different prefix).
				r.Post("/albums/{albumId}/shares", wrapper.GrantAlbumShare)
				r.Delete("/albums/{albumId}/shares/{userId}", wrapper.RevokeAlbumShare)
				r.Get("/users", wrapper.ListInstanceUsers)
				// The timeline-preference face (M3c Task 4): the
				// caller's per-user "keep this album out of my
				// timeline" switch -- one bounded row upsert and
				// one bounded row delete over an album the caller
				// can see, self-affecting state that publishes no
				// event. The static /timeline-preference segment
				// never contends with the /albums/{albumId}
				// routes (a longer path shape).
				r.Put("/albums/{albumId}/timeline-preference", wrapper.SetAlbumTimelinePreference)
				r.Delete("/albums/{albumId}/timeline-preference", wrapper.ResetAlbumTimelinePreference)
				// The people surface reads (M3b Task 1): the named
				// region, the unnamed region's keyset pages and one
				// cluster's detail. All are bounded row reads (the
				// unnamed page size is capped by the contract); the
				// static /people/unnamed segment wins over the
				// /people/{clusterId} parameter below.
				r.Get("/people", wrapper.ListPeople)
				r.Get("/people/unnamed", wrapper.ListUnnamedPeople)
				r.Get("/people/{clusterId}", wrapper.GetPersonCluster)
				// The places surface read (M3d Task 1): the caller's city
				// grouping -- two bounded row reads (the grouping with its
				// honest per-caller counts, then at most four cover
				// candidates per listed place), the same cost class as the
				// people listing.
				r.Get("/places", wrapper.ListPlaces)
				// The people surface writes (M3b Task 2): the caller's
				// label and hidden switches, the cluster merge and the
				// single face move. All are bounded row reads and
				// writes (the merge touches at most 50 source
				// clusters); the static /people/merge segment never
				// contends with the /people/{clusterId} routes (a
				// different method and path shape).
				r.Put("/people/{clusterId}/label", wrapper.LabelPerson)
				r.Delete("/people/{clusterId}/label", wrapper.UnlabelPerson)
				r.Put("/people/{clusterId}/hidden", wrapper.SetPersonHidden)
				r.Post("/people/merge", wrapper.MergePeople)
				r.Post("/faces/{faceId}/move", wrapper.MoveFace)
				// The face crop pixels (M3b Task 3): the 256x256 AVIF
				// crop of one face, derived at query time from its
				// asset's view derivative -- one bounded decode of a
				// serving copy plus one AVIF encode, the same cost
				// class as the thumbnail job's single-file
				// derivations, and cached on the client by the
				// response's immutable ETag.
				r.Get("/faces/{faceId}/thumbnail", wrapper.GetFaceThumbnail)
			})

			// Authed admin group (M2d Task 8): the user management
			// surface sits behind requireSession AND the admin-role
			// guard, inside the handler timeout like every other
			// bounded JSON handler (small row reads and writes).
			r.Group(func(r chi.Router) {
				r.Use(timeoutMiddleware(timeout), s.requireSession, s.requireAdmin)
				r.Get("/admin/users", wrapper.ListUsers)
				r.Post("/admin/users", wrapper.CreateUser)
				r.Patch("/admin/users/{id}", wrapper.UpdateUser)
				// The model download trigger (M3a Task 3): the
				// default group's first-boot download. The handler is
				// one bounded row read plus one enqueue; the heavy
				// work lives on the queue.
				r.Post("/admin/models/download", wrapper.DownloadModels)
				// The model management and job panel surface (M3a
				// Task 4): the listing, the activation switch, the
				// rerun trigger and the custom model scan, plus the
				// job runs listing. All are bounded row reads and
				// writes (the rerun's heavy work lives on the queue
				// behind its enqueue); the scan route's static
				// segment never contends with the {id} routes (a
				// different method, and two segments for the rerun).
				r.Get("/admin/models", wrapper.ListModels)
				r.Patch("/admin/models/{id}", wrapper.ActivateModel)
				r.Post("/admin/models/{id}/rerun", wrapper.RerunModel)
				r.Post("/admin/models/scan", wrapper.ScanModels)
				// The places re-resolution trigger (M3d Task 1): the
				// places-data-update path's run start. The handler is
				// one bounded row read plus one enqueue; the walk over
				// every GPS-bearing asset lives on the queue.
				r.Post("/admin/places/reresolve", wrapper.ReresolvePlaces)
				r.Get("/admin/jobs", wrapper.ListJobs)
			})

			// Authed streaming group: the SSE activity stream. It
			// requires a live session like the group above but
			// deliberately runs OUTSIDE the handler timeout:
			// http.TimeoutHandler buffers the response until the
			// handler returns (it implements no Flusher) and cancels
			// the request context at the deadline, which would stall
			// and then cut a stream that is meant to outlive any
			// bound; its liveness bound is the client connection,
			// not a timer.
			r.Group(func(r chi.Router) {
				r.Use(s.requireSession)
				r.Get("/events", wrapper.GetEvents)
			})
		})

		// Authed upload group: POST /assets streams multipart bodies
		// that are neither JSON nor bounded by any timer, so it sits
		// outside limitJSONBody and the handler timeout. It still
		// requires a session like every authed route; the receiver's
		// per-file size limit is the bound that governs the body.
		r.Group(func(r chi.Router) {
			r.Use(s.requireSession)
			r.Post("/assets", wrapper.UploadAsset)
		})

		// Authed media download group: GET /assets/{id}/original and
		// GET /assets/{id}/play stream media files whose size is
		// bounded only by the per-file upload limit (2 GiB by default
		// for the original; the transcoded play copy is a re-encode of
		// it) and whose transfer runs at client speed. They require a
		// session like every authed route but deliberately run
		// OUTSIDE the handler timeout, mirroring the upload and SSE
		// exemptions: http.TimeoutHandler buffers the whole response
		// until the handler returns (see timeoutMiddleware), so a
		// wrapped media stream would first read the entire file into
		// memory and only then let its first byte reach the client --
		// an unbounded per-request memory commitment. The handlers
		// themselves stay bounded: one row read, one file open, then
		// http.ServeContent streams the bytes.
		r.Group(func(r chi.Router) {
			r.Use(s.requireSession)
			r.Get("/assets/{id}/original", wrapper.GetAssetOriginal)
			r.Get("/assets/{id}/play", wrapper.PlayAsset)
		})
	})
	// The SPA catch-all, registered last: content-hashed _app/ assets
	// (immutable cache) and the shell fallback for every other unknown
	// GET path, so unknown routes belong to the client-side router.
	// chi matches the static routes above and the /api/v1 mount ahead
	// of this wildcard; an unknown /api/v1 path 404s inside its own
	// router and never falls through to the shell.
	r.Get("/*", spa.ServeHTTP)
	return r
}

// server implements apigen.ServerInterface on top of the injected Deps.
// It embeds apigen.Unimplemented so that every endpoint not yet
// overridden answers 501; implemented endpoints override the promoted
// methods. Later milestones keep adding overrides one operation at a
// time -- do not restructure this embed.
type server struct {
	apigen.Unimplemented

	// pool begins the setup transaction; queries serves read paths.
	pool    *pgxpool.Pool
	queries *dbgen.Queries
	version string
	logger  *slog.Logger
	// events feeds the SSE activity stream.
	events *events.Bus
	// limiter gates login attempts per address+account pair.
	limiter *auth.Limiter
	// pairingBudget bounds pairing token generation per user per UTC
	// day.
	pairingBudget *auth.DailyLimiter
	// dataDir is the photo data root the upload endpoint stages into.
	dataDir string
	// uploadLimit caps each uploaded file.
	uploadLimit int64
	// jobs enqueues the pipeline jobs of a committed upload; nil means
	// no queue is wired and uploads simply stay not-ready.
	jobs AssetJobEnqueuer
	// modelDownloads starts the default model group's download runs;
	// nil means no queue is wired and the download endpoint answers
	// its own failure code.
	modelDownloads ModelDownloadEnqueuer
	// models is the instance's shared model registry; nil means no
	// registry is wired and the model management endpoints answer the
	// generic internal error.
	models *ml.Registry
	// modelsDir is the model store root the scan endpoint walks.
	modelsDir string
	// mlReruns starts the ML rerun runs of a slot; nil means no queue
	// is wired and the rerun endpoint answers the generic internal
	// error.
	mlReruns MlRerunEnqueuer
	// placesReruns starts the place re-resolution runs; nil means no
	// queue is wired and the re-resolution endpoint answers the
	// generic internal error.
	placesReruns PlacesRerunEnqueuer
}

// writeJSON emits v as the JSON response body with the given status.
func (s *server) writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	// The status is already written, so an encode failure (client gone)
	// cannot be reported; the write is best-effort.
	_ = json.NewEncoder(w).Encode(v)
}

// writeError emits the shared error envelope: a stable code clients
// translate through their i18n tables (translation key = lowercase code),
// an English debug message and optional structured details.
func (s *server) writeError(w http.ResponseWriter, status int, code apigen.ErrorCode, message string, details *map[string]interface{}) {
	s.writeJSON(w, status, apigen.Error{
		Code:    code,
		Message: &message,
		Details: details,
	})
}

// internalError logs an unexpected failure and answers the 500 error
// envelope with the INTERNAL code registered in the contract. The
// triggering error is logged server-side only; clients get a generic
// message because internal details are not theirs to see.
func (s *server) internalError(w http.ResponseWriter, r *http.Request, op string, err error) {
	s.logger.Error("internal error", "op", op, "error", err, "method", r.Method, "path", r.URL.Path)
	s.writeError(w, http.StatusInternalServerError, apigen.INTERNAL,
		"internal server error", nil)
}

// binderError adapts the generated request binders onto the shared
// error envelope: a request the binder refuses (a malformed uuid path
// parameter, a duplicated Last-Event-ID header) is a client input
// problem, so the answer is 400 VALIDATION_FAILED naming the offending
// parameter instead of the generated default plain-text 400.
func (s *server) binderError(w http.ResponseWriter, r *http.Request, err error) {
	s.logger.Debug("request binder rejected input", "error", err, "method", r.Method, "path", r.URL.Path)
	var details *map[string]interface{}
	if param, ok := binderParamName(err); ok {
		details = &map[string]interface{}{"fields": []interface{}{param}}
	}
	s.writeError(w, http.StatusBadRequest, apigen.VALIDATIONFAILED,
		"invalid request parameter", details)
}

// binderParamName extracts the parameter a generated binder rejected.
// Every binder error type of the generated wrapper carries a ParamName
// field; this lists them once so new ones surface here on the next
// regen.
func binderParamName(err error) (string, bool) {
	var invalidFormat *apigen.InvalidParamFormatError
	if errors.As(err, &invalidFormat) {
		return invalidFormat.ParamName, true
	}
	var tooMany *apigen.TooManyValuesForParamError
	if errors.As(err, &tooMany) {
		return tooMany.ParamName, true
	}
	var requiredHeader *apigen.RequiredHeaderError
	if errors.As(err, &requiredHeader) {
		return requiredHeader.ParamName, true
	}
	var requiredParam *apigen.RequiredParamError
	if errors.As(err, &requiredParam) {
		return requiredParam.ParamName, true
	}
	var unmarshaling *apigen.UnmarshalingParamError
	if errors.As(err, &unmarshaling) {
		return unmarshaling.ParamName, true
	}
	var unescapedCookie *apigen.UnescapedCookieParamError
	if errors.As(err, &unescapedCookie) {
		return unescapedCookie.ParamName, true
	}
	return "", false
}

// handleSetupPage serves GET /setup: the embedded first-run wizard. It
// stays reachable after setup completes; the setup API itself refuses
// completed instances.
func handleSetupPage(w http.ResponseWriter, _ *http.Request) {
	servePage(w, "setup.html")
}

// servePage writes one of the embedded HTML pages. A missing page is a
// packaging bug (the files ship in the binary), so it degrades to a
// plain 500 rather than an undocumented error envelope.
func servePage(w http.ResponseWriter, name string) {
	body, err := static.FS.ReadFile(name)
	if err != nil {
		http.Error(w, "page not found", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	// The status is already written, so a write failure (client gone)
	// cannot be reported; the write is best-effort.
	_, _ = w.Write(body)
}

// healthzResponse is the body served on GET /healthz.
type healthzResponse struct {
	Status string `json:"status"`
}

func handleHealthz(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	// The status is already written, so an encode failure (client gone)
	// cannot be reported; the write is best-effort.
	_ = json.NewEncoder(w).Encode(healthzResponse{Status: "ok"})
}
