//go:build !unix

package extension

import "os/exec"

// configureProcess keeps the default Cancel (Process.Kill) on platforms without POSIX process groups.
func configureProcess(_ *exec.Cmd) {}

func killProcessTree(cmd *exec.Cmd) error {
	if cmd.Process == nil {
		return nil
	}
	return cmd.Process.Kill()
}

// checkFileTrust is a no-op outside Unix: POSIX ownership/mode bits do not describe Windows ACLs,
// and pretending they do would be a false guarantee. The SHA-256 pin remains the sole admission
// check on these platforms; this limitation is documented in the package doc.
func checkFileTrust(_ string) error { return nil }
