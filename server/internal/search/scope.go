// The scope composer: the shared WHERE fragments every
// search statement builds on -- the timeline's live-row filters, the
// caller-visible union predicate, the library representative and the
// filter dimensions. Each facet query re-composes the scope EXCLUDING
// its own dimension; the result pages compose it with every dimension
// applied. All dynamic values ride as placeholder arguments.
package search

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// dimension identifies one facet dimension: the scope a facet counts
// excludes exactly its own dimension's filters.
type dimension int

const (
	dimensionNone dimension = iota
	dimensionPeople
	dimensionPlaces
	dimensionTime
	dimensionCameras
	dimensionKinds
	dimensionAlbums
	dimensionTags
)

// scope composes the WHERE body over the assets alias `a` (every
// consumer of the scope aliases assets as `a`, the facet domains
// included: they join the hits table onto `a`).
type scope struct {
	clauses []string
	values  []any
	// callerArg is the placeholder index of the caller argument (the
	// visibility predicate's viewer): consumers that need the caller
	// again (the per-user favorite flag) reuse the placeholder.
	callerArg int
}

// args answers the composed arguments in placeholder order.
func (s *scope) args() []any {
	return s.values
}

// newScope starts the scope with the timeline's live-row filters, the
// library representative and the caller-visible union predicate: the
// search wall is a library face, so one photo held as an uploaded
// copy and as a directory album's position row is one hit.
func newScope(caller uuid.UUID) *scope {
	s := &scope{}
	s.clauses = append(s.clauses,
		"a.deleted_at IS NULL",
		"a.is_hidden = false",
		"a.is_component = false",
		"asset_library_representative(a.id)",
	)
	s.callerArg = len(s.values) + 1
	s.values = append(s.values, caller)
	s.clauses = append(s.clauses, fmt.Sprintf("caller_asset_visible($%d, a.id)", s.callerArg))
	return s
}

// addFilters appends the filter clauses of every dimension except the
// excluded one (dimensionNone keeps them all -- the result pages).
func (s *scope) addFilters(filters Filters, exclude dimension) {
	if len(filters.PersonIDs) > 0 && exclude != dimensionPeople {
		s.clause("EXISTS (SELECT 1 FROM faces f WHERE f.asset_id = a.id AND f.cluster_id = ANY(%s::uuid[]))",
			filters.PersonIDs)
	}
	if len(filters.PlaceIDs) > 0 && exclude != dimensionPlaces {
		s.clause("a.place_id = ANY(%s::int4[])", filters.PlaceIDs)
	}
	if filters.From != nil && exclude != dimensionTime {
		s.clause("a.taken_at >= %s", *filters.From)
	}
	if filters.To != nil && exclude != dimensionTime {
		s.clause("a.taken_at < %s", *filters.To)
	}
	if len(filters.Cameras) > 0 && exclude != dimensionCameras {
		s.clause(`EXISTS (
			SELECT 1 FROM exif e
			WHERE e.asset_id = a.id
			  AND coalesce(btrim(e.camera_make), '') || '|' || coalesce(btrim(e.camera_model), '') = ANY(%s::text[])
		)`, filters.Cameras)
	}
	if len(filters.Kinds) > 0 && exclude != dimensionKinds {
		s.clause("a.media_kind = ANY(%s::media_kind[])", filters.Kinds)
	}
	if len(filters.AlbumIDs) > 0 && exclude != dimensionAlbums {
		s.clause("EXISTS (SELECT 1 FROM album_items ai WHERE ai.asset_id = a.id AND ai.album_id = ANY(%s::uuid[]))",
			filters.AlbumIDs)
	}
	// isMotion is not a facet dimension: every facet count includes it
	// (a Live Photo's facets are its own), and the result pages apply
	// it like any other narrowing.
	if filters.IsMotion != nil {
		if *filters.IsMotion {
			s.clauses = append(s.clauses, "a.live_photo_video_id IS NOT NULL")
		} else {
			s.clauses = append(s.clauses, "a.live_photo_video_id IS NULL")
		}
	}
	if len(filters.TagIDs) > 0 && exclude != dimensionTags {
		s.clause("EXISTS (SELECT 1 FROM asset_tags atg WHERE atg.asset_id = a.id AND atg.tag_id = ANY(%s::uuid[]))",
			filters.TagIDs)
	}
}

// clause appends one parameterized clause: value becomes the next
// placeholder argument and the %s in the fragment is its placeholder.
func (s *scope) clause(fragment string, value any) {
	s.values = append(s.values, value)
	s.clauses = append(s.clauses, fmt.Sprintf(fragment, fmt.Sprintf("$%d", len(s.values))))
}

// where renders the composed WHERE body (without the WHERE keyword).
func (s *scope) where() string {
	return strings.Join(s.clauses, "\n  AND ")
}

// itemColumns is the shared result-row projection: the timeline
// item's fields plus the per-user favorite flag. $favorite is a
// placeholder for the caller's argument index.
const itemColumns = `a.id, a.owner_id, a.media_kind::text, a.original_filename,
       a.file_size, a.mime_type, a.checksum, a.created_at, a.taken_at,
       a.taken_at_source::text, a.ready_at IS NOT NULL, a.width, a.height,
       a.duration_ms, a.blurhash, a.live_photo_video_id,
       EXISTS (
           SELECT 1 FROM user_favorites uf
           WHERE uf.asset_id = a.id AND uf.user_id = $%d
       )`

// itemColumnsOf renders the projection with the caller's placeholder
// index substituted.
func (s *scope) itemColumnsOf() string {
	return fmt.Sprintf(itemColumns, s.callerArg)
}

// scanItem scans one result row in the projection's order.
func scanItem(row pgxRow) (Item, error) {
	var item Item
	var takenAt time.Time
	err := row.Scan(
		&item.ID, &item.OwnerID, &item.Kind, &item.OriginalFilename,
		&item.FileSize, &item.MimeType, &item.Checksum, &item.CreatedAt, &takenAt,
		&item.TakenAtSource, &item.IsReady, &item.Width, &item.Height,
		&item.DurationMs, &item.Blurhash, &item.LivePhotoVideoID, &item.IsFavorite,
	)
	item.TakenAt = takenAt.UTC()
	return item, err
}

// pgxRow is the scanning seam shared by Query rows and QueryRow.
type pgxRow interface {
	Scan(dest ...any) error
}
