package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/arturonavax/env/internal/catalog"
	"github.com/arturonavax/env/internal/domain"
	"github.com/arturonavax/env/internal/platform"
	"github.com/arturonavax/env/internal/security"
	"github.com/arturonavax/env/internal/state"
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

	// Enter advances to StagePlanning asynchronously
	nm, cmd := w.Update(tea.KeyPressMsg{Text: "enter"})
	w = nm.(WizardModel)
	if w.stage != StagePlanning {
		t.Fatalf("expected StagePlanning after enter, got %v", w.stage)
	}
	if cmd == nil {
		t.Fatalf("expected non-nil tea.Cmd for async planning")
	}

	// Deliver wizardPlanDoneMsg
	nm, _ = w.Update(wizardPlanDoneMsg{plan: nil, diffs: nil, err: nil})
	w = nm.(WizardModel)
	if w.stage != StageReviewPlan {
		t.Fatalf("expected StageReviewPlan after plan done, got %v", w.stage)
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

func TestAdjustScrollOffset(t *testing.T) {
	// 80 items, 10 visible lines
	// 1. Initial state
	offset := AdjustScrollOffset(0, 0, 10, 80)
	if offset != 0 {
		t.Fatalf("expected offset 0, got %d", offset)
	}

	// 2. Cursor at line 9 (still in view)
	offset = AdjustScrollOffset(9, 0, 10, 80)
	if offset != 0 {
		t.Fatalf("expected offset 0 at cursor 9, got %d", offset)
	}

	// 3. Cursor moves to line 10 -> offset slides to 1
	offset = AdjustScrollOffset(10, 0, 10, 80)
	if offset != 1 {
		t.Fatalf("expected offset 1 at cursor 10, got %d", offset)
	}

	// 4. Cursor moves back up to 9 while offset is 1 -> offset stays 1 (no erratic jump)
	offset = AdjustScrollOffset(9, 1, 10, 80)
	if offset != 1 {
		t.Fatalf("expected offset 1 at cursor 9, got %d", offset)
	}

	// 5. Cursor moves up to 0 while offset is 1 -> offset slides to 0
	offset = AdjustScrollOffset(0, 1, 10, 80)
	if offset != 0 {
		t.Fatalf("expected offset 0 at cursor 0, got %d", offset)
	}

	// 6. Fewer items than visible lines
	offset = AdjustScrollOffset(2, 5, 10, 5)
	if offset != 0 {
		t.Fatalf("expected offset 0 when totalItems < visibleLines, got %d", offset)
	}
}

func TestBuildScrollablePanel(t *testing.T) {
	pal := theme.ByName("tokyo-night")
	st := theme.NewStyles(pal)

	header := []string{"TEST HEADER"}
	items := make([]string, 50)
	for i := 0; i < 50; i++ {
		items[i] = strings.Repeat("item ", 5)
	}

	// height 12 -> innerHeight 10 -> availItemLines 9
	panel := BuildScrollablePanel(st, header, items, 15, 0, 60, 12, true)
	lines := strings.Split(panel, "\n")
	if len(lines) != 12 {
		t.Fatalf("expected exactly 12 lines, got %d", len(lines))
	}
	if !strings.Contains(panel, "TEST HEADER") {
		t.Fatalf("expected panel to contain header")
	}
}

func TestWizardScrollAndAsyncApply(t *testing.T) {
	deps := newTestDeps(t)
	wiz := NewWizardModel(deps.Platform, deps.Catalog, nil)
	wiz.width = 100
	wiz.height = 20 // small height to enforce scrolling

	// Stage 1 -> Stage 2
	nm, _ := wiz.Update(tea.KeyPressMsg{Text: "enter"})
	w := nm.(WizardModel)

	// Stage 2 -> Stage 3 (Packages)
	nm, _ = w.Update(tea.KeyPressMsg{Text: "enter"})
	w = nm.(WizardModel)
	if w.stage != StageSelectPackages {
		t.Fatalf("expected StageSelectPackages, got %v", w.stage)
	}

	// Scroll down in packages
	for i := 0; i < 15; i++ {
		nm, _ = w.Update(tea.KeyPressMsg{Text: "j"})
		w = nm.(WizardModel)
	}
	if w.pkgCursor != 15 {
		t.Fatalf("expected pkgCursor 15, got %d", w.pkgCursor)
	}
	if w.pkgScrollOffset == 0 {
		t.Fatalf("expected pkgScrollOffset > 0 after scrolling down 15 items in 20-row terminal")
	}

	// View packages
	pkgView := w.View().Content
	if !strings.Contains(pkgView, "STAGE 3/5") {
		t.Fatalf("expected stage 3 view")
	}

	// Advance to Review Plan via StagePlanning
	nm, cmdPlan := w.Update(tea.KeyPressMsg{Text: "enter"})
	w = nm.(WizardModel)
	if w.stage != StagePlanning {
		t.Fatalf("expected StagePlanning after enter on StageSelectPackages, got %v", w.stage)
	}
	if cmdPlan == nil {
		t.Fatalf("expected non-nil tea.Cmd for async planning")
	}

	// Deliver wizardPlanDoneMsg
	nm, _ = w.Update(wizardPlanDoneMsg{plan: nil, diffs: nil, err: nil})
	w = nm.(WizardModel)
	if w.stage != StageReviewPlan {
		t.Fatalf("expected StageReviewPlan, got %v", w.stage)
	}

	// Scroll plan down with j
	nm, _ = w.Update(tea.KeyPressMsg{Text: "j"})
	w = nm.(WizardModel)
	if w.planScrollOffset != 1 {
		t.Fatalf("expected planScrollOffset 1, got %d", w.planScrollOffset)
	}
	nm, _ = w.Update(tea.KeyPressMsg{Text: "k"})
	w = nm.(WizardModel)
	if w.planScrollOffset != 0 {
		t.Fatalf("expected planScrollOffset 0 after k, got %d", w.planScrollOffset)
	}

	// Press Enter to apply -> MUST transition to StageApplying immediately and return tea.Cmd (non-blocking!)
	nm, cmd := w.Update(tea.KeyPressMsg{Text: "enter"})
	w = nm.(WizardModel)
	if w.stage != StageApplying {
		t.Fatalf("expected StageApplying after enter on StageReviewPlan, got %v", w.stage)
	}
	if cmd == nil {
		t.Fatalf("expected non-nil tea.Cmd for async execution")
	}

	// Check applying view has STAGE 5/5
	applyingView := w.View().Content
	if !strings.Contains(applyingView, "STAGE 5/5") {
		t.Fatalf("expected STAGE 5/5 in applying view message, got: %s", applyingView)
	}

	// Deliver progress message
	nm, _ = w.Update(wizardProgressMsg{
		StepID:      3,
		TotalSteps:  10,
		Phase:       "Purging (Destroyer)",
		Action:      "purge",
		ResourceID:  "nvim",
		Description: "Purging ~/.config/nvim",
		Percent:     30,
	})
	w = nm.(WizardModel)
	if w.progressPercent != 30 {
		t.Fatalf("expected progressPercent 30, got %d", w.progressPercent)
	}
	progView := w.View().Content
	if !strings.Contains(progView, "30%") || !strings.Contains(progView, "Purging (Destroyer)") {
		t.Fatalf("expected progress and phase in applying view, got: %s", progView)
	}

	// Deliver completion message
	nm, _ = w.Update(wizardApplyDoneMsg{err: nil})
	w = nm.(WizardModel)
	if w.stage != StageCompleted {
		t.Fatalf("expected StageCompleted after wizardApplyDoneMsg, got %v", w.stage)
	}
	completedView := w.View().Content
	if !strings.Contains(completedView, "SUCCESSFULLY") {
		t.Fatalf("expected success message in completedView, got: %s", completedView)
	}
}

func TestWizardEscBackwardNavigation(t *testing.T) {
	deps := newTestDeps(t)
	wiz := NewWizardModel(deps.Platform, deps.Catalog, nil)

	// Stage 1 (Profile) -> Stage 2 (Environment)
	nm, _ := wiz.Update(tea.KeyPressMsg{Text: "enter"})
	w := nm.(WizardModel)
	if w.stage != StageSelectEnvironment {
		t.Fatalf("expected StageSelectEnvironment, got %v", w.stage)
	}

	// Stage 2 (Environment) -> Stage 3 (Packages)
	nm, _ = w.Update(tea.KeyPressMsg{Text: "enter"})
	w = nm.(WizardModel)
	if w.stage != StageSelectPackages {
		t.Fatalf("expected StageSelectPackages, got %v", w.stage)
	}

	// Stage 3 (Packages) -> StagePlanning
	nm, _ = w.Update(tea.KeyPressMsg{Text: "enter"})
	w = nm.(WizardModel)
	if w.stage != StagePlanning {
		t.Fatalf("expected StagePlanning, got %v", w.stage)
	}

	// Esc in StagePlanning -> returns to StageSelectPackages
	nm, _ = w.Update(tea.KeyPressMsg{Text: "esc"})
	w = nm.(WizardModel)
	if w.stage != StageSelectPackages {
		t.Fatalf("expected StageSelectPackages after esc on StagePlanning, got %v", w.stage)
	}

	// Esc in StageSelectPackages -> returns to StageSelectEnvironment
	nm, _ = w.Update(tea.KeyPressMsg{Text: "esc"})
	w = nm.(WizardModel)
	if w.stage != StageSelectEnvironment {
		t.Fatalf("expected StageSelectEnvironment after esc on StageSelectPackages, got %v", w.stage)
	}

	// Esc in StageSelectEnvironment -> returns to StageSelectProfile
	nm, _ = w.Update(tea.KeyPressMsg{Text: "esc"})
	w = nm.(WizardModel)
	if w.stage != StageSelectProfile {
		t.Fatalf("expected StageSelectProfile after esc on StageSelectEnvironment, got %v", w.stage)
	}

	// Esc in StageSelectProfile -> quits
	nm, _ = w.Update(tea.KeyPressMsg{Text: "esc"})
	w = nm.(WizardModel)
	if !w.quitting {
		t.Fatalf("expected quitting to be true after esc on StageSelectProfile")
	}
}

func TestWizardPackageDependenciesAutoSelection(t *testing.T) {
	cat := catalog.NewDefaultRegistry()
	plat := platform.Detect()
	w := NewWizardModel(plat, cat, nil)

	// Inject a custom package hierarchy to verify auto-selection and auto-deselection
	toolA := domain.ManagedResource{ID: "tool-a", DependsOn: []string{"dep-b"}}
	depB := domain.ManagedResource{ID: "dep-b", DependsOn: []string{"dep-c"}}
	depC := domain.ManagedResource{ID: "dep-c"}

	w.packages = []domain.ManagedResource{toolA, depB, depC}
	w.pkgMap = map[string]domain.ManagedResource{
		"tool-a": toolA,
		"dep-b":  depB,
		"dep-c":  depC,
	}
	w.requiredBy = map[string][]string{
		"dep-b": {"tool-a"},
		"dep-c": {"dep-b"},
	}
	w.selectedPkgs = map[string]bool{
		"tool-a": false,
		"dep-b":  false,
		"dep-c":  false,
	}
	w.stage = StageSelectPackages
	w.pkgCursor = 0 // On tool-a

	// Press space to enable tool-a -> should transitively select dep-b and dep-c
	nm, _ := w.Update(tea.KeyPressMsg{Code: ' ', Text: " "})
	w = nm.(WizardModel)
	if !w.selectedPkgs["tool-a"] || !w.selectedPkgs["dep-b"] || !w.selectedPkgs["dep-c"] {
		t.Fatalf("expected tool-a, dep-b, and dep-c to all be marked true, got: %+v", w.selectedPkgs)
	}

	// Move cursor to dep-c (index 2) and press space to disable dep-c
	w.pkgCursor = 2
	nm, _ = w.Update(tea.KeyPressMsg{Code: ' ', Text: " "})
	w = nm.(WizardModel)
	if w.selectedPkgs["dep-c"] {
		t.Fatalf("expected dep-c to be false")
	}
	// Because dep-c was unmarked, dep-b (depends on c) and tool-a (depends on b) must both be unmarked!
	if w.selectedPkgs["dep-b"] || w.selectedPkgs["tool-a"] {
		t.Fatalf("expected dep-b and tool-a to be auto-unmarked when dep-c is unmarked, got: %+v", w.selectedPkgs)
	}
}

func TestWizardProfilePersistence(t *testing.T) {
	tempDir := t.TempDir()
	store, err := state.OpenStore(tempDir)
	if err != nil {
		t.Fatalf("failed to open store: %v", err)
	}
	defer store.Close()

	cat := catalog.NewDefaultRegistry()
	plat := platform.Detect()
	w := NewWizardModel(plat, cat, nil, WithWizardStore(store))

	// Find index of "full" profile if available, or first profile
	fullIdx := 0
	for i, p := range w.profiles {
		if p.Name == "full" {
			fullIdx = i
			break
		}
	}
	w.selectedProf = fullIdx
	targetProfile := w.profiles[fullIdx].Name

	// Deliver successful apply done message
	nm, _ := w.Update(wizardApplyDoneMsg{err: nil})
	w = nm.(WizardModel)

	if !w.IsCompleted() {
		t.Fatalf("expected wizard to be completed")
	}
	if w.ChosenProfile() != targetProfile {
		t.Fatalf("expected chosen profile %q, got %q", targetProfile, w.ChosenProfile())
	}
	if store.ActiveProfile() != targetProfile {
		t.Fatalf("expected persisted active profile to be %q, got %q", targetProfile, store.ActiveProfile())
	}
}

func TestConfigsIntegrationStatus(t *testing.T) {
	deps := newTestDeps(t)
	m := NewModel(deps)
	m.SetDimensions(120, 40)

	// Jump to TabConfigs (tab 3)
	nm, _ := m.Update(tea.KeyPressMsg{Text: "3"})
	m = nm.(Model)
	if m.ActiveTab() != TabConfigs {
		t.Fatalf("expected TabConfigs, got %v", m.ActiveTab())
	}

	view := m.View()
	content := view.Content
	if !strings.Contains(content, "[Integrated ✓]") && !strings.Contains(content, "[Not Integrated ✗]") {
		t.Fatalf("expected configs tab to display integration badges, got:\n%s", content)
	}
}

func TestRenderSelectedRowContrast(t *testing.T) {
	pal := theme.ByName("tokyo-night")
	st := theme.NewStyles(pal)

	// An input with nested ANSI color codes and resets
	nestedInput := "\x1b[32m[✓]\x1b[0m \x1b[1;36mripgrep\x1b[0m : Fast search"
	rendered := st.RenderSelectedRow(nestedInput)

	// Rendered text must not contain inner ANSI color codes leaking before reset
	if strings.Contains(rendered, "\x1b[32m") || strings.Contains(rendered, "\x1b[1;36m") {
		t.Fatalf("expected inner ANSI color codes to be stripped in RenderSelectedRow, got: %q", rendered)
	}
}
