// Serving tests of the asset detail and media endpoints: the detail
// mapping (including its null-before-ready semantics), the
// anti-enumeration 404 shared by every endpoint, the derived-file
// serving mechanics (headers, conditional requests, the poster of a
// video), the original download with its escaped file name, and
// ranges. The full pipeline that PRODUCES the derived files runs in
// the jobs package (its env wires the queue); these tests drive the
// handler layer directly and write the derived bytes the pipeline
// would have left behind.
package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	apigen "github.com/welpher/lumemor/server/internal/api/gen"
	"github.com/welpher/lumemor/server/internal/storage"
)

// serveGet fires a GET request against the environment's router with
// the given session cookie and extra headers (nil cookie sends it
// unauthenticated).
func (e *assetEnv) serveGet(t *testing.T, target string, cookie *http.Cookie, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, target, nil)
	if cookie != nil {
		req.AddCookie(cookie)
	}
	for name, value := range headers {
		req.Header.Set(name, value)
	}
	rec := httptest.NewRecorder()
	e.router.ServeHTTP(rec, req)
	return rec
}

// writeDerivedFile stores the bytes of a derived media file at the
// storage path the serving handler resolves -- standing in for the
// thumbnail job's output (covered in the jobs package with the real
// pipeline).
func (e *assetEnv) writeDerivedFile(t *testing.T, path string, content []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("create derived directory of %s: %v", path, err)
	}
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("write derived file %s: %v", path, err)
	}
}

// TestGetAssetDetail walks the detail mapping: the response before the
// pipeline ran (the pipeline-dependent fields null, isReady false) and
// after it completed (the refinements the jobs wrote surfaced through
// the joined row). The row updates mirror what the metadata and
// thumbnail jobs write; their behavior is pinned in the jobs package.
func TestGetAssetDetail(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	payload := jpegFixture(t)
	deviceCreatedAt := "2026-01-02T15:04:05.123456+08:00"
	wantTakenAt, err := time.Parse(time.RFC3339, deviceCreatedAt)
	if err != nil {
		t.Fatalf("parse deviceCreatedAt fixture: %v", err)
	}

	created := decodeAsset(t, env.postUpload(t, map[string]string{
		"deviceAssetId":   "device-detail-1",
		"fileName":        "IMG_1234.jpg",
		"deviceCreatedAt": deviceCreatedAt,
	}, []assetUploadFile{{field: "photo", name: "IMG_1234.jpg", content: payload}}))
	if created.State != apigen.Created {
		t.Fatalf("upload state = %q, want %q", created.State, apigen.Created)
	}

	// Before the pipeline: nothing refined, nothing ready.
	rec := env.serveGet(t, "/api/v1/assets/"+created.Id.String(), env.cookie, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("detail status = %d, want %d (body %q)", rec.Code, http.StatusOK, rec.Body.String())
	}
	var fresh apigen.AssetDetail
	if err := json.Unmarshal(rec.Body.Bytes(), &fresh); err != nil {
		t.Fatalf("decode fresh detail %q: %v", rec.Body.String(), err)
	}
	if fresh.IsReady {
		t.Error("isReady = true before the pipeline ran, want false")
	}
	if fresh.Width != nil || fresh.Height != nil {
		t.Errorf("dimensions = %v/%v, want null before the metadata job", fresh.Width, fresh.Height)
	}
	if fresh.Blurhash != nil {
		t.Errorf("blurhash = %v, want null before the thumbnail job", *fresh.Blurhash)
	}
	if fresh.DurationMs != nil {
		t.Errorf("durationMs = %v, want null for an image", *fresh.DurationMs)
	}
	if fresh.LivePhotoVideoId != nil {
		t.Error("livePhotoVideoId set, want null for a plain photo")
	}
	if fresh.TakenAtSource != apigen.AssetDetailTakenAtSourceImport {
		t.Errorf("takenAtSource = %q, want %q", fresh.TakenAtSource, apigen.AssetDetailTakenAtSourceImport)
	}
	if !fresh.TakenAt.Truncate(time.Microsecond).Equal(wantTakenAt) {
		t.Errorf("fresh takenAt = %v, want the device-reported %v", fresh.TakenAt, wantTakenAt)
	}
	if fresh.Exif.CameraMake != nil || fresh.Exif.Iso != nil || fresh.Exif.Lat != nil {
		t.Errorf("exif hot fields = %v/%v/%v, want null before the metadata job",
			fresh.Exif.CameraMake, fresh.Exif.Iso, fresh.Exif.Lat)
	}

	// The pipeline-completed state the jobs produce: dimensions,
	// blurhash, ready_at, the exif extraction, and the EXIF-resolved
	// capture time with its source.
	takenAtExif := time.Date(2023, 7, 14, 4, 45, 30, 0, time.UTC).Truncate(time.Microsecond)
	env.exec(t, `UPDATE assets SET
		width = 64, height = 64, blurhash = 'UHL6aAof00WB00WB00WB00',
		taken_at = $2, taken_at_source = 'exif', ready_at = now(), updated_at = now()
		WHERE id = $1`, created.Id, takenAtExif)
	env.exec(t, `UPDATE exif SET
		camera_make = 'Lumemor', camera_model = 'Golden Camera',
		lens = 'Golden 35mm f/2.8', iso = 200, aperture = 2.8,
		shutter = '1/250', focal = 35, lat = 48.8584, lon = 2.2945,
		alt = 35, tz_offset = '+05:30', updated_at = now()
		WHERE asset_id = $1`, created.Id)

	rec = env.serveGet(t, "/api/v1/assets/"+created.Id.String(), env.cookie, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("refined detail status = %d, want %d (body %q)", rec.Code, http.StatusOK, rec.Body.String())
	}
	var detail apigen.AssetDetail
	if err := json.Unmarshal(rec.Body.Bytes(), &detail); err != nil {
		t.Fatalf("decode refined detail %q: %v", rec.Body.String(), err)
	}

	if detail.Id != created.Id {
		t.Errorf("id = %s, want the uploaded asset %s", detail.Id, created.Id)
	}
	if detail.OwnerId != env.sessionUserID(t) {
		t.Errorf("ownerId = %s, want the session owner %s", detail.OwnerId, env.sessionUserID(t))
	}
	if detail.Kind != apigen.AssetDetailKindImage {
		t.Errorf("kind = %q, want %q", detail.Kind, apigen.AssetDetailKindImage)
	}
	if detail.OriginalFilename != "IMG_1234.jpg" {
		t.Errorf("originalFilename = %q, want IMG_1234.jpg", detail.OriginalFilename)
	}
	if detail.FileSize != int64(len(payload)) {
		t.Errorf("fileSize = %d, want %d", detail.FileSize, len(payload))
	}
	if detail.MimeType != "image/jpeg" {
		t.Errorf("mimeType = %q, want image/jpeg", detail.MimeType)
	}
	if detail.Checksum != sha256Hex(payload) {
		t.Errorf("checksum = %q, want %q", detail.Checksum, sha256Hex(payload))
	}
	if detail.IsTrashed {
		t.Error("isTrashed = true, want false")
	}
	if !detail.IsReady {
		t.Error("isReady = false after the pipeline completed, want true")
	}
	if detail.CreatedAt.IsZero() {
		t.Error("createdAt = zero, want the row's creation stamp")
	}
	if !detail.TakenAt.Truncate(time.Microsecond).Equal(takenAtExif) {
		t.Errorf("takenAt = %v, want the EXIF-refined %v", detail.TakenAt, takenAtExif)
	}
	if detail.TakenAtSource != apigen.AssetDetailTakenAtSourceExif {
		t.Errorf("takenAtSource = %q, want %q", detail.TakenAtSource, apigen.AssetDetailTakenAtSourceExif)
	}
	if detail.Width == nil || *detail.Width != 64 {
		t.Errorf("width = %v, want 64", detail.Width)
	}
	if detail.Height == nil || *detail.Height != 64 {
		t.Errorf("height = %v, want 64", detail.Height)
	}
	if detail.Blurhash == nil || *detail.Blurhash != "UHL6aAof00WB00WB00WB00" {
		t.Errorf("blurhash = %v, want the stored hash", detail.Blurhash)
	}
	if detail.DurationMs != nil {
		t.Errorf("durationMs = %v, want null for an image", *detail.DurationMs)
	}
	if detail.LivePhotoVideoId != nil {
		t.Error("livePhotoVideoId set, want null for a plain photo")
	}

	exif := detail.Exif
	if exif.CameraMake == nil || *exif.CameraMake != "Lumemor" {
		t.Errorf("exif.cameraMake = %v, want Lumemor", exif.CameraMake)
	}
	if exif.CameraModel == nil || *exif.CameraModel != "Golden Camera" {
		t.Errorf("exif.cameraModel = %v, want Golden Camera", exif.CameraModel)
	}
	if exif.Lens == nil || *exif.Lens != "Golden 35mm f/2.8" {
		t.Errorf("exif.lens = %v, want Golden 35mm f/2.8", exif.Lens)
	}
	if exif.Iso == nil || *exif.Iso != 200 {
		t.Errorf("exif.iso = %v, want 200", exif.Iso)
	}
	if exif.Aperture == nil || *exif.Aperture != 2.8 {
		t.Errorf("exif.aperture = %v, want 2.8", exif.Aperture)
	}
	if exif.Shutter == nil || *exif.Shutter != "1/250" {
		t.Errorf("exif.shutter = %v, want 1/250", exif.Shutter)
	}
	if exif.Focal == nil || *exif.Focal != 35 {
		t.Errorf("exif.focal = %v, want 35", exif.Focal)
	}
	if exif.Lat == nil || *exif.Lat != 48.8584 {
		t.Errorf("exif.lat = %v, want 48.8584", exif.Lat)
	}
	if exif.Lon == nil || *exif.Lon != 2.2945 {
		t.Errorf("exif.lon = %v, want 2.2945", exif.Lon)
	}
	if exif.Alt == nil || *exif.Alt != 35 {
		t.Errorf("exif.alt = %v, want 35", exif.Alt)
	}
	if exif.TzOffset == nil || *exif.TzOffset != "+05:30" {
		t.Errorf("exif.tzOffset = %v, want +05:30", exif.TzOffset)
	}

	// The trashed state stays readable, flagged.
	env.exec(t, "UPDATE assets SET deleted_at = now() WHERE id = $1", created.Id)
	rec = env.serveGet(t, "/api/v1/assets/"+created.Id.String(), env.cookie, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("trashed detail status = %d, want %d", rec.Code, http.StatusOK)
	}
	var trashed apigen.AssetDetail
	if err := json.Unmarshal(rec.Body.Bytes(), &trashed); err != nil {
		t.Fatalf("decode trashed detail %q: %v", rec.Body.String(), err)
	}
	if !trashed.IsTrashed {
		t.Error("isTrashed = false for a trashed asset, want true")
	}
}

// TestAssetEndpointsOwnerScope pins the anti-enumeration discipline of
// every asset endpoint: a foreign id and an unknown id answer the same
// 404, byte for byte, and unauthenticated requests answer 401. The
// byte-identical comparison runs over the whole body, not just the
// code -- the M2a login discipline carried into the asset domain.
func TestAssetEndpointsOwnerScope(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	payload := jpegFixture(t)
	created := decodeAsset(t, env.postUpload(t, map[string]string{
		"deviceAssetId": "device-scope-1",
		"fileName":      "IMG_1234.jpg",
	}, []assetUploadFile{{field: "photo", name: "IMG_1234.jpg", content: payload}}))

	random, err := uuid.NewV7()
	if err != nil {
		t.Fatalf("generate random id: %v", err)
	}
	_, other := env.createSecondUser(t, "grace@example.com")

	targets := []string{
		"/api/v1/assets/" + created.Id.String(),
		"/api/v1/assets/" + created.Id.String() + "/thumbnail?size=tile",
		"/api/v1/assets/" + created.Id.String() + "/original",
	}
	for _, target := range targets {
		// The foreign id: the second account asks for the owner's
		// asset. The unknown id: the same account asks for a random
		// uuid. The two responses must be indistinguishable.
		unknownTarget := strings.Replace(target, created.Id.String(), random.String(), 1)

		foreign := env.serveGet(t, target, other, nil)
		if foreign.Code != http.StatusNotFound {
			t.Fatalf("foreign GET %s status = %d, want %d (body %q)",
				target, foreign.Code, http.StatusNotFound, foreign.Body.String())
		}
		unknown := env.serveGet(t, unknownTarget, other, nil)
		if unknown.Code != http.StatusNotFound {
			t.Fatalf("unknown GET %s status = %d, want %d (body %q)",
				unknownTarget, unknown.Code, http.StatusNotFound, unknown.Body.String())
		}
		if foreign.Body.String() != unknown.Body.String() {
			t.Errorf("404 bodies differ between the foreign and the unknown id of %s:\nforeign: %s\nunknown: %s",
				target, foreign.Body.String(), unknown.Body.String())
		}
		apiErr, _ := decodeError(t, foreign)
		if apiErr.Code != apigen.ASSETNOTFOUND {
			t.Errorf("code = %q, want %q", apiErr.Code, apigen.ASSETNOTFOUND)
		}

		// The session gate precedes everything: an unauthenticated
		// request never learns whether the id exists.
		unauthenticated := env.serveGet(t, target, nil, nil)
		if unauthenticated.Code != http.StatusUnauthorized {
			t.Fatalf("unauthenticated GET %s status = %d, want %d",
				target, unauthenticated.Code, http.StatusUnauthorized)
		}
	}
}

// TestGetAssetThumbnailNotReady pins the derived-file-miss state: the
// asset exists (the upload answered 201) but the thumbnail job has not
// produced its file -- the environment wires no queue, so nothing ever
// runs. The answer is 404 ASSET_NOT_READY, not ASSET_NOT_FOUND: the
// client keeps the asset and waits for asset.ready. The original, by
// contrast, is the uploaded file itself and serves immediately.
func TestGetAssetThumbnailNotReady(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	payload := jpegFixture(t)
	created := decodeAsset(t, env.postUpload(t, map[string]string{
		"deviceAssetId": "device-notready-1",
		"fileName":      "IMG_1234.jpg",
	}, []assetUploadFile{{field: "photo", name: "IMG_1234.jpg", content: payload}}))
	if created.State != apigen.Created {
		t.Fatalf("upload state = %q, want %q", created.State, apigen.Created)
	}
	row := env.assetByChecksum(t, sha256Hex(payload))
	if row.PendingJobs == 0 || row.ReadyAt.Valid {
		t.Fatalf("fixture row pending_jobs = %d ready_at = %v, want a pending pipeline", row.PendingJobs, row.ReadyAt)
	}

	rec := env.serveGet(t, "/api/v1/assets/"+created.Id.String()+"/thumbnail?size=tile", env.cookie, nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("not-ready thumbnail status = %d, want %d (body %q)", rec.Code, http.StatusNotFound, rec.Body.String())
	}
	apiErr, _ := decodeError(t, rec)
	if apiErr.Code != apigen.ASSETNOTREADY {
		t.Errorf("code = %q, want %q", apiErr.Code, apigen.ASSETNOTREADY)
	}

	// The original is not pipeline-gated.
	original := env.serveGet(t, "/api/v1/assets/"+created.Id.String()+"/original", env.cookie, nil)
	if original.Code != http.StatusOK {
		t.Fatalf("original of a not-ready asset status = %d, want %d", original.Code, http.StatusOK)
	}
}

// TestGetAssetThumbnailServing walks the serving mechanics of the
// thumbnail endpoint with the derived files the pipeline would leave
// behind: the headers of the static-asset contract (content type,
// strong checksum-prefix ETag, immutable private caching), the file
// of each requested size, the poster of a video asset for every size,
// the conditional 304 (strong, weak, and star If-None-Match), and the
// size validation.
func TestGetAssetThumbnailServing(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	payload := jpegFixture(t)
	created := decodeAsset(t, env.postUpload(t, map[string]string{
		"deviceAssetId": "device-thumb-1",
		"fileName":      "IMG_1234.jpg",
	}, []assetUploadFile{{field: "photo", name: "IMG_1234.jpg", content: payload}}))
	checksum := sha256Hex(payload)
	assetID := created.Id.String()

	// The derived file set of the image asset, one distinct payload
	// per size (the real AVIF bytes and their dimensions are pinned by
	// the jobs package's pipeline tests).
	derived := map[string][]byte{
		"tile":     []byte("tile-avif-bytes"),
		"view":     []byte("view-avif-bytes"),
		"original": []byte("original-avif-bytes"),
	}
	for size, content := range derived {
		env.writeDerivedFile(t, storage.ThumbPath(assetID, size), content)
	}
	wantETag := `"` + checksum[:16] + `"`

	t.Run("ServesEachSize", func(t *testing.T) {
		for size, content := range derived {
			rec := env.serveGet(t, "/api/v1/assets/"+assetID+"/thumbnail?size="+size, env.cookie, nil)
			if rec.Code != http.StatusOK {
				t.Fatalf("size %s status = %d, want %d (body %q)", size, rec.Code, http.StatusOK, rec.Body.String())
			}
			if got := rec.Header().Get("Content-Type"); got != "image/avif" {
				t.Errorf("size %s content-type = %q, want image/avif", size, got)
			}
			if got := rec.Header().Get("ETag"); got != wantETag {
				t.Errorf("size %s etag = %q, want %q", size, got, wantETag)
			}
			if got := rec.Header().Get("Cache-Control"); got != "private, max-age=31536000, immutable" {
				t.Errorf("size %s cache-control = %q, want the immutable private policy", size, got)
			}
			if !bytes.Equal(rec.Body.Bytes(), content) {
				t.Errorf("size %s served %q, want the derived file bytes", size, rec.Body.String())
			}
		}
	})

	t.Run("ConditionalRequests", func(t *testing.T) {
		cases := []struct {
			name        string
			ifNoneMatch string
			wantStatus  int
		}{
			{"StrongMatch", wantETag, http.StatusNotModified},
			{"WeakMatch", `W/` + wantETag, http.StatusNotModified},
			{"Star", "*", http.StatusNotModified},
			{"ListWithMatch", `"<other>", ` + wantETag, http.StatusNotModified},
			{"Mismatch", `"0000000000000000"`, http.StatusOK},
		}
		for _, testCase := range cases {
			rec := env.serveGet(t, "/api/v1/assets/"+assetID+"/thumbnail?size=tile", env.cookie,
				map[string]string{"If-None-Match": testCase.ifNoneMatch})
			if rec.Code != testCase.wantStatus {
				t.Fatalf("%s status = %d, want %d", testCase.name, rec.Code, testCase.wantStatus)
			}
			if got := rec.Header().Get("ETag"); got != wantETag {
				t.Errorf("%s etag = %q, want %q also on the 304", testCase.name, got, wantETag)
			}
			if testCase.wantStatus == http.StatusNotModified && rec.Body.Len() != 0 {
				t.Errorf("%s body = %q, want no body on a 304", testCase.name, rec.Body.String())
			}
		}
	})

	t.Run("VideoServesPosterForEverySize", func(t *testing.T) {
		video := movFixture()
		uploaded := decodeAsset(t, env.postUpload(t, map[string]string{
			"deviceAssetId": "device-thumb-video",
			"fileName":      "VID_0001.mov",
		}, []assetUploadFile{{field: "photo", name: "VID_0001.mov", content: video}}))
		if uploaded.Kind != apigen.AssetResponseKindVideo {
			t.Fatalf("video upload kind = %q, want video", uploaded.Kind)
		}
		poster := []byte("poster-jpeg-bytes")
		env.writeDerivedFile(t, storage.PosterPath(uploaded.Id.String()), poster)

		for _, size := range []string{"tile", "view", "original"} {
			rec := env.serveGet(t, "/api/v1/assets/"+uploaded.Id.String()+"/thumbnail?size="+size, env.cookie, nil)
			if rec.Code != http.StatusOK {
				t.Fatalf("video size %s status = %d, want %d", size, rec.Code, http.StatusOK)
			}
			if got := rec.Header().Get("Content-Type"); got != "image/jpeg" {
				t.Errorf("video size %s content-type = %q, want image/jpeg (the poster)", size, got)
			}
			if got := rec.Header().Get("ETag"); got != `"`+sha256Hex(video)[:16]+`"` {
				t.Errorf("video size %s etag = %q, want the checksum prefix", size, got)
			}
			if !bytes.Equal(rec.Body.Bytes(), poster) {
				t.Errorf("video size %s served %q, want the poster bytes", size, rec.Body.String())
			}
		}
	})

	t.Run("InvalidSize", func(t *testing.T) {
		for _, target := range []string{
			"/api/v1/assets/" + assetID + "/thumbnail?size=large",
			"/api/v1/assets/" + assetID + "/thumbnail",
		} {
			rec := env.serveGet(t, target, env.cookie, nil)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("GET %s status = %d, want %d (body %q)", target, rec.Code, http.StatusBadRequest, rec.Body.String())
			}
			apiErr, fields := decodeError(t, rec)
			if apiErr.Code != apigen.VALIDATIONFAILED {
				t.Errorf("GET %s code = %q, want %q", target, apiErr.Code, apigen.VALIDATIONFAILED)
			}
			if !contains(fields, "size") {
				t.Errorf("GET %s details.fields = %v, want it to contain size", target, fields)
			}
		}
	})
}

// TestGetAssetOriginalDownload pins the original download: the stored
// bytes verbatim, the stored mime type, the cache validators, and the
// RFC 5987 escaped Content-Disposition of a non-ASCII file name -- the
// expected header value is spelled out literally, so any drift in the
// escaping fails the assertion instead of echoing the implementation.
func TestGetAssetOriginalDownload(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	payload := jpegFixture(t)
	created := decodeAsset(t, env.postUpload(t, map[string]string{
		"deviceAssetId": "device-original-1",
		"fileName":      "照片 IMG_1234.jpg",
	}, []assetUploadFile{{field: "photo", name: "照片 IMG_1234.jpg", content: payload}}))
	checksum := sha256Hex(payload)

	rec := env.serveGet(t, "/api/v1/assets/"+created.Id.String()+"/original", env.cookie, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("original status = %d, want %d (body %q)", rec.Code, http.StatusOK, rec.Body.String())
	}
	if got := rec.Header().Get("Content-Type"); got != "image/jpeg" {
		t.Errorf("content-type = %q, want the stored image/jpeg", got)
	}
	if got := rec.Header().Get("ETag"); got != `"`+checksum[:16]+`"` {
		t.Errorf("etag = %q, want the quoted checksum prefix", got)
	}
	if got := rec.Header().Get("Cache-Control"); got != "private, max-age=31536000, immutable" {
		t.Errorf("cache-control = %q, want the immutable private policy", got)
	}
	// 照 = E7 85 A7, 片 = E7 89 87, space percent-encoded: every byte
	// outside the RFC 5987 attribute set is %XX, none raw.
	wantDisposition := "attachment; filename*=UTF-8''%E7%85%A7%E7%89%87%20IMG_1234.jpg"
	if got := rec.Header().Get("Content-Disposition"); got != wantDisposition {
		t.Errorf("content-disposition = %q, want %q", got, wantDisposition)
	}
	if !bytes.Equal(rec.Body.Bytes(), payload) {
		t.Errorf("original body = %d bytes, want the %d uploaded bytes verbatim", rec.Body.Len(), len(payload))
	}

	// A conditional download answers 304 like any static asset.
	conditional := env.serveGet(t, "/api/v1/assets/"+created.Id.String()+"/original", env.cookie,
		map[string]string{"If-None-Match": `"` + checksum[:16] + `"`})
	if conditional.Code != http.StatusNotModified {
		t.Fatalf("conditional original status = %d, want %d", conditional.Code, http.StatusNotModified)
	}
}

// TestGetAssetOriginalRange pins the range semantics through a real
// uploaded video: a satisfiable single range answers 206 with the
// exact slice and Content-Range, an unsatisfiable range answers 416
// with the total size (the Go ServeContent semantics the contract
// documents).
func TestGetAssetOriginalRange(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	video, err := os.ReadFile(filepath.Join("..", "media", "testdata", "tiny.mp4"))
	if err != nil {
		t.Fatalf("read video fixture: %v", err)
	}
	created := decodeAsset(t, env.postUpload(t, map[string]string{
		"deviceAssetId": "device-range-1",
		"fileName":      "VID_0002.mp4",
	}, []assetUploadFile{{field: "photo", name: "VID_0002.mp4", content: video}}))
	if created.Kind != apigen.AssetResponseKindVideo {
		t.Fatalf("upload kind = %q, want video", created.Kind)
	}

	target := "/api/v1/assets/" + created.Id.String() + "/original"

	partial := env.serveGet(t, target, env.cookie, map[string]string{"Range": "bytes=0-99"})
	if partial.Code != http.StatusPartialContent {
		t.Fatalf("range status = %d, want %d (body %q)", partial.Code, http.StatusPartialContent, partial.Body.String())
	}
	if got := partial.Header().Get("Content-Range"); got != "bytes 0-99/101796" {
		t.Errorf("content-range = %q, want bytes 0-99/101796", got)
	}
	if !bytes.Equal(partial.Body.Bytes(), video[0:100]) {
		t.Errorf("range body = %d bytes, want the first 100 bytes of the file", partial.Body.Len())
	}

	mid := env.serveGet(t, target, env.cookie, map[string]string{"Range": "bytes=1000-1099"})
	if mid.Code != http.StatusPartialContent {
		t.Fatalf("mid range status = %d, want %d", mid.Code, http.StatusPartialContent)
	}
	if !bytes.Equal(mid.Body.Bytes(), video[1000:1100]) {
		t.Errorf("mid range body = %d bytes, want the 1000-1099 slice", mid.Body.Len())
	}

	unsatisfiable := env.serveGet(t, target, env.cookie, map[string]string{"Range": "bytes=999999999-"})
	if unsatisfiable.Code != http.StatusRequestedRangeNotSatisfiable {
		t.Fatalf("unsatisfiable range status = %d, want %d", unsatisfiable.Code, http.StatusRequestedRangeNotSatisfiable)
	}
	if got := unsatisfiable.Header().Get("Content-Range"); got != "bytes */101796" {
		t.Errorf("unsatisfiable content-range = %q, want bytes */101796", got)
	}
}
