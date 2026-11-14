// The background work's budget: the gate the task panel's run
// settings drive. Every pipeline job consults it at its start -- the
// per-kind pause switches, the night window, and the concurrency
// dial -- and the gate's answers take effect on the NEXT job, so a
// save is immediate without a restart.
//
// THE NEVER-BLOCKS RULE is structural here: the gate lives inside
// worker Work() functions and nowhere else. The upload path enqueues
// without consulting it, the serving endpoints read without it, and
// a held job is a River SNOOZE -- the job reschedules itself without
// consuming an attempt and without occupying a worker slot, so a
// paused kind holds nothing but its own future wake-ups.
package jobs

import (
	"context"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/welpher/lumemor/server/internal/db"
	dbgen "github.com/welpher/lumemor/server/internal/db/gen"
)

// The budget's job classes: the five pause switches' names, one per
// background-work family. A class maps to its pause switch and --
// for the heavy ones -- to the night window.
const (
	classThumbnail = "thumbnail"
	classMetadata  = "metadata"
	classTranscode = "transcode"
	classMl        = "ml"
	classPlaces    = "places"
)

// pauseHold is the snooze granularity of a paused kind's hold: the
// job wakes this often to re-read the switch, so an un-pause takes
// effect within a minute. A long pause costs one tiny settings read
// per wake.
const pauseHold = 30 * time.Second

// windowHoldCap bounds one window hold: a job outside the window
// wakes at most once a minute to re-read the budget, so a config
// change (or a window whose opening moved) lands within the minute
// -- the hold is one tiny settings read per wake, and a held week
// costs a thousand of them.
const windowHoldCap = time.Minute

// maxDialWorkers is the dial's ceiling: the queue's worker pool is
// sized for the biggest mapping the dial can produce (the document's
// own ceiling, db.DialWorkers's cap -- one number, one meaning).
const maxDialWorkers = 8

// Budget is the gate over the pipeline workers: it reads the run
// budget document per check and owns the concurrency dial's
// semaphore. Construct with NewBudget; nil Budgets (the unit tests'
// directly constructed workers) gate nothing.
type Budget struct {
	pool   *pgxpool.Pool
	logger *slog.Logger
	sem    chan struct{}
}

// NewBudget builds the gate over the pool.
func NewBudget(pool *pgxpool.Pool, logger *slog.Logger) *Budget {
	if logger == nil {
		logger = slog.Default()
	}
	return &Budget{pool: pool, logger: logger, sem: make(chan struct{}, maxDialWorkers)}
}

// dialWorkers maps the dial's preset onto worker counts (the
// document's own interpretation, db.DialWorkers).
func dialWorkers(preset string) int {
	return db.DialWorkers(preset)
}

// Hold answers how long the job of the named class should snooze
// before re-reading the budget: the pause hold when the class's
// switch is on, the window hold when a heavy class sits outside the
// window, zero otherwise. A corrupt settings row answers the
// defaults (the read is logged): the pipeline keeps running while
// the log points at the row.
func (b *Budget) Hold(ctx context.Context, class string) time.Duration {
	budget, err := db.RunBudgetOf(ctx, dbgen.New(b.pool))
	if err != nil {
		b.logger.Error("read the run budget for a hold; answering the defaults", "error", err)
	}
	if budget.PausedJobOf(class) {
		return pauseHold
	}
	if heavyClass(class) && budget.NightlyEnabled {
		start, end := windowClock(budget.NightlyStart), windowClock(budget.NightlyEnd)
		now := windowNow()
		if hold := windowHold(now, start, end); hold > 0 {
			return hold
		}
	}
	return 0
}

// Gate combines the hold check and the dial acquisition, the one
// call a worker's Work makes at its start: a positive hold means
// "snooze and return" (no slot taken -- the job reschedules itself);
// zero means the dial's slot (for the classes that share the
// pipeline pool) is held until Release.
func (b *Budget) Gate(ctx context.Context, class string) (time.Duration, error) {
	if hold := b.Hold(ctx, class); hold > 0 {
		return hold, nil
	}
	if dialsConcurrency(class) {
		if err := b.Acquire(ctx, class); err != nil {
			return 0, err
		}
	}
	return 0, nil
}

// dialsConcurrency reports whether the class shares the pipeline
// pool the dial throttles: the metadata, thumbnail, transcode and
// rebuild work. The ML jobs run on their own conservative pool (the
// pinned discipline that keeps a burst of inference from starving
// the media pipeline) and the places walk is a microsecond per
// asset -- neither dials.
func dialsConcurrency(class string) bool {
	switch class {
	case classThumbnail, classMetadata, classTranscode:
		return true
	}
	return false
}

// Acquire takes one of the dial's concurrency slots for the named
// class's work, blocking until a slot is free or the context ends.
// The slot count is re-read per acquisition, so a dial change lands
// on the next job. The ML jobs run on their own conservative pool
// and never dial; the callers know their class.
func (b *Budget) Acquire(ctx context.Context, class string) error {
	for {
		budget, err := db.RunBudgetOf(ctx, dbgen.New(b.pool))
		if err != nil {
			b.logger.Error("read the run budget for an acquisition; answering the defaults", "error", err)
		}
		if len(b.sem) < dialWorkers(budget.ConcurrencyPreset) {
			select {
			case b.sem <- struct{}{}:
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		// The dial is full for this job: wait a beat and re-read --
		// a slot frees, or the preset changes under us.
		select {
		case <-time.After(250 * time.Millisecond):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// Release returns the dial slot an acquisition took.
func (b *Budget) Release() {
	<-b.sem
}

// heavyClass reports whether the class's work is window-gated: the
// heavy three of the design pin -- the derivative rebuild, the video
// transcodes, and the machine learning work. The light classes
// (metadata, thumbnails, place resolution) run whenever there is
// work, so an upload stays visible whatever the window says.
func heavyClass(class string) bool {
	switch class {
	case classTranscode, classMl:
		return true
	}
	return false
}

// windowClock parses one HH:MM clock time into minutes-since-midnight.
// The save validated the shape; a corrupt stored value reads as
// midnight (the accessor's per-field degrade already answered the
// default for a bad time, so this is unreachable in practice).
func windowClock(text string) int {
	if len(text) != 5 || text[2] != ':' {
		return 0
	}
	hh := int(text[0]-'0')*10 + int(text[1]-'0')
	mm := int(text[3]-'0')*10 + int(text[4]-'0')
	return hh*60 + mm
}

// windowNow is the server's local clock as minutes-since-midnight:
// the window runs on the server's wall time (the self-hosted box's
// own timezone).
func windowNow() int {
	now := time.Now()
	return now.Hour()*60 + now.Minute()
}

// windowContains reports whether now sits inside [start, end) -- or,
// when the start follows the end (a window spanning midnight), in
// [start, 24h) ∪ [0, end). The opening minute is in, the closing
// minute is out: a window that ends at 06:00 stops the work AT
// 06:00.
func windowContains(now, start, end int) bool {
	if start <= end {
		return now >= start && now < end
	}
	return now >= start || now < end
}

// windowHold answers the wait from now to the window's next opening,
// capped at the chunk bound so a config change shortens the wait at
// the next wake. Inside the window (or a zero-length gap) is zero.
func windowHold(now, start, end int) time.Duration {
	if windowContains(now, start, end) {
		return 0
	}
	until := 0
	if start <= end {
		// A same-day window: either the opening is ahead of us, or
		// the window closed this morning and opens tomorrow.
		if now < start {
			until = start - now
		} else {
			until = 24*60 - now + start
		}
	} else {
		// A midnight-spanning window: outside means now sits in
		// [end, start), and the opening is ahead today.
		until = start - now
	}
	hold := time.Duration(until) * time.Minute
	if hold > windowHoldCap {
		hold = windowHoldCap
	}
	return hold
}

// clockDelta answers the wall-time gap between two clock times
// (minutes-since-midnight), for the tests' sanity bounds.
func clockDelta(from, to int) time.Duration {
	delta := to - from
	if delta < 0 {
		delta += 24 * 60
	}
	return time.Duration(delta) * time.Minute
}
