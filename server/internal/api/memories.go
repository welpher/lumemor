// The memory creation face: POST /memories, the one write every
// member holds. A memory is a reference set -- photos live in the
// albums they were ingested through, the memory holds references to
// them -- so the creation names no source, moves no file and borrows
// nothing but what the creator already sees. That shape is the gate's
// own reasoning: mounting an existing photo directory is a
// configuration act and stays the administrator's, but curating a
// personal set out of photos one already sees is personal
// expression -- the opposite gate, no role check exists to refuse.
// The row lands bare (owner, title, type), the creator's
// timeline-exclusion preference row lands with it (an own memory
// never feeds the creator's aggregated timeline -- the arms of the
// aggregate already hold that line, and the row states it in the
// data the preference machinery reads), and the request's audience
// converges through the audience face's own full-set semantics.
// The response is the fresh row's wall card widened with the honest
// audience report; the write publishes album.created plus one
// album.shared per grant that landed -- the album event family
// carries the memory like any album, ids-only payloads.
package api

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"

	apigen "github.com/welpher/lumemor/server/internal/api/gen"
	dbgen "github.com/welpher/lumemor/server/internal/db/gen"
)

// CreateMemory implements POST /api/v1/memories: the every-member
// creation. The body is a small JSON document (the route group's
// 1 MiB cap bounds it before this decode); the title runs the same
// trim-and-bound the album titles run, and the audience rides the
// creation through the audience face's convergence -- unusable ids
// skip honestly, never errors.
func (s *server) CreateMemory(w http.ResponseWriter, r *http.Request) {
	requester, ok := authFrom(r.Context())
	if !ok {
		s.rejectUnauthenticated(w)
		return
	}

	var request apigen.MemoryCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		s.writeValidationError(w, "request body must be valid JSON", []string{"body"})
		return
	}
	title, ok := s.albumTitle(w, request.Title)
	if !ok {
		return
	}
	if request.Audience != nil && len(*request.Audience) > albumAudienceMaxIds {
		s.writeValidationError(w, "audience exceeds the limit of 200", []string{"audience"})
		return
	}
	var audience []uuid.UUID
	if request.Audience != nil {
		audience = distinctUserIDs(*request.Audience)
	}

	ctx := r.Context()

	id, err := uuid.NewV7()
	if err != nil {
		s.internalError(w, r, "generate memory id", err)
		return
	}
	if _, err := s.queries.CreateMemoryAlbum(ctx, dbgen.CreateMemoryAlbumParams{
		ID:      pgUUID(id),
		OwnerID: pgUUID(requester.userID),
		Title:   title,
	}); err != nil {
		s.internalError(w, r, "create the memory", err)
		return
	}

	// The timeline exclusion, written HERE, in the creation: an own
	// memory never feeds the creator's aggregated timeline (in-stream
	// curation would be pure duplication), and the preference row is
	// that statement in the data the exclusion machinery reads -- the
	// aggregate's arms already leave own memories out, and the row
	// keeps the exclusion true whatever arms a later formula admits.
	// The exclusion is the creator's own: the audience members' feeds
	// are governed by their own preference rows, never by this write.
	if _, err := s.queries.UpsertAlbumTimelinePreference(ctx, dbgen.UpsertAlbumTimelinePreferenceParams{
		UserID:     pgUUID(requester.userID),
		AlbumID:    pgUUID(id),
		InTimeline: false,
	}); err != nil {
		s.internalError(w, r, "record the timeline exclusion", err)
		return
	}

	s.publishAlbumEvent("album.created", id, requester.userID)

	// The creation's audience lands through the same convergence the
	// audience face runs: the grants that land fire their events,
	// the unusable ids ride the response's skipped report.
	_, _, skipped, ok := s.applyAlbumAudience(w, r, id, requester.userID, requester.userID, audience)
	if !ok {
		return
	}

	summary, ok := s.albumSummaryCardForID(w, r, requester.userID, id)
	if !ok {
		return
	}
	s.writeJSON(w, http.StatusCreated, memoryCreateResponse(summary, skipped))
}

// memoryCreateResponse widens one wall card with the creation's
// honest audience report. The summary fields copy verbatim (the
// generated allOf flattening makes the widening a field copy), and
// the skipped list rides along -- never nil, so the JSON carries
// the empty array.
func memoryCreateResponse(card apigen.AlbumSummary, skipped []apigen.AlbumAudienceSkip) apigen.MemoryCreateResponse {
	response := apigen.MemoryCreateResponse{
		Id:            card.Id,
		Title:         card.Title,
		Type:          apigen.MemoryCreateResponseType(card.Type),
		IsShared:      card.IsShared,
		CoverAssetIds: card.CoverAssetIds,
		ItemCount:     card.ItemCount,
		UpdatedAt:     card.UpdatedAt,
		CallerRole:    apigen.MemoryCreateResponseCallerRole(card.CallerRole),
		Skipped:       skipped,
	}
	if response.Skipped == nil {
		response.Skipped = []apigen.AlbumAudienceSkip{}
	}
	if card.Owner != nil {
		response.Owner = card.Owner
	}
	if card.InTimeline != nil {
		response.InTimeline = card.InTimeline
	}
	return response
}
