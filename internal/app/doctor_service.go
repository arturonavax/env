package app

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/arturonavax/env/internal/catalog"
	"github.com/arturonavax/env/internal/domain"
	"github.com/arturonavax/env/internal/platform"
	"github.com/arturonavax/env/internal/state"
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

// DoctorOption configures optional DoctorService dependencies.
type DoctorOption func(*DoctorService)

// WithDoctorStore injects state.Store for orphaned resource detection.
func WithDoctorStore(store *state.Store) DoctorOption {
	return func(d *DoctorService) {
		d.store = store
	}
}

// DoctorService evaluates system health, missing tools, and configuration drift.
type DoctorService struct {
	platform *platform.Info
	catalog  *catalog.Registry
	store    *state.Store
}

// NewDoctorService creates a DoctorService.
func NewDoctorService(plat *platform.Info, cat *catalog.Registry, opts ...DoctorOption) *DoctorService {
	d := &DoctorService{platform: plat, catalog: cat}
	for _, opt := range opts {
		if opt != nil {
			opt(d)
		}
	}
	return d
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

	// 5. Orphaned Components Detection (state.json vs active catalog)
	if d.store != nil && d.catalog != nil {
		allPkgs := d.catalog.ListPackages()
		desiredIDs := make([]string, 0, len(allPkgs))
		for _, p := range allPkgs {
			desiredIDs = append(desiredIDs, p.ID)
		}
		for _, c := range d.catalog.ListConfigurations() {
			desiredIDs = append(desiredIDs, c.ID)
		}
		orphans := d.store.DetectOrphaned(desiredIDs)
		if len(orphans) > 0 {
			orphanNames := make([]string, 0, len(orphans))
			for _, o := range orphans {
				orphanNames = append(orphanNames, o.ResourceID)
			}
			checks = append(checks, DoctorCheck{
				Name:        "Orphaned Components",
				Verdict:     VerdictWarn,
				Message:     fmt.Sprintf("%d installed resource(s) are no longer declared in manifests or active profile: %s", len(orphans), strings.Join(orphanNames, ", ")),
				Remediation: "Run 'anx cleanup' (or 'anx prune') to safely uninstall orphaned packages",
			})
		}
	}

	// 6. Deprecated / Purge-Recommended Components
	if d.catalog != nil {
		for _, pkg := range d.catalog.ListPackages() {
			if pkg.Deprecated && pkg.RecommendPurge {
				checks = append(checks, DoctorCheck{
					Name:        fmt.Sprintf("Deprecated Package: %s", pkg.ID),
					Verdict:     VerdictWarn,
					Message:     fmt.Sprintf("%s is deprecated: %s", pkg.Name, pkg.DeprecatedReason),
					Remediation: fmt.Sprintf("Run 'anx cleanup' to purge deprecated components (recommended replacement: %s)", pkg.Replacement),
				})
			}
		}
	}

	// 7. Archived Component Legacy Remnants on Host Filesystem
	if d.catalog != nil {
		home := d.platform.HomeDir()
		for _, arc := range d.catalog.ListArchives() {
			for _, cleanPath := range arc.CleanupPaths {
				expanded := cleanPath
				if strings.HasPrefix(expanded, "~/") {
					expanded = filepath.Join(home, expanded[2:])
				}
				if info, err := os.Stat(expanded); err == nil {
					fileType := "file"
					if info.IsDir() {
						fileType = "directory"
					}
					checks = append(checks, DoctorCheck{
						Name:        fmt.Sprintf("Legacy Archive Remnant: %s", arc.ID),
						Verdict:     VerdictWarn,
						Message:     fmt.Sprintf("Found %s %s from decommissioned component %s (%s)", fileType, expanded, arc.Name, arc.Reason),
						Remediation: fmt.Sprintf("Run 'anx cleanup' to safely purge decommissioned remnants (replaced by %s)", arc.Replacement),
					})
				}
			}
		}
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
