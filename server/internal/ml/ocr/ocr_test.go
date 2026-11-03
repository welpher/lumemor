// The OCR pipeline wrapper tests (M3a Task 8): the CTC greedy decode's
// rules (blank dropping, repeat collapsing, the space class), the
// embedded dictionary's integrity (the class-count contract the
// recognition graphs must agree with), the detection postprocess over
// the committed det fixture's constant probability map (binarization,
// connected components, the score filter, the min-area rects, the
// unclip expansion), the reading-order row clustering, and the full
// Extract wiring over the det+rec fixture pair. The real-model smoke
// is env-gated.
//
// Hosts without the onnxruntime shared library skip the inference
// tests; the containerized `make docker-test` gate runs everything for
// real.
package mlocr

import (
	"image/png"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/welpher/lumemor/server/internal/ml"
)

// The measured graph facts of the committed fixtures and the default
// group's real OCR pair, pinned by digest.
const (
	ocrDetFixtureSHA256 = "4323256515d055a117c3605a2e8efb4d71274fb08b476658c45daed885348955"
	ocrRecFixtureSHA256 = "3a7aaaf4f653584c2d68d1e8a82f844f5f73d1cebf835d24b9275ee0fa1340c4"
	ocrDetWeightsSHA256 = "d2a7720d45a54257208b1e13e36a8479894cb74155a5efe29462512d42f49da9"
	ocrRecWeightsSHA256 = "48fc40f24f6d2a207a2b1091d3437eb3cc3eb6b676dc3ef9c37384005483683b"
)

// loadSession opens a session over one committed fixture, with the
// library skip guard every real-inference test carries.
func loadSession(t *testing.T, name, sha string) *ml.Session {
	t.Helper()
	if !ml.RuntimeAvailable() {
		t.Skip("onnxruntime shared library not present; run the suite inside the dev container (make docker-test)")
	}
	sess, err := ml.NewSession(filepath.Join("..", "testdata", name), sha)
	if err != nil {
		t.Fatalf("NewSession(%s) error = %v, want nil", name, err)
	}
	t.Cleanup(func() { _ = sess.Close() })
	return sess
}

// fixturePipeline builds the pipeline over the committed det+rec
// fixture pair.
func fixturePipeline(t *testing.T) *Pipeline {
	t.Helper()
	return NewPipeline(
		loadSession(t, "ocr_det.onnx", ocrDetFixtureSHA256),
		loadSession(t, "ocr_rec.onnx", ocrRecFixtureSHA256),
	)
}

// TestCtcGreedyDecode pins the decode's rules over a synthetic class
// table and the committed dictionary's measured positions: adjacent
// equal classes collapse, blanks break the collapse, blank runs vanish,
// and the table's last class is the space.
func TestCtcGreedyDecode(t *testing.T) {
	table := []string{"", "a", "b", " "}
	for _, test := range []struct {
		name string
		seq  []int
		want string
	}{
		{name: "repeats collapse", seq: []int{1, 1, 2, 2, 2}, want: "ab"},
		{name: "blanks break the collapse", seq: []int{1, 0, 1, 0, 1}, want: "aaa"},
		{name: "blank runs vanish", seq: []int{0, 0, 1, 0, 0, 0, 2, 0, 0}, want: "ab"},
		{name: "all blank", seq: []int{0, 0, 0}, want: ""},
		{name: "alternating", seq: []int{1, 2, 1, 2}, want: "abab"},
		{name: "space collapses too", seq: []int{3, 3, 1}, want: " a"},
	} {
		if got := ctcGreedyDecode(test.seq, table); got != test.want {
			t.Errorf("%s: ctcGreedyDecode(%v) = %q, want %q", test.name, test.seq, got, test.want)
		}
	}

	// The fixture's exact class sequence through the REAL dictionary's
	// class table: the measured positions of O, C and R decode to the
	// word the rec fixture stands for.
	real := classTableOf(loadDictionary())
	if got := ctcGreedyDecode([]int{0, 4741, 4741, 0, 4244, 0, 5127, 5127, 0, 0}, real); got != "OCR" {
		t.Errorf("the fixture sequence over the real dictionary = %q, want %q", got, "OCR")
	}
}

// TestEmbeddedDictionary pins the committed dictionary's integrity:
// 6623 single-character lines, no space line (the space is the class
// table's appended last entry, never a dictionary line), and a class
// table of exactly 6625 classes -- the count the recognition graph's
// output width must agree with.
func TestEmbeddedDictionary(t *testing.T) {
	dict := loadDictionary()
	if len(dict) != 6623 {
		t.Fatalf("dictionary lines = %d, want 6623", len(dict))
	}
	if dict[0] != "'" {
		t.Errorf("dictionary line 0 = %q, want the upstream first line %q", dict[0], "'")
	}
	for i, entry := range dict {
		if entry == "" {
			t.Fatalf("dictionary line %d is empty, want one character per line", i)
		}
		if entry == " " {
			t.Fatalf("dictionary line %d is a space, want none (the space is the class table's last class)", i)
		}
	}
	// The measured positions the rec fixture's sequence encodes.
	for entry, want := range map[string]int{"O": 4740, "C": 4243, "R": 5126} {
		if dict[want] != entry {
			t.Errorf("dictionary line %d = %q, want the measured %q", want, dict[want], entry)
		}
	}
	table := classTableOf(dict)
	if len(table) != 6625 {
		t.Fatalf("class table = %d classes, want 6625 (blank + 6623 + space)", len(table))
	}
	if table[0] != "" || table[6624] != " " {
		t.Errorf("class table ends = [%q, ..., %q], want blank first and space last", table[0], table[6624])
	}
}

// TestReadingOrderRows pins the reading-order sort: boxes cluster into
// rows by vertical-center proximity (a quarter of the summed heights),
// rows order top-to-bottom, and each row orders left-to-right.
func TestReadingOrderRows(t *testing.T) {
	// Five boxes of a two-word first row, a second row of two words,
	// and a third single box -- handed over OUT of reading order.
	rows := readingOrderRows([]textBox{
		{x: 30, y: 200, w: 50, h: 22},  // row 2, left word
		{x: 60, y: 102, w: 40, h: 18},  // row 1, right word
		{x: 0, y: 300, w: 30, h: 20},   // row 3
		{x: 10, y: 100, w: 40, h: 20},  // row 1, left word
		{x: 100, y: 205, w: 30, h: 12}, // row 2, right word
	})
	if len(rows) != 3 {
		t.Fatalf("rows = %d, want 3", len(rows))
	}
	wantX := [][]float64{{10, 60}, {30, 100}, {0}}
	for r, row := range rows {
		if len(row) != len(wantX[r]) {
			t.Fatalf("row %d carries %d boxes, want %d", r, len(row), len(wantX[r]))
		}
		for i, box := range row {
			if box.x != wantX[r][i] {
				t.Errorf("row %d position %d = box at x %v, want x %v", r, i, box.x, wantX[r][i])
			}
		}
	}

	// A tall box joins a row of shorter words when their centers stay
	// close relative to the summed heights: a caption line's ascenders
	// and parentheses do not split the line.
	rows = readingOrderRows([]textBox{
		{x: 10, y: 100, w: 40, h: 20}, // center y 110
		{x: 200, y: 96, w: 20, h: 40}, // center y 116, much taller
	})
	if len(rows) != 1 || len(rows[0]) != 2 {
		t.Fatalf("tall box rows = %d (first row %d boxes), want one row of 2", len(rows), len(rows[0]))
	}
	if rows[0][0].x != 10 || rows[0][1].x != 200 {
		t.Errorf("tall box row order = [%v, %v], want left-to-right", rows[0][0].x, rows[0][1].x)
	}

	// Boxes whose centers sit half a height apart belong to different
	// rows: a two-line layout never merges.
	rows = readingOrderRows([]textBox{
		{x: 0, y: 0, w: 30, h: 20},
		{x: 0, y: 20, w: 30, h: 20},
	})
	if len(rows) != 2 {
		t.Fatalf("stacked rows = %d, want 2 (centers 20 apart with heights 20)", len(rows))
	}
}

// TestFindTextBoxes pins the detection postprocess over the committed
// det fixture's constant probability map: the map's three 0.9 regions
// survive as min-area rects expanded by the unclip ratio (the area
// doubling solved as a uniform edge offset), the 0.4 region's component
// exists but the score filter drops it, and the tiny stray regions
// below the minimum box edge never surface. The expected offsets are
// derived by hand from the unclip equation, independently of the
// implementation.
func TestFindTextBoxes(t *testing.T) {
	p := fixturePipeline(t)
	prob, mapW, mapH, err := p.runDet(gradientImage(128, 96), defaultExtractOptions())
	if err != nil {
		t.Fatalf("runDet error = %v, want nil", err)
	}
	if mapW != 64 || mapH != 48 {
		t.Fatalf("det geometry = %dx%d, want the fixture's declared 64x48", mapW, mapH)
	}
	if len(prob) != mapW*mapH {
		t.Fatalf("probability map carries %d values, want %d", len(prob), mapW*mapH)
	}

	boxes := findTextBoxes(prob, mapW, mapH, defaultExtractOptions())
	if len(boxes) != 3 {
		t.Fatalf("text boxes = %d, want 3 (the 0.4 region's component is the score filter's victim)", len(boxes))
	}
	// The unclip equation for a w x h rect doubling its area:
	// 4d^2 + 2(w+h)d - w*h = 0, solved by hand for each region.
	const (
		dA = 2.2462111 // region (4, 8, 16, 8)
		dB = 2.4342865 // region (30, 8, 20, 8)
		dC = 2.7578088 // region (10, 40, 30, 8)
	)
	want := []textBox{
		{x: 4 - dA, y: 8 - dA, w: 16 + 2*dA, h: 8 + 2*dA},
		{x: 30 - dB, y: 8 - dB, w: 20 + 2*dB, h: 8 + 2*dB},
		{x: 10 - dC, y: 40 - dC, w: 30 + 2*dC, h: 8 + 2*dC},
	}
	for i, box := range boxes {
		for f, field := range []float64{box.x, box.y, box.w, box.h} {
			var expected float64
			switch f {
			case 0:
				expected = want[i].x
			case 1:
				expected = want[i].y
			case 2:
				expected = want[i].w
			case 3:
				expected = want[i].h
			}
			if diff := math.Abs(field - expected); diff > 0.001 {
				t.Errorf("box %d field %d = %v, want %v (diff %v)", i, f, field, expected, diff)
			}
		}
	}
}

// TestFindTextBoxesDropsTinyComponents pins the minimum-edge guard: a
// component whose box is smaller than the minimum edge never surfaces,
// so the probability map's stray single-pixel specks (the real
// detector's maps carry them) do not become boxes.
func TestFindTextBoxesDropsTinyComponents(t *testing.T) {
	prob := make([]float32, 64*48)
	// Two 1x1 specks and one real 3x3 word.
	for _, p := range [][2]int{{5, 5}, {40, 30}} {
		prob[p[1]*64+p[0]] = 0.9
	}
	for y := 20; y < 23; y++ {
		for x := 10; x < 13; x++ {
			prob[y*64+x] = 0.9
		}
	}
	boxes := findTextBoxes(prob, 64, 48, defaultExtractOptions())
	if len(boxes) != 1 {
		t.Fatalf("text boxes = %d, want 1 (the specks stay under the minimum edge)", len(boxes))
	}
	if boxes[0].x >= 10 || boxes[0].x+boxes[0].w <= 13 {
		t.Errorf("the surviving box = %v, want the 3x3 region at (10, 20)", boxes[0])
	}
}

// TestExtractFixturePipeline pins the full wiring over the committed
// det+rec fixture pair: the det fixture's three words over two rows,
// the boxes mapped back into the raster's coordinates, every crop
// through the rec fixture's fixed CTC sequence (the word "OCR"), and
// the reading-order join -- two words of the first row separated by a
// space, the second row after a newline.
func TestExtractFixturePipeline(t *testing.T) {
	p := fixturePipeline(t)

	text, err := p.Extract(gradientImage(128, 96))
	if err != nil {
		t.Fatalf("Extract error = %v, want nil", err)
	}
	if text != "OCR OCR\nOCR" {
		t.Errorf("Extract = %q, want %q", text, "OCR OCR\nOCR")
	}

	// A foreign-size raster runs the same declared det geometry and
	// answers the same text: the boxes scale, the decode does not.
	text, err = p.Extract(gradientImage(320, 240))
	if err != nil {
		t.Fatalf("Extract(foreign size) error = %v, want nil", err)
	}
	if text != "OCR OCR\nOCR" {
		t.Errorf("Extract(foreign size) = %q, want %q", text, "OCR OCR\nOCR")
	}
}

// TestExtractNoBoxesIsEmpty pins the zero-box boundary: a binarize
// threshold above every map value yields no components, and Extract
// answers the empty string with no error -- an image without text is a
// success, not a failure. The strict threshold also pins the params
// plumbing (detThreshold reaching the postprocess).
func TestExtractNoBoxesIsEmpty(t *testing.T) {
	p := fixturePipeline(t)

	text, err := p.ExtractWithParams(gradientImage(128, 96), map[string]any{"detThreshold": 0.99})
	if err != nil {
		t.Fatalf("ExtractWithParams(strict threshold) error = %v, want nil (zero boxes is a success)", err)
	}
	if text != "" {
		t.Errorf("ExtractWithParams(strict threshold) = %q, want the empty string", text)
	}
}

// TestExtractRejectsDegenerateRaster pins the raster contract: the
// refusal precedes any inference.
func TestExtractRejectsDegenerateRaster(t *testing.T) {
	p := fixturePipeline(t)

	bad := ml.Image{Width: 4, Height: 3, Pixels: make([]byte, 7)}
	if _, err := p.Extract(bad); err == nil {
		t.Error("Extract over a short raster error = nil, want the raster refusal")
	}
}

// TestExtractRealWeights is the optional real-model smoke run:
// LUMEMOR_TEST_OCR_DET_WEIGHTS and LUMEMOR_TEST_OCR_REC_WEIGHTS point
// at the downloaded PP-OCRv4 pair (tools/models/cache/ in a checkout),
// and the test drives the whole pipeline for real over the committed
// text-bearing image: detection, postprocess, reading order, crops and
// the CTC decode. The assertion is the measured decode of the
// committed fixture (the pipeline is deterministic end to end).
func TestExtractRealWeights(t *testing.T) {
	if !ml.RuntimeAvailable() {
		t.Skip("onnxruntime shared library not present")
	}
	detPath := os.Getenv("LUMEMOR_TEST_OCR_DET_WEIGHTS")
	recPath := os.Getenv("LUMEMOR_TEST_OCR_REC_WEIGHTS")
	if detPath == "" || recPath == "" {
		t.Skip("LUMEMOR_TEST_OCR_DET_WEIGHTS / LUMEMOR_TEST_OCR_REC_WEIGHTS not set; point them at the downloaded OCR weights to run the real-model smoke")
	}

	det, err := ml.NewSession(detPath, ocrDetWeightsSHA256)
	if err != nil {
		t.Fatalf("NewSession(det) error = %v, want nil", err)
	}
	t.Cleanup(func() { _ = det.Close() })
	rec, err := ml.NewSession(recPath, ocrRecWeightsSHA256)
	if err != nil {
		t.Fatalf("NewSession(rec) error = %v, want nil", err)
	}
	t.Cleanup(func() { _ = rec.Close() })

	img, err := decodeTextFixture(t)
	if err != nil {
		t.Fatalf("decode the text fixture: %v", err)
	}
	text, err := NewPipeline(det, rec).Extract(img)
	if err != nil {
		t.Fatalf("Extract(real weights) error = %v, want nil", err)
	}
	t.Logf("the real pipeline decoded %q", text)
	if strings.TrimSpace(text) == "" {
		t.Fatalf("the real pipeline decoded an empty text from the text-bearing fixture")
	}
	if !strings.Contains(text, "2026") {
		t.Errorf("the real pipeline decoded %q, want it to contain the rendered 2026", text)
	}
}

// decodeTextFixture decodes the committed text-bearing PNG into the
// RGB raster contract (the stdlib decoder; the package stays free of
// the govips dependency).
func decodeTextFixture(t *testing.T) (ml.Image, error) {
	t.Helper()
	file, err := os.Open(filepath.Join("..", "testdata", "ocr-text.png"))
	if err != nil {
		return ml.Image{}, err
	}
	defer func() { _ = file.Close() }()
	decoded, err := png.Decode(file)
	if err != nil {
		return ml.Image{}, err
	}
	bounds := decoded.Bounds()
	img := ml.Image{Width: bounds.Dx(), Height: bounds.Dy(), Pixels: make([]byte, bounds.Dx()*bounds.Dy()*3)}
	for y := 0; y < img.Height; y++ {
		for x := 0; x < img.Width; x++ {
			r, g, b, _ := decoded.At(bounds.Min.X+x, bounds.Min.Y+y).RGBA()
			i := (y*img.Width + x) * 3
			img.Pixels[i] = byte(r >> 8)
			img.Pixels[i+1] = byte(g >> 8)
			img.Pixels[i+2] = byte(b >> 8)
		}
	}
	return img, nil
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
