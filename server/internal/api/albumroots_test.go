// The albums-as-sources path domain tests (reshaped by album
// autonomy): the mount-root allowlist's whole retirement
// (the routes answer 404, and a second album over the same tree is
// the operator's choice, not a refusal), the system-level
// source-path completion (the container's real visible filesystem,
// the managed tree never listed, the admin guard over the browse),
// the creation-time path validation matrix (every surviving branch
// its own error code), and the creation-time write probe (the
// source_writable fact -- failure is never a creation failure, and
// the probe file never survives a passing probe).
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

// provisionMountRoot creates one external directory -- a sibling of
// the environment's data root, never inside it -- the shape a
// compose mount exposes to the container. The retirement removed
// the allowlist save this helper once performed: the boundary is
// the mounts themselves, so provisioning is filesystem work alone.
// The caller derives source directories under it.
func (e *assetEnv) provisionMountRoot(t *testing.T) string {
	t.Helper()
	return t.TempDir()
}

// sourceDir creates one subdirectory of an external root as a
// creation source. Tests that create several albums derive one
// sibling each: siblings never share a path, so every creation
// answers 201.
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
func (e *assetEnv) completions(t *testing.T, cookie *http.Cookie, prefix string) []apigen.AlbumPathCompletion {
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
	return response.Completions
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

// requirePaths asserts a completion answer's paths exactly (order
// included); the rows' write facts are the writable arms' business.
func requirePaths(t *testing.T, got []apigen.AlbumPathCompletion, want ...string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("completions = %v, want %v", got, want)
	}
	for i := range want {
		if got[i].Path != want[i] {
			t.Fatalf("completions = %v, want %v", got, want)
		}
	}
}

// requireNoProbeFile fails the test when the source directory still
// holds a probe file: a probe that passed (or failed at its write
// arm) leaves nothing behind.
func requireNoProbeFile(t *testing.T, dir string) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("list the source directory %s: %v", dir, err)
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), "probe-") {
			t.Errorf("the source directory still holds the probe file %s", entry.Name())
		}
	}
}

// TestMountRootsSettingsRetired pins the allowlist's whole removal:
// the contract no longer names the paths, so the router answers its
// plain 404 for the GET and the PUT, whatever role asks -- the
// retirement's thoroughness pin (grep mount-roots to zero is the
// repo's own discipline; this is the behavioral half of it).
func TestMountRootsSettingsRetired(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	_, graceCookie := env.createSecondUser(t, "mount-roots-grace@example.com")

	for _, target := range []string{"/api/v1/admin/settings/mount-roots"} {
		for _, cookie := range []*http.Cookie{env.cookie, graceCookie, nil} {
			for _, method := range []string{http.MethodGet, http.MethodPut} {
				rec := env.writeAlbumRaw(t, cookie, method, target, "")
				if rec.Code != http.StatusNotFound {
					t.Errorf("%s %s (cookie %v) status = %d, want 404 (the allowlist is retired)",
						method, target, cookie != nil, rec.Code)
				}
			}
		}
	}
}

// TestAlbumPathCompletions pins the system-level completion: the
// matching rules over the container's real visible filesystem (the
// root on the empty prefix, the root's children on the bare
// separator, the parent's matching children for a mid-path
// fragment, the trailing separator's drill-in), the row bound, the
// managed-tree exclusion (the one confine that survived the
// retirement), and the admin guard over the browse.
func TestAlbumPathCompletions(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	_, graceCookie := env.createSecondUser(t, "completions-grace@example.com")

	// A structured external tree: root holds the drilled fixtures
	// and a sibling directory.
	root := t.TempDir()
	for _, name := range []string{"family-2025", "family-2026", "other"} {
		if err := os.MkdirAll(filepath.Join(root, name), 0o755); err != nil {
			t.Fatalf("create %s: %v", name, err)
		}
	}
	if err := os.MkdirAll(filepath.Join(root, "family-2026", "spring"), 0o755); err != nil {
		t.Fatalf("create spring: %v", err)
	}

	t.Run("EmptyPrefixListsTheFilesystemRoot", func(t *testing.T) {
		requirePaths(t, env.completions(t, env.cookie, ""), "/")
	})

	t.Run("BareSeparatorListsTheRootsChildren", func(t *testing.T) {
		// The leading separator says "I am at the root": the
		// top-level real directories, no standalone "/" row riding
		// along. /usr is a real directory on every host this suite
		// runs on (the macOS host and the Linux dev container
		// alike).
		paths := env.completions(t, env.cookie, "/")
		if !completionsContain(paths, "/usr") {
			t.Errorf("completions of \"/\" = %v, want /usr among the rows", paths)
		}
		for _, row := range paths {
			if row.Path == "/" {
				t.Errorf("the standalone root row rode along the bare-separator drill")
			}
		}
		// A leading fragment of a top-level name narrows to it.
		requirePaths(t, env.completions(t, env.cookie, "/us"), "/usr")
	})

	t.Run("MidPathFragmentMatchesTheLastSegment", func(t *testing.T) {
		requirePaths(t, env.completions(t, env.cookie, root+"/fam"),
			filepath.Join(root, "family-2025"), filepath.Join(root, "family-2026"))
		requirePaths(t, env.completions(t, env.cookie, root+"/family-2026"),
			filepath.Join(root, "family-2026"))
		// A fragment that matches nothing answers the honest empty
		// list, never an error.
		requirePaths(t, env.completions(t, env.cookie, root+"/zzz"))
	})

	t.Run("TrailingSeparatorDrillsIn", func(t *testing.T) {
		// The trailing slash says "inside this directory": the
		// children, no parent row riding along.
		requirePaths(t, env.completions(t, env.cookie, root+"/"),
			filepath.Join(root, "family-2025"), filepath.Join(root, "family-2026"), filepath.Join(root, "other"))
		requirePaths(t, env.completions(t, env.cookie, root+"/family-2026/"),
			filepath.Join(root, "family-2026", "spring"))
	})

	t.Run("NeverListsTheManagedTree", func(t *testing.T) {
		// The pinned confine, the one survivor of the retirement:
		// the server's own data tree never lists. Drilling into the
		// data root's parent hides the data root itself; a prefix
		// that would match only managed paths answers nothing; and
		// the data root's own children never surface.
		parent := filepath.Dir(env.dataDir)
		for _, row := range env.completions(t, env.cookie, parent+"/") {
			if pathUnder(row.Path, env.dataDir) {
				t.Errorf("completion suggests the managed tree path %s", row.Path)
			}
		}
		requirePaths(t, env.completions(t, env.cookie, env.dataDir+"/origi"))
		requirePaths(t, env.completions(t, env.cookie, env.dataDir+"/"))
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
		for i, row := range paths {
			want := filepath.Join(root, fmt.Sprintf("bulk-%02d", i+1))
			if row.Path != want {
				t.Fatalf("row %d = %s, want %s (the first 20, sorted)", i, row.Path, want)
			}
		}
	})

	t.Run("WritableArms", func(t *testing.T) {
		// The write fact rides every row: a writable directory
		// answers true (and no probe file stays behind), a read-only
		// one answers false as a FACT -- the row lists alongside the
		// others, the refusal is information, never an error.
		writable := filepath.Join(root, "writable")
		if err := os.Mkdir(writable, 0o755); err != nil {
			t.Fatalf("create %s: %v", writable, err)
		}
		rows := env.completions(t, env.cookie, root+"/writable")
		if len(rows) != 1 || rows[0].Path != writable || !rows[0].Writable {
			t.Fatalf("completions = %v, want the writable row with writable true", rows)
		}
		requireNoProbeFile(t, writable)

		if os.Geteuid() == 0 {
			t.Skip("the read-only arm is untestable as root: the kernel lets root write anything")
		}
		readOnly := filepath.Join(root, "read-only")
		if err := os.Mkdir(readOnly, 0o755); err != nil {
			t.Fatalf("create %s: %v", readOnly, err)
		}
		if err := os.Chmod(readOnly, 0o555); err != nil {
			t.Fatalf("chmod the read-only row: %v", err)
		}
		t.Cleanup(func() { _ = os.Chmod(readOnly, 0o755) })
		roRows := env.completions(t, env.cookie, root+"/read")
		if len(roRows) != 1 || roRows[0].Path != readOnly || roRows[0].Writable {
			t.Fatalf("completions = %v, want the read-only row with writable false", roRows)
		}
		requireNoProbeFile(t, readOnly)
	})

	t.Run("UnwritableRowIsAFactNotAnError", func(t *testing.T) {
		// The root-proof arm of the same fact: procfs refuses the
		// probe's write for uid 0 too, so the containerized gate
		// exercises the false arm even where the permission arms
		// skip. The row lists like any other, never an error.
		if _, err := os.Stat("/proc"); err != nil {
			t.Skipf("/proc is unavailable on this host (%v); the unwritable arm needs procfs", err)
		}
		rows := env.completions(t, env.cookie, "/proc")
		found := false
		for _, row := range rows {
			if row.Path == "/proc" {
				found = true
				if row.Writable {
					t.Error("the /proc row's writable = true, want the probe's false")
				}
			}
		}
		if !found {
			t.Error("the /proc row never surfaced (the listing skipped it)")
		}
	})

	t.Run("MemberRefused", func(t *testing.T) {
		// The browse belongs to the configuration act the creation
		// is: a plain member answers the admin surfaces' own 403
		// before the prefix is read.
		rec := env.completionsRaw(t, graceCookie, root+"/fam")
		if rec.Code != http.StatusForbidden {
			t.Fatalf("GET /albums/path-completions as a member status = %d, want %d (body %q)",
				rec.Code, http.StatusForbidden, rec.Body.String())
		}
		apiErr, _ := decodeError(t, rec)
		if apiErr.Code != apigen.AUTHFORBIDDEN {
			t.Errorf("member completion code = %q, want %q", apiErr.Code, apigen.AUTHFORBIDDEN)
		}
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

// completionsContain reports whether the path sits in the rows.
func completionsContain(paths []apigen.AlbumPathCompletion, value string) bool {
	for _, row := range paths {
		if row.Path == value {
			return true
		}
	}
	return false
}

// TestCreateAlbumPathMatrix pins the creation-time path validation
// matrix: every surviving branch answers its own error code, the
// resolved path is what the row stores, the retired branches stay
// retired (a second album over the same tree is the administrator's
// visible choice), and the write probe lands its fact without ever
// failing the creation.
func TestCreateAlbumPathMatrix(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	_, graceCookie := env.createSecondUser(t, "matrix-grace@example.com")
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

	t.Run("InManagedTree", func(t *testing.T) {
		// The managed tree is refused absolutely: the data root
		// itself and the originals tree answer their own branch,
		// with no covering root to lean on (the allowlist is gone).
		requireAlbumPathError(t, create(t, filepath.Join(env.dataDir, "originals")), apigen.ALBUMPATHINMANAGEDTREE)
		requireAlbumPathError(t, create(t, env.dataDir), apigen.ALBUMPATHINMANAGEDTREE)
	})

	t.Run("SymlinkIntoTheManagedTree", func(t *testing.T) {
		// A typed path outside the managed tree that RESOLVES into
		// it answers the managed-tree branch: the row stores the
		// resolution, and the resolution sits in the tree.
		escape := filepath.Join(root, "into-data")
		if err := os.Symlink(env.dataDir, escape); err != nil {
			t.Fatalf("create the managed-tree symlink: %v", err)
		}
		requireAlbumPathError(t, create(t, escape), apigen.ALBUMPATHINMANAGEDTREE)
	})

	t.Run("SymlinkResolutionStoresTheResolvedPath", func(t *testing.T) {
		// The row stores the RESOLVED path: a later symlink swap
		// cannot repoint the album.
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

	t.Run("MemberRefused", func(t *testing.T) {
		// Mounting an existing photo directory is the
		// administrator's act: a plain member answers the admin
		// surfaces' own 403 before the title, the path or the
		// audience is considered.
		rec := env.createAlbumRaw(t, graceCookie, createAlbumBody(t, "Grace's album", source, nil))
		if rec.Code != http.StatusForbidden {
			t.Fatalf("POST /albums as a member status = %d, want %d (body %q)",
				rec.Code, http.StatusForbidden, rec.Body.String())
		}
		apiErr, _ := decodeError(t, rec)
		if apiErr.Code != apigen.AUTHFORBIDDEN {
			t.Errorf("member creation code = %q, want %q", apiErr.Code, apigen.AUTHFORBIDDEN)
		}
		if got := env.queryInt(t, "SELECT count(*) FROM albums WHERE root_path = $1", source); got != 0 {
			t.Errorf("albums over the refused source = %d, want 0 (the refusal landed nothing)", got)
		}
	})

	t.Run("SecondAlbumOverTheSameTree", func(t *testing.T) {
		// The overlap branch retired with the allowlist: with
		// creation in the administrator's hands, a second album over
		// the same tree is a visible operator choice, not a
		// refusal -- two albums, two ids, one source.
		first := env.createAlbum(t, env.cookie, "The source", source, nil)
		second := env.createAlbum(t, env.cookie, "The mirror", source, nil)
		if uuid.UUID(first.Id) == uuid.UUID(second.Id) {
			t.Fatal("the second creation reused the first album's id")
		}
		if got := env.queryInt(t, "SELECT count(*) FROM albums WHERE root_path = $1", source); got != 2 {
			t.Errorf("albums over %s = %d, want 2 (the overlap branch is retired)", source, got)
		}
	})

	t.Run("ProbeReportsAWritableSource", func(t *testing.T) {
		summary := env.createAlbum(t, env.cookie, "Writable", source, nil)
		if !summary.SourceWritable {
			t.Error("sourceWritable = false, want the probe's true over a writable source")
		}
		requireNoProbeFile(t, source)
		if got := env.queryInt(t, "SELECT source_writable::int FROM albums WHERE id = $1",
			pgUUID(uuid.UUID(summary.Id))); got != 1 {
			t.Errorf("stored source_writable = %d, want 1", got)
		}
	})

	t.Run("ProbeFailureIsNotCreationFailure", func(t *testing.T) {
		if os.Geteuid() == 0 {
			t.Skip("the read-only arm is untestable as root: the kernel lets root write anything")
		}
		// A readable but read-only directory: the album is created,
		// the fact is false, and the response says so honestly --
		// the pinned "probe failure != create failure" arm.
		ro := sourceDir(t, root, "read-only")
		if err := os.Chmod(ro, 0o555); err != nil {
			t.Fatalf("chmod the read-only source: %v", err)
		}
		t.Cleanup(func() { _ = os.Chmod(ro, 0o755) })
		summary := env.createAlbum(t, env.cookie, "Read-only source", ro, nil)
		if summary.SourceWritable {
			t.Error("sourceWritable = true, want the probe's false over a read-only source")
		}
		requireNoProbeFile(t, ro)
	})

	t.Run("UnwritableSourceIsStillCreated", func(t *testing.T) {
		// The root-proof arm of the same pin: procfs refuses the
		// file creation for uid 0 too (the write answers ENOENT),
		// so the containerized gate exercises the probe's failure
		// even where the permission arms skip. The album still
		// lands, the fact is false, and no probe file ever existed
		// to leave behind.
		if _, err := os.Stat("/proc"); err != nil {
			t.Skipf("/proc is unavailable on this host (%v); the unwritable arm needs procfs", err)
		}
		summary := env.createAlbum(t, env.cookie, "Unwritable source", "/proc", nil)
		if summary.SourceWritable {
			t.Error("sourceWritable = true, want the probe's false over a source the write cannot land in")
		}
		requireNoProbeFile(t, "/proc")
		if got := env.queryInt(t, "SELECT source_writable::int FROM albums WHERE id = $1",
			pgUUID(uuid.UUID(summary.Id))); got != 0 {
			t.Errorf("stored source_writable = %d, want 0", got)
		}
	})
}
