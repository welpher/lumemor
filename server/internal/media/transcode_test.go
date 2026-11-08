// Transcode tests: the ffmpeg discipline of the playable-copy
// derivation. The output validity assertions are ffprobe-based (the
// dev container and CI carry ffprobe as part of the ffmpeg package),
// plus a byte-level MP4 box walk that pins the load-bearing
// +faststart layout -- the source fixture's moov sits BEHIND its mdat
// (a default ffmpeg mux), so the assertion is discriminating: only a
// +faststart remux passes it.
package media

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/welpher/lumemor/server/internal/storage"
)

// requireTranscodeFFmpeg skips the transcode tests when the real
// ffmpeg is not on the host: they are exercised by `make docker-test`
// and CI, the dev container carrying the tool (with its libx264 and
// aac encoders).
func requireTranscodeFFmpeg(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath(transcodeBinary); err != nil {
		t.Skipf("ffmpeg is not installed on this host; run the suite inside the dev container (make docker-test): %v", err)
	}
}

// ffprobeInstalled reports whether this host carries ffprobe. The
// transcode tests' codec/pixel-format/duration assertions probe the
// output with it; a host without it (the dev container and CI carry
// ffprobe as part of the ffmpeg package) still runs every pure-Go
// assertion -- the +faststart box walk, the file-set checks -- and
// only the probes drop out. Tests read this once at the top so the
// skip decision stays scoped to the probe assertions and never
// aborts the pure-Go ones around them.
func ffprobeInstalled() bool {
	_, err := exec.LookPath("ffprobe")
	return err == nil
}

// probeStream is one stream of an ffprobe report.
type probeStream struct {
	CodecType string `json:"codec_type"`
	CodecName string `json:"codec_name"`
	PixFmt    string `json:"pix_fmt"`
	Duration  string `json:"duration"`
	Width     int    `json:"width"`
	Height    int    `json:"height"`
}

// probeReport is the decoded ffprobe output of one media file.
type probeReport struct {
	Streams []probeStream `json:"streams"`
	Format  struct {
		Duration string `json:"duration"`
	} `json:"format"`
}

// probeOf runs ffprobe over the file and decodes its JSON report. It
// hard-requires ffprobe (Fatal, never Skip): only tests that already
// read ffprobeInstalled call it, so a missing binary is a test-wiring
// bug, not a condition to silently abort the caller mid-assertion.
// The fixed invocation carries no user-controlled input beyond the
// path the test itself names.
func probeOf(t *testing.T, path string) probeReport {
	t.Helper()
	if _, err := exec.LookPath("ffprobe"); err != nil {
		t.Fatalf("probeOf requires ffprobe; the caller must scope its call on ffprobeInstalled (run the suite inside the dev container): %v", err)
	}
	out, err := exec.Command("ffprobe", "-v", "error", "-show_streams", "-show_entries", "format=duration", "-of", "json", path).Output()
	if err != nil {
		t.Fatalf("probe %s: %v: %s", path, err, out)
	}
	var report probeReport
	if err := json.Unmarshal(out, &report); err != nil {
		t.Fatalf("decode probe report of %s: %v", path, err)
	}
	return report
}

// probeDurationSeconds parses the probe's duration string.
func probeDurationSeconds(t *testing.T, report probeReport) float64 {
	t.Helper()
	raw := report.Format.Duration
	if raw == "" && len(report.Streams) > 0 {
		raw = report.Streams[0].Duration
	}
	seconds, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
	if err != nil {
		t.Fatalf("parse probe duration %q: %v", raw, err)
	}
	return seconds
}

// probeVideoStream returns the video stream of the report.
func probeVideoStream(t *testing.T, report probeReport) probeStream {
	t.Helper()
	for _, stream := range report.Streams {
		if stream.CodecType == "video" {
			return stream
		}
	}
	t.Fatalf("probe report carries no video stream: %+v", report.Streams)
	return probeStream{}
}

// probeAudioCodec returns the codec name of the audio stream, or ""
// when the file has none.
func probeAudioCodec(report probeReport) string {
	for _, stream := range report.Streams {
		if stream.CodecType == "audio" {
			return stream.CodecName
		}
	}
	return ""
}

// requireFaststartLayout walks the MP4 box structure of the file and
// pins the +faststart layout: the file starts with an ftyp box and the
// moov box (the index a player needs before it can start) appears
// BEFORE the mdat box (the media bytes). The walk handles the plain
// 32-bit box sizes of a healthy file and the to-EOF and 64-bit size
// forms without following them into corruption.
func requireFaststartLayout(t *testing.T, path string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s for the box walk: %v", path, err)
	}
	var types []string
	moovBeforeMDat := false
	pos := 0
	for pos+8 <= len(data) {
		size := int(uint32(data[pos])<<24 | uint32(data[pos+1])<<16 | uint32(data[pos+2])<<8 | uint32(data[pos+3]))
		boxType := string(data[pos+4 : pos+8])
		types = append(types, boxType)
		if boxType == "moov" && !containsString(types[:len(types)-1], "mdat") {
			moovBeforeMDat = true
		}
		if size == 0 {
			// A to-EOF box: nothing can follow it.
			break
		}
		if size == 1 {
			// A 64-bit extended size: the following 8 bytes carry it.
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
		t.Errorf("%s starts with boxes %v, want ftyp first", path, types)
	}
	if !moovBeforeMDat {
		t.Errorf("%s box order = %v, want moov before mdat (the +faststart layout)", path, types)
	}
}

// containsString reports whether the slice carries the value.
func containsString(values []string, value string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}

// TestGenerateTranscode pins the transcode discipline: a real (tiny)
// MP4 is re-encoded into the web-safe pair -- H.264 video at yuv420p,
// the pixel format every browser decodes -- written atomically to
// storage/encoded/<assetID>/play.mp4, the container duration preserved
// within transcode tolerances, and the +faststart layout present (the
// source's moov sits behind its mdat, so this is the remux assertion).
// The codec and duration facts need ffprobe; a host without it still
// runs the output-existence and box-walk assertions.
func TestGenerateTranscode(t *testing.T) {
	requireTranscodeFFmpeg(t)
	// Determined once at the top: a host without ffprobe loses only
	// the probes below, never the assertions around them.
	probeCodecs := ffprobeInstalled()
	dataDir := newMediaEnv(t)
	assetID := writeOriginal(t, dataDir, "tiny.mp4", fixtureBytes(t, "tiny.mp4"))

	if err := GenerateTranscode(context.Background(), filepath.Join(dataDir, "originals", "aa", "bb", "tiny.mp4"), assetID, 0); err != nil {
		t.Fatalf("GenerateTranscode: %v", err)
	}

	outPath := storage.EncodedPath(assetID)
	if _, err := os.Stat(outPath); err != nil {
		t.Fatalf("transcoded %s missing: %v", outPath, err)
	}

	if probeCodecs {
		report := probeOf(t, outPath)
		video := probeVideoStream(t, report)
		if video.CodecName != "h264" {
			t.Errorf("video codec = %q, want h264", video.CodecName)
		}
		if video.PixFmt != "yuv420p" {
			t.Errorf("video pixel format = %q, want yuv420p", video.PixFmt)
		}
		// The fixture is video-only; the transcode keeps exactly that
		// (the optional audio map adds nothing when the source has
		// none).
		if probeAudioCodec(report) != "" {
			t.Errorf("audio codec = %q, want none for the video-only fixture", probeAudioCodec(report))
		}

		// Duration tolerance: the container timing is preserved
		// through the re-encode; the fixture runs 0.3s.
		seconds := probeDurationSeconds(t, report)
		if diff := seconds - 0.3; diff < -0.2 || diff > 0.2 {
			t.Errorf("transcoded duration = %fs, want 0.3s within the 0.2s tolerance", seconds)
		}
	} else {
		t.Log("ffprobe is not installed on this host; the codec/pix_fmt/duration probes are skipped (run the suite inside the dev container)")
	}

	requireFaststartLayout(t, outPath)
}

// TestGenerateTranscodeScalesToPresetBound pins the transcoding
// settings' numeric preset: a positive maxHeight bound downscales the
// re-encode (the aspect ratio kept, the -2 width form's even
// rounding), while the zero bound (the `auto` preset) keeps the
// source resolution. The resolution facts are the whole assertion,
// so the test hard-requires ffprobe.
func TestGenerateTranscodeScalesToPresetBound(t *testing.T) {
	requireTranscodeFFmpeg(t)
	if !ffprobeInstalled() {
		t.Skip("ffprobe is not installed on this host; the resolution probes are the whole assertion (run the suite inside the dev container)")
	}
	dataDir := newMediaEnv(t)
	source := lavfiVideoFixture(t, dataDir, "scaled-video.mp4", 320, 240)

	scaled := "99999999-7777-6666-5555-444444444444"
	if err := GenerateTranscode(context.Background(), source, scaled, 120); err != nil {
		t.Fatalf("GenerateTranscode with the 120p bound: %v", err)
	}
	video := probeVideoStream(t, probeOf(t, storage.EncodedPath(scaled)))
	if video.Height != 120 {
		t.Errorf("scaled height = %d, want the 120 bound", video.Height)
	}
	if video.Width != 160 {
		t.Errorf("scaled width = %d, want the aspect-kept 160", video.Width)
	}

	// The no-upscale arm: a 720p bound on a 320x240 source keeps the
	// source resolution (the min() in the filter is the guard).
	keptUnderBigBound := "99999999-7777-6666-5555-222222222222"
	if err := GenerateTranscode(context.Background(), source, keptUnderBigBound, 720); err != nil {
		t.Fatalf("GenerateTranscode with the 720p bound on a 240px source: %v", err)
	}
	bigVideo := probeVideoStream(t, probeOf(t, storage.EncodedPath(keptUnderBigBound)))
	if bigVideo.Height != 240 {
		t.Errorf("no-upscale height = %d, want the source's 240 (the min guard prevents upscaling)", bigVideo.Height)
	}

	kept := "99999999-7777-6666-5555-333333333333"
	if err := GenerateTranscode(context.Background(), source, kept, 0); err != nil {
		t.Fatalf("GenerateTranscode with the auto bound: %v", err)
	}
	video = probeVideoStream(t, probeOf(t, storage.EncodedPath(kept)))
	if video.Height != 240 || video.Width != 320 {
		t.Errorf("auto resolution = %dx%d, want the source's 320x240", video.Width, video.Height)
	}
}

// TestGenerateTranscodeWithAudio pins the audio arm: a source carrying
// an audio track transcodes into AAC alongside the H.264 video.
func TestGenerateTranscodeWithAudio(t *testing.T) {
	requireTranscodeFFmpeg(t)
	probeCodecs := ffprobeInstalled()
	dataDir := newMediaEnv(t)
	source := audioVideoFixture(t, dataDir)
	assetID := "99999999-8888-7777-6666-555555555555"

	if err := GenerateTranscode(context.Background(), source, assetID, 0); err != nil {
		t.Fatalf("GenerateTranscode: %v", err)
	}

	if probeCodecs {
		report := probeOf(t, storage.EncodedPath(assetID))
		video := probeVideoStream(t, report)
		if video.CodecName != "h264" {
			t.Errorf("video codec = %q, want h264", video.CodecName)
		}
		if got := probeAudioCodec(report); got != "aac" {
			t.Errorf("audio codec = %q, want aac", got)
		}

		// The source runs 1s; the transcode preserves it.
		seconds := probeDurationSeconds(t, report)
		if diff := seconds - 1; diff < -0.25 || diff > 0.25 {
			t.Errorf("transcoded duration = %fs, want 1s within the 0.25s tolerance", seconds)
		}
	} else {
		t.Log("ffprobe is not installed on this host; the codec/duration probes are skipped (run the suite inside the dev container)")
	}
	requireFaststartLayout(t, storage.EncodedPath(assetID))
}

// TestGenerateTranscodeOverwritesSamePath pins the transcode
// idempotency: a second run over the same video converges to the same
// single play.mp4 -- one file, no duplicates and no temporary residue
// beside it (each run stages its output in a unique scratch directory
// and renames it onto the play file).
func TestGenerateTranscodeOverwritesSamePath(t *testing.T) {
	requireTranscodeFFmpeg(t)
	dataDir := newMediaEnv(t)
	assetID := writeOriginal(t, dataDir, "tiny.mp4", fixtureBytes(t, "tiny.mp4"))
	path := filepath.Join(dataDir, "originals", "aa", "bb", "tiny.mp4")

	if err := GenerateTranscode(context.Background(), path, assetID, 0); err != nil {
		t.Fatalf("first GenerateTranscode: %v", err)
	}
	if err := GenerateTranscode(context.Background(), path, assetID, 0); err != nil {
		t.Fatalf("second GenerateTranscode: %v", err)
	}

	dir := filepath.Dir(storage.EncodedPath(assetID))
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("list encoded directory: %v", err)
	}
	if len(entries) != 1 || entries[0].Name() != "play.mp4" {
		names := make([]string, len(entries))
		for i, entry := range entries {
			names[i] = entry.Name()
		}
		t.Fatalf("encoded directory = %v, want exactly [play.mp4]", names)
	}
}

// TestGenerateTranscodeUndecodableVideo fails cleanly on a file that
// is not a video: ffmpeg's bounded stderr rides along in the error,
// no play file is left behind, and neither is any staging residue.
func TestGenerateTranscodeUndecodableVideo(t *testing.T) {
	requireTranscodeFFmpeg(t)
	dataDir := newMediaEnv(t)
	assetID := writeOriginal(t, dataDir, "not-a-video.mp4", []byte("definitely not an mp4 container"))

	err := GenerateTranscode(context.Background(), filepath.Join(dataDir, "originals", "aa", "bb", "not-a-video.mp4"), assetID, 0)
	if err == nil {
		t.Fatal("GenerateTranscode error = nil, want the undecodable input to fail")
	}
	if _, statErr := os.Stat(storage.EncodedPath(assetID)); statErr == nil {
		t.Error("play.mp4 exists after a failed transcode, want none")
	}
	entries, listErr := os.ReadDir(filepath.Dir(storage.EncodedPath(assetID)))
	if listErr != nil {
		t.Fatalf("list encoded directory: %v", listErr)
	}
	if len(entries) != 0 {
		names := make([]string, len(entries))
		for i, entry := range entries {
			names[i] = entry.Name()
		}
		t.Errorf("encoded directory after a failed transcode = %v, want it empty", names)
	}
}

// TestTranscodeTimeoutKillsProcessGroup pins the subprocess
// discipline's hard bound: a wedged ffmpeg (the same wrapper script
// the poster timeout test uses -- it ignores everything and sleeps)
// fails the transcode with the context's deadline, and the kill takes
// the whole process group, so the child the wrapper spawned does not
// survive as an orphan.
func TestTranscodeTimeoutKillsProcessGroup(t *testing.T) {
	newMediaEnv(t)
	script, pidFile := slowFfmpegScript(t)

	previousBinary, previousTimeout := transcodeBinary, transcodeTimeout
	transcodeBinary, transcodeTimeout = script, slowFfmpegTestTimeout
	t.Cleanup(func() { transcodeBinary, transcodeTimeout = previousBinary, previousTimeout })

	err := GenerateTranscode(context.Background(), "fixture.mp4", "99999999-9999-9999-9999-999999999999", 0)
	if err == nil {
		t.Fatal("transcode error = nil, want the timeout to fail it")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("transcode error = %v, want it to wrap context.DeadlineExceeded", err)
	}

	requireNoOrphan(t, pidFile)
}

// audioVideoFixture generates a one-second video WITH an audio track
// through ffmpeg's lavfi sources (the encoders are whatever this
// ffmpeg's mp4 muxer defaults to; the transcode re-encodes anyway).
// Generated at test time because a committed binary fixture would be
// redundant with the tiny.mp4 that pins the video-only arm.
func audioVideoFixture(t *testing.T, dataDir string) string {
	t.Helper()
	path := filepath.Join(dataDir, "audio-video.mp4")
	cmd := exec.Command("ffmpeg",
		"-f", "lavfi", "-i", "testsrc2=size=320x240:rate=25",
		"-f", "lavfi", "-i", "sine=frequency=440:sample_rate=44100",
		"-t", "1", "-pix_fmt", "yuv420p", path)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("generate audio video fixture: %v: %s", err, out)
	}
	return path
}

// lavfiVideoFixture generates a one-second silent video of the exact
// given size through ffmpeg's lavfi test source (the preset test's
// controllable-resolution input).
func lavfiVideoFixture(t *testing.T, dataDir, name string, width, height int) string {
	t.Helper()
	path := filepath.Join(dataDir, name)
	cmd := exec.Command("ffmpeg",
		"-f", "lavfi", "-i", fmt.Sprintf("testsrc2=size=%dx%d:rate=25", width, height),
		"-t", "1", "-pix_fmt", "yuv420p", path)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("generate lavfi video fixture: %v: %s", err, out)
	}
	return path
}
