// The directory album rescan: the three-arm diff that
// reconciles a directory album's membership with what its source
// root holds on the filesystem. The endpoint (POST
// /albums/{id}/rescan) starts one run through EnqueueAlbumRescan;
// this worker executes it.
//
// The diff's shape is the pinned arm table, one arm per
// resolution:
//
//   - ADD: a file no live member matches is ingested by reference
//     (a new external asset row pointing AT the file where it lies,
//     zero copy zero move, plus its album membership) through the
//     EXISTING ingest pipeline -- the same metadata / thumbnail /
//     transcode jobs an upload's transaction commits enqueue. No
//     parallel pipeline exists here: the format whitelist and
//     magic-byte sniffing are the storage package's own tables, and
//     the pipeline jobs read the external row's real path through the
//     kind-aware resolver. The row is keyed by its POSITION
//     (owner+file_path), never by content: a folder mirroring files
//     the owner also uploaded gets its own rows -- the album must be
//     able to delete its own copy through the application, which a
//     borrowed upload row could never do. The position's three states
//     resolve the one collision the insert can meet: a LIVE row at
//     the path is another album's reference (or a racing scan's) and
//     gains this album's membership; an ORPHAN-marked row at the path
//     is this very file's earlier death (VANISH below) and revives
//     carrying the file's current identity; a row whose bytes the
//     trash holds never matches (its path names the trash copy).
//
//   - VANISH (the pinned orphan arm, 2026-11-08): a live member
//     whose file is gone from disk is marked deleted_at directly --
//     the file is not the server's and the trash has no meaning for
//     it. No missing state: the membership row stays (soft delete
//     never removed membership rows), and a file that returns to the
//     SAME path revives that row through the add arm's position-key
//     resolution, while one that returns elsewhere lands a fresh row.
//
//   - CONVERGE (the position model, migration 00027): after the walk
//     and the vanish arm, a same-content membership of this album that
//     is not one of its own position rows leaves -- the pre-fork
//     scan's dedup links (a blob row the album borrowed; another
//     album's twin row) swap for the album's own copy. Idempotent: a
//     converged album has no such memberships left to swap.
//
//   - CHANGE: a member whose mtime+size snapshot differs has its
//     checksum re-read (the full verification runs only now -- never
//     pre-read), the snapshot re-stamped, and the metadata pipeline
//     re-run (the jobs' own idempotent re-run contract: the exif
//     upsert converges, the derivations overwrite the same outputs,
//     the guarded readiness decrement does not fire, no second
//     asset.ready). The ML arm follows that path's answer -- nothing
//     re-fires, readiness is already stamped; the admin's ML rerun
//     surface owns any deliberate re-run.
//
// The run's job_runs row carries the progress the places
// re-resolution established: started, throttled progress, finished,
// done/failed counting resolved walk entries and vanished members.
// An interruption leaves the row running and the work resumable:
// River retries the job and the re-walk converges -- every arm is
// the same deterministic function of the same filesystem and
// database state, so re-running an arm is a no-op (the guards in
// the vanishing and snapshot statements match nothing on the
// second pass, the add arm's insert collides and resolves to the
// membership it already landed, the convergence deletes nothing).
package jobs

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"

	dbgen "github.com/welpher/lumemor/server/internal/db/gen"
	"github.com/welpher/lumemor/server/internal/events"
	"github.com/welpher/lumemor/server/internal/storage"
)

// AlbumRescanKind is the job_runs kind of the directory rescan runs
// and the River job kind of their driver, pinned by the shared
// vocabulary (the places rerun family's naming applied to the
// rescan domain: one driver, no per-asset dispatch -- the arms
// resolve inline, the pipeline jobs the add arm enqueues are the
// per-asset work).
const AlbumRescanKind = "album_rescan"

// albumRescanTimeout raises the worker's execution envelope above
// River's one minute default: the walk plus the per-file full reads
// (a first scan hashes every file of the source) legitimately
// outlast a small envelope, and an expired attempt hands the job to
// the retry, whose re-walk converges (every arm idempotent).
const albumRescanTimeout = 30 * time.Minute

// albumRescanProgressBatch bounds the row's progress writes: one
// bounded update per batch of resolved units keeps the run's own
// bookkeeping cheap while the panel's truth stays batch-fresh.
const albumRescanProgressBatch = 100

// albumRescanProgressInterval paces the run's progress events onto
// the activity stream: at most one per interval, read off the
// resolution's own clock (the row itself updates on every batch,
// the events ride the batches they land on).
const albumRescanProgressInterval = time.Second

// uniqueViolationCode is PostgreSQL's 23505, the only error the add
// arm's insert resolves instead of failing: the owner+file_path
// uniqueness IS the position key, so a violation routes into the
// three-state resolution exactly as the upload path's conflict
// resolution routes its own. Any other constraint (the blob domain's
// content key is not reachable from an external insert) stays an
// honest failure.
const uniqueViolationCode = "23505"

// externalPositionConstraint is the position unique index's name: the
// one violation the add arm resolves.
const externalPositionConstraint = "assets_owner_file_path_external_uq"

// RescanDispatchFunc runs one externally ingested asset's pipeline
// enqueue: the metadata and thumbnail jobs every asset counts, plus
// the transcode job a video counts -- the same fan-out the upload
// handler performs after its transaction commits, applied to the
// external row the rescan just landed. The queue wires its real
// dispatcher by default; the tests inject fakes of the same
// contract. The change arm calls the same seam: a re-run is the
// pipeline's own idempotent re-run, the identical job set the
// ingest enqueued.
type RescanDispatchFunc func(ctx context.Context, assetID uuid.UUID, kind dbgen.MediaKind) error

// AlbumRescanArgs identifies the run the worker walks and the album
// whose source root it reconciles.
type AlbumRescanArgs struct {
	JobRunID uuid.UUID `json:"jobRunId"`
	AlbumID  uuid.UUID `json:"albumId"`
}

// Kind is River's stable job kind identifier.
func (AlbumRescanArgs) Kind() string { return AlbumRescanKind }

// AlbumRescanWorker runs one directory rescan.
type AlbumRescanWorker struct {
	river.WorkerDefaults[AlbumRescanArgs]

	pool *pgxpool.Pool
	bus  *events.Bus
	// dispatch is the ingest seam (the queue's real dispatcher; the
	// tests' fakes). Nil means a directly constructed worker -- a
	// wiring bug -- and the run closes failed rather than ingesting
	// assets whose pipeline never starts.
	dispatch RescanDispatchFunc
	// snapshots is the folder sync's baseline store: the run records
	// the walk's own view of the source here, which is what the
	// periodic diff compares against. Nil (a directly constructed
	// worker) simply leaves the diff to its next convergence scan.
	snapshots *AlbumSnapshots
	logger    *slog.Logger
}

// Timeout is the worker's execution envelope; see albumRescanTimeout.
func (w *AlbumRescanWorker) Timeout(job *river.Job[AlbumRescanArgs]) time.Duration {
	return albumRescanTimeout
}

// rescanMember is one live external member's diff identity: the
// path the walk matches on (the map key, held by the caller), the
// mtime+size snapshot the freshness comparison reads, the media
// kind the event payloads and pipeline dispatch carry, and the
// row's id for the arms' writes. The checksum stays unread here on
// purpose -- the pin is "never pre-read": the comparison is the
// snapshot alone, and the checksum is read only inside the change
// arm's full verification.
type rescanMember struct {
	id      uuid.UUID
	kind    dbgen.MediaKind
	size    int64
	mtime   time.Time
	hasSnap bool
}

// albumDiffer carries the shared diff's dependencies. The single
// album worker and the instance-wide sweep both project their fields
// into one (see differ) and run the same code, so an album resolves
// identically whichever run asked.
type albumDiffer struct {
	pool      *pgxpool.Pool
	bus       *events.Bus
	dispatch  RescanDispatchFunc
	snapshots *AlbumSnapshots
	logger    *slog.Logger
}

// differ projects the single-album worker's dependencies for the
// shared diff.
func (w *AlbumRescanWorker) differ() *albumDiffer {
	return &albumDiffer{
		pool:      w.pool,
		bus:       w.bus,
		dispatch:  w.dispatch,
		snapshots: w.snapshots,
		logger:    w.logger,
	}
}

// albumDiffOptions carries the caller's run wiring for one directory
// album's diff. The single-album worker and the all-albums sweep run
// the SAME diff (diffAlbum below): only the bookkeeping around it
// differs -- the single run sizes its row to the diff's units, the
// sweep counts albums.
type albumDiffOptions struct {
	// jobRunID correlates the diff's log lines and names the run whose
	// stop flag the walk honors.
	jobRunID uuid.UUID
	// currentTotal is the run row's total as the caller announced it.
	// The diff compares its sized total against it and writes through
	// sizeRun only when the two diverge.
	currentTotal int
	// sizeRun writes the sized diff total to the caller's run row. The
	// single-album run passes it (its row's units ARE the diff's); the
	// all-albums sweep passes nil (its total counts albums).
	sizeRun func(ctx context.Context, total int) error
	// progress records one batch of resolved units on the caller's run
	// row and announces it on the stream. The single-album run passes
	// it; the sweep passes nil (its own driver counts albums).
	progress func(ctx context.Context, done, failed, total int)
	// stopRequested reads the caller's run stop flag at the unit-batch
	// cadence. Nil honors no stop (a directly constructed caller).
	stopRequested func(ctx context.Context) (bool, error)
}

// albumDiffOutcome is one album's diff result: the converged unit
// counts, whether the walk stopped at its cancel boundary, and
// whether the album could not be diffed at all. The caller decides
// what a fatal outcome means for its run.
type albumDiffOutcome struct {
	total     int
	done      int
	failed    int
	cancelled bool
	fatal     bool
}

// diffAlbum executes the three-arm diff of one directory album -- the
// single-album worker's own work, shared with the all-albums sweep so
// the two can never disagree about an album's resolution.
func (d *albumDiffer) diffAlbum(ctx context.Context, albumID uuid.UUID, opts albumDiffOptions) (albumDiffOutcome, error) {
	jobRunID := opts.jobRunID
	queries := dbgen.New(d.pool)

	// The dispatch seam: no dispatcher wired means the run cannot
	// start any ingest's pipeline (only a directly constructed
	// worker gets here -- the queue wires the real one).
	if d.dispatch == nil {
		d.logger.Error("rescan run started without a pipeline dispatch wired",
			"job_run_id", jobRunID, "album_id", albumID)
		return albumDiffOutcome{fatal: true}, nil
	}

	// The subject: the album's owner (whose assets the arms write)
	// and its resolved root. A row that no longer answers (the album
	// stopped being a directory album -- nothing in v1 does this)
	// closes the run failed instead of walking a source the album
	// no longer claims.
	album, err := queries.GetRescanAlbum(ctx, pgUUID(albumID))
	if errors.Is(err, pgx.ErrNoRows) {
		d.logger.Error("rescan run found no directory album", "job_run_id", jobRunID, "album_id", albumID)
		return albumDiffOutcome{fatal: true}, nil
	}
	if err != nil {
		return albumDiffOutcome{}, fmt.Errorf("load the rescan album %s: %w", albumID, err)
	}
	owner := uuid.UUID(album.OwnerID.Bytes)
	root := album.RootPath.String

	// The root guard: the album row stores the fully RESOLVED path
	// (symlinks evaluated at creation), so a root that resolves to
	// itself is the real directory it always was. A root that now
	// resolves elsewhere -- replaced by a symlink after creation --
	// fails the run: walking the swapped target would ingest a tree
	// the album never claimed, and walking nothing would mark every
	// member vanished. Both are the owner's state to see, not the
	// scan's to perform.
	resolved, err := filepath.EvalSymlinks(root)
	if err != nil {
		d.logger.Error("the rescan source root cannot be resolved", "job_run_id", jobRunID,
			"album_id", albumID, "root", root, "error", err)
		return albumDiffOutcome{fatal: true}, nil
	}
	if resolved != root {
		d.logger.Error("the rescan source root was replaced by a symlink", "job_run_id", jobRunID,
			"album_id", albumID, "root", root, "resolves_to", resolved)
		return albumDiffOutcome{fatal: true}, nil
	}

	// The database half of the diff: every live external member,
	// keyed by the path the walk matches on.
	memberRows, err := queries.ListAlbumExternalMembers(ctx, pgUUID(albumID))
	if err != nil {
		return albumDiffOutcome{}, fmt.Errorf("list the members of the rescan album %s: %w", albumID, err)
	}
	members := make(map[string]rescanMember, len(memberRows))
	for _, row := range memberRows {
		member := rescanMember{
			id:   uuid.UUID(row.ID.Bytes),
			kind: row.MediaKind,
			size: row.FileSize,
		}
		if row.FileMtime.Valid {
			// The snapshot comparison runs at microsecond
			// precision: the column is timestamptz (microseconds in
			// PostgreSQL), and a filesystem mtime carrying
			// nanoseconds would compare unequal against its own
			// stored round-trip forever -- every scan would re-run
			// an unchanged file's pipeline. Both sides truncate.
			member.mtime = row.FileMtime.Time.Truncate(time.Microsecond)
			member.hasSnap = true
		}
		members[row.FilePath] = member
	}

	// The filesystem half: the walk. Symlinked entries are skipped
	// outright (files and directories alike -- the pin), and only
	// regular files whose extension sits in the media whitelist are
	// candidates -- the walk's README and .DS_Store are the source's
	// own noise, not work units of the diff (the sizing below and
	// the done count agree on the candidate set; a whitelisted file
	// whose bytes fail the sniff stays a unit and counts failed).
	// WalkDir visits in lexical order, so the resolution order is
	// deterministic and a retry's re-walk resolves the same units in
	// the same sequence. The walk's stamps are the folder sync's
	// comparison set as well, so the periodic diff and this scan can
	// never disagree about what the folder holds.
	files, err := walkSource(root)
	if err != nil {
		// An unreadable subtree (or a vanished root racing the
		// guard above) fails the run: the diff would be a lie about
		// half its inputs.
		d.logger.Error("the rescan walk failed", "job_run_id", jobRunID,
			"album_id", albumID, "root", root, "error", err)
		return albumDiffOutcome{fatal: true}, nil
	}
	// The folder sync's baseline, recorded at the walk's own moment
	// and BEFORE any arm moves: a file that arrives (or changes) while
	// the arms work is absent from this set, so the next tick sees it
	// as a difference and enqueues the scan that resolves it. The
	// conservative direction on purpose -- a redundant scan, never a
	// missed file.
	if d.snapshots != nil {
		d.snapshots.Store(albumID, stampsOf(files))
	}

	// The sizing write: the enqueue-time total was the database's
	// half; the walk completes it. Matched files resolve through the
	// file arms, unmatched members through the vanish arm -- every
	// unit counted exactly once. A total that already equals the
	// row's value skips the write: the idempotent re-run of an
	// unchanged directory writes nothing to the run row either
	// (beyond the run lifecycle's own transitions).
	matched := 0
	for _, file := range files {
		if _, ok := members[file.path]; ok {
			matched++
		}
	}
	total := len(files) + (len(members) - matched)
	if opts.sizeRun != nil && total != opts.currentTotal {
		if err := opts.sizeRun(ctx, total); err != nil {
			return albumDiffOutcome{}, fmt.Errorf("size the rescan run %s: %w", jobRunID, err)
		}
	}

	// The resolution. done counts resolved units (files and
	// vanishings alike, the no-op matches included -- a unit the
	// diff answered is work performed); failed counts the units the
	// arms could not resolve. liveSetChanged collects whether the
	// album's LIVE membership moved at all -- the add arm's growth or
	// the vanish arm's departure -- for the one album.items-changed
	// the run publishes; diffChanged collects whether the run wrote
	// anything at all (the album's scan clock stamps only then -- the
	// idempotence pin keeps an unchanged re-scan free of every
	// write).
	resolvedUnits, done, failed := 0, 0, 0
	liveSetChanged := false
	diffChanged := false
	progress := func() {
		if resolvedUnits%albumRescanProgressBatch != 0 {
			return
		}
		if opts.progress != nil {
			opts.progress(ctx, done, failed, total)
		}
	}

	// The cancel boundary (E41): the flag check rides the progress
	// batch's cadence -- the walk stops at the next batch boundary
	// and every block already committed keeps its writes. The check
	// forces itself on the FIRST call (a queued-then-cancelled run
	// must close at its pickup, not after another hundred units),
	// and a failed flag read never stops the scan on a guess: the
	// log carries it, the walk continues.
	unitsSinceStopCheck := albumRescanProgressBatch
	stopRequested := func() bool {
		unitsSinceStopCheck++
		if unitsSinceStopCheck <= albumRescanProgressBatch {
			return false
		}
		unitsSinceStopCheck = 0
		if opts.stopRequested == nil {
			return false
		}
		stop, err := opts.stopRequested(ctx)
		if err != nil {
			d.logger.Warn("check the rescan run's stop flag", "job_run_id", jobRunID, "error", err)
			return false
		}
		return stop
	}

	cancelledWalk := false
	for _, file := range files {
		if stopRequested() {
			cancelledWalk = true
			break
		}
		path := file.path
		resolvedUnits++
		info, err := os.Stat(path)
		if err != nil {
			failed++
			d.logger.Warn("the rescan could not stat a source file", "job_run_id", jobRunID,
				"album_id", albumID, "path", path, "error", err)
			progress()
			continue
		}
		mtime := info.ModTime().Truncate(time.Microsecond)

		if member, ok := members[path]; ok {
			delete(members, path)
			// The freshness comparison: mtime+size snapshot, the
			// checksum never pre-read. An equal snapshot is the
			// pinned no-op -- no write, no event, done.
			if member.hasSnap && member.mtime.Equal(mtime) && member.size == info.Size() {
				done++
				progress()
				continue
			}
			wrote, err := d.changeArm(ctx, queries, owner, member, path, mtime, info.Size())
			if err != nil {
				failed++
				d.logger.Warn("the rescan change arm failed", "job_run_id", jobRunID,
					"album_id", albumID, "asset_id", member.id, "path", path, "error", err)
			} else {
				done++
				if wrote {
					diffChanged = true
				}
			}
			progress()
			continue
		}

		added, err := d.addArm(ctx, queries, jobRunID, albumID, owner, path, mtime)
		if err != nil {
			failed++
			d.logger.Warn("the rescan add arm failed", "job_run_id", jobRunID,
				"album_id", albumID, "path", path, "error", err)
		} else {
			done++
			if added {
				liveSetChanged = true
				diffChanged = true
			}
		}
		progress()
	}
	// The vanish arm: every member left unmatched by the walk. The
	// map's iteration order is random; the arms are independent, but
	// the events ride a deterministic sequence so a retry replays
	// the same order (and the tests assert stable orderings). A
	// cancelled run skips the arm -- the scan stopped at its
	// boundary, and the next scan's vanish arm owns the rest.
	vanished := make([]string, 0, len(members))
	if !cancelledWalk {
		for path := range members {
			vanished = append(vanished, path)
		}
	}
	sort.Strings(vanished)
	for _, path := range vanished {
		resolvedUnits++
		member := members[path]
		rows, err := queries.MarkExternalAssetVanished(ctx, dbgen.MarkExternalAssetVanishedParams{
			ID:      pgUUID(member.id),
			OwnerID: pgUUID(owner),
		})
		if err != nil {
			failed++
			d.logger.Warn("the rescan vanish arm failed", "job_run_id", jobRunID,
				"album_id", albumID, "asset_id", member.id, "path", path, "error", err)
			progress()
			continue
		}
		done++
		if rows > 0 {
			// Only a state change announces itself: the retry of an
			// already-marked row (or a row a racing trash claimed)
			// matches nothing and stays silent -- the batch surface's
			// own event discipline, applied per arm.
			diffChanged = true
			liveSetChanged = true
			d.bus.Publish("asset.trashed", map[string]string{
				"assetId": member.id.String(),
				"kind":    string(member.kind),
			})
		}
		progress()
	}

	// The convergence, after the walk and the vanish arm: every live
	// external member of the album is by now a position row at a
	// walked path, so a same-content member that is NOT one of them --
	// a blob row the album borrowed under the pre-fork dedup, another
	// album's twin -- gives its membership up to this album's own
	// copy. The blob row keeps its sync membership; the directory
	// album stops mirroring a file it does not hold. One statement per
	// run, idempotent: a converged album matches nothing.
	// A cancelled run skips the convergence: the scan stopped at its
	// boundary, and the next scan's convergence arm owns the rest.
	if !cancelledWalk {
		converged, err := queries.ConvergeAlbumMembersToPositionRows(ctx, dbgen.ConvergeAlbumMembersToPositionRowsParams{
			AlbumID: pgUUID(albumID),
			OwnerID: pgUUID(owner),
		})
		if err != nil {
			return albumDiffOutcome{}, fmt.Errorf("converge the album's memberships onto its position rows: %w", err)
		}
		if converged > 0 {
			liveSetChanged = true
			diffChanged = true
			d.logger.Info("the rescan converged borrowed memberships onto the album's own copies",
				"job_run_id", jobRunID, "album_id", albumID, "converged", converged)
		}
	}

	// The album's story: one event when the album's LIVE membership
	// moved in either direction -- the add arm's growth (a row landed
	// or a live duplicate was linked) or the vanish arm's departure
	// (the row is marked, the membership row stays, and the live
	// views' deleted_at filter is what hides it). The vanish arm
	// speaks at the album level too, because the held-open surfaces
	// count MEMBERS, not rows: without this event an open album page
	// would lose the tile to asset.trashed while its count kept the
	// departed member until a manual reload. One event per run
	// either way -- the payload is the album id and its owner, never
	// state.
	if liveSetChanged {
		d.bus.Publish("album.items-changed", map[string]string{
			"albumId": albumID.String(),
			"userId":  owner.String(),
		})
	}

	// The scan's own clock: stamped only when the diff changed
	// something, so the unchanged re-scan writes nothing at all --
	// the idempotence pin, held even for the album row.
	if diffChanged {
		if err := queries.StampAlbumScanned(ctx, pgUUID(albumID)); err != nil {
			// The diff's writes already committed; the clock is
			// bookkeeping, so its failure logs and the run still
			// closes on its real counts.
			d.logger.Warn("stamp the album's scan clock", "job_run_id", jobRunID, "album_id", albumID, "error", err)
		}
	}

	return albumDiffOutcome{
		total:     total,
		done:      done,
		failed:    failed,
		cancelled: cancelledWalk,
	}, nil
}

// Work executes one rescan run: the row flips to running and
// job.started fires (echoing the enqueue-time total), the source
// root is walked (never following symlinks), the row's total is
// sized to the full diff, the three arms resolve every unit, and
// the run closes with the converged counts and job.finished --
// failed when any unit failed, succeeded otherwise. A vanished or
// repointed source root fails the run honestly: an empty walk
// would mark every member vanished, and a source the album no
// longer truthfully points at is a state for the owner to see, not
// a mass soft-delete to perform silently.
func (w *AlbumRescanWorker) Work(ctx context.Context, job *river.Job[AlbumRescanArgs]) error {
	jobRunID := job.Args.JobRunID
	albumID := job.Args.AlbumID
	queries := dbgen.New(w.pool)

	flagged, err := queries.MarkJobRunRunning(ctx, pgUUID(jobRunID))
	if err != nil {
		return fmt.Errorf("mark the rescan run %s running: %w", jobRunID, err)
	}
	if flagged == 0 {
		// Two ways the pickup matched nothing: the cancel closed the
		// run while it sat queued (E41 -- the row stays cancelled,
		// this driver never starts, no started event for a run that
		// ended before its first chunk), or this attempt IS a retry
		// resuming an interrupted run (the interrupted attempt left
		// the row running; the work proceeds below).
		run, err := queries.GetJobRun(ctx, pgUUID(jobRunID))
		if err != nil {
			return fmt.Errorf("load the rescan run %s: %w", jobRunID, err)
		}
		if run.State == dbgen.JobStateCancelled {
			return nil
		}
	}
	// The started event echoes the row's enqueue-time total -- the
	// durable truth the endpoint handed out -- not a recount; the
	// sizing write completes it once the walk listed the root.
	run, err := queries.GetJobRun(ctx, pgUUID(jobRunID))
	if err != nil {
		return fmt.Errorf("load the rescan run %s: %w", jobRunID, err)
	}
	w.bus.Publish("job.started", jobEventPayload(jobRunID, AlbumRescanKind, int(run.Total), 0, 0, dbgen.JobStateRunning))

	// The run's own progress record: the row write on every batch,
	// the event throttled onto the activity stream.
	lastEvent := time.Now()
	outcome, err := w.differ().diffAlbum(ctx, albumID, albumDiffOptions{
		jobRunID:     jobRunID,
		currentTotal: int(run.Total),
		sizeRun: func(ctx context.Context, total int) error {
			return dbgen.New(w.pool).SetJobRunTotal(ctx, dbgen.SetJobRunTotalParams{
				ID:    pgUUID(jobRunID),
				Total: int32(total),
			})
		},
		progress: func(ctx context.Context, done, failed, total int) {
			if err := dbgen.New(w.pool).UpdateJobRunProgress(ctx, dbgen.UpdateJobRunProgressParams{
				ID:   pgUUID(jobRunID),
				Done: int32(done),
			}); err != nil {
				w.logger.Warn("record the rescan progress", "job_run_id", jobRunID, "error", err)
			}
			if now := time.Now(); now.Sub(lastEvent) >= albumRescanProgressInterval {
				lastEvent = now
				w.bus.Publish("job.progress",
					jobEventPayload(jobRunID, AlbumRescanKind, total, done, failed, dbgen.JobStateRunning))
			}
		},
		stopRequested: func(ctx context.Context) (bool, error) {
			return queries.GetJobRunStopRequested(ctx, pgUUID(jobRunID))
		},
	})
	if err != nil {
		return err
	}
	if outcome.fatal {
		// The album itself cannot be diffed: the run closes failed on
		// the enqueue-time total, exactly as it announced it.
		return w.closeRescan(ctx, jobRunID, int(run.Total), 0, 0, dbgen.JobStateFailed)
	}

	state := dbgen.JobStateSucceeded
	if outcome.cancelled {
		state = dbgen.JobStateCancelled
	} else if outcome.failed > 0 {
		state = dbgen.JobStateFailed
	}
	// The final progress write lands the converged counts even when
	// the last batch was partial.
	if err := dbgen.New(w.pool).UpdateJobRunProgress(ctx, dbgen.UpdateJobRunProgressParams{
		ID:   pgUUID(jobRunID),
		Done: int32(outcome.done),
	}); err != nil {
		w.logger.Warn("record the rescan's final progress", "job_run_id", jobRunID, "error", err)
	}
	return w.closeRescan(ctx, jobRunID, outcome.total, outcome.done, outcome.failed, state)
}

// addArm ingests one walked file by reference: the format gates the
// pipeline's own tables apply (the extension whitelist -- already
// applied by the walk's candidate filter, kept here so the arm
// stays total for any caller -- then the magic bytes over the sniff
// window), the full read computes the checksum the row records, and
// the row, its exif placeholder and its album membership land in ONE
// transaction -- the upload path's own shape, minus the upload (the
// file is never staged, linked or moved: file_path IS the walked
// path, and the size and mtime the row stamps are the read's and the
// stat's own values). The content key never resolves an ingest here:
// the row's identity is its position, so content the owner uploaded
// (or another folder holds) lands its own row. answered true when
// the album's membership GREW (a new row landed, a live row at this
// position joined, or an orphan mark revived); the asset event fires
// for the rows that (re)entered the library, the link arm staying
// silent like every idempotent convergence.
func (d *albumDiffer) addArm(ctx context.Context, queries *dbgen.Queries, jobRunID, albumID, owner uuid.UUID, path string, mtime time.Time) (bool, error) {
	ext := extensionOf(path)
	format, known := storage.LookupUploadFormat(ext)
	if !known {
		// Unreachable through the worker's own walk (the candidate
		// filter applied the same whitelist); the guard keeps the
		// arm total for any directly constructed caller.
		d.logger.Debug("the rescan skips a non-media source file", "job_run_id", jobRunID, "path", path)
		return false, nil
	}

	checksum, size, head, err := storage.ReadExternalFile(path)
	if err != nil {
		return false, fmt.Errorf("read the source file: %w", err)
	}
	if !storage.FormatMatchesHead(ext, head) {
		// A whitelisted extension whose bytes do not match it is a
		// broken unit -- the failed count is the honest report, the
		// same rejection the upload receiver's Seal answers.
		return false, fmt.Errorf("the source file's content does not match its %q extension", ext)
	}

	kind := dbgen.MediaKindImage
	pendingJobs := int32(2)
	if format.Video {
		kind = dbgen.MediaKindVideo
		pendingJobs = 3
	}

	tx, err := d.pool.Begin(ctx)
	if err != nil {
		return false, fmt.Errorf("begin the ingest transaction: %w", err)
	}
	// The rollback is a no-op after a successful commit; the context
	// is the job's own, still live mid-Work.
	defer func() { _ = tx.Rollback(ctx) }()
	txQueries := dbgen.New(tx)

	assetID, err := uuid.NewV7()
	if err != nil {
		return false, fmt.Errorf("generate the external asset id: %w", err)
	}
	ingest := dbgen.CreateExternalAssetParams{
		ID:               pgUUID(assetID),
		OwnerID:          pgUUID(owner),
		FilePath:         path,
		Checksum:         checksum,
		OriginalFilename: filepath.Base(path),
		FileSize:         size,
		MimeType:         format.MIMEType,
		MediaKind:        kind,
		TakenAt:          pgtype.Timestamptz{Time: mtime, Valid: true},
		PendingJobs:      pendingJobs,
		FileMtime:        pgtype.Timestamptz{Time: mtime, Valid: true},
	}
	asset, err := txQueries.CreateExternalAsset(ctx, ingest)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == uniqueViolationCode &&
			pgErr.ConstraintName == externalPositionConstraint {
			// The position key at work: a row already holds this
			// path. The row never landed (the transaction rolls
			// back); the three-state resolution owns the answer.
			_ = tx.Rollback(ctx)
			return d.resolvePositionCollision(ctx, queries, albumID, owner, ingest, jobRunID)
		}
		return false, fmt.Errorf("insert the external asset: %w", err)
	}
	if err := txQueries.CreateExifPlaceholder(ctx, dbgen.CreateExifPlaceholderParams{
		AssetID: asset.ID,
		Exif:    []byte("{}"),
	}); err != nil {
		return false, fmt.Errorf("insert the external exif placeholder: %w", err)
	}
	if err := txQueries.AddAssetToAlbum(ctx, dbgen.AddAssetToAlbumParams{
		AlbumID: pgUUID(albumID),
		AssetID: asset.ID,
		AddedBy: pgUUID(owner),
	}); err != nil {
		return false, fmt.Errorf("add the external asset to the album: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return false, fmt.Errorf("commit the ingest transaction: %w", err)
	}

	// The pipeline starts only now, after the commit -- the jobs
	// read the rows the transaction wrote (the upload handler's own
	// discipline). The context is detached from the job's own so an
	// envelope expiring mid-run cannot cancel the enqueue of the
	// work the commit already made durable.
	if err := d.dispatch(context.WithoutCancel(ctx), assetID, kind); err != nil {
		return false, fmt.Errorf("enqueue the pipeline of the ingested asset: %w", err)
	}

	d.bus.Publish("asset.added", map[string]string{
		"assetId": assetID.String(),
		"kind":    string(kind),
	})
	return true, nil
}

// resolvePositionCollision resolves the add arm's position-key
// collision -- a row already occupies this path -- in the three
// states the position model pins:
//
//   - LIVE: the file is already ingested, by another album's source
//     (overlapping roots) or by a racing scan of this one. The album
//     links the row: membership grows, no second row.
//
//   - ORPHAN-MARKED (dead, no trash copy): the file died at this very
//     position once -- the vanish arm marked it -- and it is back, so
//     the row revives carrying the file's current identity (the walk
//     just read it) and rejoins the album. The metadata pipeline
//     re-runs, converging the derivations onto the new bytes.
//
//   - TRASHED WITH A COPY (dead, trash_path set): the row's bytes sit
//     in the server's trash area, and its file_path names that copy,
//     so a walked source path can never be this row's. Nothing to
//     resolve; the guard keeps the arm total.
//
// answered is the membership signal: a live link is a new tile, a
// revival is a row back in the library (the asset event fires), and
// the unreachable arm stays silent.
func (d *albumDiffer) resolvePositionCollision(ctx context.Context, queries *dbgen.Queries, albumID, owner uuid.UUID, ingest dbgen.CreateExternalAssetParams, jobRunID uuid.UUID) (bool, error) {
	path := ingest.FilePath
	row, err := queries.GetExternalAssetByOwnerFilePath(ctx, dbgen.GetExternalAssetByOwnerFilePathParams{
		OwnerID:  pgUUID(owner),
		FilePath: path,
	})
	if err != nil {
		return false, fmt.Errorf("resolve the position of %s: %w", path, err)
	}
	if !row.DeletedAt.Valid {
		if err := queries.AddAssetToAlbum(ctx, dbgen.AddAssetToAlbumParams{
			AlbumID: pgUUID(albumID),
			AssetID: row.ID,
			AddedBy: pgUUID(owner),
		}); err != nil {
			return false, fmt.Errorf("link the live row at %s into the album: %w", path, err)
		}
		return true, nil
	}
	if row.TrashPath.Valid {
		d.logger.Info("the rescan add arm matched a trashed copy's position; leaving it in the trash",
			"job_run_id", jobRunID, "asset_id", uuid.UUID(row.ID.Bytes), "path", path)
		return false, nil
	}
	revived, err := queries.ReviveExternalAsset(ctx, dbgen.ReviveExternalAssetParams{
		ID:               row.ID,
		OwnerID:          pgUUID(owner),
		Checksum:         ingest.Checksum,
		OriginalFilename: ingest.OriginalFilename,
		FileSize:         ingest.FileSize,
		MimeType:         ingest.MimeType,
		MediaKind:        ingest.MediaKind,
		FileMtime:        ingest.FileMtime,
	})
	if err != nil {
		return false, fmt.Errorf("revive the row at %s: %w", path, err)
	}
	if revived == 0 {
		// A racing scan revived it first; its own run owns the
		// event and the membership is already in place.
		return false, nil
	}
	if err := queries.AddAssetToAlbum(ctx, dbgen.AddAssetToAlbumParams{
		AlbumID: pgUUID(albumID),
		AssetID: row.ID,
		AddedBy: pgUUID(owner),
	}); err != nil {
		return false, fmt.Errorf("rejoin the revived row at %s to the album: %w", path, err)
	}
	if err := d.dispatch(context.WithoutCancel(ctx), uuid.UUID(row.ID.Bytes), ingest.MediaKind); err != nil {
		return false, fmt.Errorf("re-enqueue the pipeline of the revived asset: %w", err)
	}
	d.bus.Publish("asset.added", map[string]string{
		"assetId": uuid.UUID(row.ID.Bytes).String(),
		"kind":    string(ingest.MediaKind),
	})
	return true, nil
}

// changeArm re-runs one member's metadata pipeline: the full read
// the snapshot change earned re-lands the checksum (the content
// identity the serving ETags read -- a changed file must not answer
// requests with the old bytes' tag), the snapshot re-stamps, and
// the pipeline jobs re-enqueue through the same dispatch the add
// arm uses (the jobs' own idempotent re-run contract: converging
// upserts, overwritten derivations, the guarded readiness decrement
// matching nothing, no second asset.ready). The event is the
// family's metadata-change transition. A guard matching zero rows
// (the row vanished or trashed mid-run) resolves as a no-op, not a
// failure -- there is no row left to re-run -- and answers wrote
// false accordingly.
func (d *albumDiffer) changeArm(ctx context.Context, queries *dbgen.Queries, owner uuid.UUID, member rescanMember, path string, mtime time.Time, size int64) (bool, error) {
	checksum, _, _, err := storage.ReadExternalFile(path)
	if err != nil {
		return false, fmt.Errorf("re-read the changed source file: %w", err)
	}
	rows, err := queries.RefreshExternalSnapshot(ctx, dbgen.RefreshExternalSnapshotParams{
		ID:        pgUUID(member.id),
		Checksum:  checksum,
		FileSize:  size,
		FileMtime: pgtype.Timestamptz{Time: mtime, Valid: true},
		OwnerID:   pgUUID(owner),
	})
	if err != nil {
		return false, fmt.Errorf("refresh the snapshot of the changed asset: %w", err)
	}
	if rows == 0 {
		return false, nil
	}
	if err := d.dispatch(context.WithoutCancel(ctx), member.id, member.kind); err != nil {
		return true, fmt.Errorf("re-enqueue the pipeline of the changed asset: %w", err)
	}
	d.bus.Publish("asset.updated", map[string]string{
		"assetId": member.id.String(),
		"kind":    string(member.kind),
	})
	return true, nil
}

// closeRescan converges the row and announces the close, the places
// rerun's own closing discipline: the final bookkeeping runs
// detached from the job context so the row and the finished event
// land even when the run consumed the context's last moments; a
// failure here returns the error and River retries the job, whose
// re-walk re-converges the row.
func (w *AlbumRescanWorker) closeRescan(ctx context.Context, jobRunID uuid.UUID, total, done, failed int, state dbgen.JobState) error {
	finishCtx := context.WithoutCancel(ctx)
	if err := dbgen.New(w.pool).FinishJobRun(finishCtx, dbgen.FinishJobRunParams{
		ID:     pgUUID(jobRunID),
		State:  state,
		Done:   int32(done),
		Failed: int32(failed),
	}); err != nil {
		return fmt.Errorf("finish the rescan run %s: %w", jobRunID, err)
	}
	w.bus.Publish("job.finished", jobEventPayload(jobRunID, AlbumRescanKind, total, done, failed, state))
	return nil
}

// sourceFile is one walked candidate: the path the diff matches on,
// and the size+mtime stamp the folder sync's baseline keeps. The
// scan's arms re-stat the file at their own moment (the freshness
// comparison and the change arm read the filesystem NOW); the
// walk-time stamp is the diff's view of what the folder held when
// the listing happened.
type sourceFile struct {
	path  string
	size  int64
	mtime time.Time
}

// walkSource lists one source root's candidate files in the walk's
// lexical order: the scan's candidacy filter and the folder sync's
// snapshot in one pass, so the periodic diff and the scan's own walk
// can never disagree about which entries count. Symlinked entries are
// skipped outright (files and directories alike -- the pin), and only
// regular files whose extension sits in the media whitelist are
// candidates: a README or a .DS_Store in the source is the family's
// own noise, and its change must never wake a scan.
func walkSource(root string) ([]sourceFile, error) {
	files := make([]sourceFile, 0, 64)
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.Type()&fs.ModeSymlink != 0 {
			return nil
		}
		if !d.Type().IsRegular() {
			return nil
		}
		if _, known := storage.LookupUploadFormat(extensionOf(path)); !known {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		files = append(files, sourceFile{
			path:  path,
			size:  info.Size(),
			mtime: info.ModTime().Truncate(time.Microsecond),
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	return files, nil
}

// stampsOf projects the walk's candidates onto the folder sync's
// comparison set, keyed by path -- the identity both sides match on
// (the album row stores the resolved root and the walk follows no
// symlink, so plain path equality IS identity).
func stampsOf(files []sourceFile) map[string]fileStamp {
	stamps := make(map[string]fileStamp, len(files))
	for _, file := range files {
		stamps[file.path] = fileStamp{size: file.size, mtime: file.mtime}
	}
	return stamps
}

// extensionOf derives the canonical (lowercase, dotless) extension
// of one source file name -- the same normalization the upload
// receiver applies to the file part's name before the whitelist
// decides.
func extensionOf(name string) string {
	return strings.ToLower(strings.TrimPrefix(filepath.Ext(name), "."))
}

// dispatchRescan is the rescan dispatch seam's real wiring: the
// pipeline fan-out the upload handler performs for a committed
// upload, applied to the external row the rescan landed -- the
// metadata and thumbnail jobs every asset counts, plus the
// transcode job a video counts. The change arm re-runs the same
// set: a video's serving story is three jobs, and re-running two
// of three would leave the playable stream stale against the
// changed checksum the row now carries.
func (q *Queue) dispatchRescan(ctx context.Context, assetID uuid.UUID, kind dbgen.MediaKind) error {
	if _, err := q.client.Insert(ctx, MetadataArgs{AssetID: assetID}, nil); err != nil {
		return fmt.Errorf("insert the rescan's metadata job: %w", err)
	}
	if _, err := q.client.Insert(ctx, ThumbnailArgs{AssetID: assetID}, nil); err != nil {
		return fmt.Errorf("insert the rescan's thumbnail job: %w", err)
	}
	if kind == dbgen.MediaKindVideo {
		if _, err := q.client.Insert(ctx, TranscodeArgs{AssetID: assetID}, nil); err != nil {
			return fmt.Errorf("insert the rescan's transcode job: %w", err)
		}
	}
	return nil
}

// EnqueueAlbumRescan starts one rescan of the album: the job_runs
// row (the album_rescan kind, the album's live external member
// count as the enqueue-time total, the album as the run's subject)
// and the River job land in ONE transaction, so an enqueue that
// fails halfway leaves neither behind. The row starts queued; the
// worker flips it to running when the walk begins. The endpoint's
// in-flight guard (the album-scoped lookup over the kind) runs
// BEFORE the enqueue; a racing double-enqueue prowuces two
// convergent walks, not corruption -- every arm is idempotent.
func (q *Queue) EnqueueAlbumRescan(ctx context.Context, albumID uuid.UUID) (uuid.UUID, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return uuid.Nil, fmt.Errorf("generate the rescan run id: %w", err)
	}
	tx, err := q.pool.Begin(ctx)
	if err != nil {
		return uuid.Nil, fmt.Errorf("begin the rescan enqueue: %w", err)
	}
	// The rollback is a no-op after a successful commit; arguments
	// are captured while the request context is still live.
	defer func() { _ = tx.Rollback(ctx) }()

	// The scope's count, read inside the same transaction as the
	// insert: the total the row announces is exactly the half of
	// the diff the enqueue saw (the walk's half arrives with the
	// worker's sizing write).
	queries := dbgen.New(tx)
	total, err := queries.CountLiveAlbumExternalMembers(ctx, pgUUID(albumID))
	if err != nil {
		return uuid.Nil, fmt.Errorf("count the rescan scope: %w", err)
	}
	if err := queries.InsertAlbumRescanRun(ctx, dbgen.InsertAlbumRescanRunParams{
		ID:      pgUUID(id),
		Kind:    AlbumRescanKind,
		Total:   int32(total),
		AlbumID: pgUUID(albumID),
	}); err != nil {
		return uuid.Nil, fmt.Errorf("insert the rescan run row: %w", err)
	}
	if _, err := q.client.InsertTx(ctx, tx, AlbumRescanArgs{JobRunID: id, AlbumID: albumID}, nil); err != nil {
		return uuid.Nil, fmt.Errorf("insert the rescan job: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return uuid.Nil, fmt.Errorf("commit the rescan enqueue: %w", err)
	}
	return id, nil
}
