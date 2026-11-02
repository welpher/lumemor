package web

import (
	"io/fs"
	"net/http"
	"net/http/httptest"
	"path"
	"strings"
	"testing"
)

// The suite needs the staged build output: `make web-build` must have
// run. That dependency is structural -- without the staged tree the
// package does not even compile (the embed directive fails), so the
// tests cannot run against a missing app.

func get(handler http.Handler, target string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, target, nil))
	return rec
}

// TestRootServesShell pins the app shell contract: the root path serves
// the built index.html (the SPA root div mounts the client router) as
// HTML with the revalidate cache policy.
func TestRootServesShell(t *testing.T) {
	rec := get(Handler(), "/")

	if rec.Code != http.StatusOK {
		t.Fatalf("GET / status = %d, want %d", rec.Code, http.StatusOK)
	}
	if got := rec.Header().Get("Content-Type"); !strings.Contains(got, "text/html") {
		t.Errorf("GET / content-type = %q, want it to contain %q", got, "text/html")
	}
	if got := rec.Header().Get("Cache-Control"); got != revalidateCache {
		t.Errorf("GET / cache-control = %q, want %q", got, revalidateCache)
	}
	for _, want := range []string{`<div id="app"`, "_app/"} {
		if !strings.Contains(rec.Body.String(), want) {
			t.Errorf("GET / body does not contain %q", want)
		}
	}
}

// TestUnknownPathFallsBackToShell pins the client-side routing
// fallback: a GET path that matches no embedded file serves the shell,
// letting the SPA router decide the outcome.
func TestUnknownPathFallsBackToShell(t *testing.T) {
	for _, target := range []string{"/timeline", "/albums/some-id", "/login"} {
		rec := get(Handler(), target)
		if rec.Code != http.StatusOK {
			t.Fatalf("GET %s status = %d, want %d", target, rec.Code, http.StatusOK)
		}
		if got := rec.Header().Get("Content-Type"); !strings.Contains(got, "text/html") {
			t.Errorf("GET %s content-type = %q, want it to contain %q", target, got, "text/html")
		}
		if !strings.Contains(rec.Body.String(), `<div id="app"`) {
			t.Errorf("GET %s body does not contain the SPA root div", target)
		}
	}
}

// firstImmutableAsset returns one file path under _app/immutable/ from
// the staged build, so the asset test follows the adapter's actual
// output names instead of pinning a hash.
func firstImmutableAsset(t *testing.T) string {
	t.Helper()
	var found string
	err := fs.WalkDir(root, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if found == "" && !d.IsDir() && strings.HasPrefix(p, "_app/immutable/") {
			found = p
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walking the embedded build output: %v", err)
	}
	if found == "" {
		t.Fatal("no file under _app/immutable/ in the staged build output")
	}
	return found
}

// TestHashedAssetServesImmutable pins the immutable cache policy for
// the adapter's content-hashed output.
func TestHashedAssetServesImmutable(t *testing.T) {
	asset := firstImmutableAsset(t)
	rec := get(Handler(), "/"+asset)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /%s status = %d, want %d", asset, rec.Code, http.StatusOK)
	}
	if got := rec.Header().Get("Cache-Control"); got != immutableCache {
		t.Errorf("GET /%s cache-control = %q, want %q", asset, got, immutableCache)
	}
	if ext := path.Ext(asset); ext != "" && !strings.Contains(rec.Header().Get("Content-Type"), strings.TrimPrefix(ext, ".")) {
		t.Errorf("GET /%s content-type = %q, want it to name %q", asset, rec.Header().Get("Content-Type"), strings.TrimPrefix(ext, "."))
	}
}

// TestMissingAssetNotFound pins that a broken asset URL 404s instead
// of serving the HTML shell: a script or stylesheet fetched as HTML
// would break the app load.
func TestMissingAssetNotFound(t *testing.T) {
	rec := get(Handler(), "/_app/immutable/entry/nope.js")

	if rec.Code != http.StatusNotFound {
		t.Fatalf("GET /_app/immutable/entry/nope.js status = %d, want %d", rec.Code, http.StatusNotFound)
	}
	if strings.Contains(rec.Body.String(), `<div id="app"`) {
		t.Error("GET of a missing asset served the app shell instead of 404")
	}
}

// TestVersionMarkerRevalidates pins that the adapter's unhashed version
// marker shares the shell's revalidate policy, so clients detect new
// deploys.
func TestVersionMarkerRevalidates(t *testing.T) {
	rec := get(Handler(), "/"+versionPage)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /%s status = %d, want %d", versionPage, rec.Code, http.StatusOK)
	}
	if got := rec.Header().Get("Cache-Control"); got != revalidateCache {
		t.Errorf("GET /%s cache-control = %q, want %q", versionPage, got, revalidateCache)
	}
}

// TestNonGETNotAllowed pins the read-only surface: the SPA is served
// for GET/HEAD only.
func TestNonGETNotAllowed(t *testing.T) {
	rec := httptest.NewRecorder()
	Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/timeline", nil))

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("POST /timeline status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
	if got := rec.Header().Get("Allow"); got != "GET, HEAD" {
		t.Errorf("POST /timeline allow = %q, want %q", got, "GET, HEAD")
	}
}
