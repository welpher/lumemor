// Package api exposes the HTTP surface of the lumemor server.
package api

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	apigen "github.com/welpher/lumemor/server/internal/api/gen"
	"github.com/welpher/lumemor/server/internal/api/static"
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
}

// NewRouter builds the HTTP routes of the server: the operational
// endpoints (/healthz, /metrics), the embedded HTML pages (/setup, /)
// and the generated /api/v1 API surface.
func NewRouter(deps *Deps) *chi.Mux {
	s := &server{
		pool:    deps.Pool,
		queries: dbgen.New(deps.Pool),
		version: deps.Version,
		logger:  deps.Logger,
		events:  deps.Events,
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
		apigen.HandlerFromMux(s, r)
	})
	return r
}

// server implements apigen.ServerInterface on top of the injected Deps.
// It embeds apigen.Unimplemented so that every endpoint not yet
// overridden answers 501; implemented endpoints (GetServerInfo,
// CreateAdmin today) override the promoted methods. Later milestones
// keep adding overrides one operation at a time -- do not restructure
// this embed.
type server struct {
	apigen.Unimplemented

	// pool begins the setup transaction; queries serves read paths.
	pool    *pgxpool.Pool
	queries *dbgen.Queries
	version string
	logger  *slog.Logger
	// events feeds the SSE activity stream.
	events *events.Bus
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

// internalError logs an unexpected failure and answers a bare 500: the
// error registry has no code for internal failures yet, so no envelope is
// emitted rather than inventing an undocumented code.
func (s *server) internalError(w http.ResponseWriter, r *http.Request, op string, err error) {
	s.logger.Error("internal error", "op", op, "error", err, "method", r.Method, "path", r.URL.Path)
	w.WriteHeader(http.StatusInternalServerError)
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
