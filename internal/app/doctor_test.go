package app_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/arturonavax/env/internal/app"
	"github.com/arturonavax/env/internal/catalog"
	"github.com/arturonavax/env/internal/domain"
	"github.com/arturonavax/env/internal/platform"
	"github.com/arturonavax/env/internal/security"
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

type fixedTrustVerifier struct {
	report security.TrustReport
}

func (f fixedTrustVerifier) VerifyDirectory(path string) security.TrustReport {
	r := f.report
	r.Path = path
	return r
}

func TestDoctorService_DiagnoseIsReadOnly(t *testing.T) {
	plat := platform.Detect()
	cat := catalog.NewDefaultRegistry()

	tmpDir := t.TempDir()
	manifestsDir := filepath.Join(tmpDir, "manifests")
	if err := os.MkdirAll(manifestsDir, 0755); err != nil {
		t.Fatalf("failed to create manifests dir: %v", err)
	}

	stateDir := plat.StateDir()
	_, stateErrBefore := os.Stat(stateDir)

	docSvc := app.NewDoctorService(plat, cat,
		app.WithDoctorRepoRoot(tmpDir),
		app.WithDoctorTrustVerifier(fixedTrustVerifier{
			report: security.TrustReport{Level: security.TrustTrusted},
		}),
	)

	_, err := docSvc.Diagnose(context.Background())
	if err != nil {
		t.Fatalf("Diagnose returned unexpected error: %v", err)
	}

	_, stateErrAfter := os.Stat(stateDir)
	if os.IsNotExist(stateErrBefore) && !os.IsNotExist(stateErrAfter) {
		t.Errorf("Diagnose must be strictly read-only and must not create stateDir (%s)", stateDir)
	}
}

func TestDoctorService_StateDirectoryVerdictFollowsTrust(t *testing.T) {
	plat := platform.Detect()
	cat := catalog.NewDefaultRegistry()

	cases := []struct {
		name        string
		setupState  func(t *testing.T)
		report      security.TrustReport
		wantVerdict app.CheckVerdict
	}{
		{
			name: "trusted directory passes",
			setupState: func(t *testing.T) {
				_ = os.MkdirAll(plat.StateDir(), 0700)
			},
			report:      security.TrustReport{Level: security.TrustTrusted},
			wantVerdict: app.VerdictPass,
		},
		{
			name: "degraded directory warns",
			setupState: func(t *testing.T) {
				_ = os.MkdirAll(plat.StateDir(), 0700)
			},
			report: security.TrustReport{
				Level:   security.TrustDegraded,
				Reasons: []string{"world-writable"},
			},
			wantVerdict: app.VerdictWarn,
		},
		{
			name: "untrusted directory fails",
			setupState: func(t *testing.T) {
				_ = os.MkdirAll(plat.StateDir(), 0700)
			},
			report: security.TrustReport{
				Level:   security.TrustUntrusted,
				Reasons: []string{"owned by someone else"},
			},
			wantVerdict: app.VerdictFail,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tc.setupState(t)
			docSvc := app.NewDoctorService(plat, cat,
				app.WithDoctorTrustVerifier(fixedTrustVerifier{report: tc.report}),
			)

			checks, err := docSvc.Diagnose(context.Background())
			if err != nil {
				t.Fatalf("Diagnose failed: %v", err)
			}

			var stateCheck *app.DoctorCheck
			for i := range checks {
				if checks[i].Name == "State Directory" {
					stateCheck = &checks[i]
					break
				}
			}
			if stateCheck == nil {
				t.Fatal("missing State Directory check")
			}
			if stateCheck.Verdict != tc.wantVerdict {
				t.Errorf("got verdict %s, want %s (message: %s)", stateCheck.Verdict, tc.wantVerdict, stateCheck.Message)
			}
		})
	}
}

func TestDoctorService_RemediationNeverSuggestsChmodOnWindows(t *testing.T) {
	plat := platform.Detect()
	cat := catalog.NewDefaultRegistry()

	untrustedVerifier := fixedTrustVerifier{
		report: security.TrustReport{
			Level:   security.TrustUntrusted,
			Reasons: []string{"insecure ACL"},
		},
	}

	docSvc := app.NewDoctorService(plat, cat,
		app.WithDoctorTrustVerifier(untrustedVerifier),
	)

	checks, err := docSvc.Diagnose(context.Background())
	if err != nil {
		t.Fatalf("Diagnose failed: %v", err)
	}

	for _, c := range checks {
		if c.Remediation == "" {
			continue
		}
		if plat.OS() == "windows" {
			if strings.Contains(c.Remediation, "chmod") {
				t.Errorf("check %q suggested 'chmod' on Windows: %s", c.Name, c.Remediation)
			}
			if strings.Contains(c.Name, "State") && !strings.Contains(c.Remediation, "icacls") {
				t.Errorf("check %q should suggest icacls on Windows: %s", c.Name, c.Remediation)
			}
		}
	}
}
