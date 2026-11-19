package security_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/arturonavax/env/internal/security"
)

func TestSecurityAuditorProfilesAndFindings(t *testing.T) {
	procDir := t.TempDir()
	etcDir := t.TempDir()
	homeDir := t.TempDir()

	// 1. Setup mock sysctl in proc
	aslrPath := filepath.Join(procDir, "sys", "kernel", "randomize_va_space")
	_ = os.MkdirAll(filepath.Dir(aslrPath), 0o755)
	_ = os.WriteFile(aslrPath, []byte("0\n"), 0o644) // disabled ASLR -> High finding

	dmesgPath := filepath.Join(procDir, "sys", "kernel", "dmesg_restrict")
	_ = os.WriteFile(dmesgPath, []byte("0\n"), 0o644)

	// 2. Setup mock etc/passwd with duplicate UID 0
	_ = os.WriteFile(filepath.Join(etcDir, "passwd"), []byte("root:x:0:0:root:/root:/bin/bash\nbackdoor:x:0:0:evil:/root:/bin/bash\n"), 0o644)

	// 3. Setup mock SSH private key with 0644 in homeDir
	sshDir := filepath.Join(homeDir, ".ssh")
	_ = os.MkdirAll(sshDir, 0o700)
	_ = os.WriteFile(filepath.Join(sshDir, "id_ed25519"), []byte("-----BEGIN OPENSSH PRIVATE KEY-----\nfake\n"), 0o644) // 0644 -> High finding

	// Test Workstation profile
	auditorWorkstation := security.NewSecurityAuditor(
		security.WithAuditorProfile("workstation"),
		security.WithAuditorRoots(procDir, "", etcDir),
		security.WithAuditorHome(homeDir),
	)

	rep, err := auditorWorkstation.Audit(context.Background())
	if err != nil {
		t.Fatalf("Audit failed: %v", err)
	}

	if rep.Profile != "workstation" {
		t.Errorf("expected profile workstation, got %s", rep.Profile)
	}
	if rep.HardeningScore >= 100 {
		t.Errorf("expected score < 100 with findings, got %d", rep.HardeningScore)
	}

	hasASLR := false
	hasUID0 := false
	hasSSHKey := false
	for _, f := range rep.Findings {
		if f.ID == "SEC-KRNL-ASLR" {
			hasASLR = true
		}
		if f.ID == "SEC-AUTH-UID0" {
			hasUID0 = true
		}
		if f.ID == "SEC-PERM-SSHKEY" {
			hasSSHKey = true
		}
	}

	if !hasASLR {
		t.Errorf("expected SEC-KRNL-ASLR finding")
	}
	if !hasUID0 {
		t.Errorf("expected SEC-AUTH-UID0 finding")
	}
	if !hasSSHKey {
		t.Errorf("expected SEC-PERM-SSHKEY finding")
	}

	// Test offline script synthesis
	script, err := auditorWorkstation.SynthesizeScript(context.Background(), rep)
	if err != nil {
		t.Fatalf("SynthesizeScript failed: %v", err)
	}
	if !strings.HasPrefix(script, "#!/usr/bin/env bash") {
		t.Errorf("expected bash shebang, got:\n%s", script)
	}
	if !strings.Contains(script, "chmod 600") {
		t.Errorf("expected chmod 600 in remediation script, got:\n%s", script)
	}

	// Validate script passes guardrails
	violations := auditorWorkstation.ValidateScript(script)
	if len(violations) > 0 {
		t.Fatalf("synthesized script triggered guardrails: %+v", violations)
	}

	// Test dry-run execution
	msg, err := auditorWorkstation.ExecuteScript(context.Background(), script, true)
	if err != nil {
		t.Fatalf("dry-run execute failed: %v", err)
	}
	if !strings.Contains(msg, "dry-run") {
		t.Errorf("expected dry-run confirmation, got: %s", msg)
	}
}

func TestSecurityAuditorAIPromptSynthesis(t *testing.T) {
	aiCalled := false
	auditor := security.NewSecurityAuditor(
		security.WithAuditorPromptFunc(func(ctx context.Context, prompt string) (string, error) {
			aiCalled = true
			if !strings.Contains(prompt, "Profile:") {
				t.Errorf("expected Profile in AI prompt")
			}
			return "```bash\n#!/usr/bin/env bash\nset -euo pipefail\necho 'remediated'\n```", nil
		}),
	)

	rep := &security.SecurityAuditReport{
		Profile:        "server",
		HardeningScore: 60,
	}

	script, err := auditor.SynthesizeScript(context.Background(), rep)
	if err != nil {
		t.Fatalf("AI script synthesis failed: %v", err)
	}
	if !aiCalled {
		t.Fatal("expected AI prompt func to be invoked")
	}
	if !strings.Contains(script, "remediated") {
		t.Fatalf("unexpected script: %s", script)
	}
}

func TestSecurityAuditorGuardrailBlocksMaliciousScript(t *testing.T) {
	auditor := security.NewSecurityAuditor(
		security.WithAuditorPromptFunc(func(ctx context.Context, prompt string) (string, error) {
			return "```bash\n#!/usr/bin/env bash\nrm -rf /etc\n```", nil
		}),
	)

	rep := &security.SecurityAuditReport{Profile: "server"}
	_, err := auditor.SynthesizeScript(context.Background(), rep)
	if err == nil {
		t.Fatal("expected guardrail error on destructive script, got nil")
	}
}

func TestSecurityAuditorSSHAgentChecks(t *testing.T) {
	findIDs := func(rep *security.SecurityAuditReport) map[string]security.SecurityFinding {
		m := map[string]security.SecurityFinding{}
		for _, f := range rep.Findings {
			m[f.ID] = f
		}
		return m
	}

	t.Run("healthy linux ssh-agent passes all SSH checks", func(t *testing.T) {
		procDir := t.TempDir()
		etcDir := t.TempDir()
		homeDir := t.TempDir()
		sshDir := filepath.Join(homeDir, ".ssh")
		_ = os.MkdirAll(sshDir, 0o700)
		_ = os.WriteFile(filepath.Join(sshDir, "config"), []byte("Host *\n    AddKeysToAgent yes\n    IdentitiesOnly yes\n    IgnoreUnknown UseKeychain\n    UseKeychain yes\n"), 0o600)
		_ = os.WriteFile(filepath.Join(homeDir, ".zshrc"), []byte("# clean zshrc\n"), 0o644)

		aud := security.NewSecurityAuditor(
			security.WithAuditorRoots(procDir, "", etcDir),
			security.WithAuditorHome(homeDir),
			security.WithAuditorOS("linux"),
			security.WithAuditorGetenv(func(k string) string {
				switch k {
				case "XDG_RUNTIME_DIR":
					return "/run/user/1000"
				case "SSH_AUTH_SOCK":
					return "/run/user/1000/ssh-agent.socket"
				}
				return ""
			}),
			security.WithAuditorLookPath(func(bin string) (string, error) {
				if bin == "ssh-add" {
					return "/usr/bin/ssh-add", nil
				}
				return "", os.ErrNotExist
			}),
			security.WithAuditorCommandRunner(func(_ context.Context, name string, args ...string) (int, string, string, error) {
				if name == "ssh-add" && len(args) == 1 && args[0] == "-l" {
					return 0, "256 SHA256:abc key (ED25519)", "", nil
				}
				return 127, "", "", os.ErrNotExist
			}),
		)

		rep, err := aud.Audit(context.Background())
		if err != nil {
			t.Fatalf("Audit failed: %v", err)
		}
		for _, f := range rep.Findings {
			if strings.HasPrefix(f.ID, "SEC-SSH-") {
				t.Errorf("unexpected SSH finding on healthy Linux host: %+v", f)
			}
		}
	})

	t.Run("detects loose perms, ForwardAgent *, gcr hijack, broken agent, and RC spawn", func(t *testing.T) {
		procDir := t.TempDir()
		etcDir := t.TempDir()
		homeDir := t.TempDir()
		sshDir := filepath.Join(homeDir, ".ssh")
		_ = os.MkdirAll(sshDir, 0o755)
		_ = os.WriteFile(filepath.Join(sshDir, "config"), []byte("Host *\n    ForwardAgent yes\n"), 0o644)
		_ = os.WriteFile(filepath.Join(homeDir, ".bashrc"), []byte("# ok comment eval \"$(ssh-agent -s)\"\neval \"$(ssh-agent -s)\"\n"), 0o644)

		aud := security.NewSecurityAuditor(
			security.WithAuditorRoots(procDir, "", etcDir),
			security.WithAuditorHome(homeDir),
			security.WithAuditorOS("linux"),
			security.WithAuditorGetenv(func(k string) string {
				switch k {
				case "XDG_RUNTIME_DIR":
					return "/run/user/1000"
				case "SSH_AUTH_SOCK":
					return "/run/user/1000/gcr/ssh"
				}
				return ""
			}),
			security.WithAuditorLookPath(func(bin string) (string, error) {
				if bin == "ssh-add" {
					return "/usr/bin/ssh-add", nil
				}
				return "", os.ErrNotExist
			}),
			security.WithAuditorCommandRunner(func(_ context.Context, _ string, _ ...string) (int, string, string, error) {
				return 2, "", "Could not open a connection to your authentication agent.", nil
			}),
		)

		rep, err := aud.Audit(context.Background())
		if err != nil {
			t.Fatalf("Audit failed: %v", err)
		}
		byID := findIDs(rep)
		for _, id := range []string{
			"SEC-SSH-DIR-PERM",
			"SEC-SSH-CFG-PERM",
			"SEC-SSH-FORWARDAGENT",
			"SEC-SSH-AGENT-HIJACK",
			"SEC-SSH-AGENT-UNREACHABLE",
			"SEC-SSH-RC-SPAWN",
		} {
			if _, ok := byID[id]; !ok {
				t.Errorf("expected finding %q, got findings: %+v", id, rep.Findings)
			}
		}
		script, err := aud.SynthesizeScript(context.Background(), rep)
		if err != nil {
			t.Fatalf("SynthesizeScript failed: %v", err)
		}
		if violations := aud.ValidateScript(script); len(violations) > 0 {
			t.Fatalf("remediation script failed guardrails: %+v", violations)
		}
	})

	t.Run("macOS exit 1 emits non-blocking keychain hint and specific Host ForwardAgent is allowed", func(t *testing.T) {
		procDir := t.TempDir()
		etcDir := t.TempDir()
		homeDir := t.TempDir()
		sshDir := filepath.Join(homeDir, ".ssh")
		_ = os.MkdirAll(sshDir, 0o700)
		_ = os.WriteFile(filepath.Join(sshDir, "id_ed25519"), []byte("private"), 0o600)
		_ = os.WriteFile(filepath.Join(sshDir, "config"), []byte("Host bastion\n    ForwardAgent yes\n\nHost *\n    AddKeysToAgent yes\n    UseKeychain yes\n"), 0o600)

		aud := security.NewSecurityAuditor(
			security.WithAuditorRoots(procDir, "", etcDir),
			security.WithAuditorHome(homeDir),
			security.WithAuditorOS("darwin"),
			security.WithAuditorGetenv(func(k string) string {
				if k == "SSH_AUTH_SOCK" {
					return "/private/tmp/com.apple.launchd.XYZ/Listeners"
				}
				return ""
			}),
			security.WithAuditorLookPath(func(bin string) (string, error) {
				if bin == "ssh-add" {
					return "/usr/bin/ssh-add", nil
				}
				return "", os.ErrNotExist
			}),
			security.WithAuditorCommandRunner(func(_ context.Context, _ string, _ ...string) (int, string, string, error) {
				return 1, "", "The agent has no identities.", nil
			}),
		)

		rep, err := aud.Audit(context.Background())
		if err != nil {
			t.Fatalf("Audit failed: %v", err)
		}
		byID := findIDs(rep)
		if _, hasFA := byID["SEC-SSH-FORWARDAGENT"]; hasFA {
			t.Errorf("ForwardAgent under named host should not trigger Host * warning")
		}
		noKeys, ok := byID["SEC-SSH-NO-KEYS"]
		if !ok {
			t.Fatalf("expected SEC-SSH-NO-KEYS finding on exit code 1, got %+v", rep.Findings)
		}
		if !strings.Contains(noKeys.Remediation, "ssh-add --apple-use-keychain") {
			t.Errorf("expected Keychain hint in SEC-SSH-NO-KEYS, got %+v", noKeys)
		}
	})
}
