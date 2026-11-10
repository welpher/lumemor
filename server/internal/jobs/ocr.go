// The OCR job: the asset-ready seam's ocr arm and the
// ocr rerun's per-asset unit. The job resolves the active OCR model's
// TWO graphs from its model directory (the registry's multi-file
// convention: the manifest's first file is the detection graph, the
// second the recognition graph), decodes the stored original upright,
// runs the pipeline's Extract, and upserts the text into ocr_texts --
// keyed asset_id, so a re-run overwrites and a rerun converges. The
// language column stays NULL: the default group's model reads a mixed
// Chinese-English corpus, and NULL is the honest statement of that.
//
// Failure discipline mirrors the other ML jobs: the digest gates and
// the decode run before any database work, a failure anywhere leaves
// the table exactly as it was, and River's retry owns the recovery.
// The job never touches pending_jobs.
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
	"github.com/welpher/lumemor/server/internal/media"
	"github.com/welpher/lumemor/server/internal/ml"
	mlocr "github.com/welpher/lumemor/server/internal/ml/ocr"
	"github.com/welpher/lumemor/server/internal/storage"
)

// OcrKind pins the River job kind of the OCR job.
const OcrKind = "ocr"

// ocrTimeout is the job's execution envelope: the detection over the
// bounded geometry plus one recognition per text box -- a few seconds
// on CPU for a text-heavy photo; the bound keeps a wedged job
// retryable.
const ocrTimeout = 120 * time.Second

// ocrWorkingLongEdge bounds the decoded raster the job hands the
// pipeline: the detector's own input bound is the max side of 960, so
// the bound keeps small text legible through the decode of a large
// original without a full-resolution raster crossing the libvips
// boundary.
const ocrWorkingLongEdge = 1280

// OcrArgs identifies the asset whose text the job extracts, and --
// when the rerun driver dispatched it -- the run the completion
// reports into (uuid.Nil for a seam-dispatched job).
type OcrArgs struct {
	AssetID  uuid.UUID `json:"assetId"`
	JobRunID uuid.UUID `json:"jobRunId"`
}

// Kind is River's stable job kind identifier.
func (OcrArgs) Kind() string { return OcrKind }

// OcrWorker runs the OCR job.
type OcrWorker struct {
	river.WorkerDefaults[OcrArgs]

	pool     *pgxpool.Pool
	dataDir  string
	registry *ml.Registry
	logger   *slog.Logger
}

// Timeout is the job's execution envelope; see ocrTimeout.
func (w *OcrWorker) Timeout(job *river.Job[OcrArgs]) time.Duration {
	return ocrTimeout
}

// Work executes one OCR job and, when the rerun driver dispatched it,
// reports the outcome into its run's row -- the rerun completion
// counting (see ml_rerun.go).
func (w *OcrWorker) Work(ctx context.Context, job *river.Job[OcrArgs]) error {
	err := w.work(ctx, job)
	if runID := job.Args.JobRunID; runID != uuid.Nil {
		reportMlRunCompletion(ctx, w.pool, w.logger, runID, err != nil)
	}
	return err
}

// work is the job's core: the gates, the pipeline, and the upsert.
func (w *OcrWorker) work(ctx context.Context, job *river.Job[OcrArgs]) error {
	assetID := job.Args.AssetID

	row, err := dbgen.New(w.pool).GetAssetForMetadata(ctx, pgUUID(assetID))
	if errors.Is(err, pgx.ErrNoRows) {
		w.logger.Info("ocr job skipped: asset no longer exists", "asset_id", assetID)
		return nil
	}
	if err != nil {
		return fmt.Errorf("load asset: %w", err)
	}
	if row.MediaKind != dbgen.MediaKindImage {
		w.logger.Info("ocr job skipped: asset is not an image", "asset_id", assetID,
			"media_kind", string(row.MediaKind))
		return nil
	}

	model, err := w.registry.Active(ml.SlotOcr)
	if errors.Is(err, ml.ErrNoActiveModel) {
		w.logger.Info("ocr job skipped: no active ocr model", "asset_id", assetID)
		return nil
	}
	if err != nil {
		return fmt.Errorf("resolve the active ocr model: %w", err)
	}

	// The model's two graphs: the manifest's first file is the
	// detection graph (the models row's registered primary), the
	// second the recognition graph.
	files, err := ml.ModelFiles(model)
	if err != nil {
		return fmt.Errorf("resolve the files of the ocr model %s: %w", model.ID, err)
	}
	if len(files) != 2 {
		return fmt.Errorf("the ocr model %s pins %d weights files, want the detection and recognition pair", model.ID, len(files))
	}

	// The digest gates run before the decode: a drifted model fails
	// fast.
	detSess, err := ml.NewSession(files[0].Path, files[0].SHA256)
	if err != nil {
		return fmt.Errorf("load the ocr detection graph: %w", err)
	}
	defer func() { _ = detSess.Close() }()
	recSess, err := ml.NewSession(files[1].Path, files[1].SHA256)
	if err != nil {
		return fmt.Errorf("load the ocr recognition graph: %w", err)
	}
	defer func() { _ = recSess.Close() }()

	path, err := storage.ResolveAssetPath(w.dataDir, row.FilePath, string(row.StorageKind))
	if err != nil {
		return fmt.Errorf("resolve file of asset %s: %w", assetID, err)
	}
	img, err := media.DecodeInferenceImage(path, ocrWorkingLongEdge)
	if err != nil {
		return fmt.Errorf("decode asset %s for ocr: %w", assetID, err)
	}

	text, err := mlocr.NewPipeline(detSess, recSess).ExtractWithParams(img, model.Params)
	if err != nil {
		return fmt.Errorf("extract the text of asset %s: %w", assetID, err)
	}

	if err := dbgen.New(w.pool).UpsertOcrText(ctx, dbgen.UpsertOcrTextParams{
		AssetID: pgUUID(assetID),
		ModelID: model.ID,
		Text:    text,
	}); err != nil {
		return fmt.Errorf("store the text of asset %s: %w", assetID, err)
	}

	w.logger.Info("ocr complete", "asset_id", assetID,
		"model_id", model.ID, "chars", len(text))
	return nil
}

// EnqueueOcr inserts the OCR job for the asset on the ml queue (the
// ML jobs' conservative worker pool). It must be called after the
// pipeline completed; re-enqueueing runs the job again idempotently --
// the upsert overwrites.
func (q *Queue) EnqueueOcr(ctx context.Context, assetID uuid.UUID) error {
	if _, err := q.client.Insert(ctx, OcrArgs{AssetID: assetID}, &river.InsertOpts{Queue: MlQueue}); err != nil {
		return fmt.Errorf("insert ocr job: %w", err)
	}
	return nil
}
