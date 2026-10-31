package extension_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
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

	exeName := "mock_plugin"
	if runtime.GOOS == "windows" {
		exeName += ".exe"
	}
	mockPluginBin = filepath.Join(tempDir, exeName)

	cmd := exec.Command("go", "build", "-o", mockPluginBin, "./testdata/mock_plugin")
	if out, err := cmd.CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to build mock plugin: %v, output: %s\n", err, out)
		_ = os.RemoveAll(tempDir)
		os.Exit(1)
	}

	code := m.Run()
	_ = os.RemoveAll(tempDir)
	os.Exit(code)
}

// pluginSpec copies the mock plugin under a variant name (the mock selects behaviour from its
// file name), pins it by SHA-256 and returns a Spec granting exactly `perms`.
func pluginSpec(t *testing.T, variant string, perms ...string) extension.Spec {
	t.Helper()
	ext := ""
	if runtime.GOOS == "windows" {
		ext = ".exe"
	}
	dir := t.TempDir()
	if err := os.Chmod(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(dir, variant+ext)

	data, err := os.ReadFile(mockPluginBin)
	if err != nil {
		t.Fatalf("reading mock plugin template: %v", err)
	}
	if err := os.WriteFile(dest, data, 0o755); err != nil {
		t.Fatalf("writing plugin variant: %v", err)
	}
	sum, err := extension.HashFile(dest)
	if err != nil {
		t.Fatal(err)
	}
	return extension.Spec{Path: dest, SHA256: sum, Declared: perms, Allowed: perms}
}

func ctx5s(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)
	return ctx
}

func TestExtensionHandshakeAndExecute(t *testing.T) {
	client, err := extension.Launch(ctx5s(t), pluginSpec(t, "standard_plugin", "read"))
	if err != nil {
		t.Fatalf("failed launching extension: %v", err)
	}
	defer client.Close()

	if client.Info().Name != "test-plugin" {
		t.Errorf("expected plugin name test-plugin, got %s", client.Info().Name)
	}

	res, err := client.Execute(ctx5s(t), "echo", []string{"foo"})
	if err != nil {
		t.Fatalf("failed executing plugin action: %v", err)
	}
	if res.ExitCode != 0 || res.Output != "hello from plugin" {
		t.Errorf("unexpected execute result: %+v", res)
	}
}

func TestExtensionProtocolVersionMismatch(t *testing.T) {
	_, err := extension.Launch(ctx5s(t), pluginSpec(t, "incompatible_plugin"))
	if !errors.Is(err, extension.ErrProtocolMismatch) {
		t.Fatalf("expected ErrProtocolMismatch, got: %v", err)
	}
}

// The binary asks for more than it declared: rejected at handshake.
func TestExtensionHandshakeRejectsUndeclaredPermission(t *testing.T) {
	_, err := extension.Launch(ctx5s(t), pluginSpec(t, "unpermitted_plugin", "read"))
	if !errors.Is(err, extension.ErrPermissionDenied) {
		t.Fatalf("expected ErrPermissionDenied, got: %v", err)
	}
}

func TestExtensionCorruptedOutput(t *testing.T) {
	_, err := extension.Launch(ctx5s(t), pluginSpec(t, "corrupted_plugin"))
	if !errors.Is(err, extension.ErrCorruptedResponse) {
		t.Fatalf("expected ErrCorruptedResponse, got: %v", err)
	}
}

func TestExtensionTimeoutKillsHangingProcess(t *testing.T) {
	client, err := extension.Launch(context.Background(), pluginSpec(t, "hanging_plugin"))
	if err != nil {
		t.Fatalf("launch failed: %v", err)
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	if _, err = client.Execute(ctx, "hang", nil); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected DeadlineExceeded, got: %v", err)
	}
}

// SEC-01 regression: over-privileged extension must be refused BEFORE any code of it runs.
func TestAdmissionDeniesBeforeExecution(t *testing.T) {
	skipUnlessUnix(t)

	marker := filepath.Join(t.TempDir(), "executed")
	script := shellMarkerScript(t, marker)

	tests := []struct {
		name    string
		mutate  func(*extension.Spec)
		wantErr error
	}{
		{"declared exceeds allowed", func(s *extension.Spec) {
			s.Declared = []string{"root_access"}
			s.Allowed = []string{"read"}
		}, extension.ErrPermissionDenied},
		{"unpinned", func(s *extension.Spec) { s.SHA256 = "" }, extension.ErrNotPinned},
		{"malformed pin", func(s *extension.Spec) { s.SHA256 = "not-hex" }, extension.ErrNotPinned},
		{"tampered binary", func(s *extension.Spec) { s.SHA256 = strings.Repeat("0", 64) }, extension.ErrIntegrity},
		{"relative path", func(s *extension.Spec) { s.Path = "plugin.sh" }, extension.ErrUntrustedBinary},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			spec := script
			tc.mutate(&spec)

			_, err := extension.Launch(ctx5s(t), spec)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("want %v, got %v", tc.wantErr, err)
			}
			if _, statErr := os.Stat(marker); statErr == nil {
				t.Fatal("extension executed despite failed admission")
			}
		})
	}
}

func TestAdmissionRejectsWritableBinaryAndDirectory(t *testing.T) {
	skipUnlessUnix(t)

	spec := pluginSpec(t, "standard_plugin", "read")
	if err := os.Chmod(spec.Path, 0o777); err != nil {
		t.Fatal(err)
	}
	if _, err := extension.Launch(ctx5s(t), spec); !errors.Is(err, extension.ErrUntrustedBinary) {
		t.Fatalf("world-writable binary: want ErrUntrustedBinary, got %v", err)
	}

	spec = pluginSpec(t, "standard_plugin", "read")
	if err := os.Chmod(filepath.Dir(spec.Path), 0o777); err != nil {
		t.Fatal(err)
	}
	if _, err := extension.Launch(ctx5s(t), spec); !errors.Is(err, extension.ErrUntrustedBinary) {
		t.Fatalf("world-writable directory: want ErrUntrustedBinary, got %v", err)
	}
}

func TestAdmissionFollowsSymlinkToPinnedTarget(t *testing.T) {
	skipUnlessUnix(t)

	spec := pluginSpec(t, "standard_plugin", "read")
	link := filepath.Join(filepath.Dir(spec.Path), "link")
	if err := os.Symlink(spec.Path, link); err != nil {
		t.Fatal(err)
	}
	spec.Path = link

	client, err := extension.Launch(ctx5s(t), spec)
	if err != nil {
		t.Fatalf("symlink to pinned binary should be admitted: %v", err)
	}
	_ = client.Close()
}

func TestChildEnvironmentIsScrubbed(t *testing.T) {
	t.Setenv("ANX_TEST_SECRET", "hunter2")

	client, err := extension.Launch(ctx5s(t), pluginSpec(t, "envprobe_plugin"))
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	name := client.Info().Name
	if strings.Contains(name, "hunter2") {
		t.Fatalf("host secret leaked into extension environment: %s", name)
	}
	if !strings.Contains(name, `secret=""`) {
		t.Fatalf("unexpected probe output: %s", name)
	}
}

func TestExplicitEnvIsPassedThrough(t *testing.T) {
	spec := pluginSpec(t, "envprobe_plugin")
	spec.Env = []string{"ANX_TEST_SECRET=granted"}

	client, err := extension.Launch(ctx5s(t), spec)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	if !strings.Contains(client.Info().Name, `secret="granted"`) {
		t.Fatalf("explicit env not delivered: %s", client.Info().Name)
	}
}

func TestChildRunsInPrivateScratchDirRemovedOnClose(t *testing.T) {
	client, err := extension.Launch(ctx5s(t), pluginSpec(t, "envprobe_plugin"))
	if err != nil {
		t.Fatal(err)
	}

	_, after, ok := strings.Cut(client.Info().Name, "cwd=")
	if !ok {
		t.Fatalf("no cwd in probe output: %s", client.Info().Name)
	}
	cwd := after
	if cwd == mustGetwd(t) {
		t.Fatal("extension inherited the host working directory")
	}
	if _, err := os.Stat(cwd); err != nil {
		t.Fatalf("scratch dir should exist while running: %v", err)
	}

	_ = client.Close()
	if _, err := os.Stat(cwd); !os.IsNotExist(err) {
		t.Fatalf("scratch dir must be removed on Close, stat err = %v", err)
	}
}

func TestOversizedMessageIsBoundedAndKillsExtension(t *testing.T) {
	spec := pluginSpec(t, "flood_plugin")
	spec.MaxMessageBytes = 64 * 1024

	_, err := extension.Launch(ctx5s(t), spec)
	if !errors.Is(err, extension.ErrMessageTooLarge) {
		t.Fatalf("want ErrMessageTooLarge, got %v", err)
	}
}

func TestMessageLimitIsPerMessageNotPerLifetime(t *testing.T) {
	spec := pluginSpec(t, "standard_plugin", "read")
	spec.MaxMessageBytes = 4096

	client, err := extension.Launch(ctx5s(t), spec)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	// Total traffic far exceeds MaxMessageBytes, yet every message is small: all must succeed.
	for i := range 200 {
		if _, err := client.Execute(ctx5s(t), "echo", nil); err != nil {
			t.Fatalf("call %d failed: %v", i, err)
		}
	}
}

func skipUnlessUnix(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("POSIX ownership/permission and shell-script semantics")
	}
}

func shellMarkerScript(t *testing.T, marker string) extension.Spec {
	t.Helper()
	dir := t.TempDir()
	if err := os.Chmod(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "plugin.sh")
	body := fmt.Sprintf("#!/bin/sh\ntouch %q\n", marker)
	if err := os.WriteFile(path, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	sum, err := extension.HashFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return extension.Spec{Path: path, SHA256: sum, Declared: []string{"read"}, Allowed: []string{"read"}}
}

func mustGetwd(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	return wd
}
