package wizard_test

import (
	"path/filepath"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/arturonavax/env/internal/ai/config"
	"github.com/arturonavax/env/internal/ai/harness"
	"github.com/arturonavax/env/internal/ai/wizard"
)

func TestWizardStateTransitionsAndAtomicSave(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.json")
	cfg, err := config.DefaultConfig()
	if err != nil {
		t.Fatal(err)
	}
	reg := harness.NewRegistry(cfg)

	var m tea.Model = wizard.NewModel(cfg, cfgPath, reg)
	m, _ = m.Update(tea.WindowSizeMsg{Width: 100, Height: 24})

	// Step 1: move down to 'claude' (index 1) and press enter
	m, _ = m.Update(tea.KeyPressMsg{Code: 'j', Text: "j"})
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})

	// Step 2 (effort): move down to 'medium' and press enter
	m, _ = m.Update(tea.KeyPressMsg{Code: 'j', Text: "j"})
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})

	// Step 3 (commit template): press enter on default
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})

	// Step 4 (PR template): press enter on default
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})

	// Step 5 (auto-wake): press enter
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})

	// Render view before final save to verify Golden Rules view assembly
	if v := m.View(); v.Content == "" {
		t.Fatal("expected non-empty wizard review View")
	}

	// Step 6 (review & save): press enter on 'Save configuration'
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})

	wm := m.(wizard.Model)
	if wm.SaveError() != nil {
		t.Fatalf("wizard SaveError(): %v", wm.SaveError())
	}
	if !wm.Saved() {
		t.Fatal("expected wizard Saved() == true")
	}

	reloaded, err := config.Load(cfgPath)
	if err != nil {
		t.Fatalf("Load() saved wizard config: %v", err)
	}
	if reloaded.DefaultCommitBackend != "claude" {
		t.Errorf("saved DefaultCommitBackend = %q, want claude", reloaded.DefaultCommitBackend)
	}
	if reloaded.Commit.Effort != "medium" {
		t.Errorf("saved Commit.Effort = %q, want medium", reloaded.Commit.Effort)
	}
}
