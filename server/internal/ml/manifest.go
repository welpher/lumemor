// Package ml holds the machine-learning foundation of the lumemor
// server: the default model manifest, the onnxruntime session
// wrapper and the model registry, and later the first-boot
// download service and the inference wrappers (face detection, face
// recognition, CLIP embedding, OCR).
//
// The manifest is pure links: every artifact is an upstream URL plus
// a sha256 digest and a byte count, and the repository commits no
// weights. That keeps the licensing surface minimal (nothing is
// re-distributed) while the pinned digests make any upstream drift
// detectable at download and load time. The registry doubles the
// verification: the digests are checked again whenever a model is
// activated or loaded, so a file that drifted after download never
// reaches inference.
package ml

import (
	"encoding/json"
	"fmt"
	"os"
)

// Manifest is the parsed form of a model manifest JSON file. The
// committed default lives at manifests/default.json (embedded by the
// download service in a later task); per-model copies are written
// next to the downloaded weights on disk.
type Manifest struct {
	// Revision is the manifest schema revision, currently 1.
	Revision int `json:"revision"`
	// Models lists the pinned models. A model may carry several
	// files (the OCR model ships a detection and a recognition
	// network).
	Models []ManifestModel `json:"models"`
}

// ManifestModel pins one model of the default group: where its files
// come from, how they are verified, and the model facts the inference
// wrappers and the admin UI display.
type ManifestModel struct {
	// ID is the stable identity of the model, e.g. "yunet-2023mar".
	// It doubles as the directory name under the models dir.
	ID string `json:"id"`
	// Slot is the functional slot the model fills once activated:
	// face-detect, face-recognize, clip-embed or ocr.
	Slot string `json:"slot"`
	// Version distinguishes upstream releases of the same model
	// family (date tags or variant names).
	Version string `json:"version"`
	// License is the upstream license identifier, cross-referenced
	// with the provenance record in tools/models/NOTICES.md.
	License string `json:"license"`
	// SourceURL is the primary download URL, always https. Revision
	// 1's two-field shape: it and FallbackURL are the whole chain.
	SourceURL string `json:"sourceUrl"`
	// FallbackURL is the secondary URL tried when the primary fails.
	// It is nil when no natural second source exists; the mirror
	// rewrite then remains the only redundancy.
	FallbackURL *string `json:"fallbackUrl"`
	// Urls is the ordered candidate chain, revision 2's shape: the
	// fetcher tries every entry in order, each hop's host validated
	// and the bytes digest-checked, until one lands. It replaces
	// (and subsumes) SourceURL and FallbackURL, which a revision 2
	// manifest leaves empty.
	Urls []string `json:"urls,omitempty"`
	// Hosts names the hosts the chain travels through (including
	// the CDN hosts a redirect can reach), lowercase. It is the
	// model's declaration the listing and the import surface check
	// their sources against; a host the manifest declares is always
	// a permitted hop, while the instance's own allowed-hosts list
	// may name more.
	Hosts []string `json:"hosts,omitempty"`
	// Notice is the short provenance line surfaced in the admin UI.
	Notice string `json:"notice"`
	// InputSize is the expected input image geometry in pixels.
	InputSize ManifestSize `json:"inputSize"`
	// Dim is the embedding dimension for embedding models; 0 for
	// models that do not embed (detectors, OCR).
	Dim int `json:"dim"`
	// Params carries per-model inference defaults (thresholds,
	// normalization constants) consumed by the wrappers. Numbers
	// arrive as float64, as usual for JSON-decoded maps.
	Params map[string]any `json:"params"`
	// Files pins every artifact of the model by name, sha256 and
	// byte count. A downloaded model is only registered once every
	// file matches its digest.
	Files []ManifestFile `json:"files"`
}

// CandidateURLs answers the model's ordered download chain: revision
// 2's Urls list when it carries one, else the revision 1 pair (the
// primary and, when present, the fallback). Every reader -- the
// downloader, the offline import's provenance check -- goes through
// this accessor, so both revisions share one walk.
func (m *ManifestModel) CandidateURLs() []string {
	if len(m.Urls) > 0 {
		return m.Urls
	}
	candidates := make([]string, 0, 2)
	if m.SourceURL != "" {
		candidates = append(candidates, m.SourceURL)
	}
	if m.FallbackURL != nil && *m.FallbackURL != "" {
		candidates = append(candidates, *m.FallbackURL)
	}
	return candidates
}

// Deduplicate merges other into the manifest's model list by sha256:
// the same digest is the same artifact wherever it was found, so a
// custom import that reproduces an official model's bytes registers
// nothing new and the official entry -- the one with the provenance
// record -- stays the single face of those bytes. The merge is the
// manifest's dedup discipline: first-seen entries win their slot in
// the list.
func (m *Manifest) Deduplicate(other []ManifestModel) int {
	seen := map[string]bool{}
	for i := range m.Models {
		for _, file := range m.Models[i].Files {
			seen[file.SHA256] = true
		}
	}
	added := 0
	for _, model := range other {
		duplicated := false
		for _, file := range model.Files {
			if seen[file.SHA256] {
				duplicated = true
				break
			}
		}
		if duplicated {
			continue
		}
		m.Models = append(m.Models, model)
		for _, file := range model.Files {
			seen[file.SHA256] = true
		}
		added++
	}
	return added
}

// ManifestSize is the input geometry of a model, in pixels.
type ManifestSize struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

// ManifestFile pins one downloadable artifact.
type ManifestFile struct {
	// Name is the file name inside the model directory.
	Name string `json:"name"`
	// SHA256 is the lowercase hex digest of the pinned upstream
	// bytes.
	SHA256 string `json:"sha256"`
	// Bytes is the exact byte count of the pinned upstream bytes.
	Bytes int64 `json:"bytes"`
}

// LoadManifest reads and parses the manifest JSON at path. It is
// deliberately thin -- schema-level validation lives in the tests --
// so the same loader serves the embedded default manifest and the
// per-model manifests written to the models dir.
func LoadManifest(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read model manifest: %w", err)
	}
	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("parse model manifest %s: %w", path, err)
	}
	return &manifest, nil
}

// marshal renders the manifest as JSON, the bundle builder's and the
// tests' writer side of LoadManifest.
func (m *Manifest) marshal() ([]byte, error) {
	return json.Marshal(m)
}
