// The model download service tests: every behavior runs
// against local httptest file servers standing in for the upstream
// sources -- no real network is ever touched -- with a migrated
// throwaway database behind the real registry. The pins are:
//
//   - the landing contract: every file is fetched, checksummed against
//     its pinned digest, atomically renamed into <modelsDir>/<id>/,
//     joined by the per-model manifest copy, registered as source
//     'downloaded', and auto-activated into its empty slot through the
//     real registry;
//   - the source chain: a failing primary falls back to the model's
//     secondary URL, and a model without a fallback simply fails;
//   - the interrupted-retry convergence: a stale partial file is
//     replaced, never trusted;
//   - the checksum gate: tampered upstream bytes never register;
//   - the HF domain mirror rewrite: huggingface.co hostnames are
//     rewritten wholesale, every other host stays untouched;
//   - the already-verified skip: a second run re-downloads nothing;
//   - the progress throttle: dense per-chunk updates collapse to at
//     most one progress callback per second, plus the mandatory first
//     and final ones.
package ml

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/welpher/lumemor/server/internal/db"
	"github.com/welpher/lumemor/server/internal/db/dbtest"
	dbgen "github.com/welpher/lumemor/server/internal/db/gen"
)

// downloadEnv is the download service's test environment: a migrated
// throwaway database, a real registry over a fresh models dir, and the
// queries the download service registers its models through.
type downloadEnv struct {
	pool      *pgxpool.Pool
	queries   *dbgen.Queries
	registry  *Registry
	modelsDir string
}

func newDownloadEnv(t *testing.T) *downloadEnv {
	t.Helper()
	pool := dbtest.New(t)
	if err := db.Migrate(t.Context(), pool); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
	modelsDir := t.TempDir()
	return &downloadEnv{
		pool:      pool,
		queries:   dbgen.New(pool),
		registry:  NewRegistry(pool, dbgen.New(pool), modelsDir),
		modelsDir: modelsDir,
	}
}

// downloadFile is one fixture artifact of a download model.
type downloadFile struct {
	name    string
	content []byte
}

// downloadModel describes one fixture model: the slot it fills, the
// URLs that anchor its files, and the files themselves.
type downloadModel struct {
	id       string
	slot     string
	dim      int
	source   string
	fallback string
	files    []downloadFile
}

// downloadManifest builds the manifest of the models, pinning every
// file's sha256 and byte count from its content. The files keep their
// slice order: the first file is the model's primary weights file.
func downloadManifest(models ...downloadModel) *Manifest {
	out := &Manifest{Revision: 1}
	for _, m := range models {
		mm := ManifestModel{
			ID:        m.id,
			Slot:      m.slot,
			Version:   "fixture",
			License:   "MIT",
			SourceURL: m.source,
			InputSize: ManifestSize{Width: 8, Height: 8},
			Dim:       m.dim,
			Params:    map[string]any{"scoreThreshold": 0.6},
		}
		if m.fallback != "" {
			mm.FallbackURL = &m.fallback
		}
		for _, f := range m.files {
			digest := sha256.Sum256(f.content)
			mm.Files = append(mm.Files, ManifestFile{
				Name:   f.name,
				SHA256: hex.EncodeToString(digest[:]),
				Bytes:  int64(len(f.content)),
			})
		}
		out.Models = append(out.Models, mm)
	}
	return out
}

// sourceFiles flattens the models' files into a source server's
// serving map, keyed by the path the models' URLs anchor at.
func sourceFiles(models ...downloadModel) map[string][]byte {
	serving := map[string][]byte{}
	for _, m := range models {
		for _, f := range m.files {
			serving["/"+m.id+"/"+f.name] = f.content
		}
	}
	return serving
}

// anchored returns the models with their manifest URLs anchored at the
// given sources: the source and fallback URLs point at each model's
// primary file, the same directory the remaining files derive from.
func anchored(models []downloadModel, primary, fallback string) []downloadModel {
	out := make([]downloadModel, len(models))
	for i, m := range models {
		m.source = primary + "/" + m.id + "/" + m.files[0].name
		if fallback != "" {
			m.fallback = fallback + "/" + m.id + "/" + m.files[0].name
		}
		out[i] = m
	}
	return out
}

// downloadSource is a local file server standing in for an upstream
// source: it serves the fixture files by path and counts the requests
// it served.
type downloadSource struct {
	*httptest.Server

	mu      sync.Mutex
	hits    int64
	serving map[string][]byte
}

// handler is the serving logic the plain and the TLS variants share.
func (s *downloadSource) handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	})
}

func newDownloadSource(t *testing.T, serving map[string][]byte) *downloadSource {
	t.Helper()
	s := &downloadSource{serving: serving}
	s.Server = httptest.NewServer(s.handler())
	t.Cleanup(s.Close)
	return s
}

// newTLSDownloadSource is the HTTPS variant: a rewritten huggingface.co
// URL keeps the https scheme, so the mirror test's source must speak
// TLS and its client must trust its certificate.
func newTLSDownloadSource(t *testing.T, serving map[string][]byte) *downloadSource {
	t.Helper()
	s := &downloadSource{serving: serving}
	s.Server = httptest.NewTLSServer(s.handler())
	t.Cleanup(s.Close)
	return s
}

func (s *downloadSource) hitCount() int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.hits
}

// progressPoint is one progress callback invocation.
type progressPoint struct {
	done  int
	total int
}

// runDownload runs one download over the environment and records the
// progress callbacks.
func (e *downloadEnv) runDownload(t *testing.T, manifest *Manifest, client *http.Client, mirror string, now func() time.Time) (DownloadReport, []progressPoint, error) {
	t.Helper()
	var points []progressPoint
	report, err := RunDownload(t.Context(), RunDownloadDeps{
		Manifest:  manifest,
		ModelsDir: e.modelsDir,
		Client:    client,
		Registry:  e.registry,
		Queries:   e.queries,
		HFMirror:  mirror,
		Now:       now,
	}, func(done, total int) {
		points = append(points, progressPoint{done: done, total: total})
	})
	return report, points, err
}

// modelRow reads one models row straight from the table: the tests pin
// registration facts where they live.
type modelRow struct {
	id       string
	slot     string
	source   string
	filePath string
	isActive bool
}

func (e *downloadEnv) modelRowOf(t *testing.T, id string) (modelRow, bool) {
	t.Helper()
	var row modelRow
	err := e.pool.QueryRow(t.Context(),
		"SELECT id, slot, source, file_path, is_active FROM models WHERE id = $1", id).
		Scan(&row.id, &row.slot, &row.source, &row.filePath, &row.isActive)
	if errors.Is(err, pgx.ErrNoRows) {
		return row, false
	}
	if err != nil {
		t.Fatalf("scan models row %s: %v", id, err)
	}
	return row, true
}

func (e *downloadEnv) modelCount(t *testing.T) int {
	t.Helper()
	var n int
	if err := e.pool.QueryRow(t.Context(), "SELECT count(*) FROM models").Scan(&n); err != nil {
		t.Fatalf("count models: %v", err)
	}
	return n
}

// assertModelLanded pins the full landing contract of one model: the
// files on disk match their pinned bytes, the manifest copy sits next
// to them, the models row registers as source 'downloaded', and the
// slot's active model resolves to it through the real registry.
func (e *downloadEnv) assertModelLanded(t *testing.T, m downloadModel) {
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
	blob, err := os.ReadFile(filepath.Join(e.modelsDir, m.id, manifestFileName))
	if err != nil {
		t.Fatalf("read the manifest copy of %s: %v", m.id, err)
	}
	var copy ManifestModel
	if err := json.Unmarshal(blob, &copy); err != nil {
		t.Fatalf("parse the manifest copy of %s: %v", m.id, err)
	}
	if copy.ID != m.id {
		t.Errorf("the manifest copy of %s declares id %q", m.id, copy.ID)
	}
	row, ok := e.modelRowOf(t, m.id)
	if !ok {
		t.Fatalf("model %s is not registered", m.id)
	}
	if row.source != "downloaded" {
		t.Errorf("model %s source = %q, want %q", m.id, row.source, "downloaded")
	}
	if want := m.id + "/" + m.files[0].name; row.filePath != want {
		t.Errorf("model %s file_path = %q, want %q", m.id, row.filePath, want)
	}
	if !row.isActive {
		t.Errorf("model %s is_active = false, want the empty slot auto-activated", m.id)
	}
	slot := manifestSlotValues[m.slot]
	active, err := e.registry.Active(slot)
	if err != nil {
		t.Fatalf("the active model of slot %s: %v", slot, err)
	}
	if active.ID != m.id {
		t.Errorf("the active model of slot %s = %q, want %q", slot, active.ID, m.id)
	}
}

// downloadGroup is the four-model fixture group: one model per slot,
// the OCR model carrying two files, mirroring the default group's
// shape.
func downloadGroup() []downloadModel {
	return []downloadModel{
		{
			id:   "face-a",
			slot: "face-detect",
			files: []downloadFile{
				{name: "face.onnx", content: bytes.Repeat([]byte{0x0a}, 128*1024)},
			},
		},
		{
			id:   "face-b",
			slot: "face-recognize",
			dim:  128,
			files: []downloadFile{
				{name: "recognize.onnx", content: bytes.Repeat([]byte{0x0b}, 192*1024)},
			},
		},
		{
			id:   "clip-a",
			slot: "clip-embed",
			dim:  512,
			files: []downloadFile{
				{name: "vision.onnx", content: bytes.Repeat([]byte{0x0c}, 64*1024)},
			},
		},
		{
			id:   "ocr-a",
			slot: "ocr",
			files: []downloadFile{
				{name: "det.onnx", content: bytes.Repeat([]byte{0x0d}, 96*1024)},
				{name: "rec.onnx", content: bytes.Repeat([]byte{0x0e}, 160*1024)},
			},
		},
	}
}

// TestRunDownloadLandsTheGroup pins the happy path: the whole group
// lands from the primary sources (the fallbacks stay untouched), every
// model registers with source 'downloaded' and auto-activates into its
// empty slot, and the progress starts at zero, ends at the pinned
// total and never moves backwards.
func TestRunDownloadLandsTheGroup(t *testing.T) {
	e := newDownloadEnv(t)
	group := downloadGroup()
	primary := newDownloadSource(t, sourceFiles(group...))
	fallback := newDownloadSource(t, sourceFiles(group...))
	manifest := downloadManifest(anchored(group, primary.URL, fallback.URL)...)

	report, points, err := e.runDownload(t, manifest, http.DefaultClient, "", nil)
	if err != nil {
		t.Fatalf("RunDownload error = %v, want nil", err)
	}
	if report.ModelsDone != 4 || report.ModelsFailed != 0 {
		t.Errorf("report = %d done, %d failed, want 4 done, 0 failed", report.ModelsDone, report.ModelsFailed)
	}
	if report.BytesDone != report.BytesTotal || report.BytesTotal == 0 {
		t.Errorf("report bytes = %d done of %d total, want equal and non-zero", report.BytesDone, report.BytesTotal)
	}
	for _, m := range group {
		e.assertModelLanded(t, m)
	}
	if got := e.modelCount(t); got != 4 {
		t.Errorf("models rows = %d, want 4", got)
	}
	if got := fallback.hitCount(); got != 0 {
		t.Errorf("fallback source requests = %d, want 0 (the primary served everything)", got)
	}
	if len(points) == 0 {
		t.Fatalf("no progress callbacks recorded")
	}
	if points[0].done != 0 || points[0].total != report.BytesTotal {
		t.Errorf("first progress = %d/%d, want 0/%d", points[0].done, points[0].total, report.BytesTotal)
	}
	last := points[len(points)-1]
	if last.done != last.total || last.total != report.BytesTotal {
		t.Errorf("last progress = %d/%d, want %d/%d", last.done, last.total, report.BytesTotal, report.BytesTotal)
	}
	for i := 1; i < len(points); i++ {
		if points[i].done < points[i-1].done {
			t.Errorf("progress went backwards: %d then %d", points[i-1].done, points[i].done)
		}
	}
}

// TestRunDownloadFallsBackWhenPrimaryFails pins the source chain: a
// model whose primary fails lands through its fallback URL, and a
// model without a fallback simply fails (unregistered, its slot stays
// empty) without failing the other models of the run.
func TestRunDownloadFallsBackWhenPrimaryFails(t *testing.T) {
	e := newDownloadEnv(t)
	group := []downloadModel{
		{
			id:   "face-a",
			slot: "face-detect",
			files: []downloadFile{
				{name: "face.onnx", content: bytes.Repeat([]byte{0x0a}, 64*1024)},
			},
		},
		{
			id:   "clip-a",
			slot: "clip-embed",
			dim:  512,
			files: []downloadFile{
				{name: "vision.onnx", content: bytes.Repeat([]byte{0x0c}, 64*1024)},
			},
		},
	}
	primary := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "upstream unavailable", http.StatusServiceUnavailable)
	}))
	t.Cleanup(primary.Close)
	fallback := newDownloadSource(t, sourceFiles(group...))

	// face-a carries the fallback; clip-a has none.
	anchoredModels := anchored(group, primary.URL, fallback.URL)
	anchoredModels[1].fallback = ""
	manifest := downloadManifest(anchoredModels...)

	report, _, err := e.runDownload(t, manifest, http.DefaultClient, "", nil)
	if err == nil {
		t.Fatalf("RunDownload error = nil, want the fallback-less model to fail the run")
	}
	if report.ModelsDone != 1 || report.ModelsFailed != 1 {
		t.Errorf("report = %d done, %d failed, want 1 done, 1 failed", report.ModelsDone, report.ModelsFailed)
	}
	if len(report.FailedModels) != 1 || report.FailedModels[0] != "clip-a" {
		t.Errorf("failed models = %v, want [clip-a]", report.FailedModels)
	}
	e.assertModelLanded(t, group[0])
	if _, ok := e.modelRowOf(t, "clip-a"); ok {
		t.Errorf("clip-a is registered despite its source failure")
	}
	if _, err := e.registry.Active(SlotClipEmbed); !errors.Is(err, ErrNoActiveModel) {
		t.Errorf("clip-embed slot error = %v, want ErrNoActiveModel (nothing activated)", err)
	}
}

// TestRunDownloadReplacesStalePartialFiles pins the interrupted-retry
// convergence: a half file left behind by an interrupted run (its
// digest cannot match) is replaced by the verified bytes, and the
// model directory ends up holding exactly the landed model -- weights
// plus the manifest copy, no staging leftovers.
func TestRunDownloadReplacesStalePartialFiles(t *testing.T) {
	e := newDownloadEnv(t)
	group := []downloadModel{{
		id:   "face-a",
		slot: "face-detect",
		files: []downloadFile{
			{name: "face.onnx", content: bytes.Repeat([]byte{0x0a}, 64*1024)},
		},
	}}
	primary := newDownloadSource(t, sourceFiles(group...))
	manifest := downloadManifest(anchored(group, primary.URL, "")...)
	file := group[0].files[0]

	dir := filepath.Join(e.modelsDir, group[0].id)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("create the stale model dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, file.name), file.content[:len(file.content)/2], 0o644); err != nil {
		t.Fatalf("write the stale partial file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".staging-"+file.name), []byte("interrupted junk"), 0o644); err != nil {
		t.Fatalf("write the stale staging leftover: %v", err)
	}

	report, _, err := e.runDownload(t, manifest, http.DefaultClient, "", nil)
	if err != nil {
		t.Fatalf("RunDownload error = %v, want nil", err)
	}
	if report.ModelsDone != 1 || report.BytesDone != report.BytesTotal {
		t.Errorf("report = %d done of %d total bytes, %d models, want the full model", report.BytesDone, report.BytesTotal, report.ModelsDone)
	}

	data, err := os.ReadFile(filepath.Join(dir, file.name))
	if err != nil {
		t.Fatalf("read the replaced file: %v", err)
	}
	if !bytes.Equal(data, file.content) {
		t.Errorf("the stale partial file was not replaced by the verified bytes")
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read the model dir: %v", err)
	}
	names := map[string]bool{}
	for _, entry := range entries {
		names[entry.Name()] = true
	}
	want := map[string]bool{file.name: true, manifestFileName: true}
	if len(names) != len(want) || names[file.name] != want[file.name] || names[manifestFileName] != want[manifestFileName] {
		t.Errorf("model dir holds %v, want exactly %v (no staging leftovers)", names, want)
	}
	e.assertModelLanded(t, group[0])
}

// TestRunDownloadRejectsChecksumMismatch pins the checksum gate: a
// source that serves bytes drifting from the pinned digest fails its
// model (unregistered, slot stays empty) while the other models of the
// run still land.
func TestRunDownloadRejectsChecksumMismatch(t *testing.T) {
	e := newDownloadEnv(t)
	good := downloadModel{
		id:   "face-a",
		slot: "face-detect",
		files: []downloadFile{
			{name: "face.onnx", content: bytes.Repeat([]byte{0x0a}, 64*1024)},
		},
	}
	bad := downloadModel{
		id:   "ocr-a",
		slot: "ocr",
		files: []downloadFile{
			{name: "det.onnx", content: bytes.Repeat([]byte{0x0d}, 64*1024)},
			{name: "rec.onnx", content: bytes.Repeat([]byte{0x0e}, 32*1024)},
		},
	}
	// The manifest pins the true digests; the source serves tampered
	// bytes for the bad model's first file.
	serving := sourceFiles(good, bad)
	tampered := append([]byte{}, serving["/ocr-a/det.onnx"]...)
	tampered[len(tampered)/2] ^= 0xff
	serving["/ocr-a/det.onnx"] = tampered
	primary := newDownloadSource(t, serving)
	manifest := downloadManifest(anchored([]downloadModel{good, bad}, primary.URL, "")...)

	report, _, err := e.runDownload(t, manifest, http.DefaultClient, "", nil)
	if err == nil {
		t.Fatalf("RunDownload error = nil, want the tampered model to fail the run")
	}
	if report.ModelsFailed != 1 || len(report.FailedModels) != 1 || report.FailedModels[0] != "ocr-a" {
		t.Errorf("failed models = %v, want [ocr-a]", report.FailedModels)
	}
	e.assertModelLanded(t, good)
	if _, ok := e.modelRowOf(t, "ocr-a"); ok {
		t.Errorf("the tampered model is registered")
	}
	if _, err := e.registry.Active(SlotOcr); !errors.Is(err, ErrNoActiveModel) {
		t.Errorf("ocr slot error = %v, want ErrNoActiveModel (nothing activated)", err)
	}
}

// TestMirrorURLRewritesOnlyTheHuggingFaceHost pins the mirror rewrite
// rule: the huggingface.co hostname is replaced wholesale (the mirror
// may carry a port), an empty mirror stays direct, and every other
// host stays exactly as pinned.
func TestMirrorURLRewritesOnlyTheHuggingFaceHost(t *testing.T) {
	cases := []struct {
		name   string
		raw    string
		mirror string
		want   string
	}{
		{
			name:   "rewrites the HF host wholesale",
			raw:    "https://huggingface.co/opencv/opencv_zoo/resolve/main/models/face_detection_yunet/face_detection_yunet_2023mar.onnx",
			mirror: "hf-mirror.com",
			want:   "https://hf-mirror.com/opencv/opencv_zoo/resolve/main/models/face_detection_yunet/face_detection_yunet_2023mar.onnx",
		},
		{
			name:   "empty mirror stays direct",
			raw:    "https://huggingface.co/a/b.onnx",
			mirror: "",
			want:   "https://huggingface.co/a/b.onnx",
		},
		{
			name:   "other hosts stay untouched",
			raw:    "https://media.githubusercontent.com/media/opencv/opencv_zoo/main/models/face_detection_yunet/face_detection_yunet_2023mar.onnx",
			mirror: "hf-mirror.com",
			want:   "https://media.githubusercontent.com/media/opencv/opencv_zoo/main/models/face_detection_yunet/face_detection_yunet_2023mar.onnx",
		},
		{
			name:   "mirror may carry a port",
			raw:    "https://huggingface.co/a/b.onnx",
			mirror: "127.0.0.1:9371",
			want:   "https://127.0.0.1:9371/a/b.onnx",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := mirrorURL(tc.raw, tc.mirror)
			if err != nil {
				t.Fatalf("mirrorURL(%q, %q) error = %v, want nil", tc.raw, tc.mirror, err)
			}
			if got != tc.want {
				t.Errorf("mirrorURL(%q, %q) = %q, want %q", tc.raw, tc.mirror, got, tc.want)
			}
		})
	}
}

// TestRunDownloadThroughHFMirror pins the mirror end to end: a
// huggingface.co URL is sent to the mirror source (which speaks TLS,
// the rewritten URL keeping the https scheme), while a non-huggingface
// URL reaches its own source untouched.
func TestRunDownloadThroughHFMirror(t *testing.T) {
	e := newDownloadEnv(t)
	faceA := downloadGroup()[0]
	clipA := downloadGroup()[2]
	mirrorSource := newTLSDownloadSource(t, sourceFiles(faceA))
	directSource := newDownloadSource(t, sourceFiles(clipA))

	models := []downloadModel{faceA, clipA}
	// face-a's manifest URL is an upstream huggingface.co URL the
	// rewrite redirects; clip-a's URL names the direct source's own
	// host, which the rewrite must not touch.
	models[0].source = "https://huggingface.co/" + faceA.id + "/" + faceA.files[0].name
	models[1].source = directSource.URL + "/" + clipA.id + "/" + clipA.files[0].name
	manifest := downloadManifest(models...)

	mirrorHost := strings.TrimPrefix(mirrorSource.URL, "https://")
	report, _, err := e.runDownload(t, manifest, mirrorSource.Client(), mirrorHost, nil)
	if err != nil {
		t.Fatalf("RunDownload error = %v, want nil", err)
	}
	if report.ModelsDone != 2 || report.ModelsFailed != 0 {
		t.Errorf("report = %d done, %d failed, want 2 done, 0 failed", report.ModelsDone, report.ModelsFailed)
	}
	e.assertModelLanded(t, faceA)
	e.assertModelLanded(t, clipA)
	if got := mirrorSource.hitCount(); got != 1 {
		t.Errorf("mirror source requests = %d, want 1 (the HF URL was rewritten to it)", got)
	}
	if got := directSource.hitCount(); got != 1 {
		t.Errorf("direct source requests = %d, want 1 (the non-HF URL reached it untouched)", got)
	}
}

// TestRunDownloadSkipsAlreadyVerifiedFiles pins the idempotent re-run:
// a second run over a fully landed group fetches nothing (every file
// verifies on disk) and still reports the complete group.
func TestRunDownloadSkipsAlreadyVerifiedFiles(t *testing.T) {
	e := newDownloadEnv(t)
	group := downloadGroup()
	primary := newDownloadSource(t, sourceFiles(group...))
	manifest := downloadManifest(anchored(group, primary.URL, "")...)

	if _, _, err := e.runDownload(t, manifest, http.DefaultClient, "", nil); err != nil {
		t.Fatalf("first run error = %v, want nil", err)
	}
	firstHits := primary.hitCount()

	report, points, err := e.runDownload(t, manifest, http.DefaultClient, "", nil)
	if err != nil {
		t.Fatalf("second run error = %v, want nil", err)
	}
	if report.ModelsDone != 4 || report.BytesDone != report.BytesTotal {
		t.Errorf("second run report = %d models, %d of %d bytes, want the full group", report.ModelsDone, report.BytesDone, report.BytesTotal)
	}
	if got := primary.hitCount(); got != firstHits {
		t.Errorf("source requests after the second run = %d, want %d (nothing re-downloaded)", got, firstHits)
	}
	if len(points) == 0 || points[len(points)-1].done != report.BytesTotal {
		t.Errorf("second run progress ends at %d, want %d (skipped files still count)", points[len(points)-1].done, report.BytesTotal)
	}
}

// fakeClock advances a fixed step on every read, so dense progress
// attempts translate into measurable fake elapsed time without any
// real sleeping.
type fakeClock struct {
	at   time.Time
	step time.Duration
}

func (c *fakeClock) Now() time.Time {
	c.at = c.at.Add(c.step)
	return c.at
}

// TestRunDownloadThrottlesProgress pins the progress throttle: one
// 2 MiB file over the 64 KiB read buffer yields 32 chunk-level
// progress attempts, and the throttle collapses them to a handful of
// callbacks -- at most one per second of (fake) elapsed time, plus the
// mandatory first and final ones.
func TestRunDownloadThrottlesProgress(t *testing.T) {
	e := newDownloadEnv(t)
	group := []downloadModel{{
		id:   "clip-a",
		slot: "clip-embed",
		dim:  512,
		files: []downloadFile{
			{name: "vision.onnx", content: bytes.Repeat([]byte{0x0c}, 2*1024*1024)},
		},
	}}
	primary := newDownloadSource(t, sourceFiles(group...))
	manifest := downloadManifest(anchored(group, primary.URL, "")...)

	clock := &fakeClock{at: time.Now(), step: 100 * time.Millisecond}
	report, points, err := e.runDownload(t, manifest, http.DefaultClient, "", clock.Now)
	if err != nil {
		t.Fatalf("RunDownload error = %v, want nil", err)
	}

	if len(points) > 10 {
		t.Errorf("progress callbacks = %d, want at most 10 (the throttle must collapse the 32 chunk attempts)", len(points))
	}
	if len(points) < 2 {
		t.Errorf("progress callbacks = %d, want at least the mandatory first and final ones", len(points))
	}
	if points[0].done != 0 {
		t.Errorf("first progress done = %d, want 0", points[0].done)
	}
	last := points[len(points)-1]
	if last.done != last.total || last.total != report.BytesTotal {
		t.Errorf("last progress = %d/%d, want %d/%d (the final report is mandatory)", last.done, last.total, report.BytesTotal, report.BytesTotal)
	}
	for i := 1; i < len(points); i++ {
		if points[i].done < points[i-1].done {
			t.Errorf("progress went backwards: %d then %d", points[i-1].done, points[i].done)
		}
	}
}
