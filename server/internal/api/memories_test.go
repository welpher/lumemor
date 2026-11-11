// Memory creation tests: the every-member gate (a plain member's
// POST succeeds -- curation is personal expression, not
// configuration), the response card's creation-time shape, the
// timeline-exclusion row that lands with the memory (an own memory
// never feeds the creator's aggregated timeline), the audience's
// full-set convergence through the creation, and the event family
// the memory rides. The membership curation arms and the shared
// views' live filter land in their own tests (albums_write_test.go
// and the visibility suite).
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

// fixtureMemoryAssetID builds a deterministic fixture asset id of
// the memory tests, in a namespace disjoint from every other
// fixture family.
func fixtureMemoryAssetID(suffix string) uuid.UUID {
	return uuid.MustParse("90000000-0000-0000-0000-0000000000" + suffix)
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
