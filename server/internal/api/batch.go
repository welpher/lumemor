// The batch handler (the external lifecycle widened by the
// albums-as-sources and album-autonomy reworks): POST /assets/batch, the set-semantics
// surface of the curation actions. One request applies one action
// (favorite, unfavorite, hide, unhide, trash, restore) to a list of
// ids in ONE transaction: resolution decides which ids are the
// caller's to act on, the per-action primitives apply them (foreign
// and already-in-state ids are no-ops, never errors), and the
// report partitions the distinct ids into affected and missing.
// Events for the ids whose state actually changed publish after the
// commit, the same transition-only discipline as the single-asset
// endpoints.
//
// The trash and restore arms split on the row's storage kind: a
// blob row is marked in place (the existing
// queries, Live Photo pair included), while an external row's file
// MOVES -- into the server's trash area on trash, back to the
// recorded original location on restore -- ahead of the row's write
// and inside the batch's own failure story: a move the operating
// system refuses fails the WHOLE batch with the move's own error
// code, the transaction rolls back, and every file the batch
// already moved is compensated back to where the rolled-back rows
// still say it is. The album lock's delete arm joins the same
// story: a live external member of a LOCKED directory album refuses
// the trash with the curation writes' own 409 before any file
// moves, the rescan endpoint's refusal applied to the deletion
// itself.
package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	openapi_types "github.com/oapi-codegen/runtime/types"

	apigen "github.com/welpher/lumemor/server/internal/api/gen"
	dbgen "github.com/welpher/lumemor/server/internal/db/gen"
	"github.com/welpher/lumemor/server/internal/storage"
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

// The external lifecycle's move failures, the one error class the
// batch answers with its own codes (the move needs
// the operating system's permission, and no permission means the
// delete fails explicitly -- never a silent skip, never a nameless
// 500 either).
var (
	errBatchTrashMove   = errors.New("the external file could not be moved into the trash")
	errBatchRestoreMove = errors.New("the external file could not be moved back to its original location")

	// errBatchAlbumLocked is the album lock's delete arm (the
	// content-protection freeze widened from the rescan to the
	// deletion): the trash of a live external member of a locked
	// directory album is refused before any file moves, with the
	// curation writes' own 409 -- the owner unlocks first.
	errBatchAlbumLocked = errors.New("the asset's album is locked")
)

// batchFileMove records one external lifecycle move the batch
// performed ahead of its row write (trash: source to the trash
// home; restore: the trash home back to the recorded original
// location), together with the trash home the move created -- the
// compensation empties the home back out. The batch's failure paths
// compensate the collected moves in reverse order -- the files
// return to exactly where they were, so a rolled-back transaction
// leaves nothing half-moved.
type batchFileMove struct {
	from      string
	to        string
	trashHome string
}

// compensateBatchMoves undoes the collected moves (best-effort, in
// reverse order): the transaction rolled back, so the rows still
// describe the pre-move state and the files must match it -- each
// moved file returns to its source path, and the trash home a
// trash-move created goes away with it (it is empty the moment the
// file left). A failed compensation is logged for the operator --
// the row never claimed the moved file, so a stray copy is a leak
// the trash area's own layout keeps visible, never data loss.
func (s *server) compensateBatchMoves(r *http.Request, moves []batchFileMove) {
	for i := len(moves) - 1; i >= 0; i-- {
		if err := storage.MoveFileBack(moves[i].to, moves[i].from); err != nil {
			s.logger.Error("compensate the external lifecycle move of a failed batch",
				"error", err, "from", moves[i].to, "to", moves[i].from,
				"method", r.Method, "path", r.URL.Path)
			continue
		}
		if moves[i].trashHome != "" {
			// The home is empty now (the file just left); the remove
			// is a no-op when anything else sits inside it.
			_ = os.Remove(moves[i].trashHome)
		}
	}
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
	// what lets the whole batch commit as the one transaction it is
	// -- EXCEPT the external lifecycle's moves, the one failure that
	// must answer explicitly: a move that fails fails
	// the batch, the transaction rolls back, and the moves the batch
	// already performed are compensated (the files return to where
	// they were). changed collects the ids whose state actually
	// transitioned: the event discipline below reports changes, not
	// repeats.
	var changed []uuid.UUID
	var moves []batchFileMove
	for _, id := range ids {
		if _, ok := kinds[id]; !ok {
			continue // unresolvable: ignored, reported in missing
		}
		transitioned, move, err := s.applyBatchAction(ctx, queries, request.Action, requester.userID, id)
		if err != nil {
			// The compensation runs before the response is written:
			// the client reads a state where nothing moved.
			s.compensateBatchMoves(r, moves)
			switch {
			case errors.Is(err, errBatchAlbumLocked):
				// The lock's refusal rolls the batch back exactly
				// like a move failure: nothing moved, nothing was
				// written, the rows stay live.
				s.writeAlbumLocked(w)
			case errors.Is(err, errBatchTrashMove):
				s.logger.Error("the batch's external trash move failed", "error", err,
					"asset_id", id, "method", r.Method, "path", r.URL.Path)
				s.writeError(w, http.StatusInternalServerError, apigen.ASSETTRASHMOVEFAILED,
					"the file could not be moved into the trash", nil)
			case errors.Is(err, errBatchRestoreMove):
				s.logger.Error("the batch's external restore move failed", "error", err,
					"asset_id", id, "method", r.Method, "path", r.URL.Path)
				s.writeError(w, http.StatusInternalServerError, apigen.ASSETRESTOREMOVEFAILED,
					"the file could not be moved back to its original location", nil)
			default:
				s.internalError(w, r, "apply batch action", err)
			}
			return
		}
		if move != nil {
			moves = append(moves, *move)
		}
		if transitioned {
			changed = append(changed, id)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		// The rows never landed: the moved files return to where the
		// rolled-back rows still say they are.
		s.compensateBatchMoves(r, moves)
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

	// The album walls' convergence signal (the deletability
	// chain): the trash and the restore of an album member change the
	// album's LIVE membership -- the derived count the detail renders
	// and the members the wall walks -- while the membership rows
	// themselves never move. The album-narrowed wall converges
	// exactly through album.items-changed (the rescan's own restart
	// signal), so these two arms publish it for every album holding
	// the changed ids. The favorites and the hide/unhide arms never
	// move the trash lifecycle's live predicate and publish nothing
	// album-level; a refused batch never reaches this block at all.
	if request.Action == apigen.Trash || request.Action == apigen.Restore {
		s.publishAlbumMembershipEvents(r.Context(), requester.userID, changed)
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
// stay silent) and the external lifecycle's move when one performed
// (nil for every arm that moves nothing -- the caller collects the
// moves for its failure compensations). Favorite and unfavorite
// never report a transition -- they publish no events at all,
// whatever they write. Trash and restore are the trash lifecycle's
// own queries, pair predicates included for the blob rows (trashing
// a Live Photo carries its component video row along, so one
// request id can move two rows while counting once), while the
// external rows take the external move semantics: the file moves
// first (inside the batch's failure story), the row's write records
// the move's truth. The external arms are single-row by
// construction -- the rescan never pairs Live Photos -- and the
// guards keep the already-in-state ids no-ops (the move is skipped
// alongside the write, so a repeat trash touches no file either).
func (s *server) applyBatchAction(ctx context.Context, queries *dbgen.Queries, action apigen.BatchRequestAction, owner, id uuid.UUID) (bool, *batchFileMove, error) {
	switch action {
	case apigen.Favorite:
		_, err := queries.FavoriteAsset(ctx, dbgen.FavoriteAssetParams{
			UserID:  pgUUID(owner),
			AssetID: pgUUID(id),
		})
		return false, nil, err
	case apigen.Unfavorite:
		_, err := queries.UnfavoriteAsset(ctx, dbgen.UnfavoriteAssetParams{
			UserID:  pgUUID(owner),
			AssetID: pgUUID(id),
		})
		return false, nil, err
	case apigen.Hide, apigen.Unhide:
		rows, err := queries.SetAssetHidden(ctx, dbgen.SetAssetHiddenParams{
			ID:      pgUUID(id),
			OwnerID: pgUUID(owner),
			Hidden:  action == apigen.Hide,
		})
		return rows > 0, nil, err
	case apigen.Trash:
		row, err := queries.GetAsset(ctx, dbgen.GetAssetParams{
			ID:      pgUUID(id),
			OwnerID: pgUUID(owner),
		})
		if err != nil {
			return false, nil, err
		}
		if row.StorageKind == dbgen.StorageKindExternal {
			// The move semantics: a live row's file moves into the
			// trash home ahead of the write; a row already in the
			// trash (or an orphan the rescan marked, whose file is
			// not ours to move at all) skips the move and the write
			// below answers the zero-row no-op.
			var move *batchFileMove
			if !row.DeletedAt.Valid {
				// The album lock's delete gate, ahead of any file
				// work: a live external member of a LOCKED directory
				// album refuses here -- the rescan endpoint's own
				// refusal, the owner's content-protection freeze
				// over the album's photos. The already-trashed row
				// never reaches this arm (the idempotent no-op), and
				// the restore below never does either: recovery is
				// not the deletion the freeze governs.
				locked, err := queries.CountLockedDirectoryAlbumsOfAsset(ctx, pgUUID(id))
				if err != nil {
					return false, nil, err
				}
				if locked > 0 {
					return false, nil, errBatchAlbumLocked
				}
				trashPath, err := storage.MoveToTrash(s.dataDir, row.FilePath, id.String())
				if err != nil {
					return false, nil, fmt.Errorf("%w: %v", errBatchTrashMove, err)
				}
				move = &batchFileMove{
					from:      row.FilePath,
					to:        trashPath,
					trashHome: storage.TrashHome(s.dataDir, id.String()),
				}
				rows, err := queries.SoftDeleteExternalAsset(ctx, dbgen.SoftDeleteExternalAssetParams{
					ID:        pgUUID(id),
					OwnerID:   pgUUID(owner),
					TrashPath: pgtype.Text{String: trashPath, Valid: true},
				})
				return rows > 0, move, err
			}
			return false, nil, nil
		}
		rows, err := queries.SoftDeleteAsset(ctx, dbgen.SoftDeleteAssetParams{
			ID:      pgUUID(id),
			OwnerID: pgUUID(owner),
		})
		return rows > 0, nil, err
	case apigen.Restore:
		row, err := queries.GetAsset(ctx, dbgen.GetAssetParams{
			ID:      pgUUID(id),
			OwnerID: pgUUID(owner),
		})
		if err != nil {
			return false, nil, err
		}
		if row.StorageKind == dbgen.StorageKindExternal {
			// The move-back semantics: a row that sat in the trash
			// area (trash_path recorded) has its file moved back to
			// the original location ahead of the write; a live row,
			// or an orphan mark whose file was never ours, skips the
			// move and the write below answers the zero-row no-op.
			var move *batchFileMove
			if row.DeletedAt.Valid && row.TrashPath.Valid {
				if err := storage.MoveFileBack(row.TrashPath.String, row.OriginalPath.String); err != nil {
					return false, nil, fmt.Errorf("%w: %v", errBatchRestoreMove, err)
				}
				move = &batchFileMove{from: row.TrashPath.String, to: row.OriginalPath.String}
				rows, err := queries.RestoreExternalAsset(ctx, dbgen.RestoreExternalAssetParams{
					ID:      pgUUID(id),
					OwnerID: pgUUID(owner),
				})
				return rows > 0, move, err
			}
			return false, nil, nil
		}
		rows, err := queries.RestoreAsset(ctx, dbgen.RestoreAssetParams{
			ID:      pgUUID(id),
			OwnerID: pgUUID(owner),
		})
		return rows > 0, nil, err
	default:
		// Unreachable: the action was validated at the door. The arm
		// exists so the switch stays total over future enum values.
		return false, nil, fmt.Errorf("unknown batch action %q", action)
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
// discipline. The row-shaped variant in trash.go delegates
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
