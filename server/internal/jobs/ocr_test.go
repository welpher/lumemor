// The OCR job tests: the worker wiring over the committed
// det+rec fixture pair registered as ONE two-file model (the registry's
// multi-file convention: the manifest's first file is the detection
// graph, the second the recognition graph), the upsert idempotence
// (the ocr_texts primary key keeps one row per asset), and the degraded
// modes.
package jobs

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/riverqueue/river"

	dbgen "github.com/welpher/lumemor/server/internal/db/gen"
	"github.com/welpher/lumemor/server/internal/ml"
)

// ocrFixtureFiles are the committed OCR fixture pair with their pinned
// digests: the detection graph (the manifest's primary file) and the
// recognition graph.
var ocrFixtureFiles = []ml.ManifestFile{
	{
		Name:   "ocr_det.onnx",
		SHA256: "4323256515d055a117c3605a2e8efb4d71274fb08b476658c45daed885348955",
	},
	{
		Name:   "ocr_rec.onnx",
		SHA256: "3a7aaaf4f653584c2d68d1e8a82f844f5f73d1cebf835d24b9275ee0fa1340c4",
	},
}

// registerOcrFixtureModel lays the two-file OCR fixture model out under
// the environment's models dir (manifest.json listing the detection
// graph first, the recognition graph second -- the registry's
// multi-file convention), scans it into the registry, activates it and
// answers the model id.
func (e *env) registerOcrFixtureModel(t *testing.T) string {
	t.Helper()

	id := "ocr-fixture"
	dir := filepath.Join(e.modelsDir, id)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("create model dir: %v", err)
	}
	files := make([]ml.ManifestFile, len(ocrFixtureFiles))
	for i, file := range ocrFixtureFiles {
		weights, err := os.ReadFile(filepath.Join("..", "ml", "testdata", file.Name))
		if err != nil {
			t.Fatalf("read fixture weights %s: %v", file.Name, err)
		}
		if err := os.WriteFile(filepath.Join(dir, file.Name), weights, 0o644); err != nil {
			t.Fatalf("write fixture weights: %v", err)
		}
		files[i] = ml.ManifestFile{
			Name:   file.Name,
			SHA256: file.SHA256,
			Bytes:  int64(len(weights)),
		}
	}
	manifest := ml.ManifestModel{
		ID:        id,
		Slot:      "ocr",
		Version:   "fixture",
		License:   "fixture",
		InputSize: ml.ManifestSize{Width: 960, Height: 960},
		Params:    map[string]any{"detThreshold": 0.3, "recHeight": 48},
		Files:     files,
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
	if _, err := e.registry.Activate(ml.SlotOcr, id); err != nil {
		t.Fatalf("activate the fixture model %s: %v", id, err)
	}
	return id
}

// ocrWorker builds the worker over the environment for a direct Work
// invocation (the River machinery has its own tests).
func (e *env) ocrWorker() *OcrWorker {
	return &OcrWorker{
		pool:     e.pool,
		dataDir:  e.dataDir,
		registry: e.registry,
		logger:   slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
}

// runOcr drives one direct Work invocation for the asset.
func runOcr(t *testing.T, e *env, assetID uuid.UUID) error {
	t.Helper()
	return e.ocrWorker().Work(context.Background(), &river.Job[OcrArgs]{
		Args: OcrArgs{AssetID: assetID},
	})
}

// TestOcrJobPersistsText pins the worker wiring end to end on the
// fixture pair: the asset's text lands as one ocr_texts row with every
// field asserted -- the active model's id, the fixture pipeline's
// reading-order text, and the language column left NULL (the honest
// tag for a mixed corpus).
func TestOcrJobPersistsText(t *testing.T) {
	requireThumbnails(t)
	requireMlRuntime(t)
	e := newEnv(t)
	modelID := e.registerOcrFixtureModel(t)

	id := e.insertFaceAsset(t, "originals/ocr/portrait.jpg", portraitFixture(t, "face-portrait.jpg"))
	if err := runOcr(t, e, id); err != nil {
		t.Fatalf("Work error = %v, want nil", err)
	}

	row := e.ocrTextOf(t, id)
	if row == nil {
		t.Fatal("ocr_texts row missing after the job ran")
	}
	if row.modelID != modelID {
		t.Errorf("ocr text model_id = %q, want the active ocr model %q", row.modelID, modelID)
	}
	if row.text != "OCR OCR\nOCR" {
		t.Errorf("ocr text = %q, want the fixture pipeline's reading-order text %q", row.text, "OCR OCR\nOCR")
	}
	if row.language != nil {
		t.Errorf("ocr text language = %q, want NULL (the mixed corpus carries no honest tag)", *row.language)
	}
}

// TestOcrJobUpsertIdempotent pins the upsert contract: a re-run
// overwrites the same row (the asset's primary key keeps one row,
// never a pile) with the same converged text.
func TestOcrJobUpsertIdempotent(t *testing.T) {
	requireThumbnails(t)
	requireMlRuntime(t)
	e := newEnv(t)
	e.registerOcrFixtureModel(t)

	id := e.insertFaceAsset(t, "originals/ocr/portrait.jpg", portraitFixture(t, "face-portrait.jpg"))
	for run := 1; run <= 2; run++ {
		if err := runOcr(t, e, id); err != nil {
			t.Fatalf("run %d Work error = %v, want nil", run, err)
		}
		var count int
		if err := e.pool.QueryRow(context.Background(),
			"SELECT count(*) FROM ocr_texts WHERE asset_id = $1", pgUUID(id)).Scan(&count); err != nil {
			t.Fatalf("count ocr rows: %v", err)
		}
		if count != 1 {
			t.Errorf("run %d ocr_texts rows = %d, want exactly 1 (overwritten, not piled)", run, count)
		}
	}
}

// TestOcrJobNoActiveModelSkipsCleanly pins the degraded-mode worker.
func TestOcrJobNoActiveModelSkipsCleanly(t *testing.T) {
	e := newEnv(t)

	id := e.insertFaceAsset(t, "originals/ocr/none.jpg", portraitFixture(t, "face-portrait.jpg"))
	if err := runOcr(t, e, id); err != nil {
		t.Fatalf("Work error = %v, want the clean skip", err)
	}
	if row := e.ocrTextOf(t, id); row != nil {
		t.Errorf("ocr_texts row = %+v, want none", row)
	}
}

// TestOcrJobNonImageIsNoOp pins the kind gate: a video row
// (mis-routed job) completes as a no-op.
func TestOcrJobNonImageIsNoOp(t *testing.T) {
	e := newEnv(t)
	e.registerOcrFixtureModel(t)

	takenAt := time.Date(2020, 3, 1, 12, 0, 0, 0, time.UTC).Truncate(time.Microsecond)
	id := e.insertAssetKind(t, "originals/ocr/video.mp4", portraitFixture(t, "face-portrait.jpg"), takenAt,
		`{}`, dbgen.MediaKindVideo, "video/mp4", 0)
	if err := runOcr(t, e, id); err != nil {
		t.Fatalf("Work error = %v, want the no-op", err)
	}
	if row := e.ocrTextOf(t, id); row != nil {
		t.Errorf("ocr_texts row = %+v, want none", row)
	}
}

// TestOcrJobMissingAssetCompletes pins the gone-asset path.
func TestOcrJobMissingAssetCompletes(t *testing.T) {
	e := newEnv(t)

	id, err := uuid.NewV7()
	if err != nil {
		t.Fatalf("new asset id: %v", err)
	}
	if err := runOcr(t, e, id); err != nil {
		t.Fatalf("Work error = %v, want the no-op", err)
	}
}
