// Timeline tests: the aggregated timeline's semantics
// (dedup, per-user preference exclusion, the live-asset filters, the
// second user's view, the album scope), the keyset pagination with
// the taken_at tie boundary, and the sync-album wiring that makes
// uploads surface in the timeline at all. Fixtures insert users,
// albums, shares, preferences and assets directly in the database:
// user management is later work, album APIs later still, and the
// timeline must be honest about rows no API can produce yet.
package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	apigen "github.com/welpher/lumemor/server/internal/api/gen"
)

// fixtureAssetID builds a deterministic fixture asset id from a
// two-character hex suffix; the byte order of the ids is part of the
// fixture (the pagination tie pair relies on it).
func fixtureAssetID(suffix string) uuid.UUID {
	return uuid.MustParse("00000000-0000-0000-0000-0000000000" + suffix)
}

// fixtureAlbumID builds a deterministic fixture album id from a
// two-character hex suffix, in a namespace disjoint from the asset
// ids.
func fixtureAlbumID(suffix string) uuid.UUID {
	return uuid.MustParse("10000000-0000-0000-0000-0000000000" + suffix)
}

// bulkAssetID builds the i-th id of the bulk pagination fixtures, in
// a namespace disjoint from every hand-picked fixture id.
func bulkAssetID(i int) uuid.UUID {
	return uuid.MustParse(fmt.Sprintf("00000000-0000-0000-0001-%012x", i))
}

// timelineAssetOpts carries the optional column values of a fixture
// asset; the zero value is a plain live, visible, not-ready photo.
type timelineAssetOpts struct {
	isComponent bool
	isHidden    bool
	deleted     bool
	width       int32
	height      int32
	blurhash    string
	ready       bool
	// video switches the row's media kind: the photo-count fixtures
	// need a member that is real but not a photo.
	video bool
}

// insertTimelineAsset inserts one fixture asset row directly. The row
// needs no file on disk: the timeline reads database rows only, which
// is exactly what makes these fixtures cheap.
func (e *assetEnv) insertTimelineAsset(t *testing.T, id, owner uuid.UUID, taken time.Time, opts timelineAssetOpts) {
	t.Helper()
	var deletedAt, readyAt any
	if opts.deleted {
		deletedAt = time.Now()
	}
	if opts.ready {
		readyAt = time.Now()
	}
	var blurhash any
	if opts.blurhash != "" {
		blurhash = opts.blurhash
	}
	kind, mimeType, name := "image", "image/jpeg", "fixture-"+id.String()+".jpg"
	if opts.video {
		kind, mimeType, name = "video", "video/mp4", "fixture-"+id.String()+".mp4"
	}
	if _, err := e.pool.Exec(context.Background(), `
		INSERT INTO assets (
			id, owner_id, storage_kind, file_path, checksum, original_filename,
			file_size, mime_type, media_kind, width, height, taken_at,
			taken_at_source, is_component, is_hidden, deleted_at, blurhash,
			ready_at, pending_jobs
		) VALUES (
			$1, $2, 'blob', $3, $4, $5, 1024, $6, $7, $8, $9,
			$10, 'import', $11, $12, $13, $14, $15, 0
		)`,
		pgUUID(id), pgUUID(owner), "originals/fixtures/"+id.String(),
		"fixture-checksum-"+id.String(), name, mimeType, kind,
		opts.width, opts.height, taken, opts.isComponent, opts.isHidden,
		deletedAt, blurhash, readyAt); err != nil {
		t.Fatalf("insert fixture asset %s: %v", id, err)
	}
}

// insertTwinAsset inserts one fixture source row directly: an
// external asset at a real path, carrying the given content checksum.
// The row needs no file on disk (the same cheapness the blob fixture
// has), and its position key is the path.
func (e *assetEnv) insertTwinAsset(t *testing.T, id, owner uuid.UUID, path, checksum, name string, taken time.Time) {
	t.Helper()
	if _, err := e.pool.Exec(context.Background(), `
		INSERT INTO assets (
			id, owner_id, storage_kind, file_path, checksum, original_filename,
			file_size, mime_type, media_kind, width, height, taken_at,
			taken_at_source, pending_jobs
		) VALUES (
			$1, $2, 'external', $3, $4, $5, 1024, 'image/jpeg', 'image',
			0, 0, $6, 'file', 0
		)`,
		pgUUID(id), pgUUID(owner), path, checksum, name, taken); err != nil {
		t.Fatalf("insert fixture external asset %s: %v", id, err)
	}
}

// insertTimelineAlbum inserts one fixture album of the given type.
func (e *assetEnv) insertTimelineAlbum(t *testing.T, id, owner uuid.UUID, albumType string) {
	t.Helper()
	if _, err := e.pool.Exec(context.Background(),
		"INSERT INTO albums (id, owner_id, title, type) VALUES ($1, $2, $3, $4)",
		pgUUID(id), pgUUID(owner), "Fixture album "+id.String(), albumType); err != nil {
		t.Fatalf("insert fixture album %s: %v", id, err)
	}
}

// addTimelineItem materializes one album membership row.
func (e *assetEnv) addTimelineItem(t *testing.T, album, asset, addedBy uuid.UUID) {
	t.Helper()
	if _, err := e.pool.Exec(context.Background(),
		"INSERT INTO album_items (album_id, asset_id, added_by) VALUES ($1, $2, $3)",
		pgUUID(album), pgUUID(asset), pgUUID(addedBy)); err != nil {
		t.Fatalf("insert fixture album item %s -> %s: %v", album, asset, err)
	}
}

// insertOwnedAsset inserts one fixture asset row AND lands it in the
// owner's sync album -- the production invariant the upload path
// guarantees (every owned asset is a sync member, ensured in the
// upload transaction), which the caller-visible predicate builds on:
// the sync arm is how an owner sees their own rows whatever their
// state. Seeded accounts (no wizard) get their sync album ensured
// here, exactly like the upload path does.
func (e *assetEnv) insertOwnedAsset(t *testing.T, id, owner uuid.UUID, taken time.Time, opts timelineAssetOpts) {
	t.Helper()
	e.insertTimelineAsset(t, id, owner, taken, opts)
	e.exec(t, `
		INSERT INTO albums (id, owner_id, title, type)
		VALUES ($1, $2, 'Sync', 'sync')
		ON CONFLICT (owner_id) WHERE type = 'sync' DO NOTHING`,
		pgUUID(uuid.New()), pgUUID(owner))
	e.addTimelineItem(t, e.syncAlbumIDOf(t, owner), id, owner)
}

// shareTimelineAlbum grants one user the view ACL of an album.
func (e *assetEnv) shareTimelineAlbum(t *testing.T, album, grantee, grantedBy uuid.UUID) {
	t.Helper()
	if _, err := e.pool.Exec(context.Background(),
		"INSERT INTO album_shares (album_id, user_id, granted_by) VALUES ($1, $2, $3)",
		pgUUID(album), pgUUID(grantee), pgUUID(grantedBy)); err != nil {
		t.Fatalf("insert fixture album share %s -> %s: %v", album, grantee, err)
	}
}

// setTimelinePref records the per-user timeline preference of an
// album: false keeps the album's items out of the user's aggregated
// timeline.
func (e *assetEnv) setTimelinePref(t *testing.T, user, album uuid.UUID, inTimeline bool) {
	t.Helper()
	if _, err := e.pool.Exec(context.Background(),
		"INSERT INTO user_album_prefs (user_id, album_id, in_timeline) VALUES ($1, $2, $3)",
		pgUUID(user), pgUUID(album), inTimeline); err != nil {
		t.Fatalf("insert fixture album pref %s -> %s: %v", user, album, err)
	}
}

// syncAlbumIDOf resolves the user's sync album id; the wizard's admin
// owns one by construction, fixture users get one inserted.
func (e *assetEnv) syncAlbumIDOf(t *testing.T, user uuid.UUID) uuid.UUID {
	t.Helper()
	var id pgtype.UUID
	if err := e.pool.QueryRow(context.Background(),
		"SELECT id FROM albums WHERE owner_id = $1 AND type = 'sync'", pgUUID(user)).Scan(&id); err != nil {
		t.Fatalf("resolve sync album of %s: %v", user, err)
	}
	return uuid.UUID(id.Bytes)
}

// timelineItemIDs collects the ids of a page's items in order.
func timelineItemIDs(items []apigen.TimelineItem) []uuid.UUID {
	ids := make([]uuid.UUID, len(items))
	for i, item := range items {
		ids[i] = uuid.UUID(item.Id)
	}
	return ids
}

// requireTimelineIDs asserts the page carries exactly the given ids,
// in order.
func requireTimelineIDs(t *testing.T, items []apigen.TimelineItem, want ...uuid.UUID) {
	t.Helper()
	got := timelineItemIDs(items)
	if len(got) != len(want) {
		t.Fatalf("timeline page holds %d items, want %d (%v)", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("timeline item %d = %s, want %s (page %v)", i, got[i], want[i], got)
		}
	}
}

// getTimeline fires a timeline request with the given cookie (nil
// sends it unauthenticated) and the raw query string, and decodes the
// 200 response body.
func (e *assetEnv) getTimeline(t *testing.T, cookie *http.Cookie, rawQuery string) apigen.TimelineResponse {
	t.Helper()
	rec := e.getTimelineRaw(t, cookie, rawQuery)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /assets/timeline?%s status = %d, want %d (body %q)",
			rawQuery, rec.Code, http.StatusOK, rec.Body.String())
	}
	var response apigen.TimelineResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode timeline body %q: %v", rec.Body.String(), err)
	}
	return response
}

// getTimelineRaw fires the timeline request and returns the recorder.
func (e *assetEnv) getTimelineRaw(t *testing.T, cookie *http.Cookie, rawQuery string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/assets/timeline", nil)
	if rawQuery != "" {
		req.URL.RawQuery = rawQuery
	}
	if cookie != nil {
		req.AddCookie(cookie)
	}
	rec := httptest.NewRecorder()
	e.router.ServeHTTP(rec, req)
	return rec
}

// timelineUploadAs fires one upload with an explicit session cookie
// (the env's own cookie helper serves the admin; this variant serves
// fixture users).
func (e *assetEnv) timelineUploadAs(t *testing.T, cookie *http.Cookie, deviceAssetID string, taken time.Time) apigen.AssetResponse {
	t.Helper()
	body, contentType := buildUploadBody(t, map[string]string{
		"deviceAssetId":   deviceAssetID,
		"fileName":        deviceAssetID + ".jpg",
		"deviceCreatedAt": taken.Format(time.RFC3339),
	}, []assetUploadFile{{field: uploadPartPhoto, name: deviceAssetID + ".jpg", content: jpegFixture(t)}})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/assets", body)
	req.Header.Set("Content-Type", contentType)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	e.router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("upload as fixture user status = %d, want %d (body %q)",
			rec.Code, http.StatusCreated, rec.Body.String())
	}
	return decodeAsset(t, rec)
}

// TestTimelineUploadWiring covers the load-bearing prerequisite of
// the timeline: uploads actually land in the owner's sync album. The
// wizard's admin creation seeds the sync album with the account, the
// upload transaction inserts the membership, and an account whose
// rows predate the wiring gets its album ensured on the first upload.
func TestTimelineUploadWiring(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	adminID := env.sessionUserID(t)

	t.Run("AdminCreationSeedsSyncAlbum", func(t *testing.T) {
		count := env.queryInt(t,
			"SELECT count(*) FROM albums WHERE owner_id = $1 AND type = 'sync'", pgUUID(adminID))
		if count != 1 {
			t.Fatalf("admin owns %d sync albums, want 1", count)
		}
	})

	t.Run("UploadLandsInTimeline", func(t *testing.T) {
		wantTaken := time.Date(2026, 10, 13, 12, 30, 0, 0, time.UTC)
		asset := env.timelineUploadAs(t, env.cookie, "timeline-wiring-1", wantTaken)

		syncAlbum := env.syncAlbumIDOf(t, adminID)
		members := env.queryInt(t, "SELECT count(*) FROM album_items WHERE album_id = $1 AND asset_id = $2",
			pgUUID(syncAlbum), pgUUID(uuid.UUID(asset.Id)))
		if members != 1 {
			t.Fatalf("uploaded asset has %d sync album memberships, want 1", members)
		}

		page := env.getTimeline(t, env.cookie, "")
		requireTimelineIDs(t, page.Items, uuid.UUID(asset.Id))
		item := page.Items[0]
		if item.IsTrashed {
			t.Error("isTrashed = true, want false for a live timeline item")
		}
		if item.IsReady {
			t.Error("isReady = true, want false before any pipeline job ran")
		}
		if item.Kind != apigen.TimelineItemKindImage {
			t.Errorf("kind = %q, want image", item.Kind)
		}
		if item.TakenAtSource != apigen.TimelineItemTakenAtSourceImport {
			t.Errorf("takenAtSource = %q, want import", item.TakenAtSource)
		}
		if !item.TakenAt.Truncate(time.Microsecond).Equal(wantTaken) {
			t.Errorf("takenAt = %v, want the device-reported %v", item.TakenAt, wantTaken)
		}
		// Width and height are the raw assets columns (the justified
		// wall's contract): always present, the 0 placeholder until
		// the metadata job refines them -- never null.
		if item.Width != 0 || item.Height != 0 {
			t.Errorf("width/height = %d/%d, want 0/0 (the upload placeholder) before the jobs ran",
				item.Width, item.Height)
		}
		if item.Blurhash != nil || item.DurationMs != nil || item.LivePhotoVideoId != nil {
			t.Errorf("pipeline fields = %v/%v/%v, want all null before the jobs ran",
				item.Blurhash, item.DurationMs, item.LivePhotoVideoId)
		}
		if item.Checksum != asset.Checksum || item.MimeType != asset.MimeType ||
			item.OriginalFilename != asset.OriginalFilename || item.FileSize != asset.FileSize {
			t.Errorf("identity fields = %q/%q/%q/%d, want the upload's %q/%q/%q/%d",
				item.Checksum, item.MimeType, item.OriginalFilename, item.FileSize,
				asset.Checksum, asset.MimeType, asset.OriginalFilename, asset.FileSize)
		}
	})

	t.Run("UploadEnsuresSyncAlbumOfDirectlySeededUser", func(t *testing.T) {
		graceID, graceCookie := env.createSecondUser(t, "wiring-grace@example.com")
		// The directly seeded account owns no sync album: the first
		// upload must ensure it, not fail over it.
		if before := env.queryInt(t,
			"SELECT count(*) FROM albums WHERE owner_id = $1 AND type = 'sync'", pgUUID(graceID)); before != 0 {
			t.Fatalf("fixture user owns %d sync albums before the upload, want 0", before)
		}

		taken := time.Date(2026, 10, 14, 9, 0, 0, 0, time.UTC)
		asset := env.timelineUploadAs(t, graceCookie, "timeline-wiring-2", taken)

		if after := env.queryInt(t,
			"SELECT count(*) FROM albums WHERE owner_id = $1 AND type = 'sync'", pgUUID(graceID)); after != 1 {
			t.Fatalf("fixture user owns %d sync albums after the upload, want 1", after)
		}
		syncAlbum := env.syncAlbumIDOf(t, graceID)
		members := env.queryInt(t, "SELECT count(*) FROM album_items WHERE album_id = $1 AND asset_id = $2",
			pgUUID(syncAlbum), pgUUID(uuid.UUID(asset.Id)))
		if members != 1 {
			t.Fatalf("uploaded asset has %d sync album memberships, want 1", members)
		}

		page := env.getTimeline(t, graceCookie, "")
		requireTimelineIDs(t, page.Items, uuid.UUID(asset.Id))
	})
}

// TestTimeline exercises the aggregation formula's semantics: the
// dedup across albums, the per-user preference exclusion (and the
// sync album's exemption from it), the live-asset filters, the
// visibility boundary against another user's content, and the album
// scope of the albumId query parameter.
func TestTimeline(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	adminID := env.sessionUserID(t)
	graceID, graceCookie := env.createSecondUser(t, "timeline-grace@example.com")

	// Fixture layout. The admin's sync album comes from the wizard;
	// Grace's is a fixture row (her account was seeded directly).
	adminSync := env.syncAlbumIDOf(t, adminID)
	graceSync := fixtureAlbumID("01")
	env.insertTimelineAlbum(t, graceSync, graceID, "sync")

	// Albums shared to the admin by Grace (memory type: the shared
	// arm accepts every type). toAdminOff carries the admin's
	// "keep out of the timeline" preference.
	toAdmin := fixtureAlbumID("02")
	env.insertTimelineAlbum(t, toAdmin, graceID, "memory")
	env.shareTimelineAlbum(t, toAdmin, adminID, graceID)
	toAdminOff := fixtureAlbumID("03")
	env.insertTimelineAlbum(t, toAdminOff, graceID, "memory")
	env.shareTimelineAlbum(t, toAdminOff, adminID, graceID)
	env.setTimelinePref(t, adminID, toAdminOff, false)

	// The admin's own album, shared TO Grace: it feeds Grace's
	// timeline, never the admin's own.
	adminShared := fixtureAlbumID("04")
	env.insertTimelineAlbum(t, adminShared, adminID, "memory")
	env.shareTimelineAlbum(t, adminShared, graceID, adminID)

	// The admin's own memory album, unshared: feeds no one's
	// timeline, but its own in-album view works.
	ownMemory := fixtureAlbumID("05")
	env.insertTimelineAlbum(t, ownMemory, adminID, "memory")

	// Admin assets in the sync album.
	a1 := fixtureAssetID("01") // dedup fixture: sync + shared album
	h1 := fixtureAssetID("02") // hidden
	d1 := fixtureAssetID("03") // trashed
	c1 := fixtureAssetID("04") // Live Photo component
	s1 := fixtureAssetID("05") // the share Grace receives
	e1 := fixtureAssetID("06") // own memory only
	e2 := fixtureAssetID("07") // own memory only
	x1 := fixtureAssetID("15") // sync-ONLY live asset
	env.insertTimelineAsset(t, a1, adminID, time.Date(2026, 1, 10, 12, 0, 0, 0, time.UTC), timelineAssetOpts{})
	env.insertTimelineAsset(t, h1, adminID, time.Date(2026, 1, 11, 12, 0, 0, 0, time.UTC), timelineAssetOpts{isHidden: true})
	env.insertTimelineAsset(t, d1, adminID, time.Date(2026, 1, 12, 12, 0, 0, 0, time.UTC), timelineAssetOpts{deleted: true})
	env.insertTimelineAsset(t, c1, adminID, time.Date(2026, 1, 13, 12, 0, 0, 0, time.UTC), timelineAssetOpts{isComponent: true})
	env.insertTimelineAsset(t, s1, adminID, time.Date(2026, 2, 20, 12, 0, 0, 0, time.UTC), timelineAssetOpts{})
	env.insertTimelineAsset(t, e1, adminID, time.Date(2026, 3, 5, 12, 0, 0, 0, time.UTC), timelineAssetOpts{})
	env.insertTimelineAsset(t, e2, adminID, time.Date(2026, 3, 6, 12, 0, 0, 0, time.UTC), timelineAssetOpts{width: 4032, height: 3024, blurhash: "UHL6aAof00WBWB00ayWB00", ready: true})
	env.insertTimelineAsset(t, x1, adminID, time.Date(2026, 1, 8, 12, 0, 0, 0, time.UTC), timelineAssetOpts{})
	for _, id := range []uuid.UUID{a1, h1, d1, c1, x1} {
		env.addTimelineItem(t, adminSync, id, adminID)
	}

	// Grace's assets: b2 and v1 reach the admin through the shared
	// memory album, b1 sits in the pref-switched-off album, p1 stays
	// private in Grace's sync album. The three the memories curate
	// ride her sync stream as well -- the production invariant of the
	// upload path (an owner sees their own rows through the sync
	// arm), which the memory's re-share ground reads.
	b1 := fixtureAssetID("11")
	b2 := fixtureAssetID("12")
	v1 := fixtureAssetID("13")
	p1 := fixtureAssetID("14")
	env.insertOwnedAsset(t, b1, graceID, time.Date(2026, 2, 10, 12, 0, 0, 0, time.UTC), timelineAssetOpts{})
	env.insertOwnedAsset(t, b2, graceID, time.Date(2026, 2, 15, 12, 0, 0, 0, time.UTC), timelineAssetOpts{})
	env.insertOwnedAsset(t, v1, graceID, time.Date(2026, 3, 20, 12, 0, 0, 0, time.UTC), timelineAssetOpts{})
	env.insertTimelineAsset(t, p1, graceID, time.Date(2026, 1, 5, 12, 0, 0, 0, time.UTC), timelineAssetOpts{})
	env.addTimelineItem(t, toAdmin, a1, graceID)
	env.addTimelineItem(t, toAdmin, b2, graceID)
	env.addTimelineItem(t, toAdmin, v1, graceID)
	env.addTimelineItem(t, toAdminOff, b1, graceID)
	env.addTimelineItem(t, adminShared, s1, adminID)
	env.addTimelineItem(t, ownMemory, e1, adminID)
	env.addTimelineItem(t, ownMemory, e2, adminID)
	env.addTimelineItem(t, graceSync, p1, graceID)

	t.Run("AggregateDedupAndExclusions", func(t *testing.T) {
		// The admin's timeline: v1 (shared, March), b2 (shared,
		// February), a1 (own sync + shared, once), x1 (own sync only,
		// January). Everything else is filtered: the hidden, trashed
		// and component rows; the pref-switched-off album's b1; the
		// own memory album's e1/e2; the album shared TO someone else
		// (s1); Grace's private p1.
		page := env.getTimeline(t, env.cookie, "")
		requireTimelineIDs(t, page.Items, v1, b2, a1, x1)
		if page.NextCursor != nil {
			t.Error("nextCursor set, want absent: the whole timeline fits one page")
		}
	})

	t.Run("SyncAlbumPreferenceRowDoesNotExclude", func(t *testing.T) {
		// A preference row on the own sync album excludes nothing: the
		// core stream is not excludable. x1 is the discriminating
		// asset: it lives ONLY in the sync album, so an implementation
		// that (wrongly) let the preference filter the sync arm would
		// drop it -- a1 would survive through its second path (the
		// shared album toAdmin), which is exactly why the old fixture
		// could not tell the two implementations apart.
		env.setTimelinePref(t, adminID, adminSync, false)
		defer env.exec(t, "DELETE FROM user_album_prefs WHERE user_id = $1 AND album_id = $2",
			pgUUID(adminID), pgUUID(adminSync))

		page := env.getTimeline(t, env.cookie, "")
		requireTimelineIDs(t, page.Items, v1, b2, a1, x1)
	})

	t.Run("SecondUserTimeline", func(t *testing.T) {
		// Grace sees her own sync stream (p1 and the three her
		// memories curate -- once each, through their own arms: the
		// memory albums themselves are no arm of the union and inject
		// nothing) and the album the admin shared to her (s1); the
		// admin's private content stays out.
		page := env.getTimeline(t, graceCookie, "")
		requireTimelineIDs(t, page.Items, v1, s1, b2, b1, p1)
	})

	t.Run("AlbumScopeOwn", func(t *testing.T) {
		page := env.getTimeline(t, env.cookie, "albumId="+ownMemory.String())
		requireTimelineIDs(t, page.Items, e2, e1)
	})

	t.Run("AlbumScopeSharedToCaller", func(t *testing.T) {
		// The sharee's walk carries the memory's re-share ground: the
		// members the memory's owner currently sees (her own v1 and
		// b2, through her sync arm) stay, while a1 -- a member whose
		// owner-sight died (the revocation shape) -- leaves the walk
		// live, no snapshot residue.
		page := env.getTimeline(t, env.cookie, "albumId="+toAdmin.String())
		requireTimelineIDs(t, page.Items, v1, b2)
	})

	t.Run("AlbumScopeIgnoresTimelinePreference", func(t *testing.T) {
		// The preference scopes the aggregate, not the album's own
		// view: browsing the switched-off album still shows its items.
		page := env.getTimeline(t, env.cookie, "albumId="+toAdminOff.String())
		requireTimelineIDs(t, page.Items, b1)
	})

	t.Run("AlbumScopeForeignAlbumIsEmpty", func(t *testing.T) {
		// Grace's sync album is not visible to the admin: the empty
		// list, indistinguishable from an unknown id -- no existence
		// leak through a 404.
		page := env.getTimeline(t, env.cookie, "albumId="+graceSync.String())
		requireTimelineIDs(t, page.Items)
		if page.NextCursor != nil {
			t.Error("nextCursor set on the empty album scope, want absent")
		}

		unknown := uuid.MustParse("21000000-0000-0000-0000-000000000001")
		page = env.getTimeline(t, env.cookie, "albumId="+unknown.String())
		requireTimelineIDs(t, page.Items)
	})

	t.Run("ReadyAssetCarriesPipelineFields", func(t *testing.T) {
		page := env.getTimeline(t, env.cookie, "albumId="+ownMemory.String())
		requireTimelineIDs(t, page.Items, e2, e1)
		item := page.Items[0]
		if !item.IsReady {
			t.Error("isReady = false, want true for the fixture with ready_at set")
		}
		// The refined dimensions cross as plain integers (the
		// justified wall's contract pins them NOT NULL).
		if item.Width != 4032 {
			t.Errorf("width = %d, want 4032", item.Width)
		}
		if item.Height != 3024 {
			t.Errorf("height = %d, want 3024", item.Height)
		}
		if item.Blurhash == nil || *item.Blurhash != "UHL6aAof00WBWB00ayWB00" {
			t.Errorf("blurhash = %v, want the fixture value", item.Blurhash)
		}
	})

	t.Run("RequiresSession", func(t *testing.T) {
		rec := env.getTimelineRaw(t, nil, "")
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
		}
	})

	t.Run("InvalidCursor", func(t *testing.T) {
		rec := env.getTimelineRaw(t, env.cookie, "cursor=not-a-cursor")
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d (body %q)", rec.Code, http.StatusBadRequest, rec.Body.String())
		}
		var apiErr apigen.Error
		if err := json.Unmarshal(rec.Body.Bytes(), &apiErr); err != nil {
			t.Fatalf("decode error body %q: %v", rec.Body.String(), err)
		}
		if apiErr.Code != apigen.VALIDATIONFAILED {
			t.Errorf("code = %q, want %q", apiErr.Code, apigen.VALIDATIONFAILED)
		}
	})

	t.Run("InvalidLimit", func(t *testing.T) {
		for _, raw := range []string{"limit=0", "limit=-5"} {
			rec := env.getTimelineRaw(t, env.cookie, raw)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("GET %s status = %d, want %d (body %q)", raw, rec.Code, http.StatusBadRequest, rec.Body.String())
			}
			var apiErr apigen.Error
			if err := json.Unmarshal(rec.Body.Bytes(), &apiErr); err != nil {
				t.Fatalf("decode error body %q: %v", rec.Body.String(), err)
			}
			if apiErr.Code != apigen.VALIDATIONFAILED {
				t.Errorf("GET %s code = %q, want %q", raw, apiErr.Code, apigen.VALIDATIONFAILED)
			}
		}
	})
}

// TestTimelinePagination walks the keyset pagination of the aggregate:
// page sizes, cursor continuity, the disjoint union of the pages, and
// the exact-taken_at tie pair spanning a page boundary (the id
// tiebreak keeps both rows, each exactly once). The limit clamp to
// the contract maximum rides on a bulk fixture.
// TestTimelineLibraryRepresentative pins the position model's display
// dedup on the aggregated timeline: one content held as an uploaded
// copy and as a directory album's position row is ONE tile -- the blob
// while it is live, the earliest position row when the blob is dead --
// while the directory album's own wall lists both members (a wall
// mirrors its source).
func TestTimelineLibraryRepresentative(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	owner := env.sessionUserID(t)

	taken := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	blobID := fixtureAssetID("b1")
	twinID := fixtureAssetID("b2")
	env.insertOwnedAsset(t, blobID, owner, taken, timelineAssetOpts{})
	// The blob row's content key is the fixture's own; the twin
	// carries the SAME checksum at a source path (the dual-home state
	// the fork makes representable).
	env.exec(t, "UPDATE assets SET checksum = 'shared-content' WHERE id = $1", pgUUID(blobID))
	env.insertTwinAsset(t, twinID, owner, "/mnt/source/twin.jpg", "shared-content", "twin.jpg", taken.Add(time.Hour))
	dirAlbum := fixtureAlbumID("b2")
	env.insertTimelineAlbum(t, dirAlbum, owner, "directory")
	env.addTimelineItem(t, dirAlbum, twinID, owner)

	// The aggregate: one tile, the blob (the upload wins a live pair).
	page := env.getTimeline(t, env.cookie, "")
	requireTimelineIDs(t, page.Items, blobID)

	// The directory album's wall is the source's mirror: both rows
	// would be its members, and here only the twin is one -- the wall
	// lists it even though the timeline dedups it away.
	wall := env.getTimeline(t, env.cookie, "albumId="+dirAlbum.String())
	requireTimelineIDs(t, wall.Items, twinID)

	// The blob leaves the library: the earliest live position row
	// takes the group's representative slot with no maintenance step.
	env.exec(t, "UPDATE assets SET deleted_at = now() WHERE id = $1", pgUUID(blobID))
	page = env.getTimeline(t, env.cookie, "")
	requireTimelineIDs(t, page.Items, twinID)

	// The trash view is not a library face: it lists the row raw.
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/trash", nil)
	req.AddCookie(env.cookie)
	env.router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /trash status = %d, want %d", rec.Code, http.StatusOK)
	}
	var trash apigen.TrashResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &trash); err != nil {
		t.Fatalf("decode trash body: %v", err)
	}
	if len(trash.Items) != 1 || trash.Items[0].Id != blobID {
		t.Errorf("trash items = %v, want the trashed blob row alone", trash.Items)
	}

	// The last copy dies and the group leaves the timeline.
	env.exec(t, "UPDATE assets SET deleted_at = now() WHERE id = $1", pgUUID(twinID))
	page = env.getTimeline(t, env.cookie, "")
	if len(page.Items) != 0 {
		t.Errorf("timeline items = %d, want none (every copy dead)", len(page.Items))
	}
}

// TestTimelineEarliestPositionRow pins the tie among position rows:
// two source copies of one content rank by ingest order, so the
// earliest row alone is the tile and a re-scan never reshuffles it.
func TestTimelineEarliestPositionRow(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	owner := env.sessionUserID(t)

	taken := time.Date(2026, 5, 2, 12, 0, 0, 0, time.UTC)
	firstID := fixtureAssetID("c1")
	secondID := fixtureAssetID("c2")
	env.insertTwinAsset(t, firstID, owner, "/mnt/a/copy.jpg", "twin-content", "copy.jpg", taken)
	env.insertTwinAsset(t, secondID, owner, "/mnt/b/copy.jpg", "twin-content", "copy.jpg", taken)
	// The ingest order is the row's created_at (the fixture inserts
	// stamp now(); the second row is forced later to be explicit).
	env.exec(t, "UPDATE assets SET created_at = created_at - interval '1 hour' WHERE id = $1", pgUUID(firstID))
	albumA, albumB := fixtureAlbumID("c1"), fixtureAlbumID("c2")
	env.insertTimelineAlbum(t, albumA, owner, "directory")
	env.insertTimelineAlbum(t, albumB, owner, "directory")
	env.addTimelineItem(t, albumA, firstID, owner)
	env.addTimelineItem(t, albumB, secondID, owner)

	page := env.getTimeline(t, env.cookie, "")
	requireTimelineIDs(t, page.Items, firstID)

	// Each album's wall keeps its own copy: the source is the truth.
	requireTimelineIDs(t, env.getTimeline(t, env.cookie, "albumId="+albumA.String()).Items, firstID)
	requireTimelineIDs(t, env.getTimeline(t, env.cookie, "albumId="+albumB.String()).Items, secondID)
}

func TestTimelinePagination(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	adminID := env.sessionUserID(t)
	adminSync := env.syncAlbumIDOf(t, adminID)

	// Five assets, taken_at descending except the bottom tie pair:
	// the two lowest share one taken_at, and "25"'s id sorts above
	// "24"'s, so the expected full order is 21, 22, 23, 25, 24 with
	// the tie spanning the boundary between the second and third
	// page of limit-2 pages.
	base := time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC)
	tie := base.Add(time.Minute)
	takenOf := map[uuid.UUID]time.Time{
		fixtureAssetID("21"): base.Add(5 * time.Minute),
		fixtureAssetID("22"): base.Add(4 * time.Minute),
		fixtureAssetID("23"): base.Add(3 * time.Minute),
		fixtureAssetID("24"): tie,
		fixtureAssetID("25"): tie,
	}
	for id, taken := range takenOf {
		env.insertTimelineAsset(t, id, adminID, taken, timelineAssetOpts{})
		env.addTimelineItem(t, adminSync, id, adminID)
	}
	wantOrder := []uuid.UUID{
		fixtureAssetID("21"), fixtureAssetID("22"),
		fixtureAssetID("23"), fixtureAssetID("25"),
		fixtureAssetID("24"),
	}
	tieHigh, tieLow := fixtureAssetID("25"), fixtureAssetID("24")

	seen := make(map[uuid.UUID]bool)
	var walked []uuid.UUID
	cursor := ""
	pages := 0
	for {
		rawQuery := "limit=2"
		if cursor != "" {
			rawQuery += "&cursor=" + cursor
		}
		page := env.getTimeline(t, env.cookie, rawQuery)
		for _, id := range timelineItemIDs(page.Items) {
			if seen[id] {
				t.Fatalf("asset %s appeared on two pages (cursor %q)", id, cursor)
			}
			seen[id] = true
			walked = append(walked, id)
		}
		pages++
		if page.NextCursor == nil {
			break
		}
		cursor = *page.NextCursor
	}

	if pages != 3 {
		t.Fatalf("walked %d pages, want 3 (2+2+1)", pages)
	}
	if len(seen) != len(wantOrder) {
		t.Fatalf("walked %d distinct assets, want %d", len(seen), len(wantOrder))
	}
	for i, want := range wantOrder {
		if walked[i] != want {
			t.Fatalf("walked order[%d] = %s, want %s (walked %v)", i, walked[i], want, walked)
		}
	}
	// The tie pair both appear exactly once, high id first.
	if !seen[tieHigh] || !seen[tieLow] {
		t.Fatalf("the tie pair %s/%s must both appear (seen %v)", tieHigh, tieLow, seen)
	}
	for i, id := range walked {
		if id == tieHigh && i+1 < len(walked) && walked[i+1] != tieLow {
			t.Errorf("the tie pair must stay adjacent, got %s after %s", walked[i+1], tieHigh)
		}
	}

	t.Run("TiePairSharesTakenAt", func(t *testing.T) {
		page := env.getTimeline(t, env.cookie, "limit=5")
		requireTimelineIDs(t, page.Items, wantOrder...)
		high, low := page.Items[3], page.Items[4]
		if !high.TakenAt.Truncate(time.Microsecond).Equal(tie) {
			t.Errorf("tie high takenAt = %v, want %v", high.TakenAt, tie)
		}
		if !low.TakenAt.Truncate(time.Microsecond).Equal(tie) {
			t.Errorf("tie low takenAt = %v, want %v", low.TakenAt, tie)
		}
	})

	t.Run("CursorOfLastRowYieldsEmptyPage", func(t *testing.T) {
		// Continuing after the very last row: no items, no cursor.
		lastTaken := pgtype.Timestamptz{Time: tie, Valid: true}
		lastCursor := encodeTimelineCursor(lastTaken, pgUUID(tieLow))
		page := env.getTimeline(t, env.cookie, "limit=2&cursor="+lastCursor)
		requireTimelineIDs(t, page.Items)
		if page.NextCursor != nil {
			t.Error("nextCursor set on the page after the last row, want absent")
		}
	})

	t.Run("LimitClampedToMaximum", func(t *testing.T) {
		// 205 further assets push the visible count past the contract
		// maximum; a request for 999 must answer the clamped page of
		// 200 items with a follow-up cursor.
		for i := 0; i < 205; i++ {
			id := bulkAssetID(i)
			taken := base.Add(time.Duration(10+i) * time.Minute)
			env.insertTimelineAsset(t, id, adminID, taken, timelineAssetOpts{})
			env.addTimelineItem(t, adminSync, id, adminID)
		}

		page := env.getTimeline(t, env.cookie, "limit=999")
		if len(page.Items) != timelineMaxLimit {
			t.Fatalf("limit=999 answered %d items, want the clamped %d", len(page.Items), timelineMaxLimit)
		}
		if page.NextCursor == nil {
			t.Error("nextCursor absent on a full clamped page with more rows after it, want present")
		}
	})
}

// TestTimelinePersonScope pins the person detail's wall data: the
// clusterId narrow answers one face cluster's own photos -- the
// deduped live assets carrying a member face, in the timeline's
// ordering with the caller's own favourite flags -- and nothing else:
// an invisible cluster reads as the empty list (no existence leak), a
// trashed or hidden asset's face never counts, and the two narrows
// refuse to combine.
func TestTimelinePersonScope(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	adminID := env.sessionUserID(t)
	graceID, graceCookie := env.createSecondUser(t, "person-scope-grace@example.com")
	env.insertPeopleModel(t)

	// The admin's four assets: two live, one trashed, one hidden.
	// takenAt ascends with the suffix, so the walk reads newest first.
	a1 := fixturePeopleAssetID("91")
	a2 := fixturePeopleAssetID("92")
	aTrash := fixturePeopleAssetID("93")
	aHidden := fixturePeopleAssetID("94")
	env.insertOwnedAsset(t, a1, adminID, time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC), timelineAssetOpts{})
	env.insertOwnedAsset(t, a2, adminID, time.Date(2026, 7, 2, 12, 0, 0, 0, time.UTC), timelineAssetOpts{})
	env.insertOwnedAsset(t, aTrash, adminID, time.Date(2026, 7, 3, 12, 0, 0, 0, time.UTC), timelineAssetOpts{deleted: true})
	env.insertOwnedAsset(t, aHidden, adminID, time.Date(2026, 7, 4, 12, 0, 0, 0, time.UTC), timelineAssetOpts{isHidden: true})

	// The cluster: two faces on a1 (the wall dedupes by asset), one
	// on a2, and one each on the trashed and hidden assets.
	f1 := fixturePeopleFaceID("91")
	f2 := fixturePeopleFaceID("92")
	f3 := fixturePeopleFaceID("93")
	f4 := fixturePeopleFaceID("94")
	f5 := fixturePeopleFaceID("95")
	env.insertFace(t, f1, a1, 0.90)
	env.insertFace(t, f2, a1, 0.80)
	env.insertFace(t, f3, a2, 0.95)
	env.insertFace(t, f4, aTrash, 0.99)
	env.insertFace(t, f5, aHidden, 0.98)
	cluster := fixturePeopleClusterID("91")
	env.seedPersonCluster(t, cluster, f1, f2, f3, f4, f5)

	// A second cluster whose only face sits on Grace's asset: the
	// admin's walk of it is the empty list.
	foreign := fixturePeopleClusterID("92")
	foreignFace := fixturePeopleFaceID("96")
	env.insertOwnedAsset(t, fixturePeopleAssetID("95"), graceID,
		time.Date(2026, 7, 5, 12, 0, 0, 0, time.UTC), timelineAssetOpts{})
	env.insertFace(t, foreignFace, fixturePeopleAssetID("95"), 0.90)
	env.seedPersonCluster(t, foreign, foreignFace)

	// The admin favourites a1: the per-user flag rides the person's
	// walk exactly like the aggregate's.
	if _, err := env.pool.Exec(t.Context(),
		"INSERT INTO user_favorites (user_id, asset_id) VALUES ($1, $2)",
		pgUUID(adminID), pgUUID(a1)); err != nil {
		t.Fatalf("seed the favourite row: %v", err)
	}

	t.Run("PersonWalkIsDedupedAndLive", func(t *testing.T) {
		page := env.getTimeline(t, env.cookie, "clusterId="+cluster.String())
		// Newest first: a2 (one face), then a1 (two faces, ONE item);
		// the trashed and hidden assets' faces never surface.
		requireTimelineIDs(t, page.Items, a2, a1)
		if !page.Items[1].IsFavorite {
			t.Error("a1 isFavorite = false, want the caller's own favourite flag")
		}
		if page.Items[0].IsFavorite {
			t.Error("a2 isFavorite = true, want false")
		}
	})

	t.Run("InvisibleOrUnknownClusterIsEmpty", func(t *testing.T) {
		for _, id := range []uuid.UUID{foreign, fixturePeopleClusterID("ff")} {
			page := env.getTimeline(t, env.cookie, "clusterId="+id.String())
			requireTimelineIDs(t, page.Items)
			if page.NextCursor != nil {
				t.Errorf("cluster %s: nextCursor set on the empty walk, want absent", id)
			}
		}
		// Grace reads her own cluster; the admin's stays empty for her.
		page := env.getTimeline(t, graceCookie, "clusterId="+foreign.String())
		requireTimelineIDs(t, page.Items, fixturePeopleAssetID("95"))
	})

	t.Run("BothNarrowsRejected", func(t *testing.T) {
		album := env.syncAlbumIDOf(t, adminID)
		rec := env.getTimelineRaw(t, env.cookie, "albumId="+album.String()+"&clusterId="+cluster.String())
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d (body %q)", rec.Code, http.StatusBadRequest, rec.Body.String())
		}
		apiErr, fields := decodeError(t, rec)
		if apiErr.Code != apigen.VALIDATIONFAILED {
			t.Errorf("code = %q, want %q", apiErr.Code, apigen.VALIDATIONFAILED)
		}
		// Only the narrows the caller sent: naming placeIds (absent
		// from this request) would point at a field never written.
		if len(fields) != 2 || !contains(fields, "clusterId") || !contains(fields, "albumId") {
			t.Errorf("details.fields = %v, want exactly the two narrows sent", fields)
		}
		if contains(fields, "placeIds") {
			t.Errorf("details.fields = %v, want the absent placeIds unnamed", fields)
		}
	})

	t.Run("GarbageClusterIDIsRejected", func(t *testing.T) {
		rec := env.getTimelineRaw(t, env.cookie, "clusterId=not-a-uuid")
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d (body %q)", rec.Code, http.StatusBadRequest, rec.Body.String())
		}
	})
}

// TestTimelinePlaceScope pins the place domain's wall data: the
// placeIds narrow answers the live assets resolved to those places
// among the CALLER's visible union -- deduped by asset, in the
// timeline's ordering with the caller's own favourite flags -- and
// nothing else: another account's unshared asset at the same place
// never surfaces, an id no visible asset resolves to reads as the
// empty list, and the narrows refuse to combine.
func TestTimelinePlaceScope(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	adminID := env.sessionUserID(t)
	graceID, _ := env.createSecondUser(t, "place-scope-grace@example.com")
	env.insertPlacesFixture(t)

	// The admin's four Paris assets: two live, one trashed, one
	// hidden. takenAt ascends with the suffix, so the walk reads
	// newest first.
	a1 := placeAssetID("31")
	a2 := placeAssetID("32")
	aTrash := placeAssetID("33")
	aHidden := placeAssetID("34")
	env.insertOwnedAsset(t, a1, adminID, time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC), timelineAssetOpts{})
	env.insertOwnedAsset(t, a2, adminID, time.Date(2026, 7, 2, 12, 0, 0, 0, time.UTC), timelineAssetOpts{})
	env.insertOwnedAsset(t, aTrash, adminID, time.Date(2026, 7, 3, 12, 0, 0, 0, time.UTC), timelineAssetOpts{deleted: true})
	env.insertOwnedAsset(t, aHidden, adminID, time.Date(2026, 7, 4, 12, 0, 0, 0, time.UTC), timelineAssetOpts{isHidden: true})
	env.exec(t, "UPDATE assets SET place_id = $1 WHERE id = ANY($2)",
		placeParisID, []uuid.UUID{a1, a2, aTrash, aHidden})

	// The admin's one Beijing asset: the union of two places is the
	// map panel's bubble read.
	aBeijing := placeAssetID("35")
	env.insertOwnedAsset(t, aBeijing, adminID, time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC), timelineAssetOpts{})
	env.exec(t, "UPDATE assets SET place_id = $1 WHERE id = $2", placeBeijingID, aBeijing)

	// Grace's own Paris asset: visible to her walk, never the admin's.
	gParis := placeAssetID("36")
	env.insertOwnedAsset(t, gParis, graceID, time.Date(2026, 7, 5, 12, 0, 0, 0, time.UTC), timelineAssetOpts{})
	env.exec(t, "UPDATE assets SET place_id = $1 WHERE id = $2", placeParisID, gParis)

	// The admin favourites a1: the per-user flag rides the place walk.
	if _, err := env.pool.Exec(t.Context(),
		"INSERT INTO user_favorites (user_id, asset_id) VALUES ($1, $2)",
		pgUUID(adminID), pgUUID(a1)); err != nil {
		t.Fatalf("seed the favourite row: %v", err)
	}

	t.Run("PlaceWalkIsLiveAndCallerScoped", func(t *testing.T) {
		page := env.getTimeline(t, env.cookie, fmt.Sprintf("placeIds=%d", placeParisID))
		requireTimelineIDs(t, page.Items, a2, a1)
		if !page.Items[1].IsFavorite {
			t.Error("a1 isFavorite = false, want the caller's own favourite flag")
		}
		if page.Items[0].IsFavorite {
			t.Error("a2 isFavorite = true, want false")
		}
	})

	t.Run("BubbleUnionIsOneWalk", func(t *testing.T) {
		// The map panel's read: the union of a bubble's places in the
		// same order, deduplicated.
		page := env.getTimeline(t, env.cookie,
			fmt.Sprintf("placeIds=%d,%d", placeParisID, placeBeijingID))
		requireTimelineIDs(t, page.Items, a2, a1, aBeijing)
	})

	t.Run("UnknownPlaceIsEmpty", func(t *testing.T) {
		page := env.getTimeline(t, env.cookie, "placeIds=999999999")
		requireTimelineIDs(t, page.Items)
		if page.NextCursor != nil {
			t.Error("nextCursor set on the empty walk, want absent")
		}
	})

	t.Run("NarrowCombinationRejected", func(t *testing.T) {
		album := env.syncAlbumIDOf(t, adminID)
		rec := env.getTimelineRaw(t, env.cookie,
			fmt.Sprintf("albumId=%s&placeIds=%d", album.String(), placeParisID))
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d (body %q)", rec.Code, http.StatusBadRequest, rec.Body.String())
		}
		apiErr, fields := decodeError(t, rec)
		if apiErr.Code != apigen.VALIDATIONFAILED {
			t.Errorf("code = %q, want %q", apiErr.Code, apigen.VALIDATIONFAILED)
		}
		if len(fields) != 2 || !contains(fields, "albumId") || !contains(fields, "placeIds") {
			t.Errorf("details.fields = %v, want exactly the two narrows sent", fields)
		}
		if contains(fields, "clusterId") {
			t.Errorf("details.fields = %v, want the absent clusterId unnamed", fields)
		}
	})

	t.Run("MalformedPlaceIDsRejected", func(t *testing.T) {
		for _, query := range []string{"placeIds=0", "placeIds=-5", "placeIds=abc", "placeIds="} {
			rec := env.getTimelineRaw(t, env.cookie, query)
			if rec.Code != http.StatusBadRequest {
				t.Errorf("%s: status = %d, want %d (body %q)", query, rec.Code, http.StatusBadRequest, rec.Body.String())
			}
		}
	})
}
