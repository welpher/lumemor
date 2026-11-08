package api

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	apigen "github.com/welpher/lumemor/server/internal/api/gen"
)

// TestMiddlewareChain pins the request pipeline of the server: every
// JSON body is capped at 1 MiB, every response carries the security
// headers (the root-level chain covers the API tree and the
// operational endpoints alike), and non-GET requests with a foreign
// Origin header are refused before any handler runs. The handlers
// themselves are irrelevant here -- a malformed login payload reaches
// none of the database code, so the router runs without a pool.
func TestMiddlewareChain(t *testing.T) {
	router := NewRouter(&Deps{Version: testVersion})
	postLogin := func(origin string, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		if origin != "" {
			req.Header.Set("Origin", origin)
		}
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return rec
	}

	t.Run("SecurityHeadersOnEveryResponse", func(t *testing.T) {
		// The chain lives at the router root: the API tree, the
		// operational endpoints, the wizard page and the SPA shell
		// (with its assets -- see static_test.go for the SPA-surface
		// half) all carry the two headers.
		rec := postLogin("", `not json`)

		if got := rec.Header().Get("X-Content-Type-Options"); got != "nosniff" {
			t.Errorf("X-Content-Type-Options = %q, want nosniff", got)
		}
		if got := rec.Header().Get("Referrer-Policy"); got != "strict-origin-when-cross-origin" {
			t.Errorf("Referrer-Policy = %q, want strict-origin-when-cross-origin", got)
		}

		// The operational endpoints share the root chain.
		rec = httptest.NewRecorder()
		router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))
		if got := rec.Header().Get("X-Content-Type-Options"); got != "nosniff" {
			t.Errorf("GET /healthz X-Content-Type-Options = %q, want nosniff (root-level chain)", got)
		}
		if got := rec.Header().Get("Referrer-Policy"); got != "strict-origin-when-cross-origin" {
			t.Errorf("GET /healthz Referrer-Policy = %q, want strict-origin-when-cross-origin", got)
		}

		// The first-run wizard page too.
		rec = httptest.NewRecorder()
		router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/setup", nil))
		if got := rec.Header().Get("X-Content-Type-Options"); got != "nosniff" {
			t.Errorf("GET /setup X-Content-Type-Options = %q, want nosniff (root-level chain)", got)
		}
		if got := rec.Header().Get("Referrer-Policy"); got != "strict-origin-when-cross-origin" {
			t.Errorf("GET /setup Referrer-Policy = %q, want strict-origin-when-cross-origin", got)
		}

		// The SPA client-side-route fallback (the catch-all) is a
		// page response like the shell itself.
		rec = httptest.NewRecorder()
		router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/login", nil))
		if got := rec.Header().Get("X-Content-Type-Options"); got != "nosniff" {
			t.Errorf("GET /login X-Content-Type-Options = %q, want nosniff (root-level chain)", got)
		}
		if got := rec.Header().Get("Referrer-Policy"); got != "strict-origin-when-cross-origin" {
			t.Errorf("GET /login Referrer-Policy = %q, want strict-origin-when-cross-origin", got)
		}
	})

	t.Run("OriginCheckNonGET", func(t *testing.T) {
		// A foreign Origin on a state-changing request is refused.
		rec := postLogin("https://evil.example", `not json`)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("cross-origin POST: status = %d, want %d", rec.Code, http.StatusForbidden)
		}

		// A same-origin Origin passes through to the handler (which
		// rejects the payload on its own).
		rec = postLogin("http://example.com", `not json`)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("same-origin POST: status = %d, want %d (handler reached)", rec.Code, http.StatusBadRequest)
		}

		// No Origin at all -- curl, the mobile app, any non-browser
		// client -- passes.
		rec = postLogin("", `not json`)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("origin-less POST: status = %d, want %d (handler reached)", rec.Code, http.StatusBadRequest)
		}

		// GET is exempt: reads are not CSRF surface. The probe uses an
		// unknown path so it stays off the database: passing the origin
		// check yields chi's 404, a (wrongly) refused GET would 403.
		req := httptest.NewRequest(http.MethodGet, "/api/v1/unknown-path", nil)
		req.Header.Set("Origin", "https://evil.example")
		rec = httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code == http.StatusForbidden {
			t.Fatalf("cross-origin GET: status = 403, want anything else (GET is exempt from the origin check)")
		}
	})

	t.Run("BodyLimitOneMiB", func(t *testing.T) {
		// A body over the cap is cut by the middleware's reader and
		// reported by the handler's decode as a 400 naming the body.
		rec := postLoginRaw(router, `{"email":"ada@example.com","password":"`+strings.Repeat("a", 1<<20)+`"}`, "")

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("oversized body: status = %d, want %d (body %q)", rec.Code, http.StatusBadRequest, rec.Body.String())
		}
		apiErr, fields := decodeError(t, rec)
		if apiErr.Code != apigen.VALIDATIONFAILED {
			t.Errorf("code = %q, want %q", apiErr.Code, apigen.VALIDATIONFAILED)
		}
		if !contains(fields, "body") {
			t.Errorf("details.fields = %v, want it to contain %q", fields, "body")
		}
	})
}

// TestAPIRoutesCoverContract guards the hand-mounted route groups
// against contract drift: the mounted (method, path) set under /api/v1
// must exactly match the current operations of the generated
// ServerInterface, so an endpoint added to the OpenAPI contract cannot
// silently stay unrouted (and answer 404), and a removed one cannot
// linger.
func TestAPIRoutesCoverContract(t *testing.T) {
	router := NewRouter(&Deps{Version: testVersion})

	got := make(map[string]bool)
	err := chi.Walk(router, func(method, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		if strings.HasPrefix(route, "/api/v1/") {
			got[method+" "+route] = true
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk the router: %v", err)
	}

	want := []string{
		"POST /api/v1/auth/login",
		"POST /api/v1/auth/logout",
		"POST /api/v1/auth/pairing/consume",
		"POST /api/v1/auth/pairing/token",
		"GET /api/v1/auth/sessions",
		"DELETE /api/v1/auth/sessions/{id}",
		"GET /api/v1/events",
		"GET /api/v1/server/info",
		"POST /api/v1/server/setup/admin",
		"POST /api/v1/assets",
		"POST /api/v1/assets/batch",
		"GET /api/v1/assets/by-device-id/{deviceAssetId}",
		"GET /api/v1/assets/timeline",
		"GET /api/v1/assets/favorites",
		"GET /api/v1/assets/hidden",
		"GET /api/v1/assets/{id}",
		"PATCH /api/v1/assets/{id}",
		"GET /api/v1/assets/{id}/thumbnail",
		"GET /api/v1/assets/{id}/original",
		"GET /api/v1/assets/{id}/play",
		"GET /api/v1/trash",
		"POST /api/v1/trash/{id}/restore",
		"DELETE /api/v1/assets/{id}",
		"GET /api/v1/albums",
		"GET /api/v1/albums/path-completions",
		"GET /api/v1/albums/{albumId}",
		"POST /api/v1/albums",
		"PATCH /api/v1/albums/{albumId}",
		"DELETE /api/v1/albums/{albumId}",
		"DELETE /api/v1/albums/{albumId}/items",
		"PUT /api/v1/albums/{albumId}/audience",
		"PUT /api/v1/albums/{albumId}/timeline-preference",
		"DELETE /api/v1/albums/{albumId}/timeline-preference",
		"GET /api/v1/users",
		"GET /api/v1/users/me",
		"PATCH /api/v1/users/me",
		"PUT /api/v1/users/me/password",
		"GET /api/v1/users/me/sessions",
		"DELETE /api/v1/users/me/sessions/{id}",
		"GET /api/v1/people",
		"GET /api/v1/people/unnamed",
		"GET /api/v1/people/{clusterId}",
		"PUT /api/v1/people/{clusterId}/label",
		"DELETE /api/v1/people/{clusterId}/label",
		"PUT /api/v1/people/{clusterId}/hidden",
		"POST /api/v1/people/merge",
		"POST /api/v1/faces/{faceId}/move",
		"GET /api/v1/faces/{faceId}/thumbnail",
		"GET /api/v1/places",
		"GET /api/v1/tags",
		"POST /api/v1/tags",
		"PATCH /api/v1/tags/{tagId}",
		"DELETE /api/v1/tags/{tagId}",
		"POST /api/v1/tags/{tagId}/assets",
		"DELETE /api/v1/tags/{tagId}/assets",
		"GET /api/v1/search",
		"GET /api/v1/admin/users",
		"POST /api/v1/admin/users",
		"PATCH /api/v1/admin/users/{id}",
		"POST /api/v1/admin/models/download",
		"GET /api/v1/admin/models",
		"PATCH /api/v1/admin/models/{id}",
		"POST /api/v1/admin/models/{id}/rerun",
		"POST /api/v1/admin/models/scan",
		"POST /api/v1/admin/places/reresolve",
		"GET /api/v1/admin/jobs",
		"GET /api/v1/admin/settings/storage",
		"PUT /api/v1/admin/settings/storage",
		"GET /api/v1/admin/settings/trash",
		"PUT /api/v1/admin/settings/trash",
		"GET /api/v1/admin/settings/transcoding",
		"PUT /api/v1/admin/settings/transcoding",
		"GET /api/v1/admin/settings/mount-roots",
		"PUT /api/v1/admin/settings/mount-roots",
	}
	for _, route := range want {
		if !got[route] {
			t.Errorf("route %s is not mounted", route)
		}
		delete(got, route)
	}
	for route := range got {
		t.Errorf("route %s is mounted but not expected (contract drift?)", route)
	}

	// The mirror check: every operation the contract generates must
	// have a route, so the counts must agree.
	operations := reflect.TypeOf((*apigen.ServerInterface)(nil)).Elem().NumMethod()
	if operations != len(want) {
		t.Errorf("ServerInterface declares %d operations, want %d mounted routes (a contract change needs router.go updates)", operations, len(want))
	}
}
