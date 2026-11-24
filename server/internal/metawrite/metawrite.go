// Package metawrite writes a capture-time edit into an original file's
// embedded metadata: EXIF for images, the container's creation_time for
// videos. The write is staged beside the original and committed with an
// atomic rename, so a failure leaves the original byte-identical.
//
// The staging file keeps the original's extension: ffmpeg selects its
// muxer from the output name, and an extension-less temp has none.
//
// The rename replaces the inode: the mode is restored and the owner is
// preserved best-effort, but hard links, extended attributes and ACLs of
// the original do not ride along. That is the atomicity trade: the file
// is never half-written, and the product's own tools read EXIF/container
// metadata rather than those side channels.
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

// The tool seams: tests substitute the binaries or shorten the budgets.
var (
	exiftoolBinary = "exiftool"
	ffmpegBinary   = "ffmpeg"
	imageTimeout   = 30 * time.Second
	videoTimeout   = 10 * time.Minute
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
// discard the staging file. It runs no database work.
func Stage(ctx context.Context, target, mime string, at time.Time) (*Staged, error) {
	if !Supported(mime) {
		return nil, ErrUnsupported
	}
	info, err := os.Stat(target)
	if err != nil {
		return nil, fmt.Errorf("stat the original: %w", err)
	}
	// The extension rides the staging name: ffmpeg's muxer is chosen
	// from the output file, and an extension-less temp has none.
	ext := filepath.Ext(target)
	f, err := os.CreateTemp(filepath.Dir(target), ".lumemor-meta-*"+ext)
	if err != nil {
		return nil, fmt.Errorf("create the staging file: %w", err)
	}
	stagedPath := f.Name()
	if err := f.Close(); err != nil {
		_ = os.Remove(stagedPath)
		return nil, err
	}
	// Both tools want to create the output themselves.
	if err := os.Remove(stagedPath); err != nil {
		return nil, err
	}

	switch {
	case strings.HasPrefix(mime, "image/"):
		err = run(ctx, imageTimeout, exiftoolBinary,
			"-o", stagedPath,
			"-DateTimeOriginal="+at.Format("2006:01:02 15:04:05"),
			"-OffsetTimeOriginal="+at.Format("-07:00"),
			"-charset", "filename=utf8",
			target,
		)
	case strings.HasPrefix(mime, "video/"):
		// The container field is UTC; the display zone stays the
		// library's own column. The app itself reads DateTimeOriginal
		// only -- a video's creation_time is for other tools.
		err = run(ctx, videoTimeout, ffmpegBinary,
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
	// Keep the original's mode and (best-effort) its owner: a 0600
	// staging temp must not narrow a 0644 source, and the source's
	// group must not flip to the server's.
	if err := os.Chmod(stagedPath, info.Mode().Perm()); err != nil {
		_ = os.Remove(stagedPath)
		return nil, fmt.Errorf("restore the original mode: %w", err)
	}
	if sys, ok := info.Sys().(*syscall.Stat_t); ok {
		_ = os.Chown(stagedPath, int(sys.Uid), int(sys.Gid))
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
// user-controlled argument beyond the server-resolved paths. On Linux
// the child carries the same address-space cap every metadata
// subprocess in this server does.
func run(ctx context.Context, timeout time.Duration, binary string, args ...string) error {
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	name, argv := wrapCommand(binary, args)
	cmd := exec.CommandContext(runCtx, name, argv...)
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
			return fmt.Errorf("%s exceeded its %s budget: %w", binary, timeout, runCtx.Err())
		}
		return fmt.Errorf("%s failed: %w: %s", binary, err, strings.TrimSpace(stderr.String()))
	}
	return nil
}
