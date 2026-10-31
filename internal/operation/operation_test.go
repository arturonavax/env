package operation

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/arturonavax/env/internal/domain"
)

func TestEscapePOSIX(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"", "''"},
		{"simple", "simple"},
		{"with spaces", "'with spaces'"},
		{"with'quote", `'with'\''quote'`},
		{"path/to/file.txt", "path/to/file.txt"},
		{"flag=value", "flag=value"},
		{"special$var;rm", "'special$var;rm'"},
	}

	for _, tc := range cases {
		got := EscapePOSIX(tc.input)
		if got != tc.want {
			t.Errorf("EscapePOSIX(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestEscapePowerShell(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"", "''"},
		{"simple", "simple"},
		{"with spaces", "'with spaces'"},
		{"with'quote", "'with''quote'"},
		{"path/to/file.txt", "path/to/file.txt"},
	}

	for _, tc := range cases {
		got := EscapePowerShell(tc.input)
		if got != tc.want {
			t.Errorf("EscapePowerShell(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestRenderers(t *testing.T) {
	plan := PlanIR{
		ID:            "plan-test-1",
		SchemaVersion: 1,
		Operations: []Operation{
			{
				OpID:        "op-1",
				Kind:        OpEnsureDirectory,
				TargetPath:  "/tmp/anx-test-dir",
				Description: "Create test directory",
			},
			{
				OpID:        "op-2",
				Kind:        OpWriteAtomic,
				TargetPath:  "/tmp/anx-test-dir/config.toml",
				Content:     "[server]\nport = 8080\n",
				Description: "Write config file",
			},
			{
				OpID:        "op-3",
				Kind:        OpPackageInstall,
				Argv:        []string{"apt-get", "install", "-y", "ripgrep"},
				Description: "Install ripgrep",
			},
		},
	}

	// 1. ShRenderer Deterministic
	shR := &ShRenderer{}
	out1, err := shR.Render(plan, RenderOptions{Deterministic: true, IncludeHeader: true})
	if err != nil {
		t.Fatalf("ShRenderer failed: %v", err)
	}
	out2, _ := shR.Render(plan, RenderOptions{Deterministic: true, IncludeHeader: true})
	if string(out1) != string(out2) {
		t.Errorf("ShRenderer is not deterministic across identical calls")
	}
	if !strings.Contains(string(out1), "set -eu") {
		t.Errorf("ShRenderer missing set -eu: %s", string(out1))
	}

	// 2. BashRenderer
	bashR := &BashRenderer{}
	bashOut, err := bashR.Render(plan, RenderOptions{Deterministic: true, IncludeHeader: true})
	if err != nil {
		t.Fatalf("BashRenderer failed: %v", err)
	}
	if !strings.Contains(string(bashOut), "set -euo pipefail") {
		t.Errorf("BashRenderer missing set -euo pipefail: %s", string(bashOut))
	}

	// 3. PowerShellRenderer
	pwshR := &PowerShellRenderer{}
	pwshOut, err := pwshR.Render(plan, RenderOptions{Deterministic: true, IncludeHeader: true})
	if err != nil {
		t.Fatalf("PowerShellRenderer failed: %v", err)
	}
	if !strings.Contains(string(pwshOut), "$ErrorActionPreference = 'Stop'") {
		t.Errorf("PowerShellRenderer missing error action preference: %s", string(pwshOut))
	}
}

func TestGoExecutor(t *testing.T) {
	dir := t.TempDir()
	targetDir := filepath.Join(dir, "sub", "testdir")
	targetFile := filepath.Join(targetDir, "file.txt")
	copyFile := filepath.Join(dir, "copied.txt")

	plan := PlanIR{
		ID:            "test-exec-1",
		SchemaVersion: 1,
		Operations: []Operation{
			{
				OpID:       "op-mkdir",
				Kind:       OpEnsureDirectory,
				TargetPath: targetDir,
			},
			{
				OpID:       "op-write",
				Kind:       OpWriteAtomic,
				TargetPath: targetFile,
				Content:    "Hello from GoExecutor\n",
				Postconditions: []Condition{
					{Type: "file_exists", Target: targetFile},
				},
			},
			{
				OpID:       "op-copy",
				Kind:       OpCopyAtomic,
				SourcePath: targetFile,
				TargetPath: copyFile,
				Postconditions: []Condition{
					{Type: "file_exists", Target: copyFile},
				},
			},
		},
	}

	exec := NewGoExecutor(nil, nil)
	if err := exec.ExecutePlan(context.Background(), plan); err != nil {
		t.Fatalf("ExecutePlan failed: %v", err)
	}

	data, err := os.ReadFile(copyFile)
	if err != nil || string(data) != "Hello from GoExecutor\n" {
		t.Fatalf("unexpected copied file content: %s (err: %v)", string(data), err)
	}
}

func TestCompileIR(t *testing.T) {
	execPlan := &domain.ExecutionPlan{
		ID:       "test-plan-id",
		Platform: "linux",
		Profile:  "developer",
		Steps: []domain.PlanStep{
			{
				ID:          1,
				ResourceID:  "ripgrep",
				Action:      domain.ActionInstall,
				Provider:    "apt",
				Description: "Install ripgrep",
			},
			{
				ID:          2,
				ResourceID:  "starship",
				Action:      domain.ActionSync,
				Strategy:    "symlink",
				SourcePath:  "/tmp/starship.toml",
				TargetPath:  "/tmp/.config/starship.toml",
				Description: "Sync starship config",
			},
		},
	}

	ir, err := CompileIR(execPlan, "linux", "amd64")
	if err != nil {
		t.Fatalf("CompileIR failed: %v", err)
	}

	if len(ir.Operations) != 2 {
		t.Fatalf("expected 2 operations, got %d", len(ir.Operations))
	}
	if ir.Operations[0].Kind != OpPackageInstall || ir.Operations[0].Argv[0] != "apt-get" {
		t.Errorf("unexpected first op: %+v", ir.Operations[0])
	}
	if ir.Operations[1].Kind != OpCreateSymlinkOwned {
		t.Errorf("unexpected second op: %+v", ir.Operations[1])
	}

	digest, err := ir.CanonicalDigest()
	if err != nil || digest == "" {
		t.Fatalf("CanonicalDigest failed: %v, digest: %s", err, digest)
	}
}

func TestCompileIR_MultiLevelDependsOn(t *testing.T) {
	execPlan := &domain.ExecutionPlan{
		ID:       "dag-plan",
		Platform: "linux",
		Steps: []domain.PlanStep{
			{ID: 1, ResourceID: "base", Action: domain.ActionInstall, Provider: "apt"},
			{ID: 2, ResourceID: "mid", Action: domain.ActionInstall, Provider: "apt", DependsOn: []int{1}},
			{ID: 3, ResourceID: "top", Action: domain.ActionInstall, Provider: "apt", DependsOn: []int{2}},
		},
	}

	ir, err := CompileIR(execPlan, "linux", "amd64")
	if err != nil {
		t.Fatalf("CompileIR failed: %v", err)
	}

	if len(ir.Operations[1].DependsOn) != 1 || ir.Operations[1].DependsOn[0] != "step-1-base" {
		t.Fatalf("expected op 2 to depend on 'step-1-base', got %v", ir.Operations[1].DependsOn)
	}
	if len(ir.Operations[2].DependsOn) != 1 || ir.Operations[2].DependsOn[0] != "step-2-mid" {
		t.Fatalf("expected op 3 to depend on 'step-2-mid', got %v", ir.Operations[2].DependsOn)
	}
}

func TestGoExecutor_DAGCycleAndMissingDeps(t *testing.T) {
	exec := NewGoExecutor(nil, nil)

	// Missing dependency
	missingPlan := PlanIR{
		Operations: []Operation{
			{OpID: "op-1", Kind: OpEnsureDirectory, TargetPath: "/tmp", DependsOn: []string{"op-nonexistent"}},
		},
	}
	if err := exec.ExecutePlan(context.Background(), missingPlan); err == nil {
		t.Fatal("expected error for non-existent dependency")
	}

	// Cycle
	cyclePlan := PlanIR{
		Operations: []Operation{
			{OpID: "op-1", Kind: OpEnsureDirectory, TargetPath: "/tmp", DependsOn: []string{"op-2"}},
			{OpID: "op-2", Kind: OpEnsureDirectory, TargetPath: "/tmp", DependsOn: []string{"op-1"}},
		},
	}
	if err := exec.ExecutePlan(context.Background(), cyclePlan); err == nil {
		t.Fatal("expected error for dependency cycle")
	}
}
