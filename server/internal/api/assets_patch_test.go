// Asset attribute PATCH tests: the partial edit of the
// description and the archive switch -- the applied values surfacing
// in the response and the detail read, the timeline effect of the
// archive switch, the one-event-per-change discipline, the set
// semantics of repeated values, the validation door (neither field,
// pure nulls, the 10,000-rune bound, malformed bodies), and the
// owner-only anti-enumeration 404. Fixtures insert asset rows
// directly: the PATCH writes database state only, never files.
package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	apigen "github.com/welpher/lumemor/server/internal/api/gen"
)

// patchAssetBody marshals one patch request body out of its fields (a
// nil value marshals as JSON null).
func patchAssetBody(t *testing.T, fields map[string]any) []byte {
	t.Helper()
	body, err := json.Marshal(fields)
	if err != nil {
		t.Fatalf("marshal patch body: %v", err)
	}
	return body
}

// patchAssetRaw fires one PATCH request with the raw body and returns
// the recorder (nil cookie sends it unauthenticated).
func (e *assetEnv) patchAssetRaw(t *testing.T, cookie *http.Cookie, id uuid.UUID, body []byte) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/assets/"+id.String(), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if cookie != nil {
		req.AddCookie(cookie)
	}
	rec := httptest.NewRecorder()
	e.router.ServeHTTP(rec, req)
	return rec
}

// patchAsset applies one patch and decodes the 200 detail response.
func (e *assetEnv) patchAsset(t *testing.T, cookie *http.Cookie, id uuid.UUID, fields map[string]any) apigen.AssetDetail {
	t.Helper()
	rec := e.patchAssetRaw(t, cookie, id, patchAssetBody(t, fields))
	if rec.Code != http.StatusOK {
		t.Fatalf("PATCH /assets/%s status = %d, want %d (body %q)", id, rec.Code, http.StatusOK, rec.Body.String())
	}
	var detail apigen.AssetDetail
	if err := json.Unmarshal(rec.Body.Bytes(), &detail); err != nil {
		t.Fatalf("decode patched detail %q: %v", rec.Body.String(), err)
	}
	return detail
}

// requirePatchValidation asserts the 400 VALIDATION_FAILED envelope
// naming the offending field.
func requirePatchValidation(t *testing.T, rec *httptest.ResponseRecorder, field string) {
	t.Helper()
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("patch status = %d, want %d (body %q)", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
	var apiErr apigen.Error
	if err := json.Unmarshal(rec.Body.Bytes(), &apiErr); err != nil {
		t.Fatalf("decode patch error body %q: %v", rec.Body.String(), err)
	}
	if apiErr.Code != apigen.VALIDATIONFAILED {
		t.Fatalf("patch error code = %q, want %q", apiErr.Code, apigen.VALIDATIONFAILED)
	}
	if apiErr.Details == nil {
		t.Fatalf("patch error details absent, want fields naming %q", field)
	}
	raw, ok := (*apiErr.Details)["fields"]
	if !ok {
		t.Fatalf("patch error details carry no fields, want %q", field)
	}
	fields, ok := raw.([]any)
	if !ok {
		t.Fatalf("patch error fields %v are not a list, want it to contain %q", raw, field)
	}
	for _, entry := range fields {
		if name, ok := entry.(string); ok && name == field {
			return
		}
	}
	t.Errorf("patch error fields = %v, want it to contain %q", fields, field)
}

// requireAssetUpdatedEvents asserts the bus carries exactly count
// asset.updated events for the given asset, every payload shaped
// {assetId, kind}.
func requireAssetUpdatedEvents(t *testing.T, e *assetEnv, count int, id uuid.UUID, kind string) {
	t.Helper()
	events := busEventsOf(t, e.bus, "asset.updated")
	if len(events) != count {
		t.Fatalf("asset.updated events = %d, want %d (payloads %v)", len(events), count, events)
	}
	for i, payload := range events {
		if payload["assetId"] != id.String() {
			t.Errorf("asset.updated[%d] assetId = %q, want %q", i, payload["assetId"], id)
		}
		if payload["kind"] != kind {
			t.Errorf("asset.updated[%d] kind = %q, want %q", i, payload["kind"], kind)
		}
	}
}

// TestPatchAssetDescription pins the description half of the patch:
// the applied value surfaces in the response AND the detail read
// (the same shape the GET endpoint serves), the empty string clears
// the text, each actual change publishes one asset.updated event,
// and a trashed asset stays editable like it stays readable.
func TestPatchAssetDescription(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	id := fixtureAssetID("d1")
	env.insertOwnedAsset(t, id, env.sessionUserID(t),
		time.Date(2026, 7, 5, 10, 0, 0, 0, time.UTC), timelineAssetOpts{})

	// The fresh row carries no description and is visible; the patch
	// response is the detail with the applied value.
	detail := env.patchAsset(t, env.cookie, id, map[string]any{"description": "Sunset over the lake"})
	if detail.Id != id {
		t.Errorf("patched detail id = %s, want the patched asset %s", detail.Id, id)
	}
	if detail.Description != "Sunset over the lake" {
		t.Errorf("patched detail description = %q, want %q", detail.Description, "Sunset over the lake")
	}
	if detail.IsArchived {
		t.Error("patched detail isArchived = true, want untouched false")
	}
	requireAssetUpdatedEvents(t, env, 1, id, "image")

	// The detail read reflects the stored value.
	rec := env.serveGet(t, "/api/v1/assets/"+id.String(), env.cookie, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET detail status = %d, want %d (body %q)", rec.Code, http.StatusOK, rec.Body.String())
	}
	var served apigen.AssetDetail
	if err := json.Unmarshal(rec.Body.Bytes(), &served); err != nil {
		t.Fatalf("decode detail %q: %v", rec.Body.String(), err)
	}
	if served.Description != "Sunset over the lake" {
		t.Errorf("detail description = %q, want the patched %q", served.Description, "Sunset over the lake")
	}
	if served.IsArchived {
		t.Error("detail isArchived = true, want false")
	}
	if got := env.queryText(t, "SELECT description FROM assets WHERE id = $1", pgUUID(id)); got != "Sunset over the lake" {
		t.Errorf("row description = %q, want the patched value", got)
	}

	// The empty string is a valid patch and clears the text.
	detail = env.patchAsset(t, env.cookie, id, map[string]any{"description": ""})
	if detail.Description != "" {
		t.Errorf("cleared detail description = %q, want the empty string", detail.Description)
	}
	if got := env.queryText(t, "SELECT description FROM assets WHERE id = $1", pgUUID(id)); got != "" {
		t.Errorf("row description after the clear = %q, want the empty string", got)
	}
	requireAssetUpdatedEvents(t, env, 2, id, "image")

	// A trashed asset stays editable like it stays readable: the edit
	// lands on the row and resurfaces with the restore.
	env.exec(t, "UPDATE assets SET deleted_at = now(), updated_at = now() WHERE id = $1", pgUUID(id))
	detail = env.patchAsset(t, env.cookie, id, map[string]any{"description": "kept over the trash"})
	if detail.Description != "kept over the trash" {
		t.Errorf("trashed detail description = %q, want the patched value", detail.Description)
	}
	if !detail.IsTrashed {
		t.Error("trashed detail isTrashed = false, want true")
	}
	requireAssetUpdatedEvents(t, env, 3, id, "image")
}

// TestPatchAssetArchiveAffectsTimeline pins the archive half: the
// archive switch the patch flips removes the asset from the timeline
// (the live filter of the timeline query, verified through the
// endpoint, the same integration shape as the batch's),
// the unarchive returns it, and each transition publishes one
// asset.updated event.
func TestPatchAssetArchiveAffectsTimeline(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	adminID := env.sessionUserID(t)
	sync := env.syncAlbumIDOf(t, adminID)

	taken := time.Date(2026, 7, 6, 10, 0, 0, 0, time.UTC)
	hidden := fixtureAssetID("d2")
	visible := fixtureAssetID("d3")
	env.insertTimelineAsset(t, hidden, adminID, taken.Add(time.Minute), timelineAssetOpts{})
	env.insertTimelineAsset(t, visible, adminID, taken, timelineAssetOpts{})
	for _, id := range []uuid.UUID{hidden, visible} {
		env.addTimelineItem(t, sync, id, adminID)
	}

	page := env.getTimeline(t, env.cookie, "")
	requireTimelineIDs(t, page.Items, hidden, visible)

	// The hide patch flips the owner-level switch and the asset
	// leaves the timeline.
	detail := env.patchAsset(t, env.cookie, hidden, map[string]any{"isArchived": true})
	if !detail.IsArchived {
		t.Error("patched detail isArchived = false, want true")
	}
	if detail.Description != "" {
		t.Errorf("patched detail description = %q, want untouched empty", detail.Description)
	}
	page = env.getTimeline(t, env.cookie, "")
	requireTimelineIDs(t, page.Items, visible)
	if row := env.assetRowOf(t, hidden); !row.IsHidden {
		t.Error("the hidden asset's row kept is_hidden false, want true")
	}
	requireAssetUpdatedEvents(t, env, 1, hidden, "image")

	// The unarchive patch returns it.
	detail = env.patchAsset(t, env.cookie, hidden, map[string]any{"isArchived": false})
	if detail.IsArchived {
		t.Error("unarchived detail isArchived = true, want false")
	}
	page = env.getTimeline(t, env.cookie, "")
	requireTimelineIDs(t, page.Items, hidden, visible)
	requireAssetUpdatedEvents(t, env, 2, hidden, "image")
}

// TestPatchAssetBothFields pins the partial-edit contract: one
// request may carry both fields (they land together, ONE event
// covers the change), and a request that carries only one leaves
// the other's value untouched.
func TestPatchAssetBothFields(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	id := fixtureAssetID("d4")
	env.insertOwnedAsset(t, id, env.sessionUserID(t),
		time.Date(2026, 7, 7, 10, 0, 0, 0, time.UTC), timelineAssetOpts{})

	// Both fields in one request: the response carries both applied
	// values, and exactly one event covers the patch.
	detail := env.patchAsset(t, env.cookie, id, map[string]any{"description": "Both at once", "isArchived": true})
	if detail.Description != "Both at once" {
		t.Errorf("detail description = %q, want %q", detail.Description, "Both at once")
	}
	if !detail.IsArchived {
		t.Error("detail isArchived = false, want true")
	}
	requireAssetUpdatedEvents(t, env, 1, id, "image")

	// Only the description carried: it updates, the archive switch
	// keeps its value (partial edit, not a reset).
	detail = env.patchAsset(t, env.cookie, id, map[string]any{"description": "Only the text"})
	if detail.Description != "Only the text" {
		t.Errorf("detail description = %q, want %q", detail.Description, "Only the text")
	}
	if !detail.IsArchived {
		t.Error("detail isArchived = false, want the untouched true")
	}
	requireAssetUpdatedEvents(t, env, 2, id, "image")
}

// TestPatchAssetValidation pins the request door: a body that is not
// valid JSON (a field of the wrong type included), one that carries
// neither field (nulls count as absent), and a description past the
// 10,000-rune bound all answer 400 VALIDATION_FAILED naming the
// offending field; the bound counts runes (not bytes), so exactly
// 10,000 multi-byte characters pass; the door sits before any
// database access, and unauthenticated requests never reach it.
func TestPatchAssetValidation(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	id := fixtureAssetID("d5")
	env.insertOwnedAsset(t, id, env.sessionUserID(t),
		time.Date(2026, 7, 8, 10, 0, 0, 0, time.UTC), timelineAssetOpts{})

	// Neither field carried -- the empty object and the pure-null
	// bodies alike.
	for _, body := range []string{
		`{}`,
		`{"description": null}`,
		`{"isArchived": null}`,
		`{"description": null, "isArchived": null}`,
	} {
		rec := env.patchAssetRaw(t, env.cookie, id, []byte(body))
		requirePatchValidation(t, rec, "body")
	}

	// Bodies that do not decode: malformed JSON, a non-string
	// description, a non-boolean isArchived.
	for _, body := range []string{
		`{`,
		`{"description": 5}`,
		`{"isArchived": "yes"}`,
	} {
		rec := env.patchAssetRaw(t, env.cookie, id, []byte(body))
		requirePatchValidation(t, rec, "body")
	}

	// The description bound: exactly 10,000 runes of a multi-byte
	// character pass (the count is over runes, not bytes); one more
	// answers 400 naming the field.
	atBound := strings.Repeat("é", 10000)
	detail := env.patchAsset(t, env.cookie, id, map[string]any{"description": atBound})
	if detail.Description != atBound {
		t.Errorf("description at the bound came back changed, want the exact value")
	}
	overBound := strings.Repeat("é", 10001)
	rec := env.patchAssetRaw(t, env.cookie, id, patchAssetBody(t, map[string]any{"description": overBound}))
	requirePatchValidation(t, rec, "description")

	// The door sits before any database access: the same malformed
	// body answers 400 for an id that does not exist too (no
	// existence leak -- the refusal never consults the rows).
	unknown := uuid.MustParse("31000000-0000-0000-0000-0000000000d6")
	rec = env.patchAssetRaw(t, env.cookie, unknown, []byte(`{}`))
	requirePatchValidation(t, rec, "body")

	// Nothing the door refused reached the row, and nothing was
	// announced.
	if row := env.assetRowOf(t, id); row.IsHidden {
		t.Error("the validation-rejected patch flipped is_hidden, want untouched")
	}
	if got := env.queryText(t, "SELECT description FROM assets WHERE id = $1", pgUUID(id)); got != atBound {
		t.Errorf("row description = %d runes, want the at-bound value", len([]rune(got)))
	}
	if events := busEventsOf(t, env.bus, "asset.updated"); len(events) != 1 {
		t.Errorf("asset.updated events = %d, want 1 (only the at-bound patch changed the row)", len(events))
	}

	// Unauthenticated requests never reach the door.
	rec = env.patchAssetRaw(t, nil, id, patchAssetBody(t, map[string]any{"description": "no session"}))
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("unauthenticated patch status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

// TestPatchAssetSetSemantics pins the idempotent repeat: patching the
// value the row already carries answers the same 200 with the
// current state, writes nothing (updated_at stays un-stamped) and
// publishes no further event -- the endpoint is a convergence
// operation, not a state-change counter.
func TestPatchAssetSetSemantics(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	id := fixtureAssetID("d7")
	env.insertOwnedAsset(t, id, env.sessionUserID(t),
		time.Date(2026, 7, 9, 10, 0, 0, 0, time.UTC), timelineAssetOpts{})

	// Repeating the row's fresh values (empty description, hidden
	// false) is the no-op twice over.
	for run := 1; run <= 2; run++ {
		detail := env.patchAsset(t, env.cookie, id, map[string]any{"description": "", "isArchived": false})
		if detail.Description != "" || detail.IsArchived {
			t.Errorf("run %d no-op detail = %q/%v, want the untouched values", run, detail.Description, detail.IsArchived)
		}
	}
	requireAssetUpdatedEvents(t, env, 0, id, "image")

	// A real change, then its repeat: the repeat answers 200 with the
	// same state and stays silent.
	env.patchAsset(t, env.cookie, id, map[string]any{"description": "kept"})
	requireAssetUpdatedEvents(t, env, 1, id, "image")
	mark := env.assetRowOf(t, id).UpdatedAt
	detail := env.patchAsset(t, env.cookie, id, map[string]any{"description": "kept"})
	if detail.Description != "kept" {
		t.Errorf("repeat detail description = %q, want %q", detail.Description, "kept")
	}
	requireAssetUpdatedEvents(t, env, 1, id, "image")

	// The no-op wrote nothing: updated_at kept the stamp of the
	// change that preceded it (two DB reads of the same value).
	if kept := env.assetRowOf(t, id).UpdatedAt; !kept.Time.Equal(mark.Time) {
		t.Errorf("the repeat re-stamped updated_at to %v, want the original %v", kept.Time, mark.Time)
	}

	// The archive switch repeats the same way: archive twice, one event.
	env.patchAsset(t, env.cookie, id, map[string]any{"isArchived": true})
	requireAssetUpdatedEvents(t, env, 2, id, "image")
	env.patchAsset(t, env.cookie, id, map[string]any{"isArchived": true})
	requireAssetUpdatedEvents(t, env, 2, id, "image")

	// The mixed no-op: both fields already at their target values.
	detail = env.patchAsset(t, env.cookie, id, map[string]any{"description": "kept", "isArchived": true})
	if detail.Description != "kept" || !detail.IsArchived {
		t.Errorf("mixed no-op detail = %q/%v, want the current state", detail.Description, detail.IsArchived)
	}
	requireAssetUpdatedEvents(t, env, 2, id, "image")
}

// TestPatchAssetNotFound pins the owner-only gate and the
// anti-enumeration 404: another account's asset and an unknown id
// answer the same 404 byte for byte, the caller here being an
// administrator (asset attribute editing has no admin override -- the
// permission matrix gives administrators no row on this surface,
// unlike the permanent delete), and the foreign row stays untouched.
func TestPatchAssetNotFound(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	graceID, _ := env.createSecondUser(t, "patch-grace@example.com")
	foreign := fixtureAssetID("d8")
	env.insertTimelineAsset(t, foreign, graceID,
		time.Date(2026, 7, 10, 10, 0, 0, 0, time.UTC), timelineAssetOpts{})
	unknown := uuid.MustParse("31000000-0000-0000-0000-0000000000d9")
	body := patchAssetBody(t, map[string]any{"description": "not yours", "isArchived": true})

	// The env's cookie is the first administrator: still 404, the
	// pin that PATCH has no administrator override.
	foreignRec := env.patchAssetRaw(t, env.cookie, foreign, body)
	if foreignRec.Code != http.StatusNotFound {
		t.Fatalf("patch of a foreign asset status = %d, want %d (body %q)",
			foreignRec.Code, http.StatusNotFound, foreignRec.Body.String())
	}
	unknownRec := env.patchAssetRaw(t, env.cookie, unknown, body)
	if unknownRec.Code != http.StatusNotFound {
		t.Fatalf("patch of an unknown id status = %d, want %d", unknownRec.Code, http.StatusNotFound)
	}
	if foreignRec.Body.String() != unknownRec.Body.String() {
		t.Errorf("the 404 bodies of the foreign (%q) and unknown (%q) ids differ",
			foreignRec.Body.String(), unknownRec.Body.String())
	}
	apiErr, _ := decodeError(t, foreignRec)
	if apiErr.Code != apigen.ASSETNOTFOUND {
		t.Errorf("patch 404 code = %q, want %q", apiErr.Code, apigen.ASSETNOTFOUND)
	}

	// The foreign row is untouched for its owner and nothing was
	// announced.
	if row := env.assetRowOf(t, foreign); row.IsHidden || row.Description != "" {
		t.Errorf("the foreign asset was patched to %q/%v, want untouched", row.Description, row.IsHidden)
	}
	if events := busEventsOf(t, env.bus, "asset.updated"); len(events) != 0 {
		t.Errorf("asset.updated events = %d, want 0", len(events))
	}
}
