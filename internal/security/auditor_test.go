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
