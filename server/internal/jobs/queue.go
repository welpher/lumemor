// Package jobs runs the lumemor asset pipeline on a River queue
// (PostgreSQL-native, over the shared pgxpool): the metadata job of
// Task 4, the thumbnail job of Task 5, and the transcode job of
// Task 6 -- three jobs per video asset, two per image. River owns its
// queue tables (river_job and friends) -- they are created by River's
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

	dbgen "github.com/welpher/lumemor/server/internal/db/gen"
	"github.com/welpher/lumemor/server/internal/events"
	"github.com/welpher/lumemor/server/internal/ml"
)

// maxWorkers caps the worker pool. The pipeline jobs are subprocess-
// and IO-bound, not CPU-bound, so one worker per CPU stops paying off
// quickly and more only adds contention on the shared pool.
const maxWorkers = 4

// softStopTimeout bounds a graceful drain: once it elapses, in-flight
// job contexts are cancelled (River escalates the soft stop to a hard
// one) instead of the drain waiting forever. It sits above the
// pipeline's largest job envelope -- the transcode job's 150s (whose
// ffmpeg subprocess carries a 120s cap of its own; the thumbnail
// job's 90s carries the poster extraction's 60s subprocess cap, and
// the metadata job's exiftool only 30s) -- so a healthy job always
// finishes within the drain.
const softStopTimeout = 155 * time.Second

// Config carries the dependencies of the job queue.
type Config struct {
	// Pool is the shared PostgreSQL connection pool; River reuses it.
	// River's own tables live in the same database, in the default
	// schema, outside the goose-managed schema.
	Pool *pgxpool.Pool
	// Bus receives the asset.ready event when an asset's pipeline
	// completes, and the job.* events when a model download run
	// progresses.
	Bus *events.Bus
	// DataDir is the photo data root: the jobs resolve the assets'
	// relative file_path columns against it.
	DataDir string
	// ModelsDir is the model store root: the model download job lands
	// each model in <ModelsDir>/<id>/ and the registry resolves
	// models.file_path against it.
	ModelsDir string
	// ModelsHfMirror is the HF domain mirror host (LUMEMOR_MODELS_HF_MIRROR
	// in production): the model download rewrites huggingface.co
	// hostnames to it wholesale. Empty = direct connections.
	ModelsHfMirror string
	// Registry registers and activates downloaded models. It is the
	// instance's SHARED registry -- the admin surface of a later task
	// consumes the same instance -- so the caller that owns one passes
	// it; nil builds one over the pool and ModelsDir.
	Registry *ml.Registry
	// ModelsManifest overrides the manifest the model download job
	// lands. Production leaves it nil (the embedded default group,
	// pure upstream links); tests point it at their local sources.
	ModelsManifest *ml.Manifest
	// MlRerunDispatch runs one asset's ML work of a rerun: the seam
	// the milestone's later tasks wire their per-asset ML workers
	// into. Production leaves it nil in this task -- the per-asset
	// workers do not exist yet, so a rerun of a non-empty scope fails
	// its run honestly (see ml_rerun.go's boundary note); the rerun
	// tests inject a fake.
	MlRerunDispatch MlRerunDispatchFunc
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
	// pool carries the model download's enqueue transaction (the
	// job_runs row and the River job land together).
	pool *pgxpool.Pool
	// models is the manifest the model download job lands, pinned at
	// Start: the enqueue computes the run's pinned byte total from it.
	models *ml.Manifest
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
	if cfg.ModelsDir == "" {
		return nil, errors.New("jobs: Config.ModelsDir is required")
	}
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}

	// The shared registry: the caller's instance when it owns one, a
	// fresh one over the pool otherwise.
	registry := cfg.Registry
	if registry == nil {
		registry = ml.NewRegistry(cfg.Pool, dbgen.New(cfg.Pool), cfg.ModelsDir)
	}

	// The manifest the download job lands: the caller's override, the
	// embedded default group otherwise.
	models := cfg.ModelsManifest
	if models == nil {
		var err error
		models, err = ml.DefaultManifest()
		if err != nil {
			return nil, fmt.Errorf("jobs: load the default model manifest: %w", err)
		}
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
	river.AddWorker(workers, &ThumbnailWorker{
		pool:    cfg.Pool,
		bus:     cfg.Bus,
		dataDir: cfg.DataDir,
		logger:  logger,
	})
	river.AddWorker(workers, &TranscodeWorker{
		pool:    cfg.Pool,
		bus:     cfg.Bus,
		dataDir: cfg.DataDir,
		logger:  logger,
	})
	river.AddWorker(workers, &SweepWorker{
		pool:   cfg.Pool,
		logger: logger,
	})
	river.AddWorker(workers, &RetentionWorker{
		pool:    cfg.Pool,
		dataDir: cfg.DataDir,
		bus:     cfg.Bus,
		logger:  logger,
	})
	river.AddWorker(workers, &ModelDownloadWorker{
		pool:      cfg.Pool,
		bus:       cfg.Bus,
		registry:  registry,
		modelsDir: cfg.ModelsDir,
		hfMirror:  cfg.ModelsHfMirror,
		manifest:  models,
		logger:    logger,
	})
	river.AddWorker(workers, &MlRerunWorker{
		pool:     cfg.Pool,
		bus:      cfg.Bus,
		dispatch: cfg.MlRerunDispatch,
		logger:   logger,
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
		// its own tighter 30 second bound inside that envelope; the
		// thumbnail job raises its envelope to 90s for the ffmpeg
		// poster extraction's 60s budget (see ThumbnailWorker.Timeout);
		// the transcode job raises its envelope to 150s for its
		// ffmpeg re-encode's 120s budget (see TranscodeWorker.Timeout).
		//
		// The two hourly maintenance jobs run through River's periodic
		// job scheduler: the elected leader enqueues the periodic jobs
		// -- in this single-process deployment, this client -- and the
		// workers run the shared functions (the boot runs of main.go
		// are the other trigger point of each). The requeue sweep
		// recovers stuck pipeline assets; the trash expiry purges
		// soft-deleted assets past their retention. They are separate
		// periodic registrations and separate workers on purpose:
		// independent failure and retry, independent kinds and counts
		// -- see retentionPeriodicJobs.
		PeriodicJobs: periodicJobs(),
	})
	if err != nil {
		return nil, fmt.Errorf("jobs: create river client: %w", err)
	}

	if err := client.Start(context.WithoutCancel(ctx)); err != nil {
		return nil, fmt.Errorf("jobs: start river client: %w", err)
	}

	return &Queue{client: client, pool: cfg.Pool, models: models}, nil
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

// EnqueueThumbnail inserts the thumbnail job for the asset. Like
// EnqueueMetadata it must run after the upload transaction committed:
// the job reads the asset row (and its file) that the transaction
// wrote. Re-enqueueing runs the job again idempotently: the derivations
// overwrite the same output paths, the guarded readiness decrement
// does not fire twice, and no second asset.ready event is published.
func (q *Queue) EnqueueThumbnail(ctx context.Context, assetID uuid.UUID) error {
	if _, err := q.client.Insert(ctx, ThumbnailArgs{AssetID: assetID}, nil); err != nil {
		return fmt.Errorf("insert thumbnail job: %w", err)
	}
	return nil
}

// EnqueueTranscode inserts the transcode job for a VIDEO asset: the
// playable web-safe MP4 only videos need (the upload handler gates the
// call on the media kind; an image asset's pipeline counts two jobs,
// not three) -- and for a Live Photo's video-family component, the
// one job its counter counts. Like the other enqueues it must run
// after the upload transaction committed. Re-enqueueing runs the job
// again idempotently: the re-encode overwrites the same play.mp4, the
// guarded readiness decrement does not fire twice, and no second
// asset.ready event is published.
func (q *Queue) EnqueueTranscode(ctx context.Context, assetID uuid.UUID) error {
	if _, err := q.client.Insert(ctx, TranscodeArgs{AssetID: assetID}, nil); err != nil {
		return fmt.Errorf("insert transcode job: %w", err)
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
