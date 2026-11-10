// Expiry tests: the retention window (the default, a
// custom setting, corrupt and zero values), the shared-blob
// reference rule on the physical file, the batch bound of one run,
// the restore race guard, and the no-op over a clean database. The
// fixtures write real files and real rows; the deletion marks are
// backdated directly, because expiry is a deadline computation, never
// a sleep.
package trash

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/welpher/lumemor/server/internal/db"
	"github.com/welpher/lumemor/server/internal/db/dbtest"
	dbgen "github.com/welpher/lumemor/server/internal/db/gen"
	"github.com/welpher/lumemor/server/internal/events"
)

// env is the expiry environment: a migrated throwaway database and a
// fresh data root. No queue, no HTTP surface -- the expiry is the
// pure operation under test.
type env struct {
	pool    *pgxpool.Pool
	queries *dbgen.Queries
	dataDir string
	bus     *events.Bus
}

// newEnv provisions the environment.
func newEnv(t *testing.T) *env {
	t.Helper()
	pool := dbtest.New(t)
	if err := db.Migrate(t.Context(), pool); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
	bus := events.NewBus(5*time.Minute, time.Now)
	return &env{
		pool:    pool,
		queries: dbgen.New(pool),
		dataDir: t.TempDir(),
		bus:     bus,
	}
}

// insertAsset writes a real file under the data root's originals
// tree, inserts its row (trashed, deletion mark backdated by the
// given interval), and returns the id. The checksum is derived from
// the id so bulk fixtures never collide on the owner+checksum
// uniqueness.
func (e *env) insertAsset(t *testing.T, owner uuid.UUID, i int, backdate string) uuid.UUID {
	t.Helper()
	id := uuid.MustParse("00000000-0000-0000-0002-" + fixedHex12(i))
	rel := "originals/expiry/" + fixedHex12(i) + ".jpg"
	path := filepath.Join(e.dataDir, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("create fixture directory: %v", err)
	}
	if err := os.WriteFile(path, []byte("expiry fixture "+fixedHex12(i)), 0o644); err != nil {
		t.Fatalf("write fixture file: %v", err)
	}
	if _, err := e.pool.Exec(context.Background(), `
		INSERT INTO assets (
			id, owner_id, storage_kind, file_path, checksum, original_filename,
			file_size, mime_type, media_kind, width, height, taken_at,
			taken_at_source, deleted_at
		) VALUES (
			$1, $2, 'blob', $3, $4, $5, 1024, 'image/jpeg', 'image', 0, 0,
			now(), 'import', now() - `+backdate+`
		)`,
		id, owner, rel, "expiry-checksum-"+fixedHex12(i), "fixture-"+fixedHex12(i)+".jpg"); err != nil {
		t.Fatalf("insert fixture asset %d: %v", i, err)
	}
	return id
}

// insertUser seeds a plain user row and returns the id.
func (e *env) insertUser(t *testing.T, i int) uuid.UUID {
	t.Helper()
	id := uuid.MustParse("20000000-0000-0000-0000-" + fixedHex12(i))
	if _, err := e.pool.Exec(context.Background(),
		"INSERT INTO users (id, name, email, password_hash) VALUES ($1, $2, $3, 'x')",
		id, "expiry user "+fixedHex12(i), "expiry-"+fixedHex12(i)+"@example.com"); err != nil {
		t.Fatalf("insert fixture user %d: %v", i, err)
	}
	return id
}

// setRetention writes the instance's trash retention setting; an
// empty value removes the row (the default applies).
func (e *env) setRetention(t *testing.T, value string) {
	t.Helper()
	if value == "" {
		if _, err := e.pool.Exec(context.Background(),
			"DELETE FROM instance_settings WHERE key = 'trash_retention_days'"); err != nil {
			t.Fatalf("clear retention setting: %v", err)
		}
		return
	}
	if _, err := e.pool.Exec(context.Background(), `
		INSERT INTO instance_settings (key, value) VALUES ('trash_retention_days', $1::jsonb)
		ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, updated_at = now()`, value); err != nil {
		t.Fatalf("set retention setting %q: %v", value, err)
	}
}

// insertLivePhotoPair writes a trashed Live Photo pair: the photo
// row linked to its video component row, both deletion marks
// backdated by the given interval, real distinct files on disk, and
// returns both ids together with their stored file paths. The pair's
// shape mirrors the upload's: the link lives on the photo row, the
// flag on the component, the component's counter counts its
// transcode.
func (e *env) insertLivePhotoPair(t *testing.T, owner uuid.UUID, i int, backdate string) (photo, component uuid.UUID, photoRel, componentRel string) {
	t.Helper()
	photo = uuid.MustParse("00000000-0000-0000-0005-" + fixedHex12(i*2))
	component = uuid.MustParse("00000000-0000-0000-0005-" + fixedHex12(i*2+1))
	photoRel = "originals/expiry/pair-photo-" + fixedHex12(i*2) + ".jpg"
	componentRel = "originals/expiry/pair-video-" + fixedHex12(i*2+1) + ".mov"
	rows := []struct {
		id     uuid.UUID
		rel    string
		kind   string
		mime   string
		link   *uuid.UUID
		isComp bool
	}{
		// The component first: the photo's live_photo_video_id foreign
		// key must find its row already present.
		{id: component, rel: componentRel, kind: "video", mime: "video/quicktime", isComp: true},
		{id: photo, rel: photoRel, kind: "image", mime: "image/jpeg", link: &component},
	}
	for _, row := range rows {
		path := filepath.Join(e.dataDir, filepath.FromSlash(row.rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("create pair fixture directory: %v", err)
		}
		if err := os.WriteFile(path, []byte("expiry pair fixture "+row.rel), 0o644); err != nil {
			t.Fatalf("write pair fixture file: %v", err)
		}
		if _, err := e.pool.Exec(context.Background(), `
			INSERT INTO assets (
				id, owner_id, storage_kind, file_path, checksum, original_filename,
				file_size, mime_type, media_kind, live_photo_video_id, is_component,
				width, height, taken_at, taken_at_source, deleted_at
			) VALUES (
				$1, $2, 'blob', $3, $4, $5, 1024, $6, $7::media_kind, $8, $9,
				0, 0, now(), 'import', now() - `+backdate+`
			)`,
			row.id, owner, row.rel, "pair-checksum-"+fixedHex12(i)+row.rel, row.rel, row.mime, row.kind,
			row.link, row.isComp); err != nil {
			t.Fatalf("insert pair fixture row %s: %v", row.id, err)
		}
	}
	return photo, component, photoRel, componentRel
}

// run fires the expiry under test.
func (e *env) run(t *testing.T) ExpiryCounts {
	t.Helper()
	counts, err := RunExpiry(context.Background(), e.pool, e.dataDir, e.bus)
	if err != nil {
		t.Fatalf("run expiry: %v", err)
	}
	return counts
}

// rowExists reports whether the asset row is still present.
func (e *env) rowExists(t *testing.T, id uuid.UUID) bool {
	t.Helper()
	var found bool
	if err := e.pool.QueryRow(context.Background(),
		"SELECT EXISTS (SELECT 1 FROM assets WHERE id = $1)", id).Scan(&found); err != nil {
		t.Fatalf("probe asset row %s: %v", id, err)
	}
	return found
}

// assetDeletedEvents collects the asset.deleted payloads off the bus.
func (e *env) assetDeletedEvents(t *testing.T) []map[string]string {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	subscription := e.bus.Subscribe(ctx, 0)
	var payloads []map[string]string
	for {
		event, ok := subscription.Next()
		if !ok {
			return payloads
		}
		if event.Name != "asset.deleted" {
			continue
		}
		var payload map[string]string
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			t.Fatalf("decode asset.deleted payload %s: %v", event.Payload, err)
		}
		payloads = append(payloads, payload)
	}
}

// fixedHex12 renders i as a 12-digit hex string (the tail of a
// fixture uuid).
func fixedHex12(i int) string {
	const digits = "0123456789abcdef"
	out := make([]byte, 12)
	for pos := 11; pos >= 0; pos-- {
		out[pos] = digits[i&0xf]
		i >>= 4
	}
	return string(out)
}

// TestExpiryRetentionWindow pins the deadline computation against the
// instance setting: the default 30 days, a custom value, a corrupt
// value degrading to the default (the run must not fail over it),
// and zero meaning no retention at all.
func TestExpiryRetentionWindow(t *testing.T) {
	e := newEnv(t)
	owner := e.insertUser(t, 1)

	// The default: 29 days past the mark survives, 31 days goes.
	fresh := e.insertAsset(t, owner, 1, "make_interval(days => 29)")
	ancient := e.insertAsset(t, owner, 2, "make_interval(days => 31)")

	counts := e.run(t)
	if counts.Expired != 1 || counts.FilesRemoved != 1 {
		t.Fatalf("default retention counts = expired %d files %d, want 1 and 1", counts.Expired, counts.FilesRemoved)
	}
	if !e.rowExists(t, fresh) {
		t.Error("the 29-day-old row was expired, want it inside the default 30-day window")
	}
	if e.rowExists(t, ancient) {
		t.Error("the 31-day-old row survived, want it expired under the default retention")
	}

	// A custom value: one day. 23 hours past the mark survives, 25
	// hours goes. The default phase's 29-day-old survivor is now
	// outside the window and expires with it.
	e.setRetention(t, "1")
	hours23 := e.insertAsset(t, owner, 3, "make_interval(hours => 23)")
	hours25 := e.insertAsset(t, owner, 4, "make_interval(hours => 25)")
	counts = e.run(t)
	if counts.Expired != 2 {
		t.Fatalf("one-day retention counts = expired %d, want 2 (the 29-day-old and the 25-hour-old)", counts.Expired)
	}
	if !e.rowExists(t, hours23) {
		t.Error("the 23-hour-old row was expired, want it inside the one-day window")
	}
	if e.rowExists(t, hours25) {
		t.Error("the 25-hour-old row survived, want it expired under the one-day retention")
	}

	// A corrupt setting degrades to the default: the run completes,
	// and the deadline is the default's, not an error.
	e.setRetention(t, `"not a number"`)
	edge29 := e.insertAsset(t, owner, 5, "make_interval(days => 29)")
	edge31 := e.insertAsset(t, owner, 6, "make_interval(days => 31)")
	counts = e.run(t)
	if counts.Expired != 1 {
		t.Fatalf("corrupt setting counts = expired %d, want 1 (the default applies)", counts.Expired)
	}
	if !e.rowExists(t, edge29) || !e.rowExists(t, hours23) {
		t.Error("a row inside the default window was expired over the corrupt setting, want the default fallback")
	}
	if e.rowExists(t, edge31) {
		t.Error("the 31-day-old row survived over the corrupt setting, want the default's expiry")
	}

	// A negative value is corrupt the same way: the default applies.
	e.setRetention(t, "-5")
	if _, err := RunExpiry(context.Background(), e.pool, e.dataDir, nil); err != nil {
		t.Fatalf("run over a negative setting: %v", err)
	}
	if !e.rowExists(t, edge29) {
		t.Error("the negative setting was honored as a shorter window, want the default fallback")
	}

	// Zero means no retention: everything trashed is expired.
	e.setRetention(t, "0")
	justNow := e.insertAsset(t, owner, 7, "make_interval(mins => 0)")
	counts = e.run(t)
	if counts.Expired != 3 { // justNow, edge29 and hours23
		t.Fatalf("zero retention counts = expired %d, want the remaining 3 rows", counts.Expired)
	}
	if e.rowExists(t, justNow) || e.rowExists(t, edge29) || e.rowExists(t, hours23) {
		t.Error("rows survived a zero retention, want everything trashed expired")
	}
}

// TestExpirySharedBlobReference pins the shared-blob rule at the expiry
// level: two rows of two owners sharing one physical file, both
// expired -- the rows go, the file goes exactly once (the second
// row's count observes the first row's delete), and each purge
// announces itself.
func TestExpirySharedBlobReference(t *testing.T) {
	e := newEnv(t)
	ownerA := e.insertUser(t, 1)
	ownerB := e.insertUser(t, 2)

	// Two rows, one shared file: the second insert overwrites the
	// first's file with identical bytes (content addressing in one
	// line). The file exists exactly once.
	shared := "originals/expiry/shared/ab/cd/shared.jpg"
	path := filepath.Join(e.dataDir, filepath.FromSlash(shared))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("create shared fixture directory: %v", err)
	}
	if err := os.WriteFile(path, []byte("shared expiry fixture"), 0o644); err != nil {
		t.Fatalf("write shared fixture file: %v", err)
	}
	rowA := uuid.MustParse("00000000-0000-0000-0003-000000000001")
	rowB := uuid.MustParse("00000000-0000-0000-0003-000000000002")
	for _, row := range []struct {
		id    uuid.UUID
		owner uuid.UUID
	}{{rowA, ownerA}, {rowB, ownerB}} {
		if _, err := e.pool.Exec(context.Background(), `
			INSERT INTO assets (
				id, owner_id, storage_kind, file_path, checksum, original_filename,
				file_size, mime_type, media_kind, width, height, taken_at,
				taken_at_source, deleted_at
			) VALUES (
				$1, $2, 'blob', $3, 'shared-checksum', $4, 1024, 'image/jpeg', 'image', 0, 0,
				now(), 'import', now() - make_interval(days => 31)
			)`,
			row.id, row.owner, shared, "shared.jpg"); err != nil {
			t.Fatalf("insert shared row %s: %v", row.id, err)
		}
	}

	counts := e.run(t)
	if counts.Expired != 2 {
		t.Fatalf("counts.Expired = %d, want both rows", counts.Expired)
	}
	if counts.FilesRemoved != 1 {
		t.Fatalf("counts.FilesRemoved = %d, want the ONE shared file", counts.FilesRemoved)
	}
	if e.rowExists(t, rowA) || e.rowExists(t, rowB) {
		t.Error("a shared row survived, want both gone")
	}
	if _, err := os.Stat(path); err == nil {
		t.Error("the shared file survived the expiry of its last row, want it removed")
	}

	// Both purges announced themselves, each with id and kind.
	payloads := e.assetDeletedEvents(t)
	if len(payloads) != 2 {
		t.Fatalf("asset.deleted events = %d, want 2", len(payloads))
	}
	seen := map[string]bool{}
	for _, payload := range payloads {
		seen[payload["assetId"]] = true
		if payload["kind"] != "image" {
			t.Errorf("asset.deleted kind = %q, want image", payload["kind"])
		}
	}
	if !seen[rowA.String()] || !seen[rowB.String()] {
		t.Errorf("asset.deleted payloads = %v, want one per purged row", seen)
	}
}

// TestExpiryRestoredRowSurvives pins the other half of the mandated
// restore race: the row is backdated past the deadline but its mark
// is CLEARED before the run, and the guarded delete must skip it
// (its file untouched too).
func TestExpiryRestoredRowSurvives(t *testing.T) {
	e := newEnv(t)
	owner := e.insertUser(t, 1)
	id := e.insertAsset(t, owner, 1, "make_interval(days => 31)")
	file := filepath.Join(e.dataDir, "originals", "expiry", fixedHex12(1)+".jpg")

	// The restore: the mark clears between the backdating and the run.
	if _, err := e.pool.Exec(context.Background(),
		"UPDATE assets SET deleted_at = NULL WHERE id = $1", id); err != nil {
		t.Fatalf("restore the fixture row: %v", err)
	}

	counts := e.run(t)
	if counts.Expired != 0 || counts.FilesRemoved != 0 {
		t.Fatalf("counts = expired %d files %d, want the zero no-op over the restored row", counts.Expired, counts.FilesRemoved)
	}
	if !e.rowExists(t, id) {
		t.Fatal("the restored row was purged, want it alive")
	}
	if _, err := os.Stat(file); err != nil {
		t.Fatalf("the restored row's file was removed, want it in place: %v", err)
	}
	if events := e.assetDeletedEvents(t); len(events) != 0 {
		t.Errorf("asset.deleted events = %d, want none over the restored row", len(events))
	}
}

// TestExpiryBatchLimit pins the run's bound: 501 expired rows answer
// a run of 500, the remainder waits for the next run. The fixture
// rows' files do not exist on disk, which doubles as the pin that a
// missing file is a tolerated, logged condition, not a run failure.
func TestExpiryBatchLimit(t *testing.T) {
	e := newEnv(t)
	owner := e.insertUser(t, 1)

	// One statement, 501 rows, distinct checksums, one shared
	// nonexistent file path.
	values := make([]string, 0, 501)
	args := make([]any, 0, 501*2)
	for i := 0; i < 501; i++ {
		values = append(values, fmt.Sprintf("($%d, $%d, 'blob', 'originals/expiry/bulk.jpg', $%d, 'bulk.jpg', 1024, 'image/jpeg', 'image', 0, 0, now(), 'import', now() - make_interval(days => 31))",
			len(args)+1, len(args)+2, len(args)+3))
		args = append(args, uuid.MustParse("00000000-0000-0000-0004-"+fixedHex12(i)), owner, "bulk-checksum-"+fixedHex12(i))
	}
	if _, err := e.pool.Exec(context.Background(), `
		INSERT INTO assets (
			id, owner_id, storage_kind, file_path, checksum, original_filename,
			file_size, mime_type, media_kind, width, height, taken_at,
			taken_at_source, deleted_at
		) VALUES `+strings.Join(values, ", "), args...); err != nil {
		t.Fatalf("insert bulk expired rows: %v", err)
	}

	counts := e.run(t)
	if counts.Expired != 500 {
		t.Fatalf("counts.Expired = %d, want the batch-limited 500", counts.Expired)
	}

	// The one row past the bound survives for the next run.
	remaining := 0
	if err := e.pool.QueryRow(context.Background(),
		"SELECT count(*) FROM assets WHERE deleted_at IS NOT NULL").Scan(&remaining); err != nil {
		t.Fatalf("count remaining trashed rows: %v", err)
	}
	if remaining != 1 {
		t.Fatalf("remaining trashed rows = %d, want the 1 past the batch bound", remaining)
	}
}

// TestExpiryLivePhotoPairSilentComponent pins the expiry fix for
// Live Photo pairs: the component row is purged like any trashed row
// (its file handled under the same shared-blob rule), but its
// asset.deleted event is suppressed -- the trash view never listed
// components (the pair is represented by its photo), so announcing
// their expiry would be noise to every connected client. The photo
// row's own announcement is unchanged.
func TestExpiryLivePhotoPairSilentComponent(t *testing.T) {
	e := newEnv(t)
	owner := e.insertUser(t, 1)

	photo, component, photoRel, componentRel := e.insertLivePhotoPair(t, owner, 1, "make_interval(days => 31)")
	photoFile := filepath.Join(e.dataDir, filepath.FromSlash(photoRel))
	componentFile := filepath.Join(e.dataDir, filepath.FromSlash(componentRel))

	counts := e.run(t)
	if counts.Expired != 2 {
		t.Fatalf("counts.Expired = %d, want both rows of the pair purged", counts.Expired)
	}
	if counts.FilesRemoved != 2 {
		t.Fatalf("counts.FilesRemoved = %d, want the two distinct files removed", counts.FilesRemoved)
	}
	if e.rowExists(t, photo) || e.rowExists(t, component) {
		t.Error("a row of the pair survived, want both purged (only the component's event is suppressed)")
	}
	if _, err := os.Stat(photoFile); err == nil {
		t.Error("the photo's file survived the expiry, want it removed")
	}
	if _, err := os.Stat(componentFile); err == nil {
		t.Error("the component's file survived the expiry, want it removed")
	}

	// Exactly one announcement, the photo's: the component never
	// appears in the trash view, so its purge stays silent.
	payloads := e.assetDeletedEvents(t)
	if len(payloads) != 1 {
		t.Fatalf("asset.deleted events = %d, want exactly 1 (the photo; the component stays silent)", len(payloads))
	}
	if payloads[0]["assetId"] != photo.String() {
		t.Errorf("asset.deleted assetId = %q, want the photo %s", payloads[0]["assetId"], photo)
	}
	if payloads[0]["kind"] != "image" {
		t.Errorf("asset.deleted kind = %q, want image", payloads[0]["kind"])
	}
}

// TestExpiryCleanDatabaseIsNoop pins the no-op: nothing trashed,
// nothing done, no error, no events.
func TestExpiryCleanDatabaseIsNoop(t *testing.T) {
	e := newEnv(t)
	owner := e.insertUser(t, 1)
	live := e.insertAsset(t, owner, 1, "make_interval(mins => 0)")
	if _, err := e.pool.Exec(context.Background(),
		"UPDATE assets SET deleted_at = NULL WHERE id = $1", live); err != nil {
		t.Fatalf("un-trash the fixture row: %v", err)
	}

	counts := e.run(t)
	if counts.Expired != 0 || counts.FilesRemoved != 0 {
		t.Fatalf("counts = expired %d files %d, want the zero no-op", counts.Expired, counts.FilesRemoved)
	}
	if !e.rowExists(t, live) {
		t.Error("the live row was touched, want it untouched")
	}
	if events := e.assetDeletedEvents(t); len(events) != 0 {
		t.Errorf("asset.deleted events = %d, want none", len(events))
	}
}
