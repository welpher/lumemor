package api

import (
	"bufio"
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// testTimeoutShort is the handler timeout of the test router variant:
// 300x shorter than production's handlerTimeout, so the wrapper's
// behavior is observable in bounded time while the real wiring it
// exercises is the production one (only the duration differs).
const testTimeoutShort = 100 * time.Millisecond

// TestHandlerTimeout pins the per-request handler timeout wiring: the
// non-streaming route groups run inside http.TimeoutHandler and answer
// 503 when a handler outlives the bound, while the SSE activity stream
// is deliberately outside the wrapper and keeps streaming long past it.
func TestHandlerTimeout(t *testing.T) {
	env := newAuthEnv(t, nil)
	env.createAdmin(t, "ada@example.com", testPassword)

	// The short router serves the same database and bus with the
	// 100ms timeout. Its logger discards: the parked query below winds
	// down through the request context after the 503 is out, and its
	// internal-error log line would only add noise.
	short := newRouter(&Deps{
		Pool:    env.pool,
		Version: testVersion,
		Logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
		Events:  env.bus,
	}, testTimeoutShort)

	t.Run("SSEStreamOutlivesTheHandlerTimeout", func(t *testing.T) {
		cookie, _ := loginCookie(t, env.router, "ada@example.com", testPassword, "")

		server := httptest.NewServer(short)
		t.Cleanup(server.Close)
		client := server.Client()
		client.Timeout = streamTimeout

		resp := getEvents(t, client, server, cookie, "")
		defer func() { _ = resp.Body.Close() }()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("GET /api/v1/events: status = %d, want %d", resp.StatusCode, http.StatusOK)
		}
		if ct := resp.Header.Get("Content-Type"); !strings.Contains(ct, "text/event-stream") {
			t.Fatalf("Content-Type = %q, want it to contain %q", ct, "text/event-stream")
		}

		// Ticks published throughout the read window keep blocks
		// arriving; the publisher stops with the subtest.
		ticks := make(chan struct{})
		t.Cleanup(func() { close(ticks) })
		go func() {
			ticker := time.NewTicker(testTimeoutShort)
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					env.bus.Publish("test.tick", map[string]string{"k": "v"})
				case <-ticks:
					return
				}
			}
		}()

		// The stream is still delivering events more
		// than a second in -- 10x the test timeout, 30x nothing for a
		// route (wrongly) wrapped in the timeout wrapper: that would
		// buffer every write (no Flusher) and cut the stream at the
		// deadline, failing this read loop.
		body := bufio.NewReader(resp.Body)
		const window = 1200 * time.Millisecond
		start := time.Now()
		ticksSeen := 0
		for time.Since(start) < window {
			_, name, _ := readSSEBlock(t, body)
			if name == "test.tick" {
				ticksSeen++
			}
		}
		if ticksSeen < 6 {
			t.Errorf("stream delivered %d tick events in %v, want at least 6 (a wrapped stream would be long dead)", ticksSeen, window)
		}
	})

	t.Run("SlowNonStreamingHandlerAnswers503", func(t *testing.T) {
		// Park the login handler before it can decide anything: an
		// ACCESS EXCLUSIVE lock on users blocks the handler's user
		// lookup for as long as the lock holder wants, deterministically
		// past the 100ms handler timeout.
		lockTx, err := env.pool.Begin(context.Background())
		if err != nil {
			t.Fatalf("begin lock transaction: %v", err)
		}
		t.Cleanup(func() { _ = lockTx.Rollback(context.Background()) })
		if _, err := lockTx.Exec(context.Background(), "LOCK TABLE users IN ACCESS EXCLUSIVE MODE"); err != nil {
			t.Fatalf("lock users table: %v", err)
		}

		// The request's own context is the fallback bound that keeps
		// the unwrapped (failing) case from hanging the test forever.
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login",
			strings.NewReader(loginBody(t, "ada@example.com", testPassword))).WithContext(ctx)
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		start := time.Now()
		short.ServeHTTP(rec, req)
		elapsed := time.Since(start)

		if rec.Code != http.StatusServiceUnavailable {
			t.Fatalf("login with a parked handler: status = %d, want %d (body %q)", rec.Code, http.StatusServiceUnavailable, rec.Body.String())
		}
		// The 503 comes from the handler timeout, well
		// before the parked query resolves on its own at the request
		// context bound.
		if elapsed > 2*time.Second {
			t.Errorf("503 arrived after %v, want it within the 100ms handler timeout (the parked query resolves only at the 5s context bound)", elapsed)
		}
	})
}
