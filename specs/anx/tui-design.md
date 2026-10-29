# anx — TUI Design & Tokyo Night Visual Architecture

## 1. Bubbletea v2 Architecture

`anx` uses `charm.land/bubbletea/v2` with Elm Architecture:
- **Model**: Immutable-by-convention state struct storing window dimensions, active tab, focused pane, scroll positions, loaded data, and notifications.
- **Update**: Pure message dispatcher handling `tea.KeyPressMsg`, `tea.WindowSizeMsg`, `tea.MouseClickMsg`, and custom typed events (`ProgressMsg`, `TaskCompletedMsg`).
- **View**: Returns `tea.View` configured with `AltScreen: true` and declarative mouse tracking mode.

## 2. The 4 Golden Rules of Layout (Mandatory)

1. **Always Account for Borders**:
   Subtract 2 from height calculations BEFORE rendering panels:
   `availableHeight = windowHeight - headerHeight - statusBarHeight - 2`
   Never set explicit `.Height()` on bordered Lipgloss styles; fill content lines instead.

2. **Never Auto-Wrap in Bordered Panels**:
   Explicitly truncate text to `panelWidth - 4` (accounting for borders and padding) to prevent unexpected line multiplication and border distortion.

3. **Match Mouse Detection to Layout**:
   Use X coordinates for horizontal splits, Y coordinates for vertical stacks.

4. **Use Weights, Not Pixels**:
   Proportional weight-based calculations (`width * weight / totalWeight`) scale smoothly across terminal resizes.

## 3. Tokyo Night Semantic Palette

| Semantic Token | Color Code | Application |
|---|---|---|
| Background | `#1a1b26` | Terminal background |
| Foreground | `#c0caf5` | Primary body text |
| Accent / Blue | `#7aa2f7` | Active tab, focused borders, highlights |
| Cyan | `#7dcfff` | Keybindings, secondary indicators, path tokens |
| Purple | `#bb9af7` | Headers, section titles, category badges |
| Green | `#9ece6a` | Success, installed state, added diff lines |
| Yellow | `#e0af68` | Warnings, drifted state, pending updates |
| Red | `#f7768e` | Errors, missing tools, deleted diff lines |
| Muted | `#565f89` | Dim text, inactive borders, breadcrumbs |

## 4. Responsive Breakpoints

- **Wide Layout (≥ 120 cols)**:
  - Header: System stats, active profile, anx version, time.
  - Body: Dual-pane layout. Left pane (40% weight): navigation tree / package list. Right pane (60% weight): detailed inspector / live preview / diff viewer.
  - Footer: Contextual keyboard shortcuts, status message, transaction indicator.

- **Medium Layout (80–119 cols)**:
  - Header: Condensed system stats.
  - Body: Tab-based views (Dashboard, Packages, Configs, Updates, Security, Doctor, History). Focused view takes full width.
  - Footer: Compact shortcuts.

- **Narrow Layout (< 80 cols / mobile Termux)**:
  - Single column with vertical stacking.
  - Minimalistic borders or simple text dividers.
  - Truncated labels with indicators.
