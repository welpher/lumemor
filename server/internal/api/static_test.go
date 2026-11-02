package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// newStaticRouter builds a router whose first-run state comes from an
// injected stub instead of the database, so the routing tests below need
// neither a pool nor a container runtime.
func newStaticRouter(initialized bool) http.Handler {
	return NewRouter(&Deps{
		Version:     testVersion,
		Initialized: func(context.Context) bool { return initialized },
	})
}

// TestSetupPage pins the embedded wizard page: a HTML document with a
// title, labels for every input, served without touching the database.
func TestSetupPage(t *testing.T) {
	router := newStaticRouter(false)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/setup", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /setup status = %d, want %d", rec.Code, http.StatusOK)
	}
	if got := rec.Header().Get("Content-Type"); !strings.Contains(got, "text/html") {
		t.Errorf("GET /setup content-type = %q, want it to contain %q", got, "text/html")
	}
	for _, want := range []string{"setup", "<title>", "<label"} {
		if !strings.Contains(rec.Body.String(), want) {
			t.Errorf("GET /setup body does not contain %q", want)
		}
	}
}

// TestRootRedirectsToSetupBeforeInitialization pins the first-run
// redirect: an instance without an administrator sends browsers to the
// wizard instead of the web app.
func TestRootRedirectsToSetupBeforeInitialization(t *testing.T) {
	router := newStaticRouter(false)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if rec.Code != http.StatusFound {
		t.Fatalf("GET / status = %d, want %d", rec.Code, http.StatusFound)
	}
	if got := rec.Header().Get("Location"); got != "/setup" {
		t.Errorf("GET / Location = %q, want %q", got, "/setup")
	}
}

// TestRootServesSPAAfterInitialization pins the web app shell an
// initialized instance serves on the root path: the SvelteKit
// single-page app's index document from server/internal/web (make
// web-build stages the adapter output its embed directive requires).
func TestRootServesSPAAfterInitialization(t *testing.T) {
	router := newStaticRouter(true)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("GET / status = %d, want %d", rec.Code, http.StatusOK)
	}
	if got := rec.Header().Get("Content-Type"); !strings.Contains(got, "text/html") {
		t.Errorf("GET / content-type = %q, want it to contain %q", got, "text/html")
	}
	for _, want := range []string{`<div id="app"`, "_app/"} {
		if !strings.Contains(rec.Body.String(), want) {
			t.Errorf("GET / body does not contain %q", want)
		}
	}
}

// TestUnknownPathServesSPAShell pins the client-side routing fallback:
// a GET on a route the server does not know (a client-side route)
// serves the app shell instead of 404ing, letting the SPA router
// decide the outcome.
func TestUnknownPathServesSPAShell(t *testing.T) {
	router := newStaticRouter(true)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/timeline", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /timeline status = %d, want %d", rec.Code, http.StatusOK)
	}
	if !strings.Contains(rec.Header().Get("Content-Type"), "text/html") {
		t.Errorf("GET /timeline content-type = %q, want it to contain %q", rec.Header().Get("Content-Type"), "text/html")
	}
	if !strings.Contains(rec.Body.String(), `<div id="app"`) {
		t.Error("GET /timeline body does not contain the SPA root div")
	}
}
