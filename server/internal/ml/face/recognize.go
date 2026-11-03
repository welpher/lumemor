// The face recognition wrappers (M3a Task 7): the alignment warp
// that carries a detected face's five landmarks onto the canonical
// positions of the 112x112 aligned crop, and the embedding wrapper
// that feeds the aligned crop to a recognition graph and answers the
// L2-normalized feature vector the clustering consumes.
//
// The semantics are understood from the recognition family's public
// demo and implemented here from scratch: the aligned crop is the
// 112x112 canvas whose eyes, nose tip and mouth corners sit at the
// canonical positions below; the warp solving the five point
// correspondences is the least-squares similarity transform
// (rotation, uniform scale, translation -- the four degrees of
// freedom a similarity carries, fit in closed form over the five
// pairs); the graph itself consumes a float NCHW raster at 0-255
// amplitude in RGB channel order (unlike the detection family's BGR
// inputs -- each family follows its own training convention), and
// its feature becomes a unit vector here so the cosine distance of
// the clusterer is a plain dot product away.
package mlface

import (
	"fmt"
	"math"

	ort "github.com/yalue/onnxruntime_go"

	"github.com/welpher/lumemor/server/internal/ml"
)

// alignedSize is the aligned crop's edge: 112 pixels on the side,
// the geometry the recognition family's graphs are trained on.
const alignedSize = 112

// alignedLandmarks is the canonical five-point template of the
// aligned crop, in the detector's landmark order (right eye, left
// eye, nose tip, right mouth corner, left mouth corner): the
// positions a face's landmarks are warped onto.
var alignedLandmarks = [5][2]float64{
	{38.2946, 51.6963}, // right eye
	{73.5318, 51.5014}, // left eye
	{56.0252, 71.7366}, // nose tip
	{41.5493, 92.3655}, // right mouth corner
	{70.7299, 92.2041}, // left mouth corner
}

// AlignCrop warps the image so the five landmarks land on the
// canonical template of the 112x112 aligned crop: the
// least-squares similarity transform from the landmarks onto the
// template, applied by inverse mapping (every output pixel samples
// the source at the transform's inverse, bilinearly). The output is
// the recognizer's input contract; the caller owns the landmark
// coordinate space, which must be the image's own pixels (the
// recognition job decodes its original at the same bound the
// detection job used, so the detector's landmarks apply directly).
//
// Sampling notes, pinned by the wrapper's tests: the identity case
// (landmarks already on the template over a 112x112 raster)
// reproduces the input byte-for-byte, and a uniform integer scale
// lands every output pixel on an exact source pixel. Source samples
// outside the raster clamp to its edge (the warp of a landmark set
// near an image border never reads out of bounds and never fills
// black bands).
func AlignCrop(img ml.Image, landmarks [5][2]float64) (ml.Image, error) {
	if err := validateRaster(img); err != nil {
		return ml.Image{}, err
	}
	warp, err := similarityOf(landmarks, alignedLandmarks)
	if err != nil {
		return ml.Image{}, err
	}
	inverse := warp.inverse()

	out := ml.Image{Width: alignedSize, Height: alignedSize, Pixels: make([]byte, alignedSize*alignedSize*3)}
	for y := 0; y < alignedSize; y++ {
		for x := 0; x < alignedSize; x++ {
			sx, sy := inverse.apply(float64(x), float64(y))
			// A sample the inverse lands outside the source clamps to
			// its edge (replication, never black bands).
			sx = clampF(sx, 0, float64(img.Width-1))
			sy = clampF(sy, 0, float64(img.Height-1))
			x0, x1, wx := neighbors(sx, img.Width)
			y0, y1, wy := neighbors(sy, img.Height)
			i := (y*alignedSize + x) * 3
			for c := 0; c < 3; c++ {
				top := mix(pixelAt(img, x0, y0, c), pixelAt(img, x1, y0, c), wx)
				bottom := mix(pixelAt(img, x0, y1, c), pixelAt(img, x1, y1, c), wx)
				out.Pixels[i+c] = byte(mix(top, bottom, wy) + 0.5)
			}
		}
	}
	return out, nil
}

// clampF bounds value into [lo, hi].
func clampF(value, lo, hi float64) float64 {
	if value < lo {
		return lo
	}
	if value > hi {
		return hi
	}
	return value
}

// similarity is a 2-D similarity transform (rotation, uniform scale,
// translation) in the affine form
//
//	u = a*x - b*y + tx
//	v = b*x + a*y + ty
//
// where (a, b) is the rotation-scale pair: a = s*cos, b = s*sin.
type similarity struct {
	a, b, tx, ty float64
}

// similarityOf solves the least-squares similarity transform mapping
// source onto target: the closed-form fit over the point pairs,
// exact for five landmarks of a real face. The centroids carry the
// translation; the centered cross-correlations carry the rotation
// and scale.
func similarityOf(source, target [5][2]float64) (similarity, error) {
	var scx, scy, tcx, tcy float64
	for i := range source {
		scx += source[i][0]
		scy += source[i][1]
		tcx += target[i][0]
		tcy += target[i][1]
	}
	scx /= 5
	scy /= 5
	tcx /= 5
	tcy /= 5

	var xx, cross, skew float64
	for i := range source {
		dx := source[i][0] - scx
		dy := source[i][1] - scy
		du := target[i][0] - tcx
		dv := target[i][1] - tcy
		xx += dx*dx + dy*dy
		cross += dx*du + dy*dv
		skew += dx*dv - dy*du
	}
	if xx == 0 {
		return similarity{}, fmt.Errorf("mlface: the five landmarks coincide; the alignment transform is undefined")
	}
	warp := similarity{
		a: cross / xx,
		b: skew / xx,
	}
	if warp.a*warp.a+warp.b*warp.b == 0 {
		return similarity{}, fmt.Errorf("mlface: the five landmarks carry no scale or rotation; the alignment transform is undefined")
	}
	warp.tx = tcx - (warp.a*scx - warp.b*scy)
	warp.ty = tcy - (warp.b*scx + warp.a*scy)
	return warp, nil
}

// inverse is the transform's inverse, the map the warp's sampler
// needs (output pixels ask for their source): inverting the
// rotation-scale pair and rotating the translation through it.
func (s similarity) inverse() similarity {
	det := s.a*s.a + s.b*s.b
	return similarity{
		a:  s.a / det,
		b:  -s.b / det,
		tx: (-s.a*s.tx - s.b*s.ty) / det,
		ty: (s.b*s.tx - s.a*s.ty) / det,
	}
}

// apply maps one point through the transform.
func (s similarity) apply(x, y float64) (float64, float64) {
	return s.a*x - s.b*y + s.tx, s.b*x + s.a*y + s.ty
}

// Embed feeds one aligned crop to the recognition graph and answers
// its feature vector, L2-normalized: the graph's own output
// convention is a direction, and the unit norm is the wrapper's
// contract, so the clusterer's cosine distance is one dot product
// away.
//
// The input geometry follows the session's own metadata, never any
// manifest declaration: a fixed-geometry graph gets the crop resized
// to exactly what it declares, and a dynamic-geometry graph (the
// wiring tests' fixture family) gets the crop's own size -- the
// graph says it accepts anything, so it receives the aligned 112x112
// raster as it is. The channel order is the recognition family's
// RGB at 0-255 amplitude in NCHW float layout.
func Embed(sess *ml.Session, crop ml.Image) ([]float32, error) {
	if err := validateRaster(crop); err != nil {
		return nil, err
	}
	if len(sess.Outputs()) != 1 {
		return nil, fmt.Errorf("mlface: the recognition graph carries %d outputs, want 1", len(sess.Outputs()))
	}
	width, height, err := embedGeometry(sess, crop)
	if err != nil {
		return nil, err
	}

	tensor, err := ort.NewTensor(ort.NewShape(1, 3, int64(height), int64(width)),
		rgbCHW(resizeRaster(crop, width, height)))
	if err != nil {
		return nil, fmt.Errorf("mlface: build the embedding input tensor: %w", err)
	}
	defer func() { _ = tensor.Destroy() }()

	outputs := make([]ort.Value, 1)
	if err := sess.Run([]ort.Value{tensor}, outputs); err != nil {
		return nil, fmt.Errorf("mlface: run the recognizer: %w", err)
	}
	defer func() { _ = outputs[0].Destroy() }()

	name := sess.Outputs()[0].Name
	feature, err := tensorData(outputs[0], name)
	if err != nil {
		return nil, err
	}
	if len(feature) == 0 {
		return nil, fmt.Errorf("mlface: the recognizer %s emitted an empty feature", name)
	}

	norm := 0.0
	for _, component := range feature {
		norm += float64(component) * float64(component)
	}
	if norm == 0 {
		return nil, fmt.Errorf("mlface: the recognizer %s emitted a zero vector; it cannot be normalized", name)
	}
	scale := float32(1 / math.Sqrt(norm))
	unit := make([]float32, len(feature))
	for i, component := range feature {
		unit[i] = component * scale
	}
	return unit, nil
}

// embedGeometry resolves the tensor geometry for one crop: the
// session's declared input when the graph pins it, the crop's own
// size when the graph leaves it dynamic. The input must be the
// single NCHW image tensor.
func embedGeometry(sess *ml.Session, crop ml.Image) (width, height int, err error) {
	inputs := sess.Inputs()
	if len(inputs) != 1 {
		return 0, 0, fmt.Errorf("mlface: the recognition graph carries %d inputs, want 1", len(inputs))
	}
	dims := inputs[0].Dimensions
	if len(dims) != 4 {
		return 0, 0, fmt.Errorf("mlface: the recognizer input has %d dimensions, want NCHW (4)", len(dims))
	}
	if dims[0] != 1 && dims[0] > 0 {
		return 0, 0, fmt.Errorf("mlface: the recognizer input batch is %d, want 1", dims[0])
	}
	if dims[1] != 3 {
		return 0, 0, fmt.Errorf("mlface: the recognizer input carries %d channels, want 3", dims[1])
	}
	if dims[2] > 0 && dims[3] > 0 {
		return int(dims[3]), int(dims[2]), nil
	}
	// Dynamic extent: the graph accepts any geometry, so the crop's
	// own -- the aligned raster the caller already produced.
	return crop.Width, crop.Height, nil
}

// rgbCHW flattens an RGB raster into the graph's channel-first float
// tensor data at 0-255 amplitude -- the recognition family's input
// convention (it was trained on RGB imagery; the detection family's
// graphs, fed through their own wrapper, take BGR instead).
func rgbCHW(img ml.Image) []float32 {
	plane := img.Width * img.Height
	data := make([]float32, 3*plane)
	for i := 0; i < plane; i++ {
		data[i] = float32(img.Pixels[i*3])
		data[plane+i] = float32(img.Pixels[i*3+1])
		data[2*plane+i] = float32(img.Pixels[i*3+2])
	}
	return data
}
