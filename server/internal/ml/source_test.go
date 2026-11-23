// The source probe's transport tests: the bounded client, the timeout
// that settles a wedged fetch, the per-hop redirect gate and the
// declared flag's final-hop rule. Local httptest sources only -- no
// database, no network.
package ml

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// probeSettle polls a probe until it leaves the running state.
func probeSettle(t *testing.T, pm *ProbeManager, id string) ProbeSnapshot {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for {
		record, ok := pm.Get(id)
		if !ok {
			t.Fatalf("probe %s vanished", id)
		}
		if record.State != ProbeQueued && record.State != ProbeRunning {
			return record
		}
		if time.Now().After(deadline) {
			t.Fatalf("probe %s never settled (state %q)", id, record.State)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// probeStagingFiles answers the files left under one probe's staging
// directory.
func probeStagingFiles(t *testing.T, modelsDir, id string) []string {
	t.Helper()
	dir := filepath.Join(modelsDir, probeStagingRoot, id)
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		t.Fatalf("read the staging dir: %v", err)
	}
	var names []string
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	return names
}

// TestProbeClientIsBounded pins the default client the manager builds:
// every pre-body phase carries a timeout (the dial, the handshake, the
// response headers) and the client itself carries none -- the transfer
// rides the request's deadline, so a client-level timeout would abort a
// legitimate large download on a slow link.
func TestProbeClientIsBounded(t *testing.T) {
	pm := NewProbeManager(t.TempDir(), nil, nil, nil, nil)
	if pm.deadline != probeDeadline {
		t.Errorf("deadline = %v, want %v", pm.deadline, probeDeadline)
	}
	if pm.client.Timeout != 0 {
		t.Errorf("client timeout = %v, want none (the request deadline bounds the transfer)", pm.client.Timeout)
	}
	transport, ok := pm.client.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("client transport = %T, want *http.Transport", pm.client.Transport)
	}
	if transport.TLSHandshakeTimeout != probeTLSHandshakeTimeout {
		t.Errorf("TLSHandshakeTimeout = %v, want %v", transport.TLSHandshakeTimeout, probeTLSHandshakeTimeout)
	}
	if transport.ResponseHeaderTimeout != probeResponseHeaderTimeout {
		t.Errorf("ResponseHeaderTimeout = %v, want %v", transport.ResponseHeaderTimeout, probeResponseHeaderTimeout)
	}
	if transport.DialContext == nil {
		t.Error("DialContext = nil, want the bounded dialer")
	}
}

// TestProbeTimesOutAndCleansStaging pins the deadline: a source that
// sends headers and then stalls settles the probe as the usual error,
// and the staging file it was writing is gone. The shortened deadline
// is the production one under test, not a different mechanism.
func TestProbeTimesOutAndCleansStaging(t *testing.T) {
	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "4096")
		w.WriteHeader(http.StatusOK)
		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		}
		// Send a first chunk so the staging file has bytes, then
		// stall until the client's deadline cancels the request.
		_, _ = w.Write([]byte("first chunk"))
		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		}
		<-r.Context().Done()
	}))
	t.Cleanup(source.Close)

	modelsDir := t.TempDir()
	pm := NewProbeManager(modelsDir, nil, nil, nil, nil)
	pm.deadline = 200 * time.Millisecond

	id, err := pm.Start(SlotClipEmbed, source.URL+"/model.onnx", "")
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	record := probeSettle(t, pm, id)
	if record.State != ProbeError {
		t.Fatalf("state = %q (reason %q), want %q", record.State, record.Reason, ProbeError)
	}
	if !strings.Contains(record.Reason, "deadline exceeded") {
		t.Errorf("reason = %q, want the deadline named", record.Reason)
	}
	if files := probeStagingFiles(t, modelsDir, id); len(files) != 0 {
		t.Errorf("staging left %v, want nothing", files)
	}
}

// TestProbeRefusesUndeclaredRedirectHop pins the per-hop gate: a
// redirect from a declared host to a host outside the list aborts the
// fetch with the refused host named, and the probe settles as the usual
// error with no staged bytes. The two spellings of loopback let the
// test name one host and redirect to the other.
func TestProbeRefusesUndeclaredRedirectHop(t *testing.T) {
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("unreachable"))
	}))
	t.Cleanup(target.Close)
	redirectTo := strings.Replace(target.URL, "127.0.0.1", "localhost", 1)

	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, redirectTo+"/model.onnx", http.StatusFound)
	}))
	t.Cleanup(source.Close)

	modelsDir := t.TempDir()
	pm := NewProbeManager(modelsDir, func() []string { return []string{"127.0.0.1"} }, nil, nil, nil)
	pm.deadline = 5 * time.Second

	id, err := pm.Start(SlotClipEmbed, source.URL+"/model.onnx", "")
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	record := probeSettle(t, pm, id)
	if record.State != ProbeError {
		t.Fatalf("state = %q (reason %q), want %q", record.State, record.Reason, ProbeError)
	}
	if !strings.Contains(record.Reason, "localhost") || !strings.Contains(record.Reason, "not in the allowed list") {
		t.Errorf("reason = %q, want the refused redirect host named", record.Reason)
	}
	if files := probeStagingFiles(t, modelsDir, id); len(files) != 0 {
		t.Errorf("staging left %v, want nothing", files)
	}
}

// TestProbeRecordsDeclaredFromFinalHop pins the honest flag: the first
// hop is outside the list (its provisional flag is false) and the
// redirect lands on a declared host, so the settled record says
// declared -- the host that actually served the bytes decides.
func TestProbeRecordsDeclaredFromFinalHop(t *testing.T) {
	if !RuntimeAvailable() {
		t.Skip("onnxruntime shared library not present")
	}
	payload, err := os.ReadFile(filepath.Join("testdata", "const512.onnx"))
	if err != nil {
		t.Fatalf("read the fixture: %v", err)
	}
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(payload)
	}))
	t.Cleanup(target.Close)
	redirectTo := strings.Replace(target.URL, "127.0.0.1", "localhost", 1)

	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, redirectTo+"/model.onnx", http.StatusFound)
	}))
	t.Cleanup(source.Close)

	pm := NewProbeManager(t.TempDir(), func() []string { return []string{"localhost"} }, nil, nil, nil)
	id, err := pm.Start(SlotClipEmbed, source.URL+"/model.onnx", "")
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	record := probeSettle(t, pm, id)
	if record.State != ProbeOK {
		t.Fatalf("state = %q (reason %q), want %q", record.State, record.Reason, ProbeOK)
	}
	if !record.Declared {
		t.Error("declared = false, want true (the final hop is on the list)")
	}
	if record.Dims != 512 {
		t.Errorf("dims = %d, want 512", record.Dims)
	}
}
