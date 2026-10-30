// The timeline handler (M2c Task 2): the aggregated timeline of the
// caller (and its per-album narrowing), one keyset page at a time.
package api

import (
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	apigen "github.com/welpher/lumemor/server/internal/api/gen"
	dbgen "github.com/welpher/lumemor/server/internal/db/gen"
)

const (
	// timelineDefaultLimit is the page size of a request without a
	// limit parameter, pinned by the contract.
	timelineDefaultLimit = 100

	// timelineMaxLimit caps the page size however large the request
	// asks: one response stays a bounded JSON document.
	timelineMaxLimit = 200
)

// timelineCursor is the decoded continuation token: the (taken_at, id)
// pair of the last item of the previous page, the exact ordering key
// of the keyset predicate. The zero cursor (a zero time) is the first
// page: the handler passes the pair as NULL arguments instead, which
// short-circuits the predicate to TRUE.
type timelineCursor struct {
	takenAt pgtype.Timestamptz
	id      pgtype.UUID
}

// cursorWireBytes is the byte length of the wire form: 8 bytes of
// nanoseconds since the epoch plus the 16 bytes of the uuid.
const cursorWireBytes = 24

// encodeTimelineCursor renders the cursor of one page's last row: the
// opaque base64 of the binary pair. Clients treat it as an opaque
// string; the pair layout keeps it compact (32 characters) and stable
// under the page's own values.
func encodeTimelineCursor(takenAt pgtype.Timestamptz, id pgtype.UUID) string {
	var buf [cursorWireBytes]byte
	binary.BigEndian.PutUint64(buf[:8], uint64(takenAt.Time.UnixNano()))
	copy(buf[8:], id.Bytes[:])
	return base64.RawURLEncoding.EncodeToString(buf[:])
}

// decodeTimelineCursor parses a cursor this endpoint produced. The
// decoding fails on anything but the exact wire form -- a mangled or
// truncated value, or one of another length, is a client error, not a
// server one. A well-formed but fabricated cursor is harmless: it
// simply addresses some (taken_at, id) point in the ordering, which
// yields the items after that point or none.
func decodeTimelineCursor(value string) (timelineCursor, error) {
	raw, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return timelineCursor{}, err
	}
	if len(raw) != cursorWireBytes {
		return timelineCursor{}, fmt.Errorf("cursor holds %d bytes, want %d", len(raw), cursorWireBytes)
	}
	nanos := int64(binary.BigEndian.Uint64(raw[:8]))
	var id uuid.UUID
	copy(id[:], raw[8:])
	return timelineCursor{
		takenAt: pgtype.Timestamptz{Time: unixNanoTime(nanos), Valid: true},
		id:      pgtype.UUID{Bytes: id, Valid: true},
	}, nil
}

// GetTimeline implements GET /api/v1/assets/timeline: one keyset page
// of the caller's aggregated timeline, or -- with albumId -- of that
// album's own timeline.
func (s *server) GetTimeline(w http.ResponseWriter, r *http.Request, params apigen.GetTimelineParams) {
	requester, ok := authFrom(r.Context())
	if !ok {
		s.rejectUnauthenticated(w)
		return
	}

	// Page size: absent means the contract default, over the maximum
	// clamps to it, below 1 is a client bug the 400 names. The clamp
	// of the upper bound (rather than a 400) keeps honest clients
	// future-proof: a page size the instance currently considers
	// generous still answers.
	limit := timelineDefaultLimit
	if params.Limit != nil {
		if *params.Limit < 1 {
			s.writeValidationError(w, "limit must be at least 1", []string{"limit"})
			return
		}
		if *params.Limit > timelineMaxLimit {
			limit = timelineMaxLimit
		} else {
			limit = *params.Limit
		}
	}

	// The continuation point: a cursor is either absent (the first
	// page) or exactly the string this endpoint handed out. Anything
	// else the decode refuses with the 400 of the contract.
	var cursor timelineCursor
	if params.Cursor != nil && *params.Cursor != "" {
		decoded, err := decodeTimelineCursor(*params.Cursor)
		if err != nil {
			s.writeValidationError(w, "invalid timeline cursor", []string{"cursor"})
			return
		}
		cursor = decoded
	}

	// One row beyond the page is the exact "another page exists"
	// signal: the response carries nextCursor precisely when it
	// observed it, so the last page is the one without the follow-up
	// and no run ever walks into an empty trailing page.
	fetch := int32(limit) + 1

	var (
		rows  []dbgen.Asset
		query string
		err   error
	)
	if params.AlbumId != nil {
		query = "album timeline"
		rows, err = s.queries.ListAlbumTimelineAssets(r.Context(), dbgen.ListAlbumTimelineAssetsParams{
			AlbumID:       pgUUID(uuid.UUID(*params.AlbumId)),
			UserID:        pgUUID(requester.userID),
			CursorTakenAt: cursor.takenAt,
			CursorID:      cursor.id,
			LimitCount:    fetch,
		})
	} else {
		query = "timeline"
		rows, err = s.queries.ListTimelineAssets(r.Context(), dbgen.ListTimelineAssetsParams{
			CursorTakenAt: cursor.takenAt,
			CursorID:      cursor.id,
			UserID:        pgUUID(requester.userID),
			LimitCount:    fetch,
		})
	}
	// The queries are SELECTs over committed rows; a failure is an
	// infrastructure problem, not a client one. An invisible album is
	// NOT an error path: it resolves to no rows -- the empty list,
	// indistinguishable from an unknown id -- never to a 404.
	if err != nil {
		s.internalError(w, r, "load "+query, err)
		return
	}

	response := apigen.TimelineResponse{Items: make([]apigen.TimelineItem, 0, len(rows))}
	if len(rows) > limit {
		rows = rows[:limit]
		last := rows[limit-1]
		next := encodeTimelineCursor(last.TakenAt, last.ID)
		response.NextCursor = &next
	}
	for _, row := range rows {
		response.Items = append(response.Items, timelineItemFrom(row))
	}

	s.writeJSON(w, http.StatusOK, response)
}

// timelineItemFrom maps one timeline row onto the API shape. The row
// is live by construction (the query filters hidden, trashed and
// component rows), so isTrashed is always false here -- carried only
// to keep the shape identical to the asset detail's. The
// pipeline-dependent values cross as null while the jobs have not
// produced them, exactly like the detail's.
func timelineItemFrom(row dbgen.Asset) apigen.TimelineItem {
	return apigen.TimelineItem{
		Id:               uuid.UUID(row.ID.Bytes),
		OwnerId:          uuid.UUID(row.OwnerID.Bytes),
		Kind:             apigen.TimelineItemKind(row.MediaKind),
		OriginalFilename: row.OriginalFilename,
		FileSize:         row.FileSize,
		MimeType:         row.MimeType,
		Checksum:         row.Checksum,
		CreatedAt:        row.CreatedAt.Time.UTC(),
		TakenAt:          row.TakenAt.Time.UTC(),
		TakenAtSource:    apigen.TimelineItemTakenAtSource(row.TakenAtSource),
		IsTrashed:        row.DeletedAt.Valid,
		IsReady:          row.ReadyAt.Valid,
		Width:            dimensionValue(row.Width),
		Height:           dimensionValue(row.Height),
		DurationMs:       int4Value(row.DurationMs),
		Blurhash:         textValue(row.Blurhash),
		LivePhotoVideoId: uuidValue(row.LivePhotoVideoID),
	}
}

// unixNanoTime rebuilds a time.Time from cursor nanoseconds. Out-of-
// range values (a fabricated cursor) map to the time package's own
// saturation points, which simply address a point before or after
// every asset -- a harmless, empty page.
func unixNanoTime(nanos int64) time.Time {
	return time.Unix(0, nanos).UTC()
}
