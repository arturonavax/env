package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/arturonavax/env/internal/catalog"
	"github.com/arturonavax/env/internal/domain"
	"github.com/arturonavax/env/internal/execution"
	"github.com/arturonavax/env/internal/journal"
	"github.com/arturonavax/env/internal/manifest"
	"github.com/arturonavax/env/internal/platform"
	"github.com/arturonavax/env/internal/providers"
	"github.com/arturonavax/env/internal/security"
	"github.com/arturonavax/env/internal/snapshot"
	"github.com/arturonavax/env/internal/state"
)

// InstallService orchestrates planning and execution of packages and configuration sync steps.
type InstallService struct {
	mu         sync.RWMutex
	platform   *platform.Info
	catalog    *catalog.Registry
	journal    *journal.FileJournal
	snapshots  *snapshot.Manager
	provider   providers.Provider
	store      *state.Store
	sync       *SyncService
	engine     *execution.Engine
	onProgress EventHandler
}

// InstallDeps bundles the collaborators required by InstallService.
type InstallDeps struct {
	Platform   *platform.Info
	Catalog    *catalog.Registry
	Journal    *journal.FileJournal
	Snapshots  *snapshot.Manager
	Provider   providers.Provider
	Store      *state.Store
	Sync       *SyncService
	OnProgress EventHandler
}

// NewInstallService constructs an InstallService from its dependencies.
func NewInstallService(deps InstallDeps) *InstallService {
	workers := runtime.NumCPU() * 2
	if workers < 4 {
		workers = 4
	}
	if workers > 32 {
		workers = 32
	}
	return &InstallService{
		platform:   deps.Platform,
		catalog:    deps.Catalog,
		journal:    deps.Journal,
		snapshots:  deps.Snapshots,
		provider:   deps.Provider,
		store:      deps.Store,
		sync:       deps.Sync,
		engine:     execution.NewEngine(workers, execution.WithContinueOnError(true), execution.WithRollbackOnError(false)),
		onProgress: deps.OnProgress,
	}
}

// SetOnProgress sets or updates the progress event listener.
func (s *InstallService) SetOnProgress(h EventHandler) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.onProgress = h
}

func (s *InstallService) getOnProgress() EventHandler {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.onProgress
}

// InstallLogPath returns the persistent installation log file path.
func (s *InstallService) InstallLogPath() string {
	if s.platform != nil && s.platform.StateDir() != "" {
		return filepath.Join(s.platform.StateDir(), "logs", "install.log")
	}
	return filepath.Join(os.Getenv("HOME"), ".local", "state", "anx", "logs", "install.log")
}

// StrategyDependencyResolver optionally provides host-specific winning strategy and dependency selection.
type StrategyDependencyResolver interface {
	SelectStrategy(pkgID string) (manifest.StrategySpec, bool)
	EffectiveDependencies(pkgID string) []string
}

// baseToolRank defines explicit priority order for core environment tools.
// Base tools must always be installed before other direct tools and dependencies
// so that a functional core environment is guaranteed even if later steps fail.
var baseToolRank = map[string]int{
	"curl":            1,
	"build-essential": 1,
	"git":             2,
	"htop":            3,
	"ghostty":         4,
	"zsh":             5,
	"herdr":           6,
	"neovim":          7,
	"nvim":            7,
	"anx":             8,
}

func (s *InstallService) isSpecSupportedOnHost(spec manifest.PackageSpec) bool {
	if s.platform == nil {
		return true
	}
	return spec.SupportsOS(s.platform.OS()) && spec.SupportsDisplayServer(s.platform.DisplayServer())
}

// EffectiveDependencies returns the host-applicable dependency IDs for pkgID.
func (s *InstallService) EffectiveDependencies(pkgID string) []string {
	if resolver, ok := s.provider.(StrategyDependencyResolver); ok {
		return resolver.EffectiveDependencies(pkgID)
	}
	if s.catalog == nil {
		return nil
	}
	spec, ok := s.catalog.GetPackageSpec(pkgID)
	if !ok || !s.isSpecSupportedOnHost(spec) {
		return nil
	}
	out := make([]string, 0, len(spec.DependsOn))
	for _, depID := range spec.DependsOn {
		if depSpec, found := s.catalog.GetPackageSpec(depID); found && !s.isSpecSupportedOnHost(depSpec) {
			continue
		}
		out = append(out, depID)
	}
	return out
}

func (s *InstallService) effectiveDeps(spec manifest.PackageSpec) []string {
	if resolver, ok := s.provider.(StrategyDependencyResolver); ok {
		return resolver.EffectiveDependencies(spec.ID)
	}
	return s.EffectiveDependencies(spec.ID)
}

// WinningStrategy returns the host-selected strategy for pkgID, or fallback if unresolved.
func (s *InstallService) WinningStrategy(pkgID string, fallback domain.StrategyType) domain.StrategyType {
	if resolver, ok := s.provider.(StrategyDependencyResolver); ok {
		if st, found := resolver.SelectStrategy(pkgID); found && st.Type != "" {
			return domain.StrategyType(st.Type)
		}
	}
	return fallback
}

// PrioritizePackageIDs reorders package IDs into an optimal topological sequence:
// Tier 0: Base core tools (git, curl, ghostty, zsh, herdr, neovim/nvim, anx)
// Tier 1: Direct interactive user tools (defined in domain.DirectToolsSet)
// Tier 2: Dependencies directly or transitively required by direct tools
// Tier 3: Standalone / unlinked dependencies
func (s *InstallService) PrioritizePackageIDs(ids []string) []string {
	if len(ids) <= 1 {
		return slices.Clone(ids)
	}

	toolDeps := make(map[string]bool)
	if s.catalog != nil {
		var collectDeps func(id string)
		collectDeps = func(id string) {
			for _, dep := range s.EffectiveDependencies(id) {
				depNorm := strings.ToLower(dep)
				if !toolDeps[depNorm] {
					toolDeps[depNorm] = true
					collectDeps(depNorm)
				}
			}
		}
		for toolID := range domain.DirectToolsSet {
			collectDeps(strings.ToLower(toolID))
		}
	}

	type rankedID struct {
		id   string
		tier int
		rank int
	}

	ranked := make([]rankedID, len(ids))
	for i, id := range ids {
		norm := strings.ToLower(id)
		var tier, rank int
		if r, ok := baseToolRank[norm]; ok {
			tier = 0
			rank = r
		} else if _, ok := domain.DirectToolsSet[norm]; ok {
			tier = 1
			rank = 0
		} else if toolDeps[norm] {
			tier = 2
			rank = 0
		} else {
			tier = 3
			rank = 0
		}
		ranked[i] = rankedID{id: id, tier: tier, rank: rank}
	}

	slices.SortStableFunc(ranked, func(a, b rankedID) int {
		if a.tier != b.tier {
			return a.tier - b.tier
		}
		if a.tier == 0 && a.rank != b.rank {
			return a.rank - b.rank
		}
		return strings.Compare(a.id, b.id)
	})

	result := make([]string, len(ranked))
	for i, r := range ranked {
		result[i] = r.id
	}
	return result
}

// determinePackageLocks derives fine-grained lock domains for pkg to prevent package manager
// database contention while permitting concurrent execution of independent downloads, language runtimes,
// and file synchronization.
func (s *InstallService) determinePackageLocks(pkg domain.ManagedResource) []string {
	locks := []string{"pkg:" + pkg.ID}
	if pkg.BinaryName != "" {
		locks = append(locks, "bin:"+pkg.BinaryName)
	}

	strat := s.WinningStrategy(pkg.ID, pkg.Strategy)
	if strat == "" {
		strat = domain.StrategyNative
	}

	switch strat {
	case domain.StrategyNative:
		pmLock := "lock:pm-native"
		if s.platform != nil {
			switch s.platform.OS() {
			case "darwin":
				pmLock = "lock:pm-brew"
			case "windows":
				pmLock = "lock:pm-windows"
			}
		}
		locks = append(locks, pmLock)
	case domain.StrategyLanguagePkg:
		eco := pkg.Ecosystem
		if eco == "" {
			eco = "generic"
		}
		locks = append(locks, "runtime:"+eco)
	case domain.StrategyGitHubRelease, domain.StrategyBinary, domain.StrategyScript, domain.StrategySource:
		// Standalone strategies do not hold a shared package manager lock, enabling concurrent downloads and builds.
	}

	return locks
}

// Plan resolves requested packages (or a profile) into a dependency-ordered plan.
// Already-installed packages are skipped, making Apply idempotent and Resume trivial.
func (s *InstallService) Plan(ctx context.Context, profileName string, requested []string) (*domain.ExecutionPlan, error) {
	return s.PlanWithReinstall(ctx, profileName, requested, false)
}

// PlanWithReinstall resolves requested packages into a dependency-ordered plan.
// If force is true, already-installed checks are bypassed to schedule fresh installation steps.
func (s *InstallService) PlanWithReinstall(ctx context.Context, profileName string, requested []string, force bool) (*domain.ExecutionPlan, error) {
	now := time.Now()
	plan := &domain.ExecutionPlan{
		ID:        fmt.Sprintf("plan-%d", now.UnixNano()),
		CreatedAt: now,
		Platform:  s.platform.OS(),
		Profile:   profileName,
		Steps:     make([]domain.PlanStep, 0, 16),
	}

	ids := requested
	includeConfigs := false
	if len(ids) == 0 {
		if profileName == "" {
			profileName = "developer"
			plan.Profile = profileName
		}
		resolved, err := s.catalog.ResolveProfilePackages(profileName)
		if err != nil {
			return nil, err
		}
		ids = resolved
		includeConfigs = true
	}

	prioritizedIDs := s.PrioritizePackageIDs(ids)
	ordered, err := s.catalog.ResolveInstallOrderWithDeps(prioritizedIDs, s.isSpecSupportedOnHost, s.effectiveDeps)
	if err != nil {
		return nil, err
	}

	stepID := 1
	installStepIDs := make(map[string]int, len(ordered))
	for _, pkg := range ordered {
		if !pkg.SupportsOS(s.platform.OS()) || !pkg.SupportsDisplayServer(s.platform.DisplayServer()) {
			continue
		}
		if !force {
			installed, _, _ := s.provider.IsInstalled(ctx, pkg)
			if installed {
				continue
			}
		}
		winningStrat := s.WinningStrategy(pkg.ID, pkg.Strategy)
		plan.Steps = append(plan.Steps, domain.PlanStep{
			ID:          stepID,
			ResourceID:  pkg.ID,
			Action:      domain.ActionInstall,
			Description: fmt.Sprintf("Install %s (%s)", pkg.Name, pkg.Description),
			Provider:    s.provider.Name(),
			Strategy:    string(winningStrat),
			DependsOn:   installDependencies(s.EffectiveDependencies(pkg.ID), installStepIDs),
			Locks:       s.determinePackageLocks(pkg),
			IsExclusive: false,
			Status:      domain.StepPending,
		})
		installStepIDs[pkg.ID] = stepID
		stepID++
	}

	if includeConfigs && s.sync != nil {
		cfgs, err := s.catalog.ResolveProfileConfigurations(profileName)
		if err != nil {
			return nil, err
		}
		var dotfileCfgs, osAssetCfgs, osScriptCfgs []domain.ConfigurationResource
		for _, cfg := range cfgs {
			if !cfg.SupportsOS(s.platform.OS()) {
				continue
			}
			if cfg.Group != "osconfig" {
				dotfileCfgs = append(dotfileCfgs, cfg)
			} else if len(cfg.PostSyncCmds) > 0 {
				osScriptCfgs = append(osScriptCfgs, cfg)
			} else {
				osAssetCfgs = append(osAssetCfgs, cfg)
			}
		}
		appendSyncCfgs := func(list []domain.ConfigurationResource, descPrefix string) {
			for _, cfg := range list {
				target := s.platform.ExpandPath(cfg.TargetForOS(s.platform.OS()))
				plan.Steps = append(plan.Steps, domain.PlanStep{
					ID:          stepID,
					ResourceID:  cfg.ID,
					Action:      domain.ActionSync,
					Description: fmt.Sprintf("%s %s", descPrefix, cfg.Name),
					SourcePath:  cfg.Source,
					TargetPath:  target,
					Locks:       []string{"file:" + target},
					Status:      domain.StepPending,
				})
				stepID++
			}
		}
		appendSyncCfgs(dotfileCfgs, "Synchronize")
		appendSyncCfgs(osAssetCfgs, "Synchronize")
		appendSyncCfgs(osScriptCfgs, "Apply OS Configuration:")
	}

	return plan, nil
}

// installDependencies maps a package's declared dependencies to the plan step IDs that install them.
// Dependencies absent from stepIDs are already satisfied (installed or filtered out) and need no edge.
func installDependencies(depIDs []string, stepIDs map[string]int) []int {
	var deps []int
	for _, dep := range depIDs {
		if id, ok := stepIDs[dep]; ok {
			deps = append(deps, id)
		}
	}
	return deps
}

// Apply executes plan under an exclusive advisory lock with journaling and compensation on failure.
func (s *InstallService) Apply(ctx context.Context, plan *domain.ExecutionPlan) error {
	if plan == nil || len(plan.Steps) == 0 {
		return nil
	}
	if plan.DryRun {
		return nil
	}

	if s.platform != nil && s.platform.HasSudo() && !s.platform.IsRoot() && security.IsSudoAuthenticated() {
		stopKeepalive := security.StartSudoKeepalive(ctx)
		defer stopKeepalive()
	}

	lock, err := state.AcquireLock(s.platform.StateDir(), "apply")
	if err != nil {
		return domain.NewExitError(domain.ExitPreflightFailed, err)
	}
	defer func() { _ = lock.Release() }()

	txID := fmt.Sprintf("tx-%d", time.Now().UnixNano())
	if err := s.journalAppend(domain.JournalEntry{
		TxID: txID, Timestamp: time.Now(), Type: "begin",
		Details: fmt.Sprintf("plan %s (%d steps)", plan.ID, len(plan.Steps)),
	}); err != nil {
		return domain.NewExitError(domain.ExitRollbackOrPartial, fmt.Errorf("journal begin failed: %w", err))
	}

	logPath := s.InstallLogPath()
	_ = os.MkdirAll(filepath.Dir(logPath), 0o755)
	logFile, _ := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	var logMu sync.Mutex
	writeLog := func(format string, args ...any) {
		if logFile == nil {
			return
		}
		ts := time.Now().Format("2006-01-02 15:04:05")
		msg := fmt.Sprintf(format, args...)
		logMu.Lock()
		defer logMu.Unlock()
		_, _ = fmt.Fprintf(logFile, "[%s] %s\n", ts, msg)
	}
	if logFile != nil {
		defer logFile.Close()
		writeLog("================================================================================")
		writeLog("ANX INSTALLATION SESSION STARTED (Plan: %s, Profile: %s, Steps: %d)", plan.ID, plan.Profile, len(plan.Steps))
		writeLog("================================================================================")
	}

	var (
		detailsMu   sync.Mutex
		stepDetails = map[int]string{}
	)
	recordDetail := func(stepID int, detail string) {
		if detail == "" {
			return
		}
		detailsMu.Lock()
		stepDetails[stepID] = detail
		detailsMu.Unlock()
	}

	var installTasks, dotfileTasks, osAssetTasks, osScriptTasks []execution.Task
	for i := range plan.Steps {
		task := s.buildTaskWithDetail(txID, plan.Steps[i], writeLog, recordDetail)
		if task.Step.Action != domain.ActionSync {
			installTasks = append(installTasks, task)
			continue
		}
		task.Step.DependsOn = nil
		if s.catalog != nil {
			if cfg, ok := s.catalog.GetConfiguration(task.Step.ResourceID); ok && cfg.Group == "osconfig" {
				if len(cfg.PostSyncCmds) > 0 {
					osScriptTasks = append(osScriptTasks, task)
				} else {
					osAssetTasks = append(osAssetTasks, task)
				}
				continue
			}
		}
		dotfileTasks = append(dotfileTasks, task)
	}

	byID := make(map[int]*domain.PlanStep, len(plan.Steps))
	for i := range plan.Steps {
		byID[plan.Steps[i].ID] = &plan.Steps[i]
	}

	onResult := func(res execution.Result) {
		step := byID[res.StepID]
		if step == nil {
			return
		}
		step.Status = res.Status
		if res.Error != nil {
			step.Error = res.Error.Error()
		}
		detailsMu.Lock()
		outLine := stepDetails[res.StepID]
		detailsMu.Unlock()
		if progressFn := s.getOnProgress(); progressFn != nil {
			progressFn(ProgressEvent{
				StepID: step.ID, TotalSteps: len(plan.Steps), ResourceID: step.ResourceID,
				Action: step.Action, Description: step.Description, Status: res.Status, Error: step.Error, OutputLine: outLine,
			})
		}
	}

	var execErrs []error
	for _, phaseTasks := range [][]execution.Task{installTasks, dotfileTasks, osAssetTasks, osScriptTasks} {
		if len(phaseTasks) == 0 {
			continue
		}
		if ctx.Err() != nil {
			execErrs = append(execErrs, execution.ErrExecutionInterrupted)
			break
		}
		if _, err := s.engine.ExecuteTasks(ctx, phaseTasks, onResult); err != nil {
			execErrs = append(execErrs, err)
			if errors.Is(err, execution.ErrExecutionInterrupted) {
				break
			}
		}
	}
	execErr := errors.Join(execErrs...)

	if execErr != nil {
		writeLog("================================================================================")
		writeLog("INSTALLATION SESSION COMPLETED WITH FAILURES:\n%s", execErr.Error())
		writeLog("================================================================================")
		_ = s.journalAppend(domain.JournalEntry{
			TxID: txID, Timestamp: time.Now(), Type: "rollback", Details: execErr.Error(),
		})
		if errors.Is(execErr, execution.ErrExecutionInterrupted) {
			return domain.NewExitError(domain.ExitInterrupted, execErr)
		}
		return domain.NewExitError(domain.ExitRollbackOrPartial, execErr)
	}

	writeLog("================================================================================")
	writeLog("INSTALLATION SESSION COMPLETED SUCCESSFULLY: %d steps reconciled", len(plan.Steps))
	writeLog("================================================================================")

	if err := s.journalAppend(domain.JournalEntry{
		TxID: txID, Timestamp: time.Now(), Type: "commit", Details: "all steps succeeded",
	}); err != nil {
		return domain.NewExitError(domain.ExitRollbackOrPartial, fmt.Errorf("journal commit failed: %w", err))
	}
	return nil
}

func (s *InstallService) buildTask(txID string, step domain.PlanStep, writeLog func(string, ...any)) execution.Task {
	return s.buildTaskWithDetail(txID, step, writeLog, nil)
}

func (s *InstallService) buildTaskWithDetail(txID string, step domain.PlanStep, writeLog func(string, ...any), recordDetail func(int, string)) execution.Task {
	task := execution.Task{Step: step, Exclusive: step.IsExclusive, Locks: step.Locks}

	switch step.Action {
	case domain.ActionInstall:
		pkg, ok := s.catalog.GetPackage(step.ResourceID)
		if !ok {
			pkg = domain.ManagedResource{ID: step.ResourceID, Name: step.ResourceID}
		}
		task.Timeout = 30 * time.Minute
		task.Execute = func(ctx context.Context) error {
			outHandler := func(line string) {
				if writeLog != nil {
					writeLog("[STEP %d] [OUTPUT] [%s] %s", step.ID, step.ResourceID, line)
				}
				if progressFn := s.getOnProgress(); progressFn != nil {
					progressFn(ProgressEvent{
						StepID:      step.ID,
						ResourceID:  step.ResourceID,
						Action:      step.Action,
						Description: step.Description,
						Status:      domain.StepRunning,
						OutputLine:  line,
					})
				}
			}
			taskCtx := security.ContextWithOutputHandler(ctx, outHandler)
			if writeLog != nil {
				writeLog("[STEP %d] [START] Installing %s (%s)", step.ID, pkg.Name, pkg.Description)
			}
			outHandler(fmt.Sprintf("Starting installation of %s...", pkg.Name))

			if err := s.journalStep(txID, "step_start", step, ""); err != nil {
				return fmt.Errorf("journal step_start: %w", err)
			}
			if err := s.provider.Install(taskCtx, pkg); err != nil {
				if writeLog != nil {
					writeLog("[STEP %d] [FAILED] %s: %v", step.ID, pkg.Name, err)
				}
				_ = s.journalStep(txID, "step_failed", step, err.Error())
				outHandler(fmt.Sprintf("Failed to install %s: %v", pkg.Name, err))
				return err
			}
			_, ver, _ := s.provider.IsInstalled(taskCtx, pkg)
			if err := s.recordInstalled(pkg, ver); err != nil {
				if writeLog != nil {
					writeLog("[STEP %d] [FAILED] Recording %s: %v", step.ID, pkg.Name, err)
				}
				_ = s.journalStep(txID, "step_failed", step, err.Error())
				return fmt.Errorf("recording installed %s: %w", pkg.ID, err)
			}
			if writeLog != nil {
				writeLog("[STEP %d] [SUCCESS] Installed %s (%s)", step.ID, pkg.Name, ver)
			}
			outHandler(fmt.Sprintf("Successfully installed %s (%s)", pkg.Name, ver))
			if err := s.journalStep(txID, "step_done", step, ""); err != nil {
				return fmt.Errorf("journal step_done: %w", err)
			}
			return nil
		}
		task.Compensate = func(ctx context.Context) error {
			// Only compensate if ANX recorded the package as managed by ANX
			if s.store != nil {
				rec, exists := s.store.GetResource(pkg.ID)
				if !exists || !rec.ManagedByAnx {
					return nil
				}
			}
			if err := s.provider.Uninstall(ctx, pkg); err != nil {
				return err
			}
			if s.store != nil {
				_ = s.store.RemoveResource(pkg.ID)
			}
			return nil
		}

	case domain.ActionSync:
		var snapID string
		task.Execute = func(ctx context.Context) error {
			if writeLog != nil {
				writeLog("[STEP %d] [SYNC-START] Synchronizing %s", step.ID, step.ResourceID)
			}
			if err := s.journalStep(txID, "step_start", step, ""); err != nil {
				return fmt.Errorf("journal step_start: %w", err)
			}
			if s.sync == nil {
				return errors.New("sync service not configured")
			}
			syncCtx := security.ContextWithOutputHandler(ctx, func(line string) {
				if writeLog != nil {
					writeLog("[STEP %d] [OUTPUT] [%s] %s", step.ID, step.ResourceID, line)
				}
				if progressFn := s.getOnProgress(); progressFn != nil {
					progressFn(ProgressEvent{
						StepID:      step.ID,
						ResourceID:  step.ResourceID,
						Action:      step.Action,
						Description: step.Description,
						Status:      domain.StepRunning,
						OutputLine:  line,
					})
				}
			})
			res, err := s.sync.SyncOne(syncCtx, step.ResourceID, SyncOptions{})
			if err != nil {
				if writeLog != nil {
					writeLog("[STEP %d] [SYNC-FAILED] %s: %v", step.ID, step.ResourceID, err)
				}
				_ = s.journalStep(txID, "step_failed", step, err.Error())
				return err
			}
			snapID = res.SnapshotID
			if res.Detail != "" && recordDetail != nil {
				recordDetail(step.ID, res.Detail)
			}
			if writeLog != nil {
				writeLog("[STEP %d] [SYNC-SUCCESS] Synchronized %s (snapshot: %s)", step.ID, step.ResourceID, snapID)
			}
			if err := s.journalStep(txID, "step_done", step, ""); err != nil {
				return fmt.Errorf("journal step_done: %w", err)
			}
			return nil
		}
		task.Compensate = func(_ context.Context) error {
			if snapID == "" || s.snapshots == nil {
				return nil
			}
			return s.snapshots.Restore(snapID)
		}
	}
	return task
}

// Uninstall removes managed packages and clears their state records.
func (s *InstallService) Uninstall(ctx context.Context, ids []string) error {
	lock, err := state.AcquireLock(s.platform.StateDir(), "uninstall")
	if err != nil {
		return domain.NewExitError(domain.ExitPreflightFailed, err)
	}
	defer func() { _ = lock.Release() }()

	for _, id := range ids {
		// Strict ownership check: only uninstall resources recorded as ManagedByAnx
		if s.store != nil {
			rec, exists := s.store.GetResource(id)
			if !exists || !rec.ManagedByAnx {
				// Resource was pre-existing or not managed by ANX; protect user/system state
				continue
			}
		}

		pkg, ok := s.catalog.GetPackage(id)
		if !ok {
			pkg = domain.ManagedResource{ID: id, Name: id}
		}
		if err := s.provider.Uninstall(ctx, pkg); err != nil {
			return fmt.Errorf("uninstalling %s: %w", id, err)
		}
		if s.store != nil {
			if err := s.store.RemoveResource(id); err != nil {
				return fmt.Errorf("removing %s from state store: %w", id, err)
			}
		}
	}
	return nil
}

// Rollback restores the most recent configuration snapshot.
func (s *InstallService) Rollback(_ context.Context) (*domain.Snapshot, error) {
	if s.snapshots == nil {
		return nil, errors.New("snapshot manager not configured")
	}
	return s.snapshots.RestoreLatest()
}

// PendingTransaction returns the last unfinished journal transaction, if any.
func (s *InstallService) PendingTransaction() (*domain.Transaction, error) {
	return s.journal.GetLastPendingTx()
}

// Resume re-plans using the active profile; completed steps are skipped because Plan is idempotent.
func (s *InstallService) Resume(ctx context.Context) (*domain.ExecutionPlan, error) {
	profile := "developer"
	if s.store != nil {
		profile = s.store.ActiveProfile()
	}
	plan, err := s.Plan(ctx, profile, nil)
	if err != nil {
		return nil, err
	}
	return plan, s.Apply(ctx, plan)
}

func (s *InstallService) recordInstalled(pkg domain.ManagedResource, version string) error {
	if s.store == nil {
		return nil
	}
	return s.store.RecordResource(state.InstalledRecord{
		ResourceID:   pkg.ID,
		Version:      strings.TrimSpace(version),
		Strategy:     string(pkg.Strategy),
		Provider:     s.provider.Name(),
		ManagedByAnx: true,
	})
}

func (s *InstallService) journalAppend(e domain.JournalEntry) error {
	if s.journal != nil {
		return s.journal.Append(e)
	}
	return nil
}

func (s *InstallService) journalStep(txID, typ string, step domain.PlanStep, errMsg string) error {
	return s.journalAppend(domain.JournalEntry{
		TxID: txID, Timestamp: time.Now(), Type: typ, StepID: step.ID,
		Action: step.Action, Target: step.TargetPath, Error: errMsg,
	})
}

// PlanRequiresSudo checks if any step in plan requires root/sudo privileges on this host.
func (s *InstallService) PlanRequiresSudo(plan *domain.ExecutionPlan) bool {
	if s.platform == nil || s.platform.IsRoot() || s.platform.IsTermux() || s.platform.OS() == "windows" || !s.platform.HasSudo() {
		return false
	}
	if plan == nil || len(plan.Steps) == 0 {
		return false
	}
	for _, step := range plan.Steps {
		if step.Action == domain.ActionSync {
			if s.platform.OS() == "darwin" && s.catalog != nil {
				if cfg, ok := s.catalog.GetConfiguration(step.ResourceID); ok && cfg.Group == "osconfig" && len(cfg.PostSyncCmds) > 0 {
					return true
				}
			}
			continue
		}
		if step.Action != domain.ActionInstall && step.Action != domain.ActionUninstall {
			continue
		}
		if s.catalog != nil {
			if spec, ok := s.catalog.GetPackageSpec(step.ResourceID); ok {
				if s.packageSpecRequiresSudo(spec) {
					return true
				}
				continue
			}
		}
		pm := s.platform.PrimaryPackageManager()
		if isNativeSudoPM(pm) {
			return true
		}
	}
	return false
}

func (s *InstallService) packageSpecRequiresSudo(spec manifest.PackageSpec) bool {
	if s.platform == nil || s.platform.IsRoot() || s.platform.IsTermux() || s.platform.OS() == "windows" || !s.platform.HasSudo() {
		return false
	}
	if resolver, ok := s.provider.(StrategyDependencyResolver); ok {
		st, found := resolver.SelectStrategy(spec.ID)
		if !found {
			return false
		}
		return s.nativeStrategyRequiresSudo(st)
	}
	for _, st := range spec.Strategies {
		if s.nativeStrategyRequiresSudo(st) {
			return true
		}
	}
	return false
}

func (s *InstallService) nativeStrategyRequiresSudo(st manifest.StrategySpec) bool {
	if st.Type != string(domain.StrategyNative) {
		return false
	}
	for _, pm := range s.platform.PackageManagers() {
		if !isNativeSudoPM(pm) {
			continue
		}
		if pkgs, ok := st.Packages[pm]; ok && len(pkgs) > 0 {
			return true
		}
		if repos, ok := st.Repositories[pm]; ok && len(repos) > 0 {
			return true
		}
	}
	return false
}

func isNativeSudoPM(pm string) bool {
	switch pm {
	case "apt", "dnf", "pacman", "zypper", "apk":
		return true
	default:
		return false
	}
}
