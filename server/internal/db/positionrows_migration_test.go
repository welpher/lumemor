package db

import (
	"testing"

	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"github.com/welpher/lumemor/server/internal/db/dbtest"
	"github.com/welpher/lumemor/server/migrations"
)

// TestPositionRowsMigrationDown pins the fork migration's reversibility:
// the position-row model lets one owner hold one content twice (an
// uploaded copy plus a directory source's position row), and the DOWN
// migration must therefore fold every such twin back into one row --
// re-linking the albums the dropped row fed -- before the previous
// total uniqueness can even exist. A down that only swapped indexes
// would fail on data the model legitimized, which is what this test
// exists to catch.
func TestPositionRowsMigrationDown(t *testing.T) {
	pool := dbtest.New(t)
	ctx := t.Context()
	if err := Migrate(ctx, pool); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}

	// One account, one sync album, one directory album.
	var owner string
	if err := pool.QueryRow(ctx, `
		INSERT INTO users (id, name, email, password_hash)
		VALUES (gen_random_uuid(), 'Fork', 'fork@migration.invalid', 'x')
		RETURNING id::text`).Scan(&owner); err != nil {
		t.Fatalf("insert the fixture owner: %v", err)
	}
	var syncAlbum, dirAlbum string
	if err := pool.QueryRow(ctx, `
		INSERT INTO albums (id, owner_id, title, type)
		VALUES (gen_random_uuid(), $1, 'Sync', 'sync')
		RETURNING id::text`, owner).Scan(&syncAlbum); err != nil {
		t.Fatalf("insert the sync album: %v", err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO albums (id, owner_id, title, type, root_path)
		VALUES (gen_random_uuid(), $1, 'Source', 'directory', '/srv/family')
		RETURNING id::text`, owner).Scan(&dirAlbum); err != nil {
		t.Fatalf("insert the directory album: %v", err)
	}

	// The dual home the fork makes representable: one uploaded blob row
	// and one source position row of the SAME content.
	var blobID, positionID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO assets (
			id, owner_id, storage_kind, file_path, checksum, original_filename,
			file_size, mime_type, media_kind, width, height, taken_at,
			taken_at_source, pending_jobs
		) VALUES (gen_random_uuid(), $1, 'blob', 'originals/a.jpg', 'sum-twin',
			'a.jpg', 10, 'image/jpeg', 'image', 0, 0, now(), 'import', 0)
		RETURNING id::text`, owner).Scan(&blobID); err != nil {
		t.Fatalf("insert the blob row: %v", err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO assets (
			id, owner_id, storage_kind, file_path, checksum, original_filename,
			file_size, mime_type, media_kind, width, height, taken_at,
			taken_at_source, pending_jobs
		) VALUES (gen_random_uuid(), $1, 'external', '/srv/family/a.jpg', 'sum-twin',
			'a.jpg', 10, 'image/jpeg', 'image', 0, 0, now(), 'file', 0)
		RETURNING id::text`, owner).Scan(&positionID); err != nil {
		t.Fatalf("insert the position row: %v", err)
	}
	if _, err := pool.Exec(ctx,
		"INSERT INTO album_items (album_id, asset_id, added_by) VALUES ($1, $2, $3), ($4, $5, $3)",
		syncAlbum, blobID, owner, dirAlbum, positionID); err != nil {
		t.Fatalf("insert the memberships: %v", err)
	}

	// The migration's own transaction boundary: the down runs, then the
	// up restores the model the test's rows were written under.
	sqlDB := stdlib.OpenDBFromPool(pool)
	defer func() { _ = sqlDB.Close() }()
	goose.SetBaseFS(migrations.FS)
	goose.SetLogger(goose.NopLogger())
	if err := goose.SetDialect("postgres"); err != nil {
		t.Fatalf("set goose dialect: %v", err)
	}
	if err := goose.DownContext(ctx, sqlDB, "."); err != nil {
		t.Fatalf("goose down: %v", err)
	}

	// The collapse: one row of the content (the blob, the survivor
	// rule), and the position row's album membership re-linked onto it.
	var rows int
	if err := pool.QueryRow(ctx,
		"SELECT count(*) FROM assets WHERE owner_id = $1 AND checksum = 'sum-twin'", owner).Scan(&rows); err != nil {
		t.Fatalf("count the surviving rows: %v", err)
	}
	if rows != 1 {
		t.Fatalf("rows of the content after the down = %d, want 1 (the twin folded away)", rows)
	}
	var survivor string
	if err := pool.QueryRow(ctx,
		"SELECT id::text FROM assets WHERE owner_id = $1 AND checksum = 'sum-twin'", owner).Scan(&survivor); err != nil {
		t.Fatalf("load the survivor: %v", err)
	}
	if survivor != blobID {
		t.Errorf("survivor = %s, want the blob row %s", survivor, blobID)
	}
	if err := pool.QueryRow(ctx,
		"SELECT count(*) FROM album_items WHERE asset_id = $1", blobID).Scan(&rows); err != nil {
		t.Fatalf("count the survivor's memberships: %v", err)
	}
	if rows != 2 {
		t.Errorf("the survivor's membership rows = %d, want 2 (the sync and the re-linked directory album)", rows)
	}

	// The up again: the fork returns, and the previous state is
	// representable once more (the migration pair is a round trip, not a
	// one-way door).
	if err := goose.UpContext(ctx, sqlDB, "."); err != nil {
		t.Fatalf("goose up after the down: %v", err)
	}
	var hasBlobIndex, hasPositionIndex bool
	if err := pool.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM pg_indexes WHERE indexname = 'assets_owner_checksum_blob_uq'),
		       EXISTS (SELECT 1 FROM pg_indexes WHERE indexname = 'assets_owner_file_path_external_uq')`).
		Scan(&hasBlobIndex, &hasPositionIndex); err != nil {
		t.Fatalf("inspect the indexes: %v", err)
	}
	if !hasBlobIndex || !hasPositionIndex {
		t.Errorf("indexes after the up = (blob %v, position %v), want both", hasBlobIndex, hasPositionIndex)
	}
	// And the predicate the fork introduced answers again.
	var representative bool
	if err := pool.QueryRow(ctx, "SELECT asset_library_representative($1::uuid)", blobID).Scan(&representative); err != nil {
		t.Fatalf("call the predicate after the up: %v", err)
	}
	if !representative {
		t.Error("asset_library_representative(blob) = false, want the lone live row to represent")
	}
}