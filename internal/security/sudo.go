package security

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

// SudoLocalPath is the macOS update-safe PAM drop-in managed by anx macos_osconfig (overridable in tests).
var SudoLocalPath = "/etc/pam.d/sudo_local"

// SystemPAMFallbackModules lists Apple-signed system PAM modules that can temporarily stand in for a
// missing optional module (such as /opt/homebrew/.../pam_reattach.so after an interrupted destroyer purge)
// so that /usr/bin/sudo can complete pam_start() and prompt for the user's password to sanitize sudo_local.
var SystemPAMFallbackModules = []string{
	"/usr/lib/pam/pam_permit.so.2",
	"/usr/lib/pam/pam_permit.so",
	"/usr/lib/pam/pam_deny.so.2",
	"/usr/lib/pam/pam_deny.so",
}

// PamReattachBrewPrefixes lists standard Homebrew prefixes where pam-reattach may be linked or shimmed.
var PamReattachBrewPrefixes = []string{
	"/opt/homebrew",
	"/usr/local",
}

func isDarwinOrTestSudoLocal() bool {
	return runtime.GOOS == "darwin" || SudoLocalPath != "/etc/pam.d/sudo_local"
}

// SudoCleanEnv strips environment variables that could invoke graphical askpass helpers.
// This strictly enforces that all authentication occurs in the command line / controlling terminal.
func SudoCleanEnv(base []string) []string {
	cleaned := make([]string, 0, len(base)+4)
	for _, kv := range base {
		if strings.HasPrefix(kv, "SUDO_ASKPASS=") ||
			strings.HasPrefix(kv, "SSH_ASKPASS=") ||
			strings.HasPrefix(kv, "SSH_ASKPASS_REQUIRE=") {
			continue
		}
		cleaned = append(cleaned, kv)
	}
	// Explicitly unset askpass to prevent sudo from falling back to system-wide GUI dialogs
	cleaned = append(cleaned, "SUDO_ASKPASS=", "SSH_ASKPASS=")
	return cleaned
}

// IsSudoAuthenticated checks whether the current user has cached sudo credentials without prompting.
// Before invoking sudo, it repairs any missing user-space PAM module referenced in sudo_local so
// pam_start() does not fail with "sudo: unable to initialize PAM: No such file or directory".
func IsSudoAuthenticated() bool {
	_, _ = EnsureSudoPAMHealthy()
	cmd := exec.Command("sudo", "-n", "true")
	cmd.Env = SudoCleanEnv(os.Environ())
	if cmd.Run() == nil {
		_, _ = SanitizeSudoLocalPAM(context.Background())
		return true
	}
	return false
}

// sudoAuthScript repairs missing user-space PAM modules referenced in /etc/pam.d/sudo_local before
// invoking sudo, and refreshes sudo credentials with a guaranteed password fallback.
const sudoAuthScript = `repair_macos_sudo_pam_if_needed() {
	[ -r /etc/pam.d/sudo_local ] || return 0
	_shim=""
	for _cand in /usr/lib/pam/pam_permit.so.2 /usr/lib/pam/pam_permit.so /usr/lib/pam/pam_deny.so.2 /usr/lib/pam/pam_deny.so; do
		if [ -f "$_cand" ]; then
			_shim="$_cand"
			break
		fi
	done
	[ -n "$_shim" ] || return 0
	while IFS= read -r _line || [ -n "$_line" ]; do
		case "$_line" in
			""|\#*) continue ;;
		esac
		for _tok in $_line; do
			case "$_tok" in
				/*.so)
					if [ ! -f "$_tok" ]; then
						_dir="$(dirname "$_tok")"
						mkdir -p "$_dir" 2>/dev/null || true
						rm -f "$_tok" 2>/dev/null || true
						ln -sfn "$_shim" "$_tok" 2>/dev/null || true
					fi
					;;
			esac
		done
	done < /etc/pam.d/sudo_local
}
repair_macos_sudo_pam_if_needed
announce_sudo_wait() {
	if [ "$(uname -s)" = "Darwin" ]; then
		printf '\n[anx] >>> WAITING FOR SUDO AUTHORIZATION <<<\n[anx] If a Touch ID dialog appears (it may be unfocused), click it and place your finger, or type your password here.\n' >&2
	else
		printf '\n[anx] >>> WAITING FOR SUDO AUTHORIZATION: enter your password below <<<\n' >&2
	fi
}
announce_sudo_wait
prompt='[sudo] Password for %u (ANX package installation): '
if sudo -p "$prompt" -v; then exit 0; fi
printf '\n[anx] sudo authentication failed; resetting cached credentials and retrying with password...\n' >&2
sudo -k >/dev/null 2>&1 || true
repair_macos_sudo_pam_if_needed
announce_sudo_wait
exec sudo -p "$prompt" -v`

// NewSudoAuthCmd creates an exec.Cmd configured for terminal-only sudo credential refresh.
// Standard input, output, and error are attached to the controlling terminal.
// Graphical askpass helpers are stripped to prevent any GUI popups.
// Touch ID is never required: a failed/cancelled biometric prompt falls back to the password,
// and missing user-space PAM modules in /etc/pam.d/sudo_local are shimmed before sudo starts.
func NewSudoAuthCmd() *exec.Cmd {
	_, _ = EnsureSudoPAMHealthy()
	cmd := exec.Command("sh", "-c", sudoAuthScript)
	cmd.Env = SudoCleanEnv(os.Environ())
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd
}

// AuthenticateSudoCLI executes an interactive sudo credential refresh in the controlling terminal.
// It ensures that no graphical authentication dialog (Polkit GUI, pinentry, askpass) is ever spawned.
func AuthenticateSudoCLI() error {
	_, _ = EnsureSudoPAMHealthy()
	cmd := NewSudoAuthCmd()
	if err := cmd.Run(); err != nil {
		if hint := SudoPAMRecoveryHint(); hint != "" {
			return fmt.Errorf("sudo authentication failed: %w\n%s", err, hint)
		}
		return fmt.Errorf("sudo authentication failed: %w", err)
	}
	_, _ = SanitizeSudoLocalPAM(context.Background())
	return nil
}

// isTemporaryPAMShim reports whether soPath is a symlink pointing to a system PAM fallback module.
func isTemporaryPAMShim(soPath string) bool {
	info, err := os.Lstat(soPath)
	if err != nil || info.Mode()&os.ModeSymlink == 0 {
		return false
	}
	target, err := os.Readlink(soPath)
	if err != nil {
		return false
	}
	for _, sysMod := range SystemPAMFallbackModules {
		if target == sysMod {
			return true
		}
	}
	return strings.HasPrefix(target, "/usr/lib/pam/")
}

func firstSystemPAMFallback() string {
	for _, cand := range SystemPAMFallbackModules {
		if info, err := os.Stat(cand); err == nil && !info.IsDir() {
			return cand
		}
	}
	return ""
}

func findExistingRealPamReattach(excludePath string) string {
	var candidates []string
	for _, prefix := range PamReattachBrewPrefixes {
		candidates = append(candidates,
			filepath.Join(prefix, "opt", "pam-reattach", "lib", "pam", "pam_reattach.so"),
			filepath.Join(prefix, "lib", "pam", "pam_reattach.so"),
		)
		if matches, err := filepath.Glob(filepath.Join(prefix, "Cellar", "pam-reattach", "*", "lib", "pam", "pam_reattach.so")); err == nil {
			candidates = append(candidates, matches...)
		}
	}
	for _, cand := range candidates {
		if cand == excludePath || isTemporaryPAMShim(cand) {
			continue
		}
		if info, err := os.Stat(cand); err == nil && !info.IsDir() {
			return cand
		}
	}
	return ""
}

// HasValidPamReattachModule reports whether a non-shim pam_reattach.so exists at the standard Homebrew link path.
func HasValidPamReattachModule() bool {
	for _, prefix := range PamReattachBrewPrefixes {
		for _, cand := range []string{
			filepath.Join(prefix, "lib", "pam", "pam_reattach.so"),
			filepath.Join(prefix, "opt", "pam-reattach", "lib", "pam", "pam_reattach.so"),
		} {
			if isTemporaryPAMShim(cand) {
				continue
			}
			if info, err := os.Stat(cand); err == nil && !info.IsDir() {
				return true
			}
		}
	}
	return false
}

// CleanupPamReattachBrewPaths removes temporary PAM recovery shims and unowned real directories at
// <brew-prefix>/opt/pam-reattach so Homebrew can cleanly create its opt/pam-reattach -> ../Cellar/...
// symlink and link lib/pam/pam_reattach.so without conflict.
func CleanupPamReattachBrewPaths() {
	if !isDarwinOrTestSudoLocal() {
		return
	}
	for _, prefix := range PamReattachBrewPrefixes {
		soPaths := []string{
			filepath.Join(prefix, "opt", "pam-reattach", "lib", "pam", "pam_reattach.so"),
			filepath.Join(prefix, "lib", "pam", "pam_reattach.so"),
		}
		for _, soPath := range soPaths {
			if SudoLocalReferencesModule(soPath) {
				continue
			}
			if isTemporaryPAMShim(soPath) {
				_ = os.Remove(soPath)
				continue
			}
			// Remove dangling symlinks left behind after a Cellar uninstall.
			if linfo, lerr := os.Lstat(soPath); lerr == nil && linfo.Mode()&os.ModeSymlink != 0 {
				if _, serr := os.Stat(soPath); serr != nil {
					_ = os.Remove(soPath)
				}
			}
		}

		// In Homebrew, <prefix>/opt/pam-reattach MUST be a symlink to ../Cellar/pam-reattach/<version>,
		// never a real directory. If emergency mkdir -p created a real directory there, remove it once
		// sudo_local no longer depends on it.
		optDir := filepath.Join(prefix, "opt", "pam-reattach")
		if SudoLocalReferencesModule(optDir) {
			continue
		}
		if linfo, err := os.Lstat(optDir); err == nil && linfo.IsDir() && linfo.Mode()&os.ModeSymlink == 0 {
			optSO := filepath.Join(optDir, "lib", "pam", "pam_reattach.so")
			if isTemporaryPAMShim(optSO) {
				_ = os.Remove(optSO)
			}
			if info, serr := os.Stat(optSO); serr != nil || info.IsDir() {
				_ = os.RemoveAll(optDir)
			}
		}
	}
}

// EnsureSudoPAMHealthy inspects /etc/pam.d/sudo_local for absolute .so module paths (such as
// /opt/homebrew/lib/pam/pam_reattach.so) that were removed by an interrupted destroyer run.
// Because macOS OpenPAM aborts pam_start() with "unable to initialize PAM: No such file or directory"
// before evaluating "optional" whenever a referenced .so file is missing on disk, and because
// /opt/homebrew (and /usr/local/lib/pam) is user-writable without root, EnsureSudoPAMHealthy places a
// temporary symlink to a valid PAM module at the missing path so sudo can start and sanitize sudo_local.
func EnsureSudoPAMHealthy() ([]string, error) {
	if !isDarwinOrTestSudoLocal() {
		return nil, nil
	}
	raw, err := os.ReadFile(SudoLocalPath)
	if err != nil {
		return nil, nil
	}

	var repaired []string
	fallbackMod := firstSystemPAMFallback()

	for _, line := range strings.Split(string(raw), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		for _, field := range strings.Fields(trimmed) {
			if !filepath.IsAbs(field) || !strings.HasSuffix(field, ".so") {
				continue
			}
			if info, err := os.Stat(field); err == nil && !info.IsDir() {
				continue
			}
			target := ""
			if strings.HasSuffix(field, "pam_reattach.so") {
				target = findExistingRealPamReattach(field)
			}
			if target == "" {
				target = fallbackMod
			}
			if target == "" {
				continue
			}
			if err := os.MkdirAll(filepath.Dir(field), 0o755); err != nil {
				continue
			}
			// Remove any dangling symlink before creating the recovery symlink.
			_ = os.Remove(field)
			if err := os.Symlink(target, field); err == nil {
				repaired = append(repaired, field)
			}
		}
	}
	return repaired, nil
}

// brokenSudoLocalModules returns absolute PAM module paths referenced by sudo_local that either no longer
// exist on disk or are currently shimmed to a temporary system fallback module.
func brokenSudoLocalModules(content string) []string {
	var missing []string
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		for _, field := range strings.Fields(trimmed) {
			if filepath.IsAbs(field) && strings.HasSuffix(field, ".so") {
				if isTemporaryPAMShim(field) {
					missing = append(missing, field)
					continue
				}
				if _, err := os.Stat(field); err != nil {
					missing = append(missing, field)
				}
			}
		}
	}
	return missing
}

// externalSudoLocalModules returns non-system absolute PAM module paths (outside /usr/lib/pam/)
// referenced by sudo_local, such as /opt/homebrew/.../pam_reattach.so.
func externalSudoLocalModules(content string) []string {
	var ext []string
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		for _, field := range strings.Fields(trimmed) {
			if filepath.IsAbs(field) && strings.HasSuffix(field, ".so") && !strings.HasPrefix(field, "/usr/lib/pam/") {
				ext = append(ext, field)
			}
		}
	}
	return ext
}

// SudoLocalReferencesModule reports whether /etc/pam.d/sudo_local currently contains an active line
// referencing the given module name or path substring.
func SudoLocalReferencesModule(moduleSubstr string) bool {
	if !isDarwinOrTestSudoLocal() || moduleSubstr == "" {
		return false
	}
	raw, err := os.ReadFile(SudoLocalPath)
	if err != nil {
		return false
	}
	for _, line := range strings.Split(string(raw), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if strings.Contains(trimmed, moduleSubstr) {
			return true
		}
	}
	return false
}

// SudoPAMRecoveryHint returns an actionable message when sudo_local references missing PAM modules.
func SudoPAMRecoveryHint() string {
	if !isDarwinOrTestSudoLocal() {
		return ""
	}
	raw, err := os.ReadFile(SudoLocalPath)
	if err != nil {
		return ""
	}
	missing := brokenSudoLocalModules(string(raw))
	if len(missing) == 0 {
		return ""
	}
	return fmt.Sprintf("hint: %s references missing PAM module(s) %s. "+
		"Run `mkdir -p $(dirname %s) && ln -sfn /usr/lib/pam/pam_permit.so.2 %s` as your user to unblock sudo immediately, "+
		"then remove the stale line with `sudo sed -i '' '/pam_reattach.so/d' %s`.",
		SudoLocalPath, strings.Join(missing, ", "), missing[0], missing[0], SudoLocalPath)
}

func rewriteSudoLocalFiltering(ctx context.Context, dropLine func(line string) ([]string, bool)) (bool, error) {
	if !isDarwinOrTestSudoLocal() {
		return false, nil
	}
	// Ensure any missing .so is shimmed first so "sudo -n install" itself can initialize PAM.
	_, _ = EnsureSudoPAMHealthy()

	raw, err := os.ReadFile(SudoLocalPath)
	if err != nil {
		return false, nil
	}
	var (
		kept        []string
		shimmedPaths []string
		changed     bool
	)
	for _, line := range strings.Split(string(raw), "\n") {
		if paths, drop := dropLine(line); drop {
			changed = true
			shimmedPaths = append(shimmedPaths, paths...)
			continue
		}
		kept = append(kept, line)
	}
	if !changed {
		CleanupPamReattachBrewPaths()
		return false, nil
	}

	tmp, err := os.CreateTemp("", "anx-sudo_local.*")
	if err != nil {
		return false, err
	}
	defer func() { _ = os.Remove(tmp.Name()) }()
	if _, err := tmp.WriteString(strings.Join(kept, "\n")); err != nil {
		_ = tmp.Close()
		return false, err
	}
	if err := tmp.Close(); err != nil {
		return false, err
	}
	cmd := exec.CommandContext(ctx, "sudo", "-n", "install", "-m", "0644", tmp.Name(), SudoLocalPath)
	cmd.Env = SudoCleanEnv(os.Environ())
	if err := cmd.Run(); err != nil {
		return false, fmt.Errorf("sanitizing %s: %w", SudoLocalPath, err)
	}
	// Only remove temporary shims after sudo_local no longer references them.
	for _, p := range shimmedPaths {
		if isTemporaryPAMShim(p) {
			_ = os.Remove(p)
		}
	}
	CleanupPamReattachBrewPaths()
	return true, nil
}

// SanitizeSudoLocalPAM removes stale or shimmed PAM module lines from sudo_local using cached sudo
// credentials only (never prompts). It returns true when the file was rewritten.
func SanitizeSudoLocalPAM(ctx context.Context) (bool, error) {
	return rewriteSudoLocalFiltering(ctx, func(line string) ([]string, bool) {
		broken := brokenSudoLocalModules(line)
		return broken, len(broken) > 0
	})
}

// PrepareSudoLocalForPurge strips external (non-/usr/lib/pam/) PAM module lines from sudo_local
// before Destroyer uninstalls packages (such as pam-reattach), preventing an interrupted purge
// from ever leaving sudo_local pointing at a deleted module.
func PrepareSudoLocalForPurge(ctx context.Context) (bool, error) {
	return rewriteSudoLocalFiltering(ctx, func(line string) ([]string, bool) {
		ext := externalSudoLocalModules(line)
		return ext, len(ext) > 0
	})
}

// StartSudoKeepalive starts a background loop that periodically refreshes the sudo timestamp
// using "sudo -n -v" so that credentials do not expire during long installation processes.
// It returns a cancel/stop function.
func StartSudoKeepalive(ctx context.Context) func() {
	stopCh := make(chan struct{})
	var once sync.Once

	go func() {
		ticker := time.NewTicker(60 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-stopCh:
				return
			case <-ticker.C:
				_, _ = EnsureSudoPAMHealthy()
				cmd := exec.CommandContext(ctx, "sudo", "-n", "-v")
				cmd.Env = SudoCleanEnv(os.Environ())
				_ = cmd.Run()
			}
		}
	}()

	return func() {
		once.Do(func() {
			close(stopCh)
		})
	}
}
