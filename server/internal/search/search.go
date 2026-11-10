// Package search implements the unified search engine:
// the unaccented trigram-substring text path, the CLIP semantic path,
// their deterministic rank fusion, the keyset pages of both modes and
// the honest facet counts of the current filtered set minus each
// dimension. The HTTP surface lives in the api package; this package
// owns the SQL.
//
// The queries here are composed, not generated: the filter set
// changes the shape of every scope (each facet counts the scope MINUS
// its own dimension), which is beyond sqlc's static analysis -- the
// same boundary the registry's DDL and the geodata import bypass
// generated code at. The composition is strictly parameterized (every
// dynamic value is a placeholder argument, never string-surgery into
// the SQL text) and the behavior is pinned by the api package's
// endpoint tests over a real database.
//
// Two invariants run through every statement:
//
//   - VISIBILITY: the album union (caller_asset_visible) plus the
//     timeline's live-row filters gate the text path, the semantic
//     path AND every facet count -- a sharee never sees an invisible
//     asset from either arm.
//   - DETERMINISM: every ordering is a total order (the asset id is
//     the final tiebreak everywhere), the fusion is integer
//     arithmetic over the arms' fixed-depth orderings, and the whole
//     request runs inside one transaction snapshot -- the same query
//     over unchanged data answers the same page.
package search

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// armDepth is each path's ranked-hit cut: the text path and the
// semantic path each contribute at most this many ranked hits to the
// fusion, so the fused page is bounded and the arms' orderings (and
// therefore the fusion) stay deterministic. The contract documents
// the value.
const armDepth = 500

// Filters is the validated, raw filter set of one search request. The
// id filters arrive unresolved; Run resolves them against the
// caller's domain (caller-visible albums, the caller's own tags) and
// drops the ids that do not resolve -- filters narrow, they never
// probe.
type Filters struct {
	// Query is the trimmed text query. Fewer than two runes takes no
	// text path and no semantic path (the filter browse mode).
	Query string
	// QueryEmbedding is the query's text-tower feature, computed by
	// the caller when the text tower is active. Nil disables the
	// semantic path.
	QueryEmbedding []float32
	// VisionModelID is the active clip-embed model whose embedding
	// rows the semantic path searches. Empty disables the path.
	VisionModelID string
	// PersonIDs restricts to assets carrying a face of one of these
	// clusters.
	PersonIDs []uuid.UUID
	// PlaceIDs restricts to assets resolved to one of these places.
	PlaceIDs []int32
	// From and To bound the taken-at range (inclusive, exclusive).
	From *time.Time
	To   *time.Time
	// Cameras restricts to EXIF cameras by facet id ("make|model").
	Cameras []string
	// Kinds restricts to media families ("image"/"video").
	Kinds []string
	// AlbumIDs restricts to album membership (resolved against
	// caller-visible albums).
	AlbumIDs []uuid.UUID
	// TagIDs restricts to the caller's own tags (foreign ids drop).
	TagIDs []uuid.UUID
}

// CursorMode tags the cursor's ordering: the filter browse pages in
// the timeline's order, the text query pages in the fused-rank order.
type CursorMode byte

const (
	// CursorFilterBrowse pages in (taken_at DESC, id DESC).
	CursorFilterBrowse CursorMode = 0
	// CursorFusedRank pages in (fused score DESC, id ASC).
	CursorFusedRank CursorMode = 1
)

// Cursor is one search page's continuation point: the mode's ordering
// key of the last item of the previous page. The zero value addresses
// the first page.
type Cursor struct {
	Mode    CursorMode
	Score   int64
	TakenAt time.Time
	ID      uuid.UUID
}

// Item is one result row in the timeline item's shape (the api
// package maps it onto the contract type).
type Item struct {
	ID               uuid.UUID
	OwnerID          uuid.UUID
	Kind             string
	OriginalFilename string
	FileSize         int64
	MimeType         string
	Checksum         string
	CreatedAt        time.Time
	TakenAt          time.Time
	TakenAtSource    string
	IsFavorite       bool
	IsReady          bool
	Width            int32
	Height           int32
	DurationMs       *int32
	Blurhash         *string
	LivePhotoVideoID *uuid.UUID
}

// Facets is the filter bar's counts of the current filtered set
// minus each dimension.
type Facets struct {
	People  []PersonFacet
	Places  []PlaceFacet
	Years   []YearFacet
	Cameras []CameraFacet
	Kinds   []KindFacet
	Albums  []AlbumFacet
	Tags    []TagFacet
}

// PersonFacet is one named person of the people facet.
type PersonFacet struct {
	ID         uuid.UUID
	Name       string
	AssetCount int32
}

// PlaceFacet is one place of the places facet.
type PlaceFacet struct {
	PlaceID    int32
	Name       string
	Country    string
	AssetCount int32
}

// YearFacet is one year of the time-range facet.
type YearFacet struct {
	Year       int32
	AssetCount int32
}

// CameraFacet is one camera of the cameras facet.
type CameraFacet struct {
	ID         string
	Make       *string
	Model      *string
	AssetCount int32
}

// KindFacet is one media family of the kinds facet.
type KindFacet struct {
	Kind       string
	AssetCount int32
}

// AlbumFacet is one album of the albums facet.
type AlbumFacet struct {
	ID         uuid.UUID
	Title      string
	AssetCount int32
}

// TagFacet is one tag of the tags facet.
type TagFacet struct {
	ID         uuid.UUID
	Name       string
	AssetCount int32
}

// Result is one search page.
type Result struct {
	Items      []Item
	NextCursor *Cursor
	Facets     Facets
}

// IsTextQuery reports whether the filters carry a text query that
// takes the text paths: at least two runes after trimming.
func (f Filters) IsTextQuery() bool {
	return len([]rune(strings.TrimSpace(f.Query))) >= 2
}

// Run executes one search request: the resolution of the id filters,
// the two paths (when the query takes them), the fusion, the keyset
// page and the facets, all inside one transaction snapshot.
func Run(ctx context.Context, pool *pgxpool.Pool, caller uuid.UUID, filters Filters, locale string, cursor Cursor, limit int) (Result, error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return Result{}, fmt.Errorf("search: begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }() // no-op once committed

	filters.AlbumIDs, err = resolveCallerAlbums(ctx, tx, caller, filters.AlbumIDs)
	if err != nil {
		return Result{}, err
	}
	filters.TagIDs, err = resolveCallerTags(ctx, tx, caller, filters.TagIDs)
	if err != nil {
		return Result{}, err
	}

	var result Result
	if filters.IsTextQuery() {
		result, err = runTextQuery(ctx, tx, caller, filters, locale, cursor, limit)
	} else {
		result, err = runFilterBrowse(ctx, tx, caller, filters, locale, cursor, limit)
	}
	if err != nil {
		return Result{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Result{}, fmt.Errorf("search: commit: %w", err)
	}
	return result, nil
}

// resolveCallerAlbums keeps the album ids the caller can see (own
// albums of any type, or albums shared to them): the others drop out
// of the filter -- they contribute nothing, never an empty result.
func resolveCallerAlbums(ctx context.Context, tx pgx.Tx, caller uuid.UUID, ids []uuid.UUID) ([]uuid.UUID, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	rows, err := tx.Query(ctx, `
		SELECT id FROM albums
		WHERE id = ANY($1::uuid[])
		  AND (owner_id = $2 OR EXISTS (
		      SELECT 1 FROM album_shares s
		      WHERE s.album_id = albums.id AND s.user_id = $2
		  ))`, ids, caller)
	if err != nil {
		return nil, fmt.Errorf("search: resolve the album filter: %w", err)
	}
	defer rows.Close()
	resolved, err := pgx.CollectRows(rows, pgx.RowTo[uuid.UUID])
	if err != nil {
		return nil, fmt.Errorf("search: read the resolved album filter: %w", err)
	}
	return resolved, nil
}

// resolveCallerTags keeps the tag ids the CALLER owns (tags are
// per-user state): foreign ids drop out of the filter.
func resolveCallerTags(ctx context.Context, tx pgx.Tx, caller uuid.UUID, ids []uuid.UUID) ([]uuid.UUID, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	rows, err := tx.Query(ctx, `
		SELECT id FROM tags
		WHERE id = ANY($1::uuid[]) AND owner_id = $2`, ids, caller)
	if err != nil {
		return nil, fmt.Errorf("search: resolve the tag filter: %w", err)
	}
	defer rows.Close()
	resolved, err := pgx.CollectRows(rows, pgx.RowTo[uuid.UUID])
	if err != nil {
		return nil, fmt.Errorf("search: read the resolved tag filter: %w", err)
	}
	return resolved, nil
}
