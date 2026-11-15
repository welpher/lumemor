//go:build darwin

package jobs

import (
	"context"
	"os/exec"
	"syscall"
)

// exiftoolCommand builds the fixed exiftool invocation for macOS: the
// plain template with no wrapper.
//
// Degradation (documented): macOS offers no
// mechanism to apply an address-space cap to a child before exec --
// no SysProcAttr field, and no prlimit(1) equivalent -- so the memory
// limit of the subprocess discipline cannot be applied here.
// The remaining discipline is intact: the fixed argument template,
// the context timeout, and the process-group kill. Development and CI
// also run the suite inside the Linux dev container, where the cap is
// kernel-enforced.
func exiftoolCommand(ctx context.Context, path string) *exec.Cmd {
	argv := append([]string{exiftoolBinary}, exiftoolArgs...)
	argv = append(argv, path)
	return exec.CommandContext(ctx, exiftoolBinary, argv...)
}

// exiftoolBinaryCommand builds the fixed -b invocation that streams
// one binary tag's bytes: the same template plus the tag flag (a
// package-internal constant, never a caller's string), no shell.
func exiftoolBinaryCommand(ctx context.Context, path, tag string) *exec.Cmd {
	argv := append([]string{exiftoolBinary, "-b", "-" + tag, "-charset", "filename=utf8"}, path)
	return exec.CommandContext(ctx, exiftoolBinary, argv...)
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
