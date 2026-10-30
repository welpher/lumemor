package jobs

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"

	dbgen "github.com/welpher/lumemor/server/internal/db/gen"
)

// The requeue sweep is the asset pipeline's recovery path, closing
// the hard-gate handoff M2b's final review left open: an upload whose
// transaction committed but whose job enqueues failed (an insert
// failure, or a crash in the commit-to-enqueue window -- the
// disconnect window itself is already closed by the upload handler's
// WithoutCancel enqueue context) leaves the row with pending_jobs
// counting jobs that never entered the queue and ready_at never set.
// A re-upload answers duplicate without re-enqueueing, so no API path
// recovers the row. The sweep does: it finds the stuck rows and
// re-inserts the pipeline jobs their media kind calls for.
//
// The counter is deliberately NOT reset: it already counts exactly
// the missing jobs, the re-enqueued jobs' completions drive it down,
// and CompleteAssetJob's WHERE pending_jobs > 0 guard turns the extra
// decrements of any duplicate run into no-ops. River's unique opts
// over kind + encoded args (the asset id) make the re-insert itself
// idempotent, so a row whose jobs are actually still in the queue --
// the stuck-row query cannot tell the difference -- is left untouched
// instead of double-run.

// SweepInterval is the periodic cadence of the requeue sweep. One
// hour bounds how long a stuck asset stays stuck after the process is
// back, while keeping the sweep's scan of the assets table off the
// hot path of live uploads.
const SweepInterval = time.Hour

// sweepPeriodicJobID identifies the sweep's registration in River's
// periodic job bundle, so the registration is assertable by
// RemoveByID.
const sweepPeriodicJobID = "requeue-sweep"

// SweepCounts reports one sweep run: the stuck assets found and the
// pipeline jobs the run actually inserted. A requeue insert that
// River deduplicated (an equivalent job already exists) does not
// count -- operationally what matters is how much new work entered
// the queue. The two numbers are the single log line each sweep run
// emits.
type SweepCounts struct {
	Stuck    int
	Enqueued int
}

// AssetRequeuer is the sweep's enqueue seam. It mirrors the upload
// path's AssetJobEnqueuer shape, but its implementations insert
// through the sweep's unique-keyed path: the insert must be
// idempotent over the job kind and the asset id, so a job that
// already exists in the queue is not re-inserted.
type AssetRequeuer interface {
	// RequeueMetadata inserts the metadata job for the asset unless
	// an equivalent job already exists in River. It reports whether
	// a new job was actually inserted; false means a duplicate was
	// skipped, not an error.
	RequeueMetadata(ctx context.Context, assetID uuid.UUID) (bool, error)
	// RequeueThumbnail inserts the thumbnail job for the asset under
	// the same unique-keyed contract.
	RequeueThumbnail(ctx context.Context, assetID uuid.UUID) (bool, error)
}

// RunRequeueSweep finds the stuck pipeline assets and re-enqueues
// their jobs. A stuck asset is pending_jobs > 0 (the uploader counted
// jobs to enqueue) with ready_at still NULL (no completion ever
// emptied the counter) and not soft-deleted (a trashed asset is
// headed for expiry, not recovery). pending_jobs is NOT reset -- see
// the file comment. This function is the one shared implementation
// of both trigger points: the boot sweep in main.go and the hourly
// periodic SweepWorker call it.
func RunRequeueSweep(ctx context.Context, pool *pgxpool.Pool, enqueuer AssetRequeuer) (SweepCounts, error) {
	rows, err := dbgen.New(pool).ListStuckAssets(ctx)
	if err != nil {
		return SweepCounts{}, fmt.Errorf("jobs: list stuck assets: %w", err)
	}

	counts := SweepCounts{Stuck: len(rows)}
	for _, row := range rows {
		assetID := uuid.UUID(row.ID.Bytes)

		// The row's media kind picks the jobs the recovery re-inserts.
		// Both kinds share the same pair today -- the M2b two-job
		// pipeline, a video's poster frame riding the thumbnail job.
		// Task 6's transcode job splits the arms (video: metadata +
		// thumbnail + transcode) and raises the upload's pending_jobs
		// initialization to 3. Values outside the column's enum cannot
		// reach here; one that somehow did is skipped, not fatal.
		switch row.MediaKind {
		case dbgen.MediaKindImage, dbgen.MediaKindVideo:
			inserted, err := enqueuer.RequeueMetadata(ctx, assetID)
			if err != nil {
				return counts, fmt.Errorf("jobs: requeue metadata job of asset %s: %w", assetID, err)
			}
			if inserted {
				counts.Enqueued++
			}
			inserted, err = enqueuer.RequeueThumbnail(ctx, assetID)
			if err != nil {
				return counts, fmt.Errorf("jobs: requeue thumbnail job of asset %s: %w", assetID, err)
			}
			if inserted {
				counts.Enqueued++
			}
		}
	}
	return counts, nil
}

// RequeueMetadata inserts the metadata job for the asset through the
// sweep path. Unlike the upload path's EnqueueMetadata it inserts
// with River's unique opts over the job kind and the encoded args
// (the asset id): when a metadata job of this asset is still queued
// (available, pending, running, retryable, or scheduled) or has
// completed within River's completed-job retention (24 hours by
// default), the insert is skipped as a duplicate and this returns
// false without error. That is what makes the blind re-enqueue safe:
// rows whose jobs are merely in flight, not lost, are left alone.
func (q *Queue) RequeueMetadata(ctx context.Context, assetID uuid.UUID) (bool, error) {
	return requeueInsert(ctx, q.client, MetadataArgs{AssetID: assetID})
}

// RequeueThumbnail inserts the thumbnail job for the asset under the
// same unique-keyed contract as RequeueMetadata.
func (q *Queue) RequeueThumbnail(ctx context.Context, assetID uuid.UUID) (bool, error) {
	return requeueInsert(ctx, q.client, ThumbnailArgs{AssetID: assetID})
}

// requeueInsert inserts one pipeline job with uniqueness over its
// kind and encoded args, reporting whether a new job entered the
// queue (River marks a unique-skipped insert through
// UniqueSkippedAsDuplicate and returns the existing row).
func requeueInsert(ctx context.Context, client *river.Client[pgx.Tx], args river.JobArgs) (bool, error) {
	result, err := client.Insert(ctx, args, &river.InsertOpts{
		// ByArgs folds the full encoded args -- the asset id, the only
		// argument the pipeline jobs carry -- into the unique key, so
		// uniqueness is (kind, asset id). The default unique state set
		// covers every state a live job can hold plus completed (until
		// the job cleaner removes the row after its retention), which
		// keeps the dedupe window open over jobs that already ran. A
		// duplicate run would be harmless -- the pipeline jobs are
		// idempotent and the guarded readiness decrement no-ops -- but
		// not running it is cheaper, and the in-flight rows the sweep
		// cannot distinguish from stuck ones stay untouched.
		UniqueOpts: river.UniqueOpts{ByArgs: true},
	})
	if err != nil {
		return false, fmt.Errorf("insert %s job: %w", args.Kind(), err)
	}
	return !result.UniqueSkippedAsDuplicate, nil
}

// SweepArgs is the periodic requeue sweep job: it carries no payload,
// because the sweep reads its stuck rows from the database.
type SweepArgs struct{}

// Kind is River's stable job kind identifier.
func (SweepArgs) Kind() string { return "requeue_sweep" }

// SweepWorker runs the periodic sweep job: the shared RunRequeueSweep
// over the queue's own pool, with the requeue inserts issued through
// the River client of the running job (River injects the client into
// the Work context -- the idiomatic way for a job to enqueue other
// jobs; the workers are registered before the client exists, so the
// worker cannot hold the *Queue itself). The boot sweep of main.go
// runs the same function through the *Queue's own requeue methods.
type SweepWorker struct {
	river.WorkerDefaults[SweepArgs]

	pool   *pgxpool.Pool
	logger *slog.Logger
}

// Work executes one sweep run. The single log line carries the
// counts; per-asset logs are deliberately absent (a large recovery
// must not spam).
func (w *SweepWorker) Work(ctx context.Context, job *river.Job[SweepArgs]) error {
	counts, err := RunRequeueSweep(ctx, w.pool, sweepClientRequeuer{client: river.ClientFromContext[pgx.Tx](ctx)})
	if err != nil {
		// The sweep job itself fails and River retries it under its
		// default policy; the partial run's counts still log.
		w.logger.Error("requeue sweep failed", "error", err, "stuck", counts.Stuck, "enqueued", counts.Enqueued)
		return fmt.Errorf("requeue sweep: %w", err)
	}
	w.logger.Info("requeue sweep complete", "stuck", counts.Stuck, "enqueued", counts.Enqueued)
	return nil
}

// sweepClientRequeuer adapts the Work context's River client to the
// sweep's enqueue seam: the periodic path resolves the client out of
// the context River injects instead of holding a *Queue.
type sweepClientRequeuer struct {
	client *river.Client[pgx.Tx]
}

// RequeueMetadata implements AssetRequeuer through the shared
// unique-keyed insert.
func (r sweepClientRequeuer) RequeueMetadata(ctx context.Context, assetID uuid.UUID) (bool, error) {
	return requeueInsert(ctx, r.client, MetadataArgs{AssetID: assetID})
}

// RequeueThumbnail implements AssetRequeuer through the shared
// unique-keyed insert.
func (r sweepClientRequeuer) RequeueThumbnail(ctx context.Context, assetID uuid.UUID) (bool, error) {
	return requeueInsert(ctx, r.client, ThumbnailArgs{AssetID: assetID})
}

// sweepPeriodicJobs returns the periodic job registrations the queue
// starts with: the requeue sweep on its hourly schedule. RunOnStart
// is deliberately off -- the boot sweep in main.go covers the process
// start once, and a RunOnStart periodic would double it.
func sweepPeriodicJobs() []*river.PeriodicJob {
	return []*river.PeriodicJob{
		river.NewPeriodicJob(sweepSchedule(), sweepJobConstructor,
			&river.PeriodicJobOpts{ID: sweepPeriodicJobID}),
	}
}

// sweepSchedule is the sweep's periodic schedule: fixed hourly
// intervals. Split out so the cadence is unit-testable without
// waiting a wall-clock hour.
func sweepSchedule() river.PeriodicSchedule {
	return river.PeriodicInterval(SweepInterval)
}

// sweepJobConstructor is the periodic job's insert payload: the
// argumentless sweep job on the default queue with the default
// priority -- an hourly maintenance job has no business preempting
// pipeline work.
func sweepJobConstructor() (river.JobArgs, *river.InsertOpts) {
	return SweepArgs{}, nil
}
