package quota

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

const verticalBreakpointCols = 88

type tickMsg time.Time

type evalSnapshotMsg struct {
	evals []TargetEvaluation
	err   error
	at    time.Time
}

type wakeFinishedMsg struct {
	report *WakeReport
	err    error
}

// WatchModel implements a Bubble Tea v2 live quota dashboard adhering to the 4 Golden Rules.
type WatchModel struct {
	ctx          context.Context
	svc          *Service
	interval     time.Duration
	targetFilter string

	width        int
	height       int
	focusedPanel string // "left" or "right"
	cursor       int
	evals        []TargetEvaluation
	lastUpdated  time.Time
	statusNote   string
	quitting     bool
}

// NewWatchModel constructs an initialized WatchModel for Bubble Tea v2 or direct Update() testing.
func NewWatchModel(ctx context.Context, svc *Service, interval time.Duration, targetFilter string) WatchModel {
	if interval <= 0 {
		interval = 30 * time.Second
	}
	return WatchModel{
		ctx:          ctx,
		svc:          svc,
		interval:     interval,
		targetFilter: targetFilter,
		width:        100,
		height:       26,
		focusedPanel: "left",
		evals:        []TargetEvaluation{},
		statusNote:   "Loading quota telemetry...",
	}
}

// Init kicks off initial quota evaluation and periodic refresh ticking.
func (m WatchModel) Init() tea.Cmd {
	return tea.Batch(m.fetchCmd(), m.tickCmd())
}

func (m WatchModel) fetchCmd() tea.Cmd {
	return func() tea.Msg {
		evals, err := m.svc.EvaluateAll(m.ctx, m.targetFilter)
		return evalSnapshotMsg{
			evals: evals,
			err:   err,
			at:    time.Now(),
		}
	}
}

func (m WatchModel) tickCmd() tea.Cmd {
	return tea.Tick(m.interval, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m WatchModel) wakeSelectedCmd(targetID string) tea.Cmd {
	return func() tea.Msg {
		rep, err := m.svc.Wake(m.ctx, true, targetID)
		return wakeFinishedMsg{report: rep, err: err}
	}
}

// Update processes window resizes, keyboard actions, mouse clicks, and periodic telemetry ticks.
func (m WatchModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch typed := msg.(type) {
	case tea.WindowSizeMsg:
		if typed.Width > 0 {
			m.width = typed.Width
		}
		if typed.Height > 0 {
			m.height = typed.Height
		}
		return m, nil

	case tickMsg:
		return m, tea.Batch(m.fetchCmd(), m.tickCmd())

	case evalSnapshotMsg:
		m.lastUpdated = typed.at
		if typed.err != nil {
			m.statusNote = fmt.Sprintf("Refresh error: %v", typed.err)
			return m, nil
		}
		m.evals = typed.evals
		if m.cursor >= len(m.evals) {
			m.cursor = max(0, len(m.evals)-1)
		}
		m.statusNote = fmt.Sprintf("Refreshed at %s (interval %s)", typed.at.Format("15:04:05"), m.interval)
		return m, nil

	case wakeFinishedMsg:
		if typed.err != nil {
			m.statusNote = fmt.Sprintf("Wake probe failed: %v", typed.err)
		} else {
			m.statusNote = fmt.Sprintf("Wake probe succeeded (%d woken)", len(typed.report.Woken))
		}
		return m, m.fetchCmd()

	case tea.KeyPressMsg:
		return m.handleKey(typed)

	case tea.MouseClickMsg:
		return m.handleMouseClick(typed)
	}

	return m, nil
}

func (m WatchModel) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c", "esc":
		m.quitting = true
		return m, tea.Quit
	case "tab":
		if m.focusedPanel == "left" {
			m.focusedPanel = "right"
		} else {
			m.focusedPanel = "left"
		}
	case "j", "down":
		if m.cursor+1 < len(m.evals) {
			m.cursor++
		}
	case "k", "up":
		if m.cursor > 0 {
			m.cursor--
		}
	case "r":
		m.statusNote = "Refreshing quota telemetry..."
		return m, m.fetchCmd()
	case "w":
		if m.cursor >= 0 && m.cursor < len(m.evals) {
			id := m.evals[m.cursor].ID
			m.statusNote = fmt.Sprintf("Sending wakeup probe to %s...", id)
			return m, m.wakeSelectedCmd(id)
		}
	}
	return m, nil
}

// Golden Rule #3: Match mouse detection coordinates to active layout orientation.
func (m WatchModel) handleMouseClick(msg tea.MouseClickMsg) (tea.Model, tea.Cmd) {
	mouse := msg.Mouse()
	const titleLines = 2
	if m.isVerticalStack() {
		topInner, _ := m.verticalInnerHeights(titleLines, 1)
		topTotal := topInner + 2 // +2 for top panel borders
		relY := mouse.Y - titleLines
		if relY < topTotal {
			m.focusedPanel = "left"
		} else {
			m.focusedPanel = "right"
		}
		return m, nil
	}

	leftWidth, _ := m.horizontalWidths()
	if mouse.X < leftWidth {
		m.focusedPanel = "left"
	} else {
		m.focusedPanel = "right"
	}
	return m, nil
}

func (m WatchModel) isVerticalStack() bool {
	return m.width < verticalBreakpointCols
}

// Golden Rule #4: Weight-based proportional horizontal sizing.
func (m WatchModel) horizontalWidths() (leftWidth, rightWidth int) {
	avail := max(40, m.width)
	leftWeight, rightWeight := 2, 1
	if m.focusedPanel == "right" {
		leftWeight, rightWeight = 1, 2
	}
	totalWeight := leftWeight + rightWeight
	leftWidth = (avail * leftWeight) / totalWeight
	rightWidth = avail - leftWidth
	return leftWidth, rightWidth
}

// Golden Rule #1: Account for borders (-2 per bordered panel) BEFORE rendering.
func (m WatchModel) verticalInnerHeights(titleLines, statusLines int) (topInner, bottomInner int) {
	// Two stacked bordered panels consume 4 border rows total.
	availInner := max(6, m.height-titleLines-statusLines-4)
	topWeight, botWeight := 2, 1
	if m.focusedPanel == "right" {
		topWeight, botWeight = 1, 2
	}
	topInner = max(2, (availInner*topWeight)/(topWeight+botWeight))
	bottomInner = max(2, availInner-topInner)
	return topInner, bottomInner
}

// View renders the dual-pane quota dashboard into a Bubble Tea v2 tea.View.
func (m WatchModel) View() tea.View {
	if m.quitting {
		return tea.NewView("")
	}

	totalWidth := max(44, m.width)
	const (
		titleLines  = 2
		statusLines = 1
	)

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7aa2f7"))
	subStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#a9b1d6"))
	statusStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#9ece6a"))

	header1 := ansi.Truncate(titleStyle.Render("aitools watch · Live Rolling Quota Dashboard"), totalWidth, "…")
	header2 := ansi.Truncate(subStyle.Render(m.statusNote), totalWidth, "…")
	footer := ansi.Truncate(
		statusStyle.Render("Keys: [j/k] Select  [tab] Focus Pane  [r] Refresh  [w] Wake Target  [q] Quit"),
		totalWidth,
		"…",
	)

	var body string
	if m.isVerticalStack() {
		topInner, botInner := m.verticalInnerHeights(titleLines, statusLines)
		topPanel := renderBorderedPanel(
			"Quota Targets",
			m.buildTargetsLines(),
			totalWidth,
			topInner,
			m.focusedPanel == "left",
		)
		botPanel := renderBorderedPanel(
			"Target Telemetry",
			m.buildDetailLines(),
			totalWidth,
			botInner,
			m.focusedPanel == "right",
		)
		body = lipgloss.JoinVertical(lipgloss.Left, topPanel, botPanel)
	} else {
		// Golden Rule #1: Subtract titleLines, statusLines, and -2 for panel borders.
		innerHeight := max(4, m.height-titleLines-statusLines-2)
		leftW, rightW := m.horizontalWidths()
		leftPanel := renderBorderedPanel(
			"Quota Targets",
			m.buildTargetsLines(),
			leftW,
			innerHeight,
			m.focusedPanel == "left",
		)
		rightPanel := renderBorderedPanel(
			"Target Telemetry",
			m.buildDetailLines(),
			rightW,
			innerHeight,
			m.focusedPanel == "right",
		)
		body = lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, rightPanel)
	}

	full := lipgloss.JoinVertical(lipgloss.Left, header1, header2, body, footer)
	v := tea.NewView(full)
	v.AltScreen = true
	return v
}

func (m WatchModel) buildTargetsLines() []string {
	if len(m.evals) == 0 {
		return []string{"(no targets configured)"}
	}
	lines := make([]string, 0, len(m.evals)+1)
	lines = append(lines, fmt.Sprintf("  %-12s %-7s %-17s %s", "ID", "TOOL", "QUOTA", "STATE"))
	for i, e := range m.evals {
		prefix := "  "
		if i == m.cursor {
			prefix = "> "
		}
		bar := FormatProgressBar(e.RemainingPercent, 8)
		lines = append(lines, fmt.Sprintf("%s%-12s %-7s %-17s %s", prefix, e.ID, e.Tool, bar, strings.ToUpper(e.State)))
	}
	return lines
}

func (m WatchModel) buildDetailLines() []string {
	if len(m.evals) == 0 || m.cursor < 0 || m.cursor >= len(m.evals) {
		return []string{"Select a target to inspect rolling window details."}
	}
	e := m.evals[m.cursor]
	remH := e.RemainingSeconds / 3600
	remM := (e.RemainingSeconds % 3600) / 60
	return []string{
		fmt.Sprintf("Name:      %s", e.Name),
		fmt.Sprintf("ID:        %s", e.ID),
		fmt.Sprintf("Harness:   %s (%s)", e.Tool, e.Account),
		fmt.Sprintf("Telemetry: %s", e.TelemetryType),
		fmt.Sprintf("State:     %s", strings.ToUpper(e.State)),
		fmt.Sprintf("Remaining: %.1f%% (%dh %02dm)", e.RemainingPercent, remH, remM),
		fmt.Sprintf("Resets At: %s", e.ResetTimeLocal),
		fmt.Sprintf("Summary:   %s", e.StatusLabel),
	}
}

// Golden Rules #1 & #2: Fill lines to exact innerHeight (no .Height() on bordered style)
// and truncate every line to panelWidth-4 (-2 border, -2 padding) to prevent wrapping.
func renderBorderedPanel(title string, rawLines []string, panelWidth, innerHeight int, focused bool) string {
	maxTextWidth := max(4, panelWidth-4)
	lines := make([]string, 0, innerHeight)

	headerLine := ansi.Truncate("["+title+"]", maxTextWidth, "…")
	lines = append(lines, padRightVisible(headerLine, maxTextWidth))

	for i := 0; i < len(rawLines) && len(lines) < innerHeight; i++ {
		truncated := ansi.Truncate(rawLines[i], maxTextWidth, "…")
		lines = append(lines, padRightVisible(truncated, maxTextWidth))
	}
	for len(lines) < innerHeight {
		lines = append(lines, strings.Repeat(" ", maxTextWidth))
	}

	borderColor := lipgloss.Color("#565f89")
	if focused {
		borderColor = lipgloss.Color("#7dcfff")
	}

	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Padding(0, 1)

	return style.Render(strings.Join(lines, "\n"))
}

func padRightVisible(s string, targetWidth int) string {
	w := ansi.StringWidth(s)
	if w >= targetWidth {
		return s
	}
	return s + strings.Repeat(" ", targetWidth-w)
}

// RunWatch launches the interactive Bubble Tea v2 dashboard when isTTY is true,
// or writes a single formatted snapshot to out when running in non-TTY / piped environments.
func RunWatch(
	ctx context.Context,
	out io.Writer,
	svc *Service,
	interval time.Duration,
	targetFilter string,
	isTTY bool,
) error {
	if !isTTY {
		evals, err := svc.EvaluateAll(ctx, targetFilter)
		if err != nil {
			return err
		}
		_, err = fmt.Fprintln(out, FormatTable(evals))
		return err
	}

	m := NewWatchModel(ctx, svc, interval, targetFilter)
	p := tea.NewProgram(m, tea.WithContext(ctx), tea.WithOutput(out))
	_, err := p.Run()
	return err
}
