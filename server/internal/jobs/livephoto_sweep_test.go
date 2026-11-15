package jobs

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// The pair maintenance's pins: an orphan (a component nothing
// references) goes with its bytes, a broken pair (a component whose
// file vanished) takes the still's link with it -- the still degrades to
// a plain photo -- a healthy pair is untouched, and a young row is
// never mistaken for an orphan (the extraction may be mid-flight).

// sweepFixture seeds one still, one motion row and their files, and
// answers their ids. `stillLinked` decides whether the still points at
// the motion; `withFile` decides whether the motion's bytes exist.
func sweepFixture(t *testing.T, e *env, stillLinked, withFile bool) (stillID, motionID uuid.UUID) {
	t.Helper()
	stillID = uuid.New()
	motionID = uuid.New()

	stillRel := "originals/sweep/" + stillID.String() + ".jpg"
	motionRel := "originals/sweep/" + stillID.String() + ".mp4"
	writeSweepFile(t, filepath.Join(e.dataDir, filepath.FromSlash(stillRel)), []byte("still"))
	if withFile {
		writeSweepFile(t, filepath.Join(e.dataDir, filepath.FromSlash(motionRel)), []byte("motion-bytes"))
	}

	if _, err := e.pool.Exec(context.Background(), `
		INSERT INTO assets (id, owner_id, storage_kind, file_path, checksum, original_filename,
		                    file_size, mime_type, media_kind, width, height, taken_at, taken_at_source)
		VALUES ($1, $2, 'blob', $3, $4, $5, 5, 'image/jpeg', 'image', 64, 64, now(), 'import')`,
		pgUUID(stillID), pgUUID(e.ownerID), stillRel, "still-"+stillID.String(), "still.jpg"); err != nil {
		t.Fatalf("insert the still: %v", err)
	}
	if _, err := e.pool.Exec(context.Background(), `
		INSERT INTO assets (id, owner_id, storage_kind, file_path, checksum, original_filename,
		                    file_size, mime_type, media_kind, width, height, is_component,
		                    taken_at, taken_at_source, created_at)
		VALUES ($1, $2, 'blob', $3, $4, 'motion.mp4', 12, 'video/mp4', 'video', 0, 0, true,
		        now(), 'import', now() - make_interval(hours => 2))`,
		pgUUID(motionID), pgUUID(e.ownerID), motionRel, "motion-"+motionID.String()); err != nil {
		t.Fatalf("insert the motion: %v", err)
	}
	if stillLinked {
		if _, err := e.pool.Exec(context.Background(),
			`UPDATE assets SET live_photo_video_id = $2 WHERE id = $1`,
			pgUUID(stillID), pgUUID(motionID)); err != nil {
			t.Fatalf("link the pair: %v", err)
		}
	}
	return stillID, motionID
}

// writeSweepFile lands one fixture file.
func writeSweepFile(t *testing.T, path string, content []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("create the fixture directory: %v", err)
	}
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("write the fixture file: %v", err)
	}
}

// rowCount answers one row's presence.
func rowCount(t *testing.T, e *env, id uuid.UUID) int {
	t.Helper()
	var count int
	if err := e.pool.QueryRow(context.Background(),
		`SELECT count(*) FROM assets WHERE id = $1`, pgUUID(id)).Scan(&count); err != nil {
		t.Fatalf("count the row: %v", err)
	}
	return count
}

// TestLivePhotoSweepRemovesOrphansAndDegradesBrokenPairs walks the
// three shapes in one run.
func TestLivePhotoSweepRemovesOrphansAndDegradesBrokenPairs(t *testing.T) {
	e := newEnv(t)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	// A healthy pair: untouched.
	healthyStill, healthyMotion := sweepFixture(t, e, true, true)
	// An orphan: a component no still references.
	_, orphan := sweepFixture(t, e, false, true)
	// A broken pair: the component's file is gone.
	brokenStill, brokenMotion := sweepFixture(t, e, true, false)
	// A young orphan: inside the grace window, so not yet judged.
	_, young := sweepFixture(t, e, false, true)
	if _, err := e.pool.Exec(context.Background(),
		`UPDATE assets SET created_at = now() WHERE id = $1`, pgUUID(young)); err != nil {
		t.Fatalf("re-age the young row: %v", err)
	}

	counts, err := RunLivePhotoSweep(context.Background(), e.pool, e.dataDir, e.bus, logger)
	if err != nil {
		t.Fatalf("run the sweep: %v", err)
	}
	if counts.Orphans != 1 || counts.Broken != 1 {
		t.Errorf("counts = %+v, want one orphan and one broken pair", counts)
	}

	// The healthy pair survives whole.
	if rowCount(t, e, healthyStill) != 1 || rowCount(t, e, healthyMotion) != 1 {
		t.Error("the healthy pair lost a row")
	}
	// The orphan's row and file are gone.
	if rowCount(t, e, orphan) != 0 {
		t.Error("the orphan's row survived")
	}
	var orphanPath string
	_ = e.pool.QueryRow(context.Background(),
		`SELECT file_path FROM assets WHERE id = $1`, pgUUID(orphan)).Scan(&orphanPath)
	if _, err := os.Stat(filepath.Join(e.dataDir, "originals/sweep", orphanPath)); err == nil && orphanPath != "" {
		t.Errorf("the orphan's file %q survived", orphanPath)
	}
	// The broken pair degrades: the component row is gone and the
	// still's link nulls out through the schema.
	if rowCount(t, e, brokenMotion) != 0 {
		t.Error("the broken component's row survived")
	}
	var brokenLink pgtype.UUID
	if err := e.pool.QueryRow(context.Background(),
		`SELECT live_photo_video_id FROM assets WHERE id = $1`, pgUUID(brokenStill)).Scan(&brokenLink); err != nil {
		t.Fatalf("read the degraded still: %v", err)
	}
	if brokenLink.Valid {
		t.Error("the degraded still still carries a motion link")
	}
	// A still that lost its motion announced the change: the walls
	// converge without a reload.
	payload := eventPayloadOf(t, e.bus, "asset.updated")
	if payload["assetId"] != brokenStill.String() {
		t.Errorf("asset.updated assetId = %q, want the degraded still %s", payload["assetId"], brokenStill)
	}
	// The young orphan waits for a later run.
	if rowCount(t, e, young) != 1 {
		t.Error("the young component was judged before the grace window closed")
	}
}
