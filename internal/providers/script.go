package providers

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

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

	scriptCtx, cancelScript := context.WithTimeout(ctx, 3*time.Minute)
	defer cancelScript()

	var watcherDone chan struct{}
	if spec.Creates != "" {
		expectedPath := expandTemplateString(spec.Creates, d.platform, d.repoDir)
		watcherDone = make(chan struct{})
		go func() {
			ticker := time.NewTicker(500 * time.Millisecond)
			defer ticker.Stop()
			for {
				select {
				case <-watcherDone:
					return
				case <-scriptCtx.Done():
					return
				case <-ticker.C:
					if info, err := os.Stat(expectedPath); err == nil && info.Size() > 0 {
						select {
						case <-watcherDone:
							return
						case <-scriptCtx.Done():
							return
						case <-time.After(2 * time.Second):
							if handler := security.OutputHandlerFromContext(scriptCtx); handler != nil {
								handler(fmt.Sprintf("Target artifact %s verified; completing installer script.", filepath.Base(expectedPath)))
							}
							cancelScript()
							return
						}
					}
				}
			}
		}()
	}

	_, err = d.runner.RunWithOptions(scriptCtx, "", env, interpreter, execArgs...)
	if watcherDone != nil {
		close(watcherDone)
	}

	if spec.Creates != "" {
		expected := expandTemplateString(spec.Creates, d.platform, d.repoDir)
		if _, statErr := os.Stat(expected); statErr == nil {
			return nil
		}
	}

	if err != nil {
		return err
	}

	return nil
}
