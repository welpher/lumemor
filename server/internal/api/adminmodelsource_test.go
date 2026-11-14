// The model source pipeline's API tests: the probe POST/GET pair and
// the switch POST, walked against the real router over a throwaway
// database and the committed ONNX fixtures.
//
//   - the probe pair: a file probe answers 202 with an id, the GET
//     carries the settled facts (dims read off the graph), a
//     dimension mismatch and a non-ONNX file are honest error states,
//     an undeclared host is the warn state, and the face slots refuse
//     custom imports outright;
//   - the switch: a probe that did not finish ok is refused
//     (MODEL_PROBE_REQUIRED), a passing probe registers the
//     content-named model and activates it, the same-bytes reuse
//     activates the existing registration instead of storing a
//     duplicate, and the official-list switch needs its entry on
//     disk.
package api

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	apigen "github.com/welpher/lumemor/server/internal/api/gen"
	"github.com/welpher/lumemor/server/internal/ml"
)

// probeFileBody builds a multipart probe body carrying the fixture
// file's bytes as the file part, with the given text fields.
func probeFileBody(t *testing.T, fileField, fixture string, fields map[string]string) (*bytes.Buffer, string) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "ml", "testdata", fixture))
	if err != nil {
		t.Fatalf("read fixture %s: %v", fixture, err)
	}
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	for _, name := range []string{"kind", "url", "sha256"} {
		if value, ok := fields[name]; ok {
			if err := writer.WriteField(name, value); err != nil {
				t.Fatalf("write field %s: %v", name, err)
			}
		}
	}
	if fileField != "" {
		part, err := writer.CreateFormFile(fileField, fixture)
		if err != nil {
			t.Fatalf("create the file part: %v", err)
		}
		if _, err := part.Write(data); err != nil {
			t.Fatalf("write the file part: %v", err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close the multipart writer: %v", err)
	}
	return body, writer.FormDataContentType()
}

// probeFile fires one file probe POST.
func (e *adminModelsEnv) probeFile(t *testing.T, fixture string) *httptest.ResponseRecorder {
	t.Helper()
	body, contentType := probeFileBody(t, "file", fixture, map[string]string{"kind": "file"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/models/clip_embed/probe", body)
	req.Header.Set("Content-Type", contentType)
	req.AddCookie(e.cookie)
	rec := httptest.NewRecorder()
	e.router.ServeHTTP(rec, req)
	return rec
}

// probeGET reads one probe by id.
func (e *adminModelsEnv) probeGET(t *testing.T, id string) apigen.ModelProbeResponse {
	t.Helper()
	rec := e.do(t, http.MethodGet, "/api/v1/admin/models/probes/"+id, e.cookie)
	if rec.Code != http.StatusOK {
		t.Fatalf("read probe: status = %d, want %d (body %q)", rec.Code, http.StatusOK, rec.Body.String())
	}
	var body apigen.ModelProbeResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode the probe answer %q: %v", rec.Body.String(), err)
	}
	return body
}

// probeGETSettled polls one probe until it settles (the probe runs in
// the background; a URL download and the sandbox load are both
// bounded by the poll window).
func (e *adminModelsEnv) probeGETSettled(t *testing.T, id string) apigen.ModelProbeResponse {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for {
		probe := e.probeGET(t, id)
		if probe.State != "queued" && probe.State != "running" {
			return probe
		}
		if time.Now().After(deadline) {
			t.Fatalf("probe %s never settled (state %q)", id, probe.State)
		}
		time.Sleep(100 * time.Millisecond)
	}
}

// TestModelProbeFileAnswersTheGraphFacts pins the file probe: 202 with
// an identifier, and the settled GET carries the dims the graph
// declares (the fixture's constant 512 output) with the declared-host
// flag true (a local file has no host to refuse).
func TestModelProbeFileAnswersTheGraphFacts(t *testing.T) {
	e := newAdminModelsEnv(t)

	rec := e.probeFile(t, "const512.onnx")
	if rec.Code != http.StatusAccepted {
		t.Fatalf("probe: status = %d, want %d (body %q)", rec.Code, http.StatusAccepted, rec.Body.String())
	}
	var accepted apigen.ModelProbeAcceptedResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &accepted); err != nil {
		t.Fatalf("decode the accepted answer: %v", err)
	}
	probe := e.probeGETSettled(t, accepted.ProbeId)
	if probe.State != "ok" {
		t.Fatalf("probe state = %q (reason %q), want ok", probe.State, deref(probe.Reason))
	}
	if probe.Dims == nil || *probe.Dims != 512 {
		t.Errorf("probe dims = %v, want 512", probe.Dims)
	}
	if !probe.HostDeclared {
		t.Errorf("probe hostDeclared = false, want true (a local file has no host)")
	}
	if probe.Sha256 == nil || len(*probe.Sha256) != 64 {
		t.Errorf("probe sha256 = %v, want the computed digest", probe.Sha256)
	}
}

// deref answers a string pointer's value or the empty string.
func deref(raw *string) string {
	if raw == nil {
		return ""
	}
	return *raw
}

// TestModelProbeRefusesDimensionMismatch pins the slot contract: a
// 128-dim candidate for the 512-dim embed column is an honest error
// state naming both numbers -- and a failed probe saves nothing (the
// switch refuses its probe id).
func TestModelProbeRefusesDimensionMismatch(t *testing.T) {
	e := newAdminModelsEnv(t)

	rec := e.probeFile(t, "const128.onnx")
	if rec.Code != http.StatusAccepted {
		t.Fatalf("probe: status = %d, want %d", rec.Code, http.StatusAccepted)
	}
	var accepted apigen.ModelProbeAcceptedResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &accepted); err != nil {
		t.Fatalf("decode the accepted answer: %v", err)
	}
	probe := e.probeGETSettled(t, accepted.ProbeId)
	if probe.State != "error" {
		t.Fatalf("probe state = %q, want error", probe.State)
	}
	reason := deref(probe.Reason)
	if !strings.Contains(reason, "128") || !strings.Contains(reason, "512") {
		t.Errorf("probe reason %q does not name both dimensions", reason)
	}

	// The failed probe cannot be saved.
	switchBody := `{"kind":"url","probeId":"` + accepted.ProbeId + `"}`
	rec2 := e.doJSON(t, http.MethodPost, "/api/v1/admin/models/clip_embed/source", switchBody, e.cookie)
	if rec2.Code != http.StatusBadRequest {
		t.Fatalf("switch on a failed probe: status = %d, want %d (body %q)", rec2.Code, http.StatusBadRequest, rec2.Body.String())
	}
	if !strings.Contains(rec2.Body.String(), "MODEL_PROBE_REQUIRED") {
		t.Errorf("switch error = %q, want MODEL_PROBE_REQUIRED", rec2.Body.String())
	}
}

// TestModelProbeRefusesNonOnnx pins the content sniff: bytes that are
// not a readable ONNX model answer the error state, whatever the file
// name said.
func TestModelProbeRefusesNonOnnx(t *testing.T) {
	e := newAdminModelsEnv(t)

	rec := e.probeFile(t, "cn_clip_vocab.txt")
	if rec.Code != http.StatusAccepted {
		t.Fatalf("probe: status = %d, want %d", rec.Code, http.StatusAccepted)
	}
	var accepted apigen.ModelProbeAcceptedResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &accepted); err != nil {
		t.Fatalf("decode the accepted answer: %v", err)
	}
	probe := e.probeGETSettled(t, accepted.ProbeId)
	if probe.State != "error" {
		t.Fatalf("probe state = %q, want error", probe.State)
	}
	// Nothing staged survived: the failed probe left no file.
	entries, err := os.ReadDir(filepath.Join(e.modelsDir, ml.ProbeStagingRoot))
	if err == nil {
		for _, entry := range entries {
			dir := filepath.Join(e.modelsDir, ml.ProbeStagingRoot, entry.Name())
			if leftover, _ := os.ReadDir(dir); len(leftover) > 0 {
				t.Errorf("the failed probe left staged bytes under %s", dir)
			}
		}
	}
}

// TestModelProbeWarnsUndeclaredHost pins the warn state: a URL probe
// whose host is outside the allowed list settles as warn -- the checks
// passed, the save stays the administrator's act.
func TestModelProbeWarnsUndeclaredHost(t *testing.T) {
	// The local source serves the fixture bytes; the manager's hosts
	// list refuses the loopback host it answers on. The gate is the
	// answer's flag, not the download's refusal: the probe downloads
	// and settles warn.
	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		data, err := os.ReadFile(filepath.Join("..", "ml", "testdata", "const512.onnx"))
		if err != nil {
			http.Error(w, "no fixture", http.StatusNotFound)
			return
		}
		_, _ = w.Write(data)
	}))
	t.Cleanup(source.Close)

	e := newAdminModelsEnvHosts(t, func() []string { return []string{"declared.example"} })
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	if err := writer.WriteField("kind", "url"); err != nil {
		t.Fatalf("write the kind field: %v", err)
	}
	if err := writer.WriteField("url", source.URL+"/model.onnx"); err != nil {
		t.Fatalf("write the url field: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close the multipart writer: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/models/clip_embed/probe", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.AddCookie(e.cookie)
	rec := httptest.NewRecorder()
	e.router.ServeHTTP(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("probe: status = %d, want %d (body %q)", rec.Code, http.StatusAccepted, rec.Body.String())
	}
	var accepted apigen.ModelProbeAcceptedResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &accepted); err != nil {
		t.Fatalf("decode the accepted answer: %v", err)
	}
	probe := e.probeGETSettled(t, accepted.ProbeId)
	if probe.State != "warn" {
		t.Fatalf("probe state = %q (reason %q), want warn", probe.State, deref(probe.Reason))
	}
	if probe.HostDeclared {
		t.Errorf("probe hostDeclared = true, want false (the loopback host is not on the list)")
	}
	if probe.Dims == nil || *probe.Dims != 512 {
		t.Errorf("probe dims = %v, want 512 (the warn carries the facts too)", probe.Dims)
	}
}

// TestModelProbeRefusesFaceSlots pins the slot boundary: the face
// slots and OCR take the official list only.
func TestModelProbeRefusesFaceSlots(t *testing.T) {
	e := newAdminModelsEnv(t)

	rec := e.do(t, http.MethodPost, "/api/v1/admin/models/face_detect/probe", e.cookie)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("face detect probe: status = %d, want %d (body %q)", rec.Code, http.StatusBadRequest, rec.Body.String())
	}

	// An unknown slot is a 404.
	rec = e.do(t, http.MethodPost, "/api/v1/admin/models/nonsense/probe", e.cookie)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("unknown slot probe: status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

// TestModelSwitchCustomRegistersAndActivates pins the register+swap:
// a passing probe's bytes register under the content's own name
// (the digest's short form), the row activates, and the listing's
// slot row carries the custom source facts.
func TestModelSwitchCustomRegistersAndActivates(t *testing.T) {
	e := newAdminModelsEnv(t)

	rec := e.probeFile(t, "const512.onnx")
	var accepted apigen.ModelProbeAcceptedResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &accepted); err != nil {
		t.Fatalf("decode the accepted answer: %v", err)
	}
	probe := e.probeGETSettled(t, accepted.ProbeId)
	if probe.State != "ok" {
		t.Fatalf("probe state = %q, want ok", probe.State)
	}

	switchBody := `{"kind":"file","probeId":"` + accepted.ProbeId + `"}`
	rec2 := e.doJSON(t, http.MethodPost, "/api/v1/admin/models/clip_embed/source", switchBody, e.cookie)
	if rec2.Code != http.StatusAccepted {
		t.Fatalf("switch: status = %d, want %d (body %q)", rec2.Code, http.StatusAccepted, rec2.Body.String())
	}
	var switched apigen.ModelSourceSwitchResponse
	if err := json.Unmarshal(rec2.Body.Bytes(), &switched); err != nil {
		t.Fatalf("decode the switch answer: %v", err)
	}
	if !strings.HasSuffix(switched.Activated.Id, "@"+(*probe.Sha256)[:8]) {
		t.Errorf("activated id = %q, want the content-named slug", switched.Activated.Id)
	}
	if switched.Activated.Slot != apigen.ClipEmbed {
		t.Errorf("activated slot = %q, want clip_embed", switched.Activated.Slot)
	}
	// Same dimension as the column: no recalculation is due.
	if switched.RecalcRunId != nil {
		t.Errorf("recalcRunId = %q, want null (same-dimension switches keep both generations)", *switched.RecalcRunId)
	}
	if e.stub.calls != 0 {
		t.Errorf("rerun enqueues = %d, want 0", e.stub.calls)
	}

	// The probe's staged bytes are consumed: the id stages nothing
	// twice.
	rec3 := e.doJSON(t, http.MethodPost, "/api/v1/admin/models/clip_embed/source", switchBody, e.cookie)
	if rec3.Code != http.StatusBadRequest {
		t.Errorf("re-switch of a consumed probe: status = %d, want %d", rec3.Code, http.StatusBadRequest)
	}

	// The listing's slot row: active, custom, the host facts carried.
	rec4 := e.do(t, http.MethodGet, "/api/v1/admin/models", e.cookie)
	var listing apigen.ModelListResponse
	if err := json.Unmarshal(rec4.Body.Bytes(), &listing); err != nil {
		t.Fatalf("decode the listing: %v", err)
	}
	for _, slot := range listing.Slots {
		if slot.Slot != apigen.ClipEmbed {
			continue
		}
		if slot.State != apigen.SlotStateActive {
			t.Errorf("clip slot state = %q, want active", slot.State)
		}
		if slot.Source == nil || slot.Source.Kind != apigen.SlotSourceResponseKindCustom {
			t.Errorf("clip slot source = %+v, want custom", slot.Source)
		}
	}
	// The official list names the manifest entry as not-downloaded
	// (the mirror carries no weights).
	var official int
	for _, entry := range listing.Official {
		if entry.Slot == apigen.ClipEmbed {
			official++
			if entry.Downloaded {
				t.Errorf("official entry %q downloaded = true, want false (no weights on disk)", entry.Id)
			}
		}
	}
	if official != 1 {
		t.Errorf("official clip entries = %d, want 1", official)
	}
}

// TestModelSwitchSameBytesReusesTheRegistration pins the same-bytes
// rule: a custom import whose bytes a registered model already
// carries activates that registration and stages nothing twice.
func TestModelSwitchSameBytesReusesTheRegistration(t *testing.T) {
	e := newAdminModelsEnv(t, fixtureClipA)
	e.scanModels(t)

	// The fixture's bytes are already on disk under clip-a.
	rec := e.probeFile(t, "const512.onnx")
	var accepted apigen.ModelProbeAcceptedResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &accepted); err != nil {
		t.Fatalf("decode the accepted answer: %v", err)
	}
	switchBody := `{"kind":"file","probeId":"` + accepted.ProbeId + `"}`
	rec2 := e.doJSON(t, http.MethodPost, "/api/v1/admin/models/clip_embed/source", switchBody, e.cookie)
	if rec2.Code != http.StatusAccepted {
		t.Fatalf("switch: status = %d, want %d (body %q)", rec2.Code, http.StatusAccepted, rec2.Body.String())
	}
	var switched apigen.ModelSourceSwitchResponse
	if err := json.Unmarshal(rec2.Body.Bytes(), &switched); err != nil {
		t.Fatalf("decode the switch answer: %v", err)
	}
	if switched.Activated.Id != "clip-a" {
		t.Errorf("activated id = %q, want clip-a (the same bytes are one model)", switched.Activated.Id)
	}
}

// TestModelSwitchBuiltinNeedsTheEntryOnDisk pins the official-list
// switch's gate: an entry whose weights are not on disk is refused
// with the download-first message, and nothing changes.
func TestModelSwitchBuiltinNeedsTheEntryOnDisk(t *testing.T) {
	e := newAdminModelsEnv(t)

	switchBody := `{"kind":"builtin","modelId":"cnclip-vit-b16-vision"}`
	rec := e.doJSON(t, http.MethodPost, "/api/v1/admin/models/clip_embed/source", switchBody, e.cookie)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("builtin switch: status = %d, want %d (body %q)", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "MODEL_INVALID") {
		t.Errorf("builtin switch error = %q, want MODEL_INVALID", rec.Body.String())
	}

	// An entry of another slot, or an unknown id: 404.
	rec = e.doJSON(t, http.MethodPost, "/api/v1/admin/models/clip_embed/source", `{"kind":"builtin","modelId":"yunet-2023mar"}`, e.cookie)
	if rec.Code != http.StatusNotFound {
		t.Errorf("builtin switch of a foreign entry: status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

// TestModelSwitchBuiltinActivatesAndRecalcs pins the official-list
// switch with the entry on disk: the face slot's switch activates the
// model and enqueues the recalculation run the slot's semantics
// demand.
func TestModelSwitchBuiltinActivatesAndRecalcs(t *testing.T) {
	e := newAdminModelsEnv(t, fixtureDetect)
	// Lay the official entry's files out: the catalog's downloaded
	// marker is a presence probe.
	dir := filepath.Join(e.modelsDir, "yunet-2023mar")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("create the entry dir: %v", err)
	}
	data, err := os.ReadFile(filepath.Join("..", "ml", "testdata", "face_detection_yunet_2023mar.onnx"))
	if err != nil {
		t.Fatalf("read the yunet fixture: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "face_detection_yunet_2023mar.onnx"), data, 0o644); err != nil {
		t.Fatalf("write the entry weights: %v", err)
	}

	switchBody := `{"kind":"builtin","modelId":"yunet-2023mar"}`
	rec := e.doJSON(t, http.MethodPost, "/api/v1/admin/models/face_detect/source", switchBody, e.cookie)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("builtin switch: status = %d, want %d (body %q)", rec.Code, http.StatusAccepted, rec.Body.String())
	}
	var switched apigen.ModelSourceSwitchResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &switched); err != nil {
		t.Fatalf("decode the switch answer: %v", err)
	}
	if switched.Activated.Id != "yunet-2023mar" {
		t.Errorf("activated id = %q, want yunet-2023mar", switched.Activated.Id)
	}
	// A face slot's switch recomputes its products: the run rides the
	// response.
	if switched.RecalcRunId == nil {
		t.Errorf("recalcRunId = null, want the enqueued run")
	}
	if e.stub.calls != 1 || len(e.stub.slots) != 1 || e.stub.slots[0] != ml.SlotFaceDetect {
		t.Errorf("rerun enqueues = %d slots %v, want one for face_detect", e.stub.calls, e.stub.slots)
	}
}

// TestModelSwitchCrossDimensionRequiresConfirm pins the destructive
// arm's gate: a cross-dimension builtin switch without the
// confirmation answers MODEL_CONFIRM_REQUIRED, and the confirmed one
// reports the wipe and enqueues the rebuild. The fixture row rides
// the official entry's id (the dialog's official group names catalog
// entries; the catalog's single clip entry is the id the gate fires
// under).
func TestModelSwitchCrossDimensionRequiresConfirm(t *testing.T) {
	e := newAdminModelsEnv(t)
	// The official entry laid out at a different dimension than the
	// column's: the same id the catalog names, the files present, the
	// row registered by the scan (the entry was landed by another
	// path, the bundle's case).
	dir := filepath.Join(e.modelsDir, "cnclip-vit-b16-vision")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("create the entry dir: %v", err)
	}
	data, err := os.ReadFile(filepath.Join("..", "ml", "testdata", "const768.onnx"))
	if err != nil {
		t.Fatalf("read the fixture: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "cn_clip_vision.onnx"), data, 0o644); err != nil {
		t.Fatalf("write the entry weights: %v", err)
	}
	manifest := ml.ManifestModel{
		ID:      "cnclip-vit-b16-vision",
		Slot:    "clip-embed",
		Version: "fixture-768",
		License: "Apache-2.0",
		Notice:  "Fixture entry at 768.",
		InputSize: ml.ManifestSize{Width: 8, Height: 8},
		Dim:     768,
		Params:  map[string]any{},
		Files:   []ml.ManifestFile{{Name: "cn_clip_vision.onnx", SHA256: adminConst768SHA256, Bytes: int64(len(data))}},
	}
	blob, err := json.Marshal(manifest)
	if err != nil {
		t.Fatalf("marshal the entry manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), blob, 0o644); err != nil {
		t.Fatalf("write the entry manifest: %v", err)
	}
	e.scanModels(t)

	// The cross-dimension switch must be confirmed.
	switchBody := `{"kind":"builtin","modelId":"cnclip-vit-b16-vision"}`
	rec := e.doJSON(t, http.MethodPost, "/api/v1/admin/models/clip_embed/source", switchBody, e.cookie)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("cross-dim switch: status = %d, want %d (body %q)", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "MODEL_CONFIRM_REQUIRED") {
		t.Errorf("cross-dim switch error = %q, want MODEL_CONFIRM_REQUIRED", rec.Body.String())
	}

	// The confirmed switch runs the wipe and enqueues the rebuild.
	rec = e.doJSON(t, http.MethodPost, "/api/v1/admin/models/clip_embed/source",
		`{"kind":"builtin","modelId":"cnclip-vit-b16-vision","confirmWipe":true}`, e.cookie)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("confirmed switch: status = %d, want %d (body %q)", rec.Code, http.StatusAccepted, rec.Body.String())
	}
	var switched apigen.ModelSourceSwitchResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &switched); err != nil {
		t.Fatalf("decode the switch answer: %v", err)
	}
	if switched.DimensionChanged == nil || switched.DimensionChanged.From != 512 || switched.DimensionChanged.To != 768 {
		t.Errorf("dimensionChanged = %+v, want 512 to 768", switched.DimensionChanged)
	}
	if switched.RecalcRunId == nil {
		t.Errorf("recalcRunId = null, want the enqueued rebuild")
	}
}

// TestModelSwitchSlotLock pins the serialization: a second switch of
// the same slot while one holds the lock answers 409.
func TestModelSwitchSlotLock(t *testing.T) {
	e := newAdminModelsEnv(t)

	unlock, locked := modelSlotLocks.tryLock("clip_embed")
	if !locked {
		t.Fatalf("the test could not take the slot lock")
	}
	t.Cleanup(unlock)
	rec := e.doJSON(t, http.MethodPost, "/api/v1/admin/models/clip_embed/source", `{"kind":"url"}`, e.cookie)
	if rec.Code != http.StatusConflict {
		t.Errorf("switch under a held lock: status = %d, want %d", rec.Code, http.StatusConflict)
	}
	if !strings.Contains(rec.Body.String(), "MODEL_SLOT_LOCKED") {
		t.Errorf("locked switch error = %q, want MODEL_SLOT_LOCKED", rec.Body.String())
	}
}
