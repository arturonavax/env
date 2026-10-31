package tui

import (
	"context"

	tea "charm.land/bubbletea/v2"
	"github.com/arturonavax/env/internal/app"
	"github.com/arturonavax/env/internal/domain"
	"github.com/arturonavax/env/internal/security"
	"github.com/arturonavax/env/internal/theme"
)

// Model represents the top-level Bubble Tea v2 state model.
type Model struct {
	width           int
	height          int
	activeTab       TabID
	selectedIdx     int
	scrollOffset    int
	focusedPanel    string // "left" or "right" in dual-pane layouts
	accordionMode   bool
	deps            Deps
	styles          *theme.Styles
	packages        []domain.ManagedResource
	configs         []domain.ConfigurationResource
	doctorChecks    []app.DoctorCheck
	auditFindings   []domain.AuditFinding
	securityReport  *security.SecurityAuditReport
	securityProfile string
	detectedHacks   []domain.HackProfile
	showHelp        bool
	statusMsg       string
	quitting        bool
}

// NewModel instantiates the dashboard model from configured dependencies.
func NewModel(deps Deps) Model {
	st := deps.Styles
	if st == nil {
		if deps.Theme != nil {
			st = theme.NewStyles(*deps.Theme)
		} else {
			st = theme.DefaultStyles()
		}
	}

	var pkgs []domain.ManagedResource
	var cfgs []domain.ConfigurationResource
	if deps.Query != nil {
		if inv, err := deps.Query.Inventory(context.Background(), app.InventoryFilter{}); err == nil && inv != nil {
			pkgs = inv.Packages
			cfgs = inv.Configurations
		}
	}
	if len(pkgs) == 0 && deps.Catalog != nil {
		pkgs = deps.Catalog.ListPackages()
	}
	if len(cfgs) == 0 && deps.Catalog != nil {
		cfgs = deps.Catalog.ListConfigurations()
	}

	return Model{
		activeTab:       TabOverview,
		selectedIdx:     0,
		focusedPanel:    "left",
		accordionMode:   true,
		deps:            deps,
		styles:          st,
		packages:        pkgs,
		configs:         cfgs,
		securityProfile: security.ProfileWorkstation,
		statusMsg:       "Ready. Press [?] for help, [Tab] to switch view, [q] to quit.",
	}
}

// Init implements tea.Model, triggering concurrent background loading of diagnostics and security findings.
func (m Model) Init() tea.Cmd {
	ctx := context.Background()
	return tea.Batch(
		loadDoctorCmd(ctx, m.deps.Doctor),
		loadAuditCmd(ctx, m.deps.Audit),
		loadSystemAuditCmd(ctx, m.deps.Audit, m.securityProfile),
		loadHacksCmd(ctx, m.deps.Hacks),
	)
}

// ActiveTab returns the currently displayed tab index.
func (m Model) ActiveTab() TabID {
	return m.activeTab
}

// SetDimensions updates the viewport dimensions for testing or manual sizing.
func (m *Model) SetDimensions(w, h int) {
	m.width = w
	m.height = h
}
