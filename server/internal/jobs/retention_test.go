// Retention queue tests: the hourly periodic
// registration of the trash expiry (pinned through its schedule
// config and the started queue's bundle, never by waiting a
// wall-clock hour), and the worker path end to end: a RetentionArgs
// job inserted into the live queue is worked by the registered
// RetentionWorker, which runs the shared trash.RunExpiry over the
// queue's own pool and data root, purges the expired row with its
// file, announces the purge on the bus, and emits exactly ONE log
// line for the run -- the counts, never a line per asset.
package jobs

import (
	"bytes"
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	dbgen "github.com/welpher/lumemor/server/internal/db/gen"
)

// TestRetentionPeriodicRegistration pins the hourly periodic wiring
// without waiting a wall-clock hour: the cadence is one hour, the
// schedule advances by exactly that interval, the constructor
// inserts the argumentless expiry job on the default queue, and a
// started queue actually carries the registration in River's
// periodic job bundle (RemoveByID reports its existence) -- the
// sweep's registration alongside it, the two never folded into one
// job.
func TestRetentionPeriodicRegistration(t *testing.T) {
	if RetentionInterval != time.Hour {
		t.Errorf("RetentionInterval = %s, want the hourly 1h", RetentionInterval)
	}

	now := time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC)
	if got := retentionSchedule().Next(now); !got.Equal(now.Add(time.Hour)) {
		t.Errorf("retention schedule next(%s) = %s, want +1h", now, got)
	}

	args, opts := retentionJobConstructor()
	if _, ok := args.(RetentionArgs); !ok {
		t.Errorf("periodic job args = %T, want RetentionArgs", args)
	}
	if args.Kind() != "trash_expiry" {
		t.Errorf("periodic job kind = %q, want trash_expiry", args.Kind())
	}
	if opts != nil {
		t.Errorf("periodic job insert opts = %+v, want the defaults (nil)", opts)
	}

	if got := len(retentionPeriodicJobs()); got != 1 {
		t.Errorf("retention periodic jobs = %d, want exactly the one registration", got)
	}
	if got := len(periodicJobs()); got != 2 {
		t.Errorf("queue periodic jobs = %d, want the sweep and the expiry (two registrations)", got)
	}

	e := newEnv(t)
	if !e.queue.client.PeriodicJobs().RemoveByID(retentionPeriodicJobID) {
		t.Error("the trash expiry periodic job is not registered in the started queue")
	}
	if !e.queue.client.PeriodicJobs().RemoveByID(sweepPeriodicJobID) {
		t.Error("the requeue sweep periodic job lost its registration")
	}
}

// TestRetentionWorkerJobRunsExpiry drives the periodic job's worker
// path end to end: a RetentionArgs job inserted into the live queue
// is worked by the registered RetentionWorker, which runs the shared
// expiry over the queue's own pool and data root, removes the expired
// row together with its file, publishes the asset.deleted
// announcement, and emits exactly one log line for the run.
func TestRetentionWorkerJobRunsExpiry(t *testing.T) {
	var logs bytes.Buffer
	e := newEnvLogger(t, nil, slog.New(slog.NewTextHandler(&logs, nil)), nil, nil)

	deviceTakenAt := time.Date(2021, 8, 9, 10, 11, 12, 0, time.UTC).Truncate(time.Microsecond)
	id := e.insertAssetKind(t, "originals/retention/worker.jpg", jpegBytes(t), deviceTakenAt,
		`{}`, dbgen.MediaKindImage, "image/jpeg", 0)
	// The expired row: trashed, backdated past the default retention.
	// The deadline is a computation, never a sleep.
	e.softDeleteAsset(t, id)
	if _, err := e.pool.Exec(context.Background(),
		`UPDATE assets SET deleted_at = now() - make_interval(days => 31) WHERE id = $1`, id); err != nil {
		t.Fatalf("backdate the trashed asset: %v", err)
	}

	// The periodic scheduler inserts exactly this job on the hourly
	// cadence; here it is inserted directly to drive the worker now.
	if _, err := e.queue.client.Insert(context.Background(), RetentionArgs{}, nil); err != nil {
		t.Fatalf("insert trash expiry job: %v", err)
	}
	eventually(t, jobWaitTimeout, "the trash expiry job to complete", func() bool {
		var state string
		err := e.pool.QueryRow(context.Background(),
			`SELECT state FROM river_job WHERE kind = 'trash_expiry'`).Scan(&state)
		return err == nil && state == "completed"
	})

	// The worker's run purged the row and its file.
	if _, err := e.queries.GetAssetByID(context.Background(), pgUUID(id)); err == nil {
		t.Fatal("the expired asset row survived the worker's run, want it purged")
	}
	if _, err := os.Stat(filepath.Join(e.dataDir, "originals", "retention", "worker.jpg")); err == nil {
		t.Error("the expired asset's file survived the worker's run, want it removed")
	}

	// The purge announced itself on the bus.
	payload := eventPayloadOf(t, e.bus, "asset.deleted")
	if payload["assetId"] != id.String() {
		t.Errorf("asset.deleted assetId = %q, want %s", payload["assetId"], id)
	}
	if payload["kind"] != "image" {
		t.Errorf("asset.deleted kind = %q, want image", payload["kind"])
	}

	// The log discipline: one line per run, carrying the counts. The
	// buffer also carries River's own service diagnostics; only the
	// expiry's own line is counted.
	if got := strings.Count(logs.String(), "trash expiry complete"); got != 1 {
		t.Errorf("trash expiry complete log lines = %d, want exactly 1", got)
	}
}
