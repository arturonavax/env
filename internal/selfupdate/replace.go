package selfupdate

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// ErrPostInstallVerification indicates the new binary failed to run after being swapped in; the
// previous binary has been restored.
var ErrPostInstallVerification = errors.New("updated binary failed post-install verification; previous version restored")

const selfTestTimeout = 15 * time.Second

// selfTestBinary proves a binary can start on this machine by running `<path> version`.
func selfTestBinary(ctx context.Context, path string) error {
	ctx, cancel := context.WithTimeout(ctx, selfTestTimeout)
	defer cancel()

	out, err := exec.CommandContext(ctx, path, "version").CombinedOutput()
	if err != nil {
		return fmt.Errorf("self-test of %q failed: %w (output: %.200s)", path, err, out)
	}
	return nil
}

// replaceBinary atomically swaps stage into target and rolls back if the new binary does not run.
//
// Sequence (same directory, hence same filesystem, so every rename is atomic):
//
//	target -> target.old   (running executables may be renamed on Unix and Windows)
//	stage  -> target
//	verify(target)         (rollback target.old -> target on failure)
//
// target.old is kept as a manual rollback point and replaced on the next update.
func replaceBinary(
	ctx context.Context,
	stage, target string,
	verify func(context.Context, string) error,
) (backup string, err error) {
	backup = target + ".old"
	_ = os.Remove(backup)

	hadTarget := true
	if err := os.Rename(target, backup); err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return "", explainPermission(fmt.Errorf("moving current executable aside: %w", err), target)
		}
		hadTarget = false
	}

	restore := func() {
		_ = os.Remove(target)
		if hadTarget {
			_ = os.Rename(backup, target)
		}
	}

	if err := os.Rename(stage, target); err != nil {
		restore()
		return "", explainPermission(fmt.Errorf("placing new executable: %w", err), target)
	}

	if err := verify(ctx, target); err != nil {
		restore()
		return "", fmt.Errorf("%w: %v", ErrPostInstallVerification, err)
	}

	if !hadTarget {
		backup = ""
	}
	return backup, nil
}

func explainPermission(err error, target string) error {
	if errors.Is(err, fs.ErrPermission) {
		return fmt.Errorf("%w (cannot write to %s; re-run with sufficient privileges or reinstall under a user-owned prefix)",
			err, filepath.Dir(target))
	}
	return err
}
