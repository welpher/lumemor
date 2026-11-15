// The cover's sampling tests: the weekly rotation and its freeze, the
// small-collection floor, the pin's precedence and its dead-cell
// fill. The instance clock is injected so a week boundary is one
// assignment away -- the rotation is derived from the ISO week of the
// instance's own instant, never a rolling window, so a fixed instant
// and a fixed instant seven days later are exactly what the boundary
// is.
package api

import (
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"

	apigen "github.com/welpher/lumemor/server/internal/api/gen"
)

// coverFixtureAssetID builds a deterministic asset id in the cover
// tests' own namespace (the wall fixtures' 60.. family, suffixed).
func coverFixtureAssetID(index int) uuid.UUID {
	return uuid.MustParse(fmt.Sprintf("60000000-0000-0000-0000-00000000c0%02d", index))
}

// insertCoverAlbum inserts one fixture album with count live members
// taken one hour apart (the newest first: index 0 is the newest).
func insertCoverAlbum(t *testing.T, env *assetEnv, albumID, owner uuid.UUID, albumType string, count int) []uuid.UUID {
	t.Helper()
	env.insertWallAlbum(t, albumID, owner, albumType, "Cover fixture", wallAlbumOpts{})
	base := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	ids := make([]uuid.UUID, 0, count)
	for index := 0; index < count; index++ {
		id := coverFixtureAssetID(index)
		env.insertTimelineAsset(t, id, owner, base.Add(-time.Duration(index)*time.Hour), timelineAssetOpts{})
		env.addTimelineItem(t, albumID, id, owner)
		ids = append(ids, id)
	}
	return ids
}

// wallCardOf reads the caller's wall card of one album.
func wallCardOf(t *testing.T, env *assetEnv, albumID uuid.UUID) apigen.AlbumSummary {
	t.Helper()
	wall := env.getAlbumWall(t, env.cookie)
	for _, card := range wall.Albums {
		if uuid.UUID(card.Id) == albumID {
			return card
		}
	}
	t.Fatalf("the wall holds no card for %s", albumID)
	return apigen.AlbumSummary{}
}

// idSet folds plain ids into a membership set (the fixture side of
// frameSet, which speaks the wire type).
func idSet(ids ...uuid.UUID) map[uuid.UUID]bool {
	set := make(map[uuid.UUID]bool, len(ids))
	for _, id := range ids {
		set[id] = true
	}
	return set
}

// TestIsoWeekOf pins the week identifier's boundaries: the instance
// location's Monday 00:00 is the turn, a Sunday 23:59:59 belongs to
// the week it ends, the padded form sorts, and a year holding a 53rd
// week (the ISO rule: the week of the year's last Thursday) is named
// by its own numbering rather than rolling into week 1.
func TestIsoWeekOf(t *testing.T) {
	berlin, err := time.LoadLocation("Europe/Berlin")
	if err != nil {
		t.Skipf("no zone database: %v", err)
	}
	cases := []struct {
		name string
		at   time.Time
		want string
	}{
		{
			name: "SundayNightBelongsToTheClosingWeek",
			at:   time.Date(2026, 11, 15, 23, 59, 59, 0, time.UTC),
			want: "2026-W46",
		},
		{
			name: "MondayMidnightTurns",
			at:   time.Date(2026, 11, 16, 0, 0, 0, 0, time.UTC),
			want: "2026-W47",
		},
		{
			name: "TheInstantLocationDecides",
			// 23:30 UTC Sunday is already Monday 00:30 in Berlin: the
			// instance zone's calendar turns the week, not UTC's.
			at:   time.Date(2026, 11, 15, 23, 30, 0, 0, time.UTC).In(berlin),
			want: "2026-W47",
		},
		{
			name: "A53WeekYear",
			// 2026-12-31 is a Thursday: ISO week 53 of 2026.
			at:   time.Date(2026, 12, 31, 12, 0, 0, 0, time.UTC),
			want: "2026-W53",
		},
		{
			name: "NewYearsDayKeepsTheOldWeek",
			// 2027-01-01 is a Friday: still the week of 2026's last
			// Thursday.
			at:   time.Date(2027, 1, 1, 12, 0, 0, 0, time.UTC),
			want: "2026-W53",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := isoWeekOf(tc.at); got != tc.want {
				t.Errorf("isoWeekOf(%s) = %s, want %s", tc.at, got, tc.want)
			}
		})
	}
}

// TestCoverRotationFreezesAndTurns walks the weekly rotation's whole
// story on one album: the same week answers the same group (two
// reads, then a brand-new upload mid-week), the next week samples a
// different group, and the stored rows are one group per week.
func TestCoverRotationFreezesAndTurns(t *testing.T) {
	instant := time.Date(2026, 11, 11, 10, 0, 0, 0, time.UTC) // a Wednesday, 2026-W46
	env := newAssetEnvWithClock(t, testUploadLimit, func() time.Time { return instant })
	adminID := env.sessionUserID(t)
	album := fixtureWallAlbumID("a1")
	members := insertCoverAlbum(t, env, album, adminID, "memory", 12)

	// The memory band holds six frames; twelve members clear the
	// twice-the-frame-count rotation floor.
	first := wallCardOf(t, env, album)
	if len(first.CoverAssetIds) != 6 {
		t.Fatalf("cover frames = %d, want 6 (the memory band's count)", len(first.CoverAssetIds))
	}
	for _, raw := range first.CoverAssetIds {
		if !idSet(members...)[uuid.UUID(raw)] {
			t.Errorf("cover frame %s is not a live member", raw)
		}
	}

	// The same week, a second read: the frozen group, id for id.
	second := wallCardOf(t, env, album)
	if fmt.Sprint(second.CoverAssetIds) != fmt.Sprint(first.CoverAssetIds) {
		t.Errorf("the group moved inside one week: %v then %v", first.CoverAssetIds, second.CoverAssetIds)
	}

	// A new upload mid-week cannot move a frame: the week's group was
	// resolved once and stays.
	added := coverFixtureAssetID(50)
	env.insertTimelineAsset(t, added, adminID, time.Date(2026, 11, 12, 9, 0, 0, 0, time.UTC), timelineAssetOpts{})
	env.addTimelineItem(t, album, added, adminID)
	third := wallCardOf(t, env, album)
	if fmt.Sprint(third.CoverAssetIds) != fmt.Sprint(first.CoverAssetIds) {
		t.Errorf("a mid-week upload moved the group: %v then %v", first.CoverAssetIds, third.CoverAssetIds)
	}

	// The next ISO week re-samples whole.
	instant = instant.AddDate(0, 0, 7) // the following Wednesday, 2026-W47
	turned := wallCardOf(t, env, album)
	if len(turned.CoverAssetIds) != 6 {
		t.Fatalf("the turned group holds %d frames, want 6", len(turned.CoverAssetIds))
	}
	if fmt.Sprint(turned.CoverAssetIds) == fmt.Sprint(first.CoverAssetIds) {
		t.Errorf("the turn kept the previous week's group: %v", turned.CoverAssetIds)
	}

	// The store holds one frozen group per week (the first week's six
	// rows and the second's).
	if got := env.queryInt(t,
		"SELECT count(DISTINCT iso_week) FROM album_cover_week_picks WHERE album_id = $1", pgUUID(album)); got != 2 {
		t.Errorf("stored weeks = %d, want 2 (the frozen groups of both reads)", got)
	}
}

// TestCoverSmallCollectionNeverRotates pins the floor: an album
// holding fewer than twice the frame count keeps one stable group
// across weeks (there is nothing to rotate through), and no week row
// is ever stored for it.
func TestCoverSmallCollectionNeverRotates(t *testing.T) {
	instant := time.Date(2026, 11, 11, 10, 0, 0, 0, time.UTC)
	env := newAssetEnvWithClock(t, testUploadLimit, func() time.Time { return instant })
	adminID := env.sessionUserID(t)
	album := fixtureWallAlbumID("a2")
	// Eleven members: one short of the memory band's rotation floor.
	insertCoverAlbum(t, env, album, adminID, "memory", 11)

	first := wallCardOf(t, env, album)
	if len(first.CoverAssetIds) != 6 {
		t.Fatalf("small-collection frames = %d, want 6 (a sample of the eleven)", len(first.CoverAssetIds))
	}
	// The freeze below the floor: a mid-week upload cannot move the
	// stored group (the week row answers before any size test). The
	// store is one row per frame, so the frozen group of six answers
	// as six rows.
	extra := uuid.MustParse("0192f5b3-6c7d-8e9f-a0b1-2c3d4e5f6f99")
	env.insertTimelineAsset(t, extra, adminID, time.Date(2026, 5, 2, 9, 0, 0, 0, time.UTC), timelineAssetOpts{})
	env.addTimelineItem(t, album, extra, adminID)
	midWeek := wallCardOf(t, env, album)
	if fmt.Sprint(midWeek.CoverAssetIds) != fmt.Sprint(first.CoverAssetIds) {
		t.Errorf("a mid-week upload moved the small group: %v then %v", first.CoverAssetIds, midWeek.CoverAssetIds)
	}
	if got := env.queryInt(t,
		"SELECT count(*) FROM album_cover_week_picks WHERE album_id = $1", pgUUID(album)); got != 6 {
		t.Errorf("small-collection week rows = %d, want 6 (one stored group of six frames)", got)
	}
	// The next week re-samples over the grown pool: the collection's
	// own truth (the new member earns its chance at the boundary),
	// never a mid-week move -- the rotation letter's "no rotation"
	// governs time, not the pool's membership.
	instant = instant.AddDate(0, 0, 14)
	second := wallCardOf(t, env, album)
	if len(second.CoverAssetIds) != 6 {
		t.Fatalf("the boundary re-sample frames = %d, want 6", len(second.CoverAssetIds))
	}
}

// TestCoverPinStopsRotation pins the manual path: a full pinned set
// answers in pick order across weeks, and no week row is stored for
// the album at all.
func TestCoverPinStopsRotation(t *testing.T) {
	instant := time.Date(2026, 11, 11, 10, 0, 0, 0, time.UTC)
	env := newAssetEnvWithClock(t, testUploadLimit, func() time.Time { return instant })
	adminID := env.sessionUserID(t)
	album := fixtureWallAlbumID("a3")
	members := insertCoverAlbum(t, env, album, adminID, "memory", 12)

	pin := []uuid.UUID{members[11], members[3], members[7], members[1], members[9], members[5]}
	summary := env.patchAlbum(t, env.cookie, album, coverPinBody(pin...))
	if len(summary.CoverAssetIds) != len(pin) {
		t.Fatalf("pinned card frames = %v, want the six picks", summary.CoverAssetIds)
	}
	for index, id := range pin {
		if uuid.UUID(summary.CoverAssetIds[index]) != id {
			t.Errorf("pinned frame %d = %s, want %s (pick order)", index, summary.CoverAssetIds[index], id)
		}
	}

	instant = instant.AddDate(0, 0, 21)
	later := wallCardOf(t, env, album)
	if fmt.Sprint(later.CoverAssetIds) != fmt.Sprint(summary.CoverAssetIds) {
		t.Errorf("the pin rotated: %v then %v", summary.CoverAssetIds, later.CoverAssetIds)
	}
	if got := env.queryInt(t,
		"SELECT count(*) FROM album_cover_week_picks WHERE album_id = $1", pgUUID(album)); got != 0 {
		t.Errorf("pinned-album week rows = %d, want 0 (a pin stops the rotation)", got)
	}

	// Restoring the automatic group clears the pin and resumes the
	// rotation: the card answers a sampled group again.
	env.patchAlbum(t, env.cookie, album, `{"coverAssetIds":[]}`)
	restored := wallCardOf(t, env, album)
	if len(restored.CoverAssetIds) != 6 {
		t.Fatalf("restored frames = %d, want 6 (the automatic group)", len(restored.CoverAssetIds))
	}
	if fmt.Sprint(restored.CoverAssetIds) == fmt.Sprint(pin) {
		t.Errorf("the restored group still answers the old pin: %v", restored.CoverAssetIds)
	}
}

// TestCoverPinnedDeadCellFills pins the dead-cell rule: a pinned
// frame the caller can no longer see is replaced IN ITS POSITION from
// the visible pool (not by recency), the surviving picks keep their
// order, the detail still names the stored pick (so the picker can
// mark it), and a fully dead pin falls back to the automatic group.
func TestCoverPinnedDeadCellFills(t *testing.T) {
	instant := time.Date(2026, 11, 11, 10, 0, 0, 0, time.UTC)
	env := newAssetEnvWithClock(t, testUploadLimit, func() time.Time { return instant })
	adminID := env.sessionUserID(t)
	album := fixtureWallAlbumID("a4")
	members := insertCoverAlbum(t, env, album, adminID, "memory", 12)

	pin := []uuid.UUID{members[11], members[3], members[7], members[1], members[9], members[5]}
	env.patchAlbum(t, env.cookie, album, coverPinBody(pin...))

	// The third pick leaves the album (the owner tidies it out).
	env.exec(t, "DELETE FROM album_items WHERE album_id = $1 AND asset_id = $2",
		pgUUID(album), pgUUID(pin[2]))

	card := wallCardOf(t, env, album)
	if len(card.CoverAssetIds) != 6 {
		t.Fatalf("card frames after the dead pick = %d, want 6 (the cell filled)", len(card.CoverAssetIds))
	}
	for index, want := range pin {
		if index == 2 {
			continue
		}
		if uuid.UUID(card.CoverAssetIds[index]) != want {
			t.Errorf("card frame %d = %s, want the surviving pick %s (its position kept)",
				index, card.CoverAssetIds[index], want)
		}
	}
	if uuid.UUID(card.CoverAssetIds[2]) == pin[2] {
		t.Errorf("card frame 2 still names the removed pick %s", pin[2])
	}
	if !idSet(members...)[uuid.UUID(card.CoverAssetIds[2])] {
		t.Errorf("the fill frame %s is not a live member of the album", card.CoverAssetIds[2])
	}

	// The detail keeps the stored pin, dead frame included.
	detail := env.getAlbumDetail(t, env.cookie, album, "")
	if len(detail.CoverAssetIds) != 6 {
		t.Fatalf("detail frames = %v, want the stored six", detail.CoverAssetIds)
	}
	if uuid.UUID(detail.CoverAssetIds[2]) != pin[2] {
		t.Errorf("detail frame 2 = %s, want the stored pick %s (the picker marks it)",
			detail.CoverAssetIds[2], pin[2])
	}

	// Every pinned member leaves: the dead pin falls back to the
	// automatic group (the card never shows an empty cover while the
	// album holds live members).
	for _, id := range pin {
		env.exec(t, "DELETE FROM album_items WHERE album_id = $1 AND asset_id = $2",
			pgUUID(album), pgUUID(id))
	}
	fallback := wallCardOf(t, env, album)
	if len(fallback.CoverAssetIds) != 6 {
		t.Fatalf("fallback frames = %v, want the automatic six", fallback.CoverAssetIds)
	}
	dead := idSet(pin...)
	for _, raw := range fallback.CoverAssetIds {
		if dead[uuid.UUID(raw)] {
			t.Errorf("the fallback group names a dead pick: %s", raw)
		}
	}
}

// TestCoverSampleIsNotRecency pins the sampling's shape: the sample
// of a pool larger than the frame count is not the newest frames, the
// same seed answers the same order twice, and another week's seed
// answers another set.
func TestCoverSampleIsNotRecency(t *testing.T) {
	albumID := fixtureWallAlbumID("a5")
	pool := make([]uuid.UUID, 0, 40)
	for index := 0; index < 40; index++ {
		pool = append(pool, coverFixtureAssetID(index))
	}
	first := coverSample(weekSeed(albumID, "2026-W46"), pool, 6)
	if len(first) != 6 {
		t.Fatalf("sample = %v, want 6 ids", first)
	}
	recent := make(map[uuid.UUID]bool, 6)
	for _, id := range pool[:6] {
		recent[id] = true
	}
	matching := 0
	for _, id := range first {
		if recent[id] {
			matching++
		}
	}
	if matching == 6 {
		t.Errorf("the sample is exactly the newest six frames: %v", first)
	}
	if again := coverSample(weekSeed(albumID, "2026-W46"), pool, 6); fmt.Sprint(again) != fmt.Sprint(first) {
		t.Errorf("the same seed answered another order: %v then %v", first, again)
	}
	if other := coverSample(weekSeed(albumID, "2026-W47"), pool, 6); fmt.Sprint(other) == fmt.Sprint(first) {
		t.Errorf("another week answered the same group: %v", other)
	}
}

// coverPinBody renders the cover write's request from a pick order.
func coverPinBody(ids ...uuid.UUID) string {
	body := `{"coverAssetIds":[`
	for index, id := range ids {
		if index > 0 {
			body += ","
		}
		body += `"` + id.String() + `"`
	}
	return body + `]}`
}
