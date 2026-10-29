package media

import (
	"fmt"
	"os"
	"path/filepath"
)

// The derived files (thumbnails, poster frames) are served while the
// thumbnail job may re-run: the derivation is idempotent, but a
// truncate-in-place write would let a concurrent reader observe a
// half-written file. Both writers therefore land their bytes in a
// uniquely named temporary file BESIDE the destination -- same
// directory, therefore same filesystem -- and rename it into place:
// a reader sees either the previous complete file or the new one,
// never a partial one. The rename is the atomicity the upload
// receiver's staging commit already relies on.

// writeFileAtomic writes data to path through a temporary file in the
// same directory and an atomic rename into place. A failed write
// removes the temporary file, so a failed derivation leaves no
// residue beside the previous file.
func writeFileAtomic(path string, data []byte) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), ".tmp-*")
	if err != nil {
		return fmt.Errorf("create temporary file beside %s: %w", path, err)
	}
	name := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		discardTemp(tmp, name)
		return fmt.Errorf("write temporary file %s: %w", name, err)
	}
	if err := tmp.Close(); err != nil {
		discardTemp(nil, name)
		return fmt.Errorf("close temporary file %s: %w", name, err)
	}
	// CreateTemp mode is 0600; the derived files keep the 0644 the
	// serving process and the layout's other files use.
	if err := os.Chmod(name, 0o644); err != nil {
		discardTemp(nil, name)
		return fmt.Errorf("permission temporary file %s: %w", name, err)
	}
	if err := os.Rename(name, path); err != nil {
		discardTemp(nil, name)
		return fmt.Errorf("rename %s to %s: %w", name, path, err)
	}
	return nil
}

// stageTempDir reserves a uniquely named scratch directory inside
// parent for a derived-media writer that produces its destination
// through an external process (the poster's ffmpeg): the scratch file
// lives in a directory that did not exist a moment ago, so the
// external writer's target cannot pre-exist -- ffmpeg, invoked
// without -y by its fixed argument template, refuses to overwrite an
// existing output file. The scratch directory sits in parent's own
// tree (same filesystem), so renaming the finished file onto its
// destination stays atomic; the caller removes the directory --
// emptied by the rename -- through the returned cleanup.
func stageTempDir(parent string) (scratch string, cleanup func(), err error) {
	scratch, err = os.MkdirTemp(parent, ".tmp-*")
	if err != nil {
		return "", nil, fmt.Errorf("create temporary directory in %s: %w", parent, err)
	}
	return scratch, func() { _ = os.RemoveAll(scratch) }, nil
}

// discardTemp closes (when the file is still open) and removes a
// temporary file of a failed write. Best-effort: the caller already
// holds the real error.
func discardTemp(file *os.File, name string) {
	if file != nil {
		_ = file.Close()
	}
	_ = os.Remove(name)
}
