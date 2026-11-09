// The trash handlers (M2c Task 3, the external lifecycle widened by
// E4): the owner-scoped trash listing with its retention countdown,
// the idempotent restore, and the explicit permanent delete. The
// permission model splits along the two delete surfaces: the trash
// (soft delete, restore) is strictly owner-scoped -- a foreign id
// answers the anti-enumeration 404, byte-identical with an unknown
// one -- while the explicit permanent delete also admits
// administrators, the application layer of the spec's two-layer
// delete gate.
//
// The file half splits on the row's storage kind (the §A6/E4
// table): a blob row's file never moves through the lifecycle (the
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

	response := apigen.TrashResponse{Items: make([]apigen.TrashItem, 0, len(rows))}
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
// The external row splits from the blob (E4, the §A6/E4 table's
// restore row): its file was MOVED into the server's trash area
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
	}
	w.WriteHeader(http.StatusNoContent)
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

	// The photo row goes before its linked component video row: the
	// component delete would otherwise trip the foreign key the photo
	// still holds on it.
	deleted, err := s.purgeAssetRow(r, row)
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

	if row.LivePhotoVideoID.Valid {
		component, err := s.queries.GetAssetByID(r.Context(), row.LivePhotoVideoID)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			s.internalError(w, r, "load Live Photo component for delete", err)
			return
		}
		if err == nil {
			// The component is never the client-visible asset, so it
			// purges without its own announcement; its file answers
			// the same reference rule through the same helper.
			if _, err := s.purgeAssetRow(r, component); err != nil {
				s.internalError(w, r, "delete Live Photo component permanently", err)
				return
			}
		}
	}

	s.publishAssetEvent("asset.deleted", row)
	w.WriteHeader(http.StatusNoContent)
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
// The file half splits on the storage kind (E4, the §A6/E4 table's
// expiry row): a blob row's original goes under the shared-file
// rule (the reference count decides -- another row on the same
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
// storage era (E4): the trash view is honest for both kinds, and
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
// discipline of M2a.
func (s *server) publishAssetEvent(name string, row dbgen.Asset) {
	s.publishAssetEventByID(name, uuid.UUID(row.ID.Bytes), row.MediaKind)
}
