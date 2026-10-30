package cli

import (
	"context"
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
