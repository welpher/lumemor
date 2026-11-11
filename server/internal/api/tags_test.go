// Tags surface tests: the caller's private vocabulary
// and its batch assignment. The domain's privacy shape is favorites'
// : every tag, every count and every assignment row is the
// caller's own -- another account's tags never surface on any face,
// and the asset detail's tags array answers the CALLER's tags only,
// so the same shared asset answers different arrays for the owner
// and the sharee. The assignment's skip semantics follow the batch
// discipline: ids not visible to the caller (the album union) and
// already/not-tagged ids are skipped silently, and an all-skipped
// request still answers 200.
package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	apigen "github.com/welpher/lumemor/server/internal/api/gen"
)

// tagAssetID builds a deterministic fixture asset id of the tags
// family, in a namespace disjoint from every other fixture family.
func tagAssetID(suffix string) uuid.UUID {
	return uuid.MustParse("70000000-0000-0000-0000-0000000000" + suffix)
}

// writeTagRaw is the tags write dispatcher: an empty body sends no
// request content at all, anything else goes out as JSON. A nil
// cookie sends the request unauthenticated.
func (e *assetEnv) writeTagRaw(t *testing.T, cookie *http.Cookie, method, target, body string) *httptest.ResponseRecorder {
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

// tagNameBody assembles the name-carrying body of the tag writes.
func tagNameBody(t *testing.T, name string) string {
	t.Helper()
	quoted, err := json.Marshal(name)
	if err != nil {
		t.Fatalf("quote name %q: %v", name, err)
	}
	return `{"name":` + string(quoted) + `}`
}

// tagAssetsBody assembles the assignment write body.
func tagAssetsBody(ids ...uuid.UUID) string {
	quoted := make([]string, len(ids))
	for i, id := range ids {
		quoted[i] = `"` + id.String() + `"`
	}
	return `{"assetIds":[` + strings.Join(quoted, ",") + `]}`
}

// createTag fires POST /api/v1/tags and decodes the tag summary.
func (e *assetEnv) createTag(t *testing.T, cookie *http.Cookie, name string) (int, apigen.TagSummary) {
	t.Helper()
	rec := e.writeTagRaw(t, cookie, http.MethodPost, "/api/v1/tags", tagNameBody(t, name))
	if rec.Code != http.StatusCreated && rec.Code != http.StatusOK {
		t.Fatalf("POST /tags status = %d, want 201 or 200 (body %q)", rec.Code, rec.Body.String())
	}
	var summary apigen.TagSummary
	if err := json.Unmarshal(rec.Body.Bytes(), &summary); err != nil {
		t.Fatalf("decode create tag body %q: %v", rec.Body.String(), err)
	}
	return rec.Code, summary
}

// listTags fires GET /api/v1/tags and decodes the listing.
func listTags(t *testing.T, router http.Handler, cookie *http.Cookie) []apigen.TagSummary {
	t.Helper()
	rec := authedRequest(router, http.MethodGet, "/api/v1/tags", cookie)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /tags status = %d, want %d (body %q)", rec.Code, http.StatusOK, rec.Body.String())
	}
	var tags []apigen.TagSummary
	if err := json.Unmarshal(rec.Body.Bytes(), &tags); err != nil {
		t.Fatalf("decode tags %q: %v", rec.Body.String(), err)
	}
	return tags
}

// requireTagSummary asserts one tag summary's identity fields.
func requireTagSummary(t *testing.T, got apigen.TagSummary, name string, assetCount int) {
	t.Helper()
	if got.Name != name || got.AssetCount != assetCount {
		t.Errorf("tag = {%q count %d}, want {%q count %d}", got.Name, got.AssetCount, name, assetCount)
	}
	if got.Id == uuid.Nil {
		t.Errorf("tag %q carries the nil id", name)
	}
	if got.CreatedAt.IsZero() || got.UpdatedAt.IsZero() {
		t.Errorf("tag %q carries zero timestamps (%v, %v)", name, got.CreatedAt, got.UpdatedAt)
	}
}

// requireTagStatus asserts the tags face answered the given status
// and code.
func requireTagStatus(t *testing.T, rec *httptest.ResponseRecorder, status int, code apigen.ErrorCode) {
	t.Helper()
	if rec.Code != status {
		t.Fatalf("status = %d, want %d (body %q)", rec.Code, status, rec.Body.String())
	}
	apiErr, _ := decodeError(t, rec)
	if apiErr.Code != code {
		t.Errorf("code = %q, want %q", apiErr.Code, code)
	}
}

// requireTagReport asserts one assignment report's shape.
func requireTagReport(t *testing.T, rec *httptest.ResponseRecorder, applied, skipped int) {
	t.Helper()
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d (body %q)", rec.Code, http.StatusOK, rec.Body.String())
	}
	var body map[string]int
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode report %q: %v", rec.Body.String(), err)
	}
	if len(body) != 2 {
		t.Fatalf("report keys = %v, want exactly the two counts", body)
	}
	if body["skipped"] != skipped {
		t.Errorf("skipped = %d, want %d", body["skipped"], skipped)
	}
	if _, ok := body["added"]; ok && body["added"] != applied {
		t.Errorf("added = %d, want %d", body["added"], applied)
	}
	if _, ok := body["removed"]; ok && body["removed"] != applied {
		t.Errorf("removed = %d, want %d", body["removed"], applied)
	}
}

// TestTagsVocabulary pins the tags CRUD face: the idempotent create
// (a duplicate name converges on the existing tag, 200 vs 201), the
// name validation, the rename with its no-op and collision arms, the
// delete with the cascading assignment rows, and the listing's name
// ordering with the honest counts.
func TestTagsVocabulary(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	adminID := env.sessionUserID(t)

	// The assets the assignment arms act on: three live own assets.
	a1 := tagAssetID("01")
	a2 := tagAssetID("02")
	a3 := tagAssetID("03")
	for i, id := range []uuid.UUID{a1, a2, a3} {
		env.insertOwnedAsset(t, id, adminID, time.Date(2026, 7, 1+i, 12, 0, 0, 0, time.UTC), timelineAssetOpts{})
	}

	t.Run("UnauthenticatedRefused", func(t *testing.T) {
		rec := env.writeTagRaw(t, nil, http.MethodGet, "/api/v1/tags", "")
		requireTagStatus(t, rec, http.StatusUnauthorized, apigen.AUTHINVALIDCREDENTIALS)
	})

	t.Run("CreateIsIdempotent", func(t *testing.T) {
		status, fresh := env.createTag(t, env.cookie, "Trip")
		if status != http.StatusCreated {
			t.Fatalf("fresh create status = %d, want %d", status, http.StatusCreated)
		}
		requireTagSummary(t, fresh, "Trip", 0)

		// The duplicate ask converges on the existing tag: 200, the
		// SAME id, nothing written (the timestamps untouched).
		status, again := env.createTag(t, env.cookie, "Trip")
		if status != http.StatusOK {
			t.Fatalf("duplicate create status = %d, want %d", status, http.StatusOK)
		}
		if again.Id != fresh.Id {
			t.Errorf("duplicate create id = %s, want the existing %s", again.Id, fresh.Id)
		}
		if !again.UpdatedAt.Equal(fresh.UpdatedAt) {
			t.Errorf("duplicate create updatedAt = %v, want the untouched %v", again.UpdatedAt, fresh.UpdatedAt)
		}
		requireTagSummary(t, again, "Trip", 0)

		// The trim: whitespace around the name never survives.
		status, trimmed := env.createTag(t, env.cookie, "  Trip  ")
		if status != http.StatusOK {
			t.Fatalf("trimmed duplicate status = %d, want %d", status, http.StatusOK)
		}
		if trimmed.Id != fresh.Id {
			t.Errorf("trimmed duplicate id = %s, want the existing %s", trimmed.Id, fresh.Id)
		}

		// Case-sensitive names are distinct vocabulary entries (the
		// constraint's own equality): a fresh 201.
		status, cased := env.createTag(t, env.cookie, "trip")
		if status != http.StatusCreated {
			t.Fatalf("case-distinct create status = %d, want %d", status, http.StatusCreated)
		}
		if cased.Id == fresh.Id {
			t.Errorf("case-distinct create collapsed onto %s", fresh.Id)
		}
	})

	t.Run("CreateValidation", func(t *testing.T) {
		for name, label := range map[string]string{
			"":            "empty",
			"   ":         "whitespace-only",
			"\t\n":        "whitespace-only tab",
			"0123456789a": "65 runes", // padded below to 65
		} {
			value := name
			if label == "65 runes" {
				value = strings.Repeat("0123456789a", 6)[:65]
			}
			rec := env.writeTagRaw(t, env.cookie, http.MethodPost, "/api/v1/tags", tagNameBody(t, value))
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("create %q status = %d, want %d (body %q)", label, rec.Code, http.StatusBadRequest, rec.Body.String())
			}
			apiErr, fields := decodeError(t, rec)
			if apiErr.Code != apigen.VALIDATIONFAILED {
				t.Errorf("create %q code = %q, want VALIDATION_FAILED", label, apiErr.Code)
			}
			if len(fields) != 1 || fields[0] != "name" {
				t.Errorf("create %q fields = %v, want [name]", label, fields)
			}
		}

		// 64 runes pass (the boundary).
		boundary := strings.Repeat("字", 64)
		status, _ := env.createTag(t, env.cookie, boundary)
		if status != http.StatusCreated {
			t.Fatalf("64-rune create status = %d, want %d", status, http.StatusCreated)
		}

		// The body must be valid JSON.
		rec := env.writeTagRaw(t, env.cookie, http.MethodPost, "/api/v1/tags", "{")
		requireTagStatus(t, rec, http.StatusBadRequest, apigen.VALIDATIONFAILED)
	})

	t.Run("RenameArms", func(t *testing.T) {
		_, trip := env.createTag(t, env.cookie, "Rename Trip")
		_, beach := env.createTag(t, env.cookie, "Rename Beach")

		// The plain rename.
		rec := env.writeTagRaw(t, env.cookie, http.MethodPatch, "/api/v1/tags/"+trip.Id.String(), tagNameBody(t, "Roadtrip"))
		if rec.Code != http.StatusOK {
			t.Fatalf("rename status = %d, want %d (body %q)", rec.Code, http.StatusOK, rec.Body.String())
		}
		var renamed apigen.TagSummary
		if err := json.Unmarshal(rec.Body.Bytes(), &renamed); err != nil {
			t.Fatalf("decode rename body %q: %v", rec.Body.String(), err)
		}
		if renamed.Id != trip.Id || renamed.Name != "Roadtrip" {
			t.Errorf("rename = {%s %q}, want {%s Roadtrip}", renamed.Id, renamed.Name, trip.Id)
		}
		if !renamed.UpdatedAt.After(trip.UpdatedAt) {
			t.Errorf("rename updatedAt = %v, want it past %v", renamed.UpdatedAt, trip.UpdatedAt)
		}

		// The idempotent no-op: renaming to the row's own name writes
		// nothing -- not even updatedAt.
		before := renamed.UpdatedAt
		rec = env.writeTagRaw(t, env.cookie, http.MethodPatch, "/api/v1/tags/"+trip.Id.String(), tagNameBody(t, "Roadtrip"))
		if rec.Code != http.StatusOK {
			t.Fatalf("no-op rename status = %d, want %d", rec.Code, http.StatusOK)
		}
		var noop apigen.TagSummary
		if err := json.Unmarshal(rec.Body.Bytes(), &noop); err != nil {
			t.Fatalf("decode no-op rename body %q: %v", rec.Body.String(), err)
		}
		if !noop.UpdatedAt.Equal(before) {
			t.Errorf("no-op rename updatedAt = %v, want the untouched %v", noop.UpdatedAt, before)
		}

		// The collision: another of the caller's tags carries the
		// name -- 409, the vocabulary keeps one row per name.
		rec = env.writeTagRaw(t, env.cookie, http.MethodPatch, "/api/v1/tags/"+trip.Id.String(), tagNameBody(t, "Rename Beach"))
		requireTagStatus(t, rec, http.StatusConflict, apigen.TAGNAMETAKEN)

		// The unknown id: single-shape 404.
		unknown := uuid.MustParse("70000000-0000-0000-0000-00ffffff00ff")
		rec = env.writeTagRaw(t, env.cookie, http.MethodPatch, "/api/v1/tags/"+unknown.String(), tagNameBody(t, "Nowhere"))
		requireTagStatus(t, rec, http.StatusNotFound, apigen.TAGNOTFOUND)

		// The name validation is the create's own.
		rec = env.writeTagRaw(t, env.cookie, http.MethodPatch, "/api/v1/tags/"+trip.Id.String(), tagNameBody(t, "  "))
		requireTagStatus(t, rec, http.StatusBadRequest, apigen.VALIDATIONFAILED)

		_ = beach
	})

	t.Run("DeleteCascades", func(t *testing.T) {
		_, tag := env.createTag(t, env.cookie, "Cascade")
		rec := env.writeTagRaw(t, env.cookie, http.MethodPost,
			"/api/v1/tags/"+tag.Id.String()+"/assets", tagAssetsBody(a1, a2))
		requireTagReport(t, rec, 2, 0)
		if got := env.queryInt(t, "SELECT count(*) FROM asset_tags WHERE tag_id = $1", tag.Id.String()); got != 2 {
			t.Fatalf("assignment rows = %d, want 2", got)
		}

		rec = env.writeTagRaw(t, env.cookie, http.MethodDelete, "/api/v1/tags/"+tag.Id.String(), "")
		if rec.Code != http.StatusNoContent {
			t.Fatalf("delete status = %d, want %d", rec.Code, http.StatusNoContent)
		}
		if got := env.queryInt(t, "SELECT count(*) FROM asset_tags WHERE tag_id = $1", tag.Id.String()); got != 0 {
			t.Fatalf("assignment rows after delete = %d, want 0 (the FK cascade)", got)
		}

		// The repeat answers 404: the tag is gone exactly like an
		// unknown id (the album delete's discipline).
		rec = env.writeTagRaw(t, env.cookie, http.MethodDelete, "/api/v1/tags/"+tag.Id.String(), "")
		requireTagStatus(t, rec, http.StatusNotFound, apigen.TAGNOTFOUND)
	})

	t.Run("ListingOrdersAndCounts", func(t *testing.T) {
		// A fresh environment state: everything above still lists, so
		// count over the whole vocabulary and check the ordering.
		tags := listTags(t, env.router, env.cookie)
		if len(tags) < 2 {
			t.Fatalf("tags = %d entries, want the vocabulary of this test", len(tags))
		}
		for i := 1; i < len(tags); i++ {
			if strings.ToLower(tags[i-1].Name) > strings.ToLower(tags[i].Name) {
				t.Fatalf("listing order broke at %q before %q", tags[i-1].Name, tags[i].Name)
			}
		}
		// The vocabulary carries no assignment rows anymore (the
		// cascade arm removed them), so every count is the honest 0.
		for _, tag := range tags {
			if tag.AssetCount != 0 {
				t.Errorf("tag %q count = %d, want 0", tag.Name, tag.AssetCount)
			}
		}
	})
}

// TestTagAssignmentBatch pins the assignment face: the honest
// {added|removed, skipped} report over the distinct ids, the union
// predicate's resolution domain (a sharee tags what they see, an
// invisible id is skipped, never an error), the list bounds, and the
// per-user state's survival of the asset's live state.
func TestTagAssignmentBatch(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	adminID := env.sessionUserID(t)
	graceID, graceCookie := env.createSecondUser(t, "tag-batch-grace@example.com")

	// The admin's assets: a1 and a2 shared to Grace through a memory
	// album, a3 unshared.
	a1 := tagAssetID("11")
	a2 := tagAssetID("12")
	a3 := tagAssetID("13")
	for i, id := range []uuid.UUID{a1, a2, a3} {
		env.insertOwnedAsset(t, id, adminID, time.Date(2026, 8, 1+i, 12, 0, 0, 0, time.UTC), timelineAssetOpts{})
	}
	sharedAlbum := fixtureAlbumID("e1")
	env.insertTimelineAlbum(t, sharedAlbum, adminID, "memory")
	env.shareTimelineAlbum(t, sharedAlbum, graceID, adminID)
	env.addTimelineItem(t, sharedAlbum, a1, adminID)
	env.addTimelineItem(t, sharedAlbum, a2, adminID)

	// Grace's own asset.
	g1 := tagAssetID("14")
	env.insertOwnedAsset(t, g1, graceID, time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC), timelineAssetOpts{})

	_, tag := env.createTag(t, env.cookie, "Batch")

	t.Run("AddReportAndBounds", func(t *testing.T) {
		rec := env.writeTagRaw(t, env.cookie, http.MethodPost,
			"/api/v1/tags/"+tag.Id.String()+"/assets", tagAssetsBody(a1, a2))
		requireTagReport(t, rec, 2, 0)

		// The idempotent re-add: every id is already tagged.
		rec = env.writeTagRaw(t, env.cookie, http.MethodPost,
			"/api/v1/tags/"+tag.Id.String()+"/assets", tagAssetsBody(a1, a2))
		requireTagReport(t, rec, 0, 2)

		// Mixed: one fresh, the rest skipped -- a3 is the caller's
		// own but was never tagged, the unknown id resolves nothing,
		// the duplicate collapses (the counts cover the DISTINCT ids:
		// two of them here).
		unknown := uuid.MustParse("70000000-0000-0000-0000-00ffffff00ff")
		rec = env.writeTagRaw(t, env.cookie, http.MethodPost,
			"/api/v1/tags/"+tag.Id.String()+"/assets",
			tagAssetsBody(a3, a3, unknown))
		requireTagReport(t, rec, 1, 1)

		// All-skipped still answers 200.
		rec = env.writeTagRaw(t, env.cookie, http.MethodPost,
			"/api/v1/tags/"+tag.Id.String()+"/assets", tagAssetsBody(unknown))
		requireTagReport(t, rec, 0, 1)

		// The bounds: an empty list and a list past 200 are client
		// bugs the 400 names.
		rec = env.writeTagRaw(t, env.cookie, http.MethodPost,
			"/api/v1/tags/"+tag.Id.String()+"/assets", `{"assetIds":[]}`)
		requireTagStatus(t, rec, http.StatusBadRequest, apigen.VALIDATIONFAILED)
		rec = env.writeTagRaw(t, env.cookie, http.MethodPost,
			"/api/v1/tags/"+tag.Id.String()+"/assets", `{"assetIds":[`+strings.Repeat(`"`+a1.String()+`",`, 201)+`"`+a1.String()+`"]}`)
		requireTagStatus(t, rec, http.StatusBadRequest, apigen.VALIDATIONFAILED)
	})

	t.Run("RemoveReport", func(t *testing.T) {
		rec := env.writeTagRaw(t, env.cookie, http.MethodDelete,
			"/api/v1/tags/"+tag.Id.String()+"/assets", tagAssetsBody(a1, a2, a3))
		requireTagReport(t, rec, 3, 0)

		// The ids that were no assignments of this tag.
		rec = env.writeTagRaw(t, env.cookie, http.MethodDelete,
			"/api/v1/tags/"+tag.Id.String()+"/assets", tagAssetsBody(a1, a2))
		requireTagReport(t, rec, 0, 2)
	})

	t.Run("UnionCarriesTheSharee", func(t *testing.T) {
		// The caller-visible widening over the album-items precedent: a sharee
		// tags what they see with THEIR OWN tag.
		_, graceTag := env.createTag(t, graceCookie, "Grace Pick")
		rec := env.writeTagRaw(t, graceCookie, http.MethodPost,
			"/api/v1/tags/"+graceTag.Id.String()+"/assets", tagAssetsBody(a1))
		requireTagReport(t, rec, 1, 0)

		// Grace's own asset carries her tag too.
		rec = env.writeTagRaw(t, graceCookie, http.MethodPost,
			"/api/v1/tags/"+graceTag.Id.String()+"/assets", tagAssetsBody(g1))
		requireTagReport(t, rec, 1, 0)

		// An asset the caller cannot see is skipped, never an error:
		// a3 is the admin's unshared one.
		rec = env.writeTagRaw(t, graceCookie, http.MethodPost,
			"/api/v1/tags/"+graceTag.Id.String()+"/assets", tagAssetsBody(a3))
		requireTagReport(t, rec, 0, 1)

		// Her listing counts the union honestly: a1 (shared) and g1
		// (her own).
		graceTags := listTags(t, env.router, graceCookie)
		if len(graceTags) != 1 {
			t.Fatalf("grace tags = %d entries, want 1", len(graceTags))
		}
		requireTagSummary(t, graceTags[0], "Grace Pick", 2)
	})

	t.Run("LiveStateSurvivesUnderTheCount", func(t *testing.T) {
		// The assignment row is per-user state written whatever the
		// asset's live state: the count filters, the row survives.
		_, liveTag := env.createTag(t, env.cookie, "Live State")
		rec := env.writeTagRaw(t, env.cookie, http.MethodPost,
			"/api/v1/tags/"+liveTag.Id.String()+"/assets", tagAssetsBody(a1))
		requireTagReport(t, rec, 1, 0)

		env.exec(t, "UPDATE assets SET deleted_at = now() WHERE id = $1", a1.String())
		for _, tag := range listTags(t, env.router, env.cookie) {
			if tag.Id == liveTag.Id && tag.AssetCount != 0 {
				t.Errorf("trashed member still counts: %d", tag.AssetCount)
			}
		}
		env.exec(t, "UPDATE assets SET deleted_at = NULL WHERE id = $1", a1.String())
		for _, tag := range listTags(t, env.router, env.cookie) {
			if tag.Id == liveTag.Id && tag.AssetCount != 1 {
				t.Errorf("restored member does not count: %d", tag.AssetCount)
			}
		}
	})

	t.Run("ForeignTagIdRefused", func(t *testing.T) {
		// The admin's tag through Grace's session: a tag another
		// account owns does not exist for her (single-shape 404).
		rec := env.writeTagRaw(t, graceCookie, http.MethodPost,
			"/api/v1/tags/"+tag.Id.String()+"/assets", tagAssetsBody(a1))
		requireTagStatus(t, rec, http.StatusNotFound, apigen.TAGNOTFOUND)
		rec = env.writeTagRaw(t, graceCookie, http.MethodDelete,
			"/api/v1/tags/"+tag.Id.String()+"/assets", tagAssetsBody(a1))
		requireTagStatus(t, rec, http.StatusNotFound, apigen.TAGNOTFOUND)
		rec = env.writeTagRaw(t, graceCookie, http.MethodPatch,
			"/api/v1/tags/"+tag.Id.String(), tagNameBody(t, "Stolen"))
		requireTagStatus(t, rec, http.StatusNotFound, apigen.TAGNOTFOUND)
		rec = env.writeTagRaw(t, graceCookie, http.MethodDelete,
			"/api/v1/tags/"+tag.Id.String(), "")
		requireTagStatus(t, rec, http.StatusNotFound, apigen.TAGNOTFOUND)

		// The unknown id answers the same shape.
		unknown := uuid.MustParse("70000000-0000-0000-0000-00ffffff00ff")
		rec = env.writeTagRaw(t, env.cookie, http.MethodPost,
			"/api/v1/tags/"+unknown.String()+"/assets", tagAssetsBody(a1))
		requireTagStatus(t, rec, http.StatusNotFound, apigen.TAGNOTFOUND)
	})
}

// TestTagsCallerIsolation pins the per-user discipline across two accounts: the same
// shared asset carries each viewer's own tags and only those, the
// listings never leak, the foreign tag id folds into the 404, and a
// revoked share retracts the sharee's count while her assignment row
// survives underneath.
func TestTagsCallerIsolation(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	adminID := env.sessionUserID(t)
	graceID, graceCookie := env.createSecondUser(t, "tag-iso-grace@example.com")

	// One shared asset, one unshared.
	shared := tagAssetID("21")
	private := tagAssetID("22")
	env.insertOwnedAsset(t, shared, adminID, time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC), timelineAssetOpts{})
	env.insertOwnedAsset(t, private, adminID, time.Date(2026, 9, 2, 12, 0, 0, 0, time.UTC), timelineAssetOpts{})
	album := fixtureAlbumID("e2")
	env.insertTimelineAlbum(t, album, adminID, "memory")
	env.shareTimelineAlbum(t, album, graceID, adminID)
	env.addTimelineItem(t, album, shared, adminID)

	// Each account tags the SAME asset with their own vocabulary.
	_, adminTag := env.createTag(t, env.cookie, "Sunset")
	_, adminOther := env.createTag(t, env.cookie, "Alpha")
	rec := env.writeTagRaw(t, env.cookie, http.MethodPost,
		"/api/v1/tags/"+adminTag.Id.String()+"/assets", tagAssetsBody(shared))
	requireTagReport(t, rec, 1, 0)

	_, graceTag := env.createTag(t, graceCookie, "Grace Favorite")
	rec = env.writeTagRaw(t, graceCookie, http.MethodPost,
		"/api/v1/tags/"+graceTag.Id.String()+"/assets", tagAssetsBody(shared))
	requireTagReport(t, rec, 1, 0)

	getAssetTags := func(t *testing.T, cookie *http.Cookie, id uuid.UUID) []apigen.AssetTagSummary {
		t.Helper()
		rec := authedRequest(env.router, http.MethodGet, "/api/v1/assets/"+id.String(), cookie)
		if rec.Code != http.StatusOK {
			t.Fatalf("GET /assets/%s status = %d, want %d (body %q)", id, rec.Code, http.StatusOK, rec.Body.String())
		}
		var detail apigen.AssetDetail
		if err := json.Unmarshal(rec.Body.Bytes(), &detail); err != nil {
			t.Fatalf("decode detail %q: %v", rec.Body.String(), err)
		}
		if detail.Tags == nil {
			t.Fatalf("detail tags = null, want an array (body %q)", rec.Body.String())
		}
		return detail.Tags
	}

	t.Run("DetailAnswersTheCallersTagsOnly", func(t *testing.T) {
		adminTags := getAssetTags(t, env.cookie, shared)
		if len(adminTags) != 1 || adminTags[0].Id != adminTag.Id || adminTags[0].Name != "Sunset" {
			t.Fatalf("admin tags = %v, want exactly [Sunset]", adminTags)
		}
		graceTags := getAssetTags(t, graceCookie, shared)
		if len(graceTags) != 1 || graceTags[0].Id != graceTag.Id || graceTags[0].Name != "Grace Favorite" {
			t.Fatalf("grace tags = %v, want exactly [Grace Favorite]", graceTags)
		}
	})

	t.Run("ListingsNeverLeak", func(t *testing.T) {
		adminTags := listTags(t, env.router, env.cookie)
		if len(adminTags) != 2 {
			t.Fatalf("admin tags = %d entries, want 2", len(adminTags))
		}
		for _, tag := range adminTags {
			if tag.Id == graceTag.Id {
				t.Errorf("grace's tag %q leaked into the admin's listing", tag.Name)
			}
		}
		graceTags := listTags(t, env.router, graceCookie)
		if len(graceTags) != 1 || graceTags[0].Id != graceTag.Id {
			t.Fatalf("grace tags = %v, want exactly her own", graceTags)
		}
	})

	t.Run("DetailTagsOrderByName", func(t *testing.T) {
		// A second admin tag on the shared asset: the array answers
		// name ascending.
		rec := env.writeTagRaw(t, env.cookie, http.MethodPost,
			"/api/v1/tags/"+adminOther.Id.String()+"/assets", tagAssetsBody(shared))
		requireTagReport(t, rec, 1, 0)
		tags := getAssetTags(t, env.cookie, shared)
		if len(tags) != 2 {
			t.Fatalf("admin tags = %d entries, want 2", len(tags))
		}
		if tags[0].Name != "Alpha" || tags[1].Name != "Sunset" {
			t.Errorf("tags order = [%s, %s], want [Alpha, Sunset]", tags[0].Name, tags[1].Name)
		}

		// The PATCH response serves the same array.
		patch := env.writeTagRaw(t, env.cookie, http.MethodPatch,
			"/api/v1/assets/"+shared.String(), `{"description":"tagged"}`)
		if patch.Code != http.StatusOK {
			t.Fatalf("PATCH /assets status = %d, want %d", patch.Code, http.StatusOK)
		}
		var detail apigen.AssetDetail
		if err := json.Unmarshal(patch.Body.Bytes(), &detail); err != nil {
			t.Fatalf("decode patched detail %q: %v", patch.Body.String(), err)
		}
		if len(detail.Tags) != 2 || detail.Tags[0].Name != "Alpha" {
			t.Fatalf("patched tags = %v, want [Alpha, Sunset]", detail.Tags)
		}
	})

	t.Run("RevokedShareRetractsTheCountNotTheRow", func(t *testing.T) {
		env.exec(t, "DELETE FROM album_shares WHERE album_id = $1 AND user_id = $2",
			album.String(), graceID.String())

		// The asset itself is invisible to Grace now.
		rec := authedRequest(env.router, http.MethodGet, "/api/v1/assets/"+shared.String(), graceCookie)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("sharee detail status = %d, want %d", rec.Code, http.StatusNotFound)
		}

		// Her count retracts with the visibility; her assignment row
		// survives underneath (the favorites discipline).
		graceTags := listTags(t, env.router, graceCookie)
		if len(graceTags) != 1 || graceTags[0].AssetCount != 0 {
			t.Fatalf("grace tags after revoke = %v, want her tag with count 0", graceTags)
		}
		if got := env.queryInt(t, "SELECT count(*) FROM asset_tags WHERE tag_id = $1", graceTag.Id.String()); got != 1 {
			t.Fatalf("grace assignment rows after revoke = %d, want 1 (the row survives)", got)
		}

		// Re-granting returns the count wholesale.
		env.shareTimelineAlbum(t, album, graceID, adminID)
		graceTags = listTags(t, env.router, graceCookie)
		if len(graceTags) != 1 || graceTags[0].AssetCount != 1 {
			t.Fatalf("grace tags after re-grant = %v, want her tag with count 1", graceTags)
		}
	})

	t.Run("UntaggedDetailAnswersAnEmptyArray", func(t *testing.T) {
		tags := getAssetTags(t, env.cookie, private)
		if len(tags) != 0 {
			t.Fatalf("untagged detail tags = %v, want []", tags)
		}
	})
}
