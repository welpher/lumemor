// Sweep tests (M2c Task 1): the requeue sweep is the recovery path of
// assets whose upload committed but whose pipeline jobs never entered
// the queue -- the enqueue insert failure and the crash in the
// commit-to-enqueue window that M2b's final review left as the
// hard-gate handoff. The contracts pinned here are the mandated ones:
// a stuck row is recovered end to end (jobs re-enqueued, queue
// drained, asset.ready exactly once), a clean database is a no-op,
// soft-deleted stuck rows are skipped, a double sweep is idempotent
// (no duplicate jobs, no second event), and the hourly periodic
// registration exists -- pinned through its schedule config, never by
// waiting a wall-clock hour.
package jobs

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	dbgen "github.com/welpher/lumemor/server/internal/db/gen"
)

// riverJobsOf counts River's jobs of one asset by kind, across every
// state: the sweep's enqueue assertions read the queue's own table.
// Completed rows stay in river_job until the job cleaner removes them
// (24h retention by default), so the count of a test's own jobs is
// stable for the test's lifetime -- exactly what the sweep's
// duplication assertions need.
func (e *env) riverJobsOf(t *testing.T, id uuid.UUID) map[string]int {
	t.Helper()
	kinds := map[string]int{}
	rows, err := e.pool.Query(context.Background(),
		`SELECT kind FROM river_job WHERE args->>'assetId' = $1`, id.String())
	if err != nil {
		t.Fatalf("query river jobs of asset %s: %v", id, err)
	}
	defer rows.Close()
	for rows.Next() {
		var kind string
		if err := rows.Scan(&kind); err != nil {
			t.Fatalf("scan river job kind: %v", err)
		}
		kinds[kind]++
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate river jobs: %v", err)
	}
	return kinds
}

// riverJobTotal sums the asset's River jobs across kinds.
func (e *env) riverJobTotal(t *testing.T, id uuid.UUID) int {
	t.Helper()
	total := 0
	for _, count := range e.riverJobsOf(t, id) {
		total += count
	}
	return total
}

// softDeleteAsset marks the asset trashed directly on the row: the
// trash endpoint arrives with this milestone's later tasks, and the
// sweep's skip contract is a property of the column, not the endpoint.
func (e *env) softDeleteAsset(t *testing.T, id uuid.UUID) {
	t.Helper()
	if _, err := e.pool.Exec(context.Background(),
		`UPDATE assets SET deleted_at = now() WHERE id = $1`, id); err != nil {
		t.Fatalf("soft delete asset %s: %v", id, err)
	}
}

// TestRequeueSweepRecoversStuckAsset is the mandated recovery flow:
// a row the uploader committed with pending_jobs = 2 but whose jobs
// never entered the queue (the crash-in-the-commit-to-enqueue-window
// shape) is found by the sweep, both pipeline jobs of its kind are
// re-enqueued, the queue drains them, and the row reaches ready_at
// with asset.ready fired exactly once.
func TestRequeueSweepRecoversStuckAsset(t *testing.T) {
	requireExiftool(t)
	requireThumbnails(t)
	e := newEnv(t)

	deviceTakenAt := time.Date(2021, 8, 9, 10, 11, 12, 0, time.UTC).Truncate(time.Microsecond)
	id := e.insertAssetKind(t, "originals/sweep/stuck.jpg", jpegBytes(t), deviceTakenAt,
		`{"deviceCreatedAt":"2021-08-09T10:11:12Z","deviceTimeZone":"Asia/Shanghai"}`,
		dbgen.MediaKindImage, "image/jpeg", 2)
	if got := e.riverJobTotal(t, id); got != 0 {
		t.Fatalf("river jobs before the sweep = %d, want 0 (the stuck precondition: no jobs were ever inserted)", got)
	}

	counts, err := RunRequeueSweep(context.Background(), e.pool, e.queue)
	if err != nil {
		t.Fatalf("run requeue sweep: %v", err)
	}
	if counts.Stuck != 1 || counts.Enqueued != 2 {
		t.Errorf("sweep counts = stuck %d enqueued %d, want stuck 1 enqueued 2", counts.Stuck, counts.Enqueued)
	}

	// Both jobs of the row's kind entered the queue, exactly once
	// each.
	kinds := e.riverJobsOf(t, id)
	if kinds["metadata"] != 1 || kinds["thumbnail"] != 1 {
		t.Errorf("river jobs = %v, want exactly one metadata and one thumbnail job", kinds)
	}

	// The counter was never reset: the two re-enqueued completions
	// drive it from 2 to 0, and the completion that empties it stamps
	// ready_at.
	eventually(t, jobWaitTimeout, "the swept asset to become ready", func() bool {
		return e.assetOf(t, id).readyAt.Valid
	})
	state := e.assetOf(t, id)
	if state.pending != 0 {
		t.Errorf("pending_jobs = %d, want 0 after the drained recovery", state.pending)
	}
	if !state.readyAt.Valid {
		t.Fatal("the swept asset never became ready")
	}
	if got := countEvents(e.bus, "asset.ready"); got != 1 {
		t.Errorf("asset.ready events = %d, want exactly 1", got)
	}
	payload := eventPayloadOf(t, e.bus, "asset.ready")
	if payload["assetId"] != id.String() {
		t.Errorf("asset.ready assetId = %q, want %s", payload["assetId"], id)
	}
	if payload["kind"] != "image" {
		t.Errorf("asset.ready kind = %q, want image", payload["kind"])
	}
}

// TestRequeueSweepCleanDatabaseIsNoop pins the no-op contract: a
// pipeline with no stuck rows produces no enqueues, no queue rows,
// and no error.
func TestRequeueSweepCleanDatabaseIsNoop(t *testing.T) {
	e := newEnv(t)

	counts, err := RunRequeueSweep(context.Background(), e.pool, e.queue)
	if err != nil {
		t.Fatalf("run requeue sweep: %v", err)
	}
	if counts.Stuck != 0 || counts.Enqueued != 0 {
		t.Errorf("sweep counts = stuck %d enqueued %d, want the zero no-op", counts.Stuck, counts.Enqueued)
	}
	var jobs int
	if err := e.pool.QueryRow(context.Background(),
		`SELECT count(*) FROM river_job`).Scan(&jobs); err != nil {
		t.Fatalf("count river jobs: %v", err)
	}
	if jobs != 0 {
		t.Errorf("river_job rows = %d, want 0 (the sweep enqueued nothing)", jobs)
	}
}

// TestRequeueSweepSkipsSoftDeleted pins the stuck-row definition's
// deleted_at arm: a trashed asset is headed for expiry, not recovery
// -- the sweep must not requeue jobs for it.
func TestRequeueSweepSkipsSoftDeleted(t *testing.T) {
	e := newEnv(t)

	deviceTakenAt := time.Date(2021, 8, 9, 10, 11, 12, 0, time.UTC).Truncate(time.Microsecond)
	id := e.insertAssetKind(t, "originals/sweep/trashed.jpg", jpegBytes(t), deviceTakenAt,
		`{}`, dbgen.MediaKindImage, "image/jpeg", 2)
	e.softDeleteAsset(t, id)

	counts, err := RunRequeueSweep(context.Background(), e.pool, e.queue)
	if err != nil {
		t.Fatalf("run requeue sweep: %v", err)
	}
	if counts.Stuck != 0 || counts.Enqueued != 0 {
		t.Errorf("sweep counts = stuck %d enqueued %d, want the soft-deleted row skipped", counts.Stuck, counts.Enqueued)
	}
	if got := e.riverJobTotal(t, id); got != 0 {
		t.Errorf("river jobs of the soft-deleted asset = %d, want 0", got)
	}
	state := e.assetOf(t, id)
	if state.pending != 2 {
		t.Errorf("pending_jobs = %d, want the untouched 2", state.pending)
	}
	if state.readyAt.Valid {
		t.Error("ready_at became set for the soft-deleted asset, want it left alone")
	}
}

// TestRequeueSweepTwiceIsIdempotent pins the double-run contract over
// the same stuck row: the second sweep may find the row again (the
// counter is deliberately not reset, so the row stays stuck until its
// jobs complete) but inserts NO duplicate jobs -- River's unique opts
// over kind + asset id dedupe while the jobs are queued or completed
// -- and the drained pipeline still fires asset.ready exactly once
// (the guarded decrement turns extra completions into no-ops).
func TestRequeueSweepTwiceIsIdempotent(t *testing.T) {
	requireExiftool(t)
	requireThumbnails(t)
	e := newEnv(t)

	deviceTakenAt := time.Date(2021, 8, 9, 10, 11, 12, 0, time.UTC).Truncate(time.Microsecond)
	id := e.insertAssetKind(t, "originals/sweep/twice.jpg", jpegBytes(t), deviceTakenAt,
		`{}`, dbgen.MediaKindImage, "image/jpeg", 2)

	first, err := RunRequeueSweep(context.Background(), e.pool, e.queue)
	if err != nil {
		t.Fatalf("first sweep: %v", err)
	}
	if first.Stuck != 1 || first.Enqueued != 2 {
		t.Fatalf("first sweep counts = stuck %d enqueued %d, want stuck 1 enqueued 2", first.Stuck, first.Enqueued)
	}

	// The immediate second sweep over the same row. Whether the first
	// sweep's jobs are still queued (available, running, retryable)
	// or already completed (retained in river_job), every insert
	// dedupes: nothing new enters the queue and no error surfaces.
	second, err := RunRequeueSweep(context.Background(), e.pool, e.queue)
	if err != nil {
		t.Fatalf("second sweep: %v", err)
	}
	if second.Enqueued != 0 {
		t.Errorf("second sweep enqueued %d jobs, want 0 (the inserts must dedupe)", second.Enqueued)
	}
	if got := e.riverJobTotal(t, id); got != 2 {
		t.Errorf("river jobs after the second sweep = %d, want the same 2 (no duplicates)", got)
	}

	// The pipeline drains to readiness; the settle makes the event
	// count deterministic (a second, wrong event would follow its
	// commit synchronously).
	eventually(t, jobWaitTimeout, "the twice-swept asset to become ready", func() bool {
		return e.assetOf(t, id).readyAt.Valid
	})
	time.Sleep(300 * time.Millisecond)

	state := e.assetOf(t, id)
	if state.pending != 0 {
		t.Errorf("pending_jobs = %d, want 0 after both sweeps drained", state.pending)
	}
	if got := countEvents(e.bus, "asset.ready"); got != 1 {
		t.Errorf("asset.ready events = %d, want exactly 1 across both sweeps", got)
	}

	// A third sweep once the row is ready: a clean no-op.
	third, err := RunRequeueSweep(context.Background(), e.pool, e.queue)
	if err != nil {
		t.Fatalf("third sweep: %v", err)
	}
	if third.Stuck != 0 || third.Enqueued != 0 {
		t.Errorf("third sweep counts = stuck %d enqueued %d, want the zero no-op on the ready row", third.Stuck, third.Enqueued)
	}
	if got := e.riverJobTotal(t, id); got != 2 {
		t.Errorf("river jobs after the third sweep = %d, want still 2", got)
	}
}

// TestSweepPeriodicRegistration pins the hourly periodic wiring
// without waiting a wall-clock hour: the cadence is one hour, the
// schedule advances by exactly that interval, the constructor
// inserts the argumentless sweep job on the default queue, and a
// started queue actually carries the registration in River's
// periodic job bundle (RemoveByID reports its existence).
func TestSweepPeriodicRegistration(t *testing.T) {
	if SweepInterval != time.Hour {
		t.Errorf("SweepInterval = %s, want the hourly 1h", SweepInterval)
	}

	now := time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC)
	if got := sweepSchedule().Next(now); !got.Equal(now.Add(time.Hour)) {
		t.Errorf("sweep schedule next(%s) = %s, want +1h", now, got)
	}

	args, opts := sweepJobConstructor()
	if _, ok := args.(SweepArgs); !ok {
		t.Errorf("periodic job args = %T, want SweepArgs", args)
	}
	if args.Kind() != "requeue_sweep" {
		t.Errorf("periodic job kind = %q, want requeue_sweep", args.Kind())
	}
	if opts != nil {
		t.Errorf("periodic job insert opts = %+v, want the defaults (nil)", opts)
	}

	if got := len(sweepPeriodicJobs()); got != 1 {
		t.Errorf("periodic jobs = %d, want exactly the sweep registration", got)
	}
	e := newEnv(t)
	if !e.queue.client.PeriodicJobs().RemoveByID(sweepPeriodicJobID) {
		t.Error("the requeue sweep periodic job is not registered in the started queue")
	}
}

// TestSweepWorkerJobRunsSweep drives the periodic job's worker path
// end to end: a SweepArgs job inserted into the live queue is worked
// by the registered SweepWorker, which runs the shared sweep with the
// requeue inserts issued through the River client of the running
// job, recovers the stuck row through the full pipeline, and emits
// exactly ONE log line for the run -- the counts, never a line per
// asset.
func TestSweepWorkerJobRunsSweep(t *testing.T) {
	requireExiftool(t)
	requireThumbnails(t)

	var logs bytes.Buffer
	e := newEnvLogger(t, slog.New(slog.NewTextHandler(&logs, nil)))

	deviceTakenAt := time.Date(2021, 8, 9, 10, 11, 12, 0, time.UTC).Truncate(time.Microsecond)
	id := e.insertAssetKind(t, "originals/sweep/worker.jpg", jpegBytes(t), deviceTakenAt,
		`{}`, dbgen.MediaKindImage, "image/jpeg", 2)

	// The periodic scheduler inserts exactly this job on the hourly
	// cadence; here it is inserted directly to drive the worker now.
	if _, err := e.queue.client.Insert(context.Background(), SweepArgs{}, nil); err != nil {
		t.Fatalf("insert sweep job: %v", err)
	}
	eventually(t, jobWaitTimeout, "the sweep job to complete", func() bool {
		var state string
		err := e.pool.QueryRow(context.Background(),
			`SELECT state FROM river_job WHERE kind = 'requeue_sweep'`).Scan(&state)
		return err == nil && state == "completed"
	})

	// The worker's sweep recovered the stuck row through the full
	// pipeline.
	eventually(t, jobWaitTimeout, "the worker-swept asset to become ready", func() bool {
		return e.assetOf(t, id).readyAt.Valid
	})
	if got := countEvents(e.bus, "asset.ready"); got != 1 {
		t.Errorf("asset.ready events = %d, want exactly 1", got)
	}

	// The log discipline: one line per sweep run, carrying the counts.
	// The buffer also carries River's own service diagnostics (the
	// queue logs through the same logger); only the sweep's own line
	// is counted.
	if got := strings.Count(logs.String(), "requeue sweep complete"); got != 1 {
		t.Errorf("requeue sweep complete log lines = %d, want exactly 1", got)
	}
}
