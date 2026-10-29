package db

import (
	"context"
	"io/fs"
	"slices"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/welpher/lumemor/server/internal/db/dbtest"
	"github.com/welpher/lumemor/server/migrations"
)

// wantTables is the complete v1 table set created by the migrations.
var wantTables = []string{
	// user domain
	"users",
	"sessions",
	"pairing_tokens",
	"api_keys",
	"instance_settings",
	// asset domain
	"assets",
	"exif",
	"places",
	"place_names",
	// album domain
	"albums",
	"album_items",
	"album_shares",
	"user_album_prefs",
	"user_favorites",
	"tags",
	"asset_tags",
	// intelligence domain
	"models",
	"faces",
	"face_clusters",
	"user_face_labels",
	"clip_embeddings",
	"ocr_texts",
	// jobs
	"job_runs",
}

func TestMigrate(t *testing.T) {
	pool := dbtest.New(t)
	ctx := t.Context()

	if err := Migrate(ctx, pool); err != nil {
		t.Fatalf("Migrate() error = %v, want nil", err)
	}

	t.Run("RecordsEveryMigration", func(t *testing.T) {
		var count int
		if err := pool.QueryRow(ctx, "SELECT count(*) FROM goose_db_version").Scan(&count); err != nil {
			t.Fatalf("count goose_db_version rows: %v", err)
		}
		if count == 0 {
			t.Fatal("goose_db_version has no rows, want at least one applied migration")
		}

		var maxVersion int64
		if err := pool.QueryRow(ctx, "SELECT coalesce(max(version_id), 0) FROM goose_db_version").Scan(&maxVersion); err != nil {
			t.Fatalf("max goose_db_version version_id: %v", err)
		}
		want := int64(len(embeddedMigrations(t)))
		if maxVersion != want {
			t.Errorf("max goose_db_version version_id = %d, want %d (one per embedded migration)", maxVersion, want)
		}
	})

	t.Run("CreatesAllTables", func(t *testing.T) {
		got, err := publicTables(ctx, pool)
		if err != nil {
			t.Fatalf("list public tables: %v", err)
		}

		// goose_db_version is the only extra table allowed in the public
		// schema; anything else means the schema drifted from this test.
		want := slices.Clone(wantTables)
		want = append(want, "goose_db_version")
		slices.Sort(want)
		if !slices.Equal(got, want) {
			t.Errorf("public tables = %v, want exactly %v", got, want)
		}
	})

	t.Run("Idempotent", func(t *testing.T) {
		if err := Migrate(ctx, pool); err != nil {
			t.Fatalf("second Migrate() error = %v, want nil", err)
		}
	})
}

// TestConnectTimeout verifies that Connect gives up after the configured
// wait when the database never becomes reachable, instead of hanging or
// failing fast.
func TestConnectTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	start := time.Now()
	_, err := Connect(ctx, "postgres://lumemor:lumemor@127.0.0.1:1/lumemor", 2*time.Second)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("Connect() error = nil, want an error for an unreachable database")
	}
	if elapsed < 1500*time.Millisecond {
		t.Errorf("Connect() gave up after %s, want it to keep retrying for about 2s", elapsed)
	}
}

// embeddedMigrations returns the *.sql files embedded in the migrations
// package, so the test tracks the migration set without manual updates.
func embeddedMigrations(t *testing.T) []string {
	t.Helper()
	names, err := fs.Glob(migrations.FS, "*.sql")
	if err != nil {
		t.Fatalf("glob embedded migrations: %v", err)
	}
	if len(names) == 0 {
		t.Fatal("no embedded migrations found")
	}
	return names
}

// publicTables returns the sorted base table names in the public schema.
func publicTables(ctx context.Context, pool *pgxpool.Pool) ([]string, error) {
	rows, err := pool.Query(ctx, `
		SELECT table_name
		FROM information_schema.tables
		WHERE table_schema = 'public' AND table_type = 'BASE TABLE'
		ORDER BY table_name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		tables = append(tables, name)
	}
	return tables, rows.Err()
}
