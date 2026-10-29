package api

import (
	"bufio"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	apigen "github.com/welpher/lumemor/server/internal/api/gen"
	"github.com/welpher/lumemor/server/internal/events"
)

// streamTimeout bounds the test HTTP client so a wedged stream fails
// the test instead of hanging it.
const streamTimeout = 10 * time.Second

// readSSEBlock reads lines from an SSE stream until one event block
// (id/event/data followed by a blank line) is complete and returns its
// fields. Keepalive comment lines are skipped wherever they appear.
func readSSEBlock(t *testing.T, r *bufio.Reader) (id, name, data string) {
	t.Helper()
	var gotID, gotName, gotData bool
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			t.Fatalf("read SSE line: %v (block so far: id=%q event=%q data=%q)", err, id, name, data)
		}
		line = strings.TrimSuffix(line, "\n")
		switch {
		case strings.HasPrefix(line, ":"):
			// Comment line (keepalive); not part of an event block.
			continue
		case line == "":
			if gotID && gotName && gotData {
				return id, name, data
			}
			// A blank line after a keepalive block; the next event
			// follows.
			continue
		case strings.HasPrefix(line, "id: "):
			gotID, id = true, line[len("id: "):]
		case strings.HasPrefix(line, "event: "):
			gotName, name = true, line[len("event: "):]
		case strings.HasPrefix(line, "data: "):
			gotData, data = true, line[len("data: "):]
		default:
			t.Fatalf("unexpected SSE line %q", line)
		}
	}
}

// newEventsServer builds a router over the given bus plus a running
// test server for streaming requests. The pool stays nil: /events
// never touches the database.
func newEventsServer(t *testing.T, bus *events.Bus) *httptest.Server {
	t.Helper()
	router := NewRouter(&Deps{Version: testVersion, Events: bus})
	server := httptest.NewServer(router)
	t.Cleanup(server.Close)
	client := server.Client()
	client.Timeout = streamTimeout
	return server
}

// TestGetEventsStream walks the reconnect contract end to end against
// a live server: the first request replays the retention ring, the
// second request connects with Last-Event-ID and receives only the
// increment, then follows live publishes.
func TestGetEventsStream(t *testing.T) {
	bus := events.NewBus(5*time.Minute, time.Now)
	server := newEventsServer(t, bus)
	client := server.Client()

	// Two events exist before any client connects: a fresh subscriber
	// must receive them from the ring, in publish order.
	if id := bus.Publish("test.alpha", map[string]string{"k": "a"}); id != 1 {
		t.Fatalf("first Publish id = %d, want 1", id)
	}
	if id := bus.Publish("test.beta", map[string]string{"k": "b"}); id != 2 {
		t.Fatalf("second Publish id = %d, want 2", id)
	}

	resp, err := client.Get(server.URL + "/api/v1/events")
	if err != nil {
		t.Fatalf("first GET /api/v1/events: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("first stream status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.Contains(ct, "text/event-stream") {
		t.Errorf("Content-Type = %q, want it to contain %q", ct, "text/event-stream")
	}
	if cc := resp.Header.Get("Cache-Control"); cc != "no-cache" {
		t.Errorf("Cache-Control = %q, want %q", cc, "no-cache")
	}

	// The blocks only reach the client because the handler flushes
	// after every event; without flushing this read would block.
	body := bufio.NewReader(resp.Body)
	for _, want := range []struct{ id, name, data string }{
		{"1", "test.alpha", `{"k":"a"}`},
		{"2", "test.beta", `{"k":"b"}`},
	} {
		id, name, data := readSSEBlock(t, body)
		if id != want.id || name != want.name || data != want.data {
			t.Errorf("first stream block = {id %s, event %s, data %s}, want {id %s, event %s, data %s}",
				id, name, data, want.id, want.name, want.data)
		}
	}
	resp.Body.Close()

	// The reconnect: publish an increment, then connect again with
	// Last-Event-ID naming the last block already seen. Only the
	// increment may arrive.
	bus.Publish("test.gamma", map[string]string{"k": "c"})
	req, err := http.NewRequest(http.MethodGet, server.URL+"/api/v1/events", nil)
	if err != nil {
		t.Fatalf("build reconnect request: %v", err)
	}
	req.Header.Set("Last-Event-ID", "2")
	resp2, err := client.Do(req)
	if err != nil {
		t.Fatalf("reconnect GET /api/v1/events: %v", err)
	}
	defer resp2.Body.Close()

	body2 := bufio.NewReader(resp2.Body)
	id, name, data := readSSEBlock(t, body2)
	if id != "3" || name != "test.gamma" || data != `{"k":"c"}` {
		t.Errorf("reconnect first block = {id %s, event %s, data %s}, want only the increment {id 3, event %s, data %s}",
			id, name, data, "test.gamma", `{"k":"c"}`)
	}

	// The reconnected stream stays live: an event published now is
	// delivered without a new request.
	bus.Publish("test.delta", map[string]string{"k": "d"})
	id, name, data = readSSEBlock(t, body2)
	if id != "4" || name != "test.delta" || data != `{"k":"d"}` {
		t.Errorf("live block after reconnect = {id %s, event %s, data %s}, want {id 4, event %s, data %s}",
			id, name, data, "test.delta", `{"k":"d"}`)
	}
	resp2.Body.Close()
}

// TestGetEventsHeartbeat verifies the keepalive comment line: with the
// heartbeat interval shortened, an idle stream emits ": keepalive"
// blocks so proxies keep the connection open.
func TestGetEventsHeartbeat(t *testing.T) {
	oldInterval := sseHeartbeatInterval
	sseHeartbeatInterval = 20 * time.Millisecond
	defer func() { sseHeartbeatInterval = oldInterval }()

	bus := events.NewBus(5*time.Minute, time.Now)
	server := newEventsServer(t, bus)
	client := server.Client()

	resp, err := client.Get(server.URL + "/api/v1/events")
	if err != nil {
		t.Fatalf("GET /api/v1/events: %v", err)
	}
	defer resp.Body.Close()

	body := bufio.NewReader(resp.Body)
	for {
		line, err := body.ReadString('\n')
		if err != nil {
			t.Fatalf("read heartbeat line: %v", err)
		}
		if strings.HasPrefix(line, ": keepalive") {
			return
		}
	}
}

// TestGetEventsRejectsInvalidLastEventID verifies a Last-Event-ID that
// is not a non-negative integer is refused with the shared validation
// envelope instead of being replayed from an arbitrary position.
func TestGetEventsRejectsInvalidLastEventID(t *testing.T) {
	router := NewRouter(&Deps{Version: testVersion, Events: events.NewBus(5*time.Minute, time.Now)})

	for _, value := range []string{"not-a-number", "-1", "1.5"} {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/events", nil)
		req.Header.Set("Last-Event-ID", value)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("Last-Event-ID %q: status = %d, want %d (body %q)", value, rec.Code, http.StatusBadRequest, rec.Body.String())
		}
		apiErr, fields := decodeError(t, rec)
		if apiErr.Code != apigen.VALIDATIONFAILED {
			t.Errorf("Last-Event-ID %q: code = %q, want %q", value, apiErr.Code, apigen.VALIDATIONFAILED)
		}
		if !slices.Contains(fields, "Last-Event-ID") {
			t.Errorf("Last-Event-ID %q: details.fields = %v, want it to contain %q", value, fields, "Last-Event-ID")
		}
	}
}

// TestGetEventsRequiresBus verifies the nil-bus guard: a router wired
// without an event bus reports the documented 500 rather than
// panicking on a nil dereference.
func TestGetEventsRequiresBus(t *testing.T) {
	router := NewRouter(&Deps{Version: testVersion})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/events", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("GET /api/v1/events without a bus: status = %d, want %d (body %q)",
			rec.Code, http.StatusInternalServerError, rec.Body.String())
	}
	apiErr, _ := decodeError(t, rec)
	if apiErr.Code != apigen.INTERNAL {
		t.Errorf("code = %q, want %q", apiErr.Code, apigen.INTERNAL)
	}
}
