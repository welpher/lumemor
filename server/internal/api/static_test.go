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
// wizard instead of the landing page.
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

// TestRootServesIndexAfterInitialization pins the landing page an
// initialized instance serves on the root path.
func TestRootServesIndexAfterInitialization(t *testing.T) {
	router := newStaticRouter(true)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("GET / status = %d, want %d", rec.Code, http.StatusOK)
	}
	if got := rec.Header().Get("Content-Type"); !strings.Contains(got, "text/html") {
		t.Errorf("GET / content-type = %q, want it to contain %q", got, "text/html")
	}
	for _, want := range []string{"lumemor is running", "<title>"} {
		if !strings.Contains(rec.Body.String(), want) {
			t.Errorf("GET / body does not contain %q", want)
		}
	}
}
