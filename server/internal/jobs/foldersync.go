// The folder live sync: the periodic directory-snapshot diff that
// keeps every directory album's wall converged with the folder it
// references -- a file dropped into the source reaches the wall, a
// file removed from it leaves, no manual rescan.
//
// The mechanism is a periodic full-listing comparison, not a
// filesystem watcher. The source roots arrive as container bind
// mounts, and a mount boundary does not carry the host's filesystem
// events into the container: a watcher would sit silent on the one
// deployment shape this product ships. A periodic walk sees the
// truth on every platform, and the comparison is free on an idle
// folder -- no watcher to keep, no handle to leak, no schema.
//
// The diff's whole job is the DECISION. The walk's stamp set (path,
// size, mtime) is compared against the baseline the last resolving
// scan of the album observed; a difference enqueues the album's OWN
// rescan -- the three-arm diff that already owns the ingest, the
// vanish mark and the snapshot refresh. There is no second ingest
// path here: the folder's changes reach the wall through exactly the
// pipeline a manual rescan walks (the same add arm, the same
// owner+checksum dedup, the same metadata/thumbnail/transcode
// fan-out, the same asset.* events), and the deletion of a file the
// folder no longer holds is the pinned orphan semantics that arm
// already implements -- the row is marked deleted_at directly, the
// trash has no meaning for a file that was never the server's. A
// rename is therefore a delete plus an add, resolved by the same two
// arms.
//
// Two disciplines keep the periodic walk honest:
//
//   - An idle folder costs no writes in the album domain. The
//     comparison is the walk's memory-resident stamps against the
//     in-memory baseline (see AlbumSnapshots); equal sets touch
//     nothing -- no asset row, no album row, no job_runs row, no
//     event. The sweep's own tick is River's periodic bookkeeping (one
//     river_job row per window, independent of how many folders exist
//     or what they hold), never a per-album cost.
//
//   - The sweep never races a scan of the same album. An in-flight
//     album_rescan is jumped over (the endpoint's own in-flight
//     rule, read from this side; the scan in flight owns the next
//     baseline), and a tick that overlaps the previous tick is
//     skipped outright (the worker's own try-lock), so a slow walk
//     can never stack scans behind itself.
package jobs

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"

	dbgen "github.com/welpher/lumemor/server/internal/db/gen"
)

// folderSyncKind is the River job kind of the periodic sweep: one
// registration, one worker, no per-album dispatch -- the sweep's
// only enqueue is the existing album_rescan job.
const folderSyncKind = "folder_sync"

// folderSyncPeriodicJobID is the periodic registration's identity:
// River keys its periodic bundle by it, and the tests remove the
// registration by the same id.
const folderSyncPeriodicJobID = "folder_sync"

// defaultFolderSyncInterval is the sweep's production cadence: the
// window inside which a folder change reaches the wall. Sixty
// seconds is the product pin (a family folder is not a chat feed;
// the wall converging within a minute reads as instant, and a more
// eager walk multiplies the idle listing cost of every album for
// nothing). The deployment can shorten it (the e2e harness does) or
// turn the sweep off through LUMEMOR_FOLDER_SYNC_INTERVAL.
const defaultFolderSyncInterval = time.Minute

// folderSyncTimeout raises the sweep's execution envelope above
// River's one minute default: one tick walks every directory album's
// source root, and a family with many albums on a slow volume
// legitimately outlasts a small envelope. An expired attempt hands
// the tick to the retry; the next tick's comparison converges the
// same way (the walk and the comparison are the same deterministic
// function of the same filesystem).
const folderSyncTimeout = 10 * time.Minute

// fileStamp is one file's identity in the folder sync's comparison:
// the size and the mtime at the same microsecond precision the scan
// compares at (the PostgreSQL timestamptz round-trip), so a stamp
// can never drift against its own stored form and an unchanged file
// can never look changed.
type fileStamp struct {
	size  int64
	mtime time.Time
}

// AlbumSnapshots is the folder sync's baseline store: per directory
// album, the stamp set its last resolving walk observed. It lives in
// memory on purpose.
//
// The value is a derived cache of a filesystem the process does not
// own, and its only truthful state after a restart is "unknown": the
// process was down, the folders may have changed without a witness,
// so the first tick after a boot re-scans every album once
// (the enqueue is idempotent and the unchanged re-scan writes
// nothing) and re-baselines. A table would persist exactly the value
// nobody may trust at that moment, and it would turn the idle folder
// -- this design's one refusal -- into a write every time somebody
// moved a file.
type AlbumSnapshots struct {
	mu     sync.Mutex
	albums map[uuid.UUID]map[string]fileStamp
}

// NewAlbumSnapshots builds an empty baseline store: no album known,
// so every directory album's first sweep enqueues its convergence
// scan.
func NewAlbumSnapshots() *AlbumSnapshots {
	return &AlbumSnapshots{albums: make(map[uuid.UUID]map[string]fileStamp)}
}

// Same answers whether the walk's stamps equal the album's baseline,
// and whether a baseline exists at all -- the sweep's whole
// comparison, done under the store's lock and without copying. An
// unknown album is "not the same": the sweep cannot vouch for
// changes it never saw, so it converges the album once.
func (s *AlbumSnapshots) Same(albumID uuid.UUID, stamps map[string]fileStamp) (known, same bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	baseline, ok := s.albums[albumID]
	if !ok {
		return false, false
	}
	if len(baseline) != len(stamps) {
		return true, false
	}
	for path, stamp := range stamps {
		held, ok := baseline[path]
		if !ok || held.size != stamp.size || !held.mtime.Equal(stamp.mtime) {
			return true, false
		}
	}
	return true, true
}

// Store replaces an album's baseline with the stamps of the walk that
// just resolved it. The scan worker calls it with the walk's own view
// (taken BEFORE its arms run), so anything that changes while the
// arms work is still a difference the next tick sees -- the
// conservative direction: a redundant scan, never a missed file.
func (s *AlbumSnapshots) Store(albumID uuid.UUID, stamps map[string]fileStamp) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.albums[albumID] = stamps
}

// Prune drops every baseline whose album the sweep no longer listed:
// a removed album (the un-ingest) has nothing left to compare, and
// its stamps must not outlive the row for the process's lifetime.
// The sweep is the only reader, so it is the natural place to keep
// the store sized to the live albums.
func (s *AlbumSnapshots) Prune(live map[uuid.UUID]struct{}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for albumID := range s.albums {
		if _, ok := live[albumID]; !ok {
			delete(s.albums, albumID)
		}
	}
}

// FolderSyncEnqueueFunc starts one album's rescan: the seam the
// queue wires to EnqueueAlbumRescan (the run's job_runs row and its
// River job in one transaction). The tests inject fakes of the same
// contract.
type FolderSyncEnqueueFunc func(ctx context.Context, albumID uuid.UUID) (uuid.UUID, error)

// FolderSyncCounts carries one sweep's report: the albums listed,
// the ones the baseline already matched (the idle arm), the ones a
// scan in flight already covers, and the scans this sweep enqueued.
type FolderSyncCounts struct {
	Albums    int
	Unchanged int
	Skipped   int
	Enqueued  int
}

// RunFolderSync performs one sweep: list the directory albums, walk
// each source root, and enqueue the album's rescan wherever the walk
// disagrees with the baseline (or no baseline exists yet). It is the
// shared function the periodic worker runs and the tests drive
// directly -- the sweep's own shape, independent of River's clock.
//
// A root that cannot be walked is skipped with a warning, never
// enqueued: an unwalkable root is the owner's state to see (the
// manual rescan reports it honestly as a failed run), and a
// diff-driven scan over half a listing would mark members vanished
// on the strength of an I/O error. The listing itself failing fails
// the sweep -- the caller retries it, and nothing was decided.
func RunFolderSync(ctx context.Context, pool *pgxpool.Pool, snapshots *AlbumSnapshots, enqueue FolderSyncEnqueueFunc, logger *slog.Logger) (FolderSyncCounts, error) {
	var counts FolderSyncCounts
	if snapshots == nil || enqueue == nil {
		// A wiring gap, not a runtime state: the sweep has nothing to
		// compare against or nowhere to send its decisions.
		return counts, errors.New("jobs: the folder sync needs a baseline store and an enqueue seam")
	}
	queries := dbgen.New(pool)
	albums, err := queries.ListDirectoryAlbums(ctx)
	if err != nil {
		return counts, fmt.Errorf("list the directory albums: %w", err)
	}
	counts.Albums = len(albums)
	live := make(map[uuid.UUID]struct{}, len(albums))

	for _, album := range albums {
		if err := ctx.Err(); err != nil {
			// The envelope ran out mid-sweep: the undecided albums are
			// the next tick's work, and returning the error keeps the
			// truncation visible instead of reporting a clean sweep.
			return counts, err
		}
		albumID := uuid.UUID(album.ID.Bytes)
		live[albumID] = struct{}{}
		root := album.RootPath.String
		// The scan's own root guard, mirrored on the sweep's side: a
		// root that now resolves elsewhere (a symlink swapped in
		// after creation) fails every scan this sweep could enqueue
		// -- the walk would see the swapped target's tree, the diff
		// would ask for a scan, and the scan would refuse the root,
		// once per tick, forever. Guarding here keeps the sweep from
		// enqueueing a run that can only fail.
		resolved, resolveErr := filepath.EvalSymlinks(root)
		if resolveErr != nil || resolved != root {
			logger.Warn("the folder sync skips a source root the scan would refuse",
				"album_id", albumID, "root", root, "error", resolveErr)
			counts.Skipped++
			continue
		}
		files, err := walkSource(root)
		if err != nil {
			logger.Warn("the folder sync could not walk a source root",
				"album_id", albumID, "root", root, "error", err)
			counts.Skipped++
			continue
		}
		if known, same := snapshots.Same(albumID, stampsOf(files)); known && same {
			// The idle arm: the folder is exactly what the last scan
			// left, so nothing is written anywhere and nothing fires.
			counts.Unchanged++
			continue
		}
		// A scan of this album still in flight owns the next baseline:
		// it is walking the same folder RIGHT NOW, and its walk will
		// record what it saw. Enqueueing a second run would only
		// queue behind it.
		if _, err := queries.FindInFlightAlbumRescan(ctx, dbgen.FindInFlightAlbumRescanParams{
			Kind:    AlbumRescanKind,
			AlbumID: album.ID,
		}); err == nil {
			counts.Skipped++
			continue
		} else if !errors.Is(err, pgx.ErrNoRows) {
			logger.Warn("the folder sync could not read an album's in-flight scan",
				"album_id", albumID, "error", err)
			counts.Skipped++
			continue
		}
		if _, err := enqueue(ctx, albumID); err != nil {
			logger.Warn("the folder sync could not enqueue an album's scan",
				"album_id", albumID, "error", err)
			counts.Skipped++
			continue
		}
		counts.Enqueued++
	}
	snapshots.Prune(live)
	return counts, nil
}

// FolderSyncArgs is the periodic sweep job: it carries no payload,
// because the sweep reads its albums and their roots from the
// database.
type FolderSyncArgs struct{}

// Kind is River's stable job kind identifier.
func (FolderSyncArgs) Kind() string { return folderSyncKind }

// FolderSyncWorker runs the periodic sweep job. Like the other
// maintenance workers it holds the pool directly (the workers are
// registered before the client exists, so it cannot hold the *Queue
// itself); the enqueue seam is wired at start.
type FolderSyncWorker struct {
	river.WorkerDefaults[FolderSyncArgs]

	pool      *pgxpool.Pool
	snapshots *AlbumSnapshots
	enqueue   FolderSyncEnqueueFunc
	logger    *slog.Logger
	// tick is the sweep's single-flight: a tick that overlaps the
	// previous one is dropped whole. River's periodic scheduler
	// inserts a job per tick and would happily run two sweeps at
	// once; refusing the overlap keeps the sweep's cost bounded and
	// its scan decisions unstacked.
	tick sync.Mutex
}

// Timeout is the sweep's execution envelope; see folderSyncTimeout.
func (w *FolderSyncWorker) Timeout(job *river.Job[FolderSyncArgs]) time.Duration {
	return folderSyncTimeout
}

// Work executes one sweep. The try-lock's loser logs and returns
// clean: the tick it skipped is the running sweep's own work, not a
// failure to record. Nothing is logged for an idle sweep -- one
// line per minute per process would be the only noise the feature
// makes -- and one line per sweep that decided something is.
func (w *FolderSyncWorker) Work(ctx context.Context, job *river.Job[FolderSyncArgs]) error {
	if !w.tick.TryLock() {
		w.logger.Debug("the folder sync tick skipped: the previous sweep is still walking")
		return nil
	}
	defer w.tick.Unlock()

	counts, err := RunFolderSync(ctx, w.pool, w.snapshots, w.enqueue, w.logger)
	if err != nil {
		w.logger.Error("the folder sync sweep failed",
			"error", err, "albums", counts.Albums, "enqueued", counts.Enqueued)
		return fmt.Errorf("folder sync: %w", err)
	}
	if counts.Enqueued > 0 {
		w.logger.Info("the folder sync enqueued the scans of changed folders",
			"albums", counts.Albums, "unchanged", counts.Unchanged,
			"skipped", counts.Skipped, "enqueued", counts.Enqueued)
	}
	return nil
}

// folderSyncCadence resolves the configured sweep interval: the
// config's zero value means the production default, a negative value
// stays negative (the disabled marker periodicJobs reads), anything
// positive is the deployment's own window.
func folderSyncCadence(configured time.Duration) time.Duration {
	if configured == 0 {
		return defaultFolderSyncInterval
	}
	return configured
}

// folderSyncPeriodicJobs returns the sweep's periodic registration on
// the configured cadence. RunOnStart is ON here (unlike the hourly
// maintenance jobs): a boot's first sweep is what converges the
// folders that changed while the server was down, and there is no
// boot-time equivalent running the same function -- the empty
// baseline store already makes that first sweep a full convergence
// pass by construction.
func folderSyncPeriodicJobs(interval time.Duration) []*river.PeriodicJob {
	return []*river.PeriodicJob{
		river.NewPeriodicJob(folderSyncSchedule(interval), folderSyncJobConstructor,
			&river.PeriodicJobOpts{ID: folderSyncPeriodicJobID, RunOnStart: true}),
	}
}

// folderSyncSchedule is the sweep's periodic schedule: the cadence
// the deployment configured. Split out so the cadence is
// unit-testable without waiting a wall-clock minute.
func folderSyncSchedule(interval time.Duration) river.PeriodicSchedule {
	return river.PeriodicInterval(interval)
}

// folderSyncJobConstructor is the periodic job's insert payload: the
// argumentless sweep job on the default queue with the default
// priority -- a per-minute maintenance job has no business
// preempting pipeline work.
func folderSyncJobConstructor() (river.JobArgs, *river.InsertOpts) {
	return FolderSyncArgs{}, nil
}
