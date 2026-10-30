//go:build !windows

package ops

import "syscall"

func syncFilesystems() {
	syscall.Sync()
}
