package tui

import (
	"strings"

	"github.com/charmbracelet/x/ansi"

	"github.com/arturonavax/env/internal/theme"
)

// AnsiTruncate safely truncates text to maxLen respecting ANSI escapes (Golden Rule #2).
func AnsiTruncate(s string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}
	return ansi.Truncate(s, maxLen, "…")
}

// BuildPanel constructs a bordered panel adhering strictly to Golden Rules #1 & #2:
// - Never sets .Height() on a bordered Lipgloss style
// - Fills lines to exact innerHeight so panel height is deterministic
// - Truncates each line to maxWidth - 4 (2 border, 2 padding) to prevent wrapping
func BuildPanel(styles *theme.Styles, lines []string, width, height int, active bool) string {
	if width < 10 {
		width = 10
	}
	if height < 3 {
		height = 3
	}

	// Golden Rule #1: borders take 2 lines (top and bottom)
	innerHeight := height - 2
	maxTextWidth := width - 4
	if maxTextWidth < 1 {
		maxTextWidth = 1
	}

	paddedLines := make([]string, 0, innerHeight)
	for i := 0; i < innerHeight; i++ {
		if i < len(lines) {
			paddedLines = append(paddedLines, AnsiTruncate(lines[i], maxTextWidth))
		} else {
			paddedLines = append(paddedLines, "")
		}
	}

	content := strings.Join(paddedLines, "\n")
	style := styles.BorderInactive.Width(width)
	if active {
		style = styles.BorderActive.Width(width)
	}

	return style.Render(content)
}

// JoinHorizontal renders two vertical panels side-by-side with a clean gap.
func JoinHorizontal(left, right string, gap int) string {
	lLines := strings.Split(left, "\n")
	rLines := strings.Split(right, "\n")

	maxLines := len(lLines)
	if len(rLines) > maxLines {
		maxLines = len(rLines)
	}

	gapStr := strings.Repeat(" ", gap)
	var sb strings.Builder
	for i := 0; i < maxLines; i++ {
		var l, r string
		if i < len(lLines) {
			l = lLines[i]
		}
		if i < len(rLines) {
			r = rLines[i]
		}
		sb.WriteString(l)
		sb.WriteString(gapStr)
		sb.WriteString(r)
		if i < maxLines-1 {
			sb.WriteString("\n")
		}
	}

	return sb.String()
}
