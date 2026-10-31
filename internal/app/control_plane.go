package app

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/arturonavax/env/internal/capability"
	"github.com/arturonavax/env/internal/catalog"
	"github.com/arturonavax/env/internal/domain"
	"github.com/arturonavax/env/internal/platform"
	"github.com/arturonavax/env/internal/state"
	"github.com/arturonavax/env/internal/storage/sqlite"
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
	Plan           *domain.ExecutionPlan `json:"-"`
	PlanDigest     string                `json:"plan_digest"`
	Profile        string                `json:"profile,omitempty"`
	Requested      []string              `json:"requested,omitempty"`
	DryRun         bool                  `json:"dry_run"`
	IdempotencyKey string                `json:"idempotency_key,omitempty"`
}

// CommandService defines the mutation contract of the ANX control plane.
type CommandService interface {
	BuildPlan(ctx context.Context, req PlanRequest) (*domain.PlanSpec, error)
	Apply(ctx context.Context, req ApplyRequest) (string, error)
	Cancel(ctx context.Context, jobID string) error
	ListJobs(ctx context.Context, limit int) ([]sqlite.JobRecord, error)
	GetJob(ctx context.Context, jobID string) (sqlite.JobRecord, bool, error)
	GetJobEvents(ctx context.Context, jobID string) ([]sqlite.EventRecord, error)
}

// ControlPlaneService coordinates queries and commands across domain services.
type ControlPlaneService struct {
	platform     *platform.Info
	catalog      *catalog.Registry
	installSvc   *InstallService
	syncSvc      *SyncService
	hacksSvc     *HacksService
	toolsSvc     *ToolsService
	capabilities *capability.Registry
	store        *state.Store
	mu           sync.Mutex
	activeJobs   map[string]context.CancelFunc
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
		platform:     plat,
		catalog:      cat,
		installSvc:   installSvc,
		syncSvc:      syncSvc,
		hacksSvc:     hacksSvc,
		toolsSvc:     toolsSvc,
		capabilities: capability.NewRegistry(),
		store:        store,
		activeJobs:   make(map[string]context.CancelFunc),
	}
}

// Capabilities returns the registered capability registry.
func (c *ControlPlaneService) Capabilities() *capability.Registry {
	return c.capabilities
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

	// 5. Check Capabilities
	if c.capabilities != nil {
		if prov, err := c.capabilities.Explain(id); err == nil {
			prov.ActiveProfile = activeProfile
			return prov, nil
		}
	}

	return nil, fmt.Errorf("resource %q not found in packages, configurations, tools, hacks, or capabilities", rawID)
}

// BuildPlan produces an immutable PlanSpec and persists the plan record.
func (c *ControlPlaneService) BuildPlan(ctx context.Context, req PlanRequest) (*domain.PlanSpec, error) {
	if c.installSvc == nil {
		return nil, fmt.Errorf("install service not available")
	}
	execPlan, err := c.installSvc.Plan(ctx, req.Profile, req.Requested)
	if err != nil {
		return nil, err
	}
	execPlan.DryRun = req.DryRun

	if c.store != nil && c.store.DB() != nil && !req.DryRun {
		now := time.Now().UTC()
		if err := c.store.DB().SavePlan(ctx, sqlite.PlanRecord{
			PlanID:            execPlan.ID,
			SchemaVersion:     1,
			CanonicalIRDigest: execPlan.Digest(),
			Target:            fmt.Sprintf("%s/%s", c.platform.OS(), c.platform.Arch()),
			CreatedAt:         now,
			ExpiresAt:         now.Add(24 * time.Hour),
			Status:            "created",
		}); err != nil {
			return nil, fmt.Errorf("persisting execution plan: %w", err)
		}
	}

	return execPlan.ToSpec(), nil
}

// Apply executes a planned change without regenerating an arbitrary unverified plan,
// tracking durable job lifecycle states and events in the database.
func (c *ControlPlaneService) Apply(ctx context.Context, req ApplyRequest) (string, error) {
	if c.installSvc == nil {
		return "", fmt.Errorf("install service not available")
	}

	plan := req.Plan
	if plan == nil {
		var err error
		plan, err = c.installSvc.Plan(ctx, req.Profile, req.Requested)
		if err != nil {
			return "", fmt.Errorf("building execution plan: %w", err)
		}
	}

	// Verify digest if supplied (rejects stale plans and TOCTOU)
	if req.PlanDigest != "" && plan.Digest() != req.PlanDigest {
		return "", fmt.Errorf("plan digest mismatch: expected %s, computed %s (rejecting stale or unauthorized plan)", req.PlanDigest, plan.Digest())
	}

	plan.DryRun = req.DryRun

	if len(plan.Steps) == 0 || req.DryRun {
		return plan.ID, nil
	}

	var db *sqlite.DB
	if c.store != nil {
		db = c.store.DB()
	}

	// Durable job lifecycle: queued
	now := time.Now().UTC()
	authTime := now
	if db != nil {
		if err := db.SavePlan(ctx, sqlite.PlanRecord{
			PlanID:            plan.ID,
			SchemaVersion:     1,
			CanonicalIRDigest: plan.Digest(),
			Target:            fmt.Sprintf("%s/%s", c.platform.OS(), c.platform.Arch()),
			CreatedAt:         now,
			ExpiresAt:         now.Add(24 * time.Hour),
			AuthorizedAt:      &authTime,
			AuthorizedBy:      "user",
			Status:            "authorized",
		}); err != nil {
			return "", fmt.Errorf("persisting authorized plan: %w", err)
		}

		if err := db.CreateJob(ctx, sqlite.JobRecord{
			JobID:          plan.ID,
			PlanID:         plan.ID,
			IdempotencyKey: req.IdempotencyKey,
			State:          "queued",
			Cursor:         0,
			QueuedAt:       now,
			Attempt:        1,
		}); err != nil {
			return "", fmt.Errorf("creating durable job: %w", err)
		}

		if err := db.AppendEvent(ctx, sqlite.EventRecord{
			JobID:         plan.ID,
			EventType:     "job.started",
			SchemaVersion: 1,
			PayloadJSON:   fmt.Sprintf(`{"plan_id":%q,"steps":%d}`, plan.ID, len(plan.Steps)),
			Timestamp:     now,
		}); err != nil {
			return "", fmt.Errorf("recording job start event: %w", err)
		}

		if err := db.UpdateJobState(ctx, plan.ID, "running", 0); err != nil {
			return "", fmt.Errorf("transitioning job to running: %w", err)
		}
	}

	jobCtx, cancel := context.WithCancel(ctx)
	c.mu.Lock()
	c.activeJobs[plan.ID] = cancel
	c.mu.Unlock()
	defer func() {
		c.mu.Lock()
		delete(c.activeJobs, plan.ID)
		c.mu.Unlock()
	}()

	err := c.installSvc.Apply(jobCtx, plan)

	if db != nil {
		finTime := time.Now().UTC()
		if err != nil {
			if errors.Is(err, context.Canceled) || jobCtx.Err() != nil {
				_ = db.UpdateJobState(ctx, plan.ID, "cancelled", 130)
				_ = db.AppendEvent(ctx, sqlite.EventRecord{
					JobID:         plan.ID,
					EventType:     "job.cancelled",
					SchemaVersion: 1,
					PayloadJSON:   `{"reason":"cancelled by user or signal"}`,
					Timestamp:     finTime,
				})
			} else {
				_ = db.UpdateJobState(ctx, plan.ID, "failed", 1)
				_ = db.AppendEvent(ctx, sqlite.EventRecord{
					JobID:         plan.ID,
					EventType:     "job.failed",
					SchemaVersion: 1,
					PayloadJSON:   fmt.Sprintf(`{"error":%q}`, err.Error()),
					Timestamp:     finTime,
				})
			}
		} else {
			if updateErr := db.UpdateJobState(ctx, plan.ID, "succeeded", 0); updateErr != nil {
				_ = db.UpdateJobState(ctx, plan.ID, "needs_reconcile", 1)
				_ = db.AppendEvent(ctx, sqlite.EventRecord{
					JobID:         plan.ID,
					EventType:     "job.needs_reconcile",
					SchemaVersion: 1,
					PayloadJSON:   fmt.Sprintf(`{"error":%q,"details":"external actions completed but state persistence failed"}`, updateErr.Error()),
					Timestamp:     finTime,
				})
				return plan.ID, fmt.Errorf("actions succeeded but failed persisting completion state: %w", updateErr)
			}
			_ = db.AppendEvent(ctx, sqlite.EventRecord{
				JobID:         plan.ID,
				EventType:     "job.succeeded",
				SchemaVersion: 1,
				PayloadJSON:   fmt.Sprintf(`{"steps_completed":%d}`, len(plan.Steps)),
				Timestamp:     finTime,
			})
		}
	}

	return plan.ID, err
}

// Cancel interrupts an active job and records the cancellation event in SQLite.
func (c *ControlPlaneService) Cancel(ctx context.Context, jobID string) error {
	c.mu.Lock()
	cancel, ok := c.activeJobs[jobID]
	c.mu.Unlock()

	if c.store != nil && c.store.DB() != nil {
		_ = c.store.DB().UpdateJobState(ctx, jobID, "cancelled", 130)
		_ = c.store.DB().AppendEvent(ctx, sqlite.EventRecord{
			JobID:         jobID,
			EventType:     "job.cancelled",
			SchemaVersion: 1,
			PayloadJSON:   `{"reason":"explicit cancel requested"}`,
			Timestamp:     time.Now().UTC(),
		})
	}

	if !ok {
		return fmt.Errorf("job %s not running in active process", jobID)
	}
	cancel()
	return nil
}

// ListJobs returns the most recent execution jobs from durable SQLite storage.
func (c *ControlPlaneService) ListJobs(ctx context.Context, limit int) ([]sqlite.JobRecord, error) {
	if c.store == nil || c.store.DB() == nil {
		return nil, fmt.Errorf("state storage not available")
	}
	return c.store.DB().ListJobs(ctx, limit)
}

// GetJob returns a specific job record from durable SQLite storage.
func (c *ControlPlaneService) GetJob(ctx context.Context, jobID string) (sqlite.JobRecord, bool, error) {
	if c.store == nil || c.store.DB() == nil {
		return sqlite.JobRecord{}, false, fmt.Errorf("state storage not available")
	}
	return c.store.DB().GetJob(ctx, jobID)
}

// GetJobEvents returns chronological audit events for a job.
func (c *ControlPlaneService) GetJobEvents(ctx context.Context, jobID string) ([]sqlite.EventRecord, error) {
	if c.store == nil || c.store.DB() == nil {
		return nil, fmt.Errorf("state storage not available")
	}
	return c.store.DB().GetEventsForJob(ctx, jobID)
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
