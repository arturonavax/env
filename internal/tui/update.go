package tui

import (
	tea "charm.land/bubbletea/v2"
)

// Update implements the core Bubble Tea v2 message loop.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case doctorLoadedMsg:
		m.doctorChecks = msg
		return m, nil

	case auditLoadedMsg:
		m.auditFindings = msg
		return m, nil

	case systemAuditLoadedMsg:
		m.securityReport = msg
		return m, nil

	case hacksLoadedMsg:
		m.detectedHacks = msg
		return m, nil

	case updateCheckLoadedMsg:
		if msg.err == nil {
			m.updateCheck = &msg.result
		}
		return m, nil

	case statusUpdateMsg:
		m.statusMsg = string(msg)
		return m, nil

	case errMsg:
		m.statusMsg = "Error: " + msg.err.Error()
		return m, nil

	case tea.KeyPressMsg:
		return m.handleKeyPress(msg)

	case tea.MouseClickMsg, tea.MouseWheelMsg:
		return m.handleMouse(msg)
	}

	return m, nil
}
