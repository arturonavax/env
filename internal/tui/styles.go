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

// AdjustScrollOffset ensures the cursor remains visible within a window of visibleLines,
// clamping and updating the viewport scroll offset smoothly without sudden jumps.
func AdjustScrollOffset(cursor, currentOffset, visibleLines, totalItems int) int {
	if visibleLines <= 0 || totalItems <= 0 {
		return 0
	}
	if cursor < 0 {
		cursor = 0
	}
	if cursor >= totalItems {
		cursor = totalItems - 1
	}

	offset := currentOffset
	if cursor < offset {
		offset = cursor
	} else if cursor >= offset+visibleLines {
		offset = cursor - visibleLines + 1
	}

	maxOffset := totalItems - visibleLines
	if maxOffset < 0 {
		maxOffset = 0
	}
	if offset > maxOffset {
		offset = maxOffset
	}
	if offset < 0 {
		offset = 0
	}
	return offset
}

// BuildScrollablePanel constructs a bordered panel with fixed header lines and a scrollable window
// of item lines, ensuring the cursor index is always within view.
// It adheres strictly to Golden Rules #1 & #2:
// - Never sets .Height() on a bordered Lipgloss style
// - Fills lines to exact innerHeight so panel height is deterministic
// - Truncates each line to maxWidth - 4 (2 border, 2 padding) to prevent wrapping
func BuildScrollablePanel(styles *theme.Styles, headerLines, itemLines []string, cursor, offset, width, height int, active bool) string {
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

	availItemLines := innerHeight - len(headerLines)
	if availItemLines < 1 {
		availItemLines = 1
	}

	effectiveOffset := AdjustScrollOffset(cursor, offset, availItemLines, len(itemLines))
	start := effectiveOffset
	end := start + availItemLines
	if end > len(itemLines) {
		end = len(itemLines)
	}

	var visibleItems []string
	if start < len(itemLines) {
		visibleItems = itemLines[start:end]
	}

	paddedLines := make([]string, 0, innerHeight)
	for _, hLine := range headerLines {
		if len(paddedLines) < innerHeight {
			paddedLines = append(paddedLines, AnsiTruncate(hLine, maxTextWidth))
		}
	}
	for _, it := range visibleItems {
		if len(paddedLines) < innerHeight {
			paddedLines = append(paddedLines, AnsiTruncate(it, maxTextWidth))
		}
	}
	for len(paddedLines) < innerHeight {
		paddedLines = append(paddedLines, "")
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
