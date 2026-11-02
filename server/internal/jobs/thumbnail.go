package jobs

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"

	dbgen "github.com/welpher/lumemor/server/internal/db/gen"
	"github.com/welpher/lumemor/server/internal/events"
	"github.com/welpher/lumemor/server/internal/media"
	"github.com/welpher/lumemor/server/internal/storage"
)

// ThumbnailArgs identifies the asset whose serving copies the
// thumbnail job derives.
type ThumbnailArgs struct {
	AssetID uuid.UUID `json:"assetId"`
}

// Kind is River's stable job kind identifier.
func (ThumbnailArgs) Kind() string { return "thumbnail" }

// ThumbnailWorker runs the thumbnail job: the serving copies of the
// asset's content -- three AVIF sizes for an image, a JPEG poster
// frame for a video -- and the blurhash of the small tile stored on
// the asset row. The pipeline completion is recorded through the same
// guarded decrement the metadata job uses, so the completion that
// empties pending_jobs publishes asset.ready exactly once.
type ThumbnailWorker struct {
	river.WorkerDefaults[ThumbnailArgs]

	pool    *pgxpool.Pool
	bus     *events.Bus
	dataDir string
	logger  *slog.Logger
}

// Timeout raises the job's execution envelope above River's one
// minute default: the ffmpeg poster extraction alone may legitimately
// consume its full 60s budget, and the AVIF encodes of a large image
// add to that. The 90s bound still leaves a wedged job retryable
// instead of pinning a worker slot forever.
func (w *ThumbnailWorker) Timeout(job *river.Job[ThumbnailArgs]) time.Duration {
	return 90 * time.Second
}

// Work executes one thumbnail job. The derivation (libvips or the
// ffmpeg subprocess, up to its 60s budget) runs before the database
// transaction opens: no connection is held while media is processed.
func (w *ThumbnailWorker) Work(ctx context.Context, job *river.Job[ThumbnailArgs]) error {
	assetID := job.Args.AssetID

	row, err := dbgen.New(w.pool).GetAssetForMetadata(ctx, pgUUID(assetID))
	if errors.Is(err, pgx.ErrNoRows) {
		// The asset is gone (deleted while the job waited in the
		// queue): there is nothing to derive and retrying cannot
		// change that, so the job completes rather than burning its
		// attempts.
		w.logger.Info("thumbnail job skipped: asset no longer exists", "asset_id", assetID)
		return nil
	}
	if err != nil {
		return fmt.Errorf("load asset: %w", err)
	}

	path, err := storage.ResolveStoredPath(w.dataDir, row.FilePath)
	if err != nil {
		return fmt.Errorf("resolve file of asset %s: %w", assetID, err)
	}

	// Image assets get the three AVIF sizes; video assets get the
	// poster frame instead -- never AVIF thumbnails (the poster IS the
	// video's thumbnail, served by the thumbnail endpoint; the
	// playable stream is the transcode job's product).
	var hash string
	if row.MediaKind == dbgen.MediaKindVideo {
		hash, err = media.GeneratePoster(ctx, path, assetID.String())
	} else {
		hash, err = media.GenerateThumbnails(path, assetID.String())
	}
	if err != nil {
		return fmt.Errorf("derive serving copies of asset %s: %w", assetID, err)
	}

	tx, err := w.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin thumbnail transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := dbgen.New(tx)

	if err := queries.UpdateAssetBlurhash(ctx, dbgen.UpdateAssetBlurhashParams{
		ID:       pgUUID(assetID),
		Blurhash: pgtype.Text{String: hash, Valid: true},
	}); err != nil {
		return fmt.Errorf("store blurhash: %w", err)
	}
	// The component guard mirrors the metadata job's: a Live Photo
	// component row's counter counts its transcode alone (M2d), so a
	// thumbnail completion (no enqueue path produces one for a
	// component, but a stale or mis-routed job can reach here) must
	// not consume the count or announce on its behalf.
	completion := dbgen.CompleteAssetJobRow{BecameReady: false}
	if !row.IsComponent {
		completion, err = queries.CompleteAssetJob(ctx, pgUUID(assetID))
		if errors.Is(err, pgx.ErrNoRows) {
			// The guarded decrement found nothing to decrement: either the
			// pipeline already completed (a re-run: idempotency -- no
			// double decrement, no second event), or the row counts no
			// pipeline of its own. Both are successful no-ops.
			completion = dbgen.CompleteAssetJobRow{BecameReady: false}
		} else if err != nil {
			return fmt.Errorf("complete pipeline job: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit thumbnail transaction: %w", err)
	}

	if completion.BecameReady {
		// Published after the commit so a subscriber that turns around
		// and reads the database sees the final state. The payload
		// discipline mirrors asset.uploaded and the metadata job's
		// completion: the resource id and its media kind, nothing more.
		w.bus.Publish("asset.ready", map[string]string{
			"assetId": assetID.String(),
			"kind":    string(completion.MediaKind),
		})
	}
	return nil
}
