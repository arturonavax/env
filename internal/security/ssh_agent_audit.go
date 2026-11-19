package security

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"
)

var sshAgentManualSpawnPattern = regexp.MustCompile(`(?i)(eval\s+["']?\$\(\s*ssh-agent|eval\s+` + "`" + `\s*ssh-agent|\bssh-agent\s+-[sc]\b)`)

func (sa *SecurityAuditor) defaultCommandRunner(ctx context.Context, name string, args ...string) (int, string, string, error) {
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
	}
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdin = nil
	env := append([]string{}, os.Environ()...)
	if sa.getenv != nil {
		if sock := sa.getenv("SSH_AUTH_SOCK"); sock != "" {
			env = append(env, "SSH_AUTH_SOCK="+sock)
		}
	}
	if name == "ssh-add" {
		env = append(env, "SSH_ASKPASS_REQUIRE=never", "SSH_ASKPASS=")
	}
	cmd.Env = env

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

func (sa *SecurityAuditor) auditSSHAgent(ctx context.Context, report *SecurityAuditReport) {
	sa.auditSSHPermissionsAndConfig(report)
	sa.auditSSHRCFiles(report)
	sa.auditSSHAgentSocketAndConnectivity(ctx, report)
}

func (sa *SecurityAuditor) auditSSHPermissionsAndConfig(report *SecurityAuditReport) {
	if sa.homeDir == "" {
		return
	}
	sshDir := filepath.Join(sa.homeDir, ".ssh")
	if info, err := os.Stat(sshDir); err == nil && info.IsDir() {
		perm := info.Mode().Perm()
		if perm != 0o700 {
			report.Findings = append(report.Findings, SecurityFinding{
				ID:                 "SEC-SSH-DIR-PERM",
				Category:           CategoryPermissions,
				Severity:           SeverityHigh,
				Title:              fmt.Sprintf("~/.ssh directory has unsafe permissions (%04o)", perm),
				Description:        fmt.Sprintf("Directory %s should be 0700 to prevent local inspection.", sshDir),
				Remediation:        fmt.Sprintf("chmod 700 %s", sshDir),
				RemediationCommand: fmt.Sprintf("chmod 700 %q", sshDir),
				Impact:             "OpenSSH rejects or weakens strict host/agent boundary checks.",
				ItemPath:           sshDir,
			})
		}
	}

	sshCfgPath := filepath.Join(sshDir, "config")
	info, err := os.Stat(sshCfgPath)
	if err != nil {
		return
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		report.Findings = append(report.Findings, SecurityFinding{
			ID:                 "SEC-SSH-CFG-PERM",
			Category:           CategoryPermissions,
			Severity:           SeverityHigh,
			Title:              fmt.Sprintf("~/.ssh/config has unsafe permissions (%04o)", perm),
			Description:        fmt.Sprintf("File %s has mode %04o (expected 0600).", sshCfgPath, perm),
			Remediation:        fmt.Sprintf("chmod 600 %s", sshCfgPath),
			RemediationCommand: fmt.Sprintf("chmod 600 %q", sshCfgPath),
			Impact:             "OpenSSH refuses to read ~/.ssh/config when group or other permissions are set.",
			ItemPath:           sshCfgPath,
		})
	}

	raw, err := os.ReadFile(sshCfgPath)
	if err != nil {
		return
	}
	if hasWildcardForwardAgent(string(raw)) {
		report.Findings = append(report.Findings, SecurityFinding{
			ID:          "SEC-SSH-FORWARDAGENT",
			Category:    CategoryAuth,
			Severity:    SeverityHigh,
			Title:       "Global 'ForwardAgent yes' enabled under Host * in ~/.ssh/config",
			Description: "ForwardAgent yes under Host * exposes your local SSH agent socket to every remote server.",
			Remediation: "Remove 'ForwardAgent yes' from Host * in ~/.ssh/config (enable only per trusted bastion host if needed).",
			Impact:      "A root user on any remote host can authenticate elsewhere using your local agent.",
			ItemPath:    sshCfgPath,
		})
	}
}

func hasWildcardForwardAgent(content string) bool {
	inWildcardHost := false
	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if hashIdx := strings.IndexByte(line, '#'); hashIdx != -1 {
			line = strings.TrimSpace(line[:hashIdx])
		}
		fields := strings.Fields(strings.ReplaceAll(line, "=", " "))
		if len(fields) < 2 {
			continue
		}
		key := strings.ToLower(fields[0])
		switch key {
		case "host":
			inWildcardHost = slices.Contains(fields[1:], "*")
		case "match":
			inWildcardHost = false
		case "forwardagent":
			if inWildcardHost && strings.EqualFold(fields[1], "yes") {
				return true
			}
		}
	}
	return false
}

func (sa *SecurityAuditor) auditSSHRCFiles(report *SecurityAuditReport) {
	if sa.homeDir == "" {
		return
	}
	rcCandidates := []string{
		".bashrc",
		".zshrc",
		".base.zsh",
		".tools.sh",
		".base.bash",
		".zshenv",
		".profile",
	}
	for _, rel := range rcCandidates {
		p := filepath.Join(sa.homeDir, rel)
		raw, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		scanner := bufio.NewScanner(strings.NewReader(string(raw)))
		for lineNo := 1; scanner.Scan(); lineNo++ {
			trimmed := strings.TrimSpace(scanner.Text())
			if trimmed == "" || strings.HasPrefix(trimmed, "#") {
				continue
			}
			if sshAgentManualSpawnPattern.MatchString(trimmed) {
				report.Findings = append(report.Findings, SecurityFinding{
					ID:          "SEC-SSH-RC-SPAWN",
					Category:    CategoryAuth,
					Severity:    SeverityMedium,
					Title:       fmt.Sprintf("Manual ssh-agent startup detected in %s:%d", rel, lineNo),
					Description: fmt.Sprintf("Found legacy agent spawn %q in %s.", trimmed, p),
					Remediation: fmt.Sprintf("Remove manual ssh-agent startup from %s and run 'anx apply' to use the deterministic OS socket.", p),
					Impact:      "Spawns orphaned per-shell ssh-agent processes and fragments cached keys.",
					ItemPath:    p,
				})
				break
			}
		}
	}
}

func (sa *SecurityAuditor) auditSSHAgentSocketAndConnectivity(ctx context.Context, report *SecurityAuditReport) {
	if sa.getenv == nil {
		return
	}
	sock := strings.TrimSpace(sa.getenv("SSH_AUTH_SOCK"))
	xdg := strings.TrimSpace(sa.getenv("XDG_RUNTIME_DIR"))

	switch sa.goos {
	case "linux":
		lowerSock := strings.ToLower(sock)
		switch {
		case strings.Contains(sock, "/gcr/") || strings.Contains(sock, "/keyring/") || strings.Contains(lowerSock, "gnome-keyring"):
			report.Findings = append(report.Findings, SecurityFinding{
				ID:          "SEC-SSH-AGENT-HIJACK",
				Category:    CategoryAuth,
				Severity:    SeverityMedium,
				Title:       "SSH_AUTH_SOCK is overridden by gnome-keyring or gcr-ssh-agent",
				Description: fmt.Sprintf("SSH_AUTH_SOCK=%q bypasses systemd --user OpenSSH agent socket.", sock),
				Remediation: "Disable gcr-ssh-agent/gnome-keyring SSH wrapper or run 'anx apply' to propagate XDG_RUNTIME_DIR/ssh-agent.socket.",
				ItemPath:    sock,
			})
		case sock == "":
			report.Findings = append(report.Findings, SecurityFinding{
				ID:          "SEC-SSH-SOCK-UNSET",
				Category:    CategoryAuth,
				Severity:    SeverityMedium,
				Title:       "SSH_AUTH_SOCK is not set in current environment",
				Description: "SSH client and non-interactive tools cannot locate the persistent user ssh-agent socket.",
				Remediation: "Run 'anx apply' (or 'systemctl --user enable --now ssh-agent.service') and reload your session.",
			})
		case xdg != "" && sock != filepath.Join(xdg, "ssh-agent.socket"):
			expected := filepath.Join(xdg, "ssh-agent.socket")
			report.Findings = append(report.Findings, SecurityFinding{
				ID:          "SEC-SSH-SOCK-MISMATCH",
				Category:    CategoryAuth,
				Severity:    SeverityMedium,
				Title:       "SSH_AUTH_SOCK does not match deterministic OpenSSH socket",
				Description: fmt.Sprintf("Expected %q, found %q.", expected, sock),
				Remediation: "Re-sync environment.d / .zshenv / .profile with 'anx sync terminal' and reload your shell.",
				ItemPath:    sock,
			})
		}
	case "darwin":
		if sock == "" {
			report.Findings = append(report.Findings, SecurityFinding{
				ID:          "SEC-SSH-SOCK-UNSET",
				Category:    CategoryAuth,
				Severity:    SeverityMedium,
				Title:       "SSH_AUTH_SOCK is not set (expected macOS launchd agent socket)",
				Description: "macOS launchd normally provides SSH_AUTH_SOCK automatically for native Keychain integration.",
				Remediation: "Ensure shell dotfiles do not unset SSH_AUTH_SOCK.",
			})
		} else if !strings.Contains(sock, "com.apple.launchd") {
			report.Findings = append(report.Findings, SecurityFinding{
				ID:          "SEC-SSH-SOCK-MISMATCH",
				Category:    CategoryAuth,
				Severity:    SeverityLow,
				Title:       "SSH_AUTH_SOCK overrides native macOS launchd socket",
				Description: fmt.Sprintf("Found non-launchd socket %q; macOS Keychain persistence expects the launchd listener.", sock),
				Remediation: "Ensure ~/.zshenv and ~/.profile guard Linux-only SSH_AUTH_SOCK overrides with uname.",
				ItemPath:    sock,
			})
		}
	}

	if sa.lookupBin == nil || sa.runCmd == nil {
		return
	}
	if _, err := sa.lookupBin("ssh-add"); err != nil {
		return
	}

	code, _, errOut, _ := sa.runCmd(ctx, "ssh-add", "-l")
	switch code {
	case 0:
		// Agent active with loaded keys.
	case 1:
		hint := fmt.Sprintf("ssh-add %s", PreferredSSHKeyHint(sa.homeDir))
		if sa.goos == "darwin" {
			hint = fmt.Sprintf("ssh-add --apple-use-keychain %s", PreferredSSHKeyHint(sa.homeDir))
		}
		report.Findings = append(report.Findings, SecurityFinding{
			ID:          "SEC-SSH-NO-KEYS",
			Category:    CategoryAuth,
			Severity:    SeverityInfo,
			Title:       "SSH agent is active with no loaded identities (ssh-add -l exit 1)",
			Description: "The SSH agent is reachable, but no private keys are currently loaded into memory/Keychain.",
			Remediation: fmt.Sprintf("Load a key non-automatically when ready: %s", hint),
		})
	default:
		remediationCmd := ""
		if sa.goos == "linux" && xdg != "" {
			remediationCmd = "systemctl --user restart ssh-agent.service"
		}
		report.Findings = append(report.Findings, SecurityFinding{
			ID:                 "SEC-SSH-AGENT-UNREACHABLE",
			Category:           CategoryAuth,
			Severity:           SeverityHigh,
			Title:              fmt.Sprintf("SSH agent connection failed (ssh-add -l exit %d)", code),
			Description:        fmt.Sprintf("Unable to reach ssh-agent at SSH_AUTH_SOCK=%q (%s).", sock, strings.TrimSpace(errOut)),
			Remediation:        "Start or restart the user ssh-agent service (Linux: systemctl --user enable --now ssh-agent.service; macOS: verify launchd socket).",
			RemediationCommand: remediationCmd,
			Impact:             "SSH, git over SSH, and agent subprocesses cannot authenticate without manual agent setup.",
			ItemPath:           sock,
		})
	}
}

// PreferredSSHKeyHint returns the first detected default private key path or a readable placeholder.
func PreferredSSHKeyHint(home string) string {
	if home == "" {
		return "<key>"
	}
	sshDir := filepath.Join(home, ".ssh")
	for _, candidate := range []string{"id_ed25519", "id_ecdsa", "id_rsa"} {
		p := filepath.Join(sshDir, candidate)
		if info, err := os.Stat(p); err == nil && !info.IsDir() {
			return p
		}
	}
	return "~/.ssh/<key>"
}
