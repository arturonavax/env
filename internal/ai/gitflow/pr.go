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
)

// PROptions specifies parameters for the AI Pull Request description generator.
type PROptions struct {
	WorkDir       string
	Base          string
	Head          string
	Backend       string
	Model         string
	Template      string
	TaskID        string
	OutputFile    string
	CopyClipboard bool
	PrintRaw      bool
	DryRun        bool
	ListTemplates bool
	Interactive   bool
	In            io.Reader
	Out           io.Writer
	ErrOut        io.Writer
}

// DetectBaseBranch resolves the target base branch for Pull Request comparison.
func (s *Service) DetectBaseBranch(ctx context.Context, dir, explicitBase string) string {
	if strings.TrimSpace(explicitBase) != "" {
		return strings.TrimSpace(explicitBase)
	}
	if strings.TrimSpace(s.cfg.PR.DefaultBase) != "" {
		return strings.TrimSpace(s.cfg.PR.DefaultBase)
	}

	curr := s.CurrentBranch(ctx, dir)
	if curr == "main" || curr == "master" {
		if upstream, err := s.gitRun(ctx, dir, "rev-parse", "--abbrev-ref", "@{upstream}"); err == nil {
			if u := strings.TrimSpace(upstream); u != "" {
				return u
			}
		}
	}

	if symRef, err := s.gitRun(ctx, dir, "symbolic-ref", "--quiet", "refs/remotes/origin/HEAD"); err == nil {
		ref := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(symRef), "refs/remotes/"))
		if ref != "" && strings.TrimPrefix(ref, "origin/") != curr {
			return ref
		}
	}

	for _, cand := range []string{"main", "master", "develop", "dev"} {
		if cand == curr {
			continue
		}
		if _, err := s.gitRun(ctx, dir, "rev-parse", "--verify", "--quiet", "refs/remotes/origin/"+cand); err == nil {
			return "origin/" + cand
		}
		if _, err := s.gitRun(ctx, dir, "rev-parse", "--verify", "--quiet", "refs/heads/"+cand); err == nil {
			return cand
		}
	}

	if curr == "main" {
		return "origin/main"
	}
	return "main"
}

// DiscoverRepoPRTemplate searches well-known GitHub/GitLab repository Pull Request template paths.
func (s *Service) DiscoverRepoPRTemplate(ctx context.Context, dir string) (rawTemplate, source string) {
	root, err := s.RepoRoot(ctx, dir)
	if err != nil {
		return "", ""
	}

	candidates := []string{
		filepath.Join(".github", "pull_request_template.md"),
		filepath.Join(".github", "PULL_REQUEST_TEMPLATE.md"),
		filepath.Join(".github", "PULL_REQUEST_TEMPLATE", "pull_request_template.md"),
		filepath.Join(".github", "PULL_REQUEST_TEMPLATE", "default.md"),
		"pull_request_template.md",
		"PULL_REQUEST_TEMPLATE.md",
		filepath.Join("docs", "pull_request_template.md"),
		filepath.Join("docs", "PULL_REQUEST_TEMPLATE.md"),
		filepath.Join(".gitlab", "merge_request_templates", "Default.md"),
		filepath.Join(".gitlab", "merge_request_templates", "default.md"),
	}

	for _, rel := range candidates {
		full := filepath.Join(root, rel)
		if data, readErr := os.ReadFile(full); readErr == nil {
			if trimmed := strings.TrimSpace(string(data)); trimmed != "" {
				return trimmed, "repo:" + rel
			}
		}
	}

	dirCandidates := []string{
		filepath.Join(root, ".github", "PULL_REQUEST_TEMPLATE"),
		filepath.Join(root, ".github", "pull_request_template"),
		filepath.Join(root, ".gitlab", "merge_request_templates"),
	}
	for _, d := range dirCandidates {
		entries, readErr := os.ReadDir(d)
		if readErr != nil {
			continue
		}
		names := make([]string, 0, len(entries))
		for _, e := range entries {
			if !e.IsDir() && !strings.HasPrefix(e.Name(), ".") &&
				(strings.HasSuffix(e.Name(), ".md") || strings.HasSuffix(e.Name(), ".txt")) {
				names = append(names, e.Name())
			}
		}
		sort.Strings(names)
		for _, name := range names {
			full := filepath.Join(d, name)
			if data, fErr := os.ReadFile(full); fErr == nil {
				if trimmed := strings.TrimSpace(string(data)); trimmed != "" {
					rel, _ := filepath.Rel(root, full)
					return trimmed, "repo:" + rel
				}
			}
		}
	}

	return "", ""
}

func wrapRepoPRTemplate(rawTpl, source string) string {
	if strings.Contains(rawTpl, "{diff}") && (strings.Contains(rawTpl, "{commits}") || strings.Contains(rawTpl, "{branch}")) {
		return rawTpl
	}
	return fmt.Sprintf(`You are an expert developer and technical lead writing a Pull Request (PR) description in GitHub Flavored Markdown.
Analyze the following branch commits, changed files, and git diff. Fill out the repository's official Pull Request template (%s) completely, accurately, and professionally.

Context:
- Current Branch: {branch}
- Target Base Branch: {base}
- Task / Issue ID: {task_id}
- Commits on Branch:
{commits}
- Modified Files:
{files}
- Diff Summary:
{stat}

Repository Official PR Template:
--------------------------------------------------------------------------------
{repo_template}
--------------------------------------------------------------------------------

Instructions:
1. Maintain every section, heading, and table from the repository template above.
2. Fill in all details thoroughly based on the branch changes, commits, and diff.
3. Check relevant checklist boxes with [x] (and leave non-applicable ones with [ ] or explain).
4. If a Task or Issue ID is provided ({task_id}), reference it appropriately.
5. Write in clean, professional English.
6. Do NOT wrap the entire response in a top-level `+"```markdown"+` fence. Output ONLY the filled-in Pull Request Markdown body.

Git Branch Diff:
{diff}`, source)
}

// ResolvePRTemplate selects the PR prompt template, its display name, and any discovered raw repo template.
func (s *Service) ResolvePRTemplate(ctx context.Context, dir, requested string) (promptTpl, tplName, rawRepoTpl string) {
	templates := s.cfg.EffectivePRTemplates()
	rawRepo, repoSrc := s.DiscoverRepoPRTemplate(ctx, dir)

	req := strings.TrimSpace(requested)
	if req != "" {
		switch strings.ToLower(req) {
		case "repo", "git", "github", "pr_template":
			if rawRepo != "" {
				return wrapRepoPRTemplate(rawRepo, repoSrc), repoSrc, rawRepo
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
			return wrapRepoPRTemplate(raw, base), base, raw
		}
	}

	if s.cfg.PR.AutoDetectRepoTemplate && rawRepo != "" {
		return wrapRepoPRTemplate(rawRepo, repoSrc), repoSrc, rawRepo
	}

	defName := strings.TrimSpace(s.cfg.PR.DefaultTemplate)
	if defName == "" {
		defName = "standard"
	}
	if t, ok := templates[defName]; ok {
		return t, defName, rawRepo
	}
	return config.PRPromptStandard, "standard", rawRepo
}

// BuildPRPrompt gathers git commits, stat, modified files, and bounded diff to construct the PR prompt.
func (s *Service) BuildPRPrompt(ctx context.Context, opts PROptions) (prompt, baseBranch, tplName, taskID string, err error) {
	if _, err := s.RepoRoot(ctx, opts.WorkDir); err != nil {
		return "", "", "", "", err
	}

	currBranch := s.CurrentBranch(ctx, opts.WorkDir)
	base := s.DetectBaseBranch(ctx, opts.WorkDir, opts.Base)
	head := strings.TrimSpace(opts.Head)
	if head == "" {
		head = "HEAD"
	}

	resolvedTask := strings.TrimSpace(opts.TaskID)
	if resolvedTask == "" {
		resolvedTask = strings.TrimSpace(s.cfg.PR.TaskID)
	}
	if resolvedTask == "" {
		resolvedTask = ExtractTaskID(currBranch)
	}

	commitsOut, _ := s.gitRun(ctx, opts.WorkDir, "log", "--no-merges", "--format=- %s (%h)", base+".."+head)
	commitsStr := strings.TrimSpace(commitsOut)
	statOut, _ := s.gitRun(ctx, opts.WorkDir, "diff", "--stat", base+"..."+head)
	filesOut, _ := s.gitRun(ctx, opts.WorkDir, "diff", "--name-only", base+"..."+head)
	diffOut, diffErr := s.gitRun(ctx, opts.WorkDir, "diff", base+"..."+head)
	if diffErr != nil {
		// Fallback to two-dot diff when merge-base does not exist in shallow histories.
		diffOut, diffErr = s.gitRun(ctx, opts.WorkDir, "diff", base+".."+head)
	}
	if diffErr != nil || (strings.TrimSpace(diffOut) == "" && commitsStr == "") {
		return "", base, "", resolvedTask, fmt.Errorf("no commits or changes found between base %q and %q", base, head)
	}

	maxLen := s.cfg.PR.MaxDiffLength
	if maxLen <= 0 {
		maxLen = 25000
	}
	boundedDiff := diffOut
	if len(boundedDiff) > maxLen {
		boundedDiff = boundedDiff[:maxLen] + "\n... [diff truncated for length]"
	}

	filesList := nonEmptyLines(filesOut)
	if len(filesList) > 25 {
		filesList = filesList[:25]
	}
	filesLabel := "None"
	if len(filesList) > 0 {
		filesLabel = strings.Join(filesList, ", ")
	}
	if commitsStr == "" {
		commitsStr = "(No individual commit logs)"
	}
	statStr := strings.TrimSpace(statOut)
	if statStr == "" {
		statStr = "(No diff stat)"
	}

	taskRef := ""
	if resolvedTask != "" {
		taskRef = "- **Related Issue / Ticket**: " + resolvedTask
	}

	tplStr, resolvedTplName, rawRepoTpl := s.ResolvePRTemplate(ctx, opts.WorkDir, opts.Template)
	replacements := map[string]string{
		"branch":        currBranch,
		"base":          base,
		"head":          head,
		"task_id":       resolvedTask,
		"issue_id":      resolvedTask,
		"ticket_id":     resolvedTask,
		"ticket":        resolvedTask,
		"task_ref":      taskRef,
		"commits":       commitsStr,
		"files":         filesLabel,
		"stat":          statStr,
		"diff":          boundedDiff,
		"model":         strings.TrimSpace(opts.Model),
		"repo_template": rawRepoTpl,
	}

	return interpolateTemplate(tplStr, replacements), base, resolvedTplName, resolvedTask, nil
}

// RunPR executes the Pull Request description generation workflow.
func (s *Service) RunPR(ctx context.Context, opts PROptions) (string, error) {
	out := opts.Out
	if out == nil {
		out = io.Discard
	}
	errOut := opts.ErrOut
	if errOut == nil {
		errOut = io.Discard
	}

	if opts.ListTemplates {
		s.printPRTemplates(ctx, opts.WorkDir, out)
		return "", nil
	}

	prompt, baseBranch, tplName, taskID, err := s.BuildPRPrompt(ctx, opts)
	if err != nil {
		return "", err
	}

	backend := strings.TrimSpace(opts.Backend)
	if backend == "" {
		backend = s.cfg.PR.DefaultBackend
	}
	if backend == "" {
		backend = s.cfg.DefaultCommitBackend
	}
	model := strings.TrimSpace(opts.Model)
	if model == "" {
		model = s.cfg.PR.Model
	}
	effort := s.cfg.PR.Effort
	if effort == "" {
		effort = "low"
	}

	if !opts.PrintRaw {
		fmt.Fprintf(errOut, "==> Generating PR description [backend: %s, base: %s, template: %s", backend, baseBranch, tplName)
		if taskID != "" {
			fmt.Fprintf(errOut, ", task: %s", taskID)
		}
		fmt.Fprintln(errOut, "]...")
	}

	cleanMD := true
	generate := func() (string, error) {
		md, _, qErr := s.registry.Query(ctx, backend, harness.Request{
			Prompt:        prompt,
			Model:         model,
			Effort:        effort,
			Mode:          "chat",
			OutputFormat:  "text",
			Timeout:       60 * time.Second,
			CleanMarkdown: &cleanMD,
		})
		return strings.TrimSpace(md), qErr
	}

	prMarkdown, err := generate()
	if err != nil {
		return "", fmt.Errorf("generating PR description: %w", err)
	}

	if opts.OutputFile != "" {
		if err := os.MkdirAll(filepath.Dir(opts.OutputFile), 0o755); err != nil {
			return prMarkdown, fmt.Errorf("creating output directory: %w", err)
		}
		if err := os.WriteFile(opts.OutputFile, []byte(prMarkdown+"\n"), 0o644); err != nil {
			return prMarkdown, fmt.Errorf("writing PR output file %q: %w", opts.OutputFile, err)
		}
		if !opts.PrintRaw {
			fmt.Fprintf(errOut, "PR description written to %s\n", opts.OutputFile)
		}
	}

	if opts.CopyClipboard {
		_, clipErr := CopyToClipboardWithOptions(ctx, s.runner, prMarkdown, ClipboardOptions{
			FallbackWriter: errOut,
		})
		if clipErr != nil && !opts.PrintRaw {
			fmt.Fprintf(errOut, "Clipboard warning: %v\n", clipErr)
		}
	}

	if opts.PrintRaw || opts.DryRun || !opts.Interactive || opts.In == nil {
		fmt.Fprintln(out, prMarkdown)
		return prMarkdown, nil
	}

	reader := bufio.NewReader(opts.In)
	_, hasGHErr := exec.LookPath("gh")
	hasGH := hasGHErr == nil

	for {
		fmt.Fprintf(out, "\n--- Proposed Pull Request Description (%s) ---\n%s\n----------------------------------------------\n", tplName, prMarkdown)
		if hasGH {
			fmt.Fprint(out, "Action: [a] Accept / [c] Copy / [s] Save / [e] Edit / [r] Regenerate / [p] gh pr create / [q] Quit: ")
		} else {
			fmt.Fprint(out, "Action: [a] Accept / [c] Copy / [s] Save / [e] Edit / [r] Regenerate / [q] Quit: ")
		}

		line, readErr := reader.ReadString('\n')
		if readErr != nil && !errors.Is(readErr, io.EOF) {
			return prMarkdown, readErr
		}
		choice := strings.ToLower(strings.TrimSpace(line))

		switch choice {
		case "", "a", "accept", "y":
			fmt.Fprintln(out, prMarkdown)
			return prMarkdown, nil

		case "c", "copy":
			method, clipErr := CopyToClipboardWithOptions(ctx, s.runner, prMarkdown, ClipboardOptions{
				FallbackWriter: errOut,
			})
			if clipErr != nil {
				fmt.Fprintf(errOut, "Clipboard unavailable: %v\n", clipErr)
			} else {
				fmt.Fprintf(out, "Copied PR description via %s.\n", method)
			}

		case "s", "save":
			dest := opts.OutputFile
			if dest == "" {
				dest = "PR_DESCRIPTION.md"
			}
			if wErr := os.WriteFile(dest, []byte(prMarkdown+"\n"), 0o644); wErr != nil {
				fmt.Fprintf(errOut, "Failed to write %s: %v\n", dest, wErr)
			} else {
				fmt.Fprintf(out, "Saved PR description to %s.\n", dest)
			}

		case "e", "edit":
			edited, editErr := editInExternalEditor(ctx, prMarkdown, ".pr_description.md")
			if editErr != nil {
				fmt.Fprintf(errOut, "Editor error: %v\n", editErr)
				continue
			}
			if strings.TrimSpace(edited) != "" {
				prMarkdown = strings.TrimSpace(edited)
			}

		case "r", "regen", "regenerate":
			regenerated, rErr := generate()
			if rErr != nil {
				fmt.Fprintf(errOut, "Regeneration failed: %v\n", rErr)
				continue
			}
			prMarkdown = regenerated

		case "p", "gh":
			if !hasGH {
				continue
			}
			if ghErr := s.createGitHubPR(ctx, opts.WorkDir, baseBranch, prMarkdown); ghErr != nil {
				fmt.Fprintf(errOut, "gh pr create failed: %v\n", ghErr)
				continue
			}
			fmt.Fprintln(out, "Pull Request created via GitHub CLI.")
			return prMarkdown, nil

		case "q", "quit", "n", "cancel":
			return prMarkdown, nil

		default:
			if errors.Is(readErr, io.EOF) {
				return prMarkdown, nil
			}
		}
	}
}

func (s *Service) createGitHubPR(ctx context.Context, dir, baseBranch, body string) error {
	title := ""
	for _, line := range strings.Split(body, "\n") {
		clean := strings.TrimSpace(strings.TrimLeft(line, "#"))
		if clean != "" {
			title = clean
			break
		}
	}
	if title == "" {
		curr := s.CurrentBranch(ctx, dir)
		title = "feat: updates from " + curr
	}

	tf, err := os.CreateTemp("", "aitools-pr-body-*.md")
	if err != nil {
		return err
	}
	bodyPath := tf.Name()
	defer os.Remove(bodyPath)

	if _, err := tf.WriteString(body); err != nil {
		_ = tf.Close()
		return err
	}
	if err := tf.Close(); err != nil {
		return err
	}

	_, err = s.runner.RunWithOptions(
		ctx,
		dir,
		nil,
		"gh",
		"pr",
		"create",
		"--base",
		baseBranch,
		"--title",
		title,
		"--body-file",
		bodyPath,
	)
	return err
}

func (s *Service) printPRTemplates(ctx context.Context, dir string, out io.Writer) {
	fmt.Fprintln(out, "Available Pull Request (PR) Description Templates:")
	_, repoSrc := s.DiscoverRepoPRTemplate(ctx, dir)
	if repoSrc != "" {
		fmt.Fprintf(out, "  - %-16s [%s] Discovered repository PR template\n", "repo", repoSrc)
	}
	templates := s.cfg.EffectivePRTemplates()
	names := make([]string, 0, len(templates))
	for k := range templates {
		names = append(names, k)
	}
	sort.Strings(names)
	for _, name := range names {
		marker := ""
		if name == s.cfg.PR.DefaultTemplate {
			marker = " (default)"
		}
		fmt.Fprintf(out, "  - %s%s\n", name, marker)
	}
}
