package app_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/arturonavax/env/internal/app"
	"github.com/arturonavax/env/internal/catalog"
	"github.com/arturonavax/env/internal/domain"
	"github.com/arturonavax/env/internal/platform"
)

func TestDoctorService_Diagnose(t *testing.T) {
	plat := platform.Detect()
	cat := catalog.NewDefaultRegistry()

	tmpDir := t.TempDir()
	manifestsDir := filepath.Join(tmpDir, "manifests")
	if err := os.MkdirAll(manifestsDir, 0755); err != nil {
		t.Fatalf("failed to create manifests dir: %v", err)
	}

	docSvc := app.NewDoctorService(plat, cat, app.WithDoctorRepoRoot(tmpDir))

	checks, err := docSvc.Diagnose(context.Background())
	if err != nil {
		t.Fatalf("Diagnose returned unexpected error: %v", err)
	}

	if len(checks) == 0 {
		t.Fatal("expected at least one doctor check evaluation")
	}

	var foundPlatform, foundRepo, foundPM bool
	for _, c := range checks {
		switch c.Name {
		case "Platform Detection":
			foundPlatform = true
			if c.Verdict != app.VerdictPass {
				t.Errorf("expected Platform Detection PASS, got %s", c.Verdict)
			}
		case "Repository Root":
			foundRepo = true
			if c.Verdict != app.VerdictPass {
				t.Errorf("expected Repository Root PASS with valid manifests dir, got %s", c.Verdict)
			}
		case "Package Manager":
			foundPM = true
		}
	}

	if !foundPlatform {
		t.Error("Platform Detection check was not emitted")
	}
	if !foundRepo {
		t.Error("Repository Root check was not emitted")
	}
	if !foundPM {
		t.Error("Package Manager check was not emitted")
	}

	status := docSvc.HealthStatus(checks)
	if status != domain.HealthHealthy && status != domain.HealthWarning {
		t.Errorf("expected HealthHealthy or HealthWarning status, got %s", status)
	}
}

func TestDoctorService_NoRepoRoot(t *testing.T) {
	plat := platform.Detect()
	cat := catalog.NewDefaultRegistry()

	docSvc := app.NewDoctorService(plat, cat, app.WithDoctorRepoRoot(""))

	checks, err := docSvc.Diagnose(context.Background())
	if err != nil {
		t.Fatalf("Diagnose returned unexpected error: %v", err)
	}

	var repoCheck *app.DoctorCheck
	for i := range checks {
		if checks[i].Name == "Repository Root" {
			repoCheck = &checks[i]
			break
		}
	}

	if repoCheck == nil {
		t.Fatal("expected Repository Root check to be emitted")
	}
	if repoCheck.Verdict != app.VerdictWarn {
		t.Errorf("expected Repository Root WARN when unconfigured, got %s", repoCheck.Verdict)
	}
	if repoCheck.Remediation == "" {
		t.Error("expected non-empty remediation advice for missing repository root")
	}
}
