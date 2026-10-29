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
// it and panic while it is unset.
var dataRoot string

// requireDataRoot panics when no successful EnsureLayout call has
// recorded the data root: the path helpers would otherwise build
// paths relative to the process working directory, where file
// operations can silently succeed.
func requireDataRoot(caller string) {
	if dataRoot == "" {
		panic(fmt.Sprintf("storage: EnsureLayout must succeed before %s is called", caller))
	}
}

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

// SweepTmp empties the upload staging directory <dataDir>/tmp at boot,
// right after EnsureLayout: a single-process boot has no in-flight
// uploads, so anything in staging is crash debris from the previous
// run and is removed unconditionally -- files and subdirectories
// alike. The tmp directory itself survives (uploads stage into it
// immediately afterwards), and nothing outside it is touched.
// A staging directory that cannot even be listed is an error: startup
// fails rather than declaring a layout clean that it never looked at.
func SweepTmp(dataDir string) error {
	tmp := filepath.Join(dataDir, "tmp")
	entries, err := os.ReadDir(tmp)
	if err != nil {
		return fmt.Errorf("storage sweep: list upload staging %s: %w", tmp, err)
	}
	for _, entry := range entries {
		path := filepath.Join(tmp, entry.Name())
		if err := os.RemoveAll(path); err != nil {
			return fmt.Errorf("storage sweep: remove staged %s: %w", path, err)
		}
	}
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
		// Same hygiene as the rename failure below: do not strand the
		// probe file in upload staging.
		_ = os.Remove(name)
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
// checksum. It panics when EnsureLayout has not succeeded yet, or when
// arguments are malformed: checksums arrive as lowercase hex digests
// computed server side, extensions come from the upload format
// whitelist, and the data root comes from startup, so all of these are
// programming errors.
func OriginalPath(checksum, ext string) string {
	requireDataRoot("OriginalPath")
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
// storage/thumbs/<assetID>/<size>.avif. It panics when EnsureLayout
// has not succeeded yet.
func ThumbPath(assetID, size string) string {
	requireDataRoot("ThumbPath")
	return filepath.Join(dataRoot, "storage", "thumbs", assetID, size+".avif")
}

// PosterPath returns the path of the poster frame of a video asset
// under the data root: storage/thumbs/<assetID>/poster.jpg. The poster
// is a JPEG, not an AVIF thumbnail: it is extracted by ffmpeg from the
// video's first frame, and Task 6 serves it wherever a video asset's
// thumbnail is requested. It panics when EnsureLayout has not
// succeeded yet.
func PosterPath(assetID string) string {
	requireDataRoot("PosterPath")
	return filepath.Join(dataRoot, "storage", "thumbs", assetID, "poster.jpg")
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
