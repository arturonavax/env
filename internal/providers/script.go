package providers

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/arturonavax/env/internal/manifest"
	"github.com/arturonavax/env/internal/security"
)

func (d *Dispatcher) installScript(ctx context.Context, spec manifest.ScriptSpec) error {
	var scriptPath string
	var cleanup func()

	switch {
	case strings.TrimSpace(spec.LocalScript) != "":
		resolved := spec.LocalScript
		if !filepath.IsAbs(resolved) && d.repoDir != "" {
			safe, err := security.SafePath(d.repoDir, resolved)
			if err != nil {
				return fmt.Errorf("local_script rejected: %w", err)
			}
			resolved = safe
		}
		if _, err := os.Stat(resolved); err != nil {
			return fmt.Errorf("local_script %q not found: %w", resolved, err)
		}
		scriptPath = resolved

	case strings.TrimSpace(spec.URL) != "":
		tmpDir, err := os.MkdirTemp("", "anx-script-*")
		if err != nil {
			return fmt.Errorf("creating temp script dir: %w", err)
		}
		cleanup = func() { _ = os.RemoveAll(tmpDir) }
		scriptName := "install.sh"
		interp := strings.ToLower(strings.TrimSpace(spec.Interpreter))
		if interp == "cmd" || interp == "cmd.exe" {
			scriptName = "install.cmd"
		} else if interp == "powershell" || interp == "pwsh" {
			scriptName = "install.ps1"
		}
		scriptPath = filepath.Join(tmpDir, scriptName)
		if err := d.downloader.DownloadToFile(ctx, strings.TrimSpace(spec.URL), scriptPath, spec.SHA256); err != nil {
			cleanup()
			return err
		}

	default:
		return errors.New("script strategy requires either url or local_script")
	}

	if cleanup != nil {
		defer cleanup()
	}

	scriptBytes, err := os.ReadFile(scriptPath)
	if err != nil {
		return fmt.Errorf("reading installer script: %w", err)
	}
	if violations := security.ValidateScriptGuardrails(string(scriptBytes)); len(violations) > 0 {
		v := violations[0]
		return fmt.Errorf("installer script blocked by security guardrail %s (line %d: %s)", v.RuleID, v.Line, v.Description)
	}

	if d.platform.LocalBinDir() != "" {
		_ = os.MkdirAll(d.platform.LocalBinDir(), 0o755)
	}

	interpreter := strings.TrimSpace(spec.Interpreter)
	if interpreter == "" {
		interpreter = "sh"
	}

	execArgs := make([]string, 0, len(spec.Args)+3)
	lowerInterp := strings.ToLower(interpreter)
	if lowerInterp == "cmd" || lowerInterp == "cmd.exe" {
		execArgs = append(execArgs, "/c")
	} else if lowerInterp == "powershell" || lowerInterp == "pwsh" {
		execArgs = append(execArgs, "-ExecutionPolicy", "Bypass", "-File")
	}
	execArgs = append(execArgs, scriptPath)
	for _, rawArg := range spec.Args {
		execArgs = append(execArgs, expandTemplateString(rawArg, d.platform, d.repoDir))
	}

	env := d.buildSubprocessEnv(spec.Env)
	if _, err := d.runner.RunWithOptions(ctx, "", env, interpreter, execArgs...); err != nil {
		return err
	}

	if spec.Creates != "" {
		expected := expandTemplateString(spec.Creates, d.platform, d.repoDir)
		if _, err := os.Stat(expected); err != nil {
			return fmt.Errorf("script finished without creating expected path %q: %w", expected, err)
		}
	}

	return nil
}
