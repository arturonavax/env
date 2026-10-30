package app

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/arturonavax/env/internal/catalog"
	"github.com/arturonavax/env/internal/domain"
	"github.com/arturonavax/env/internal/platform"
	"github.com/arturonavax/env/internal/state"
)

// InventoryFilter filters resources returned by Inventory.
type InventoryFilter struct {
	Kind     domain.ResourceKind `json:"kind,omitempty"`
	Category string              `json:"category,omitempty"`
	Profile  string              `json:"profile,omitempty"`
}

// InventoryResult holds the list of resources matching an inventory query.
type InventoryResult struct {
	Packages       []domain.ManagedResource       `json:"packages,omitempty"`
	Configurations []domain.ConfigurationResource `json:"configurations,omitempty"`
	Profiles       []string                       `json:"profiles,omitempty"`
}

// QueryService defines the read-only inspection contract of the ANX control plane.
type QueryService interface {
	Inventory(ctx context.Context, filter InventoryFilter) (*InventoryResult, error)
	Observe(ctx context.Context, ids []domain.ResourceID) ([]domain.ObservedState, error)
	Explain(ctx context.Context, id domain.ResourceID) (*domain.Provenance, error)
}

// PlanRequest specifies inputs to build an ExecutionPlan or PlanSpec.
type PlanRequest struct {
	Profile      string   `json:"profile"`
	Requested    []string `json:"requested,omitempty"`
	IncludeHacks bool     `json:"include_hacks"`
	DryRun       bool     `json:"dry_run"`
}

// ApplyRequest specifies options to apply a plan.
type ApplyRequest struct {
	PlanDigest string `json:"plan_digest"`
	DryRun     bool   `json:"dry_run"`
}

// CommandService defines the mutation contract of the ANX control plane.
type CommandService interface {
	BuildPlan(ctx context.Context, req PlanRequest) (*domain.PlanSpec, error)
	Apply(ctx context.Context, req ApplyRequest) (string, error)
	Cancel(ctx context.Context, jobID string) error
}

// ControlPlaneService coordinates queries and commands across domain services.
type ControlPlaneService struct {
	platform   *platform.Info
	catalog    *catalog.Registry
	installSvc *InstallService
	syncSvc    *SyncService
	hacksSvc   *HacksService
	toolsSvc   *ToolsService
	store      *state.Store
}

// NewControlPlaneService creates a unified ControlPlaneService.
func NewControlPlaneService(
	plat *platform.Info,
	cat *catalog.Registry,
	installSvc *InstallService,
	syncSvc *SyncService,
	hacksSvc *HacksService,
	toolsSvc *ToolsService,
	store *state.Store,
) *ControlPlaneService {
	return &ControlPlaneService{
		platform:   plat,
		catalog:    cat,
		installSvc: installSvc,
		syncSvc:    syncSvc,
		hacksSvc:   hacksSvc,
		toolsSvc:   toolsSvc,
		store:      store,
	}
}

// Inventory queries catalogued resources by optional filter.
func (c *ControlPlaneService) Inventory(_ context.Context, filter InventoryFilter) (*InventoryResult, error) {
	res := &InventoryResult{}

	if c.catalog != nil {
		if filter.Kind == "" || filter.Kind == domain.KindPackage {
			pkgs := c.catalog.ListPackages()
			for _, p := range pkgs {
				if filter.Category != "" && p.Category != filter.Category {
					continue
				}
				res.Packages = append(res.Packages, p)
			}
		}

		if filter.Kind == "" || filter.Kind == domain.KindConfig {
			cfgs := c.catalog.ListConfigurations()
			for _, cfg := range cfgs {
				if filter.Category != "" && cfg.Group != filter.Category {
					continue
				}
				res.Configurations = append(res.Configurations, cfg)
			}
		}

		for _, prof := range c.catalog.ListProfiles() {
			res.Profiles = append(res.Profiles, prof.Name)
		}
	}

	return res, nil
}

// Observe checks live state across requested resources.
func (c *ControlPlaneService) Observe(ctx context.Context, ids []domain.ResourceID) ([]domain.ObservedState, error) {
	states := make([]domain.ObservedState, 0, len(ids))

	for _, rawID := range ids {
		id := normalizeResourceID(rawID)
		st := domain.ObservedState{
			ResourceID:   rawID,
			LastObserved: time.Now(),
		}

		// 1. Try Package
		if c.catalog != nil {
			if pkg, ok := c.catalog.GetPackage(id); ok {
				st.Kind = domain.KindPackage
				if c.installSvc != nil && c.installSvc.provider != nil {
					installed, ver, _ := c.installSvc.provider.IsInstalled(ctx, pkg)
					st.Exists = installed
					st.InstalledVersion = ver
					if installed {
						st.Drift = "clean"
					} else {
						st.Drift = "missing"
					}
					st.Healthy = true
				}
				states = append(states, st)
				continue
			}
		}

		// 2. Try Config
		if c.syncSvc != nil {
			if cfg, ok := c.catalog.GetConfiguration(id); ok {
				st.Kind = domain.KindConfig
				diffs, err := c.syncSvc.Diff(ctx, cfg.Group)
				if err == nil {
					for _, d := range diffs {
						if d.ConfigID == cfg.ID {
							st.Drift = d.State
							st.Exists = d.State != "missing"
							st.Path = d.Target
							break
						}
					}
				}
				states = append(states, st)
				continue
			}
		}

		// 3. Fallback unknown
		st.Drift = "unknown"
		states = append(states, st)
	}

	return states, nil
}

// Explain resolves provenance, policies, conditions, and reasons for any resource or setting.
func (c *ControlPlaneService) Explain(ctx context.Context, rawID domain.ResourceID) (*domain.Provenance, error) {
	id := normalizeResourceID(rawID)

	activeProfile := "developer"
	if c.store != nil {
		activeProfile = c.store.ActiveProfile()
	}

	// 1. Check Packages
	if c.catalog != nil {
		if pkg, ok := c.catalog.GetPackage(id); ok {
			prov := &domain.Provenance{
				ResourceID:    domain.ResourceID(pkg.ID),
				Kind:          domain.KindPackage,
				Origin:        "manifests/packages",
				ActiveProfile: activeProfile,
				Conditions:    pkg.SupportedOS,
				WinningPolicy: fmt.Sprintf("strategy:%s", pkg.Strategy),
				Reason:        pkg.Description,
			}
			for _, dep := range pkg.DependsOn {
				prov.Dependencies = append(prov.Dependencies, domain.ResourceID(dep))
			}
			return prov, nil
		}
	}

	// 2. Check Configurations
	if c.catalog != nil {
		if cfg, ok := c.catalog.GetConfiguration(id); ok {
			prov := &domain.Provenance{
				ResourceID:    domain.ResourceID(cfg.ID),
				Kind:          domain.KindConfig,
				Origin:        cfg.Source,
				ActiveProfile: activeProfile,
				Conditions:    cfg.SupportedOS,
				SelectedBy:    []string{cfg.Group},
				WinningPolicy: fmt.Sprintf("strategy:%s", cfg.Strategy),
				Reason:        cfg.Description,
			}
			return prov, nil
		}
	}

	// 3. Check Personal Tools
	if c.toolsSvc != nil {
		if tool, err := c.toolsSvc.Get(ctx, id); err == nil && tool.ID != "" {
			prov := &domain.Provenance{
				ResourceID:    domain.ResourceID(tool.ID),
				Kind:          domain.KindTool,
				Origin:        tool.Path,
				ActiveProfile: activeProfile,
				WinningPolicy: "decoupled-personal-tool",
				Reason:        tool.Description,
			}
			return prov, nil
		}
	}

	// 4. Check Workstation Hacks
	if c.hacksSvc != nil {
		for _, h := range c.hacksSvc.List() {
			if h.ID == id {
				conditions := []string{}
				if h.Match.DMIVendor != "" {
					conditions = append(conditions, "vendor:"+h.Match.DMIVendor)
				}
				if h.Match.DMIProduct != "" {
					conditions = append(conditions, "product:"+h.Match.DMIProduct)
				}
				prov := &domain.Provenance{
					ResourceID:    domain.ResourceID(h.ID),
					Kind:          domain.KindHack,
					Origin:        "manifests/hacks",
					ActiveProfile: activeProfile,
					Conditions:    conditions,
					WinningPolicy: "hardware-dmi-match",
					Reason:        h.Description,
				}
				return prov, nil
			}
		}
	}

	return nil, fmt.Errorf("resource %q not found in packages, configurations, tools, or hacks", rawID)
}

// BuildPlan produces an immutable PlanSpec.
func (c *ControlPlaneService) BuildPlan(ctx context.Context, req PlanRequest) (*domain.PlanSpec, error) {
	if c.installSvc == nil {
		return nil, fmt.Errorf("install service not available")
	}
	execPlan, err := c.installSvc.Plan(ctx, req.Profile, req.Requested)
	if err != nil {
		return nil, err
	}
	execPlan.DryRun = req.DryRun
	return execPlan.ToSpec(), nil
}

// Apply executes a planned change.
func (c *ControlPlaneService) Apply(ctx context.Context, req ApplyRequest) (string, error) {
	if c.installSvc == nil {
		return "", fmt.Errorf("install service not available")
	}
	// Verify digest if supplied
	plan, err := c.installSvc.Plan(ctx, "", nil)
	if err != nil {
		return "", err
	}
	if req.PlanDigest != "" && plan.Digest() != req.PlanDigest {
		return "", fmt.Errorf("plan digest mismatch: expected %s, computed %s", req.PlanDigest, plan.Digest())
	}
	plan.DryRun = req.DryRun
	err = c.installSvc.Apply(ctx, plan)
	return plan.ID, err
}

// Cancel interrupts an active job.
func (c *ControlPlaneService) Cancel(_ context.Context, jobID string) error {
	return fmt.Errorf("job %s cancellation not implemented for synchronous runner", jobID)
}

func normalizeResourceID(id domain.ResourceID) string {
	s := string(id)
	for _, prefix := range []string{"package.", "config.", "tool.", "hack.", "capability."} {
		if strings.HasPrefix(s, prefix) {
			return strings.TrimPrefix(s, prefix)
		}
	}
	return s
}
