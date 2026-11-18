// Package maintenance carries the boot-time repairs of the storage tree:
// the work no request owns, run once against a quiet server.
//
// The one sweep so far reclaims ORPHAN ORIGINALS. An upload places its
// files before it commits its rows, and a COMMIT whose outcome is
// unknown -- the connection dropped, the proxy cut the reply, the
// database restarted inside the reply window -- must not delete the
// placed files: the commit may well be durable, and destroying the
// bytes the row points at is the one unrecoverable outcome. The price
// is a leak: files that landed for a transaction that truly never
// committed, referenced by nothing. A boot sweep over the originals
// tree recovers that price -- a file no blob row names, older than the
// safety window, goes -- while anything referenced keeps its bytes.
package maintenance

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"
)

// BlobPathLister answers every stored file path a blob asset row names,
// in the portable form the rows carry: a slash-separated path relative
// to the data root. Trashed and component rows count (their bytes are
// still theirs); external rows do not appear at all, because their
// file_path names a family folder outside the managed tree.
type BlobPathLister interface {
	ListBlobFilePaths(ctx context.Context) ([]string, error)
}

// SweepResult is one sweep run's counts: how many regular files the
// walk looked at and how many it removed. The caller logs them as the
// run's single line.
type SweepResult struct {
	Scanned int
	Removed int
}

// SweepOrphanOriginals walks <dataDir>/originals and removes every
// regular file that no blob row references and whose modification time
// is at least minAge old. The age window is the sweep's whole safety
// story: a placement in flight -- an upload between its link and its
// commit -- is a young file, so the boot run (where nothing can be in
// flight by construction) may pass a short window while a periodic
// run must pass one longer than any upload's placement-to-commit gap.
//
// A referenced file keeps its bytes whatever its age and however many
// rows are gone; only the difference between the tree and the rows is
// removable, which is exactly the leak the ambiguous-commit rule
// accepts. Empty directories stay: they are cheap, and removing them
// would need a second bottom-up pass for no storage gain.
//
// now is injected so tests move the age window without sleeping; a nil
// lister or an empty dataDir is a caller bug and fails loudly instead
// of walking a relative path.
func SweepOrphanOriginals(ctx context.Context, lister BlobPathLister, dataDir string, minAge time.Duration, now time.Time) (SweepResult, error) {
	if lister == nil {
		return SweepResult{}, errors.New("maintenance: orphan sweep: no blob path lister")
	}
	if dataDir == "" {
		return SweepResult{}, errors.New("maintenance: orphan sweep: empty data root")
	}

	paths, err := lister.ListBlobFilePaths(ctx)
	if err != nil {
		return SweepResult{}, fmt.Errorf("maintenance: orphan sweep: list the referenced originals: %w", err)
	}
	referenced := make(map[string]struct{}, len(paths))
	for _, path := range paths {
		referenced[path] = struct{}{}
	}

	root := filepath.Join(dataDir, "originals")
	result := SweepResult{}
	var errs []error
	walkErr := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.Type().IsRegular() {
			return nil
		}
		result.Scanned++

		rel, err := filepath.Rel(dataDir, path)
		if err != nil {
			return err
		}
		if _, known := referenced[filepath.ToSlash(rel)]; known {
			return nil
		}

		info, err := entry.Info()
		if err != nil {
			return err
		}
		if now.Sub(info.ModTime()) < minAge {
			// Young enough to be a placement whose commit is still
			// deciding: the next run judges it.
			return nil
		}
		if err := os.Remove(path); err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				// Another sweep removed it between the walk entry and
				// the delete: already the desired state.
				return nil
			}
			errs = append(errs, fmt.Errorf("remove orphan original %s: %w", path, err))
			return nil
		}
		result.Removed++
		return nil
	})
	if walkErr != nil {
		errs = append(errs, fmt.Errorf("walk the originals tree %s: %w", root, walkErr))
	}
	return result, errors.Join(errs...)
}
