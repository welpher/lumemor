// The folder sync's activity gate tests: whether a tick sweeps at all
// is a function of the instance's activity signal and the local clock,
// so both are injected here -- the gate is driven on a fixed clock with
// a fabricated signal, never on River's cadence or the wall clock.
//
// The gate's pins, in the order the cases read: an active house sweeps
// on every tick; an idle house sweeps once per local day, in the first
// hour after local midnight; the first tick after the house wakes is an
// active tick and therefore sweeps. The worker test at the bottom drives
// the real query the production wiring uses (HasActiveSession) against
// the session rows the test environment's own login left behind.
package jobs

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	dbgen "github.com/welpher/lumemor/server/internal/db/gen"
)

// TestFolderSyncGateActivityAndNightlySlot walks the gate's decision
// table: active sweeps, idle waits for the nightly slot, the slot runs
// once per local day, and the hour boundary is the local one.
func TestFolderSyncGateActivityAndNightlySlot(t *testing.T) {
	shanghai := time.FixedZone("UTC+8", 8*60*60)
	noon := time.Date(2026, 11, 20, 12, 0, 0, 0, time.UTC)

	gate := &folderSyncGate{}
	if sweep, nightly := gate.allow(noon, true); !sweep || nightly {
		t.Errorf("an active tick at noon = (sweep %v, nightly %v), want the plain cadence sweep", sweep, nightly)
	}

	// Idle outside the first hour: nothing to do, however many ticks
	// arrive.
	if sweep, _ := gate.allow(noon, false); sweep {
		t.Error("an idle tick at noon swept, want it quiet")
	}
	if sweep, _ := gate.allow(noon.Add(time.Hour), false); sweep {
		t.Error("an idle tick an hour later swept, want it quiet")
	}

	// The nightly slot: the first idle tick of the local day's 0 hour
	// sweeps once, and the rest of that hour stays quiet.
	midnight := time.Date(2026, 11, 21, 0, 20, 0, 0, time.UTC)
	if sweep, nightly := gate.allow(midnight, false); !sweep || !nightly {
		t.Errorf("the first idle tick of the local 0 hour = (sweep %v, nightly %v), want the nightly sweep", sweep, nightly)
	}
	if sweep, _ := gate.allow(midnight.Add(30*time.Minute), false); sweep {
		t.Error("a second idle tick in the same 0 hour swept, want the once-a-day pin to hold")
	}

	// The next local day's slot opens again.
	if sweep, nightly := gate.allow(midnight.Add(24*time.Hour), false); !sweep || !nightly {
		t.Errorf("the next day's first idle tick of the 0 hour = (sweep %v, nightly %v), want the nightly sweep", sweep, nightly)
	}

	// The slot is the FIRST hour only: an idle boot at 01:00 waits for
	// the next midnight rather than sweeping late.
	gate = &folderSyncGate{}
	if sweep, _ := gate.allow(time.Date(2026, 11, 21, 1, 0, 0, 0, time.UTC), false); sweep {
		t.Error("an idle tick at 01:00 swept, want the slot to be the first hour only")
	}

	// Midnight follows the process's local zone, not UTC: 00:20 in
	// UTC+8 is 16:20 UTC, and it is that local date's slot.
	gate = &folderSyncGate{}
	localMidnight := time.Date(2026, 11, 21, 0, 20, 0, 0, shanghai)
	if sweep, nightly := gate.allow(localMidnight, false); !sweep || !nightly {
		t.Errorf("the idle tick at local 00:20 (UTC %s) = (sweep %v, nightly %v), want the nightly sweep",
			localMidnight.UTC().Format(time.RFC3339), sweep, nightly)
	}
	// The same instant is 16:20 UTC; the local date is what makes it a
	// slot, and that date's slot is now spent.
	if sweep, _ := gate.allow(localMidnight.Add(20*time.Minute), false); sweep {
		t.Error("a later tick of the same local date swept again, want the once-a-day pin to follow the local date")
	}
}

// TestFolderSyncGateFirstActiveTickSweeps pins the wake-up edge: an
// idle stretch of skipped ticks must not delay the first sweep after
// somebody signs in. The transition carries no state -- an active tick
// sweeps -- and the gate's own nightly bookkeeping survives it.
func TestFolderSyncGateFirstActiveTickSweeps(t *testing.T) {
	gate := &folderSyncGate{}
	start := time.Date(2026, 11, 20, 9, 0, 0, 0, time.UTC)
	for i := 0; i < 10; i++ {
		if sweep, _ := gate.allow(start.Add(time.Duration(i)*time.Minute), false); sweep {
			t.Fatalf("idle tick %d at 09:%02d swept, want the house asleep", i, i)
		}
	}
	// The next tick after the sign-in: one cadence window, not one
	// night.
	if sweep, _ := gate.allow(start.Add(10*time.Minute), true); !sweep {
		t.Error("the first tick after idle-to-active did not sweep, want the immediate convergence")
	}
	// And the active stretch keeps sweeping on every tick.
	if sweep, _ := gate.allow(start.Add(11*time.Minute), true); !sweep {
		t.Error("a later active tick did not sweep, want the configured cadence")
	}
}

// TestFolderSyncWorkerSweepsOnActivityOnly drives the worker's own tick:
// the same album, the same change, two activity answers. Idle at noon
// decides nothing; active sweeps it. The activity signal is the real
// query against the test environment's session rows, served through a
// seam that reports what the query answered.
func TestFolderSyncWorkerSweepsOnActivityOnly(t *testing.T) {
	requireExiftool(t)
	requireThumbnails(t)
	e := newEnv(t)

	source := filepath.Join(t.TempDir(), "family")
	if err := os.MkdirAll(source, 0o755); err != nil {
		t.Fatalf("create the source directory: %v", err)
	}
	stamp := time.Date(2026, 11, 20, 8, 0, 0, 0, time.UTC)
	writeSourceFile(t, filepath.Join(source, "a.jpg"), jpegBytes(t), stamp)

	album := e.createDirectoryAlbum(t, "Family archive", source)
	first := e.waitCreationScan(t, album)
	// The creation scan owns the first baseline; a fresh sweep over the
	// unchanged folder is what the gate will skip or run.
	e.syncNow(t)

	// The change under test: one dropped file, decided by whichever
	// tick the gate lets through.
	writeSourceFile(t, filepath.Join(source, "b.jpg"), jpegBytes(t), stamp.Add(time.Hour))

	queries := dbgen.New(e.pool)
	activity := func(ctx context.Context) (bool, error) {
		return queries.HasActiveSession(ctx, pgtype.Timestamptz{
			Time:  time.Now().Add(-folderActivityWindow),
			Valid: true,
		})
	}
	clock := time.Date(2026, 11, 20, 12, 0, 0, 0, time.UTC)
	worker := &FolderSyncWorker{
		pool:      e.pool,
		snapshots: e.queue.snapshots,
		enqueue:   e.queue.EnqueueAlbumRescan,
		logger:    slog.New(slog.NewTextHandler(os.Stderr, nil)),
		activity:  activity,
		now:       func() time.Time { return clock },
	}

	// The environment's login is minutes old at this point, so the real
	// signal answers active and the first tick sweeps.
	if active, err := activity(context.Background()); err != nil || !active {
		t.Fatalf("HasActiveSession with a fresh login = (%v, %v), want active", active, err)
	}
	if err := worker.Work(context.Background(), nil); err != nil {
		t.Fatalf("the active tick returned %v, want nil", err)
	}
	second := e.waitNewScan(t, album, first)

	// Now the house sleeps: the session's last-seen moves past the
	// window, and the same tick (with another change on disk) decides
	// nothing.
	if _, err := e.pool.Exec(context.Background(),
		"UPDATE sessions SET last_seen_at = now() - interval '10 minutes'"); err != nil {
		t.Fatalf("backdate the sessions: %v", err)
	}
	if active, err := activity(context.Background()); err != nil || active {
		t.Fatalf("HasActiveSession after backdating = (%v, %v), want idle", active, err)
	}
	writeSourceFile(t, filepath.Join(source, "c.jpg"), jpegBytes(t), stamp.Add(2*time.Hour))
	runsBefore := e.countRuns(t, album)
	if err := worker.Work(context.Background(), nil); err != nil {
		t.Fatalf("the idle tick returned %v, want a quiet skip", err)
	}
	if got := e.countRuns(t, album); got != runsBefore {
		t.Errorf("the idle tick left %d runs, want the unchanged %d", got, runsBefore)
	}

	// The idle query's exclusions: an expired web row is not activity,
	// and a mobile row never expires.
	if _, err := e.pool.Exec(context.Background(),
		"UPDATE sessions SET last_seen_at = now(), expires_at = now() - interval '1 minute'"); err != nil {
		t.Fatalf("expire the web session: %v", err)
	}
	if active, err := activity(context.Background()); err != nil || active {
		t.Fatalf("HasActiveSession with an expired web session = (%v, %v), want idle", active, err)
	}
	if _, err := e.pool.Exec(context.Background(),
		"UPDATE sessions SET kind = 'mobile', expires_at = now() - interval '1 minute'"); err != nil {
		t.Fatalf("turn the session mobile: %v", err)
	}
	if active, err := activity(context.Background()); err != nil || !active {
		t.Fatalf("HasActiveSession with a revoked-by-expiry mobile session = (%v, %v), want active (mobile rows never expire)", active, err)
	}

	// The active tick over the next change still converges: the gate
	// gates the cadence, never the diff's correctness.
	if err := worker.Work(context.Background(), nil); err != nil {
		t.Fatalf("the active tick returned %v, want nil", err)
	}
	e.waitNewScan(t, album, second)
}
