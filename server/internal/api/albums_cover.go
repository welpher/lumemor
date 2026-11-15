// The cover's sampling and resolution (the cover-picker block). A
// cover is a SET of frames -- four on the album card's 2x2 collage,
// six on the memory band's 3-across 2-down strip -- and it has two
// sources with one precedence: the owner's pinned picks win (their
// order is the render order), and an album without pins renders the
// automatic group the SERVER samples.
//
// The automatic group is a SAMPLE of the whole collection, never the
// newest N: the card is the first thing drawn and it cannot see the
// collection, so "otherwise the newest" would have made every album's
// cover the last event it holds. The sampling hashes (album, ISO
// week) over the live member ids, so the same week answers the same
// group on every device and under every member, and the weekly row is
// frozen the first time it is computed -- a mid-week upload cannot
// move a frame (the freeze), while the next ISO week re-samples
// whole. An album smaller than twice the frame count never rotates
// (there is nothing to rotate through); it keeps one stable sample
// seeded by the album alone.
//
// Dead cells fill silently from the caller's visible pool: a pinned
// frame that left the album, was trashed or hidden is replaced in its
// position, and a fully dead pin falls back to the automatic group
// (the pin itself stays stored -- a restored photo returns to its
// cell).
package api

import (
	"context"
	"fmt"
	"hash/fnv"
	"net/http"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	apigen "github.com/welpher/lumemor/server/internal/api/gen"
	dbgen "github.com/welpher/lumemor/server/internal/db/gen"
)

const (
	// albumCoverFrames is the frame count of the album (and place)
	// card's 4:3 collage: a 2x2 of 4:3 cells.
	albumCoverFrames = 4

	// memoryCoverFrames is the frame count of the memory card's 3:1
	// band: 3 columns by 2 rows of 2:1 cells.
	memoryCoverFrames = 6
)

// coverFramesFor answers the frame count of one album type: the
// memory band holds six, every other card four. A sync album never
// renders a pin (its card is the default album's face), but it
// samples like the album card it borrows.
func coverFramesFor(albumType dbgen.AlbumType) int {
	if albumType == dbgen.AlbumTypeMemory {
		return memoryCoverFrames
	}
	return albumCoverFrames
}

// isoWeekOf renders the ISO-8601 week identifier ("2026-W46") of one
// instant in the instant's own location: the week boundary is the
// instance timezone's Monday 00:00, not a rolling 7x24h window (a
// daylight-saving shift would slide a fixed-width window off the
// calendar week). ISOWeek is the calendar's own arithmetic -- week 1
// is the week holding the first Thursday, a year may hold 53 weeks --
// and the padded form sorts and compares as text.
func isoWeekOf(at time.Time) string {
	year, week := at.ISOWeek()
	return fmt.Sprintf("%04d-W%02d", year, week)
}

// coverRank scores one asset for one sampling seed: FNV-1a over the
// seed and the asset's bytes. The hash is deterministic across
// processes and restarts (the freeze and the cross-device agreement
// rest on that), which is why it replaces an ORDER BY random(): a
// database-side random breaks pagination and cannot be replayed for
// the same week. The asset id is the tiebreak, so two ids never
// compare equal and a pool's order never decides a frame.
func coverRank(seed string, assetID uuid.UUID) [16]byte {
	hasher := fnv.New128a()
	_, _ = hasher.Write([]byte(seed))
	_, _ = hasher.Write(assetID[:])
	var sum [16]byte
	copy(sum[:], hasher.Sum(nil))
	return sum
}

// coverSample ranks the pool under one seed and answers up to count
// ids: the deterministic "which frames does this seed pick" the
// rotation and the stable small-collection group share.
func coverSample(seed string, pool []uuid.UUID, count int) []uuid.UUID {
	ranked := make([]uuid.UUID, len(pool))
	copy(ranked, pool)
	sort.SliceStable(ranked, func(left, right int) bool {
		leftRank := coverRank(seed, ranked[left])
		rightRank := coverRank(seed, ranked[right])
		if comparison := compareRanks(leftRank, rightRank); comparison != 0 {
			return comparison < 0
		}
		return ranked[left].String() < ranked[right].String()
	})
	if len(ranked) > count {
		ranked = ranked[:count]
	}
	return ranked
}

// compareRanks orders two 128-bit ranks as unsigned big-endian
// numbers: byte-wise comparison is that order.
func compareRanks(left, right [16]byte) int {
	for index := range left {
		if left[index] != right[index] {
			if left[index] < right[index] {
				return -1
			}
			return 1
		}
	}
	return 0
}

// coverSlots assembles one cover from the frames already chosen (a
// pinned pick or a frozen week's cell, keyed by position) and the
// pool's remaining candidates: the surviving frames keep their
// positions, the holes fill in position order from the seed's
// sampling, and a pool too small to fill every hole answers the short
// group it can (a collection under the frame count has fewer frames
// than the card).
func coverSlots(seed string, slots []uuid.UUID, pool []uuid.UUID) []uuid.UUID {
	used := make(map[uuid.UUID]bool, len(slots))
	remaining := make([]uuid.UUID, 0, len(pool))
	for _, id := range slots {
		if id != uuid.Nil {
			used[id] = true
		}
	}
	for _, id := range pool {
		if !used[id] {
			remaining = append(remaining, id)
		}
	}
	fillers := coverSample(seed, remaining, len(slots))
	next := 0
	group := make([]uuid.UUID, 0, len(slots))
	for _, id := range slots {
		if id == uuid.Nil {
			if next >= len(fillers) {
				break
			}
			id = fillers[next]
			next++
		}
		group = append(group, id)
	}
	return group
}

// coverType is one album's cover input: the frame count derives from
// the type, the live member count decides whether the album rotates
// at all, and the ids key the answer.
type coverType struct {
	ID        uuid.UUID
	Type      dbgen.AlbumType
	ItemCount int64
	OwnerID   uuid.UUID
}

// coverGroups resolves the rendered cover of every given album: the
// pinned set (filled where a frame died), or the automatic group --
// the frozen weekly sample for an album holding at least twice the
// frame count, the stable album-seeded sample below that, and the
// empty group for an album with no live members. The answer maps an
// album id to its ordered frame ids; the callers write those ids into
// the wall card exactly as answered (the render order is the cover's
// own).
func (s *server) coverGroups(ctx context.Context, callerID uuid.UUID, albums []coverType) (map[uuid.UUID][]uuid.UUID, error) {
	groups := make(map[uuid.UUID][]uuid.UUID, len(albums))
	ids := make([]uuid.UUID, 0, len(albums))
	byID := make(map[uuid.UUID]coverType, len(albums))
	for _, album := range albums {
		byID[album.ID] = album
		if album.ItemCount > 0 {
			ids = append(ids, album.ID)
		} else {
			groups[album.ID] = []uuid.UUID{}
		}
	}
	if len(ids) == 0 {
		return groups, nil
	}

	week := isoWeekOf(s.now())

	// The owner flag per album: the week store is the owner's truth
	// alone -- a member's read never rewrites the group the owner
	// froze (a member's narrower pool would otherwise re-fill a dead
	// cell with a frame the owner cannot see).
	isOwner := make(map[uuid.UUID]bool, len(ids))
	for _, id := range ids {
		isOwner[id] = byID[id].OwnerID == callerID
	}

	// The live pinned cells and the frozen week's live cells: two
	// bounded reads whose rows already carry the liveness verdict, so
	// the steady state (a live pin, a full week row) never loads an
	// album's whole pool.
	pinnedRows, err := s.queries.ListAlbumLiveCoverPicks(ctx, dbgen.ListAlbumLiveCoverPicksParams{
		AlbumIds: pgUUIDs(ids),
		UserID:   pgUUID(callerID),
	})
	if err != nil {
		return nil, fmt.Errorf("load cover picks: %w", err)
	}
	weekRows, err := s.queries.ListAlbumLiveCoverWeekPicks(ctx, dbgen.ListAlbumLiveCoverWeekPicksParams{
		AlbumIds: pgUUIDs(ids),
		IsoWeek:  week,
		UserID:   pgUUID(callerID),
	})
	if err != nil {
		return nil, fmt.Errorf("load cover week picks: %w", err)
	}

	type slotRows map[int16]uuid.UUID
	pins := make(map[uuid.UUID]slotRows, len(ids))
	for _, row := range pinnedRows {
		albumID := uuid.UUID(row.AlbumID.Bytes)
		if pins[albumID] == nil {
			pins[albumID] = slotRows{}
		}
		pins[albumID][row.Position] = uuid.UUID(row.AssetID.Bytes)
	}
	weeks := make(map[uuid.UUID]slotRows, len(ids))
	for _, row := range weekRows {
		albumID := uuid.UUID(row.AlbumID.Bytes)
		if weeks[albumID] == nil {
			weeks[albumID] = slotRows{}
		}
		weeks[albumID][row.Position] = uuid.UUID(row.AssetID.Bytes)
	}

	// Which albums still need their pool: a partial pin (a dead cell
	// to fill), a rotating album (a missing or short week row), and
	// the small collection (no rotation, a fresh stable sample).
	//
	// A rotating album whose week row is complete needs nothing more
	// -- the freeze's whole point.
	poolNeeded := make([]uuid.UUID, 0, len(ids))
	for _, id := range ids {
		album := byID[id]
		frames := coverFramesFor(album.Type)
		if len(pins[id]) >= frames {
			continue
		}
		if len(pins[id]) > 0 {
			poolNeeded = append(poolNeeded, id)
			continue
		}
		if album.ItemCount < int64(2*frames) {
			poolNeeded = append(poolNeeded, id)
			continue
		}
		if len(weeks[id]) < frames {
			poolNeeded = append(poolNeeded, id)
		}
	}
	pools := make(map[uuid.UUID][]uuid.UUID, len(poolNeeded))
	if len(poolNeeded) > 0 {
		poolRows, err := s.queries.ListAlbumLiveCoverPool(ctx, dbgen.ListAlbumLiveCoverPoolParams{
			AlbumIds: pgUUIDs(poolNeeded),
			UserID:   pgUUID(callerID),
		})
		if err != nil {
			return nil, fmt.Errorf("load cover pool: %w", err)
		}
		for _, row := range poolRows {
			albumID := uuid.UUID(row.AlbumID.Bytes)
			pools[albumID] = append(pools[albumID], uuid.UUID(row.AssetID.Bytes))
		}
	}

	// The weekly rows to rewrite: a missing or short one is sampled
	// now and stored, so the week's group is frozen from its first
	// resolution on. A dead cell rewrites the row with the same live
	// frames in their positions plus the fill -- the fill is the
	// group's new truth, and the rest of the week's uploads cannot
	// move it either.
	type weekWrite struct {
		isoWeek string
		ids     []uuid.UUID
	}
	weekWrites := make(map[uuid.UUID]weekWrite)

	for _, id := range ids {
		album := byID[id]
		frames := coverFramesFor(album.Type)

		if len(pins[id]) > 0 {
			// The pinned set: the live picks keep their positions and
			// the holes fill from the album's own stable sample (the
			// pin's seed never rotates -- a filled cell must not move
			// when the calendar turns). A fully dead pin never reaches
			// here: no live pick means the automatic group answers.
			slots := make([]uuid.UUID, frames)
			for position, assetID := range pins[id] {
				if position < int16(frames) {
					slots[position] = assetID
				}
			}
			groups[id] = coverSlots(pinSeed(id), slots, pools[id])
			continue
		}

		if len(weeks[id]) >= frames {
			// A full stored week answers BEFORE any size test: the
			// freeze belongs to every album, and an album that grew
			// across the rotation floor mid-week keeps the group it
			// froze (the crossing picks up the rotation NEXT week).
			slots := make([]uuid.UUID, frames)
			for position, assetID := range weeks[id] {
				if position < int16(frames) {
					slots[position] = assetID
				}
			}
			groups[id] = coverSlots(weekSeed(id, week), slots, pools[id])
			continue
		}

		if album.ItemCount < int64(2*frames) {
			// Below twice the frame count there is nothing to rotate
			// through -- but the freeze still applies: the group is
			// stored on first resolution under the week key (the seed
			// stays album-only, so the stored group is identical week
			// to week while a mid-week upload cannot move it). A
			// stored row's survivors keep their positions and the
			// dead cells fill -- the fill becomes the stored truth.
			var group []uuid.UUID
			if len(weeks[id]) > 0 {
				slots := make([]uuid.UUID, frames)
				for position, assetID := range weeks[id] {
					if position < int16(frames) {
						slots[position] = assetID
					}
				}
				group = coverSlots(pinSeed(id), slots, pools[id])
			} else {
				group = coverSample(pinSeed(id), pools[id], frames)
			}
			groups[id] = group
			if isOwner[id] {
				weekWrites[id] = weekWrite{isoWeek: week, ids: group}
			}
			continue
		}

		slots := make([]uuid.UUID, frames)
		for position, assetID := range weeks[id] {
			if position < int16(frames) {
				slots[position] = assetID
			}
		}
		if len(weeks[id]) != frames && isOwner[id] {
			weekWrites[id] = weekWrite{isoWeek: week}
		}
		group := coverSlots(weekSeed(id, week), slots, pools[id])
		groups[id] = group
		if len(weeks[id]) != frames && isOwner[id] {
			entry := weekWrites[id]
			entry.ids = group
			weekWrites[id] = entry
		}
	}

	// Persisting the week's sampling is best-effort on the read path:
	// a write that loses to a concurrent resolution costs the next
	// read one resample, never the answer (both computed the same
	// deterministic group). A SHORT group is never stored -- it would
	// be re-derived on every read anyway (the row could never satisfy
	// the freeze's full-set check), so the write would only churn.
	for id, write := range weekWrites {
		if len(write.ids) != coverFramesFor(byID[id].Type) {
			continue
		}
		if err := s.queries.DeleteAlbumCoverWeek(ctx, dbgen.DeleteAlbumCoverWeekParams{
			AlbumID: pgUUID(id),
			IsoWeek: write.isoWeek,
		}); err != nil {
			s.logger.Warn("cover week reset failed", "error", err)
			continue
		}
		if err := s.queries.InsertAlbumCoverWeekPicks(ctx, dbgen.InsertAlbumCoverWeekPicksParams{
			AlbumID:  pgUUID(id),
			IsoWeek:  write.isoWeek,
			AssetIds: pgUUIDs(write.ids),
		}); err != nil {
			s.logger.Warn("cover week store failed", "error", err)
		}
	}

	return groups, nil
}

// writeCoverNotMember emits the cover write's refusal: the request
// named frames that are not live members of the album any more. The
// message key is the client's translation anchor and the count says
// how many frames died, so the picker can point at them instead of
// asking the owner to re-pick the whole set blind.
func (s *server) writeCoverNotMember(w http.ResponseWriter, count int) {
	details := map[string]interface{}{"key": "album_cover_not_member", "count": count}
	s.writeError(w, http.StatusBadRequest, apigen.VALIDATIONFAILED,
		"the cover must name live members of this album", &details)
}

// pinSeed is the stable sampling seed of one album's pinned fills and
// its small-collection group: the album alone, no week -- the group
// must not move when the calendar turns.
func pinSeed(albumID uuid.UUID) string {
	return "cover-pin|" + albumID.String()
}

// weekSeed is the rotating sampling seed: the album and the ISO week.
func weekSeed(albumID uuid.UUID, isoWeek string) string {
	return "cover-week|" + isoWeek + "|" + albumID.String()
}

// coverStoredPicks reads one album's stored pin, in pick order: the
// owner's set exactly as written, dead ids included. The detail read
// answers it so the picker can seed the current pin and name the
// frames that are no longer in the album; the card never renders this
// list raw (it renders coverGroups' resolution).
func (s *server) coverStoredPicks(ctx context.Context, albumID uuid.UUID) ([]uuid.UUID, error) {
	rows, err := s.queries.ListAlbumCoverPicks(ctx, []pgtype.UUID{pgUUID(albumID)})
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return []uuid.UUID{}, nil
	}
	slots := make(map[int16]uuid.UUID, len(rows))
	highest := int16(-1)
	for _, row := range rows {
		slots[row.Position] = uuid.UUID(row.AssetID.Bytes)
		if row.Position > highest {
			highest = row.Position
		}
	}
	picks := make([]uuid.UUID, 0, len(rows))
	for position := int16(0); position <= highest; position++ {
		if id, ok := slots[position]; ok {
			picks = append(picks, id)
		}
	}
	return picks, nil
}
