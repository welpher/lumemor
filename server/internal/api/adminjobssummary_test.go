// The task catalog's own read and the all-albums sweep's start:
// GET /admin/jobs/summary's queue-count mapping (waiting sums the
// waiting states, active counts running, failed counts discarded),
// the budget's pause switches, the latest batch runs (and the folder
// sweep's always-null slot), and the admin gate; POST
// /admin/albums/rescan's 202/200 in-flight join, the admin gate and
// the unwired-queue 500.
package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivermigrate"

	apigen "github.com/welpher/lumemor/server/internal/api/gen"
	"github.com/welpher/lumemor/server/internal/db"
	dbgen "github.com/welpher/lumemor/server/internal/db/gen"
)

// stubAlbumRescanAlls records the sweep enqueues the endpoint made and
// answers the configured result, the seam-fake pattern of the rescan
// and rerun endpoints.
type stubAlbumRescanAlls struct {
	calls int
	err   error
	id    uuid.UUID
}

// EnqueueAlbumRescanAll implements AlbumRescanAllEnqueuer.
func (s *stubAlbumRescanAlls) EnqueueAlbumRescanAll(context.Context) (uuid.UUID, error) {
	s.calls++
	if s.err != nil {
		return uuid.Nil, s.err
	}
	return s.id, nil
}

// catalogEnv provisions the catalog's test environment: a migrated
// throwaway database with the queue's own tables applied (the summary
// reads river_job), the administrator signed in, and the router wired
// to the stub sweep seam.
type catalogEnv struct {
	router http.Handler
	pool   *pgxpool.Pool
	stub   *stubAlbumRescanAlls
	cookie *http.Cookie
}

// newCatalogEnv provisions the environment with the administrator
// signed in.
func newCatalogEnv(t *testing.T) *catalogEnv {
	t.Helper()
	pool := newSetupEnv(t)
	// The queue's tables: the summary's pipeline read is the queue's
	// own table (production applies these migrations at queue start).
	driver := riverpgxv5.New(pool)
	migrator, err := rivermigrate.New(driver, nil)
	if err != nil {
		t.Fatalf("create river migrator: %v", err)
	}
	if _, err := migrator.Migrate(t.Context(), rivermigrate.DirectionUp, nil); err != nil {
		t.Fatalf("apply river migrations: %v", err)
	}

	stub := &stubAlbumRescanAlls{id: uuid.MustParse("0192f8b1-4c5d-8e9f-a0b1-2c3d4e5f6a70")}
	router := NewRouter(&Deps{
		Pool:            pool,
		Version:         testVersion,
		AlbumRescanAlls: stub,
	})
	rec := postAdmin(t, router, createAdminBody(t, "ada@example.com", testPassword, nil))
	if rec.Code != http.StatusCreated {
		t.Fatalf("create admin: status = %d, want %d (body %q)", rec.Code, http.StatusCreated, rec.Body.String())
	}
	cookie, _ := loginCookie(t, router, "ada@example.com", testPassword, "catalog-tests")
	return &catalogEnv{router: router, pool: pool, stub: stub, cookie: cookie}
}

// insertQueueJob seeds one queue job in the named kind and state (the
// river_job row the pipeline read aggregates). The finalized states
// carry their finish time (the table's own constraint).
func (e *catalogEnv) insertQueueJob(t *testing.T, kind, state string) {
	t.Helper()
	var finalizedAt any
	switch state {
	case "completed", "discarded", "cancelled":
		finalizedAt = time.Now()
	}
	if _, err := e.pool.Exec(context.Background(),
		"INSERT INTO river_job (kind, state, args, max_attempts, finalized_at) VALUES ($1, $2, '{}', 25, $3)",
		kind, state, finalizedAt); err != nil {
		t.Fatalf("insert %s job in state %s: %v", kind, state, err)
	}
}

// insertRun seeds one job_runs row of the named kind.
func (e *catalogEnv) insertRun(t *testing.T, id uuid.UUID, kind, state string, total, done, failed int, startedAt, finishedAt time.Time) {
	t.Helper()
	if _, err := e.pool.Exec(context.Background(), `
		INSERT INTO job_runs (id, kind, state, total, done, failed, started_at, finished_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		pgUUID(id), kind, state, total, done, failed,
		pgTimestamp(startedAt), pgTimestamp(finishedAt)); err != nil {
		t.Fatalf("insert %s run: %v", kind, err)
	}
}

// pgTimestamp renders one timestamp for the fixture inserts; the zero
// value lands NULL.
func pgTimestamp(value time.Time) any {
	if value.IsZero() {
		return nil
	}
	return value
}

// inFlightSweep seeds one running sweep row (the join's subject).
func (e *catalogEnv) inFlightSweep(t *testing.T) uuid.UUID {
	t.Helper()
	id := uuid.MustParse("0192f8c2-5d6e-7f80-91a2-b3c4d5e6f708")
	e.insertRun(t, id, "album_rescan_all", "running", 3, 1, 0, time.Now().Add(-time.Minute), time.Time{})
	return id
}

// summary fires GET /admin/jobs/summary with the given cookie (nil
// sends it unauthenticated).
func (e *catalogEnv) summary(t *testing.T, cookie *http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/jobs/summary", nil)
	if cookie != nil {
		req.AddCookie(cookie)
	}
	rec := httptest.NewRecorder()
	e.router.ServeHTTP(rec, req)
	return rec
}

// rescanAll fires POST /admin/albums/rescan with the given cookie.
func (e *catalogEnv) rescanAll(t *testing.T, cookie *http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/albums/rescan", nil)
	if cookie != nil {
		req.AddCookie(cookie)
	}
	rec := httptest.NewRecorder()
	e.router.ServeHTTP(rec, req)
	return rec
}

// TestJobsSummaryReadsTheCatalog pins the summary's mapping: the
// queue's five-state rows fold into waiting/active/failed (a
// completed or cancelled job contributes nothing), the budget's pause
// switches ride the document, the latest run of each batch kind maps
// onto the summary shape, and the folder sweep's slot stays null (it
// records no run row by design).
func TestJobsSummaryReadsTheCatalog(t *testing.T) {
	e := newCatalogEnv(t)

	// The thumbnail kind exercises every state the mapping reads; a
	// completed job (retained by the queue) must not leak into any
	// count.
	for _, state := range []string{"available", "available", "scheduled", "retryable", "running", "running", "discarded", "completed", "cancelled"} {
		e.insertQueueJob(t, "thumbnail", state)
	}
	e.insertQueueJob(t, "metadata", "running")
	e.insertQueueJob(t, "face_detect", "available")
	e.insertQueueJob(t, "face_detect", "available")
	e.insertQueueJob(t, "face_detect", "available")

	// The budget: two switches on, the rest off.
	if err := db.SaveRunBudget(context.Background(), dbgen.New(e.pool), db.RunBudget{
		ConcurrencyPreset: "standard",
		NightlyEnabled:    true,
		NightlyStart:      "02:00",
		NightlyEnd:        "06:00",
		PausedJobs: db.PausedJobs{
			Thumbnail: false,
			Metadata:  true,
			Transcode: true,
			Ml:        false,
			Places:    false,
		},
	}); err != nil {
		t.Fatalf("save the run budget: %v", err)
	}

	started := time.Date(2026, 11, 19, 2, 0, 0, 0, time.UTC)
	finished := started.Add(250 * time.Second)
	rebuildID := uuid.MustParse("0192f8b0-1c2d-3e4f-5a6b-7c8d9e0f1a2b")
	e.insertRun(t, rebuildID, "rebuild_derivatives", "succeeded", 1240, 1240, 0, started, finished)
	placesID := uuid.MustParse("0192f8c0-2d3e-4f5a-6b7c-8d9e0f1a2b3c")
	e.insertRun(t, placesID, "places_rerun", "failed", 12847, 12000, 2, started, finished)
	sweepID := uuid.MustParse("0192f8d0-3e4f-5a6b-7c8d-9e0f1a2b3c4d")
	e.insertRun(t, sweepID, "album_rescan_all", "succeeded", 12, 12, 0, started, finished)
	downloadID := uuid.MustParse("0192f8e0-4f5a-6b7c-8d9e-0f1a2b3c4d5e")
	e.insertRun(t, downloadID, "model_download", "succeeded", 400371536, 400371536, 0, started, finished)
	// A newer per-album rescan must never answer the sweep's slot.
	e.insertRun(t, uuid.MustParse("0192f8f0-5a6b-7c8d-9e0f-1a2b3c4d5e6f"), "album_rescan", "succeeded", 2, 2, 0, started, finished)

	rec := e.summary(t, e.cookie)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /admin/jobs/summary status = %d, want %d (body %q)", rec.Code, http.StatusOK, rec.Body.String())
	}
	var body apigen.JobsSummaryResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode summary body %q: %v", rec.Body.String(), err)
	}

	thumbnail := body.Pipeline.Thumbnail
	if thumbnail.Waiting != 4 || thumbnail.Active != 2 || thumbnail.Failed != 1 {
		t.Errorf("thumbnail counts = %+v, want waiting 4, active 2, failed 1 (completed/cancelled contribute nothing)", thumbnail)
	}
	if body.Pipeline.Metadata.Active != 1 {
		t.Errorf("metadata active = %d, want 1", body.Pipeline.Metadata.Active)
	}
	if body.Pipeline.FaceDetect.Waiting != 3 {
		t.Errorf("face detect waiting = %d, want 3", body.Pipeline.FaceDetect.Waiting)
	}

	if body.Paused.Metadata != true || body.Paused.Transcode != true {
		t.Errorf("paused = %+v, want metadata and transcode on", body.Paused)
	}
	if body.Paused.Thumbnail || body.Paused.Ml || body.Paused.Places {
		t.Errorf("paused = %+v, want thumbnail/ml/places off", body.Paused)
	}

	if body.Runs.RebuildDerivatives == nil {
		t.Fatal("rebuildDerivatives = nil, want the seeded run")
	}
	rebuild := body.Runs.RebuildDerivatives
	if rebuild.Id != rebuildID || rebuild.State != apigen.JobRunStateSucceeded ||
		rebuild.Total != 1240 || rebuild.Done != 1240 || rebuild.Failed != 0 {
		t.Errorf("rebuild summary = %+v, want the seeded run's facts", rebuild)
	}
	if rebuild.StartedAt == nil || !rebuild.StartedAt.Equal(started) {
		t.Errorf("rebuild startedAt = %v, want %s", rebuild.StartedAt, started)
	}
	if rebuild.FinishedAt == nil || !rebuild.FinishedAt.Equal(finished) {
		t.Errorf("rebuild finishedAt = %v, want %s", rebuild.FinishedAt, finished)
	}
	if body.Runs.PlacesRerun == nil || body.Runs.PlacesRerun.State != apigen.JobRunStateFailed ||
		body.Runs.PlacesRerun.Failed != 2 {
		t.Errorf("placesRerun = %+v, want the failed seed", body.Runs.PlacesRerun)
	}
	if body.Runs.AlbumRescanAll == nil || body.Runs.AlbumRescanAll.Id != sweepID {
		t.Errorf("albumRescanAll = %+v, want the sweep's row %s", body.Runs.AlbumRescanAll, sweepID)
	}
	if body.Runs.FolderSync != nil {
		t.Errorf("folderSync = %+v, want null (the sweep records no run row)", body.Runs.FolderSync)
	}
	if body.Runs.ModelDownload == nil || body.Runs.ModelDownload.Id != downloadID {
		t.Errorf("modelDownload = %+v, want the download's row %s", body.Runs.ModelDownload, downloadID)
	}
}

// TestJobsSummaryEmptyInstance: a fresh instance answers zeros and
// nulls, never an error -- the catalog must render before any work
// ever ran.
func TestJobsSummaryEmptyInstance(t *testing.T) {
	e := newCatalogEnv(t)

	rec := e.summary(t, e.cookie)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /admin/jobs/summary status = %d, want %d (body %q)", rec.Code, http.StatusOK, rec.Body.String())
	}
	var body apigen.JobsSummaryResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode summary body %q: %v", rec.Body.String(), err)
	}
	if body.Pipeline.Thumbnail != (apigen.PipelineJobCounts{}) {
		t.Errorf("thumbnail counts = %+v, want the zero value", body.Pipeline.Thumbnail)
	}
	if body.Runs.RebuildDerivatives != nil || body.Runs.PlacesRerun != nil ||
		body.Runs.AlbumRescanAll != nil || body.Runs.FolderSync != nil || body.Runs.ModelDownload != nil {
		t.Errorf("runs = %+v, want every slot null on a fresh instance", body.Runs)
	}
}

// TestJobsSummaryAdminGate: the summary is the admin catalog -- a
// signed-in plain user answers 403 and no session 401.
func TestJobsSummaryAdminGate(t *testing.T) {
	e := newCatalogEnv(t)

	if rec := e.summary(t, nil); rec.Code != http.StatusUnauthorized {
		t.Errorf("unauthenticated summary status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
	_, plainCookie := seedPlainUser(t, e.pool, "plain@example.com")
	if rec := e.summary(t, plainCookie); rec.Code != http.StatusForbidden {
		t.Errorf("plain-user summary status = %d, want %d", rec.Code, http.StatusForbidden)
	}
}

// TestRescanAllAlbumsStartsAndJoins pins the sweep's endpoint: a 202
// with the seam's fresh run, a 200 with the in-flight sweep's id (and
// no second enqueue), and the shared admin gate.
func TestRescanAllAlbumsStartsAndJoins(t *testing.T) {
	e := newCatalogEnv(t)

	rec := e.rescanAll(t, e.cookie)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("POST /admin/albums/rescan status = %d, want %d (body %q)", rec.Code, http.StatusAccepted, rec.Body.String())
	}
	var started apigen.AlbumRescanAllResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &started); err != nil {
		t.Fatalf("decode rescan body %q: %v", rec.Body.String(), err)
	}
	if started.JobRunId != e.stub.id {
		t.Errorf("started jobRunId = %s, want the seam's %s", started.JobRunId, e.stub.id)
	}
	if e.stub.calls != 1 {
		t.Errorf("enqueue calls = %d, want 1", e.stub.calls)
	}

	// An in-flight sweep joins: 200 with its id and no new enqueue.
	sweepID := e.inFlightSweep(t)
	rec = e.rescanAll(t, e.cookie)
	if rec.Code != http.StatusOK {
		t.Fatalf("join status = %d, want %d (body %q)", rec.Code, http.StatusOK, rec.Body.String())
	}
	var joined apigen.AlbumRescanAllResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &joined); err != nil {
		t.Fatalf("decode join body %q: %v", rec.Body.String(), err)
	}
	if joined.JobRunId != sweepID {
		t.Errorf("joined jobRunId = %s, want the in-flight %s", joined.JobRunId, sweepID)
	}
	if e.stub.calls != 1 {
		t.Errorf("enqueue calls after the join = %d, want the first one only", e.stub.calls)
	}

	// A finished sweep never blocks a new one.
	if _, err := e.pool.Exec(context.Background(),
		"UPDATE job_runs SET state = 'succeeded', finished_at = now() WHERE id = $1",
		pgUUID(sweepID)); err != nil {
		t.Fatalf("close the seeded sweep: %v", err)
	}
	if rec := e.rescanAll(t, e.cookie); rec.Code != http.StatusAccepted {
		t.Errorf("post-finish rescan status = %d, want %d", rec.Code, http.StatusAccepted)
	}
	if e.stub.calls != 2 {
		t.Errorf("enqueue calls after the finished sweep = %d, want 2", e.stub.calls)
	}
}

// TestRescanAllAlbumsGateAndUnwiredQueue: the admin gate and the
// no-queue 500.
func TestRescanAllAlbumsGateAndUnwiredQueue(t *testing.T) {
	e := newCatalogEnv(t)

	if rec := e.rescanAll(t, nil); rec.Code != http.StatusUnauthorized {
		t.Errorf("unauthenticated rescan status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
	_, plainCookie := seedPlainUser(t, e.pool, "plain@example.com")
	if rec := e.rescanAll(t, plainCookie); rec.Code != http.StatusForbidden {
		t.Errorf("plain-user rescan status = %d, want %d", rec.Code, http.StatusForbidden)
	}
	if e.stub.calls != 0 {
		t.Errorf("gated requests enqueued %d runs, want 0", e.stub.calls)
	}

	// No queue wired: the endpoint answers the generic internal error.
	unwired := NewRouter(&Deps{Pool: e.pool, Version: testVersion})
	cookie, _ := loginCookie(t, unwired, "ada@example.com", testPassword, "catalog-tests-2")
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/albums/rescan", nil)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	unwired.ServeHTTP(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("unwired rescan status = %d, want %d (body %q)", rec.Code, http.StatusInternalServerError, rec.Body.String())
	}
}
