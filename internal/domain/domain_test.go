package domain_test

import (
	"errors"
	"testing"
	"time"

	"github.com/arturonavax/env/internal/domain"
)

func TestDomainHermetic(t *testing.T) {
	var zeroStep domain.PlanStep
	if zeroStep.Status != domain.StepUnknown || zeroStep.Status.String() != "unknown" {
		t.Fatalf("expected zero-value StepStatus to be StepUnknown ('unknown'), got %v (%q)", zeroStep.Status, zeroStep.Status.String())
	}

	res := domain.ManagedResource{
		ID:       "ripgrep",
		Name:     "ripgrep",
		Category: "cli-tools",
		Status:   domain.StatusInstalled,
		Health:   domain.HealthHealthy,
	}

	if res.Status.String() != "installed" {
		t.Fatalf("expected 'installed', got '%s'", res.Status.String())
	}
	if res.Health.String() != "healthy" {
		t.Fatalf("expected 'healthy', got '%s'", res.Health.String())
	}

	step := domain.PlanStep{
		ID:          1,
		ResourceID:  "ripgrep",
		Action:      domain.ActionInstall,
		Status:      domain.StepPending,
		Description: "Install ripgrep via binary",
	}

	plan := domain.ExecutionPlan{
		ID:        "plan-01",
		CreatedAt: time.Now(),
		Steps:     []domain.PlanStep{step},
	}

	if plan.StepCount() != 1 {
		t.Fatalf("expected step count 1, got %d", plan.StepCount())
	}
	if plan.HasDestructiveSteps() {
		t.Fatalf("expected no destructive steps")
	}

	tx := domain.Transaction{
		ID:    "tx-01",
		State: domain.TxCommitted,
	}
	if tx.State.String() != "committed" {
		t.Fatalf("expected 'committed', got '%s'", tx.State.String())
	}

	cfg := domain.ConfigurationResource{
		ID:     "vscode-settings",
		Target: "~/.config/Code/User/settings.json",
		TargetsByOS: map[string]string{
			"darwin": "~/Library/Application Support/Code/User/settings.json",
		},
		SupportedOS: []string{"linux", "darwin"},
	}
	if got := cfg.TargetForOS("darwin"); got != "~/Library/Application Support/Code/User/settings.json" {
		t.Fatalf("unexpected darwin target: %q", got)
	}
	if cfg.SupportsOS("windows") {
		t.Fatal("expected windows to be unsupported for cfg")
	}

	errWrapped := domain.NewExitError(domain.ExitConflictOrAudit, errors.New("audit failed"))
	if code := domain.ResolveExitCode(errWrapped); code != domain.ExitConflictOrAudit {
		t.Fatalf("expected exit code %d, got %d", domain.ExitConflictOrAudit, code)
	}
	if code := domain.ResolveExitCode(nil); code != domain.ExitSuccess {
		t.Fatalf("expected exit code 0, got %d", code)
	}
}
