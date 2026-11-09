// The directory album rescan (E4 Task 2): the three-arm diff that
// reconciles a directory album's membership with what its source
// root holds on the filesystem. The endpoint (POST
// /albums/{id}/rescan) starts one run through EnqueueAlbumRescan;
// this worker executes it.
//
// The diff's shape is the plan's pinned table, one arm per
// resolution:
//
//   - ADD: a file no live member matches is ingested by reference
//     (a new external asset row pointing AT the file where it lies,
//     zero copy zero move, plus its album membership) through the
//     EXISTING ingest pipeline -- the same owner+checksum dedup the
//     upload path enforces (a live duplicate gains the membership of
//     the row the owner already holds; a trashed duplicate is left
//     alone: no revival semantics), and the same metadata /
//     thumbnail / transcode jobs an upload's transaction commits
//     enqueue. No parallel pipeline exists here: the format
//     whitelist and magic-byte sniffing are the storage package's
//     own tables, and the pipeline jobs read the external row's real
//     path through the kind-aware resolver.
//
//   - VANISH (the pinned orphan arm, 2026-11-08): a live member
//     whose file is gone from disk is marked deleted_at directly --
//     the file is not the server's and the trash has no meaning for
//     it. No missing state, no revival: the membership row stays
//     (soft delete never removed membership rows), and a file that
//     returns is ingested as a NEW row by the next scan's add arm.
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
// membership it already landed).
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
// and the River job kind of their driver, pinned by the milestone
// plan's vocabulary (the places rerun family's naming applied to the
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
// arm's insert resolves instead of failing: the owner+checksum
// uniqueness IS the ingest dedup, so a violation routes into the
// duplicate arms exactly as the upload path's conflict resolution
// routes its own.
const uniqueViolationCode = "23505"

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
	logger   *slog.Logger
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

	if err := queries.MarkJobRunRunning(ctx, pgUUID(jobRunID)); err != nil {
		return fmt.Errorf("mark the rescan run %s running: %w", jobRunID, err)
	}
	// The started event echoes the row's enqueue-time total -- the
	// durable truth the endpoint handed out -- not a recount; the
	// sizing write below completes it once the walk listed the root.
	run, err := queries.GetJobRun(ctx, pgUUID(jobRunID))
	if err != nil {
		return fmt.Errorf("load the rescan run %s: %w", jobRunID, err)
	}
	w.bus.Publish("job.started", jobEventPayload(jobRunID, AlbumRescanKind, int(run.Total), 0, 0, dbgen.JobStateRunning))

	// The dispatch seam: no dispatcher wired means the run cannot
	// start any ingest's pipeline (only a directly constructed
	// worker gets here -- the queue wires the real one).
	if w.dispatch == nil {
		w.logger.Error("rescan run started without a pipeline dispatch wired",
			"job_run_id", jobRunID, "album_id", albumID)
		return w.closeRescan(ctx, jobRunID, int(run.Total), 0, 0, dbgen.JobStateFailed)
	}

	// The subject: the album's owner (whose assets the arms write)
	// and its resolved root. A row that no longer answers (the album
	// stopped being a directory album -- nothing in v1 does this)
	// closes the run failed instead of walking a source the album
	// no longer claims.
	album, err := queries.GetRescanAlbum(ctx, pgUUID(albumID))
	if errors.Is(err, pgx.ErrNoRows) {
		w.logger.Error("rescan run found no directory album", "job_run_id", jobRunID, "album_id", albumID)
		return w.closeRescan(ctx, jobRunID, int(run.Total), 0, 0, dbgen.JobStateFailed)
	}
	if err != nil {
		return fmt.Errorf("load the rescan album %s: %w", albumID, err)
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
		w.logger.Error("the rescan source root cannot be resolved", "job_run_id", jobRunID,
			"album_id", albumID, "root", root, "error", err)
		return w.closeRescan(ctx, jobRunID, int(run.Total), 0, 0, dbgen.JobStateFailed)
	}
	if resolved != root {
		w.logger.Error("the rescan source root was replaced by a symlink", "job_run_id", jobRunID,
			"album_id", albumID, "root", root, "resolves_to", resolved)
		return w.closeRescan(ctx, jobRunID, int(run.Total), 0, 0, dbgen.JobStateFailed)
	}

	// The database half of the diff: every live external member,
	// keyed by the path the walk matches on.
	memberRows, err := queries.ListAlbumExternalMembers(ctx, pgUUID(albumID))
	if err != nil {
		return fmt.Errorf("list the members of the rescan album %s: %w", albumID, err)
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
	// the same sequence.
	files := make([]string, 0, len(members))
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
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
		files = append(files, path)
		return nil
	})
	if err != nil {
		// An unreadable subtree (or a vanished root racing the
		// guard above) fails the run: the diff would be a lie about
		// half its inputs.
		w.logger.Error("the rescan walk failed", "job_run_id", jobRunID,
			"album_id", albumID, "root", root, "error", err)
		return w.closeRescan(ctx, jobRunID, int(run.Total), 0, 0, dbgen.JobStateFailed)
	}

	// The sizing write: the enqueue-time total was the database's
	// half; the walk completes it. Matched files resolve through the
	// file arms, unmatched members through the vanish arm -- every
	// unit counted exactly once. A total that already equals the
	// row's value skips the write: the idempotent re-run of an
	// unchanged directory writes nothing to the run row either
	// (beyond the run lifecycle's own transitions).
	matched := 0
	for _, path := range files {
		if _, ok := members[path]; ok {
			matched++
		}
	}
	total := len(files) + (len(members) - matched)
	if total != int(run.Total) {
		if err := queries.SetJobRunTotal(ctx, dbgen.SetJobRunTotalParams{
			ID:    pgUUID(jobRunID),
			Total: int32(total),
		}); err != nil {
			return fmt.Errorf("size the rescan run %s: %w", jobRunID, err)
		}
	}

	// The resolution. done counts resolved units (files and
	// vanishings alike, the no-op matches included -- a unit the
	// diff answered is work performed); failed counts the units the
	// arms could not resolve. membershipGrown collects the add arm's
	// story for the one album.items-changed the run publishes if
	// any membership row landed; diffChanged collects whether the
	// run wrote anything at all (the album's scan clock stamps only
	// then -- the idempotence pin keeps an unchanged re-scan free
	// of every write).
	resolvedUnits, done, failed := 0, 0, 0
	lastEvent := time.Now()
	membershipGrown := false
	diffChanged := false
	progress := func() {
		if resolvedUnits%albumRescanProgressBatch != 0 {
			return
		}
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
	}

	for _, path := range files {
		resolvedUnits++
		info, err := os.Stat(path)
		if err != nil {
			failed++
			w.logger.Warn("the rescan could not stat a source file", "job_run_id", jobRunID,
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
			wrote, err := w.changeArm(ctx, queries, owner, member, path, mtime, info.Size())
			if err != nil {
				failed++
				w.logger.Warn("the rescan change arm failed", "job_run_id", jobRunID,
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

		added, err := w.addArm(ctx, queries, jobRunID, albumID, owner, path, mtime)
		if err != nil {
			failed++
			w.logger.Warn("the rescan add arm failed", "job_run_id", jobRunID,
				"album_id", albumID, "path", path, "error", err)
		} else {
			done++
			if added {
				membershipGrown = true
				diffChanged = true
			}
		}
		progress()
	}
	// The vanish arm: every member left unmatched by the walk. The
	// map's iteration order is random; the arms are independent, but
	// the events ride a deterministic sequence so a retry replays
	// the same order (and the tests assert stable orderings).
	vanished := make([]string, 0, len(members))
	for path := range members {
		vanished = append(vanished, path)
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
			w.logger.Warn("the rescan vanish arm failed", "job_run_id", jobRunID,
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
			w.bus.Publish("asset.trashed", map[string]string{
				"assetId": member.id.String(),
				"kind":    string(member.kind),
			})
		}
		progress()
	}

	// The album's story: one event when the add arm grew the
	// membership (the membership write's own predicate, widened to
	// both directions of the write). The vanish arm published
	// nothing album-level -- the membership rows never changed, the
	// live views' deleted_at filter is the whole story.
	if membershipGrown {
		w.bus.Publish("album.items-changed", map[string]string{
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
			w.logger.Warn("stamp the album's scan clock", "job_run_id", jobRunID, "album_id", albumID, "error", err)
		}
	}

	state := dbgen.JobStateSucceeded
	if failed > 0 {
		state = dbgen.JobStateFailed
	}
	// The final progress write lands the converged counts even when
	// the last batch was partial.
	if err := dbgen.New(w.pool).UpdateJobRunProgress(ctx, dbgen.UpdateJobRunProgressParams{
		ID:   pgUUID(jobRunID),
		Done: int32(done),
	}); err != nil {
		w.logger.Warn("record the rescan's final progress", "job_run_id", jobRunID, "error", err)
	}
	return w.closeRescan(ctx, jobRunID, total, done, failed, state)
}

// addArm ingests one walked file by reference: the format gates the
// pipeline's own tables apply (the extension whitelist -- already
// applied by the walk's candidate filter, kept here so the arm
// stays total for any caller -- then the magic bytes over the sniff
// window), the full read computes the checksum the owner+checksum
// dedup resolves on, and the row, its exif placeholder and its
// album membership land in ONE transaction -- the upload path's own
// shape, minus the upload (the file is never staged, linked or
// moved: file_path IS the walked path, and the size and mtime the
// row stamps are the read's and the stat's own values). answered
// true when the album's membership GREW (a new row landed, or a
// live duplicate of the content was linked in); the asset event
// fires only for the new row, the duplicate arms staying silent
// like every idempotent convergence.
func (w *AlbumRescanWorker) addArm(ctx context.Context, queries *dbgen.Queries, jobRunID, albumID, owner uuid.UUID, path string, mtime time.Time) (bool, error) {
	ext := extensionOf(path)
	format, known := storage.LookupUploadFormat(ext)
	if !known {
		// Unreachable through the worker's own walk (the candidate
		// filter applied the same whitelist); the guard keeps the
		// arm total for any directly constructed caller.
		w.logger.Debug("the rescan skips a non-media source file", "job_run_id", jobRunID, "path", path)
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

	tx, err := w.pool.Begin(ctx)
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
	asset, err := txQueries.CreateExternalAsset(ctx, dbgen.CreateExternalAssetParams{
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
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == uniqueViolationCode {
			// The owner+checksum dedup at work: the walked content
			// is already the owner's. The row never landed (the
			// transaction rolls back); the duplicate resolves
			// through the same lookup the upload path's conflict
			// resolution uses.
			return w.linkDuplicate(ctx, queries, albumID, owner, checksum, path, jobRunID)
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
	if err := w.dispatch(context.WithoutCancel(ctx), assetID, kind); err != nil {
		return false, fmt.Errorf("enqueue the pipeline of the ingested asset: %w", err)
	}

	w.bus.Publish("asset.added", map[string]string{
		"assetId": assetID.String(),
		"kind":    string(kind),
	})
	return true, nil
}

// linkDuplicate resolves the add arm's owner+checksum collision:
// the content is the owner's already. A LIVE row gains the album
// membership of the existing row (the album's wall grows, no new
// row -- the uniqueness IS the server's content identity); a
// TRASHED row is left alone, the no-revival pin read from this
// side: the next scan of a restored duplicate, or the restore
// itself, re-links it. answered is the membership signal: a live
// duplicate's link grew the album's story without a new row.
func (w *AlbumRescanWorker) linkDuplicate(ctx context.Context, queries *dbgen.Queries, albumID, owner uuid.UUID, checksum, path string, jobRunID uuid.UUID) (bool, error) {
	row, err := queries.GetAssetByOwnerChecksum(ctx, dbgen.GetAssetByOwnerChecksumParams{
		OwnerID:  pgUUID(owner),
		Checksum: checksum,
	})
	if err != nil {
		return false, fmt.Errorf("resolve the duplicate of %s: %w", path, err)
	}
	if row.DeletedAt.Valid {
		w.logger.Info("the rescan add arm matched trashed content; leaving it in the trash",
			"job_run_id", jobRunID, "asset_id", uuid.UUID(row.ID.Bytes), "path", path)
		return false, nil
	}
	if err := queries.AddAssetToAlbum(ctx, dbgen.AddAssetToAlbumParams{
		AlbumID: pgUUID(albumID),
		AssetID: row.ID,
		AddedBy: pgUUID(owner),
	}); err != nil {
		return false, fmt.Errorf("link the live duplicate into the album: %w", err)
	}
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
func (w *AlbumRescanWorker) changeArm(ctx context.Context, queries *dbgen.Queries, owner uuid.UUID, member rescanMember, path string, mtime time.Time, size int64) (bool, error) {
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
	if err := w.dispatch(context.WithoutCancel(ctx), member.id, member.kind); err != nil {
		return true, fmt.Errorf("re-enqueue the pipeline of the changed asset: %w", err)
	}
	w.bus.Publish("asset.updated", map[string]string{
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
// BEFORE the enqueue; a racing double-enqueue produces two
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
