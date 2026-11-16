// The task panel's cancel (E41): the endpoint's faces -- the 204 over
// an in-flight run (the flag lands), the queued run's instant close
// with its finished event, the finished run's 409, the unknown id's
// 404 -- plus the subject label's ride on the listing (a rescan run
// names its album, every other kind answers null) and the model
// params' validated domain (the out-of-bounds 400 naming the field,
// the corrupt stored row answering the pipeline defaults).
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
	"github.com/jackc/pgx/v5"

	apigen "github.com/welpher/lumemor/server/internal/api/gen"
	"github.com/welpher/lumemor/server/internal/db"
	dbgen "github.com/welpher/lumemor/server/internal/db/gen"
)

// TestCancelJobRunFaces pins the cancel endpoint's four answers: an
// unknown id 404s, a finished run 409s, a running run 204s with the
// flag set (the drivers' boundary check reads it), and a queued run
// closes cancelled AT ONCE (nothing ever started, so there is no
// boundary to wait for) with the finished event announcing it.
func TestCancelJobRunFaces(t *testing.T) {
	env, _ := newDerivativesEnv(t)

	cancel := func(id uuid.UUID) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/jobs/"+id.String()+"/cancel", nil)
		req.AddCookie(env.cookie)
		rec := httptest.NewRecorder()
		env.router.ServeHTTP(rec, req)
		return rec
	}

	// The unknown id: 404 with the code named.
	unknown := cancel(uuid.MustParse("0192f8e9-2def-8f0a-b1c2-3d4e5f6caba7"))
	if unknown.Code != http.StatusNotFound {
		t.Fatalf("cancel unknown status = %d, want %d (body %q)", unknown.Code, http.StatusNotFound, unknown.Body.String())
	}
	var envelope struct {
		Code string `json:"code"`
	}
	if err := json.Unmarshal(unknown.Body.Bytes(), &envelope); err != nil || envelope.Code != "JOB_RUN_NOT_FOUND" {
		t.Errorf("cancel unknown code = %q err %v, want JOB_RUN_NOT_FOUND", envelope.Code, err)
	}

	// A finished run: 409, the row's own state is the honest answer.
	finished := uuid.MustParse("0192f8e9-3efa-9f0a-b1c2-3d4e5f6cbba8")
	insertJobRunKindRow(t, env.pool, finished, "clip_embed_rerun", "succeeded", 9, 9, 0, time.Now())
	notCancellable := cancel(finished)
	if notCancellable.Code != http.StatusConflict {
		t.Fatalf("cancel finished status = %d, want %d (body %q)", notCancellable.Code, http.StatusConflict, notCancellable.Body.String())
	}
	if err := json.Unmarshal(notCancellable.Body.Bytes(), &envelope); err != nil || envelope.Code != "JOB_NOT_CANCELLABLE" {
		t.Errorf("cancel finished code = %q err %v, want JOB_NOT_CANCELLABLE", envelope.Code, err)
	}

	// A running run: 204 and the flag lands -- the drivers' boundary
	// check reads it and closes the run cancelled with the committed
	// portion kept.
	running := uuid.MustParse("0192f8e9-4fab-8f0a-b1c2-3d4e5f6caba9")
	insertJobRunKindRow(t, env.pool, running, "clip_embed_rerun", "running", 9, 4, 0, time.Now())
	flagged := cancel(running)
	if flagged.Code != http.StatusNoContent {
		t.Fatalf("cancel running status = %d, want %d (body %q)", flagged.Code, http.StatusNoContent, flagged.Body.String())
	}
	var stopRequested bool
	if err := env.pool.QueryRow(context.Background(),
		"SELECT stop_requested FROM job_runs WHERE id = $1", pgUUID(running)).Scan(&stopRequested); err != nil {
		t.Fatalf("load the flagged run: %v", err)
	}
	if !stopRequested {
		t.Error("cancel running: stop_requested = false, want true (the drivers' boundary check)")
	}

	// A queued run: 204 and the row closes cancelled at once -- the
	// finished event is the page's convergence, the counts stay the
	// row's own zeros.
	queued := uuid.MustParse("0192f8e9-5abc-9f0a-b1c2-3d4e5f6cbba0")
	insertJobRunKindRow(t, env.pool, queued, "clip_embed_rerun", "queued", 9, 0, 0, time.Now())
	closed := cancel(queued)
	if closed.Code != http.StatusNoContent {
		t.Fatalf("cancel queued status = %d, want %d (body %q)", closed.Code, http.StatusNoContent, closed.Body.String())
	}
	var state string
	if err := env.pool.QueryRow(context.Background(),
		"SELECT state::text FROM job_runs WHERE id = $1", pgUUID(queued)).Scan(&state); err != nil {
		t.Fatalf("load the closed run: %v", err)
	}
	if state != "cancelled" {
		t.Errorf("cancelled queued run state = %s, want cancelled (nothing ever started)", state)
	}
}

// TestCancelJobRunFinishedByRace pins the raced close: a flag write
// that matched nothing (the driver closed the run between the
// endpoint's read and its write) answers the 409, never a silent 204.
func TestCancelJobRunFinishedByRace(t *testing.T) {
	env, _ := newDerivativesEnv(t)

	runID := uuid.MustParse("0192f8e9-6abd-8f0a-b1c2-3d4e5f6caba1")
	// The row starts running; the endpoint's read sees it in flight,
	// and the "driver" closes it before the flag write runs.
	insertJobRunKindRow(t, env.pool, runID, "clip_embed_rerun", "running", 9, 4, 0, time.Now())
	if _, err := env.pool.Exec(context.Background(),
		"UPDATE job_runs SET state = 'succeeded' WHERE id = $1", pgUUID(runID)); err != nil {
		t.Fatalf("close the run: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/jobs/"+runID.String()+"/cancel", nil)
	req.AddCookie(env.cookie)
	rec := httptest.NewRecorder()
	env.router.ServeHTTP(rec, req)
	if rec.Code != http.StatusConflict {
		t.Fatalf("raced cancel status = %d, want %d (body %q)", rec.Code, http.StatusConflict, rec.Body.String())
	}
	var envelope struct {
		Code string `json:"code"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil || envelope.Code != "JOB_NOT_CANCELLABLE" {
		t.Errorf("raced cancel code = %q err %v, want JOB_NOT_CANCELLABLE", envelope.Code, err)
	}
}

// TestListJobsCarriesTheRescanSubject pins the subject's ride on the
// listing (E41): a rescan run whose album still exists names it; every
// other kind answers null.
func TestListJobsCarriesTheRescanSubject(t *testing.T) {
	e := newAdminModelsEnv(t)

	owner := e.ownerID
	albumID := uuid.MustParse("0192f8e9-7abe-9f0a-b1c2-3d4e5f6caba2")
	if _, err := e.pool.Exec(context.Background(),
		`INSERT INTO albums (id, owner_id, title, kind) VALUES ($1, $2, 'Family archive', 'directory')`,
		pgUUID(albumID), pgUUID(owner)); err != nil {
		t.Fatalf("seed the subject album: %v", err)
	}
	rescanRun := uuid.MustParse("0192f8e9-8abf-8f0a-b1c2-3d4e5f6caba3")
	if _, err := e.pool.Exec(context.Background(),
		`INSERT INTO job_runs (id, kind, total, done, failed, state, album_id) VALUES ($1, 'album_rescan', 2, 2, 0, 'succeeded', $2)`,
		pgUUID(rescanRun), pgUUID(albumID)); err != nil {
		t.Fatalf("seed the rescan run: %v", err)
	}
	plainRun := uuid.MustParse("0192f8e9-9ac0-9f0a-b1c2-3d4e5f6cbba4")
	if _, err := e.pool.Exec(context.Background(),
		`INSERT INTO job_runs (id, kind, total, done, failed, state) VALUES ($1, 'clip_embed_rerun', 2, 2, 0, 'succeeded')`,
		pgUUID(plainRun)); err != nil {
		t.Fatalf("seed the plain run: %v", err)
	}

	body := decodeJobList(t, e.do(t, http.MethodGet, "/api/v1/admin/jobs", e.cookie))
	for _, job := range body.Jobs {
		switch job.Id {
		case rescanRun:
			if job.Subject == nil || *job.Subject != "Family archive" {
				t.Errorf("rescan run subject = %v, want the album's title", job.Subject)
			}
		case plainRun:
			if job.Subject != nil {
				t.Errorf("plain rerun subject = %q, want null (the instance-wide reruns never name one)", *job.Subject)
			}
		}
	}
}

// TestModelParamsDomain pins the params' validated domain (E41): a
// value outside the closed 0-1 domain answers 400 naming the field
// with nothing written, the whole-object save carries the params, and
// a corrupt stored row answers the pipeline defaults on the read.
func TestModelParamsDomain(t *testing.T) {
	env := newNetworkingEnv(t, http.StatusOK, `{}`)

	// The defaults: a fresh instance reads the pipeline's own posture.
	rec := env.do(t, http.MethodGet, "/api/v1/admin/settings/models", "", env.cookie)
	var initial apigen.ModelSettingsResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &initial); err != nil {
		t.Fatalf("decode the default: %v", err)
	}
	if initial.Params.FaceDetectMinScore != 0.6 || initial.Params.FaceRecognizeThreshold != 0.55 {
		t.Errorf("default params = %v/%v, want the pipeline's own 0.6/0.55",
			initial.Params.FaceDetectMinScore, initial.Params.FaceRecognizeThreshold)
	}

	// The out-of-bounds save: 400, the field named, nothing written.
	for name, body := range map[string]string{
		"detect high":  `{"autoDownload":true,"allowedHosts":["huggingface.co"],"mirrorBaseUrl":"","params":{"faceDetectMinScore":1.5,"faceRecognizeThreshold":0.55}}`,
		"detect low":   `{"autoDownload":true,"allowedHosts":["huggingface.co"],"mirrorBaseUrl":"","params":{"faceDetectMinScore":-0.05,"faceRecognizeThreshold":0.55}}`,
		"cluster high": `{"autoDownload":true,"allowedHosts":["huggingface.co"],"mirrorBaseUrl":"","params":{"faceDetectMinScore":0.6,"faceRecognizeThreshold":1.01}}`,
		"missing knob": `{"autoDownload":true,"allowedHosts":["huggingface.co"],"mirrorBaseUrl":"","params":{"faceDetectMinScore":0.6}}`,
	} {
		if rec := env.do(t, http.MethodPut, "/api/v1/admin/settings/models", body, env.cookie); rec.Code != http.StatusBadRequest {
			t.Errorf("%s: status = %d, want %d", name, rec.Code, http.StatusBadRequest)
		}
	}
	var stored string
	err := env.pool.QueryRow(context.Background(),
		`SELECT value::text FROM instance_settings WHERE key = 'model_settings'`).Scan(&stored)
	if err == nil {
		t.Errorf("the refused saves wrote a row: %s", stored)
	} else if !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("check the refused saves' writes: %v", err)
	}

	// The whole-object save carries the params; the read answers them.
	saved := env.do(t, http.MethodPut, "/api/v1/admin/settings/models",
		`{"autoDownload":true,"allowedHosts":["huggingface.co"],"mirrorBaseUrl":"","params":{"faceDetectMinScore":0.8,"faceRecognizeThreshold":0.45}}`,
		env.cookie)
	if saved.Code != http.StatusOK {
		t.Fatalf("save: status = %d, want %d (body %q)", saved.Code, http.StatusOK, saved.Body.String())
	}
	var answer apigen.ModelSettingsResponse
	if err := json.Unmarshal(saved.Body.Bytes(), &answer); err != nil {
		t.Fatalf("decode the save answer: %v", err)
	}
	if answer.Params.FaceDetectMinScore != 0.8 || answer.Params.FaceRecognizeThreshold != 0.45 {
		t.Errorf("saved params = %v/%v, want 0.8/0.45",
			answer.Params.FaceDetectMinScore, answer.Params.FaceRecognizeThreshold)
	}

	// The corrupt row: the params object replaced with garbage answers
	// the pipeline defaults, the read never fails -- the trash
	// retention's corrupt-row discipline, at the params' granularity.
	if _, err := env.pool.Exec(context.Background(),
		`UPDATE instance_settings SET value = '{"autoDownload":true,"allowedHosts":["huggingface.co"],"mirrorBaseUrl":"","params":{"faceDetectMinScore":7,"faceRecognizeThreshold":"bad"}}' WHERE key = 'model_settings'`); err != nil {
		t.Fatalf("corrupt the params row: %v", err)
	}
	settings, err := db.ModelSettingsOf(context.Background(), dbgen.New(env.pool))
	if err != nil {
		t.Fatalf("read the corrupt row: %v", err)
	}
	if settings.Params.FaceDetectMinScore != 0.6 || settings.Params.FaceRecognizeThreshold != 0.55 {
		t.Errorf("corrupt-row params = %v/%v, want the pipeline defaults 0.6/0.55",
			settings.Params.FaceDetectMinScore, settings.Params.FaceRecognizeThreshold)
	}
	if !settings.AutoDownload {
		t.Error("corrupt-row policy = false, want true (the honest rest of the document survives)")
	}
}
