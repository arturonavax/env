package configsync_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/arturonavax/env/internal/configsync"
	"github.com/arturonavax/env/internal/domain"
	"github.com/arturonavax/env/internal/security"
)

// HS01: Herdr config capturable, unauthorized file unknown-secret-2027.json never captured.
func TestHS01_HerdrConfigCaptureAllowlistOnly(t *testing.T) {
	home := t.TempDir()
	repo := t.TempDir()

	herdrUserDir := filepath.Join(home, ".config", "herdr")
	if err := os.MkdirAll(herdrUserDir, 0o755); err != nil {
		t.Fatal(err)
	}
	_ = os.WriteFile(filepath.Join(herdrUserDir, "config.toml"), []byte("theme = 'tokyo-night'\n"), 0o644)
	_ = os.WriteFile(filepath.Join(herdrUserDir, "unknown-secret-2027.json"), []byte("{\"secret\":\"leaked\"}"), 0o600)

	herdrRepoDir := filepath.Join(repo, "files", "herdr")
	if err := os.MkdirAll(herdrRepoDir, 0o755); err != nil {
		t.Fatal(err)
	}
	_ = os.WriteFile(filepath.Join(herdrRepoDir, "config.toml"), []byte("theme = 'default'\n"), 0o644)

	cfg := domain.ConfigurationResource{
		ID:       "herdr-config",
		Source:   "files/herdr/config.toml",
		Strategy: domain.StrategyThreeWay,
		Target:   "~/.config/herdr/config.toml",
	}

	captured, err := configsync.CaptureToRepoDetailed(home, repo, cfg, "linux", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(captured) != 1 || filepath.ToSlash(captured[0]) != "files/herdr/config.toml" {
		t.Fatalf("expected only files/herdr/config.toml captured, got: %v", captured)
	}

	// Verify unknown-secret was NOT copied into repo
	if _, err := os.Stat(filepath.Join(herdrRepoDir, "unknown-secret-2027.json")); !os.IsNotExist(err) {
		t.Fatalf("unauthorized secret file must NOT exist in repo!")
	}
}

// HS02: session.json, herdr.sock, session-snapshots, logs, sqlite db remain excluded in directory sync.
func TestHS02_RuntimeFilesExcludedInDirectorySync(t *testing.T) {
	home := t.TempDir()
	repo := t.TempDir()

	userDir := filepath.Join(home, ".config", "app")
	repoDir := filepath.Join(repo, "files", "app")
	_ = os.MkdirAll(userDir, 0o755)
	_ = os.MkdirAll(repoDir, 0o755)
	_ = os.MkdirAll(filepath.Join(userDir, "session-snapshots"), 0o755)

	_ = os.WriteFile(filepath.Join(userDir, "valid.conf"), []byte("setting=1\n"), 0o644)
	_ = os.WriteFile(filepath.Join(userDir, "session.json"), []byte("{\"active\":true}\n"), 0o600)
	_ = os.WriteFile(filepath.Join(userDir, "debug.log"), []byte("log data\n"), 0o600)
	_ = os.WriteFile(filepath.Join(userDir, "state.db"), []byte("sqlite binary\n"), 0o600)
	_ = os.WriteFile(filepath.Join(userDir, "state.sqlite-wal"), []byte("wal\n"), 0o600)
	_ = os.WriteFile(filepath.Join(userDir, "session-snapshots", "snap1.json"), []byte("snap\n"), 0o600)

	cfg := domain.ConfigurationResource{
		ID:       "app-dir",
		Source:   "files/app",
		Strategy: domain.StrategyDirectorySync,
		Target:   "~/.config/app",
	}

	captured, err := configsync.CaptureToRepoDetailed(home, repo, cfg, "linux", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, p := range captured {
		base := filepath.Base(p)
		if base == "session.json" || strings.HasSuffix(base, ".log") || strings.HasSuffix(base, ".db") || strings.HasSuffix(base, ".sqlite-wal") || strings.Contains(p, "session-snapshots") {
			t.Fatalf("runtime artifact %q leaked into captured list: %v", p, captured)
		}
	}
	if len(captured) != 1 || filepath.Base(captured[0]) != "valid.conf" {
		t.Fatalf("expected only valid.conf captured, got: %v", captured)
	}
}

// HS04: dry-run causes 0 file writes in repository.
func TestHS04_DryRunZeroWrites(t *testing.T) {
	home := t.TempDir()
	repo := t.TempDir()

	userFile := filepath.Join(home, ".config", "tool.conf")
	repoFile := filepath.Join(repo, "files", "tool.conf")
	_ = os.MkdirAll(filepath.Dir(userFile), 0o755)
	_ = os.MkdirAll(filepath.Dir(repoFile), 0o755)

	_ = os.WriteFile(userFile, []byte("user-content\n"), 0o644)
	_ = os.WriteFile(repoFile, []byte("repo-content\n"), 0o644)

	cfg := domain.ConfigurationResource{
		ID:       "tool",
		Source:   "files/tool.conf",
		Strategy: domain.StrategyThreeWay,
		Target:   "~/.config/tool.conf",
	}

	captured, err := configsync.CaptureToRepoDetailed(home, repo, cfg, "linux", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(captured) != 1 {
		t.Fatalf("expected 1 drift path reported in dry-run, got %d", len(captured))
	}

	// Verify file in repo was NOT modified
	data, _ := os.ReadFile(repoFile)
	if string(data) != "repo-content\n" {
		t.Fatalf("dry-run mutated repository content: %q", string(data))
	}
}

// HS06: Path traversal outside repository root is rejected.
func TestHS06_PathTraversalRejected(t *testing.T) {
	home := t.TempDir()
	repo := t.TempDir()

	cfg := domain.ConfigurationResource{
		ID:       "evil",
		Source:   "../../etc/passwd",
		Strategy: domain.StrategyThreeWay,
		Target:   "~/.evil",
	}

	_, err := configsync.CaptureToRepoDetailed(home, repo, cfg, "linux", false)
	if err == nil {
		t.Fatal("expected error on path traversal source, got nil")
	}
}

// HS07: Empty or invalid paths fail closed.
func TestHS07_EmptyPathsFailClosed(t *testing.T) {
	_, err := configsync.CaptureToRepoDetailed("", "", domain.ConfigurationResource{}, "linux", false)
	if err == nil {
		t.Fatal("expected error on empty paths, got nil")
	}
}

// HS08: Secret scan blocks credentials in captured files.
func TestHS08_SecretScanDetection(t *testing.T) {
	secretData := `
[api]
token = "AKIAIOSFODNN7EXAMPLE"
`
	findings := security.ScanForSecrets(secretData)
	if len(findings) == 0 {
		t.Fatal("expected secret findings for AWS key, got 0")
	}
	if findings[0].RuleID != "SEC-AWSKEY" {
		t.Fatalf("expected SEC-AWSKEY, got %s", findings[0].RuleID)
	}
}

// HS11: Preserves executable permissions when capturing executable files.
func TestHS11_PreservesPermissions(t *testing.T) {
	home := t.TempDir()
	repo := t.TempDir()

	userDir := filepath.Join(home, ".local", "bin")
	repoDir := filepath.Join(repo, "files", "scripts")
	_ = os.MkdirAll(userDir, 0o755)

	exeFile := filepath.Join(userDir, "run.sh")
	_ = os.WriteFile(exeFile, []byte("#!/bin/sh\necho hello\n"), 0o755)

	cfg := domain.ConfigurationResource{
		ID:       "scripts-dir",
		Source:   "files/scripts",
		Strategy: domain.StrategyDirectorySync,
		Target:   "~/.local/bin",
	}

	captured, err := configsync.CaptureToRepoDetailed(home, repo, cfg, "linux", false)
	if err != nil {
		t.Fatalf("capture error: %v", err)
	}
	if len(captured) != 1 {
		t.Fatalf("expected 1 captured script, got %d", len(captured))
	}

	info, err := os.Stat(filepath.Join(repoDir, "run.sh"))
	if err != nil {
		t.Fatalf("stat repo file: %v", err)
	}
	if info.Mode()&0o111 == 0 {
		t.Errorf("expected executable bit preserved on run.sh, got mode: %v", info.Mode())
	}
}
