// The favorites and hidden view tests (M2c Task 4): the two curated
// views' listings (selection, live filters, per-user isolation, the
// per-user isFavorite flag), their keyset pagination (the same
// machinery the timeline uses), and the isFavorite computation on the
// timeline itself -- the same asset a favorite for one viewer and a
// plain tile for another.
package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	apigen "github.com/welpher/lumemor/server/internal/api/gen"
)

// getFavorites fires the favorites view request and decodes the 200
// page (nil cookie sends it unauthenticated).
func (e *assetEnv) getFavorites(t *testing.T, cookie *http.Cookie, rawQuery string) apigen.TimelineResponse {
	t.Helper()
	rec := e.getFavoritesRaw(t, cookie, rawQuery)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /assets/favorites?%s status = %d, want %d (body %q)",
			rawQuery, rec.Code, http.StatusOK, rec.Body.String())
	}
	var response apigen.TimelineResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode favorites body %q: %v", rec.Body.String(), err)
	}
	return response
}

// getFavoritesRaw fires the favorites view request and returns the
// recorder.
func (e *assetEnv) getFavoritesRaw(t *testing.T, cookie *http.Cookie, rawQuery string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/assets/favorites", nil)
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

// getHidden fires the hidden view request and decodes the 200 page.
func (e *assetEnv) getHidden(t *testing.T, cookie *http.Cookie, rawQuery string) apigen.TimelineResponse {
	t.Helper()
	rec := e.getHiddenRaw(t, cookie, rawQuery)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /assets/hidden?%s status = %d, want %d (body %q)",
			rawQuery, rec.Code, http.StatusOK, rec.Body.String())
	}
	var response apigen.TimelineResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode hidden body %q: %v", rec.Body.String(), err)
	}
	return response
}

// getHiddenRaw fires the hidden view request and returns the
// recorder.
func (e *assetEnv) getHiddenRaw(t *testing.T, cookie *http.Cookie, rawQuery string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/assets/hidden", nil)
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

// requireFavoriteFlags asserts every item of the page carries the
// given isFavorite value.
func requireFavoriteFlags(t *testing.T, items []apigen.TimelineItem, want bool) {
	t.Helper()
	for _, item := range items {
		if item.IsFavorite != want {
			t.Errorf("item %s carries isFavorite %v, want %v", item.Id, item.IsFavorite, want)
		}
	}
}

// TestFavoritesView pins the favorites listing: the caller's
// favorites rows joined to live assets (hidden and trashed favorites
// filtered out until unhidden/restored, the favorite row riding both
// states out), every item isFavorite true by construction, the
// per-user isolation between accounts, and the unfavorite removal.
func TestFavoritesView(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	adminID := env.sessionUserID(t)
	graceID, graceCookie := env.createSecondUser(t, "favorites-grace@example.com")

	taken := time.Date(2026, 8, 1, 10, 0, 0, 0, time.UTC)
	fav1 := fixtureAssetID("91")
	fav2 := fixtureAssetID("92")
	fav3 := fixtureAssetID("93")
	favHidden := fixtureAssetID("94")
	favTrashed := fixtureAssetID("95")
	plain := fixtureAssetID("96")
	env.insertTimelineAsset(t, fav1, adminID, taken.Add(4*time.Minute), timelineAssetOpts{})
	env.insertTimelineAsset(t, fav2, adminID, taken.Add(3*time.Minute), timelineAssetOpts{})
	env.insertTimelineAsset(t, fav3, adminID, taken.Add(2*time.Minute), timelineAssetOpts{})
	env.insertTimelineAsset(t, favHidden, adminID, taken.Add(time.Minute), timelineAssetOpts{})
	env.insertTimelineAsset(t, favTrashed, adminID, taken, timelineAssetOpts{})
	env.insertTimelineAsset(t, plain, adminID, taken.Add(-time.Minute), timelineAssetOpts{})

	// Five favorites (the plain asset stays unfavorited), then one
	// hidden and one trashed: the favorite rows ride both states out
	// -- the view filters on the asset's state, the rows survive.
	env.batchAssets(t, env.cookie, "favorite", fav1, fav2, fav3, favHidden, favTrashed)
	env.batchAssets(t, env.cookie, "hide", favHidden)
	env.batchAssets(t, env.cookie, "trash", favTrashed)

	// The view lists exactly the live favorites, newest capture
	// first, every item isFavorite true by construction.
	page := env.getFavorites(t, env.cookie, "")
	requireTimelineIDs(t, page.Items, fav1, fav2, fav3)
	requireFavoriteFlags(t, page.Items, true)
	if page.NextCursor != nil {
		t.Error("nextCursor set on the whole-favorites page, want absent")
	}

	// Grace's favorites are hers: her empty view stays empty of the
	// admin's favorites, and her own favorite shows in her view alone.
	graceFav := fixtureAssetID("97")
	env.insertTimelineAsset(t, graceFav, graceID, taken.Add(5*time.Minute), timelineAssetOpts{})
	env.batchAssets(t, graceCookie, "favorite", graceFav)
	page = env.getFavorites(t, graceCookie, "")
	requireTimelineIDs(t, page.Items, graceFav)
	requireFavoriteFlags(t, page.Items, true)
	page = env.getFavorites(t, env.cookie, "")
	requireTimelineIDs(t, page.Items, fav1, fav2, fav3)

	// Unfavoriting removes the row and with it the item.
	env.batchAssets(t, env.cookie, "unfavorite", fav2)
	page = env.getFavorites(t, env.cookie, "")
	requireTimelineIDs(t, page.Items, fav1, fav3)

	// The hidden favorite returns with the unhide, the trashed one
	// with the restore: the favorite row was never the thing that
	// left. (taken_at descending: fav1, fav3, then the +1 minute
	// favHidden, then favTrashed.)
	env.batchAssets(t, env.cookie, "unhide", favHidden)
	env.batchAssets(t, env.cookie, "restore", favTrashed)
	page = env.getFavorites(t, env.cookie, "")
	requireTimelineIDs(t, page.Items, fav1, fav3, favHidden, favTrashed)
}

// TestFavoritesPagination walks the keyset pagination of the favorites
// view: the same parameters, default and cap, the same cursor
// semantics as the timeline, the disjoint union of the pages, and the
// validation door the shared parameter parsing answers.
func TestFavoritesPagination(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	adminID := env.sessionUserID(t)

	base := time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC)
	ids := make([]uuid.UUID, 5)
	for i := range ids {
		ids[i] = fixtureAssetID("a" + string(rune('1'+i)))
		env.insertTimelineAsset(t, ids[i], adminID, base.Add(time.Duration(5-i)*time.Minute), timelineAssetOpts{})
	}
	env.batchAssets(t, env.cookie, "favorite", ids...)

	// Walk limit-2 pages: 2 + 2 + 1, the disjoint union in the
	// taken_at-descending order.
	seen := make(map[uuid.UUID]bool)
	var walked []uuid.UUID
	cursor := ""
	pages := 0
	for {
		rawQuery := "limit=2"
		if cursor != "" {
			rawQuery += "&cursor=" + cursor
		}
		page := env.getFavorites(t, env.cookie, rawQuery)
		for _, id := range timelineItemIDs(page.Items) {
			if seen[id] {
				t.Fatalf("asset %s appeared on two favorites pages (cursor %q)", id, cursor)
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
		t.Fatalf("walked %d favorites pages, want 3 (2+2+1)", pages)
	}
	for i, want := range ids {
		if walked[i] != want {
			t.Fatalf("favorites walk order[%d] = %s, want %s (walked %v)", i, walked[i], want, walked)
		}
	}

	// Continuing after the very last row: no items, no cursor.
	last := encodeTimelineCursor(pgTimestamptz(base.Add(time.Minute)), pgUUID(ids[4]))
	page := env.getFavorites(t, env.cookie, "limit=2&cursor="+last)
	requireTimelineIDs(t, page.Items)
	if page.NextCursor != nil {
		t.Error("nextCursor set on the favorites page after the last row, want absent")
	}

	// The default page size (no limit parameter) and the empty view.
	page = env.getFavorites(t, env.cookie, "")
	if len(page.Items) != 5 {
		t.Fatalf("favorites page without limit carries %d items, want 5 (the default fits)", len(page.Items))
	}
	env.batchAssets(t, env.cookie, "unfavorite", ids...)
	page = env.getFavorites(t, env.cookie, "")
	requireTimelineIDs(t, page.Items)
	if page.NextCursor != nil {
		t.Error("nextCursor set on the empty favorites view, want absent")
	}

	// The shared validation door: the same 400s the timeline answers.
	for _, raw := range []string{"limit=0", "cursor=not-a-cursor"} {
		if rec := env.getFavoritesRaw(t, env.cookie, raw); rec.Code != http.StatusBadRequest {
			t.Errorf("GET /assets/favorites?%s status = %d, want %d", raw, rec.Code, http.StatusBadRequest)
		}
	}
	if rec := env.getFavoritesRaw(t, nil, ""); rec.Code != http.StatusUnauthorized {
		t.Errorf("unauthenticated favorites view status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

// pgTimestamptz builds the timestamptz wire value of a time.
func pgTimestamptz(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t, Valid: true}
}

// TestHiddenView pins the hidden listing: the caller's own hidden
// rows (owner scope, live rows only, no Live Photo component rows),
// the isFavorite flag still reporting the favorites state of a hidden
// asset, the trashed-hidden row staying in the trash until restored
// -- and returning here still hidden -- and the unhide returning an
// asset to the timeline.
func TestHiddenView(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	adminID := env.sessionUserID(t)
	sync := env.syncAlbumIDOf(t, adminID)
	graceID, graceCookie := env.createSecondUser(t, "hidden-grace@example.com")

	taken := time.Date(2026, 8, 3, 10, 0, 0, 0, time.UTC)
	h1 := fixtureAssetID("b1") // hidden, favorited: the isFavorite flag of a hidden asset
	h2 := fixtureAssetID("b2") // hidden, then trashed, then restored: still hidden afterwards
	h3 := fixtureAssetID("b3") // hidden, then unhidden: returns to the timeline
	env.insertTimelineAsset(t, h1, adminID, taken.Add(2*time.Minute), timelineAssetOpts{})
	env.insertTimelineAsset(t, h2, adminID, taken.Add(time.Minute), timelineAssetOpts{})
	env.insertTimelineAsset(t, h3, adminID, taken, timelineAssetOpts{})
	for _, id := range []uuid.UUID{h1, h2, h3} {
		env.addTimelineItem(t, sync, id, adminID)
	}
	// The component row (a Live Photo's video half) never lists even
	// when its is_hidden flag is set: every view filters component
	// rows, the pair is represented by its photo.
	component := fixtureAssetID("b4")
	env.insertTimelineAsset(t, component, adminID, taken.Add(3*time.Minute), timelineAssetOpts{isComponent: true, isHidden: true})
	// Grace's hidden asset is hers: the view is owner-scoped.
	graceHidden := fixtureAssetID("b5")
	env.insertTimelineAsset(t, graceHidden, graceID, taken.Add(4*time.Minute), timelineAssetOpts{})
	env.batchAssets(t, graceCookie, "hide", graceHidden)

	// Hide all three and favorite one: the flag survives the hiding.
	env.batchAssets(t, env.cookie, "hide", h1, h2, h3)
	env.batchAssets(t, env.cookie, "favorite", h1)

	page := env.getHidden(t, env.cookie, "")
	requireTimelineIDs(t, page.Items, h1, h2, h3)
	if len(page.Items) != 3 {
		t.Fatalf("hidden view holds %d items, want 3", len(page.Items))
	}
	if !page.Items[0].IsFavorite {
		t.Error("the hidden favorited asset carries isFavorite false, want true (the favorite row survives the hiding)")
	}
	if page.Items[1].IsFavorite || page.Items[2].IsFavorite {
		t.Error("the unfavorited hidden assets carry isFavorite true, want false")
	}

	// The owner scope: grace's hidden asset never lists for the
	// admin, and hers holds exactly her own.
	page = env.getHidden(t, graceCookie, "")
	requireTimelineIDs(t, page.Items, graceHidden)

	// The timeline holds none of the hidden rows.
	timeline := env.getTimeline(t, env.cookie, "")
	requireTimelineIDs(t, timeline.Items)

	// A trashed hidden row leaves the hidden view (the trash owns it
	// until restored or expired) and lands in the trash listing.
	env.batchAssets(t, env.cookie, "trash", h2)
	page = env.getHidden(t, env.cookie, "")
	requireTimelineIDs(t, page.Items, h1, h3)
	if items := env.getTrash(t, env.cookie).Items; len(items) != 1 || uuid.UUID(items[0].Id) != h2 {
		t.Errorf("trash after the trashed-hidden batch holds %v, want exactly %s", trashItemIDs(items), h2)
	}

	// The restore returns the row still hidden: it re-enters the
	// hidden view, not the timeline.
	env.batchAssets(t, env.cookie, "restore", h2)
	page = env.getHidden(t, env.cookie, "")
	requireTimelineIDs(t, page.Items, h1, h2, h3)
	timeline = env.getTimeline(t, env.cookie, "")
	requireTimelineIDs(t, timeline.Items)

	// The unhide returns the asset to every live view.
	env.batchAssets(t, env.cookie, "unhide", h1)
	page = env.getHidden(t, env.cookie, "")
	requireTimelineIDs(t, page.Items, h2, h3)
	timeline = env.getTimeline(t, env.cookie, "")
	requireTimelineIDs(t, timeline.Items, h1)

	// The pagination: the hidden view pages like the timeline (h2 and
	// h3 remain hidden, so limit-1 pages split them).
	page = env.getHidden(t, env.cookie, "limit=1")
	requireTimelineIDs(t, page.Items, h2)
	if page.NextCursor == nil {
		t.Fatal("nextCursor absent on a full hidden page with a row behind it, want present")
	}
	next := env.getHidden(t, env.cookie, "limit=1&cursor="+*page.NextCursor)
	requireTimelineIDs(t, next.Items, h3)
	if next.NextCursor != nil {
		t.Error("nextCursor set on the final hidden page, want absent")
	}

	// The validation door and the authentication requirement.
	for _, raw := range []string{"limit=0", "cursor=not-a-cursor"} {
		if rec := env.getHiddenRaw(t, env.cookie, raw); rec.Code != http.StatusBadRequest {
			t.Errorf("GET /assets/hidden?%s status = %d, want %d", raw, rec.Code, http.StatusBadRequest)
		}
	}
	if rec := env.getHiddenRaw(t, nil, ""); rec.Code != http.StatusUnauthorized {
		t.Errorf("unauthenticated hidden view status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

// TestTimelineIsFavorite pins the per-user isFavorite computation on
// the timeline itself: the flag reports the REQUESTING user's
// favorites rows, so the same shared asset is a favorite tile for its
// owner and a plain tile for the viewer it was shared to.
func TestTimelineIsFavorite(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	adminID := env.sessionUserID(t)
	graceID, graceCookie := env.createSecondUser(t, "timeline-fav-grace@example.com")

	// Grace's asset g1 reaches the admin through an album shared to
	// him, and stays in grace's own sync stream; a1 is the admin's
	// own, visible to him alone.
	adminSync := env.syncAlbumIDOf(t, adminID)
	graceSync := fixtureAlbumID("c1")
	env.insertTimelineAlbum(t, graceSync, graceID, "sync")
	toAdmin := fixtureAlbumID("c2")
	env.insertTimelineAlbum(t, toAdmin, graceID, "virtual")
	env.shareTimelineAlbum(t, toAdmin, adminID, graceID)

	taken := time.Date(2026, 8, 4, 10, 0, 0, 0, time.UTC)
	g1 := fixtureAssetID("c3")
	a1 := fixtureAssetID("c4")
	env.insertTimelineAsset(t, g1, graceID, taken, timelineAssetOpts{})
	env.addTimelineItem(t, graceSync, g1, graceID)
	env.addTimelineItem(t, toAdmin, g1, graceID)
	env.insertTimelineAsset(t, a1, adminID, taken.Add(time.Minute), timelineAssetOpts{})
	env.addTimelineItem(t, adminSync, a1, adminID)

	// Each favorites their own asset (the batch actions are
	// owner-scoped in this milestone).
	env.batchAssets(t, env.cookie, "favorite", a1)
	env.batchAssets(t, graceCookie, "favorite", g1)

	// The admin's timeline: his favorite a1 carries the flag, the
	// shared g1 does not -- grace's favorite is grace's state, not
	// his.
	adminPage := env.getTimeline(t, env.cookie, "")
	requireTimelineIDs(t, adminPage.Items, a1, g1)
	if !adminPage.Items[0].IsFavorite {
		t.Error("the admin's own favorite a1 carries isFavorite false on his timeline, want true")
	}
	if adminPage.Items[1].IsFavorite {
		t.Error("the shared asset g1 carries isFavorite true on the admin's timeline, want false (grace's favorite is per-user state)")
	}

	// Grace's timeline: her favorite g1 carries the flag for her.
	gracePage := env.getTimeline(t, graceCookie, "")
	requireTimelineIDs(t, gracePage.Items, g1)
	if !gracePage.Items[0].IsFavorite {
		t.Error("grace's favorite g1 carries isFavorite false on her timeline, want true")
	}

	// The favorites views agree: each account's view holds its own
	// favorite, the other's holds nothing of it.
	if page := env.getFavorites(t, env.cookie, ""); len(page.Items) != 1 || uuid.UUID(page.Items[0].Id) != a1 {
		t.Errorf("admin's favorites view = %v, want exactly a1", timelineItemIDs(page.Items))
	}
	if page := env.getFavorites(t, graceCookie, ""); len(page.Items) != 1 || uuid.UUID(page.Items[0].Id) != g1 {
		t.Errorf("grace's favorites view = %v, want exactly g1", timelineItemIDs(page.Items))
	}
}
