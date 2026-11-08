// The albums-as-sources path domain (E4): the mount-root allowlist
// as a live domain, the source-path completion behind GET
// /albums/path-completions, and the path validation matrix the
// directory album creation runs before anything is written.
//
// One pin governs the whole file: the completion domain IS the
// validation domain. Both read the same effective roots -- the
// configured allowlist resolved to the directories that actually
// exist, are directories, and sit outside the server's managed data
// tree -- and the completion never lists a path the creation would
// refuse for being outside the allowlist. The family instance's
// trust boundary is drawn at the roots: anything outside them can
// never become an album, whatever the filesystem lets the server
// read.
package api

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	apigen "github.com/welpher/lumemor/server/internal/api/gen"
	"github.com/welpher/lumemor/server/internal/db"
)

const (
	// albumPathMaxBytes is the path bound of the face: the
	// filesystem's own ceiling, so a stored root can always be
	// handed to the operating system verbatim.
	albumPathMaxBytes = 4096

	// albumPathCompletionLimit caps the completion rows: a combobox
	// page, not a directory dump -- past the cap the user narrows
	// by typing.
	albumPathCompletionLimit = 20
)

// pathUnder reports whether path lies at or under root, component
// boundaries only: "/a/b" is under "/a" but "/a/b-extra" is not.
// The root "/" carries every absolute path (an administrator who
// allows the filesystem root allows the filesystem).
func pathUnder(path, root string) bool {
	if path == root {
		return true
	}
	if root == "/" {
		return strings.HasPrefix(path, "/")
	}
	return strings.HasPrefix(path, root+"/")
}

// pathUnderAny reports whether path lies at or under one of roots.
func pathUnderAny(path string, roots []string) bool {
	for _, root := range roots {
		if pathUnder(path, root) {
			return true
		}
	}
	return false
}

// resolveMountRoot maps one configured root onto its live form: the
// path cleaned and fully resolved (symlinks evaluated), proven to be
// an existing directory outside the server's managed tree. A root
// that fails any arm contributes nothing to the domain -- the
// not-yet-mounted volume, the file where a directory was expected,
// the root that resolves into the managed tree are all honestly
// inert until the configuration changes, and the completion can
// never suggest what the creation would refuse.
func resolveMountRoot(configured, managedDir string) (string, bool) {
	if configured == "" {
		return "", false
	}
	resolved, err := filepath.EvalSymlinks(filepath.Clean(configured))
	if err != nil {
		return "", false
	}
	info, err := os.Stat(resolved)
	if err != nil || !info.IsDir() {
		return "", false
	}
	if pathUnder(resolved, filepath.Clean(managedDir)) {
		return "", false
	}
	return resolved, true
}

// effectiveMountRoots computes the live domain: the stored allowlist
// resolved root by root (an unusable root is skipped, not fatal --
// the closed state stays closed, the working roots keep working),
// deduplicated, sorted ascending. A corrupt setting row reads as the
// empty list with the error logged server-side, the corrupt-row
// discipline of every instance setting.
func (s *server) effectiveMountRoots(ctx context.Context) []string {
	stored, err := db.AllowedMountRoots(ctx, s.queries)
	if err != nil {
		s.logger.Error("read the mount roots setting; answering the empty allowlist", "error", err)
	}
	seen := make(map[string]bool, len(stored))
	roots := make([]string, 0, len(stored))
	for _, configured := range stored {
		resolved, ok := resolveMountRoot(configured, s.dataDir)
		if !ok || seen[resolved] {
			continue
		}
		seen[resolved] = true
		roots = append(roots, resolved)
	}
	sort.Strings(roots)
	return roots
}

// validateAlbumRootPath runs the creation-time path matrix over one
// candidate source path. It answers the RESOLVED absolute path (the
// value the album row stores, so a later symlink swap can never
// repoint the album), the failing branch's contract error code (the
// empty code means the path passed), and an error for the unexpected
// I/O failures the caller answers with a 500. The branches, in
// order: existence, directory-ness, readability, the resolved
// allowlist domain (the symlink-escape arm distinguishes a typed
// path that sat inside a root but resolved out of it), the managed
// tree (refused whatever the allowlist says), and the overlap or
// nesting against every other directory album's root on the
// instance.
func validateAlbumRootPath(roots []string, managedDir, rootPath string, existingRoots []string) (string, apigen.ErrorCode, error) {
	// The shape arm: the handler already names the empty, relative
	// and over-long values; this guard keeps the matrix total for
	// any caller.
	if rootPath == "" || !filepath.IsAbs(rootPath) || len(rootPath) > albumPathMaxBytes {
		return "", apigen.VALIDATIONFAILED, nil
	}
	clean := filepath.Clean(rootPath)

	// Existence and directory-ness: Stat follows symlinks, so a
	// dangling link and a link to a file answer the honest branches
	// of the thing the typed path resolves to.
	info, err := os.Stat(clean)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return "", apigen.ALBUMPATHNOTFOUND, nil
		}
		if errors.Is(err, fs.ErrPermission) {
			// An unreadable ancestor keeps even the stat out: the
			// server can no more read this path than list it.
			return "", apigen.ALBUMPATHNOTREADABLE, nil
		}
		return "", "", fmt.Errorf("stat the source path %q: %w", clean, err)
	}
	if !info.IsDir() {
		return "", apigen.ALBUMPATHNOTDIRECTORY, nil
	}

	resolved, err := filepath.EvalSymlinks(clean)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			// The target vanished between the stat and the
			// resolution: the path does not exist.
			return "", apigen.ALBUMPATHNOTFOUND, nil
		}
		return "", "", fmt.Errorf("resolve the symlinks of %q: %w", clean, err)
	}

	// Readability: the rescan lists this directory, so the matrix
	// proves the listing works now -- an open plus one name read
	// (an empty directory answers EOF, which is readable).
	dir, err := os.Open(resolved)
	if err != nil {
		if errors.Is(err, fs.ErrPermission) {
			return "", apigen.ALBUMPATHNOTREADABLE, nil
		}
		if errors.Is(err, fs.ErrNotExist) {
			return "", apigen.ALBUMPATHNOTFOUND, nil
		}
		return "", "", fmt.Errorf("open the source path %q: %w", resolved, err)
	}
	_, readErr := dir.Readdirnames(1)
	closeErr := dir.Close()
	if readErr != nil && !errors.Is(readErr, io.EOF) {
		if errors.Is(readErr, fs.ErrPermission) {
			return "", apigen.ALBUMPATHNOTREADABLE, nil
		}
		return "", "", fmt.Errorf("read the source path %q: %w", resolved, readErr)
	}
	if closeErr != nil {
		return "", "", fmt.Errorf("close the source path %q: %w", resolved, closeErr)
	}

	// The allowlist domain, on the RESOLVED path: what the album
	// stores is what must sit at or under a live mount root. The
	// typed path sitting inside a root while the resolution leaves
	// them all is the symlink escape -- its own branch, distinct
	// from the plainly-outside path.
	if !pathUnderAny(resolved, roots) {
		if pathUnderAny(clean, roots) {
			return "", apigen.ALBUMPATHSYMLINKESCAPE, nil
		}
		return "", apigen.ALBUMPATHOUTSIDEROOTS, nil
	}

	// The managed tree: the server's own data root (originals/,
	// derivatives, trash) is never an album source, whatever the
	// allowlist says -- the branch stands between a misconfigured
	// root and the feedback loop of referencing the managed files.
	if pathUnder(resolved, filepath.Clean(managedDir)) {
		return "", apigen.ALBUMPATHINMANAGEDTREE, nil
	}

	// The overlap arm: no other directory album's root may be equal
	// to, inside, or containing this one. The check runs across the
	// whole instance (the pin has no owner scope): two albums
	// mirroring the same tree is confusing double-scan state
	// whatever accounts own them.
	for _, other := range existingRoots {
		if other == "" {
			continue
		}
		if pathUnder(resolved, other) || pathUnder(other, resolved) {
			return "", apigen.ALBUMPATHOVERLAPS, nil
		}
	}

	return resolved, "", nil
}

// albumPathErrorMessage maps each matrix branch onto the English
// debug message of its 400 envelope (the i18n key is the code's
// lowercase form; the message is for logs and debugging only).
func albumPathErrorMessage(code apigen.ErrorCode) string {
	switch code {
	case apigen.ALBUMPATHNOTFOUND:
		return "the source path does not exist"
	case apigen.ALBUMPATHNOTDIRECTORY:
		return "the source path is not a directory"
	case apigen.ALBUMPATHNOTREADABLE:
		return "the server cannot read the source path"
	case apigen.ALBUMPATHSYMLINKESCAPE:
		return "the source path resolves outside the allowed mount roots through a symlink"
	case apigen.ALBUMPATHINMANAGEDTREE:
		return "the source path sits inside the server's managed data tree"
	case apigen.ALBUMPATHOVERLAPS:
		return "the source path overlaps another directory album's root"
	case apigen.ALBUMPATHOUTSIDEROOTS:
		return "the source path is outside the allowed mount roots"
	default:
		return "the source path is invalid"
	}
}

// writeAlbumPathError emits the path matrix's 400 for one branch:
// the branch's own error code, never the generic validation
// envelope.
func (s *server) writeAlbumPathError(w http.ResponseWriter, code apigen.ErrorCode) {
	s.writeError(w, http.StatusBadRequest, code, albumPathErrorMessage(code), nil)
}

// completeAlbumRootPath lists the completion rows for one typed
// prefix over the live domain: the pickable directories of the
// creation dialog's combobox. The rows are always full absolute
// paths at or under a live root and never inside the server's
// managed tree -- the completion-domain pin: the endpoint never
// lists a path the creation would refuse for being outside the
// allowlist or inside the managed tree. The matching rule: the
// roots whose paths the typed prefix leads, plus the entries of
// the prefix's parent whose names start with the prefix's last
// segment, both deduplicated, sorted ascending, capped at the row
// limit. An empty prefix lists the roots; a trailing separator
// drills into the cleaned prefix itself and suppresses the root
// rows ("I am inside this directory"); a relative prefix, an
// unreadable parent and a domain miss all answer the empty list --
// never an error, and never a row outside the domain.
func completeAlbumRootPath(roots []string, managedDir, prefix string) []string {
	paths := make([]string, 0, len(roots))
	if prefix == "" {
		paths = append(paths, roots...)
		if len(paths) > albumPathCompletionLimit {
			paths = paths[:albumPathCompletionLimit]
		}
		return paths
	}
	if !filepath.IsAbs(prefix) {
		return paths
	}

	// The trailing separator carries meaning Clean would eat: it
	// says the last segment is complete, so the completion lists
	// the children of the cleaned prefix itself.
	trailing := strings.HasSuffix(prefix, "/") && prefix != "/"
	cleaned := filepath.Clean(prefix)
	parent, base := filepath.Dir(cleaned), filepath.Base(cleaned)
	if trailing {
		parent, base = cleaned, ""
	}

	candidates := make(map[string]bool, len(roots))
	if !trailing {
		// The root arm: every root the typed prefix leads into.
		// The effective roots are in the domain by construction
		// (resolution dropped the unusable and the managed ones),
		// so no filter applies here.
		for _, root := range roots {
			if strings.HasPrefix(root, cleaned) {
				candidates[root] = true
			}
		}
	}

	// The drill arm: the parent's real directories (symlinked
	// entries are never listed -- the rescan walks real trees and
	// the validation resolves symlinks, so a suggestion the picker
	// cannot trust is no suggestion) whose names match the last
	// segment, kept only when they sit at or under a live root and
	// outside the managed tree -- the confine, both axes of it (a
	// root configured over an ancestor of the managed tree must
	// not leak the server's own storage into the suggestions). A
	// parent that cannot be read simply contributes nothing: the
	// completion is a suggestion box, not a probe.
	if entries, err := os.ReadDir(parent); err == nil {
		for _, entry := range entries {
			if !entry.IsDir() || !strings.HasPrefix(entry.Name(), base) {
				continue
			}
			child := filepath.Join(parent, entry.Name())
			if pathUnderAny(child, roots) && !pathUnder(child, filepath.Clean(managedDir)) {
				candidates[child] = true
			}
		}
	}

	for path := range candidates {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	if len(paths) > albumPathCompletionLimit {
		paths = paths[:albumPathCompletionLimit]
	}
	return paths
}

// CompleteAlbumPath implements GET /api/v1/albums/path-completions:
// the source-path completion of the creation dialog. The face is
// user-scoped -- every signed-in member may create a directory
// album, so every signed-in member may complete a path; the admin
// role adds nothing. The answer is bounded work (one settings row
// read, at most one directory listing, at most 20 rows), and the
// empty list is an honest "nothing matches", never an error.
func (s *server) CompleteAlbumPath(w http.ResponseWriter, r *http.Request, params apigen.CompleteAlbumPathParams) {
	if _, ok := authFrom(r.Context()); !ok {
		s.rejectUnauthenticated(w)
		return
	}
	prefix := ""
	if params.Prefix != nil {
		prefix = *params.Prefix
		if len(prefix) > albumPathMaxBytes {
			s.writeValidationError(w, "prefix exceeds the limit of 4096 bytes", []string{"prefix"})
			return
		}
	}
	roots := s.effectiveMountRoots(r.Context())
	s.writeJSON(w, http.StatusOK, apigen.AlbumPathCompletionsResponse{
		Paths: completeAlbumRootPath(roots, s.dataDir, prefix),
	})
}
