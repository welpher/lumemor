// Transcode pipeline tests (M2c Task 6): the video asset's third job
// end to end. Review Focus ⑥ lives here: a video upload counts THREE
// pipeline jobs (metadata, thumbnail, transcode), asset.ready fires
// exactly once when all three completed, and a re-run of the
// transcode job converges without re-firing it. The sweep's video arm
// re-enqueues all three; its image arm keeps two (the image sweep
// tests pin that side).
package jobs

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/riverqueue/river"

	dbgen "github.com/welpher/lumemor/server/internal/db/gen"
	"github.com/welpher/lumemor/server/internal/storage"
)

// requireTranscodeToolchain skips the video pipeline tests when the
// real ffmpeg is not on the host: the transcode job runs it, the
// poster extraction runs it, and readiness waits for both.
func requireTranscodeToolchain(t *testing.T) {
	t.Helper()
	requireExiftool(t)
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skipf("ffmpeg is not installed on this host; run the suite inside the dev container (make docker-test): %v", err)
	}
}

// ffprobeInstalled reports whether this host carries ffprobe. The
// pipeline tests' codec/pixel-format assertions probe the transcoded
// output with it; a host without it (the dev container and CI carry
// ffprobe as part of the ffmpeg package) still runs every pure-Go
// assertion -- the +faststart box walk, the poster file set, the
// ready-once event count -- and only the probes drop out. Tests read
// this once at the top so the skip decision stays scoped to the probe
// assertions and never aborts the pure-Go ones around them.
func ffprobeInstalled() bool {
	_, err := exec.LookPath("ffprobe")
	return err == nil
}

// transcodeProbeStream is one stream of an ffprobe report over the
// transcoded file (the output-validity assertion of the pipeline
// tests, mirroring the media package's transcode tests).
type transcodeProbeStream struct {
	CodecType string `json:"codec_type"`
	CodecName string `json:"codec_name"`
	PixFmt    string `json:"pix_fmt"`
}

type transcodeProbeReport struct {
	Streams []transcodeProbeStream `json:"streams"`
}

// requireTranscodedOutput asserts the pipeline's playable copy at the
// pure-Go layer: the file exists and the container holds the
// +faststart layout -- ftyp first, moov before mdat (the source
// fixture's moov sits BEHIND its mdat, so the walk is discriminating:
// only a +faststart remux passes it). No external tool is involved;
// the codec facts are requireTranscodeCodecs' business.
func requireTranscodedOutput(t *testing.T, assetID uuid.UUID) {
	t.Helper()
	outPath := storage.EncodedPath(assetID.String())
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read transcoded %s: %v", outPath, err)
	}

	// The +faststart box walk: ftyp first, moov before mdat.
	var types []string
	moovBeforeMDat := false
	pos := 0
	for pos+8 <= len(data) {
		size := int(uint32(data[pos])<<24 | uint32(data[pos+1])<<16 | uint32(data[pos+2])<<8 | uint32(data[pos+3]))
		boxType := string(data[pos+4 : pos+8])
		types = append(types, boxType)
		if boxType == "moov" && !transcodeBoxSeen(types[:len(types)-1], "mdat") {
			moovBeforeMDat = true
		}
		if size == 0 {
			break
		}
		if size == 1 {
			if pos+16 > len(data) {
				break
			}
			size = 16
		}
		if size < 8 || pos+size > len(data) {
			break
		}
		pos += size
	}
	if len(types) == 0 || types[0] != "ftyp" {
		t.Errorf("%s starts with boxes %v, want ftyp first", outPath, types)
	}
	if !moovBeforeMDat {
		t.Errorf("%s box order = %v, want moov before mdat (the +faststart layout)", outPath, types)
	}
}

// requireTranscodeCodecs asserts the ffprobe-backed codec facts of the
// transcoded copy: its video stream is H.264 at yuv420p, its audio
// (when the source carries one) is AAC. It hard-requires ffprobe
// (Fatal, never Skip): only tests that already read ffprobeInstalled
// call it, so a missing binary is a test-wiring bug, not a condition
// to silently abort the caller mid-assertion.
func requireTranscodeCodecs(t *testing.T, assetID uuid.UUID) {
	t.Helper()
	outPath := storage.EncodedPath(assetID.String())
	if _, err := exec.LookPath("ffprobe"); err != nil {
		t.Fatalf("requireTranscodeCodecs requires ffprobe; the caller must scope its call on ffprobeInstalled (run the suite inside the dev container): %v", err)
	}
	out, err := exec.Command("ffprobe", "-v", "error", "-show_streams", "-of", "json", outPath).Output()
	if err != nil {
		t.Fatalf("probe %s: %v: %s", outPath, err, out)
	}
	var report transcodeProbeReport
	if err := json.Unmarshal(out, &report); err != nil {
		t.Fatalf("decode probe report of %s: %v", outPath, err)
	}
	var video *transcodeProbeStream
	for i := range report.Streams {
		switch report.Streams[i].CodecType {
		case "video":
			video = &report.Streams[i]
		case "audio":
			if report.Streams[i].CodecName != "aac" {
				t.Errorf("audio codec = %q, want aac", report.Streams[i].CodecName)
			}
		}
	}
	if video == nil {
		t.Fatalf("probe report of %s carries no video stream: %+v", outPath, report.Streams)
	}
	if video.CodecName != "h264" {
		t.Errorf("video codec = %q, want h264", video.CodecName)
	}
	if video.PixFmt != "yuv420p" {
		t.Errorf("video pixel format = %q, want yuv420p", video.PixFmt)
	}
}

// transcodeBoxSeen reports whether the box list carries the type.
func transcodeBoxSeen(types []string, boxType string) bool {
	for _, candidate := range types {
		if candidate == boxType {
			return true
		}
	}
	return false
}

// encodedFiles lists the transcoded directory of an asset (nil when
// the directory does not exist).
func (e *env) encodedFiles(t *testing.T, id uuid.UUID) []string {
	t.Helper()
	dir := filepath.Dir(storage.EncodedPath(id.String()))
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		t.Fatalf("list encoded directory of %s: %v", id, err)
	}
	names := make([]string, len(entries))
	for i, entry := range entries {
		names[i] = entry.Name()
	}
	return names
}

// transcodeJobStates counts the transcode jobs of an asset by state.
func (e *env) transcodeJobStates(t *testing.T, id uuid.UUID) map[string]int {
	t.Helper()
	states := map[string]int{}
	rows, err := e.pool.Query(context.Background(),
		`SELECT state FROM river_job WHERE kind = 'transcode' AND args->>'assetId' = $1`, id.String())
	if err != nil {
		t.Fatalf("query transcode jobs of asset %s: %v", id, err)
	}
	defer rows.Close()
	for rows.Next() {
		var state string
		if err := rows.Scan(&state); err != nil {
			t.Fatalf("scan transcode job state: %v", err)
		}
		states[state]++
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate transcode jobs: %v", err)
	}
	return states
}

// encodedModifiedAt stats the transcoded file's modification time.
func (e *env) encodedModifiedAt(t *testing.T, id uuid.UUID) time.Time {
	t.Helper()
	info, err := os.Stat(storage.EncodedPath(id.String()))
	if err != nil {
		t.Fatalf("stat transcoded file of %s: %v", id, err)
	}
	return info.ModTime()
}

// TestUploadVideoThreeJobsToReady is the mandated three-job flow
// (Review Focus ⑥): a video upload through the HTTP endpoint counts
// pending_jobs = 3 (metadata, thumbnail, transcode), all three jobs
// run, asset.ready fires EXACTLY once -- only when the last of the
// three completed -- and the playable copy the transcode produced is
// valid: the +faststart layout always (a pure-Go box walk), the
// H.264/yuv420p codec facts when this host carries ffprobe. The poster
// the thumbnail job derived still exists alongside it.
func TestUploadVideoThreeJobsToReady(t *testing.T) {
	requireTranscodeToolchain(t)
	// The codec probes need ffprobe; everything else in this test is
	// pure Go (the endpoint assertions, the box walk, the poster file
	// set, the event count). Determined once at the top: a host
	// without ffprobe loses only the probes, never the rest.
	probeCodecs := ffprobeInstalled()
	e := newEnv(t)

	video, err := os.ReadFile(filepath.Join("..", "media", "testdata", "tiny.mp4"))
	if err != nil {
		t.Fatalf("read video fixture: %v", err)
	}
	rec := e.postUpload(t, map[string]string{
		"deviceAssetId":   "device-transcode-1",
		"fileName":        "VID_0001.mp4",
		"deviceCreatedAt": "2020-03-01T12:00:00Z",
	}, []uploadFileFixture{{field: "photo", name: "VID_0001.mp4", content: video}})
	if rec.Code != http.StatusCreated {
		t.Fatalf("video upload status = %d, want %d (body %q)", rec.Code, http.StatusCreated, rec.Body.String())
	}
	asset := decodeServedAsset(t, rec)

	// The uploader counts all three jobs: readiness only follows
	// their combined completion.
	if state := e.assetOf(t, asset.Id); state.pending != 3 {
		t.Errorf("pending_jobs right after the video upload = %d, want 3", state.pending)
	}

	eventually(t, jobWaitTimeout, "the video asset to become ready", func() bool {
		return e.assetOf(t, asset.Id).readyAt.Valid
	})

	// The play endpoint serves the transcoded copy the job produced,
	// with the static-asset validators of a derived media file.
	play := e.serveGet(t, "/api/v1/assets/"+asset.Id.String()+"/play", nil)
	if play.Code != http.StatusOK {
		t.Fatalf("play status = %d, want %d (body %q)", play.Code, http.StatusOK, play.Body.String())
	}
	if got := play.Header().Get("Content-Type"); got != "video/mp4" {
		t.Errorf("play content-type = %q, want video/mp4", got)
	}
	checksum := sha256Hex(video)
	if got := play.Header().Get("ETag"); got != `"`+checksum[:16]+`"` {
		t.Errorf("play etag = %q, want the quoted checksum prefix", got)
	}
	onDisk, err := os.ReadFile(storage.EncodedPath(asset.Id.String()))
	if err != nil {
		t.Fatalf("read transcoded file: %v", err)
	}
	if play.Body.Len() != len(onDisk) {
		t.Errorf("play body = %d bytes, want the %d bytes of the transcoded file", play.Body.Len(), len(onDisk))
	}

	requireTranscodedOutput(t, asset.Id)
	if probeCodecs {
		requireTranscodeCodecs(t, asset.Id)
	} else {
		t.Log("ffprobe is not installed on this host; the codec/pix_fmt probes are skipped (run the suite inside the dev container)")
	}

	// The thumbnail job's poster still exists: three jobs, two
	// derived-file families, one readiness.
	if files := e.thumbFiles(t, asset.Id); len(files) != 1 || files[0] != "poster.jpg" {
		t.Errorf("thumbnail files of the video = %v, want exactly [poster.jpg]", files)
	}

	// Ready fired exactly once, after the last of the three.
	if got := countEvents(e.bus, "asset.ready"); got != 1 {
		t.Errorf("asset.ready events = %d, want exactly 1", got)
	}
	payload := eventPayloadOf(t, e.bus, "asset.ready")
	if payload["assetId"] != asset.Id.String() {
		t.Errorf("asset.ready assetId = %q, want %s", payload["assetId"], asset.Id)
	}
	if payload["kind"] != "video" {
		t.Errorf("asset.ready kind = %q, want video", payload["kind"])
	}
	state := e.assetOf(t, asset.Id)
	if state.pending != 0 {
		t.Errorf("pending_jobs = %d, want 0", state.pending)
	}
}

// TestTranscodeJobRerunIsIdempotent pins the re-run contract: a second
// run of the transcode job on an already-ready asset overwrites the
// SAME output path (still exactly one play.mp4, no residue), the
// guarded readiness decrement is a no-op, ready_at stays unchanged,
// and no second asset.ready is published.
func TestTranscodeJobRerunIsIdempotent(t *testing.T) {
	requireTranscodeToolchain(t)
	e := newEnv(t)

	video, err := os.ReadFile(filepath.Join("..", "media", "testdata", "tiny.mp4"))
	if err != nil {
		t.Fatalf("read video fixture: %v", err)
	}
	deviceTakenAt := time.Date(2020, 3, 1, 12, 0, 0, 0, time.UTC).Truncate(time.Microsecond)
	id := e.insertAssetKind(t, "originals/video/rerun.mp4", video, deviceTakenAt, `{}`,
		dbgen.MediaKindVideo, "video/mp4", 3)

	// The full three-job pipeline, as the uploader enqueues it.
	for _, enqueue := range []func(context.Context, uuid.UUID) error{
		e.queue.EnqueueMetadata,
		e.queue.EnqueueThumbnail,
		e.queue.EnqueueTranscode,
	} {
		if err := enqueue(context.Background(), id); err != nil {
			t.Fatalf("enqueue pipeline job: %v", err)
		}
	}
	eventually(t, jobWaitTimeout, "the video asset to become ready", func() bool {
		return e.assetOf(t, id).readyAt.Valid
	})
	firstReadyAt := e.assetOf(t, id).readyAt
	firstModified := e.encodedModifiedAt(t, id)

	// The manual re-enqueue: the review-focus rerun must be safe.
	if err := e.queue.EnqueueTranscode(context.Background(), id); err != nil {
		t.Fatalf("re-enqueue transcode job: %v", err)
	}
	// The re-run completes when its job row lands in River's
	// completed state; the file is rewritten (not appended).
	eventually(t, jobWaitTimeout, "the transcode re-run to complete", func() bool {
		return e.transcodeJobStates(t, id)["completed"] >= 2
	})
	// The event, if one were wrongly published, follows the commit
	// synchronously; a short settle makes the count deterministic.
	time.Sleep(300 * time.Millisecond)

	if files := e.encodedFiles(t, id); len(files) != 1 || files[0] != "play.mp4" {
		t.Errorf("encoded files after the re-run = %v, want exactly [play.mp4]", files)
	}
	// The re-run really rewrote the output: the modification time
	// moved past the first run's.
	if !e.encodedModifiedAt(t, id).After(firstModified) {
		t.Errorf("play.mp4 modification time did not advance on the re-run, want the overwritten file")
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

// TestRequeueSweepRecoversStuckVideoAsset pins the sweep's video arm
// (the three-site sync): a video row the uploader left stuck at
// pending_jobs = 3 is recovered with ALL THREE jobs re-enqueued --
// exactly once each -- and the drained pipeline fires asset.ready
// exactly once. The image arm keeps its two-job shape (the image sweep
// tests pin it).
func TestRequeueSweepRecoversStuckVideoAsset(t *testing.T) {
	requireTranscodeToolchain(t)
	e := newEnv(t)

	video, err := os.ReadFile(filepath.Join("..", "media", "testdata", "tiny.mp4"))
	if err != nil {
		t.Fatalf("read video fixture: %v", err)
	}
	deviceTakenAt := time.Date(2021, 8, 9, 10, 11, 12, 0, time.UTC).Truncate(time.Microsecond)
	id := e.insertAssetKind(t, "originals/sweep/video.mp4", video, deviceTakenAt,
		`{"deviceCreatedAt":"2021-08-09T10:11:12Z"}`,
		dbgen.MediaKindVideo, "video/mp4", 3)
	if got := e.riverJobTotal(t, id); got != 0 {
		t.Fatalf("river jobs before the sweep = %d, want 0 (the stuck precondition)", got)
	}

	counts, err := RunRequeueSweep(context.Background(), e.pool, e.queue)
	if err != nil {
		t.Fatalf("run requeue sweep: %v", err)
	}
	if counts.Stuck != 1 || counts.Enqueued != 3 {
		t.Errorf("sweep counts = stuck %d enqueued %d, want stuck 1 enqueued 3", counts.Stuck, counts.Enqueued)
	}
	kinds := e.riverJobsOf(t, id)
	if kinds["metadata"] != 1 || kinds["thumbnail"] != 1 || kinds["transcode"] != 1 {
		t.Errorf("river jobs = %v, want exactly one metadata, one thumbnail and one transcode job", kinds)
	}

	eventually(t, jobWaitTimeout, "the swept video asset to become ready", func() bool {
		return e.assetOf(t, id).readyAt.Valid
	})
	state := e.assetOf(t, id)
	if state.pending != 0 {
		t.Errorf("pending_jobs = %d, want 0 after the drained recovery", state.pending)
	}
	if got := countEvents(e.bus, "asset.ready"); got != 1 {
		t.Errorf("asset.ready events = %d, want exactly 1", got)
	}
	payload := eventPayloadOf(t, e.bus, "asset.ready")
	if payload["kind"] != "video" {
		t.Errorf("asset.ready kind = %q, want video", payload["kind"])
	}
}

// TestTranscodeJobNonVideoIsNoop pins the defensive arm: a transcode
// job that reaches a non-video row (no real enqueue path produces
// one -- the upload and the sweep both gate on the kind) completes
// without deriving anything, without decrementing the row's pipeline
// counter, and without publishing readiness.
func TestTranscodeJobNonVideoIsNoop(t *testing.T) {
	requireTranscodeToolchain(t)
	e := newEnv(t)

	deviceTakenAt := time.Date(2020, 3, 1, 12, 0, 0, 0, time.UTC).Truncate(time.Microsecond)
	id := e.insertAssetKind(t, "originals/video/oops.jpg", jpegBytes(t), deviceTakenAt, `{}`,
		dbgen.MediaKindImage, "image/jpeg", 2)

	worker := &TranscodeWorker{
		pool:    e.pool,
		bus:     e.bus,
		dataDir: e.dataDir,
		logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
	job := &river.Job[TranscodeArgs]{Args: TranscodeArgs{AssetID: id}}
	if err := worker.Work(context.Background(), job); err != nil {
		t.Fatalf("Work of a non-video asset = %v, want nil", err)
	}

	state := e.assetOf(t, id)
	if state.pending != 2 {
		t.Errorf("pending_jobs = %d, want the un-decremented 2", state.pending)
	}
	if state.readyAt.Valid {
		t.Error("ready_at is set after the non-video no-op")
	}
	if files := e.encodedFiles(t, id); files != nil {
		t.Errorf("encoded files of the image asset = %v, want none", files)
	}
	if got := countEvents(e.bus, "asset.ready"); got != 0 {
		t.Errorf("asset.ready events = %d, want 0", got)
	}
}

// TestTranscodeJobTrashedPairComponentCompletes pins the trashed-pair
// semantics of the component's transcode job. The verified behavior
// it matches: the metadata and thumbnail jobs hold NO deleted_at
// check -- an in-flight job whose row was soft-deleted mid-queue runs
// to completion, because the soft delete deliberately leaves pipeline
// state alone (a trashed row finishing its pipeline is harmless; its
// readiness never surfaces in any live view). The transcode job rides
// the same discipline: over a trashed pair it still derives the
// playable copy and completes the component's counter -- but, being a
// component, it never announces. (The gone-row no-op above stays the
// only early exit: an EXPIRED row answers no row at all.)
func TestTranscodeJobTrashedPairComponentCompletes(t *testing.T) {
	requireTranscodeToolchain(t)
	e := newEnv(t)

	video, err := os.ReadFile(filepath.Join("..", "media", "testdata", "tiny.mp4"))
	if err != nil {
		t.Fatalf("read video fixture: %v", err)
	}
	deviceTakenAt := time.Date(2021, 8, 9, 10, 11, 12, 0, time.UTC).Truncate(time.Microsecond)
	componentID := e.insertAssetKind(t, "originals/video/trashed-pair.mov", video, deviceTakenAt,
		`{}`, dbgen.MediaKindVideo, "video/quicktime", 1)
	photoID := e.insertAssetKind(t, "originals/video/trashed-pair.jpg", jpegBytes(t), deviceTakenAt,
		`{}`, dbgen.MediaKindImage, "image/jpeg", 2)
	// The pair's shape: the photo links to its component, the
	// component carries the flag -- then the pair is trashed, both
	// rows together (the pair-trash semantics of the trash handler,
	// stamped directly here to keep the worker drive deterministic).
	if _, err := e.pool.Exec(context.Background(), `
		UPDATE assets SET live_photo_video_id = $1 WHERE id = $2`, pgUUID(componentID), pgUUID(photoID)); err != nil {
		t.Fatalf("link the pair: %v", err)
	}
	if _, err := e.pool.Exec(context.Background(), `
		UPDATE assets SET is_component = true WHERE id = $1`, pgUUID(componentID)); err != nil {
		t.Fatalf("flag the component: %v", err)
	}
	if _, err := e.pool.Exec(context.Background(), `
		UPDATE assets SET deleted_at = now() WHERE id IN ($1, $2)`,
		pgUUID(componentID), pgUUID(photoID)); err != nil {
		t.Fatalf("trash the pair: %v", err)
	}

	// The worker is invoked directly, mirroring a job that was
	// already in the queue when the pair was trashed.
	worker := &TranscodeWorker{
		pool:    e.pool,
		bus:     e.bus,
		dataDir: e.dataDir,
		logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
	job := &river.Job[TranscodeArgs]{Args: TranscodeArgs{AssetID: componentID}}
	if err := worker.Work(context.Background(), job); err != nil {
		t.Fatalf("Work of a trashed pair's component = %v, want nil", err)
	}

	// The derivation ran (no deleted_at no-op): the playable copy
	// exists with the +faststart layout, the counter drained, and
	// ready_at is stamped -- the trashed row's completion is inert to
	// every live view, exactly like the metadata and thumbnail jobs.
	requireTranscodedOutput(t, componentID)
	state := e.assetOf(t, componentID)
	if state.pending != 0 {
		t.Errorf("component pending_jobs = %d, want 0 after the completion", state.pending)
	}
	if !state.readyAt.Valid {
		t.Error("component ready_at is not stamped after the completion, want the drained counter")
	}

	// The component never announces -- trashed or not.
	time.Sleep(300 * time.Millisecond)
	if got := countEvents(e.bus, "asset.ready"); got != 0 {
		t.Errorf("asset.ready events = %d, want 0 (a component announces nothing)", got)
	}
}
