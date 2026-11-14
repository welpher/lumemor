//go:build cgo

package media

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/welpher/lumemor/server/internal/storage"
)

// TestDeriveWithOptionsHonorsSizesAndFormat pins the derivation's
// parameterization on a 1600x800 source: the tile and view sizes
// follow the settings (320 box, 1440 long edge), the format's file
// extension carries the encoding, and the full-size copy exists when
// the settings ask for it.
func TestDeriveWithOptionsHonorsSizesAndFormat(t *testing.T) {
	dataDir := newMediaEnv(t)
	assetID := writeOriginal(t, dataDir, "tuned.jpg", fixtureBytes(t, "golden-large.jpg"))

	hash, err := GenerateThumbnailsWithOptions(
		filepath.Join(dataDir, "originals", "aa", "bb", "tuned.jpg"),
		assetID,
		DeriveOptions{TileBox: 320, ViewLongEdge: 1440, Format: FormatWebp, FullSize: true, RawEmbedded: true},
	)
	if err != nil {
		t.Fatalf("GenerateThumbnailsWithOptions: %v", err)
	}
	if len(hash) < 6 {
		t.Errorf("blurhash = %q, want a non-empty encoded hash", hash)
	}

	requireThumbDimensions(t, assetID, SizeTile, "webp", 320, 160)
	requireThumbDimensions(t, assetID, SizeView, "webp", 1440, 720)
	requireThumbDimensions(t, assetID, SizeOriginal, "webp", 1600, 800)
}

// TestDeriveWithOptionsFullSizeOffSkipsOriginal pins the full-size
// switch: off, the derivation produces the tile and the view and
// NOTHING else -- the full-resolution serving copy is disk the
// settings declined to spend.
func TestDeriveWithOptionsFullSizeOffSkipsOriginal(t *testing.T) {
	dataDir := newMediaEnv(t)
	assetID := writeOriginal(t, dataDir, "compact.jpg", fixtureBytes(t, "golden-large.jpg"))

	if _, err := GenerateThumbnailsWithOptions(
		filepath.Join(dataDir, "originals", "aa", "bb", "compact.jpg"),
		assetID,
		DeriveOptions{TileBox: 250, ViewLongEdge: 1080, Format: FormatAvif, FullSize: false, RawEmbedded: true},
	); err != nil {
		t.Fatalf("GenerateThumbnailsWithOptions: %v", err)
	}

	requireThumbDimensions(t, assetID, SizeTile, "avif", 250, 125)
	requireThumbDimensions(t, assetID, SizeView, "avif", 1080, 540)
	if _, err := os.Stat(storage.ThumbPath(assetID, SizeOriginal, "avif")); !os.IsNotExist(err) {
		t.Errorf("original derivative stat err = %v, want NotExist when fullSize is off", err)
	}
}

// TestDeriveWithOptionsUnknownFormatFails pins the format guard: an
// encoding the pipeline does not carry fails the derivation instead
// of silently encoding something else.
func TestDeriveWithOptionsUnknownFormatFails(t *testing.T) {
	dataDir := newMediaEnv(t)
	assetID := writeOriginal(t, dataDir, "odd.jpg", fixtureBytes(t, "golden-large.jpg"))

	_, err := GenerateThumbnailsWithOptions(
		filepath.Join(dataDir, "originals", "aa", "bb", "odd.jpg"),
		assetID,
		DeriveOptions{TileBox: 250, ViewLongEdge: 1080, Format: "jpegxl", FullSize: false, RawEmbedded: true},
	)
	if err == nil || !strings.Contains(err.Error(), "encoding") {
		t.Fatalf("GenerateThumbnailsWithOptions err = %v, want an unknown-encoding failure", err)
	}
}
