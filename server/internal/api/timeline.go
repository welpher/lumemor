// The timeline handler: the aggregated timeline of the
// caller (and its per-album narrowing), one keyset page at a time.
// The page-parameter validation and the item mapping here are the
// shared machinery of every timeline-shaped view: the favorites and
// hidden views answer the same page shape through the same
// helpers.
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
// album's own timeline, or -- with clusterId -- of that person's
// photos (the detail's wall). The two narrows are mutually exclusive:
// a request carrying both names no single view, so it is a client
// bug the 400 reports.
func (s *server) GetTimeline(w http.ResponseWriter, r *http.Request, params apigen.GetTimelineParams) {
	requester, ok := authFrom(r.Context())
	if !ok {
		s.rejectUnauthenticated(w)
		return
	}

	if params.AlbumId != nil && params.ClusterId != nil {
		s.writeValidationError(w, "albumId and clusterId are mutually exclusive", []string{"clusterId", "albumId"})
		return
	}

	limit, cursor, ok := s.timelinePageParams(w, params.Limit, params.Cursor)
	if !ok {
		return
	}

	// One row beyond the page is the exact "another page exists"
	// signal: the response carries nextCursor precisely when it
	// observed it, so the last page is the one without the follow-up
	// and no run ever walks into an empty trailing page.
	fetch := int32(limit) + 1

	// The queries are SELECTs over committed rows; a failure is an
	// infrastructure problem, not a client one. An invisible album is
	// NOT an error path: it resolves to no rows -- the empty list,
	// indistinguishable from an unknown id -- never to a 404.
	var response apigen.TimelineResponse
	if params.ClusterId != nil {
		// The person's own photos: an invisible cluster resolves to
		// no rows, the same empty-list answer the album narrow gives
		// (never a 404 -- cluster ids cannot be probed here).
		rows, err := s.queries.ListPersonTimelineAssets(r.Context(), dbgen.ListPersonTimelineAssetsParams{
			ClusterID:     pgUUID(uuid.UUID(*params.ClusterId)),
			UserID:        pgUUID(requester.userID),
			CursorTakenAt: cursor.takenAt,
			CursorID:      cursor.id,
			LimitCount:    fetch,
		})
		if err != nil {
			s.internalError(w, r, "load person timeline", err)
			return
		}
		response = timelinePage(rows, limit, personAssetFields)
	} else if params.AlbumId != nil {
		rows, err := s.queries.ListAlbumTimelineAssets(r.Context(), dbgen.ListAlbumTimelineAssetsParams{
			AlbumID:       pgUUID(uuid.UUID(*params.AlbumId)),
			UserID:        pgUUID(requester.userID),
			CursorTakenAt: cursor.takenAt,
			CursorID:      cursor.id,
			LimitCount:    fetch,
		})
		if err != nil {
			s.internalError(w, r, "load album timeline", err)
			return
		}
		response = timelinePage(rows, limit, albumAssetFields)
	} else {
		rows, err := s.queries.ListTimelineAssets(r.Context(), dbgen.ListTimelineAssetsParams{
			CursorTakenAt: cursor.takenAt,
			CursorID:      cursor.id,
			UserID:        pgUUID(requester.userID),
			LimitCount:    fetch,
		})
		if err != nil {
			s.internalError(w, r, "load timeline", err)
			return
		}
		response = timelinePage(rows, limit, timelineAssetFields)
	}

	s.writeJSON(w, http.StatusOK, response)
}

// timelinePage renders one keyset page out of the rows a
// timeline-shaped query fetched (limit+1 of them when another page
// follows): it trims the look-ahead row, encodes the continuation
// cursor from the last kept row and maps the survivors onto the API
// shape. The generic row folds through fields, which adapts each
// query's generated row type -- they all carry the embedded asset row
// and the per-user is_favorite flag; the views differ in how they
// select rows, not in how a page renders.
func timelinePage[T any](rows []T, limit int, fields func(T) (dbgen.Asset, bool)) apigen.TimelineResponse {
	response := apigen.TimelineResponse{Items: make([]apigen.TimelineItem, 0, len(rows))}
	if len(rows) > limit {
		rows = rows[:limit]
		last, _ := fields(rows[limit-1])
		next := encodeTimelineCursor(last.TakenAt, last.ID)
		response.NextCursor = &next
	}
	for _, row := range rows {
		asset, isFavorite := fields(row)
		response.Items = append(response.Items, timelineItemFrom(asset, isFavorite))
	}
	return response
}

// The field extractors of the five timeline-shaped queries (the
// aggregate, the album, the person, the favorites view, the hidden
// view): each folds its generated row type onto the (asset,
// isFavorite) pair the shared page renderer consumes.
func timelineAssetFields(row dbgen.ListTimelineAssetsRow) (dbgen.Asset, bool) {
	return row.Asset, row.IsFavorite
}

func albumAssetFields(row dbgen.ListAlbumTimelineAssetsRow) (dbgen.Asset, bool) {
	return row.Asset, row.IsFavorite
}

func personAssetFields(row dbgen.ListPersonTimelineAssetsRow) (dbgen.Asset, bool) {
	return row.Asset, row.IsFavorite
}

func favoriteAssetFields(row dbgen.ListFavoriteAssetsRow) (dbgen.Asset, bool) {
	return row.Asset, row.IsFavorite
}

func hiddenAssetFields(row dbgen.ListHiddenAssetsRow) (dbgen.Asset, bool) {
	return row.Asset, row.IsFavorite
}

// timelinePageParams validates and normalizes the pagination
// parameters every timeline-shaped view shares (the aggregate, the
// favorites, the hidden): the limit -- absent means the contract
// default, over the maximum clamps to it, below 1 is a client bug the
// 400 names (the clamp of the upper bound rather than a 400 keeps
// honest clients future-proof: a page size the instance currently
// considers generous still answers) -- and the cursor -- absent is
// the first page, anything else must be exactly the string this
// endpoint family produced. It answers ok=false once it has written
// the 400 envelope.
func (s *server) timelinePageParams(w http.ResponseWriter, limit *int, cursor *string) (int, timelineCursor, bool) {
	pageLimit := timelineDefaultLimit
	if limit != nil {
		if *limit < 1 {
			s.writeValidationError(w, "limit must be at least 1", []string{"limit"})
			return 0, timelineCursor{}, false
		}
		if *limit > timelineMaxLimit {
			pageLimit = timelineMaxLimit
		} else {
			pageLimit = *limit
		}
	}

	var pageCursor timelineCursor
	if cursor != nil && *cursor != "" {
		decoded, err := decodeTimelineCursor(*cursor)
		if err != nil {
			s.writeValidationError(w, "invalid timeline cursor", []string{"cursor"})
			return 0, timelineCursor{}, false
		}
		pageCursor = decoded
	}
	return pageLimit, pageCursor, true
}

// timelineItemFrom maps one timeline row onto the API shape. The row
// is live by construction (every timeline-shaped query filters
// hidden, trashed and component rows), so isTrashed is always false
// here -- carried only to keep the shape identical to the asset
// detail's. isFavorite is the per-user state the query computed for
// the requesting user. Width and height cross as the raw NOT NULL
// column values (the justified wall's contract: always present, the
// 0 placeholder while the metadata job has not refined them); the
// other pipeline-dependent values cross as null, exactly like the
// detail's.
func timelineItemFrom(row dbgen.Asset, isFavorite bool) apigen.TimelineItem {
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
		IsFavorite:       isFavorite,
		IsReady:          row.ReadyAt.Valid,
		Width:            int(row.Width),
		Height:           int(row.Height),
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
