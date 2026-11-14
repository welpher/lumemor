// The admin model management and job panel endpoint tests :
// the admin-role guard of the whole surface, the listing's mapping
// (registry rows plus the activation flags, the files-present probe
// and the download run summary), the activation switch's two
// semantics -- same-dimension switches keep every product row of the
// old model in place (the coexist contract: rows stay, active-model
// filtered queries stop seeing them, switching back revives them),
// cross-dimension switches wipe the embeddings and re-dimension the
// column in one transaction and report both facts, and a failed ALTER
// rolls the whole switch back -- the model error mapping, the rerun
// trigger's 202/200 idempotence, the custom model scan, and the job
// run listing's ordering and limit.
package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pgvector/pgvector-go"

	apigen "github.com/welpher/lumemor/server/internal/api/gen"
	dbgen "github.com/welpher/lumemor/server/internal/db/gen"
	"github.com/welpher/lumemor/server/internal/ml"
)

// Pinned digests of the committed ml fixtures these tests copy into
// their throwaway model directories (the same files the registry
// tests pin).
const (
	adminConst128SHA256 = "e5d4940db6bddc912dee319434394700cf82fd8950cc32e609435ae7f9597263"
	adminConst512SHA256 = "92d6b93b89bf6ccec24222be86689b6ff18370503d3d71316d5d325bd2116766"
	adminConst768SHA256 = "0081d3e576a85d76090f2a59fb7e7ff3a6623d989cdac4474e8ed7ead85a9304"
)

// adminFixtureModel describes one fixture model these tests lay out on
// disk exactly the way the download service and the directory scan lay
// out real ones.
type adminFixtureModel struct {
	id      string
	slot    string // the manifest's hyphenated slot value
	file    string // the file under ../ml/testdata
	sha     string
	dim     int
	license string
	notice  string // the manifest's provenance notice line
}

// writeAdminFixtureModel lays out one model directory under modelsDir:
// manifest.json plus the weights, pinned by digest.
func writeAdminFixtureModel(t *testing.T, modelsDir string, fm adminFixtureModel) {
	t.Helper()

	dir := filepath.Join(modelsDir, fm.id)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("create model dir %s: %v", dir, err)
	}
	data, err := os.ReadFile(filepath.Join("..", "ml", "testdata", fm.file))
	if err != nil {
		t.Fatalf("read fixture %s: %v", fm.file, err)
	}
	if err := os.WriteFile(filepath.Join(dir, fm.file), data, 0o644); err != nil {
		t.Fatalf("write model weights: %v", err)
	}

	manifest := ml.ManifestModel{
		ID:        fm.id,
		Slot:      fm.slot,
		Version:   "fixture",
		License:   fm.license,
		Notice:    fm.notice,
		InputSize: ml.ManifestSize{Width: 8, Height: 8},
		Dim:       fm.dim,
		Params:    map[string]any{"scoreThreshold": 0.6},
		Files:     []ml.ManifestFile{{Name: fm.file, SHA256: fm.sha, Bytes: int64(len(data))}},
	}
	blob, err := json.Marshal(manifest)
	if err != nil {
		t.Fatalf("marshal fixture manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), blob, 0o644); err != nil {
		t.Fatalf("write fixture manifest: %v", err)
	}
}

// The fixture vocabulary the activation tests share: two same-dimension
// clip models, one clip model of a different dimension, a face detect
// model and a face recognize model (the faces rows reference both).
var (
	fixtureClipA = adminFixtureModel{id: "clip-a", slot: "clip-embed", file: "const512.onnx", sha: adminConst512SHA256, dim: 512, license: "MIT", notice: "Fixture provenance notice."}
	fixtureClipB = adminFixtureModel{id: "clip-b", slot: "clip-embed", file: "const512.onnx", sha: adminConst512SHA256, dim: 512, license: "Apache-2.0"}
	fixtureClip7 = adminFixtureModel{id: "clip-7", slot: "clip-embed", file: "const768.onnx", sha: adminConst768SHA256, dim: 768, license: "Apache-2.0"}
	// clip-huge carries a dimension beyond what a vector column can
	// hold: activating it fails at the ALTER, after the wipe, which is
	// exactly the mid-transaction failure the rollback test injects.
	fixtureClipHuge = adminFixtureModel{id: "clip-huge", slot: "clip-embed", file: "const768.onnx", sha: adminConst768SHA256, dim: 30000, license: "MIT"}
	fixtureDetect   = adminFixtureModel{id: "detect-fx", slot: "face-detect", file: "const128.onnx", sha: adminConst128SHA256, dim: 0, license: "MIT"}
	fixtureRec      = adminFixtureModel{id: "rec-fx", slot: "face-recognize", file: "const128.onnx", sha: adminConst128SHA256, dim: 128, license: "MIT"}
)

// stubMlReruns is the rerun enqueue seam under test: it counts the
// calls, records the slots it was called with, answers a fixed run
// identifier, and can be told to fail.
type stubMlReruns struct {
	calls int
	slots []ml.Slot
	id    uuid.UUID
	err   error
}

// EnqueueMlRerun implements MlRerunEnqueuer.
func (s *stubMlReruns) EnqueueMlRerun(ctx context.Context, slot ml.Slot) (uuid.UUID, error) {
	s.calls++
	s.slots = append(s.slots, slot)
	if s.err != nil {
		return uuid.Nil, s.err
	}
	return s.id, nil
}

// adminModelsEnv provisions the surface's test environment: a migrated
// throwaway database, a real registry over a throwaway models dir
// carrying the given fixtures, the router wired to the stub rerun
// seam, and both an administrator and a plain signed-in user.
type adminModelsEnv struct {
	router    http.Handler
	pool      *pgxpool.Pool
	registry  *ml.Registry
	modelsDir string
	stub      *stubMlReruns
	cookie    *http.Cookie
	plain     *http.Cookie
	ownerID   uuid.UUID
	// probeHosts answers the download sources list the source
	// pipeline's declared-host check reads (nil: every host declared).
	probeHosts func() []string
}

// newAdminModelsEnv provisions the environment with the given fixture
// models laid out (not yet registered: registering is the scan
// endpoint's job, which the tests exercise explicitly).
func newAdminModelsEnv(t *testing.T, fixtures ...adminFixtureModel) *adminModelsEnv {
	return newAdminModelsEnvHosts(t, nil, fixtures...)
}

// newAdminModelsEnvHosts is newAdminModelsEnv with the source
// pipeline's allowed-hosts function injected (the warn-path tests).
func newAdminModelsEnvHosts(t *testing.T, hosts func() []string, fixtures ...adminFixtureModel) *adminModelsEnv {
	t.Helper()

	pool := newSetupEnv(t)
	modelsDir := t.TempDir()
	for _, fm := range fixtures {
		writeAdminFixtureModel(t, modelsDir, fm)
	}
	registry := ml.NewRegistry(pool, dbgen.New(pool), modelsDir)
	stub := &stubMlReruns{id: uuid.MustParse("0192f8b1-4c5d-8e9f-a0b1-2c3d4e5f6a70")}
	router := NewRouter(&Deps{
		Pool:       pool,
		Version:    testVersion,
		Models:     registry,
		ModelsDir:  modelsDir,
		MlReruns:   stub,
		ProbeHosts: hosts,
	})

	rec := postAdmin(t, router, createAdminBody(t, "ada@example.com", testPassword, nil))
	if rec.Code != http.StatusCreated {
		t.Fatalf("create admin: status = %d, want %d (body %q)", rec.Code, http.StatusCreated, rec.Body.String())
	}
	cookie, _ := loginCookie(t, router, "ada@example.com", testPassword, "admin-models-tests")
	_, plain := seedPlainUser(t, pool, "grace@example.com")

	var adminID uuid.UUID
	if err := pool.QueryRow(context.Background(),
		"SELECT id FROM users WHERE email = $1", "ada@example.com").Scan(&adminID); err != nil {
		t.Fatalf("resolve admin id: %v", err)
	}

	return &adminModelsEnv{
		router:    router,
		pool:      pool,
		registry:  registry,
		modelsDir: modelsDir,
		stub:      stub,
		cookie:    cookie,
		plain:     plain,
		ownerID:   adminID,
	}
}

// do fires one request with the given session cookie (nil = no
// session).
func (e *adminModelsEnv) do(t *testing.T, method, target string, cookie *http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, target, nil)
	if cookie != nil {
		req.AddCookie(cookie)
	}
	rec := httptest.NewRecorder()
	e.router.ServeHTTP(rec, req)
	return rec
}

// doJSON fires one request with a JSON body and the given session
// cookie.
func (e *adminModelsEnv) doJSON(t *testing.T, method, target, body string, cookie *http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if cookie != nil {
		req.AddCookie(cookie)
	}
	rec := httptest.NewRecorder()
	e.router.ServeHTTP(rec, req)
	return rec
}

// scanModels registers every fixture through the scan endpoint.
func (e *adminModelsEnv) scanModels(t *testing.T) {
	t.Helper()
	rec := e.do(t, http.MethodPost, "/api/v1/admin/models/scan", e.cookie)
	if rec.Code != http.StatusOK {
		t.Fatalf("scan models: status = %d, want %d (body %q)", rec.Code, http.StatusOK, rec.Body.String())
	}
}

// activateModel flips the model active through the PATCH endpoint.
func (e *adminModelsEnv) activateModel(t *testing.T, id string) apigen.ModelActivateResponse {
	t.Helper()
	rec := e.doJSON(t, http.MethodPatch, "/api/v1/admin/models/"+id, "{}", e.cookie)
	if rec.Code != http.StatusOK {
		t.Fatalf("activate %s: status = %d, want %d (body %q)", id, rec.Code, http.StatusOK, rec.Body.String())
	}
	var body apigen.ModelActivateResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode activation response %q: %v", rec.Body.String(), err)
	}
	return body
}

// insertAssetRow inserts one image asset row owned by the admin and
// returns its id.
func (e *adminModelsEnv) insertAssetRow(t *testing.T) uuid.UUID {
	t.Helper()
	id, err := uuid.NewV7()
	if err != nil {
		t.Fatalf("new asset id: %v", err)
	}
	if _, err := e.queries().CreateAsset(context.Background(), dbgen.CreateAssetParams{
		ID:               pgUUID(id),
		OwnerID:          pgUUID(e.ownerID),
		FilePath:         "originals/models-test/" + id.String() + ".jpg",
		Checksum:         "admin-models-" + id.String(),
		OriginalFilename: id.String() + ".jpg",
		FileSize:         8,
		MimeType:         "image/jpeg",
		MediaKind:        dbgen.MediaKindImage,
		TakenAt:          pgtype.Timestamptz{Time: time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC), Valid: true},
		PendingJobs:      0,
	}); err != nil {
		t.Fatalf("insert asset: %v", err)
	}
	return id
}

// queries builds a query handle over the pool.
func (e *adminModelsEnv) queries() *dbgen.Queries {
	return dbgen.New(e.pool)
}

// seedClipEmbedding inserts one clip embedding row of the given
// dimension for the model.
func (e *adminModelsEnv) seedClipEmbedding(t *testing.T, assetID uuid.UUID, modelID string, dim int) {
	t.Helper()
	values := make([]float32, dim)
	for i := range values {
		values[i] = float32(i)
	}
	if _, err := e.pool.Exec(context.Background(),
		"INSERT INTO clip_embeddings (asset_id, model_id, embedding) VALUES ($1, $2, $3)",
		pgUUID(assetID), modelID, pgvector.NewVector(values)); err != nil {
		t.Fatalf("seed clip embedding: %v", err)
	}
}

// seedFaceRow inserts one face row carrying a 128-dimension embedding
// of the recognize model (the product rows the coexist contract must
// leave untouched by a clip switch).
func (e *adminModelsEnv) seedFaceRow(t *testing.T, assetID uuid.UUID) {
	t.Helper()
	id, err := uuid.NewV7()
	if err != nil {
		t.Fatalf("new face id: %v", err)
	}
	embedding := make([]float32, 128)
	for i := range embedding {
		embedding[i] = float32(i) / 128
	}
	if _, err := e.pool.Exec(context.Background(), `
		INSERT INTO faces (id, asset_id, bbox_x, bbox_y, bbox_w, bbox_h, score, model_id, embedding, embed_model_id)
		VALUES ($1, $2, 100, 150, 200, 250, 0.9, 'detect-fx', $3, 'rec-fx')`,
		pgUUID(id), pgUUID(assetID), pgvector.NewVector(embedding)); err != nil {
		t.Fatalf("seed face row: %v", err)
	}
}

// countRows counts the rows the SQL leaves.
func countRows(t *testing.T, pool *pgxpool.Pool, query string, args ...any) int {
	t.Helper()
	var count int
	if err := pool.QueryRow(context.Background(), query, args...).Scan(&count); err != nil {
		t.Fatalf("count rows (%s): %v", query, err)
	}
	return count
}

// embeddingColumnDim reads the clip embeddings column's current
// dimension from the database catalog.
func embeddingColumnDim(t *testing.T, pool *pgxpool.Pool) int {
	t.Helper()
	var dim int
	if err := pool.QueryRow(context.Background(), `
		SELECT atttypmod FROM pg_attribute
		WHERE attrelid = 'clip_embeddings'::regclass AND attname = 'embedding'`).Scan(&dim); err != nil {
		t.Fatalf("read embedding column dim: %v", err)
	}
	return dim
}

// modelActiveFlag reads one model's is_active flag.
func modelActiveFlag(t *testing.T, pool *pgxpool.Pool, id string) bool {
	t.Helper()
	var active bool
	if err := pool.QueryRow(context.Background(),
		"SELECT is_active FROM models WHERE id = $1", id).Scan(&active); err != nil {
		t.Fatalf("read is_active of %s: %v", id, err)
	}
	return active
}

// insertJobRunKindRow inserts one job_runs row directly with the given
// kind, state, counts and creation time.
func insertJobRunKindRow(t *testing.T, pool *pgxpool.Pool, id uuid.UUID, kind, state string, total, done, failed int, createdAt time.Time) {
	t.Helper()
	_, err := pool.Exec(context.Background(),
		"INSERT INTO job_runs (id, kind, total, done, failed, state, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $7)",
		pgUUID(id), kind, total, done, failed, state, createdAt.Truncate(time.Microsecond))
	if err != nil {
		t.Fatalf("insert the job_runs row: %v", err)
	}
}

// TestAdminModelSurfaceGuardsTheAdminRole pins the permission model of
// the whole surface: a signed-in plain user answers 403
// AUTH_FORBIDDEN on every route, and a request without a session 401,
// both before the rerun seam is ever called.
func TestAdminModelSurfaceGuardsTheAdminRole(t *testing.T) {
	e := newAdminModelsEnv(t, fixtureClipA)

	routes := []struct {
		method string
		target string
	}{
		{http.MethodGet, "/api/v1/admin/models"},
		{http.MethodPatch, "/api/v1/admin/models/clip-a"},
		{http.MethodPost, "/api/v1/admin/models/clip-a/rerun"},
		{http.MethodPost, "/api/v1/admin/models/scan"},
		{http.MethodGet, "/api/v1/admin/jobs"},
	}
	for _, route := range routes {
		rec := e.do(t, route.method, route.target, e.plain)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("plain user %s %s status = %d, want %d (body %q)",
				route.method, route.target, rec.Code, http.StatusForbidden, rec.Body.String())
		}
		apiErr, _ := decodeError(t, rec)
		if apiErr.Code != apigen.AUTHFORBIDDEN {
			t.Errorf("plain user %s %s code = %q, want %q",
				route.method, route.target, apiErr.Code, apigen.AUTHFORBIDDEN)
		}
	}

	rec := e.do(t, http.MethodGet, "/api/v1/admin/models", nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("sessionless GET status = %d, want %d (body %q)", rec.Code, http.StatusUnauthorized, rec.Body.String())
	}
	apiErr, _ := decodeError(t, rec)
	if apiErr.Code != apigen.AUTHINVALIDCREDENTIALS {
		t.Errorf("sessionless code = %q, want %q", apiErr.Code, apigen.AUTHINVALIDCREDENTIALS)
	}

	if e.stub.calls != 0 {
		t.Errorf("the rerun seam was called %d times, want 0 (the guard rejects first)", e.stub.calls)
	}
}

// TestListModelsMapsTheRegistryRows pins the listing's mapping: every
// registered model with its facts (the registry's row plus the
// manifest's license and input geometry), the activation flag from the
// table, the files-present probe (a model whose weights went missing
// still lists, with filesPresent false), and the most recent download
// run as the summary (null when none exists).
func TestListModelsMapsTheRegistryRows(t *testing.T) {
	e := newAdminModelsEnv(t, fixtureClipA, fixtureClipB, fixtureDetect)
	e.scanModels(t)

	rec := e.do(t, http.MethodGet, "/api/v1/admin/models", e.cookie)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET status = %d, want %d (body %q)", rec.Code, http.StatusOK, rec.Body.String())
	}
	var body apigen.ModelListResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode the response %q: %v", rec.Body.String(), err)
	}

	if body.Download != nil {
		t.Errorf("download = %v, want null before any download run", body.Download)
	}

	// No model is active yet: the scan registers without activating.
	if len(body.Models) != 3 {
		t.Fatalf("models = %d, want 3 (body %q)", len(body.Models), rec.Body.String())
	}
	// The listing groups by slot then id, the slot order being the
	// model_slot enum's declaration order (face_detect before
	// clip_embed).
	detect, clipA, clipB := body.Models[0], body.Models[1], body.Models[2]
	if clipA.Id != "clip-a" || clipA.Slot != apigen.ClipEmbed {
		t.Errorf("models[1] = %q slot %q, want clip-a of clip_embed", clipA.Id, clipA.Slot)
	}
	if detect.Id != "detect-fx" || detect.Slot != apigen.FaceDetect {
		t.Errorf("models[0] = %q slot %q, want detect-fx of face_detect", detect.Id, detect.Slot)
	}
	if clipA.Version != "fixture" || clipA.Source != apigen.ModelSourceCustom {
		t.Errorf("clip-a version/source = %q/%q, want fixture/custom", clipA.Version, clipA.Source)
	}
	if clipA.License != "MIT" {
		t.Errorf("clip-a license = %q, want MIT (the manifest's license)", clipA.License)
	}
	if clipA.Dim == nil || *clipA.Dim != 512 {
		t.Errorf("clip-a dim = %v, want 512", clipA.Dim)
	}
	if clipA.InputSize == nil || clipA.InputSize.Width != 8 || clipA.InputSize.Height != 8 {
		t.Errorf("clip-a inputSize = %v, want 8x8", clipA.InputSize)
	}
	if clipA.IsActive {
		t.Error("clip-a isActive = true, want false (the scan never activates)")
	}
	if !clipA.FilesPresent {
		t.Error("clip-a filesPresent = false, want true")
	}
	if clipA.Params == nil || clipA.Params["scoreThreshold"] != 0.6 {
		t.Errorf("clip-a params = %v, want the manifest's scoreThreshold 0.6", clipA.Params)
	}
	// The provenance notice: the manifest's notice line, null when the
	// manifest is unreadable (the fact is unknowable).
	if clipA.Notice == nil || *clipA.Notice != "Fixture provenance notice." {
		t.Errorf("clip-a notice = %v, want the manifest's provenance notice", clipA.Notice)
	}
	// A detector carries no embedding dimension.
	if detect.Dim != nil {
		t.Errorf("detect-fx dim = %v, want null (detectors do not embed)", detect.Dim)
	}

	// One model's weights go missing: the row still lists (the admin
	// needs to see it), with filesPresent false.
	if err := os.Remove(filepath.Join(e.modelsDir, "clip-b", "const512.onnx")); err != nil {
		t.Fatalf("remove clip-b weights: %v", err)
	}
	body = decodeModelList(t, e.do(t, http.MethodGet, "/api/v1/admin/models", e.cookie))
	clipB = modelOf(t, body, "clip-b")
	if clipB.FilesPresent {
		t.Error("clip-b filesPresent = true after the weights went missing, want false")
	}

	// The manifest going missing takes the unknowable facts with it:
	// notice turns null (license empty, geometry null) while the row
	// and its database facts still list.
	if err := os.Remove(filepath.Join(e.modelsDir, "clip-b", "manifest.json")); err != nil {
		t.Fatalf("remove clip-b manifest: %v", err)
	}
	body = decodeModelList(t, e.do(t, http.MethodGet, "/api/v1/admin/models", e.cookie))
	clipB = modelOf(t, body, "clip-b")
	if clipB.Notice != nil {
		t.Errorf("clip-b notice = %v, want null (the manifest is unreadable)", *clipB.Notice)
	}
	if clipB.License != "" {
		t.Errorf("clip-b license = %q, want empty (the manifest is unreadable)", clipB.License)
	}

	// The download summary: the most recent model_download row.
	download := uuid.MustParse("0192f8c2-5d6e-8f9a-b0c1-2d3d4e5f6b81")
	insertJobRunRow(t, e.pool, download, "succeeded", time.Now().Add(-time.Minute))
	body = decodeModelList(t, e.do(t, http.MethodGet, "/api/v1/admin/models", e.cookie))
	if body.Download == nil {
		t.Fatal("download = null, want the most recent model_download row")
	}
	if body.Download.Id != download || body.Download.Kind != apigen.ModelDownload {
		t.Errorf("download = %v kind %q, want %s of kind model_download", body.Download.Id, body.Download.Kind, download)
	}
	if body.Download.State != apigen.JobRunStateSucceeded || body.Download.Total != 1000 {
		t.Errorf("download state/total = %q/%d, want succeeded/1000", body.Download.State, body.Download.Total)
	}
}

// TestPatchModelSameDimensionKeepsProducts pins the coexist contract
// (the pinned payload): a same-dimension switch flips the
// activation flags and deletes NOTHING -- the old model's embedding
// rows and the faces' embedding rows stay in the table, the
// active-model filtered query stops seeing the old rows, and
// switching back revives them with zero recompute.
func TestPatchModelSameDimensionKeepsProducts(t *testing.T) {
	e := newAdminModelsEnv(t, fixtureClipA, fixtureClipB, fixtureDetect, fixtureRec)
	e.scanModels(t)

	activated := e.activateModel(t, "clip-a")
	if activated.Activated.Id != "clip-a" || activated.Activated.Slot != apigen.ClipEmbed {
		t.Fatalf("activated = %q slot %q, want clip-a of clip_embed", activated.Activated.Id, activated.Activated.Slot)
	}
	if activated.DimensionChanged != nil || activated.WipedEmbeddings != nil {
		t.Errorf("same-dimension activation reported dimensionChanged=%v wipedEmbeddings=%v, want neither",
			activated.DimensionChanged, activated.WipedEmbeddings)
	}

	assetA, assetB := e.insertAssetRow(t), e.insertAssetRow(t)
	e.seedClipEmbedding(t, assetA, "clip-a", 512)
	e.seedClipEmbedding(t, assetB, "clip-a", 512)
	e.seedFaceRow(t, assetA)

	activated = e.activateModel(t, "clip-b")
	if activated.DimensionChanged != nil || activated.WipedEmbeddings != nil {
		t.Errorf("same-dimension activation reported dimensionChanged=%v wipedEmbeddings=%v, want neither",
			activated.DimensionChanged, activated.WipedEmbeddings)
	}

	// Nothing was deleted: the old model's rows and the faces rows all
	// survive the switch.
	if got := countRows(t, e.pool, "SELECT count(*) FROM clip_embeddings"); got != 2 {
		t.Errorf("clip_embeddings rows = %d, want 2 (a same-dimension switch deletes nothing)", got)
	}
	if got := countRows(t, e.pool, "SELECT count(*) FROM faces"); got != 1 {
		t.Errorf("faces rows = %d, want 1 (a clip switch never touches faces)", got)
	}
	if !modelActiveFlag(t, e.pool, "clip-b") || modelActiveFlag(t, e.pool, "clip-a") {
		t.Error("the activation flags did not flip to clip-b")
	}
	// The active-model filtered query sees none of the old rows.
	if got := countRows(t, e.pool, "SELECT count(*) FROM clip_embeddings WHERE model_id = 'clip-b'"); got != 0 {
		t.Errorf("clip-b embeddings = %d, want 0 (the old rows are invisible to the active model)", got)
	}

	// Switching back revives the old rows: zero recompute, zero
	// re-download.
	e.activateModel(t, "clip-a")
	if got := countRows(t, e.pool, "SELECT count(*) FROM clip_embeddings WHERE model_id = 'clip-a'"); got != 2 {
		t.Errorf("clip-a embeddings after switching back = %d, want 2 (the old rows revive)", got)
	}
	if got := countRows(t, e.pool, "SELECT count(*) FROM clip_embeddings"); got != 2 {
		t.Errorf("clip_embeddings rows after switching back = %d, want 2 (no duplication)", got)
	}
}

// TestPatchModelCrossDimensionWipesAndRedimensions pins the
// cross-dimension switch: the embeddings are wiped, the column is
// re-typed to the new dimension, the response reports both facts, the
// old rows are gone, and a vector of the old dimension is rejected by
// the column afterwards.
func TestPatchModelCrossDimensionWipesAndRedimensions(t *testing.T) {
	e := newAdminModelsEnv(t, fixtureClipA, fixtureClip7)
	e.scanModels(t)
	e.activateModel(t, "clip-a")

	assetA, assetB := e.insertAssetRow(t), e.insertAssetRow(t)
	e.seedClipEmbedding(t, assetA, "clip-a", 512)
	e.seedClipEmbedding(t, assetB, "clip-a", 512)

	activated := e.activateModel(t, "clip-7")
	if activated.DimensionChanged == nil {
		t.Fatal("dimensionChanged = nil, want {512, 768}")
	}
	if activated.DimensionChanged.From != 512 || activated.DimensionChanged.To != 768 {
		t.Errorf("dimensionChanged = %d -> %d, want 512 -> 768",
			activated.DimensionChanged.From, activated.DimensionChanged.To)
	}
	if activated.WipedEmbeddings == nil || *activated.WipedEmbeddings != 2 {
		t.Errorf("wipedEmbeddings = %v, want 2", activated.WipedEmbeddings)
	}
	if activated.Activated.Id != "clip-7" {
		t.Errorf("activated id = %q, want clip-7", activated.Activated.Id)
	}

	if got := countRows(t, e.pool, "SELECT count(*) FROM clip_embeddings"); got != 0 {
		t.Errorf("clip_embeddings rows = %d, want 0 (the wipe ran)", got)
	}
	if got := embeddingColumnDim(t, e.pool); got != 768 {
		t.Errorf("embedding column dim = %d, want 768", got)
	}
	if !modelActiveFlag(t, e.pool, "clip-7") || modelActiveFlag(t, e.pool, "clip-a") {
		t.Error("the activation flags did not flip to clip-7")
	}

	// The old dimension's vectors no longer fit the column.
	if _, err := e.pool.Exec(context.Background(),
		"INSERT INTO clip_embeddings (asset_id, model_id, embedding) VALUES ($1, 'clip-7', $2)",
		pgUUID(assetA), pgvector.NewVector(make([]float32, 512))); err == nil {
		t.Error("a 512-dimension vector was accepted by the 768-dimension column, want rejection")
	}
}

// TestPatchModelCrossDimensionRollsBackOnAlterFailure pins the atomic
// of the cross-dimension switch (the pinned payload): a model
// whose dimension no vector column can hold fails the switch at the
// ALTER -- after the wipe already ran inside the transaction -- and
// the rollback restores the wiped rows, the column dimension and the
// activation flags exactly.
func TestPatchModelCrossDimensionRollsBackOnAlterFailure(t *testing.T) {
	e := newAdminModelsEnv(t, fixtureClipA, fixtureClipHuge)
	e.scanModels(t)
	e.activateModel(t, "clip-a")

	assetA, assetB := e.insertAssetRow(t), e.insertAssetRow(t)
	e.seedClipEmbedding(t, assetA, "clip-a", 512)
	e.seedClipEmbedding(t, assetB, "clip-a", 512)

	// clip-huge declares a dimension beyond what a vector column can
	// hold: the activation's ALTER fails mid-transaction.
	rec := e.doJSON(t, http.MethodPatch, "/api/v1/admin/models/clip-huge", "{}", e.cookie)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("PATCH clip-huge status = %d, want %d (body %q)",
			rec.Code, http.StatusInternalServerError, rec.Body.String())
	}
	apiErr, _ := decodeError(t, rec)
	if apiErr.Code != apigen.INTERNAL {
		t.Errorf("PATCH clip-huge code = %q, want %q", apiErr.Code, apigen.INTERNAL)
	}

	if got := countRows(t, e.pool, "SELECT count(*) FROM clip_embeddings"); got != 2 {
		t.Errorf("clip_embeddings rows = %d, want 2 (the rollback restored the wiped rows)", got)
	}
	if got := embeddingColumnDim(t, e.pool); got != 512 {
		t.Errorf("embedding column dim = %d, want 512 (the rollback restored the column)", got)
	}
	if !modelActiveFlag(t, e.pool, "clip-a") {
		t.Error("clip-a is_active = false, want true (the rollback restored the activation)")
	}
	if modelActiveFlag(t, e.pool, "clip-huge") {
		t.Error("clip-huge is_active = true, want false (the rollback restored the activation)")
	}
}

// TestPatchModelMapsTheModelErrors pins the error mapping: an unknown
// model identifier answers 404 MODEL_NOT_FOUND, a model whose files
// went missing answers 400 MODEL_INVALID (the registry's checksum
// gate), and neither ever flips an activation flag.
func TestPatchModelMapsTheModelErrors(t *testing.T) {
	e := newAdminModelsEnv(t, fixtureClipA, fixtureClipB)
	e.scanModels(t)

	rec := e.doJSON(t, http.MethodPatch, "/api/v1/admin/models/no-such-model", "{}", e.cookie)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("unknown model PATCH status = %d, want %d (body %q)", rec.Code, http.StatusNotFound, rec.Body.String())
	}
	apiErr, _ := decodeError(t, rec)
	if apiErr.Code != apigen.MODELNOTFOUND {
		t.Errorf("unknown model code = %q, want %q", apiErr.Code, apigen.MODELNOTFOUND)
	}

	if err := os.Remove(filepath.Join(e.modelsDir, "clip-a", "const512.onnx")); err != nil {
		t.Fatalf("remove clip-a weights: %v", err)
	}
	rec = e.doJSON(t, http.MethodPatch, "/api/v1/admin/models/clip-a", "{}", e.cookie)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("missing files PATCH status = %d, want %d (body %q)", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
	apiErr, _ = decodeError(t, rec)
	if apiErr.Code != apigen.MODELINVALID {
		t.Errorf("missing files code = %q, want %q", apiErr.Code, apigen.MODELINVALID)
	}
	if modelActiveFlag(t, e.pool, "clip-a") || modelActiveFlag(t, e.pool, "clip-b") {
		t.Error("a rejected activation flipped a flag, want none")
	}
}

// TestRerunModelStartsAndJoinsRuns pins the rerun trigger: the first
// call starts a run through the seam (202) for the model's slot; a run
// of the same kind still in flight answers 200 with its identifier and
// starts nothing; once nothing is in flight a new call starts a fresh
// run (the rerun is manual, a finished run never blocks it); an
// unknown model answers 404 and a model that is not the active one of
// its slot answers 400 MODEL_INVALID.
func TestRerunModelStartsAndJoinsRuns(t *testing.T) {
	e := newAdminModelsEnv(t, fixtureClipA, fixtureClipB)
	e.scanModels(t)
	e.activateModel(t, "clip-a")

	// A model that is registered but NOT the slot's active one: its
	// rerun would fill products no active-model query could see.
	rec := e.do(t, http.MethodPost, "/api/v1/admin/models/clip-b/rerun", e.cookie)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("inactive model rerun status = %d, want %d (body %q)", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
	apiErr, _ := decodeError(t, rec)
	if apiErr.Code != apigen.MODELINVALID {
		t.Errorf("inactive model code = %q, want %q", apiErr.Code, apigen.MODELINVALID)
	}

	rec = e.do(t, http.MethodPost, "/api/v1/admin/models/clip-a/rerun", e.cookie)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("first rerun status = %d, want %d (body %q)", rec.Code, http.StatusAccepted, rec.Body.String())
	}
	var body apigen.ModelRerunResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode the rerun response %q: %v", rec.Body.String(), err)
	}
	if body.JobRunId != e.stub.id {
		t.Errorf("first rerun jobRunId = %q, want the enqueued %q", body.JobRunId, e.stub.id)
	}
	if e.stub.calls != 1 || len(e.stub.slots) != 1 || e.stub.slots[0] != ml.SlotClipEmbed {
		t.Errorf("the seam was called %d times with slots %v, want once with clip_embed", e.stub.calls, e.stub.slots)
	}

	// An in-flight run of the same kind answers 200 as-is.
	inFlight := uuid.MustParse("0192f8d3-6e7f-9f0a-b1c2-3d4e5f6c7b92")
	insertJobRunKindRow(t, e.pool, inFlight, "clip_embed_rerun", "running", 5, 2, 0, time.Now().Add(-time.Minute))
	rec = e.do(t, http.MethodPost, "/api/v1/admin/models/clip-a/rerun", e.cookie)
	if rec.Code != http.StatusOK {
		t.Fatalf("in-flight rerun status = %d, want %d (body %q)", rec.Code, http.StatusOK, rec.Body.String())
	}
	body = apigen.ModelRerunResponse{}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode the in-flight rerun response %q: %v", rec.Body.String(), err)
	}
	if body.JobRunId != inFlight {
		t.Errorf("in-flight rerun jobRunId = %q, want the existing %q", body.JobRunId, inFlight)
	}
	if e.stub.calls != 1 {
		t.Errorf("the seam was called %d times, want 1 (the in-flight run is returned as-is)", e.stub.calls)
	}

	// Nothing in flight anymore: the rerun is manual, a finished run
	// never blocks a fresh one.
	if _, err := e.pool.Exec(context.Background(),
		"UPDATE job_runs SET state = 'succeeded', finished_at = now() WHERE id = $1", pgUUID(inFlight)); err != nil {
		t.Fatalf("finish the in-flight run: %v", err)
	}
	rec = e.do(t, http.MethodPost, "/api/v1/admin/models/clip-a/rerun", e.cookie)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("rerun after a finished run status = %d, want %d (body %q)", rec.Code, http.StatusAccepted, rec.Body.String())
	}
	if e.stub.calls != 2 {
		t.Errorf("the seam was called %d times, want 2 (a finished run never blocks a fresh one)", e.stub.calls)
	}

	rec = e.do(t, http.MethodPost, "/api/v1/admin/models/no-such-model/rerun", e.cookie)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("unknown model rerun status = %d, want %d (body %q)", rec.Code, http.StatusNotFound, rec.Body.String())
	}
	apiErr, _ = decodeError(t, rec)
	if apiErr.Code != apigen.MODELNOTFOUND {
		t.Errorf("unknown model rerun code = %q, want %q", apiErr.Code, apigen.MODELNOTFOUND)
	}
}

// TestRerunModelMapsStartFailure pins the 500 mapping when the rerun
// cannot be started: an enqueue that fails (and a surface with no
// queue wired at all) answers the generic internal error.
func TestRerunModelMapsStartFailure(t *testing.T) {
	e := newAdminModelsEnv(t, fixtureClipA)
	e.scanModels(t)
	e.activateModel(t, "clip-a")
	e.stub.err = errors.New("the queue insert failed")

	rec := e.do(t, http.MethodPost, "/api/v1/admin/models/clip-a/rerun", e.cookie)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("failing enqueue status = %d, want %d (body %q)", rec.Code, http.StatusInternalServerError, rec.Body.String())
	}
	apiErr, _ := decodeError(t, rec)
	if apiErr.Code != apigen.INTERNAL {
		t.Errorf("failing enqueue code = %q, want %q", apiErr.Code, apigen.INTERNAL)
	}

	// A surface without the rerun seam wired answers the same way.
	noSeam := NewRouter(&Deps{Pool: e.pool, Version: testVersion, Models: e.registry, ModelsDir: e.modelsDir})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/models/clip-a/rerun", nil)
	req.AddCookie(e.cookie)
	rec = httptest.NewRecorder()
	noSeam.ServeHTTP(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("unwired rerun status = %d, want %d (body %q)", rec.Code, http.StatusInternalServerError, rec.Body.String())
	}
	apiErr, _ = decodeError(t, rec)
	if apiErr.Code != apigen.INTERNAL {
		t.Errorf("unwired rerun code = %q, want %q", apiErr.Code, apigen.INTERNAL)
	}
}

// TestScanModelsRegistersWithoutActivating pins the scan: every model
// directory under the models dir that carries a manifest is registered
// with source custom (a re-scan converges over the same rows) and
// nothing activates; a directory whose file drifted from its pinned
// digest fails the scan with 400 MODEL_INVALID and registers nothing.
func TestScanModelsRegistersWithoutActivating(t *testing.T) {
	e := newAdminModelsEnv(t, fixtureClipA, fixtureDetect)

	rec := e.do(t, http.MethodPost, "/api/v1/admin/models/scan", e.cookie)
	if rec.Code != http.StatusOK {
		t.Fatalf("scan status = %d, want %d (body %q)", rec.Code, http.StatusOK, rec.Body.String())
	}
	var body apigen.ModelScanResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode the scan response %q: %v", rec.Body.String(), err)
	}
	if len(body.Registered) != 2 {
		t.Fatalf("registered = %d models, want 2 (body %q)", len(body.Registered), rec.Body.String())
	}
	for _, model := range body.Registered {
		if model.Source != apigen.ModelSourceCustom {
			t.Errorf("registered %s source = %q, want custom", model.Id, model.Source)
		}
		if model.IsActive {
			t.Errorf("registered %s isActive = true, want false (the scan never activates)", model.Id)
		}
		if !model.FilesPresent {
			t.Errorf("registered %s filesPresent = false, want true", model.Id)
		}
	}

	// A re-scan converges: the same rows, still nothing active.
	rec = e.do(t, http.MethodPost, "/api/v1/admin/models/scan", e.cookie)
	if rec.Code != http.StatusOK {
		t.Fatalf("re-scan status = %d, want %d (body %q)", rec.Code, http.StatusOK, rec.Body.String())
	}
	if got := countRows(t, e.pool, "SELECT count(*) FROM models"); got != 2 {
		t.Errorf("models rows after the re-scan = %d, want 2 (the scan converges)", got)
	}
	if got := countRows(t, e.pool, "SELECT count(*) FROM models WHERE is_active"); got != 0 {
		t.Errorf("active models after the scan = %d, want 0", got)
	}

	// A drifted file fails the whole scan: nothing registers.
	drifted := newAdminModelsEnv(t, fixtureClipB)
	weights := filepath.Join(drifted.modelsDir, "clip-b", "const512.onnx")
	if err := os.WriteFile(weights, []byte("tampered bytes"), 0o644); err != nil {
		t.Fatalf("tamper clip-b weights: %v", err)
	}
	rec = drifted.do(t, http.MethodPost, "/api/v1/admin/models/scan", drifted.cookie)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("drifted scan status = %d, want %d (body %q)", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
	apiErr, _ := decodeError(t, rec)
	if apiErr.Code != apigen.MODELINVALID {
		t.Errorf("drifted scan code = %q, want %q", apiErr.Code, apigen.MODELINVALID)
	}
	if got := countRows(t, drifted.pool, "SELECT count(*) FROM models"); got != 0 {
		t.Errorf("models rows after the drifted scan = %d, want 0 (a failed scan registers nothing)", got)
	}
}

// TestListJobsOrdersNewestFirstAndHonorsLimit pins the job panel
// listing: runs come back newest first, the limit parameter bounds the
// page (default 50, maximum 200, values below 1 answer 400, a value
// over the maximum is clamped to it), and every row maps its full
// shape.
func TestListJobsOrdersNewestFirstAndHonorsLimit(t *testing.T) {
	e := newAdminModelsEnv(t, fixtureClipA)

	base := time.Now().Add(-10 * time.Minute).Truncate(time.Microsecond)
	ids := []uuid.UUID{
		uuid.MustParse("0192f8e4-7f8a-9f0a-b1c2-3d4e5f6c7ba3"),
		uuid.MustParse("0192f8e5-8a9b-8f0a-b1c2-3d4e5f6c8ba4"),
		uuid.MustParse("0192f8e6-9bac-9f0a-b1c2-3d4e5f6c9ba5"),
		uuid.MustParse("0192f8e7-0cbd-8f0a-b1c2-3d4e5f6caba6"),
		uuid.MustParse("0192f8e8-1dce-9f0a-b1c2-3d4e5f6cbba7"),
	}
	insertJobRunKindRow(t, e.pool, ids[0], "model_download", "succeeded", 1000, 1000, 0, base)
	insertJobRunKindRow(t, e.pool, ids[1], "clip_embed_rerun", "running", 9, 4, 1, base.Add(time.Minute))
	insertJobRunKindRow(t, e.pool, ids[2], "face_detect_rerun", "queued", 9, 0, 0, base.Add(2*time.Minute))
	insertJobRunKindRow(t, e.pool, ids[3], "face_recognize_rerun", "failed", 7, 5, 2, base.Add(3*time.Minute))
	insertJobRunKindRow(t, e.pool, ids[4], "ocr_rerun", "succeeded", 7, 7, 0, base.Add(4*time.Minute))

	rec := e.do(t, http.MethodGet, "/api/v1/admin/jobs", e.cookie)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET jobs status = %d, want %d (body %q)", rec.Code, http.StatusOK, rec.Body.String())
	}
	var body apigen.JobRunListResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode the jobs response %q: %v", rec.Body.String(), err)
	}
	if len(body.Jobs) != 5 {
		t.Fatalf("jobs = %d, want 5 (body %q)", len(body.Jobs), rec.Body.String())
	}
	for i, want := range []uuid.UUID{ids[4], ids[3], ids[2], ids[1], ids[0]} {
		if body.Jobs[i].Id != want {
			t.Errorf("jobs[%d] id = %q, want %q (newest first)", i, body.Jobs[i].Id, want)
		}
	}

	// The full row shape, on the running rerun row.
	job := body.Jobs[3]
	if job.Id != ids[1] || job.Kind != apigen.ClipEmbedRerun {
		t.Errorf("job = %q kind %q, want the clip_embed_rerun run", job.Id, job.Kind)
	}
	if job.State != apigen.JobRunStateRunning || job.Total != 9 || job.Done != 4 || job.Failed != 1 {
		t.Errorf("job state/total/done/failed = %q/%d/%d/%d, want running/9/4/1",
			job.State, job.Total, job.Done, job.Failed)
	}
	if job.StartedAt != nil || job.FinishedAt != nil {
		t.Errorf("job startedAt/finishedAt = %v/%v, want null (the row never started or finished)",
			job.StartedAt, job.FinishedAt)
	}

	// The limit: a value below 1 answers 400, a value over the maximum
	// is clamped to it, and a small value bounds the page to the
	// newest runs.
	if rec := e.do(t, http.MethodGet, "/api/v1/admin/jobs?limit=0", e.cookie); rec.Code != http.StatusBadRequest {
		t.Fatalf("limit=0 status = %d, want %d (body %q)", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
	if rec := e.do(t, http.MethodGet, "/api/v1/admin/jobs?limit=notanumber", e.cookie); rec.Code != http.StatusBadRequest {
		t.Fatalf("limit=notanumber status = %d, want %d (body %q)", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
	body = decodeJobList(t, e.do(t, http.MethodGet, "/api/v1/admin/jobs?limit=500", e.cookie))
	if len(body.Jobs) != 5 {
		t.Errorf("limit=500 jobs = %d, want 5 (clamped to the maximum)", len(body.Jobs))
	}
	body = decodeJobList(t, e.do(t, http.MethodGet, "/api/v1/admin/jobs?limit=2", e.cookie))
	if len(body.Jobs) != 2 || body.Jobs[0].Id != ids[4] || body.Jobs[1].Id != ids[3] {
		t.Errorf("limit=2 jobs = %v, want the two newest runs", body.Jobs)
	}
}

// decodeModelList decodes one models listing response.
func decodeModelList(t *testing.T, rec *httptest.ResponseRecorder) apigen.ModelListResponse {
	t.Helper()
	if rec.Code != http.StatusOK {
		t.Fatalf("GET models status = %d, want %d (body %q)", rec.Code, http.StatusOK, rec.Body.String())
	}
	var body apigen.ModelListResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode the models response %q: %v", rec.Body.String(), err)
	}
	return body
}

// decodeJobList decodes one jobs listing response.
func decodeJobList(t *testing.T, rec *httptest.ResponseRecorder) apigen.JobRunListResponse {
	t.Helper()
	if rec.Code != http.StatusOK {
		t.Fatalf("GET jobs status = %d, want %d (body %q)", rec.Code, http.StatusOK, rec.Body.String())
	}
	var body apigen.JobRunListResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode the jobs response %q: %v", rec.Body.String(), err)
	}
	return body
}

// modelOf returns the listed model of the given id.
func modelOf(t *testing.T, body apigen.ModelListResponse, id string) apigen.ModelResponse {
	t.Helper()
	for _, model := range body.Models {
		if model.Id == id {
			return model
		}
	}
	t.Fatalf("model %q is not in the listing", id)
	return apigen.ModelResponse{}
}
