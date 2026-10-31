package app_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/arturonavax/env/internal/app"
	"github.com/arturonavax/env/internal/catalog"
	"github.com/arturonavax/env/internal/platform"
	"github.com/arturonavax/env/internal/state"
)

func TestDestroyerService_PlanPurge_All(t *testing.T) {
	plat := platform.Detect()
	cat := catalog.NewDefaultRegistry()
	tmpDir := t.TempDir()
	store, err := state.OpenStore(filepath.Join(tmpDir, "state"))
	if err != nil {
		t.Fatalf("failed to open store: %v", err)
	}

	destroyer := app.NewDestroyerService(plat, cat, nil, nil, store)

	plan, err := destroyer.PlanPurge(context.Background(), app.PurgeOptions{})
	if err != nil {
		t.Fatalf("PlanPurge failed: %v", err)
	}

	if plan == nil {
		t.Fatal("expected non-nil DestroyerPlan")
	}

	// In the embedded catalog, we have declared packages
	if len(plan.PackagesToUninstall) == 0 {
		t.Error("expected non-empty PackagesToUninstall from catalog")
	}
}

func TestDestroyerService_PlanPurge_ProfileScoped(t *testing.T) {
	plat := platform.Detect()
	cat := catalog.NewDefaultRegistry()
	tmpDir := t.TempDir()
	store, err := state.OpenStore(filepath.Join(tmpDir, "state"))
	if err != nil {
		t.Fatalf("failed to open store: %v", err)
	}

	destroyer := app.NewDestroyerService(plat, cat, nil, nil, store)

	// Plan purge for "minimal" profile
	plan, err := destroyer.PlanPurge(context.Background(), app.PurgeOptions{Profile: "minimal"})
	if err != nil {
		t.Fatalf("PlanPurge for minimal profile failed: %v", err)
	}

	if plan.Profile != "minimal" {
		t.Errorf("expected plan profile 'minimal', got %q", plan.Profile)
	}

	// Verify that packages resolved match the minimal profile packages
	expectedMinimalPkgs := map[string]bool{
		"git": true, "curl": true, "jq": true, "fzf": true,
		"ripgrep": true, "fd": true, "bat": true, "eza": true, "starship": true,
	}

	for _, pkg := range plan.PackagesToUninstall {
		if !expectedMinimalPkgs[pkg] {
			t.Errorf("package %q was not expected in minimal profile purge plan", pkg)
		}
	}
}

func TestDestroyerService_ExecutePurge_DryRun(t *testing.T) {
	plat := platform.Detect()
	cat := catalog.NewDefaultRegistry()
	tmpDir := t.TempDir()
	store, err := state.OpenStore(filepath.Join(tmpDir, "state"))
	if err != nil {
		t.Fatalf("failed to open store: %v", err)
	}

	destroyer := app.NewDestroyerService(plat, cat, nil, nil, store)

	plan, err := destroyer.PlanPurge(context.Background(), app.PurgeOptions{Profile: "minimal"})
	if err != nil {
		t.Fatalf("PlanPurge failed: %v", err)
	}

	res, err := destroyer.ExecutePurge(context.Background(), plan, true, false)
	if err != nil {
		t.Fatalf("ExecutePurge dry-run failed: %v", err)
	}

	if res == nil {
		t.Fatal("expected non-nil DestroyerResult")
	}

	if res.UninstalledPackages != len(plan.PackagesToUninstall) {
		t.Errorf("expected %d uninstalled packages in dry run, got %d", len(plan.PackagesToUninstall), res.UninstalledPackages)
	}
}

func TestPurgeConfigTarget_MarkerBlock(t *testing.T) {
	tmpDir := t.TempDir()
	targetFile := filepath.Join(tmpDir, ".zshrc")

	initialContent := `# User custom aliases
alias l="ls -la"

# >>> arturonavax/env >>>
# Managed by ANX
export PATH="/managed/bin:$PATH"
# <<< arturonavax/env <<<

# User custom function
my_func() { echo "hi"; }
`
	if err := os.WriteFile(targetFile, []byte(initialContent), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	if err := app.PurgeConfigTarget(targetFile); err != nil {
		t.Fatalf("PurgeConfigTarget failed: %v", err)
	}

	data, err := os.ReadFile(targetFile)
	if err != nil {
		t.Fatalf("failed to read cleansed file: %v", err)
	}

	cleansed := string(data)
	if strings.Contains(cleansed, "# >>> arturonavax/env >>>") {
		t.Error("expected marker start to be stripped")
	}
	if strings.Contains(cleansed, "# <<< arturonavax/env <<<") {
		t.Error("expected marker end to be stripped")
	}
	if !strings.Contains(cleansed, `alias l="ls -la"`) {
		t.Error("user alias was unexpectedly removed")
	}
	if !strings.Contains(cleansed, "my_func()") {
		t.Error("user function was unexpectedly removed")
	}
}

func TestPurgeConfigTargetWithMarker_CustomMarkers(t *testing.T) {
	tmpDir := t.TempDir()
	targetFile := filepath.Join(tmpDir, "custom.conf")

	initialContent := `# User custom config
setting = 1

// BEGIN CUSTOM MARKER
managed_setting = true
// END CUSTOM MARKER

other_setting = 2
`
	if err := os.WriteFile(targetFile, []byte(initialContent), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	if err := app.PurgeConfigTargetWithMarker(targetFile, "// BEGIN CUSTOM MARKER", "// END CUSTOM MARKER"); err != nil {
		t.Fatalf("PurgeConfigTargetWithMarker failed: %v", err)
	}

	data, err := os.ReadFile(targetFile)
	if err != nil {
		t.Fatalf("failed to read cleansed file: %v", err)
	}

	cleansed := string(data)
	if strings.Contains(cleansed, "BEGIN CUSTOM MARKER") || strings.Contains(cleansed, "managed_setting") {
		t.Error("expected custom marker block to be stripped")
	}
	if !strings.Contains(cleansed, "setting = 1") || !strings.Contains(cleansed, "other_setting = 2") {
		t.Error("user settings were unexpectedly removed")
	}
}

func TestPurgeConfigTarget_FileRemoval(t *testing.T) {
	tmpDir := t.TempDir()
	targetFile := filepath.Join(tmpDir, "some-config.json")

	if err := os.WriteFile(targetFile, []byte(`{"key": "value"}`), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	if err := app.PurgeConfigTarget(targetFile); err != nil {
		t.Fatalf("PurgeConfigTarget failed: %v", err)
	}

	if _, err := os.Stat(targetFile); !os.IsNotExist(err) {
		t.Errorf("expected file %s to be deleted, but it still exists", targetFile)
	}
}
