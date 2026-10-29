package events

import (
	"context"
	"sync"
	"testing"
	"time"
)

// testTimeout bounds every wait in this file so a wedged Next() fails
// the test instead of hanging it.
const testTimeout = 5 * time.Second

// mockClock is a manually advanced clock so retention tests never sleep.
type mockClock struct {
	mu  sync.Mutex
	now time.Time
}

func newMockClock() *mockClock {
	return &mockClock{now: time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)}
}

func (c *mockClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *mockClock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

// TestPublishAssignsMonotonicIDs pins the first mandatory behavior:
// Publish hands out ids starting at 1 and strictly increasing, and the
// payload is JSON-encoded once, at publish time.
func TestPublishAssignsMonotonicIDs(t *testing.T) {
	bus := NewBus(5*time.Minute, time.Now)

	for i, want := range []int64{1, 2, 3} {
		if got := bus.Publish("test.tick", map[string]int{"n": i}); got != want {
			t.Fatalf("Publish call #%d returned id %d, want %d", i+1, got, want)
		}
	}

	sub := bus.Subscribe(t.Context(), 0)
	ev, ok := sub.Next()
	if !ok {
		t.Fatal("Next() = false on a ring that holds events, want true")
	}
	if ev.ID != 1 || ev.Name != "test.tick" {
		t.Errorf("first event = {ID %d, Name %q}, want {ID 1, Name %q}", ev.ID, ev.Name, "test.tick")
	}
	if got := string(ev.Payload); got != `{"n":0}` {
		t.Errorf("payload = %s, want the JSON encoding of the first publish", got)
	}
}

// TestLateSubscriberReplaysRing pins the second mandatory behavior: a
// subscription opened after events were published still receives them,
// in publish order, from the retention ring.
func TestLateSubscriberReplaysRing(t *testing.T) {
	clock := newMockClock()
	bus := NewBus(5*time.Minute, clock.Now)

	bus.Publish("test.alpha", map[string]string{"k": "a"})
	bus.Publish("test.beta", map[string]string{"k": "b"})

	ctx, cancel := context.WithCancel(t.Context())
	sub := bus.Subscribe(ctx, 0)

	want := []struct {
		id   int64
		name string
		data string
	}{
		{1, "test.alpha", `{"k":"a"}`},
		{2, "test.beta", `{"k":"b"}`},
	}
	for _, w := range want {
		ev, ok := sub.Next()
		if !ok {
			t.Fatalf("Next() = false before %s, want true", w.name)
		}
		if ev.ID != w.id || ev.Name != w.name || string(ev.Payload) != w.data {
			t.Errorf("ring replay = {ID %d, Name %q, Payload %s}, want {ID %d, Name %q, Payload %s}",
				ev.ID, ev.Name, ev.Payload, w.id, w.name, w.data)
		}
	}

	// The ring is drained: canceling must end the stream rather than
	// leave Next() blocked forever.
	cancel()
	if _, ok := sub.Next(); ok {
		t.Error("Next() = true after the ring is drained and ctx canceled, want false")
	}
}

// TestExpiredRingEntriesAreNotReplayed pins the third mandatory
// behavior: entries older than the retention window are never
// delivered, neither to a subscriber from id 0 nor to one whose
// afterID points into the expired region.
func TestExpiredRingEntriesAreNotReplayed(t *testing.T) {
	clock := newMockClock()
	bus := NewBus(5*time.Minute, clock.Now)

	bus.Publish("test.old", map[string]int{"k": 1})
	clock.Advance(5*time.Minute + time.Second)
	bus.Publish("test.fresh", map[string]int{"k": 2})

	for _, afterID := range []int64{0, 1} {
		ctx, cancel := context.WithCancel(context.Background())
		sub := bus.Subscribe(ctx, afterID)

		ev, ok := sub.Next()
		if !ok {
			t.Fatalf("Next() = false for afterID %d, want the fresh event", afterID)
		}
		if ev.ID != 2 || ev.Name != "test.fresh" {
			t.Errorf("afterID %d delivered {ID %d, Name %q}, want only {ID 2, Name %q}: the expired event must be skipped",
				afterID, ev.ID, ev.Name, "test.fresh")
		}
		cancel()
		if _, ok := sub.Next(); ok {
			t.Errorf("afterID %d: Next() = true after cancel, want false", afterID)
		}
	}
}

// TestContextCancelEndsNext pins the fourth mandatory behavior: a
// blocked Next() returns false once the subscription context is
// canceled, and stays false on later calls.
func TestContextCancelEndsNext(t *testing.T) {
	bus := NewBus(5*time.Minute, time.Now)

	ctx, cancel := context.WithCancel(t.Context())
	sub := bus.Subscribe(ctx, 0)

	// Next() blocks because nothing was published; the cancel must
	// release it. t.Fatalf is not allowed on this goroutine, so the
	// result travels back over a channel.
	result := make(chan bool, 1)
	go func() {
		_, ok := sub.Next()
		result <- ok
	}()

	cancel()
	select {
	case ok := <-result:
		if ok {
			t.Error("Next() = true after ctx cancel, want false")
		}
	case <-time.After(testTimeout):
		t.Fatal("Next() did not return within the timeout after ctx cancel")
	}

	if _, ok := sub.Next(); ok {
		t.Error("Next() = true on the call after cancel, want false (repeated calls stay false)")
	}
}

// TestSubscribersAdvanceIndependently pins the fifth mandatory
// behavior: every subscription walks the ring with its own cursor, so
// one subscriber consuming events never skips or reorders them for
// another, and a mid-stream afterID only yields newer events.
func TestSubscribersAdvanceIndependently(t *testing.T) {
	bus := NewBus(5*time.Minute, time.Now)

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	subA := bus.Subscribe(ctx, 0)
	subB := bus.Subscribe(ctx, 0)

	bus.Publish("test.one", nil)
	bus.Publish("test.two", nil)

	// A takes one step; B must still start from the first event.
	a1, ok := subA.Next()
	if !ok || a1.ID != 1 {
		t.Fatalf("subscriber A first Next() = {ID %d, ok %v}, want {ID 1, ok true}", a1.ID, ok)
	}
	b1, ok := subB.Next()
	if !ok || b1.ID != 1 {
		t.Fatalf("subscriber B first Next() = {ID %d, ok %v}, want {ID 1, ok true}: A's progress must not advance B", b1.ID, ok)
	}
	// B overtakes A, then A catches up; both see the full sequence.
	b2, ok := subB.Next()
	if !ok || b2.ID != 2 {
		t.Fatalf("subscriber B second Next() = {ID %d, ok %v}, want {ID 2, ok true}", b2.ID, ok)
	}
	a2, ok := subA.Next()
	if !ok || a2.ID != 2 {
		t.Fatalf("subscriber A second Next() = {ID %d, ok %v}, want {ID 2, ok true}", a2.ID, ok)
	}

	// A subscriber joining mid-stream with afterID gets only the newer
	// events, never the ones its cursor starts past.
	subC := bus.Subscribe(ctx, 1)
	c1, ok := subC.Next()
	if !ok || c1.ID != 2 {
		t.Fatalf("subscriber C first Next() = {ID %d, ok %v}, want {ID 2, ok true}", c1.ID, ok)
	}
}
