// The favorites and hidden views: the two curated
// timelines behind GET /assets/favorites and GET /assets/hidden.
// Both answer TimelineItem pages through the timeline's own
// machinery -- the same page-parameter validation, the same keyset
// cursor, the same page rendering -- so the wall UI that walks the
// timeline walks these views unchanged. They differ only in which
// rows the queries select: the caller's favorites rows joined to
// live assets, or the caller's own live rows whose visibility switch
// is off.
package api

import (
	"net/http"

	apigen "github.com/welpher/lumemor/server/internal/api/gen"
	dbgen "github.com/welpher/lumemor/server/internal/db/gen"
)

// GetFavorites implements GET /api/v1/assets/favorites: one keyset
// page of the caller's favorites, live assets only, every item
// isFavorite true by construction.
func (s *server) GetFavorites(w http.ResponseWriter, r *http.Request, params apigen.GetFavoritesParams) {
	requester, ok := authFrom(r.Context())
	if !ok {
		s.rejectUnauthenticated(w)
		return
	}

	limit, cursor, ok := s.timelinePageParams(w, params.Limit, params.Cursor)
	if !ok {
		return
	}

	// The look-ahead row decides nextCursor, exactly as the timeline.
	rows, err := s.queries.ListFavoriteAssets(r.Context(), dbgen.ListFavoriteAssetsParams{
		UserID:        pgUUID(requester.userID),
		CursorTakenAt: cursor.takenAt,
		CursorID:      cursor.id,
		LimitCount:    int32(limit) + 1,
	})
	if err != nil {
		s.internalError(w, r, "load favorites", err)
		return
	}

	s.writeJSON(w, http.StatusOK, timelinePage(rows, limit, favoriteAssetFields))
}

// GetHidden implements GET /api/v1/assets/hidden: one keyset page of
// the caller's hidden assets -- the owner-level visibility switch
// turned off, live rows only, no Live Photo component rows.
func (s *server) GetHidden(w http.ResponseWriter, r *http.Request, params apigen.GetHiddenParams) {
	requester, ok := authFrom(r.Context())
	if !ok {
		s.rejectUnauthenticated(w)
		return
	}

	limit, cursor, ok := s.timelinePageParams(w, params.Limit, params.Cursor)
	if !ok {
		return
	}

	rows, err := s.queries.ListHiddenAssets(r.Context(), dbgen.ListHiddenAssetsParams{
		UserID:        pgUUID(requester.userID),
		CursorTakenAt: cursor.takenAt,
		CursorID:      cursor.id,
		LimitCount:    int32(limit) + 1,
	})
	if err != nil {
		s.internalError(w, r, "load hidden assets", err)
		return
	}

	s.writeJSON(w, http.StatusOK, timelinePage(rows, limit, hiddenAssetFields))
}
