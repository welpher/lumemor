package media

import "math"

// The pinned parameters of the face crop thumbnail, the people
// surface's pixels: the geometry the endpoint derives from one face
// row's bounding box against the pixels of the asset's view
// derivative. The derivation is a pure function of (derivative
// pixels, bbox), which is what makes the endpoint's immutable
// caching honest.
const (
	// faceCropEdge is the output edge in pixels: every face crop
	// renders to exactly 256x256, the size the people grids consume.
	// Unlike the serving sizes this is a zoom, not a fit: a window
	// smaller than the edge is upscaled, because a face smaller than
	// 256 pixels still deserves a full grid cell.
	faceCropEdge = 256

	// faceCropMargin is the margin the square crop window carries
	// around the face: the window's side is 1.25x the larger of the
	// bounding box's pixel width and height -- 25% of margin so the
	// crop shows the face's context (hair, chin, shoulders) instead
	// of shaving it.
	faceCropMargin = 1.25
)

// FaceCropWindow computes the square crop window of one face in the
// coordinate space of a source image of imgW x imgH pixels, for a
// face whose bounding box is given in per-mille units of the same
// image (the faces table's bbox_x/y/w/h columns). The view derivative
// preserves the original's framing exactly, so the per-mille box
// addresses the derivative's own pixels directly -- no original-side
// conversion ever happens.
//
// The window is the square the endpoint crops: the side is
// faceCropMargin x the larger of the box's pixel dimensions, capped at
// the image's smaller edge (a face larger than the image cannot carry
// a window wider than the image); the square centers on the box and
// clamps into the image on both axes, so the returned rectangle always
// satisfies 0 <= left <= imgW-side and 0 <= top <= imgH-side. The
// computation is pure float math on the inputs, rounded once at each
// edge, so the crop of one face row is deterministic across runs,
// hosts and libvips versions. It is exported for the face thumbnail
// endpoint's tests, which pin the exact rectangles -- the same role
// ImageDimensions plays for size assertions.
func FaceCropWindow(imgW, imgH, bboxX, bboxY, bboxW, bboxH int) (left, top, side int) {
	// The per-mille box in source pixels, kept as floats so the
	// centering below never loses the fractional position.
	x := float64(bboxX) / 1000 * float64(imgW)
	y := float64(bboxY) / 1000 * float64(imgH)
	boxW := float64(bboxW) / 1000 * float64(imgW)
	boxH := float64(bboxH) / 1000 * float64(imgH)

	// The square side with its 25% margin, capped at the image's
	// smaller edge; at least one pixel so a degenerate box still
	// crops something rather than nothing.
	sideF := faceCropMargin * math.Max(boxW, boxH)
	sideF = math.Min(sideF, float64(imgW))
	sideF = math.Min(sideF, float64(imgH))
	side = max(1, int(math.Round(sideF)))
	side = min(side, imgW, imgH)

	// Centered on the box, then clamped into the image on both axes.
	left = clampInt(int(math.Round(x+boxW/2))-side/2, 0, imgW-side)
	top = clampInt(int(math.Round(y+boxH/2))-side/2, 0, imgH-side)
	return left, top, side
}

// clampInt bounds v into [lo, hi]. Every caller passes a valid range
// (side <= imgW above guarantees hi >= 0).
func clampInt(v, lo, hi int) int {
	return min(max(v, lo), hi)
}
