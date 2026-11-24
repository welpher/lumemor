// Package metawrite writes a capture-time edit into an original file's
// embedded metadata: EXIF for images, the container's creation_time for
// videos. The write is staged beside the original and committed with an
// atomic rename, so a failure leaves the original byte-identical.
package metawrite

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

// ErrUnsupported reports a format with no standard capture-time field
// the app can write (PNG and the like).
var ErrUnsupported = errors.New("metawrite: the format carries no standard capture-time field")

const (
	// imageTimeout bounds one exiftool write; the read path pins the
	// same 30s budget.
	imageTimeout = 30 * time.Second
	// videoTimeout bounds one remux: a 2 GiB video on a slow disk is
	// legitimate, so the budget is generous, the bound still finite.
	videoTimeout = 10 * time.Minute
)

// Supported reports whether the format carries a capture-time field the
// app can write.
func Supported(mime string) bool {
	switch mime {
	case "image/jpeg", "image/heic", "image/heif", "image/tiff",
		"video/mp4", "video/quicktime":
		return true
	}
	return false
}

// Staged is one prepared write: Path holds the modified bytes beside
// the original, Target is the original it commits over.
type Staged struct {
	Path   string
	Target string
}

// Stage writes the modified copy beside the original. The caller
// decides when to commit (after the database write is ready) or to
// discard the staging file.
func Stage(ctx context.Context, target, mime string, at time.Time) (*Staged, error) {
	if !Supported(mime) {
		return nil, ErrUnsupported
	}
	info, err := os.Stat(target)
	if err != nil {
		return nil, fmt.Errorf("stat the original: %w", err)
	}
	f, err := os.CreateTemp(filepath.Dir(target), ".lumemor-meta-*")
	if err != nil {
		return nil, fmt.Errorf("create the staging file: %w", err)
	}
	stagedPath := f.Name()
	if err := f.Close(); err != nil {
		_ = os.Remove(stagedPath)
		return nil, err
	}
	// exiftool's -o wants a fresh output name; ffmpeg writes the file
	// itself. Neither needs the placeholder.
	if err := os.Remove(stagedPath); err != nil {
		return nil, err
	}

	switch {
	case strings.HasPrefix(mime, "image/"):
		err = run(ctx, imageTimeout, "exiftool",
			"-o", stagedPath,
			"-DateTimeOriginal="+at.Format("2006:01:02 15:04:05"),
			"-OffsetTimeOriginal="+at.Format("-07:00"),
			"-charset", "filename=utf8",
			target,
		)
	case strings.HasPrefix(mime, "video/"):
		// The container field is UTC; the display zone stays the
		// library's own column.
		err = run(ctx, videoTimeout, "ffmpeg",
			"-nostdin", "-y",
			"-i", target,
			"-map", "0", "-c", "copy",
			"-metadata", "creation_time="+at.UTC().Format("2006-01-02T15:04:05.000000Z"),
			stagedPath,
		)
	}
	if err != nil {
		_ = os.Remove(stagedPath)
		return nil, err
	}
	// Keep the original's mode: a 0600 staging temp must not silently
	// narrow a 0644 source.
	if err := os.Chmod(stagedPath, info.Mode().Perm()); err != nil {
		_ = os.Remove(stagedPath)
		return nil, fmt.Errorf("restore the original mode: %w", err)
	}
	return &Staged{Path: stagedPath, Target: target}, nil
}

// Fingerprint answers the staged file's content digest, size and
// modification time -- the row's new filesystem facts.
func (s *Staged) Fingerprint() (sha string, size int64, mtime time.Time, err error) {
	f, err := os.Open(s.Path)
	if err != nil {
		return "", 0, time.Time{}, err
	}
	defer func() { _ = f.Close() }()
	digest := sha256.New()
	written, err := io.Copy(digest, f)
	if err != nil {
		return "", 0, time.Time{}, err
	}
	info, err := f.Stat()
	if err != nil {
		return "", 0, time.Time{}, err
	}
	return hex.EncodeToString(digest.Sum(nil)), written, info.ModTime(), nil
}

// Commit renames the staged bytes over the original.
func (s *Staged) Commit() error {
	if err := os.Rename(s.Path, s.Target); err != nil {
		return fmt.Errorf("replace the original: %w", err)
	}
	return nil
}

// Discard removes the staging file (a no-op after a successful commit).
func (s *Staged) Discard() {
	_ = os.Remove(s.Path)
}

// run executes one fixed-argument invocation under a timeout, in its
// own process group so a timeout kills the whole tree; no shell, no
// user-controlled argument beyond the server-resolved paths.
func run(ctx context.Context, timeout time.Duration, name string, args ...string) error {
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	cmd := exec.CommandContext(runCtx, name, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		if cmd.Process == nil {
			return nil
		}
		return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}
	cmd.WaitDelay = 5 * time.Second
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if runCtx.Err() != nil {
			return fmt.Errorf("%s exceeded its %s budget: %w", name, timeout, runCtx.Err())
		}
		return fmt.Errorf("%s failed: %w: %s", name, err, strings.TrimSpace(stderr.String()))
	}
	return nil
}
