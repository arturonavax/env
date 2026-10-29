package app

import (
	"context"
	"fmt"

	"github.com/arturonavax/env/internal/domain"
	"github.com/arturonavax/env/internal/hacks"
	"github.com/arturonavax/env/internal/platform"
	"github.com/arturonavax/env/internal/security"
	"github.com/arturonavax/env/internal/snapshot"
	"github.com/arturonavax/env/internal/state"
)

// HacksService exposes machine-specific hack discovery, planning and application.
type HacksService struct {
	platform  *platform.Info
	registry  *hacks.Registry
	runner    *security.Runner
	snapshots *snapshot.Manager
	store     *state.Store
	repoRoot  string
}

// NewHacksService wires a HacksService.
func NewHacksService(
	plat *platform.Info,
	reg *hacks.Registry,
	runner *security.Runner,
	snap *snapshot.Manager,
	store *state.Store,
	repoRoot string,
) *HacksService {
	return &HacksService{platform: plat, registry: reg, runner: runner, snapshots: snap, store: store, repoRoot: repoRoot}
}

// List returns every registered hack profile.
func (h *HacksService) List() []domain.HackProfile { return h.registry.List() }

// Applied reports whether a hack was already applied on this machine.
func (h *HacksService) Applied(id string) bool { return h.store != nil && h.store.IsHackApplied(id) }

// Detect probes hardware and returns the profile plus the hacks that match this host.
func (h *HacksService) Detect(ctx context.Context) (domain.HardwareProfile, []domain.HackProfile) {
	hw := hacks.ProbeHardware(ctx, h.platform, h.runner)
	return hw, hacks.MatchHacks(hw, h.registry.List())
}

// Plan previews a hack without mutating the system.
func (h *HacksService) Plan(id string) (*hacks.HackPlan, error) {
	p, ok := h.registry.Get(id)
	if !ok {
		return nil, fmt.Errorf("unknown hack %q", id)
	}
	return hacks.PlanHack(h.platform, p)
}

// Apply runs a hack after taking a pre-hack snapshot; it never runs implicitly.
func (h *HacksService) Apply(ctx context.Context, id string) (*hacks.HackApplyResult, error) {
	p, ok := h.registry.Get(id)
	if !ok {
		return nil, fmt.Errorf("unknown hack %q", id)
	}
	lock, err := state.AcquireLock(h.platform.StateDir(), "hack:"+id)
	if err != nil {
		return nil, domain.NewExitError(domain.ExitPreflightFailed, err)
	}
	defer func() { _ = lock.Release() }()

	return hacks.ApplyHack(ctx, p, hacks.ApplyOptions{
		Platform:  h.platform,
		Runner:    h.runner,
		Snapshots: h.snapshots,
		Store:     h.store,
		WorkDir:   h.repoRoot,
	})
}
