// The face recognition wrapper tests (M3a Task 7): the alignment
// warp's pixel-level pins -- the canonical five-point template maps
// an aligned raster onto itself byte-for-byte, and a known uniform
// scale lands the output on the exact subsampled source pixels -- and
// the embedding wrapper's wiring pins over the committed const128
// fixture (a dynamic-geometry graph that emits a non-unit constant
// vector, so the wrapper's L2 normalization is observable). The
// real-weight smoke test is env-gated: LUMEMOR_TEST_SFACE_WEIGHTS
// pointing at the downloaded SFace graph runs the whole chain for
// real (the detector's landmarks through the alignment into the
// embedder), anything else skips with a counted skip.
//
// Hosts without the onnxruntime shared library skip the Embed tests;
// the containerized `make docker-test` gate runs everything for real.
package mlface

import (
	"bytes"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/welpher/lumemor/server/internal/ml"
)

// The measured graph facts of the committed fixtures, pinned by
// digest like the detector's weights.
const (
	const128FixtureSHA256 = "e5d4940db6bddc912dee319434394700cf82fd8950cc32e609435ae7f9597263"
	sfaceFixtureSHA256    = "0ba9fbfa01b5270c96627c4ef784da859931e02f04419c829e83484087c34e79"
)

// canonicalLandmarks is the aligned crop's five-point template in the
// detector's landmark order (right eye, left eye, nose tip, right
// mouth corner, left mouth corner): the standard positions of the
// 112x112 aligned face, where a warp that receives them lands the
// identity transform.
var canonicalLandmarks = [5][2]float64{
	{38.2946, 51.6963},
	{73.5318, 51.5014},
	{56.0252, 71.7366},
	{41.5493, 92.3655},
	{70.7299, 92.2041},
}

// scaledLandmarks scales every template point by factor.
func scaledLandmarks(factor float64) [5][2]float64 {
	var out [5][2]float64
	for i, point := range canonicalLandmarks {
		out[i] = [2]float64{point[0] * factor, point[1] * factor}
	}
	return out
}

// patternRaster builds a deterministic raster whose channels are
// linear in x and y (plus a constant blue): a geometry any warp
// assertion can predict exactly.
func patternRaster(width, height int) ml.Image {
	img := ml.Image{Width: width, Height: height, Pixels: make([]byte, width*height*3)}
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			i := (y*width + x) * 3
			img.Pixels[i] = byte(x)
			img.Pixels[i+1] = byte(y)
			img.Pixels[i+2] = 7
		}
	}
	return img
}

// TestAlignCropTemplateMapsToItself is the identity pin: a 112x112
// raster whose five landmarks already sit on the canonical template
// warps onto itself byte-for-byte -- the least-squares similarity
// from the template onto the template is the identity, and the
// sampler reproduces integer source coordinates exactly.
func TestAlignCropTemplateMapsToItself(t *testing.T) {
	img := patternRaster(112, 112)

	aligned, err := AlignCrop(img, canonicalLandmarks)
	if err != nil {
		t.Fatalf("AlignCrop(template) error = %v, want nil", err)
	}
	if aligned.Width != 112 || aligned.Height != 112 {
		t.Fatalf("aligned raster = %dx%d, want 112x112", aligned.Width, aligned.Height)
	}
	if !bytes.Equal(aligned.Pixels, img.Pixels) {
		t.Error("aligned raster differs from the input, want the byte-for-byte identity")
	}
}

// TestAlignCropScalesKnownGeometry pins the warp's scale geometry: a
// 224x224 raster whose landmarks sit at twice the template positions
// aligns through a uniform half-scale, and the half-scale maps every
// output pixel onto the exact source pixel at twice its index -- no
// interpolation blur, because the inverse coordinates are integers.
func TestAlignCropScalesKnownGeometry(t *testing.T) {
	img := patternRaster(224, 224)

	aligned, err := AlignCrop(img, scaledLandmarks(2))
	if err != nil {
		t.Fatalf("AlignCrop(2x) error = %v, want nil", err)
	}
	if aligned.Width != 112 || aligned.Height != 112 {
		t.Fatalf("aligned raster = %dx%d, want 112x112", aligned.Width, aligned.Height)
	}
	for y := 0; y < 112; y++ {
		for x := 0; x < 112; x++ {
			i := (y*112 + x) * 3
			src := ((2*y)*224 + 2*x) * 3
			if aligned.Pixels[i] != img.Pixels[src] ||
				aligned.Pixels[i+1] != img.Pixels[src+1] ||
				aligned.Pixels[i+2] != img.Pixels[src+2] {
				t.Fatalf("aligned pixel (%d, %d) = %v, want the source pixel (%d, %d) = %v",
					x, y, aligned.Pixels[i:i+3], 2*x, 2*y, img.Pixels[src:src+3])
			}
		}
	}
}

// TestAlignCropRejectsDegenerateLandmarks pins the refusal: five
// identical landmarks carry no orientation or scale to solve for, so
// the similarity transform is undefined and the warp refuses instead
// of dividing by zero.
func TestAlignCropRejectsDegenerateLandmarks(t *testing.T) {
	same := [5][2]float64{{50, 50}, {50, 50}, {50, 50}, {50, 50}, {50, 50}}
	if _, err := AlignCrop(patternRaster(112, 112), same); err == nil {
		t.Error("AlignCrop over identical landmarks error = nil, want the degeneracy refusal")
	}
}

// TestAlignCropRejectsDegenerateRaster pins the input contract: a
// raster whose pixel count contradicts its dimensions is refused
// before any warp work.
func TestAlignCropRejectsDegenerateRaster(t *testing.T) {
	bad := ml.Image{Width: 4, Height: 3, Pixels: make([]byte, 7)}
	if _, err := AlignCrop(bad, canonicalLandmarks); err == nil {
		t.Error("AlignCrop over a short raster error = nil, want the raster refusal")
	}
}

// loadConst128 opens a session over the committed constant-embedding
// fixture, with the library skip guard every real-inference test
// carries.
func loadConst128(t *testing.T) *ml.Session {
	t.Helper()
	if !ml.RuntimeAvailable() {
		t.Skip("onnxruntime shared library not present; run the suite inside the dev container (make docker-test)")
	}
	sess, err := ml.NewSession(filepath.Join("..", "testdata", "const128.onnx"), const128FixtureSHA256)
	if err != nil {
		t.Fatalf("NewSession(const128) error = %v, want nil", err)
	}
	t.Cleanup(func() { _ = sess.Close() })
	return sess
}

// TestEmbedNormalizesFixtureOutput pins the wrapper's wiring and its
// L2 normalization: the const128 graph emits 128 ones (a vector of
// norm sqrt(128), deliberately NOT unit length), and Embed answers
// the unit vector along the same direction -- 128 components, each
// 1/sqrt(128), norm 1. The graph's input geometry is dynamic, so the
// crop's own size feeds the tensor: both an aligned-size crop and a
// foreign-size crop run through the same graph.
func TestEmbedNormalizesFixtureOutput(t *testing.T) {
	sess := loadConst128(t)

	for _, crop := range []ml.Image{gradientImage(112, 112), gradientImage(64, 48)} {
		embedding, err := Embed(sess, crop)
		if err != nil {
			t.Fatalf("Embed error = %v, want nil", err)
		}
		if len(embedding) != 128 {
			t.Fatalf("embedding dimension = %d, want 128 (the graph's output)", len(embedding))
		}
		want := float32(1 / math.Sqrt(128))
		norm := 0.0
		for _, component := range embedding {
			norm += float64(component) * float64(component)
			if diff := math.Abs(float64(component - want)); diff > 1e-6 {
				t.Fatalf("component = %v, want the shared unit fraction %v", component, want)
			}
		}
		if diff := math.Abs(norm - 1); diff > 1e-5 {
			t.Errorf("L2 norm = %v, want 1 (the wrapper normalizes)", norm)
		}
	}
}

// TestEmbedRejectsForeignGraph pins the honest refusal: a session
// whose input is not an NCHW image tensor (the identity fixture
// echoes a rank-1 vector) answers an error, never a misfed tensor.
func TestEmbedRejectsForeignGraph(t *testing.T) {
	if !ml.RuntimeAvailable() {
		t.Skip("onnxruntime shared library not present")
	}
	identity, err := ml.NewSession(filepath.Join("..", "testdata", "identity.onnx"),
		"7f6a5659d2eb72e8fa03a0a7eabad28c8e529bfb7f26106911e8617b4455be3b")
	if err != nil {
		t.Fatalf("NewSession(identity) error = %v, want nil", err)
	}
	t.Cleanup(func() { _ = identity.Close() })

	if _, err := Embed(identity, gradientImage(64, 64)); err == nil {
		t.Error("Embed over the identity graph error = nil, want the input-contract refusal")
	}
}

// TestEmbedRejectsDegenerateRaster pins the raster contract on the
// embedding path too: the refusal precedes any inference.
func TestEmbedRejectsDegenerateRaster(t *testing.T) {
	sess := loadConst128(t)

	bad := ml.Image{Width: 4, Height: 3, Pixels: make([]byte, 7)}
	if _, err := Embed(sess, bad); err == nil {
		t.Error("Embed over a short raster error = nil, want the raster refusal")
	}
}

// TestEmbedRealSFaceWeights is the optional real-model smoke run:
// LUMEMOR_TEST_SFACE_WEIGHTS points at the downloaded SFace graph
// (tools/models/cache/ in a checkout), and the test drives the whole
// recognition chain for real -- the portrait's detected landmarks
// through the alignment warp into the embedder -- pinning the
// measured graph geometry (input 1x3x112x112, output 128), the
// 128-dimension unit-norm embedding, and the framing stability the
// alignment exists for: the same face, re-rendered turned and
// rescaled, embeds close to itself. (The EXIF-orientation twin of
// the portrait belongs to the jobs-level tests, which decode through
// the production libvips path; this package's fixtures decode
// tag-blind.)
func TestEmbedRealSFaceWeights(t *testing.T) {
	if !ml.RuntimeAvailable() {
		t.Skip("onnxruntime shared library not present")
	}
	path := os.Getenv("LUMEMOR_TEST_SFACE_WEIGHTS")
	if path == "" {
		t.Skip("LUMEMOR_TEST_SFACE_WEIGHTS not set; point it at the downloaded SFace weights to run the real-model smoke")
	}

	sess, err := ml.NewSession(path, sfaceFixtureSHA256)
	if err != nil {
		t.Fatalf("NewSession(sface) error = %v, want nil", err)
	}
	t.Cleanup(func() { _ = sess.Close() })

	// The measured graph geometry: the session's own metadata, never
	// any manifest declaration (the type system pins it -- Embed's
	// inputs are the session and the raster).
	inputs := sess.Inputs()
	if len(inputs) != 1 || len(inputs[0].Dimensions) != 4 ||
		inputs[0].Dimensions[0] != 1 || inputs[0].Dimensions[1] != 3 ||
		inputs[0].Dimensions[2] != 112 || inputs[0].Dimensions[3] != 112 {
		t.Fatalf("sface input metadata = %v, want the measured 1x3x112x112", inputs)
	}
	outputs := sess.Outputs()
	if len(outputs) != 1 || len(outputs[0].Dimensions) != 2 ||
		outputs[0].Dimensions[0] != 1 || outputs[0].Dimensions[1] != 128 {
		t.Fatalf("sface output metadata = %v, want the measured 1x128 feature", outputs)
	}

	detector := loadYunet(t)
	embedPortrait := func(img ml.Image, what string) []float32 {
		t.Helper()
		detections, err := Detect(detector, img)
		if err != nil {
			t.Fatalf("Detect(%s) error = %v, want nil", what, err)
		}
		if len(detections) != 1 {
			t.Fatalf("Detect(%s) found %d faces, want 1", what, len(detections))
		}
		crop, err := AlignCrop(img, detections[0].Landmarks)
		if err != nil {
			t.Fatalf("AlignCrop(%s) error = %v, want nil", what, err)
		}
		embedding, err := Embed(sess, crop)
		if err != nil {
			t.Fatalf("Embed(%s) error = %v, want nil", what, err)
		}
		if len(embedding) != 128 {
			t.Fatalf("Embed(%s) dimension = %d, want 128", what, len(embedding))
		}
		norm := 0.0
		for _, component := range embedding {
			norm += float64(component) * float64(component)
		}
		if diff := math.Abs(norm - 1); diff > 1e-5 {
			t.Errorf("Embed(%s) L2 norm = %v, want 1", what, norm)
		}
		return embedding
	}

	// The same face through two framings: the portrait itself, and the
	// portrait re-rendered turned by a few degrees and uniformly
	// smaller -- the alignment's whole purpose is that the landmark
	// warp absorbs exactly this, so the two embeddings stay close
	// (their dot product, both being unit vectors).
	upright := embedPortrait(decodeFixtureJPEG(t, "face-portrait.jpg"), "the portrait")
	reframed := embedPortrait(reframed(decodeFixtureJPEG(t, "face-portrait.jpg"), 4, 0.9), "the reframed portrait")

	similarity := 0.0
	for i := range upright {
		similarity += float64(upright[i]) * float64(reframed[i])
	}
	t.Logf("cosine similarity of the portrait's two framings = %v", similarity)
	if similarity < 0.6 {
		t.Errorf("cosine similarity of the portrait's two framings = %v, want at least 0.6 (the same identity)", similarity)
	}
}

// reframed renders the raster turned by degrees about its center and
// uniformly scaled by factor: the stand-in for a differently framed
// shot of the same face, sampling bilinearly off the source.
func reframed(img ml.Image, degrees, factor float64) ml.Image {
	radians := degrees * math.Pi / 180
	cos, sin := math.Cos(radians), math.Sin(radians)
	cx, cy := float64(img.Width)/2, float64(img.Height)/2
	out := ml.Image{Width: img.Width, Height: img.Height, Pixels: make([]byte, len(img.Pixels))}
	for y := 0; y < img.Height; y++ {
		for x := 0; x < img.Width; x++ {
			dx, dy := float64(x)-cx, float64(y)-cy
			sx := cx + (cos*dx+sin*dy)/factor
			sy := cy + (-sin*dx+cos*dy)/factor
			sx = clampF(sx, 0, float64(img.Width-1))
			sy = clampF(sy, 0, float64(img.Height-1))
			x0, x1, wx := neighbors(sx, img.Width)
			y0, y1, wy := neighbors(sy, img.Height)
			for c := 0; c < 3; c++ {
				top := mix(pixelAt(img, x0, y0, c), pixelAt(img, x1, y0, c), wx)
				bottom := mix(pixelAt(img, x0, y1, c), pixelAt(img, x1, y1, c), wx)
				out.Pixels[(y*img.Width+x)*3+c] = byte(mix(top, bottom, wy) + 0.5)
			}
		}
	}
	return out
}
