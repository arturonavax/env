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
func IsSudoAuthenticated() bool {
	cmd := exec.Command("sudo", "-n", "true")
	cmd.Env = SudoCleanEnv(os.Environ())
	return cmd.Run() == nil
}

// sudoAuthScript refreshes sudo credentials with a guaranteed password fallback.
// pam_tid (Touch ID) is "sufficient", so cancelling or failing biometrics falls through to the
// normal password prompt; if the first attempt still fails (stale/corrupt timestamp, interrupted
// Touch ID session), the timestamp is reset with "sudo -k" and the password prompt is retried.
const sudoAuthScript = `prompt='[sudo] Password for %u (ANX package installation): '
if sudo -p "$prompt" -v; then exit 0; fi
printf '\n[anx] sudo authentication failed; resetting cached credentials and retrying with password...\n' >&2
sudo -k >/dev/null 2>&1 || true
exec sudo -p "$prompt" -v`

// NewSudoAuthCmd creates an exec.Cmd configured for terminal-only sudo credential refresh.
// Standard input, output, and error are attached to the controlling terminal.
// Graphical askpass helpers are stripped to prevent any GUI popups.
// Touch ID is never required: a failed/cancelled biometric prompt falls back to the password.
func NewSudoAuthCmd() *exec.Cmd {
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

// brokenSudoLocalModules returns absolute PAM module paths referenced by sudo_local that no longer exist
// (e.g. pam_reattach.so after "brew uninstall pam-reattach" during an interrupted destroyer run).
func brokenSudoLocalModules(content string) []string {
	var missing []string
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		for _, field := range strings.Fields(trimmed) {
			if filepath.IsAbs(field) && strings.HasSuffix(field, ".so") {
				if _, err := os.Stat(field); err != nil {
					missing = append(missing, field)
				}
			}
		}
	}
	return missing
}

// SudoPAMRecoveryHint returns an actionable message when sudo_local references missing PAM modules.
func SudoPAMRecoveryHint() string {
	if runtime.GOOS != "darwin" {
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
	return fmt.Sprintf("hint: %s references missing PAM module(s) %s. Remove the stale line(s) from an admin session: "+
		"sudo sed -i '' '/pam_reattach.so/d' %s (or `sudo rm %s`; anx macos-osconfig regenerates it). "+
		"Password login always works once the stale module is gone; Touch ID is optional.",
		SudoLocalPath, strings.Join(missing, ", "), SudoLocalPath, SudoLocalPath)
}

// SanitizeSudoLocalPAM removes stale PAM module lines (modules that no longer exist on disk) from sudo_local
// using cached sudo credentials only (never prompts). It returns true when the file was rewritten.
func SanitizeSudoLocalPAM(ctx context.Context) (bool, error) {
	if runtime.GOOS != "darwin" {
		return false, nil
	}
	raw, err := os.ReadFile(SudoLocalPath)
	if err != nil {
		return false, nil
	}
	if len(brokenSudoLocalModules(string(raw))) == 0 {
		return false, nil
	}
	var kept []string
	for _, line := range strings.Split(string(raw), "\n") {
		if len(brokenSudoLocalModules(line)) > 0 {
			continue
		}
		kept = append(kept, line)
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
	return true, nil
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
