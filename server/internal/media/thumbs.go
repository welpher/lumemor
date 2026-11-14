//go:build cgo

// Package media derives the serving copies of an asset: the three
// AVIF thumbnail sizes of an image (tile, view, original) through
// libvips, the poster frame of a video through ffmpeg, the blurhash
// of the small tile, and the query-time face crop of a face row's
// bounding box out of the view derivative.
//
// This file carries the libvips half; it builds only with CGO enabled
// (the govips binding is a cgo wrapper around libvips). A build
// without CGO keeps the package usable through the stub in
// thumbs_nocgo.go, whose GenerateThumbnails reports the missing
// pipeline instead of failing the build.
package media

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/bbrks/go-blurhash"
	"github.com/davidbyttow/govips/v2/vips"

	"github.com/welpher/lumemor/server/internal/storage"
)

// The three serving sizes of an image asset, the <size> segment of
// storage.ThumbPath's storage/thumbs/<assetID>/<size>.avif layout.
const (
	// SizeTile is the timeline tile: fit within a 250px box.
	SizeTile = "tile"
	// SizeView is the inline view: long edge 1080 (never upscaled).
	SizeView = "view"
	// SizeOriginal is the full-resolution serving copy; the lossless
	// original stays in the content-addressed originals tree.
	SizeOriginal = "original"
)

// The pinned derivation parameters.
const (
	// tileBox is the tile's bounding box in pixels: the tile fits
	// within it, aspect preserved, never upscaled.
	tileBox = 250

	// viewLongEdge caps the view's long edge in pixels; the square
	// box form is equivalent to a long-edge bound for every aspect.
	viewLongEdge = 1080

	// avifQuality is the AVIF encode quality of all three sizes. The
	// serving copies are intentionally lossy: the lossless originals
	// remain in the originals tree, and Q50 keeps the derived files
	// small without visible degradation at these sizes.
	avifQuality = 50

	// webpQuality is the WebP encode quality of the serving copies,
	// the same posture as the AVIF quality: visually comparable at
	// these sizes, with the encoding the faster side of the trade.
	webpQuality = 80
)

// ThumbnailsSupported reports whether this build linked the libvips
// thumbnail pipeline. It is false only in CGO-less builds, where the
// media tests skip and the thumbnail job fails with a clear error.
const ThumbnailsSupported = true

// Startup initializes libvips for the process. main.go calls it at
// boot, before the job queue starts; the derivation functions also
// tolerate an uninitialized process by starting it lazily (Startup is
// idempotent). One-shot per process: after Shutdown, libvips cannot
// restart.
func Startup() error {
	return vips.Startup(nil)
}

// Shutdown releases libvips. main.go calls it after the job queue has
// drained, so no worker can be mid-derivation.
func Shutdown() {
	vips.Shutdown()
}

// LibvipsVersion reports the linked libvips version string (for the
// boot log).
func LibvipsVersion() string {
	return vips.Version
}

// DerivativeFormats are the encodings the derivation writes: the
// file extension equals the format name, so the format string is
// also the stored path's extension.
const (
	// FormatAvif is the shipped encoding: smaller files, slower
	// encodes.
	FormatAvif = "avif"
	// FormatWebp is the fast encoding: quicker encodes, a little
	// larger files.
	FormatWebp = "webp"
)

// ValidDerivativeFormat reports whether format is an encoding the
// derivation writes.
func ValidDerivativeFormat(format string) bool {
	return format == FormatAvif || format == FormatWebp
}

// GenerateThumbnails derives the serving copies of the image at
// originalPath at the SHIPPED settings -- the parameterized entry
// point is GenerateThumbnailsWithOptions, and this wrapper keeps the
// pinned defaults legible for the tests that exercise the shipped
// posture.
func GenerateThumbnails(originalPath, assetID string) (string, error) {
	return GenerateThumbnailsWithOptions(originalPath, assetID, DeriveOptions{
		TileBox:      tileBox,
		ViewLongEdge: viewLongEdge,
		Format:       FormatAvif,
		FullSize:     true,
		RawEmbedded:  true,
	})
}

// DeriveOptions carries the image settings one derivation runs at:
// the tile's bounding box, the view's long edge, the serving copies'
// encoding, whether the full-resolution copy is derived too, and the
// RAW-embedded-preview posture (consulted for RAW-typed sources;
// this pipeline derives every decode-able image the same way).
type DeriveOptions struct {
	TileBox      int
	ViewLongEdge int
	Format       string
	FullSize     bool
	RawEmbedded  bool
}

// GenerateThumbnailsWithOptions derives the serving copies of the
// image at originalPath -- storage/thumbs/<assetID>/{tile,view[,original]}.ext
// -- at the settings the options carry, and returns the blurhash
// computed from the tile. It runs idempotently: every invocation
// overwrites the same paths, so a re-run converges to the same file
// set without duplicates. The full-size copy derives only when the
// options ask for it: the viewer reads the preview, and the
// full-resolution copy is disk the settings may decline to spend.
//
// Orientation is handled here, at render time: the loaded image is
// auto-rotated once before any size is derived, so every copy
// presents the upright image (the metadata job, by contrast, stores
// the sensor-order dimensions the file reports). EXIF and every
// other metadata block are stripped from the serving copies: the
// database owns the extracted metadata.
func GenerateThumbnailsWithOptions(originalPath, assetID string, opts DeriveOptions) (string, error) {
	if !ValidDerivativeFormat(opts.Format) {
		return "", fmt.Errorf("unknown derivative encoding %q", opts.Format)
	}
	if err := os.MkdirAll(thumbDir(assetID), 0o755); err != nil {
		return "", fmt.Errorf("create thumbnail directory of asset %s: %w", assetID, err)
	}

	// libvips initializes lazily here when the process has not started
	// it explicitly (main.go does at boot); Startup is idempotent.
	if err := vips.Startup(nil); err != nil {
		return "", fmt.Errorf("start libvips: %w", err)
	}

	image, err := vips.NewImageFromFile(originalPath)
	if err != nil {
		return "", fmt.Errorf("decode %s: %w", originalPath, err)
	}
	defer image.Close()

	if err := image.AutoRotate(); err != nil {
		return "", fmt.Errorf("orient %s: %w", originalPath, err)
	}

	// The tile: fit within the box, never upscaled. The blurhash is
	// computed from this in-memory tile -- cheaper than decoding the
	// encoded file back.
	tile, err := image.Copy()
	if err != nil {
		return "", fmt.Errorf("copy for tile: %w", err)
	}
	defer tile.Close()
	if err := tile.ThumbnailWithSize(opts.TileBox, opts.TileBox, vips.InterestingNone, vips.SizeDown); err != nil {
		return "", fmt.Errorf("derive tile of %s: %w", originalPath, err)
	}
	hash, err := tileBlurhash(tile)
	if err != nil {
		return "", err
	}
	if err := writeDerivative(tile, storage.ThumbPath(assetID, SizeTile, opts.Format), opts.Format); err != nil {
		return "", err
	}

	// The view: long edge capped, never upscaled.
	view, err := image.Copy()
	if err != nil {
		return "", fmt.Errorf("copy for view: %w", err)
	}
	defer view.Close()
	if err := view.ThumbnailWithSize(opts.ViewLongEdge, opts.ViewLongEdge, vips.InterestingNone, vips.SizeDown); err != nil {
		return "", fmt.Errorf("derive view of %s: %w", originalPath, err)
	}
	if err := writeDerivative(view, storage.ThumbPath(assetID, SizeView, opts.Format), opts.Format); err != nil {
		return "", err
	}

	// The full-size copy: the full-resolution re-encode at the
	// configured encoding; no resize at all. Only a settings posture
	// that asks for it pays this disk.
	if opts.FullSize {
		if err := writeDerivative(image, storage.ThumbPath(assetID, SizeOriginal, opts.Format), opts.Format); err != nil {
			return "", err
		}
	}
	return hash, nil
}

// writeDerivative encodes ref at the configured encoding (metadata
// stripped) and writes it to path atomically: the bytes land in a
// temporary file beside the destination and are renamed into place,
// so a concurrent reader (the serving endpoint mid re-run) never
// observes a partial file. A re-run converges to the same path set.
func writeDerivative(ref *vips.ImageRef, path, format string) error {
	var buf []byte
	var err error
	switch format {
	case FormatWebp:
		buf, _, err = ref.ExportWebp(&vips.WebpExportParams{
			Quality:       webpQuality,
			StripMetadata: true,
		})
		if err != nil {
			return fmt.Errorf("encode WebP %s: %w", path, err)
		}
	default:
		buf, _, err = ref.ExportAvif(&vips.AvifExportParams{
			Quality:       avifQuality,
			StripMetadata: true,
		})
		if err != nil {
			return fmt.Errorf("encode AVIF %s: %w", path, err)
		}
	}
	if err := writeFileAtomic(path, buf); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

// tileBlurhash computes the blurhash of an already tile-sized image
// reference, straight from the in-memory pixels: no encode/decode
// round-trip.
func tileBlurhash(ref *vips.ImageRef) (string, error) {
	pixels, err := ref.ToGoImage()
	if err != nil {
		return "", fmt.Errorf("read tile pixels: %w", err)
	}
	hash, err := blurhash.Encode(blurhashXComponents, blurhashYComponents, pixels)
	if err != nil {
		return "", fmt.Errorf("encode blurhash: %w", err)
	}
	return hash, nil
}

// HEICDecodeSupported reports whether this build's libvips carries the
// libheif HEIC decoder. It starts libvips if the process has not (the
// support table is filled at startup).
func HEICDecodeSupported() bool {
	if err := vips.Startup(nil); err != nil {
		return false
	}
	return vips.IsTypeSupported(vips.ImageTypeHEIF)
}

// ImageDimensions decodes the image at path and returns its pixel
// dimensions. It exists for the pipeline's tests, which verify the
// produced AVIF files this way rather than trusting file existence.
func ImageDimensions(path string) (int, int, error) {
	if err := vips.Startup(nil); err != nil {
		return 0, 0, fmt.Errorf("start libvips: %w", err)
	}
	ref, err := vips.NewImageFromFile(path)
	if err != nil {
		return 0, 0, fmt.Errorf("decode %s: %w", path, err)
	}
	defer ref.Close()
	return ref.Width(), ref.Height(), nil
}

// FaceCropThumbnail derives the 256x256 AVIF face crop of one face
// from the serving copy at viewPath -- the asset's view derivative,
// never the original -- for a bounding box in per-mille units of the
// full image (the faces row's bbox_x/y/w/h). The bytes are computed
// at query time and never stored: they are a pure function of the
// derivative's pixels and the face row, which is what lets the
// endpoint answer with immutable caching.
//
// The window is FaceCropWindow's pinned square; the output renders to
// exactly faceCropEdge pixels -- upscaled when the window is smaller,
// because a face crop is a zoom, not a fit (the serving sizes' no-
// upscale rule does not apply here).
func FaceCropThumbnail(viewPath string, bboxX, bboxY, bboxW, bboxH int) ([]byte, error) {
	// libvips initializes lazily here when the process has not started
	// it explicitly (main.go does at boot); Startup is idempotent.
	if err := vips.Startup(nil); err != nil {
		return nil, fmt.Errorf("start libvips: %w", err)
	}

	image, err := vips.NewImageFromFile(viewPath)
	if err != nil {
		return nil, fmt.Errorf("decode %s: %w", viewPath, err)
	}
	defer image.Close()

	left, top, side := FaceCropWindow(image.Width(), image.Height(), bboxX, bboxY, bboxW, bboxH)
	if err := image.ExtractArea(left, top, side, side); err != nil {
		return nil, fmt.Errorf("crop face window of %s: %w", viewPath, err)
	}
	if err := image.ThumbnailWithSize(faceCropEdge, faceCropEdge, vips.InterestingCentre, vips.SizeBoth); err != nil {
		return nil, fmt.Errorf("resize face crop of %s: %w", viewPath, err)
	}

	buf, _, err := image.ExportAvif(&vips.AvifExportParams{
		Quality:       avifQuality,
		StripMetadata: true,
	})
	if err != nil {
		return nil, fmt.Errorf("encode AVIF face crop of %s: %w", viewPath, err)
	}
	return buf, nil
}

// ImageAverageColor decodes the image at path and answers the mean
// of its red, green and blue channels (0-255 scale) over all pixels,
// in sRGB. It exists for the pipeline's and the endpoint's tests,
// which verify a crop's pixel REGION by averaging the encoded output
// -- solid-color regions survive the lossy encode nearly exactly, so
// a quadrant's mean identifies the region the window covered (the
// same role ImageDimensions plays for size assertions).
func ImageAverageColor(path string) (float64, float64, float64, error) {
	if err := vips.Startup(nil); err != nil {
		return 0, 0, 0, fmt.Errorf("start libvips: %w", err)
	}
	ref, err := vips.NewImageFromFile(path)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("decode %s: %w", path, err)
	}
	defer ref.Close()
	goImage, err := ref.ToGoImage()
	if err != nil {
		return 0, 0, 0, fmt.Errorf("read pixels of %s: %w", path, err)
	}

	var sumR, sumG, sumB, pixels float64
	bounds := goImage.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, _ := goImage.At(x, y).RGBA()
			sumR += float64(r) / 65535
			sumG += float64(g) / 65535
			sumB += float64(b) / 65535
			pixels++
		}
	}
	if pixels == 0 {
		return 0, 0, 0, fmt.Errorf("average an empty image %s", path)
	}
	return sumR / pixels * 255, sumG / pixels * 255, sumB / pixels * 255, nil
}

// thumbDir returns the thumbnail directory of an asset.
func thumbDir(assetID string) string {
	return filepath.Dir(storage.ThumbPath(assetID, SizeTile, FormatAvif))
}
