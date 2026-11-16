// The trash handlers (the external lifecycle widened by
// albums-as-sources): the owner-scoped trash listing with its retention countdown,
// the idempotent restore, and the explicit permanent delete. The
// permission model splits along the two delete surfaces: the trash
// (soft delete, restore) is strictly owner-scoped -- a foreign id
// answers the anti-enumeration 404, byte-identical with an unknown
// one -- while the explicit permanent delete also admits
// administrators, the application layer of the two-layer
// delete gate.
//
// The file half splits on the row's storage kind: a
// blob row's file never moves through the lifecycle (the
// shared-file rule spans both storage eras: legacy content-addressed
// rows store identical bytes of any owner under one physical file,
// the original going only when no other row references it -- see
// CountAssetFileReferences -- while template-era rows are
// per-owner and unique; the derived copies are keyed by the asset
// id alone and always go), while an external row's file MOVES --
// into data/trash/<assetId>/ when the owner trashes it (the move
// runs in the batch handler, before the row's write, and its
// failure fails the whole batch explicitly), back to the recorded
// original location on restore, and away entirely at expiry or
// permanent delete. The trash listing stays honest for both kinds;
// the rescan's orphan marks (deleted externals with no trash file)
// never appear in it.
package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	openapi_types "github.com/oapi-codegen/runtime/types"

	apigen "github.com/welpher/lumemor/server/internal/api/gen"
	"github.com/welpher/lumemor/server/internal/db"
	dbgen "github.com/welpher/lumemor/server/internal/db/gen"
	"github.com/welpher/lumemor/server/internal/storage"
)

// trashListCap bounds the trash listing: the view answers at most
// this many of the newest deletions. The trash is a recovery view
// for recent deletes, and rows age out through expiry on their own,
// so the bound is a response-size guard, not a pagination seam.
const trashListCap = 200

// GetTrash implements GET /api/v1/trash: the caller's trashed assets,
// newest deletion first, each with the countdown of whole days until
// the expiry job removes it.
func (s *server) GetTrash(w http.ResponseWriter, r *http.Request) {
	requester, ok := authFrom(r.Context())
	if !ok {
		s.rejectUnauthenticated(w)
		return
	}

	// The retention window of the countdown. A corrupt setting row
	// degrades to the default (the accessor's contract) so a bad row
	// cannot stop the trash view; the error is logged for the
	// operator to fix.
	retention, err := db.TrashRetentionDays(r.Context(), s.queries)
	if err != nil {
		s.logger.Error("read trash retention setting", "error", err)
	}

	rows, err := s.queries.ListTrash(r.Context(), dbgen.ListTrashParams{
		OwnerID:       pgUUID(requester.userID),
		RetentionDays: int32(retention),
		LimitCount:    trashListCap,
	})
	if err != nil {
		s.internalError(w, r, "list trash", err)
		return
	}

	response := apigen.TrashResponse{
		Items: make([]apigen.TrashItem, 0, len(rows)),
		// The note's number rides the response: the same setting the
		// countdowns above were computed against, degraded to the
		// default on a corrupt row exactly like they are.
		RetentionDays: int(retention),
	}
	for _, row := range rows {
		response.Items = append(response.Items, trashItemFrom(row))
	}
	s.writeJSON(w, http.StatusOK, response)
}

// RestoreAsset implements POST /api/v1/trash/{id}/restore: clears
// the trash mark of one asset (its linked Live Photo component row
// rides along, in the query), following set semantics -- a row that
// is already live matches the guard's zero rows and the response
// stays the 204 success, because the row is already in the desired
// state. Only an unknown or foreign id answers 404.
//
// The external row splits from the blob (the restore row's own
// semantics): its file was MOVED into the server's trash area
// when the owner trashed it, so the restore moves it back to the
// row's recorded original location FIRST -- a location that can no
// longer receive the file (the source folder unmounted, the
// directory gone) fails the restore explicitly with 500
// ASSET_RESTORE_MOVE_FAILED, the row staying trashed and the file
// staying in the trash area exactly where a retry after remounting
// finds it.
func (s *server) RestoreAsset(w http.ResponseWriter, r *http.Request, id openapi_types.UUID) {
	requester, ok := authFrom(r.Context())
	if !ok {
		s.rejectUnauthenticated(w)
		return
	}

	// The owner-scoped read decides the 404: unknown and foreign ids
	// resolve to no row and answer the byte-identical envelope, so
	// asset ids cannot be enumerated. A row already purged by the
	// expiry job lands here too -- a purge is not reversible.
	row, err := s.queries.GetAsset(r.Context(), dbgen.GetAssetParams{
		ID:      pgUUID(id),
		OwnerID: pgUUID(requester.userID),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		s.writeAssetNotFound(w)
		return
	}
	if err != nil {
		s.internalError(w, r, "load asset for restore", err)
		return
	}

	// The external arm: the file moves back to the recorded original
	// location before the row's mark clears, so the row going live
	// and the file's return land together. The move failure answers
	// its own code -- the pin's explicit failure, never a silent
	// half-restore. A row that never sat in the move semantics (an
	// orphan the rescan marked, trash_path NULL) skips the move: its
	// file was never ours, there is nothing to move back.
	if row.StorageKind == dbgen.StorageKindExternal && row.DeletedAt.Valid && row.TrashPath.Valid {
		if err := storage.MoveFileBack(row.TrashPath.String, row.OriginalPath.String); err != nil {
			s.logger.Error("move the external file back for restore", "error", err,
				"asset_id", id, "method", r.Method, "path", r.URL.Path)
			s.writeError(w, http.StatusInternalServerError, apigen.ASSETRESTOREMOVEFAILED,
				"the file could not be moved back to its original location", nil)
			return
		}
		restored, err := s.queries.RestoreExternalAsset(r.Context(), dbgen.RestoreExternalAssetParams{
			ID:      pgUUID(id),
			OwnerID: pgUUID(requester.userID),
		})
		if err != nil {
			// The row write failed after the move: put the file back
			// where the trash kept it, so the row (still trashed) and
			// the file stay consistent and the retry starts clean.
			if _, moveBackErr := storage.MoveToTrash(s.dataDir, row.OriginalPath.String, id.String()); moveBackErr != nil {
				// The compensation failed too: the row still names
				// the trash path it no longer sits at. The expiry's
				// removal treats a missing trash file as success, so
				// the lifecycle stays honest; the operator log line
				// carries the inconsistency.
				s.logger.Error("compensate the failed external restore", "error", moveBackErr,
					"asset_id", id, "method", r.Method, "path", r.URL.Path)
			}
			s.internalError(w, r, "restore external asset", err)
			return
		}
		if restored > 0 {
			// Only a state change announces itself; the idempotent
			// no-op stays silent.
			s.publishAssetEvent("asset.restored", row)
			// The album walls' convergence signal rides the same state
			// change: the restored member rejoined every holding
			// album's live membership, and the album-narrowed wall
			// restarts its walk on this event (the deletability
			// chain -- the batch arm publishes the same signal).
			s.publishAlbumMembershipEvents(r.Context(), requester.userID, []uuid.UUID{uuid.UUID(id)})
		}
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// The blob arm's restore write itself: the WHERE guard makes
	// re-restoring a live row (or a restore racing another restore)
	// a zero-row no-op, not an error -- the row is already where the
	// caller wants it.
	restored, err := s.queries.RestoreAsset(r.Context(), dbgen.RestoreAssetParams{
		ID:      pgUUID(id),
		OwnerID: pgUUID(requester.userID),
	})
	if err != nil {
		s.internalError(w, r, "restore asset", err)
		return
	}
	if restored > 0 {
		// Only a state change announces itself; the idempotent no-op
		// stays silent.
		s.publishAssetEvent("asset.restored", row)
		// The same album signal as the external arm: a blob member's
		// restore rejoins its albums' live membership identically.
		s.publishAlbumMembershipEvents(r.Context(), requester.userID, []uuid.UUID{uuid.UUID(id)})
	}
	w.WriteHeader(http.StatusNoContent)
}

// trashSweepChunk bounds one chunk of the whole-trash sweeps (the
// restore-all and the empty): the same shape the expiry job's run
// pins (500 rows per pass), the caller looping the chunked listing
// until it comes back empty. The bound keeps one request's sweep a
// bounded pass of transactions and file operations however large the
// trash grew; a family instance's trash rarely reaches it in one
// lifetime.
const trashSweepChunk = 500

// restoreTrashedRow restores one trashed row under the semantics the
// single-asset restore pins, for the whole-trash arms that apply the
// same rules to many rows: the external arm's file moves back to the
// recorded original location first (a refusal fails the row with the
// shared move-failure class -- the sweep answers it explicitly like
// the single restore), the blob arm's guarded write makes a row
// already gone from the trash a zero-row no-op, and a row write that
// failed after the move is compensated (the file returns to the
// trash area, the retry starts clean). It reports whether the row's
// state actually changed -- the signal the event discipline needs.
//
// The row arrives from the sweep's listing (the full row shape the
// move semantics need), so the guards below re-check nothing the
// listing already proved: the row is the caller's and it is trashed.
func (s *server) restoreTrashedRow(ctx context.Context, row dbgen.Asset) (bool, error) {
	if row.StorageKind == dbgen.StorageKindExternal && row.TrashPath.Valid {
		if err := storage.MoveFileBack(row.TrashPath.String, row.OriginalPath.String); err != nil {
			return false, fmt.Errorf("%w: %v", errExternalRestoreMove, err)
		}
		restored, err := s.queries.RestoreExternalAsset(ctx, dbgen.RestoreExternalAssetParams{
			ID:      row.ID,
			OwnerID: row.OwnerID,
		})
		if err != nil {
			// The row write failed after the move: put the file back
			// where the trash kept it, so the row (still trashed) and
			// the file stay consistent and the retry starts clean --
			// the single restore's compensation, row for row.
			if _, moveBackErr := storage.MoveToTrash(s.dataDir, row.OriginalPath.String, uuid.UUID(row.ID.Bytes).String()); moveBackErr != nil {
				// The compensation failed too: the row still names the
				// trash path it no longer sits at. The expiry's removal
				// treats a missing trash file as success, so the
				// lifecycle stays honest; the operator log line
				// carries the inconsistency.
				s.logger.Error("compensate the failed external restore", "error", moveBackErr,
					"asset_id", uuid.UUID(row.ID.Bytes))
			}
			return false, err
		}
		return restored > 0, nil
	}

	// The blob arm's guarded write: the pair predicate carries a Live
	// Photo's component row along, and the deleted_at guard makes a
	// row restored or purged since the listing a zero-row no-op.
	restored, err := s.queries.RestoreAsset(ctx, dbgen.RestoreAssetParams{
		ID:      row.ID,
		OwnerID: row.OwnerID,
	})
	return restored > 0, err
}

// RestoreAllTrash implements POST /api/v1/trash/restore-all: the
// whole-trash restore. Every row of the caller's trash right now
// returns to the live views, chunk by chunk (the bounded sweep), each
// chunk's rows applying the single-asset restore's semantics row by
// row. The rows that actually left the trash announce one
// asset.restored each, the album walls get the same membership
// convergence signal per chunk, and the count of announced rows is
// the response's affected -- an empty trash answers 0 (the
// idempotent no-op; there is no not-found surface, the sweep scans
// what is in the trash at the time of the request).
//
// The loop's termination: a chunk's rows either transition (they are
// live and excluded from the next listing) or fail the request
// outright -- a row that stays trashed through a zero-row no-op left
// the trash concurrently, so the next chunk cannot contain it again.
func (s *server) RestoreAllTrash(w http.ResponseWriter, r *http.Request) {
	requester, ok := authFrom(r.Context())
	if !ok {
		s.rejectUnauthenticated(w)
		return
	}

	ctx := r.Context()
	affected := 0
	for {
		rows, err := s.queries.ListTrashedForSweep(ctx, dbgen.ListTrashedForSweepParams{
			OwnerID:    pgUUID(requester.userID),
			LimitCount: trashSweepChunk,
		})
		if err != nil {
			s.internalError(w, r, "list the trash for restore-all", err)
			return
		}
		if len(rows) == 0 {
			break
		}
		restored := make([]uuid.UUID, 0, len(rows))
		for _, row := range rows {
			transitioned, err := s.restoreTrashedRow(ctx, row)
			if err != nil {
				if errors.Is(err, errExternalRestoreMove) {
					// The explicit failure of the single restore's
					// external arm: the row stays trashed, the file
					// stays in the trash area, and the rows restored
					// before it stay restored -- a retry after the
					// source location returns finds the rest.
					s.logger.Error("move the external file back for restore-all", "error", err,
						"asset_id", uuid.UUID(row.ID.Bytes), "method", r.Method, "path", r.URL.Path)
					s.writeError(w, http.StatusInternalServerError, apigen.ASSETRESTOREMOVEFAILED,
						"the file could not be moved back to its original location", nil)
					return
				}
				s.internalError(w, r, "restore asset in restore-all", err)
				return
			}
			if !transitioned {
				continue
			}
			affected++
			s.publishAssetEventByID("asset.restored", uuid.UUID(row.ID.Bytes), row.MediaKind)
			restored = append(restored, uuid.UUID(row.ID.Bytes))
		}
		if len(restored) > 0 {
			// The album walls' convergence signal (the deletability
			// chain), per chunk: the restored members rejoined every
			// holding album's live membership.
			s.publishAlbumMembershipEvents(ctx, requester.userID, restored)
		}
		if len(rows) < trashSweepChunk {
			break
		}
	}
	s.writeJSON(w, http.StatusOK, apigen.TrashSweepResponse{Affected: affected})
}

// EmptyTrash implements POST /api/v1/trash/empty: the whole-trash
// delete. Every row of the caller's trash right now goes for good
// under the permanent delete's rules, row by row (the pair-aware
// purge the single delete shares), each purged row announcing one
// asset.deleted -- the connected surfaces converge exactly the way
// they follow the expiry job's purges. An empty trash answers 0 (the
// idempotent no-op; the sweep scans what is in the trash at the time
// of the request, so the single delete's 404 never appears). The
// caller confirms first, in its own client; the endpoint itself is
// the caller's own trash alone.
func (s *server) EmptyTrash(w http.ResponseWriter, r *http.Request) {
	requester, ok := authFrom(r.Context())
	if !ok {
		s.rejectUnauthenticated(w)
		return
	}

	ctx := r.Context()
	affected := 0
	for {
		rows, err := s.queries.ListTrashedForSweep(ctx, dbgen.ListTrashedForSweepParams{
			OwnerID:    pgUUID(requester.userID),
			LimitCount: trashSweepChunk,
		})
		if err != nil {
			s.internalError(w, r, "list the trash for empty", err)
			return
		}
		if len(rows) == 0 {
			break
		}
		for _, row := range rows {
			deleted, err := s.purgeAssetWithPair(r, row)
			if err != nil {
				s.internalError(w, r, "empty the trash", err)
				return
			}
			if !deleted {
				// A concurrent delete or restore won the race: the row
				// left the trash before this sweep's purge landed,
				// nothing was destroyed.
				continue
			}
			affected++
			// The event follows the row's own committed purge: the
			// same after-the-commit discipline the batch arms keep,
			// applied per row of the sweep.
			s.publishAssetEventByID("asset.deleted", uuid.UUID(row.ID.Bytes), row.MediaKind)
		}
		if len(rows) < trashSweepChunk {
			break
		}
	}
	s.writeJSON(w, http.StatusOK, apigen.TrashSweepResponse{Affected: affected})
}

// BatchTrash implements POST /api/v1/trash/batch: the trash view's
// multi-select surface, one action over a list of ids. The
// resolution mirrors the assets batch's set semantics with a
// TRASH-SCOPED guard: only an id in the caller's trash at the time of
// the request resolves -- unknown, foreign, already-live and
// already-purged ids land in the response's missing list together
// (the anti-enumeration discipline holds; the caller handed the ids
// over themselves, so reporting them back hides nothing). The
// resolved ids' actions run per row under the single-row semantics
// the trash endpoints pin, and the rows whose state actually changed
// announce the same events per row. affected counts the ids the
// resolution processed -- a row that left the trash between the
// listing and the click is a missing row, not a no-op.
func (s *server) BatchTrash(w http.ResponseWriter, r *http.Request) {
	requester, ok := authFrom(r.Context())
	if !ok {
		s.rejectUnauthenticated(w)
		return
	}

	// The body is a small JSON document (the route group's 1 MiB cap
	// bounds it before this decode); anything that does not decode --
	// malformed JSON, an id that is not a uuid -- is a client bug,
	// the same reading the assets batch answers.
	var request apigen.TrashBatchRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		s.writeValidationError(w, "request body must be valid JSON", []string{"body"})
		return
	}
	if !request.Action.Valid() {
		s.writeValidationError(w, "unknown trash batch action", []string{"action"})
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
	// operations over trash rows, and a duplicated id addresses one
	// row. The order of first appearance is the deterministic
	// application and report order.
	ids := make([]uuid.UUID, 0, len(request.Ids))
	seen := make(map[uuid.UUID]bool, len(request.Ids))
	for _, raw := range request.Ids {
		id := uuid.UUID(raw)
		if !seen[id] {
			seen[id] = true
			ids = append(ids, id)
		}
	}

	// The trash-scoped resolution: the (id, kind) pairs of the rows
	// that are in the caller's trash right now.
	resolved, err := s.queries.ResolveTrashedAssets(r.Context(), dbgen.ResolveTrashedAssetsParams{
		Ids:     pgUUIDs(ids),
		OwnerID: pgUUID(requester.userID),
	})
	if err != nil {
		s.internalError(w, r, "resolve trash batch ids", err)
		return
	}
	kinds := make(map[uuid.UUID]dbgen.MediaKind, len(resolved))
	for _, row := range resolved {
		kinds[uuid.UUID(row.ID.Bytes)] = row.MediaKind
	}

	// One application per resolved id, in request order. A resolved
	// row that left the trash between the resolution and its own
	// application (another device's action) is the guard's no-op: it
	// stays counted in affected (it was in the trash at request
	// time) and stays silent (no event for a state change that never
	// happened here).
	ctx := r.Context()
	affected := len(kinds)
	restored := make([]uuid.UUID, 0, len(ids))
	for _, id := range ids {
		kind, ok := kinds[id]
		if !ok {
			continue // unresolvable: reported in missing
		}
		row, err := s.queries.GetAsset(ctx, dbgen.GetAssetParams{
			ID:      pgUUID(id),
			OwnerID: pgUUID(requester.userID),
		})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				// The row left the trash (and the owner's rows) since
				// the resolution: nothing to apply.
				continue
			}
			s.internalError(w, r, "load asset for trash batch", err)
			return
		}
		switch request.Action {
		case apigen.TrashBatchRequestActionRestore:
			transitioned, err := s.restoreTrashedRow(ctx, row)
			if err != nil {
				if errors.Is(err, errExternalRestoreMove) {
					s.logger.Error("move the external file back for the trash batch restore", "error", err,
						"asset_id", id, "method", r.Method, "path", r.URL.Path)
					s.writeError(w, http.StatusInternalServerError, apigen.ASSETRESTOREMOVEFAILED,
						"the file could not be moved back to its original location", nil)
					return
				}
				s.internalError(w, r, "restore asset in trash batch", err)
				return
			}
			if transitioned {
				s.publishAssetEventByID("asset.restored", id, kind)
				restored = append(restored, id)
			}
		case apigen.TrashBatchRequestActionDelete:
			deleted, err := s.purgeAssetWithPair(r, row)
			if err != nil {
				s.internalError(w, r, "delete asset in trash batch", err)
				return
			}
			if deleted {
				s.publishAssetEventByID("asset.deleted", id, kind)
			}
		default:
			// Unreachable: the action was validated at the door.
			s.internalError(w, r, "unknown trash batch action", fmt.Errorf("action %q", request.Action))
			return
		}
	}

	// The album walls' convergence signal rides the restore arm: the
	// restored members rejoined every holding album's live
	// membership. The delete arm announces through its per-row
	// asset.deleted events alone.
	if len(restored) > 0 {
		s.publishAlbumMembershipEvents(ctx, requester.userID, restored)
	}

	// The report: affected counts every distinct id the resolution
	// processed; missing lists the rest, in first-appearance order.
	missing := make([]openapi_types.UUID, 0)
	for _, id := range ids {
		if _, ok := kinds[id]; !ok {
			missing = append(missing, openapi_types.UUID(id))
		}
	}
	s.writeJSON(w, http.StatusOK, apigen.TrashBatchResponse{
		Affected: affected,
		Missing:  missing,
	})
}

// DeleteAsset implements DELETE /api/v1/assets/{id}: the explicit
// permanent deletion. The row goes whatever its state (this is the
// surface beside the trash, not a detour through it), the Live
// Photo's component video row follows its photo, and the files go
// under the shared-blob rule. The permission gate is the owner or an
// administrator; any other caller gets the anti-enumeration 404. A
// request that names the component video row directly deletes just
// that row -- the photo it belongs to survives, its component link
// nulling out through the schema's ON DELETE SET NULL.
func (s *server) DeleteAsset(w http.ResponseWriter, r *http.Request, id openapi_types.UUID) {
	requester, ok := authFrom(r.Context())
	if !ok {
		s.rejectUnauthenticated(w)
		return
	}

	// The row is read WITHOUT the owner filter because the caller may
	// be an administrator; the refusal below, not the read, shapes
	// the response, so the enumeration discipline holds.
	row, err := s.queries.GetAssetByID(r.Context(), pgUUID(id))
	if errors.Is(err, pgx.ErrNoRows) {
		s.writeAssetNotFound(w)
		return
	}
	if err != nil {
		s.internalError(w, r, "load asset for delete", err)
		return
	}
	if uuid.UUID(row.OwnerID.Bytes) != requester.userID && requester.role != dbgen.UserRoleAdmin {
		s.writeAssetNotFound(w)
		return
	}

	// The pair-aware purge: the stills' reads, the row's purge, the
	// component follow-up and the orphaned stills' announcements are
	// one shared helper -- the whole-trash arms (the empty and the
	// trash batch's delete) run the same rules per row, so the
	// single-asset surface is not its own special case.
	deleted, err := s.purgeAssetWithPair(r, row)
	if err != nil {
		s.internalError(w, r, "delete asset permanently", err)
		return
	}
	if !deleted {
		// A concurrent delete got there first: the asset no longer
		// exists, exactly like an unknown id.
		s.writeAssetNotFound(w)
		return
	}

	s.publishAssetEvent("asset.deleted", row)
	w.WriteHeader(http.StatusNoContent)
}

// purgeAssetWithPair removes one asset row under the permanent
// delete's rules and, when the row is a Live Photo still, the
// component video row the delete leaves unreferenced. It reports
// whether THIS call deleted the row: false means a concurrent delete
// won the race and the row is already gone.
//
// A COMPONENT'S DELETE CHANGES ITS STILLS' SHAPE: the link nulls
// through the schema, so the stills are read BEFORE the row goes
// (afterwards nothing points at it) and each one announces its change
// below -- the same convergence the pair maintenance publishes, so an
// open wall drops the type mark instead of keeping a mark for a
// motion that no longer exists.
func (s *server) purgeAssetWithPair(r *http.Request, row dbgen.Asset) (bool, error) {
	var orphanedStills []dbgen.ListLivePhotoStillsOfMotionRow
	if row.IsComponent {
		stills, err := s.queries.ListLivePhotoStillsOfMotion(r.Context(), row.ID)
		if err != nil {
			return false, err
		}
		orphanedStills = stills
	}

	// The photo row goes before its linked component video row: the
	// component delete would otherwise trip the foreign key the photo
	// still holds on it.
	deleted, err := s.purgeAssetRow(r, row)
	if err != nil {
		return false, err
	}
	if !deleted {
		return false, nil
	}

	if row.LivePhotoVideoID.Valid {
		// THE LAST STILL KEEPS THE MOTION. A burst (or a duplicated
		// pair) can leave two stills pointing at one motion; the motion
		// goes with the last of them, never with the first. The photo
		// row is already gone, so the remaining count is exactly "does
		// anything else still need these bytes".
		references, err := s.queries.CountLivePhotoReferences(r.Context(), row.LivePhotoVideoID)
		if err != nil {
			return true, err
		}
		var component dbgen.Asset
		if references == 0 {
			component, err = s.queries.GetAssetByID(r.Context(), row.LivePhotoVideoID)
			if err != nil && !errors.Is(err, pgx.ErrNoRows) {
				return true, err
			}
		}
		if references == 0 && err == nil {
			// The component is never the client-visible asset, so it
			// purges without its own announcement; its file answers
			// the same reference rule through the same helper.
			if _, err := s.purgeAssetRow(r, component); err != nil {
				return true, err
			}
		}
	}

	for _, still := range orphanedStills {
		if still.DeletedAt.Valid {
			// A trashed still has no surface to converge (the trash
			// view renders no type mark).
			continue
		}
		s.events.Publish("asset.updated", map[string]string{
			"assetId": uuid.UUID(still.ID.Bytes).String(),
			"kind":    string(dbgen.MediaKindImage),
		})
	}
	return true, nil
}

// purgeAssetRow removes one asset row and its on-disk artifacts: the
// shared-blob reference count is taken first, the row delete runs,
// and the file removal follows the commit -- a failure in between is
// a storage leak, never data loss, so file errors are logged and
// swallowed while the row's delete stands. It reports whether THIS
// call deleted the row; false means a concurrent delete won the race
// and the row is already gone.
//
// The row delete and the face cluster sweep land in one transaction:
// the cascade removes the asset's faces, and a face that is a
// cluster's representative must re-home or dissolve BEFORE the
// cascade touches it (the schema's foreign key holds every
// representative to a live face -- see faces.sql's sweep queries).
//
// The file half splits on the storage kind (the expiry row's own
// semantics): a blob row's original goes under the shared-file
// rule (the referencing rows decide -- another row on the same
// file_path keeps it, only the row goes), while an external row's
// only file of ours is the TRASH COPY a user-initiated trash landed
// in data/trash/<assetID>/ -- a live external's source file is the
// family's own and is never touched, whatever the caller asked the
// server to delete.
func (s *server) purgeAssetRow(r *http.Request, row dbgen.Asset) (bool, error) {
	ctx := r.Context()
	assetID := uuid.UUID(row.ID.Bytes)

	// The blob arm's reference count first: it excludes the row
	// itself by id, so it stays meaningful whichever side of the row
	// delete it runs on, and the task's pinned flow -- check, delete
	// the row, then decide the file -- reads in exactly that order.
	// The external arm counts nothing: its only file of ours is the
	// trash copy, keyed by the asset id alone.
	var refs int64
	if row.StorageKind != dbgen.StorageKindExternal {
		var err error
		if refs, err = s.queries.CountAssetFileReferences(ctx, dbgen.CountAssetFileReferencesParams{
			FilePath: row.FilePath,
			ID:       row.ID,
		}); err != nil {
			return false, err
		}
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return false, fmt.Errorf("begin the asset purge transaction: %w", err)
	}
	// The rollback is a no-op after a successful commit; arguments are
	// captured while the request context is still live.
	defer func() { _ = tx.Rollback(ctx) }()
	queries := dbgen.New(tx)

	// The sweep over the asset's faces (every model): the
	// representative re-homes to a surviving member, a cluster left
	// without members dissolves with its user_face_labels.
	if err := queries.RehomeClusterRepresentatives(ctx, dbgen.RehomeClusterRepresentativesParams{
		AssetID: row.ID,
	}); err != nil {
		return false, fmt.Errorf("rehome the clusters of the asset's faces: %w", err)
	}
	if err := queries.DissolveVacatedClusters(ctx, dbgen.DissolveVacatedClustersParams{
		AssetID: row.ID,
	}); err != nil {
		return false, fmt.Errorf("dissolve the clusters the asset's faces held: %w", err)
	}

	deleted, err := queries.DeleteAssetPermanently(ctx, row.ID)
	if err != nil {
		return false, err
	}
	if err := tx.Commit(ctx); err != nil {
		return false, fmt.Errorf("commit the asset purge: %w", err)
	}
	if deleted == 0 {
		return false, nil
	}

	// The row is committed gone; the files follow best-effort. The
	// context is captured while the request is still live.
	if row.StorageKind == dbgen.StorageKindExternal {
		// The trash copy alone is ours: a live external's source
		// file is the family's, and the derived trees are keyed by
		// the asset id. An orphaned row (trash_path NULL, the
		// rescan's vanish arm) has no file of ours at all.
		var trashPath string
		if row.TrashPath.Valid {
			trashPath = row.TrashPath.String
		}
		if err := storage.RemoveExternalAssetFiles(s.dataDir, trashPath, assetID.String()); err != nil {
			s.logger.Error("remove files of permanently deleted external asset",
				"error", err, "asset_id", assetID, "method", r.Method, "path", r.URL.Path)
		}
		return true, nil
	}
	if err := storage.RemoveAssetFiles(s.dataDir, row.FilePath, assetID.String(), refs); err != nil {
		s.logger.Error("remove files of permanently deleted asset",
			"error", err, "asset_id", assetID, "method", r.Method, "path", r.URL.Path)
	}
	return true, nil
}

// trashItemFrom maps one trashed row onto the API shape. The
// countdown arrives from the query itself (remaining_days), computed
// against the database clock -- whole days rounded up, clamped at
// zero once the deadline passed. storageKind carries the row's
// storage era: the trash view is honest for both kinds, and
// the external items are the ones whose restore moves a file back
// to its recorded source location.
func trashItemFrom(row dbgen.ListTrashRow) apigen.TrashItem {
	return apigen.TrashItem{
		Id:               uuid.UUID(row.ID.Bytes),
		OwnerId:          uuid.UUID(row.OwnerID.Bytes),
		Kind:             apigen.TrashItemKind(row.MediaKind),
		StorageKind:      apigen.TrashItemStorageKind(row.StorageKind),
		OriginalFilename: row.OriginalFilename,
		FileSize:         row.FileSize,
		MimeType:         row.MimeType,
		Checksum:         row.Checksum,
		CreatedAt:        row.CreatedAt.Time.UTC(),
		TakenAt:          row.TakenAt.Time.UTC(),
		TakenAtSource:    apigen.TrashItemTakenAtSource(row.TakenAtSource),
		IsReady:          row.ReadyAt.Valid,
		Width:            dimensionValue(row.Width),
		Height:           dimensionValue(row.Height),
		DurationMs:       int4Value(row.DurationMs),
		Blurhash:         textValue(row.Blurhash),
		LivePhotoVideoId: uuidValue(row.LivePhotoVideoID),
		DeletedAt:        row.DeletedAt.Time.UTC(),
		RemainingDays:    int(row.RemainingDays),
	}
}

// publishAssetEvent reports a trash lifecycle transition onto the
// activity stream through the shared (id, kind) publisher: the
// resource id and its media kind, nothing more -- the event audience
// discipline.
func (s *server) publishAssetEvent(name string, row dbgen.Asset) {
	s.publishAssetEventByID(name, uuid.UUID(row.ID.Bytes), row.MediaKind)
}
