package metawrite

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// requireBin skips a test on a host without the tool (the CI runner
// installs ffmpeg but not exiftool; the dev container has both).
func requireBin(t *testing.T, name string) {
	t.Helper()
	if _, err := exec.LookPath(name); err != nil {
		t.Skipf("%s is not installed", name)
	}
}

// copyFixture writes one committed fixture under a fresh temp dir
// (the write mutates the file, and the repo's fixture must not).
func copyFixture(t *testing.T, src, name string) string {
	t.Helper()
	data, err := os.ReadFile(src)
	if err != nil {
		t.Fatalf("read %s: %v", src, err)
	}
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
	return path
}

func TestSupported(t *testing.T) {
	cases := map[string]bool{
		"image/jpeg": true, "image/heic": true, "image/tiff": true,
		"video/mp4": true, "video/quicktime": true,
		"image/png": false, "image/webp": false, "application/pdf": false,
	}
	for mime, want := range cases {
		if got := Supported(mime); got != want {
			t.Errorf("Supported(%q) = %v, want %v", mime, got, want)
		}
	}
}

func TestStageRefusesUnsupported(t *testing.T) {
	path := filepath.Join(t.TempDir(), "scan.png")
	if err := os.WriteFile(path, []byte("not a real png"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Stage(context.Background(), path, "image/png", time.Now()); !errors.Is(err, ErrUnsupported) {
		t.Fatalf("Stage(png) error = %v, want ErrUnsupported", err)
	}
}

func TestStageImageWritesExifAndLeavesTheOriginal(t *testing.T) {
	requireBin(t, exiftoolBinary)
	path := copyFixture(t, filepath.Join("..", "jobs", "testdata", "golden.jpg"), "photo.jpg")
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	at := time.Date(2020, 1, 2, 3, 4, 5, 0, time.FixedZone("", 8*3600))
	staged, err := Stage(context.Background(), path, "image/jpeg", at)
	if err != nil {
		t.Fatalf("Stage: %v", err)
	}
	if !strings.HasSuffix(staged.Path, ".jpg") {
		t.Errorf("staged name %q, want the original's extension", staged.Path)
	}
	out, err := exec.Command(exiftoolBinary, "-s3", "-DateTimeOriginal", staged.Path).Output()
	if err != nil {
		staged.Discard()
		t.Fatalf("read back: %v", err)
	}
	if got := strings.TrimSpace(string(out)); got != "2020:01:02 03:04:05" {
		t.Errorf("staged DateTimeOriginal = %q, want 2020:01:02 03:04:05", got)
	}

	// Until the commit, the original is byte-identical.
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(before) {
		t.Error("the original changed before the commit")
	}

	if err := staged.Commit(); err != nil {
		t.Fatalf("Commit: %v", err)
	}
	out, err = exec.Command(exiftoolBinary, "-s3", "-DateTimeOriginal", path).Output()
	if err != nil {
		t.Fatalf("read back the committed file: %v", err)
	}
	if got := strings.TrimSpace(string(out)); got != "2020:01:02 03:04:05" {
		t.Errorf("committed DateTimeOriginal = %q, want 2020:01:02 03:04:05", got)
	}
}

func TestStageVideoWritesCreationTime(t *testing.T) {
	requireBin(t, ffmpegBinary)
	requireBin(t, "ffprobe")
	path := copyFixture(t, filepath.Join("..", "media", "testdata", "tiny.mp4"), "clip.mp4")

	at := time.Date(2020, 1, 2, 3, 4, 5, 0, time.UTC)
	staged, err := Stage(context.Background(), path, "video/mp4", at)
	if err != nil {
		t.Fatalf("Stage(video): %v", err)
	}
	defer staged.Discard()
	if !strings.HasSuffix(staged.Path, ".mp4") {
		t.Errorf("staged name %q, want the original's extension", staged.Path)
	}
	out, err := exec.Command("ffprobe", "-v", "error", "-show_entries", "format_tags=creation_time",
		"-of", "csv=p=0", staged.Path).Output()
	if err != nil {
		t.Fatalf("ffprobe: %v", err)
	}
	if got := strings.TrimSpace(string(out)); !strings.HasPrefix(got, "2020-01-02T03:04:05") {
		t.Errorf("staged creation_time = %q, want 2020-01-02T03:04:05", got)
	}
}
