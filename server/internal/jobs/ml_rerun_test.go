// The ML rerun orchestration tests (M3a Task 4): the rerun framework
// this task delivers -- the job_runs row's lifecycle (created with the
// asset count as its total, flipped to running when the driver picks
// it up, closed with the converged counts), the per-asset dispatch
// seam and its completion counting, the job.* events, and the rerun
// kinds' vocabulary. The honest boundary: the per-asset ML workers
// themselves (face detection, recognition, CLIP embedding, OCR)
// arrive with the milestone's later tasks; these tests drive a fake
// dispatcher, and a run started without any dispatcher wired fails
// cleanly -- the placeholder the later tasks replace.
package jobs

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	dbgen "github.com/welpher/lumemor/server/internal/db/gen"
	"github.com/welpher/lumemor/server/internal/ml"
)

// Pinned digests of the committed ml fixtures the rerun endpoint test
// copies into its throwaway model directory.
const (
	rerunConst512SHA256 = "92d6b93b89bf6ccec24222be86689b6ff18370503d3d71316d5d325bd2116766"
)

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

// TestMlRerunRunLifecycleAndCounting pins the framework's main path:
// the run row is created with the image asset count as its total
// (videos, still-pipelined and trashed assets are out of scope), the
// driver flips it to running, dispatches every asset through the seam,
// counts the completions (successes into done, failures into failed),
// closes the row with a failed state when any completion failed, and
// announces the run through the job.* events.
func TestMlRerunRunLifecycleAndCounting(t *testing.T) {
	// The failing asset id is only known after the env exists; the
	// closure reads it at dispatch time, when it is long set.
	var failing uuid.UUID
	var mu sync.Mutex
	dispatched := map[uuid.UUID]int{}
	dispatch := func(ctx context.Context, slot ml.Slot, assetID uuid.UUID) error {
		mu.Lock()
		defer mu.Unlock()
		dispatched[assetID]++
		if assetID == failing {
			return errors.New("injected per-asset failure")
		}
		return nil
	}
	e := newEnvRerun(t, dispatch)

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
		t.Errorf("job.progress events = %d, want at most 3 (the throttle collapses the fast dispatches)", got)
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
// every dispatch succeeding closes the run as succeeded with the
// counts converged and no failure anywhere.
func TestMlRerunAllAssetsSucceedFlipsSucceeded(t *testing.T) {
	e := newEnvRerun(t, func(ctx context.Context, slot ml.Slot, assetID uuid.UUID) error {
		return nil
	})
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
	e := newEnv(t) // no assets, no dispatcher wired

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
// boundary: with no per-asset dispatcher wired (the production wiring
// of this task -- the ML workers arrive with the milestone's later
// tasks), a rerun of a non-empty scope fails cleanly instead of
// pretending to have run.
func TestMlRerunWithoutDispatchFailsTheRun(t *testing.T) {
	e := newEnv(t) // no dispatcher wired
	e.insertReadyImage(t, "originals/rerun-placeholder/one.jpg")
	e.insertReadyImage(t, "originals/rerun-placeholder/two.jpg")

	jobRunID, err := e.queue.EnqueueMlRerun(context.Background(), ml.SlotFaceDetect)
	if err != nil {
		t.Fatalf("enqueue the face detect rerun: %v", err)
	}
	eventually(t, jobWaitTimeout, "the undisposable rerun to close failed", func() bool {
		row := e.jobRunOf(t, jobRunID)
		return row.state == "failed"
	})
	row := e.jobRunOf(t, jobRunID)
	if row.done != 0 || row.failed != 0 {
		t.Errorf("row done/failed = %d/%d, want zeros (nothing was dispatched)", row.done, row.failed)
	}
	finished := eventPayloadOf(t, e.bus, "job.finished")
	if finished["state"] != "failed" {
		t.Errorf("job.finished state = %q, want failed", finished["state"])
	}
}

// TestRerunEndpointJoinsInFlightRunThroughTheQueue pins the endpoint
// and the queue against each other: an in-flight run of the slot's
// kind answers 200 with its identifier through the real queue, and
// once nothing is in flight the endpoint starts a fresh run -- the
// api surface's kind vocabulary and the jobs package's agree.
func TestRerunEndpointJoinsInFlightRunThroughTheQueue(t *testing.T) {
	e := newEnvRerun(t, func(ctx context.Context, slot ml.Slot, assetID uuid.UUID) error {
		return nil
	})

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
