package providers

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/arturonavax/env/internal/domain"
	"github.com/arturonavax/env/internal/manifest"
	"github.com/arturonavax/env/internal/security"
)

func (d *Dispatcher) installFromSource(ctx context.Context, src manifest.SourceSpec) error {
	if strings.TrimSpace(src.GitURL) == "" {
		return errors.New("source strategy requires git_url")
	}
	if len(src.Steps) == 0 {
		return errors.New("source strategy requires at least one build step")
	}

	// Best-effort ensure declared build_deps are installed first.
	for _, depID := range src.BuildDeps {
		depRes := domain.ManagedResource{ID: depID, Name: depID}
		installed, _, _ := d.IsInstalled(ctx, depRes)
		if !installed {
			if err := d.Install(ctx, depRes); err != nil {
				return fmt.Errorf("installing source build dependency %q: %w", depID, err)
			}
		}
	}

	tmpDir, err := os.MkdirTemp("", "anx-srcbuild-*")
	if err != nil {
		return fmt.Errorf("creating source build directory: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	repoCheckout := filepath.Join(tmpDir, "src")
	cloneArgs := []string{"clone", "--depth", "1"}
	if strings.TrimSpace(src.Ref) != "" {
		cloneArgs = append(cloneArgs, "--branch", strings.TrimSpace(src.Ref))
	}
	cloneArgs = append(cloneArgs, strings.TrimSpace(src.GitURL), repoCheckout)

	env := d.buildSubprocessEnv(src.Env)
	if _, err := d.runner.RunWithInput(
		ctx,
		security.RunOptions{
			Env:     env,
			Timeout: 10 * time.Minute,
		},
		"git",
		cloneArgs...,
	); err != nil {
		return fmt.Errorf("cloning %s: %w", src.GitURL, err)
	}

	for idx, rawStep := range src.Steps {
		if len(rawStep) == 0 {
			continue
		}
		bin := expandTemplateString(rawStep[0], d.platform, d.repoDir)
		stepArgs := make([]string, 0, len(rawStep)-1)
		for _, a := range rawStep[1:] {
			stepArgs = append(stepArgs, expandTemplateString(a, d.platform, d.repoDir))
		}
		if _, err := d.runner.RunWithInput(
			ctx,
			security.RunOptions{
				Dir:     repoCheckout,
				Env:     env,
				Timeout: 30 * time.Minute,
			},
			bin,
			stepArgs...,
		); err != nil {
			return fmt.Errorf("source build step %d (%s) failed: %w", idx+1, bin, err)
		}
	}

	return nil
}
