package extension

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"sync"
	"time"
)

// DefaultMaxOutputBytes limits stdout buffers from extensions (1 MB).
const DefaultMaxOutputBytes = 1024 * 1024

// DefaultHandshakeTimeout sets maximum time allowed for plugin handshake.
const DefaultHandshakeTimeout = 5 * time.Second

// ErrProtocolMismatch indicates the plugin reported an incompatible protocol version.
var ErrProtocolMismatch = errors.New("plugin protocol version mismatch")

// ErrPermissionDenied indicates the plugin requested unpermitted permissions under default-deny.
var ErrPermissionDenied = errors.New("plugin permission denied by policy")

// ErrCorruptedResponse indicates invalid JSON received from extension.
var ErrCorruptedResponse = errors.New("corrupted JSON response from extension")

// Client manages an active stdio connection to an external extension.
type Client struct {
	cmd     *exec.Cmd
	cancel  context.CancelFunc
	stdin   io.WriteCloser
	stdout  *bufio.Reader
	mu      sync.Mutex
	nextID  int
	info    InitializeResult
	maxSize int
}

// Launch starts the extension subprocess, establishes the stdio JSON-RPC handshake, and enforces default-deny.
func Launch(ctx context.Context, binaryPath string, allowedPerms []string, maxOutput int) (*Client, error) {
	if maxOutput <= 0 {
		maxOutput = DefaultMaxOutputBytes
	}

	cmdCtx, cancel := context.WithCancel(ctx)
	cmd := exec.CommandContext(cmdCtx, binaryPath)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("creating stdin pipe: %w", err)
	}

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		_ = stdin.Close()
		return nil, fmt.Errorf("creating stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		cancel()
		_ = stdin.Close()
		return nil, fmt.Errorf("starting extension %q: %w", binaryPath, err)
	}

	client := &Client{
		cmd:     cmd,
		cancel:  cancel,
		stdin:   stdin,
		stdout:  bufio.NewReader(io.LimitReader(stdoutPipe, int64(maxOutput))),
		maxSize: maxOutput,
	}

	// Handshake
	hsCtx, hsCancel := context.WithTimeout(cmdCtx, DefaultHandshakeTimeout)
	defer hsCancel()

	var hsResult InitializeResult
	err = client.Call(hsCtx, "initialize", InitializeParams{
		HostVersion:     "1.0.0",
		ProtocolVersion: ProtocolVersion,
		Capabilities:    []string{"stdio", "streaming", "cancellation"},
	}, &hsResult)

	if err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("handshake failed: %w", err)
	}

	// Verify protocol version
	if hsResult.ProtocolVersion != ProtocolVersion {
		_ = client.Close()
		return nil, fmt.Errorf("%w: expected %s, got %s", ErrProtocolMismatch, ProtocolVersion, hsResult.ProtocolVersion)
	}

	// Enforce default-deny permissions
	allowedMap := make(map[string]bool, len(allowedPerms))
	for _, p := range allowedPerms {
		allowedMap[p] = true
	}
	for _, reqPerm := range hsResult.Permissions {
		if !allowedMap[reqPerm] {
			_ = client.Close()
			return nil, fmt.Errorf("%w: requested unpermitted %q", ErrPermissionDenied, reqPerm)
		}
	}

	client.info = hsResult
	return client, nil
}

// Info returns the handshake metadata from the plugin.
func (c *Client) Info() InitializeResult {
	return c.info
}

// Call executes a synchronous JSON-RPC request over stdio.
func (c *Client) Call(ctx context.Context, method string, params any, result any) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.nextID++
	id := c.nextID

	var rawParams json.RawMessage
	if params != nil {
		b, err := json.Marshal(params)
		if err != nil {
			return fmt.Errorf("marshaling params: %w", err)
		}
		rawParams = b
	}

	req := Request{
		JSONRPC: JSONRPCVersion,
		ID:      id,
		Method:  method,
		Params:  rawParams,
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
		line, err := c.stdout.ReadBytes('\n')
		readCh <- readResult{line: line, err: err}
	}()

	select {
	case <-ctx.Done():
		// Terminate subprocess to unblock background ReadBytes and prevent goroutine leak or reader desync
		_ = c.stdin.Close()
		if c.cmd != nil && c.cmd.Process != nil {
			_ = c.cmd.Process.Kill()
		}
		<-readCh
		return ctx.Err()
	case res := <-readCh:
		if res.err != nil {
			return fmt.Errorf("reading response: %w", res.err)
		}

		var resp Response
		if err := json.Unmarshal(res.line, &resp); err != nil {
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
}

// Execute invokes an action in the extension.
func (c *Client) Execute(ctx context.Context, action string, args []string) (*ExecuteResult, error) {
	var res ExecuteResult
	err := c.Call(ctx, "execute", ExecuteParams{
		Action: action,
		Args:   args,
	}, &res)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

// Close gracefully terminates the extension.
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.cancel != nil {
		c.cancel()
	}
	_ = c.stdin.Close()
	if c.cmd != nil && c.cmd.Process != nil {
		_ = c.cmd.Process.Kill()
		_ = c.cmd.Wait()
	}
	return nil
}
