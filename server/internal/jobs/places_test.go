// Places resolution tests: the metadata job's GPS step
// and the admin re-resolution run, both against a fixture grid. The
// determinism is pinned three ways
// here: the same extraction resolves to the same place on a re-run,
// the re-resolution walk converges to the same answers the metadata
// step wrote (one deterministic function of the coordinates), and a
// walk over stale resolutions MOVES them to the grid's current
// answers -- the places-data-update path's whole point.
package jobs

import (
	"context"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/riverqueue/river"

	dbgen "github.com/welpher/lumemor/server/internal/db/gen"
	"github.com/welpher/lumemor/server/internal/geodata"
)

// The fixture grid's place ids, matching the seeded places rows.
const (
	placeParisFixtureID   = 2988507
	placeBeijingFixtureID = 1816670
)

// placesFixtureGrid builds the grid: Paris and Beijing centroids.
func placesFixtureGrid() *geodata.Index {
	return geodata.NewIndex([]geodata.Centroid{
		{ID: placeParisFixtureID, Lat: 48.85341, Lon: 2.3488},
		{ID: placeBeijingFixtureID, Lat: 39.9075, Lon: 116.39723},
	})
}

// seedPlacesRows seeds the places table rows the grid's ids address
// (the assets.place_id foreign key needs them).
func (e *env) seedPlacesRows(t *testing.T) {
	t.Helper()
	e.exec(t, `
		INSERT INTO places (id, name, country_code, admin1_code, timezone, lat, lon) VALUES
			(2988507, 'Paris', 'FR', 'FR.11', 'Europe/Paris', 48.85341, 2.3488),
			(1816670, 'Beijing', 'CN', 'CN.22', 'Asia/Shanghai', 39.9075, 116.39723)`)
}

// placeOf answers the asset's current place_id (nil when NULL).
func (e *env) placeOf(t *testing.T, id uuid.UUID) *int32 {
	t.Helper()
	var placeID *int32
	if err := e.pool.QueryRow(context.Background(),
		"SELECT place_id FROM assets WHERE id = $1", id).Scan(&placeID); err != nil {
		t.Fatalf("scan place_id: %v", err)
	}
	return placeID
}

// exec runs one statement against the test database.
func (e *env) exec(t *testing.T, statement string, args ...any) {
	t.Helper()
	if _, err := e.pool.Exec(context.Background(), statement, args...); err != nil {
		t.Fatalf("exec %q: %v", statement, err)
	}
}

// requirePlaceID asserts the asset's place_id.
func requirePlaceID(t *testing.T, e *env, id uuid.UUID, want *int32, what string) {
	t.Helper()
	got := e.placeOf(t, id)
	if (got == nil) != (want == nil) || (got != nil && *got != *want) {
		t.Fatalf("%s: place_id = %v, want %v", what, got, want)
	}
}

// setExifGPS writes the exif row's GPS coordinates directly (the
// re-resolution walk reads the database, not files).
func (e *env) setExifGPS(t *testing.T, id uuid.UUID, lat, lon float64) {
	t.Helper()
	e.exec(t, "UPDATE exif SET lat = $1, lon = $2 WHERE asset_id = $3", lat, lon, id)
}

// writeGPSFixture copies the plain JPEG fixture and stamps GPS
// coordinates into its EXIF with exiftool, answering the file's
// bytes. The decimal form lands as proper EXIF rationals; the
// metadata job's -n extraction reads it back as the same decimal.
func writeGPSFixture(t *testing.T, lat, lon string) []byte {
	t.Helper()
	requireExiftool(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "gps.jpg")
	if err := os.WriteFile(path, jpegBytes(t), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	cmd := exec.Command(exiftoolBinary, "-overwrite_original", "-q",
		"-GPSLatitude="+lat, "-GPSLongitude="+lon, path)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("stamp GPS EXIF (%s %s): %v: %s", lat, lon, err, out)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read stamped fixture: %v", err)
	}
	return data
}

// TestMetadataJobResolvesPlace pins the GPS step's happy path and its
// determinism: the golden fixture's Eiffel Tower coordinates resolve
// to Paris (~0.9 km) on the first run, and a re-run of the same
// extraction answers the same place -- the same function of the same
// coordinates, nothing order- or time-dependent in between.
func TestMetadataJobResolvesPlace(t *testing.T) {
	requireExiftool(t)
	e := newEnvPlaces(t, placesFixtureGrid())
	e.seedPlacesRows(t)

	golden := goldenFixture(t)
	deviceTakenAt := time.Date(2020, 3, 1, 12, 0, 0, 0, time.UTC).Truncate(time.Microsecond)
	placeholder := `{"deviceCreatedAt":"2020-03-01T12:00:00Z","deviceTimeZone":"Europe/Paris"}`
	id := e.insertAsset(t, "originals/golden/golden.jpg", golden, deviceTakenAt, placeholder)

	if err := e.queue.EnqueueMetadata(context.Background(), id); err != nil {
		t.Fatalf("enqueue metadata job: %v", err)
	}
	eventually(t, jobWaitTimeout, "the golden asset to become ready", func() bool {
		return e.assetOf(t, id).readyAt.Valid
	})
	requirePlaceID(t, e, id, ptrInt32(placeParisFixtureID), "the first resolution")

	// The re-run converges to the same place (idempotent metadata
	// discipline: no churn, no second event).
	if err := e.queue.EnqueueMetadata(context.Background(), id); err != nil {
		t.Fatalf("re-enqueue metadata job: %v", err)
	}
	eventually(t, jobWaitTimeout, "the re-run to complete", func() bool {
		return countEvents(e.bus, "asset.ready") == 1
	})
	requirePlaceID(t, e, id, ptrInt32(placeParisFixtureID), "the re-run's resolution")
}

// TestMetadataJobPlaceBeyondThresholdIsNull pins the hard
// edge: GPS coordinates with no grid city within 25 km resolve to
// NULL -- no fallback, no widening, the honest absence.
func TestMetadataJobPlaceBeyondThresholdIsNull(t *testing.T) {
	requireExiftool(t)
	e := newEnvPlaces(t, placesFixtureGrid())
	e.seedPlacesRows(t)

	// (47.0, 2.0) sits ~207 km from Paris and ~8,000 km from Beijing:
	// far beyond the threshold on the fixture grid.
	content := writeGPSFixture(t, "47.0", "2.0")
	deviceTakenAt := time.Date(2020, 3, 1, 12, 0, 0, 0, time.UTC).Truncate(time.Microsecond)
	placeholder := `{"deviceCreatedAt":"2020-03-01T12:00:00Z","deviceTimeZone":"Europe/Paris"}`
	id := e.insertAsset(t, "originals/gps/beyond.jpg", content, deviceTakenAt, placeholder)

	if err := e.queue.EnqueueMetadata(context.Background(), id); err != nil {
		t.Fatalf("enqueue metadata job: %v", err)
	}
	eventually(t, jobWaitTimeout, "the beyond-threshold asset to become ready", func() bool {
		return e.assetOf(t, id).readyAt.Valid
	})
	requirePlaceID(t, e, id, nil, "the beyond-threshold resolution")

	// The coordinates themselves landed in the exif row (the
	// resolution's input, exactly as extracted).
	row, ok := e.exifOf(t, id)
	if !ok {
		t.Fatal("exif row missing after the job ran")
	}
	if row.lat == nil || *row.lat != 47.0 || row.lon == nil || *row.lon != 2.0 {
		t.Errorf("exif GPS = (%v, %v), want (47, 2)", row.lat, row.lon)
	}
}

// TestMetadataJobWithoutGridKeepsPlace pins the feature-off wiring:
// a process without the place data leaves assets.place_id exactly
// as it found it -- the metadata refinement never destroys a
// resolution a data-carrying boot produced.
func TestMetadataJobWithoutGridKeepsPlace(t *testing.T) {
	requireExiftool(t)
	e := newEnv(t) // no place grid wired
	e.seedPlacesRows(t)

	golden := goldenFixture(t)
	deviceTakenAt := time.Date(2020, 3, 1, 12, 0, 0, 0, time.UTC).Truncate(time.Microsecond)
	placeholder := `{"deviceCreatedAt":"2020-03-01T12:00:00Z","deviceTimeZone":"Europe/Paris"}`
	id := e.insertAsset(t, "originals/golden/golden.jpg", golden, deviceTakenAt, placeholder)
	// A resolution from a previous, data-carrying boot.
	e.exec(t, "UPDATE assets SET place_id = $1 WHERE id = $2", placeBeijingFixtureID, id)

	if err := e.queue.EnqueueMetadata(context.Background(), id); err != nil {
		t.Fatalf("enqueue metadata job: %v", err)
	}
	eventually(t, jobWaitTimeout, "the gridless asset to become ready", func() bool {
		return e.assetOf(t, id).readyAt.Valid
	})
	requirePlaceID(t, e, id, ptrInt32(placeBeijingFixtureID), "the gridless boot")
}

// TestPlacesRerunWalk pins the re-resolution run: the enqueue's scope
// count (GPS-bearing assets only), the walk's answers (the same
// nearest-city-within-threshold function the metadata step applies --
// including MOVING a stale resolution to the grid's current answer,
// the places-data-update path's whole point), the run row's
// bookkeeping, and the refusal of a queue without the grid.
func TestPlacesRerunWalk(t *testing.T) {
	e := newEnvPlaces(t, placesFixtureGrid())
	e.seedPlacesRows(t)

	takenAt := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC).Truncate(time.Microsecond)
	// a1: GPS near the Eiffel Tower -- resolves Paris, whatever it
	// carried before (the stale resolution moves: the data-update
	// arm).
	a1 := e.insertAsset(t, "originals/places/a1.jpg", jpegBytes(t), takenAt,
		`{"deviceCreatedAt":"2026-07-01T12:00:00Z"}`)
	e.setExifGPS(t, a1, 48.8584, 2.2945)
	e.exec(t, "UPDATE assets SET place_id = $1 WHERE id = $2", placeBeijingFixtureID, a1)
	// a2: GPS far beyond the threshold -- resolves NULL, even from a
	// stale non-NULL resolution.
	a2 := e.insertAsset(t, "originals/places/a2.jpg", jpegBytes(t), takenAt,
		`{"deviceCreatedAt":"2026-07-01T12:00:00Z"}`)
	e.setExifGPS(t, a2, 47.0, 2.0)
	e.exec(t, "UPDATE assets SET place_id = $1 WHERE id = $2", placeParisFixtureID, a2)
	// a3: no GPS at all -- outside the walk's scope, its resolution
	// stays whatever it was.
	a3 := e.insertAsset(t, "originals/places/a3.jpg", jpegBytes(t), takenAt,
		`{"deviceCreatedAt":"2026-07-01T12:00:00Z"}`)
	e.exec(t, "UPDATE assets SET place_id = $1 WHERE id = $2", placeBeijingFixtureID, a3)

	runID, err := e.queue.EnqueuePlacesRerun(context.Background(), PlacesRerunModeAll)
	if err != nil {
		t.Fatalf("enqueue the places rerun: %v", err)
	}
	eventually(t, jobWaitTimeout, "the places rerun run to finish", func() bool {
		row, err := e.queries.GetJobRun(context.Background(), pgUUID(runID))
		return err == nil && (row.State == dbgen.JobStateSucceeded || row.State == dbgen.JobStateFailed)
	})

	row, err := e.queries.GetJobRun(context.Background(), pgUUID(runID))
	if err != nil {
		t.Fatalf("load the run row: %v", err)
	}
	if row.State != dbgen.JobStateSucceeded {
		t.Fatalf("run state = %s, want succeeded (done %d, failed %d)", row.State, row.Done, row.Failed)
	}
	if row.Total != 2 {
		t.Errorf("run total = %d, want 2 (the GPS-bearing assets only)", row.Total)
	}
	if row.Done != 2 || row.Failed != 0 {
		t.Errorf("run done/failed = %d/%d, want 2/0", row.Done, row.Failed)
	}
	if row.Done != row.Total {
		t.Errorf("run done/total = %d/%d, want equal (the count is the walk's own scope)", row.Done, row.Total)
	}
	if !row.StartedAt.Valid || !row.FinishedAt.Valid {
		t.Errorf("run started/finished = %v/%v, want both set", row.StartedAt, row.FinishedAt)
	}
	if row.Kind != "places_rerun" {
		t.Errorf("run kind = %q, want places_rerun", row.Kind)
	}

	requirePlaceID(t, e, a1, ptrInt32(placeParisFixtureID), "the stale Beijing resolution moving to Paris")
	requirePlaceID(t, e, a2, nil, "the beyond-threshold resolution releasing to NULL")
	requirePlaceID(t, e, a3, ptrInt32(placeBeijingFixtureID), "the no-GPS asset outside the walk's scope")

	// The finished event announced the run with its converged counts.
	payload := eventPayloadOf(t, e.bus, "job.finished")
	if payload["jobRunId"] != runID.String() {
		t.Errorf("job.finished jobRunId = %q, want %s", payload["jobRunId"], runID)
	}
	if payload["kind"] != "places_rerun" {
		t.Errorf("job.finished kind = %q, want places_rerun", payload["kind"])
	}

	// A second run over the converged state is a no-op walk: the same
	// answers re-write (a1 Paris, a2 NULL), the counts stay honest.
	second, err := e.queue.EnqueuePlacesRerun(context.Background(), PlacesRerunModeAll)
	if err != nil {
		t.Fatalf("enqueue the second places rerun: %v", err)
	}
	eventually(t, jobWaitTimeout, "the second run to finish", func() bool {
		row, err := e.queries.GetJobRun(context.Background(), pgUUID(second))
		return err == nil && (row.State == dbgen.JobStateSucceeded || row.State == dbgen.JobStateFailed)
	})
	requirePlaceID(t, e, a1, ptrInt32(placeParisFixtureID), "the second walk's a1")
	requirePlaceID(t, e, a2, nil, "the second walk's a2")

	// A queue without the grid refuses the enqueue: there is no grid
	// to re-resolve against, and the endpoint maps the refusal to the
	// same 500 the broken-transaction arm answers.
	bare := &Queue{pool: e.pool}
	if _, err := bare.EnqueuePlacesRerun(context.Background(), PlacesRerunModeAll); err == nil {
		t.Fatal("EnqueuePlacesRerun without a place grid succeeded, want the refusal")
	}
}

// TestPlacesRerunMissingMode pins the fill-in mode: its scope is the
// GPS-bearing rows whose place is absent (a resolved row is never
// rewritten, a row beyond the threshold stays NULL), its total is
// that scope's count, and the full mode runs afterwards move the
// resolved row again -- the two modes are the dialog's honest pair,
// and the mode rides the job payload.
func TestPlacesRerunMissingMode(t *testing.T) {
	e := newEnvPlaces(t, placesFixtureGrid())
	e.seedPlacesRows(t)

	takenAt := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC).Truncate(time.Microsecond)
	// a1: GPS near Paris, no place -- the fill-in's whole point.
	a1 := e.insertAsset(t, "originals/places/m1.jpg", jpegBytes(t), takenAt,
		`{"deviceCreatedAt":"2026-07-01T12:00:00Z"}`)
	e.setExifGPS(t, a1, 48.8584, 2.2945)
	// a2: GPS near Beijing but resolved to Paris -- a stale answer the
	// fill-in mode must leave untouched (it is not missing).
	a2 := e.insertAsset(t, "originals/places/m2.jpg", jpegBytes(t), takenAt,
		`{"deviceCreatedAt":"2026-07-01T12:00:00Z"}`)
	e.setExifGPS(t, a2, 39.9075, 116.39723)
	e.exec(t, "UPDATE assets SET place_id = $1 WHERE id = $2", placeParisFixtureID, a2)
	// a3: GPS beyond the threshold, no place -- walked and left NULL.
	a3 := e.insertAsset(t, "originals/places/m3.jpg", jpegBytes(t), takenAt,
		`{"deviceCreatedAt":"2026-07-01T12:00:00Z"}`)
	e.setExifGPS(t, a3, 47.0, 2.0)
	// a4: no GPS at all -- outside every mode's scope.
	a4 := e.insertAsset(t, "originals/places/m4.jpg", jpegBytes(t), takenAt,
		`{"deviceCreatedAt":"2026-07-01T12:00:00Z"}`)
	// a5: GPS, no place, but the owner located it by hand -- outside
	// BOTH modes (the manual decision owns the row), so it is neither
	// counted nor walked and the run converges to done = total.
	a5 := e.insertAsset(t, "originals/places/m5.jpg", jpegBytes(t), takenAt,
		`{"deviceCreatedAt":"2026-07-01T12:00:00Z"}`)
	e.setExifGPS(t, a5, 39.9075, 116.39723)
	e.exec(t, "UPDATE assets SET location_lat = 48.855, location_lon = 2.35 WHERE id = $1", a5)

	runID, err := e.queue.EnqueuePlacesRerun(context.Background(), PlacesRerunModeMissing)
	if err != nil {
		t.Fatalf("enqueue the missing places rerun: %v", err)
	}
	eventually(t, jobWaitTimeout, "the missing places rerun to finish", func() bool {
		row, err := e.queries.GetJobRun(context.Background(), pgUUID(runID))
		return err == nil && (row.State == dbgen.JobStateSucceeded || row.State == dbgen.JobStateFailed)
	})

	row, err := e.queries.GetJobRun(context.Background(), pgUUID(runID))
	if err != nil {
		t.Fatalf("load the run row: %v", err)
	}
	if row.State != dbgen.JobStateSucceeded {
		t.Fatalf("run state = %s, want succeeded (done %d, failed %d)", row.State, row.Done, row.Failed)
	}
	if row.Total != 2 {
		t.Errorf("run total = %d, want 2 (the GPS rows without a place)", row.Total)
	}
	if row.Done != 2 || row.Failed != 0 {
		t.Errorf("run done/failed = %d/%d, want 2/0", row.Done, row.Failed)
	}
	if row.Done != row.Total {
		t.Errorf("run done/total = %d/%d, want equal (the count is the walk's own scope)", row.Done, row.Total)
	}

	requirePlaceID(t, e, a1, ptrInt32(placeParisFixtureID), "the missing scope filling a1")
	requirePlaceID(t, e, a2, ptrInt32(placeParisFixtureID), "the stale resolution the fill-in leaves alone")
	requirePlaceID(t, e, a3, nil, "the beyond-threshold row staying NULL")
	requirePlaceID(t, e, a4, nil, "the no-GPS row staying untouched")
	requirePlaceID(t, e, a5, nil, "the manual-location row the fill-in never visits")

	// The mode rode the job payload: the worker reads the scope from
	// it, so a retry re-walks the same one.
	var mode string
	if err := e.pool.QueryRow(context.Background(),
		`SELECT args->>'mode' FROM river_job WHERE args->>'jobRunId' = $1`, runID.String()).Scan(&mode); err != nil {
		t.Fatalf("read the job payload's mode: %v", err)
	}
	if mode != PlacesRerunModeMissing {
		t.Errorf("job payload mode = %q, want %q", mode, PlacesRerunModeMissing)
	}

	// The full mode walks every GPS row: a2 moves to its grid answer,
	// the fill-in's work converges, and the total is the GPS count.
	full, err := e.queue.EnqueuePlacesRerun(context.Background(), PlacesRerunModeAll)
	if err != nil {
		t.Fatalf("enqueue the full places rerun: %v", err)
	}
	eventually(t, jobWaitTimeout, "the full places rerun to finish", func() bool {
		row, err := e.queries.GetJobRun(context.Background(), pgUUID(full))
		return err == nil && (row.State == dbgen.JobStateSucceeded || row.State == dbgen.JobStateFailed)
	})
	fullRow, err := e.queries.GetJobRun(context.Background(), pgUUID(full))
	if err != nil {
		t.Fatalf("load the full run row: %v", err)
	}
	if fullRow.Total != 3 || fullRow.Done != 3 {
		t.Errorf("full run total/done = %d/%d, want 3/3 (every GPS row the walk owns)", fullRow.Total, fullRow.Done)
	}
	requirePlaceID(t, e, a2, ptrInt32(placeBeijingFixtureID), "the full mode moving the stale resolution")
	requirePlaceID(t, e, a1, ptrInt32(placeParisFixtureID), "the full mode's a1")
	requirePlaceID(t, e, a5, nil, "the manual-location row the full mode never visits")
	// The manual location itself is untouched: the walk rewrote
	// nothing about that row.
	var manualLat float64
	if err := e.pool.QueryRow(context.Background(),
		"SELECT location_lat FROM assets WHERE id = $1", a5).Scan(&manualLat); err != nil {
		t.Fatalf("read the manual location: %v", err)
	}
	if manualLat != 48.855 {
		t.Errorf("manual location = %v, want 48.855 (the walk leaves it alone)", manualLat)
	}

	// A mode outside the enum is refused before any row lands.
	if _, err := e.queue.EnqueuePlacesRerun(context.Background(), "everything"); err == nil {
		t.Fatal("EnqueuePlacesRerun with a mode outside the enum succeeded, want the refusal")
	}
	// A queue without the grid refuses the enqueue in either mode.
	bare := &Queue{pool: e.pool}
	if _, err := bare.EnqueuePlacesRerun(context.Background(), PlacesRerunModeMissing); err == nil {
		t.Fatal("EnqueuePlacesRerun without a place grid succeeded, want the refusal")
	}
}

// TestPlacesRerunEmptyModeWalksAll pins the durable queue's upgrade
// path: a payload queued before the mode field existed (no `mode`
// key) walks the full scope, the contract's absent-mode default.
func TestPlacesRerunEmptyModeWalksAll(t *testing.T) {
	e := newEnvPlaces(t, placesFixtureGrid())
	e.seedPlacesRows(t)

	takenAt := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC).Truncate(time.Microsecond)
	a1 := e.insertAsset(t, "originals/places/o1.jpg", jpegBytes(t), takenAt,
		`{"deviceCreatedAt":"2026-07-01T12:00:00Z"}`)
	e.setExifGPS(t, a1, 48.8584, 2.2945)
	e.exec(t, "UPDATE assets SET place_id = $1 WHERE id = $2", placeBeijingFixtureID, a1)

	runID, err := uuid.NewV7()
	if err != nil {
		t.Fatalf("generate the run id: %v", err)
	}
	if err := e.queries.InsertJobRun(context.Background(), dbgen.InsertJobRunParams{
		ID:    pgUUID(runID),
		Kind:  PlacesRerunKind,
		Total: 1,
	}); err != nil {
		t.Fatalf("seed the run row: %v", err)
	}
	worker := &PlacesRerunWorker{
		pool:   e.pool,
		bus:    e.bus,
		places: placesFixtureGrid(),
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
	if err := worker.Work(context.Background(), &river.Job[PlacesRerunArgs]{
		Args: PlacesRerunArgs{JobRunID: runID},
	}); err != nil {
		t.Fatalf("run the empty-mode walk: %v", err)
	}

	row, err := e.queries.GetJobRun(context.Background(), pgUUID(runID))
	if err != nil {
		t.Fatalf("load the run row: %v", err)
	}
	if row.State != dbgen.JobStateSucceeded || row.Done != 1 {
		t.Errorf("run state/done = %s/%d, want succeeded/1", row.State, row.Done)
	}
	requirePlaceID(t, e, a1, ptrInt32(placeParisFixtureID), "the empty mode's full walk")
}

// ptrInt32 answers a pointer to the value (test-fixture ergonomics).
func ptrInt32(v int32) *int32 {
	return &v
}
