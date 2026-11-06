// The filter-browse mode (M3d Task 3): the request whose query takes
// no text path (absent, whitespace or single-character). The result
// is the caller's visible set narrowed by the filters, paged in the
// timeline's own ordering and keyset shape (the search page opens as
// the filter browse and the results wall walks it exactly like the
// timeline), with the facets over the scope minus each dimension.
package search

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// runFilterBrowse executes the filter-browse mode.
func runFilterBrowse(ctx context.Context, tx pgx.Tx, caller uuid.UUID, filters Filters, locale string, cursor Cursor, limit int) (Result, error) {
	if cursor.Mode != CursorFilterBrowse {
		return Result{}, fmt.Errorf("search: the cursor mode %d does not address the filter browse", cursor.Mode)
	}

	scope := newScope(caller)
	scope.addFilters(filters, dimensionNone)

	args := scope.args()
	where := scope.where()
	if cursor.ID != uuid.Nil {
		args = append(args, cursor.TakenAt, cursor.ID)
		where += fmt.Sprintf(" AND (a.taken_at, a.id) < ($%d, $%d)", len(args)-1, len(args))
	}

	rows, err := tx.Query(ctx, fmt.Sprintf(`
		SELECT %s
		FROM assets a
		WHERE %s
		ORDER BY a.taken_at DESC, a.id DESC
		LIMIT %d`, scope.itemColumnsOf(), where, limit+1), args...)
	if err != nil {
		return Result{}, fmt.Errorf("search: load the filter-browse page: %w", err)
	}
	defer rows.Close()

	items := make([]Item, 0, limit+1)
	for rows.Next() {
		item, err := scanItem(rows)
		if err != nil {
			return Result{}, fmt.Errorf("search: scan a filter-browse row: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return Result{}, fmt.Errorf("search: iterate the filter-browse rows: %w", err)
	}

	result := Result{Items: items}
	if len(items) > limit {
		items = items[:limit]
		result.Items = items
		last := items[len(items)-1]
		result.NextCursor = &Cursor{Mode: CursorFilterBrowse, TakenAt: last.TakenAt, ID: last.ID}
	}

	result.Facets, err = collectFacets(ctx, tx, caller, filters, locale, false)
	if err != nil {
		return Result{}, err
	}
	return result, nil
}

// The unused-import guard keeps edits honest.
var _ = uuid.Nil
