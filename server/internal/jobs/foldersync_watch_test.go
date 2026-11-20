// The folder sync's watcher tests: the accelerator marks the album a
// change happened in, debounces a burst into one look, follows new
// subdirectories, drops a vanished root without calling it a mass
// deletion, and degrades to the periodic diff on watch exhaustion.
//
// Two backends are exercised on purpose. The rules are driven through
// the seam (a fake fsnotify) so every platform proves the same table --
// in-place writes are not surfaced on a directory watch by every
// kernel, and the rules are not platform facts. The real fsnotify
// backend is exercised where directory-entry events are universal
// (a created file, a created subdirectory), because that is the path
// production runs.
package jobs

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/google/uuid"
)

// fakeWatchBackend is the watcher seam without a kernel: the test adds
// directories, pushes events, and reads what the watcher decided.
type fakeWatchBackend struct {
	mu      sync.Mutex
	addErr  error
	added   map[string]int
	dropped map[string]int
	events  chan fsnotify.Event
	errs    chan error
	closed  sync.Once
}

func newFakeWatchBackend() *fakeWatchBackend {
	return &fakeWatchBackend{
		added:   make(map[string]int),
		dropped: make(map[string]int),
		events:  make(chan fsnotify.Event, 64),
		errs:    make(chan error, 8),
	}
}

func (f *fakeWatchBackend) Add(name string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.addErr != nil {
		return f.addErr
	}
	f.added[name]++
	return nil
}

func (f *fakeWatchBackend) Remove(name string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.dropped[name]++
	return nil
}

func (f *fakeWatchBackend) Events() <-chan fsnotify.Event { return f.events }
func (f *fakeWatchBackend) Errors() <-chan error          { return f.errs }
func (f *fakeWatchBackend) Close() error {
	f.closed.Do(func() {
		close(f.events)
		close(f.errs)
	})
	return nil
}

// watchOf reports whether the backend currently holds name's watch.
func (f *fakeWatchBackend) watchOf(name string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.added[name] > f.dropped[name]
}

// testWatcher builds a watcher over a fake backend and returns it with
// its wake channel: one receive per settled debounce window.
func testWatcher(t *testing.T, backend folderWatchBackend) (*folderWatcher, chan struct{}) {
	t.Helper()
	wakes := make(chan struct{}, 16)
	watcher := newFolderWatcher(backend, 40*time.Millisecond,
		slog.New(slog.NewTextHandler(os.Stderr, nil)), func() {
			select {
			case wakes <- struct{}{}:
			default:
			}
		})
	t.Cleanup(func() {
		if err := watcher.Close(); err != nil {
			t.Errorf("close the watcher: %v", err)
		}
	})
	return watcher, wakes
}

// waitSettle waits for one settled window: the debounce elapsed and the
// wake fired.
func waitSettle(t *testing.T, wakes <-chan struct{}) {
	t.Helper()
	select {
	case <-wakes:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for the watcher's debounced wake")
	}
}

// expectQuiet fails when a wake arrives inside the window.
func expectQuiet(t *testing.T, wakes <-chan struct{}, window time.Duration) {
	t.Helper()
	select {
	case <-wakes:
		t.Fatal("the watcher woke the sweep, want it quiet")
	case <-time.After(window):
	}
}

// TestFolderWatcherEventRules drives the accelerator's table over the
// seam: which events mark which album, which ones are noise, and how a
// burst collapses into one wake.
func TestFolderWatcherEventRules(t *testing.T) {
	albumA, albumB := uuid.New(), uuid.New()
	rootA := filepath.Join(t.TempDir(), "a")
	rootB := filepath.Join(t.TempDir(), "b")
	for _, root := range []string{rootA, rootB} {
		if err := os.MkdirAll(root, 0o755); err != nil {
			t.Fatalf("create the source root: %v", err)
		}
	}

	backend := newFakeWatchBackend()
	watcher, wakes := testWatcher(t, backend)
	watcher.syncRoots(map[uuid.UUID]string{albumA: rootA, albumB: rootB})
	if !watcher.Watched(albumA) || !watcher.Watched(albumB) {
		t.Fatal("the watcher did not register both album roots")
	}

	// A write marks the album it happened in, and only that album.
	backend.events <- fsnotify.Event{Name: filepath.Join(rootA, "photo.jpg"), Op: fsnotify.Write}
	waitSettle(t, wakes)
	dirty := watcher.SwapDirty()
	if _, ok := dirty[albumA]; !ok || len(dirty) != 1 {
		t.Errorf("the dirty set after a write in A = %v, want A alone", dirty)
	}

	// Metadata-only churn is not a content change: the diff would
	// compare the same stamp.
	backend.events <- fsnotify.Event{Name: filepath.Join(rootA, "photo.jpg"), Op: fsnotify.Chmod}
	expectQuiet(t, wakes, 200*time.Millisecond)

	// A path outside every album root is not ours to watch.
	backend.events <- fsnotify.Event{Name: filepath.Join(t.TempDir(), "elsewhere.jpg"), Op: fsnotify.Write}
	expectQuiet(t, wakes, 200*time.Millisecond)
	if dirty := watcher.SwapDirty(); dirty != nil {
		t.Errorf("the dirty set after noise = %v, want empty", dirty)
	}

	// A burst spanning both albums settles into ONE wake carrying both:
	// the debounce is the batch's quiet window, not a per-event timer.
	backend.events <- fsnotify.Event{Name: filepath.Join(rootA, "one.jpg"), Op: fsnotify.Create}
	backend.events <- fsnotify.Event{Name: filepath.Join(rootB, "two.jpg"), Op: fsnotify.Create}
	backend.events <- fsnotify.Event{Name: filepath.Join(rootA, "three.jpg"), Op: fsnotify.Write}
	waitSettle(t, wakes)
	dirty = watcher.SwapDirty()
	if len(dirty) != 2 {
		t.Errorf("the burst's dirty set = %v, want both albums", dirty)
	}
	expectQuiet(t, wakes, 200*time.Millisecond)

	// A root that disappears is not a mass deletion: the watch goes and
	// nothing is marked -- the diff's unwalkable-root skip owns the
	// state.
	backend.events <- fsnotify.Event{Name: rootB, Op: fsnotify.Remove}
	expectQuiet(t, wakes, 200*time.Millisecond)
	if watcher.Watched(albumB) {
		t.Error("the vanished root kept its watch, want it dropped")
	}
	if dirty := watcher.SwapDirty(); dirty != nil {
		t.Errorf("the vanished root marked %v dirty, want nothing", dirty)
	}

	// The reconciliation re-arms a root that came back: the album is
	// still live, and the next sweep's Sync is what notices it.
	watcher.syncRoots(map[uuid.UUID]string{albumA: rootA, albumB: rootB})
	if !watcher.Watched(albumB) {
		t.Error("the reconciliation did not re-watch the live album's root")
	}

	// An album that leaves the listing loses its root and its watches.
	watcher.syncRoots(map[uuid.UUID]string{albumA: rootA})
	if watcher.Watched(albumB) {
		t.Error("the removed album kept its watch, want it dropped with the row")
	}
	if backend.watchOf(rootB) {
		t.Error("the removed album's root kept its backend watch, want it removed")
	}
}

// TestFolderWatcherDegradesOnWatchExhaustion pins the honest ceiling: an
// inotify table that cannot take another watch turns the accelerator off
// once, and nothing it saw afterwards marks a sweep.
func TestFolderWatcherDegradesOnWatchExhaustion(t *testing.T) {
	album := uuid.New()
	root := filepath.Join(t.TempDir(), "family")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatalf("create the source root: %v", err)
	}

	backend := newFakeWatchBackend()
	backend.addErr = syscall.ENOSPC
	watcher, wakes := testWatcher(t, backend)
	watcher.syncRoots(map[uuid.UUID]string{album: root})

	if !watcher.Degraded() {
		t.Fatal("the watcher did not degrade on the exhausted watch table")
	}
	if watcher.Watched(album) {
		t.Error("the degraded watcher still reports a watch")
	}
	backend.events <- fsnotify.Event{Name: filepath.Join(root, "photo.jpg"), Op: fsnotify.Write}
	expectQuiet(t, wakes, 200*time.Millisecond)
	if dirty := watcher.SwapDirty(); dirty != nil {
		t.Errorf("the degraded watcher marked %v dirty, want nothing", dirty)
	}

	// A backend error on the error channel is logged and survived: the
	// stream is not the decider, so it must not stop the watcher.
	other := newFakeWatchBackend()
	healthy, healthyWakes := testWatcher(t, other)
	healthy.syncRoots(map[uuid.UUID]string{album: root})
	other.errs <- errors.New("the event stream overflowed")
	other.events <- fsnotify.Event{Name: filepath.Join(root, "late.jpg"), Op: fsnotify.Create}
	waitSettle(t, healthyWakes)
	if _, ok := healthy.SwapDirty()[album]; !ok {
		t.Error("the watcher stopped marking after a stream error, want it to carry on")
	}
}

// TestFolderWatcherFollowsTheTree exercises the real fsnotify backend on
// the events every platform surfaces for a directory watch: a created
// file marks the album, a created subdirectory is watched, and a file
// inside that subdirectory marks the album too.
func TestFolderWatcherFollowsTheTree(t *testing.T) {
	album := uuid.New()
	root := filepath.Join(t.TempDir(), "family")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatalf("create the source root: %v", err)
	}

	wakes := make(chan struct{}, 16)
	watcher, err := newFSNotifyWatcher(40*time.Millisecond,
		slog.New(slog.NewTextHandler(os.Stderr, nil)), func() {
			select {
			case wakes <- struct{}{}:
			default:
			}
		})
	if err != nil {
		t.Fatalf("create the real watcher: %v", err)
	}
	t.Cleanup(func() {
		if err := watcher.Close(); err != nil {
			t.Errorf("close the watcher: %v", err)
		}
	})
	watcher.syncRoots(map[uuid.UUID]string{album: root})

	if err := os.WriteFile(filepath.Join(root, "a.jpg"), []byte("one"), 0o644); err != nil {
		t.Fatalf("write the source file: %v", err)
	}
	waitSettle(t, wakes)
	if _, ok := watcher.SwapDirty()[album]; !ok {
		t.Fatal("a created file did not mark its album dirty")
	}

	// The new directory's own creation is an event; once that window
	// settles, the watcher has registered it (the handler adds the
	// watch before it marks), so the file inside it is observed.
	sub := filepath.Join(root, "2026")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatalf("create the subdirectory: %v", err)
	}
	waitSettle(t, wakes)
	watcher.SwapDirty()

	if err := os.WriteFile(filepath.Join(sub, "b.jpg"), []byte("two"), 0o644); err != nil {
		t.Fatalf("write the nested source file: %v", err)
	}
	waitSettle(t, wakes)
	if _, ok := watcher.SwapDirty()[album]; !ok {
		t.Fatal("a file inside a new subdirectory did not mark the album dirty")
	}
}

// TestFolderSyncWatcherUnchangedStampDecidesNothing is the pin that
// keeps the accelerator an accelerator: an event only asks the diff to
// look, and the stamp comparison is what decides. A write that leaves
// the file's identity intact produces no scan; a real new file in the
// same folder produces one through the same sweep.
func TestFolderSyncWatcherUnchangedStampDecidesNothing(t *testing.T) {
	requireExiftool(t)
	requireThumbnails(t)
	e := newEnv(t)

	source := filepath.Join(t.TempDir(), "family")
	if err := os.MkdirAll(source, 0o755); err != nil {
		t.Fatalf("create the source directory: %v", err)
	}
	stamp := time.Date(2026, 11, 20, 7, 0, 0, 0, time.UTC)
	writeSourceFile(t, filepath.Join(source, "a.jpg"), jpegBytes(t), stamp)

	album := e.createDirectoryAlbum(t, "Family archive", source)
	first := e.waitCreationScan(t, album)
	e.syncNow(t)

	backend := newFakeWatchBackend()
	watcher, wakes := testWatcher(t, backend)
	watcher.syncRoots(map[uuid.UUID]string{album: source})

	// The event: the album's file was written, but its stamp (size and
	// mtime) is exactly the baseline's -- a re-save that changed no
	// identity, the case the watcher must not turn into churn. The diff
	// decides, and decides nothing.
	backend.events <- fsnotify.Event{Name: filepath.Join(source, "a.jpg"), Op: fsnotify.Write}
	waitSettle(t, wakes)
	if _, ok := watcher.SwapDirty()[album]; !ok {
		t.Fatal("the event did not mark the album dirty, want the diff to be asked to look")
	}
	counts, err := runFolderSync(context.Background(), e.pool, e.queue.snapshots,
		e.queue.EnqueueAlbumRescan, e.queue.logger, map[uuid.UUID]struct{}{album: {}})
	if err != nil {
		t.Fatalf("run the scoped sweep: %v", err)
	}
	if counts.Enqueued != 0 || counts.Unchanged != 1 || counts.Albums != 1 {
		t.Errorf("the scoped sweep over an unchanged stamp = %+v, want the album unchanged and nothing enqueued", counts)
	}

	// The decider still works on the same path: a real new file, an
	// event, and the scoped sweep enqueues the album's own scan.
	writeSourceFile(t, filepath.Join(source, "b.jpg"), jpegBytes(t), stamp.Add(time.Hour))
	backend.events <- fsnotify.Event{Name: filepath.Join(source, "b.jpg"), Op: fsnotify.Create}
	waitSettle(t, wakes)
	if _, ok := watcher.SwapDirty()[album]; !ok {
		t.Fatal("the created file did not mark the album dirty")
	}
	counts, err = runFolderSync(context.Background(), e.pool, e.queue.snapshots,
		e.queue.EnqueueAlbumRescan, e.queue.logger, map[uuid.UUID]struct{}{album: {}})
	if err != nil {
		t.Fatalf("run the scoped sweep: %v", err)
	}
	if counts.Enqueued != 1 {
		t.Errorf("the scoped sweep over a created file = %+v, want the album's scan enqueued", counts)
	}
	e.waitNewScan(t, album, first)

	// The scoped sweep speaks only for the albums it walked: another
	// album's baseline must survive it (a full sweep is what prunes).
	other := filepath.Join(t.TempDir(), "other")
	if err := os.MkdirAll(other, 0o755); err != nil {
		t.Fatalf("create the second source directory: %v", err)
	}
	writeSourceFile(t, filepath.Join(other, "c.jpg"), jpegBytes(t), stamp)
	otherAlbum := e.createDirectoryAlbum(t, "Second archive", other)
	e.waitCreationScan(t, otherAlbum)
	e.syncNow(t)
	if _, err := runFolderSync(context.Background(), e.pool, e.queue.snapshots,
		e.queue.EnqueueAlbumRescan, e.queue.logger, map[uuid.UUID]struct{}{album: {}}); err != nil {
		t.Fatalf("run the scoped sweep: %v", err)
	}
	if known, _ := e.queue.snapshots.Same(otherAlbum, nil); !known {
		t.Error("the scoped sweep pruned an album it never walked, want the baseline kept")
	}
}
