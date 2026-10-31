package security_test

import (
	"testing"

	"github.com/arturonavax/env/internal/security"
)

func TestScanForSecrets(t *testing.T) {
	cleanContent := `
[profile]
theme = "tokyo-night"
editor = "nvim"
`
	if findings := security.ScanForSecrets(cleanContent); len(findings) > 0 {
		t.Fatalf("expected 0 findings for clean content, got %d", len(findings))
	}

	awsContent := `
AWS_ACCESS_KEY_ID = AKIAIOSFODNN7EXAMPLE
`
	findings := security.ScanForSecrets(awsContent)
	if len(findings) != 1 || findings[0].RuleID != "SEC-AWSKEY" {
		t.Fatalf("expected SEC-AWSKEY finding, got: %+v", findings)
	}

	privKeyContent := `
-----BEGIN OPENSSH PRIVATE KEY-----
b3BlbnNzaC1rZXktdjEAAAAABG5vbmUAAAAEbm9uZQAAAAAAAAABAAABlwAAAAdzc2gtcn
-----END OPENSSH PRIVATE KEY-----
`
	findings = security.ScanForSecrets(privKeyContent)
	if len(findings) == 0 || findings[0].RuleID != "SEC-PRIVKEY" {
		t.Fatalf("expected SEC-PRIVKEY finding, got: %+v", findings)
	}
}
