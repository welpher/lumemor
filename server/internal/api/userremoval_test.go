// The account disposal tests: the refusals of DELETE
// /admin/users/{id}, the transfer arm's settlement (the stream fold,
// the photo and container handover, the grant and authorship rows,
// the label merge, the duplicate fold), the trash arm's walk into the
// caller's trash (the retention countdown and the restore it
// promises, the external file move), and the transaction's atomicity
// (a failure rolls the whole disposal back, file moves included).
package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	apigen "github.com/welpher/lumemor/server/internal/api/gen"
	"github.com/welpher/lumemor/server/internal/events"
)

// deleteUser fires the disposal request at the router with the given
// JSON body.
func (e *assetEnv) deleteUser(t *testing.T, cookie *http.Cookie, id uuid.UUID, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/admin/users/"+id.String(), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if cookie != nil {
		req.AddCookie(cookie)
	}
	rec := httptest.NewRecorder()
	e.router.ServeHTTP(rec, req)
	return rec
}

// transferBody renders one transfer-arm disposition.
func transferBody(t *testing.T, receiver uuid.UUID) string {
	t.Helper()
	body, err := json.Marshal(map[string]string{"transferTo": receiver.String()})
	if err != nil {
		t.Fatalf("marshal transfer body: %v", err)
	}
	return string(body)
}

// trashBody renders one trash-arm disposition.
func trashBody() string {
	return `{"dispose":"trash"}`
}

// removalAssetOpts carries the fixture values the fold's cases vary:
// the checksum the storage's uniqueness rule keys on, the state, the
// storage kind, the Live Photo link and the device identity.
type removalAssetOpts struct {
	checksum  string
	component bool
	external  bool
	// deletedAt stamps the row's trash mark; the zero value keeps the
	// row live (and the fixture's own clock out of the assertion).
	deletedAt time.Time
	livePhoto uuid.UUID
	deviceID  string
	filePath  string
}

// insertRemovalAsset inserts one fixture row with an EXPLICIT checksum:
// a collision is two rows over the SAME content, which
// insertTimelineAsset's per-id checksum cannot express.
func (e *assetEnv) insertRemovalAsset(t *testing.T, id, owner uuid.UUID, opts removalAssetOpts) {
	t.Helper()
	path := opts.filePath
	if path == "" {
		path = "originals/fixtures/" + id.String()
	}
	storageKind := "blob"
	if opts.external {
		storageKind = "external"
	}
	mediaKind, mime, name := "image", "image/jpeg", "fixture-"+id.String()+".jpg"
	if opts.component {
		mediaKind, mime, name = "video", "video/quicktime", "fixture-"+id.String()+".mov"
	}
	var deletedAt, readyAt, deviceID, livePhoto any
	if !opts.deletedAt.IsZero() {
		deletedAt = opts.deletedAt
	}
	if !opts.component {
		readyAt = time.Now()
	}
	if opts.deviceID != "" {
		deviceID = opts.deviceID
	}
	if opts.livePhoto != uuid.Nil {
		livePhoto = pgUUID(opts.livePhoto)
	}
	if _, err := e.pool.Exec(context.Background(), `
		INSERT INTO assets (
			id, owner_id, storage_kind, file_path, checksum, original_filename,
			file_size, mime_type, media_kind, width, height, taken_at,
			taken_at_source, is_component, deleted_at, ready_at, pending_jobs,
			device_asset_id, live_photo_video_id
		) VALUES (
			$1, $2, $3, $4, $5, $6, 1024, $7, $8, 40, 30,
			now() - interval '1 hour', 'import', $9, $10, $11, 0, $12, $13
		)`,
		pgUUID(id), pgUUID(owner), storageKind, path, opts.checksum, name,
		mime, mediaKind, opts.component, deletedAt, readyAt, deviceID, livePhoto); err != nil {
		t.Fatalf("insert fixture asset %s: %v", id, err)
	}
}

// requireAssetDeleted asserts the fold's event discipline: exactly one
// asset.deleted, naming the row the fold purged (a purged Live Photo's
// component stays silent -- no view ever listed it).
func requireAssetDeleted(t *testing.T, bus *events.Bus, want uuid.UUID) {
	t.Helper()
	published := busEventsOf(t, bus, "asset.deleted")
	if len(published) != 1 {
		t.Fatalf("asset.deleted events = %d, want 1", len(published))
	}
	if got := published[0]["assetId"]; got != want.String() {
		t.Errorf("asset.deleted names %s, want the folded row %s", got, want)
	}
}

// newMember seeds one plain account with its own stream and a live
// session -- the shape the create endpoint provisions, built directly
// so the disposal tests own every row they assert.
func (e *assetEnv) newMember(t *testing.T, email string) (uuid.UUID, *http.Cookie) {
	t.Helper()
	id, cookie := e.createSecondUser(t, email)
	e.exec(t, `INSERT INTO albums (id, owner_id, title, type) VALUES ($1, $2, 'Sync', 'sync')`,
		pgUUID(uuid.New()), pgUUID(id))
	return id, cookie
}

// requireStatus asserts one recorded response's status and, when the
// request failed, the envelope code.
func requireStatus(t *testing.T, rec *httptest.ResponseRecorder, want int, code apigen.ErrorCode) {
	t.Helper()
	if rec.Code != want {
		t.Fatalf("status = %d, want %d (body %q)", rec.Code, want, rec.Body.String())
	}
	if code == "" {
		return
	}
	apiErr, _ := decodeError(t, rec)
	if apiErr.Code != code {
		t.Errorf("code = %q, want %q (body %q)", apiErr.Code, code, rec.Body.String())
	}
}

// TestDeleteUserRefusals pins the guards: the role gate, the payload
// shape, the receiver rules, the last-administrator line and the
// caller's own account.
func TestDeleteUserRefusals(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	graceID, graceCookie := env.newMember(t, "grace@example.com")
	adminID := env.sessionUserID(t)

	t.Run("RejectsUnauthenticated", func(t *testing.T) {
		rec := env.deleteUser(t, nil, graceID, trashBody())
		requireStatus(t, rec, http.StatusUnauthorized, apigen.AUTHINVALIDCREDENTIALS)
	})

	t.Run("RejectsNonAdmin", func(t *testing.T) {
		rec := env.deleteUser(t, graceCookie, adminID, trashBody())
		requireStatus(t, rec, http.StatusForbidden, apigen.AUTHFORBIDDEN)
	})

	t.Run("UnknownAccount", func(t *testing.T) {
		rec := env.deleteUser(t, env.cookie, uuid.New(), trashBody())
		requireStatus(t, rec, http.StatusNotFound, apigen.USERNOTFOUND)
	})

	t.Run("MalformedBody", func(t *testing.T) {
		rec := env.deleteUser(t, env.cookie, graceID, `{`)
		requireStatus(t, rec, http.StatusBadRequest, apigen.VALIDATIONFAILED)
	})

	t.Run("BothArms", func(t *testing.T) {
		body := `{"transferTo":"` + adminID.String() + `","dispose":"trash"}`
		rec := env.deleteUser(t, env.cookie, graceID, body)
		requireStatus(t, rec, http.StatusBadRequest, apigen.VALIDATIONFAILED)
		if fields := envelopeFields(t, rec); len(fields) != 2 {
			t.Errorf("fields = %v, want both disposition fields", fields)
		}
	})

	t.Run("NoArm", func(t *testing.T) {
		rec := env.deleteUser(t, env.cookie, graceID, `{}`)
		requireStatus(t, rec, http.StatusBadRequest, apigen.VALIDATIONFAILED)
	})

	t.Run("UnknownDisposeValue", func(t *testing.T) {
		rec := env.deleteUser(t, env.cookie, graceID, `{"dispose":"purge"}`)
		requireStatus(t, rec, http.StatusBadRequest, apigen.VALIDATIONFAILED)
	})

	t.Run("UnknownReceiver", func(t *testing.T) {
		rec := env.deleteUser(t, env.cookie, graceID, transferBody(t, uuid.New()))
		requireStatus(t, rec, http.StatusBadRequest, apigen.VALIDATIONFAILED)
		if fields := envelopeFields(t, rec); len(fields) != 1 || fields[0] != "transferTo" {
			t.Errorf("fields = %v, want [transferTo]", fields)
		}
	})

	t.Run("ReceiverIsTheAccount", func(t *testing.T) {
		rec := env.deleteUser(t, env.cookie, graceID, transferBody(t, graceID))
		requireStatus(t, rec, http.StatusConflict, apigen.USERTRANSFERTOSELF)
	})

	t.Run("LastAdministrator", func(t *testing.T) {
		// The instance's only administrator is necessarily the caller:
		// the narrower refusal (the instance would be left without one)
		// answers before the self-delete guard.
		rec := env.deleteUser(t, env.cookie, adminID, trashBody())
		requireStatus(t, rec, http.StatusConflict, apigen.USERLASTADMIN)
	})

	t.Run("SelfDeleteWithAnotherAdministrator", func(t *testing.T) {
		// A second administrator makes the instance safe; the caller's
		// own account is still refused (the disposal hands content to
		// a survivor, and the caller performs the handover).
		role := apigen.UserRoleAdmin
		req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/users",
			strings.NewReader(adminCreateUserBody(t, "kid@example.com", testPassword, nil, &role)))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(env.cookie)
		created := httptest.NewRecorder()
		env.router.ServeHTTP(created, req)
		if created.Code != http.StatusCreated {
			t.Fatalf("create second admin: status = %d (body %q)", created.Code, created.Body.String())
		}
		rec := env.deleteUser(t, env.cookie, adminID, trashBody())
		requireStatus(t, rec, http.StatusBadRequest, apigen.VALIDATIONFAILED)
		if fields := envelopeFields(t, rec); len(fields) != 1 || fields[0] != "id" {
			t.Errorf("fields = %v, want [id]", fields)
		}
	})

	t.Run("AnotherAdministratorIsDeletable", func(t *testing.T) {
		// Two administrators exist, so removing one keeps the surface
		// usable: the guard counts, it does not simply forbid admins.
		var kidID pgtype.UUID
		if err := env.pool.QueryRow(context.Background(),
			"SELECT id FROM users WHERE email = 'kid@example.com'").Scan(&kidID); err != nil {
			t.Fatalf("resolve the second administrator: %v", err)
		}
		rec := env.deleteUser(t, env.cookie, uuid.UUID(kidID.Bytes), transferBody(t, adminID))
		requireStatus(t, rec, http.StatusNoContent, "")
		if got := env.queryInt(t, "SELECT count(*) FROM users WHERE id = $1", kidID); got != 0 {
			t.Errorf("the deleted account survived the request")
		}
		if got := env.queryInt(t, "SELECT count(*) FROM users WHERE role = 'admin'"); got != 1 {
			t.Errorf("administrators after the disposal = %d, want 1", got)
		}
	})
}

// envelopeFields answers the field names a failure envelope carries.
func envelopeFields(t *testing.T, rec *httptest.ResponseRecorder) []string {
	t.Helper()
	apiErr, fields := decodeError(t, rec)
	if apiErr.Code == "" {
		t.Fatalf("the response carries no error envelope (body %q)", rec.Body.String())
	}
	return fields
}

// TestDeleteUserTransferSettlesContent pins the transfer arm's whole
// settlement in one disposal: the photos and containers change hands,
// the departing stream folds into the receiver's, the grants and the
// authorship ride the content, the labels merge, and the rows that
// live with the account alone are gone.
func TestDeleteUserTransferSettlesContent(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	graceID, graceCookie := env.newMember(t, "grace@example.com")
	adminID := env.sessionUserID(t)
	adminSync := env.syncAlbumIDOf(t, adminID)
	graceSync := env.syncAlbumIDOf(t, graceID)

	// The departing account's content: one live photo in its stream,
	// one already-trashed photo, one memory album holding the live
	// photo, and one directory album holding an external member.
	liveID := uuid.MustParse("40000000-0000-0000-0000-000000000001")
	trashedID := uuid.MustParse("40000000-0000-0000-0000-000000000002")
	memoryID := uuid.MustParse("40000000-0000-0000-0000-000000000003")
	directoryID := uuid.MustParse("40000000-0000-0000-0000-000000000004")
	sourceDir := t.TempDir()
	env.insertRemovalDirectory(t, directoryID, graceID, sourceDir)
	externalID := env.insertRemovalMember(t, directoryID, graceID, sourceDir, "grace-0001.jpg",
		[]byte("grace external bytes"), time.Now().Add(-time.Hour))
	env.insertTimelineAsset(t, liveID, graceID, time.Now().Add(-2*time.Hour), timelineAssetOpts{ready: true, width: 40, height: 30})
	env.insertTimelineAsset(t, trashedID, graceID, time.Now().Add(-3*time.Hour), timelineAssetOpts{deleted: true, width: 40, height: 30})
	env.addTimelineItem(t, graceSync, liveID, graceID)
	env.insertTimelineAlbum(t, memoryID, graceID, "memory")
	env.addTimelineItem(t, memoryID, liveID, graceID)

	// The receiver's own album carrying the departing account as a
	// sharee, and the departing account's album carrying a grant.
	adminAlbumID := uuid.MustParse("40000000-0000-0000-0000-000000000005")
	env.insertTimelineAlbum(t, adminAlbumID, adminID, "memory")
	env.shareTimelineAlbum(t, adminAlbumID, graceID, adminID)
	env.shareTimelineAlbum(t, memoryID, adminID, graceID)

	// The per-user rows: a favorite of the receiver's photo, a
	// timeline preference, a private face name, a session (seeded by
	// newMember) plus an API key and a pairing token.
	env.exec(t, "INSERT INTO user_favorites (user_id, asset_id) VALUES ($1, $2)", pgUUID(graceID), pgUUID(liveID))
	env.setTimelinePref(t, graceID, adminAlbumID, false)
	env.insertPeopleModel(t)
	faceID := uuid.MustParse("40000000-0000-0000-0000-000000000006")
	clusterID := uuid.MustParse("40000000-0000-0000-0000-000000000007")
	env.insertFace(t, faceID, liveID, 0.9)
	env.insertFaceCluster(t, clusterID, faceID)
	env.assignFaceCluster(t, faceID, clusterID)
	env.labelPersonCluster(t, graceID, clusterID, "Grandma")
	env.exec(t, "INSERT INTO api_keys (id, user_id, name, key_hash) VALUES ($1, $2, 'phone', 'hash-grace')",
		pgUUID(uuid.New()), pgUUID(graceID))
	env.exec(t, "INSERT INTO pairing_tokens (id, token_hash, created_by, expires_at) VALUES ($1, 'pair-grace', $2, now() + interval '1 hour')",
		pgUUID(uuid.New()), pgUUID(graceID))

	// The labels: a name both accounts carry (the merge) and one only
	// the departing account carries (the handover).
	sharedTagID := uuid.MustParse("40000000-0000-0000-0000-000000000008")
	adminTagID := uuid.MustParse("40000000-0000-0000-0000-000000000009")
	tripsTagID := uuid.MustParse("40000000-0000-0000-0000-00000000000a")
	adminAssetID := uuid.MustParse("40000000-0000-0000-0000-00000000000b")
	env.insertTimelineAsset(t, adminAssetID, adminID, time.Now().Add(-4*time.Hour), timelineAssetOpts{ready: true, width: 40, height: 30})
	env.addTimelineItem(t, adminSync, adminAssetID, adminID)
	env.exec(t, "INSERT INTO tags (id, owner_id, name) VALUES ($1, $2, 'family'), ($3, $2, 'trips')",
		pgUUID(sharedTagID), pgUUID(graceID), pgUUID(tripsTagID))
	env.exec(t, "INSERT INTO tags (id, owner_id, name) VALUES ($1, $2, 'family')", pgUUID(adminTagID), pgUUID(adminID))
	env.exec(t, "INSERT INTO asset_tags (asset_id, tag_id) VALUES ($1, $2), ($1, $3), ($4, $5)",
		pgUUID(liveID), pgUUID(sharedTagID), pgUUID(tripsTagID), pgUUID(adminAssetID), pgUUID(adminTagID))
	// The departing account's favorite of the receiver's photo, and
	// the receiver's favorite of the departing photo (which rides the
	// content and stays).
	env.exec(t, "INSERT INTO user_favorites (user_id, asset_id) VALUES ($1, $2)",
		pgUUID(graceID), pgUUID(adminAssetID))
	env.exec(t, "INSERT INTO user_favorites (user_id, asset_id) VALUES ($1, $2)",
		pgUUID(adminID), pgUUID(liveID))

	rec := env.deleteUser(t, env.cookie, graceID, transferBody(t, adminID))
	requireStatus(t, rec, http.StatusNoContent, "")

	// The account is gone and its credential can never authenticate
	// again: the session row cascaded, so the token resolves to
	// nothing.
	if got := env.queryInt(t, "SELECT count(*) FROM users WHERE id = $1", pgUUID(graceID)); got != 0 {
		t.Errorf("the account row survived")
	}
	if rec := env.getTimelineRaw(t, graceCookie, ""); rec.Code != http.StatusUnauthorized {
		t.Errorf("the deleted account's session answered %d, want 401", rec.Code)
	}

	// THE PHOTOS AND CONTAINERS: every row changed hands, trashed rows
	// kept their mark, and the containers moved whole.
	for _, id := range []uuid.UUID{liveID, externalID} {
		if owner := env.queryText(t, "SELECT owner_id::text FROM assets WHERE id = $1", pgUUID(id)); owner != adminID.String() {
			t.Errorf("asset %s owner = %s, want the receiver %s", id, owner, adminID)
		}
	}
	if deleted := env.queryInt(t, "SELECT count(*) FROM assets WHERE id = $1 AND deleted_at IS NOT NULL", pgUUID(trashedID)); deleted != 1 {
		t.Errorf("the trashed photo lost its mark across the handover")
	}
	for _, id := range []uuid.UUID{memoryID, directoryID} {
		if owner := env.queryText(t, "SELECT owner_id::text FROM albums WHERE id = $1", pgUUID(id)); owner != adminID.String() {
			t.Errorf("album %s owner = %s, want the receiver %s", id, owner, adminID)
		}
	}

	// THE STREAM FOLD: the departing stream is gone, its memberships
	// are the receiver's (authored by the receiver).
	if got := env.queryInt(t, "SELECT count(*) FROM albums WHERE id = $1", pgUUID(graceSync)); got != 0 {
		t.Errorf("the departing stream survived the fold")
	}
	if got := env.queryInt(t,
		"SELECT count(*) FROM album_items WHERE album_id = $1 AND asset_id = $2 AND added_by = $3",
		pgUUID(adminSync), pgUUID(liveID), pgUUID(adminID)); got != 1 {
		t.Errorf("the folded membership is not in the receiver's stream")
	}
	// The person's photos are visible in the receiver's timeline.
	items := env.getTimeline(t, env.cookie, "")
	found := false
	for _, item := range items.Items {
		if uuid.UUID(item.Id) == liveID {
			found = true
		}
	}
	if !found {
		t.Errorf("the transferred photo is absent from the receiver's timeline")
	}

	// THE SHARES: the grant held BY the departing account is gone; the
	// grant it ISSUED rides the container to the new owner (and the
	// grantee keeps the album).
	if got := env.queryInt(t, "SELECT count(*) FROM album_shares WHERE album_id = $1 AND user_id = $2",
		pgUUID(adminAlbumID), pgUUID(graceID)); got != 0 {
		t.Errorf("the departing account's held share survived")
	}
	if grantedBy := env.queryText(t, "SELECT granted_by::text FROM album_shares WHERE album_id = $1 AND user_id = $2",
		pgUUID(memoryID), pgUUID(adminID)); grantedBy != adminID.String() {
		t.Errorf("the issued grant's author = %s, want the receiver %s", grantedBy, adminID)
	}

	// THE LABELS: one 'family' tag survives, carrying both photos; the
	// departing account's own 'trips' tag changed hands with its photo.
	if got := env.queryInt(t, "SELECT count(*) FROM tags WHERE name = 'family'"); got != 1 {
		t.Errorf("'family' tags after the merge = %d, want 1", got)
	}
	if got := env.queryInt(t, "SELECT count(*) FROM tags WHERE owner_id = $1", pgUUID(graceID)); got != 0 {
		t.Errorf("the departing account still owns tags")
	}
	for _, id := range []uuid.UUID{liveID, adminAssetID} {
		if got := env.queryInt(t,
			"SELECT count(*) FROM asset_tags WHERE asset_id = $1 AND tag_id IN (SELECT id FROM tags WHERE name = 'family')",
			pgUUID(id)); got != 1 {
			t.Errorf("photo %s lost its 'family' label across the merge", id)
		}
	}
	if owner := env.queryText(t, "SELECT owner_id::text FROM tags WHERE id = $1", pgUUID(tripsTagID)); owner != adminID.String() {
		t.Errorf("the 'trips' tag owner = %s, want the receiver", owner)
	}

	// THE ROWS THAT LIVE WITH THE ACCOUNT: favorites, preferences and
	// the private face names die; the receiver's own favorites stay.
	if got := env.queryInt(t, "SELECT count(*) FROM user_favorites WHERE user_id = $1", pgUUID(graceID)); got != 0 {
		t.Errorf("the departing account's favorites survived")
	}
	if got := env.queryInt(t, "SELECT count(*) FROM user_album_prefs WHERE user_id = $1", pgUUID(graceID)); got != 0 {
		t.Errorf("the departing account's timeline preferences survived")
	}
	if got := env.queryInt(t, "SELECT count(*) FROM user_face_labels WHERE user_id = $1", pgUUID(graceID)); got != 0 {
		t.Errorf("the departing account's private face names survived")
	}
	if got := env.queryInt(t, "SELECT count(*) FROM face_clusters WHERE id = $1", pgUUID(clusterID)); got != 1 {
		t.Errorf("the shared cluster went with the account, want it kept")
	}
	if got := env.queryInt(t, "SELECT count(*) FROM api_keys WHERE user_id = $1", pgUUID(graceID)); got != 0 {
		t.Errorf("an API key survived the account")
	}
	if got := env.queryInt(t, "SELECT count(*) FROM pairing_tokens WHERE created_by = $1", pgUUID(graceID)); got != 0 {
		t.Errorf("a pairing token survived the account")
	}
	if got := env.queryInt(t, "SELECT count(*) FROM user_favorites WHERE user_id = $1 AND asset_id = $2",
		pgUUID(adminID), pgUUID(liveID)); got != 1 {
		t.Errorf("the receiver's favorite of the transferred photo was lost")
	}

	// THE EVENTS: the moved containers announce themselves.
	for _, id := range []uuid.UUID{memoryID, directoryID} {
		if events := eventsOfAlbum(t, env.bus, "album.updated", id); len(events) != 1 {
			t.Errorf("album.updated for %s = %d events, want 1", id, len(events))
		}
	}
	if events := eventsOfAlbum(t, env.bus, "album.deleted", graceSync); len(events) != 0 {
		t.Errorf("the folded stream announced itself as deleted, want silence")
	}
}

// TestDeleteUserTransferFoldsDuplicate pins the both-live collision
// pair: a photo the receiver already holds live folds into the
// receiver's copy -- memberships, labels, favorites and the album
// cover follow the surviving row, the redundant row goes, and the
// shared file stays because the survivor names it. (The mixed and
// both-trashed pairs are the state-pair test's.)
func TestDeleteUserTransferFoldsDuplicate(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	graceID, _ := env.newMember(t, "grace@example.com")
	adminID := env.sessionUserID(t)

	// The same bytes under both accounts: the content-addressed
	// storage keeps ONE physical file, and the live-copy rule keeps
	// one row per owner.
	sharedPath := filepath.Join(t.TempDir(), "IMG_DUPLICATE.jpg")
	if err := os.WriteFile(sharedPath, []byte("identical bytes"), 0o644); err != nil {
		t.Fatalf("write the shared file: %v", err)
	}
	insertLiveAsset := func(id, owner uuid.UUID, checksum string) {
		t.Helper()
		env.exec(t, `INSERT INTO assets (
				id, owner_id, storage_kind, file_path, checksum, original_filename,
				file_size, mime_type, media_kind, width, height, taken_at,
				taken_at_source, pending_jobs
			) VALUES ($1, $2, 'blob', $3, $4, 'IMG_DUPLICATE.jpg', 15, 'image/jpeg',
				'image', 40, 30, now() - interval '1 hour', 'import', 0)`,
			pgUUID(id), pgUUID(owner), sharedPath, checksum)
	}
	keepID := uuid.MustParse("41000000-0000-0000-0000-000000000001")
	dupID := uuid.MustParse("41000000-0000-0000-0000-000000000002")
	insertLiveAsset(keepID, adminID, "duplicate-checksum")
	insertLiveAsset(dupID, graceID, "duplicate-checksum")

	// The duplicate's own context: a memory album membership, a label,
	// a favorite of the receiver's, and an album it pins as cover.
	dupAlbumID := uuid.MustParse("41000000-0000-0000-0000-000000000003")
	env.insertTimelineAlbum(t, dupAlbumID, graceID, "memory")
	env.addTimelineItem(t, dupAlbumID, dupID, graceID)
	tagID := uuid.MustParse("41000000-0000-0000-0000-000000000004")
	env.exec(t, "INSERT INTO tags (id, owner_id, name) VALUES ($1, $2, 'trips')", pgUUID(tagID), pgUUID(graceID))
	env.exec(t, "INSERT INTO asset_tags (asset_id, tag_id) VALUES ($1, $2)", pgUUID(dupID), pgUUID(tagID))
	env.exec(t, "INSERT INTO user_favorites (user_id, asset_id) VALUES ($1, $2)", pgUUID(adminID), pgUUID(dupID))
	env.exec(t, "UPDATE albums SET cover_asset_id = $1 WHERE id = $2", pgUUID(dupID), pgUUID(dupAlbumID))

	rec := env.deleteUser(t, env.cookie, graceID, transferBody(t, adminID))
	requireStatus(t, rec, http.StatusNoContent, "")

	// The redundant row went; the receiver's copy survives with the
	// duplicate's whole context attached to it.
	if got := env.queryInt(t, "SELECT count(*) FROM assets WHERE id = $1", pgUUID(dupID)); got != 0 {
		t.Errorf("the folded duplicate's row survived")
	}
	if owner := env.queryText(t, "SELECT owner_id::text FROM assets WHERE id = $1", pgUUID(keepID)); owner != adminID.String() {
		t.Errorf("the surviving copy's owner = %s, want the receiver", owner)
	}
	if got := env.queryInt(t, "SELECT count(*) FROM album_items WHERE album_id = $1 AND asset_id = $2",
		pgUUID(dupAlbumID), pgUUID(keepID)); got != 1 {
		t.Errorf("the duplicate's album membership did not follow the surviving row")
	}
	if got := env.queryInt(t, "SELECT count(*) FROM asset_tags WHERE asset_id = $1 AND tag_id = $2",
		pgUUID(keepID), pgUUID(tagID)); got != 1 {
		t.Errorf("the duplicate's label did not follow the surviving row")
	}
	if got := env.queryInt(t, "SELECT count(*) FROM user_favorites WHERE user_id = $1 AND asset_id = $2",
		pgUUID(adminID), pgUUID(keepID)); got != 1 {
		t.Errorf("the duplicate's favorite did not follow the surviving row")
	}
	if cover := env.queryText(t, "SELECT cover_asset_id::text FROM albums WHERE id = $1", pgUUID(dupAlbumID)); cover != keepID.String() {
		t.Errorf("album cover = %s, want the surviving copy %s", cover, keepID)
	}
	// The shared file survives: the survivor names it, which is the
	// shared-blob rule the file half reads.
	if _, err := os.Stat(sharedPath); err != nil {
		t.Errorf("the shared file went with the folded row: %v", err)
	}
}

// TestDeleteUserTransferFoldsEveryStatePair pins the collision rule
// over the four state pairs the total (owner, checksum) index allows
// -- live/live, live/trashed, trashed/live and trashed/trashed. The
// live copy wins a mixed pair (the arriving content stays visible; a
// trashed copy of it does not bind it), a tie keeps the receiver's own
// row (under the later deletion mark when both are trashed), the
// losing row's context moves onto the survivor, a losing Live Photo
// takes its component row along, and the losing row announces
// asset.deleted. The 204 is the regression's own proof: the live-only
// predicate this rule replaced answered 500 on three of the four pairs
// (the ownership write tripped the total index).
func TestDeleteUserTransferFoldsEveryStatePair(t *testing.T) {
	const shared = "state-pair-checksum"
	cases := []struct {
		name             string
		departingDeleted time.Duration
		existingDeleted  time.Duration
		wantSurvivor     string // "departing" | "existing"
		component        bool
	}{
		{name: "BothLive", wantSurvivor: "existing", component: true},
		{name: "DepartingLiveReceiverTrashed", existingDeleted: 2 * time.Hour, wantSurvivor: "departing"},
		{name: "DepartingTrashedReceiverLive", departingDeleted: 2 * time.Hour, wantSurvivor: "existing"},
		{name: "BothTrashed", departingDeleted: time.Hour, existingDeleted: 5 * time.Hour, wantSurvivor: "existing"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			env := newAssetEnv(t, testUploadLimit)
			graceID, _ := env.newMember(t, "grace@example.com")
			adminID := env.sessionUserID(t)

			departingID, existingID := uuid.New(), uuid.New()
			departingOpts := removalAssetOpts{checksum: shared}
			if tc.departingDeleted > 0 {
				departingOpts.deletedAt = time.Now().Add(-tc.departingDeleted)
			}
			existingOpts := removalAssetOpts{checksum: shared}
			if tc.existingDeleted > 0 {
				existingOpts.deletedAt = time.Now().Add(-tc.existingDeleted)
			}
			componentID := uuid.Nil
			if tc.component {
				// The pair: the component row lands first (the photo's
				// foreign key names it), then the photo links to it.
				componentID = uuid.New()
				env.insertRemovalAsset(t, componentID, graceID, removalAssetOpts{
					checksum: "state-pair-component", component: true,
				})
				departingOpts.livePhoto = componentID
			}
			env.insertRemovalAsset(t, departingID, graceID, departingOpts)
			env.insertRemovalAsset(t, existingID, adminID, existingOpts)

			// The losing copy's context: a memory membership, a label, a
			// favorite of the receiver's, and an album it pins as cover.
			albumID, tagID := uuid.New(), uuid.New()
			env.insertTimelineAlbum(t, albumID, graceID, "memory")
			env.addTimelineItem(t, albumID, departingID, graceID)
			env.exec(t, "INSERT INTO tags (id, owner_id, name) VALUES ($1, $2, 'trips')",
				pgUUID(tagID), pgUUID(graceID))
			env.exec(t, "INSERT INTO asset_tags (asset_id, tag_id) VALUES ($1, $2)",
				pgUUID(departingID), pgUUID(tagID))
			env.exec(t, "INSERT INTO user_favorites (user_id, asset_id) VALUES ($1, $2)",
				pgUUID(adminID), pgUUID(departingID))
			env.exec(t, "UPDATE albums SET cover_asset_id = $1 WHERE id = $2",
				pgUUID(departingID), pgUUID(albumID))

			rec := env.deleteUser(t, env.cookie, graceID, transferBody(t, adminID))
			requireStatus(t, rec, http.StatusNoContent, "")

			survivor, loser := existingID, departingID
			if tc.wantSurvivor == "departing" {
				survivor, loser = departingID, existingID
			}

			// ONE row of the content survives, under the receiver.
			if got := env.queryInt(t, "SELECT count(*) FROM assets WHERE checksum = $1", shared); got != 1 {
				t.Fatalf("rows over the shared content = %d, want 1", got)
			}
			if owner := env.queryText(t, "SELECT owner_id::text FROM assets WHERE checksum = $1", shared); owner != adminID.String() {
				t.Errorf("the surviving copy's owner = %s, want the receiver", owner)
			}
			if got := env.queryInt(t, "SELECT count(*) FROM assets WHERE id = $1", pgUUID(loser)); got != 0 {
				t.Errorf("the losing copy's row survived")
			}

			// The survivor's state: live wins a mixed pair, a tie keeps
			// the destination's row under the LATER mark.
			wantLive := tc.departingDeleted == 0 || tc.existingDeleted == 0
			if got := env.queryInt(t, "SELECT count(*) FROM assets WHERE id = $1 AND deleted_at IS NULL", pgUUID(survivor)); (got == 1) != wantLive {
				t.Errorf("survivor live = %v, want %v", got == 1, wantLive)
			}
			if !wantLive {
				if got := env.queryInt(t, "SELECT count(*) FROM assets WHERE id = $1 AND deleted_at > now() - interval '2 hours'", pgUUID(survivor)); got != 1 {
					t.Errorf("the survivor kept the earlier mark, want the later of the two")
				}
			}

			// The context rode the content onto the survivor.
			if got := env.queryInt(t, "SELECT count(*) FROM album_items WHERE album_id = $1 AND asset_id = $2",
				pgUUID(albumID), pgUUID(survivor)); got != 1 {
				t.Errorf("the losing copy's album membership did not follow the survivor")
			}
			if got := env.queryInt(t, "SELECT count(*) FROM asset_tags WHERE asset_id = $1 AND tag_id = $2",
				pgUUID(survivor), pgUUID(tagID)); got != 1 {
				t.Errorf("the losing copy's label did not follow the survivor")
			}
			if got := env.queryInt(t, "SELECT count(*) FROM user_favorites WHERE user_id = $1 AND asset_id = $2",
				pgUUID(adminID), pgUUID(survivor)); got != 1 {
				t.Errorf("the losing copy's favorite did not follow the survivor")
			}
			if cover := env.queryText(t, "SELECT cover_asset_id::text FROM albums WHERE id = $1", pgUUID(albumID)); cover != survivor.String() {
				t.Errorf("album cover = %s, want the survivor %s", cover, survivor)
			}

			// A folded Live Photo takes its component along: the pair is
			// one asset, and an orphan component row would outlive its
			// photo.
			if tc.component {
				if got := env.queryInt(t, "SELECT count(*) FROM assets WHERE id = $1", pgUUID(componentID)); got != 0 {
					t.Errorf("the folded photo's Live Photo component survived as an orphan row")
				}
			}
			requireAssetDeleted(t, env.bus, loser)
		})
	}
}

// TestDeleteUserTrashArmFoldsCallersCopy pins the trash arm's collision
// rule: the acting administrator may hold the same content in any
// state, and the total (owner, checksum) index refuses the re-own that
// ignores it -- the same fold, the caller's row being the destination.
// A live departing row survives a trashed caller's copy (and is trashed
// by this arm), a live caller's copy is never trashed by someone else's
// disposal, and a both-trashed pair keeps the later mark.
func TestDeleteUserTrashArmFoldsCallersCopy(t *testing.T) {
	const shared = "trash-arm-collision"
	cases := []struct {
		name             string
		departingDeleted time.Duration
		existingDeleted  time.Duration
		wantSurvivor     string // "departing" | "existing"
	}{
		{name: "BothLive", wantSurvivor: "existing"},
		{name: "DepartingLiveCallerTrashed", existingDeleted: 3 * time.Hour, wantSurvivor: "departing"},
		{name: "BothTrashed", departingDeleted: time.Hour, existingDeleted: 6 * time.Hour, wantSurvivor: "existing"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			env := newAssetEnv(t, testUploadLimit)
			graceID, _ := env.newMember(t, "grace@example.com")
			adminID := env.sessionUserID(t)

			departingID, existingID := uuid.New(), uuid.New()
			departingOpts := removalAssetOpts{checksum: shared}
			if tc.departingDeleted > 0 {
				departingOpts.deletedAt = time.Now().Add(-tc.departingDeleted)
			}
			existingOpts := removalAssetOpts{checksum: shared}
			if tc.existingDeleted > 0 {
				existingOpts.deletedAt = time.Now().Add(-tc.existingDeleted)
			}
			env.insertRemovalAsset(t, departingID, graceID, departingOpts)
			env.insertRemovalAsset(t, existingID, adminID, existingOpts)

			tagID := uuid.New()
			env.exec(t, "INSERT INTO tags (id, owner_id, name) VALUES ($1, $2, 'trips')",
				pgUUID(tagID), pgUUID(graceID))
			env.exec(t, "INSERT INTO asset_tags (asset_id, tag_id) VALUES ($1, $2)",
				pgUUID(departingID), pgUUID(tagID))

			rec := env.deleteUser(t, env.cookie, graceID, trashBody())
			requireStatus(t, rec, http.StatusNoContent, "")

			survivor, loser := existingID, departingID
			if tc.wantSurvivor == "departing" {
				survivor, loser = departingID, existingID
			}
			if got := env.queryInt(t, "SELECT count(*) FROM assets WHERE checksum = $1", shared); got != 1 {
				t.Fatalf("rows over the shared content = %d, want 1", got)
			}
			if owner := env.queryText(t, "SELECT owner_id::text FROM assets WHERE id = $1", pgUUID(survivor)); owner != adminID.String() {
				t.Errorf("the surviving copy's owner = %s, want the caller", owner)
			}
			if got := env.queryInt(t, "SELECT count(*) FROM assets WHERE id = $1", pgUUID(loser)); got != 0 {
				t.Errorf("the losing copy's row survived")
			}

			// The survivor's state: the arm lands every arriving live
			// row trashed, and a caller's live copy of the content is
			// not the caller's to lose.
			wantLive := tc.name == "BothLive"
			if got := env.queryInt(t, "SELECT count(*) FROM assets WHERE id = $1 AND deleted_at IS NULL", pgUUID(survivor)); (got == 1) != wantLive {
				t.Errorf("survivor live = %v, want %v", got == 1, wantLive)
			}
			if tc.departingDeleted > 0 && tc.existingDeleted > 0 {
				if got := env.queryInt(t, "SELECT count(*) FROM assets WHERE id = $1 AND deleted_at > now() - interval '2 hours'", pgUUID(survivor)); got != 1 {
					t.Errorf("the survivor kept the earlier mark, want the later of the two")
				}
			}

			// The label rode the content onto the survivor (the departing
			// account's tags are the caller's now).
			if got := env.queryInt(t, "SELECT count(*) FROM asset_tags WHERE asset_id = $1 AND tag_id = $2",
				pgUUID(survivor), pgUUID(tagID)); got != 1 {
				t.Errorf("the folded label did not follow the survivor")
			}
			requireAssetDeleted(t, env.bus, loser)
			if tc.name == "BothLive" {
				// Nothing transitioned into the trash: the caller already
				// held the content live.
				if published := busEventsOf(t, env.bus, "asset.trashed"); len(published) != 0 {
					t.Errorf("asset.trashed events = %d, want none", len(published))
				}
			}
		})
	}
}

// TestDeleteUserTrashArmMovesPhotosIntoCallersTrash pins the trash
// arm: the photos re-own to the caller with their deletion marks, the
// account's containers go, an external source file moves into the
// server's trash area, and the caller's trash view holds them under
// the instance's retention window -- restorable until it ends.
func TestDeleteUserTrashArmMovesPhotosIntoCallersTrash(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	graceID, graceCookie := env.newMember(t, "grace@example.com")
	adminID := env.sessionUserID(t)
	graceSync := env.syncAlbumIDOf(t, graceID)

	liveID := uuid.MustParse("42000000-0000-0000-0000-000000000001")
	env.insertTimelineAsset(t, liveID, graceID, time.Now().Add(-time.Hour), timelineAssetOpts{ready: true, width: 40, height: 30})
	env.addTimelineItem(t, graceSync, liveID, graceID)

	// A memory the account curated: its removal is the one the events
	// announce (the stream is no album resource, so it stays silent).
	memoryID := uuid.MustParse("42000000-0000-0000-0000-000000000003")
	env.insertTimelineAlbum(t, memoryID, graceID, "memory")
	env.addTimelineItem(t, memoryID, liveID, graceID)

	sourceDir := t.TempDir()
	directoryID := uuid.MustParse("42000000-0000-0000-0000-000000000002")
	env.insertRemovalDirectory(t, directoryID, graceID, sourceDir)
	externalID := env.insertRemovalMember(t, directoryID, graceID, sourceDir, "grace-0002.jpg",
		[]byte("grace external bytes"), time.Now().Add(-time.Hour))
	sourceFile := filepath.Join(sourceDir, "grace-0002.jpg")

	env.exec(t, "INSERT INTO user_favorites (user_id, asset_id) VALUES ($1, $2)", pgUUID(graceID), pgUUID(liveID))

	rec := env.deleteUser(t, env.cookie, graceID, trashBody())
	requireStatus(t, rec, http.StatusNoContent, "")

	if got := env.queryInt(t, "SELECT count(*) FROM users WHERE id = $1", pgUUID(graceID)); got != 0 {
		t.Errorf("the account row survived")
	}
	if rec := env.getTimelineRaw(t, graceCookie, ""); rec.Code != http.StatusUnauthorized {
		t.Errorf("the deleted account's session answered %d, want 401", rec.Code)
	}

	// The containers went with the account; the photos survive as the
	// caller's trash items.
	if got := env.queryInt(t, "SELECT count(*) FROM albums WHERE owner_id = $1", pgUUID(graceID)); got != 0 {
		t.Errorf("the departing account still owns albums")
	}
	for _, id := range []uuid.UUID{liveID, externalID} {
		if owner := env.queryText(t, "SELECT owner_id::text FROM assets WHERE id = $1", pgUUID(id)); owner != adminID.String() {
			t.Errorf("trashed asset %s owner = %s, want the caller %s", id, owner, adminID)
		}
		if deleted := env.queryInt(t, "SELECT count(*) FROM assets WHERE id = $1 AND deleted_at IS NOT NULL", pgUUID(id)); deleted != 1 {
			t.Errorf("asset %s carries no trash mark", id)
		}
	}

	// The external file followed the row into the trash area; the
	// family's source location is empty.
	if _, err := os.Stat(sourceFile); !os.IsNotExist(err) {
		t.Errorf("the source file survived the trash move (stat err %v)", err)
	}
	trashPath := env.queryText(t, "SELECT trash_path FROM assets WHERE id = $1", pgUUID(externalID))
	if !strings.Contains(trashPath, filepath.Join("trash", externalID.String())) {
		t.Errorf("trash path %q does not sit in the asset's trash home", trashPath)
	}
	if _, err := os.Stat(trashPath); err != nil {
		t.Errorf("the trashed file is missing: %v", err)
	}

	// The caller's trash view holds them under the instance's window,
	// and a restore brings one back -- the retention's own promise.
	trash := env.getTrash(t, env.cookie)
	ids := trashItemIDs(trash.Items)
	if len(ids) != 2 {
		t.Fatalf("trash items = %v, want the two photos", ids)
	}
	for _, item := range trash.Items {
		if item.RemainingDays <= 0 {
			t.Errorf("trash item %s remaining days = %d, want the retention window", item.Id, item.RemainingDays)
		}
	}
	if rec := env.restoreAsset(t, env.cookie, liveID); rec.Code != http.StatusNoContent {
		t.Fatalf("restore status = %d, want 204 (body %q)", rec.Code, rec.Body.String())
	}
	if live := env.queryInt(t, "SELECT count(*) FROM assets WHERE id = $1 AND deleted_at IS NULL", pgUUID(liveID)); live != 1 {
		t.Errorf("the restored photo is still trashed")
	}

	// The per-user rows went with the account.
	if got := env.queryInt(t, "SELECT count(*) FROM user_favorites WHERE user_id = $1", pgUUID(graceID)); got != 0 {
		t.Errorf("the departing account's favorites survived")
	}

	// The events: every photo announced its transition, the containers
	// their removal.
	if events := busEventsOf(t, env.bus, "asset.trashed"); len(events) != 2 {
		t.Errorf("asset.trashed events = %d, want 2", len(events))
	}
	if events := eventsOfAlbum(t, env.bus, "album.deleted", memoryID); len(events) != 1 {
		t.Errorf("album.deleted for the curated memory = %d events, want 1", len(events))
	}
	if events := eventsOfAlbum(t, env.bus, "album.deleted", graceSync); len(events) != 0 {
		t.Errorf("album.deleted for the stream = %d events, want silence", len(events))
	}
}

// TestDeleteUserReleasesCollidingDeviceIdentity pins the second
// uniqueness axis: device_asset_id is an upload-idempotency key, and
// its partial index allows one LIVE row per (owner, device asset) --
// a pair the checksum fold cannot cover, because the two rows carry
// DIFFERENT content (a device may re-render a file under the same
// library identity, so two backups of it hold one id with two
// checksums). The transfer arm releases the arriving row's identity;
// the trash arm needs no such pass, because its write lands every
// arriving row trashed in the same statement and the index covers live
// rows alone.
func TestDeleteUserReleasesCollidingDeviceIdentity(t *testing.T) {
	const device = "device-asset-identity"

	t.Run("TransferArmReleasesTheArrivingIdentity", func(t *testing.T) {
		env := newAssetEnv(t, testUploadLimit)
		graceID, _ := env.newMember(t, "grace@example.com")
		adminID := env.sessionUserID(t)

		departingID, existingID := uuid.New(), uuid.New()
		env.insertTimelineAsset(t, existingID, adminID, time.Now().Add(-2*time.Hour), timelineAssetOpts{ready: true, width: 40, height: 30})
		env.exec(t, "UPDATE assets SET device_asset_id = $1 WHERE id = $2", device, pgUUID(existingID))
		env.insertRemovalAsset(t, departingID, graceID, removalAssetOpts{
			checksum: "device-identity-departing", deviceID: device,
		})

		rec := env.deleteUser(t, env.cookie, graceID, transferBody(t, adminID))
		requireStatus(t, rec, http.StatusNoContent, "")

		if got := env.queryInt(t, "SELECT count(*) FROM assets WHERE id = $1 AND device_asset_id IS NULL", pgUUID(departingID)); got != 1 {
			t.Errorf("the arriving row kept the device identity the receiver already holds live")
		}
		if got := env.queryInt(t, "SELECT count(*) FROM assets WHERE id = $1 AND device_asset_id = $2", pgUUID(existingID), device); got != 1 {
			t.Errorf("the receiver's own row lost its device identity")
		}
		if owner := env.queryText(t, "SELECT owner_id::text FROM assets WHERE id = $1", pgUUID(departingID)); owner != adminID.String() {
			t.Errorf("the arriving row's owner = %s, want the receiver", owner)
		}
	})

	t.Run("TrashArmNeedsNoRelease", func(t *testing.T) {
		// Same pair, the trash arm: the arriving row is trashed by the
		// same statement that re-owns it, so the live-only index never
		// sees it -- and the identity survives, which is the honest
		// state (a restore brings the row back under its own id).
		env := newAssetEnv(t, testUploadLimit)
		graceID, _ := env.newMember(t, "grace@example.com")
		adminID := env.sessionUserID(t)

		departingID, existingID := uuid.New(), uuid.New()
		env.insertTimelineAsset(t, existingID, adminID, time.Now().Add(-2*time.Hour), timelineAssetOpts{ready: true, width: 40, height: 30})
		env.exec(t, "UPDATE assets SET device_asset_id = $1 WHERE id = $2", device, pgUUID(existingID))
		env.insertRemovalAsset(t, departingID, graceID, removalAssetOpts{
			checksum: "device-identity-departing", deviceID: device,
		})

		rec := env.deleteUser(t, env.cookie, graceID, trashBody())
		requireStatus(t, rec, http.StatusNoContent, "")

		if got := env.queryInt(t, "SELECT count(*) FROM assets WHERE id = $1 AND deleted_at IS NOT NULL AND device_asset_id = $2",
			pgUUID(departingID), device); got != 1 {
			t.Errorf("the trashed arriving row lost its own device identity")
		}
		if got := env.queryInt(t, "SELECT count(*) FROM assets WHERE id = $1 AND deleted_at IS NULL", pgUUID(existingID)); got != 1 {
			t.Errorf("the caller's live copy of another content was trashed")
		}
	})
}

// TestDeleteUserIsAtomic pins the transaction's whole-or-nothing
// promise on both failure shapes: a refused external trash move (the
// earlier moves are compensated, the rows untouched) and a database
// failure at the last step (every write before it rolls back).
func TestDeleteUserIsAtomic(t *testing.T) {
	t.Run("TrashMoveFailure", func(t *testing.T) {
		env := newAssetEnv(t, testUploadLimit)
		graceID, _ := env.newMember(t, "grace@example.com")
		graceSync := env.syncAlbumIDOf(t, graceID)

		liveID := uuid.MustParse("43000000-0000-0000-0000-000000000001")
		env.insertTimelineAsset(t, liveID, graceID, time.Now().Add(-time.Hour), timelineAssetOpts{ready: true, width: 40, height: 30})
		env.addTimelineItem(t, graceSync, liveID, graceID)

		sourceDir := t.TempDir()
		directoryID := uuid.MustParse("43000000-0000-0000-0000-000000000002")
		env.insertRemovalDirectory(t, directoryID, graceID, sourceDir)
		movedID := env.insertRemovalMember(t, directoryID, graceID, sourceDir, "grace-0003.jpg",
			[]byte("grace external bytes"), time.Now().Add(-time.Hour))
		// The second external's source is gone from the host, so its
		// move fails after the first one already ran.
		vanishedID := env.insertRemovalMember(t, directoryID, graceID, sourceDir, "grace-0004.jpg",
			[]byte("vanished bytes"), time.Now().Add(-time.Hour))
		if err := os.Remove(filepath.Join(sourceDir, "grace-0004.jpg")); err != nil {
			t.Fatalf("remove the second source file: %v", err)
		}

		rec := env.deleteUser(t, env.cookie, graceID, trashBody())
		requireStatus(t, rec, http.StatusInternalServerError, apigen.ASSETTRASHMOVEFAILED)

		// Nothing changed: the account, its containers and its photos
		// are exactly where they were, and the file the first move
		// took is back at its source.
		if got := env.queryInt(t, "SELECT count(*) FROM users WHERE id = $1", pgUUID(graceID)); got != 1 {
			t.Errorf("the account row went despite the refused move")
		}
		if got := env.queryInt(t, "SELECT count(*) FROM albums WHERE id = $1", pgUUID(graceSync)); got != 1 {
			t.Errorf("the departed stream went despite the refused move")
		}
		for _, id := range []uuid.UUID{liveID, movedID, vanishedID} {
			if owner := env.queryText(t, "SELECT owner_id::text FROM assets WHERE id = $1", pgUUID(id)); owner != graceID.String() {
				t.Errorf("asset %s owner = %s, want the departing account %s", id, owner, graceID)
			}
			if deleted := env.queryInt(t, "SELECT count(*) FROM assets WHERE id = $1 AND deleted_at IS NOT NULL", pgUUID(id)); deleted != 0 {
				t.Errorf("asset %s carries a trash mark despite the refusal", id)
			}
		}
		if _, err := os.Stat(filepath.Join(sourceDir, "grace-0003.jpg")); err != nil {
			t.Errorf("the compensated file is not back at its source: %v", err)
		}
	})

	t.Run("DatabaseFailureRollsEverythingBack", func(t *testing.T) {
		env := newAssetEnv(t, testUploadLimit)
		graceID, _ := env.newMember(t, "grace@example.com")
		adminID := env.sessionUserID(t)
		graceSync := env.syncAlbumIDOf(t, graceID)

		liveID := uuid.MustParse("43000000-0000-0000-0000-000000000011")
		env.insertTimelineAsset(t, liveID, graceID, time.Now().Add(-time.Hour), timelineAssetOpts{ready: true, width: 40, height: 30})
		env.addTimelineItem(t, graceSync, liveID, graceID)
		memoryID := uuid.MustParse("43000000-0000-0000-0000-000000000012")
		env.insertTimelineAlbum(t, memoryID, graceID, "memory")
		env.addTimelineItem(t, memoryID, liveID, graceID)
		tagID := uuid.MustParse("43000000-0000-0000-0000-000000000013")
		env.exec(t, "INSERT INTO tags (id, owner_id, name) VALUES ($1, $2, 'trips')", pgUUID(tagID), pgUUID(graceID))
		env.exec(t, "INSERT INTO asset_tags (asset_id, tag_id) VALUES ($1, $2)", pgUUID(liveID), pgUUID(tagID))
		env.exec(t, "INSERT INTO user_favorites (user_id, asset_id) VALUES ($1, $2)", pgUUID(graceID), pgUUID(liveID))

		// The last step -- the account row's deletion -- is blocked, so
		// every settlement the transaction performed before it must
		// roll back.
		env.exec(t, `
			CREATE FUNCTION test_block_user_delete() RETURNS trigger AS $$
			BEGIN
				RAISE EXCEPTION 'the test blocks this deletion';
			END
			$$ LANGUAGE plpgsql`)
		env.exec(t, `CREATE TRIGGER test_block_user_delete BEFORE DELETE ON users
			FOR EACH ROW EXECUTE FUNCTION test_block_user_delete()`)

		rec := env.deleteUser(t, env.cookie, graceID, transferBody(t, adminID))
		requireStatus(t, rec, http.StatusInternalServerError, apigen.INTERNAL)

		if owner := env.queryText(t, "SELECT owner_id::text FROM assets WHERE id = $1", pgUUID(liveID)); owner != graceID.String() {
			t.Errorf("the photo was handed over despite the rollback (owner %s)", owner)
		}
		if owner := env.queryText(t, "SELECT owner_id::text FROM albums WHERE id = $1", pgUUID(memoryID)); owner != graceID.String() {
			t.Errorf("the album was handed over despite the rollback (owner %s)", owner)
		}
		if owner := env.queryText(t, "SELECT owner_id::text FROM tags WHERE id = $1", pgUUID(tagID)); owner != graceID.String() {
			t.Errorf("the label was handed over despite the rollback (owner %s)", owner)
		}
		if got := env.queryInt(t, "SELECT count(*) FROM album_items WHERE album_id = $1 AND added_by = $2",
			pgUUID(env.syncAlbumIDOf(t, graceID)), pgUUID(graceID)); got != 1 {
			t.Errorf("the stream membership changed despite the rollback")
		}
		if got := env.queryInt(t, "SELECT count(*) FROM user_favorites WHERE user_id = $1", pgUUID(graceID)); got != 1 {
			t.Errorf("the account's favorites died despite the rollback")
		}
		if got := env.queryInt(t, "SELECT count(*) FROM users WHERE id = $1", pgUUID(graceID)); got != 1 {
			t.Errorf("the account row went despite the blocked deletion")
		}
	})
}
