package tui

import (
	tea "charm.land/bubbletea/v2"
)

// handleMouse implements mouse handling adhering strictly to Golden Rule #3:
// - Uses X coordinates for horizontal tab bar and side-by-side dual-pane panels
// - Uses Y coordinates for vertical list item selection
func (m Model) handleMouse(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.MouseClickMsg:
		if msg.Button != tea.MouseLeft {
			return m, nil
		}

		// Row 1 is the horizontal Tab bar -> Use X coordinates to switch tabs
		if msg.Y == 1 {
			tabIdx := m.calculateTabFromX(msg.X)
			if tabIdx >= 0 && tabIdx < 8 {
				m.activeTab = TabID(tabIdx)
				m.selectedIdx = 0
				m.scrollOffset = 0
				return m, nil
			}
		}

		// Content rows (msg.Y >= 2)
		if m.isWideLayout() {
			// Side-by-side mode: use X coordinates to focus panel
			leftWidth, _ := m.calculateDualPaneWidths()
			if msg.X < leftWidth {
				m.focusedPanel = "left"
			} else {
				m.focusedPanel = "right"
			}
		}

		// Vertical list selection: use relative Y coordinate
		contentStartY := 3 // Header + Tabs + Panel top border
		relY := msg.Y - contentStartY
		if relY >= 0 {
			maxIdx := m.maxSelectedIndex()
			if relY <= maxIdx {
				m.selectedIdx = relY
			}
		}

	case tea.MouseWheelMsg:
		switch msg.Button {
		case tea.MouseWheelUp:
			if m.selectedIdx > 0 {
				m.selectedIdx--
			}
		case tea.MouseWheelDown:
			if m.selectedIdx < m.maxSelectedIndex() {
				m.selectedIdx++
			}
		}
	}

	return m, nil
}

// calculateTabFromX matches horizontal tab clicks to TabID (Golden Rule #3).
func (m Model) calculateTabFromX(x int) int {
	// Approximate character ranges for:
	// " 1:Overview  2:Packages  3:Configs  4:AI  5:Hacks  6:Doctor  7:History  8:Security "
	switch {
	case x < 13:
		return int(TabOverview)
	case x < 25:
		return int(TabPackages)
	case x < 36:
		return int(TabConfigs)
	case x < 44:
		return int(TabAI)
	case x < 54:
		return int(TabHacks)
	case x < 65:
		return int(TabDoctor)
	case x < 77:
		return int(TabHistory)
	default:
		return int(TabSecurity)
	}
}

// isWideLayout checks if the terminal is wide enough for side-by-side panels.
func (m Model) isWideLayout() bool {
	return m.width >= 100
}

// calculateDualPaneWidths computes weights-based panel widths (Golden Rule #4).
func (m Model) calculateDualPaneWidths() (int, int) {
	availWidth := m.width
	if availWidth < 40 {
		availWidth = 40
	}

	leftWeight, rightWeight := 1, 1
	if m.accordionMode {
		if m.focusedPanel == "left" {
			leftWeight = 2
		} else {
			rightWeight = 2
		}
	}

	totalWeight := leftWeight + rightWeight
	leftWidth := (availWidth * leftWeight) / totalWeight
	rightWidth := availWidth - leftWidth - 2 // 2 char horizontal gap

	if leftWidth < 20 {
		leftWidth = 20
	}
	if rightWidth < 20 {
		rightWidth = 20
	}

	return leftWidth, rightWidth
}
