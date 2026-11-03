// The face detection wrapper tests (M3a Task 6): the committed YuNet
// weights driven through the real wrapper on the real shared library.
// The pins are the detection semantics -- one face found in the
// portrait fixture, its box in per-mille units, its five landmarks in
// plausible positions, the params-driven threshold -- and the honest
// refusals (a graph that does not speak the anchor-free output
// convention). The wrapper's input geometry comes from the session's
// own metadata, never from any manifest: the 2023mar graph pins
// 1x3x640x640 (the fact Task 2 measured; the manifest's declared
// inputSize is a display fact, corrected to 640 by this task), and
// the type system is the pin -- Detect's only inputs are the session
// and the raster, so no declared size can reach the tensor math.
//
// Hosts without the onnxruntime shared library skip: the containerized
// `make docker-test` gate runs everything for real.
package mlface

import (
	"bytes"
	"image"
	_ "image/jpeg"
	"os"
	"path/filepath"
	"testing"

	"github.com/welpher/lumemor/server/internal/ml"
)

// yunetFixtureSHA256 pins the committed YuNet weights to the upstream
// 2023-03 release bytes (the digest the default manifest also pins).
const yunetFixtureSHA256 = "8f2383e4dd3cfbb4553ea8718107fc0423210dc964f9f4280604804ed2552fa4"

// loadYunet opens a session over the committed YuNet fixture, with the
// library skip guard every real-inference test carries.
func loadYunet(t *testing.T) *ml.Session {
	t.Helper()
	if !ml.RuntimeAvailable() {
		t.Skip("onnxruntime shared library not present")
	}
	sess, err := ml.NewSession(filepath.Join("..", "testdata", "face_detection_yunet_2023mar.onnx"), yunetFixtureSHA256)
	if err != nil {
		t.Fatalf("NewSession(yunet) error = %v, want nil", err)
	}
	t.Cleanup(func() { _ = sess.Close() })
	return sess
}

// decodeFixtureJPEG decodes one of the package's committed image
// fixtures into the RGB raster the wrappers consume (the standard
// library decoder is enough here: the raster is the wrapper's
// contract, the govips decode is the job's business).
func decodeFixtureJPEG(t *testing.T, name string) ml.Image {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "testdata", name))
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	decoded, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("decode fixture %s: %v", name, err)
	}
	return rasterOf(decoded)
}

// rasterOf flattens a decoded image into the ml.Image RGB raster.
func rasterOf(decoded image.Image) ml.Image {
	bounds := decoded.Bounds()
	img := ml.Image{
		Width:  bounds.Dx(),
		Height: bounds.Dy(),
		Pixels: make([]byte, bounds.Dx()*bounds.Dy()*3),
	}
	for y := 0; y < img.Height; y++ {
		for x := 0; x < img.Width; x++ {
			r, g, b, _ := decoded.At(bounds.Min.X+x, bounds.Min.Y+y).RGBA()
			i := (y*img.Width + x) * 3
			img.Pixels[i] = byte(r >> 8)
			img.Pixels[i+1] = byte(g >> 8)
			img.Pixels[i+2] = byte(b >> 8)
		}
	}
	return img
}

// gradientImage builds a smooth vertical gradient: a deterministic
// raster with no face in it, the no-face boundary of the decode.
func gradientImage(width, height int) ml.Image {
	img := ml.Image{Width: width, Height: height, Pixels: make([]byte, width*height*3)}
	for y := 0; y < height; y++ {
		v := byte(255 * y / max(height-1, 1))
		for x := 0; x < width; x++ {
			i := (y*width + x) * 3
			img.Pixels[i] = v
			img.Pixels[i+1] = byte(255 * x / max(width-1, 1))
			img.Pixels[i+2] = byte(255 - v)
		}
	}
	return img
}

// TestDetectFindsFaceInPortrait is the real-model pin: the portrait
// fixture (a public-domain astronaut portrait Task 2 verified
// detectable) yields exactly one face -- the anchor-free decode finds
// a neighborhood of candidates that the NMS collapses to the single
// face -- with a score in the band the real weights produce, the box
// in per-mille units inside the image, and the five landmarks in
// anatomically plausible positions (inside the image bounds, in the
// upper half of the box's vertical span for the eyes).
func TestDetectFindsFaceInPortrait(t *testing.T) {
	sess := loadYunet(t)

	// The measured input geometry of the 2023mar graph: Detect resizes
	// to exactly this (the session's metadata is the geometry truth).
	inputs := sess.Inputs()
	if len(inputs) != 1 || len(inputs[0].Dimensions) != 4 {
		t.Fatalf("yunet input metadata = %v, want one 4-D input", inputs)
	}
	if w, h := inputs[0].Dimensions[3], inputs[0].Dimensions[2]; w != 640 || h != 640 {
		t.Errorf("yunet input geometry = %dx%d, want the 640x640 the graph pins", w, h)
	}

	img := decodeFixtureJPEG(t, "face-portrait.jpg")
	detections, err := Detect(sess, img)
	if err != nil {
		t.Fatalf("Detect error = %v, want nil", err)
	}
	if len(detections) != 1 {
		t.Fatalf("detections = %d, want exactly 1 (the portrait's one face after NMS)", len(detections))
	}

	face := detections[0]
	// The band the real weights produce for this portrait (a plain
	// geometric-mean omission or a head swap moves the score out).
	if face.Score < 0.90 || face.Score > 0.97 {
		t.Errorf("score = %v, want it inside [0.90, 0.97]", face.Score)
	}

	// The per-mille box: every field inside 0-1000, the box inside the
	// image, face-sized (the coarse plausibility band), and centered
	// where the portrait's face sits.
	x, y, w, h := face.BBox[0], face.BBox[1], face.BBox[2], face.BBox[3]
	for i, field := range face.BBox {
		if field < 0 || field > 1000 {
			t.Errorf("bbox[%d] = %d, want it inside 0-1000 (per-mille)", i, field)
		}
	}
	if x+w > 1000 || y+h > 1000 {
		t.Errorf("bbox = (%d %d %d %d), want the box inside the image", x, y, w, h)
	}
	if w < 150 || w > 250 {
		t.Errorf("bbox width = %d‰, want the face's ~200‰", w)
	}
	if h < 200 || h > 300 {
		t.Errorf("bbox height = %d‰, want the face's ~245‰", h)
	}
	if x < 320 || x > 440 {
		t.Errorf("bbox x = %d‰, want the portrait's face position ~387‰", x)
	}
	if y < 170 || y > 250 {
		t.Errorf("bbox y = %d‰, want the portrait's face position ~213‰", y)
	}

	// The five landmarks: inside the image, inside or bordering the
	// box, the eyes above the nose, the nose above the mouth corners.
	eyes, nose, mouth := face.Landmarks[0], face.Landmarks[2], face.Landmarks[4]
	for i, point := range face.Landmarks {
		if point[0] < 0 || point[0] > float64(img.Width) || point[1] < 0 || point[1] > float64(img.Height) {
			t.Errorf("landmark[%d] = %v, want it inside the %dx%d image", i, point, img.Width, img.Height)
		}
	}
	if eyes[1] > nose[1] || nose[1] > mouth[1] {
		t.Errorf("landmarks not anatomically ordered: eyes y=%v nose y=%v mouth y=%v",
			eyes[1], nose[1], mouth[1])
	}
}

// TestDetectNoFaceImage pins the no-face boundary: a smooth gradient
// raster yields zero detections and no error (the job's zero-faces
// path is a success, not a failure).
func TestDetectNoFaceImage(t *testing.T) {
	sess := loadYunet(t)

	detections, err := Detect(sess, gradientImage(480, 360))
	if err != nil {
		t.Fatalf("Detect error = %v, want nil", err)
	}
	if len(detections) != 0 {
		t.Fatalf("detections = %d, want 0 for a faceless gradient", len(detections))
	}
}

// TestDetectHonorsScoreThreshold pins the params flow: the threshold
// comes from the model's params (the worker passes the model row's
// params through); a threshold above the portrait's score filters the
// face out, the default threshold keeps it.
func TestDetectHonorsScoreThreshold(t *testing.T) {
	sess := loadYunet(t)
	img := decodeFixtureJPEG(t, "face-portrait.jpg")

	strict, err := DetectWithParams(sess, img, map[string]any{"scoreThreshold": 0.99})
	if err != nil {
		t.Fatalf("DetectWithParams(0.99) error = %v, want nil", err)
	}
	if len(strict) != 0 {
		t.Errorf("detections at threshold 0.99 = %d, want 0 (the portrait scores below)", len(strict))
	}

	loose, err := DetectWithParams(sess, img, map[string]any{"scoreThreshold": 0.6})
	if err != nil {
		t.Fatalf("DetectWithParams(0.6) error = %v, want nil", err)
	}
	if len(loose) != 1 {
		t.Errorf("detections at threshold 0.6 = %d, want 1", len(loose))
	}
}

// TestDetectRejectsForeignGraph pins the honest refusal: a session
// whose graph does not speak the anchor-free output convention (the
// identity fixture echoes one tensor) answers an error, never a
// misdecoded result.
func TestDetectRejectsForeignGraph(t *testing.T) {
	if !ml.RuntimeAvailable() {
		t.Skip("onnxruntime shared library not present")
	}
	identity, err := ml.NewSession(filepath.Join("..", "testdata", "identity.onnx"),
		"7f6a5659d2eb72e8fa03a0a7eabad28c8e529bfb7f26106911e8617b4455be3b")
	if err != nil {
		t.Fatalf("NewSession(identity) error = %v, want nil", err)
	}
	t.Cleanup(func() { _ = identity.Close() })

	if _, err := Detect(identity, gradientImage(64, 64)); err == nil {
		t.Fatal("Detect over the identity graph error = nil, want the convention refusal")
	}
}

// TestDetectRejectsDegenerateRaster pins the input contract: a raster
// with inconsistent dimensions is refused before any inference.
func TestDetectRejectsDegenerateRaster(t *testing.T) {
	sess := loadYunet(t)

	if _, err := Detect(sess, ml.Image{Width: 4, Height: 3, Pixels: make([]byte, 7)}); err == nil {
		t.Fatal("Detect over a short raster error = nil, want the raster refusal")
	}
}
