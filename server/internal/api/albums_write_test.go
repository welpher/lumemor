// Album surface write tests (M3c Task 2): virtual album creation,
// the property patch (with the cover's live-member validation and the
// locked gate), the delete (virtual only, cascades, never locked-
// gated) and the membership curation with the batch's honest
// {added|removed, skipped} report. The two-tier authorization answer
// runs through every write: an invisible album answers the reads'
// byte-identical 404, an album shared to the caller answers the
// honest 403, and an administrator overrides the owner check. The
// four live album.* events carry ids only -- no album state rides
// the shared stream. Fixtures insert albums, memberships and assets
// directly in the database; every write goes through the API.
package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	apigen "github.com/welpher/lumemor/server/internal/api/gen"
)

// albumIDBody builds a list of quoted ids for the membership bodies.
func albumIDList(ids []uuid.UUID) string {
	quoted := make([]string, len(ids))
	for i, id := range ids {
		quoted[i] = `"` + id.String() + `"`
	}
	return strings.Join(quoted, ",")
}

// albumItemsBody assembles the membership write body.
func albumItemsBody(ids ...uuid.UUID) string {
	return `{"assetIds":[` + albumIDList(ids) + `]}`
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

// createAlbumBody assembles the creation body; a nil description
// omits the field.
func createAlbumBody(t *testing.T, title string, description *string) string {
	t.Helper()
	quotedTitle, err := json.Marshal(title)
	if err != nil {
		t.Fatalf("quote title %q: %v", title, err)
	}
	body := `{"title":` + string(quotedTitle)
	if description != nil {
		quotedDescription, err := json.Marshal(*description)
		if err != nil {
			t.Fatalf("quote description %q: %v", *description, err)
		}
		body += `,"description":` + string(quotedDescription)
	}
	return body + "}"
}

// createAlbum fires POST /albums and decodes the 201 body.
func (e *assetEnv) createAlbum(t *testing.T, cookie *http.Cookie, title string, description *string) apigen.AlbumSummary {
	t.Helper()
	rec := e.writeAlbumRaw(t, cookie, http.MethodPost, "/api/v1/albums", createAlbumBody(t, title, description))
	if rec.Code != http.StatusCreated {
		t.Fatalf("POST /albums status = %d, want %d (body %q)",
			rec.Code, http.StatusCreated, rec.Body.String())
	}
	var summary apigen.AlbumSummary
	if err := json.Unmarshal(rec.Body.Bytes(), &summary); err != nil {
		t.Fatalf("decode create album body %q: %v", rec.Body.String(), err)
	}
	return summary
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

// postAlbumItems fires POST /albums/{albumId}/items with the raw body.
func (e *assetEnv) postAlbumItems(t *testing.T, cookie *http.Cookie, albumID uuid.UUID, body string) *httptest.ResponseRecorder {
	t.Helper()
	return e.writeAlbumRaw(t, cookie, http.MethodPost,
		"/api/v1/albums/"+albumID.String()+"/items", body)
}

// deleteAlbumItems fires DELETE /albums/{albumId}/items with the raw
// body.
func (e *assetEnv) deleteAlbumItems(t *testing.T, cookie *http.Cookie, albumID uuid.UUID, body string) *httptest.ResponseRecorder {
	t.Helper()
	return e.writeAlbumRaw(t, cookie, http.MethodDelete,
		"/api/v1/albums/"+albumID.String()+"/items", body)
}

// decodeAddItemsReport decodes the 200 add report.
func decodeAddItemsReport(t *testing.T, rec *httptest.ResponseRecorder) apigen.AddAlbumItemsResponse {
	t.Helper()
	if rec.Code != http.StatusOK {
		t.Fatalf("POST /albums/{id}/items status = %d, want %d (body %q)",
			rec.Code, http.StatusOK, rec.Body.String())
	}
	var report apigen.AddAlbumItemsResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &report); err != nil {
		t.Fatalf("decode add items body %q: %v", rec.Body.String(), err)
	}
	return report
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

// requireAlbumLocked asserts the 409 of the locked gate.
func requireAlbumLocked(t *testing.T, rec *httptest.ResponseRecorder) {
	t.Helper()
	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d (body %q)",
			rec.Code, http.StatusConflict, rec.Body.String())
	}
	apiErr, _ := decodeError(t, rec)
	if apiErr.Code != apigen.ALBUMLOCKED {
		t.Errorf("code = %q, want %q", apiErr.Code, apigen.ALBUMLOCKED)
	}
}

// requireAlbumCoverNotMember asserts the 400 VALIDATION_FAILED whose
// details carry the album_cover_not_member message key.
func requireAlbumCoverNotMember(t *testing.T, rec *httptest.ResponseRecorder) {
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
// creation, the patch, the delete and both membership writes refuse
// the unauthenticated caller before any handler logic runs.
func TestAlbumWritesRequireSession(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	albumID := fixtureWallAlbumID("ff").String()

	for _, target := range []struct {
		method string
		target string
		body   string
	}{
		{http.MethodPost, "/api/v1/albums", `{"title":"No session"}`},
		{http.MethodPatch, "/api/v1/albums/" + albumID, `{"title":"No session"}`},
		{http.MethodDelete, "/api/v1/albums/" + albumID, ""},
		{http.MethodPost, "/api/v1/albums/" + albumID + "/items", `{"assetIds":["` + fixtureWallAssetID("ff").String() + `"]}`},
		{http.MethodDelete, "/api/v1/albums/" + albumID + "/items", `{"assetIds":["` + fixtureWallAssetID("ff").String() + `"]}`},
	} {
		rec := env.writeAlbumRaw(t, nil, target.method, target.target, target.body)
		requireUnauthorized(t, rec)
	}
}

// TestCreateAlbum pins the creation: the ONLY creatable type is the
// virtual album, the answer is the wall card of the fresh row (owner
// role, no share, no cover, no members), and the wall lists it. The
// title validation spans the trim, the rune count and the null body.
func TestCreateAlbum(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	adminID := env.sessionUserID(t)

	t.Run("CreatesAndListsOnWall", func(t *testing.T) {
		description := "The coastal week in the van."
		summary := env.createAlbum(t, env.cookie, "Road trip 2026", &description)

		if summary.Title != "Road trip 2026" || summary.Description != description {
			t.Errorf("title/description = %q/%q, want the stored values", summary.Title, summary.Description)
		}
		if summary.Type != apigen.AlbumSummaryTypeVirtual {
			t.Errorf("type = %q, want virtual (the only creatable type)", summary.Type)
		}
		if summary.Locked {
			t.Error("locked = true, want false")
		}
		if summary.IsShared {
			t.Error("isShared = true, want false (nothing is shared yet)")
		}
		if summary.CoverAssetIds == nil || len(summary.CoverAssetIds) != 0 {
			t.Errorf("coverAssetIds = %v, want an empty array (no members)", summary.CoverAssetIds)
		}
		if summary.ItemCount != 0 {
			t.Errorf("itemCount = %d, want 0", summary.ItemCount)
		}
		if summary.CallerRole != apigen.AlbumSummaryCallerRoleOwner {
			t.Errorf("callerRole = %q, want owner", summary.CallerRole)
		}
		if summary.Owner != nil {
			t.Errorf("owner = %+v, want absent on an own card", summary.Owner)
		}
		if summary.UpdatedAt.IsZero() {
			t.Error("updatedAt = zero, want the creation time")
		}
		if got := env.queryInt(t, "SELECT count(*) FROM albums WHERE id = $1 AND type = 'virtual' AND owner_id = $2",
			pgUUID(uuid.UUID(summary.Id)), pgUUID(adminID)); got != 1 {
			t.Errorf("stored album rows = %d, want 1", got)
		}

		// The wall lists exactly the new card (the caller's sync
		// album never counts).
		wall := env.getAlbumWall(t, env.cookie)
		if len(wall.Albums) != 1 || uuid.UUID(wall.Albums[0].Id) != uuid.UUID(summary.Id) {
			t.Fatalf("wall = %v, want exactly the created album", wall.Albums)
		}
		detail := env.getAlbumDetail(t, env.cookie, uuid.UUID(summary.Id), "")
		if detail.Title != "Road trip 2026" || detail.ItemCount != 0 {
			t.Errorf("detail title/itemCount = %q/%d, want the stored title and 0", detail.Title, detail.ItemCount)
		}
	})

	t.Run("TrimsTitle", func(t *testing.T) {
		summary := env.createAlbum(t, env.cookie, "  Weekend picks  ", nil)
		if summary.Title != "Weekend picks" {
			t.Errorf("title = %q, want the trimmed %q", summary.Title, "Weekend picks")
		}
	})

	t.Run("DescriptionDefaultsToEmpty", func(t *testing.T) {
		summary := env.createAlbum(t, env.cookie, "No description", nil)
		if summary.Description != "" {
			t.Errorf("description = %q, want the empty default", summary.Description)
		}
	})

	t.Run("ValidatesTitle", func(t *testing.T) {
		for _, title := range []string{
			"",                      // missing entirely
			"   ",                   // whitespace only after the trim
			strings.Repeat("a", 65), // past the bound
		} {
			rec := env.writeAlbumRaw(t, env.cookie, http.MethodPost, "/api/v1/albums",
				createAlbumBody(t, title, nil))
			requireValidationFields(t, rec, "title")
		}

		// The boundary itself answers, and runes (not bytes) are
		// the count: 64 ascii and a short CJK title both land.
		env.createAlbum(t, env.cookie, strings.Repeat("a", 64), nil)
		env.createAlbum(t, env.cookie, "家庭相册", nil)
	})
}

// TestUpdateAlbum pins the property patch: every field lands, the
// cover validation refuses every id that is not a live member of the
// album itself, and an unknown album answers the reads'
// single-shape 404.
func TestUpdateAlbum(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	adminID := env.sessionUserID(t)

	album := fixtureWriteAlbumID("71")
	env.insertWallAlbum(t, album, adminID, "virtual", "Original title", wallAlbumOpts{
		description: "Original description.",
	})

	// Two live members, three non-live members, one live non-member
	// of the caller's own, and one live asset of another account.
	member := fixtureWallAssetID("71")
	older := fixtureWallAssetID("72")
	hiddenMember := fixtureWallAssetID("73")
	trashedMember := fixtureWallAssetID("74")
	componentMember := fixtureWallAssetID("75")
	nonMember := fixtureWallAssetID("76")
	base := time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)
	env.insertTimelineAsset(t, member, adminID, base, timelineAssetOpts{})
	env.insertTimelineAsset(t, older, adminID, base.Add(-time.Hour), timelineAssetOpts{})
	env.insertTimelineAsset(t, hiddenMember, adminID, base.Add(time.Hour), timelineAssetOpts{isHidden: true})
	env.insertTimelineAsset(t, trashedMember, adminID, base.Add(time.Hour), timelineAssetOpts{deleted: true})
	env.insertTimelineAsset(t, componentMember, adminID, base.Add(time.Hour), timelineAssetOpts{isComponent: true})
	env.insertTimelineAsset(t, nonMember, adminID, base.Add(time.Hour), timelineAssetOpts{})
	graceID, _ := env.createSecondUser(t, "patch-cover-grace@example.com")
	foreign := fixtureWallAssetID("77")
	env.insertTimelineAsset(t, foreign, graceID, base.Add(time.Hour), timelineAssetOpts{})
	for _, asset := range []uuid.UUID{member, older, hiddenMember, trashedMember, componentMember} {
		env.addTimelineItem(t, album, asset, adminID)
	}

	t.Run("PatchesEachField", func(t *testing.T) {
		summary := env.patchAlbum(t, env.cookie, album, `{"title":"Renamed"}`)
		if summary.Title != "Renamed" || summary.Description != "Original description." {
			t.Errorf("title/description = %q/%q, want the patched title over the kept description",
				summary.Title, summary.Description)
		}

		summary = env.patchAlbum(t, env.cookie, album, `{"description":"A new description."}`)
		if summary.Description != "A new description." || summary.Title != "Renamed" {
			t.Errorf("title/description = %q/%q, want the patched description over the kept title",
				summary.Title, summary.Description)
		}

		cover := `{"coverAssetId":"` + older.String() + `"}`
		summary = env.patchAlbum(t, env.cookie, album, cover)
		// The pinned cover rides the summary as the single-element
		// collage, and the detail reports the raw pin.
		if len(summary.CoverAssetIds) != 1 || uuid.UUID(summary.CoverAssetIds[0]) != older {
			t.Errorf("summary coverAssetIds = %v, want [%s] (the pin)", summary.CoverAssetIds, older)
		}
		detail := env.getAlbumDetail(t, env.cookie, album, "")
		if detail.CoverAssetId == nil || uuid.UUID(*detail.CoverAssetId) != older {
			t.Errorf("detail coverAssetId = %v, want the pinned %s", detail.CoverAssetId, older)
		}

		summary = env.patchAlbum(t, env.cookie, album, `{"locked":true}`)
		if !summary.Locked {
			t.Error("locked = false, want true after the patch")
		}
		// The unlock: locked itself is never gated.
		summary = env.patchAlbum(t, env.cookie, album, `{"locked":false}`)
		if summary.Locked {
			t.Error("locked = true, want false after the unlock patch")
		}

		// The row itself carries the last patch's values.
		if got := env.queryText(t, "SELECT title FROM albums WHERE id = $1", pgUUID(album)); got != "Renamed" {
			t.Errorf("stored title = %q, want Renamed", got)
		}
		if got := env.queryText(t, "SELECT description FROM albums WHERE id = $1", pgUUID(album)); got != "A new description." {
			t.Errorf("stored description = %q, want the patched one", got)
		}
		if got := env.queryInt(t, "SELECT locked::int FROM albums WHERE id = $1", pgUUID(album)); got != 0 {
			t.Errorf("stored locked = %d, want 0 (unlocked again)", got)
		}
	})

	t.Run("PatchesMultipleFieldsAtOnce", func(t *testing.T) {
		body := `{"title":"Combined","description":"One write.","locked":true}`
		summary := env.patchAlbum(t, env.cookie, album, body)
		if summary.Title != "Combined" || summary.Description != "One write." || !summary.Locked {
			t.Errorf("title/description/locked = %q/%q/%v, want all three landed",
				summary.Title, summary.Description, summary.Locked)
		}
		env.patchAlbum(t, env.cookie, album, `{"locked":false}`) // leave it unlocked
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
		body := `{"coverAssetId":"` + nonMember.String() + `"}`
		rec := env.patchAlbumRaw(t, env.cookie, album, body)
		requireAlbumCoverNotMember(t, rec)
	})

	t.Run("RejectsCoverOfNonLiveMember", func(t *testing.T) {
		// A hidden, trashed or component member row is a member no
		// more: the cover must be a LIVE member.
		for _, asset := range []uuid.UUID{hiddenMember, trashedMember, componentMember} {
			body := `{"coverAssetId":"` + asset.String() + `"}`
			rec := env.patchAlbumRaw(t, env.cookie, album, body)
			requireAlbumCoverNotMember(t, rec)
		}
	})

	t.Run("RejectsForeignCover", func(t *testing.T) {
		body := `{"coverAssetId":"` + foreign.String() + `"}`
		rec := env.patchAlbumRaw(t, env.cookie, album, body)
		requireAlbumCoverNotMember(t, rec)
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
	env.insertWallAlbum(t, shared, adminID, "virtual", "Shared onward", wallAlbumOpts{})
	env.shareWallAlbum(t, shared, graceID, adminID, time.Date(2026, 11, 5, 9, 0, 0, 0, time.UTC))

	unshared := fixtureWriteAlbumID("74")
	env.insertWallAlbum(t, unshared, adminID, "virtual", "Private", wallAlbumOpts{})

	graceOwn := fixtureWriteAlbumID("75")
	env.insertWallAlbum(t, graceOwn, graceID, "virtual", "Grace's album", wallAlbumOpts{})

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

// TestUpdateAlbumLocked pins the locked gate: the property fields are
// refused with 409 ALBUM_LOCKED (a mixed patch that carries the
// unlock is refused too -- unlock first), while a patch of locked
// alone IS the unlock and always answers.
func TestUpdateAlbumLocked(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	adminID := env.sessionUserID(t)

	album := fixtureWriteAlbumID("76")
	env.insertWallAlbum(t, album, adminID, "virtual", "Frozen", wallAlbumOpts{locked: true})
	member := fixtureWallAssetID("76")
	env.insertTimelineAsset(t, member, adminID, time.Date(2026, 8, 2, 12, 0, 0, 0, time.UTC), timelineAssetOpts{})
	env.addTimelineItem(t, album, member, adminID)

	t.Run("RefusesPropertyFields", func(t *testing.T) {
		for _, body := range []string{
			`{"title":"Nope"}`,
			`{"description":"Nope"}`,
			`{"coverAssetId":"` + member.String() + `"}`,
		} {
			rec := env.patchAlbumRaw(t, env.cookie, album, body)
			requireAlbumLocked(t, rec)
		}
		if got := env.queryText(t, "SELECT title FROM albums WHERE id = $1", pgUUID(album)); got != "Frozen" {
			t.Errorf("stored title = %q, want the untouched %q", got, "Frozen")
		}
	})

	t.Run("RefusesMixedPatchCarryingTheUnlock", func(t *testing.T) {
		// The unlock must land alone: a patch that also edits a
		// property is refused whole -- unlock first.
		rec := env.patchAlbumRaw(t, env.cookie, album, `{"title":"Nope","locked":false}`)
		requireAlbumLocked(t, rec)
		if got := env.queryInt(t, "SELECT locked::int FROM albums WHERE id = $1", pgUUID(album)); got != 1 {
			t.Errorf("stored locked = %d, want still 1 (the mixed patch landed nothing)", got)
		}
	})

	t.Run("UnlockOnlyPatchIsAllowed", func(t *testing.T) {
		summary := env.patchAlbum(t, env.cookie, album, `{"locked":false}`)
		if summary.Locked {
			t.Error("locked = true, want false after the unlock")
		}
		if got := env.queryInt(t, "SELECT locked::int FROM albums WHERE id = $1", pgUUID(album)); got != 0 {
			t.Errorf("stored locked = %d, want 0", got)
		}
		// Unlocked, the property fields answer again.
		summary = env.patchAlbum(t, env.cookie, album, `{"title":"Thawed"}`)
		if summary.Title != "Thawed" {
			t.Errorf("title = %q, want the post-unlock edit", summary.Title)
		}
	})
}

// TestDeleteAlbum pins the delete: virtual albums only (directory
// and sync ids answer the single-shape 404), the membership and
// share rows cascade with the row, the locked gate never applies,
// and the two-tier authorization answer holds.
func TestDeleteAlbum(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	adminID := env.sessionUserID(t)
	graceID, graceCookie := env.createSecondUser(t, "delete-grace@example.com")

	virtual := fixtureWriteAlbumID("81")
	env.insertWallAlbum(t, virtual, adminID, "virtual", "Going away", wallAlbumOpts{})
	member := fixtureWallAssetID("81")
	second := fixtureWallAssetID("82")
	env.insertTimelineAsset(t, member, adminID, time.Date(2026, 8, 3, 12, 0, 0, 0, time.UTC), timelineAssetOpts{})
	env.insertTimelineAsset(t, second, adminID, time.Date(2026, 8, 3, 13, 0, 0, 0, time.UTC), timelineAssetOpts{})
	env.addTimelineItem(t, virtual, member, adminID)
	env.addTimelineItem(t, virtual, second, adminID)
	env.shareWallAlbum(t, virtual, graceID, adminID, time.Date(2026, 11, 5, 9, 0, 0, 0, time.UTC))

	locked := fixtureWriteAlbumID("82")
	env.insertWallAlbum(t, locked, adminID, "virtual", "Frozen going away", wallAlbumOpts{locked: true})
	directory := fixtureWriteAlbumID("83")
	env.insertWallAlbum(t, directory, adminID, "directory", "Archive", wallAlbumOpts{})
	sharedToGrace := fixtureWriteAlbumID("84")
	env.insertWallAlbum(t, sharedToGrace, adminID, "virtual", "Shared", wallAlbumOpts{})
	env.shareWallAlbum(t, sharedToGrace, graceID, adminID, time.Date(2026, 11, 5, 9, 0, 0, 0, time.UTC))
	private := fixtureWriteAlbumID("85")
	env.insertWallAlbum(t, private, adminID, "virtual", "Private", wallAlbumOpts{})

	t.Run("DeletesVirtualAndCascades", func(t *testing.T) {
		rec := env.deleteAlbumRaw(t, env.cookie, virtual)
		if rec.Code != http.StatusNoContent {
			t.Fatalf("DELETE /albums/%s status = %d, want %d (body %q)",
				virtual, rec.Code, http.StatusNoContent, rec.Body.String())
		}
		if got := env.queryInt(t, "SELECT count(*) FROM albums WHERE id = $1", pgUUID(virtual)); got != 0 {
			t.Errorf("album rows = %d, want 0", got)
		}
		// The membership and share rows cascade with the row.
		if got := env.queryInt(t, "SELECT count(*) FROM album_items WHERE album_id = $1", pgUUID(virtual)); got != 0 {
			t.Errorf("membership rows = %d, want 0 (cascaded)", got)
		}
		if got := env.queryInt(t, "SELECT count(*) FROM album_shares WHERE album_id = $1", pgUUID(virtual)); got != 0 {
			t.Errorf("share rows = %d, want 0 (cascaded)", got)
		}
		// The assets themselves survive: deleting an album never
		// deletes photos.
		if got := env.queryInt(t, "SELECT count(*) FROM assets WHERE id = ANY($1)",
			pgUUIDs([]uuid.UUID{member, second})); got != 2 {
			t.Errorf("member assets = %d, want 2 (untouched)", got)
		}

		wall := env.getAlbumWall(t, env.cookie)
		for _, card := range wall.Albums {
			if uuid.UUID(card.Id) == virtual {
				t.Fatalf("the deleted album is still on the wall (%v)", wall.Albums)
			}
		}
		rec = env.getAlbumsRaw(t, env.cookie, "/api/v1/albums/"+virtual.String())
		requireAlbumNotFound(t, rec)
	})

	t.Run("SecondDeleteIsNotFound", func(t *testing.T) {
		rec := env.deleteAlbumRaw(t, env.cookie, virtual)
		requireAlbumNotFound(t, rec)
	})

	t.Run("DeletesLockedAlbum", func(t *testing.T) {
		// The permission matrix pins deletion to the UNLOCKED
		// state: the lock gates the delete like every other write,
		// and the rows stay intact.
		rec := env.deleteAlbumRaw(t, env.cookie, locked)
		if rec.Code != http.StatusConflict {
			t.Fatalf("DELETE locked album status = %d, want %d (body %q)",
				rec.Code, http.StatusConflict, rec.Body.String())
		}
		requireAlbumLocked(t, rec)
		// The refused delete leaves the album standing: the detail
		// read fatals on anything but 200.
		if detail := env.getAlbumDetail(t, env.cookie, locked, ""); uuid.UUID(detail.Id) != locked {
			t.Errorf("locked album id = %v, want %v", detail.Id, locked)
		}
	})

	t.Run("RefusesDirectoryAlbum", func(t *testing.T) {
		rec := env.deleteAlbumRaw(t, env.cookie, directory)
		requireAlbumNotFound(t, rec)
		if got := env.queryInt(t, "SELECT count(*) FROM albums WHERE id = $1", pgUUID(directory)); got != 1 {
			t.Errorf("directory album rows = %d, want 1 (directory removal is its own flow)", got)
		}
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

// TestAddAlbumItems pins the membership add: live caller-owned
// assets land with the caller as added_by, everything else skips
// silently (foreign, trashed, hidden, component, unknown, and the
// already-members -- the idempotent add), repeated ids collapse, and
// the two-tier authorization plus the locked gate hold.
func TestAddAlbumItems(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	adminID := env.sessionUserID(t)
	graceID, graceCookie := env.createSecondUser(t, "add-items-grace@example.com")

	album := fixtureWriteAlbumID("91")
	env.insertWallAlbum(t, album, adminID, "virtual", "Curation", wallAlbumOpts{})
	locked := fixtureWriteAlbumID("92")
	env.insertWallAlbum(t, locked, adminID, "virtual", "Frozen", wallAlbumOpts{locked: true})
	sharedToGrace := fixtureWriteAlbumID("93")
	env.insertWallAlbum(t, sharedToGrace, adminID, "virtual", "Shared", wallAlbumOpts{})
	env.shareWallAlbum(t, sharedToGrace, graceID, adminID, time.Date(2026, 11, 5, 9, 0, 0, 0, time.UTC))
	directory := fixtureWriteAlbumID("95")
	env.insertWallAlbum(t, directory, adminID, "directory", "Archive", wallAlbumOpts{})

	base := time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC)
	first := fixtureWallAssetID("91")
	second := fixtureWallAssetID("92")
	third := fixtureWallAssetID("93")
	fourth := fixtureWallAssetID("94")
	for i, asset := range []uuid.UUID{first, second, third, fourth} {
		env.insertTimelineAsset(t, asset, adminID, base.Add(time.Duration(i)*time.Hour), timelineAssetOpts{})
	}
	foreign := fixtureWallAssetID("95")
	env.insertTimelineAsset(t, foreign, graceID, base, timelineAssetOpts{})
	trashed := fixtureWallAssetID("96")
	env.insertTimelineAsset(t, trashed, adminID, base, timelineAssetOpts{deleted: true})
	hidden := fixtureWallAssetID("97")
	env.insertTimelineAsset(t, hidden, adminID, base, timelineAssetOpts{isHidden: true})
	component := fixtureWallAssetID("98")
	env.insertTimelineAsset(t, component, adminID, base, timelineAssetOpts{isComponent: true})
	unknown := fixtureWallAssetID("99")

	t.Run("AddsLiveCallerAssets", func(t *testing.T) {
		report := decodeAddItemsReport(t, env.postAlbumItems(t, env.cookie, album, albumItemsBody(first, second, third)))
		if report.Added != 3 || report.Skipped != 0 {
			t.Errorf("added/skipped = %d/%d, want 3/0", report.Added, report.Skipped)
		}
		if got := env.queryInt(t, "SELECT count(*) FROM album_items WHERE album_id = $1", pgUUID(album)); got != 3 {
			t.Errorf("membership rows = %d, want 3", got)
		}
		for _, asset := range []uuid.UUID{first, second, third} {
			if got := env.queryText(t,
				"SELECT added_by::text FROM album_items WHERE album_id = $1 AND asset_id = $2",
				pgUUID(album), pgUUID(asset)); got != adminID.String() {
				t.Errorf("added_by of %s = %s, want the caller %s", asset, got, adminID)
			}
		}

		detail := env.getAlbumDetail(t, env.cookie, album, "")
		if detail.ItemCount != 3 {
			t.Errorf("detail itemCount = %d, want 3", detail.ItemCount)
		}
		wall := env.getAlbumWall(t, env.cookie)
		for _, card := range wall.Albums {
			if uuid.UUID(card.Id) == album && card.ItemCount != 3 {
				t.Errorf("wall itemCount = %d, want 3", card.ItemCount)
			}
		}
	})

	t.Run("IdempotentReaddSkips", func(t *testing.T) {
		report := decodeAddItemsReport(t, env.postAlbumItems(t, env.cookie, album, albumItemsBody(first, second, third)))
		if report.Added != 0 || report.Skipped != 3 {
			t.Errorf("added/skipped = %d/%d, want 0/3 (already members)", report.Added, report.Skipped)
		}
		if got := env.queryInt(t, "SELECT count(*) FROM album_items WHERE album_id = $1", pgUUID(album)); got != 3 {
			t.Errorf("membership rows = %d, want 3 (the set is unchanged)", got)
		}
	})

	t.Run("MixedNewAndExisting", func(t *testing.T) {
		report := decodeAddItemsReport(t, env.postAlbumItems(t, env.cookie, album, albumItemsBody(first, fourth)))
		if report.Added != 1 || report.Skipped != 1 {
			t.Errorf("added/skipped = %d/%d, want 1/1 (the new member lands, the member skips)", report.Added, report.Skipped)
		}
		if got := env.queryInt(t, "SELECT count(*) FROM album_items WHERE album_id = $1", pgUUID(album)); got != 4 {
			t.Errorf("membership rows = %d, want 4", got)
		}
	})

	t.Run("SkipsForeignTrashedHiddenComponentUnknown", func(t *testing.T) {
		report := decodeAddItemsReport(t, env.postAlbumItems(t, env.cookie, album,
			albumItemsBody(foreign, trashed, hidden, component, unknown)))
		if report.Added != 0 || report.Skipped != 5 {
			t.Errorf("added/skipped = %d/%d, want 0/5 (all skipped, still a 200)", report.Added, report.Skipped)
		}
		if got := env.queryInt(t, "SELECT count(*) FROM album_items WHERE album_id = $1", pgUUID(album)); got != 4 {
			t.Errorf("membership rows = %d, want 4 (nothing landed)", got)
		}
	})

	t.Run("CollapsesRepeatedIds", func(t *testing.T) {
		fresh := fixtureWallAssetID("9a")
		env.insertTimelineAsset(t, fresh, adminID, base.Add(5*time.Hour), timelineAssetOpts{})
		report := decodeAddItemsReport(t, env.postAlbumItems(t, env.cookie, album, albumItemsBody(fresh, fresh)))
		if report.Added != 1 || report.Skipped != 0 {
			t.Errorf("added/skipped = %d/%d, want 1/0 (repeated ids collapse to one application)",
				report.Added, report.Skipped)
		}
		if got := env.queryInt(t,
			"SELECT count(*) FROM album_items WHERE album_id = $1 AND asset_id = $2",
			pgUUID(album), pgUUID(fresh)); got != 1 {
			t.Errorf("rows for the repeated id = %d, want 1", got)
		}
	})

	t.Run("ValidatesListBounds", func(t *testing.T) {
		rec := env.postAlbumItems(t, env.cookie, album, `{"assetIds":[]}`)
		requireValidationFields(t, rec, "assetIds")

		over := make([]uuid.UUID, 201)
		for i := range over {
			over[i] = uuid.MustParse(fmt.Sprintf("70000000-0000-0000-0002-%012x", i))
		}
		rec = env.postAlbumItems(t, env.cookie, album, albumItemsBody(over...))
		requireValidationFields(t, rec, "assetIds")

		rec = env.postAlbumItems(t, env.cookie, album, `{"assetIds":[`)
		requireValidationFields(t, rec, "body")

		// The 200 boundary itself answers -- the ids simply skip.
		atCap := over[:200]
		report := decodeAddItemsReport(t, env.postAlbumItems(t, env.cookie, album, albumItemsBody(atCap...)))
		if report.Added != 0 || report.Skipped != 200 {
			t.Errorf("added/skipped = %d/%d, want 0/200 at the cap boundary", report.Added, report.Skipped)
		}
	})

	t.Run("RefusesLockedAlbum", func(t *testing.T) {
		rec := env.postAlbumItems(t, env.cookie, locked, albumItemsBody(first))
		requireAlbumLocked(t, rec)
		if got := env.queryInt(t, "SELECT count(*) FROM album_items WHERE album_id = $1", pgUUID(locked)); got != 0 {
			t.Errorf("locked album membership rows = %d, want 0", got)
		}
	})

	t.Run("ShareeIsForbidden", func(t *testing.T) {
		rec := env.postAlbumItems(t, graceCookie, sharedToGrace, albumItemsBody(first))
		requireAlbumWriteForbidden(t, rec)
	})

	t.Run("StrangerIsNotFound", func(t *testing.T) {
		rec := env.postAlbumItems(t, graceCookie, album, albumItemsBody(first))
		requireAlbumNotFound(t, rec)
	})

	t.Run("RefusesDirectoryAlbum", func(t *testing.T) {
		rec := env.postAlbumItems(t, env.cookie, directory, albumItemsBody(first))
		requireAlbumNotFound(t, rec)
		if got := env.queryInt(t, "SELECT count(*) FROM album_items WHERE album_id = $1", pgUUID(directory)); got != 0 {
			t.Errorf("directory membership rows = %d, want 0 (the rescan owns membership)", got)
		}
	})

	t.Run("RefusesSyncAlbum", func(t *testing.T) {
		ownSync := env.syncAlbumIDOf(t, adminID)
		rec := env.postAlbumItems(t, env.cookie, ownSync, albumItemsBody(first))
		requireAlbumNotFound(t, rec)
	})

	t.Run("UnknownAlbum", func(t *testing.T) {
		rec := env.postAlbumItems(t, env.cookie, fixtureWallAlbumID("ff"), albumItemsBody(first))
		requireAlbumNotFound(t, rec)
	})
}

// TestRemoveAlbumItems pins the membership remove: the rows that
// exist leave, absent ids skip, the trashed member's row leaves too
// (no live predicate on the removal), and the authorization, type
// and lock gates mirror the add.
func TestRemoveAlbumItems(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	adminID := env.sessionUserID(t)
	graceID, graceCookie := env.createSecondUser(t, "remove-items-grace@example.com")

	album := fixtureWriteAlbumID("a1")
	env.insertWallAlbum(t, album, adminID, "virtual", "Curation", wallAlbumOpts{})
	locked := fixtureWriteAlbumID("a2")
	env.insertWallAlbum(t, locked, adminID, "virtual", "Frozen", wallAlbumOpts{locked: true})
	sharedToGrace := fixtureWriteAlbumID("a3")
	env.insertWallAlbum(t, sharedToGrace, adminID, "virtual", "Shared", wallAlbumOpts{})
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
	lockedMember := fixtureWallAssetID("a5")
	env.insertTimelineAsset(t, lockedMember, adminID, base.Add(4*time.Hour), timelineAssetOpts{})
	env.addTimelineItem(t, locked, lockedMember, adminID)

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

	t.Run("RefusesLockedAlbum", func(t *testing.T) {
		rec := env.deleteAlbumItems(t, env.cookie, locked, albumItemsBody(lockedMember))
		requireAlbumLocked(t, rec)
		if got := env.queryInt(t, "SELECT count(*) FROM album_items WHERE album_id = $1", pgUUID(locked)); got != 1 {
			t.Errorf("locked album membership rows = %d, want 1 (the removal landed nothing)", got)
		}
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

// TestAlbumWriteEvents pins the four live event names of the write
// surface: creation, patch, membership change and delete, each with
// the ids-only payload -- the album id and the acting user's id, and
// nothing else. No album state (title, description, cover, lock,
// counts, membership) ever rides the shared bus.
func TestAlbumWriteEvents(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	adminID := env.sessionUserID(t)

	// The full lifecycle through the write endpoints: create, patch,
	// add two members, re-add one (the idempotent no-op), remove
	// one, delete.
	created := env.createAlbum(t, env.cookie, "The eventful album", nil)
	albumID := uuid.UUID(created.Id)
	env.patchAlbum(t, env.cookie, albumID, `{"description":"Now with a description."}`)

	member := fixtureWallAssetID("b1")
	second := fixtureWallAssetID("b2")
	base := time.Date(2026, 8, 6, 12, 0, 0, 0, time.UTC)
	env.insertTimelineAsset(t, member, adminID, base, timelineAssetOpts{})
	env.insertTimelineAsset(t, second, adminID, base.Add(time.Hour), timelineAssetOpts{})

	addReport := decodeAddItemsReport(t, env.postAlbumItems(t, env.cookie, albumID, albumItemsBody(member, second)))
	if addReport.Added != 2 {
		t.Fatalf("the add report = %+v, want 2 added (the fixture is broken)", addReport)
	}
	readd := decodeAddItemsReport(t, env.postAlbumItems(t, env.cookie, albumID, albumItemsBody(member)))
	if readd.Added != 0 {
		t.Fatalf("the idempotent re-add report = %+v, want 0 added", readd)
	}
	removeReport := decodeRemoveItemsReport(t, env.deleteAlbumItems(t, env.cookie, albumID, albumItemsBody(member)))
	if removeReport.Removed != 1 {
		t.Fatalf("the remove report = %+v, want 1 removed", removeReport)
	}
	rec := env.deleteAlbumRaw(t, env.cookie, albumID)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("DELETE /albums/%s status = %d, want %d", albumID, rec.Code, http.StatusNoContent)
	}

	cases := []struct {
		name  string
		count int
	}{
		{"album.created", 1},
		{"album.updated", 1}, // the single patch, unconditional on success
		{"album.items-changed", 2},
		{"album.deleted", 1},
	}
	for _, want := range cases {
		payloads := busEventsOf(t, env.bus, want.name)
		if len(payloads) != want.count {
			t.Fatalf("%s events = %d, want exactly %d (payloads %v)",
				want.name, len(payloads), want.count, payloads)
		}
		for _, payload := range payloads {
			if payload["albumId"] != albumID.String() {
				t.Errorf("%s albumId = %q, want %s", want.name, payload["albumId"], albumID)
			}
			if payload["userId"] != adminID.String() {
				t.Errorf("%s userId = %q, want the acting caller %s", want.name, payload["userId"], adminID)
			}
			// ids only: exactly the two keys, no album state.
			if len(payload) != 2 {
				t.Errorf("%s payload = %v, want exactly the albumId/userId pair", want.name, payload)
			}
			for _, banned := range []string{
				"title", "description", "type", "locked", "coverAssetId",
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
