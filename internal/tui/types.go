package tui

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/arturonavax/env/internal/app"
	"github.com/arturonavax/env/internal/catalog"
	"github.com/arturonavax/env/internal/domain"
	"github.com/arturonavax/env/internal/platform"
	"github.com/arturonavax/env/internal/security"
	"github.com/arturonavax/env/internal/state"
	"github.com/arturonavax/env/internal/theme"
)

// TabID represents an active view screen in the TUI dashboard.
type TabID int

const (
	TabOverview TabID = iota
	TabPackages
	TabConfigs
	TabAI
	TabHacks
	TabDoctor
	TabHistory
	TabSecurity
)

// TabNames provides human-readable labels for the dashboard tabs.
var TabNames = []string{
	"1:Overview",
	"2:Packages",
	"3:Configs",
	"4:AI",
	"5:Hacks",
	"6:Doctor",
	"7:History",
	"8:Security",
}

// Deps encapsulates external services and runtime dependencies passed to the TUI.
type Deps struct {
	Platform *platform.Info
	Catalog  *catalog.Registry
	Query    app.QueryService
	Doctor   *app.DoctorService
	Audit    *app.AuditService
	Hacks    *app.HacksService
	Sync     *app.SyncService
	Store    *state.Store
	Theme    *theme.Palette
	Styles   *theme.Styles
	RepoRoot string
}

// UpdateCheckResult holds the status of background update checks across repo and dotfiles.
type UpdateCheckResult struct {
	RepoBehind   int       `json:"repo_behind"`
	RepoBranch   string    `json:"repo_branch"`
	RepoClean    bool      `json:"repo_clean"`
	ConfigDrifts int       `json:"config_drifts"`
	HasUpdates   bool      `json:"has_updates"`
	CheckedAt    time.Time `json:"checked_at"`
}

// Async message types for loading data without freezing the UI.
type (
	doctorLoadedMsg      []app.DoctorCheck
	auditLoadedMsg       []domain.AuditFinding
	systemAuditLoadedMsg *security.SecurityAuditReport
	hacksLoadedMsg       []domain.HackProfile
	updateCheckLoadedMsg struct {
		result UpdateCheckResult
		err    error
	}
	statusUpdateMsg string
	errMsg          struct{ err error }
)

// loadUpdateCheckCmd asynchronously inspects repo commits behind and configuration drifts.
func loadUpdateCheckCmd(ctx context.Context, repoRoot string, syncSvc *app.SyncService) tea.Cmd {
	return func() tea.Msg {
		res := UpdateCheckResult{
			RepoClean: true,
			CheckedAt: time.Now(),
		}
		if repoRoot != "" {
			if out, err := exec.CommandContext(ctx, "git", "-C", repoRoot, "rev-parse", "--abbrev-ref", "HEAD").Output(); err == nil {
				res.RepoBranch = strings.TrimSpace(string(out))
			}
			if out, err := exec.CommandContext(ctx, "git", "-C", repoRoot, "status", "--porcelain").Output(); err == nil {
				res.RepoClean = len(bytes.TrimSpace(out)) == 0
			}
			if out, err := exec.CommandContext(ctx, "git", "-C", repoRoot, "rev-list", "--count", "HEAD..@{u}").Output(); err == nil {
				var count int
				if _, err := fmt.Sscanf(string(out), "%d", &count); err == nil {
					res.RepoBehind = count
				}
			}
		}
		if syncSvc != nil {
			if diffs, err := syncSvc.Diff(ctx, "all"); err == nil {
				for _, d := range diffs {
					if d.State != "clean" && d.State != "synced" {
						res.ConfigDrifts++
					}
				}
			}
		}
		res.HasUpdates = (res.RepoBehind > 0 || res.ConfigDrifts > 0)
		return updateCheckLoadedMsg{result: res}
	}
}

// loadDoctorCmd returns a tea.Cmd fetching doctor health checks asynchronously.
func loadDoctorCmd(ctx context.Context, svc *app.DoctorService) tea.Cmd {
	return func() tea.Msg {
		if svc == nil {
			return doctorLoadedMsg(nil)
		}
		checks, err := svc.Diagnose(ctx)
		if err != nil {
			return errMsg{err: err}
		}
		return doctorLoadedMsg(checks)
	}
}

// loadAuditCmd returns a tea.Cmd running OSV vulnerability scanning asynchronously.
func loadAuditCmd(ctx context.Context, svc *app.AuditService) tea.Cmd {
	return func() tea.Msg {
		if svc == nil {
			return auditLoadedMsg(nil)
		}
		findings, err := svc.Audit(ctx)
		if err != nil {
			return errMsg{err: err}
		}
		return auditLoadedMsg(findings)
	}
}

// loadSystemAuditCmd returns a tea.Cmd executing full system security auditing asynchronously.
func loadSystemAuditCmd(ctx context.Context, svc *app.AuditService, profile string) tea.Cmd {
	return func() tea.Msg {
		if svc == nil {
			return systemAuditLoadedMsg(nil)
		}
		res, err := svc.SystemAudit(ctx, app.SecurityAuditRequest{Profile: profile}, true)
		if err != nil {
			return errMsg{err: err}
		}
		if res != nil {
			return systemAuditLoadedMsg(res.Report)
		}
		return systemAuditLoadedMsg(nil)
	}
}

// loadHacksCmd returns a tea.Cmd detecting hardware/system hacks asynchronously.
func loadHacksCmd(ctx context.Context, svc *app.HacksService) tea.Cmd {
	return func() tea.Msg {
		if svc == nil {
			return hacksLoadedMsg(nil)
		}
		_, detected := svc.Detect(ctx)
		return hacksLoadedMsg(detected)
	}
}
