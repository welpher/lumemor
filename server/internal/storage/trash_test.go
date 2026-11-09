package storage

import (
	"os"
	"path/filepath"
	"testing"
)

// writeTrashFixture writes one file and returns its path, failing
// the test on any error.
func writeTrashFixture(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write fixture %s: %v", path, err)
	}
	return path
}

// requireGone fails the test when path still exists.
func requireGone(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Lstat(path); err == nil {
		t.Fatalf("%s still exists, want it gone", path)
	}
}

// TestMoveToTrashAndBack pins the external lifecycle's move
// primitives: the trash move lands the file under
// trash/<assetID>/ keyed by the asset's own id, the move back
// returns it to the recorded original location, and each half
// leaves the file at exactly one of the two paths.
func TestMoveToTrashAndBack(t *testing.T) {
	dataDir := t.TempDir()
	if err := EnsureLayout(dataDir, t.TempDir()); err != nil {
		t.Fatalf("EnsureLayout: %v", err)
	}
	source := writeTrashFixture(t, t.TempDir(), "IMG_0001.jpg", "bytes")

	trashPath, err := MoveToTrash(dataDir, source, "0192f8a1-0b1c-2d3e-4f5a-6b7c8d9e0f10")
	if err != nil {
		t.Fatalf("MoveToTrash: %v", err)
	}
	if want := filepath.Join(TrashHome(dataDir, "0192f8a1-0b1c-2d3e-4f5a-6b7c8d9e0f10"), "IMG_0001.jpg"); trashPath != want {
		t.Errorf("trash path = %q, want %q", trashPath, want)
	}
	requireGone(t, source)
	if moved, err := os.ReadFile(trashPath); err != nil || string(moved) != "bytes" {
		t.Errorf("the trashed file = (%q, %v), want the moved bytes", moved, err)
	}

	if err := MoveFileBack(trashPath, source); err != nil {
		t.Fatalf("MoveFileBack: %v", err)
	}
	requireGone(t, trashPath)
	if moved, err := os.ReadFile(source); err != nil || string(moved) != "bytes" {
		t.Errorf("the restored file = (%q, %v), want the moved bytes", moved, err)
	}
}

// TestMoveToTrashCrossDeviceFallback pins the copy-then-remove
// fallback: a source on a filesystem rename cannot reach (here a
// directory whose write permission the rename needs) still moves
// when the fallback's own steps can run. The rename refuses the
// unwritable parent, the copy succeeds (the file stays readable),
// and the remove -- which needs the same permission the rename did
// -- fails, cleans the partial target, and answers the error with
// the source untouched: the move's failure story is "nothing
// half-moved", which is what the batch's explicit failure and its
// compensations build on. (The full success of the fallback needs
// two filesystems, which the host cannot provide portably; the
// no-permission arm here exercises the same code path's failure
// half, and the restore test covers the copy direction.)
func TestMoveToTrashCrossDeviceFallback(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("the permission arms are untestable as root: the kernel lets root write anything")
	}
	dataDir := t.TempDir()
	if err := EnsureLayout(dataDir, t.TempDir()); err != nil {
		t.Fatalf("EnsureLayout: %v", err)
	}
	// The source's parent is writable for the setup, then read-only
	// for the move: rename out of it and remove from it both refuse.
	parent := t.TempDir()
	source := writeTrashFixture(t, parent, "IMG_0002.jpg", "bytes")
	if err := os.Chmod(parent, 0o500); err != nil {
		t.Fatalf("chmod the source parent: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(parent, 0o700) })

	_, err := MoveToTrash(dataDir, source, "0192f8a1-0b1c-2d3e-4f5a-6b7c8d9e0f11")
	if err == nil {
		t.Fatal("MoveToTrash error = nil, want the refused move to fail")
	}
	// Nothing half-moved: the source keeps the file, and the trash
	// home the call created is gone with the failed move (the move's
	// own failure paths clean their partial targets, and the home
	// removal leaves the state exactly as it was found).
	if moved, err := os.ReadFile(source); err != nil || string(moved) != "bytes" {
		t.Errorf("the source file = (%q, %v), want it untouched", moved, err)
	}
	requireGone(t, TrashHome(dataDir, "0192f8a1-0b1c-2d3e-4f5a-6b7c8d9e0f11"))
}

// TestMoveFileCrossDeviceSuccess pins the copy-then-remove fallback's
// SUCCESS arm on a real cross-filesystem pair (the EXDEV audit of
// E6's album autonomy: the trash move of a directory album's source
// file faces the mount-versus-data-root split on every delete). The
// probe file sits on /dev/shm -- tmpfs on the Linux dev container,
// a directory the host may not have (macOS has no portable second
// filesystem; those hosts skip, and `make docker-test` runs the arm
// for real) -- and the target in the test's own tree. A rename probe
// first proves the pair really is cross-device: a same-device pair
// would exercise the rename arm instead of the fallback, so it skips
// honestly. The move itself then answers success with the bytes
// intact at the target and nothing left at the source -- the exact
// two-step every cross-device trash move runs.
func TestMoveFileCrossDeviceSuccess(t *testing.T) {
	if _, err := os.Stat("/dev/shm"); err != nil {
		t.Skipf("/dev/shm is unavailable on this host (%v); the cross-device arm needs a second filesystem -- run the suite in the dev container (make docker-test)", err)
	}
	source := filepath.Join("/dev/shm", "lumemor-exdev-probe")
	if err := os.WriteFile(source, []byte("cross-device bytes"), 0o644); err != nil {
		t.Skipf("write the probe into /dev/shm: %v (the cross-device arm needs a writable tmpfs)", err)
	}
	t.Cleanup(func() { _ = os.Remove(source) })

	target := filepath.Join(t.TempDir(), "IMG_0006.jpg")
	if err := os.Rename(source, target); err == nil {
		// The pair turned out same-device (a container that mounts
		// /dev/shm over the same filesystem as the temp tree): the
		// rename arm would run, not the fallback. Put the file back
		// and say so instead of pinning the wrong arm.
		if err := os.Rename(target, source); err != nil {
			t.Fatalf("restore the probe file after the same-device rename: %v", err)
		}
		t.Skip("the pair is same-device; the fallback arm needs two filesystems")
	}

	if err := moveFile(source, target); err != nil {
		t.Fatalf("moveFile across devices: %v", err)
	}
	requireGone(t, source)
	if moved, err := os.ReadFile(target); err != nil || string(moved) != "cross-device bytes" {
		t.Errorf("the moved bytes = (%q, %v), want the copied content", moved, err)
	}
}

// TestMoveFileBackRefusesMissingDestination pins the restore's
// honest refusal: an original location that no longer exists fails
// the move instead of fabricating the path (a fabricated directory
// at an unmounted volume's mount point would strand the file where
// the family would never look).
func TestMoveFileBackRefusesMissingDestination(t *testing.T) {
	dataDir := t.TempDir()
	if err := EnsureLayout(dataDir, t.TempDir()); err != nil {
		t.Fatalf("EnsureLayout: %v", err)
	}
	source := writeTrashFixture(t, t.TempDir(), "IMG_0003.jpg", "bytes")
	trashPath, err := MoveToTrash(dataDir, source, "0192f8a1-0b1c-2d3e-4f5a-6b7c8d9e0f12")
	if err != nil {
		t.Fatalf("MoveToTrash: %v", err)
	}

	// The original location's directory is gone entirely.
	if err := MoveFileBack(trashPath, filepath.Join(t.TempDir(), "gone", "IMG_0003.jpg")); err == nil {
		t.Fatal("MoveFileBack error = nil, want the missing destination to fail")
	}
	// The trash keeps the file: the row stays trashed and a retry
	// after the source returns finds it exactly here.
	if moved, err := os.ReadFile(trashPath); err != nil || string(moved) != "bytes" {
		t.Errorf("the trashed file = (%q, %v), want it kept for the retry", moved, err)
	}
}

// TestResolveAssetPath pins the kind-aware resolver's two eras: a
// blob row's relative path resolves inside the originals tree (and
// a path escaping it refuses), an external row's absolute path
// resolves to its cleaned self, and an external path that is not
// absolute or that points inside the managed tree refuses -- the
// invariants the rescan's ingest guarantees and the resolver keeps
// defended.
func TestResolveAssetPath(t *testing.T) {
	dataDir := t.TempDir()

	if got, err := ResolveAssetPath(dataDir, "originals/owner/a.jpg", StorageKindBlob); err != nil {
		t.Errorf("blob resolution of a stored path: %v", err)
	} else if want := filepath.Join(dataDir, "originals", "owner", "a.jpg"); got != want {
		t.Errorf("blob path = %q, want %q", got, want)
	}
	if _, err := ResolveAssetPath(dataDir, "storage/thumbs/x", StorageKindBlob); err == nil {
		t.Error("blob resolution of a path escaping the originals tree = nil error, want the refusal")
	}

	external := filepath.Join(t.TempDir(), "family", "IMG_0001.jpg")
	if got, err := ResolveAssetPath(dataDir, external+"/.", StorageKindExternal); err != nil {
		t.Errorf("external resolution: %v", err)
	} else if got != external {
		t.Errorf("external path = %q, want the cleaned %q", got, external)
	}
	if _, err := ResolveAssetPath(dataDir, "family/IMG_0001.jpg", StorageKindExternal); err == nil {
		t.Error("external resolution of a relative path = nil error, want the refusal")
	}
	if _, err := ResolveAssetPath(dataDir, filepath.Join(dataDir, "trash", "x.jpg"), StorageKindExternal); err == nil {
		t.Error("external resolution of a path inside the managed tree = nil error, want the refusal")
	}
}

// TestRemoveExternalAssetFiles pins the removal rule: the derived
// trees always go, the trash copy goes with its directory, a
// missing trash path (the orphan) removes no original, and a trash
// path outside the trash area refuses rather than removing a file
// that was never the server's.
func TestRemoveExternalAssetFiles(t *testing.T) {
	dataDir := t.TempDir()
	if err := EnsureLayout(dataDir, t.TempDir()); err != nil {
		t.Fatalf("EnsureLayout: %v", err)
	}
	assetID := "0192f8a1-0b1c-2d3e-4f5a-6b7c8d9e0f13"
	thumbs := filepath.Join(dataDir, "storage", "thumbs", assetID)
	encoded := filepath.Join(dataDir, "storage", "encoded", assetID)
	for _, dir := range []string{thumbs, encoded} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("create %s: %v", dir, err)
		}
		writeTrashFixture(t, dir, "tile.avif", "x")
	}

	// The trash copy: moved in through the primitive, removed with
	// its directory.
	source := writeTrashFixture(t, t.TempDir(), "IMG_0004.jpg", "bytes")
	trashPath, err := MoveToTrash(dataDir, source, assetID)
	if err != nil {
		t.Fatalf("MoveToTrash: %v", err)
	}
	if err := RemoveExternalAssetFiles(dataDir, trashPath, assetID); err != nil {
		t.Errorf("RemoveExternalAssetFiles of the trash copy: %v", err)
	}
	requireGone(t, trashPath)
	requireGone(t, TrashHome(dataDir, assetID))
	requireGone(t, thumbs)
	requireGone(t, encoded)

	// The orphan (no trash path): the derived trees alone, nothing
	// else -- a source file that reappeared at the row's old
	// file_path must survive the purge (the next scan ingests it as
	// a NEW row, the pin).
	returned := writeTrashFixture(t, t.TempDir(), "IMG_0004.jpg", "bytes")
	if err := RemoveExternalAssetFiles(dataDir, "", assetID); err != nil {
		t.Errorf("RemoveExternalAssetFiles of the orphan: %v", err)
	}
	if _, err := os.Stat(returned); err != nil {
		t.Errorf("the source file after the orphan purge: %v, want it untouched", err)
	}

	// A trash path outside the trash area: the refusal, never a
	// removal of the family's file.
	foreign := writeTrashFixture(t, t.TempDir(), "IMG_0005.jpg", "bytes")
	if err := RemoveExternalAssetFiles(dataDir, foreign, assetID); err == nil {
		t.Error("RemoveExternalAssetFiles of a path outside the trash area = nil error, want the refusal")
	}
	if _, err := os.Stat(foreign); err != nil {
		t.Errorf("the foreign file after the refused removal: %v, want it untouched", err)
	}
}
