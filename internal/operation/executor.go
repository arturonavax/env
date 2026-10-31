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

// ExecutePlan executes all operations in plan in validated DAG topological order.
func (e *GoExecutor) ExecutePlan(ctx context.Context, plan PlanIR) error {
	orderedOps, err := sortOperationsByDAG(plan.Operations)
	if err != nil {
		return fmt.Errorf("validating operation DAG: %w", err)
	}

	for _, op := range orderedOps {
		if err := e.ExecuteOp(ctx, op); err != nil {
			return fmt.Errorf("executing operation %s (%s): %w", op.OpID, op.Kind, err)
		}
	}
	return nil
}

// sortOperationsByDAG validates dependencies and orders operations topologically.
func sortOperationsByDAG(ops []Operation) ([]Operation, error) {
	byID := make(map[string]Operation, len(ops))
	indegree := make(map[string]int, len(ops))
	dependents := make(map[string][]string, len(ops))

	for _, op := range ops {
		if _, exists := byID[op.OpID]; exists {
			return nil, fmt.Errorf("duplicate operation ID %q", op.OpID)
		}
		byID[op.OpID] = op
		indegree[op.OpID] = 0
	}

	for _, op := range ops {
		for _, dep := range op.DependsOn {
			if _, exists := byID[dep]; !exists {
				return nil, fmt.Errorf("operation %q depends on non-existent operation %q", op.OpID, dep)
			}
			indegree[op.OpID]++
			dependents[dep] = append(dependents[dep], op.OpID)
		}
	}

	var queue []string
	for _, op := range ops {
		if indegree[op.OpID] == 0 {
			queue = append(queue, op.OpID)
		}
	}

	ordered := make([]Operation, 0, len(ops))
	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]
		ordered = append(ordered, byID[curr])

		for _, next := range dependents[curr] {
			indegree[next]--
			if indegree[next] == 0 {
				queue = append(queue, next)
			}
		}
	}

	if len(ordered) != len(ops) {
		return nil, fmt.Errorf("dependency cycle detected among plan operations")
	}

	return ordered, nil
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
		cleanPath := filepath.Clean(op.TargetPath)
		if cleanPath == "/" || cleanPath == "." || filepath.VolumeName(cleanPath) == cleanPath {
			return fmt.Errorf("refusing to delete root path %q", cleanPath)
		}
		if err := os.Remove(cleanPath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("removing %s: %w", cleanPath, err)
		}

	case OpCreateSymlinkOwned:
		cleanTarget := filepath.Clean(op.TargetPath)
		if cleanTarget == "/" || cleanTarget == "." || filepath.VolumeName(cleanTarget) == cleanTarget {
			return fmt.Errorf("invalid symlink target path %q", cleanTarget)
		}
		if fi, err := os.Lstat(cleanTarget); err == nil {
			if fi.Mode()&os.ModeSymlink != 0 {
				_ = os.Remove(cleanTarget)
			} else {
				_ = os.Remove(cleanTarget)
			}
		}
		if err := os.MkdirAll(filepath.Dir(cleanTarget), 0755); err != nil {
			return err
		}
		if err := os.Symlink(op.SourcePath, cleanTarget); err != nil {
			return fmt.Errorf("symlink %s -> %s: %w", cleanTarget, op.SourcePath, err)
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
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return fmt.Errorf("syncing %s: %w", tmpName, err)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpName)
		return fmt.Errorf("closing %s: %w", tmpName, err)
	}

	_ = os.Chmod(tmpName, fileMode)

	if err := os.Rename(tmpName, target); err != nil {
		_ = os.Remove(tmpName)
		return err
	}
	return nil
}
