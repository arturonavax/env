package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/arturonavax/env/internal/catalog"
	"github.com/arturonavax/env/internal/domain"
	"github.com/arturonavax/env/internal/platform"
	"github.com/arturonavax/env/internal/state"
	"github.com/arturonavax/env/internal/tools"
)

// PurgeOptions controls how destroyer purges components based on manifest declarations.
type PurgeOptions struct {
	Profile       string `json:"profile,omitempty"`
	PurgeBinaries bool   `json:"purge_binaries,omitempty"`
	PurgeState    bool   `json:"purge_state,omitempty"`
}

// DestroyerPlan outlines every item that will be uninstalled and purged based on manifest declarations.
type DestroyerPlan struct {
	Profile             string               `json:"profile,omitempty"`
	PackagesToUninstall []string             `json:"packages_to_uninstall"`
	ToolsDirectories    []string             `json:"tools_directories"`
	ConfigTargets       []string             `json:"config_targets"`
	ConfigMarkers       map[string][2]string `json:"config_markers,omitempty"` // target path -> [markerStart, markerEnd]
	ArchiveCleanupPaths []string             `json:"archive_cleanup_paths,omitempty"`
	Binaries            []string             `json:"binaries,omitempty"`
	StateDirectories    []string             `json:"state_directories"`
	CacheDirectories    []string             `json:"cache_directories"`
}

// DestroyerResult summarizes the operations performed by a destroyer action.
type DestroyerResult struct {
	Profile             string   `json:"profile,omitempty"`
	UninstalledPackages int      `json:"uninstalled_packages"`
	PurgedConfigs       int      `json:"purged_configs"`
	PurgedPaths         []string `json:"purged_paths"`
	Reinstalled         bool     `json:"reinstalled"`
	ReinstallPlanID     string   `json:"reinstall_plan_id,omitempty"`
}

// DestroyerOption configures DestroyerService.
type DestroyerOption func(*DestroyerService)

// WithDestroyerToolsRegistry sets the personal tools registry for dynamic executable discovery.
func WithDestroyerToolsRegistry(tr *tools.Registry) DestroyerOption {
	return func(d *DestroyerService) {
		d.toolsReg = tr
	}
}

// DestroyerService coordinates aggressive teardown, purge, and reinstallation driven by YAML manifests.
type DestroyerService struct {
	platform   *platform.Info
	catalog    *catalog.Registry
	toolsReg   *tools.Registry
	installSvc *InstallService
	syncSvc    *SyncService
	store      *state.Store
}

// NewDestroyerService constructs a DestroyerService.
func NewDestroyerService(
	plat *platform.Info,
	cat *catalog.Registry,
	installSvc *InstallService,
	syncSvc *SyncService,
	store *state.Store,
	opts ...DestroyerOption,
) *DestroyerService {
	svc := &DestroyerService{
		platform:   plat,
		catalog:    cat,
		installSvc: installSvc,
		syncSvc:    syncSvc,
		store:      store,
	}
	for _, opt := range opts {
		opt(svc)
	}
	return svc
}

// PlanPurge inspects declarative manifests and system state to discover all managed packages,
// configurations, archive remnants, tools, and state to purge for the given options.
func (d *DestroyerService) PlanPurge(_ context.Context, opts PurgeOptions) (*DestroyerPlan, error) {
	plan := &DestroyerPlan{
		Profile:             opts.Profile,
		PackagesToUninstall: make([]string, 0, 32),
		ToolsDirectories:    make([]string, 0, 4),
		ConfigTargets:       make([]string, 0, 32),
		ConfigMarkers:       make(map[string][2]string),
		ArchiveCleanupPaths: make([]string, 0, 8),
		Binaries:            make([]string, 0, 8),
		StateDirectories:    make([]string, 0, 4),
		CacheDirectories:    make([]string, 0, 4),
	}

	// 1. Discover packages to uninstall dynamically from manifests and profiles
	seenPkgs := make(map[string]bool)
	if opts.Profile != "" && opts.Profile != "all" && d.catalog != nil {
		profilePkgs, err := d.catalog.ResolveProfilePackages(opts.Profile)
		if err != nil {
			return nil, fmt.Errorf("resolving profile packages for %q: %w", opts.Profile, err)
		}
		for _, id := range profilePkgs {
			if !seenPkgs[id] {
				seenPkgs[id] = true
				plan.PackagesToUninstall = append(plan.PackagesToUninstall, id)
			}
		}
	} else {
		// All packages: check installed store records first
		if d.store != nil {
			for id := range d.store.AllResources() {
				if !seenPkgs[id] {
					seenPkgs[id] = true
					plan.PackagesToUninstall = append(plan.PackagesToUninstall, id)
				}
			}
		}
		// If store is empty, fall back to all declared packages in catalog
		if len(plan.PackagesToUninstall) == 0 && d.catalog != nil {
			for _, pkg := range d.catalog.ListPackages() {
				if !seenPkgs[pkg.ID] {
					seenPkgs[pkg.ID] = true
					plan.PackagesToUninstall = append(plan.PackagesToUninstall, pkg.ID)
				}
			}
		}
	}
	sort.Strings(plan.PackagesToUninstall)

	// 2. Discover configuration targets and declared markers from manifests
	if d.catalog != nil {
		seenTargets := make(map[string]bool)
		var cfgs []domain.ConfigurationResource
		if opts.Profile != "" && opts.Profile != "all" {
			var err error
			cfgs, err = d.catalog.ResolveProfileConfigurations(opts.Profile)
			if err != nil {
				return nil, fmt.Errorf("resolving profile configurations for %q: %w", opts.Profile, err)
			}
		} else {
			cfgs = d.catalog.ListConfigurations()
		}

		for _, cfg := range cfgs {
			target := d.platform.ExpandPath(cfg.TargetForOS(d.platform.OS()))
			if target != "" && !seenTargets[target] {
				if _, err := os.Lstat(target); err == nil {
					seenTargets[target] = true
					plan.ConfigTargets = append(plan.ConfigTargets, target)
					plan.ConfigMarkers[target] = [2]string{cfg.MarkerStart, cfg.MarkerEnd}
				}
			}
		}
		sort.Strings(plan.ConfigTargets)

		// 3. Discover archive cleanup paths declared in manifests/archives.yaml
		for _, arch := range d.catalog.ListArchives() {
			for _, cleanupPath := range arch.CleanupPaths {
				target := d.platform.ExpandPath(cleanupPath)
				if target != "" && !seenTargets[target] {
					if _, err := os.Lstat(target); err == nil {
						seenTargets[target] = true
						plan.ArchiveCleanupPaths = append(plan.ArchiveCleanupPaths, target)
					}
				}
			}
		}
		sort.Strings(plan.ArchiveCleanupPaths)
	}

	// 4. Discover tools directories
	toolsBin := d.platform.ToolsBinDir()
	if info, err := os.Stat(toolsBin); err == nil && info.IsDir() {
		plan.ToolsDirectories = append(plan.ToolsDirectories, toolsBin)
	}

	// 5. Discover platform binaries dynamically from manifests (if requested)
	if opts.PurgeBinaries {
		seenBinaries := make(map[string]bool)
		localBin := d.platform.LocalBinDir()

		// Discover from catalog packages (binary_name and binary_aliases)
		if d.catalog != nil {
			for _, pkg := range d.catalog.ListPackages() {
				if pkg.BinaryName != "" {
					p := filepath.Join(localBin, pkg.BinaryName)
					if _, err := os.Lstat(p); err == nil && !seenBinaries[p] {
						seenBinaries[p] = true
						plan.Binaries = append(plan.Binaries, p)
					}
				}
				if spec, ok := d.catalog.GetPackageSpec(pkg.ID); ok {
					for _, alias := range spec.BinaryAliases {
						p := filepath.Join(localBin, alias)
						if _, err := os.Lstat(p); err == nil && !seenBinaries[p] {
							seenBinaries[p] = true
							plan.Binaries = append(plan.Binaries, p)
						}
					}
				}
			}
		}

		// Discover from tools registry manifests
		if d.toolsReg != nil {
			for _, tool := range d.toolsReg.List() {
				if tool.Executable != "" {
					p := filepath.Join(localBin, tool.Executable)
					if _, err := os.Lstat(p); err == nil && !seenBinaries[p] {
						seenBinaries[p] = true
						plan.Binaries = append(plan.Binaries, p)
					}
				}
			}
		}

		// Ensure anx core platform binaries and standard multicall links are discovered
		for _, b := range []string{"anx", "aitools", "safe-suspend"} {
			p := filepath.Join(localBin, b)
			if _, err := os.Lstat(p); err == nil && !seenBinaries[p] {
				seenBinaries[p] = true
				plan.Binaries = append(plan.Binaries, p)
			}
		}
		sort.Strings(plan.Binaries)
	}

	// 6. Discover state & cache directories
	stateDir := d.platform.StateDir()
	if _, err := os.Stat(stateDir); err == nil {
		plan.StateDirectories = append(plan.StateDirectories, stateDir)
	}
	cfgDir := d.platform.ConfigDir()
	if _, err := os.Stat(cfgDir); err == nil {
		plan.StateDirectories = append(plan.StateDirectories, cfgDir)
	}

	cacheDir := d.platform.CacheDir()
	if _, err := os.Stat(cacheDir); err == nil {
		plan.CacheDirectories = append(plan.CacheDirectories, cacheDir)
	}

	return plan, nil
}

// ExecutePurge performs the actual removal of packages, configurations, archives, and tools.
func (d *DestroyerService) ExecutePurge(
	ctx context.Context,
	plan *DestroyerPlan,
	dryRun bool,
	purgeState bool,
) (*DestroyerResult, error) {
	totalPaths := len(plan.ConfigTargets) + len(plan.ArchiveCleanupPaths) + len(plan.ToolsDirectories) + len(plan.CacheDirectories) + len(plan.Binaries)
	res := &DestroyerResult{
		Profile:     plan.Profile,
		PurgedPaths: make([]string, 0, totalPaths),
	}

	if dryRun {
		res.UninstalledPackages = len(plan.PackagesToUninstall)
		res.PurgedConfigs = len(plan.ConfigTargets) + len(plan.ArchiveCleanupPaths)
		res.PurgedPaths = append(res.PurgedPaths, plan.ConfigTargets...)
		res.PurgedPaths = append(res.PurgedPaths, plan.ArchiveCleanupPaths...)
		res.PurgedPaths = append(res.PurgedPaths, plan.ToolsDirectories...)
		res.PurgedPaths = append(res.PurgedPaths, plan.Binaries...)
		if purgeState {
			res.PurgedPaths = append(res.PurgedPaths, plan.StateDirectories...)
		}
		res.PurgedPaths = append(res.PurgedPaths, plan.CacheDirectories...)
		return res, nil
	}

	// 1. Uninstall managed packages
	if len(plan.PackagesToUninstall) > 0 && d.installSvc != nil {
		_ = d.installSvc.Uninstall(ctx, plan.PackagesToUninstall)
		res.UninstalledPackages = len(plan.PackagesToUninstall)
	}

	// 2. Purge configuration targets with declared markers
	for _, target := range plan.ConfigTargets {
		markerStart, markerEnd := "", ""
		if markers, ok := plan.ConfigMarkers[target]; ok {
			markerStart = markers[0]
			markerEnd = markers[1]
		}
		if err := PurgeConfigTargetWithMarker(target, markerStart, markerEnd); err == nil {
			res.PurgedConfigs++
			res.PurgedPaths = append(res.PurgedPaths, target)
		}
	}

	// 3. Purge archive cleanup paths declared in manifests/archives.yaml
	for _, target := range plan.ArchiveCleanupPaths {
		if err := os.RemoveAll(target); err == nil {
			res.PurgedConfigs++
			res.PurgedPaths = append(res.PurgedPaths, target)
		}
	}

	// 4. Purge personal tools directories
	for _, dir := range plan.ToolsDirectories {
		if err := os.RemoveAll(dir); err == nil {
			res.PurgedPaths = append(res.PurgedPaths, dir)
		}
	}

	// 5. Purge platform binaries if discovered
	for _, binPath := range plan.Binaries {
		if err := os.Remove(binPath); err == nil {
			res.PurgedPaths = append(res.PurgedPaths, binPath)
		}
	}

	// 6. Purge caches
	for _, dir := range plan.CacheDirectories {
		if err := os.RemoveAll(dir); err == nil {
			res.PurgedPaths = append(res.PurgedPaths, dir)
		}
	}

	// 7. Purge state directories if requested
	if purgeState {
		for _, dir := range plan.StateDirectories {
			if err := os.RemoveAll(dir); err == nil {
				res.PurgedPaths = append(res.PurgedPaths, dir)
			}
		}
	} else if d.store != nil {
		// Reset store records without deleting directory structure
		_ = d.store.Reset()
	}

	return res, nil
}

// DestroyerReinstall purges everything for the target profile and performs a fresh installation and dotfile sync.
func (d *DestroyerService) DestroyerReinstall(
	ctx context.Context,
	profile string,
	dryRun bool,
) (*DestroyerResult, error) {
	opts := PurgeOptions{
		Profile:       profile,
		PurgeBinaries: false,
		PurgeState:    false,
	}

	purgePlan, err := d.PlanPurge(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("planning destroyer purge: %w", err)
	}

	res, err := d.ExecutePurge(ctx, purgePlan, dryRun, false)
	if err != nil {
		return nil, fmt.Errorf("executing destroyer purge: %w", err)
	}

	if dryRun {
		if d.installSvc != nil {
			if plan, err := d.installSvc.Plan(ctx, profile, nil); err == nil {
				res.Reinstalled = true
				res.ReinstallPlanID = plan.ID
			}
		}
		return res, nil
	}

	// Execute fresh install
	if d.installSvc != nil {
		freshPlan, err := d.installSvc.Plan(ctx, profile, nil)
		if err != nil {
			return res, fmt.Errorf("planning fresh installation: %w", err)
		}

		if err := d.installSvc.Apply(ctx, freshPlan); err != nil {
			return res, fmt.Errorf("applying fresh installation: %w", err)
		}

		res.Reinstalled = true
		res.ReinstallPlanID = freshPlan.ID
	}

	// Execute fresh dotfile sync
	if d.syncSvc != nil {
		_, _ = d.syncSvc.Sync(ctx, SyncOptions{Force: true})
	}

	return res, nil
}

// DestroyerUninstall executes a system purge and removes managed packages, configurations, and binaries.
func (d *DestroyerService) DestroyerUninstall(
	ctx context.Context,
	profile string,
	dryRun bool,
	purgeBinaries bool,
) (*DestroyerResult, error) {
	opts := PurgeOptions{
		Profile:       profile,
		PurgeBinaries: purgeBinaries,
		PurgeState:    true,
	}

	purgePlan, err := d.PlanPurge(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("planning destroyer purge: %w", err)
	}

	res, err := d.ExecutePurge(ctx, purgePlan, dryRun, true)
	if err != nil {
		return nil, fmt.Errorf("executing destroyer purge: %w", err)
	}

	return res, nil
}

// PurgeConfigTarget removes or cleanses a target configuration file or directory using default markers.
func PurgeConfigTarget(path string) error {
	return PurgeConfigTargetWithMarker(path, "", "")
}

// PurgeConfigTargetWithMarker removes or cleanses a target configuration file or directory using declared markers.
func PurgeConfigTargetWithMarker(path, markerStart, markerEnd string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return nil
	}

	if info.IsDir() {
		return os.RemoveAll(path)
	}

	if markerStart == "" {
		markerStart = "# >>> arturonavax/env >>>"
	}
	if markerEnd == "" {
		markerEnd = "# <<< arturonavax/env <<<"
	}

	// If file has marker block, strip the marker block rather than deleting user file
	data, err := os.ReadFile(path)
	if err == nil {
		content := string(data)
		if strings.Contains(content, markerStart) {
			startIdx := strings.Index(content, markerStart)
			endIdx := strings.Index(content, markerEnd)
			if startIdx != -1 && endIdx != -1 && endIdx > startIdx {
				endIdx += len(markerEnd)
				// Remove trailing newline if present
				if endIdx < len(content) && content[endIdx] == '\n' {
					endIdx++
				}
				cleaned := content[:startIdx] + content[endIdx:]
				return os.WriteFile(path, []byte(cleaned), info.Mode())
			}
		}
	}

	// Otherwise remove the file/symlink
	return os.Remove(path)
}
