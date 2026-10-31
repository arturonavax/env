package cli

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/viper"
)

func TestParseGitRepoOwner(t *testing.T) {
	tests := []struct {
		url      string
		expected string
	}{
		{"git@github.com:arturonavax/env.git", "arturonavax"},
		{"git@github.com:octocat/my-fork.git", "octocat"},
		{"https://github.com/arturonavax/env.git", "arturonavax"},
		{"https://github.com/contributor/env", "contributor"},
		{"http://gitlab.internal.corp/team-devops/configs.git", "team-devops"},
		{"invalid-url", ""},
		{"", ""},
	}

	for _, tc := range tests {
		got := parseGitRepoOwner(tc.url)
		if got != tc.expected {
			t.Errorf("parseGitRepoOwner(%q) = %q, want %q", tc.url, got, tc.expected)
		}
	}
}

func TestCheckGitPushAuthorization(t *testing.T) {
	ctx := context.Background()

	// 1. When allowed_users is empty, anyone is authorized
	appCtxOpen := &AppContext{
		RepoRoot: t.TempDir(),
		Viper:    viper.New(),
	}
	resOpen := checkGitPushAuthorization(ctx, appCtxOpen)
	if !resOpen.IsAuthorized {
		t.Errorf("expected open authorization when no restrictions configured, got denied: %s", resOpen.Reason)
	}

	// 2. When repo_owner matches active user
	vWithAllowed := viper.New()
	vWithAllowed.Set("repo_owner", "alice")
	vWithAllowed.Set("allowed_push_users", []string{"bob", "carol"})

	appCtxRestricted := &AppContext{
		RepoRoot: t.TempDir(),
		Viper:    vWithAllowed,
	}

	// Active user is nil/empty in test environment without git/gh user,
	// so it should be denied against ["alice", "bob", "carol"]
	resRestricted := checkGitPushAuthorization(ctx, appCtxRestricted)
	if resRestricted.IsAuthorized {
		t.Errorf("expected unauthorized when active user is empty and not in allowed list")
	}
	if len(resRestricted.AllowedUsers) != 3 {
		t.Errorf("expected 3 allowed users, got %d: %v", len(resRestricted.AllowedUsers), resRestricted.AllowedUsers)
	}
}

func TestCaptureDryRunZeroGitMutations(t *testing.T) {
	repo := t.TempDir()
	appCtx := &AppContext{
		RepoRoot: repo,
		Flags: GlobalFlagState{
			DryRun: true,
		},
	}
	cmd := newCaptureCmd(appCtx)
	cmd.SetContext(context.Background())

	// Calling executeGitCommitAndPush with DryRun=true must NOT fail even if repo is not a git repo
	err := executeGitCommitAndPush(cmd, appCtx, "test-branch", "commit msg", []string{"files/herdr/config.toml"}, true)
	if err != nil {
		t.Fatalf("expected nil error in dry-run, got: %v", err)
	}
}

func TestCaptureSecretScanBlocksCommit(t *testing.T) {
	repo := t.TempDir()
	secretFile := "files/leaked.conf"
	fullPath := filepath.Join(repo, secretFile)
	_ = os.MkdirAll(filepath.Dir(fullPath), 0o755)
	_ = os.WriteFile(fullPath, []byte("API_KEY=\"AKIAIOSFODNN7EXAMPLE\"\n"), 0o600)

	appCtx := &AppContext{
		RepoRoot: repo,
		Flags: GlobalFlagState{
			DryRun: false,
		},
	}
	cmd := newCaptureCmd(appCtx)
	cmd.SetContext(context.Background())

	err := executeGitCommitAndPush(cmd, appCtx, "test-branch", "commit msg", []string{secretFile}, false)
	if err == nil {
		t.Fatal("expected secret scan to block commit and return error, got nil")
	}
	if !strings.Contains(err.Error(), "security guardrail: secret detected") {
		t.Fatalf("unexpected error message: %v", err)
	}
}
