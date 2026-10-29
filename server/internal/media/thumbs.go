//go:build cgo

// Package media derives the serving copies of an asset: the three
// AVIF thumbnail sizes of an image (tile, view, original) through
// libvips, the poster frame of a video through ffmpeg, and the
// blurhash of the small tile.
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

// GenerateThumbnails derives the three AVIF serving copies of the
// image at originalPath -- storage/thumbs/<assetID>/{tile,view,original}.avif
// -- and returns the blurhash computed from the tile. It runs
// idempotently: every invocation overwrites the same three paths, so
// a re-run converges to the same file set without duplicates.
//
// Orientation is handled here, at render time: the loaded image is
// auto-rotated once before any size is derived, so all three copies
// present the upright image (the metadata job, by contrast, stores
// the sensor-order dimensions the file reports). EXIF and every other
// metadata block are stripped from the serving copies: the database
// owns the extracted metadata.
func GenerateThumbnails(originalPath, assetID string) (string, error) {
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
	// encoded AVIF back.
	tile, err := image.Copy()
	if err != nil {
		return "", fmt.Errorf("copy for tile: %w", err)
	}
	defer tile.Close()
	if err := tile.ThumbnailWithSize(tileBox, tileBox, vips.InterestingNone, vips.SizeDown); err != nil {
		return "", fmt.Errorf("derive tile of %s: %w", originalPath, err)
	}
	hash, err := tileBlurhash(tile)
	if err != nil {
		return "", err
	}
	if err := writeAvif(tile, storage.ThumbPath(assetID, SizeTile)); err != nil {
		return "", err
	}

	// The view: long edge capped, never upscaled.
	view, err := image.Copy()
	if err != nil {
		return "", fmt.Errorf("copy for view: %w", err)
	}
	defer view.Close()
	if err := view.ThumbnailWithSize(viewLongEdge, viewLongEdge, vips.InterestingNone, vips.SizeDown); err != nil {
		return "", fmt.Errorf("derive view of %s: %w", originalPath, err)
	}
	if err := writeAvif(view, storage.ThumbPath(assetID, SizeView)); err != nil {
		return "", err
	}

	// The original: the full-resolution re-encode; no resize at all.
	if err := writeAvif(image, storage.ThumbPath(assetID, SizeOriginal)); err != nil {
		return "", err
	}
	return hash, nil
}

// writeAvif encodes ref as AVIF (Q50, metadata stripped) and writes it
// to path, overwriting whatever a previous run left there.
func writeAvif(ref *vips.ImageRef, path string) error {
	buf, _, err := ref.ExportAvif(&vips.AvifExportParams{
		Quality:       avifQuality,
		StripMetadata: true,
	})
	if err != nil {
		return fmt.Errorf("encode AVIF %s: %w", path, err)
	}
	if err := os.WriteFile(path, buf, 0o644); err != nil {
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

// thumbDir returns the thumbnail directory of an asset.
func thumbDir(assetID string) string {
	return filepath.Dir(storage.ThumbPath(assetID, SizeTile))
}
