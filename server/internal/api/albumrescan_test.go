// The rescan endpoint tests: POST /albums/{albumId}/rescan's
// own contract -- the two-tier authorization answer (unauthenticated
// 401, sharee 403, invisible 404), the type gate (the rescan is the
// directory album's own flow: virtual and sync albums fold into the
// single-shape 404), the locked gate's 409, the album-scoped
// in-flight idempotence (200 with the joined run's identifier),
// the 202 of a fresh enqueue through the seam, and the 500 of the
// unwired queue. The three arms themselves live in the jobs
// package's tests, where the real worker runs.
package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	apigen "github.com/welpher/lumemor/server/internal/api/gen"
	"github.com/welpher/lumemor/server/internal/storage"
)

// stubAlbumRescans records the rescan enqueues the endpoint made and
// answers the configured result, the seam-fake pattern of the rerun
// and re-resolution endpoints.
type stubAlbumRescans struct {
	albums []uuid.UUID
	err    error
	id     uuid.UUID
}

// EnqueueAlbumRescan implements AlbumRescanEnqueuer.
func (s *stubAlbumRescans) EnqueueAlbumRescan(ctx context.Context, albumID uuid.UUID) (uuid.UUID, error) {
	s.albums = append(s.albums, albumID)
	if s.err != nil {
		return uuid.Nil, s.err
	}
	return s.id, nil
}

// rescanEnv provisions the endpoint's test environment: a migrated
// throwaway database, a fresh storage layout, and the router wired
// to the stub rescan seam.
type rescanEnv struct {
	router  http.Handler
	pool    *pgxpool.Pool
	stub    *stubAlbumRescans
	cookie  *http.Cookie
	dataDir string
	adminID uuid.UUID
}

// newRescanEnv provisions the environment with the administrator
// signed in.
func newRescanEnv(t *testing.T) *rescanEnv {
	t.Helper()
	pool := newSetupEnv(t)
	dataDir := t.TempDir()
	if err := storage.EnsureLayout(dataDir, t.TempDir()); err != nil {
		t.Fatalf("prepare storage layout: %v", err)
	}
	stub := &stubAlbumRescans{id: uuid.MustParse("0192f8b1-4c5d-8e9f-a0b1-2c3d4e5f6a70")}
	router := NewRouter(&Deps{
		Pool:    pool,
		Version: testVersion,
		DataDir: dataDir,
		// The rescan endpoint's seam: the stub records the enqueues;
		// the three arms' truth lives in the jobs package's tests.
		AlbumRescans: stub,
	})
	rec := postAdmin(t, router, createAdminBody(t, "ada@example.com", testPassword, nil))
	if rec.Code != http.StatusCreated {
		t.Fatalf("create admin: status = %d, want %d (body %q)", rec.Code, http.StatusCreated, rec.Body.String())
	}
	cookie, _ := loginCookie(t, router, "ada@example.com", testPassword, "rescan-tests")
	var adminID uuid.UUID
	if err := pool.QueryRow(context.Background(),
		"SELECT id FROM users WHERE email = $1", "ada@example.com").Scan(&adminID); err != nil {
		t.Fatalf("resolve the admin id: %v", err)
	}
	return &rescanEnv{router: router, pool: pool, stub: stub, cookie: cookie, dataDir: dataDir, adminID: adminID}
}

// rescanRaw fires POST /albums/{albumId}/rescan with the given
// cookie (nil sends it unauthenticated).
func (e *rescanEnv) rescanRaw(t *testing.T, cookie *http.Cookie, albumID uuid.UUID) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/albums/"+albumID.String()+"/rescan", nil)
	if cookie != nil {
		req.AddCookie(cookie)
	}
	rec := httptest.NewRecorder()
	e.router.ServeHTTP(rec, req)
	return rec
}

// insertAlbum inserts one fixture album row directly (the endpoint
// tests gate on rows the real creation flow already shaped; the
// mount-root matrix lives in albumroots_test.go).
func (e *rescanEnv) insertAlbum(t *testing.T, id, owner uuid.UUID, albumType, title string, locked bool) {
	t.Helper()
	if _, err := e.pool.Exec(context.Background(), `
		INSERT INTO albums (id, owner_id, title, type, locked)
		VALUES ($1, $2, $3, $4, $5)`,
		pgUUID(id), pgUUID(owner), title, albumType, locked); err != nil {
		t.Fatalf("insert fixture album %s: %v", id, err)
	}
}

// shareAlbum grants one user the view ACL of an album.
func (e *rescanEnv) shareAlbum(t *testing.T, album, grantee, grantedBy uuid.UUID) {
	t.Helper()
	if _, err := e.pool.Exec(context.Background(),
		"INSERT INTO album_shares (album_id, user_id, granted_by, granted_at) VALUES ($1, $2, $3, now())",
		pgUUID(album), pgUUID(grantee), pgUUID(grantedBy)); err != nil {
		t.Fatalf("insert fixture album share %s -> %s: %v", album, grantee, err)
	}
}

// rescan decodes the endpoint's JSON body.
func rescanResponse(t *testing.T, rec *httptest.ResponseRecorder) apigen.AlbumRescanResponse {
	t.Helper()
	var response apigen.AlbumRescanResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode rescan body %q: %v", rec.Body.String(), err)
	}
	return response
}

// TestRescanAlbumGates pins the endpoint's own matrix: the
// unauthenticated 401, the single-shape 404 of the unknown id, the
// honest 403 of the sharee, the 404 of the sync and virtual albums
// (the rescan is the directory album's own flow), and the 409 of
// the locked album.
func TestRescanAlbumGates(t *testing.T) {
	env := newRescanEnv(t)
	adminID := env.adminID
	graceID, graceCookie := seedPlainUser(t, env.pool, "rescan-grace@example.com")

	// The caller's own sync album: not a rescan target.
	var syncAlbum uuid.UUID
	if err := env.pool.QueryRow(context.Background(),
		"SELECT id FROM albums WHERE owner_id = $1 AND type = 'sync'", pgUUID(adminID)).Scan(&syncAlbum); err != nil {
		t.Fatalf("load the sync album: %v", err)
	}
	virtual := fixtureWallAlbumID("aa")
	env.insertAlbum(t, virtual, adminID, "virtual", "Picks", false)
	directory := fixtureWallAlbumID("ab")
	env.insertAlbum(t, directory, adminID, "directory", "Family", false)
	locked := fixtureWallAlbumID("ac")
	env.insertAlbum(t, locked, adminID, "directory", "Locked", true)
	shared := fixtureWallAlbumID("ad")
	env.insertAlbum(t, shared, adminID, "directory", "Shared", false)
	env.shareAlbum(t, shared, graceID, adminID)

	t.Run("Unauthenticated", func(t *testing.T) {
		rec := env.rescanRaw(t, nil, directory)
		requireUnauthorized(t, rec)
	})

	t.Run("UnknownAlbum", func(t *testing.T) {
		rec := env.rescanRaw(t, env.cookie, fixtureWallAlbumID("ae"))
		if rec.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want %d (body %q)", rec.Code, http.StatusNotFound, rec.Body.String())
		}
		apiErr, _ := decodeError(t, rec)
		if apiErr.Code != apigen.ALBUMNOTFOUND {
			t.Errorf("code = %q, want %q", apiErr.Code, apigen.ALBUMNOTFOUND)
		}
	})

	t.Run("SyncAlbum", func(t *testing.T) {
		rec := env.rescanRaw(t, env.cookie, syncAlbum)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want the single-shape 404 (body %q)", rec.Code, rec.Body.String())
		}
		apiErr, _ := decodeError(t, rec)
		if apiErr.Code != apigen.ALBUMNOTFOUND {
			t.Errorf("code = %q, want %q", apiErr.Code, apigen.ALBUMNOTFOUND)
		}
	})

	t.Run("VirtualAlbum", func(t *testing.T) {
		rec := env.rescanRaw(t, env.cookie, virtual)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want the single-shape 404 (body %q)", rec.Code, rec.Body.String())
		}
		apiErr, _ := decodeError(t, rec)
		if apiErr.Code != apigen.ALBUMNOTFOUND {
			t.Errorf("code = %q, want %q", apiErr.Code, apigen.ALBUMNOTFOUND)
		}
	})

	t.Run("Sharee", func(t *testing.T) {
		rec := env.rescanRaw(t, graceCookie, shared)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("status = %d, want %d (body %q)", rec.Code, http.StatusForbidden, rec.Body.String())
		}
		apiErr, _ := decodeError(t, rec)
		if apiErr.Code != apigen.AUTHFORBIDDEN {
			t.Errorf("code = %q, want %q", apiErr.Code, apigen.AUTHFORBIDDEN)
		}
	})

	t.Run("Locked", func(t *testing.T) {
		rec := env.rescanRaw(t, env.cookie, locked)
		if rec.Code != http.StatusConflict {
			t.Fatalf("status = %d, want %d (body %q)", rec.Code, http.StatusConflict, rec.Body.String())
		}
		apiErr, _ := decodeError(t, rec)
		if apiErr.Code != apigen.ALBUMLOCKED {
			t.Errorf("code = %q, want %q", apiErr.Code, apigen.ALBUMLOCKED)
		}
	})
}

// TestRescanAlbumStartsAndJoins pins the endpoint's happy paths and
// its idempotence: a fresh album answers 202 with the enqueued run's
// identifier and the enqueue saw exactly this album, an in-flight
// run of the SAME album answers 200 with its identifier and starts
// nothing (the stub records no second enqueue), and another album's
// in-flight run never blocks this one.
func TestRescanAlbumStartsAndJoins(t *testing.T) {
	env := newRescanEnv(t)
	adminID := env.adminID
	directory := fixtureWallAlbumID("ba")
	other := fixtureWallAlbumID("bb")
	env.insertAlbum(t, directory, adminID, "directory", "Family", false)
	env.insertAlbum(t, other, adminID, "directory", "Other", false)

	t.Run("Starts", func(t *testing.T) {
		rec := env.rescanRaw(t, env.cookie, directory)
		if rec.Code != http.StatusAccepted {
			t.Fatalf("status = %d, want %d (body %q)", rec.Code, http.StatusAccepted, rec.Body.String())
		}
		response := rescanResponse(t, rec)
		if response.JobRunId != env.stub.id {
			t.Errorf("jobRunId = %s, want the seam's identifier %s", response.JobRunId, env.stub.id)
		}
		if len(env.stub.albums) != 1 || env.stub.albums[0] != directory {
			t.Errorf("the seam saw %v, want exactly [%s]", env.stub.albums, directory)
		}
	})

	t.Run("JoinsTheInFlightRun", func(t *testing.T) {
		// A run of this album still in flight (the worker's flip to
		// running is the worker's business; the queued state is what
		// the enqueue left behind).
		runID := uuid.MustParse("0192f8b1-4c5d-8e9f-a0b1-2c3d4e5f6a71")
		if _, err := env.pool.Exec(context.Background(),
			"INSERT INTO job_runs (id, kind, total, album_id) VALUES ($1, 'album_rescan', 0, $2)",
			pgUUID(runID), pgUUID(directory)); err != nil {
			t.Fatalf("seed the in-flight run: %v", err)
		}

		rec := env.rescanRaw(t, env.cookie, directory)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d (body %q)", rec.Code, http.StatusOK, rec.Body.String())
		}
		response := rescanResponse(t, rec)
		if response.JobRunId != runID {
			t.Errorf("jobRunId = %s, want the joined run %s", response.JobRunId, runID)
		}
		// The join started nothing: the stub still holds the single
		// enqueue of the Starts arm.
		if len(env.stub.albums) != 1 {
			t.Errorf("the seam saw %d enqueues after the join, want still 1", len(env.stub.albums))
		}
	})

	t.Run("OtherAlbumInFlightDoesNotBlock", func(t *testing.T) {
		// A run of ANOTHER album in flight: this album's rescan
		// starts its own -- the guard's scope is the album.
		otherRun := uuid.MustParse("0192f8b1-4c5d-8e9f-a0b1-2c3d4e5f6a72")
		if _, err := env.pool.Exec(context.Background(),
			"INSERT INTO job_runs (id, kind, total, state, album_id) VALUES ($1, 'album_rescan', 0, 'running', $2)",
			pgUUID(otherRun), pgUUID(other)); err != nil {
			t.Fatalf("seed the other album's in-flight run: %v", err)
		}
		// This album's own run is finished (not in flight), so a
		// fresh rescan of it enqueues anew.
		if _, err := env.pool.Exec(context.Background(),
			"UPDATE job_runs SET state = 'succeeded', finished_at = now() WHERE album_id = $1",
			pgUUID(directory)); err != nil {
			t.Fatalf("finish this album's run: %v", err)
		}

		rec := env.rescanRaw(t, env.cookie, directory)
		if rec.Code != http.StatusAccepted {
			t.Fatalf("status = %d, want %d (body %q)", rec.Code, http.StatusAccepted, rec.Body.String())
		}
		if len(env.stub.albums) != 2 || env.stub.albums[1] != directory {
			t.Errorf("the seam saw %v, want the second enqueue of this album", env.stub.albums)
		}
	})
}

// TestRescanAlbumWithoutQueue pins the unwired seam's answer: the
// endpoint answers the generic internal error and no enqueue was
// attempted (the gate refused before the seam).
func TestRescanAlbumWithoutQueue(t *testing.T) {
	env := newRescanEnv(t)
	adminID := env.adminID
	directory := fixtureWallAlbumID("bc")
	env.insertAlbum(t, directory, adminID, "directory", "Family", false)

	unwired := NewRouter(&Deps{Pool: env.pool, Version: testVersion, DataDir: env.dataDir})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/albums/"+directory.String()+"/rescan", nil)
	req.AddCookie(env.cookie)
	rec := httptest.NewRecorder()
	unwired.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d (body %q)", rec.Code, http.StatusInternalServerError, rec.Body.String())
	}
	apiErr, _ := decodeError(t, rec)
	if apiErr.Code != apigen.INTERNAL {
		t.Errorf("code = %q, want %q", apiErr.Code, apigen.INTERNAL)
	}
}
