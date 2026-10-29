package configsync

import (
	"bufio"
	"strings"
)

const (
	DefaultMarkerStart = "# >>> arturonavax/env >>>"
	DefaultMarkerEnd   = "# <<< arturonavax/env <<<"
)

// SmartAppend updates or appends a marked block within targetContent, preserving surrounding user lines.
func SmartAppend(targetContent, newBlock, markerStart, markerEnd string) string {
	if markerStart == "" {
		markerStart = DefaultMarkerStart
	}
	if markerEnd == "" {
		markerEnd = DefaultMarkerEnd
	}

	lines := make([]string, 0, 64)
	scanner := bufio.NewScanner(strings.NewReader(targetContent))
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	startIdx := -1
	endIdx := -1

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == markerStart {
			startIdx = i
		} else if trimmed == markerEnd && startIdx != -1 {
			endIdx = i
			break
		}
	}

	blockLines := []string{markerStart}
	bScanner := bufio.NewScanner(strings.NewReader(strings.TrimSpace(newBlock)))
	for bScanner.Scan() {
		blockLines = append(blockLines, bScanner.Text())
	}
	blockLines = append(blockLines, markerEnd)

	// If block not found, append to end.
	if startIdx == -1 || endIdx == -1 {
		result := make([]string, 0, len(lines)+len(blockLines)+2)
		result = append(result, lines...)
		if len(result) > 0 && result[len(result)-1] != "" {
			result = append(result, "")
		}
		result = append(result, blockLines...)
		return strings.Join(result, "\n") + "\n"
	}

	// If block found, replace lines from startIdx to endIdx.
	result := make([]string, 0, len(lines)+len(blockLines))
	result = append(result, lines[:startIdx]...)
	result = append(result, blockLines...)
	if endIdx+1 < len(lines) {
		result = append(result, lines[endIdx+1:]...)
	}

	return strings.Join(result, "\n") + "\n"
}

// ExtractMarkerBlock extracts the inner lines strictly between markerStart and markerEnd without personal lines outside markers.
func ExtractMarkerBlock(content, markerStart, markerEnd string) (string, bool) {
	if markerStart == "" {
		markerStart = DefaultMarkerStart
	}
	if markerEnd == "" {
		markerEnd = DefaultMarkerEnd
	}

	lines, _ := splitLogicalLines(content)
	startIdx := -1
	endIdx := -1

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == markerStart && startIdx == -1 {
			startIdx = i
		} else if trimmed == markerEnd && startIdx != -1 {
			endIdx = i
			break
		}
	}

	if startIdx == -1 || endIdx == -1 || endIdx < startIdx+1 {
		return "", false
	}

	inner := lines[startIdx+1 : endIdx]
	if len(inner) == 0 {
		return "", true
	}
	return strings.Join(inner, "\n") + "\n", true
}
