// The release checker's tests: the outbound discipline (internal
// targets refused, redirects never followed, a local fixture readable
// through the permissive policy the tests pin), the tag read of both
// document shapes, the verdicts, the persisted answer across a
// restart, and the silence of every failure. The database is a
// throwaway container with the schema applied, the same harness the
// rest of the server suite uses.
package updatecheck

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/welpher/lumemor/server/internal/db"
	"github.com/welpher/lumemor/server/internal/db/dbtest"
	dbgen "github.com/welpher/lumemor/server/internal/db/gen"
)

// quietLogger keeps the checker's per-attempt lines out of the test
// output.
func quietLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// newCheckEnv provisions the checker's environment: a migrated
// throwaway database and the queries over it.
func newCheckEnv(t *testing.T) (*pgxpool.Pool, *dbgen.Queries) {
	t.Helper()
	pool := dbtest.New(t)
	if err := db.Migrate(context.Background(), pool); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
	return pool, dbgen.New(pool)
}

// localClient is the permissive client the tests point at their own
// fixture servers: the same production client with the internal-target
// refusal turned off, so the refusal itself stays pinned separately.
func localClient() *http.Client {
	return NewClient(ClientPolicy{RefuseInternal: false})
}

// feedServer serves a fixed status and body for the checker's probe.
func feedServer(t *testing.T, status int, body string) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = io.WriteString(w, body)
	}))
	t.Cleanup(server.Close)
	return server
}

// saveSettings writes the check's settings row.
func saveSettings(t *testing.T, queries *dbgen.Queries, settings db.UpdateCheckSettings) {
	t.Helper()
	if err := db.SaveUpdateCheckSettings(context.Background(), queries, settings); err != nil {
		t.Fatalf("save the settings: %v", err)
	}
}

// TestClientRefusesInternalTargets pins the SSRF discipline: the
// production policy refuses a loopback fixture before any byte moves.
func TestClientRefusesInternalTargets(t *testing.T) {
	server := feedServer(t, http.StatusOK, `{"tag":"v9.9.9"}`)
	client := NewClient(ClientPolicy{RefuseInternal: true})
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, server.URL, nil)
	if err != nil {
		t.Fatalf("build the request: %v", err)
	}
	response, err := client.Do(req)
	if response != nil {
		_ = response.Body.Close()
	}
	if err == nil {
		t.Fatal("the production client reached a loopback target, want a refusal")
	}
	if !errors.Is(err, ErrInternalTarget) {
		t.Fatalf("client error = %v, want ErrInternalTarget", err)
	}
}

// TestPublicIP pins the classification the dial guard runs: the
// internal ranges refused, ordinary public addresses admitted.
func TestPublicIP(t *testing.T) {
	internal := []string{"127.0.0.1", "10.1.2.3", "172.16.0.1", "192.168.1.20", "169.254.1.1",
		"100.64.0.1", "0.0.0.0", "224.0.0.1", "::1", "fc00::1", "fe80::1"}
	for _, address := range internal {
		ip := parseIP(t, address)
		if publicIP(ip) {
			t.Errorf("publicIP(%s) = true, want false", address)
		}
	}
	public := []string{"8.8.8.8", "1.1.1.1", "140.82.121.4", "2606:4700:4700::1111"}
	for _, address := range public {
		ip := parseIP(t, address)
		if !publicIP(ip) {
			t.Errorf("publicIP(%s) = false, want true", address)
		}
	}
}

// TestCheckNowRecordsTheTag pins a successful probe: the tag rides the
// state, the comparison names the newer release, and a FRESH checker
// (the restart) reads the same answer back.
func TestCheckNowRecordsTheTag(t *testing.T) {
	_, queries := newCheckEnv(t)
	server := feedServer(t, http.StatusOK, `{"tag":"v0.5.0"}`)
	saveSettings(t, queries, db.UpdateCheckSettings{Enabled: true, Endpoint: server.URL, IntervalHours: 24})

	checker := New(Options{Queries: queries, Logger: quietLogger(), Client: localClient(), Version: "0.4.1"})
	snapshot := checker.CheckNow(context.Background())
	if snapshot.Status != StatusUpdateAvailable {
		t.Fatalf("status = %q, want %q", snapshot.Status, StatusUpdateAvailable)
	}
	if snapshot.LatestVersion != "v0.5.0" {
		t.Errorf("latestVersion = %q, want v0.5.0", snapshot.LatestVersion)
	}
	if snapshot.CheckedAt == nil {
		t.Fatal("checkedAt = nil, want the probe's stamp")
	}

	restarted := New(Options{Queries: queries, Logger: quietLogger(), Client: localClient(), Version: "0.4.1"})
	again := restarted.Snapshot(context.Background())
	if again.Status != StatusUpdateAvailable || again.LatestVersion != "v0.5.0" {
		t.Errorf("after restart = %+v, want the persisted update_available answer", again)
	}
}

// TestCheckNowReadsTheTagNameShape pins the release-API spelling of
// the tag, and the up-to-date verdict for an equal tag.
func TestCheckNowReadsTheTagNameShape(t *testing.T) {
	_, queries := newCheckEnv(t)
	server := feedServer(t, http.StatusOK, `{"tag_name":"v0.4.1","html_url":"https://example.net/release"}`)
	saveSettings(t, queries, db.UpdateCheckSettings{Enabled: true, Endpoint: server.URL, IntervalHours: 24})

	checker := New(Options{Queries: queries, Logger: quietLogger(), Client: localClient(), Version: "0.4.1"})
	snapshot := checker.CheckNow(context.Background())
	if snapshot.Status != StatusUpToDate {
		t.Fatalf("status = %q, want %q", snapshot.Status, StatusUpToDate)
	}
	if snapshot.LatestVersion != "v0.4.1" {
		t.Errorf("latestVersion = %q, want v0.4.1", snapshot.LatestVersion)
	}
}

// TestCheckNowFailureStaysSilent pins the silent failure: a non-200,
// an unreadable document and a document without a tag all land on
// `unknown` -- 200-shaped state, no message, the stamp recorded.
func TestCheckNowFailureStaysSilent(t *testing.T) {
	cases := []struct {
		name   string
		status int
		body   string
		// wantLatest keeps the tag a successful probe named even when
		// the comparison cannot rank it: the face reports what the
		// feed said, never a verdict it cannot support.
		wantLatest string
	}{
		{"server error", http.StatusInternalServerError, "boom", ""},
		{"not json", http.StatusOK, "<html>not a release feed</html>", ""},
		{"no tag", http.StatusOK, `{"message":"nothing here"}`, ""},
		{"uncomparable tag", http.StatusOK, `{"tag":"nightly"}`, "nightly"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, queries := newCheckEnv(t)
			server := feedServer(t, tc.status, tc.body)
			saveSettings(t, queries, db.UpdateCheckSettings{Enabled: true, Endpoint: server.URL, IntervalHours: 24})

			checker := New(Options{Queries: queries, Logger: quietLogger(), Client: localClient(), Version: "0.4.1"})
			snapshot := checker.CheckNow(context.Background())
			if snapshot.Status != StatusUnknown {
				t.Fatalf("status = %q, want %q", snapshot.Status, StatusUnknown)
			}
			if snapshot.LatestVersion != tc.wantLatest {
				t.Errorf("latestVersion = %q, want %q", snapshot.LatestVersion, tc.wantLatest)
			}
			if snapshot.CheckedAt == nil {
				t.Error("checkedAt = nil, want the failed probe's stamp")
			}
		})
	}
}

// TestCheckNowRedirectIsNotFollowed pins the redirect refusal: the
// 3xx is the target's answer, and the probe reports unknown.
func TestCheckNowRedirectIsNotFollowed(t *testing.T) {
	_, queries := newCheckEnv(t)
	target := feedServer(t, http.StatusOK, `{"tag":"v9.9.9"}`)
	redirector := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL, http.StatusFound)
	}))
	t.Cleanup(redirector.Close)
	saveSettings(t, queries, db.UpdateCheckSettings{Enabled: true, Endpoint: redirector.URL, IntervalHours: 24})

	checker := New(Options{Queries: queries, Logger: quietLogger(), Client: localClient(), Version: "0.4.1"})
	snapshot := checker.CheckNow(context.Background())
	if snapshot.Status != StatusUnknown {
		t.Fatalf("status = %q, want %q (the redirect must not be followed)", snapshot.Status, StatusUnknown)
	}
}

// TestCheckNowDisabledProbesNothing pins both off switches: the
// settings switch and the deployment switch answer `disabled` without
// touching the network (the fixture counts its hits).
func TestCheckNowDisabledProbesNothing(t *testing.T) {
	for _, tc := range []struct {
		name       string
		settings   bool
		deployment bool
	}{
		{"settings switch off", false, false},
		{"deployment switch off", true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, queries := newCheckEnv(t)
			hits := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				hits++
				_, _ = io.WriteString(w, `{"tag":"v9.9.9"}`)
			}))
			t.Cleanup(server.Close)
			saveSettings(t, queries, db.UpdateCheckSettings{Enabled: tc.settings, Endpoint: server.URL, IntervalHours: 24})

			checker := New(Options{Queries: queries, Logger: quietLogger(), Client: localClient(),
				Version: "0.4.1", Disabled: tc.deployment})
			snapshot := checker.CheckNow(context.Background())
			if snapshot.Status != StatusDisabled {
				t.Fatalf("status = %q, want %q", snapshot.Status, StatusDisabled)
			}
			if snapshot.Enabled {
				t.Error("enabled = true, want false")
			}
			if hits != 0 {
				t.Errorf("the fixture was probed %d times, want 0", hits)
			}
		})
	}
}

// TestVerdictUncomparableVersionStaysUnknown pins the development
// build: a feed tag exists, but the running version cannot be ranked,
// so no claim is made.
func TestVerdictUncomparableVersionStaysUnknown(t *testing.T) {
	_, queries := newCheckEnv(t)
	server := feedServer(t, http.StatusOK, `{"tag":"v0.5.0"}`)
	saveSettings(t, queries, db.UpdateCheckSettings{Enabled: true, Endpoint: server.URL, IntervalHours: 24})

	checker := New(Options{Queries: queries, Logger: quietLogger(), Client: localClient(), Version: "dev"})
	snapshot := checker.CheckNow(context.Background())
	if snapshot.Status != StatusUnknown {
		t.Fatalf("status = %q, want %q", snapshot.Status, StatusUnknown)
	}
}

// TestCheckIfDueHonorsTheInterval pins the cadence: a recent answer
// skips the probe, a stale one runs it.
func TestCheckIfDueHonorsTheInterval(t *testing.T) {
	_, queries := newCheckEnv(t)
	hits := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits++
		_, _ = io.WriteString(w, `{"tag":"v0.5.0"}`)
	}))
	t.Cleanup(server.Close)
	saveSettings(t, queries, db.UpdateCheckSettings{Enabled: true, Endpoint: server.URL, IntervalHours: 24})

	now := time.Date(2026, 11, 14, 12, 0, 0, 0, time.UTC)
	recent := now.Add(-time.Hour)
	if err := db.SaveUpdateCheckState(context.Background(), queries, db.UpdateCheckState{CheckedAt: &recent, LatestTag: "v0.5.0"}); err != nil {
		t.Fatalf("seed the state: %v", err)
	}

	checker := New(Options{Queries: queries, Logger: quietLogger(), Client: localClient(),
		Version: "0.4.1", Now: func() time.Time { return now }})
	checker.checkIfDue(context.Background())
	if hits != 0 {
		t.Fatalf("a fresh answer was probed again (%d hits), want 0", hits)
	}

	stale := now.Add(-25 * time.Hour)
	if err := db.SaveUpdateCheckState(context.Background(), queries, db.UpdateCheckState{CheckedAt: &stale, LatestTag: "v0.5.0"}); err != nil {
		t.Fatalf("seed the stale state: %v", err)
	}
	checker.checkIfDue(context.Background())
	if hits != 1 {
		t.Fatalf("a stale answer was probed %d times, want 1", hits)
	}
}

// parseIP parses a fixture address, failing the test on a typo.
func parseIP(t *testing.T, address string) net.IP {
	t.Helper()
	parsed := net.ParseIP(address)
	if parsed == nil {
		t.Fatalf("parse fixture address %q", address)
	}
	return parsed
}
