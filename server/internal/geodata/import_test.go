// Import tests: the idempotent startup load of the baked reference
// data into places/place_names. The data-version marker (a digest of
// the loaded bytes) is the skip key: an unchanged dataset imports
// nothing on the next boot, a changed dataset replaces the tables
// atomically -- and because assets.place_id carries ON DELETE SET
// NULL, a replacement releases every stale resolution, which is
// exactly the state the admin re-resolution run then refills.
package geodata

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/welpher/lumemor/server/internal/db"
	"github.com/welpher/lumemor/server/internal/db/dbtest"
)

// importEnv provisions a migrated throwaway database.
func importEnv(t *testing.T) *pgxpool.Pool {
	t.Helper()
	pool := dbtest.New(t)
	if err := db.Migrate(t.Context(), pool); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
	return pool
}

// writeFixtureGeoDir copies the testdata bake into a fresh temp
// directory the import can point at.
func writeFixtureGeoDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	for _, name := range []string{CitiesFile, Admin1File, AlternateZhFile} {
		data, err := os.ReadFile(filepath.Join("testdata", name))
		if err != nil {
			t.Fatalf("read fixture %s: %v", name, err)
		}
		if err := os.WriteFile(filepath.Join(dir, name), data, 0o644); err != nil {
			t.Fatalf("copy fixture %s: %v", name, err)
		}
	}
	return dir
}

// requireCounts asserts the table row counts.
func requireCounts(t *testing.T, pool *pgxpool.Pool, wantPlaces, wantNames int) {
	t.Helper()
	var places, names int
	if err := pool.QueryRow(context.Background(), "SELECT count(*) FROM places").Scan(&places); err != nil {
		t.Fatalf("count places: %v", err)
	}
	if err := pool.QueryRow(context.Background(), "SELECT count(*) FROM place_names").Scan(&names); err != nil {
		t.Fatalf("count place_names: %v", err)
	}
	if places != wantPlaces || names != wantNames {
		t.Fatalf("rows = %d places / %d names, want %d / %d", places, names, wantPlaces, wantNames)
	}
}

// TestImportIdempotentByVersionMarker pins the startup discipline:
// the first boot imports the dataset and writes the version marker;
// the second boot over the identical dataset imports nothing (the
// marker matches); a dataset whose bytes changed imports again, and
// the replacement leaves exactly the new rows.
func TestImportIdempotentByVersionMarker(t *testing.T) {
	pool := importEnv(t)
	ds := loadFixtureDataset(t)

	imported, err := Import(context.Background(), pool, ds)
	if err != nil {
		t.Fatalf("first Import: %v", err)
	}
	if !imported {
		t.Fatal("the first Import reported no import, want one")
	}
	requireCounts(t, pool, 4, 10)

	// The second boot over the same bytes: the marker skips the work.
	imported, err = Import(context.Background(), pool, ds)
	if err != nil {
		t.Fatalf("second Import: %v", err)
	}
	if imported {
		t.Fatal("the second Import over identical bytes imported again, want the version marker to skip it")
	}
	requireCounts(t, pool, 4, 10)

	// A mutated dataset (one city dropped, one name row changed):
	// the marker no longer matches, the import replaces the tables.
	mutated := &Dataset{
		Version: "changed-version",
		Places:  []Place{{ID: 1, Name: "One", CountryCode: "AA", Admin1Code: "AA.01", Timezone: "UTC", Lat: 1, Lon: 1}},
		Names:   []PlaceName{{GeonameID: 1, Locale: LocaleEn, Name: "One"}},
	}
	imported, err = Import(context.Background(), pool, mutated)
	if err != nil {
		t.Fatalf("mutated Import: %v", err)
	}
	if !imported {
		t.Fatal("the mutated Import reported no import, want a replacement")
	}
	requireCounts(t, pool, 1, 1)

	// And the replaced marker is the new one: re-importing the
	// original dataset swaps back (the marker tracks the CURRENT
	// data, not the history).
	if imported, err := Import(context.Background(), pool, ds); err != nil || !imported {
		t.Fatalf("re-Import of the original dataset = (%v, %v), want a swap-back import", imported, err)
	}
	requireCounts(t, pool, 4, 10)
}

// TestImportReplacesReleasesAssetPlaces pins the FK semantics the
// data-update path relies on: deleting the old places rows (the
// replacement's first step) sets every referencing assets.place_id
// to NULL through the schema's ON DELETE SET NULL -- the honest
// "resolution belongs to the data version that produced it" state.
func TestImportReplacesReleasesAssetPlaces(t *testing.T) {
	pool := importEnv(t)
	ds := loadFixtureDataset(t)
	if _, err := Import(context.Background(), pool, ds); err != nil {
		t.Fatalf("Import: %v", err)
	}

	// A user row first: the asset's owner foreign key needs one, and
	// the walk below never depends on its identity.
	var ownerID string
	if err := pool.QueryRow(context.Background(), `
		INSERT INTO users (id, name, email, password_hash)
		VALUES (gen_random_uuid(), 'Owner', 'owner@example.test', 'x')
		RETURNING id`).Scan(&ownerID); err != nil {
		t.Fatalf("seed owner: %v", err)
	}

	var assetID string
	if err := pool.QueryRow(context.Background(), `
		INSERT INTO assets (
			id, owner_id, storage_kind, file_path, checksum, original_filename,
			file_size, mime_type, media_kind, width, height, taken_at, taken_at_source,
			place_id
		) VALUES (
			gen_random_uuid(), $1, 'blob', 'originals/x/a.jpg', 'ck',
			'a.jpg', 1, 'image/jpeg', 'image', 1, 1, now(), 'import',
			2988507
		)
		RETURNING id`, ownerID).Scan(&assetID); err != nil {
		t.Fatalf("seed asset with a place: %v", err)
	}

	var placeID *int32
	if err := pool.QueryRow(context.Background(),
		"SELECT place_id FROM assets WHERE id = $1", assetID).Scan(&placeID); err != nil {
		t.Fatalf("read place_id: %v", err)
	}
	if placeID == nil || *placeID != 2988507 {
		t.Fatalf("place_id = %v, want 2988507", placeID)
	}

	// A data update replaces the tables; Paris is gone in the
	// replacement, so the asset's resolution releases to NULL.
	replacement := &Dataset{
		Version: "replacement-version",
		Places:  []Place{{ID: 1816670, Name: "Beijing", CountryCode: "CN", Admin1Code: "CN.22", Timezone: "Asia/Shanghai", Lat: 39.9075, Lon: 116.39723}},
		Names:   []PlaceName{{GeonameID: 1816670, Locale: LocaleEn, Name: "Beijing"}},
	}
	if _, err := Import(context.Background(), pool, replacement); err != nil {
		t.Fatalf("replacement Import: %v", err)
	}
	if err := pool.QueryRow(context.Background(),
		"SELECT place_id FROM assets WHERE id = $1", assetID).Scan(&placeID); err != nil {
		t.Fatalf("re-read place_id: %v", err)
	}
	if placeID != nil {
		t.Fatalf("place_id after the data replacement = %v, want NULL", *placeID)
	}
}

// TestPrepareWiresTheIndexToTheDatabase pins the boot orchestration:
// the dataset loads from the directory, imports (or skips on a
// matching marker), and the in-memory grid is built FROM THE
// DATABASE rows -- the resolver can only ever answer place ids the
// places table actually carries, which is what keeps the metadata
// job's writes inside the foreign key.
func TestPrepareWiresTheIndexToTheDatabase(t *testing.T) {
	pool := importEnv(t)

	// An empty geo dir configuration (the local development default)
	// answers no index and no error.
	if ix, _, err := Prepare(context.Background(), pool, ""); err != nil || ix != nil {
		t.Fatalf("Prepare with an empty dir = (%v, %v), want (nil, nil)", ix, err)
	}

	// A configured but missing directory is a deployment error, not
	// a silent no-op.
	if _, _, err := Prepare(context.Background(), pool, filepath.Join(t.TempDir(), "missing")); err == nil {
		t.Fatal("Prepare over a missing directory succeeded, want an error")
	}

	ix, imported, err := Prepare(context.Background(), pool, writeFixtureGeoDir(t))
	if err != nil {
		t.Fatalf("Prepare: %v", err)
	}
	if !imported {
		t.Error("the first Prepare reported no import, want one")
	}
	if ix == nil {
		t.Fatal("Prepare answered no index, want one over the fixture dataset")
	}
	if ix.Len() != 4 {
		t.Fatalf("index Len() = %d, want 4 (the database's places rows)", ix.Len())
	}
	// The index resolves from the database rows: the Eiffel Tower
	// query lands on Paris.
	if id, _, ok := ix.NearestWithin(48.8584, 2.2945, ResolutionRadiusKm); !ok || id != 2988507 {
		t.Errorf("NearestWithin(Eiffel Tower) = (%d, %v), want 2988507", id, ok)
	}

	// The second boot skips the import (the marker matches) but
	// still answers a working index.
	ix, imported, err = Prepare(context.Background(), pool, writeFixtureGeoDir(t))
	if err != nil {
		t.Fatalf("second Prepare: %v", err)
	}
	if imported {
		t.Error("the second Prepare imported again, want the marker to skip the import")
	}
	if ix == nil || ix.Len() != 4 {
		t.Fatalf("second Prepare index = %v, want a 4-row index", ix)
	}
}
