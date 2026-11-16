package jobs

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"

	"github.com/welpher/lumemor/server/internal/events"
	"github.com/welpher/lumemor/server/internal/trash"
)

// The trash expiry's queue plumbing: the River periodic
// job registration and the worker that runs the shared expiry
// implementation of the trash package (the database-and-filesystem
// operation itself lives there, next to its own tests, free of the
// queue and the media pipeline). The expiry purges soft-deleted
// assets whose retention window closed, under the shared-blob rule
// the pipeline handoff pinned: another row referencing the same physical
// file keeps the file, only the row goes.

// RetentionInterval is the periodic cadence of the trash expiry. One
// hour bounds how long an expired asset lingers past its deadline,
// matching the requeue sweep's maintenance rhythm while keeping the
// scan off the hot path of live traffic.
const RetentionInterval = time.Hour

// retentionPeriodicJobID identifies the expiry's registration in
// River's periodic job bundle, so the registration is assertable by
// RemoveByID.
const retentionPeriodicJobID = "trash_expiry"

// RetentionArgs is the periodic trash expiry job: it carries no
// payload, because the run reads its expired rows and its retention
// setting from the database.
type RetentionArgs struct{}

// Kind is River's stable job kind identifier.
func (RetentionArgs) Kind() string { return "trash_expiry" }

// RetentionWorker runs the periodic expiry job: the shared
// trash.RunExpiry over the queue's own pool and data root, with the
// asset.deleted announcements going to the bus the queue was wired
// with. Like the sweep worker it holds the pool directly (the
// workers are registered before the client exists, so it cannot hold
// the *Queue itself).
type RetentionWorker struct {
	river.WorkerDefaults[RetentionArgs]

	pool    *pgxpool.Pool
	dataDir string
	bus     *events.Bus
	logger  *slog.Logger
}

// Work executes one expiry run. The single log line carries the
// counts; per-asset logs are deliberately absent (a large backlog
// must not spam).
func (w *RetentionWorker) Work(ctx context.Context, job *river.Job[RetentionArgs]) error {
	counts, err := trash.RunExpiry(ctx, w.pool, w.dataDir, w.bus)
	if err != nil {
		// The job fails and River retries it under its default
		// policy; the partial run's counts still log.
		w.logger.Error("trash expiry failed", "error", err, "expired", counts.Expired, "files", counts.FilesRemoved)
		return fmt.Errorf("trash expiry: %w", err)
	}
	w.logger.Info("trash expiry complete", "expired", counts.Expired, "files", counts.FilesRemoved)
	return nil
}

// periodicJobs returns the queue's full periodic bundle: the requeue
// sweep, the trash expiry and the Live Photo pair maintenance, each on
// its own hourly schedule, plus the folder live sync on its own
// cadence. The combined list is what the River client starts with.
// The folder sync's cadence is the deployment's (60s unless
// configured otherwise); a negative interval omits its registration
// entirely -- the off switch, and the unit tests' determinism.
func periodicJobs(folderSyncInterval time.Duration) []*river.PeriodicJob {
	jobs := append(sweepPeriodicJobs(), retentionPeriodicJobs()...)
	jobs = append(jobs, livePhotoSweepPeriodicJobs()...)
	if folderSyncInterval > 0 {
		jobs = append(jobs, folderSyncPeriodicJobs(folderSyncInterval)...)
	}
	return jobs
}

// retentionPeriodicJobs returns the periodic job registration of the
// trash expiry on its hourly schedule. It is a registration of its
// own, deliberately NOT folded into the requeue sweep's job: the two
// maintenances fail and retry independently (a broken expiry must
// not fail the pipeline recovery's run, or the other way around),
// each keeps its own kind, counts and log line, and the River bundle
// stays additive -- one more registration, no shared worker state.
// RunOnStart is off, mirroring the sweep: the boot expiry in main.go
// covers the process start once, and a RunOnStart periodic would
// double it.
func retentionPeriodicJobs() []*river.PeriodicJob {
	return []*river.PeriodicJob{
		river.NewPeriodicJob(retentionSchedule(), retentionJobConstructor,
			&river.PeriodicJobOpts{ID: retentionPeriodicJobID}),
	}
}

// retentionSchedule is the expiry's periodic schedule: fixed hourly
// intervals, split out so the cadence is unit-testable without
// waiting a wall-clock hour.
func retentionSchedule() river.PeriodicSchedule {
	return river.PeriodicInterval(RetentionInterval)
}

// retentionJobConstructor is the periodic job's insert payload: the
// argumentless expiry job on the default queue with the default
// priority -- like the sweep, an hourly maintenance job has no
// business preempting pipeline work.
func retentionJobConstructor() (river.JobArgs, *river.InsertOpts) {
	return RetentionArgs{}, nil
}
