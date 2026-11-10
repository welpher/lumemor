// Package events provides the in-process event bus of the lumemor
// server: producers publish domain events (session.started,
// session.revoked, auth.login_failed, pairing.token_created),
// consumers replay and follow them through the SSE activity stream
// (GET /api/v1/events).
//
// The bus is the plugin seam of the design spec: v1 deliberately keeps
// the surface to publish/subscribe with a retention ring -- no
// filtering, routing or external broker. Extend by adding event names,
// not interface surface.
package events

import (
	"context"
	"encoding/json"
	"sort"
	"sync"
	"time"
)

// Event is one activity record. Payload holds the JSON encoding of the
// value passed to Publish, produced once at publish time; the
// text/event-stream writer emits it verbatim as the data line.
type Event struct {
	ID      int64
	Name    string
	Payload []byte
}

// ringEntry is an Event stamped with its publish time, the input for
// the retention trim.
type ringEntry struct {
	event Event
	at    time.Time
}

// Bus is an in-process event bus with a time-bounded retention ring.
// The zero value is not usable; construct with NewBus.
//
// Concurrency: Publish and Subscribe may be called from any goroutine;
// every Subscription is used by one goroutine (its Next loop).
type Bus struct {
	mu   sync.Mutex
	cond *sync.Cond

	// retention is how long a published event stays replayable.
	retention time.Duration
	// now supplies the publish times that drive retention; injected
	// so tests advance it without sleeping.
	now func() time.Time

	// nextID is the id of the last published event; ids are assigned
	// monotonically starting at 1 and never reused, even after the
	// corresponding ring entry is trimmed.
	nextID int64
	// ring holds the retained entries in publish (id) order.
	ring []ringEntry
}

// NewBus builds a bus retaining published events for the given
// duration. now supplies the clock; a nil now falls back to time.Now.
func NewBus(retention time.Duration, now func() time.Time) *Bus {
	if now == nil {
		now = time.Now
	}
	bus := &Bus{
		retention: retention,
		now:       now,
	}
	bus.cond = sync.NewCond(&bus.mu)
	return bus
}

// Publish appends one event: payload is JSON-encoded here, exactly
// once, so subscribers and the SSE writer share the same bytes. It
// returns the assigned id, starting at 1 and strictly increasing.
// A payload that cannot be JSON-encoded is a producer bug; the event
// is dropped and 0 is returned rather than the stream being disturbed.
func (bus *Bus) Publish(name string, payload any) int64 {
	data, err := json.Marshal(payload)
	if err != nil {
		return 0
	}

	bus.mu.Lock()
	defer bus.mu.Unlock()

	bus.nextID++
	id := bus.nextID
	bus.ring = append(bus.ring, ringEntry{
		event: Event{ID: id, Name: name, Payload: data},
		at:    bus.now(),
	})
	bus.trimLocked()
	bus.cond.Broadcast()
	return id
}

// Subscribe opens a subscription delivering every event with an id
// greater than afterID, starting from the retained ring (entries that
// already fell out of the ring are skipped, per the reconnect
// contract) and then following live publishes.
//
// The subscription lives as long as ctx: Next returns false once ctx
// is canceled, and a small watcher goroutine (ended by the cancel or
// by the first false Next) exists to unblock a waiting Next. Abandon
// a subscription by canceling its context.
func (bus *Bus) Subscribe(ctx context.Context, afterID int64) *Subscription {
	bus.mu.Lock()
	// Trim first so entries that expired while nobody published or
	// subscribed are never replayed to a late subscriber.
	bus.trimLocked()
	bus.mu.Unlock()

	sub := &Subscription{
		bus:  bus,
		ctx:  ctx,
		last: afterID,
		stop: make(chan struct{}),
	}
	go sub.watch()
	return sub
}

// Subscription is one consumer's cursor over the bus. It is safe for
// exactly one goroutine to call Next.
type Subscription struct {
	bus *Bus
	ctx context.Context
	// last is the id of the most recent event this subscription
	// delivered; the next delivery is the first retained id above it.
	last int64
	// closeOnce guards stop, which ends the watcher goroutine.
	closeOnce sync.Once
	stop      chan struct{}
}

// watch wakes all waiting Next calls when ctx ends, so a blocked
// consumer returns instead of waiting for the next publish. It exits
// on either the cancel or the subscription being closed.
func (sub *Subscription) watch() {
	select {
	case <-sub.ctx.Done():
		sub.bus.wake()
	case <-sub.stop:
	}
}

// Next blocks until an event with an id above the last delivered one
// is available and returns it, or until the subscription context is
// canceled, in which case it returns false. Repeated calls after the
// cancellation keep returning false.
func (sub *Subscription) Next() (Event, bool) {
	bus := sub.bus
	bus.mu.Lock()
	defer bus.mu.Unlock()

	for {
		if sub.ctx.Err() != nil {
			sub.close()
			return Event{}, false
		}
		// Expired entries disappear between calls too, so a slow
		// consumer never receives events past the retention window.
		bus.trimLocked()

		// The ring is ordered by id; find the first retained entry
		// above this subscription's cursor. A cursor below the oldest
		// retained id resolves to index 0: fallen-out events are
		// skipped, not waited for.
		i := sort.Search(len(bus.ring), func(i int) bool {
			return bus.ring[i].event.ID > sub.last
		})
		if i < len(bus.ring) {
			event := bus.ring[i].event
			sub.last = event.ID
			return event, true
		}

		// Nothing to deliver yet; the publish (Broadcast) or the ctx
		// cancel (watch -> wake) releases the wait.
		bus.cond.Wait()
	}
}

// close ends the watcher goroutine; safe to call repeatedly.
func (sub *Subscription) close() {
	sub.closeOnce.Do(func() { close(sub.stop) })
}

// wake broadcasts to every Next call waiting on the condition.
func (bus *Bus) wake() {
	bus.mu.Lock()
	bus.cond.Broadcast()
	bus.mu.Unlock()
}

// trimLocked drops the ring entries older than the retention window.
// It is called on every Publish and Subscribe (and inside the Next
// loop), so stale entries cannot survive to a later subscriber even
// when nothing is published for a long time. The expired prefix is
// sliced off; append reallocates the backing array once the live
// slice outgrows it, so memory stays bounded by the retention window.
// bus.mu must be held.
func (bus *Bus) trimLocked() {
	if len(bus.ring) == 0 {
		return
	}
	cutoff := bus.now().Add(-bus.retention)
	drop := 0
	for drop < len(bus.ring) && bus.ring[drop].at.Before(cutoff) {
		drop++
	}
	if drop > 0 {
		bus.ring = bus.ring[drop:]
	}
}
