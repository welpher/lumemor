//go:build darwin

package metawrite

// wrapCommand is the identity on macOS: no prlimit(1) equivalent exists
// to cap a child's address space before exec. The remaining discipline
// (fixed argv, timeout, process-group kill) stays intact; development
// and CI run the suite in the Linux dev container, where the cap is
// kernel-enforced.
func wrapCommand(binary string, args []string) (string, []string) {
	return binary, args
}
