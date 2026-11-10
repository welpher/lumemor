// The idempotent database import of the baked reference data,
// plus Prepare, the boot orchestration that ties the bake,
// the import and the in-memory grid together.
//
// The import's skip key is the data-version marker: a sha256 digest
// of the exact staged bytes, stored in instance_settings. An
// unchanged dataset (the normal reboot) answers imported=false and
// touches nothing; a changed one (a rebuilt image) replaces the
// places/place_names tables wholesale inside one transaction -- the
// DELETE pair empties them, the bulk COPY lands the new rows, and the
// marker writes with them, so the marker and the rows it describes
// commit together. Because assets.place_id carries ON DELETE SET
// NULL, the replacement releases every stale resolution: resolutions
// belong to the data version that produced them, and the admin
// re-resolution run is the path that refills them.
//
// The COPY rides pgx's binary copy protocol through a CopyFromSource
// over the parsed dataset -- no per-row statements for the ~185k
// places and ~260k locale names, and no intermediate row slice: the
// source streams the parsed structs directly.
//
// The grid is built FROM THE DATABASE rows, not the parsed bake: the
// resolver can then only ever answer place ids the places table
// actually carries, which is what keeps the metadata job's writes
// inside the foreign key whatever raced the import.
package geodata

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	dbgen "github.com/welpher/lumemor/server/internal/db/gen"
)

// VersionSettingKey is the instance_settings key holding the loaded
// bake's data-version marker.
const VersionSettingKey = "geodata_version"

// Import lands the dataset in the places/place_names tables when its
// version differs from the marker, answering whether it imported. An
// unchanged version is a full skip -- no transaction, no writes, the
// normal reboot's fast path. A changed version replaces the tables
// atomically: the DELETE pair empties them (releasing every
// assets.place_id through ON DELETE SET NULL), the COPY lands the new
// rows, and the marker records the new version, all in one
// transaction whose failure leaves the previous state untouched.
func Import(ctx context.Context, pool *pgxpool.Pool, ds *Dataset) (bool, error) {
	queries := dbgen.New(pool)
	current, err := queries.GetGeodataVersion(ctx, VersionSettingKey)
	if errors.Is(err, pgx.ErrNoRows) {
		current = ""
	} else if err != nil {
		return false, fmt.Errorf("geodata: read the data-version marker: %w", err)
	}
	if current == ds.Version {
		return false, nil
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return false, fmt.Errorf("geodata: begin the import: %w", err)
	}
	// The rollback is a no-op after a successful commit; it is
	// deferred while the context is still live so the cancel path
	// cannot leak the transaction.
	defer func() { _ = tx.Rollback(ctx) }()

	txQueries := dbgen.New(tx)
	if err := txQueries.DeleteAllPlaceNames(ctx); err != nil {
		return false, fmt.Errorf("geodata: empty place_names: %w", err)
	}
	if err := txQueries.DeleteAllPlaces(ctx); err != nil {
		return false, fmt.Errorf("geodata: empty places: %w", err)
	}

	places := &placeSource{rows: ds.Places}
	if _, err := tx.CopyFrom(ctx, pgx.Identifier{"places"},
		[]string{"id", "name", "country_code", "admin1_code", "timezone", "lat", "lon"}, places); err != nil {
		return false, fmt.Errorf("geodata: copy places: %w", err)
	}
	names := &placeNameSource{rows: ds.Names}
	if _, err := tx.CopyFrom(ctx, pgx.Identifier{"place_names"},
		[]string{"geoname_id", "locale", "name"}, names); err != nil {
		return false, fmt.Errorf("geodata: copy place_names: %w", err)
	}

	if err := txQueries.SetGeodataVersion(ctx, dbgen.SetGeodataVersionParams{
		Key:     VersionSettingKey,
		Version: ds.Version,
	}); err != nil {
		return false, fmt.Errorf("geodata: record the data-version marker: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return false, fmt.Errorf("geodata: commit the import: %w", err)
	}
	return true, nil
}

// Prepare is the boot orchestration over one geo dir: it loads the
// bake (a missing file or malformed row is a deployment error, and
// an EMPTY dir string is the feature-off configuration that answers
// no index and no error -- a deployment without the baked place data
// keeps the place resolution honestly absent instead of failing),
// imports it when the data-version marker differs, and builds the
// in-memory grid from the database's places rows. The second return
// reports whether the import ran (a first boot or a data update);
// the normal reboot skips it and still answers a working grid.
func Prepare(ctx context.Context, pool *pgxpool.Pool, dir string) (*Index, bool, error) {
	if dir == "" {
		return nil, false, nil
	}
	ds, err := Load(dir)
	if err != nil {
		return nil, false, err
	}
	imported, err := Import(ctx, pool, ds)
	if err != nil {
		return nil, false, err
	}

	rows, err := dbgen.New(pool).ListPlaceCentroids(ctx)
	if err != nil {
		return nil, imported, fmt.Errorf("geodata: read the places centroids: %w", err)
	}
	centroids := make([]Centroid, 0, len(rows))
	for _, row := range rows {
		centroids = append(centroids, Centroid{ID: row.ID, Lat: row.Lat, Lon: row.Lon})
	}
	return NewIndex(centroids), imported, nil
}

// placeSource streams the places rows into pgx's COPY protocol: one
// destination slice per column, refilled per row.
type placeSource struct {
	rows []Place
	next int
}

// Next advances to the next row, reporting whether one exists.
func (s *placeSource) Next() bool {
	s.next++
	return s.next-1 < len(s.rows)
}

// Err answers the source's error state: a plain slice never errors.
func (s *placeSource) Err() error {
	return nil
}

// Values fills the COPY row from the current place.
func (s *placeSource) Values() ([]any, error) {
	row := s.rows[s.next-1]
	return []any{row.ID, row.Name, row.CountryCode, row.Admin1Code, row.Timezone, row.Lat, row.Lon}, nil
}

// placeNameSource streams the place_names rows into pgx's COPY
// protocol.
type placeNameSource struct {
	rows []PlaceName
	next int
}

// Next advances to the next row, reporting whether one exists.
func (s *placeNameSource) Next() bool {
	s.next++
	return s.next-1 < len(s.rows)
}

// Err answers the source's error state: a plain slice never errors.
func (s *placeNameSource) Err() error {
	return nil
}

// Values fills the COPY row from the current place name.
func (s *placeNameSource) Values() ([]any, error) {
	row := s.rows[s.next-1]
	return []any{row.GeonameID, row.Locale, row.Name}, nil
}
