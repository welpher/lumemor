// The facet collection: the filter bar's honest counts.
// Every facet counts the CURRENT filtered set MINUS its own
// dimension -- "how many results would I have if I added this value"
// -- over the same caller-scoped domain the result page walks (the
// text-match domain in the query mode, the plain scope in the filter
// browse). The orderings are total orders (count descending with a
// per-dimension deterministic tiebreak), so the facets are as stable
// as the pages.
package search

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// domain renders a facet query's FROM body: the query mode joins the
// text-match hits onto the assets, the filter browse walks the assets
// directly (the scope's live-row and visibility clauses then do the
// same narrowing on both shapes).
func domain(textMode bool) string {
	if textMode {
		return "search_text_hits h JOIN assets a ON a.id = h.asset_id"
	}
	return "assets a"
}

// collectFacets runs the seven facet queries.
func collectFacets(ctx context.Context, tx pgx.Tx, caller uuid.UUID, filters Filters, locale string, textMode bool) (Facets, error) {
	var facets Facets
	var err error
	if facets.People, err = facetPeople(ctx, tx, caller, filters, textMode); err != nil {
		return Facets{}, err
	}
	if facets.Places, err = facetPlaces(ctx, tx, caller, filters, locale, textMode); err != nil {
		return Facets{}, err
	}
	if facets.Years, err = facetYears(ctx, tx, caller, filters, textMode); err != nil {
		return Facets{}, err
	}
	if facets.Cameras, err = facetCameras(ctx, tx, caller, filters, textMode); err != nil {
		return Facets{}, err
	}
	if facets.Kinds, err = facetKinds(ctx, tx, caller, filters, textMode); err != nil {
		return Facets{}, err
	}
	if facets.Albums, err = facetAlbums(ctx, tx, caller, filters, textMode); err != nil {
		return Facets{}, err
	}
	if facets.Tags, err = facetTags(ctx, tx, caller, filters, textMode); err != nil {
		return Facets{}, err
	}
	return facets, nil
}

// facetPeople counts the caller's named, non-hidden people: one
// entry per labeled face cluster with a face on the scoped domain.
func facetPeople(ctx context.Context, tx pgx.Tx, caller uuid.UUID, filters Filters, textMode bool) ([]PersonFacet, error) {
	scope := newScope(caller)
	scope.addFilters(filters, dimensionPeople)
	rows, err := tx.Query(ctx, fmt.Sprintf(`
		SELECT l.cluster_id, l.name, count(*)::int AS asset_count
		FROM %s
		JOIN faces f ON f.asset_id = a.id
		JOIN user_face_labels l ON l.cluster_id = f.cluster_id AND l.user_id = $%d
		WHERE %s AND l.is_hidden = false
		GROUP BY l.cluster_id, l.name
		ORDER BY asset_count DESC, lower(l.name), l.cluster_id`,
		domain(textMode), scope.callerArg, scope.where()), scope.args()...)
	if err != nil {
		return nil, fmt.Errorf("search: collect the people facet: %w", err)
	}
	defer rows.Close()
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (PersonFacet, error) {
		var facet PersonFacet
		err := row.Scan(&facet.ID, &facet.Name, &facet.AssetCount)
		return facet, err
	})
}

// facetPlaces counts the resolved places of the scoped domain with
// the locale display name.
func facetPlaces(ctx context.Context, tx pgx.Tx, caller uuid.UUID, filters Filters, locale string, textMode bool) ([]PlaceFacet, error) {
	scope := newScope(caller)
	scope.addFilters(filters, dimensionPlaces)
	scope.values = append(scope.values, locale)
	localeArg := len(scope.values)
	rows, err := tx.Query(ctx, fmt.Sprintf(`
		SELECT a.place_id, COALESCE(pn.name, p.name) AS name, p.country_code,
		       count(*)::int AS asset_count
		FROM %s
		JOIN places p ON p.id = a.place_id
		LEFT JOIN place_names pn ON pn.geoname_id = p.id AND pn.locale = $%d
		WHERE %s
		GROUP BY a.place_id, p.name, pn.name, p.country_code
		ORDER BY asset_count DESC, a.place_id ASC`,
		domain(textMode), localeArg, scope.where()), scope.args()...)
	if err != nil {
		return nil, fmt.Errorf("search: collect the places facet: %w", err)
	}
	defer rows.Close()
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (PlaceFacet, error) {
		var facet PlaceFacet
		err := row.Scan(&facet.PlaceID, &facet.Name, &facet.Country, &facet.AssetCount)
		return facet, err
	})
}

// facetYears counts the taken-at years of the scoped domain, newest
// first.
func facetYears(ctx context.Context, tx pgx.Tx, caller uuid.UUID, filters Filters, textMode bool) ([]YearFacet, error) {
	scope := newScope(caller)
	scope.addFilters(filters, dimensionTime)
	rows, err := tx.Query(ctx, fmt.Sprintf(`
		SELECT extract(year FROM a.taken_at)::int AS year, count(*)::int AS asset_count
		FROM %s
		WHERE %s
		GROUP BY year
		ORDER BY year DESC`,
		domain(textMode), scope.where()), scope.args()...)
	if err != nil {
		return nil, fmt.Errorf("search: collect the years facet: %w", err)
	}
	defer rows.Close()
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (YearFacet, error) {
		var facet YearFacet
		err := row.Scan(&facet.Year, &facet.AssetCount)
		return facet, err
	})
}

// facetCameras counts the EXIF cameras of the scoped domain; the
// facet id is the round-trip filter value (make|model).
func facetCameras(ctx context.Context, tx pgx.Tx, caller uuid.UUID, filters Filters, textMode bool) ([]CameraFacet, error) {
	scope := newScope(caller)
	scope.addFilters(filters, dimensionCameras)
	rows, err := tx.Query(ctx, fmt.Sprintf(`
		SELECT coalesce(btrim(e.camera_make), '') || '|' || coalesce(btrim(e.camera_model), '') AS camera_id,
		       e.camera_make, e.camera_model, count(*)::int AS asset_count
		FROM %s
		JOIN exif e ON e.asset_id = a.id
		WHERE %s AND (e.camera_make IS NOT NULL OR e.camera_model IS NOT NULL)
		GROUP BY camera_id, e.camera_make, e.camera_model
		ORDER BY asset_count DESC, camera_id ASC`,
		domain(textMode), scope.where()), scope.args()...)
	if err != nil {
		return nil, fmt.Errorf("search: collect the cameras facet: %w", err)
	}
	defer rows.Close()
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (CameraFacet, error) {
		var facet CameraFacet
		err := row.Scan(&facet.ID, &facet.Make, &facet.Model, &facet.AssetCount)
		return facet, err
	})
}

// facetKinds counts the media families of the scoped domain.
func facetKinds(ctx context.Context, tx pgx.Tx, caller uuid.UUID, filters Filters, textMode bool) ([]KindFacet, error) {
	scope := newScope(caller)
	scope.addFilters(filters, dimensionKinds)
	rows, err := tx.Query(ctx, fmt.Sprintf(`
		SELECT a.media_kind::text AS kind, count(*)::int AS asset_count
		FROM %s
		WHERE %s
		GROUP BY kind
		ORDER BY asset_count DESC, kind ASC`,
		domain(textMode), scope.where()), scope.args()...)
	if err != nil {
		return nil, fmt.Errorf("search: collect the kinds facet: %w", err)
	}
	defer rows.Close()
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (KindFacet, error) {
		var facet KindFacet
		err := row.Scan(&facet.Kind, &facet.AssetCount)
		return facet, err
	})
}

// facetAlbums counts the caller-visible non-sync albums holding
// members of the scoped domain.
func facetAlbums(ctx context.Context, tx pgx.Tx, caller uuid.UUID, filters Filters, textMode bool) ([]AlbumFacet, error) {
	scope := newScope(caller)
	scope.addFilters(filters, dimensionAlbums)
	rows, err := tx.Query(ctx, fmt.Sprintf(`
		SELECT al.id, al.title, count(*)::int AS asset_count
		FROM %s
		JOIN album_items ai ON ai.asset_id = a.id
		JOIN albums al ON al.id = ai.album_id
		WHERE %s
		  AND al.type <> 'sync'
		  AND (al.owner_id = $%d OR EXISTS (
		      SELECT 1 FROM album_shares s
		      WHERE s.album_id = al.id AND s.user_id = $%d
		  ))
		GROUP BY al.id, al.title
		ORDER BY asset_count DESC, lower(al.title), al.id`,
		domain(textMode), scope.where(), scope.callerArg, scope.callerArg), scope.args()...)
	if err != nil {
		return nil, fmt.Errorf("search: collect the albums facet: %w", err)
	}
	defer rows.Close()
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (AlbumFacet, error) {
		var facet AlbumFacet
		err := row.Scan(&facet.ID, &facet.Title, &facet.AssetCount)
		return facet, err
	})
}

// facetTags counts the CALLER's tags with members in the scoped
// domain (another account's tags never surface).
func facetTags(ctx context.Context, tx pgx.Tx, caller uuid.UUID, filters Filters, textMode bool) ([]TagFacet, error) {
	scope := newScope(caller)
	scope.addFilters(filters, dimensionTags)
	rows, err := tx.Query(ctx, fmt.Sprintf(`
		SELECT t.id, t.name, count(*)::int AS asset_count
		FROM %s
		JOIN asset_tags atg ON atg.asset_id = a.id
		JOIN tags t ON t.id = atg.tag_id
		WHERE %s AND t.owner_id = $%d
		GROUP BY t.id, t.name
		ORDER BY asset_count DESC, lower(t.name), t.id`,
		domain(textMode), scope.where(), scope.callerArg), scope.args()...)
	if err != nil {
		return nil, fmt.Errorf("search: collect the tags facet: %w", err)
	}
	defer rows.Close()
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (TagFacet, error) {
		var facet TagFacet
		err := row.Scan(&facet.ID, &facet.Name, &facet.AssetCount)
		return facet, err
	})
}
