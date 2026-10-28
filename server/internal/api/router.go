// Package api exposes the HTTP surface of the lumemor server.
package api

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	apigen "github.com/welpher/lumemor/server/internal/api/gen"
	"github.com/welpher/lumemor/server/internal/buildinfo"
	dbgen "github.com/welpher/lumemor/server/internal/db/gen"
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
}

// NewRouter builds the HTTP routes of the server: the operational
// endpoints (/healthz, /metrics) and the generated /api/v1 API surface.
func NewRouter(deps *Deps) *chi.Mux {
	s := &server{
		queries: dbgen.New(deps.Pool),
		version: deps.Version,
		logger:  deps.Logger,
	}
	if s.version == "" {
		s.version = buildinfo.Version
	}
	if s.logger == nil {
		s.logger = slog.Default()
	}

	r := chi.NewRouter()
	r.Get("/healthz", handleHealthz)
	r.Handle("/metrics", promhttp.Handler())
	r.Route("/api/v1", func(r chi.Router) {
		apigen.HandlerFromMux(s, r)
	})
	return r
}

// server implements apigen.ServerInterface on top of the injected Deps.
type server struct {
	queries *dbgen.Queries
	version string
	logger  *slog.Logger
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
