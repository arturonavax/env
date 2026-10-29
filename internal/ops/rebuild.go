package ops

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strings"
	"time"

	"github.com/arturonavax/env/internal/platform"
	"github.com/arturonavax/env/internal/security"
	"golang.org/x/sync/errgroup"
)

const goRebuildConcurrencyLimit = 4

var (
	canonicalRebuildScopes = []string{"go", "rust", "node", "python", "nvim", "zsh"}
	cargoListHeaderRe      = regexp.MustCompile(`^([a-zA-Z0-9_-]+)\s+v[0-9][^\s:]*(\s+\(([^)]+)\))?:$`)
)

// RebuilderOption configures a Rebuilder instance.
type RebuilderOption func(*Rebuilder)

// WithDryRun configures whether Rebuild previews commands without executing mutations.
func WithDryRun(dryRun bool) RebuilderOption {
	return func(r *Rebuilder) {
		r.DryRun = dryRun
	}
}

// WithEnv overrides the process environment used for tool discovery and child execution.
func WithEnv(env []string) RebuilderOption {
	return func(r *Rebuilder) {
		r.env = append([]string{}, env...)
	}
}

// RebuildResult summarizes the rebuild execution for one language or editor scope.
type RebuildResult struct {
	Scope    string        `json:"scope"`
	Commands []string      `json:"commands"`
	Rebuilt  int           `json:"rebuilt"`
	Skipped  bool          `json:"skipped"`
	DryRun   bool          `json:"dry_run"`
	Message  string        `json:"message,omitempty"`
	Duration time.Duration `json:"duration"`
	Error    string        `json:"error,omitempty"`
}

// Rebuilder synchronizes and recompiles global developer tools across language ecosystems.
type Rebuilder struct {
	plat   *platform.Info
	runner *security.Runner
	env    []string
	DryRun bool
}

// NewRebuilder creates a new native Go tool Rebuilder.
func NewRebuilder(plat *platform.Info, runner *security.Runner, opts ...RebuilderOption) *Rebuilder {
	if runner == nil {
		runner = security.NewRunner(5 * time.Minute)
	}
	r := &Rebuilder{
		plat:   plat,
		runner: runner,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(r)
		}
	}
	return r
}

// Rebuild executes or previews tool rebuilds across the requested scopes (all, go, rust, node, python, nvim, zsh).
func (r *Rebuilder) Rebuild(ctx context.Context, scopes []string) ([]RebuildResult, error) {
	resolvedScopes, err := resolveRebuildScopes(scopes)
	if err != nil {
		return nil, err
	}

	results := make([]RebuildResult, 0, len(resolvedScopes))
	var errs []error

	for _, scope := range resolvedScopes {
		if err := ctx.Err(); err != nil {
			return results, err
		}

		start := time.Now()
		var res RebuildResult
		var scopeErr error

		switch scope {
		case "go":
			res, scopeErr = r.rebuildGo(ctx)
		case "rust":
			res, scopeErr = r.rebuildRust(ctx)
		case "node":
			res, scopeErr = r.rebuildNode(ctx)
		case "python":
			res, scopeErr = r.rebuildPython(ctx)
		case "nvim":
			res, scopeErr = r.rebuildNvim(ctx)
		case "zsh":
			res, scopeErr = r.rebuildZsh(ctx)
		}

		res.Scope = scope
		res.DryRun = r.DryRun
		res.Duration = time.Since(start)
		if res.Commands == nil {
			res.Commands = []string{}
		}
		if scopeErr != nil {
			res.Error = scopeErr.Error()
			errs = append(errs, fmt.Errorf("scope %s: %w", scope, scopeErr))
		}
		results = append(results, res)
	}

	return results, errors.Join(errs...)
}

func (r *Rebuilder) rebuildGo(ctx context.Context) (RebuildResult, error) {
	res := RebuildResult{Commands: []string{}}
	goBin, ok := r.lookupBin("go")
	if !ok {
		res.Skipped = true
		res.Message = "go compiler not found in PATH"
		return res, nil
	}

	candidates := r.discoverGoBinaries()
	pkgSet := map[string]bool{}
	for _, binPath := range candidates {
		out, err := r.runCmd(ctx, goBin, "version", "-m", binPath)
		if err != nil {
			continue
		}
		if pkgPath := parseGoVersionModulePath(out); pkgPath != "" {
			pkgSet[pkgPath] = true
		}
	}

	pkgs := make([]string, 0, len(pkgSet))
	for pkg := range pkgSet {
		pkgs = append(pkgs, pkg)
	}
	slices.Sort(pkgs)

	specs := make([]string, 0, len(pkgs))
	for _, pkg := range pkgs {
		spec := pkg + "@latest"
		specs = append(specs, spec)
		res.Commands = append(res.Commands, fmt.Sprintf("go install -ldflags='-s -w' %s", spec))
	}

	if len(specs) == 0 {
		res.Message = "no Go module binaries discovered to rebuild"
		return res, nil
	}

	if r.DryRun {
		res.Rebuilt = len(specs)
		res.Message = fmt.Sprintf("dry-run: %d Go binaries scheduled", len(specs))
		return res, nil
	}

	installErrs := make([]error, len(specs))
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(goRebuildConcurrencyLimit)

	for idx, spec := range specs {
		i := idx
		pkgSpec := spec
		g.Go(func() error {
			_, err := r.runCmd(gctx, goBin, "install", "-ldflags=-s -w", pkgSpec)
			if err != nil {
				installErrs[i] = fmt.Errorf("installing %s: %w", pkgSpec, err)
			}
			return nil
		})
	}
	_ = g.Wait()

	var joinedErr error
	rebuilt := 0
	for _, iErr := range installErrs {
		if iErr == nil {
			rebuilt++
		} else {
			joinedErr = errors.Join(joinedErr, iErr)
		}
	}
	res.Rebuilt = rebuilt
	return res, joinedErr
}

func (r *Rebuilder) rebuildRust(ctx context.Context) (RebuildResult, error) {
	res := RebuildResult{Commands: []string{}}
	rustupBin, hasRustup := r.lookupBin("rustup")
	cargoBin, hasCargo := r.lookupBin("cargo")

	if !hasRustup && !hasCargo {
		res.Skipped = true
		res.Message = "neither rustup nor cargo found in PATH"
		return res, nil
	}

	var errs []error
	if hasRustup {
		res.Commands = append(res.Commands, "rustup update stable")
		if !r.DryRun {
			if _, err := r.runCmd(ctx, rustupBin, "update", "stable"); err != nil {
				errs = append(errs, fmt.Errorf("rustup update stable: %w", err))
			}
		}
	}

	if !hasCargo {
		return res, errors.Join(errs...)
	}

	listOut, err := r.runCmd(ctx, cargoBin, "install", "--list")
	if err != nil {
		return res, errors.Join(append(errs, fmt.Errorf("cargo install --list: %w", err))...)
	}

	crates := parseCargoInstalledCrates(listOut)
	for _, crate := range crates {
		res.Commands = append(res.Commands, "cargo install "+crate)
	}

	if r.DryRun {
		res.Rebuilt = len(crates)
		return res, nil
	}

	rebuilt := 0
	for _, crate := range crates {
		if _, err := r.runCmd(ctx, cargoBin, "install", crate); err != nil {
			errs = append(errs, fmt.Errorf("cargo install %s: %w", crate, err))
			continue
		}
		rebuilt++
	}
	res.Rebuilt = rebuilt
	return res, errors.Join(errs...)
}

func (r *Rebuilder) rebuildNode(ctx context.Context) (RebuildResult, error) {
	res := RebuildResult{Commands: []string{}}
	if pnpmBin, ok := r.lookupBin("pnpm"); ok {
		res.Commands = append(res.Commands, "pnpm update -g")
		res.Rebuilt = 1
		if r.DryRun {
			return res, nil
		}
		if _, err := r.runCmd(ctx, pnpmBin, "update", "-g"); err != nil {
			return res, err
		}
		return res, nil
	}

	if npmBin, ok := r.lookupBin("npm"); ok {
		res.Commands = append(res.Commands, "npm update -g")
		res.Rebuilt = 1
		if r.DryRun {
			return res, nil
		}
		if _, err := r.runCmd(ctx, npmBin, "update", "-g"); err != nil {
			return res, err
		}
		return res, nil
	}

	res.Skipped = true
	res.Message = "neither pnpm nor npm found in PATH"
	return res, nil
}

func (r *Rebuilder) rebuildPython(ctx context.Context) (RebuildResult, error) {
	res := RebuildResult{Commands: []string{}}
	pipxBin, ok := r.lookupBin("pipx")
	if !ok {
		res.Skipped = true
		res.Message = "pipx not found in PATH"
		return res, nil
	}

	res.Commands = append(res.Commands, "pipx reinstall-all")
	res.Rebuilt = 1
	if r.DryRun {
		return res, nil
	}

	if _, err := r.runCmd(ctx, pipxBin, "reinstall-all"); err != nil {
		return res, err
	}
	return res, nil
}

func (r *Rebuilder) rebuildNvim(ctx context.Context) (RebuildResult, error) {
	res := RebuildResult{Commands: []string{}}
	nvimBin, ok := r.lookupBin("nvim")
	if !ok {
		res.Skipped = true
		res.Message = "nvim not found in PATH"
		return res, nil
	}

	res.Commands = append(
		res.Commands,
		`nvim --headless "+Lazy! sync" +qa`,
		`nvim --headless "+MasonUpdate" +qa`,
	)
	if r.DryRun {
		res.Rebuilt = 2
		return res, nil
	}

	var errs []error
	rebuilt := 0
	for _, cmdArg := range []string{"+Lazy! sync", "+MasonUpdate"} {
		if _, err := r.runCmd(ctx, nvimBin, "--headless", cmdArg, "+qa"); err != nil {
			errs = append(errs, err)
			continue
		}
		rebuilt++
	}
	res.Rebuilt = rebuilt
	return res, errors.Join(errs...)
}

func (r *Rebuilder) rebuildZsh(ctx context.Context) (RebuildResult, error) {
	res := RebuildResult{Commands: []string{}}
	zshBin, ok := r.lookupBin("zsh")
	if !ok {
		res.Skipped = true
		res.Message = "zsh not found in PATH"
		return res, nil
	}

	home := r.homeDir()
	if home == "" {
		res.Skipped = true
		res.Message = "home directory unavailable for zsh word-code cache compilation"
		return res, nil
	}

	candidates := []string{".zshrc", ".base.zsh", ".tools.sh", ".lscolors.sh", ".zcompdump"}
	existingTargets := make([]string, 0, len(candidates))
	for _, rel := range candidates {
		full := filepath.Join(home, rel)
		if info, err := os.Stat(full); err == nil && !info.IsDir() {
			existingTargets = append(existingTargets, full)
			res.Commands = append(res.Commands, fmt.Sprintf("zsh -c 'zcompile \"$1\"' _ %s", full))
		}
	}

	if len(existingTargets) == 0 {
		res.Message = "no zsh startup files found to recompile"
		return res, nil
	}

	if r.DryRun {
		res.Rebuilt = len(existingTargets)
		return res, nil
	}

	var errs []error
	rebuilt := 0
	for _, target := range existingTargets {
		_ = os.Remove(target + ".zwc")
		if _, err := r.runCmd(ctx, zshBin, "-c", `zcompile "$1"`, "_", target); err != nil {
			errs = append(errs, fmt.Errorf("zcompile %s: %w", target, err))
			continue
		}
		rebuilt++
	}
	res.Rebuilt = rebuilt
	return res, errors.Join(errs...)
}

func (r *Rebuilder) discoverGoBinaries() []string {
	home := r.homeDir()
	dirs := []string{}

	if gobin := r.getenv("GOBIN"); gobin != "" {
		dirs = append(dirs, gobin)
	}
	if home != "" {
		dirs = append(dirs, filepath.Join(home, "go", "bin"))
	}
	if r.plat != nil && r.plat.LocalBinDir() != "" {
		dirs = append(dirs, r.plat.LocalBinDir())
	} else if home != "" {
		dirs = append(dirs, filepath.Join(home, ".local", "bin"))
	}

	seenDirs := map[string]bool{}
	seenBins := map[string]bool{}
	bins := []string{}

	for _, dir := range dirs {
		cleanDir := filepath.Clean(dir)
		if cleanDir == "" || seenDirs[cleanDir] {
			continue
		}
		seenDirs[cleanDir] = true

		entries, err := os.ReadDir(cleanDir)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if entry.IsDir() || entry.Name() == "anx" || entry.Name() == "anx.exe" {
				continue
			}
			full := filepath.Join(cleanDir, entry.Name())
			info, err := os.Stat(full)
			if err != nil || info.IsDir() {
				continue
			}
			if runtime.GOOS != "windows" && info.Mode()&0o111 == 0 {
				continue
			}
			if !seenBins[full] {
				seenBins[full] = true
				bins = append(bins, full)
			}
		}
	}

	slices.Sort(bins)
	return bins
}

func (r *Rebuilder) effectiveEnv() []string {
	if len(r.env) > 0 {
		return r.env
	}
	return os.Environ()
}

func (r *Rebuilder) getenv(key string) string {
	return lookupEnvValue(r.effectiveEnv(), key)
}

func (r *Rebuilder) homeDir() string {
	if r.plat != nil && r.plat.HomeDir() != "" {
		return r.plat.HomeDir()
	}
	return r.getenv("HOME")
}

func (r *Rebuilder) lookupBin(name string) (string, bool) {
	return lookupBinaryInEnv(name, r.effectiveEnv())
}

func (r *Rebuilder) runCmd(ctx context.Context, binary string, args ...string) (string, error) {
	return r.runner.RunWithOptions(ctx, "", r.effectiveEnv(), binary, args...)
}

func resolveRebuildScopes(scopes []string) ([]string, error) {
	if len(scopes) == 0 {
		return append([]string{}, canonicalRebuildScopes...), nil
	}

	seen := map[string]bool{}
	for _, raw := range scopes {
		norm := strings.ToLower(strings.TrimSpace(raw))
		if norm == "" {
			continue
		}
		if norm == "all" {
			return append([]string{}, canonicalRebuildScopes...), nil
		}
		if !slices.Contains(canonicalRebuildScopes, norm) {
			return nil, fmt.Errorf("unsupported rebuild scope %q (allowed: all, %s)", raw, strings.Join(canonicalRebuildScopes, ", "))
		}
		seen[norm] = true
	}

	if len(seen) == 0 {
		return append([]string{}, canonicalRebuildScopes...), nil
	}

	ordered := make([]string, 0, len(seen))
	for _, s := range canonicalRebuildScopes {
		if seen[s] {
			ordered = append(ordered, s)
		}
	}
	return ordered, nil
}

func parseGoVersionModulePath(output string) string {
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "path\t") && !strings.HasPrefix(line, "path ") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		pkgPath := strings.TrimSpace(fields[1])
		if pkgPath == "" ||
			pkgPath == "command-line-arguments" ||
			strings.HasPrefix(pkgPath, ".") ||
			strings.HasPrefix(pkgPath, "/") {
			return ""
		}
		return pkgPath
	}
	return ""
}

func parseCargoInstalledCrates(output string) []string {
	crates := []string{}
	seen := map[string]bool{}

	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		rawLine := scanner.Text()
		if rawLine == "" || rawLine[0] == ' ' || rawLine[0] == '\t' {
			continue
		}
		matches := cargoListHeaderRe.FindStringSubmatch(strings.TrimSpace(rawLine))
		if len(matches) < 4 {
			continue
		}
		crateName := matches[1]
		source := strings.TrimSpace(matches[3])
		// Skip local path installs that cannot be reinstalled by crate name alone.
		if strings.HasPrefix(source, "/") || strings.HasPrefix(source, ".") {
			continue
		}
		if !seen[crateName] {
			seen[crateName] = true
			crates = append(crates, crateName)
		}
	}

	slices.Sort(crates)
	return crates
}

func lookupEnvValue(env []string, key string) string {
	prefix := key + "="
	for _, kv := range env {
		if strings.HasPrefix(kv, prefix) {
			return strings.TrimPrefix(kv, prefix)
		}
	}
	return ""
}

func lookupBinaryInEnv(name string, env []string) (string, bool) {
	pathVal := lookupEnvValue(env, "PATH")
	if pathVal == "" && len(env) == 0 {
		pathVal = os.Getenv("PATH")
	}
	if pathVal == "" {
		return "", false
	}

	exts := []string{""}
	if runtime.GOOS == "windows" {
		exts = []string{"", ".exe", ".cmd", ".bat"}
	}

	for _, dir := range filepath.SplitList(pathVal) {
		if dir == "" {
			continue
		}
		for _, ext := range exts {
			candidate := filepath.Join(dir, name+ext)
			info, err := os.Stat(candidate)
			if err != nil || info.IsDir() {
				continue
			}
			if runtime.GOOS != "windows" && info.Mode()&0o111 == 0 {
				continue
			}
			return candidate, true
		}
	}
	return "", false
}
