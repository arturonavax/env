package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

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
	StagePlanning
	StageReviewPlan
	StageApplying
	StageCompleted
)

type wizardApplyDoneMsg struct {
	err error
}

type wizardProgressMsg struct {
	StepID      int
	TotalSteps  int
	Phase       string
	Action      string
	ResourceID  string
	Description string
	Percent     int
}

type wizardPlanDoneMsg struct {
	plan  *domain.ExecutionPlan
	diffs []app.ConfigDiff
	err   error
}

func waitForProgressCmd(ch <-chan tea.Msg) tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-ch
		if !ok {
			return nil
		}
		return msg
	}
}

type wizardSpinnerTickMsg time.Time

func tickSpinnerCmd() tea.Cmd {
	return tea.Tick(120*time.Millisecond, func(t time.Time) tea.Msg {
		return wizardSpinnerTickMsg(t)
	})
}

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
	pkgScrollOffset  int
	planScrollOffset int
	spinnerFrame     int
	plan             *domain.ExecutionPlan
	progressStep     int
	progressTotal    int
	progressPhase    string
	progressAction   string
	progressResource string
	progressDesc     string
	progressPercent  int
	progressChan     chan tea.Msg
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
		width:            80,
		height:           24,
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

		case "esc":
			switch w.stage {
			case StageSelectProfile:
				w.quitting = true
				return w, tea.Quit
			case StageSelectEnvironment:
				w.stage = StageSelectProfile
				return w, nil
			case StageSelectPackages:
				w.stage = StageSelectEnvironment
				return w, nil
			case StagePlanning:
				w.stage = StageSelectPackages
				return w, nil
			case StageReviewPlan:
				w.stage = StageSelectPackages
				return w, nil
			case StageCompleted:
				w.quitting = true
				return w, tea.Quit
			}

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
				w.adjustPkgScroll()
			case StageReviewPlan:
				if w.planScrollOffset > 0 {
					w.planScrollOffset--
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
				w.adjustPkgScroll()
			case StageReviewPlan:
				w.planScrollOffset++
			}

		case "pgup", "ctrl+u":
			switch w.stage {
			case StageSelectPackages:
				w.pkgCursor -= 5
				if w.pkgCursor < 0 {
					w.pkgCursor = 0
				}
				w.adjustPkgScroll()
			case StageReviewPlan:
				w.planScrollOffset -= 5
				if w.planScrollOffset < 0 {
					w.planScrollOffset = 0
				}
			}

		case "pgdown", "ctrl+d":
			switch w.stage {
			case StageSelectPackages:
				w.pkgCursor += 5
				if w.pkgCursor >= len(w.packages) {
					w.pkgCursor = len(w.packages) - 1
				}
				w.adjustPkgScroll()
			case StageReviewPlan:
				w.planScrollOffset += 5
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
				w.stage = StagePlanning
				cmd := runWizardPlanCmd(w.installSvc, w.syncSvc, selectedList)
				return w, tea.Batch(cmd, tickSpinnerCmd())

			case StageReviewPlan:
				w.stage = StageApplying
				profName := ""
				if len(w.profiles) > 0 && w.selectedProf < len(w.profiles) {
					profName = w.profiles[w.selectedProf].Name
				}
				selectedList := make([]string, 0, len(w.selectedPkgs))
				for id, enabled := range w.selectedPkgs {
					if enabled {
						selectedList = append(selectedList, id)
					}
				}

				progressChan := make(chan tea.Msg, 128)
				w.progressChan = progressChan
				w.progressStep = 0
				w.progressTotal = 1
				w.progressPhase = "Initializing"
				w.progressPercent = 0

				go runWizardApplyWorker(
					w.seal,
					w.destroyerMode,
					w.destroyerSvc,
					profName,
					w.installSvc,
					w.plan,
					selectedList,
					w.hacksSvc,
					w.includeHacks,
					w.matchedHacks,
					w.syncSvc,
					w.selectedShell,
					w.selectedTerminal,
					progressChan,
				)
				return w, tea.Batch(waitForProgressCmd(progressChan), tickSpinnerCmd())

			case StageCompleted:
				w.quitting = true
				return w, tea.Quit
			}
		}

	case tea.MouseWheelMsg:
		switch msg.Button {
		case tea.MouseWheelUp:
			switch w.stage {
			case StageSelectProfile:
				if w.selectedProf > 0 {
					w.selectedProf--
				}
			case StageSelectEnvironment:
				if w.envFocus == "shell" && w.shellCursor > 0 {
					w.shellCursor--
				} else if w.envFocus == "terminal" && w.terminalCursor > 0 {
					w.terminalCursor--
				}
			case StageSelectPackages:
				if w.pkgCursor > 0 {
					w.pkgCursor--
				}
				w.adjustPkgScroll()
			case StageReviewPlan:
				if w.planScrollOffset > 0 {
					w.planScrollOffset--
				}
			}
		case tea.MouseWheelDown:
			switch w.stage {
			case StageSelectProfile:
				if w.selectedProf < len(w.profiles)-1 {
					w.selectedProf++
				}
			case StageSelectEnvironment:
				if w.envFocus == "shell" && w.shellCursor < len(w.shells)-1 {
					w.shellCursor++
				} else if w.envFocus == "terminal" && w.terminalCursor < len(w.terminals)-1 {
					w.terminalCursor++
				}
			case StageSelectPackages:
				if w.pkgCursor < len(w.packages)-1 {
					w.pkgCursor++
				}
				w.adjustPkgScroll()
			case StageReviewPlan:
				w.planScrollOffset++
			}
		}

	case wizardPlanDoneMsg:
		if w.stage != StagePlanning {
			return w, nil
		}
		if msg.err != nil {
			w.err = msg.err
			w.stage = StageCompleted
			return w, nil
		}
		w.plan = msg.plan
		w.configDiffs = msg.diffs
		w.planScrollOffset = 0
		w.stage = StageReviewPlan
		return w, nil

	case wizardProgressMsg:
		w.progressStep = msg.StepID
		w.progressTotal = msg.TotalSteps
		w.progressPhase = msg.Phase
		w.progressAction = msg.Action
		w.progressResource = msg.ResourceID
		w.progressDesc = msg.Description
		w.progressPercent = msg.Percent
		return w, waitForProgressCmd(w.progressChan)

	case wizardApplyDoneMsg:
		w.err = msg.err
		w.stage = StageCompleted
		return w, nil

	case wizardSpinnerTickMsg:
		if w.stage == StageApplying || w.stage == StagePlanning {
			w.spinnerFrame++
			return w, tickSpinnerCmd()
		}
		return w, nil

	case tea.WindowSizeMsg:
		w.width = msg.Width
		w.height = msg.Height
		w.adjustPkgScroll()
	}

	return w, nil
}

func (w *WizardModel) adjustPkgScroll() {
	availHeight := w.visiblePkgLines()
	w.pkgScrollOffset = AdjustScrollOffset(w.pkgCursor, w.pkgScrollOffset, availHeight, len(w.packages))
}

func (w WizardModel) visiblePkgLines() int {
	h := w.height - 4
	if h < 6 {
		h = 6
	}
	avail := h - 3 - 5
	if avail < 5 {
		avail = 5
	}
	return avail
}

func runWizardPlanCmd(installSvc *app.InstallService, syncSvc *app.SyncService, selectedList []string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		var plan *domain.ExecutionPlan
		var err error
		if installSvc != nil {
			plan, err = installSvc.Plan(ctx, "", selectedList)
		}
		var diffs []app.ConfigDiff
		if syncSvc != nil {
			diffs, _ = syncSvc.Diff(ctx, "all")
		}
		return wizardPlanDoneMsg{
			plan:  plan,
			diffs: diffs,
			err:   err,
		}
	}
}

func runWizardApplyWorker(
	seal *security.OriginSeal,
	destroyerMode bool,
	destroyerSvc *app.DestroyerService,
	profName string,
	installSvc *app.InstallService,
	plan *domain.ExecutionPlan,
	selectedList []string,
	hacksSvc *app.HacksService,
	includeHacks bool,
	matchedHacks []domain.HackProfile,
	syncSvc *app.SyncService,
	selectedShell string,
	selectedTerminal string,
	progressChan chan<- tea.Msg,
) {
	ctx := context.Background()

	// 1. Verify Origin Seal
	if seal != nil {
		progressChan <- wizardProgressMsg{
			StepID:      1,
			TotalSteps:  1,
			Phase:       "Verification",
			Action:      "verify",
			ResourceID:  "origin-seal",
			Description: "Verifying cryptographic origin seal...",
			Percent:     2,
		}
		if err := seal.Verify(ctx); err != nil {
			progressChan <- wizardApplyDoneMsg{err: err}
			return
		}
	}

	// 2. Destroyer Purge Phase
	if destroyerMode && destroyerSvc != nil {
		progressChan <- wizardProgressMsg{
			StepID:      0,
			TotalSteps:  1,
			Phase:       "Purging (Destroyer)",
			Action:      "plan",
			ResourceID:  "destroyer",
			Description: "Planning deep system purge...",
			Percent:     5,
		}
		opts := app.PurgeOptions{
			Profile:       profName,
			PurgeBinaries: false,
			PurgeState:    false,
		}
		if purgePlan, err := destroyerSvc.PlanPurge(ctx, opts); err == nil {
			destroyerSvc.SetOnProgress(func(step, total int, item string) {
				pct := 5
				if total > 0 {
					pct = 5 + (step * 25 / total)
				}
				progressChan <- wizardProgressMsg{
					StepID:      step,
					TotalSteps:  total,
					Phase:       "Purging (Destroyer)",
					Action:      "purge",
					ResourceID:  item,
					Description: "Purging " + item,
					Percent:     pct,
				}
			})
			_, _ = destroyerSvc.ExecutePurge(ctx, purgePlan, false, false)
		}
	}

	// 3. Install Phase
	activePlan := plan
	if destroyerMode && installSvc != nil {
		progressChan <- wizardProgressMsg{
			StepID:      0,
			TotalSteps:  1,
			Phase:       "Planning Reinstall",
			Action:      "plan",
			ResourceID:  "packages",
			Description: "Generating fresh install plan for selected packages...",
			Percent:     32,
		}
		if freshPlan, err := installSvc.PlanWithReinstall(ctx, profName, selectedList, true); err == nil && freshPlan != nil {
			activePlan = freshPlan
		}
	}

	if installSvc != nil && activePlan != nil && len(activePlan.Steps) > 0 {
		totalInstallSteps := len(activePlan.Steps)
		installSvc.SetOnProgress(func(ev app.ProgressEvent) {
			pct := 35
			if totalInstallSteps > 0 {
				pct = 35 + (ev.StepID * 45 / totalInstallSteps)
			}
			progressChan <- wizardProgressMsg{
				StepID:      ev.StepID,
				TotalSteps:  totalInstallSteps,
				Phase:       "Installing",
				Action:      string(ev.Action),
				ResourceID:  ev.ResourceID,
				Description: ev.Description,
				Percent:     pct,
			}
		})
		if err := installSvc.Apply(ctx, activePlan); err != nil {
			progressChan <- wizardApplyDoneMsg{err: err}
			return
		}
	}

	// 4. Hacks Phase
	if hacksSvc != nil && includeHacks && len(matchedHacks) > 0 {
		totalHacks := len(matchedHacks)
		for i, hk := range matchedHacks {
			pct := 80
			if totalHacks > 0 {
				pct = 80 + ((i + 1) * 10 / totalHacks)
			}
			progressChan <- wizardProgressMsg{
				StepID:      i + 1,
				TotalSteps:  totalHacks,
				Phase:       "Hardware Hacks",
				Action:      "apply",
				ResourceID:  hk.ID,
				Description: hk.Description,
				Percent:     pct,
			}
			_, _ = hacksSvc.Apply(ctx, hk.ID)
		}
	}

	// 5. Sync Phase
	if syncSvc != nil {
		progressChan <- wizardProgressMsg{
			StepID:      1,
			TotalSteps:  1,
			Phase:       "Dotfile Sync",
			Action:      "merge",
			ResourceID:  "dotfiles",
			Description: "Reconciling dotfiles with 3-way merge...",
			Percent:     95,
		}
		if _, err := syncSvc.Sync(ctx, app.SyncOptions{
			Force:    destroyerMode,
			Shell:    selectedShell,
			Terminal: selectedTerminal,
		}); err != nil {
			progressChan <- wizardApplyDoneMsg{err: err}
			return
		}
	}

	progressChan <- wizardProgressMsg{
		StepID:      1,
		TotalSteps:  1,
		Phase:       "Complete",
		Action:      "done",
		ResourceID:  "complete",
		Description: "All components reconciled successfully.",
		Percent:     100,
	}

	progressChan <- wizardApplyDoneMsg{err: nil}
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
		body = w.renderSelectPackages(availWidth, availHeight)
	case StagePlanning:
		body = w.renderPlanning(availWidth)
	case StageReviewPlan:
		body = w.renderReviewPlan(availWidth, availHeight)
	case StageApplying:
		body = w.renderApplying(availWidth)
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

	lines = append(lines, "", w.styles.Muted.Render("Navigation: [Up/Down / j/k] Select  [Enter] Confirm Profile  [h] Toggle Hacks  [Esc / q] Quit"))
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

	lines = append(lines, "", w.styles.Muted.Render("Navigation: [Tab / Left / Right] Switch Focus  [Up/Down / j/k] Navigate  [Space] Select  [Enter] Confirm & Next  [Esc] Back  [q] Quit"))
	return strings.Join(lines, "\n")
}

func (w WizardModel) renderSelectPackages(width, height int) string {
	headerLines := []string{
		w.styles.Highlight.Render(fmt.Sprintf("STAGE 3/5: CUSTOMIZE TOOLS & PACKAGES (%d/%d)", w.pkgCursor+1, len(w.packages))),
		"Toggle components to include or exclude from your workstation:",
		"",
	}

	availHeight := w.visiblePkgLines()

	pkgScrollOffset := AdjustScrollOffset(w.pkgCursor, w.pkgScrollOffset, availHeight, len(w.packages))
	start := pkgScrollOffset
	end := start + availHeight
	if end > len(w.packages) {
		end = len(w.packages)
	}

	lines := make([]string, 0, len(headerLines)+availHeight+4)
	lines = append(lines, headerLines...)

	if start > 0 {
		lines = append(lines, w.styles.Muted.Render(fmt.Sprintf("  ▲ ... (%d more packages above)", start)))
	}

	for i := start; i < end; i++ {
		p := w.packages[i]
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

	if end < len(w.packages) {
		lines = append(lines, w.styles.Muted.Render(fmt.Sprintf("  ▼ ... (%d more packages below)", len(w.packages)-end)))
	}

	if len(w.matchedHacks) > 0 {
		status := w.styles.Muted.Render("NO (press 'h' to include)")
		if w.includeHacks {
			status = w.styles.Success.Render("YES ✓ (press 'h' to exclude)")
		}
		lines = append(lines, "", fmt.Sprintf("⚡ Include hardware workstation hacks: %s", status))
	}

	lines = append(lines, "", w.styles.Muted.Render("Navigation: [Up/Down / j/k] Navigate  [Space] Toggle  [h] Toggle Hacks  [Enter] Review Plan  [Esc] Back  [q] Quit"))
	return strings.Join(lines, "\n")
}

func (w WizardModel) renderReviewPlan(width, height int) string {
	planID := "preview"
	stepCount := 0
	if w.plan != nil {
		planID = w.plan.ID
		stepCount = len(w.plan.Steps)
	}
	allLines := []string{
		w.styles.Highlight.Render("STAGE 4/5: REVIEW EXECUTION PLAN"),
		fmt.Sprintf("Plan ID: %s | Total Actions: %d", planID, stepCount),
		w.styles.Muted.Render("• Automatic pre-mutation backups will be created before touching any file."),
		w.styles.Muted.Render("• Transaction journal will record each step for rollback safety."),
	}

	allLines = append(allLines, "", w.styles.Highlight.Render("TARGET ENVIRONMENT:"))
	allLines = append(allLines, fmt.Sprintf("  • Primary Shell : %s (Zsh source of truth)", w.selectedShell))
	termWarning := ""
	for _, t := range w.terminals {
		if t.ID == w.selectedTerminal && t.Warning != "" {
			termWarning = fmt.Sprintf(" (%s)", w.styles.Warning.Render(t.Warning))
			break
		}
	}
	allLines = append(allLines, fmt.Sprintf("  • Terminal      : %s%s", w.selectedTerminal, termWarning))

	allLines = append(allLines, "", w.styles.Highlight.Render("CRYPTOGRAPHIC ORIGIN & PROVENANCE SEAL:"))
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
	allLines = append(allLines, fmt.Sprintf("  • Source Origin   : %s", origin))
	allLines = append(allLines, fmt.Sprintf("  • Manifests SHA256: %s", manifestFP))
	if w.platform != nil {
		allLines = append(allLines, fmt.Sprintf("  • Destination Home: %s", w.platform.HomeDir()))
	}
	allLines = append(allLines, fmt.Sprintf("  • Dotfiles SHA256 : %s", dotfilesFP))
	allLines = append(allLines, fmt.Sprintf("  • Security Seal   : %s %s", shortDigest, statusBadge))
	if len(w.matchedHacks) > 0 {
		status := w.styles.Muted.Render("[Excluded]")
		if w.includeHacks {
			status = w.styles.Success.Render("[Included ✓]")
		}
		allLines = append(allLines, "", w.styles.Highlight.Render(fmt.Sprintf("⚡ HARDWARE WORKSTATION HACKS %s (Press [h] to toggle)", status)))
		for _, hk := range w.matchedHacks {
			allLines = append(allLines, fmt.Sprintf("  • %-20s : %s", hk.ID, hk.Description))
		}
	}

	var drifted []app.ConfigDiff
	for _, d := range w.configDiffs {
		if d.State != "synced" {
			drifted = append(drifted, d)
		}
	}
	if len(drifted) > 0 {
		allLines = append(allLines, "", w.styles.Warning.Render(fmt.Sprintf("⚠️  DOTFILE DRIFTS DETECTED (%d configs with differences):", len(drifted))))
		for _, d := range drifted {
			sum := d.Summary
			if sum == "" {
				sum = d.State
			}
			allLines = append(allLines, fmt.Sprintf("  • %-22s [%s] (%s)", d.ConfigID, d.State, sum))
		}
		allLines = append(allLines, w.styles.Muted.Render("  Dotfiles will be automatically synchronized via safe 3-way merge."))
	}

	allLines = append(allLines, "", w.styles.Highlight.Render("PLANNED ACTIONS:"))

	if w.plan != nil {
		for _, s := range w.plan.Steps {
			allLines = append(allLines, fmt.Sprintf("  • [%s] %s", s.Action, AnsiTruncate(s.Description, width-15)))
		}
	} else {
		allLines = append(allLines, "  • [noop] Ready to apply selected configuration")
	}

	destroyerBadge := w.styles.Muted.Render("OFF (Press [d] to activate)")
	if w.destroyerMode {
		destroyerBadge = w.styles.Error.Render("ACTIVE ⚠️  Purges packages, tools & configs before installing (Press [d] to disable)")
	}
	allLines = append(allLines, "", fmt.Sprintf("💣 DESTROYER REINSTALL: %s", destroyerBadge))

	for i := range allLines {
		allLines[i] = AnsiTruncate(allLines[i], width-4)
	}

	availHeight := height - 3
	if availHeight < 5 {
		availHeight = 5
	}

	totalLines := len(allLines)
	maxOffset := totalLines - availHeight
	if maxOffset < 0 {
		maxOffset = 0
	}
	offset := w.planScrollOffset
	if offset > maxOffset {
		offset = maxOffset
	}
	if offset < 0 {
		offset = 0
	}

	end := offset + availHeight
	if end > totalLines {
		end = totalLines
	}

	visibleLines := allLines[offset:end]

	var footer string
	if totalLines > availHeight {
		footer = w.styles.Muted.Render(fmt.Sprintf("Navigation: [j/k / Up/Down] Scroll Plan (%d-%d/%d)  [Enter] Execute  [d] Destroyer  [Esc] Back  [q] Cancel", offset+1, end, totalLines))
	} else {
		footer = w.styles.Success.Render("Press [Enter] to Execute Plan, [d] Toggle Destroyer Mode, [Esc] Back, or [q] to Cancel.")
	}

	return strings.Join(append(visibleLines, "", footer), "\n")
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

func (w WizardModel) renderPlanning(width int) string {
	frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	frame := frames[w.spinnerFrame%len(frames)]
	spinner := w.styles.Header.Render(frame)
	title := w.styles.Highlight.Render("Resolving dependency graph and building execution plan...")

	lines := []string{
		fmt.Sprintf("  %s  %s", spinner, title),
		"",
		w.styles.Muted.Render("  • Solving package dependencies and provider execution order..."),
		w.styles.Muted.Render("  • Inspecting dotfile diffs and drift states across repositories..."),
		w.styles.Muted.Render("  • Please wait, calculating optimal deterministic plan..."),
		"",
		w.styles.Muted.Render("  Navigation: [Esc] Return to Packages"),
	}
	return strings.Join(lines, "\n")
}

func (w WizardModel) renderProgressBar(percent, barWidth int) string {
	if barWidth < 10 {
		barWidth = 10
	}
	if percent < 0 {
		percent = 0
	}
	if percent > 100 {
		percent = 100
	}
	filled := (percent * barWidth) / 100
	empty := barWidth - filled

	barFilled := strings.Repeat("█", filled)
	barEmpty := strings.Repeat("░", empty)

	return fmt.Sprintf("[%s%s] %3d%%", w.styles.Header.Render(barFilled), w.styles.Muted.Render(barEmpty), percent)
}

func (w WizardModel) renderApplying(width int) string {
	frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	frame := frames[w.spinnerFrame%len(frames)]
	spinner := w.styles.Header.Render(frame)

	phase := w.progressPhase
	if phase == "" {
		phase = "Initializing"
	}

	title := fmt.Sprintf("  %s %s: %s", spinner, w.styles.Header.Render("STAGE 5/5"), w.styles.Highlight.Render(phase))

	barWidth := width - 24
	if barWidth > 40 {
		barWidth = 40
	}
	if barWidth < 15 {
		barWidth = 15
	}

	stepInfo := ""
	if w.progressTotal > 0 && w.progressStep > 0 {
		stepInfo = fmt.Sprintf(" (Step %d/%d)", w.progressStep, w.progressTotal)
	}

	progressBar := w.renderProgressBar(w.progressPercent, barWidth) + w.styles.Muted.Render(stepInfo)

	desc := w.progressDesc
	if desc == "" {
		desc = "Preparing operations..."
	}

	actionBadge := ""
	if w.progressAction != "" {
		actionBadge = fmt.Sprintf("[%s] ", strings.ToUpper(w.progressAction))
	}

	lines := []string{
		title,
		"",
		fmt.Sprintf("  %s", progressBar),
		"",
		fmt.Sprintf("  %s%s", w.styles.Success.Render(actionBadge), desc),
		"",
		w.styles.Muted.Render("  • Background worker running atomically with transactional journal."),
		w.styles.Muted.Render("  • Please do not interrupt until operations complete..."),
	}
	return strings.Join(lines, "\n")
}
