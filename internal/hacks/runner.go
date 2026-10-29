package hacks

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	embeddedhacks "github.com/arturonavax/env/hacks"
	"github.com/arturonavax/env/internal/domain"
	"github.com/arturonavax/env/internal/platform"
	"github.com/arturonavax/env/internal/security"
	"github.com/arturonavax/env/internal/snapshot"
	"github.com/arturonavax/env/internal/state"
)

// ErrHackGuardrailViolation indicates a hack script violated static security guardrails.
var ErrHackGuardrailViolation = errors.New("hack script blocked by security guardrails")

var supplementaryHackRootRmRe = regexp.MustCompile(
	`(?i)\brm\s+(?:[^\n;|&]*\s+)?-(?:[a-zA-Z]*[rf][a-zA-Z]*|-recursive|-force)(?:\s+[^\n;|&]*)?\s+(/|/\*|/boot|/etc|/usr|/var|/home|--no-preserve-root)(?:\s|$)`,
)

// HackPlanStep previews one concrete step inside a HackPlan.
type HackPlanStep struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	CommandLine string   `json:"command_line"`
	ScriptPath  string   `json:"script_path,omitempty"`
	Touches     []string `json:"touches"`
	Privileged  bool     `json:"privileged"`
}

// HackPlan describes the non-destructive execution preview of a HackProfile.
type HackPlan struct {
	HackID      string         `json:"hack_id"`
	Name        string         `json:"name"`
	Category    string         `json:"category"`
	SafeByOptIn bool           `json:"safe_by_opt_in"`
	BackupPaths []string       `json:"backup_paths"`
	Steps       []HackPlanStep `json:"steps"`
}

// ApplyOptions bundles dependencies and execution overrides for ApplyHack.
type ApplyOptions struct {
	Platform  *platform.Info
	Runner    *security.Runner
	Snapshots *snapshot.Manager
	Store     *state.Store
	Env       []string
	WorkDir   string
}

// HackApplyResult summarizes a completed hack application.
type HackApplyResult struct {
	HackID       string   `json:"hack_id"`
	SnapshotID   string   `json:"snapshot_id,omitempty"`
	AppliedSteps []string `json:"applied_steps"`
	Outputs      []string `json:"outputs"`
}

type preparedStep struct {
	step       domain.HackStep
	binary     string
	args       []string
	cleanupDir string
}

// PlanHack builds a dry-run execution plan expanding target and backup paths against plat.
func PlanHack(plat *platform.Info, profile domain.HackProfile) (*HackPlan, error) {
	if strings.TrimSpace(profile.ID) == "" {
		return nil, errors.New("hack profile id is required")
	}

	backupPaths := expandPathSlice(plat, profile.BackupPaths)
	steps := make([]HackPlanStep, 0, len(profile.Steps))

	for _, st := range profile.Steps {
		touches := expandPathSlice(plat, st.Touches)
		var scriptPath string
		var cmdParts []string

		if st.Privileged && (plat == nil || (!plat.IsRoot() && plat.HasSudo())) {
			cmdParts = append(cmdParts, "sudo")
		}

		if strings.TrimSpace(st.Script) != "" {
			if profile.Dir != "" {
				safeScript, err := security.SafePath(profile.Dir, st.Script)
				if err != nil {
					return nil, fmt.Errorf("step %q script path invalid: %w", st.ID, err)
				}
				scriptPath = safeScript
			} else {
				scriptPath = path.Join("embedded://hacks", profile.ID, st.Script)
			}
			cmdParts = append(cmdParts, "bash", scriptPath)
			cmdParts = append(cmdParts, st.Args...)
		} else if len(st.Command) > 0 {
			cmdParts = append(cmdParts, st.Command...)
			cmdParts = append(cmdParts, st.Args...)
		} else {
			return nil, fmt.Errorf("step %q has neither script nor command", st.ID)
		}

		steps = append(steps, HackPlanStep{
			ID:          st.ID,
			Name:        st.Name,
			Description: st.Description,
			CommandLine: strings.Join(cmdParts, " "),
			ScriptPath:  scriptPath,
			Touches:     touches,
			Privileged:  st.Privileged,
		})
	}

	return &HackPlan{
		HackID:      profile.ID,
		Name:        profile.Name,
		Category:    profile.Category,
		SafeByOptIn: profile.SafeByOptIn,
		BackupPaths: backupPaths,
		Steps:       steps,
	}, nil
}

// ApplyHack validates all referenced scripts with security guardrails, creates a pre-hack snapshot,
// executes each step via security.Runner, and records completion in state.Store.
func ApplyHack(
	ctx context.Context,
	profile domain.HackProfile,
	opts ApplyOptions,
) (*HackApplyResult, error) {
	if strings.TrimSpace(profile.ID) == "" {
		return nil, errors.New("hack profile id is required")
	}

	runner := opts.Runner
	if runner == nil {
		runner = security.NewRunner(2 * time.Minute)
	}

	// Phase 1: Resolve and guardrail-validate every step BEFORE taking backups or mutating files.
	prepared := make([]preparedStep, 0, len(profile.Steps))
	defer func() {
		for _, ps := range prepared {
			if ps.cleanupDir != "" {
				_ = os.RemoveAll(ps.cleanupDir)
			}
		}
	}()

	for _, st := range profile.Steps {
		ps, err := prepareAndValidateStep(profile, st, opts.Platform)
		if err != nil {
			return nil, err
		}
		prepared = append(prepared, ps)
	}

	result := &HackApplyResult{
		HackID:       profile.ID,
		AppliedSteps: make([]string, 0, len(prepared)),
		Outputs:      make([]string, 0, len(prepared)),
	}

	// Phase 2: Pre-hack backup snapshot of configured BackupPaths.
	expandedBackups := expandPathSlice(opts.Platform, profile.BackupPaths)
	if opts.Snapshots != nil && len(expandedBackups) > 0 {
		snap, err := opts.Snapshots.CreateSnapshot("pre-hack:"+profile.ID, expandedBackups)
		if err != nil {
			return nil, fmt.Errorf("creating pre-hack backup snapshot: %w", err)
		}
		if snap != nil {
			result.SnapshotID = snap.ID
		}
	}

	// Phase 3: Execute steps sequentially with explicit HOME injection.
	execEnv := buildHackExecutionEnv(opts.Platform, opts.Env)
	workDir := opts.WorkDir
	if workDir == "" && profile.Dir != "" {
		workDir = profile.Dir
	}

	for _, ps := range prepared {
		if err := ctx.Err(); err != nil {
			return result, err
		}
		out, err := runner.RunWithOptions(ctx, workDir, execEnv, ps.binary, ps.args...)
		result.Outputs = append(result.Outputs, out)
		if err != nil {
			return result, fmt.Errorf("executing hack %q step %q: %w", profile.ID, ps.step.ID, err)
		}
		result.AppliedSteps = append(result.AppliedSteps, ps.step.ID)
	}

	// Phase 4: Record applied hack in state.json if store is provided.
	if opts.Store != nil {
		if err := opts.Store.RecordHack(profile.ID, result.AppliedSteps); err != nil {
			return result, fmt.Errorf("recording applied hack %q in state store: %w", profile.ID, err)
		}
	}

	return result, nil
}

func prepareAndValidateStep(
	profile domain.HackProfile,
	st domain.HackStep,
	plat *platform.Info,
) (preparedStep, error) {
	useSudo := st.Privileged && plat != nil && !plat.IsRoot() && plat.HasSudo()

	if strings.TrimSpace(st.Script) != "" {
		scriptPath, scriptContent, cleanupDir, err := resolveHackScript(profile, st.Script)
		if err != nil {
			return preparedStep{}, fmt.Errorf("resolving script for step %q: %w", st.ID, err)
		}

		if violations := validateHackGuardrails(scriptContent); len(violations) > 0 {
			if cleanupDir != "" {
				_ = os.RemoveAll(cleanupDir)
			}
			return preparedStep{}, formatHackGuardrailError(profile.ID, st.ID, violations)
		}

		cmdArgs := make([]string, 0, len(st.Args)+2)
		binary := "bash"
		if useSudo {
			binary = "sudo"
			cmdArgs = append(cmdArgs, "bash", scriptPath)
		} else {
			cmdArgs = append(cmdArgs, scriptPath)
		}
		cmdArgs = append(cmdArgs, st.Args...)

		return preparedStep{
			step:       st,
			binary:     binary,
			args:       cmdArgs,
			cleanupDir: cleanupDir,
		}, nil
	}

	if len(st.Command) == 0 {
		return preparedStep{}, fmt.Errorf("step %q requires script or command", st.ID)
	}

	joinedCmd := strings.Join(append(append([]string{}, st.Command...), st.Args...), " ")
	if violations := validateHackGuardrails(joinedCmd); len(violations) > 0 {
		return preparedStep{}, formatHackGuardrailError(profile.ID, st.ID, violations)
	}

	if useSudo {
		args := make([]string, 0, len(st.Command)+len(st.Args))
		args = append(args, st.Command...)
		args = append(args, st.Args...)
		return preparedStep{
			step:   st,
			binary: "sudo",
			args:   args,
		}, nil
	}

	args := make([]string, 0, len(st.Command)-1+len(st.Args))
	args = append(args, st.Command[1:]...)
	args = append(args, st.Args...)
	return preparedStep{
		step:   st,
		binary: st.Command[0],
		args:   args,
	}, nil
}

func resolveHackScript(profile domain.HackProfile, relScript string) (string, string, string, error) {
	if profile.Dir != "" {
		safePath, err := security.SafePath(profile.Dir, relScript)
		if err != nil {
			return "", "", "", err
		}
		raw, err := os.ReadFile(safePath)
		if err != nil {
			return "", "", "", err
		}
		return safePath, string(raw), "", nil
	}

	cleanRel := path.Clean(strings.TrimLeft(filepath.ToSlash(relScript), "/"))
	if strings.HasPrefix(cleanRel, "..") {
		return "", "", "", security.ErrPathTraversal
	}
	embedPath := path.Join(profile.ID, cleanRel)
	raw, err := fs.ReadFile(embeddedhacks.EmbeddedFS, embedPath)
	if err != nil {
		return "", "", "", fmt.Errorf("reading embedded hack script %q: %w", embedPath, err)
	}

	tmpDir, err := os.MkdirTemp("", "anx-hack-*")
	if err != nil {
		return "", "", "", err
	}
	tmpScript := filepath.Join(tmpDir, filepath.Base(cleanRel))
	if err := os.WriteFile(tmpScript, raw, 0o700); err != nil {
		_ = os.RemoveAll(tmpDir)
		return "", "", "", err
	}
	return tmpScript, string(raw), tmpDir, nil
}

func expandPathSlice(plat *platform.Info, rawPaths []string) []string {
	if len(rawPaths) == 0 {
		return []string{}
	}
	out := make([]string, 0, len(rawPaths))
	for _, raw := range rawPaths {
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" {
			continue
		}
		if plat != nil {
			out = append(out, plat.ExpandPath(trimmed))
		} else {
			out = append(out, trimmed)
		}
	}
	return out
}

func buildHackExecutionEnv(plat *platform.Info, baseEnv []string) []string {
	src := baseEnv
	if len(src) == 0 {
		src = os.Environ()
	}
	if plat == nil || plat.HomeDir() == "" {
		return src
	}

	homeKV := "HOME=" + plat.HomeDir()
	out := make([]string, 0, len(src)+1)
	replaced := false
	for _, kv := range src {
		if strings.HasPrefix(kv, "HOME=") {
			out = append(out, homeKV)
			replaced = true
		} else {
			out = append(out, kv)
		}
	}
	if !replaced {
		out = append(out, homeKV)
	}
	return out
}

func validateHackGuardrails(script string) []security.GuardrailViolation {
	violations := security.ValidateScriptGuardrails(script)
	hasLineGR001 := map[int]bool{}
	for _, v := range violations {
		if v.RuleID == "GR-001" {
			hasLineGR001[v.Line] = true
		}
	}
	for idx, rawLine := range strings.Split(script, "\n") {
		trimmed := strings.TrimSpace(rawLine)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") || hasLineGR001[idx+1] {
			continue
		}
		if supplementaryHackRootRmRe.MatchString(trimmed) {
			violations = append(violations, security.GuardrailViolation{
				RuleID:      "GR-001",
				Description: "recursive deletion targeting root or top-level system directories",
				Line:        idx + 1,
				Snippet:     trimmed,
			})
		}
	}
	return violations
}

func formatHackGuardrailError(hackID, stepID string, violations []security.GuardrailViolation) error {
	parts := make([]string, 0, len(violations))
	for _, v := range violations {
		parts = append(parts, fmt.Sprintf("[%s line %d: %s]", v.RuleID, v.Line, v.Description))
	}
	return fmt.Errorf("%w in hack %q step %q: %s", ErrHackGuardrailViolation, hackID, stepID, strings.Join(parts, "; "))
}
