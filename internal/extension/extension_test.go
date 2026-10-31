package extension_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/arturonavax/env/internal/extension"
)

var mockPluginBin string

func TestMain(m *testing.M) {
	tempDir, err := os.MkdirTemp("", "mock-plugin-build-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create temp dir: %v\n", err)
		os.Exit(1)
	}
	defer os.RemoveAll(tempDir)

	exeName := "mock_plugin"
	if runtime.GOOS == "windows" {
		exeName += ".exe"
	}
	mockPluginBin = filepath.Join(tempDir, exeName)

	cmd := exec.Command("go", "build", "-o", mockPluginBin, "./testdata/mock_plugin")
	if out, err := cmd.CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to build mock plugin: %v, output: %s\n", err, out)
		os.Exit(1)
	}

	os.Exit(m.Run())
}

func getPluginVariant(t *testing.T, variant string) string {
	t.Helper()
	ext := ""
	if runtime.GOOS == "windows" {
		ext = ".exe"
	}
	destPath := filepath.Join(t.TempDir(), variant+ext)
	data, err := os.ReadFile(mockPluginBin)
	if err != nil {
		t.Fatalf("reading mock plugin template: %v", err)
	}
	if err := os.WriteFile(destPath, data, 0o755); err != nil {
		t.Fatalf("writing plugin variant: %v", err)
	}
	return destPath
}

func TestExtensionHandshakeAndExecute(t *testing.T) {
	scriptPath := getPluginVariant(t, "standard_plugin")

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
	scriptPath := getPluginVariant(t, "incompatible_plugin")

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
	scriptPath := getPluginVariant(t, "unpermitted_plugin")

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
	scriptPath := getPluginVariant(t, "corrupted_plugin")

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
	scriptPath := getPluginVariant(t, "hanging_plugin")

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
