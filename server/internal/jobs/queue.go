// Package jobs runs the lumemor asset pipeline on a River queue
// (PostgreSQL-native, over the shared pgxpool): the metadata job of
// Task 4 and, from Task 5 on, the thumbnail job. River owns its queue
// tables (river_job and friends) -- they are created by River's
// bundled migrations at startup and are deliberately not part of the
// goose-managed schema.
package jobs

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"runtime"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivermigrate"

	"github.com/welpher/lumemor/server/internal/events"
)

// maxWorkers caps the worker pool. The pipeline jobs are subprocess-
// and IO-bound, not CPU-bound, so one worker per CPU stops paying off
// quickly and more only adds contention on the shared pool.
const maxWorkers = 4

// softStopTimeout bounds a graceful drain: once it elapses, in-flight
// job contexts are cancelled (River escalates the soft stop to a hard
// one) instead of the drain waiting forever. It sits above the
// exiftool timeout so a healthy metadata job always finishes within
// the drain.
const softStopTimeout = 35 * time.Second

// Config carries the dependencies of the job queue.
type Config struct {
	// Pool is the shared PostgreSQL connection pool; River reuses it.
	// River's own tables live in the same database, in the default
	// schema, outside the goose-managed schema.
	Pool *pgxpool.Pool
	// Bus receives the asset.ready event when an asset's pipeline
	// completes.
	Bus *events.Bus
	// DataDir is the photo data root: the jobs resolve the assets'
	// relative file_path columns against it.
	DataDir string
	// Workers is the size of the worker pool. A non-positive value
	// falls back to min(GOMAXPROCS, maxWorkers).
	Workers int
	// Logger receives queue diagnostics; nil falls back to
	// slog.Default().
	Logger *slog.Logger
}

// Queue is the started River client behind the enqueue surface the
// upload handler uses. Construct with Start; stop with Stop.
type Queue struct {
	client *river.Client[pgx.Tx]
}

// Start applies River's own migrations (idempotent: applied versions
// are recorded and skipped), registers the workers, and starts the
// client. The queue's services deliberately do not die with ctx: only
// the migration and startup use it, because a cancelled start context
// would be a hard stop that cancels in-flight jobs at once. The
// graceful shutdown path is Stop.
func Start(ctx context.Context, cfg Config) (*Queue, error) {
	if cfg.Pool == nil {
		return nil, errors.New("jobs: Config.Pool is required")
	}
	if cfg.Bus == nil {
		return nil, errors.New("jobs: Config.Bus is required")
	}
	if cfg.DataDir == "" {
		return nil, errors.New("jobs: Config.DataDir is required")
	}
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}

	driver := riverpgxv5.New(cfg.Pool)

	migrator, err := rivermigrate.New(driver, &rivermigrate.Config{Logger: logger})
	if err != nil {
		return nil, fmt.Errorf("jobs: create river migrator: %w", err)
	}
	if _, err := migrator.Migrate(ctx, rivermigrate.DirectionUp, nil); err != nil {
		return nil, fmt.Errorf("jobs: apply river migrations: %w", err)
	}

	workers := river.NewWorkers()
	river.AddWorker(workers, &MetadataWorker{
		pool:    cfg.Pool,
		bus:     cfg.Bus,
		dataDir: cfg.DataDir,
		logger:  logger,
	})

	workerCount := cfg.Workers
	if workerCount <= 0 {
		workerCount = min(runtime.GOMAXPROCS(0), maxWorkers)
	}

	client, err := river.NewClient(driver, &river.Config{
		Queues: map[string]river.QueueConfig{
			river.QueueDefault: {MaxWorkers: workerCount},
		},
		Workers:         workers,
		Logger:          logger,
		SoftStopTimeout: softStopTimeout,
		// Everything else keeps River's defaults: the default retry
		// policy (25 attempts, exponential backoff) and the one minute
		// job timeout. The metadata job's exiftool subprocess carries
		// its own tighter 30 second bound inside that envelope.
	})
	if err != nil {
		return nil, fmt.Errorf("jobs: create river client: %w", err)
	}

	if err := client.Start(context.WithoutCancel(ctx)); err != nil {
		return nil, fmt.Errorf("jobs: start river client: %w", err)
	}

	return &Queue{client: client}, nil
}

// EnqueueMetadata inserts the metadata job for the asset. It must be
// called after the upload transaction committed: the job reads the
// asset and its exif sidecar row, which must be visible. Re-enqueueing
// an asset whose pipeline already completed runs the job again
// idempotently: the exif upsert converges to the same row and values,
// the guarded readiness decrement does not fire twice, and no second
// asset.ready event is published.
func (q *Queue) EnqueueMetadata(ctx context.Context, assetID uuid.UUID) error {
	if _, err := q.client.Insert(ctx, MetadataArgs{AssetID: assetID}, nil); err != nil {
		return fmt.Errorf("insert metadata job: %w", err)
	}
	return nil
}

// Stop drains the queue gracefully: producers stop fetching new jobs,
// in-flight jobs finish, and after softStopTimeout their contexts are
// cancelled. ctx bounds the wait; on its expiry Stop returns the
// context's error and shutdown proceeds.
func (q *Queue) Stop(ctx context.Context) error {
	return q.client.Stop(ctx)
}
