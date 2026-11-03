// Package mlface wraps the face detection graphs of the YuNet family
// (the anchor-free graphs OpenCV Zoo publishes as ONNX): it feeds the
// session's own input geometry, decodes the per-stride score / box /
// landmark heads back into image coordinates, and de-duplicates the
// anchor neighborhoods with an overlap suppression pass.
//
// The tensor convention (understood from the public OpenCV Zoo demo
// semantics, implemented here from scratch): the graph consumes one
// 1x3xHxW float raster holding BGR channel order at 0-255 amplitude;
// it emits, for each feature stride s of its three scales, four
// row-per-anchor matrices -- cls_s and obj_s (one confidence channel
// each), bbox_s (four box channels) and kps_s (ten landmark channels).
// H and W come from the session's input metadata, never from a
// manifest declaration: the graph is the geometry authority.
package mlface

import (
	"fmt"
	"math"
	"regexp"
	"sort"
	"strconv"

	ort "github.com/yalue/onnxruntime_go"

	"github.com/welpher/lumemor/server/internal/ml"
)

// Detection is one detected face of the image Detect received.
type Detection struct {
	// BBox is the face's bounding box in per-mille units (0-1000)
	// relative to the image Detect received: x, y, w, h. Per-mille
	// survive every aspect-preserving rescale, which is what the
	// faces table stores and every consumer renders.
	BBox [4]int
	// Score is the detection's confidence in [0, 1]: the geometric
	// mean of the head's two confidence channels.
	Score float32
	// Landmarks are the five face landmarks -- right eye, left eye,
	// nose tip, right mouth corner, left mouth corner -- in pixel
	// coordinates of the image Detect received. They are deliberately
	// NOT part of the persisted face row: the recognition job (Task 7)
	// consumes them to align the crop, at whatever decode scale it
	// owns.
	Landmarks [5][2]float64
}

// The decode defaults. The default group's manifest carries the same
// three values as the yunet model's params; a model whose params
// carry none of them decodes with these.
const (
	// DefaultScoreThreshold is the confidence a candidate needs to
	// survive into the result.
	DefaultScoreThreshold = 0.6
	// DefaultNmsThreshold is the box-overlap ratio above which a
	// lower-scored candidate is suppressed as a duplicate.
	DefaultNmsThreshold = 0.3
	// DefaultTopK caps the returned detections.
	DefaultTopK = 5000
)

// Detect runs the detector with the decode defaults (the params-less
// convenience over DetectWithParams).
func Detect(sess *ml.Session, img ml.Image) ([]Detection, error) {
	return DetectWithParams(sess, img, nil)
}

// DetectWithParams runs the detector over one raster with the model's
// params steering the decode: scoreThreshold (default 0.6),
// nmsThreshold (default 0.3) and topK (default 5000). Malformed param
// values fall back to their defaults -- the params arrived with a
// registered model, so a drifted value degrades the threshold, it
// does not fail the asset's ML work.
//
// The raster is resized to the session's input geometry (a direct
// resize; the family's graphs are fully convolutional and tolerate
// the aspect change), the decoded boxes and landmarks map back into
// the raster's own coordinates, and the per-mille conversion divides
// by the raster's dimensions -- so per-mille values are identical to
// what a full-resolution decode would produce.
func DetectWithParams(sess *ml.Session, img ml.Image, params map[string]any) ([]Detection, error) {
	if err := validateRaster(img); err != nil {
		return nil, err
	}
	geometry, err := readInputGeometry(sess)
	if err != nil {
		return nil, err
	}

	heads, err := runGraph(sess, img, geometry)
	if err != nil {
		return nil, err
	}

	opts := decodeOptionsOf(params)
	faces := decodeFaces(heads, geometry, opts)
	faces = suppressOverlaps(faces, opts)
	if len(faces) > opts.topK {
		faces = faces[:opts.topK]
	}

	detections := make([]Detection, 0, len(faces))
	for _, face := range faces {
		detections = append(detections, toDetection(face, img, geometry))
	}
	return detections, nil
}

// decodeOptions are the steering values of one decode.
type decodeOptions struct {
	scoreThreshold float64
	nmsThreshold   float64
	topK           int
}

// decodeOptionsOf reads the steering values out of the model's params
// object, falling back to the defaults for absent or malformed
// entries.
func decodeOptionsOf(params map[string]any) decodeOptions {
	opts := decodeOptions{
		scoreThreshold: DefaultScoreThreshold,
		nmsThreshold:   DefaultNmsThreshold,
		topK:           DefaultTopK,
	}
	if v, ok := paramFloat(params, "scoreThreshold"); ok {
		opts.scoreThreshold = v
	}
	if v, ok := paramFloat(params, "nmsThreshold"); ok {
		opts.nmsThreshold = v
	}
	if v, ok := paramFloat(params, "topK"); ok && v >= 1 {
		opts.topK = int(v)
	}
	return opts
}

// paramFloat reads one numeric param. JSON numbers arrive as float64;
// integers as int. Anything else is absent.
func paramFloat(params map[string]any, name string) (float64, bool) {
	value, ok := params[name]
	if !ok {
		return 0, false
	}
	switch typed := value.(type) {
	case float64:
		return typed, true
	case int:
		return float64(typed), true
	case int32:
		return float64(typed), true
	case int64:
		return float64(typed), true
	case float32:
		return float64(typed), true
	}
	return 0, false
}

// inputGeometry is the session graph's own input shape: the tensor
// Detect feeds and the grid the heads decode against. A dynamic or
// non-raster input (anything but 1-or-dynamic batch, 3 channels,
// positive height and width) is refused -- the geometry truth cannot
// be guessed.
type inputGeometry struct {
	width  int
	height int
}

// gridCells is the anchor grid of one stride over the input geometry:
// the cells covering the input rounded up to the stride's parent
// scale (32), the family's divisibility convention. The second return
// is false when the stride does not divide the padded extent -- the
// heads of such a graph cannot be addressed honestly.
func (g inputGeometry) gridCells(stride int) (rows, cols int, ok bool) {
	padWidth := roundUpTo(g.width, strideDivisor)
	padHeight := roundUpTo(g.height, strideDivisor)
	if padWidth%stride != 0 || padHeight%stride != 0 {
		return 0, 0, false
	}
	return padHeight / stride, padWidth / stride, true
}

// strideDivisor is the scale every feature stride of the family
// divides: the input extent is interpreted in its multiples.
const strideDivisor = 32

// readInputGeometry reads the graph's input metadata.
func readInputGeometry(sess *ml.Session) (inputGeometry, error) {
	inputs := sess.Inputs()
	if len(inputs) != 1 {
		return inputGeometry{}, fmt.Errorf("mlface: the detector graph carries %d inputs, want 1", len(inputs))
	}
	dims := inputs[0].Dimensions
	if len(dims) != 4 {
		return inputGeometry{}, fmt.Errorf("mlface: the detector input has %d dimensions, want NCHW (4)", len(dims))
	}
	if dims[0] != 1 && dims[0] > 0 {
		return inputGeometry{}, fmt.Errorf("mlface: the detector input batch is %d, want 1", dims[0])
	}
	if dims[1] != 3 {
		return inputGeometry{}, fmt.Errorf("mlface: the detector input carries %d channels, want 3", dims[1])
	}
	if dims[2] <= 0 || dims[3] <= 0 {
		return inputGeometry{}, fmt.Errorf("mlface: the detector input geometry %dx%d is dynamic; the wrapper pins fixed-geometry graphs", dims[3], dims[2])
	}
	return inputGeometry{width: int(dims[3]), height: int(dims[2])}, nil
}

// validateRaster checks the ml.Image contract before any work.
func validateRaster(img ml.Image) error {
	if img.Width <= 0 || img.Height <= 0 {
		return fmt.Errorf("mlface: raster is %dx%d, want positive dimensions", img.Width, img.Height)
	}
	if len(img.Pixels) != img.Width*img.Height*3 {
		return fmt.Errorf("mlface: raster carries %d bytes, want %d for %dx%d RGB",
			len(img.Pixels), img.Width*img.Height*3, img.Width, img.Height)
	}
	return nil
}

// headSet is one feature stride's decoded head quartet.
type headSet struct {
	stride int
	// presence carries the class confidence per anchor.
	presence []float32
	// objectness carries the object confidence per anchor.
	objectness []float32
	// boxOffsets carries, per anchor: the cell-unit offsets of the
	// box center from the anchor's cell, then the log sizes of the
	// box's width and height in stride units.
	boxOffsets []float32
	// landmarkOffsets carries, per anchor, the five landmarks'
	// cell-unit offsets from the anchor's cell.
	landmarkOffsets []float32
}

// runGraph feeds the resized raster through the session and collects
// the per-stride head quartets, keyed by stride and validated against
// the anchor grids.
func runGraph(sess *ml.Session, img ml.Image, geometry inputGeometry) ([]headSet, error) {
	tensor, err := ort.NewTensor(ort.NewShape(1, 3, int64(geometry.height), int64(geometry.width)),
		bgrCHW(resizeRaster(img, geometry.width, geometry.height)))
	if err != nil {
		return nil, fmt.Errorf("mlface: build the input tensor: %w", err)
	}
	defer func() { _ = tensor.Destroy() }()

	outputs := make([]ort.Value, len(sess.Outputs()))
	if err := sess.Run([]ort.Value{tensor}, outputs); err != nil {
		return nil, fmt.Errorf("mlface: run the detector: %w", err)
	}
	for _, output := range outputs {
		if output != nil {
			defer func() { _ = output.Destroy() }()
		}
	}

	byStride := map[int]*headSet{}
	for i, info := range sess.Outputs() {
		kind, stride, ok := splitHeadName(info.Name)
		if !ok {
			return nil, fmt.Errorf("mlface: the detector output %q does not follow the head naming convention", info.Name)
		}
		head := byStride[stride]
		if head == nil {
			head = &headSet{stride: stride}
			byStride[stride] = head
		}
		data, err := tensorData(outputs[i], info.Name)
		if err != nil {
			return nil, err
		}
		if err := head.bind(kind, data, geometry); err != nil {
			return nil, err
		}
	}
	if len(byStride) == 0 {
		return nil, fmt.Errorf("mlface: the detector graph emits no anchor-free heads")
	}

	// Deterministic order: ascending stride. Every quartet must be
	// complete before the decode walks it.
	strides := make([]int, 0, len(byStride))
	for stride := range byStride {
		strides = append(strides, stride)
	}
	sort.Ints(strides)
	heads := make([]headSet, 0, len(strides))
	for _, stride := range strides {
		head := byStride[stride]
		if err := head.validate(); err != nil {
			return nil, err
		}
		heads = append(heads, *head)
	}
	return heads, nil
}

// bind fills one head channel set, checking its width against the
// convention.
func (h *headSet) bind(kind string, data []float32, geometry inputGeometry) error {
	rows, cols, ok := geometry.gridCells(h.stride)
	if !ok {
		return fmt.Errorf("mlface: stride %d does not divide the input extent", h.stride)
	}
	anchors := rows * cols
	switch kind {
	case "cls":
		if len(data) != anchors {
			return fmt.Errorf("mlface: cls_%d carries %d values, want %d", h.stride, len(data), anchors)
		}
		h.presence = data
	case "obj":
		if len(data) != anchors {
			return fmt.Errorf("mlface: obj_%d carries %d values, want %d", h.stride, len(data), anchors)
		}
		h.objectness = data
	case "bbox":
		if len(data) != anchors*4 {
			return fmt.Errorf("mlface: bbox_%d carries %d values, want %d", h.stride, len(data), anchors*4)
		}
		h.boxOffsets = data
	case "kps":
		if len(data) != anchors*10 {
			return fmt.Errorf("mlface: kps_%d carries %d values, want %d", h.stride, len(data), anchors*10)
		}
		h.landmarkOffsets = data
	default:
		return fmt.Errorf("mlface: unknown head kind %q", kind)
	}
	return nil
}

// validate refuses a stride whose quartet is incomplete.
func (h *headSet) validate() error {
	switch {
	case h.presence == nil:
		return fmt.Errorf("mlface: stride %d carries no cls head", h.stride)
	case h.objectness == nil:
		return fmt.Errorf("mlface: stride %d carries no obj head", h.stride)
	case h.boxOffsets == nil:
		return fmt.Errorf("mlface: stride %d carries no bbox head", h.stride)
	case h.landmarkOffsets == nil:
		return fmt.Errorf("mlface: stride %d carries no kps head", h.stride)
	}
	return nil
}

// headNamePattern matches the family's output names: a head kind
// (cls / obj / bbox / kps) underscore a feature stride.
var headNamePattern = regexp.MustCompile(`^(cls|obj|bbox|kps)_(\d+)$`)

// splitHeadName parses one output name into its kind and stride.
func splitHeadName(name string) (kind string, stride int, ok bool) {
	match := headNamePattern.FindStringSubmatch(name)
	if match == nil {
		return "", 0, false
	}
	stride, err := strconv.Atoi(match[2])
	if err != nil || stride <= 0 {
		return "", 0, false
	}
	return match[1], stride, true
}

// tensorData reads one output value as a float32 tensor.
func tensorData(value ort.Value, name string) ([]float32, error) {
	tensor, ok := value.(*ort.Tensor[float32])
	if !ok {
		return nil, fmt.Errorf("mlface: output %q is %T, want a float32 tensor", name, value)
	}
	return tensor.GetData(), nil
}

// faceBox is one decoded candidate, in the input geometry's pixel
// space, before the overlap suppression.
type faceBox struct {
	x1, y1   float64
	width    float64
	height   float64
	score    float64
	landmark [5][2]float64
}

// decodeFaces walks every anchor of every stride, keeps the candidates
// whose geometric-mean confidence meets the threshold, decodes their
// boxes and landmarks out of the cell-unit offsets, and returns them
// sorted by descending confidence.
func decodeFaces(heads []headSet, geometry inputGeometry, opts decodeOptions) []faceBox {
	var faces []faceBox
	for _, head := range heads {
		rows, cols, _ := geometry.gridCells(head.stride)
		for row := 0; row < rows; row++ {
			for col := 0; col < cols; col++ {
				anchor := row*cols + col
				score := math.Sqrt(
					float64(clampUnit(head.presence[anchor])) *
						float64(clampUnit(head.objectness[anchor])))
				if score < opts.scoreThreshold {
					continue
				}
				stride := float64(head.stride)
				centerX := (float64(col) + float64(head.boxOffsets[anchor*4])) * stride
				centerY := (float64(row) + float64(head.boxOffsets[anchor*4+1])) * stride
				width := math.Exp(float64(head.boxOffsets[anchor*4+2])) * stride
				height := math.Exp(float64(head.boxOffsets[anchor*4+3])) * stride
				var face faceBox
				face.x1 = centerX - width/2
				face.y1 = centerY - height/2
				face.width = width
				face.height = height
				face.score = score
				for n := 0; n < 5; n++ {
					face.landmark[n][0] = (float64(col) + float64(head.landmarkOffsets[anchor*10+2*n])) * stride
					face.landmark[n][1] = (float64(row) + float64(head.landmarkOffsets[anchor*10+2*n+1])) * stride
				}
				faces = append(faces, face)
			}
		}
	}
	sort.SliceStable(faces, func(i, j int) bool { return faces[i].score > faces[j].score })
	return faces
}

// suppressOverlaps drops every candidate whose box overlaps a kept
// higher-scored candidate by more than the overlap threshold: the
// anchor neighborhoods around one face collapse to their best member.
func suppressOverlaps(faces []faceBox, opts decodeOptions) []faceBox {
	kept := make([]faceBox, 0, len(faces))
	for _, face := range faces {
		duplicate := false
		for _, other := range kept {
			if overlapRatio(face, other) > opts.nmsThreshold {
				duplicate = true
				break
			}
		}
		if !duplicate {
			kept = append(kept, face)
		}
	}
	return kept
}

// overlapRatio is the intersection-over-union of two boxes.
func overlapRatio(a, b faceBox) float64 {
	left := math.Max(a.x1, b.x1)
	top := math.Max(a.y1, b.y1)
	right := math.Min(a.x1+a.width, b.x1+b.width)
	bottom := math.Min(a.y1+a.height, b.y1+b.height)
	overlap := math.Max(0, right-left) * math.Max(0, bottom-top)
	union := a.width*a.height + b.width*b.height - overlap
	if union <= 0 {
		return 0
	}
	return overlap / union
}

// toDetection maps one suppressed face from the input geometry's
// pixel space back into the raster's coordinates: the landmarks as
// raster pixels, the box as per-mille of the raster's dimensions
// (clamped into the table's 0-1000 CHECK bounds and inside the
// image).
func toDetection(face faceBox, img ml.Image, geometry inputGeometry) Detection {
	scaleX := float64(img.Width) / float64(geometry.width)
	scaleY := float64(img.Height) / float64(geometry.height)

	var detection Detection
	detection.Score = float32(face.score)
	x := face.x1 * scaleX
	y := face.y1 * scaleY
	w := face.width * scaleX
	h := face.height * scaleY
	detection.BBox[0] = perMille(x, float64(img.Width))
	detection.BBox[1] = perMille(y, float64(img.Height))
	detection.BBox[2] = clamp(perMille(w, float64(img.Width)), 0, 1000-detection.BBox[0])
	detection.BBox[3] = clamp(perMille(h, float64(img.Height)), 0, 1000-detection.BBox[1])
	for n := 0; n < 5; n++ {
		detection.Landmarks[n][0] = face.landmark[n][0] * scaleX
		detection.Landmarks[n][1] = face.landmark[n][1] * scaleY
	}
	return detection
}

// perMille converts one coordinate into thousandths of the whole,
// clamped into [0, 1000].
func perMille(value, whole float64) int {
	return clamp(int(math.Round(value/whole*1000)), 0, 1000)
}

// clamp bounds value into [lo, hi].
func clamp(value, lo, hi int) int {
	if value < lo {
		return lo
	}
	if value > hi {
		return hi
	}
	return value
}

// clampUnit bounds a confidence channel into [0, 1]: the heads are
// sigmoid-shaped, an outlier clamps instead of escaping the score.
func clampUnit(value float32) float32 {
	if value < 0 {
		return 0
	}
	if value > 1 {
		return 1
	}
	return value
}

// roundUpTo rounds extent up to a multiple of unit.
func roundUpTo(extent, unit int) int {
	return ((extent-1)/unit + 1) * unit
}

// bgrCHW flattens an RGB raster into the graph's channel-first BGR
// float tensor data at 0-255 amplitude -- the family's graphs were
// trained on BGR imagery.
func bgrCHW(img ml.Image) []float32 {
	plane := img.Width * img.Height
	data := make([]float32, 3*plane)
	for i := 0; i < plane; i++ {
		r := float32(img.Pixels[i*3])
		g := float32(img.Pixels[i*3+1])
		b := float32(img.Pixels[i*3+2])
		data[i] = b
		data[plane+i] = g
		data[2*plane+i] = r
	}
	return data
}

// resizeRaster resizes the raster to width x height with a bilinear
// filter, the direct (aspect-changing) resize the family's fully
// convolutional graphs tolerate. A same-size input returns the
// original raster bytes (the caller copies through the tensor anyway).
func resizeRaster(img ml.Image, width, height int) ml.Image {
	if img.Width == width && img.Height == height {
		return img
	}
	out := ml.Image{Width: width, Height: height, Pixels: make([]byte, width*height*3)}
	srcX := float64(img.Width) / float64(width)
	srcY := float64(img.Height) / float64(height)
	for y := 0; y < height; y++ {
		fy := (float64(y) + 0.5) * srcY
		y0, y1, wy := neighbors(fy-0.5, img.Height)
		for x := 0; x < width; x++ {
			fx := (float64(x) + 0.5) * srcX
			x0, x1, wx := neighbors(fx-0.5, img.Width)
			for c := 0; c < 3; c++ {
				top := mix(pixelAt(img, x0, y0, c), pixelAt(img, x1, y0, c), wx)
				bottom := mix(pixelAt(img, x0, y1, c), pixelAt(img, x1, y1, c), wx)
				out.Pixels[(y*width+x)*3+c] = byte(mix(top, bottom, wy) + 0.5)
			}
		}
	}
	return out
}

// neighbors resolves the two source pixels a half-open sample falls
// between, with its weight on the second.
func neighbors(position float64, extent int) (lo, hi int, weight float64) {
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

// pixelAt reads one raster channel.
func pixelAt(img ml.Image, x, y, c int) float64 {
	return float64(img.Pixels[(y*img.Width+x)*3+c])
}

// mix interpolates linearly.
func mix(a, b, t float64) float64 {
	return a*(1-t) + b*t
}
