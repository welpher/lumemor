// The face recognition job (M3a Task 7): the second arm of the
// asset-ready ML chain, dispatched by the detection job's chaining
// restore (face_detect.go's completion site -- the one incremental
// detection-to-recognition link of the milestone) and by the model
// rerun surface when it wires this slot.
//
// THE LANDMARK-REDETECTION ARCHITECTURE: the faces table stores only
// the per-mille box (the schema carries no landmark column), so the
// worker re-runs the ACTIVE detect model over the stored original
// (the same decode bound and the same params the detection job used,
// so the redetected landmarks and the stored boxes share one pixel
// space), matches each redetected face against the asset's stored
// face rows of that model by box overlap -- a match needs at least
// half the union -- and embeds every match: the landmark warp onto
// the canonical 112x112 template, then the recognize graph's feature
// with L2 normalization. An unmatched stored row keeps its NULL
// embedding: its geometry no longer redetects, and re-stamping it
// would launder a stale box into a fresh embedding.
//
// THE CLUSTERING STEP runs in the same worker, after the embedding
// updates, over the WHOLE library: every face the active recognize
// model embedded (rows of other models are invisible -- the
// coexistence policy on the embedding side), clustered by the
// brute-force DBSCAN whose eps and minPts come from the model's
// params. One transaction holds the embedding updates and the
// cluster diff-apply together: new clusters are born with their
// highest-score member as the representative; CLUSTER IDENTITY is
// preserved -- a new cluster that contains an existing cluster's
// representative, or the strict majority of its members, reuses that
// cluster's row, so a converging re-cluster never moves a user's
// labels; a cluster the new clustering no longer populates is
// deleted with its user_face_labels cascading (a person without
// faces is a person without a name); and every embedded face's
// cluster_id is rewritten, noise releasing its face to NULL. The
// face.clustered event announces the library's cluster totals after
// the commit.
//
// Concurrency: the transaction takes the face-clustering advisory
// lock, so two recognition jobs (two assets uploading at once)
// serialize their diff-applies instead of racing reads and writes on
// the same cluster rows; the eventual-convergence property would
// heal a race on the next run anyway, but the lock keeps every
// intermediate state a state a people surface could render.
//
// Failure discipline mirrors the detection job's: the inference runs
// before the transaction opens (no connection held while pixels
// move), a failure anywhere leaves the table exactly as it was, and
// River's retry owns the recovery. The job never touches pending_jobs
// -- the asset was ready before the ML chain began.
//
// THE SESSION POLICY: every job loads and digest-verifies both model
// files (the recognize graph is 38 MB -- the load is a fraction of
// the job's own inference cost at family scale, and a cached session
// would keep serving a model after a switch flipped the active flag;
// fresh loads follow the registry's truth). The clustering itself is
// O(n^2) over the library in application memory -- the cost the
// milestone accepted for a family-scale library, with incremental
// clustering a later optimization recorded in the task's report.
package jobs

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pgvector/pgvector-go"
	"github.com/riverqueue/river"

	dbgen "github.com/welpher/lumemor/server/internal/db/gen"
	"github.com/welpher/lumemor/server/internal/events"
	"github.com/welpher/lumemor/server/internal/media"
	"github.com/welpher/lumemor/server/internal/ml"
	mlface "github.com/welpher/lumemor/server/internal/ml/face"
	"github.com/welpher/lumemor/server/internal/storage"
)

// FaceRecognizeKind pins the River job kind of the face recognition
// job.
const FaceRecognizeKind = "face_recognize"

// faceRecognizeTimeout is the job's execution envelope: the decode
// and the per-face inference of the detection job's scale, plus the
// library-wide clustering pass -- seconds at family scale, bounded
// so a wedged job stays retryable instead of pinning a worker slot
// forever.
const faceRecognizeTimeout = 120 * time.Second

// faceClusteringLockKey is the advisory lock key serializing the
// cluster maintenance transactions (the embedding updates and the
// diff-apply below). The value spells "face" in hex -- a constant
// arbitrary to every other lock in the database.
const faceClusteringLockKey int64 = 0x66616365

// FaceRecognizeArgs identifies the asset whose faces the job
// recognizes.
type FaceRecognizeArgs struct {
	AssetID uuid.UUID `json:"assetId"`
}

// Kind is River's stable job kind identifier.
func (FaceRecognizeArgs) Kind() string { return FaceRecognizeKind }

// FaceRecognizeWorker runs the face recognition job: the landmark
// redetection and overlap matching over the asset's stored faces of
// the active detect model, the embedding writes stamped with the
// active recognize model, and the library-wide cluster diff-apply.
type FaceRecognizeWorker struct {
	river.WorkerDefaults[FaceRecognizeArgs]

	pool     *pgxpool.Pool
	dataDir  string
	registry *ml.Registry
	bus      *events.Bus
	logger   *slog.Logger
}

// Timeout is the job's execution envelope; see faceRecognizeTimeout.
func (w *FaceRecognizeWorker) Timeout(job *river.Job[FaceRecognizeArgs]) time.Duration {
	return faceRecognizeTimeout
}

// Work executes one face recognition job. All the inference runs
// before the transaction opens; the embedding updates and the
// cluster diff-apply land together inside it.
func (w *FaceRecognizeWorker) Work(ctx context.Context, job *river.Job[FaceRecognizeArgs]) error {
	assetID := job.Args.AssetID

	row, err := dbgen.New(w.pool).GetAssetForMetadata(ctx, pgUUID(assetID))
	if errors.Is(err, pgx.ErrNoRows) {
		w.logger.Info("face recognize job skipped: asset no longer exists", "asset_id", assetID)
		return nil
	}
	if err != nil {
		return fmt.Errorf("load asset: %w", err)
	}
	if row.MediaKind != dbgen.MediaKindImage {
		w.logger.Info("face recognize job skipped: asset is not an image", "asset_id", assetID,
			"media_kind", string(row.MediaKind))
		return nil
	}

	// Both slots must be active: the recognition embeds through the
	// recognize model and redetects landmarks through the detect one.
	// An unfilled slot is the degraded mode -- a stale or mis-routed
	// job completes silently rather than burning retries.
	recognizeModel, err := w.registry.Active(ml.SlotFaceRecognize)
	if errors.Is(err, ml.ErrNoActiveModel) {
		w.logger.Info("face recognize job skipped: no active recognize model", "asset_id", assetID)
		return nil
	}
	if err != nil {
		return fmt.Errorf("resolve the active face recognize model: %w", err)
	}
	detectModel, err := w.registry.Active(ml.SlotFaceDetect)
	if errors.Is(err, ml.ErrNoActiveModel) {
		w.logger.Info("face recognize job skipped: no active detect model (landmark redetection needs it)", "asset_id", assetID)
		return nil
	}
	if err != nil {
		return fmt.Errorf("resolve the active face detect model: %w", err)
	}

	// The digest gates run before the decode: drifted weights fail
	// fast without burning the decode of a large original.
	recognizeSess, err := ml.NewSession(recognizeModel.Path, recognizeModel.SHA256)
	if err != nil {
		return fmt.Errorf("load the face recognize model: %w", err)
	}
	defer func() { _ = recognizeSess.Close() }()
	detectSess, err := ml.NewSession(detectModel.Path, detectModel.SHA256)
	if err != nil {
		return fmt.Errorf("load the face detect model for the landmark redetection: %w", err)
	}
	defer func() { _ = detectSess.Close() }()

	path, err := storage.ResolveStoredPath(w.dataDir, row.FilePath)
	if err != nil {
		return fmt.Errorf("resolve file of asset %s: %w", assetID, err)
	}
	// The SAME decode bound the detection job uses: the redetected
	// landmarks and this raster share one pixel space, which is what
	// the overlap matching and the alignment warp both consume.
	img, err := media.DecodeInferenceImage(path, detectWorkingLongEdge)
	if err != nil {
		return fmt.Errorf("decode asset %s for recognition: %w", assetID, err)
	}

	// The landmark redetection: the active detect model with its own
	// params, exactly as the detection job ran it.
	detections, err := mlface.DetectWithParams(detectSess, img, detectModel.Params)
	if err != nil {
		return fmt.Errorf("redetect the faces of asset %s: %w", assetID, err)
	}

	// The stored faces of this asset and this detect model -- the
	// coexistence-visible set the matching walks.
	stored, err := dbgen.New(w.pool).ListFacesByAsset(ctx, dbgen.ListFacesByAssetParams{
		AssetID: pgUUID(assetID),
		ModelID: detectModel.ID,
	})
	if err != nil {
		return fmt.Errorf("list the stored faces of asset %s: %w", assetID, err)
	}

	updates, err := w.embedMatches(assetID, recognizeSess, img, detections, stored)
	if err != nil {
		return err
	}

	// One transaction: the advisory lock, the embedding writes, the
	// cluster diff-apply, and the counts the event announces.
	tx, err := w.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin face recognize transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, "SELECT pg_advisory_xact_lock($1)", faceClusteringLockKey); err != nil {
		return fmt.Errorf("take the face clustering lock: %w", err)
	}
	queries := dbgen.New(tx)

	for _, update := range updates {
		if err := queries.UpdateFaceEmbedding(ctx, dbgen.UpdateFaceEmbeddingParams{
			ID:           pgUUID(update.faceID),
			Embedding:    pgvector.NewVector(update.embedding),
			EmbedModelID: pgtype.Text{String: recognizeModel.ID, Valid: true},
		}); err != nil {
			return fmt.Errorf("store the embedding of a face of asset %s: %w", assetID, err)
		}
	}

	totalClusters, labeledClusters, err := applyFaceClustering(ctx, queries, recognizeModel)
	if err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit face recognize transaction: %w", err)
	}

	// The event's payload follows the asset domain's discipline:
	// camelCase keys, string values, the resource's honest counts.
	w.bus.Publish("face.clustered", map[string]string{
		"totalClusters":   strconv.Itoa(totalClusters),
		"labeledClusters": strconv.Itoa(labeledClusters),
	})

	w.logger.Info("face recognition complete", "asset_id", assetID,
		"recognize_model_id", recognizeModel.ID, "detect_model_id", detectModel.ID,
		"faces_embedded", len(updates), "total_clusters", totalClusters, "labeled_clusters", labeledClusters)
	return nil
}

// faceUpdate is one matched face's new embedding: the row to stamp
// and the vector to stamp it with.
type faceUpdate struct {
	faceID    uuid.UUID
	embedding []float32
}

// embedMatches pairs the redetected faces with the stored rows by
// box overlap and embeds every pair: each redetection claims the
// unclaimed stored row it overlaps most (at least half the union --
// the brief's matching rule; a smaller overlap never matches), the
// redetections walking in their score order and the ties falling to
// the stored listing's deterministic order. A stored row without a
// match is left exactly as it was.
func (w *FaceRecognizeWorker) embedMatches(assetID uuid.UUID, sess *ml.Session, img ml.Image, detections []mlface.Detection, stored []dbgen.ListFacesByAssetRow) ([]faceUpdate, error) {
	claimed := make([]bool, len(stored))
	var updates []faceUpdate
	for _, detection := range detections {
		best, bestOverlap := -1, 0.0
		for i, face := range stored {
			if claimed[i] {
				continue
			}
			overlap := boxOverlap(
				pixelBox(detection.BBox, img.Width, img.Height),
				pixelBox([4]int{int(face.BboxX), int(face.BboxY), int(face.BboxW), int(face.BboxH)}, img.Width, img.Height))
			if overlap > bestOverlap {
				best, bestOverlap = i, overlap
			}
		}
		if best < 0 || bestOverlap < 0.5 {
			continue
		}
		claimed[best] = true

		crop, err := mlface.AlignCrop(img, detection.Landmarks)
		if err != nil {
			return nil, fmt.Errorf("align a face of asset %s: %w", assetID, err)
		}
		embedding, err := mlface.Embed(sess, crop)
		if err != nil {
			return nil, fmt.Errorf("embed a face of asset %s: %w", assetID, err)
		}
		updates = append(updates, faceUpdate{
			faceID:    uuid.UUID(stored[best].ID.Bytes),
			embedding: embedding,
		})
	}
	return updates, nil
}

// pixelBox maps one per-mille box into the raster's pixel space.
func pixelBox(bbox [4]int, width, height int) [4]float64 {
	return [4]float64{
		float64(bbox[0]) / 1000 * float64(width),
		float64(bbox[1]) / 1000 * float64(height),
		float64(bbox[2]) / 1000 * float64(width),
		float64(bbox[3]) / 1000 * float64(height),
	}
}

// boxOverlap is the intersection-over-union of two x, y, w, h pixel
// boxes: the matching rule's measure, scale-free within one raster.
func boxOverlap(a, b [4]float64) float64 {
	left := max(a[0], b[0])
	top := max(a[1], b[1])
	right := min(a[0]+a[2], b[0]+b[2])
	bottom := min(a[1]+a[3], b[1]+b[3])
	overlap := max(0, right-left) * max(0, bottom-top)
	union := a[2]*a[3] + b[2]*b[3] - overlap
	if union <= 0 {
		return 0
	}
	return overlap / union
}

// embeddedFace is one clustering input row: the face's identity, its
// detection score (the representative election's currency) and the
// cluster it currently belongs to.
type embeddedFace struct {
	id      uuid.UUID
	score   float32
	cluster uuid.UUID // uuid.Nil when none
}

// existingCluster is one cluster row as the diff-apply reads it: its
// identity, its representative, and its members before this pass's
// assignments.
type existingCluster struct {
	id             uuid.UUID
	representative uuid.UUID
	members        map[uuid.UUID]bool
}

// applyFaceClustering runs the clustering step's diff-apply over the
// active recognize model's embedded faces, inside the caller's
// transaction (after the embedding writes and under the advisory
// lock), and answers the library's cluster totals for the event.
//
// The identity rule, the load-bearing half of Review Focus #1's
// cluster side: a new cluster reuses an existing cluster's row when
// it contains that cluster's representative face, or the strict
// majority of that cluster's members -- so a converging re-cluster
// keeps the row (and with it every user label), a split keeps the
// representative's branch, and a merge picks the branch with the
// larger overlap. Candidate clusters with no overlap are invisible;
// the best-overlapping candidate wins, ties resolved by the listing
// order (the smaller id).
func applyFaceClustering(ctx context.Context, queries *dbgen.Queries, recognizeModel *ml.Model) (totalClusters, labeledClusters int, err error) {
	eps, minPts := mlface.ClusterOptionsOf(recognizeModel.Params)

	rows, err := queries.ListEmbeddedFacesByEmbedModel(ctx, pgtype.Text{String: recognizeModel.ID, Valid: true})
	if err != nil {
		return 0, 0, fmt.Errorf("list the embedded faces: %w", err)
	}
	faces := make([]embeddedFace, 0, len(rows))
	embeddings := make([][]float32, 0, len(rows))
	for _, row := range rows {
		face := embeddedFace{id: uuid.UUID(row.ID.Bytes), score: row.Score}
		if row.ClusterID.Valid {
			face.cluster = uuid.UUID(row.ClusterID.Bytes)
		}
		faces = append(faces, face)
		embeddings = append(embeddings, row.Embedding.Slice())
	}

	labels := mlface.Cluster(embeddings, eps, minPts)

	// The groups, in the listing's order (score descending, then id):
	// each group's first face is its highest-score member -- the
	// representative a birth or a reuse points at.
	groups := map[int][]int{}
	maxLabel := -1
	for i, label := range labels {
		if label < 0 {
			continue
		}
		groups[label] = append(groups[label], i)
		if label > maxLabel {
			maxLabel = label
		}
	}

	// The existing state, read before any assignment: every cluster
	// row with its representative and its members.
	existing, err := listExistingClusters(ctx, queries)
	if err != nil {
		return 0, 0, err
	}

	// The identity resolution: one existing cluster reused by at most
	// one new cluster.
	used := map[uuid.UUID]bool{}
	clusterOfLabel := make([]uuid.UUID, maxLabel+1)
	for label := 0; label <= maxLabel; label++ {
		group := groups[label]
		if len(group) == 0 {
			continue
		}
		best, bestOverlap := -1, 0
		for ci, cluster := range existing {
			if used[cluster.id] {
				continue
			}
			overlap := 0
			representativeInside := false
			for _, fi := range group {
				if cluster.members[faces[fi].id] {
					overlap++
				}
				if faces[fi].id == cluster.representative {
					representativeInside = true
				}
			}
			if overlap == 0 {
				continue
			}
			if !representativeInside && overlap*2 <= len(cluster.members) {
				// Neither the representative nor the strict majority
				// of the members: not this cluster's identity.
				continue
			}
			if overlap > bestOverlap {
				best, bestOverlap = ci, overlap
			}
		}
		if best >= 0 {
			clusterOfLabel[label] = existing[best].id
			used[existing[best].id] = true
		}
	}

	// The writes: births, representative refreshes for the reused
	// rows, and every face's assignment (a NULL release for noise).
	for label := 0; label <= maxLabel; label++ {
		group := groups[label]
		if len(group) == 0 {
			continue
		}
		representative := faces[group[0]].id
		clusterID := clusterOfLabel[label]
		if clusterID == uuid.Nil {
			id, err := uuid.NewV7()
			if err != nil {
				return 0, 0, fmt.Errorf("generate a cluster id: %w", err)
			}
			if err := queries.InsertCluster(ctx, dbgen.InsertClusterParams{
				ID:                   pgUUID(id),
				RepresentativeFaceID: pgUUID(representative),
			}); err != nil {
				return 0, 0, fmt.Errorf("create a face cluster: %w", err)
			}
			clusterOfLabel[label] = id
		} else {
			if err := queries.UpdateClusterRepresentative(ctx, dbgen.UpdateClusterRepresentativeParams{
				ID:                   pgUUID(clusterID),
				RepresentativeFaceID: pgUUID(representative),
			}); err != nil {
				return 0, 0, fmt.Errorf("refresh a face cluster's representative: %w", err)
			}
		}
	}
	for i, label := range labels {
		var assignment pgtype.UUID
		if label >= 0 && clusterOfLabel[label] != uuid.Nil {
			assignment = pgUUID(clusterOfLabel[label])
		}
		if err := queries.UpdateFaceCluster(ctx, dbgen.UpdateFaceClusterParams{
			ID:        pgUUID(faces[i].id),
			ClusterID: assignment,
		}); err != nil {
			return 0, 0, fmt.Errorf("assign a face's cluster: %w", err)
		}
	}

	// The dissolutions: a cluster no new cluster claimed keeps its
	// row only while it still has members (faces of other recognize
	// models keep theirs -- the coexistence policy). A memberless row
	// goes, its user_face_labels cascading with it.
	for _, cluster := range existing {
		if used[cluster.id] {
			continue
		}
		remaining, err := queries.ListClusterFaces(ctx, pgUUID(cluster.id))
		if err != nil {
			return 0, 0, fmt.Errorf("list the members of a face cluster: %w", err)
		}
		if len(remaining) == 0 {
			if err := queries.DeleteCluster(ctx, pgUUID(cluster.id)); err != nil {
				return 0, 0, fmt.Errorf("dissolve a vacated face cluster: %w", err)
			}
		}
	}

	total, err := queries.CountFaceClusters(ctx)
	if err != nil {
		return 0, 0, fmt.Errorf("count the face clusters: %w", err)
	}
	labeled, err := queries.CountLabeledFaceClusters(ctx)
	if err != nil {
		return 0, 0, fmt.Errorf("count the labeled face clusters: %w", err)
	}
	return int(total), int(labeled), nil
}

// listExistingClusters reads every cluster row with its members, in
// id order -- the deterministic order the identity ties resolve
// against.
func listExistingClusters(ctx context.Context, queries *dbgen.Queries) ([]existingCluster, error) {
	rows, err := queries.ListClusters(ctx)
	if err != nil {
		return nil, fmt.Errorf("list the face clusters: %w", err)
	}
	existing := make([]existingCluster, 0, len(rows))
	for _, row := range rows {
		cluster := existingCluster{
			id:             uuid.UUID(row.ID.Bytes),
			representative: uuid.UUID(row.RepresentativeFaceID.Bytes),
			members:        map[uuid.UUID]bool{},
		}
		members, err := queries.ListClusterFaces(ctx, row.ID)
		if err != nil {
			return nil, fmt.Errorf("list the members of cluster %s: %w", cluster.id, err)
		}
		for _, member := range members {
			cluster.members[uuid.UUID(member.ID.Bytes)] = true
		}
		existing = append(existing, cluster)
	}
	return existing, nil
}

// EnqueueFaceRecognize inserts the face recognition job for the
// asset. The detection job's chaining site calls it after its own
// commit (the job reads the asset's stored faces); re-enqueueing
// runs the job again idempotently -- the matching re-embeds the same
// rows and the clustering converges onto the same clusters.
func (q *Queue) EnqueueFaceRecognize(ctx context.Context, assetID uuid.UUID) error {
	if _, err := q.client.Insert(ctx, FaceRecognizeArgs{AssetID: assetID}, nil); err != nil {
		return fmt.Errorf("insert face recognize job: %w", err)
	}
	return nil
}
