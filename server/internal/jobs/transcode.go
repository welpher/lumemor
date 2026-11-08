package jobs

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"

	"github.com/welpher/lumemor/server/internal/db"
	dbgen "github.com/welpher/lumemor/server/internal/db/gen"
	"github.com/welpher/lumemor/server/internal/events"
	"github.com/welpher/lumemor/server/internal/media"
	"github.com/welpher/lumemor/server/internal/storage"
)

// TranscodeArgs identifies the asset whose playable copy the
// transcode job derives.
type TranscodeArgs struct {
	AssetID uuid.UUID `json:"assetId"`
}

// Kind is River's stable job kind identifier.
func (TranscodeArgs) Kind() string { return "transcode" }

// transcodeGate bounds how many transcode jobs may run their ffmpeg
// re-encode at once: the transcoding settings' concurrency, enforced
// across every worker of the process. The gate is RE-SIZED to the
// setting each job reads before acquiring, so a saved change governs
// the jobs that start after it (an in-flight encode finishes under
// the bound it started under -- a running ffmpeg subprocess is never
// preempted for a settings change). The process is single-binary and
// the workers share one pool, so package state is the honest scope.
var transcodeGate = &dynamicGate{}

// dynamicGate is a counting semaphore whose capacity can move: each
// acquirer states the size it knows (the setting's current value),
// the gate swaps its channel when the size changed, and acquirers
// hold the channel they acquired (a swap never strands a release).
// Shrinking below the in-flight count is eventual, not instant: the
// new channel fills, further jobs wait, the in-flight releases land
// on their own captured channel -- exactly the semantics a
// settings-driven bound needs.
type dynamicGate struct {
	mu   sync.Mutex
	size int
	sem  chan struct{}
}

// enter acquires a slot at the stated size and returns the release.
// The acquire is ctx-aware: a job queued past its timeout does not
// begin an encode whose ctx is already dead.
func (g *dynamicGate) enter(ctx context.Context, size int) func() {
	if size < 1 {
		size = 1
	}
	g.mu.Lock()
	if g.sem == nil || g.size != size {
		g.sem = make(chan struct{}, size)
		g.size = size
	}
	sem := g.sem
	g.mu.Unlock()
	select {
	case sem <- struct{}{}:
	case <-ctx.Done():
		// The release is a no-op: nothing was acquired.
		return func() {}
	}
	return func() { <-sem }
}

// TranscodeWorker runs the transcode job: the web-safe MP4 of a video
// asset -- H.264 + AAC, +faststart -- written atomically to
// storage/encoded/<assetID>/play.mp4. It serves standalone videos
// (the third job of their three-job pipeline) and, since M2d, the
// Live Photo component video (the ONE job its counter counts -- the
// component's playable stream is what the pair's motion plays
// through, its completion stamps ready_at but never announces). The
// pipeline completion is recorded through the same guarded decrement
// the metadata and thumbnail jobs use, and the completion that empties
// pending_jobs (three for a standalone video: metadata, thumbnail,
// transcode) publishes asset.ready through the shared completion step
// (announceAssetReady in face_detect.go), which the ML readiness seam
// hangs off.
type TranscodeWorker struct {
	river.WorkerDefaults[TranscodeArgs]

	pool       *pgxpool.Pool
	bus        *events.Bus
	dataDir    string
	logger     *slog.Logger
	mlDispatch mlDispatchFunc
}

// Timeout raises the job's execution envelope above River's one
// minute default and above the ffmpeg subprocess's own 120s budget:
// the re-encode alone may legitimately consume that whole budget, and
// the completion transaction adds to it. The 150s bound still leaves
// a wedged job retryable instead of pinning a worker slot forever.
func (w *TranscodeWorker) Timeout(job *river.Job[TranscodeArgs]) time.Duration {
	return 150 * time.Second
}

// Work executes one transcode job. The derivation (the ffmpeg
// subprocess, up to its 120s budget) runs before the database
// transaction opens: no connection is held while media is processed.
// The job reads the instance's transcoding settings AT RUN TIME --
// the contract's new-ingest-only pin: the profile the job sees is
// the profile it obeys, so a saved change governs every video whose
// job runs after it (videos already transcoded are never
// re-encoded).
func (w *TranscodeWorker) Work(ctx context.Context, job *river.Job[TranscodeArgs]) error {
	assetID := job.Args.AssetID

	row, err := dbgen.New(w.pool).GetAssetForMetadata(ctx, pgUUID(assetID))
	if errors.Is(err, pgx.ErrNoRows) {
		// The asset is gone (deleted while the job waited in the
		// queue): there is nothing to transcode and retrying cannot
		// change that, so the job completes rather than burning its
		// attempts.
		w.logger.Info("transcode job skipped: asset no longer exists", "asset_id", assetID)
		return nil
	}
	if err != nil {
		return fmt.Errorf("load asset: %w", err)
	}

	// A playable stream exists only for video-family rows. Both
	// enqueue paths (the upload handler, the requeue sweep) gate on
	// the kind, so a non-video row here is a mis-routed job, not a
	// retryable failure: completing it as a no-op (no derivation, no
	// decrement, no event) keeps the row's own pipeline untouched
	// instead of corrupting its counter or burning retries against
	// an image that can never transcode.
	if row.MediaKind != dbgen.MediaKindVideo {
		w.logger.Info("transcode job skipped: asset is not a video", "asset_id", assetID,
			"media_kind", string(row.MediaKind))
		return nil
	}

	// The settings read at run time (the new-ingest-only pin). A
	// corrupt row degrades to the documented default inside the
	// accessor; the logged error points at the row to fix.
	profile, settingsErr := db.TranscodingSettings(ctx, dbgen.New(w.pool))
	if settingsErr != nil {
		w.logger.Error("transcode job: transcoding settings are corrupt, using the default", "error", settingsErr)
	}

	// The `off` preset: no re-encode at all. The original stays in
	// place and downloadable (the original endpoint serves it), the
	// pipeline still completes below -- a video uploaded under `off`
	// is a stored original, not a pending derivation.
	if profile.Preset == "off" {
		w.logger.Info("transcode job skipped: the transcoding preset is off", "asset_id", assetID)
		return w.completeTranscode(ctx, assetID, row)
	}

	path, err := storage.ResolveStoredPath(w.dataDir, row.FilePath)
	if err != nil {
		return fmt.Errorf("resolve file of asset %s: %w", assetID, err)
	}

	// The concurrency bound: the gate holds the re-encode slot for
	// the duration of the ffmpeg subprocess alone -- no database
	// transaction is held inside it (the derivation runs before any
	// transaction opens, the standing discipline of this worker).
	release := transcodeGate.enter(ctx, profile.Concurrency)
	if err := media.GenerateTranscode(ctx, path, assetID.String(), presetHeight(profile.Preset)); err != nil {
		release()
		// The failure policy the settings pin: keepOriginal=true
		// completes the job quietly -- the original file stays in
		// place and downloadable through the original endpoint, the
		// video's pipeline still finishes, and the instance does not
		// burn retries against a source that fails to encode.
		// keepOriginal=false fails the job: River's retry policy
		// re-runs it.
		if profile.KeepOriginal {
			w.logger.Error("transcode failed; keeping the original and completing the job",
				"error", err, "asset_id", assetID)
			return w.completeTranscode(ctx, assetID, row)
		}
		return fmt.Errorf("derive playable copy of asset %s: %w", assetID, err)
	}
	release()

	return w.completeTranscode(ctx, assetID, row)
}

// completeTranscode records the pipeline completion of one video:
// the guarded decrement (idempotent on re-runs) and, when it empties
// the row's counter, the asset.ready announcement and the ML
// readiness seam -- the same steps the metadata and thumbnail jobs
// end with.
func (w *TranscodeWorker) completeTranscode(ctx context.Context, assetID uuid.UUID, row dbgen.GetAssetForMetadataRow) error {
	tx, err := w.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transcode transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := dbgen.New(tx)

	completion, err := queries.CompleteAssetJob(ctx, pgUUID(assetID))
	if errors.Is(err, pgx.ErrNoRows) {
		// The guarded decrement found nothing to decrement: either the
		// pipeline already completed (a re-run: idempotency -- no
		// double decrement, no second event) or the row counts no
		// pipeline of its own. Both are successful no-ops.
		completion = dbgen.CompleteAssetJobRow{BecameReady: false}
	} else if err != nil {
		return fmt.Errorf("complete pipeline job: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transcode transaction: %w", err)
	}

	if completion.BecameReady {
		// The shared completion step (the one site where pending_jobs
		// empties): the asset.ready announcement and the ML readiness
		// seam, after the commit so a subscriber that turns around
		// and reads the database sees the final state. A Live Photo
		// component row (M2d) never announces and never enters the
		// seam: its completion drains the one-job counter (the
		// ready_at stamp is also what lets the requeue sweep stop
		// matching the row), but the pair's announcement belongs to
		// the photo row alone -- a component id in the timeline's
		// event stream would be noise about an asset no view lists.
		announceAssetReady(ctx, w.bus, w.logger, w.mlDispatch, assetID, completion.MediaKind, row.IsComponent)
	}
	return nil
}

// presetHeight maps the transcoding settings' preset onto the
// re-encode's vertical-resolution bound: the numeric presets' own
// pixel bound, `auto` (and any unknown value) keeping the source
// resolution.
func presetHeight(preset string) int {
	switch preset {
	case "1080p":
		return 1080
	case "720p":
		return 720
	case "480p":
		return 480
	default:
		return 0
	}
}
