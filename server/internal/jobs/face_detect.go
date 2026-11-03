// The face detection job (M3a Task 6): the asset-ready seam's first
// real arm. The pipeline jobs complete an image asset, the shared
// completion step (announceAssetReady below, wired into the metadata,
// thumbnail and transcode workers by queue.go) consults the
// slot-activation matrix in EnqueueMlJobs, and an active detect model
// gains the asset a face_detect job here. The job decodes the stored
// original upright (media's shared inference decode: EXIF orientation
// applied, long edge bounded), runs the committed-weights session over
// it, and replaces the asset's faces OF THAT MODEL -- the model-switch
// coexistence policy: other models' rows survive, queries filter by
// the active model, and switching back revives them without
// recomputing.
//
// Failure discipline: the job holds the model's session and the
// asset's rows only inside its own transaction -- a failure anywhere
// (the digest gate, the decode, the inference) leaves the asset's
// readiness exactly as the pipeline left it, and River's retry owns
// the recovery. The ML jobs deliberately never touch pending_jobs:
// readiness means the serving copies exist.
//
// THE CHAINING DECISION (pinned, handed to Task 7): the detection
// completion enqueues NOTHING. The recognition job's trigger surface
// is the model rerun plus Task 7's own incremental-trigger decision,
// which that task owns and must justify on its own evidence. There is
// no automatic detection-to-recognition chain in this milestone by
// design: a fresh upload's faces sit detected-but-unrecognized until
// the user's rerun or Task 7's deliberate trigger.
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
	"github.com/welpher/lumemor/server/internal/ml"
	mlface "github.com/welpher/lumemor/server/internal/ml/face"
	"github.com/welpher/lumemor/server/internal/storage"
)

// FaceDetectKind pins the River job kind of the face detection job.
const FaceDetectKind = "face_detect"

// faceDetectTimeout is the job's execution envelope. The CPU
// inference over the bounded decode is hundreds of milliseconds; the
// bound absorbs a pathological original (a huge decode) and keeps a
// wedged job retryable instead of pinning a worker slot forever.
const faceDetectTimeout = 60 * time.Second

// detectWorkingLongEdge bounds the decoded raster the job hands the
// detector: twice the default group's 640px detection input, so the
// wrapper's final resize works from real detail without a
// full-resolution raster ever crossing the libvips boundary.
const detectWorkingLongEdge = 1280

// FaceDetectArgs identifies the asset whose faces the job detects.
type FaceDetectArgs struct {
	AssetID uuid.UUID `json:"assetId"`
}

// Kind is River's stable job kind identifier.
func (FaceDetectArgs) Kind() string { return FaceDetectKind }

// FaceDetectWorker runs the face detection job: the active detect
// model's session over the asset's upright decode, the detections
// persisted as the model's faces rows (the per-mille box and the
// score only -- the landmarks travel in the Detection value for the
// recognition job of Task 7, never into the table).
type FaceDetectWorker struct {
	river.WorkerDefaults[FaceDetectArgs]

	pool     *pgxpool.Pool
	dataDir  string
	registry *ml.Registry
	logger   *slog.Logger
}

// Timeout is the job's execution envelope; see faceDetectTimeout.
func (w *FaceDetectWorker) Timeout(job *river.Job[FaceDetectArgs]) time.Duration {
	return faceDetectTimeout
}

// Work executes one face detection job. The decode and the inference
// run before the database transaction opens: no connection is held
// while pixels are processed.
func (w *FaceDetectWorker) Work(ctx context.Context, job *river.Job[FaceDetectArgs]) error {
	assetID := job.Args.AssetID

	row, err := dbgen.New(w.pool).GetAssetForMetadata(ctx, pgUUID(assetID))
	if errors.Is(err, pgx.ErrNoRows) {
		// The asset is gone (deleted while the job waited in the
		// queue): there is nothing to detect and retrying cannot
		// change that, so the job completes rather than burning its
		// attempts.
		w.logger.Info("face detect job skipped: asset no longer exists", "asset_id", assetID)
		return nil
	}
	if err != nil {
		return fmt.Errorf("load asset: %w", err)
	}

	// Only image assets carry faces here: the seam gates the enqueue
	// on the kind, so a non-image row is a mis-routed job, not a
	// retryable failure -- completing it as a no-op keeps the row's
	// pipeline untouched instead of burning retries.
	if row.MediaKind != dbgen.MediaKindImage {
		w.logger.Info("face detect job skipped: asset is not an image", "asset_id", assetID,
			"media_kind", string(row.MediaKind))
		return nil
	}

	// The slot's active model. An unfilled slot is the degraded mode:
	// a stale or mis-routed job completes silently (the readiness
	// seam dispatched nothing in the first place).
	model, err := w.registry.Active(ml.SlotFaceDetect)
	if errors.Is(err, ml.ErrNoActiveModel) {
		w.logger.Info("face detect job skipped: no active detect model", "asset_id", assetID)
		return nil
	}
	if err != nil {
		return fmt.Errorf("resolve the active face-detect model: %w", err)
	}

	// The session load's digest gate (the load-time half of the dual
	// checksum verification) runs before the decode: a drifted model
	// fails fast without burning the decode of a large original.
	sess, err := ml.NewSession(model.Path, model.SHA256)
	if err != nil {
		return fmt.Errorf("load the face detection model: %w", err)
	}
	defer func() { _ = sess.Close() }()

	path, err := storage.ResolveStoredPath(w.dataDir, row.FilePath)
	if err != nil {
		return fmt.Errorf("resolve file of asset %s: %w", assetID, err)
	}
	img, err := media.DecodeInferenceImage(path, detectWorkingLongEdge)
	if err != nil {
		return fmt.Errorf("decode asset %s for detection: %w", assetID, err)
	}

	detections, err := mlface.DetectWithParams(sess, img, model.Params)
	if err != nil {
		return fmt.Errorf("detect faces of asset %s: %w", assetID, err)
	}

	// The write: this model's faces of the asset replace themselves --
	// delete then insert, in one transaction, so a re-run converges to
	// the same rows and every other model's rows survive untouched.
	tx, err := w.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin face detect transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := dbgen.New(tx)

	if err := queries.DeleteFacesByAsset(ctx, dbgen.DeleteFacesByAssetParams{
		AssetID: pgUUID(assetID),
		ModelID: model.ID,
	}); err != nil {
		return fmt.Errorf("replace the model's faces of asset %s: %w", assetID, err)
	}
	for _, face := range detections {
		id, err := uuid.NewV7()
		if err != nil {
			return fmt.Errorf("generate face id: %w", err)
		}
		if err := queries.InsertFace(ctx, dbgen.InsertFaceParams{
			ID:      pgUUID(id),
			AssetID: pgUUID(assetID),
			BboxX:   int32(face.BBox[0]),
			BboxY:   int32(face.BBox[1]),
			BboxW:   int32(face.BBox[2]),
			BboxH:   int32(face.BBox[3]),
			Score:   face.Score,
			ModelID: model.ID,
		}); err != nil {
			return fmt.Errorf("store a face of asset %s: %w", assetID, err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit face detect transaction: %w", err)
	}

	// No continuation, by the pinned decision in the package comment:
	// recognition's trigger surface is the rerun plus Task 7's own
	// incremental decision.
	w.logger.Info("face detection complete", "asset_id", assetID,
		"model_id", model.ID, "faces", len(detections))
	return nil
}

// mlDispatchFunc dispatches one asset's ML work at the readiness
// seam. The queue wires its EnqueueMlJobs method value; workers
// constructed directly (the unit tests) carry none, and the announce
// step then skips the seam. The slot-activation matrix lives in
// EnqueueMlJobs.
type mlDispatchFunc func(ctx context.Context, assetID uuid.UUID, kind dbgen.MediaKind) error

// announceAssetReady is the shared completion step of the three
// pipeline jobs -- the one site where pending_jobs empties: after the
// worker's transaction committed and the guarded decrement made the
// asset ready, the completion announces asset.ready and hands the
// asset to the ML readiness seam. It runs after the commit so a
// subscriber that turns around and reads the database sees the final
// state.
//
// The seam's failure discipline mirrors the ML jobs' own: the asset's
// readiness is already committed fact, so a dispatch failure logs and
// moves on -- the model rerun surface recovers the missed work, and
// the pipeline job that completed stays completed (a retry would not
// re-fire this step anyway: the guarded decrement is a no-op by
// then).
func announceAssetReady(ctx context.Context, bus *events.Bus, logger *slog.Logger, dispatch mlDispatchFunc, assetID uuid.UUID, kind dbgen.MediaKind, isComponent bool) {
	if isComponent {
		// A Live Photo component row never announces (M2d) and never
		// enters the seam: the pair's photo row owns the visible
		// asset's announcement and ML surface.
		return
	}
	// The payload discipline mirrors asset.uploaded and the metadata
	// job's completion: the resource id and its media kind, nothing
	// more.
	bus.Publish("asset.ready", map[string]string{
		"assetId": assetID.String(),
		"kind":    string(kind),
	})
	if dispatch == nil {
		return
	}
	if err := dispatch(ctx, assetID, kind); err != nil {
		logger.Warn("the ML readiness dispatch failed; the model rerun surface recovers the work",
			"asset_id", assetID, "kind", string(kind), "error", err)
	}
}

// EnqueueMlJobs dispatches one asset's ML work at the readiness seam:
// the shared completion path of the pipeline jobs calls it when an
// asset's pending-jobs counter empties. Only image assets enter --
// video ML belongs to a later milestone. Within the image family the
// slot-activation matrix decides: a slot with an active model
// dispatches its per-asset job, an unfilled slot is a silent no-op --
// the degraded mode that keeps uploads flowing and the admin surface
// honest about the empty state.
//
// THE HONEST BOUNDARY of this task: only the face-detect arm is
// wired. The clip-embed and ocr slots are gated (their active models
// are resolved here) but dispatch NOTHING yet -- Task 8 wires their
// workers, and until then an active clip or OCR model changes nothing
// at readiness. The seam tests pin this as the deliberate tripwire:
// Task 8's wiring flips their expected counts.
func (q *Queue) EnqueueMlJobs(ctx context.Context, assetID uuid.UUID, kind dbgen.MediaKind) error {
	if kind != dbgen.MediaKindImage {
		return nil
	}

	if _, err := q.registry.Active(ml.SlotFaceDetect); err == nil {
		if err := q.EnqueueFaceDetect(ctx, assetID); err != nil {
			return fmt.Errorf("enqueue the face detection job of asset %s: %w", assetID, err)
		}
	} else if !errors.Is(err, ml.ErrNoActiveModel) {
		return fmt.Errorf("resolve the active face-detect model: %w", err)
	}

	// clip-embed and ocr: Task 8's wiring -- see the boundary note
	// above.
	return nil
}

// EnqueueFaceDetect inserts the face detection job for the asset. It
// must be called after the pipeline completed (the job reads the
// asset's stored original); re-enqueueing runs the job again
// idempotently -- the model's faces of the asset replace themselves.
func (q *Queue) EnqueueFaceDetect(ctx context.Context, assetID uuid.UUID) error {
	if _, err := q.client.Insert(ctx, FaceDetectArgs{AssetID: assetID}, nil); err != nil {
		return fmt.Errorf("insert face detect job: %w", err)
	}
	return nil
}
