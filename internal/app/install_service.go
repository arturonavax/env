package app

import (
	"context"
	"errors"
	"fmt"
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
	return &InstallService{
		platform:   deps.Platform,
		catalog:    deps.Catalog,
		journal:    deps.Journal,
		snapshots:  deps.Snapshots,
		provider:   deps.Provider,
		store:      deps.Store,
		sync:       deps.Sync,
		engine:     execution.NewEngine(4),
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

	ordered, err := s.catalog.ResolveInstallOrder(ids)
	if err != nil {
		return nil, err
	}

	stepID := 1
	installStepIDs := make(map[string]int, len(ordered))
	for _, pkg := range ordered {
		if len(pkg.SupportedOS) > 0 && !containsOS(pkg.SupportedOS, s.platform.OS()) {
			continue
		}
		if !force {
			installed, _, _ := s.provider.IsInstalled(ctx, pkg)
			if installed {
				continue
			}
		}
		plan.Steps = append(plan.Steps, domain.PlanStep{
			ID:          stepID,
			ResourceID:  pkg.ID,
			Action:      domain.ActionInstall,
			Description: fmt.Sprintf("Install %s (%s)", pkg.Name, pkg.Description),
			Provider:    s.provider.Name(),
			Strategy:    string(pkg.Strategy),
			DependsOn:   installDependencies(pkg, installStepIDs),
			IsExclusive: true,
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
		for _, cfg := range cfgs {
			if !cfg.SupportsOS(s.platform.OS()) {
				continue
			}
			target := s.platform.ExpandPath(cfg.TargetForOS(s.platform.OS()))
			plan.Steps = append(plan.Steps, domain.PlanStep{
				ID:          stepID,
				ResourceID:  cfg.ID,
				Action:      domain.ActionSync,
				Description: fmt.Sprintf("Synchronize %s", cfg.Name),
				SourcePath:  cfg.Source,
				TargetPath:  target,
				// Serialize configs that write the same path while letting unrelated ones overlap.
				Locks:  []string{"file:" + target},
				Status: domain.StepPending,
			})
			stepID++
		}
	}

	return plan, nil
}

// installDependencies maps a package's declared dependencies to the plan step IDs that install them.
// Dependencies absent from stepIDs are already satisfied (installed or filtered out) and need no edge.
func installDependencies(pkg domain.ManagedResource, stepIDs map[string]int) []int {
	var deps []int
	for _, dep := range pkg.DependsOn {
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

	tasks := make([]execution.Task, len(plan.Steps))
	for i := range plan.Steps {
		tasks[i] = s.buildTask(txID, plan.Steps[i])
	}

	byID := make(map[int]*domain.PlanStep, len(plan.Steps))
	for i := range plan.Steps {
		byID[plan.Steps[i].ID] = &plan.Steps[i]
	}

	_, execErr := s.engine.ExecuteTasks(ctx, tasks, func(res execution.Result) {
		step := byID[res.StepID]
		if step == nil {
			return
		}
		step.Status = res.Status
		if res.Error != nil {
			step.Error = res.Error.Error()
		}
		if progressFn := s.getOnProgress(); progressFn != nil {
			progressFn(ProgressEvent{
				StepID: step.ID, TotalSteps: len(plan.Steps), ResourceID: step.ResourceID,
				Action: step.Action, Description: step.Description, Status: res.Status, Error: step.Error,
			})
		}
	})

	if execErr != nil {
		_ = s.journalAppend(domain.JournalEntry{
			TxID: txID, Timestamp: time.Now(), Type: "rollback", Details: execErr.Error(),
		})
		if errors.Is(execErr, execution.ErrExecutionInterrupted) {
			return domain.NewExitError(domain.ExitInterrupted, execErr)
		}
		return domain.NewExitError(domain.ExitRollbackOrPartial, execErr)
	}

	if err := s.journalAppend(domain.JournalEntry{
		TxID: txID, Timestamp: time.Now(), Type: "commit", Details: "all steps succeeded",
	}); err != nil {
		return domain.NewExitError(domain.ExitRollbackOrPartial, fmt.Errorf("journal commit failed: %w", err))
	}
	return nil
}

func (s *InstallService) buildTask(txID string, step domain.PlanStep) execution.Task {
	task := execution.Task{Step: step, Exclusive: step.IsExclusive}

	switch step.Action {
	case domain.ActionInstall:
		pkg, ok := s.catalog.GetPackage(step.ResourceID)
		if !ok {
			pkg = domain.ManagedResource{ID: step.ResourceID, Name: step.ResourceID}
		}
		task.Timeout = 30 * time.Minute
		task.Execute = func(ctx context.Context) error {
			if err := s.journalStep(txID, "step_start", step, ""); err != nil {
				return fmt.Errorf("journal step_start: %w", err)
			}
			if err := s.provider.Install(ctx, pkg); err != nil {
				_ = s.journalStep(txID, "step_failed", step, err.Error())
				return err
			}
			_, ver, _ := s.provider.IsInstalled(ctx, pkg)
			if err := s.recordInstalled(pkg, ver); err != nil {
				_ = s.journalStep(txID, "step_failed", step, err.Error())
				return fmt.Errorf("recording installed %s: %w", pkg.ID, err)
			}
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
			if err := s.journalStep(txID, "step_start", step, ""); err != nil {
				return fmt.Errorf("journal step_start: %w", err)
			}
			if s.sync == nil {
				return errors.New("sync service not configured")
			}
			res, err := s.sync.SyncOne(ctx, step.ResourceID, SyncOptions{})
			if err != nil {
				_ = s.journalStep(txID, "step_failed", step, err.Error())
				return err
			}
			snapID = res.SnapshotID
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

func containsOS(list []string, goos string) bool {
	for _, v := range list {
		if v == goos || v == "all" {
			return true
		}
	}
	return false
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
	for _, st := range spec.Strategies {
		if st.Type == string(domain.StrategyNative) {
			for _, pm := range s.platform.PackageManagers() {
				if isNativeSudoPM(pm) {
					if pkgs, ok := st.Packages[pm]; ok && len(pkgs) > 0 {
						return true
					}
					if repos, ok := st.Repositories[pm]; ok && len(repos) > 0 {
						return true
					}
				}
			}
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
