// The batch handler (M2c Task 4): POST /assets/batch, the set-semantics
// surface of the curation actions. One request applies one action
// (favorite, unfavorite, hide, unhide, trash, restore) to a list of
// ids in ONE transaction: resolution decides which ids are the
// caller's to act on, the per-action primitives apply them (foreign
// and already-in-state ids are no-ops, never errors), and the report
// partitions the distinct ids into affected and missing. Events for
// the ids whose state actually changed publish after the commit, the
// same transition-only discipline as the single-asset endpoints.
package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	openapi_types "github.com/oapi-codegen/runtime/types"

	apigen "github.com/welpher/lumemor/server/internal/api/gen"
	dbgen "github.com/welpher/lumemor/server/internal/db/gen"
)

// batchIDLimit is the request bound pinned by the contract: the
// largest id list one batch request may carry. Above it the request
// is a client bug, not a workload -- 1000 ids keep one request a
// bounded transaction and a bounded response.
const batchIDLimit = 1000

// batchEventNames maps each state-carrying action onto the event its
// transitions announce. The favorites arms are absent on purpose:
// favorites are per-user state with no cross-user visibility effect,
// so no other session has anything to refresh -- the silence is the
// documented choice, not an omission.
var batchEventNames = map[apigen.BatchRequestAction]string{
	apigen.Trash:   "asset.trashed",
	apigen.Restore: "asset.restored",
	apigen.Hide:    "asset.updated",
	apigen.Unhide:  "asset.updated",
}

// BatchAssets implements POST /api/v1/assets/batch: one action over
// a list of asset ids, with the set-semantics report {affected,
// missing} partitioning the request's distinct ids.
func (s *server) BatchAssets(w http.ResponseWriter, r *http.Request) {
	requester, ok := authFrom(r.Context())
	if !ok {
		s.rejectUnauthenticated(w)
		return
	}

	// The body is a small JSON document (the route group's 1 MiB cap
	// bounds it before this decode); anything that does not decode --
	// malformed JSON, an id that is not a uuid -- is a client bug.
	var request apigen.BatchRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		s.writeValidationError(w, "request body must be valid JSON", []string{"body"})
		return
	}
	if !request.Action.Valid() {
		s.writeValidationError(w, "unknown batch action", []string{"action"})
		return
	}
	if len(request.Ids) == 0 {
		s.writeValidationError(w, "ids must carry at least one asset id", []string{"ids"})
		return
	}
	if len(request.Ids) > batchIDLimit {
		s.writeValidationError(w, fmt.Sprintf("ids exceed the batch limit of %d", batchIDLimit), []string{"ids"})
		return
	}

	// Repeated ids collapse to one application: the actions are set
	// operations, and a duplicated id addresses one row. The order of
	// first appearance is the deterministic application and report
	// order.
	ids := make([]uuid.UUID, 0, len(request.Ids))
	seen := make(map[uuid.UUID]bool, len(request.Ids))
	for _, raw := range request.Ids {
		id := uuid.UUID(raw)
		if !seen[id] {
			seen[id] = true
			ids = append(ids, id)
		}
	}

	ctx := r.Context()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		s.internalError(w, r, "begin batch transaction", err)
		return
	}
	// The rollback is a no-op after a successful commit; the context
	// is captured while the request is still live.
	defer func() { _ = tx.Rollback(ctx) }()
	queries := dbgen.New(tx)

	// Resolution: the guard each action resolves against -- the
	// curation arms own the assets they act on, the favorites arms
	// only need to see them (the album union, so a sharee favorites
	// what they see). A foreign or invisible id resolves to no row
	// exactly like an unknown one, so neither can be probed apart;
	// media_kind rides along for the events of the ids that end up
	// transitioning.
	resolved, err := resolveBatchTargets(ctx, queries, request.Action, requester.userID, ids)
	if err != nil {
		s.internalError(w, r, "resolve batch ids", err)
		return
	}
	kinds := make(map[uuid.UUID]dbgen.MediaKind, len(resolved))
	for _, row := range resolved {
		kinds[uuid.UUID(row.ID.Bytes)] = row.MediaKind
	}

	// One application per resolved id, in request order. The guards
	// inside each primitive make every item a no-op or a write, never
	// an error, so the loop has no per-item failure path -- which is
	// what lets the whole batch commit as the one transaction it is.
	// changed collects the ids whose state actually transitioned: the
	// event discipline below reports changes, not repeats.
	var changed []uuid.UUID
	for _, id := range ids {
		if _, ok := kinds[id]; !ok {
			continue // unresolvable: ignored, reported in missing
		}
		transitioned, err := applyBatchAction(ctx, queries, request.Action, requester.userID, id)
		if err != nil {
			s.internalError(w, r, "apply batch action", err)
			return
		}
		if transitioned {
			changed = append(changed, id)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		s.internalError(w, r, "commit batch", err)
		return
	}

	// The events follow the commit: a subscriber that reacts to
	// asset.trashed reads rows that are already trashed. One event
	// per changed id, the minimal id-plus-kind payload every asset
	// event carries; the idempotent no-op stayed silent.
	if name, publishes := batchEventNames[request.Action]; publishes {
		for _, id := range changed {
			s.publishAssetEventByID(name, id, kinds[id])
		}
	}

	// The report: affected counts every distinct id the server
	// resolved and applied -- including the idempotent no-ops -- and
	// missing lists the rest, in first-appearance order.
	missing := make([]openapi_types.UUID, 0)
	for _, id := range ids {
		if _, ok := kinds[id]; !ok {
			missing = append(missing, openapi_types.UUID(id))
		}
	}
	s.writeJSON(w, http.StatusOK, apigen.BatchResponse{
		Affected: len(kinds),
		Missing:  missing,
	})
}

// resolveBatchTargets runs the batch's id resolution against the
// guard the action implies: the favorites arms resolve the
// caller-VISIBLE ids (the album union -- the caller's own arms plus
// every album shared to them, so a sharee's favorites cover what
// they see), every other arm resolves the caller's OWNED ids
// (curation and the trash lifecycle are the owner's alone). The two
// resolutions answer the same row shape, so the handler's report
// partitioning never branches on the action.
func resolveBatchTargets(ctx context.Context, queries *dbgen.Queries, action apigen.BatchRequestAction, caller uuid.UUID, ids []uuid.UUID) ([]dbgen.ResolveBatchAssetsRow, error) {
	if action == apigen.Favorite || action == apigen.Unfavorite {
		rows, err := queries.ResolveVisibleAssets(ctx, dbgen.ResolveVisibleAssetsParams{
			Ids:    pgUUIDs(ids),
			Viewer: pgUUID(caller),
		})
		if err != nil {
			return nil, err
		}
		resolved := make([]dbgen.ResolveBatchAssetsRow, len(rows))
		for i, row := range rows {
			// The two resolutions answer the same (id, kind) shape;
			// the conversion is the identity on identical structs.
			resolved[i] = dbgen.ResolveBatchAssetsRow(row)
		}
		return resolved, nil
	}
	return queries.ResolveBatchAssets(ctx, dbgen.ResolveBatchAssetsParams{
		Ids:     pgUUIDs(ids),
		OwnerID: pgUUID(caller),
	})
}

// applyBatchAction runs one resolved id's share of the batch inside
// the caller's transaction, reporting whether the row's state
// actually changed (the signal the event discipline needs: repeats
// stay silent). Favorite and unfavorite never report a transition --
// they publish no events at all, whatever they write. Trash and
// restore are the trash lifecycle's own queries, pair predicates
// included: trashing a Live Photo carries its component video row
// along, so one request id can move two rows while counting once.
func applyBatchAction(ctx context.Context, queries *dbgen.Queries, action apigen.BatchRequestAction, owner, id uuid.UUID) (bool, error) {
	switch action {
	case apigen.Favorite:
		_, err := queries.FavoriteAsset(ctx, dbgen.FavoriteAssetParams{
			UserID:  pgUUID(owner),
			AssetID: pgUUID(id),
		})
		return false, err
	case apigen.Unfavorite:
		_, err := queries.UnfavoriteAsset(ctx, dbgen.UnfavoriteAssetParams{
			UserID:  pgUUID(owner),
			AssetID: pgUUID(id),
		})
		return false, err
	case apigen.Hide, apigen.Unhide:
		rows, err := queries.SetAssetHidden(ctx, dbgen.SetAssetHiddenParams{
			ID:      pgUUID(id),
			OwnerID: pgUUID(owner),
			Hidden:  action == apigen.Hide,
		})
		return rows > 0, err
	case apigen.Trash:
		rows, err := queries.SoftDeleteAsset(ctx, dbgen.SoftDeleteAssetParams{
			ID:      pgUUID(id),
			OwnerID: pgUUID(owner),
		})
		return rows > 0, err
	case apigen.Restore:
		rows, err := queries.RestoreAsset(ctx, dbgen.RestoreAssetParams{
			ID:      pgUUID(id),
			OwnerID: pgUUID(owner),
		})
		return rows > 0, err
	default:
		// Unreachable: the action was validated at the door. The arm
		// exists so the switch stays total over future enum values.
		return false, fmt.Errorf("unknown batch action %q", action)
	}
}

// pgUUIDs converts a slice of ids onto the wire form the array
// parameter of the resolution query takes.
func pgUUIDs(ids []uuid.UUID) []pgtype.UUID {
	converted := make([]pgtype.UUID, len(ids))
	for i, id := range ids {
		converted[i] = pgUUID(id)
	}
	return converted
}

// publishAssetEventByID reports one asset event onto the activity
// stream from the (id, kind) pair the batch resolution carries: the
// resource id and its media kind, nothing more -- the event audience
// discipline of M2a. The row-shaped variant in trash.go delegates
// here so every asset event renders the same payload.
func (s *server) publishAssetEventByID(name string, id uuid.UUID, kind dbgen.MediaKind) {
	if s.events == nil {
		return
	}
	s.events.Publish(name, map[string]string{
		"assetId": id.String(),
		"kind":    string(kind),
	})
}
