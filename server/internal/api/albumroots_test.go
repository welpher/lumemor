// The albums-as-sources path domain tests (E4): the mount-root
// allowlist admin settings (the default-closed posture, the shape
// validation, the clean-and-dedupe save), the source-path completion
// (the completion-domain-equals-validation-domain pin -- the
// endpoint never lists outside the allowlist -- plus the matching
// rules, the row bound and the user scoping), and the creation-time
// path validation matrix (every branch its own error code: not
// exists, not a directory, not readable, symlink escape, the managed
// tree, overlap and nesting, outside the roots).
package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"

	apigen "github.com/welpher/lumemor/server/internal/api/gen"
)

// mountRootsBody renders the mount-roots save body.
func mountRootsBody(roots ...string) string {
	quoted := make([]string, len(roots))
	for i, root := range roots {
		quoted[i] = `"` + root + `"`
	}
	return `{"roots":[` + strings.Join(quoted, ",") + `]}`
}

// setMountRoots saves the allowlist through the admin face and fails
// on anything but the 200 carrying the stored value.
func (e *assetEnv) setMountRoots(t *testing.T, cookie *http.Cookie, roots ...string) {
	t.Helper()
	rec := e.writeAlbumRaw(t, cookie, http.MethodPut, "/api/v1/admin/settings/mount-roots", mountRootsBody(roots...))
	if rec.Code != http.StatusOK {
		t.Fatalf("PUT /admin/settings/mount-roots status = %d, want %d (body %q)",
			rec.Code, http.StatusOK, rec.Body.String())
	}
}

// provisionMountRoot creates one external directory -- a sibling of
// the environment's data root, never inside it -- opens it as the
// instance's only mount root, and answers its path. The caller
// derives source directories under it.
func (e *assetEnv) provisionMountRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	e.setMountRoots(t, e.cookie, root)
	return root
}

// sourceDir creates one subdirectory of a mount root as a creation
// source. Tests that create several albums derive one sibling each:
// siblings never overlap, so every creation answers 201.
func sourceDir(t *testing.T, root, name string) string {
	t.Helper()
	dir := filepath.Join(root, name)
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatalf("create the source directory %s: %v", dir, err)
	}
	return dir
}

// completionsRaw fires GET /albums/path-completions with the prefix
// (a nil cookie sends it unauthenticated).
func (e *assetEnv) completionsRaw(t *testing.T, cookie *http.Cookie, prefix string) *httptest.ResponseRecorder {
	t.Helper()
	target := "/api/v1/albums/path-completions"
	if prefix != "" {
		target += "?prefix=" + url.QueryEscape(prefix)
	}
	req := httptest.NewRequest(http.MethodGet, target, nil)
	if cookie != nil {
		req.AddCookie(cookie)
	}
	rec := httptest.NewRecorder()
	e.router.ServeHTTP(rec, req)
	return rec
}

// completions decodes the 200 completion body.
func (e *assetEnv) completions(t *testing.T, cookie *http.Cookie, prefix string) []string {
	t.Helper()
	rec := e.completionsRaw(t, cookie, prefix)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /albums/path-completions?prefix=%q status = %d, want %d (body %q)",
			prefix, rec.Code, http.StatusOK, rec.Body.String())
	}
	var response apigen.AlbumPathCompletionsResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode completions body %q: %v", rec.Body.String(), err)
	}
	return response.Paths
}

// requireAlbumPathError asserts one path matrix branch: the 400
// carrying exactly the branch's error code.
func requireAlbumPathError(t *testing.T, rec *httptest.ResponseRecorder, code apigen.ErrorCode) {
	t.Helper()
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d (body %q)", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
	apiErr, _ := decodeError(t, rec)
	if apiErr.Code != code {
		t.Errorf("code = %q, want %q", apiErr.Code, code)
	}
}

// requirePaths asserts a completion answer exactly (order included).
func requirePaths(t *testing.T, got []string, want ...string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("completions = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("completions = %v, want %v", got, want)
		}
	}
}

// TestMountRootsSettings pins the allowlist admin face: the default
// is the empty list (directory album creation closed), the save
// validates the shape (absolute paths, the count and length bounds,
// the required field), cleans and deduplicates, and the read answers
// the stored value. The admin role gates both verbs.
func TestMountRootsSettings(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	_, graceCookie := env.createSecondUser(t, "mount-roots-grace@example.com")

	t.Run("DefaultIsEmpty", func(t *testing.T) {
		rec := env.writeAlbumRaw(t, env.cookie, http.MethodGet, "/api/v1/admin/settings/mount-roots", "")
		if rec.Code != http.StatusOK {
			t.Fatalf("GET /admin/settings/mount-roots status = %d, want %d (body %q)",
				rec.Code, http.StatusOK, rec.Body.String())
		}
		if body := strings.TrimSpace(rec.Body.String()); body != `{"roots":[]}` {
			t.Errorf("default body = %q, want the empty (never null) list", body)
		}
	})

	t.Run("SavesAndReadsBack", func(t *testing.T) {
		root := t.TempDir()
		second := t.TempDir()
		env.setMountRoots(t, env.cookie, root, second)
		rec := env.writeAlbumRaw(t, env.cookie, http.MethodGet, "/api/v1/admin/settings/mount-roots", "")
		var response apigen.MountRootsSettingsResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
			t.Fatalf("decode mount roots body %q: %v", rec.Body.String(), err)
		}
		if len(response.Roots) != 2 || response.Roots[0] != root || response.Roots[1] != second {
			t.Errorf("stored roots = %v, want [%s %s]", response.Roots, root, second)
		}
	})

	t.Run("CleansAndCollapsesDuplicates", func(t *testing.T) {
		root := t.TempDir()
		rec := env.writeAlbumRaw(t, env.cookie, http.MethodPut, "/api/v1/admin/settings/mount-roots",
			mountRootsBody(root+"/", root, root+"/./"))
		if rec.Code != http.StatusOK {
			t.Fatalf("PUT status = %d, want %d (body %q)", rec.Code, http.StatusOK, rec.Body.String())
		}
		var response apigen.MountRootsSettingsResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
			t.Fatalf("decode mount roots body %q: %v", rec.Body.String(), err)
		}
		if len(response.Roots) != 1 || response.Roots[0] != root {
			t.Errorf("stored roots = %v, want the cleaned [%s] (duplicates collapse)", response.Roots, root)
		}
	})

	t.Run("EmptyListCloses", func(t *testing.T) {
		env.setMountRoots(t, env.cookie)
		rec := env.writeAlbumRaw(t, env.cookie, http.MethodGet, "/api/v1/admin/settings/mount-roots", "")
		if body := strings.TrimSpace(rec.Body.String()); body != `{"roots":[]}` {
			t.Errorf("closed body = %q, want the empty list", body)
		}
	})

	t.Run("ValidatesShape", func(t *testing.T) {
		for _, body := range []string{
			`{}`,                   // the field must ride the body
			`{"roots":null}`,       // null reads as absent
			`{"roots":["Photos"]}`, // relative
			`{"roots":[""]}`,       // empty
			`{"roots":["` + strings.Repeat("a", 4097) + `"]}`, // past the bound
		} {
			rec := env.writeAlbumRaw(t, env.cookie, http.MethodPut, "/api/v1/admin/settings/mount-roots", body)
			requireValidationFields(t, rec, "roots")
		}
		over := make([]string, 33)
		for i := range over {
			over[i] = "/mnt/root-" + strings.Repeat("x", i+1)
		}
		rec := env.writeAlbumRaw(t, env.cookie, http.MethodPut, "/api/v1/admin/settings/mount-roots", mountRootsBody(over...))
		requireValidationFields(t, rec, "roots")

		// The malformed body answers the plain validation envelope
		// (the settings faces name no field for a body that does
		// not decode).
		rec = env.writeAlbumRaw(t, env.cookie, http.MethodPut, "/api/v1/admin/settings/mount-roots", `{"roots":[`)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("PUT malformed body status = %d, want %d (body %q)",
				rec.Code, http.StatusBadRequest, rec.Body.String())
		}
		apiErr, _ := decodeError(t, rec)
		if apiErr.Code != apigen.VALIDATIONFAILED {
			t.Errorf("PUT malformed body code = %q, want %q", apiErr.Code, apigen.VALIDATIONFAILED)
		}
	})

	t.Run("RequiresAdminRole", func(t *testing.T) {
		rec := env.writeAlbumRaw(t, graceCookie, http.MethodGet, "/api/v1/admin/settings/mount-roots", "")
		if rec.Code != http.StatusForbidden {
			t.Fatalf("GET as a plain member status = %d, want %d (body %q)",
				rec.Code, http.StatusForbidden, rec.Body.String())
		}
		rec = env.writeAlbumRaw(t, graceCookie, http.MethodPut, "/api/v1/admin/settings/mount-roots", `{"roots":[]}`)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("PUT as a plain member status = %d, want %d (body %q)",
				rec.Code, http.StatusForbidden, rec.Body.String())
		}
	})

	t.Run("RequiresSession", func(t *testing.T) {
		rec := env.writeAlbumRaw(t, nil, http.MethodGet, "/api/v1/admin/settings/mount-roots", "")
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("GET unauthenticated status = %d, want %d", rec.Code, http.StatusUnauthorized)
		}
		rec = env.writeAlbumRaw(t, nil, http.MethodPut, "/api/v1/admin/settings/mount-roots", `{"roots":[]}`)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("PUT unauthenticated status = %d, want %d", rec.Code, http.StatusUnauthorized)
		}
	})
}

// TestAlbumPathCompletions pins the completion: the matching rules
// (the roots on the empty prefix and on a leading fragment, the
// parent's matching children for a mid-path fragment, the trailing
// separator's drill-in), the row bound, and -- the one pin of the
// endpoint -- the domain confine: nothing outside the allowlist is
// ever listed, and the default closed posture lists nothing at all.
func TestAlbumPathCompletions(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	_, graceCookie := env.createSecondUser(t, "completions-grace@example.com")

	// A structured domain: outer holds the allowed root and a
	// sibling directory that must never leak into a completion.
	outer := t.TempDir()
	root := filepath.Join(outer, "allowed")
	if err := os.MkdirAll(filepath.Join(root, "family-2025"), 0o755); err != nil {
		t.Fatalf("create family-2025: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, "family-2026"), 0o755); err != nil {
		t.Fatalf("create family-2026: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, "other"), 0o755); err != nil {
		t.Fatalf("create other: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, "family-2026", "spring"), 0o755); err != nil {
		t.Fatalf("create spring: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(outer, "outside", "secret"), 0o755); err != nil {
		t.Fatalf("create the outside sibling: %v", err)
	}
	env.setMountRoots(t, env.cookie, root)

	t.Run("EmptyPrefixListsTheRoots", func(t *testing.T) {
		requirePaths(t, env.completions(t, env.cookie, ""), root)
	})

	t.Run("LeadingFragmentFindsTheRoot", func(t *testing.T) {
		// The typed prefix is a leading string of the root's path.
		requirePaths(t, env.completions(t, env.cookie, outer+"/all"), root)
		// A fragment that matches nothing in the domain answers the
		// empty list -- the honest miss, never an error.
		requirePaths(t, env.completions(t, env.cookie, outer+"/zzz"))
	})

	t.Run("MidPathFragmentMatchesTheLastSegment", func(t *testing.T) {
		requirePaths(t, env.completions(t, env.cookie, root+"/fam"),
			filepath.Join(root, "family-2025"), filepath.Join(root, "family-2026"))
		requirePaths(t, env.completions(t, env.cookie, root+"/family-2026"),
			filepath.Join(root, "family-2026"))
	})

	t.Run("TrailingSeparatorDrillsIn", func(t *testing.T) {
		// The trailing slash says "inside this directory": the
		// children of the root, no root row riding along.
		requirePaths(t, env.completions(t, env.cookie, root+"/"),
			filepath.Join(root, "family-2025"), filepath.Join(root, "family-2026"), filepath.Join(root, "other"))
		requirePaths(t, env.completions(t, env.cookie, root+"/family-2026/"),
			filepath.Join(root, "family-2026", "spring"))
	})

	t.Run("NeverListsOutsideTheAllowlist", func(t *testing.T) {
		// The pinned confine: the children of a directory that
		// itself sits outside the domain are filtered to the
		// in-domain ones -- "outside" never appears, whatever
		// prefix would match it.
		requirePaths(t, env.completions(t, env.cookie, outer+"/"), root)
		requirePaths(t, env.completions(t, env.cookie, outer+"/o"))
		requirePaths(t, env.completions(t, env.cookie, outer+"/outs"))
		// A path in a wholly unconfigured tree answers nothing.
		requirePaths(t, env.completions(t, env.cookie, t.TempDir()+"/sec"))
	})

	t.Run("RelativePrefixAnswersNothing", func(t *testing.T) {
		requirePaths(t, env.completions(t, env.cookie, "fam"))
	})

	t.Run("CapsTheRows", func(t *testing.T) {
		for i := 1; i <= 25; i++ {
			dir := filepath.Join(root, fmt.Sprintf("bulk-%02d", i))
			if err := os.Mkdir(dir, 0o755); err != nil {
				t.Fatalf("create %s: %v", dir, err)
			}
		}
		paths := env.completions(t, env.cookie, root+"/bulk-")
		if len(paths) != 20 {
			t.Fatalf("completions = %d rows, want the cap of 20", len(paths))
		}
		for i, path := range paths {
			want := filepath.Join(root, fmt.Sprintf("bulk-%02d", i+1))
			if path != want {
				t.Fatalf("row %d = %s, want %s (the first 20, sorted)", i, path, want)
			}
		}
	})

	t.Run("InertRootsContributeNothing", func(t *testing.T) {
		// A root that does not exist (removable media), one that is
		// a file, and one inside the server's managed tree all stay
		// configured while contributing no rows.
		file := filepath.Join(outer, "a-file")
		if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
			t.Fatalf("create the file root: %v", err)
		}
		env.setMountRoots(t, env.cookie,
			filepath.Join(outer, "not-mounted"), file, filepath.Join(env.dataDir, "originals"))
		requirePaths(t, env.completions(t, env.cookie, ""))
	})

	t.Run("UserScoped", func(t *testing.T) {
		// Every signed-in member may create a directory album, so
		// every signed-in member may complete a path: a plain
		// member's call answers 200 like the admin's. (The earlier
		// subtests reconfigured the allowlist; re-open the root
		// first so the member sees the same domain.)
		env.setMountRoots(t, env.cookie, root)
		requirePaths(t, env.completions(t, graceCookie, outer+"/all"), root)
	})

	t.Run("RequiresSession", func(t *testing.T) {
		rec := env.completionsRaw(t, nil, "")
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("GET /albums/path-completions unauthenticated status = %d, want %d",
				rec.Code, http.StatusUnauthorized)
		}
	})

	t.Run("ValidatesThePrefixBound", func(t *testing.T) {
		rec := env.completionsRaw(t, env.cookie, "/"+strings.Repeat("a", 4096))
		requireValidationFields(t, rec, "prefix")
	})
}

// TestCreateAlbumPathMatrix pins the creation-time path validation
// matrix: every branch answers its own error code, and the branches
// that pass (the symlink that resolves INTO the domain) store the
// resolved path. The default-closed posture (no configured root)
// answers ALBUM_PATH_OUTSIDE_ROOTS for anything.
func TestCreateAlbumPathMatrix(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	root := env.provisionMountRoot(t)
	source := filepath.Join(root, "family")
	if err := os.Mkdir(source, 0o755); err != nil {
		t.Fatalf("create the source directory: %v", err)
	}
	aFile := filepath.Join(root, "a-file")
	if err := os.WriteFile(aFile, []byte("x"), 0o644); err != nil {
		t.Fatalf("create the file: %v", err)
	}

	create := func(t *testing.T, rootPath string) *httptest.ResponseRecorder {
		t.Helper()
		return env.createAlbumRaw(t, env.cookie, createAlbumBody(t, "Matrix", rootPath, nil))
	}

	t.Run("MissingOrRelativeOrOverlong", func(t *testing.T) {
		for _, rootPath := range []string{"", "Photos/family", strings.Repeat("/", 1) + strings.Repeat("a", 4096)} {
			rec := create(t, rootPath)
			requireValidationFields(t, rec, "rootPath")
		}
	})

	t.Run("NotExists", func(t *testing.T) {
		requireAlbumPathError(t, create(t, filepath.Join(root, "nope")), apigen.ALBUMPATHNOTFOUND)
	})

	t.Run("NotADirectory", func(t *testing.T) {
		requireAlbumPathError(t, create(t, aFile), apigen.ALBUMPATHNOTDIRECTORY)
	})

	t.Run("NotReadable", func(t *testing.T) {
		if os.Geteuid() == 0 {
			t.Skip("the readability branch is untestable as root: the kernel lets root read anything")
		}
		secret := filepath.Join(root, "secret")
		if err := os.Mkdir(secret, 0o755); err != nil {
			t.Fatalf("create the secret directory: %v", err)
		}
		if err := os.Chmod(secret, 0o0000); err != nil {
			t.Fatalf("chmod the secret directory: %v", err)
		}
		t.Cleanup(func() { _ = os.Chmod(secret, 0o755) })
		requireAlbumPathError(t, create(t, secret), apigen.ALBUMPATHNOTREADABLE)
	})

	t.Run("SymlinkEscape", func(t *testing.T) {
		// The typed path sits inside the root, the resolution lands
		// outside: the escape's own branch, distinct from the
		// plainly-outside path.
		escape := filepath.Join(root, "escape")
		if err := os.Symlink("/etc", escape); err != nil {
			t.Fatalf("create the escaping symlink: %v", err)
		}
		requireAlbumPathError(t, create(t, escape), apigen.ALBUMPATHSYMLINKESCAPE)
	})

	t.Run("OutsideTheRoots", func(t *testing.T) {
		// A real directory the server can read, but no configured
		// root covers it.
		requireAlbumPathError(t, create(t, t.TempDir()), apigen.ALBUMPATHOUTSIDEROOTS)
	})

	t.Run("InManagedTree", func(t *testing.T) {
		// The managed tree is refused WHATEVER the allowlist says:
		// this run opens a root that covers the environment's whole
		// data directory (its parent), and the originals tree still
		// answers its own branch.
		env.setMountRoots(t, env.cookie, filepath.Dir(env.dataDir))
		requireAlbumPathError(t, create(t, filepath.Join(env.dataDir, "originals")), apigen.ALBUMPATHINMANAGEDTREE)
		requireAlbumPathError(t, create(t, env.dataDir), apigen.ALBUMPATHINMANAGEDTREE)
		// The data root itself is also refused for the completion
		// domain (an inert root): the completion under the covering
		// root never suggests the managed tree.
		for _, path := range env.completions(t, env.cookie, env.dataDir+"/ori") {
			if pathUnder(path, env.dataDir) {
				t.Errorf("completion suggests the managed tree path %s", path)
			}
		}
		// Back to the tight root for the branches below.
		env.setMountRoots(t, env.cookie, root)
	})

	t.Run("DefaultClosedPosture", func(t *testing.T) {
		// A fresh instance configured nothing: every source path
		// answers the outside-roots branch -- directory album
		// creation is closed until an administrator opens a root.
		fresh := newAssetEnv(t, testUploadLimit)
		rec := fresh.createAlbumRaw(t, fresh.cookie, createAlbumBody(t, "Closed", t.TempDir(), nil))
		requireAlbumPathError(t, rec, apigen.ALBUMPATHOUTSIDEROOTS)
		if got := fresh.queryInt(t, "SELECT count(*) FROM albums WHERE type = 'directory'"); got != 0 {
			t.Errorf("directory album rows = %d, want 0 (nothing landed)", got)
		}
	})

	t.Run("SymlinkIntoTheDomainStoresTheResolvedPath", func(t *testing.T) {
		// A link outside the domain whose target sits inside it
		// passes: the RESOLVED path is the domain subject, and the
		// row stores it -- a later symlink swap cannot repoint the
		// album.
		target := sourceDir(t, root, "inbound")
		link := filepath.Join(t.TempDir(), "into-the-family")
		if err := os.Symlink(target, link); err != nil {
			t.Fatalf("create the inbound symlink: %v", err)
		}
		summary := env.createAlbum(t, env.cookie, "Via the link", link, nil)
		if got := env.queryText(t, "SELECT root_path FROM albums WHERE id = $1",
			pgUUID(uuid.UUID(summary.Id))); got != target {
			t.Errorf("stored root_path = %q, want the resolved %q", got, target)
		}
	})

	t.Run("OverlapAndNesting", func(t *testing.T) {
		_, graceCookie := env.createSecondUser(t, "overlap-grace@example.com")

		// The first album owns the source.
		first := env.createAlbum(t, env.cookie, "The source", source, nil)

		// Equal root: the overlap branch.
		requireAlbumPathError(t, create(t, source), apigen.ALBUMPATHOVERLAPS)
		// Nested inside the existing root (either direction): the
		// same branch.
		nested := filepath.Join(source, "nested")
		if err := os.Mkdir(nested, 0o755); err != nil {
			t.Fatalf("create the nested directory: %v", err)
		}
		requireAlbumPathError(t, create(t, nested), apigen.ALBUMPATHOVERLAPS)
		// The containing direction: a root that would contain the
		// existing one.
		requireAlbumPathError(t, create(t, root), apigen.ALBUMPATHOVERLAPS)

		// The overlap check runs across the WHOLE instance: another
		// account's album over the same tree answers the same
		// branch.
		rec := env.createAlbumRaw(t, graceCookie, createAlbumBody(t, "Grace's overlap", nested, nil))
		requireAlbumPathError(t, rec, apigen.ALBUMPATHOVERLAPS)

		// The component boundary holds: a sibling whose name merely
		// shares the prefix is its own directory, no overlap.
		sibling := filepath.Join(root, "family-2")
		if err := os.Mkdir(sibling, 0o755); err != nil {
			t.Fatalf("create the sibling directory: %v", err)
		}
		summary := env.createAlbum(t, env.cookie, "The sibling", sibling, nil)
		if uuid.UUID(summary.Id) == uuid.UUID(first.Id) {
			t.Fatal("the sibling creation reused the first album's id")
		}
	})
}
