// The download hosts' gate tests: the per-hop validation that fronts
// the chain walk, against local httptest sources only.
//
//   - the matching rule: exact hostnames, `*.` wildcard suffixes at
//     label boundaries, ports stripped, an EMPTY list admitting
//     everything (the unconfigured gate);
//   - the chain walk: a candidate on a refused host is skipped with
//     the host named and the next candidate still lands the file;
//   - the redirect gate: a CDN 302 to an admitted host passes, a
//     redirect to an unnamed host stops the transfer with the host
//     named.
package ml

import (
	"bytes"
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// dialContext is context.Context under the dialer's name.
type dialContext = context.Context

// TestHostAllowedMatchingRule pins the entry grammar: exact matches,
// wildcard suffixes at label boundaries (never bare-suffix hits), port
// stripping, case folding, and the empty list's unconditional yes.
func TestHostAllowedMatchingRule(t *testing.T) {
	allowed := []string{"huggingface.co", "*.hf.co"}
	cases := []struct {
		host    string
		allowed bool
	}{
		{"huggingface.co", true},
		{"HUGGINGFACE.COM", false},
		{"cdn.hf.co", true},
		{"hf.co", false},
		{"evilhf.co", false},
		{"other.example.com", false},
	}
	for _, tc := range cases {
		if got := HostAllowed(allowed, tc.host); got != tc.allowed {
			t.Errorf("HostAllowed(%q) = %v, want %v", tc.host, got, tc.allowed)
		}
	}
	if !HostAllowed(nil, "anything.example") {
		t.Errorf("an empty list must admit every host (the unconfigured gate)")
	}
}

// sourceRedirect is a source that answers every GET with a redirect.
type sourceRedirect struct {
	*httptest.Server
}

func newSourceRedirect(t *testing.T, target string) *sourceRedirect {
	t.Helper()
	s := &sourceRedirect{}
	s.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target, http.StatusFound)
	}))
	t.Cleanup(s.Close)
	return s
}

// TestRunDownloadSkipsRefusedHostsToNextCandidate pins the chain walk
// under the host gate: a model whose first candidate names a refused
// host moves on to the second one (which lands the file), and the
// failure names the host -- the report is what the admin face shows.
// The empty list keeps the same walk unrestricted (the unconfigured
// gate the tests run under everywhere else).
func TestRunDownloadSkipsRefusedHostsToNextCandidate(t *testing.T) {
	e := newDownloadEnv(t)
	group := []downloadModel{{
		id:   "face-a",
		slot: "face-detect",
		files: []downloadFile{
			{name: "face.onnx", content: bytes.Repeat([]byte{0x0a}, 32*1024)},
		},
	}}
	secondary := newDownloadSource(t, sourceFiles(group...))
	file := group[0].files[0]

	// Revision 2's ordered chain: the first entry names a host the
	// list refuses (its bytes would never be reached), the second
	// names the allowed source.
	manifest := downloadManifest(group...)
	manifest.Models[0].Urls = []string{
		"https://off-list.example/" + group[0].id + "/" + file.name,
		secondary.URL + "/" + group[0].id + "/" + file.name,
	}
	manifest.Models[0].Hosts = []string{"huggingface.co"}

	// The gate: the on-list source admitted, the off-list host
	// refused. The walk moves on and the file still lands.
	report, _, err := e.runDownloadList(t, manifest, []string{"127.0.0.1"})
	if err != nil {
		t.Fatalf("RunDownload error = %v, want nil (the allowed candidate landed the file)", err)
	}
	if report.ModelsDone != 1 {
		t.Errorf("report models done = %d, want 1", report.ModelsDone)
	}
	e.assertModelLanded(t, group[0])

	// The named failure: with every candidate refused, the error
	// carries the host, not a bare transport failure. The refusal
	// case runs on a fresh environment: the landed file of the first
	// half would be skipped as already verified, and the walk would
	// never fire.
	e2 := newDownloadEnv(t)
	manifest2 := downloadManifest(group...)
	manifest2.Models[0].Urls = []string{
		"https://off-list.example/" + group[0].id + "/" + file.name,
	}
	_, _, walkErr := e2.runDownloadList(t, manifest2, []string{"127.0.0.1"})
	if walkErr == nil {
		t.Fatalf("RunDownload error = nil, want the refused host to fail the model")
	}
	if !strings.Contains(walkErr.Error(), "off-list.example") {
		t.Errorf("error %q does not name the refused host", walkErr)
	}
}

// TestRunDownloadValidatesRedirectHops pins the per-hop rule on the
// redirect path: a redirect to an admitted host passes and lands the
// bytes (the CDN hop), a redirect to an unnamed host aborts with the
// host named. Both sources answer on the loopback; the URLs carry the
// names namedHostClient's dialer maps back onto it, so the gate's
// hostname matching is what the two cases separate.
func TestRunDownloadValidatesRedirectHops(t *testing.T) {
	e := newDownloadEnv(t)
	group := []downloadModel{{
		id:   "face-a",
		slot: "face-detect",
		files: []downloadFile{
			{name: "face.onnx", content: bytes.Repeat([]byte{0x0a}, 32*1024)},
		},
	}}
	file := group[0].files[0]
	allowed := newDownloadSource(t, sourceFiles(group...))
	// The redirector answers every GET with a 302 to the allowed
	// source's file, the URL named allowed.test instead of 127.0.0.1.
	target := strings.Replace(allowed.URL, "127.0.0.1", "allowed.test", 1) + "/" + group[0].id + "/" + file.name
	redirector := newSourceRedirect(t, target)
	redirectURL := strings.Replace(redirector.URL, "127.0.0.1", "redirector.test", 1)
	manifest := downloadManifest(group...)
	manifest.Models[0].Urls = []string{redirectURL + "/" + group[0].id + "/" + file.name}

	report, _, err := e.runDownloadList(t, manifest, []string{"allowed.test", "redirector.test"})
	if err != nil {
		t.Fatalf("RunDownload error = %v, want nil (the redirect's host is admitted)", err)
	}
	e.assertModelLanded(t, group[0])
	if report.ModelsDone != 1 {
		t.Errorf("report models done = %d, want 1", report.ModelsDone)
	}

	// The same shape with the redirect target's host outside the
	// list: the transfer stops at the hop, the model fails.
	e2 := newDownloadEnv(t)
	if _, _, err := e2.runDownloadList(t, manifest, []string{"redirector.test"}); err == nil {
		t.Fatalf("RunDownload error = nil, want the redirect to a refused host to fail")
	}
	if _, ok := e2.modelRowOf(t, "face-a"); ok {
		t.Errorf("the redirected model registered despite its refused hop")
	}
}

// runDownloadList runs one download with the given allowed hosts.
func (e *downloadEnv) runDownloadList(t *testing.T, manifest *Manifest, allowed []string) (DownloadReport, []progressPoint, error) {
	t.Helper()
	var points []progressPoint
	report, err := RunDownload(t.Context(), RunDownloadDeps{
		Manifest:     manifest,
		ModelsDir:    e.modelsDir,
		Client:       namedHostClient,
		Registry:     e.registry,
		Queries:      e.queries,
		AllowedHosts: allowed,
	}, func(done, total int) {
		points = append(points, progressPoint{done: done, total: total})
	})
	return report, points, err
}

// namedHostClient dials the loopback under the names the redirect
// tests use: both httptest sources answer on 127.0.0.1, and the gate
// matches hostnames, so the test's URLs carry the names the dialer
// maps back onto the loopback.
var namedHostClient = &http.Client{Transport: &http.Transport{
	DialContext: func(dialCtx dialContext, network, addr string) (net.Conn, error) {
		if host, port, err := net.SplitHostPort(addr); err == nil && strings.HasSuffix(host, ".test") {
			addr = "127.0.0.1:" + port
		}
		var dialer net.Dialer
		return dialer.DialContext(dialCtx, network, addr)
	},
}}

// TestCandidateURLsPinsTheChainOrder pins the accessor both manifest
// revisions share: revision 2's Urls list is answered verbatim in
// order; a revision 1 model answers primary then fallback.
func TestCandidateURLsPinsTheChainOrder(t *testing.T) {
	v2 := ManifestModel{Urls: []string{"https://a/x", "https://b/x", "https://c/x"}}
	if got := v2.CandidateURLs(); len(got) != 3 || got[0] != "https://a/x" || got[2] != "https://c/x" {
		t.Errorf("revision 2 chain = %v, want the ordered three", got)
	}
	fallback := "https://b/x"
	v1 := ManifestModel{SourceURL: "https://a/x", FallbackURL: &fallback}
	got := v1.CandidateURLs()
	if len(got) != 2 || got[0] != "https://a/x" || got[1] != "https://b/x" {
		t.Errorf("revision 1 chain = %v, want primary then fallback", got)
	}
	empty := ManifestModel{}
	if got := empty.CandidateURLs(); len(got) != 0 {
		t.Errorf("a model without sources answers %v, want none", got)
	}
}

// TestManifestDeduplicatePinsTheSameBytesRule pins the dedup: a custom
// entry whose sha256 an official entry already carries registers
// nothing (the official one stays the single face of those bytes),
// while a genuinely new entry is appended once and counted.
func TestManifestDeduplicatePinsTheSameBytesRule(t *testing.T) {
	base := &Manifest{Models: []ManifestModel{{
		ID:    "official-a",
		Slot:  "face-detect",
		Files: []ManifestFile{{Name: "a.onnx", SHA256: "aabb", Bytes: 3}},
	}}}
	other := []ManifestModel{
		{
			ID:    "custom-copy",
			Slot:  "face-detect",
			Files: []ManifestFile{{Name: "copy.onnx", SHA256: "aabb", Bytes: 3}},
		},
		{
			ID:    "custom-new",
			Slot:  "ocr",
			Files: []ManifestFile{{Name: "new.onnx", SHA256: "ccdd", Bytes: 4}},
		},
		{
			ID:    "custom-new-again",
			Slot:  "ocr",
			Files: []ManifestFile{{Name: "new2.onnx", SHA256: "ccdd", Bytes: 4}},
		},
	}
	if added := base.Deduplicate(other); added != 1 {
		t.Errorf("Deduplicate added %d, want 1 (the byte-identical copy stays out, the new one joins once)", added)
	}
	if len(base.Models) != 2 || base.Models[1].ID != "custom-new" {
		t.Errorf("manifest models = %v, want [official-a custom-new]", modelIDs(base))
	}
}

func modelIDs(m *Manifest) []string {
	out := make([]string, 0, len(m.Models))
	for _, model := range m.Models {
		out = append(out, model.ID)
	}
	return out
}
