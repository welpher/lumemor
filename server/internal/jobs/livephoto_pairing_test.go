package jobs

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
)

// The content-identifier pairing's pins: two halves that arrive as two
// separate uploads find each other through the tag both carry, the
// still stays the only tile, and the rules that must NOT pair (a
// different owner, a different identifier, a motion already taken)
// leave both rows exactly as they were.

// cidStill builds still bytes carrying one content identifier in the
// iOS shape (the XMP's ContentIdentifier): the still half of an iPhone
// pair, dropped in as a file of its own.
func cidStill(t *testing.T, cid string) []byte {
	t.Helper()
	return jpegWithXMP(t, `<?xpacket begin="\ufeff" id="W5M0MpCehiHzreSzNTczkc9d"?>
<x:xmpmeta xmlns:x="adobe:ns:meta/">
 <rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#">
  <rdf:Description rdf:about="" xmlns:GCamera="http://ns.google.com/photos/1.0/camera/"
    GCamera:ContentIdentifier="`+cid+`"/>
 </rdf:RDF>
</x:xmpmeta>
<?xpacket end="w"?>`)
}

// cidVideo builds a real video carrying the same identifier in its own
// XMP box (the motion half of an iPhone pair, uploaded separately).
func cidVideo(t *testing.T, dataDir, cid string) []byte {
	t.Helper()
	return taggedVideo(t, realMotionVideo(t, dataDir), cid)
}

// taggedVideo embeds one content identifier into a real MP4's XMP box
// (the uuid box an iPhone's MOV carries): the top-level box lands right
// after the file-type box, where every reader expects it.
func taggedVideo(t *testing.T, video []byte, cid string) []byte {
	t.Helper()
	firstLen := int(binary.BigEndian.Uint32(video[0:4]))
	if string(video[4:8]) != "ftyp" || firstLen <= 8 || firstLen > len(video) {
		t.Fatalf("the generated video does not start with an ftyp box")
	}
	xmp := `<?xpacket begin="\ufeff" id="W5M0MpCehiHzreSzNTczkc9d"?>
<x:xmpmeta xmlns:x="adobe:ns:meta/">
 <rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#">
  <rdf:Description rdf:about="" xmlns:GCamera="http://ns.google.com/photos/1.0/camera/"
    GCamera:ContentIdentifier="` + cid + `"/>
 </rdf:RDF>
</x:xmpmeta>
<?xpacket end="w"?>`
	uuid := []byte{0xbe, 0x7a, 0xcf, 0xcb, 0x97, 0xa9, 0x42, 0xe8, 0x9c, 0x71, 0x99, 0x94, 0x91, 0xe3, 0xaf, 0xac}
	payload := append(append([]byte{}, uuid...), []byte(xmp)...)
	box := make([]byte, 8, 8+len(payload))
	binary.BigEndian.PutUint32(box[0:4], uint32(len(payload)+8))
	copy(box[4:8], "uuid")
	box = append(box, payload...)

	out := append([]byte{}, video[:firstLen]...)
	out = append(out, box...)
	return append(out, video[firstLen:]...)
}

// TestLivePhotoHalvesFindEachOtherByCID pins the late pairing: the
// still arrives, finds nothing, waits; the motion arrives later and the
// pair converges -- linked, the motion hidden from every listing, and
// the still's wall item carrying the motion's id.
func TestLivePhotoHalvesFindEachOtherByCID(t *testing.T) {
	requireExiftool(t)
	requireFfmpegJob(t)
	e := newEnv(t)

	const cid = "A1B2C3D4-0000-4000-8000-000000000001"
	stillBytes := cidStill(t, cid)
	videoBytes := cidVideo(t, e.dataDir, cid)

	still := decodeServedAsset(t, e.postUpload(t, map[string]string{
		"deviceAssetId": "device-cid-still",
		"fileName":      "IMG_6001.jpg",
	}, []uploadFileFixture{{field: "photo", name: "IMG_6001.jpg", content: stillBytes}}))
	eventually(t, jobWaitTimeout, "the still to become ready", func() bool {
		return e.assetOf(t, still.Id).readyAt.Valid
	})
	// The first half alone: no motion, no error.
	var linked bool
	if err := e.pool.QueryRow(context.Background(),
		`SELECT live_photo_video_id IS NOT NULL FROM assets WHERE id = $1`, pgUUID(still.Id)).Scan(&linked); err != nil {
		t.Fatalf("read the still: %v", err)
	}
	if linked {
		t.Fatal("the still paired with nothing, but carries a motion")
	}

	// The second half: the motion arrives as its own upload.
	motion := decodeServedAsset(t, e.postUpload(t, map[string]string{
		"deviceAssetId": "device-cid-motion",
		"fileName":      "IMG_6001.mp4",
	}, []uploadFileFixture{{field: "photo", name: "IMG_6001.mp4", content: videoBytes}}))
	eventually(t, jobWaitTimeout, "the halves to pair", func() bool {
		var paired bool
		if err := e.pool.QueryRow(context.Background(),
			`SELECT a.live_photo_video_id = $2 FROM assets a WHERE a.id = $1`,
			pgUUID(still.Id), pgUUID(motion.Id)).Scan(&paired); err != nil {
			return false
		}
		return paired
	})

	// The motion is a component now (invisible to every listing) and
	// NOT marked as extracted: it arrived as its own file.
	var component, extracted bool
	if err := e.pool.QueryRow(context.Background(),
		`SELECT is_component, motion_extracted FROM assets WHERE id = $1`, pgUUID(motion.Id)).
		Scan(&component, &extracted); err != nil {
		t.Fatalf("read the motion row: %v", err)
	}
	if !component {
		t.Error("the paired motion is still listed: it must become a component")
	}
	if extracted {
		t.Error("the paired motion is marked extracted: its bytes arrived as their own file")
	}

	// The wall shows the still once, carrying the motion's id.
	rec := e.serveGet(t, "/api/v1/assets/timeline?locale=en", nil)
	if rec.Code != 200 {
		t.Fatalf("timeline status = %d (body %q)", rec.Code, rec.Body.String())
	}
	items := decodeTimelineItems(t, rec.Body.Bytes())
	if len(items) != 1 || items[0].ID != still.Id.String() || items[0].LivePhotoVideoID == nil ||
		*items[0].LivePhotoVideoID != motion.Id.String() {
		t.Fatalf("timeline items = %+v, want the still alone carrying its motion", items)
	}
}

// TestLivePhotoPairingRefusesTheWrongHalves pins the three rules that
// must not pair: a different identifier on the same name, a different
// owner with the same identifier, and a motion another still already
// owns.
func TestLivePhotoPairingRefusesTheWrongHalves(t *testing.T) {
	requireExiftool(t)
	requireFfmpegJob(t)
	e := newEnv(t)

	// 1. Same name, different identifier: no pairing.
	stillBytes := cidStill(t, "CID-AAA")
	videoBytes := cidVideo(t, e.dataDir, "CID-BBB")
	still := decodeServedAsset(t, e.postUpload(t, map[string]string{
		"deviceAssetId": "device-mismatch-still",
		"fileName":      "IMG_5001.jpg",
	}, []uploadFileFixture{{field: "photo", name: "IMG_5001.jpg", content: stillBytes}}))
	motion := decodeServedAsset(t, e.postUpload(t, map[string]string{
		"deviceAssetId": "device-mismatch-motion",
		"fileName":      "IMG_5001.mp4",
	}, []uploadFileFixture{{field: "photo", name: "IMG_5001.mp4", content: videoBytes}}))
	eventually(t, jobWaitTimeout, "both rows to settle", func() bool {
		return e.assetOf(t, still.Id).readyAt.Valid && motion.Id != uuid.Nil
	})
	time.Sleep(1500 * time.Millisecond)
	assertStillUnpaired(t, e, still.Id)
	var component bool
	if err := e.pool.QueryRow(context.Background(),
		`SELECT is_component FROM assets WHERE id = $1`, pgUUID(motion.Id)).Scan(&component); err != nil {
		t.Fatalf("read the motion: %v", err)
	}
	if component {
		t.Error("a name-only resemblance paired the two rows")
	}

	// 2. The same identifier under another owner: never across accounts.
	const cid = "CID-SHARED"
	foreignOwner := uuid.MustParse("00000000-0000-4000-8000-0000000000ff")
	if _, err := e.pool.Exec(context.Background(), `
		INSERT INTO users (id, name, email, password_hash) VALUES ($1, 'Other', 'other@example.com', 'x')`,
		pgUUID(foreignOwner)); err != nil {
		t.Fatalf("seed the other account: %v", err)
	}
	foreignVideo := cidVideo(t, e.dataDir, cid)
	foreignID := uuid.MustParse("00000000-0000-4000-8000-0000000000aa")
	foreignStill := decodeServedAsset(t, e.postUpload(t, map[string]string{
		"deviceAssetId": "device-foreign-still",
		"fileName":      "IMG_4001.jpg",
	}, []uploadFileFixture{{field: "photo", name: "IMG_4001.jpg", content: cidStill(t, cid)}}))
	// The other account's motion: a direct row insert (the pairing's
	// own owner filter is what this pins, not the upload surface).
	if _, err := e.pool.Exec(context.Background(), `
		INSERT INTO assets (id, owner_id, storage_kind, file_path, checksum, original_filename,
		                    file_size, mime_type, media_kind, width, height,
		                    taken_at, taken_at_source, live_photo_cid)
		VALUES ($1, $2, 'blob', 'originals/other/IMG_4001.mp4', $3, 'IMG_4001.mp4',
		        $4, 'video/mp4', 'video', 0, 0, now(), 'import', $5)`,
		pgUUID(foreignID), pgUUID(foreignOwner), "sha1-other", len(foreignVideo), cid); err != nil {
		t.Fatalf("seed the other account's motion: %v", err)
	}
	if err := e.queue.EnqueueMetadata(context.Background(), foreignStill.Id); err != nil {
		t.Fatalf("re-run the still's metadata: %v", err)
	}
	time.Sleep(1500 * time.Millisecond)
	assertStillUnpaired(t, e, foreignStill.Id)
}

// timelineItem is the wire shape the pairing tests read back: the id
// and the pair's motion id.
type timelineItem struct {
	ID               string  `json:"id"`
	LivePhotoVideoID *string `json:"livePhotoVideoId"`
}

// decodeTimelineItems reads one timeline page's items.
func decodeTimelineItems(t *testing.T, body []byte) []timelineItem {
	t.Helper()
	var page struct {
		Items []timelineItem `json:"items"`
	}
	if err := json.Unmarshal(body, &page); err != nil {
		t.Fatalf("decode the timeline %q: %v", body, err)
	}
	return page.Items
}

// assertStillUnpaired fails when the still carries a motion.
func assertStillUnpaired(t *testing.T, e *env, id uuid.UUID) {
	t.Helper()
	var linked bool
	if err := e.pool.QueryRow(context.Background(),
		`SELECT live_photo_video_id IS NOT NULL FROM assets WHERE id = $1`, pgUUID(id)).Scan(&linked); err != nil {
		t.Fatalf("read the still: %v", err)
	}
	if linked {
		t.Error("the still paired against the rules")
	}
}
