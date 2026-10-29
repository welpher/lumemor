package auth

import (
	"sync"
	"testing"
	"time"
)

// fakeClock is a manually advanced clock for driving the lockout curve
// without sleeping.
type fakeClock struct {
	mu  sync.Mutex
	now time.Time
}

func (c *fakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *fakeClock) advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

// newTestLimiter builds a limiter on a deterministic clock anchored at a
// fixed instant.
func newTestLimiter(t *testing.T) (*Limiter, *fakeClock) {
	t.Helper()
	clock := &fakeClock{now: time.Date(2026, 10, 29, 12, 0, 0, 0, time.UTC)}
	return NewLimiter(clock.Now), clock
}

// burnLockout records the five failures that trigger one lockout.
func burnLockout(l *Limiter, key string) {
	for i := 0; i < lockoutThreshold; i++ {
		l.Fail(key)
	}
}

// TestLimiterLocksAfterFiveFailures pins the threshold: four failures
// leave the key usable, the fifth locks it, and a locked key stays
// locked on repeated Allow probes (a correct password cannot unlock it
// -- Allow answers for the lock, not the credentials).
func TestLimiterLocksAfterFiveFailures(t *testing.T) {
	limiter, _ := newTestLimiter(t)

	for i := 0; i < lockoutThreshold-1; i++ {
		limiter.Fail("ip|ada@example.com")
		if !limiter.Allow("ip|ada@example.com") {
			t.Fatalf("Allow() = false after %d failures, want true (lock starts at %d)", i+1, lockoutThreshold)
		}
	}

	limiter.Fail("ip|ada@example.com")
	if limiter.Allow("ip|ada@example.com") {
		t.Fatalf("Allow() = true after %d failures, want false (locked)", lockoutThreshold)
	}
	if limiter.Allow("ip|ada@example.com") {
		t.Error("Allow() = true on a repeat probe of a locked key, want false")
	}

	// The lock is per key; other keys are unaffected.
	if !limiter.Allow("ip|grace@example.com") {
		t.Error("Allow() of an untouched key = false, want true")
	}
}

// TestLimiterCurve walks the doubling lockout curve: the first lockout
// lasts 15 minutes, every subsequent lockout doubles the duration up to
// the 1-hour cap, and each lockout ends exactly at its deadline.
func TestLimiterCurve(t *testing.T) {
	limiter, clock := newTestLimiter(t)
	const key = "ip|ada@example.com"

	// Lockout 1: 15 minutes.
	burnLockout(limiter, key)
	if got, want := limiter.RetryAfter(key), 15*time.Minute; got != want {
		t.Fatalf("first lockout RetryAfter() = %v, want %v", got, want)
	}
	clock.advance(15 * time.Minute)
	if !limiter.Allow(key) {
		t.Fatal("Allow() = false at the first lockout deadline, want true (unlocked)")
	}

	// Lockout 2: doubled to 30 minutes.
	burnLockout(limiter, key)
	if got, want := limiter.RetryAfter(key), 30*time.Minute; got != want {
		t.Fatalf("second lockout RetryAfter() = %v, want %v", got, want)
	}
	clock.advance(20 * time.Minute)
	if limiter.Allow(key) {
		t.Fatal("Allow() = true 10 minutes before the second lockout ends, want false")
	}
	clock.advance(10 * time.Minute)
	if !limiter.Allow(key) {
		t.Fatal("Allow() = false at the second lockout deadline, want true (unlocked)")
	}

	// Lockout 3: doubled to the 1-hour cap.
	burnLockout(limiter, key)
	if got, want := limiter.RetryAfter(key), time.Hour; got != want {
		t.Fatalf("third lockout RetryAfter() = %v, want %v (cap)", got, want)
	}
	clock.advance(time.Hour)
	if !limiter.Allow(key) {
		t.Fatal("Allow() = false at the third lockout deadline, want true (unlocked)")
	}

	// Lockout 4: still the cap, not a further doubling.
	burnLockout(limiter, key)
	if got, want := limiter.RetryAfter(key), time.Hour; got != want {
		t.Fatalf("fourth lockout RetryAfter() = %v, want %v (cap)", got, want)
	}
}

// TestLimiterRetryAfterDecays verifies RetryAfter reports the remaining
// lockout, not the full duration, and zero for unlocked keys.
func TestLimiterRetryAfterDecays(t *testing.T) {
	limiter, clock := newTestLimiter(t)
	const key = "ip|ada@example.com"

	if got := limiter.RetryAfter(key); got != 0 {
		t.Fatalf("RetryAfter() of an untouched key = %v, want 0", got)
	}
	burnLockout(limiter, key)
	clock.advance(5 * time.Minute)
	if got, want := limiter.RetryAfter(key), 10*time.Minute; got != want {
		t.Errorf("RetryAfter() 5 minutes into a 15-minute lockout = %v, want %v", got, want)
	}
}

// TestLimiterResetClearsFailures pins the login-success path: a
// successful login clears the accumulated failure count (but the
// failure history of a locked key is only reachable through Allow,
// which already refuses it).
func TestLimiterResetClearsFailures(t *testing.T) {
	limiter, _ := newTestLimiter(t)
	const key = "ip|ada@example.com"

	for i := 0; i < lockoutThreshold-1; i++ {
		limiter.Fail(key)
	}
	limiter.Reset(key)

	// The counter restarted: these four failures stay below the
	// threshold.
	for i := 0; i < lockoutThreshold-1; i++ {
		limiter.Fail(key)
	}
	if !limiter.Allow(key) {
		t.Fatal("Allow() = false after Reset plus 4 failures, want true (counter was cleared)")
	}
}

// TestLimiterFailuresExpire documents the failure window: failures
// older than the base lockout duration do not accumulate toward a
// lockout, so scattered mistakes over weeks never lock an account.
func TestLimiterFailuresExpire(t *testing.T) {
	limiter, clock := newTestLimiter(t)
	const key = "ip|ada@example.com"

	for i := 0; i < lockoutThreshold-1; i++ {
		limiter.Fail(key)
	}
	// The whole burst ages out before one more failure arrives.
	clock.advance(baseLockout + time.Minute)
	limiter.Fail(key)
	if !limiter.Allow(key) {
		t.Fatalf("Allow() = false after one fresh failure, want true (the %d stale failures expired)", lockoutThreshold-1)
	}
}

// TestLimiterKeysAreIndependent verifies one key's lock never touches
// another (different account or different address).
func TestLimiterKeysAreIndependent(t *testing.T) {
	limiter, _ := newTestLimiter(t)
	burnLockout(limiter, "ip1|ada@example.com")

	if limiter.Allow("ip2|ada@example.com") != true {
		t.Error("Allow() of another address = false, want true")
	}
	if limiter.Allow("ip1|grace@example.com") != true {
		t.Error("Allow() of another account = false, want true")
	}
}

// TestLimiterConcurrentAccess hammers the limiter from many goroutines;
// the -race build verifies the mutex actually guards the map.
func TestLimiterConcurrentAccess(t *testing.T) {
	limiter, _ := newTestLimiter(t)
	var wg sync.WaitGroup
	for g := 0; g < 8; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			key := "worker"
			for i := 0; i < 200; i++ {
				limiter.Allow(key)
				limiter.Fail(key)
				limiter.RetryAfter(key)
			}
		}(g)
	}
	wg.Wait()
}
