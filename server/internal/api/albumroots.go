// The albums-as-sources path domain (reshaped by album
// autonomy): the source-path completion behind GET
// /albums/path-completions, the creation-time path validation
// matrix, and the creation-time write probe that discovers the
// source directory's writability once.
//
// Two rulings reshaped the file. First, the boundary: the
// mount-root allowlist retired WHOLE -- the compose file's mounts
// ARE the boundary (the operator controls what the container sees;
// the software layer neither can nor needs to restrict the operator
// further), so the completion's domain became the container's real
// visible filesystem and the matrix lost its allowlist-domain and
// root-overlap branches. The one confine that survives both sides
// is the managed tree: the server's own data root never lists and
// never becomes an album source. Second, the administration:
// mounting an existing photo directory is a configuration act, so
// the creation and the completion that browses for it are the
// administrator's alone -- the handlers carry the guard, this file
// carries the domain.
package api

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/google/uuid"

	apigen "github.com/welpher/lumemor/server/internal/api/gen"
	dbgen "github.com/welpher/lumemor/server/internal/db/gen"
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
func pathUnder(path, root string) bool {
	if path == root {
		return true
	}
	if root == "/" {
		return strings.HasPrefix(path, "/")
	}
	return strings.HasPrefix(path, root+"/")
}

// validateAlbumRootPath runs the creation-time path matrix over one
// candidate source path. It answers the RESOLVED absolute path (the
// value the album row stores, so a later symlink swap can never
// repoint an album), the failing branch's contract error code (the
// empty code means the path passed), and an error for the unexpected
// I/O failures the caller answers with a 500. The branches, in
// order: existence, directory-ness, readability, and the managed
// tree on the RESOLVED path. The allowlist-domain and root-overlap
// arms of the path matrix retired with the allowlist itself: the
// boundary is the compose mounts now, and with creation in the
// administrator's hands a second album over the same tree is an
// operator's visible choice, not an attack to refuse.
func validateAlbumRootPath(managedDir, rootPath string) (string, apigen.ErrorCode, error) {
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

	// The managed tree, on the RESOLVED path: the server's own data
	// root (originals, derivatives, trash) is never an album source
	// -- the branch that survived the allowlist retirement whole, a
	// typed path outside the tree that resolves into it answering
	// the same refusal as the tree itself.
	if pathUnder(resolved, filepath.Clean(managedDir)) {
		return "", apigen.ALBUMPATHINMANAGEDTREE, nil
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
	case apigen.ALBUMPATHINMANAGEDTREE:
		return "the source path sits inside the server's managed data tree"
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

// probeDirectoryWritable discovers whether the application can
// delete through the directory: it writes a uniquely named probe
// file, reads it back, and removes it. The three steps are
// the delete's own shape -- the trash move renames a file OUT of
// this directory, which needs exactly the write permission the
// probe's write and remove exercise -- so a directory that passes
// them all is a directory the delete flow can work with, and one
// that refuses any step is a directory whose photos cannot be
// deleted through the application. The probe never FAILS its
// caller: the answer is a fact (the `sourceWritable` column at
// creation; the completion row's `writable` at listing time), and a
// false answer is a legitimate album, not a refused creation.
//
// The one state that leaves a wart is write-yes-remove-no (a
// directory entry the process can create but not unlink): the
// probe file stays, because the same refusal that failed the probe
// would fail its cleanup. The wart is honest -- that directory is
// already in a state its own owner must fix -- and the probe's
// unique name keeps it identifiable.
func probeDirectoryWritable(resolvedDir string) bool {
	id, err := uuid.NewV7()
	if err != nil {
		// A UUIDv7 generation failure is a runtime outage with no
		// honest "writable" claim to make; the conservative fact
		// stands.
		return false
	}
	payload := []byte(id.String())
	probe := filepath.Join(resolvedDir, "probe-"+id.String())
	if err := os.WriteFile(probe, payload, 0o644); err != nil {
		return false
	}
	readBack, err := os.ReadFile(probe)
	if err != nil {
		// The write landed but the read could not see it: a
		// half-broken directory. The remove is best-effort -- the
		// write proved it might work.
		_ = os.Remove(probe)
		return false
	}
	if !bytes.Equal(readBack, payload) {
		// Different bytes back than were written: not this
		// process's file, or a filesystem playing games. Either
		// way the directory is not honestly usable for deletes.
		_ = os.Remove(probe)
		return false
	}
	if err := os.Remove(probe); err != nil {
		// Write-yes-remove-no: the wart state the comment above
		// owns. The fact is false -- removal is half of every
		// delete this application performs.
		return false
	}
	return true
}

// completeAlbumRootPath lists the completion rows for one typed
// prefix over the container's real visible filesystem: the pickable
// directories of the creation dialog's combobox. The domain change
// of the retirement -- the allowlist that used to confine the
// rows is gone, and the honest reflection of what the operating
// system lets the server read takes its place. The ONE exclusion
// that stays is the managed tree: no row ever names the server's
// own data root or anything under it (a prefix drilled into the
// managed tree answers its non-managed children only, and the root
// itself is never suggested when it would be the managed tree). The
// matching rule: an empty prefix lists the filesystem root itself;
// a prefix of exactly "/" lists the root's top-level directories;
// otherwise the entries of the prefix's parent whose names start
// with the prefix's last segment; a trailing separator switches to
// the children of the cleaned prefix itself and suppresses the
// standalone row ("I am inside this directory"). Only real
// directories are listed (symlinked entries are not), the rows are
// deduplicated, sorted ascending, capped at the row limit, and a
// relative prefix or an unreadable parent answers the empty list --
// never an error, never a managed-tree row.
func completeAlbumRootPath(managedDir, prefix string) []string {
	managed := filepath.Clean(managedDir)
	paths := make([]string, 0, albumPathCompletionLimit)
	if prefix == "" {
		// The dialog's first open: the filesystem root itself, the
		// one "start here" row. The degenerate managed-tree-is-root
		// posture (a data root of "/") would make the root itself
		// a managed-tree path; the exclusion stays total even
		// there, and the completion answers nothing.
		if !pathUnder("/", managed) {
			paths = append(paths, "/")
		}
		return paths
	}
	if !filepath.IsAbs(prefix) {
		return paths
	}

	// The trailing separator carries meaning Clean would eat: it
	// says the last segment is complete, so the completion lists
	// the children of the cleaned prefix itself. A prefix of "/"
	// joins this arm too -- the separator says "I am at the root,
	// show me what it holds".
	trailing := strings.HasSuffix(prefix, "/")
	cleaned := filepath.Clean(prefix)
	parent, base := filepath.Dir(cleaned), filepath.Base(cleaned)
	if trailing {
		parent, base = cleaned, ""
	}

	candidates := make(map[string]bool)

	// The drill arm: the parent's real directories (symlinked
	// entries are never listed -- the rescan walks real trees and
	// the validation resolves symlinks, so a suggestion the picker
	// cannot trust is no suggestion) whose names match the last
	// segment, kept only when they sit outside the managed tree --
	// the exclusion, on every row. A parent that cannot be read
	// simply contributes nothing: the completion is a suggestion
	// box, not a probe.
	if entries, err := os.ReadDir(parent); err == nil {
		for _, entry := range entries {
			if !entry.IsDir() || !strings.HasPrefix(entry.Name(), base) {
				continue
			}
			child := filepath.Join(parent, entry.Name())
			if !pathUnder(child, managed) {
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
// ADMIN-scoped -- mounting an existing photo directory is the
// administrator's act, and the completion that browses the
// container's filesystem on the dialog's behalf belongs to the same
// configuration act; a plain member answers the admin surfaces'
// own 403 before the prefix is read (the members' contribution
// path is the upload stream, and a member never needs to see the
// filesystem). The answer is bounded work (at most one directory
// listing, at most 20 rows, one three-step probe per row), and the
// empty list is an honest "nothing matches", never an error. Each
// row carries the write fact its probe discovered -- the dialog's
// deletability-switch render condition, learned before any album
// is created.
func (s *server) CompleteAlbumPath(w http.ResponseWriter, r *http.Request, params apigen.CompleteAlbumPathParams) {
	requester, ok := authFrom(r.Context())
	if !ok {
		s.rejectUnauthenticated(w)
		return
	}
	if requester.role != dbgen.UserRoleAdmin {
		s.writeError(w, http.StatusForbidden, apigen.AUTHFORBIDDEN,
			"administrator role required", nil)
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
	paths := completeAlbumRootPath(s.dataDir, prefix)
	completions := make([]apigen.AlbumPathCompletion, 0, len(paths))
	for _, path := range paths {
		completions = append(completions, apigen.AlbumPathCompletion{
			Path:     path,
			Writable: probeDirectoryWritable(path),
		})
	}
	s.writeJSON(w, http.StatusOK, apigen.AlbumPathCompletionsResponse{
		Completions: completions,
	})
}
