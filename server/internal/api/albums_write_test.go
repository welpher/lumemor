// Album surface write tests (reshaped by the albums-as-sources
// rework): the directory album creation behind POST
// /albums (title + the validated source path + the audience + the
// two dialog switches, with the honest skip report of unusable
// audience entries), the property patch (with the cover's
// live-member validation) and the delete (memory only, cascades)
// and the membership removal with the batch's honest
// {removed, skipped} report. The two-tier authorization answer
// runs through every write: an invisible album answers the reads'
// byte-identical 404, an album shared to the caller answers the
// honest 403, and an administrator overrides the owner check. The
// live album.* events carry ids only -- no album state rides
// the shared stream. Fixtures insert albums, memberships and assets
// directly in the database; every write goes through the API. The
// path validation matrix and the completion domain live in
// albumroots_test.go.
package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	apigen "github.com/welpher/lumemor/server/internal/api/gen"
)

// albumIDList builds a list of quoted ids for the audience and
// membership bodies.
func albumIDList(ids []uuid.UUID) string {
	quoted := make([]string, len(ids))
	for i, id := range ids {
		quoted[i] = `"` + id.String() + `"`
	}
	return strings.Join(quoted, ",")
}

// albumItemsBody assembles the membership removal body.
func albumItemsBody(ids ...uuid.UUID) string {
	return `{"assetIds":[` + albumIDList(ids) + `]}`
}

// audienceBody assembles the audience set body; a nil audience
// omits the field, an empty slice writes the explicit empty set.
func audienceBody(ids []uuid.UUID) string {
	if ids == nil {
		return `{}`
	}
	return `{"audience":[` + albumIDList(ids) + `]}`
}

// writeAlbumRaw is the album write dispatcher: an empty body sends no
// request content at all, anything else goes out as JSON. A nil
// cookie sends the request unauthenticated.
func (e *assetEnv) writeAlbumRaw(t *testing.T, cookie *http.Cookie, method, target, body string) *httptest.ResponseRecorder {
	t.Helper()
	var req *http.Request
	if body == "" {
		req = httptest.NewRequest(method, target, nil)
	} else {
		req = httptest.NewRequest(method, target, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	}
	if cookie != nil {
		req.AddCookie(cookie)
	}
	rec := httptest.NewRecorder()
	e.router.ServeHTTP(rec, req)
	return rec
}

// createAlbumBody assembles the creation body: the title, the source
// path, and the audience (a nil slice omits the field, an empty
// slice writes the explicit empty set).
func createAlbumBody(t *testing.T, title, rootPath string, audience []uuid.UUID) string {
	t.Helper()
	quotedTitle, err := json.Marshal(title)
	if err != nil {
		t.Fatalf("quote title %q: %v", title, err)
	}
	quotedPath, err := json.Marshal(rootPath)
	if err != nil {
		t.Fatalf("quote rootPath %q: %v", rootPath, err)
	}
	body := `{"title":` + string(quotedTitle) + `,"rootPath":` + string(quotedPath)
	if audience != nil {
		body += `,"audience":[` + albumIDList(audience) + `]`
	}
	return body + "}"
}

// createAlbumRaw fires POST /albums with the raw body (a nil cookie
// sends it unauthenticated).
func (e *assetEnv) createAlbumRaw(t *testing.T, cookie *http.Cookie, body string) *httptest.ResponseRecorder {
	t.Helper()
	return e.writeAlbumRaw(t, cookie, http.MethodPost, "/api/v1/albums", body)
}

// createAlbum fires POST /albums over one source path and decodes
// the 201 body; a nil audience omits the field.
func (e *assetEnv) createAlbum(t *testing.T, cookie *http.Cookie, title, rootPath string, audience []uuid.UUID) apigen.AlbumCreateResponse {
	t.Helper()
	rec := e.createAlbumRaw(t, cookie, createAlbumBody(t, title, rootPath, audience))
	if rec.Code != http.StatusCreated {
		t.Fatalf("POST /albums status = %d, want %d (body %q)",
			rec.Code, http.StatusCreated, rec.Body.String())
	}
	var response apigen.AlbumCreateResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode create album body %q: %v", rec.Body.String(), err)
	}
	return response
}

// patchAlbumRaw fires PATCH /albums/{albumId} with the raw body.
func (e *assetEnv) patchAlbumRaw(t *testing.T, cookie *http.Cookie, albumID uuid.UUID, body string) *httptest.ResponseRecorder {
	t.Helper()
	return e.writeAlbumRaw(t, cookie, http.MethodPatch, "/api/v1/albums/"+albumID.String(), body)
}

// patchAlbum fires PATCH /albums/{albumId} and decodes the 200 body.
func (e *assetEnv) patchAlbum(t *testing.T, cookie *http.Cookie, albumID uuid.UUID, body string) apigen.AlbumSummary {
	t.Helper()
	rec := e.patchAlbumRaw(t, cookie, albumID, body)
	if rec.Code != http.StatusOK {
		t.Fatalf("PATCH /albums/%s status = %d, want %d (body %q)",
			albumID, rec.Code, http.StatusOK, rec.Body.String())
	}
	var summary apigen.AlbumSummary
	if err := json.Unmarshal(rec.Body.Bytes(), &summary); err != nil {
		t.Fatalf("decode patch album body %q: %v", rec.Body.String(), err)
	}
	return summary
}

// deleteAlbumRaw fires DELETE /albums/{albumId}.
func (e *assetEnv) deleteAlbumRaw(t *testing.T, cookie *http.Cookie, albumID uuid.UUID) *httptest.ResponseRecorder {
	t.Helper()
	return e.writeAlbumRaw(t, cookie, http.MethodDelete, "/api/v1/albums/"+albumID.String(), "")
}

// deleteAlbumItems fires DELETE /albums/{albumId}/items with the raw
// body.
func (e *assetEnv) deleteAlbumItems(t *testing.T, cookie *http.Cookie, albumID uuid.UUID, body string) *httptest.ResponseRecorder {
	t.Helper()
	return e.writeAlbumRaw(t, cookie, http.MethodDelete,
		"/api/v1/albums/"+albumID.String()+"/items", body)
}

// decodeRemoveItemsReport decodes the 200 remove report.
func decodeRemoveItemsReport(t *testing.T, rec *httptest.ResponseRecorder) apigen.RemoveAlbumItemsResponse {
	t.Helper()
	if rec.Code != http.StatusOK {
		t.Fatalf("DELETE /albums/{id}/items status = %d, want %d (body %q)",
			rec.Code, http.StatusOK, rec.Body.String())
	}
	var report apigen.RemoveAlbumItemsResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &report); err != nil {
		t.Fatalf("decode remove items body %q: %v", rec.Body.String(), err)
	}
	return report
}

// requireAlbumWriteForbidden asserts the honest 403 of the two-tier
// answer: the caller can see the album (a share row exists) but the
// share is view-only.
func requireAlbumWriteForbidden(t *testing.T, rec *httptest.ResponseRecorder) {
	t.Helper()
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d (body %q)",
			rec.Code, http.StatusForbidden, rec.Body.String())
	}
	apiErr, _ := decodeError(t, rec)
	if apiErr.Code != apigen.AUTHFORBIDDEN {
		t.Errorf("code = %q, want %q", apiErr.Code, apigen.AUTHFORBIDDEN)
	}
}

// requireAlbumCoverNotMember asserts the 400 VALIDATION_FAILED whose
// details carry the album_cover_not_member message key.
func requireAlbumCoverNotMember(t *testing.T, rec *httptest.ResponseRecorder, wantCount int) {
	t.Helper()
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d (body %q)",
			rec.Code, http.StatusBadRequest, rec.Body.String())
	}
	apiErr, _ := decodeError(t, rec)
	if apiErr.Code != apigen.VALIDATIONFAILED {
		t.Errorf("code = %q, want %q", apiErr.Code, apigen.VALIDATIONFAILED)
	}
	if apiErr.Details == nil {
		t.Fatal("details absent, want the album_cover_not_member message key")
	}
	if key, _ := (*apiErr.Details)["key"]; key != "album_cover_not_member" {
		t.Errorf("details.key = %v, want album_cover_not_member", key)
	}
	if count, _ := (*apiErr.Details)["count"].(float64); int(count) != wantCount {
		t.Errorf("details.count = %v, want %d (the refused frames)", (*apiErr.Details)["count"], wantCount)
	}
}

// requireValidationFields asserts the 400 naming exactly the given
// fields.
func requireValidationFields(t *testing.T, rec *httptest.ResponseRecorder, field string) {
	t.Helper()
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d (body %q)",
			rec.Code, http.StatusBadRequest, rec.Body.String())
	}
	apiErr, fields := decodeError(t, rec)
	if apiErr.Code != apigen.VALIDATIONFAILED {
		t.Errorf("code = %q, want %q", apiErr.Code, apigen.VALIDATIONFAILED)
	}
	if !contains(fields, field) {
		t.Errorf("details.fields = %v, want it to contain %q", fields, field)
	}
}

// requireUnauthorized asserts the 401 of an unauthenticated write.
func requireUnauthorized(t *testing.T, rec *httptest.ResponseRecorder) {
	t.Helper()
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d (body %q)",
			rec.Code, http.StatusUnauthorized, rec.Body.String())
	}
}

// fixtureWriteAlbumID builds a deterministic fixture album id of the
// write tests, in the wall fixture family with a fresh suffix range.
func fixtureWriteAlbumID(suffix string) uuid.UUID {
	return fixtureWallAlbumID(suffix)
}

// TestAlbumWritesRequireSession pins the 401 of every write face:
// creation, the audience write, the patch, the delete and the
// membership removal refuse the unauthenticated caller before any
// handler logic runs.
func TestAlbumWritesRequireSession(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	albumID := fixtureWallAlbumID("ff").String()

	for _, target := range []struct {
		method string
		target string
		body   string
	}{
		{http.MethodPost, "/api/v1/albums", `{"title":"No session"}`},
		{http.MethodPut, "/api/v1/albums/" + albumID + "/audience", `{"audience":[]}`},
		{http.MethodPatch, "/api/v1/albums/" + albumID, `{"title":"No session"}`},
		{http.MethodDelete, "/api/v1/albums/" + albumID, ""},
		{http.MethodDelete, "/api/v1/albums/" + albumID + "/items", `{"assetIds":["` + fixtureWallAssetID("ff").String() + `"]}`},
	} {
		rec := env.writeAlbumRaw(t, nil, target.method, target.target, target.body)
		requireUnauthorized(t, rec)
	}
}

// TestCreateAlbum pins the creation: the ONLY creatable type is the
// directory album (the albums-as-sources core), the answer is the
// wall card of the fresh row (owner role, no cover, no members)
// widened with the honest audience report, the wall lists it, and
// the detail carries the stored source path. The title validation
// spans the trim, the rune count and the null body; the audience's
// unusable entries skip with their reasons while the album still
// lands.
func TestCreateAlbum(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	adminID := env.sessionUserID(t)
	root := env.provisionMountRoot(t)

	t.Run("CreatesAndListsOnWall", func(t *testing.T) {
		source := sourceDir(t, root, "family")
		summary := env.createAlbum(t, env.cookie, "Family archive", source, nil)

		if summary.Title != "Family archive" {
			t.Errorf("title = %q, want the stored title", summary.Title)
		}
		if summary.Type != apigen.AlbumCreateResponseTypeDirectory {
			t.Errorf("type = %q, want directory (the only creatable type)", summary.Type)
		}
		if summary.IsShared {
			t.Error("isShared = true, want false (no audience landed)")
		}
		if summary.CoverAssetIds == nil || len(summary.CoverAssetIds) != 0 {
			t.Errorf("coverAssetIds = %v, want an empty array (no members)", summary.CoverAssetIds)
		}
		if summary.ItemCount != 0 {
			t.Errorf("itemCount = %d, want 0", summary.ItemCount)
		}
		if summary.CallerRole != apigen.AlbumCreateResponseCallerRoleOwner {
			t.Errorf("callerRole = %q, want owner", summary.CallerRole)
		}
		if summary.Owner != nil {
			t.Errorf("owner = %+v, want absent on an own card", summary.Owner)
		}
		if summary.UpdatedAt.IsZero() {
			t.Error("updatedAt = zero, want the creation time")
		}
		if summary.Skipped == nil || len(summary.Skipped) != 0 {
			t.Errorf("skipped = %v, want the empty array (no audience requested)", summary.Skipped)
		}
		if summary.InTimeline == nil || !*summary.InTimeline {
			t.Errorf("inTimeline = %v, want true (the directory card is excludable)", summary.InTimeline)
		}
		if got := env.queryInt(t, "SELECT count(*) FROM albums WHERE id = $1 AND type = 'directory' AND owner_id = $2 AND root_path = $3",
			pgUUID(uuid.UUID(summary.Id)), pgUUID(adminID), source); got != 1 {
			t.Errorf("stored album rows = %d, want 1 (the row stores the resolved path %s)", got, source)
		}

		// The wall lists the new card after the pinned sync card
		// (the default album always leads the own group).
		wall := env.getAlbumWall(t, env.cookie)
		if len(wall.Albums) != 2 {
			t.Fatalf("GET /albums holds %d items, want 2 (the pinned sync + the created album; got %v)",
				len(wall.Albums), wall.Albums)
		}
		if wall.Albums[0].Type != apigen.AlbumSummaryTypeSync {
			t.Errorf("wall[0].type = %q, want the pinned sync card first", wall.Albums[0].Type)
		}
		if uuid.UUID(wall.Albums[1].Id) != uuid.UUID(summary.Id) {
			t.Fatalf("wall[1].id = %s, want the created album %s", wall.Albums[1].Id, summary.Id)
		}
		detail := env.getAlbumDetail(t, env.cookie, uuid.UUID(summary.Id), "")
		if detail.Title != "Family archive" || detail.ItemCount != 0 {
			t.Errorf("detail title/itemCount = %q/%d, want the stored title and 0", detail.Title, detail.ItemCount)
		}
		if detail.RootPath == nil || *detail.RootPath != source {
			t.Errorf("detail rootPath = %v, want the stored source %s", detail.RootPath, source)
		}
	})

	t.Run("TrimsTitle", func(t *testing.T) {
		summary := env.createAlbum(t, env.cookie, "  Weekend picks  ", sourceDir(t, root, "picks"), nil)
		if summary.Title != "Weekend picks" {
			t.Errorf("title = %q, want the trimmed %q", summary.Title, "Weekend picks")
		}
	})

	t.Run("AudienceLandsAndSkipsHonestly", func(t *testing.T) {
		graceID, _ := env.createSecondUser(t, "create-audience-grace@example.com")
		unknown := uuid.MustParse("70000000-0000-0000-0003-000000000001")
		disabledID, _ := env.createSecondUser(t, "create-audience-disabled@example.com")
		env.exec(t, "UPDATE users SET is_disabled = true WHERE id = $1", pgUUID(disabledID))
		source := sourceDir(t, root, "audience")

		// The request names the owner, a grantee, an unknown id and
		// a disabled account: one lands, three skip with reasons,
		// and the creation still answers 201.
		summary := env.createAlbum(t, env.cookie, "Audience matrix", source,
			[]uuid.UUID{adminID, graceID, unknown, disabledID, graceID})
		albumID := uuid.UUID(summary.Id)

		if summary.IsShared != true {
			t.Error("isShared = false, want true (one audience row landed)")
		}
		if got := env.shareRowsOf(t, albumID); got != 1 {
			t.Errorf("share rows = %d, want 1 (the usable grantee alone)", got)
		}
		if row := env.queryText(t,
			"SELECT granted_by::text FROM album_shares WHERE album_id = $1 AND user_id = $2",
			pgUUID(albumID), pgUUID(graceID)); row != adminID.String() {
			t.Errorf("granted_by = %s, want the creating caller %s", row, adminID)
		}
		skipped := make(map[uuid.UUID]apigen.AlbumAudienceSkipReason, len(summary.Skipped))
		for _, entry := range summary.Skipped {
			skipped[uuid.UUID(entry.UserId)] = entry.Reason
		}
		if len(skipped) != 3 {
			t.Fatalf("skipped = %v, want exactly the owner, the unknown id and the disabled account", summary.Skipped)
		}
		if reason := skipped[adminID]; reason != apigen.AlbumAudienceSkipReasonOwner {
			t.Errorf("owner skip reason = %q, want owner", reason)
		}
		for _, id := range []uuid.UUID{unknown, disabledID} {
			if reason := skipped[id]; reason != apigen.AlbumAudienceSkipReasonInvalidAccount {
				t.Errorf("skip reason of %s = %q, want invalid_account", id, reason)
			}
		}
		// The repeated grantee collapses: one row, one grant.
		if got := env.queryInt(t, "SELECT count(*) FROM album_shares WHERE album_id = $1 AND user_id = $2",
			pgUUID(albumID), pgUUID(graceID)); got != 1 {
			t.Errorf("rows for the repeated grantee = %d, want 1 (repeated ids collapse)", got)
		}
		// The creation's events: album.created once, album.shared
		// exactly once for the landed grantee, nothing for the
		// skips -- the ring scan filtered on this album's unique id
		// is independent of the earlier subtests' creations.
		if events := eventsOfAlbum(t, env.bus, "album.created", albumID); len(events) != 1 {
			t.Errorf("album.created events = %d, want 1", len(events))
		}
		events := eventsOfAlbum(t, env.bus, "album.shared", albumID)
		if len(events) != 1 {
			t.Fatalf("album.shared events = %d, want 1 (the landed grantee alone)", len(events))
		}
		if payload := events[0]; payload["userId"] != graceID.String() {
			t.Errorf("album.shared payload userId = %s, want the grantee %s", payload["userId"], graceID)
		}
		for _, name := range []string{"album.unshared", "album.updated", "album.deleted", "album.items-changed"} {
			if events := eventsOfAlbum(t, env.bus, name, albumID); len(events) != 0 {
				t.Errorf("%s events = %d, want 0 (the creation fires nothing else)", name, len(events))
			}
		}
	})

	t.Run("ValidatesTitle", func(t *testing.T) {
		unused := sourceDir(t, root, "unused")
		for _, title := range []string{
			"",                      // missing entirely
			"   ",                   // whitespace only after the trim
			strings.Repeat("a", 65), // past the bound
		} {
			rec := env.createAlbumRaw(t, env.cookie, createAlbumBody(t, title, unused, nil))
			requireValidationFields(t, rec, "title")
		}

		// The boundary itself answers, and runes (not bytes) are
		// the count: 64 ascii and a short CJK title both land (each
		// at its own source -- siblings never overlap).
		env.createAlbum(t, env.cookie, strings.Repeat("a", 64), sourceDir(t, root, "ascii"), nil)
		env.createAlbum(t, env.cookie, "家庭相册", sourceDir(t, root, "cjk"), nil)
	})
}

// TestCreateAlbumDialogSwitches pins the two create-dialog switches
// landing in the creation itself: the deletability switch stores
// onto the row (on by default, off when the request carries false),
// and an `inTimeline: false` lands the CREATOR's own preference row
// so the album is born out of their aggregated timeline -- the
// audience member's feed is their own preference row's business,
// and an absent field writes no preference row at all (no row IS
// the feeding default, the reset face's own semantics).
func TestCreateAlbumDialogSwitches(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	graceID, graceCookie := env.createSecondUser(t, "create-switches-grace@example.com")
	root := env.provisionMountRoot(t)

	createSwitched := func(t *testing.T, title, dir string, allowAppDeletion, inTimeline *bool, audience []uuid.UUID) apigen.AlbumCreateResponse {
		t.Helper()
		body := map[string]any{"title": title, "rootPath": dir}
		if allowAppDeletion != nil {
			body["allowAppDeletion"] = *allowAppDeletion
		}
		if inTimeline != nil {
			body["inTimeline"] = *inTimeline
		}
		if audience != nil {
			body["audience"] = audience
		}
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal the create body: %v", err)
		}
		rec := env.createAlbumRaw(t, env.cookie, string(raw))
		if rec.Code != http.StatusCreated {
			t.Fatalf("POST /albums status = %d, want %d (body %q)",
				rec.Code, http.StatusCreated, rec.Body.String())
		}
		var response apigen.AlbumCreateResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
			t.Fatalf("decode create album body %q: %v", rec.Body.String(), err)
		}
		return response
	}

	t.Run("Defaults", func(t *testing.T) {
		summary := createSwitched(t, "Switch defaults", sourceDir(t, root, "defaults"), nil, nil, nil)
		if got := env.queryInt(t, "SELECT allow_app_deletion::int FROM albums WHERE id = $1",
			pgUUID(uuid.UUID(summary.Id))); got != 1 {
			t.Errorf("stored allow_app_deletion = %d, want 1 (the default)", got)
		}
		if got := env.queryInt(t, "SELECT count(*) FROM user_album_prefs WHERE album_id = $1",
			pgUUID(uuid.UUID(summary.Id))); got != 0 {
			t.Errorf("preference rows = %d, want 0 (absent writes none; no row IS the feeding default)", got)
		}
		if summary.InTimeline == nil || !*summary.InTimeline {
			t.Errorf("inTimeline = %v, want true (the card feeds the timeline)", summary.InTimeline)
		}
	})

	t.Run("BothSwitchesOffLandInCreation", func(t *testing.T) {
		off := false
		summary := createSwitched(t, "Switched off", sourceDir(t, root, "switched"), &off, &off,
			[]uuid.UUID{graceID})
		albumID := uuid.UUID(summary.Id)
		if got := env.queryInt(t, "SELECT allow_app_deletion::int FROM albums WHERE id = $1",
			pgUUID(albumID)); got != 0 {
			t.Errorf("stored allow_app_deletion = %d, want 0 (the request's switch)", got)
		}
		if got := env.queryInt(t, "SELECT count(*) FROM user_album_prefs WHERE album_id = $1 AND user_id = $2 AND in_timeline = false",
			pgUUID(albumID), pgUUID(env.sessionUserID(t))); got != 1 {
			t.Errorf("creator exclusion rows = %d, want 1 (the creation carried the exclusion)", got)
		}
		// The exclusion is the CREATOR's own: the audience member's
		// feed is governed by their own (absent) preference row.
		if got := env.queryInt(t, "SELECT count(*) FROM user_album_prefs WHERE album_id = $1 AND user_id = $2",
			pgUUID(albumID), pgUUID(graceID)); got != 0 {
			t.Errorf("grantee preference rows = %d, want 0 (the exclusion never reaches the audience)", got)
		}
		// The response card carries the landed exclusion.
		if summary.InTimeline == nil || *summary.InTimeline {
			t.Errorf("inTimeline = %v, want false (the exclusion landed)", summary.InTimeline)
		}
		// The sharee's own wall card of the same album feeds their
		// timeline (their row is absent, the default).
		wall := env.getAlbumWall(t, graceCookie)
		card, ok := wallAlbumOf(wall, albumID)
		if !ok {
			t.Fatalf("the sharee wall holds no card for the shared album")
		}
		if card.InTimeline == nil || !*card.InTimeline {
			t.Errorf("the sharee card's inTimeline = %v, want true (their own default)", card.InTimeline)
		}
	})
}

// TestUpdateAlbum pins the property patch: every field lands, the
// cover write accepts only a full set of distinct live members of the
// album itself (0 clears the pin), and an unknown album answers the
// reads' single-shape 404.
func TestUpdateAlbum(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	adminID := env.sessionUserID(t)

	// A directory album: four frames, so a full pin is four ids.
	album := fixtureWriteAlbumID("71")
	env.insertWallAlbum(t, album, adminID, "directory", "Original title", wallAlbumOpts{})

	// Four live members, three non-live members, one live non-member
	// of the caller's own, and one live asset of another account.
	member := fixtureWallAssetID("71")
	older := fixtureWallAssetID("72")
	third := fixtureWallAssetID("78")
	fourth := fixtureWallAssetID("79")
	hiddenMember := fixtureWallAssetID("73")
	trashedMember := fixtureWallAssetID("74")
	componentMember := fixtureWallAssetID("75")
	nonMember := fixtureWallAssetID("76")
	base := time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)
	env.insertTimelineAsset(t, member, adminID, base, timelineAssetOpts{})
	env.insertTimelineAsset(t, older, adminID, base.Add(-time.Hour), timelineAssetOpts{})
	env.insertTimelineAsset(t, third, adminID, base.Add(-2*time.Hour), timelineAssetOpts{})
	env.insertTimelineAsset(t, fourth, adminID, base.Add(-3*time.Hour), timelineAssetOpts{})
	env.insertTimelineAsset(t, hiddenMember, adminID, base.Add(time.Hour), timelineAssetOpts{isHidden: true})
	env.insertTimelineAsset(t, trashedMember, adminID, base.Add(time.Hour), timelineAssetOpts{deleted: true})
	env.insertTimelineAsset(t, componentMember, adminID, base.Add(time.Hour), timelineAssetOpts{isComponent: true})
	env.insertTimelineAsset(t, nonMember, adminID, base.Add(time.Hour), timelineAssetOpts{})
	graceID, _ := env.createSecondUser(t, "patch-cover-grace@example.com")
	foreign := fixtureWallAssetID("77")
	env.insertTimelineAsset(t, foreign, graceID, base.Add(time.Hour), timelineAssetOpts{})
	for _, asset := range []uuid.UUID{member, older, third, fourth, hiddenMember, trashedMember, componentMember} {
		env.addTimelineItem(t, album, asset, adminID)
	}
	// pinArray renders the set in the given pick order; pinBody is
	// the one-field patch carrying it.
	pinArray := func(ids ...uuid.UUID) string {
		quoted := make([]string, 0, len(ids))
		for _, id := range ids {
			quoted = append(quoted, `"`+id.String()+`"`)
		}
		return `[` + strings.Join(quoted, ",") + `]`
	}
	pinBody := func(ids ...uuid.UUID) string {
		return `{"coverAssetIds":` + pinArray(ids...) + `}`
	}

	t.Run("PatchesEachField", func(t *testing.T) {
		summary := env.patchAlbum(t, env.cookie, album, `{"title":"Renamed"}`)
		if summary.Title != "Renamed" {
			t.Errorf("title = %q, want the patched title", summary.Title)
		}

		pin := []uuid.UUID{older, member, fourth, third}
		summary = env.patchAlbum(t, env.cookie, album, pinBody(pin...))
		// The pinned set rides the summary in pick order, and the
		// detail reports the same set as written.
		if len(summary.CoverAssetIds) != len(pin) {
			t.Fatalf("summary coverAssetIds = %v, want the four picks %v", summary.CoverAssetIds, pin)
		}
		for i, id := range pin {
			if uuid.UUID(summary.CoverAssetIds[i]) != id {
				t.Errorf("summary coverAssetIds[%d] = %s, want %s (pick order)",
					i, summary.CoverAssetIds[i], id)
			}
		}
		detail := env.getAlbumDetail(t, env.cookie, album, "")
		if len(detail.CoverAssetIds) != len(pin) {
			t.Fatalf("detail coverAssetIds = %v, want the four picks %v", detail.CoverAssetIds, pin)
		}
		for i, id := range pin {
			if uuid.UUID(detail.CoverAssetIds[i]) != id {
				t.Errorf("detail coverAssetIds[%d] = %s, want %s (pick order)", i, detail.CoverAssetIds[i], id)
			}
		}

		// The empty set clears the pin and returns the album to the
		// automatic group (the card answers a sampled group; the detail
		// answers the empty pin).
		summary = env.patchAlbum(t, env.cookie, album, `{"coverAssetIds":[]}`)
		if len(summary.CoverAssetIds) != 4 {
			t.Errorf("summary coverAssetIds after the clear = %v, want the 4 automatic frames",
				summary.CoverAssetIds)
		}
		detail = env.getAlbumDetail(t, env.cookie, album, "")
		if len(detail.CoverAssetIds) != 0 {
			t.Errorf("detail coverAssetIds after the clear = %v, want empty (automatic)", detail.CoverAssetIds)
		}
		env.patchAlbum(t, env.cookie, album, pinBody(pin...)) // leave the pin of this scene

		// The row itself carries the last patch's values.
		if got := env.queryText(t, "SELECT title FROM albums WHERE id = $1", pgUUID(album)); got != "Renamed" {
			t.Errorf("stored title = %q, want Renamed", got)
		}
	})

	t.Run("PatchesMultipleFieldsAtOnce", func(t *testing.T) {
		body := `{"title":"Combined","coverAssetIds":` + pinArray(fourth, third, member, older) + `}`
		summary := env.patchAlbum(t, env.cookie, album, body)
		if summary.Title != "Combined" {
			t.Errorf("title = %q, want Combined", summary.Title)
		}
		if len(summary.CoverAssetIds) != 4 || uuid.UUID(summary.CoverAssetIds[0]) != fourth {
			t.Errorf("coverAssetIds = %v, want the new pick order starting at %s", summary.CoverAssetIds, fourth)
		}
		env.patchAlbum(t, env.cookie, album, pinBody(member, older, third, fourth)) // leave the scene's pin
	})

	t.Run("RejectsPartialSet", func(t *testing.T) {
		// A cover is 0 or the card's frame count: a half-picked set is
		// the picker's un-saveable state, never a stored one.
		for _, ids := range [][]uuid.UUID{
			{member},
			{member, older},
			{member, older, third},
			{member, older, third, fourth, fixtureWallAssetID("80")},
		} {
			rec := env.patchAlbumRaw(t, env.cookie, album, pinBody(ids...))
			requireValidationFields(t, rec, "coverAssetIds")
		}
	})

	t.Run("RejectsRepeatedFrame", func(t *testing.T) {
		rec := env.patchAlbumRaw(t, env.cookie, album, pinBody(member, older, member, third))
		requireValidationFields(t, rec, "coverAssetIds")
	})

	t.Run("RequiresAtLeastOneField", func(t *testing.T) {
		for _, body := range []string{`{}`, `null`} {
			rec := env.patchAlbumRaw(t, env.cookie, album, body)
			requireValidationFields(t, rec, "body")
		}
	})

	t.Run("ValidatesTitle", func(t *testing.T) {
		for _, title := range []string{"", "   ", strings.Repeat("a", 65)} {
			body, err := json.Marshal(title)
			if err != nil {
				t.Fatalf("quote title: %v", err)
			}
			rec := env.patchAlbumRaw(t, env.cookie, album, `{"title":`+string(body)+`}`)
			requireValidationFields(t, rec, "title")
		}
		// The 64-rune boundary lands.
		env.patchAlbum(t, env.cookie, album, `{"title":"`+strings.Repeat("a", 64)+`"}`)
	})

	t.Run("RejectsCoverOutsideMembership", func(t *testing.T) {
		rec := env.patchAlbumRaw(t, env.cookie, album, pinBody(nonMember, member, older, third))
		requireAlbumCoverNotMember(t, rec, 1)
	})

	t.Run("RejectsCoverOfNonLiveMember", func(t *testing.T) {
		// A hidden, trashed or component member row is a member no
		// more: the cover must be a LIVE member.
		for _, asset := range []uuid.UUID{hiddenMember, trashedMember, componentMember} {
			rec := env.patchAlbumRaw(t, env.cookie, album, pinBody(member, older, third, asset))
			requireAlbumCoverNotMember(t, rec, 1)
		}
		// Two dead frames name two.
		rec := env.patchAlbumRaw(t, env.cookie, album, pinBody(member, hiddenMember, trashedMember, older))
		requireAlbumCoverNotMember(t, rec, 2)
	})

	t.Run("RejectsForeignCover", func(t *testing.T) {
		rec := env.patchAlbumRaw(t, env.cookie, album, pinBody(foreign, member, older, third))
		requireAlbumCoverNotMember(t, rec, 1)
	})

	t.Run("UnknownAlbumAnswersReadShape404", func(t *testing.T) {
		unknown := fixtureWallAlbumID("ff")
		reference := env.getAlbumsRaw(t, env.cookie, "/api/v1/albums/"+unknown.String())
		requireAlbumNotFound(t, reference)
		rec := env.patchAlbumRaw(t, env.cookie, unknown, `{"title":"No such album"}`)
		requireAlbumNotFound(t, rec)
		if rec.Body.String() != reference.Body.String() {
			t.Fatalf("patch 404 body = %q, want byte-identical to the read's %q",
				rec.Body.String(), reference.Body.String())
		}
	})

	t.Run("RefusesSyncAlbum", func(t *testing.T) {
		// The own sync album's read now answers 200 (the default
		// album's detail) -- the write face keeps its refusal
		// exactly as before: the album is system-managed, and the
		// patch folds into the single-shape 404, byte-identical to
		// the unknown id's, with nothing stored.
		ownSync := env.syncAlbumIDOf(t, adminID)
		reference := env.getAlbumsRaw(t, env.cookie, "/api/v1/albums/"+fixtureWallAlbumID("ff").String())
		requireAlbumNotFound(t, reference)
		rec := env.patchAlbumRaw(t, env.cookie, ownSync, `{"title":"Rename the stream"}`)
		requireAlbumNotFound(t, rec)
		if rec.Body.String() != reference.Body.String() {
			t.Fatalf("patch-own-sync 404 body = %q, want byte-identical to the unknown id's %q",
				rec.Body.String(), reference.Body.String())
		}
		if got := env.queryText(t, "SELECT title FROM albums WHERE id = $1", pgUUID(ownSync)); got != "Sync" {
			t.Errorf("stored sync title = %q, want the untouched %q", got, "Sync")
		}
	})
}

// TestUpdateAlbumDeletability pins the deletability switch's PATCH
// face: the owner flips it on a directory album (idempotently, the
// property patch's own convergence), the lock refuses it like every
// property field (a mixed patch carrying the unlock is refused
// whole -- unlock first, then edit), and a memory album answers
// the switch's own message key -- the setting is the directory
// album's; a memory album's deletions never move files.
func TestUpdateAlbumDeletability(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	adminID := env.sessionUserID(t)
	root := env.provisionMountRoot(t)

	directory := uuid.UUID(env.createAlbum(t, env.cookie, "Deletable source",
		sourceDir(t, root, "deletable"), nil).Id)
	memory := fixtureWriteAlbumID("77")
	env.insertWallAlbum(t, memory, adminID, "memory", "Legacy picks", wallAlbumOpts{})
	stored := func(t *testing.T, album uuid.UUID) int {
		t.Helper()
		return env.queryInt(t, "SELECT allow_app_deletion::int FROM albums WHERE id = $1", pgUUID(album))
	}

	t.Run("PatchesAndConverges", func(t *testing.T) {
		env.patchAlbum(t, env.cookie, directory, `{"allowAppDeletion":false}`)
		if got := stored(t, directory); got != 0 {
			t.Errorf("stored allow_app_deletion = %d, want 0 (switched off)", got)
		}
		// Idempotent: repeating the stored value converges.
		env.patchAlbum(t, env.cookie, directory, `{"allowAppDeletion":false}`)
		if got := stored(t, directory); got != 0 {
			t.Errorf("stored allow_app_deletion after the repeat = %d, want 0 (converged)", got)
		}
		env.patchAlbum(t, env.cookie, directory, `{"allowAppDeletion":true}`)
		if got := stored(t, directory); got != 1 {
			t.Errorf("stored allow_app_deletion after the flip back = %d, want 1", got)
		}
	})

	t.Run("MemoryAlbumRefusesTheSwitch", func(t *testing.T) {
		rec := env.patchAlbumRaw(t, env.cookie, memory, `{"allowAppDeletion":false}`)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("PATCH the memory album status = %d, want %d (body %q)",
				rec.Code, http.StatusBadRequest, rec.Body.String())
		}
		apiErr, _ := decodeError(t, rec)
		if apiErr.Code != apigen.VALIDATIONFAILED {
			t.Errorf("code = %q, want %q", apiErr.Code, apigen.VALIDATIONFAILED)
		}
		if apiErr.Details == nil {
			t.Fatal("details absent, want the album_deletable_not_directory message key")
		}
		if key, _ := (*apiErr.Details)["key"]; key != "album_deletable_not_directory" {
			t.Errorf("details.key = %v, want album_deletable_not_directory", key)
		}
		if got := stored(t, memory); got != 1 {
			t.Errorf("stored allow_app_deletion of the memory album = %d, want 1 (untouched)", got)
		}
	})
}

// TestUpdateAlbumAuthorization pins the two-tier answer of the write
// face: an album the caller cannot see answers the single-shape 404,
// an album shared to the caller answers the honest 403 (the share is
// view-only), and an administrator overrides the owner check.
func TestUpdateAlbumAuthorization(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	adminID := env.sessionUserID(t)
	graceID, graceCookie := env.createSecondUser(t, "patch-auth-grace@example.com")

	shared := fixtureWriteAlbumID("73")
	env.insertWallAlbum(t, shared, adminID, "memory", "Shared onward", wallAlbumOpts{})
	env.shareWallAlbum(t, shared, graceID, adminID, time.Date(2026, 11, 5, 9, 0, 0, 0, time.UTC))

	unshared := fixtureWriteAlbumID("74")
	env.insertWallAlbum(t, unshared, adminID, "memory", "Private", wallAlbumOpts{})

	graceOwn := fixtureWriteAlbumID("75")
	env.insertWallAlbum(t, graceOwn, graceID, "memory", "Grace's album", wallAlbumOpts{})

	t.Run("ShareeIsForbidden", func(t *testing.T) {
		rec := env.patchAlbumRaw(t, graceCookie, shared, `{"title":"Grace's edit"}`)
		requireAlbumWriteForbidden(t, rec)
		if got := env.queryText(t, "SELECT title FROM albums WHERE id = $1", pgUUID(shared)); got != "Shared onward" {
			t.Errorf("stored title = %q, want the untouched %q", got, "Shared onward")
		}
	})

	t.Run("StrangerIsNotFound", func(t *testing.T) {
		unknown := fixtureWallAlbumID("ff")
		reference := env.getAlbumsRaw(t, graceCookie, "/api/v1/albums/"+unknown.String())
		requireAlbumNotFound(t, reference)
		rec := env.patchAlbumRaw(t, graceCookie, unshared, `{"title":"Grace's edit"}`)
		requireAlbumNotFound(t, rec)
		if rec.Body.String() != reference.Body.String() {
			t.Fatalf("stranger 404 body = %q, want byte-identical to the unknown id's %q",
				rec.Body.String(), reference.Body.String())
		}
	})

	t.Run("AdminOverridesOwnerCheck", func(t *testing.T) {
		// The administrator holds no share of Grace's album and is
		// not its owner; the admin role bypasses the owner check.
		summary := env.patchAlbum(t, env.cookie, graceOwn, `{"title":"Admin's rename"}`)
		if summary.Title != "Admin's rename" {
			t.Errorf("title = %q, want the administrator's rename", summary.Title)
		}
		if got := env.queryText(t, "SELECT title FROM albums WHERE id = $1", pgUUID(graceOwn)); got != "Admin's rename" {
			t.Errorf("stored title = %q, want the administrator's rename", got)
		}
	})
}

// TestAlbumLockRetired pins the lock field's retirement at the
// patch face: the contract no longer carries a `locked` field
// anywhere, so a patch of only the dead field answers the
// no-recognized-field 400 (the unknown key decodes as absent), a
// mixed patch lands its LIVE fields and ignores the dead one, and
// neither the card nor the detail JSON carries the key at all --
// the shape-level proof that the freeze is gone whole.
func TestAlbumLockRetired(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	adminID := env.sessionUserID(t)

	album := fixtureWriteAlbumID("76")
	env.insertWallAlbum(t, album, adminID, "memory", "Frozen", wallAlbumOpts{})
	member := fixtureWallAssetID("76")
	env.insertTimelineAsset(t, member, adminID, time.Date(2026, 8, 2, 12, 0, 0, 0, time.UTC), timelineAssetOpts{})
	env.addTimelineItem(t, album, member, adminID)

	t.Run("DeadFieldAloneIsAnEmptyPatch", func(t *testing.T) {
		for _, body := range []string{`{"locked":true}`, `{"locked":false}`} {
			rec := env.patchAlbumRaw(t, env.cookie, album, body)
			requireValidationFields(t, rec, "body")
		}
		if got := env.queryText(t, "SELECT title FROM albums WHERE id = $1", pgUUID(album)); got != "Frozen" {
			t.Errorf("stored title = %q, want the untouched %q", got, "Frozen")
		}
	})

	t.Run("MixedPatchLandsTheLiveFields", func(t *testing.T) {
		summary := env.patchAlbum(t, env.cookie, album, `{"title":"Thawed","locked":true}`)
		if summary.Title != "Thawed" {
			t.Errorf("title = %q, want the live field's patch", summary.Title)
		}
	})

	t.Run("TheShapeCarriesNoLockKey", func(t *testing.T) {
		var card map[string]any
		rec := env.patchAlbumRaw(t, env.cookie, album, `{"title":"Keyless"}`)
		if err := json.Unmarshal(rec.Body.Bytes(), &card); err != nil {
			t.Fatalf("decode the card: %v", err)
		}
		if _, present := card["locked"]; present {
			t.Error("the wall card still carries the retired `locked` key")
		}
		detail := env.getAlbumsRaw(t, env.cookie, "/api/v1/albums/"+album.String())
		var page map[string]any
		if err := json.Unmarshal(detail.Body.Bytes(), &page); err != nil {
			t.Fatalf("decode the detail: %v", err)
		}
		if _, present := page["locked"]; present {
			t.Error("the album detail still carries the retired `locked` key")
		}
	})
}

// TestMemoryTypeRetired pins the type rename whole: the retired enum
// value 'virtual' refuses every write at the database's own enum
// guard, and rows of the live value surface as `memory` on every
// face. The word leaves the product vocabulary with the value.
func TestMemoryTypeRetired(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	adminID := env.sessionUserID(t)

	t.Run("RetiredValueRefusesDirectInserts", func(t *testing.T) {
		if _, err := env.pool.Exec(context.Background(),
			"INSERT INTO albums (id, owner_id, title, type) VALUES ($1, $2, 'Ghost', 'virtual')",
			pgUUID(fixtureWriteAlbumID("e1")), pgUUID(adminID)); err == nil {
			t.Fatal("an album row of the retired type value 'virtual' inserted, want the enum's refusal")
		}
		if got := env.queryInt(t, "SELECT count(*) FROM albums WHERE type::text = 'virtual'"); got != 0 {
			t.Errorf("rows of the retired type value = %d, want 0", got)
		}
	})

	t.Run("RenamedRowsCarryTheMemoryType", func(t *testing.T) {
		memory := fixtureWriteAlbumID("e2")
		env.insertWallAlbum(t, memory, adminID, "memory", "The kept set", wallAlbumOpts{})
		wall := env.getAlbumWall(t, env.cookie)
		found := false
		for _, card := range wall.Albums {
			if uuid.UUID(card.Id) == memory {
				found = true
				if card.Type != apigen.AlbumSummaryTypeMemory {
					t.Errorf("wall card type = %q, want memory", card.Type)
				}
			}
		}
		if !found {
			t.Fatalf("the memory album is missing from the wall (%v)", wall.Albums)
		}
		detail := env.getAlbumDetail(t, env.cookie, memory, "")
		if detail.Type != apigen.AlbumDetailTypeMemory {
			t.Errorf("detail type = %q, want memory", detail.Type)
		}
	})
}

// TestDeleteAlbum pins the delete's memory arm: the row, the
// membership and the share rows cascade together, the members'
// asset rows survive, and the two-tier authorization answer holds.
// The directory arm is the un-ingest removal -- albumremoval_test.go
// owns its pins.
func TestDeleteAlbum(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	adminID := env.sessionUserID(t)
	graceID, graceCookie := env.createSecondUser(t, "delete-grace@example.com")

	memory := fixtureWriteAlbumID("81")
	env.insertWallAlbum(t, memory, adminID, "memory", "Going away", wallAlbumOpts{})
	member := fixtureWallAssetID("81")
	second := fixtureWallAssetID("82")
	env.insertTimelineAsset(t, member, adminID, time.Date(2026, 8, 3, 12, 0, 0, 0, time.UTC), timelineAssetOpts{})
	env.insertTimelineAsset(t, second, adminID, time.Date(2026, 8, 3, 13, 0, 0, 0, time.UTC), timelineAssetOpts{})
	env.addTimelineItem(t, memory, member, adminID)
	env.addTimelineItem(t, memory, second, adminID)
	env.shareWallAlbum(t, memory, graceID, adminID, time.Date(2026, 11, 5, 9, 0, 0, 0, time.UTC))

	sharedToGrace := fixtureWriteAlbumID("84")
	env.insertWallAlbum(t, sharedToGrace, adminID, "memory", "Shared", wallAlbumOpts{})
	env.shareWallAlbum(t, sharedToGrace, graceID, adminID, time.Date(2026, 11, 5, 9, 0, 0, 0, time.UTC))
	private := fixtureWriteAlbumID("85")
	env.insertWallAlbum(t, private, adminID, "memory", "Private", wallAlbumOpts{})

	t.Run("DeletesMemoryAndCascades", func(t *testing.T) {
		// The birth row rides the cascade too: a created memory
		// carries the creator's timeline-exclusion preference from
		// its POST, and the delete takes it with the row.
		env.setTimelinePref(t, graceID, memory, false)
		rec := env.deleteAlbumRaw(t, env.cookie, memory)
		if rec.Code != http.StatusNoContent {
			t.Fatalf("DELETE /albums/%s status = %d, want %d (body %q)",
				memory, rec.Code, http.StatusNoContent, rec.Body.String())
		}
		if got := env.queryInt(t, "SELECT count(*) FROM albums WHERE id = $1", pgUUID(memory)); got != 0 {
			t.Errorf("album rows = %d, want 0", got)
		}
		// The membership, share and preference rows cascade with the
		// row.
		if got := env.queryInt(t, "SELECT count(*) FROM album_items WHERE album_id = $1", pgUUID(memory)); got != 0 {
			t.Errorf("membership rows = %d, want 0 (cascaded)", got)
		}
		if got := env.queryInt(t, "SELECT count(*) FROM album_shares WHERE album_id = $1", pgUUID(memory)); got != 0 {
			t.Errorf("share rows = %d, want 0 (cascaded)", got)
		}
		if got := env.queryInt(t, "SELECT count(*) FROM user_album_prefs WHERE album_id = $1", pgUUID(memory)); got != 0 {
			t.Errorf("preference rows = %d, want 0 (cascaded)", got)
		}
		// The assets themselves survive: deleting an album never
		// deletes photos.
		if got := env.queryInt(t, "SELECT count(*) FROM assets WHERE id = ANY($1)",
			pgUUIDs([]uuid.UUID{member, second})); got != 2 {
			t.Errorf("member assets = %d, want 2 (untouched)", got)
		}

		wall := env.getAlbumWall(t, env.cookie)
		for _, card := range wall.Albums {
			if uuid.UUID(card.Id) == memory {
				t.Fatalf("the deleted album is still on the wall (%v)", wall.Albums)
			}
		}
		rec = env.getAlbumsRaw(t, env.cookie, "/api/v1/albums/"+memory.String())
		requireAlbumNotFound(t, rec)
	})

	t.Run("SecondDeleteIsNotFound", func(t *testing.T) {
		rec := env.deleteAlbumRaw(t, env.cookie, memory)
		requireAlbumNotFound(t, rec)
	})

	t.Run("RefusesSyncAlbum", func(t *testing.T) {
		ownSync := env.syncAlbumIDOf(t, adminID)
		rec := env.deleteAlbumRaw(t, env.cookie, ownSync)
		requireAlbumNotFound(t, rec)
	})

	t.Run("ShareeIsForbidden", func(t *testing.T) {
		rec := env.deleteAlbumRaw(t, graceCookie, sharedToGrace)
		requireAlbumWriteForbidden(t, rec)
		if got := env.queryInt(t, "SELECT count(*) FROM albums WHERE id = $1", pgUUID(sharedToGrace)); got != 1 {
			t.Errorf("shared album rows = %d, want 1 (the sharee's delete landed nothing)", got)
		}
	})

	t.Run("StrangerIsNotFound", func(t *testing.T) {
		rec := env.deleteAlbumRaw(t, graceCookie, private)
		requireAlbumNotFound(t, rec)
	})

	t.Run("UnknownAlbum", func(t *testing.T) {
		rec := env.deleteAlbumRaw(t, env.cookie, fixtureWallAlbumID("ff"))
		requireAlbumNotFound(t, rec)
	})
}

// TestRemoveAlbumItems pins the membership remove: the rows that
// exist leave, absent ids skip, the trashed member's row leaves too
// (no live predicate on the removal), and the authorization and type
// gates hold.
func TestRemoveAlbumItems(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	adminID := env.sessionUserID(t)
	graceID, graceCookie := env.createSecondUser(t, "remove-items-grace@example.com")

	album := fixtureWriteAlbumID("a1")
	env.insertWallAlbum(t, album, adminID, "memory", "Curation", wallAlbumOpts{})
	sharedToGrace := fixtureWriteAlbumID("a3")
	env.insertWallAlbum(t, sharedToGrace, adminID, "memory", "Shared", wallAlbumOpts{})
	env.shareWallAlbum(t, sharedToGrace, graceID, adminID, time.Date(2026, 11, 5, 9, 0, 0, 0, time.UTC))
	directory := fixtureWriteAlbumID("a4")
	env.insertWallAlbum(t, directory, adminID, "directory", "Archive", wallAlbumOpts{})

	base := time.Date(2026, 8, 5, 12, 0, 0, 0, time.UTC)
	first := fixtureWallAssetID("a1")
	second := fixtureWallAssetID("a2")
	trashedMember := fixtureWallAssetID("a3")
	absent := fixtureWallAssetID("a4")
	env.insertTimelineAsset(t, first, adminID, base, timelineAssetOpts{})
	env.insertTimelineAsset(t, second, adminID, base.Add(time.Hour), timelineAssetOpts{})
	env.insertTimelineAsset(t, trashedMember, adminID, base.Add(2*time.Hour), timelineAssetOpts{deleted: true})
	env.insertTimelineAsset(t, absent, adminID, base.Add(3*time.Hour), timelineAssetOpts{})
	for _, asset := range []uuid.UUID{first, second, trashedMember} {
		env.addTimelineItem(t, album, asset, adminID)
	}
	t.Run("RemovesPresentAndSkipsAbsent", func(t *testing.T) {
		report := decodeRemoveItemsReport(t, env.deleteAlbumItems(t, env.cookie, album, albumItemsBody(first, absent)))
		if report.Removed != 1 || report.Skipped != 1 {
			t.Errorf("removed/skipped = %d/%d, want 1/1", report.Removed, report.Skipped)
		}
		if got := env.queryInt(t, "SELECT count(*) FROM album_items WHERE album_id = $1", pgUUID(album)); got != 2 {
			t.Errorf("membership rows = %d, want 2 (one row left)", got)
		}
	})

	t.Run("RemovesTrashedMemberRow", func(t *testing.T) {
		report := decodeRemoveItemsReport(t, env.deleteAlbumItems(t, env.cookie, album, albumItemsBody(trashedMember)))
		if report.Removed != 1 || report.Skipped != 0 {
			t.Errorf("removed/skipped = %d/%d, want 1/0 (the removal carries no live predicate)",
				report.Removed, report.Skipped)
		}
	})

	t.Run("AllAbsentStillAnswers", func(t *testing.T) {
		report := decodeRemoveItemsReport(t, env.deleteAlbumItems(t, env.cookie, album, albumItemsBody(absent)))
		if report.Removed != 0 || report.Skipped != 1 {
			t.Errorf("removed/skipped = %d/%d, want 0/1 (still a 200)", report.Removed, report.Skipped)
		}
	})

	t.Run("CollapsesRepeatedIds", func(t *testing.T) {
		report := decodeRemoveItemsReport(t, env.deleteAlbumItems(t, env.cookie, album, albumItemsBody(second, second)))
		if report.Removed != 1 || report.Skipped != 0 {
			t.Errorf("removed/skipped = %d/%d, want 1/0 (repeated ids collapse)", report.Removed, report.Skipped)
		}
		if got := env.queryInt(t, "SELECT count(*) FROM album_items WHERE album_id = $1", pgUUID(album)); got != 0 {
			t.Errorf("membership rows = %d, want 0 (the album is empty now)", got)
		}
	})

	t.Run("ValidatesListBounds", func(t *testing.T) {
		rec := env.deleteAlbumItems(t, env.cookie, album, `{"assetIds":[]}`)
		requireValidationFields(t, rec, "assetIds")
	})

	t.Run("ShareeIsForbidden", func(t *testing.T) {
		rec := env.deleteAlbumItems(t, graceCookie, sharedToGrace, albumItemsBody(first))
		requireAlbumWriteForbidden(t, rec)
	})

	t.Run("RefusesDirectoryAlbum", func(t *testing.T) {
		rec := env.deleteAlbumItems(t, env.cookie, directory, albumItemsBody(first))
		requireAlbumNotFound(t, rec)
	})

	t.Run("RefusesSyncAlbum", func(t *testing.T) {
		ownSync := env.syncAlbumIDOf(t, adminID)
		rec := env.deleteAlbumItems(t, env.cookie, ownSync, albumItemsBody(first))
		requireAlbumNotFound(t, rec)
	})

	t.Run("UnknownAlbum", func(t *testing.T) {
		rec := env.deleteAlbumItems(t, env.cookie, fixtureWallAlbumID("ff"), albumItemsBody(first))
		requireAlbumNotFound(t, rec)
	})
}

// TestAlbumWriteEvents pins the live event names of the write
// surface: creation, patch, membership change and delete, each with
// the ids-only payload -- the album id and the acting user's id, and
// nothing else. No album state (title, cover, lock,
// counts, membership) ever rides the shared bus.
func TestAlbumWriteEvents(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	adminID := env.sessionUserID(t)
	root := env.provisionMountRoot(t)

	// The creation through the API: one album.created event. The
	// lifecycle writes below ride a fixture memory album (whose
	// plain cascade publishes exactly the one album.deleted).
	created := env.createAlbum(t, env.cookie, "The eventful album", root, nil)
	directoryID := uuid.UUID(created.Id)

	album := fixtureWriteAlbumID("b0")
	env.insertWallAlbum(t, album, adminID, "memory", "The eventful album", wallAlbumOpts{})
	env.patchAlbum(t, env.cookie, album, `{"title":"The renamed album"}`)

	member := fixtureWallAssetID("b1")
	second := fixtureWallAssetID("b2")
	base := time.Date(2026, 8, 6, 12, 0, 0, 0, time.UTC)
	env.insertTimelineAsset(t, member, adminID, base, timelineAssetOpts{})
	env.insertTimelineAsset(t, second, adminID, base.Add(time.Hour), timelineAssetOpts{})
	env.addTimelineItem(t, album, member, adminID)
	env.addTimelineItem(t, album, second, adminID)

	removeReport := decodeRemoveItemsReport(t, env.deleteAlbumItems(t, env.cookie, album, albumItemsBody(member)))
	if removeReport.Removed != 1 {
		t.Fatalf("the remove report = %+v, want 1 removed", removeReport)
	}
	rec := env.deleteAlbumRaw(t, env.cookie, album)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("DELETE /albums/%s status = %d, want %d", album, rec.Code, http.StatusNoContent)
	}

	cases := []struct {
		name  string
		count int
	}{
		{"album.created", 1},
		{"album.updated", 1}, // the single patch, unconditional on success
		{"album.items-changed", 1},
		{"album.deleted", 1},
	}
	for _, want := range cases {
		payloads := busEventsOf(t, env.bus, want.name)
		if len(payloads) != want.count {
			t.Fatalf("%s events = %d, want exactly %d (payloads %v)",
				want.name, len(payloads), want.count, payloads)
		}
		// The created event names the created directory album, the
		// lifecycle events the fixture memory album.
		wantAlbum := album
		if want.name == "album.created" {
			wantAlbum = directoryID
		}
		for _, payload := range payloads {
			if payload["albumId"] != wantAlbum.String() {
				t.Errorf("%s albumId = %q, want %s", want.name, payload["albumId"], wantAlbum)
			}
			if payload["userId"] != adminID.String() {
				t.Errorf("%s userId = %q, want the acting caller %s", want.name, payload["userId"], adminID)
			}
			// ids only: exactly the two keys, no album state.
			if len(payload) != 2 {
				t.Errorf("%s payload = %v, want exactly the albumId/userId pair", want.name, payload)
			}
			for _, banned := range []string{
				"title", "type", "coverAssetIds",
				"itemCount", "assetIds", "added", "removed", "skipped",
			} {
				if payload[banned] != "" {
					t.Errorf("%s payload carries %q = %q -- album state must not ride the shared bus",
						want.name, banned, payload[banned])
				}
			}
		}
	}
}
