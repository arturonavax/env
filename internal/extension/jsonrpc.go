package extension

import (
	"encoding/json"
	"fmt"
)

// JSONRPCVersion defines the standard JSON-RPC protocol version.
const JSONRPCVersion = "2.0"

// ProtocolVersion specifies the ANX stdio extension protocol specification version.
const ProtocolVersion = "1.0"

// Request represents a JSON-RPC 2.0 request.
type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// Response represents a JSON-RPC 2.0 response.
type Response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *ErrorObject    `json:"error,omitempty"`
}

// ErrorObject represents a JSON-RPC 2.0 error payload.
type ErrorObject struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

func (e *ErrorObject) Error() string {
	return fmt.Sprintf("JSON-RPC error %d: %s", e.Code, e.Message)
}

// Standard JSON-RPC error codes
const (
	CodeParseError     = -32700
	CodeInvalidRequest = -32600
	CodeMethodNotFound = -32601
	CodeInvalidParams  = -32602
	CodeInternalError  = -32603
	CodeSecurityDenied = -32000
)

// InitializeParams is sent by the host during handshake.
type InitializeParams struct {
	HostVersion     string   `json:"host_version"`
	ProtocolVersion string   `json:"protocol_version"`
	Capabilities    []string `json:"capabilities"`
}

// InitializeResult is returned by the extension during handshake.
type InitializeResult struct {
	Name            string   `json:"name"`
	Version         string   `json:"version"`
	ProtocolVersion string   `json:"protocol_version"`
	Permissions     []string `json:"permissions"`
	Capabilities    []string `json:"capabilities"`
}

// ExecuteParams specifies an action execution request.
type ExecuteParams struct {
	Action string   `json:"action"`
	Args   []string `json:"args,omitempty"`
}

// ExecuteResult contains the execution result payload.
type ExecuteResult struct {
	ExitCode int    `json:"exit_code"`
	Output   string `json:"output,omitempty"`
	Error    string `json:"error,omitempty"`
}
