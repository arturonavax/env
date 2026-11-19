package app

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/arturonavax/env/internal/domain"
	"github.com/arturonavax/env/internal/security"
)

// SSHCommandRunner abstracts OS command execution and env inspection for SSH agent setup.
type SSHCommandRunner interface {
	Run(ctx context.Context, name string, args ...string) (exitCode int, stdout, stderr string, err error)
	LookPath(name string) (string, error)
	Getenv(key string) string
}

// SyncServiceOption configures optional dependencies on SyncService.
type SyncServiceOption func(*SyncService)

// WithSyncSSHRunner injects a custom SSHCommandRunner (primarily for deterministic tests).
func WithSyncSSHRunner(runner SSHCommandRunner) SyncServiceOption {
	return func(s *SyncService) {
		if runner != nil {
			s.sshRunner = runner
		}
	}
}

type defaultSSHCommandRunner struct{}

func (defaultSSHCommandRunner) LookPath(name string) (string, error) {
	return exec.LookPath(name)
}

func (defaultSSHCommandRunner) Getenv(key string) string {
	return os.Getenv(key)
}

func (defaultSSHCommandRunner) Run(ctx context.Context, name string, args ...string) (int, string, string, error) {
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
	}
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdin = nil
	if name == "ssh-add" {
		// Ensure non-interactive calls never block on macOS Keychain or graphical askpass helpers.
		cmd.Env = append(os.Environ(), "SSH_ASKPASS_REQUIRE=never", "SSH_ASKPASS=")
	}
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	err := cmd.Run()
	if err == nil {
		return 0, outBuf.String(), errBuf.String(), nil
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode(), outBuf.String(), errBuf.String(), err
	}
	return -1, outBuf.String(), errBuf.String(), err
}

func appendSyncDetail(res *SyncResult, msg string) {
	if msg == "" {
		return
	}
	if res.Detail == "" {
		res.Detail = msg
		return
	}
	res.Detail += "; " + msg
}

func (s *SyncService) reconcileSSHAgent(
	ctx context.Context,
	cfg domain.ConfigurationResource,
	opts SyncOptions,
	res *SyncResult,
) error {
	if s.platform == nil || s.sshRunner == nil {
		return nil
	}
	switch s.platform.OS() {
	case "linux":
		if cfg.ID != "ssh-agent-service" {
			return nil
		}
		return s.reconcileLinuxSystemdAgent(ctx, opts, res)
	case "darwin":
		if cfg.ID != "ssh-config" {
			return nil
		}
		s.reconcileDarwinKeychainHint(ctx, opts, res)
		return nil
	default:
		return nil
	}
}

func (s *SyncService) reconcileLinuxSystemdAgent(ctx context.Context, opts SyncOptions, res *SyncResult) error {
	if opts.DryRun {
		appendSyncDetail(res, "would run systemctl --user daemon-reload && systemctl --user enable --now ssh-agent.service")
		return nil
	}
	if _, err := s.sshRunner.LookPath("systemctl"); err != nil {
		appendSyncDetail(res, "systemctl not found; start ssh-agent.service manually")
		return nil
	}
	if code, _, _, err := s.sshRunner.Run(ctx, "systemctl", "--user", "show-environment"); err != nil || code != 0 {
		// Containers, CI runners, and chroots have no active user systemd bus: degrade safely.
		appendSyncDetail(res, "systemd --user session unavailable; skipping ssh-agent.service activation")
		return nil
	}
	if code, _, stderr, err := s.sshRunner.Run(ctx, "systemctl", "--user", "daemon-reload"); err != nil || code != 0 {
		return fmt.Errorf("systemctl --user daemon-reload failed (exit %d): %s", code, strings.TrimSpace(stderr))
	}
	if code, _, stderr, err := s.sshRunner.Run(ctx, "systemctl", "--user", "enable", "--now", "ssh-agent.service"); err != nil || code != 0 {
		return fmt.Errorf("systemctl --user enable --now ssh-agent.service failed (exit %d): %s", code, strings.TrimSpace(stderr))
	}
	appendSyncDetail(res, "systemd --user ssh-agent.service active")
	return nil
}

func (s *SyncService) reconcileDarwinKeychainHint(ctx context.Context, opts SyncOptions, res *SyncResult) {
	if opts.DryRun {
		appendSyncDetail(res, "preserves launchd SSH_AUTH_SOCK; would check ssh-add -l for Keychain hint")
		return
	}
	if _, err := s.sshRunner.LookPath("ssh-add"); err != nil {
		return
	}
	code, _, _, _ := s.sshRunner.Run(ctx, "ssh-add", "-l")
	if code == 1 {
		keyPath := security.PreferredSSHKeyHint(s.platform.HomeDir())
		appendSyncDetail(res, fmt.Sprintf("hint: no SSH keys loaded in Keychain; run \"ssh-add --apple-use-keychain %s\" when ready", keyPath))
	}
}
