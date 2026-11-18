package api

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"image"
	"image/color"
	"image/jpeg"
	"io"
	"io/fs"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	apigen "github.com/welpher/lumemor/server/internal/api/gen"
	"github.com/welpher/lumemor/server/internal/auth"
	"github.com/welpher/lumemor/server/internal/db"
	"github.com/welpher/lumemor/server/internal/db/dbtest"
	dbgen "github.com/welpher/lumemor/server/internal/db/gen"
	"github.com/welpher/lumemor/server/internal/events"
	"github.com/welpher/lumemor/server/internal/maintenance"
	"github.com/welpher/lumemor/server/internal/storage"
)

// testUploadLimit is the per-file upload limit of the standard asset
// test environment: generous for fixtures, small enough to allocate.
const testUploadLimit int64 = 32 << 20

// assetEventWait bounds how long a test keeps observing the bus when
// asserting that no further asset.uploaded events arrive: publishes
// are synchronous with the handler, so everything observable is on
// the bus once the response is back.
const assetEventWait = 200 * time.Millisecond

// assetEnv provisions the upload environment: a migrated throwaway
// database, a fresh storage layout (the receiver's content-addressed
// paths derive from its data root), a live event bus, and a session
// of the first administrator.
type assetEnv struct {
	router  http.Handler
	pool    *pgxpool.Pool
	queries *dbgen.Queries
	bus     *events.Bus
	dataDir string
	cookie  *http.Cookie
}

// newAssetEnv provisions the environment with the given per-file
// upload limit and returns it with the administrator signed in.
func newAssetEnv(t *testing.T, uploadLimit int64) *assetEnv {
	t.Helper()
	return newAssetEnvWithClock(t, uploadLimit, nil)
}

// newAssetEnvWithClock is newAssetEnv with the instance clock injected:
// the cover rotation derives its ISO week from it, so a test walks a
// week boundary without waiting for one. A nil clock keeps time.Now.
func newAssetEnvWithClock(t *testing.T, uploadLimit int64, now func() time.Time) *assetEnv {
	t.Helper()
	pool := dbtest.New(t)
	if err := db.Migrate(t.Context(), pool); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
	dataDir := t.TempDir()
	if err := storage.EnsureLayout(dataDir, t.TempDir()); err != nil {
		t.Fatalf("prepare storage layout: %v", err)
	}
	bus := events.NewBus(5*time.Minute, time.Now)
	router := NewRouter(&Deps{
		Pool:             pool,
		Version:          testVersion,
		Events:           bus,
		DataDir:          dataDir,
		UploadLimitBytes: uploadLimit,
		Now:              now,
	})
	env := &assetEnv{
		router:  router,
		pool:    pool,
		queries: dbgen.New(pool),
		bus:     bus,
		dataDir: dataDir,
	}
	// The owner of every upload: the first administrator, signed in
	// through the wizard and the login it fronts.
	rec := postAdmin(t, router, createAdminBody(t, "ada@example.com", testPassword, nil))
	if rec.Code != http.StatusCreated {
		t.Fatalf("create admin: status = %d, want %d (body %q)", rec.Code, http.StatusCreated, rec.Body.String())
	}
	cookie, _ := loginCookie(t, router, "ada@example.com", testPassword, "asset-tests")
	env.cookie = cookie
	return env
}

// exec runs a maintenance statement against the test database with a
// background context (it also backs assertions that run after the
// response completed; the test context stays live regardless).
func (e *assetEnv) exec(t *testing.T, statement string, args ...any) {
	t.Helper()
	if _, err := e.pool.Exec(context.Background(), statement, args...); err != nil {
		t.Fatalf("exec %q: %v", statement, err)
	}
}

// queryInt returns the single integer result of a scalar query.
func (e *assetEnv) queryInt(t *testing.T, query string, args ...any) int {
	t.Helper()
	var value int
	if err := e.pool.QueryRow(context.Background(), query, args...).Scan(&value); err != nil {
		t.Fatalf("query %q: %v", query, err)
	}
	return value
}

// queryText returns the single text result of a scalar query.
func (e *assetEnv) queryText(t *testing.T, query string, args ...any) string {
	t.Helper()
	var value string
	if err := e.pool.QueryRow(context.Background(), query, args...).Scan(&value); err != nil {
		t.Fatalf("query %q: %v", query, err)
	}
	return value
}

// sessionUserID resolves the owner behind the environment's session.
func (e *assetEnv) sessionUserID(t *testing.T) uuid.UUID {
	t.Helper()
	row, err := e.queries.GetSessionByTokenHash(context.Background(), auth.HashSessionToken(e.cookie.Value))
	if err != nil {
		t.Fatalf("resolve session user: %v", err)
	}
	return uuid.UUID(row.UserID.Bytes)
}

// assetByChecksum loads the owner's asset row of a checksum.
func (e *assetEnv) assetByChecksum(t *testing.T, checksum string) dbgen.Asset {
	t.Helper()
	row, err := e.queries.GetAssetByOwnerChecksum(context.Background(), dbgen.GetAssetByOwnerChecksumParams{
		OwnerID:  pgUUID(e.sessionUserID(t)),
		Checksum: checksum,
	})
	if err != nil {
		t.Fatalf("GetAssetByOwnerChecksum(%q) error = %v", checksum, err)
	}
	return row
}

// assetUploadFixture is one file part of an upload request.
type assetUploadFile struct {
	field   string
	name    string
	content []byte
}

// buildUploadBody assembles a multipart upload body in contract
// order: the text fields first, then the file parts.
func buildUploadBody(t *testing.T, fields map[string]string, files []assetUploadFile) (*bytes.Buffer, string) {
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
	return &body, writer.FormDataContentType()
}

// postUpload fires an upload request with the environment's session
// cookie; a nil cookie sends it unauthenticated.
func (e *assetEnv) postUpload(t *testing.T, fields map[string]string, files []assetUploadFile) *httptest.ResponseRecorder {
	t.Helper()
	body, contentType := buildUploadBody(t, fields, files)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/assets", body)
	req.Header.Set("Content-Type", contentType)
	if e.cookie != nil {
		req.AddCookie(e.cookie)
	}
	rec := httptest.NewRecorder()
	e.router.ServeHTTP(rec, req)
	return rec
}

// postUploadRaw fires an upload request from a prebuilt body.
func (e *assetEnv) postUploadRaw(t *testing.T, body *bytes.Buffer, contentType string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/assets", body)
	req.Header.Set("Content-Type", contentType)
	if e.cookie != nil {
		req.AddCookie(e.cookie)
	}
	rec := httptest.NewRecorder()
	e.router.ServeHTTP(rec, req)
	return rec
}

// decodeAsset unmarshals an AssetResponse body.
func decodeAsset(t *testing.T, rec *httptest.ResponseRecorder) apigen.AssetResponse {
	t.Helper()
	var asset apigen.AssetResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &asset); err != nil {
		t.Fatalf("decode asset body %q: %v", rec.Body.String(), err)
	}
	return asset
}

// jpegFixture encodes a real 1x1 JPEG through the standard encoder.
func jpegFixture(t *testing.T) []byte {
	t.Helper()
	return jpegColorFixture(t, color.RGBA{R: 0x7f, G: 0x33, B: 0xcc, A: 0xff})
}

// jpegColorFixture encodes a real 1x1 JPEG of the given pixel color;
// distinct colors give distinct bytes, and therefore distinct
// content-addressed blobs.
func jpegColorFixture(t *testing.T, pixel color.RGBA) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	img.Set(0, 0, pixel)
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, nil); err != nil {
		t.Fatalf("encode JPEG fixture: %v", err)
	}
	return buf.Bytes()
}

// movFixture assembles a minimal QuickTime container: an ftyp box
// carrying the qt brand (size, "ftyp", major brand, minor version),
// which the structural fallback of the upload receiver identifies,
// plus trailing payload.
func movFixture() []byte {
	box := make([]byte, 20)
	binary.BigEndian.PutUint32(box, uint32(len(box)))
	box[4], box[5], box[6], box[7] = 'f', 't', 'y', 'p'
	copy(box[8:12], "qt  ")
	return append(box, make([]byte, 64)...)
}

// sha256Hex digests data independently of the receiver under test.
func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// requireUploadedFileStored verifies the content-addressed original of
// an asset row exists on disk under the environment's data root and
// holds exactly the uploaded bytes.
func (e *assetEnv) requireUploadedFileStored(t *testing.T, row dbgen.Asset, want []byte) {
	t.Helper()
	path := filepath.Join(e.dataDir, filepath.FromSlash(row.FilePath))
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read stored original %s: %v", path, err)
	}
	if !bytes.Equal(data, want) {
		t.Errorf("stored original %s holds %d bytes, want the %d uploaded bytes", path, len(data), len(want))
	}
}

// requireOriginalsEmpty verifies no file landed anywhere in the
// originals tree of the environment's data root.
func (e *assetEnv) requireOriginalsEmpty(t *testing.T) {
	t.Helper()
	if got := e.countFiles(t, filepath.Join(e.dataDir, "originals")); got != 0 {
		t.Errorf("originals tree holds %d files, want 0", got)
	}
}

// requireStagingEmpty verifies the upload staging directory carries no
// residue.
func (e *assetEnv) requireStagingEmpty(t *testing.T) {
	t.Helper()
	dir := filepath.Join(e.dataDir, "tmp")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read staging %s: %v", dir, err)
	}
	if len(entries) != 0 {
		t.Errorf("staging holds %d entries after the rejected upload, want 0", len(entries))
	}
}

// countFiles counts the regular files under root.
func (e *assetEnv) countFiles(t *testing.T, root string) int {
	t.Helper()
	count := 0
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.Type().IsRegular() {
			count++
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", root, err)
	}
	return count
}

// awaitAssetEvent scans the bus for the asset.uploaded event of one
// asset; subscribing before the request that publishes it keeps the
// assertion honest.
func awaitAssetEvent(t *testing.T, bus *events.Bus, assetID string) events.Event {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	subscription := bus.Subscribe(ctx, 0)
	for {
		event, ok := subscription.Next()
		if !ok {
			t.Fatalf("no asset.uploaded event for asset %s reached the bus", assetID)
		}
		if event.Name != "asset.uploaded" {
			continue
		}
		if eventPayload(t, event)["assetId"] == assetID {
			return event
		}
	}
}

// countAssetUploadedEvents observes the bus for a short window and
// counts the asset.uploaded events on it.
func countAssetUploadedEvents(t *testing.T, bus *events.Bus) int {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), assetEventWait)
	defer cancel()
	subscription := bus.Subscribe(ctx, 0)
	count := 0
	for {
		event, ok := subscription.Next()
		if !ok {
			return count
		}
		if event.Name == "asset.uploaded" {
			count++
		}
	}
}

// TestUploadAssetCreateState walks the created state of the upload
// idempotency end to end: the 201 response, the asset row with its
// content-addressed file on disk, the EXIF placeholder sidecar, and
// the asset.uploaded event on the activity bus.
func TestUploadAssetCreateState(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	payload := jpegFixture(t)
	deviceCreatedAt := "2026-01-02T15:04:05+08:00"
	wantTakenAt, err := time.Parse(time.RFC3339, deviceCreatedAt)
	if err != nil {
		t.Fatalf("parse deviceCreatedAt fixture: %v", err)
	}

	// The subscription opens before the request: the assertion must
	// observe the publish, not reconstruct it afterwards.
	subCtx, subCancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer subCancel()
	subscription := env.bus.Subscribe(subCtx, 0)

	rec := env.postUpload(t, map[string]string{
		"deviceAssetId":   "device-1",
		"fileName":        "IMG_1234.jpg",
		"deviceCreatedAt": deviceCreatedAt,
		"deviceTimeZone":  "Asia/Shanghai",
	}, []assetUploadFile{{field: "photo", name: "IMG_1234.jpg", content: payload}})

	if rec.Code != http.StatusCreated {
		t.Fatalf("upload status = %d, want %d (body %q)", rec.Code, http.StatusCreated, rec.Body.String())
	}
	asset := decodeAsset(t, rec)

	// The response describes the stored asset.
	checksum := sha256Hex(payload)
	if asset.State != apigen.Created {
		t.Errorf("state = %q, want %q", asset.State, apigen.Created)
	}
	if asset.Kind != apigen.AssetResponseKindImage {
		t.Errorf("kind = %q, want %q", asset.Kind, apigen.AssetResponseKindImage)
	}
	if asset.Checksum != checksum {
		t.Errorf("checksum = %q, want %q", asset.Checksum, checksum)
	}
	if asset.OriginalFilename != "IMG_1234.jpg" {
		t.Errorf("originalFilename = %q, want %q", asset.OriginalFilename, "IMG_1234.jpg")
	}
	if asset.FileSize != int64(len(payload)) {
		t.Errorf("fileSize = %d, want %d", asset.FileSize, len(payload))
	}
	if asset.MimeType != "image/jpeg" {
		t.Errorf("mimeType = %q, want %q", asset.MimeType, "image/jpeg")
	}
	if asset.IsTrashed {
		t.Error("isTrashed = true, want false")
	}
	if asset.Id.Version() != 7 {
		t.Errorf("id = %s, want a UUIDv7", asset.Id)
	}
	if !asset.TakenAt.Truncate(time.Microsecond).Equal(wantTakenAt) {
		t.Errorf("takenAt = %v, want the deviceCreatedAt %v", asset.TakenAt, wantTakenAt)
	}
	if asset.CreatedAt.IsZero() {
		t.Error("createdAt = zero, want the stored creation time")
	}

	// The asset row exists with the pinned server-side column values.
	row := env.assetByChecksum(t, checksum)
	if row.ID.Bytes != asset.Id {
		t.Errorf("row id = %s, want the response id %s", uuid.UUID(row.ID.Bytes), asset.Id)
	}
	if row.MediaKind != dbgen.MediaKindImage {
		t.Errorf("row media_kind = %q, want %q", row.MediaKind, dbgen.MediaKindImage)
	}
	if row.StorageKind != dbgen.StorageKindBlob {
		t.Errorf("row storage_kind = %q, want %q", row.StorageKind, dbgen.StorageKindBlob)
	}
	if row.TakenAtSource != dbgen.TakenAtSourceImport {
		t.Errorf("row taken_at_source = %q, want %q", row.TakenAtSource, dbgen.TakenAtSourceImport)
	}
	if !row.TakenAt.Time.Truncate(time.Microsecond).Equal(wantTakenAt) {
		t.Errorf("row taken_at = %v, want %v", row.TakenAt.Time, wantTakenAt)
	}
	if row.Width != 0 || row.Height != 0 {
		t.Errorf("row width/height = %d/%d, want 0/0 (the metadata job refines dimensions)", row.Width, row.Height)
	}
	if !row.DeviceAssetID.Valid || row.DeviceAssetID.String != "device-1" {
		t.Errorf("row device_asset_id = %+v, want device-1", row.DeviceAssetID)
	}
	if row.IsComponent {
		t.Error("row is_component = true, want false")
	}

	// The file lands at the template-rendered path under the owner's
	// originals root: the default template organizes by the capture
	// wall clock (the device-reported +08:00 time of this fixture is
	// January 2026 in its own zone), and the file name carries the
	// sanitized stem plus the normalized extension.
	wantPath := "originals/" + env.sessionUserID(t).String() + "/2026/01/IMG_1234.jpg"
	if row.FilePath != wantPath {
		t.Errorf("row file_path = %q, want %q", row.FilePath, wantPath)
	}
	env.requireUploadedFileStored(t, row, payload)

	// The EXIF placeholder row exists and carries the device-side
	// upload fields for the metadata job.
	if got := env.queryInt(t, "SELECT count(*) FROM exif WHERE asset_id = $1", row.ID); got != 1 {
		t.Fatalf("exif placeholder rows = %d, want 1", got)
	}
	if got := env.queryText(t, "SELECT exif->>'deviceTimeZone' FROM exif WHERE asset_id = $1", row.ID); got != "Asia/Shanghai" {
		t.Errorf("exif placeholder deviceTimeZone = %q, want %q", got, "Asia/Shanghai")
	}
	if got := env.queryText(t, "SELECT exif->>'deviceCreatedAt' FROM exif WHERE asset_id = $1", row.ID); got != deviceCreatedAt {
		t.Errorf("exif placeholder deviceCreatedAt = %q, want %q", got, deviceCreatedAt)
	}

	// The event reached the bus with the minimal payload.
	for {
		event, ok := subscription.Next()
		if !ok {
			t.Fatal("no asset.uploaded event reached the bus within the subscription window")
		}
		if event.Name != "asset.uploaded" {
			continue
		}
		got := eventPayload(t, event)
		if got["assetId"] != asset.Id.String() {
			t.Errorf("event assetId = %q, want %q", got["assetId"], asset.Id)
		}
		if got["kind"] != "image" {
			t.Errorf("event kind = %q, want image", got["kind"])
		}
		break
	}
}

// TestUploadAssetDuplicateState verifies the idempotent second state:
// re-uploading identical bytes answers 200 duplicate with the existing
// asset and never creates a second row.
func TestUploadAssetDuplicateState(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	payload := jpegFixture(t)
	fields := map[string]string{
		"deviceAssetId": "device-1",
		"fileName":      "IMG_1234.jpg",
	}
	files := []assetUploadFile{{field: "photo", name: "IMG_1234.jpg", content: payload}}

	first := env.postUpload(t, fields, files)
	if first.Code != http.StatusCreated {
		t.Fatalf("first upload status = %d, want %d (body %q)", first.Code, http.StatusCreated, first.Body.String())
	}
	created := decodeAsset(t, first)

	// The retry carries the same device identity, like a background
	// downloader redelivering the file.
	second := env.postUpload(t, fields, files)
	if second.Code != http.StatusOK {
		t.Fatalf("retry status = %d, want %d (body %q)", second.Code, http.StatusOK, second.Body.String())
	}
	duplicate := decodeAsset(t, second)
	if duplicate.State != apigen.Duplicate {
		t.Errorf("state = %q, want %q", duplicate.State, apigen.Duplicate)
	}
	if duplicate.Id != created.Id {
		t.Errorf("retry id = %s, want the existing asset %s", duplicate.Id, created.Id)
	}
	if duplicate.IsTrashed {
		t.Error("retry isTrashed = true, want false")
	}
	// A plain live duplicate has nothing to tell: the notice is for a
	// row that stopped listing on its own (a Live Photo's motion).
	if duplicate.Notices != nil && len(*duplicate.Notices) != 0 {
		t.Errorf("notices = %v, want none", *duplicate.Notices)
	}

	if got := env.queryInt(t, "SELECT count(*) FROM assets"); got != 1 {
		t.Errorf("assets rows = %d after the retry, want 1", got)
	}
}

// TestUploadAssetTrashedState verifies the third state: a re-upload of
// content whose asset sits in the trash answers 200 trashed with that
// asset, without inserting a row.
func TestUploadAssetTrashedState(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	payload := jpegFixture(t)
	fields := map[string]string{
		"deviceAssetId": "device-1",
		"fileName":      "IMG_1234.jpg",
	}
	files := []assetUploadFile{{field: "photo", name: "IMG_1234.jpg", content: payload}}

	created := decodeAsset(t, env.postUpload(t, fields, files))

	// The trash flow is another surface's work; the test drives the
	// soft-delete directly, which is the state it produces.
	env.exec(t, "UPDATE assets SET deleted_at = now() WHERE id = $1", created.Id)

	reupload := env.postUpload(t, fields, files)
	if reupload.Code != http.StatusOK {
		t.Fatalf("re-upload status = %d, want %d (body %q)", reupload.Code, http.StatusOK, reupload.Body.String())
	}
	trashed := decodeAsset(t, reupload)
	if trashed.State != apigen.Trashed {
		t.Errorf("state = %q, want %q", trashed.State, apigen.Trashed)
	}
	if trashed.Id != created.Id {
		t.Errorf("re-upload id = %s, want the trashed asset %s", trashed.Id, created.Id)
	}
	if !trashed.IsTrashed {
		t.Error("re-upload isTrashed = false, want true")
	}

	// The idempotency must hold row-wise too: the database constraint,
	// not a handler check, keeps the trashed content single-rowed.
	if got := env.queryInt(t, "SELECT count(*) FROM assets"); got != 1 {
		t.Errorf("assets rows = %d after the trashed re-upload, want 1", got)
	}
	// No second asset.uploaded fired for the trashed hit.
	if got := countAssetUploadedEvents(t, env.bus); got != 1 {
		t.Errorf("asset.uploaded events = %d, want 1 (only the original upload)", got)
	}
}

// TestUploadAssetPositionRows pins the position model's upload arm
// (the owner's second ruling): the content lookup walks the FULL
// domain, so an upload of bytes a directory album already mirrors
// answers duplicate with nothing placed -- the upload direction never
// creates a second row -- while an orphan-marked source row (the file
// vanished, the row is dead and no trash copy exists) is content the
// owner no longer holds, and the upload replaces it with a fresh blob
// row.
func TestUploadAssetPositionRows(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	payload := jpegFixture(t)
	fields := map[string]string{
		"deviceAssetId": "device-position-1",
		"fileName":      "IMG_POSITION.jpg",
	}
	files := []assetUploadFile{{field: "photo", name: "IMG_POSITION.jpg", content: payload}}
	ownerID := env.sessionUserID(t)
	checksum := sha256Hex(payload)

	// The source row a scan would have landed: an external row at its
	// real path, carrying the upload's content.
	env.exec(t, `INSERT INTO assets (
			id, owner_id, storage_kind, file_path, checksum, original_filename,
			file_size, mime_type, media_kind, width, height, taken_at,
			taken_at_source, pending_jobs
		) VALUES ($1, $2, 'external', $3, $4, 'IMG_POSITION.jpg', $5, 'image/jpeg',
			'image', 0, 0, now(), 'file', 0)`,
		pgUUID(uuid.Must(uuid.NewV7())), pgUUID(ownerID),
		"/mnt/source/IMG_POSITION.jpg", checksum, int64(len(payload)))

	// The live source copy answers duplicate: no new row lands, and
	// nothing is placed under the owner's originals.
	rec := env.postUpload(t, fields, files)
	if rec.Code != http.StatusOK {
		t.Fatalf("upload status = %d, want %d (body %q)", rec.Code, http.StatusOK, rec.Body.String())
	}
	live := decodeAsset(t, rec)
	if live.State != apigen.Duplicate {
		t.Errorf("state = %q, want %q (the content lives in the source row)", live.State, apigen.Duplicate)
	}
	if got := env.queryInt(t, "SELECT count(*) FROM assets WHERE owner_id = $1 AND storage_kind = 'blob'", pgUUID(ownerID)); got != 0 {
		t.Errorf("blob rows = %d, want 0 (the upload never lands beside live content)", got)
	}
	env.requireOriginalsEmpty(t)

	// The source file vanishes: the scan's orphan mark leaves the row
	// dead, with no trash copy -- a state no restore can reach.
	env.exec(t, "UPDATE assets SET deleted_at = now() WHERE owner_id = $1 AND storage_kind = 'external'", pgUUID(ownerID))

	// The upload now REPLACES the dead row: the fresh blob row lands
	// (created), and the orphan mark is gone.
	rec = env.postUpload(t, fields, files)
	if rec.Code != http.StatusCreated {
		t.Fatalf("upload over the orphan mark status = %d, want %d (body %q)", rec.Code, http.StatusCreated, rec.Body.String())
	}
	created := decodeAsset(t, rec)
	if created.State != apigen.Created {
		t.Errorf("state = %q, want %q", created.State, apigen.Created)
	}
	if got := env.queryInt(t, "SELECT count(*) FROM assets WHERE owner_id = $1 AND storage_kind = 'external'", pgUUID(ownerID)); got != 0 {
		t.Errorf("external rows = %d, want the orphan mark replaced", got)
	}
	row := env.assetByChecksum(t, checksum)
	if row.StorageKind != dbgen.StorageKindBlob {
		t.Errorf("the content's row kind = %q, want blob after the replace", row.StorageKind)
	}
	env.requireUploadedFileStored(t, row, payload)

	// And the replacement is in "My photos": the sync membership
	// landed with it.
	if got := env.queryInt(t, `SELECT count(*) FROM album_items ai
		JOIN albums al ON al.id = ai.album_id
		WHERE al.type = 'sync' AND al.owner_id = $1 AND ai.asset_id = $2`,
		pgUUID(ownerID), row.ID); got != 1 {
		t.Errorf("sync memberships of the replacement = %d, want 1", got)
	}
}

// TestUploadAssetConcurrent mirrors the setup concurrency pattern:
// two uploads of identical bytes, genuinely in flight at the same
// time, must resolve through the database constraint into exactly one
// created asset and one duplicate answer.
func TestUploadAssetConcurrent(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	payload := jpegFixture(t)

	// The two devices back up the same photo under their own device
	// identities, so only the checksum races.
	bodies := make([]*bytes.Buffer, 2)
	types := make([]string, 2)
	for i := range bodies {
		bodies[i], types[i] = buildUploadBody(t, map[string]string{
			"deviceAssetId": "device-" + string(rune('a'+i)),
			"fileName":      "IMG_1234.jpg",
		}, []assetUploadFile{{field: "photo", name: "IMG_1234.jpg", content: payload}})
	}

	// The barrier releases both goroutines only after they exist, so
	// the two requests are genuinely in flight at the same time.
	start := make(chan struct{})
	codes := make([]int, len(bodies))
	var wg sync.WaitGroup
	for i := range bodies {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			req := httptest.NewRequest(http.MethodPost, "/api/v1/assets", bodies[i])
			req.Header.Set("Content-Type", types[i])
			req.AddCookie(env.cookie)
			rec := httptest.NewRecorder()
			env.router.ServeHTTP(rec, req)
			codes[i] = rec.Code
		}(i)
	}
	close(start)
	wg.Wait()

	created, duplicate := 0, 0
	for _, code := range codes {
		switch code {
		case http.StatusCreated:
			created++
		case http.StatusOK:
			duplicate++
		default:
			t.Errorf("concurrent upload status = %d, want %d or %d", code, http.StatusCreated, http.StatusOK)
		}
	}
	if created != 1 || duplicate != 1 {
		t.Errorf("concurrent upload outcomes = %d created / %d duplicate, want exactly 1/1 (codes %v)", created, duplicate, codes)
	}
	if got := env.queryInt(t, "SELECT count(*) FROM assets"); got != 1 {
		t.Errorf("assets rows = %d after concurrent uploads, want 1", got)
	}
}

// TestUploadAssetLivePhoto verifies the pairing: one request with
// photo and liveVideo commits both files, inserts the video as a
// linked component row (never in the timeline), links the photo to
// it, and fires exactly one asset.uploaded event -- for the photo.
// The component row counts ONE pipeline job of its own (the
// transcode -- the playable stream of the pair's motion), while the
// photo row keeps its unchanged two.
func TestUploadAssetLivePhoto(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	photo := jpegFixture(t)
	video := movFixture()

	rec := env.postUpload(t, map[string]string{
		"deviceAssetId":   "device-live-1",
		"fileName":        "IMG_2000.jpg",
		"deviceCreatedAt": "2026-02-03T10:11:12Z",
	}, []assetUploadFile{
		{field: "photo", name: "IMG_2000.jpg", content: photo},
		{field: "liveVideo", name: "IMG_2000.mov", content: video},
	})

	if rec.Code != http.StatusCreated {
		t.Fatalf("live photo upload status = %d, want %d (body %q)", rec.Code, http.StatusCreated, rec.Body.String())
	}
	asset := decodeAsset(t, rec)
	if asset.Kind != apigen.AssetResponseKindImage {
		t.Errorf("response kind = %q, want %q (the photo is the asset)", asset.Kind, apigen.AssetResponseKindImage)
	}

	// The photo row links to the video row, which is a component
	// without a device identity of its own.
	photoRow := env.assetByChecksum(t, sha256Hex(photo))
	if !photoRow.LivePhotoVideoID.Valid {
		t.Fatalf("photo row live_photo_video_id is NULL, want the video component")
	}
	videoRow, err := env.queries.GetAssetByOwnerChecksum(context.Background(), dbgen.GetAssetByOwnerChecksumParams{
		OwnerID:  pgUUID(env.sessionUserID(t)),
		Checksum: sha256Hex(video),
	})
	if err != nil {
		t.Fatalf("load video component row: %v", err)
	}
	if videoRow.ID != photoRow.LivePhotoVideoID {
		t.Errorf("photo links to asset %s, want the video row %s", photoRow.LivePhotoVideoID, videoRow.ID)
	}
	if !videoRow.IsComponent {
		t.Error("video row is_component = false, want true (never in the timeline)")
	}
	if videoRow.MediaKind != dbgen.MediaKindVideo {
		t.Errorf("video row media_kind = %q, want %q", videoRow.MediaKind, dbgen.MediaKindVideo)
	}
	if videoRow.PendingJobs != 1 {
		t.Errorf("video row pending_jobs = %d, want 1 (the transcode job alone)", videoRow.PendingJobs)
	}
	if photoRow.PendingJobs != 2 {
		t.Errorf("photo row pending_jobs = %d, want the unchanged 2 (metadata + thumbnail)", photoRow.PendingJobs)
	}
	if videoRow.DeviceAssetID.Valid {
		t.Errorf("video row device_asset_id = %q, want NULL (components carry no device identity)", videoRow.DeviceAssetID.String)
	}
	if videoRow.OriginalFilename != "IMG_2000.mov" {
		t.Errorf("video row original_filename = %q, want %q", videoRow.OriginalFilename, "IMG_2000.mov")
	}
	if videoRow.Checksum != sha256Hex(video) {
		t.Errorf("video row checksum = %q, want %q", videoRow.Checksum, sha256Hex(video))
	}
	if videoRow.MimeType != "video/quicktime" {
		t.Errorf("video row mime_type = %q, want video/quicktime", videoRow.MimeType)
	}

	// Both files are stored content-addressed.
	env.requireUploadedFileStored(t, photoRow, photo)
	env.requireUploadedFileStored(t, videoRow, video)

	// Exactly one event, for the photo asset.
	if got := countAssetUploadedEvents(t, env.bus); got != 1 {
		t.Errorf("asset.uploaded events = %d, want 1 (the component stays silent)", got)
	}
	event := awaitAssetEvent(t, env.bus, uuid.UUID(photoRow.ID.Bytes).String())
	if payload := eventPayload(t, event); payload["kind"] != "image" {
		t.Errorf("event kind = %q, want image", payload["kind"])
	}

	// Both rows carry their metadata sidecar for the pipeline.
	if got := env.queryInt(t, "SELECT count(*) FROM exif WHERE asset_id IN ($1, $2)",
		photoRow.ID, videoRow.ID); got != 2 {
		t.Errorf("exif placeholder rows = %d, want 2", got)
	}
}

// TestUploadAssetRejectsDisguisedFile verifies the content gate: a
// text file renamed to .jpg answers 400 and leaves nothing behind --
// no file in staging or originals, no database row, no event.
func TestUploadAssetRejectsDisguisedFile(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)

	rec := env.postUpload(t, map[string]string{
		"deviceAssetId": "device-evil",
		"fileName":      "evil.jpg",
	}, []assetUploadFile{{field: "photo", name: "evil.jpg", content: []byte("This is definitely not a JPEG.")}})

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("disguised upload status = %d, want %d (body %q)", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
	apiErr, fields := decodeError(t, rec)
	if apiErr.Code != apigen.VALIDATIONFAILED {
		t.Errorf("code = %q, want %q", apiErr.Code, apigen.VALIDATIONFAILED)
	}
	if !contains(fields, "photo") {
		t.Errorf("details.fields = %v, want it to contain %q", fields, "photo")
	}

	env.requireOriginalsEmpty(t)
	env.requireStagingEmpty(t)
	if got := env.queryInt(t, "SELECT count(*) FROM assets"); got != 0 {
		t.Errorf("assets rows = %d, want 0", got)
	}
	if got := countAssetUploadedEvents(t, env.bus); got != 0 {
		t.Errorf("asset.uploaded events = %d, want 0", got)
	}
}

// TestUploadAssetOverLimit verifies the per-file limit: an upload
// crossing the configured cap aborts mid-stream with 413
// ASSET_TOO_LARGE and leaves nothing behind.
func TestUploadAssetOverLimit(t *testing.T) {
	env := newAssetEnv(t, 512)
	payload := append(jpegFixture(t), make([]byte, 2048)...)

	rec := env.postUpload(t, map[string]string{
		"deviceAssetId": "device-big",
		"fileName":      "big.jpg",
	}, []assetUploadFile{{field: "photo", name: "big.jpg", content: payload}})

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("over-limit upload status = %d, want %d (body %q)", rec.Code, http.StatusRequestEntityTooLarge, rec.Body.String())
	}
	apiErr, _ := decodeError(t, rec)
	if apiErr.Code != apigen.ASSETTOOLARGE {
		t.Errorf("code = %q, want %q", apiErr.Code, apigen.ASSETTOOLARGE)
	}

	env.requireOriginalsEmpty(t)
	if got := env.queryInt(t, "SELECT count(*) FROM assets"); got != 0 {
		t.Errorf("assets rows = %d, want 0", got)
	}
}

// TestUploadAssetRequiresSession verifies the upload route sits
// behind the session middleware despite its timeout exemption.
func TestUploadAssetRequiresSession(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	env.cookie = nil

	rec := env.postUpload(t, map[string]string{
		"deviceAssetId": "device-1",
		"fileName":      "IMG_1234.jpg",
	}, []assetUploadFile{{field: "photo", name: "IMG_1234.jpg", content: jpegFixture(t)}})

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated upload status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
	apiErr, _ := decodeError(t, rec)
	if apiErr.Code != apigen.AUTHINVALIDCREDENTIALS {
		t.Errorf("code = %q, want %q", apiErr.Code, apigen.AUTHINVALIDCREDENTIALS)
	}
}

// failingEnqueuer rejects every enqueue, standing in for a broken
// queue behind the upload endpoint's job seam.
type failingEnqueuer struct{}

// EnqueueMetadata implements AssetJobEnqueuer by always failing.
func (failingEnqueuer) EnqueueMetadata(context.Context, uuid.UUID) error {
	return errors.New("queue unavailable")
}

// EnqueueThumbnail implements AssetJobEnqueuer by always failing.
func (failingEnqueuer) EnqueueThumbnail(context.Context, uuid.UUID) error {
	return errors.New("queue unavailable")
}

// EnqueueTranscode implements AssetJobEnqueuer by always failing.
func (failingEnqueuer) EnqueueTranscode(context.Context, uuid.UUID) error {
	return errors.New("queue unavailable")
}

// TestUploadAssetEnqueueFailureStillCreated pins the enqueue seam's
// failure semantics: the enqueue happens AFTER the upload transaction
// commits, so a failing queue cannot fail the response -- the upload
// succeeded, its event still fires, and the asset simply stays
// not-ready (its pending pipeline job never started).
func TestUploadAssetEnqueueFailureStillCreated(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	env.router = NewRouter(&Deps{
		Pool:             env.pool,
		Version:          testVersion,
		Events:           env.bus,
		DataDir:          env.dataDir,
		UploadLimitBytes: testUploadLimit,
		Jobs:             failingEnqueuer{},
	})

	content := jpegFixture(t)
	rec := env.postUpload(t, map[string]string{
		"deviceAssetId": "device-noqueue",
		"fileName":      "IMG_noqueue.jpg",
	}, []assetUploadFile{{field: "photo", name: "IMG_noqueue.jpg", content: content}})
	if rec.Code != http.StatusCreated {
		t.Fatalf("upload with failing queue status = %d, want %d (body %q)", rec.Code, http.StatusCreated, rec.Body.String())
	}

	asset := env.assetByChecksum(t, sha256Hex(content))
	if asset.PendingJobs != 2 {
		t.Errorf("pending_jobs = %d, want the enqueued-but-never-run 2", asset.PendingJobs)
	}
	if asset.ReadyAt.Valid {
		t.Error("ready_at is set without a pipeline run, want NULL")
	}
	if got := countAssetUploadedEvents(t, env.bus); got != 1 {
		t.Errorf("asset.uploaded events = %d, want 1: the upload itself succeeded", got)
	}
}

// recordingEnqueuer records every enqueue it is asked to perform and
// succeeds at all of them, standing in for a healthy queue behind the
// upload endpoint's job seam.
type recordingEnqueuer struct {
	metadataContexts  []context.Context
	metadataIDs       []uuid.UUID
	thumbnailContext  context.Context
	thumbnailID       uuid.UUID
	transcodeContexts []context.Context
	transcodeIDs      []uuid.UUID
}

// EnqueueMetadata implements AssetJobEnqueuer by recording the call.
func (r *recordingEnqueuer) EnqueueMetadata(ctx context.Context, id uuid.UUID) error {
	r.metadataContexts = append(r.metadataContexts, ctx)
	r.metadataIDs = append(r.metadataIDs, id)
	return nil
}

// EnqueueThumbnail implements AssetJobEnqueuer by recording the call.
func (r *recordingEnqueuer) EnqueueThumbnail(ctx context.Context, id uuid.UUID) error {
	r.thumbnailContext = ctx
	r.thumbnailID = id
	return nil
}

// EnqueueTranscode implements AssetJobEnqueuer by recording the call.
func (r *recordingEnqueuer) EnqueueTranscode(ctx context.Context, id uuid.UUID) error {
	r.transcodeContexts = append(r.transcodeContexts, ctx)
	r.transcodeIDs = append(r.transcodeIDs, id)
	return nil
}

// TestEnqueueAssetJobsSurvivesClientDisconnect pins the disconnect half
// of the enqueue seam's semantics: the pipeline jobs are enqueued on
// the request context with its cancellation removed, so a client that
// vanishes in the commit-to-enqueue window cannot fail the enqueue and
// strand the asset's pending-jobs counter forever. The Live Photo arm
// pins the component enqueue: a video-family liveVideo part adds
// exactly one transcode job -- for the COMPONENT id -- on the same
// disconnect-safe context, while an image-family part adds none (the
// enqueue mirrors the pending_jobs initialization).
func TestEnqueueAssetJobsSurvivesClientDisconnect(t *testing.T) {
	assetID := uuid.Must(uuid.NewV7())
	recorder := &recordingEnqueuer{}
	s := &server{
		jobs:   recorder,
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	// The client is already gone: the request context is canceled.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/assets", nil).WithContext(ctx)

	s.enqueueAssetJobs(req, dbgen.Asset{ID: pgUUID(assetID)}, nil, nil)

	if len(recorder.metadataIDs) != 1 || recorder.metadataIDs[0] != assetID {
		t.Fatalf("metadata enqueues = %v, want exactly [%s]", recorder.metadataIDs, assetID)
	}
	if recorder.thumbnailID != assetID {
		t.Errorf("thumbnail enqueue id = %s, want %s", recorder.thumbnailID, assetID)
	}
	if len(recorder.transcodeIDs) != 0 {
		t.Errorf("transcode enqueues = %v, want none for an image-kind row", recorder.transcodeIDs)
	}
	contexts := append(append([]context.Context(nil), recorder.metadataContexts...), recorder.thumbnailContext)
	for i, ctx := range contexts {
		if err := ctx.Err(); err != nil {
			t.Errorf("enqueue context %d err = %v, want nil: the client disconnect must not cancel the enqueue", i, err)
		}
	}

	// A standalone video upload adds the transcode job to the same
	// disconnect-safe context: its counter counts three jobs.
	videoRecorder := &recordingEnqueuer{}
	s.jobs = videoRecorder
	s.enqueueAssetJobs(req, dbgen.Asset{ID: pgUUID(assetID), MediaKind: dbgen.MediaKindVideo}, nil, nil)
	if len(videoRecorder.transcodeIDs) != 1 || videoRecorder.transcodeIDs[0] != assetID {
		t.Errorf("video transcode enqueues = %v, want exactly [%s]", videoRecorder.transcodeIDs, assetID)
	}
	if err := videoRecorder.transcodeContexts[0].Err(); err != nil {
		t.Errorf("video transcode enqueue context err = %v, want nil: the disconnect must not cancel it", err)
	}
	if len(videoRecorder.metadataIDs) != 1 || videoRecorder.thumbnailID != assetID {
		t.Errorf("video enqueue set = metadata %v thumbnail %s, want one of each for %s",
			videoRecorder.metadataIDs, videoRecorder.thumbnailID, assetID)
	}

	// A Live Photo: the metadata loop covers BOTH rows, the thumbnail
	// stays with the photo, and the video-family liveVideo part adds
	// exactly one transcode -- for the component id (the photo row is
	// image-kind and gets none of its own).
	componentID := uuid.Must(uuid.NewV7())
	liveRecorder := &recordingEnqueuer{}
	s.jobs = liveRecorder
	s.enqueueAssetJobs(req, dbgen.Asset{
		ID:               pgUUID(assetID),
		LivePhotoVideoID: pgUUID(componentID),
	}, &sealedUpload{format: storage.UploadFormat{Video: true}}, nil)
	if len(liveRecorder.metadataIDs) != 2 {
		t.Errorf("live photo metadata enqueues = %v, want one per row (photo and component)", liveRecorder.metadataIDs)
	}
	seen := map[uuid.UUID]bool{}
	for _, id := range liveRecorder.metadataIDs {
		seen[id] = true
	}
	if !seen[assetID] || !seen[componentID] {
		t.Errorf("live photo metadata enqueues = %v, want the photo %s and the component %s",
			liveRecorder.metadataIDs, assetID, componentID)
	}
	if liveRecorder.thumbnailID != assetID {
		t.Errorf("live photo thumbnail enqueue id = %s, want the photo %s", liveRecorder.thumbnailID, assetID)
	}
	if len(liveRecorder.transcodeIDs) != 1 || liveRecorder.transcodeIDs[0] != componentID {
		t.Errorf("live photo transcode enqueues = %v, want exactly the component [%s]",
			liveRecorder.transcodeIDs, componentID)
	}
	if err := liveRecorder.transcodeContexts[0].Err(); err != nil {
		t.Errorf("component transcode enqueue context err = %v, want nil: the disconnect must not cancel it", err)
	}

	// The gate mirrors the counter: an image-family liveVideo part
	// counts no transcode, so none is enqueued for it either.
	imagePartRecorder := &recordingEnqueuer{}
	s.jobs = imagePartRecorder
	s.enqueueAssetJobs(req, dbgen.Asset{
		ID:               pgUUID(assetID),
		LivePhotoVideoID: pgUUID(componentID),
	}, &sealedUpload{}, nil)
	if len(imagePartRecorder.transcodeIDs) != 0 {
		t.Errorf("image-family component transcode enqueues = %v, want none (no job can run for it)",
			imagePartRecorder.transcodeIDs)
	}

	// An ABSORBED motion gets nothing: the reused row already ran its
	// whole pipeline, so the upload's queue is the photo's jobs alone
	// -- no component metadata, no component transcode.
	absorbedRow := dbgen.Asset{ID: pgUUID(componentID)}
	absorbedRecorder := &recordingEnqueuer{}
	s.jobs = absorbedRecorder
	s.enqueueAssetJobs(req, dbgen.Asset{
		ID:               pgUUID(assetID),
		LivePhotoVideoID: pgUUID(componentID),
	}, &sealedUpload{format: storage.UploadFormat{Video: true}}, &absorbedRow)
	if len(absorbedRecorder.metadataIDs) != 1 || absorbedRecorder.metadataIDs[0] != assetID {
		t.Errorf("absorbed motion metadata enqueues = %v, want the photo alone", absorbedRecorder.metadataIDs)
	}
	if absorbedRecorder.thumbnailID != assetID {
		t.Errorf("absorbed motion thumbnail enqueue id = %s, want the photo %s", absorbedRecorder.thumbnailID, assetID)
	}
	if len(absorbedRecorder.transcodeIDs) != 0 {
		t.Errorf("absorbed motion transcode enqueues = %v, want none (its pipeline already ran)",
			absorbedRecorder.transcodeIDs)
	}
}

// TestUploadAssetValidatesStructure walks the structural contract of
// the multipart body: required fields, unknown parts, duplicate file
// parts, and value validation. None of the rejected bodies may stage
// a file or insert a row.
func TestUploadAssetValidatesStructure(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	payload := jpegFixture(t)

	cases := []struct {
		name   string
		fields map[string]string
		files  []assetUploadFile
	}{
		{
			name: "MissingPhoto",
			fields: map[string]string{
				"deviceAssetId": "device-1",
				"fileName":      "IMG_1234.jpg",
			},
		},
		{
			name: "MissingDeviceAssetId",
			fields: map[string]string{
				"fileName": "IMG_1234.jpg",
			},
			files: []assetUploadFile{{field: "photo", name: "IMG_1234.jpg", content: payload}},
		},
		{
			name: "MissingFileName",
			fields: map[string]string{
				"deviceAssetId": "device-1",
			},
			files: []assetUploadFile{{field: "photo", name: "IMG_1234.jpg", content: payload}},
		},
		{
			name: "UnparseableDeviceCreatedAt",
			fields: map[string]string{
				"deviceAssetId":   "device-1",
				"fileName":        "IMG_1234.jpg",
				"deviceCreatedAt": "not-a-timestamp",
			},
			files: []assetUploadFile{{field: "photo", name: "IMG_1234.jpg", content: payload}},
		},
		{
			name: "DeviceAssetIdTooLong",
			fields: map[string]string{
				"deviceAssetId": strings.Repeat("x", 257),
				"fileName":      "IMG_1234.jpg",
			},
			files: []assetUploadFile{{field: "photo", name: "IMG_1234.jpg", content: payload}},
		},
		{
			name: "EmptyDeviceAssetId",
			fields: map[string]string{
				"deviceAssetId": "   ",
				"fileName":      "IMG_1234.jpg",
			},
			files: []assetUploadFile{{field: "photo", name: "IMG_1234.jpg", content: payload}},
		},
		{
			// The pairing video must BE a video: an image-family part
			// would build a component nothing can play, so it is
			// refused loudly (the link's own "is a video" check).
			name: "ImageFamilyLiveVideoIsRejected",
			fields: map[string]string{
				"deviceAssetId": "device-2",
				"fileName":      "IMG_1235.jpg",
			},
			files: []assetUploadFile{
				{field: "photo", name: "IMG_1235.jpg", content: payload},
				{field: "liveVideo", name: "IMG_1235.jpg", content: jpegColorFixture(t, color.RGBA{R: 0x22, G: 0x33, B: 0x44, A: 0xff})},
			},
		},
		{
			// The pairing video must follow its photo: an early
			// liveVideo is refused before anything is staged.
			name: "LiveVideoBeforePhotoIsRejected",
			fields: map[string]string{
				"deviceAssetId": "device-1",
				"fileName":      "IMG_1234.jpg",
			},
			files: []assetUploadFile{{field: "liveVideo", name: "IMG_1234.mov", content: movFixture()}},
		},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			rec := env.postUpload(t, testCase.fields, testCase.files)

			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d (body %q)", rec.Code, http.StatusBadRequest, rec.Body.String())
			}
			apiErr, _ := decodeError(t, rec)
			if apiErr.Code != apigen.VALIDATIONFAILED {
				t.Errorf("code = %q, want %q", apiErr.Code, apigen.VALIDATIONFAILED)
			}
		})
	}

	// Unknown parts are refused outright.
	t.Run("UnknownPart", func(t *testing.T) {
		body, contentType := buildUploadBody(t, map[string]string{
			"deviceAssetId": "device-1",
			"fileName":      "IMG_1234.jpg",
		}, []assetUploadFile{{field: "evil", name: "evil.txt", content: []byte("x")}})
		rec := env.postUploadRaw(t, body, contentType)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d (body %q)", rec.Code, http.StatusBadRequest, rec.Body.String())
		}
	})

	// A body that is not multipart at all is refused before anything
	// is staged.
	t.Run("NotMultipart", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/assets", strings.NewReader("not multipart"))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(env.cookie)
		rec := httptest.NewRecorder()
		env.router.ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d (body %q)", rec.Code, http.StatusBadRequest, rec.Body.String())
		}
	})

	// None of the rejected bodies left anything behind.
	env.requireOriginalsEmpty(t)
	env.requireStagingEmpty(t)
	if got := env.queryInt(t, "SELECT count(*) FROM assets"); got != 0 {
		t.Errorf("assets rows = %d after rejected bodies, want 0", got)
	}
}

// errClientDisconnected stands in for the body read error a request
// suffers when the client vanishes mid-upload: the connection dies
// after the file part started streaming.
var errClientDisconnected = errors.New("client disconnected mid-upload")

// TestUploadAssetClientDisconnectMidStream pins the cleanup contract
// of an upload whose body fails mid-file-part -- the brief's "client
// aborts mid-stream" case at the API layer: the fields and part of
// the photo bytes are delivered, then the body errors instead of ever
// delivering the closing boundary. The staged file is discarded
// (upload staging stays empty), nothing reaches the content-addressed
// originals tree, no row is inserted and no event fires. The handler
// answers 500 INTERNAL: a body read that fails mid-stream cannot be
// distinguished from a server-side read failure, so it is not
// reported as a client validation problem (and the client that
// dropped the connection is typically not there to read any answer).
func TestUploadAssetClientDisconnectMidStream(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	// The fixture is built in the main goroutine: jpegColorFixture
	// uses t.Fatalf, which must not run on a spawned goroutine.
	payload := jpegColorFixture(t, color.RGBA{R: 0x44, G: 0x55, B: 0x66, A: 0xff})

	pipeReader, pipeWriter := io.Pipe()
	writer := multipart.NewWriter(pipeWriter)
	contentType := writer.FormDataContentType()
	go func() {
		for field, value := range map[string]string{
			"deviceAssetId": "device-abort",
			"fileName":      "IMG_abort.jpg",
		} {
			if err := writer.WriteField(field, value); err != nil {
				_ = pipeWriter.CloseWithError(err)
				return
			}
		}
		part, err := writer.CreateFormFile("photo", "IMG_abort.jpg")
		if err != nil {
			_ = pipeWriter.CloseWithError(err)
			return
		}
		if _, err := part.Write(payload); err != nil {
			_ = pipeWriter.CloseWithError(err)
			return
		}
		// The client vanishes: the body fails mid-part, before the
		// closing boundary.
		_ = pipeWriter.CloseWithError(errClientDisconnected)
	}()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/assets", pipeReader)
	req.Header.Set("Content-Type", contentType)
	req.AddCookie(env.cookie)
	rec := httptest.NewRecorder()
	env.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("mid-stream disconnect status = %d, want %d (body %q)",
			rec.Code, http.StatusInternalServerError, rec.Body.String())
	}
	apiErr, _ := decodeError(t, rec)
	if apiErr.Code != apigen.INTERNAL {
		t.Errorf("code = %q, want %q (a failed body read is not a validation problem)",
			apiErr.Code, apigen.INTERNAL)
	}

	// The aborted upload left nothing behind anywhere.
	env.requireStagingEmpty(t)
	env.requireOriginalsEmpty(t)
	if got := env.queryInt(t, "SELECT count(*) FROM assets"); got != 0 {
		t.Errorf("assets rows = %d, want 0", got)
	}
	if got := countAssetUploadedEvents(t, env.bus); got != 0 {
		t.Errorf("asset.uploaded events = %d, want 0", got)
	}
}

// TestUploadAssetRejectedAfterSeal verifies the residue discipline of
// rejections that fire after a file part legitimately sealed: the
// staged files are discarded, so a rejected upload leaves no rows, no
// events, no files -- not even the sealed parts.
func TestUploadAssetRejectedAfterSeal(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	payload := jpegFixture(t)

	// A second photo part after the first sealed.
	duplicate := env.postUpload(t, map[string]string{
		"deviceAssetId": "device-1",
		"fileName":      "IMG_1234.jpg",
	}, []assetUploadFile{
		{field: "photo", name: "IMG_1234.jpg", content: payload},
		{field: "photo", name: "IMG_4321.jpg", content: payload},
	})
	if duplicate.Code != http.StatusBadRequest {
		t.Fatalf("duplicate photo part status = %d, want %d (body %q)", duplicate.Code, http.StatusBadRequest, duplicate.Body.String())
	}

	// A text field arriving after the file part breaks the streaming
	// contract: the handler has validated the fields and sealed the
	// file before the late value arrives, so the request is refused
	// instead of the value being silently ignored. Distinct bytes keep
	// this request's payload distinct from the one above.
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	for field, value := range map[string]string{
		"deviceAssetId": "device-2",
		"fileName":      "IMG_1235.jpg",
	} {
		if err := writer.WriteField(field, value); err != nil {
			t.Fatalf("write field %s: %v", field, err)
		}
	}
	part, err := writer.CreateFormFile("photo", "IMG_1235.jpg")
	if err != nil {
		t.Fatalf("create photo part: %v", err)
	}
	if _, err := part.Write(jpegColorFixture(t, color.RGBA{R: 0x11, G: 0x22, B: 0x33, A: 0xff})); err != nil {
		t.Fatalf("write photo part: %v", err)
	}
	if err := writer.WriteField("deviceTimeZone", "Asia/Shanghai"); err != nil {
		t.Fatalf("write late field: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}
	lateField := env.postUploadRaw(t, &body, writer.FormDataContentType())
	if lateField.Code != http.StatusBadRequest {
		t.Fatalf("late field status = %d, want %d (body %q)", lateField.Code, http.StatusBadRequest, lateField.Body.String())
	}

	// No rows, no events, no files anywhere: the sealed parts were
	// discarded, not placed.
	if got := env.queryInt(t, "SELECT count(*) FROM assets"); got != 0 {
		t.Errorf("assets rows = %d, want 0", got)
	}
	if got := countAssetUploadedEvents(t, env.bus); got != 0 {
		t.Errorf("asset.uploaded events = %d, want 0", got)
	}
	env.requireStagingEmpty(t)
	env.requireOriginalsEmpty(t)
}

// TestUploadRouteOutsideTimeoutAndBodyLimit pins the wiring of the
// upload group: a request whose body streams slower than the handler
// timeout and larger than the JSON body cap still succeeds, because
// POST /assets deliberately runs outside both -- the receiver's own
// per-file limit is the bound that governs the body.
func TestUploadRouteOutsideTimeoutAndBodyLimit(t *testing.T) {
	env := newAssetEnv(t, 8<<20)
	// The same database, bus and layout as the environment's router,
	// but with the test timeout: 300x shorter than production, so a
	// body that trickles for 250ms would be cut by TimeoutHandler if
	// the upload route sat behind it.
	short := newRouter(&Deps{
		Pool:             env.pool,
		Version:          testVersion,
		Events:           env.bus,
		DataDir:          env.dataDir,
		UploadLimitBytes: 8 << 20,
	}, testTimeoutShort)

	// Over 1 MiB of payload (past the JSON body cap) delivered slower
	// than the handler timeout: fields, then the photo bytes, then a
	// pause before the closing boundary.
	payload := append(jpegFixture(t), make([]byte, 1536*1024)...)
	pipeReader, pipeWriter := io.Pipe()
	writer := multipart.NewWriter(pipeWriter)
	contentType := writer.FormDataContentType()
	go func() {
		for field, value := range map[string]string{
			"deviceAssetId": "device-slow",
			"fileName":      "IMG_slow.jpg",
		} {
			if err := writer.WriteField(field, value); err != nil {
				_ = pipeWriter.CloseWithError(err)
				return
			}
		}
		part, err := writer.CreateFormFile("photo", "IMG_slow.jpg")
		if err != nil {
			_ = pipeWriter.CloseWithError(err)
			return
		}
		if _, err := part.Write(payload); err != nil {
			_ = pipeWriter.CloseWithError(err)
			return
		}
		time.Sleep(250 * time.Millisecond)
		if err := writer.Close(); err != nil {
			_ = pipeWriter.CloseWithError(err)
			return
		}
		_ = pipeWriter.Close()
	}()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/assets", pipeReader)
	req.Header.Set("Content-Type", contentType)
	req.AddCookie(env.cookie)
	rec := httptest.NewRecorder()
	short.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("slow large upload status = %d, want %d (body %q)", rec.Code, http.StatusCreated, rec.Body.String())
	}
	asset := decodeAsset(t, rec)
	if asset.State != apigen.Created {
		t.Errorf("state = %q, want %q", asset.State, apigen.Created)
	}
}

// TestGetAssetByDeviceId walks the idempotency probe: a hit answers
// the asset with the state a re-upload would produce, an unknown
// identifier answers 404, a trashed asset keeps answering with the
// trashed state, and another owner's asset is invisible.
func TestGetAssetByDeviceId(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	payload := jpegFixture(t)
	created := decodeAsset(t, env.postUpload(t, map[string]string{
		"deviceAssetId": "device-1",
		"fileName":      "IMG_1234.jpg",
	}, []assetUploadFile{{field: "photo", name: "IMG_1234.jpg", content: payload}}))

	get := func(cookie *http.Cookie, deviceAssetID string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/assets/by-device-id/"+deviceAssetID, nil)
		if cookie != nil {
			req.AddCookie(cookie)
		}
		rec := httptest.NewRecorder()
		env.router.ServeHTTP(rec, req)
		return rec
	}

	t.Run("HitAnswersTheAsset", func(t *testing.T) {
		rec := get(env.cookie, "device-1")
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d (body %q)", rec.Code, http.StatusOK, rec.Body.String())
		}
		asset := decodeAsset(t, rec)
		if asset.Id != created.Id {
			t.Errorf("id = %s, want %s", asset.Id, created.Id)
		}
		if asset.State != apigen.Duplicate {
			t.Errorf("state = %q, want %q (the probe mirrors the re-upload outcome)", asset.State, apigen.Duplicate)
		}
		if asset.Checksum != sha256Hex(payload) {
			t.Errorf("checksum = %q, want %q", asset.Checksum, sha256Hex(payload))
		}
	})

	t.Run("UnknownAnswers404", func(t *testing.T) {
		rec := get(env.cookie, "never-uploaded")
		if rec.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want %d (body %q)", rec.Code, http.StatusNotFound, rec.Body.String())
		}
		apiErr, _ := decodeError(t, rec)
		if apiErr.Code != apigen.ASSETNOTFOUND {
			t.Errorf("code = %q, want %q", apiErr.Code, apigen.ASSETNOTFOUND)
		}
	})

	t.Run("TrashedAnswersTrashedState", func(t *testing.T) {
		env.exec(t, "UPDATE assets SET deleted_at = now() WHERE id = $1", created.Id)
		rec := get(env.cookie, "device-1")
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d (body %q)", rec.Code, http.StatusOK, rec.Body.String())
		}
		asset := decodeAsset(t, rec)
		if asset.State != apigen.Trashed {
			t.Errorf("state = %q, want %q", asset.State, apigen.Trashed)
		}
		if !asset.IsTrashed {
			t.Error("isTrashed = false, want true")
		}
	})

	t.Run("RequiresSession", func(t *testing.T) {
		rec := get(nil, "device-1")
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
		}
	})

	t.Run("OwnerScoped", func(t *testing.T) {
		// A second account cannot see the first one's device asset.
		_, other := env.createSecondUser(t, "grace@example.com")
		rec := get(other, "device-1")
		if rec.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want %d (another owner's asset is invisible)", rec.Code, http.StatusNotFound)
		}
	})
}

// createSecondUser seeds a plain user directly in the database (admin
// user management is a separate surface) and returns the user's id together with a
// session cookie of that user. The row carries no sync album: callers
// that need one insert it (fixtures) or let the upload path's ensure
// create it, which is exactly the recovery the ensure exists for.
func (e *assetEnv) createSecondUser(t *testing.T, email string) (uuid.UUID, *http.Cookie) {
	t.Helper()
	hash, err := auth.Hash(testPassword)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	userID, err := uuid.NewV7()
	if err != nil {
		t.Fatalf("generate user id: %v", err)
	}
	if _, err := e.pool.Exec(context.Background(),
		"INSERT INTO users (id, name, email, password_hash) VALUES ($1, $2, $3, $4)",
		pgUUID(userID), email, email, hash); err != nil {
		t.Fatalf("insert second user: %v", err)
	}
	sessionID, err := uuid.NewV7()
	if err != nil {
		t.Fatalf("generate session id: %v", err)
	}
	raw := "second-user-session-token-" + email
	if _, err := e.pool.Exec(context.Background(),
		"INSERT INTO sessions (id, user_id, token_hash, kind, expires_at) VALUES ($1, $2, $3, 'mobile', $4)",
		pgUUID(sessionID), pgUUID(userID), auth.HashSessionToken(raw),
		time.Now().Add(24*time.Hour)); err != nil {
		t.Fatalf("insert second user session: %v", err)
	}
	return userID, &http.Cookie{Name: sessionCookieName, Value: raw}
}

// TestUploadAmbiguousCommitKeepsOriginals pins the commit-error rule:
// a commit whose outcome is unknown must never delete the placed
// originals, because the rows may be durable. The fixture makes the
// commit fail deterministically with a deferred constraint trigger --
// the inserts succeed, COMMIT raises -- and asserts the file is still
// on disk after the 500, no row exists, and the boot orphan sweep then
// reclaims exactly that leak while keeping a referenced file.
func TestUploadAmbiguousCommitKeepsOriginals(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)

	// The deferred failure: every assets insert marks a commit-time
	// error, so the transaction lands its statements and dies at the
	// commit -- the ambiguous outcome without a dropped connection.
	env.exec(t, `CREATE OR REPLACE FUNCTION fail_assets_commit() RETURNS trigger AS $$
		BEGIN
			RAISE EXCEPTION 'deferred asset insert failure (test)';
		END;
		$$ LANGUAGE plpgsql`)
	env.exec(t, `CREATE CONSTRAINT TRIGGER fail_assets_commit
		AFTER INSERT ON assets DEFERRABLE INITIALLY DEFERRED
		FOR EACH ROW EXECUTE FUNCTION fail_assets_commit()`)

	payload := jpegFixture(t)
	rec := env.postUpload(t, map[string]string{
		"deviceAssetId": "device-ambiguous",
		"fileName":      "IMG_AMBIGUOUS.jpg",
	}, []assetUploadFile{{field: "photo", name: "IMG_AMBIGUOUS.jpg", content: payload}})
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("upload status = %d, want %d (body %q)", rec.Code, http.StatusInternalServerError, rec.Body.String())
	}

	// The placed original STAYS: the commit error does not prove the
	// rows did not land, so the bytes are kept.
	if got := env.countFiles(t, filepath.Join(env.dataDir, "originals")); got != 1 {
		t.Fatalf("originals files = %d after the ambiguous commit, want the placed original kept", got)
	}
	if got := env.queryInt(t, "SELECT count(*) FROM assets"); got != 0 {
		t.Errorf("assets rows = %d after the rolled-back commit, want 0", got)
	}

	// The leak is what the boot sweep exists for: no row names the
	// file, so an age-free sweep removes it.
	result, err := maintenance.SweepOrphanOriginals(context.Background(), env.queries, env.dataDir, 0, time.Now())
	if err != nil {
		t.Fatalf("sweep orphan originals: %v", err)
	}
	if result.Removed != 1 {
		t.Errorf("sweep removed = %d, want the leaked original", result.Removed)
	}
	if got := env.countFiles(t, filepath.Join(env.dataDir, "originals")); got != 0 {
		t.Errorf("originals files = %d after the sweep, want the leak reclaimed", got)
	}

	// The sweep's other half: a committed upload's file is referenced
	// and survives the same age-free sweep.
	env.exec(t, `DROP TRIGGER fail_assets_commit ON assets`)
	env.exec(t, `DROP FUNCTION fail_assets_commit()`)
	created := env.postUpload(t, map[string]string{
		"deviceAssetId": "device-ambiguous-2",
		"fileName":      "IMG_KEPT.jpg",
	}, []assetUploadFile{{field: "photo", name: "IMG_KEPT.jpg", content: jpegFixture(t)}})
	if created.Code != http.StatusCreated {
		t.Fatalf("follow-up upload status = %d, want %d (body %q)", created.Code, http.StatusCreated, created.Body.String())
	}
	row := env.assetByChecksum(t, sha256Hex(jpegFixture(t)))
	result, err = maintenance.SweepOrphanOriginals(context.Background(), env.queries, env.dataDir, 0, time.Now())
	if err != nil {
		t.Fatalf("second sweep: %v", err)
	}
	if result.Removed != 0 {
		t.Errorf("second sweep removed = %d, want the referenced original kept", result.Removed)
	}
	env.requireUploadedFileStored(t, row, jpegFixture(t))
}

// TestSettleAmbiguousCommit pins the helper's two answers: a row the
// owner holds for the checksum means the commit landed (the upload is
// answered created), and an absent row returns the original commit
// error. The probe runs on a detached context, so the request's own
// cancellation -- one of the ways a commit fails -- cannot blind it.
func TestSettleAmbiguousCommit(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	payload := jpegFixture(t)
	created := decodeAsset(t, env.postUpload(t, map[string]string{
		"deviceAssetId": "device-settle",
		"fileName":      "IMG_SETTLE.jpg",
	}, []assetUploadFile{{field: "photo", name: "IMG_SETTLE.jpg", content: payload}}))
	checksum := sha256Hex(payload)
	owner := env.sessionUserID(t)
	commitErr := errors.New("write tcp: connection reset by peer")
	sv := env.purgeTestServer()

	// The client is gone: the request context is already cancelled.
	dead, cancel := context.WithCancel(context.Background())
	cancel()
	row, state, err := sv.settleAmbiguousCommit(dead, owner, checksum, commitErr)
	if err != nil {
		t.Fatalf("settle with the row present: %v", err)
	}
	if state != apigen.Created {
		t.Errorf("state = %q, want %q (the commit landed)", state, apigen.Created)
	}
	if uuid.UUID(row.ID.Bytes) != uuid.UUID(created.Id) {
		t.Errorf("settled row = %s, want the uploaded asset %s", uuid.UUID(row.ID.Bytes), created.Id)
	}

	// Nothing landed for a checksum the owner does not hold: the
	// original error stands (the caller reports the failure; the
	// placed files are not its to delete either way).
	_, _, err = sv.settleAmbiguousCommit(context.Background(), owner, strings.Repeat("0", 64), commitErr)
	if !errors.Is(err, commitErr) {
		t.Errorf("settle with no row error = %v, want the original commit error", err)
	}
}

// TestUploadLivePhotoAbsorbsStandaloneVideo pins the motion reuse: a
// Live Photo whose video bytes are already a LIVE standalone video of
// the same owner does not store a second copy. The part is not placed,
// no second row appears, the existing row becomes the pair's component
// (the same hide write the extraction reuse and the identifier pairing
// apply), the still commits with 201 created, and the response carries
// exactly one notice naming the reused row -- the standalone video
// stopped listing, and the user has to be told.
func TestUploadLivePhotoAbsorbsStandaloneVideo(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	video := movFixture()
	still := jpegColorFixture(t, color.RGBA{R: 0x11, G: 0x44, B: 0x77, A: 0xff})
	videoChecksum := sha256Hex(video)

	standalone := env.postUpload(t, map[string]string{
		"deviceAssetId": "device-absorb-standalone",
		"fileName":      "clip.mov",
	}, []assetUploadFile{{field: "photo", name: "clip.mov", content: video}})
	if standalone.Code != http.StatusCreated {
		t.Fatalf("standalone video upload status = %d, want %d (body %q)",
			standalone.Code, http.StatusCreated, standalone.Body.String())
	}
	standaloneRow := env.assetByChecksum(t, videoChecksum)
	if standaloneRow.MediaKind != dbgen.MediaKindVideo || standaloneRow.IsComponent {
		t.Fatalf("standalone row = kind %q component %v, want a live video",
			standaloneRow.MediaKind, standaloneRow.IsComponent)
	}

	rec := env.postUpload(t, map[string]string{
		"deviceAssetId": "device-absorb-pair",
		"fileName":      "IMG_ABSORB.jpg",
	}, []assetUploadFile{
		{field: "photo", name: "IMG_ABSORB.jpg", content: still},
		{field: "liveVideo", name: "IMG_ABSORB.mov", content: video},
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("live photo upload status = %d, want %d (body %q)", rec.Code, http.StatusCreated, rec.Body.String())
	}
	created := decodeAsset(t, rec)
	if created.State != apigen.Created {
		t.Errorf("state = %q, want %q (the still landed)", created.State, apigen.Created)
	}
	if created.Kind != apigen.AssetResponseKindImage {
		t.Errorf("response kind = %q, want the still's image", created.Kind)
	}

	// The notice: exactly one, naming the absorbed row and its name.
	if created.Notices == nil || len(*created.Notices) != 1 {
		t.Fatalf("notices = %v, want exactly one", created.Notices)
	}
	notice := (*created.Notices)[0]
	if notice.Code != apigen.AssetNoticeCodeLivePhotoMotionAbsorbed {
		t.Errorf("notice code = %q, want %q", notice.Code, apigen.AssetNoticeCodeLivePhotoMotionAbsorbed)
	}
	if notice.AssetId != uuid.UUID(standaloneRow.ID.Bytes) {
		t.Errorf("notice assetId = %s, want the absorbed row %s", notice.AssetId, uuid.UUID(standaloneRow.ID.Bytes))
	}
	if notice.Name != "clip.mov" {
		t.Errorf("notice name = %q, want the absorbed row's filename", notice.Name)
	}

	// The still row exists and links to the absorbed motion.
	photoRow := env.assetByChecksum(t, sha256Hex(still))
	if !photoRow.LivePhotoVideoID.Valid || photoRow.LivePhotoVideoID != standaloneRow.ID {
		t.Fatalf("the still links to %v, want the absorbed row", photoRow.LivePhotoVideoID)
	}
	env.requireUploadedFileStored(t, photoRow, still)

	// The absorbed row is now the pair's component, and its bytes
	// stayed exactly where they were: one row holds those video bytes,
	// one physical file (the standalone upload's), no second copy.
	absorbed := env.assetRowOf(t, uuid.UUID(standaloneRow.ID.Bytes))
	if !absorbed.IsComponent {
		t.Error("the absorbed row is_component = false, want true")
	}
	if absorbed.FilePath != standaloneRow.FilePath {
		t.Errorf("the absorbed row's file_path moved to %q, want its own %q", absorbed.FilePath, standaloneRow.FilePath)
	}
	if got := env.queryInt(t, "SELECT count(*) FROM assets WHERE checksum = $1", videoChecksum); got != 1 {
		t.Errorf("rows holding the video bytes = %d, want 1 (the reused row)", got)
	}
	if got := env.countFiles(t, filepath.Join(env.dataDir, "originals")); got != 2 {
		t.Errorf("originals files = %d, want the still plus the standalone video's one copy", got)
	}

	// No second EXIF placeholder: the absorbed row's sidecar is its own
	// upload's (refined long ago), so only the still's row is new.
	if got := env.queryInt(t, "SELECT count(*) FROM exif WHERE asset_id IN ($1, $2)",
		photoRow.ID, absorbed.ID); got != 2 {
		t.Errorf("exif rows = %d, want one per row (the still's fresh sidecar and the absorbed row's own)", got)
	}
}

// TestUploadLivePhotoTrashedMotionConflicts pins the other half of the
// motion decision: the same bytes held ONLY by a trashed row are not
// reusable (an invisible motion cannot be the pair's) and must not be
// duplicated either, so the upload answers 409
// ASSET_COMPONENT_CONFLICT, stores nothing, and leaves the trashed row
// exactly as it was. A duplicate or trashed answer naming the video row
// would silently lose the still instead.
func TestUploadLivePhotoTrashedMotionConflicts(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	adminID := env.sessionUserID(t)
	video := movFixture()
	still := jpegColorFixture(t, color.RGBA{R: 0x22, G: 0x55, B: 0x88, A: 0xff})
	videoChecksum := sha256Hex(video)

	standalone := env.postUpload(t, map[string]string{
		"deviceAssetId": "device-conflict-standalone",
		"fileName":      "clip.mov",
	}, []assetUploadFile{{field: "photo", name: "clip.mov", content: video}})
	if standalone.Code != http.StatusCreated {
		t.Fatalf("standalone video upload status = %d, want %d (body %q)",
			standalone.Code, http.StatusCreated, standalone.Body.String())
	}
	standaloneRow := env.assetByChecksum(t, videoChecksum)
	if rows := env.softDeleteAsset(t, uuid.UUID(standaloneRow.ID.Bytes), adminID); rows != 1 {
		t.Fatalf("soft delete rows = %d, want the video trashed", rows)
	}
	trashedBefore := env.assetRowOf(t, uuid.UUID(standaloneRow.ID.Bytes))

	rec := env.postUpload(t, map[string]string{
		"deviceAssetId": "device-conflict-pair",
		"fileName":      "IMG_CONFLICT.jpg",
	}, []assetUploadFile{
		{field: "photo", name: "IMG_CONFLICT.jpg", content: still},
		{field: "liveVideo", name: "IMG_CONFLICT.mov", content: video},
	})
	if rec.Code != http.StatusConflict {
		t.Fatalf("live photo upload status = %d, want %d (body %q)", rec.Code, http.StatusConflict, rec.Body.String())
	}
	code, _ := decodeError(t, rec)
	if code.Code != apigen.ASSETCOMPONENTCONFLICT {
		t.Errorf("conflict code = %q, want %q", code.Code, apigen.ASSETCOMPONENTCONFLICT)
	}

	// Nothing was stored: no still row, no second video row, no placed
	// original beside the standalone upload's one file.
	if got := env.queryInt(t, "SELECT count(*) FROM assets WHERE checksum = $1", sha256Hex(still)); got != 0 {
		t.Errorf("still rows = %d, want 0 (the conflict stored nothing)", got)
	}
	if got := env.queryInt(t, "SELECT count(*) FROM assets WHERE checksum = $1", videoChecksum); got != 1 {
		t.Errorf("video rows = %d, want the trashed one alone", got)
	}
	if got := env.countFiles(t, filepath.Join(env.dataDir, "originals")); got != 1 {
		t.Errorf("originals files = %d, want the standalone upload's file alone", got)
	}

	// The trashed row is untouched: still trashed, still a standalone
	// video, its bytes still its own.
	after := env.assetRowOf(t, uuid.UUID(standaloneRow.ID.Bytes))
	if !after.DeletedAt.Valid {
		t.Error("the trashed row is live after the conflict, want it untouched")
	}
	if after.IsComponent {
		t.Error("the trashed row became a component, want it untouched")
	}
	if after.FilePath != trashedBefore.FilePath || after.Checksum != trashedBefore.Checksum {
		t.Errorf("the trashed row changed to %q/%q, want it untouched", after.FilePath, after.Checksum)
	}
}

// TestUploadLivePhotoIgnoresExternalMotionBytes pins the blob-domain
// narrowing of the reuse: an EXTERNAL row carrying the same checksum is
// the family's own source file, never a motion this server may hide as
// a component (that would put the family's file under the pair's
// delete). The Live Photo stores its own video row instead, and no
// notice claims anything was absorbed.
func TestUploadLivePhotoIgnoresExternalMotionBytes(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	ownerID := env.sessionUserID(t)
	video := movFixture()
	still := jpegColorFixture(t, color.RGBA{R: 0x33, G: 0x66, B: 0x99, A: 0xff})
	videoChecksum := sha256Hex(video)

	env.exec(t, `INSERT INTO assets (
			id, owner_id, storage_kind, file_path, checksum, original_filename,
			file_size, mime_type, media_kind, width, height, taken_at,
			taken_at_source, pending_jobs
		) VALUES ($1, $2, 'external', $3, $4, 'clip.mov', $5, 'video/quicktime',
			'video', 0, 0, now(), 'file', 0)`,
		pgUUID(uuid.Must(uuid.NewV7())), pgUUID(ownerID),
		"/mnt/source/clip.mov", videoChecksum, int64(len(video)))

	rec := env.postUpload(t, map[string]string{
		"deviceAssetId": "device-external-motion",
		"fileName":      "IMG_EXTERNAL.jpg",
	}, []assetUploadFile{
		{field: "photo", name: "IMG_EXTERNAL.jpg", content: still},
		{field: "liveVideo", name: "IMG_EXTERNAL.mov", content: video},
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("live photo upload status = %d, want %d (body %q)", rec.Code, http.StatusCreated, rec.Body.String())
	}
	created := decodeAsset(t, rec)
	if created.Notices != nil && len(*created.Notices) != 0 {
		t.Errorf("notices = %v, want none (nothing was absorbed)", *created.Notices)
	}

	// The pair's own blob row landed, and the external row stayed a
	// live standalone source (never hidden, never a component).
	blobRow, err := env.queries.GetBlobAssetByOwnerChecksum(context.Background(), dbgen.GetBlobAssetByOwnerChecksumParams{
		OwnerID:  pgUUID(ownerID),
		Checksum: videoChecksum,
	})
	if err != nil {
		t.Fatalf("load the pair's video row: %v", err)
	}
	if !blobRow.IsComponent {
		t.Error("the pair's video row is_component = false, want true")
	}
	if blobRow.PendingJobs != 1 {
		t.Errorf("the pair's video row pending_jobs = %d, want 1 (its own transcode)", blobRow.PendingJobs)
	}
	externalID, err := uuid.Parse(env.queryText(t,
		"SELECT id::text FROM assets WHERE owner_id = $1 AND storage_kind = 'external'", pgUUID(ownerID)))
	if err != nil {
		t.Fatalf("parse the external row id: %v", err)
	}
	external := env.assetRowOf(t, externalID)
	if external.IsComponent {
		t.Error("the external source row became a component, want the family's file untouched")
	}
	if external.DeletedAt.Valid {
		t.Error("the external source row is trashed, want it untouched")
	}
	if external.FilePath != "/mnt/source/clip.mov" {
		t.Errorf("the external source path = %q, want it untouched", external.FilePath)
	}
}

// absorbedMotionPair uploads a standalone video and then a Live Photo
// whose motion bytes are that video's: the server absorbs the
// standalone row as the pair's component and the still links to it.
// It returns the still's row and the component's row.
func (e *assetEnv) absorbedMotionPair(t *testing.T, video, still []byte, standaloneDevice string) (dbgen.Asset, dbgen.Asset) {
	t.Helper()
	standalone := e.postUpload(t, map[string]string{
		"deviceAssetId": standaloneDevice,
		"fileName":      "clip.mov",
	}, []assetUploadFile{{field: "photo", name: "clip.mov", content: video}})
	if standalone.Code != http.StatusCreated {
		t.Fatalf("standalone video upload status = %d, want %d (body %q)",
			standalone.Code, http.StatusCreated, standalone.Body.String())
	}
	pair := e.postUpload(t, map[string]string{
		"deviceAssetId": standaloneDevice + "-pair",
		"fileName":      "IMG_PAIR.jpg",
	}, []assetUploadFile{
		{field: "photo", name: "IMG_PAIR.jpg", content: still},
		{field: "liveVideo", name: "IMG_PAIR.mov", content: video},
	})
	if pair.Code != http.StatusCreated {
		t.Fatalf("live photo upload status = %d, want %d (body %q)",
			pair.Code, http.StatusCreated, pair.Body.String())
	}
	component := e.assetByChecksum(t, sha256Hex(video))
	if !component.IsComponent {
		t.Fatalf("the motion row is_component = false, want the absorbed component")
	}
	return e.assetByChecksum(t, sha256Hex(still)), component
}

// TestUploadStandaloneVideoAnnouncesItsStoredMotion pins the reverse
// direction of the motion reuse: uploading a video whose bytes already
// play as a Live Photo's motion hits that hidden component row through
// the content checksum. The answer stays 200 duplicate -- the content
// genuinely exists, and a conflict would send the user to fix what is
// not broken -- but it carries exactly one notice naming the component,
// so the client can say why the video does not list on its own. Nothing
// is placed, no row is added, and the component stays hidden.
func TestUploadStandaloneVideoAnnouncesItsStoredMotion(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	video := movFixture()
	still := jpegColorFixture(t, color.RGBA{R: 0x44, G: 0x77, B: 0xaa, A: 0xff})
	videoChecksum := sha256Hex(video)
	photoRow, component := env.absorbedMotionPair(t, video, still, "device-reverse-checksum")
	originalsBefore := env.countFiles(t, filepath.Join(env.dataDir, "originals"))

	rec := env.postUpload(t, map[string]string{
		"deviceAssetId": "device-reverse-checksum-standalone-again",
		"fileName":      "clip.mov",
	}, []assetUploadFile{{field: "photo", name: "clip.mov", content: video}})
	if rec.Code != http.StatusOK {
		t.Fatalf("standalone re-upload status = %d, want %d (body %q)",
			rec.Code, http.StatusOK, rec.Body.String())
	}
	duplicate := decodeAsset(t, rec)
	if duplicate.State != apigen.Duplicate {
		t.Errorf("state = %q, want %q (the bytes exist; nothing is broken)", duplicate.State, apigen.Duplicate)
	}
	if duplicate.IsTrashed {
		t.Error("isTrashed = true, want the live component's answer")
	}
	if duplicate.Id != uuid.UUID(component.ID.Bytes) {
		t.Errorf("duplicate id = %s, want the component %s",
			duplicate.Id, uuid.UUID(component.ID.Bytes))
	}
	if duplicate.Kind != apigen.AssetResponseKindVideo {
		t.Errorf("kind = %q, want the video component's", duplicate.Kind)
	}

	// Exactly one notice, naming the component and its filename.
	if duplicate.Notices == nil || len(*duplicate.Notices) != 1 {
		t.Fatalf("notices = %v, want exactly one", duplicate.Notices)
	}
	notice := (*duplicate.Notices)[0]
	if notice.Code != apigen.AssetNoticeCodeLivePhotoMotionAlreadyStored {
		t.Errorf("notice code = %q, want %q",
			notice.Code, apigen.AssetNoticeCodeLivePhotoMotionAlreadyStored)
	}
	if notice.AssetId != uuid.UUID(component.ID.Bytes) {
		t.Errorf("notice assetId = %s, want the component %s",
			notice.AssetId, uuid.UUID(component.ID.Bytes))
	}
	if notice.Name != "clip.mov" {
		t.Errorf("notice name = %q, want the component's filename", notice.Name)
	}

	// Nothing was stored: one row still holds the video bytes, the
	// pair is untouched, and no file landed beside the owned ones.
	if got := env.queryInt(t, "SELECT count(*) FROM assets WHERE checksum = $1", videoChecksum); got != 1 {
		t.Errorf("rows holding the video bytes = %d, want the reused row alone", got)
	}
	after := env.assetRowOf(t, uuid.UUID(component.ID.Bytes))
	if !after.IsComponent || after.DeletedAt.Valid {
		t.Errorf("component after the re-upload = is_component %v, deleted %v, want a live hidden component",
			after.IsComponent, after.DeletedAt.Valid)
	}
	if after.FilePath != component.FilePath {
		t.Errorf("component file_path = %q, want its own %q", after.FilePath, component.FilePath)
	}
	if got := env.countFiles(t, filepath.Join(env.dataDir, "originals")); got != originalsBefore {
		t.Errorf("originals files = %d, want the %d already stored", got, originalsBefore)
	}
	if stillRow := env.assetRowOf(t, uuid.UUID(photoRow.ID.Bytes)); stillRow.DeletedAt.Valid {
		t.Error("the pair's still is trashed, want it untouched")
	}
	env.requireStagingEmpty(t)
}

// TestUploadStandaloneVideoAnnouncesStoredMotionByDeviceIdentity pins
// the other lookup path: an upload whose content is new but whose
// deviceAssetId belongs to a Live Photo's absorbed motion resolves to
// that component row, and the duplicate answer carries the same notice.
// The distinct bytes are the point -- identical bytes would hit the
// content checksum first, leaving the device arm unexercised.
func TestUploadStandaloneVideoAnnouncesStoredMotionByDeviceIdentity(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	video := movFixture()
	still := jpegColorFixture(t, color.RGBA{R: 0x55, G: 0x88, B: 0xbb, A: 0xff})
	_, component := env.absorbedMotionPair(t, video, still, "device-reverse-device")

	// Another QuickTime container: the same header (it sniffs as a
	// video) with a different trailing byte, so the checksum misses
	// and only the device identity can answer.
	other := append([]byte(nil), video...)
	other[len(other)-1] = 0x01
	if sha256Hex(other) == sha256Hex(video) {
		t.Fatal("the two fixtures share a checksum, want distinct content")
	}
	rowsBefore := env.queryInt(t, "SELECT count(*) FROM assets")
	originalsBefore := env.countFiles(t, filepath.Join(env.dataDir, "originals"))

	rec := env.postUpload(t, map[string]string{
		"deviceAssetId": "device-reverse-device",
		"fileName":      "clip-reencoded.mov",
	}, []assetUploadFile{{field: "photo", name: "clip-reencoded.mov", content: other}})
	if rec.Code != http.StatusOK {
		t.Fatalf("device-identity re-upload status = %d, want %d (body %q)",
			rec.Code, http.StatusOK, rec.Body.String())
	}
	duplicate := decodeAsset(t, rec)
	if duplicate.State != apigen.Duplicate {
		t.Errorf("state = %q, want %q", duplicate.State, apigen.Duplicate)
	}
	if duplicate.Id != uuid.UUID(component.ID.Bytes) {
		t.Errorf("duplicate id = %s, want the component %s",
			duplicate.Id, uuid.UUID(component.ID.Bytes))
	}
	if duplicate.Notices == nil || len(*duplicate.Notices) != 1 {
		t.Fatalf("notices = %v, want exactly one", duplicate.Notices)
	}
	notice := (*duplicate.Notices)[0]
	if notice.Code != apigen.AssetNoticeCodeLivePhotoMotionAlreadyStored {
		t.Errorf("notice code = %q, want %q",
			notice.Code, apigen.AssetNoticeCodeLivePhotoMotionAlreadyStored)
	}
	if notice.AssetId != uuid.UUID(component.ID.Bytes) {
		t.Errorf("notice assetId = %s, want the component %s",
			notice.AssetId, uuid.UUID(component.ID.Bytes))
	}
	// The notice names the row the bytes landed in, not the new
	// client-side name of the colliding upload.
	if notice.Name != "clip.mov" {
		t.Errorf("notice name = %q, want the component's filename", notice.Name)
	}

	// The device-identity answer places nothing: no row, no file.
	if got := env.queryInt(t, "SELECT count(*) FROM assets"); got != rowsBefore {
		t.Errorf("assets rows = %d after the answer, want the %d already stored", got, rowsBefore)
	}
	if got := env.countFiles(t, filepath.Join(env.dataDir, "originals")); got != originalsBefore {
		t.Errorf("originals files = %d, want the %d already stored", got, originalsBefore)
	}
	env.requireStagingEmpty(t)
}

// TestUploadTrashedMotionKeepsTrashedStateWithoutNotice pins the
// trashed half: the same video re-uploaded while its pair sits in the
// trash answers the ordinary 200 trashed -- the client's restore /
// save-as dialog already explains that -- and adds no notice. An
// invisible row has no listing to lose, so there is nothing extra to
// tell.
func TestUploadTrashedMotionKeepsTrashedStateWithoutNotice(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	video := movFixture()
	still := jpegColorFixture(t, color.RGBA{R: 0x66, G: 0x99, B: 0xcc, A: 0xff})
	photoRow, component := env.absorbedMotionPair(t, video, still, "device-reverse-trashed")

	// The pair moves through the trash as one asset: trashing the
	// still stamps its component too.
	if rows := env.softDeleteAsset(t, uuid.UUID(photoRow.ID.Bytes), env.sessionUserID(t)); rows != 2 {
		t.Fatalf("soft delete rows = %d, want the pair (still and component)", rows)
	}

	rec := env.postUpload(t, map[string]string{
		"deviceAssetId": "device-reverse-trashed-again",
		"fileName":      "clip.mov",
	}, []assetUploadFile{{field: "photo", name: "clip.mov", content: video}})
	if rec.Code != http.StatusOK {
		t.Fatalf("standalone re-upload status = %d, want %d (body %q)",
			rec.Code, http.StatusOK, rec.Body.String())
	}
	trashed := decodeAsset(t, rec)
	if trashed.State != apigen.Trashed {
		t.Errorf("state = %q, want %q (the content sits in the trash)", trashed.State, apigen.Trashed)
	}
	if !trashed.IsTrashed {
		t.Error("isTrashed = false, want true")
	}
	if trashed.Id != uuid.UUID(component.ID.Bytes) {
		t.Errorf("trashed id = %s, want the component %s",
			trashed.Id, uuid.UUID(component.ID.Bytes))
	}
	if trashed.Notices != nil && len(*trashed.Notices) != 0 {
		t.Errorf("notices = %v, want none (the restore dialog explains the trash)", *trashed.Notices)
	}
	if got := env.queryInt(t, "SELECT count(*) FROM assets WHERE checksum = $1", sha256Hex(video)); got != 1 {
		t.Errorf("rows holding the video bytes = %d, want the trashed component alone", got)
	}
	if after := env.assetRowOf(t, uuid.UUID(component.ID.Bytes)); !after.DeletedAt.Valid {
		t.Error("the component is live after the answer, want it still trashed")
	}
}
