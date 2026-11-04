//go:build !cgo

// Package media derives the serving copies of an asset. See
// thumbs.go for the libvips half; this file is the CGO-less stub that
// keeps the package (and everything importing it) buildable without a
// C toolchain or libvips. The stub refuses to start: a server without
// the thumbnail pipeline is broken enough to fail boot rather than
// accept uploads whose serving copies can never be derived.
package media

import "errors"

// The size names exist in every build: the poster path and the
// thumbnail paths share the storage/thumbs/<assetID> directory.
const (
	// SizeTile is the timeline tile: fit within a 250px box.
	SizeTile = "tile"
	// SizeView is the inline view: long edge 1080 (never upscaled).
	SizeView = "view"
	// SizeOriginal is the full-resolution serving copy; the lossless
	// original stays in the content-addressed originals tree.
	SizeOriginal = "original"
)

// ThumbnailsSupported reports whether this build linked the libvips
// thumbnail pipeline: false here, so the media tests skip with a
// pointer at the containerized suite.
const ThumbnailsSupported = false

// Startup is the CGO-less stub: a build without libvips refuses to
// boot rather than running a server whose thumbnail pipeline is
// silently broken.
func Startup() error {
	return errNoThumbnails
}

// Shutdown is the CGO-less stub: nothing was started.
func Shutdown() {}

// LibvipsVersion reports the linked libvips version string: none in a
// CGO-less build.
func LibvipsVersion() string {
	return ""
}

// errNoThumbnails is the CGO-less stub's refusal.
var errNoThumbnails = errors.New("media: thumbnail generation requires a CGO build with libvips")

// GenerateThumbnails is the CGO-less stub: the AVIF derivation is
// unavailable without libvips.
func GenerateThumbnails(originalPath, assetID string) (string, error) {
	return "", errNoThumbnails
}

// HEICDecodeSupported reports whether this build's libvips carries the
// libheif HEIC decoder: never in a CGO-less build.
func HEICDecodeSupported() bool {
	return false
}

// ImageDimensions is the CGO-less stub: images cannot be decoded
// without libvips.
func ImageDimensions(path string) (int, int, error) {
	return 0, 0, errNoThumbnails
}

// FaceCropThumbnail is the CGO-less stub: the query-time face crop
// derivation is unavailable without libvips.
func FaceCropThumbnail(viewPath string, bboxX, bboxY, bboxW, bboxH int) ([]byte, error) {
	return nil, errNoThumbnails
}

// ImageAverageColor is the CGO-less stub: pixels cannot be read
// without libvips.
func ImageAverageColor(path string) (float64, float64, float64, error) {
	return 0, 0, 0, errNoThumbnails
}
