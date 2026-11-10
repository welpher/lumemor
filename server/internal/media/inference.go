//go:build cgo

package media

// The ML inference decode: the ML jobs (face detection, the CLIP and
// OCR jobs) consume the stored originals as
// upright RGB rasters -- the EXIF orientation applied once (the same
// render-time discipline the thumbnail job applies) and the long edge
// bounded, so the heavy decode of a large original never hands a
// full-resolution raster across the libvips boundary. This file is
// the libvips half; the CGO-less stub in inference_nocgo.go keeps the
// package buildable without it.

import (
	"fmt"
	"image"

	"github.com/davidbyttow/govips/v2/vips"

	"github.com/welpher/lumemor/server/internal/ml"
)

// DecodeInferenceImage decodes the stored original at originalPath
// into the upright RGB raster the inference wrappers consume: the
// EXIF orientation applied (a sensor-rotated frame decodes upright),
// the long edge capped at longEdge (aspect preserved, never
// upscaled). The raster's per-mille geometry is identical to the
// original's: only the pixel count is bounded.
func DecodeInferenceImage(originalPath string, longEdge int) (ml.Image, error) {
	if longEdge <= 0 {
		return ml.Image{}, fmt.Errorf("media: inference decode needs a positive long-edge bound, got %d", longEdge)
	}

	// libvips initializes lazily here when the process has not started
	// it explicitly (main.go does at boot); Startup is idempotent.
	if err := vips.Startup(nil); err != nil {
		return ml.Image{}, fmt.Errorf("start libvips: %w", err)
	}

	ref, err := vips.NewImageFromFile(originalPath)
	if err != nil {
		return ml.Image{}, fmt.Errorf("decode %s: %w", originalPath, err)
	}
	defer ref.Close()

	if err := ref.AutoRotate(); err != nil {
		return ml.Image{}, fmt.Errorf("orient %s: %w", originalPath, err)
	}
	if err := ref.ThumbnailWithSize(longEdge, longEdge, vips.InterestingNone, vips.SizeDown); err != nil {
		return ml.Image{}, fmt.Errorf("bound %s to the %dpx long edge: %w", originalPath, longEdge, err)
	}

	// ToGoImage hands back an 8-bit sRGB image.Image (NRGBA for
	// three-band input, Gray for one-band); the flatten below is the
	// small conversion into the wrapper's packed RGB raster.
	decoded, err := ref.ToGoImage()
	if err != nil {
		return ml.Image{}, fmt.Errorf("read pixels of %s: %w", originalPath, err)
	}
	return flattenToRaster(decoded), nil
}

// flattenToRaster converts one decoded Go image into the ml.Image RGB
// raster. The raster is already long-edge bounded by the caller, so
// the generic per-pixel walk costs microseconds.
func flattenToRaster(decoded image.Image) ml.Image {
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
