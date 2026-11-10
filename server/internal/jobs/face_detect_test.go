// The face detection job tests: the worker wiring (the
// committed YuNet weights as the registry's active detect model, the
// real decode, the faces rows landing with every field), the failure
// discipline (a broken model never touches the asset's
// readiness), the readiness seam's slot-activation matrix
// (the degraded mode: no active model, zero dispatch,
// zero errors), and the NO-AUTO-CHAIN decision (the detection
// completion enqueues nothing: recognition's trigger surface is the
// rerun plus the chaining restore's incremental decision).
//
// The inference-bearing tests skip on hosts without the onnxruntime
// shared library (and the decode-bearing ones without libvips): the
// containerized `make docker-test` gate runs everything for real. The
// matrix and failure tests need neither -- the seam consults the
// registry's rows only, and a tampered weight file is refused at the
// digest gate that precedes any library call.
package jobs

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"image"
	"image/color"
	"image/jpeg"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/riverqueue/river"

	dbgen "github.com/welpher/lumemor/server/internal/db/gen"
	"github.com/welpher/lumemor/server/internal/ml"
)

// requireMlRuntime skips the real-inference tests on hosts without
// the onnxruntime shared library.
func requireMlRuntime(t *testing.T) {
	t.Helper()
	if !ml.RuntimeAvailable() {
		t.Skip("onnxruntime shared library not present; run the suite inside the dev container (make docker-test)")
	}
}

// fixtureModelSpec describes one model directory the tests lay out
// under the environment's models dir: the registered id, the slot it
// fills, the weights file (one of the committed ml testdata fixtures)
// and its pinned digest.
type fixtureModelSpec struct {
	id        string
	slot      ml.Slot
	file      string
	sha256    string
	inputSize ml.ManifestSize
	dim       int
	params    map[string]any
}

// The real YuNet weights as a registered model: the same bytes the
// default manifest pins, laid out the way a downloaded model lands.
var yunetFixtureSpec = fixtureModelSpec{
	id:        "yunet-fixture",
	slot:      ml.SlotFaceDetect,
	file:      "face_detection_yunet_2023mar.onnx",
	sha256:    "8f2383e4dd3cfbb4553ea8718107fc0423210dc964f9f4280604804ed2552fa4",
	inputSize: ml.ManifestSize{Width: 640, Height: 640},
	params:    map[string]any{"scoreThreshold": 0.6, "nmsThreshold": 0.3, "topK": 5000},
}

// The identity graph as a registered model for the slots that need
// no real inference (the seam matrix's unfilled-by-real-models
// slots): a valid, verifiable weight file that registers and
// activates like any other.
var identityFixtureSpec = fixtureModelSpec{
	id:        "identity-fixture",
	file:      "identity.onnx",
	sha256:    "7f6a5659d2eb72e8fa03a0a7eabad28c8e529bfb7f26106911e8617b4455be3b",
	inputSize: ml.ManifestSize{Width: 8, Height: 8},
	params:    map[string]any{"scoreThreshold": 0.6},
}

// manifestSlotNames maps the registry's Slot values onto the
// hyphenated slot names the per-model manifests carry (the same
// mapping the registry's scan applies in reverse).
var manifestSlotNames = map[ml.Slot]string{
	ml.SlotFaceDetect:    "face-detect",
	ml.SlotFaceRecognize: "face-recognize",
	ml.SlotClipEmbed:     "clip-embed",
	ml.SlotClipText:      "clip-text",
	ml.SlotOcr:           "ocr",
}

// registerFixtureModel lays the fixture's model directory out under the
// environment's models dir (manifest.json plus the weights, pinned by
// digest -- the layout of a landed download), scans it into the
// registry and returns the model id. It does NOT activate.
func (e *env) registerFixtureModel(t *testing.T, spec fixtureModelSpec) string {
	t.Helper()

	dir := filepath.Join(e.modelsDir, spec.id)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("create model dir: %v", err)
	}
	weights, err := os.ReadFile(filepath.Join("..", "ml", "testdata", spec.file))
	if err != nil {
		t.Fatalf("read fixture weights %s: %v", spec.file, err)
	}
	if err := os.WriteFile(filepath.Join(dir, spec.file), weights, 0o644); err != nil {
		t.Fatalf("write fixture weights: %v", err)
	}
	manifest := ml.ManifestModel{
		ID:        spec.id,
		Slot:      manifestSlotNames[spec.slot],
		Version:   "fixture",
		License:   "fixture",
		InputSize: spec.inputSize,
		Dim:       spec.dim,
		Params:    spec.params,
		Files: []ml.ManifestFile{{
			Name:   spec.file,
			SHA256: spec.sha256,
			Bytes:  int64(len(weights)),
		}},
	}
	blob, err := json.Marshal(manifest)
	if err != nil {
		t.Fatalf("marshal fixture manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), blob, 0o644); err != nil {
		t.Fatalf("write fixture manifest: %v", err)
	}

	if _, err := e.registry.Scan(e.modelsDir); err != nil {
		t.Fatalf("scan the fixture model in: %v", err)
	}
	return spec.id
}

// activateFixtureModel registers and activates the fixture's model in its
// slot, failing the test unless it became the active one.
func (e *env) activateFixtureModel(t *testing.T, spec fixtureModelSpec) string {
	t.Helper()
	id := e.registerFixtureModel(t, spec)
	if _, err := e.registry.Activate(spec.slot, id); err != nil {
		t.Fatalf("activate the fixture model %s: %v", id, err)
	}
	return id
}

// faceRow is the faces row as the assertions read it. The embedding's
// presence is read as a flag: the detection job must leave it NULL
// (the recognition job owns it).
type faceRow struct {
	clusterID    pgtype.UUID
	bbox         [4]int32
	score        float32
	modelID      string
	embedModelID pgtype.Text
	embeddingSet bool
	createdAt    time.Time
}

// facesOf loads the asset's faces across every model, strongest
// first.
func (e *env) facesOf(t *testing.T, assetID uuid.UUID) []faceRow {
	t.Helper()
	rows, err := e.pool.Query(context.Background(), `
		SELECT cluster_id, bbox_x, bbox_y, bbox_w, bbox_h, score, model_id,
		       embed_model_id, embedding IS NOT NULL, created_at
		FROM faces WHERE asset_id = $1
		ORDER BY score DESC`, pgUUID(assetID))
	if err != nil {
		t.Fatalf("query faces of asset %s: %v", assetID, err)
	}
	defer rows.Close()
	var out []faceRow
	for rows.Next() {
		var row faceRow
		if err := rows.Scan(&row.clusterID, &row.bbox[0], &row.bbox[1], &row.bbox[2], &row.bbox[3],
			&row.score, &row.modelID, &row.embedModelID, &row.embeddingSet, &row.createdAt); err != nil {
			t.Fatalf("scan face row: %v", err)
		}
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate faces: %v", err)
	}
	return out
}

// faceDetectWorker builds the worker over the environment for a
// direct Work invocation (the River machinery has its own tests).
func (e *env) faceDetectWorker() *FaceDetectWorker {
	return &FaceDetectWorker{
		pool:     e.pool,
		dataDir:  e.dataDir,
		registry: e.registry,
		logger:   slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
}

// runFaceDetect drives one direct Work invocation for the asset.
func runFaceDetect(t *testing.T, e *env, assetID uuid.UUID) error {
	t.Helper()
	return e.faceDetectWorker().Work(context.Background(), &river.Job[FaceDetectArgs]{
		Args: FaceDetectArgs{AssetID: assetID},
	})
}

// portraitFixture reads the committed face portrait (or its
// orientation-6 twin).
func portraitFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "ml", "testdata", name))
	if err != nil {
		t.Fatalf("read portrait fixture %s: %v", name, err)
	}
	return data
}

// insertFaceAsset stores one image asset carrying the given portrait
// fixture, with its pipeline already counted and drained (the
// detection job runs after readiness; the direct tests bypass the
// pipeline) and readiness stamped.
func (e *env) insertFaceAsset(t *testing.T, rel string, portrait []byte) uuid.UUID {
	t.Helper()
	takenAt := time.Date(2020, 3, 1, 12, 0, 0, 0, time.UTC).Truncate(time.Microsecond)
	id := e.insertAssetKind(t, rel, portrait, takenAt,
		`{"deviceCreatedAt":"2020-03-01T12:00:00Z","deviceTimeZone":"Asia/Shanghai"}`,
		dbgen.MediaKindImage, "image/jpeg", 0)
	if _, err := e.pool.Exec(context.Background(),
		`UPDATE assets SET ready_at = now() WHERE id = $1`, pgUUID(id)); err != nil {
		t.Fatalf("stamp the asset ready: %v", err)
	}
	return id
}

// requireFaceBounds asserts the per-mille geometry contract of one
// face row.
func requireFaceBounds(t *testing.T, row faceRow, what string) {
	t.Helper()
	for i, field := range row.bbox {
		if field < 0 || field > 1000 {
			t.Errorf("%s bbox[%d] = %d, want it inside 0-1000 (per-mille)", what, i, field)
		}
	}
	if row.bbox[0]+row.bbox[2] > 1000 || row.bbox[1]+row.bbox[3] > 1000 {
		t.Errorf("%s bbox = %v, want the box inside the image", what, row.bbox)
	}
	if row.score <= 0.6 {
		t.Errorf("%s score = %v, want it above the 0.6 threshold", what, row.score)
	}
}

// TestFaceDetectJobPersistsFaces pins the worker wiring end to end on
// the real weights: the portrait asset's detection lands as one faces
// row with every field asserted -- the per-mille box inside the image,
// the score above the threshold, the active model's id, the embedding
// and cluster left NULL (the recognition job owns them), the creation timestamp in
// the run's window.
func TestFaceDetectJobPersistsFaces(t *testing.T) {
	requireThumbnails(t)
	requireMlRuntime(t)
	e := newEnv(t)
	modelID := e.activateFixtureModel(t, yunetFixtureSpec)

	before := time.Now().Truncate(time.Microsecond)
	id := e.insertFaceAsset(t, "originals/face/portrait.jpg", portraitFixture(t, "face-portrait.jpg"))
	if err := runFaceDetect(t, e, id); err != nil {
		t.Fatalf("Work error = %v, want nil", err)
	}
	after := time.Now().Truncate(time.Microsecond)

	faces := e.facesOf(t, id)
	if len(faces) != 1 {
		t.Fatalf("faces rows = %d, want exactly 1 (the portrait's one face)", len(faces))
	}
	face := faces[0]
	requireFaceBounds(t, face, "the portrait's face")
	if face.modelID != modelID {
		t.Errorf("face model_id = %q, want the active detect model %q", face.modelID, modelID)
	}
	if face.clusterID.Valid {
		t.Errorf("face cluster_id = %v, want NULL (clustering is the recognition job's)", face.clusterID)
	}
	if face.embeddingSet {
		t.Error("face embedding is set, want NULL (the recognition job owns it)")
	}
	if face.embedModelID.Valid {
		t.Errorf("face embed_model_id = %v, want NULL", face.embedModelID)
	}
	if face.createdAt.Before(before) || face.createdAt.After(after) {
		t.Errorf("face created_at = %s, want it inside the run's window [%s, %s]",
			face.createdAt, before, after)
	}

	// The face sits where the portrait's face is (the same coarse
	// bands the ml-level test pins): the worker's govips decode and
	// downscale preserved the per-mille geometry.
	if face.bbox[0] < 320 || face.bbox[0] > 440 {
		t.Errorf("face bbox x = %d‰, want the portrait's face position", face.bbox[0])
	}
	if face.bbox[1] < 170 || face.bbox[1] > 250 {
		t.Errorf("face bbox y = %d‰, want the portrait's face position", face.bbox[1])
	}
}

// TestFaceDetectJobOrientation pins the EXIF orientation correction
// of the worker's decode: the orientation-6 twin of the portrait (a
// 90-degree sensor frame whose tag says so) detects the face at the
// SAME per-mille position as the upright portrait -- the decode
// auto-rotated before the raster reached the detector, so the
// geometry survives the round trip.
func TestFaceDetectJobOrientation(t *testing.T) {
	requireThumbnails(t)
	requireMlRuntime(t)
	e := newEnv(t)
	e.activateFixtureModel(t, yunetFixtureSpec)

	upright := e.insertFaceAsset(t, "originals/face/upright.jpg", portraitFixture(t, "face-portrait.jpg"))
	if err := runFaceDetect(t, e, upright); err != nil {
		t.Fatalf("Work(upright) error = %v, want nil", err)
	}
	oriented := e.insertFaceAsset(t, "originals/face/oriented.jpg", portraitFixture(t, "face-portrait-orientation6.jpg"))
	if err := runFaceDetect(t, e, oriented); err != nil {
		t.Fatalf("Work(oriented) error = %v, want nil", err)
	}

	uprightFaces := e.facesOf(t, upright)
	orientedFaces := e.facesOf(t, oriented)
	if len(uprightFaces) != 1 || len(orientedFaces) != 1 {
		t.Fatalf("faces = upright %d, oriented %d, want 1 and 1", len(uprightFaces), len(orientedFaces))
	}
	// The JPEG re-encode of the rotated twin and the decode's rescale
	// move the box a little; the correction keeps it inside a tight
	// per-mille neighborhood.
	const tolerance = 30
	for i := 0; i < 4; i++ {
		if delta := uprightFaces[0].bbox[i] - orientedFaces[0].bbox[i]; delta < -tolerance || delta > tolerance {
			t.Errorf("bbox[%d] drifted across the orientation fix: upright %d‰, oriented %d‰",
				i, uprightFaces[0].bbox[i], orientedFaces[0].bbox[i])
		}
	}
}

// TestFaceDetectJobNoActiveModelSkipsCleanly pins the degraded-mode
// worker: a stale or mis-routed job against a slot with no active
// model completes as a no-op (no error, no rows, no retry burn).
func TestFaceDetectJobNoActiveModelSkipsCleanly(t *testing.T) {
	e := newEnv(t)

	id := e.insertFaceAsset(t, "originals/face/portrait.jpg", portraitFixture(t, "face-portrait.jpg"))
	if err := runFaceDetect(t, e, id); err != nil {
		t.Fatalf("Work error = %v, want the clean skip", err)
	}
	if faces := e.facesOf(t, id); len(faces) != 0 {
		t.Errorf("faces rows = %d, want 0", len(faces))
	}
}

// TestFaceDetectJobNonImageIsNoOp pins the kind gate: a video row
// (mis-routed job) completes as a no-op instead of burning retries
// against an asset that can never detect.
func TestFaceDetectJobNonImageIsNoOp(t *testing.T) {
	e := newEnv(t)
	e.activateFixtureModel(t, yunetFixtureSpec)

	takenAt := time.Date(2020, 3, 1, 12, 0, 0, 0, time.UTC).Truncate(time.Microsecond)
	id := e.insertAssetKind(t, "originals/face/video.mp4", portraitFixture(t, "face-portrait.jpg"), takenAt,
		`{}`, dbgen.MediaKindVideo, "video/mp4", 0)
	if err := runFaceDetect(t, e, id); err != nil {
		t.Fatalf("Work error = %v, want the no-op", err)
	}
	if faces := e.facesOf(t, id); len(faces) != 0 {
		t.Errorf("faces rows = %d, want 0", len(faces))
	}
}

// TestFaceDetectJobMissingAssetCompletes pins the gone-asset path: a
// job whose asset disappeared completes rather than retrying forever.
func TestFaceDetectJobMissingAssetCompletes(t *testing.T) {
	e := newEnv(t)

	id, err := uuid.NewV7()
	if err != nil {
		t.Fatalf("new asset id: %v", err)
	}
	if err := runFaceDetect(t, e, id); err != nil {
		t.Fatalf("Work error = %v, want the no-op", err)
	}
}

// TestFaceDetectJobNoFaceImage pins the no-face boundary at the
// worker level: a valid image with no face in it completes with zero
// rows and no error (the empty detection set is a success, not a
// failure -- the delete-then-insert converges on the model's empty
// row set).
func TestFaceDetectJobNoFaceImage(t *testing.T) {
	requireThumbnails(t)
	requireMlRuntime(t)
	e := newEnv(t)
	e.activateFixtureModel(t, yunetFixtureSpec)

	id := e.insertFaceAsset(t, "originals/face/none.jpg", facelessJPEG(t))
	if err := runFaceDetect(t, e, id); err != nil {
		t.Fatalf("Work error = %v, want nil (zero faces is a success)", err)
	}
	if faces := e.facesOf(t, id); len(faces) != 0 {
		t.Errorf("faces rows = %d, want 0 for a faceless image", len(faces))
	}
}

// facelessJPEG encodes a smooth gradient as a real JPEG: a
// deterministic file with no face in it, the no-face boundary's
// on-disk stand-in.
func facelessJPEG(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 480, 360))
	for y := 0; y < 360; y++ {
		for x := 0; x < 480; x++ {
			img.SetRGBA(x, y, color.RGBA{
				R: uint8(255 * y / 359),
				G: uint8(255 * x / 479),
				B: uint8(255 - 255*y/359),
				A: 255,
			})
		}
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, nil); err != nil {
		t.Fatalf("encode the faceless gradient JPEG: %v", err)
	}
	return buf.Bytes()
}

// TestFaceDetectJobRerunReplacesOwnModelFaces pins the replace
// semantics and the model-switch coexistence: a second run of the
// same model replaces that model's faces (no pile-up), and the faces
// of a different model survive untouched (switching
// models keeps the old products in the table, filtered
// out of the active model's queries).
func TestFaceDetectJobRerunReplacesOwnModelFaces(t *testing.T) {
	requireThumbnails(t)
	requireMlRuntime(t)
	e := newEnv(t)
	active := e.activateFixtureModel(t, yunetFixtureSpec)
	other := e.registerFixtureModel(t, fixtureModelSpec{
		id:        "yunet-other",
		slot:      ml.SlotFaceDetect,
		file:      yunetFixtureSpec.file,
		sha256:    yunetFixtureSpec.sha256,
		inputSize: yunetFixtureSpec.inputSize,
		params:    yunetFixtureSpec.params,
	})

	id := e.insertFaceAsset(t, "originals/face/portrait.jpg", portraitFixture(t, "face-portrait.jpg"))

	// The other model's face row: seeded directly, the way a previous
	// active model's products sit in the table after a switch.
	seeded, err := uuid.NewV7()
	if err != nil {
		t.Fatalf("new face id: %v", err)
	}
	if err := e.queries.InsertFace(context.Background(), dbgen.InsertFaceParams{
		ID:      pgUUID(seeded),
		AssetID: pgUUID(id),
		BboxX:   111, BboxY: 222, BboxW: 333, BboxH: 444,
		Score:   0.9,
		ModelID: other,
	}); err != nil {
		t.Fatalf("seed the other model's face: %v", err)
	}

	for run := 1; run <= 2; run++ {
		if err := runFaceDetect(t, e, id); err != nil {
			t.Fatalf("run %d Work error = %v, want nil", run, err)
		}
		faces := e.facesOf(t, id)
		if len(faces) != 2 {
			t.Fatalf("run %d faces rows = %d, want 2 (the active model's replacement plus the other model's survivor)", run, len(faces))
		}
		var activeRows, otherRows int
		for _, face := range faces {
			if face.modelID == active {
				activeRows++
				requireFaceBounds(t, face, "the active model's face")
			}
			if face.modelID == other {
				otherRows++
			}
		}
		if activeRows != 1 {
			t.Errorf("run %d active model faces = %d, want exactly 1 (replaced, not piled)", run, activeRows)
		}
		if otherRows != 1 {
			t.Errorf("run %d other model faces = %d, want the survivor 1", run, otherRows)
		}
	}
}

// TestFaceDetectJobFailureKeepsAssetReady pins the failure discipline:
// the pipeline completes and announces asset.ready,
// the seam dispatches the detection job, and the job's failure -- a
// tampered weight file refused at the session's digest gate -- leaves
// the asset's readiness untouched: pending_jobs stays zero, ready_at
// stands, exactly one asset.ready fired, no faces landed, and River
// holds the job for its retry (attempts grow, state retryable).
func TestFaceDetectJobFailureKeepsAssetReady(t *testing.T) {
	requireExiftool(t)
	e := newEnv(t)
	e.activateFixtureModel(t, yunetFixtureSpec)

	// Tamper the model's weights after activation: the registry hands
	// the model out (activation verified it earlier), and the session
	// loader's digest gate refuses the drifted file -- the refusal a
	// direct Work invocation pins before the queue's own retry
	// machinery runs below.
	weights := filepath.Join(e.modelsDir, yunetFixtureSpec.id, yunetFixtureSpec.file)
	if err := os.WriteFile(weights, []byte("tampered bytes"), 0o644); err != nil {
		t.Fatalf("tamper the model weights: %v", err)
	}
	direct := e.insertFaceAsset(t, "originals/face/direct.jpg", portraitFixture(t, "face-portrait.jpg"))
	if err := runFaceDetect(t, e, direct); !errors.Is(err, ml.ErrInvalidModel) {
		t.Fatalf("direct Work over tampered weights error = %v, want the digest refusal", err)
	}

	takenAt := time.Date(2020, 3, 1, 12, 0, 0, 0, time.UTC).Truncate(time.Microsecond)
	id := e.insertAsset(t, "originals/face/portrait.jpg", portraitFixture(t, "face-portrait.jpg"), takenAt,
		`{"deviceCreatedAt":"2020-03-01T12:00:00Z","deviceTimeZone":"Asia/Shanghai"}`)
	if err := e.queue.EnqueueMetadata(context.Background(), id); err != nil {
		t.Fatalf("enqueue metadata job: %v", err)
	}
	eventually(t, jobWaitTimeout, "the asset to become ready", func() bool {
		return e.assetOf(t, id).readyAt.Valid
	})

	// The pipeline completed; the ML job failed; nothing about the
	// asset moved.
	state := e.assetOf(t, id)
	if state.pending != 0 {
		t.Errorf("pending_jobs = %d, want 0 (the ML failure never touches the pipeline counter)", state.pending)
	}
	if !state.readyAt.Valid {
		t.Fatal("ready_at is unset, want the pipeline's stamp to stand")
	}
	if got := countEvents(e.bus, "asset.ready"); got != 1 {
		t.Errorf("asset.ready events = %d, want exactly 1 (the pipeline's own)", got)
	}
	if faces := e.facesOf(t, id); len(faces) != 0 {
		t.Errorf("faces rows = %d, want 0 (a failed detection persists nothing)", len(faces))
	}

	// River holds the failed job for its retry: the state settles to
	// retryable once the failed attempt returns (the attempt counter
	// already shows the run happened).
	eventually(t, jobWaitTimeout, "the failed detection job to reach the retryable state", func() bool {
		return e.riverJobState(t, id, FaceDetectKind) == "retryable"
	})
	if got := e.riverJobAttempts(t, id, FaceDetectKind); got < 1 {
		t.Errorf("face detect job attempt = %d, want at least 1 (the failed run)", got)
	}
}

// riverJobAttempts reads the attempt count of one River job of the
// asset. River inserts jobs at attempt 0 and increments on every run,
// so a failed first run reads 1.
func (e *env) riverJobAttempts(t *testing.T, id uuid.UUID, kind string) int {
	t.Helper()
	var attempts int
	if err := e.pool.QueryRow(context.Background(),
		`SELECT attempt FROM river_job WHERE kind = $1 AND args->>'assetId' = $2`, kind, id.String()).
		Scan(&attempts); err != nil {
		t.Fatalf("read river job attempt: %v", err)
	}
	return attempts
}

// riverJobState reads the state of one River job of the asset.
func (e *env) riverJobState(t *testing.T, id uuid.UUID, kind string) string {
	t.Helper()
	var state string
	if err := e.pool.QueryRow(context.Background(),
		`SELECT state FROM river_job WHERE kind = $1 AND args->>'assetId' = $2`, kind, id.String()).
		Scan(&state); err != nil {
		t.Fatalf("read river job state: %v", err)
	}
	return state
}

// TestEnqueueMlJobsMatrix pins the seam's slot-activation matrix on
// the queue's own dispatch surface: an image asset with an active
// detect model gains exactly one face detection job; with no model
// active the dispatch is a silent zero; a video asset dispatches
// nothing.
func TestEnqueueMlJobsMatrix(t *testing.T) {
	e := newEnv(t)
	takenAt := time.Date(2020, 3, 1, 12, 0, 0, 0, time.UTC).Truncate(time.Microsecond)
	id := e.insertAsset(t, "originals/face/portrait.jpg", portraitFixture(t, "face-portrait.jpg"), takenAt, `{}`)
	video := e.insertAssetKind(t, "originals/face/video.mp4", portraitFixture(t, "face-portrait.jpg"), takenAt,
		`{}`, dbgen.MediaKindVideo, "video/mp4", 0)

	// No active model anywhere: the seam is a silent no-op.
	if err := e.queue.EnqueueMlJobs(context.Background(), id, dbgen.MediaKindImage); err != nil {
		t.Fatalf("EnqueueMlJobs without models error = %v, want the silent nil", err)
	}
	if got := e.riverJobsOf(t, id)["face_detect"]; got != 0 {
		t.Errorf("face detect jobs = %d, want 0 with no active model", got)
	}

	// An active detect model dispatches exactly one job.
	e.activateFixtureModel(t, yunetFixtureSpec)
	if err := e.queue.EnqueueMlJobs(context.Background(), id, dbgen.MediaKindImage); err != nil {
		t.Fatalf("EnqueueMlJobs with an active model error = %v, want nil", err)
	}
	if got := e.riverJobsOf(t, id)["face_detect"]; got != 1 {
		t.Errorf("face detect jobs = %d, want exactly 1", got)
	}

	// A video asset dispatches nothing: video ML belongs to a later
	// milestone.
	if err := e.queue.EnqueueMlJobs(context.Background(), video, dbgen.MediaKindVideo); err != nil {
		t.Fatalf("EnqueueMlJobs of a video asset error = %v, want nil", err)
	}
	if got := e.riverJobsOf(t, video)["face_detect"]; got != 0 {
		t.Errorf("video face detect jobs = %d, want 0", got)
	}
}

// TestEnqueueMlJobsWiredSlots pins the seam's closed matrix: the
// clip-embed and ocr arms are WIRED (this test is the unwired-era
// tripwire flipped green: its former self asserted zero dispatches
// while the arms waited for their workers). With their slots
// active, an image asset gains exactly one clip_embed and one ocr job;
// the face-detect arm stays silent with its own slot unfilled.
func TestEnqueueMlJobsWiredSlots(t *testing.T) {
	e := newEnv(t)
	e.activateFixtureModel(t, fixtureModelSpec{
		id:        "clip-fixture",
		slot:      ml.SlotClipEmbed,
		file:      identityFixtureSpec.file,
		sha256:    identityFixtureSpec.sha256,
		inputSize: identityFixtureSpec.inputSize,
		dim:       512,
		params:    identityFixtureSpec.params,
	})
	e.activateFixtureModel(t, fixtureModelSpec{
		id:        "ocr-fixture",
		slot:      ml.SlotOcr,
		file:      identityFixtureSpec.file,
		sha256:    identityFixtureSpec.sha256,
		inputSize: identityFixtureSpec.inputSize,
		params:    identityFixtureSpec.params,
	})

	takenAt := time.Date(2020, 3, 1, 12, 0, 0, 0, time.UTC).Truncate(time.Microsecond)
	id := e.insertAsset(t, "originals/face/portrait.jpg", portraitFixture(t, "face-portrait.jpg"), takenAt, `{}`)
	if err := e.queue.EnqueueMlJobs(context.Background(), id, dbgen.MediaKindImage); err != nil {
		t.Fatalf("EnqueueMlJobs error = %v, want nil", err)
	}
	kinds := e.riverJobsOf(t, id)
	if kinds["clip_embed"] != 1 || kinds["ocr"] != 1 {
		t.Errorf("river jobs = %v, want exactly one clip_embed and one ocr job (the wired arms)", kinds)
	}
	if got := e.riverJobTotal(t, id); got != 2 {
		t.Errorf("river jobs = %d, want 2 (the face-detect slot is unfilled and dispatches nothing)", got)
	}
}

// TestSeamFiresFaceDetectAtAssetReady pins the seam's position: the
// shared pipeline completion -- the guarded decrement emptying
// pending_jobs -- dispatches the face detection job for an image
// asset with an active model, the job runs through the queue for
// real, and the faces land. It also pins the NO-AUTO-CHAIN decision:
// the asset's river jobs stay exactly the pipeline's metadata job and
// the detection job -- the detection's completion enqueues NOTHING
// (the recognition trigger surface is the model rerun plus the
// chaining restore's incremental decision, recorded in the
// worker's comment).
func TestSeamFiresFaceDetectAtAssetReady(t *testing.T) {
	requireExiftool(t)
	requireThumbnails(t)
	requireMlRuntime(t)
	e := newEnv(t)
	modelID := e.activateFixtureModel(t, yunetFixtureSpec)

	takenAt := time.Date(2020, 3, 1, 12, 0, 0, 0, time.UTC).Truncate(time.Microsecond)
	id := e.insertAsset(t, "originals/face/portrait.jpg", portraitFixture(t, "face-portrait.jpg"), takenAt,
		`{"deviceCreatedAt":"2020-03-01T12:00:00Z","deviceTimeZone":"Asia/Shanghai"}`)
	if err := e.queue.EnqueueMetadata(context.Background(), id); err != nil {
		t.Fatalf("enqueue metadata job: %v", err)
	}
	eventually(t, jobWaitTimeout, "the face rows to land", func() bool {
		return len(e.facesOf(t, id)) >= 1
	})

	faces := e.facesOf(t, id)
	if len(faces) != 1 {
		t.Fatalf("faces rows = %d, want exactly 1", len(faces))
	}
	requireFaceBounds(t, faces[0], "the seam-dispatched face")
	if faces[0].modelID != modelID {
		t.Errorf("face model_id = %q, want the active detect model %q", faces[0].modelID, modelID)
	}

	kinds := e.riverJobsOf(t, id)
	if kinds["metadata"] != 1 || kinds["face_detect"] != 1 || e.riverJobTotal(t, id) != 2 {
		t.Errorf("river jobs = %v, want exactly one metadata and one face_detect job (no continuation)", kinds)
	}
	if got := countEvents(e.bus, "asset.ready"); got != 1 {
		t.Errorf("asset.ready events = %d, want exactly 1", got)
	}
}

// TestSeamWithoutModelsLeavesPipelineClean pins the degraded mode end
// to end: the full pipeline of an image asset with
// no active ML model runs and announces exactly as before -- the seam
// dispatched nothing and errored nowhere.
func TestSeamWithoutModelsLeavesPipelineClean(t *testing.T) {
	requireExiftool(t)
	e := newEnv(t)

	takenAt := time.Date(2020, 3, 1, 12, 0, 0, 0, time.UTC).Truncate(time.Microsecond)
	id := e.insertAsset(t, "originals/face/portrait.jpg", portraitFixture(t, "face-portrait.jpg"), takenAt,
		`{"deviceCreatedAt":"2020-03-01T12:00:00Z","deviceTimeZone":"Asia/Shanghai"}`)
	if err := e.queue.EnqueueMetadata(context.Background(), id); err != nil {
		t.Fatalf("enqueue metadata job: %v", err)
	}
	eventually(t, jobWaitTimeout, "the asset to become ready", func() bool {
		return e.assetOf(t, id).readyAt.Valid
	})

	if got := e.riverJobsOf(t, id)["face_detect"]; got != 0 {
		t.Errorf("face detect jobs = %d, want 0 with no active model", got)
	}
	if got := countEvents(e.bus, "asset.ready"); got != 1 {
		t.Errorf("asset.ready events = %d, want exactly 1", got)
	}
}
