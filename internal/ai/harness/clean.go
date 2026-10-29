package harness

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Compiled once at package level per golang-design-patterns.
var (
	ansiOSCRegex = regexp.MustCompile(`\x1b\][^\x07\x1b]*(?:\x07|\x1b\\)`)
	ansiCSIRegex = regexp.MustCompile(`\x1b(?:[@-Z\\-_]|\[[0-?]*[ -/]*[@-~])`)
)

// StripANSI removes ANSI OSC, CSI, and Fe escape sequences and carriage returns from text.
func StripANSI(s string) string {
	if s == "" {
		return ""
	}
	clean := ansiOSCRegex.ReplaceAllString(s, "")
	clean = ansiCSIRegex.ReplaceAllString(clean, "")
	clean = strings.ReplaceAll(clean, "\r", "")
	return clean
}

// StripMarkdownFences removes a single wrapping outer Markdown triple-backtick code fence
// (e.g. ```markdown\n...\n```) and wrapping outer quotes without touching interior code blocks.
func StripMarkdownFences(s string) string {
	trimmed := strings.TrimSpace(s)
	if strings.HasPrefix(trimmed, "```") && strings.HasSuffix(trimmed, "```") {
		firstNewline := strings.IndexByte(trimmed, '\n')
		if firstNewline != -1 {
			inner := trimmed[firstNewline+1:]
			if strings.HasSuffix(inner, "```") {
				body := strings.TrimSuffix(inner, "```")
				// Only strip if there is no intermediate closing/opening triple fence at line start.
				if !containsStandaloneFence(body) {
					trimmed = strings.TrimSpace(body)
				}
			}
		}
	}

	if len(trimmed) >= 2 && strings.HasPrefix(trimmed, `"`) && strings.HasSuffix(trimmed, `"`) && !strings.Contains(trimmed[1:len(trimmed)-1], "\n") {
		trimmed = strings.TrimSpace(trimmed[1 : len(trimmed)-1])
	}
	return trimmed
}

func containsStandaloneFence(s string) bool {
	for _, line := range strings.Split(strings.TrimSpace(s), "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			return true
		}
	}
	return false
}

// ExtractJSONPath traverses a JSON document using dot notation (e.g. "choices.0.message.content")
// supporting both object map keys and slice integer indices.
func ExtractJSONPath(raw []byte, dotPath string) (string, error) {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" {
		return "", fmt.Errorf("empty JSON payload")
	}

	var root any
	if err := json.Unmarshal([]byte(trimmed), &root); err != nil {
		return "", fmt.Errorf("decoding JSON output: %w", err)
	}

	if dotPath == "" {
		return stringifyJSONValue(root)
	}

	curr := root
	for _, part := range strings.Split(dotPath, ".") {
		if curr == nil {
			return "", fmt.Errorf("JSON path %q encountered null at %q", dotPath, part)
		}
		switch node := curr.(type) {
		case map[string]any:
			val, exists := node[part]
			if !exists {
				return "", fmt.Errorf("JSON key %q not found in path %q", part, dotPath)
			}
			curr = val
		case []any:
			idx, err := strconv.Atoi(part)
			if err != nil || idx < 0 || idx >= len(node) {
				return "", fmt.Errorf("JSON array index %q out of bounds in path %q", part, dotPath)
			}
			curr = node[idx]
		default:
			return "", fmt.Errorf("cannot traverse %T at segment %q in path %q", curr, part, dotPath)
		}
	}

	return stringifyJSONValue(curr)
}

func stringifyJSONValue(v any) (string, error) {
	switch typed := v.(type) {
	case nil:
		return "", nil
	case string:
		return typed, nil
	case float64, bool:
		return fmt.Sprintf("%v", typed), nil
	default:
		b, err := json.Marshal(typed)
		if err != nil {
			return "", err
		}
		return string(b), nil
	}
}
