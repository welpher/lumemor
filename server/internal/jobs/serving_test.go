// Serving tests of the asset endpoints through the REAL pipeline:
// the environment wires the queue, the upload endpoint enqueues the
// metadata and thumbnail jobs, and the assertions read what the
// endpoints serve after asset.ready fired. The handler mechanics
// (headers, conditionals, ranges, anti-enumeration) are pinned by the
// api package's serving tests; these pin the pipeline-produced truth:
// the detail the jobs refined, the AVIF bytes the thumbnail job
// derived, and the poster of a video asset.
package jobs

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	apigen "github.com/welpher/lumemor/server/internal/api/gen"
	"github.com/welpher/lumemor/server/internal/media"
	"github.com/welpher/lumemor/server/internal/storage"
)

// serveGet fires an authenticated GET against the environment's HTTP
// surface with optional extra headers and returns its recorder.
func (e *env) serveGet(t *testing.T, target string, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, target, nil)
	req.AddCookie(e.cookie)
	for name, value := range headers {
		req.Header.Set(name, value)
	}
	rec := httptest.NewRecorder()
	e.router.ServeHTTP(rec, req)
	return rec
}

// decodeServedAsset unmarshals an AssetResponse body (the upload
// endpoint's answer shape).
func decodeServedAsset(t *testing.T, rec *httptest.ResponseRecorder) apigen.AssetResponse {
	t.Helper()
	var asset apigen.AssetResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &asset); err != nil {
		t.Fatalf("decode asset body %q: %v", rec.Body.String(), err)
	}
	return asset
}

// decodeServedDetail unmarshals an AssetDetail body (the detail
// endpoint's answer shape).
func decodeServedDetail(t *testing.T, rec *httptest.ResponseRecorder) apigen.AssetDetail {
	t.Helper()
	var detail apigen.AssetDetail
	if err := json.Unmarshal(rec.Body.Bytes(), &detail); err != nil {
		t.Fatalf("decode detail body %q: %v", rec.Body.String(), err)
	}
	return detail
}

// sha256Hex digests data independently of the upload receiver.
func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// readFileOf reads the bytes at path, failing the test when they are
// absent.
func readFileOf(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return data
}

// requireServedDimensions decodes the served image bytes through the
// media package's decoder (parked in a scratch file first) and pins
// their pixel dimensions -- the served copy is asserted, not only the
// file the job wrote.
func requireServedDimensions(t *testing.T, body []byte, wantWidth, wantHeight int) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "served-copy")
	if err := os.WriteFile(path, body, 0o644); err != nil {
		t.Fatalf("write served copy for decoding: %v", err)
	}
	width, height, err := media.ImageDimensions(path)
	if err != nil {
		t.Fatalf("decode served copy: %v", err)
	}
	if width != wantWidth || height != wantHeight {
		t.Errorf("served copy = %dx%d, want %dx%d", width, height, wantWidth, wantHeight)
	}
}

// TestServeReadyAssetDetailAndThumbnail is the mandated ready flow:
// the golden JPEG (real EXIF) is uploaded through the HTTP endpoint,
// the pipeline refines it (metadata job + thumbnail job), and the
// detail endpoint answers the refinements -- blurhash, the EXIF hot
// fields, dimensions, the EXIF-resolved capture time, readiness --
// while the thumbnail endpoint serves the derived AVIF with the
// static-asset validators, its bytes identical to the file the job
// left on disk, and a matching If-None-Match answers 304.
func TestServeReadyAssetDetailAndThumbnail(t *testing.T) {
	requireExiftool(t)
	requireThumbnails(t)
	e := newEnv(t)

	golden := goldenFixture(t)
	rec := e.postUpload(t, map[string]string{
		"deviceAssetId":   "device-serve-image",
		"fileName":        "IMG_1234.jpg",
		"deviceCreatedAt": "2020-03-01T12:00:00Z",
		"deviceTimeZone":  "Asia/Shanghai",
	}, []uploadFileFixture{{field: "photo", name: "IMG_1234.jpg", content: golden}})
	if rec.Code != http.StatusCreated {
		t.Fatalf("golden upload status = %d, want %d (body %q)", rec.Code, http.StatusCreated, rec.Body.String())
	}
	asset := decodeServedAsset(t, rec)
	checksum := sha256Hex(golden)

	eventually(t, jobWaitTimeout, "the golden asset to become ready", func() bool {
		return e.assetOf(t, asset.Id).readyAt.Valid
	})

	detail := decodeServedDetail(t, e.serveGet(t, "/api/v1/assets/"+asset.Id.String(), nil))

	// The detail carries the refinements the jobs produced.
	if detail.Id != asset.Id {
		t.Errorf("detail id = %s, want the uploaded asset %s", detail.Id, asset.Id)
	}
	if detail.Kind != apigen.AssetDetailKindImage {
		t.Errorf("detail kind = %q, want image", detail.Kind)
	}
	if detail.MimeType != "image/jpeg" {
		t.Errorf("detail mimeType = %q, want image/jpeg", detail.MimeType)
	}
	if detail.Checksum != checksum {
		t.Errorf("detail checksum = %q, want %q", detail.Checksum, checksum)
	}
	if !detail.IsReady {
		t.Error("detail isReady = false after the pipeline completed, want true")
	}
	if detail.Blurhash == nil || len(*detail.Blurhash) < 6 {
		t.Errorf("detail blurhash = %v, want the thumbnail job's stored hash", detail.Blurhash)
	}
	if detail.Width == nil || *detail.Width != 64 {
		t.Errorf("detail width = %v, want the extracted 64", detail.Width)
	}
	if detail.Height == nil || *detail.Height != 64 {
		t.Errorf("detail height = %v, want the extracted 64", detail.Height)
	}
	// 2023:07:14 10:15:30 with the +05:30 EXIF offset is 04:45:30Z.
	wantTakenAt := time.Date(2023, 7, 14, 4, 45, 30, 0, time.UTC).Truncate(time.Microsecond)
	if !detail.TakenAt.Truncate(time.Microsecond).Equal(wantTakenAt) {
		t.Errorf("detail takenAt = %v, want the EXIF instant %v", detail.TakenAt, wantTakenAt)
	}
	if detail.TakenAtSource != apigen.AssetDetailTakenAtSourceExif {
		t.Errorf("detail takenAtSource = %q, want exif", detail.TakenAtSource)
	}
	if detail.Exif.CameraMake == nil || *detail.Exif.CameraMake != "Lumemor" {
		t.Errorf("detail exif.cameraMake = %v, want Lumemor", detail.Exif.CameraMake)
	}
	if detail.Exif.CameraModel == nil || *detail.Exif.CameraModel != "Golden Camera" {
		t.Errorf("detail exif.cameraModel = %v, want Golden Camera", detail.Exif.CameraModel)
	}
	if detail.Exif.Lens == nil || *detail.Exif.Lens != "Golden 35mm f/2.8" {
		t.Errorf("detail exif.lens = %v, want Golden 35mm f/2.8", detail.Exif.Lens)
	}
	if detail.Exif.Iso == nil || *detail.Exif.Iso != 200 {
		t.Errorf("detail exif.iso = %v, want 200", detail.Exif.Iso)
	}
	if detail.Exif.Aperture == nil || *detail.Exif.Aperture != 2.8 {
		t.Errorf("detail exif.aperture = %v, want 2.8", detail.Exif.Aperture)
	}
	if detail.Exif.Shutter == nil || *detail.Exif.Shutter != "1/250" {
		t.Errorf("detail exif.shutter = %v, want 1/250", detail.Exif.Shutter)
	}
	if detail.Exif.Focal == nil || *detail.Exif.Focal != 35 {
		t.Errorf("detail exif.focal = %v, want 35", detail.Exif.Focal)
	}
	if detail.Exif.Lat == nil || *detail.Exif.Lat != 48.8584 {
		t.Errorf("detail exif.lat = %v, want 48.8584", detail.Exif.Lat)
	}
	if detail.Exif.Lon == nil || *detail.Exif.Lon != 2.2945 {
		t.Errorf("detail exif.lon = %v, want 2.2945", detail.Exif.Lon)
	}
	if detail.Exif.TzOffset == nil || *detail.Exif.TzOffset != "+05:30" {
		t.Errorf("detail exif.tzOffset = %v, want +05:30", detail.Exif.TzOffset)
	}
	if detail.LivePhotoVideoId != nil {
		t.Error("detail livePhotoVideoId set, want null for a plain photo")
	}
	if detail.DurationMs != nil {
		t.Errorf("detail durationMs = %v, want null for an image", *detail.DurationMs)
	}

	// The thumbnail serves the derived AVIF with the static-asset
	// validators: the bytes are exactly the file the job left on disk.
	thumbnail := e.serveGet(t, "/api/v1/assets/"+asset.Id.String()+"/thumbnail?size=tile", nil)
	if thumbnail.Code != http.StatusOK {
		t.Fatalf("thumbnail status = %d, want %d (body %q)", thumbnail.Code, http.StatusOK, thumbnail.Body.String())
	}
	if got := thumbnail.Header().Get("Content-Type"); got != "image/avif" {
		t.Errorf("thumbnail content-type = %q, want image/avif", got)
	}
	if got := thumbnail.Header().Get("ETag"); got != `"`+checksum[:16]+`"` {
		t.Errorf("thumbnail etag = %q, want the quoted checksum prefix", got)
	}
	if got := thumbnail.Header().Get("Cache-Control"); got != "private, max-age=31536000, immutable" {
		t.Errorf("thumbnail cache-control = %q, want the immutable private policy", got)
	}
	onDisk := readFileOf(t, storage.ThumbPath(asset.Id.String(), media.SizeTile, media.FormatAvif))
	if !bytes.Equal(thumbnail.Body.Bytes(), onDisk) {
		t.Errorf("thumbnail body = %d bytes, want the %d bytes of the derived file",
			thumbnail.Body.Len(), len(onDisk))
	}
	requireServedDimensions(t, thumbnail.Body.Bytes(), 64, 64)

	// A matching If-None-Match turns the second request into a 304.
	conditional := e.serveGet(t, "/api/v1/assets/"+asset.Id.String()+"/thumbnail?size=tile",
		map[string]string{"If-None-Match": `"` + checksum[:16] + `"`})
	if conditional.Code != http.StatusNotModified {
		t.Fatalf("conditional thumbnail status = %d, want %d", conditional.Code, http.StatusNotModified)
	}
	if conditional.Body.Len() != 0 {
		t.Errorf("conditional thumbnail body = %q, want no body on a 304", conditional.Body.String())
	}
}

// TestServeVideoPosterThumbnail is the mandated video flow: a real
// (tiny) MP4 uploaded through the endpoint becomes ready through the
// metadata job and the poster extraction, and its thumbnail endpoint
// answers the poster JPEG -- whatever size the request names -- with
// the poster bytes the job derived.
func TestServeVideoPosterThumbnail(t *testing.T) {
	requirePosterFFmpeg(t)
	requireExiftool(t)
	e := newEnv(t)

	video := readFileOf(t, filepath.Join("..", "media", "testdata", "tiny.mp4"))
	rec := e.postUpload(t, map[string]string{
		"deviceAssetId":   "device-serve-video",
		"fileName":        "VID_0001.mp4",
		"deviceCreatedAt": "2020-03-01T12:00:00Z",
	}, []uploadFileFixture{{field: "photo", name: "VID_0001.mp4", content: video}})
	if rec.Code != http.StatusCreated {
		t.Fatalf("video upload status = %d, want %d (body %q)", rec.Code, http.StatusCreated, rec.Body.String())
	}
	asset := decodeServedAsset(t, rec)
	if asset.Kind != apigen.AssetResponseKindVideo {
		t.Fatalf("video upload kind = %q, want video", asset.Kind)
	}
	checksum := sha256Hex(video)

	eventually(t, jobWaitTimeout, "the video asset to become ready", func() bool {
		return e.assetOf(t, asset.Id).readyAt.Valid
	})

	for _, size := range []string{"tile", "view", "original"} {
		poster := e.serveGet(t, "/api/v1/assets/"+asset.Id.String()+"/thumbnail?size="+size, nil)
		if poster.Code != http.StatusOK {
			t.Fatalf("video thumbnail size %s status = %d, want %d", size, poster.Code, http.StatusOK)
		}
		if got := poster.Header().Get("Content-Type"); got != "image/jpeg" {
			t.Errorf("video thumbnail size %s content-type = %q, want image/jpeg (the poster frame)", size, got)
		}
		if got := poster.Header().Get("ETag"); got != `"`+checksum[:16]+`"` {
			t.Errorf("video thumbnail size %s etag = %q, want the quoted checksum prefix", size, got)
		}
		onDisk := readFileOf(t, storage.PosterPath(asset.Id.String()))
		if !bytes.Equal(poster.Body.Bytes(), onDisk) {
			t.Errorf("video thumbnail size %s body = %d bytes, want the %d poster bytes the job derived",
				size, poster.Body.Len(), len(onDisk))
		}
	}

	// The detail of a video answers its refined duration.
	detail := decodeServedDetail(t, e.serveGet(t, "/api/v1/assets/"+asset.Id.String(), nil))
	if detail.DurationMs == nil || *detail.DurationMs <= 0 {
		t.Errorf("video detail durationMs = %v, want the extracted positive duration", detail.DurationMs)
	}
	if !detail.IsReady {
		t.Error("video detail isReady = false after the pipeline completed, want true")
	}
}
