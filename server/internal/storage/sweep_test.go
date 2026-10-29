package storage

import (
	"os"
	"path/filepath"
	"testing"
)

// TestSweepTmpClearsStagingButKeepsLayout verifies the boot-time sweep
// of the upload staging directory: every file and subdirectory under
// <dataDir>/tmp is removed unconditionally (a single-process boot has
// no in-flight uploads, so leftovers can only be crash debris), the
// tmp directory itself survives, and the rest of the layout is
// untouched.
func TestSweepTmpClearsStagingButKeepsLayout(t *testing.T) {
	dataDir, tmpDir := setupLayout(t)

	// Crash debris in staging: a bare file plus a nested directory
	// holding another one.
	if err := os.WriteFile(filepath.Join(tmpDir, "upload-1"), []byte("debris"), 0o644); err != nil {
		t.Fatalf("write staging debris: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(tmpDir, "upload-2.d"), 0o755); err != nil {
		t.Fatalf("create staging debris directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "upload-2.d", "upload-3"), []byte("debris"), 0o644); err != nil {
		t.Fatalf("write nested staging debris: %v", err)
	}

	// Content the sweep must not touch: an original in the sharded
	// tree and a derived thumbnail.
	originals := filepath.Join(dataDir, "originals", "ab", "cd")
	if err := os.MkdirAll(originals, 0o755); err != nil {
		t.Fatalf("create originals shard: %v", err)
	}
	original := filepath.Join(originals, "deadbeef.jpg")
	if err := os.WriteFile(original, []byte("original"), 0o644); err != nil {
		t.Fatalf("write original: %v", err)
	}
	thumbs := filepath.Join(dataDir, "storage", "thumbs", "0192f4a1")
	if err := os.MkdirAll(thumbs, 0o755); err != nil {
		t.Fatalf("create thumbs directory: %v", err)
	}
	thumb := filepath.Join(thumbs, "tile.avif")
	if err := os.WriteFile(thumb, []byte("thumbnail"), 0o644); err != nil {
		t.Fatalf("write thumbnail: %v", err)
	}

	if err := SweepTmp(dataDir); err != nil {
		t.Fatalf("SweepTmp() error = %v", err)
	}

	requireEmptyDir(t, tmpDir)
	for _, kept := range []string{original, thumb} {
		if _, err := os.Stat(kept); err != nil {
			t.Errorf("stat %s: %v (the sweep must leave non-staging files alone)", kept, err)
		}
	}
}

// TestSweepTmpFailsWhenStagingIsMissing pins the fail-fast contract:
// EnsureLayout must have created the staging directory, so a sweep
// that cannot even list it reports the error instead of silently
// declaring the layout clean.
func TestSweepTmpFailsWhenStagingIsMissing(t *testing.T) {
	dataDir, tmpDir := setupLayout(t)
	if err := os.Remove(tmpDir); err != nil {
		t.Fatalf("remove staging directory: %v", err)
	}

	if err := SweepTmp(dataDir); err == nil {
		t.Error("SweepTmp() on a missing staging directory = nil, want an error")
	}
}
