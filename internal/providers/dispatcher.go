package providers

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/arturonavax/env/internal/domain"
	"github.com/arturonavax/env/internal/manifest"
	"github.com/arturonavax/env/internal/platform"
	"github.com/arturonavax/env/internal/security"
)

// Dispatcher executes declarative multi-tier package detection, installation, fallback, and removal.
type Dispatcher struct {
	platform   *platform.Info
	catalog    *manifest.Catalog
	runner     *security.Runner
	downloader *security.SafeDownloader
	repoDir    string
}

var _ Provider = (*Dispatcher)(nil)

// DispatcherOption configures a Dispatcher instance.
type DispatcherOption func(*Dispatcher)

// WithSafeDownloader overrides the HTTP downloader used for GitHub releases and remote scripts.
func WithSafeDownloader(dl *security.SafeDownloader) DispatcherOption {
	return func(d *Dispatcher) {
		if dl != nil {
			d.downloader = dl
		}
	}
}

// NewDispatcher constructs a multi-strategy Dispatcher backed by platform info, declarative manifests, and a hardened runner.
func NewDispatcher(
	plat *platform.Info,
	cat *manifest.Catalog,
	runner *security.Runner,
	repoDir string,
	opts ...DispatcherOption,
) *Dispatcher {
	if plat == nil {
		plat = platform.Detect()
	}
	if runner == nil {
		runner = security.NewRunner(5 * time.Minute)
	}
	d := &Dispatcher{
		platform:   plat,
		catalog:    cat,
		runner:     runner,
		downloader: security.NewSafeDownloader(5*time.Minute, 512*1024*1024),
		repoDir:    repoDir,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(d)
		}
	}
	return d
}

func (d *Dispatcher) Name() string {
	if pm := d.platform.PrimaryPackageManager(); pm != "" {
		return "multi(" + pm + ")"
	}
	return "multi"
}

func (d *Dispatcher) IsAvailable(_ context.Context) bool { return true }

// SelectStrategy returns the first eligible StrategySpec for pkgID on the current host.
func (d *Dispatcher) SelectStrategy(pkgID string) (manifest.StrategySpec, bool) {
	spec := d.resolveSpec(domain.ManagedResource{ID: pkgID, Name: pkgID})
	if !d.isSpecSupportedOnHost(spec) {
		return manifest.StrategySpec{}, false
	}
	for _, st := range spec.Strategies {
		if !d.isStrategyEligible(st) {
			continue
		}
		return st, true
	}
	return manifest.StrategySpec{}, false
}

func (d *Dispatcher) isStrategyEligible(st manifest.StrategySpec) bool {
	if !d.matchesCondition(st.When) {
		return false
	}
	switch st.Type {
	case string(domain.StrategyNative):
		pm, _ := d.selectNativePackages(st.Packages)
		return pm != ""
	case string(domain.StrategyGitHubRelease):
		if st.GitHubRelease == nil {
			return false
		}
		targetKey := d.platform.OS() + "/" + d.platform.Arch()
		if direct := strings.TrimSpace(st.GitHubRelease.DirectURLs[targetKey]); direct != "" {
			return true
		}
		return strings.TrimSpace(st.GitHubRelease.AssetPatterns[targetKey]) != ""
	case string(domain.StrategyScript):
		return st.Script != nil
	case string(domain.StrategySource):
		return st.Source != nil
	case string(domain.StrategyLanguagePkg):
		return st.LanguagePkg != nil
	default:
		return true
	}
}

func (d *Dispatcher) isSpecSupportedOnHost(spec manifest.PackageSpec) bool {
	return spec.SupportsOS(d.platform.OS()) && spec.SupportsDisplayServer(d.platform.DisplayServer())
}

// EffectiveDependencies returns the deduplicated host-applicable dependencies for pkgID
// based on package-level depends_on and the winning eligible strategy's depends_on.
func (d *Dispatcher) EffectiveDependencies(pkgID string) []string {
	spec, ok := d.lookupSpec(domain.ManagedResource{ID: pkgID, Name: pkgID})
	if !ok || !d.isSpecSupportedOnHost(spec) {
		return nil
	}

	raw := make([]string, 0, len(spec.DependsOn)+4)
	raw = append(raw, spec.DependsOn...)
	if st, found := d.SelectStrategy(pkgID); found {
		raw = append(raw, st.DependsOn...)
	}

	seen := make(map[string]bool, len(raw))
	out := make([]string, 0, len(raw))
	for _, depID := range raw {
		trimmed := strings.TrimSpace(depID)
		if trimmed == "" || seen[trimmed] {
			continue
		}
		if depSpec, exists := d.lookupSpec(domain.ManagedResource{ID: trimmed, Name: trimmed}); exists {
			if !d.isSpecSupportedOnHost(depSpec) {
				continue
			}
		}
		seen[trimmed] = true
		out = append(out, trimmed)
	}
	return out
}

// PreviewStrategy returns a human-readable description of the primary eligible strategy for pkg on this host.
func (d *Dispatcher) PreviewStrategy(pkgID string) string {
	st, ok := d.SelectStrategy(pkgID)
	if !ok {
		return "unsupported"
	}
	switch st.Type {
	case string(domain.StrategyNative):
		if pm, pkgs := d.selectNativePackages(st.Packages); pm != "" {
			return fmt.Sprintf("native:%s (%s)", pm, strings.Join(pkgs, " "))
		}
	case string(domain.StrategyGitHubRelease):
		if st.GitHubRelease != nil && st.GitHubRelease.Repo != "" {
			return "github_release:" + st.GitHubRelease.Repo
		}
		return "github_release"
	case string(domain.StrategyScript):
		if st.Script != nil && st.Script.LocalScript != "" {
			return "script:" + st.Script.LocalScript
		}
		if st.Script != nil && st.Script.URL != "" {
			return "script:" + st.Script.URL
		}
		return "script"
	case string(domain.StrategySource):
		if st.Source != nil {
			return "source:" + st.Source.GitURL
		}
		return "source"
	case string(domain.StrategyLanguagePkg):
		if st.LanguagePkg != nil {
			return fmt.Sprintf("language_pkg:%s (%s)", st.LanguagePkg.Manager, st.LanguagePkg.Package)
		}
	}
	return "unsupported"
}

// IsInstalled detects whether pkg is installed in user-local bin paths, system PATH, or the host package database.
func (d *Dispatcher) IsInstalled(ctx context.Context, pkg domain.ManagedResource) (bool, string, error) {
	spec, inCatalog := d.lookupSpec(pkg)

	candidates := []string{}
	if spec.BinaryName != "" {
		candidates = append(candidates, spec.BinaryName)
	} else if pkg.BinaryName != "" {
		candidates = append(candidates, pkg.BinaryName)
	} else if !inCatalog && pkg.ID != "" {
		candidates = append(candidates, pkg.ID)
	}
	for _, alias := range spec.BinaryAliases {
		if alias != "" && !slices.Contains(candidates, alias) {
			candidates = append(candidates, alias)
		}
	}

	pkgID := spec.ID
	if pkgID == "" {
		pkgID = pkg.ID
	}
	for _, bin := range candidates {
		if binPath := d.locateBinary(bin); binPath != "" {
			if d.isBrokenManagedBinary(ctx, pkgID, binPath) {
				continue
			}
			ver := probeBinaryVersion(ctx, d.runner, binPath, spec.VersionArgs)
			return true, ver, nil
		}
	}

	// Check explicit "creates" artifact paths declared on script strategies (e.g. ~/.gvm/scripts/gvm).
	for _, st := range spec.Strategies {
		if st.Script != nil && st.Script.Creates != "" {
			createdPath := expandTemplateString(st.Script.Creates, d.platform, d.repoDir)
			if info, err := os.Stat(createdPath); err == nil && !info.IsDir() {
				if pkgID == "zsh-plugins" && !d.hasAllZshPlugins() {
					continue
				}
				if len(candidates) == 0 {
					return true, "installed", nil
				}
			}
		}
	}

	// For header/library packages without a binary name, probe the active native package manager database.
	if len(candidates) == 0 {
		for _, st := range spec.Strategies {
			if st.Type == string(domain.StrategyNative) {
				if installed, ver := d.isNativeInstalled(ctx, st.Packages); installed {
					return true, ver, nil
				}
			}
		}
	}

	return false, "", nil
}

// isBrokenManagedBinary catches shim or incomplete single-file installs that exist on PATH
// but cannot execute cleanly without toolchain configuration or package metadata.
func (d *Dispatcher) isBrokenManagedBinary(ctx context.Context, pkgID, binPath string) bool {
	switch pkgID {
	case "rust":
		if d.runner == nil {
			return false
		}
		opts := security.RunOptions{
			Timeout: 4 * time.Second,
			Env:     d.buildSubprocessEnv(nil),
		}
		if d.platform != nil {
			if home := d.platform.HomeDir(); home != "" {
				if info, err := os.Stat(home); err == nil && info.IsDir() {
					opts.Dir = home
				}
			}
		}
		_, err := d.runner.RunWithInput(ctx, opts, binPath, "--version")
		return err != nil
	case "codex-cli":
		if d.platform == nil || d.platform.LocalBinDir() == "" || d.platform.HomeDir() == "" {
			return false
		}
		if filepath.Clean(filepath.Dir(binPath)) != filepath.Clean(d.platform.LocalBinDir()) {
			return false
		}
		marker := filepath.Join(
			d.platform.HomeDir(),
			".codex",
			"packages",
			"standalone",
			"current",
			"codex-package.json",
		)
		if info, err := os.Stat(marker); err == nil && !info.IsDir() {
			return false
		}
		if linfo, err := os.Lstat(binPath); err == nil && linfo.Mode()&os.ModeSymlink != 0 {
			if target, rerr := filepath.EvalSymlinks(binPath); rerr == nil &&
				filepath.Clean(filepath.Dir(target)) != filepath.Clean(d.platform.LocalBinDir()) {
				return false
			}
		}
		return true
	default:
		return false
	}
}

func (d *Dispatcher) hasAllZshPlugins() bool {
	if d.platform == nil || d.platform.HomeDir() == "" {
		return false
	}
	zshDir := filepath.Join(d.platform.HomeDir(), ".zsh")
	required := []string{
		filepath.Join(zshDir, "fast-syntax-highlighting", "fast-syntax-highlighting.plugin.zsh"),
		filepath.Join(zshDir, "zsh-autosuggestions", "zsh-autosuggestions.zsh"),
		filepath.Join(zshDir, "fzf-tab", "fzf-tab.plugin.zsh"),
		filepath.Join(zshDir, "zsh-completions", "zsh-completions.plugin.zsh"),
	}
	for _, path := range required {
		if info, err := os.Stat(path); err != nil || info.IsDir() {
			return false
		}
	}
	return true
}

// Install executes the ordered declarative strategies for pkg, automatically falling back across tiers on error.
func (d *Dispatcher) Install(ctx context.Context, pkg domain.ManagedResource) error {
	spec := d.resolveSpec(pkg)
	if !spec.SupportsOS(d.platform.OS()) {
		return domain.NewExitError(
			domain.ExitPreflightFailed,
			fmt.Errorf("package %q does not support OS %q", spec.ID, d.platform.OS()),
		)
	}

	tierErrors := []error{}
	attempted := 0

	for idx, st := range spec.Strategies {
		if !d.isStrategyEligible(st) {
			continue
		}

		if len(st.DependsOn) > 0 {
			if depErr := d.ensureDependencies(ctx, st.DependsOn, "strategy dependency"); depErr != nil {
				tierErrors = append(tierErrors, fmt.Errorf("tier %d (%s): %w", idx+1, st.Type, depErr))
				continue
			}
		}

		var err error
		switch st.Type {
		case string(domain.StrategyNative):
			if pm, _ := d.selectNativePackages(st.Packages); pm == "" {
				continue
			}
			attempted++
			err = d.installNative(ctx, st.Packages, st.Repositories)
		case string(domain.StrategyGitHubRelease):
			if st.GitHubRelease == nil {
				continue
			}
			attempted++
			err = d.installGitHubRelease(ctx, spec, *st.GitHubRelease)
		case string(domain.StrategyScript):
			if st.Script == nil {
				continue
			}
			attempted++
			err = d.installScript(ctx, *st.Script)
		case string(domain.StrategySource):
			if st.Source == nil {
				continue
			}
			attempted++
			err = d.installFromSource(ctx, *st.Source)
		case string(domain.StrategyLanguagePkg):
			if st.LanguagePkg == nil {
				continue
			}
			attempted++
			err = d.installLanguagePkg(ctx, *st.LanguagePkg)
		default:
			err = fmt.Errorf("unsupported strategy type %q", st.Type)
		}

		if err == nil {
			return nil
		}
		tierErrors = append(tierErrors, fmt.Errorf("tier %d (%s): %w", idx+1, st.Type, err))
	}

	if attempted == 0 {
		return fmt.Errorf(
			"no eligible installation strategy for package %q on %s/%s (package managers: %v)",
			spec.ID,
			d.platform.OS(),
			d.platform.Arch(),
			d.platform.PackageManagers(),
		)
	}

	return fmt.Errorf("all %d installation tiers failed for %q: %w", attempted, spec.ID, errors.Join(tierErrors...))
}

func (d *Dispatcher) ensureDependencies(ctx context.Context, depIDs []string, label string) error {
	for _, depID := range depIDs {
		depSpec := d.resolveSpec(domain.ManagedResource{ID: depID, Name: depID})
		if !d.isSpecSupportedOnHost(depSpec) {
			continue
		}
		depRes := depSpec.ToManagedResource()
		installed, _, _ := d.IsInstalled(ctx, depRes)
		if !installed {
			if err := d.Install(ctx, depRes); err != nil {
				return fmt.Errorf("installing %s %q: %w", label, depID, err)
			}
		}
	}
	return nil
}

// Uninstall removes user-local artifacts and/or native packages for pkg.
func (d *Dispatcher) Uninstall(ctx context.Context, pkg domain.ManagedResource) error {
	spec := d.resolveSpec(pkg)
	removedLocal := false

	candidates := []string{}
	if spec.BinaryName != "" {
		candidates = append(candidates, filepath.Join(d.platform.LocalBinDir(), spec.BinaryName))
		if d.platform.OS() == "windows" && !strings.HasSuffix(spec.BinaryName, ".exe") {
			candidates = append(candidates, filepath.Join(d.platform.LocalBinDir(), spec.BinaryName+".exe"))
		}
	}

	for _, st := range spec.Strategies {
		if st.GitHubRelease != nil {
			if st.GitHubRelease.InstallTo != "" {
				candidates = append(candidates, expandTemplateString(st.GitHubRelease.InstallTo, d.platform, d.repoDir))
			}
			if st.GitHubRelease.ExtractDirTo != "" {
				candidates = append(candidates, expandTemplateString(st.GitHubRelease.ExtractDirTo, d.platform, d.repoDir))
			}
		}
		if st.Script != nil && st.Script.Creates != "" {
			candidates = append(candidates, expandTemplateString(st.Script.Creates, d.platform, d.repoDir))
		}
		if st.Source != nil {
			for _, up := range st.Source.UninstallPaths {
				candidates = append(candidates, expandTemplateString(up, d.platform, d.repoDir))
			}
		}
	}

	home := d.platform.HomeDir()
	for _, target := range candidates {
		if target == "" || home == "" {
			continue
		}
		safeTarget, err := security.SafePath(home, target)
		if err != nil {
			continue
		}
		if _, statErr := os.Lstat(safeTarget); statErr == nil {
			if rmErr := os.RemoveAll(safeTarget); rmErr == nil {
				removedLocal = true
			}
		}
	}

	for _, st := range spec.Strategies {
		if !d.matchesCondition(st.When) {
			continue
		}
		switch st.Type {
		case string(domain.StrategyLanguagePkg):
			if st.LanguagePkg != nil {
				if err := d.uninstallLanguagePkg(ctx, *st.LanguagePkg); err == nil {
					removedLocal = true
				}
			}
		case string(domain.StrategyNative):
			if !removedLocal {
				if pm, _ := d.selectNativePackages(st.Packages); pm != "" {
					if err := d.uninstallNative(ctx, st.Packages); err == nil {
						return nil
					}
				}
			}
		}
	}

	return nil
}

func (d *Dispatcher) lookupSpec(pkg domain.ManagedResource) (manifest.PackageSpec, bool) {
	if d.catalog != nil {
		if spec, ok := d.catalog.GetPackageSpec(pkg.ID); ok {
			return spec, true
		}
	}
	return manifest.PackageSpec{}, false
}

func (d *Dispatcher) resolveSpec(pkg domain.ManagedResource) manifest.PackageSpec {
	if spec, ok := d.lookupSpec(pkg); ok {
		return spec
	}
	allPMs := map[string][]string{
		"apt":    {pkg.ID},
		"dnf":    {pkg.ID},
		"pacman": {pkg.ID},
		"zypper": {pkg.ID},
		"apk":    {pkg.ID},
		"brew":   {pkg.ID},
		"pkg":    {pkg.ID},
		"winget": {pkg.ID},
		"scoop":  {pkg.ID},
	}
	return manifest.PackageSpec{
		ID:         pkg.ID,
		Name:       pkg.Name,
		BinaryName: pkg.BinaryName,
		Strategies: []manifest.StrategySpec{
			{
				Type:     string(domain.StrategyNative),
				Packages: allPMs,
			},
		},
	}
}

func (d *Dispatcher) matchesCondition(cond *manifest.StrategyCondition) bool {
	if cond == nil {
		return true
	}
	if len(cond.OS) > 0 && !slices.Contains(cond.OS, d.platform.OS()) {
		return false
	}
	if len(cond.Arch) > 0 && !slices.Contains(cond.Arch, d.platform.Arch()) {
		return false
	}
	if len(cond.Distros) > 0 {
		distro := d.platform.Distribution()
		family := d.platform.DistroFamily()
		if !slices.Contains(cond.Distros, distro) && !slices.Contains(cond.Distros, family) {
			return false
		}
	}
	if len(cond.Desktops) > 0 {
		de := d.platform.DesktopEnv()
		matched := false
		for _, allowed := range cond.Desktops {
			if strings.EqualFold(allowed, de) || allowed == "all" {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}
	if len(cond.DisplayServers) > 0 {
		ds := d.platform.DisplayServer()
		matched := false
		for _, allowed := range cond.DisplayServers {
			if strings.EqualFold(allowed, ds) || allowed == "all" {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}
	if cond.RequiresBinary != "" && d.locateBinary(cond.RequiresBinary) == "" {
		return false
	}
	return true
}

func (d *Dispatcher) locateBinary(name string) string {
	if name == "" {
		return ""
	}
	names := []string{name}
	if d.platform.OS() == "windows" {
		lower := strings.ToLower(name)
		if !strings.HasSuffix(lower, ".exe") && !strings.HasSuffix(lower, ".cmd") && !strings.HasSuffix(lower, ".bat") {
			names = []string{name + ".exe", name + ".cmd", name + ".bat", name}
		}
	}

	home := d.platform.HomeDir()
	searchDirs := []string{d.platform.LocalBinDir()}
	if home != "" {
		searchDirs = append(
			searchDirs,
			filepath.Join(home, ".cargo", "bin"),
			filepath.Join(home, "go", "bin"),
			filepath.Join(home, ".atuin", "bin"),
			filepath.Join(home, ".local", "share", "fnm"),
			filepath.Join(home, ".local", "share", "pnpm"),
			filepath.Join(home, ".pyenv", "bin"),
			filepath.Join(home, ".gvm", "bin"),
		)
	}

	for _, dir := range searchDirs {
		if dir == "" {
			continue
		}
		for _, candidate := range names {
			full := filepath.Join(dir, candidate)
			if info, err := os.Stat(full); err == nil && !info.IsDir() {
				return full
			}
		}
	}

	for _, candidate := range names {
		if found, err := exec.LookPath(candidate); err == nil {
			return found
		}
	}
	return ""
}

func (d *Dispatcher) buildSubprocessEnv(extra map[string]string) []string {
	baseEnv := os.Environ()
	home := d.platform.HomeDir()
	localBin := d.platform.LocalBinDir()

	prependPaths := []string{}
	if localBin != "" {
		prependPaths = append(prependPaths, localBin)
	}
	if home != "" {
		prependPaths = append(
			prependPaths,
			filepath.Join(home, ".cargo", "bin"),
			filepath.Join(home, "go", "bin"),
			filepath.Join(home, ".local", "share", "pnpm"),
			filepath.Join(home, ".local", "share", "fnm"),
		)
	}

	envMap := make(map[string]string, len(baseEnv)+len(extra)+2)
	var currentPath string
	for _, kv := range baseEnv {
		k, v, ok := strings.Cut(kv, "=")
		if !ok {
			continue
		}
		if k == "PATH" {
			currentPath = v
		}
		envMap[k] = v
	}

	if home != "" {
		envMap["HOME"] = home
	}

	// Enforce non-interactive execution across all child processes and scripts
	envMap["DEBIAN_FRONTEND"] = "noninteractive"
	envMap["CI"] = "true"
	envMap["NONINTERACTIVE"] = "1"
	envMap["AUTOMATED"] = "1"
	envMap["INTERACTIVE"] = "0"
	envMap["GIT_TERMINAL_PROMPT"] = "0"
	envMap["ATUIN_NON_INTERACTIVE"] = "yes"
	envMap["HOMEBREW_NO_AUTO_UPDATE"] = "1"
	envMap["HOMEBREW_NO_INSTALL_CLEANUP"] = "1"
	envMap["HOMEBREW_NO_INTERACTIVE"] = "1"
	envMap["NEEDRESTART_MODE"] = "a"
	envMap["UCF_FORCE_CONFFOLD"] = "1"
	if len(prependPaths) > 0 {
		joined := strings.Join(prependPaths, string(os.PathListSeparator))
		if currentPath != "" {
			envMap["PATH"] = joined + string(os.PathListSeparator) + currentPath
		} else {
			envMap["PATH"] = joined
		}
	}
	for k, v := range extra {
		envMap[k] = expandTemplateString(v, d.platform, d.repoDir)
	}

	keys := make([]string, 0, len(envMap))
	for k := range envMap {
		keys = append(keys, k)
	}
	slices.Sort(keys)

	out := make([]string, 0, len(keys))
	for _, k := range keys {
		out = append(out, k+"="+envMap[k])
	}
	return out
}
