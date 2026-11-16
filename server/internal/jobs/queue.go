// Package jobs runs the lumemor asset pipeline on a River queue
// (PostgreSQL-native, over the shared pgxpool): the metadata job,
// the thumbnail job, and the transcode job --
// three jobs per video asset, two per image. River owns its
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
	"github.com/welpher/lumemor/server/internal/geodata"
	"github.com/welpher/lumemor/server/internal/ml"
)

// maxWorkers caps the default queue's worker pool. The pipeline jobs
// are subprocess- and IO-bound, not CPU-bound, so more workers add
// contention on the shared pool; the run budget's dial decides how
// many slots are actually open (its semaphore), and the pool is sized
// to fit the dial's ceiling so a full-speed dial on a many-core NAS
// is never silently short of workers.
const maxWorkers = 8

// MlQueue is the River queue of the per-asset ML jobs (face
// detection, face recognition, CLIP embedding, OCR):
// its worker pool is deliberately SMALL, because the ML jobs are the
// pipeline's only CPU-heavy work (onnxruntime inference over decoded
// rasters), and the pinned discipline keeps their concurrency
// conservative so a burst of ML work never starves the media pipeline
// or the API of worker slots. The separation is also structural for
// the rerun driver: the driver runs on the default queue and WAITS
// for the per-asset jobs of its run -- a driver sharing the ml
// workers with the jobs it waits for could starve them (four
// concurrent slot reruns holding every default worker while the ml
// queue sits behind them).
const MlQueue = "ml"

// mlQueueWorkers is the ml queue's conservative worker count.
const mlQueueWorkers = 2

// softStopTimeout bounds a graceful drain: once it elapses, in-flight
// job contexts are cancelled (River escalates the soft stop to a hard
// one) instead of the drain waiting forever. It sits above the
// pipeline's largest job envelope -- the transcode job's 150s (whose
// ffmpeg subprocess carries a 120s cap of its own; the thumbnail
// job's 90s carries the poster extraction's 60s subprocess cap, the
// metadata job's exiftool only 30s, the face detection and CLIP
// embedding jobs' CPU inference 60s each, and the face recognition
// and OCR jobs' envelopes 120s, the recognition's library-wide
// clustering pass and the OCR's per-box recognition being the largest
// ML steps) -- so a healthy job always finishes within the drain.
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
	// whose production wiring is the queue's real dispatcher (the
	// rerun closure -- the four slots' per-asset enqueues onto the ml
	// queue). Production leaves it nil and the queue wires its own
	// dispatcher; the rerun tests inject fakes of the same contract.
	MlRerunDispatch MlRerunDispatchFunc
	// RescanDispatch runs one externally ingested asset's pipeline
	// enqueue (the rescan worker's add and change arms): the seam
	// whose production wiring is the queue's own dispatcher (the
	// metadata, thumbnail and per-kind transcode jobs, the same
	// fan-out the upload handler performs). Production leaves it nil
	// and the queue wires its own dispatcher; the rescan tests inject
	// fakes of the same contract.
	RescanDispatch RescanDispatchFunc
	// FolderSyncInterval is the folder live sync's cadence: the
	// periodic directory-snapshot diff that keeps every directory
	// album's wall converged with its folder. Zero means the default
	// production window (60s); a negative value disables the periodic
	// registration entirely (the tests' determinism, and the
	// operator's off switch for a deployment that never references a
	// folder). The deployment knob is LUMEMOR_FOLDER_SYNC_INTERVAL;
	// the e2e harness shortens it so a scene observes the diff inside
	// the test's own bounds.
	FolderSyncInterval time.Duration
	// Places is the in-memory place grid the metadata job's GPS step
	// and the places re-resolution run resolve against, built once at
	// boot from the baked GeoNames data (geodata.Prepare). Nil is the
	// feature-off wiring (no reference data): the metadata job leaves
	// assets.place_id untouched and the places rerun enqueue refuses
	// to start.
	Places *geodata.Index
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
	// registry resolves the ML readiness seam's active models, pinned
	// at Start: EnqueueMlJobs consults it on every dispatch.
	registry *ml.Registry
	// places is the in-memory place grid (the bake's centroids, built
	// at boot); the places rerun enqueue refuses to start without it.
	places *geodata.Index
	// snapshots is the folder sync's baseline store, shared by the two
	// halves that touch it: the rescan workers record what their walks
	// saw, the periodic sweep compares against it.
	snapshots *AlbumSnapshots
	// logger is the queue's own diagnostics sink: the automatic model
	// fill's policy read and the missing-weights lines land here.
	logger *slog.Logger
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

	// The background work's budget: the gate every pipeline worker
	// consults at its start. The dial's concurrency lives here, so
	// the queue's own worker pools are sized to the mapping's
	// ceiling and the dial decides how many slots are actually open.
	budget := NewBudget(cfg.Pool, logger)

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
	metadataWorker := &MetadataWorker{
		pool:    cfg.Pool,
		bus:     cfg.Bus,
		dataDir: cfg.DataDir,
		places:  cfg.Places,
		logger:  logger,

		budget: budget,
	}
	river.AddWorker(workers, metadataWorker)
	thumbnailWorker := &ThumbnailWorker{
		pool:    cfg.Pool,
		bus:     cfg.Bus,
		dataDir: cfg.DataDir,
		logger:  logger,

		budget: budget,
	}
	river.AddWorker(workers, thumbnailWorker)
	transcodeWorker := &TranscodeWorker{
		pool:    cfg.Pool,
		bus:     cfg.Bus,
		dataDir: cfg.DataDir,
		logger:  logger,

		budget: budget,
	}
	river.AddWorker(workers, transcodeWorker)
	river.AddWorker(workers, &SweepWorker{
		pool:   cfg.Pool,
		logger: logger,
	})
	river.AddWorker(workers, &LivePhotoSweepWorker{
		pool:    cfg.Pool,
		dataDir: cfg.DataDir,
		bus:     cfg.Bus,
		logger:  logger,
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
	mlRerunWorker := &MlRerunWorker{
		pool:   cfg.Pool,
		bus:    cfg.Bus,
		logger: logger,
		budget: budget,
	}
	river.AddWorker(workers, mlRerunWorker)
	river.AddWorker(workers, &PlacesRerunWorker{
		pool:   cfg.Pool,
		bus:    cfg.Bus,
		places: cfg.Places,
		logger: logger,
		budget: budget,
	})
	river.AddWorker(workers, &RebuildWorker{
		pool:    cfg.Pool,
		bus:     cfg.Bus,
		dataDir: cfg.DataDir,
		logger:  logger,
		budget:  budget,
	})
	// The folder sync's baseline store: in memory, per process (see
	// AlbumSnapshots), shared by the rescan workers that record a
	// walk and the periodic sweep that compares against it.
	snapshots := NewAlbumSnapshots()
	rescanWorker := &AlbumRescanWorker{
		pool:      cfg.Pool,
		bus:       cfg.Bus,
		snapshots: snapshots,
		logger:    logger,
	}
	river.AddWorker(workers, rescanWorker)
	folderSyncWorker := &FolderSyncWorker{
		pool:      cfg.Pool,
		snapshots: snapshots,
		logger:    logger,
	}
	river.AddWorker(workers, folderSyncWorker)
	faceDetectWorker := &FaceDetectWorker{
		pool:     cfg.Pool,
		dataDir:  cfg.DataDir,
		registry: registry,
		logger:   logger,
		budget:   budget,
	}
	river.AddWorker(workers, faceDetectWorker)
	recognizeWorker := &FaceRecognizeWorker{
		pool:     cfg.Pool,
		dataDir:  cfg.DataDir,
		registry: registry,
		bus:      cfg.Bus,
		logger:   logger,
		budget:   budget,
	}
	river.AddWorker(workers, recognizeWorker)
	clipEmbedWorker := &ClipEmbedWorker{
		pool:     cfg.Pool,
		dataDir:  cfg.DataDir,
		registry: registry,
		bus:      cfg.Bus,
		logger:   logger,
		budget:   budget,
	}
	river.AddWorker(workers, clipEmbedWorker)
	ocrWorker := &OcrWorker{
		pool:     cfg.Pool,
		dataDir:  cfg.DataDir,
		registry: registry,
		logger:   logger,
		budget:   budget,
	}
	river.AddWorker(workers, ocrWorker)

	workerCount := cfg.Workers
	if workerCount <= 0 {
		workerCount = min(runtime.GOMAXPROCS(0), maxWorkers)
	}

	client, err := river.NewClient(driver, &river.Config{
		Queues: map[string]river.QueueConfig{
			river.QueueDefault: {MaxWorkers: workerCount},
			// The ML jobs' own conservative pool; see MlQueue.
			MlQueue: {MaxWorkers: mlQueueWorkers},
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
		PeriodicJobs: periodicJobs(folderSyncCadence(cfg.FolderSyncInterval)),
	})
	if err != nil {
		return nil, fmt.Errorf("jobs: create river client: %w", err)
	}

	queue := &Queue{
		client:    client,
		pool:      cfg.Pool,
		models:    models,
		registry:  registry,
		places:    cfg.Places,
		snapshots: snapshots,
		logger:    logger,
	}

	// Wire the ML readiness seam into the three pipeline workers: the
	// dispatch is the queue's EnqueueMlJobs method value, and the
	// queue exists only now that the client does. This runs BEFORE
	// the client starts -- no worker is executing yet, so the wiring
	// cannot race the first dispatch. Workers constructed directly
	// (the unit tests) carry no seam and the announce step skips it.
	// The detection worker's recognize continuation and the rerun
	// driver's real dispatch wire the same way: the recognition chain
	// fires when faces land and the recognize slot is active, and the
	// rerun dispatch (the queue's own dispatcher, unless the config
	// injected one) is the closed seam -- the four slots'
	// per-asset enqueues the rerun drivers wait on.
	metadataWorker.mlDispatch = queue.EnqueueMlJobs
	// The single-file motion photo's own pipeline: the extraction
	// creates the component row, and this seam queues that row's
	// metadata and transcode -- the exact pair the paired upload
	// queues for its component. Same wiring discipline: the queue
	// exists only now that the client does, and this runs before the
	// client starts.
	metadataWorker.motionDispatch = queue.enqueueMotionJobs
	thumbnailWorker.mlDispatch = queue.EnqueueMlJobs
	transcodeWorker.mlDispatch = queue.EnqueueMlJobs
	faceDetectWorker.recognizeDispatch = queue.EnqueueFaceRecognize
	// The automatic model fill: every inference worker's readiness gate
	// joins it behind a configured-but-missing slot, and the seam
	// reads the policy switch per attempt, so flipping the switch
	// takes effect on the next job without a restart.
	faceDetectWorker.modelFill = queue.ensureModelDownload
	recognizeWorker.modelFill = queue.ensureModelDownload
	clipEmbedWorker.modelFill = queue.ensureModelDownload
	ocrWorker.modelFill = queue.ensureModelDownload
	if cfg.MlRerunDispatch != nil {
		mlRerunWorker.dispatch = cfg.MlRerunDispatch
	} else {
		mlRerunWorker.dispatch = queue.dispatchMlRerun
	}
	// The rescan dispatch, the same wiring discipline: the config's
	// injection (the tests' fakes) or the queue's own fan-out. This
	// runs BEFORE the client starts -- no worker is executing yet,
	// so the wiring cannot race the first dispatch.
	if cfg.RescanDispatch != nil {
		rescanWorker.dispatch = cfg.RescanDispatch
	} else {
		rescanWorker.dispatch = queue.dispatchRescan
	}
	// The folder sync's enqueue: the periodic diff starts an album's
	// scan through the queue's own enqueue (the job_runs row and the
	// River job in one transaction), the same call the rescan endpoint
	// makes. Wired before the client starts, like every other seam.
	folderSyncWorker.enqueue = queue.EnqueueAlbumRescan

	if err := client.Start(context.WithoutCancel(ctx)); err != nil {
		return nil, fmt.Errorf("jobs: start river client: %w", err)
	}

	return queue, nil
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

// enqueueMotionJobs queues one extracted motion component's pipeline:
// its metadata row (duration and dimensions) and its transcode (the
// playable copy the pair's motion plays through). Failures are logged
// and swallowed -- the upload path's enqueue discipline: the row is
// committed, the requeue sweep is the recovery, and a failed enqueue
// must not fail the still's own job.
func (q *Queue) enqueueMotionJobs(ctx context.Context, assetID uuid.UUID) {
	if err := q.EnqueueMetadata(ctx, assetID); err != nil {
		q.logger.Error("enqueue the extracted motion's metadata job", "error", err, "asset_id", assetID)
	}
	if err := q.EnqueueTranscode(ctx, assetID); err != nil {
		q.logger.Error("enqueue the extracted motion's transcode job", "error", err, "asset_id", assetID)
	}
}

// Stop drains the queue gracefully: producers stop fetching new jobs,
// in-flight jobs finish, and after softStopTimeout their contexts are
// cancelled. ctx bounds the wait; on its expiry Stop returns the
// context's error and shutdown proceeds.
func (q *Queue) Stop(ctx context.Context) error {
	return q.client.Stop(ctx)
}
