// The model registry tests: every behavior runs against a
// migrated throwaway database (the dbtest container pattern of the api
// tests) with fixture models laid out on disk exactly the way the
// download service and the directory scan lay out real ones: one
// directory per model id holding manifest.json plus the weights. The
// pins are the activation contract (unique active per slot, old rows
// surviving a switch), the dimension policy (same-dimension switches
// coexist without touching a single embedding row; cross-dimension
// switches wipe and re-dimension the embeddings column in the same
// transaction, and a mid-transaction failure rolls back to the
// original state), the checksum gate on activation, and the scan's
// registration semantics.
package ml

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pgvector/pgvector-go"

	"github.com/welpher/lumemor/server/internal/db"
	"github.com/welpher/lumemor/server/internal/db/dbtest"
	dbgen "github.com/welpher/lumemor/server/internal/db/gen"
)

// Pinned digests of the committed fixtures the registry tests copy
// into their throwaway model directories.
const (
	const128FixtureSHA256 = "e5d4940db6bddc912dee319434394700cf82fd8950cc32e609435ae7f9597263"
	const512FixtureSHA256 = "92d6b93b89bf6ccec24222be86689b6ff18370503d3d71316d5d325bd2116766"
	const768FixtureSHA256 = "0081d3e576a85d76090f2a59fb7e7ff3a6623d989cdac4474e8ed7ead85a9304"
)

// fixtureModel describes one fixture model to lay out on disk: the
// testdata file it copies, the slot it fills and the dimension it
// reports.
type fixtureModel struct {
	id   string
	slot string // the manifest's hyphenated slot value
	file string
	sha  string
	dim  int
}

// fixtureModelParams is the per-model params object the fixture
// manifest carries; the scan must round-trip it into the database and
// Active must hand it back as the model's params.
var fixtureModelParams = map[string]any{"scoreThreshold": 0.6}

// writeFixtureModel lays out one model directory under modelsDir the
// way real models are laid out: manifest.json plus the weights file,
// pinned by digest.
func writeFixtureModel(t *testing.T, modelsDir string, fm fixtureModel) {
	t.Helper()

	dir := filepath.Join(modelsDir, fm.id)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("create model dir %s: %v", dir, err)
	}
	data, err := os.ReadFile(filepath.Join("testdata", fm.file))
	if err != nil {
		t.Fatalf("read fixture %s: %v", fm.file, err)
	}
	if err := os.WriteFile(filepath.Join(dir, fm.file), data, 0o644); err != nil {
		t.Fatalf("write model weights: %v", err)
	}

	manifest := ManifestModel{
		ID:        fm.id,
		Slot:      fm.slot,
		Version:   "fixture",
		InputSize: ManifestSize{Width: 8, Height: 8},
		Dim:       fm.dim,
		Params:    fixtureModelParams,
		Files:     []ManifestFile{{Name: fm.file, SHA256: fm.sha, Bytes: int64(len(data))}},
	}
	blob, err := json.Marshal(manifest)
	if err != nil {
		t.Fatalf("marshal fixture manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), blob, 0o644); err != nil {
		t.Fatalf("write fixture manifest: %v", err)
	}
}

// newRegistryEnv provisions a migrated throwaway database and a
// registry whose models dir carries the given fixture models (not yet
// registered: registering is the scan's job, which the tests exercise
// explicitly).
func newRegistryEnv(t *testing.T, models ...fixtureModel) (*Registry, *pgxpool.Pool, string) {
	t.Helper()

	modelsDir := t.TempDir()
	for _, m := range models {
		writeFixtureModel(t, modelsDir, m)
	}

	pool := dbtest.New(t)
	if err := db.Migrate(t.Context(), pool); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
	return NewRegistry(pool, dbgen.New(pool), modelsDir), pool, modelsDir
}

// pgUUID wraps a uuid.UUID the way the sqlc-generated code expects.
func pgUUID(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: id, Valid: true}
}

// seedClipEmbedding inserts one asset with a clip embedding row for
// the given model, the minimum the dimension-policy tests need on top
// of the migrated schema.
func seedClipEmbedding(t *testing.T, pool *pgxpool.Pool, modelID string, dim int) {
	t.Helper()

	userID, err := uuid.NewV7()
	if err != nil {
		t.Fatalf("generate user id: %v", err)
	}
	if _, err := pool.Exec(t.Context(),
		"INSERT INTO users (id, name, email, password_hash) VALUES ($1, $2, $3, $4)",
		pgUUID(userID), "Registry Test", "registry-"+userID.String()+"@example.com", "x"); err != nil {
		t.Fatalf("insert user: %v", err)
	}
	assetID, err := uuid.NewV7()
	if err != nil {
		t.Fatalf("generate asset id: %v", err)
	}
	if _, err := pool.Exec(t.Context(), `
		INSERT INTO assets (id, owner_id, storage_kind, file_path, checksum, original_filename,
			file_size, mime_type, media_kind, width, height, taken_at, taken_at_source)
		VALUES ($1, $2, 'blob', $3, $4, $5, 1, 'image/jpeg', 'image', 64, 64, $6, 'import')`,
		pgUUID(assetID), pgUUID(userID), "registry/"+assetID.String()+".jpg",
		assetID.String(), assetID.String()+".jpg", time.Now()); err != nil {
		t.Fatalf("insert asset: %v", err)
	}
	if _, err := pool.Exec(t.Context(),
		"INSERT INTO clip_embeddings (asset_id, model_id, embedding) VALUES ($1, $2, $3)",
		pgUUID(assetID), modelID, pgvector.NewVector(make([]float32, dim))); err != nil {
		t.Fatalf("insert clip embedding: %v", err)
	}
}

// columnDim reads the embedding column's current dimension from the
// database catalog (pgvector stores it in the typmod).
func columnDim(t *testing.T, pool *pgxpool.Pool) int {
	t.Helper()
	var dim int
	if err := pool.QueryRow(t.Context(), `
		SELECT atttypmod FROM pg_attribute
		WHERE attrelid = 'clip_embeddings'::regclass AND attname = 'embedding'`).Scan(&dim); err != nil {
		t.Fatalf("read embedding column dim: %v", err)
	}
	return dim
}

// countEmbeddings counts the clip_embeddings rows.
func countEmbeddings(t *testing.T, pool *pgxpool.Pool) int {
	t.Helper()
	var n int
	if err := pool.QueryRow(t.Context(), "SELECT count(*) FROM clip_embeddings").Scan(&n); err != nil {
		t.Fatalf("count clip_embeddings: %v", err)
	}
	return n
}

// modelActiveFlag reads one model row's is_active straight from the
// database: the Model value the registry hands out deliberately
// carries no activation flag -- activation state is a table fact, and
// the tests pin it where it lives.
func modelActiveFlag(t *testing.T, pool *pgxpool.Pool, id string) bool {
	t.Helper()
	var active bool
	if err := pool.QueryRow(t.Context(), "SELECT is_active FROM models WHERE id = $1", id).Scan(&active); err != nil {
		t.Fatalf("read is_active of %q: %v", id, err)
	}
	return active
}

// hnswIndexValid reports whether the embeddings HNSW index exists and
// is valid: the cross-dimension switch rebuilds it on the emptied
// table, and the rollback must leave the original one intact.
func hnswIndexValid(t *testing.T, pool *pgxpool.Pool) bool {
	t.Helper()
	var valid bool
	if err := pool.QueryRow(t.Context(), `
		SELECT i.indisvalid FROM pg_index i
		JOIN pg_class c ON c.oid = i.indexrelid
		WHERE c.relname = 'clip_embeddings_hnsw_idx'`).Scan(&valid); err != nil {
		t.Fatalf("read hnsw index state: %v", err)
	}
	return valid
}

// TestListEmpty pins the honest empty state: a fresh instance lists no
// models and reports no error.
func TestListEmpty(t *testing.T) {
	registry, _, _ := newRegistryEnv(t)

	models, err := registry.List()
	if err != nil {
		t.Fatalf("List() error = %v, want nil", err)
	}
	if len(models) != 0 {
		t.Errorf("List() = %d models, want 0", len(models))
	}
}

// TestActiveWithoutModel pins the no-model degradation contract: an
// unfilled slot answers ErrNoActiveModel, the sentinel the job layer
// uses to skip ML dispatch without an error.
func TestActiveWithoutModel(t *testing.T) {
	registry, _, _ := newRegistryEnv(t)

	_, err := registry.Active(SlotFaceRecognize)
	if !errors.Is(err, ErrNoActiveModel) {
		t.Errorf("Active(unfilled slot) error = %v, want ErrNoActiveModel", err)
	}
}

// TestScanRegistersCustomModels pins the scan's registration
// semantics: every directory carrying manifest.json becomes a
// registered custom model with the manifest's facts (slot, dimension,
// params, digest), directories without a manifest are skipped, no
// model is activated by scanning alone, and a re-scan converges (the
// upsert refreshes without duplicating).
func TestScanRegistersCustomModels(t *testing.T) {
	registry, pool, modelsDir := newRegistryEnv(t,
		fixtureModel{id: "face-a", slot: "face-recognize", file: "const128.onnx", sha: const128FixtureSHA256, dim: 128},
		fixtureModel{id: "clip-a", slot: "clip-embed", file: "const512.onnx", sha: const512FixtureSHA256, dim: 512},
	)

	// Neither a bare file nor a directory without a manifest is a
	// model; the scan walks past both.
	if err := os.WriteFile(filepath.Join(modelsDir, "stray.txt"), []byte("not a model"), 0o644); err != nil {
		t.Fatalf("write stray file: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(modelsDir, "no-manifest"), 0o755); err != nil {
		t.Fatalf("create manifest-less dir: %v", err)
	}

	if _, err := registry.Scan(modelsDir); err != nil {
		t.Fatalf("Scan(modelsDir) error = %v, want nil", err)
	}

	models, err := registry.List()
	if err != nil {
		t.Fatalf("List() error = %v, want nil", err)
	}
	if len(models) != 2 {
		t.Fatalf("List() = %d models, want 2 (%+v)", len(models), models)
	}
	byID := map[string]Model{}
	for _, m := range models {
		byID[m.ID] = m
	}

	face := byID["face-a"]
	if face.Slot != SlotFaceRecognize {
		t.Errorf("face-a slot = %q, want %q", face.Slot, SlotFaceRecognize)
	}
	if face.Source != "custom" {
		t.Errorf("face-a source = %q, want %q", face.Source, "custom")
	}
	if face.Dim != 128 {
		t.Errorf("face-a dim = %d, want 128", face.Dim)
	}
	if face.Version != "fixture" {
		t.Errorf("face-a version = %q, want %q", face.Version, "fixture")
	}
	if want := filepath.Join(modelsDir, "face-a", "const128.onnx"); face.Path != want {
		t.Errorf("face-a path = %q, want %q", face.Path, want)
	}
	if face.SHA256 != const128FixtureSHA256 {
		t.Errorf("face-a sha256 = %q, want the pinned fixture digest", face.SHA256)
	}
	if face.Width != 8 || face.Height != 8 {
		t.Errorf("face-a input size = %dx%d, want 8x8", face.Width, face.Height)
	}
	if got := face.Params["scoreThreshold"]; got != 0.6 {
		t.Errorf("face-a params[scoreThreshold] = %v, want 0.6", got)
	}

	clip := byID["clip-a"]
	if clip.Slot != SlotClipEmbed {
		t.Errorf("clip-a slot = %q, want %q", clip.Slot, SlotClipEmbed)
	}
	if clip.Dim != 512 {
		t.Errorf("clip-a dim = %d, want 512", clip.Dim)
	}

	// Scanning registers, it never activates.
	if _, err := registry.Active(SlotClipEmbed); !errors.Is(err, ErrNoActiveModel) {
		t.Errorf("Active after scan error = %v, want ErrNoActiveModel (scan must not activate)", err)
	}

	// A re-scan converges: the upsert refreshes the same rows.
	if _, err := registry.Scan(modelsDir); err != nil {
		t.Fatalf("re-scan error = %v, want nil", err)
	}
	var n int
	if err := pool.QueryRow(t.Context(), "SELECT count(*) FROM models").Scan(&n); err != nil {
		t.Fatalf("count models: %v", err)
	}
	if n != 2 {
		t.Errorf("models rows after re-scan = %d, want 2 (upsert must not duplicate)", n)
	}
}

// TestScanRejectsChecksumMismatch pins the scan's half of the dual
// verification: a directory whose weights drifted from the pinned
// digest fails the scan with ErrInvalidModel and registers nothing.
// The tampered directory sorts first, so the failure precedes the good
// model's registration.
func TestScanRejectsChecksumMismatch(t *testing.T) {
	registry, pool, modelsDir := newRegistryEnv(t,
		fixtureModel{id: "aaa-tampered", slot: "clip-embed", file: "const512.onnx", sha: const512FixtureSHA256, dim: 512},
		fixtureModel{id: "zzz-good", slot: "face-recognize", file: "const128.onnx", sha: const128FixtureSHA256, dim: 128},
	)
	weights := filepath.Join(modelsDir, "aaa-tampered", "const512.onnx")
	data, err := os.ReadFile(weights)
	if err != nil {
		t.Fatalf("read tampered weights: %v", err)
	}
	data[len(data)/2] ^= 0xff
	if err := os.WriteFile(weights, data, 0o644); err != nil {
		t.Fatalf("write tampered weights: %v", err)
	}

	if _, err := registry.Scan(modelsDir); !errors.Is(err, ErrInvalidModel) {
		t.Errorf("Scan with drifted weights error = %v, want ErrInvalidModel", err)
	}
	var n int
	if err := pool.QueryRow(t.Context(), "SELECT count(*) FROM models").Scan(&n); err != nil {
		t.Fatalf("count models: %v", err)
	}
	if n != 0 {
		t.Errorf("models rows = %d, want 0 (nothing registered from a failed scan)", n)
	}
}

// TestActivateSwitchesActiveModel pins the activation contract on a
// non-clip slot: activating a model turns the slot's previous active
// row off, both rows survive, and Active resolves the new one. A
// corrupted manifest under the active model answers ErrInvalidModel
// instead of a loadable model.
func TestActivateSwitchesActiveModel(t *testing.T) {
	registry, pool, modelsDir := newRegistryEnv(t,
		fixtureModel{id: "face-a", slot: "face-recognize", file: "const128.onnx", sha: const128FixtureSHA256, dim: 128},
		fixtureModel{id: "face-b", slot: "face-recognize", file: "const128.onnx", sha: const128FixtureSHA256, dim: 128},
	)
	if _, err := registry.Scan(modelsDir); err != nil {
		t.Fatalf("Scan(modelsDir) error = %v, want nil", err)
	}

	if _, err := registry.Activate(SlotFaceRecognize, "face-a"); err != nil {
		t.Fatalf("Activate(face-a) error = %v, want nil", err)
	}
	active, err := registry.Active(SlotFaceRecognize)
	if err != nil {
		t.Fatalf("Active() error = %v, want nil", err)
	}
	if active.ID != "face-a" {
		t.Errorf("Active() id = %q, want face-a", active.ID)
	}

	if _, err := registry.Activate(SlotFaceRecognize, "face-b"); err != nil {
		t.Fatalf("Activate(face-b) error = %v, want nil", err)
	}
	active, err = registry.Active(SlotFaceRecognize)
	if err != nil {
		t.Fatalf("Active() error = %v, want nil", err)
	}
	if active.ID != "face-b" {
		t.Errorf("Active() id = %q, want face-b", active.ID)
	}

	// Both rows survive the switch; exactly one is active.
	if !modelActiveFlag(t, pool, "face-b") {
		t.Errorf("face-b is_active = false, want true")
	}
	if modelActiveFlag(t, pool, "face-a") {
		t.Errorf("face-a is_active = true, want false (the switch must deactivate the old row)")
	}
	models, err := registry.List()
	if err != nil {
		t.Fatalf("List() error = %v, want nil", err)
	}
	if len(models) != 2 {
		t.Errorf("List() = %d models, want 2 (old rows must survive a switch)", len(models))
	}

	// A model whose manifest disappeared is not loadable: Active
	// answers ErrInvalidModel rather than handing out an unverified
	// session.
	if err := os.Remove(filepath.Join(modelsDir, "face-b", "manifest.json")); err != nil {
		t.Fatalf("remove manifest: %v", err)
	}
	if _, err := registry.Active(SlotFaceRecognize); !errors.Is(err, ErrInvalidModel) {
		t.Errorf("Active without manifest error = %v, want ErrInvalidModel", err)
	}
}

// TestActivateRejectsUnknownModel pins the not-found error: activating
// an id no row carries answers ErrModelNotFound.
func TestActivateRejectsUnknownModel(t *testing.T) {
	registry, _, _ := newRegistryEnv(t)

	if _, err := registry.Activate(SlotFaceRecognize, "no-such-model"); !errors.Is(err, ErrModelNotFound) {
		t.Errorf("Activate(unknown id) error = %v, want ErrModelNotFound", err)
	}
}

// TestActivateRejectsSlotMismatch pins that the slot argument selects
// the row: a model of another slot cannot be smuggled into a slot by
// id.
func TestActivateRejectsSlotMismatch(t *testing.T) {
	registry, _, modelsDir := newRegistryEnv(t,
		fixtureModel{id: "face-a", slot: "face-recognize", file: "const128.onnx", sha: const128FixtureSHA256, dim: 128},
	)
	if _, err := registry.Scan(modelsDir); err != nil {
		t.Fatalf("Scan(modelsDir) error = %v, want nil", err)
	}

	if _, err := registry.Activate(SlotClipEmbed, "face-a"); !errors.Is(err, ErrInvalidModel) {
		t.Errorf("Activate(slot mismatch) error = %v, want ErrInvalidModel", err)
	}
}

// TestActivateRejectsMissingFile pins the file gate of activation: a
// registered row whose weights are gone answers ErrInvalidModel --
// activation never turns a model without loadable files into the
// active one.
func TestActivateRejectsMissingFile(t *testing.T) {
	registry, pool, _ := newRegistryEnv(t)

	queries := dbgen.New(pool)
	if err := queries.InsertModel(t.Context(), dbgen.InsertModelParams{
		ID:        "ghost",
		Slot:      dbgen.ModelSlotFaceRecognize,
		Version:   "ghost",
		Source:    dbgen.ModelSourceCustom,
		FilePath:  "ghost/model.onnx",
		InputSize: pgtype.Int4{Int32: 8, Valid: true},
		Dim:       pgtype.Int4{Int32: 128, Valid: true},
		Params:    []byte("{}"),
	}); err != nil {
		t.Fatalf("InsertModel(ghost) error = %v, want nil", err)
	}

	if _, err := registry.Activate(SlotFaceRecognize, "ghost"); !errors.Is(err, ErrInvalidModel) {
		t.Errorf("Activate(missing file) error = %v, want ErrInvalidModel", err)
	}
}

// TestSameDimensionActivationCoexists pins the same-dimension half of
// the dimension policy (review focus: switching keeps products):
// activating a second clip model of the column's current dimension
// flips the active flag and touches nothing else -- the embedding rows
// survive and the column dimension stays as it was.
func TestSameDimensionActivationCoexists(t *testing.T) {
	registry, pool, modelsDir := newRegistryEnv(t,
		fixtureModel{id: "clip-512a", slot: "clip-embed", file: "const512.onnx", sha: const512FixtureSHA256, dim: 512},
		fixtureModel{id: "clip-512b", slot: "clip-embed", file: "const512.onnx", sha: const512FixtureSHA256, dim: 512},
	)
	if _, err := registry.Scan(modelsDir); err != nil {
		t.Fatalf("Scan(modelsDir) error = %v, want nil", err)
	}
	if _, err := registry.Activate(SlotClipEmbed, "clip-512a"); err != nil {
		t.Fatalf("Activate(clip-512a) error = %v, want nil", err)
	}
	seedClipEmbedding(t, pool, "clip-512a", 512)

	if _, err := registry.Activate(SlotClipEmbed, "clip-512b"); err != nil {
		t.Fatalf("Activate(clip-512b) error = %v, want nil", err)
	}

	if got := countEmbeddings(t, pool); got != 1 {
		t.Errorf("clip_embeddings rows = %d, want 1 (same-dimension switch must not wipe)", got)
	}
	if got := columnDim(t, pool); got != 512 {
		t.Errorf("embedding column dim = %d, want 512 (unchanged)", got)
	}
	if !modelActiveFlag(t, pool, "clip-512b") {
		t.Errorf("clip-512b is_active = false, want true")
	}
	if modelActiveFlag(t, pool, "clip-512a") {
		t.Errorf("clip-512a is_active = true, want false")
	}
	active, err := registry.Active(SlotClipEmbed)
	if err != nil {
		t.Fatalf("Active() error = %v, want nil", err)
	}
	if active.ID != "clip-512b" || active.Dim != 512 {
		t.Errorf("Active() = %q (dim %d), want clip-512b (dim 512)", active.ID, active.Dim)
	}
}

// TestCrossDimensionActivationWipesAndAlters pins the cross-dimension
// half of the dimension policy: activating a clip model whose
// dimension differs from the column's current one clears
// clip_embeddings, ALTERs the column to the new dimension, and leaves
// a valid HNSW index behind on the emptied table -- all in the one
// activation transaction. The old model's row survives, deactivated.
func TestCrossDimensionActivationWipesAndAlters(t *testing.T) {
	registry, pool, modelsDir := newRegistryEnv(t,
		fixtureModel{id: "clip-512a", slot: "clip-embed", file: "const512.onnx", sha: const512FixtureSHA256, dim: 512},
		fixtureModel{id: "clip-768", slot: "clip-embed", file: "const768.onnx", sha: const768FixtureSHA256, dim: 768},
	)
	if _, err := registry.Scan(modelsDir); err != nil {
		t.Fatalf("Scan(modelsDir) error = %v, want nil", err)
	}
	if _, err := registry.Activate(SlotClipEmbed, "clip-512a"); err != nil {
		t.Fatalf("Activate(clip-512a) error = %v, want nil", err)
	}
	seedClipEmbedding(t, pool, "clip-512a", 512)
	seedClipEmbedding(t, pool, "clip-512a", 512)

	if _, err := registry.Activate(SlotClipEmbed, "clip-768"); err != nil {
		t.Fatalf("Activate(clip-768) error = %v, want nil", err)
	}

	if got := countEmbeddings(t, pool); got != 0 {
		t.Errorf("clip_embeddings rows = %d, want 0 (cross-dimension switch must wipe)", got)
	}
	if got := columnDim(t, pool); got != 768 {
		t.Errorf("embedding column dim = %d, want 768", got)
	}
	if !hnswIndexValid(t, pool) {
		t.Errorf("hnsw index invalid after the dimension switch, want a valid rebuilt index")
	}
	active, err := registry.Active(SlotClipEmbed)
	if err != nil {
		t.Fatalf("Active() error = %v, want nil", err)
	}
	if active.ID != "clip-768" || active.Dim != 768 {
		t.Errorf("Active() = %q (dim %d), want clip-768 (dim 768)", active.ID, active.Dim)
	}
	if !modelActiveFlag(t, pool, "clip-768") {
		t.Errorf("clip-768 is_active = false, want true")
	}
	if modelActiveFlag(t, pool, "clip-512a") {
		t.Errorf("clip-512a is_active = true, want false")
	}
	// The old model's row survives the switch, deactivated: switching
	// back is a flip, not a re-download.
	models, err := registry.List()
	if err != nil {
		t.Fatalf("List() error = %v, want nil", err)
	}
	if len(models) != 2 {
		t.Errorf("List() = %d models, want 2 (old rows must survive a switch)", len(models))
	}
}

// TestCrossDimensionActivationRollsBack pins the atomicity of the
// dimension switch (review focus: half-done switches must roll back):
// when the ALTER fails after the wipe, the transaction rolls the
// embedding rows, the column dimension, the index and the activation
// flags back to their original state.
func TestCrossDimensionActivationRollsBack(t *testing.T) {
	registry, pool, modelsDir := newRegistryEnv(t,
		fixtureModel{id: "clip-512a", slot: "clip-embed", file: "const512.onnx", sha: const512FixtureSHA256, dim: 512},
		fixtureModel{id: "clip-768", slot: "clip-embed", file: "const768.onnx", sha: const768FixtureSHA256, dim: 768},
	)
	if _, err := registry.Scan(modelsDir); err != nil {
		t.Fatalf("Scan(modelsDir) error = %v, want nil", err)
	}
	if _, err := registry.Activate(SlotClipEmbed, "clip-512a"); err != nil {
		t.Fatalf("Activate(clip-512a) error = %v, want nil", err)
	}
	seedClipEmbedding(t, pool, "clip-512a", 512)
	seedClipEmbedding(t, pool, "clip-512a", 512)

	// Inject the mid-transaction ALTER failure: the wipe has already
	// run inside the transaction when this fires.
	registry.alterEmbeddingDim = func(ctx context.Context, tx pgx.Tx, dim int) error {
		return fmt.Errorf("injected ALTER failure for vector(%d)", dim)
	}

	if _, err := registry.Activate(SlotClipEmbed, "clip-768"); err == nil {
		t.Fatal("Activate with injected ALTER failure returned nil, want an error")
	}

	if got := countEmbeddings(t, pool); got != 2 {
		t.Errorf("clip_embeddings rows = %d, want 2 (rollback must restore the wiped rows)", got)
	}
	if got := columnDim(t, pool); got != 512 {
		t.Errorf("embedding column dim = %d, want 512 (rollback must restore the column)", got)
	}
	if !hnswIndexValid(t, pool) {
		t.Errorf("hnsw index invalid after the rollback, want the original valid index")
	}
	if !modelActiveFlag(t, pool, "clip-512a") {
		t.Errorf("clip-512a is_active = false, want true (rollback must restore activation)")
	}
	if modelActiveFlag(t, pool, "clip-768") {
		t.Errorf("clip-768 is_active = true, want false (rollback must restore activation)")
	}
	active, err := registry.Active(SlotClipEmbed)
	if err != nil {
		t.Fatalf("Active() error = %v, want nil", err)
	}
	if active.ID != "clip-512a" {
		t.Errorf("Active() id = %q, want clip-512a (still the active model)", active.ID)
	}
}
