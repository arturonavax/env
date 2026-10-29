package wizard

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/arturonavax/env/internal/ai/config"
	"github.com/arturonavax/env/internal/ai/harness"
	"github.com/charmbracelet/x/ansi"
)

const wizardVerticalBreakpoint = 86

type stepChoice struct {
	Value       string
	Label       string
	Description string
}

type wizardStep struct {
	Title    string
	HelpText string
	Choices  []stepChoice
}

// Model implements a Bubble Tea v2 interactive configuration wizard adhering to the 4 Golden Rules.
type Model struct {
	cfg        *config.Config
	configPath string
	steps      []wizardStep

	stepIdx      int
	cursor       int
	scrollOffset int
	focusedPane  string // "left" or "right"
	width        int
	height       int

	selectedBackend  string
	selectedEffort   string
	selectedCommit   string
	selectedPR       string
	selectedAutoWake string // "enabled_resuspend", "enabled_awake", "disabled"

	saved    bool
	aborted  bool
	saveErr  error
	quitting bool
}

// NewModel constructs an interactive configuration wizard Model seeded from cfg.
func NewModel(cfg *config.Config, configPath string, reg *harness.Registry) Model {
	if cfg == nil {
		cfg, _ = config.DefaultConfig()
	}
	if configPath == "" {
		configPath = config.DefaultConfigPath("")
	}
	if reg == nil {
		reg = harness.NewRegistry(cfg)
	}

	statuses := reg.ListStatuses()
	statusMap := make(map[string]harness.BackendStatus, len(statuses))
	for _, st := range statuses {
		statusMap[st.Name] = st
	}

	harnessOrder := []string{"agy", "claude", "codex", "opencode", "pi", "llm", "ollama"}
	for _, st := range statuses {
		if !slices.Contains(harnessOrder, st.Name) {
			harnessOrder = append(harnessOrder, st.Name)
		}
	}

	harnessChoices := make([]stepChoice, 0, len(harnessOrder))
	for _, name := range harnessOrder {
		st, ok := statusMap[name]
		badge := "not in PATH"
		desc := "AI CLI harness"
		if ok {
			desc = st.Description
			if st.Available {
				badge = "installed"
			}
		}
		harnessChoices = append(harnessChoices, stepChoice{
			Value:       name,
			Label:       fmt.Sprintf("%-10s [%s]", name, badge),
			Description: desc,
		})
	}

	steps := []wizardStep{
		{
			Title:    "Step 1/6 · Default AI Harness",
			HelpText: "Select the primary AI CLI harness used across commit, PR, llm, and declarative pipelines.",
			Choices:  harnessChoices,
		},
		{
			Title:    "Step 2/6 · Default Reasoning Effort",
			HelpText: "Choose default reasoning budget passed to harnesses supporting effort flags.",
			Choices: []stepChoice{
				{Value: "low", Label: "low", Description: "Fastest responses, minimal token usage"},
				{Value: "medium", Label: "medium", Description: "Balanced reasoning speed and depth"},
				{Value: "high", Label: "high", Description: "Deep architectural and multi-file reasoning"},
				{Value: "xhigh", Label: "xhigh", Description: "Extended verification reasoning"},
				{Value: "max", Label: "max", Description: "Maximum reasoning budget"},
			},
		},
		{
			Title:    "Step 3/6 · Default Commit Template",
			HelpText: "Choose the default git commit prompt structure (repository .gitmessage auto-detection remains active).",
			Choices: []stepChoice{
				{Value: "conventional", Label: "conventional", Description: "<type>(<scope>): <summary>"},
				{Value: "task", Label: "task", Description: "[<TASK-ID>] <type>(<scope>): <summary>"},
				{Value: "task-scope", Label: "task-scope", Description: "<type>(<TASK-ID>): <summary>"},
				{Value: "detailed", Label: "detailed", Description: "Header + WHAT/WHY paragraph + architectural bullets"},
				{Value: "repo", Label: "repo", Description: "Strictly prioritize repository commit template"},
			},
		},
		{
			Title:    "Step 4/6 · Pull Request Template Mode",
			HelpText: "Select the default Pull Request description structure when no .github/pull_request_template.md is present.",
			Choices: []stepChoice{
				{Value: "standard", Label: "standard", Description: "Summary, Key Changes, Decisions, Verification & Checklist"},
				{Value: "task", Label: "task", Description: "Ticket-linked PR ([TASK-ID] Title, Problem, Solution)"},
				{Value: "minimal", Label: "minimal", Description: "Concise Overview, Changes & Verification"},
				{Value: "detailed", Label: "detailed", Description: "Full architectural deep-dive specification"},
				{Value: "repo", Label: "repo", Description: "Repository PR template with standard fallback"},
			},
		},
		{
			Title:    "Step 5/6 · Hardware RTC Auto-Wake & Quota Priming",
			HelpText: "Configure daily 04:00 AM rolling 5-hour quota priming and automatic power resuspend.",
			Choices: []stepChoice{
				{Value: "enabled_resuspend", Label: "Enabled + Auto-Resuspend", Description: "Wake at 04:00, verify network & probe, resuspend in 12s"},
				{Value: "enabled_awake", Label: "Enabled (Stay Awake)", Description: "Wake at 04:00 and prime quotas without resuspending"},
				{Value: "disabled", Label: "Disabled", Description: "Do not schedule automatic RTC wakeups"},
			},
		},
		{
			Title:    "Step 6/6 · Review & Save Configuration",
			HelpText: "Confirm your preferences to write atomically to config.json.",
			Choices: []stepChoice{
				{Value: "save", Label: "Save configuration", Description: "Write preferences atomically to " + configPath},
				{Value: "cancel", Label: "Cancel without saving", Description: "Discard changes and exit"},
			},
		},
	}

	awMode := "disabled"
	if cfg.Daemon.AutoWake.Enabled {
		if cfg.Daemon.AutoWake.AutoResuspend {
			awMode = "enabled_resuspend"
		} else {
			awMode = "enabled_awake"
		}
	}

	defBackend := cfg.Commit.DefaultBackend
	if defBackend == "" {
		defBackend = cfg.DefaultCommitBackend
	}
	if defBackend == "" {
		defBackend = "agy"
	}

	defEffort := cfg.Commit.Effort
	if defEffort == "" {
		defEffort = "low"
	}

	defCommit := cfg.Commit.DefaultTemplate
	if defCommit == "" {
		defCommit = "conventional"
	}

	defPR := cfg.PR.DefaultTemplate
	if defPR == "" {
		defPR = "standard"
	}

	m := Model{
		cfg:              cfg,
		configPath:       configPath,
		steps:            steps,
		focusedPane:      "left",
		width:            96,
		height:           26,
		selectedBackend:  defBackend,
		selectedEffort:   defEffort,
		selectedCommit:   defCommit,
		selectedPR:       defPR,
		selectedAutoWake: awMode,
	}
	m.syncCursorToCurrentValue()
	return m
}

// Init satisfies tea.Model.
func (m Model) Init() tea.Cmd { return nil }

// Saved reports whether the user confirmed and saved configuration.
func (m Model) Saved() bool { return m.saved }

// SaveError returns any error encountered while persisting configuration.
func (m Model) SaveError() error { return m.saveErr }

// Update handles Bubble Tea v2 messages (WindowSizeMsg, KeyPressMsg, MouseClickMsg).
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch typed := msg.(type) {
	case tea.WindowSizeMsg:
		if typed.Width > 0 {
			m.width = typed.Width
		}
		if typed.Height > 0 {
			m.height = typed.Height
		}
		m.clampScroll()
		return m, nil

	case tea.KeyPressMsg:
		return m.handleKey(typed)

	case tea.MouseClickMsg:
		return m.handleMouse(typed)
	}
	return m, nil
}

func (m Model) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	choices := m.steps[m.stepIdx].Choices

	switch msg.String() {
	case "ctrl+c", "q":
		m.aborted = true
		m.quitting = true
		return m, tea.Quit

	case "esc", "backspace", "h", "left":
		if m.stepIdx > 0 {
			m.stepIdx--
			m.syncCursorToCurrentValue()
		}
		return m, nil

	case "tab":
		if m.focusedPane == "left" {
			m.focusedPane = "right"
		} else {
			m.focusedPane = "left"
		}
		return m, nil

	case "j", "down":
		if m.cursor+1 < len(choices) {
			m.cursor++
			m.clampScroll()
		}
		return m, nil

	case "k", "up":
		if m.cursor > 0 {
			m.cursor--
			m.clampScroll()
		}
		return m, nil

	case "enter", "l", "right":
		return m.confirmStepSelection()
	}

	return m, nil
}

// Golden Rule #3: Match mouse coordinates to horizontal vs vertical layout orientation.
func (m Model) handleMouse(msg tea.MouseClickMsg) (tea.Model, tea.Cmd) {
	mouse := msg.Mouse()
	const titleLines = 2
	if m.isVerticalStack() {
		topInner, _ := m.verticalInnerHeights(titleLines, 1)
		if mouse.Y-titleLines < topInner+2 {
			m.focusedPane = "left"
		} else {
			m.focusedPane = "right"
		}
		return m, nil
	}

	leftW, _ := m.horizontalWidths()
	if mouse.X < leftW {
		m.focusedPane = "left"
	} else {
		m.focusedPane = "right"
	}
	return m, nil
}

func (m Model) confirmStepSelection() (tea.Model, tea.Cmd) {
	choices := m.steps[m.stepIdx].Choices
	if m.cursor < 0 || m.cursor >= len(choices) {
		return m, nil
	}
	val := choices[m.cursor].Value

	switch m.stepIdx {
	case 0:
		m.selectedBackend = val
	case 1:
		m.selectedEffort = val
	case 2:
		m.selectedCommit = val
	case 3:
		m.selectedPR = val
	case 4:
		m.selectedAutoWake = val
	case 5:
		if val == "save" {
			m.saveErr = m.persistSelections()
			m.saved = m.saveErr == nil
		} else {
			m.aborted = true
		}
		m.quitting = true
		return m, tea.Quit
	}

	if m.stepIdx+1 < len(m.steps) {
		m.stepIdx++
		m.syncCursorToCurrentValue()
	}
	return m, nil
}

func (m *Model) persistSelections() error {
	m.cfg.DefaultCommitBackend = m.selectedBackend
	m.cfg.LLM.DefaultBackend = m.selectedBackend
	m.cfg.LLM.DefaultEffort = m.selectedEffort
	m.cfg.Commit.DefaultBackend = m.selectedBackend
	m.cfg.Commit.Effort = m.selectedEffort
	m.cfg.Commit.DefaultTemplate = m.selectedCommit
	m.cfg.PR.DefaultBackend = m.selectedBackend
	m.cfg.PR.Effort = m.selectedEffort
	m.cfg.PR.DefaultTemplate = m.selectedPR

	switch m.selectedAutoWake {
	case "enabled_resuspend":
		m.cfg.Daemon.AutoWake.Enabled = true
		m.cfg.Daemon.AutoWake.AutoResuspend = true
	case "enabled_awake":
		m.cfg.Daemon.AutoWake.Enabled = true
		m.cfg.Daemon.AutoWake.AutoResuspend = false
	default:
		m.cfg.Daemon.AutoWake.Enabled = false
	}

	if existing, err := os.ReadFile(m.configPath); err == nil {
		_ = os.WriteFile(m.configPath+".bak", existing, 0o600)
	}
	return config.SaveConfig(m.configPath, m.cfg)
}

func (m *Model) syncCursorToCurrentValue() {
	var target string
	switch m.stepIdx {
	case 0:
		target = m.selectedBackend
	case 1:
		target = m.selectedEffort
	case 2:
		target = m.selectedCommit
	case 3:
		target = m.selectedPR
	case 4:
		target = m.selectedAutoWake
	case 5:
		target = "save"
	}
	m.cursor = 0
	for i, c := range m.steps[m.stepIdx].Choices {
		if c.Value == target {
			m.cursor = i
			break
		}
	}
	m.scrollOffset = 0
	m.clampScroll()
}

func (m *Model) clampScroll() {
	visible := m.visibleOptionSlots()
	if visible <= 0 {
		visible = 1
	}
	if m.cursor < m.scrollOffset {
		m.scrollOffset = m.cursor
	}
	if m.cursor >= m.scrollOffset+visible {
		m.scrollOffset = m.cursor - visible + 1
	}
	if m.scrollOffset < 0 {
		m.scrollOffset = 0
	}
}

func (m Model) visibleOptionSlots() int {
	const (
		titleLines  = 2
		statusLines = 1
	)
	var inner int
	if m.isVerticalStack() {
		inner, _ = m.verticalInnerHeights(titleLines, statusLines)
	} else {
		inner = max(5, m.height-titleLines-statusLines-2)
	}
	// Reserve 2 rows inside panel for header and help summary.
	return max(1, inner-2)
}

func (m Model) isVerticalStack() bool {
	return m.width < wizardVerticalBreakpoint
}

// Golden Rule #4: Weight-based proportional sizing (3:2 default split).
func (m Model) horizontalWidths() (leftW, rightW int) {
	avail := max(44, m.width)
	lw, rw := 3, 2
	if m.focusedPane == "right" {
		lw, rw = 2, 3
	}
	leftW = (avail * lw) / (lw + rw)
	rightW = avail - leftW
	return leftW, rightW
}

// Golden Rule #1: Subtract 4 border lines across two vertically stacked bordered panels.
func (m Model) verticalInnerHeights(titleLines, statusLines int) (topInner, botInner int) {
	avail := max(6, m.height-titleLines-statusLines-4)
	topInner = max(3, (avail*3)/5)
	botInner = max(2, avail-topInner)
	return topInner, botInner
}

// View renders the dual-pane wizard layout in Bubble Tea v2.
func (m Model) View() tea.View {
	if m.quitting {
		return tea.NewView("")
	}

	totalWidth := max(44, m.width)
	const (
		titleLines  = 2
		statusLines = 1
	)

	step := m.steps[m.stepIdx]
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7aa2f7"))
	subStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#a9b1d6"))
	footerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#9ece6a"))

	h1 := ansi.Truncate(titleStyle.Render("aitools · Interactive Setup & Configuration Wizard"), totalWidth, "…")
	h2 := ansi.Truncate(subStyle.Render(step.HelpText), totalWidth, "…")
	footer := ansi.Truncate(
		footerStyle.Render("Keys: [j/k] Navigate  [enter] Select  [backspace] Previous Step  [tab] Focus  [q] Cancel"),
		totalWidth,
		"…",
	)

	var body string
	if m.isVerticalStack() {
		topInner, botInner := m.verticalInnerHeights(titleLines, statusLines)
		topPanel := renderWizardPanel(step.Title, m.buildLeftLines(topInner-1), totalWidth, topInner, m.focusedPane == "left")
		botPanel := renderWizardPanel("Live Configuration Summary", m.buildRightLines(), totalWidth, botInner, m.focusedPane == "right")
		body = lipgloss.JoinVertical(lipgloss.Left, topPanel, botPanel)
	} else {
		// Golden Rule #1: Account for top + bottom borders (-2) before rendering panels.
		innerHeight := max(5, m.height-titleLines-statusLines-2)
		leftW, rightW := m.horizontalWidths()
		leftPanel := renderWizardPanel(step.Title, m.buildLeftLines(innerHeight-1), leftW, innerHeight, m.focusedPane == "left")
		rightPanel := renderWizardPanel("Live Configuration Summary", m.buildRightLines(), rightW, innerHeight, m.focusedPane == "right")
		body = lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, rightPanel)
	}

	v := tea.NewView(lipgloss.JoinVertical(lipgloss.Left, h1, h2, body, footer))
	v.AltScreen = true
	return v
}

func (m Model) buildLeftLines(maxRows int) []string {
	choices := m.steps[m.stepIdx].Choices
	if maxRows <= 0 {
		maxRows = 1
	}
	slots := max(1, maxRows)
	end := min(len(choices), m.scrollOffset+slots)

	lines := make([]string, 0, slots)
	for i := m.scrollOffset; i < end; i++ {
		c := choices[i]
		cursor := "  "
		if i == m.cursor {
			cursor = "> "
		}
		lines = append(lines, fmt.Sprintf("%s%s — %s", cursor, c.Label, c.Description))
	}
	return lines
}

func (m Model) buildRightLines() []string {
	return []string{
		fmt.Sprintf("Target File:     %s", m.configPath),
		"",
		fmt.Sprintf("Default Harness: %s", m.selectedBackend),
		fmt.Sprintf("Effort Level:    %s", m.selectedEffort),
		fmt.Sprintf("Commit Template: %s", m.selectedCommit),
		fmt.Sprintf("PR Template:     %s", m.selectedPR),
		fmt.Sprintf("RTC Auto-Wake:   %s", m.selectedAutoWake),
	}
}

// Golden Rules #1 & #2: Exact line count padding (no .Height() on bordered style)
// and explicit ansi.Truncate to panelWidth-4 (-2 border, -2 padding).
func renderWizardPanel(title string, rawLines []string, panelWidth, innerHeight int, focused bool) string {
	maxTextWidth := max(4, panelWidth-4)
	lines := make([]string, 0, innerHeight)

	hdr := ansi.Truncate("["+title+"]", maxTextWidth, "…")
	lines = append(lines, padVisible(hdr, maxTextWidth))

	for i := 0; i < len(rawLines) && len(lines) < innerHeight; i++ {
		trunc := ansi.Truncate(rawLines[i], maxTextWidth, "…")
		lines = append(lines, padVisible(trunc, maxTextWidth))
	}
	for len(lines) < innerHeight {
		lines = append(lines, strings.Repeat(" ", maxTextWidth))
	}

	borderCol := lipgloss.Color("#565f89")
	if focused {
		borderCol = lipgloss.Color("#7aa2f7")
	}
	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderCol).
		Padding(0, 1)

	return style.Render(strings.Join(lines, "\n"))
}

func padVisible(s string, width int) string {
	w := ansi.StringWidth(s)
	if w >= width {
		return s
	}
	return s + strings.Repeat(" ", width-w)
}

// Run launches the interactive Bubble Tea v2 setup wizard when running in an interactive terminal.
func Run(ctx context.Context, out io.Writer, cfg *config.Config, configPath string, reg *harness.Registry, isTTY bool) error {
	if !isTTY {
		return errors.New("interactive wizard requires a TTY terminal (use 'config init' in non-interactive pipelines)")
	}
	m := NewModel(cfg, configPath, reg)
	p := tea.NewProgram(m, tea.WithContext(ctx), tea.WithOutput(out))
	finalModel, err := p.Run()
	if err != nil {
		return err
	}
	wm, ok := finalModel.(Model)
	if !ok {
		return nil
	}
	if wm.SaveError() != nil {
		return wm.SaveError()
	}
	if wm.Saved() {
		fmt.Fprintf(out, "Configuration saved to %s\n", configPath)
	}
	return nil
}
