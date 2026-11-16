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
	"net/url"
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

// downloadTransport reads the download sources' policy for one run:
// the allowed hosts (the transport gate every hop is validated
// against) and the mirror base (the rewrite of the pinned
// huggingface.co hostnames). The settings' mirror wins; the
// deployment's environment fallback answers only when no mirror is
// configured in the admin face -- the same rewrite either way, owned
// by the surface that edits it.
//
// The mirror's OWN host joins the gate either way: the rewrite sends
// the pinned hostnames at it, so a mirror the list refuses would
// fail on its own first hop and silently fall back to the real
// internet (the source chain's host validation, 32699df, walks the
// REWRITTEN urls) -- the settings save's auto-join discipline,
// applied to the deployment's environment mirror too.
func (w *ModelDownloadWorker) downloadTransport(ctx context.Context) (mirror string, allowed []string) {
	queries := dbgen.New(w.pool)
	settings, err := db.ModelSettingsOf(ctx, queries)
	if err != nil {
		// The accessor answered the documented default (the manifest's
		// own hosts); the line points at the row to fix.
		w.logger.Warn("read the model download sources; answering the defaults", "error", err)
	}
	mirror = w.hfMirror
	if settings.MirrorBaseUrl != "" {
		mirror = settings.MirrorBaseUrl
	}
	allowed = settings.AllowedHosts
	if host := mirrorHostOf(mirror); host != "" && !ml.HostAllowed(allowed, host) {
		allowed = append(allowed, host)
	}
	return mirror, allowed
}

// mirrorHostOf answers the host a mirror string names: a bare host,
// a host:port pair (the deployment's env form -- a scheme-less parse
// would misread the port as the scheme, so the authority form rides
// the // prefix), or a full http(s) URL. The empty string for
// anything that names no host.
func mirrorHostOf(mirror string) string {
	if mirror == "" {
		return ""
	}
	hostOf := func(raw string) string {
		parsed, err := url.Parse(raw)
		if err != nil {
			return ""
		}
		return parsed.Hostname()
	}
	if host := hostOf(mirror); host != "" {
		return host
	}
	return hostOf("//" + mirror)
}

// Work executes one model download attempt: the row flips to running
// and job.started fires before the first byte moves, the download
// service drives the throttled progress reports into the row and the
// job.progress events, and the attempt closes with the row and
// job.finished converging on the report's counts.
func (w *ModelDownloadWorker) Work(ctx context.Context, job *river.Job[ModelDownloadArgs]) error {
	jobRunID := job.Args.JobRunID
	queries := dbgen.New(w.pool)

	flagged, err := queries.MarkJobRunRunning(ctx, pgUUID(jobRunID))
	if err != nil {
		return fmt.Errorf("mark the model download run %s running: %w", jobRunID, err)
	}
	if flagged == 0 {
		// Two ways the pickup matched nothing: the cancel closed the
		// run while it sat queued (E41 -- the row stays cancelled,
		// this driver never starts, no started event for a run that
		// ended before its first chunk), or this attempt IS a retry
		// resuming an interrupted run (the interrupted attempt left
		// the row running; the work proceeds below).
		run, err := queries.GetJobRun(ctx, pgUUID(jobRunID))
		if err != nil {
			return fmt.Errorf("load the model download run %s: %w", jobRunID, err)
		}
		if run.State == dbgen.JobStateCancelled {
			return nil
		}
	}
	total := w.manifest.PinnedBytes()
	w.bus.Publish("job.started", jobEventPayload(jobRunID, ModelDownloadKind, total, 0, 0, dbgen.JobStateRunning))

	// The cancel boundary (E41): the flag check rides the download
	// service's own throttled report cadence -- one flag read per
	// report, and a set flag cancels the DERIVED context, which stops
	// the transfer at the file that owns it. The parent context stays
	// alive, so the interruption branch below cannot mistake the
	// admin's ending for an envelope death; the post-run flag read
	// closes the run cancelled with the row's own counts.
	downloadCtx, cancelDownload := context.WithCancel(ctx)
	defer cancelDownload()
	cancelRequested := func() bool {
		stop, err := queries.GetJobRunStopRequested(ctx, pgUUID(jobRunID))
		if err != nil {
			// A failed flag read never aborts the download on a
			// guess: the log line carries it, the transfer continues.
			w.logger.Warn("check the model download run's stop flag", "error", err, "job_run_id", jobRunID)
			return false
		}
		return stop
	}
	progress := func(done, total int) {
		if cancelRequested() {
			w.logger.Info("model download cancelled at its own boundary", "job_run_id", jobRunID, "bytes_done", done)
			cancelDownload()
			return
		}
		if err := queries.UpdateJobRunProgress(ctx, dbgen.UpdateJobRunProgressParams{
			ID:   pgUUID(jobRunID),
			Done: int32(done),
		}); err != nil {
			w.logger.Warn("update the model download progress", "error", err, "job_run_id", jobRunID)
			return
		}
		w.bus.Publish("job.progress", jobEventPayload(jobRunID, ModelDownloadKind, total, done, 0, dbgen.JobStateRunning))
	}

	mirror, allowed := w.downloadTransport(ctx)
	report, runErr := ml.RunDownload(downloadCtx, ml.RunDownloadDeps{
		Manifest:     w.manifest,
		ModelsDir:    w.modelsDir,
		Registry:     w.registry,
		Queries:      queries,
		HFMirror:     mirror,
		AllowedHosts: allowed,
	}, progress)

	// The cancel close: the flag owns the ending, the row's own
	// counts own the numbers (the report of a context-killed transfer
	// carries no honest partial). A run that finished its work before
	// the flag's read keeps its succeeded close -- the boundary lets
	// the in-flight chunk complete, and a completed run is not
	// cancelled. The envelope-expiry retry path below cannot mask the
	// ending: the flag is still set on the retry, whose first report
	// closes the run.
	if runErr != nil {
		if stop, err := queries.GetJobRunStopRequested(context.WithoutCancel(ctx), pgUUID(jobRunID)); err == nil && stop {
			run, err := queries.GetJobRun(context.WithoutCancel(ctx), pgUUID(jobRunID))
			if err != nil {
				return fmt.Errorf("load the cancelled model download run %s: %w", jobRunID, err)
			}
			w.bus.Publish("job.finished", jobEventPayload(jobRunID, ModelDownloadKind,
				total, int(run.Done), int(run.Failed), dbgen.JobStateCancelled))
			w.logger.Info("model download run closed cancelled", "job_run_id", jobRunID,
				"bytes_done", run.Done)
			return nil
		}
	}

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

// joinOrStartModelDownload decides what the automatic fill's seam does
// with the group's runs. Nil means a run is in flight (joined) or was
// just started, so the caller waits for it through its own retry;
// errModelDownloadFailed means the most recent run failed and no new
// one starts -- a broken source must not be re-fetched by every AI job,
// and the download endpoint's retry stays the administrator's.
//
// The rules, in order:
//
//   - a run still in flight is joined (its identifier is not needed
//     here: the caller only waits);
//   - the most recent run succeeded but the slot's files are missing
//     again -- the store was wiped below a configured instance -- so a
//     fresh run starts (its verified-file skip keeps the re-run cheap);
//   - the most recent run failed: no new run per job;
//   - no run at all starts one.
//
// The download endpoint keeps its own idempotence (a click joins an
// in-flight or the most recent run); this decision exists for the
// automatic path, whose trigger is a missing file rather than a click.
//
// The check-then-enqueue window is NOT closed, by design: two workers
// that read the run table in the same millisecond can both decide "no
// run in flight" and both enqueue, landing two queued run rows. In
// practice the enqueue's own transaction commits the queued row before
// the second worker's read runs, so the burst normally collapses to
// one run; the property the fill promises (a burst of AI jobs starts
// one download) is a practical one, not a database-guaranteed one.
// Closing it would need an arbiter on the read -- a lock across the
// table on every ML job's path -- for a case whose cost is one extra
// run row: the duplicate checks the file store again (the verified-
// file skip) and, at worst, re-fetches bytes the first run had not
// landed yet.
func (q *Queue) joinOrStartModelDownload(ctx context.Context) error {
	queries := dbgen.New(q.pool)
	if _, err := queries.FindInFlightJobRunByKind(ctx, ModelDownloadKind); err == nil {
		return nil
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("find the in-flight model download run: %w", err)
	}
	row, err := queries.LatestJobRunByKind(ctx, ModelDownloadKind)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		_, enqueueErr := q.EnqueueModelDownload(ctx)
		return enqueueErr
	case err != nil:
		return fmt.Errorf("find the latest model download run: %w", err)
	case row.State == dbgen.JobStateSucceeded:
		// The store lost files a finished run had landed: start again.
		_, enqueueErr := q.EnqueueModelDownload(ctx)
		return enqueueErr
	default:
		return errModelDownloadFailed
	}
}

// modelSettings reads the model store's policy through the shared
// accessor (the documented default answers a corrupt row).
func (q *Queue) modelSettings(ctx context.Context) (db.ModelSettings, error) {
	return db.ModelSettingsOf(ctx, dbgen.New(q.pool))
}
