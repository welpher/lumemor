// The places re-resolution run: the places-data-update
// path. The startup import of a changed GeoNames bake replaces the
// reference tables and -- through the schema's ON DELETE SET NULL --
// releases every stale resolution; this run is the deliberate step
// that refills them: one walk over the mode's scope (every GPS-bearing
// asset row, or only the rows whose place is absent), resolving each
// against the process's in-memory grid (the same grid the metadata
// job's GPS step uses, loaded once per process) and writing the answer
// -- the nearest city within the threshold, or NULL when the grid
// answers nothing.
//
// Unlike the ML rerun driver there is no per-asset dispatch: place
// resolution is a microsecond in-memory query plus one bounded row
// write, so the walk itself IS the work. The run's job_runs row
// carries the protocol the task panel and the activity stream read
// (started, throttled progress, finished; done/failed counting
// assets), and an interruption (the envelope expiring, the process
// dying) leaves the row running and the work resumable: River retries
// the job and the re-walk converges -- every write is the same
// deterministic function of the same GPS coordinates, so re-running
// rows is a no-op and the counts re-climb from zero.
//
// The enqueue refuses a queue whose process runs without the place
// reference data: there is no grid to re-resolve against, and a run
// that would null every resolution is not a run worth recording.
package jobs

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"

	dbgen "github.com/welpher/lumemor/server/internal/db/gen"
	"github.com/welpher/lumemor/server/internal/events"
	"github.com/welpher/lumemor/server/internal/geodata"
)

// PlacesRerunKind is the job_runs kind of the place re-resolution
// runs, pinned by the shared vocabulary (the rerun family's
// naming, applied to the places domain). The River job kind carries
// the same string: one driver, no per-asset dispatch.
const PlacesRerunKind = "places_rerun"

// The re-resolution modes, the contract's enum: the fill-in pair the
// map & places card's dialog chooses between. Missing walks only the
// GPS-bearing rows whose place is absent; all walks every GPS-bearing
// row (the full re-resolve that may rewrite history).
const (
	PlacesRerunModeMissing = "missing"
	PlacesRerunModeAll     = "all"
)

// placesRerunTimeout raises the worker's execution envelope above
// River's one minute default: the walk over a large library's
// GPS-bearing rows (one bounded page read per 500 assets plus one
// row write per asset) legitimately outlasts a small envelope, and
// an expired attempt hands the job to the retry, whose re-walk
// converges over the same deterministic answers.
const placesRerunTimeout = 30 * time.Minute

// placesRerunPageSize bounds one page of the walk: the scope's asset
// ids and coordinates ride in memory one page at a time, so a rerun
// over a large library never holds the whole set at once.
const placesRerunPageSize = 500

// placesRerunProgressInterval paces the run's progress events onto
// the activity stream: at most one per interval, read off the walk's
// own clock. The job_runs row itself updates on every page (a cheap
// bounded write), so the panel's truth stays page-fresh.
const placesRerunProgressInterval = time.Second

// PlacesRerunArgs identifies the run the worker walks; the mode names
// which scope it walks (the contract's enum). The rows themselves
// need no payload: the walk reads them from the database, so a retry
// after new GPS-bearing assets arriving simply includes them.
type PlacesRerunArgs struct {
	JobRunID uuid.UUID `json:"jobRunId"`
	Mode     string    `json:"mode"`
}

// Kind is River's stable job kind identifier.
func (PlacesRerunArgs) Kind() string { return PlacesRerunKind }

// PlacesRerunWorker runs the place re-resolution walk.
type PlacesRerunWorker struct {
	river.WorkerDefaults[PlacesRerunArgs]

	pool   *pgxpool.Pool
	bus    *events.Bus
	places *geodata.Index
	logger *slog.Logger
	// budget is the background work's gate. Nil gates nothing (the
	// directly constructed workers of the unit tests).
	budget *Budget
}

// Timeout is the worker's execution envelope; see placesRerunTimeout.
func (w *PlacesRerunWorker) Timeout(job *river.Job[PlacesRerunArgs]) time.Duration {
	return placesRerunTimeout
}

// Work executes one re-resolution run: the row flips to running and
// job.started fires, the walk resolves every GPS-bearing asset page
// by page against the grid (each write the deterministic function of
// the row's coordinates -- the nearest city within the threshold,
// else NULL), the throttled progress events ride the pages, and the
// run closes with the converged counts and job.finished -- failed
// when any write failed, succeeded otherwise.
func (w *PlacesRerunWorker) Work(ctx context.Context, job *river.Job[PlacesRerunArgs]) error {
	// The budget gate: a paused places walk snoozes without consuming
	// an attempt. The walk itself is a microsecond per asset, so the
	// gate reads once at the run's start -- a pause mid-run lets the
	// started run finish, the honest "the one already running
	// finishes". Nil budgets gate nothing.
	if w.budget != nil {
		if hold := w.budget.Hold(ctx, classPlaces); hold > 0 {
			return river.JobSnooze(hold)
		}
	}

	jobRunID := job.Args.JobRunID
	queries := dbgen.New(w.pool)

	flagged, err := queries.MarkJobRunRunning(ctx, pgUUID(jobRunID))
	if err != nil {
		return fmt.Errorf("mark the places rerun run %s running: %w", jobRunID, err)
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
			return fmt.Errorf("load the places rerun run %s: %w", jobRunID, err)
		}
		if run.State == dbgen.JobStateCancelled {
			return nil
		}
	}
	// The started event echoes the row's enqueue-time total -- the
	// durable truth the endpoint handed out -- not a recount.
	run, err := queries.GetJobRun(ctx, pgUUID(jobRunID))
	if err != nil {
		return fmt.Errorf("load the places rerun run %s: %w", jobRunID, err)
	}
	total := int(run.Total)
	w.bus.Publish("job.started", jobEventPayload(jobRunID, PlacesRerunKind, total, 0, 0, dbgen.JobStateRunning))

	// The mode: the contract's enum. An absent mode is the contract's
	// own default -- the full walk, which is also the scope a run
	// queued by an older build's payload was counted against, so a
	// durable queue survives the field's arrival. A non-empty value
	// outside the enum is a wiring bug; the run closes failed instead
	// of pretending to have resolved anything.
	mode := job.Args.Mode
	if mode == "" {
		mode = PlacesRerunModeAll
	}
	if mode != PlacesRerunModeMissing && mode != PlacesRerunModeAll {
		w.logger.Error("places rerun run started with a mode outside the enum",
			"job_run_id", jobRunID, "mode", job.Args.Mode)
		return w.finishRerun(ctx, jobRunID, total, 0, 0, dbgen.JobStateFailed)
	}

	// The resolution seam: no grid wired means the run cannot do its
	// work. The enqueue refuses this state; only a directly
	// constructed worker (a wiring bug) gets here, and the run closes
	// failed instead of pretending to have resolved anything.
	if w.places == nil {
		w.logger.Error("places rerun run started without the place reference data",
			"job_run_id", jobRunID, "total", total)
		return w.finishRerun(ctx, jobRunID, total, 0, 0, dbgen.JobStateFailed)
	}

	done, failed := 0, 0
	lastEvent := time.Now()
	cursor := pgUUID(uuid.Nil) // the zero value starts the walk
	cancelled := false
	for {
		// The cancel boundary (E41): the flag check rides the page
		// edge -- the walk stops at the next page boundary, the rows
		// already written keep their answers. The first call reads
		// too: a queued-then-cancelled run closes at its pickup.
		stop, err := queries.GetJobRunStopRequested(ctx, pgUUID(jobRunID))
		if err != nil {
			return fmt.Errorf("check the stop flag of places rerun run %s: %w", jobRunID, err)
		}
		if stop {
			cancelled = true
			w.logger.Info("places rerun run cancelled at its own page boundary", "job_run_id", jobRunID, "done", done)
			break
		}
		rows, err := listPlacesRerunPage(ctx, queries, mode, cursor)
		if err != nil {
			return fmt.Errorf("list the places rerun scope of run %s: %w", jobRunID, err)
		}
		if len(rows) == 0 {
			break
		}
		for _, row := range rows {
			cursor = row.id
			assetID := uuid.UUID(row.id.Bytes)
			// The same deterministic resolution the metadata job's
			// GPS step applies: the grid's nearest city within the
			// threshold, else NULL (no country-polygon fallback).
			placeID := pgtype.Int4{}
			if id, _, ok := w.places.NearestWithin(row.lat.Float64, row.lon.Float64, geodata.ResolutionRadiusKm); ok {
				placeID.Int32, placeID.Valid = id, true
			}
			if err := queries.SetAssetPlace(ctx, dbgen.SetAssetPlaceParams{
				ID:      row.id,
				PlaceID: placeID,
			}); err != nil {
				failed++
				w.logger.Warn("places rerun write failed", "job_run_id", jobRunID,
					"asset_id", assetID, "error", err)
				continue
			}
			done++
		}
		if err := queries.UpdateJobRunProgress(ctx, dbgen.UpdateJobRunProgressParams{
			ID:   pgUUID(jobRunID),
			Done: int32(done),
		}); err != nil {
			return fmt.Errorf("record the places rerun progress of run %s: %w", jobRunID, err)
		}
		if now := time.Now(); now.Sub(lastEvent) >= placesRerunProgressInterval {
			lastEvent = now
			w.bus.Publish("job.progress",
				jobEventPayload(jobRunID, PlacesRerunKind, total, done, failed, dbgen.JobStateRunning))
		}
	}

	state := dbgen.JobStateSucceeded
	if cancelled {
		// The admin's own deliberate ending (E41): the pages already
		// resolved keep their writes, the state reads cancelled --
		// never a broken run.
		state = dbgen.JobStateCancelled
	} else if failed > 0 {
		state = dbgen.JobStateFailed
	}
	return w.finishRerun(ctx, jobRunID, total, done, failed, state)
}

// finishRerun converges the row and announces the close. The final
// bookkeeping runs detached from the job context so the row and the
// finished event land even when the run consumed the context's last
// moments; a failure here returns the error and River retries the
// job, whose re-walk re-converges the row.
func (w *PlacesRerunWorker) finishRerun(ctx context.Context, jobRunID uuid.UUID, total, done, failed int, state dbgen.JobState) error {
	finishCtx := context.WithoutCancel(ctx)
	if err := dbgen.New(w.pool).FinishJobRun(finishCtx, dbgen.FinishJobRunParams{
		ID:     pgUUID(jobRunID),
		State:  state,
		Done:   int32(done),
		Failed: int32(failed),
	}); err != nil {
		return fmt.Errorf("finish the places rerun run %s: %w", jobRunID, err)
	}
	w.bus.Publish("job.finished", jobEventPayload(jobRunID, PlacesRerunKind, total, done, failed, state))
	return nil
}

// EnqueuePlacesRerun starts one re-resolution run: the job_runs row
// (the places rerun kind, the mode's scope count as its total) and the
// River job land in ONE transaction, so an enqueue that fails halfway
// leaves neither behind. The row starts queued; the worker flips it to
// running when the walk begins. The mode is the contract's enum and
// rides the job payload, so a retry re-walks the same scope. A queue
// whose process runs without the place reference data refuses the
// enqueue: there is no grid to re-resolve against, and the endpoint
// maps the refusal to the same 500 the broken-transaction arm answers
// (the log line carries the reason).
func (q *Queue) EnqueuePlacesRerun(ctx context.Context, mode string) (uuid.UUID, error) {
	if q.places == nil {
		return uuid.Nil, fmt.Errorf("jobs: the place resolver is not wired (no reference data)")
	}
	if mode != PlacesRerunModeMissing && mode != PlacesRerunModeAll {
		return uuid.Nil, fmt.Errorf("jobs: the places rerun mode %q is outside the enum", mode)
	}
	id, err := uuid.NewV7()
	if err != nil {
		return uuid.Nil, fmt.Errorf("generate the places rerun run id: %w", err)
	}
	tx, err := q.pool.Begin(ctx)
	if err != nil {
		return uuid.Nil, fmt.Errorf("begin the places rerun enqueue: %w", err)
	}
	// The rollback is a no-op after a successful commit; arguments
	// are captured while the request context is still live.
	defer func() { _ = tx.Rollback(ctx) }()

	// The scope's count, read inside the same transaction as the
	// insert: the total the row announces is exactly the scope the
	// enqueue saw.
	queries := dbgen.New(tx)
	var total int64
	if mode == PlacesRerunModeMissing {
		total, err = queries.CountMissingPlaceAssets(ctx)
	} else {
		total, err = queries.CountGpsAssets(ctx)
	}
	if err != nil {
		return uuid.Nil, fmt.Errorf("count the places rerun scope: %w", err)
	}
	if err := queries.InsertJobRun(ctx, dbgen.InsertJobRunParams{
		ID:    pgUUID(id),
		Kind:  PlacesRerunKind,
		Total: int32(total),
	}); err != nil {
		return uuid.Nil, fmt.Errorf("insert the places rerun run row: %w", err)
	}
	if _, err := q.client.InsertTx(ctx, tx, PlacesRerunArgs{JobRunID: id, Mode: mode}, nil); err != nil {
		return uuid.Nil, fmt.Errorf("insert the places rerun job: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return uuid.Nil, fmt.Errorf("commit the places rerun enqueue: %w", err)
	}
	return id, nil
}

// placesRerunRow is one page row of either scope: the asset id and
// the resolution input. Both page queries answer the same three
// columns, so the walk reads them through one shape.
type placesRerunRow struct {
	id  pgtype.UUID
	lat pgtype.Float8
	lon pgtype.Float8
}

// listPlacesRerunPage answers one page of the mode's scope: the
// fill-in set (place absent) or the full GPS-bearing set, both
// keyset-ordered by the asset id. The mode was validated by the
// worker before the walk began.
func listPlacesRerunPage(ctx context.Context, queries *dbgen.Queries, mode string, cursor pgtype.UUID) ([]placesRerunRow, error) {
	if mode == PlacesRerunModeMissing {
		rows, err := queries.ListMissingPlaceAssetPage(ctx, dbgen.ListMissingPlaceAssetPageParams{
			Cursor:     cursor,
			LimitCount: placesRerunPageSize,
		})
		if err != nil {
			return nil, err
		}
		page := make([]placesRerunRow, 0, len(rows))
		for _, row := range rows {
			page = append(page, placesRerunRow{id: row.ID, lat: row.Lat, lon: row.Lon})
		}
		return page, nil
	}
	rows, err := queries.ListGpsAssetPage(ctx, dbgen.ListGpsAssetPageParams{
		Cursor:     cursor,
		LimitCount: placesRerunPageSize,
	})
	if err != nil {
		return nil, err
	}
	page := make([]placesRerunRow, 0, len(rows))
	for _, row := range rows {
		page = append(page, placesRerunRow{id: row.ID, lat: row.Lat, lon: row.Lon})
	}
	return page, nil
}
