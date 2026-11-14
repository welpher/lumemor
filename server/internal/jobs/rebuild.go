// The derivative rebuild run: the image card's explicit
// re-derivation. One walk over every live image asset, deriving the
// serving copies at the settings the instance carries -- the sizes,
// the encoding, the full-size copy switch -- and updating each
// asset's blurhash (the tile changed, so its hash does).
//
// THE COST NEVER REDOES WORK BEHIND YOUR BACK: this run exists only
// when the admin asked for it (the settings save re-derives nothing),
// and the two scopes are the honest pair -- `missing` skips the
// assets whose current-format serving files already exist; `all`
// re-derives everything and sweeps the other encoding's stale files.
//
// THE BREAKPOINT: the walk persists its cursor on the job_runs row
// (the id of the last asset it finished). The budget gate reads per
// asset, so a window edge or a pause stops the walk AT the asset it
// just finished -- the running one completes, the next snoozes, and
// the next window or un-pause re-enters past everything already
// re-derived, never from zero. The stop flag closes the run between
// assets the same way: the work already done stays on disk, paid for
// exactly once.
package jobs

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"

	"github.com/welpher/lumemor/server/internal/db"
	dbgen "github.com/welpher/lumemor/server/internal/db/gen"
	"github.com/welpher/lumemor/server/internal/events"
	"github.com/welpher/lumemor/server/internal/media"
	"github.com/welpher/lumemor/server/internal/storage"
)

// RebuildKind is the job_runs kind of the derivative rebuild runs,
// pinned by the contract's JobRunKind enum. The River job kind
// carries the same string: one driver, no per-asset dispatch.
const RebuildKind = "rebuild_derivatives"

// The rebuild's two scopes, the contract's enum.
const (
	RebuildScopeMissing = "missing"
	RebuildScopeAll     = "all"
)

// rebuildTimeout raises the worker's execution envelope above River's
// one minute default: the walk over a family library legitimately
// outlasts a small envelope, and an expired attempt hands the job to
// the retry, whose re-entry resumes from the persisted cursor.
const rebuildTimeout = 30 * time.Minute

// rebuildPageSize bounds one page of the walk: the scope's asset ids
// and file paths ride in memory one page at a time, so a rebuild of a
// large library never holds the whole set at once.
const rebuildPageSize = 200

// rebuildProgressInterval paces the run's progress events onto the
// activity stream, read off the walk's own clock. The job_runs row
// itself updates per page (a cheap bounded write), so the panel's
// truth stays page-fresh.
const rebuildProgressInterval = time.Second

// RebuildArgs identifies the run the worker walks; the scope names
// the honest pair the dialog chose.
type RebuildArgs struct {
	JobRunID uuid.UUID `json:"jobRunId"`
	Scope    string    `json:"scope"`
}

// Kind is River's stable job kind identifier.
func (RebuildArgs) Kind() string { return RebuildKind }

// RebuildWorker runs the derivative rebuild walk.
type RebuildWorker struct {
	river.WorkerDefaults[RebuildArgs]

	pool    *pgxpool.Pool
	bus     *events.Bus
	dataDir string
	logger  *slog.Logger
	budget  *Budget
}

// Timeout is the worker's execution envelope; see rebuildTimeout.
func (w *RebuildWorker) Timeout(job *river.Job[RebuildArgs]) time.Duration {
	return rebuildTimeout
}

// Work executes one rebuild run: the row flips to running and
// job.started fires, the dial takes one pipeline slot for the whole
// walk (the rebuild IS thumbnail-class work -- the other pipeline
// jobs share what remains), the walk derives per asset with the gate
// reading between assets, and the run closes with the converged
// counts and job.finished -- failed when any derivation failed, or
// when the stop was asked for (failed counting nothing; the panel
// renders the reason line as stopped).
func (w *RebuildWorker) Work(ctx context.Context, job *river.Job[RebuildArgs]) error {
	jobRunID := job.Args.JobRunID
	queries := dbgen.New(w.pool)

	if err := queries.MarkJobRunRunning(ctx, pgUUID(jobRunID)); err != nil {
		return fmt.Errorf("mark the rebuild run %s running: %w", jobRunID, err)
	}
	// The started event echoes the row's enqueue-time total -- the
	// durable truth the endpoint handed out -- not a recount.
	run, err := queries.GetJobRun(ctx, pgUUID(jobRunID))
	if err != nil {
		return fmt.Errorf("load the rebuild run %s: %w", jobRunID, err)
	}
	total := int(run.Total)
	w.bus.Publish("job.started", jobEventPayload(jobRunID, RebuildKind, total, 0, 0, dbgen.JobStateRunning))

	// The scope: the contract's enum. A directly constructed worker
	// with a wrong scope is a wiring bug; the run closes failed
	// instead of pretending to have rebuilt anything.
	if job.Args.Scope != RebuildScopeMissing && job.Args.Scope != RebuildScopeAll {
		w.logger.Error("rebuild run started with a scope outside the enum",
			"job_run_id", jobRunID, "scope", job.Args.Scope)
		return w.finishRebuild(ctx, jobRunID, total, 0, 0, dbgen.JobStateFailed)
	}

	// The dial: one pipeline slot for the whole walk. Nil budgets
	// (the unit tests' directly constructed workers) gate nothing.
	if w.budget != nil {
		if err := w.budget.Acquire(ctx, classThumbnail); err != nil {
			return err
		}
		defer w.budget.Release()
	}

	// The settings the walk derives at: read once at the walk's
	// start, and the walk keeps them -- a settings save mid-rebuild
	// binds at the NEXT rebuild, never retroactively (the design
	// pin: the run uses its starting parameters).
	image, imageErr := db.ImageSettingsOf(ctx, dbgen.New(w.pool))
	if imageErr != nil {
		w.logger.Error("rebuild run: image settings are corrupt, using the default", "error", imageErr)
	}
	opts := media.DeriveOptions{
		TileBox:      image.TileSize,
		ViewLongEdge: image.PreviewSize,
		Format:       image.Format,
		FullSize:     image.FullSize,
		RawEmbedded:  image.RawEmbedded,
	}

	// The re-entry: the persisted cursor is the id of the last asset
	// the walk finished. Zero starts the walk; a re-entry (the
	// window reopened, the pause lifted, a retry after a crash)
	// skips past everything already re-derived.
	done := int(run.Done)
	failed := 0
	lastEvent := time.Now()
	cursor := run.ResumeFrom
	if !cursor.Valid {
		// A fresh run starts the walk at the zero id (the keyset's
		// floor); the row's NULL is "not started", not a filter.
		cursor = pgUUID(uuid.Nil)
	}
	for {
		// The stop check per page: one boolean read. The run closes
		// failed-with-zero-failures -- the panel's reason line says
		// stopped -- and the work already done stays on disk.
		stop, err := queries.GetJobRunStopRequested(ctx, pgUUID(jobRunID))
		if err != nil {
			return fmt.Errorf("check the stop flag of rebuild run %s: %w", jobRunID, err)
		}
		if stop {
			w.logger.Info("rebuild run stopped at its own breakpoint", "job_run_id", jobRunID, "done", done)
			return w.finishRebuild(ctx, jobRunID, total, done, 0, dbgen.JobStateFailed)
		}

		rows, err := queries.ListImageAssetPage(ctx, dbgen.ListImageAssetPageParams{
			Cursor:   cursor,
			PageSize: rebuildPageSize,
		})
		if err != nil {
			return fmt.Errorf("list the rebuild scope of run %s: %w", jobRunID, err)
		}
		if len(rows) == 0 {
			break
		}
		for _, row := range rows {
			// The gate per asset: the breakpoint. A paused or
			// window-held walk snoozes HERE -- after the last
			// finished asset's cursor write -- so the next window
			// re-enters exactly past it. The dial slot releases on
			// the return (the deferred release); the snoozed job
			// holds nothing but its future wake-up.
			if w.budget != nil {
				if hold := w.budget.Hold(ctx, classThumbnail); hold > 0 {
					return river.JobSnooze(hold)
				}
			}

			cursor = row.ID
			assetID := uuid.UUID(row.ID.Bytes)
			path, err := storage.ResolveAssetPath(w.dataDir, row.FilePath, string(row.StorageKind))
			if err != nil {
				failed++
				w.logger.Warn("rebuild resolve failed", "job_run_id", jobRunID, "asset_id", assetID, "error", err)
				continue
			}

			// The missing scope skips the assets whose
			// current-format serving files already exist: a library
			// that has never been rebuilt, or one that gained the
			// full-size copy, pays only for what it lacks.
			if job.Args.Scope == RebuildScopeMissing &&
				derivativesPresent(assetID.String(), image.Format, image.FullSize) {
				done++
				continue
			}

			hash, err := media.GenerateThumbnailsWithOptions(path, assetID.String(), opts)
			if err != nil {
				failed++
				w.logger.Warn("rebuild derivation failed", "job_run_id", jobRunID, "asset_id", assetID, "error", err)
				continue
			}
			// The tile changed (maybe its size, maybe its encoding):
			// the row's blurhash follows it.
			if err := queries.UpdateAssetBlurhash(ctx, dbgen.UpdateAssetBlurhashParams{
				ID:       row.ID,
				Blurhash: pgtype.Text{String: hash, Valid: true},
			}); err != nil {
				failed++
				w.logger.Warn("rebuild blurhash write failed", "job_run_id", jobRunID, "asset_id", assetID, "error", err)
				continue
			}
			// The full rebuild sweeps the OTHER encoding's stale
			// files: the new encoding's copies replaced them, and
			// the old bytes are disk nobody reads again.
			if job.Args.Scope == RebuildScopeAll {
				sweepStaleDerivatives(assetID.String(), image.Format)
			}
			done++
		}
		if err := queries.UpdateJobRunProgress(ctx, dbgen.UpdateJobRunProgressParams{
			ID:   pgUUID(jobRunID),
			Done: int32(done),
		}); err != nil {
			return fmt.Errorf("record the rebuild progress of run %s: %w", jobRunID, err)
		}
		// The breakpoint write: the cursor persists AFTER the page's
		// assets, so a re-entry past it re-derives nothing.
		if err := queries.SetJobRunResumeFrom(ctx, dbgen.SetJobRunResumeFromParams{
			ID:         pgUUID(jobRunID),
			ResumeFrom: cursor,
		}); err != nil {
			return fmt.Errorf("persist the rebuild cursor of run %s: %w", jobRunID, err)
		}
		if now := time.Now(); now.Sub(lastEvent) >= rebuildProgressInterval {
			lastEvent = now
			w.bus.Publish("job.progress",
				jobEventPayload(jobRunID, RebuildKind, total, done, failed, dbgen.JobStateRunning))
		}
	}

	state := dbgen.JobStateSucceeded
	if failed > 0 {
		state = dbgen.JobStateFailed
	}
	return w.finishRebuild(ctx, jobRunID, total, done, failed, state)
}

// finishRebuild converges the row and announces the close, clearing
// the walk's cursor scratch. The final bookkeeping runs detached from
// the job context so the row and the finished event land even when
// the run consumed the context's last moments; a failure here returns
// the error and River retries the job, whose re-entry re-converges
// the row.
func (w *RebuildWorker) finishRebuild(ctx context.Context, jobRunID uuid.UUID, total, done, failed int, state dbgen.JobState) error {
	finishCtx := context.WithoutCancel(ctx)
	queries := dbgen.New(w.pool)
	if err := queries.FinishJobRun(finishCtx, dbgen.FinishJobRunParams{
		ID:     pgUUID(jobRunID),
		State:  state,
		Done:   int32(done),
		Failed: int32(failed),
	}); err != nil {
		return fmt.Errorf("finish the rebuild run %s: %w", jobRunID, err)
	}
	if err := queries.ClearJobRunResumeFrom(finishCtx, pgUUID(jobRunID)); err != nil {
		return fmt.Errorf("clear the rebuild cursor of run %s: %w", jobRunID, err)
	}
	w.bus.Publish("job.finished", jobEventPayload(jobRunID, RebuildKind, total, done, failed, state))
	return nil
}

// derivativesPresent reports whether the asset's current-format
// serving files all exist: the tile and the view always, the
// full-size copy when the settings ask for it. The missing scope's
// skip is built on this -- one stat per file, no decodes.
func derivativesPresent(assetID, format string, fullSize bool) bool {
	sizes := []string{media.SizeTile, media.SizeView}
	if fullSize {
		sizes = append(sizes, media.SizeOriginal)
	}
	for _, size := range sizes {
		if _, err := os.Stat(storage.ThumbPath(assetID, size, format)); err != nil {
			return false
		}
	}
	return true
}

// sweepStaleDerivatives removes the other encoding's serving files of
// one asset, best-effort and idempotent: a file already gone is
// success, and a failure is logged (the bytes are disk nobody reads
// under the current settings; the next full rebuild sweeps again).
func sweepStaleDerivatives(assetID, format string) {
	other := media.FormatWebp
	if format == media.FormatWebp {
		other = media.FormatAvif
	}
	for _, size := range []string{media.SizeTile, media.SizeView, media.SizeOriginal} {
		if err := os.Remove(storage.ThumbPath(assetID, size, other)); err != nil && !errors.Is(err, os.ErrNotExist) {
			slog.Warn("rebuild stale-format sweep failed", "asset_id", assetID, "path", size+"."+other, "error", err)
		}
	}
}

// EnqueueDerivativesRebuild starts one rebuild run: the job_runs row
// (the rebuild kind, the image scope's asset count as its total) and
// the River job land in ONE transaction, so an enqueue that fails
// halfway leaves neither behind. The row starts queued; the worker
// flips it to running when the walk begins.
func (q *Queue) EnqueueDerivativesRebuild(ctx context.Context, scope string) (uuid.UUID, error) {
	if scope != RebuildScopeMissing && scope != RebuildScopeAll {
		return uuid.Nil, fmt.Errorf("jobs: rebuild scope %q is outside the enum", scope)
	}
	id, err := uuid.NewV7()
	if err != nil {
		return uuid.Nil, fmt.Errorf("generate the rebuild run id: %w", err)
	}
	tx, err := q.pool.Begin(ctx)
	if err != nil {
		return uuid.Nil, fmt.Errorf("begin the rebuild enqueue: %w", err)
	}
	// The rollback is a no-op after a successful commit; arguments
	// are captured while the request context is still live.
	defer func() { _ = tx.Rollback(ctx) }()

	// The scope's count, read inside the same transaction as the
	// insert: the total the row announces is exactly the scope the
	// enqueue saw.
	queries := dbgen.New(tx)
	total, err := queries.CountImageAssets(ctx)
	if err != nil {
		return uuid.Nil, fmt.Errorf("count the rebuild scope: %w", err)
	}
	if err := queries.InsertJobRun(ctx, dbgen.InsertJobRunParams{
		ID:    pgUUID(id),
		Kind:  RebuildKind,
		Total: int32(total),
	}); err != nil {
		return uuid.Nil, fmt.Errorf("insert the rebuild run row: %w", err)
	}
	if _, err := q.client.InsertTx(ctx, tx, RebuildArgs{JobRunID: id, Scope: scope}, nil); err != nil {
		return uuid.Nil, fmt.Errorf("insert the rebuild job: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return uuid.Nil, fmt.Errorf("commit the rebuild enqueue: %w", err)
	}
	return id, nil
}
