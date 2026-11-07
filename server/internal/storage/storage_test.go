package storage

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnsureLayoutCreatesDirectoryTree(t *testing.T) {
	dataDir := t.TempDir()
	configDir := t.TempDir()

	if err := EnsureLayout(dataDir, configDir); err != nil {
		t.Fatalf("EnsureLayout() error = %v, want nil", err)
	}

	for _, dir := range []string{
		filepath.Join(dataDir, "originals"),
		filepath.Join(dataDir, "tmp"),
		filepath.Join(dataDir, "storage", "thumbs"),
		filepath.Join(dataDir, "storage", "encoded"),
		filepath.Join(dataDir, "storage", "sidecars"),
		filepath.Join(configDir, "models"),
	} {
		info, err := os.Stat(dir)
		if err != nil {
			t.Errorf("stat %s: %v", dir, err)
			continue
		}
		if !info.IsDir() {
			t.Errorf("%s is not a directory", dir)
		}
		// The tree is created with 0o755; the umask may clear read
		// bits, but group and world write bits must never be set.
		if perm := info.Mode().Perm(); perm&0o022 != 0 {
			t.Errorf("%s has permission %o, want no group or world write bits", dir, perm)
		}
	}
}

func TestEnsureLayoutProbeCleansUpAfterItself(t *testing.T) {
	dataDir := t.TempDir()
	configDir := t.TempDir()

	if err := EnsureLayout(dataDir, configDir); err != nil {
		t.Fatalf("EnsureLayout() error = %v, want nil", err)
	}

	// A successful rename round-trip must leave the probe file in
	// neither directory.
	for _, dir := range []string{
		filepath.Join(dataDir, "tmp"),
		filepath.Join(dataDir, "originals"),
	} {
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatalf("read %s: %v", dir, err)
		}
		if len(entries) != 0 {
			t.Errorf("%s holds %d entries after the layout probe, want 0", dir, len(entries))
		}
	}
}

func TestEnsureLayoutReportsRenameProbeFailure(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root: read-only directories do not block renames")
	}

	dataDir := t.TempDir()
	configDir := t.TempDir()
	if err := EnsureLayout(dataDir, configDir); err != nil {
		t.Fatalf("EnsureLayout() error = %v, want nil", err)
	}

	// Make originals unwritable: the probe file can be created in tmp
	// but not renamed across, which exercises the same error path as a
	// cross-device layout.
	originals := filepath.Join(dataDir, "originals")
	tmp := filepath.Join(dataDir, "tmp")
	if err := os.Chmod(originals, 0o555); err != nil {
		t.Fatalf("chmod %s read-only: %v", originals, err)
	}
	t.Cleanup(func() {
		_ = os.Chmod(originals, 0o755)
	})

	err := EnsureLayout(dataDir, configDir)
	if err == nil {
		t.Fatal("EnsureLayout() error = nil, want an error when the rename probe cannot move files into originals")
	}
	for _, dir := range []string{tmp, originals} {
		if !strings.Contains(err.Error(), dir) {
			t.Errorf("error %q does not name the directories involved: %s", err.Error(), dir)
		}
	}

	// A failed probe must not strand its file in upload staging.
	entries, err := os.ReadDir(tmp)
	if err != nil {
		t.Fatalf("read %s: %v", tmp, err)
	}
	if len(entries) != 0 {
		t.Errorf("tmp holds %d entries after the failed probe, want 0", len(entries))
	}
}

// resetDataRoot simulates a fresh process that has not run a
// successful EnsureLayout: the recorded data root is cleared for the
// duration of the test and restored afterwards.
func resetDataRoot(t *testing.T) {
	t.Helper()
	saved := dataRoot
	dataRoot = ""
	t.Cleanup(func() { dataRoot = saved })
}

func TestPathHelpersPanicBeforeEnsureLayout(t *testing.T) {
	resetDataRoot(t)

	calls := []struct {
		name string
		call func()
	}{
		{"ThumbPath", func() { ThumbPath("0197a1b2-c3d4-4e5f-8a9b-0c1d2e3f4a5b", "tile") }},
	}
	for _, c := range calls {
		t.Run(c.name, func(t *testing.T) {
			defer func() {
				panicValue := recover()
				if panicValue == nil {
					t.Fatalf("%s did not panic, want panic before EnsureLayout has succeeded", c.name)
				}
				msg, ok := panicValue.(string)
				if !ok || !strings.Contains(msg, "EnsureLayout") {
					t.Errorf("panic message = %v, want it to mention EnsureLayout", panicValue)
				}
			}()
			c.call()
		})
	}
}

func TestThumbPath(t *testing.T) {
	dataDir := t.TempDir()
	if err := EnsureLayout(dataDir, t.TempDir()); err != nil {
		t.Fatalf("EnsureLayout() error = %v, want nil", err)
	}

	assetID := "0197a1b2-c3d4-4e5f-8a9b-0c1d2e3f4a5b"
	got := ThumbPath(assetID, "tile")
	want := filepath.Join(dataDir, "storage", "thumbs", assetID, "tile.avif")
	if got != want {
		t.Errorf("ThumbPath() = %q, want %q", got, want)
	}
}
