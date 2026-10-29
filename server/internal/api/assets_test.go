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

	// The file lives at the content-addressed path of the checksum.
	wantPath := "originals/" + checksum[:2] + "/" + checksum[2:4] + "/" + checksum + ".jpg"
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

	// The trash flow is M2c work; the test drives the soft-delete
	// directly, which is the state it produces.
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

// TestUploadAssetConcurrent mirrors the setup concurrency pattern of
// M1: two uploads of identical bytes, genuinely in flight at the same
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

// TestUploadAssetRejectedAfterCommit pins the residue contract of a
// request that turns invalid only after a file part has legitimately
// committed: the blob stays (a content-addressed original may be
// shared with another owner's asset, so removing it is unsafe), but
// no row is inserted, no event fires, and staging stays clean.
func TestUploadAssetRejectedAfterCommit(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	payload := jpegFixture(t)

	// A second photo part after the first committed.
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
	// contract: the handler has validated the fields and committed the
	// file before the late value arrives, so the request is refused
	// instead of the value being silently ignored. Distinct bytes keep
	// this request's blob distinct from the one above.
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

	// No rows, no events, clean staging; the two committed blobs
	// remain as the documented residue.
	if got := env.queryInt(t, "SELECT count(*) FROM assets"); got != 0 {
		t.Errorf("assets rows = %d, want 0", got)
	}
	if got := countAssetUploadedEvents(t, env.bus); got != 0 {
		t.Errorf("asset.uploaded events = %d, want 0", got)
	}
	env.requireStagingEmpty(t)
	if got := env.countFiles(t, filepath.Join(env.dataDir, "originals")); got != 2 {
		t.Errorf("originals tree holds %d files, want the 2 committed blobs left as residue", got)
	}
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
		other := env.createSecondUser(t, "grace@example.com")
		rec := get(other, "device-1")
		if rec.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want %d (another owner's asset is invisible)", rec.Code, http.StatusNotFound)
		}
	})
}

// createSecondUser seeds a plain user directly in the database (admin
// user management is M2c) and returns a session cookie of that user.
func (e *assetEnv) createSecondUser(t *testing.T, email string) *http.Cookie {
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
	raw := "second-user-session-token"
	if _, err := e.pool.Exec(context.Background(),
		"INSERT INTO sessions (id, user_id, token_hash, kind, expires_at) VALUES ($1, $2, $3, 'mobile', $4)",
		pgUUID(sessionID), pgUUID(userID), auth.HashSessionToken(raw),
		time.Now().Add(24*time.Hour)); err != nil {
		t.Fatalf("insert second user session: %v", err)
	}
	return &http.Cookie{Name: sessionCookieName, Value: raw}
}
