//go:build windows

package security

import "os/exec"

func setProcessAttributes(cmd *exec.Cmd, binary string) {
	// No Setsid on Windows
}
