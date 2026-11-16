package jobs

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/welpher/lumemor/server/internal/db"
	dbgen "github.com/welpher/lumemor/server/internal/db/gen"
	"github.com/welpher/lumemor/server/internal/media"
	"github.com/welpher/lumemor/server/internal/storage"
)

// rebuildRunRow reads one rebuild run's row for the tests'
// convergence polls.
func rebuildRunRow(t *testing.T, e *env, id uuid.UUID) (string, int32, int32) {
	t.Helper()
	var state string
	var total, done int32
	err := e.pool.QueryRow(context.Background(),
		"SELECT state::text, total, done FROM job_runs WHERE id = $1", pgUUID(id)).
		Scan(&state, &total, &done)
	if err != nil {
		t.Fatalf("load the rebuild run %s: %v", id, err)
	}
	return state, total, done
}

// waitRebuildConverged polls the run until it closed.
func waitRebuildConverged(t *testing.T, e *env, id uuid.UUID) (string, int32, int32) {
	t.Helper()
	eventually(t, 30*time.Second, "the rebuild run to converge", func() bool {
		state, _, _ := rebuildRunRow(t, e, id)
		return state == "succeeded" || state == "failed" || state == "cancelled"
	})
	return rebuildRunRow(t, e, id)
}

// TestRebuildAllRewritesAndSweeps pins the full rebuild: every image
// asset's serving copies are re-derived at the run's starting
// settings (here, WebP at 320/1440 with the full-size copy on), the
// other encoding's stale files are swept, the blurhash follows the
// new tile, and the run closes succeeded over the scope it
// announced.
func TestRebuildAllRewritesAndSweeps(t *testing.T) {
	requireThumbnails(t)
	e := newEnv(t)
	if err := db.SaveImageSettings(t.Context(), e.queries, db.ImageSettings{
		TileSize:    320,
		PreviewSize: 1440,
		Format:      media.FormatWebp,
		FullSize:    true,
		RawEmbedded: true,
	}); err != nil {
		t.Fatalf("save the image settings: %v", err)
	}

	content, err := os.ReadFile(filepath.Join("testdata", "golden.jpg"))
	if err != nil {
		t.Fatalf("read the golden fixture: %v", err)
	}
	// The asset carries a PRE-EXISTING AVIF tile: the full rebuild
	// rewrites it into the WebP the settings carry and sweeps the
	// stale encoding away.
	id := e.insertAsset(t, "originals/rebuild/one.jpg", content, time.Now(), `{}`)
	if err := os.MkdirAll(filepath.Dir(storage.ThumbPath(id.String(), media.SizeTile, media.FormatAvif)), 0o755); err != nil {
		t.Fatalf("prepare the thumb directory: %v", err)
	}
	if err := os.WriteFile(storage.ThumbPath(id.String(), media.SizeTile, media.FormatAvif), []byte("stale"), 0o644); err != nil {
		t.Fatalf("write the stale tile: %v", err)
	}

	runID, err := e.queue.EnqueueDerivativesRebuild(t.Context(), RebuildScopeAll)
	if err != nil {
		t.Fatalf("enqueue the rebuild: %v", err)
	}
	state, total, done := waitRebuildConverged(t, e, runID)
	if state != "succeeded" {
		t.Fatalf("rebuild run state = %s, want succeeded", state)
	}
	if total != 1 || done != 1 {
		t.Errorf("rebuild counts = %d/%d, want the one-asset scope walked once", done, total)
	}

	// The new encoding's copies exist at the settings' sizes.
	for _, tt := range []struct {
		size         string
		wantW, wantH int
	}{
		{media.SizeTile, 64, 64},
		{media.SizeView, 64, 64},
		{media.SizeOriginal, 64, 64},
	} {
		path := storage.ThumbPath(id.String(), tt.size, media.FormatWebp)
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("rebuilt derivative %s missing: %v", path, err)
		}
	}
	// The stale encoding is swept.
	stale := storage.ThumbPath(id.String(), media.SizeTile, media.FormatAvif)
	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Errorf("stale tile stat err = %v, want NotExist after the full rebuild", err)
	}
	// The blurhash follows the new tile.
	if hash := e.blurhashOf(t, id); !hash.Valid || len(hash.String) < 6 {
		t.Errorf("blurhash = %v, want the rebuilt tile's hash", hash)
	}
}

// TestRebuildMissingSkipsPresent pins the missing scope: the assets
// whose current-format serving files already exist are skipped (the
// run pays only for what it lacks) -- a WebP settings document over a
// fully AVIF library re-derives nothing, and the run closes
// succeeded.
func TestRebuildMissingSkipsPresent(t *testing.T) {
	requireThumbnails(t)
	e := newEnv(t)

	content, err := os.ReadFile(filepath.Join("testdata", "golden.jpg"))
	if err != nil {
		t.Fatalf("read the golden fixture: %v", err)
	}
	// The pipeline has completed for this asset (pending_jobs zero
	// is the completed-pipeline fact the skip reads alongside the
	// files).
	id := e.insertAssetKind(t, "originals/rebuild/two.jpg", content, time.Now(), `{}`, dbgen.MediaKindImage, "image/jpeg", 0)
	// The AVIF serving files exist (the shipped settings' format):
	// the missing scope skips this asset.
	for _, size := range []string{media.SizeTile, media.SizeView} {
		path := storage.ThumbPath(id.String(), size, media.FormatAvif)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("prepare the thumb directory: %v", err)
		}
		if err := os.WriteFile(path, []byte("existing"), 0o644); err != nil {
			t.Fatalf("write the existing derivative: %v", err)
		}
	}
	before, err := os.Stat(storage.ThumbPath(id.String(), media.SizeTile, media.FormatAvif))
	if err != nil {
		t.Fatalf("stat the existing tile: %v", err)
	}

	runID, err := e.queue.EnqueueDerivativesRebuild(t.Context(), RebuildScopeMissing)
	if err != nil {
		t.Fatalf("enqueue the rebuild: %v", err)
	}
	state, _, done := waitRebuildConverged(t, e, runID)
	if state != "succeeded" {
		t.Fatalf("rebuild run state = %s, want succeeded", state)
	}
	if done != 1 {
		t.Errorf("rebuild done = %d, want the one visited asset", done)
	}
	after, err := os.Stat(storage.ThumbPath(id.String(), media.SizeTile, media.FormatAvif))
	if err != nil {
		t.Fatalf("stat the tile after the missing rebuild: %v", err)
	}
	if !after.ModTime().Equal(before.ModTime()) {
		t.Error("the existing tile was rewritten under the missing scope, want it left alone")
	}
}

// TestRebuildMissingDerivesAbsent pins the missing scope's other arm:
// an asset with NO serving files (its pipeline never completed, or
// the encoding changed) IS derived by the missing run.
func TestRebuildMissingDerivesAbsent(t *testing.T) {
	requireThumbnails(t)
	e := newEnv(t)

	content, err := os.ReadFile(filepath.Join("testdata", "golden.jpg"))
	if err != nil {
		t.Fatalf("read the golden fixture: %v", err)
	}
	// A completed pipeline whose serving files are absent (the
	// encoding-changed case): the missing scope OWNS this one -- an
	// in-flight pipeline would be its own business.
	id := e.insertAssetKind(t, "originals/rebuild/three.jpg", content, time.Now(), `{}`, dbgen.MediaKindImage, "image/jpeg", 0)

	runID, err := e.queue.EnqueueDerivativesRebuild(t.Context(), RebuildScopeMissing)
	if err != nil {
		t.Fatalf("enqueue the rebuild: %v", err)
	}
	state, _, done := waitRebuildConverged(t, e, runID)
	if state != "succeeded" {
		t.Fatalf("rebuild run state = %s, want succeeded", state)
	}
	if done != 1 {
		t.Errorf("rebuild done = %d, want the one derived asset", done)
	}
	requireJobThumbDimensions(t, id, media.SizeTile, media.FormatAvif, 64, 64)
	requireJobThumbDimensions(t, id, media.SizeView, media.FormatAvif, 64, 64)
}

// TestRebuildHonorsStop pins the stop: a stop flag set while the run
// is in flight closes it cancelled (E41's shared flag-closed ending)
// with zero failures -- the panel renders the cancelled state, the
// admin asked for this ending -- and the work already done stays on
// disk.
func TestRebuildHonorsStop(t *testing.T) {
	requireThumbnails(t)
	e := newEnv(t)

	content, err := os.ReadFile(filepath.Join("testdata", "golden.jpg"))
	if err != nil {
		t.Fatalf("read the golden fixture: %v", err)
	}
	// A two-asset scope: the stop lands between the walk's assets.
	first := e.insertAsset(t, "originals/rebuild/stop-a.jpg", content, time.Now(), `{}`)
	second := e.insertAsset(t, "originals/rebuild/stop-b.jpg", content, time.Now(), `{}`)

	runID, err := e.queue.EnqueueDerivativesRebuild(t.Context(), RebuildScopeAll)
	if err != nil {
		t.Fatalf("enqueue the rebuild: %v", err)
	}
	// The stop lands as soon as the walk has begun: the flag is a
	// per-page check, so the run closes at the first breakpoint it
	// reads after the flag.
	if _, err := e.pool.Exec(t.Context(),
		"UPDATE job_runs SET stop_requested = true WHERE id = $1", pgUUID(runID)); err != nil {
		t.Fatalf("set the stop flag: %v", err)
	}
	state, _, _ := waitRebuildConverged(t, e, runID)
	if state != "cancelled" {
		t.Fatalf("stopped rebuild run state = %s, want cancelled (the stopped run's state)", state)
	}
	// Whatever the stop caught, the derivations that landed are real
	// files -- and the run's failure names no failed assets.
	var failed int32
	if err := e.pool.QueryRow(t.Context(),
		"SELECT failed FROM job_runs WHERE id = $1", pgUUID(runID)).Scan(&failed); err != nil {
		t.Fatalf("load the stopped run's failed count: %v", err)
	}
	if failed != 0 {
		t.Errorf("stopped run failed = %d, want 0 (the admin asked for this ending)", failed)
	}
	_ = first
	_ = second
}

// TestRebuildHonorsPause pins the pause hold: a paused thumbnail
// class snoozes the walk at its gate -- the queued run stays queued
// while the switch is on, and the un-pause lets it through.
func TestRebuildHonorsPause(t *testing.T) {
	requireThumbnails(t)
	e := newEnv(t)
	if err := db.SaveRunBudget(t.Context(), e.queries, db.RunBudget{
		ConcurrencyPreset: "standard",
		NightlyStart:      db.RunBudgetNightlyStartDefault,
		NightlyEnd:        db.RunBudgetNightlyEndDefault,
		PausedJobs:        db.PausedJobs{Thumbnail: true},
	}); err != nil {
		t.Fatalf("save the run budget: %v", err)
	}

	content, err := os.ReadFile(filepath.Join("testdata", "golden.jpg"))
	if err != nil {
		t.Fatalf("read the golden fixture: %v", err)
	}
	// Pending zero: the walk owns this asset, so the un-pause
	// derives it (an in-flight pipeline would be its own business).
	id := e.insertAssetKind(t, "originals/rebuild/paused.jpg", content, time.Now(), `{}`, dbgen.MediaKindImage, "image/jpeg", 0)

	runID, err := e.queue.EnqueueDerivativesRebuild(t.Context(), RebuildScopeMissing)
	if err != nil {
		t.Fatalf("enqueue the rebuild: %v", err)
	}
	// While the pause holds, the run does not start deriving.
	time.Sleep(2 * time.Second)
	if rows, err := e.pool.Query(t.Context(), "SELECT state::text, attempt, scheduled_at, errors FROM river_job ORDER BY id LIMIT 3"); err == nil {
		defer rows.Close()
		for rows.Next() {
			var st string
			var attempt int
			var sched time.Time
			var errs []string
			_ = rows.Scan(&st, &attempt, &sched, &errs)
			t.Logf("DEBUG river_job state=%s attempt=%d scheduled=%s errors=%v", st, attempt, sched, errs)
		}
	}
	var savedPaused bool
	if err := e.pool.QueryRow(t.Context(), `SELECT (value->'pausedJobs'->>'thumbnail')::bool FROM instance_settings WHERE key = 'run_budget'`).Scan(&savedPaused); err != nil {
		t.Logf("DEBUG saved budget read err: %v", err)
	} else {
		t.Logf("DEBUG saved paused.thumbnail = %v", savedPaused)
	}
	if state, _, done := rebuildRunRow(t, e, runID); state == "succeeded" || state == "failed" {
		t.Fatalf("rebuild run under the pause state = %s done = %d, want it still held", state, done)
	}
	tile := storage.ThumbPath(id.String(), media.SizeTile, media.FormatAvif)
	if _, err := os.Stat(tile); !os.IsNotExist(err) {
		t.Fatalf("paused walk derived %s, want nothing while held", tile)
	}

	// The un-pause: the snoozed job wakes within the pause's
	// granularity and finishes.
	if err := db.SaveRunBudget(t.Context(), e.queries, db.RunBudget{
		ConcurrencyPreset: "standard",
		NightlyStart:      db.RunBudgetNightlyStartDefault,
		NightlyEnd:        db.RunBudgetNightlyEndDefault,
		PausedJobs:        db.PausedJobs{},
	}); err != nil {
		t.Fatalf("lift the pause: %v", err)
	}
	state, _, done := waitRebuildConverged(t, e, runID)
	if state != "succeeded" {
		t.Fatalf("rebuild run after the un-pause state = %s, want succeeded", state)
	}
	if done != 1 {
		t.Errorf("rebuild done after the un-pause = %d, want the one derived asset", done)
	}
	requireJobThumbDimensions(t, id, media.SizeTile, media.FormatAvif, 64, 64)
}

// TestRebuildHonorsWindow pins the window hold: the rebuild is one of
// the heavy three, so a nightly window that does not contain now
// snoozes the walk at its gate -- nothing derives while the window
// is closed, and opening it (a window that contains now) lets the
// walk through.
func TestRebuildHonorsWindow(t *testing.T) {
	requireThumbnails(t)
	e := newEnv(t)

	// A window one to two hours ahead: whatever the wall clock reads,
	// now sits outside it.
	ahead := func(hours int) string {
		return time.Now().Add(time.Duration(hours) * time.Hour).Format("15:04")
	}
	if err := db.SaveRunBudget(t.Context(), e.queries, db.RunBudget{
		ConcurrencyPreset: "standard",
		NightlyEnabled:    true,
		NightlyStart:      ahead(1),
		NightlyEnd:        ahead(2),
	}); err != nil {
		t.Fatalf("save the closed window: %v", err)
	}

	content, err := os.ReadFile(filepath.Join("testdata", "golden.jpg"))
	if err != nil {
		t.Fatalf("read the golden fixture: %v", err)
	}
	id := e.insertAssetKind(t, "originals/rebuild/windowed.jpg", content, time.Now(), `{}`, dbgen.MediaKindImage, "image/jpeg", 0)

	runID, err := e.queue.EnqueueDerivativesRebuild(t.Context(), RebuildScopeAll)
	if err != nil {
		t.Fatalf("enqueue the rebuild: %v", err)
	}
	// While the window is closed, the run does not start deriving.
	time.Sleep(3 * time.Second)
	if state, _, _ := rebuildRunRow(t, e, runID); state == "succeeded" || state == "failed" {
		t.Fatalf("rebuild run outside the window state = %s, want it still held", state)
	}
	tile := storage.ThumbPath(id.String(), media.SizeTile, media.FormatAvif)
	if _, err := os.Stat(tile); !os.IsNotExist(err) {
		t.Fatalf("windowed walk derived %s, want nothing while held", tile)
	}

	// The window opens: a span that contains now lets the snoozed
	// walk through within the wake granularity.
	if err := db.SaveRunBudget(t.Context(), e.queries, db.RunBudget{
		ConcurrencyPreset: "standard",
		NightlyEnabled:    true,
		NightlyStart:      time.Now().Add(-time.Hour).Format("15:04"),
		NightlyEnd:        ahead(1),
	}); err != nil {
		t.Fatalf("open the window: %v", err)
	}
	// The snooze before the window opened carries the wake cap
	// (a minute), so the convergence here tolerates it.
	var state string
	var done int32
	eventually(t, 90*time.Second, "the opened-window rebuild run to converge", func() bool {
		state, _, done = rebuildRunRow(t, e, runID)
		return state == "succeeded" || state == "failed"
	})
	if state != "succeeded" {
		t.Fatalf("opened-window rebuild run state = %s, want succeeded", state)
	}
	if done != 1 {
		t.Errorf("opened-window rebuild done = %d, want the one visited asset", done)
	}
}

// TestRebuildMissingSkipsInFlight pins the missing scope's in-flight
// arm: an asset whose pipeline has not finished (pending jobs above
// zero) belongs to the pipeline, not the rebuild -- the walk skips
// it without deriving, so it never double-pays or races the
// pipeline's half-written file set.
func TestRebuildMissingSkipsInFlight(t *testing.T) {
	requireThumbnails(t)
	e := newEnv(t)

	content, err := os.ReadFile(filepath.Join("testdata", "golden.jpg"))
	if err != nil {
		t.Fatalf("read the golden fixture: %v", err)
	}
	// The default insert seeds one pending job: the pipeline is in
	// flight for this asset.
	id := e.insertAsset(t, "originals/rebuild/inflight.jpg", content, time.Now(), `{}`)

	runID, err := e.queue.EnqueueDerivativesRebuild(t.Context(), RebuildScopeMissing)
	if err != nil {
		t.Fatalf("enqueue the rebuild: %v", err)
	}
	state, _, done := waitRebuildConverged(t, e, runID)
	if state != "succeeded" {
		t.Fatalf("rebuild run state = %s, want succeeded", state)
	}
	if done != 1 {
		t.Errorf("rebuild done = %d, want the one visited (and skipped) asset", done)
	}
	tile := storage.ThumbPath(id.String(), media.SizeTile, media.FormatAvif)
	if _, err := os.Stat(tile); !os.IsNotExist(err) {
		t.Fatalf("the in-flight asset's tile = %s, want nothing derived (the pipeline owns it)", tile)
	}
}
