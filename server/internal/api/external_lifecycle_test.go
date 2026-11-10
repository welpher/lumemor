// The external asset lifecycle tests: the
// external move semantics through the real surfaces -- the
// batch trash arm MOVES the file into data/trash/<assetId>/ (records
// original_path and trash_path, and the failure the operating system
// refuses fails the whole batch explicitly with nothing
// half-moved), the restore moves it back by the record (and a
// vanished original location fails the restore with the row staying
// trashed), expiry physically removes the trash copy and the row,
// and the permanent delete never touches the family's source file.
// The trash listing stays honest for both kinds: the moved external
// lists (with its storage kind), the rescan's orphan marks never
// do.
package api

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"image/color"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"

	apigen "github.com/welpher/lumemor/server/internal/api/gen"
	dbgen "github.com/welpher/lumemor/server/internal/db/gen"
	"github.com/welpher/lumemor/server/internal/events"
	"github.com/welpher/lumemor/server/internal/storage"
	"github.com/welpher/lumemor/server/internal/trash"
)

// externalFixture is one referenced source file plus the asset row
// pointing at it: the directory album's ingest shape, seeded
// directly (the rescan worker's own tests drive the real ingest;
// these tests exercise the lifecycle the rows enter).
type externalFixture struct {
	id    uuid.UUID
	album uuid.UUID
	path  string
	bytes []byte
}

// insertExternalAsset writes one source file under the given mount
// root (a sibling of the data root, never inside it) and inserts
// the external asset row plus its directory album membership. The
// checksum is derived from the content, the snapshot from the
// file's stamped mtime.
func (e *assetEnv) insertExternalAsset(t *testing.T, sourceDir, name string, content []byte, mtime time.Time) externalFixture {
	t.Helper()
	path := filepath.Join(sourceDir, name)
	if err := os.MkdirAll(sourceDir, 0o755); err != nil {
		t.Fatalf("create the source directory: %v", err)
	}
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("write the source file %s: %v", path, err)
	}
	if err := os.Chtimes(path, mtime, mtime); err != nil {
		t.Fatalf("stamp the source mtime: %v", err)
	}

	owner := e.sessionUserID(t)
	id, err := uuid.NewV7()
	if err != nil {
		t.Fatalf("generate the asset id: %v", err)
	}
	albumID, err := uuid.NewV7()
	if err != nil {
		t.Fatalf("generate the album id: %v", err)
	}
	if _, err := e.pool.Exec(context.Background(), `
		INSERT INTO albums (id, owner_id, title, type, root_path)
		VALUES ($1, $2, 'Family archive', 'directory', $3)`,
		pgUUID(albumID), pgUUID(owner), sourceDir); err != nil {
		t.Fatalf("insert the fixture album: %v", err)
	}
	if _, err := e.pool.Exec(context.Background(), `
		INSERT INTO assets (
			id, owner_id, storage_kind, file_path, checksum, original_filename,
			file_size, mime_type, media_kind, width, height, taken_at,
			taken_at_source, pending_jobs, file_mtime
		) VALUES (
			$1, $2, 'external', $3, $4, $5, $6, 'image/jpeg', 'image', 0, 0,
			$7, 'file', 0, $7
		)`,
		pgUUID(id), pgUUID(owner), path, sha256Of(content), name,
		len(content), mtime); err != nil {
		t.Fatalf("insert the external asset row: %v", err)
	}
	if _, err := e.pool.Exec(context.Background(),
		"INSERT INTO album_items (album_id, asset_id, added_by) VALUES ($1, $2, $3)",
		pgUUID(albumID), pgUUID(id), pgUUID(owner)); err != nil {
		t.Fatalf("insert the membership: %v", err)
	}
	return externalFixture{id: id, album: albumID, path: path, bytes: content}
}

// sha256Of answers the lowercase hex digest of the bytes.
func sha256Of(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// trashItems answers the caller's trash listing, failing the test
// on anything but the 200.
func (e *assetEnv) trashItems(t *testing.T) []apigen.TrashItem {
	t.Helper()
	return e.getTrash(t, e.cookie).Items
}

// requireFileBytes asserts the file at path holds exactly want.
func requireFileBytes(t *testing.T, path, want string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if string(data) != want {
		t.Errorf("the bytes at %s = %q, want %q", path, data, want)
	}
}

// requirePathGone asserts nothing exists at path.
func requirePathGone(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Lstat(path); err == nil {
		t.Fatalf("%s still exists, want it gone", path)
	}
}

// externalRow loads the fixture's row.
func (e *assetEnv) externalRow(t *testing.T, id uuid.UUID) dbgen.Asset {
	t.Helper()
	return e.assetRowOf(t, id)
}

// TestExternalTrashAndRestore is the external lifecycle's first two
// rows through the real surfaces: the batch trash arm moves the file
// into the server's trash area (original_path and trash_path
// recorded, file_path naming where the bytes now are), the trash
// listing shows the item honestly with its storage kind, and the
// restore moves the file back to the recorded original location
// with the row live again and the record columns cleared.
func TestExternalTrashAndRestore(t *testing.T) {
	e := newAssetEnv(t, testUploadLimit)
	sourceDir := filepath.Join(t.TempDir(), "family")
	fixture := e.insertExternalAsset(t, sourceDir, "IMG_0001.jpg", []byte("external bytes"), time.Now())

	// The trash arm: the file moves, the row records the move.
	report := e.batchAssets(t, e.cookie, "trash", fixture.id)
	requireBatchReport(t, report, 1)

	row := e.externalRow(t, fixture.id)
	if !row.DeletedAt.Valid {
		t.Fatal("the external row is not trashed after the batch")
	}
	if !row.OriginalPath.Valid || row.OriginalPath.String != fixture.path {
		t.Errorf("original_path = %v, want the real source %s", row.OriginalPath, fixture.path)
	}
	if !row.TrashPath.Valid {
		t.Fatal("trash_path is NULL, want the recorded trash location")
	}
	if want := filepath.Join(storage.TrashHome(e.dataDir, fixture.id.String()), "IMG_0001.jpg"); row.TrashPath.String != want {
		t.Errorf("trash_path = %q, want %q", row.TrashPath.String, want)
	}
	if row.FilePath != row.TrashPath.String {
		t.Errorf("file_path = %q, want the row naming where the bytes now are (%q)", row.FilePath, row.TrashPath.String)
	}
	requirePathGone(t, fixture.path)
	requireFileBytes(t, row.TrashPath.String, "external bytes")

	// The trash listing: honest for the external kind.
	items := e.trashItems(t)
	if len(items) != 1 {
		t.Fatalf("trash items = %d, want the trashed external", len(items))
	}
	if uuid.UUID(items[0].Id) != fixture.id {
		t.Errorf("the listed id = %s, want %s", items[0].Id, fixture.id)
	}
	if items[0].StorageKind != apigen.External {
		t.Errorf("storageKind = %q, want external", items[0].StorageKind)
	}

	// The restore: the file moves back by the record, the row goes
	// live, the record columns clear.
	if rec := e.restoreAsset(t, e.cookie, fixture.id); rec.Code != http.StatusNoContent {
		t.Fatalf("POST /trash/{id}/restore status = %d, want %d (body %q)", rec.Code, http.StatusNoContent, rec.Body.String())
	}
	restored := e.externalRow(t, fixture.id)
	if restored.DeletedAt.Valid {
		t.Fatal("the external row is still trashed after the restore")
	}
	if restored.OriginalPath.Valid || restored.TrashPath.Valid {
		t.Errorf("the record columns = (%v, %v), want both cleared", restored.OriginalPath, restored.TrashPath)
	}
	if restored.FilePath != fixture.path {
		t.Errorf("file_path after the restore = %q, want the original location %q", restored.FilePath, fixture.path)
	}
	requirePathGone(t, row.TrashPath.String)
	requireFileBytes(t, fixture.path, string(fixture.bytes))
}

// TestExternalTrashMoveFailure pins the explicit-failure arm and the
// batch's compensation: an external file the move cannot complete
// fails the WHOLE batch with the move's own error code, the rows
// stay live, and a file the batch already moved is compensated back
// to its source path -- nothing half-moves. The refusal is
// triggered by a source file that vanished between the batch's
// resolution and its move (the state change mid-flight every
// permission refusal shares the code path with), and by the
// operating system's own refusal where the suite runs unprivileged.
func TestExternalTrashMoveFailure(t *testing.T) {
	t.Run("VanishedSourceFile", func(t *testing.T) {
		e := newAssetEnv(t, testUploadLimit)
		movable := e.insertExternalAsset(t, filepath.Join(t.TempDir(), "writable"), "IMG_0002.jpg", []byte("movable bytes"), time.Now())
		vanished := e.insertExternalAsset(t, filepath.Join(t.TempDir(), "gone"), "IMG_0003.jpg", []byte("vanished bytes"), time.Now())

		// The second id's file is gone before the batch runs: the
		// resolution still sees the row (it is live and owned), the
		// move finds nothing to move. The movable id comes FIRST, so
		// its successful move is exactly what the failure path must
		// compensate.
		if err := os.Remove(vanished.path); err != nil {
			t.Fatalf("remove the vanished fixture's file: %v", err)
		}

		rec := e.postBatch(t, e.cookie, batchBody(t, "trash", []uuid.UUID{movable.id, vanished.id}))
		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("POST /assets/batch status = %d, want %d (body %q)", rec.Code, http.StatusInternalServerError, rec.Body.String())
		}
		apiErr, _ := decodeError(t, rec)
		if apiErr.Code != apigen.ASSETTRASHMOVEFAILED {
			t.Errorf("code = %q, want %q", apiErr.Code, apigen.ASSETTRASHMOVEFAILED)
		}

		// Nothing half-moved: both rows stay live, the movable file
		// is back at its source path, the trash area holds nothing
		// of theirs.
		for _, fixture := range []externalFixture{movable, vanished} {
			if row := e.externalRow(t, fixture.id); row.DeletedAt.Valid {
				t.Errorf("the row of %s is trashed after the failed batch, want it live", fixture.id)
			}
		}
		requireFileBytes(t, movable.path, string(movable.bytes))
		entries, err := os.ReadDir(filepath.Join(e.dataDir, "trash"))
		if err != nil {
			t.Fatalf("read the trash area: %v", err)
		}
		if len(entries) != 0 {
			t.Errorf("the trash area holds %d entries after the failed batch, want the compensation to leave none", len(entries))
		}
	})

	t.Run("PermissionRefused", func(t *testing.T) {
		if os.Geteuid() == 0 {
			t.Skip("the permission arm is untestable as root: the kernel lets root write anything")
		}
		e := newAssetEnv(t, testUploadLimit)
		// The refused fixture's directory is made read-only for the
		// move: neither the rename out of it nor the fallback's
		// remove can run.
		refusedDir := filepath.Join(t.TempDir(), "readonly")
		refused := e.insertExternalAsset(t, refusedDir, "IMG_0004.jpg", []byte("refused bytes"), time.Now())
		if err := os.Chmod(refusedDir, 0o500); err != nil {
			t.Fatalf("chmod the refused source directory: %v", err)
		}
		t.Cleanup(func() { _ = os.Chmod(refusedDir, 0o700) })

		rec := e.postBatch(t, e.cookie, batchBody(t, "trash", []uuid.UUID{refused.id}))
		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("POST /assets/batch status = %d, want %d (body %q)", rec.Code, http.StatusInternalServerError, rec.Body.String())
		}
		apiErr, _ := decodeError(t, rec)
		if apiErr.Code != apigen.ASSETTRASHMOVEFAILED {
			t.Errorf("code = %q, want %q", apiErr.Code, apigen.ASSETTRASHMOVEFAILED)
		}
		if row := e.externalRow(t, refused.id); row.DeletedAt.Valid {
			t.Errorf("the row of %s is trashed after the refused move, want it live", refused.id)
		}
		requireFileBytes(t, refused.path, string(refused.bytes))
	})
}

// TestExternalRestoreFailure pins the restore's honest refusal: an
// original location that no longer exists fails the restore with
// the row staying trashed and the file staying in the trash area
// exactly where a retry after the source returns finds it.
func TestExternalRestoreFailure(t *testing.T) {
	e := newAssetEnv(t, testUploadLimit)
	sourceDir := filepath.Join(t.TempDir(), "family")
	fixture := e.insertExternalAsset(t, sourceDir, "IMG_0004.jpg", []byte("bytes to keep"), time.Now())

	report := e.batchAssets(t, e.cookie, "trash", fixture.id)
	requireBatchReport(t, report, 1)
	trashed := e.externalRow(t, fixture.id)

	// The original location vanishes entirely.
	if err := os.RemoveAll(sourceDir); err != nil {
		t.Fatalf("remove the source directory: %v", err)
	}

	rec := e.restoreAsset(t, e.cookie, fixture.id)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("POST /trash/{id}/restore status = %d, want %d (body %q)", rec.Code, http.StatusInternalServerError, rec.Body.String())
	}
	apiErr, _ := decodeError(t, rec)
	if apiErr.Code != apigen.ASSETRESTOREMOVEFAILED {
		t.Errorf("code = %q, want %q", apiErr.Code, apigen.ASSETRESTOREMOVEFAILED)
	}

	// The row stays trashed; the file stays in the trash area for
	// the retry.
	if row := e.externalRow(t, fixture.id); !row.DeletedAt.Valid {
		t.Error("the row went live after the failed restore, want it staying trashed")
	}
	requireFileBytes(t, trashed.TrashPath.String, string(fixture.bytes))
}

// TestExternalExpiry is the external expiry row: the run
// purges the row and physically removes the trash copy; the
// family's source location (wherever the file no longer sits) is
// never touched. The orphan's row (the rescan's vanish arm, no
// trash file) purges alone.
func TestExternalExpiry(t *testing.T) {
	e := newAssetEnv(t, testUploadLimit)
	sourceDir := filepath.Join(t.TempDir(), "family")
	moved := e.insertExternalAsset(t, sourceDir, "IMG_0005.jpg", []byte("expired bytes"), time.Now())
	orphan := e.insertExternalAsset(t, sourceDir, "IMG_0006.jpg", []byte("orphan bytes"), time.Now())

	report := e.batchAssets(t, e.cookie, "trash", moved.id)
	requireBatchReport(t, report, 1)
	trashed := e.externalRow(t, moved.id)
	trashFilePath := trashed.TrashPath.String

	// The orphan's mark: the rescan's vanish arm shape -- deleted_at
	// with no trash file of ours (its source file is gone by
	// definition, so the fixture removes it).
	if err := os.Remove(orphan.path); err != nil {
		t.Fatalf("remove the orphan's source file: %v", err)
	}
	e.exec(t, "UPDATE assets SET deleted_at = now() - make_interval(days => 40) WHERE id = $1", pgUUID(orphan.id))
	e.backdateTrash(t, moved.id, 40)

	// A source file that RETURNED at the orphan's old path: the
	// expiry must never remove it (the next scan ingests it as a NEW
	// row, the pin).
	returned := "the returned bytes"
	if err := os.WriteFile(orphan.path, []byte(returned), 0o644); err != nil {
		t.Fatalf("write the returned source file: %v", err)
	}

	counts, err := trash.RunExpiry(context.Background(), e.pool, e.dataDir, e.bus)
	if err != nil {
		t.Fatalf("RunExpiry: %v", err)
	}
	if counts.Expired != 2 {
		t.Errorf("expired = %d, want both the moved row and the orphan", counts.Expired)
	}

	// The moved row: gone, the trash copy gone with it.
	if _, err := e.queries.GetAssetByID(context.Background(), pgUUID(moved.id)); err == nil {
		t.Error("the moved external row survived the expiry, want it purged")
	}
	requirePathGone(t, trashFilePath)
	requirePathGone(t, storage.TrashHome(e.dataDir, moved.id.String()))

	// The orphan's row: gone, the returned source file untouched.
	if _, err := e.queries.GetAssetByID(context.Background(), pgUUID(orphan.id)); err == nil {
		t.Error("the orphaned external row survived the expiry, want it purged")
	}
	requireFileBytes(t, orphan.path, returned)
}

// TestExternalPermanentDelete pins the explicit delete's file
// story: the row and the derived trees go, the family's source file
// is never touched while the asset is live -- and the trash copy
// goes once the row sits in the trash area.
func TestExternalPermanentDelete(t *testing.T) {
	e := newAssetEnv(t, testUploadLimit)
	sourceDir := filepath.Join(t.TempDir(), "family")
	live := e.insertExternalAsset(t, sourceDir, "IMG_0007.jpg", []byte("live bytes"), time.Now())
	trashed := e.insertExternalAsset(t, sourceDir, "IMG_0008.jpg", []byte("trashed bytes"), time.Now())

	// The live external: the row goes, the source file survives.
	if rec := e.deleteAsset(t, e.cookie, live.id); rec.Code != http.StatusNoContent {
		t.Fatalf("DELETE /assets/{id} status = %d, want %d (body %q)", rec.Code, http.StatusNoContent, rec.Body.String())
	}
	if _, err := e.queries.GetAssetByID(context.Background(), pgUUID(live.id)); err == nil {
		t.Error("the live external row survived the permanent delete, want it gone")
	}
	requireFileBytes(t, live.path, string(live.bytes))

	// The trashed external: the row goes and the trash copy goes
	// with it.
	report := e.batchAssets(t, e.cookie, "trash", trashed.id)
	requireBatchReport(t, report, 1)
	trashRow := e.externalRow(t, trashed.id)
	if rec := e.deleteAsset(t, e.cookie, trashed.id); rec.Code != http.StatusNoContent {
		t.Fatalf("DELETE /assets/{id} status = %d, want %d (body %q)", rec.Code, http.StatusNoContent, rec.Body.String())
	}
	if _, err := e.queries.GetAssetByID(context.Background(), pgUUID(trashed.id)); err == nil {
		t.Error("the trashed external row survived the permanent delete, want it gone")
	}
	requirePathGone(t, trashRow.TrashPath.String)
}

// TestTrashListingExcludesOrphans pins the listing's honest shape:
// an external row the rescan's vanish arm marked (deleted_at, no
// trash file of ours) never appears -- the trash has no meaning for
// a file that was never the server's -- while a blob row trashed in
// place and a user-trashed external both list.
func TestTrashListingExcludesOrphans(t *testing.T) {
	e := newAssetEnv(t, testUploadLimit)
	sourceDir := filepath.Join(t.TempDir(), "family")

	orphan := e.insertExternalAsset(t, sourceDir, "IMG_0009.jpg", []byte("orphan bytes"), time.Now())
	if err := os.Remove(orphan.path); err != nil {
		t.Fatalf("remove the orphan's source file: %v", err)
	}
	e.exec(t, "UPDATE assets SET deleted_at = now() WHERE id = $1", pgUUID(orphan.id))

	userTrashed := e.insertExternalAsset(t, sourceDir, "IMG_0010.jpg", []byte("trashed bytes"), time.Now())
	report := e.batchAssets(t, e.cookie, "trash", userTrashed.id)
	requireBatchReport(t, report, 1)

	blob := e.uploadPhotoAs(t, e.cookie, "blob-neighbor", time.Now(), jpegColorFixture(t, color.RGBA{R: 0x12, G: 0x34, B: 0x56, A: 0xff}))
	report = e.batchAssets(t, e.cookie, "trash", blob.Id)
	requireBatchReport(t, report, 1)

	items := e.trashItems(t)
	ids := make(map[uuid.UUID]apigen.TrashItemStorageKind, len(items))
	for _, item := range items {
		ids[uuid.UUID(item.Id)] = item.StorageKind
	}
	if _, listed := ids[orphan.id]; listed {
		t.Error("the trash listing shows the orphaned external, want it never listed")
	}
	if kind, listed := ids[userTrashed.id]; !listed || kind != apigen.External {
		t.Errorf("the user-trashed external = (%v, %q), want it listed as external", listed, kind)
	}
	if kind, listed := ids[blob.Id]; !listed || kind != apigen.Blob {
		t.Errorf("the trashed blob = (%v, %q), want it listed as blob", listed, kind)
	}
}

// TestAlbumMembershipEvents pins the album walls' convergence signal
// of the deletability chain: the trash and the restore of
// an album member change the album's LIVE membership -- the derived
// count the detail renders and the members the wall walks -- and the
// album-narrowed wall converges exactly through album.items-changed,
// the rescan's own restart signal, published here for the curation
// arms that move the live membership without touching its rows (the
// membership row survives the trash; the live views' deleted_at
// filter is the change). Both restore surfaces carry the signal (the
// batch arm and the trash view's single restore), the idempotent
// repeat stays silent (the event discipline: only a state change
// announces itself), and the arms that never touch a membership
// publish nothing album-level.
func TestAlbumMembershipEvents(t *testing.T) {
	t.Run("BatchTrashAndRestorePublish", func(t *testing.T) {
		e := newAssetEnv(t, testUploadLimit)
		fixture := e.insertExternalAsset(t, filepath.Join(t.TempDir(), "event-family"), "IMG_0012.jpg", []byte("event bytes"), time.Now())

		subCtx, subCancel := context.WithTimeout(t.Context(), 10*time.Second)
		defer subCancel()
		subscription := e.bus.Subscribe(subCtx, 0)
		owner := e.sessionUserID(t)

		report := e.batchAssets(t, e.cookie, "trash", fixture.id)
		requireBatchReport(t, report, 1)
		requireAlbumItemsChanged(t, subscription, fixture.album, owner)

		// The restore answers the same signal: the live membership
		// returns, and the album wall's walk restarts on it.
		report = e.batchAssets(t, e.cookie, "restore", fixture.id)
		requireBatchReport(t, report, 1)
		requireAlbumItemsChanged(t, subscription, fixture.album, owner)

		// The idempotent repeats never announce: neither the
		// already-trashed no-op nor the already-live restore writes a
		// row, so no album event follows either.
		e.bus.Publish("test.marker", nil)
		report = e.batchAssets(t, e.cookie, "trash", fixture.id)
		requireBatchReport(t, report, 1)
		report = e.batchAssets(t, e.cookie, "restore", fixture.id)
		requireBatchReport(t, report, 1)
		requireNextEventName(t, subscription, "test.marker")
	})

	t.Run("SingleRestorePublishes", func(t *testing.T) {
		e := newAssetEnv(t, testUploadLimit)
		fixture := e.insertExternalAsset(t, filepath.Join(t.TempDir(), "single-family"), "IMG_0013.jpg", []byte("single bytes"), time.Now())
		e.batchAssets(t, e.cookie, "trash", fixture.id)

		subCtx, subCancel := context.WithTimeout(t.Context(), 10*time.Second)
		defer subCancel()
		subscription := e.bus.Subscribe(subCtx, 0)
		owner := e.sessionUserID(t)

		rec := e.restoreAsset(t, e.cookie, fixture.id)
		if rec.Code != http.StatusNoContent {
			t.Fatalf("POST /trash/{id}/restore status = %d, want %d (body %q)",
				rec.Code, http.StatusNoContent, rec.Body.String())
		}
		requireAlbumItemsChanged(t, subscription, fixture.album, owner)
	})
}

// requireAlbumItemsChanged consumes bus events until one
// album.items-changed names the given album (the owner's payload),
// failing the test on the subscription window's end. Events the
// publication legitimately interleaves (the arms' own asset events)
// are skipped -- the assertion is about the album event, not the
// ordering.
func requireAlbumItemsChanged(t *testing.T, subscription *events.Subscription, album, owner uuid.UUID) {
	t.Helper()
	for {
		event, ok := subscription.Next()
		if !ok {
			t.Fatalf("no album.items-changed for %s reached the bus within the subscription window", album)
		}
		if event.Name != "album.items-changed" {
			continue
		}
		got := eventPayload(t, event)
		if got["albumId"] != album.String() {
			t.Errorf("event albumId = %q, want %q", got["albumId"], album)
		}
		if got["userId"] != owner.String() {
			t.Errorf("event userId = %q, want the owner %q", got["userId"], owner)
		}
		return
	}
}

// requireNextEventName consumes bus events until one carries the
// given name, failing on anything else first: the marker pattern for
// asserting that NOTHING was published in between.
func requireNextEventName(t *testing.T, subscription *events.Subscription, name string) {
	t.Helper()
	for {
		event, ok := subscription.Next()
		if !ok {
			t.Fatalf("no %s event reached the bus within the subscription window", name)
		}
		if event.Name == name {
			return
		}
		t.Errorf("an event was published between the marker and its read: %s", event.Name)
	}
}
