// The networking surface's API tests: the release check's admin guard,
// the cached read (no network access), the manual run's fresh answer
// and its silent failure, the three new settings pairs' defaults,
// round-trips and validation, and the map config's null/URL answers.
// The release feed is a local fixture server driven through the
// permissive client policy, so the suite never touches a real network.
package api

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	apigen "github.com/welpher/lumemor/server/internal/api/gen"
	"github.com/welpher/lumemor/server/internal/db"
	dbgen "github.com/welpher/lumemor/server/internal/db/gen"
	"github.com/welpher/lumemor/server/internal/updatecheck"
)

// networkingEnv provisions the environment: a migrated throwaway
// database, the router wired to a checker over a local fixture feed,
// and both an administrator and a plain signed-in user.
type networkingEnv struct {
	router http.Handler
	pool   *pgxpool.Pool
	hits   *atomic.Int32
	server *httptest.Server
	cookie *http.Cookie
	plain  *http.Cookie
}

// newNetworkingEnv provisions the environment. The fixture feed
// answers the configured body and counts its hits, so the cached
// read's no-network pin is observable.
func newNetworkingEnv(t *testing.T, status int, body string) *networkingEnv {
	t.Helper()
	pool := newSetupEnv(t)
	hits := &atomic.Int32{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = io.WriteString(w, body)
	}))
	t.Cleanup(server.Close)

	checker := updatecheck.New(updatecheck.Options{
		Queries: dbgen.New(pool),
		Logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
		Client:  updatecheck.NewClient(updatecheck.ClientPolicy{RefuseInternal: false}),
		Version: "0.4.1",
		Now:     func() time.Time { return time.Date(2026, 11, 14, 9, 0, 0, 0, time.UTC) },
	})
	router := NewRouter(&Deps{Pool: pool, Version: testVersion, UpdateChecks: checker})

	rec := postAdmin(t, router, createAdminBody(t, "ada@example.com", testPassword, nil))
	if rec.Code != http.StatusCreated {
		t.Fatalf("create admin: status = %d, want %d (body %q)", rec.Code, http.StatusCreated, rec.Body.String())
	}
	cookie, _ := loginCookie(t, router, "ada@example.com", testPassword, "networking-tests")
	_, plainCookie := seedPlainUser(t, pool, "grace@example.com")

	return &networkingEnv{router: router, pool: pool, hits: hits, server: server, cookie: cookie, plain: plainCookie}
}

// saveCheckSettings writes the release check's settings row.
func (e *networkingEnv) saveCheckSettings(t *testing.T, enabled bool) {
	t.Helper()
	if err := db.SaveUpdateCheckSettings(context.Background(), dbgen.New(e.pool), db.UpdateCheckSettings{
		Enabled:       enabled,
		Endpoint:      e.server.URL,
		IntervalHours: 24,
	}); err != nil {
		t.Fatalf("save the release check settings: %v", err)
	}
}

// do fires one request with the session cookie (nil = none).
func (e *networkingEnv) do(t *testing.T, method, path string, body string, cookie *http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, reader)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if cookie != nil {
		req.AddCookie(cookie)
	}
	rec := httptest.NewRecorder()
	e.router.ServeHTTP(rec, req)
	return rec
}

// decodeCheck decodes one release check response.
func decodeCheck(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d (body %q)", rec.Code, http.StatusOK, rec.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode the body: %v (body %q)", err, rec.Body.String())
	}
	return body
}

// TestUpdateCheckGuardsTheAdminRole pins the permission model: a
// signed-in plain user answers 403, no session answers 401, on both
// the read and the manual run.
func TestUpdateCheckGuardsTheAdminRole(t *testing.T) {
	env := newNetworkingEnv(t, http.StatusOK, `{"tag":"v9.9.9"}`)
	for _, path := range []string{"/api/v1/server/update-check"} {
		for _, method := range []string{http.MethodGet, http.MethodPost} {
			if rec := env.do(t, method, path, "", env.plain); rec.Code != http.StatusForbidden {
				t.Errorf("%s %s as a plain user: status = %d, want %d", method, path, rec.Code, http.StatusForbidden)
			}
			if rec := env.do(t, method, path, "", nil); rec.Code != http.StatusUnauthorized {
				t.Errorf("%s %s without a session: status = %d, want %d", method, path, rec.Code, http.StatusUnauthorized)
			}
		}
	}
}

// TestGetUpdateCheckReadsTheCache pins the read: it answers the
// persisted verdict and never probes (the fixture's hit count stays
// zero).
func TestGetUpdateCheckReadsTheCache(t *testing.T) {
	env := newNetworkingEnv(t, http.StatusOK, `{"tag":"v0.5.0"}`)
	env.saveCheckSettings(t, true)
	checkedAt := time.Date(2026, 11, 13, 9, 0, 0, 0, time.UTC)
	if err := db.SaveUpdateCheckState(context.Background(), dbgen.New(env.pool), db.UpdateCheckState{
		CheckedAt: &checkedAt, LatestTag: "v0.5.0",
	}); err != nil {
		t.Fatalf("seed the state: %v", err)
	}

	body := decodeCheck(t, env.do(t, http.MethodGet, "/api/v1/server/update-check", "", env.cookie))
	if body["status"] != "update_available" {
		t.Errorf("status = %v, want update_available", body["status"])
	}
	if body["latestVersion"] != "v0.5.0" {
		t.Errorf("latestVersion = %v, want v0.5.0", body["latestVersion"])
	}
	if body["currentVersion"] != "0.4.1" {
		t.Errorf("currentVersion = %v, want 0.4.1", body["currentVersion"])
	}
	if env.hits.Load() != 0 {
		t.Errorf("the cached read probed the feed %d times, want 0", env.hits.Load())
	}
}

// TestRunUpdateCheckProbes pins the manual run: one probe, the fresh
// verdict, the persisted stamp.
func TestRunUpdateCheckProbes(t *testing.T) {
	env := newNetworkingEnv(t, http.StatusOK, `{"tag":"v0.5.0"}`)
	env.saveCheckSettings(t, true)

	body := decodeCheck(t, env.do(t, http.MethodPost, "/api/v1/server/update-check", "", env.cookie))
	if body["status"] != "update_available" {
		t.Errorf("status = %v, want update_available", body["status"])
	}
	if body["checkedAt"] == nil {
		t.Error("checkedAt = null, want the probe's stamp")
	}
	if env.hits.Load() != 1 {
		t.Errorf("the manual run probed %d times, want 1", env.hits.Load())
	}
}

// TestRunUpdateCheckFailureStaysQuiet pins the silent failure: a
// failing feed answers 200 with unknown and no latest version.
func TestRunUpdateCheckFailureStaysQuiet(t *testing.T) {
	env := newNetworkingEnv(t, http.StatusInternalServerError, "boom")
	env.saveCheckSettings(t, true)

	body := decodeCheck(t, env.do(t, http.MethodPost, "/api/v1/server/update-check", "", env.cookie))
	if body["status"] != "unknown" {
		t.Errorf("status = %v, want unknown", body["status"])
	}
	if body["latestVersion"] != nil {
		t.Errorf("latestVersion = %v, want null", body["latestVersion"])
	}
}

// TestRunUpdateCheckDisabledProbesNothing pins the off switch: a
// disabled check answers `disabled` through the manual run and never
// touches the feed.
func TestRunUpdateCheckDisabledProbesNothing(t *testing.T) {
	env := newNetworkingEnv(t, http.StatusOK, `{"tag":"v9.9.9"}`)
	env.saveCheckSettings(t, false)

	body := decodeCheck(t, env.do(t, http.MethodPost, "/api/v1/server/update-check", "", env.cookie))
	if body["status"] != "disabled" {
		t.Errorf("status = %v, want disabled", body["status"])
	}
	if body["enabled"] != false {
		t.Errorf("enabled = %v, want false", body["enabled"])
	}
	if env.hits.Load() != 0 {
		t.Errorf("the disabled run probed the feed %d times, want 0", env.hits.Load())
	}
}

// TestUpdateCheckSettingsRoundTrip pins the defaults and a save.
func TestUpdateCheckSettingsRoundTrip(t *testing.T) {
	env := newNetworkingEnv(t, http.StatusOK, `{}`)

	rec := env.do(t, http.MethodGet, "/api/v1/admin/settings/update-check", "", env.cookie)
	if rec.Code != http.StatusOK {
		t.Fatalf("get: status = %d, want %d", rec.Code, http.StatusOK)
	}
	var defaults map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &defaults); err != nil {
		t.Fatalf("decode the defaults: %v", err)
	}
	if defaults["enabled"] != true || defaults["intervalHours"] != float64(24) ||
		defaults["endpoint"] != db.UpdateCheckEndpointDefault {
		t.Fatalf("defaults = %v, want the documented posture", defaults)
	}

	saved := env.do(t, http.MethodPut, "/api/v1/admin/settings/update-check",
		`{"enabled":false,"endpoint":"https://releases.example.net/latest.json","intervalHours":6}`, env.cookie)
	if saved.Code != http.StatusOK {
		t.Fatalf("put: status = %d, want %d (body %q)", saved.Code, http.StatusOK, saved.Body.String())
	}

	// The save lands as a row the next read (and the scheduler) sees.
	settings, err := db.UpdateCheckSettingsOf(context.Background(), dbgen.New(env.pool))
	if err != nil {
		t.Fatalf("read the saved settings: %v", err)
	}
	if settings.Enabled || settings.IntervalHours != 6 || settings.Endpoint != "https://releases.example.net/latest.json" {
		t.Errorf("saved settings = %+v, want the saved values", settings)
	}
}

// TestUpdateCheckSettingsValidation pins the refusals: a missing field,
// a non-URL endpoint and an interval outside 1-168.
func TestUpdateCheckSettingsValidation(t *testing.T) {
	env := newNetworkingEnv(t, http.StatusOK, `{}`)
	cases := []string{
		`{"endpoint":"https://example.net/x","intervalHours":24}`,
		`{"enabled":true,"intervalHours":24}`,
		`{"enabled":true,"endpoint":"ftp://example.net/x","intervalHours":24}`,
		`{"enabled":true,"endpoint":"not a url","intervalHours":24}`,
		`{"enabled":true,"endpoint":"https://example.net/x","intervalHours":0}`,
		`{"enabled":true,"endpoint":"https://example.net/x","intervalHours":169}`,
	}
	for _, body := range cases {
		rec := env.do(t, http.MethodPut, "/api/v1/admin/settings/update-check", body, env.cookie)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("PUT %s: status = %d, want %d (body %q)", body, rec.Code, http.StatusBadRequest, rec.Body.String())
		}
	}
	if rec := env.do(t, http.MethodPut, "/api/v1/admin/settings/update-check", `{"enabled":true,"endpoint":"https://example.net/x","intervalHours":24}`, env.plain); rec.Code != http.StatusForbidden {
		t.Errorf("save as a plain user: status = %d, want %d", rec.Code, http.StatusForbidden)
	}
}

// TestTilesSettingsRoundTrip pins the empty default, a save, the clear
// and the shape refusal.
func TestTilesSettingsRoundTrip(t *testing.T) {
	env := newNetworkingEnv(t, http.StatusOK, `{}`)

	rec := env.do(t, http.MethodGet, "/api/v1/admin/settings/tiles", "", env.cookie)
	var initial map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &initial); err != nil {
		t.Fatalf("decode the default: %v", err)
	}
	if initial["tilejsonUrl"] != "" {
		t.Fatalf("default tilejsonUrl = %v, want the empty string", initial["tilejsonUrl"])
	}

	if rec := env.do(t, http.MethodPut, "/api/v1/admin/settings/tiles", `{"tilejsonUrl":"http://192.168.1.20:8080/tilejson.json"}`, env.cookie); rec.Code != http.StatusOK {
		t.Fatalf("save: status = %d, want %d (body %q)", rec.Code, http.StatusOK, rec.Body.String())
	}
	mapConfig := env.do(t, http.MethodGet, "/api/v1/map/config", "", env.cookie)
	var config map[string]any
	if err := json.Unmarshal(mapConfig.Body.Bytes(), &config); err != nil {
		t.Fatalf("decode the map config: %v", err)
	}
	if config["tilejsonUrl"] != "http://192.168.1.20:8080/tilejson.json" {
		t.Errorf("map config = %v, want the saved source", config["tilejsonUrl"])
	}

	// The clear answers the empty string and the map config null.
	if rec := env.do(t, http.MethodPut, "/api/v1/admin/settings/tiles", `{"tilejsonUrl":""}`, env.cookie); rec.Code != http.StatusOK {
		t.Fatalf("clear: status = %d, want %d", rec.Code, http.StatusOK)
	}
	cleared := env.do(t, http.MethodGet, "/api/v1/map/config", "", env.cookie)
	if !strings.Contains(cleared.Body.String(), `"tilejsonUrl":null`) {
		t.Errorf("cleared map config = %q, want null", cleared.Body.String())
	}

	for _, body := range []string{`{}`, `{"tilejsonUrl":"ftp://example.net/tiles.json"}`, `{"tilejsonUrl":"nonsense"}`} {
		if rec := env.do(t, http.MethodPut, "/api/v1/admin/settings/tiles", body, env.cookie); rec.Code != http.StatusBadRequest {
			t.Errorf("PUT %s: status = %d, want %d", body, rec.Code, http.StatusBadRequest)
		}
	}
}

// TestMapConfigIsReadableByEveryMember pins the read's audience: the
// tile source is map chrome, so a plain signed-in member reads it
// (while the settings pair stays the administrator's).
func TestMapConfigIsReadableByEveryMember(t *testing.T) {
	env := newNetworkingEnv(t, http.StatusOK, `{}`)
	if rec := env.do(t, http.MethodGet, "/api/v1/map/config", "", env.plain); rec.Code != http.StatusOK {
		t.Errorf("map config as a plain user: status = %d, want %d", rec.Code, http.StatusOK)
	}
	if rec := env.do(t, http.MethodGet, "/api/v1/admin/settings/tiles", "", env.plain); rec.Code != http.StatusForbidden {
		t.Errorf("tile settings as a plain user: status = %d, want %d", rec.Code, http.StatusForbidden)
	}
	if rec := env.do(t, http.MethodGet, "/api/v1/map/config", "", nil); rec.Code != http.StatusUnauthorized {
		t.Errorf("map config without a session: status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

// TestModelSettingsRoundTrip pins the defaults (the switch on, the
// manifest's own hosts), the whole-object save (the hosts replaced
// wholesale, the mirror's host auto-joining the list) and the
// missing-field refusal.
func TestModelSettingsRoundTrip(t *testing.T) {
	env := newNetworkingEnv(t, http.StatusOK, `{}`)

	rec := env.do(t, http.MethodGet, "/api/v1/admin/settings/models", "", env.cookie)
	var initial map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &initial); err != nil {
		t.Fatalf("decode the default: %v", err)
	}
	if initial["autoDownload"] != true {
		t.Fatalf("default autoDownload = %v, want true", initial["autoDownload"])
	}
	hosts, ok := initial["allowedHosts"].([]any)
	if !ok || len(hosts) == 0 {
		t.Fatalf("default allowedHosts = %v, want the shipped list", initial["allowedHosts"])
	}
	if initial["mirrorBaseUrl"] != "" {
		t.Fatalf("default mirrorBaseUrl = %v, want the empty string", initial["mirrorBaseUrl"])
	}

	// The save replaces the hosts wholesale and the mirror's host
	// joins the saved list automatically: a mirror the list refuses
	// would fail on its own first hop.
	saved := env.do(t, http.MethodPut, "/api/v1/admin/settings/models",
		`{"autoDownload":false,"allowedHosts":["models.example.net"],"mirrorBaseUrl":"mirror.example.net"}`, env.cookie)
	if saved.Code != http.StatusOK {
		t.Fatalf("save: status = %d, want %d", saved.Code, http.StatusOK)
	}
	var answer apigen.ModelSettingsResponse
	if err := json.Unmarshal(saved.Body.Bytes(), &answer); err != nil {
		t.Fatalf("decode the save answer: %v", err)
	}
	if answer.AutoDownload {
		t.Error("autoDownload = true, want false after the save")
	}
	if len(answer.AllowedHosts) != 2 {
		t.Fatalf("saved hosts = %v, want the saved entry plus the mirror's", answer.AllowedHosts)
	}
	sorted := append([]string(nil), answer.AllowedHosts...)
	sort.Strings(sorted)
	if sorted[0] != "mirror.example.net" || sorted[1] != "models.example.net" {
		t.Errorf("saved hosts = %v, want the saved entry plus the mirror's host", sorted)
	}
	settings, err := db.ModelSettingsOf(context.Background(), dbgen.New(env.pool))
	if err != nil {
		t.Fatalf("read the saved policy: %v", err)
	}
	if settings.MirrorBaseUrl != "mirror.example.net" {
		t.Errorf("mirrorBaseUrl = %q, want mirror.example.net", settings.MirrorBaseUrl)
	}

	if rec := env.do(t, http.MethodPut, "/api/v1/admin/settings/models", `{"autoDownload":false}`, env.cookie); rec.Code != http.StatusBadRequest {
		t.Errorf("missing field: status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
	// A malformed host entry is refused with the field named; the
	// wildcard label is accepted (the shipped list carries one).
	if rec := env.do(t, http.MethodPut, "/api/v1/admin/settings/models",
		`{"autoDownload":false,"allowedHosts":["bad host*"],"mirrorBaseUrl":""}`, env.cookie); rec.Code != http.StatusBadRequest {
		t.Errorf("malformed host: status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
	saved2 := env.do(t, http.MethodPut, "/api/v1/admin/settings/models",
		`{"autoDownload":false,"allowedHosts":["*.hf.co","huggingface.co"],"mirrorBaseUrl":""}`, env.cookie)
	if saved2.Code != http.StatusOK {
		t.Fatalf("wildcard host save: status = %d, want %d (body %q)", saved2.Code, http.StatusOK, saved2.Body.String())
	}
}
