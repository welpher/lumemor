// The folder live sync's filesystem watcher: an accelerator, not a
// second ingest path and not a second decider.
//
// The periodic diff (foldersync.go) is the truth on every platform.
// This file adds one thing on top: where the platform carries
// filesystem events, a change in a directory album's source root
// marks that album dirty, and a debounced wake asks the sweep to look
// at those albums at once -- seconds instead of one cadence window.
// The sweep still compares stamps; a write that leaves the file's
// identity intact (the same size, the same mtime) decides nothing,
// and a temp file that appears and vanishes decides nothing either.
// No event ever creates an asset, enqueues a scan, or writes a row:
// the dirty marker is a hint about WHERE to look, never a conclusion
// about WHAT changed.
//
// What is watched, and what is not:
//
//   - Every directory album's source root, recursively. New
//     subdirectories are watched as their creation events arrive; a
//     subdirectory that disappears loses its watches.
//   - Never the managed originals tree. Album roots are resolved and
//     validated at creation against the data root (the API's path
//     matrix refuses anything under it), and the watcher only ever
//     registers roots the album table names -- so the server's own
//     storage is out of scope by construction, not by a check here.
//
// The watch set's life: it is established when the queue starts, and
// reconciled by the sweep -- the one place that already lists every
// directory album and its root. Album creation and removal therefore
// need no second trigger: creating an album means a human was active,
// which means the next cadence tick sweeps and reconciles. A root
// that disappears (an unmount, a pulled disk) just loses its watches:
// the album is left to the diff's unwalkable-root skip, which decides
// nothing and reports nothing to the wall, and a root that comes back
// is re-watched by the next reconciliation.
//
// Graceful degradation is the design, not a fallback: a platform whose
// mounts carry no events (the Docker Desktop bind mounts, network
// shares) simply never wakes the accelerator, and the gated diff is
// the coverage. Watch exhaustion (inotify's max_user_watches, ENOSPC)
// or any other fsnotify failure is logged once and the watcher stops
// marking; the diff keeps converging. Nothing here probes the source
// folders by writing -- the watcher is strictly read-only on the
// family's folders.
package jobs

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/google/uuid"

	dbgen "github.com/welpher/lumemor/server/internal/db/gen"
)

// folderWatchDebounce is the accelerator's quiet window: a change
// marks its album dirty immediately, but the wake waits for the
// folder to be quiet this long, so a copy of two hundred photos is
// one look at the album instead of two hundred. One to two seconds is
// the pin: long enough to swallow a batch, short enough that an
// active house reads the result as instant.
const folderWatchDebounce = 1500 * time.Millisecond

// folderWatchBackend is the slice of fsnotify the watcher uses: the
// tests substitute a fake to drive events deterministically and to
// prove the exhaustion path without filling a real inotify table.
type folderWatchBackend interface {
	Add(name string) error
	Remove(name string) error
	Events() <-chan fsnotify.Event
	Errors() <-chan error
	Close() error
}

// fsnotifyBackend adapts the real watcher to the seam: fsnotify
// exposes its channels as fields, the seam as accessors.
type fsnotifyBackend struct {
	watcher *fsnotify.Watcher
}

func (b fsnotifyBackend) Add(name string) error         { return b.watcher.Add(name) }
func (b fsnotifyBackend) Remove(name string) error      { return b.watcher.Remove(name) }
func (b fsnotifyBackend) Events() <-chan fsnotify.Event { return b.watcher.Events }
func (b fsnotifyBackend) Errors() <-chan error          { return b.watcher.Errors }
func (b fsnotifyBackend) Close() error                  { return b.watcher.Close() }

// folderWatcher is the accelerator: the watch set, the dirty set, the
// debounce timer, and the one goroutine that turns events into dirty
// markers. It is safe for concurrent use -- the sweep reads and syncs
// it from the worker goroutine while the timer and the event loop
// write it.
type folderWatcher struct {
	backend  folderWatchBackend
	debounce time.Duration
	logger   *slog.Logger
	// wake asks for a sweep now. It runs on the debounce timer's
	// goroutine and must not block: production inserts a sweep job.
	wake func()

	mu sync.Mutex
	// roots maps each directory album to the source root its row
	// names. It is the event router: a path is "inside" every album
	// whose root prefixes it (album roots may nest), and each of them
	// is marked.
	roots map[uuid.UUID]string
	// watched is every directory with a live watch: the bookkeeping
	// that keeps a re-sync from re-walking a tree it already watches.
	watched map[string]struct{}
	// dirty is the handoff to the sweep: the albums a settled batch
	// touched. SwapDirty drains it.
	dirty map[uuid.UUID]struct{}
	// settle is the shared debounce timer: one quiet window across
	// every album, so a batch spanning albums wakes the sweep once.
	settle *time.Timer
	// degraded is set once a backend failure makes events
	// untrustworthy; from then on the watcher marks nothing and the
	// periodic diff is the whole coverage.
	degraded bool
	closed   bool

	done chan struct{}
	wg   sync.WaitGroup
}

// newFolderWatcher builds the watcher over a backend and starts its
// event loop. debounce is configurable so the tests need not wait the
// production window; wake may be nil (a watcher used only for its
// dirty set, as the tests' does).
func newFolderWatcher(backend folderWatchBackend, debounce time.Duration, logger *slog.Logger, wake func()) *folderWatcher {
	w := &folderWatcher{
		backend:  backend,
		debounce: debounce,
		logger:   logger,
		wake:     wake,
		roots:    make(map[uuid.UUID]string),
		watched:  make(map[string]struct{}),
		dirty:    make(map[uuid.UUID]struct{}),
		done:     make(chan struct{}),
	}
	w.wg.Add(1)
	go w.run()
	return w
}

// newFSNotifyWatcher builds the production watcher over a fresh
// fsnotify instance. The only failure is the platform's own watcher
// creation (no inotify descriptor available); the caller degrades to
// the diff alone.
func newFSNotifyWatcher(debounce time.Duration, logger *slog.Logger, wake func()) (*folderWatcher, error) {
	raw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("create the filesystem watcher: %w", err)
	}
	return newFolderWatcher(fsnotifyBackend{watcher: raw}, debounce, logger, wake), nil
}

// Sync reconciles the watch set with the directory albums the database
// lists: removed albums' watches go, new and moved roots are watched
// (recursively, once), and a directory whose watch survived is left
// alone -- its subtree's new directories arrive as events, so a
// re-walk every cadence tick would reintroduce the idle listing cost
// the gate exists to remove.
func (w *folderWatcher) Sync(ctx context.Context, queries *dbgen.Queries) error {
	albums, err := queries.ListDirectoryAlbums(ctx)
	if err != nil {
		return fmt.Errorf("list the directory albums: %w", err)
	}
	roots := make(map[uuid.UUID]string, len(albums))
	for _, album := range albums {
		roots[uuid.UUID(album.ID.Bytes)] = filepath.Clean(album.RootPath.String)
	}
	w.syncRoots(roots)
	return nil
}

// syncRoots applies one album listing to the watch set. Sync is its
// query side; the tests drive this directly.
func (w *folderWatcher) syncRoots(roots map[uuid.UUID]string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.closed || w.degraded {
		return
	}

	for albumID := range w.roots {
		if _, ok := roots[albumID]; !ok {
			// The un-ingest: a removed album has no folder to follow.
			delete(w.roots, albumID)
		}
	}
	for albumID, root := range roots {
		w.roots[albumID] = root
	}

	// A watch no live root covers is dropped: the removed album's
	// tree must not keep a descriptor, and a vanished root must not
	// be re-walked. Two albums over the same (or nested) trees share
	// watches, so the check is coverage, not identity.
	for path := range w.watched {
		if w.coveredLocked(path) {
			continue
		}
		delete(w.watched, path)
		if err := w.backend.Remove(path); err != nil && !errors.Is(err, fsnotify.ErrNonExistentWatch) {
			w.logger.Debug("the folder watcher could not drop a watch", "path", path, "error", err)
		}
	}

	for _, root := range w.roots {
		w.addTreeLocked(root)
	}
}

// coveredLocked reports whether some live root covers path: the
// bookkeeping that lets nested album roots share one watch.
func (w *folderWatcher) coveredLocked(path string) bool {
	for _, root := range w.roots {
		if pathWithin(root, path) {
			return true
		}
	}
	return false
}

// addTreeLocked registers every directory under root. A root already
// watched is skipped whole; a directory that cannot be read (a
// subtree the process lacks permission on) is skipped, not an error --
// the diff's walk owns the honest report of an unwalkable album.
func (w *folderWatcher) addTreeLocked(root string) {
	if w.degraded || w.closed {
		return
	}
	if _, ok := w.watched[root]; ok {
		return
	}
	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if w.degraded {
			// The table is full: every further Add would fail the
			// same way, so the walk stops here.
			return fs.SkipAll
		}
		if err != nil || !d.IsDir() {
			return nil
		}
		w.addWatchLocked(path)
		return nil
	})
}

// addWatchLocked registers one directory. Watch exhaustion is the
// expected ceiling of a big family tree on a default Linux host: it
// degrades the whole watcher once (with the hint that names the
// knob), and the diff carries on. Any other failure is a directory
// that came or went between the listing and the watch: logged at
// debug, covered by the diff.
func (w *folderWatcher) addWatchLocked(path string) {
	if w.degraded {
		return
	}
	if _, ok := w.watched[path]; ok {
		return
	}
	if err := w.backend.Add(path); err != nil {
		if errors.Is(err, syscall.ENOSPC) {
			w.degradeLocked("the inotify watch table is full; raise fs.inotify.max_user_watches or leave the periodic diff to cover the folders", err)
			return
		}
		w.logger.Debug("the folder watcher could not watch a directory", "path", path, "error", err)
		return
	}
	w.watched[path] = struct{}{}
}

// degradeLocked turns the watcher off for the process's lifetime. The
// backend keeps its descriptors (and its goroutine keeps draining
// events) until Close, so nothing blocks on the fsnotify channels;
// marking stops. The line is Error, not Warn: an operator who wanted
// events is told exactly which limit bit and which diff now covers
// the folders.
func (w *folderWatcher) degradeLocked(reason string, err error) {
	if w.degraded {
		return
	}
	w.degraded = true
	w.watched = make(map[string]struct{})
	if w.settle != nil {
		w.settle.Stop()
		w.settle = nil
	}
	w.logger.Error("the folder watcher is off; the periodic diff covers every folder",
		"reason", reason, "error", err)
}

// forgetTree drops the watches of a directory that left the tree.
func (w *folderWatcher) forgetTree(path string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	for watched := range w.watched {
		if !pathWithin(path, watched) {
			continue
		}
		delete(w.watched, watched)
		if err := w.backend.Remove(watched); err != nil && !errors.Is(err, fsnotify.ErrNonExistentWatch) {
			w.logger.Debug("the folder watcher could not drop a watch", "path", watched, "error", err)
		}
	}
}

// albumsUnderLocked resolves an event path to the albums whose roots
// contain it: nested album roots mean one file can be in two albums'
// folders, and both walls must converge.
func (w *folderWatcher) albumsUnderLocked(path string) []uuid.UUID {
	albums := make([]uuid.UUID, 0, 2)
	for albumID, root := range w.roots {
		if pathWithin(root, path) {
			albums = append(albums, albumID)
		}
	}
	return albums
}

// rootOfLocked reports whether path IS one of the live album roots.
func (w *folderWatcher) isRootLocked(path string) bool {
	for _, root := range w.roots {
		if root == path {
			return true
		}
	}
	return false
}

// mark records the albums a batch of events touched and (re)arms the
// shared debounce timer: the wake fires once the folder has been
// quiet for the debounce window.
func (w *folderWatcher) mark(albums []uuid.UUID) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.degraded || w.closed {
		return
	}
	for _, albumID := range albums {
		w.dirty[albumID] = struct{}{}
	}
	if w.settle == nil {
		w.settle = time.AfterFunc(w.debounce, w.settled)
		return
	}
	w.settle.Reset(w.debounce)
}

// settled runs when the debounce window elapses: the batch is over,
// so ask for a sweep. It is the only place wake is called, so a busy
// folder produces one wake per quiet window, never one per event.
func (w *folderWatcher) settled() {
	w.mu.Lock()
	w.settle = nil
	quiet := !w.degraded && !w.closed
	w.mu.Unlock()
	if !quiet || w.wake == nil {
		return
	}
	w.wake()
}

// SwapDirty hands the sweep the albums that changed and empties the
// set. An empty answer is nil, so the caller's scope check is the
// same one that skips the walk.
func (w *folderWatcher) SwapDirty() map[uuid.UUID]struct{} {
	w.mu.Lock()
	defer w.mu.Unlock()
	if len(w.dirty) == 0 {
		return nil
	}
	dirty := w.dirty
	w.dirty = make(map[uuid.UUID]struct{})
	return dirty
}

// Watched reports whether any directory under the album's root
// currently has a watch: the accelerator's own health view, for the
// worker's diagnostics and the tests' assertions.
func (w *folderWatcher) Watched(albumID uuid.UUID) bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	root, ok := w.roots[albumID]
	if !ok {
		return false
	}
	for path := range w.watched {
		if pathWithin(root, path) {
			return true
		}
	}
	return false
}

// Degraded reports whether the watcher gave up on events and left the
// folders to the periodic diff.
func (w *folderWatcher) Degraded() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.degraded
}

// Close stops the watcher: no more marking, the debounce timer is
// dropped, and the descriptor is released. The event loop exits; the
// call waits for it.
func (w *folderWatcher) Close() error {
	w.mu.Lock()
	if w.closed {
		w.mu.Unlock()
		return nil
	}
	w.closed = true
	if w.settle != nil {
		w.settle.Stop()
		w.settle = nil
	}
	w.mu.Unlock()

	close(w.done)
	err := w.backend.Close()
	w.wg.Wait()
	return err
}

// run is the event loop: it resolves each event to albums and marks
// them, watching directories as they appear and dropping watches as
// they vanish. A closed channel (a degraded backend, or Close) ends
// it; fsnotify's own errors are logged and swallowed -- the diff is
// the coverage, so an event stream hiccup is never fatal.
func (w *folderWatcher) run() {
	defer w.wg.Done()
	events, errs := w.backend.Events(), w.backend.Errors()
	for {
		select {
		case <-w.done:
			return
		case event, ok := <-events:
			if !ok {
				events = nil
				if errs == nil {
					return
				}
				continue
			}
			w.handleEvent(event)
		case err, ok := <-errs:
			if !ok {
				errs = nil
				if events == nil {
					return
				}
				continue
			}
			if err == nil {
				continue
			}
			w.logger.Warn("the folder watcher saw a filesystem error; the periodic diff covers the folders", "error", err)
		}
	}
}

// handleEvent turns one fsnotify event into watch bookkeeping and, if
// it names a real content change, a dirty mark. Metadata-only churn
// (a chmod, an atime) is ignored outright: the diff would compare the
// same stamp, so waking it would be pure disk I/O for no decision.
//
// A root that disappears is the one event that does NOT mark: an
// unmount is not a mass deletion, and the sweep's unwalkable-root
// skip is what owns that state. The watch simply goes.
func (w *folderWatcher) handleEvent(event fsnotify.Event) {
	const changes = fsnotify.Create | fsnotify.Write | fsnotify.Remove | fsnotify.Rename
	if event.Op&changes == 0 {
		return
	}
	path := filepath.Clean(event.Name)

	if event.Op&fsnotify.Create != 0 {
		if info, err := os.Lstat(path); err == nil && info.IsDir() {
			w.mu.Lock()
			w.addTreeLocked(path)
			w.mu.Unlock()
		}
	}
	if event.Op&(fsnotify.Remove|fsnotify.Rename) != 0 {
		w.forgetTree(path)
	}

	w.mu.Lock()
	vanishedRoot := event.Op&(fsnotify.Remove|fsnotify.Rename) != 0 && w.isRootLocked(path)
	albums := w.albumsUnderLocked(path)
	w.mu.Unlock()
	if vanishedRoot || len(albums) == 0 {
		return
	}
	w.mark(albums)
}

// pathWithin reports whether path is root itself or lies below it, on
// component boundaries only: "/a/b" is within "/a", "/a/b-extra" is
// not. Roots are cleaned before they reach here, and so is every
// event path.
func pathWithin(root, path string) bool {
	if path == root {
		return true
	}
	if root == "/" {
		return strings.HasPrefix(path, "/")
	}
	return strings.HasPrefix(path, root+string(filepath.Separator))
}
