// The trash handlers (M2c Task 3): the owner-scoped trash listing
// with its retention countdown, the idempotent restore, and the
// explicit permanent delete. The permission model splits along the
// two delete surfaces: the trash (soft delete, restore) is strictly
// owner-scoped -- a foreign id answers the anti-enumeration 404,
// byte-identical with an unknown one -- while the explicit permanent
// delete also admits administrators, the application layer of the
// spec's two-layer delete gate. The file half of every permanent
// delete is the shared-blob rule: content addressing stores identical
// bytes of any owner under one physical file, so the original is
// removed only when no other row references it (see
// CountAssetFileReferences); the derived copies are keyed by the
// asset id alone and always go.
package api

import (
	"errors"
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

	// The restore write itself: the WHERE guard makes re-restoring a
	// live row (or a restore racing another restore) a zero-row
	// no-op, not an error -- the row is already where the caller
	// wants it.
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
func (s *server) purgeAssetRow(r *http.Request, row dbgen.Asset) (bool, error) {
	ctx := r.Context()
	assetID := uuid.UUID(row.ID.Bytes)

	// The reference count first: it excludes the row itself by id, so
	// it stays meaningful whichever side of the row delete it runs
	// on, and the task's pinned flow -- check, delete the row, then
	// decide the file -- reads in exactly that order.
	refs, err := s.queries.CountAssetFileReferences(ctx, dbgen.CountAssetFileReferencesParams{
		FilePath: row.FilePath,
		ID:       row.ID,
	})
	if err != nil {
		return false, err
	}

	deleted, err := s.queries.DeleteAssetPermanently(ctx, row.ID)
	if err != nil {
		return false, err
	}
	if deleted == 0 {
		return false, nil
	}

	// The row is committed gone; the files follow best-effort. The
	// context is captured while the request is still live.
	if err := storage.RemoveAssetFiles(s.dataDir, row.FilePath, assetID.String(), refs); err != nil {
		s.logger.Error("remove files of permanently deleted asset",
			"error", err, "asset_id", assetID, "method", r.Method, "path", r.URL.Path)
	}
	return true, nil
}

// trashItemFrom maps one trashed row onto the API shape. The
// countdown arrives from the query itself (remaining_days), computed
// against the database clock -- whole days rounded up, clamped at
// zero once the deadline passed.
func trashItemFrom(row dbgen.ListTrashRow) apigen.TrashItem {
	return apigen.TrashItem{
		Id:               uuid.UUID(row.ID.Bytes),
		OwnerId:          uuid.UUID(row.OwnerID.Bytes),
		Kind:             apigen.TrashItemKind(row.MediaKind),
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
// activity stream: the resource id and its media kind, nothing more
// -- the event audience discipline of M2a.
func (s *server) publishAssetEvent(name string, row dbgen.Asset) {
	if s.events == nil {
		return
	}
	s.events.Publish(name, map[string]string{
		"assetId": uuid.UUID(row.ID.Bytes).String(),
		"kind":    string(row.MediaKind),
	})
}
