package security

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// Standard profiles for security auditing.
const (
	ProfileWorkstation = "workstation"
	ProfileServer      = "server"
	ProfileMinimal     = "minimal"
)

// Severity levels for security findings.
const (
	SeverityCritical = "CRITICAL"
	SeverityHigh     = "HIGH"
	SeverityMedium   = "MEDIUM"
	SeverityLow      = "LOW"
	SeverityInfo     = "INFO"
)

// Finding categories.
const (
	CategoryNetwork     = "network"
	CategoryAuth        = "auth"
	CategoryPermissions = "permissions"
	CategoryKernel      = "kernel"
	CategoryUpdates     = "updates"
)

// SecurityFinding describes an individual vulnerability or misconfiguration.
type SecurityFinding struct {
	ID                 string `json:"id"`
	Category           string `json:"category"`
	Severity           string `json:"severity"`
	Title              string `json:"title"`
	Description        string `json:"description"`
	Remediation        string `json:"remediation"`
	RemediationCommand string `json:"remediation_command,omitempty"`
	Impact             string `json:"impact,omitempty"`
	ItemPath           string `json:"item_path,omitempty"`
}

// SecurityAuditReport summarizes system security posture, hardening score, and findings.
type SecurityAuditReport struct {
	Profile        string            `json:"profile"`
	HardeningScore int               `json:"hardening_score"` // 0-100
	TotalChecks    int               `json:"total_checks"`
	PassedChecks   int               `json:"passed_checks"`
	Findings       []SecurityFinding `json:"findings"`
	Summary        string            `json:"summary"`
	ScannedAt      time.Time         `json:"scanned_at"`
}

// MarkdownSummary produces a human-readable, beautifully structured Markdown report.
func (r *SecurityAuditReport) MarkdownSummary() string {
	var sb strings.Builder
	sb.WriteString("# System Security Audit Report\n\n")
	fmt.Fprintf(&sb, "- Profile: **%s**\n", strings.ToUpper(r.Profile))
	fmt.Fprintf(&sb, "- Hardening Index: **%d / 100**\n", r.HardeningScore)
	fmt.Fprintf(&sb, "- Checks Passed: %d / %d\n", r.PassedChecks, r.TotalChecks)
	fmt.Fprintf(&sb, "- Findings: %d\n\n", len(r.Findings))

	if len(r.Findings) == 0 {
		sb.WriteString("✅ **All security checks passed!** System adheres strictly to the chosen profile.\n")
		return sb.String()
	}

	sb.WriteString("## Audit Findings\n\n")
	sb.WriteString("| Severity | ID | Category | Title | Remediation |\n")
	sb.WriteString("|:---|:---|:---|:---|:---|\n")
	for _, f := range r.Findings {
		fmt.Fprintf(&sb, "| **%s** | `%s` | %s | %s | %s |\n", f.Severity, f.ID, f.Category, f.Title, f.Remediation)
	}

	sb.WriteString("\n## Remediation Details\n\n")
	for i, f := range r.Findings {
		fmt.Fprintf(&sb, "### %d. [%s] %s (`%s`)\n", i+1, f.Severity, f.Title, f.ID)
		fmt.Fprintf(&sb, "- **Category:** %s\n", f.Category)
		fmt.Fprintf(&sb, "- **Description:** %s\n", f.Description)
		if f.Impact != "" {
			fmt.Fprintf(&sb, "- **Impact:** %s\n", f.Impact)
		}
		if f.ItemPath != "" {
			fmt.Fprintf(&sb, "- **Affected Target:** `%s`\n", f.ItemPath)
		}
		fmt.Fprintf(&sb, "- **Recommendation:** %s\n", f.Remediation)
		if f.RemediationCommand != "" {
			fmt.Fprintf(&sb, "- **Fix Command:**\n```bash\n%s\n```\n", f.RemediationCommand)
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// SecurityAuditorOptions configures the SecurityAuditor instance.
type SecurityAuditorOption func(*SecurityAuditor)

// SecurityPromptFunc synthesizes a remediation script from an audit prompt using LLM harness.
type SecurityPromptFunc func(ctx context.Context, prompt string) (string, error)

// SecurityAuditor performs multi-level, profile-aware system security audits.
type SecurityAuditor struct {
	profile    string
	procRoot   string
	sysRoot    string
	etcRoot    string
	homeDir    string
	runner     *Runner
	promptFunc SecurityPromptFunc
	lookupBin  func(string) (string, error)
}

// NewSecurityAuditor constructs a configured SecurityAuditor.
func NewSecurityAuditor(opts ...SecurityAuditorOption) *SecurityAuditor {
	sa := &SecurityAuditor{
		profile:   ProfileWorkstation,
		procRoot:  "/proc",
		sysRoot:   "/sys",
		etcRoot:   "/etc",
		homeDir:   os.Getenv("HOME"),
		lookupBin: exec.LookPath,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(sa)
		}
	}
	if sa.homeDir == "" {
		if h, err := os.UserHomeDir(); err == nil {
			sa.homeDir = h
		}
	}
	return sa
}

func WithAuditorProfile(profile string) SecurityAuditorOption {
	return func(sa *SecurityAuditor) {
		p := strings.ToLower(strings.TrimSpace(profile))
		if p == ProfileServer || p == ProfileMinimal || p == ProfileWorkstation {
			sa.profile = p
		}
	}
}

func WithAuditorRoots(procRoot, sysRoot, etcRoot string) SecurityAuditorOption {
	return func(sa *SecurityAuditor) {
		if procRoot != "" {
			sa.procRoot = procRoot
		}
		if sysRoot != "" {
			sa.sysRoot = sysRoot
		}
		if etcRoot != "" {
			sa.etcRoot = etcRoot
		}
	}
}

func WithAuditorHome(home string) SecurityAuditorOption {
	return func(sa *SecurityAuditor) {
		if home != "" {
			sa.homeDir = home
		}
	}
}

func WithAuditorPromptFunc(fn SecurityPromptFunc) SecurityAuditorOption {
	return func(sa *SecurityAuditor) {
		sa.promptFunc = fn
	}
}

func WithAuditorRunner(r *Runner) SecurityAuditorOption {
	return func(sa *SecurityAuditor) {
		sa.runner = r
	}
}

func WithAuditorLookPath(fn func(string) (string, error)) SecurityAuditorOption {
	return func(sa *SecurityAuditor) {
		sa.lookupBin = fn
	}
}

// Audit evaluates the host system against the selected security profile.
func (sa *SecurityAuditor) Audit(ctx context.Context) (*SecurityAuditReport, error) {
	report := &SecurityAuditReport{
		Profile:   sa.profile,
		ScannedAt: time.Now(),
		Findings:  make([]SecurityFinding, 0),
	}

	checksRun := 0
	recordCheck := func() {
		checksRun++
	}

	// 1. Kernel ASLR (randomize_va_space)
	recordCheck()
	aslrPath := filepath.Join(sa.procRoot, "sys", "kernel", "randomize_va_space")
	if data, err := os.ReadFile(aslrPath); err == nil {
		val := strings.TrimSpace(string(data))
		if val != "2" {
			report.Findings = append(report.Findings, SecurityFinding{
				ID:                 "SEC-KRNL-ASLR",
				Category:           CategoryKernel,
				Severity:           SeverityHigh,
				Title:              "ASLR memory randomization is disabled or partial",
				Description:        fmt.Sprintf("kernel.randomize_va_space is %q (expected 2)", val),
				Remediation:        "Set kernel.randomize_va_space = 2 in /etc/sysctl.d/99-security.conf",
				RemediationCommand: "sysctl -w kernel.randomize_va_space=2",
				Impact:             "Increases susceptibility to buffer overflow and ROP exploits.",
				ItemPath:           aslrPath,
			})
		}
	}

	// 2. Kernel dmesg restriction
	recordCheck()
	dmesgPath := filepath.Join(sa.procRoot, "sys", "kernel", "dmesg_restrict")
	if data, err := os.ReadFile(dmesgPath); err == nil {
		val := strings.TrimSpace(string(data))
		if val != "1" && sa.profile == ProfileServer {
			report.Findings = append(report.Findings, SecurityFinding{
				ID:                 "SEC-KRNL-DMESG",
				Category:           CategoryKernel,
				Severity:           SeverityMedium,
				Title:              "Kernel ring buffer (dmesg) is readable by unprivileged users",
				Description:        "kernel.dmesg_restrict is 0",
				Remediation:        "Set kernel.dmesg_restrict = 1 in sysctl",
				RemediationCommand: "sysctl -w kernel.dmesg_restrict=1",
				Impact:             "Unprivileged users can inspect kernel addresses and hardware logs.",
				ItemPath:           dmesgPath,
			})
		}
	}

	// 3. Kernel kptr_restrict
	recordCheck()
	kptrPath := filepath.Join(sa.procRoot, "sys", "kernel", "kptr_restrict")
	if data, err := os.ReadFile(kptrPath); err == nil {
		val := strings.TrimSpace(string(data))
		minVal := 1
		if sa.profile == ProfileServer {
			minVal = 2
		}
		intVal, _ := strconv.Atoi(val)
		if intVal < minVal {
			report.Findings = append(report.Findings, SecurityFinding{
				ID:                 "SEC-KRNL-KPTR",
				Category:           CategoryKernel,
				Severity:           SeverityMedium,
				Title:              "Kernel pointer addresses exposed in /proc",
				Description:        fmt.Sprintf("kernel.kptr_restrict is %d (expected >= %d)", intVal, minVal),
				Remediation:        fmt.Sprintf("Set kernel.kptr_restrict = %d in sysctl", minVal),
				RemediationCommand: fmt.Sprintf("sysctl -w kernel.kptr_restrict=%d", minVal),
				Impact:             "Exposes kernel memory addresses assisting privilege escalation exploits.",
				ItemPath:           kptrPath,
			})
		}
	}

	// 4. Filesystem protected hardlinks and symlinks
	recordCheck()
	for _, setting := range []string{"protected_hardlinks", "protected_symlinks"} {
		p := filepath.Join(sa.procRoot, "sys", "fs", setting)
		if data, err := os.ReadFile(p); err == nil {
			if strings.TrimSpace(string(data)) != "1" {
				report.Findings = append(report.Findings, SecurityFinding{
					ID:                 "SEC-FS-" + strings.ToUpper(setting),
					Category:           CategoryKernel,
					Severity:           SeverityHigh,
					Title:              fmt.Sprintf("Filesystem %s protection is disabled", setting),
					Description:        fmt.Sprintf("fs.%s is 0", setting),
					Remediation:        fmt.Sprintf("Set fs.%s = 1 in sysctl", setting),
					RemediationCommand: fmt.Sprintf("sysctl -w fs.%s=1", setting),
					Impact:             "Enables link-following race condition exploits in /tmp.",
					ItemPath:           p,
				})
			}
		}
	}

	// 5. Network: Firewall Status
	recordCheck()
	fwActive := false
	if sa.lookupBin != nil {
		if ufwBin, err := sa.lookupBin("ufw"); err == nil && ufwBin != "" {
			if out, rErr := exec.CommandContext(ctx, ufwBin, "status").Output(); rErr == nil {
				if strings.Contains(strings.ToLower(string(out)), "status: active") {
					fwActive = true
				}
			}
		}
		if !fwActive {
			if nftBin, err := sa.lookupBin("nft"); err == nil && nftBin != "" {
				if out, rErr := exec.CommandContext(ctx, nftBin, "list", "ruleset").Output(); rErr == nil && len(out) > 0 {
					fwActive = true
				}
			}
		}
		if !fwActive {
			if iptBin, err := sa.lookupBin("iptables"); err == nil && iptBin != "" {
				if out, rErr := exec.CommandContext(ctx, iptBin, "-L", "-n").Output(); rErr == nil && strings.Contains(string(out), "Chain") {
					fwActive = true
				}
			}
		}
	}

	if !fwActive && sa.profile != ProfileMinimal {
		sev := SeverityMedium
		if sa.profile == ProfileServer {
			sev = SeverityCritical
		}
		report.Findings = append(report.Findings, SecurityFinding{
			ID:                 "SEC-NET-FIREWALL",
			Category:           CategoryNetwork,
			Severity:           sev,
			Title:              "Host firewall is inactive or unconfigured",
			Description:        "Neither UFW, nftables, nor active iptables rules were detected.",
			Remediation:        "Enable UFW or configure an nftables baseline firewall.",
			RemediationCommand: "ufw default deny incoming && ufw default allow outgoing && ufw enable",
			Impact:             "All listening local network ports remain exposed to the network.",
		})
	}

	// 6. Network: IP Source Routing & ICMP Redirects
	if sa.profile == ProfileServer {
		recordCheck()
		srPath := filepath.Join(sa.procRoot, "sys", "net", "ipv4", "conf", "all", "accept_source_route")
		if data, err := os.ReadFile(srPath); err == nil && strings.TrimSpace(string(data)) != "0" {
			report.Findings = append(report.Findings, SecurityFinding{
				ID:                 "SEC-NET-SRV-SRCROUTE",
				Category:           CategoryNetwork,
				Severity:           SeverityHigh,
				Title:              "IPv4 source routing is accepted",
				Description:        "net.ipv4.conf.all.accept_source_route is 1",
				Remediation:        "Set net.ipv4.conf.all.accept_source_route = 0",
				RemediationCommand: "sysctl -w net.ipv4.conf.all.accept_source_route=0",
				Impact:             "Permits malicious packet routing and spoofing.",
				ItemPath:           srPath,
			})
		}
	}

	// 7. Auth: Duplicate UID 0 accounts in /etc/passwd
	recordCheck()
	passwdPath := filepath.Join(sa.etcRoot, "passwd")
	if f, err := os.Open(passwdPath); err == nil {
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			line := scanner.Text()
			fields := strings.Split(line, ":")
			if len(fields) >= 3 {
				username := fields[0]
				uid := fields[2]
				if uid == "0" && username != "root" {
					report.Findings = append(report.Findings, SecurityFinding{
						ID:          "SEC-AUTH-UID0",
						Category:    CategoryAuth,
						Severity:    SeverityCritical,
						Title:       fmt.Sprintf("Non-root user with UID 0 detected (%s)", username),
						Description: fmt.Sprintf("Account %s has superuser privileges (UID 0)", username),
						Remediation: fmt.Sprintf("Remove or change UID of account %s", username),
						Impact:      "Unmonitored full root access to the entire operating system.",
						ItemPath:    passwdPath,
					})
				}
			}
		}
		_ = f.Close()
	}

	// 8. Auth: Sudoers NOPASSWD entries
	recordCheck()
	sudoersPath := filepath.Join(sa.etcRoot, "sudoers")
	sa.auditSudoersFile(sudoersPath, report)
	sudoersDDir := filepath.Join(sa.etcRoot, "sudoers.d")
	if entries, err := os.ReadDir(sudoersDDir); err == nil {
		for _, e := range entries {
			if !e.IsDir() && !strings.HasSuffix(e.Name(), "~") && !strings.HasPrefix(e.Name(), ".") {
				sa.auditSudoersFile(filepath.Join(sudoersDDir, e.Name()), report)
			}
		}
	}

	// 9. SSH Configuration audit
	recordCheck()
	sshdConfigPath := filepath.Join(sa.etcRoot, "ssh", "sshd_config")
	if data, err := os.ReadFile(sshdConfigPath); err == nil {
		content := string(data)
		permitRoot := regexp.MustCompile(`(?i)^\s*PermitRootLogin\s+(yes|prohibit-password|without-password)\b`)
		if permitRoot.MatchString(content) && sa.profile == ProfileServer {
			report.Findings = append(report.Findings, SecurityFinding{
				ID:                 "SEC-SSH-ROOTLOGIN",
				Category:           CategoryAuth,
				Severity:           SeverityHigh,
				Title:              "SSH allows direct root login",
				Description:        "PermitRootLogin is enabled in sshd_config",
				Remediation:        "Set PermitRootLogin no in /etc/ssh/sshd_config",
				RemediationCommand: "sed -i 's/^\\s*PermitRootLogin.*/PermitRootLogin no/' /etc/ssh/sshd_config",
				Impact:             "Root account is exposed to automated brute-force attacks.",
				ItemPath:           sshdConfigPath,
			})
		}

		emptyPass := regexp.MustCompile(`(?i)^\s*PermitEmptyPasswords\s+yes\b`)
		if emptyPass.MatchString(content) {
			report.Findings = append(report.Findings, SecurityFinding{
				ID:                 "SEC-SSH-EMPTYPASS",
				Category:           CategoryAuth,
				Severity:           SeverityCritical,
				Title:              "SSH allows empty passwords",
				Description:        "PermitEmptyPasswords yes is set in sshd_config",
				Remediation:        "Set PermitEmptyPasswords no in /etc/ssh/sshd_config",
				RemediationCommand: "sed -i 's/^\\s*PermitEmptyPasswords.*/PermitEmptyPasswords no/' /etc/ssh/sshd_config",
				Impact:             "Anyone can log into accounts without setting a password.",
				ItemPath:           sshdConfigPath,
			})
		}
	}

	// 10. Permissions: SSH Private Keys in user directory with loose permissions
	recordCheck()
	if sa.homeDir != "" {
		sshDir := filepath.Join(sa.homeDir, ".ssh")
		if entries, err := os.ReadDir(sshDir); err == nil {
			for _, e := range entries {
				if e.IsDir() {
					continue
				}
				name := strings.ToLower(e.Name())
				if strings.HasPrefix(name, "id_") && !strings.HasSuffix(name, ".pub") {
					p := filepath.Join(sshDir, e.Name())
					if info, err := e.Info(); err == nil {
						perm := info.Mode().Perm()
						if perm&0o077 != 0 {
							report.Findings = append(report.Findings, SecurityFinding{
								ID:                 "SEC-PERM-SSHKEY",
								Category:           CategoryPermissions,
								Severity:           SeverityHigh,
								Title:              fmt.Sprintf("SSH private key %s has overly permissive permissions (%04o)", e.Name(), perm),
								Description:        fmt.Sprintf("File %s is readable by group or other users", p),
								Remediation:        fmt.Sprintf("chmod 600 %s", p),
								RemediationCommand: fmt.Sprintf("chmod 600 %q", p),
								Impact:             "Local users on the machine can copy private authentication credentials.",
								ItemPath:           p,
							})
						}
					}
				}
			}
		}
	}

	// 11. Permissions: World-writable files in /etc
	recordCheck()
	if sa.etcRoot != "" {
		_ = filepath.WalkDir(sa.etcRoot, func(p string, d fs.DirEntry, walkErr error) error {
			if walkErr != nil || d.IsDir() {
				return nil
			}
			if info, err := d.Info(); err == nil {
				if info.Mode()&0o002 != 0 {
					report.Findings = append(report.Findings, SecurityFinding{
						ID:                 "SEC-PERM-ETCWRITE",
						Category:           CategoryPermissions,
						Severity:           SeverityCritical,
						Title:              fmt.Sprintf("World-writable configuration file in /etc (%s)", d.Name()),
						Description:        fmt.Sprintf("File %s has mode %04o", p, info.Mode().Perm()),
						Remediation:        fmt.Sprintf("chmod o-w %s", p),
						RemediationCommand: fmt.Sprintf("chmod o-w %q", p),
						Impact:             "Any local unprivileged user or process can tamper with system settings.",
						ItemPath:           p,
					})
				}
			}
			return nil
		})
	}

	// 12. External Cybersecurity Tools: Lynis & ss socket inspection
	recordCheck()
	sa.auditLynis(ctx, report)
	recordCheck()
	sa.auditListeningSockets(ctx, report)

	// Calculate Hardening Index
	score := 100
	for _, f := range report.Findings {
		switch f.Severity {
		case SeverityCritical:
			score -= 25
		case SeverityHigh:
			score -= 15
		case SeverityMedium:
			score -= 8
		case SeverityLow:
			score -= 3
		}
	}
	if score < 0 {
		score = 0
	}
	report.HardeningScore = score
	report.TotalChecks = checksRun
	report.PassedChecks = checksRun - len(report.Findings)
	if report.PassedChecks < 0 {
		report.PassedChecks = 0
	}

	return report, nil
}

func (sa *SecurityAuditor) auditSudoersFile(path string, report *SecurityAuditReport) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "#") {
			continue
		}
		if strings.Contains(line, "NOPASSWD: ALL") || strings.Contains(line, "NOPASSWD:ALL") {
			sev := SeverityMedium
			if sa.profile == ProfileServer {
				sev = SeverityHigh
			}
			report.Findings = append(report.Findings, SecurityFinding{
				ID:          "SEC-AUTH-NOPASSWD",
				Category:    CategoryAuth,
				Severity:    sev,
				Title:       "Passwordless sudo (NOPASSWD: ALL) detected",
				Description: fmt.Sprintf("Line in %s grants unconditional passwordless root execution: %s", filepath.Base(path), line),
				Remediation: "Require password for sudo elevation or restrict commands to specific binaries.",
				Impact:      "Compromised unprivileged user session immediately translates into full root access.",
				ItemPath:    path,
			})
		}
	}
}

// auditLynis runs lynis system audit if available and extracts high-priority findings.
func (sa *SecurityAuditor) auditLynis(ctx context.Context, report *SecurityAuditReport) {
	if sa.lookupBin == nil {
		return
	}
	lynisBin, err := sa.lookupBin("lynis")
	if err != nil || lynisBin == "" {
		return
	}

	datPath := filepath.Join(sa.etcRoot, "..", "var", "log", "lynis-report.dat")
	if data, err := os.ReadFile(datPath); err == nil {
		sa.parseLynisReportDat(data, report)
		return
	}

	if sa.runner != nil {
		out, err := sa.runner.Run(ctx, lynisBin, "audit", "system", "--quick", "--auditor", "anx", "--cronjob")
		if err == nil || len(out) > 0 {
			sa.parseLynisOutput(out, report)
		}
	}
}

func (sa *SecurityAuditor) parseLynisReportDat(data []byte, report *SecurityAuditReport) {
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "warning[]=") {
			warn := strings.TrimPrefix(line, "warning[]=")
			parts := strings.SplitN(warn, "|", 3)
			title := warn
			if len(parts) > 1 {
				title = parts[1]
			}
			report.Findings = append(report.Findings, SecurityFinding{
				ID:          "LYNIS-WARN",
				Category:    "Lynis",
				Severity:    SeverityHigh,
				Title:       "Lynis warning: " + title,
				Description: "Lynis automated compliance scan warning from /var/log/lynis-report.dat",
				Remediation: "Inspect Lynis log at /var/log/lynis.log",
				Impact:      "Host configuration violates standard CIS/Lynis hardening checks.",
			})
		}
	}
}

func (sa *SecurityAuditor) parseLynisOutput(output string, report *SecurityAuditReport) {
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "Warning:") || strings.HasPrefix(line, "! ") {
			report.Findings = append(report.Findings, SecurityFinding{
				ID:          "LYNIS-WARN",
				Category:    "Lynis",
				Severity:    SeverityHigh,
				Title:       line,
				Description: "Lynis automated compliance scan warning",
				Remediation: "Review Lynis audit log in /var/log/lynis.log",
				Impact:      "Potential host security weakness identified by Lynis.",
			})
		}
	}
}

// auditListeningSockets inspects network sockets via ss or netstat.
func (sa *SecurityAuditor) auditListeningSockets(ctx context.Context, report *SecurityAuditReport) {
	if sa.lookupBin == nil || sa.runner == nil {
		return
	}
	ssBin, err := sa.lookupBin("ss")
	if err != nil || ssBin == "" {
		return
	}

	out, err := sa.runner.Run(ctx, ssBin, "-tulpn")
	if err != nil || len(out) == 0 {
		return
	}

	scanner := bufio.NewScanner(strings.NewReader(out))
	unconfinedListeners := 0
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.Contains(line, "0.0.0.0:") || strings.Contains(line, "*:") || strings.Contains(line, ":::*") {
			if !strings.Contains(line, "127.0.0.1") && !strings.Contains(line, "[::1]") {
				unconfinedListeners++
			}
		}
	}

	if unconfinedListeners > 0 && sa.profile == ProfileServer {
		report.Findings = append(report.Findings, SecurityFinding{
			ID:          "SEC-NET-UNCONFINED",
			Category:    CategoryNetwork,
			Severity:    SeverityHigh,
			Title:       fmt.Sprintf("%d unconfined public network socket listeners detected", unconfinedListeners),
			Description: "Active network daemons listening globally on 0.0.0.0 or ::: without host-only binding.",
			Remediation: "Bind daemons to 127.0.0.1 or enforce strict ingress firewall policies.",
			Impact:      "Services exposed to unsolicited external network probes and potential exploitation.",
		})
	}
}

// SynthesizeScript generates an idempotent shell remediation script either via AI or offline engine.
func (sa *SecurityAuditor) SynthesizeScript(ctx context.Context, report *SecurityAuditReport) (string, error) {
	if report == nil {
		return "", errors.New("audit report cannot be nil")
	}

	if sa.promptFunc != nil {
		prompt := fmt.Sprintf(`You are an expert security engineer. Generate a production-ready, idempotent Bash remediation script based on this system security audit report.

Profile: %s
Hardening Score: %d / 100
Findings:
%s

Requirements:
1. Output ONLY the bash script enclosed in a single `+"```bash"+` code block.
2. Must start with #!/usr/bin/env bash and set -euo pipefail.
3. Every remediation step must be commented with the Finding ID.
4. Commands must be idempotent (safe to re-run multiple times).
5. Do NOT include destructive rm -rf commands or wipe configurations.
`, report.Profile, report.HardeningScore, report.MarkdownSummary())

		raw, err := sa.promptFunc(ctx, prompt)
		if err != nil {
			return "", fmt.Errorf("AI script synthesis failed: %w", err)
		}
		script := extractCodeBlock(raw)
		if violations := sa.ValidateScript(script); len(violations) > 0 {
			return "", formatGuardrailError(violations)
		}
		return script, nil
	}

	// Deterministic offline remediation generator
	var sb strings.Builder
	sb.WriteString("#!/usr/bin/env bash\n")
	sb.WriteString("# ANX Security Audit Remediation Script\n")
	fmt.Fprintf(&sb, "# Generated for profile: %s (Hardening Score: %d/100)\n", report.Profile, report.HardeningScore)
	sb.WriteString("set -euo pipefail\n\n")

	if len(report.Findings) == 0 {
		sb.WriteString("echo '==> No security remediations required. System is fully hardened.'\n")
		return sb.String(), nil
	}

	sb.WriteString("echo '==> Applying ANX Security Audit Remediations...'\n\n")

	appliedCmds := make(map[string]bool)
	for _, f := range report.Findings {
		if f.RemediationCommand == "" || appliedCmds[f.RemediationCommand] {
			continue
		}
		appliedCmds[f.RemediationCommand] = true

		fmt.Fprintf(&sb, "# [%s] %s (%s)\n", f.Severity, f.Title, f.ID)
		fmt.Fprintf(&sb, "echo '--> Remediating %s...'\n", f.ID)
		sb.WriteString(f.RemediationCommand + "\n\n")
	}

	sb.WriteString("echo '==> All security remediations applied successfully.'\n")
	script := sb.String()

	if violations := sa.ValidateScript(script); len(violations) > 0 {
		return "", formatGuardrailError(violations)
	}

	return script, nil
}

// ValidateScript runs static security guardrails against a synthesized remediation script.
func (sa *SecurityAuditor) ValidateScript(script string) []GuardrailViolation {
	return ValidateScriptGuardrails(script)
}

// ExecuteScript validates guardrails and runs the script via bash.
func (sa *SecurityAuditor) ExecuteScript(ctx context.Context, script string, dryRun bool) (string, error) {
	if strings.TrimSpace(script) == "" {
		return "", errors.New("security remediation script is empty")
	}

	if violations := sa.ValidateScript(script); len(violations) > 0 {
		return "", formatGuardrailError(violations)
	}

	if dryRun {
		return "[dry-run] Security script validated successfully against all guardrails. Zero mutations performed.", nil
	}

	bashBin := "bash"
	if sa.lookupBin != nil {
		if b, err := sa.lookupBin("bash"); err == nil && b != "" {
			bashBin = b
		}
	}

	tmpDir, err := os.MkdirTemp("", "anx-sec-remediate-*")
	if err != nil {
		return "", fmt.Errorf("creating temp directory: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	scriptPath := filepath.Join(tmpDir, "remediate.sh")
	if err := os.WriteFile(scriptPath, []byte(script), 0o700); err != nil {
		return "", fmt.Errorf("writing remediation script: %w", err)
	}

	if runtime.GOOS != "windows" {
		cmdSyntax := exec.CommandContext(ctx, bashBin, "-n", scriptPath)
		if out, sErr := cmdSyntax.CombinedOutput(); sErr != nil {
			return "", fmt.Errorf("bash syntax check failed: %s (%w)", string(out), sErr)
		}
	}

	cmdExec := exec.CommandContext(ctx, bashBin, scriptPath)
	out, err := cmdExec.CombinedOutput()
	if err != nil {
		return string(out), fmt.Errorf("executing remediation script: %w", err)
	}

	return string(out), nil
}

func extractCodeBlock(raw string) string {
	raw = strings.TrimSpace(raw)
	if strings.HasPrefix(raw, "```") {
		lines := strings.Split(raw, "\n")
		if len(lines) >= 2 {
			end := len(lines) - 1
			for i := len(lines) - 1; i >= 1; i-- {
				if strings.HasPrefix(strings.TrimSpace(lines[i]), "```") {
					end = i
					break
				}
			}
			return strings.Join(lines[1:end], "\n") + "\n"
		}
	}
	return raw
}

func formatGuardrailError(violations []GuardrailViolation) error {
	var msgs []string
	for _, v := range violations {
		msgs = append(msgs, fmt.Sprintf("line %d [%s]: %s", v.Line, v.RuleID, v.Description))
	}
	return fmt.Errorf("security guardrail violation: %s", strings.Join(msgs, "; "))
}
