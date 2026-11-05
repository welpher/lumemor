// Batch action tests (M2c Task 4): the set semantics of the batch
// endpoint -- the mixed-resolution report (own, foreign, unknown and
// already-trashed ids in one request), the idempotent convergence of
// repeated actions, the validation door, the timeline effect of hide
// and unhide, and the event discipline (asset.trashed /
// asset.restored / asset.updated after the commit, favorites silent).
// Fixtures insert asset rows directly: the batch writes database
// state only, never files, so the row is the whole fixture.
package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image/color"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	apigen "github.com/welpher/lumemor/server/internal/api/gen"
)

// batchBody marshals one batch request body out of the action and the
// id list (a nil list marshals as the empty list).
func batchBody(t *testing.T, action string, ids []uuid.UUID) []byte {
	t.Helper()
	body, err := json.Marshal(map[string]any{"action": action, "ids": ids})
	if err != nil {
		t.Fatalf("marshal batch body: %v", err)
	}
	return body
}

// postBatch fires one batch request with the raw body and returns the
// recorder (nil cookie sends it unauthenticated).
func (e *assetEnv) postBatch(t *testing.T, cookie *http.Cookie, body []byte) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/assets/batch", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if cookie != nil {
		req.AddCookie(cookie)
	}
	rec := httptest.NewRecorder()
	e.router.ServeHTTP(rec, req)
	return rec
}

// batchAssets fires one batch request with the given action and ids
// and decodes the 200 report.
func (e *assetEnv) batchAssets(t *testing.T, cookie *http.Cookie, action string, ids ...uuid.UUID) apigen.BatchResponse {
	t.Helper()
	rec := e.postBatch(t, cookie, batchBody(t, action, ids))
	if rec.Code != http.StatusOK {
		t.Fatalf("POST /assets/batch %s status = %d, want %d (body %q)",
			action, rec.Code, http.StatusOK, rec.Body.String())
	}
	var response apigen.BatchResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode batch body %q: %v", rec.Body.String(), err)
	}
	return response
}

// requireBatchReport asserts the exact affected/missing report: the
// count and the missing ids in their first-appearance order.
func requireBatchReport(t *testing.T, got apigen.BatchResponse, affected int, missing ...uuid.UUID) {
	t.Helper()
	if got.Affected != affected {
		t.Errorf("batch affected = %d, want %d", got.Affected, affected)
	}
	if len(got.Missing) != len(missing) {
		t.Fatalf("batch missing = %v, want %v", got.Missing, missing)
	}
	for i, want := range missing {
		if uuid.UUID(got.Missing[i]) != want {
			t.Errorf("batch missing[%d] = %s, want %s", i, got.Missing[i], want)
		}
	}
}

// requireBatchValidation asserts the 400 VALIDATION_FAILED envelope
// naming the offending field.
func requireBatchValidation(t *testing.T, rec *httptest.ResponseRecorder, field string) {
	t.Helper()
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("batch status = %d, want %d (body %q)", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
	var apiErr apigen.Error
	if err := json.Unmarshal(rec.Body.Bytes(), &apiErr); err != nil {
		t.Fatalf("decode batch error body %q: %v", rec.Body.String(), err)
	}
	if apiErr.Code != apigen.VALIDATIONFAILED {
		t.Fatalf("batch error code = %q, want %q", apiErr.Code, apigen.VALIDATIONFAILED)
	}
	if apiErr.Details == nil {
		t.Fatalf("batch error details absent, want fields naming %q", field)
	}
	raw, ok := (*apiErr.Details)["fields"]
	if !ok {
		t.Fatalf("batch error details carry no fields, want %q", field)
	}
	fields, ok := raw.([]any)
	if !ok {
		t.Fatalf("batch error fields %v are not a list, want it to contain %q", raw, field)
	}
	for _, entry := range fields {
		if name, ok := entry.(string); ok && name == field {
			return
		}
	}
	t.Errorf("batch error fields = %v, want it to contain %q", fields, field)
}

// TestBatchMixedResolution pins the mixed-resolution report of one
// batch: the own ids apply (an already-trashed one included -- the
// idempotent no-op still counts as acted on), the foreign and unknown
// ids are ignored and reported in missing, in first-appearance order,
// and repeated ids collapse. The favorite action resolves with the
// same ownership guard, whatever the row's state.
func TestBatchMixedResolution(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	adminID := env.sessionUserID(t)
	graceID, _ := env.createSecondUser(t, "batch-mixed-grace@example.com")

	taken := time.Date(2026, 7, 1, 10, 0, 0, 0, time.UTC)
	live := fixtureAssetID("41")
	alreadyTrashed := fixtureAssetID("42")
	foreign := fixtureAssetID("43")
	env.insertOwnedAsset(t, live, adminID, taken, timelineAssetOpts{})
	env.insertOwnedAsset(t, alreadyTrashed, adminID, taken.Add(time.Minute), timelineAssetOpts{deleted: true})
	env.insertOwnedAsset(t, foreign, graceID, taken.Add(2*time.Minute), timelineAssetOpts{})
	unknown := uuid.MustParse("31000000-0000-0000-0000-000000000041")
	trashedMark := env.assetRowOf(t, alreadyTrashed).DeletedAt

	// One mixed trash batch: the own live id, a foreign one, an
	// unknown one, the own already-trashed id, and the live id a
	// second time. The report partitions the four distinct ids.
	report := env.batchAssets(t, env.cookie, "trash", live, foreign, unknown, alreadyTrashed, live)
	requireBatchReport(t, report, 2, foreign, unknown)

	// The write landed where the report says it did: the live row is
	// trashed now, the already-trashed row kept its original mark
	// (the no-op did not re-stamp it), and the foreign row is
	// untouched for its owner.
	for _, id := range []uuid.UUID{live, alreadyTrashed} {
		if !env.assetRowOf(t, id).DeletedAt.Valid {
			t.Errorf("asset %s is live after its trash batch, want trashed", id)
		}
	}
	if kept := env.assetRowOf(t, alreadyTrashed).DeletedAt; !kept.Time.Equal(trashedMark.Time) {
		t.Errorf("the repeat trash re-stamped deletedAt to %v, want the original %v", kept.Time, trashedMark.Time)
	}
	if env.assetRowOf(t, foreign).DeletedAt.Valid {
		t.Error("the foreign asset was trashed by another user's batch, want untouched")
	}

	// The favorite action guards by ownership alone: the trashed own
	// id is a valid favorite target (the row rides the trash out and
	// resurfaces with the restore), the foreign id is not.
	report = env.batchAssets(t, env.cookie, "favorite", live, foreign)
	requireBatchReport(t, report, 1, foreign)
	if count := env.queryInt(t, "SELECT count(*) FROM user_favorites WHERE user_id = $1 AND asset_id = $2",
		pgUUID(adminID), pgUUID(live)); count != 1 {
		t.Errorf("favorites rows for the trashed favorite = %d, want 1", count)
	}
}

// TestBatchSetSemantics pins the convergence of repeated actions: the
// report is NOT a state-change counter -- a repeat answers the same
// affected count (the idempotent no-op is still acted on), and only
// the actual transition announces itself on the bus.
func TestBatchSetSemantics(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	adminID := env.sessionUserID(t)

	taken := time.Date(2026, 7, 2, 10, 0, 0, 0, time.UTC)
	f1 := fixtureAssetID("51")
	f2 := fixtureAssetID("52")
	f3 := fixtureAssetID("53")
	env.insertOwnedAsset(t, f1, adminID, taken, timelineAssetOpts{})
	env.insertOwnedAsset(t, f2, adminID, taken.Add(time.Minute), timelineAssetOpts{})
	env.insertOwnedAsset(t, f3, adminID, taken.Add(2*time.Minute), timelineAssetOpts{})

	// Favoriting twice converges: both runs report the same full
	// count, and the rows exist exactly once.
	for run := 1; run <= 2; run++ {
		report := env.batchAssets(t, env.cookie, "favorite", f1, f2)
		requireBatchReport(t, report, 2)
	}
	if count := env.queryInt(t, "SELECT count(*) FROM user_favorites WHERE user_id = $1",
		pgUUID(adminID)); count != 2 {
		t.Errorf("favorites rows after the repeats = %d, want 2", count)
	}

	// Unfavoriting an id without a favorites row (never favorited, or
	// already unfavorited) is the same no-op: applied and reported,
	// not an error.
	report := env.batchAssets(t, env.cookie, "unfavorite", f3)
	requireBatchReport(t, report, 1)
	for run := 1; run <= 2; run++ {
		report := env.batchAssets(t, env.cookie, "unfavorite", f1, f2)
		requireBatchReport(t, report, 2)
	}
	if count := env.queryInt(t, "SELECT count(*) FROM user_favorites WHERE user_id = $1",
		pgUUID(adminID)); count != 0 {
		t.Errorf("favorites rows after the unfavorites = %d, want 0", count)
	}

	// Trashing twice: the first run transitions and announces both
	// ids; the second is the idempotent no-op -- reported as affected
	// (the pin: affected is not a state-change counter) and silent.
	env.batchAssets(t, env.cookie, "trash", f1, f2)
	if events := busEventsOf(t, env.bus, "asset.trashed"); len(events) != 2 {
		t.Fatalf("asset.trashed events after the first trash = %d, want 2", len(events))
	}
	report = env.batchAssets(t, env.cookie, "trash", f1, f2)
	requireBatchReport(t, report, 2)
	if events := busEventsOf(t, env.bus, "asset.trashed"); len(events) != 2 {
		t.Errorf("asset.trashed events after the repeat trash = %d, want still 2 (the no-op stays silent)", len(events))
	}

	// Restoring a live row (never trashed): applied, reported, silent.
	report = env.batchAssets(t, env.cookie, "restore", f3)
	requireBatchReport(t, report, 1)
	if events := busEventsOf(t, env.bus, "asset.restored"); len(events) != 0 {
		t.Errorf("asset.restored events after the live-row restore = %d, want 0", len(events))
	}
}

// TestBatchValidation pins the request door: the empty id list, the
// id list past the cap, the action outside the enum, bodies that are
// not valid JSON (an id that is not a uuid included), and the
// authentication requirement.
func TestBatchValidation(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	id := fixtureAssetID("61")
	env.insertTimelineAsset(t, id, env.sessionUserID(t), time.Date(2026, 7, 3, 10, 0, 0, 0, time.UTC), timelineAssetOpts{})

	// An empty id list is a client bug the 400 names.
	rec := env.postBatch(t, env.cookie, batchBody(t, "trash", nil))
	requireBatchValidation(t, rec, "ids")

	// The cap: 1001 ids is one past the bound.
	over := make([]uuid.UUID, 1001)
	for i := range over {
		over[i] = uuid.MustParse(fmt.Sprintf("31000000-0000-0000-0002-%012x", i))
	}
	rec = env.postBatch(t, env.cookie, batchBody(t, "trash", over))
	requireBatchValidation(t, rec, "ids")

	// An action outside the enum.
	rec = env.postBatch(t, env.cookie, batchBody(t, "expunge", []uuid.UUID{id}))
	requireBatchValidation(t, rec, "action")

	// Bodies that do not decode: malformed JSON, or an id that is not
	// a uuid.
	for _, body := range []string{"{", `{"action":"trash","ids":["not-a-uuid"]}`} {
		rec = env.postBatch(t, env.cookie, []byte(body))
		requireBatchValidation(t, rec, "body")
	}

	// Unauthenticated requests never reach any of this.
	if rec := env.postBatch(t, nil, batchBody(t, "trash", []uuid.UUID{id})); rec.Code != http.StatusUnauthorized {
		t.Errorf("unauthenticated batch status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

// TestBatchHideAffectsTimeline pins the timeline half of the hide
// switch: the batch hide removes the asset from the caller's timeline
// (the live filter of the timeline query, Task 2's filter verified
// through the endpoint), and the unhide returns it.
func TestBatchHideAffectsTimeline(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	adminID := env.sessionUserID(t)
	sync := env.syncAlbumIDOf(t, adminID)

	taken := time.Date(2026, 7, 4, 10, 0, 0, 0, time.UTC)
	hidden := fixtureAssetID("71")
	visible := fixtureAssetID("72")
	env.insertTimelineAsset(t, hidden, adminID, taken.Add(time.Minute), timelineAssetOpts{})
	env.insertTimelineAsset(t, visible, adminID, taken, timelineAssetOpts{})
	for _, id := range []uuid.UUID{hidden, visible} {
		env.addTimelineItem(t, sync, id, adminID)
	}

	page := env.getTimeline(t, env.cookie, "")
	requireTimelineIDs(t, page.Items, hidden, visible)

	// The hide batch flips the owner-level switch and the asset leaves
	// the timeline.
	report := env.batchAssets(t, env.cookie, "hide", hidden)
	requireBatchReport(t, report, 1)
	page = env.getTimeline(t, env.cookie, "")
	requireTimelineIDs(t, page.Items, visible)
	if row := env.assetRowOf(t, hidden); !row.IsHidden {
		t.Error("the hidden asset's row kept is_hidden false, want true")
	}

	// The unhide batch returns it.
	report = env.batchAssets(t, env.cookie, "unhide", hidden)
	requireBatchReport(t, report, 1)
	page = env.getTimeline(t, env.cookie, "")
	requireTimelineIDs(t, page.Items, hidden, visible)
}

// TestBatchLivePhotoPair pins the pair-through-batch behavior the
// affected count documents: trashing a Live Photo by its photo id
// moves the pair's TWO rows (the component video rides along, Task
// 3's pair predicate) while counting ONE affected id and announcing
// ONE event -- and restoring by the COMPONENT id brings the pair
// back as one, the predicate addressing the pair from either side.
func TestBatchLivePhotoPair(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)

	// A Live Photo upload: the photo row plus its linked component
	// video row (the upload endpoint's pairing, M2b).
	pairBody, pairContentType := buildUploadBody(t, map[string]string{
		"deviceAssetId":   "batch-pair",
		"fileName":        "pair.jpg",
		"deviceCreatedAt": time.Date(2026, 7, 6, 10, 0, 0, 0, time.UTC).Format(time.RFC3339),
	}, []assetUploadFile{
		{field: uploadPartPhoto, name: "pair.jpg", content: jpegColorFixture(t, color.RGBA{R: 0x91, G: 0x92, B: 0x93, A: 0xff})},
		{field: uploadPartLiveVideo, name: "pair.mov", content: movFixture()},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/assets", pairBody)
	req.Header.Set("Content-Type", pairContentType)
	req.AddCookie(env.cookie)
	rec := httptest.NewRecorder()
	env.router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("live photo upload status = %d, want %d (body %q)", rec.Code, http.StatusCreated, rec.Body.String())
	}
	pair := decodeAsset(t, rec)

	var component pgtype.UUID
	if err := env.pool.QueryRow(context.Background(),
		"SELECT live_photo_video_id FROM assets WHERE id = $1", pgUUID(uuid.UUID(pair.Id))).Scan(&component); err != nil {
		t.Fatalf("load the pair's component link: %v", err)
	}
	componentID := uuid.UUID(component.Bytes)

	// Trashing the photo id: both rows marked, ONE affected id, ONE
	// event (the pair is one asset, whichever row names it).
	report := env.batchAssets(t, env.cookie, "trash", uuid.UUID(pair.Id))
	requireBatchReport(t, report, 1)
	if !env.assetRowOf(t, uuid.UUID(pair.Id)).DeletedAt.Valid || !env.assetRowOf(t, componentID).DeletedAt.Valid {
		t.Error("the pair did not trash together, want the photo and its component both trashed")
	}
	if events := busEventsOf(t, env.bus, "asset.trashed"); len(events) != 1 || events[0]["assetId"] != pair.Id.String() {
		t.Errorf("asset.trashed events = %v, want exactly one for the photo id", events)
	}

	// Restoring by the COMPONENT id brings the pair back as one.
	report = env.batchAssets(t, env.cookie, "restore", componentID)
	requireBatchReport(t, report, 1)
	if env.assetRowOf(t, uuid.UUID(pair.Id)).DeletedAt.Valid || env.assetRowOf(t, componentID).DeletedAt.Valid {
		t.Error("the pair did not restore together, want both rows live")
	}
}

// TestBatchEvents pins the event discipline of the batch: favorites
// publish nothing (per-user state), hide/unhide announce one
// asset.updated per actual transition, trash/restore announce one
// event per id once the transaction committed, every payload the
// id-plus-kind minimal shape.
func TestBatchEvents(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	adminID := env.sessionUserID(t)

	taken := time.Date(2026, 7, 5, 10, 0, 0, 0, time.UTC)
	e1 := fixtureAssetID("81")
	e2 := fixtureAssetID("82")
	for _, id := range []uuid.UUID{e1, e2} {
		env.insertTimelineAsset(t, id, adminID, taken, timelineAssetOpts{})
	}

	// Favorites are silent: no event of any name fires for the
	// per-user mark.
	env.batchAssets(t, env.cookie, "favorite", e1)
	for _, name := range []string{"asset.updated", "asset.trashed", "asset.restored"} {
		if events := busEventsOf(t, env.bus, name); len(events) != 0 {
			t.Errorf("%s events after a favorite = %v, want none (per-user state publishes nothing)", name, events)
		}
	}

	// Hide and unhide announce the visibility change: one
	// asset.updated per actual transition, the id-plus-kind payload of
	// every asset event. The repeat stays silent.
	env.batchAssets(t, env.cookie, "hide", e1)
	updated := busEventsOf(t, env.bus, "asset.updated")
	if len(updated) != 1 || updated[0]["assetId"] != e1.String() || updated[0]["kind"] != "image" {
		t.Errorf("asset.updated events after the hide = %v, want exactly one for %s", updated, e1)
	}
	env.batchAssets(t, env.cookie, "hide", e1)
	if updated = busEventsOf(t, env.bus, "asset.updated"); len(updated) != 1 {
		t.Errorf("asset.updated events after the repeat hide = %d, want still 1", len(updated))
	}
	env.batchAssets(t, env.cookie, "unhide", e1)
	if updated = busEventsOf(t, env.bus, "asset.updated"); len(updated) != 2 {
		t.Errorf("asset.updated events after the unhide = %d, want 2", len(updated))
	}

	// The trash batch announces one asset.trashed per id, the restore
	// one asset.restored per id -- published only because the
	// transaction committed.
	env.batchAssets(t, env.cookie, "trash", e1, e2)
	trashed := busEventsOf(t, env.bus, "asset.trashed")
	if len(trashed) != 2 {
		t.Fatalf("asset.trashed events = %v, want one per batch id", trashed)
	}
	for _, event := range trashed {
		if event["assetId"] != e1.String() && event["assetId"] != e2.String() {
			t.Errorf("asset.trashed payload %v names neither batch id", event)
		}
		if event["kind"] != "image" {
			t.Errorf("asset.trashed payload %v carries kind %q, want image", event, event["kind"])
		}
	}
	env.batchAssets(t, env.cookie, "restore", e1, e2)
	if restored := busEventsOf(t, env.bus, "asset.restored"); len(restored) != 2 {
		t.Errorf("asset.restored events = %d, want one per batch id", len(restored))
	}
}
