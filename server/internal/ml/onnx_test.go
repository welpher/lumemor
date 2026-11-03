// The onnxruntime session wrapper tests (M3a Task 2): the checksum
// gate runs before any library load, so its rejections are pinned on
// every host; the runtime guard and the identity round trip need the
// shared library, so they carry the skip guard that keeps the host
// suite green while `make docker-test` runs them for real inside the
// dev container.
package ml

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"testing"

	ort "github.com/yalue/onnxruntime_go"
)

// identityFixtureSHA256 pins the committed echo fixture by digest:
// NewSession refuses to load bytes that drifted from it.
const identityFixtureSHA256 = "7f6a5659d2eb72e8fa03a0a7eabad28c8e529bfb7f26106911e8617b4455be3b"

// TestNewSessionRejectsChecksumMismatch pins the load-time half of the
// dual checksum verification: the digest is computed from the actual
// bytes on disk, so a file that drifted from its pinned sha256 is
// refused with ErrInvalidModel. The check runs before the onnxruntime
// library is touched, so the pin holds on hosts without the library.
func TestNewSessionRejectsChecksumMismatch(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tampered-identity.onnx")
	data, err := os.ReadFile(filepath.Join("testdata", "identity.onnx"))
	if err != nil {
		t.Fatalf("read identity fixture: %v", err)
	}
	data[len(data)/2] ^= 0xff
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write tampered fixture: %v", err)
	}

	_, err = NewSession(path, identityFixtureSHA256)
	if !errors.Is(err, ErrInvalidModel) {
		t.Errorf("NewSession on drifted bytes error = %v, want ErrInvalidModel", err)
	}
}

// TestNewSessionRejectsUnpinnedModel pins that an empty digest is not
// a way around the gate: a model without a pinned checksum is refused
// rather than loaded unverified.
func TestNewSessionRejectsUnpinnedModel(t *testing.T) {
	_, err := NewSession(filepath.Join("testdata", "identity.onnx"), "")
	if !errors.Is(err, ErrInvalidModel) {
		t.Errorf("NewSession with empty sha256 error = %v, want ErrInvalidModel", err)
	}
}

// TestNewSessionRejectsMissingFile pins that a dangling path is
// ErrInvalidModel, not a library error.
func TestNewSessionRejectsMissingFile(t *testing.T) {
	_, err := NewSession(filepath.Join(t.TempDir(), "absent.onnx"), identityFixtureSHA256)
	if !errors.Is(err, ErrInvalidModel) {
		t.Errorf("NewSession on missing file error = %v, want ErrInvalidModel", err)
	}
}

// TestEnsureRuntimeIsIdempotent pins the once-semantics of the
// environment: repeated calls agree (available stays available,
// unavailable stays unavailable with the same answer), and
// RuntimeAvailable reports exactly what EnsureRuntime resolved. The
// guard is what the inference tests skip on, so a flipped answer would
// both break skips and double-initialize the library.
func TestEnsureRuntimeIsIdempotent(t *testing.T) {
	first := EnsureRuntime()
	second := EnsureRuntime()
	if (first == nil) != (second == nil) {
		t.Errorf("EnsureRuntime flipped between calls: first = %v, second = %v", first, second)
	}
	if got := RuntimeAvailable(); got != (first == nil) {
		t.Errorf("RuntimeAvailable() = %v, want %v (what EnsureRuntime resolved)", got, first == nil)
	}
}

// TestSessionIdentityRoundtrip pins the session surface against the
// committed echo fixture: loading derives the graph's input and output
// names, Run copies the input tensor through the real onnxruntime
// library into an auto-allocated output tensor, and the data comes
// back byte-equal. This is the wiring every inference wrapper builds
// on, so it only runs where the shared library exists.
func TestSessionIdentityRoundtrip(t *testing.T) {
	if !RuntimeAvailable() {
		t.Skip("onnxruntime shared library not present")
	}

	sess, err := NewSession(filepath.Join("testdata", "identity.onnx"), identityFixtureSHA256)
	if err != nil {
		t.Fatalf("NewSession(identity.onnx) error = %v, want nil", err)
	}
	defer func() { _ = sess.Close() }()

	inputs := sess.Inputs()
	if len(inputs) != 1 {
		t.Fatalf("session reports %d inputs, want 1", len(inputs))
	}
	if inputs[0].Name != "input" {
		t.Errorf("input name = %q, want %q", inputs[0].Name, "input")
	}
	outputs := sess.Outputs()
	if len(outputs) != 1 {
		t.Fatalf("session reports %d outputs, want 1", len(outputs))
	}
	if outputs[0].Name != "output" {
		t.Errorf("output name = %q, want %q", outputs[0].Name, "output")
	}

	want := []float32{0.25, -0.5, 1.5, 3, -2}
	input, err := ort.NewTensor(ort.NewShape(int64(len(want))), want)
	if err != nil {
		t.Fatalf("NewTensor error = %v, want nil", err)
	}
	defer func() { _ = input.Destroy() }()

	// A nil output value asks onnxruntime to allocate the output; Run
	// replaces it with the Go-managed tensor.
	outputs2 := []ort.Value{nil}
	if err := sess.Run([]ort.Value{input}, outputs2); err != nil {
		t.Fatalf("Run error = %v, want nil", err)
	}
	t.Cleanup(func() { _ = outputs2[0].Destroy() })

	tensor, ok := outputs2[0].(*ort.Tensor[float32])
	if !ok {
		t.Fatalf("auto-allocated output is %T, want *ort.Tensor[float32]", outputs2[0])
	}
	if got := tensor.GetShape(); !slices.Equal(got, ort.NewShape(int64(len(want)))) {
		t.Errorf("output shape = %v, want %v", got, ort.NewShape(int64(len(want))))
	}
	if got := tensor.GetData(); !slices.Equal(got, want) {
		t.Errorf("identity output = %v, want the input %v echoed back", got, want)
	}

	if err := sess.Close(); err != nil {
		t.Errorf("Close error = %v, want nil", err)
	}
}

// TestSessionRunRecoversPanics pins the wrapper's panic containment:
// an onnxruntime call that panics (here: a session whose underlying
// handle was never created) surfaces as an error instead of taking the
// worker process down. The ML jobs run after asset readiness, so a
// panic must never become a crash.
func TestSessionRunRecoversPanics(t *testing.T) {
	sess := &Session{}
	err := sess.Run(nil, nil)
	if err == nil {
		t.Fatal("Run on an uninitialized session returned nil error, want a recovered-panic error")
	}
}
