// The tags surface: the caller's private vocabulary
// behind GET /tags (the search surface's word list) with its CRUD
// face (the idempotent create, the rename, the delete) and the batch
// assignment face behind POST and DELETE /tags/{tagId}/assets.
//
// Tags are per-user state with the favorites' privacy shape:
// every read and write is caller-scoped, the honest counts cover
// caller-visible live assets only, and another account's tags never
// surface anywhere -- a sharee sees their own tags on a shared asset
// (the union carries the assignment), never the owner's. The tag
// gate is therefore single-shape: an unknown id and a foreign tag
// answer the same 404, and there is no honest 403 because no tag is
// ever visible to anyone but its owner -- no share, no viewer role,
// no administrator override (the per-user-state surfaces never carry
// one).
//
// The domain publishes NO events: no tag.* family is pinned in the
// contract's events list, and the reasoning is the timeline
// preference's own -- the only views that change are the caller's
// own (their vocabulary, their asset details), and the writes'
// responses carry the converged state themselves.
package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	openapi_types "github.com/oapi-codegen/runtime/types"

	apigen "github.com/welpher/lumemor/server/internal/api/gen"
	dbgen "github.com/welpher/lumemor/server/internal/db/gen"
)

const (
	// tagNameMinRunes and tagNameMaxRunes bound the tag name:
	// trimmed, counted in characters (a byte bound would shorten
	// non-ASCII names), pinned by the contract -- the album title's
	// and the person name's own bound.
	tagNameMinRunes = 1
	tagNameMaxRunes = 64

	// tagAssetsMaxIds caps the assignment write's id list, pinned by
	// the contract: 200 ids keep one request a bounded statement and
	// a bounded report, the batch surface's own discipline (the
	// album membership's cap).
	tagAssetsMaxIds = 200
)

// writeTagNotFound emits the tags surface's single-shape 404: the
// same envelope every read and write answers for a tag that is not
// the caller's own -- unknown or foreign alike -- so tag ids cannot
// be probed for existence.
func (s *server) writeTagNotFound(w http.ResponseWriter) {
	s.writeError(w, http.StatusNotFound, apigen.TAGNOTFOUND,
		"no such tag for this caller", nil)
}

// writeTagNameTaken emits the rename's 409: the vocabulary keeps one
// row per name, and the collision's honest answer is the conflict,
// not a silent merge.
func (s *server) writeTagNameTaken(w http.ResponseWriter) {
	s.writeError(w, http.StatusConflict, apigen.TAGNAMETAKEN,
		"another tag already carries this name", nil)
}

// tagName trims the name and bounds its rune count; it answers
// ok=false once it has written the 400.
func (s *server) tagName(w http.ResponseWriter, name string) (string, bool) {
	trimmed := strings.TrimSpace(name)
	if runes := utf8.RuneCountInString(trimmed); runes < tagNameMinRunes || runes > tagNameMaxRunes {
		s.writeValidationError(w, "name must span 1-64 characters", []string{"name"})
		return "", false
	}
	return trimmed, true
}

// tagAssetIds validates the assignment write's id list and collapses
// it to the distinct ids in first-appearance order (the batch
// surface's pinned pattern: the actions are set operations, and a
// duplicated id addresses one row). It answers ok=false once it has
// written the 400.
func (s *server) tagAssetIds(w http.ResponseWriter, assetIds []openapi_types.UUID) ([]uuid.UUID, bool) {
	if len(assetIds) == 0 {
		s.writeValidationError(w, "assetIds must carry at least one asset id", []string{"assetIds"})
		return nil, false
	}
	if len(assetIds) > tagAssetsMaxIds {
		s.writeValidationError(w,
			fmt.Sprintf("assetIds exceed the limit of %d", tagAssetsMaxIds),
			[]string{"assetIds"})
		return nil, false
	}
	ids := make([]uuid.UUID, 0, len(assetIds))
	seen := make(map[uuid.UUID]bool, len(assetIds))
	for _, raw := range assetIds {
		id := uuid.UUID(raw)
		if !seen[id] {
			seen[id] = true
			ids = append(ids, id)
		}
	}
	return ids, true
}

// tagSummary maps one tag row with its honest count onto the API
// shape every tags-face response answers, so a write's answer is the
// entry the listing lists.
func tagSummary(tag dbgen.Tag, assetCount int64) apigen.TagSummary {
	return apigen.TagSummary{
		Id:         uuid.UUID(tag.ID.Bytes),
		Name:       tag.Name,
		AssetCount: int(assetCount),
		CreatedAt:  tag.CreatedAt.Time.UTC(),
		UpdatedAt:  tag.UpdatedAt.Time.UTC(),
	}
}

// tagGate resolves the caller's own tag behind a tagId-addressed
// write: the summary row (the response shape both the rename and the
// assignment endpoints build on) or the single-shape 404 for every
// id that is not the caller's own. It answers ok=false once it has
// written the response.
func (s *server) tagGate(w http.ResponseWriter, r *http.Request, callerID, tagID uuid.UUID) (dbgen.GetTagSummaryRow, bool) {
	row, err := s.queries.GetTagSummary(r.Context(), dbgen.GetTagSummaryParams{
		UserID: pgUUID(callerID),
		TagID:  pgUUID(tagID),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		s.writeTagNotFound(w)
		return dbgen.GetTagSummaryRow{}, false
	}
	if err != nil {
		s.internalError(w, r, "load the tag", err)
		return dbgen.GetTagSummaryRow{}, false
	}
	return row, true
}

// ListTags implements GET /api/v1/tags: the caller's whole
// vocabulary, sorted by name case-insensitively with the tag id as
// the tiebreak (the named-people listing's ordering discipline), each
// entry carrying its honest count over caller-visible live assets.
// Unpaginated like the places listing -- a family instance's
// vocabulary is a bounded set.
func (s *server) ListTags(w http.ResponseWriter, r *http.Request) {
	requester, ok := authFrom(r.Context())
	if !ok {
		s.rejectUnauthenticated(w)
		return
	}

	rows, err := s.queries.ListCallerTags(r.Context(), pgUUID(requester.userID))
	if err != nil {
		s.internalError(w, r, "load tags", err)
		return
	}
	// A non-nil empty slice: the fixed-shape contract wants an
	// array, never JSON null, for a caller without vocabulary.
	tags := make([]apigen.TagSummary, 0, len(rows))
	for _, row := range rows {
		tags = append(tags, tagSummary(row.Tag, row.AssetCount))
	}
	s.writeJSON(w, http.StatusOK, tags)
}

// CreateTag implements POST /api/v1/tags: the idempotent creator.
// A fresh name answers 201 with the new row; a name the caller
// already owns answers 200 with the EXISTING tag (nothing written,
// updatedAt untouched) -- the assignment face's set semantics
// applied to the vocabulary, never a 409. The unique constraint is
// the race guard: concurrent creators converge on one row, and the
// RETURNING id tells the arms apart (the generated id landed = fresh;
// any other id = the existing row the conflict arm returned).
func (s *server) CreateTag(w http.ResponseWriter, r *http.Request) {
	requester, ok := authFrom(r.Context())
	if !ok {
		s.rejectUnauthenticated(w)
		return
	}

	var request apigen.TagNameRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		s.writeValidationError(w, "request body must be valid JSON", []string{"body"})
		return
	}
	name, ok := s.tagName(w, request.Name)
	if !ok {
		return
	}

	ctx := r.Context()
	id, err := uuid.NewV7()
	if err != nil {
		s.internalError(w, r, "generate the tag id", err)
		return
	}
	landed, err := s.queries.CreateTag(ctx, dbgen.CreateTagParams{
		ID:      pgUUID(id),
		OwnerID: pgUUID(requester.userID),
		Name:    name,
	})
	if err != nil {
		s.internalError(w, r, "create the tag", err)
		return
	}

	// The honest post-write summary: the id RETURNING answers is the
	// tag that exists now (ours or the converged one), and the
	// re-read is the response shape the listing lists.
	row, err := s.queries.GetTagSummary(ctx, dbgen.GetTagSummaryParams{
		UserID: pgUUID(requester.userID),
		TagID:  landed,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		// The tag vanished between the write and the re-read (a
		// concurrent delete won the race): gone is gone, exactly
		// like an unknown id.
		s.writeTagNotFound(w)
		return
	}
	if err != nil {
		s.internalError(w, r, "load the created tag", err)
		return
	}

	status := http.StatusOK
	if uuid.UUID(landed.Bytes) == id {
		// The generated id landed: this was the fresh row. Any other
		// id is the row the conflict arm returned -- the convergence.
		status = http.StatusCreated
	}
	s.writeJSON(w, status, tagSummary(row.Tag, row.AssetCount))
}

// RenameTag implements PATCH /api/v1/tags/{tagId}: the rename.
// Renaming to the row's own name is the idempotent no-op (the
// statement's updated_at guard writes nothing); renaming to a name
// another of the caller's tags carries trips the unique constraint,
// the 409. The response is the tag's summary after the edit.
func (s *server) RenameTag(w http.ResponseWriter, r *http.Request, tagId openapi_types.UUID) {
	requester, ok := authFrom(r.Context())
	if !ok {
		s.rejectUnauthenticated(w)
		return
	}

	var request apigen.TagNameRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		s.writeValidationError(w, "request body must be valid JSON", []string{"body"})
		return
	}
	name, ok := s.tagName(w, request.Name)
	if !ok {
		return
	}

	tagID := uuid.UUID(tagId)
	callerID := pgUUID(requester.userID)

	// The gate first: an unknown or foreign tag answers the
	// single-shape 404 before any write is attempted.
	if _, ok := s.tagGate(w, r, requester.userID, tagID); !ok {
		return
	}

	// The rename itself: violation-driven collision detection (no
	// lookup-first race -- the constraint is the check, the upload
	// path's discipline). The unique constraint on (owner_id, name)
	// is the only one this statement can trip, so a 23505 can only
	// be the name collision.
	if _, err := s.queries.RenameTag(r.Context(), dbgen.RenameTagParams{
		Name:    name,
		TagID:   pgUUID(tagID),
		OwnerID: callerID,
	}); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" { // unique_violation
			s.writeTagNameTaken(w)
			return
		}
		if errors.Is(err, pgx.ErrNoRows) {
			// The tag vanished between the gate and the write.
			s.writeTagNotFound(w)
			return
		}
		s.internalError(w, r, "rename the tag", err)
		return
	}

	// The summary after the edit -- the response shape the listing
	// lists (the rename never moved the count, but the read is the
	// honest post-write state).
	row, err := s.queries.GetTagSummary(r.Context(), dbgen.GetTagSummaryParams{
		UserID: callerID,
		TagID:  pgUUID(tagID),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		s.writeTagNotFound(w)
		return
	}
	if err != nil {
		s.internalError(w, r, "load the renamed tag", err)
		return
	}
	s.writeJSON(w, http.StatusOK, tagSummary(row.Tag, row.AssetCount))
}

// DeleteTag implements DELETE /api/v1/tags/{tagId}: the delete. The
// row goes and every assignment row follows through the schema's
// cascade; zero rows (unknown, foreign, or a concurrent delete that
// won the race) is the single-shape 404. The owner scope in the
// statement is the whole gate.
func (s *server) DeleteTag(w http.ResponseWriter, r *http.Request, tagId openapi_types.UUID) {
	requester, ok := authFrom(r.Context())
	if !ok {
		s.rejectUnauthenticated(w)
		return
	}

	deleted, err := s.queries.DeleteTag(r.Context(), dbgen.DeleteTagParams{
		TagID:   pgUUID(tagId),
		OwnerID: pgUUID(requester.userID),
	})
	if err != nil {
		s.internalError(w, r, "delete the tag", err)
		return
	}
	if deleted == 0 {
		s.writeTagNotFound(w)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// AddTagAssets implements POST /api/v1/tags/{tagId}/assets: the
// assignment add with the batch surface's set semantics. Resolution
// and insert are one statement whose WHERE arm IS the resolution --
// every id must address an asset the caller can see through the
// album union (the caller-visible widening over the album-items add: a
// sharee tags what they see, with their own tag) -- so unknown, foreign and
// invisible ids are the skipped of the honest report, never errors,
// and an all-skipped request still answers 200. No live-state filter:
// the assignment row is per-user state written whatever the asset's
// live state, and the counts and views filter it (the favorites
// arms' discipline).
func (s *server) AddTagAssets(w http.ResponseWriter, r *http.Request, tagId openapi_types.UUID) {
	requester, ok := authFrom(r.Context())
	if !ok {
		s.rejectUnauthenticated(w)
		return
	}

	var request apigen.TagAssetsRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		s.writeValidationError(w, "request body must be valid JSON", []string{"body"})
		return
	}
	ids, ok := s.tagAssetIds(w, request.AssetIds)
	if !ok {
		return
	}

	if _, ok := s.tagGate(w, r, requester.userID, uuid.UUID(tagId)); !ok {
		return
	}

	added, err := s.queries.AddTagAssets(r.Context(), dbgen.AddTagAssetsParams{
		TagID:    pgUUID(tagId),
		AssetIds: pgUUIDs(ids),
		UserID:   pgUUID(requester.userID),
	})
	if err != nil {
		s.internalError(w, r, "add the tag to the assets", err)
		return
	}

	s.writeJSON(w, http.StatusOK, apigen.AddTagAssetsResponse{
		Added:   int(added),
		Skipped: len(ids) - int(added),
	})
}

// RemoveTagAssets implements DELETE /api/v1/tags/{tagId}/assets:
// the assignment remove. The caller's rows that address the
// request's ids leave; the ids that carry no assignment are the
// skipped of the honest report. No visibility predicate -- the
// removal tidies the caller's own set (a trashed member's assignment
// row still leaves).
func (s *server) RemoveTagAssets(w http.ResponseWriter, r *http.Request, tagId openapi_types.UUID) {
	requester, ok := authFrom(r.Context())
	if !ok {
		s.rejectUnauthenticated(w)
		return
	}

	var request apigen.TagAssetsRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		s.writeValidationError(w, "request body must be valid JSON", []string{"body"})
		return
	}
	ids, ok := s.tagAssetIds(w, request.AssetIds)
	if !ok {
		return
	}

	if _, ok := s.tagGate(w, r, requester.userID, uuid.UUID(tagId)); !ok {
		return
	}

	removed, err := s.queries.RemoveTagAssets(r.Context(), dbgen.RemoveTagAssetsParams{
		TagID:    pgUUID(tagId),
		AssetIds: pgUUIDs(ids),
	})
	if err != nil {
		s.internalError(w, r, "remove the tag from the assets", err)
		return
	}

	s.writeJSON(w, http.StatusOK, apigen.RemoveTagAssetsResponse{
		Removed: int(removed),
		Skipped: len(ids) - int(removed),
	})
}

// assetTagsForCaller reads one asset's tags joined through the
// CALLER's own tag rows (another account's tags on the same
// asset never surface), sorted by name case-insensitively with the
// tag id as the tiebreak -- the read both detail-serving endpoints
// (GET and PATCH) share, so the PATCH response carries the tags
// exactly as the GET serves them. Callers must have proven the
// asset caller-visible first; the tags carry no gate of their own
// (the faces read's discipline).
func (s *server) assetTagsForCaller(ctx context.Context, assetID, callerID pgtype.UUID) ([]apigen.AssetTagSummary, error) {
	rows, err := s.queries.ListAssetTagsForCaller(ctx, dbgen.ListAssetTagsForCallerParams{
		AssetID: assetID,
		UserID:  callerID,
	})
	if err != nil {
		return nil, err
	}
	// A non-nil empty slice: the fixed-shape contract wants an
	// array, never JSON null, for an untagged asset.
	tags := make([]apigen.AssetTagSummary, 0, len(rows))
	for _, row := range rows {
		tags = append(tags, apigen.AssetTagSummary{
			Id:   uuid.UUID(row.ID.Bytes),
			Name: row.Name,
		})
	}
	return tags, nil
}
