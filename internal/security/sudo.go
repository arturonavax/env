package security

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

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

// NewSudoAuthCmd creates an exec.Cmd configured for terminal-only sudo credential refresh.
// Standard input, output, and error are attached to the controlling terminal.
// Graphical askpass helpers are stripped to prevent any GUI popups.
func NewSudoAuthCmd() *exec.Cmd {
	cmd := exec.Command("sudo", "-p", "[sudo] Password for %u (ANX package installation): ", "-v")
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
		return fmt.Errorf("sudo authentication failed: %w", err)
	}
	return nil
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
