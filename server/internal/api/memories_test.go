// Memory creation tests: the every-member gate (a plain member's
// POST succeeds -- curation is personal expression, not
// configuration), the response card's creation-time shape, the
// timeline-exclusion row that lands with the memory (an own memory
// never feeds the creator's aggregated timeline), the audience's
// full-set convergence through the creation, and the event family
// the memory rides. The membership curation add's re-share ground
// and refusal matrix land here too; the shared views' live filter
// lands in its own test below.
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

// fixtureMemoryAlbumID builds a deterministic fixture album id of
// the memory tests, in a namespace disjoint from every other
// fixture family (the wall's 50.. and the ad-hoc 70.. ids).
func fixtureMemoryAlbumID(suffix string) uuid.UUID {
	return uuid.MustParse("80000000-0000-0000-0000-0000000000" + suffix)
}

// fixtureMemoryAssetID builds a deterministic fixture asset id of
// the memory tests, in a namespace disjoint from every other
// fixture family.
func fixtureMemoryAssetID(suffix string) uuid.UUID {
	return uuid.MustParse("90000000-0000-0000-0000-0000000000" + suffix)
}

// TestMemorySharedViewsLiveFilter pins the re-share ground at
// display time: a sharee's every view of a shared memory -- the
// detail's pages, its count, the wall card's count and collage --
// carries the live members the memory's owner CURRENTLY sees. The
// source-revocation cascade falls out live: when the album that
// carried a photo to the owner is unshared, the photo leaves the
// sharee's memory views on the next read, no snapshot residue; the
// owner's own view keeps the plain live membership (the curation is
// the owner's to tidy, not the cascade's to rewrite).
func TestMemorySharedViewsLiveFilter(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	ownerID, ownerCookie := env.createSecondUser(t, "memory-owner-grace@example.com")
	viewerID, viewerCookie := env.createSecondUser(t, "memory-viewer-hopper@example.com")

	// Grace (the owner) holds two photos through a directory album
	// Hopper shares with her: the photos live in Hopper's album,
	// Grace sees them through the share -- the re-share's own
	// ground. The own upload rides her sync arm.
	source := fixtureMemoryAlbumID("b1")
	env.insertTimelineAlbum(t, source, viewerID, "directory")
	env.shareTimelineAlbum(t, source, ownerID, viewerID)
	base := time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC)
	own := fixtureMemoryAssetID("21")      // the owner's upload
	viaShare := fixtureMemoryAssetID("22") // lives in Hopper's album, seen by the owner through the share
	env.insertOwnedAsset(t, own, ownerID, base, timelineAssetOpts{})
	env.insertTimelineAsset(t, viaShare, viewerID, base.Add(time.Hour), timelineAssetOpts{})
	env.addTimelineItem(t, source, viaShare, viewerID)

	created := env.createMemory(t, ownerCookie, "The re-shared set", []uuid.UUID{viewerID})
	memory := uuid.UUID(created.Id)
	env.addMemoryItems(t, ownerCookie, memory, albumItemsBody(own, viaShare))
	// The pages order taken_at descending: viaShare (the newer) leads.
	requireMemoryMemberIDs(t, env.getAlbumDetail(t, viewerCookie, memory, ""), viaShare, own)

	t.Run("ShareeViewsCarryTheGround", func(t *testing.T) {
		detail := env.getAlbumDetail(t, viewerCookie, memory, "")
		if detail.ItemCount != 2 {
			t.Errorf("sharee itemCount = %d, want 2 (both members pass the owner's current sight)", detail.ItemCount)
		}
		requireMemoryMemberIDs(t, detail, viaShare, own)

		// The wall card agrees: the count and the collage frames are
		// the owner-visible live members, not the raw membership.
		card := memoryCardOf(t, env, viewerCookie, memory)
		if card.ItemCount != 2 {
			t.Errorf("sharee card itemCount = %d, want 2", card.ItemCount)
		}
		if got := frameSet(card.CoverAssetIds); len(got) != 2 || !got[viaShare] || !got[own] {
			t.Errorf("sharee card coverAssetIds = %v, want viaShare and own among them (the live owner-visible members)", card.CoverAssetIds)
		}
	})

	t.Run("SourceRevocationCascadesLive", func(t *testing.T) {
		// Hopper revokes the source album's share: the owner's sight
		// of viaShare dies, and with it every shared view of the
		// memory drops the photo -- the re-share never outlives the
		// right it was grounded in. The row itself stays (the set is
		// the owner's to tidy), the owner's own view keeps it, and
		// the owner's other surfaces (the timeline, the asset faces)
		// agree the photo is gone.
		env.exec(t, "DELETE FROM album_shares WHERE album_id = $1 AND user_id = $2",
			pgUUID(source), pgUUID(ownerID))

		detail := env.getAlbumDetail(t, viewerCookie, memory, "")
		if detail.ItemCount != 1 {
			t.Errorf("sharee itemCount after the revocation = %d, want 1 (the re-shared photo left live)", detail.ItemCount)
		}
		requireMemoryMemberIDs(t, detail, own)

		card := memoryCardOf(t, env, viewerCookie, memory)
		if card.ItemCount != 1 {
			t.Errorf("sharee card itemCount after the revocation = %d, want 1", card.ItemCount)
		}
		if len(card.CoverAssetIds) != 1 || uuid.UUID(card.CoverAssetIds[0]) != own {
			t.Errorf("sharee card coverAssetIds after the revocation = %v, want [own] (the only frame the caller can see)", card.CoverAssetIds)
		}

		// The owner's own view keeps the plain live membership: the
		// curation is the owner's, and the row stays until the owner
		// tidies it.
		ownerDetail := env.getAlbumDetail(t, ownerCookie, memory, "")
		if ownerDetail.ItemCount != 2 {
			t.Errorf("owner itemCount after the revocation = %d, want 2 (the owner's view keeps the live membership)", ownerDetail.ItemCount)
		}

		// The convergence signals the clients refetch on already
		// fire: the audience write's album.unshared for the revoked
		// source grant, and the membership itself publishes nothing
		// new -- the views recompute on every read, which is what
		// makes the drop live rather than announced.
		if events := eventsOfAlbum(t, env.bus, "album.items-changed", memory); len(events) != 1 {
			t.Errorf("album.items-changed events of the memory = %d, want still 1 (the revocation publishes no membership event)",
				len(events))
		}
	})

	t.Run("AlbumNarrowWalkCarriesTheGround", func(t *testing.T) {
		// The wall walk (the timeline endpoint's albumId narrow) is
		// the memory detail page's own tiles, and it carries the same
		// re-share ground as the pages, the counts and the collages:
		// the sharee's walk drops the revoked photo -- the page never
		// renders a grounded count beside a tile the count already
		// dropped -- while the owner's walk keeps the plain live
		// membership.
		shareeWalk := env.getTimeline(t, viewerCookie, "albumId="+memory.String())
		requireTimelineIDs(t, shareeWalk.Items, own)
		ownerWalk := env.getTimeline(t, ownerCookie, "albumId="+memory.String())
		requireTimelineIDs(t, ownerWalk.Items, viaShare, own)
	})

	t.Run("HiddenMemberLeavesSharedViewsLive", func(t *testing.T) {
		// The family-wide live predicate rides the same rule: a
		// hidden member leaves every view the moment the row hides.
		env.exec(t, "UPDATE assets SET is_hidden = true WHERE id = $1", pgUUID(own))
		detail := env.getAlbumDetail(t, viewerCookie, memory, "")
		if detail.ItemCount != 0 {
			t.Errorf("sharee itemCount after the hide = %d, want 0 (the hidden photo is invisible to the whole family)", detail.ItemCount)
		}
	})
}

// memoryCardOf resolves one memory's wall card for the given caller.
func memoryCardOf(t *testing.T, env *assetEnv, cookie *http.Cookie, memory uuid.UUID) apigen.AlbumSummary {
	t.Helper()
	wall := env.getAlbumWall(t, cookie)
	for _, card := range wall.Albums {
		if uuid.UUID(card.Id) == memory {
			return card
		}
	}
	t.Fatalf("the memory %s is missing from the wall (%v)", memory, wall.Albums)
	return apigen.AlbumSummary{}
}

// requireMemoryMemberIDs asserts the detail page carries exactly the
// given member ids, in the page's order.
func requireMemoryMemberIDs(t *testing.T, detail apigen.AlbumDetail, want ...uuid.UUID) {
	t.Helper()
	got := make([]uuid.UUID, 0, len(detail.Items))
	for _, item := range detail.Items {
		got = append(got, uuid.UUID(item.AssetId))
	}
	if len(got) != len(want) {
		t.Fatalf("detail members = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("detail member %d = %s, want %s (page %v)", i, got[i], want[i], got)
		}
	}
}

// createMemoryRaw fires POST /memories with the raw body (a nil
// cookie sends it unauthenticated).
func (e *assetEnv) createMemoryRaw(t *testing.T, cookie *http.Cookie, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/memories", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if cookie != nil {
		req.AddCookie(cookie)
	}
	rec := httptest.NewRecorder()
	e.router.ServeHTTP(rec, req)
	return rec
}

// createMemoryBody assembles the creation body: the title and the
// audience (a nil slice omits the field, an empty slice writes the
// explicit empty set).
func createMemoryBody(t *testing.T, title string, audience []uuid.UUID, extra string) string {
	t.Helper()
	quotedTitle, err := json.Marshal(title)
	if err != nil {
		t.Fatalf("quote title %q: %v", title, err)
	}
	body := `{"title":` + string(quotedTitle)
	if audience != nil {
		body += `,"audience":[` + albumIDList(audience) + `]`
	}
	if extra != "" {
		body += `,` + extra
	}
	return body + "}"
}

// createMemory fires POST /memories and decodes the 201 body.
func (e *assetEnv) createMemory(t *testing.T, cookie *http.Cookie, title string, audience []uuid.UUID) apigen.MemoryCreateResponse {
	t.Helper()
	rec := e.createMemoryRaw(t, cookie, createMemoryBody(t, title, audience, ""))
	if rec.Code != http.StatusCreated {
		t.Fatalf("POST /memories status = %d, want %d (body %q)",
			rec.Code, http.StatusCreated, rec.Body.String())
	}
	var response apigen.MemoryCreateResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode create memory body %q: %v", rec.Body.String(), err)
	}
	return response
}

// TestMemoryCreation pins the creation face: the EVERY-member gate
// (a plain member's POST succeeds -- the opposite of the directory
// album's administrator-only gate), the response card's
// creation-time shape, the title bounds, the audience's honest
// convergence, and the shape's refusal to carry any source field.
func TestMemoryCreation(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	adminID := env.sessionUserID(t)
	graceID, graceCookie := env.createSecondUser(t, "memory-grace@example.com")
	unknownID := uuid.MustParse("80000000-0000-0000-00ff-000000000001")

	t.Run("EveryMemberCreates", func(t *testing.T) {
		// THE gate pin: a plain member creates their own memory --
		// personal curation is no configuration act, so no role
		// check exists to refuse.
		created := env.createMemory(t, graceCookie, "Grace's picks", nil)
		if created.CallerRole != apigen.MemoryCreateResponseCallerRoleOwner {
			t.Errorf("callerRole = %q, want owner (the member owns what they create)", created.CallerRole)
		}
		if got := env.queryInt(t, "SELECT count(*) FROM albums WHERE id = $1 AND type = 'memory' AND owner_id = $2 AND root_path IS NULL",
			pgUUID(uuid.UUID(created.Id)), pgUUID(graceID)); got != 1 {
			t.Errorf("stored memory rows = %d, want 1 (a bare reference-set row: owner, title, type, no path)", got)
		}
	})

	t.Run("CardShapeAndWall", func(t *testing.T) {
		created := env.createMemory(t, env.cookie, "Road trip 2026", []uuid.UUID{graceID})
		if created.Title != "Road trip 2026" {
			t.Errorf("title = %q, want the stored title", created.Title)
		}
		if created.Type != apigen.MemoryCreateResponseTypeMemory {
			t.Errorf("type = %q, want memory", created.Type)
		}
		if created.IsShared != true {
			t.Errorf("isShared = %v, want true (the audience landed a row)", created.IsShared)
		}
		if created.CoverAssetIds == nil || len(created.CoverAssetIds) != 0 {
			t.Errorf("coverAssetIds = %v, want an empty array (no members)", created.CoverAssetIds)
		}
		if created.ItemCount != 0 {
			t.Errorf("itemCount = %d, want 0", created.ItemCount)
		}
		if created.InTimeline != nil {
			// An own memory card omits the field: it never feeds the
			// owner's timeline, so there is nothing to exclude.
			t.Errorf("inTimeline = %v, want absent", created.InTimeline)
		}
		if created.Owner != nil {
			t.Errorf("owner = %+v, want absent on an own card", created.Owner)
		}
		if created.Skipped == nil || len(created.Skipped) != 0 {
			t.Errorf("skipped = %v, want the empty array (the audience landed)", created.Skipped)
		}
		if got := env.queryInt(t, "SELECT count(*) FROM albums WHERE id = $1 AND type = 'memory' AND owner_id = $2 AND root_path IS NULL",
			pgUUID(uuid.UUID(created.Id)), pgUUID(adminID)); got != 1 {
			t.Errorf("stored memory rows = %d, want 1 (a bare row: no source path exists to store)", got)
		}

		// The wall lists the fresh card after the pinned sync card.
		wall := env.getAlbumWall(t, env.cookie)
		found := false
		for _, card := range wall.Albums {
			if uuid.UUID(card.Id) == uuid.UUID(created.Id) {
				found = true
				if card.Type != apigen.AlbumSummaryTypeMemory {
					t.Errorf("wall card type = %q, want memory", card.Type)
				}
				if card.InTimeline != nil {
					t.Errorf("wall card inTimeline = %v, want absent", card.InTimeline)
				}
			}
		}
		if !found {
			t.Fatalf("the fresh memory is missing from the wall (%v)", wall.Albums)
		}

		// The detail carries no source line: rootPath is the
		// directory album's own field.
		detail := env.getAlbumDetail(t, env.cookie, uuid.UUID(created.Id), "")
		if detail.RootPath != nil {
			t.Errorf("detail rootPath = %v, want absent (a memory has no source)", detail.RootPath)
		}
		if detail.InTimeline != nil {
			t.Errorf("detail inTimeline = %v, want absent", detail.InTimeline)
		}
	})

	t.Run("SourceFieldCarriesNoHome", func(t *testing.T) {
		// A client sending the directory album's own fields names
		// fields this shape does not carry: the JSON decoder drops
		// them, the row stores nothing of them. The refusal is the
		// shape's own -- no error, no home.
		created := env.createMemoryRaw(t, env.cookie,
			createMemoryBody(t, "Shape probe", nil, `"rootPath":"/Volumes/Photos","allowAppDeletion":false,"inTimeline":true`))
		if created.Code != http.StatusCreated {
			t.Fatalf("POST /memories status = %d, want %d (body %q)",
				created.Code, http.StatusCreated, created.Body.String())
		}
		var response apigen.MemoryCreateResponse
		if err := json.Unmarshal(created.Body.Bytes(), &response); err != nil {
			t.Fatalf("decode create memory body %q: %v", created.Body.String(), err)
		}
		if got := env.queryInt(t, "SELECT count(*) FROM albums WHERE id = $1 AND root_path IS NOT NULL",
			pgUUID(uuid.UUID(response.Id))); got != 0 {
			t.Errorf("rows with a stored source path = %d, want 0 (a memory owns no path)", got)
		}
	})

	t.Run("TrimsTitle", func(t *testing.T) {
		created := env.createMemory(t, env.cookie, "  Padded  ", nil)
		if created.Title != "Padded" {
			t.Errorf("title = %q, want the trimmed %q", created.Title, "Padded")
		}
	})

	t.Run("ValidatesTitle", func(t *testing.T) {
		for _, title := range []string{
			"",                      // missing entirely
			"   ",                   // whitespace only after the trim
			strings.Repeat("m", 65), // past the bound
		} {
			rec := env.createMemoryRaw(t, env.cookie, createMemoryBody(t, title, nil, ""))
			requireValidationFields(t, rec, "title")
		}
		// A body that does not decode is a client bug of its own
		// class (the body, not the title).
		rec := env.createMemoryRaw(t, env.cookie, `not json`)
		requireValidationFields(t, rec, "body")
		// The bound itself is inclusive: 64 runes land, counted
		// over runes so multi-byte titles stay within it.
		rec = env.createMemoryRaw(t, env.cookie, createMemoryBody(t, strings.Repeat("梦", 64), nil, ""))
		if rec.Code != http.StatusCreated {
			t.Errorf("POST /memories of a 64-rune title status = %d, want %d (body %q)",
				rec.Code, http.StatusCreated, rec.Body.String())
		}
	})

	t.Run("AudienceLandsAndSkipsHonestly", func(t *testing.T) {
		created := env.createMemory(t, env.cookie, "Shared picks", []uuid.UUID{graceID, adminID, unknownID})
		if len(created.Skipped) != 2 {
			t.Fatalf("skipped = %v, want the owner's own id and the unknown id", created.Skipped)
		}
		reasons := map[uuid.UUID]string{}
		for _, skip := range created.Skipped {
			reasons[uuid.UUID(skip.UserId)] = string(skip.Reason)
		}
		if reasons[adminID] != "owner" {
			t.Errorf("skip reason of the caller's own id = %q, want owner", reasons[adminID])
		}
		if reasons[unknownID] != "invalid_account" {
			t.Errorf("skip reason of the unknown id = %q, want invalid_account", reasons[unknownID])
		}
		if got := env.shareRowsOf(t, uuid.UUID(created.Id)); got != 1 {
			t.Errorf("share rows = %d, want 1 (only the usable grantee landed)", got)
		}
	})

	t.Run("RequiresSession", func(t *testing.T) {
		rec := env.createMemoryRaw(t, nil, createMemoryBody(t, "No session", nil, ""))
		requireUnauthorized(t, rec)
	})
}

// TestMemoryTimelineExclusion pins the exclusion whole: the
// creation writes the creator's own preference row (inTimeline
// false), the aggregated timeline never feeds the memory's photos
// (an own memory is no arm of the union -- in-stream curation would
// be pure duplication), the wall card omits the switch, and the
// preference face refuses the own memory (the exclusion is
// structural, not a user switch to flip). The sharee's side feeds:
// a memory shared to them is their own shared arm, governed by
// their own preference row.
func TestMemoryTimelineExclusion(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	adminID := env.sessionUserID(t)
	graceID, graceCookie := env.createSecondUser(t, "memory-exclusion-grace@example.com")

	// The member's memory, created through the face, carrying two
	// photos that live NOWHERE else: their only album is the memory
	// itself, so the aggregate's answer for them is the memory's
	// answer alone.
	created := env.createMemory(t, graceCookie, "The kept set", []uuid.UUID{adminID})
	memory := uuid.UUID(created.Id)
	m1 := fixtureMemoryAssetID("01")
	m2 := fixtureMemoryAssetID("02")
	env.insertTimelineAsset(t, m1, graceID, time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC), timelineAssetOpts{})
	env.insertTimelineAsset(t, m2, graceID, time.Date(2026, 9, 2, 12, 0, 0, 0, time.UTC), timelineAssetOpts{})
	env.addTimelineItem(t, memory, m1, graceID)
	env.addTimelineItem(t, memory, m2, graceID)

	t.Run("ExclusionRowLandsAtBirth", func(t *testing.T) {
		// The row the creation writes: the creator's own preference,
		// false -- the durable statement the exclusion machinery
		// reads, whatever arms a later formula admits.
		stored := env.queryText(t, "SELECT in_timeline::text FROM user_album_prefs WHERE user_id = $1 AND album_id = $2",
			pgUUID(graceID), pgUUID(memory))
		if stored != "false" {
			t.Errorf("stored in_timeline = %q, want false (the memory is born out of the creator's timeline)", stored)
		}
	})

	t.Run("AggregateNeverFeedsTheCreator", func(t *testing.T) {
		// The creator's timeline: the memory's photos never surface
		// -- an own memory is no arm of the union, whatever the row
		// says. (Grace's timeline holds nothing at all here: she owns
		// no live asset outside the memory.)
		page := env.getTimeline(t, graceCookie, "limit=50")
		requireTimelineIDs(t, page.Items)
	})

	t.Run("SharedArmFeedsTheAudience", func(t *testing.T) {
		// The admin's timeline: the same photos arrive through the
		// memory's share -- the shared arm admits every type, and
		// the grantee's own preference row (absent here) keeps the
		// default of feeding.
		page := env.getTimeline(t, env.cookie, "limit=50")
		requireTimelineIDs(t, page.Items, m2, m1)
	})

	t.Run("PreferenceFaceRefusesTheOwnMemory", func(t *testing.T) {
		// The exclusion is not a user switch: the face's 400 keeps
		// the structural exclusion from being flipped off, and the
		// row stays as the creation wrote it.
		rec := env.setTimelinePreferenceRaw(t, graceCookie, memory, `{"inTimeline":true}`)
		requireAlbumPrefKey(t, rec, "album_pref_not_excludable")
		if rows := env.prefRowsOf(t, graceID, memory); rows != 1 {
			t.Errorf("preference rows = %d, want 1 (the birth row survives the refused write)", rows)
		}
	})
}

// TestMemoryCurationAdd pins the membership add's re-share ground
// and report: every id must be a photo the creator CURRENTLY sees
// as a live row -- an own upload (the sync arm), a photo of an
// album shared to the creator (the shared arm) -- and every other
// id (a foreign unshared photo, a trashed or hidden row, an id
// already a member) stays out as the honest skipped count. The
// event fires exactly when a row lands, and the type and
// authorization gates hold the memory's own shape.
func TestMemoryCurationAdd(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	adminID := env.sessionUserID(t)
	graceID, graceCookie := env.createSecondUser(t, "memory-curation-grace@example.com")

	created := env.createMemory(t, env.cookie, "The curated set", nil)
	memory := uuid.UUID(created.Id)

	// Grace's album shared to the admin: its member is a photo the
	// admin sees through the shared arm (the re-share's chain
	// ground -- a photo seen because someone shared it with you).
	sharedAlbum := fixtureMemoryAlbumID("a1")
	env.insertTimelineAlbum(t, sharedAlbum, graceID, "directory")
	env.shareTimelineAlbum(t, sharedAlbum, adminID, graceID)

	base := time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)
	own := fixtureMemoryAssetID("11")       // the admin's upload (sync arm)
	sharedIn := fixtureMemoryAssetID("12")  // grace's photo inside the album shared to the admin
	foreign := fixtureMemoryAssetID("13")   // grace's unshared photo -- invisible to the admin
	trashed := fixtureMemoryAssetID("14")   // the admin's own upload, trashed
	hidden := fixtureMemoryAssetID("15")    // the admin's own upload, hidden
	component := fixtureMemoryAssetID("16") // the admin's own upload, a Live Photo component row
	env.insertOwnedAsset(t, own, adminID, base, timelineAssetOpts{})
	env.insertTimelineAsset(t, sharedIn, graceID, base.Add(time.Hour), timelineAssetOpts{})
	env.addTimelineItem(t, sharedAlbum, sharedIn, graceID)
	env.insertTimelineAsset(t, foreign, graceID, base.Add(2*time.Hour), timelineAssetOpts{})
	env.insertOwnedAsset(t, trashed, adminID, base.Add(3*time.Hour), timelineAssetOpts{deleted: true})
	env.insertOwnedAsset(t, hidden, adminID, base.Add(4*time.Hour), timelineAssetOpts{isHidden: true})
	env.insertOwnedAsset(t, component, adminID, base.Add(5*time.Hour), timelineAssetOpts{isComponent: true})

	// A second memory shared to Grace: the sharee's refusal arm.
	sharedToGrace := fixtureMemoryAlbumID("a2")
	env.insertTimelineAlbum(t, sharedToGrace, adminID, "memory")
	env.shareTimelineAlbum(t, sharedToGrace, graceID, adminID)
	// A directory album and the caller's own sync: the type refusals.
	directory := fixtureMemoryAlbumID("a3")
	env.insertTimelineAlbum(t, directory, adminID, "directory")

	t.Run("ReShareGroundAdmitsTheVisibleLiveRows", func(t *testing.T) {
		report := env.addMemoryItems(t, env.cookie, memory, albumItemsBody(own, sharedIn, foreign, trashed, hidden, component))
		if report.Added != 2 || report.Skipped != 4 {
			t.Errorf("added/skipped = %d/%d, want 2/4 (the own upload and the shared album's member land; the foreign, trashed, hidden and component rows stay out)",
				report.Added, report.Skipped)
		}
		for _, id := range []uuid.UUID{own, sharedIn} {
			if got := env.queryInt(t, "SELECT count(*) FROM album_items WHERE album_id = $1 AND asset_id = $2",
				pgUUID(memory), pgUUID(id)); got != 1 {
				t.Errorf("membership rows for %s = %d, want 1", id, got)
			}
		}
		for _, id := range []uuid.UUID{foreign, trashed, hidden, component} {
			if got := env.queryInt(t, "SELECT count(*) FROM album_items WHERE album_id = $1 AND asset_id = $2",
				pgUUID(memory), pgUUID(id)); got != 0 {
				t.Errorf("membership rows for %s = %d, want 0 (the re-share ground refused it)", id, got)
			}
		}
	})

	t.Run("AlreadyMemberIsTheIdempotentSkip", func(t *testing.T) {
		// The same id again: no error, no row, no event -- the
		// set's own no-op.
		before := len(eventsOfAlbum(t, env.bus, "album.items-changed", memory))
		report := env.addMemoryItems(t, env.cookie, memory, albumItemsBody(own))
		if report.Added != 0 || report.Skipped != 1 {
			t.Errorf("added/skipped = %d/%d, want 0/1 (already a member)", report.Added, report.Skipped)
		}
		if after := len(eventsOfAlbum(t, env.bus, "album.items-changed", memory)); after != before {
			t.Errorf("album.items-changed events after the no-op = %d, want still %d", after, before)
		}
	})

	t.Run("CollapsesRepeatedIds", func(t *testing.T) {
		fresh := env.createMemory(t, env.cookie, "The repeat probe", nil)
		report := env.addMemoryItems(t, env.cookie, uuid.UUID(fresh.Id), albumItemsBody(own, own))
		if report.Added != 1 || report.Skipped != 0 {
			t.Errorf("added/skipped = %d/%d, want 1/0 (repeated ids collapse)", report.Added, report.Skipped)
		}
	})

	t.Run("EventFiresPerLandingWrite", func(t *testing.T) {
		fresh := env.createMemory(t, env.cookie, "The eventful add", nil)
		env.addMemoryItems(t, env.cookie, uuid.UUID(fresh.Id), albumItemsBody(own))
		events := eventsOfAlbum(t, env.bus, "album.items-changed", uuid.UUID(fresh.Id))
		if len(events) != 1 {
			t.Fatalf("album.items-changed events = %d, want exactly 1", len(events))
		}
		for _, payload := range events {
			if len(payload) != 2 {
				t.Errorf("album.items-changed payload = %v, want exactly the albumId/userId pair", payload)
			}
		}
	})

	t.Run("ValidatesListBounds", func(t *testing.T) {
		rec := env.addMemoryItemsRaw(t, env.cookie, memory, `{"assetIds":[]}`)
		requireValidationFields(t, rec, "assetIds")
		rec = env.addMemoryItemsRaw(t, env.cookie, memory, `not json`)
		requireValidationFields(t, rec, "body")
	})

	t.Run("ShareeIsForbidden", func(t *testing.T) {
		rec := env.addMemoryItemsRaw(t, graceCookie, sharedToGrace, albumItemsBody(own))
		requireAlbumWriteForbidden(t, rec)
	})

	t.Run("RefusesDirectoryAlbum", func(t *testing.T) {
		// A directory album is a source: its membership is its
		// scan's own verdict, never a hand-picked set.
		rec := env.addMemoryItemsRaw(t, env.cookie, directory, albumItemsBody(own))
		requireAlbumNotFound(t, rec)
	})

	t.Run("RefusesSyncAlbum", func(t *testing.T) {
		ownSync := env.syncAlbumIDOf(t, adminID)
		rec := env.addMemoryItemsRaw(t, env.cookie, ownSync, albumItemsBody(own))
		requireAlbumNotFound(t, rec)
	})

	t.Run("UnknownAlbum", func(t *testing.T) {
		rec := env.addMemoryItemsRaw(t, env.cookie, fixtureMemoryAlbumID("ff"), albumItemsBody(own))
		requireAlbumNotFound(t, rec)
	})

	t.Run("RequiresSession", func(t *testing.T) {
		rec := env.addMemoryItemsRaw(t, nil, memory, albumItemsBody(own))
		requireUnauthorized(t, rec)
	})
}

// addMemoryItemsRaw fires POST /albums/{albumId}/items with the raw
// body (a nil cookie sends it unauthenticated).
func (e *assetEnv) addMemoryItemsRaw(t *testing.T, cookie *http.Cookie, albumID uuid.UUID, body string) *httptest.ResponseRecorder {
	t.Helper()
	return e.writeAlbumRaw(t, cookie, http.MethodPost,
		"/api/v1/albums/"+albumID.String()+"/items", body)
}

// addMemoryItems fires POST /albums/{albumId}/items and decodes the
// 200 report.
func (e *assetEnv) addMemoryItems(t *testing.T, cookie *http.Cookie, albumID uuid.UUID, body string) apigen.AddAlbumItemsResponse {
	t.Helper()
	rec := e.addMemoryItemsRaw(t, cookie, albumID, body)
	if rec.Code != http.StatusOK {
		t.Fatalf("POST /albums/%s/items status = %d, want %d (body %q)",
			albumID, rec.Code, http.StatusOK, rec.Body.String())
	}
	var report apigen.AddAlbumItemsResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &report); err != nil {
		t.Fatalf("decode add items body %q: %v", rec.Body.String(), err)
	}
	return report
}

// TestMemoryCreationEvents pins the creation's events: one
// album.created with the ids-only payload (the album id and the
// acting user, nothing else), one album.shared per audience row
// that landed, and silence for the skips.
func TestMemoryCreationEvents(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	graceID, _ := env.createSecondUser(t, "memory-events-grace@example.com")
	unknownID := uuid.MustParse("80000000-0000-0000-00fe-000000000002")

	created := env.createMemory(t, env.cookie, "The eventful set", []uuid.UUID{graceID, unknownID})
	memory := uuid.UUID(created.Id)

	createdEvents := eventsOfAlbum(t, env.bus, "album.created", memory)
	if len(createdEvents) != 1 {
		t.Fatalf("album.created events = %d, want exactly 1", len(createdEvents))
	}
	for _, payload := range createdEvents {
		if len(payload) != 2 {
			t.Errorf("album.created payload = %v, want exactly the albumId/userId pair", payload)
		}
		if _, present := payload["title"]; present {
			t.Error("album.created payload carries the title -- private state must not ride the bus")
		}
	}

	shared := eventsOfAlbum(t, env.bus, "album.shared", memory)
	if len(shared) != 1 {
		t.Fatalf("album.shared events = %d, want 1 (the usable grantee alone; the skip stays silent)", len(shared))
	}
	if shared[0]["userId"] != graceID.String() {
		t.Errorf("album.shared userId = %q, want the grantee %s", shared[0]["userId"], graceID)
	}
}
