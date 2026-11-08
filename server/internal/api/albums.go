// The album surface reads (M3c Task 1, the M3d default-album pin):
// the wall behind GET /albums (the caller's pinned sync card first,
// then the own group, then the shared group -- one concatenated
// array the web client sections off the callerRole discriminator)
// and one album's detail behind GET /albums/{albumId} (metadata
// plus one keyset page of the live items). The caller's OWN sync
// album surfaces through BOTH faces, read-only: it is the default
// "My photos" album every upload lands in, so a fresh account's
// wall is never empty -- the stored title is the system's internal
// label and the client renders the localized display name keyed by
// the type. A sync album of another account stays invisible on
// every face, and the detail's 404 is single-shape: unknown,
// invisible and foreign-sync ids answer the same byte-identical
// envelope, the anti-enumeration discipline of ASSET_NOT_FOUND on
// the album face. The WRITE gates (albums_write.go,
// albumaudience.go, albumprefs.go) keep refusing every sync album,
// whoever owns it: the read admitting the caller's own sync opens
// no write path.
package api

import (
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	oapitypes "github.com/oapi-codegen/runtime/types"

	apigen "github.com/welpher/lumemor/server/internal/api/gen"
	dbgen "github.com/welpher/lumemor/server/internal/db/gen"
)

const (
	// albumDefaultLimit is the page size of a detail request without
	// a limit parameter, pinned by the contract.
	albumDefaultLimit = 60

	// albumMaxLimit caps the page size: unlike the timeline, a value
	// over the cap is a client bug the 400 names, not a clamp -- the
	// wall's grid pages at 60 and the cap only bounds honest bulk
	// walkers (the people surface's discipline, pinned by the
	// contract's 400 on limit).
	albumMaxLimit = 200
)

// albumPageParams validates and normalizes the detail's pagination
// parameters: the limit -- absent means the contract default, below 1
// or over the maximum is a client bug the 400 names -- and the
// cursor -- absent is the first page, anything else must be exactly a
// string the timeline cursor codec produced, because the ordering key
// is the timeline's own (taken_at, id) pair: the album items page
// with the timeline's exact keyset semantics. It answers ok=false
// once it has written the 400 envelope.
func (s *server) albumPageParams(w http.ResponseWriter, limit *int, cursor *string) (int, timelineCursor, bool) {
	pageLimit := albumDefaultLimit
	if limit != nil {
		if *limit < 1 {
			s.writeValidationError(w, "limit must be at least 1", []string{"limit"})
			return 0, timelineCursor{}, false
		}
		if *limit > albumMaxLimit {
			s.writeValidationError(w, "limit must be at most 200", []string{"limit"})
			return 0, timelineCursor{}, false
		}
		pageLimit = *limit
	}

	var pageCursor timelineCursor
	if cursor != nil && *cursor != "" {
		decoded, err := decodeTimelineCursor(*cursor)
		if err != nil {
			s.writeValidationError(w, "invalid album cursor", []string{"cursor"})
			return 0, timelineCursor{}, false
		}
		pageCursor = decoded
	}
	return pageLimit, pageCursor, true
}

// writeAlbumNotFound emits the album surface's single-shape 404: the
// same envelope every read and write answers for an album the caller
// cannot see -- unknown, foreign-unshared, revoked, another
// account's sync (and every sync id through the write faces,
// whoever owns it) -- so the faces stay byte-identical and album ids
// cannot be probed for existence.
func (s *server) writeAlbumNotFound(w http.ResponseWriter) {
	s.writeError(w, http.StatusNotFound, apigen.ALBUMNOTFOUND,
		"no such album for this caller", nil)
}

// albumSummaryCard maps one album row plus its honest companions
// onto the wall card shape: the collage source (the pinned cover or
// the derived candidates), the live member count, the share badge,
// the caller's role and the caller's timeline preference. The wall's
// own mapping and the write face's responses share this one
// authority, so a write's answer is the card the wall lists.
func albumSummaryCard(album dbgen.Album, isShared bool, itemCount int64, role apigen.AlbumSummaryCallerRole, derived []uuid.UUID, inTimeline *bool) apigen.AlbumSummary {
	return apigen.AlbumSummary{
		Id:            uuid.UUID(album.ID.Bytes),
		Title:         album.Title,
		Type:          apigen.AlbumSummaryType(album.Type),
		Locked:        album.Locked,
		IsShared:      isShared,
		CoverAssetIds: coverAssetIds(album.CoverAssetID, derived),
		ItemCount:     int(itemCount),
		UpdatedAt:     album.UpdatedAt.Time.UTC(),
		CallerRole:    role,
		InTimeline:    inTimeline,
	}
}

// timelinePrefField maps one album's raw preference row onto the
// card's field: the excludable cards (the caller's own directory
// albums and every shared album) carry the switch -- the stored
// value, or true when no row exists (the default of feeding the
// timeline) -- while the cards that never feed an excludable
// timeline omit the field entirely: the caller's own virtual albums
// (in-stream curation would be pure duplication) and every sync
// card (the core stream is never excludable; the sync arm is
// owner-scoped, so the role is always owner when the type is sync).
// The role rides along because the write face's cards are always
// the owner's.
func timelinePrefField(albumType dbgen.AlbumType, role string, stored pgtype.Bool) *bool {
	if albumType == dbgen.AlbumTypeSync {
		return nil
	}
	if albumType == dbgen.AlbumTypeVirtual && role == "owner" {
		return nil
	}
	value := true
	if stored.Valid {
		value = stored.Bool
	}
	return &value
}

// assetTileThumbnailPath renders the tile-size serving URL of one
// asset: the existing asset thumbnail endpoint with the tile size,
// the same serving copy the timeline's grid requests -- the album
// detail renders the same grid class.
func assetTileThumbnailPath(assetID uuid.UUID) string {
	return "/api/v1/assets/" + assetID.String() + "/thumbnail?size=tile"
}

// coverAssetIds resolves one wall card's collage source: the pinned
// cover asset as the single-element list when the owner set one,
// otherwise the derived candidates (up to the four most recent live
// members). The answer is never nil -- an album without live members
// renders an empty collage, not a null.
func coverAssetIds(pinned pgtype.UUID, derived []uuid.UUID) []uuid.UUID {
	if pinned.Valid {
		return []uuid.UUID{uuid.UUID(pinned.Bytes)}
	}
	ids := make([]uuid.UUID, 0, len(derived))
	ids = append(ids, derived...)
	return ids
}

// ListAlbums implements GET /api/v1/albums: the caller's wall. The
// query answers all three groups in one statement -- the pinned
// sync card (the caller's own core stream, the default album)
// first, then the own group (the caller's virtual and directory
// albums, updated_at descending), then the shared group (the albums
// shared to the caller, the grant time descending) with the owner
// attribution joined in. The albums without a pinned cover derive
// their collage from their most recent live members in one bounded
// follow-up query (the sync card included: live members, same as
// any album).
func (s *server) ListAlbums(w http.ResponseWriter, r *http.Request) {
	requester, ok := authFrom(r.Context())
	if !ok {
		s.rejectUnauthenticated(w)
		return
	}

	rows, err := s.queries.ListWallAlbumsForCaller(r.Context(), pgUUID(requester.userID))
	if err != nil {
		s.internalError(w, r, "load album wall", err)
		return
	}

	// The derived covers: only the albums without a pinned cover need
	// candidates, and the query bounds itself at four rows per album.
	derived := make([]pgtype.UUID, 0, len(rows))
	for _, row := range rows {
		if !row.Album.CoverAssetID.Valid {
			derived = append(derived, row.Album.ID)
		}
	}
	covers := make(map[uuid.UUID][]uuid.UUID)
	if len(derived) > 0 {
		candidates, err := s.queries.ListAlbumCoverCandidates(r.Context(), derived)
		if err != nil {
			s.internalError(w, r, "load album covers", err)
			return
		}
		for _, candidate := range candidates {
			id := uuid.UUID(candidate.AlbumID.Bytes)
			covers[id] = append(covers[id], uuid.UUID(candidate.AssetID.Bytes))
		}
	}

	albums := make([]apigen.AlbumSummary, 0, len(rows))
	for _, row := range rows {
		albumID := uuid.UUID(row.Album.ID.Bytes)
		summary := albumSummaryCard(row.Album, row.IsShared, row.ItemCount,
			apigen.AlbumSummaryCallerRole(row.CallerRole), covers[albumID],
			timelinePrefField(row.Album.Type, row.CallerRole, row.InTimeline))
		// The owner attribution rides only on shared items: an own
		// card's owner is the caller, and the query leaves the pair
		// NULL for exactly those rows.
		if row.OwnerUserID.Valid {
			summary.Owner = &apigen.AlbumOwnerRef{
				Id:    uuid.UUID(row.OwnerUserID.Bytes),
				Email: oapitypes.Email(row.OwnerEmail.String),
			}
		}
		albums = append(albums, summary)
	}

	s.writeJSON(w, http.StatusOK, apigen.AlbumWallResponse{Albums: albums})
}

// GetAlbum implements GET /api/v1/albums/{albumId}: the album's
// metadata with the caller's role and one keyset page of its live
// items. The visibility gate is structural: GetAlbumForCaller
// answers no row for an unknown id, an album the caller neither
// owns nor holds a share of, and every sync album of ANOTHER
// account -- so all of them fold into the one byte-identical 404
// and album ids cannot be probed for existence. The caller's OWN
// sync album resolves: the default album's read-only detail (the
// handler answers it exactly like any album's -- the owner role,
// the live members keyset-paged -- while the write gates keep
// their own refusals of the same id).
func (s *server) GetAlbum(w http.ResponseWriter, r *http.Request, albumId uuid.UUID, params apigen.GetAlbumParams) {
	requester, ok := authFrom(r.Context())
	if !ok {
		s.rejectUnauthenticated(w)
		return
	}

	limit, cursor, ok := s.albumPageParams(w, params.Limit, params.Cursor)
	if !ok {
		return
	}

	album, err := s.queries.GetAlbumForCaller(r.Context(), dbgen.GetAlbumForCallerParams{
		UserID:  pgUUID(requester.userID),
		AlbumID: pgUUID(albumId),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		s.writeAlbumNotFound(w)
		return
	}
	if err != nil {
		s.internalError(w, r, "load album", err)
		return
	}

	// One row beyond the page is the exact "another page exists"
	// signal, the look-ahead pattern of the timeline: the response
	// carries nextCursor precisely when it observed the follow-up, so
	// no run ever walks into an empty trailing page. The query needs
	// no visibility re-check: it runs only for an album the gate
	// above already resolved for this caller.
	rows, err := s.queries.ListAlbumItemsForCaller(r.Context(), dbgen.ListAlbumItemsForCallerParams{
		AlbumID:       pgUUID(albumId),
		CursorTakenAt: cursor.takenAt,
		CursorID:      cursor.id,
		LimitCount:    int32(limit) + 1,
	})
	if err != nil {
		s.internalError(w, r, "load album items", err)
		return
	}

	detail := apigen.AlbumDetail{
		Id:           albumId,
		Title:        album.Album.Title,
		Type:         apigen.AlbumDetailType(album.Album.Type),
		Locked:       album.Album.Locked,
		CoverAssetId: nil,
		CallerRole:   apigen.AlbumDetailCallerRole(album.CallerRole),
		ItemCount:    int(album.ItemCount),
		Items:        make([]apigen.AlbumItem, 0, len(rows)),
		InTimeline:   timelinePrefField(album.Album.Type, album.CallerRole, album.InTimeline),
	}
	// The source line of the edit dialog: the directory album's
	// stored root, present exactly on the directory type (fixtures
	// may carry a NULL root; the row is the honest source of the
	// field).
	if album.Album.Type == dbgen.AlbumTypeDirectory && album.Album.RootPath.Valid {
		rootPath := album.Album.RootPath.String
		detail.RootPath = &rootPath
	}
	// The owner-only grantee list (the share management dialog's
	// rows): the field rides the owner's view and only that view --
	// a sharee sees the album, never its ACL. The non-nil pointer
	// keeps the field present in the owner's JSON even when the
	// album holds no grants (the empty array is the dialog's honest
	// "no one yet" state).
	if album.CallerRole == "owner" {
		entries, err := s.queries.ListAlbumShares(r.Context(), pgUUID(albumId))
		if err != nil {
			s.internalError(w, r, "load album shares", err)
			return
		}
		shares := make([]apigen.AlbumShareEntry, 0, len(entries))
		for _, entry := range entries {
			shares = append(shares, apigen.AlbumShareEntry{
				UserId:    uuid.UUID(entry.UserID.Bytes),
				Email:     oapitypes.Email(entry.Email),
				GrantedAt: entry.GrantedAt.Time.UTC(),
			})
		}
		detail.Shares = &shares
	}
	if album.Album.CoverAssetID.Valid {
		cover := uuid.UUID(album.Album.CoverAssetID.Bytes)
		detail.CoverAssetId = &cover
	}
	if len(rows) > limit {
		rows = rows[:limit]
		last := rows[limit-1]
		next := encodeTimelineCursor(last.TakenAt, last.AssetID)
		detail.NextCursor = &next
	}
	for _, row := range rows {
		assetID := uuid.UUID(row.AssetID.Bytes)
		detail.Items = append(detail.Items, apigen.AlbumItem{
			AssetId: assetID,
			AddedAt: row.AddedAt.Time.UTC(),
			Asset: apigen.AlbumItemAsset{
				TakenAt:          row.TakenAt.Time.UTC(),
				OriginalFilename: row.OriginalFilename,
				Kind:             apigen.AlbumItemAssetKind(row.MediaKind),
				Thumbnail:        assetTileThumbnailPath(assetID),
			},
		})
	}

	s.writeJSON(w, http.StatusOK, detail)
}
