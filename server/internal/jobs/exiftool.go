package jobs

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"time"
)

// The exiftool subprocess discipline: a
// FIXED argument template, no shell, no user-controlled arguments --
// the one file path is the only variable, and it is a server-generated
// content-addressed path guarded by the job's assetPath. Every
// invocation runs under a context timeout, in its own process group so
// a timeout kills the whole group (exiftool plus anything it
// spawned), and on Linux under an address-space cap.
//
// exiftoolBinary and exiftoolTimeout are package variables on purpose:
// they are the test seam the timeout test pins (a wrapper script as
// the binary, a shortened timeout).
var (
	// exiftoolBinary is the exiftool executable the job runs. The PATH
	// lookup covers the dev container and the runtime image; tests
	// substitute a wrapper script through this variable.
	exiftoolBinary = "exiftool"

	// exiftoolTimeout bounds one extraction. The budget is pinned
	// at 30s: generous for pathological files, small enough that a wedged
	// process cannot stall a worker slot for long.
	exiftoolTimeout = 30 * time.Second
)

// exiftoolArgs is the fixed argument template beyond the binary:
// -json (machine-readable output), -n (numeric tag values, so GPS
// coordinates arrive as numbers instead of degree-minute-second
// strings), -charset filename=utf8 (file names are interpreted as
// UTF-8). Nothing here is user-controlled; the one file path is
// appended by the platform command builders and guarded by the job's
// assetPath.
var exiftoolArgs = []string{"-json", "-n", "-charset", "filename=utf8"}

// extractMetadata runs one exiftool invocation over path with the
// fixed argument template and returns the raw JSON output. On timeout
// the whole process group is killed; the error wraps the context's
// cause. The caller holds no database connection while this runs.
func extractMetadata(ctx context.Context, path string) ([]byte, error) {
	runCtx, cancel := context.WithTimeout(ctx, exiftoolTimeout)
	defer cancel()

	cmd := exiftoolCommand(runCtx, path)
	cmd.SysProcAttr = procAttr()
	// CommandContext's default cancellation kills only the process
	// itself; override it to kill the whole process group so nothing
	// exiftool spawned survives a timeout.
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

	out, err := cmd.Output()
	if runCtx.Err() != nil {
		return nil, fmt.Errorf("exiftool exceeded its %s budget and its process group was killed: %w", exiftoolTimeout, runCtx.Err())
	}
	if err != nil {
		return nil, fmt.Errorf("run exiftool: %w: %s", err, stderrOf(err))
	}
	return out, nil
}

// stderrOf extracts the captured stderr of a failed command, bounded
// to keep the error message readable.
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
