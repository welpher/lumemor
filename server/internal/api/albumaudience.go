// The album audience face (the albums-as-sources rework): the
// full-set write behind PUT /albums/{albumId}/audience, the one
// write surface of "who can see this album". The audience is an
// album property, and this endpoint converges the album_shares rows
// onto exactly the requested set -- the one-row grant/revoke pair
// retired with it (one concept, one endpoint). The share is
// view-only forever (the ACL's one bit), so the answer rides the
// writes' two-tier rule: an album the caller cannot see at all
// (unknown, foreign-unshared, every sync album -- the caller's own
// included, whose face is the timeline) answers the reads'
// byte-identical single-shape 404, and a sharee answers the honest
// 403. Locked never gates this face (the freeze is a curation gate;
// an audience member only ever reads).
package api

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"

	apigen "github.com/welpher/lumemor/server/internal/api/gen"
	dbgen "github.com/welpher/lumemor/server/internal/db/gen"
)

// albumAudienceMaxIds caps the audience write's id list, pinned by
// the contract: 200 ids keep one request a bounded statement and a
// bounded report, the batch surface's own discipline.
const albumAudienceMaxIds = 200

// publishAlbumShareEvent reports one audience transition onto the
// activity stream. The payload carries the album's id and the
// GRANTEE's user id -- not the acting owner: the owner is derivable
// server-side, the grantee is the state that changed, and the grant
// time and the grantee's email are state the subscribers refetch
// (the ids-only discipline of every album.* event, with the one
// documented twist that the audience events' userId is the grantee
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

// albumAudiencePlan resolves one requested audience set against the
// account table: the ids that may land as share rows (the usable
// accounts, the album owner's own id excluded -- the owner sees the
// album by construction, a row for them would be noise) and the
// honest skip list, one entry per unactionable id with its reason.
// The owner reason wins when the owner's id is also unusable (the
// disabled-owner edge): it is the owner-ness, not the account state,
// that keeps the id out of the audience. An unknown id and a
// disabled account fold into the same invalid_account reason so user
// ids cannot be probed through this face, the discipline the
// one-row grant kept before the set write folded it in.
func (s *server) albumAudiencePlan(ctx context.Context, ownerID uuid.UUID, requested []uuid.UUID) ([]uuid.UUID, []apigen.AlbumAudienceSkip, error) {
	usable, err := s.queries.ListUsableShareTargets(ctx, pgUUIDs(requested))
	if err != nil {
		return nil, nil, err
	}
	usableSet := make(map[uuid.UUID]bool, len(usable))
	for _, id := range usable {
		usableSet[uuid.UUID(id.Bytes)] = true
	}
	targets := make([]uuid.UUID, 0, len(requested))
	skipped := make([]apigen.AlbumAudienceSkip, 0)
	for _, id := range requested {
		switch {
		case id == ownerID:
			skipped = append(skipped, apigen.AlbumAudienceSkip{
				UserId: openapi_types.UUID(id),
				Reason: apigen.AlbumAudienceSkipReasonOwner,
			})
		case usableSet[id]:
			targets = append(targets, id)
		default:
			skipped = append(skipped, apigen.AlbumAudienceSkip{
				UserId: openapi_types.UUID(id),
				Reason: apigen.AlbumAudienceSkipReasonInvalidAccount,
			})
		}
	}
	return targets, skipped, nil
}

// applyAlbumAudience converges one album's audience onto the
// requested set and reports the transitions: the grants land first
// (ON CONFLICT DO NOTHING keeps an already-granted id a no-op), then
// the rows whose grantees the set no longer names leave. The two
// statements' RETURNING sets are the transition truth the response
// reports and the event discipline reads -- a concurrent writer's
// rows never surface as this write's transitions, and the idempotent
// repeat of the stored set lands nothing, removes nothing and fires
// nothing. It answers ok=false once it has written the response.
func (s *server) applyAlbumAudience(w http.ResponseWriter, r *http.Request, albumID, actorID, ownerID uuid.UUID, requested []uuid.UUID) (granted, revoked []uuid.UUID, skipped []apigen.AlbumAudienceSkip, ok bool) {
	ctx := r.Context()
	targets, skipped, err := s.albumAudiencePlan(ctx, ownerID, requested)
	if err != nil {
		s.internalError(w, r, "resolve the album audience", err)
		return nil, nil, nil, false
	}

	grantedRows, err := s.queries.AddAlbumAudienceGrants(ctx, dbgen.AddAlbumAudienceGrantsParams{
		AlbumID:   pgUUID(albumID),
		GrantedBy: pgUUID(actorID),
		UserIds:   pgUUIDs(targets),
	})
	if err != nil {
		if personFKViolation(err) {
			// The album vanished between the gate and the insert:
			// the row is gone exactly like an unknown id.
			s.writeAlbumNotFound(w)
			return nil, nil, nil, false
		}
		s.internalError(w, r, "write the album audience", err)
		return nil, nil, nil, false
	}
	granted = make([]uuid.UUID, 0, len(grantedRows))
	for _, row := range grantedRows {
		granted = append(granted, uuid.UUID(row.Bytes))
	}

	revokedRows, err := s.queries.RemoveAlbumAudienceRows(ctx, dbgen.RemoveAlbumAudienceRowsParams{
		AlbumID: pgUUID(albumID),
		UserIds: pgUUIDs(requested),
	})
	if err != nil {
		s.internalError(w, r, "write the album audience", err)
		return nil, nil, nil, false
	}
	revoked = make([]uuid.UUID, 0, len(revokedRows))
	for _, row := range revokedRows {
		revoked = append(revoked, uuid.UUID(row.Bytes))
	}

	// The event discipline: one album.shared per row this write
	// landed, one album.unshared per row it removed, the grantee in
	// the payload; the idempotent no-op stays silent.
	for _, grantee := range granted {
		s.publishAlbumShareEvent("album.shared", albumID, grantee)
	}
	for _, grantee := range revoked {
		s.publishAlbumShareEvent("album.unshared", albumID, grantee)
	}
	return granted, revoked, skipped, true
}

// UpdateAlbumAudience implements PUT /api/v1/albums/{albumId}/
// audience: the full-set audience write. The request carries the
// COMPLETE grantee set; the write converges the rows onto it,
// idempotently (the same set twice answers the same state, the
// second write lands nothing and fires nothing) and honestly (the
// requested ids that cannot be granted are skipped with a reason,
// never errors). An empty set is the valid "only me"; the owner's
// own access is the ownership row and never moves. Authorization is
// the writes' two-tier answer (invisible 404, sharee 403, the owner
// or an admin override writes); sync albums fold into the 404
// whatever the caller; locked never gates (the freeze is a curation
// gate; an audience member only ever reads).
func (s *server) UpdateAlbumAudience(w http.ResponseWriter, r *http.Request, albumId openapi_types.UUID) {
	requester, ok := authFrom(r.Context())
	if !ok {
		s.rejectUnauthenticated(w)
		return
	}

	// The body is a small JSON document (the route group's 1 MiB cap
	// bounds it before this decode); anything that does not decode
	// is a client bug.
	var request apigen.AlbumAudienceRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		s.writeValidationError(w, "request body must be valid JSON", []string{"body"})
		return
	}
	// The field must ride the body even when empty: an explicit
	// empty array IS the "only me" request, while an absent field
	// would silently read as the same thing -- the destructive-
	// silent-default guard of the settings writes.
	if request.Audience == nil {
		s.writeValidationError(w, "audience is required", []string{"audience"})
		return
	}
	if len(request.Audience) > albumAudienceMaxIds {
		s.writeValidationError(w, "audience exceeds the limit of 200", []string{"audience"})
		return
	}
	requested := distinctUserIDs(request.Audience)

	ctx := r.Context()
	gate, ok := s.albumWriteGate(w, r, requester, uuid.UUID(albumId))
	if !ok {
		return
	}
	// Locked never gates this face: the freeze is a curation gate,
	// and an audience member only ever reads.

	granted, revoked, skipped, ok := s.applyAlbumAudience(w, r, uuid.UUID(albumId),
		requester.userID, uuid.UUID(gate.Album.OwnerID.Bytes), requested)
	if !ok {
		return
	}

	// The audience as it stands after the write: the same grantee
	// list the owner's detail carries, the edit dialog's chips.
	entries, err := s.queries.ListAlbumShares(ctx, pgUUID(albumId))
	if err != nil {
		s.internalError(w, r, "load the album audience", err)
		return
	}
	audience := make([]apigen.AlbumShareEntry, 0, len(entries))
	for _, entry := range entries {
		audience = append(audience, apigen.AlbumShareEntry{
			UserId:    openapi_types.UUID(uuid.UUID(entry.UserID.Bytes)),
			Email:     openapi_types.Email(entry.Email),
			GrantedAt: entry.GrantedAt.Time.UTC(),
		})
	}

	s.writeJSON(w, http.StatusOK, apigen.AlbumAudienceResponse{
		AlbumId:  albumId,
		Audience: audience,
		Granted:  uuidList(granted),
		Revoked:  uuidList(revoked),
		Skipped:  skipped,
	})
}

// distinctUserIDs collapses an audience id list to the distinct ids
// in first-appearance order (the set discipline: a duplicated id
// addresses one account). The caller bounds the raw list first.
func distinctUserIDs(raw []openapi_types.UUID) []uuid.UUID {
	ids := make([]uuid.UUID, 0, len(raw))
	seen := make(map[uuid.UUID]bool, len(raw))
	for _, id := range raw {
		value := uuid.UUID(id)
		if !seen[value] {
			seen[value] = true
			ids = append(ids, value)
		}
	}
	return ids
}

// uuidList widens one id slice onto the response's element type,
// answering a non-nil slice so the JSON carries an empty array --
// never null -- for the empty transition.
func uuidList(ids []uuid.UUID) []openapi_types.UUID {
	out := make([]openapi_types.UUID, 0, len(ids))
	for _, id := range ids {
		out = append(out, openapi_types.UUID(id))
	}
	return out
}
