//go:build linux

package media

import (
	"context"
	"os/exec"
	"strconv"
	"syscall"
)

// ffmpegMemLimitBytes is the Linux address-space cap of the ffmpeg
// child (RLIMIT_AS applied through prlimit below). 4 GiB: a video
// decoder maps generous virtual memory (codec init, frame buffers,
// allocator arenas), so the metadata job's 1 GiB exiftool cap would
// fail legitimate decodes; anything runaway still dies at the limit
// long before it threatens the server.
const ffmpegMemLimitBytes = 4 << 30

// ffmpegCommand builds the fixed ffmpeg poster invocation for Linux.
//
// The security spec's address-space cap cannot ride on
// syscall.SysProcAttr: Go deliberately supports no user code between
// fork and exec (anything allocating there is unsafe), and no Rlimit
// field was ever added to the struct. The kernel-enforced equivalent
// is prlimit(1) of util-linux (a Priority: required package on every
// Debian base, so the dependency is honest): it applies RLIMIT_AS and
// then execs the target, taking its place -- the process this command
// manages is ffmpeg itself. The template stays a fixed argv with no
// shell and no user-controlled argument beyond the two guarded paths.
func ffmpegCommand(ctx context.Context, inputPath, outPath string) *exec.Cmd {
	argv := append([]string{
		"--as=" + strconv.FormatUint(ffmpegMemLimitBytes, 10),
		ffmpegBinary,
	}, posterArgs(inputPath, outPath)...)
	return exec.CommandContext(ctx, "prlimit", argv...)
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
