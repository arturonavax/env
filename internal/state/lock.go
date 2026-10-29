package state

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"syscall"
	"time"
)

const lockFileName = "anx.lock"

// ErrStateLocked indicates another live anx process currently holds the mutation lock.
var ErrStateLocked = errors.New("anx state is locked by another running process")

type lockPayload struct {
	PID        int       `json:"pid"`
	Operation  string    `json:"operation"`
	AcquiredAt time.Time `json:"acquired_at"`
}

// FileLock represents an acquired exclusive mutation lock on the anx state directory.
type FileLock struct {
	path     string
	released bool
}

// AcquireLock creates an exclusive advisory lock file, automatically recovering stale locks from dead PIDs.
func AcquireLock(stateDir, operation string) (*FileLock, error) {
	if err := os.MkdirAll(stateDir, 0o700); err != nil {
		return nil, fmt.Errorf("creating lock directory: %w", err)
	}
	lockPath := filepath.Join(stateDir, lockFileName)

	for attempt := range 2 {
		f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
		if err == nil {
			payload := lockPayload{
				PID:        os.Getpid(),
				Operation:  operation,
				AcquiredAt: time.Now().UTC(),
			}
			encErr := json.NewEncoder(f).Encode(payload)
			closeErr := f.Close()
			if encErr != nil || closeErr != nil {
				_ = os.Remove(lockPath)
				return nil, fmt.Errorf("writing lock metadata: %w", errors.Join(encErr, closeErr))
			}
			return &FileLock{path: lockPath}, nil
		}

		if !errors.Is(err, os.ErrExist) {
			return nil, fmt.Errorf("creating lock file: %w", err)
		}

		// Inspect existing lock for stale PID
		if attempt == 0 && tryCleanStaleLock(lockPath) {
			continue
		}

		existing := readExistingLockSummary(lockPath)
		return nil, fmt.Errorf("%w (%s)", ErrStateLocked, existing)
	}

	return nil, ErrStateLocked
}

// Release removes the advisory lock file.
func (l *FileLock) Release() error {
	if l == nil || l.released {
		return nil
	}
	l.released = true
	if err := os.Remove(l.path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("releasing lock file: %w", err)
	}
	return nil
}

func tryCleanStaleLock(lockPath string) bool {
	raw, err := os.ReadFile(lockPath)
	if err != nil {
		return false
	}
	var payload lockPayload
	if err := json.Unmarshal(raw, &payload); err != nil || payload.PID <= 0 {
		return false
	}
	if isProcessAlive(payload.PID) {
		return false
	}
	return os.Remove(lockPath) == nil
}

func isProcessAlive(pid int) bool {
	if pid == os.Getpid() {
		return true
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	if runtime.GOOS == "windows" {
		// On Windows, FindProcess fails if the PID no longer exists.
		return true
	}
	err = proc.Signal(syscall.Signal(0))
	return err == nil || errors.Is(err, syscall.EPERM)
}

func readExistingLockSummary(lockPath string) string {
	raw, err := os.ReadFile(lockPath)
	if err != nil {
		return lockPath
	}
	var payload lockPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return lockPath
	}
	return fmt.Sprintf("pid=%d operation=%q since=%s", payload.PID, payload.Operation, payload.AcquiredAt.Format(time.RFC3339))
}
