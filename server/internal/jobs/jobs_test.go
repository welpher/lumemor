package jobs

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"image"
	"image/color"
	"image/jpeg"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	oapitypes "github.com/oapi-codegen/runtime/types"
	"github.com/riverqueue/river"

	"github.com/welpher/lumemor/server/internal/api"
	apigen "github.com/welpher/lumemor/server/internal/api/gen"
	"github.com/welpher/lumemor/server/internal/auth"
	"github.com/welpher/lumemor/server/internal/db"
	"github.com/welpher/lumemor/server/internal/db/dbtest"
	dbgen "github.com/welpher/lumemor/server/internal/db/gen"
	"github.com/welpher/lumemor/server/internal/events"
	"github.com/welpher/lumemor/server/internal/media"
	"github.com/welpher/lumemor/server/internal/ml"
	"github.com/welpher/lumemor/server/internal/storage"
)

// jobWaitTimeout bounds every wait for asynchronous work: River picks
// up a job within its fetch poll interval (1s) and the metadata job
// itself runs exiftool in well under a second on the fixtures, so the
// bound only absorbs slow container starts.
const jobWaitTimeout = 30 * time.Second

const (
	// adminEmail and adminPassword are the credentials of the first
	// administrator every test environment creates through the wizard.
	adminEmail    = "ada@example.com"
	adminPassword = "correct horse battery staple"

	// sessionCookieName is the web session cookie of the login flow.
	sessionCookieName = "lumemor_session"
)

// requireExiftool skips the test when the real exiftool is not on the
// host: the extraction tests are exercised by `make docker-test`, the
// dev container carrying the toolchain.
func requireExiftool(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath(exiftoolBinary); err != nil {
		t.Skipf("exiftool is not installed on this host; run the suite inside the dev container (make docker-test): %v", err)
	}
}

// env is the full pipeline environment: a migrated throwaway database,
// a fresh storage layout, a fresh models dir behind a real model
// registry, a live event bus, a started one-worker queue, and the HTTP
// surface of the upload endpoint wired to it.
type env struct {
	pool      *pgxpool.Pool
	queries   *dbgen.Queries
	bus       *events.Bus
	dataDir   string
	modelsDir string
	registry  *ml.Registry
	queue     *Queue
	router    http.Handler
	cookie    *http.Cookie
	ownerID   uuid.UUID
}

// newEnv provisions the environment and signs its administrator in.
// River logs chatty service diagnostics; those go to io.Discard so
// the test output stays on the assertions.
func newEnv(t *testing.T) *env {
	t.Helper()
	return newEnvManifest(t, nil)
}

// newEnvManifest is newEnv with the model download job's manifest
// overridden: the model download tests point the group at their local
// httptest sources; every other test passes nil and keeps the embedded
// default group, which those tests never download.
func newEnvManifest(t *testing.T, manifest *ml.Manifest) *env {
	t.Helper()
	return newEnvLogger(t, manifest, slog.New(slog.NewTextHandler(io.Discard, nil)))
}

// newEnvLogger is newEnv with the manifest and the queue's logger
// injected: the sweep tests capture the queue's log output to pin the
// one-log-line-per-run discipline of the requeue sweep.
func newEnvLogger(t *testing.T, manifest *ml.Manifest, logger *slog.Logger) *env {
	t.Helper()

	pool := dbtest.New(t)
	if err := db.Migrate(t.Context(), pool); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}

	dataDir := t.TempDir()
	if err := storage.EnsureLayout(dataDir, t.TempDir()); err != nil {
		t.Fatalf("prepare storage layout: %v", err)
	}

	modelsDir := t.TempDir()
	registry := ml.NewRegistry(pool, dbgen.New(pool), modelsDir)

	bus := events.NewBus(5*time.Minute, time.Now)
	queue, err := Start(t.Context(), Config{
		Pool:           pool,
		Bus:            bus,
		DataDir:        dataDir,
		Workers:        1,
		Logger:         logger,
		ModelsDir:      modelsDir,
		Registry:       registry,
		ModelsManifest: manifest,
	})
	if err != nil {
		t.Fatalf("start queue: %v", err)
	}
	t.Cleanup(func() {
		stopCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := queue.Stop(stopCtx); err != nil {
			t.Logf("stop queue: %v", err)
		}
	})

	router := api.NewRouter(&api.Deps{
		Pool:             pool,
		Version:          "jobs-tests",
		Events:           bus,
		DataDir:          dataDir,
		UploadLimitBytes: 32 << 20,
		Jobs:             queue,
		ModelDownloads:   queue,
	})

	e := &env{
		pool:      pool,
		queries:   dbgen.New(pool),
		bus:       bus,
		dataDir:   dataDir,
		modelsDir: modelsDir,
		registry:  registry,
		queue:     queue,
		router:    router,
	}

	// The owner of every asset: the first administrator, created
	// through the wizard and signed in through the login it fronts.
	// The request helpers mirror the api package's internal test
	// helpers, which are not importable from here.
	adminBody, err := json.Marshal(apigen.CreateAdminRequest{
		Email:    oapitypes.Email(adminEmail),
		Password: adminPassword,
	})
	if err != nil {
		t.Fatalf("marshal admin body: %v", err)
	}
	rec, req := jsonRequest(http.MethodPost, "/api/v1/server/setup/admin", adminBody)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create admin: status = %d, want %d (body %q)", rec.Code, http.StatusCreated, rec.Body.String())
	}

	loginBody, err := json.Marshal(apigen.LoginRequest{
		Email:    oapitypes.Email(adminEmail),
		Password: adminPassword,
	})
	if err != nil {
		t.Fatalf("marshal login body: %v", err)
	}
	rec, req = jsonRequest(http.MethodPost, "/api/v1/auth/login", loginBody)
	req.Header.Set("User-Agent", "jobs-tests")
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("login: status = %d, want %d (body %q)", rec.Code, http.StatusOK, rec.Body.String())
	}
	for _, cookie := range rec.Result().Cookies() {
		if cookie.Name == sessionCookieName {
			e.cookie = cookie
		}
	}
	if e.cookie == nil {
		t.Fatalf("login response carries no %s cookie", sessionCookieName)
	}

	session, err := e.queries.GetSessionByTokenHash(context.Background(), auth.HashSessionToken(e.cookie.Value))
	if err != nil {
		t.Fatalf("resolve session user: %v", err)
	}
	e.ownerID = uuid.UUID(session.UserID.Bytes)
	return e
}

// jsonRequest builds a JSON POST request with its recorder.
func jsonRequest(method, target string, body []byte) (*httptest.ResponseRecorder, *http.Request) {
	req := httptest.NewRequest(method, target, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	return httptest.NewRecorder(), req
}

// insertAsset stores content under the originals tree, inserts the
// asset row with its exif placeholder, and returns the new asset id.
// The path is relative to the data root, mirroring the receiver's
// stored file_path form. The row pins the pipeline shape of the plain
// metadata-only tests: image kind, one pending job.
func (e *env) insertAsset(t *testing.T, rel string, content []byte, takenAt time.Time, placeholder string) uuid.UUID {
	t.Helper()
	return e.insertAssetKind(t, rel, content, takenAt, placeholder, dbgen.MediaKindImage, "image/jpeg", 1)
}

// insertAssetKind is insertAsset with the row's media kind, mime type,
// and pending-jobs count explicit: the thumbnail tests drive video
// rows and two-job pipelines through it.
func (e *env) insertAssetKind(t *testing.T, rel string, content []byte, takenAt time.Time, placeholder string, kind dbgen.MediaKind, mimeType string, pendingJobs int32) uuid.UUID {
	t.Helper()

	path := filepath.Join(e.dataDir, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("create fixture directory: %v", err)
	}
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("write fixture file: %v", err)
	}

	id, err := uuid.NewV7()
	if err != nil {
		t.Fatalf("new asset id: %v", err)
	}
	if _, err := e.queries.CreateAsset(context.Background(), dbgen.CreateAssetParams{
		ID:               pgUUID(id),
		OwnerID:          pgUUID(e.ownerID),
		FilePath:         rel,
		Checksum:         "test-" + id.String(),
		OriginalFilename: filepath.Base(rel),
		FileSize:         int64(len(content)),
		MimeType:         mimeType,
		MediaKind:        kind,
		TakenAt:          pgtype.Timestamptz{Time: takenAt, Valid: true},
		PendingJobs:      pendingJobs,
	}); err != nil {
		t.Fatalf("insert asset: %v", err)
	}
	if err := e.queries.CreateExifPlaceholder(context.Background(), dbgen.CreateExifPlaceholderParams{
		AssetID: pgUUID(id),
		Exif:    []byte(placeholder),
	}); err != nil {
		t.Fatalf("insert exif placeholder: %v", err)
	}
	return id
}

// uploadFileFixture is one file part of an upload request.
type uploadFileFixture struct {
	field   string
	name    string
	content []byte
}

// postUpload fires a multipart upload with the environment's session.
func (e *env) postUpload(t *testing.T, fields map[string]string, files []uploadFileFixture) *httptest.ResponseRecorder {
	t.Helper()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	for _, name := range []string{"deviceAssetId", "fileName", "deviceCreatedAt", "deviceTimeZone"} {
		if value, ok := fields[name]; ok {
			if err := writer.WriteField(name, value); err != nil {
				t.Fatalf("write field %s: %v", name, err)
			}
		}
	}
	for _, file := range files {
		part, err := writer.CreateFormFile(file.field, file.name)
		if err != nil {
			t.Fatalf("create file part %s: %v", file.field, err)
		}
		if _, err := part.Write(file.content); err != nil {
			t.Fatalf("write file part %s: %v", file.field, err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/assets", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.AddCookie(e.cookie)
	rec := httptest.NewRecorder()
	e.router.ServeHTTP(rec, req)
	return rec
}

// assetState reads the pipeline-relevant columns of an asset row.
type assetState struct {
	takenAt  time.Time
	source   string
	pending  int32
	readyAt  pgtype.Timestamptz
	width    int32
	height   int32
	duration pgtype.Int4
}

// assetOf loads the assetState of an asset.
func (e *env) assetOf(t *testing.T, id uuid.UUID) assetState {
	t.Helper()
	var state assetState
	err := e.pool.QueryRow(context.Background(), `
		SELECT taken_at, taken_at_source, pending_jobs, ready_at,
		       width, height, duration_ms
		FROM assets WHERE id = $1`, id).
		Scan(&state.takenAt, &state.source, &state.pending, &state.readyAt,
			&state.width, &state.height, &state.duration)
	if err != nil {
		t.Fatalf("scan asset row: %v", err)
	}
	return state
}

// exifRow is the exif table row as the assertions read it.
type exifRow struct {
	exif        []byte
	cameraMake  *string
	cameraModel *string
	lens        *string
	iso         *int32
	aperture    *float32
	shutter     *string
	focal       *float32
	lat         *float64
	lon         *float64
	alt         *float32
	tzOffset    *string
	updatedAt   time.Time
}

// exifOf loads the exif row of an asset.
func (e *env) exifOf(t *testing.T, id uuid.UUID) (exifRow, bool) {
	t.Helper()
	var row exifRow
	err := e.pool.QueryRow(context.Background(), `
		SELECT exif, camera_make, camera_model, lens, iso, aperture,
		       shutter, focal, lat, lon, alt, tz_offset, updated_at
		FROM exif WHERE asset_id = $1`, id).
		Scan(&row.exif, &row.cameraMake, &row.cameraModel, &row.lens, &row.iso,
			&row.aperture, &row.shutter, &row.focal, &row.lat, &row.lon,
			&row.alt, &row.tzOffset, &row.updatedAt)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("scan exif row: %v", err)
	}
	return row, err == nil
}

// exifRowCount counts the exif rows of an asset.
func (e *env) exifRowCount(t *testing.T, id uuid.UUID) int {
	t.Helper()
	var count int
	if err := e.pool.QueryRow(context.Background(),
		"SELECT count(*) FROM exif WHERE asset_id = $1", id).Scan(&count); err != nil {
		t.Fatalf("count exif rows: %v", err)
	}
	return count
}

// countEvents replays the bus's retained ring and counts the events of
// one name. The ring keeps five minutes of history in tests, so every
// event of a finished pipeline is still replayable.
func countEvents(bus *events.Bus, name string) int {
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	subscription := bus.Subscribe(ctx, 0)
	count := 0
	for {
		event, ok := subscription.Next()
		if !ok {
			return count
		}
		if event.Name == name {
			count++
		}
	}
}

// eventPayloadOf returns the payload of the first event of the name.
func eventPayloadOf(t *testing.T, bus *events.Bus, name string) map[string]string {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	subscription := bus.Subscribe(ctx, 0)
	for {
		event, ok := subscription.Next()
		if !ok {
			t.Fatalf("no %s event found on the bus", name)
		}
		if event.Name != name {
			continue
		}
		var payload map[string]string
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			t.Fatalf("decode %s payload %s: %v", name, event.Payload, err)
		}
		return payload
	}
}

// eventually polls cond until it holds or the timeout elapses.
func eventually(t *testing.T, timeout time.Duration, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("timed out after %s waiting for %s", timeout, what)
}

// jpegBytes encodes a real JPEG of one pixel through the standard
// encoder: no EXIF whatsoever.
func jpegBytes(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	img.Set(0, 0, color.RGBA{R: 0x7f, G: 0x33, B: 0xcc, A: 0xff})
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, nil); err != nil {
		t.Fatalf("encode JPEG: %v", err)
	}
	return buf.Bytes()
}

// goldenFixture reads the golden JPEG: a real small image carrying
// EXIF (DateTimeOriginal with OffsetTime, GPS coordinates, camera and
// lens identifiers). It is generated inside the dev container and
// committed as a binary fixture.
func goldenFixture(t *testing.T) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "golden.jpg"))
	if err != nil {
		t.Fatalf("read golden fixture: %v", err)
	}
	return data
}

// goldenNoOffsetFixture reads the golden JPEG variant whose EXIF
// carries DateTimeOriginal but no OffsetTime: the taken-at zone then
// resolves through the device-declared time zone or not at all.
func goldenNoOffsetFixture(t *testing.T) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "golden-nooffset.jpg"))
	if err != nil {
		t.Fatalf("read no-offset golden fixture: %v", err)
	}
	return data
}

// goldenPerTagOffsetFixture reads the golden JPEG variant whose EXIF
// carries DateTimeOriginal with ONLY the per-tag OffsetTimeOriginal
// (no composite OffsetTime): the zone then resolves through that
// per-tag offset, ahead of the device-declared zone.
func goldenPerTagOffsetFixture(t *testing.T) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "golden-offsetoriginal.jpg"))
	if err != nil {
		t.Fatalf("read per-tag offset golden fixture: %v", err)
	}
	return data
}

// slowExiftoolScript writes a wrapper script that stands in for the
// exiftool binary: it spawns a long sleep, records that child's pid,
// and itself sleeps long enough to blow any test timeout. It returns
// the script path and the pidfile path; the recorded pid is the
// orphan the process-group kill must reap.
func slowExiftoolScript(t *testing.T) (script, pidFile string) {
	t.Helper()
	dir := t.TempDir()
	script = filepath.Join(dir, "slow-exiftool")
	pidFile = filepath.Join(dir, "child.pid")
	body := "#!/bin/sh\nsleep 300 &\necho $! > " + pidFile + "\nsleep 300\n"
	if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
		t.Fatalf("write wrapper script: %v", err)
	}
	return script, pidFile
}

// slowExiftoolTestTimeout is the shortened exiftool budget of the
// timeout tests. It must stay well above the ~0.4s a freshly exec'd
// shell script needs to reach its first commands on macOS: the point
// of the tests is the kill after the budget, not a race against the
// script even starting.
const slowExiftoolTestTimeout = 2 * time.Second

// requireNoOrphan polls until the process recorded in pidFile is dead:
// the group kill reaped the wrapper's child. Dead means the /proc entry
// is gone or, on Linux, shows the zombie state -- a zombie runs no
// code, holds no pipes and no memory, and only its exit status
// delivery is pending. Under `go test` in the dev container nothing in
// the process tree collects reparented orphans, so the zombie is
// expected to linger; in production the server reaps its direct child
// through Wait and init collects the reparented rest.
func requireNoOrphan(t *testing.T, pidFile string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	var pid int
	for time.Now().Before(deadline) {
		data, err := os.ReadFile(pidFile)
		if err == nil {
			pid, err = strconv.Atoi(strings.TrimSpace(string(data)))
			if err == nil {
				break
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	if pid == 0 {
		t.Fatalf("wrapper script never recorded its child pid in %s", pidFile)
	}
	eventually(t, 5*time.Second, "the wrapper's child process to be dead", func() bool {
		return processDead(pid)
	})
}

// processDead reports whether pid is gone (or a zombie on Linux, which
// has /proc; on macOS the kill probe is the equivalent, and the system
// reaps orphans eagerly enough that no zombie window is observable).
func processDead(pid int) bool {
	if stat, err := os.ReadFile("/proc/" + strconv.Itoa(pid) + "/stat"); err == nil {
		// The process state is the field right after the command name
		// in parentheses; the name itself may contain parentheses, so
		// the LAST closing parenthesis is the separator.
		text := string(stat)
		if idx := strings.LastIndexByte(text, ')'); idx >= 0 && idx+2 < len(text) {
			return text[idx+2] == 'Z'
		}
		return false
	}
	return errors.Is(syscall.Kill(pid, 0), syscall.ESRCH)
}

// TestExiftoolTimeoutKillsProcessGroup pins the subprocess discipline's
// hard bound: a wedged exiftool (here a wrapper script that ignores
// everything and sleeps) fails the extraction with the context's
// deadline, and the kill takes the whole process group, so the child
// the wrapper spawned does not survive as an orphan.
func TestExiftoolTimeoutKillsProcessGroup(t *testing.T) {
	script, pidFile := slowExiftoolScript(t)

	previousBinary, previousTimeout := exiftoolBinary, exiftoolTimeout
	exiftoolBinary, exiftoolTimeout = script, slowExiftoolTestTimeout
	t.Cleanup(func() { exiftoolBinary, exiftoolTimeout = previousBinary, previousTimeout })

	_, err := extractMetadata(context.Background(), "fixture.jpg")
	if err == nil {
		t.Fatal("extraction error = nil, want the timeout to fail it")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("extraction error = %v, want it to wrap context.DeadlineExceeded", err)
	}

	requireNoOrphan(t, pidFile)
}

// TestMetadataJobTimeoutFailsJob pins the job-level half of the
// timeout discipline: Work propagates the extraction failure, the
// database stays untouched (placeholder sidecar, unrefined taken_at,
// un-decremented pipeline counter), and no asset.ready event fires.
func TestMetadataJobTimeoutFailsJob(t *testing.T) {
	e := newEnv(t)

	script, pidFile := slowExiftoolScript(t)
	previousBinary, previousTimeout := exiftoolBinary, exiftoolTimeout
	exiftoolBinary, exiftoolTimeout = script, slowExiftoolTestTimeout
	t.Cleanup(func() { exiftoolBinary, exiftoolTimeout = previousBinary, previousTimeout })

	deviceTakenAt := time.Date(2020, 3, 1, 12, 0, 0, 0, time.UTC).Truncate(time.Microsecond)
	placeholder := `{"deviceCreatedAt":"2020-03-01T12:00:00Z","deviceTimeZone":"Asia/Shanghai"}`
	id := e.insertAsset(t, "originals/timeout/fixture.jpg", jpegBytes(t), deviceTakenAt, placeholder)

	// The worker is invoked directly: the assertion is about Work's
	// own behavior, not River's retry machinery around it.
	worker := &MetadataWorker{
		pool:    e.pool,
		bus:     e.bus,
		dataDir: e.dataDir,
		logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
	err := worker.Work(context.Background(), &river.Job[MetadataArgs]{Args: MetadataArgs{AssetID: id}})
	if err == nil {
		t.Fatal("Work error = nil, want the timeout to fail the job")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("Work error = %v, want it to wrap context.DeadlineExceeded", err)
	}

	row, ok := e.exifOf(t, id)
	if !ok {
		t.Fatal("exif placeholder row disappeared after a failed job")
	}
	// jsonb round-trips are not byte-stable (PostgreSQL normalizes the
	// document), so the untouched assertion compares the decoded form.
	var sidecar map[string]string
	if err := json.Unmarshal(row.exif, &sidecar); err != nil {
		t.Fatalf("decode exif jsonb %s: %v", row.exif, err)
	}
	if sidecar["deviceCreatedAt"] != "2020-03-01T12:00:00Z" || sidecar["deviceTimeZone"] != "Asia/Shanghai" {
		t.Errorf("exif jsonb = %s, want the untouched placeholder fields", row.exif)
	}
	state := e.assetOf(t, id)
	if state.pending != 1 {
		t.Errorf("pending_jobs = %d, want the un-decremented 1", state.pending)
	}
	if state.readyAt.Valid {
		t.Error("ready_at is set after a failed job")
	}
	if !state.takenAt.Equal(deviceTakenAt) {
		t.Errorf("taken_at = %s, want the untouched device value %s", state.takenAt, deviceTakenAt)
	}
	if got := countEvents(e.bus, "asset.ready"); got != 0 {
		t.Errorf("asset.ready events = %d, want 0", got)
	}

	requireNoOrphan(t, pidFile)
}

// TestMetadataJobGoldenFixture is the mandated golden flow: a real
// exiftool run over a JPEG whose EXIF carries DateTimeOriginal with an
// offset, GPS, and camera identifiers fills the exif row (hot fields
// plus the raw jsonb), refines taken_at to the EXIF instant with
// taken_at_source 'exif', and completes the pipeline -- ready_at is
// stamped and asset.ready reaches the bus with the minimal payload.
func TestMetadataJobGoldenFixture(t *testing.T) {
	requireExiftool(t)
	e := newEnv(t)

	golden := goldenFixture(t)
	deviceTakenAt := time.Date(2020, 3, 1, 12, 0, 0, 0, time.UTC).Truncate(time.Microsecond)
	placeholder := `{"deviceCreatedAt":"2020-03-01T12:00:00Z","deviceTimeZone":"Asia/Shanghai"}`
	id := e.insertAsset(t, "originals/golden/golden.jpg", golden, deviceTakenAt, placeholder)

	if err := e.queue.EnqueueMetadata(context.Background(), id); err != nil {
		t.Fatalf("enqueue metadata job: %v", err)
	}
	eventually(t, jobWaitTimeout, "the golden asset to become ready", func() bool {
		return e.assetOf(t, id).readyAt.Valid
	})

	row, ok := e.exifOf(t, id)
	if !ok {
		t.Fatal("exif row missing after the job ran")
	}
	if e.exifRowCount(t, id) != 1 {
		t.Errorf("exif rows = %d, want exactly 1", e.exifRowCount(t, id))
	}
	if row.cameraMake == nil || *row.cameraMake != "Lumemor" {
		t.Errorf("camera_make = %v, want Lumemor", row.cameraMake)
	}
	if row.cameraModel == nil || *row.cameraModel != "Golden Camera" {
		t.Errorf("camera_model = %v, want Golden Camera", row.cameraModel)
	}
	if row.lens == nil || *row.lens != "Golden 35mm f/2.8" {
		t.Errorf("lens = %v, want Golden 35mm f/2.8", row.lens)
	}
	if row.iso == nil || *row.iso != 200 {
		t.Errorf("iso = %v, want 200", row.iso)
	}
	if row.aperture == nil || *row.aperture != 2.8 {
		t.Errorf("aperture = %v, want 2.8", row.aperture)
	}
	if row.shutter == nil || *row.shutter != "1/250" {
		t.Errorf("shutter = %v, want 1/250", row.shutter)
	}
	if row.focal == nil || *row.focal != 35 {
		t.Errorf("focal = %v, want 35", row.focal)
	}
	if row.lat == nil || *row.lat != 48.8584 {
		t.Errorf("lat = %v, want 48.8584", row.lat)
	}
	if row.lon == nil || *row.lon != 2.2945 {
		t.Errorf("lon = %v, want 2.2945", row.lon)
	}
	if row.alt == nil || *row.alt != 35 {
		t.Errorf("alt = %v, want 35", row.alt)
	}
	if row.tzOffset == nil || *row.tzOffset != "+05:30" {
		t.Errorf("tz_offset = %v, want +05:30", row.tzOffset)
	}

	// The raw jsonb is the verbatim exiftool output: a JSON object
	// carrying the extraction's own tag names.
	var raw map[string]any
	if err := json.Unmarshal(row.exif, &raw); err != nil {
		t.Fatalf("decode exif jsonb %s: %v", row.exif, err)
	}
	if value, ok := raw["DateTimeOriginal"].(string); !ok || value != "2023:07:14 10:15:30" {
		t.Errorf("jsonb DateTimeOriginal = %v, want 2023:07:14 10:15:30", raw["DateTimeOriginal"])
	}

	// 2023:07:14 10:15:30 with a +05:30 offset is 04:45:30Z.
	state := e.assetOf(t, id)
	wantTakenAt := time.Date(2023, 7, 14, 4, 45, 30, 0, time.UTC)
	if !state.takenAt.Equal(wantTakenAt) {
		t.Errorf("taken_at = %s, want the EXIF instant %s", state.takenAt, wantTakenAt)
	}
	if state.source != "exif" {
		t.Errorf("taken_at_source = %q, want exif", state.source)
	}
	if state.pending != 0 {
		t.Errorf("pending_jobs = %d, want 0", state.pending)
	}
	if state.width != 64 || state.height != 64 {
		t.Errorf("dimensions = %dx%d, want 64x64", state.width, state.height)
	}

	payload := eventPayloadOf(t, e.bus, "asset.ready")
	if payload["assetId"] != id.String() {
		t.Errorf("asset.ready assetId = %q, want %s", payload["assetId"], id)
	}
	if payload["kind"] != "image" {
		t.Errorf("asset.ready kind = %q, want image", payload["kind"])
	}
	if got := countEvents(e.bus, "asset.ready"); got != 1 {
		t.Errorf("asset.ready events = %d, want exactly 1", got)
	}
}

// TestMetadataJobNoExifDate pins the fallback-chain guarantee: a JPEG
// without an EXIF capture date leaves taken_at and taken_at_source
// exactly as the upload wrote them (the device value, source 'import').
func TestMetadataJobNoExifDate(t *testing.T) {
	requireExiftool(t)
	e := newEnv(t)

	deviceTakenAt := time.Date(2019, 11, 30, 8, 30, 0, 0, time.UTC).Truncate(time.Microsecond)
	placeholder := `{"deviceCreatedAt":"2019-11-30T08:30:00Z","deviceTimeZone":"Asia/Shanghai"}`
	id := e.insertAsset(t, "originals/plain/plain.jpg", jpegBytes(t), deviceTakenAt, placeholder)

	if err := e.queue.EnqueueMetadata(context.Background(), id); err != nil {
		t.Fatalf("enqueue metadata job: %v", err)
	}
	eventually(t, jobWaitTimeout, "the plain asset to become ready", func() bool {
		return e.assetOf(t, id).readyAt.Valid
	})

	state := e.assetOf(t, id)
	if !state.takenAt.Equal(deviceTakenAt) {
		t.Errorf("taken_at = %s, want the untouched device value %s", state.takenAt, deviceTakenAt)
	}
	if state.source != "import" {
		t.Errorf("taken_at_source = %q, want the unchanged import", state.source)
	}

	// The extraction still ran: the sidecar carries the raw output,
	// dimensions refine, hot fields stay NULL (the file has no EXIF).
	row, ok := e.exifOf(t, id)
	if !ok {
		t.Fatal("exif row missing after the job ran")
	}
	var raw map[string]any
	if err := json.Unmarshal(row.exif, &raw); err != nil {
		t.Fatalf("decode exif jsonb %s: %v", row.exif, err)
	}
	if _, ok := raw["DateTimeOriginal"]; ok {
		t.Error("jsonb carries DateTimeOriginal, want none for an EXIF-less JPEG")
	}
	if state.width != 1 || state.height != 1 {
		t.Errorf("dimensions = %dx%d, want the extracted 1x1", state.width, state.height)
	}
	if row.cameraMake != nil || row.iso != nil || row.lat != nil || row.tzOffset != nil {
		t.Errorf("hot fields = make:%v iso:%v lat:%v tz:%v, want all NULL for an EXIF-less JPEG",
			row.cameraMake, row.iso, row.lat, row.tzOffset)
	}
}

// TestMetadataJobTakenAtZoneFallback pins the zone resolution chain
// below the EXIF offset: DateTimeOriginal without OffsetTime resolves
// through the per-tag OffsetTimeOriginal when only that one is present
// (many cameras write it exclusively), then through the device-declared
// IANA zone, and without any zone at all it does not refine -- the
// upload's value stands rather than a guessed-zone instant twelve
// hours wrong.
func TestMetadataJobTakenAtZoneFallback(t *testing.T) {
	requireExiftool(t)
	e := newEnv(t)

	deviceTakenAt := time.Date(2019, 6, 15, 6, 0, 0, 0, time.UTC).Truncate(time.Microsecond)
	withZone := e.insertAsset(t, "originals/fallback/zone.jpg", goldenNoOffsetFixture(t), deviceTakenAt,
		`{"deviceCreatedAt":"2019-06-15T06:00:00Z","deviceTimeZone":"Asia/Shanghai"}`)
	withoutZone := e.insertAsset(t, "originals/fallback/none.jpg", goldenNoOffsetFixture(t), deviceTakenAt,
		`{}`)
	withPerTagOffset := e.insertAsset(t, "originals/fallback/pertag.jpg", goldenPerTagOffsetFixture(t), deviceTakenAt,
		`{"deviceCreatedAt":"2019-06-15T06:00:00Z","deviceTimeZone":"Asia/Shanghai"}`)

	for _, id := range []uuid.UUID{withZone, withoutZone, withPerTagOffset} {
		if err := e.queue.EnqueueMetadata(context.Background(), id); err != nil {
			t.Fatalf("enqueue metadata job: %v", err)
		}
	}
	eventually(t, jobWaitTimeout, "the fallback assets to become ready", func() bool {
		return e.assetOf(t, withZone).readyAt.Valid && e.assetOf(t, withoutZone).readyAt.Valid &&
			e.assetOf(t, withPerTagOffset).readyAt.Valid
	})

	// 2023:07:14 10:15:30 in Asia/Shanghai (+08:00) is 02:15:30Z.
	zoneState := e.assetOf(t, withZone)
	wantTakenAt := time.Date(2023, 7, 14, 2, 15, 30, 0, time.UTC)
	if !zoneState.takenAt.Equal(wantTakenAt) {
		t.Errorf("zone fallback taken_at = %s, want %s", zoneState.takenAt, wantTakenAt)
	}
	if zoneState.source != "exif" {
		t.Errorf("zone fallback taken_at_source = %q, want exif", zoneState.source)
	}
	zoneRow, ok := e.exifOf(t, withZone)
	if !ok {
		t.Fatal("exif row missing for the zone fallback asset")
	}
	if zoneRow.tzOffset == nil || *zoneRow.tzOffset != "+08:00" {
		t.Errorf("tz_offset = %v, want the device zone's offset +08:00", zoneRow.tzOffset)
	}

	// The per-tag offset wins over the device zone: the fixture's
	// OffsetTimeOriginal +07:00 resolves DateTimeOriginal to 03:15:30Z
	// -- NOT the Asia/Shanghai +08:00 instant the device zone would
	// produce.
	perTagState := e.assetOf(t, withPerTagOffset)
	wantPerTagTakenAt := time.Date(2023, 7, 14, 3, 15, 30, 0, time.UTC)
	if !perTagState.takenAt.Equal(wantPerTagTakenAt) {
		t.Errorf("per-tag offset taken_at = %s, want %s", perTagState.takenAt, wantPerTagTakenAt)
	}
	if perTagState.source != "exif" {
		t.Errorf("per-tag offset taken_at_source = %q, want exif", perTagState.source)
	}
	perTagRow, ok := e.exifOf(t, withPerTagOffset)
	if !ok {
		t.Fatal("exif row missing for the per-tag offset asset")
	}
	if perTagRow.tzOffset == nil || *perTagRow.tzOffset != "+07:00" {
		t.Errorf("tz_offset = %v, want the per-tag offset +07:00", perTagRow.tzOffset)
	}

	// No zone anywhere: the wall clock is left unrefined.
	noneState := e.assetOf(t, withoutZone)
	if !noneState.takenAt.Equal(deviceTakenAt) {
		t.Errorf("zoneless taken_at = %s, want the untouched device value %s", noneState.takenAt, deviceTakenAt)
	}
	if noneState.source != "import" {
		t.Errorf("zoneless taken_at_source = %q, want the unchanged import", noneState.source)
	}
}

// TestMetadataJobRerunIsIdempotent pins the re-run contract: a second
// run of the job on an already-completed asset converges to the same
// single exif row with the same values, leaves the readiness state
// untouched, and does not publish a second asset.ready.
func TestMetadataJobRerunIsIdempotent(t *testing.T) {
	requireExiftool(t)
	e := newEnv(t)

	golden := goldenFixture(t)
	deviceTakenAt := time.Date(2020, 3, 1, 12, 0, 0, 0, time.UTC).Truncate(time.Microsecond)
	placeholder := `{"deviceCreatedAt":"2020-03-01T12:00:00Z","deviceTimeZone":"Asia/Shanghai"}`
	id := e.insertAsset(t, "originals/rerun/golden.jpg", golden, deviceTakenAt, placeholder)

	if err := e.queue.EnqueueMetadata(context.Background(), id); err != nil {
		t.Fatalf("enqueue metadata job: %v", err)
	}
	eventually(t, jobWaitTimeout, "the first run to complete", func() bool {
		return e.assetOf(t, id).readyAt.Valid
	})
	firstRun, _ := e.exifOf(t, id)
	firstReadyAt := e.assetOf(t, id).readyAt

	// The manual re-enqueue: the review-focus rerun must be safe.
	if err := e.queue.EnqueueMetadata(context.Background(), id); err != nil {
		t.Fatalf("re-enqueue metadata job: %v", err)
	}
	eventually(t, jobWaitTimeout, "the re-run to complete", func() bool {
		row, ok := e.exifOf(t, id)
		return ok && row.updatedAt.After(firstRun.updatedAt)
	})
	// The event, if one were wrongly published, follows the commit
	// synchronously; a short settle makes the count deterministic.
	time.Sleep(300 * time.Millisecond)

	if got := e.exifRowCount(t, id); got != 1 {
		t.Errorf("exif rows = %d, want exactly 1 after the re-run", got)
	}
	secondRun, _ := e.exifOf(t, id)
	if !equalExifHotFields(firstRun, secondRun) {
		t.Errorf("exif hot fields changed on re-run:\nfirst:  %+v\nsecond: %+v",
			exifHotFieldsOf(firstRun), exifHotFieldsOf(secondRun))
	}
	state := e.assetOf(t, id)
	if state.pending != 0 {
		t.Errorf("pending_jobs = %d, want 0 after the re-run", state.pending)
	}
	if !state.readyAt.Time.Equal(firstReadyAt.Time) {
		t.Errorf("ready_at = %s, want the unchanged %s", state.readyAt.Time, firstReadyAt.Time)
	}
	if got := countEvents(e.bus, "asset.ready"); got != 1 {
		t.Errorf("asset.ready events = %d, want the single original one", got)
	}
}

// equalExifHotFields compares the hot fields (everything but the raw
// jsonb and the timestamps) of two exif rows.
func equalExifHotFields(a, b exifRow) bool {
	return exifHotFieldsOf(a) == exifHotFieldsOf(b)
}

// exifHotFieldsOf renders the comparable form of a row's hot fields.
func exifHotFieldsOf(row exifRow) string {
	values := []any{row.cameraMake, row.cameraModel, row.lens, row.iso,
		row.aperture, row.shutter, row.focal, row.lat, row.lon, row.alt, row.tzOffset}
	parts := make([]string, len(values))
	for i, value := range values {
		encoded, err := json.Marshal(value)
		if err != nil {
			parts[i] = err.Error()
			continue
		}
		parts[i] = string(encoded)
	}
	return strings.Join(parts, ",")
}

// TestUploadToReadyFullFlow is the mandated end-to-end: a multipart
// upload through the HTTP endpoint enqueues both pipeline jobs after
// the transaction commits (pending_jobs starts at 2), the jobs refine
// the asset and derive its serving copies, and asset.ready -- with
// asset.uploaded before it -- reaches the bus exactly once, only after
// BOTH jobs completed.
func TestUploadToReadyFullFlow(t *testing.T) {
	requireExiftool(t)
	requireThumbnails(t)
	e := newEnv(t)

	golden := goldenFixture(t)
	rec := e.postUpload(t, map[string]string{
		"deviceAssetId":   "device-golden-1",
		"fileName":        "golden.jpg",
		"deviceCreatedAt": "2020-05-05T09:00:00Z",
		"deviceTimeZone":  "Asia/Shanghai",
	}, []uploadFileFixture{
		{field: "photo", name: "golden.jpg", content: golden},
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("upload: status = %d, want %d (body %q)", rec.Code, http.StatusCreated, rec.Body.String())
	}
	var response apigen.AssetResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode upload response %q: %v", rec.Body.String(), err)
	}
	assetID := response.Id
	if response.TakenAt.UTC() != time.Date(2020, 5, 5, 9, 0, 0, 0, time.UTC) {
		t.Errorf("upload response taken_at = %s, want the device value", response.TakenAt.UTC())
	}

	// The uploader counts both jobs: readiness only follows their
	// combined completion.
	if state := e.assetOf(t, assetID); state.pending != 2 {
		t.Errorf("pending_jobs right after upload = %d, want 2", state.pending)
	}
	eventually(t, jobWaitTimeout, "the uploaded asset to become ready", func() bool {
		return e.assetOf(t, assetID).readyAt.Valid
	})

	// The upload's taken_at (device value) is refined to the EXIF
	// instant by the job.
	state := e.assetOf(t, assetID)
	wantTakenAt := time.Date(2023, 7, 14, 4, 45, 30, 0, time.UTC)
	if !state.takenAt.Equal(wantTakenAt) {
		t.Errorf("taken_at = %s, want the EXIF instant %s", state.takenAt, wantTakenAt)
	}
	if state.source != "exif" {
		t.Errorf("taken_at_source = %q, want exif", state.source)
	}

	// The thumbnail job ran too: the three AVIF serving copies exist
	// (64x64 source: tile and view keep the source size) and the
	// blurhash is stored on the row.
	requireJobThumbDimensions(t, assetID, media.SizeTile, 64, 64)
	requireJobThumbDimensions(t, assetID, media.SizeView, 64, 64)
	requireJobThumbDimensions(t, assetID, media.SizeOriginal, 64, 64)
	if hash := e.blurhashOf(t, assetID); !hash.Valid || len(hash.String) < 6 {
		t.Errorf("blurhash = %v, want the stored non-empty hash", hash)
	}

	if got := countEvents(e.bus, "asset.uploaded"); got != 1 {
		t.Errorf("asset.uploaded events = %d, want 1", got)
	}
	if got := countEvents(e.bus, "asset.ready"); got != 1 {
		t.Errorf("asset.ready events = %d, want 1", got)
	}
	payload := eventPayloadOf(t, e.bus, "asset.ready")
	if payload["assetId"] != assetID.String() {
		t.Errorf("asset.ready assetId = %q, want %s", payload["assetId"], assetID)
	}
	if payload["kind"] != "image" {
		t.Errorf("asset.ready kind = %q, want image", payload["kind"])
	}
}

// TestUploadLivePhotoPipeline pins the Live Photo pipeline shape: the
// upload enqueues a metadata job for BOTH rows (each carries an exif
// sidecar to refine), a thumbnail job for the photo row ONLY (the
// component video is never the client-visible asset, so it gets no
// poster), and -- M2d -- exactly one transcode job for the COMPONENT
// row (its playable stream; the photo row itself is an image and gets
// none). The component's counter counts that transcode alone (1),
// while the photo row's keeps its two; the metadata completion on the
// component must not consume the transcode's count, and the
// component's transcode completion stamps its ready_at but never
// announces -- asset.ready fires exactly once, for the photo. The
// component's play endpoint then serves the stream the job produced.
func TestUploadLivePhotoPipeline(t *testing.T) {
	requireExiftool(t)
	requireThumbnails(t)
	requireTranscodeToolchain(t)
	e := newEnv(t)

	golden := goldenFixture(t)
	// A real video: the component's transcode re-encodes it, so the
	// structural 28-byte mp4 fixture is not enough here.
	video, err := os.ReadFile(filepath.Join("..", "media", "testdata", "tiny.mp4"))
	if err != nil {
		t.Fatalf("read video fixture: %v", err)
	}
	rec := e.postUpload(t, map[string]string{
		"deviceAssetId":   "device-livephoto-1",
		"fileName":        "golden.jpg",
		"deviceCreatedAt": "2020-05-05T09:00:00Z",
		"deviceTimeZone":  "Asia/Shanghai",
	}, []uploadFileFixture{
		{field: "photo", name: "golden.jpg", content: golden},
		{field: "liveVideo", name: "live.mp4", content: video},
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("upload: status = %d, want %d (body %q)", rec.Code, http.StatusCreated, rec.Body.String())
	}
	var response apigen.AssetResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode upload response %q: %v", rec.Body.String(), err)
	}
	photoID := response.Id

	// The component video row: linked from the photo, counting its
	// one transcode job.
	var videoID uuid.UUID
	var videoPending int32
	var videoReadyAt pgtype.Timestamptz
	err = e.pool.QueryRow(context.Background(), `
		SELECT id, pending_jobs, ready_at FROM assets
		WHERE owner_id = $1 AND is_component AND live_photo_video_id IS NULL`, pgUUID(e.ownerID)).
		Scan(&videoID, &videoPending, &videoReadyAt)
	if err != nil {
		t.Fatalf("load component video row: %v", err)
	}
	if videoPending != 1 {
		t.Errorf("component pending_jobs = %d, want 1 (the transcode job alone)", videoPending)
	}

	// Exactly one transcode job entered the queue for the component
	// (plus its metadata job, which refines the sidecar); the photo
	// row keeps its own metadata and thumbnail pair and gains no
	// transcode of its own (it is an image). The enqueue happens
	// before the upload response is written, so the queue state is
	// already settled here.
	kinds := e.riverJobsOf(t, videoID)
	if kinds["metadata"] != 1 || kinds["transcode"] != 1 || e.riverJobTotal(t, videoID) != 2 {
		t.Errorf("component river jobs = %v, want exactly one metadata and one transcode job", kinds)
	}
	if kinds["thumbnail"] != 0 {
		t.Errorf("component thumbnail jobs = %d, want 0 (never the client-visible asset)", kinds["thumbnail"])
	}
	if photoKinds := e.riverJobsOf(t, photoID); photoKinds["transcode"] != 0 {
		t.Errorf("photo transcode jobs = %d, want 0 (the pair's visible asset is the image)", photoKinds["transcode"])
	}

	eventually(t, jobWaitTimeout, "the photo to become ready", func() bool {
		return e.assetOf(t, photoID).readyAt.Valid
	})
	// The video's metadata job ran too: its sidecar carries the raw
	// extraction, not the upload placeholder.
	eventually(t, jobWaitTimeout, "the component's sidecar to be refined", func() bool {
		var refined bool
		if err := e.pool.QueryRow(context.Background(),
			"SELECT exif::jsonb ? 'MajorBrand' FROM exif WHERE asset_id = $1", videoID).Scan(&refined); err != nil {
			return false
		}
		return refined
	})
	// The component's transcode completes: its own pipeline (the one
	// job it counts) drains and stamps ready_at.
	eventually(t, jobWaitTimeout, "the component to become ready", func() bool {
		return e.assetOf(t, videoID).readyAt.Valid
	})

	state := e.assetOf(t, photoID)
	if !state.readyAt.Valid {
		t.Fatal("photo never became ready")
	}
	if got := countEvents(e.bus, "asset.ready"); got != 1 {
		t.Errorf("asset.ready events = %d, want exactly 1 (the photo only; components never announce)", got)
	}
	payload := eventPayloadOf(t, e.bus, "asset.ready")
	if payload["assetId"] != photoID.String() {
		t.Errorf("asset.ready assetId = %q, want the photo %s", payload["assetId"], photoID)
	}
	if payload["kind"] != "image" {
		t.Errorf("asset.ready kind = %q, want the photo's image", payload["kind"])
	}

	videoState := e.assetOf(t, videoID)
	if !videoState.readyAt.Valid {
		t.Fatal("component video never became ready, want its transcode completion to stamp ready_at")
	}
	if videoState.pending != 0 {
		t.Errorf("component pending_jobs = %d, want 0 after its transcode completed", videoState.pending)
	}

	// The photo's serving copies exist; the component video derives
	// no thumbnails (no thumbnail job was enqueued for it) but its
	// playable stream does exist -- with the +faststart layout the
	// transcode produces.
	requireJobThumbDimensions(t, photoID, media.SizeTile, 64, 64)
	requireJobThumbDimensions(t, photoID, media.SizeOriginal, 64, 64)
	if files := e.thumbFiles(t, videoID); files != nil {
		t.Errorf("component video derived files = %v, want none", files)
	}
	requireTranscodedOutput(t, videoID)

	// The pair's motion plays: the component's play endpoint serves
	// the transcoded stream with the static-asset validators of a
	// derived media file.
	play := e.serveGet(t, "/api/v1/assets/"+videoID.String()+"/play", nil)
	if play.Code != http.StatusOK {
		t.Fatalf("component play status = %d, want %d (body %q)", play.Code, http.StatusOK, play.Body.String())
	}
	if got := play.Header().Get("Content-Type"); got != "video/mp4" {
		t.Errorf("component play content-type = %q, want video/mp4", got)
	}
	videoChecksum := sha256Hex(video)
	if got := play.Header().Get("ETag"); got != `"`+videoChecksum[:16]+`"` {
		t.Errorf("component play etag = %q, want the component checksum prefix", got)
	}
	onDisk, err := os.ReadFile(storage.EncodedPath(videoID.String()))
	if err != nil {
		t.Fatalf("read transcoded file of the component: %v", err)
	}
	if play.Body.Len() != len(onDisk) {
		t.Errorf("component play body = %d bytes, want the %d bytes of the transcoded file", play.Body.Len(), len(onDisk))
	}
	partial := e.serveGet(t, "/api/v1/assets/"+videoID.String()+"/play",
		map[string]string{"Range": "bytes=0-99"})
	if partial.Code != http.StatusPartialContent {
		t.Fatalf("component play range status = %d, want %d", partial.Code, http.StatusPartialContent)
	}
	if got := partial.Header().Get("Content-Range"); got != "bytes 0-99/"+strconv.Itoa(len(onDisk)) {
		t.Errorf("component play content-range = %q, want bytes 0-99/%d", got, len(onDisk))
	}
}

// TestParseOffsetAndWallClock pins the pure parsing helpers on inputs
// the fixtures and real files produce.
func TestParseOffsetAndWallClock(t *testing.T) {
	for _, test := range []struct {
		input string
		want  int
		ok    bool
	}{
		{input: "+05:30", want: 5*3600 + 30*60, ok: true},
		{input: "-08:00", want: -8 * 3600, ok: true},
		{input: "+00:00", want: 0, ok: true},
		{input: "05:30", ok: false},
		{input: "+5:30", ok: false},
		{input: "+24:00", ok: false},
		{input: "", ok: false},
	} {
		got, ok := parseOffset(test.input)
		if ok != test.ok || got != test.want {
			t.Errorf("parseOffset(%q) = %d, %v, want %d, %v", test.input, got, ok, test.want, test.ok)
		}
	}
	if formatOffset(5*3600+30*60) != "+05:30" || formatOffset(-8*3600) != "-08:00" || formatOffset(0) != "+00:00" {
		t.Errorf("formatOffset round trip broke: %s %s %s",
			formatOffset(5*3600+30*60), formatOffset(-8*3600), formatOffset(0))
	}
}

// TestShutterText pins the exposure-time rendering: whole-number
// reciprocals render as fractions, everything else as seconds.
func TestShutterText(t *testing.T) {
	for _, test := range []struct {
		seconds float64
		want    string
	}{
		{seconds: 0.004, want: "1/250"},
		{seconds: 0.5, want: "1/2"},
		{seconds: 30, want: "30s"},
		{seconds: 0.0025, want: "1/400"},
	} {
		got := shutterText(map[string]any{"ExposureTime": test.seconds})
		if !got.Valid || got.String != test.want {
			t.Errorf("shutterText(%v) = %v, want %s", test.seconds, got, test.want)
		}
	}
	if got := shutterText(map[string]any{}); got.Valid {
		t.Errorf("shutterText of a missing tag = %v, want invalid", got)
	}
}
