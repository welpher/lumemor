// The directory rescan tests (E4 Task 2): the three arms of the
// diff through the real worker and queue -- the add arm's
// reference-style ingest (external row, real path, the existing
// pipeline's jobs), the orphan arm's pinned mark (deleted_at, the
// membership row staying, no revival), the change arm's snapshot
// re-run (checksum re-landed, the pipeline re-enqueued) -- plus the
// no-symlink walk and the idempotent re-scan (a second run over an
// unchanged directory writes nothing and publishes nothing). The
// duplicate arms pin the owner+checksum dedup: a live duplicate
// gains the membership, a trashed duplicate stays in the trash.
package jobs

import (
	"context"
	"encoding/json"
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
)

// rescanAssetRow loads the pipeline- and rescan-relevant columns of
// one asset row by its file path (the diff's matching key).
type rescanAssetRow struct {
	id            uuid.UUID
	storageKind   string
	checksum      string
	filePath      string
	fileSize      int64
	fileMtime     pgtype.Timestamptz
	takenAtSource string
	pending       int32
	readyAt       pgtype.Timestamptz
	deletedAt     pgtype.Timestamptz
	updatedAt     time.Time
}

// rescanAssetByPath loads the asset row of one owner by its exact
// file_path -- the walk's key.
func (e *env) rescanAssetByPath(t *testing.T, owner uuid.UUID, path string) (rescanAssetRow, bool) {
	t.Helper()
	var row rescanAssetRow
	err := e.pool.QueryRow(context.Background(), `
		SELECT id, storage_kind, checksum, file_path, file_size, file_mtime,
		       taken_at_source, pending_jobs, ready_at, deleted_at, updated_at
		FROM assets WHERE owner_id = $1 AND file_path = $2`,
		pgUUID(owner), path).
		Scan(&row.id, &row.storageKind, &row.checksum, &row.filePath, &row.fileSize, &row.fileMtime,
			&row.takenAtSource, &row.pending, &row.readyAt, &row.deletedAt, &row.updatedAt)
	if err != nil {
		return row, false
	}
	return row, true
}

// setMountRoots saves the mount-root allowlist through the admin
// face, opening the directory-album trust boundary over the given
// root.
func (e *env) setMountRoots(t *testing.T, roots ...string) {
	t.Helper()
	quoted := make([]string, len(roots))
	for i, root := range roots {
		quoted[i] = `"` + root + `"`
	}
	body := `{"roots":[` + strings.Join(quoted, ",") + `]}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/settings/mount-roots", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(e.cookie)
	rec := httptest.NewRecorder()
	e.router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("PUT /admin/settings/mount-roots status = %d, want %d (body %q)",
			rec.Code, http.StatusOK, rec.Body.String())
	}
}

// createDirectoryAlbum creates one directory album through the API
// over the given source directory and answers its id.
func (e *env) createDirectoryAlbum(t *testing.T, title, source string) uuid.UUID {
	t.Helper()
	body, err := json.Marshal(apigen.AlbumCreateRequest{
		Title:    title,
		RootPath: source,
	})
	if err != nil {
		t.Fatalf("marshal album body: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/albums", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(e.cookie)
	rec := httptest.NewRecorder()
	e.router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("POST /albums status = %d, want %d (body %q)", rec.Code, http.StatusCreated, rec.Body.String())
	}
	var response apigen.AlbumCreateResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode album body %q: %v", rec.Body.String(), err)
	}
	return response.Id
}

// rescan fires POST /albums/{albumId}/rescan and answers the run
// identifier the endpoint handed out.
func (e *env) rescan(t *testing.T, albumID uuid.UUID) uuid.UUID {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/albums/"+albumID.String()+"/rescan", nil)
	req.AddCookie(e.cookie)
	rec := httptest.NewRecorder()
	e.router.ServeHTTP(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("POST /albums/%s/rescan status = %d, want %d (body %q)",
			albumID, rec.Code, http.StatusAccepted, rec.Body.String())
	}
	var response apigen.AlbumRescanResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode rescan body %q: %v", rec.Body.String(), err)
	}
	return response.JobRunId
}

// waitRescanFinished polls the run's row until it closes, answering
// its final state.
func (e *env) waitRescanFinished(t *testing.T, runID uuid.UUID) string {
	t.Helper()
	var state string
	eventually(t, jobWaitTimeout, "the rescan run to finish", func() bool {
		return e.pool.QueryRow(context.Background(),
			"SELECT state FROM job_runs WHERE id = $1", pgUUID(runID)).Scan(&state) == nil &&
			(state == "succeeded" || state == "failed")
	})
	return state
}

// albumMemberCount counts the album's membership rows (every row,
// live or trashed -- the diff never removes them).
func (e *env) albumMemberCount(t *testing.T, albumID uuid.UUID) int {
	t.Helper()
	var count int
	if err := e.pool.QueryRow(context.Background(),
		"SELECT count(*) FROM album_items WHERE album_id = $1", pgUUID(albumID)).Scan(&count); err != nil {
		t.Fatalf("count album members: %v", err)
	}
	return count
}

// albumScannedAt loads the album's scan clock.
func (e *env) albumScannedAt(t *testing.T, albumID uuid.UUID) pgtype.Timestamptz {
	t.Helper()
	var scannedAt pgtype.Timestamptz
	if err := e.pool.QueryRow(context.Background(),
		"SELECT last_scanned_at FROM albums WHERE id = $1", pgUUID(albumID)).Scan(&scannedAt); err != nil {
		t.Fatalf("load the album's scan clock: %v", err)
	}
	return scannedAt
}

// writeSourceFile writes one source fixture with an explicit mtime
// (deterministic snapshots, no dependence on the clock the write
// happened at).
func writeSourceFile(t *testing.T, path string, content []byte, mtime time.Time) {
	t.Helper()
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("write source file %s: %v", path, err)
	}
	if err := os.Chtimes(path, mtime, mtime); err != nil {
		t.Fatalf("stamp the source mtime %s: %v", path, err)
	}
}

// TestAlbumRescanThreeArms is the mandated end-to-end of the diff:
// the first scan ingests the source by reference (external rows at
// their real paths, the pipeline jobs, the events), and the second
// scan resolves all three arms -- the vanished file's row is marked
// (the pinned orphan semantics: the membership row stays, the trash
// listing never shows it), the new file is ingested, and the changed
// file's snapshot, checksum and pipeline re-run.
func TestAlbumRescanThreeArms(t *testing.T) {
	requireExiftool(t)
	requireThumbnails(t)
	e := newEnv(t)

	root := t.TempDir()
	source := filepath.Join(root, "family")
	if err := os.MkdirAll(source, 0o755); err != nil {
		t.Fatalf("create the source directory: %v", err)
	}
	stamp := time.Date(2026, 3, 1, 9, 30, 0, int(time.Millisecond*300), time.UTC)
	golden, plain := goldenFixture(t), jpegBytes(t)
	writeSourceFile(t, filepath.Join(source, "a.jpg"), golden, stamp)
	writeSourceFile(t, filepath.Join(source, "b.jpg"), plain, stamp)

	e.setMountRoots(t, root)
	album := e.createDirectoryAlbum(t, "Family archive", source)

	// The first scan: the add arm over the whole source.
	run := e.rescan(t, album)
	if state := e.waitRescanFinished(t, run); state != "succeeded" {
		t.Fatalf("the first rescan closed %s, want succeeded", state)
	}

	a, ok := e.rescanAssetByPath(t, e.ownerID, filepath.Join(source, "a.jpg"))
	if !ok {
		t.Fatal("the first scan never ingested a.jpg")
	}
	b, ok := e.rescanAssetByPath(t, e.ownerID, filepath.Join(source, "b.jpg"))
	if !ok {
		t.Fatal("the first scan never ingested b.jpg")
	}
	for _, row := range []rescanAssetRow{a, b} {
		if row.storageKind != "external" {
			t.Errorf("storage_kind of %s = %q, want external (the reference-style ingest)", row.filePath, row.storageKind)
		}
		if row.takenAtSource != "file" {
			t.Errorf("taken_at_source of %s = %q, want file (the source the schema reserved for this importer)", row.filePath, row.takenAtSource)
		}
		if row.pending != 2 {
			t.Errorf("pending_jobs of %s = %d, want 2 (metadata + thumbnail, the image pipeline)", row.filePath, row.pending)
		}
		if !row.fileMtime.Valid || !row.fileMtime.Time.Equal(stamp.Truncate(time.Microsecond)) {
			t.Errorf("file_mtime of %s = %v, want the file's own %s", row.filePath, row.fileMtime, stamp)
		}
	}
	if a.checksum != sha256Hex(golden) {
		t.Errorf("checksum of a.jpg = %q, want the content digest %q", a.checksum, sha256Hex(golden))
	}
	if a.fileSize != int64(len(golden)) || b.fileSize != int64(len(plain)) {
		t.Errorf("sizes = %d/%d, want %d/%d", a.fileSize, b.fileSize, len(golden), len(plain))
	}
	if got := e.albumMemberCount(t, album); got != 2 {
		t.Errorf("membership rows after the first scan = %d, want 2", got)
	}
	// The scan's own clock: stamped by the run that changed the
	// album's content.
	if !e.albumScannedAt(t, album).Valid {
		t.Error("last_scanned_at is NULL after the first scan changed the album, want it stamped")
	}
	if got := countEvents(e.bus, "asset.added"); got != 2 {
		t.Errorf("asset.added events = %d, want 2", got)
	}
	if got := countEvents(e.bus, "album.items-changed"); got != 1 {
		t.Errorf("album.items-changed events = %d, want the one membership growth", got)
	}

	// The pipeline ran over the referenced files: both rows became
	// ready exactly like an upload's (the existing pipeline, not a
	// parallel one -- the jobs read the external path through the
	// kind-aware resolver).
	eventually(t, jobWaitTimeout, "the first scan's assets to become ready", func() bool {
		readyA, _ := e.rescanAssetByPath(t, e.ownerID, filepath.Join(source, "a.jpg"))
		readyB, _ := e.rescanAssetByPath(t, e.ownerID, filepath.Join(source, "b.jpg"))
		return readyA.readyAt.Valid && readyB.readyAt.Valid
	})
	if got := countEvents(e.bus, "asset.ready"); got != 2 {
		t.Errorf("asset.ready events = %d, want 2 (the pipeline of both ingested files)", got)
	}

	// The second scan: all three arms in one diff. a.jpg vanishes
	// from disk, c.jpg arrives, b.jpg's content changes.
	if err := os.Remove(filepath.Join(source, "a.jpg")); err != nil {
		t.Fatalf("remove a.jpg: %v", err)
	}
	changedPlain := append([]byte(nil), plain...)
	changedPlain = append(changedPlain, []byte(" changed")...)
	writeSourceFile(t, filepath.Join(source, "b.jpg"), changedPlain, stamp.Add(time.Minute))
	writeSourceFile(t, filepath.Join(source, "c.jpg"), plain, stamp)

	run2 := e.rescan(t, album)
	if state := e.waitRescanFinished(t, run2); state != "succeeded" {
		t.Fatalf("the second rescan closed %s, want succeeded", state)
	}

	// The vanish arm: the row is marked, the membership row stays,
	// and the trash listing never shows the orphan (the pin -- the
	// file is not the server's).
	vanished, ok := e.rescanAssetByPath(t, e.ownerID, filepath.Join(source, "a.jpg"))
	if !ok || !vanished.deletedAt.Valid {
		t.Fatalf("the vanished a.jpg row = (%v, %v), want the row still present with deleted_at set", ok, vanished.deletedAt)
	}
	if got := e.albumMemberCount(t, album); got != 3 {
		t.Errorf("membership rows after the second scan = %d, want 3 (soft delete never removed one)", got)
	}
	if got := countEvents(e.bus, "asset.trashed"); got != 1 {
		t.Errorf("asset.trashed events = %d, want the vanish arm's one", got)
	}
	requireTrashExcludes(t, e, vanished.id)

	// The add arm: c.jpg landed through the same reference-style
	// ingest.
	added, ok := e.rescanAssetByPath(t, e.ownerID, filepath.Join(source, "c.jpg"))
	if !ok {
		t.Fatal("the second scan never ingested c.jpg")
	}
	if added.storageKind != "external" || added.checksum != sha256Hex(plain) {
		t.Errorf("c.jpg row = (%q, %q), want the external ingest of the plain bytes",
			added.storageKind, added.checksum)
	}
	if got := countEvents(e.bus, "asset.added"); got != 3 {
		t.Errorf("asset.added events = %d, want 3 (two of the first scan, one of c.jpg)", got)
	}

	// The change arm: b.jpg's snapshot and checksum re-landed from
	// the re-read, and the pipeline re-enqueued (the river jobs of
	// the asset doubled -- the metadata and thumbnail pair of both
	// scans).
	changed, ok := e.rescanAssetByPath(t, e.ownerID, filepath.Join(source, "b.jpg"))
	if !ok {
		t.Fatal("b.jpg's row vanished, want the change arm to keep it")
	}
	if changed.checksum != sha256Hex(changedPlain) {
		t.Errorf("checksum of the changed b.jpg = %q, want the re-read digest %q", changed.checksum, sha256Hex(changedPlain))
	}
	if changed.fileSize != int64(len(changedPlain)) {
		t.Errorf("file_size of the changed b.jpg = %d, want %d", changed.fileSize, len(changedPlain))
	}
	if !changed.fileMtime.Time.Equal(stamp.Add(time.Minute).Truncate(time.Microsecond)) {
		t.Errorf("file_mtime of the changed b.jpg = %v, want the new %s", changed.fileMtime, stamp.Add(time.Minute))
	}
	if got := countEvents(e.bus, "asset.updated"); got != 1 {
		t.Errorf("asset.updated events = %d, want the change arm's one", got)
	}
	kinds := e.riverJobsOf(t, b.id)
	if kinds["metadata"] != 2 || kinds["thumbnail"] != 2 {
		t.Errorf("b.jpg river jobs = %v, want the metadata and thumbnail pair of both scans (the re-run)", kinds)
	}
	// The re-run's readiness discipline: no second asset.ready for
	// b.jpg, its ready_at keeps the first stamp.
	eventually(t, jobWaitTimeout, "the changed file's re-extract to land", func() bool {
		var updatedAt time.Time
		return e.pool.QueryRow(context.Background(),
			"SELECT updated_at FROM exif WHERE asset_id = $1", pgUUID(b.id)).Scan(&updatedAt) == nil &&
			updatedAt.After(b.updatedAt)
	})
	if got := countEvents(e.bus, "asset.ready"); got != 3 {
		t.Errorf("asset.ready events = %d, want 3 (c.jpg's pipeline alone; the re-run announces nothing)", got)
	}
}

// requireTrashExcludes asserts the trash listing never shows the
// given asset: the orphan's file is not the server's, so the trash
// has no meaning for it.
func requireTrashExcludes(t *testing.T, e *env, assetID uuid.UUID) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/trash", nil)
	req.AddCookie(e.cookie)
	rec := httptest.NewRecorder()
	e.router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /trash status = %d, want %d (body %q)", rec.Code, http.StatusOK, rec.Body.String())
	}
	var response apigen.TrashResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode trash body %q: %v", rec.Body.String(), err)
	}
	for _, item := range response.Items {
		if item.Id == assetID {
			t.Errorf("the trash listing shows the orphaned external asset %s, want it never listed", assetID)
		}
	}
}

// TestAlbumRescanIdempotent pins the plan's pinned contract: the
// SAME directory scanned twice with no disk changes performs ZERO
// product writes (no new rows, no membership changes, no snapshot
// updates -- every row's updated_at keeps its stamp) and publishes
// ZERO asset or album events (the run's own job_runs bookkeeping
// and job.* events are the progress surface, not product state).
// The snapshot waits for the first scan's pipeline to settle: the
// pipeline jobs of the INGEST are that scan's own work, and their
// writes must not be mistaken for the second run's.
func TestAlbumRescanIdempotent(t *testing.T) {
	requireExiftool(t)
	requireThumbnails(t)
	e := newEnv(t)

	root := t.TempDir()
	source := filepath.Join(root, "family")
	if err := os.MkdirAll(source, 0o755); err != nil {
		t.Fatalf("create the source directory: %v", err)
	}
	stamp := time.Date(2026, 4, 2, 11, 0, 0, 0, time.UTC)
	plain := jpegBytes(t)
	writeSourceFile(t, filepath.Join(source, "a.jpg"), plain, stamp)
	// A non-media neighbor: the walk's noise, never a candidate and
	// never a work unit of the run's counts.
	if err := os.WriteFile(filepath.Join(source, "notes.txt"), []byte("ignore"), 0o644); err != nil {
		t.Fatalf("write the noise file: %v", err)
	}

	e.setMountRoots(t, root)
	album := e.createDirectoryAlbum(t, "Family archive", source)

	if state := e.waitRescanFinished(t, e.rescan(t, album)); state != "succeeded" {
		t.Fatalf("the first rescan closed %s, want succeeded", state)
	}
	row, ok := e.rescanAssetByPath(t, e.ownerID, filepath.Join(source, "a.jpg"))
	if !ok {
		t.Fatal("the first scan never ingested a.jpg")
	}
	// The pipeline settles before the snapshot: its writes are the
	// first scan's ingest work.
	eventually(t, jobWaitTimeout, "the first scan's asset to become ready", func() bool {
		settled, _ := e.rescanAssetByPath(t, e.ownerID, filepath.Join(source, "a.jpg"))
		return settled.readyAt.Valid
	})
	row, ok = e.rescanAssetByPath(t, e.ownerID, filepath.Join(source, "a.jpg"))
	if !ok {
		t.Fatal("the settled row vanished")
	}

	// The settled counts the second run must not move.
	assetsBefore := e.countAssets(t)
	scannedBefore := e.albumScannedAt(t, album)
	assetEventsBefore := map[string]int{
		"asset.added":         countEvents(e.bus, "asset.added"),
		"asset.updated":       countEvents(e.bus, "asset.updated"),
		"asset.trashed":       countEvents(e.bus, "asset.trashed"),
		"asset.restored":      countEvents(e.bus, "asset.restored"),
		"asset.ready":         countEvents(e.bus, "asset.ready"),
		"album.items-changed": countEvents(e.bus, "album.items-changed"),
	}
	updatedBefore := row.updatedAt

	// The second scan: the same directory, no disk changes.
	if state := e.waitRescanFinished(t, e.rescan(t, album)); state != "succeeded" {
		t.Fatalf("the idempotent rescan closed %s, want succeeded", state)
	}

	// The events are synchronous with the run's close; a short
	// settle makes the counts deterministic.
	time.Sleep(300 * time.Millisecond)

	if got := e.countAssets(t); got != assetsBefore {
		t.Errorf("asset rows after the idempotent rescan = %d, want the unchanged %d", got, assetsBefore)
	}
	if after, ok := e.rescanAssetByPath(t, e.ownerID, filepath.Join(source, "a.jpg")); !ok || !after.updatedAt.Equal(updatedBefore) {
		t.Errorf("the unchanged a.jpg's updated_at moved, want zero writes")
	}
	for name, before := range assetEventsBefore {
		if got := countEvents(e.bus, name); got != before {
			t.Errorf("%s events after the idempotent rescan = %d, want the unchanged %d", name, got, before)
		}
	}
	if got := e.albumMemberCount(t, album); got != 1 {
		t.Errorf("membership rows after the idempotent rescan = %d, want the unchanged 1", got)
	}
	// The scan clock keeps its stamp: an unchanged re-scan writes
	// nothing, the album row included.
	if after := e.albumScannedAt(t, album); !after.Time.Equal(scannedBefore.Time) || after.Valid != scannedBefore.Valid {
		t.Errorf("last_scanned_at moved on the unchanged re-scan, want the kept stamp %s", scannedBefore.Time)
	}
	// The run row itself is the progress surface: the run exists,
	// closed succeeded, and its counts carry the diff's size (one
	// candidate, nothing else -- the noise file is not a unit).
	var state string
	var totalI, doneI, failedI int32
	if err := e.pool.QueryRow(context.Background(), `
		SELECT state, total, done, failed FROM job_runs
		WHERE kind = 'album_rescan' ORDER BY created_at DESC, id DESC LIMIT 1`).
		Scan(&state, &totalI, &doneI, &failedI); err != nil {
		t.Fatalf("load the second run's row: %v", err)
	}
	if state != "succeeded" || totalI != 1 || doneI != 1 || failedI != 0 {
		t.Errorf("the idempotent run's row = (%s, total %d, done %d, failed %d), want (succeeded, 1, 1, 0)",
			state, totalI, doneI, failedI)
	}
}

// countAssets counts the owner's asset rows.
func (e *env) countAssets(t *testing.T) int {
	t.Helper()
	var count int
	if err := e.pool.QueryRow(context.Background(),
		"SELECT count(*) FROM assets WHERE owner_id = $1", pgUUID(e.ownerID)).Scan(&count); err != nil {
		t.Fatalf("count assets: %v", err)
	}
	return count
}

// TestAlbumRescanSkipsSymlinks pins the walk's no-symlink
// discipline: a symlinked FILE inside the root is never ingested
// and a symlinked DIRECTORY is never descended -- the walk resolves
// real trees only, and the source's link to outside content stays
// the family's own business.
func TestAlbumRescanSkipsSymlinks(t *testing.T) {
	e := newEnv(t)

	root := t.TempDir()
	source := filepath.Join(root, "family")
	if err := os.MkdirAll(source, 0o755); err != nil {
		t.Fatalf("create the source directory: %v", err)
	}
	stamp := time.Date(2026, 5, 3, 8, 0, 0, 0, time.UTC)
	writeSourceFile(t, filepath.Join(source, "real.jpg"), jpegBytes(t), stamp)

	// A symlinked file: a linked copy of the real photo under a
	// whitelisted name.
	if err := os.Symlink(filepath.Join(source, "real.jpg"), filepath.Join(source, "linked.jpg")); err != nil {
		t.Fatalf("symlink the file entry: %v", err)
	}
	// A symlinked directory whose target holds an ingestable file:
	// the walk must not descend into it.
	outside := t.TempDir()
	writeSourceFile(t, filepath.Join(outside, "escape.jpg"), jpegBytes(t), stamp)
	if err := os.Symlink(outside, filepath.Join(source, "linked-dir")); err != nil {
		t.Fatalf("symlink the directory entry: %v", err)
	}

	e.setMountRoots(t, root)
	album := e.createDirectoryAlbum(t, "Family archive", source)

	if state := e.waitRescanFinished(t, e.rescan(t, album)); state != "succeeded" {
		t.Fatalf("the rescan closed %s, want succeeded", state)
	}

	if _, ok := e.rescanAssetByPath(t, e.ownerID, filepath.Join(source, "linked.jpg")); ok {
		t.Error("the walk ingested the symlinked file entry, want it skipped")
	}
	if _, ok := e.rescanAssetByPath(t, e.ownerID, filepath.Join(source, "linked-dir", "escape.jpg")); ok {
		t.Error("the walk descended into the symlinked directory, want it never followed")
	}
	if _, ok := e.rescanAssetByPath(t, e.ownerID, filepath.Join(source, "real.jpg")); !ok {
		t.Error("the walk missed the real file beside the links, want it ingested")
	}
}

// TestAlbumRescanDuplicates pins the add arm's owner+checksum dedup
// through the existing uniqueness: a file whose content the owner
// already holds as a LIVE upload gains the album membership of the
// existing row (no second row -- the content identity is the
// server's), and a file matching TRASHED content is left alone (no
// revival semantics; the next scan after a restore re-links it).
func TestAlbumRescanDuplicates(t *testing.T) {
	e := newEnv(t)

	// One live blob upload of the golden content.
	golden := goldenFixture(t)
	rec := e.postUpload(t, map[string]string{
		"deviceAssetId":   "device-rescan-dup-1",
		"fileName":        "golden.jpg",
		"deviceCreatedAt": "2020-05-05T09:00:00Z",
	}, []uploadFileFixture{
		{field: "photo", name: "golden.jpg", content: golden},
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("the seeding upload status = %d, want %d (body %q)", rec.Code, http.StatusCreated, rec.Body.String())
	}
	var uploaded apigen.AssetResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &uploaded); err != nil {
		t.Fatalf("decode the upload body: %v", err)
	}
	// One trashed blob upload of distinct content.
	plain := jpegBytes(t)
	rec = e.postUpload(t, map[string]string{
		"deviceAssetId":   "device-rescan-dup-2",
		"fileName":        "plain.jpg",
		"deviceCreatedAt": "2020-05-05T09:00:00Z",
	}, []uploadFileFixture{
		{field: "photo", name: "plain.jpg", content: plain},
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("the trashed seeding upload status = %d, want %d (body %q)", rec.Code, http.StatusCreated, rec.Body.String())
	}
	var trashedUpload apigen.AssetResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &trashedUpload); err != nil {
		t.Fatalf("decode the trashed upload body: %v", err)
	}
	if _, err := e.pool.Exec(context.Background(),
		"UPDATE assets SET deleted_at = now() WHERE id = $1", pgUUID(trashedUpload.Id)); err != nil {
		t.Fatalf("trash the seeded upload: %v", err)
	}

	root := t.TempDir()
	source := filepath.Join(root, "family")
	if err := os.MkdirAll(source, 0o755); err != nil {
		t.Fatalf("create the source directory: %v", err)
	}
	stamp := time.Date(2026, 6, 4, 10, 0, 0, 0, time.UTC)
	writeSourceFile(t, filepath.Join(source, "same-content.jpg"), golden, stamp)
	writeSourceFile(t, filepath.Join(source, "trashed-content.jpg"), plain, stamp)

	e.setMountRoots(t, root)
	album := e.createDirectoryAlbum(t, "Family archive", source)

	if state := e.waitRescanFinished(t, e.rescan(t, album)); state != "succeeded" {
		t.Fatalf("the rescan closed %s, want succeeded", state)
	}

	// The live duplicate: no new row, the existing row gained the
	// membership.
	if _, ok := e.rescanAssetByPath(t, e.ownerID, filepath.Join(source, "same-content.jpg")); ok {
		t.Error("the add arm created a second row of content the owner already holds, want the uniqueness respected")
	}
	var members int
	if err := e.pool.QueryRow(context.Background(),
		"SELECT count(*) FROM album_items WHERE album_id = $1 AND asset_id = $2",
		pgUUID(album), pgUUID(uploaded.Id)).Scan(&members); err != nil {
		t.Fatalf("count the live duplicate's membership: %v", err)
	}
	if members != 1 {
		t.Errorf("the live duplicate's membership rows = %d, want the existing row linked in", members)
	}
	// The album's story grew without a new row: the one
	// items-changed event, no asset.added for a duplicate.
	if got := countEvents(e.bus, "asset.added"); got != 0 {
		t.Errorf("asset.added events = %d, want 0 (no new row landed)", got)
	}
	if got := countEvents(e.bus, "album.items-changed"); got != 1 {
		t.Errorf("album.items-changed events = %d, want the one membership growth", got)
	}

	// The trashed duplicate: left alone -- no membership, no row, no
	// revival.
	var trashedMembers int
	if err := e.pool.QueryRow(context.Background(),
		"SELECT count(*) FROM album_items WHERE album_id = $1 AND asset_id = $2",
		pgUUID(album), pgUUID(trashedUpload.Id)).Scan(&trashedMembers); err != nil {
		t.Fatalf("count the trashed duplicate's membership: %v", err)
	}
	if trashedMembers != 0 {
		t.Errorf("the trashed duplicate's membership rows = %d, want it left alone (no revival)", trashedMembers)
	}
	var stillTrashed pgtype.Timestamptz
	if err := e.pool.QueryRow(context.Background(),
		"SELECT deleted_at FROM assets WHERE id = $1", pgUUID(trashedUpload.Id)).Scan(&stillTrashed); err != nil {
		t.Fatalf("load the trashed duplicate's row: %v", err)
	}
	if !stillTrashed.Valid {
		t.Error("the trashed duplicate was revived by the scan, want it staying in the trash")
	}
}

// TestAlbumRescanRootGuard pins the run's honest failure when the
// source root can no longer be walked truthfully: a root that
// vanished (the unmounted volume) or was replaced by a symlink
// (the swapped source) closes the run FAILED -- an empty walk
// would have marked every member vanished, which is the owner's
// state to see, not the scan's to perform silently.
func TestAlbumRescanRootGuard(t *testing.T) {
	e := newEnv(t)

	root := t.TempDir()
	source := filepath.Join(root, "family")
	if err := os.MkdirAll(source, 0o755); err != nil {
		t.Fatalf("create the source directory: %v", err)
	}
	stamp := time.Date(2026, 7, 5, 12, 0, 0, 0, time.UTC)
	writeSourceFile(t, filepath.Join(source, "a.jpg"), jpegBytes(t), stamp)

	e.setMountRoots(t, root)
	album := e.createDirectoryAlbum(t, "Family archive", source)
	if state := e.waitRescanFinished(t, e.rescan(t, album)); state != "succeeded" {
		t.Fatalf("the first rescan closed %s, want succeeded", state)
	}

	t.Run("VanishedRoot", func(t *testing.T) {
		if err := os.RemoveAll(source); err != nil {
			t.Fatalf("remove the source root: %v", err)
		}
		if state := e.waitRescanFinished(t, e.rescan(t, album)); state != "failed" {
			t.Fatalf("the vanished-root rescan closed %s, want failed", state)
		}
		// The members stayed live: the run failed, nothing was
		// marked.
		if got := e.countAssets(t); got != 1 {
			t.Errorf("asset rows after the vanished-root run = %d, want the unchanged 1", got)
		}
	})

	t.Run("RootReplacedBySymlink", func(t *testing.T) {
		// The root's directory returns, then is replaced by a
		// symlink to another real tree: the walk's guard refuses.
		if err := os.Symlink(t.TempDir(), source); err != nil {
			t.Fatalf("replace the source root with a symlink: %v", err)
		}
		if state := e.waitRescanFinished(t, e.rescan(t, album)); state != "failed" {
			t.Fatalf("the symlinked-root rescan closed %s, want failed", state)
		}
		if got := e.countAssets(t); got != 1 {
			t.Errorf("asset rows after the symlinked-root run = %d, want the unchanged 1", got)
		}
	})
}
