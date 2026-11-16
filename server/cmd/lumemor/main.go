// Command lumemor runs the lumemor server.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/welpher/lumemor/server/internal/api"
	"github.com/welpher/lumemor/server/internal/buildinfo"
	"github.com/welpher/lumemor/server/internal/config"
	"github.com/welpher/lumemor/server/internal/db"
	dbgen "github.com/welpher/lumemor/server/internal/db/gen"
	"github.com/welpher/lumemor/server/internal/events"
	"github.com/welpher/lumemor/server/internal/geodata"
	"github.com/welpher/lumemor/server/internal/jobs"
	"github.com/welpher/lumemor/server/internal/media"
	"github.com/welpher/lumemor/server/internal/ml"
	"github.com/welpher/lumemor/server/internal/storage"
	"github.com/welpher/lumemor/server/internal/trash"
	"github.com/welpher/lumemor/server/internal/updatecheck"
)

const (
	// shutdownTimeout bounds how long in-flight requests may finish after a
	// termination signal before the server exits anyway.
	shutdownTimeout = 10 * time.Second

	// dbConnectWait bounds how long startup waits for the database to
	// become reachable, tolerating the race against a freshly launched
	// database container.
	dbConnectWait = 60 * time.Second

	// eventRetention is the activity ring window pinned by the API
	// contract: GET /api/v1/events replays this much history on
	// reconnect.
	eventRetention = 5 * time.Minute

	// queueStopTimeout bounds the job queue's graceful drain after the
	// HTTP server stopped. It sits above the queue's internal soft stop
	// timeout (which itself sits above the pipeline's largest job
	// envelope, the thumbnail job's 90s) so a healthy job always
	// finishes; anything wedged beyond it is dropped back to the queue
	// and retried on the next start.
	queueStopTimeout = 110 * time.Second

	// envModelsHfMirror overrides the HF domain mirror the model
	// download rewrites huggingface.co hostnames to (restricted
	// networks; e.g. "hf-mirror.com"). Empty or unset = direct
	// connections; the admin settings' mirror base, when configured,
	// wins over this deployment-level fallback.
	envModelsHfMirror = "LUMEMOR_MODELS_HF_MIRROR"

	// envModelsBundle points the boot at an offline model bundle
	// (models-<version>.tar with its checksums.txt), the no-network
	// first deployment's path into the same registry the online
	// download writes. Empty or unset = no import.
	envModelsBundle = "LUMEMOR_MODELS_BUNDLE"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	if err := run(logger); err != nil {
		logger.Error("server exited with error", "error", err)
		os.Exit(1)
	}
}

// run executes the server lifecycle: load config, prepare the storage
// layout, connect to the database and apply migrations, start the HTTP
// server, then shut down gracefully on SIGINT/SIGTERM.
func run(logger *slog.Logger) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	// Fail fast on unusable storage mounts: the layout must exist and
	// upload staging must share a filesystem with the originals
	// directory so uploads can finalize with an atomic rename.
	if err := storage.EnsureLayout(cfg.DataDir, cfg.ConfigDir); err != nil {
		return err
	}
	logger.Info("storage layout verified", "data_dir", cfg.DataDir, "config_dir", cfg.ConfigDir)

	// libvips starts before anything can use it (the thumbnail job's
	// workers) and shuts down after the job queue has drained: the
	// deferred Shutdown therefore runs last, behind the queue's
	// explicit Stop at the end of run. Startup is one-shot per
	// process -- govips cannot restart it afterwards. The media
	// package owns the libvips seam so this file carries no direct
	// CGO import.
	if err := media.Startup(); err != nil {
		return fmt.Errorf("start libvips: %w", err)
	}
	defer media.Shutdown()
	logger.Info("libvips started", "version", media.LibvipsVersion())

	// Boot-time sweep of the upload staging directory: a single
	// process just started, so nothing can be in flight and anything
	// left in tmp/ is crash debris of the previous run.
	if err := storage.SweepTmp(cfg.DataDir); err != nil {
		return err
	}
	logger.Info("upload staging swept")

	pool, err := db.Connect(ctx, cfg.DatabaseURL, dbConnectWait)
	if err != nil {
		return err
	}
	defer pool.Close()

	if err := db.Migrate(ctx, pool); err != nil {
		return err
	}
	logger.Info("schema migrations applied")

	// The first-run predicate mirrors how NewRouter builds its internal
	// server struct: the sqlc queries over the pool. It derives from the
	// users table, so a restarted server routes the same way; an
	// unreachable database falls back to the wizard, whose own endpoints
	// surface the real errors.
	queries := dbgen.New(pool)

	// The event bus is the SSE activity stream's backbone and the
	// plugin seam: handlers publish domain events (the session
	// lifecycle) and every connected stream follows them.
	bus := events.NewBus(eventRetention, time.Now)

	// The place reference plane: the baked GeoNames data loads
	// and imports here, idempotently -- the data-version marker skips
	// the import on an unchanged dataset, so the normal reboot only
	// reads the rows back. The in-memory grid the metadata job's GPS
	// step and the places rerun resolve against builds from the
	// database's rows, so the resolver can only answer ids the places
	// table actually carries. An empty LUMEMOR_GEODATA_DIR (the
	// local-development default) keeps the feature honestly absent;
	// a configured but unreadable directory is a deployment error the
	// boot surfaces.
	placeIndex, geoImported, err := geodata.Prepare(ctx, pool, cfg.GeodataDir)
	if err != nil {
		return fmt.Errorf("prepare the place reference data: %w", err)
	}
	if placeIndex != nil {
		logger.Info("place reference data ready",
			"places", placeIndex.Len(), "imported", geoImported, "geodata_dir", cfg.GeodataDir)
	}

	// The model store: the ML models live under the instance
	// configuration root, each model in its own directory. The registry
	// over it is the instance's SHARED registry -- the job queue's
	// download job registers and activates through it, and the later
	// admin surface consumes the same instance.
	modelsDir := filepath.Join(cfg.ConfigDir, "models")
	if err := os.MkdirAll(modelsDir, 0o755); err != nil {
		return fmt.Errorf("prepare the models dir: %w", err)
	}
	registry := ml.NewRegistry(pool, queries, modelsDir)

	// The offline bundle import (the no-network first deployment's
	// path): a configured bundle path is unpacked, digest-verified
	// and registered here, before the queue starts -- so a boot
	// carrying a bundle lands its models the same way an online
	// download would, and the slots it filled are simply active. A
	// failed import does not fail the boot: the instance comes up
	// with the honest unfilled slots, the same state a network-less
	// boot without a bundle produces.
	if bundle := os.Getenv(envModelsBundle); bundle != "" {
		report, err := ml.ImportBundle(ctx, bundle, modelsDir, registry, queries)
		if err != nil {
			logger.Error("offline model bundle import failed", "error", err, "bundle", bundle)
		} else {
			logger.Info("offline model bundle imported",
				"bundle", bundle, "members", report.Members, "models", report.ModelsDone)
		}
	}

	// The job queue starts only after the schema exists: River applies
	// its own table migrations here (river_job and friends live outside
	// the goose schema by design), then the workers begin consuming.
	// Its lifecycle is owned by the explicit Stop below, not by this
	// context -- a cancelled context would be a hard stop.
	queue, err := jobs.Start(ctx, jobs.Config{
		Pool:           pool,
		Bus:            bus,
		DataDir:        cfg.DataDir,
		Logger:         logger,
		ModelsDir:      modelsDir,
		ModelsHfMirror: os.Getenv(envModelsHfMirror),
		Registry:       registry,
		// The place grid the metadata job's GPS step and the places
		// rerun resolve against (nil without the baked place data).
		Places: placeIndex,
		// The folder live sync's cadence: the deployment's own window
		// (60s unless configured), the e2e harness's shortened one, or
		// the disabled marker.
		FolderSyncInterval: cfg.FolderSyncInterval,
	})
	if err != nil {
		return err
	}
	logger.Info("job queue started", "models_dir", modelsDir, "models_hf_mirror", os.Getenv(envModelsHfMirror))

	// The boot requeue sweep: the recovery of the pipeline assets stuck
	// by a previous run (an enqueue insert failure, or a crash in the
	// commit-to-enqueue window). It runs after the queue started -- the
	// re-enqueued jobs are picked up immediately -- and before the HTTP
	// server begins listening, so the first client request already sees
	// the recovered state; the cost on a healthy database is one SELECT
	// plus the re-inserts of the rare genuinely stuck row. Its failure
	// does not fail the boot: the sweep is a recovery mechanism, the
	// server is fully usable without it, and the hourly periodic sweep
	// retries.
	sweepCounts, err := jobs.RunRequeueSweep(ctx, pool, queue)
	if err != nil {
		// The counts ride along even on the aborted run (partial on a
		// mid-sweep enqueue failure; zero when the listing itself
		// failed), mirroring the worker's failure line.
		logger.Error("requeue sweep failed", "error", err,
			"stuck", sweepCounts.Stuck, "enqueued", sweepCounts.Enqueued)
	} else {
		// The run's single log line; per-asset logs are deliberately
		// absent.
		logger.Info("requeue sweep complete",
			"stuck", sweepCounts.Stuck, "enqueued", sweepCounts.Enqueued)
	}

	// The boot trash expiry, the same pattern as the sweep: rows whose
	// retention closed while the process was down leave with the boot
	// instead of waiting for the first hourly periodic run. It is
	// bounded (the run caps its own batch) and idempotent, its failure
	// does not fail the boot, and the hourly periodic job retries.
	expiryCounts, err := trash.RunExpiry(ctx, pool, cfg.DataDir, bus)
	if err != nil {
		logger.Error("trash expiry failed", "error", err,
			"expired", expiryCounts.Expired, "files", expiryCounts.FilesRemoved)
	} else {
		logger.Info("trash expiry complete",
			"expired", expiryCounts.Expired, "files", expiryCounts.FilesRemoved)
	}

	// The release check: the instance's one scheduled outbound probe.
	// It starts with the server and follows the same signal context,
	// so a shutdown cancels a probe in flight; the answer it already
	// persisted survives. A deployment that must stay off the network
	// turns it off through the environment, independently of the
	// admin settings switch.
	releaseChecker := updatecheck.New(updatecheck.Options{
		Queries:  queries,
		Logger:   logger,
		Version:  buildinfo.Version,
		Disabled: cfg.UpdateCheckDisabled,
	})
	go releaseChecker.Run(ctx)
	logger.Info("release check started", "disabled", cfg.UpdateCheckDisabled, "version", buildinfo.Version)

	srv := &http.Server{
		Addr: cfg.Listen,
		Handler: api.NewRouter(&api.Deps{
			Pool:    pool,
			Logger:  logger,
			Version: buildinfo.Version,
			Events:  bus,
			// The upload endpoint stages and finalizes files under the
			// data root and honors the configured per-file size limit.
			DataDir:          cfg.DataDir,
			UploadLimitBytes: cfg.UploadLimitBytes,
			// The upload handler enqueues the metadata job of every
			// committed asset through the queue.
			Jobs: queue,
			// The download endpoint enqueues the default model group's
			// download through the same queue.
			ModelDownloads: queue,
			// The model management endpoints share the registry and
			// its models dir with the job queue, and the rerun
			// endpoint enqueues through the same queue.
			Models:    registry,
			ModelsDir: modelsDir,
			MlReruns:  queue,
			// The source pipeline's host check reads the same
			// allowed-hosts list the downloader's transport gate
			// reads: one settings document owns both.
			ProbeHosts: func() []string {
				settings, err := db.ModelSettingsOf(context.Background(), queries)
				if err != nil {
					logger.Warn("read the model download sources; answering the defaults", "error", err)
				}
				return settings.AllowedHosts
			},
			// The places re-resolution endpoint enqueues through the
			// same queue.
			PlacesReruns: queue,
			// The manual location edit resolves its place against the
			// same in-memory grid the metadata job's GPS step uses.
			Places: placeIndex,
			// The derivative rebuild endpoint enqueues through the
			// same queue.
			DerivativesRebuilds: queue,
			// The album rescan endpoint enqueues through the same
			// queue.
			AlbumRescans: queue,
			// The release check face reads and runs the checker the
			// schedule above drives.
			UpdateChecks: releaseChecker,
			Initialized: func(ctx context.Context) bool {
				admins, err := queries.CountAdmins(ctx)
				if err != nil {
					logger.Error("count admins for root routing", "error", err)
					return false
				}
				return admins > 0
			},
		}),
		ReadHeaderTimeout: 10 * time.Second,
	}

	serveErr := make(chan error, 1)
	go func() {
		logger.Info("server listening", "addr", cfg.Listen)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serveErr <- err
		}
	}()

	select {
	case err := <-serveErr:
		return err
	case <-ctx.Done():
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return err
	}
	logger.Info("server stopped")

	// Drain the job queue after the HTTP surface went quiet: no new
	// enqueues can arrive, in-flight jobs finish, and the bounded wait
	// keeps a wedged job from stalling the shutdown forever (the queue
	// cancels stragglers itself after its soft stop timeout).
	queueCtx, queueCancel := context.WithTimeout(context.Background(), queueStopTimeout)
	defer queueCancel()
	if err := queue.Stop(queueCtx); err != nil {
		logger.Error("drain job queue", "error", err)
	}
	logger.Info("job queue stopped")
	return nil
}
