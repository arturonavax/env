package bootstrap_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/arturonavax/env/internal/bootstrap"
)

func TestNewContainer(t *testing.T) {
	c, err := bootstrap.NewContainer(
		bootstrap.WithVersion("1.0.0-test", "abcdef", "2026-10-30"),
		bootstrap.WithOffline(true),
	)
	if err != nil {
		t.Fatalf("failed to create bootstrap container: %v", err)
	}
	t.Cleanup(func() { _ = c.Close() })

	if c.Platform == nil {
		t.Errorf("expected platform info in container")
	}
	if c.Catalog == nil {
		t.Errorf("expected catalog in container")
	}
	if c.InstallSvc == nil {
		t.Errorf("expected InstallSvc in container")
	}
	if c.SyncSvc == nil {
		t.Errorf("expected SyncSvc in container")
	}
	if c.Version != "1.0.0-test" {
		t.Errorf("expected version '1.0.0-test', got %q", c.Version)
	}
}

func TestResolveRepoRoot(t *testing.T) {
	tmpDir := t.TempDir()
	manifestsDir := filepath.Join(tmpDir, "manifests")
	filesDir := filepath.Join(tmpDir, "files")
	if err := os.MkdirAll(manifestsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filesDir, 0o755); err != nil {
		t.Fatal(err)
	}

	resolved := bootstrap.ResolveRepoRoot(tmpDir, "", "")
	if resolved != tmpDir {
		t.Errorf("expected resolved repo root %q, got %q", tmpDir, resolved)
	}

	// When configured is empty and cwd is not a repo (or when testing explicit configured)
	otherDir := filepath.Join(tmpDir, "other")
	if err := os.MkdirAll(filepath.Join(otherDir, "manifests"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(otherDir, "files"), 0o755); err != nil {
		t.Fatal(err)
	}
	resolvedOther := bootstrap.ResolveRepoRoot(otherDir, "", "")
	if resolvedOther != otherDir {
		t.Errorf("expected resolved configured %q, got %q", otherDir, resolvedOther)
	}
}
