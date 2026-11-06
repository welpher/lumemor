// The asset attribute PATCH (M2c Task 5): PATCH /assets/{id}, the
// owner's partial edit of the two owner-level attributes -- the
// free-text description the detail panel renders, and the hidden
// switch that keeps the asset out of every live view. The permission
// model is owner-only, the attribute-editing row of the spec's
// matrix: no administrator override here (unlike the permanent
// delete), so the owner-scoped read decides the anti-enumeration 404
// and the writes carry the owner guard themselves. Set semantics
// throughout: the guards inside both writes make a repeat of the
// stored value the zero-row no-op, and the rows-affected counts --
// not the request -- decide whether anything changed; only an actual
// change publishes the single asset.updated event, after the commit.
package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	openapi_types "github.com/oapi-codegen/runtime/types"

	apigen "github.com/welpher/lumemor/server/internal/api/gen"
	dbgen "github.com/welpher/lumemor/server/internal/db/gen"
)

// descriptionLimit is the contract's bound on the asset description:
// 10,000 Unicode characters (runes) -- enough for a long caption,
// small enough that the column stays a bounded text document. The
// handler enforces it because the generated binder does not check
// the schema's maxLength.
const descriptionLimit = 10000

// PatchAsset implements PATCH /api/v1/assets/{id}: the partial edit
// of the description and the hidden switch. The response is the
// detail after the edit -- the same read and serializer the GET
// endpoint serves, so the client sees the applied values (and the
// idempotent no-op sees the row's current state unchanged).
func (s *server) PatchAsset(w http.ResponseWriter, r *http.Request, id openapi_types.UUID) {
	requester, ok := authFrom(r.Context())
	if !ok {
		s.rejectUnauthenticated(w)
		return
	}

	// The body is a small JSON document (the route group's 1 MiB cap
	// bounds it before this decode); anything that does not decode --
	// malformed JSON, a field of the wrong type -- is a client bug.
	// A field set to JSON null decodes as absent, so the
	// neither-field check below also refuses a body of pure nulls.
	var request apigen.AssetPatch
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		s.writeValidationError(w, "request body must be valid JSON", []string{"body"})
		return
	}
	if request.Description == nil && request.IsHidden == nil {
		s.writeValidationError(w, "the patch must carry description or isHidden", []string{"body"})
		return
	}
	if request.Description != nil && utf8.RuneCountInString(*request.Description) > descriptionLimit {
		s.writeValidationError(w,
			fmt.Sprintf("description exceeds the limit of %d characters", descriptionLimit),
			[]string{"description"})
		return
	}

	// The owner-scoped read decides the 404: unknown and foreign ids
	// resolve to no row and answer the byte-identical envelope, so
	// asset ids cannot be enumerated. The row also feeds the event's
	// media kind. No state filter: a trashed asset is editable like
	// it is readable (the detail answers 200 with isTrashed true),
	// and the edit resurfaces with the restore.
	row, err := s.queries.GetAsset(r.Context(), dbgen.GetAssetParams{
		ID:      pgUUID(id),
		OwnerID: pgUUID(requester.userID),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		s.writeAssetNotFound(w)
		return
	}
	if err != nil {
		s.internalError(w, r, "load asset for patch", err)
		return
	}

	// One transaction for the whole patch: when the request carries
	// both fields they land together or not at all -- a partial patch
	// never surfaces. Each write's set-semantics guard makes the
	// idempotent repeat match zero rows, and the rows-affected count
	// is the change signal of the event discipline below. The isHidden
	// arm is the batch's own SetAssetHidden -- one authority for the
	// switch -- and the description arm is its owner-guarded sibling.
	ctx := r.Context()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		s.internalError(w, r, "begin patch transaction", err)
		return
	}
	// The rollback is a no-op after a successful commit; the context
	// is captured while the request is still live.
	defer func() { _ = tx.Rollback(ctx) }()
	queries := dbgen.New(tx)

	changed := false
	if request.Description != nil {
		rows, err := queries.UpdateAssetDescription(ctx, dbgen.UpdateAssetDescriptionParams{
			ID:          pgUUID(id),
			OwnerID:     pgUUID(requester.userID),
			Description: *request.Description,
		})
		if err != nil {
			s.internalError(w, r, "update asset description", err)
			return
		}
		if rows > 0 {
			changed = true
		}
	}
	if request.IsHidden != nil {
		rows, err := queries.SetAssetHidden(ctx, dbgen.SetAssetHiddenParams{
			ID:      pgUUID(id),
			OwnerID: pgUUID(requester.userID),
			Hidden:  *request.IsHidden,
		})
		if err != nil {
			s.internalError(w, r, "update asset hidden switch", err)
			return
		}
		if rows > 0 {
			changed = true
		}
	}

	if err := tx.Commit(ctx); err != nil {
		s.internalError(w, r, "commit asset patch", err)
		return
	}

	// One event per patch that actually changed something, after the
	// commit: a subscriber that reacts to asset.updated reads rows
	// that already carry the new values. Both fields are owner-level
	// asset state -- the description is detail-panel content with no
	// timeline effect of its own, but it is asset state the same --
	// so one asset.updated covers whichever field changed; the
	// idempotent no-op stays silent, the transition-only discipline
	// of the batch and the trash lifecycle.
	if changed {
		s.publishAssetEventByID("asset.updated", uuid.UUID(id), row.MediaKind)
	}

	// The response: the detail after the edit, fetched fresh so it
	// reflects the applied values. A row deleted between the commit
	// and this read answers the 404 of a gone asset, exactly like
	// the delete endpoint's repeat. The faces ride along through the
	// shared read: the PATCH answer is the detail the GET serves,
	// people section included. The read rides the caller-visible
	// union like the GET's own -- the gate above already proved the
	// caller owns the row, and an owner's sync membership is what
	// the union resolves it through.
	detail, err := s.queries.GetAssetDetail(r.Context(), dbgen.GetAssetDetailParams{
		ID:     pgUUID(id),
		Viewer: pgUUID(requester.userID),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		s.writeAssetNotFound(w)
		return
	}
	if err != nil {
		s.internalError(w, r, "load patched asset detail", err)
		return
	}
	faces, err := s.assetFacesForCaller(r.Context(), pgUUID(id), pgUUID(requester.userID))
	if err != nil {
		s.internalError(w, r, "load patched asset faces", err)
		return
	}
	tags, err := s.assetTagsForCaller(r.Context(), pgUUID(id), pgUUID(requester.userID))
	if err != nil {
		s.internalError(w, r, "load patched asset tags", err)
		return
	}
	s.writeJSON(w, http.StatusOK, assetDetailFrom(detail, faces, tags))
}
