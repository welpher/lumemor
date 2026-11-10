// The model download endpoint tests: the admin-role guard
// of the surface (403 for a signed-in plain user, 401 without a
// session), the 202 of the first call (the run is enqueued through the
// seam), the 200 idempotent return of the in-flight -- or, when none
// is in flight, the most recent -- run without re-enqueueing, and the
// MODEL_DOWNLOAD_FAILED mapping when the download cannot be started
// (no queue wired to the surface, or an enqueue that fails).
package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	apigen "github.com/welpher/lumemor/server/internal/api/gen"
)

// stubModelDownloads is the enqueue seam under test: it counts the
// calls, answers a fixed run identifier, and can be told to fail.
type stubModelDownloads struct {
	calls int
	id    uuid.UUID
	err   error
}

// EnqueueModelDownload implements ModelDownloadEnqueuer.
func (s *stubModelDownloads) EnqueueModelDownload(ctx context.Context) (uuid.UUID, error) {
	s.calls++
	if s.err != nil {
		return uuid.Nil, s.err
	}
	return s.id, nil
}

// modelDownloadEnv provisions the endpoint's test environment: a
// migrated throwaway database, the router wired to the stub enqueue
// seam, and both an administrator and a plain signed-in user.
type modelDownloadEnv struct {
	router http.Handler
	pool   *pgxpool.Pool
	stub   *stubModelDownloads
	cookie *http.Cookie
	plain  *http.Cookie
}

func newModelDownloadEnv(t *testing.T) *modelDownloadEnv {
	t.Helper()
	pool := newSetupEnv(t)
	stub := &stubModelDownloads{id: uuid.MustParse("0192f7a1-3b4c-7d8e-9f0a-1b2c3d4e5f60")}
	router := NewRouter(&Deps{Pool: pool, Version: testVersion, ModelDownloads: stub})

	rec := postAdmin(t, router, createAdminBody(t, "ada@example.com", testPassword, nil))
	if rec.Code != http.StatusCreated {
		t.Fatalf("create admin: status = %d, want %d (body %q)", rec.Code, http.StatusCreated, rec.Body.String())
	}
	cookie, _ := loginCookie(t, router, "ada@example.com", testPassword, "model-download-tests")
	_, plainCookie := seedPlainUser(t, pool, "grace@example.com")

	return &modelDownloadEnv{router: router, pool: pool, stub: stub, cookie: cookie, plain: plainCookie}
}

// postDownload fires the download trigger with the given session
// cookie (nil = no session).
func (e *modelDownloadEnv) postDownload(t *testing.T, cookie *http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/models/download", nil)
	if cookie != nil {
		req.AddCookie(cookie)
	}
	rec := httptest.NewRecorder()
	e.router.ServeHTTP(rec, req)
	return rec
}

// insertJobRunRow inserts one model_download job_runs row directly,
// with the given state and creation time (the ordering key of the
// in-flight/latest lookup).
func insertJobRunRow(t *testing.T, pool *pgxpool.Pool, id uuid.UUID, state string, createdAt time.Time) {
	t.Helper()
	_, err := pool.Exec(context.Background(),
		"INSERT INTO job_runs (id, kind, total, state, created_at, updated_at) VALUES ($1, 'model_download', 1000, $2, $3, $3)",
		pgUUID(id), state, createdAt)
	if err != nil {
		t.Fatalf("insert the job_runs row: %v", err)
	}
}

// TestDownloadModelsGuardsTheAdminRole pins the permission model: a
// signed-in plain user answers 403 AUTH_FORBIDDEN and a request
// without a session 401, both before the enqueue seam is ever called.
func TestDownloadModelsGuardsTheAdminRole(t *testing.T) {
	e := newModelDownloadEnv(t)

	rec := e.postDownload(t, e.plain)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("plain user POST status = %d, want %d (body %q)", rec.Code, http.StatusForbidden, rec.Body.String())
	}
	apiErr, _ := decodeError(t, rec)
	if apiErr.Code != apigen.AUTHFORBIDDEN {
		t.Errorf("plain user code = %q, want %q", apiErr.Code, apigen.AUTHFORBIDDEN)
	}

	rec = e.postDownload(t, nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("sessionless POST status = %d, want %d (body %q)", rec.Code, http.StatusUnauthorized, rec.Body.String())
	}
	apiErr, _ = decodeError(t, rec)
	if apiErr.Code != apigen.AUTHINVALIDCREDENTIALS {
		t.Errorf("sessionless code = %q, want %q", apiErr.Code, apigen.AUTHINVALIDCREDENTIALS)
	}

	if e.stub.calls != 0 {
		t.Errorf("the enqueue seam was called %d times, want 0 (the guard rejects first)", e.stub.calls)
	}
}

// TestDownloadModelsStartsTheFirstDownload pins the first call: with
// no run of the kind at all, the endpoint enqueues through the seam
// and answers 202 with the new run's identifier.
func TestDownloadModelsStartsTheFirstDownload(t *testing.T) {
	e := newModelDownloadEnv(t)

	rec := e.postDownload(t, e.cookie)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("first POST status = %d, want %d (body %q)", rec.Code, http.StatusAccepted, rec.Body.String())
	}
	var body apigen.ModelDownloadResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode the response %q: %v", rec.Body.String(), err)
	}
	if body.JobRunId != e.stub.id {
		t.Errorf("first POST jobRunId = %q, want the enqueued %q", body.JobRunId, e.stub.id)
	}
	if e.stub.calls != 1 {
		t.Errorf("the enqueue seam was called %d times, want 1", e.stub.calls)
	}
}

// TestDownloadModelsReturnsTheInFlightRun pins the in-flight arm of
// the idempotence: a run still queued or running answers 200 with its
// identifier and nothing is enqueued.
func TestDownloadModelsReturnsTheInFlightRun(t *testing.T) {
	e := newModelDownloadEnv(t)
	queued := uuid.MustParse("0192f7b2-6c7d-8e9f-a0b1-2c3d4e5f6070")
	running := uuid.MustParse("0192f7c3-7d8e-9f0a-b1c2-3d4e5f607180")
	insertJobRunRow(t, e.pool, queued, "queued", time.Now().Add(-2*time.Minute))
	insertJobRunRow(t, e.pool, running, "running", time.Now().Add(-time.Minute))

	rec := e.postDownload(t, e.cookie)
	if rec.Code != http.StatusOK {
		t.Fatalf("in-flight POST status = %d, want %d (body %q)", rec.Code, http.StatusOK, rec.Body.String())
	}
	var body apigen.ModelDownloadResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode the response %q: %v", rec.Body.String(), err)
	}
	if body.JobRunId != running {
		t.Errorf("in-flight POST jobRunId = %q, want the most recent in-flight %q", body.JobRunId, running)
	}
	if e.stub.calls != 0 {
		t.Errorf("the enqueue seam was called %d times, want 0 (the in-flight run is returned as-is)", e.stub.calls)
	}
}

// TestDownloadModelsReturnsTheMostRecentFinishedRun pins the finished
// arm of the idempotence: with no run in flight, the most recent
// finished run answers 200 with its identifier -- a repeated click
// neither re-downloads a landed group nor piles up runs, the job's own
// retries owning the recovery of a failed one.
func TestDownloadModelsReturnsTheMostRecentFinishedRun(t *testing.T) {
	e := newModelDownloadEnv(t)
	older := uuid.MustParse("0192f7d4-8e9f-a0b1-b2c3-4d5e6f707290")
	latest := uuid.MustParse("0192f7e5-9f0a-b1c2-c3d4-5e6f7071803a")
	insertJobRunRow(t, e.pool, older, "succeeded", time.Now().Add(-2*time.Minute))
	insertJobRunRow(t, e.pool, latest, "failed", time.Now().Add(-time.Minute))

	rec := e.postDownload(t, e.cookie)
	if rec.Code != http.StatusOK {
		t.Fatalf("finished-run POST status = %d, want %d (body %q)", rec.Code, http.StatusOK, rec.Body.String())
	}
	var body apigen.ModelDownloadResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode the response %q: %v", rec.Body.String(), err)
	}
	if body.JobRunId != latest {
		t.Errorf("finished-run POST jobRunId = %q, want the most recent %q", body.JobRunId, latest)
	}
	if e.stub.calls != 0 {
		t.Errorf("the enqueue seam was called %d times, want 0 (the most recent run is returned as-is)", e.stub.calls)
	}
}

// TestDownloadModelsPrefersTheInFlightRun pins the priority between
// the two lookup arms: an older finished run never shadows a newer
// in-flight one.
func TestDownloadModelsPrefersTheInFlightRun(t *testing.T) {
	e := newModelDownloadEnv(t)
	finished := uuid.MustParse("0192f7f6-a0b1-b2c2-d3e4-5f60718049b0")
	inFlight := uuid.MustParse("0192f807-b1c2-c3d4-e5f6-6071804925c1")
	insertJobRunRow(t, e.pool, finished, "succeeded", time.Now().Add(-2*time.Minute))
	insertJobRunRow(t, e.pool, inFlight, "running", time.Now().Add(-time.Minute))

	rec := e.postDownload(t, e.cookie)
	if rec.Code != http.StatusOK {
		t.Fatalf("priority POST status = %d, want %d (body %q)", rec.Code, http.StatusOK, rec.Body.String())
	}
	var body apigen.ModelDownloadResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode the response %q: %v", rec.Body.String(), err)
	}
	if body.JobRunId != inFlight {
		t.Errorf("priority POST jobRunId = %q, want the in-flight %q, not the older finished %q", body.JobRunId, inFlight, finished)
	}
	if e.stub.calls != 0 {
		t.Errorf("the enqueue seam was called %d times, want 0", e.stub.calls)
	}
}

// TestDownloadModelsMapsStartFailure pins the MODEL_DOWNLOAD_FAILED
// mapping: an enqueue that fails (and a surface with no queue wired at
// all) answers 500 with the surface's own failure code -- the download
// could not be started.
func TestDownloadModelsMapsStartFailure(t *testing.T) {
	e := newModelDownloadEnv(t)
	e.stub.err = errors.New("the queue insert failed")

	rec := e.postDownload(t, e.cookie)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("failing enqueue POST status = %d, want %d (body %q)", rec.Code, http.StatusInternalServerError, rec.Body.String())
	}
	apiErr, _ := decodeError(t, rec)
	if apiErr.Code != apigen.MODELDOWNLOADFAILED {
		t.Errorf("failing enqueue code = %q, want %q", apiErr.Code, apigen.MODELDOWNLOADFAILED)
	}

	// A surface without the enqueue seam wired answers the same code.
	noSeam := NewRouter(&Deps{Pool: e.pool, Version: testVersion})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/models/download", nil)
	req.AddCookie(e.cookie)
	rec = httptest.NewRecorder()
	noSeam.ServeHTTP(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("unwired POST status = %d, want %d (body %q)", rec.Code, http.StatusInternalServerError, rec.Body.String())
	}
	apiErr, _ = decodeError(t, rec)
	if apiErr.Code != apigen.MODELDOWNLOADFAILED {
		t.Errorf("unwired code = %q, want %q", apiErr.Code, apigen.MODELDOWNLOADFAILED)
	}
}
