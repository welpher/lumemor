// The CLIP embedding wrapper tests (M3a Task 8): the wiring pins over
// the committed const512 fixture (a dynamic-geometry graph emitting a
// non-unit constant vector, so the wrapper's defensive L2
// normalization is observable -- the real vision tower already emits
// unit vectors, and the wrapper normalizes anyway), the input-contract
// refusals, and the env-gated real-model smoke.
//
// Hosts without the onnxruntime shared library skip the inference
// tests; the containerized `make docker-test` gate runs everything for
// real.
package mlclip

import (
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/welpher/lumemor/server/internal/ml"
)

// The measured graph facts of the committed fixtures and the default
// group's real vision tower, pinned by digest.
const (
	const512FixtureSHA256 = "92d6b93b89bf6ccec24222be86689b6ff18370503d3d71316d5d325bd2116766"
	identityFixtureSHA256 = "7f6a5659d2eb72e8fa03a0a7eabad28c8e529bfb7f26106911e8617b4455be3b"
	clipWeightsSHA256     = "980020bf226e528a202de26c6a6186df125b74c612b09cfcd797a082e046b628"
)

// loadConst512 opens a session over the committed constant-embedding
// fixture, with the library skip guard every real-inference test
// carries.
func loadConst512(t *testing.T) *ml.Session {
	t.Helper()
	if !ml.RuntimeAvailable() {
		t.Skip("onnxruntime shared library not present; run the suite inside the dev container (make docker-test)")
	}
	sess, err := ml.NewSession(filepath.Join("..", "testdata", "const512.onnx"), const512FixtureSHA256)
	if err != nil {
		t.Fatalf("NewSession(const512) error = %v, want nil", err)
	}
	t.Cleanup(func() { _ = sess.Close() })
	return sess
}

// gradientImage builds a deterministic RGB raster whose channels are
// linear in x and y.
func gradientImage(width, height int) ml.Image {
	img := ml.Image{Width: width, Height: height, Pixels: make([]byte, width*height*3)}
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			i := (y*width + x) * 3
			img.Pixels[i] = byte(x)
			img.Pixels[i+1] = byte(y)
			img.Pixels[i+2] = byte(255 - x)
		}
	}
	return img
}

// TestEmbedImageNormalizesFixtureOutput pins the wrapper's wiring and
// its defensive L2 normalization: the const512 graph emits 512 ones (a
// vector of norm sqrt(512), deliberately NOT unit length), and
// EmbedImage answers the unit vector along the same direction -- 512
// components, each 1/sqrt(512), norm 1. The graph's input geometry is
// dynamic, so the raster's own size feeds the tensor: both a
// 224x224-sized raster (the real tower's geometry) and a foreign-size
// one run through the same graph.
func TestEmbedImageNormalizesFixtureOutput(t *testing.T) {
	sess := loadConst512(t)

	for _, img := range []ml.Image{gradientImage(224, 224), gradientImage(64, 48)} {
		embedding, err := EmbedImage(sess, img)
		if err != nil {
			t.Fatalf("EmbedImage error = %v, want nil", err)
		}
		if len(embedding) != 512 {
			t.Fatalf("embedding dimension = %d, want 512 (the graph's output)", len(embedding))
		}
		want := float32(1 / math.Sqrt(512))
		norm := 0.0
		for _, component := range embedding {
			norm += float64(component) * float64(component)
			if diff := math.Abs(float64(component - want)); diff > 1e-6 {
				t.Fatalf("component = %v, want the shared unit fraction %v", component, want)
			}
		}
		if diff := math.Abs(norm - 1); diff > 1e-5 {
			t.Errorf("L2 norm = %v, want 1 (the wrapper normalizes defensively)", norm)
		}
	}
}

// TestEmbedImageRejectsForeignGraph pins the honest refusal: a session
// whose input is not an NCHW image tensor (the identity fixture echoes
// a rank-1 vector) answers an error, never a misfed tensor.
func TestEmbedImageRejectsForeignGraph(t *testing.T) {
	if !ml.RuntimeAvailable() {
		t.Skip("onnxruntime shared library not present")
	}
	identity, err := ml.NewSession(filepath.Join("..", "testdata", "identity.onnx"), identityFixtureSHA256)
	if err != nil {
		t.Fatalf("NewSession(identity) error = %v, want nil", err)
	}
	t.Cleanup(func() { _ = identity.Close() })

	if _, err := EmbedImage(identity, gradientImage(64, 64)); err == nil {
		t.Error("EmbedImage over the identity graph error = nil, want the input-contract refusal")
	}
}

// TestEmbedImageRejectsDegenerateRaster pins the raster contract: the
// refusal precedes any inference.
func TestEmbedImageRejectsDegenerateRaster(t *testing.T) {
	sess := loadConst512(t)

	bad := ml.Image{Width: 4, Height: 3, Pixels: make([]byte, 7)}
	if _, err := EmbedImage(sess, bad); err == nil {
		t.Error("EmbedImage over a short raster error = nil, want the raster refusal")
	}
}

// TestEmbedImageRealWeights is the optional real-model smoke run:
// LUMEMOR_TEST_CLIP_WEIGHTS points at the downloaded vision tower
// (tools/models/cache/ in a checkout), and the test pins the measured
// graph geometry from the session's own metadata (input 1x3x224x224,
// output [batch, 512]) and the embedding contract: 512 dimensions,
// unit norm. The graph's declared geometry is load-bearing here -- a
// tensor of any other shape is refused by the runtime, so the smoke
// passing proves the wrapper fed exactly what the session declared.
func TestEmbedImageRealWeights(t *testing.T) {
	if !ml.RuntimeAvailable() {
		t.Skip("onnxruntime shared library not present")
	}
	path := os.Getenv("LUMEMOR_TEST_CLIP_WEIGHTS")
	if path == "" {
		t.Skip("LUMEMOR_TEST_CLIP_WEIGHTS not set; point it at the downloaded CLIP vision weights to run the real-model smoke")
	}

	sess, err := ml.NewSession(path, clipWeightsSHA256)
	if err != nil {
		t.Fatalf("NewSession(clip) error = %v, want nil", err)
	}
	t.Cleanup(func() { _ = sess.Close() })

	inputs := sess.Inputs()
	// The batch extent is a named dynamic dimension upstream; the
	// binding reports it as -1. The geometry that matters -- three
	// channels at 224x224 -- is fixed.
	if len(inputs) != 1 || len(inputs[0].Dimensions) != 4 ||
		inputs[0].Dimensions[1] != 3 ||
		inputs[0].Dimensions[2] != 224 || inputs[0].Dimensions[3] != 224 {
		t.Fatalf("clip input metadata = %v, want the measured [-1, 3, 224, 224]", inputs)
	}
	outputs := sess.Outputs()
	if len(outputs) != 1 || len(outputs[0].Dimensions) != 2 || outputs[0].Dimensions[1] != 512 {
		t.Fatalf("clip output metadata = %v, want the measured [-1, 512] feature", outputs)
	}

	embedding, err := EmbedImage(sess, gradientImage(500, 625))
	if err != nil {
		t.Fatalf("EmbedImage error = %v, want nil", err)
	}
	if len(embedding) != 512 {
		t.Fatalf("embedding dimension = %d, want 512", len(embedding))
	}
	norm := 0.0
	for _, component := range embedding {
		norm += float64(component) * float64(component)
	}
	if diff := math.Abs(norm - 1); diff > 1e-4 {
		t.Errorf("L2 norm = %v, want 1", norm)
	}
}
