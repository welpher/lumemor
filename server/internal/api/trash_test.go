// Trash lifecycle tests (M2c Task 3): the owner-scoped listing with
// its retention countdown, the idempotent restore, the explicit
// permanent delete with its permission split (owner or
// administrator), and -- the mandated Review Focus cases -- the
// cross-owner shared blob whose file survives one owner's expiry
// because another row still references it, and the restore-vs-expiry
// race whose guard keeps a restored row alive. Soft deletes go
// through the SoftDeleteAsset query (the endpoint of that operation
// is Task 4's batch handler; the query is this task's deliverable),
// expiry through the real trash.RunExpiry.
package api

import (
	"context"
	"encoding/json"
	"fmt"
	"image/color"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	apigen "github.com/welpher/lumemor/server/internal/api/gen"
	dbgen "github.com/welpher/lumemor/server/internal/db/gen"
	"github.com/welpher/lumemor/server/internal/events"
	"github.com/welpher/lumemor/server/internal/trash"
)

// uploadPhotoAs fires one single-photo upload with the given session
// cookie and explicit content (distinct pixel colors give distinct
// checksums, so the shared-blob fixtures can upload the SAME bytes
// twice and the sole-reference fixtures unique bytes).
func (e *assetEnv) uploadPhotoAs(t *testing.T, cookie *http.Cookie, deviceAssetID string, taken time.Time, content []byte) apigen.AssetResponse {
	t.Helper()
	body, contentType := buildUploadBody(t, map[string]string{
		"deviceAssetId":   deviceAssetID,
		"fileName":        deviceAssetID + ".jpg",
		"deviceCreatedAt": taken.Format(time.RFC3339),
	}, []assetUploadFile{{field: uploadPartPhoto, name: deviceAssetID + ".jpg", content: content}})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/assets", body)
	req.Header.Set("Content-Type", contentType)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	e.router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("upload %s status = %d, want %d (body %q)", deviceAssetID, rec.Code, http.StatusCreated, rec.Body.String())
	}
	return decodeAsset(t, rec)
}

// softDeleteAsset trashes the asset through the query the batch
// handler of Task 4 will call: the query (with its owner guard and
// its set-semantics arm) is this task's deliverable, so the tests
// exercise it directly instead of hand-writing UPDATE statements.
// It returns the rows the write marked.
func (e *assetEnv) softDeleteAsset(t *testing.T, id, owner uuid.UUID) int64 {
	t.Helper()
	rows, err := e.queries.SoftDeleteAsset(context.Background(), dbgen.SoftDeleteAssetParams{
		ID:      pgUUID(id),
		OwnerID: pgUUID(owner),
	})
	if err != nil {
		t.Fatalf("soft delete asset %s: %v", id, err)
	}
	return rows
}

// backdateTrash moves the asset's deletion mark into the past, past
// the retention window the test wants expired. Expiry is a deadline
// computation, never a sleep.
func (e *assetEnv) backdateTrash(t *testing.T, id uuid.UUID, days int) {
	t.Helper()
	e.exec(t, `UPDATE assets SET deleted_at = now() - make_interval(days => $2) WHERE id = $1`,
		pgUUID(id), days)
}

// assetRowOf loads the full row of an asset by id, whatever its
// state and owner (the tests own their fixtures).
func (e *assetEnv) assetRowOf(t *testing.T, id uuid.UUID) dbgen.Asset {
	t.Helper()
	row, err := e.queries.GetAssetByID(context.Background(), pgUUID(id))
	if err != nil {
		t.Fatalf("load asset row %s: %v", id, err)
	}
	return row
}

// storedFileOf resolves the on-disk path of an asset's original.
func (e *assetEnv) storedFileOf(t *testing.T, row dbgen.Asset) string {
	t.Helper()
	return filepath.Join(e.dataDir, filepath.FromSlash(row.FilePath))
}

// shareBlobBetween rewires two owners' rows of the same bytes onto ONE
// stored file: the shared-blob state of assets ingested before the
// storage-template feature, whose content-addressed paths let two
// owners' identical bytes share one physical file (new uploads render
// per-owner paths instead). The per-owner files the template flow
// created are removed, one shared file is materialized at the legacy
// content-addressed shape, and both rows point at it -- the exact
// state the delete paths' reference counting still serves.
func (e *assetEnv) shareBlobBetween(t *testing.T, first, second dbgen.Asset, content []byte) string {
	t.Helper()
	checksum := sha256Hex(content)
	sharedRel := "originals/" + checksum[:2] + "/" + checksum[2:4] + "/" + checksum + ".jpg"
	shared := filepath.Join(e.dataDir, filepath.FromSlash(sharedRel))
	if err := os.MkdirAll(filepath.Dir(shared), 0o755); err != nil {
		t.Fatalf("create shared blob directory: %v", err)
	}
	if err := os.WriteFile(shared, content, 0o644); err != nil {
		t.Fatalf("write shared blob: %v", err)
	}
	for _, row := range []dbgen.Asset{first, second} {
		if err := os.Remove(e.storedFileOf(t, row)); err != nil {
			t.Fatalf("remove the per-owner file of %s: %v", uuid.UUID(row.ID.Bytes), err)
		}
	}
	e.exec(t, `UPDATE assets SET file_path = $2 WHERE id = $1 OR id = $3`,
		pgUUID(uuid.UUID(first.ID.Bytes)), sharedRel, pgUUID(uuid.UUID(second.ID.Bytes)))
	return shared
}

// requireFileState asserts the existence of a file path.
func requireFileState(t *testing.T, path string, want bool) {
	t.Helper()
	_, err := os.Stat(path)
	if want && err != nil {
		t.Fatalf("file %s does not exist, want it present: %v", path, err)
	}
	if !want && err == nil {
		t.Fatalf("file %s exists, want it gone", path)
	}
}

// writeThumbFixture materializes a derived thumbs directory for an
// asset: the upload-only test environment runs no pipeline, so the
// fixture stands in for the thumbnail job's output directory (which
// the permanent deletes must remove).
func (e *assetEnv) writeThumbFixture(t *testing.T, assetID uuid.UUID) string {
	t.Helper()
	dir := filepath.Join(e.dataDir, "storage", "thumbs", assetID.String())
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("create thumbs fixture dir %s: %v", dir, err)
	}
	file := filepath.Join(dir, "tile.avif")
	if err := os.WriteFile(file, []byte("fixture thumb"), 0o644); err != nil {
		t.Fatalf("write thumbs fixture %s: %v", file, err)
	}
	return dir
}

// setRetention writes the instance's trash retention setting. An
// empty value removes the row (the default applies).
func (e *assetEnv) setRetention(t *testing.T, value string) {
	t.Helper()
	if value == "" {
		e.exec(t, "DELETE FROM instance_settings WHERE key = 'trash_retention_days'")
		return
	}
	e.exec(t, `INSERT INTO instance_settings (key, value) VALUES ('trash_retention_days', $1::jsonb)
		ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, updated_at = now()`, value)
}

// getTrash fires the trash listing and decodes the 200 response.
func (e *assetEnv) getTrash(t *testing.T, cookie *http.Cookie) apigen.TrashResponse {
	t.Helper()
	rec := e.getTrashRaw(t, cookie)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /trash status = %d, want %d (body %q)", rec.Code, http.StatusOK, rec.Body.String())
	}
	var response apigen.TrashResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode trash body %q: %v", rec.Body.String(), err)
	}
	return response
}

// getTrashRaw fires the trash listing and returns the recorder.
func (e *assetEnv) getTrashRaw(t *testing.T, cookie *http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/trash", nil)
	if cookie != nil {
		req.AddCookie(cookie)
	}
	rec := httptest.NewRecorder()
	e.router.ServeHTTP(rec, req)
	return rec
}

// restoreAsset fires the restore request and returns the recorder.
func (e *assetEnv) restoreAsset(t *testing.T, cookie *http.Cookie, id uuid.UUID) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/trash/"+id.String()+"/restore", nil)
	if cookie != nil {
		req.AddCookie(cookie)
	}
	rec := httptest.NewRecorder()
	e.router.ServeHTTP(rec, req)
	return rec
}

// deleteAsset fires the permanent-delete request and returns the
// recorder.
func (e *assetEnv) deleteAsset(t *testing.T, cookie *http.Cookie, id uuid.UUID) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/assets/"+id.String(), nil)
	if cookie != nil {
		req.AddCookie(cookie)
	}
	rec := httptest.NewRecorder()
	e.router.ServeHTTP(rec, req)
	return rec
}

// runExpiry runs the real expiry implementation against the
// environment's pool and data root, on the environment's bus.
func (e *assetEnv) runExpiry(t *testing.T) trash.ExpiryCounts {
	t.Helper()
	counts, err := trash.RunExpiry(context.Background(), e.pool, e.dataDir, e.bus)
	if err != nil {
		t.Fatalf("run trash expiry: %v", err)
	}
	return counts
}

// trashItemIDs collects the ids of the trash listing in order.
func trashItemIDs(items []apigen.TrashItem) []uuid.UUID {
	ids := make([]uuid.UUID, len(items))
	for i, item := range items {
		ids[i] = uuid.UUID(item.Id)
	}
	return ids
}

// busEventsOf collects the payloads of one event name off the bus's
// retained ring.
func busEventsOf(t *testing.T, bus *events.Bus, name string) []map[string]string {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), assetEventWait)
	defer cancel()
	subscription := bus.Subscribe(ctx, 0)
	var payloads []map[string]string
	for {
		event, ok := subscription.Next()
		if !ok {
			return payloads
		}
		if event.Name != name {
			continue
		}
		var payload map[string]string
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			t.Fatalf("decode %s payload %s: %v", name, event.Payload, err)
		}
		payloads = append(payloads, payload)
	}
}

// TestTrashSharedBlobExpiry is the mandated Review Focus 1 case: two
// owners hold rows over the SAME physical file -- the shared-blob
// state of assets ingested before the storage-template feature, whose
// content-addressed paths let identical bytes of any owner share one
// file. The owner who trashes and expires loses the row; the FILE
// stays (the other owner's row references it), the other owner's
// timeline keeps the photo, and the other owner's derived files are
// untouched. The deleting owner's own derived directory goes with the
// row.
func TestTrashSharedBlobExpiry(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	adminID := env.sessionUserID(t)
	_, graceCookie := env.createSecondUser(t, "trash-shared-grace@example.com")

	taken := time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC)
	shared := jpegColorFixture(t, color.RGBA{R: 0x11, G: 0x22, B: 0x33, A: 0xff})
	adminAsset := env.uploadPhotoAs(t, env.cookie, "trash-shared-admin", taken, shared)
	graceAsset := env.uploadPhotoAs(t, graceCookie, "trash-shared-grace", taken, shared)

	adminRow := env.assetRowOf(t, uuid.UUID(adminAsset.Id))
	graceRow := env.assetRowOf(t, uuid.UUID(graceAsset.Id))
	// The two owners' rows land over one shared legacy file.
	sharedFile := env.shareBlobBetween(t, adminRow, graceRow, shared)
	adminRow = env.assetRowOf(t, uuid.UUID(adminAsset.Id))
	graceRow = env.assetRowOf(t, uuid.UUID(graceAsset.Id))
	if adminRow.FilePath != graceRow.FilePath {
		t.Fatalf("the two rewired rows store %q and %q, want the one shared file path",
			adminRow.FilePath, graceRow.FilePath)
	}
	requireFileState(t, sharedFile, true)
	adminThumbs := env.writeThumbFixture(t, uuid.UUID(adminAsset.Id))
	graceThumbs := env.writeThumbFixture(t, uuid.UUID(graceAsset.Id))

	// Soft delete one owner's row and read both trash views.
	if rows := env.softDeleteAsset(t, uuid.UUID(adminAsset.Id), adminID); rows != 1 {
		t.Fatalf("soft delete marked %d rows, want 1", rows)
	}

	adminTrash := env.getTrash(t, env.cookie)
	if ids := trashItemIDs(adminTrash.Items); len(ids) != 1 || ids[0] != uuid.UUID(adminAsset.Id) {
		t.Fatalf("admin's trash holds %v, want exactly the shared photo %s", ids, adminAsset.Id)
	}
	item := adminTrash.Items[0]
	if item.RemainingDays != 30 {
		t.Errorf("remainingDays = %d, want the default retention's 30", item.RemainingDays)
	}
	if item.Kind != apigen.TrashItemKindImage || item.TakenAtSource != apigen.TrashItemTakenAtSourceImport {
		t.Errorf("kind/takenAtSource = %q/%q, want image/import", item.Kind, item.TakenAtSource)
	}
	if item.IsReady || item.Blurhash != nil || item.Width != nil {
		t.Errorf("pipeline fields = isReady %v blurhash %v width %v, want the not-ready shape", item.IsReady, item.Blurhash, item.Width)
	}
	if item.Checksum != adminAsset.Checksum || item.OriginalFilename != adminAsset.OriginalFilename ||
		item.FileSize != adminAsset.FileSize || item.MimeType != adminAsset.MimeType {
		t.Errorf("identity fields diverge from the upload response: %+v", item)
	}
	// The deletion mark is the server's now; a window assertion (the
	// clock of the database container and the test host are not the
	// same clock), truncated to the database's microsecond precision.
	before := time.Now().Add(-5 * time.Second)
	after := time.Now().Add(5 * time.Second)
	if mark := item.DeletedAt.Truncate(time.Microsecond); mark.Before(before) || mark.After(after) {
		t.Errorf("deletedAt = %v, want a fresh deletion mark near now", item.DeletedAt)
	}

	if graceTrash := env.getTrash(t, graceCookie); len(graceTrash.Items) != 0 {
		t.Fatalf("grace's trash holds %v, want empty", trashItemIDs(graceTrash.Items))
	}

	// The trashed photo leaves the owner's timeline; the other
	// owner's stays.
	adminTimeline := env.getTimeline(t, env.cookie, "")
	for _, id := range timelineItemIDs(adminTimeline.Items) {
		if id == uuid.UUID(adminAsset.Id) {
			t.Error("the trashed asset still appears in the owner's timeline")
		}
	}
	graceTimeline := env.getTimeline(t, graceCookie, "")
	if ids := timelineItemIDs(graceTimeline.Items); len(ids) != 1 || ids[0] != uuid.UUID(graceAsset.Id) {
		t.Fatalf("grace's timeline holds %v, want her copy of the shared photo (the file is hers to keep)", ids)
	}

	// Expiry: backdate past the default 30 days and run the job.
	env.backdateTrash(t, uuid.UUID(adminAsset.Id), 31)
	counts := env.runExpiry(t)
	if counts.Expired != 1 || counts.FilesRemoved != 0 {
		t.Fatalf("expiry counts = expired %d files %d, want expired 1 files 0 (the file is shared)", counts.Expired, counts.FilesRemoved)
	}

	// The row is gone; the FILE survives -- the load-bearing pin.
	env.requireNoAssetRow(t, uuid.UUID(adminAsset.Id))
	requireFileState(t, sharedFile, true)
	if graceTrash := env.getTrash(t, graceCookie); len(graceTrash.Items) != 0 {
		t.Errorf("grace's trash changed to %v, want empty", trashItemIDs(graceTrash.Items))
	}
	graceTimeline = env.getTimeline(t, graceCookie, "")
	if ids := timelineItemIDs(graceTimeline.Items); len(ids) != 1 || ids[0] != uuid.UUID(graceAsset.Id) {
		t.Errorf("grace's timeline after the expiry holds %v, want her photo still there", ids)
	}

	// The deleting owner's derived directory went; the other owner's
	// is untouched (they are keyed by the asset id, never shared).
	requireFileState(t, adminThumbs, false)
	requireFileState(t, graceThumbs, true)

	// The purge announced itself exactly once.
	deleted := busEventsOf(t, env.bus, "asset.deleted")
	if len(deleted) != 1 || deleted[0]["assetId"] != adminAsset.Id.String() {
		t.Errorf("asset.deleted events = %v, want exactly one for the expired asset", deleted)
	}

	// The row's absence also ends its restore: no resurrection.
	if rec := env.restoreAsset(t, env.cookie, uuid.UUID(adminAsset.Id)); rec.Code != http.StatusNotFound {
		t.Errorf("restore of the purged asset status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

// TestTrashSoleReferenceExpiry is the mandated sole-reference case:
// the only row of a unique photo expires, and row, original file and
// derived directory all go together.
func TestTrashSoleReferenceExpiry(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	adminID := env.sessionUserID(t)

	taken := time.Date(2026, 6, 2, 10, 0, 0, 0, time.UTC)
	unique := jpegColorFixture(t, color.RGBA{R: 0x44, G: 0x55, B: 0x66, A: 0xff})
	asset := env.uploadPhotoAs(t, env.cookie, "trash-sole", taken, unique)

	row := env.assetRowOf(t, uuid.UUID(asset.Id))
	file := env.storedFileOf(t, row)
	thumbs := env.writeThumbFixture(t, uuid.UUID(asset.Id))
	requireFileState(t, file, true)

	if rows := env.softDeleteAsset(t, uuid.UUID(asset.Id), adminID); rows != 1 {
		t.Fatalf("soft delete marked %d rows, want 1", rows)
	}
	env.backdateTrash(t, uuid.UUID(asset.Id), 31)

	counts := env.runExpiry(t)
	if counts.Expired != 1 || counts.FilesRemoved != 1 {
		t.Fatalf("expiry counts = expired %d files %d, want 1 and 1 (sole reference)", counts.Expired, counts.FilesRemoved)
	}

	env.requireNoAssetRow(t, uuid.UUID(asset.Id))
	requireFileState(t, file, false)
	requireFileState(t, thumbs, false)
	if trashItems := env.getTrash(t, env.cookie).Items; len(trashItems) != 0 {
		t.Errorf("trash after the expiry holds %v, want empty", trashItemIDs(trashItems))
	}

	// Restore of the purged row: 404, the row is gone for good.
	if rec := env.restoreAsset(t, env.cookie, uuid.UUID(asset.Id)); rec.Code != http.StatusNotFound {
		t.Errorf("restore after purge status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

// TestTrashRestoreBeforeExpirySurvives is the deterministic half of
// the mandated restore-vs-expiry race (Review Focus 4): the row is
// backdated past the retention deadline but RESTORED before the run
// -- the expiry's guarded delete must skip it, and the row and its
// file survive.
func TestTrashRestoreBeforeExpirySurvives(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	adminID := env.sessionUserID(t)

	taken := time.Date(2026, 6, 3, 10, 0, 0, 0, time.UTC)
	asset := env.uploadPhotoAs(t, env.cookie, "trash-race", taken, jpegColorFixture(t, color.RGBA{R: 0x77, G: 0x88, B: 0x99, A: 0xff}))
	assetID := uuid.UUID(asset.Id)

	row := env.assetRowOf(t, assetID)
	file := env.storedFileOf(t, row)

	env.softDeleteAsset(t, assetID, adminID)
	env.backdateTrash(t, assetID, 31)

	// The restore lands between the backdating and the run.
	if rec := env.restoreAsset(t, env.cookie, assetID); rec.Code != http.StatusNoContent {
		t.Fatalf("restore status = %d, want %d (body %q)", rec.Code, http.StatusNoContent, rec.Body.String())
	}

	counts := env.runExpiry(t)
	if counts.Expired != 0 || counts.FilesRemoved != 0 {
		t.Fatalf("expiry counts = expired %d files %d, want the zero no-op over the restored row", counts.Expired, counts.FilesRemoved)
	}

	// The row survived as live, with its file, and it is back in the
	// timeline.
	live := env.assetRowOf(t, assetID)
	if live.DeletedAt.Valid {
		t.Error("the restored row is still marked deleted, want live")
	}
	requireFileState(t, file, true)
	page := env.getTimeline(t, env.cookie, "")
	if ids := timelineItemIDs(page.Items); len(ids) != 1 || ids[0] != assetID {
		t.Errorf("timeline after the restore holds %v, want the restored asset", ids)
	}
	if items := env.getTrash(t, env.cookie).Items; len(items) != 0 {
		t.Errorf("trash after the restore holds %v, want empty", trashItemIDs(items))
	}

	// The restore announced itself; nothing was deleted.
	restored := busEventsOf(t, env.bus, "asset.restored")
	if len(restored) != 1 || restored[0]["assetId"] != assetID.String() {
		t.Errorf("asset.restored events = %v, want exactly one", restored)
	}
	if deleted := busEventsOf(t, env.bus, "asset.deleted"); len(deleted) != 0 {
		t.Errorf("asset.deleted events = %v, want none", deleted)
	}
}

// TestTrashRestoreSetSemantics pins the mandated set semantics of the
// restore: restoring twice answers 204 both times (the second is the
// idempotent no-op over a live row), the same as restoring a row that
// was never trashed -- the two states are one and the same live row,
// which is exactly why the no-op is the only honest answer. Only
// unknown and foreign ids answer 404. The soft delete's own
// set-semantics arm (re-trashing a trashed row marks nothing) is
// pinned at the query level for Task 4's handler.
func TestTrashRestoreSetSemantics(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	adminID := env.sessionUserID(t)

	taken := time.Date(2026, 6, 4, 10, 0, 0, 0, time.UTC)
	trashed := env.uploadPhotoAs(t, env.cookie, "trash-twice", taken, jpegColorFixture(t, color.RGBA{R: 0xaa, G: 0xbb, B: 0xcc, A: 0xff}))
	neverTrashed := env.uploadPhotoAs(t, env.cookie, "trash-never", taken.Add(time.Hour), jpegColorFixture(t, color.RGBA{R: 0x01, G: 0x02, B: 0x03, A: 0xff}))

	// The soft delete is idempotent at the query level: the repeat
	// marks nothing (Task 4's handler answers both as the same 204).
	if rows := env.softDeleteAsset(t, uuid.UUID(trashed.Id), adminID); rows != 1 {
		t.Fatalf("first soft delete marked %d rows, want 1", rows)
	}
	if rows := env.softDeleteAsset(t, uuid.UUID(trashed.Id), adminID); rows != 0 {
		t.Fatalf("second soft delete marked %d rows, want 0 (set semantics)", rows)
	}

	// Restore twice: 204 both times.
	for i := 1; i <= 2; i++ {
		rec := env.restoreAsset(t, env.cookie, uuid.UUID(trashed.Id))
		if rec.Code != http.StatusNoContent {
			t.Fatalf("restore %d status = %d, want %d (body %q)", i, rec.Code, http.StatusNoContent, rec.Body.String())
		}
	}
	// The idempotent second restore is silent: one event, not two.
	if restored := busEventsOf(t, env.bus, "asset.restored"); len(restored) != 1 {
		t.Errorf("asset.restored events = %d, want exactly 1 (the no-op stays silent)", len(restored))
	}

	// A live row that was never trashed restores to the same 204: the
	// server cannot distinguish it from the just-restored row, and
	// both are already in the state the caller asked for.
	rec := env.restoreAsset(t, env.cookie, uuid.UUID(neverTrashed.Id))
	if rec.Code != http.StatusNoContent {
		t.Fatalf("restore of a never-trashed live asset status = %d, want %d (the idempotent 204)", rec.Code, http.StatusNoContent)
	}
	if restored := busEventsOf(t, env.bus, "asset.restored"); len(restored) != 1 {
		t.Errorf("asset.restored events after the no-op restore = %d, want still 1", len(restored))
	}

	// An unknown id answers 404.
	unknown := uuid.MustParse("31000000-0000-0000-0000-000000000001")
	if rec := env.restoreAsset(t, env.cookie, unknown); rec.Code != http.StatusNotFound {
		t.Fatalf("restore of an unknown id status = %d, want %d", rec.Code, http.StatusNotFound)
	}

	// A foreign trashed row is invisible: 404, byte-identical with
	// the unknown id's answer (the anti-enumeration discipline).
	graceID, graceCookie := env.createSecondUser(t, "trash-set-grace@example.com")
	graceAsset := env.uploadPhotoAs(t, graceCookie, "trash-set-grace", taken, jpegColorFixture(t, color.RGBA{R: 0x04, G: 0x05, B: 0x06, A: 0xff}))
	env.softDeleteAsset(t, uuid.UUID(graceAsset.Id), graceID)

	foreign := env.restoreAsset(t, env.cookie, uuid.UUID(graceAsset.Id))
	if foreign.Code != http.StatusNotFound {
		t.Fatalf("restore of a foreign asset status = %d, want %d", foreign.Code, http.StatusNotFound)
	}
	if unknownBody := env.restoreAsset(t, env.cookie, unknown).Body.String(); unknownBody != foreign.Body.String() {
		t.Errorf("the 404 bodies of the foreign (%q) and unknown (%q) ids differ", foreign.Body.String(), unknownBody)
	}
	// Grace can restore her own.
	if rec := env.restoreAsset(t, graceCookie, uuid.UUID(graceAsset.Id)); rec.Code != http.StatusNoContent {
		t.Fatalf("grace's restore of her own asset status = %d, want %d", rec.Code, http.StatusNoContent)
	}
}

// TestTrashDeletePermissions pins the permission split of the two
// delete surfaces: restore is owner-only (a foreign id is a 404 even
// for the administrator -- the trash view itself is owner-scoped, so
// an admin restore would be asymmetric), while the explicit permanent
// delete admits the owner and the administrator. A plain user
// deleting another's asset answers the anti-enumeration 404,
// byte-identical with an unknown id.
func TestTrashDeletePermissions(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	graceID, graceCookie := env.createSecondUser(t, "trash-perm-grace@example.com")

	taken := time.Date(2026, 6, 5, 10, 0, 0, 0, time.UTC)
	graceAsset := env.uploadPhotoAs(t, graceCookie, "trash-perm-grace", taken, jpegColorFixture(t, color.RGBA{R: 0x10, G: 0x20, B: 0x30, A: 0xff}))
	adminAsset := env.uploadPhotoAs(t, env.cookie, "trash-perm-admin", taken, jpegColorFixture(t, color.RGBA{R: 0x40, G: 0x50, B: 0x60, A: 0xff}))

	// Restore is owner-only: the administrator gets the same 404 a
	// stranger would.
	env.softDeleteAsset(t, uuid.UUID(graceAsset.Id), graceID)
	if rec := env.restoreAsset(t, env.cookie, uuid.UUID(graceAsset.Id)); rec.Code != http.StatusNotFound {
		t.Fatalf("admin restore of grace's asset status = %d, want %d (restore is owner-only)", rec.Code, http.StatusNotFound)
	}
	if rec := env.restoreAsset(t, graceCookie, uuid.UUID(graceAsset.Id)); rec.Code != http.StatusNoContent {
		t.Fatalf("grace's restore of her own asset status = %d, want %d", rec.Code, http.StatusNoContent)
	}

	// The permanent delete admits the administrator: grace's asset
	// goes at the admin's request.
	if rec := env.deleteAsset(t, env.cookie, uuid.UUID(graceAsset.Id)); rec.Code != http.StatusNoContent {
		t.Fatalf("admin delete of grace's asset status = %d, want %d (the admin override)", rec.Code, http.StatusNoContent)
	}
	env.requireNoAssetRow(t, uuid.UUID(graceAsset.Id))

	// A plain user cannot delete another's asset, and the 404 is
	// byte-identical with the unknown id's -- no existence leak.
	foreign := env.deleteAsset(t, graceCookie, uuid.UUID(adminAsset.Id))
	if foreign.Code != http.StatusNotFound {
		t.Fatalf("grace's delete of the admin's asset status = %d, want %d", foreign.Code, http.StatusNotFound)
	}
	unknown := uuid.MustParse("31000000-0000-0000-0000-000000000002")
	unknownRec := env.deleteAsset(t, graceCookie, unknown)
	if unknownRec.Code != http.StatusNotFound {
		t.Fatalf("delete of an unknown id status = %d, want %d", unknownRec.Code, http.StatusNotFound)
	}
	if foreign.Body.String() != unknownRec.Body.String() {
		t.Errorf("the 404 bodies of the foreign (%q) and unknown (%q) ids differ", foreign.Body.String(), unknownRec.Body.String())
	}

	// The admin's own asset goes at her own request.
	if rec := env.deleteAsset(t, env.cookie, uuid.UUID(adminAsset.Id)); rec.Code != http.StatusNoContent {
		t.Fatalf("owner delete status = %d, want %d", rec.Code, http.StatusNoContent)
	}
	env.requireNoAssetRow(t, uuid.UUID(adminAsset.Id))
}

// TestDeleteAssetSemantics pins the explicit permanent delete
// itself: it works on live and trashed rows alike (it is the surface
// beside the trash, not a detour through it), it answers 404 once
// the row is gone (a repeated request finds nothing left), it keeps
// shared blobs under the reference rule, and it takes the Live Photo
// pair together.
func TestDeleteAssetSemantics(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	_, graceCookie := env.createSecondUser(t, "trash-del-grace@example.com")
	adminID := env.sessionUserID(t)

	taken := time.Date(2026, 6, 6, 10, 0, 0, 0, time.UTC)
	live := env.uploadPhotoAs(t, env.cookie, "del-live", taken, jpegColorFixture(t, color.RGBA{R: 0x61, G: 0x62, B: 0x63, A: 0xff}))
	trashed := env.uploadPhotoAs(t, env.cookie, "del-trashed", taken.Add(time.Minute), jpegColorFixture(t, color.RGBA{R: 0x64, G: 0x65, B: 0x66, A: 0xff}))

	// A live row (never trashed) deletes permanently.
	liveRow := env.assetRowOf(t, uuid.UUID(live.Id))
	liveFile := env.storedFileOf(t, liveRow)
	env.writeThumbFixture(t, uuid.UUID(live.Id))
	if rec := env.deleteAsset(t, env.cookie, uuid.UUID(live.Id)); rec.Code != http.StatusNoContent {
		t.Fatalf("delete of a live asset status = %d, want %d", rec.Code, http.StatusNoContent)
	}
	env.requireNoAssetRow(t, uuid.UUID(live.Id))
	requireFileState(t, liveFile, false)
	requireFileState(t, filepath.Join(env.dataDir, "storage", "thumbs", live.Id.String()), false)

	// A trashed row deletes permanently too, with the same rules.
	trashedRow := env.assetRowOf(t, uuid.UUID(trashed.Id))
	trashedFile := env.storedFileOf(t, trashedRow)
	env.softDeleteAsset(t, uuid.UUID(trashed.Id), adminID)
	if rec := env.deleteAsset(t, env.cookie, uuid.UUID(trashed.Id)); rec.Code != http.StatusNoContent {
		t.Fatalf("delete of a trashed asset status = %d, want %d", rec.Code, http.StatusNoContent)
	}
	env.requireNoAssetRow(t, uuid.UUID(trashed.Id))
	requireFileState(t, trashedFile, false)

	// The repeat finds nothing left: 404.
	if rec := env.deleteAsset(t, env.cookie, uuid.UUID(live.Id)); rec.Code != http.StatusNotFound {
		t.Errorf("second delete status = %d, want %d", rec.Code, http.StatusNotFound)
	}

	// The shared blob: the first owner's delete keeps the file (the
	// other row references it); the second owner's delete removes it.
	shared := jpegColorFixture(t, color.RGBA{R: 0x67, G: 0x68, B: 0x69, A: 0xff})
	adminShared := env.uploadPhotoAs(t, env.cookie, "del-shared-admin", taken.Add(2*time.Minute), shared)
	graceShared := env.uploadPhotoAs(t, graceCookie, "del-shared-grace", taken.Add(2*time.Minute), shared)
	adminSharedRow := env.assetRowOf(t, uuid.UUID(adminShared.Id))
	graceSharedRow := env.assetRowOf(t, uuid.UUID(graceShared.Id))
	sharedFile := env.shareBlobBetween(t, adminSharedRow, graceSharedRow, shared)

	if rec := env.deleteAsset(t, env.cookie, uuid.UUID(adminShared.Id)); rec.Code != http.StatusNoContent {
		t.Fatalf("delete of the shared blob's first row status = %d, want %d", rec.Code, http.StatusNoContent)
	}
	requireFileState(t, sharedFile, true)
	if rec := env.deleteAsset(t, graceCookie, uuid.UUID(graceShared.Id)); rec.Code != http.StatusNoContent {
		t.Fatalf("delete of the shared blob's last row status = %d, want %d", rec.Code, http.StatusNoContent)
	}
	requireFileState(t, sharedFile, false)

	// The Live Photo pair goes together: the photo's delete removes
	// the linked component video row and both files.
	pairBody, pairContentType := buildUploadBody(t, map[string]string{
		"deviceAssetId":   "del-pair",
		"fileName":        "pair.jpg",
		"deviceCreatedAt": taken.Add(3 * time.Minute).Format(time.RFC3339),
	}, []assetUploadFile{
		{field: uploadPartPhoto, name: "pair.jpg", content: jpegColorFixture(t, color.RGBA{R: 0x70, G: 0x71, B: 0x72, A: 0xff})},
		{field: uploadPartLiveVideo, name: "pair.mov", content: movFixture()},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/assets", pairBody)
	req.Header.Set("Content-Type", pairContentType)
	req.AddCookie(env.cookie)
	rec := httptest.NewRecorder()
	env.router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("live photo upload status = %d, want %d (body %q)", rec.Code, http.StatusCreated, rec.Body.String())
	}
	pair := decodeAsset(t, rec)

	pairRow := env.assetRowOf(t, uuid.UUID(pair.Id))
	var componentRaw pgtype.UUID
	if err := env.pool.QueryRow(context.Background(),
		"SELECT id FROM assets WHERE owner_id = $1 AND is_component", pgUUID(adminID)).Scan(&componentRaw); err != nil {
		t.Fatalf("load the live photo's component row: %v", err)
	}
	componentID := uuid.UUID(componentRaw.Bytes)
	componentRow := env.assetRowOf(t, componentID)
	pairFile := env.storedFileOf(t, pairRow)
	componentFile := env.storedFileOf(t, componentRow)
	if rec := env.deleteAsset(t, env.cookie, uuid.UUID(pair.Id)); rec.Code != http.StatusNoContent {
		t.Fatalf("delete of the live photo status = %d, want %d", rec.Code, http.StatusNoContent)
	}
	env.requireNoAssetRow(t, uuid.UUID(pair.Id))
	env.requireNoAssetRow(t, componentID)
	requireFileState(t, pairFile, false)
	requireFileState(t, componentFile, false)

	// Unauthenticated requests never reach any of this.
	if rec := env.deleteAsset(t, nil, uuid.UUID(pair.Id)); rec.Code != http.StatusUnauthorized {
		t.Errorf("unauthenticated delete status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
	if rec := env.getTrashRaw(t, nil); rec.Code != http.StatusUnauthorized {
		t.Errorf("unauthenticated trash listing status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
	if rec := env.restoreAsset(t, nil, uuid.UUID(pair.Id)); rec.Code != http.StatusUnauthorized {
		t.Errorf("unauthenticated restore status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

// TestTrashListSemantics pins the listing: the deletion-order, the
// retention countdown under the default, a custom and a corrupt
// setting, the zero of a passed deadline, the hidden component rows
// of Live Photos, the owner scope, and the response bound.
func TestTrashListSemantics(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	adminID := env.sessionUserID(t)
	graceID, graceCookie := env.createSecondUser(t, "trash-list-grace@example.com")

	taken := time.Date(2026, 6, 7, 10, 0, 0, 0, time.UTC)
	older := env.uploadPhotoAs(t, env.cookie, "trash-list-1", taken, jpegColorFixture(t, color.RGBA{R: 0x80, G: 0x81, B: 0x82, A: 0xff}))
	newer := env.uploadPhotoAs(t, env.cookie, "trash-list-2", taken.Add(time.Minute), jpegColorFixture(t, color.RGBA{R: 0x83, G: 0x84, B: 0x85, A: 0xff}))

	// Distinct deletion instants: newer first.
	env.softDeleteAsset(t, uuid.UUID(older.Id), adminID)
	time.Sleep(10 * time.Millisecond)
	env.softDeleteAsset(t, uuid.UUID(newer.Id), adminID)

	items := env.getTrash(t, env.cookie).Items
	if ids := trashItemIDs(items); len(ids) != 2 || ids[0] != uuid.UUID(newer.Id) || ids[1] != uuid.UUID(older.Id) {
		t.Fatalf("trash order = %v, want the newest deletion first", ids)
	}

	// The countdown follows the retention setting: default 30, a
	// custom 1, and a corrupt value degrading to the default.
	if items[0].RemainingDays != 30 {
		t.Errorf("remainingDays with no setting = %d, want the default 30", items[0].RemainingDays)
	}
	env.setRetention(t, "1")
	if items = env.getTrash(t, env.cookie).Items; items[0].RemainingDays != 1 {
		t.Errorf("remainingDays with retention 1 = %d, want 1", items[0].RemainingDays)
	}
	env.setRetention(t, `"not a number"`)
	if items = env.getTrash(t, env.cookie).Items; items[0].RemainingDays != 30 {
		t.Errorf("remainingDays with a corrupt setting = %d, want the fallback default 30", items[0].RemainingDays)
	}

	// A deadline already passed (before the job's next run) counts
	// down to zero, not below it. The backdated row is no longer the
	// newest deletion, so the assertion addresses it by id.
	env.setRetention(t, "1")
	env.backdateTrash(t, uuid.UUID(newer.Id), 2)
	items = env.getTrash(t, env.cookie).Items
	var pastDeadline *apigen.TrashItem
	for i := range items {
		if uuid.UUID(items[i].Id) == uuid.UUID(newer.Id) {
			pastDeadline = &items[i]
		}
	}
	if pastDeadline == nil {
		t.Fatalf("the backdated asset %s is missing from the trash listing", newer.Id)
	}
	if pastDeadline.RemainingDays != 0 {
		t.Errorf("remainingDays past the deadline = %d, want the clamped 0", pastDeadline.RemainingDays)
	}
	env.setRetention(t, "")

	// The owner scope: grace's trash is not the admin's.
	graceAsset := env.uploadPhotoAs(t, graceCookie, "trash-list-grace", taken, jpegColorFixture(t, color.RGBA{R: 0x86, G: 0x87, B: 0x88, A: 0xff}))
	env.softDeleteAsset(t, uuid.UUID(graceAsset.Id), graceID)
	if items = env.getTrash(t, env.cookie).Items; len(items) != 2 {
		t.Errorf("admin's trash holds %d items, want 2 (grace's stays hers)", len(items))
	}
	if items = env.getTrash(t, graceCookie).Items; len(items) != 1 || uuid.UUID(items[0].Id) != uuid.UUID(graceAsset.Id) {
		t.Errorf("grace's trash = %v, want exactly her own asset", trashItemIDs(items))
	}

	// The Live Photo pair is one trash item: the component video row
	// (trashed with the photo) never lists.
	pairBody, pairContentType := buildUploadBody(t, map[string]string{
		"deviceAssetId":   "trash-list-pair",
		"fileName":        "pair.jpg",
		"deviceCreatedAt": taken.Add(time.Minute).Format(time.RFC3339),
	}, []assetUploadFile{
		{field: uploadPartPhoto, name: "pair.jpg", content: jpegColorFixture(t, color.RGBA{R: 0x89, G: 0x8a, B: 0x8b, A: 0xff})},
		{field: uploadPartLiveVideo, name: "pair.mov", content: movFixture()},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/assets", pairBody)
	req.Header.Set("Content-Type", pairContentType)
	req.AddCookie(env.cookie)
	rec := httptest.NewRecorder()
	env.router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("live photo upload status = %d, want %d", rec.Code, http.StatusCreated)
	}
	pair := decodeAsset(t, rec)
	if rows := env.softDeleteAsset(t, uuid.UUID(pair.Id), adminID); rows != 2 {
		t.Fatalf("soft delete of the live photo marked %d rows, want 2 (the pair rides along)", rows)
	}
	items = env.getTrash(t, env.cookie).Items
	if ids := trashItemIDs(items); len(ids) != 3 || ids[0] != uuid.UUID(pair.Id) {
		t.Fatalf("trash with the pair = %v, want the pair as ONE item (the photo), newest first", ids)
	}
	// The pair item keeps the component link of the detail shape; the
	// component row itself (trashed with the photo) never lists.
	var pairComponent pgtype.UUID
	if err := env.pool.QueryRow(context.Background(),
		"SELECT live_photo_video_id FROM assets WHERE id = $1", pgUUID(uuid.UUID(pair.Id))).Scan(&pairComponent); err != nil {
		t.Fatalf("load the pair's component link: %v", err)
	}
	if !pairComponent.Valid || uuid.UUID(pairComponent.Bytes) == uuid.Nil {
		t.Fatal("the pair row carries no component link")
	}
	if items[0].LivePhotoVideoId == nil || uuid.UUID(*items[0].LivePhotoVideoId) != uuid.UUID(pairComponent.Bytes) {
		t.Errorf("the pair item's livePhotoVideoId = %v, want the component's id", items[0].LivePhotoVideoId)
	}

	// The response bound: more trashed rows than the cap answer the
	// newest 200, oldest excess dropped.
	env.insertBulkTrashed(t, adminID, 205, taken.Add(2*time.Minute))
	items = env.getTrash(t, env.cookie).Items
	if len(items) != trashListCap {
		t.Fatalf("trash listing with 208 trashed rows holds %d items, want the capped %d", len(items), trashListCap)
	}
	// The newest rows are the bulk ones (trashed just now); the two
	// older fixture rows fell off the end of the bounded list.
	if ids := trashItemIDs(items); ids[0] == uuid.UUID(pair.Id) || ids[0] == uuid.UUID(newer.Id) || ids[0] == uuid.UUID(older.Id) {
		t.Errorf("the bounded listing's first item is a fixture row, want a newer bulk row")
	}
}

// insertBulkTrashed seeds n directly-trashed rows of one owner in one
// statement (the batch-action endpoint is Task 4's; the listing's
// bound is a property of the rows, not the endpoint). The checksums
// are distinct so the owner+checksum uniqueness holds across the
// bulk.
func (e *assetEnv) insertBulkTrashed(t *testing.T, owner uuid.UUID, n int, taken time.Time) {
	t.Helper()
	values := make([]string, 0, n)
	args := make([]any, 0, n*4)
	for i := 0; i < n; i++ {
		base := len(args)
		values = append(values, fmt.Sprintf(
			"($%d, $%d, $%d, $%d, 'blob', 'originals/trash/bulk.jpg', 'trash-bulk', 1024, 'image/jpeg', 'image', 0, 0, 'import', false, false, now())",
			base+1, base+2, base+3, base+4))
		args = append(args,
			pgUUID(bulkAssetID(i)), pgUUID(owner), fmt.Sprintf("trash-bulk-checksum-%d", i), taken)
	}
	e.exec(t, `INSERT INTO assets (
			id, owner_id, checksum, taken_at, storage_kind, file_path, original_filename,
			file_size, mime_type, media_kind, width, height, taken_at_source,
			is_hidden, is_component, deleted_at
		) VALUES `+strings.Join(values, ", "), args...)
}

// requireNoAssetRow asserts the asset row is gone.
func (e *assetEnv) requireNoAssetRow(t *testing.T, id uuid.UUID) {
	t.Helper()
	if _, err := e.queries.GetAssetByID(context.Background(), pgUUID(id)); err == nil {
		t.Fatalf("asset row %s still exists, want it gone", id)
	}
}
