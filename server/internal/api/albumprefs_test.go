// Album timeline-preference tests: the per-user
// "keep this album out of my timeline" switch behind
// PUT/DELETE /albums/{albumId}/timeline-preference -- the scope
// (own directory + shared albums only), the upsert and reset
// semantics, the single-shape 404 of an invisible album, and the
// no-event discipline of a self-affecting write. The exclusion
// itself is observed through the one consumer that honors it: the
// aggregated timeline (the album's own view never filters on it).
package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	apigen "github.com/welpher/lumemor/server/internal/api/gen"
)

// prefTarget renders the preference write's path.
func prefTarget(albumID uuid.UUID) string {
	return "/api/v1/albums/" + albumID.String() + "/timeline-preference"
}

// setTimelinePreferenceRaw fires PUT /albums/{albumId}/timeline-preference
// with the raw body (a nil cookie sends it unauthenticated).
func (e *assetEnv) setTimelinePreferenceRaw(t *testing.T, cookie *http.Cookie, albumID uuid.UUID, body string) *httptest.ResponseRecorder {
	t.Helper()
	return e.writeAlbumRaw(t, cookie, http.MethodPut, prefTarget(albumID), body)
}

// setTimelinePreference fires the PUT and decodes the 200 body.
func (e *assetEnv) setTimelinePreference(t *testing.T, cookie *http.Cookie, albumID uuid.UUID, inTimeline bool) apigen.AlbumTimelinePreferenceResponse {
	t.Helper()
	body := `{"inTimeline":` + (map[bool]string{true: "true", false: "false"})[inTimeline] + `}`
	rec := e.setTimelinePreferenceRaw(t, cookie, albumID, body)
	if rec.Code != http.StatusOK {
		t.Fatalf("PUT /albums/%s/timeline-preference status = %d, want %d (body %q)",
			albumID, rec.Code, http.StatusOK, rec.Body.String())
	}
	var pref apigen.AlbumTimelinePreferenceResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &pref); err != nil {
		t.Fatalf("decode timeline preference body %q: %v", rec.Body.String(), err)
	}
	return pref
}

// resetTimelinePreferenceRaw fires DELETE
// /albums/{albumId}/timeline-preference (a nil cookie sends it
// unauthenticated).
func (e *assetEnv) resetTimelinePreferenceRaw(t *testing.T, cookie *http.Cookie, albumID uuid.UUID) *httptest.ResponseRecorder {
	t.Helper()
	return e.writeAlbumRaw(t, cookie, http.MethodDelete, prefTarget(albumID), "")
}

// prefRowsOf counts the caller's preference rows (the upsert and
// reset's row-level truth).
func (e *assetEnv) prefRowsOf(t *testing.T, userID, albumID uuid.UUID) int {
	t.Helper()
	return e.queryInt(t, "SELECT count(*) FROM user_album_prefs WHERE user_id = $1 AND album_id = $2",
		pgUUID(userID), pgUUID(albumID))
}

// requireAlbumPrefKey asserts the 400 message-key answer:
// VALIDATION_FAILED carrying exactly the given key in details.key.
func requireAlbumPrefKey(t *testing.T, rec *httptest.ResponseRecorder, key string) {
	t.Helper()
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d (body %q)", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
	var apiErr apigen.Error
	if err := json.Unmarshal(rec.Body.Bytes(), &apiErr); err != nil {
		t.Fatalf("decode error body %q: %v", rec.Body.String(), err)
	}
	if apiErr.Code != apigen.VALIDATIONFAILED {
		t.Fatalf("code = %q, want %q", apiErr.Code, apigen.VALIDATIONFAILED)
	}
	if apiErr.Details == nil {
		t.Fatalf("details absent, want key %q", key)
	}
	if got, _ := (*apiErr.Details)["key"]; got != key {
		t.Errorf("details key = %v, want %q", got, key)
	}
}

// TestAlbumTimelinePrefsRequireSession pins the 401 of both faces.
func TestAlbumTimelinePrefsRequireSession(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	album := fixtureAlbumID("a1")
	env.insertTimelineAlbum(t, album, env.sessionUserID(t), "directory")

	for method, rec := range map[string]*httptest.ResponseRecorder{
		http.MethodPut:    env.setTimelinePreferenceRaw(t, nil, album, `{"inTimeline":false}`),
		http.MethodDelete: env.resetTimelinePreferenceRaw(t, nil, album),
	} {
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("%s timeline-preference status = %d, want %d (body %q)",
				method, rec.Code, http.StatusUnauthorized, rec.Body.String())
		}
	}
}

// TestSetAlbumTimelinePreference walks the write's whole contract:
// the exclusion toggle observable through the aggregated timeline,
// the upsert convergence, the shared album's eligibility, the wall
// and detail carrying the switch, and the validation and
// authorization doors.
func TestSetAlbumTimelinePreference(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	adminID := env.sessionUserID(t)
	graceID, graceCookie := env.createSecondUser(t, "pref-grace@example.com")

	// The caller's own directory album with one live member, and
	// the asset also sits in the caller's sync album -- the dedup
	// shape: excluded through the directory arm, still visible
	// through the sync arm (the union keeps it either way, so the
	// discriminating fixture needs an asset that ONLY sits in the
	// directory album).
	adminSync := env.syncAlbumIDOf(t, adminID)
	ownDirectory := fixtureAlbumID("b1")
	env.insertTimelineAlbum(t, ownDirectory, adminID, "directory")
	dirOnly := fixtureAssetID("b1")
	syncOnly := fixtureAssetID("b2")
	env.insertTimelineAsset(t, dirOnly, adminID, time.Date(2026, 9, 10, 12, 0, 0, 0, time.UTC), timelineAssetOpts{})
	env.insertTimelineAsset(t, syncOnly, adminID, time.Date(2026, 9, 9, 12, 0, 0, 0, time.UTC), timelineAssetOpts{})
	env.addTimelineItem(t, ownDirectory, dirOnly, adminID)
	env.addTimelineItem(t, adminSync, syncOnly, adminID)

	// Grace shares one virtual album carrying two of her assets to
	// the admin; her private one stays in her own sync stream (the
	// helper ensures the seeded account's sync album, exactly like
	// the upload path would).
	sharedVirtual := fixtureAlbumID("b3")
	env.insertTimelineAlbum(t, sharedVirtual, graceID, "virtual")
	env.shareTimelineAlbum(t, sharedVirtual, adminID, graceID)
	sharedA := fixtureAssetID("b3")
	sharedB := fixtureAssetID("b4")
	gracePrivate := fixtureAssetID("b5")
	env.insertOwnedAsset(t, sharedA, graceID, time.Date(2026, 9, 20, 12, 0, 0, 0, time.UTC), timelineAssetOpts{})
	env.insertOwnedAsset(t, sharedB, graceID, time.Date(2026, 9, 21, 12, 0, 0, 0, time.UTC), timelineAssetOpts{})
	env.insertOwnedAsset(t, gracePrivate, graceID, time.Date(2026, 9, 22, 12, 0, 0, 0, time.UTC), timelineAssetOpts{})
	env.addTimelineItem(t, sharedVirtual, sharedA, graceID)
	env.addTimelineItem(t, sharedVirtual, sharedB, graceID)

	// The admin's own virtual album: never excludable.
	ownVirtual := fixtureAlbumID("b6")
	env.insertTimelineAlbum(t, ownVirtual, adminID, "virtual")

	t.Run("ExcludesDirectoryArmOnly", func(t *testing.T) {
		// Baseline: the shared album's assets (the share is live
		// from the fixture), the directory arm's asset and the sync
		// arm's asset all feed the timeline.
		page := env.getTimeline(t, env.cookie, "")
		requireTimelineIDs(t, page.Items, sharedB, sharedA, dirOnly, syncOnly)

		pref := env.setTimelinePreference(t, env.cookie, ownDirectory, false)
		if pref.AlbumId != ownDirectory {
			t.Errorf("albumId = %s, want %s", pref.AlbumId, ownDirectory)
		}
		if pref.InTimeline {
			t.Error("inTimeline = true, want false")
		}
		if rows := env.prefRowsOf(t, adminID, ownDirectory); rows != 1 {
			t.Fatalf("preference rows = %d, want 1", rows)
		}

		// The exclusion removes exactly the directory arm's asset;
		// the sync arm's asset and the shared album stay (the sync
		// arm is exempt from the switch, and the shared arm is
		// untouched by another album's preference).
		page = env.getTimeline(t, env.cookie, "")
		requireTimelineIDs(t, page.Items, sharedB, sharedA, syncOnly)

		// The album's own view ignores the preference: browsing
		// the switched-off album still shows its items.
		scoped := env.getTimeline(t, env.cookie, "albumId="+ownDirectory.String())
		requireTimelineIDs(t, scoped.Items, dirOnly)

		// Flipping the switch back returns the asset.
		pref = env.setTimelinePreference(t, env.cookie, ownDirectory, true)
		if !pref.InTimeline {
			t.Error("inTimeline = false, want true after the flip")
		}
		if rows := env.prefRowsOf(t, adminID, ownDirectory); rows != 1 {
			t.Fatalf("preference rows after the upsert = %d, want 1 (the upsert converges)", rows)
		}
		page = env.getTimeline(t, env.cookie, "")
		requireTimelineIDs(t, page.Items, sharedB, sharedA, dirOnly, syncOnly)
	})

	t.Run("SharedAlbumIsEligible", func(t *testing.T) {
		page := env.getTimeline(t, env.cookie, "")
		requireTimelineIDs(t, page.Items, sharedB, sharedA, dirOnly, syncOnly)

		env.setTimelinePreference(t, env.cookie, sharedVirtual, false)
		page = env.getTimeline(t, env.cookie, "")
		requireTimelineIDs(t, page.Items, dirOnly, syncOnly)

		// The sharee's own timeline is untouched: the preference is
		// the CALLER's row, and Grace holds none -- her own stream
		// (all three of her assets, shared ones included: sharing
		// never removes anything from the owner's own view) reads
		// the same as before the admin's toggle.
		gracePage := env.getTimeline(t, graceCookie, "")
		requireTimelineIDs(t, gracePage.Items, gracePrivate, sharedB, sharedA)
		if rows := env.prefRowsOf(t, graceID, sharedVirtual); rows != 0 {
			t.Errorf("grace preference rows = %d, want 0 (the pref is the caller's own)", rows)
		}
	})

	t.Run("WallAndDetailCarryTheSwitch", func(t *testing.T) {
		// The subtests above leave the shared album excluded; reset
		// it so the assertions below read one known state, then
		// flip it through the wall's own value.
		env.setTimelinePreference(t, env.cookie, sharedVirtual, true)
		wall := env.getAlbumWall(t, env.cookie)
		byID := make(map[uuid.UUID]apigen.AlbumSummary, len(wall.Albums))
		for _, card := range wall.Albums {
			byID[uuid.UUID(card.Id)] = card
		}
		// The excludable cards carry the field with the stored
		// state (true -- the last write flipped it back), the own
		// virtual card omits it.
		if card := byID[ownDirectory]; card.InTimeline == nil || !*card.InTimeline {
			t.Errorf("own directory card inTimeline = %v, want true", card.InTimeline)
		}
		if card := byID[sharedVirtual]; card.InTimeline == nil || !*card.InTimeline {
			t.Errorf("shared card inTimeline = %v, want true", card.InTimeline)
		}
		if card := byID[ownVirtual]; card.InTimeline != nil {
			t.Errorf("own virtual card inTimeline = %v, want absent", card.InTimeline)
		}

		detail := env.getAlbumDetail(t, env.cookie, ownDirectory, "")
		if detail.InTimeline == nil || !*detail.InTimeline {
			t.Errorf("own directory detail inTimeline = %v, want true", detail.InTimeline)
		}
		sharedDetail := env.getAlbumDetail(t, env.cookie, sharedVirtual, "")
		if sharedDetail.InTimeline == nil || !*sharedDetail.InTimeline {
			t.Errorf("shared detail inTimeline = %v, want true", sharedDetail.InTimeline)
		}
		virtualDetail := env.getAlbumDetail(t, env.cookie, ownVirtual, "")
		if virtualDetail.InTimeline != nil {
			t.Errorf("own virtual detail inTimeline = %v, want absent", virtualDetail.InTimeline)
		}

		// And the toggle's value follows the row, not the default.
		env.setTimelinePreference(t, env.cookie, sharedVirtual, false)
		if card := byIDWall(t, env, sharedVirtual); card.InTimeline == nil || *card.InTimeline {
			t.Errorf("shared card inTimeline after exclusion = %v, want false", card.InTimeline)
		}
	})

	t.Run("MalformedBodies", func(t *testing.T) {
		for _, body := range []string{"", "not json", `{}`, `{"inTimeline":"yes"}`} {
			rec := env.setTimelinePreferenceRaw(t, env.cookie, ownDirectory, body)
			if rec.Code != http.StatusBadRequest {
				t.Errorf("PUT body %q status = %d, want %d (body %q)",
					body, rec.Code, http.StatusBadRequest, rec.Body.String())
				continue
			}
			var apiErr apigen.Error
			if err := json.Unmarshal(rec.Body.Bytes(), &apiErr); err != nil {
				t.Fatalf("decode error body %q: %v", rec.Body.String(), err)
			}
			if apiErr.Code != apigen.VALIDATIONFAILED {
				t.Errorf("PUT body %q code = %q, want %q", body, apiErr.Code, apigen.VALIDATIONFAILED)
			}
		}
		if rows := env.prefRowsOf(t, adminID, ownDirectory); rows != 1 {
			t.Errorf("preference rows after malformed writes = %d, want 1 (nothing landed)", rows)
		}
	})

	t.Run("SyncAndOwnVirtualRefused", func(t *testing.T) {
		for _, album := range []uuid.UUID{adminSync, ownVirtual} {
			rec := env.setTimelinePreferenceRaw(t, env.cookie, album, `{"inTimeline":false}`)
			requireAlbumPrefKey(t, rec, "album_pref_not_excludable")
			if rows := env.prefRowsOf(t, adminID, album); rows != 0 {
				t.Errorf("preference rows on %s = %d, want 0", album, rows)
			}
			// The reset refuses the same kinds, the same shape.
			reset := env.resetTimelinePreferenceRaw(t, env.cookie, album)
			requireAlbumPrefKey(t, reset, "album_pref_not_excludable")
		}
	})

	t.Run("NotFoundSingleShape", func(t *testing.T) {
		unknown := fixtureAlbumID("ff")
		foreignUnshared := fixtureAlbumID("b7")
		env.insertTimelineAlbum(t, foreignUnshared, graceID, "virtual")

		for _, album := range []uuid.UUID{unknown, foreignUnshared} {
			put := env.setTimelinePreferenceRaw(t, env.cookie, album, `{"inTimeline":false}`)
			requireAlbumNotFound(t, put)
			del := env.resetTimelinePreferenceRaw(t, env.cookie, album)
			requireAlbumNotFound(t, del)
		}
		// Byte-identical to the unknown id's body.
		unknownPut := env.setTimelinePreferenceRaw(t, env.cookie, unknown, `{"inTimeline":false}`)
		foreignPut := env.setTimelinePreferenceRaw(t, env.cookie, foreignUnshared, `{"inTimeline":false}`)
		if unknownPut.Body.String() != foreignPut.Body.String() {
			t.Errorf("foreign album 404 body %q differs from the unknown id's %q",
				foreignPut.Body.String(), unknownPut.Body.String())
		}
	})

	t.Run("NoEventsForSelfAffectingWrites", func(t *testing.T) {
		// A full toggle cycle on the bus: the preference changes
		// publish nothing -- no album.* and no asset.* (the
		// timeline's contents shift, but the event discipline
		// covers writes other sessions refetch, and only the
		// caller's own view changed).
		env.setTimelinePreference(t, env.cookie, sharedVirtual, false)
		env.setTimelinePreference(t, env.cookie, sharedVirtual, false)
		env.setTimelinePreference(t, env.cookie, sharedVirtual, true)
		reset := env.resetTimelinePreferenceRaw(t, env.cookie, sharedVirtual)
		if reset.Code != http.StatusNoContent {
			t.Fatalf("DELETE status = %d, want %d (body %q)",
				reset.Code, http.StatusNoContent, reset.Body.String())
		}
		for _, name := range []string{
			"album.created", "album.updated", "album.deleted",
			"album.items-changed", "album.shared", "album.unshared",
			"asset.updated", "asset.trashed", "asset.restored",
		} {
			if events := busEventsOf(t, env.bus, name); len(events) != 0 {
				t.Errorf("%s fired %d events for preference writes, want 0", name, len(events))
			}
		}
	})
}

// TestResetAlbumTimelinePreference pins the reset: the row goes, the
// album returns to the default of feeding the timeline, and the
// delete is idempotent whether or not a row exists.
func TestResetAlbumTimelinePreference(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	adminID := env.sessionUserID(t)

	directory := fixtureAlbumID("c1")
	env.insertTimelineAlbum(t, directory, adminID, "directory")
	member := fixtureAssetID("c1")
	env.insertTimelineAsset(t, member, adminID, time.Date(2026, 9, 15, 12, 0, 0, 0, time.UTC), timelineAssetOpts{})
	env.addTimelineItem(t, directory, member, adminID)

	// Default state: no row, the member feeds the timeline.
	page := env.getTimeline(t, env.cookie, "")
	requireTimelineIDs(t, page.Items, member)

	env.setTimelinePreference(t, env.cookie, directory, false)
	page = env.getTimeline(t, env.cookie, "")
	requireTimelineIDs(t, page.Items)

	rec := env.resetTimelinePreferenceRaw(t, env.cookie, directory)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("DELETE status = %d, want %d (body %q)", rec.Code, http.StatusNoContent, rec.Body.String())
	}
	if rows := env.prefRowsOf(t, adminID, directory); rows != 0 {
		t.Fatalf("preference rows after reset = %d, want 0", rows)
	}
	page = env.getTimeline(t, env.cookie, "")
	requireTimelineIDs(t, page.Items, member)

	// Idempotent: resetting an album with no row still answers 204.
	rec = env.resetTimelinePreferenceRaw(t, env.cookie, directory)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("second DELETE status = %d, want %d (body %q)", rec.Code, http.StatusNoContent, rec.Body.String())
	}

	// A sharee resets their OWN preference of a shared album: the
	// row is theirs, so the departure semantics never apply here.
	graceID, graceCookie := env.createSecondUser(t, "reset-grace@example.com")
	shared := fixtureAlbumID("c2")
	env.insertTimelineAlbum(t, shared, graceID, "virtual")
	env.shareTimelineAlbum(t, shared, adminID, graceID)

	rec = env.resetTimelinePreferenceRaw(t, env.cookie, shared)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("reset of a never-set shared album status = %d, want %d (body %q)",
			rec.Code, http.StatusNoContent, rec.Body.String())
	}
	env.setTimelinePreference(t, env.cookie, shared, false)
	if rows := env.prefRowsOf(t, adminID, shared); rows != 1 {
		t.Fatalf("preference rows on the shared album = %d, want 1", rows)
	}
	if rows := env.prefRowsOf(t, graceID, shared); rows != 0 {
		t.Fatalf("grace preference rows = %d, want 0 (only the caller's row was written)", rows)
	}

	// The ALBUM'S OWNER holds no preference scope either: their own
	// virtual album never feeds their own timeline, so Grace's own
	// write on it answers the same 400 as the admin's own virtual.
	own := env.setTimelinePreferenceRaw(t, graceCookie, shared, `{"inTimeline":false}`)
	requireAlbumPrefKey(t, own, "album_pref_not_excludable")
	if rows := env.prefRowsOf(t, graceID, shared); rows != 0 {
		t.Fatalf("grace preference rows after her own refusal = %d, want 0", rows)
	}
}

// byIDWall fetches the wall and answers one album's card.
func byIDWall(t *testing.T, env *assetEnv, albumID uuid.UUID) apigen.AlbumSummary {
	t.Helper()
	wall := env.getAlbumWall(t, env.cookie)
	for _, card := range wall.Albums {
		if uuid.UUID(card.Id) == albumID {
			return card
		}
	}
	t.Fatalf("album %s carries no wall card (wall %v)", albumID, wall.Albums)
	return apigen.AlbumSummary{}
}
