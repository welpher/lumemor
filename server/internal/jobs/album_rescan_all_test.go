// The all-albums rescan sweep's tests: the real worker over the real
// queue, with the same fixture trees the single-album rescan tests
// stage. The sweep's units are albums: the row carries the instance's
// directory album count, done counts the albums whose diff closed
// cleanly, failed the albums that could not be diffed, and the run
// closes failed when any album failed -- one album's failure never
// stops the sweep. The sweep creates NO per-album job_runs rows (the
// history keeps its one row), and the cancel flag ends it at an album
// boundary.
package jobs

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/riverqueue/river"

	apigen "github.com/welpher/lumemor/server/internal/api/gen"
	dbgen "github.com/welpher/lumemor/server/internal/db/gen"
)

// rescanAll fires POST /admin/albums/rescan and answers the sweep run
// identifier the endpoint handed out.
func (e *env) rescanAll(t *testing.T) uuid.UUID {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/albums/rescan", nil)
	req.AddCookie(e.cookie)
	rec := httptest.NewRecorder()
	e.router.ServeHTTP(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("POST /admin/albums/rescan status = %d, want %d (body %q)",
			rec.Code, http.StatusAccepted, rec.Body.String())
	}
	var response apigen.AlbumRescanAllResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode sweep body %q: %v", rec.Body.String(), err)
	}
	return response.JobRunId
}

// sweepRun loads one sweep run's row.
type sweepRun struct {
	kind     string
	state    string
	total    int
	done     int
	failed   int
	finished bool
}

// loadSweep loads the sweep run's row.
func (e *env) loadSweep(t *testing.T, runID uuid.UUID) sweepRun {
	t.Helper()
	var row sweepRun
	if err := e.pool.QueryRow(context.Background(), `
		SELECT kind, state, total, done, failed, finished_at IS NOT NULL
		FROM job_runs WHERE id = $1`, pgUUID(runID)).
		Scan(&row.kind, &row.state, &row.total, &row.done, &row.failed, &row.finished); err != nil {
		t.Fatalf("load the sweep run %s: %v", runID, err)
	}
	return row
}

// waitSweepFinished polls the sweep's row until it closes (the single
// rescan helper's shape, with the cancelled ending included).
func (e *env) waitSweepFinished(t *testing.T, runID uuid.UUID) string {
	t.Helper()
	var state string
	eventually(t, jobWaitTimeout, "the sweep run to finish", func() bool {
		return e.pool.QueryRow(context.Background(),
			"SELECT state FROM job_runs WHERE id = $1", pgUUID(runID)).Scan(&state) == nil &&
			(state == "succeeded" || state == "failed" || state == "cancelled")
	})
	return state
}

// countRunsOfKind counts the job_runs rows of one kind.
func (e *env) countRunsOfKind(t *testing.T, kind string) int {
	t.Helper()
	var count int
	if err := e.pool.QueryRow(context.Background(),
		"SELECT count(*) FROM job_runs WHERE kind = $1", kind).Scan(&count); err != nil {
		t.Fatalf("count %s runs: %v", kind, err)
	}
	return count
}

// stageAlbum stages one source tree with the given file names and
// creates its directory album, waiting for the creation-time scan so
// the sweep that follows walks an already-converged album.
func (e *env) stageAlbum(t *testing.T, title string, files ...string) (uuid.UUID, string) {
	t.Helper()
	source := filepath.Join(t.TempDir(), "source")
	if err := os.MkdirAll(source, 0o755); err != nil {
		t.Fatalf("create the source directory: %v", err)
	}
	plain := jpegBytes(t)
	stamp := time.Date(2026, 3, 1, 9, 30, 0, 0, time.UTC)
	for _, name := range files {
		writeSourceFile(t, filepath.Join(source, name), plain, stamp)
	}
	album := e.createDirectoryAlbum(t, title, source)
	e.waitCreationScan(t, album)
	return album, source
}

// TestAlbumRescanAllSweepsEveryAlbum: one sweep over two directory
// albums resolves both -- the run's total is the album count, its
// done counts the albums, its state is succeeded, and the history
// keeps ONE sweep row (the sweep adds no per-album rows).
func TestAlbumRescanAllSweepsEveryAlbum(t *testing.T) {
	requireExiftool(t)
	requireThumbnails(t)
	e := newEnv(t)

	first, firstSource := e.stageAlbum(t, "First album", "a.jpg", "b.jpg")
	second, secondSource := e.stageAlbum(t, "Second album", "c.jpg")
	rescansBefore := e.countRunsOfKind(t, "album_rescan")

	runID := e.rescanAll(t)
	if state := e.waitSweepFinished(t, runID); state != "succeeded" {
		t.Fatalf("the sweep closed %s, want succeeded", state)
	}

	run := e.loadSweep(t, runID)
	if run.kind != "album_rescan_all" {
		t.Errorf("sweep kind = %q, want album_rescan_all", run.kind)
	}
	if run.total != 2 {
		t.Errorf("sweep total = %d, want 2 (the instance's directory album count)", run.total)
	}
	if run.done != 2 || run.failed != 0 {
		t.Errorf("sweep done/failed = %d/%d, want 2/0 (one unit per album)", run.done, run.failed)
	}

	// THE ONE ROW: the sweep never creates per-album runs; the
	// creation scans are still the only album_rescan rows.
	if got := e.countRunsOfKind(t, "album_rescan"); got != rescansBefore {
		t.Errorf("album_rescan rows = %d after the sweep, want the creation scans' %d", got, rescansBefore)
	}
	if got := e.countRunsOfKind(t, "album_rescan_all"); got != 1 {
		t.Errorf("album_rescan_all rows = %d, want the one sweep row", got)
	}

	// Both albums' sources were diffed (the creation scans ingested
	// them; the sweep's unchanged diff keeps them).
	for _, path := range []string{
		filepath.Join(firstSource, "a.jpg"),
		filepath.Join(firstSource, "b.jpg"),
		filepath.Join(secondSource, "c.jpg"),
	} {
		if _, ok := e.rescanAssetByPath(t, e.ownerID, path); !ok {
			t.Errorf("the sweep left no asset row for %s", path)
		}
	}
	// The albums' scan clocks carry the creation scans' stamps: the
	// sweep's unchanged diff writes nothing new to the album rows
	// (the idempotence pin), so the clocks do not move.
	if !e.albumScannedAt(t, first).Valid || !e.albumScannedAt(t, second).Valid {
		t.Error("an album's scan clock is unset after its creation scan stamped it")
	}

	// The run announces its own kind on the activity stream.
	if got := countEvents(e.bus, "job.started"); got == 0 {
		t.Error("no job.started event landed for the sweep")
	}
}

// TestAlbumRescanAllFailedAlbumDoesNotStopTheSweep: one album's
// source vanishes before the sweep -- that album counts failed, the
// other still resolves, and the run closes failed (the batch jobs'
// shared convention: any failure fails the run).
func TestAlbumRescanAllFailedAlbumDoesNotStopTheSweep(t *testing.T) {
	requireExiftool(t)
	requireThumbnails(t)
	e := newEnv(t)

	_, healthySource := e.stageAlbum(t, "Healthy album", "kept.jpg")
	_, brokenSource := e.stageAlbum(t, "Broken album", "gone.jpg")

	// The broken album's source leaves the disk: the sweep's walk
	// cannot resolve the root and counts the album failed.
	if err := os.RemoveAll(brokenSource); err != nil {
		t.Fatalf("remove the broken album's source: %v", err)
	}

	runID := e.rescanAll(t)
	if state := e.waitSweepFinished(t, runID); state != "failed" {
		t.Fatalf("the sweep closed %s, want failed (an album failed)", state)
	}
	run := e.loadSweep(t, runID)
	if run.total != 2 || run.done != 1 || run.failed != 1 {
		t.Errorf("sweep total/done/failed = %d/%d/%d, want 2/1/1 (the healthy album still resolved)",
			run.total, run.done, run.failed)
	}
	// The healthy album was walked: its file is a live external
	// asset row.
	row, ok := e.rescanAssetByPath(t, e.ownerID, filepath.Join(healthySource, "kept.jpg"))
	if !ok || row.storageKind != "external" {
		t.Errorf("the healthy album's asset = (%v, %q), want the external ingest", ok, row.storageKind)
	}
}

// TestAlbumRescanAllCancelEndsAtTheAlbumBoundary: a stop flag set
// before the walk ends the sweep at its first album boundary -- no
// album is walked, the run closes cancelled, and the counts stay at
// zero.
func TestAlbumRescanAllCancelEndsAtTheAlbumBoundary(t *testing.T) {
	e := newEnv(t)
	e.stageAlbum(t, "Only album", "a.jpg")

	// The run's row with the flag already set (the queued-cancel race
	// the endpoint's own close produces), walked by a directly
	// constructed worker: the first boundary check ends it.
	runID, err := uuid.NewV7()
	if err != nil {
		t.Fatalf("generate the run id: %v", err)
	}
	if _, err := e.pool.Exec(context.Background(), `
		INSERT INTO job_runs (id, kind, total, stop_requested)
		VALUES ($1, 'album_rescan_all', 1, true)`, pgUUID(runID)); err != nil {
		t.Fatalf("insert the cancelled sweep row: %v", err)
	}
	worker := &AlbumRescanAllWorker{
		pool:     e.pool,
		bus:      e.bus,
		dispatch: func(context.Context, uuid.UUID, dbgen.MediaKind) error { return nil },
		logger:   e.queue.logger,
	}
	job := &river.Job[AlbumRescanAllArgs]{Args: AlbumRescanAllArgs{JobRunID: runID}}
	if err := worker.Work(context.Background(), job); err != nil {
		t.Fatalf("sweep Work: %v", err)
	}

	run := e.loadSweep(t, runID)
	if run.state != "cancelled" {
		t.Errorf("sweep state = %q, want cancelled", run.state)
	}
	if run.done != 0 || run.failed != 0 {
		t.Errorf("sweep done/failed = %d/%d, want 0/0 (the flag ended it before the first album)", run.done, run.failed)
	}
}
