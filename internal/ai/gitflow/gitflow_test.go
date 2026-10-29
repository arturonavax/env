package gitflow_test

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/arturonavax/env/internal/ai/config"
	"github.com/arturonavax/env/internal/ai/gitflow"
	"github.com/arturonavax/env/internal/ai/harness"
	"github.com/arturonavax/env/internal/security"
)

func TestExtractTaskID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		branch string
		want   string
	}{
		{name: "Jira project key", branch: "feature/PROJ-1234-oauth-flow", want: "PROJ-1234"},
		{name: "Linear lowercase team key", branch: "bugfix/eng-42-fix-race", want: "ENG-42"},
		{name: "Bare Jira key", branch: "CORE-908", want: "CORE-908"},
		{name: "GitHub issue-N format", branch: "issue-42", want: "#42"},
		{name: "GitHub gh-N format", branch: "feat/gh-42-cli", want: "#42"},
		{name: "Leading issue number format", branch: "42-fix-null-pointer", want: "#42"},
		{name: "Prefixed slash leading issue number", branch: "fix/42-crash", want: "#42"},
		{name: "Explicit hash issue format", branch: "feat/#105-dashboard", want: "#105"},
		{name: "Main branch has no ticket", branch: "main", want: ""},
		{name: "Semver release branch has no ticket", branch: "release/v1.2.3", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := gitflow.ExtractTaskID(tt.branch); got != tt.want {
				t.Errorf("ExtractTaskID(%q) = %q, want %q", tt.branch, got, tt.want)
			}
		})
	}
}

func TestClipboardOSC52FallbackAndBinary(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	runner := security.NewRunner(5 * time.Second)

	// OSC-52 fallback when no clipboard binaries exist
	var oscBuf bytes.Buffer
	method, err := gitflow.CopyToClipboardWithOptions(ctx, runner, "hello-pr", gitflow.ClipboardOptions{
		FallbackWriter: &oscBuf,
		LookPath: func(string) (string, error) {
			return "", errors.New("missing")
		},
		Getenv: func(string) string { return "" },
	})
	if err != nil {
		t.Fatalf("CopyToClipboardWithOptions OSC52 error: %v", err)
	}
	if method != "osc52" || !strings.HasPrefix(oscBuf.String(), "\x1b]52;c;") {
		t.Errorf("got method=%q buf=%q, want osc52 escape", method, oscBuf.String())
	}

	// Fake wl-copy binary on Wayland
	binDir := t.TempDir()
	outCapture := filepath.Join(binDir, "clip.txt")
	wlCopy := filepath.Join(binDir, "wl-copy")
	script := "#!/bin/sh\ncat > " + outCapture + "\n"
	if err := os.WriteFile(wlCopy, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}

	method, err = gitflow.CopyToClipboardWithOptions(ctx, runner, "wayland-payload", gitflow.ClipboardOptions{
		LookPath: func(name string) (string, error) {
			if name == "wl-copy" {
				return wlCopy, nil
			}
			return "", errors.New("not found")
		},
		Getenv: func(k string) string {
			if k == "WAYLAND_DISPLAY" {
				return "wayland-0"
			}
			return ""
		},
	})
	if err != nil {
		t.Fatalf("wl-copy error: %v", err)
	}
	if method != "wl-copy" {
		t.Errorf("method = %q, want wl-copy", method)
	}
	copied, _ := os.ReadFile(outCapture)
	if string(copied) != "wayland-payload" {
		t.Errorf("captured clipboard = %q, want wayland-payload", string(copied))
	}
}

type mockHarnessRunner struct {
	lastStdin string
	lastArgs  []string
	response  string
}

func (m *mockHarnessRunner) ExecuteProcess(_ context.Context, spec harness.ProcessSpec) (harness.ProcessResult, error) {
	m.lastArgs = append([]string{}, spec.Args...)
	if spec.Stdin != nil {
		var b bytes.Buffer
		_, _ = b.ReadFrom(spec.Stdin)
		m.lastStdin = b.String()
	}
	return harness.ProcessResult{Stdout: m.response}, nil
}

func TestCommitAndPRWorkflowsInTempGitRepo(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	runner := security.NewRunner(15 * time.Second)
	repoDir := t.TempDir()

	gitEnv := append(os.Environ(),
		"GIT_AUTHOR_NAME=Test Engineer",
		"GIT_AUTHOR_EMAIL=test@example.com",
		"GIT_COMMITTER_NAME=Test Engineer",
		"GIT_COMMITTER_EMAIL=test@example.com",
	)
	runGit := func(args ...string) {
		t.Helper()
		if _, err := runner.RunWithOptions(ctx, repoDir, gitEnv, "git", args...); err != nil {
			t.Fatalf("git %v failed: %v", args, err)
		}
	}

	runGit("init", "-b", "main")
	if err := os.WriteFile(filepath.Join(repoDir, "README.md"), []byte("# Demo\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Add repo commit & PR templates to test auto-discovery!
	if err := os.WriteFile(filepath.Join(repoDir, ".gitmessage"), []byte("[{task_id}] <type>: <summary>\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	ghDir := filepath.Join(repoDir, ".github")
	if err := os.MkdirAll(ghDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ghDir, "pull_request_template.md"), []byte("## Summary\n\n## Testing\n- [ ] Unit tests\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	runGit("add", ".")
	runGit("commit", "-m", "chore: initial commit")

	// Create feature branch with Jira key
	runGit("checkout", "-b", "feature/ENG-42-add-service")
	if err := os.WriteFile(filepath.Join(repoDir, "service.go"), []byte("package demo\nfunc Run() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit("add", "service.go")

	cfg, err := config.DefaultConfig()
	if err != nil {
		t.Fatal(err)
	}
	mockProc := &mockHarnessRunner{response: "feat(demo): add service entrypoint"}
	reg := harness.NewRegistry(
		cfg,
		harness.WithProcessRunner(mockProc),
		harness.WithLookPath(func(bin string) (string, error) { return "/usr/bin/" + bin, nil }),
	)

	svc := gitflow.NewService(cfg, reg, runner)

	// 1. Test dry-run commit with repo template discovery and task extraction
	var commitOut, commitErr bytes.Buffer
	msg, err := svc.RunCommit(ctx, gitflow.CommitOptions{
		WorkDir: repoDir,
		DryRun:  true,
		Out:     &commitOut,
		ErrOut:  &commitErr,
	})
	if err != nil {
		t.Fatalf("RunCommit() dry-run error: %v", err)
	}
	if msg != "feat(demo): add service entrypoint" {
		t.Errorf("commit msg = %q", msg)
	}
	joinedArgs := strings.Join(mockProc.lastArgs, " ")
	if !strings.Contains(joinedArgs, "ENG-42") || !strings.Contains(joinedArgs, "Diff Stat:") {
		t.Errorf("expected ENG-42 task ID and Diff Stat in assembled prompt:\n%s", joinedArgs)
	}
	if !strings.Contains(commitErr.String(), "repo:.gitmessage") {
		t.Errorf("expected discovered repo:.gitmessage template in status output: %s", commitErr.String())
	}

	// Commit change so PR branch diff has a commit against main
	runGit("commit", "-m", msg)

	// 2. Test PR generation with discovered .github/pull_request_template.md
	mockProc.response = "## Summary\nImplements ENG-42 service.\n\n## Testing\n- [x] Unit tests"
	prOutFile := filepath.Join(t.TempDir(), "PR.md")
	var prOut, prErr bytes.Buffer
	prBody, err := svc.RunPR(ctx, gitflow.PROptions{
		WorkDir:    repoDir,
		OutputFile: prOutFile,
		PrintRaw:   true,
		Out:        &prOut,
		ErrOut:     &prErr,
	})
	if err != nil {
		t.Fatalf("RunPR() error: %v", err)
	}
	if !strings.Contains(prBody, "Implements ENG-42 service.") {
		t.Errorf("unexpected PR body: %q", prBody)
	}
	joinedPRArgs := strings.Join(mockProc.lastArgs, " ")
	if !strings.Contains(joinedPRArgs, "Repository Official PR Template:") || !strings.Contains(joinedPRArgs, "ENG-42") {
		t.Errorf("expected repo PR template and ENG-42 in PR prompt:\n%s", joinedPRArgs)
	}
	savedPR, err := os.ReadFile(prOutFile)
	if err != nil {
		t.Fatalf("reading saved PR file: %v", err)
	}
	if !strings.Contains(string(savedPR), "Implements ENG-42 service.") {
		t.Errorf("saved PR file mismatch: %q", string(savedPR))
	}
}
