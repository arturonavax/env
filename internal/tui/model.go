package tui

import (
	"context"
	"os"
	"slices"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/arturonavax/env/internal/app"
	"github.com/arturonavax/env/internal/domain"
	"github.com/arturonavax/env/internal/platform"
	"github.com/arturonavax/env/internal/security"
	"github.com/arturonavax/env/internal/theme"
)

// Model represents the top-level Bubble Tea v2 state model.
type Model struct {
	width                    int
	height                   int
	activeTab                TabID
	selectedIdx              int
	scrollOffset             int
	focusedPanel             string // "left" or "right" in dual-pane layouts
	accordionMode            bool
	deps                     Deps
	styles                   *theme.Styles
	packages                 []domain.ManagedResource
	tools                    []domain.ManagedResource
	depsList                 []domain.ManagedResource
	inspectingPackage        *domain.ManagedResource
	toolsSelectedIdx         int
	toolsScrollOffset        int
	depsSelectedIdx          int
	depsScrollOffset         int
	configs                  []domain.ConfigurationResource
	integratedConfigs        []domain.ConfigurationResource
	notIntegratedConfigs     []domain.ConfigurationResource
	intConfigsSelectedIdx    int
	intConfigsScrollOffset   int
	notIntConfigsSelectedIdx int
	notIntConfigsScrollOffset int
	doctorChecks             []app.DoctorCheck
	auditFindings            []domain.AuditFinding
	securityReport           *security.SecurityAuditReport
	securityProfile          string
	detectedHacks            []domain.HackProfile
	updateCheck              *UpdateCheckResult
	showHelp                 bool
	statusMsg                string
	quitting                 bool
}

// partitionAndSortPackages separates packages into direct Tools and Dependencies,
// pinning anx to index 0 of Tools, and ordering each section by Category then ID.
func partitionAndSortPackages(pkgs []domain.ManagedResource) ([]domain.ManagedResource, []domain.ManagedResource, []domain.ManagedResource) {
	var anxPkg *domain.ManagedResource
	var tools []domain.ManagedResource
	var deps []domain.ManagedResource

	for _, p := range pkgs {
		id := strings.ToLower(p.ID)
		if id == "anx" {
			anxCopy := p
			anxPkg = &anxCopy
			continue
		}
		if _, ok := domain.DirectToolsSet[id]; ok {
			tools = append(tools, p)
		} else {
			deps = append(deps, p)
		}
	}

	// Sort tools by Category, then ID
	slices.SortFunc(tools, func(a, b domain.ManagedResource) int {
		if c := strings.Compare(a.Category, b.Category); c != 0 {
			return c
		}
		return strings.Compare(a.ID, b.ID)
	})

	// Sort dependencies by Category, then ID
	slices.SortFunc(deps, func(a, b domain.ManagedResource) int {
		if c := strings.Compare(a.Category, b.Category); c != 0 {
			return c
		}
		return strings.Compare(a.ID, b.ID)
	})

	// anx is always pinned to index 0 of tools and combined
	finalTools := make([]domain.ManagedResource, 0, len(tools)+1)
	if anxPkg != nil {
		finalTools = append(finalTools, *anxPkg)
	}
	finalTools = append(finalTools, tools...)

	combined := make([]domain.ManagedResource, 0, len(finalTools)+len(deps))
	combined = append(combined, finalTools...)
	combined = append(combined, deps...)

	return finalTools, deps, combined
}

// partitionAndSortConfigs separates configurations into Integrated and Not Integrated,
// ordering each group by category/group then ID.
func partitionAndSortConfigs(cfgs []domain.ConfigurationResource, plat *platform.Info) ([]domain.ConfigurationResource, []domain.ConfigurationResource, []domain.ConfigurationResource) {
	var integrated []domain.ConfigurationResource
	var notIntegrated []domain.ConfigurationResource

	for _, c := range cfgs {
		targetPath := c.Target
		if plat != nil {
			targetPath = plat.ExpandPath(c.TargetForOS(plat.OS()))
		}
		if fi, err := os.Lstat(targetPath); err == nil && fi != nil {
			integrated = append(integrated, c)
		} else {
			notIntegrated = append(notIntegrated, c)
		}
	}

	sortFn := func(a, b domain.ConfigurationResource) int {
		if c := strings.Compare(a.Group, b.Group); c != 0 {
			return c
		}
		return strings.Compare(a.ID, b.ID)
	}

	slices.SortFunc(integrated, sortFn)
	slices.SortFunc(notIntegrated, sortFn)

	combined := make([]domain.ConfigurationResource, 0, len(integrated)+len(notIntegrated))
	combined = append(combined, integrated...)
	combined = append(combined, notIntegrated...)

	return integrated, notIntegrated, combined
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

	tools, depsList, sortedPkgs := partitionAndSortPackages(pkgs)
	intCfgs, notIntCfgs, sortedCfgs := partitionAndSortConfigs(cfgs, deps.Platform)

	if deps.RepoRoot == "" && deps.Sync != nil {
		deps.RepoRoot = deps.Sync.RepoRoot()
	}

	return Model{
		activeTab:            TabOverview,
		selectedIdx:          0,
		focusedPanel:         "left",
		accordionMode:        true,
		deps:                 deps,
		styles:               st,
		packages:             sortedPkgs,
		tools:                tools,
		depsList:             depsList,
		configs:              sortedCfgs,
		integratedConfigs:    intCfgs,
		notIntegratedConfigs: notIntCfgs,
		securityProfile:      security.ProfileWorkstation,
		statusMsg:            "Ready. Press [?] for help, [Tab] to switch view, [q] to quit.",
	}
}

// Init implements tea.Model, triggering concurrent background loading of diagnostics, security findings, and update checks.
func (m Model) Init() tea.Cmd {
	ctx := context.Background()
	return tea.Batch(
		loadDoctorCmd(ctx, m.deps.Doctor),
		loadAuditCmd(ctx, m.deps.Audit),
		loadSystemAuditCmd(ctx, m.deps.Audit, m.securityProfile),
		loadHacksCmd(ctx, m.deps.Hacks),
		loadUpdateCheckCmd(ctx, m.deps.RepoRoot, m.deps.Sync),
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
