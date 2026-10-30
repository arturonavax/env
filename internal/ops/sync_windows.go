//go:build windows

package ops

func syncFilesystems() {
	// Windows does not expose a global POSIX-style sync(2) system call.
	// Buffer flushes are managed automatically or per-handle by the Windows cache manager.
}
