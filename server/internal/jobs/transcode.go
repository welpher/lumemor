package jobs

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"

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

	path, err := storage.ResolveStoredPath(w.dataDir, row.FilePath)
	if err != nil {
		return fmt.Errorf("resolve file of asset %s: %w", assetID, err)
	}

	if err := media.GenerateTranscode(ctx, path, assetID.String()); err != nil {
		return fmt.Errorf("derive playable copy of asset %s: %w", assetID, err)
	}

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
