//go:build linux

package metawrite

import "strconv"

// The Linux address-space caps of the metadata children: exiftool's
// Perl runtime maps generously (1 GiB, the jobs package's own pin),
// and the ffmpeg remux copies streams (4 GiB, the poster cap).
const (
	exiftoolMemLimitBytes uint64 = 1 << 30
	ffmpegMemLimitBytes   uint64 = 4 << 30
)

// wrapCommand applies the binary's address-space cap through
// prlimit(1), which sets RLIMIT_AS and then execs the target in its
// place -- the process this command manages is the tool itself. The
// template stays a fixed argv with no shell.
func wrapCommand(binary string, args []string) (string, []string) {
	limit := exiftoolMemLimitBytes
	if binary == ffmpegBinary {
		limit = ffmpegMemLimitBytes
	}
	argv := append([]string{"--as=" + strconv.FormatUint(limit, 10), binary}, args...)
	return "prlimit", argv
}