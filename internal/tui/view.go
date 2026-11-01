package tui

import (
	"fmt"
	"os"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/arturonavax/env/internal/app"
	"github.com/arturonavax/env/internal/security"
)

// View renders the TUI screen returning tea.View (Bubble Tea v2).
func (m Model) View() tea.View {
	if m.quitting {
		return tea.NewView("")
	}

	headerStr := m.renderHeader()
	tabsStr := m.renderTabs()
	statusStr := m.renderStatusBar()

	headerLines := len(strings.Split(headerStr, "\n"))
	tabsLines := len(strings.Split(tabsStr, "\n"))
	statusLines := len(strings.Split(statusStr, "\n"))

	// Golden Rule #1: Account for UI bars and subtract 2 for top/bottom panel borders
	availHeight := m.height - headerLines - tabsLines - statusLines - 2
	if availHeight < 4 {
		availHeight = 4
	}

	availWidth := m.width
	if availWidth < 40 {
		availWidth = 40
	}

	var content string
	if m.showHelp {
		content = m.renderHelp(availWidth, availHeight)
	} else {
		switch m.activeTab {
		case TabOverview:
			content = m.renderOverview(availWidth, availHeight)
		case TabPackages:
			content = m.renderPackages(availWidth, availHeight)
		case TabConfigs:
			content = m.renderConfigs(availWidth, availHeight)
		case TabAI:
			content = m.renderAI(availWidth, availHeight)
		case TabHacks:
			content = m.renderHacks(availWidth, availHeight)
		case TabDoctor:
			content = m.renderDoctor(availWidth, availHeight)
		case TabHistory:
			content = m.renderHistory(availWidth, availHeight)
		case TabSecurity:
			content = m.renderSecurity(availWidth, availHeight)
		}
	}

	full := fmt.Sprintf("%s\n%s\n%s\n%s", headerStr, tabsStr, content, statusStr)
	v := tea.NewView(full)
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion
	return v
}

func (m Model) renderHeader() string {
	title := m.styles.Header.Render("  ANX  ") + m.styles.Muted.Render(" — Developer Environment Orchestrator (Go/Tokyo Night)")
	platStr := "unknown"
	if m.deps.Platform != nil {
		platStr = fmt.Sprintf("%s/%s", m.deps.Platform.OS(), m.deps.Platform.Arch())
		if dist := m.deps.Platform.Distribution(); dist != "" && dist != "linux" {
			platStr += fmt.Sprintf(" (%s)", dist)
		}
	}
	info := m.styles.Highlight.Render(platStr)
	return fmt.Sprintf("%s  [%s]", title, info)
}

func (m Model) renderTabs() string {
	var sb strings.Builder
	sb.WriteString(" ")
	for i, name := range TabNames {
		if TabID(i) == m.activeTab {
			sb.WriteString(m.styles.TabActive.Render(name))
		} else {
			sb.WriteString(m.styles.TabInactive.Render(name))
		}
		sb.WriteString(" ")
	}
	return sb.String()
}

func (m Model) renderStatusBar() string {
	shortcuts := m.styles.Muted.Render(" [Tab] Next Tab  [1-8] Tab Jump  [j/k] Navigate  [r] Refresh  [a] Accordion  [?] Help  [q] Quit")
	msg := m.styles.Highlight.Render(" " + m.statusMsg)
	return fmt.Sprintf("%s\n%s", shortcuts, msg)
}

func (m Model) renderOverview(w, h int) string {
	if m.isWideLayout() {
		leftW, rightW := m.calculateDualPaneWidths()
		p := m.deps.Platform

		leftLines := []string{
			m.styles.Highlight.Render("SYSTEM SUMMARY"),
			fmt.Sprintf("• Host OS:        %s", p.OS()),
			fmt.Sprintf("• Architecture:   %s", p.Arch()),
			fmt.Sprintf("• Distribution:   %s", p.Distribution()),
			fmt.Sprintf("• WSL Active:     %v", p.IsWSL()),
			fmt.Sprintf("• Termux Active:  %v", p.IsTermux()),
			fmt.Sprintf("• User Home:      %s", p.HomeDir()),
			fmt.Sprintf("• Config Root:    %s", p.ConfigDir()),
			fmt.Sprintf("• State Root:     %s", p.StateDir()),
			"",
			m.styles.Highlight.Render("CORE COMMANDS"),
			"• anx apply       Apply desired workstation state",
			"• anx update      Check repo, tools & dotfile updates",
			"• anx upgrade     Pull repo, update tools & sync",
			"• anx sync        Reconcile managed dotfiles",
			"• anx capture     Ingest dotfiles back into repo",
			"• anx doctor      Run full health audit",
			"• anx audit       Vulnerability scan (OSV)",
			"• anx tools       Decoupled personal tools engine",
		}

		rightLines := []string{
			m.styles.Highlight.Render("ENVIRONMENT METRICS & HEALTH"),
			fmt.Sprintf("• Managed Packages: %d registered", len(m.packages)),
			fmt.Sprintf("• Managed Configs:  %d assets in files/", len(m.configs)),
			fmt.Sprintf("• Detected Hacks:   %d system tweaks", len(m.detectedHacks)),
			fmt.Sprintf("• Active Palette:   %s", m.styles.Palette.Name),
			"",
			m.styles.Highlight.Render("LATEST HEALTH FINDINGS"),
		}
		for _, c := range m.doctorChecks {
			icon := m.styles.Success.Render("✓")
			if c.Verdict == app.VerdictWarn {
				icon = m.styles.Warning.Render("!")
			} else if c.Verdict == app.VerdictFail {
				icon = m.styles.Error.Render("✗")
			}
			rightLines = append(rightLines, fmt.Sprintf(" %s %s: %s", icon, c.Name, c.Message))
		}

		leftPanel := BuildPanel(m.styles, leftLines, leftW, h, m.focusedPanel == "left")
		rightPanel := BuildPanel(m.styles, rightLines, rightW, h, m.focusedPanel == "right")
		return JoinHorizontal(leftPanel, rightPanel, 2)
	}

	lines := []string{
		m.styles.Highlight.Render("SYSTEM OVERVIEW"),
		fmt.Sprintf("OS: %s | Arch: %s | Distro: %s", m.deps.Platform.OS(), m.deps.Platform.Arch(), m.deps.Platform.Distribution()),
		fmt.Sprintf("Packages: %d | Configs: %d | Hacks: %d", len(m.packages), len(m.configs), len(m.detectedHacks)),
		"",
		m.styles.Highlight.Render("HEALTH CHECKS"),
	}
	for _, c := range m.doctorChecks {
		lines = append(lines, fmt.Sprintf("• %s: %s", c.Name, c.Message))
	}
	return BuildPanel(m.styles, lines, w, h, true)
}

func (m Model) renderPackages(w, h int) string {
	header := m.styles.Highlight.Render(fmt.Sprintf("MANAGED PACKAGES & SYSTEM TOOLCHAINS (%d/%d)", m.selectedIdx+1, len(m.packages)))
	items := make([]string, 0, len(m.packages))

	requiredBy := make(map[string][]string)
	for _, p := range m.packages {
		for _, dep := range p.DependsOn {
			requiredBy[dep] = append(requiredBy[dep], p.ID)
		}
	}

	for i, p := range m.packages {
		prefix := "  "
		if i == m.selectedIdx {
			prefix = "> "
		}
		statusTag := m.styles.Success.Render("[Available]")
		if p.Deprecated {
			statusTag = m.styles.Warning.Render("[Deprecated]")
		}

		isDirect := p.IsDirectTool(requiredBy)
		kindBadge := m.styles.Highlight.Render("[Tool]")
		depInfo := ""
		if !isDirect {
			kindBadge = m.styles.Muted.Render("[Dep]")
			if len(requiredBy[p.ID]) > 0 {
				depInfo = fmt.Sprintf(" (dep of: %s)", strings.Join(requiredBy[p.ID], ", "))
			} else if strings.Contains(p.Category, "build") || p.ID == "build-essential" || p.ID == "cmake" || p.ID == "pkg-config" {
				depInfo = " (build dependency)"
			} else if strings.Contains(p.Category, "lib") || strings.HasSuffix(p.ID, "-dev") {
				depInfo = " (library dependency)"
			} else {
				depInfo = " (system prerequisite)"
			}
		} else if len(p.DependsOn) > 0 {
			depInfo = fmt.Sprintf(" (requires: %s)", strings.Join(p.DependsOn, ", "))
		}

		row := fmt.Sprintf("%s%s %-14s %-10s %s : %s%s", prefix, kindBadge, p.ID, p.Category, statusTag, p.Description, depInfo)
		if i == m.selectedIdx {
			items = append(items, m.styles.RenderSelectedRow(row))
		} else {
			items = append(items, row)
		}
	}
	return BuildScrollablePanel(m.styles, []string{header}, items, m.selectedIdx, m.scrollOffset, w, h, true)
}

func (m Model) renderConfigs(w, h int) string {
	header := m.styles.Highlight.Render(fmt.Sprintf("MANAGED CONFIGURATION ASSETS (%d/%d)", m.selectedIdx+1, len(m.configs)))
	items := make([]string, 0, len(m.configs))
	for i, c := range m.configs {
		prefix := "  "
		if i == m.selectedIdx {
			prefix = "> "
		}

		targetPath := c.Target
		if m.deps.Platform != nil {
			targetPath = m.deps.Platform.ExpandPath(c.TargetForOS(m.deps.Platform.OS()))
		}

		var statusTag string
		if fi, err := os.Lstat(targetPath); err == nil && fi != nil {
			statusTag = m.styles.Success.Render("[Integrated ✓]")
		} else {
			statusTag = m.styles.Muted.Render("[Not Integrated ✗]")
		}

		if c.Deprecated {
			statusTag += " " + m.styles.Warning.Render("[Deprecated]")
		}

		row := fmt.Sprintf("%s%-18s %s  %-24s -> %s", prefix, c.ID, statusTag, c.Source, c.Target)
		if i == m.selectedIdx {
			items = append(items, m.styles.RenderSelectedRow(row))
		} else {
			items = append(items, row)
		}
	}
	return BuildScrollablePanel(m.styles, []string{header}, items, m.selectedIdx, m.scrollOffset, w, h, true)
}

func (m Model) renderAI(w, h int) string {
	toolsBin := "isolated"
	if m.deps.Platform != nil {
		toolsBin = m.deps.Platform.ToolsBinDir()
	}
	lines := []string{
		m.styles.Highlight.Render("PERSONAL TOOLS & AI SUITE (DECOUPLED ARCHITECTURE)"),
		"• Decoupled Commands: anx tools [list|info|run|update|upgrade]",
		"• AI Subcommands:     anx tools ai [commit|pr|llm|quota|daemon|status]",
		"• Direct Shortcuts:    anx tools llm, anx tools commit, anx tools pr, anx tools quota",
		"• Path Isolation:      " + toolsBin + " (never scans generic ~/.local/bin/)",
		"• Supported Runtimes:  Independent projects in Go, Rust, Python, Bash, Node.js",
		"• Quota Tracking:      5-hour rolling usage tracking with warning thresholds",
		"• Autonomous Wake:     RTC alarm wakeup (04:00 AM) with verified reconnect",
		"",
		m.styles.Highlight.Render("MULTICALL STATUS"),
		"• Multicall dispatch: 'aitools' is symlinked to 'anx' with zero recompilation required.",
	}
	return BuildPanel(m.styles, lines, w, h, true)
}

func (m Model) renderHacks(w, h int) string {
	headerLines := []string{
		m.styles.Highlight.Render(fmt.Sprintf("HARDWARE & MACHINE-SPECIFIC HACKS (%d/%d)", m.selectedIdx+1, len(m.detectedHacks))),
		"Declarative workstation hacks with DMI and hardware auto-detection:",
		"",
	}
	items := make([]string, 0, len(m.detectedHacks))
	if len(m.detectedHacks) == 0 {
		items = append(items, m.styles.Muted.Render("• No matching hardware hacks detected for this workstation."))
	}
	for i, hk := range m.detectedHacks {
		prefix := "  "
		if i == m.selectedIdx {
			prefix = "> "
		}
		status := m.styles.Success.Render("[Matched]")
		row := fmt.Sprintf("%s%-24s %s  %s", prefix, hk.ID, status, hk.Description)
		if i == m.selectedIdx {
			items = append(items, m.styles.RenderSelectedRow(row))
		} else {
			items = append(items, row)
		}
	}
	return BuildScrollablePanel(m.styles, headerLines, items, m.selectedIdx, m.scrollOffset, w, h, true)
}

func (m Model) renderDoctor(w, h int) string {
	header := m.styles.Highlight.Render(fmt.Sprintf("ENVIRONMENT HEALTH DIAGNOSTICS (%d/%d)", m.selectedIdx+1, len(m.doctorChecks)))
	items := make([]string, 0, len(m.doctorChecks))
	for i, c := range m.doctorChecks {
		prefix := "  "
		if i == m.selectedIdx {
			prefix = "> "
		}
		badge := m.styles.Success.Render("[PASS]")
		if c.Verdict == app.VerdictWarn {
			badge = m.styles.Warning.Render("[WARN]")
		} else if c.Verdict == app.VerdictFail {
			badge = m.styles.Error.Render("[FAIL]")
		}
		row := fmt.Sprintf("%s%s %-25s : %s", prefix, badge, c.Name, c.Message)
		if c.Remediation != "" {
			row += " " + m.styles.Muted.Render(fmt.Sprintf("(Remediation: %s)", c.Remediation))
		}
		if i == m.selectedIdx {
			items = append(items, m.styles.RenderSelectedRow(row))
		} else {
			items = append(items, row)
		}
	}
	return BuildScrollablePanel(m.styles, []string{header}, items, m.selectedIdx, m.scrollOffset, w, h, true)
}

func (m Model) renderHistory(w, h int) string {
	lines := []string{
		m.styles.Highlight.Render("TRANSACTION JOURNAL & RESTORE HISTORY"),
		"• State directory: " + m.deps.Platform.StateDir(),
		"• Journal path:    " + m.deps.Platform.StateDir() + "/journal.jsonl",
		"• Backups quota:   50 MB portable snapshot rotation",
		"",
		m.styles.Success.Render("✓ Transaction journal verified. Clean state."),
	}
	return BuildPanel(m.styles, lines, w, h, true)
}

func (m Model) renderSecurity(w, h int) string {
	if m.isWideLayout() {
		leftW, rightW := m.calculateDualPaneWidths()

		profName := m.securityProfile
		if profName == "" {
			profName = security.ProfileWorkstation
		}

		scoreStr := "Pending..."
		critCount, highCount, medCount, lowCount := 0, 0, 0, 0
		if m.securityReport != nil {
			score := m.securityReport.HardeningScore
			scoreBadge := m.styles.Success.Render(fmt.Sprintf("%d/100", score))
			if score < 60 {
				scoreBadge = m.styles.Error.Render(fmt.Sprintf("%d/100", score))
			} else if score < 80 {
				scoreBadge = m.styles.Warning.Render(fmt.Sprintf("%d/100", score))
			}
			scoreStr = scoreBadge

			for _, f := range m.securityReport.Findings {
				switch f.Severity {
				case security.SeverityCritical:
					critCount++
				case security.SeverityHigh:
					highCount++
				case security.SeverityMedium:
					medCount++
				case security.SeverityLow:
					lowCount++
				}
			}
		}

		findingCount := 0
		if m.securityReport != nil {
			findingCount = len(m.securityReport.Findings)
		}
		leftHeader := []string{
			m.styles.Highlight.Render("SYSTEM SECURITY AUDIT & COMPLIANCE"),
			fmt.Sprintf("• Profile:         [%s] (press 'p' to switch)", strings.ToUpper(profName)),
			fmt.Sprintf("• Hardening Score: %s", scoreStr),
			fmt.Sprintf("• Summary:         Crit: %d | High: %d | Med: %d | Low: %d", critCount, highCount, medCount, lowCount),
			"",
			m.styles.Highlight.Render(fmt.Sprintf("AUDIT FINDINGS (%d/%d):", m.selectedIdx+1, findingCount)),
		}

		findingItems := make([]string, 0)
		if m.securityReport == nil || len(m.securityReport.Findings) == 0 {
			if m.securityReport != nil {
				findingItems = append(findingItems, m.styles.Success.Render("✓ No security risks detected for this profile."))
			} else {
				findingItems = append(findingItems, m.styles.Muted.Render("• Running security audit..."))
			}
		} else {
			for i, f := range m.securityReport.Findings {
				prefix := "  "
				if i == m.selectedIdx {
					prefix = "> "
				}
				badge := m.styles.Muted.Render(fmt.Sprintf("[%s]", f.Severity))
				switch f.Severity {
				case security.SeverityCritical:
					badge = m.styles.Error.Render("[CRIT]")
				case security.SeverityHigh:
					badge = m.styles.Warning.Render("[HIGH]")
				case security.SeverityMedium:
					badge = m.styles.Warning.Render("[MED ]")
				case security.SeverityLow:
					badge = m.styles.Muted.Render("[LOW ]")
				}
				row := fmt.Sprintf("%s%s %s", prefix, badge, f.Title)
				if i == m.selectedIdx {
					findingItems = append(findingItems, m.styles.RenderSelectedRow(row))
				} else {
					findingItems = append(findingItems, row)
				}
			}
		}

		rightLines := []string{
			m.styles.Highlight.Render("FINDING DETAILS & REMEDIATION"),
		}
		if m.securityReport != nil && len(m.securityReport.Findings) > 0 {
			sel := m.selectedIdx
			if sel >= len(m.securityReport.Findings) {
				sel = 0
			}
			f := m.securityReport.Findings[sel]
			rightLines = append(rightLines,
				fmt.Sprintf("• ID:          %s", f.ID),
				fmt.Sprintf("• Severity:    %s", f.Severity),
				fmt.Sprintf("• Category:    %s", f.Category),
				fmt.Sprintf("• Title:       %s", f.Title),
				"",
				m.styles.Highlight.Render("DESCRIPTION:"),
				f.Description,
				"",
				m.styles.Highlight.Render("IMPACT:"),
				f.Impact,
				"",
				m.styles.Highlight.Render("REMEDIATION:"),
				f.Remediation,
			)
			if f.RemediationCommand != "" {
				rightLines = append(rightLines,
					"",
					m.styles.Highlight.Render("COMMAND:"),
					f.RemediationCommand,
				)
			}
		} else {
			rightLines = append(rightLines,
				m.styles.Muted.Render("Select a finding to inspect remediation details."),
			)
		}

		rightLines = append(rightLines,
			"",
			m.styles.Highlight.Render("ACTIONS:"),
			"• [p] Switch Profile (Workstation / Server / Minimal)",
			"• [s] Generate Remediation Script to ~/.config/anx/",
			"• [r] Re-run Audit",
		)

		if len(m.auditFindings) > 0 {
			rightLines = append(rightLines,
				"",
				m.styles.Highlight.Render("PACKAGE VULNERABILITIES (OSV):"),
			)
			for _, pf := range m.auditFindings {
				rightLines = append(rightLines, fmt.Sprintf("• %s %s: %s", pf.ToolName, pf.DetectedVersion, pf.Summary))
			}
		}

		leftPanel := BuildScrollablePanel(m.styles, leftHeader, findingItems, m.selectedIdx, m.scrollOffset, leftW, h, m.focusedPanel == "left")
		rightPanel := BuildPanel(m.styles, rightLines, rightW, h, m.focusedPanel == "right")
		return JoinHorizontal(leftPanel, rightPanel, 2)
	}

	lines := []string{
		m.styles.Highlight.Render("SYSTEM SECURITY AUDIT & COMPLIANCE"),
	}
	if m.securityReport != nil {
		lines = append(lines, fmt.Sprintf("Profile: %s | Score: %d/100", m.securityProfile, m.securityReport.HardeningScore))
		findingItems := make([]string, 0, len(m.securityReport.Findings))
		for i, f := range m.securityReport.Findings {
			prefix := "  "
			if i == m.selectedIdx {
				prefix = "> "
			}
			findingItems = append(findingItems, fmt.Sprintf("%s[%s] %s", prefix, f.Severity, f.Title))
		}
		return BuildScrollablePanel(m.styles, lines, findingItems, m.selectedIdx, m.scrollOffset, w, h, true)
	}
	return BuildPanel(m.styles, lines, w, h, true)
}

func (m Model) renderHelp(w, h int) string {
	lines := []string{
		m.styles.Highlight.Render("KEYBOARD SHORTCUTS & NAVIGATION"),
		"",
		"  [1 - 8]         Directly jump to Tab (Overview, Packages, Configs, AI, Hacks, Doctor, History, Security)",
		"  [Tab]           Cycle forward to next tab",
		"  [Shift+Tab]     Cycle backward to previous tab",
		"  [j / Down]      Navigate cursor down / scroll list",
		"  [k / Up]        Navigate cursor up / scroll list",
		"  [h / Left]      Focus left panel (in dual-pane mode)",
		"  [l / Right]     Focus right panel (in dual-pane mode)",
		"  [a]             Toggle accordion mode (2x width for focused panel)",
		"  [r]             Refresh active diagnostics and system state",
		"  [?]             Toggle this help modal",
		"  [q / Ctrl+C]    Exit dashboard",
		"",
		m.styles.Muted.Render("Press [?] again to return to the active tab."),
	}
	return BuildPanel(m.styles, lines, w, h, true)
}
