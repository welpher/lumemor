// The text-query mode: the request whose query carries
// at least two runes. The two paths run inside the request's
// transaction snapshot:
//
//   - the TEXT path materializes the text-match domain into a
//     connection-local temporary table (the unaccented substring
//     matches over the caller-visible live assets, with their best
//     matched source's trigram similarity as the rank); the result
//     arm then intersects the domain with the full scope, and every
//     facet counts the domain intersected with its own
//     scope-minus-dimension -- one match computation, seven honest
//     consumers.
//   - the SEMANTIC path (when the caller armed it with the query's
//     text-tower embedding and the active vision model) ranks the
//     caller-visible embeddings by cosine distance over the same
//     scope.
//
// The arms fuse by the deterministic rank-sum rule (fuse.go); the
// fused list pages by (score DESC, id ASC).
package search

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/pgvector/pgvector-go"
)

// runTextQuery executes the query mode.
func runTextQuery(ctx context.Context, tx pgx.Tx, caller uuid.UUID, filters Filters, locale string, cursor Cursor, limit int) (Result, error) {
	if cursor.Mode != CursorFusedRank {
		return Result{}, fmt.Errorf("search: the cursor mode %d does not address a text query", cursor.Mode)
	}

	// The unaccented query text: the ILIKE wildcards are escaped first
	// (the query is a literal substring, never a pattern), the accent
	// rule runs here so the pattern and the indexed expressions share
	// exactly one rule. The unaccent rules never emit a wildcard or
	// the escape character, so escaping first stays honest. The rank
	// signal compares against the unescaped unaccented text.
	var escaped, plain string
	if err := tx.QueryRow(ctx, "SELECT f_unaccent($1), f_unaccent($2)",
		escapeLike(filters.Query), filters.Query).Scan(&escaped, &plain); err != nil {
		return Result{}, fmt.Errorf("search: unaccent the query: %w", err)
	}
	pattern := "%" + escaped + "%"

	if err := createTextHits(ctx, tx, caller, pattern, plain); err != nil {
		return Result{}, err
	}

	// The text path's ranked arm.
	textHits, err := textArm(ctx, tx, caller, filters)
	if err != nil {
		return Result{}, err
	}

	// The semantic path's ranked arm (armed by the caller; the
	// degradation with no text tower or no vision tower is the
	// handler's honest empty answer).
	var vectorHits []rankedHit
	if filters.QueryEmbedding != nil && filters.VisionModelID != "" {
		vectorHits, err = vectorArm(ctx, tx, caller, filters)
		if err != nil {
			return Result{}, err
		}
	}

	// The fusion, the page and the page's full rows.
	fused := fuse(textHits, vectorHits)
	if cursor.ID != uuid.Nil {
		fused = fusedAfter(fused, cursor)
	}
	page := fused
	if len(page) > limit {
		page = page[:limit]
	}
	items, err := fetchItems(ctx, tx, caller, rankedIDs(page))
	if err != nil {
		return Result{}, err
	}

	result := Result{Items: items}
	if len(fused) > limit && len(page) > 0 {
		last := page[len(page)-1]
		result.NextCursor = &Cursor{Mode: CursorFusedRank, Score: last.score, ID: last.assetID}
	}

	result.Facets, err = collectFacets(ctx, tx, caller, filters, locale, true)
	if err != nil {
		return Result{}, err
	}
	return result, nil
}

// createTextHits materializes the text-match domain into the
// connection-local hits table: every caller-visible live asset
// matching the pattern through ANY of the six text sources, with the
// best matched source's trigram similarity as its rank. The table
// lives for the transaction; ON COMMIT DROP cleans it. The
// relational sources (tags, album titles, person labels) are per-user
// state: they probe the CALLER's rows only.
func createTextHits(ctx context.Context, tx pgx.Tx, caller uuid.UUID, pattern, plain string) error {
	_, err := tx.Exec(ctx, `
		CREATE TEMP TABLE search_text_hits ON COMMIT DROP AS
		SELECT a.id AS asset_id,
		       greatest(
		           CASE WHEN f_unaccent(a.original_filename) ILIKE $2
		                THEN similarity(f_unaccent(a.original_filename), $3) ELSE 0 END,
		           CASE WHEN f_unaccent(a.description) ILIKE $2
		                THEN similarity(f_unaccent(a.description), $3) ELSE 0 END,
		           COALESCE((
		               SELECT similarity(f_unaccent(o.text), $3)
		               FROM ocr_texts o
		               WHERE o.asset_id = a.id AND f_unaccent(o.text) ILIKE $2
		           ), 0),
		           COALESCE((
		               SELECT max(similarity(f_unaccent(t.name), $3))
		               FROM asset_tags atg
		               JOIN tags t ON t.id = atg.tag_id
		               WHERE atg.asset_id = a.id
		                 AND t.owner_id = $1
		                 AND f_unaccent(t.name) ILIKE $2
		           ), 0),
		           COALESCE((
		               SELECT max(similarity(f_unaccent(al.title), $3))
		               FROM album_items ai
		               JOIN albums al ON al.id = ai.album_id
		               WHERE ai.asset_id = a.id
		                 AND al.type <> 'sync'
		                 AND (al.owner_id = $1 OR EXISTS (
		                     SELECT 1 FROM album_shares s
		                     WHERE s.album_id = al.id AND s.user_id = $1
		                 ))
		                 AND f_unaccent(al.title) ILIKE $2
		           ), 0),
		           COALESCE((
		               SELECT max(similarity(f_unaccent(l.name), $3))
		               FROM faces f2
		               JOIN user_face_labels l
		                    ON l.cluster_id = f2.cluster_id AND l.user_id = $1
		               WHERE f2.asset_id = a.id
		                 AND f_unaccent(l.name) ILIKE $2
		           ), 0)
		       ) AS rank
		FROM assets a
		WHERE a.deleted_at IS NULL
		  AND a.is_hidden = false
		  AND a.is_component = false
		  AND caller_asset_visible($1, a.id)
		  AND (
		      f_unaccent(a.original_filename) ILIKE $2
		      OR f_unaccent(a.description) ILIKE $2
		      OR EXISTS (
		          SELECT 1 FROM ocr_texts o
		          WHERE o.asset_id = a.id AND f_unaccent(o.text) ILIKE $2
		      )
		      OR EXISTS (
		          SELECT 1 FROM asset_tags atg
		          JOIN tags t ON t.id = atg.tag_id
		          WHERE atg.asset_id = a.id
		            AND t.owner_id = $1
		            AND f_unaccent(t.name) ILIKE $2
		      )
		      OR EXISTS (
		          SELECT 1 FROM album_items ai
		          JOIN albums al ON al.id = ai.album_id
		          WHERE ai.asset_id = a.id
		            AND al.type <> 'sync'
		            AND (al.owner_id = $1 OR EXISTS (
		                SELECT 1 FROM album_shares s
		                WHERE s.album_id = al.id AND s.user_id = $1
		            ))
		            AND f_unaccent(al.title) ILIKE $2
		      )
		      OR EXISTS (
		          SELECT 1 FROM faces f2
		          JOIN user_face_labels l
		               ON l.cluster_id = f2.cluster_id AND l.user_id = $1
		          WHERE f2.asset_id = a.id
		            AND f_unaccent(l.name) ILIKE $2
		      )
		  )`, caller, pattern, plain)
	if err != nil {
		return fmt.Errorf("search: materialize the text-match domain: %w", err)
	}
	return nil
}

// textArm ranks the text path's hits: the domain intersected with the
// full scope, ordered by the rank descending with the asset id
// ascending (the pinned arm order), cut at armDepth.
func textArm(ctx context.Context, tx pgx.Tx, caller uuid.UUID, filters Filters) ([]rankedHit, error) {
	scope := newScope(caller)
	scope.addFilters(filters, dimensionNone)

	rows, err := tx.Query(ctx, fmt.Sprintf(`
		SELECT h.asset_id
		FROM search_text_hits h
		JOIN assets a ON a.id = h.asset_id
		WHERE %s
		ORDER BY h.rank DESC, h.asset_id ASC
		LIMIT %d`, scope.where(), armDepth), scope.args()...)
	if err != nil {
		return nil, fmt.Errorf("search: rank the text path: %w", err)
	}
	defer rows.Close()
	return collectHits(rows, "text path")
}

// vectorArm ranks the semantic path: the active vision model's
// embedding rows over the caller-visible scope, ordered by cosine
// distance ascending (similarity descending) with the asset id
// ascending, cut at armDepth.
func vectorArm(ctx context.Context, tx pgx.Tx, caller uuid.UUID, filters Filters) ([]rankedHit, error) {
	scope := newScope(caller)
	scope.addFilters(filters, dimensionNone)
	scope.values = append(scope.values, pgvector.NewVector(filters.QueryEmbedding), filters.VisionModelID)
	vectorArg := len(scope.values) - 1
	modelArg := len(scope.values)

	rows, err := tx.Query(ctx, fmt.Sprintf(`
		SELECT ce.asset_id
		FROM clip_embeddings ce
		JOIN assets a ON a.id = ce.asset_id
		WHERE ce.model_id = $%d AND %s
		ORDER BY ce.embedding <=> $%d::vector, ce.asset_id ASC
		LIMIT %d`, modelArg, scope.where(), vectorArg, armDepth), scope.args()...)
	if err != nil {
		return nil, fmt.Errorf("search: rank the semantic path: %w", err)
	}
	defer rows.Close()
	return collectHits(rows, "semantic path")
}

// collectHits scans one arm's ranked asset ids in order.
func collectHits(rows pgx.Rows, what string) ([]rankedHit, error) {
	var hits []rankedHit
	for rows.Next() {
		var hit rankedHit
		if err := rows.Scan(&hit.assetID); err != nil {
			return nil, fmt.Errorf("search: scan a %s hit: %w", what, err)
		}
		hits = append(hits, hit)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("search: iterate the %s hits: %w", what, err)
	}
	return hits, nil
}

// rankedIDs maps the fused page onto its asset ids.
func rankedIDs(hits []rankedHit) []uuid.UUID {
	ids := make([]uuid.UUID, 0, len(hits))
	for _, hit := range hits {
		ids = append(ids, hit.assetID)
	}
	return ids
}

// fusedAfter keeps the fused rows strictly after the cursor's
// (score, id) point of the (score DESC, id ASC) ordering.
func fusedAfter(hits []rankedHit, cursor Cursor) []rankedHit {
	kept := make([]rankedHit, 0, len(hits))
	for _, hit := range hits {
		if hit.score < cursor.Score || (hit.score == cursor.Score && compareUUID(hit.assetID, cursor.ID) > 0) {
			kept = append(kept, hit)
		}
	}
	return kept
}

// fetchItems loads the page's full result rows (the timeline item's
// shape plus the per-user favorite flag) and returns them in the
// given id order. The ids arrive already scoped by the arms inside
// the same transaction snapshot.
func fetchItems(ctx context.Context, tx pgx.Tx, caller uuid.UUID, ids []uuid.UUID) ([]Item, error) {
	if len(ids) == 0 {
		return []Item{}, nil
	}

	rows, err := tx.Query(ctx, fmt.Sprintf(`
		SELECT %s
		FROM assets a
		WHERE a.id = ANY($1::uuid[])
		ORDER BY a.taken_at DESC, a.id DESC`, fmt.Sprintf(itemColumns, 2)), ids, caller)
	if err != nil {
		return nil, fmt.Errorf("search: load the page's rows: %w", err)
	}
	defer rows.Close()

	byID := make(map[uuid.UUID]Item, len(ids))
	for rows.Next() {
		item, err := scanItem(rows)
		if err != nil {
			return nil, fmt.Errorf("search: scan a result row: %w", err)
		}
		byID[item.ID] = item
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("search: iterate the result rows: %w", err)
	}

	items := make([]Item, 0, len(ids))
	for _, id := range ids {
		item, ok := byID[id]
		if !ok {
			// Inside one snapshot this cannot happen; the honest drop
			// keeps the page consistent if it ever does.
			continue
		}
		items = append(items, item)
	}
	return items, nil
}
