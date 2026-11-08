//go:build darwin

package media

import (
	"context"
	"os/exec"
	"syscall"
)

// ffmpegCommand builds the fixed ffmpeg poster invocation for macOS:
// the plain template with no wrapper.
//
// Degradation (documented per the security spec): macOS offers no
// mechanism to apply an address-space cap to a child before exec --
// no SysProcAttr field, and no prlimit(1) equivalent -- so the memory
// limit of the spec's subprocess discipline cannot be applied here.
// The remaining discipline is intact: the fixed argument template,
// the context timeout, and the process-group kill. Development and CI
// also run the suite inside the Linux dev container, where the cap is
// kernel-enforced.
func ffmpegCommand(ctx context.Context, inputPath, outPath string) *exec.Cmd {
	return exec.CommandContext(ctx, ffmpegBinary, posterArgs(inputPath, outPath)...)
}

// transcodeCommand builds the fixed ffmpeg transcode invocation for
// macOS: the plain template with no wrapper, under the same
// degradation documented on ffmpegCommand (no address-space cap
// before exec on this platform). The argument template itself is the
// shared one (transcode.go); maxHeight is the settings preset's
// bound.
func transcodeCommand(ctx context.Context, inputPath, outPath string, maxHeight int) *exec.Cmd {
	return exec.CommandContext(ctx, transcodeBinary, transcodeArgs(inputPath, outPath, maxHeight)...)
}

// posterArgs is the fixed argument template beyond the binary: seek to
// the first frame, take exactly one frame, scale it into the tile box,
// write the JPEG. Nothing here is user-controlled; the two paths are
// server-generated (the input is guarded by the job's asset path
// resolution, the output derived from the asset id).
func posterArgs(inputPath, outPath string) []string {
	return []string{
		"-ss", "0",
		"-i", inputPath,
		"-frames:v", "1",
		"-vf", posterScaleFilter,
		"-q:v", "5",
		outPath,
	}
}

// procAttr puts the child in its own process group so a timeout can
// kill it together with anything it spawned.
func procAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{Setpgid: true}
}

// killProcessGroup signals the whole process group of pid. With
// Setpgid set, the child is its own group leader, so -pid addresses
// the child and every descendant it spawned.
func killProcessGroup(pid int) error {
	return syscall.Kill(-pid, syscall.SIGKILL)
}
