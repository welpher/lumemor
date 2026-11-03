// The CLIP embedding wrapper (M3a Task 8): it feeds one decoded raster
// to a CLIP vision tower and answers the L2-normalized image feature
// the similarity search of a later milestone consumes.
//
// The preprocessing follows the CLIP family's public convention,
// implemented here from scratch: the raster resizes to the session's
// declared input geometry (a dynamic-geometry graph receives the
// raster's own size), the channels scale to [0, 1] and normalize with
// the standard CLIP mean and std below (the values the default group's
// manifest also pins as display truth), and the tensor layout is RGB
// NCHW float. The feature is L2-normalized defensively: the vision
// tower's card already claims unit-length outputs, but the wrapper's
// contract is the unit vector, so a graph that drifts from the claim
// still answers a comparable direction (and the wiring tests observe
// the normalization through a fixture that deliberately emits a
// non-unit vector).
package mlclip

import (
	"fmt"
	"math"

	ort "github.com/yalue/onnxruntime_go"

	"github.com/welpher/lumemor/server/internal/ml"
)

// clipMean and clipStd are the CLIP family's channel normalization
// constants: the means and standard deviations of the ImageNet-scale
// statistics the vision towers were trained against.
var (
	clipMean = [3]float64{0.48145466, 0.4578275, 0.40821073}
	clipStd  = [3]float64{0.26862954, 0.26130258, 0.27577711}
)

// EmbedImage feeds the raster to the vision tower and answers its
// image feature, L2-normalized. The output's dimension is the graph's
// own (the model declares it; the caller's dimension guards compare
// against the database's column, never an assumption).
func EmbedImage(sess *ml.Session, img ml.Image) ([]float32, error) {
	if err := validateRaster(img); err != nil {
		return nil, err
	}
	if len(sess.Outputs()) != 1 {
		return nil, fmt.Errorf("mlclip: the vision tower carries %d outputs, want 1", len(sess.Outputs()))
	}
	width, height, err := embedGeometry(sess, img)
	if err != nil {
		return nil, err
	}

	tensor, err := ort.NewTensor(ort.NewShape(1, 3, int64(height), int64(width)),
		clipCHW(ml.ResizeRaster(img, width, height)))
	if err != nil {
		return nil, fmt.Errorf("mlclip: build the embedding input tensor: %w", err)
	}
	defer func() { _ = tensor.Destroy() }()

	outputs := make([]ort.Value, 1)
	if err := sess.Run([]ort.Value{tensor}, outputs); err != nil {
		return nil, fmt.Errorf("mlclip: run the vision tower: %w", err)
	}
	defer func() { _ = outputs[0].Destroy() }()

	name := sess.Outputs()[0].Name
	feature, err := tensorData(outputs[0], name)
	if err != nil {
		return nil, err
	}
	if len(feature) == 0 {
		return nil, fmt.Errorf("mlclip: the vision tower %s emitted an empty feature", name)
	}

	norm := 0.0
	for _, component := range feature {
		norm += float64(component) * float64(component)
	}
	if norm == 0 {
		return nil, fmt.Errorf("mlclip: the vision tower %s emitted a zero vector; it cannot be normalized", name)
	}
	scale := float32(1 / math.Sqrt(norm))
	unit := make([]float32, len(feature))
	for i, component := range feature {
		unit[i] = component * scale
	}
	return unit, nil
}

// embedGeometry resolves the tensor geometry for one raster: the
// session's declared input when the graph pins it, the raster's own
// size when the graph leaves it dynamic. The input must be the single
// NCHW image tensor.
func embedGeometry(sess *ml.Session, img ml.Image) (width, height int, err error) {
	inputs := sess.Inputs()
	if len(inputs) != 1 {
		return 0, 0, fmt.Errorf("mlclip: the vision tower carries %d inputs, want 1", len(inputs))
	}
	dims := inputs[0].Dimensions
	if len(dims) != 4 {
		return 0, 0, fmt.Errorf("mlclip: the vision tower input has %d dimensions, want NCHW (4)", len(dims))
	}
	if dims[0] != 1 && dims[0] > 0 {
		return 0, 0, fmt.Errorf("mlclip: the vision tower input batch is %d, want 1", dims[0])
	}
	if dims[1] != 3 {
		return 0, 0, fmt.Errorf("mlclip: the vision tower input carries %d channels, want 3", dims[1])
	}
	if dims[2] > 0 && dims[3] > 0 {
		return int(dims[3]), int(dims[2]), nil
	}
	// Dynamic extent: the graph accepts any geometry, so the raster's
	// own.
	return img.Width, img.Height, nil
}

// clipCHW flattens an RGB raster into the tower's channel-first float
// tensor data: [0, 1] amplitude, per-channel CLIP mean and std.
func clipCHW(img ml.Image) []float32 {
	plane := img.Width * img.Height
	data := make([]float32, 3*plane)
	for i := 0; i < plane; i++ {
		for c := 0; c < 3; c++ {
			value := float64(img.Pixels[i*3+c]) / 255.0
			data[c*plane+i] = float32((value - clipMean[c]) / clipStd[c])
		}
	}
	return data
}

// validateRaster checks the ml.Image contract before any work.
func validateRaster(img ml.Image) error {
	if img.Width <= 0 || img.Height <= 0 {
		return fmt.Errorf("mlclip: raster is %dx%d, want positive dimensions", img.Width, img.Height)
	}
	if len(img.Pixels) != img.Width*img.Height*3 {
		return fmt.Errorf("mlclip: raster carries %d bytes, want %d for %dx%d RGB",
			len(img.Pixels), img.Width*img.Height*3, img.Width, img.Height)
	}
	return nil
}

// tensorData reads one output value as a float32 tensor.
func tensorData(value ort.Value, name string) ([]float32, error) {
	tensor, ok := value.(*ort.Tensor[float32])
	if !ok {
		return nil, fmt.Errorf("mlclip: output %q is %T, want a float32 tensor", name, value)
	}
	return tensor.GetData(), nil
}
