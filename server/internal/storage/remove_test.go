// RemoveAssetFiles tests (M2c Task 3): the on-disk half of the
// permanent delete -- the derived directories always go, the
// content-addressed original only at reference count zero, and a
// missing file is tolerated, never an error.
package storage

import (
	"os"
	"path/filepath"
	"testing"
)

// writeLayoutFixture materializes the on-disk state of one asset: the
// original under the data root's originals tree and the derived
// thumbs/encoded directories, returning the paths that matter.
func writeLayoutFixture(t *testing.T, dataDir, rel, assetID string) (original, thumbs, encoded string) {
	t.Helper()
	original = filepath.Join(dataDir, filepath.FromSlash(rel))
	thumbs = filepath.Join(dataDir, "storage", "thumbs", assetID, "tile.avif")
	encoded = filepath.Join(dataDir, "storage", "encoded", assetID, "play.mp4")
	for _, path := range []string{original, thumbs, encoded} {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("create fixture dir of %s: %v", path, err)
		}
		if err := os.WriteFile(path, []byte("fixture "+path), 0o644); err != nil {
			t.Fatalf("write fixture %s: %v", path, err)
		}
	}
	return original, thumbs, encoded
}

func TestRemoveAssetFilesSoleReferenceRemovesEverything(t *testing.T) {
	dataDir := t.TempDir()
	rel := "originals/ab/cd/abcdef.jpg"
	original, thumbs, encoded := writeLayoutFixture(t, dataDir, rel, "asset-1")

	if err := RemoveAssetFiles(dataDir, rel, "asset-1", 0); err != nil {
		t.Fatalf("RemoveAssetFiles() error = %v, want nil", err)
	}
	for _, path := range []string{original, thumbs, encoded} {
		if _, err := os.Stat(path); err == nil {
			t.Errorf("%s survived, want it removed", path)
		}
	}
}

func TestRemoveAssetFilesSharedReferenceKeepsOriginal(t *testing.T) {
	dataDir := t.TempDir()
	rel := "originals/ab/cd/abcdef.jpg"
	original, thumbs, _ := writeLayoutFixture(t, dataDir, rel, "asset-2")

	// Another row still references the file: only the row's own
	// derived directories may go.
	if err := RemoveAssetFiles(dataDir, rel, "asset-2", 1); err != nil {
		t.Fatalf("RemoveAssetFiles() error = %v, want nil", err)
	}
	if _, err := os.Stat(original); err != nil {
		t.Errorf("the shared original was removed, want it kept: %v", err)
	}
	if _, err := os.Stat(thumbs); err == nil {
		t.Error("the derived thumbs file survived, want the directory removed")
	}
}

func TestRemoveAssetFilesToleratesMissingArtifacts(t *testing.T) {
	dataDir := t.TempDir()
	// Nothing was ever written for this asset (its pipeline never
	// ran, its file was already removed by a concurrent delete): the
	// removal is a no-op, not an error.
	if err := RemoveAssetFiles(dataDir, "originals/ab/cd/never-here.jpg", "asset-3", 0); err != nil {
		t.Fatalf("RemoveAssetFiles() error = %v, want the tolerated no-op", err)
	}
}

func TestRemoveAssetFilesRefusesEmptyAssetID(t *testing.T) {
	dataDir := t.TempDir()
	thumbsRoot := filepath.Join(dataDir, "storage", "thumbs")
	if err := os.MkdirAll(thumbsRoot, 0o755); err != nil {
		t.Fatalf("create thumbs root: %v", err)
	}
	if err := RemoveAssetFiles(dataDir, "originals/ab/cd/x.jpg", "", 0); err == nil {
		t.Fatal("RemoveAssetFiles() with an empty asset id succeeded, want the refusal")
	}
	if _, err := os.Stat(thumbsRoot); err != nil {
		t.Errorf("the thumbs root was touched: %v", err)
	}
}

func TestRemoveAssetFilesRefusesPathOutsideOriginals(t *testing.T) {
	dataDir := t.TempDir()
	// A corrupted row whose file_path escapes the originals tree: the
	// guard refuses, and the derived directories still go.
	rel := "storage/sidecars/evil.json"
	_, thumbs, _ := writeLayoutFixture(t, dataDir, rel, "asset-4")

	if err := RemoveAssetFiles(dataDir, rel, "asset-4", 0); err == nil {
		t.Fatal("RemoveAssetFiles() of an escaping path succeeded, want the refusal")
	}
	if _, err := os.Stat(thumbs); err == nil {
		t.Error("the derived thumbs file survived the refusal, want the directories removed")
	}
}
