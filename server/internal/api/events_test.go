package api

import (
	"bufio"
	"context"
	"net/http"
	"net/http/httptest"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	apigen "github.com/welpher/lumemor/server/internal/api/gen"
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

// getEvents opens the activity stream against the live test server
// with the session cookie attached.
func getEvents(t *testing.T, client *http.Client, server *httptest.Server, cookie *http.Cookie, lastEventID string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, server.URL+"/api/v1/events", nil)
	if err != nil {
		t.Fatalf("build stream request: %v", err)
	}
	req.AddCookie(cookie)
	if lastEventID != "" {
		req.Header.Set("Last-Event-ID", lastEventID)
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("GET /api/v1/events: %v", err)
	}
	return resp
}

// TestGetEvents walks the SSE endpoint behind the session middleware:
// the live stream is cookie-authenticated, the reconnect contract
// holds, malformed and duplicated Last-Event-ID headers answer the
// validation envelope, the keepalive comments keep flowing, and a
// router without a bus reports its documented 500.
func TestGetEvents(t *testing.T) {
	env := newAuthEnv(t, nil)
	env.createAdmin(t, "ada@example.com", testPassword)
	cookie, info := loginCookie(t, env.router, "ada@example.com", testPassword, "")

	server := httptest.NewServer(env.router)
	t.Cleanup(server.Close)
	client := server.Client()
	client.Timeout = streamTimeout

	t.Run("StreamAndReconnect", func(t *testing.T) {
		// The login already published session.started into the ring,
		// so the ids here are relative: capture what Publish returns
		// and assert the blocks carry exactly those.
		alphaID := env.bus.Publish("test.alpha", map[string]string{"k": "a"})
		betaID := env.bus.Publish("test.beta", map[string]string{"k": "b"})

		resp := getEvents(t, client, server, cookie, "")
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
		// The login's own session.started leads the ring: a fresh
		// subscriber sees it replayed too.
		body := bufio.NewReader(resp.Body)
		sessionData := `{"id":"` + info.Id.String() + `","kind":"web"}`
		for _, want := range []struct {
			id   string
			name string
			data string
		}{
			{"1", "session.started", sessionData},
			{strconv.FormatInt(alphaID, 10), "test.alpha", `{"k":"a"}`},
			{strconv.FormatInt(betaID, 10), "test.beta", `{"k":"b"}`},
		} {
			id, name, data := readSSEBlock(t, body)
			if id != want.id || name != want.name || data != want.data {
				t.Errorf("first stream block = {id %s, event %s, data %s}, want {id %s, event %s, data %s}",
					id, name, data, want.id, want.name, want.data)
			}
		}
		_ = resp.Body.Close()

		// The reconnect: publish an increment, then connect again with
		// Last-Event-ID naming the last block already seen. Only the
		// increment may arrive.
		gammaID := env.bus.Publish("test.gamma", map[string]string{"k": "c"})
		resp2 := getEvents(t, client, server, cookie, strconv.FormatInt(betaID, 10))

		body2 := bufio.NewReader(resp2.Body)
		id, name, data := readSSEBlock(t, body2)
		if want := strconv.FormatInt(gammaID, 10); id != want || name != "test.gamma" || data != `{"k":"c"}` {
			t.Errorf("reconnect first block = {id %s, event %s, data %s}, want only the increment {id %s, event %s, data %s}",
				id, name, data, want, "test.gamma", `{"k":"c"}`)
		}

		// The reconnected stream stays live: an event published now is
		// delivered without a new request.
		deltaID := env.bus.Publish("test.delta", map[string]string{"k": "d"})
		id, name, data = readSSEBlock(t, body2)
		if want := strconv.FormatInt(deltaID, 10); id != want || name != "test.delta" || data != `{"k":"d"}` {
			t.Errorf("live block after reconnect = {id %s, event %s, data %s}, want {id %s, event %s, data %s}",
				id, name, data, want, "test.delta", `{"k":"d"}`)
		}
		_ = resp2.Body.Close()
	})

	t.Run("Heartbeat", func(t *testing.T) {
		oldInterval := sseHeartbeatInterval
		sseHeartbeatInterval = 20 * time.Millisecond
		defer func() { sseHeartbeatInterval = oldInterval }()

		resp := getEvents(t, client, server, cookie, "")
		defer func() { _ = resp.Body.Close() }()

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
	})

	t.Run("RejectsInvalidLastEventID", func(t *testing.T) {
		for _, value := range []string{"not-a-number", "-1", "1.5"} {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/events", nil)
			req.AddCookie(cookie)
			req.Header.Set("Last-Event-ID", value)
			rec := httptest.NewRecorder()
			env.router.ServeHTTP(rec, req)

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
	})

	t.Run("RejectsDuplicatedLastEventID", func(t *testing.T) {
		// Two values of one header trip the generated binder; the
		// shared error envelope must answer, not the plain-text
		// default.
		req := httptest.NewRequest(http.MethodGet, "/api/v1/events", nil)
		req.AddCookie(cookie)
		req.Header.Add("Last-Event-ID", "1")
		req.Header.Add("Last-Event-ID", "2")
		rec := httptest.NewRecorder()
		env.router.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("duplicated Last-Event-ID: status = %d, want %d (body %q)", rec.Code, http.StatusBadRequest, rec.Body.String())
		}
		apiErr, fields := decodeError(t, rec)
		if apiErr.Code != apigen.VALIDATIONFAILED {
			t.Errorf("code = %q, want %q", apiErr.Code, apigen.VALIDATIONFAILED)
		}
		if !slices.Contains(fields, "Last-Event-ID") {
			t.Errorf("details.fields = %v, want it to contain %q", fields, "Last-Event-ID")
		}
	})

	t.Run("RequiresBus", func(t *testing.T) {
		// A router wired without a bus reports the documented 500 for
		// authenticated requests; the session lives in the shared
		// database, so the cookie transfers.
		noBus := NewRouter(&Deps{Pool: env.pool, Version: testVersion})
		rec := authedRequest(noBus, http.MethodGet, "/api/v1/events", cookie)

		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("GET /api/v1/events without a bus: status = %d, want %d", rec.Code, http.StatusInternalServerError)
		}
		apiErr, _ := decodeError(t, rec)
		if apiErr.Code != apigen.INTERNAL {
			t.Errorf("code = %q, want %q", apiErr.Code, apigen.INTERNAL)
		}
	})
}

// TestGetEventsWithoutSessionRepeatsThe401: the stream is part of the
// authed surface, and an SSE request with a bounded context that fails
// authentication must never start streaming.
func TestGetEventsWithoutSession(t *testing.T) {
	env := newAuthEnv(t, nil)
	env.createAdmin(t, "ada@example.com", testPassword)

	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
	defer cancel()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/events", nil).WithContext(ctx)
	rec := httptest.NewRecorder()
	env.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("GET /api/v1/events without a session: status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
	apiErr, _ := decodeError(t, rec)
	if apiErr.Code != apigen.AUTHINVALIDCREDENTIALS {
		t.Errorf("code = %q, want %q", apiErr.Code, apigen.AUTHINVALIDCREDENTIALS)
	}
}
