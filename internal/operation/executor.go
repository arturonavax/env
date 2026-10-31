package operation

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
)

// GoExecutor executes typed IR operations natively without shell wrapper processes.
type GoExecutor struct {
	Stdout io.Writer
	Stderr io.Writer
}

// NewGoExecutor creates a default native Go executor.
func NewGoExecutor(stdout, stderr io.Writer) *GoExecutor {
	if stdout == nil {
		stdout = io.Discard
	}
	if stderr == nil {
		stderr = io.Discard
	}
	return &GoExecutor{Stdout: stdout, Stderr: stderr}
}

// ExecutePlan executes all operations in plan in DAG/sequence order.
func (e *GoExecutor) ExecutePlan(ctx context.Context, plan PlanIR) error {
	for _, op := range plan.Operations {
		if err := e.ExecuteOp(ctx, op); err != nil {
			return fmt.Errorf("executing operation %s (%s): %w", op.OpID, op.Kind, err)
		}
	}
	return nil
}

// ExecuteOp validates preconditions, executes the typed operation, and validates postconditions.
func (e *GoExecutor) ExecuteOp(ctx context.Context, op Operation) error {
	// 1. Verify Preconditions
	for _, cond := range op.Preconditions {
		if err := checkCondition(cond); err != nil {
			return fmt.Errorf("precondition failed: %w", err)
		}
	}

	// 2. Execute Operation
	switch op.Kind {
	case OpEnsureDirectory:
		mode := os.FileMode(0755)
		if op.Mode != 0 {
			mode = os.FileMode(op.Mode)
		}
		if err := os.MkdirAll(op.TargetPath, mode); err != nil {
			return fmt.Errorf("mkdir -p %s: %w", op.TargetPath, err)
		}

	case OpWriteAtomic:
		if err := writeAtomic(op.TargetPath, []byte(op.Content), op.Mode); err != nil {
			return fmt.Errorf("write atomic %s: %w", op.TargetPath, err)
		}

	case OpCopyAtomic:
		content, err := os.ReadFile(op.SourcePath)
		if err != nil {
			return fmt.Errorf("reading source %s: %w", op.SourcePath, err)
		}
		if err := writeAtomic(op.TargetPath, content, op.Mode); err != nil {
			return fmt.Errorf("copying %s to %s: %w", op.SourcePath, op.TargetPath, err)
		}

	case OpDeleteOwned:
		if err := os.Remove(op.TargetPath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("removing %s: %w", op.TargetPath, err)
		}

	case OpCreateSymlinkOwned:
		_ = os.Remove(op.TargetPath)
		if err := os.MkdirAll(filepath.Dir(op.TargetPath), 0755); err != nil {
			return err
		}
		if err := os.Symlink(op.SourcePath, op.TargetPath); err != nil {
			return fmt.Errorf("symlink %s -> %s: %w", op.TargetPath, op.SourcePath, err)
		}

	case OpExecArgv, OpPackageInstall, OpPackageRemove, OpPackageUpgrade:
		if len(op.Argv) == 0 {
			return fmt.Errorf("empty argv")
		}
		cmd := exec.CommandContext(ctx, op.Argv[0], op.Argv[1:]...)
		cmd.Stdout = e.Stdout
		cmd.Stderr = e.Stderr
		if op.Cwd != "" {
			cmd.Dir = op.Cwd
		}
		if len(op.Env) > 0 {
			cmd.Env = append(os.Environ(), op.Env...)
		}
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("executing %v: %w", op.Argv, err)
		}

	case OpAnxRuntimeDelegation:
		if len(op.Argv) > 0 {
			cmd := exec.CommandContext(ctx, op.Argv[0], op.Argv[1:]...)
			cmd.Stdout = e.Stdout
			cmd.Stderr = e.Stderr
			return cmd.Run()
		}
	}

	// 3. Verify Postconditions
	for _, cond := range op.Postconditions {
		if err := checkCondition(cond); err != nil {
			return fmt.Errorf("postcondition failed: %w", err)
		}
	}

	return nil
}

func checkCondition(c Condition) error {
	switch c.Type {
	case "file_exists":
		if _, err := os.Stat(c.Target); err != nil {
			return fmt.Errorf("expected file %s to exist: %w", c.Target, err)
		}
	case "file_missing":
		if _, err := os.Stat(c.Target); err == nil {
			return fmt.Errorf("expected file %s to be absent", c.Target)
		}
	case "hash_equals":
		raw, err := os.ReadFile(c.Target)
		if err != nil {
			return fmt.Errorf("reading %s for hash verification: %w", c.Target, err)
		}
		sum := sha256.Sum256(raw)
		hexSum := hex.EncodeToString(sum[:])
		if hexSum != c.Expected {
			return fmt.Errorf("hash mismatch for %s: got %s, expected %s", c.Target, hexSum, c.Expected)
		}
	}
	return nil
}

func writeAtomic(target string, data []byte, mode uint32) error {
	dir := filepath.Dir(target)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	fileMode := os.FileMode(0644)
	if mode != 0 {
		fileMode = os.FileMode(mode)
	}

	tmp, err := os.CreateTemp(dir, "atomic-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return err
	}
	_ = tmp.Sync()
	_ = tmp.Close()

	_ = os.Chmod(tmpName, fileMode)

	if err := os.Rename(tmpName, target); err != nil {
		_ = os.Remove(tmpName)
		return err
	}
	return nil
}
