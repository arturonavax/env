package providers

import (
	"context"
	"fmt"
	"strings"
)

func (d *Dispatcher) selectNativePackages(matrix map[string][]string) (string, []string) {
	if len(matrix) == 0 {
		return "", nil
	}
	for _, pm := range d.platform.PackageManagers() {
		if pkgs, ok := matrix[pm]; ok && len(pkgs) > 0 {
			return pm, pkgs
		}
	}
	return "", nil
}

func (d *Dispatcher) isNativeInstalled(ctx context.Context, matrix map[string][]string) (bool, string) {
	pm, pkgs := d.selectNativePackages(matrix)
	if pm == "" || len(pkgs) == 0 {
		return false, ""
	}
	first := pkgs[0]
	if strings.HasPrefix(first, "--") || strings.HasPrefix(first, "@") {
		return false, ""
	}

	var bin string
	var args []string
	switch pm {
	case "apt", "pkg":
		bin = "dpkg-query"
		args = []string{"-W", "-f=${Status} ${Version}", first}
	case "dnf", "zypper":
		bin = "rpm"
		args = []string{"-q", "--qf", "%{VERSION}", first}
	case "pacman":
		bin = "pacman"
		args = []string{"-Qi", first}
	case "apk":
		bin = "apk"
		args = []string{"info", "-e", first}
	case "brew":
		bin = "brew"
		args = []string{"list", "--versions", first}
	default:
		return false, ""
	}

	out, err := d.runner.Run(ctx, bin, args...)
	if err != nil {
		return false, ""
	}
	if pm == "apt" || pm == "pkg" {
		if !strings.Contains(out, "install ok installed") {
			return false, ""
		}
	}
	return true, "installed"
}

func (d *Dispatcher) installNative(ctx context.Context, matrix map[string][]string, repos map[string][]string) error {
	pm, pkgs := d.selectNativePackages(matrix)
	if pm == "" {
		return fmt.Errorf("no matching native package manager available on host")
	}

	env := d.buildSubprocessEnv(map[string]string{
		"DEBIAN_FRONTEND": "noninteractive",
	})

	if len(repos) > 0 {
		if repoList, ok := repos[pm]; ok && len(repoList) > 0 {
			switch pm {
			case "apt":
				for _, r := range repoList {
					if strings.HasPrefix(r, "ppa:") {
						cmd, args := d.maybeSudo("add-apt-repository", "-y", r)
						_, _ = d.runner.RunWithOptions(ctx, "", env, cmd, args...)
					}
				}
				updateCmd, updateArgs := d.maybeSudo("apt-get", "update")
				_, _ = d.runner.RunWithOptions(ctx, "", env, updateCmd, updateArgs...)
			case "dnf":
				for _, r := range repoList {
					cmd, args := d.maybeSudo("dnf", "copr", "enable", "-y", r)
					_, _ = d.runner.RunWithOptions(ctx, "", env, cmd, args...)
				}
			case "brew":
				for _, r := range repoList {
					_, _ = d.runner.RunWithOptions(ctx, "", env, "brew", "tap", r)
				}
			}
		}
	}

	switch pm {
	case "apt":
		cmd, args := d.maybeSudo("apt-get", append([]string{"install", "-y", "--no-install-recommends"}, pkgs...)...)
		_, err := d.runner.RunWithOptions(ctx, "", env, cmd, args...)
		return err
	case "dnf":
		cmd, args := d.maybeSudo("dnf", append([]string{"install", "-y"}, pkgs...)...)
		_, err := d.runner.RunWithOptions(ctx, "", env, cmd, args...)
		return err
	case "pacman":
		cmd, args := d.maybeSudo("pacman", append([]string{"-S", "--noconfirm", "--needed"}, pkgs...)...)
		_, err := d.runner.RunWithOptions(ctx, "", env, cmd, args...)
		return err
	case "zypper":
		cmd, args := d.maybeSudo("zypper", append([]string{"--non-interactive", "install"}, pkgs...)...)
		_, err := d.runner.RunWithOptions(ctx, "", env, cmd, args...)
		return err
	case "apk":
		cmd, args := d.maybeSudo("apk", append([]string{"add", "--no-cache"}, pkgs...)...)
		_, err := d.runner.RunWithOptions(ctx, "", env, cmd, args...)
		return err
	case "brew":
		_, err := d.runner.RunWithOptions(ctx, "", env, "brew", append([]string{"install"}, pkgs...)...)
		return err
	case "pkg":
		_, err := d.runner.RunWithOptions(ctx, "", env, "pkg", append([]string{"install", "-y"}, pkgs...)...)
		return err
	case "winget":
		for _, p := range pkgs {
			args := []string{
				"install",
				"--id", p,
				"--exact",
				"--silent",
				"--accept-package-agreements",
				"--accept-source-agreements",
			}
			if _, err := d.runner.RunWithOptions(ctx, "", env, "winget", args...); err != nil {
				return err
			}
		}
		return nil
	case "scoop":
		_, err := d.runner.RunWithOptions(ctx, "", env, "scoop", append([]string{"install"}, pkgs...)...)
		return err
	default:
		return fmt.Errorf("unsupported native package manager %q", pm)
	}
}

func (d *Dispatcher) uninstallNative(ctx context.Context, matrix map[string][]string) error {
	pm, pkgs := d.selectNativePackages(matrix)
	if pm == "" {
		return fmt.Errorf("no matching native package manager available on host")
	}

	env := d.buildSubprocessEnv(map[string]string{
		"DEBIAN_FRONTEND": "noninteractive",
	})

	switch pm {
	case "apt":
		cmd, args := d.maybeSudo("apt-get", append([]string{"remove", "-y"}, pkgs...)...)
		_, err := d.runner.RunWithOptions(ctx, "", env, cmd, args...)
		return err
	case "dnf":
		cmd, args := d.maybeSudo("dnf", append([]string{"remove", "-y"}, pkgs...)...)
		_, err := d.runner.RunWithOptions(ctx, "", env, cmd, args...)
		return err
	case "pacman":
		cmd, args := d.maybeSudo("pacman", append([]string{"-Rns", "--noconfirm"}, pkgs...)...)
		_, err := d.runner.RunWithOptions(ctx, "", env, cmd, args...)
		return err
	case "zypper":
		cmd, args := d.maybeSudo("zypper", append([]string{"--non-interactive", "remove"}, pkgs...)...)
		_, err := d.runner.RunWithOptions(ctx, "", env, cmd, args...)
		return err
	case "apk":
		cmd, args := d.maybeSudo("apk", append([]string{"del"}, pkgs...)...)
		_, err := d.runner.RunWithOptions(ctx, "", env, cmd, args...)
		return err
	case "brew":
		_, err := d.runner.RunWithOptions(ctx, "", env, "brew", append([]string{"uninstall"}, pkgs...)...)
		return err
	case "pkg":
		_, err := d.runner.RunWithOptions(ctx, "", env, "pkg", append([]string{"uninstall", "-y"}, pkgs...)...)
		return err
	case "winget":
		for _, p := range pkgs {
			if _, err := d.runner.RunWithOptions(ctx, "", env, "winget", "uninstall", "--id", p, "--exact", "--silent"); err != nil {
				return err
			}
		}
		return nil
	case "scoop":
		_, err := d.runner.RunWithOptions(ctx, "", env, "scoop", append([]string{"uninstall"}, pkgs...)...)
		return err
	default:
		return fmt.Errorf("unsupported native package manager %q", pm)
	}
}

func (d *Dispatcher) maybeSudo(binary string, args ...string) (string, []string) {
	if d.platform.IsRoot() || d.platform.IsTermux() || d.platform.OS() == "windows" || !d.platform.HasSudo() {
		return binary, args
	}
	sudoArgs := make([]string, 0, len(args)+2)
	sudoArgs = append(sudoArgs, "-n", binary)
	sudoArgs = append(sudoArgs, args...)
	return "sudo", sudoArgs
}
