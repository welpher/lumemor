// The inference decode tests (M3a Task 6): the ML jobs consume the
// stored originals as upright RGB rasters, and this decode is the one
// shared path -- the face detection job today, the CLIP and OCR jobs
// of Task 8 through the same helper. The pins: the EXIF orientation is
// applied (an orientation-6 sensor frame decodes upright), the long
// edge is capped without ever upscaling, and the raster honors the
// ml.Image contract.
package media

import (
	"path/filepath"
	"testing"

	"github.com/welpher/lumemor/server/internal/ml"
)

// TestDecodeInferenceImageOrientsAndCaps pins the two transformations
// of the decode: the orientation fix (the 200x100 orientation-6
// fixture decodes as the upright 100x200) and the long-edge bound (the
// 1600x800 fixture capped at 1280 decodes as 1280x640).
func TestDecodeInferenceImageOrientsAndCaps(t *testing.T) {
	dataDir := newMediaEnv(t)

	writeOriginal(t, dataDir, "portrait.jpg", fixtureBytes(t, "portrait-orientation6.jpg"))
	portrait, err := DecodeInferenceImage(filepath.Join(dataDir, "originals", "aa", "bb", "portrait.jpg"), 1280)
	if err != nil {
		t.Fatalf("DecodeInferenceImage(portrait) error = %v, want nil", err)
	}
	if portrait.Width != 100 || portrait.Height != 200 {
		t.Errorf("portrait raster = %dx%d, want the upright 100x200", portrait.Width, portrait.Height)
	}
	requireInferenceRaster(t, portrait)

	writeOriginal(t, dataDir, "large.jpg", fixtureBytes(t, "golden-large.jpg"))
	large, err := DecodeInferenceImage(filepath.Join(dataDir, "originals", "aa", "bb", "large.jpg"), 1280)
	if err != nil {
		t.Fatalf("DecodeInferenceImage(large) error = %v, want nil", err)
	}
	if large.Width != 1280 || large.Height != 640 {
		t.Errorf("large raster = %dx%d, want the capped 1280x640", large.Width, large.Height)
	}
	requireInferenceRaster(t, large)
}

// TestDecodeInferenceImageNeverUpscales pins the bound's direction:
// a raster already inside the bound passes through at its own size.
func TestDecodeInferenceImageNeverUpscales(t *testing.T) {
	dataDir := newMediaEnv(t)

	writeOriginal(t, dataDir, "portrait.jpg", fixtureBytes(t, "portrait-orientation6.jpg"))
	img, err := DecodeInferenceImage(filepath.Join(dataDir, "originals", "aa", "bb", "portrait.jpg"), 5000)
	if err != nil {
		t.Fatalf("DecodeInferenceImage error = %v, want nil", err)
	}
	if img.Width != 100 || img.Height != 200 {
		t.Errorf("raster = %dx%d, want the un-upscaled 100x200", img.Width, img.Height)
	}
	requireInferenceRaster(t, img)
}

// TestDecodeInferenceImageRejectsMissingFile pins the failure shape:
// an undecodable path is an error, never a zero raster.
func TestDecodeInferenceImageRejectsMissingFile(t *testing.T) {
	if _, err := DecodeInferenceImage(filepath.Join(t.TempDir(), "absent.jpg"), 1280); err == nil {
		t.Fatal("DecodeInferenceImage of a missing file error = nil, want the decode error")
	}
}

// requireInferenceRaster checks the ml.Image contract of one decoded
// raster: three bytes per pixel, row-major, no padding.
func requireInferenceRaster(t *testing.T, img ml.Image) {
	t.Helper()
	if len(img.Pixels) != img.Width*img.Height*3 {
		t.Fatalf("raster carries %d bytes, want %d for %dx%d RGB",
			len(img.Pixels), img.Width*img.Height*3, img.Width, img.Height)
	}
}
