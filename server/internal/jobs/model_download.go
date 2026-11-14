// The model download job: the admin surface's download
// endpoint enqueues one run per click (idempotent at the API: an
// in-flight or most recent run answers 200 without a new enqueue), and
// this worker drives the ml download service behind it. The job_runs
// row is the run's durable progress -- kind model_download, total and
// done carrying the group's pinned byte counts, failed the failed
// model count -- and the job.started / job.progress / job.finished
// events announce it onto the activity stream, the progress events
// throttled to at most one per second by the download service itself.
//
// Failure semantics: a model whose files cannot be landed or verified
// is not registered, and the run fails (state failed, with the done
// and failed counts). The worker returns the error so River retries
// the whole job, and the retry is idempotent: every file already
// verified on disk is skipped, so the attempt resumes instead of
// restarting. An interrupted attempt (the job's context cancelled by
// its envelope or a shutdown drain) leaves the row running -- it is
// interrupted, not failed -- and the next attempt picks it up.
package jobs

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"

	"github.com/welpher/lumemor/server/internal/db"
	dbgen "github.com/welpher/lumemor/server/internal/db/gen"
	"github.com/welpher/lumemor/server/internal/events"
	"github.com/welpher/lumemor/server/internal/ml"
)

// ModelDownloadKind pins the job_runs kind of the model download runs;
// the River job kind is the same string.
const ModelDownloadKind = "model_download"

// modelDownloadTimeout raises the job's execution envelope above
// River's one minute default: the default group is roughly 400 MiB of
// pinned bytes, and a slow link legitimately takes minutes. The bound
// keeps a wedged download retryable instead of pinning a worker slot
// forever.
const modelDownloadTimeout = 30 * time.Minute

// ModelDownloadArgs identifies the job_runs row the download drives.
type ModelDownloadArgs struct {
	JobRunID uuid.UUID `json:"jobRunId"`
}

// Kind is River's stable job kind identifier.
func (ModelDownloadArgs) Kind() string { return ModelDownloadKind }

// ModelDownloadWorker runs the model download job.
type ModelDownloadWorker struct {
	river.WorkerDefaults[ModelDownloadArgs]

	pool      *pgxpool.Pool
	bus       *events.Bus
	registry  *ml.Registry
	modelsDir string
	hfMirror  string
	manifest  *ml.Manifest
	logger    *slog.Logger
}

// Timeout is the job's execution envelope; see modelDownloadTimeout.
func (w *ModelDownloadWorker) Timeout(job *river.Job[ModelDownloadArgs]) time.Duration {
	return modelDownloadTimeout
}

// Work executes one model download attempt: the row flips to running
// and job.started fires before the first byte moves, the download
// service drives the throttled progress reports into the row and the
// job.progress events, and the attempt closes with the row and
// job.finished converging on the report's counts.
func (w *ModelDownloadWorker) Work(ctx context.Context, job *river.Job[ModelDownloadArgs]) error {
	jobRunID := job.Args.JobRunID
	queries := dbgen.New(w.pool)

	if err := queries.MarkJobRunRunning(ctx, pgUUID(jobRunID)); err != nil {
		return fmt.Errorf("mark the model download run %s running: %w", jobRunID, err)
	}
	total := w.manifest.PinnedBytes()
	w.bus.Publish("job.started", jobEventPayload(jobRunID, ModelDownloadKind, total, 0, 0, dbgen.JobStateRunning))

	// One row update and one job.progress event per progress report;
	// the download service throttles the reports themselves, so the
	// events inherit the bound. A progress update that fails (the job
	// context dying mid-download) logs and moves on: the run's final
	// bookkeeping below is where the row converges.
	progress := func(done, total int) {
		if err := queries.UpdateJobRunProgress(ctx, dbgen.UpdateJobRunProgressParams{
			ID:   pgUUID(jobRunID),
			Done: int32(done),
		}); err != nil {
			w.logger.Warn("update the model download progress", "error", err, "job_run_id", jobRunID)
			return
		}
		w.bus.Publish("job.progress", jobEventPayload(jobRunID, ModelDownloadKind, total, done, 0, dbgen.JobStateRunning))
	}

	report, runErr := ml.RunDownload(ctx, ml.RunDownloadDeps{
		Manifest:  w.manifest,
		ModelsDir: w.modelsDir,
		Registry:  w.registry,
		Queries:   queries,
		HFMirror:  w.hfMirror,
	}, progress)

	if runErr != nil && ctx.Err() != nil {
		// The context died (the job's envelope, or a shutdown drain):
		// the run is interrupted, not failed. The row stays running and
		// River retries; the next attempt resumes from the verified
		// files.
		w.logger.Warn("model download interrupted", "error", runErr, "job_run_id", jobRunID)
		return fmt.Errorf("model download run %s interrupted: %w", jobRunID, runErr)
	}

	state := dbgen.JobStateSucceeded
	failed := report.ModelsFailed
	if runErr != nil {
		state = dbgen.JobStateFailed
		w.logger.Error("model download failed", "error", runErr, "job_run_id", jobRunID,
			"models_done", report.ModelsDone, "models_failed", report.ModelsFailed)
	}

	// The final bookkeeping runs detached from the job context: the
	// row and the finished event must land even when the run consumed
	// the context's last moments.
	finishCtx := context.WithoutCancel(ctx)
	if err := queries.FinishJobRun(finishCtx, dbgen.FinishJobRunParams{
		ID:     pgUUID(jobRunID),
		State:  state,
		Done:   int32(report.BytesDone),
		Failed: int32(failed),
	}); err != nil {
		return fmt.Errorf("finish the model download run %s: %w", jobRunID, err)
	}
	w.bus.Publish("job.finished",
		jobEventPayload(jobRunID, ModelDownloadKind, report.BytesTotal, report.BytesDone, failed, state))

	if runErr != nil {
		// The returned error makes River retry: verified files are
		// skipped, so the retry resumes rather than restarts.
		return fmt.Errorf("model download run %s: %w", jobRunID, runErr)
	}
	w.logger.Info("model download complete", "job_run_id", jobRunID,
		"models_done", report.ModelsDone,
		"bytes_done", report.BytesDone, "bytes_total", report.BytesTotal)
	return nil
}

// jobEventPayload renders the shared payload of the job.* events: the
// run's identifier, kind and progress, nothing more -- the event
// audience discipline the asset events pin (camelCase field names,
// string values). The kind is the run's job_runs kind (the model
// download's, or the slot's pinned rerun name).
func jobEventPayload(jobRunID uuid.UUID, kind string, total, done, failed int, state dbgen.JobState) map[string]string {
	return map[string]string{
		"jobRunId": jobRunID.String(),
		"kind":     kind,
		"total":    strconv.Itoa(total),
		"done":     strconv.Itoa(done),
		"failed":   strconv.Itoa(failed),
		"state":    string(state),
	}
}

// EnqueueModelDownload starts one download run: the job_runs row and
// the River job land in ONE transaction, so an enqueue that fails
// halfway leaves neither behind. The row starts queued with the
// group's pinned byte count as its total; the worker flips it to
// running when the download begins.
func (q *Queue) EnqueueModelDownload(ctx context.Context) (uuid.UUID, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return uuid.Nil, fmt.Errorf("generate the model download run id: %w", err)
	}
	tx, err := q.pool.Begin(ctx)
	if err != nil {
		return uuid.Nil, fmt.Errorf("begin the model download enqueue: %w", err)
	}
	// The rollback is a no-op after a successful commit; arguments are
	// captured while the request context is still live.
	defer func() { _ = tx.Rollback(ctx) }()

	if err := dbgen.New(tx).InsertJobRun(ctx, dbgen.InsertJobRunParams{
		ID:    pgUUID(id),
		Kind:  ModelDownloadKind,
		Total: int32(q.models.PinnedBytes()),
	}); err != nil {
		return uuid.Nil, fmt.Errorf("insert the model download run row: %w", err)
	}
	if _, err := q.client.InsertTx(ctx, tx, ModelDownloadArgs{JobRunID: id}, nil); err != nil {
		return uuid.Nil, fmt.Errorf("insert the model download job: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return uuid.Nil, fmt.Errorf("commit the model download enqueue: %w", err)
	}
	return id, nil
}

// EnsureModelDownload joins the model group's download instead of
// stacking runs on it: a run still in flight answers its identifier,
// and so does the most recent finished run -- the download endpoint's
// idempotence, shared here so a burst of AI jobs behind one missing
// model starts one download rather than one per job. A group that
// already landed (or whose failed run owns its own retries) is
// therefore not re-fetched; the administrator's explicit retry is the
// path for that, exactly as the endpoint has it.
//
// The automatic fill's seam lives beside it (see ml_gate.go).
func (q *Queue) EnsureModelDownload(ctx context.Context) (uuid.UUID, error) {
	queries := dbgen.New(q.pool)
	if row, err := queries.FindInFlightJobRunByKind(ctx, ModelDownloadKind); err == nil {
		return uuid.UUID(row.ID.Bytes), nil
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, fmt.Errorf("find the in-flight model download run: %w", err)
	}
	if row, err := queries.LatestJobRunByKind(ctx, ModelDownloadKind); err == nil {
		return uuid.UUID(row.ID.Bytes), nil
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, fmt.Errorf("find the latest model download run: %w", err)
	}
	return q.EnqueueModelDownload(ctx)
}

// modelSettings reads the model store's policy through the shared
// accessor (the documented default answers a corrupt row).
func (q *Queue) modelSettings(ctx context.Context) (db.ModelSettings, error) {
	return db.ModelSettingsOf(ctx, dbgen.New(q.pool))
}
