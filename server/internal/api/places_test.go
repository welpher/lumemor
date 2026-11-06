// Places surface tests (M3d Task 1): the caller's city grouping and
// the admin re-resolution trigger. The read discipline mirrors the
// people surface's: a place lists when it holds at least one
// caller-visible live asset, counts are honest per caller (a sharee
// counts only what the album union lets them see), covers are the
// place's own visible members, and names resolve through the
// requested locale's place_names row with the canonical name as the
// fallback. The trigger mirrors the model rerun's idempotence: an
// in-flight run answers 200, anything else starts a fresh run
// through the queue seam.
package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	apigen "github.com/welpher/lumemor/server/internal/api/gen"
	dbgen "github.com/welpher/lumemor/server/internal/db/gen"
)

// The fixture places: Paris and Beijing carry locale names, Highland
// Falls carries none (the canonical fallback arm).
const (
	placeParisID    = 2988507
	placeBeijingID  = 1816670
	placeHighlandID = 5132638
)

// insertPlacesFixture seeds the places and place_names rows the read
// tests join: the canonical names of three cities, the zh-Hans rows
// of two (the third exercises the canonical fallback).
func (e *assetEnv) insertPlacesFixture(t *testing.T) {
	t.Helper()
	e.exec(t, `
		INSERT INTO places (id, name, country_code, admin1_code, timezone, lat, lon) VALUES
			($1, 'Paris', 'FR', 'FR.11', 'Europe/Paris', 48.85341, 2.3488),
			($2, 'Beijing', 'CN', 'CN.22', 'Asia/Shanghai', 39.9075, 116.39723),
			($3, 'Highland Falls', 'US', 'US.NY', 'America/New_York', 41.3675, -73.9705)`,
		placeParisID, placeBeijingID, placeHighlandID)
	e.exec(t, `
		INSERT INTO place_names (geoname_id, locale, name) VALUES
			($1, 'zh-Hans', '巴黎'),
			($2, 'zh-Hans', '北京')`,
		placeParisID, placeBeijingID)
}

// placeAssetID builds a deterministic fixture asset id of the places
// family, in a namespace disjoint from every other fixture family.
func placeAssetID(suffix string) uuid.UUID {
	return uuid.MustParse("60000000-0000-0000-0000-0000000000" + suffix)
}

// listPlaces fires GET /api/v1/places (with the optional locale).
func listPlaces(router http.Handler, cookie *http.Cookie, locale string) *httptest.ResponseRecorder {
	target := "/api/v1/places"
	if locale != "" {
		target += "?locale=" + locale
	}
	return authedRequest(router, http.MethodGet, target, cookie)
}

// decodePlaces decodes a listing response.
func decodePlaces(t *testing.T, rec *httptest.ResponseRecorder) []apigen.PlaceSummary {
	t.Helper()
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d (body %q)", rec.Code, http.StatusOK, rec.Body.String())
	}
	var places []apigen.PlaceSummary
	if err := json.Unmarshal(rec.Body.Bytes(), &places); err != nil {
		t.Fatalf("decode places %q: %v", rec.Body.String(), err)
	}
	return places
}

// requirePlace asserts one listing entry's shape.
func requirePlace(t *testing.T, got apigen.PlaceSummary, placeID int32, name, country string, assetCount int, coverLen int) {
	t.Helper()
	if got.PlaceId != placeID || got.Name != name || got.Country != country || got.AssetCount != assetCount {
		t.Errorf("place = {%d %q %q count %d}, want {%d %q %q count %d}",
			got.PlaceId, got.Name, got.Country, got.AssetCount, placeID, name, country, assetCount)
	}
	if len(got.RepresentativeCoverAssetIds) != coverLen {
		t.Errorf("place %d covers = %v, want %d ids", placeID, got.RepresentativeCoverAssetIds, coverLen)
	}
}

// TestPlacesCityGrouping pins the caller's city grouping: the locale
// join (the requested locale's row, the canonical fallback), the
// honest count ordering (asset count descending, the place id as the
// tiebreak), the wall's derived covers (the four most recent visible
// live members, newest first), the unauthenticated refusal, and the
// out-of-set locale's 400.
func TestPlacesCityGrouping(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	adminID := env.sessionUserID(t)
	env.insertPlacesFixture(t)

	// The admin's Paris set: three live (jul 1, 2, 3), one hidden
	// (never counts, never covers), one trashed, one Live Photo
	// component -- plus two Beijing assets and one Highland Falls.
	aParis1 := placeAssetID("01")
	aParis2 := placeAssetID("02")
	aParis3 := placeAssetID("03")
	aParisHidden := placeAssetID("04")
	aParisTrash := placeAssetID("05")
	aParisComp := placeAssetID("06")
	aBj1 := placeAssetID("07")
	aBj2 := placeAssetID("08")
	aHf := placeAssetID("09")
	env.insertOwnedAsset(t, aParis1, adminID, time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC), timelineAssetOpts{})
	env.insertOwnedAsset(t, aParis2, adminID, time.Date(2026, 7, 2, 12, 0, 0, 0, time.UTC), timelineAssetOpts{})
	env.insertOwnedAsset(t, aParis3, adminID, time.Date(2026, 7, 3, 12, 0, 0, 0, time.UTC), timelineAssetOpts{})
	env.insertOwnedAsset(t, aParisHidden, adminID, time.Date(2026, 7, 4, 12, 0, 0, 0, time.UTC), timelineAssetOpts{isHidden: true})
	env.insertOwnedAsset(t, aParisTrash, adminID, time.Date(2026, 7, 5, 12, 0, 0, 0, time.UTC), timelineAssetOpts{deleted: true})
	env.insertOwnedAsset(t, aParisComp, adminID, time.Date(2026, 7, 6, 12, 0, 0, 0, time.UTC), timelineAssetOpts{isComponent: true})
	env.insertOwnedAsset(t, aBj1, adminID, time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC), timelineAssetOpts{})
	env.insertOwnedAsset(t, aBj2, adminID, time.Date(2026, 8, 2, 12, 0, 0, 0, time.UTC), timelineAssetOpts{})
	env.insertOwnedAsset(t, aHf, adminID, time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC), timelineAssetOpts{})
	env.exec(t, "UPDATE assets SET place_id = $1 WHERE id = ANY($2)",
		placeParisID, []uuid.UUID{aParis1, aParis2, aParis3, aParisHidden, aParisTrash, aParisComp})
	env.exec(t, "UPDATE assets SET place_id = $1 WHERE id = ANY($2)",
		placeBeijingID, []uuid.UUID{aBj1, aBj2})
	env.exec(t, "UPDATE assets SET place_id = $1 WHERE id = $2", placeHighlandID, aHf)

	t.Run("EnLocaleDefault", func(t *testing.T) {
		places := decodePlaces(t, listPlaces(env.router, env.cookie, ""))
		if len(places) != 3 {
			t.Fatalf("places = %d entries, want 3 (Paris 3, Beijing 2, Highland Falls 1)", len(places))
		}
		requirePlace(t, places[0], placeParisID, "Paris", "FR", 3, 3)
		requirePlace(t, places[1], placeBeijingID, "Beijing", "CN", 2, 2)
		// Highland Falls carries no zh-Hans row anywhere; in en it is
		// its canonical self.
		requirePlace(t, places[2], placeHighlandID, "Highland Falls", "US", 1, 1)
		// The covers are the newest live members first: jul 3, jul 2,
		// jul 1 -- the hidden, trashed and component rows never cover.
		want := []uuid.UUID{aParis3, aParis2, aParis1}
		for i, id := range want {
			if places[0].RepresentativeCoverAssetIds[i] != id {
				t.Errorf("Paris cover[%d] = %s, want %s", i, places[0].RepresentativeCoverAssetIds[i], id)
			}
		}
	})

	t.Run("ZhHansLocaleJoin", func(t *testing.T) {
		places := decodePlaces(t, listPlaces(env.router, env.cookie, "zh-Hans"))
		if len(places) != 3 {
			t.Fatalf("places = %d entries, want 3", len(places))
		}
		requirePlace(t, places[0], placeParisID, "巴黎", "FR", 3, 3)
		requirePlace(t, places[1], placeBeijingID, "北京", "CN", 2, 2)
		// No zh-Hans row: the canonical name answers the fallback.
		requirePlace(t, places[2], placeHighlandID, "Highland Falls", "US", 1, 1)
	})

	t.Run("UnknownLocaleRejected", func(t *testing.T) {
		rec := listPlaces(env.router, env.cookie, "fr")
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400 (body %q)", rec.Code, rec.Body.String())
		}
		apiErr, fields := decodeError(t, rec)
		if apiErr.Code != apigen.VALIDATIONFAILED {
			t.Errorf("code = %q, want VALIDATION_FAILED", apiErr.Code)
		}
		if len(fields) != 1 || fields[0] != "locale" {
			t.Errorf("details.fields = %v, want [locale]", fields)
		}
	})

	t.Run("UnauthenticatedRefused", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/places", nil)
		rec := httptest.NewRecorder()
		env.router.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want 401", rec.Code)
		}
	})

	t.Run("EmptyForNothingResolved", func(t *testing.T) {
		// A caller whose assets carry no place at all sees the empty
		// list, not an error.
		_, plainCookie := env.createSecondUser(t, "places-plain@example.com")
		places := decodePlaces(t, listPlaces(env.router, plainCookie, ""))
		if len(places) != 0 {
			t.Fatalf("places = %d entries, want 0", len(places))
		}
	})
}

// TestPlacesShareeVisibility pins the union predicate's reach into
// the city grouping: a sharee counts (and covers) only the assets
// the album union lets them see -- the shared album's members and
// their own -- never the owner's unshared assets; a revoked share
// retracts them.
func TestPlacesShareeVisibility(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	adminID := env.sessionUserID(t)
	graceID, graceCookie := env.createSecondUser(t, "places-shared-grace@example.com")
	env.insertPlacesFixture(t)

	// The admin's album shared to Grace, carrying a1 and aTrash; a2
	// stays unshared.
	sharedAlbum := fixtureAlbumID("e1")
	env.insertTimelineAlbum(t, sharedAlbum, adminID, "virtual")
	env.shareTimelineAlbum(t, sharedAlbum, graceID, adminID)
	a1 := placeAssetID("11")
	a2 := placeAssetID("12")
	aTrash := placeAssetID("13")
	env.insertOwnedAsset(t, a1, adminID, time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC), timelineAssetOpts{})
	env.insertOwnedAsset(t, a2, adminID, time.Date(2026, 7, 2, 12, 0, 0, 0, time.UTC), timelineAssetOpts{})
	env.insertOwnedAsset(t, aTrash, adminID, time.Date(2026, 7, 3, 12, 0, 0, 0, time.UTC), timelineAssetOpts{deleted: true})
	env.addTimelineItem(t, sharedAlbum, a1, adminID)
	env.addTimelineItem(t, sharedAlbum, aTrash, adminID)
	env.exec(t, "UPDATE assets SET place_id = $1 WHERE id = ANY($2)",
		placeParisID, []uuid.UUID{a1, a2, aTrash})

	// Grace's own Beijing asset.
	g1 := placeAssetID("14")
	env.insertOwnedAsset(t, g1, graceID, time.Date(2026, 7, 4, 12, 0, 0, 0, time.UTC), timelineAssetOpts{})
	env.exec(t, "UPDATE assets SET place_id = $1 WHERE id = $2", placeBeijingID, g1)

	places := decodePlaces(t, listPlaces(env.router, graceCookie, "zh-Hans"))
	if len(places) != 2 {
		t.Fatalf("places = %d entries, want 2 (Paris 1 shared-live, Beijing 1 own)", len(places))
	}
	// The tie on count 1 breaks on the smaller place id first.
	requirePlace(t, places[0], placeBeijingID, "北京", "CN", 1, 1)
	if places[0].RepresentativeCoverAssetIds[0] != g1 {
		t.Errorf("Beijing cover = %s, want Grace's own asset %s", places[0].RepresentativeCoverAssetIds[0], g1)
	}
	requirePlace(t, places[1], placeParisID, "巴黎", "FR", 1, 1)
	if places[1].RepresentativeCoverAssetIds[0] != a1 {
		t.Errorf("Paris cover = %s, want the shared live member %s (the trashed one never covers)", places[1].RepresentativeCoverAssetIds[0], a1)
	}

	// A revoked share retracts the shared counts and covers.
	env.exec(t, "DELETE FROM album_shares WHERE album_id = $1 AND user_id = $2",
		pgUUID(sharedAlbum), pgUUID(graceID))
	places = decodePlaces(t, listPlaces(env.router, graceCookie, ""))
	if len(places) != 1 {
		t.Fatalf("places after revoke = %d entries, want 1 (Beijing only)", len(places))
	}
	requirePlace(t, places[0], placeBeijingID, "Beijing", "CN", 1, 1)
}

// TestPlacesCoversCapAtFour pins the cover window: a place with more
// than four visible members covers exactly the four most recent.
func TestPlacesCoversCapAtFour(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	adminID := env.sessionUserID(t)
	env.insertPlacesFixture(t)

	ids := make([]uuid.UUID, 0, 6)
	for i := 0; i < 6; i++ {
		id := placeAssetID(fmt.Sprintf("2%d", i))
		ids = append(ids, id)
		env.insertOwnedAsset(t, id, adminID,
			time.Date(2026, 7, 1, 12, 0, i, 0, time.UTC), timelineAssetOpts{})
	}
	env.exec(t, "UPDATE assets SET place_id = $1 WHERE id = ANY($2)", placeParisID, ids)

	places := decodePlaces(t, listPlaces(env.router, env.cookie, ""))
	if len(places) != 1 {
		t.Fatalf("places = %d entries, want 1", len(places))
	}
	requirePlace(t, places[0], placeParisID, "Paris", "FR", 6, 4)
	want := []uuid.UUID{ids[5], ids[4], ids[3], ids[2]} // newest first
	for i, id := range want {
		if places[0].RepresentativeCoverAssetIds[i] != id {
			t.Errorf("cover[%d] = %s, want %s", i, places[0].RepresentativeCoverAssetIds[i], id)
		}
	}
}

// TestReresolvePlaces pins the admin re-resolution trigger: the role
// guards (401 without a session, 403 for a plain user), the 202 of
// the first call (the run enqueued through the seam), the 200 of the
// in-flight idempotence (no second enqueue), and the 500 when no
// queue is wired or the enqueue fails.
func TestReresolvePlaces(t *testing.T) {
	pool := newSetupEnv(t)
	stub := &stubPlacesReruns{id: uuid.MustParse("0192f7a1-4d5c-8e9f-a0b1-2c3d4e5f6071")}
	router := NewRouter(&Deps{Pool: pool, Version: testVersion, PlacesReruns: stub})

	rec := postAdmin(t, router, createAdminBody(t, "ada@example.com", testPassword, nil))
	if rec.Code != http.StatusCreated {
		t.Fatalf("create admin: status = %d (body %q)", rec.Code, rec.Body.String())
	}
	adminCookie, _ := loginCookie(t, router, "ada@example.com", testPassword, "places-rerun-tests")

	// Unauthenticated: 401.
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/v1/admin/places/reresolve", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated status = %d, want 401", rec.Code)
	}

	// The plain user: 403 (the admin-role guard).
	env := &assetEnv{pool: pool}
	_, plainCookie := env.createSecondUser(t, "places-plain@example.com")
	rec = authedRequest(router, http.MethodPost, "/api/v1/admin/places/reresolve", plainCookie)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("plain user status = %d, want 403", rec.Code)
	}

	// The first call: 202 with the fresh run's identifier.
	rec = authedRequest(router, http.MethodPost, "/api/v1/admin/places/reresolve", adminCookie)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("first call status = %d, want 202 (body %q)", rec.Code, rec.Body.String())
	}
	var started apigen.PlacesRerunResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &started); err != nil {
		t.Fatalf("decode response %q: %v", rec.Body.String(), err)
	}
	if started.JobRunId != stub.id {
		t.Errorf("jobRunId = %s, want %s", started.JobRunId, stub.id)
	}
	if stub.calls != 1 {
		t.Errorf("enqueue calls = %d, want 1", stub.calls)
	}

	// An in-flight run answers 200 with its identifier and enqueues
	// nothing: the seeded row is the join point.
	inFlight := uuid.MustParse("60000000-0000-0000-0002-000000000002")
	if _, err := dbgen.New(pool).InsertJobRun(context.Background(), dbgen.InsertJobRunParams{
		ID:    pgUUID(inFlight),
		Kind:  "places_rerun",
		Total: 3,
	}); err != nil {
		t.Fatalf("seed in-flight run: %v", err)
	}
	rec = authedRequest(router, http.MethodPost, "/api/v1/admin/places/reresolve", adminCookie)
	if rec.Code != http.StatusOK {
		t.Fatalf("in-flight status = %d, want 200 (body %q)", rec.Code, rec.Body.String())
	}
	var joined apigen.PlacesRerunResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &joined); err != nil {
		t.Fatalf("decode response %q: %v", rec.Body.String(), err)
	}
	if joined.JobRunId != inFlight {
		t.Errorf("jobRunId = %s, want the in-flight run %s", joined.JobRunId, inFlight)
	}
	if stub.calls != 1 {
		t.Errorf("enqueue calls after the in-flight join = %d, want 1 (no second enqueue)", stub.calls)
	}

	// A wired-but-failing enqueue maps to the 500 envelope.
	stub.err = errPlacesRerunRefused
	rec = authedRequest(router, http.MethodPost, "/api/v1/admin/places/reresolve", adminCookie)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("failed enqueue status = %d, want 500", rec.Code)
	}
	if code, _ := decodeError(t, rec); code != apigen.INTERNAL {
		t.Errorf("code = %q, want INTERNAL", code)
	}
}

// errPlacesRerunRefused is the stub's failure token.
var errPlacesRerunRefused = errors.New("the place resolver is not wired")

// stubPlacesReruns is the enqueue seam under test: it counts the
// calls, answers a fixed run identifier, and can be told to fail.
type stubPlacesReruns struct {
	calls int
	id    uuid.UUID
	err   error
}

// EnqueuePlacesRerun implements PlacesRerunEnqueuer.
func (s *stubPlacesReruns) EnqueuePlacesRerun(ctx context.Context) (uuid.UUID, error) {
	s.calls++
	if s.err != nil {
		return uuid.Nil, s.err
	}
	return s.id, nil
}
