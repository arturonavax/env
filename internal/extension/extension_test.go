package extension_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/arturonavax/env/internal/extension"
)

func writeExecutableScript(t *testing.T, path, script string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
}

func TestExtensionHandshakeAndExecute(t *testing.T) {
	tempDir := t.TempDir()
	scriptPath := filepath.Join(tempDir, "mock_plugin.sh")

	// Mock plugin implementing stdio JSON-RPC v1.0
	script := `#!/bin/sh
while IFS= read -r line; do
  method=$(echo "$line" | grep -o '"method":"[^"]*"' | cut -d'"' -f4)
  id=$(echo "$line" | grep -o '"id":[0-9]*' | cut -d':' -f2)

  if [ "$method" = "initialize" ]; then
    echo "{\"jsonrpc\":\"2.0\",\"id\":$id,\"result\":{\"name\":\"test-plugin\",\"version\":\"0.1.0\",\"protocol_version\":\"1.0\",\"permissions\":[\"read\"],\"capabilities\":[\"echo\"]}}"
  elif [ "$method" = "execute" ]; then
    echo "{\"jsonrpc\":\"2.0\",\"id\":$id,\"result\":{\"exit_code\":0,\"output\":\"hello from plugin\"}}"
  else
    echo "{\"jsonrpc\":\"2.0\",\"id\":$id,\"error\":{\"code\":-32601,\"message\":\"Method not found\"}}"
  fi
done
`
	writeExecutableScript(t, scriptPath, script)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	client, err := extension.Launch(ctx, scriptPath, []string{"read"}, 0)
	if err != nil {
		t.Fatalf("failed launching extension: %v", err)
	}
	defer client.Close()

	if client.Info().Name != "test-plugin" {
		t.Errorf("expected plugin name test-plugin, got %s", client.Info().Name)
	}

	execRes, err := client.Execute(ctx, "echo", []string{"foo"})
	if err != nil {
		t.Fatalf("failed executing plugin action: %v", err)
	}
	if execRes.ExitCode != 0 || execRes.Output != "hello from plugin" {
		t.Errorf("unexpected execute result: %+v", execRes)
	}
}

func TestExtensionProtocolVersionMismatch(t *testing.T) {
	tempDir := t.TempDir()
	scriptPath := filepath.Join(tempDir, "incompatible_plugin.sh")

	script := `#!/bin/sh
while IFS= read -r line; do
  id=$(echo "$line" | grep -o '"id":[0-9]*' | cut -d':' -f2)
  echo "{\"jsonrpc\":\"2.0\",\"id\":$id,\"result\":{\"name\":\"bad\",\"version\":\"2.0\",\"protocol_version\":\"99.0\",\"permissions\":[]}}"
done
`
	writeExecutableScript(t, scriptPath, script)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := extension.Launch(ctx, scriptPath, nil, 0)
	if err == nil {
		t.Fatal("expected error on incompatible protocol version, got nil")
	}
	if !errors.Is(err, extension.ErrProtocolMismatch) {
		t.Fatalf("expected ErrProtocolMismatch, got: %v", err)
	}
}

func TestExtensionDefaultDenyUnpermitted(t *testing.T) {
	tempDir := t.TempDir()
	scriptPath := filepath.Join(tempDir, "unpermitted_plugin.sh")

	// Plugin asks for "root_access", host only allows "read"
	script := `#!/bin/sh
while IFS= read -r line; do
  id=$(echo "$line" | grep -o '"id":[0-9]*' | cut -d':' -f2)
  echo "{\"jsonrpc\":\"2.0\",\"id\":$id,\"result\":{\"name\":\"evil\",\"version\":\"1.0\",\"protocol_version\":\"1.0\",\"permissions\":[\"root_access\"]}}"
done
`
	writeExecutableScript(t, scriptPath, script)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := extension.Launch(ctx, scriptPath, []string{"read"}, 0)
	if err == nil {
		t.Fatal("expected error on unpermitted permission, got nil")
	}
	if !errors.Is(err, extension.ErrPermissionDenied) {
		t.Fatalf("expected ErrPermissionDenied, got: %v", err)
	}
}

func TestExtensionCorruptedOutput(t *testing.T) {
	tempDir := t.TempDir()
	scriptPath := filepath.Join(tempDir, "corrupted_plugin.sh")

	script := `#!/bin/sh
while IFS= read -r line; do
  echo "GARBAGE NOT JSON DATA"
done
`
	writeExecutableScript(t, scriptPath, script)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := extension.Launch(ctx, scriptPath, nil, 0)
	if err == nil {
		t.Fatal("expected error on corrupted output, got nil")
	}
	if !errors.Is(err, extension.ErrCorruptedResponse) {
		t.Fatalf("expected ErrCorruptedResponse, got: %v", err)
	}
}

func TestExtensionTimeoutKillsHangingProcess(t *testing.T) {
	tempDir := t.TempDir()
	scriptPath := filepath.Join(tempDir, "hanging_plugin.sh")

	// Plugin accepts initialize, then hangs on execute
	script := `#!/bin/sh
while IFS= read -r line; do
  method=$(echo "$line" | grep -o '"method":"[^"]*"' | cut -d'"' -f4)
  id=$(echo "$line" | grep -o '"id":[0-9]*' | cut -d':' -f2)

  if [ "$method" = "initialize" ]; then
    echo "{\"jsonrpc\":\"2.0\",\"id\":$id,\"result\":{\"name\":\"hang\",\"version\":\"1.0\",\"protocol_version\":\"1.0\",\"permissions\":[]}}"
  else
    sleep 30
  fi
done
`
	writeExecutableScript(t, scriptPath, script)

	client, err := extension.Launch(context.Background(), scriptPath, nil, 0)
	if err != nil {
		t.Fatalf("launch failed: %v", err)
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err = client.Execute(ctx, "hang", nil)
	if err == nil {
		t.Fatal("expected timeout error on hanging plugin, got nil")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected DeadlineExceeded, got: %v", err)
	}
}
