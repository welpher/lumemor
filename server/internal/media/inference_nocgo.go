//go:build !cgo

package media

// The CGO-less stub of the ML inference decode; see inference.go for
// the libvips half. The stub refuses: a build without libvips cannot
// decode originals for the ML jobs, and failing the job with a clear
// error beats handing the wrappers an empty raster.

import (
	"errors"

	"github.com/welpher/lumemor/server/internal/ml"
)

// errNoInferenceDecode is the CGO-less stub's refusal.
var errNoInferenceDecode = errors.New("media: the inference decode requires a CGO build with libvips")

// DecodeInferenceImage is the CGO-less stub: the decode is unavailable
// without libvips.
func DecodeInferenceImage(originalPath string, longEdge int) (ml.Image, error) {
	return ml.Image{}, errNoInferenceDecode
}
