package api

import (
	"net/http"

	apigen "github.com/welpher/lumemor/server/internal/api/gen"
)

// GetServerInfo implements GET /api/v1/server/info: the unauthenticated
// capability report clients use to locate the API, to decide between the
// first-run wizard and the login screen, and to adapt to the features the
// instance enables.
func (s *server) GetServerInfo(w http.ResponseWriter, r *http.Request) {
	// `initialized` is deliberately derived from the database, never from
	// process state, so a restarted server reports the same answer.
	admins, err := s.queries.CountAdmins(r.Context())
	if err != nil {
		s.internalError(w, r, "count admins", err)
		return
	}

	s.writeJSON(w, http.StatusOK, apigen.ServerInfo{
		Version:     s.version,
		Initialized: admins > 0,
		// No feature switches exist yet; the map stays non-nil so the
		// field serializes as {} rather than null.
		Features: map[string]bool{},
	})
}
