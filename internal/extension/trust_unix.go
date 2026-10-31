//go:build unix

package extension

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"
)

// configureProcess puts the child in its own process group and makes cancellation kill the whole
// group, so grandchildren cannot outlive Close.
func configureProcess(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error { return killProcessTree(cmd) }
	cmd.WaitDelay = 2 * time.Second
}

func killProcessTree(cmd *exec.Cmd) error {
	if cmd.Process == nil {
		return nil
	}
	// A negative pid signals the whole process group created by Setpgid.
	if err := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL); err != nil {
		return cmd.Process.Kill()
	}
	return nil
}

// checkFileTrust enforces ownership and write-permission invariants on the binary and its directory.
func checkFileTrust(path string) error {
	if err := checkOwnerAndMode(path, false); err != nil {
		return err
	}
	return checkOwnerAndMode(filepath.Dir(path), true)
}

func checkOwnerAndMode(path string, isDir bool) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrUntrustedBinary, err)
	}
	st, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return fmt.Errorf("%w: cannot read ownership of %q", ErrUntrustedBinary, path)
	}

	uid := uint32(os.Geteuid())
	if st.Uid != uid && st.Uid != 0 {
		return fmt.Errorf("%w: %q is owned by uid %d, want %d or root", ErrUntrustedBinary, path, st.Uid, uid)
	}

	const groupOrWorldWritable = 0o022
	if mode := info.Mode().Perm(); mode&groupOrWorldWritable != 0 {
		kind := "file"
		if isDir {
			kind = "directory"
		}
		return fmt.Errorf("%w: %s %q is group/world-writable (mode %04o)", ErrUntrustedBinary, kind, path, mode)
	}
	return nil
}
