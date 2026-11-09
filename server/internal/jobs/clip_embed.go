// The CLIP embedding job (M3a Task 8): the asset-ready seam's clip
// arm and the clip rerun's per-asset unit. The job decodes the stored
// original upright (the shared inference decode), runs the active
// clip-embed model's vision tower over it, and upserts the embedding
// into clip_embeddings -- keyed (asset_id, model_id), so a re-run
// overwrites and a rerun converges, while every other model's rows
// survive untouched (the model-switch coexistence policy).
//
// THE INSERT GUARD: the embedding's length must equal the
// clip_embeddings column's CURRENT dimension, read from the database
// catalog at run time -- the insert-side half of the dimension policy
// (Review Focus #5): a drifted state (an active model whose dimension
// disagrees with the column, an activation that bypassed the
// cross-dimension ALTER) fails the job BEFORE any insert, with zero
// rows landing, instead of letting the database reject the vector
// mid-transaction.
//
// Failure discipline mirrors the face jobs: the digest gate and the
// decode run before any database work, a failure anywhere leaves the
// table exactly as it was, and River's retry owns the recovery. The
// job never touches pending_jobs -- the asset was ready before the ML
// chain began.
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
	"github.com/pgvector/pgvector-go"
	"github.com/riverqueue/river"

	dbgen "github.com/welpher/lumemor/server/internal/db/gen"
	"github.com/welpher/lumemor/server/internal/events"
	"github.com/welpher/lumemor/server/internal/media"
	"github.com/welpher/lumemor/server/internal/ml"
	mlclip "github.com/welpher/lumemor/server/internal/ml/clip"
	"github.com/welpher/lumemor/server/internal/storage"
)

// ClipEmbedKind pins the River job kind of the CLIP embedding job.
const ClipEmbedKind = "clip_embed"

// clipEmbedTimeout is the job's execution envelope: the 224x224-input
// tower over the bounded decode is well under a second on CPU; the
// bound absorbs a pathological original and keeps a wedged job
// retryable.
const clipEmbedTimeout = 60 * time.Second

// clipWorkingLongEdge bounds the decoded raster the job hands the
// wrapper: the tower resizes to its declared 224x224 (or the raster's
// own size through a dynamic graph), so the bound only keeps the
// decode of a large original from handing a full-resolution raster
// across the libvips boundary.
const clipWorkingLongEdge = 1280

// ClipEmbedArgs identifies the asset whose CLIP embedding the job
// computes, and -- when the rerun driver dispatched it -- the run the
// completion reports into (uuid.Nil for a seam-dispatched job).
type ClipEmbedArgs struct {
	AssetID  uuid.UUID `json:"assetId"`
	JobRunID uuid.UUID `json:"jobRunId"`
}

// Kind is River's stable job kind identifier.
func (ClipEmbedArgs) Kind() string { return ClipEmbedKind }

// ClipEmbedWorker runs the CLIP embedding job.
type ClipEmbedWorker struct {
	river.WorkerDefaults[ClipEmbedArgs]

	pool     *pgxpool.Pool
	dataDir  string
	registry *ml.Registry
	bus      *events.Bus
	logger   *slog.Logger
}

// Timeout is the job's execution envelope; see clipEmbedTimeout.
func (w *ClipEmbedWorker) Timeout(job *river.Job[ClipEmbedArgs]) time.Duration {
	return clipEmbedTimeout
}

// Work executes one CLIP embedding job and, when the rerun driver
// dispatched it, reports the outcome into its run's row -- the rerun
// completion counting (see ml_rerun.go). The report runs detached from
// the job context so it lands even when the envelope consumed the
// context's last moments.
func (w *ClipEmbedWorker) Work(ctx context.Context, job *river.Job[ClipEmbedArgs]) error {
	err := w.work(ctx, job)
	if runID := job.Args.JobRunID; runID != uuid.Nil {
		reportMlRunCompletion(ctx, w.pool, w.logger, runID, err != nil)
	}
	return err
}

// work is the job's core: the gates, the inference, and the guarded
// upsert.
func (w *ClipEmbedWorker) work(ctx context.Context, job *river.Job[ClipEmbedArgs]) error {
	assetID := job.Args.AssetID

	row, err := dbgen.New(w.pool).GetAssetForMetadata(ctx, pgUUID(assetID))
	if errors.Is(err, pgx.ErrNoRows) {
		// The asset is gone: there is nothing to embed, and retrying
		// cannot change that.
		w.logger.Info("clip embed job skipped: asset no longer exists", "asset_id", assetID)
		return nil
	}
	if err != nil {
		return fmt.Errorf("load asset: %w", err)
	}
	if row.MediaKind != dbgen.MediaKindImage {
		// Only image assets carry embeddings: a non-image row is a
		// mis-routed job, not a retryable failure.
		w.logger.Info("clip embed job skipped: asset is not an image", "asset_id", assetID,
			"media_kind", string(row.MediaKind))
		return nil
	}

	// The slot's active model. An unfilled slot is the degraded mode:
	// a stale or mis-routed job completes silently.
	model, err := w.registry.Active(ml.SlotClipEmbed)
	if errors.Is(err, ml.ErrNoActiveModel) {
		w.logger.Info("clip embed job skipped: no active clip model", "asset_id", assetID)
		return nil
	}
	if err != nil {
		return fmt.Errorf("resolve the active clip-embed model: %w", err)
	}

	// The session load's digest gate runs before the decode: a drifted
	// model fails fast.
	sess, err := ml.NewSession(model.Path, model.SHA256)
	if err != nil {
		return fmt.Errorf("load the clip model: %w", err)
	}
	defer func() { _ = sess.Close() }()

	path, err := storage.ResolveAssetPath(w.dataDir, row.FilePath, string(row.StorageKind))
	if err != nil {
		return fmt.Errorf("resolve file of asset %s: %w", assetID, err)
	}
	img, err := media.DecodeInferenceImage(path, clipWorkingLongEdge)
	if err != nil {
		return fmt.Errorf("decode asset %s for embedding: %w", assetID, err)
	}

	embedding, err := mlclip.EmbedImage(sess, img)
	if err != nil {
		return fmt.Errorf("embed asset %s: %w", assetID, err)
	}

	// The insert guard: the embedding's length against the column's
	// CURRENT dimension, read from the catalog (the same read the
	// activation transaction's cross-dimension switch makes). A
	// disagreement fails the job with zero rows -- the honest refusal
	// of a drifted state the activation should have re-dimensioned.
	columnDim, err := ml.EmbeddingColumnDim(ctx, w.pool)
	if err != nil {
		return fmt.Errorf("read the clip_embeddings column dimension: %w", err)
	}
	if columnDim != len(embedding) {
		return fmt.Errorf("clip embed job of asset %s: the active model %s emits %d-dimensional embeddings but the clip_embeddings column carries vector(%d); refusing the insert (activate the model through the registry to re-dimension the column)",
			assetID, model.ID, len(embedding), columnDim)
	}

	if err := dbgen.New(w.pool).UpsertClipEmbedding(ctx, dbgen.UpsertClipEmbeddingParams{
		AssetID:   pgUUID(assetID),
		ModelID:   model.ID,
		Embedding: pgvector.NewVector(embedding),
	}); err != nil {
		return fmt.Errorf("store the embedding of asset %s: %w", assetID, err)
	}

	// THE LATE-EMBED CONVERGENCE (2026-11-10, the m3-search flake's
	// second strike -- the structural gap this closes): the job lands
	// AFTER the asset went ready (the M3a seam: pending_jobs never
	// counted the ML), so the search page's live surfaces never
	// re-executed -- a semantic hit that arrived late never appeared
	// until some OTHER asset event or a manual reload. The upsert is
	// the transition (a new row or a rerun's overwrite both change
	// what a search would answer), so it publishes asset.updated and
	// the established convergence (the refetch-on-event, burst
	// collapsed) picks the new answer up.
	w.bus.Publish("asset.updated", map[string]string{
		"id": assetID.String(),
	})

	w.logger.Info("clip embedding complete", "asset_id", assetID,
		"model_id", model.ID, "dims", len(embedding))
	return nil
}

// EnqueueClipEmbed inserts the CLIP embedding job for the asset on
// the ml queue (the ML jobs' conservative worker pool). It must be
// called after the pipeline completed; re-enqueueing runs the job
// again idempotently -- the upsert overwrites.
func (q *Queue) EnqueueClipEmbed(ctx context.Context, assetID uuid.UUID) error {
	if _, err := q.client.Insert(ctx, ClipEmbedArgs{AssetID: assetID}, &river.InsertOpts{Queue: MlQueue}); err != nil {
		return fmt.Errorf("insert clip embed job: %w", err)
	}
	return nil
}
