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

func TestPlanSpecDeterministicDigest(t *testing.T) {
	// Plan A created at time T1 with ID A
	planA := domain.ExecutionPlan{
		ID:        "plan-12345",
		CreatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		Platform:  "linux",
		Profile:   "developer",
		Steps: []domain.PlanStep{
			{ID: 1, ResourceID: "ripgrep", Action: domain.ActionInstall, Description: "Install ripgrep"},
			{ID: 2, ResourceID: "ghostty", Action: domain.ActionInstall, Description: "Install ghostty"},
		},
	}

	// Plan B created at time T2 with ID B, but identical canonical content
	planB := domain.ExecutionPlan{
		ID:        "plan-99999",
		CreatedAt: time.Date(2026, 10, 30, 18, 0, 0, 0, time.UTC),
		Platform:  "linux",
		Profile:   "developer",
		Steps: []domain.PlanStep{
			{ID: 1, ResourceID: "ripgrep", Action: domain.ActionInstall, Description: "Install ripgrep"},
			{ID: 2, ResourceID: "ghostty", Action: domain.ActionInstall, Description: "Install ghostty"},
		},
	}

	digestA := planA.Digest()
	digestB := planB.Digest()

	if digestA == "" {
		t.Fatalf("expected non-empty digest for planA")
	}
	if digestA != digestB {
		t.Fatalf("AC-03 violation: expected identical digest for identical plans regardless of ID/time, got %q vs %q", digestA, digestB)
	}

	// Plan C with modified action must produce different digest
	planC := planB
	planC.Steps = []domain.PlanStep{
		{ID: 1, ResourceID: "ripgrep", Action: domain.ActionInstall, Description: "Install ripgrep"},
		{ID: 2, ResourceID: "bat", Action: domain.ActionInstall, Description: "Install bat"},
	}
	digestC := planC.Digest()
	if digestA == digestC {
		t.Fatalf("expected different digest when actions differ, got identical %q", digestA)
	}

	// Canonical PlanSpec with metadata
	spec1 := &domain.PlanSpec{
		SchemaVersion: "v1",
		Platform:      "linux",
		Profile:       "developer",
		Actions: []domain.PlanAction{
			{ID: 1, ResourceID: "ripgrep", Action: domain.ActionInstall, Preconditions: []string{"b", "a"}},
		},
		Metadata: map[string]string{"foo": "1", "bar": "2"},
	}
	spec2 := &domain.PlanSpec{
		SchemaVersion: "v1",
		Platform:      "linux",
		Profile:       "developer",
		Actions: []domain.PlanAction{
			{ID: 1, ResourceID: "ripgrep", Action: domain.ActionInstall, Preconditions: []string{"a", "b"}},
		},
		Metadata: map[string]string{"bar": "2", "foo": "1"},
	}

	d1, err1 := spec1.ComputeDigest()
	d2, err2 := spec2.ComputeDigest()
	if err1 != nil || err2 != nil {
		t.Fatalf("unexpected error computing digest: %v / %v", err1, err2)
	}
	if d1 != d2 {
		t.Fatalf("expected identical digest for canonical metadata/slices, got %s vs %s", d1, d2)
	}

	// Nil plan check
	var nilPlan *domain.PlanSpec
	_, errNil := nilPlan.ComputeDigest()
	if errNil == nil || !errors.Is(errNil, domain.ErrPlanDigest) {
		t.Fatalf("expected ErrPlanDigest for nil plan, got: %v", errNil)
	}
}

func TestIsDirectTool(t *testing.T) {
	requiredBy := map[string][]string{
		"binutils": {"build-essential"},
		"fzf":      {"neovim"},
	}

	directTools := []domain.ManagedResource{
		{ID: "ghostty", Category: "terminal"},
		{ID: "herdr", Category: "terminal"},
		{ID: "neovim", Category: "editor"},
		{ID: "lazygit", Category: "devtools"},
		{ID: "zsh", Category: "shell"},
		{ID: "atuin", Category: "shell"},
		{ID: "bat", Category: "cli-tools"},
		{ID: "fd", Category: "cli-tools"},
		{ID: "ripgrep", Category: "cli-tools"},
		{ID: "eza", Category: "cli-tools"},
		{ID: "git", Category: "core"},
		{ID: "rtk", Category: "ai"},
	}

	for _, tool := range directTools {
		if !tool.IsDirectTool(requiredBy) {
			t.Errorf("expected tool %s to be classified as direct tool", tool.ID)
		}
	}

	dependencies := []domain.ManagedResource{
		{ID: "binutils", Category: "core-build"},
		{ID: "build-essential", Category: "core-build"},
		{ID: "bz2-dev", Category: "core-libs"},
		{ID: "ca-certificates", Category: "core"},
		{ID: "libssl-dev", Category: "core-libs"},
		{ID: "make", Category: "core-build"},
		{ID: "gcc", Category: "core-build"},
		{ID: "cmake", Category: "core-build"},
		{ID: "fontconfig", Category: "desktop-libs"},
		{ID: "fzf", Category: "cli-tools"}, // in requiredBy
	}

	for _, dep := range dependencies {
		if dep.IsDirectTool(requiredBy) {
			t.Errorf("expected dependency %s to be classified as dependency, got tool", dep.ID)
		}
	}
}
