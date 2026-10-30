package tools

import (
	"context"
	"fmt"
	"io"
	"os/exec"

	"github.com/arturonavax/env/internal/domain"
)

// BuiltinHandler defines the execution signature for tools built directly into anx.
type BuiltinHandler func(ctx context.Context, toolID string, args []string, in io.Reader, out, errOut io.Writer) error

// Runner handles decoupled execution of external executables, scripts, and built-in tool engines.
type Runner struct {
	registry *Registry
	builtins map[string]BuiltinHandler
}

// NewRunner constructs a Runner backed by a tool registry.
func NewRunner(r *Registry) *Runner {
	return &Runner{
		registry: r,
		builtins: make(map[string]BuiltinHandler, 4),
	}
}

// RegisterBuiltin associates a Go built-in handler for a tool ID.
func (r *Runner) RegisterBuiltin(toolID string, h BuiltinHandler) {
	if toolID != "" && h != nil {
		r.builtins[toolID] = h
	}
}

// Execute runs the specified tool, delegating to external executable or built-in engine.
func (r *Runner) Execute(
	ctx context.Context,
	toolID string,
	args []string,
	in io.Reader,
	out, errOut io.Writer,
) error {
	status, err := r.registry.CheckStatus(ctx, toolID)
	if err != nil {
		return err
	}

	if !status.Available {
		return fmt.Errorf("tool %q is not installed or available on this system (use 'anx tools list' to inspect)", toolID)
	}

	// 1. External executable or script preferred (decoupled project)
	if status.IsExternal && status.ExternalPath != "" {
		cmd := exec.CommandContext(ctx, status.ExternalPath, args...)
		cmd.Stdin = in
		cmd.Stdout = out
		cmd.Stderr = errOut
		if err := cmd.Run(); err != nil {
			var exitErr *exec.ExitError
			if errorsAs(err, &exitErr) {
				return domain.NewExitError(exitErr.ExitCode(), err)
			}
			return err
		}
		return nil
	}

	// 2. Built-in engine fallback (compiled into anx)
	if status.HasBuiltin {
		if handler, ok := r.builtins[toolID]; ok {
			return handler(ctx, toolID, args, in, out, errOut)
		}
		return fmt.Errorf("tool %q declared builtin fallback but no handler is registered", toolID)
	}

	return fmt.Errorf("tool %q is missing an executable target", toolID)
}

func errorsAs(err error, target any) bool {
	return execExitAs(err, target)
}

func execExitAs(err error, target any) bool {
	if exitErr, ok := err.(*exec.ExitError); ok {
		if ptr, ok := target.(**exec.ExitError); ok {
			*ptr = exitErr
			return true
		}
	}
	return false
}
