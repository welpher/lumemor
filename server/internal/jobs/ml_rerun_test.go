// The ML rerun orchestration tests (M3a Task 4, closed by Task 8): the
// rerun framework -- the job_runs row's lifecycle (created with the
// asset count as its total, flipped to running when the driver picks
// it up, closed with the converged counts), the per-asset dispatch
// seam, the job.* events, and the rerun kinds' vocabulary.
//
// Task 8 replaced the fake per-asset dispatcher with the real one: the
// dispatcher enqueues the slot's per-asset ML job (carrying the run's
// identifier) onto the ml queue, the workers report their own
// completions into the run's row, and the driver waits for the counts
// to converge and closes the row itself. The fake-dispatcher tests
// below keep driving the seam (their fakes enqueue nothing but report
// completions directly, the honest shape of the seam's contract), and
// the real-dispatch tests drive the whole closed loop through the real
// workers.
package jobs

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/riverqueue/river"

	dbgen "github.com/welpher/lumemor/server/internal/db/gen"
	"github.com/welpher/lumemor/server/internal/ml"
)

// Pinned digests of the committed ml fixtures the rerun tests copy
// into their throwaway model directories.
const rerunConst512SHA256 = "92d6b93b89bf6ccec24222be86689b6ff18370503d3d71316d5d325bd2116766"

// countJobRunKindRows counts the job_runs rows of one kind.
func (e *env) countJobRunKindRows(t *testing.T, kind string) int {
	t.Helper()
	var count int
	if err := e.pool.QueryRow(context.Background(),
		"SELECT count(*) FROM job_runs WHERE kind = $1", kind).Scan(&count); err != nil {
		t.Fatalf("count job_runs rows of kind %s: %v", kind, err)
	}
	return count
}

// insertReadyImage stores one image asset whose pipeline is complete
// (ready, live): the rerun's scope.
func (e *env) insertReadyImage(t *testing.T, rel string) uuid.UUID {
	t.Helper()
	id := e.insertAsset(t, rel, jpegBytes(t),
		time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC).Truncate(time.Microsecond), `{}`)
	if _, err := e.pool.Exec(context.Background(),
		"UPDATE assets SET ready_at = now(), pending_jobs = 0 WHERE id = $1", pgUUID(id)); err != nil {
		t.Fatalf("ready the asset: %v", err)
	}
	return id
}

// insertPipelineImage stores one image asset still mid-pipeline: out
// of the rerun's scope (ML jobs belong after asset readiness).
func (e *env) insertPipelineImage(t *testing.T, rel string) uuid.UUID {
	t.Helper()
	return e.insertAsset(t, rel, jpegBytes(t),
		time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC).Truncate(time.Microsecond), `{}`)
}

// insertTrashedImage stores one image asset sitting in the trash: out
// of the rerun's scope (its products serve nothing while trashed).
func (e *env) insertTrashedImage(t *testing.T, rel string) uuid.UUID {
	t.Helper()
	id := e.insertReadyImage(t, rel)
	if _, err := e.pool.Exec(context.Background(),
		"UPDATE assets SET deleted_at = now() WHERE id = $1", pgUUID(id)); err != nil {
		t.Fatalf("trash the asset: %v", err)
	}
	return id
}

// insertRerunVideo stores one video asset: out of the rerun's scope
// (only image assets enter the ML pipeline in this milestone).
func (e *env) insertRerunVideo(t *testing.T, rel string) uuid.UUID {
	t.Helper()
	id := e.insertAssetKind(t, rel, jpegBytes(t),
		time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC).Truncate(time.Microsecond), `{}`,
		dbgen.MediaKindVideo, "video/mp4", 1)
	if _, err := e.pool.Exec(context.Background(),
		"UPDATE assets SET ready_at = now(), pending_jobs = 0 WHERE id = $1", pgUUID(id)); err != nil {
		t.Fatalf("ready the video asset: %v", err)
	}
	return id
}

// newEnvReportingDispatch provisions an environment whose rerun
// dispatch is a fake of the seam's honest shape: the enqueue itself
// succeeds (nil return) and the per-asset job's outcome -- which the
// real workers report themselves -- is reported directly by the fake.
// failing (read at dispatch time) is the asset whose job failed. The
// dispatched map records the dispatch scope.
//
// The fake captures the environment variable by reference: the driver
// only executes the fake after the queue started and the test
// enqueued a run, long after the variable is bound.
func newEnvReportingDispatch(t *testing.T, dispatched *map[uuid.UUID]int, mu *sync.Mutex, failing *uuid.UUID) *env {
	t.Helper()
	var e *env
	fake := func(ctx context.Context, slot ml.Slot, jobRunID, assetID uuid.UUID) error {
		mu.Lock()
		defer mu.Unlock()
		(*dispatched)[assetID]++
		if assetID == *failing {
			return e.queries.IncrementJobRunFailed(ctx, pgUUID(jobRunID))
		}
		return e.queries.IncrementJobRunDone(ctx, pgUUID(jobRunID))
	}
	e = newEnvRerun(t, fake)
	return e
}

// discardLogger builds the quiet logger the direct worker
// constructions use.
func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// TestMlRerunRunLifecycleAndCounting pins the framework's main path:
// the run row is created with the image asset count as its total
// (videos, still-pipelined and trashed assets are out of scope), the
// driver flips it to running, dispatches every asset through the seam,
// the completions count into the row (successes into done, failures
// into failed), the run closes with a failed state when any completion
// failed, and the job.* events announce it with the run's identity,
// counts and state.
func TestMlRerunRunLifecycleAndCounting(t *testing.T) {
	var failing uuid.UUID
	var mu sync.Mutex
	dispatched := map[uuid.UUID]int{}
	e := newEnvReportingDispatch(t, &dispatched, &mu, &failing)

	good := e.insertReadyImage(t, "originals/rerun/good-1.jpg")
	good2 := e.insertReadyImage(t, "originals/rerun/good-2.jpg")
	failing = e.insertReadyImage(t, "originals/rerun/failing.jpg")
	e.insertRerunVideo(t, "originals/rerun/video.mp4")
	e.insertPipelineImage(t, "originals/rerun/pipelining.jpg")
	e.insertTrashedImage(t, "originals/rerun/trashed.jpg")

	jobRunID, err := e.queue.EnqueueMlRerun(context.Background(), ml.SlotClipEmbed)
	if err != nil {
		t.Fatalf("enqueue the clip rerun: %v", err)
	}

	// The row lands with the scope's count as its total and the slot's
	// pinned kind.
	row := e.jobRunOf(t, jobRunID)
	if row.kind != "clip_embed_rerun" {
		t.Errorf("row kind = %q, want clip_embed_rerun", row.kind)
	}
	if row.total != 3 {
		t.Errorf("row total = %d, want 3 (the three ready images alone)", row.total)
	}

	// The run converges: two successes, one failure, a failed state.
	eventually(t, jobWaitTimeout, "the rerun run to close", func() bool {
		row := e.jobRunOf(t, jobRunID)
		return row.state == "failed" && row.done == 2 && row.failed == 1 &&
			row.startedAt.Valid && row.finishedAt.Valid
	})
	row = e.jobRunOf(t, jobRunID)
	if !row.startedAt.Valid {
		t.Error("started_at is null, want the running flip's time")
	}
	if !row.finishedAt.Valid {
		t.Error("finished_at is null, want the closing time")
	}

	// The dispatch scope: exactly the three ready images, each once.
	mu.Lock()
	defer mu.Unlock()
	if len(dispatched) != 3 || dispatched[good] != 1 || dispatched[good2] != 1 || dispatched[failing] != 1 {
		t.Errorf("dispatched assets = %v, want exactly the three ready images once each", dispatched)
	}

	// The events: one start, a bounded number of progress reports, one
	// close -- every payload carrying the run's identifier, kind,
	// counts and state.
	if got := countEvents(e.bus, "job.started"); got != 1 {
		t.Errorf("job.started events = %d, want 1", got)
	}
	started := eventPayloadOf(t, e.bus, "job.started")
	if started["jobRunId"] != jobRunID.String() || started["kind"] != "clip_embed_rerun" ||
		started["total"] != "3" || started["done"] != "0" || started["failed"] != "0" ||
		started["state"] != "running" {
		t.Errorf("job.started payload = %v, want the run's identity and zero progress", started)
	}
	if got := countEvents(e.bus, "job.progress"); got > 3 {
		t.Errorf("job.progress events = %d, want at most 3 (the throttle collapses the fast completions)", got)
	}
	if got := countEvents(e.bus, "job.finished"); got != 1 {
		t.Errorf("job.finished events = %d, want 1", got)
	}
	finished := eventPayloadOf(t, e.bus, "job.finished")
	if finished["jobRunId"] != jobRunID.String() || finished["kind"] != "clip_embed_rerun" ||
		finished["total"] != "3" || finished["done"] != "2" || finished["failed"] != "1" ||
		finished["state"] != "failed" {
		t.Errorf("job.finished payload = %v, want the converged counts", finished)
	}
}

// TestMlRerunAllAssetsSucceedFlipsSucceeded pins the all-success close:
// every dispatched job succeeding closes the run as succeeded with the
// counts converged and no failure anywhere.
func TestMlRerunAllAssetsSucceedFlipsSucceeded(t *testing.T) {
	var failing uuid.UUID
	var mu sync.Mutex
	dispatched := map[uuid.UUID]int{}
	e := newEnvReportingDispatch(t, &dispatched, &mu, &failing)
	e.insertReadyImage(t, "originals/rerun-ok/one.jpg")
	e.insertReadyImage(t, "originals/rerun-ok/two.jpg")

	jobRunID, err := e.queue.EnqueueMlRerun(context.Background(), ml.SlotOcr)
	if err != nil {
		t.Fatalf("enqueue the ocr rerun: %v", err)
	}
	eventually(t, jobWaitTimeout, "the rerun run to close succeeded", func() bool {
		row := e.jobRunOf(t, jobRunID)
		return row.state == "succeeded" && row.done == 2 && row.failed == 0
	})
}

// TestMlRerunKindsAndTheEmptyLibrary pin the kind vocabulary of the
// four rerun slots and the empty-library edge: every slot maps onto
// its own job_runs kind, and a run over an empty scope closes
// immediately as succeeded with nothing to count.
func TestMlRerunKindsAndTheEmptyLibrary(t *testing.T) {
	e := newEnv(t) // no assets; the real dispatch is wired but has nothing to do

	for slot, wantKind := range map[ml.Slot]string{
		ml.SlotFaceDetect:    "face_detect_rerun",
		ml.SlotFaceRecognize: "face_recognize_rerun",
		ml.SlotClipEmbed:     "clip_embed_rerun",
		ml.SlotOcr:           "ocr_rerun",
	} {
		jobRunID, err := e.queue.EnqueueMlRerun(context.Background(), slot)
		if err != nil {
			t.Fatalf("enqueue the %s rerun: %v", slot, err)
		}
		eventually(t, jobWaitTimeout, "the empty rerun to close", func() bool {
			row := e.jobRunOf(t, jobRunID)
			return row.state == "succeeded"
		})
		row := e.jobRunOf(t, jobRunID)
		if row.kind != wantKind {
			t.Errorf("%s rerun kind = %q, want %q", slot, row.kind, wantKind)
		}
		if row.total != 0 || row.done != 0 || row.failed != 0 {
			t.Errorf("%s rerun total/done/failed = %d/%d/%d, want zeros", slot, row.total, row.done, row.failed)
		}
	}

	// A slot outside the ML pipeline has no rerun: the enqueue refuses
	// it instead of guessing a kind.
	if _, err := e.queue.EnqueueMlRerun(context.Background(), ml.Slot("stt")); err == nil {
		t.Error("EnqueueMlRerun of an unrerunnable slot returned nil, want an error")
	}
}

// TestMlRerunWithoutDispatchFailsTheRun pins the honest placeholder
// boundary: a driver constructed without any dispatcher wired (a
// configuration only direct construction can still produce -- the
// queue wires the real dispatcher by default) fails a non-empty run
// cleanly instead of pretending to have run.
func TestMlRerunWithoutDispatchFailsTheRun(t *testing.T) {
	e := newEnv(t)
	e.insertReadyImage(t, "originals/rerun-placeholder/one.jpg")
	e.insertReadyImage(t, "originals/rerun-placeholder/two.jpg")

	id, err := uuid.NewV7()
	if err != nil {
		t.Fatalf("new run id: %v", err)
	}
	if _, err := e.pool.Exec(context.Background(), `
		INSERT INTO job_runs (id, kind, total) VALUES ($1, 'face_detect_rerun', 2)`,
		pgUUID(id)); err != nil {
		t.Fatalf("insert the run row: %v", err)
	}
	worker := &MlRerunWorker{pool: e.pool, bus: e.bus, logger: discardLogger()}
	if err := worker.Work(context.Background(), &river.Job[MlRerunArgs]{
		Args: MlRerunArgs{JobRunID: id, Slot: string(ml.SlotFaceDetect)},
	}); err != nil {
		t.Fatalf("Work error = %v, want nil (the row closes failed; the driver's error semantics are River's business)", err)
	}
	row := e.jobRunOf(t, id)
	if row.state != "failed" {
		t.Errorf("row state = %q, want failed", row.state)
	}
	if row.done != 0 || row.failed != 0 {
		t.Errorf("row done/failed = %d/%d, want zeros (nothing was dispatched)", row.done, row.failed)
	}
	finished := eventPayloadOf(t, e.bus, "job.finished")
	if finished["state"] != "failed" {
		t.Errorf("job.finished state = %q, want failed", finished["state"])
	}
}

// TestMlRerunRealDispatchCountsWorkerCompletions pins the closed loop
// with the REAL dispatcher and the REAL workers: a clip rerun over
// three ready images dispatches three clip_embed jobs onto the ml
// queue, the jobs run for real over the const512 fixture model, their
// completions report into the run's row (the one whose original file
// went missing fails -- its job fails, one attempt, counted once), and
// the driver closes the row with the converged counts. The two
// successful assets' embeddings land; the failed one leaves no row.
func TestMlRerunRealDispatchCountsWorkerCompletions(t *testing.T) {
	requireThumbnails(t)
	requireMlRuntime(t)
	e := newEnv(t) // the real dispatch is wired by production default
	e.activateFixtureModel(t, clipFixtureSpec)

	good := e.insertReadyImage(t, "originals/rerun-real/good-1.jpg")
	good2 := e.insertReadyImage(t, "originals/rerun-real/good-2.jpg")
	failing := e.insertReadyImage(t, "originals/rerun-real/failing.jpg")
	// The failing asset's original vanishes after the row landed: the
	// dispatched job's decode fails.
	if err := os.Remove(filepath.Join(e.dataDir, "originals", "rerun-real", "failing.jpg")); err != nil {
		t.Fatalf("remove the failing asset's file: %v", err)
	}

	jobRunID, err := e.queue.EnqueueMlRerun(context.Background(), ml.SlotClipEmbed)
	if err != nil {
		t.Fatalf("enqueue the clip rerun: %v", err)
	}
	eventually(t, jobWaitTimeout, "the rerun to close with the real completions", func() bool {
		row := e.jobRunOf(t, jobRunID)
		return row.state == "failed" && row.done == 2 && row.failed == 1 && row.finishedAt.Valid
	})

	// The two successful assets carry their embeddings; the failed one
	// carries none.
	for _, id := range []uuid.UUID{good, good2} {
		if rows := e.clipEmbeddingsOf(t, id); len(rows) != 1 || rows[0].dims != 512 {
			t.Errorf("asset %s clip rows = %+v, want the single 512-dim embedding", id, rows)
		}
	}
	if rows := e.clipEmbeddingsOf(t, failing); len(rows) != 0 {
		t.Errorf("the failed asset clip rows = %d, want 0", len(rows))
	}

	// The dispatched jobs are the real per-asset jobs, each once.
	for _, id := range []uuid.UUID{good, good2, failing} {
		if got := e.riverJobsOf(t, id)["clip_embed"]; got != 1 {
			t.Errorf("asset %s clip_embed jobs = %d, want 1", id, got)
		}
	}

	finished := eventPayloadOf(t, e.bus, "job.finished")
	if finished["jobRunId"] != jobRunID.String() || finished["kind"] != "clip_embed_rerun" ||
		finished["total"] != "3" || finished["done"] != "2" || finished["failed"] != "1" ||
		finished["state"] != "failed" {
		t.Errorf("job.finished payload = %v, want the converged counts of the real completions", finished)
	}
}

// TestMlRerunDispatchesEachSlotsJob pins the four rerun kinds' dispatch
// vocabulary: each slot's dispatch enqueues exactly that slot's
// per-asset job (face_recognize_rerun dispatches the face_recognize
// job DIRECTLY -- the recognition worker redetects its own landmarks,
// so it stands alone; the rerun does not first re-run detection).
func TestMlRerunDispatchesEachSlotsJob(t *testing.T) {
	e := newEnv(t)
	id := e.insertReadyImage(t, "originals/rerun-map/one.jpg")

	runID, err := uuid.NewV7()
	if err != nil {
		t.Fatalf("new run id: %v", err)
	}
	for slot, want := range map[ml.Slot]string{
		ml.SlotFaceDetect:    FaceDetectKind,
		ml.SlotFaceRecognize: FaceRecognizeKind,
		ml.SlotClipEmbed:     ClipEmbedKind,
		ml.SlotOcr:           OcrKind,
	} {
		if err := e.queue.dispatchMlRerun(context.Background(), slot, runID, id); err != nil {
			t.Fatalf("dispatchMlRerun(%s) error = %v, want nil", slot, err)
		}
		if got := e.riverJobsOf(t, id)[want]; got != 1 {
			t.Errorf("asset river jobs after dispatching %s = %v, want one %s job", slot, e.riverJobsOf(t, id), want)
		}
	}

	// Re-dispatching the same run's asset is a no-op: the per-run args
	// uniqueness holds (a driver retry's re-walk re-converges without
	// piling duplicate jobs or double-counting completions).
	if err := e.queue.dispatchMlRerun(context.Background(), ml.SlotClipEmbed, runID, id); err != nil {
		t.Fatalf("re-dispatch error = %v, want nil (the unique insert dedupes)", err)
	}
	if got := e.riverJobsOf(t, id)[ClipEmbedKind]; got != 1 {
		t.Errorf("clip_embed jobs after the re-dispatch = %d, want 1 (the per-run uniqueness)", got)
	}
}

// TestMlRerunFaceRecognizeRerunEmbedsDirectly pins the recognition
// rerun's pinned semantics end to end: a face_detect_rerun first
// lands the portrait's face (the recognition worker matches its
// redetections against stored faces rows), the recognize slot fills
// only AFTER that (so the detection's completion chains nothing --
// the chain needs the slot active at completion time), and the
// face_recognize_rerun run then dispatches face_recognize jobs
// DIRECTLY (no second face_detect_rerun run of its own): the
// recognition worker redetects the portrait's landmarks itself,
// embeds the matched face with the active recognize model, and the
// run's row converges succeeded over the reported completions.
func TestMlRerunFaceRecognizeRerunEmbedsDirectly(t *testing.T) {
	requireThumbnails(t)
	requireMlRuntime(t)
	e := newEnv(t)
	e.activateFixtureModel(t, yunetFixtureSpec)

	id := e.insertFaceAsset(t, "originals/rerun-recognize/portrait.jpg", portraitFixture(t, "face-portrait.jpg"))

	// The detection rerun: the portrait's face lands, and with the
	// recognize slot still empty the completion chains nothing.
	detectRun, err := e.queue.EnqueueMlRerun(context.Background(), ml.SlotFaceDetect)
	if err != nil {
		t.Fatalf("enqueue the face detect rerun: %v", err)
	}
	eventually(t, jobWaitTimeout, "the detection rerun to close and the face to land", func() bool {
		row := e.jobRunOf(t, detectRun)
		return row.state == "succeeded" && row.done >= 1 && len(e.facesOf(t, id)) == 1
	})

	recognizeModel := e.activateFixtureModel(t, fixtureModelSpec{
		id:        "sface-const128",
		slot:      ml.SlotFaceRecognize,
		file:      "const128.onnx",
		sha256:    "e5d4940db6bddc912dee319434394700cf82fd8950cc32e609435ae7f9597263",
		inputSize: ml.ManifestSize{Width: 112, Height: 112},
		dim:       128,
	})

	// The recognition rerun: dispatches the recognition job directly.
	jobRunID, err := e.queue.EnqueueMlRerun(context.Background(), ml.SlotFaceRecognize)
	if err != nil {
		t.Fatalf("enqueue the face recognize rerun: %v", err)
	}
	eventually(t, jobWaitTimeout, "the recognition rerun to close succeeded", func() bool {
		row := e.jobRunOf(t, jobRunID)
		return row.state == "succeeded" && row.done >= 1 && row.finishedAt.Valid
	})

	// The recognition rerun dispatched recognition directly: exactly
	// the one face_detect_rerun run this test started exists -- the
	// recognition run spawned no detection detour of its own.
	if got := e.countJobRunKindRows(t, "face_detect_rerun"); got != 1 {
		t.Errorf("face_detect_rerun rows = %d, want 1 (the recognition rerun dispatches recognition directly, no detection detour)", got)
	}

	// The portrait's face carries the recognition model's embedding.
	faces := e.facesOf(t, id)
	if len(faces) != 1 {
		t.Fatalf("faces rows = %d, want 1 (the detection rerun's product)", len(faces))
	}
	if !faces[0].embeddingSet || !faces[0].embedModelID.Valid ||
		faces[0].embedModelID.String != recognizeModel {
		t.Errorf("face embedding = set %v by %v, want the recognize model %s's stamp",
			faces[0].embeddingSet, faces[0].embedModelID, recognizeModel)
	}

	// The river jobs: the detection rerun's face_detect job and the
	// recognition rerun's own face_recognize dispatch.
	kinds := e.riverJobsOf(t, id)
	if kinds["face_detect"] != 1 {
		t.Errorf("face_detect jobs = %d, want 1 (the detection rerun's dispatch)", kinds["face_detect"])
	}
	if kinds["face_recognize"] != 1 {
		t.Errorf("face_recognize jobs = %d, want 1 (the recognition rerun's direct dispatch; the detection chained nothing -- the slot was empty at its completion)", kinds["face_recognize"])
	}
}

// TestRerunEndpointJoinsInFlightRunThroughTheQueue pins the endpoint
// and the queue against each other: an in-flight run of the slot's
// kind answers 200 with its identifier through the real queue, and
// once nothing is in flight the endpoint starts a fresh run -- the
// api surface's kind vocabulary and the jobs package's agree.
func TestRerunEndpointJoinsInFlightRunThroughTheQueue(t *testing.T) {
	var failing uuid.UUID
	var mu sync.Mutex
	dispatched := map[uuid.UUID]int{}
	e := newEnvReportingDispatch(t, &dispatched, &mu, &failing)

	// One registered, active clip model: the rerun endpoint's subject.
	writeRerunFixtureModel(t, e.modelsDir, "rerun-clip", "clip-embed", "const512.onnx", rerunConst512SHA256, 512)
	if _, err := e.registry.Scan(e.modelsDir); err != nil {
		t.Fatalf("scan the models dir: %v", err)
	}
	if _, err := e.registry.Activate(ml.SlotClipEmbed, "rerun-clip"); err != nil {
		t.Fatalf("activate rerun-clip: %v", err)
	}

	inFlight := uuid.MustParse("0192f8f9-2ecf-9f0a-b1c2-3d4e5f6cddba")
	_, err := e.pool.Exec(context.Background(), `
		INSERT INTO job_runs (id, kind, total, state, created_at, updated_at)
		VALUES ($1, 'clip_embed_rerun', 0, 'running', now(), now())`, pgUUID(inFlight))
	if err != nil {
		t.Fatalf("insert the in-flight rerun row: %v", err)
	}

	rec, req := jsonRequest(http.MethodPost, "/api/v1/admin/models/rerun-clip/rerun", nil)
	req.AddCookie(e.cookie)
	e.router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("in-flight rerun POST status = %d, want %d (body %q)", rec.Code, http.StatusOK, rec.Body.String())
	}
	if got := e.countJobRunKindRows(t, "clip_embed_rerun"); got != 1 {
		t.Errorf("clip_embed_rerun rows = %d, want 1 (the in-flight run was returned as-is)", got)
	}

	// Nothing in flight anymore: the endpoint starts a fresh run.
	if _, err := e.pool.Exec(context.Background(),
		"UPDATE job_runs SET state = 'succeeded', finished_at = now() WHERE id = $1", pgUUID(inFlight)); err != nil {
		t.Fatalf("finish the in-flight rerun row: %v", err)
	}
	rec, req = jsonRequest(http.MethodPost, "/api/v1/admin/models/rerun-clip/rerun", nil)
	req.AddCookie(e.cookie)
	e.router.ServeHTTP(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("fresh rerun POST status = %d, want %d (body %q)", rec.Code, http.StatusAccepted, rec.Body.String())
	}
	var body struct {
		JobRunId string `json:"jobRunId"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode the rerun response %q: %v", rec.Body.String(), err)
	}
	created := uuid.MustParse(body.JobRunId)
	if created == inFlight {
		t.Fatal("the fresh rerun returned the finished run's id, want a new run")
	}
	eventually(t, jobWaitTimeout, "the fresh rerun to close (empty scope)", func() bool {
		return e.jobRunOf(t, created).state == "succeeded"
	})
	if got := e.countJobRunKindRows(t, "clip_embed_rerun"); got != 2 {
		t.Errorf("clip_embed_rerun rows = %d, want 2", got)
	}
}

// writeRerunFixtureModel lays out one fixture model directory under
// modelsDir the way real models are laid out.
func writeRerunFixtureModel(t *testing.T, modelsDir, id, slot, file, sha string, dim int) {
	t.Helper()
	dir := filepath.Join(modelsDir, id)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("create model dir %s: %v", dir, err)
	}
	data, err := os.ReadFile(filepath.Join("..", "ml", "testdata", file))
	if err != nil {
		t.Fatalf("read fixture %s: %v", file, err)
	}
	if err := os.WriteFile(filepath.Join(dir, file), data, 0o644); err != nil {
		t.Fatalf("write model weights: %v", err)
	}
	manifest := ml.ManifestModel{
		ID:        id,
		Slot:      slot,
		Version:   "fixture",
		License:   "MIT",
		InputSize: ml.ManifestSize{Width: 8, Height: 8},
		Dim:       dim,
		Files:     []ml.ManifestFile{{Name: file, SHA256: sha, Bytes: int64(len(data))}},
	}
	blob, err := json.Marshal(manifest)
	if err != nil {
		t.Fatalf("marshal fixture manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), blob, 0o644); err != nil {
		t.Fatalf("write fixture manifest: %v", err)
	}
}
