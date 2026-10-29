package providers

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/arturonavax/env/internal/manifest"
	"github.com/arturonavax/env/internal/security"
)

func (d *Dispatcher) installLanguagePkg(ctx context.Context, spec manifest.LanguagePkgSpec) error {
	mgr := strings.ToLower(strings.TrimSpace(spec.Manager))
	pkg := strings.TrimSpace(spec.Package)
	if mgr == "" || pkg == "" {
		return fmt.Errorf("language_pkg requires manager and package")
	}

	var binName string
	var args []string

	switch mgr {
	case "go":
		binName = "go"
		args = append([]string{"install"}, spec.Args...)
		args = append(args, pkg)
	case "cargo":
		binName = "cargo"
		args = append([]string{"install"}, spec.Args...)
		args = append(args, pkg)
	case "pnpm":
		binName = "pnpm"
		args = append([]string{"add", "-g"}, spec.Args...)
		args = append(args, pkg)
	case "npm":
		binName = "npm"
		args = append([]string{"install", "-g"}, spec.Args...)
		args = append(args, pkg)
	case "pipx":
		binName = "pipx"
		args = append([]string{"install"}, spec.Args...)
		args = append(args, pkg)
	default:
		return fmt.Errorf("unsupported language_pkg manager %q", mgr)
	}

	binPath := d.locateBinary(binName)
	if binPath == "" {
		return fmt.Errorf("language manager binary %q is not installed", binName)
	}

	env := d.buildSubprocessEnv(map[string]string{
		"GOBIN": d.platform.LocalBinDir(),
	})

	_, err := d.runner.RunWithInput(
		ctx,
		security.RunOptions{
			Env:     env,
			Timeout: 15 * time.Minute,
		},
		binPath,
		args...,
	)
	return err
}

func (d *Dispatcher) uninstallLanguagePkg(ctx context.Context, spec manifest.LanguagePkgSpec) error {
	mgr := strings.ToLower(strings.TrimSpace(spec.Manager))
	pkg := strings.TrimSpace(spec.Package)
	if mgr == "" || pkg == "" {
		return nil
	}

	var binName string
	var args []string

	switch mgr {
	case "cargo":
		binName = "cargo"
		args = []string{"uninstall", pkg}
	case "pnpm":
		binName = "pnpm"
		args = []string{"remove", "-g", pkg}
	case "npm":
		binName = "npm"
		args = []string{"uninstall", "-g", pkg}
	case "pipx":
		binName = "pipx"
		args = []string{"uninstall", pkg}
	default:
		return nil
	}

	binPath := d.locateBinary(binName)
	if binPath == "" {
		return nil
	}
	_, err := d.runner.RunWithOptions(ctx, "", d.buildSubprocessEnv(nil), binPath, args...)
	return err
}
