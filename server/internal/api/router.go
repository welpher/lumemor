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
	"github.com/welpher/lumemor/server/internal/geodata"
	"github.com/welpher/lumemor/server/internal/ml"
	"github.com/welpher/lumemor/server/internal/updatecheck"
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
	// ProbeHosts answers the download sources list the source
	// pipeline checks its URL candidates against: the same settings
	// the downloader's own gate reads. Nil admits every host (the
	// unconfigured gate).
	ProbeHosts func() []string
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
	// AlbumRescans starts the directory album rescan runs: the
	// rescan endpoint enqueues the album_rescan job through this
	// seam. Nil keeps the endpoint answering 500 INTERNAL (no queue
	// wired -- tests, wiring order); production passes the jobs
	// queue, which implements it structurally.
	AlbumRescans AlbumRescanEnqueuer
	// DerivativesRebuilds starts the derivative rebuild runs: the
	// rebuild endpoint enqueues the rebuild_derivatives job through
	// this seam. Nil keeps the endpoint answering 500 INTERNAL (no
	// queue wired -- tests, wiring order); production passes the
	// jobs queue, which implements it structurally.
	DerivativesRebuilds DerivativesRebuildEnqueuer
	// Now is the instance's clock: the cover rotation derives its ISO
	// week from it (the boundary is the returned instant's own
	// location -- the instance timezone). Nil keeps time.Now, which
	// is the production wiring; a test injects a fixed instant to
	// walk a week boundary without waiting for one.
	Now func() time.Time
	// UpdateChecks is the instance's release checker: the admin face
	// reads its cached answer and the manual run probes through it.
	// Nil keeps those endpoints answering 500 INTERNAL (no checker
	// wired -- tests, wiring order); production passes the checker
	// main also runs on its schedule.
	UpdateChecks *updatecheck.Checker
	// Places is the in-memory place grid the manual location edit
	// resolves its city against (the same grid the metadata job's
	// GPS step and the places rerun resolve against). Nil keeps the
	// edit writing coordinates only: the resolved place keeps
	// whatever the asset already carried (the feature-off wiring,
	// the metadata step's own discipline).
	Places *geodata.Index
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

// AlbumRescanEnqueuer starts one directory album rescan run. It is
// the seam between the rescan endpoint and the job queue:
// production passes the jobs package's Queue, which implements it
// structurally.
// DerivativesRebuildEnqueuer starts one derivative rebuild run. It
// is the seam between the rebuild endpoint and the job queue:
// production passes the jobs package's Queue, which implements it
// structurally.
type DerivativesRebuildEnqueuer interface {
	// EnqueueDerivativesRebuild creates the run's job_runs row and
	// enqueues the River job in one transaction, returning the run's
	// identifier. The scope is the contract's enum (missing, all).
	EnqueueDerivativesRebuild(ctx context.Context, scope string) (uuid.UUID, error)
}

type AlbumRescanEnqueuer interface {
	// EnqueueAlbumRescan creates the run's job_runs row and enqueues
	// the River job in one transaction, returning the run's
	// identifier. The caller has already gated the album (owner,
	// type, lock); the enqueue trusts the gate's answer.
	EnqueueAlbumRescan(ctx context.Context, albumID uuid.UUID) (uuid.UUID, error)
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
// (the HTML-surface item; the CSP arrives with the
// map feature). The /api/v1 pipeline order is pinned and
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

// probeManagerOf builds the source pipeline's probe manager from the
// router's deps: present whenever a models dir is wired (the staging
// lives under it), with the allowed-hosts check reading the download
// sources' settings and the dimension requirement reading the
// embedding column's current dimension (the database's truth).
func probeManagerOf(deps *Deps) *ml.ProbeManager {
	if deps.ModelsDir == "" {
		return nil
	}
	requireDims := func(slot ml.Slot) (int, bool) {
		if slot != ml.SlotClipEmbed || deps.Pool == nil {
			return 0, false
		}
		dim, err := ml.EmbeddingColumnDim(context.Background(), deps.Pool)
		if err != nil || dim <= 0 {
			return 0, false
		}
		return dim, true
	}
	return ml.NewProbeManager(deps.ModelsDir, deps.ProbeHosts, requireDims, nil, nil)
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
		clock:   deps.Now,
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
		// The source pipeline: a models dir wired with a registry
		// carries the probe manager (the transient probes live in
		// this process; a restart clears them with their staging).
		// Its host check reads the same allowed-hosts list the
		// downloader's gate reads.
		probes: probeManagerOf(deps),
		// The rerun enqueue seam: nil is a valid (no queue) wiring.
		mlReruns: deps.MlReruns,
		// The places re-resolution enqueue seam: nil is a valid (no
		// queue) wiring.
		placesReruns: deps.PlacesReruns,
		// The album rescan enqueue seam: nil is a valid (no queue)
		// wiring.
		albumRescans: deps.AlbumRescans,
		// The derivative rebuild enqueue seam: nil is a valid (no
		// queue) wiring.
		derivativesRebuilds: deps.DerivativesRebuilds,
		// The release checker: nil is a valid (no check) wiring.
		updateChecks: deps.UpdateChecks,
		// The place grid: nil is a valid (no place data) wiring; the
		// manual location edit then writes coordinates only.
		places: deps.Places,
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
	// and the operational endpoints (the HTML-surface item).
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
				// The batch and curation views: the
				// set-semantics batch write, and the favorites and
				// archived listings. The static segments win over the
				// /assets/{id} parameter below, so mounting them in
				// this group keeps every asset read a bounded row
				// read plus a bounded JSON document.
				r.Post("/assets/batch", wrapper.BatchAssets)
				r.Get("/assets/favorites", wrapper.GetFavorites)
				r.Get("/assets/archived", wrapper.GetArchived)
				r.Get("/assets/{id}", wrapper.GetAsset)
				r.Patch("/assets/{id}", wrapper.PatchAsset)
				r.Get("/assets/{id}/thumbnail", wrapper.GetAssetThumbnail)
				// The trash lifecycle endpoints: the
				// owner-scoped listing and restore, and the explicit
				// permanent delete. All three are bounded operations
				// (row reads and writes plus metadata-only file
				// removals), so they belong in this group like every
				// other JSON handler.
				r.Get("/trash", wrapper.GetTrash)
				// The whole-trash pair and the multi-select batch:
				// static segments, deliberately mounted BEFORE the
				// {id}/restore parametric route's siblings so chi's
				// static-over-param precedence keeps them unambiguous.
				r.Post("/trash/restore-all", wrapper.RestoreAllTrash)
				r.Post("/trash/empty", wrapper.EmptyTrash)
				r.Post("/trash/batch", wrapper.BatchTrash)
				r.Post("/trash/{id}/restore", wrapper.RestoreAsset)
				r.Delete("/assets/{id}", wrapper.DeleteAsset)
				// The album surface reads: the wall
				// (the caller's own and shared albums) and one
				// album's detail with its keyset-paged items. Both
				// are bounded row reads (the detail's page size is
				// capped by the contract); the wall's derived
				// covers are a second bounded read of at most four
				// candidates per album.
				r.Get("/albums", wrapper.ListAlbums)
				// The source-path completion of the creation dialog
				// (the albums-as-sources rework): one settings row read
				// plus at most one directory listing, at most 20
				// answer rows -- the bounded-JSON cost class. The
				// static path-completions segment never contends
				// with the /albums/{albumId} parameter below (chi
				// prefers static patterns).
				r.Get("/albums/path-completions", wrapper.CompleteAlbumPath)
				r.Get("/albums/{albumId}", wrapper.GetAlbum)
				// The album surface writes (reshaped by
				// albums-as-sources): the directory album creation (the path
				// validation matrix plus one bounded row insert and
				// the audience grants), the property patch, the
				// delete and the membership removal. All are
				// bounded row writes -- the membership and audience
				// lists are capped at 200 ids by the contract, and
				// the cascades ride the row deletes -- so they
				// belong in this group like every other bounded JSON
				// handler. The static /albums/{albumId}/items
				// segment never contends with the
				// /albums/{albumId} routes (a longer path shape).
				r.Post("/albums", wrapper.CreateAlbum)
				r.Patch("/albums/{albumId}", wrapper.UpdateAlbum)
				r.Delete("/albums/{albumId}", wrapper.DeleteAlbum)
				// The memory's membership curation pair: the add
				// (the re-share ground's validation folded into one
				// bounded insert) and the remove (one bounded delete),
				// both capped at 200 ids by the contract.
				r.Post("/albums/{albumId}/items", wrapper.AddAlbumItems)
				r.Delete("/albums/{albumId}/items", wrapper.RemoveAlbumItems)
				// The memory creation (every member's own write): one
				// bounded row insert, the creator's one preference row
				// and the audience grants -- the same bounded cost class
				// as the album writes, and the static /memories segment
				// never contends with any /albums route.
				r.Post("/memories", wrapper.CreateMemory)
				// The album audience face (the
				// albums-as-sources rework): the full-set
				// view-ACL write behind PUT
				// /albums/{albumId}/audience, the one write surface
				// of "who can see this album" (the one-row
				// grant/revoke pair retired with it). Two bounded
				// statements over at most 200 ids, the batch
				// surface's cost class. The static audience segment
				// never contends with the /albums/{albumId} routes
				// (a longer path shape).
				r.Put("/albums/{albumId}/audience", wrapper.UpdateAlbumAudience)
				// The rescan trigger (the
				// albums-as-sources rework): one bounded row read
				// plus one enqueue; the walk over the source root and
				// the three-arm diff live on the queue. The static
				// rescan segment never contends with the
				// /albums/{albumId} routes (a longer path shape).
				r.Post("/albums/{albumId}/rescan", wrapper.RescanAlbum)
				r.Get("/users", wrapper.ListInstanceUsers)
				// The me-scoped user settings face (the settings
				// feature): the caller's own profile, password and
				// signed-in devices. NO userId parameter exists on
				// any of these routes -- the session IS the identity,
				// the middleware's authContext decides the row -- and
				// the static /users/me segment never contends with
				// the bare /users listing above (a longer path
				// shape). The password change and the device revoke
				// are bounded row writes, the device list one bounded
				// read, so they belong in this group like every
				// other bounded JSON handler.
				r.Get("/users/me", wrapper.GetUserSelf)
				r.Patch("/users/me", wrapper.UpdateUserSelf)
				r.Put("/users/me/password", wrapper.ChangeUserPassword)
				r.Get("/users/me/sessions", wrapper.ListUserDevices)
				r.Delete("/users/me/sessions/{id}", wrapper.RevokeUserDevice)
				// The timeline-preference face: the
				// caller's per-user "keep this album out of my
				// timeline" switch -- one bounded row upsert and
				// one bounded row delete over an album the caller
				// can see, self-affecting state that publishes no
				// event. The static /timeline-preference segment
				// never contends with the /albums/{albumId}
				// routes (a longer path shape).
				r.Put("/albums/{albumId}/timeline-preference", wrapper.SetAlbumTimelinePreference)
				r.Delete("/albums/{albumId}/timeline-preference", wrapper.ResetAlbumTimelinePreference)
				// The people surface reads: the named
				// region, the unnamed region's keyset pages and one
				// cluster's detail. All are bounded row reads (the
				// unnamed page size is capped by the contract); the
				// static /people/unnamed segment wins over the
				// /people/{clusterId} parameter below.
				r.Get("/people", wrapper.ListPeople)
				r.Get("/people/unnamed", wrapper.ListUnnamedPeople)
				r.Get("/people/{clusterId}", wrapper.GetPersonCluster)
				// The places surface read: the caller's city
				// grouping -- two bounded row reads (the grouping with its
				// honest per-caller counts, then at most four cover
				// candidates per listed place), the same cost class as the
				// people listing.
				r.Get("/places", wrapper.ListPlaces)
				// The map's client-side tile configuration: one
				// settings row read. The map view and the viewer's
				// location card load their source through it; the
				// tile bytes never pass through this server.
				r.Get("/map/config", wrapper.GetMapConfig)
				// The tags surface: the caller's private
				// vocabulary and its batch assignment. The listing is
				// one bounded read (a family instance's vocabulary);
				// every write is a bounded row write (the
				// assignment lists are capped at 200 ids by the
				// contract, the cascades ride the row deletes).
				// Self-affecting per-user state: no events, and the
				// /tags/{tagId}/assets segment never contends with
				// the /tags/{tagId} routes (a longer path shape).
				r.Get("/tags", wrapper.ListTags)
				r.Post("/tags", wrapper.CreateTag)
				r.Patch("/tags/{tagId}", wrapper.RenameTag)
				r.Delete("/tags/{tagId}", wrapper.DeleteTag)
				r.Post("/tags/{tagId}/assets", wrapper.AddTagAssets)
				r.Delete("/tags/{tagId}/assets", wrapper.RemoveTagAssets)
				// The unified search surface: the two
				// merged paths over caller-visible assets plus the
				// filter facets. Bounded reads (the arms' depth cut,
				// the page cap, the family-scale facet lists) plus
				// at most one model inference per request -- the same
				// bounded-JSON cost class as the timeline.
				r.Get("/search", wrapper.SearchAssets)
				// The people surface writes: the caller's
				// label, the person entity's birth date
				// and avatar, the cluster merge, the
				// single face move, the from-face person
				// creation and the single-face detach.
				// All are bounded row
				// reads and writes (the merge touches at
				// most 50 source clusters, the face writes
				// exactly one row plus the vacated
				// cluster's maintenance); the static
				// /people/merge segment never contends
				// with the /people/{clusterId} routes (a
				// different method and path shape).
				r.Put("/people/{clusterId}/label", wrapper.LabelPerson)
				r.Delete("/people/{clusterId}/label", wrapper.UnlabelPerson)
				r.Put("/people/{clusterId}/birthdate", wrapper.SetPersonBirthdate)
				r.Put("/people/{clusterId}/avatar", wrapper.SetPersonAvatar)
				r.Post("/people/merge", wrapper.MergePeople)
				// The manual face: one drawn box becomes one face
				// row -- the same owner-gated curation discipline as
				// the move below, one insert, no cluster to maintain.
				r.Post("/faces", wrapper.CreateFace)
				r.Post("/faces/{faceId}/move", wrapper.MoveFace)
				// The from-face creation and the detach:
				// the same owner-gated curation discipline
				// as the move, one face at a time -- the
				// creation seeds a brand-new cluster with
				// the face, the detach releases the face to
				// the not-yet-grouped pool.
				r.Post("/faces/{faceId}/cluster", wrapper.ClusterFace)
				r.Delete("/faces/{faceId}", wrapper.DetachFace)
				// The face crop pixels: the 256x256 AVIF
				// crop of one face, derived at query time from its
				// asset's view derivative -- one bounded decode of a
				// serving copy plus one AVIF encode, the same cost
				// class as the thumbnail job's single-file
				// derivations, and cached on the client by the
				// response's immutable ETag.
				r.Get("/faces/{faceId}/thumbnail", wrapper.GetFaceThumbnail)
			})

			// Authed admin group: the user management
			// surface sits behind requireSession AND the admin-role
			// guard, inside the handler timeout like every other
			// bounded JSON handler (small row reads and writes).
			r.Group(func(r chi.Router) {
				r.Use(timeoutMiddleware(timeout), s.requireSession, s.requireAdmin)
				r.Get("/admin/users", wrapper.ListUsers)
				r.Post("/admin/users", wrapper.CreateUser)
				r.Patch("/admin/users/{id}", wrapper.UpdateUser)
				// The account disposal: the two-step delete's
				// transaction (both arms settle the departing
				// account's content before the row goes).
				r.Delete("/admin/users/{id}", wrapper.DeleteUser)
				// The storage settings face (the storage-template
				// feature): the template read and the three-stage
				// validated save -- one bounded setting row read or
				// write plus in-memory renders.
				r.Get("/admin/settings/storage", wrapper.GetStorageSettings)
				r.Put("/admin/settings/storage", wrapper.UpdateStorageSettings)
				// The trash retention and video transcoding faces
				// (the settings feature): the same GET/PUT shape over
				// the instance_settings rows -- bounded reads and
				// writes plus in-memory validation, the storage
				// settings face's own discipline.
				r.Get("/admin/settings/trash", wrapper.GetTrashSettings)
				r.Put("/admin/settings/trash", wrapper.UpdateTrashSettings)
				r.Get("/admin/settings/transcoding", wrapper.GetTranscodingSettings)
				r.Put("/admin/settings/transcoding", wrapper.UpdateTranscodingSettings)
				// The release check and its settings (the networking
				// block): the cached state read, the manual run and
				// the three knobs behind one GET/PUT pair. The
				// cached read is one settings row read; the manual
				// run is the checker's own bounded outbound probe.
				r.Get("/server/update-check", wrapper.GetUpdateCheck)
				r.Post("/server/update-check", wrapper.RunUpdateCheck)
				r.Get("/admin/settings/update-check", wrapper.GetUpdateCheckSettings)
				r.Put("/admin/settings/update-check", wrapper.UpdateUpdateCheckSettings)
				// The map's tile source (the networking block): the
				// admin settings pair. The browser fetches the
				// tiles themselves; this surface stores and hands
				// back the URL.
				r.Get("/admin/settings/tiles", wrapper.GetTilesSettings)
				r.Put("/admin/settings/tiles", wrapper.UpdateTilesSettings)
				// The model store's automatic download switch (the
				// networking block): one settings row read/write.
				r.Get("/admin/settings/models", wrapper.GetModelSettings)
				r.Put("/admin/settings/models", wrapper.UpdateModelSettings)
				// The derivative image settings (the background work's
				// cost block): the sizes and encoding every NEW
				// derivation runs at. The save recomputes nothing --
				// the rebuild is the explicit action below.
				r.Get("/admin/settings/image", wrapper.GetImageSettings)
				r.Put("/admin/settings/image", wrapper.UpdateImageSettings)
				// The background work's budget (the task panel's
				// run-settings strip): the concurrency dial, the night
				// window and the per-kind pause switches. The save
				// governs the queue's workers only -- uploads and
				// browsing are never consulted.
				r.Get("/admin/settings/run-budget", wrapper.GetRunBudgetSettings)
				r.Put("/admin/settings/run-budget", wrapper.UpdateRunBudgetSettings)
				// The derivative actions (the image card's cost line and
				// the explicit rebuild): the summary read, the run's
				// start (one queue seam) and its stop (a flag the
				// driver honors between assets). The heavy work lives
				// on the queue.
				r.Get("/admin/derivatives/summary", wrapper.GetDerivativesSummary)
				r.Post("/admin/derivatives/rebuild", wrapper.RebuildDerivatives)
				r.Post("/admin/derivatives/rebuild/stop", wrapper.StopDerivativesRebuild)
				// The model download trigger: the
				// default group's first-boot download. The handler is
				// one bounded row read plus one enqueue; the heavy
				// work lives on the queue.
				r.Post("/admin/models/download", wrapper.DownloadModels)
				// The model management and job panel surface:
				// the listing, the activation switch, the
				// rerun trigger and the custom model scan, plus the
				// job runs listing. All are bounded row reads and
				// writes (the rerun's heavy work lives on the queue
				// behind its enqueue); the scan route's static
				// segment never contends with the {id} routes (a
				// different method, and two segments for the rerun).
				r.Get("/admin/models", wrapper.ListModels)
				r.Patch("/admin/models/{id}", wrapper.ActivateModel)
				r.Delete("/admin/models/{id}", wrapper.DeleteModel)
				r.Post("/admin/models/{id}/rerun", wrapper.RerunModel)
				r.Post("/admin/models/scan", wrapper.ScanModels)
				// The source pipeline: the probe pair (the POST is
				// multipart and streams; the GET is the dialog's
				// poll) and the switch. The {slot} param name
				// differs from the {id} param at the same position
				// only across DIFFERENT static suffixes, which chi
				// routes independently.
				r.Post("/admin/models/{slot}/probe", wrapper.ProbeModelSource)
				r.Post("/admin/models/{slot}/source", wrapper.SwitchModelSource)
				r.Get("/admin/models/probes/{probeId}", wrapper.GetModelProbe)
				// The places re-resolution trigger: the
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
// methods. Later surfaces keep adding overrides one operation at a
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
	// probes is the source pipeline's probe manager: the stage and
	// probe steps of a custom import, and the register step's key
	// holder. Nil means no pipeline is wired.
	probes *ml.ProbeManager
	// mlReruns starts the ML rerun runs of a slot; nil means no queue
	// is wired and the rerun endpoint answers the generic internal
	// error.
	mlReruns MlRerunEnqueuer
	// placesReruns starts the place re-resolution runs; nil means no
	// queue is wired and the re-resolution endpoint answers the
	// generic internal error.
	placesReruns PlacesRerunEnqueuer
	// derivativesRebuilds starts the derivative rebuild runs; nil
	// means no queue is wired.
	derivativesRebuilds DerivativesRebuildEnqueuer
	// albumRescans starts the directory album rescan runs; nil means
	// no queue is wired and the rescan endpoint answers the generic
	// internal error.
	albumRescans AlbumRescanEnqueuer
	// updateChecks is the release checker; nil means no checker is
	// wired and the release check endpoints answer the generic
	// internal error.
	updateChecks *updatecheck.Checker
	// places is the in-memory place grid the manual location edit
	// resolves against; nil means no grid is wired and the edit
	// writes coordinates only (the resolved place stays).
	places *geodata.Index

	// textTowerCache keeps the search's semantic path warm: the
	// active clip-text model's loaded session and tokenizer (loading
	// a 400 MB graph per request is not a cost the page shape can
	// carry). A model switch replaces it under its mutex.
	textTowerCache textTowerSession
	// clock reads the instance's current instant; nil keeps time.Now
	// (the production wiring -- tests inject one to pin a week).
	clock func() time.Time
}

// now answers the instance's current instant: the cover rotation's
// ISO week reads it, and the nil arm is time.Now.
func (s *server) now() time.Time {
	if s.clock != nil {
		return s.clock()
	}
	return time.Now()
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
