// Daily generation budget: a fixed number of events per key per UTC
// calendar day. This is the shape behind "10 pairing tokens a day per
// user" and is deliberately a sibling of the login Limiter, not part
// of it: the Limiter counts failures and answers with escalating
// lockouts, while a daily budget counts allowed events and returns as
// a whole at UTC midnight. State is in-memory per process (a family
// instance is a single process), so a restart forgets the day's
// counting -- the same trade the login limiter makes to keep the auth
// path store-free.
package auth

import (
	"sync"
	"time"
)

// dailyWindow is the length of one budget window: the UTC calendar day.
const dailyWindow = 24 * time.Hour

// dailyEntry is the budget state of one key.
type dailyEntry struct {
	// windowStart is the UTC midnight the count belongs to; a stale
	// window resets the count in place.
	windowStart time.Time
	// count is how many events the key consumed during windowStart's
	// day.
	count int
}

// DailyLimiter bounds events per key per UTC day. The zero value is
// not usable; construct with NewDailyLimiter. All methods are safe for
// concurrent use.
type DailyLimiter struct {
	mu sync.Mutex
	// limit is the per-day budget of one key.
	limit int
	// entries holds one record per key. The map is bounded by the
	// number of distinct keys, and the callers derive keys from stored
	// accounts (user ids), so no sweep is needed: unlike the login
	// limiter there is no escalation history worth retaining either.
	entries map[string]*dailyEntry
	// now supplies the clock driving the window; injected so tests
	// cross UTC midnight without sleeping.
	now func() time.Time
}

// NewDailyLimiter builds a limiter allowing limit events per key per
// UTC day; a nil now falls back to time.Now. A non-positive limit
// allows nothing, which keeps a misconfiguration fail-closed.
func NewDailyLimiter(limit int, now func() time.Time) *DailyLimiter {
	if now == nil {
		now = time.Now
	}
	return &DailyLimiter{
		limit:   limit,
		entries: make(map[string]*dailyEntry),
		now:     now,
	}
}

// Allow reports whether one more event on key fits today's budget and
// counts it when it does. The budget is consumed by allowed events,
// not by rejected ones: the pairing endpoint mints exactly when Allow
// says yes, so count and minted tokens agree (a mint that fails after
// the allowance burns the budget -- the conservative direction, an
// attacker gains nothing from making the database fail).
func (l *DailyLimiter) Allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	today := l.now().UTC().Truncate(dailyWindow)
	entry := l.entries[key]
	if entry == nil {
		entry = &dailyEntry{}
		l.entries[key] = entry
	}
	if !entry.windowStart.Equal(today) {
		// A new day (or the first sighting of the key): the window
		// rolls and the budget returns.
		entry.windowStart = today
		entry.count = 0
	}
	if entry.count >= l.limit {
		return false
	}
	entry.count++
	return true
}

// RetryAfter reports how much longer key stays over budget (zero when
// it is not), for the 429 response's Retry-After header: the remaining
// time of the current UTC day.
func (l *DailyLimiter) RetryAfter(key string) time.Duration {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	today := now.UTC().Truncate(dailyWindow)
	entry, ok := l.entries[key]
	if !ok || !entry.windowStart.Equal(today) || entry.count < l.limit {
		return 0
	}
	return today.Add(dailyWindow).Sub(now)
}
