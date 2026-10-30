package media

import (
	"context"
	"errors"
	"fmt"
	"image/jpeg"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/bbrks/go-blurhash"

	"github.com/welpher/lumemor/server/internal/storage"
)

// The ffmpeg poster-frame discipline, mirroring the exiftool subprocess
// discipline of the metadata job (security spec section 3): a FIXED
// argument template, no shell, no user-controlled arguments -- the one
// input path is server-generated and guarded by the job's asset path
// resolution, the one output path is derived from the asset id. Every
// invocation runs under a context timeout, in its own process group so
// a timeout kills the whole group (ffmpeg plus anything it spawned),
// and on Linux under an address-space cap.
//
// ffmpegBinary and ffmpegTimeout are package variables on purpose: they
// are the test seam (a wrapper script as the binary, a shortened
// timeout), exactly like the metadata job's exiftool seam.
var (
	// ffmpegBinary is the ffmpeg executable the poster extraction runs.
	// The PATH lookup covers the dev container, CI, and the runtime
	// image; tests substitute a wrapper script through this variable.
	ffmpegBinary = "ffmpeg"

	// ffmpegTimeout bounds one poster extraction. Video decode is
	// heavier than exiftool's metadata read, so the budget sits at the
	// security spec's 60s subprocess ceiling: generous for a first
	// frame of any realistic family video, small enough that a wedged
	// decoder cannot pin a worker slot for long.
	ffmpegTimeout = 60 * time.Second
)

// posterTileBox is the poster's scale target: the same 250px box the
// image tile fits within. The poster is a JPEG (not an AVIF thumbnail):
// ffmpeg's AVIF encode availability varies across builds, while its
// JPEG encoder is universal, so the discipline pins the JPEG.
const posterTileBox = 250

// posterScaleFilter is the fixed -vf scale expression of the poster
// extraction: fit within the 250px box, aspect preserved, never
// upscaled (force_original_aspect_ratio=decrease).
var posterScaleFilter = fmt.Sprintf("scale=%d:%d:force_original_aspect_ratio=decrease", posterTileBox, posterTileBox)

// GeneratePoster extracts the first frame of the video at originalPath
// into storage/thumbs/<assetID>/poster.jpg and returns the blurhash
// computed from that poster. It runs idempotently and atomically: the
// frame lands in a uniquely named temporary file beside the poster
// and is renamed into place only once it is complete, so a re-run
// converges to the same single file with no duplicates, and a
// concurrent reader never observes a partial poster. A failed
// extraction removes the temporary file and leaves the previous
// poster (if any) untouched.
//
// The ctx bounds the ffmpeg subprocess (timeout, cancellation); the
// blurhash itself decodes the small JPEG in-process.
func GeneratePoster(ctx context.Context, originalPath, assetID string) (string, error) {
	dir, outPath, err := posterTarget(assetID)
	if err != nil {
		return "", err
	}
	// The frame lands in a scratch directory beside the poster: its
	// file cannot pre-exist (ffmpeg's fixed argv carries no -y, so it
	// refuses existing outputs), the rename onto the poster publishes
	// only complete bytes, and the deferred cleanup removes the
	// emptied directory -- or the residue of a failed run.
	scratch, cleanup, err := stageTempDir(dir)
	if err != nil {
		return "", err
	}
	defer cleanup()
	staged := filepath.Join(scratch, "poster.jpg")

	if err := extractPosterFrame(ctx, originalPath, staged); err != nil {
		return "", fmt.Errorf("extract poster of %s: %w", originalPath, err)
	}

	hash, err := posterBlurhash(staged)
	if err != nil {
		return "", err
	}

	if err := os.Rename(staged, outPath); err != nil {
		return "", fmt.Errorf("publish poster of asset %s: %w", assetID, err)
	}
	return hash, nil
}

// extractPosterFrame runs the fixed ffmpeg invocation: input-seek to
// the very first frame (-ss 0 before -i), exactly one video frame
// (-frames:v 1), scaled into the tile box, written as a JPEG at the
// pinned -q:v 5 quality. On timeout the whole process group is killed;
// the error wraps the context's cause. The caller holds no database
// connection while this runs.
func extractPosterFrame(ctx context.Context, inputPath, outPath string) error {
	runCtx, cancel := context.WithTimeout(ctx, ffmpegTimeout)
	defer cancel()

	cmd := ffmpegCommand(runCtx, inputPath, outPath)
	cmd.SysProcAttr = procAttr()
	// CommandContext's default cancellation kills only the process
	// itself; override it to kill the whole process group so nothing
	// ffmpeg spawned survives a timeout.
	cmd.Cancel = func() error {
		if cmd.Process == nil {
			return nil
		}
		return killProcessGroup(cmd.Process.Pid)
	}
	// Backstop: if the group kill fails or a descendant keeps the
	// pipes open past the kill, force the pipes closed and terminate
	// instead of hanging on Wait.
	cmd.WaitDelay = 5 * time.Second

	// Output: ffmpeg writes its progress log to stderr; only a failed
	// run surfaces it (bounded by stderrOf).
	if _, err := cmd.Output(); err != nil {
		if runCtx.Err() != nil {
			return fmt.Errorf("ffmpeg exceeded its %s budget and its process group was killed: %w", ffmpegTimeout, runCtx.Err())
		}
		return fmt.Errorf("run ffmpeg: %w: %s", err, stderrOf(err))
	}
	return nil
}

// posterTarget resolves the poster output directory and path and
// creates the directory.
func posterTarget(assetID string) (dir, outPath string, err error) {
	outPath = storage.PosterPath(assetID)
	dir = filepath.Dir(outPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", "", fmt.Errorf("create poster directory of asset %s: %w", assetID, err)
	}
	return dir, outPath, nil
}

// stderrOf extracts the captured stderr of a failed command, bounded
// to keep the error message readable (ffmpeg's progress log is
// verbose). It mirrors the jobs package's helper for exiftool.
func stderrOf(err error) string {
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) && len(exitErr.Stderr) > 0 {
		const max = 512
		stderr := string(exitErr.Stderr)
		if len(stderr) > max {
			stderr = stderr[:max]
		}
		return stderr
	}
	return ""
}

// posterBlurhash computes the blurhash of the poster JPEG through the
// standard library's decoder: the poster is a 250px-box JPEG, so the
// decode is cheap and the hash directly placeholders the poster.
func posterBlurhash(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("open poster %s: %w", path, err)
	}
	defer func() { _ = file.Close() }()
	poster, err := jpeg.Decode(file)
	if err != nil {
		return "", fmt.Errorf("decode poster %s: %w", path, err)
	}
	hash, err := blurhash.Encode(blurhashXComponents, blurhashYComponents, poster)
	if err != nil {
		return "", fmt.Errorf("encode blurhash: %w", err)
	}
	return hash, nil
}
