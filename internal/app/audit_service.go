package app

import (
	"context"

	"github.com/arturonavax/env/internal/catalog"
	"github.com/arturonavax/env/internal/domain"
	"github.com/arturonavax/env/internal/providers"
	"github.com/arturonavax/env/internal/security"
)

// AuditService coordinates vulnerability scanning across managed tools.
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
