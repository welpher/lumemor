package api

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"image"
	"image/color"
	"image/jpeg"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	dbgen "github.com/welpher/lumemor/server/internal/db/gen"
)

// The ingest-time storage template tests: the path the upload
// pipeline places files at renders from the instance's template with
// the taken-at fallback chain (EXIF header time, device time, upload
// time), the (owner+checksum) dedup runs before any placement so true
// duplicates never collide, collisions append ~n under sequential and
// concurrent same-target uploads, a Live Photo's two parts land in
// one folder, and changing the template moves new uploads without
// touching what already exists.

// exifTag is one ASCII tag of the test fixture's EXIF structure.
type exifTag struct {
	tag   uint16
	value string
}

// buildExifTIFF assembles the minimal TIFF structure the ingest's
// header parser reads: IFD0 with the camera identity and the Exif
// sub-IFD pointer, the sub-IFD with the capture wall clock and its
// offsets. Byte order is little-endian, the parser's common case.
func buildExifTIFF(ifd0, exif []exifTag) []byte {
	bo := binary.LittleEndian
	ifd0Size := 2 + 12*(len(ifd0)+1) + 4 // +1: the Exif sub-IFD pointer
	exifSize := 2 + 12*len(exif) + 4
	exifOffset := 8 + ifd0Size
	valueBase := exifOffset + exifSize

	valueOffsets := make(map[uint16]uint32)
	var valueArea []byte
	entry := func(tag, typ uint16, count uint32, value uint32) []byte {
		buf := make([]byte, 12)
		bo.PutUint16(buf[0:2], tag)
		bo.PutUint16(buf[2:4], typ)
		bo.PutUint32(buf[4:8], count)
		bo.PutUint32(buf[8:12], value)
		return buf
	}
	asciiEntry := func(e exifTag) []byte {
		count := uint32(len(e.value) + 1)
		if count <= 4 {
			value := make([]byte, 4)
			copy(value, e.value)
			return entry(e.tag, 2, count, bo.Uint32(value))
		}
		if len(valueArea)%2 == 1 {
			valueArea = append(valueArea, 0)
		}
		valueOffsets[e.tag] = uint32(valueBase + len(valueArea))
		valueArea = append(valueArea, e.value...)
		valueArea = append(valueArea, 0)
		return entry(e.tag, 2, count, valueOffsets[e.tag])
	}

	buf := make([]byte, 0, valueBase)
	// The header: "II", the 42 magic and the IFD0 offset at 8, all in
	// little-endian byte order.
	buf = append(buf, 'I', 'I', 42, 0, 8, 0, 0, 0)
	var count [2]byte
	bo.PutUint16(count[:], uint16(len(ifd0)+1))
	buf = append(buf, count[:]...)
	for _, e := range ifd0 {
		buf = append(buf, asciiEntry(e)...)
	}
	buf = append(buf, entry(0x8769, 4, 1, uint32(exifOffset))...)
	buf = append(buf, 0, 0, 0, 0)
	bo.PutUint16(count[:], uint16(len(exif)))
	buf = append(buf, count[:]...)
	for _, e := range exif {
		buf = append(buf, asciiEntry(e)...)
	}
	buf = append(buf, 0, 0, 0, 0)
	buf = append(buf, valueArea...)
	return buf
}

// exifJPEG builds a valid JPEG carrying one EXIF APP1 segment: real
// encoded image bytes, so the upload receiver's content sniff accepts
// the payload while the header parser reads the capture context.
func exifJPEG(t *testing.T, ifd0, exif []exifTag) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	img.Set(0, 0, color.RGBA{R: 0x7f, G: 0x33, B: 0xcc, A: 0xff})
	var encoded bytes.Buffer
	if err := jpeg.Encode(&encoded, img, nil); err != nil {
		t.Fatalf("encode JPEG body: %v", err)
	}
	segmentLength := 6 + len(buildExifTIFF(ifd0, exif))
	var buf bytes.Buffer
	buf.Write([]byte{0xFF, 0xD8})
	buf.Write([]byte{0xFF, 0xE1})
	buf.Write(binary.BigEndian.AppendUint16(nil, uint16(2+segmentLength)))
	buf.WriteString("Exif\x00\x00")
	buf.Write(buildExifTIFF(ifd0, exif))
	buf.Write(encoded.Bytes()[2:])
	return buf.Bytes()
}

// setStorageTemplate writes the instance's storage template setting;
// an empty value removes the row (the default applies).
func (e *assetEnv) setStorageTemplate(t *testing.T, value string) {
	t.Helper()
	if value == "" {
		e.exec(t, "DELETE FROM instance_settings WHERE key = 'storage_template'")
		return
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("encode template %q: %v", value, err)
	}
	e.exec(t, `INSERT INTO instance_settings (key, value) VALUES ('storage_template', $1::jsonb)
		ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, updated_at = now()`, string(encoded))
}

// requireStoredBytes asserts the file at rel exists under the
// environment's data root and holds exactly want bytes.
func (e *assetEnv) requireStoredBytes(t *testing.T, rel string, want []byte) {
	t.Helper()
	path := filepath.Join(e.dataDir, filepath.FromSlash(rel))
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read stored original %s: %v", path, err)
	}
	if !bytes.Equal(data, want) {
		t.Errorf("stored original %s holds %d bytes, want the %d uploaded bytes", path, len(data), len(want))
	}
}

// TestUploadPathRendersInstanceTemplate pins the instance setting as
// the render source: with the month-name preset, a device-reported
// capture time of March 2026 lands the file under 2026/March, with
// the sanitized file name and the normalized extension -- and the
// ROW's taken_at stays the device-reported time (the metadata job
// refines it; only the path renders at ingest).
func TestUploadPathRendersInstanceTemplate(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	env.setStorageTemplate(t, "{{y}}/{{MMMM}}/{{filename}}")
	payload := jpegFixture(t)

	rec := env.postUpload(t, map[string]string{
		"deviceAssetId":   "tpl-1",
		"fileName":        "IMG_777.jpeg",
		"deviceCreatedAt": "2026-03-05T09:15:00+08:00",
	}, []assetUploadFile{{field: "photo", name: "IMG_777.jpeg", content: payload}})
	if rec.Code != http.StatusCreated {
		t.Fatalf("upload status = %d, want %d (body %q)", rec.Code, http.StatusCreated, rec.Body.String())
	}
	asset := decodeAsset(t, rec)

	wantPath := "originals/" + env.sessionUserID(t).String() + "/2026/March/IMG_777.jpg"
	row := env.assetByChecksum(t, sha256Hex(payload))
	if row.FilePath != wantPath {
		t.Errorf("row file_path = %q, want %q", row.FilePath, wantPath)
	}
	if row.ID.Bytes != asset.Id {
		t.Errorf("row id = %s, want the response id %s", uuid.UUID(row.ID.Bytes), asset.Id)
	}
	env.requireStoredBytes(t, wantPath, payload)
	if !row.TakenAt.Time.Equal(time.Date(2026, 3, 5, 1, 15, 0, 0, time.UTC)) {
		t.Errorf("row taken_at = %v, want the device-reported instant (the job refines it, the row does not re-render)",
			row.TakenAt.Time)
	}
}

// TestUploadPathUsesExifHeaderTime pins the fallback chain's first
// link: an EXIF capture wall clock WITH its offset beats the
// device-reported time for the PATH -- the folder says 2023/07, the
// capture's own wall clock -- while the row's taken_at keeps the
// device time until the metadata job refines it.
func TestUploadPathUsesExifHeaderTime(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	payload := exifJPEG(t,
		[]exifTag{
			{tag: 0x010F, value: "Lumemor"},    // Make
			{tag: 0x0110, value: "Golden Cam"}, // Model
		},
		[]exifTag{
			{tag: 0x9003, value: "2023:07:14 10:15:30"}, // DateTimeOriginal
			{tag: 0x9010, value: "+05:30"},              // OffsetTime
		})

	rec := env.postUpload(t, map[string]string{
		"deviceAssetId":   "tpl-exif-1",
		"fileName":        "IMG_0001.jpg",
		"deviceCreatedAt": "2020-01-01T00:00:00Z",
	}, []assetUploadFile{{field: "photo", name: "IMG_0001.jpg", content: payload}})
	if rec.Code != http.StatusCreated {
		t.Fatalf("upload status = %d, want %d (body %q)", rec.Code, http.StatusCreated, rec.Body.String())
	}

	row := env.assetByChecksum(t, sha256Hex(payload))
	wantPath := "originals/" + env.sessionUserID(t).String() + "/2023/07/IMG_0001.jpg"
	if row.FilePath != wantPath {
		t.Errorf("row file_path = %q, want the EXIF wall clock's %q", row.FilePath, wantPath)
	}
	env.requireStoredBytes(t, wantPath, payload)
	if want, got := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC), row.TakenAt.Time; !got.Equal(want) {
		t.Errorf("row taken_at = %v, want the device time %v (the metadata job refines the row)", got, want)
	}
}

// TestUploadPathExifZoneChain pins the zone resolution of the EXIF
// link: the per-tag OffsetTimeOriginal when the composite OffsetTime
// is absent, the device-declared zone when neither offset tag is
// written, and the device-reported time when the wall clock cannot
// resolve a zone at all (a bare clock is never guessed).
func TestUploadPathExifZoneChain(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	owner := env.sessionUserID(t).String()

	// The per-tag offset: 2023:07:14 10:15:30 at +07:00 -- July 2023.
	perTag := exifJPEG(t, nil,
		[]exifTag{
			{tag: 0x9003, value: "2023:07:14 10:15:30"},
			{tag: 0x9011, value: "+07:00"},
		})
	rec := env.postUpload(t, map[string]string{
		"deviceAssetId": "tpl-exif-2",
		"fileName":      "pertag.jpg",
	}, []assetUploadFile{{field: "photo", name: "pertag.jpg", content: perTag}})
	if rec.Code != http.StatusCreated {
		t.Fatalf("per-tag offset upload status = %d, want %d", rec.Code, http.StatusCreated)
	}
	row := env.assetByChecksum(t, sha256Hex(perTag))
	if want := "originals/" + owner + "/2023/07/pertag.jpg"; row.FilePath != want {
		t.Errorf("per-tag offset path = %q, want %q", row.FilePath, want)
	}

	// No offset tags, but the device declared its zone: the wall clock
	// 2024:12:31 23:30 read in Asia/Shanghai stays December 2024.
	deviceZone := exifJPEG(t, nil,
		[]exifTag{{tag: 0x9003, value: "2024:12:31 23:30:00"}})
	rec = env.postUpload(t, map[string]string{
		"deviceAssetId":   "tpl-exif-3",
		"fileName":        "zone.jpg",
		"deviceCreatedAt": "2020-01-01T00:00:00Z",
		"deviceTimeZone":  "Asia/Shanghai",
	}, []assetUploadFile{{field: "photo", name: "zone.jpg", content: deviceZone}})
	if rec.Code != http.StatusCreated {
		t.Fatalf("device zone upload status = %d, want %d", rec.Code, http.StatusCreated)
	}
	row = env.assetByChecksum(t, sha256Hex(deviceZone))
	if want := "originals/" + owner + "/2024/12/zone.jpg"; row.FilePath != want {
		t.Errorf("device zone path = %q, want %q", row.FilePath, want)
	}

	// No offset, no zone: the wall clock is not guessed; the
	// device-reported time governs.
	noZone := exifJPEG(t, nil,
		[]exifTag{{tag: 0x9003, value: "1999:01:01 00:00:00"}})
	rec = env.postUpload(t, map[string]string{
		"deviceAssetId":   "tpl-exif-4",
		"fileName":        "nozone.jpg",
		"deviceCreatedAt": "2025-06-15T12:00:00Z",
	}, []assetUploadFile{{field: "photo", name: "nozone.jpg", content: noZone}})
	if rec.Code != http.StatusCreated {
		t.Fatalf("zoneless upload status = %d, want %d", rec.Code, http.StatusCreated)
	}
	row = env.assetByChecksum(t, sha256Hex(noZone))
	if want := "originals/" + owner + "/2025/06/nozone.jpg"; row.FilePath != want {
		t.Errorf("zoneless path = %q, want the device time's %q", row.FilePath, want)
	}
}

// TestUploadPathCollisionSuffixesSequential pins the collision
// contract: three DIFFERENT payloads whose template render lands on
// one target (the same client-declared fileName and capture date)
// place at the base name, ~1 and ~2 -- no error, no overwrite, every
// file holding its own bytes.
func TestUploadPathCollisionSuffixesSequential(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	owner := env.sessionUserID(t).String()

	colors := []color.RGBA{
		{R: 0x11, G: 0x22, B: 0x33, A: 0xff},
		{R: 0x44, G: 0x55, B: 0x66, A: 0xff},
		{R: 0x77, G: 0x88, B: 0x99, A: 0xff},
	}
	want := []string{
		"originals/" + owner + "/2026/07/IMG_5100.jpg",
		"originals/" + owner + "/2026/07/IMG_5100~1.jpg",
		"originals/" + owner + "/2026/07/IMG_5100~2.jpg",
	}
	for i, pixel := range colors {
		payload := jpegColorFixture(t, pixel)
		if i > 0 && sha256Hex(payload) == sha256Hex(jpegColorFixture(t, colors[i-1])) {
			t.Fatalf("collision fixtures %d and %d encode to identical bytes; pick more distinct colors", i, i-1)
		}
		rec := env.postUpload(t, map[string]string{
			"deviceAssetId":   "collide-" + string(rune('a'+i)),
			"fileName":        "IMG_5100.jpg",
			"deviceCreatedAt": "2026-07-20T10:00:00Z",
		}, []assetUploadFile{{field: "photo", name: "IMG_5100.jpg", content: payload}})
		if rec.Code != http.StatusCreated {
			t.Fatalf("collision upload %d status = %d, want %d (body %q)", i, rec.Code, http.StatusCreated, rec.Body.String())
		}
		row := env.assetByChecksum(t, sha256Hex(payload))
		if row.FilePath != want[i] {
			t.Errorf("collision upload %d path = %q, want %q", i, row.FilePath, want[i])
		}
		env.requireStoredBytes(t, want[i], payload)
	}
	if got := env.countFiles(t, filepath.Join(env.dataDir, "originals")); got != len(want) {
		t.Errorf("originals holds %d files after the collisions, want %d", got, len(want))
	}
}

// TestUploadPathCollisionConcurrent pins the collision handling under
// concurrency: eight uploads of one rendered target race (distinct
// bytes, distinct device ids), every one answers 201 with its own
// file, and the paths are all distinct -- the link's atomicity hands
// each racer the base name or its own suffix, never an overwrite.
func TestUploadPathCollisionConcurrent(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	const racers = 8

	bodies := make([]*bytes.Buffer, racers)
	types := make([]string, racers)
	contents := make([][]byte, racers)
	seenChecksums := make(map[string]bool)
	for i := range bodies {
		contents[i] = jpegColorFixture(t, color.RGBA{
			R: uint8(0x10 + i*0x12), G: uint8(0x50 + i*0x11), B: uint8(0x90 + i*0x13), A: 0xff,
		})
		if seenChecksums[sha256Hex(contents[i])] {
			t.Fatalf("racer %d's payload collides with an earlier racer's bytes; pick more distinct colors", i)
		}
		seenChecksums[sha256Hex(contents[i])] = true
		bodies[i], types[i] = buildUploadBody(t, map[string]string{
			"deviceAssetId":   "race-" + string(rune('a'+i)),
			"fileName":        "IMG_6000.jpg",
			"deviceCreatedAt": "2026-07-21T10:00:00Z",
		}, []assetUploadFile{{field: "photo", name: "IMG_6000.jpg", content: contents[i]}})
	}

	start := make(chan struct{})
	codes := make([]int, racers)
	ids := make([]string, racers)
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
			ids[i] = decodeAsset(t, rec).Id.String()
		}(i)
	}
	close(start)
	wg.Wait()

	paths := make(map[string]bool)
	for i, code := range codes {
		if code != http.StatusCreated {
			t.Fatalf("concurrent collision upload %d status = %d, want %d", i, code, http.StatusCreated)
		}
		row := env.assetRowOf(t, uuid.MustParse(ids[i]))
		if paths[row.FilePath] {
			t.Errorf("uploads %d and an earlier racer share the path %q", i, row.FilePath)
		}
		paths[row.FilePath] = true
		env.requireStoredBytes(t, row.FilePath, contents[i])
	}
	if got := env.countFiles(t, filepath.Join(env.dataDir, "originals")); got != racers {
		t.Errorf("originals holds %d files after the race, want %d", got, racers)
	}
}

// TestUploadPathDedupShortCircuitsBeforePlacement pins the dedup's
// position: the SAME owner re-uploading the SAME bytes (whatever the
// file name) answers 200 duplicate and places NOTHING -- no ~1 twin,
// no second file, the originals tree exactly as the first upload
// left it.
func TestUploadPathDedupShortCircuitsBeforePlacement(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	payload := jpegFixture(t)

	first := env.postUpload(t, map[string]string{
		"deviceAssetId":   "dup-1",
		"fileName":        "IMG_8000.jpg",
		"deviceCreatedAt": "2026-08-01T10:00:00Z",
	}, []assetUploadFile{{field: "photo", name: "IMG_8000.jpg", content: payload}})
	if first.Code != http.StatusCreated {
		t.Fatalf("first upload status = %d, want %d", first.Code, http.StatusCreated)
	}

	// The re-upload: same bytes, a DIFFERENT file name -- the
	// (owner+checksum) dedup, not the name, decides.
	second := env.postUpload(t, map[string]string{
		"deviceAssetId":   "dup-2",
		"fileName":        "renamed.jpg",
		"deviceCreatedAt": "2027-01-01T10:00:00Z",
	}, []assetUploadFile{{field: "photo", name: "renamed.jpg", content: payload}})
	if second.Code != http.StatusOK {
		t.Fatalf("duplicate upload status = %d, want %d (body %q)", second.Code, http.StatusOK, second.Body.String())
	}
	if state := decodeAsset(t, second).State; state != "duplicate" {
		t.Errorf("duplicate state = %q, want duplicate", state)
	}
	if got := env.countFiles(t, filepath.Join(env.dataDir, "originals")); got != 1 {
		t.Errorf("originals holds %d files after the duplicate, want 1 (the dedup ran before any placement)", got)
	}
	if got := env.queryInt(t, "SELECT count(*) FROM assets"); got != 1 {
		t.Errorf("assets rows = %d, want 1", got)
	}
	env.requireStagingEmpty(t)
}

// TestUploadLivePhotoSharesFolder pins the pair flow: the motion
// video's path renders from the STILL's capture time (its own header
// carries no EXIF), so both parts of a Live Photo land in the same
// folder under the default template, each under its own name.
func TestUploadLivePhotoSharesFolder(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	owner := env.sessionUserID(t).String()
	photo := exifJPEG(t,
		[]exifTag{{tag: 0x010F, value: "Lumemor"}},
		[]exifTag{
			{tag: 0x9003, value: "2026:09:10 08:30:00"},
			{tag: 0x9010, value: "+02:00"},
		})
	video := movFixture()

	rec := env.postUpload(t, map[string]string{
		"deviceAssetId":   "live-tpl-1",
		"fileName":        "IMG_9000.jpg",
		"deviceCreatedAt": "2020-05-05T05:05:05Z",
	}, []assetUploadFile{
		{field: "photo", name: "IMG_9000.jpg", content: photo},
		{field: "liveVideo", name: "IMG_9000.mov", content: video},
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("live photo upload status = %d, want %d (body %q)", rec.Code, http.StatusCreated, rec.Body.String())
	}

	photoRow := env.assetByChecksum(t, sha256Hex(photo))
	wantPhoto := "originals/" + owner + "/2026/09/IMG_9000.jpg"
	if photoRow.FilePath != wantPhoto {
		t.Errorf("photo path = %q, want the EXIF wall clock's %q", photoRow.FilePath, wantPhoto)
	}
	videoRow, err := env.queries.GetAssetByOwnerChecksum(context.Background(), dbgen.GetAssetByOwnerChecksumParams{
		OwnerID:  pgUUID(env.sessionUserID(t)),
		Checksum: sha256Hex(video),
	})
	if err != nil {
		t.Fatalf("load the component row: %v", err)
	}
	wantVideo := "originals/" + owner + "/2026/09/IMG_9000.mov"
	if videoRow.FilePath != wantVideo {
		t.Errorf("component path = %q, want the still's date folder %q (the pair shares it)", videoRow.FilePath, wantVideo)
	}
	env.requireStoredBytes(t, wantPhoto, photo)
	env.requireStoredBytes(t, wantVideo, video)
}

// TestUploadTemplateChangeMovesOnlyNewUploads pins the change
// semantics: after the template changes, a NEW upload follows the new
// layout while the EXISTING asset's row and file stay exactly where
// they were -- nothing re-renders, nothing moves.
func TestUploadTemplateChangeMovesOnlyNewUploads(t *testing.T) {
	env := newAssetEnv(t, testUploadLimit)
	owner := env.sessionUserID(t).String()

	before := jpegColorFixture(t, color.RGBA{R: 0xE1, G: 0x10, B: 0x10, A: 0xff})
	rec := env.postUpload(t, map[string]string{
		"deviceAssetId":   "change-1",
		"fileName":        "IMG_9500.jpg",
		"deviceCreatedAt": "2026-02-02T10:00:00Z",
	}, []assetUploadFile{{field: "photo", name: "IMG_9500.jpg", content: before}})
	if rec.Code != http.StatusCreated {
		t.Fatalf("first upload status = %d, want %d", rec.Code, http.StatusCreated)
	}
	firstRow := env.assetByChecksum(t, sha256Hex(before))
	firstPath := "originals/" + owner + "/2026/02/IMG_9500.jpg"
	if firstRow.FilePath != firstPath {
		t.Fatalf("first upload path = %q, want %q", firstRow.FilePath, firstPath)
	}

	env.setStorageTemplate(t, "{{filetype}}/{{assetId}}")

	after := jpegColorFixture(t, color.RGBA{R: 0xE2, G: 0x10, B: 0x10, A: 0xff})
	rec = env.postUpload(t, map[string]string{
		"deviceAssetId":   "change-2",
		"fileName":        "IMG_9501.jpg",
		"deviceCreatedAt": "2026-02-02T10:00:00Z",
	}, []assetUploadFile{{field: "photo", name: "IMG_9501.jpg", content: after}})
	if rec.Code != http.StatusCreated {
		t.Fatalf("second upload status = %d, want %d", rec.Code, http.StatusCreated)
	}
	second := decodeAsset(t, rec)
	secondPath := "originals/" + owner + "/IMG/" + second.Id.String() + ".jpg"
	secondRow := env.assetByChecksum(t, sha256Hex(after))
	if secondRow.FilePath != secondPath {
		t.Errorf("new upload path = %q, want the new template's %q", secondRow.FilePath, secondPath)
	}

	// The existing asset is untouched: same row path, same file, same
	// bytes.
	if row := env.assetByChecksum(t, sha256Hex(before)); row.FilePath != firstPath {
		t.Errorf("existing row's path changed to %q, want it pinned at %q", row.FilePath, firstPath)
	}
	env.requireStoredBytes(t, firstPath, before)
	env.requireStoredBytes(t, secondPath, after)
}
