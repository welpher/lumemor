// The rescan endpoint: POST /albums/{albumId}/rescan,
// the owner's manual trigger of the three-arm diff. The endpoint is
// the places re-resolution's own shape applied to the album domain:
// the permission is the album writes' two-tier answer (the owner or
// an administrator proceeds, a sharee answers the honest 403,
// everyone else folds into the reads' byte-identical 404), the type
// gate folds every non-directory album into the same 404 (the
// rescan is the directory album's own flow, exactly the membership
// writes' own type discipline), and the idempotence is the
// in-flight run joined, never a second one started -- scoped to the
// album, so one album's scan never blocks another's.
//
// The heavy work lives on the queue (the album_rescan worker of the
// jobs package), not here: the handler is one bounded row read plus
// one enqueue.
package api

import (
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	openapi_types "github.com/oapi-codegen/runtime/types"

	apigen "github.com/welpher/lumemor/server/internal/api/gen"
	dbgen "github.com/welpher/lumemor/server/internal/db/gen"
)

// albumRescanJobKind is the job_runs kind of the rescan runs -- the
// jobs package's pinned vocabulary, mirrored here the way the places
// rerun kind mirrors its own (the endpoint's in-flight join and the
// rescan test's seeded row drive the two against each other, so
// they cannot drift apart).
const albumRescanJobKind = "album_rescan"

// RescanAlbum implements POST /api/v1/albums/{albumId}/rescan: one
// rescan of a directory album's source. A run of the album still in
// flight answers 200 with its identifier (the request joins it,
// starts nothing); anything else enqueues a fresh run through the
// queue seam and answers 202.
func (s *server) RescanAlbum(w http.ResponseWriter, r *http.Request, albumId openapi_types.UUID) {
	requester, ok := authFrom(r.Context())
	if !ok {
		s.rejectUnauthenticated(w)
		return
	}
	ctx := r.Context()
	albumID := uuid.UUID(albumId)

	// The album write gate resolves the two-tier answer every album
	// write shares: the owner or an administrator proceeds, a sharee
	// answers the honest 403 (the share is view-only), everyone else
	// -- unknown ids, foreign unshared albums, every sync album --
	// folds into the reads' byte-identical 404.
	gate, ok := s.albumWriteGate(w, r, requester, albumID)
	if !ok {
		return
	}
	// The type gate: the rescan is the directory album's own flow.
	// A virtual album (the caller's own included) and the sync album
	// (already folded by the gate's read) answer the single-shape
	// 404 -- the membership writes' own type discipline, keeping
	// album ids unproammable.
	if gate.Album.Type != dbgen.AlbumTypeDirectory {
		s.writeAlbumNotFound(w)
		return
	}

	// A rescan of THIS album still in flight answers 200 with its
	// identifier: the request joins the running scan instead of
	// starting a second one over the same source. The lookup is
	// album-scoped -- another album's in-flight run matches nothing
	// here, the scans are independent.
	if run, err := s.queries.FindInFlightAlbumRescan(ctx, dbgen.FindInFlightAlbumRescanParams{
		Kind:    albumRescanJobKind,
		AlbumID: pgUUID(albumID),
	}); err == nil {
		s.writeJSON(w, http.StatusOK, apigen.AlbumRescanResponse{JobRunId: uuid.UUID(run.ID.Bytes)})
		return
	} else if !errors.Is(err, pgx.ErrNoRows) {
		s.internalError(w, r, "find the in-flight rescan run", err)
		return
	}

	if s.albumRescans == nil {
		s.logger.Error("album rescan requested without a job queue")
		s.writeError(w, http.StatusInternalServerError, apigen.INTERNAL,
			"the rescan could not be started", nil)
		return
	}
	jobRunID, err := s.albumRescans.EnqueueAlbumRescan(ctx, albumID)
	if err != nil {
		s.logger.Error("enqueue the album rescan", "error", err)
		s.writeError(w, http.StatusInternalServerError, apigen.INTERNAL,
			"the rescan could not be started", nil)
		return
	}
	s.writeJSON(w, http.StatusAccepted, apigen.AlbumRescanResponse{JobRunId: jobRunID})
}
