// Album audience tests (E4, the albums-as-sources milestone): the
// full-set write behind PUT /albums/{albumId}/audience -- the
// idempotent convergence onto the requested set, the honest skip
// report of the unusable entries, the per-transition event
// discipline, the member-visibility convergence (the wall and the
// detail of the affected accounts), and the write face's two-tier
// refusal matrix (invisible 404, sharee 403, every sync album 404
// whoever owns it, locked never gating). The owner-only grantee list
// on the album detail and the instance's minimal member list ride
// along -- the edit dialog's chips and its candidate picker.
package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/welpher/lumemor/server/internal/events"

	apigen "github.com/welpher/lumemor/server/internal/api/gen"
)

// putAlbumAudienceRaw fires PUT /albums/{albumId}/audience with the
// raw body (a nil cookie sends it unauthenticated).
func (e *assetEnv) putAlbumAudienceRaw(t *testing.T, cookie *http.Cookie, albumID uuid.UUID, body string) *httptest.ResponseRecorder {
	t.Helper()
	return e.writeAlbumRaw(t, cookie, http.MethodPut,
		"/api/v1/albums/"+albumID.String()+"/audience", body)
}

// putAlbumAudience fires PUT /albums/{albumId}/audience over one id
// set and decodes the 200 report. A zero-argument call is the
// explicit empty set (the honest "only me" request), never the
// absent field.
func (e *assetEnv) putAlbumAudience(t *testing.T, cookie *http.Cookie, albumID uuid.UUID, ids ...uuid.UUID) apigen.AlbumAudienceResponse {
	t.Helper()
	body := `{"audience":[` + albumIDList(ids) + `]}`
	rec := e.putAlbumAudienceRaw(t, cookie, albumID, body)
	if rec.Code != http.StatusOK {
		t.Fatalf("PUT /albums/%s/audience status = %d, want %d (body %q)",
			albumID, rec.Code, http.StatusOK, rec.Body.String())
	}
	var report apigen.AlbumAudienceResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &report); err != nil {
		t.Fatalf("decode audience body %q: %v", rec.Body.String(), err)
	}
	return report
}

// shareRowsOf counts the album's share rows (the convergence's
// row-level truth).
func (e *assetEnv) shareRowsOf(t *testing.T, albumID uuid.UUID) int {
	t.Helper()
	return e.queryInt(t, "SELECT count(*) FROM album_shares WHERE album_id = $1", pgUUID(albumID))
}

// eventsOfAlbum scans the bus ring for one event name's payloads of
// one album -- the id-filtered variant of busEventsOf, independent of
// whatever else the ring holds from earlier setup.
func eventsOfAlbum(t *testing.T, bus *events.Bus, name string, albumID uuid.UUID) []map[string]string {
	t.Helper()
	var payloads []map[string]string
	for _, payload := range busEventsOf(t, bus, name) {
		if payload["albumId"] == albumID.String() {
			payloads = append(payloads, payload)
		}
	}
	return payloads
}

// audienceIDs widens the report's audience entries to plain ids.
func audienceIDs(entries []apigen.AlbumShareEntry) []uuid.UUID {
	ids := make([]uuid.UUID, 0, len(entries))
	for _, entry := range entries {
		ids = append(ids, uuid.UUID(entry.UserId))
	}
	return ids
}

// requireAudienceIDs asserts one id list exactly (order included).
func requireAudienceIDs(t *testing.T, got []uuid.UUID, want ...uuid.UUID) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("ids = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("ids = %v, want %v", got, want)
		}
	}
}

// requireEventUserIds asserts one event list's grantee ids as a SET
// (the statements' RETURNING order is not a contract; the ids are).
func requireEventUserIds(t *testing.T, events []map[string]string, want ...uuid.UUID) {
	t.Helper()
	if len(events) != len(want) {
		t.Fatalf("events = %v, want the grantees %v", events, want)
	}
	got := make(map[string]bool, len(events))
	for _, payload := range events {
		got[payload["userId"]] = true
	}
	for _, id := range want {
		if !got[id.String()] {
			t.Errorf("the event set %v is missing the grantee %s", events, id)
		}
	}
}

// TestAlbumAudienceRequireSession pins the 401 of the audience face:
// the session middleware rejects the request before any handler
// runs.
func TestAlbumAudienceRequireSession(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	albumID := fixtureWallAlbumID("71")
	rec := env.putAlbumAudienceRaw(t, nil, albumID, `{"audience":[]}`)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("PUT /albums/{id}/audience unauthenticated status = %d, want %d",
			rec.Code, http.StatusUnauthorized)
	}
}

// TestUpdateAlbumAudience pins the convergence's happy path and its
// idempotency: the requested set lands (one album.shared per row),
// the same set again answers the same state with zero transitions
// and zero events (the original granted times untouched), and the
// report's audience field carries the grantee list the owner's
// detail carries.
func TestUpdateAlbumAudience(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	graceID, graceCookie := env.createSecondUser(t, "audience-grace@example.com")
	hopperID, _ := env.createSecondUser(t, "audience-hopper@example.com")

	root := env.provisionMountRoot(t)
	created := env.createAlbum(t, env.cookie, "The shared album", root, nil)
	albumID := uuid.UUID(created.Id)

	report := env.putAlbumAudience(t, env.cookie, albumID, graceID, hopperID)
	requireAudienceIDs(t, report.Granted, graceID, hopperID)
	requireAudienceIDs(t, report.Revoked)
	requireAudienceIDs(t, audienceIDs(report.Audience), graceID, hopperID)
	if len(report.Skipped) != 0 {
		t.Errorf("skipped = %v, want none", report.Skipped)
	}
	if rows := env.shareRowsOf(t, albumID); rows != 2 {
		t.Fatalf("share rows = %d, want 2", rows)
	}

	// The grantees' visibility converges immediately: the wall
	// carries the shared card, the detail answers the viewer role.
	wall := env.getAlbumWall(t, graceCookie)
	if _, ok := wallAlbumOf(wall, albumID); !ok {
		t.Fatalf("grace's wall holds no card for the shared album (wall %v)", wall.Albums)
	}
	detail := env.getAlbumDetail(t, graceCookie, albumID, "")
	if detail.CallerRole != apigen.AlbumDetailCallerRoleViewer {
		t.Errorf("sharee detail callerRole = %q, want viewer", detail.CallerRole)
	}

	// The two events: one per landed row, the grantee in the
	// payload.
	shared := eventsOfAlbum(t, env.bus, "album.shared", albumID)
	requireEventUserIds(t, shared, graceID, hopperID)
	for _, payload := range shared {
		if payload["albumId"] != albumID.String() {
			t.Errorf("album.shared albumId = %q, want %s", payload["albumId"], albumID)
		}
	}
	if unshared := eventsOfAlbum(t, env.bus, "album.unshared", albumID); len(unshared) != 0 {
		t.Errorf("album.unshared events = %d, want 0", len(unshared))
	}

	// The idempotent repeat: the same set again answers the same
	// state -- zero grants, zero revokes, zero events -- and the
	// original grant times stay untouched.
	firstGrant := env.queryText(t,
		"SELECT granted_at::text FROM album_shares WHERE album_id = $1 AND user_id = $2",
		pgUUID(albumID), pgUUID(graceID))
	again := env.putAlbumAudience(t, env.cookie, albumID, hopperID, graceID)
	requireAudienceIDs(t, again.Granted)
	requireAudienceIDs(t, again.Revoked)
	requireAudienceIDs(t, audienceIDs(again.Audience), graceID, hopperID)
	if len(again.Skipped) != 0 {
		t.Errorf("the repeat's skipped = %v, want none", again.Skipped)
	}
	if second := env.queryText(t,
		"SELECT granted_at::text FROM album_shares WHERE album_id = $1 AND user_id = $2",
		pgUUID(albumID), pgUUID(graceID)); second != firstGrant {
		t.Errorf("granted_at after the repeat = %q, want the untouched %q", second, firstGrant)
	}
	if sharedAfter := eventsOfAlbum(t, env.bus, "album.shared", albumID); len(sharedAfter) != 2 {
		t.Errorf("album.shared events after the repeat = %d, want still 2 (the no-op stays silent)", len(sharedAfter))
	}
}

// TestUpdateAlbumAudienceFullSetRemoval pins the full-set semantics
// on the removal side: a set that no longer names a grantee revokes
// them (one album.unshared, the wall and detail lose the album), a
// set that names them keeps them, and the EMPTY set is the honest
// "only me" -- every row leaves while the owner's own view never
// moves (the owner's access is the ownership row).
func TestUpdateAlbumAudienceFullSetRemoval(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	graceID, graceCookie := env.createSecondUser(t, "removal-grace@example.com")
	hopperID, hopperCookie := env.createSecondUser(t, "removal-hopper@example.com")

	root := env.provisionMountRoot(t)
	created := env.createAlbum(t, env.cookie, "The revoked album", root, nil)
	albumID := uuid.UUID(created.Id)
	env.putAlbumAudience(t, env.cookie, albumID, graceID, hopperID)
	if rows := env.shareRowsOf(t, albumID); rows != 2 {
		t.Fatalf("share rows = %d, want 2 (the fixture is broken)", rows)
	}

	// The owner drops grace from the set: hopper stays.
	report := env.putAlbumAudience(t, env.cookie, albumID, hopperID)
	requireAudienceIDs(t, report.Granted)
	requireAudienceIDs(t, report.Revoked, graceID)
	requireAudienceIDs(t, audienceIDs(report.Audience), hopperID)
	if rows := env.shareRowsOf(t, albumID); rows != 1 {
		t.Fatalf("share rows after the removal = %d, want 1", rows)
	}

	// The revoked grantee's visibility is gone: no wall card, no
	// detail.
	wall := env.getAlbumWall(t, graceCookie)
	if _, ok := wallAlbumOf(wall, albumID); ok {
		t.Errorf("grace's wall still holds the revoked album %s (wall %v)", albumID, wall.Albums)
	}
	rec := env.getAlbumsRaw(t, graceCookie, "/api/v1/albums/"+albumID.String())
	requireAlbumNotFound(t, rec)

	// The other grantee is untouched.
	hopperWall := env.getAlbumWall(t, hopperCookie)
	if _, ok := wallAlbumOf(hopperWall, albumID); !ok {
		t.Errorf("hopper's wall lost the album although his row is intact (wall %v)", hopperWall.Albums)
	}

	// One album.unshared fired, for the removed grantee.
	unshared := eventsOfAlbum(t, env.bus, "album.unshared", albumID)
	if len(unshared) != 1 {
		t.Fatalf("album.unshared events = %d, want 1 (payloads %v)", len(unshared), unshared)
	}
	if unshared[0]["userId"] != graceID.String() {
		t.Errorf("album.unshared userId = %q, want the removed grantee %s", unshared[0]["userId"], graceID)
	}

	// The empty set: every row leaves, "only me".
	empty := env.putAlbumAudience(t, env.cookie, albumID)
	requireAudienceIDs(t, empty.Granted)
	requireAudienceIDs(t, empty.Revoked, hopperID)
	requireAudienceIDs(t, audienceIDs(empty.Audience))
	if rows := env.shareRowsOf(t, albumID); rows != 0 {
		t.Fatalf("share rows after the empty set = %d, want 0", rows)
	}
	hopperWall = env.getAlbumWall(t, hopperCookie)
	if _, ok := wallAlbumOf(hopperWall, albumID); ok {
		t.Errorf("hopper's wall still holds the album after the empty set (wall %v)", hopperWall.Albums)
	}

	// The owner's own view never moved: the wall card stays, the
	// detail answers the owner role -- the owner's access is the
	// ownership row, never a share row.
	ownerWall := env.getAlbumWall(t, env.cookie)
	if _, ok := wallAlbumOf(ownerWall, albumID); !ok {
		t.Fatalf("the owner's wall lost the album (wall %v)", ownerWall.Albums)
	}
	detail := env.getAlbumDetail(t, env.cookie, albumID, "")
	if detail.CallerRole != apigen.AlbumDetailCallerRoleOwner {
		t.Errorf("owner detail callerRole = %q, want owner", detail.CallerRole)
	}

	// A set that names a current grantee again after the empty set
	// is a real transition: the row lands, the event fires.
	regained := env.putAlbumAudience(t, env.cookie, albumID, hopperID)
	requireAudienceIDs(t, regained.Granted, hopperID)
	if shared := eventsOfAlbum(t, env.bus, "album.shared", albumID); len(shared) != 3 {
		t.Errorf("album.shared events = %d, want 3 (the setup pair and the re-grant; payloads %v)",
			len(shared), shared)
	}
}

// TestUpdateAlbumAudienceSkips pins the honest skip report: the
// owner's own id and every id that names no usable account skip
// with their reasons -- never errors -- while the rest of the set
// still lands.
func TestUpdateAlbumAudienceSkips(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	adminID := env.sessionUserID(t)
	graceID, _ := env.createSecondUser(t, "skip-grace@example.com")
	unknown := uuid.MustParse("70000000-0000-0000-0004-000000000001")
	disabledID, _ := env.createSecondUser(t, "skip-disabled@example.com")
	env.exec(t, "UPDATE users SET is_disabled = true WHERE id = $1", pgUUID(disabledID))

	root := env.provisionMountRoot(t)
	created := env.createAlbum(t, env.cookie, "The skip matrix", root, nil)
	albumID := uuid.UUID(created.Id)

	report := env.putAlbumAudience(t, env.cookie, albumID,
		graceID, adminID, unknown, disabledID, graceID)
	requireAudienceIDs(t, report.Granted, graceID)
	requireAudienceIDs(t, audienceIDs(report.Audience), graceID)

	skipped := make(map[uuid.UUID]apigen.AlbumAudienceSkipReason, len(report.Skipped))
	for _, entry := range report.Skipped {
		skipped[uuid.UUID(entry.UserId)] = entry.Reason
	}
	if len(skipped) != 3 {
		t.Fatalf("skipped = %v, want the owner, the unknown id and the disabled account", report.Skipped)
	}
	if reason := skipped[adminID]; reason != apigen.AlbumAudienceSkipReasonOwner {
		t.Errorf("the owner's skip reason = %q, want owner", reason)
	}
	for _, id := range []uuid.UUID{unknown, disabledID} {
		if reason := skipped[id]; reason != apigen.AlbumAudienceSkipReasonInvalidAccount {
			t.Errorf("skip reason of %s = %q, want invalid_account", id, reason)
		}
	}
	if rows := env.shareRowsOf(t, albumID); rows != 1 {
		t.Errorf("share rows = %d, want 1 (the usable grantee alone landed)", rows)
	}
	if shared := eventsOfAlbum(t, env.bus, "album.shared", albumID); len(shared) != 1 {
		t.Errorf("album.shared events = %d, want 1 (nothing fired for the skips)", len(shared))
	}
}

// TestUpdateAlbumAudienceAuthorization pins the two-tier refusal
// matrix: an album the caller cannot see answers the single-shape
// 404, an album shared to the caller answers the honest 403 (the
// share is view-only -- a sharee cannot edit the ACL, not even to
// leave), and an administrator overrides the owner check.
func TestUpdateAlbumAudienceAuthorization(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	graceID, graceCookie := env.createSecondUser(t, "audience-auth-grace@example.com")
	hopperID, hopperCookie := env.createSecondUser(t, "audience-auth-hopper@example.com")

	root := env.provisionMountRoot(t)
	shared := uuid.UUID(env.createAlbum(t, env.cookie, "Authorization matrix", sourceDir(t, root, "auth-shared"), nil).Id)
	env.putAlbumAudience(t, env.cookie, shared, graceID)

	unshared := uuid.UUID(env.createAlbum(t, env.cookie, "Private", sourceDir(t, root, "auth-private"), nil).Id)
	graceOwn := uuid.UUID(env.createAlbum(t, graceCookie, "Grace's own", sourceDir(t, root, "grace-own"), nil).Id)

	t.Run("ShareeIsForbidden", func(t *testing.T) {
		// A sharee cannot edit the ACL at all -- not even to remove
		// their own id from the set (the share is view-only; the
		// one-row face's departure affordance retired with it).
		rec := env.putAlbumAudienceRaw(t, graceCookie, shared, audienceBody([]uuid.UUID{}))
		requireAlbumWriteForbidden(t, rec)
		if rows := env.shareRowsOf(t, shared); rows != 1 {
			t.Errorf("share rows after the sharee's attempt = %d, want still 1", rows)
		}
	})

	t.Run("StrangerIsNotFound", func(t *testing.T) {
		unknown := fixtureWallAlbumID("ff")
		reference := env.getAlbumsRaw(t, hopperCookie, "/api/v1/albums/"+unknown.String())
		requireAlbumNotFound(t, reference)
		rec := env.putAlbumAudienceRaw(t, hopperCookie, unshared, audienceBody([]uuid.UUID{}))
		requireAlbumNotFound(t, rec)
		if rec.Body.String() != reference.Body.String() {
			t.Fatalf("stranger 404 body = %q, want byte-identical to the unknown id's %q",
				rec.Body.String(), reference.Body.String())
		}
		if rows := env.shareRowsOf(t, unshared); rows != 0 {
			t.Errorf("share rows after the stranger's attempt = %d, want 0", rows)
		}
	})

	t.Run("AdminOverridesOwnerCheck", func(t *testing.T) {
		// The administrator holds no share of Grace's album and is
		// not its owner; the admin role bypasses the owner check.
		report := env.putAlbumAudience(t, env.cookie, graceOwn, hopperID)
		requireAudienceIDs(t, report.Granted, hopperID)
		if rows := env.shareRowsOf(t, graceOwn); rows != 1 {
			t.Errorf("share rows after the admin's write = %d, want 1", rows)
		}
	})

	t.Run("OwnerNeverRemovable", func(t *testing.T) {
		// A set that names the album's owner lands nothing for them
		// (the skip carries the owner reason) and protects nothing:
		// the full-set semantics still converge the rows onto the
		// requested set minus the owner -- hopper's row leaves. The
		// owner's own view never moves through any of it: their
		// access is the ownership row, never a share row.
		report := env.putAlbumAudience(t, env.cookie, graceOwn, graceID)
		if len(report.Skipped) != 1 || uuid.UUID(report.Skipped[0].UserId) != graceID ||
			report.Skipped[0].Reason != apigen.AlbumAudienceSkipReasonOwner {
			t.Errorf("skipped = %v, want the owner alone with the owner reason", report.Skipped)
		}
		requireAudienceIDs(t, audienceIDs(report.Audience))
		requireAudienceIDs(t, report.Revoked, hopperID)
		detail := env.getAlbumDetail(t, graceCookie, graceOwn, "")
		if detail.CallerRole != apigen.AlbumDetailCallerRoleOwner {
			t.Fatalf("the owner's detail callerRole = %q, want owner", detail.CallerRole)
		}
	})
}

// TestUpdateAlbumAudienceAlbumTypes pins the type matrix: every sync
// album answers the single-shape 404 through the audience face
// (whoever owns it -- the timeline is the sync stream's face), the
// directory and virtual albums converge freely, and a locked album
// is never gated (the freeze is a curation gate; an audience member
// only ever reads).
func TestUpdateAlbumAudienceAlbumTypes(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	adminID := env.sessionUserID(t)
	graceID, graceCookie := env.createSecondUser(t, "audience-types-grace@example.com")

	// The caller's OWN sync album: the write face's refusal holds
	// whatever the read admits (the sync album's own detail answers
	// 200 read-only).
	syncID := env.syncAlbumIDOf(t, adminID)
	sync := env.putAlbumAudienceRaw(t, env.cookie, syncID, audienceBody([]uuid.UUID{graceID}))
	requireAlbumNotFound(t, sync)
	unknown := env.putAlbumAudienceRaw(t, env.cookie, fixtureWallAlbumID("ff"), audienceBody([]uuid.UUID{graceID}))
	requireAlbumNotFound(t, unknown)
	if sync.Body.String() != unknown.Body.String() {
		t.Errorf("the own-sync 404 body %q differs from the unknown id's %q",
			sync.Body.String(), unknown.Body.String())
	}
	if rows := env.queryInt(t, "SELECT count(*) FROM album_shares WHERE album_id = $1", pgUUID(syncID)); rows != 0 {
		t.Errorf("sync album share rows = %d, want 0", rows)
	}

	// A directory album: the audience converges like any
	// shareable album's, and the sharee's wall carries the shared
	// directory card.
	root := env.provisionMountRoot(t)
	directory := uuid.UUID(env.createAlbum(t, env.cookie, "Archive", sourceDir(t, root, "archive"), nil).Id)
	report := env.putAlbumAudience(t, env.cookie, directory, graceID)
	requireAudienceIDs(t, report.Granted, graceID)
	wall := env.getAlbumWall(t, graceCookie)
	item, ok := wallAlbumOf(wall, directory)
	if !ok {
		t.Fatalf("sharee wall holds no card for the shared directory album (wall %v)", wall.Albums)
	}
	if item.Type != apigen.AlbumSummaryTypeDirectory {
		t.Errorf("sharee wall card type = %q, want directory", item.Type)
	}

	// A locked album: the audience write passes -- locked never
	// gates this face.
	locked := uuid.UUID(env.createAlbum(t, env.cookie, "Frozen picks", sourceDir(t, root, "frozen"), nil).Id)
	env.patchAlbum(t, env.cookie, locked, `{"locked":true}`)
	lockedReport := env.putAlbumAudience(t, env.cookie, locked, graceID)
	requireAudienceIDs(t, lockedReport.Granted, graceID)
	if revokedAgain := env.putAlbumAudience(t, env.cookie, locked); len(revokedAgain.Revoked) != 1 {
		t.Errorf("the locked album's empty-set removal revoked = %d, want 1 (locked never gates)", len(revokedAgain.Revoked))
	}
}

// TestUpdateAlbumAudienceValidatesBody pins the request's shape
// rules: the field must ride the body even when empty (the explicit
// empty array is the "only me" request; an absent field is a client
// bug), the list is bounded at 200, and the malformed body answers
// the plain validation envelope.
func TestUpdateAlbumAudienceValidatesBody(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	root := env.provisionMountRoot(t)
	albumID := uuid.UUID(env.createAlbum(t, env.cookie, "The body matrix", root, nil).Id)

	// The field must ride the body even when empty (the explicit
	// empty array is the "only me" request; an absent or null field
	// is a client bug the 400 names).
	for _, body := range []string{`{}`, `{"audience":null}`} {
		rec := env.putAlbumAudienceRaw(t, env.cookie, albumID, body)
		requireValidationFields(t, rec, "audience")
	}
	// A body that does not decode names the body itself.
	for _, body := range []string{`not json`, ``} {
		rec := env.putAlbumAudienceRaw(t, env.cookie, albumID, body)
		requireValidationFields(t, rec, "body")
	}

	over := make([]uuid.UUID, 201)
	for i := range over {
		over[i] = uuid.MustParse(fmt.Sprintf("70000000-0000-0000-0005-%012x", i))
	}
	rec := env.putAlbumAudienceRaw(t, env.cookie, albumID, audienceBody(over))
	requireValidationFields(t, rec, "audience")

	// The 200 boundary itself answers -- the unknown ids skip.
	atCap := env.putAlbumAudience(t, env.cookie, albumID, over[:200]...)
	if len(atCap.Skipped) != 200 {
		t.Errorf("skipped at the cap boundary = %d, want 200 (every id names no account)", len(atCap.Skipped))
	}
	if rows := env.shareRowsOf(t, albumID); rows != 0 {
		t.Errorf("share rows = %d, want 0 (nothing landed)", rows)
	}
}

// TestAlbumDetailSharesField pins the owner-only grantee list on the
// album detail: the owner always sees the field (an empty array when
// no grants exist, the full list in grant order otherwise, email
// included), a sharee's raw JSON never carries the key at all.
func TestAlbumDetailSharesField(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	adminID := env.sessionUserID(t)
	graceID, graceCookie := env.createSecondUser(t, "detail-shares-grace@example.com")
	hopperID, _ := env.createSecondUser(t, "detail-shares-hopper@example.com")

	root := env.provisionMountRoot(t)
	created := env.createAlbum(t, env.cookie, "The dialog's album", root, nil)
	albumID := uuid.UUID(created.Id)

	// The owner's field before any grant: present and empty.
	detail := env.getAlbumDetail(t, env.cookie, albumID, "")
	if detail.CallerRole != apigen.AlbumDetailCallerRoleOwner {
		t.Fatalf("owner detail callerRole = %q, want owner", detail.CallerRole)
	}
	if detail.Shares == nil {
		t.Fatal("the owner's detail carries no shares field, want the (empty) list")
	}
	if len(*detail.Shares) != 0 {
		t.Fatalf("the owner's empty shares = %v, want []", *detail.Shares)
	}
	raw := env.getAlbumsRaw(t, env.cookie, "/api/v1/albums/"+albumID.String())
	var rawOwner map[string]json.RawMessage
	if err := json.Unmarshal(raw.Body.Bytes(), &rawOwner); err != nil {
		t.Fatalf("decode owner detail body %q: %v", raw.Body.String(), err)
	}
	if string(rawOwner["shares"]) != "[]" {
		t.Errorf("the owner's raw shares = %s, want []", string(rawOwner["shares"]))
	}

	// Two grants at explicit times: the list answers the grant
	// order, the grantee's email riding each entry.
	first := time.Date(2026, 11, 5, 9, 0, 0, 0, time.UTC)
	second := first.Add(2 * time.Hour)
	env.shareWallAlbum(t, albumID, graceID, adminID, first)
	env.shareWallAlbum(t, albumID, hopperID, adminID, second)

	detail = env.getAlbumDetail(t, env.cookie, albumID, "")
	if detail.Shares == nil {
		t.Fatal("the owner's detail carries no shares field, want the two grants")
	}
	if len(*detail.Shares) != 2 {
		t.Fatalf("the owner's shares = %v, want 2 entries", *detail.Shares)
	}
	grantee, hopperEntry := (*detail.Shares)[0], (*detail.Shares)[1]
	if uuid.UUID(grantee.UserId) != graceID || string(grantee.Email) != "detail-shares-grace@example.com" {
		t.Errorf("shares[0] = %s/%s, want grace %s and her email",
			grantee.UserId, grantee.Email, graceID)
	}
	if !grantee.GrantedAt.Truncate(time.Microsecond).Equal(first) {
		t.Errorf("shares[0].grantedAt = %v, want %v", grantee.GrantedAt, first)
	}
	if uuid.UUID(hopperEntry.UserId) != hopperID || string(hopperEntry.Email) != "detail-shares-hopper@example.com" {
		t.Errorf("shares[1] = %s/%s, want hopper %s and his email",
			hopperEntry.UserId, hopperEntry.Email, hopperID)
	}
	if !hopperEntry.GrantedAt.Truncate(time.Microsecond).Equal(second) {
		t.Errorf("shares[1].grantedAt = %v, want %v", hopperEntry.GrantedAt, second)
	}

	// The sharee: the field is absent in the raw JSON, not null.
	detail = env.getAlbumDetail(t, graceCookie, albumID, "")
	if detail.CallerRole != apigen.AlbumDetailCallerRoleViewer {
		t.Fatalf("sharee detail callerRole = %q, want viewer", detail.CallerRole)
	}
	if detail.Shares != nil {
		t.Errorf("the sharee's typed shares = %v, want nil", *detail.Shares)
	}
	rawSharee := env.getAlbumsRaw(t, graceCookie, "/api/v1/albums/"+albumID.String())
	var rawViewer map[string]json.RawMessage
	if err := json.Unmarshal(rawSharee.Body.Bytes(), &rawViewer); err != nil {
		t.Fatalf("decode sharee detail body %q: %v", rawSharee.Body.String(), err)
	}
	if _, present := rawViewer["shares"]; present {
		t.Errorf("the sharee's raw detail carries a shares key = %s, want the key absent",
			string(rawViewer["shares"]))
	}
}

// TestListInstanceUsers pins the minimal member list: any
// authenticated account (a plain member here) enumerates the
// instance's accounts ordered by email, each entry carrying exactly
// the four identity fields -- no display name, no timestamps.
func TestListInstanceUsers(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	adminID := env.sessionUserID(t)

	// Created out of email order on purpose: the list answers email
	// ASC, not the creation order.
	deltaID, _ := env.createSecondUser(t, "list-delta@example.com")
	bravoID, bravoCookie := env.createSecondUser(t, "list-bravo@example.com")
	charlieID, _ := env.createSecondUser(t, "list-charlie@example.com")
	env.exec(t, "UPDATE users SET is_disabled = true WHERE id = $1", pgUUID(charlieID))

	// A plain member account reads the list (the family-instance
	// posture: any authenticated account may enumerate it).
	rec := env.getAlbumsRaw(t, bravoCookie, "/api/v1/users")
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /users as a plain member status = %d, want %d (body %q)",
			rec.Code, http.StatusOK, rec.Body.String())
	}
	var listed apigen.UserListResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &listed); err != nil {
		t.Fatalf("decode users body %q: %v", rec.Body.String(), err)
	}
	if listed.Users == nil {
		t.Fatal("GET /users body users = null, want an array")
	}
	if len(listed.Users) != 4 {
		t.Fatalf("users = %d entries, want 4 (ada, bravo, charlie, delta; got %v)",
			len(listed.Users), listed.Users)
	}

	// The email-ascending order.
	wantOrder := []string{"ada@example.com", "list-bravo@example.com", "list-charlie@example.com", "list-delta@example.com"}
	for i, want := range wantOrder {
		if string(listed.Users[i].Email) != want {
			t.Errorf("users[%d].email = %s, want %s (order %v)", i, listed.Users[i].Email, want, listed.Users)
		}
	}

	// The honest fields: ada the wizard admin, charlie the disabled
	// member, delta the plain member.
	if uuid.UUID(listed.Users[0].Id) != adminID || listed.Users[0].Role != apigen.UserRoleAdmin || listed.Users[0].IsDisabled {
		t.Errorf("users[0] = %s/%s/%t, want the admin %s role admin, not disabled",
			listed.Users[0].Id, listed.Users[0].Role, listed.Users[0].IsDisabled, adminID)
	}
	if uuid.UUID(listed.Users[1].Id) != bravoID || listed.Users[1].Role != apigen.UserRoleUser || listed.Users[1].IsDisabled {
		t.Errorf("users[1] = %s/%s/%t, want bravo %s role user, not disabled",
			listed.Users[1].Id, listed.Users[1].Role, listed.Users[1].IsDisabled, bravoID)
	}
	if uuid.UUID(listed.Users[2].Id) != charlieID || !listed.Users[2].IsDisabled {
		t.Errorf("users[2] = %s/%t, want charlie %s disabled",
			listed.Users[2].Id, listed.Users[2].IsDisabled, charlieID)
	}
	if uuid.UUID(listed.Users[3].Id) != deltaID || listed.Users[3].IsDisabled {
		t.Errorf("users[3] = %s/%t, want delta %s not disabled",
			listed.Users[3].Id, listed.Users[3].IsDisabled, deltaID)
	}

	// Exactly the four identity fields per entry: no name, no
	// timestamps, nothing else rides the minimal face.
	var rawListed map[string]json.RawMessage
	if err := json.Unmarshal(rec.Body.Bytes(), &rawListed); err != nil {
		t.Fatalf("decode raw users body: %v", err)
	}
	var rawUsers []map[string]json.RawMessage
	if err := json.Unmarshal(rawListed["users"], &rawUsers); err != nil {
		t.Fatalf("decode raw users array: %v", err)
	}
	for i, entry := range rawUsers {
		if len(entry) != 4 {
			t.Errorf("users[%d] carries %d fields (%v), want exactly id/email/role/isDisabled",
				i, len(entry), entry)
		}
		for _, field := range []string{"id", "email", "role", "isDisabled"} {
			if _, present := entry[field]; !present {
				t.Errorf("users[%d] is missing the %s field", i, field)
			}
		}
	}
}

// TestAlbumAudienceEvents pins the two audience events' payload and
// transition-only discipline: exactly {albumId, userId-of-grantee}
// and nothing else (no email, no grant time, no album state), one
// event per real row transition, silence on every idempotent no-op.
func TestAlbumAudienceEvents(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	graceID, _ := env.createSecondUser(t, "audience-events-grace@example.com")
	hopperID, _ := env.createSecondUser(t, "audience-events-hopper@example.com")
	zoeID, _ := env.createSecondUser(t, "audience-events-zoe@example.com")

	root := env.provisionMountRoot(t)
	created := env.createAlbum(t, env.cookie, "The eventful audience", root, nil)
	albumID := uuid.UUID(created.Id)

	// grant two, idempotent repeat, full-set swap (hopper out, zoe
	// in), empty set, re-grant hopper.
	env.putAlbumAudience(t, env.cookie, albumID, graceID, hopperID)
	env.putAlbumAudience(t, env.cookie, albumID, hopperID, graceID) // order-free repeat
	env.putAlbumAudience(t, env.cookie, albumID, graceID, zoeID)    // hopper leaves, zoe lands
	env.putAlbumAudience(t, env.cookie, albumID)                    // only me
	env.putAlbumAudience(t, env.cookie, albumID, hopperID)          // a real re-grant

	shared := eventsOfAlbum(t, env.bus, "album.shared", albumID)
	requireEventUserIds(t, shared, graceID, hopperID, zoeID, hopperID)
	for i, payload := range shared {
		if payload["albumId"] != albumID.String() {
			t.Errorf("album.shared[%d].albumId = %q, want %s", i, payload["albumId"], albumID)
		}
		if len(payload) != 2 {
			t.Errorf("album.shared[%d] payload = %v, want exactly the albumId/userId pair", i, payload)
		}
		for _, banned := range []string{
			"email", "grantedAt", "grantedBy", "title", "description",
			"locked", "type", "itemCount", "isDisabled", "role",
		} {
			if payload[banned] != "" {
				t.Errorf("album.shared[%d] payload carries %q = %q -- private state must not ride the bus",
					i, banned, payload[banned])
			}
		}
	}

	unshared := eventsOfAlbum(t, env.bus, "album.unshared", albumID)
	requireEventUserIds(t, unshared, hopperID, graceID, zoeID)
	for i, payload := range unshared {
		if payload["albumId"] != albumID.String() {
			t.Errorf("album.unshared[%d].albumId = %q, want %s", i, payload["albumId"], albumID)
		}
		if len(payload) != 2 {
			t.Errorf("album.unshared[%d] payload = %v, want exactly the albumId/userId pair", i, payload)
		}
		for _, banned := range []string{
			"email", "grantedAt", "grantedBy", "title", "description",
			"locked", "type", "itemCount", "isDisabled", "role",
		} {
			if payload[banned] != "" {
				t.Errorf("album.unshared[%d] payload carries %q = %q -- private state must not ride the bus",
					i, banned, payload[banned])
			}
		}
	}
}

// wallAlbumOf finds the wall card of one album id, reporting whether
// it exists.
func wallAlbumOf(wall apigen.AlbumWallResponse, albumID uuid.UUID) (apigen.AlbumSummary, bool) {
	for _, item := range wall.Albums {
		if uuid.UUID(item.Id) == albumID {
			return item, true
		}
	}
	return apigen.AlbumSummary{}, false
}
