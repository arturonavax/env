package scriptgen

import (
	"fmt"
	"strings"

	"github.com/arturonavax/env/internal/ops"
	"github.com/arturonavax/env/internal/platform"
	"github.com/arturonavax/env/internal/security"
)

// TranslateSync converts managed dotfile sync candidates into an auditable script.
func TranslateSync(group string, configs []string, plat *platform.Info) string {
	b := NewBuilder("sync", plat)
	b.AddSection(fmt.Sprintf("Synchronize Dotfile Group [%s]", group), "Reconcile canonical configuration assets")

	for _, cfg := range configs {
		cmd := fmt.Sprintf("anx sync --group %q", group)
		b.AddCommand(
			cmd,
			fmt.Sprintf("Asset: %s (3-way merge and template evaluation handled by ANX control plane)", cfg),
		)
	}
	return b.Render()
}

// TranslateCleanup converts orphaned files and caches into an auditable script.
func TranslateCleanup(items []string, plat *platform.Info) string {
	b := NewBuilder("cleanup", plat)
	b.AddSection("Prune Orphaned Artifacts and Caches", "Remove unmanaged dotfiles, stale sockets, and deprecated caches")

	for _, it := range items {
		b.AddCommand(fmt.Sprintf("rm -rf %q", it), "Orphan / deprecated item")
	}
	return b.Render()
}

// TranslateRebuild converts language runtime rebuild results into an auditable script.
func TranslateRebuild(results []ops.RebuildResult, plat *platform.Info) string {
	b := NewBuilder("rebuild", plat)

	for _, r := range results {
		b.AddSection(fmt.Sprintf("Toolchain Rebuild: %s", strings.ToUpper(r.Scope)), fmt.Sprintf("Commands planned: %d", len(r.Commands)))
		for _, cmd := range r.Commands {
			b.AddCommand(cmd, fmt.Sprintf("Scope: %s", r.Scope))
		}
	}
	return b.Render()
}

// TranslateCapture converts captured dotfiles and git branch/commit actions into an auditable script.
func TranslateCapture(paths []string, branch, commitMsg string, plat *platform.Info) string {
	b := NewBuilder("capture", plat)
	b.AddSection("Capture Dotfiles to Working Tree", fmt.Sprintf("Staging %d assets", len(paths)))

	for _, p := range paths {
		b.AddCommand(fmt.Sprintf("git add -- %q", p), "Stage captured asset exclusively")
	}

	if branch != "" {
		b.AddSection("Branch Creation", fmt.Sprintf("Create capture branch %s", branch))
		b.AddCommand(fmt.Sprintf("git checkout -b %q", branch), "Isolate changes in dedicated branch")
	}

	if commitMsg != "" {
		b.AddSection("Commit Captured Changes", "Create atomic conventional commit")
		b.AddCommand(fmt.Sprintf("git commit -m %q", commitMsg), "Ownership-guarded commit")
	}

	return b.Render()
}

// TranslateSecurityAudit converts security findings into an auditable remediation script.
func TranslateSecurityAudit(report *security.SecurityAuditReport, plat *platform.Info) string {
	b := NewBuilder("audit", plat)
	if report == nil {
		return b.Render()
	}

	b.AddSection(
		fmt.Sprintf("Security Hardening Remediation (Profile: %s, Score: %d/100)", report.Profile, report.HardeningScore),
		"Apply CIS/NIST and platform-specific remediations for identified vulnerabilities",
	)

	applied := make(map[string]bool)
	for _, f := range report.Findings {
		if f.RemediationCommand == "" || applied[f.RemediationCommand] {
			continue
		}
		applied[f.RemediationCommand] = true
		b.AddCommand(f.RemediationCommand, fmt.Sprintf("[%s] %s (%s)", f.Severity, f.Title, f.ID))
	}

	return b.Render()
}
