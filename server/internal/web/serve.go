package web

import (
	"io"
	"io/fs"
	"net/http"
	"strings"
)

// immutableCache is the policy for the adapter's content-hashed output
// under _app/immutable/: the bytes behind a file name never change, so
// such responses are cacheable forever.
const immutableCache = "public, max-age=31536000, immutable"

// revalidateCache is the policy for the app shell and the adapter's
// version marker: both reference the hashed assets by name, so a stale
// copy keeps serving a stale app after a deploy. "no-cache" forces
// revalidation on every load.
const revalidateCache = "no-cache"

// indexPage is the SPA app shell served for the root path and every
// unknown non-API path: the client-side router decides the outcome.
const indexPage = "index.html"

// versionPage is the adapter's unhashed deploy marker; it shares the
// shell's revalidate policy for the same reason.
const versionPage = "_app/version.json"

// root is the embedded build output re-rooted at its contents, so
// "index.html" and "_app/..." open without the dist/ prefix.
var root = mustRoot()

func mustRoot() fs.FS {
	sub, err := fs.Sub(dist, "dist")
	if err != nil {
		// Unreachable for a successfully compiled embed: the directive
		// guarantees the dist subtree exists.
		panic("web: re-rooting embedded build output: " + err.Error())
	}
	return sub
}

// Handler serves the embedded single-page app:
//
//   - GET/HEAD paths under _app/ serve the embedded asset (the
//     content-hashed _app/immutable/ tree with the immutable cache
//     policy); a path matching nothing is a broken asset URL and 404s
//     rather than serving the HTML shell;
//   - any other embedded file (the shell itself, the version marker)
//     serves as itself;
//   - every other GET/HEAD path falls back to the shell, so unknown
//     routes belong to the client-side router (the SPA renders its own
//     404s).
//
// Other methods answer 405: the app is read-only over HTTP.
func Handler() http.Handler {
	return http.HandlerFunc(serveSPA)
}

func serveSPA(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	name := strings.TrimPrefix(r.URL.Path, "/")
	if strings.HasPrefix(name, "_app/") {
		if !serveFile(w, r, name) {
			http.NotFound(w, r)
		}
		return
	}
	if serveFile(w, r, name) {
		return
	}
	if !serveFile(w, r, indexPage) {
		// Unreachable for a successfully compiled embed of a real build:
		// the shell page exists. A staged tree without one is a build
		// packaging bug.
		http.Error(w, "web app shell page missing", http.StatusInternalServerError)
	}
}

// serveFile writes the named embedded file when it exists as a regular
// file, reporting whether it did. http.ServeContent picks the content
// type from the name and handles HEAD and range requests; embedded
// files carry no modification time, so no Last-Modified is emitted.
func serveFile(w http.ResponseWriter, r *http.Request, name string) bool {
	f, err := root.Open(name)
	if err != nil {
		return false
	}
	defer func() { _ = f.Close() }()
	info, err := f.Stat()
	if err != nil || info.IsDir() {
		return false
	}
	content, ok := f.(io.ReadSeeker)
	if !ok {
		return false
	}
	switch {
	case strings.HasPrefix(name, "_app/immutable/"):
		w.Header().Set("Cache-Control", immutableCache)
	case name == indexPage, name == versionPage:
		w.Header().Set("Cache-Control", revalidateCache)
	}
	http.ServeContent(w, r, name, info.ModTime(), content)
	return true
}
