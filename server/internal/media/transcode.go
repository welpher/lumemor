package media

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/welpher/lumemor/server/internal/storage"
)

// The transcode discipline, the playable-copy sibling of the poster
// extraction: the SAME ffmpeg subprocess discipline (security spec
// section 3) -- a FIXED argument template, no shell, no
// user-controlled arguments (the input path is server-generated and
// guarded by the job's asset path resolution, the output path is
// derived from the asset id), a context timeout, the process-group
// kill on timeout, and on Linux an address-space cap through prlimit.
// The two derivations differ only in their argument template, their
// time budget, and their memory cap: a full re-encode is heavier than
// a first-frame decode on both axes.
//
// transcodeBinary and transcodeTimeout are package variables on
// purpose: they are the test seam (a wrapper script as the binary, a
// shortened timeout), exactly like the poster extraction's seam.
var (
	// transcodeBinary is the ffmpeg executable the transcode runs.
	// The PATH lookup covers the dev container, CI, and the runtime
	// image; tests substitute a wrapper script through this variable.
	transcodeBinary = "ffmpeg"

	// transcodeTimeout bounds one transcode. Re-encoding a whole
	// video (decode, scale to yuv420p, x264 encode, AAC encode,
	// remux) is far heavier than the poster's single-frame decode, so
	// the budget doubles the poster's 60s: 120s covers the family
	// videos of a self-hosted instance while staying small enough
	// that a wedged encoder cannot pin a worker slot for long -- a
	// timeout fails the job, and River's retry policy re-runs it.
	transcodeTimeout = 120 * time.Second
)

// GenerateTranscode re-encodes the video at originalPath into the
// web-safe MP4 at storage/encoded/<assetID>/play.mp4: H.264 video at
// yuv420p plus AAC audio (the codec pair every browser plays
// natively), the movie index relocated to the front of the container
// (+faststart, so playback starts while the file downloads). The
// maxHeight argument is the transcoding settings' preset bound in
// pixels of vertical resolution: 0 keeps the source resolution (the
// `auto` preset), a positive bound scales down to it (never up --
// a smaller source keeps its own resolution, the no-upscale rule
// every derivation of this package follows). It runs idempotently
// and atomically, like the poster extraction: the output lands in a
// uniquely named scratch directory beside the play file (its target
// cannot pre-exist -- the fixed argv carries no -y) and is renamed
// into place only once complete, so a re-run converges to the same
// single file with no duplicates, and a concurrent reader never
// observes a partial stream. A failed transcode removes the scratch
// directory and leaves the previous play file (if any) untouched.
//
// The ctx bounds the ffmpeg subprocess (timeout, cancellation); the
// caller holds no database connection while it runs.
func GenerateTranscode(ctx context.Context, originalPath, assetID string, maxHeight int) error {
	dir, outPath, err := transcodeTarget(assetID)
	if err != nil {
		return err
	}
	// The scratch directory discipline of the poster extraction: the
	// staged file cannot pre-exist, the rename publishes only
	// complete bytes, and the deferred cleanup removes the emptied
	// directory -- or the residue of a failed run.
	scratch, cleanup, err := stageTempDir(dir)
	if err != nil {
		return err
	}
	defer cleanup()
	staged := filepath.Join(scratch, "play.mp4")

	if err := encodeWebSafeMP4(ctx, originalPath, staged, maxHeight); err != nil {
		return fmt.Errorf("transcode %s: %w", originalPath, err)
	}

	if err := os.Rename(staged, outPath); err != nil {
		return fmt.Errorf("publish transcode of asset %s: %w", assetID, err)
	}
	return nil
}

// encodeWebSafeMP4 runs the fixed ffmpeg invocation: exactly the
// first video stream and (when the source carries one) the first
// audio stream, re-encoded to the web-safe pair -- libx264 at the
// pinned CRF 23 / preset fast, yuv420p (the pixel format every
// browser's H.264 decoder takes), AAC at 128k -- and muxed with
// +faststart. A positive maxHeight adds the fixed downscale filter
// (a bound, never an upscale: a source already below it keeps its
// own resolution). On timeout the whole process group is killed; the
// error wraps the context's cause. The caller holds no database
// connection while this runs.
func encodeWebSafeMP4(ctx context.Context, inputPath, outPath string, maxHeight int) error {
	runCtx, cancel := context.WithTimeout(ctx, transcodeTimeout)
	defer cancel()

	cmd := transcodeCommand(runCtx, inputPath, outPath, maxHeight)
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
			return fmt.Errorf("ffmpeg exceeded its %s budget and its process group was killed: %w", transcodeTimeout, runCtx.Err())
		}
		return fmt.Errorf("run ffmpeg: %w: %s", err, stderrOf(err))
	}
	return nil
}

// transcodeTarget resolves the play output directory and path and
// creates the directory.
func transcodeTarget(assetID string) (dir, outPath string, err error) {
	outPath = storage.EncodedPath(assetID)
	dir = filepath.Dir(outPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", "", fmt.Errorf("create encoded directory of asset %s: %w", assetID, err)
	}
	return dir, outPath, nil
}

// transcodeArgs is the fixed argument template beyond the binary: the
// first video stream and (optionally, the trailing question mark) the
// first audio stream, re-encoded to the web-safe pair -- libx264 at
// CRF 23 / preset fast, yuv420p, AAC at 128k -- muxed with +faststart
// so the index precedes the media bytes and playback starts while the
// file downloads. A positive maxHeight appends the fixed downscale
// filter of the transcoding settings' preset (a bound, never an
// upscale: a source already below it keeps its own resolution); zero
// is the `auto` preset and keeps the source resolution. Nothing here
// is user-controlled; the two paths are server-generated (the input
// is guarded by the job's asset path resolution, the output derived
// from the asset id), and the bound is the settings enum's own
// integer.
func transcodeArgs(inputPath, outPath string, maxHeight int) []string {
	argv := []string{
		"-i", inputPath,
		"-map", "0:v:0",
		"-map", "0:a:0?",
		"-c:v", "libx264",
		"-preset", "fast",
		"-crf", "23",
		"-pix_fmt", "yuv420p",
		"-c:a", "aac",
		"-b:a", "128k",
		"-movflags", "+faststart",
	}
	if maxHeight > 0 {
		// -2 keeps the aspect ratio and lands on a width the encoder
		// accepts (a multiple of 2 for yuv420p); the bound is the
		// height and the min() prevents upscaling a source already
		// below it (a 320x240 clip stays 320x240 at the 720p bound).
		argv = append(argv, "-vf", fmt.Sprintf("scale=-2:'min(%d,ih)'", maxHeight))
	}
	return append(argv, outPath)
}
