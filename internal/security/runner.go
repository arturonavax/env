package security

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"sync"
	"time"
)

const defaultMaxOutputBytes = 1024 * 1024 // 1 MB default output cap

// RunnerOption configures a Runner instance.
type RunnerOption func(*Runner)

// WithMaxOutputBytes overrides the maximum combined stdout/stderr bytes retained per command.
func WithMaxOutputBytes(maxBytes int) RunnerOption {
	return func(r *Runner) {
		if maxBytes > 0 {
			r.maxOutputBytes = maxBytes
		}
	}
}

// RunOptions defines per-invocation options for Runner.RunWithInput.
type RunOptions struct {
	Dir     string
	Env     []string
	Stdin   io.Reader
	Timeout time.Duration
}

// Runner executes external binaries safely without shell evaluation unless explicitly invoked.
type Runner struct {
	defaultTimeout time.Duration
	maxOutputBytes int
}

// NewRunner creates a new safe process Runner with functional options.
func NewRunner(defaultTimeout time.Duration, opts ...RunnerOption) *Runner {
	if defaultTimeout <= 0 {
		defaultTimeout = 60 * time.Second
	}
	r := &Runner{
		defaultTimeout: defaultTimeout,
		maxOutputBytes: defaultMaxOutputBytes,
	}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

// Run executes a command with arguments and returns combined stdout and stderr.
func (r *Runner) Run(ctx context.Context, binary string, args ...string) (string, error) {
	return r.RunWithInput(ctx, RunOptions{}, binary, args...)
}

// RunWithOptions executes a command with specific working directory and environment variables.
func (r *Runner) RunWithOptions(
	ctx context.Context,
	dir string,
	env []string,
	binary string,
	args ...string,
) (string, error) {
	return r.RunWithInput(
		ctx,
		RunOptions{
			Dir: dir,
			Env: env,
		},
		binary,
		args...,
	)
}

// RunWithInput executes a command with optional stdin, directory, env, and timeout override.
func (r *Runner) RunWithInput(
	ctx context.Context,
	opts RunOptions,
	binary string,
	args ...string,
) (string, error) {
	if binary == "" {
		return "", errors.New("binary cannot be empty")
	}

	timeout := r.defaultTimeout
	if opts.Timeout > 0 {
		timeout = opts.Timeout
	}

	execCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(execCtx, binary, args...)
	if opts.Dir != "" {
		cmd.Dir = opts.Dir
	}
	if len(opts.Env) > 0 {
		cmd.Env = opts.Env
	}
	if opts.Stdin != nil {
		cmd.Stdin = opts.Stdin
	}

	outBuf := newCappedBuffer(r.maxOutputBytes)
	cmd.Stdout = outBuf
	cmd.Stderr = outBuf

	err := cmd.Run()
	output := outBuf.String()

	if errors.Is(execCtx.Err(), context.DeadlineExceeded) {
		return output, fmt.Errorf("command %q timed out after %v", binary, timeout)
	}
	if err != nil {
		return output, fmt.Errorf("command %q failed: %w (output: %s)", binary, err, RedactSensitives(output))
	}

	return output, nil
}

// cappedBuffer is a goroutine-safe io.Writer that retains at most limit bytes and discards excess without failing the child process.
type cappedBuffer struct {
	mu        sync.Mutex
	buf       bytes.Buffer
	limit     int
	truncated bool
}

func newCappedBuffer(limit int) *cappedBuffer {
	if limit <= 0 {
		limit = defaultMaxOutputBytes
	}
	return &cappedBuffer{limit: limit}
}

func (c *cappedBuffer) Write(p []byte) (int, error) {
	total := len(p)
	c.mu.Lock()
	defer c.mu.Unlock()

	remaining := c.limit - c.buf.Len()
	if remaining <= 0 {
		c.truncated = true
		return total, nil
	}
	if len(p) > remaining {
		_, _ = c.buf.Write(p[:remaining])
		c.truncated = true
		return total, nil
	}
	_, _ = c.buf.Write(p)
	return total, nil
}

func (c *cappedBuffer) String() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.truncated {
		return c.buf.String() + "\n...[output truncated]..."
	}
	return c.buf.String()
}
