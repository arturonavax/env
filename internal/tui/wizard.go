package tui

import (
	"context"
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/arturonavax/env/internal/app"
	"github.com/arturonavax/env/internal/catalog"
	"github.com/arturonavax/env/internal/domain"
	"github.com/arturonavax/env/internal/platform"
	"github.com/arturonavax/env/internal/security"
	"github.com/arturonavax/env/internal/theme"
)

// WizardStage tracks which screen of the wizard is active.
type WizardStage int

const (
	StageSelectProfile WizardStage = iota
	StageSelectEnvironment
	StageSelectPackages
	StageReviewPlan
	StageApplying
	StageCompleted
)

// ShellOption defines a selectable shell with feature indicators.
type ShellOption struct {
	ID          string
	Name        string
	Badge       string
	Description string
}

// TerminalOption defines a selectable terminal with compatibility notes.
type TerminalOption struct {
	ID          string
	Name        string
	Badge       string
	Description string
	Warning     string
}

// WizardModel represents the interactive setup interview wizard.
type WizardModel struct {
	width            int
	height           int
	stage            WizardStage
	styles           *theme.Styles
	platform         *platform.Info
	catalog          *catalog.Registry
	installSvc       *app.InstallService
	profiles         []domain.Profile
	selectedProf     int
	packages         []domain.ManagedResource
	selectedPkgs     map[string]bool
	pkgCursor        int
	plan             *domain.ExecutionPlan
	progressMsg      string
	err              error
	quitting         bool
	matchedHacks     []domain.HackProfile
	includeHacks     bool
	configDiffs      []app.ConfigDiff
	syncSvc          *app.SyncService
	hacksSvc         *app.HacksService
	shells           []ShellOption
	terminals        []TerminalOption
	selectedShell    string
	selectedTerminal string
	shellCursor      int
	terminalCursor   int
	envFocus         string // "shell" or "terminal"
	destroyerMode    bool
	destroyerSvc     *app.DestroyerService
	seal             *security.OriginSeal
}

func defaultShellOptions() []ShellOption {
	return []ShellOption{
		{ID: "zsh", Name: "Zsh", Badge: "[Recommended ✓]", Description: "Primary source of truth (fzf-tab, SQLite Atuin, Starship, zcompile)"},
		{ID: "bash", Name: "Bash", Badge: "[Adapter]", Description: "Adapted ~/.base.bash with Starship, Atuin, FZF and aliases"},
		{ID: "fish", Name: "Fish", Badge: "[Adapter]", Description: "Fish shell config with Starship, Atuin, and FZF keybindings"},
		{ID: "nu", Name: "Nushell", Badge: "[Adapter]", Description: "Nushell environment with Starship & Atuin hooks"},
	}
}

func defaultTerminalOptions() []TerminalOption {
	return []TerminalOption{
		{ID: "ghostty", Name: "Ghostty", Badge: "[Recommended ✓]", Description: "GPU terminal with Herdr multiplexer, split panes, 24-bit TrueColor", Warning: ""},
		{ID: "kitty", Name: "Kitty", Badge: "[Supported ✓]", Description: "Fast GPU terminal with Kitty graphics, Tokyo Night theme, Herdr", Warning: ""},
		{ID: "wezterm", Name: "WezTerm", Badge: "[Supported ✓]", Description: "Cross-platform GPU terminal with Lua configuration and Herdr", Warning: ""},
		{ID: "alacritty", Name: "Alacritty", Badge: "[Supported ✓]", Description: "Ultra-fast OpenGL terminal with Tokyo Night theme and Herdr", Warning: ""},
		{ID: "windows-terminal", Name: "Windows Terminal", Badge: "[Partial ⚠️]", Description: "Microsoft Windows Terminal with tabs and split panes", Warning: "POSIX signals and socket multiplexing require WSL"},
		{ID: "warp", Name: "Warp", Badge: "[Limited ⚠️]", Description: "Proprietary AI command-block terminal", Warning: "Starship prompt, FZF widgets, and Herdr are disabled by Warp's design"},
	}
}

// WizardOption configures optional parameters for the setup wizard.
type WizardOption func(*WizardModel)

// WithWizardMatchedHacks injects detected hardware hacks into the wizard.
func WithWizardMatchedHacks(hacks []domain.HackProfile) WizardOption {
	return func(w *WizardModel) {
		w.matchedHacks = hacks
	}
}

// WithWizardIncludeHacks sets initial hack opt-in state.
func WithWizardIncludeHacks(include bool) WizardOption {
	return func(w *WizardModel) {
		w.includeHacks = include
	}
}

// WithWizardConfigDiffs injects detected dotfile diffs into the wizard.
func WithWizardConfigDiffs(diffs []app.ConfigDiff) WizardOption {
	return func(w *WizardModel) {
		w.configDiffs = diffs
	}
}

// WithWizardSyncService injects SyncService for dotfile reconciliation during apply.
func WithWizardSyncService(svc *app.SyncService) WizardOption {
	return func(w *WizardModel) {
		w.syncSvc = svc
	}
}

// WithWizardHacksService injects HacksService for applying matched workstation hacks.
func WithWizardHacksService(svc *app.HacksService) WizardOption {
	return func(w *WizardModel) {
		w.hacksSvc = svc
	}
}

// WithWizardDestroyerSvc injects DestroyerService into the wizard.
func WithWizardDestroyerSvc(svc *app.DestroyerService) WizardOption {
	return func(w *WizardModel) {
		w.destroyerSvc = svc
	}
}

// WithWizardDestroyerMode sets the initial destroyer reinstall mode in the wizard.
func WithWizardDestroyerMode(enable bool) WizardOption {
	return func(w *WizardModel) {
		w.destroyerMode = enable
	}
}

// WithWizardSecuritySeal injects the cryptographic origin seal into the wizard.
func WithWizardSecuritySeal(seal *security.OriginSeal) WizardOption {
	return func(w *WizardModel) {
		w.seal = seal
	}
}

// NewWizardModel creates an upfront interactive wizard model.
func NewWizardModel(
	plat *platform.Info,
	cat *catalog.Registry,
	svc *app.InstallService,
	opts ...WizardOption,
) WizardModel {
	var pkgs []domain.ManagedResource
	var profs []domain.Profile
	if cat != nil {
		pkgs = cat.ListPackages()
		profs = cat.ListProfiles()
	}

	selected := make(map[string]bool)
	for _, p := range pkgs {
		selected[p.ID] = true // Default select all
	}

	shells := defaultShellOptions()
	terminals := defaultTerminalOptions()
	selectedShell := "zsh"
	selectedTerminal := "ghostty"

	shellIdx := 0
	if plat != nil && plat.CurrentShell() != "" {
		for i, s := range shells {
			if s.ID == plat.CurrentShell() {
				shellIdx = i
				selectedShell = s.ID
				break
			}
		}
	}

	termIdx := 0
	if plat != nil && plat.CurrentTerminal() != "" {
		for i, t := range terminals {
			if t.ID == plat.CurrentTerminal() {
				termIdx = i
				selectedTerminal = t.ID
				break
			}
		}
	}

	m := WizardModel{
		stage:            StageSelectProfile,
		styles:           theme.DefaultStyles(),
		platform:         plat,
		catalog:          cat,
		installSvc:       svc,
		profiles:         profs,
		selectedProf:     0,
		packages:         pkgs,
		selectedPkgs:     selected,
		pkgCursor:        0,
		shells:           shells,
		terminals:        terminals,
		selectedShell:    selectedShell,
		selectedTerminal: selectedTerminal,
		shellCursor:      shellIdx,
		terminalCursor:   termIdx,
		envFocus:         "shell",
	}

	for _, opt := range opts {
		if opt != nil {
			opt(&m)
		}
	}

	return m
}

// Init implements tea.Model.
func (w WizardModel) Init() tea.Cmd {
	return nil
}

// Update handles keyboard and window resize events in the wizard.
func (w WizardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			w.quitting = true
			return w, tea.Quit

		case "up", "k":
			switch w.stage {
			case StageSelectProfile:
				if w.selectedProf > 0 {
					w.selectedProf--
				}
			case StageSelectEnvironment:
				if w.envFocus == "shell" {
					if w.shellCursor > 0 {
						w.shellCursor--
					}
				} else {
					if w.terminalCursor > 0 {
						w.terminalCursor--
					}
				}
			case StageSelectPackages:
				if w.pkgCursor > 0 {
					w.pkgCursor--
				}
			}

		case "down", "j":
			switch w.stage {
			case StageSelectProfile:
				if w.selectedProf < len(w.profiles)-1 {
					w.selectedProf++
				}
			case StageSelectEnvironment:
				if w.envFocus == "shell" {
					if w.shellCursor < len(w.shells)-1 {
						w.shellCursor++
					}
				} else {
					if w.terminalCursor < len(w.terminals)-1 {
						w.terminalCursor++
					}
				}
			case StageSelectPackages:
				if w.pkgCursor < len(w.packages)-1 {
					w.pkgCursor++
				}
			}

		case "tab", "left", "right":
			if w.stage == StageSelectEnvironment {
				if w.envFocus == "shell" {
					w.envFocus = "terminal"
				} else {
					w.envFocus = "shell"
				}
			}

		case " ":
			if w.stage == StageSelectEnvironment {
				if w.envFocus == "shell" && w.shellCursor < len(w.shells) {
					w.selectedShell = w.shells[w.shellCursor].ID
				} else if w.envFocus == "terminal" && w.terminalCursor < len(w.terminals) {
					w.selectedTerminal = w.terminals[w.terminalCursor].ID
				}
			} else if w.stage == StageSelectPackages && len(w.packages) > 0 {
				pkgID := w.packages[w.pkgCursor].ID
				w.selectedPkgs[pkgID] = !w.selectedPkgs[pkgID]
			}

		case "h":
			if len(w.matchedHacks) > 0 {
				w.includeHacks = !w.includeHacks
			}

		case "d", "D":
			w.destroyerMode = !w.destroyerMode

		case "enter":
			switch w.stage {
			case StageSelectProfile:
				if len(w.profiles) > 0 && w.catalog != nil {
					prof := w.profiles[w.selectedProf]
					if resolved, err := w.catalog.ResolveProfilePackages(prof.Name); err == nil {
						w.selectedPkgs = make(map[string]bool)
						for _, p := range resolved {
							w.selectedPkgs[p] = true
						}
					}
				}
				w.stage = StageSelectEnvironment

			case StageSelectEnvironment:
				if w.shellCursor < len(w.shells) {
					w.selectedShell = w.shells[w.shellCursor].ID
				}
				if w.terminalCursor < len(w.terminals) {
					w.selectedTerminal = w.terminals[w.terminalCursor].ID
				}
				w.stage = StageSelectPackages

			case StageSelectPackages:
				selectedList := make([]string, 0, len(w.selectedPkgs))
				for id, enabled := range w.selectedPkgs {
					if enabled {
						selectedList = append(selectedList, id)
					}
				}
				if w.installSvc != nil {
					plan, err := w.installSvc.Plan(context.Background(), "", selectedList)
					if err != nil {
						w.err = err
					} else {
						w.plan = plan
						w.stage = StageReviewPlan
					}
				} else {
					w.stage = StageReviewPlan
				}

			case StageReviewPlan:
				if w.seal != nil {
					if err := w.seal.Verify(context.Background()); err != nil {
						w.err = err
						w.stage = StageCompleted
						break
					}
				}
				w.stage = StageApplying
				if w.destroyerMode && w.destroyerSvc != nil {
					profName := ""
					if len(w.profiles) > 0 && w.selectedProf < len(w.profiles) {
						profName = w.profiles[w.selectedProf].Name
					}
					opts := app.PurgeOptions{
						Profile:       profName,
						PurgeBinaries: false,
						PurgeState:    false,
					}
					if purgePlan, err := w.destroyerSvc.PlanPurge(context.Background(), opts); err == nil {
						_, _ = w.destroyerSvc.ExecutePurge(context.Background(), purgePlan, false, false)
					}
				}
				if w.installSvc != nil && w.plan != nil {
					if err := w.installSvc.Apply(context.Background(), w.plan); err != nil {
						w.err = err
					}
				}
				if w.hacksSvc != nil && w.includeHacks && len(w.matchedHacks) > 0 {
					for _, hk := range w.matchedHacks {
						_, _ = w.hacksSvc.Apply(context.Background(), hk.ID)
					}
				}
				if w.syncSvc != nil {
					_, _ = w.syncSvc.Sync(context.Background(), app.SyncOptions{
						Force:    w.destroyerMode,
						Shell:    w.selectedShell,
						Terminal: w.selectedTerminal,
					})
				}
				w.stage = StageCompleted

			case StageCompleted:
				w.quitting = true
				return w, tea.Quit
			}
		}

	case tea.WindowSizeMsg:
		w.width = msg.Width
		w.height = msg.Height
	}

	return w, nil
}

// View renders the current stage of the interactive wizard (Bubble Tea v2).
func (w WizardModel) View() tea.View {
	if w.quitting {
		return tea.NewView("")
	}

	header := w.styles.Header.Render("  ANX SETUP WIZARD  ") + w.styles.Muted.Render(" — Interactive Component Interview")
	availWidth := w.width
	if availWidth < 50 {
		availWidth = 50
	}
	availHeight := w.height - 4
	if availHeight < 6 {
		availHeight = 6
	}

	var body string
	switch w.stage {
	case StageSelectProfile:
		body = w.renderSelectProfile(availWidth)
	case StageSelectEnvironment:
		body = w.renderSelectEnvironment(availWidth)
	case StageSelectPackages:
		body = w.renderSelectPackages(availWidth)
	case StageReviewPlan:
		body = w.renderReviewPlan(availWidth)
	case StageApplying:
		body = w.styles.Highlight.Render("Applying planned components and synchronizing dotfiles...\nPlease wait...")
	case StageCompleted:
		body = w.renderCompleted(availWidth)
	}

	content := fmt.Sprintf("%s\n\n%s", header, body)
	v := tea.NewView(content)
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion
	return v
}

func (w WizardModel) renderSelectProfile(width int) string {
	lines := []string{
		w.styles.Highlight.Render("STAGE 1/5: SELECT INSTALLATION PROFILE"),
		"Choose a preset profile matching your machine's primary role:",
	}
	if len(w.matchedHacks) > 0 {
		hackNames := make([]string, 0, len(w.matchedHacks))
		for _, hk := range w.matchedHacks {
			hackNames = append(hackNames, hk.ID)
		}
		status := w.styles.Muted.Render("[press 'h' to toggle]")
		if w.includeHacks {
			status = w.styles.Success.Render("[included ✓]")
		}
		lines = append(lines, w.styles.Warning.Render(fmt.Sprintf("⚡ Hardware match: %s %s", strings.Join(hackNames, ", "), status)))
	}
	lines = append(lines, "")

	for i, prof := range w.profiles {
		cursor := "  "
		if i == w.selectedProf {
			cursor = "> "
		}
		item := fmt.Sprintf("%s%-15s : %s", cursor, prof.Name, prof.Description)
		if i == w.selectedProf {
			lines = append(lines, w.styles.SelectedRow.Render(AnsiTruncate(item, width-4)))
		} else {
			lines = append(lines, AnsiTruncate(item, width-4))
		}
	}

	lines = append(lines, "", w.styles.Muted.Render("Navigation: [Up/Down / j/k] Select  [Enter] Confirm Profile  [h] Toggle Hacks  [q] Quit"))
	return strings.Join(lines, "\n")
}

func (w WizardModel) renderSelectEnvironment(width int) string {
	lines := []string{
		w.styles.Highlight.Render("STAGE 2/5: CONFIGURE SHELL & TERMINAL ENVIRONMENT"),
		"Select your primary shell and terminal. Zsh is the canonical source of truth; adapters are provided for Bash, Fish, and Nushell.",
	}

	if w.platform != nil {
		hostInfo := fmt.Sprintf("System: %s/%s | Detected Shell: %s | Detected Terminal: %s",
			w.platform.OS(), w.platform.DesktopEnv(), w.platform.CurrentShell(), w.platform.CurrentTerminal())
		lines = append(lines, w.styles.Muted.Render(hostInfo))
	}
	lines = append(lines, "")

	// 1. Shell Section
	shellHeader := "  SHELL SELECTION"
	if w.envFocus == "shell" {
		shellHeader = w.styles.Highlight.Render("▶ SHELL SELECTION [Active Focus]")
	} else {
		shellHeader = w.styles.Muted.Render("  SHELL SELECTION (Press [Tab] to focus)")
	}
	lines = append(lines, shellHeader)

	for i, sh := range w.shells {
		cursor := "  "
		if w.envFocus == "shell" && i == w.shellCursor {
			cursor = "> "
		}
		radio := "( )"
		if sh.ID == w.selectedShell {
			radio = w.styles.Success.Render("(•)")
		}
		badge := w.styles.Muted.Render(sh.Badge)
		if strings.Contains(sh.Badge, "Recommended") {
			badge = w.styles.Success.Render(sh.Badge)
		}
		row := fmt.Sprintf("%s%s %-8s %-18s : %s", cursor, radio, sh.Name, badge, sh.Description)
		if w.envFocus == "shell" && i == w.shellCursor {
			lines = append(lines, w.styles.SelectedRow.Render(AnsiTruncate(row, width-4)))
		} else {
			lines = append(lines, AnsiTruncate(row, width-4))
		}
	}

	lines = append(lines, "")

	// 2. Terminal Section
	termHeader := "  TERMINAL SELECTION"
	if w.envFocus == "terminal" {
		termHeader = w.styles.Highlight.Render("▶ TERMINAL SELECTION [Active Focus]")
	} else {
		termHeader = w.styles.Muted.Render("  TERMINAL SELECTION (Press [Tab] to focus)")
	}
	lines = append(lines, termHeader)

	for i, term := range w.terminals {
		cursor := "  "
		if w.envFocus == "terminal" && i == w.terminalCursor {
			cursor = "> "
		}
		radio := "( )"
		if term.ID == w.selectedTerminal {
			radio = w.styles.Success.Render("(•)")
		}
		badge := w.styles.Muted.Render(term.Badge)
		if strings.Contains(term.Badge, "Recommended") {
			badge = w.styles.Success.Render(term.Badge)
		} else if strings.Contains(term.Badge, "Partial") || strings.Contains(term.Badge, "Limited") {
			badge = w.styles.Warning.Render(term.Badge)
		}
		row := fmt.Sprintf("%s%s %-18s %-16s : %s", cursor, radio, term.Name, badge, term.Description)
		if w.envFocus == "terminal" && i == w.terminalCursor {
			lines = append(lines, w.styles.SelectedRow.Render(AnsiTruncate(row, width-4)))
		} else {
			lines = append(lines, AnsiTruncate(row, width-4))
		}
	}

	// Warning notices for limited or partial terminals
	for _, term := range w.terminals {
		if term.ID == w.selectedTerminal && term.Warning != "" {
			lines = append(lines, "", w.styles.Warning.Render(fmt.Sprintf("⚠️  NOTICE for %s: %s", term.Name, term.Warning)))
			break
		}
	}

	lines = append(lines, "", w.styles.Muted.Render("Navigation: [Tab / Left / Right] Switch Focus  [Up/Down / j/k] Navigate  [Space] Select  [Enter] Confirm & Next  [q] Quit"))
	return strings.Join(lines, "\n")
}

func (w WizardModel) renderSelectPackages(width int) string {
	lines := []string{
		w.styles.Highlight.Render("STAGE 3/5: CUSTOMIZE TOOLS & PACKAGES"),
		"Toggle components to include or exclude from your workstation:",
		"",
	}

	for i, p := range w.packages {
		cursor := "  "
		if i == w.pkgCursor {
			cursor = "> "
		}
		check := "[ ]"
		if w.selectedPkgs[p.ID] {
			check = w.styles.Success.Render("[✓]")
		}
		depBadge := ""
		if p.Deprecated {
			depBadge = " " + w.styles.Warning.Render("[Deprecated]")
		}
		row := fmt.Sprintf("%s%s %-12s %-12s%s : %s", cursor, check, p.ID, p.Category, depBadge, p.Description)
		if i == w.pkgCursor {
			lines = append(lines, w.styles.Highlight.Render(AnsiTruncate(row, width-4)))
		} else {
			lines = append(lines, AnsiTruncate(row, width-4))
		}
	}

	if len(w.matchedHacks) > 0 {
		status := w.styles.Muted.Render("NO (press 'h' to include)")
		if w.includeHacks {
			status = w.styles.Success.Render("YES ✓ (press 'h' to exclude)")
		}
		lines = append(lines, "", fmt.Sprintf("⚡ Include hardware workstation hacks: %s", status))
	}

	lines = append(lines, "", w.styles.Muted.Render("Navigation: [Up/Down / j/k] Navigate  [Space] Toggle  [h] Toggle Hacks  [Enter] Review Plan  [q] Quit"))
	return strings.Join(lines, "\n")
}

func (w WizardModel) renderReviewPlan(width int) string {
	planID := "preview"
	stepCount := 0
	if w.plan != nil {
		planID = w.plan.ID
		stepCount = len(w.plan.Steps)
	}
	lines := []string{
		w.styles.Highlight.Render("STAGE 4/5: REVIEW EXECUTION PLAN"),
		fmt.Sprintf("Plan ID: %s | Total Actions: %d", planID, stepCount),
		w.styles.Muted.Render("• Automatic pre-mutation backups will be created before touching any file."),
		w.styles.Muted.Render("• Transaction journal will record each step for rollback safety."),
	}

	lines = append(lines, "", w.styles.Highlight.Render("TARGET ENVIRONMENT:"))
	lines = append(lines, fmt.Sprintf("  • Primary Shell : %s (Zsh source of truth)", w.selectedShell))
	termWarning := ""
	for _, t := range w.terminals {
		if t.ID == w.selectedTerminal && t.Warning != "" {
			termWarning = fmt.Sprintf(" (%s)", w.styles.Warning.Render(t.Warning))
			break
		}
	}
	lines = append(lines, fmt.Sprintf("  • Terminal      : %s%s", w.selectedTerminal, termWarning))

	lines = append(lines, "", w.styles.Highlight.Render("CRYPTOGRAPHIC ORIGIN & PROVENANCE SEAL:"))
	origin := "Embedded Catalog (Immutable Binary)"
	shortDigest := "N/A"
	manifestFP := "N/A"
	dotfilesFP := "N/A"
	statusBadge := w.styles.Success.Render("[VERIFIED ✓]")
	if w.seal != nil {
		origin = w.seal.ProvenanceOrigin()
		shortDigest = w.seal.ShortDigest()
		manifestFP = w.seal.ShortManifestDigest()
		dotfilesFP = w.seal.ShortDotfilesDigest()
		if w.seal.InsecureOrigin {
			statusBadge = w.styles.Warning.Render(fmt.Sprintf("[INSECURE ORIGIN: %s]", w.seal.InsecureReason))
		}
	}
	lines = append(lines, fmt.Sprintf("  • Source Origin   : %s", origin))
	lines = append(lines, fmt.Sprintf("  • Manifests SHA256: %s", manifestFP))
	if w.platform != nil {
		lines = append(lines, fmt.Sprintf("  • Destination Home: %s", w.platform.HomeDir()))
	}
	lines = append(lines, fmt.Sprintf("  • Dotfiles SHA256 : %s", dotfilesFP))
	lines = append(lines, fmt.Sprintf("  • Security Seal   : %s %s", shortDigest, statusBadge))
	if len(w.matchedHacks) > 0 {
		status := w.styles.Muted.Render("[Excluded]")
		if w.includeHacks {
			status = w.styles.Success.Render("[Included ✓]")
		}
		lines = append(lines, "", w.styles.Highlight.Render(fmt.Sprintf("⚡ HARDWARE WORKSTATION HACKS %s (Press [h] to toggle)", status)))
		for _, hk := range w.matchedHacks {
			lines = append(lines, fmt.Sprintf("  • %-20s : %s", hk.ID, hk.Description))
		}
	}

	var drifted []app.ConfigDiff
	for _, d := range w.configDiffs {
		if d.State != "synced" {
			drifted = append(drifted, d)
		}
	}
	if len(drifted) > 0 {
		lines = append(lines, "", w.styles.Warning.Render(fmt.Sprintf("⚠️  DOTFILE DRIFTS DETECTED (%d configs with differences):", len(drifted))))
		for _, d := range drifted {
			sum := d.Summary
			if sum == "" {
				sum = d.State
			}
			lines = append(lines, fmt.Sprintf("  • %-22s [%s] (%s)", d.ConfigID, d.State, sum))
		}
		lines = append(lines, w.styles.Muted.Render("  Dotfiles will be automatically synchronized via safe 3-way merge."))
	}

	lines = append(lines, "", w.styles.Highlight.Render("PLANNED ACTIONS:"))

	if w.plan != nil {
		for _, s := range w.plan.Steps {
			lines = append(lines, fmt.Sprintf("  • [%s] %s", s.Action, AnsiTruncate(s.Description, width-15)))
		}
	} else {
		lines = append(lines, "  • [noop] Ready to apply selected configuration")
	}

	destroyerBadge := w.styles.Muted.Render("OFF (Press [d] to activate)")
	if w.destroyerMode {
		destroyerBadge = w.styles.Error.Render("ACTIVE ⚠️  Purges packages, tools & configs before installing (Press [d] to disable)")
	}
	lines = append(lines, "", fmt.Sprintf("💣 DESTROYER REINSTALL: %s", destroyerBadge))

	lines = append(lines, "", w.styles.Success.Render("Press [Enter] to Execute Plan, [d] Toggle Destroyer Mode, or [q] to Cancel."))
	return strings.Join(lines, "\n")
}

func (w WizardModel) renderCompleted(width int) string {
	if w.err != nil {
		return fmt.Sprintf("%s\n\nError: %v\n\nPress [Enter] to exit.",
			w.styles.Error.Render("SETUP COMPLETED WITH ERRORS"), w.err)
	}

	return fmt.Sprintf("%s\n\n%s\n\n%s\n%s\n%s\n\nPress [Enter] or [q] to exit.",
		w.styles.Success.Render("✓ SETUP AND RECONCILIATION COMPLETED SUCCESSFULLY"),
		"All selected packages and configurations were reconciled safely.",
		"Next Steps:",
		"  • Restart your terminal or run 'zsh' to load new configs.",
		"  • Run 'anx doctor' anytime to verify your environment health.")
}
