// The ML rerun orchestration (M3a Task 4, closed by Task 8): the
// admin surface's rerun endpoint starts one run of a model's slot
// through EnqueueMlRerun, and this driver walks the rerun scope --
// every image asset that finished its pipeline and is not in the
// trash -- dispatching each asset's ML work and counting the
// completions into the run's job_runs row (done for the successes,
// failed for the failures, the state closing as failed when any
// completion failed). The job.started / job.progress / job.finished
// events announce it onto the activity stream like the model
// download's runs do, the progress events throttled to at most one per
// second.
//
// THE CLOSURE (Task 8): the dispatch seam's production wiring is now
// REAL -- the queue's dispatchMlRerun enqueues the slot's per-asset ML
// job (carrying the run's identifier) onto the ml queue. The per-asset
// jobs complete asynchronously through River, so the counting works
// from the completion side: every worker reports its own outcome into
// the run's row (an atomic increment -- the jobs of one run complete
// concurrently on the ml queue), and the driver, after its walk,
// WAITS for the counts to converge (done + failed >= dispatched) and
// closes the row itself. Rerun-dispatched jobs carry one attempt
// (their failure counts exactly once; a failed asset is visible in
// the run's failed count and recovered by a fresh rerun, whose
// idempotent upserts converge) and per-run args uniqueness (a driver
// retry's re-walk re-dispatches without piling duplicate jobs or
// double-counting completions).
//
// Interruption mirrors the model download's semantics: when the
// driver's envelope dies before the counts converge (a library whose
// ML work outlasts the 30-minute envelope), the row stays RUNNING --
// it is interrupted, not failed -- and River retries the driver, whose
// re-walk re-converges (the unique inserts make the re-dispatch a
// no-op for every job that already ran or is still queued).
//
// A per-asset dispatch whose ENQUEUE fails counts into the run's
// failed count directly (the asset's work never started; the per-asset
// retry belongs to a fresh rerun). A bookkeeping failure (the row's
// own updates) returns the error so River retries the driver.
package jobs

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"

	dbgen "github.com/welpher/lumemor/server/internal/db/gen"
	"github.com/welpher/lumemor/server/internal/events"
	"github.com/welpher/lumemor/server/internal/ml"
)

// The job_runs kinds of the rerun runs, pinned by the milestone plan's
// vocabulary: one kind per ML slot, named after the slot it redoes.
const (
	FaceDetectRerunKind    = "face_detect_rerun"
	FaceRecognizeRerunKind = "face_recognize_rerun"
	ClipEmbedRerunKind     = "clip_embed_rerun"
	OcrRerunKind           = "ocr_rerun"
)

// RerunKind maps an ML slot onto its rerun runs' job_runs kind. The
// second return is false for a slot with no rerun (a slot outside the
// ML pipeline of this milestone).
func RerunKind(slot ml.Slot) (string, bool) {
	switch slot {
	case ml.SlotFaceDetect:
		return FaceDetectRerunKind, true
	case ml.SlotFaceRecognize:
		return FaceRecognizeRerunKind, true
	case ml.SlotClipEmbed:
		return ClipEmbedRerunKind, true
	case ml.SlotOcr:
		return OcrRerunKind, true
	}
	return "", false
}

// MlRerunKind is the River job kind of the rerun driver. The job_runs
// kind carries the slot's pinned rerun name; the River kind is the
// internal dispatch key of the one driver that serves all four slots.
const MlRerunKind = "ml_rerun"

// mlRerunTimeout raises the driver's execution envelope above River's
// one minute default: the walk (a row read and one enqueue per asset)
// plus the wait for the per-asset jobs' completions -- a large
// library's ML work legitimately outlasts any envelope, and an
// expired attempt hands the run to the retry (the row stays running,
// the unique inserts keep the re-walk a no-op, the accumulated counts
// keep converging).
const mlRerunTimeout = 30 * time.Minute

// mlRerunPageSize bounds one page of the scope's asset ids: the driver
// reads the scope one page at a time so a rerun over a large library
// never holds the whole id set in memory at once.
const mlRerunPageSize = 500

// mlRerunPollInterval paces the driver's convergence wait: one row
// read (and at most one progress event) per tick.
const mlRerunPollInterval = time.Second

// MlRerunDispatchFunc dispatches one asset's ML work of a rerun: the
// enqueue of the slot's per-asset job, stamped with the run it reports
// its completion into. A nil return means the enqueue succeeded (the
// job's own outcome arrives later, through the completion report); an
// error counts the asset failed directly (the work never started).
// The queue wires its real dispatcher by default; the tests inject
// fakes of the same contract.
type MlRerunDispatchFunc func(ctx context.Context, slot ml.Slot, jobRunID, assetID uuid.UUID) error

// MlRerunArgs identifies the run the driver walks and the slot whose
// work it redoes.
type MlRerunArgs struct {
	JobRunID uuid.UUID `json:"jobRunId"`
	Slot     string    `json:"slot"`
}

// Kind is River's stable job kind identifier.
func (MlRerunArgs) Kind() string { return MlRerunKind }

// MlRerunWorker runs the ML rerun driver.
type MlRerunWorker struct {
	river.WorkerDefaults[MlRerunArgs]

	pool     *pgxpool.Pool
	bus      *events.Bus
	dispatch MlRerunDispatchFunc
	logger   *slog.Logger
}

// Timeout is the driver's execution envelope; see mlRerunTimeout.
func (w *MlRerunWorker) Timeout(job *river.Job[MlRerunArgs]) time.Duration {
	return mlRerunTimeout
}

// Work executes one rerun run: the row flips to running and
// job.started fires before the first dispatch, the walk enqueues one
// per-asset job per scope asset (an enqueue failure counts failed
// directly), the wait polls the row until the dispatched jobs'
// completions converge (done + failed >= dispatched), and the run
// closes with the converged counts and job.finished -- failed when any
// completion failed, succeeded otherwise.
func (w *MlRerunWorker) Work(ctx context.Context, job *river.Job[MlRerunArgs]) error {
	jobRunID := job.Args.JobRunID
	slot := ml.Slot(job.Args.Slot)
	queries := dbgen.New(w.pool)

	if err := queries.MarkJobRunRunning(ctx, pgUUID(jobRunID)); err != nil {
		return fmt.Errorf("mark the rerun run %s running: %w", jobRunID, err)
	}
	// The started event echoes the row's enqueue-time total -- the
	// durable truth the endpoint handed out -- not a recount.
	run, err := queries.GetJobRun(ctx, pgUUID(jobRunID))
	if err != nil {
		return fmt.Errorf("load the rerun run %s: %w", jobRunID, err)
	}
	kind, ok := RerunKind(slot)
	if !ok {
		// The enqueue refused unrerunnable slots; a mismatching row
		// here is corruption, not a retriable condition.
		w.logger.Error("rerun run carries an unrerunnable slot", "job_run_id", jobRunID, "slot", slot)
		return w.closeRerun(ctx, jobRunID, kind, int(run.Total), 0, 0, dbgen.JobStateFailed)
	}
	w.bus.Publish("job.started", jobEventPayload(jobRunID, kind, int(run.Total), 0, 0, dbgen.JobStateRunning))

	// The dispatch seam: no dispatcher wired means the run cannot do
	// its work (only a directly constructed driver can get here -- the
	// queue wires the real dispatcher). An empty scope has nothing to
	// do and closes succeeded; anything else fails honestly.
	if w.dispatch == nil {
		if run.Total > 0 {
			w.logger.Error("rerun run started without a per-asset dispatch wired",
				"job_run_id", jobRunID, "slot", slot, "total", run.Total)
		}
		state := dbgen.JobStateSucceeded
		if run.Total > 0 {
			state = dbgen.JobStateFailed
		}
		return w.closeRerun(ctx, jobRunID, kind, int(run.Total), 0, 0, state)
	}

	// The scope walk: one page at a time, one enqueue per asset. The
	// completions count from the per-asset jobs' own reports (the
	// workers' increments); only an enqueue failure counts here --
	// directly, because the asset's work never started. expected is
	// the walk's own account: every asset either enqueued (its job
	// will report) or counted failed here. Assets that became ready
	// during the run join the tail (they would otherwise miss both the
	// readiness seam and the rerun).
	expected := 0
	for offset := int32(0); ; offset += mlRerunPageSize {
		ids, err := queries.ListAssetIDsByKind(ctx, dbgen.ListAssetIDsByKindParams{
			MediaKind: dbgen.MediaKindImage,
			Limit:     mlRerunPageSize,
			Offset:    offset,
		})
		if err != nil {
			return fmt.Errorf("list the rerun scope of run %s: %w", jobRunID, err)
		}
		if len(ids) == 0 {
			break
		}
		for _, id := range ids {
			assetID := uuid.UUID(id.Bytes)
			expected++
			if err := w.dispatch(ctx, slot, jobRunID, assetID); err != nil {
				// The enqueue failed: the asset's job never started, so
				// its completion never arrives -- count it here.
				if err := queries.IncrementJobRunFailed(ctx, pgUUID(jobRunID)); err != nil {
					w.logger.Warn("count the failed enqueue", "error", err, "job_run_id", jobRunID)
				}
				w.logger.Warn("rerun dispatch failed", "job_run_id", jobRunID,
					"slot", slot, "asset_id", assetID, "error", err)
			}
		}
	}

	// The convergence wait: poll the row until the dispatched jobs'
	// completions reach the walk's account. The envelope's expiry
	// interrupts (the row stays running, River retries, the re-walk
	// re-converges); the poll's reads drive the throttled progress
	// events.
	final, converged := w.awaitCompletions(ctx, jobRunID, kind, expected)
	if !converged {
		w.logger.Warn("the rerun run outlasted its envelope before the completions converged; the retry re-walks and re-waits",
			"job_run_id", jobRunID, "slot", slot, "expected", expected)
		return fmt.Errorf("the rerun run %s did not converge before its envelope: the per-asset jobs are still completing", jobRunID)
	}

	state := dbgen.JobStateSucceeded
	if final.Failed > 0 {
		state = dbgen.JobStateFailed
	}
	w.bus.Publish("job.progress",
		jobEventPayload(jobRunID, kind, int(run.Total), int(final.Done), int(final.Failed), dbgen.JobStateRunning))
	return w.closeRerun(ctx, jobRunID, kind, int(run.Total), int(final.Done), int(final.Failed), state)
}

// awaitCompletions polls the run's row until the reported completions
// reach expected, answering the converged counts (converged false when
// the context died first). Every tick reads the row and emits at most
// one progress event with the live counts.
func (w *MlRerunWorker) awaitCompletions(ctx context.Context, jobRunID uuid.UUID, kind string, expected int) (dbgen.JobRun, bool) {
	queries := dbgen.New(w.pool)
	for {
		run, err := queries.GetJobRun(ctx, pgUUID(jobRunID))
		if err != nil {
			w.logger.Warn("read the rerun progress", "error", err, "job_run_id", jobRunID)
			return dbgen.JobRun{}, false
		}
		w.bus.Publish("job.progress",
			jobEventPayload(jobRunID, kind, int(run.Total), int(run.Done), int(run.Failed), dbgen.JobStateRunning))
		if int(run.Done)+int(run.Failed) >= expected {
			return run, true
		}
		select {
		case <-ctx.Done():
			return dbgen.JobRun{}, false
		case <-time.After(mlRerunPollInterval):
		}
	}
}

// reportMlRunCompletion counts one per-asset ML job's outcome into its
// rerun run's row: the completion-reporting half of the rerun closure
// (the workers call it at the end of Work; the driver's await loop
// converges on the counts). The report runs detached from the job
// context, with its own small deadline, so it lands even when the
// job's envelope consumed the context's last moments -- and a lost
// report (a database hiccup at that moment) logs and moves on: the
// driver's re-walk and the run's retry own the recovery.
func reportMlRunCompletion(ctx context.Context, pool *pgxpool.Pool, logger *slog.Logger, jobRunID uuid.UUID, failed bool) {
	reportCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
	defer cancel()
	var err error
	if failed {
		err = dbgen.New(pool).IncrementJobRunFailed(reportCtx, pgUUID(jobRunID))
	} else {
		err = dbgen.New(pool).IncrementJobRunDone(reportCtx, pgUUID(jobRunID))
	}
	if err != nil {
		logger.Warn("report the ML rerun completion", "error", err, "job_run_id", jobRunID)
	}
}

// closeRerun converges the row and announces the close. The final
// bookkeeping runs detached from the job context so the row and the
// finished event land even when the run consumed the context's last
// moments; a failure here returns the error and River retries the
// driver, which re-walks and re-converges the row.
func (w *MlRerunWorker) closeRerun(ctx context.Context, jobRunID uuid.UUID, kind string, total, done, failed int, state dbgen.JobState) error {
	finishCtx := context.WithoutCancel(ctx)
	if err := dbgen.New(w.pool).FinishJobRun(finishCtx, dbgen.FinishJobRunParams{
		ID:     pgUUID(jobRunID),
		State:  state,
		Done:   int32(done),
		Failed: int32(failed),
	}); err != nil {
		return fmt.Errorf("finish the rerun run %s: %w", jobRunID, err)
	}
	w.bus.Publish("job.finished", jobEventPayload(jobRunID, kind, total, done, failed, state))
	return nil
}

// rerunMlInsertOpts are the insert options of a rerun-dispatched
// per-asset ML job: the ml queue, ONE attempt (the job's failure
// counts exactly once -- a failed asset is visible in the run's failed
// count and recovered by a fresh rerun, whose idempotent upserts
// converge), and per-run args uniqueness (a driver retry's re-walk
// re-dispatches without piling duplicate jobs or double-counting
// completions -- the uniqueness spans the queued, running and
// completed states, so a re-dispatch after completion is a no-op too).
var rerunMlInsertOpts = &river.InsertOpts{
	Queue:       MlQueue,
	MaxAttempts: 1,
	UniqueOpts:  river.UniqueOpts{ByArgs: true},
}

// dispatchMlRerun is the rerun dispatch seam's real wiring: it enqueues
// the slot's per-asset ML job for the asset, stamped with the run the
// job reports its completion into. face_recognize_rerun dispatches the
// face_recognize job DIRECTLY -- the recognition worker redetects its
// own landmarks through the active detect model, so it stands alone;
// the rerun never detours through detection first.
func (q *Queue) dispatchMlRerun(ctx context.Context, slot ml.Slot, jobRunID, assetID uuid.UUID) error {
	switch slot {
	case ml.SlotFaceDetect:
		if _, err := q.client.Insert(ctx, FaceDetectArgs{AssetID: assetID, JobRunID: jobRunID}, rerunMlInsertOpts); err != nil {
			return fmt.Errorf("insert the rerun's face detect job: %w", err)
		}
	case ml.SlotFaceRecognize:
		if _, err := q.client.Insert(ctx, FaceRecognizeArgs{AssetID: assetID, JobRunID: jobRunID}, rerunMlInsertOpts); err != nil {
			return fmt.Errorf("insert the rerun's face recognize job: %w", err)
		}
	case ml.SlotClipEmbed:
		if _, err := q.client.Insert(ctx, ClipEmbedArgs{AssetID: assetID, JobRunID: jobRunID}, rerunMlInsertOpts); err != nil {
			return fmt.Errorf("insert the rerun's clip embed job: %w", err)
		}
	case ml.SlotOcr:
		if _, err := q.client.Insert(ctx, OcrArgs{AssetID: assetID, JobRunID: jobRunID}, rerunMlInsertOpts); err != nil {
			return fmt.Errorf("insert the rerun's ocr job: %w", err)
		}
	default:
		return fmt.Errorf("jobs: slot %s has no per-asset ML job to dispatch", slot)
	}
	return nil
}

// EnqueueMlRerun starts one rerun of the slot: the job_runs row (the
// slot's pinned rerun kind, the scope's asset count as its total) and
// the River driver job land in ONE transaction, so an enqueue that
// fails halfway leaves neither behind. The row starts queued; the
// driver flips it to running when the walk begins.
func (q *Queue) EnqueueMlRerun(ctx context.Context, slot ml.Slot) (uuid.UUID, error) {
	kind, ok := RerunKind(slot)
	if !ok {
		return uuid.Nil, fmt.Errorf("jobs: slot %s has no rerun", slot)
	}
	id, err := uuid.NewV7()
	if err != nil {
		return uuid.Nil, fmt.Errorf("generate the rerun run id: %w", err)
	}
	tx, err := q.pool.Begin(ctx)
	if err != nil {
		return uuid.Nil, fmt.Errorf("begin the rerun enqueue: %w", err)
	}
	// The rollback is a no-op after a successful commit; arguments are
	// captured while the request context is still live.
	defer func() { _ = tx.Rollback(ctx) }()

	// The scope's count, read inside the same transaction as the
	// insert: the total the row announces is exactly the scope the
	// enqueue saw.
	queries := dbgen.New(tx)
	total, err := queries.CountAssetIDsByKind(ctx, dbgen.MediaKindImage)
	if err != nil {
		return uuid.Nil, fmt.Errorf("count the rerun scope: %w", err)
	}
	if err := queries.InsertJobRun(ctx, dbgen.InsertJobRunParams{
		ID:    pgUUID(id),
		Kind:  kind,
		Total: int32(total),
	}); err != nil {
		return uuid.Nil, fmt.Errorf("insert the rerun run row: %w", err)
	}
	if _, err := q.client.InsertTx(ctx, tx, MlRerunArgs{JobRunID: id, Slot: string(slot)}, nil); err != nil {
		return uuid.Nil, fmt.Errorf("insert the rerun driver job: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return uuid.Nil, fmt.Errorf("commit the rerun enqueue: %w", err)
	}
	return id, nil
}
