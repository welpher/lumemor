// Package api exposes the HTTP surface of the lumemor server.
package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// NewRouter builds the HTTP routes of the server.
//
// The signature intentionally takes no arguments for now; later milestones
// will pass a dependencies struct (database pool, logger, version).
func NewRouter() *chi.Mux {
	r := chi.NewRouter()
	r.Get("/healthz", handleHealthz)
	r.Handle("/metrics", promhttp.Handler())
	return r
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
