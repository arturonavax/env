package harness

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/arturonavax/env/internal/ai/config"
	"github.com/arturonavax/env/internal/security"
)

const defaultMaxHarnessBytes = 4 * 1024 * 1024 // 4 MB stdout/stderr cap per LLM invocation

// ProcessSpec describes an isolated child process execution request.
type ProcessSpec struct {
	Binary  string
	Args    []string
	Stdin   io.Reader
	Dir     string
	Env     []string
	Timeout time.Duration
}

// ProcessResult holds isolated stdout and stderr captured from a child process.
type ProcessResult struct {
	Stdout string
	Stderr string
}

// ProcessRunner executes external AI CLI binaries with isolated stdout and stderr streams.
type ProcessRunner interface {
	ExecuteProcess(ctx context.Context, spec ProcessSpec) (ProcessResult, error)
}

// DefaultProcessRunner executes subprocesses via os/exec with capped buffers and context timeouts.
type DefaultProcessRunner struct {
	MaxOutputBytes int
}

// NewDefaultProcessRunner constructs a safe DefaultProcessRunner.
func NewDefaultProcessRunner() *DefaultProcessRunner {
	return &DefaultProcessRunner{MaxOutputBytes: defaultMaxHarnessBytes}
}

// ExecuteProcess runs spec.Binary with isolated stdout/stderr buffers and strict timeout enforcement.
func (r *DefaultProcessRunner) ExecuteProcess(ctx context.Context, spec ProcessSpec) (ProcessResult, error) {
	if spec.Binary == "" {
		return ProcessResult{}, errors.New("harness binary cannot be empty")
	}

	timeout := spec.Timeout
	if timeout <= 0 {
		timeout = 60 * time.Second
	}

	execCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(execCtx, spec.Binary, spec.Args...)
	if spec.Dir != "" {
		cmd.Dir = spec.Dir
	}
	if len(spec.Env) > 0 {
		cmd.Env = spec.Env
	}
	if spec.Stdin != nil {
		cmd.Stdin = spec.Stdin
	}

	limit := r.MaxOutputBytes
	if limit <= 0 {
		limit = defaultMaxHarnessBytes
	}
	stdoutBuf := &boundedBuffer{limit: limit}
	stderrBuf := &boundedBuffer{limit: limit}
	cmd.Stdout = stdoutBuf
	cmd.Stderr = stderrBuf

	err := cmd.Run()
	res := ProcessResult{
		Stdout: stdoutBuf.String(),
		Stderr: stderrBuf.String(),
	}

	if errors.Is(execCtx.Err(), context.DeadlineExceeded) {
		return res, fmt.Errorf("harness %q timed out after %v", spec.Binary, timeout)
	}
	if err != nil {
		errDetail := strings.TrimSpace(res.Stderr)
		if errDetail == "" {
			errDetail = strings.TrimSpace(res.Stdout)
		}
		return res, fmt.Errorf("harness %q failed: %w (%s)", spec.Binary, err, security.RedactSensitives(errDetail))
	}
	return res, nil
}

type boundedBuffer struct {
	mu    sync.Mutex
	buf   bytes.Buffer
	limit int
}

func (b *boundedBuffer) Write(p []byte) (int, error) {
	total := len(p)
	b.mu.Lock()
	defer b.mu.Unlock()
	rem := b.limit - b.buf.Len()
	if rem <= 0 {
		return total, nil
	}
	if len(p) > rem {
		_, _ = b.buf.Write(p[:rem])
		return total, nil
	}
	_, _ = b.buf.Write(p)
	return total, nil
}

func (b *boundedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

// Request specifies parameters for an AI harness invocation.
type Request struct {
	Prompt        string
	SystemPrompt  string
	Model         string
	Effort        string
	Mode          string
	OutputFormat  string
	Timeout       time.Duration
	CleanMarkdown *bool
}

// Adapter normalizes input delivery, flag injection, and output sanitization for an AI CLI harness.
type Adapter struct {
	name     string
	spec     config.BackendConfig
	runner   ProcessRunner
	lookPath func(string) (string, error)
}

// NewAdapter constructs an Adapter for the named backend specification.
func NewAdapter(
	name string,
	spec config.BackendConfig,
	runner ProcessRunner,
	lookPath func(string) (string, error),
) *Adapter {
	if len(spec.Command) == 0 {
		spec.Command = []string{name}
	}
	if spec.InputMode == "" {
		spec.InputMode = "arg"
	}
	if spec.OutputMode == "" {
		spec.OutputMode = "stdout"
	}
	if spec.TimeoutSec <= 0 {
		spec.TimeoutSec = 60
	}
	if runner == nil {
		runner = NewDefaultProcessRunner()
	}
	if lookPath == nil {
		lookPath = exec.LookPath
	}
	return &Adapter{
		name:     name,
		spec:     spec,
		runner:   runner,
		lookPath: lookPath,
	}
}

// Name returns the backend identifier.
func (a *Adapter) Name() string { return a.name }

// Spec returns a copy of the backend configuration.
func (a *Adapter) Spec() config.BackendConfig { return a.spec }

// Binary returns the primary executable name or path for this harness.
func (a *Adapter) Binary() string {
	if len(a.spec.Command) == 0 {
		return a.name
	}
	return a.spec.Command[0]
}

// IsAvailable reports whether the harness executable exists on disk or in PATH.
func (a *Adapter) IsAvailable() bool {
	bin := a.Binary()
	if bin == "" {
		return false
	}
	if _, err := a.lookPath(bin); err == nil {
		return true
	}
	info, err := os.Stat(bin)
	return err == nil && !info.IsDir()
}

// Execute runs the AI harness using normalized input/output protocols and sanitization.
func (a *Adapter) Execute(ctx context.Context, req Request) (string, error) {
	mode := strings.ToLower(strings.TrimSpace(req.Mode))
	if mode == "" {
		mode = "agent"
	}
	outFmt := strings.ToLower(strings.TrimSpace(req.OutputFormat))
	if outFmt == "" {
		outFmt = "text"
	}

	activePrompt := a.framePrompt(req.Prompt, req.SystemPrompt, mode)

	rawCmd := make([]string, len(a.spec.Command))
	copy(rawCmd, a.spec.Command)

	if modeFlags, ok := a.spec.ModeFlags[mode]; ok && len(modeFlags) > 0 {
		rawCmd = append(rawCmd, modeFlags...)
	}
	if fmtFlags, ok := a.spec.FormatFlags[outFmt]; ok && len(fmtFlags) > 0 {
		rawCmd = append(rawCmd, fmtFlags...)
	}

	var stdinReader io.Reader

	switch a.spec.InputMode {
	case "temp_file":
		pf, err := os.CreateTemp("", "aitools-prompt-*.txt")
		if err != nil {
			return "", fmt.Errorf("creating prompt temp file: %w", err)
		}
		promptPath := pf.Name()
		defer os.Remove(promptPath)

		if _, err := pf.WriteString(activePrompt); err != nil {
			_ = pf.Close()
			return "", fmt.Errorf("writing prompt temp file: %w", err)
		}
		if err := pf.Close(); err != nil {
			return "", fmt.Errorf("closing prompt temp file: %w", err)
		}

		replaced := false
		for i, arg := range rawCmd {
			if strings.Contains(arg, "{prompt_file}") || strings.Contains(arg, "{input_file}") {
				arg = strings.ReplaceAll(arg, "{prompt_file}", promptPath)
				arg = strings.ReplaceAll(arg, "{input_file}", promptPath)
				rawCmd[i] = arg
				replaced = true
			}
		}
		if !replaced {
			rawCmd = append(rawCmd, promptPath)
		}

	case "stdin":
		stdinReader = strings.NewReader(activePrompt)
		filtered := make([]string, 0, len(rawCmd))
		for _, arg := range rawCmd {
			if arg != "{prompt}" {
				filtered = append(filtered, arg)
			}
		}
		rawCmd = filtered

	default: // "arg"
		replaced := false
		for i, arg := range rawCmd {
			if strings.Contains(arg, "{prompt}") {
				rawCmd[i] = strings.ReplaceAll(arg, "{prompt}", activePrompt)
				replaced = true
			}
		}
		if !replaced {
			rawCmd = append(rawCmd, activePrompt)
		}
	}

	var outputPath string
	if a.spec.OutputMode == "temp_file" {
		of, err := os.CreateTemp("", "aitools-output-*.txt")
		if err != nil {
			return "", fmt.Errorf("creating output temp file: %w", err)
		}
		outputPath = of.Name()
		_ = of.Close()
		defer os.Remove(outputPath)

		replaced := false
		for i, arg := range rawCmd {
			if strings.Contains(arg, "{output_file}") {
				rawCmd[i] = strings.ReplaceAll(arg, "{output_file}", outputPath)
				replaced = true
			}
		}
		if !replaced {
			rawCmd = append(rawCmd, "--output", outputPath)
		}
	}

	effModel := strings.TrimSpace(req.Model)
	if effModel == "" {
		effModel = strings.TrimSpace(a.spec.DefaultModel)
	}
	if effModel != "" {
		replaced := false
		for i, arg := range rawCmd {
			if strings.Contains(arg, "{model}") {
				rawCmd[i] = strings.ReplaceAll(arg, "{model}", effModel)
				replaced = true
			}
		}
		if !replaced && a.spec.ModelFlag != "" {
			rawCmd = append(rawCmd, a.spec.ModelFlag, effModel)
		}
	}

	effEffort := strings.TrimSpace(req.Effort)
	if effEffort == "" {
		effEffort = strings.TrimSpace(a.spec.DefaultEffort)
	}
	if effEffort != "" {
		replaced := false
		for i, arg := range rawCmd {
			if strings.Contains(arg, "{effort}") {
				rawCmd[i] = strings.ReplaceAll(arg, "{effort}", effEffort)
				replaced = true
			}
		}
		if !replaced && a.spec.EffortFlag != "" {
			rawCmd = append(rawCmd, a.spec.EffortFlag, effEffort)
		}
	}

	if !a.IsAvailable() {
		return "", fmt.Errorf("AI CLI binary %q not found in PATH for backend %q", a.Binary(), a.name)
	}

	timeout := req.Timeout
	if timeout <= 0 {
		timeout = time.Duration(a.spec.TimeoutSec) * time.Second
	}

	procRes, err := a.runner.ExecuteProcess(ctx, ProcessSpec{
		Binary:  rawCmd[0],
		Args:    rawCmd[1:],
		Stdin:   stdinReader,
		Timeout: timeout,
	})
	if err != nil {
		return "", err
	}

	if outFmt == "raw" {
		return procRes.Stdout, nil
	}

	var rawOutput string
	switch a.spec.OutputMode {
	case "temp_file":
		b, readErr := os.ReadFile(outputPath)
		if readErr != nil {
			return "", fmt.Errorf("reading harness temp output %q: %w", outputPath, readErr)
		}
		rawOutput = string(b)

	case "json":
		jsonKey := a.spec.JSONKey
		if jsonKey == "" {
			jsonKey = "response"
		}
		extracted, jsonErr := ExtractJSONPath([]byte(procRes.Stdout), jsonKey)
		if jsonErr != nil {
			return "", fmt.Errorf("extracting JSON path %q from %q output: %w", jsonKey, a.name, jsonErr)
		}
		rawOutput = extracted

	default:
		rawOutput = procRes.Stdout
	}

	output := rawOutput
	if a.spec.CleanANSI {
		output = StripANSI(output)
	}
	output = strings.TrimSpace(output)

	cleanMD := a.spec.CleanMarkdown
	if req.CleanMarkdown != nil {
		cleanMD = *req.CleanMarkdown
	}
	if cleanMD {
		output = StripMarkdownFences(output)
	}

	if outFmt == "json" {
		var js any
		if json.Unmarshal([]byte(output), &js) == nil {
			formatted, mErr := json.MarshalIndent(js, "", "  ")
			if mErr == nil {
				return string(formatted), nil
			}
		}
		modelLabel := effModel
		if modelLabel == "" {
			modelLabel = "default"
		}
		envelope := map[string]string{
			"backend":  a.name,
			"model":    modelLabel,
			"mode":     mode,
			"status":   "success",
			"response": output,
		}
		formatted, mErr := json.MarshalIndent(envelope, "", "  ")
		if mErr != nil {
			return "", mErr
		}
		return string(formatted), nil
	}

	return output, nil
}

func (a *Adapter) framePrompt(prompt, systemPrompt, mode string) string {
	if strings.TrimSpace(systemPrompt) != "" {
		return strings.TrimSpace(systemPrompt) + "\n\n" + prompt
	}
	hasModeFlags := len(a.spec.ModeFlags[mode]) > 0
	if hasModeFlags {
		return prompt
	}
	switch mode {
	case "chat", "ask":
		return "[Direct Single-Turn Chat Mode: Answer directly and concisely in a single turn without running multi-step agent actions or modifying files]\n\n" + prompt
	case "action", "simple-agent":
		return "[Single Action Mode: Execute the requested order directly in a single step without conversational preambles]\n\n" + prompt
	default:
		return prompt
	}
}
