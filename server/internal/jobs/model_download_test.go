// The model download job tests (M3a Task 3): the full path runs -- the
// admin endpoint enqueues the run, River picks the job up, the worker
// drives the ml download service against local httptest sources (no
// real network), the job_runs row converges, and the job.* events land
// on the bus. The pins are:
//
//   - the worker idempotence: a second run re-downloads nothing that
//     already verifies on disk;
//   - the failure accounting: a model whose source serves tampered
//     bytes fails the run (state=failed, the failed count in the row,
//     the model never registered) while the good models still land;
//   - the event payload shape and the 1/s throttle: job.started and
//     job.finished fire exactly once per attempt with the six-field
//     camelCase payload, and job.progress stays bounded even though
//     the 2 MiB fixture emits a chunk-level attempt per 64 KiB.
package jobs

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	apigen "github.com/welpher/lumemor/server/internal/api/gen"
	"github.com/welpher/lumemor/server/internal/events"
	"github.com/welpher/lumemor/server/internal/ml"
)

// modelDownloadFile is one fixture artifact of a test model.
type modelDownloadFile struct {
	name    string
	content []byte
}

// modelDownloadModel describes one fixture model of the download job's
// manifest: the slot it fills, the URLs that anchor its files, and the
// files themselves.
type modelDownloadModel struct {
	id       string
	slot     string
	dim      int
	source   string
	fallback string
	files    []modelDownloadFile
}

// modelDownloadGroup is the four-model fixture group: one model per
// slot, the OCR model carrying two files, and a 2 MiB clip model whose
// chunk-level progress attempts give the throttle something to
// collapse.
func modelDownloadGroup() []modelDownloadModel {
	return []modelDownloadModel{
		{
			id:   "face-a",
			slot: "face-detect",
			files: []modelDownloadFile{
				{name: "face.onnx", content: bytes.Repeat([]byte{0x0a}, 128*1024)},
			},
		},
		{
			id:   "face-b",
			slot: "face-recognize",
			dim:  128,
			files: []modelDownloadFile{
				{name: "recognize.onnx", content: bytes.Repeat([]byte{0x0b}, 192*1024)},
			},
		},
		{
			id:   "clip-a",
			slot: "clip-embed",
			dim:  512,
			files: []modelDownloadFile{
				{name: "vision.onnx", content: bytes.Repeat([]byte{0x0c}, 2*1024*1024)},
			},
		},
		{
			id:   "ocr-a",
			slot: "ocr",
			files: []modelDownloadFile{
				{name: "det.onnx", content: bytes.Repeat([]byte{0x0d}, 96*1024)},
				{name: "rec.onnx", content: bytes.Repeat([]byte{0x0e}, 160*1024)},
			},
		},
	}
}

// modelDownloadManifest builds the manifest of the models, pinning
// every file's sha256 and byte count from its content. The files keep
// their slice order: the first file is the model's primary weights
// file.
func modelDownloadManifest(models ...modelDownloadModel) *ml.Manifest {
	out := &ml.Manifest{Revision: 1}
	for _, m := range models {
		mm := ml.ManifestModel{
			ID:          m.id,
			Slot:        m.slot,
			Version:     "fixture",
			License:     "MIT",
			SourceURL:   m.source,
			InputSize:   ml.ManifestSize{Width: 8, Height: 8},
			Dim:         m.dim,
			Params:      map[string]any{"scoreThreshold": 0.6},
			FallbackURL: nil,
		}
		if m.fallback != "" {
			mm.FallbackURL = &m.fallback
		}
		for _, f := range m.files {
			digest := sha256.Sum256(f.content)
			mm.Files = append(mm.Files, ml.ManifestFile{
				Name:   f.name,
				SHA256: hex.EncodeToString(digest[:]),
				Bytes:  int64(len(f.content)),
			})
		}
		out.Models = append(out.Models, mm)
	}
	return out
}

// modelDownloadServing flattens the models' files into a source
// server's serving map, keyed by the path the models' URLs anchor at.
func modelDownloadServing(models ...modelDownloadModel) map[string][]byte {
	serving := map[string][]byte{}
	for _, m := range models {
		for _, f := range m.files {
			serving["/"+m.id+"/"+f.name] = f.content
		}
	}
	return serving
}

// modelDownloadAnchored returns the models with their manifest URLs
// anchored at the given sources.
func modelDownloadAnchored(models []modelDownloadModel, primary, fallback string) []modelDownloadModel {
	out := make([]modelDownloadModel, len(models))
	for i, m := range models {
		m.source = primary + "/" + m.id + "/" + m.files[0].name
		if fallback != "" {
			m.fallback = fallback + "/" + m.id + "/" + m.files[0].name
		}
		out[i] = m
	}
	return out
}

// modelDownloadSource is a local file server standing in for an
// upstream source: it serves the fixture files by path and counts the
// requests it served.
type modelDownloadSource struct {
	*httptest.Server

	mu      sync.Mutex
	hits    int64
	serving map[string][]byte
}

func newModelDownloadSource(t *testing.T, serving map[string][]byte) *modelDownloadSource {
	t.Helper()
	s := &modelDownloadSource{serving: serving}
	s.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		s.hits++
		s.mu.Unlock()
		content, ok := s.serving[r.URL.Path]
		if !ok {
			http.Error(w, "no such fixture file", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/octet-stream")
		_, _ = w.Write(content)
	}))
	t.Cleanup(s.Close)
	return s
}

func (s *modelDownloadSource) hitCount() int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.hits
}

// jobRunRow is the job_runs row as the assertions read it.
type jobRunRow struct {
	kind       string
	total      int32
	done       int32
	failed     int32
	state      string
	startedAt  pgtype.Timestamptz
	finishedAt pgtype.Timestamptz
}

// jobRunOf loads the job_runs row of the run.
func (e *env) jobRunOf(t *testing.T, id uuid.UUID) jobRunRow {
	t.Helper()
	var row jobRunRow
	err := e.pool.QueryRow(context.Background(),
		"SELECT kind, total, done, failed, state, started_at, finished_at FROM job_runs WHERE id = $1", pgUUID(id)).
		Scan(&row.kind, &row.total, &row.done, &row.failed, &row.state, &row.startedAt, &row.finishedAt)
	if err != nil {
		t.Fatalf("scan the job_runs row %s: %v", id, err)
	}
	return row
}

// jobRunCount counts the job_runs rows of the model download kind.
func (e *env) jobRunCount(t *testing.T) int {
	t.Helper()
	var n int
	if err := e.pool.QueryRow(context.Background(),
		"SELECT count(*) FROM job_runs WHERE kind = 'model_download'").Scan(&n); err != nil {
		t.Fatalf("count job_runs rows: %v", err)
	}
	return n
}

// postModelDownload fires the download trigger with the environment's
// administrator session.
func (e *env) postModelDownload(t *testing.T) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/models/download", nil)
	req.AddCookie(e.cookie)
	rec := httptest.NewRecorder()
	e.router.ServeHTTP(rec, req)
	return rec
}

// assertModelLanded pins one model's landing through the job: the
// files on disk match their pinned bytes, the manifest copy sits next
// to them, and the models row registers as source 'downloaded' and
// active in its slot.
func (e *env) assertModelLanded(t *testing.T, m modelDownloadModel) {
	t.Helper()
	for _, f := range m.files {
		data, err := os.ReadFile(filepath.Join(e.modelsDir, m.id, f.name))
		if err != nil {
			t.Fatalf("read the landed file of %s: %v", m.id, err)
		}
		if !bytes.Equal(data, f.content) {
			t.Errorf("the landed file %s/%s does not match its pinned content", m.id, f.name)
		}
	}
	if _, err := os.Stat(filepath.Join(e.modelsDir, m.id, "manifest.json")); err != nil {
		t.Fatalf("the manifest copy of %s is missing: %v", m.id, err)
	}
	var source, filePath string
	var isActive bool
	err := e.pool.QueryRow(context.Background(),
		"SELECT source, file_path, is_active FROM models WHERE id = $1", m.id).
		Scan(&source, &filePath, &isActive)
	if err != nil {
		t.Fatalf("model %s is not registered: %v", m.id, err)
	}
	if source != "downloaded" {
		t.Errorf("model %s source = %q, want %q", m.id, source, "downloaded")
	}
	if want := m.id + "/" + m.files[0].name; filePath != want {
		t.Errorf("model %s file_path = %q, want %q", m.id, filePath, want)
	}
	if !isActive {
		t.Errorf("model %s is_active = false, want the empty slot auto-activated", m.id)
	}
}

// waitForBusEvent polls the bus's retained ring until an event of the
// name whose payload satisfies pred appears, and returns that payload.
func waitForBusEvent(t *testing.T, bus *events.Bus, name string, pred func(map[string]string) bool) map[string]string {
	t.Helper()
	deadline := time.Now().Add(jobWaitTimeout)
	for time.Now().Before(deadline) {
		ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		subscription := bus.Subscribe(ctx, 0)
		var found map[string]string
		for {
			event, ok := subscription.Next()
			if !ok {
				break
			}
			if event.Name != name {
				continue
			}
			var payload map[string]string
			if err := json.Unmarshal(event.Payload, &payload); err != nil {
				cancel()
				t.Fatalf("decode the %s payload %s: %v", name, event.Payload, err)
			}
			if pred(payload) {
				found = payload
				break
			}
		}
		cancel()
		if found != nil {
			return found
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for a %s event satisfying the predicate", name)
	return nil
}

// everyBusEvent replays the retained events of the name and hands each
// payload to check.
func everyBusEvent(t *testing.T, bus *events.Bus, name string, check func(map[string]string)) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	subscription := bus.Subscribe(ctx, 0)
	for {
		event, ok := subscription.Next()
		if !ok {
			return
		}
		if event.Name != name {
			continue
		}
		var payload map[string]string
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			t.Fatalf("decode the %s payload %s: %v", name, event.Payload, err)
		}
		check(payload)
	}
}

// jobEventKeys is the six-field vocabulary of the job.* event
// payloads, matching the asset events' camelCase discipline.
var jobEventKeys = []string{"jobRunId", "kind", "total", "done", "failed", "state"}

// TestModelDownloadJobRunsToCompletion pins the full path: the first
// POST answers 202 and starts the run, the row converges to succeeded
// with the full pinned byte progress, the four models land, register
// and auto-activate, the events carry the six-field payload with the
// started/finished pair exactly once, the progress stays bounded by
// the throttle, and a second POST answers 200 with the same run
// without enqueueing anything new.
func TestModelDownloadJobRunsToCompletion(t *testing.T) {
	group := modelDownloadGroup()
	primary := newModelDownloadSource(t, modelDownloadServing(group...))
	fallback := newModelDownloadSource(t, modelDownloadServing(group...))
	manifest := modelDownloadManifest(modelDownloadAnchored(group, primary.URL, fallback.URL)...)
	e := newEnvManifest(t, manifest)

	rec := e.postModelDownload(t)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("POST /admin/models/download status = %d, want %d (body %q)", rec.Code, http.StatusAccepted, rec.Body.String())
	}
	var first apigen.ModelDownloadResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &first); err != nil {
		t.Fatalf("decode the download response %q: %v", rec.Body.String(), err)
	}
	jobRunID := first.JobRunId
	if jobRunID == uuid.Nil {
		t.Fatalf("the download response carries no jobRunId")
	}

	pinnedTotal := int32(0)
	for _, m := range manifest.Models {
		for _, f := range m.Files {
			pinnedTotal += int32(f.Bytes)
		}
	}

	eventually(t, jobWaitTimeout, "the model download run to succeed", func() bool {
		row := e.jobRunOf(t, jobRunID)
		return row.kind == "model_download" && row.state == "succeeded" &&
			row.done == row.total && row.total == pinnedTotal && row.failed == 0 && row.finishedAt.Valid
	})

	for _, m := range group {
		e.assertModelLanded(t, m)
	}

	// The started event: exactly one, announcing the run with zero
	// progress.
	if got := countEvents(e.bus, "job.started"); got != 1 {
		t.Errorf("job.started events = %d, want exactly 1", got)
	}
	started := eventPayloadOf(t, e.bus, "job.started")
	if started["jobRunId"] != jobRunID.String() {
		t.Errorf("job.started jobRunId = %q, want %q", started["jobRunId"], jobRunID.String())
	}
	if started["kind"] != "model_download" {
		t.Errorf("job.started kind = %q, want model_download", started["kind"])
	}
	if started["done"] != "0" || started["failed"] != "0" || started["state"] != "running" {
		t.Errorf("job.started = done %q, failed %q, state %q, want 0, 0, running", started["done"], started["failed"], started["state"])
	}
	if started["total"] != strconv.Itoa(int(pinnedTotal)) {
		t.Errorf("job.started total = %q, want %d", started["total"], pinnedTotal)
	}

	// The finished event: exactly one, carrying the converged row.
	finished := waitForBusEvent(t, e.bus, "job.finished", func(p map[string]string) bool {
		return p["state"] == "succeeded"
	})
	if finished["jobRunId"] != jobRunID.String() {
		t.Errorf("job.finished jobRunId = %q, want %q", finished["jobRunId"], jobRunID.String())
	}
	if finished["done"] != strconv.Itoa(int(pinnedTotal)) || finished["failed"] != "0" {
		t.Errorf("job.finished = done %q, failed %q, want %d, 0", finished["done"], finished["failed"], pinnedTotal)
	}

	// The progress events: bounded by the throttle (the 2 MiB fixture
	// alone yields 32 chunk-level attempts) and every payload carries
	// the full six-field vocabulary.
	progressCount := countEvents(e.bus, "job.progress")
	if progressCount > 8 {
		t.Errorf("job.progress events = %d, want at most 8 (the throttle must collapse the chunk attempts)", progressCount)
	}
	everyBusEvent(t, e.bus, "job.progress", func(p map[string]string) {
		for _, key := range jobEventKeys {
			if _, ok := p[key]; !ok {
				t.Errorf("a job.progress payload misses the %q field: %v", key, p)
			}
		}
		if p["jobRunId"] != jobRunID.String() {
			t.Errorf("a job.progress payload jobRunId = %q, want %q", p["jobRunId"], jobRunID.String())
		}
	})

	// The fallback stayed untouched: the primary served everything.
	if got := fallback.hitCount(); got != 0 {
		t.Errorf("fallback source requests = %d, want 0", got)
	}

	// A second POST is idempotent: the most recent run answers 200
	// with the same identifier, and nothing new is enqueued or
	// fetched.
	second := e.postModelDownload(t)
	if second.Code != http.StatusOK {
		t.Fatalf("second POST status = %d, want %d (body %q)", second.Code, http.StatusOK, second.Body.String())
	}
	var secondBody apigen.ModelDownloadResponse
	if err := json.Unmarshal(second.Body.Bytes(), &secondBody); err != nil {
		t.Fatalf("decode the second download response %q: %v", second.Body.String(), err)
	}
	if secondBody.JobRunId != jobRunID {
		t.Errorf("second POST jobRunId = %q, want the same %q", secondBody.JobRunId, jobRunID)
	}
	if got := e.jobRunCount(t); got != 1 {
		t.Errorf("job_runs rows = %d, want 1 (the idempotent return enqueues nothing)", got)
	}
}

// TestModelDownloadJobSkipsVerifiedFilesOnRerun pins the worker's
// idempotence: a second run enqueued directly at the queue level (the
// API guard keeps a second POST from reaching this point) re-downloads
// nothing -- every file already verifies on disk -- and still
// converges to succeeded.
func TestModelDownloadJobSkipsVerifiedFilesOnRerun(t *testing.T) {
	group := modelDownloadGroup()
	primary := newModelDownloadSource(t, modelDownloadServing(group...))
	manifest := modelDownloadManifest(modelDownloadAnchored(group, primary.URL, "")...)
	e := newEnvManifest(t, manifest)

	rec := e.postModelDownload(t)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("POST /admin/models/download status = %d, want %d (body %q)", rec.Code, http.StatusAccepted, rec.Body.String())
	}
	var first apigen.ModelDownloadResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &first); err != nil {
		t.Fatalf("decode the download response %q: %v", rec.Body.String(), err)
	}
	eventually(t, jobWaitTimeout, "the first model download run to succeed", func() bool {
		return e.jobRunOf(t, first.JobRunId).state == "succeeded"
	})
	hits := primary.hitCount()

	// The rerun, enqueued at the queue level like a recovery would.
	secondID, err := e.queue.EnqueueModelDownload(context.Background())
	if err != nil {
		t.Fatalf("EnqueueModelDownload: %v", err)
	}
	eventually(t, jobWaitTimeout, "the second model download run to succeed", func() bool {
		return e.jobRunOf(t, secondID).state == "succeeded"
	})

	if got := primary.hitCount(); got != hits {
		t.Errorf("source requests after the rerun = %d, want %d (verified files are skipped)", got, hits)
	}
	if got := e.jobRunCount(t); got != 2 {
		t.Errorf("job_runs rows = %d, want 2 (one per enqueued run)", got)
	}
	row := e.jobRunOf(t, secondID)
	if row.done != row.total || row.failed != 0 {
		t.Errorf("rerun row = %d done of %d, %d failed, want the full pinned progress", row.done, row.total, row.failed)
	}
}

// TestModelDownloadJobCountsFailures pins the failure accounting: a
// source serving bytes that drift from the pinned digest fails its
// model (never registered, its slot stays empty) while the good models
// of the run still land; the job_runs row flips to failed carrying the
// done and failed counts, and the finished event reports the same.
func TestModelDownloadJobCountsFailures(t *testing.T) {
	group := modelDownloadGroup()
	// The manifest pins the true digests; the source serves tampered
	// bytes for the OCR model's first file.
	serving := modelDownloadServing(group...)
	tampered := append([]byte{}, serving["/ocr-a/det.onnx"]...)
	tampered[len(tampered)/2] ^= 0xff
	serving["/ocr-a/det.onnx"] = tampered
	primary := newModelDownloadSource(t, serving)
	manifest := modelDownloadManifest(modelDownloadAnchored(group, primary.URL, "")...)
	e := newEnvManifest(t, manifest)

	rec := e.postModelDownload(t)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("POST /admin/models/download status = %d, want %d (body %q)", rec.Code, http.StatusAccepted, rec.Body.String())
	}
	var first apigen.ModelDownloadResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &first); err != nil {
		t.Fatalf("decode the download response %q: %v", rec.Body.String(), err)
	}
	jobRunID := first.JobRunId

	goodBytes := int32(0)
	for _, m := range group {
		if m.id == "ocr-a" {
			continue
		}
		for _, f := range m.files {
			goodBytes += int32(len(f.content))
		}
	}

	// The finished event of the failed attempt carries the counts (the
	// River retry re-runs and re-fails the tampered model, so the row
	// flips between running and failed; the event and the row read at
	// the moment of the first failure are the stable assertions).
	finished := waitForBusEvent(t, e.bus, "job.finished", func(p map[string]string) bool {
		return p["state"] == "failed"
	})
	if finished["jobRunId"] != jobRunID.String() {
		t.Errorf("job.finished jobRunId = %q, want %q", finished["jobRunId"], jobRunID.String())
	}
	if finished["failed"] != "1" {
		t.Errorf("job.finished failed = %q, want 1", finished["failed"])
	}
	if finished["done"] != strconv.Itoa(int(goodBytes)) {
		t.Errorf("job.finished done = %q, want %d (the good models' pinned bytes)", finished["done"], goodBytes)
	}

	var failedRow jobRunRow
	eventually(t, jobWaitTimeout, "the model download run to fail", func() bool {
		failedRow = e.jobRunOf(t, jobRunID)
		return failedRow.state == "failed"
	})
	if failedRow.kind != "model_download" || failedRow.failed != 1 {
		t.Errorf("failed row = kind %q, %d failed, want model_download, 1", failedRow.kind, failedRow.failed)
	}
	if failedRow.done != goodBytes {
		t.Errorf("failed row done = %d, want %d (the good models' pinned bytes)", failedRow.done, goodBytes)
	}

	// The good models landed; the tampered one never registered.
	for _, m := range group {
		if m.id == "ocr-a" {
			continue
		}
		e.assertModelLanded(t, m)
	}
	var n int
	if err := e.pool.QueryRow(context.Background(),
		"SELECT count(*) FROM models WHERE id = 'ocr-a'").Scan(&n); err != nil {
		t.Fatalf("count the tampered model's rows: %v", err)
	}
	if n != 0 {
		t.Errorf("the tampered model is registered (%d rows, want 0)", n)
	}
}
