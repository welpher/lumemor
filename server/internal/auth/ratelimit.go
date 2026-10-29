// Login failure rate limiting: 5 failures for one address+account pair
// lock that pair out, and every subsequent lockout doubles its duration
// up to a 1-hour cap. State is in-memory (a family instance is a single
// process), so locks reset on restart -- an acceptable trade for not
// growing a persistence dependency into the auth path.
package auth

import (
	"sync"
	"time"
)

const (
	// lockoutThreshold is the failure count that triggers a lockout.
	lockoutThreshold = 5

	// baseLockout is the duration of the first lockout; the window
	// within which failures accumulate (lockoutThreshold failures
	// spaced further apart than this never lock).
	baseLockout = 15 * time.Minute

	// maxLockout caps the doubling: no lockout exceeds one hour.
	maxLockout = time.Hour

	// sweepRetention is how long an entry outlives its last activity
	// before lazy cleanup drops it: long enough that consecutive
	// lockouts keep escalating, short enough that a day of quiet
	// resets the escalation.
	sweepRetention = 24 * time.Hour
)

// limitEntry is the failure state of one rate-limit key.
type limitEntry struct {
	// failures is the count since the last lockout or reset; only
	// failures newer than baseLockout count (lastFail anchors the
	// window).
	failures int
	// lastFail is when the most recent failure arrived.
	lastFail time.Time
	// lockouts is how many lockouts this key has accumulated; each
	// new one doubles the duration.
	lockouts int
	// lockedUntil is when the current lockout ends (zero value when
	// not locked).
	lockedUntil time.Time
}

// Limiter rate limits login attempts per key (address|account). The
// zero value is not usable; construct with NewLimiter. All methods are
// safe for concurrent use.
type Limiter struct {
	mu sync.Mutex
	// entries holds one record per key that has failed at least once.
	entries map[string]*limitEntry
	// now supplies the clock driving windows and deadlines; injected
	// so tests advance it without sleeping.
	now func() time.Time
	// lastSweep anchors the lazy cleanup passes.
	lastSweep time.Time
}

// NewLimiter builds a limiter; a nil now falls back to time.Now.
func NewLimiter(now func() time.Time) *Limiter {
	if now == nil {
		now = time.Now
	}
	return &Limiter{
		entries:   make(map[string]*limitEntry),
		now:       now,
		lastSweep: now(),
	}
}

// Allow reports whether an attempt on key may proceed: false while the
// key is locked, and locked means locked -- even a correct password is
// rejected, which is what makes brute force uneconomical.
func (l *Limiter) Allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.sweepLocked()
	entry, ok := l.entries[key]
	if !ok {
		return true
	}
	// Unlocked at the deadline itself, not one nanosecond later.
	return !l.now().Before(entry.lockedUntil)
}

// Fail records a failed attempt on key: the failure count grows (within
// its window), and reaching lockoutThreshold starts the next lockout
// with doubled duration, capped at maxLockout. Failures while already
// locked are ignored -- the handler refuses locked keys before any
// credential work, so they cannot arrive through the login flow.
func (l *Limiter) Fail(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	entry := l.entries[key]
	if entry == nil {
		entry = &limitEntry{}
		l.entries[key] = entry
	}

	if !now.Before(entry.lockedUntil) {
		// Stale failures do not accumulate: outside the window the
		// counter restarts, so scattered failures never lock.
		if entry.failures > 0 && now.Sub(entry.lastFail) > baseLockout {
			entry.failures = 0
		}
		entry.failures++
		entry.lastFail = now
		if entry.failures >= lockoutThreshold {
			entry.lockouts++
			entry.lockedUntil = now.Add(lockoutDuration(entry.lockouts))
			entry.failures = 0
		}
	}
}

// Reset clears the failure count of key after a successful login. It
// deliberately does not lift an active lock: a locked key never reaches
// verification, so Reset on a locked key is a no-op by construction.
func (l *Limiter) Reset(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	entry, ok := l.entries[key]
	if !ok {
		return
	}
	entry.failures = 0
}

// RetryAfter reports how much longer key stays locked (zero when it is
// not locked), for the 429 response's Retry-After header.
func (l *Limiter) RetryAfter(key string) time.Duration {
	l.mu.Lock()
	defer l.mu.Unlock()
	entry, ok := l.entries[key]
	if !ok {
		return 0
	}
	remaining := entry.lockedUntil.Sub(l.now())
	if remaining < 0 {
		return 0
	}
	return remaining
}

// lockoutDuration is the lock length after the nth lockout: 15 minutes
// doubled per lockout, capped at one hour.
func lockoutDuration(n int) time.Duration {
	d := baseLockout
	for i := 1; i < n; i++ {
		d *= 2
		if d >= maxLockout {
			return maxLockout
		}
	}
	return d
}

// sweepLocked is the lazy cleanup: at most once per lockout cap it
// drops entries that went quiet -- unlocked and untouched for longer
// than sweepRetention -- keeping the map bounded on a long-lived
// process. Escalation history of recently active keys survives.
// Callers must hold l.mu.
func (l *Limiter) sweepLocked() {
	now := l.now()
	if now.Sub(l.lastSweep) < maxLockout {
		return
	}
	l.lastSweep = now
	for key, entry := range l.entries {
		stale := !now.Before(entry.lockedUntil) && now.Sub(entry.lastFail) > sweepRetention
		if stale {
			delete(l.entries, key)
		}
	}
}
