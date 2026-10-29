package api

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	apigen "github.com/welpher/lumemor/server/internal/api/gen"
	"github.com/welpher/lumemor/server/internal/events"
)

// sseHeartbeatInterval is the pause between keepalive comment blocks
// on the event stream, pinned at 15 seconds by the API contract. It is
// a package-level variable only so the heartbeat test can shorten it.
var sseHeartbeatInterval = 15 * time.Second

// GetEvents implements GET /api/v1/events: the SSE activity stream.
// It subscribes to the bus with the Last-Event-ID header as the
// resumption cursor -- the retention ring replays everything newer,
// events that fell out of the ring are skipped -- then follows live
// publishes until the client disconnects.
//
// NOTE: deliberately unauthenticated for now (M2a Task 2). Task 3
// introduces the RequireSession middleware and the authed route group
// and moves this route into it, which puts the stream behind auth
// without changing this handler.
func (s *server) GetEvents(w http.ResponseWriter, r *http.Request, params apigen.GetEventsParams) {
	if s.events == nil {
		// The router is wired without a bus; production wiring always
		// provides one. Report the documented 500 ("the event stream
		// could not be started") rather than panicking on nil.
		s.writeError(w, http.StatusInternalServerError, apigen.INTERNAL,
			"event stream is not available", nil)
		return
	}

	var afterID int64
	if params.LastEventID != nil {
		id, err := strconv.ParseInt(strings.TrimSpace(*params.LastEventID), 10, 64)
		// Browser EventSource clients only ever echo back ids this
		// server wrote, so a malformed or negative value means a
		// hand-rolled client; refuse it instead of guessing a replay
		// position.
		if err != nil || id < 0 {
			fields := []interface{}{"Last-Event-ID"}
			details := map[string]interface{}{"fields": fields}
			s.writeError(w, http.StatusBadRequest, apigen.VALIDATIONFAILED,
				"Last-Event-ID must be a non-negative integer", &details)
			return
		}
		afterID = id
	}

	// The stream is open from here on: no envelope can follow, so
	// every later failure ends the stream with a log line instead.
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusOK)
	// A writer without Flush support (an exotic middleware) still gets
	// correct bytes, just buffered; the standard server and the test
	// recorder both flush.
	flusher, _ := w.(http.Flusher)
	flush := func() {
		if flusher != nil {
			flusher.Flush()
		}
	}

	// The subscription lives on the request context: a client
	// disconnect cancels it, which ends the feed below.
	subscription := s.events.Subscribe(r.Context(), afterID)

	// deliver bridges the blocking Subscription.Next loop into the
	// select below, where heartbeats compete with events. Closing done
	// (on handler return) releases the goroutine if it is mid-send;
	// a Next still parked in the bus is released by the context
	// cancel that accompanies every real disconnect.
	deliver := make(chan events.Event)
	done := make(chan struct{})
	defer close(done)
	go func() {
		for {
			event, ok := subscription.Next()
			if !ok {
				close(deliver)
				return
			}
			select {
			case deliver <- event:
			case <-done:
				return
			}
		}
	}()

	heartbeat := time.NewTicker(sseHeartbeatInterval)
	defer heartbeat.Stop()

	for {
		select {
		case event, ok := <-deliver:
			if !ok {
				// The subscription ended: the request context was
				// canceled (client gone).
				return
			}
			if err := writeEvent(w, event); err != nil {
				s.logStreamWrite(r, err)
				return
			}
			flush()
		case <-heartbeat.C:
			// Comment block: ignored by every SSE client, but it keeps
			// proxies from timing the connection out.
			if _, err := io.WriteString(w, ": keepalive\n\n"); err != nil {
				s.logStreamWrite(r, err)
				return
			}
			flush()
		case <-r.Context().Done():
			return
		}
	}
}

// writeEvent renders one event as an SSE block: the id, event and
// data lines followed by a blank line. json.Marshal never emits raw
// newlines, so the payload is always a single data line.
func writeEvent(w io.Writer, event events.Event) error {
	_, err := fmt.Fprintf(w, "id: %d\nevent: %s\ndata: %s\n\n", event.ID, event.Name, event.Payload)
	return err
}

// logStreamWrite records a failed stream write. Mid-stream write
// errors are routine -- the usual cause is the client vanishing
// between events -- so they are logged and the handler returns; the
// writer itself never recovers, the stream simply ends.
func (s *server) logStreamWrite(r *http.Request, err error) {
	s.logger.Info("event stream write failed", "error", err, "method", r.Method, "path", r.URL.Path)
}
