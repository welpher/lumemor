// The folder live sync tests: the creation-time scan (a new directory
// album's first scan is the creation's own), the periodic
// directory-snapshot diff (an idle folder writes nothing, an added
// file reaches the wall, a removed file leaves it through the pinned
// orphan semantics), and the sweep's two disciplines (an in-flight
// scan of the album is jumped over, an unwalkable root decides
// nothing).
//
// The tests drive RunFolderSync directly -- the shared function the
// periodic worker runs on its cadence -- because the comparison is
// the subject, not River's clock: the test environment disables the
// periodic registration (see jobs_test.go), so no background tick can
// race an assertion.
package jobs

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"

	apigen "github.com/welpher/lumemor/server/internal/api/gen"
	dbgen "github.com/welpher/lumemor/server/internal/db/gen"
)

// syncNow runs one sweep against the test env's own baseline store
// and enqueue seam -- the production wiring the queue built.
func (e *env) syncNow(t *testing.T) FolderSyncCounts {
	t.Helper()
	counts, err := RunFolderSync(context.Background(), e.pool, e.queue.snapshots,
		e.queue.EnqueueAlbumRescan, e.queue.logger)
	if err != nil {
		t.Fatalf("run the folder sync sweep: %v", err)
	}
	return counts
}

// waitNewScan waits for a run of the album newer than the given one to
// close succeeded, answering its id: the sweep's enqueued scan.
func (e *env) waitNewScan(t *testing.T, albumID, after uuid.UUID) uuid.UUID {
	t.Helper()
	var runID uuid.UUID
	eventually(t, jobWaitTimeout, "the folder sync's scan to appear", func() bool {
		return e.pool.QueryRow(context.Background(), `
			SELECT id FROM job_runs
			WHERE kind = 'album_rescan' AND album_id = $1 AND id <> $2
			ORDER BY created_at DESC, id DESC LIMIT 1`,
			pgUUID(albumID), pgUUID(after)).Scan(&runID) == nil
	})
	if state := e.waitRescanFinished(t, runID); state != "succeeded" {
		t.Fatalf("the folder sync's scan closed %s, want succeeded", state)
	}
	return runID
}

// liveMemberCount counts the album's LIVE members through the API's own
// detail read -- the number the wall's count renders.
func (e *env) liveMemberCount(t *testing.T, albumID uuid.UUID) int {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/albums/"+albumID.String(), nil)
	req.AddCookie(e.cookie)
	rec := httptest.NewRecorder()
	e.router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /albums/%s status = %d, want %d (body %q)", albumID, rec.Code, http.StatusOK, rec.Body.String())
	}
	var detail apigen.AlbumDetail
	if err := json.Unmarshal(rec.Body.Bytes(), &detail); err != nil {
		t.Fatalf("decode the album detail %q: %v", rec.Body.String(), err)
	}
	return detail.ItemCount
}

// TestFolderSyncCreationTimeScan pins Task 1: a newly created
// directory album scans AT CREATION. No manual rescan is fired
// anywhere in this test -- the creation's own enqueue is the first
// scan -- and the album's live membership converges onto the source
// folder's files.
func TestFolderSyncCreationTimeScan(t *testing.T) {
	requireExiftool(t)
	requireThumbnails(t)
	e := newEnv(t)

	source := filepath.Join(t.TempDir(), "family")
	if err := os.MkdirAll(source, 0o755); err != nil {
		t.Fatalf("create the source directory: %v", err)
	}
	stamp := time.Date(2026, 9, 12, 8, 0, 0, 0, time.UTC)
	writeSourceFile(t, filepath.Join(source, "a.jpg"), jpegBytes(t), stamp)

	// The creation's response is the fresh card -- zero members is the
	// scan's starting point, not the album's meaning -- and the run it
	// enqueued is already in the task panel.
	album := e.createDirectoryAlbum(t, "Family archive", source)

	eventually(t, jobWaitTimeout, "the album's live membership to converge", func() bool {
		return e.liveMemberCount(t, album) == 1
	})
	e.waitCreationScan(t, album)
	// The scan's ownership: the row is the album's own run kind, and
	// the ingest is the existing pipeline's (an external row at the
	// file's real path).
	var kind, state string
	if err := e.pool.QueryRow(context.Background(), `
		SELECT kind, state FROM job_runs WHERE album_id = $1
		ORDER BY created_at DESC, id DESC LIMIT 1`, pgUUID(album)).Scan(&kind, &state); err != nil {
		t.Fatalf("load the creation scan's row: %v", err)
	}
	if kind != AlbumRescanKind || state != "succeeded" {
		t.Errorf("the creation scan's row = (%s, %s), want (album_rescan, succeeded)", kind, state)
	}
	row, ok := e.rescanAssetByPath(t, e.ownerID, filepath.Join(source, "a.jpg"))
	if !ok || row.storageKind != "external" {
		t.Fatalf("the creation scan's ingest = (%v, %q), want an external row at the file's path", ok, row.storageKind)
	}
}

// TestFolderSyncFollowsTheFolder is the block's story: the periodic
// diff keeps the album's wall equal to the folder. The idle folder
// writes nothing; a dropped file reaches the wall through the
// existing ingest; a removed file leaves through the pinned orphan
// semantics (deleted_at directly, never the trash view); a
// non-media neighbor's change never wakes the sweep.
func TestFolderSyncFollowsTheFolder(t *testing.T) {
	requireExiftool(t)
	requireThumbnails(t)
	e := newEnv(t)

	source := filepath.Join(t.TempDir(), "family")
	if err := os.MkdirAll(source, 0o755); err != nil {
		t.Fatalf("create the source directory: %v", err)
	}
	stamp := time.Date(2026, 9, 13, 9, 0, 0, 0, time.UTC)
	plain, golden := jpegBytes(t), goldenFixture(t)
	writeSourceFile(t, filepath.Join(source, "a.jpg"), plain, stamp)
	// The source's own noise: the walk never counts it, so its change
	// must never wake a scan.
	writeSourceFile(t, filepath.Join(source, "README.md"), []byte("notes"), stamp)

	album := e.createDirectoryAlbum(t, "Family archive", source)
	first := e.waitCreationScan(t, album)

	// THE IDLE FOLDER: the baseline matches the folder exactly, so
	// the sweep decides nothing at all -- no run row, no asset write,
	// no event.
	runsBefore := e.countRuns(t, album)
	eventsBefore := map[string]int{
		"asset.added":         countEvents(e.bus, "asset.added"),
		"asset.trashed":       countEvents(e.bus, "asset.trashed"),
		"album.items-changed": countEvents(e.bus, "album.items-changed"),
	}
	counts := e.syncNow(t)
	if counts.Albums != 1 || counts.Unchanged != 1 || counts.Enqueued != 0 || counts.Skipped != 0 {
		t.Errorf("the idle sweep's counts = %+v, want the one album unchanged and nothing enqueued", counts)
	}
	time.Sleep(200 * time.Millisecond)
	if got := e.countRuns(t, album); got != runsBefore {
		t.Errorf("job runs after the idle sweep = %d, want the unchanged %d (an idle folder writes nothing)", got, runsBefore)
	}
	for name, before := range eventsBefore {
		if got := countEvents(e.bus, name); got != before {
			t.Errorf("%s events after the idle sweep = %d, want the unchanged %d", name, got, before)
		}
	}
	// The noise file's change is not a difference either.
	writeSourceFile(t, filepath.Join(source, "README.md"), []byte("edited notes"), stamp.Add(time.Hour))
	if counts := e.syncNow(t); counts.Enqueued != 0 {
		t.Errorf("the sweep enqueued %d scans for a non-media file's change, want 0", counts.Enqueued)
	}

	// THE ADD: a file dropped into the folder reaches the wall with no
	// manual action -- the sweep enqueues the album's own scan, whose
	// add arm ingests by reference (the existing pipeline).
	writeSourceFile(t, filepath.Join(source, "b.jpg"), golden, stamp)
	if counts := e.syncNow(t); counts.Enqueued != 1 {
		t.Fatalf("the sweep enqueued %d scans for the added file, want 1 (%+v)", counts.Enqueued, counts)
	}
	second := e.waitNewScan(t, album, first)
	eventually(t, jobWaitTimeout, "the added file's tile", func() bool {
		return e.liveMemberCount(t, album) == 2
	})
	added, ok := e.rescanAssetByPath(t, e.ownerID, filepath.Join(source, "b.jpg"))
	if !ok || added.storageKind != "external" {
		t.Fatalf("the added file's row = (%v, %q), want the reference-style external ingest", ok, added.storageKind)
	}

	// The baseline advanced with the scan: the next sweep over the
	// unchanged folder is idle again (no scan per tick, ever).
	if counts := e.syncNow(t); counts.Enqueued != 0 || counts.Unchanged != 1 {
		t.Errorf("the sweep after the scan's convergence = %+v, want the album unchanged and nothing enqueued", counts)
	}

	// THE REMOVE: the file leaves the folder, the wall loses the tile
	// through the pinned orphan semantics -- the row is marked
	// deleted_at directly, the membership row stays, and the trash
	// view never lists it (the file is not the server's).
	if err := os.Remove(filepath.Join(source, "b.jpg")); err != nil {
		t.Fatalf("remove the added file: %v", err)
	}
	itemsChangedBefore := countEvents(e.bus, "album.items-changed")
	if counts := e.syncNow(t); counts.Enqueued != 1 {
		t.Fatalf("the sweep enqueued %d scans for the removed file, want 1 (%+v)", counts.Enqueued, counts)
	}
	e.waitNewScan(t, album, second)
	vanished, ok := e.rescanAssetByPath(t, e.ownerID, filepath.Join(source, "b.jpg"))
	if !ok || !vanished.deletedAt.Valid {
		t.Fatalf("the removed file's row = (%v, %v), want the row kept with deleted_at set", ok, vanished.deletedAt)
	}
	eventually(t, jobWaitTimeout, "the removed tile to leave the wall", func() bool {
		return e.liveMemberCount(t, album) == 1
	})
	if got := countEvents(e.bus, "asset.trashed"); got == 0 {
		t.Error("the vanish arm published no asset.trashed, want the removed file's event")
	}
	// The album-level signal rides the vanish too: an open album page
	// restarts its walk on it, so the departed tile and its count drop
	// together -- the convergence an automatic sync must deliver.
	if got := countEvents(e.bus, "album.items-changed"); got != itemsChangedBefore+1 {
		t.Errorf("album.items-changed events after the vanish = %d, want %d (the run's one)", got, itemsChangedBefore+1)
	}
	requireTrashExcludes(t, e, vanished.id)
}

// TestFolderSyncSkipsInFlightScans pins the sweep's single-flight with
// the manual rescan: while a run of the album is in flight, the sweep
// decides nothing for it -- the run in flight owns the next baseline
// -- and starts no second walk.
func TestFolderSyncSkipsInFlightScans(t *testing.T) {
	requireExiftool(t)
	requireThumbnails(t)
	e := newEnv(t)

	source := filepath.Join(t.TempDir(), "family")
	if err := os.MkdirAll(source, 0o755); err != nil {
		t.Fatalf("create the source directory: %v", err)
	}
	stamp := time.Date(2026, 9, 14, 10, 0, 0, 0, time.UTC)
	writeSourceFile(t, filepath.Join(source, "a.jpg"), jpegBytes(t), stamp)

	album := e.createDirectoryAlbum(t, "Family archive", source)
	e.waitCreationScan(t, album)

	// A run of this album still in flight, seeded the way the
	// endpoint's own in-flight guard reads it (queued, no River job --
	// it never moves).
	runID, err := uuid.NewV7()
	if err != nil {
		t.Fatalf("generate the in-flight run id: %v", err)
	}
	if err := e.queries.InsertAlbumRescanRun(context.Background(), dbgen.InsertAlbumRescanRunParams{
		ID:      pgUUID(runID),
		Kind:    AlbumRescanKind,
		Total:   0,
		AlbumID: pgUUID(album),
	}); err != nil {
		t.Fatalf("seed the in-flight run: %v", err)
	}

	// The folder changes underneath the in-flight run: the sweep must
	// still not start a second one.
	writeSourceFile(t, filepath.Join(source, "b.jpg"), jpegBytes(t), stamp)
	if counts := e.syncNow(t); counts.Enqueued != 0 || counts.Skipped != 1 {
		t.Errorf("the sweep with a scan in flight = %+v, want it skipped and nothing enqueued", counts)
	}
	var runs int
	if err := e.pool.QueryRow(context.Background(),
		"SELECT count(*) FROM job_runs WHERE album_id = $1", pgUUID(album)).Scan(&runs); err != nil {
		t.Fatalf("count the album's runs: %v", err)
	}
	if runs != 2 {
		t.Errorf("the album's runs after the skipped sweep = %d, want the creation's and the seeded one (no third)", runs)
	}
}

// TestFolderSyncSkipsUnwalkableRoots pins the sweep's honest silence: a
// source root that cannot be walked decides nothing. An enqueued scan
// over half a listing would mark members vanished on the strength of
// an I/O error, and the manual rescan is the surface that reports the
// failure.
func TestFolderSyncSkipsUnwalkableRoots(t *testing.T) {
	requireExiftool(t)
	requireThumbnails(t)
	e := newEnv(t)

	source := filepath.Join(t.TempDir(), "family")
	if err := os.MkdirAll(source, 0o755); err != nil {
		t.Fatalf("create the source directory: %v", err)
	}
	writeSourceFile(t, filepath.Join(source, "a.jpg"), jpegBytes(t), time.Date(2026, 9, 15, 11, 0, 0, 0, time.UTC))

	album := e.createDirectoryAlbum(t, "Family archive", source)
	first := e.waitCreationScan(t, album)

	if err := os.RemoveAll(source); err != nil {
		t.Fatalf("remove the source root: %v", err)
	}
	counts := e.syncNow(t)
	if counts.Enqueued != 0 || counts.Skipped != 1 {
		t.Errorf("the sweep over an unwalkable root = %+v, want it skipped and nothing enqueued", counts)
	}
	var runs int
	if err := e.pool.QueryRow(context.Background(),
		"SELECT count(*) FROM job_runs WHERE album_id = $1 AND id <> $2",
		pgUUID(album), pgUUID(first)).Scan(&runs); err != nil {
		t.Fatalf("count the album's later runs: %v", err)
	}
	if runs != 0 {
		t.Errorf("the album gained %d runs from the unwalkable sweep, want none", runs)
	}
}

// TestFolderSyncUnknownBaselineConvergesOnce pins the store's restart
// semantic: a fresh process knows nothing about any album, and its
// first sweep of each album is a full convergence scan -- the folders
// may have changed while the server was down. The second sweep is
// idle, so the empty store costs exactly one scan per album, never a
// loop.
func TestFolderSyncUnknownBaselineConvergesOnce(t *testing.T) {
	requireExiftool(t)
	requireThumbnails(t)
	e := newEnv(t)

	source := filepath.Join(t.TempDir(), "family")
	if err := os.MkdirAll(source, 0o755); err != nil {
		t.Fatalf("create the source directory: %v", err)
	}
	writeSourceFile(t, filepath.Join(source, "a.jpg"), jpegBytes(t), time.Date(2026, 9, 16, 12, 0, 0, 0, time.UTC))

	album := e.createDirectoryAlbum(t, "Family archive", source)
	first := e.waitCreationScan(t, album)

	// The restart: the process's store knows nothing -- the same
	// state a fresh boot's empty store is in, produced the way a boot
	// empties it (nothing is live yet).
	e.queue.snapshots.Prune(nil)
	if known, _ := e.queue.snapshots.Same(album, nil); known {
		t.Fatal("the store still holds the album after the simulated restart")
	}
	counts := e.syncNow(t)
	if counts.Enqueued != 1 {
		t.Fatalf("the restarted store's first sweep enqueued %d scans, want the one convergence scan", counts.Enqueued)
	}
	e.waitNewScan(t, album, first)

	// The convergence scan re-baselined the album: the next sweep is
	// idle, so the empty store costs exactly one scan, never a loop.
	if counts := e.syncNow(t); counts.Enqueued != 0 || counts.Unchanged != 1 {
		t.Errorf("the restarted store's second sweep = %+v, want it idle after the convergence", counts)
	}
}

// TestFolderSyncPrunesRemovedAlbums pins the store's lifetime: an
// album the sweep no longer lists (the un-ingest) leaves the baseline
// store with it.
func TestFolderSyncPrunesRemovedAlbums(t *testing.T) {
	requireExiftool(t)
	requireThumbnails(t)
	e := newEnv(t)

	source := filepath.Join(t.TempDir(), "family")
	if err := os.MkdirAll(source, 0o755); err != nil {
		t.Fatalf("create the source directory: %v", err)
	}
	writeSourceFile(t, filepath.Join(source, "a.jpg"), jpegBytes(t), time.Date(2026, 9, 17, 13, 0, 0, 0, time.UTC))
	album := e.createDirectoryAlbum(t, "Family archive", source)
	e.waitCreationScan(t, album)
	e.syncNow(t)
	if known, _ := e.queue.snapshots.Same(album, nil); !known {
		t.Fatal("the sweep recorded no baseline for the album")
	}

	if _, err := e.queries.DeleteDirectoryAlbum(context.Background(), pgUUID(album)); err != nil {
		t.Fatalf("remove the album: %v", err)
	}
	if counts := e.syncNow(t); counts.Albums != 0 {
		t.Errorf("the sweep after the removal listed %d albums, want none", counts.Albums)
	}
	if known, _ := e.queue.snapshots.Same(album, nil); known {
		t.Error("the removed album's baseline survived the sweep, want it pruned with the row")
	}
}

// countRuns counts the album's scan runs.
func (e *env) countRuns(t *testing.T, albumID uuid.UUID) int {
	t.Helper()
	var count int
	if err := e.pool.QueryRow(context.Background(),
		"SELECT count(*) FROM job_runs WHERE album_id = $1", pgUUID(albumID)).Scan(&count); err != nil {
		t.Fatalf("count the album's runs: %v", err)
	}
	return count
}

// TestFolderSyncPeriodicRegistration pins the sweep's cadence
// machinery: the default window, the resolver the config feeds (zero
// = default, negative = off, positive = the deployment's own), the
// registration's identity and RunOnStart (a boot's first sweep is the
// down-time convergence), and the worker's own try-lock, which drops
// an overlapping tick instead of stacking a second sweep.
func TestFolderSyncPeriodicRegistration(t *testing.T) {
	if defaultFolderSyncInterval != time.Minute {
		t.Errorf("defaultFolderSyncInterval = %s, want the pinned 1m", defaultFolderSyncInterval)
	}
	now := time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)
	if got := folderSyncSchedule(defaultFolderSyncInterval).Next(now); !got.Equal(now.Add(time.Minute)) {
		t.Errorf("the sweep's schedule next(%s) = %s, want +1m", now, got)
	}
	for _, testCase := range []struct {
		configured time.Duration
		want       time.Duration
	}{
		{configured: 0, want: time.Minute},
		{configured: 3 * time.Second, want: 3 * time.Second},
		{configured: -1, want: -1},
	} {
		if got := folderSyncCadence(testCase.configured); got != testCase.want {
			t.Errorf("folderSyncCadence(%s) = %s, want %s", testCase.configured, got, testCase.want)
		}
	}

	if got := len(folderSyncPeriodicJobs(defaultFolderSyncInterval)); got != 1 {
		t.Errorf("the sweep's periodic registrations = %d, want exactly one", got)
	}
	args, opts := folderSyncJobConstructor()
	if _, ok := args.(FolderSyncArgs); !ok {
		t.Errorf("the sweep's periodic args = %T, want FolderSyncArgs", args)
	}
	if args.Kind() != "folder_sync" {
		t.Errorf("the sweep's job kind = %q, want folder_sync", args.Kind())
	}
	if opts != nil {
		t.Errorf("the sweep's insert opts = %+v, want the defaults (nil)", opts)
	}

	// The try-lock: a tick overlapping a running sweep is dropped
	// whole -- the fake seam proves nothing was decided.
	decision := 0
	worker := &FolderSyncWorker{
		snapshots: NewAlbumSnapshots(),
		enqueue: func(ctx context.Context, albumID uuid.UUID) (uuid.UUID, error) {
			decision++
			return uuid.Nil, nil
		},
		logger: slog.New(slog.NewTextHandler(os.Stderr, nil)),
	}
	worker.tick.Lock()
	if err := worker.Work(context.Background(), nil); err != nil {
		t.Fatalf("the overlapping tick returned %v, want a clean skip", err)
	}
	worker.tick.Unlock()
	if decision != 0 {
		t.Errorf("the overlapping tick decided %d scans, want it dropped whole", decision)
	}
}
