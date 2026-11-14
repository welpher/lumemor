//go:build cgo

package media

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/welpher/lumemor/server/internal/storage"
)

// requireThumbDimensions decodes the produced AVIF of the given size
// and pins its dimensions: the assertions read the actual encoded
// files, not the in-memory references.
func requireThumbDimensions(t *testing.T, assetID, size, ext string, wantWidth, wantHeight int) {
	t.Helper()
	path := storage.ThumbPath(assetID, size, ext)
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("thumbnail %s missing: %v", path, err)
	}
	width, height, err := ImageDimensions(path)
	if err != nil {
		t.Fatalf("decode %s: %v", path, err)
	}
	if width != wantWidth || height != wantHeight {
		t.Errorf("%s = %dx%d, want %dx%d", path, width, height, wantWidth, wantHeight)
	}
}

// TestGenerateThumbnailsSizes pins the three-size contract on a
// 1600x800 source: the tile fits within the 250px box (250x125), the
// view caps the long edge at 1080 (1080x540), and the original keeps
// the source dimensions (1600x800) -- all three re-encoded as AVIF.
func TestGenerateThumbnailsSizes(t *testing.T) {
	dataDir := newMediaEnv(t)
	assetID := writeOriginal(t, dataDir, "large.jpg", fixtureBytes(t, "golden-large.jpg"))

	hash, err := GenerateThumbnails(filepath.Join(dataDir, "originals", "aa", "bb", "large.jpg"), assetID)
	if err != nil {
		t.Fatalf("GenerateThumbnails: %v", err)
	}
	if len(hash) < 6 {
		t.Errorf("blurhash = %q, want a non-empty encoded hash", hash)
	}

	requireThumbDimensions(t, assetID, SizeTile, "avif", 250, 125)
	requireThumbDimensions(t, assetID, SizeView, "avif", 1080, 540)
	requireThumbDimensions(t, assetID, SizeOriginal, "avif", 1600, 800)
}

// TestGenerateThumbnailsNeverUpscales pins the no-upscale rule: a
// 64x64 source (the pipeline's golden JPEG) produces 64x64 in every
// size, including the original -- the serving copies never grow
// pixels the source does not have.
func TestGenerateThumbnailsNeverUpscales(t *testing.T) {
	dataDir := newMediaEnv(t)
	golden, err := os.ReadFile(filepath.Join("..", "jobs", "testdata", "golden.jpg"))
	if err != nil {
		t.Fatalf("read golden fixture: %v", err)
	}
	assetID := writeOriginal(t, dataDir, "golden.jpg", golden)

	if _, err := GenerateThumbnails(filepath.Join(dataDir, "originals", "aa", "bb", "golden.jpg"), assetID); err != nil {
		t.Fatalf("GenerateThumbnails: %v", err)
	}

	requireThumbDimensions(t, assetID, SizeTile, "avif", 64, 64)
	requireThumbDimensions(t, assetID, SizeView, "avif", 64, 64)
	requireThumbDimensions(t, assetID, SizeOriginal, "avif", 64, 64)
}

// TestGenerateThumbnailsHEIC pins the HEIC decode path: a real HEIC
// (HEVC-in-HEIF, decoded through libheif) derives the same three sizes
// as its JPEG twin. The fixture is a committed file crafted inside the
// dev container; hosts whose libvips lacks the libheif decoder skip.
func TestGenerateThumbnailsHEIC(t *testing.T) {
	if !HEICDecodeSupported() {
		t.Skip("this libvips build carries no HEIC decoder (libheif); run the suite inside the dev container (make docker-test)")
	}
	dataDir := newMediaEnv(t)
	assetID := writeOriginal(t, dataDir, "large.heic", fixtureBytes(t, "golden-large.heic"))

	hash, err := GenerateThumbnails(filepath.Join(dataDir, "originals", "aa", "bb", "large.heic"), assetID)
	if err != nil {
		t.Fatalf("GenerateThumbnails: %v", err)
	}
	if len(hash) < 6 {
		t.Errorf("blurhash = %q, want a non-empty encoded hash", hash)
	}

	requireThumbDimensions(t, assetID, SizeTile, "avif", 250, 125)
	requireThumbDimensions(t, assetID, SizeView, "avif", 1080, 540)
	requireThumbDimensions(t, assetID, SizeOriginal, "avif", 1600, 800)
}

// TestGenerateThumbnailsOrientation pins the orientation handling: a
// 200x100 sensor image carrying EXIF orientation 6 (rotate 90 CW)
// renders as 100x200, in every derived size -- the tile's dimensions
// swap, exactly as the render-side auto-rotate dictates (the metadata
// job, by contrast, keeps the sensor order).
func TestGenerateThumbnailsOrientation(t *testing.T) {
	dataDir := newMediaEnv(t)
	assetID := writeOriginal(t, dataDir, "portrait.jpg", fixtureBytes(t, "portrait-orientation6.jpg"))

	if _, err := GenerateThumbnails(filepath.Join(dataDir, "originals", "aa", "bb", "portrait.jpg"), assetID); err != nil {
		t.Fatalf("GenerateThumbnails: %v", err)
	}

	requireThumbDimensions(t, assetID, SizeTile, "avif", 100, 200)
	requireThumbDimensions(t, assetID, SizeView, "avif", 100, 200)
	requireThumbDimensions(t, assetID, SizeOriginal, "avif", 100, 200)
}

// TestGenerateThumbnailsOverwritesSamePaths pins the idempotency of
// the derivation itself: a second run over the same source rewrites
// exactly the same three paths -- the file set stays the singleton
// tile/view/original, never a duplicate beside them.
func TestGenerateThumbnailsOverwritesSamePaths(t *testing.T) {
	dataDir := newMediaEnv(t)
	assetID := writeOriginal(t, dataDir, "large.jpg", fixtureBytes(t, "golden-large.jpg"))
	path := filepath.Join(dataDir, "originals", "aa", "bb", "large.jpg")

	if _, err := GenerateThumbnails(path, assetID); err != nil {
		t.Fatalf("first GenerateThumbnails: %v", err)
	}
	if _, err := GenerateThumbnails(path, assetID); err != nil {
		t.Fatalf("second GenerateThumbnails: %v", err)
	}

	entries, err := os.ReadDir(filepath.Dir(storage.ThumbPath(assetID, SizeTile, "avif")))
	if err != nil {
		t.Fatalf("list thumbnail directory: %v", err)
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	want := []string{"original.avif", "tile.avif", "view.avif"}
	if len(names) != len(want) {
		t.Fatalf("thumbnail files = %v, want exactly %v", names, want)
	}
	for i := range want {
		if names[i] != want[i] {
			t.Errorf("thumbnail files = %v, want exactly %v", names, want)
		}
	}
	requireThumbDimensions(t, assetID, SizeTile, "avif", 250, 125)
}
