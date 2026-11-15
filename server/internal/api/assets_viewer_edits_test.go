// The viewer's edit arms: the manual capture time (the instant and
// its display zone land together, the response serves the applied
// pair, the idempotent repeat stays silent, the zone-less value is
// refused) and the manual location (the coordinates land beside the
// resolved place, the pair rides together, the camera's EXIF GPS is
// untouched). The place resolution runs against a real grid built
// over one centroid, so the nearest-city answer is a pinned one.
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
	"github.com/welpher/lumemor/server/internal/geodata"
)

// newAssetEnvWithPlaces mirrors newAssetEnv but wires the place grid:
// the manual-location edit resolves its city against it.
func newAssetEnvWithPlaces(t *testing.T, uploadLimit int64, centroids []geodata.Centroid) *assetEnv {
	t.Helper()
	env := newAssetEnv(t, uploadLimit)
	env.router = NewRouter(&Deps{
		Pool:             env.pool,
		Version:          testVersion,
		Events:           env.bus,
		DataDir:          env.dataDir,
		UploadLimitBytes: uploadLimit,
		Places:           geodata.NewIndex(centroids),
	})
	return env
}

// TestPatchAssetTakenAt pins the capture-time arm: the instant and
// the display zone land together (the offset in the value IS the
// zone), the response serves the applied pair exactly as the GET
// serves it, the row carries both, one event covers the change, and
// the idempotent repeat stays silent.
func TestPatchAssetTakenAt(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	id := fixtureAssetID("a1")
	env.insertOwnedAsset(t, id, env.sessionUserID(t),
		time.Date(2026, 7, 10, 10, 0, 0, 0, time.UTC), timelineAssetOpts{})

	detail := env.patchAsset(t, env.cookie, id, map[string]any{"takenAt": "2026-08-12T18:42:00+08:00"})
	if !detail.TakenAt.Equal(time.Date(2026, 8, 12, 10, 42, 0, 0, time.UTC)) {
		t.Errorf("patched takenAt = %v, want the offset's instant (10:42 UTC)", detail.TakenAt)
	}
	if detail.TakenAtZone == nil || *detail.TakenAtZone != "+08:00" {
		t.Errorf("patched takenAtZone = %v, want +08:00", detail.TakenAtZone)
	}
	requireAssetUpdatedEvents(t, env, 1, id, "image")

	if got := env.queryText(t, "SELECT COALESCE(taken_at_zone, '') FROM assets WHERE id = $1", pgUUID(id)); got != "+08:00" {
		t.Errorf("row taken_at_zone = %q, want +08:00", got)
	}

	// The idempotent repeat: the same instant and zone answers the
	// same success and publishes nothing further.
	env.patchAsset(t, env.cookie, id, map[string]any{"takenAt": "2026-08-12T18:42:00+08:00"})
	requireAssetUpdatedEvents(t, env, 1, id, "image")

	// A different wall time and offset (18:42+08 -> 18:42+09) is a
	// different instant carrying the new zone.
	detail = env.patchAsset(t, env.cookie, id, map[string]any{"takenAt": "2026-08-12T18:42:00+09:00"})
	if !detail.TakenAt.Equal(time.Date(2026, 8, 12, 9, 42, 0, 0, time.UTC)) {
		t.Errorf("re-zoned takenAt = %v, want the new offset's instant", detail.TakenAt)
	}
	if detail.TakenAtZone == nil || *detail.TakenAtZone != "+09:00" {
		t.Errorf("re-zoned takenAtZone = %v, want +09:00", detail.TakenAtZone)
	}
	requireAssetUpdatedEvents(t, env, 2, id, "image")

	// The zone-only arm: 17:42+08:00 is the SAME instant as the
	// stored 09:42 UTC with the other offset -- the zone is a served
	// field compared on its own, so this repeat is still a change.
	detail = env.patchAsset(t, env.cookie, id, map[string]any{"takenAt": "2026-08-12T17:42:00+08:00"})
	if !detail.TakenAt.Equal(time.Date(2026, 8, 12, 9, 42, 0, 0, time.UTC)) {
		t.Errorf("zone-only takenAt = %v, want the unchanged instant", detail.TakenAt)
	}
	if detail.TakenAtZone == nil || *detail.TakenAtZone != "+08:00" {
		t.Errorf("zone-only takenAtZone = %v, want +08:00", detail.TakenAtZone)
	}
	requireAssetUpdatedEvents(t, env, 3, id, "image")
}

// TestPatchAssetTakenAtValidation pins the zone door: the contract
// refuses a timestamp without an offset designator, and the refusal
// is the decode's own (the RFC 3339 binding), before any database
// access.
func TestPatchAssetTakenAtValidation(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	id := fixtureAssetID("a2")
	env.insertOwnedAsset(t, id, env.sessionUserID(t),
		time.Date(2026, 7, 11, 10, 0, 0, 0, time.UTC), timelineAssetOpts{})

	for _, body := range []string{
		`{"takenAt": "2026-08-12T18:42:00"}`,
		`{"takenAt": "2026-08-12 18:42:00+08:00"}`,
		`{"takenAt": "not-a-time"}`,
	} {
		rec := env.patchAssetRaw(t, env.cookie, id, []byte(body))
		requirePatchValidation(t, rec, "body")
	}
	// Nothing refused reached the row.
	if got := env.queryText(t, "SELECT COALESCE(taken_at_zone, '') FROM assets WHERE id = $1", pgUUID(id)); got != "" {
		t.Errorf("row taken_at_zone after the refusals = %q, want untouched NULL", got)
	}
	if events := busEventsOf(t, env.bus, "asset.updated"); len(events) != 0 {
		t.Errorf("asset.updated events = %d, want 0", len(events))
	}
}

// TestPatchAssetLocation pins the manual-location arm: the pair
// lands together, the response serves the coordinates (and the
// resolved place at the grid's nearest city), the row carries both,
// the camera's EXIF GPS keeps the file's own reading, and the
// idempotent repeat stays silent.
func TestPatchAssetLocation(t *testing.T) {
	env := newAssetEnvWithPlaces(t, testUploadLimit, []geodata.Centroid{
		{ID: 2988507, Lat: 48.8566, Lon: 2.3522},
	})
	// The places row the resolution writes through the FK: the grid
	// answers the id, the table must hold it.
	env.exec(t, `INSERT INTO places (id, name, country_code, admin1_code, timezone, lat, lon)
		VALUES (2988507, 'Paris', 'FR', '11', 'Europe/Paris', 48.8566, 2.3522)`)
	id := fixtureAssetID("b1")
	env.insertOwnedAsset(t, id, env.sessionUserID(t),
		time.Date(2026, 7, 12, 10, 0, 0, 0, time.UTC), timelineAssetOpts{})

	// The EXIF GPS the file recorded (the sidecar's own pair), which
	// the edit must not touch.
	env.exec(t, "INSERT INTO exif (asset_id, exif, lat, lon, alt) VALUES ($1, '{}', 48.8500, 2.3000, 33)",
		pgUUID(id))

	detail := env.patchAsset(t, env.cookie, id, map[string]any{"latitude": 48.8584, "longitude": 2.2945})
	if detail.Location == nil {
		t.Fatal("patched detail location = null, want the written pair")
	}
	if detail.Location.Lat != 48.8584 || detail.Location.Lon != 2.2945 {
		t.Errorf("patched detail location = %v, want (48.8584, 2.2945)", detail.Location)
	}
	// The resolved place follows the pin: the grid's Paris.
	if detail.Place == nil || detail.Place.PlaceId != 2988507 {
		t.Errorf("patched detail place = %v, want the resolved Paris (2988507)", detail.Place)
	}
	requireAssetUpdatedEvents(t, env, 1, id, "image")

	// The row carries the manual pair and the resolution; the EXIF
	// sidecar keeps the camera's own reading.
	row := env.assetRowOf(t, id)
	if !row.LocationLat.Valid || row.LocationLat.Float64 != 48.8584 {
		t.Errorf("row location_lat = %v, want 48.8584", row.LocationLat)
	}
	if got := env.queryText(t, "SELECT alt FROM exif WHERE asset_id = $1", pgUUID(id)); got != "33" {
		t.Errorf("exif alt = %q, want the untouched 33", got)
	}
	exifLat := env.queryText(t, "SELECT lat::text FROM exif WHERE asset_id = $1", pgUUID(id))
	if exifLat != "48.85" {
		t.Errorf("exif lat = %q, want the camera's own 48.85", exifLat)
	}

	// The idempotent repeat: the same pair and place answers the same
	// success and publishes nothing further.
	env.patchAsset(t, env.cookie, id, map[string]any{"latitude": 48.8584, "longitude": 2.2945})
	requireAssetUpdatedEvents(t, env, 1, id, "image")

	// The pair rides together: one half alone answers 400.
	rec := env.patchAssetRaw(t, env.cookie, id, []byte(`{"latitude": 48.8584}`))
	requirePatchValidation(t, rec, "latitude")
}

// TestPatchAssetLocationBeyondPlaces pins the honest absence: a pin
// beyond every city resolves to no place (the coordinates still
// stand, the place face goes null), and a deployment without the
// grid keeps the resolved place whatever it held.
func TestPatchAssetLocationBeyondPlaces(t *testing.T) {
	env := newAssetEnvWithPlaces(t, testUploadLimit, []geodata.Centroid{
		{ID: 2988507, Lat: 48.8566, Lon: 2.3522},
	})
	id := fixtureAssetID("b2")
	env.insertOwnedAsset(t, id, env.sessionUserID(t),
		time.Date(2026, 7, 13, 10, 0, 0, 0, time.UTC), timelineAssetOpts{})
	env.exec(t, "INSERT INTO exif (asset_id, exif, lat, lon) VALUES ($1, '{}', 48.85, 2.30)", pgUUID(id))
	env.exec(t, `INSERT INTO places (id, name, country_code, admin1_code, timezone, lat, lon)
		VALUES (2988507, 'Paris', 'FR', '11', 'Europe/Paris', 48.8566, 2.3522)`)

	// Mid-ocean: beyond the threshold, the coordinates land and the
	// place goes null.
	detail := env.patchAsset(t, env.cookie, id, map[string]any{"latitude": 0.0, "longitude": -160.0})
	if detail.Location == nil || detail.Location.Lat != 0 || detail.Location.Lon != -160 {
		t.Errorf("patched detail location = %v, want the written pair", detail.Location)
	}
	if detail.Place != nil {
		t.Errorf("patched detail place = %v, want null (beyond every city)", detail.Place)
	}

	// A grid-less deployment keeps the resolution: the place face
	// stays what the row carried (here the Paris join the previous
	// state set) while the coordinates update.
	env.exec(t, "UPDATE assets SET place_id = 2988507, updated_at = now() WHERE id = $1", pgUUID(id))
	plainEnv := newAssetEnv(t, testUploadLimit)
	plainEnv.router = NewRouter(&Deps{
		Pool:             plainEnv.pool,
		Version:          testVersion,
		Events:           plainEnv.bus,
		DataDir:          plainEnv.dataDir,
		UploadLimitBytes: testUploadLimit,
	})
	plainID := fixtureAssetID("b3")
	plainEnv.insertOwnedAsset(t, plainID, plainEnv.sessionUserID(t),
		time.Date(2026, 7, 14, 10, 0, 0, 0, time.UTC), timelineAssetOpts{})
	plainEnv.exec(t, `INSERT INTO places (id, name, country_code, admin1_code, timezone, lat, lon)
		VALUES (2988507, 'Paris', 'FR', '11', 'Europe/Paris', 48.8566, 2.3522)`)
	plainEnv.exec(t, "UPDATE assets SET place_id = 2988507, updated_at = now() WHERE id = $1", pgUUID(plainID))
	plainDetail := plainEnv.patchAsset(t, plainEnv.cookie, plainID,
		map[string]any{"latitude": 36.0671, "longitude": 120.3826})
	if plainDetail.Location == nil || plainDetail.Location.Lat != 36.0671 {
		t.Errorf("grid-less detail location = %v, want the written pair", plainDetail.Location)
	}
	if plainDetail.Place == nil || plainDetail.Place.PlaceId != 2988507 {
		t.Errorf("grid-less detail place = %v, want the kept 2988507", plainDetail.Place)
	}
}

// TestPatchAssetDetailPathFace pins the admin-only path face: the
// administrator's detail carries the storage-relative machine path,
// a plain user's detail reads null and its panel renders no such
// row -- the instance's internal layout is machine information.
func TestPatchAssetDetailPathFace(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	adminID := env.sessionUserID(t)
	id := fixtureAssetID("c1")
	env.insertOwnedAsset(t, id, adminID,
		time.Date(2026, 7, 15, 10, 0, 0, 0, time.UTC), timelineAssetOpts{})

	detail := env.patchAsset(t, env.cookie, id, map[string]any{"description": "path check"})
	if detail.Path == nil {
		t.Fatal("administrator detail path = null, want the storage-relative path")
	}
	if *detail.Path == "" {
		t.Error("administrator detail path = empty, want the row's file_path")
	}

	// A plain user reads null: the field stays out of their panel
	// (the share grants the content, never the machine layout).
	viewerID, viewerCookie := env.createSecondUser(t, "path-viewer@example.com")
	env.shareTimelineAlbum(t, env.syncAlbumIDOf(t, adminID), viewerID, adminID)
	rec := env.serveGet(t, "/api/v1/assets/"+id.String(), viewerCookie, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("viewer detail status = %d, want %d (body %q)", rec.Code, http.StatusOK, rec.Body.String())
	}
	var served apigen.AssetDetail
	if err := json.Unmarshal(rec.Body.Bytes(), &served); err != nil {
		t.Fatalf("decode viewer detail %q: %v", rec.Body.String(), err)
	}
	if served.Path != nil {
		t.Errorf("viewer detail path = %q, want null (machine information)", *served.Path)
	}
}

// postFaceRaw fires one POST /faces request with the raw body (a nil
// cookie sends it unauthenticated).
func postFaceRaw(t *testing.T, router http.Handler, cookie *http.Cookie, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/faces", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if cookie != nil {
		req.AddCookie(cookie)
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

// TestCreateFace pins the hand-drawn face: the drawn box becomes a
// face row of the caller's own live asset (no detector to name, the
// caller's own certainty for a score, no cluster until the assign
// flow writes one), the response carries the identity and the crop
// URL, and the detail's faces surface the row (bbox included) the
// on-photo layer renders from.
func TestCreateFace(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	id := fixtureAssetID("e1")
	env.insertOwnedAsset(t, id, env.sessionUserID(t),
		time.Date(2026, 7, 16, 10, 0, 0, 0, time.UTC), timelineAssetOpts{})

	rec := postFaceRaw(t, env.router, env.cookie,
		`{"assetId": "`+id.String()+`", "bbox": {"x": 260, "y": 300, "w": 150, "h": 150}}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("POST /faces status = %d, want %d (body %q)", rec.Code, http.StatusOK, rec.Body.String())
	}
	var created apigen.FaceCreated
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode face response %q: %v", rec.Body.String(), err)
	}
	if created.Id == uuid.Nil {
		t.Error("face response id = nil uuid, want the new row's id")
	}
	want := "/api/v1/faces/" + created.Id.String() + "/thumbnail"
	if created.Thumbnail != want {
		t.Errorf("face response thumbnail = %q, want %q", created.Thumbnail, want)
	}

	// The row: the drawn box as-is, the caller's own certainty, and
	// no detector to name.
	row := env.queryInt(t, "SELECT count(*) FROM faces WHERE id = $1", pgUUID(created.Id))
	if row != 1 {
		t.Fatalf("manual face rows = %d, want 1", row)
	}
	var modelID any
	if err := env.pool.QueryRow(context.Background(),
		"SELECT model_id FROM faces WHERE id = $1", pgUUID(created.Id)).Scan(&modelID); err != nil {
		t.Fatalf("read the manual face's model: %v", err)
	}
	if modelID != nil {
		t.Errorf("manual face model_id = %v, want NULL (no detector)", modelID)
	}
	if score := env.queryText(t, "SELECT score::text FROM faces WHERE id = $1", pgUUID(created.Id)); score != "1" {
		t.Errorf("manual face score = %q, want the caller's own certainty (1)", score)
	}

	// The detail's faces array surfaces the row with its box: the
	// face-box layer's geometry.
	page := env.serveGet(t, "/api/v1/assets/"+id.String(), env.cookie, nil)
	var detail apigen.AssetDetail
	if err := json.Unmarshal(page.Body.Bytes(), &detail); err != nil {
		t.Fatalf("decode detail %q: %v", page.Body.String(), err)
	}
	if len(detail.Faces) != 1 {
		t.Fatalf("detail faces = %d, want 1", len(detail.Faces))
	}
	face := detail.Faces[0]
	if face.Bbox.X != 260 || face.Bbox.Y != 300 || face.Bbox.W != 150 || face.Bbox.H != 150 {
		t.Errorf("detail face bbox = %v, want the drawn box", face.Bbox)
	}
	if face.ClusterId != nil {
		t.Errorf("detail face clusterId = %s, want null (no cluster yet)", *face.ClusterId)
	}
}

// TestCreateFaceValidation pins the drawn box's door: a zero-extent
// box, an out-of-bounds half and a missing half answer 400
// VALIDATION_FAILED naming the bbox; the door sits before any
// database access.
func TestCreateFaceValidation(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	id := fixtureAssetID("e2")
	env.insertOwnedAsset(t, id, env.sessionUserID(t),
		time.Date(2026, 7, 17, 10, 0, 0, 0, time.UTC), timelineAssetOpts{})

	assetJSON := `"assetId": "` + id.String() + `"`
	for _, tc := range []struct {
		body  string
		field string
	}{
		{`{` + assetJSON + `, "bbox": {"x": 100, "y": 100, "w": 0, "h": 0}}`, "bbox"},
		{`{` + assetJSON + `, "bbox": {"x": 1001, "y": 100, "w": 50, "h": 50}}`, "bbox"},
		{`{` + assetJSON + `, "bbox": {"x": 100, "y": -1, "w": 50, "h": 50}}`, "bbox"},
		{`{` + assetJSON + `}`, "bbox"},
		{`{`, "body"},
	} {
		rec := postFaceRaw(t, env.router, env.cookie, tc.body)
		requirePatchValidation(t, rec, tc.field)
	}
	if count := env.queryInt(t, "SELECT count(*) FROM faces"); count != 0 {
		t.Errorf("faces rows after the refusals = %d, want 0", count)
	}
}

// TestCreateFaceNotFound pins the face writes' refusal discipline:
// an unknown asset, a foreign one, and a trashed, hidden or Live
// Photo component asset all answer the people surface's byte
// identical 404 -- nothing was drawn on any of them.
func TestCreateFaceNotFound(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	graceID, _ := env.createSecondUser(t, "face-grace@example.com")
	unknown := uuid.MustParse("32000000-0000-0000-0000-0000000000f1")
	foreign := fixtureAssetID("e3")
	env.insertOwnedAsset(t, foreign, graceID,
		time.Date(2026, 7, 18, 10, 0, 0, 0, time.UTC), timelineAssetOpts{})
	mine := fixtureAssetID("e4")
	env.insertOwnedAsset(t, mine, env.sessionUserID(t),
		time.Date(2026, 7, 18, 11, 0, 0, 0, time.UTC), timelineAssetOpts{})
	env.exec(t, "UPDATE assets SET deleted_at = now(), updated_at = now() WHERE id = $1", pgUUID(mine))

	for _, asset := range []uuid.UUID{unknown, foreign, mine} {
		rec := postFaceRaw(t, env.router, env.cookie,
			`{"assetId": "`+asset.String()+`", "bbox": {"x": 100, "y": 100, "w": 100, "h": 100}}`)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("POST /faces for %s status = %d, want %d (body %q)",
				asset, rec.Code, http.StatusNotFound, rec.Body.String())
		}
		var apiErr apigen.Error
		if err := json.Unmarshal(rec.Body.Bytes(), &apiErr); err != nil {
			t.Fatalf("decode face error %q: %v", rec.Body.String(), err)
		}
		if apiErr.Code != apigen.PERSONNOTFOUND {
			t.Errorf("face refusal code = %q, want %q", apiErr.Code, apigen.PERSONNOTFOUND)
		}
	}
	if count := env.queryInt(t, "SELECT count(*) FROM faces"); count != 0 {
		t.Errorf("faces rows after the refusals = %d, want 0", count)
	}
}
