//go:build !windows

package security

import (
	"os/exec"
	"strings"
	"syscall"
)

func setProcessAttributes(cmd *exec.Cmd, binary string) {
	isSudo := binary == "sudo" || strings.HasSuffix(binary, "/sudo")
	if !isSudo {
		cmd.SysProcAttr = &syscall.SysProcAttr{
			Setsid: true, // Disconnect from controlling terminal (/dev/tty)
		}
		cmd.Cancel = func() error {
			if cmd.Process != nil && cmd.Process.Pid > 0 {
				return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
			}
			return nil
		}
	}
}
