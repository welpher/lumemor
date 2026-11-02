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
// re-inserts the pipeline jobs their media kind and role call for
// (the component rows of Live Photos recover their transcode alone).
//
// The counter is deliberately NOT reset: it already counts exactly
// the missing jobs, the re-enqueued jobs' completions drive it down,
// and CompleteAssetJob's WHERE pending_jobs > 0 guard turns the extra
// decrements of any duplicate run into no-ops -- which is the actual
// safety net for duplicate runs: River's unique opts over kind +
// encoded args (the asset id) only dedupe the sweep's KEYED inserts
// against each other (a previous or concurrent sweep's jobs, or a
// completed sweep job still inside River's retention). The upload
// path enqueues WITHOUT unique opts, so its in-flight rows never join
// the unique index: a stuck-looking row whose jobs are mid-pipeline
// through the upload path gets a duplicate set from the sweep (two
// jobs for an image, three for a video), and all of them run --
// converged, not prevented, by the pipeline jobs' own idempotency
// and the guarded decrement.

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
// River deduplicated against an earlier keyed insert (sweep-vs-sweep;
// see the boundary note on AssetRequeuer) does not count --
// operationally what matters is how much new work entered the queue.
// The two numbers are the single log line each sweep run emits.
type SweepCounts struct {
	Stuck    int
	Enqueued int
}

// AssetRequeuer is the sweep's enqueue seam. It mirrors the upload
// path's AssetJobEnqueuer shape, but its implementations insert
// through the sweep's unique-keyed path: the insert is idempotent
// over the job kind and the asset id AGAINST OTHER SWEEP-PATH
// INSERTS (a previous or concurrent sweep, or a completed sweep job
// inside River's retention). The upload path's unkeyed inserts are
// outside that dedupe -- the boundary is pinned in
// TestRequeueSweepUniqueKeyBoundary.
type AssetRequeuer interface {
	// RequeueMetadata inserts the metadata job for the asset unless
	// an equivalent job already exists in River. It reports whether
	// a new job was actually inserted; false means a duplicate was
	// skipped, not an error.
	RequeueMetadata(ctx context.Context, assetID uuid.UUID) (bool, error)
	// RequeueThumbnail inserts the thumbnail job for the asset under
	// the same unique-keyed contract.
	RequeueThumbnail(ctx context.Context, assetID uuid.UUID) (bool, error)
	// RequeueTranscode inserts the transcode job for the asset under
	// the same unique-keyed contract. The video arm and the Live
	// Photo component arm call it; the image arm does not.
	RequeueTranscode(ctx context.Context, assetID uuid.UUID) (bool, error)
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

		// The row's media kind and role pick the jobs the recovery
		// re-inserts. An image counts the two-job pipeline of M2b
		// (metadata + thumbnail); a video counts THREE since Task 6's
		// transcode job (metadata + thumbnail + transcode), matching
		// the upload's pending_jobs initialization. A Live Photo's
		// component row (M2d) counts ONE -- the transcode alone: its
		// counter was initialized to exactly that job, its metadata
		// refinement rides the upload path and consumes nothing off
		// the counter, so the recovery re-inserts the transcode only
		// (a metadata or thumbnail requeue here would derive files no
		// view ever reads and could consume the transcode's count
		// through an early completion). An image-family component
		// counts no pipeline at all -- nothing to requeue; values
		// outside the column's enum cannot reach here either. One of
		// those shapes being skipped, not fatal, mirrors the kind
		// arm's discipline below.
		switch {
		case row.IsComponent:
			if row.MediaKind == dbgen.MediaKindVideo {
				inserted, err := enqueuer.RequeueTranscode(ctx, assetID)
				if err != nil {
					return counts, fmt.Errorf("jobs: requeue transcode job of asset %s: %w", assetID, err)
				}
				if inserted {
					counts.Enqueued++
				}
			}
		case row.MediaKind == dbgen.MediaKindImage:
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
		case row.MediaKind == dbgen.MediaKindVideo:
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
			inserted, err = enqueuer.RequeueTranscode(ctx, assetID)
			if err != nil {
				return counts, fmt.Errorf("jobs: requeue transcode job of asset %s: %w", assetID, err)
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
// (the asset id): when a metadata job of this asset is still present
// FROM A KEYED INSERT -- a previous or concurrent sweep's insert,
// queued (available, pending, running, retryable, or scheduled) or
// completed within River's completed-job retention (24 hours by
// default) -- the insert is skipped as a duplicate and this returns
// false without error. The upload path's inserts are unkeyed and
// never match the unique key, so a row whose jobs are in flight
// through the upload path gets a duplicate job, and both run to
// completion: harmless by design (the pipeline jobs are idempotent
// and the guarded readiness decrement no-ops), which is what makes
// the blind re-enqueue safe.
func (q *Queue) RequeueMetadata(ctx context.Context, assetID uuid.UUID) (bool, error) {
	return requeueInsert(ctx, q.client, MetadataArgs{AssetID: assetID})
}

// RequeueThumbnail inserts the thumbnail job for the asset under the
// same unique-keyed contract as RequeueMetadata.
func (q *Queue) RequeueThumbnail(ctx context.Context, assetID uuid.UUID) (bool, error) {
	return requeueInsert(ctx, q.client, ThumbnailArgs{AssetID: assetID})
}

// RequeueTranscode inserts the transcode job for the asset under the
// same unique-keyed contract as RequeueMetadata.
func (q *Queue) RequeueTranscode(ctx context.Context, assetID uuid.UUID) (bool, error) {
	return requeueInsert(ctx, q.client, TranscodeArgs{AssetID: assetID})
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
		// keeps the dedupe window open over the sweep's own jobs that
		// already ran: a repeated or overlapping sweep re-inserts
		// nothing. The boundary: River's conflict clause matches rows
		// carrying the same unique key, and only KEYED inserts carry
		// one -- the upload path's EnqueueMetadata/EnqueueThumbnail/
		// EnqueueTranscode insert unkeyed, so a row whose jobs are in
		// flight through the upload path gets a duplicate set from the
		// sweep, and all of them run. That duplicate is converged, not
		// prevented: the pipeline jobs are idempotent (re-runs
		// overwrite the same outputs) and the guarded readiness
		// decrement no-ops.
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

// RequeueTranscode implements AssetRequeuer through the shared
// unique-keyed insert.
func (r sweepClientRequeuer) RequeueTranscode(ctx context.Context, assetID uuid.UUID) (bool, error) {
	return requeueInsert(ctx, r.client, TranscodeArgs{AssetID: assetID})
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
