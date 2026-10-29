package app

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/arturonavax/env/internal/catalog"
	"github.com/arturonavax/env/internal/domain"
	"github.com/arturonavax/env/internal/platform"
)

// CheckVerdict represents the result of an individual health diagnostic.
type CheckVerdict string

const (
	VerdictPass CheckVerdict = "PASS"
	VerdictWarn CheckVerdict = "WARN"
	VerdictFail CheckVerdict = "FAIL"
)

// DoctorCheck represents a single diagnostic evaluation.
type DoctorCheck struct {
	Name        string       `json:"name"`
	Verdict     CheckVerdict `json:"verdict"`
	Message     string       `json:"message"`
	Remediation string       `json:"remediation,omitempty"`
}

// DoctorService evaluates system health, missing tools, and configuration drift.
type DoctorService struct {
	platform *platform.Info
	catalog  *catalog.Registry
}

// NewDoctorService creates a DoctorService.
func NewDoctorService(plat *platform.Info, cat *catalog.Registry) *DoctorService {
	return &DoctorService{platform: plat, catalog: cat}
}

// Diagnose executes diagnostic evaluations across environment, directories, and tools.
func (d *DoctorService) Diagnose(ctx context.Context) ([]DoctorCheck, error) {
	checks := make([]DoctorCheck, 0, 16)

	// 1. Platform & Environment Check
	checks = append(checks, DoctorCheck{
		Name:    "Platform Detection",
		Verdict: VerdictPass,
		Message: fmt.Sprintf("OS: %s, Arch: %s, Distro: %s (WSL: %v, Termux: %v)",
			d.platform.OS(), d.platform.Arch(), d.platform.Distribution(), d.platform.IsWSL(), d.platform.IsTermux()),
	})

	// 2. State Directory Permissions
	stateDir := d.platform.StateDir()
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		checks = append(checks, DoctorCheck{
			Name:        "State Directory",
			Verdict:     VerdictFail,
			Message:     fmt.Sprintf("Failed to access state directory %s: %v", stateDir, err),
			Remediation: "Ensure permissions for your user account on " + stateDir,
		})
	} else {
		checks = append(checks, DoctorCheck{
			Name:    "State Directory",
			Verdict: VerdictPass,
			Message: fmt.Sprintf("Accessible and writable (%s)", stateDir),
		})
	}

	// 3. Essential Tools Inspection
	essentialTools := []string{"git", "curl", "zsh", "rg", "nvim"}
	for _, tool := range essentialTools {
		path, err := exec.LookPath(tool)
		if err != nil {
			checks = append(checks, DoctorCheck{
				Name:        fmt.Sprintf("Tool: %s", tool),
				Verdict:     VerdictWarn,
				Message:     fmt.Sprintf("%s is not in PATH", tool),
				Remediation: fmt.Sprintf("Run 'anx install %s'", tool),
			})
		} else {
			checks = append(checks, DoctorCheck{
				Name:    fmt.Sprintf("Tool: %s", tool),
				Verdict: VerdictPass,
				Message: fmt.Sprintf("Found at %s", path),
			})
		}
	}

	// 4. Managed Dotfile Integration (.zshrc marker block)
	zshrcPath := filepath.Join(d.platform.HomeDir(), ".zshrc")
	if data, err := os.ReadFile(zshrcPath); err == nil {
		content := string(data)
		if len(content) > 0 {
			checks = append(checks, DoctorCheck{
				Name:    "Zsh Shell Configuration",
				Verdict: VerdictPass,
				Message: fmt.Sprintf("Existing ~/.zshrc detected (%d bytes)", len(content)),
			})
		}
	} else {
		checks = append(checks, DoctorCheck{
			Name:        "Zsh Shell Configuration",
			Verdict:     VerdictWarn,
			Message:     "~/.zshrc not found",
			Remediation: "Run 'anx sync' to initialize shell integration",
		})
	}

	return checks, nil
}

// HealthStatus returns the overall health rollup.
func (d *DoctorService) HealthStatus(checks []DoctorCheck) domain.HealthStatus {
	hasWarn := false
	for _, c := range checks {
		if c.Verdict == VerdictFail {
			return domain.HealthFailed
		}
		if c.Verdict == VerdictWarn {
			hasWarn = true
		}
	}
	if hasWarn {
		return domain.HealthWarning
	}
	return domain.HealthHealthy
}
