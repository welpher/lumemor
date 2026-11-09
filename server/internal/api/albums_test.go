// Album surface tests (M3c Task 1): the wall (the own group then the
// shared group, covers and counts honest over live members) and the
// detail face (metadata plus the keyset-paged items, the single-shape
// ALBUM_NOT_FOUND). Fixtures insert albums, shares and assets
// directly in the database -- the album write and share endpoints
// arrive in later tasks, and the reads must be honest about rows no
// API can produce yet, exactly like the timeline fixtures before
// them.
package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"

	apigen "github.com/welpher/lumemor/server/internal/api/gen"
)

// fixtureWallAlbumID builds a deterministic fixture album id from a
// two-character hex suffix, in a namespace disjoint from the timeline
// (10..) and people (20../30../40..) fixture ids.
func fixtureWallAlbumID(suffix string) uuid.UUID {
	return uuid.MustParse("50000000-0000-0000-0000-0000000000" + suffix)
}

// fixtureWallAssetID builds a deterministic fixture asset id, in a
// namespace disjoint from every other fixture family.
func fixtureWallAssetID(suffix string) uuid.UUID {
	return uuid.MustParse("60000000-0000-0000-0000-0000000000" + suffix)
}

// bulkAlbumAssetID builds the i-th id of the bulk pagination
// fixtures, in a namespace disjoint from every hand-picked fixture
// id.
func bulkAlbumAssetID(i int) uuid.UUID {
	return uuid.MustParse(fmt.Sprintf("60000000-0000-0000-0001-%012x", i))
}

// wallAlbumOpts carries the optional column values of a fixture
// album; the zero value is a plain unlocked album with no pinned
// cover and the insert time as updated_at.
type wallAlbumOpts struct {
	locked       bool
	coverAssetID *uuid.UUID
	updatedAt    time.Time
}

// insertWallAlbum inserts one fixture album row directly, with
// explicit metadata (the wall orders the own group by updated_at, so
// the fixtures pin it instead of relying on now()).
func (e *assetEnv) insertWallAlbum(t *testing.T, id, owner uuid.UUID, albumType, title string, opts wallAlbumOpts) {
	t.Helper()
	var coverAssetID any
	if opts.coverAssetID != nil {
		coverAssetID = pgUUID(*opts.coverAssetID)
	}
	updatedAt := opts.updatedAt
	if updatedAt.IsZero() {
		updatedAt = time.Now()
	}
	if _, err := e.pool.Exec(context.Background(), `
		INSERT INTO albums (id, owner_id, title, type, locked, cover_asset_id, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		pgUUID(id), pgUUID(owner), title, albumType,
		opts.locked, coverAssetID, updatedAt); err != nil {
		t.Fatalf("insert fixture album %s: %v", id, err)
	}
}

// shareWallAlbum grants one user the view ACL of an album at an
// explicit time (the wall orders the shared group by the caller's
// grant time).
func (e *assetEnv) shareWallAlbum(t *testing.T, album, grantee, grantedBy uuid.UUID, grantedAt time.Time) {
	t.Helper()
	if _, err := e.pool.Exec(context.Background(),
		"INSERT INTO album_shares (album_id, user_id, granted_by, granted_at) VALUES ($1, $2, $3, $4)",
		pgUUID(album), pgUUID(grantee), pgUUID(grantedBy), grantedAt); err != nil {
		t.Fatalf("insert fixture album share %s -> %s: %v", album, grantee, err)
	}
}

// getAlbumsRaw fires a GET against the album surface with the given
// cookie (nil sends it unauthenticated) and returns the recorder.
func (e *assetEnv) getAlbumsRaw(t *testing.T, cookie *http.Cookie, target string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, target, nil)
	if cookie != nil {
		req.AddCookie(cookie)
	}
	rec := httptest.NewRecorder()
	e.router.ServeHTTP(rec, req)
	return rec
}

// getAlbumWall calls GET /albums and decodes the 200 body.
func (e *assetEnv) getAlbumWall(t *testing.T, cookie *http.Cookie) apigen.AlbumWallResponse {
	t.Helper()
	rec := e.getAlbumsRaw(t, cookie, "/api/v1/albums")
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /albums status = %d, want %d (body %q)",
			rec.Code, http.StatusOK, rec.Body.String())
	}
	var wall apigen.AlbumWallResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &wall); err != nil {
		t.Fatalf("decode album wall body %q: %v", rec.Body.String(), err)
	}
	return wall
}

// getAlbumDetail calls GET /albums/{albumId} with the raw query string
// and decodes the 200 body.
func (e *assetEnv) getAlbumDetail(t *testing.T, cookie *http.Cookie, albumID uuid.UUID, rawQuery string) apigen.AlbumDetail {
	t.Helper()
	target := "/api/v1/albums/" + albumID.String()
	if rawQuery != "" {
		target += "?" + rawQuery
	}
	rec := e.getAlbumsRaw(t, cookie, target)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /albums/%s?%s status = %d, want %d (body %q)",
			albumID, rawQuery, rec.Code, http.StatusOK, rec.Body.String())
	}
	var detail apigen.AlbumDetail
	if err := json.Unmarshal(rec.Body.Bytes(), &detail); err != nil {
		t.Fatalf("decode album detail body %q: %v", rec.Body.String(), err)
	}
	return detail
}

// requireAlbumNotFound asserts the response is the single-shape 404 of
// the album surface: the ALBUM_NOT_FOUND code, nothing else.
func requireAlbumNotFound(t *testing.T, rec *httptest.ResponseRecorder) {
	t.Helper()
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d (body %q)",
			rec.Code, http.StatusNotFound, rec.Body.String())
	}
	apiErr, _ := decodeError(t, rec)
	if apiErr.Code != apigen.ALBUMNOTFOUND {
		t.Errorf("code = %q, want %q", apiErr.Code, apigen.ALBUMNOTFOUND)
	}
}

// TestAlbumWallEmptyDB walks the album reads against an empty
// instance: the wall holds exactly the caller's pinned sync card
// (the default album every account carries), the detail 404s for an
// unknown id, and nothing answers without a session. The sync card
// IS the wall's floor -- a fresh account's wall is never empty.
func TestAlbumWallEmptyDB(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	ownSync := env.syncAlbumIDOf(t, env.sessionUserID(t))

	t.Run("WallIsTheSyncCardAlone", func(t *testing.T) {
		wall := env.getAlbumWall(t, env.cookie)
		if wall.Albums == nil {
			t.Fatalf("GET /albums body = null, want an array")
		}
		if len(wall.Albums) != 1 {
			t.Fatalf("GET /albums holds %d items, want 1 (the pinned sync card; got %v)",
				len(wall.Albums), wall.Albums)
		}
		card := wall.Albums[0]
		if uuid.UUID(card.Id) != ownSync {
			t.Errorf("wall[0].id = %s, want the caller's sync album %s", card.Id, ownSync)
		}
		if card.Type != apigen.AlbumSummaryTypeSync {
			t.Errorf("wall[0].type = %q, want sync", card.Type)
		}
		if card.Title != "Sync" {
			// The stored title is the system's internal label; the
			// CLIENT renders the localized display name off the type
			// (the contract's convention, pinned here).
			t.Errorf("wall[0].title = %q, want the internal label %q", card.Title, "Sync")
		}
		if card.CallerRole != apigen.AlbumSummaryCallerRoleOwner {
			t.Errorf("wall[0].callerRole = %q, want owner", card.CallerRole)
		}
		if card.IsShared {
			t.Error("wall[0].isShared = true, want false (no share can name a sync album)")
		}
		if card.ItemCount != 0 {
			t.Errorf("wall[0].itemCount = %d, want 0 (no members yet)", card.ItemCount)
		}
		if card.CoverAssetIds == nil || len(card.CoverAssetIds) != 0 {
			t.Errorf("wall[0].coverAssetIds = %v, want an empty array", card.CoverAssetIds)
		}
		if card.InTimeline != nil {
			t.Errorf("wall[0].inTimeline = %v, want absent (the core stream is never excludable)", card.InTimeline)
		}
		if card.Owner != nil {
			t.Errorf("wall[0].owner = %+v, want absent on an own card", card.Owner)
		}
	})

	t.Run("UnknownAlbumNotFound", func(t *testing.T) {
		rec := env.getAlbumsRaw(t, env.cookie,
			"/api/v1/albums/"+fixtureWallAlbumID("ff").String())
		requireAlbumNotFound(t, rec)
	})

	t.Run("RequiresSession", func(t *testing.T) {
		for _, target := range []string{
			"/api/v1/albums",
			"/api/v1/albums/" + fixtureWallAlbumID("01").String(),
		} {
			rec := env.getAlbumsRaw(t, nil, target)
			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("GET %s unauthenticated status = %d, want %d",
					target, rec.Code, http.StatusUnauthorized)
			}
		}
	})
}

// TestAlbumWall pins the brief's seeded wall: the caller's sync
// album pinned first, then the caller's own virtual and directory
// albums (updated_at descending), then another account's virtual
// album shared to the caller (the caller's grant time descending)
// with owner attribution and the viewer role. A foreign sync album
// (even carrying a share row) never appears; an unshared foreign
// album never appears.
func TestAlbumWall(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	adminID := env.sessionUserID(t)
	graceID, _ := env.createSecondUser(t, "wall-grace@example.com")
	ownSync := env.syncAlbumIDOf(t, adminID)

	// The caller's own two albums. VA carries a share to Grace (the
	// owner's isShared view) and two live members; DA is empty.
	ownVirtual := fixtureWallAlbumID("01")
	ownDirectory := fixtureWallAlbumID("02")
	env.insertWallAlbum(t, ownVirtual, adminID, "virtual", "Trip to the coast", wallAlbumOpts{
		updatedAt: time.Date(2026, 11, 4, 18, 0, 0, 0, time.UTC),
	})
	env.insertWallAlbum(t, ownDirectory, adminID, "directory", "Archive", wallAlbumOpts{
		updatedAt: time.Date(2026, 11, 3, 9, 0, 0, 0, time.UTC),
	})
	env.shareWallAlbum(t, ownVirtual, graceID, adminID, time.Date(2026, 11, 1, 10, 0, 0, 0, time.UTC))

	// Grace's shared virtual album with one live member of her own.
	shared := fixtureWallAlbumID("03")
	env.insertWallAlbum(t, shared, graceID, "virtual", "Grace picks", wallAlbumOpts{
		updatedAt: time.Date(2026, 11, 2, 14, 0, 0, 0, time.UTC),
	})
	sharedAsset := fixtureWallAssetID("03")
	env.insertTimelineAsset(t, sharedAsset, graceID, time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC), timelineAssetOpts{})
	env.addTimelineItem(t, shared, sharedAsset, graceID)
	env.shareWallAlbum(t, shared, adminID, graceID, time.Date(2026, 11, 2, 14, 0, 0, 0, time.UTC))

	// Grace's unshared album: never on the caller's wall.
	unshared := fixtureWallAlbumID("04")
	env.insertWallAlbum(t, unshared, graceID, "virtual", "Private", wallAlbumOpts{})

	// The caller's own members: two live assets (the newer first in
	// the derived cover).
	ownNewer := fixtureWallAssetID("01")
	ownOlder := fixtureWallAssetID("02")
	env.insertTimelineAsset(t, ownNewer, adminID, time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC), timelineAssetOpts{})
	env.insertTimelineAsset(t, ownOlder, adminID, time.Date(2026, 5, 9, 12, 0, 0, 0, time.UTC), timelineAssetOpts{})
	env.addTimelineItem(t, ownVirtual, ownNewer, adminID)
	env.addTimelineItem(t, ownVirtual, ownOlder, adminID)

	// A sync album carrying a share row to the caller: still never a
	// wall card, in either group.
	graceSync := fixtureWallAlbumID("05")
	env.insertWallAlbum(t, graceSync, graceID, "sync", "Sync", wallAlbumOpts{})
	env.shareWallAlbum(t, graceSync, adminID, graceID, time.Date(2026, 11, 5, 8, 0, 0, 0, time.UTC))

	wall := env.getAlbumWall(t, env.cookie)
	if len(wall.Albums) != 4 {
		t.Fatalf("GET /albums holds %d items, want 4 (the pinned sync + own virtual + own directory + one shared; got %v)",
			len(wall.Albums), wall.Albums)
	}
	syncCard, virtual, directory, sharedItem := wall.Albums[0], wall.Albums[1], wall.Albums[2], wall.Albums[3]

	// The pinned default card: the caller's own sync album, first,
	// with the internal title and the excludable field absent.
	if uuid.UUID(syncCard.Id) != ownSync {
		t.Errorf("wall[0].id = %s, want the caller's sync album %s", syncCard.Id, ownSync)
	}
	if syncCard.Type != apigen.AlbumSummaryTypeSync {
		t.Errorf("wall[0].type = %q, want sync (the default album)", syncCard.Type)
	}
	if syncCard.Title != "Sync" {
		t.Errorf("wall[0].title = %q, want the internal system label %q (the client renders the display name)", syncCard.Title, "Sync")
	}
	if syncCard.CallerRole != apigen.AlbumSummaryCallerRoleOwner {
		t.Errorf("wall[0].callerRole = %q, want owner", syncCard.CallerRole)
	}
	if syncCard.IsShared {
		t.Error("wall[0].isShared = true, want false (no share can name a sync album)")
	}
	if syncCard.ItemCount != 0 {
		t.Errorf("wall[0].itemCount = %d, want 0 (this fixture holds no sync members)", syncCard.ItemCount)
	}
	if syncCard.CoverAssetIds == nil || len(syncCard.CoverAssetIds) != 0 {
		t.Errorf("wall[0].coverAssetIds = %v, want an empty array", syncCard.CoverAssetIds)
	}
	if syncCard.InTimeline != nil {
		t.Errorf("wall[0].inTimeline = %v, want absent (the core stream is never excludable)", syncCard.InTimeline)
	}
	if syncCard.Owner != nil {
		t.Errorf("wall[0].owner = %+v, want absent on an own card", syncCard.Owner)
	}

	if uuid.UUID(virtual.Id) != ownVirtual {
		t.Errorf("wall[1].id = %s, want the own virtual %s", virtual.Id, ownVirtual)
	}
	if virtual.CallerRole != apigen.AlbumSummaryCallerRoleOwner {
		t.Errorf("wall[1].callerRole = %q, want owner", virtual.CallerRole)
	}
	if virtual.Title != "Trip to the coast" {
		t.Errorf("wall[1].title = %q, want the stored value", virtual.Title)
	}
	if virtual.Type != apigen.AlbumSummaryTypeVirtual {
		t.Errorf("wall[1].type = %q, want virtual", virtual.Type)
	}
	if virtual.Locked {
		t.Error("wall[1].locked = true, want false")
	}
	// The share to Grace: the owner's isShared view.
	if !virtual.IsShared {
		t.Error("wall[1].isShared = false, want true (the album carries a share row)")
	}
	if virtual.ItemCount != 2 {
		t.Errorf("wall[1].itemCount = %d, want 2", virtual.ItemCount)
	}
	// No pinned cover: the two most recent live members, newest first.
	if len(virtual.CoverAssetIds) != 2 ||
		uuid.UUID(virtual.CoverAssetIds[0]) != ownNewer ||
		uuid.UUID(virtual.CoverAssetIds[1]) != ownOlder {
		t.Errorf("wall[1].coverAssetIds = %v, want [%s %s] (the derived cover, newest first)",
			virtual.CoverAssetIds, ownNewer, ownOlder)
	}
	if want := time.Date(2026, 11, 4, 18, 0, 0, 0, time.UTC); !virtual.UpdatedAt.Truncate(time.Microsecond).Equal(want) {
		t.Errorf("wall[1].updatedAt = %v, want %v", virtual.UpdatedAt, want)
	}
	if virtual.Owner != nil {
		t.Errorf("wall[1].owner = %+v, want absent on an own item", virtual.Owner)
	}

	if uuid.UUID(directory.Id) != ownDirectory {
		t.Errorf("wall[2].id = %s, want the own directory %s", directory.Id, ownDirectory)
	}
	if directory.Type != apigen.AlbumSummaryTypeDirectory {
		t.Errorf("wall[2].type = %q, want directory", directory.Type)
	}
	if directory.CallerRole != apigen.AlbumSummaryCallerRoleOwner {
		t.Errorf("wall[2].callerRole = %q, want owner", directory.CallerRole)
	}
	if directory.IsShared {
		t.Error("wall[2].isShared = true, want false (no share row)")
	}
	if directory.ItemCount != 0 {
		t.Errorf("wall[2].itemCount = %d, want 0", directory.ItemCount)
	}
	if directory.CoverAssetIds == nil || len(directory.CoverAssetIds) != 0 {
		t.Errorf("wall[2].coverAssetIds = %v, want an empty array (no members)", directory.CoverAssetIds)
	}

	if uuid.UUID(sharedItem.Id) != shared {
		t.Errorf("wall[3].id = %s, want the shared virtual %s", sharedItem.Id, shared)
	}
	if sharedItem.CallerRole != apigen.AlbumSummaryCallerRoleViewer {
		t.Errorf("wall[3].callerRole = %q, want viewer", sharedItem.CallerRole)
	}
	if !sharedItem.IsShared {
		t.Error("wall[3].isShared = false, want true by construction (the caller's share row)")
	}
	if sharedItem.ItemCount != 1 {
		t.Errorf("wall[3].itemCount = %d, want 1", sharedItem.ItemCount)
	}
	if len(sharedItem.CoverAssetIds) != 1 || uuid.UUID(sharedItem.CoverAssetIds[0]) != sharedAsset {
		t.Errorf("wall[3].coverAssetIds = %v, want [%s]", sharedItem.CoverAssetIds, sharedAsset)
	}
	// The honest attribution of another account's album.
	if sharedItem.Owner == nil {
		t.Fatal("wall[3].owner = null, want the owner attribution of the shared item")
	}
	if uuid.UUID(sharedItem.Owner.Id) != graceID {
		t.Errorf("wall[3].owner.id = %s, want %s", sharedItem.Owner.Id, graceID)
	}
	if string(sharedItem.Owner.Email) != "wall-grace@example.com" {
		t.Errorf("wall[3].owner.email = %q, want the owner's address", sharedItem.Owner.Email)
	}
}

// TestAlbumWallOrdering pins the wall's ordering keys: the pinned
// sync card first, then the own group by updated_at descending with
// the album id as the tiebreak (descending), then the shared group
// by the caller's grant time descending.
func TestAlbumWallOrdering(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	adminID := env.sessionUserID(t)
	ownSync := env.syncAlbumIDOf(t, adminID)
	graceID, _ := env.createSecondUser(t, "wall-order-grace@example.com")

	// Two own albums sharing one updated_at far in the future of the
	// sync row: the pin must lead them anyway (the pin is structural,
	// not recency), and the id tiebreak walks the higher id first.
	tie := time.Date(2030, 1, 1, 12, 0, 0, 0, time.UTC)
	env.insertWallAlbum(t, fixtureWallAlbumID("61"), adminID, "virtual", "A", wallAlbumOpts{updatedAt: tie})
	env.insertWallAlbum(t, fixtureWallAlbumID("62"), adminID, "virtual", "B", wallAlbumOpts{updatedAt: tie})

	// Two shared albums: the later grant first, whatever their
	// updated_at values say (the grant time owns the shared group).
	env.insertWallAlbum(t, fixtureWallAlbumID("63"), graceID, "virtual", "C", wallAlbumOpts{
		updatedAt: time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC),
	})
	env.shareWallAlbum(t, fixtureWallAlbumID("63"), adminID, graceID, time.Date(2026, 11, 5, 9, 0, 0, 0, time.UTC))
	env.insertWallAlbum(t, fixtureWallAlbumID("64"), graceID, "virtual", "D", wallAlbumOpts{
		updatedAt: time.Date(2026, 11, 4, 0, 0, 0, 0, time.UTC),
	})
	env.shareWallAlbum(t, fixtureWallAlbumID("64"), adminID, graceID, time.Date(2026, 11, 5, 7, 0, 0, 0, time.UTC))

	wall := env.getAlbumWall(t, env.cookie)
	want := []uuid.UUID{
		ownSync,                  // the pinned default card, first
		fixtureWallAlbumID("62"), // own tie, higher id
		fixtureWallAlbumID("61"), // own tie, lower id
		fixtureWallAlbumID("63"), // shared, later grant
		fixtureWallAlbumID("64"), // shared, earlier grant
	}
	if len(wall.Albums) != len(want) {
		t.Fatalf("GET /albums holds %d items, want %d (%v)", len(wall.Albums), len(want), wall.Albums)
	}
	for i, id := range want {
		if uuid.UUID(wall.Albums[i].Id) != id {
			t.Errorf("wall[%d].id = %s, want %s (wall %v)", i, wall.Albums[i].Id, id, wall.Albums)
		}
	}
}

// TestAlbumWallCoversAndCounts pins the wall's honesty: covers and
// counts see live members only (hidden and trashed members excluded
// everywhere, A5; Live Photo component rows excluded like every live
// view), and a pinned cover wins over the derived one. The pinned
// sync card obeys the same honesty over its own members.
func TestAlbumWallCoversAndCounts(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	adminID := env.sessionUserID(t)
	ownSync := env.syncAlbumIDOf(t, adminID)

	// The sync album's own members: three live plus the three
	// non-live newest of all -- the pinned card's count and derived
	// cover must be exactly as honest as any album's.
	syncBase := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	syncLive := []uuid.UUID{
		fixtureWallAssetID("51"),
		fixtureWallAssetID("52"),
		fixtureWallAssetID("53"),
	}
	for i, id := range syncLive {
		env.insertTimelineAsset(t, id, adminID, syncBase.Add(time.Duration(3-i)*time.Hour), timelineAssetOpts{})
		env.addTimelineItem(t, ownSync, id, adminID)
	}
	for _, row := range []struct {
		id   uuid.UUID
		opts timelineAssetOpts
	}{
		{fixtureWallAssetID("61"), timelineAssetOpts{isHidden: true}},
		{fixtureWallAssetID("62"), timelineAssetOpts{deleted: true}},
		{fixtureWallAssetID("63"), timelineAssetOpts{isComponent: true}},
	} {
		env.insertTimelineAsset(t, row.id, adminID,
			syncBase.Add(time.Duration(10)*time.Hour), row.opts)
		env.addTimelineItem(t, ownSync, row.id, adminID)
	}

	// The derived-cover album: the hidden, trashed and component
	// members are the NEWEST -- an implementation that let any of them
	// through would put them into the cover's head.
	derived := fixtureWallAlbumID("11")
	env.insertWallAlbum(t, derived, adminID, "virtual", "Derived cover", wallAlbumOpts{})
	base := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	live := make([]uuid.UUID, 6)
	for i := range live {
		live[i] = fixtureWallAssetID(fmt.Sprintf("%02d", 11+i))
		env.insertTimelineAsset(t, live[i], adminID, base.Add(time.Duration(6-i)*time.Hour), timelineAssetOpts{})
		env.addTimelineItem(t, derived, live[i], adminID)
	}
	excluded := []struct {
		id   uuid.UUID
		opts timelineAssetOpts
	}{
		{fixtureWallAssetID("21"), timelineAssetOpts{isHidden: true}},    // newest of all
		{fixtureWallAssetID("22"), timelineAssetOpts{deleted: true}},     // second newest
		{fixtureWallAssetID("23"), timelineAssetOpts{isComponent: true}}, // third newest
	}
	for _, row := range excluded {
		env.insertTimelineAsset(t, row.id, adminID,
			base.Add(time.Duration(10+len(excluded))*time.Hour), row.opts)
		env.addTimelineItem(t, derived, row.id, adminID)
	}

	// The pinned-cover album: the cover is a mid-pack live member, not
	// the most recent -- the pin wins over the derived candidates.
	pinned := fixtureWallAlbumID("12")
	pinnedCover := fixtureWallAssetID("31")
	env.insertWallAlbum(t, pinned, adminID, "virtual", "Pinned cover", wallAlbumOpts{
		coverAssetID: &pinnedCover,
	})
	for i, days := range []int{5, 10, 1, 0} {
		id := fixtureWallAssetID(fmt.Sprintf("%02d", 31+i))
		taken := time.Date(2026, 6, days+1, 12, 0, 0, 0, time.UTC)
		if i == 0 {
			id = pinnedCover // taken June 6, mid-pack
			taken = time.Date(2026, 6, 6, 12, 0, 0, 0, time.UTC)
		}
		env.insertTimelineAsset(t, id, adminID, taken, timelineAssetOpts{})
		env.addTimelineItem(t, pinned, id, adminID)
	}

	wall := env.getAlbumWall(t, env.cookie)
	byID := make(map[uuid.UUID]apigen.AlbumSummary, len(wall.Albums))
	for _, album := range wall.Albums {
		byID[uuid.UUID(album.Id)] = album
	}

	derivedAlbum, ok := byID[derived]
	if !ok {
		t.Fatalf("the derived-cover album is missing from the wall (got %v)", wall.Albums)
	}
	if derivedAlbum.ItemCount != 6 {
		t.Errorf("derived-cover itemCount = %d, want 6 (the live members only)", derivedAlbum.ItemCount)
	}
	wantCover := []uuid.UUID{live[0], live[1], live[2], live[3]}
	if len(derivedAlbum.CoverAssetIds) != len(wantCover) {
		t.Fatalf("derived-cover coverAssetIds = %v, want the 4 newest live %v",
			derivedAlbum.CoverAssetIds, wantCover)
	}
	for i, id := range wantCover {
		if uuid.UUID(derivedAlbum.CoverAssetIds[i]) != id {
			t.Errorf("derived-cover coverAssetIds[%d] = %s, want %s (all %v)",
				i, derivedAlbum.CoverAssetIds[i], id, derivedAlbum.CoverAssetIds)
		}
	}

	pinnedAlbum, ok := byID[pinned]
	if !ok {
		t.Fatalf("the pinned-cover album is missing from the wall (got %v)", wall.Albums)
	}
	if pinnedAlbum.ItemCount != 4 {
		t.Errorf("pinned-cover itemCount = %d, want 4", pinnedAlbum.ItemCount)
	}
	if len(pinnedAlbum.CoverAssetIds) != 1 || uuid.UUID(pinnedAlbum.CoverAssetIds[0]) != pinnedCover {
		t.Errorf("pinned-cover coverAssetIds = %v, want exactly [%s] (the pin, not the recency)",
			pinnedAlbum.CoverAssetIds, pinnedCover)
	}

	// The pinned sync card: the same honesty over its own members --
	// three live count, the three newest live as the derived cover
	// (the non-live members are the newest of all and must lead
	// nowhere).
	syncAlbum, ok := byID[ownSync]
	if !ok {
		t.Fatalf("the caller's sync album is missing from the wall (got %v)", wall.Albums)
	}
	if syncAlbum.Type != apigen.AlbumSummaryTypeSync {
		t.Errorf("sync card type = %q, want sync", syncAlbum.Type)
	}
	if syncAlbum.ItemCount != 3 {
		t.Errorf("sync card itemCount = %d, want 3 (the live members only)", syncAlbum.ItemCount)
	}
	if len(syncAlbum.CoverAssetIds) != len(syncLive) {
		t.Fatalf("sync card coverAssetIds = %v, want the 3 newest live %v",
			syncAlbum.CoverAssetIds, syncLive)
	}
	for i, id := range syncLive {
		if uuid.UUID(syncAlbum.CoverAssetIds[i]) != id {
			t.Errorf("sync card coverAssetIds[%d] = %s, want %s (all %v)",
				i, syncAlbum.CoverAssetIds[i], id, syncAlbum.CoverAssetIds)
		}
	}

	// The detail carries the raw pin (the wall derives, the detail
	// reports the column).
	detail := env.getAlbumDetail(t, env.cookie, pinned, "")
	if detail.CoverAssetId == nil || uuid.UUID(*detail.CoverAssetId) != pinnedCover {
		t.Errorf("detail coverAssetId = %v, want the pinned %s", detail.CoverAssetId, pinnedCover)
	}
}

// TestAlbumDetail walks the detail's keyset pagination: the default
// page of 60 over 61 items with the follow-up page of one, the full
// disjoint walk with a taken_at tie pair spanning a page boundary,
// and the item shape -- live members only.
func TestAlbumDetail(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	adminID := env.sessionUserID(t)

	album := fixtureWallAlbumID("21")
	env.insertWallAlbum(t, album, adminID, "virtual", "The long walk", wallAlbumOpts{})

	// 61 live members. Asset i was taken (60 - i) minutes after the
	// base, except items 3 and 4 share one taken_at: the tie pair
	// occupies the order positions 3 and 4, and the id tiebreak walks
	// the higher id (4) first.
	base := time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC)
	takenOf := func(i int) time.Time {
		if i == 3 {
			return base.Add(56 * time.Minute) // shared with i=4
		}
		return base.Add(time.Duration(60-i) * time.Minute)
	}
	wantOrder := make([]uuid.UUID, 0, 61)
	for i := 0; i <= 60; i++ {
		env.insertTimelineAsset(t, bulkAlbumAssetID(i), adminID, takenOf(i), timelineAssetOpts{})
		env.addTimelineItem(t, album, bulkAlbumAssetID(i), adminID)
		wantOrder = append(wantOrder, bulkAlbumAssetID(i))
	}
	wantOrder[3], wantOrder[4] = wantOrder[4], wantOrder[3] // id tiebreak: 4 before 3

	// The non-live members are the newest of all: they must never
	// surface in the items, the count or the first page's head.
	for _, row := range []struct {
		id   uuid.UUID
		opts timelineAssetOpts
	}{
		{fixtureWallAssetID("41"), timelineAssetOpts{isHidden: true}},
		{fixtureWallAssetID("42"), timelineAssetOpts{deleted: true}},
		{fixtureWallAssetID("43"), timelineAssetOpts{isComponent: true}},
	} {
		env.insertTimelineAsset(t, row.id, adminID,
			base.Add(time.Duration(10+len(wantOrder))*time.Minute), row.opts)
		env.addTimelineItem(t, album, row.id, adminID)
	}

	t.Run("DefaultPageAndFollowUp", func(t *testing.T) {
		page := env.getAlbumDetail(t, env.cookie, album, "")
		if len(page.Items) != 60 {
			t.Fatalf("default page holds %d items, want the contract default of 60", len(page.Items))
		}
		if page.NextCursor == nil {
			t.Fatal("nextCursor absent on a full default page with one more row after it, want present")
		}
		rest := env.getAlbumDetail(t, env.cookie, album, "cursor="+*page.NextCursor)
		if len(rest.Items) != 1 {
			t.Fatalf("follow-up page holds %d items, want the remaining 1", len(rest.Items))
		}
		if rest.NextCursor != nil {
			t.Error("nextCursor set on the final page, want absent")
		}
		if uuid.UUID(rest.Items[0].AssetId) != wantOrder[60] {
			t.Errorf("follow-up item = %s, want %s", rest.Items[0].AssetId, wantOrder[60])
		}
	})

	t.Run("WalkWithoutDupesOrSkips", func(t *testing.T) {
		seen := make(map[uuid.UUID]bool)
		var walked []uuid.UUID
		cursor := ""
		pages := 0
		for {
			rawQuery := "limit=2"
			if cursor != "" {
				rawQuery += "&cursor=" + cursor
			}
			page := env.getAlbumDetail(t, env.cookie, album, rawQuery)
			for _, item := range page.Items {
				id := uuid.UUID(item.AssetId)
				if seen[id] {
					t.Fatalf("asset %s appeared on two pages (cursor %q)", id, cursor)
				}
				seen[id] = true
				walked = append(walked, id)
			}
			pages++
			if page.NextCursor == nil {
				break
			}
			cursor = *page.NextCursor
		}
		if pages != 31 {
			t.Fatalf("walked %d pages, want 31 (61 items at 2 per page)", pages)
		}
		if len(walked) != len(wantOrder) {
			t.Fatalf("walked %d items, want %d", len(walked), len(wantOrder))
		}
		for i, want := range wantOrder {
			if walked[i] != want {
				t.Fatalf("walked order[%d] = %s, want %s (walked %v)", i, walked[i], want, walked)
			}
		}
		// The tie pair stays adjacent across its page boundary, the
		// higher id first.
		for i, id := range walked {
			if id == bulkAlbumAssetID(4) && i+1 < len(walked) && walked[i+1] != bulkAlbumAssetID(3) {
				t.Errorf("the tie pair must stay adjacent, got %s after %s", walked[i+1], id)
			}
		}
	})

	t.Run("DetailShapeAndItems", func(t *testing.T) {
		page := env.getAlbumDetail(t, env.cookie, album, "limit=5")
		if uuid.UUID(page.Id) != album {
			t.Errorf("detail id = %s, want %s", page.Id, album)
		}
		if page.Title != "The long walk" {
			t.Errorf("detail title = %q, want the stored value", page.Title)
		}
		if page.Type != apigen.AlbumDetailTypeVirtual {
			t.Errorf("detail type = %q, want virtual", page.Type)
		}
		if page.Locked {
			t.Error("detail locked = true, want false")
		}
		if page.CoverAssetId != nil {
			t.Errorf("detail coverAssetId = %v, want null (no pin)", *page.CoverAssetId)
		}
		if page.CallerRole != apigen.AlbumDetailCallerRoleOwner {
			t.Errorf("detail callerRole = %q, want owner", page.CallerRole)
		}
		if page.ItemCount != 61 {
			t.Errorf("detail itemCount = %d, want 61 (the live members only)", page.ItemCount)
		}
		first := page.Items[0]
		if uuid.UUID(first.AssetId) != wantOrder[0] {
			t.Errorf("items[0].assetId = %s, want %s", first.AssetId, wantOrder[0])
		}
		if first.AddedAt.IsZero() {
			t.Error("items[0].addedAt = zero, want the membership's insert time")
		}
		if !first.Asset.TakenAt.Truncate(time.Microsecond).Equal(takenOf(0)) {
			t.Errorf("items[0].asset.takenAt = %v, want %v", first.Asset.TakenAt, takenOf(0))
		}
		if got := first.Asset.OriginalFilename; got != "fixture-"+bulkAlbumAssetID(0).String()+".jpg" {
			t.Errorf("items[0].asset.originalFilename = %q, want the fixture name", got)
		}
		if first.Asset.Kind != apigen.AlbumItemAssetKindImage {
			t.Errorf("items[0].asset.kind = %q, want image", first.Asset.Kind)
		}
		wantThumbnail := "/api/v1/assets/" + bulkAlbumAssetID(0).String() + "/thumbnail?size=tile"
		if first.Asset.Thumbnail != wantThumbnail {
			t.Errorf("items[0].asset.thumbnail = %q, want %q", first.Asset.Thumbnail, wantThumbnail)
		}
	})
}

// TestAlbumDetailNotFound pins the single-shape 404: an unknown id,
// another account's unshared album, another account's sync album
// (even carrying a share row), and an album whose share to the
// caller was revoked all answer the same byte-identical
// ALBUM_NOT_FOUND. The caller's OWN sync album is no longer in this
// region: it answers the read-only 200 of TestSyncAlbumDetail.
func TestAlbumDetailNotFound(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	adminID := env.sessionUserID(t)
	graceID, _ := env.createSecondUser(t, "album-404-grace@example.com")

	unshared := fixtureWallAlbumID("41")
	env.insertWallAlbum(t, unshared, graceID, "virtual", "Private", wallAlbumOpts{})

	graceSync := fixtureWallAlbumID("42")
	env.insertWallAlbum(t, graceSync, graceID, "sync", "Sync", wallAlbumOpts{})
	env.shareWallAlbum(t, graceSync, adminID, graceID, time.Date(2026, 11, 5, 8, 0, 0, 0, time.UTC))

	revoked := fixtureWallAlbumID("43")
	env.insertWallAlbum(t, revoked, graceID, "virtual", "Revoked", wallAlbumOpts{})
	env.shareWallAlbum(t, revoked, adminID, graceID, time.Date(2026, 11, 1, 8, 0, 0, 0, time.UTC))
	env.exec(t, "DELETE FROM album_shares WHERE album_id = $1 AND user_id = $2",
		pgUUID(revoked), pgUUID(adminID))

	unknown := fixtureWallAlbumID("ff")

	recorder := func(id uuid.UUID) *httptest.ResponseRecorder {
		return env.getAlbumsRaw(t, env.cookie, "/api/v1/albums/"+id.String())
	}

	reference := recorder(unknown)
	requireAlbumNotFound(t, reference)
	for name, id := range map[string]uuid.UUID{
		"unknown":             unknown,
		"unshared-foreign":    unshared,
		"foreign-sync-shared": graceSync,
		"revoked-share":       revoked,
	} {
		t.Run(name, func(t *testing.T) {
			rec := recorder(id)
			requireAlbumNotFound(t, rec)
			if rec.Body.String() != reference.Body.String() {
				t.Fatalf("404 body = %q, want byte-identical to the unknown id's %q",
					rec.Body.String(), reference.Body.String())
			}
		})
	}
}

// TestSyncAlbumDetail pins the caller's own sync album's detail: the
// read-only default-album face. The read answers 200 with the owner
// role, the live members keyset-paged exactly like any album's, the
// excludable field absent, and the owner's shares list present and
// empty (no share row can name a sync album). Another account's sync
// album stays in the 404 region. The WRITE faces' refusals of the
// same album are pinned where they live: TestUpdateAlbum's
// RefusesSyncAlbum (PATCH), TestRemoveAlbumItems (the membership
// removal), TestUpdateAlbumAudienceAlbumTypes (the audience write),
// TestSetAlbumTimelinePreference's SyncAndOwnVirtualRefused
// (the preference), and TestDeleteAlbum (the delete) -- the read
// admitting the caller's own sync changes none of them.
func TestSyncAlbumDetail(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	adminID := env.sessionUserID(t)
	graceID, graceCookie := env.createSecondUser(t, "sync-detail-grace@example.com")
	ownSync := env.syncAlbumIDOf(t, adminID)
	// Grace's own sync album (the fixture insert: createSecondUser
	// writes a bare user row, no wizard).
	graceSync := fixtureWallAlbumID("78")
	env.insertWallAlbum(t, graceSync, graceID, "sync", "Sync", wallAlbumOpts{})

	// Five live members plus the non-live newest of all: the count
	// and the pages see the live ones only.
	base := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	wantOrder := make([]uuid.UUID, 0, 5)
	for i := 0; i < 5; i++ {
		id := fixtureWallAssetID(fmt.Sprintf("7%d", i))
		env.insertTimelineAsset(t, id, adminID, base.Add(time.Duration(5-i)*time.Hour), timelineAssetOpts{})
		env.addTimelineItem(t, ownSync, id, adminID)
		wantOrder = append(wantOrder, id)
	}
	for _, row := range []struct {
		id   uuid.UUID
		opts timelineAssetOpts
	}{
		{fixtureWallAssetID("75"), timelineAssetOpts{isHidden: true}},
		{fixtureWallAssetID("76"), timelineAssetOpts{deleted: true}},
		{fixtureWallAssetID("77"), timelineAssetOpts{isComponent: true}},
	} {
		env.insertTimelineAsset(t, row.id, adminID, base.Add(time.Duration(9)*time.Hour), row.opts)
		env.addTimelineItem(t, ownSync, row.id, adminID)
	}

	t.Run("OwnSyncAnswersReadOnlyDetail", func(t *testing.T) {
		page := env.getAlbumDetail(t, env.cookie, ownSync, "limit=2")
		if uuid.UUID(page.Id) != ownSync {
			t.Errorf("detail id = %s, want the caller's sync album %s", page.Id, ownSync)
		}
		// The stored title is the internal system label; the CLIENT
		// renders the localized display name off the type (the
		// contract's convention, re-pinned here).
		if page.Title != "Sync" {
			t.Errorf("detail title = %q, want the internal label %q", page.Title, "Sync")
		}
		if page.Type != apigen.AlbumDetailTypeSync {
			t.Errorf("detail type = %q, want sync (the default album)", page.Type)
		}
		if page.Locked {
			t.Error("detail locked = true, want false (system-managed, never locked)")
		}
		if page.CoverAssetId != nil {
			t.Errorf("detail coverAssetId = %v, want null (no pin can land on a sync album)", *page.CoverAssetId)
		}
		if page.CallerRole != apigen.AlbumDetailCallerRoleOwner {
			t.Errorf("detail callerRole = %q, want owner (the caller's own stream)", page.CallerRole)
		}
		if page.ItemCount != 5 {
			t.Errorf("detail itemCount = %d, want 5 (the live members only)", page.ItemCount)
		}
		if page.InTimeline != nil {
			t.Errorf("detail inTimeline = %v, want absent (the core stream is never excludable)", page.InTimeline)
		}
		// The owner's view carries the grantee list, always empty
		// here: no share row can name a sync album.
		if page.Shares == nil || len(*page.Shares) != 0 {
			t.Errorf("detail shares = %v, want the owner's present empty list", page.Shares)
		}

		// The keyset walk: 5 live members at 2 per page.
		seen := make([]uuid.UUID, 0, 5)
		cursor := ""
		pages := 0
		for {
			rawQuery := "limit=2"
			if cursor != "" {
				rawQuery += "&cursor=" + cursor
			}
			page := env.getAlbumDetail(t, env.cookie, ownSync, rawQuery)
			seen = append(seen, idsOfDetailItems(page)...)
			pages++
			if page.NextCursor == nil {
				break
			}
			cursor = *page.NextCursor
		}
		if pages != 3 {
			t.Fatalf("walked %d pages, want 3 (5 items at 2 per page)", pages)
		}
		if len(seen) != len(wantOrder) {
			t.Fatalf("walked %d items, want %d", len(seen), len(wantOrder))
		}
		for i, want := range wantOrder {
			if seen[i] != want {
				t.Fatalf("walked order[%d] = %s, want %s (walked %v)", i, seen[i], want, seen)
			}
		}
	})

	t.Run("ForeignSyncStaysNotFound", func(t *testing.T) {
		// Grace's own sync album, seen by the admin: the same
		// byte-identical 404 as an unknown id -- and Grace herself
		// reads her own (the read is owner-scoped, not type-blind).
		reference := env.getAlbumsRaw(t, env.cookie,
			"/api/v1/albums/"+fixtureWallAlbumID("ff").String())
		requireAlbumNotFound(t, reference)
		rec := env.getAlbumsRaw(t, env.cookie, "/api/v1/albums/"+graceSync.String())
		requireAlbumNotFound(t, rec)
		if rec.Body.String() != reference.Body.String() {
			t.Fatalf("the foreign sync's 404 body %q differs from the unknown id's %q",
				rec.Body.String(), reference.Body.String())
		}
		own := env.getAlbumDetail(t, graceCookie, graceSync, "")
		if uuid.UUID(own.Id) != graceSync || own.Type != apigen.AlbumDetailTypeSync {
			t.Errorf("grace's own sync detail = %s/%q, want her own sync album", own.Id, own.Type)
		}
	})
}

// idsOfDetailItems collects one detail page's item asset ids.
func idsOfDetailItems(page apigen.AlbumDetail) []uuid.UUID {
	ids := make([]uuid.UUID, 0, len(page.Items))
	for _, item := range page.Items {
		ids = append(ids, uuid.UUID(item.AssetId))
	}
	return ids
}

// TestAlbumDetailViewer walks the detail of an album shared to the
// caller: the viewer role, the honest items of another account's
// assets, and the owner's own view of the same album. The validation
// matrix of the page parameters rides on the same fixture.
func TestAlbumDetailViewer(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	adminID := env.sessionUserID(t)
	graceID, graceCookie := env.createSecondUser(t, "viewer-grace@example.com")

	shared := fixtureWallAlbumID("51")
	env.insertWallAlbum(t, shared, graceID, "virtual", "For the family", wallAlbumOpts{
		locked: true,
	})
	newer := fixtureWallAssetID("51")
	older := fixtureWallAssetID("52")
	env.insertTimelineAsset(t, newer, graceID, time.Date(2026, 7, 2, 12, 0, 0, 0, time.UTC), timelineAssetOpts{})
	env.insertTimelineAsset(t, older, graceID, time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC), timelineAssetOpts{})
	env.addTimelineItem(t, shared, newer, graceID)
	env.addTimelineItem(t, shared, older, graceID)
	env.shareWallAlbum(t, shared, adminID, graceID, time.Date(2026, 11, 5, 9, 0, 0, 0, time.UTC))

	t.Run("ViewerSeesSharedAlbum", func(t *testing.T) {
		detail := env.getAlbumDetail(t, env.cookie, shared, "")
		if detail.CallerRole != apigen.AlbumDetailCallerRoleViewer {
			t.Errorf("callerRole = %q, want viewer for the sharee", detail.CallerRole)
		}
		if detail.Title != "For the family" {
			t.Errorf("title = %q, want the stored value", detail.Title)
		}
		if !detail.Locked {
			t.Error("locked = false, want true (metadata to a viewer)")
		}
		if detail.ItemCount != 2 {
			t.Errorf("itemCount = %d, want 2", detail.ItemCount)
		}
		want := []uuid.UUID{newer, older}
		if len(detail.Items) != len(want) {
			t.Fatalf("items = %d, want %d", len(detail.Items), len(want))
		}
		for i, id := range want {
			if uuid.UUID(detail.Items[i].AssetId) != id {
				t.Errorf("items[%d].assetId = %s, want %s (taken_at descending)", i, detail.Items[i].AssetId, id)
			}
		}
	})

	t.Run("OwnerSeesSameAlbum", func(t *testing.T) {
		detail := env.getAlbumDetail(t, graceCookie, shared, "")
		if detail.CallerRole != apigen.AlbumDetailCallerRoleOwner {
			t.Errorf("callerRole = %q, want owner for grace", detail.CallerRole)
		}
		if len(detail.Items) != 2 {
			t.Fatalf("items = %d, want 2", len(detail.Items))
		}
	})

	t.Run("RejectsInvalidPageParams", func(t *testing.T) {
		for _, raw := range []string{
			"limit=0", "limit=201", "limit=abc", "cursor=not-a-cursor",
		} {
			rec := env.getAlbumsRaw(t, env.cookie, "/api/v1/albums/"+shared.String()+"?"+raw)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("GET ?%s status = %d, want %d (body %q)",
					raw, rec.Code, http.StatusBadRequest, rec.Body.String())
			}
			apiErr, _ := decodeError(t, rec)
			if apiErr.Code != apigen.VALIDATIONFAILED {
				t.Errorf("GET ?%s code = %q, want %q", raw, apiErr.Code, apigen.VALIDATIONFAILED)
			}
		}

		// The cap boundary itself answers, it does not reject.
		page := env.getAlbumDetail(t, env.cookie, shared, "limit=200")
		if len(page.Items) != 2 {
			t.Fatalf("limit=200 page holds %d items, want all 2", len(page.Items))
		}
		if page.NextCursor != nil {
			t.Error("nextCursor set on the final page, want absent")
		}
	})
}

// TestAlbumDetailDeletionFaces pins the deletion chain's detail
// faces (E6, §7.9): the source-writable fact rides directory albums
// on both roles, the deletability switch and the derived `deletable`
// face ride the owner's view alone, and the derivation is honest
// over every arm -- a read-only source or a switched-off or locked
// album answers false without any runtime probing, and a viewer
// never receives the write faces at all.
func TestAlbumDetailDeletionFaces(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	adminID := env.sessionUserID(t)
	graceID, graceCookie := env.createSecondUser(t, "deletion-faces-grace@example.com")
	root := env.provisionMountRoot(t)

	// A writable-source directory album, shared to grace for the
	// viewer view of the same album.
	writable := uuid.UUID(env.createAlbum(t, env.cookie, "Writable source",
		sourceDir(t, root, "writable"), nil).Id)
	env.shareWallAlbum(t, writable, graceID, adminID, time.Date(2026, 11, 8, 9, 0, 0, 0, time.UTC))

	// A read-only-source directory album: the directory is made
	// read-only BEFORE the creation, so the probe itself fails and
	// the stored fact is false (skipped as root: the kernel lets
	// root write anything).
	readOnlyDir := sourceDir(t, root, "read-only")
	if os.Geteuid() != 0 {
		if err := os.Chmod(readOnlyDir, 0o555); err != nil {
			t.Fatalf("chmod the read-only source: %v", err)
		}
		t.Cleanup(func() { _ = os.Chmod(readOnlyDir, 0o755) })
	}
	readOnly := uuid.UUID(env.createAlbum(t, env.cookie, "Read-only source", readOnlyDir, nil).Id)

	// A legacy virtual album: no faces at all.
	virtual := fixtureWallAlbumID("76")
	env.insertWallAlbum(t, virtual, adminID, "virtual", "Legacy picks", wallAlbumOpts{})

	t.Run("OwnerOfAWritableUnlockedAlbum", func(t *testing.T) {
		detail := env.getAlbumDetail(t, env.cookie, writable, "")
		if detail.SourceWritable == nil || !*detail.SourceWritable {
			t.Errorf("sourceWritable = %v, want true (the probe passed)", detail.SourceWritable)
		}
		if detail.AllowAppDeletion == nil || !*detail.AllowAppDeletion {
			t.Errorf("allowAppDeletion = %v, want true (the default)", detail.AllowAppDeletion)
		}
		if detail.Deletable == nil || !*detail.Deletable {
			t.Errorf("deletable = %v, want true (owner, writable, on, unlocked)", detail.Deletable)
		}
	})

	t.Run("SwitchOffKillsTheAffordance", func(t *testing.T) {
		env.patchAlbum(t, env.cookie, writable, `{"allowAppDeletion":false}`)
		detail := env.getAlbumDetail(t, env.cookie, writable, "")
		if detail.SourceWritable == nil || !*detail.SourceWritable {
			t.Errorf("sourceWritable = %v, want true (the fact is untouched by the switch)", detail.SourceWritable)
		}
		if detail.AllowAppDeletion == nil || *detail.AllowAppDeletion {
			t.Errorf("allowAppDeletion = %v, want false (switched off)", detail.AllowAppDeletion)
		}
		if detail.Deletable == nil || *detail.Deletable {
			t.Errorf("deletable = %v, want false (the switch is off)", detail.Deletable)
		}
		env.patchAlbum(t, env.cookie, writable, `{"allowAppDeletion":true}`) // back on
	})

	t.Run("LockKillsTheAffordance", func(t *testing.T) {
		env.patchAlbum(t, env.cookie, writable, `{"locked":true}`)
		detail := env.getAlbumDetail(t, env.cookie, writable, "")
		if detail.AllowAppDeletion == nil || !*detail.AllowAppDeletion {
			t.Errorf("allowAppDeletion = %v, want true (the lock does not flip the switch)", detail.AllowAppDeletion)
		}
		if detail.Deletable == nil || *detail.Deletable {
			t.Errorf("deletable = %v, want false (locked)", detail.Deletable)
		}
		env.patchAlbum(t, env.cookie, writable, `{"locked":false}`)
	})

	t.Run("ReadOnlySourceKillsTheAffordance", func(t *testing.T) {
		if os.Geteuid() == 0 {
			t.Skip("the read-only arm is untestable as root: the kernel lets root write anything")
		}
		detail := env.getAlbumDetail(t, env.cookie, readOnly, "")
		if detail.SourceWritable == nil || *detail.SourceWritable {
			t.Errorf("sourceWritable = %v, want false (the probe failed over the read-only source)", detail.SourceWritable)
		}
		if detail.AllowAppDeletion == nil || !*detail.AllowAppDeletion {
			t.Errorf("allowAppDeletion = %v, want true (the stored switch, governing nothing)", detail.AllowAppDeletion)
		}
		if detail.Deletable == nil || *detail.Deletable {
			t.Errorf("deletable = %v, want false (the source is read-only, whatever the switch says)", detail.Deletable)
		}
	})

	t.Run("UnwritableSourceKillsTheAffordance", func(t *testing.T) {
		// The root-proof arm of the same derivation: procfs refuses
		// the probe's write for uid 0 too, so the containerized
		// gate sees the false fact's faces even where the
		// permission arms skip.
		if _, err := os.Stat("/proc"); err != nil {
			t.Skipf("/proc is unavailable on this host (%v); the unwritable arm needs procfs", err)
		}
		unwritable := uuid.UUID(env.createAlbum(t, env.cookie, "Unwritable source", "/proc", nil).Id)
		detail := env.getAlbumDetail(t, env.cookie, unwritable, "")
		if detail.SourceWritable == nil || *detail.SourceWritable {
			t.Errorf("sourceWritable = %v, want false (the probe failed over the unwritable source)", detail.SourceWritable)
		}
		if detail.Deletable == nil || *detail.Deletable {
			t.Errorf("deletable = %v, want false (the source is unwritable, whatever the switch says)", detail.Deletable)
		}
	})

	t.Run("ViewerSeesOnlyTheFact", func(t *testing.T) {
		detail := env.getAlbumDetail(t, graceCookie, writable, "")
		if detail.SourceWritable == nil || !*detail.SourceWritable {
			t.Errorf("sourceWritable = %v, want true (the fact rides the viewer's view too)", detail.SourceWritable)
		}
		if detail.AllowAppDeletion != nil {
			t.Errorf("allowAppDeletion = %v, want absent on the viewer's view (no write affordance to read it for)", detail.AllowAppDeletion)
		}
		if detail.Deletable != nil {
			t.Errorf("deletable = %v, want absent on the viewer's view (a sharee holds no delete affordance)", detail.Deletable)
		}
	})

	t.Run("VirtualAlbumCarriesNoFaces", func(t *testing.T) {
		detail := env.getAlbumDetail(t, env.cookie, virtual, "")
		if detail.SourceWritable != nil || detail.AllowAppDeletion != nil || detail.Deletable != nil {
			t.Errorf("deletion faces = (%v, %v, %v), want all absent on a virtual album",
				detail.SourceWritable, detail.AllowAppDeletion, detail.Deletable)
		}
	})
}
