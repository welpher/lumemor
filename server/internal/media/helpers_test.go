package media

import (
	"image/jpeg"
	"os"
	"path/filepath"
	"testing"

	"github.com/welpher/lumemor/server/internal/storage"
)

// newMediaEnv prepares a storage layout for one derivation test and
// returns its data root. The path helpers of the storage package
// resolve against it (storage.ThumbPath, storage.PosterPath).
func newMediaEnv(t *testing.T) string {
	t.Helper()
	dataDir := t.TempDir()
	if err := storage.EnsureLayout(dataDir, t.TempDir()); err != nil {
		t.Fatalf("prepare storage layout: %v", err)
	}
	return dataDir
}

// fixtureBytes reads a committed test fixture of this package.
func fixtureBytes(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return data
}

// writeOriginal stores a fixture under the content-addressed originals
// tree of the data root, mirroring what the upload receiver leaves
// behind, and returns the asset id to derive for.
func writeOriginal(t *testing.T, dataDir string, name string, content []byte) string {
	t.Helper()
	assetID := "11111111-2222-3333-4444-555555555555"
	path := filepath.Join(dataDir, "originals", "aa", "bb", name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("create originals directory: %v", err)
	}
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("write original %s: %v", path, err)
	}
	return assetID
}

// posterDims decodes the poster JPEG through the standard library and
// returns its dimensions.
func posterDims(t *testing.T, path string) (int, int) {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("open poster %s: %v", path, err)
	}
	defer func() { _ = file.Close() }()
	poster, err := jpeg.Decode(file)
	if err != nil {
		t.Fatalf("decode poster %s: %v", path, err)
	}
	bounds := poster.Bounds()
	return bounds.Dx(), bounds.Dy()
}
