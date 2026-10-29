package pipeline

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/arturonavax/env/internal/ai/config"
	"github.com/arturonavax/env/internal/ai/harness"
	"github.com/arturonavax/env/internal/security"
)

const defaultMaxPipelineInput = 25000

// RunOptions defines inputs and streams for executing a declarative AI pipeline command.
type RunOptions struct {
	Name       string
	Args       []string
	WorkDir    string
	Stdin      io.Reader
	StdinIsTTY bool
	Out        io.Writer
	ErrOut     io.Writer
}

// CommandSummary describes a built-in or custom command for catalog display.
type CommandSummary struct {
	Name        string `json:"name"`
	Kind        string `json:"kind"`
	InputType   string `json:"input_type"`
	Backend     string `json:"backend"`
	Description string `json:"description"`
}

// Runner executes declarative custom commands configured in cfg.Commands.
type Runner struct {
	cfg      *config.Config
	registry *harness.Registry
	proc     *security.Runner
}

// NewRunner constructs a declarative pipeline Runner.
func NewRunner(cfg *config.Config, registry *harness.Registry, proc *security.Runner) *Runner {
	if cfg == nil {
		cfg, _ = config.DefaultConfig()
	}
	if registry == nil {
		registry = harness.NewRegistry(cfg)
	}
	if proc == nil {
		proc = security.NewRunner(30 * time.Second)
	}
	return &Runner{
		cfg:      cfg,
		registry: registry,
		proc:     proc,
	}
}

// ListCustomCommands returns sorted metadata for all declarative commands in cfg.Commands.
func (r *Runner) ListCustomCommands() []CommandSummary {
	names := make([]string, 0, len(r.cfg.Commands))
	for k := range r.cfg.Commands {
		names = append(names, k)
	}
	sort.Strings(names)

	out := make([]CommandSummary, 0, len(names))
	for _, name := range names {
		spec := r.cfg.Commands[name]
		backend := spec.Backend
		if backend == "" {
			backend = r.cfg.DefaultCommitBackend
		}
		inType := spec.InputType
		if inType == "" {
			inType = "args"
		}
		desc := spec.Description
		if desc == "" {
			desc = "Declarative AI pipeline: " + name
		}
		out = append(out, CommandSummary{
			Name:        name,
			Kind:        "declarative",
			InputType:   inType,
			Backend:     backend,
			Description: desc,
		})
	}
	return out
}

// Execute runs the named declarative command from cfg.Commands.
func (r *Runner) Execute(ctx context.Context, opts RunOptions) (string, error) {
	cmdDef, ok := r.cfg.Commands[opts.Name]
	if !ok {
		names := make([]string, 0, len(r.cfg.Commands))
		for k := range r.cfg.Commands {
			names = append(names, k)
		}
		sort.Strings(names)
		return "", fmt.Errorf("unknown declarative command %q (available: %s)", opts.Name, strings.Join(names, ", "))
	}

	out := opts.Out
	if out == nil {
		out = io.Discard
	}
	errOut := opts.ErrOut
	if errOut == nil {
		errOut = io.Discard
	}

	inputText, fileArg, err := r.gatherInput(ctx, opts, cmdDef)
	if err != nil {
		return "", err
	}

	maxLen := cmdDef.MaxInputLength
	if maxLen <= 0 {
		maxLen = defaultMaxPipelineInput
	}
	if len(inputText) > maxLen {
		inputText = inputText[:maxLen] + "\n... [input truncated for length]"
	}

	branch := r.currentBranch(ctx, opts.WorkDir)
	argsJoined := strings.Join(opts.Args, " ")

	tpl := cmdDef.PromptTemplate
	if tpl == "" {
		tpl = "{input}"
	}

	prompt := tpl
	hasInputPlaceholder := strings.Contains(tpl, "{input}")
	replacements := map[string]string{
		"input":  inputText,
		"args":   argsJoined,
		"file":   fileArg,
		"branch": branch,
	}
	for k, v := range replacements {
		prompt = strings.ReplaceAll(prompt, "{"+k+"}", v)
	}
	if !hasInputPlaceholder && strings.TrimSpace(inputText) != "" {
		prompt = strings.TrimSpace(prompt) + "\n\n" + inputText
	}

	backend := strings.TrimSpace(cmdDef.Backend)
	if backend == "" {
		backend = r.cfg.DefaultCommitBackend
	}
	effort := strings.TrimSpace(cmdDef.Effort)
	if effort == "" {
		effort = "low"
	}
	mode := strings.TrimSpace(cmdDef.Mode)
	if mode == "" {
		mode = "agent"
	}
	outFmt := strings.TrimSpace(cmdDef.OutputFormat)
	if outFmt == "" {
		outFmt = "text"
	}
	timeoutSec := cmdDef.TimeoutSec
	if timeoutSec <= 0 {
		timeoutSec = 60
	}

	desc := cmdDef.Description
	if desc == "" {
		desc = opts.Name
	}
	fmt.Fprintf(errOut, "==> aitools [%s]: %s (backend: %s)\n", opts.Name, desc, backend)

	cleanMD := false
	resp, _, err := r.registry.Query(ctx, backend, harness.Request{
		Prompt:        prompt,
		Model:         cmdDef.Model,
		Effort:        effort,
		Mode:          mode,
		OutputFormat:  outFmt,
		Timeout:       time.Duration(timeoutSec) * time.Second,
		CleanMarkdown: &cleanMD,
	})
	if err != nil {
		return "", err
	}

	fmt.Fprintln(out, resp)
	return resp, nil
}

func (r *Runner) gatherInput(ctx context.Context, opts RunOptions, def config.CommandConfig) (inputText, fileArg string, err error) {
	inType := strings.ToLower(strings.TrimSpace(def.InputType))
	if inType == "" {
		inType = "args"
	}

	switch inType {
	case "args":
		if len(opts.Args) == 0 {
			return "", "", fmt.Errorf("command %q requires arguments", opts.Name)
		}
		return strings.Join(opts.Args, " "), "", nil

	case "stdin":
		if opts.Stdin == nil {
			return "", "", fmt.Errorf("command %q requires standard input", opts.Name)
		}
		b, readErr := io.ReadAll(opts.Stdin)
		if readErr != nil {
			return "", "", fmt.Errorf("reading stdin: %w", readErr)
		}
		return string(b), "", nil

	case "stdin_or_args":
		if !opts.StdinIsTTY && opts.Stdin != nil {
			b, readErr := io.ReadAll(opts.Stdin)
			if readErr == nil && strings.TrimSpace(string(b)) != "" {
				return string(b), "", nil
			}
		}
		if len(opts.Args) > 0 {
			return strings.Join(opts.Args, " "), "", nil
		}
		if opts.Stdin != nil {
			b, readErr := io.ReadAll(opts.Stdin)
			if readErr != nil {
				return "", "", fmt.Errorf("reading stdin: %w", readErr)
			}
			if strings.TrimSpace(string(b)) != "" {
				return string(b), "", nil
			}
		}
		return "", "", fmt.Errorf("command %q requires input via stdin or arguments", opts.Name)

	case "file":
		if len(opts.Args) == 0 {
			return "", "", fmt.Errorf("command %q requires a file path argument", opts.Name)
		}
		fp := opts.Args[0]
		b, readErr := os.ReadFile(fp)
		if readErr != nil {
			return "", fp, fmt.Errorf("reading file %q: %w", fp, readErr)
		}
		return string(b), fp, nil

	case "git_staged", "git_staged_diff":
		diff, gitErr := r.proc.RunWithOptions(ctx, opts.WorkDir, nil, "git", "diff", "--cached")
		if gitErr != nil || strings.TrimSpace(diff) == "" {
			return "", "", errors.New("no staged git diff found (stage files with 'git add')")
		}
		return diff, "", nil

	case "git_diff":
		diff, gitErr := r.proc.RunWithOptions(ctx, opts.WorkDir, nil, "git", "diff", "HEAD")
		if gitErr != nil || strings.TrimSpace(diff) == "" {
			diff, gitErr = r.proc.RunWithOptions(ctx, opts.WorkDir, nil, "git", "diff")
		}
		if gitErr != nil || strings.TrimSpace(diff) == "" {
			return "", "", errors.New("no git diff found in current repository")
		}
		return diff, "", nil

	case "git_log":
		logOut, gitErr := r.proc.RunWithOptions(ctx, opts.WorkDir, nil, "git", "log", "-n", "25", "--no-merges", "--format=- %s (%h)")
		if gitErr != nil || strings.TrimSpace(logOut) == "" {
			return "", "", errors.New("no git commit history found in current repository")
		}
		return strings.TrimSpace(logOut), "", nil

	case "shell":
		shellCmd := strings.TrimSpace(def.ShellCommand)
		if shellCmd == "" {
			return "", "", nil
		}
		violations := security.ValidateScriptGuardrails(shellCmd)
		if len(violations) > 0 {
			return "", "", fmt.Errorf("shell_command blocked by guardrail %s: %s", violations[0].RuleID, violations[0].Description)
		}
		shellOut, shErr := r.proc.RunWithOptions(ctx, opts.WorkDir, nil, "sh", "-c", shellCmd)
		if shErr != nil {
			return "", "", fmt.Errorf("executing pipeline shell_command: %w", shErr)
		}
		return strings.TrimSpace(shellOut), "", nil

	case "none":
		return "", "", nil

	default:
		if len(opts.Args) > 0 {
			return strings.Join(opts.Args, " "), "", nil
		}
		if opts.Stdin != nil && !opts.StdinIsTTY {
			b, _ := io.ReadAll(opts.Stdin)
			return string(b), "", nil
		}
		return "", "", nil
	}
}

func (r *Runner) currentBranch(ctx context.Context, dir string) string {
	out, err := r.proc.RunWithOptions(ctx, dir, nil, "git", "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return ""
	}
	b := strings.TrimSpace(out)
	if b == "HEAD" {
		return ""
	}
	return b
}
