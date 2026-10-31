package tools_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/arturonavax/env/internal/domain"
	"github.com/arturonavax/env/internal/platform"
	"github.com/arturonavax/env/internal/tools"
)

func TestToolRegistryEmbeddedLoading(t *testing.T) {
	r, err := tools.LoadRegistry()
	if err != nil {
		t.Fatalf("failed to load registry: %v", err)
	}

	list := r.List()
	if len(list) == 0 {
		t.Fatalf("expected embedded tools to load, got 0")
	}

	ai, ok := r.Get("ai")
	if !ok {
		t.Fatalf("expected tool 'ai' to be registered")
	}
	if !ai.FallbackBuiltin {
		t.Errorf("expected 'ai' to have FallbackBuiltin true")
	}

	st, err := r.CheckStatus(context.Background(), "ai")
	if err != nil {
		t.Fatalf("unexpected error checking ai status: %v", err)
	}
	if !st.Available {
		t.Errorf("expected 'ai' to be available")
	}
	if st.Kind != domain.ToolKindBuiltin && st.Kind != domain.ToolKindExternal {
		t.Errorf("expected kind builtin or external, got %s", st.Kind)
	}
}

func TestToolRunnerBuiltin(t *testing.T) {
	tempHome := t.TempDir()
	r, err := tools.LoadRegistry(
		tools.WithHomeDir(tempHome),
		tools.WithLookPath(func(name string) (string, error) {
			return "", fmt.Errorf("not found: %s", name)
		}),
	)
	if err != nil {
		t.Fatalf("failed to load registry: %v", err)
	}

	runner := tools.NewRunner(r)
	called := false
	runner.RegisterBuiltin("ai", func(ctx context.Context, toolID string, args []string, in io.Reader, out, errOut io.Writer) error {
		called = true
		fmt.Fprintf(out, "hello from builtin %s with args %v", toolID, args)
		return nil
	})

	var outBuf, errBuf bytes.Buffer
	err = runner.Execute(context.Background(), "ai", []string{"commit", "-m", "test"}, nil, &outBuf, &errBuf)
	if err != nil {
		t.Fatalf("unexpected runner error: %v", err)
	}
	if !called {
		t.Fatalf("expected builtin handler to be called")
	}
	if !bytes.Contains(outBuf.Bytes(), []byte("hello from builtin ai")) {
		t.Fatalf("expected output, got: %s", outBuf.String())
	}
}

func TestToolRunnerExternal(t *testing.T) {
	mockDir := t.TempDir()
	binName := "custom-tool"
	var binPath string
	if runtime.GOOS == "windows" {
		binPath = filepath.Join(mockDir, binName+".cmd")
		_ = os.WriteFile(binPath, []byte("@echo off\r\necho %*\r\n"), 0o755)
	} else {
		binPath = filepath.Join(mockDir, binName)
		_ = os.WriteFile(binPath, []byte("#!/bin/sh\necho \"$*\"\n"), 0o755)
	}

	r, err := tools.LoadRegistry(
		tools.WithLookPath(func(name string) (string, error) {
			if name == "echo" || name == "custom-tool" {
				return binPath, nil
			}
			return "", fmt.Errorf("not found")
		}),
	)
	if err != nil {
		t.Fatalf("failed to load registry: %v", err)
	}

	r.Register(domain.ToolManifest{
		ID:              "custom-tool",
		Executable:      "custom-tool",
		AllowGlobalPATH: true,
	})

	runner := tools.NewRunner(r)
	var outBuf, errBuf bytes.Buffer
	err = runner.Execute(context.Background(), "custom-tool", []string{"hello", "world"}, nil, &outBuf, &errBuf)
	if err != nil {
		t.Fatalf("unexpected error running external tool: %v", err)
	}
	if !bytes.Contains(outBuf.Bytes(), []byte("hello world")) {
		t.Fatalf("expected echo output, got: %s", outBuf.String())
	}
}

func TestToolRunnerMissing(t *testing.T) {
	r, err := tools.LoadRegistry(
		tools.WithLookPath(func(string) (string, error) {
			return "", fmt.Errorf("not found")
		}),
	)
	if err != nil {
		t.Fatalf("failed to load registry: %v", err)
	}

	r.Register(domain.ToolManifest{
		ID:              "non-existent-tool",
		Executable:      "non-existent-binary",
		FallbackBuiltin: false,
	})

	runner := tools.NewRunner(r)
	var outBuf, errBuf bytes.Buffer
	err = runner.Execute(context.Background(), "non-existent-tool", nil, nil, &outBuf, &errBuf)
	if err == nil {
		t.Fatalf("expected error running missing tool, got nil")
	}
}

func TestToolUpdater(t *testing.T) {
	r, err := tools.LoadRegistry()
	if err != nil {
		t.Fatalf("failed to load registry: %v", err)
	}

	packageUpdated := false
	updater := tools.NewUpdater(r, func(ctx context.Context, packageID string) error {
		if packageID == "usql" {
			packageUpdated = true
			return nil
		}
		return fmt.Errorf("unknown package")
	})

	res, err := updater.Update(context.Background(), "usql")
	if err != nil {
		t.Fatalf("unexpected error updating usql: %v", err)
	}
	if !res.Updated || !packageUpdated {
		t.Fatalf("expected package updater to be invoked, got: %+v", res)
	}

	// Test built-in update check
	resAI, err := updater.Update(context.Background(), "ai")
	if err != nil {
		t.Fatalf("unexpected error on ai update check: %v", err)
	}
	if resAI.Status == "error" {
		t.Fatalf("unexpected ai update status: %+v", resAI)
	}
}

func TestToolIsolationNoGenericLocalBin(t *testing.T) {
	tempHome := t.TempDir()
	plat := platform.Detect()

	r, err := tools.LoadRegistry(
		tools.WithHomeDir(tempHome),
		tools.WithPlatform(plat),
		tools.WithLookPath(func(string) (string, error) {
			return "", fmt.Errorf("not found")
		}),
	)
	if err != nil {
		t.Fatalf("failed to load registry: %v", err)
	}

	r.Register(domain.ToolManifest{
		ID:              "isolated-tool",
		Executable:      "isolated-tool",
		FallbackBuiltin: false,
		AllowGlobalPATH: false,
	})

	binSuffix := ""
	if plat.OS() == "windows" {
		binSuffix = ".exe"
	}

	// 1. Placing binary in generic ~/.local/bin/ must NOT be picked up
	legacyLocalBin := filepath.Join(tempHome, ".local", "bin")
	_ = os.MkdirAll(legacyLocalBin, 0o755)
	legacyBinary := filepath.Join(legacyLocalBin, "isolated-tool"+binSuffix)
	_ = os.WriteFile(legacyBinary, []byte("#!/bin/sh\necho legacy\n"), 0o755)

	st, err := r.CheckStatus(context.Background(), "isolated-tool")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if st.Available {
		t.Errorf("isolated tool should NOT find binary in generic ~/.local/bin")
	}

	// 2. Placing binary in isolated ~/.local/share/anx/bin/ MUST be picked up
	isolatedBin := filepath.Join(tempHome, ".local", "share", "anx", "bin")
	t.Setenv("ANX_TOOLS_BIN", isolatedBin)
	// reload registry with new env
	r2, err := tools.LoadRegistry(
		tools.WithHomeDir(tempHome),
		tools.WithLookPath(func(string) (string, error) {
			return "", fmt.Errorf("not found")
		}),
	)
	if err != nil {
		t.Fatalf("failed to reload registry: %v", err)
	}
	r2.Register(domain.ToolManifest{
		ID:              "isolated-tool",
		Executable:      "isolated-tool",
		FallbackBuiltin: false,
		AllowGlobalPATH: false,
	})

	_ = os.MkdirAll(isolatedBin, 0o755)
	isolatedFile := filepath.Join(isolatedBin, "isolated-tool"+binSuffix)
	_ = os.WriteFile(isolatedFile, []byte("#!/bin/sh\necho isolated\n"), 0o755)

	st2, err := r2.CheckStatus(context.Background(), "isolated-tool")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !st2.Available {
		t.Errorf("isolated tool SHOULD find binary in isolated directory %s", isolatedBin)
	}
	if st2.Path != isolatedFile {
		t.Errorf("expected path %s, got %s", isolatedFile, st2.Path)
	}
}
