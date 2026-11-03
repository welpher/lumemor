// The CLIP embedding job tests (M3a Task 8): the worker wiring over
// the committed const512 fixture (the digest gate, the decode, the
// embedding row landing with every field), the upsert idempotence (a
// re-run overwrites, the primary key keeps one row, other models'
// rows survive -- the coexistence policy), THE COLUMN-DIMENSION GUARD
// (Review Focus #5's insert side: an active model whose embedding
// dimension disagrees with the column's current dimension fails the
// job with zero rows), the degraded modes, and the readiness seam's
// clip arm (the Task 6 tripwire flipped: an active clip model now
// dispatches the embedding job).
//
// The inference-bearing tests skip on hosts without the onnxruntime
// shared library (and the decode-bearing ones without libvips): the
// containerized `make docker-test` gate runs everything for real. The
// guard and matrix tests need neither.
package jobs

import (
	"context"
	"io"
	"log/slog"
	"math"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/pgvector/pgvector-go"
	"github.com/riverqueue/river"

	dbgen "github.com/welpher/lumemor/server/internal/db/gen"
	"github.com/welpher/lumemor/server/internal/ml"
)

// clipFixtureSpec is the const512 graph as the active clip-embed
// model: a 512-dimensional embedding model whose output matches the
// clip_embeddings column's built-in dimension, so the insert guard
// passes and the row lands.
var clipFixtureSpec = fixtureModelSpec{
	id:        "clip-const512",
	slot:      ml.SlotClipEmbed,
	file:      "const512.onnx",
	sha256:    "92d6b93b89bf6ccec24222be86689b6ff18370503d3d71316d5d325bd2116766",
	inputSize: ml.ManifestSize{Width: 224, Height: 224},
	dim:       512,
}

// embeddingRow is the clip_embeddings row as the assertions read it.
type embeddingRow struct {
	modelID   string
	dims      int
	embedding []float32
	createdAt time.Time
	updatedAt time.Time
}

// clipEmbeddingsOf loads the asset's embedding rows across every
// model.
func (e *env) clipEmbeddingsOf(t *testing.T, assetID uuid.UUID) []embeddingRow {
	t.Helper()
	rows, err := e.pool.Query(context.Background(), `
		SELECT model_id, vector_dims(embedding), embedding, created_at, updated_at
		FROM clip_embeddings WHERE asset_id = $1`, pgUUID(assetID))
	if err != nil {
		t.Fatalf("query clip embeddings of asset %s: %v", assetID, err)
	}
	defer rows.Close()
	var out []embeddingRow
	for rows.Next() {
		var row embeddingRow
		var vector pgvector.Vector
		if err := rows.Scan(&row.modelID, &row.dims, &vector, &row.createdAt, &row.updatedAt); err != nil {
			t.Fatalf("scan clip embedding row: %v", err)
		}
		row.embedding = vector.Slice()
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate clip embeddings: %v", err)
	}
	return out
}

// clipEmbedWorker builds the worker over the environment for a direct
// Work invocation (the River machinery has its own tests).
func (e *env) clipEmbedWorker() *ClipEmbedWorker {
	return &ClipEmbedWorker{
		pool:     e.pool,
		dataDir:  e.dataDir,
		registry: e.registry,
		logger:   slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
}

// runClipEmbed drives one direct Work invocation for the asset.
func runClipEmbed(t *testing.T, e *env, assetID uuid.UUID) error {
	t.Helper()
	return e.clipEmbedWorker().Work(context.Background(), &river.Job[ClipEmbedArgs]{
		Args: ClipEmbedArgs{AssetID: assetID},
	})
}

// TestClipEmbedJobPersistsEmbedding pins the worker wiring end to end
// on the fixture graph: the asset's embedding lands as one
// clip_embeddings row with every field asserted -- the active model's
// id, the 512-dimension unit vector (the const512 graph emits 512
// ones, the wrapper normalizes), the creation timestamp inside the
// run's window (µs-truncated comparison).
func TestClipEmbedJobPersistsEmbedding(t *testing.T) {
	requireThumbnails(t)
	requireMlRuntime(t)
	e := newEnv(t)
	modelID := e.activateFixtureModel(t, clipFixtureSpec)

	before := time.Now().Truncate(time.Microsecond)
	id := e.insertFaceAsset(t, "originals/clip/portrait.jpg", portraitFixture(t, "face-portrait.jpg"))
	if err := runClipEmbed(t, e, id); err != nil {
		t.Fatalf("Work error = %v, want nil", err)
	}
	after := time.Now().Truncate(time.Microsecond)

	rows := e.clipEmbeddingsOf(t, id)
	if len(rows) != 1 {
		t.Fatalf("clip_embeddings rows = %d, want exactly 1", len(rows))
	}
	row := rows[0]
	if row.modelID != modelID {
		t.Errorf("embedding model_id = %q, want the active clip model %q", row.modelID, modelID)
	}
	if row.dims != 512 {
		t.Errorf("embedding dimension = %d, want 512 (the model's output)", row.dims)
	}
	want := float32(1 / math.Sqrt(512))
	norm := 0.0
	for _, component := range row.embedding {
		norm += float64(component) * float64(component)
		if diff := math.Abs(float64(component - want)); diff > 1e-6 {
			t.Fatalf("embedding component = %v, want the unit fraction %v", component, want)
		}
	}
	if diff := math.Abs(norm - 1); diff > 1e-5 {
		t.Errorf("embedding L2 norm = %v, want 1", norm)
	}
	if row.createdAt.Before(before) || row.createdAt.After(after) {
		t.Errorf("embedding created_at = %s, want it inside the run's window [%s, %s]",
			row.createdAt, before, after)
	}
}

// TestClipEmbedJobUpsertIdempotent pins the upsert contract and the
// model-switch coexistence: a re-run of the same model overwrites the
// same row (the primary key keeps one row, never a pile), and the
// seeded row of a different model survives both runs untouched
// (Review Focus #1's embedding side).
func TestClipEmbedJobUpsertIdempotent(t *testing.T) {
	requireThumbnails(t)
	requireMlRuntime(t)
	e := newEnv(t)
	active := e.activateFixtureModel(t, clipFixtureSpec)
	other := e.registerFixtureModel(t, fixtureModelSpec{
		id:        "clip-other",
		slot:      ml.SlotClipEmbed,
		file:      clipFixtureSpec.file,
		sha256:    clipFixtureSpec.sha256,
		inputSize: clipFixtureSpec.inputSize,
		dim:       512,
	})

	id := e.insertFaceAsset(t, "originals/clip/portrait.jpg", portraitFixture(t, "face-portrait.jpg"))

	// The other model's embedding row: seeded directly, the way a
	// previous active model's products sit in the table after a
	// switch.
	if _, err := e.pool.Exec(context.Background(), `
		INSERT INTO clip_embeddings (asset_id, model_id, embedding)
		VALUES ($1, $2, $3::vector)`, pgUUID(id), other, "["+repeatOnes(512)+"]"); err != nil {
		t.Fatalf("seed the other model's embedding: %v", err)
	}

	for run := 1; run <= 2; run++ {
		if err := runClipEmbed(t, e, id); err != nil {
			t.Fatalf("run %d Work error = %v, want nil", run, err)
		}
		rows := e.clipEmbeddingsOf(t, id)
		if len(rows) != 2 {
			t.Fatalf("run %d clip_embeddings rows = %d, want 2 (the active model's upsert plus the other model's survivor)", run, len(rows))
		}
		var activeRows, otherRows int
		for _, row := range rows {
			if row.modelID == active {
				activeRows++
			}
			if row.modelID == other {
				otherRows++
			}
		}
		if activeRows != 1 {
			t.Errorf("run %d active model rows = %d, want exactly 1 (overwritten, not piled)", run, activeRows)
		}
		if otherRows != 1 {
			t.Errorf("run %d other model rows = %d, want the survivor 1", run, otherRows)
		}
	}
}

// repeatOnes renders "1.0,1.0,..." of the given count (the seeded
// vector's array literal).
func repeatOnes(count int) string {
	const one = "1.0"
	out := ""
	for i := 0; i < count; i++ {
		if i > 0 {
			out += ","
		}
		out += one
	}
	return out
}

// TestClipEmbedJobDimensionGuardRejectsMismatch pins the insert-side
// dimension guard (Review Focus #5): an active clip model whose
// embedding dimension disagrees with the column's CURRENT dimension
// (a 768-dim model activated behind the registry's back -- the ALTER
// never ran) fails the job with zero rows landing, instead of letting
// the database reject the insert row by row.
func TestClipEmbedJobDimensionGuardRejectsMismatch(t *testing.T) {
	requireThumbnails(t)
	requireMlRuntime(t)
	e := newEnv(t)
	drifted := e.registerFixtureModel(t, fixtureModelSpec{
		id:        "clip-const768-drifted",
		slot:      ml.SlotClipEmbed,
		file:      "const768.onnx",
		sha256:    "0081d3e576a85d76090f2a59fb7e7ff3a6623d989cdac4474e8ed7ead85a9304",
		inputSize: ml.ManifestSize{Width: 224, Height: 224},
		dim:       768,
	})
	// Activate the drifted model directly in the table: the registry's
	// activation would have re-dimensioned the column (the transaction
	// of Task 2/4); bypassing it simulates the drifted state the guard
	// exists for.
	if _, err := e.pool.Exec(context.Background(),
		"UPDATE models SET is_active = true WHERE id = $1", drifted); err != nil {
		t.Fatalf("force-activate the drifted model: %v", err)
	}

	id := e.insertFaceAsset(t, "originals/clip/drifted.jpg", portraitFixture(t, "face-portrait.jpg"))
	err := runClipEmbed(t, e, id)
	if err == nil {
		t.Fatal("Work over a dimension-mismatched model error = nil, want the guard refusal")
	}
	if rows := e.clipEmbeddingsOf(t, id); len(rows) != 0 {
		t.Errorf("clip_embeddings rows = %d, want 0 (the guard precedes any insert)", len(rows))
	}

	// The asset itself is untouched: the ML failure never reaches the
	// pipeline's state.
	state := e.assetOf(t, id)
	if !state.readyAt.Valid {
		t.Error("ready_at is unset, want the pipeline's stamp to stand")
	}
}

// TestClipEmbedJobNoActiveModelSkipsCleanly pins the degraded-mode
// worker: a stale or mis-routed job against a slot with no active
// model completes as a no-op.
func TestClipEmbedJobNoActiveModelSkipsCleanly(t *testing.T) {
	e := newEnv(t)

	id := e.insertFaceAsset(t, "originals/clip/none.jpg", portraitFixture(t, "face-portrait.jpg"))
	if err := runClipEmbed(t, e, id); err != nil {
		t.Fatalf("Work error = %v, want the clean skip", err)
	}
	if rows := e.clipEmbeddingsOf(t, id); len(rows) != 0 {
		t.Errorf("clip_embeddings rows = %d, want 0", len(rows))
	}
}

// TestClipEmbedJobNonImageIsNoOp pins the kind gate: a video row
// (mis-routed job) completes as a no-op.
func TestClipEmbedJobNonImageIsNoOp(t *testing.T) {
	e := newEnv(t)
	e.activateFixtureModel(t, clipFixtureSpec)

	takenAt := time.Date(2020, 3, 1, 12, 0, 0, 0, time.UTC).Truncate(time.Microsecond)
	id := e.insertAssetKind(t, "originals/clip/video.mp4", portraitFixture(t, "face-portrait.jpg"), takenAt,
		`{}`, dbgen.MediaKindVideo, "video/mp4", 0)
	if err := runClipEmbed(t, e, id); err != nil {
		t.Fatalf("Work error = %v, want the no-op", err)
	}
	if rows := e.clipEmbeddingsOf(t, id); len(rows) != 0 {
		t.Errorf("clip_embeddings rows = %d, want 0", len(rows))
	}
}

// TestClipEmbedJobMissingAssetCompletes pins the gone-asset path.
func TestClipEmbedJobMissingAssetCompletes(t *testing.T) {
	e := newEnv(t)

	id, err := uuid.NewV7()
	if err != nil {
		t.Fatalf("new asset id: %v", err)
	}
	if err := runClipEmbed(t, e, id); err != nil {
		t.Fatalf("Work error = %v, want the no-op", err)
	}
}

// TestSeamWiresClipAndOcrAtAssetReady pins the seam's closed loop: an
// image asset finishing its pipeline with the clip-embed AND ocr slots
// active dispatches both jobs, both run through the real queue, and
// both products land -- the tripwire Task 6 left (its
// TestEnqueueMlJobsUnwiredSlots expected zero dispatches) flipped
// green by this task's wiring.
func TestSeamWiresClipAndOcrAtAssetReady(t *testing.T) {
	requireExiftool(t)
	requireThumbnails(t)
	requireMlRuntime(t)
	e := newEnv(t)
	clipModel := e.activateFixtureModel(t, clipFixtureSpec)
	ocrModel := e.registerOcrFixtureModel(t)

	takenAt := time.Date(2020, 3, 1, 12, 0, 0, 0, time.UTC).Truncate(time.Microsecond)
	id := e.insertAsset(t, "originals/clip/seam.jpg", portraitFixture(t, "face-portrait.jpg"), takenAt,
		`{"deviceCreatedAt":"2020-03-01T12:00:00Z","deviceTimeZone":"Asia/Shanghai"}`)
	if err := e.queue.EnqueueMetadata(context.Background(), id); err != nil {
		t.Fatalf("enqueue metadata job: %v", err)
	}
	eventually(t, jobWaitTimeout, "the clip embedding and ocr rows to land", func() bool {
		return len(e.clipEmbeddingsOf(t, id)) >= 1 && e.ocrTextOf(t, id) != nil
	})

	rows := e.clipEmbeddingsOf(t, id)
	if len(rows) != 1 || rows[0].modelID != clipModel || rows[0].dims != 512 {
		t.Errorf("clip embeddings = %+v, want the single 512-dim row of the active model", rows)
	}
	text := e.ocrTextOf(t, id)
	if text == nil || text.modelID != ocrModel || text.text != "OCR OCR\nOCR" || text.language != nil {
		t.Errorf("ocr text = %+v, want the fixture pipeline's row (language NULL)", text)
	}

	// The seam dispatched exactly the two ML jobs of this task's arms:
	// the face-detect slot is empty here, so the asset's river jobs are
	// the pipeline's metadata job plus the clip and ocr jobs.
	kinds := e.riverJobsOf(t, id)
	if kinds["metadata"] != 1 || kinds["clip_embed"] != 1 || kinds["ocr"] != 1 ||
		e.riverJobTotal(t, id) != 3 {
		t.Errorf("river jobs = %v, want metadata, clip_embed and ocr exactly once each", kinds)
	}
	if got := countEvents(e.bus, "asset.ready"); got != 1 {
		t.Errorf("asset.ready events = %d, want exactly 1", got)
	}
}

// ocrTextRow is the ocr_texts row as the assertions read it. The
// language column is read as a nullable string: the worker stores NULL
// (the mixed Chinese-English corpus carries no single honest language
// tag).
type ocrTextRow struct {
	modelID  string
	text     string
	language *string
}

// ocrTextOf loads the asset's ocr_texts row, or nil when none landed.
func (e *env) ocrTextOf(t *testing.T, assetID uuid.UUID) *ocrTextRow {
	t.Helper()
	var row ocrTextRow
	err := e.pool.QueryRow(context.Background(), `
		SELECT model_id, text, language FROM ocr_texts WHERE asset_id = $1`, pgUUID(assetID)).
		Scan(&row.modelID, &row.text, &row.language)
	if err == pgx.ErrNoRows {
		return nil
	}
	if err != nil {
		t.Fatalf("query ocr text of asset %s: %v", assetID, err)
	}
	return &row
}
