package scriptgen_test

import (
	"strings"
	"testing"

	"github.com/arturonavax/env/internal/domain"
	"github.com/arturonavax/env/internal/ops"
	"github.com/arturonavax/env/internal/platform"
	"github.com/arturonavax/env/internal/scriptgen"
	"github.com/arturonavax/env/internal/security"
)

func TestAuditableScriptBuilderRender(t *testing.T) {
	plat := platform.Detect()
	b := scriptgen.NewBuilder("test", plat)
	b.AddSection("Test Section", "Testing description")
	b.AddCommand("echo 'hello'", "Greeting comment")

	script := b.Render()

	if !strings.Contains(script, "#!/usr/bin/env bash") {
		t.Errorf("missing bash shebang")
	}
	if !strings.Contains(script, "AUDIT & ARCHITECTURAL BOUNDARY NOTICE") {
		t.Errorf("missing audit boundary notice")
	}
	if !strings.Contains(script, "IN-PROCESS GO RUNTIME OPERATIONS EXCLUDED") {
		t.Errorf("missing in-process Go exclusion disclaimer")
	}
	if !strings.Contains(script, "echo 'hello'") {
		t.Errorf("missing command in script")
	}
	if !strings.Contains(script, "Greeting comment") {
		t.Errorf("missing comment in script")
	}
}

func TestTranslators(t *testing.T) {
	plat := platform.Detect()

	t.Run("TranslatePlan", func(t *testing.T) {
		plan := &domain.ExecutionPlan{
			Steps: []domain.PlanStep{
				{
					Action:      domain.ActionInstall,
					ResourceID:  "ripgrep",
					Provider:    "native",
					Description: "Fast search tool",
				},
			},
		}
		script := scriptgen.TranslatePlan(plan, plat)
		if !strings.Contains(script, "Installing package ripgrep") {
			t.Errorf("expected plan command in script, got:\n%s", script)
		}
	})

	t.Run("TranslateCleanup", func(t *testing.T) {
		script := scriptgen.TranslateCleanup([]string{"/tmp/orphan.log"}, plat)
		if !strings.Contains(script, `rm -rf "/tmp/orphan.log"`) {
			t.Errorf("expected cleanup command in script, got:\n%s", script)
		}
	})

	t.Run("TranslateRebuild", func(t *testing.T) {
		res := []ops.RebuildResult{
			{
				Scope:    "go",
				Commands: []string{"go install example.com/tool@latest"},
			},
		}
		script := scriptgen.TranslateRebuild(res, plat)
		if !strings.Contains(script, "go install example.com/tool@latest") {
			t.Errorf("expected rebuild command in script, got:\n%s", script)
		}
	})

	t.Run("TranslateSecurityAudit", func(t *testing.T) {
		rep := &security.SecurityAuditReport{
			Profile:        security.ProfileWorkstation,
			HardeningScore: 90,
			Findings: []security.SecurityFinding{
				{
					ID:                 "SEC-01",
					Severity:           security.SeverityHigh,
					Title:              "Test Finding",
					RemediationCommand: "sysctl -w net.ipv4.ip_forward=0",
				},
			},
		}
		script := scriptgen.TranslateSecurityAudit(rep, plat)
		if !strings.Contains(script, "sysctl -w net.ipv4.ip_forward=0") {
			t.Errorf("expected audit remediation command in script, got:\n%s", script)
		}
	})
}
