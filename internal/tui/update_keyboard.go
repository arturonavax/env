package tui

import (
	"context"
	"time"

	tea "charm.land/bubbletea/v2"
)

// handleKeyPress processes keyboard interactions and navigation.
func (m Model) handleKeyPress(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	switch key {
	case "q", "ctrl+c":
		m.quitting = true
		return m, tea.Quit

	case "?":
		m.showHelp = !m.showHelp
		return m, nil

	case "tab":
		m.activeTab = (m.activeTab + 1) % 8
		m.selectedIdx = 0
		m.scrollOffset = 0
		return m, nil

	case "shift+tab":
		if m.activeTab == 0 {
			m.activeTab = TabSecurity
		} else {
			m.activeTab--
		}
		m.selectedIdx = 0
		m.scrollOffset = 0
		return m, nil

	case "1":
		m.activeTab = TabOverview
		m.selectedIdx = 0
	case "2":
		m.activeTab = TabPackages
		m.selectedIdx = 0
	case "3":
		m.activeTab = TabConfigs
		m.selectedIdx = 0
	case "4":
		m.activeTab = TabAI
		m.selectedIdx = 0
	case "5":
		m.activeTab = TabHacks
		m.selectedIdx = 0
	case "6":
		m.activeTab = TabDoctor
		m.selectedIdx = 0
	case "7":
		m.activeTab = TabHistory
		m.selectedIdx = 0
	case "8":
		m.activeTab = TabSecurity
		m.selectedIdx = 0

	case "j", "down":
		m.selectedIdx++
		maxIdx := m.maxSelectedIndex()
		if m.selectedIdx > maxIdx {
			m.selectedIdx = maxIdx
		}

	case "k", "up":
		if m.selectedIdx > 0 {
			m.selectedIdx--
		}

	case "h", "left":
		m.focusedPanel = "left"

	case "l", "right":
		m.focusedPanel = "right"

	case "a":
		m.accordionMode = !m.accordionMode
		if m.accordionMode {
			m.statusMsg = "Accordion mode enabled (focused panel gets 2x width)"
		} else {
			m.statusMsg = "Accordion mode disabled (equal 50/50 split)"
		}

	case "r":
		m.statusMsg = "Refreshing diagnostics at " + time.Now().Format("15:04:05")
		ctx := context.Background()
		return m, tea.Batch(
			tea.Cmd(loadDoctorCmd(ctx, m.deps.Doctor)),
			tea.Cmd(loadAuditCmd(ctx, m.deps.Audit)),
			tea.Cmd(loadHacksCmd(ctx, m.deps.Hacks)),
		)
	}

	return m, nil
}

func (m Model) maxSelectedIndex() int {
	switch m.activeTab {
	case TabPackages:
		if len(m.packages) > 0 {
			return len(m.packages) - 1
		}
	case TabConfigs:
		if len(m.configs) > 0 {
			return len(m.configs) - 1
		}
	case TabDoctor:
		if len(m.doctorChecks) > 0 {
			return len(m.doctorChecks) - 1
		}
	case TabSecurity:
		if len(m.auditFindings) > 0 {
			return len(m.auditFindings) - 1
		}
	case TabHacks:
		if len(m.detectedHacks) > 0 {
			return len(m.detectedHacks) - 1
		}
	}
	return 0
}
