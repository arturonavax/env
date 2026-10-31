package extension

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"slices"
	"sync"
	"time"
)

// DefaultMaxMessageBytes limits a single protocol message from an extension (1 MB).
const DefaultMaxMessageBytes = 1024 * 1024

// DefaultHandshakeTimeout sets maximum time allowed for plugin handshake.
const DefaultHandshakeTimeout = 5 * time.Second

// hostVersion is advertised to extensions during the handshake.
const hostVersion = "1.0.0"

// ErrProtocolMismatch indicates the plugin reported an incompatible protocol version.
var ErrProtocolMismatch = errors.New("plugin protocol version mismatch")

// ErrPermissionDenied indicates the plugin declared or requested unpermitted permissions under default-deny.
var ErrPermissionDenied = errors.New("plugin permission denied by policy")

// ErrCorruptedResponse indicates invalid JSON received from extension.
var ErrCorruptedResponse = errors.New("corrupted JSON response from extension")

// ErrMessageTooLarge indicates a single extension message exceeded the configured limit.
var ErrMessageTooLarge = errors.New("extension message exceeds size limit")

// inheritedEnv lists the only host environment variables an extension inherits. Everything else
// (tokens, API keys, SSH agent sockets, ...) is withheld by default.
var inheritedEnv = []string{"PATH", "LANG", "LC_ALL", "TMPDIR", "TEMP", "TMP", "SYSTEMROOT"}

// Spec describes an extension to admit and launch.
type Spec struct {
	// Path is the absolute path of the extension binary.
	Path string
	// SHA256 is the lowercase hex digest the binary must match. Required.
	SHA256 string
	// Declared is the permission set the extension manifest claims to need.
	Declared []string
	// Allowed is the permission set the host is willing to grant. Declared must be a subset.
	Allowed []string
	// MaxMessageBytes bounds a single message; DefaultMaxMessageBytes when zero.
	MaxMessageBytes int
	// Env holds additional KEY=VALUE entries passed to the child on top of the scrubbed baseline.
	Env []string
}

// Client manages an active stdio connection to an external extension.
type Client struct {
	cmd        *exec.Cmd
	cancel     context.CancelFunc
	stdin      io.WriteCloser
	stdout     *bufio.Reader
	scratchDir string
	mu         sync.Mutex
	nextID     int
	info       InitializeResult
	maxSize    int
}

// Launch admits spec (see package doc), starts the extension, and performs the stdio JSON-RPC handshake.
// Admission runs entirely before the process is started: an unpinned, tampered, untrusted or
// over-privileged extension never executes.
func Launch(ctx context.Context, spec Spec) (*Client, error) {
	binary, err := admit(spec)
	if err != nil {
		return nil, err
	}

	maxSize := spec.MaxMessageBytes
	if maxSize <= 0 {
		maxSize = DefaultMaxMessageBytes
	}

	scratch, err := os.MkdirTemp("", "anx-ext-*")
	if err != nil {
		return nil, fmt.Errorf("creating extension scratch dir: %w", err)
	}

	cmdCtx, cancel := context.WithCancel(ctx)
	cmd := exec.CommandContext(cmdCtx, binary)
	cmd.Dir = scratch
	cmd.Env = buildEnv(spec.Env)
	configureProcess(cmd)

	client, err := startClient(cmd, cancel, scratch, maxSize)
	if err != nil {
		cancel()
		_ = os.RemoveAll(scratch)
		return nil, err
	}

	if err := client.handshake(cmdCtx, spec.Declared); err != nil {
		_ = client.Close()
		return nil, err
	}
	return client, nil
}

func startClient(cmd *exec.Cmd, cancel context.CancelFunc, scratch string, maxSize int) (*Client, error) {
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("creating stdin pipe: %w", err)
	}
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		_ = stdin.Close()
		return nil, fmt.Errorf("creating stdout pipe: %w", err)
	}
	if err := cmd.Start(); err != nil {
		_ = stdin.Close()
		return nil, fmt.Errorf("starting extension %q: %w", cmd.Path, err)
	}

	return &Client{
		cmd:        cmd,
		cancel:     cancel,
		stdin:      stdin,
		stdout:     bufio.NewReader(stdoutPipe),
		scratchDir: scratch,
		maxSize:    maxSize,
	}, nil
}

// handshake negotiates the protocol and verifies the extension claims nothing beyond what it declared.
func (c *Client) handshake(ctx context.Context, declared []string) error {
	hsCtx, hsCancel := context.WithTimeout(ctx, DefaultHandshakeTimeout)
	defer hsCancel()

	var res InitializeResult
	err := c.Call(hsCtx, "initialize", InitializeParams{
		HostVersion:     hostVersion,
		ProtocolVersion: ProtocolVersion,
		Capabilities:    []string{"stdio", "streaming", "cancellation"},
	}, &res)
	if err != nil {
		return fmt.Errorf("handshake failed: %w", err)
	}

	if res.ProtocolVersion != ProtocolVersion {
		return fmt.Errorf("%w: expected %s, got %s", ErrProtocolMismatch, ProtocolVersion, res.ProtocolVersion)
	}

	// Admission already proved declared ⊆ allowed. A binary that asks for more than it declared
	// is misrepresenting itself, so it is terminated by the caller (Close) and rejected.
	for _, perm := range res.Permissions {
		if !slices.Contains(declared, perm) {
			return fmt.Errorf("%w: requested undeclared %q", ErrPermissionDenied, perm)
		}
	}

	c.info = res
	return nil
}

// buildEnv returns the scrubbed child environment: allowlisted host variables plus explicit extras.
func buildEnv(extra []string) []string {
	env := make([]string, 0, len(inheritedEnv)+len(extra))
	for _, key := range inheritedEnv {
		if v, ok := os.LookupEnv(key); ok {
			env = append(env, key+"="+v)
		}
	}
	return append(env, extra...)
}

// Info returns the handshake metadata from the plugin.
func (c *Client) Info() InitializeResult {
	return c.info
}

// readMessage reads one newline-delimited message, failing as soon as it exceeds limit bytes
// instead of buffering an unbounded line from an untrusted process.
func readMessage(r *bufio.Reader, limit int) ([]byte, error) {
	var msg []byte
	for {
		chunk, isPrefix, err := r.ReadLine()
		if err != nil {
			return nil, err
		}
		if len(msg)+len(chunk) > limit {
			return nil, ErrMessageTooLarge
		}
		msg = append(msg, chunk...)
		if !isPrefix {
			return msg, nil
		}
	}
}

// Call executes a synchronous JSON-RPC request over stdio.
func (c *Client) Call(ctx context.Context, method string, params any, result any) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.nextID++
	req := Request{JSONRPC: JSONRPCVersion, ID: c.nextID, Method: method}
	if params != nil {
		b, err := json.Marshal(params)
		if err != nil {
			return fmt.Errorf("marshaling params: %w", err)
		}
		req.Params = b
	}

	reqBytes, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshaling request: %w", err)
	}
	if _, err := c.stdin.Write(append(reqBytes, '\n')); err != nil {
		return fmt.Errorf("writing to extension stdin: %w", err)
	}

	type readResult struct {
		line []byte
		err  error
	}
	readCh := make(chan readResult, 1)
	go func() {
		line, err := readMessage(c.stdout, c.maxSize)
		readCh <- readResult{line: line, err: err}
	}()

	select {
	case <-ctx.Done():
		// Kill the process tree to unblock the background read and avoid reader desync.
		_ = c.stdin.Close()
		_ = killProcessTree(c.cmd)
		<-readCh
		return ctx.Err()
	case res := <-readCh:
		if errors.Is(res.err, ErrMessageTooLarge) {
			_ = killProcessTree(c.cmd) // a flooding extension is terminated, not merely rejected
			return res.err
		}
		if res.err != nil {
			return fmt.Errorf("reading response: %w", res.err)
		}
		return decodeResponse(res.line, result)
	}
}

func decodeResponse(line []byte, result any) error {
	var resp Response
	if err := json.Unmarshal(line, &resp); err != nil {
		return fmt.Errorf("%w: %v", ErrCorruptedResponse, err)
	}
	if resp.Error != nil {
		return resp.Error
	}
	if result != nil && len(resp.Result) > 0 {
		if err := json.Unmarshal(resp.Result, result); err != nil {
			return fmt.Errorf("unmarshaling result: %w", err)
		}
	}
	return nil
}

// Execute invokes an action in the extension.
func (c *Client) Execute(ctx context.Context, action string, args []string) (*ExecuteResult, error) {
	var res ExecuteResult
	if err := c.Call(ctx, "execute", ExecuteParams{Action: action, Args: args}, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

// Close terminates the extension process tree and removes its scratch directory.
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.cancel != nil {
		c.cancel()
	}
	_ = c.stdin.Close()
	if c.cmd != nil && c.cmd.Process != nil {
		_ = killProcessTree(c.cmd)
		_ = c.cmd.Wait()
	}
	if c.scratchDir != "" {
		_ = os.RemoveAll(c.scratchDir)
	}
	return nil
}
