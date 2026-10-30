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
	// handler enqueues the metadata job and the thumbnail job after its
	// transaction commits. Nil keeps the upload working without a queue
	// (tests, wiring order); the assets simply stay not-ready until one
	// runs.
	Jobs AssetJobEnqueuer
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
}

// NewRouter builds the HTTP routes of the server: the operational
// endpoints (/healthz, /metrics), the embedded HTML pages (/setup, /)
// and the generated /api/v1 API surface.
//
// The /api/v1 pipeline order is pinned and security-relevant: the
// security headers on every API response, then the origin check
// guarding every state change, then the route groups. The JSON body
// limit and the per-request handler timeout apply only to the JSON
// surface (nothing downstream of limitJSONBody may buffer an
// unbounded body, and http.TimeoutHandler -- a handler that outlives
// it answers 503 -- bounds every non-streaming JSON handler): the
// authed groups sit behind RequireSession, the SSE stream group runs
// outside the timeout, and the upload group runs outside both the
// JSON body limit and the timeout -- a multipart upload is not JSON,
// and a multi-gigabyte video legitimately outlives any timer; the
// streaming receiver's own size limit governs instead.
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
	r.Get("/healthz", handleHealthz)
	r.Handle("/metrics", promhttp.Handler())
	r.Get("/setup", handleSetupPage)
	r.Get("/", func(w http.ResponseWriter, req *http.Request) {
		if !initialized(req.Context()) {
			http.Redirect(w, req, "/setup", http.StatusFound)
			return
		}
		handleIndexPage(w, req)
	})
	r.Route("/api/v1", func(r chi.Router) {
		// Security headers and the origin check guard every API
		// response, uploads included; the JSON body limit moved into
		// the JSON group below so multipart uploads can stream past
		// it (the receiver's own size limit governs them).
		r.Use(setSecurityHeaders, checkOrigin)

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

		// Authed media download group: GET /assets/{id}/original
		// streams the stored original file, whose size is bounded only
		// by the per-file upload limit (2 GiB by default) and whose
		// transfer runs at client speed. It requires a session like
		// every authed route but deliberately runs OUTSIDE the handler
		// timeout, mirroring the upload and SSE exemptions:
		// http.TimeoutHandler buffers the whole response until the
		// handler returns (see timeoutMiddleware), so a wrapped
		// original download would first read the entire file into
		// memory and only then let its first byte reach the client --
		// an unbounded per-request memory commitment. The handler
		// itself stays bounded: one row read, one file open, then
		// http.ServeContent streams the bytes.
		r.Group(func(r chi.Router) {
			r.Use(s.requireSession)
			r.Get("/assets/{id}/original", wrapper.GetAssetOriginal)
		})
	})
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

// handleIndexPage serves GET /: the embedded landing page an initialized
// instance shows.
func handleIndexPage(w http.ResponseWriter, _ *http.Request) {
	servePage(w, "index.html")
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
