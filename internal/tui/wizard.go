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
	"github.com/arturonavax/env/internal/theme"
)

// WizardStage tracks which screen of the wizard is active.
type WizardStage int

const (
	StageSelectProfile WizardStage = iota
	StageSelectPackages
	StageReviewPlan
	StageApplying
	StageCompleted
)

// WizardModel represents the interactive setup interview wizard.
type WizardModel struct {
	width        int
	height       int
	stage        WizardStage
	styles       *theme.Styles
	platform     *platform.Info
	catalog      *catalog.Registry
	installSvc   *app.InstallService
	profiles     []domain.Profile
	selectedProf int
	packages     []domain.ManagedResource
	selectedPkgs map[string]bool
	pkgCursor    int
	plan         *domain.ExecutionPlan
	progressMsg  string
	err          error
	quitting     bool
}

// NewWizardModel creates an upfront interactive wizard model.
func NewWizardModel(
	plat *platform.Info,
	cat *catalog.Registry,
	svc *app.InstallService,
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

	return WizardModel{
		stage:        StageSelectProfile,
		styles:       theme.DefaultStyles(),
		platform:     plat,
		catalog:      cat,
		installSvc:   svc,
		profiles:     profs,
		selectedProf: 0,
		packages:     pkgs,
		selectedPkgs: selected,
		pkgCursor:    0,
	}
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
			case StageSelectPackages:
				if w.pkgCursor < len(w.packages)-1 {
					w.pkgCursor++
				}
			}

		case " ":
			if w.stage == StageSelectPackages && len(w.packages) > 0 {
				pkgID := w.packages[w.pkgCursor].ID
				w.selectedPkgs[pkgID] = !w.selectedPkgs[pkgID]
			}

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
				w.stage = StageApplying
				if w.installSvc != nil && w.plan != nil {
					if err := w.installSvc.Apply(context.Background(), w.plan); err != nil {
						w.err = err
					}
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
		w.styles.Highlight.Render("STAGE 1/4: SELECT INSTALLATION PROFILE"),
		"Choose a preset profile matching your machine's primary role:",
		"",
	}

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

	lines = append(lines, "", w.styles.Muted.Render("Navigation: [Up/Down / j/k] Select  [Enter] Confirm Profile  [q] Quit"))
	return strings.Join(lines, "\n")
}

func (w WizardModel) renderSelectPackages(width int) string {
	lines := []string{
		w.styles.Highlight.Render("STAGE 2/4: CUSTOMIZE TOOLS & PACKAGES"),
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
		row := fmt.Sprintf("%s%s %-12s %-12s : %s", cursor, check, p.ID, p.Category, p.Description)
		if i == w.pkgCursor {
			lines = append(lines, w.styles.Highlight.Render(AnsiTruncate(row, width-4)))
		} else {
			lines = append(lines, AnsiTruncate(row, width-4))
		}
	}

	lines = append(lines, "", w.styles.Muted.Render("Navigation: [Up/Down / j/k] Navigate  [Space] Toggle  [Enter] Review Plan  [q] Quit"))
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
		w.styles.Highlight.Render("STAGE 3/4: REVIEW EXECUTION PLAN"),
		fmt.Sprintf("Plan ID: %s | Total Actions: %d", planID, stepCount),
		w.styles.Muted.Render("• Automatic pre-mutation backups will be created before touching any file."),
		w.styles.Muted.Render("• Transaction journal will record each step for rollback safety."),
		"",
		w.styles.Highlight.Render("PLANNED ACTIONS:"),
	}

	if w.plan != nil {
		for _, s := range w.plan.Steps {
			lines = append(lines, fmt.Sprintf("  • [%s] %s", s.Action, AnsiTruncate(s.Description, width-15)))
		}
	} else {
		lines = append(lines, "  • [noop] Ready to apply selected configuration")
	}

	lines = append(lines, "", w.styles.Success.Render("Press [Enter] to Execute Plan, or [q] to Cancel."))
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
