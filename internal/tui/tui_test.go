package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/arturonavax/env/internal/catalog"
	"github.com/arturonavax/env/internal/platform"
	"github.com/arturonavax/env/internal/security"
	"github.com/arturonavax/env/internal/theme"
)

func newTestDeps(t *testing.T) Deps {
	t.Helper()
	cat := catalog.NewDefaultRegistry()
	plat := platform.Detect()
	pal := theme.ByName("tokyo-night")
	return Deps{
		Platform: plat,
		Catalog:  cat,
		Theme:    &pal,
		Styles:   theme.NewStyles(pal),
	}
}

func TestNewModelAndDimensions(t *testing.T) {
	deps := newTestDeps(t)
	m := NewModel(deps)
	m.SetDimensions(120, 40)

	if m.ActiveTab() != TabOverview {
		t.Fatalf("expected initial tab to be TabOverview, got %v", m.ActiveTab())
	}
	if len(m.packages) == 0 {
		t.Fatalf("expected packages to be populated from catalog")
	}

	view := m.View()
	content := view.Content
	if !strings.Contains(content, "ANX") {
		t.Fatalf("expected view to contain header 'ANX', got: %s", content)
	}
	if !strings.Contains(content, "SYSTEM SUMMARY") && !strings.Contains(content, "SYSTEM OVERVIEW") {
		t.Fatalf("expected view to contain system overview, got: %s", content)
	}
}

func TestKeyboardNavigation(t *testing.T) {
	deps := newTestDeps(t)
	m := NewModel(deps)
	m.SetDimensions(120, 40)

	// Test Tab key forward cycle
	newM, _ := m.Update(tea.KeyPressMsg{Code: '	', Text: "tab"})
	mod := newM.(Model)
	if mod.ActiveTab() != TabPackages {
		t.Fatalf("expected TabPackages after tab, got %v", mod.ActiveTab())
	}

	// Test direct tab jumping
	for i, key := range []string{"1", "2", "3", "4", "5", "6", "7", "8"} {
		nm, _ := mod.Update(tea.KeyPressMsg{Text: key})
		curr := nm.(Model)
		if int(curr.ActiveTab()) != i {
			t.Fatalf("key %s: expected tab %d, got %d", key, i, curr.ActiveTab())
		}
	}

	// Test shift+tab backward cycle from TabOverview -> TabSecurity
	nm, _ := mod.Update(tea.KeyPressMsg{Text: "1"})
	curr := nm.(Model)
	nm, _ = curr.Update(tea.KeyPressMsg{Text: "shift+tab"})
	curr = nm.(Model)
	if curr.ActiveTab() != TabSecurity {
		t.Fatalf("expected TabSecurity after shift+tab from TabOverview, got %v", curr.ActiveTab())
	}

	// Test cursor navigation
	nm, _ = curr.Update(tea.KeyPressMsg{Text: "2"}) // switch to packages
	curr = nm.(Model)
	nm, _ = curr.Update(tea.KeyPressMsg{Text: "j"})
	curr = nm.(Model)
	if curr.selectedIdx != 1 {
		t.Fatalf("expected selectedIdx to be 1 after 'j', got %d", curr.selectedIdx)
	}
	nm, _ = curr.Update(tea.KeyPressMsg{Text: "k"})
	curr = nm.(Model)
	if curr.selectedIdx != 0 {
		t.Fatalf("expected selectedIdx to be 0 after 'k', got %d", curr.selectedIdx)
	}

	// Test accordion toggle
	nm, _ = curr.Update(tea.KeyPressMsg{Text: "a"})
	curr = nm.(Model)
	if curr.accordionMode {
		t.Fatalf("expected accordion to toggle to false")
	}

	// Test help modal toggle
	nm, _ = curr.Update(tea.KeyPressMsg{Text: "?"})
	curr = nm.(Model)
	if !curr.showHelp {
		t.Fatalf("expected showHelp to be true after '?'")
	}
	view := curr.View().Content
	if !strings.Contains(view, "KEYBOARD SHORTCUTS") {
		t.Fatalf("expected help modal content in view")
	}
}

func TestMouseInteraction(t *testing.T) {
	deps := newTestDeps(t)
	m := NewModel(deps)
	m.SetDimensions(120, 40)

	// Click row 1 at X=18 (should hit TabPackages)
	clickMsg := tea.MouseClickMsg{
		X:      18,
		Y:      1,
		Button: tea.MouseLeft,
	}
	nm, _ := m.Update(clickMsg)
	curr := nm.(Model)
	if curr.ActiveTab() != TabPackages {
		t.Fatalf("expected mouse click on tab bar to select TabPackages, got %v", curr.ActiveTab())
	}

	// Mouse wheel down
	wheelDown := tea.MouseWheelMsg{
		X:      20,
		Y:      10,
		Button: tea.MouseWheelDown,
	}
	nm, _ = curr.Update(wheelDown)
	curr = nm.(Model)
	if curr.selectedIdx != 1 {
		t.Fatalf("expected wheel down to increment selectedIdx, got %d", curr.selectedIdx)
	}

	// Mouse wheel up
	wheelUp := tea.MouseWheelMsg{
		X:      20,
		Y:      10,
		Button: tea.MouseWheelUp,
	}
	nm, _ = curr.Update(wheelUp)
	curr = nm.(Model)
	if curr.selectedIdx != 0 {
		t.Fatalf("expected wheel up to decrement selectedIdx, got %d", curr.selectedIdx)
	}
}

func TestGoldenRulesFormatting(t *testing.T) {
	// Golden Rule #1 & #2: Truncation and bordered panel line filling
	pal := theme.ByName("tokyo-night")
	st := theme.NewStyles(pal)

	lines := []string{"Line 1", "Line 2 - A very long line that would wrap if not truncated with ANSI protection", "Line 3"}
	panel := BuildPanel(st, lines, 40, 10, true)

	split := strings.Split(panel, "\n")
	if len(split) != 10 {
		t.Fatalf("expected exactly 10 lines in rendered panel, got %d", len(split))
	}

	truncated := AnsiTruncate("This is a very long string that should be trimmed safely", 15)
	if len(truncated) > 20 { // UTF-8 ellipsis could be 3 bytes
		t.Fatalf("expected truncated string to be <= 20 bytes, got %d", len(truncated))
	}
}

func TestWizardModelWorkflow(t *testing.T) {
	deps := newTestDeps(t)
	wiz := NewWizardModel(deps.Platform, deps.Catalog, nil)
	wiz.width = 100
	wiz.height = 30

	if wiz.stage != StageSelectProfile {
		t.Fatalf("expected initial wizard stage StageSelectProfile, got %v", wiz.stage)
	}

	// Enter selects profile and advances to StageSelectEnvironment
	nm, _ := wiz.Update(tea.KeyPressMsg{Text: "enter"})
	w := nm.(WizardModel)
	if w.stage != StageSelectEnvironment {
		t.Fatalf("expected StageSelectEnvironment after enter, got %v", w.stage)
	}

	// Verify environment view rendering
	envView := w.View()
	if !strings.Contains(envView.Content, "SHELL SELECTION") || !strings.Contains(envView.Content, "TERMINAL SELECTION") {
		t.Fatalf("expected shell and terminal selection in envView, got: %s", envView.Content)
	}

	// Switch focus to terminal and select
	nm, _ = w.Update(tea.KeyPressMsg{Text: "tab"})
	w = nm.(WizardModel)
	if w.envFocus != "terminal" {
		t.Fatalf("expected envFocus to be terminal after tab, got %s", w.envFocus)
	}

	// Enter advances to StageSelectPackages
	nm, _ = w.Update(tea.KeyPressMsg{Text: "enter"})
	w = nm.(WizardModel)
	if w.stage != StageSelectPackages {
		t.Fatalf("expected StageSelectPackages after enter, got %v", w.stage)
	}

	// Space toggles package
	nm, _ = w.Update(tea.KeyPressMsg{Text: " "})
	w = nm.(WizardModel)

	// Enter advances to StageReviewPlan
	nm, _ = w.Update(tea.KeyPressMsg{Text: "enter"})
	w = nm.(WizardModel)
	if w.stage != StageReviewPlan {
		t.Fatalf("expected StageReviewPlan after enter, got %v", w.stage)
	}

	// View renders without panic
	view := w.View()
	if !strings.Contains(view.Content, "ANX SETUP WIZARD") {
		t.Fatalf("expected wizard header in view, got: %s", view.Content)
	}
}

func TestSecurityAuditTabAndShortcuts(t *testing.T) {
	deps := newTestDeps(t)
	m := NewModel(deps)
	m.SetDimensions(120, 40)

	// Switch to TabSecurity
	nm, _ := m.Update(tea.KeyPressMsg{Text: "8"})
	curr := nm.(Model)
	if curr.ActiveTab() != TabSecurity {
		t.Fatalf("expected activeTab to be TabSecurity, got %v", curr.ActiveTab())
	}

	// Deliver simulated systemAuditLoadedMsg
	fakeReport := &security.SecurityAuditReport{
		Profile:        security.ProfileWorkstation,
		HardeningScore: 88,
		Findings: []security.SecurityFinding{
			{
				ID:                 "SEC-TEST-001",
				Category:           security.CategoryNetwork,
				Severity:           security.SeverityCritical,
				Title:              "Unprotected Test Port",
				Description:        "Port 9999 is open",
				Remediation:        "Close test port",
				RemediationCommand: "iptables -D INPUT -p tcp --dport 9999 -j ACCEPT",
			},
		},
	}

	nm, _ = curr.Update(systemAuditLoadedMsg(fakeReport))
	curr = nm.(Model)
	if curr.securityReport == nil || len(curr.securityReport.Findings) != 1 {
		t.Fatalf("expected 1 finding in securityReport")
	}

	view := curr.View().Content
	if !strings.Contains(view, "SYSTEM SECURITY AUDIT") {
		t.Fatalf("expected security audit header in view, got: %s", view)
	}
	if !strings.Contains(view, "Unprotected Test Port") {
		t.Fatalf("expected test finding title in view, got: %s", view)
	}
	if !strings.Contains(view, "88/100") {
		t.Fatalf("expected score 88/100 in view, got: %s", view)
	}

	// Test profile cycling with 'p'
	nm, _ = curr.Update(tea.KeyPressMsg{Text: "p"})
	curr = nm.(Model)
	if curr.securityProfile != security.ProfileServer {
		t.Fatalf("expected profile to switch to server, got %s", curr.securityProfile)
	}

	// Test script generation with 's'
	nm, _ = curr.Update(tea.KeyPressMsg{Text: "s"})
	curr = nm.(Model)
	if !strings.Contains(curr.statusMsg, "script") {
		t.Fatalf("expected status message mentioning script, got: %s", curr.statusMsg)
	}
}
