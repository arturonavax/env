package security

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"
	"time"
)

type contextKey string

const outputHandlerKey contextKey = "anx.security.output_handler"

// OutputHandler is a callback that receives streaming lines emitted by child processes.
type OutputHandler func(line string)

// ContextWithOutputHandler returns a new context with the given OutputHandler attached.
func ContextWithOutputHandler(ctx context.Context, h OutputHandler) context.Context {
	if h == nil {
		return ctx
	}
	return context.WithValue(ctx, outputHandlerKey, h)
}

// OutputHandlerFromContext retrieves the OutputHandler from ctx, or nil if none is set.
func OutputHandlerFromContext(ctx context.Context) OutputHandler {
	if ctx == nil {
		return nil
	}
	h, _ := ctx.Value(outputHandlerKey).(OutputHandler)
	return h
}

type lineStreamer struct {
	mu      sync.Mutex
	buf     bytes.Buffer
	handler OutputHandler
}

func newLineStreamer(h OutputHandler) *lineStreamer {
	return &lineStreamer{handler: h}
}

func (s *lineStreamer) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, b := range p {
		if b == '\n' || b == '\r' {
			line := strings.TrimSpace(s.buf.String())
			s.buf.Reset()
			if line != "" {
				s.handler(line)
			}
		} else {
			s.buf.WriteByte(b)
		}
	}
	return len(p), nil
}

func (s *lineStreamer) Flush() {
	s.mu.Lock()
	defer s.mu.Unlock()
	line := strings.TrimSpace(s.buf.String())
	s.buf.Reset()
	if line != "" {
		s.handler(line)
	}
}

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
	Stdout  io.Writer
	Stderr  io.Writer
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
	} else {
		cmd.Stdin = strings.NewReader("")
	}
	setProcessAttributes(cmd, binary)

	outBuf := newCappedBuffer(r.maxOutputBytes)
	handler := OutputHandlerFromContext(ctx)
	var streamer *lineStreamer
	if handler != nil {
		streamer = newLineStreamer(handler)
	}

	outWriters := []io.Writer{outBuf}
	if streamer != nil {
		outWriters = append(outWriters, streamer)
	}
	if opts.Stdout != nil {
		outWriters = append(outWriters, opts.Stdout)
	}
	cmd.Stdout = io.MultiWriter(outWriters...)

	errWriters := []io.Writer{outBuf}
	if streamer != nil {
		errWriters = append(errWriters, streamer)
	}
	if opts.Stderr != nil {
		errWriters = append(errWriters, opts.Stderr)
	}
	cmd.Stderr = io.MultiWriter(errWriters...)

	err := cmd.Run()
	if streamer != nil {
		streamer.Flush()
	}
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
