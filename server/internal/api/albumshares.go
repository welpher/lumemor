// The album sharing face (M3c Task 3): the view-ACL grant behind
// POST /albums/{albumId}/shares and the revoke behind DELETE
// /albums/{albumId}/shares/{userId}. The share is view-only
// forever (the ACL's one bit), so both answers ride the writes'
// two-tier rule: an album the caller cannot see at all (unknown,
// foreign-unshared, every sync album) answers the reads'
// byte-identical single-shape 404, and a sharee answers an honest
// 403 -- with one deliberate exception the revoke makes: a grantee
// may delete their OWN share row, leaving a shared album is the
// sharee's right and rides the same endpoint as the owner's
// revocation. Locked never gates this face (the freeze is a
// curation gate; a viewer only ever reads).
package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	openapi_types "github.com/oapi-codegen/runtime/types"

	apigen "github.com/welpher/lumemor/server/internal/api/gen"
	dbgen "github.com/welpher/lumemor/server/internal/db/gen"
)

// publishAlbumShareEvent reports one share transition onto the
// activity stream. The payload carries the album's id and the
// GRANTEE's user id -- not the acting grantor: the grantor is
// derivable server-side, the grantee is the state that changed, and
// the grant time and the grantee's email are state the subscribers
// refetch (the ids-only discipline of every album.* event, with the
// one documented twist that the share events' userId is the grantee
// while the other album events' is the actor). The nil guard matches
// the album publisher (a router wired without a bus is a
// wiring-order state, tests only).
func (s *server) publishAlbumShareEvent(name string, albumID, granteeID uuid.UUID) {
	if s.events == nil {
		return
	}
	s.events.Publish(name, map[string]string{
		"albumId": albumID.String(),
		"userId":  granteeID.String(),
	})
}

// loadAlbumForShare resolves the sharing face's common tiers: the
// album row (the write face's unfiltered read -- sync albums answer
// no row, the reads' single-shape 404) plus whether the caller owns
// the album (the admin role overrides). It answers ok=false once it
// has written the response: the 404 for every album the caller can
// see nothing of, the 500 for the read failure. The sharee's answer
// stays the CALLER's -- the grant refuses onward sharing with its
// own 403, the revoke allows the own-row departure.
func (s *server) loadAlbumForShare(w http.ResponseWriter, r *http.Request, requester authContext, albumID uuid.UUID) (dbgen.GetAlbumForWriteRow, bool, bool) {
	row, err := s.queries.GetAlbumForWrite(r.Context(), dbgen.GetAlbumForWriteParams{
		AlbumID: pgUUID(albumID),
		UserID:  pgUUID(requester.userID),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		s.writeAlbumNotFound(w)
		return dbgen.GetAlbumForWriteRow{}, false, false
	}
	if err != nil {
		s.internalError(w, r, "load album for share", err)
		return dbgen.GetAlbumForWriteRow{}, false, false
	}
	owner := uuid.UUID(row.Album.OwnerID.Bytes) == requester.userID ||
		requester.role == dbgen.UserRoleAdmin
	if !owner && !row.SharedToCaller {
		// The caller can see nothing here: the same 404 as the
		// unknown id, byte for byte -- album ids cannot be probed
		// through the sharing face either.
		s.writeAlbumNotFound(w)
		return dbgen.GetAlbumForWriteRow{}, false, false
	}
	return row, owner, true
}

// GrantAlbumShare implements POST /api/v1/albums/{albumId}/shares:
// one album_shares row, one more viewer. The authorization is the
// sharing face's two-tier answer (invisible 404, sharee 403, the
// owner or an admin override grants); the target rules answer the
// 400s -- the album's own owner is never a grantee, and an unknown
// or disabled account is not a share target, both cases of
// album_share_user_invalid answering the SAME shape so user ids
// cannot be probed through this face. The grant is idempotent: the
// re-grant answers the existing row (its original grantedAt) and
// stays silent on the activity stream; only the row that actually
// lands fires album.shared. Locked never gates (the freeze is a
// curation gate; a viewer only ever reads).
func (s *server) GrantAlbumShare(w http.ResponseWriter, r *http.Request, albumId openapi_types.UUID) {
	requester, ok := authFrom(r.Context())
	if !ok {
		s.rejectUnauthenticated(w)
		return
	}

	// The body is a small JSON document (the route group's 1 MiB cap
	// bounds it before this decode); anything that does not decode
	// is a client bug.
	var request apigen.AlbumShareGrantRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		s.writeValidationError(w, "request body must be valid JSON", []string{"body"})
		return
	}
	grantee := uuid.UUID(request.UserId)
	if grantee == uuid.Nil {
		// The generated binder cannot enforce the body's required
		// field: a missing userId decodes to the zero uuid, which
		// names no account.
		s.writeValidationError(w, "the grant must name a user", []string{"userId"})
		return
	}

	ctx := r.Context()
	row, owner, ok := s.loadAlbumForShare(w, r, requester, uuid.UUID(albumId))
	if !ok {
		return
	}
	if !owner {
		// The caller sees the album (their share row exists) but
		// does not own it: sharing onward is the honest
		// authorization failure -- the share is view-only.
		s.writeError(w, http.StatusForbidden, apigen.AUTHFORBIDDEN,
			"only the album's owner may share it", nil)
		return
	}

	// The target rules, after the album gate: an invisible album
	// answers its 404 whatever the target, so the album ids keep
	// their anti-enumeration posture. The album's own owner is never
	// a grantee -- the owner sees the album by construction.
	if grantee == uuid.UUID(row.Album.OwnerID.Bytes) {
		s.writeMessageKey(w, "an album's owner cannot be its grantee", "album_share_to_owner")
		return
	}
	target, err := s.queries.GetUserForShareTarget(ctx, pgUUID(grantee))
	if errors.Is(err, pgx.ErrNoRows) {
		s.writeMessageKey(w, "the share target is not a usable account", "album_share_user_invalid")
		return
	}
	if err != nil {
		s.internalError(w, r, "load the share target", err)
		return
	}
	if target.IsDisabled {
		// The same shape as the unknown id's answer: the two cases
		// are one rule, and neither probes user ids.
		s.writeMessageKey(w, "the share target is not a usable account", "album_share_user_invalid")
		return
	}

	inserted, err := s.queries.GrantAlbumShare(ctx, dbgen.GrantAlbumShareParams{
		AlbumID:   pgUUID(albumId),
		UserID:    pgUUID(grantee),
		GrantedBy: pgUUID(requester.userID),
	})
	if err != nil {
		if personFKViolation(err) {
			// The album vanished between the gate and the insert:
			// the row is gone exactly like an unknown id.
			s.writeAlbumNotFound(w)
			return
		}
		s.internalError(w, r, "grant the album share", err)
		return
	}
	if inserted > 0 {
		s.publishAlbumShareEvent("album.shared", uuid.UUID(albumId), grantee)
	}

	// The row after the write: the fresh row for a new grant, the
	// existing row (its original grant time) for the idempotent
	// re-grant.
	share, err := s.queries.GetAlbumShare(ctx, dbgen.GetAlbumShareParams{
		AlbumID: pgUUID(albumId),
		UserID:  pgUUID(grantee),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		// A concurrent revoke won the race: the share is gone and
		// the album may be too -- the single-shape 404 keeps the
		// answer honest.
		s.writeAlbumNotFound(w)
		return
	}
	if err != nil {
		s.internalError(w, r, "load the album share", err)
		return
	}
	s.writeJSON(w, http.StatusOK, apigen.AlbumShareResponse{
		AlbumId:   albumId,
		UserId:    request.UserId,
		GrantedAt: share.GrantedAt.Time.UTC(),
	})
}

// RevokeAlbumShare implements DELETE /api/v1/albums/{albumId}/shares
// /{userId}: one album_shares row leaves, and the grantee's read
// access ends with it. The owner (or an administrator) revokes any
// grantee; a grantee may delete their OWN row -- leaving a shared
// album is the sharee's right, the same endpoint as the owner's
// revocation -- but not another account's grant (403, the honest
// refusal of a caller who can see the album yet holds no power over
// its ACL). The 204 is idempotent: a share that is already gone is
// the state the caller asked for, and only the row that actually
// leaves fires album.unshared. Locked never gates.
func (s *server) RevokeAlbumShare(w http.ResponseWriter, r *http.Request, albumId, userId openapi_types.UUID) {
	requester, ok := authFrom(r.Context())
	if !ok {
		s.rejectUnauthenticated(w)
		return
	}

	_, owner, ok := s.loadAlbumForShare(w, r, requester, uuid.UUID(albumId))
	if !ok {
		return
	}
	grantee := uuid.UUID(userId)
	if !owner && grantee != requester.userID {
		// The caller is a sharee (the invisible stranger already
		// folded into the 404 inside the gate) who is trying to
		// revoke someone else's grant: the honest 403.
		s.writeError(w, http.StatusForbidden, apigen.AUTHFORBIDDEN,
			"only the album's owner or the grantee may revoke this share", nil)
		return
	}

	removed, err := s.queries.RevokeAlbumShare(r.Context(), dbgen.RevokeAlbumShareParams{
		AlbumID: pgUUID(albumId),
		UserID:  pgUUID(grantee),
	})
	if err != nil {
		s.internalError(w, r, "revoke the album share", err)
		return
	}
	if removed > 0 {
		s.publishAlbumShareEvent("album.unshared", uuid.UUID(albumId), grantee)
	}
	w.WriteHeader(http.StatusNoContent)
}
