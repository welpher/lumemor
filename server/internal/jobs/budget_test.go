package jobs

import (
	"context"
	"testing"
	"time"

	"github.com/welpher/lumemor/server/internal/db"
	"github.com/welpher/lumemor/server/internal/db/dbtest"
	dbgen "github.com/welpher/lumemor/server/internal/db/gen"
)

// TestDialWorkersDelegatesToTheDocument pins the queue's read of the
// mapping (the mapping's own test lives with the document): the
// queue's dial read answers the document's, and an unknown preset
// reads as the light step's one -- the dial never asks for a number,
// so a corrupt value's honest reading is the quietest.
func TestDialWorkersDelegatesToTheDocument(t *testing.T) {
	if dialWorkers("turbo") != 1 {
		t.Errorf("dialWorkers(unknown) = %d, want the light step's 1 (a corrupt preset reads as light)", dialWorkers("turbo"))
	}
	if dialWorkers("light") != 1 {
		t.Errorf("dialWorkers(light) = %d, want 1", dialWorkers("light"))
	}
}

// TestWindowContainsCoversTheEdges pins the window's containment and
// its boundary behavior: the opening minute inside, the closing
// minute out (a window that ends at 06:00 stops the work AT 06:00),
// and a start-after-end window spanning midnight.
func TestWindowContainsCoversTheEdges(t *testing.T) {
	for _, tt := range []struct {
		name  string
		start string
		end   string
		now   string
		want  bool
	}{
		{"opening minute", "02:00", "06:00", "02:00", true},
		{"inside", "02:00", "06:00", "03:30", true},
		{"closing minute", "02:00", "06:00", "06:00", false},
		{"after", "02:00", "06:00", "12:00", false},
		{"before", "02:00", "06:00", "01:59", false},
		{"midnight span inside late", "22:00", "04:00", "23:30", true},
		{"midnight span inside early", "22:00", "04:00", "03:59", true},
		{"midnight span outside morning", "22:00", "04:00", "04:00", false},
		{"midnight span outside evening", "22:00", "04:00", "21:59", false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got := windowContains(windowClock(tt.now), windowClock(tt.start), windowClock(tt.end))
			if got != tt.want {
				t.Errorf("windowContains(%s of %s-%s) = %v, want %v", tt.now, tt.start, tt.end, got, tt.want)
			}
		})
	}
}

// TestWindowHoldUntilStart pins the outside-window hold: the gate
// names the wait to the window's next opening, capped at the chunk
// bound so a config change shortens the wait at the next wake.
func TestWindowHoldUntilStart(t *testing.T) {
	start, end := windowClock("22:00"), windowClock("04:00")
	now := windowClock("12:00")
	hold := windowHold(now, start, end)
	if hold <= 0 || hold > windowHoldCap {
		t.Fatalf("windowHold = %v, want a positive wait within the %v chunk cap", hold, windowHoldCap)
	}
	if want := clockDelta(windowClock("12:00"), windowClock("22:00")); hold > want {
		t.Errorf("windowHold = %v, want at most the real gap %v", hold, want)
	}
}

// TestWindowHoldInsideIsZero pins the inside-window answer: the gate
// holds nothing.
func TestWindowHoldInsideIsZero(t *testing.T) {
	if hold := windowHold(windowClock("03:00"), windowClock("02:00"), windowClock("06:00")); hold != 0 {
		t.Errorf("windowHold inside = %v, want 0", hold)
	}
}

// TestBudgetHoldHonorsPause pins the pause hold: a paused kind's job
// is told to snooze for the pause's granularity, and an unpaused one
// is not.
func TestBudgetHoldHonorsPause(t *testing.T) {
	pool := dbtest.New(t)
	if err := db.Migrate(context.Background(), pool); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if err := db.SaveRunBudget(context.Background(), dbgen.New(pool), db.RunBudget{
		ConcurrencyPreset: "standard",
		NightlyEnabled:    true,
		NightlyStart:      "00:00",
		NightlyEnd:        "23:59",
		PausedJobs:        db.PausedJobs{Thumbnail: true},
	}); err != nil {
		t.Fatalf("save the run budget: %v", err)
	}
	budget := NewBudget(pool, discardLogger())

	// The window is wide open (00:00-23:59): the only reason to hold
	// a thumbnail job is the pause.
	hold := budget.Hold(context.Background(), "thumbnail")
	if hold != pauseHold {
		t.Errorf("Hold(paused thumbnail) = %v, want the pause granularity %v", hold, pauseHold)
	}
	if hold := budget.Hold(context.Background(), "metadata"); hold != 0 {
		t.Errorf("Hold(unpaused metadata) = %v, want 0", hold)
	}
}

// TestBudgetHoldHonorsWindow pins the window hold: a heavy kind
// outside the window snoozes to the opening; a light kind runs
// whatever the window says.
func TestBudgetHoldHonorsWindow(t *testing.T) {
	pool := dbtest.New(t)
	if err := db.Migrate(context.Background(), pool); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	// The window is placed one to two hours ahead of the REAL clock:
	// whatever the wall reads, now sits outside it (a fixed window
	// would flip the test's meaning for four hours a day).
	ahead := func(hours int) string {
		return time.Now().Add(time.Duration(hours) * time.Hour).Format("15:04")
	}
	if err := db.SaveRunBudget(context.Background(), dbgen.New(pool), db.RunBudget{
		ConcurrencyPreset: "standard",
		NightlyEnabled:    true,
		NightlyStart:      ahead(1),
		NightlyEnd:        ahead(2),
		PausedJobs:        db.PausedJobs{},
	}); err != nil {
		t.Fatalf("save the run budget: %v", err)
	}
	budget := NewBudget(pool, discardLogger())

	// A heavy kind outside the window: held to the window's opening.
	hold := budget.Hold(context.Background(), "transcode")
	if hold <= 0 || hold > windowHoldCap {
		t.Errorf("Hold(heavy outside window) = %v, want a positive hold within the chunk cap", hold)
	}
	// A light kind outside the window: never window-gated.
	if hold := budget.Hold(context.Background(), "metadata"); hold != 0 {
		t.Errorf("Hold(light outside window) = %v, want 0", hold)
	}
}

// TestBudgetHoldCorruptRowAnswersDefault pins the corrupt-row
// discipline: a garbage budget row answers the defaults -- the
// pipeline keeps running while the log points at the row.
func TestBudgetHoldCorruptRowAnswersDefault(t *testing.T) {
	pool := dbtest.New(t)
	if err := db.Migrate(context.Background(), pool); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if _, err := pool.Exec(context.Background(),
		`INSERT INTO instance_settings (key, value) VALUES ('run_budget', '"not an object"'::jsonb)`); err != nil {
		t.Fatalf("seed the corrupt row: %v", err)
	}
	budget := NewBudget(pool, discardLogger())

	// The default posture: nothing paused, no window -- the hold is
	// zero whatever the kind.
	if hold := budget.Hold(context.Background(), "transcode"); hold != 0 {
		t.Errorf("Hold over a corrupt row = %v, want 0 (the default posture)", hold)
	}
}

// TestBudgetAcquireDialsTheConcurrency pins the semaphore: the dial's
// preset caps the pipeline's simultaneous slots, and a release frees
// the blocked waiter.
func TestBudgetAcquireDialsTheConcurrency(t *testing.T) {
	pool := dbtest.New(t)
	if err := db.Migrate(context.Background(), pool); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if err := db.SaveRunBudget(context.Background(), dbgen.New(pool), db.RunBudget{
		ConcurrencyPreset: "light",
	}); err != nil {
		t.Fatalf("save the run budget: %v", err)
	}
	budget := NewBudget(pool, discardLogger())

	ctx := context.Background()
	if err := budget.Acquire(ctx, "thumbnail"); err != nil {
		t.Fatalf("first Acquire: %v", err)
	}
	// The light dial caps at one: the second acquire cannot land
	// while the first holds -- it blocks, and a bounded wait proves
	// the block rather than hanging the suite.
	blockedCtx, cancel := context.WithTimeout(ctx, 300*time.Millisecond)
	defer cancel()
	if err := budget.Acquire(blockedCtx, "metadata"); err == nil {
		t.Fatal("second Acquire over the light dial = nil, want the bounded wait to expire")
	}
	// The dial's slot frees; a fresh acquisition lands.
	budget.Release()
	if err := budget.Acquire(ctx, "transcode"); err != nil {
		t.Fatalf("Acquire after the release: %v", err)
	}
	budget.Release()
}
