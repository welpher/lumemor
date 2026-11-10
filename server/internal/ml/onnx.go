// The onnxruntime runtime seam of the ml package: one process-wide
// environment initialized from the shared library, and a
// session wrapper that pins the load-time half of the dual checksum
// verification -- every model file is digest-checked immediately
// before the library parses it. The library itself is located through
// LUMEMOR_ONNX_LIB or the system candidates below; the dev and
// runtime images install the pinned release into /usr/local/lib.
package ml

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"sync"

	ort "github.com/yalue/onnxruntime_go"
)

// onnxLibVersion is the libonnxruntime release the pinned
// yalue/onnxruntime_go binding pairs with: the binding v1.38.0 bundles
// the onnxruntime 1.31.0 C API headers (ORT_API_VERSION 31), so the
// shared library is the 1.31.0 release. Both Dockerfiles install
// exactly that version.
const onnxLibVersion = "1.31.0"

// onnxLibCandidates lists where the shared library is looked for when
// LUMEMOR_ONNX_LIB is unset, in order: the versioned path the
// containers install, the loader-resolved sonames (libonnxruntime.so.1
// is what ldconfig registers), and the two common homebrew paths for
// host development on macOS.
var onnxLibCandidates = []string{
	"/usr/local/lib/libonnxruntime.so." + onnxLibVersion,
	"libonnxruntime.so.1",
	"libonnxruntime.so",
	"/opt/homebrew/lib/libonnxruntime.dylib",
	"/usr/local/lib/libonnxruntime.dylib",
	"libonnxruntime.dylib",
}

// onnxRuntimeOnce guards the process-wide environment: the shared
// library is loaded exactly once and the first outcome sticks, so
// EnsureRuntime is idempotent and RuntimeAvailable is a pure probe.
var (
	onnxRuntimeOnce sync.Once
	onnxRuntimeErr  error
)

// EnsureRuntime initializes the process-wide onnxruntime environment:
// it resolves the shared library (LUMEMOR_ONNX_LIB wins over the
// system candidates), loads it and creates the environment. It is
// idempotent -- the first call does the work, later calls return the
// first call's outcome -- so every session creation can call it
// unconditionally. The error is sticky: a failed environment never
// half-initializes on a second call.
func EnsureRuntime() error {
	onnxRuntimeOnce.Do(func() {
		onnxRuntimeErr = initOnnxRuntime()
	})
	return onnxRuntimeErr
}

// RuntimeAvailable reports whether the onnxruntime shared library is
// usable in this process. It is the inference tests' skip guard: hosts
// without the library keep the suite green, the dev container runs
// everything for real.
func RuntimeAvailable() bool {
	return EnsureRuntime() == nil
}

// initOnnxRuntime resolves and loads the library. An explicit
// LUMEMOR_ONNX_LIB path must work or initialization fails outright;
// without it every candidate is tried until one loads.
func initOnnxRuntime() error {
	if lib := os.Getenv("LUMEMOR_ONNX_LIB"); lib != "" {
		return loadOnnxLibrary(lib)
	}
	for _, lib := range onnxLibCandidates {
		if err := loadOnnxLibrary(lib); err == nil {
			return nil
		}
	}
	return fmt.Errorf("ml: no usable onnxruntime shared library (tried %v; set LUMEMOR_ONNX_LIB)",
		onnxLibCandidates)
}

// loadOnnxLibrary points the binding at the library and creates the
// environment. A load failure leaves the binding uninitialized, so
// trying the next candidate stays safe.
func loadOnnxLibrary(lib string) error {
	ort.SetSharedLibraryPath(lib)
	if err := ort.InitializeEnvironment(); err != nil {
		return fmt.Errorf("ml: load onnxruntime %s: %w", lib, err)
	}
	return nil
}

// Session is one loaded onnxruntime model: a dynamic session plus the
// graph's input and output metadata. Construct with NewSession; call
// Close when done. Run feeds values through the model.
type Session struct {
	inner   *ort.DynamicAdvancedSession
	inputs  []ort.InputOutputInfo
	outputs []ort.InputOutputInfo
	path    string
}

// NewSession loads the ONNX model at path, but only after its bytes
// match the pinned sha256 digest -- the load-time half of the dual
// checksum verification (the download time verified the same digest,
// so any drift in between is caught here). sha256 must be the pinned
// digest, never empty. On mismatch the model is refused with
// ErrInvalidModel; the runtime is initialized only after the digest
// check, so the gate holds on hosts without the library.
//
// Input and output names are derived from the model itself, so callers
// never hardcode them: Inputs and Outputs expose them together with
// the graphs' shapes.
func NewSession(path, sha256 string) (*Session, error) {
	if err := verifyModelDigest(path, sha256); err != nil {
		return nil, err
	}
	if err := EnsureRuntime(); err != nil {
		return nil, err
	}

	inputs, outputs, err := ort.GetInputOutputInfo(path)
	if err != nil {
		return nil, fmt.Errorf("ml: inspect %s: %w", path, err)
	}
	inputNames := make([]string, len(inputs))
	for i, info := range inputs {
		inputNames[i] = info.Name
	}
	outputNames := make([]string, len(outputs))
	for i, info := range outputs {
		outputNames[i] = info.Name
	}

	inner, err := ort.NewDynamicAdvancedSession(path, inputNames, outputNames, nil)
	if err != nil {
		return nil, fmt.Errorf("ml: load session %s: %w", path, err)
	}
	return &Session{
		inner:   inner,
		inputs:  inputs,
		outputs: outputs,
		path:    path,
	}, nil
}

// Inputs returns the model's input metadata: name, shape and element
// type, in the order Run expects the input values.
func (s *Session) Inputs() []ort.InputOutputInfo {
	return s.inputs
}

// Outputs returns the model's output metadata: name, shape and element
// type, in the order Run expects the output values.
func (s *Session) Outputs() []ort.InputOutputInfo {
	return s.outputs
}

// Path is the file the session was loaded from.
func (s *Session) Path() string {
	return s.path
}

// Close releases the session's onnxruntime resources. A closed session
// must not be run again.
func (s *Session) Close() error {
	if s.inner == nil {
		return nil
	}
	return s.inner.Destroy()
}

// Run feeds the input values through the model and fills outputs. The
// slices' lengths must match the model's input and output counts; a
// nil entry in outputs asks onnxruntime to allocate that tensor, which
// Run replaces with the Go-managed value the caller must Destroy.
//
// The call is panic-contained: an onnxruntime fault surfaces as an
// error instead of taking the worker process down. The ML jobs run
// after asset readiness, so a panic must never become a crash.
func (s *Session) Run(inputs, outputs []ort.Value) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("ml: onnxruntime panicked in Run(%s): %v", s.path, r)
		}
	}()
	return s.inner.Run(inputs, outputs)
}

// verifyModelDigest computes the file's sha256 and compares it with
// the pinned digest. A missing file, an empty pin or a drift all
// answer ErrInvalidModel.
func verifyModelDigest(path, pinned string) error {
	if pinned == "" {
		return fmt.Errorf("%w: %s has no pinned sha256", ErrInvalidModel, path)
	}
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("%w: open %s: %w", ErrInvalidModel, path, err)
	}
	defer func() { _ = file.Close() }() // read-only handle

	digest := sha256.New()
	if _, err := io.Copy(digest, file); err != nil {
		return fmt.Errorf("%w: read %s: %w", ErrInvalidModel, path, err)
	}
	if got := hex.EncodeToString(digest.Sum(nil)); got != pinned {
		return fmt.Errorf("%w: %s sha256 %s, want the pinned %s", ErrInvalidModel, path, got, pinned)
	}
	return nil
}
