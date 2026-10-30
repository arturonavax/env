package app

import (
	"context"
	"io"

	"github.com/arturonavax/env/internal/ops"
	"github.com/arturonavax/env/internal/platform"
	"github.com/arturonavax/env/internal/security"
)

// OpsService wraps native maintenance operations (rebuild, optimize, and safe power transitions).
type OpsService struct {
	platform *platform.Info
	runner   *security.Runner
}

// NewOpsService wires an OpsService.
func NewOpsService(plat *platform.Info, runner *security.Runner) *OpsService {
	return &OpsService{platform: plat, runner: runner}
}

// SafeSuspend executes hardware-aware power operations (status, lock, s2idle, deep, hibernate).
func (o *OpsService) SafeSuspend(ctx context.Context, mode string, dryRun bool, out io.Writer) error {
	pm := ops.NewPowerManager(o.platform, o.runner, ops.WithPowerDryRun(dryRun), ops.WithPowerOutput(out))
	return pm.Execute(ctx, mode)
}

// PowerStatus queries hardware power configuration without executing power actions.
func (o *OpsService) PowerStatus(ctx context.Context) (*ops.PowerStatus, error) {
	pm := ops.NewPowerManager(o.platform, o.runner)
	return pm.DetectStatus(ctx)
}

// Rebuild reinstalls/updates global tools for the requested scopes.
func (o *OpsService) Rebuild(ctx context.Context, scopes []string, dryRun bool) ([]ops.RebuildResult, error) {
	return ops.NewRebuilder(o.platform, o.runner, ops.WithDryRun(dryRun)).Rebuild(ctx, scopes)
}

// OptimizeRequest controls the optimization workflow.
type OptimizeRequest struct {
	AuditOnly bool
	SkipLynis bool
	Apply     bool
	Prompt    ops.PromptFunc
}

// OptimizeResult carries the audit, generated script and execution output.
type OptimizeResult struct {
	Report *ops.SystemAuditReport
	Script string
	Output string
}

// Optimize audits the host, synthesizes a guardrail-validated script and optionally applies it.
func (o *OpsService) Optimize(ctx context.Context, req OptimizeRequest) (*OptimizeResult, error) {
	opt := ops.NewOptimizer(o.platform, o.runner, ops.WithPromptFunc(req.Prompt))
	report, err := opt.AuditSystem(ctx, req.SkipLynis)
	if err != nil {
		return nil, err
	}
	res := &OptimizeResult{Report: report}
	if req.AuditOnly {
		return res, nil
	}
	script, err := opt.SynthesizeScript(ctx, report)
	if err != nil {
		return res, err
	}
	res.Script = script
	out, err := opt.ExecuteScript(ctx, script, !req.Apply)
	res.Output = out
	return res, err
}
