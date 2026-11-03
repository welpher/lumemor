// The ML rerun orchestration (M3a Task 4): the admin surface's rerun
// endpoint starts one run of a model's slot through EnqueueMlRerun,
// and this driver walks the rerun scope -- every image asset that
// finished its pipeline and is not in the trash -- dispatching each
// asset's ML work and counting the completions into the run's
// job_runs row (done for the successes, failed for the failures, the
// state closing as failed when any completion failed). The
// job.started / job.progress / job.finished events announce it onto
// the activity stream like the model download's runs do, the progress
// events throttled to at most one per second.
//
// THE HONEST BOUNDARY of this task: the per-asset ML workers (the
// face detection, recognition, CLIP embedding and OCR jobs) arrive
// with the milestone's later tasks. The dispatch seam below is the
// framework those tasks wire their workers into; until a slot's
// dispatch is wired, a rerun of that slot still creates its run but
// the run fails (state failed, zero progress) instead of pretending
// to have done work it cannot do. The run row's lifecycle, the
// dispatch counting and the aggregation ARE this task's deliverable.
//
// Failure semantics: a per-asset dispatch failure counts into the
// run's failed count -- it does not re-run the whole rerun (the
// per-asset retry belongs to the per-asset jobs of the later tasks).
// A bookkeeping failure (the row's own updates) returns the error so
// River retries the driver; the retry re-walks the scope from the
// first page, re-converging the row.
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
// one minute default: walking a whole library's scope -- a row read
// and one dispatch per asset -- legitimately takes minutes on a large
// instance.
const mlRerunTimeout = 30 * time.Minute

// mlRerunPageSize bounds one page of the scope's asset ids: the driver
// reads the scope one page at a time so a rerun over a large library
// never holds the whole id set in memory at once.
const mlRerunPageSize = 500

// MlRerunDispatchFunc runs one asset's ML work of the rerun: the seam
// the milestone's later tasks wire their per-asset ML workers into.
// A nil return counts the asset done; an error counts it failed. This
// task wires nothing -- the honest placeholder above -- so a run of a
// non-empty scope fails cleanly rather than pretending.
type MlRerunDispatchFunc func(ctx context.Context, slot ml.Slot, assetID uuid.UUID) error

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
// job.started fires before the first dispatch, every completion
// updates the row (with a throttled job.progress event), and the run
// closes with the converged counts and job.finished -- failed when
// any completion failed, succeeded otherwise.
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

	// The dispatch seam: no per-asset ML worker wired means the run
	// cannot do its work. An empty scope has nothing to do and closes
	// succeeded; anything else fails honestly instead of pretending.
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

	// The scope walk: one page at a time, one dispatch per asset, the
	// completions counted as they land. done may pass the enqueue-time
	// total -- assets that became ready during the run join the tail.
	done, failed := 0, 0
	lastProgress := time.Now()
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
			dispatchErr := w.dispatch(ctx, slot, assetID)
			if dispatchErr != nil {
				failed++
				w.logger.Warn("rerun dispatch failed", "job_run_id", jobRunID,
					"slot", slot, "asset_id", assetID, "error", dispatchErr)
			} else {
				done++
			}
			if err := queries.UpdateJobRunProgress(ctx, dbgen.UpdateJobRunProgressParams{
				ID:   pgUUID(jobRunID),
				Done: int32(done),
			}); err != nil {
				w.logger.Warn("update the rerun progress", "error", err, "job_run_id", jobRunID)
			}
			// The progress events inherit the download runs' throttle:
			// the first completion and the final counts are mandatory,
			// everything between at most one per second.
			if done+failed == 1 || time.Since(lastProgress) >= time.Second {
				w.bus.Publish("job.progress",
					jobEventPayload(jobRunID, kind, int(run.Total), done, 0, dbgen.JobStateRunning))
				lastProgress = time.Now()
			}
		}
	}

	state := dbgen.JobStateSucceeded
	if failed > 0 {
		state = dbgen.JobStateFailed
	}
	w.bus.Publish("job.progress",
		jobEventPayload(jobRunID, kind, int(run.Total), done, 0, dbgen.JobStateRunning))
	return w.closeRerun(ctx, jobRunID, kind, int(run.Total), done, failed, state)
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
