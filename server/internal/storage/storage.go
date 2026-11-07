// Package storage owns the on-disk layout of the lumemor data and
// config directories: the per-owner originals tree whose file names
// the storage template renders at ingest, upload staging that shares
// their filesystem, derived media files, and model storage.
package storage

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
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
// It then verifies with an actual link round-trip that upload
// staging shares a filesystem with the originals directory: uploads
// finalize with a hard link into the originals tree, which only works
// within one filesystem, so a layout that would break every upload
// fails startup instead of the first upload attempt. On success it
// records the data root for the path helpers below.
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

// renameProbe links a freshly created file from tmp into originals and
// removes it again, proving that both directories share a filesystem
// (a hard link, the same primitive the upload placement uses -- it
// fails with EXDEV across filesystems).
func renameProbe(tmp, originals string) error {
	probe, err := os.CreateTemp(tmp, ".layout-probe-*")
	if err != nil {
		return fmt.Errorf("storage layout: create probe file in %s: %w", tmp, err)
	}
	name := probe.Name()
	if err := probe.Close(); err != nil {
		// Same hygiene as the link failure below: do not strand the
		// probe file in upload staging.
		_ = os.Remove(name)
		return fmt.Errorf("storage layout: close probe file %s: %w", name, err)
	}

	moved := filepath.Join(originals, filepath.Base(name))
	if err := os.Link(name, moved); err != nil {
		// Best-effort cleanup so a failed probe leaves no file behind
		// in upload staging.
		_ = os.Remove(name)
		return fmt.Errorf("storage layout: upload staging %s and originals %s must share a filesystem so uploads can finalize with a link: %w", tmp, originals, err)
	}
	if err := os.Remove(moved); err != nil {
		return fmt.Errorf("storage layout: remove probe file %s: %w", moved, err)
	}
	_ = os.Remove(name)
	return nil
}

// ThumbPath returns the path of the AVIF thumbnail of the given size
// for an asset under the data root:
// storage/thumbs/<assetID>/<size>.avif. It panics when EnsureLayout
// has not succeeded yet.
func ThumbPath(assetID, size string) string {
	requireDataRoot("ThumbPath")
	return filepath.Join(dataRoot, "storage", "thumbs", assetID, size+".avif")
}

// RemoveAssetFiles removes the on-disk artifacts of one permanently
// deleted asset under the shared-blob rule. The derived directories
// (storage/thumbs/<assetID>/ and storage/encoded/<assetID>/) are
// keyed by the asset id alone -- no other asset can hold files inside
// them -- so they are always removed. The original the row's
// file_path names is removed only when references is zero. Assets
// ingested before the storage-template feature live at
// content-addressed paths where identical bytes of ANY owner share
// one physical file (another owner's upload of the same photo, or a
// live row, keeps it); assets ingested since live at per-owner
// template paths that no other row can reference -- the references
// count answers both eras with the one rule. Both removals are
// best-effort and idempotent: a file already gone is success (the
// delete that owned it may have run concurrently), and a failure is
// returned for the caller to log -- the database row is already
// deleted, so a failure here is a storage leak, never data loss.
// Unlike the path helpers above it works on any data root, not
// the one recorded by EnsureLayout: the callers hold the data root
// they were wired with.
func RemoveAssetFiles(dataDir, relPath, assetID string, references int64) error {
	if assetID == "" {
		// The callers pass the uuid string of a database row; empty
		// would remove the whole thumbs tree, so refuse it outright
		// rather than trusting a faraway caller.
		return fmt.Errorf("storage: RemoveAssetFiles: asset id is empty")
	}
	var errs []error
	for _, dir := range []string{
		filepath.Join(dataDir, "storage", "thumbs", assetID),
		filepath.Join(dataDir, "storage", "encoded", assetID),
	} {
		if err := os.RemoveAll(dir); err != nil {
			errs = append(errs, fmt.Errorf("remove derived files %s: %w", dir, err))
		}
	}
	if references == 0 {
		// The originals-tree guard is defense in depth against a
		// corrupted row: file_path values are server-generated. A
		// row whose path escapes the tree is left on disk and
		// reported, not blindly followed. Directory albums (whose
		// files live outside the tree) arrive with the album domain
		// and carry their own removal path.
		abs, err := ResolveStoredPath(dataDir, relPath)
		if err != nil {
			errs = append(errs, err)
		} else if err := removeFileIfPresent(abs); err != nil {
			errs = append(errs, fmt.Errorf("remove original %s: %w", abs, err))
		}
	}
	return errors.Join(errs...)
}

// removeFileIfPresent deletes one file, treating "already gone" as
// success: the delete of a shared blob may race an equivalent delete
// whose file removal already ran.
func removeFileIfPresent(path string) error {
	if err := os.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	return nil
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

// EncodedPath returns the path of the transcoded playable copy of a
// video asset under the data root: storage/encoded/<assetID>/play.mp4.
// The file is the ffmpeg re-encode (H.264 + AAC, the web-safe pair)
// the transcode job derives from the immutable original -- a serving
// derivative, never the archival bytes themselves. Like the other
// path helpers it panics when EnsureLayout has not succeeded yet.
func EncodedPath(assetID string) string {
	requireDataRoot("EncodedPath")
	return filepath.Join(dataRoot, "storage", "encoded", assetID, "play.mp4")
}

// ResolveStoredPath resolves an asset's stored relative file path --
// the portable slash-separated form the upload receiver records in
// file_path, relative to the data root -- against the given data root
// and refuses anything that would leave the originals tree. file_path
// values are server-generated (template-rendered at ingest since the
// storage-template feature, content-addressed before it), so the
// guard is defense in depth against a corrupted
// row, not an input validation layer; the API's media endpoints and
// both pipeline jobs resolve their input through it. Unlike the path
// helpers above it works on any data root, not the one recorded by
// EnsureLayout: the callers hold the data root they were wired with.
func ResolveStoredPath(dataDir, rel string) (string, error) {
	originals := filepath.Join(dataDir, "originals") + string(filepath.Separator)
	abs := filepath.Join(dataDir, filepath.FromSlash(rel))
	if !strings.HasPrefix(abs, originals) {
		return "", fmt.Errorf("storage: file path %q escapes the originals tree", rel)
	}
	return abs, nil
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
