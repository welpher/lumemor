// The shared raster helpers of the inference wrappers:
// the CLIP and OCR wrappers resize decoded rasters through
// one bilinear implementation. The face wrappers keep
// their own private copy -- consolidating them onto this one is a
// later refactor.
package ml

import "math"

// ResizeRaster resizes the raster to width x height with a bilinear
// filter, the direct (aspect-changing) resize the inference families
// tolerate. A same-size input returns the original raster (the caller
// copies through its tensor anyway). Samples outside the source clamp
// to its edge.
func ResizeRaster(img Image, width, height int) Image {
	if img.Width == width && img.Height == height {
		return img
	}
	out := Image{Width: width, Height: height, Pixels: make([]byte, width*height*3)}
	srcX := float64(img.Width) / float64(width)
	srcY := float64(img.Height) / float64(height)
	for y := 0; y < height; y++ {
		fy := (float64(y) + 0.5) * srcY
		y0, y1, wy := rasterNeighbors(fy-0.5, img.Height)
		for x := 0; x < width; x++ {
			fx := (float64(x) + 0.5) * srcX
			x0, x1, wx := rasterNeighbors(fx-0.5, img.Width)
			for c := 0; c < 3; c++ {
				top := rasterMix(rasterPixelAt(img, x0, y0, c), rasterPixelAt(img, x1, y0, c), wx)
				bottom := rasterMix(rasterPixelAt(img, x0, y1, c), rasterPixelAt(img, x1, y1, c), wx)
				out.Pixels[(y*width+x)*3+c] = byte(rasterMix(top, bottom, wy) + 0.5)
			}
		}
	}
	return out
}

// rasterNeighbors resolves the two source pixels a half-open sample
// falls between, with its weight on the second.
func rasterNeighbors(position float64, extent int) (lo, hi int, weight float64) {
	lo = int(math.Floor(position))
	if lo < 0 {
		lo = 0
	}
	if hi = lo + 1; hi >= extent {
		hi = extent - 1
	}
	weight = position - float64(lo)
	if weight < 0 {
		weight = 0
	}
	if weight > 1 {
		weight = 1
	}
	return lo, hi, weight
}

// rasterPixelAt reads one raster channel.
func rasterPixelAt(img Image, x, y, c int) float64 {
	return float64(img.Pixels[(y*img.Width+x)*3+c])
}

// rasterMix interpolates linearly.
func rasterMix(a, b, t float64) float64 {
	return a*(1-t) + b*t
}
