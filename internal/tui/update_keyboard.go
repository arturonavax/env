package tui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/arturonavax/env/internal/security"
)

// handleKeyPress processes keyboard interactions and navigation.
func (m Model) handleKeyPress(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	switch key {
	case "q", "ctrl+c":
		if m.inspectingPackage != nil {
			m.inspectingPackage = nil
			return m, nil
		}
		m.quitting = true
		return m, tea.Quit

	case "?":
		m.showHelp = !m.showHelp
		return m, nil

	case "enter":
		if m.inspectingPackage != nil {
			m.inspectingPackage = nil
			return m, nil
		}
		if m.activeTab == TabPackages {
			if m.focusedPanel == "right" && len(m.depsList) > 0 && m.depsSelectedIdx < len(m.depsList) {
				pkg := m.depsList[m.depsSelectedIdx]
				m.inspectingPackage = &pkg
				return m, nil
			}
			if len(m.tools) > 0 && m.toolsSelectedIdx < len(m.tools) {
				pkg := m.tools[m.toolsSelectedIdx]
				m.inspectingPackage = &pkg
				return m, nil
			}
		}

	case "esc":
		if m.inspectingPackage != nil {
			m.inspectingPackage = nil
			return m, nil
		}
		if m.showHelp {
			m.showHelp = false
			return m, nil
		}

	case "ctrl+d":
		halfPage := max(1, m.visibleItemLines()/2)
		m.selectedIdx += halfPage
		maxIdx := m.maxSelectedIndex()
		if m.selectedIdx > maxIdx {
			m.selectedIdx = maxIdx
		}
		m.adjustScroll()

	case "ctrl+u":
		halfPage := max(1, m.visibleItemLines()/2)
		m.selectedIdx -= halfPage
		if m.selectedIdx < 0 {
			m.selectedIdx = 0
		}
		m.adjustScroll()

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
		m.scrollOffset = 0
	case "2":
		m.activeTab = TabPackages
		m.selectedIdx = 0
		m.scrollOffset = 0
	case "3":
		m.activeTab = TabConfigs
		m.selectedIdx = 0
		m.scrollOffset = 0
	case "4":
		m.activeTab = TabAI
		m.selectedIdx = 0
		m.scrollOffset = 0
	case "5":
		m.activeTab = TabHacks
		m.selectedIdx = 0
		m.scrollOffset = 0
	case "6":
		m.activeTab = TabDoctor
		m.selectedIdx = 0
		m.scrollOffset = 0
	case "7":
		m.activeTab = TabHistory
		m.selectedIdx = 0
		m.scrollOffset = 0
	case "8":
		m.activeTab = TabSecurity
		m.selectedIdx = 0
		m.scrollOffset = 0

	case "j", "down":
		m.selectedIdx++
		maxIdx := m.maxSelectedIndex()
		if m.selectedIdx > maxIdx {
			m.selectedIdx = maxIdx
		}
		m.adjustScroll()

	case "k", "up":
		if m.selectedIdx > 0 {
			m.selectedIdx--
		}
		m.adjustScroll()

	case "h", "left":
		m.focusedPanel = "left"
		if m.activeTab == TabPackages {
			m.selectedIdx = m.toolsSelectedIdx
			m.scrollOffset = m.toolsScrollOffset
		} else if m.activeTab == TabConfigs {
			m.selectedIdx = m.intConfigsSelectedIdx
			m.scrollOffset = m.intConfigsScrollOffset
		}

	case "l", "right":
		m.focusedPanel = "right"
		if m.activeTab == TabPackages {
			m.selectedIdx = m.depsSelectedIdx
			m.scrollOffset = m.depsScrollOffset
		} else if m.activeTab == TabConfigs {
			m.selectedIdx = m.notIntConfigsSelectedIdx
			m.scrollOffset = m.notIntConfigsScrollOffset
		}

	case "a":
		m.accordionMode = !m.accordionMode
		if m.accordionMode {
			m.statusMsg = "Accordion mode enabled (focused panel gets 2x width)"
		} else {
			m.statusMsg = "Accordion mode disabled (equal 50/50 split)"
		}

	case "p":
		if m.activeTab == TabSecurity {
			switch m.securityProfile {
			case security.ProfileWorkstation:
				m.securityProfile = security.ProfileServer
			case security.ProfileServer:
				m.securityProfile = security.ProfileMinimal
			default:
				m.securityProfile = security.ProfileWorkstation
			}
			m.selectedIdx = 0
			m.statusMsg = fmt.Sprintf("Switched to [%s] profile. Re-auditing...", m.securityProfile)
			ctx := context.Background()
			return m, loadSystemAuditCmd(ctx, m.deps.Audit, m.securityProfile)
		}

	case "s":
		if m.activeTab == TabSecurity {
			if m.securityReport == nil {
				m.statusMsg = "No audit report available to generate script."
				return m, nil
			}
			auditor := security.NewSecurityAuditor(security.WithAuditorProfile(m.securityProfile))
			script, err := auditor.SynthesizeScript(context.Background(), m.securityReport)
			if err != nil {
				m.statusMsg = "Script generation failed: " + err.Error()
				return m, nil
			}
			targetDir := ""
			if m.deps.Platform != nil {
				targetDir = m.deps.Platform.ConfigDir()
			}
			if targetDir != "" {
				_ = os.MkdirAll(targetDir, 0755)
				outPath := filepath.Join(targetDir, "security-remediation.sh")
				if err := os.WriteFile(outPath, []byte(script), 0700); err == nil {
					m.statusMsg = "Remediation script saved to " + outPath
					return m, nil
				}
			}
			m.statusMsg = fmt.Sprintf("Remediation script generated (%d bytes).", len(script))
			return m, nil
		}

	case "r":
		m.statusMsg = "Refreshing diagnostics at " + time.Now().Format("15:04:05")
		ctx := context.Background()
		return m, tea.Batch(
			tea.Cmd(loadDoctorCmd(ctx, m.deps.Doctor)),
			tea.Cmd(loadAuditCmd(ctx, m.deps.Audit)),
			tea.Cmd(loadSystemAuditCmd(ctx, m.deps.Audit, m.securityProfile)),
			tea.Cmd(loadHacksCmd(ctx, m.deps.Hacks)),
		)
	}

	return m, nil
}

func (m Model) maxSelectedIndex() int {
	switch m.activeTab {
	case TabPackages:
		if m.focusedPanel == "right" {
			if len(m.depsList) > 0 {
				return len(m.depsList) - 1
			}
			return 0
		}
		if len(m.tools) > 0 {
			return len(m.tools) - 1
		}
		if len(m.packages) > 0 {
			return len(m.packages) - 1
		}
	case TabConfigs:
		if m.focusedPanel == "right" {
			if len(m.notIntegratedConfigs) > 0 {
				return len(m.notIntegratedConfigs) - 1
			}
			return 0
		}
		if len(m.integratedConfigs) > 0 {
			return len(m.integratedConfigs) - 1
		}
		if len(m.configs) > 0 {
			return len(m.configs) - 1
		}
	case TabDoctor:
		if len(m.doctorChecks) > 0 {
			return len(m.doctorChecks) - 1
		}
	case TabSecurity:
		if m.securityReport != nil && len(m.securityReport.Findings) > 0 {
			return len(m.securityReport.Findings) - 1
		}
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

func (m *Model) adjustScroll() {
	visibleLines := m.visibleItemLines()
	totalItems := m.maxSelectedIndex() + 1
	m.scrollOffset = AdjustScrollOffset(m.selectedIdx, m.scrollOffset, visibleLines, totalItems)
	if m.activeTab == TabPackages {
		if m.focusedPanel == "right" {
			m.depsSelectedIdx = m.selectedIdx
			m.depsScrollOffset = m.scrollOffset
		} else {
			m.toolsSelectedIdx = m.selectedIdx
			m.toolsScrollOffset = m.scrollOffset
		}
	} else if m.activeTab == TabConfigs {
		if m.focusedPanel == "right" {
			m.notIntConfigsSelectedIdx = m.selectedIdx
			m.notIntConfigsScrollOffset = m.scrollOffset
		} else {
			m.intConfigsSelectedIdx = m.selectedIdx
			m.intConfigsScrollOffset = m.scrollOffset
		}
	}
}

func (m Model) visibleItemLines() int {
	availHeight := m.height - 4 - 2 // header, tabs, status, borders
	if availHeight < 4 {
		availHeight = 4
	}
	switch m.activeTab {
	case TabPackages, TabConfigs, TabDoctor:
		return availHeight - 1
	case TabHacks:
		return availHeight - 3
	case TabSecurity:
		return availHeight - 6
	default:
		return availHeight
	}
}
