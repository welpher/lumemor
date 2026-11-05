// Album sharing tests (M3c Task 3): the view-ACL grant and revoke
// behind POST /albums/{albumId}/shares and DELETE
// /albums/{albumId}/shares/{userId}, the owner-only grantee list on
// the album detail, and the instance's minimal member list behind
// GET /users. The fixtures ride the Task 1 read oracle (the wall and
// the detail) for the sharee's visibility, the Task 2 write helpers
// for the album under test, and the bus ring for the
// transition-only discipline of the two share events.
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

// shareBody renders the grant body of one target user.
func shareBody(userID uuid.UUID) string {
	return `{"userId":"` + userID.String() + `"}`
}

// grantAlbumShareRaw fires POST /albums/{albumId}/shares with the raw
// body (a nil cookie sends it unauthenticated).
func (e *assetEnv) grantAlbumShareRaw(t *testing.T, cookie *http.Cookie, albumID uuid.UUID, body string) *httptest.ResponseRecorder {
	t.Helper()
	return e.writeAlbumRaw(t, cookie, http.MethodPost,
		"/api/v1/albums/"+albumID.String()+"/shares", body)
}

// grantAlbumShare fires POST /albums/{albumId}/shares for the target
// user and decodes the 200 body.
func (e *assetEnv) grantAlbumShare(t *testing.T, cookie *http.Cookie, albumID, userID uuid.UUID) apigen.AlbumShareResponse {
	t.Helper()
	rec := e.grantAlbumShareRaw(t, cookie, albumID, shareBody(userID))
	if rec.Code != http.StatusOK {
		t.Fatalf("POST /albums/%s/shares status = %d, want %d (body %q)",
			albumID, rec.Code, http.StatusOK, rec.Body.String())
	}
	var share apigen.AlbumShareResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &share); err != nil {
		t.Fatalf("decode grant share body %q: %v", rec.Body.String(), err)
	}
	return share
}

// revokeAlbumShareRaw fires DELETE /albums/{albumId}/shares/{userId}
// (a nil cookie sends it unauthenticated).
func (e *assetEnv) revokeAlbumShareRaw(t *testing.T, cookie *http.Cookie, albumID, userID uuid.UUID) *httptest.ResponseRecorder {
	t.Helper()
	return e.writeAlbumRaw(t, cookie, http.MethodDelete,
		"/api/v1/albums/"+albumID.String()+"/shares/"+userID.String(), "")
}

// shareRowsOf counts the album's share rows (the grant idempotency
// and the revoke's row-level truth).
func (e *assetEnv) shareRowsOf(t *testing.T, albumID uuid.UUID) int {
	t.Helper()
	return e.queryInt(t, "SELECT count(*) FROM album_shares WHERE album_id = $1", pgUUID(albumID))
}

// requireAlbumShareKey asserts the grant's 400 message-key answer:
// VALIDATION_FAILED carrying exactly the given key in details.key.
func requireAlbumShareKey(t *testing.T, rec *httptest.ResponseRecorder, key string) {
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
		t.Fatalf("details absent, want the %s message key", key)
	}
	if got, _ := (*apiErr.Details)["key"]; got != key {
		t.Errorf("details.key = %v, want %s", got, key)
	}
}

// TestAlbumSharesRequireSession pins the 401 of every sharing face
// and the member list: the session middleware rejects the request
// before any handler runs.
func TestAlbumSharesRequireSession(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	albumID := fixtureWallAlbumID("71")
	target := uuid.MustParse("70000000-0000-0000-0000-000000000001")

	for _, route := range []struct {
		method string
		target string
	}{
		{http.MethodPost, "/api/v1/albums/" + albumID.String() + "/shares"},
		{http.MethodDelete, "/api/v1/albums/" + albumID.String() + "/shares/" + target.String()},
		{http.MethodGet, "/api/v1/users"},
	} {
		rec := env.writeAlbumRaw(t, nil, route.method, route.target, "")
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("%s %s unauthenticated status = %d, want %d",
				route.method, route.target, rec.Code, http.StatusUnauthorized)
		}
	}
}

// TestGrantAlbumShare pins the grant's happy path and its
// idempotency: the owner grants, the sharee's wall and detail answer
// through the Task 1 read oracle, the re-grant answers the existing
// row (the same grantedAt, no second row, no second event).
func TestGrantAlbumShare(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	adminID := env.sessionUserID(t)
	graceID, graceCookie := env.createSecondUser(t, "grant-grace@example.com")

	created := env.createAlbum(t, env.cookie, "The shared album", nil)
	albumID := uuid.UUID(created.Id)
	member := fixtureWallAssetID("c1")
	env.insertTimelineAsset(t, member, adminID, time.Date(2026, 7, 1, 9, 0, 0, 0, time.UTC), timelineAssetOpts{})
	env.addTimelineItem(t, albumID, member, adminID)

	share := env.grantAlbumShare(t, env.cookie, albumID, graceID)
	if uuid.UUID(share.AlbumId) != albumID {
		t.Errorf("share.albumId = %s, want %s", share.AlbumId, albumID)
	}
	if uuid.UUID(share.UserId) != graceID {
		t.Errorf("share.userId = %s, want the grantee %s", share.UserId, graceID)
	}
	if share.GrantedAt.IsZero() {
		t.Error("share.grantedAt = zero, want the grant time")
	}
	if rows := env.shareRowsOf(t, albumID); rows != 1 {
		t.Fatalf("album_shares rows = %d, want 1", rows)
	}

	// The sharee's wall: the album lands in the shared group with the
	// viewer role and the owner attribution (Task 1's oracle).
	wall := env.getAlbumWall(t, graceCookie)
	item, ok := wallAlbumOf(wall, albumID)
	if !ok {
		t.Fatalf("GET /albums as the sharee holds no card for %s (wall %v)", albumID, wall.Albums)
	}
	if item.CallerRole != apigen.AlbumSummaryCallerRoleViewer {
		t.Errorf("sharee wall card callerRole = %q, want viewer", item.CallerRole)
	}
	if item.Owner == nil {
		t.Fatal("sharee wall card owner absent, want the owner attribution")
	}
	if uuid.UUID(item.Owner.Id) != adminID || string(item.Owner.Email) != "ada@example.com" {
		t.Errorf("sharee wall card owner = %s/%s, want %s (the wall lists the album's owner, not the grantee)",
			item.Owner.Id, item.Owner.Email, adminID)
	}
	if item.ItemCount != 1 {
		t.Errorf("sharee wall card itemCount = %d, want 1", item.ItemCount)
	}

	// The sharee's detail: viewer role over the owner's items.
	detail := env.getAlbumDetail(t, graceCookie, albumID, "")
	if detail.CallerRole != apigen.AlbumDetailCallerRoleViewer {
		t.Errorf("sharee detail callerRole = %q, want viewer", detail.CallerRole)
	}
	if len(detail.Items) != 1 || uuid.UUID(detail.Items[0].AssetId) != member {
		t.Errorf("sharee detail items = %v, want the one member %s", detail.Items, member)
	}

	// The idempotent re-grant: the existing row's grantedAt, no
	// second row, no second event.
	again := env.grantAlbumShare(t, env.cookie, albumID, graceID)
	if !again.GrantedAt.Truncate(time.Microsecond).Equal(share.GrantedAt.Truncate(time.Microsecond)) {
		t.Errorf("re-grant grantedAt = %v, want the existing row's %v", again.GrantedAt, share.GrantedAt)
	}
	if rows := env.shareRowsOf(t, albumID); rows != 1 {
		t.Fatalf("album_shares rows after the re-grant = %d, want still 1", rows)
	}
	if shared := busEventsOf(t, env.bus, "album.shared"); len(shared) != 1 {
		t.Errorf("album.shared events = %d, want exactly 1 (the idempotent re-grant stays silent)", len(shared))
	}
	if unshared := busEventsOf(t, env.bus, "album.unshared"); len(unshared) != 0 {
		t.Errorf("album.unshared events = %d, want none", len(unshared))
	}
}

// TestGrantAlbumShareAuthorization pins the grant's two-tier answer:
// a sharee cannot share onward (403), an invisible album folds into
// the single-shape 404 byte-identical to the unknown id, and the
// admin override grants another account's album.
func TestGrantAlbumShareAuthorization(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	graceID, graceCookie := env.createSecondUser(t, "grant-auth-grace@example.com")
	hopperID, hopperCookie := env.createSecondUser(t, "grant-auth-hopper@example.com")

	created := env.createAlbum(t, env.cookie, "Authorization matrix", nil)
	albumID := uuid.UUID(created.Id)
	env.grantAlbumShare(t, env.cookie, albumID, graceID) // grace is a sharee now

	// By a sharee: the share is view-only, sharing onward is the
	// honest authorization failure.
	sharee := env.grantAlbumShareRaw(t, graceCookie, albumID, shareBody(hopperID))
	if sharee.Code != http.StatusForbidden {
		t.Fatalf("grant by sharee status = %d, want %d (body %q)",
			sharee.Code, http.StatusForbidden, sharee.Body.String())
	}
	apiErr, _ := decodeError(t, sharee)
	if apiErr.Code != apigen.AUTHFORBIDDEN {
		t.Errorf("grant by sharee code = %q, want %q", apiErr.Code, apigen.AUTHFORBIDDEN)
	}
	if rows := env.queryInt(t, "SELECT count(*) FROM album_shares WHERE album_id = $1 AND user_id = $2",
		pgUUID(albumID), pgUUID(hopperID)); rows != 0 {
		t.Errorf("share rows for hopper = %d, want 0 (the 403 wrote nothing)", rows)
	}

	// By a stranger: the album is invisible, so the answer is the
	// single-shape 404, byte-identical to the unknown id's.
	stranger := env.grantAlbumShareRaw(t, hopperCookie, albumID, shareBody(graceID))
	requireAlbumNotFound(t, stranger)
	unknown := env.grantAlbumShareRaw(t, env.cookie, fixtureWallAlbumID("ff"), shareBody(graceID))
	requireAlbumNotFound(t, unknown)
	if stranger.Body.String() != unknown.Body.String() {
		t.Errorf("the stranger's 404 body %q differs from the unknown id's %q",
			stranger.Body.String(), unknown.Body.String())
	}

	// No event for either failed write: the setup grant is still the
	// only one on the bus.
	if shared := busEventsOf(t, env.bus, "album.shared"); len(shared) != 1 {
		t.Errorf("album.shared events = %d, want exactly the setup grant", len(shared))
	}

	// By the admin override: another account's unshared album grants
	// fine -- the owner check is what the override bypasses.
	graceAlbum := env.createAlbum(t, graceCookie, "Grace's own", nil)
	overridden := env.grantAlbumShare(t, env.cookie, uuid.UUID(graceAlbum.Id), hopperID)
	if uuid.UUID(overridden.UserId) != hopperID {
		t.Errorf("override grant userId = %s, want hopper %s", overridden.UserId, hopperID)
	}
	hopperWall := env.getAlbumWall(t, hopperCookie)
	if _, ok := wallAlbumOf(hopperWall, uuid.UUID(graceAlbum.Id)); !ok {
		t.Errorf("hopper's wall holds no card for the override-shared album %s (wall %v)",
			graceAlbum.Id, hopperWall.Albums)
	}
}

// TestGrantAlbumShareTargets pins the grant's target rules: the
// album's owner is never a grantee, an unknown and a disabled account
// answer the SAME album_share_user_invalid shape (no user-id
// enumeration through this face), and a malformed body answers the
// plain validation 400. Nothing lands.
func TestGrantAlbumShareTargets(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	adminID := env.sessionUserID(t)

	created := env.createAlbum(t, env.cookie, "Target matrix", nil)
	albumID := uuid.UUID(created.Id)

	t.Run("ShareToOwner", func(t *testing.T) {
		rec := env.grantAlbumShareRaw(t, env.cookie, albumID, shareBody(adminID))
		requireAlbumShareKey(t, rec, "album_share_to_owner")
	})

	t.Run("ShareToUnknownUser", func(t *testing.T) {
		unknownUser := uuid.MustParse("70000000-0000-0000-0000-0000000000ff")
		rec := env.grantAlbumShareRaw(t, env.cookie, albumID, shareBody(unknownUser))
		requireAlbumShareKey(t, rec, "album_share_user_invalid")
	})

	t.Run("ShareToDisabledUser", func(t *testing.T) {
		disabledID, _ := env.createSecondUser(t, "grant-disabled@example.com")
		env.exec(t, "UPDATE users SET is_disabled = true WHERE id = $1", pgUUID(disabledID))
		disabled := env.grantAlbumShareRaw(t, env.cookie, albumID, shareBody(disabledID))
		requireAlbumShareKey(t, disabled, "album_share_user_invalid")

		// The same shape as the unknown account's answer: user ids
		// cannot be probed through this face.
		unknownUser := uuid.MustParse("70000000-0000-0000-0000-0000000000fe")
		unknown := env.grantAlbumShareRaw(t, env.cookie, albumID, shareBody(unknownUser))
		if disabled.Code != unknown.Code {
			t.Fatalf("disabled target status = %d, unknown target status = %d, want equal", disabled.Code, unknown.Code)
		}
		if disabled.Body.String() != unknown.Body.String() {
			t.Errorf("the disabled target's 400 body %q differs from the unknown target's %q",
				disabled.Body.String(), unknown.Body.String())
		}
	})

	t.Run("MalformedBody", func(t *testing.T) {
		for _, body := range []string{`{not json`, `{"title":"no target"}`, ``} {
			rec := env.grantAlbumShareRaw(t, env.cookie, albumID, body)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("grant body %q status = %d, want %d (body %q)",
					body, rec.Code, http.StatusBadRequest, rec.Body.String())
			}
			apiErr, _ := decodeError(t, rec)
			if apiErr.Code != apigen.VALIDATIONFAILED {
				t.Errorf("grant body %q code = %q, want %q", body, apiErr.Code, apigen.VALIDATIONFAILED)
			}
		}
	})

	if rows := env.shareRowsOf(t, albumID); rows != 0 {
		t.Fatalf("album_shares rows = %d, want 0 (no target rule violation landed a row)", rows)
	}
	if shared := busEventsOf(t, env.bus, "album.shared"); len(shared) != 0 {
		t.Errorf("album.shared events = %d, want none (every grant failed)", len(shared))
	}
}

// TestGrantAlbumShareAlbumTypes pins the shareability rules: sync
// albums are never shareable (the single-shape 404 -- the timeline is
// their face), directory and virtual albums are, and a locked album
// grants and revokes freely -- the freeze is a curation gate, and a
// viewer only ever reads.
func TestGrantAlbumShareAlbumTypes(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	adminID := env.sessionUserID(t)
	graceID, graceCookie := env.createSecondUser(t, "grant-types-grace@example.com")

	// A sync album: not an album resource, whatever its owner.
	syncID := env.syncAlbumIDOf(t, adminID)
	sync := env.grantAlbumShareRaw(t, env.cookie, syncID, shareBody(graceID))
	requireAlbumNotFound(t, sync)
	unknown := env.grantAlbumShareRaw(t, env.cookie, fixtureWallAlbumID("ff"), shareBody(graceID))
	if sync.Body.String() != unknown.Body.String() {
		t.Errorf("the sync grant's 404 body %q differs from the unknown id's %q",
			sync.Body.String(), unknown.Body.String())
	}

	// A directory album: shareable per the pin.
	directoryID := fixtureWallAlbumID("72")
	env.insertWallAlbum(t, directoryID, adminID, "directory", "Archive", wallAlbumOpts{})
	share := env.grantAlbumShare(t, env.cookie, directoryID, graceID)
	if uuid.UUID(share.UserId) != graceID {
		t.Errorf("directory grant userId = %s, want %s", share.UserId, graceID)
	}
	wall := env.getAlbumWall(t, graceCookie)
	item, ok := wallAlbumOf(wall, directoryID)
	if !ok {
		t.Fatalf("sharee wall holds no card for the shared directory album (wall %v)", wall.Albums)
	}
	if item.Type != apigen.AlbumSummaryTypeDirectory {
		t.Errorf("sharee wall card type = %q, want directory", item.Type)
	}

	// A locked album: grant and revoke both pass -- locked never
	// gates the sharing face.
	lockedID := fixtureWallAlbumID("73")
	env.insertWallAlbum(t, lockedID, adminID, "virtual", "Frozen picks", wallAlbumOpts{locked: true})
	locked := env.grantAlbumShare(t, env.cookie, lockedID, graceID)
	if uuid.UUID(locked.UserId) != graceID {
		t.Errorf("locked grant userId = %s, want %s", locked.UserId, graceID)
	}
	if revoked := env.revokeAlbumShareRaw(t, env.cookie, lockedID, graceID); revoked.Code != http.StatusNoContent {
		t.Fatalf("revoke on a locked album status = %d, want %d", revoked.Code, http.StatusNoContent)
	}
}

// TestRevokeAlbumShare pins the revoke: the owner removes a grantee,
// the grantee's wall and detail lose the album, another grantee is
// untouched, the already-gone revoke answers the idempotent 204, and
// the events fire exactly on the real transitions.
func TestRevokeAlbumShare(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	graceID, graceCookie := env.createSecondUser(t, "revoke-grace@example.com")
	hopperID, hopperCookie := env.createSecondUser(t, "revoke-hopper@example.com")

	created := env.createAlbum(t, env.cookie, "The revoked album", nil)
	albumID := uuid.UUID(created.Id)
	env.grantAlbumShare(t, env.cookie, albumID, graceID)
	env.grantAlbumShare(t, env.cookie, albumID, hopperID)
	if rows := env.shareRowsOf(t, albumID); rows != 2 {
		t.Fatalf("album_shares rows = %d, want 2 (the fixture is broken)", rows)
	}

	// The owner revokes grace's share.
	rec := env.revokeAlbumShareRaw(t, env.cookie, albumID, graceID)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("revoke status = %d, want %d (body %q)",
			rec.Code, http.StatusNoContent, rec.Body.String())
	}
	if rows := env.shareRowsOf(t, albumID); rows != 1 {
		t.Fatalf("album_shares rows after the revoke = %d, want 1", rows)
	}

	// The sharee's visibility is gone: no wall card, no detail.
	wall := env.getAlbumWall(t, graceCookie)
	if _, ok := wallAlbumOf(wall, albumID); ok {
		t.Errorf("grace's wall still holds the revoked album %s (wall %v)", albumID, wall.Albums)
	}
	detail := env.getAlbumsRaw(t, graceCookie, "/api/v1/albums/"+albumID.String())
	requireAlbumNotFound(t, detail)

	// The other grantee is untouched.
	hopperWall := env.getAlbumWall(t, hopperCookie)
	if _, ok := wallAlbumOf(hopperWall, albumID); !ok {
		t.Errorf("hopper's wall lost the album although his share is intact (wall %v)", hopperWall.Albums)
	}

	// The idempotent revoke: the already-gone row answers 204 again.
	if again := env.revokeAlbumShareRaw(t, env.cookie, albumID, graceID); again.Code != http.StatusNoContent {
		t.Fatalf("the idempotent revoke status = %d, want %d", again.Code, http.StatusNoContent)
	}
	if rows := env.shareRowsOf(t, albumID); rows != 1 {
		t.Fatalf("album_shares rows after the idempotent revoke = %d, want still 1", rows)
	}

	// The events: one album.unshared (the real revoke), the
	// idempotent one silent; a re-grant after the revoke is a real
	// transition again (a new row fires album.shared).
	if unshared := busEventsOf(t, env.bus, "album.unshared"); len(unshared) != 1 {
		t.Fatalf("album.unshared events = %d, want exactly 1 (the idempotent revoke stays silent)", len(unshared))
	}
	if shared := busEventsOf(t, env.bus, "album.shared"); len(shared) != 2 {
		t.Fatalf("album.shared events = %d, want 2 (the setup grants)", len(shared))
	}
	env.grantAlbumShare(t, env.cookie, albumID, graceID)
	if shared := busEventsOf(t, env.bus, "album.shared"); len(shared) != 3 {
		t.Errorf("album.shared events after the re-grant = %d, want 3 (a real new row fires)", len(shared))
	}
	if unshared := busEventsOf(t, env.bus, "album.unshared"); len(unshared) != 1 {
		t.Errorf("album.unshared events after the re-grant = %d, want still 1", len(unshared))
	}
}

// TestRevokeAlbumShareAuthorization pins the revoke's caller matrix:
// a sharee may delete their OWN row (leaving is the sharee's right,
// the same endpoint as the owner's revocation) but cannot revoke
// another account's grant (403); a stranger's invisible album folds
// into the single-shape 404.
func TestRevokeAlbumShareAuthorization(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	graceID, graceCookie := env.createSecondUser(t, "revoke-auth-grace@example.com")
	hopperID, _ := env.createSecondUser(t, "revoke-auth-hopper@example.com")
	_, zoeCookie := env.createSecondUser(t, "revoke-auth-zoe@example.com")

	created := env.createAlbum(t, env.cookie, "Departure matrix", nil)
	albumID := uuid.UUID(created.Id)
	env.grantAlbumShare(t, env.cookie, albumID, graceID)
	env.grantAlbumShare(t, env.cookie, albumID, hopperID)

	// A sharee cannot revoke ANOTHER account's grant.
	foreign := env.revokeAlbumShareRaw(t, graceCookie, albumID, hopperID)
	if foreign.Code != http.StatusForbidden {
		t.Fatalf("sharee revoking another's grant status = %d, want %d (body %q)",
			foreign.Code, http.StatusForbidden, foreign.Body.String())
	}
	apiErr, _ := decodeError(t, foreign)
	if apiErr.Code != apigen.AUTHFORBIDDEN {
		t.Errorf("sharee revoking another's grant code = %q, want %q", apiErr.Code, apigen.AUTHFORBIDDEN)
	}
	if rows := env.queryInt(t, "SELECT count(*) FROM album_shares WHERE album_id = $1 AND user_id = $2",
		pgUUID(albumID), pgUUID(hopperID)); rows != 1 {
		t.Errorf("hopper's share rows = %d, want 1 (the 403 wrote nothing)", rows)
	}

	// The sharee leaves: their own row, their own right, the same
	// endpoint the owner uses.
	departure := env.revokeAlbumShareRaw(t, graceCookie, albumID, graceID)
	if departure.Code != http.StatusNoContent {
		t.Fatalf("sharee departure status = %d, want %d (body %q)",
			departure.Code, http.StatusNoContent, departure.Body.String())
	}
	if rows := env.queryInt(t, "SELECT count(*) FROM album_shares WHERE album_id = $1 AND user_id = $2",
		pgUUID(albumID), pgUUID(graceID)); rows != 0 {
		t.Fatalf("grace's share rows after the departure = %d, want 0", rows)
	}
	wall := env.getAlbumWall(t, graceCookie)
	if _, ok := wallAlbumOf(wall, albumID); ok {
		t.Errorf("grace's wall still holds the departed album %s (wall %v)", albumID, wall.Albums)
	}
	departed := busEventsOf(t, env.bus, "album.unshared")
	if len(departed) != 1 {
		t.Fatalf("album.unshared events = %d, want exactly 1 (the departure)", len(departed))
	}
	if departed[0]["userId"] != graceID.String() {
		t.Errorf("album.unshared userId = %q, want the departed grantee %s", departed[0]["userId"], graceID)
	}

	// A stranger: the album is invisible, the single-shape 404
	// byte-identical to the unknown id's.
	stranger := env.revokeAlbumShareRaw(t, zoeCookie, albumID, hopperID)
	requireAlbumNotFound(t, stranger)
	unknown := env.revokeAlbumShareRaw(t, env.cookie, fixtureWallAlbumID("ff"), hopperID)
	requireAlbumNotFound(t, unknown)
	if stranger.Body.String() != unknown.Body.String() {
		t.Errorf("the stranger's 404 body %q differs from the unknown id's %q",
			stranger.Body.String(), unknown.Body.String())
	}
	if rows := env.queryInt(t, "SELECT count(*) FROM album_shares WHERE album_id = $1 AND user_id = $2",
		pgUUID(albumID), pgUUID(hopperID)); rows != 1 {
		t.Errorf("hopper's share rows after the stranger's attempt = %d, want still 1", rows)
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

	created := env.createAlbum(t, env.cookie, "The dialog's album", nil)
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

// TestAlbumShareEvents pins the two share events' payload and
// transition-only discipline: exactly {albumId, userId-of-grantee}
// and nothing else (no email, no grant time, no album state), one
// event per real row transition, silence on every idempotent no-op.
func TestAlbumShareEvents(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	graceID, graceCookie := env.createSecondUser(t, "share-events-grace@example.com")
	hopperID, _ := env.createSecondUser(t, "share-events-hopper@example.com")

	created := env.createAlbum(t, env.cookie, "The eventful share", nil)
	albumID := uuid.UUID(created.Id)

	// grant, idempotent re-grant, second grant, revoke, idempotent
	// re-revoke, re-grant (a real new row).
	env.grantAlbumShare(t, env.cookie, albumID, graceID)
	env.grantAlbumShare(t, env.cookie, albumID, graceID)
	env.grantAlbumShare(t, env.cookie, albumID, hopperID)
	if rec := env.revokeAlbumShareRaw(t, env.cookie, albumID, hopperID); rec.Code != http.StatusNoContent {
		t.Fatalf("revoke hopper status = %d, want %d", rec.Code, http.StatusNoContent)
	}
	if rec := env.revokeAlbumShareRaw(t, env.cookie, albumID, hopperID); rec.Code != http.StatusNoContent {
		t.Fatalf("the idempotent re-revoke status = %d, want %d", rec.Code, http.StatusNoContent)
	}
	env.grantAlbumShare(t, env.cookie, albumID, hopperID)

	// The sharee's own departure: the same event name, the departed
	// grantee's id.
	if rec := env.revokeAlbumShareRaw(t, graceCookie, albumID, graceID); rec.Code != http.StatusNoContent {
		t.Fatalf("sharee departure status = %d, want %d", rec.Code, http.StatusNoContent)
	}

	shared := busEventsOf(t, env.bus, "album.shared")
	if len(shared) != 3 {
		t.Fatalf("album.shared events = %d, want 3 (grace, hopper, hopper's re-grant -- the idempotent re-grant stays silent)", len(shared))
	}
	wantGrantees := []uuid.UUID{graceID, hopperID, hopperID}
	for i, payload := range shared {
		if payload["albumId"] != albumID.String() {
			t.Errorf("album.shared[%d].albumId = %q, want %s", i, payload["albumId"], albumID)
		}
		if i < len(wantGrantees) && payload["userId"] != wantGrantees[i].String() {
			t.Errorf("album.shared[%d].userId = %q, want the grantee %s", i, payload["userId"], wantGrantees[i])
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

	unshared := busEventsOf(t, env.bus, "album.unshared")
	if len(unshared) != 2 {
		t.Fatalf("album.unshared events = %d, want 2 (hopper's revoke, grace's departure; the idempotent re-revoke silent)", len(unshared))
	}
	wantRevoked := []uuid.UUID{hopperID, graceID}
	for i, payload := range unshared {
		if payload["albumId"] != albumID.String() {
			t.Errorf("album.unshared[%d].albumId = %q, want %s", i, payload["albumId"], albumID)
		}
		if payload["userId"] != wantRevoked[i].String() {
			t.Errorf("album.unshared[%d].userId = %q, want the revoked grantee %s", i, payload["userId"], wantRevoked[i])
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
