package ml

import (
	"encoding/json"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// defaultManifestPath points at the committed default model manifest,
// relative to this package's directory (the go tool runs tests there).
const defaultManifestPath = "manifests/default.json"

// pinnedDefaultGroup describes the one and only default model group:
// which model ids exist, which slot each fills, its embedding
// dimension, and how many files it downloads. Every entry is an
// off-the-shelf upstream artifact -- the manifest pins links and
// checksums, never weights.
var pinnedDefaultGroup = map[string]struct {
	slot  string
	dim   int
	files int
}{
	"yunet-2023mar":         {slot: "face-detect", dim: 0, files: 1},
	"sface-2021dec":         {slot: "face-recognize", dim: 128, files: 1},
	"cnclip-vit-b16-vision": {slot: "clip-embed", dim: 512, files: 1},
	"ppocr-v4-mobile-ch":    {slot: "ocr", dim: 0, files: 2},
}

// TestDefaultManifestIsCommitted pins the manifest's presence on
// disk: the default group is carried by the repository as pure links,
// so the JSON file itself is the deliverable.
func TestDefaultManifestIsCommitted(t *testing.T) {
	if _, err := os.Stat(defaultManifestPath); err != nil {
		t.Fatalf("default manifest %s is missing: %v", defaultManifestPath, err)
	}
}

// TestDefaultManifestPinsTheDefaultGroup pins the group shape: schema
// revision 1, exactly the four pinned model ids, every slot filled by
// exactly one model, and the embedding dimensions the storage layer
// is built around (clip 512, face recognition 128).
func TestDefaultManifestPinsTheDefaultGroup(t *testing.T) {
	manifest, err := LoadManifest(defaultManifestPath)
	if err != nil {
		t.Fatalf("LoadManifest(%q) = %v, want no error", defaultManifestPath, err)
	}

	if manifest.Revision != 1 {
		t.Errorf("revision = %d, want 1", manifest.Revision)
	}
	if len(manifest.Models) != len(pinnedDefaultGroup) {
		t.Fatalf("models = %d, want %d", len(manifest.Models), len(pinnedDefaultGroup))
	}

	seenIDs := map[string]bool{}
	seenSlots := map[string]string{}
	for _, model := range manifest.Models {
		if seenIDs[model.ID] {
			t.Errorf("model id %q is listed twice", model.ID)
			continue
		}
		seenIDs[model.ID] = true

		want, ok := pinnedDefaultGroup[model.ID]
		if !ok {
			t.Errorf("unexpected model id %q in the default group", model.ID)
			continue
		}
		if model.Slot != want.slot {
			t.Errorf("model %q slot = %q, want %q", model.ID, model.Slot, want.slot)
		}
		if model.Dim != want.dim {
			t.Errorf("model %q dim = %d, want %d", model.ID, model.Dim, want.dim)
		}
		if len(model.Files) != want.files {
			t.Errorf("model %q lists %d files, want %d", model.ID, len(model.Files), want.files)
		}
		if model.Version == "" {
			t.Errorf("model %q has an empty version", model.ID)
		}
		if model.License == "" {
			t.Errorf("model %q has an empty license", model.ID)
		}
		if model.Notice == "" {
			t.Errorf("model %q has an empty notice", model.ID)
		}
		if model.InputSize.Width <= 0 || model.InputSize.Height <= 0 {
			t.Errorf("model %q inputSize = %+v, want positive width and height", model.ID, model.InputSize)
		}
		if other, dup := seenSlots[model.Slot]; dup {
			t.Errorf("slot %q is filled by both %q and %q, want exactly one model per slot",
				model.Slot, other, model.ID)
		}
		seenSlots[model.Slot] = model.ID
	}
	for id := range pinnedDefaultGroup {
		if !seenIDs[id] {
			t.Errorf("pinned model %q is missing from the default group", id)
		}
	}
}

// TestDefaultManifestFilesAreChecksummed pins the anti-drift contract:
// every pinned file carries a 64-hex-char sha256 that is not all
// zeros and a positive byte count, i.e. the digests were computed
// from the real upstream bytes, not left as placeholders.
func TestDefaultManifestFilesAreChecksummed(t *testing.T) {
	manifest, err := LoadManifest(defaultManifestPath)
	if err != nil {
		t.Fatalf("LoadManifest(%q) = %v, want no error", defaultManifestPath, err)
	}

	for _, model := range manifest.Models {
		seenNames := map[string]bool{}
		for _, file := range model.Files {
			if file.Name == "" {
				t.Errorf("model %q has a file with an empty name", model.ID)
				continue
			}
			if seenNames[file.Name] {
				t.Errorf("model %q lists file %q twice", model.ID, file.Name)
			}
			seenNames[file.Name] = true

			if len(file.SHA256) != 64 {
				t.Errorf("model %q file %q: sha256 %q has %d chars, want 64 hex chars",
					model.ID, file.Name, file.SHA256, len(file.SHA256))
				continue
			}
			for _, r := range file.SHA256 {
				if !strings.ContainsRune("0123456789abcdef", r) {
					t.Errorf("model %q file %q: sha256 %q is not lowercase hex",
						model.ID, file.Name, file.SHA256)
					break
				}
			}
			if strings.Trim(file.SHA256, "0") == "" {
				t.Errorf("model %q file %q: sha256 is all zeros, want the digest of the real upstream bytes",
					model.ID, file.Name)
			}
			if file.Bytes <= 0 {
				t.Errorf("model %q file %q: bytes = %d, want a positive byte count",
					model.ID, file.Name, file.Bytes)
			}
		}
	}
}

// TestDefaultManifestURLsAreHTTPS pins that every download URL (and
// every fallback) is https, and that the community-converted CLIP
// vision tower has no fallback URL: no second source exists for it,
// so the HF domain mirror is its only redundancy.
func TestDefaultManifestURLsAreHTTPS(t *testing.T) {
	manifest, err := LoadManifest(defaultManifestPath)
	if err != nil {
		t.Fatalf("LoadManifest(%q) = %v, want no error", defaultManifestPath, err)
	}

	for _, model := range manifest.Models {
		assertHTTPSURL(t, "model "+model.ID+" sourceUrl", model.SourceURL)
		if model.FallbackURL != nil {
			assertHTTPSURL(t, "model "+model.ID+" fallbackUrl", *model.FallbackURL)
		}
	}

	clip := modelByID(t, manifest, "cnclip-vit-b16-vision")
	if clip.FallbackURL != nil {
		t.Errorf("cnclip-vit-b16-vision fallbackUrl = %q, want null (no natural fallback exists)",
			*clip.FallbackURL)
	}
}

// TestDefaultManifestPinsYunetInputGeometry pins the correction Task 6
// landed: the 2023mar detector graph measures its input at 640x640
// (the session metadata Task 2 read off the real weights), and the
// manifest's declared inputSize now states that fact -- the 320x320
// declaration the manifest shipped with was the upstream demo's
// default input, not the graph's. The declaration is display truth
// (the admin surface renders it); the detection wrapper always reads
// the geometry from the session itself.
func TestDefaultManifestPinsYunetInputGeometry(t *testing.T) {
	manifest, err := LoadManifest(defaultManifestPath)
	if err != nil {
		t.Fatalf("LoadManifest(%q) = %v, want no error", defaultManifestPath, err)
	}
	yunet := modelByID(t, manifest, "yunet-2023mar")
	if yunet.InputSize.Width != 640 || yunet.InputSize.Height != 640 {
		t.Errorf("yunet inputSize = %dx%d, want the measured 640x640", yunet.InputSize.Width, yunet.InputSize.Height)
	}
}

// TestLoadManifestRoundTripsDefaultJSON pins the JSON tags: the
// default manifest re-marshaled from the parsed struct loads back
// into an equal value, so no schema field can silently drop out of
// the Go representation.
func TestLoadManifestRoundTripsDefaultJSON(t *testing.T) {
	first, err := LoadManifest(defaultManifestPath)
	if err != nil {
		t.Fatalf("LoadManifest(%q) = %v, want no error", defaultManifestPath, err)
	}

	data, err := json.Marshal(first)
	if err != nil {
		t.Fatalf("re-marshal of the default manifest failed: %v", err)
	}
	path := filepath.Join(t.TempDir(), "roundtrip.json")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("writing the re-marshaled manifest failed: %v", err)
	}

	second, err := LoadManifest(path)
	if err != nil {
		t.Fatalf("LoadManifest(%q) = %v, want no error", path, err)
	}
	if !reflect.DeepEqual(*first, *second) {
		t.Errorf("manifest changed across the round trip:\nfirst:  %+v\nsecond: %+v", *first, *second)
	}
}

// TestLoadManifestParsesEveryField pins the full schema surface with
// a synthetic manifest: every field, including a null fallbackUrl and
// an absent params object, survives a load intact. Field values are
// JSON-native (float64 numbers) so the comparison is exact.
func TestLoadManifestParsesEveryField(t *testing.T) {
	fallback := "https://example.com/mirror.onnx"
	want := &Manifest{
		Revision: 7,
		Models: []ManifestModel{
			{
				ID:          "fixture-model",
				Slot:        "clip-embed",
				Version:     "v99",
				License:     "Apache-2.0",
				SourceURL:   "https://example.com/model.onnx",
				FallbackURL: &fallback,
				Notice:      "synthetic fixture model",
				InputSize:   ManifestSize{Width: 224, Height: 224},
				Dim:         512,
				Params: map[string]any{
					"scoreThreshold": 0.6,
					"mean":           []any{0.481, 0.457, 0.408},
					"label":          "fixture",
				},
				Files: []ManifestFile{
					{Name: "model.onnx", SHA256: strings.Repeat("ab", 32), Bytes: 12345},
					{Name: "extra.bin", SHA256: strings.Repeat("cd", 32), Bytes: 2},
				},
			},
			{
				ID:        "no-fallback-model",
				Slot:      "ocr",
				Version:   "v1",
				License:   "MIT",
				SourceURL: "https://example.com/other.onnx",
				Notice:    "synthetic fixture model without a fallback",
				InputSize: ManifestSize{Width: 960, Height: 960},
				Files:     []ManifestFile{{Name: "other.onnx", SHA256: strings.Repeat("ef", 32), Bytes: 7}},
			},
		},
	}

	data, err := json.Marshal(want)
	if err != nil {
		t.Fatalf("marshaling the synthetic manifest failed: %v", err)
	}
	path := filepath.Join(t.TempDir(), "every-field.json")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("writing the synthetic manifest failed: %v", err)
	}

	got, err := LoadManifest(path)
	if err != nil {
		t.Fatalf("LoadManifest(%q) = %v, want no error", path, err)
	}
	if !reflect.DeepEqual(*got, *want) {
		t.Errorf("LoadManifest dropped or changed fields:\ngot:  %+v\nwant: %+v", *got, *want)
	}
}

// TestLoadManifestErrorsOnMissingFile pins the failure contract later
// tasks rely on when a per-model manifest on disk is absent.
func TestLoadManifestErrorsOnMissingFile(t *testing.T) {
	_, err := LoadManifest(filepath.Join(t.TempDir(), "absent.json"))
	if err == nil {
		t.Fatal("LoadManifest on a missing file returned nil error, want an error")
	}
}

// assertHTTPSURL fails the test unless raw parses as an https URL
// with a host.
func assertHTTPSURL(t *testing.T, what, raw string) {
	t.Helper()
	u, err := url.Parse(raw)
	if err != nil {
		t.Errorf("%s = %q does not parse: %v", what, raw, err)
		return
	}
	if u.Scheme != "https" || u.Host == "" {
		t.Errorf("%s = %q, want an https URL with a host", what, raw)
	}
}

// modelByID returns the manifest model with the given id, failing the
// test when it is absent.
func modelByID(t *testing.T, manifest *Manifest, id string) ManifestModel {
	t.Helper()
	for _, model := range manifest.Models {
		if model.ID == id {
			return model
		}
	}
	t.Fatalf("model %q not found in the manifest", id)
	return ManifestModel{}
}
