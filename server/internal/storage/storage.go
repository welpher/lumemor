// Package storage owns the on-disk layout of the lumemor data and
// config directories: content-addressed originals, upload staging that
// shares their filesystem, derived media files, and model storage.
package storage

import (
	"fmt"
	"os"
	"path/filepath"
)

// dataRoot is the photo data root recorded by the last successful
// EnsureLayout call; the path helpers below derive their results from
// it.
var dataRoot string

// EnsureLayout creates the full directory tree under the configured
// data and config roots:
//
//	data/    originals/, tmp/, storage/thumbs/, storage/encoded/, storage/sidecars/
//	config/  models/
//
// It then verifies with an actual rename round-trip that upload
// staging shares a filesystem with the originals directory: uploads
// finalize with an atomic rename, which only works within one
// filesystem, so a layout that would break every upload fails startup
// instead of the first upload attempt. On success it records the data
// root for the OriginalPath and ThumbPath helpers.
func EnsureLayout(dataDir, configDir string) error {
	originals := filepath.Join(dataDir, "originals")
	tmp := filepath.Join(dataDir, "tmp")
	for _, dir := range []string{
		originals,
		tmp,
		filepath.Join(dataDir, "storage", "thumbs"),
		filepath.Join(dataDir, "storage", "encoded"),
		filepath.Join(dataDir, "storage", "sidecars"),
		filepath.Join(configDir, "models"),
	} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("storage layout: create %s: %w", dir, err)
		}
	}
	if err := renameProbe(tmp, originals); err != nil {
		return err
	}
	dataRoot = dataDir
	return nil
}

// renameProbe moves a freshly created file from tmp into originals and
// removes it again, proving that both directories share a filesystem.
func renameProbe(tmp, originals string) error {
	probe, err := os.CreateTemp(tmp, ".layout-probe-*")
	if err != nil {
		return fmt.Errorf("storage layout: create probe file in %s: %w", tmp, err)
	}
	name := probe.Name()
	if err := probe.Close(); err != nil {
		return fmt.Errorf("storage layout: close probe file %s: %w", name, err)
	}

	moved := filepath.Join(originals, filepath.Base(name))
	if err := os.Rename(name, moved); err != nil {
		// Best-effort cleanup so a failed probe leaves no file behind
		// in upload staging.
		_ = os.Remove(name)
		return fmt.Errorf("storage layout: upload staging %s and originals %s must share a filesystem so uploads can finalize with an atomic rename: %w", tmp, originals, err)
	}
	if err := os.Remove(moved); err != nil {
		return fmt.Errorf("storage layout: remove probe file %s: %w", moved, err)
	}
	return nil
}

// OriginalPath returns the content-addressed path of an original file
// under the data root: originals/aa/bb/<checksum>.<ext>, where aa and
// bb are the first two and the following two hex characters of the
// checksum. Checksums arrive as lowercase hex digests computed server
// side and extensions come from the upload format whitelist, so
// malformed arguments are programming errors and panic.
func OriginalPath(checksum, ext string) string {
	if !isHex(checksum) {
		panic(fmt.Sprintf("storage: OriginalPath: checksum %q must be at least four lowercase hex characters", checksum))
	}
	if !isExtension(ext) {
		panic(fmt.Sprintf("storage: OriginalPath: extension %q must be one to five alphanumeric characters without a dot", ext))
	}
	return filepath.Join(dataRoot, "originals", checksum[:2], checksum[2:4], checksum+"."+ext)
}

// ThumbPath returns the path of the AVIF thumbnail of the given size
// for an asset under the data root:
// storage/thumbs/<assetID>/<size>.avif.
func ThumbPath(assetID, size string) string {
	return filepath.Join(dataRoot, "storage", "thumbs", assetID, size+".avif")
}

// isHex reports whether s is a checksum in canonical form: at least
// four lowercase hexadecimal characters, as produced by hex encoding a
// digest. Uppercase digests are rejected so every content hash maps to
// exactly one path.
func isHex(s string) bool {
	if len(s) < 4 {
		return false
	}
	for _, c := range s {
		switch {
		case c >= '0' && c <= '9', c >= 'a' && c <= 'f':
		default:
			return false
		}
	}
	return true
}

// isExtension reports whether s is a file extension in canonical form:
// one to five ASCII letters or digits, without a dot.
func isExtension(s string) bool {
	if len(s) < 1 || len(s) > 5 {
		return false
	}
	for _, c := range s {
		switch {
		case c >= '0' && c <= '9', c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z':
		default:
			return false
		}
	}
	return true
}
