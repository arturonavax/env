package app

import (
	"context"

	"github.com/arturonavax/env/internal/catalog"
	"github.com/arturonavax/env/internal/domain"
	"github.com/arturonavax/env/internal/providers"
	"github.com/arturonavax/env/internal/security"
)

// SecurityAuditRequest configures a full system security audit run.
type SecurityAuditRequest struct {
	Profile    string
	UseAI      bool
	Apply      bool
	PromptFunc security.SecurityPromptFunc
}

// SecurityAuditResult encapsulates the audit report, generated script, and execution output.
type SecurityAuditResult struct {
	Report *security.SecurityAuditReport `json:"report"`
	Script string                        `json:"script,omitempty"`
	Output string                        `json:"output,omitempty"`
}

// AuditService coordinates vulnerability scanning across managed tools and system security audits.
type AuditService struct {
	catalog   *catalog.Registry
	provider  providers.Provider
	osvClient *security.OSVClient
}

// NewAuditService constructs an AuditService.
func NewAuditService(
	cat *catalog.Registry,
	prov providers.Provider,
	osv *security.OSVClient,
) *AuditService {
	return &AuditService{
		catalog:   cat,
		provider:  prov,
		osvClient: osv,
	}
}

// SystemAudit runs a full system security audit with profile rules, script generation, and optional execution.
func (a *AuditService) SystemAudit(ctx context.Context, req SecurityAuditRequest, dryRun bool) (*SecurityAuditResult, error) {
	opts := []security.SecurityAuditorOption{
		security.WithAuditorProfile(req.Profile),
	}
	if req.UseAI && req.PromptFunc != nil {
		opts = append(opts, security.WithAuditorPromptFunc(req.PromptFunc))
	}

	auditor := security.NewSecurityAuditor(opts...)
	report, err := auditor.Audit(ctx)
	if err != nil {
		return nil, err
	}

	res := &SecurityAuditResult{
		Report: report,
	}

	script, err := auditor.SynthesizeScript(ctx, report)
	if err != nil {
		return res, err
	}
	res.Script = script

	if req.Apply {
		out, err := auditor.ExecuteScript(ctx, script, dryRun)
		res.Output = out
		if err != nil {
			return res, err
		}
	}

	return res, nil
}

// Audit scans all installed managed tools against OSV vulnerabilities.
func (a *AuditService) Audit(ctx context.Context) ([]domain.AuditFinding, error) {
	pkgs := a.catalog.ListPackages()
	allFindings := make([]domain.AuditFinding, 0, 8)

	for _, p := range pkgs {
		installed, ver, err := a.provider.IsInstalled(ctx, p)
		if err != nil || !installed || ver == "" {
			continue
		}

		findings, err := a.osvClient.Query(ctx, p.ID, ver)
		if err != nil {
			continue
		}

		allFindings = append(allFindings, findings...)
	}

	return allFindings, nil
}
