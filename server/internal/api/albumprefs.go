// The album timeline-preference face: the caller's
// per-user "keep this album out of my timeline" switch behind
// PUT/DELETE /albums/{albumId}/timeline-preference. The scope is the
// albums whose membership a caller could want out of their feed --
// their own directory albums and the albums shared to them; the
// caller's own sync and own memory albums answer the 400 (the core
// stream is never excludable, and an own memory album never feeds
// the owner's timeline at all). The preference is self-affecting
// state: no event publishes, the caller's own response is the only
// view that changed. The reset deletes the row, which returns the
// album to the default of feeding the timeline.
package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	apigen "github.com/welpher/lumemor/server/internal/api/gen"
	dbgen "github.com/welpher/lumemor/server/internal/db/gen"
)

// loadAlbumForTimelinePref resolves the album for a preference
// write: every type is in scope for the READ (the caller's own sync
// and own memory albums must reach the handler so their 400 can
// answer), and a row the caller neither owns nor holds a share of
// resolves to nothing -- the single-shape 404 every album face
// answers. It answers ok=false once it has written the response.
func (s *server) loadAlbumForTimelinePref(w http.ResponseWriter, r *http.Request, callerID, albumID uuid.UUID) (dbgen.GetAlbumForTimelinePrefRow, bool) {
	row, err := s.queries.GetAlbumForTimelinePref(r.Context(), dbgen.GetAlbumForTimelinePrefParams{
		UserID:  pgUUID(callerID),
		AlbumID: pgUUID(albumID),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		s.writeAlbumNotFound(w)
		return dbgen.GetAlbumForTimelinePrefRow{}, false
	}
	if err != nil {
		s.internalError(w, r, "load album for timeline preference", err)
		return dbgen.GetAlbumForTimelinePrefRow{}, false
	}
	return row, true
}

// requireExcludableAlbum answers whether the album is a preference
// target for the caller: an own directory album or any album shared
// to them. The refusal (the caller's own sync album, the caller's
// own memory album, or a sync album a share row somehow names)
// answers the 400 message key the contract pins. It answers
// ok=false once it has written the response.
func (s *server) requireExcludableAlbum(w http.ResponseWriter, row dbgen.GetAlbumForTimelinePrefRow, callerID uuid.UUID) bool {
	if row.Album.Type == dbgen.AlbumTypeSync ||
		(row.Album.Type == dbgen.AlbumTypeMemory && row.CallerRole == "owner") {
		s.writeMessageKey(w, "this album cannot be excluded from the timeline", "album_pref_not_excludable")
		return false
	}
	return true
}

// SetAlbumTimelinePreference implements PUT
// /albums/{albumId}/timeline-preference: the upsert of the caller's
// preference row. The write is self-affecting (no event), and the
// response carries the stored state -- an idempotent repeat answers
// the same row it converged on.
func (s *server) SetAlbumTimelinePreference(w http.ResponseWriter, r *http.Request, albumId uuid.UUID) {
	requester, ok := authFrom(r.Context())
	if !ok {
		s.rejectUnauthenticated(w)
		return
	}

	// The body is a small JSON document (the route group's 1 MiB cap
	// bounds it before this decode); anything that does not decode,
	// or a body without the one required boolean (a JSON null decodes
	// to the same absent pointer -- a required boolean has no
	// absent-state semantics), is a client bug. The generated binder
	// cannot enforce the field's presence itself, so the manual nil
	// check follows the share grant's userId precedent.
	var request apigen.AlbumTimelinePreferenceRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil || request.InTimeline == nil {
		s.writeValidationError(w, "request body must be valid JSON", []string{"body"})
		return
	}

	row, ok := s.loadAlbumForTimelinePref(w, r, requester.userID, uuid.UUID(albumId))
	if !ok {
		return
	}
	if !s.requireExcludableAlbum(w, row, requester.userID) {
		return
	}

	stored, err := s.queries.UpsertAlbumTimelinePreference(r.Context(), dbgen.UpsertAlbumTimelinePreferenceParams{
		UserID:     pgUUID(requester.userID),
		AlbumID:    pgUUID(uuid.UUID(albumId)),
		InTimeline: *request.InTimeline,
	})
	if err != nil {
		s.internalError(w, r, "upsert the timeline preference", err)
		return
	}

	s.writeJSON(w, http.StatusOK, apigen.AlbumTimelinePreferenceResponse{
		AlbumId:    albumId,
		InTimeline: stored,
	})
}

// ResetAlbumTimelinePreference implements DELETE
// /albums/{albumId}/timeline-preference: the caller's preference row
// goes, the album returns to the default of feeding their timeline.
// The delete is idempotent whether or not a row exists, and like the
// upsert it publishes nothing.
func (s *server) ResetAlbumTimelinePreference(w http.ResponseWriter, r *http.Request, albumId uuid.UUID) {
	requester, ok := authFrom(r.Context())
	if !ok {
		s.rejectUnauthenticated(w)
		return
	}

	row, ok := s.loadAlbumForTimelinePref(w, r, requester.userID, uuid.UUID(albumId))
	if !ok {
		return
	}
	if !s.requireExcludableAlbum(w, row, requester.userID) {
		return
	}

	if _, err := s.queries.DeleteAlbumTimelinePreference(r.Context(), dbgen.DeleteAlbumTimelinePreferenceParams{
		UserID:  pgUUID(requester.userID),
		AlbumID: pgUUID(uuid.UUID(albumId)),
	}); err != nil {
		s.internalError(w, r, "delete the timeline preference", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
