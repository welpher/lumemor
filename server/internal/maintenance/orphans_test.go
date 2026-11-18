package maintenance

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/welpher/lumemor/server/internal/db"
	"github.com/welpher/lumemor/server/internal/db/dbtest"
	dbgen "github.com/welpher/lumemor/server/internal/db/gen"
)

// seedAsset inserts one asset row naming a stored file and records its
// state: the sweep's whole input is (the rows' file paths, the tree),
// so the fixture writes the rows directly -- no pipeline runs, and the
// media metadata is filler. trashed stamps deleted_at, which must NOT
// exempt the row's file from being referenced.
func seedAsset(t *testing.T, pool *pgxpool.Pool, owner uuid.UUID, filePath, checksum string, external, isComponent, trashed bool) {
	t.Helper()
	kind := "blob"
	if external {
		kind = "external"
	}
	var deletedAt any
	if trashed {
		deletedAt = time.Now()
	}
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO assets (
			id, owner_id, storage_kind, file_path, checksum, original_filename,
			file_size, mime_type, media_kind, is_component, width, height,
			taken_at, taken_at_source, pending_jobs, deleted_at
		) VALUES ($1, $2, '`+kind+`', $3, $4, $5, 1024, 'image/jpeg', 'image', $6, 0, 0,
			now(), 'import', 0, $7)`,
		uuid.Must(uuid.NewV7()), owner, filePath, checksum,
		filepath.Base(filePath), isComponent, deletedAt); err != nil {
		t.Fatalf("insert the %s fixture row %s: %v", kind, filePath, err)
	}
}

// TestSweepOrphanOriginals pins the three states the sweep separates:
// a file a blob row references stays (live, trashed and component rows
// alike), a file no row references and older than the window goes, an
// external row's source file is outside the managed tree and is never
// touched, and a young orphan is left for the next run.
func TestSweepOrphanOriginals(t *testing.T) {
	pool := dbtest.New(t)
	if err := db.Migrate(context.Background(), pool); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
	queries := dbgen.New(pool)
	dataDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dataDir, "originals", "owner"), 0o755); err != nil {
		t.Fatalf("create the originals tree: %v", err)
	}

	owner := uuid.Must(uuid.NewV7())
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO users (id, name, email, password_hash)
		VALUES ($1, 'Ada', 'ada@example.com', 'x')`, owner); err != nil {
		t.Fatalf("insert the owner: %v", err)
	}

	// The tree: the referenced files, one old orphan, one young
	// orphan. The external row's file lives outside the data root.
	liveRel := "originals/owner/live.jpg"
	trashedRel := "originals/owner/trashed.jpg"
	componentRel := "originals/owner/component.mov"
	orphanRel := "originals/owner/orphan.jpg"
	youngRel := "originals/owner/young.jpg"
	externalPath := filepath.Join(t.TempDir(), "from-a-folder.jpg")
	for _, rel := range []string{liveRel, trashedRel, componentRel, orphanRel, youngRel} {
		writeFixture(t, filepath.Join(dataDir, filepath.FromSlash(rel)))
	}
	writeFixture(t, externalPath)

	seedAsset(t, pool, owner, liveRel, "checksum-live", false, false, false)
	seedAsset(t, pool, owner, trashedRel, "checksum-trashed", false, false, true)
	seedAsset(t, pool, owner, componentRel, "checksum-component", false, true, false)
	seedAsset(t, pool, owner, externalPath, "checksum-external", true, false, false)

	// Age everything well past the window; the young orphan keeps its
	// fresh mtime.
	old := time.Now().Add(-2 * time.Hour)
	for _, rel := range []string{liveRel, trashedRel, componentRel, orphanRel} {
		ageFixture(t, filepath.Join(dataDir, filepath.FromSlash(rel)), old)
	}

	result, err := SweepOrphanOriginals(context.Background(), queries, dataDir, time.Hour, time.Now())
	if err != nil {
		t.Fatalf("sweep orphan originals: %v", err)
	}
	if result.Scanned != 5 {
		t.Errorf("scanned = %d, want the tree's 5 files", result.Scanned)
	}
	if result.Removed != 1 {
		t.Errorf("removed = %d, want the one old unreferenced file", result.Removed)
	}

	for _, rel := range []string{liveRel, trashedRel, componentRel, youngRel} {
		requirePresent(t, filepath.Join(dataDir, filepath.FromSlash(rel)), true)
	}
	requirePresent(t, filepath.Join(dataDir, filepath.FromSlash(orphanRel)), false)
	requirePresent(t, externalPath, true)

	// The second run is idempotent: nothing new to remove.
	result, err = SweepOrphanOriginals(context.Background(), queries, dataDir, time.Hour, time.Now())
	if err != nil {
		t.Fatalf("second sweep: %v", err)
	}
	if result.Removed != 0 {
		t.Errorf("second run removed = %d, want 0", result.Removed)
	}
}

// TestSweepOrphanOriginalsRefusesNoLister pins the caller-bug guards:
// an unwired lister or an empty data root must fail loudly rather than
// sweep a relative path.
func TestSweepOrphanOriginalsRefusesNoLister(t *testing.T) {
	if _, err := SweepOrphanOriginals(context.Background(), nil, t.TempDir(), time.Minute, time.Now()); err == nil {
		t.Error("sweep with no lister = nil error, want a refusal")
	}
	if _, err := SweepOrphanOriginals(context.Background(), staticPaths(nil), "", time.Minute, time.Now()); err == nil {
		t.Error("sweep with an empty data root = nil error, want a refusal")
	}
}

// staticPaths is a BlobPathLister over a fixed set.
type staticPaths []string

func (s staticPaths) ListBlobFilePaths(context.Context) ([]string, error) { return s, nil }

func writeFixture(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("create %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte("fixture bytes"), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func ageFixture(t *testing.T, path string, when time.Time) {
	t.Helper()
	if err := os.Chtimes(path, when, when); err != nil {
		t.Fatalf("age %s: %v", path, err)
	}
}

func requirePresent(t *testing.T, path string, want bool) {
	t.Helper()
	_, err := os.Stat(path)
	if want && err != nil {
		t.Fatalf("%s is missing, want it present: %v", path, err)
	}
	if !want && err == nil {
		t.Fatalf("%s exists, want it gone", path)
	}
}
