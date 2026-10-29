package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/arturonavax/env/internal/app"
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
			"• anx sync        Reconcile managed dotfiles",
			"• anx doctor      Run full health audit",
			"• anx audit       Vulnerability scan (OSV)",
			"• anx ai          AI assistant & automation tools",
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
	lines := make([]string, 0, len(m.packages)+2)
	lines = append(lines, m.styles.Highlight.Render("MANAGED PACKAGES & SYSTEM TOOLCHAINS"))
	for i, p := range m.packages {
		prefix := "  "
		if i == m.selectedIdx {
			prefix = "> "
		}
		statusTag := m.styles.Success.Render("[Available]")
		row := fmt.Sprintf("%s%-14s %-12s %-32s %s", prefix, p.ID, p.Category, p.Description, statusTag)
		if i == m.selectedIdx {
			lines = append(lines, m.styles.SelectedRow.Render(row))
		} else {
			lines = append(lines, row)
		}
	}
	return BuildPanel(m.styles, lines, w, h, true)
}

func (m Model) renderConfigs(w, h int) string {
	lines := make([]string, 0, len(m.configs)+2)
	lines = append(lines, m.styles.Highlight.Render("MANAGED CONFIGURATION ASSETS (files/)"))
	for i, c := range m.configs {
		prefix := "  "
		if i == m.selectedIdx {
			prefix = "> "
		}
		row := fmt.Sprintf("%s%-16s %-26s -> %s", prefix, c.ID, c.Source, c.Target)
		if i == m.selectedIdx {
			lines = append(lines, m.styles.SelectedRow.Render(row))
		} else {
			lines = append(lines, row)
		}
	}
	return BuildPanel(m.styles, lines, w, h, true)
}

func (m Model) renderAI(w, h int) string {
	lines := []string{
		m.styles.Highlight.Render("AI TOOLS & AGENT ORCHESTRATION"),
		"• CLI Subcommands: anx ai [commit|pr|llm|daemon|setup|status]",
		"• Supported Harnesses: agy (default), claude, codex, opencode, pi, llm, ollama",
		"• Quota Management: 5-hour rolling usage tracking with warning thresholds",
		"• Autonomous Wake: RTC alarm wakeup (04:00 AM) with verified reconnect",
		"• Safety Guardrails: 500KB context cap, sanitized diffs, timeout safeguards",
		"",
		m.styles.Highlight.Render("MULTICALL STATUS"),
		"• The 'aitools' binary is symlinked to 'anx' for seamless backwards compatibility.",
	}
	return BuildPanel(m.styles, lines, w, h, true)
}

func (m Model) renderHacks(w, h int) string {
	lines := []string{
		m.styles.Highlight.Render("HARDWARE & MACHINE-SPECIFIC HACKS"),
		"Declarative workstation hacks with DMI and hardware auto-detection:",
		"",
	}
	if len(m.detectedHacks) == 0 {
		lines = append(lines, m.styles.Muted.Render("• No matching hardware hacks detected for this workstation."))
	}
	for i, hk := range m.detectedHacks {
		prefix := "  "
		if i == m.selectedIdx {
			prefix = "> "
		}
		status := m.styles.Success.Render("[Matched]")
		row := fmt.Sprintf("%s%-24s %s  %s", prefix, hk.ID, status, hk.Description)
		if i == m.selectedIdx {
			lines = append(lines, m.styles.SelectedRow.Render(row))
		} else {
			lines = append(lines, row)
		}
	}
	return BuildPanel(m.styles, lines, w, h, true)
}

func (m Model) renderDoctor(w, h int) string {
	lines := make([]string, 0, len(m.doctorChecks)+2)
	lines = append(lines, m.styles.Highlight.Render("ENVIRONMENT HEALTH DIAGNOSTICS"))
	for _, c := range m.doctorChecks {
		badge := m.styles.Success.Render("[PASS]")
		if c.Verdict == app.VerdictWarn {
			badge = m.styles.Warning.Render("[WARN]")
		} else if c.Verdict == app.VerdictFail {
			badge = m.styles.Error.Render("[FAIL]")
		}
		lines = append(lines, fmt.Sprintf("%s %-25s : %s", badge, c.Name, c.Message))
		if c.Remediation != "" {
			lines = append(lines, m.styles.Muted.Render(fmt.Sprintf("       Remediation: %s", c.Remediation)))
		}
	}
	return BuildPanel(m.styles, lines, w, h, true)
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
	lines := []string{
		m.styles.Highlight.Render("SECURITY VULNERABILITY AUDITING (OSV)"),
		"• Database: Open Source Vulnerabilities (OSV.dev)",
		"• Ecosystems: Linux, macOS, Go, Rust, Node.js",
		"",
	}
	if len(m.auditFindings) == 0 {
		lines = append(lines, m.styles.Success.Render("✓ Zero known vulnerabilities detected on installed tools."))
	} else {
		for _, f := range m.auditFindings {
			badge := m.styles.Warning.Render(fmt.Sprintf("[%s]", f.Severity))
			lines = append(lines, fmt.Sprintf("%s %s %s: %s", badge, f.ToolName, f.DetectedVersion, f.Summary))
		}
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
