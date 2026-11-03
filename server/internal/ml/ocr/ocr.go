// The OCR pipeline wrapper (M3a Task 8): one detection session and one
// recognition session from the SAME model directory (the default
// group's OCR model pins both graphs as its two files), plus the
// character dictionary of the recognition family, embedded from the
// committed copy. Extract runs the whole pipeline over one decoded
// raster: detection over the session's own input geometry, the
// textbook DB postprocess (probability-map binarization, connected
// components, per-component score filtering, the minimum-area bounding
// rectangle, the unclip expansion, the clip into the raster), the
// reading-order sort (row clustering by vertical-center proximity,
// left-to-right within each row), every box's crop through the
// recognition graph (height-48 proportional width, the family's input
// convention), the CTC greedy decode, and the reading-order join.
//
// The semantics are understood from the family's public documentation
// and implemented here from scratch, with the conventions measured
// against the default group's real graphs: both graphs consume BGR
// channel order (the family was trained on BGR imagery -- the same
// convention the face detection wrapper follows), the detector's input
// normalizes to ImageNet channel statistics at [0, 1] amplitude, the
// recognizer's to symmetric 0.5 mean and std, the recognizer's class
// count (6625) is exactly the dictionary's table (blank + 6623 lines +
// space), and the unclip expansion is load-bearing: the raw component
// box hugs the text's core and the recognizer misreads the clipped
// glyphs (measured: the same crop decodes garbage without the
// expansion, the rendered text verbatim with it).
package mlocr

import (
	_ "embed"
	"fmt"
	"math"
	"sort"
	"strings"

	ort "github.com/yalue/onnxruntime_go"

	"github.com/welpher/lumemor/server/internal/ml"
)

//go:embed dict/ppocr_keys_v1.txt
var dictionaryFile []byte

// Pipeline is one OCR model's two graphs and the recognition
// dictionary they decode against. Construct with NewPipeline.
type Pipeline struct {
	// Det is the text detection graph: any NCHW raster in, the
	// probability map of the same geometry out.
	Det *ml.Session
	// Rec is the text recognition graph: height-48 crops in, per
	// timestep class probabilities out.
	Rec *ml.Session
	// Dict is the recognition dictionary's lines, in class order
	// (class 1 is Dict[0]): the recognition graph's class table is
	// blank, the dictionary's lines, then the space.
	Dict []string
}

// NewPipeline builds the pipeline over the model's two graphs with the
// embedded dictionary (the recognition family's committed class
// vocabulary -- the graphs of the default group agree with it by
// construction, and the decode refuses a graph whose class count
// disagrees).
func NewPipeline(det, rec *ml.Session) *Pipeline {
	return &Pipeline{Det: det, Rec: rec, Dict: loadDictionary()}
}

// loadDictionary parses the embedded dictionary: one character per
// line, the trailing newline's empty split dropped.
func loadDictionary() []string {
	lines := strings.Split(string(dictionaryFile), "\n")
	dict := make([]string, 0, len(lines))
	for _, line := range lines {
		if line == "" {
			continue
		}
		dict = append(dict, line)
	}
	return dict
}

// classTableOf builds the recognition graph's class table: the blank
// (class 0, the empty string), the dictionary's lines (classes
// 1..len), and the space (the family's use-space-char convention, the
// last class).
func classTableOf(dict []string) []string {
	table := make([]string, 0, len(dict)+2)
	table = append(table, "")
	table = append(table, dict...)
	table = append(table, " ")
	return table
}

// The extract defaults: the family's published operating points, the
// values the default group's manifest params steer per model.
const (
	// DefaultDetThreshold binarizes the probability map.
	DefaultDetThreshold = 0.3
	// DefaultBoxThreshold filters boxes by their component's mean
	// probability.
	DefaultBoxThreshold = 0.5
	// DefaultUnclipRatio expands a box's area by this factor.
	DefaultUnclipRatio = 2.0
	// DefaultRecHeight is the recognition crop's height.
	DefaultRecHeight = 48
	// DefaultDetMaxSide bounds the detection input's long edge.
	DefaultDetMaxSide = 960
	// minBoxEdge drops components whose box is smaller than this many
	// probability-map pixels: the real detector's maps carry stray
	// high-probability specks, and a box below three pixels is noise,
	// never text.
	minBoxEdge = 3
	// detStrideDivisor is the scale the detection backbone's strides
	// divide: the wrapper's own chosen geometry rounds up to it.
	detStrideDivisor = 32
)

// extractOptions are the steering values of one extract run.
type extractOptions struct {
	detThreshold float64
	boxThreshold float64
	unclipRatio  float64
	recHeight    int
	detMaxSide   int
}

// defaultExtractOptions answers the family's operating points.
func defaultExtractOptions() extractOptions {
	return extractOptions{
		detThreshold: DefaultDetThreshold,
		boxThreshold: DefaultBoxThreshold,
		unclipRatio:  DefaultUnclipRatio,
		recHeight:    DefaultRecHeight,
		detMaxSide:   DefaultDetMaxSide,
	}
}

// extractOptionsOf reads the steering values out of the model's params
// object, falling back to the defaults for absent or malformed entries
// (the drifted-params discipline: a bad value degrades the threshold,
// it does not fail every asset's OCR).
func extractOptionsOf(params map[string]any) extractOptions {
	opts := defaultExtractOptions()
	if v, ok := paramFloat(params, "detThreshold"); ok {
		opts.detThreshold = v
	}
	if v, ok := paramFloat(params, "boxThreshold"); ok {
		opts.boxThreshold = v
	}
	if v, ok := paramFloat(params, "unclipRatio"); ok && v >= 1 {
		opts.unclipRatio = v
	}
	if v, ok := paramFloat(params, "recHeight"); ok && v >= 8 {
		opts.recHeight = int(v)
	}
	if v, ok := paramFloat(params, "detMaxSide"); ok && v >= 32 {
		opts.detMaxSide = int(v)
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

// Extract runs the pipeline with the family's default operating
// points (the params-less convenience over ExtractWithParams).
func (p *Pipeline) Extract(img ml.Image) (string, error) {
	return p.ExtractWithParams(img, nil)
}

// ExtractWithParams runs the pipeline over one raster with the model's
// params steering the decode. Boxes order into reading rows; each
// row's box texts join with a single space, the rows with a newline.
// An image with no boxes answers the empty string with no error --
// no text is a finding, not a failure.
func (p *Pipeline) ExtractWithParams(img ml.Image, params map[string]any) (string, error) {
	if err := validateRaster(img); err != nil {
		return "", err
	}
	opts := extractOptionsOf(params)

	boxes, err := p.detectBoxes(img, opts)
	if err != nil {
		return "", err
	}
	if len(boxes) == 0 {
		return "", nil
	}

	table := classTableOf(p.Dict)
	var lines []string
	for _, row := range readingOrderRows(boxes) {
		words := make([]string, 0, len(row))
		for _, box := range row {
			word, err := p.recognizeBox(img, box, opts, table)
			if err != nil {
				return "", err
			}
			words = append(words, word)
		}
		lines = append(lines, strings.Join(words, " "))
	}
	return strings.Join(lines, "\n"), nil
}

// detectBoxes runs the detection graph and its postprocess, answering
// the text boxes in the RASTER's coordinates, clipped into the raster.
func (p *Pipeline) detectBoxes(img ml.Image, opts extractOptions) ([]textBox, error) {
	prob, mapW, mapH, err := p.runDet(img, opts)
	if err != nil {
		return nil, err
	}
	mapBoxes := findTextBoxes(prob, mapW, mapH, opts)

	scaleX := float64(img.Width) / float64(mapW)
	scaleY := float64(img.Height) / float64(mapH)
	boxes := make([]textBox, 0, len(mapBoxes))
	for _, box := range mapBoxes {
		// Map the box into the raster's coordinates, then clip it into
		// the raster: an unclipped box's crop edge clamps to the
		// raster's edge, never past it.
		x := box.x * scaleX
		y := box.y * scaleY
		w := box.w * scaleX
		h := box.h * scaleY
		if x < 0 {
			w += x
			x = 0
		}
		if y < 0 {
			h += y
			y = 0
		}
		if x+w > float64(img.Width) {
			w = float64(img.Width) - x
		}
		if y+h > float64(img.Height) {
			h = float64(img.Height) - y
		}
		if w <= 0 || h <= 0 {
			continue
		}
		boxes = append(boxes, textBox{x: x, y: y, w: w, h: h})
	}
	return boxes, nil
}

// runDet feeds the raster through the detection graph and answers its
// probability map (with the map's geometry). The input geometry
// follows the session's own metadata: a graph that declares it gets
// the raster resized to exactly that, and a dynamic graph gets the
// wrapper's chosen geometry -- the long edge bounded by the max side,
// both edges rounded up to the backbone's stride divisor.
func (p *Pipeline) runDet(img ml.Image, opts extractOptions) (prob []float32, mapW, mapH int, err error) {
	width, height, err := detGeometry(p.Det, img, opts)
	if err != nil {
		return nil, 0, 0, err
	}

	tensor, err := ort.NewTensor(ort.NewShape(1, 3, int64(height), int64(width)),
		detCHW(ml.ResizeRaster(img, width, height)))
	if err != nil {
		return nil, 0, 0, fmt.Errorf("mlocr: build the detection input tensor: %w", err)
	}
	defer func() { _ = tensor.Destroy() }()

	outputs := make([]ort.Value, 1)
	if err := p.Det.Run([]ort.Value{tensor}, outputs); err != nil {
		return nil, 0, 0, fmt.Errorf("mlocr: run the detector: %w", err)
	}
	defer func() { _ = outputs[0].Destroy() }()

	name := p.Det.Outputs()[0].Name
	data, shape, err := tensorData(outputs[0], name)
	if err != nil {
		return nil, 0, 0, err
	}
	// The map's geometry: [1, 1, H, W] (or the rank-2 [H, W]), the
	// input's own extents.
	mapH, mapW = height, width
	switch len(shape) {
	case 4:
		if shape[0] != 1 || shape[1] != 1 || int(shape[2]) != mapH || int(shape[3]) != mapW {
			return nil, 0, 0, fmt.Errorf("mlocr: the detector %s emitted %v, want the input's [1, 1, %d, %d]", name, shape, mapH, mapW)
		}
	case 3:
		if shape[0] != 1 || int(shape[1]) != mapH || int(shape[2]) != mapW {
			return nil, 0, 0, fmt.Errorf("mlocr: the detector %s emitted %v, want the input's [1, %d, %d]", name, shape, mapH, mapW)
		}
	case 2:
		if int(shape[0]) != mapH || int(shape[1]) != mapW {
			return nil, 0, 0, fmt.Errorf("mlocr: the detector %s emitted %v, want the input's [%d, %d]", name, shape, mapH, mapW)
		}
	default:
		return nil, 0, 0, fmt.Errorf("mlocr: the detector %s emitted %d dimensions, want the probability map", name, len(shape))
	}
	if len(data) != mapW*mapH {
		return nil, 0, 0, fmt.Errorf("mlocr: the detector %s emitted %d values, want %d", name, len(data), mapW*mapH)
	}
	return data, mapW, mapH, nil
}

// detGeometry resolves the detection tensor geometry for one raster:
// the session's declared input when the graph pins it, the wrapper's
// own choice when the graph leaves it dynamic.
func detGeometry(sess *ml.Session, img ml.Image, opts extractOptions) (width, height int, err error) {
	inputs := sess.Inputs()
	if len(inputs) != 1 {
		return 0, 0, fmt.Errorf("mlocr: the detector graph carries %d inputs, want 1", len(inputs))
	}
	dims := inputs[0].Dimensions
	if len(dims) != 4 {
		return 0, 0, fmt.Errorf("mlocr: the detector input has %d dimensions, want NCHW (4)", len(dims))
	}
	if dims[0] != 1 && dims[0] > 0 {
		return 0, 0, fmt.Errorf("mlocr: the detector input batch is %d, want 1", dims[0])
	}
	if dims[1] != 3 {
		return 0, 0, fmt.Errorf("mlocr: the detector input carries %d channels, want 3", dims[1])
	}
	if dims[2] > 0 && dims[3] > 0 {
		return int(dims[3]), int(dims[2]), nil
	}

	// The wrapper's own geometry: the long edge bounded (never
	// upscaled), both edges rounded up to the stride divisor so the
	// backbone's feature strides divide them.
	long := max(img.Width, img.Height)
	scale := 1.0
	if long > opts.detMaxSide {
		scale = float64(opts.detMaxSide) / float64(long)
	}
	width = int(math.Round(float64(img.Width) * scale))
	height = int(math.Round(float64(img.Height) * scale))
	return roundUpTo(width, detStrideDivisor), roundUpTo(height, detStrideDivisor), nil
}

// roundUpTo rounds extent up to a multiple of unit.
func roundUpTo(extent, unit int) int {
	if extent < 1 {
		return unit
	}
	return ((extent-1)/unit + 1) * unit
}

// textBox is one text box: the top-left corner and the extents, in
// the coordinate space its producer states.
type textBox struct {
	x, y, w, h float64
}

// findTextBoxes is the detection postprocess over one probability
// map: binarize at the det threshold, label the 8-connected
// components, drop the components whose mean probability misses the
// box threshold, fit each survivor's minimum-area bounding rectangle,
// drop the boxes under the minimum edge, and expand the survivors by
// the unclip offset (the area ratio solved as a uniform edge
// expansion). The answers are map-coordinate boxes.
func findTextBoxes(prob []float32, mapW, mapH int, opts extractOptions) []textBox {
	visited := make([]bool, mapW*mapH)
	var boxes []textBox
	for y := 0; y < mapH; y++ {
		for x := 0; x < mapW; x++ {
			i := y*mapW + x
			if visited[i] || float64(prob[i]) < opts.detThreshold {
				continue
			}
			points := componentAt(prob, visited, mapW, mapH, x, y, opts.detThreshold)
			if len(points) == 0 {
				continue
			}
			sum := 0.0
			for _, p := range points {
				sum += float64(prob[p[1]*mapW+p[0]])
			}
			if sum/float64(len(points)) < opts.boxThreshold {
				continue
			}
			corners := minAreaRect(points)
			box := aabbOf(corners)
			if box.w < minBoxEdge || box.h < minBoxEdge {
				continue
			}
			offset := unclipOffset(box.w, box.h, opts.unclipRatio)
			boxes = append(boxes, textBox{
				x: box.x - offset,
				y: box.y - offset,
				w: box.w + 2*offset,
				h: box.h + 2*offset,
			})
		}
	}
	return boxes
}

// componentAt labels the 8-connected component containing (x, y) (the
// binarization threshold applied inline), marking every member visited
// and answering the component's points. A point below the threshold
// answers no component (the caller checked it).
func componentAt(prob []float32, visited []bool, mapW, mapH, x, y int, threshold float64) [][2]int {
	stack := []int{y*mapW + x}
	visited[y*mapW+x] = true
	points := make([][2]int, 0, 64)
	for len(stack) > 0 {
		i := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		cy, cx := i/mapW, i%mapW
		points = append(points, [2]int{cx, cy})
		for dy := -1; dy <= 1; dy++ {
			for dx := -1; dx <= 1; dx++ {
				ny, nx := cy+dy, cx+dx
				if ny < 0 || ny >= mapH || nx < 0 || nx >= mapW {
					continue
				}
				j := ny*mapW + nx
				if visited[j] || float64(prob[j]) < threshold {
					continue
				}
				visited[j] = true
				stack = append(stack, j)
			}
		}
	}
	return points
}

// minAreaRect answers the four corners of the minimum-area bounding
// rectangle of the points: the convex hull first, then the
// axis-aligned rectangle of the hull rotated to each hull edge, the
// smallest area winning (the rotating-calipers idea over every edge
// direction a hull edge offers -- the exact minimum for a rectangle
// with one side parallel to a hull edge, which the classic result
// guarantees). The corners walk the rectangle's perimeter; for
// axis-aligned clouds the rectangle is the plain bounding box.
func minAreaRect(points [][2]int) [4][2]float64 {
	hull := convexHull(points)
	if len(hull) < 3 {
		// A degenerate hull (one or two points): the plain bounding
		// box of the points.
		box := aabbOfInt(points)
		return [4][2]float64{
			{box.x, box.y},
			{box.x + box.w, box.y},
			{box.x + box.w, box.y + box.h},
			{box.x, box.y + box.h},
		}
	}

	best := [4][2]float64{}
	bestArea := math.Inf(1)
	for i := 0; i < len(hull); i++ {
		// The edge direction of hull side i, as a unit vector.
		ax := float64(hull[(i+1)%len(hull)][0] - hull[i][0])
		ay := float64(hull[(i+1)%len(hull)][1] - hull[i][1])
		length := math.Hypot(ax, ay)
		if length == 0 {
			continue
		}
		ax, ay = ax/length, ay/length
		bx, by := -ay, ax

		// The hull's extents along the edge's two axes.
		minA, maxA, minB, maxB := math.Inf(1), math.Inf(-1), math.Inf(1), math.Inf(-1)
		for _, p := range hull {
			pa := float64(p[0])*ax + float64(p[1])*ay
			pb := float64(p[0])*bx + float64(p[1])*by
			minA, maxA = math.Min(minA, pa), math.Max(maxA, pa)
			minB, maxB = math.Min(minB, pb), math.Max(maxB, pb)
		}
		area := (maxA - minA) * (maxB - minB)
		if area >= bestArea {
			continue
		}
		bestArea = area
		// The corner points of the rectangle, mapped back into image
		// coordinates: the two axis-aligned corners of each extent
		// pair.
		corner := func(a, b float64) [2]float64 {
			return [2]float64{ax*a + bx*b, ay*a + by*b}
		}
		best = [4][2]float64{
			corner(minA, minB),
			corner(maxA, minB),
			corner(maxA, maxB),
			corner(minA, maxB),
		}
	}
	return best
}

// convexHull answers the points' convex hull (the monotone-chain
// algorithm), in counter-clockwise order, without duplicate endpoints.
func convexHull(points [][2]int) [][2]int {
	sorted := make([][2]int, len(points))
	copy(sorted, points)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i][0] != sorted[j][0] {
			return sorted[i][0] < sorted[j][0]
		}
		return sorted[i][1] < sorted[j][1]
	})

	cross := func(o, a, b [2]int) int {
		return (a[0]-o[0])*(b[1]-o[1]) - (a[1]-o[1])*(b[0]-o[0])
	}
	lower := make([][2]int, 0, len(sorted))
	for _, p := range sorted {
		for len(lower) >= 2 && cross(lower[len(lower)-2], lower[len(lower)-1], p) <= 0 {
			lower = lower[:len(lower)-1]
		}
		lower = append(lower, p)
	}
	upper := make([][2]int, 0, len(sorted))
	for i := len(sorted) - 1; i >= 0; i-- {
		p := sorted[i]
		for len(upper) >= 2 && cross(upper[len(upper)-2], upper[len(upper)-1], p) <= 0 {
			upper = upper[:len(upper)-1]
		}
		upper = append(upper, p)
	}
	hull := append(lower[:len(lower)-1], upper[:len(upper)-1]...)
	return hull
}

// aabbOf answers the axis-aligned bounding box of a corner set, in the
// pixel-count convention every box consumer uses: the box [x, x+w)
// covers the pixels the corners' extents span (the +1 closes the span
// the min/max coordinates bound), matching aabbOfInt so the rotated
// and degenerate hull paths agree.
func aabbOf(corners [4][2]float64) textBox {
	minX, minY := math.Inf(1), math.Inf(1)
	maxX, maxY := math.Inf(-1), math.Inf(-1)
	for _, c := range corners {
		minX, minY = math.Min(minX, c[0]), math.Min(minY, c[1])
		maxX, maxY = math.Max(maxX, c[0]), math.Max(maxY, c[1])
	}
	return textBox{x: minX, y: minY, w: maxX - minX + 1, h: maxY - minY + 1}
}

// aabbOfInt answers the axis-aligned bounding box of integer points,
// in the same pixel-count convention.
func aabbOfInt(points [][2]int) textBox {
	minX, minY := points[0][0], points[0][1]
	maxX, maxY := points[0][0], points[0][1]
	for _, p := range points {
		minX, minY = min(minX, p[0]), min(minY, p[1])
		maxX, maxY = max(maxX, p[0]), max(maxY, p[1])
	}
	return textBox{
		x: float64(minX),
		y: float64(minY),
		w: float64(maxX - minX + 1),
		h: float64(maxY - minY + 1),
	}
}

// unclipOffset solves the uniform edge expansion that grows a w x h
// box's area by the ratio: 4d^2 + 2(w+h)d - (ratio-1)w*h = 0, the
// positive root. The ratio 1 answers zero (no expansion).
func unclipOffset(w, h, ratio float64) float64 {
	if ratio <= 1 || w <= 0 || h <= 0 {
		return 0
	}
	a, b, c := 4.0, 2.0*(w+h), (1.0-ratio)*w*h
	return (-b + math.Sqrt(b*b-4*a*c)) / (2 * a)
}

// readingOrderRows sorts the boxes into reading order: rows cluster by
// vertical-center proximity -- a box joins the row when its center sits
// within a quarter of the summed heights of itself and the row's mean
// member (text lines share a baseline, so their centers are close
// relative to their heights; the rule tolerates a line's ascenders and
// parentheses) -- rows order top-to-bottom, and each row orders
// left-to-right.
func readingOrderRows(boxes []textBox) [][]textBox {
	sorted := make([]textBox, len(boxes))
	copy(sorted, boxes)
	sort.SliceStable(sorted, func(i, j int) bool {
		ci := sorted[i].y + sorted[i].h/2
		cj := sorted[j].y + sorted[j].h/2
		if ci != cj {
			return ci < cj
		}
		return sorted[i].x < sorted[j].x
	})

	var rows [][]textBox
	var rowSumCy, rowSumH float64
	for _, box := range sorted {
		cy := box.y + box.h/2
		if len(rows) > 0 {
			count := float64(len(rows[len(rows)-1]))
			meanCy := rowSumCy / count
			meanH := rowSumH / count
			if math.Abs(cy-meanCy) <= (box.h+meanH)/4 {
				rows[len(rows)-1] = append(rows[len(rows)-1], box)
				rowSumCy += cy
				rowSumH += box.h
				continue
			}
		}
		rows = append(rows, []textBox{box})
		rowSumCy, rowSumH = cy, box.h
	}
	for _, row := range rows {
		sort.SliceStable(row, func(i, j int) bool { return row[i].x < row[j].x })
	}
	return rows
}

// recognizeBox crops the raster to the box, feeds the crop through the
// recognition graph, and answers the CTC greedy decode of its output.
func (p *Pipeline) recognizeBox(img ml.Image, box textBox, opts extractOptions, table []string) (string, error) {
	crop := cropRaster(img, box)
	width, height, err := recGeometry(p.Rec, crop, opts)
	if err != nil {
		return "", err
	}

	tensor, err := ort.NewTensor(ort.NewShape(1, 3, int64(height), int64(width)),
		recCHW(ml.ResizeRaster(crop, width, height)))
	if err != nil {
		return "", fmt.Errorf("mlocr: build the recognition input tensor: %w", err)
	}
	defer func() { _ = tensor.Destroy() }()

	outputs := make([]ort.Value, 1)
	if err := p.Rec.Run([]ort.Value{tensor}, outputs); err != nil {
		return "", fmt.Errorf("mlocr: run the recognizer: %w", err)
	}
	defer func() { _ = outputs[0].Destroy() }()

	name := p.Rec.Outputs()[0].Name
	data, shape, err := tensorData(outputs[0], name)
	if err != nil {
		return "", err
	}
	var steps, classes int
	switch len(shape) {
	case 3:
		if shape[0] != 1 {
			return "", fmt.Errorf("mlocr: the recognizer %s emitted batch %d, want 1", name, shape[0])
		}
		steps, classes = int(shape[1]), int(shape[2])
	case 2:
		steps, classes = int(shape[0]), int(shape[1])
	default:
		return "", fmt.Errorf("mlocr: the recognizer %s emitted %d dimensions, want the per-timestep classes", name, len(shape))
	}
	if classes != len(table) {
		return "", fmt.Errorf("mlocr: the recognizer %s carries %d classes, want the dictionary's %d (the graph and the dictionary must agree)", name, classes, len(table))
	}
	if steps*classes != len(data) {
		return "", fmt.Errorf("mlocr: the recognizer %s emitted %d values, want %d", name, len(data), steps*classes)
	}

	// The greedy decode's class sequence: the argmax of every
	// timestep's class distribution.
	sequence := make([]int, steps)
	for t := 0; t < steps; t++ {
		bestClass := 0
		bestScore := float32(math.Inf(-1))
		for c := 0; c < classes; c++ {
			if score := data[t*classes+c]; score > bestScore {
				bestScore, bestClass = score, c
			}
		}
		sequence[t] = bestClass
	}
	return ctcGreedyDecode(sequence, table), nil
}

// recGeometry resolves the recognition tensor geometry for one crop:
// a graph that declares both extents gets the crop resized to them,
// everything else answers the family's convention -- the height the
// params steer (default 48), the width proportional to the crop's
// aspect.
func recGeometry(sess *ml.Session, crop ml.Image, opts extractOptions) (width, height int, err error) {
	inputs := sess.Inputs()
	if len(inputs) != 1 {
		return 0, 0, fmt.Errorf("mlocr: the recognizer graph carries %d inputs, want 1", len(inputs))
	}
	dims := inputs[0].Dimensions
	if len(dims) != 4 {
		return 0, 0, fmt.Errorf("mlocr: the recognizer input has %d dimensions, want NCHW (4)", len(dims))
	}
	if dims[0] != 1 && dims[0] > 0 {
		return 0, 0, fmt.Errorf("mlocr: the recognizer input batch is %d, want 1", dims[0])
	}
	if dims[1] != 3 {
		return 0, 0, fmt.Errorf("mlocr: the recognizer input carries %d channels, want 3", dims[1])
	}
	if dims[2] > 0 && dims[3] > 0 {
		return int(dims[3]), int(dims[2]), nil
	}
	height = opts.recHeight
	width = int(math.Round(float64(crop.Width) * float64(height) / float64(crop.Height)))
	if width < 1 {
		width = 1
	}
	return width, height, nil
}

// ctcGreedyDecode decodes one greedy class sequence against the class
// table: adjacent equal classes collapse (a repeated character is one
// emission), the blank (class 0) breaks the collapse and vanishes.
func ctcGreedyDecode(sequence []int, table []string) string {
	var out strings.Builder
	previous := -1
	for _, class := range sequence {
		if class != previous && class != 0 {
			out.WriteString(table[class])
		}
		previous = class
	}
	return out.String()
}

// cropRaster extracts the box's region of the raster as its own
// raster: the pixel-aligned hull of the box (floor at the top-left,
// ceil at the bottom-right), clamped into the raster, never narrower
// than one pixel.
func cropRaster(img ml.Image, box textBox) ml.Image {
	x0 := int(math.Floor(box.x))
	y0 := int(math.Floor(box.y))
	x1 := int(math.Ceil(box.x + box.w))
	y1 := int(math.Ceil(box.y + box.h))
	x0, y0 = max(x0, 0), max(y0, 0)
	x1, y1 = min(x1, img.Width), min(y1, img.Height)
	x1, y1 = max(x1, x0+1), max(y1, y0+1)

	crop := ml.Image{Width: x1 - x0, Height: y1 - y0, Pixels: make([]byte, (x1-x0)*(y1-y0)*3)}
	for y := y0; y < y1; y++ {
		copy(crop.Pixels[(y-y0)*crop.Width*3:(y-y0+1)*crop.Width*3],
			img.Pixels[y*img.Width*3+x0*3:y*img.Width*3+x1*3])
	}
	return crop
}

// detCHW flattens an RGB raster into the detector's channel-first
// float tensor data: BGR plane order (plane 0 carries the blue
// channel -- the reversed order of the RGB raster's channels) at
// [0, 1] amplitude, with the ImageNet channel statistics applied
// positionally to the planes -- the tensor the family's graphs saw in
// training (the training pipeline normalized BGR imagery with the
// statistics in this plane order; the measured maps of the default
// group's graph confirm the convention is load-bearing for nothing
// coarser than a one-pixel map shift).
func detCHW(img ml.Image) []float32 {
	mean := [3]float64{0.485, 0.456, 0.406}
	std := [3]float64{0.229, 0.224, 0.225}
	plane := img.Width * img.Height
	data := make([]float32, 3*plane)
	for i := 0; i < plane; i++ {
		for p := 0; p < 3; p++ {
			// Plane p reads the raster's channel 2-p: the B, G, R
			// plane order.
			value := float64(img.Pixels[i*3+2-p]) / 255.0
			data[p*plane+i] = float32((value - mean[p]) / std[p])
		}
	}
	return data
}

// recCHW flattens an RGB raster into the recognizer's channel-first
// float tensor data: BGR plane order at [0, 1] amplitude with the
// symmetric 0.5 mean and std.
func recCHW(img ml.Image) []float32 {
	plane := img.Width * img.Height
	data := make([]float32, 3*plane)
	for i := 0; i < plane; i++ {
		for p := 0; p < 3; p++ {
			value := float64(img.Pixels[i*3+2-p]) / 255.0
			data[p*plane+i] = float32((value - 0.5) / 0.5)
		}
	}
	return data
}

// validateRaster checks the ml.Image contract before any work.
func validateRaster(img ml.Image) error {
	if img.Width <= 0 || img.Height <= 0 {
		return fmt.Errorf("mlocr: raster is %dx%d, want positive dimensions", img.Width, img.Height)
	}
	if len(img.Pixels) != img.Width*img.Height*3 {
		return fmt.Errorf("mlocr: raster carries %d bytes, want %d for %dx%d RGB",
			len(img.Pixels), img.Width*img.Height*3, img.Width, img.Height)
	}
	return nil
}

// tensorData reads one output value as a float32 tensor with its
// runtime shape.
func tensorData(value ort.Value, name string) ([]float32, []int64, error) {
	tensor, ok := value.(*ort.Tensor[float32])
	if !ok {
		return nil, nil, fmt.Errorf("mlocr: output %q is %T, want a float32 tensor", name, value)
	}
	return tensor.GetData(), tensor.GetShape(), nil
}
