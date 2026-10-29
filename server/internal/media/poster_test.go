package media

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/welpher/lumemor/server/internal/storage"
)

// TestThumbnailPipelineAvailability surfaces the skip message of the
// CGO-less build: local runs without libvips cannot even compile the
// derivation, so a build that did compile -- but stubbed it out --
// says so once, pointing at the containerized suite.
func TestThumbnailPipelineAvailability(t *testing.T) {
	if !ThumbnailsSupported {
		t.Skip("libvips thumbnail pipeline is not built into this binary (CGO disabled); run the suite inside the dev container (make docker-test)")
	}
}

// requireFFmpeg skips the poster tests when ffmpeg is not on the host:
// they are exercised by `make docker-test` and CI, which both carry the
// tool.
func requireFFmpeg(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath(ffmpegBinary); err != nil {
		t.Skipf("ffmpeg is not installed on this host; run the suite inside the dev container (make docker-test): %v", err)
	}
}

// TestGeneratePoster pins the poster discipline: the first frame of a
// real (tiny) MP4 is extracted as a JPEG, scaled into the 250px tile
// box, aspect preserved, and the blurhash of that poster comes back.
func TestGeneratePoster(t *testing.T) {
	requireFFmpeg(t)
	dataDir := newMediaEnv(t)
	assetID := writeOriginal(t, dataDir, "tiny.mp4", fixtureBytes(t, "tiny.mp4"))

	hash, err := GeneratePoster(context.Background(), filepath.Join(dataDir, "originals", "aa", "bb", "tiny.mp4"), assetID)
	if err != nil {
		t.Fatalf("GeneratePoster: %v", err)
	}
	if len(hash) < 6 {
		t.Errorf("blurhash = %q, want a non-empty encoded hash", hash)
	}

	posterPath := storage.PosterPath(assetID)
	if _, err := os.Stat(posterPath); err != nil {
		t.Fatalf("poster %s missing: %v", posterPath, err)
	}
	width, height := posterDims(t, posterPath)
	// The 1600x800 source frame fits into the box as 250x125.
	if width > 250 || height > 250 {
		t.Errorf("poster = %dx%d, want it within the 250px tile box", width, height)
	}
	if width != 250 || height != 125 {
		t.Errorf("poster = %dx%d, want the aspect-preserving 250x125", width, height)
	}
}

// TestGeneratePosterOverwritesSamePath pins the poster idempotency: a
// second run over the same video rewrites the same poster.jpg -- one
// file, no duplicate beside it (ffmpeg's fixed argv carries no -y; the
// previous poster is removed first).
func TestGeneratePosterOverwritesSamePath(t *testing.T) {
	requireFFmpeg(t)
	dataDir := newMediaEnv(t)
	assetID := writeOriginal(t, dataDir, "tiny.mp4", fixtureBytes(t, "tiny.mp4"))
	path := filepath.Join(dataDir, "originals", "aa", "bb", "tiny.mp4")

	if _, err := GeneratePoster(context.Background(), path, assetID); err != nil {
		t.Fatalf("first GeneratePoster: %v", err)
	}
	if _, err := GeneratePoster(context.Background(), path, assetID); err != nil {
		t.Fatalf("second GeneratePoster: %v", err)
	}

	dir := filepath.Dir(storage.PosterPath(assetID))
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("list poster directory: %v", err)
	}
	if len(entries) != 1 || entries[0].Name() != "poster.jpg" {
		names := make([]string, len(entries))
		for i, entry := range entries {
			names[i] = entry.Name()
		}
		t.Fatalf("poster directory = %v, want exactly [poster.jpg]", names)
	}
}

// TestGeneratePosterUndecodableVideo fails cleanly on a file that is
// not a video: ffmpeg's bounded stderr rides along in the error, the
// poster file is not left behind.
func TestGeneratePosterUndecodableVideo(t *testing.T) {
	requireFFmpeg(t)
	dataDir := newMediaEnv(t)
	assetID := writeOriginal(t, dataDir, "not-a-video.mp4", []byte("definitely not an mp4 container"))

	_, err := GeneratePoster(context.Background(), filepath.Join(dataDir, "originals", "aa", "bb", "not-a-video.mp4"), assetID)
	if err == nil {
		t.Fatal("GeneratePoster error = nil, want the undecodable input to fail")
	}
	if _, statErr := os.Stat(storage.PosterPath(assetID)); statErr == nil {
		t.Error("poster exists after a failed extraction, want none")
	}
}

// slowFfmpegScript writes a wrapper script that stands in for the
// ffmpeg binary: it spawns a long sleep, records that child's pid, and
// itself sleeps long enough to blow any test timeout. It returns the
// script path and the pidfile path; the recorded pid is the orphan the
// process-group kill must reap. (The pattern mirrors the metadata
// job's exiftool timeout test.)
func slowFfmpegScript(t *testing.T) (script, pidFile string) {
	t.Helper()
	dir := t.TempDir()
	script = filepath.Join(dir, "slow-ffmpeg")
	pidFile = filepath.Join(dir, "child.pid")
	body := "#!/bin/sh\nsleep 300 &\necho $! > " + pidFile + "\nsleep 300\n"
	if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
		t.Fatalf("write wrapper script: %v", err)
	}
	return script, pidFile
}

// slowFfmpegTestTimeout is the shortened ffmpeg budget of the timeout
// test: well above the ~0.4s a freshly exec'd shell script needs to
// reach its first commands, so the assertion is the kill after the
// budget, not a race against the script even starting.
const slowFfmpegTestTimeout = 2 * time.Second

// TestFfmpegTimeoutKillsProcessGroup pins the subprocess discipline's
// hard bound: a wedged ffmpeg (a wrapper script that ignores everything
// and sleeps) fails the extraction with the context's deadline, and
// the kill takes the whole process group, so the child the wrapper
// spawned does not survive as an orphan.
func TestFfmpegTimeoutKillsProcessGroup(t *testing.T) {
	newMediaEnv(t)
	script, pidFile := slowFfmpegScript(t)

	previousBinary, previousTimeout := ffmpegBinary, ffmpegTimeout
	ffmpegBinary, ffmpegTimeout = script, slowFfmpegTestTimeout
	t.Cleanup(func() { ffmpegBinary, ffmpegTimeout = previousBinary, previousTimeout })

	_, err := GeneratePoster(context.Background(), "fixture.mp4", "99999999-9999-9999-9999-999999999999")
	if err == nil {
		t.Fatal("poster extraction error = nil, want the timeout to fail it")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("poster extraction error = %v, want it to wrap context.DeadlineExceeded", err)
	}

	requireNoOrphan(t, pidFile)
}

// requireNoOrphan polls until the process recorded in pidFile is dead:
// the group kill reaped the wrapper's child. Dead means the /proc entry
// is gone or shows the zombie state -- a zombie runs no code, holds no
// pipes and no memory, and only its exit status delivery is pending.
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
	if dead := processDead(pid); !dead {
		// Poll briefly: the kill is asynchronous.
		deadline = time.Now().Add(5 * time.Second)
		for time.Now().Before(deadline) && !processDead(pid) {
			time.Sleep(50 * time.Millisecond)
		}
		if !processDead(pid) {
			t.Fatalf("wrapper's child process %d survived the group kill", pid)
		}
	}
}

// processDead reports whether pid is gone (or a zombie on Linux, which
// has /proc; on macOS the kill probe is the equivalent).
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

// The posterBlurhash failure path: a missing poster file is a clear
// error, not a panic or an empty hash.
func TestPosterBlurhashMissingFile(t *testing.T) {
	if _, err := posterBlurhash(filepath.Join(t.TempDir(), "absent.jpg")); err == nil {
		t.Fatal("posterBlurhash error = nil, want the missing file to fail")
	} else if !strings.Contains(fmt.Sprint(err), "open poster") {
		t.Errorf("posterBlurhash error = %v, want the open failure wrapped", err)
	}
}
