//go:build linux

package jobs

import (
	"context"
	"os/exec"
	"strconv"
	"syscall"
)

// exiftoolMemLimitBytes is the Linux address-space cap of the exiftool
// child (RLIMIT_AS applied through prlimit below). 1 GiB: Perl maps
// generous virtual memory, so a tighter cap would fail legitimate
// extractions, while anything runaway dies at the limit long before it
// threatens the server.
const exiftoolMemLimitBytes = 1 << 30

// exiftoolCommand builds the fixed exiftool invocation for Linux.
//
// The address-space cap cannot ride on
// syscall.SysProcAttr: Go deliberately supports no user code between
// fork and exec (anything allocating there is unsafe), and no Rlimit
// field was ever added to the struct. The kernel-enforced equivalent
// is prlimit(1) of util-linux (a Priority: required package on every
// Debian base, so the dependency is honest): it applies RLIMIT_AS and
// then execs the target, taking its place -- the process this command
// manages is exiftool itself. The template stays a fixed argv with no
// shell and no user-controlled argument beyond the one guarded file
// path.
func exiftoolCommand(ctx context.Context, path string) *exec.Cmd {
	argv := append([]string{"--as=" + strconv.FormatUint(exiftoolMemLimitBytes, 10), exiftoolBinary}, exiftoolArgs...)
	argv = append(argv, path)
	return exec.CommandContext(ctx, "prlimit", argv...)
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
