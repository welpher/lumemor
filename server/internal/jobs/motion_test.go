package jobs

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// The single-file motion photo's pins: the four container shapes are
// recognized from the extraction's own tags, the tail window is read
// exactly, and one metadata run carves the motion into a component row
// beside the still (linked, hidden from every listing, its own
// pipeline queued) with the bytes reused by checksum on a re-run.

// TestMotionVideoPlanRecognizesTheFourContainers pins the recognition
// table against tag maps shaped exactly like the extraction's own
// (-struct JSON): each family answers its plan, and plain files
// answer none.
func TestMotionVideoPlanRecognizesTheFourContainers(t *testing.T) {
	// 1. The Google container (HEIC): MotionPhoto plus the directory
	// entry's Length and Padding, read from the file's tail.
	google := map[string]any{
		"MotionPhoto": float64(1),
		"ContainerDirectory": []any{
			map[string]any{"Item": map[string]any{
				"Mime": "image/heic", "Semantic": "Primary", "Length": float64(0), "Padding": float64(0),
			}},
			map[string]any{"Item": map[string]any{
				"Mime": "video/mp4", "Semantic": "MotionPhoto",
				"Length": float64(1_234_567), "Padding": float64(64),
			}},
		},
	}
	plan := motionVideoPlanOf(google)
	if !plan.tail || plan.length != 1_234_567 || plan.offset != 1_234_567+64 {
		t.Errorf("google plan = %+v, want the tail length plus padding", plan)
	}

	// The JPEG shape of the same container nests one directory level
	// deeper: the walk finds the motion item either way.
	nested := map[string]any{
		"MotionPhoto": float64(1),
		"ContainerDirectory": []any{
			map[string]any{"Directory": []any{
				map[string]any{"Item": map[string]any{
					"Mime": "image/jpeg", "Semantic": "Primary", "Length": float64(0), "Padding": float64(0),
				}},
			}},
			map[string]any{"Directory": []any{
				map[string]any{"Item": map[string]any{
					"Mime": "video/mp4", "Semantic": "MotionPhoto",
					"Length": float64(987_654), "Padding": float64(0),
				}},
			}},
		},
	}
	plan = motionVideoPlanOf(nested)
	if !plan.tail || plan.length != 987_654 || plan.offset != 987_654 {
		t.Errorf("nested plan = %+v, want the nested directory's tail", plan)
	}

	// 2. The older MicroVideo shape: the offset alone addresses the tail.
	plan = motionVideoPlanOf(map[string]any{
		"MicroVideo":       float64(1),
		"MicroVideoOffset": float64(4321),
	})
	if !plan.tail || plan.length != 4321 || plan.offset != 4321 {
		t.Errorf("microvideo plan = %+v, want size-offset", plan)
	}

	// 3. Samsung's HEIC: the whole video sits in one binary tag.
	plan = motionVideoPlanOf(map[string]any{"MotionPhotoVideo": "(Binary data 2048 bytes, use -b option to extract)"})
	if plan.binaryTag != "MotionPhotoVideo" || plan.tail {
		t.Errorf("samsung plan = %+v, want the binary tag", plan)
	}

	// 4. The JPEG shape: the type flag plus the embedded file.
	plan = motionVideoPlanOf(map[string]any{
		"EmbeddedVideoType": "MotionPhoto_Data",
		"EmbeddedVideoFile": "(Binary data 900 bytes, use -b option to extract)",
	})
	if plan.binaryTag != "EmbeddedVideoFile" {
		t.Errorf("jpeg plan = %+v, want the embedded video file tag", plan)
	}

	// Not a motion photo: no flag, no plan.
	if plan := motionVideoPlanOf(map[string]any{"Make": "Lumemor"}); plan.ok() {
		t.Errorf("plain photo plan = %+v, want none", plan)
	}
	// A flag without a directory (a stripped file): nothing to read.
	if plan := motionVideoPlanOf(map[string]any{"MotionPhoto": float64(1)}); plan.ok() {
		t.Errorf("flag-only plan = %+v, want none", plan)
	}
	// The JPEG flag alone is not enough: the bytes live in the second tag.
	if plan := motionVideoPlanOf(map[string]any{"EmbeddedVideoType": "MotionPhoto_Data"}); plan.ok() {
		t.Errorf("type-only plan = %+v, want none", plan)
	}
}

// TestReadMotionBytesReadsTheTailWindow pins the tail read: the window
// is the LAST length bytes, an offset past the file and a window
// outside it are refused, and a real JPEG whose tail is the appended
// video answers exactly those bytes.
func TestReadMotionBytesReadsTheTailWindow(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "still.jpg")
	content := append([]byte("JPEG-HEAD"), []byte("THE-MOTION-TAIL")...)
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	plan := motionVideoPlan{tail: true, offset: 15, length: 15}
	video, err := readMotionBytes(context.Background(), path, int64(len(content)), plan)
	if err != nil {
		t.Fatalf("read the tail window: %v", err)
	}
	if string(video) != "THE-MOTION-TAIL" {
		t.Errorf("tail = %q, want the appended motion", video)
	}

	if _, err := readMotionBytes(context.Background(), path, int64(len(content)),
		motionVideoPlan{tail: true, offset: 5000, length: 10}); err == nil {
		t.Error("a window past the file answered no error")
	}
}

// TestMetadataCarvesTheMotionOutOfASingleFileLivePhoto walks the whole
// silent step: a Pixel-style still (the Google container's XMP plus the
// video on its tail) goes through the metadata job, and the pair comes
// out linked -- the motion a component row beside the still, its bytes
// on disk, its own pipeline queued, the wall still showing one tile,
// and a re-run reusing the row instead of writing a second copy.
func TestMetadataCarvesTheMotionOutOfASingleFileLivePhoto(t *testing.T) {
	requireExiftool(t)
	requireFfmpegJob(t)
	e := newEnv(t)

	video := realMotionVideo(t, e.dataDir)
	still := pixelMotionPhoto(t, video)

	deviceTakenAt := time.Date(2024, 5, 2, 9, 0, 0, 0, time.UTC).Truncate(time.Microsecond)
	id := e.insertAsset(t, "originals/motion/IMG_8001.jpg", still, deviceTakenAt, "{}")

	if err := e.queue.EnqueueMetadata(context.Background(), id); err != nil {
		t.Fatalf("enqueue the metadata job: %v", err)
	}
	eventually(t, jobWaitTimeout, "the still to become ready", func() bool {
		return e.assetOf(t, id).readyAt.Valid
	})

	// The still points at its motion, and that row is a component the
	// listings never show, marked as the extracted shape.
	var motionID string
	var component, extracted bool
	var checksum, relPath, mimeType string
	var fileSize int64
	if err := e.pool.QueryRow(context.Background(), `
		SELECT a.live_photo_video_id::text, m.file_path, m.checksum, m.is_component,
		       m.motion_extracted, m.mime_type, m.file_size
		FROM assets a
		JOIN assets m ON m.id = a.live_photo_video_id
		WHERE a.id = $1`, pgUUID(id)).
		Scan(&motionID, &relPath, &checksum, &component, &extracted, &mimeType, &fileSize); err != nil {
		t.Fatalf("read the pair: %v", err)
	}
	if !component {
		t.Error("the motion row is not a component: it would appear in the listings")
	}
	if !extracted {
		t.Error("the motion row is not marked motion_extracted: the download switch could not tell it from a paired file")
	}
	if mimeType != "video/mp4" {
		t.Errorf("motion mime type = %q, want video/mp4", mimeType)
	}
	if fileSize != int64(len(video)) {
		t.Errorf("motion file size = %d, want the extracted %d bytes", fileSize, len(video))
	}
	sum := sha1.Sum(video)
	if checksum != hex.EncodeToString(sum[:]) {
		t.Errorf("motion checksum = %q, want the sha1 of the extracted bytes", checksum)
	}

	// The bytes sit beside the still, under the still's own stem, and
	// they ARE the tail the fixture appended.
	if !strings.HasPrefix(relPath, "originals/motion/IMG_8001") {
		t.Errorf("motion stored at %q, want the still's own directory and stem", relPath)
	}
	absolute, err := filepath.Abs(filepath.Join(e.dataDir, filepath.FromSlash(relPath)))
	if err != nil {
		t.Fatalf("resolve the motion path: %v", err)
	}
	written, err := os.ReadFile(absolute)
	if err != nil {
		t.Fatalf("read the extracted motion: %v", err)
	}
	if string(written) != string(video) {
		t.Errorf("extracted %d bytes, want the appended %d", len(written), len(video))
	}

	// ONE TILE: the still is the only timeline row of the pair, and the
	// motion is a member of the owner's sync album (the serving union
	// reads membership, so the play stream resolves by the motion's id).
	var timelineRows int
	if err := e.pool.QueryRow(context.Background(),
		`SELECT count(*) FROM assets WHERE owner_id = $1 AND NOT is_component`,
		pgUUID(e.ownerID)).Scan(&timelineRows); err != nil {
		t.Fatalf("count the timeline rows: %v", err)
	}
	if timelineRows != 1 {
		t.Errorf("timeline rows = %d, want the still alone", timelineRows)
	}
	var albumMembers int
	if err := e.pool.QueryRow(context.Background(), `
		SELECT count(*) FROM album_items WHERE asset_id = $1`, motionID).Scan(&albumMembers); err != nil {
		t.Fatalf("count the motion's album membership: %v", err)
	}
	if albumMembers != 1 {
		t.Errorf("motion album memberships = %d, want the sync album", albumMembers)
	}

	// The still's own archive now carries the pair, with the motion
	// marked as the embedded shape (the download switch's gate).
	var downloadMarker bool
	if err := e.pool.QueryRow(context.Background(),
		`SELECT motion_extracted FROM assets WHERE id = $1`, motionID).Scan(&downloadMarker); err != nil {
		t.Fatalf("read the download marker: %v", err)
	}
	if !downloadMarker {
		t.Error("the motion_extracted marker did not land")
	}

	// A re-run reuses the row: the same bytes never land twice, and the
	// pair's link is untouched.
	if err := e.queue.EnqueueMetadata(context.Background(), id); err != nil {
		t.Fatalf("re-enqueue the metadata job: %v", err)
	}
	time.Sleep(2 * time.Second)
	var motionRows int
	if err := e.pool.QueryRow(context.Background(),
		`SELECT count(*) FROM assets WHERE owner_id = $1 AND is_component`,
		pgUUID(e.ownerID)).Scan(&motionRows); err != nil {
		t.Fatalf("count the motion rows: %v", err)
	}
	if motionRows != 1 {
		t.Errorf("motion rows after the re-run = %d, want the one reused row", motionRows)
	}
}

// requireFfmpegJob skips the tests that generate a real video when
// ffmpeg is not on the host (the dev container carries it).
func requireFfmpegJob(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skipf("ffmpeg is not installed on this host; run the suite inside the dev container: %v", err)
	}
}

// realMotionVideo generates a one-second H.264 clip: the motion a
// camera would append (a real MP4, so the motion row's own transcode
// runs for real).
func realMotionVideo(t *testing.T, dir string) []byte {
	t.Helper()
	path := filepath.Join(dir, "motion-source.mp4")
	cmd := exec.Command("ffmpeg",
		"-f", "lavfi", "-i", "testsrc2=size=160x120:rate=15",
		"-t", "1", "-pix_fmt", "yuv420p", "-y", path)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("generate the motion fixture: %v: %s", err, out)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read the motion fixture: %v", err)
	}
	_ = os.Remove(path)
	return data
}

// pixelMotionPhoto builds a still in the Google container's shape: the
// golden JPEG with its XMP packet declaring the motion directory, and
// the video appended on the tail (size - length = the window).
func pixelMotionPhoto(t *testing.T, video []byte) []byte {
	t.Helper()
	golden := goldenFixture(t)
	xmp := fmt.Sprintf(`<?xpacket begin="\ufeff" id="W5M0MpCehiHzreSzNTczkc9d"?>
<x:xmpmeta xmlns:x="adobe:ns:meta/">
 <rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#">
  <rdf:Description rdf:about=""
    xmlns:GCamera="http://ns.google.com/photos/1.0/camera/"
    xmlns:Container="http://ns.google.com/photos/1.0/container/"
    xmlns:Item="http://ns.google.com/photos/1.0/container/item/"
    GCamera:MotionPhoto="1" GCamera:MotionPhotoVersion="1"
    GCamera:MotionPhotoPresentationTimestampUs="0">
   <GCamera:ContainerDirectory>
    <rdf:Seq>
     <rdf:li rdf:parseType="Resource">
      <Container:Directory>
       <rdf:Seq>
        <rdf:li rdf:parseType="Resource">
         <Container:Item Item:Mime="image/jpeg" Item:Semantic="Primary" Item:Length="0" Item:Padding="0"/>
        </rdf:li>
       </rdf:Seq>
      </Container:Directory>
     </rdf:li>
     <rdf:li rdf:parseType="Resource">
      <Container:Directory>
       <rdf:Seq>
        <rdf:li rdf:parseType="Resource">
         <Container:Item Item:Mime="video/mp4" Item:Semantic="MotionPhoto" Item:Length="%d" Item:Padding="0"/>
        </rdf:li>
       </rdf:Seq>
      </Container:Directory>
     </rdf:li>
    </rdf:Seq>
   </GCamera:ContainerDirectory>
  </rdf:Description>
 </rdf:RDF>
</x:xmpmeta>
<?xpacket end="w"?>`, len(video))

	header := []byte("http://ns.adobe.com/xap/1.0/\x00")
	payload := append(append([]byte{}, header...), []byte(xmp)...)
	segment := []byte{0xff, 0xe1, byte((len(payload) + 2) >> 8), byte((len(payload) + 2) & 0xff)}
	if len(golden) < 2 || golden[0] != 0xff || golden[1] != 0xd8 {
		t.Fatal("the golden fixture is not a JPEG")
	}
	out := append([]byte{}, golden[:2]...)
	out = append(out, segment...)
	out = append(out, payload...)
	out = append(out, golden[2:]...)
	out = append(out, video...)
	return out
}

// TestMotionContainerItemHandlesBothDepths keeps the walk's two shapes
// pinned without a camera: the flat list (a HEIC) and the nested
// directory list (a JPEG).
func TestMotionContainerItemHandlesBothDepths(t *testing.T) {
	flat := []any{
		map[string]any{"Item": map[string]any{"Semantic": "Primary", "Length": float64(0)}},
		map[string]any{"Item": map[string]any{"Semantic": "MotionPhoto", "Length": float64(42), "Padding": float64(7)}},
	}
	if length, padding, ok := motionContainerItem(flat); !ok || length != 42 || padding != 7 {
		t.Errorf("flat walk = %d/%d/%v, want 42/7/true", length, padding, ok)
	}
	nested := []any{map[string]any{"Directory": flat}}
	if length, _, ok := motionContainerItem(nested); !ok || length != 42 {
		t.Errorf("nested walk = %d/%v, want 42/true", length, ok)
	}
	if _, _, ok := motionContainerItem([]any{map[string]any{"Item": map[string]any{"Semantic": "Primary"}}}); ok {
		t.Error("a directory without a motion item answered a match")
	}
}

// TestExtractedMotionStaysOutOfTheSearchAndTheTimeline pins the
// visibility rule from the other side: the component row the extraction
// wrote is invisible to the timeline, the search and the counts, even
// though its bytes are real and it sits in the sync album.
func TestExtractedMotionStaysOutOfTheSearchAndTheTimeline(t *testing.T) {
	requireExiftool(t)
	requireFfmpegJob(t)
	e := newEnv(t)

	video := realMotionVideo(t, e.dataDir)
	still := pixelMotionPhoto(t, video)
	// The real upload path: the still joins the owner's sync album (the
	// timeline's source) and its pipeline starts, exactly like a family
	// member's import.
	uploaded := decodeServedAsset(t, e.postUpload(t, map[string]string{
		"deviceAssetId": "device-motion-photo",
		"fileName":      "IMG_7001.jpg",
	}, []uploadFileFixture{{field: "photo", name: "IMG_7001.jpg", content: still}}))
	id := uploaded.Id
	eventually(t, jobWaitTimeout, "the still to become ready", func() bool {
		return e.assetOf(t, id).readyAt.Valid
	})
	eventually(t, jobWaitTimeout, "the still to gain its motion", func() bool {
		var live bool
		if err := e.pool.QueryRow(context.Background(),
			`SELECT live_photo_video_id IS NOT NULL FROM assets WHERE id = $1`, pgUUID(id)).Scan(&live); err != nil {
			return false
		}
		return live
	})

	// The timeline lists the still once and the motion never.
	rec := e.serveGet(t, "/api/v1/assets/timeline?locale=en", nil)
	if rec.Code != 200 {
		t.Fatalf("timeline status = %d (body %q)", rec.Code, rec.Body.String())
	}
	var timeline struct {
		Items []struct {
			ID               string  `json:"id"`
			LivePhotoVideoID *string `json:"livePhotoVideoId"`
		} `json:"items"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &timeline); err != nil {
		t.Fatalf("decode the timeline: %v", err)
	}
	if len(timeline.Items) != 1 {
		t.Fatalf("timeline items = %d, want the still alone", len(timeline.Items))
	}
	if timeline.Items[0].ID != id.String() || timeline.Items[0].LivePhotoVideoID == nil {
		t.Fatalf("timeline item = %+v, want the still carrying its motion id", timeline.Items[0])
	}

	// The search likewise answers one hit, not two.
	rec = e.serveGet(t, "/api/v1/search?query=IMG_7001&locale=en", nil)
	if rec.Code != 200 {
		t.Fatalf("search status = %d (body %q)", rec.Code, rec.Body.String())
	}
	var search struct {
		Items []struct {
			ID string `json:"id"`
		} `json:"items"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &search); err != nil {
		t.Fatalf("decode the search: %v", err)
	}
	if len(search.Items) != 1 {
		t.Errorf("search items = %d, want the still alone", len(search.Items))
	}
}
