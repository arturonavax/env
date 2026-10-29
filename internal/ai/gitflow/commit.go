package gitflow

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/arturonavax/env/internal/ai/config"
	"github.com/arturonavax/env/internal/ai/harness"
	"github.com/arturonavax/env/internal/security"
)

// CommitOptions specifies parameters for the AI conventional commit workflow.
type CommitOptions struct {
	WorkDir       string
	StageAll      bool
	Backend       string
	Model         string
	Template      string
	TaskID        string
	Scope         string
	DryRun        bool
	ListTemplates bool
	Interactive   bool
	In            io.Reader
	Out           io.Writer
	ErrOut        io.Writer
}

// Service coordinates AI-assisted Git commit and Pull Request workflows.
type Service struct {
	cfg      *config.Config
	registry *harness.Registry
	runner   *security.Runner
}

// NewService constructs a Git workflow Service.
func NewService(cfg *config.Config, registry *harness.Registry, runner *security.Runner) *Service {
	if cfg == nil {
		cfg, _ = config.DefaultConfig()
	}
	if registry == nil {
		registry = harness.NewRegistry(cfg)
	}
	if runner == nil {
		runner = security.NewRunner(30 * time.Second)
	}
	return &Service{
		cfg:      cfg,
		registry: registry,
		runner:   runner,
	}
}

func (s *Service) gitRun(ctx context.Context, dir string, args ...string) (string, error) {
	return s.runner.RunWithOptions(ctx, dir, nil, "git", args...)
}

// CurrentBranch returns the active branch name in dir (or empty string if detached/outside git).
func (s *Service) CurrentBranch(ctx context.Context, dir string) string {
	out, err := s.gitRun(ctx, dir, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return ""
	}
	b := strings.TrimSpace(out)
	if b == "HEAD" {
		return ""
	}
	return b
}

// RepoRoot returns the git top-level directory for dir.
func (s *Service) RepoRoot(ctx context.Context, dir string) (string, error) {
	out, err := s.gitRun(ctx, dir, "rev-parse", "--show-toplevel")
	if err != nil {
		return "", fmt.Errorf("not inside a git repository: %w", err)
	}
	return strings.TrimSpace(out), nil
}

// DiscoverRepoCommitTemplate searches git config commit.template and standard repository files.
func (s *Service) DiscoverRepoCommitTemplate(ctx context.Context, dir string) (rawTemplate, source string) {
	root, err := s.RepoRoot(ctx, dir)
	if err != nil {
		return "", ""
	}

	if cfgTpl, cfgErr := s.gitRun(ctx, dir, "config", "--get", "commit.template"); cfgErr == nil {
		p := strings.TrimSpace(cfgTpl)
		if p != "" {
			p = expandUserHome(p)
			if !filepath.IsAbs(p) {
				p = filepath.Join(root, p)
			}
			if data, readErr := os.ReadFile(p); readErr == nil && strings.TrimSpace(string(data)) != "" {
				return strings.TrimSpace(string(data)), "git-config:" + filepath.Base(p)
			}
		}
	}

	candidates := []string{
		".gitmessage",
		".gitmessage.txt",
		".gitmessage.md",
		".commit-template",
		".commit_template",
		".commit-template.txt",
		filepath.Join(".github", "commit_template.md"),
		filepath.Join(".github", "COMMIT_TEMPLATE.md"),
		filepath.Join(".github", "commit-template.md"),
		filepath.Join(".github", "COMMIT-TEMPLATE.md"),
		filepath.Join(".github", "commit_template.txt"),
	}

	for _, rel := range candidates {
		full := filepath.Join(root, rel)
		if data, readErr := os.ReadFile(full); readErr == nil {
			trimmed := strings.TrimSpace(string(data))
			if trimmed != "" {
				return trimmed, "repo:" + rel
			}
		}
	}
	return "", ""
}

func wrapRepoCommitTemplate(rawTpl, source string) string {
	if strings.Contains(rawTpl, "{diff}") {
		return rawTpl
	}
	return fmt.Sprintf(`You are an expert developer creating a git commit message for this repository.
Analyze the following git diff and output a single, high-quality commit message that strictly adheres to this repository's commit template (%s).

Repository Commit Template & Guidelines:
---
{repo_template}
---

Context:
- Current Branch: {branch}
- Task / Issue ID: {task_id}
- Scope Override: {scope}
- Modified Files: {files}

Requirements:
1. Follow the exact structure, casing, and guidelines shown in the repository template above.
2. Replace any task or issue placeholders with the actual Task ID ({task_id}).
3. Omit template comment lines starting with '#'.
4. Keep the summary line under 72 characters.
5. Output in English.
6. Do NOT include markdown code blocks (no `+"```"+` backticks), quotes, or explanation.
7. Do NOT add "Co-Authored-By", AI attribution, or metadata.
8. Output ONLY the raw commit message.

Git Diff:
{diff}`, source)
}

// ResolveCommitTemplate selects the commit prompt template, its display name, and any discovered raw repo template.
func (s *Service) ResolveCommitTemplate(ctx context.Context, dir, requested string) (promptTpl, tplName, rawRepoTpl string) {
	templates := s.cfg.EffectiveCommitTemplates()
	rawRepo, repoSrc := s.DiscoverRepoCommitTemplate(ctx, dir)

	req := strings.TrimSpace(requested)
	if req != "" {
		switch strings.ToLower(req) {
		case "repo", "git", "gitmessage":
			if rawRepo != "" {
				return wrapRepoCommitTemplate(rawRepo, repoSrc), repoSrc, rawRepo
			}
		}
		if t, ok := templates[req]; ok {
			return t, req, rawRepo
		}
		if strings.Contains(req, "{diff}") {
			return req, "custom-inline", rawRepo
		}
		if data, err := os.ReadFile(req); err == nil && strings.TrimSpace(string(data)) != "" {
			raw := strings.TrimSpace(string(data))
			base := filepath.Base(req)
			return wrapRepoCommitTemplate(raw, base), base, raw
		}
	}

	defName := strings.TrimSpace(s.cfg.Commit.DefaultTemplate)
	if defName == "" {
		defName = "conventional"
	}

	if defName == "repo" && rawRepo != "" {
		return wrapRepoCommitTemplate(rawRepo, repoSrc), repoSrc, rawRepo
	}
	if s.cfg.Commit.AutoDetectRepoTemplate && (defName == "conventional" || defName == "auto") && rawRepo != "" {
		return wrapRepoCommitTemplate(rawRepo, repoSrc), repoSrc, rawRepo
	}
	if t, ok := templates[defName]; ok {
		return t, defName, rawRepo
	}
	if s.cfg.Commit.PromptTemplate != "" {
		return s.cfg.Commit.PromptTemplate, "config-prompt", rawRepo
	}
	return config.CommitPromptConventional, "conventional", rawRepo
}

// BuildCommitPrompt constructs the interpolated LLM prompt for staged changes in dir.
func (s *Service) BuildCommitPrompt(
	ctx context.Context,
	dir string,
	diffStat string,
	rawDiff string,
	opts CommitOptions,
) (prompt, templateName, taskID string) {
	maxLen := s.cfg.Commit.MaxDiffLength
	if maxLen <= 0 {
		maxLen = 15000
	}

	boundedDiff := rawDiff
	if len(boundedDiff) > maxLen {
		boundedDiff = boundedDiff[:maxLen] + "\n... [diff truncated for length]"
	}

	combinedDiff := boundedDiff
	if strings.TrimSpace(diffStat) != "" {
		combinedDiff = "Diff Stat:\n" + strings.TrimSpace(diffStat) + "\n\nPatch:\n" + boundedDiff
	}

	branch := s.CurrentBranch(ctx, dir)
	resolvedTask := strings.TrimSpace(opts.TaskID)
	if resolvedTask == "" {
		resolvedTask = strings.TrimSpace(s.cfg.Commit.TaskID)
	}
	if resolvedTask == "" {
		resolvedTask = ExtractTaskID(branch)
	}

	filesOut, _ := s.gitRun(ctx, dir, "diff", "--cached", "--name-only")
	filesList := nonEmptyLines(filesOut)
	if len(filesList) > 15 {
		filesList = filesList[:15]
	}

	tplStr, tplName, rawRepoTpl := s.ResolveCommitTemplate(ctx, dir, opts.Template)

	replacements := map[string]string{
		"diff":          combinedDiff,
		"stat":          strings.TrimSpace(diffStat),
		"branch":        branch,
		"task_id":       resolvedTask,
		"issue_id":      resolvedTask,
		"ticket_id":     resolvedTask,
		"ticket":        resolvedTask,
		"jira_id":       resolvedTask,
		"files":         strings.Join(filesList, ", "),
		"scope":         strings.TrimSpace(opts.Scope),
		"model":         strings.TrimSpace(opts.Model),
		"repo_template": rawRepoTpl,
	}

	return interpolateTemplate(tplStr, replacements), tplName, resolvedTask
}

// RunCommit executes the complete commit generation and optional interactive confirmation workflow.
func (s *Service) RunCommit(ctx context.Context, opts CommitOptions) (string, error) {
	out := opts.Out
	if out == nil {
		out = io.Discard
	}
	errOut := opts.ErrOut
	if errOut == nil {
		errOut = io.Discard
	}

	if opts.ListTemplates {
		s.printCommitTemplates(ctx, opts.WorkDir, out)
		return "", nil
	}

	if _, err := s.RepoRoot(ctx, opts.WorkDir); err != nil {
		return "", err
	}

	shouldStage := opts.StageAll || s.cfg.Commit.DefaultStageAll
	if shouldStage {
		if _, err := s.gitRun(ctx, opts.WorkDir, "add", "-u"); err != nil {
			return "", fmt.Errorf("staging modified files: %w", err)
		}
	}

	var reader *bufio.Reader
	if opts.In != nil {
		reader = bufio.NewReader(opts.In)
	}

	stagedDiff, _ := s.gitRun(ctx, opts.WorkDir, "diff", "--cached")
	if strings.TrimSpace(stagedDiff) == "" && opts.Interactive && reader != nil {
		unstaged, _ := s.gitRun(ctx, opts.WorkDir, "diff")
		if strings.TrimSpace(unstaged) != "" {
			fmt.Fprintln(errOut, "No staged changes found, but unstaged modifications exist.")
			fmt.Fprint(errOut, "Stage all tracked modified files (git add -u)? [Y/n]: ")
			line, _ := reader.ReadString('\n')
			choice := strings.ToLower(strings.TrimSpace(line))
			if choice == "" || choice == "y" || choice == "yes" {
				_, _ = s.gitRun(ctx, opts.WorkDir, "add", "-u")
				stagedDiff, _ = s.gitRun(ctx, opts.WorkDir, "diff", "--cached")
			}
		}
	}

	if strings.TrimSpace(stagedDiff) == "" {
		return "", errors.New("no staged changes to commit (working tree clean or nothing staged)")
	}

	diffStat, _ := s.gitRun(ctx, opts.WorkDir, "diff", "--cached", "--stat")
	prompt, tplName, taskID := s.BuildCommitPrompt(ctx, opts.WorkDir, diffStat, stagedDiff, opts)

	backend := strings.TrimSpace(opts.Backend)
	if backend == "" {
		backend = s.cfg.Commit.DefaultBackend
	}
	if backend == "" {
		backend = s.cfg.DefaultCommitBackend
	}
	model := strings.TrimSpace(opts.Model)
	if model == "" {
		model = s.cfg.Commit.Model
	}
	effort := s.cfg.Commit.Effort
	if effort == "" {
		effort = "low"
	}

	fmt.Fprintf(errOut, "==> Generating commit message [backend: %s, template: %s", backend, tplName)
	if taskID != "" {
		fmt.Fprintf(errOut, ", task: %s", taskID)
	}
	fmt.Fprintln(errOut, "]...")

	cleanMD := true
	generate := func() (string, error) {
		msg, _, qErr := s.registry.Query(ctx, backend, harness.Request{
			Prompt:        prompt,
			Model:         model,
			Effort:        effort,
			Mode:          "chat",
			OutputFormat:  "text",
			Timeout:       45 * time.Second,
			CleanMarkdown: &cleanMD,
		})
		return strings.TrimSpace(msg), qErr
	}

	commitMsg, err := generate()
	if err != nil {
		return "", fmt.Errorf("generating commit message: %w", err)
	}

	if opts.DryRun || !opts.Interactive || reader == nil {
		fmt.Fprintln(out, commitMsg)
		return commitMsg, nil
	}

	for {
		fmt.Fprintf(out, "\n--- Proposed Commit Message ---\n%s\n-------------------------------\n", commitMsg)
		fmt.Fprint(out, "Action: [y] Commit / [e] Edit in $EDITOR / [r] Regenerate / [n] Abort: ")

		line, readErr := reader.ReadString('\n')
		if readErr != nil && !errors.Is(readErr, io.EOF) {
			return commitMsg, readErr
		}
		choice := strings.ToLower(strings.TrimSpace(line))

		switch choice {
		case "", "y", "yes", "a", "accept":
			if _, cErr := s.gitRun(ctx, opts.WorkDir, "commit", "-m", commitMsg); cErr != nil {
				return commitMsg, fmt.Errorf("git commit failed: %w", cErr)
			}
			fmt.Fprintln(out, "Successfully committed changes.")
			return commitMsg, nil

		case "e", "edit":
			edited, editErr := editInExternalEditor(ctx, commitMsg, ".gitcommit_msg")
			if editErr != nil {
				fmt.Fprintf(errOut, "Editor error: %v\n", editErr)
				continue
			}
			if strings.TrimSpace(edited) == "" {
				return "", errors.New("empty commit message after editing; commit aborted")
			}
			commitMsg = strings.TrimSpace(edited)

		case "r", "regen", "regenerate":
			regenerated, rErr := generate()
			if rErr != nil {
				fmt.Fprintf(errOut, "Regeneration failed: %v\n", rErr)
				continue
			}
			commitMsg = regenerated

		case "n", "no", "c", "cancel", "q", "quit":
			fmt.Fprintln(errOut, "Commit aborted by user.")
			return commitMsg, nil

		default:
			if errors.Is(readErr, io.EOF) {
				return commitMsg, nil
			}
		}
	}
}

func (s *Service) printCommitTemplates(ctx context.Context, dir string, out io.Writer) {
	fmt.Fprintln(out, "Available Commit Message Templates:")
	_, repoSrc := s.DiscoverRepoCommitTemplate(ctx, dir)
	if repoSrc != "" {
		fmt.Fprintf(out, "  - %-16s [%s] Discovered repository commit template\n", "repo", repoSrc)
	}
	templates := s.cfg.EffectiveCommitTemplates()
	names := make([]string, 0, len(templates))
	for k := range templates {
		if k == "jira" || k == "jira-scope" {
			continue
		}
		names = append(names, k)
	}
	sort.Strings(names)
	for _, name := range names {
		marker := ""
		if name == s.cfg.Commit.DefaultTemplate {
			marker = " (default)"
		}
		fmt.Fprintf(out, "  - %s%s\n", name, marker)
	}
}

func interpolateTemplate(tpl string, vars map[string]string) string {
	res := tpl
	for k, v := range vars {
		res = strings.ReplaceAll(res, "{"+k+"}", v)
	}
	return res
}

func nonEmptyLines(s string) []string {
	raw := strings.Split(s, "\n")
	out := make([]string, 0, len(raw))
	for _, l := range raw {
		trimmed := strings.TrimSpace(l)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func expandUserHome(p string) string {
	if p == "~" || strings.HasPrefix(p, "~/") {
		if h, err := os.UserHomeDir(); err == nil {
			if p == "~" {
				return h
			}
			return filepath.Join(h, p[2:])
		}
	}
	return p
}

func editInExternalEditor(ctx context.Context, initialContent, suffix string) (string, error) {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		if _, err := exec.LookPath("nvim"); err == nil {
			editor = "nvim"
		} else {
			editor = "vi"
		}
	}

	tf, err := os.CreateTemp("", "aitools-*"+suffix)
	if err != nil {
		return "", err
	}
	tmpPath := tf.Name()
	defer os.Remove(tmpPath)

	if _, err := tf.WriteString(initialContent); err != nil {
		_ = tf.Close()
		return "", err
	}
	if err := tf.Close(); err != nil {
		return "", err
	}

	cmd := exec.CommandContext(ctx, editor, tmpPath)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", err
	}

	edited, err := os.ReadFile(tmpPath)
	if err != nil {
		return "", err
	}
	return string(edited), nil
}
