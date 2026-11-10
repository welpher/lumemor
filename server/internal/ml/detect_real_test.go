// The real-model smoke test: the committed YuNet weights
// and the real shared library, driven through the same session surface
// the detection job will use. Only the tensor plumbing is pinned here
// -- the graph's fixed input geometry, the anchor-free output set and
// its shapes. Decoding boxes and any face semantics belong to the
// detection wrapper (the committed face-portrait.jpg fixture
// is that wrapper's input, not this test's).
package ml

import (
	"path/filepath"
	"slices"
	"testing"

	ort "github.com/yalue/onnxruntime_go"
)

// yunetFixtureSHA256 pins the committed YuNet weights to the upstream
// 2023-03 release bytes (the digest the default manifest also pins).
const yunetFixtureSHA256 = "8f2383e4dd3cfbb4553ea8718107fc0423210dc964f9f4280604804ed2552fa4"

// wantYunetOutput pins the anchor-free output set of the 2023-03
// detector: three scales (strides 8, 16, 32 over a 640x640 input --
// 6400, 1600 and 400 anchor rows) times the three head kinds, with
// bbox carrying 4 distances and kps 10 landmark coordinates.
var wantYunetOutput = map[string][2]int64{
	"cls_8":   {6400, 1},
	"cls_16":  {1600, 1},
	"cls_32":  {400, 1},
	"obj_8":   {6400, 1},
	"obj_16":  {1600, 1},
	"obj_32":  {400, 1},
	"bbox_8":  {6400, 4},
	"bbox_16": {1600, 4},
	"bbox_32": {400, 4},
	"kps_8":   {6400, 10},
	"kps_16":  {1600, 10},
	"kps_32":  {400, 10},
}

// TestYunetBareInference runs the real detector graph end to end:
// checksum-verified load, session creation, one native Run of a
// 1x3x640x640 input, and the full anchor-free output set coming back
// non-empty with the expected shapes. The 2023mar graph pins its input
// at 640x640 -- larger than the manifest's declared inputSize -- which
// the detection wrapper has to own when it resizes; this test only proves
// the runtime, the binding and the weights agree.
func TestYunetBareInference(t *testing.T) {
	if !RuntimeAvailable() {
		t.Skip("onnxruntime shared library not present")
	}

	sess, err := NewSession(filepath.Join("testdata", "face_detection_yunet_2023mar.onnx"), yunetFixtureSHA256)
	if err != nil {
		t.Fatalf("NewSession(yunet) error = %v, want nil", err)
	}
	defer func() { _ = sess.Close() }()

	inputs := sess.Inputs()
	if len(inputs) != 1 {
		t.Fatalf("session reports %d inputs, want 1", len(inputs))
	}
	if inputs[0].Name != "input" {
		t.Errorf("input name = %q, want %q", inputs[0].Name, "input")
	}
	if want := (ort.Shape{1, 3, 640, 640}); !slices.Equal(inputs[0].Dimensions, want) {
		t.Errorf("input shape = %v, want %v", inputs[0].Dimensions, want)
	}

	outputs := sess.Outputs()
	if len(outputs) != len(wantYunetOutput) {
		t.Fatalf("session reports %d outputs, want %d", len(outputs), len(wantYunetOutput))
	}

	input, err := ort.NewTensor(ort.NewShape(1, 3, 640, 640), make([]float32, 1*3*640*640))
	if err != nil {
		t.Fatalf("NewTensor error = %v, want nil", err)
	}
	defer func() { _ = input.Destroy() }()

	// All nil outputs: the runtime allocates each one; Run replaces
	// them with the Go-managed tensors.
	values := make([]ort.Value, len(outputs))
	if err := sess.Run([]ort.Value{input}, values); err != nil {
		t.Fatalf("Run error = %v, want nil", err)
	}
	for _, v := range values {
		t.Cleanup(func() { _ = v.Destroy() })
	}

	for i, info := range outputs {
		want, ok := wantYunetOutput[info.Name]
		if !ok {
			t.Errorf("unexpected output %q in the YuNet graph", info.Name)
			continue
		}
		tensor := values[i]
		if got := tensor.GetShape(); len(got) != 3 || got[0] != 1 || got[1] != want[0] || got[2] != want[1] {
			t.Errorf("output %q shape = %v, want [1 %d %d]", info.Name, got, want[0], want[1])
			continue
		}
		if size := tensor.GetShape().FlattenedSize(); size <= 0 {
			t.Errorf("output %q is empty (flattened size %d)", info.Name, size)
		}
	}
}
