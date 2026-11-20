// The all-albums rescan sweep: the task catalog's "rescan all". One
// run walks every directory album of the instance through the SAME
// three-arm diff the per-album rescan performs (diffAlbum, shared
// with the single-album worker), album by album.
//
// THE ONE ROW: the sweep records ONE job_runs row of the
// album_rescan_all kind. It creates no per-album rows -- the history
// shows "Album rescan (all)" with the album count as its total,
// while a rescan started on a directory page keeps its own row. The
// run's units are ALBUMS: done counts the albums whose diff closed
// cleanly, failed the albums that could not be diffed at all (a
// vanished row, a root that stopped resolving, an unreadable walk)
// or whose diff reported failed entries.
//
// THE BOUNDARIES: one album's failure never stops the sweep, and the
// cancel flag is honored at album boundaries -- the shared chunk
// protocol -- with the committed albums' work kept and the run
// closing cancelled. The diff itself also checks the flag at its
// unit-batch cadence, so a long album stops at its own boundary
// rather than only after finishing.
//
// The sweep's enqueue counts the instance's directory albums inside
// the same transaction that inserts the row and the job (the
// enqueue's own shape everywhere); the walk re-lists them, so an
// album created between the two sizes the row to the walk's own
// listing.
package jobs

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"

	dbgen "github.com/welpher/lumemor/server/internal/db/gen"
	"github.com/welpher/lumemor/server/internal/events"
)

// AlbumRescanAllKind is the job_runs kind of the instance-wide
// rescan sweeps and the River job kind of their driver. It is a
// distinct kind on purpose: the sweep's in-flight join is
// instance-wide (the single-album runs join per album), and the
// history row names the sweep rather than one album.
const AlbumRescanAllKind = "album_rescan_all"

// albumRescanAllTimeout raises the worker's execution envelope well
// above one album's own 30 minutes: the sweep is one album after
// another, and an expired attempt hands the job to the retry, whose
// re-walk is a no-op over every album already converged (each diff
// is the same deterministic function of the same filesystem and
// database state).
const albumRescanAllTimeout = 6 * time.Hour

// AlbumRescanAllArgs identifies the sweep run the worker walks.
type AlbumRescanAllArgs struct {
	JobRunID uuid.UUID `json:"jobRunId"`
}

// Kind is River's stable job kind identifier.
func (AlbumRescanAllArgs) Kind() string { return AlbumRescanAllKind }

// AlbumRescanAllWorker runs one instance-wide rescan sweep.
type AlbumRescanAllWorker struct {
	river.WorkerDefaults[AlbumRescanAllArgs]

	pool *pgxpool.Pool
	bus  *events.Bus
	// dispatch is the ingest seam, shared with the single-album
	// worker: every album's diff ingests through the same pipeline
	// fan-out. Nil fails the run rather than ingesting assets whose
	// pipeline never starts.
	dispatch  RescanDispatchFunc
	snapshots *AlbumSnapshots
	logger    *slog.Logger
}

// Timeout is the worker's execution envelope; see
// albumRescanAllTimeout.
func (w *AlbumRescanAllWorker) Timeout(job *river.Job[AlbumRescanAllArgs]) time.Duration {
	return albumRescanAllTimeout
}

// differ projects the sweep's dependencies for the shared per-album
// diff -- the same projection the single-album worker makes.
func (w *AlbumRescanAllWorker) differ() *albumDiffer {
	return &albumDiffer{
		pool:      w.pool,
		bus:       w.bus,
		dispatch:  w.dispatch,
		snapshots: w.snapshots,
		logger:    w.logger,
	}
}

// Work executes one sweep: the row flips to running and job.started
// fires (echoing the enqueue-time album count), the instance's
// directory albums are listed and walked one by one through the
// shared diff, the counts converge after every album, and the run
// closes with job.finished -- cancelled when the stop flag ended it,
// failed when any album failed, succeeded otherwise.
func (w *AlbumRescanAllWorker) Work(ctx context.Context, job *river.Job[AlbumRescanAllArgs]) error {
	jobRunID := job.Args.JobRunID
	queries := dbgen.New(w.pool)

	flagged, err := queries.MarkJobRunRunning(ctx, pgUUID(jobRunID))
	if err != nil {
		return fmt.Errorf("mark the album rescan sweep %s running: %w", jobRunID, err)
	}
	if flagged == 0 {
		// The cancel closed the run while it sat queued, or this
		// attempt is a retry resuming an interrupted run (the row
		// still reads running and the walk proceeds).
		run, err := queries.GetJobRun(ctx, pgUUID(jobRunID))
		if err != nil {
			return fmt.Errorf("load the album rescan sweep %s: %w", jobRunID, err)
		}
		if run.State == dbgen.JobStateCancelled {
			return nil
		}
	}
	run, err := queries.GetJobRun(ctx, pgUUID(jobRunID))
	if err != nil {
		return fmt.Errorf("load the album rescan sweep %s: %w", jobRunID, err)
	}
	w.bus.Publish("job.started", jobEventPayload(jobRunID, AlbumRescanAllKind, int(run.Total), 0, 0, dbgen.JobStateRunning))

	if w.dispatch == nil {
		w.logger.Error("album rescan sweep started without a pipeline dispatch wired", "job_run_id", jobRunID)
		return w.closeSweep(ctx, jobRunID, int(run.Total), 0, 0, dbgen.JobStateFailed)
	}

	// The walk's own listing: the sweep's scope is every directory
	// album with a resolved root, the same set the folder sync
	// follows. An album created since the enqueue joins the sweep;
	// the row is sized to the listing so the counts converge.
	albums, err := queries.ListDirectoryAlbums(ctx)
	if err != nil {
		return fmt.Errorf("list the albums of the album rescan sweep %s: %w", jobRunID, err)
	}
	total := len(albums)
	if total != int(run.Total) {
		if err := queries.SetJobRunTotal(ctx, dbgen.SetJobRunTotalParams{
			ID:    pgUUID(jobRunID),
			Total: int32(total),
		}); err != nil {
			return fmt.Errorf("size the album rescan sweep %s: %w", jobRunID, err)
		}
	}

	// The album boundary (E41): the flag is read before each album,
	// so a cancel that lands mid-album ends the sweep at the album's
	// own end and the committed albums keep their writes.
	done, failed := 0, 0
	cancelled := false
	lastEvent := time.Now()
	for _, album := range albums {
		stop, err := queries.GetJobRunStopRequested(ctx, pgUUID(jobRunID))
		if err != nil {
			return fmt.Errorf("check the stop flag of the album rescan sweep %s: %w", jobRunID, err)
		}
		if stop {
			cancelled = true
			break
		}
		albumID := uuid.UUID(album.ID.Bytes)
		outcome, err := w.differ().diffAlbum(ctx, albumID, albumDiffOptions{
			jobRunID: jobRunID,
			// The sweep's total counts albums, so the diff never
			// sizes the row to its own units -- and its per-unit
			// progress stays off: the sweep records albums.
			stopRequested: func(ctx context.Context) (bool, error) {
				return queries.GetJobRunStopRequested(ctx, pgUUID(jobRunID))
			},
		})
		switch {
		case err != nil:
			// One album's failure never stops the sweep: the album
			// counts failed and the next one is walked.
			failed++
			w.logger.Warn("the album rescan sweep failed an album", "job_run_id", jobRunID,
				"album_id", albumID, "error", err)
		case outcome.cancelled:
			// The diff stopped at its own unit boundary: the flag
			// ended the sweep here, the album keeps its committed
			// work and does not count done.
			cancelled = true
		case outcome.fatal || outcome.failed > 0:
			failed++
		default:
			done++
		}

		// The row's album-level progress after every album, the
		// event throttled onto the activity stream.
		if err := queries.UpdateJobRunProgress(ctx, dbgen.UpdateJobRunProgressParams{
			ID:   pgUUID(jobRunID),
			Done: int32(done),
		}); err != nil {
			w.logger.Warn("record the album rescan sweep progress", "job_run_id", jobRunID, "error", err)
		}
		if now := time.Now(); now.Sub(lastEvent) >= albumRescanProgressInterval {
			lastEvent = now
			w.bus.Publish("job.progress",
				jobEventPayload(jobRunID, AlbumRescanAllKind, total, done, failed, dbgen.JobStateRunning))
		}
		if cancelled {
			break
		}
	}

	state := dbgen.JobStateSucceeded
	if cancelled {
		// The admin's own deliberate ending: the committed albums
		// keep their work, the run reads cancelled.
		state = dbgen.JobStateCancelled
	} else if failed > 0 {
		state = dbgen.JobStateFailed
	}
	return w.closeSweep(ctx, jobRunID, total, done, failed, state)
}

// closeSweep converges the row and announces the close, the drivers'
// shared closing discipline: the final bookkeeping runs detached from
// the job context so the row and the finished event land even when
// the run consumed the context's last moments.
func (w *AlbumRescanAllWorker) closeSweep(ctx context.Context, jobRunID uuid.UUID, total, done, failed int, state dbgen.JobState) error {
	finishCtx := context.WithoutCancel(ctx)
	if err := dbgen.New(w.pool).FinishJobRun(finishCtx, dbgen.FinishJobRunParams{
		ID:     pgUUID(jobRunID),
		State:  state,
		Done:   int32(done),
		Failed: int32(failed),
	}); err != nil {
		return fmt.Errorf("finish the album rescan sweep %s: %w", jobRunID, err)
	}
	w.bus.Publish("job.finished", jobEventPayload(jobRunID, AlbumRescanAllKind, total, done, failed, state))
	return nil
}

// EnqueueAlbumRescanAll starts one instance-wide rescan sweep: the
// job_runs row (the album_rescan_all kind, the instance's directory
// album count as the enqueue-time total) and the River job land in
// ONE transaction. The endpoint's in-flight guard runs BEFORE this
// call; a racing double-enqueue produces two convergent sweeps, not
// corruption -- every arm is idempotent.
func (q *Queue) EnqueueAlbumRescanAll(ctx context.Context) (uuid.UUID, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return uuid.Nil, fmt.Errorf("generate the album rescan sweep id: %w", err)
	}
	tx, err := q.pool.Begin(ctx)
	if err != nil {
		return uuid.Nil, fmt.Errorf("begin the album rescan sweep enqueue: %w", err)
	}
	// The rollback is a no-op after a successful commit; arguments
	// are captured while the request context is still live.
	defer func() { _ = tx.Rollback(ctx) }()

	queries := dbgen.New(tx)
	total, err := queries.CountDirectoryAlbums(ctx)
	if err != nil {
		return uuid.Nil, fmt.Errorf("count the album rescan sweep's scope: %w", err)
	}
	if err := queries.InsertJobRun(ctx, dbgen.InsertJobRunParams{
		ID:    pgUUID(id),
		Kind:  AlbumRescanAllKind,
		Total: int32(total),
	}); err != nil {
		return uuid.Nil, fmt.Errorf("insert the album rescan sweep row: %w", err)
	}
	if _, err := q.client.InsertTx(ctx, tx, AlbumRescanAllArgs{JobRunID: id}, nil); err != nil {
		return uuid.Nil, fmt.Errorf("insert the album rescan sweep job: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return uuid.Nil, fmt.Errorf("commit the album rescan sweep enqueue: %w", err)
	}
	return id, nil
}
