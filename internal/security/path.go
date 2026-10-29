package security

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
)

var (
	ErrPathTraversal = errors.New("path traversal detected: path escapes allowed directory")
	ErrInvalidPath   = errors.New("invalid path")
)

// SafePath ensures that target is strictly contained within baseDir after resolution.
func SafePath(baseDir, target string) (string, error) {
	if baseDir == "" || target == "" {
		return "", ErrInvalidPath
	}

	cleanBase := filepath.Clean(baseDir)
	cleanTarget := filepath.Clean(target)

	// If target is relative, join it to base
	if !filepath.IsAbs(cleanTarget) {
		cleanTarget = filepath.Join(cleanBase, cleanTarget)
	}

	// Canonical relative check
	rel, err := filepath.Rel(cleanBase, cleanTarget)
	if err != nil {
		return "", fmt.Errorf("evaluating relative path: %w", err)
	}

	if strings.HasPrefix(rel, "..") || rel == "." && cleanBase != cleanTarget {
		return "", fmt.Errorf("%w: %s is outside %s", ErrPathTraversal, cleanTarget, cleanBase)
	}

	return cleanTarget, nil
}

// RedactSensitives masks tokens and secret values in output or log strings.
func RedactSensitives(input string) string {
	sensitiveKeys := []string{"token", "secret", "password", "key", "auth"}
	lines := strings.Split(input, "\n")
	redactedLines := make([]string, 0, len(lines))

	for _, line := range lines {
		lower := strings.ToLower(line)
		hasSensitive := false
		for _, key := range sensitiveKeys {
			if strings.Contains(lower, key+"=") || strings.Contains(lower, key+":") {
				hasSensitive = true
				break
			}
		}
		if hasSensitive {
			redactedLines = append(redactedLines, "[REDACTED SENSITIVE DATA]")
		} else {
			redactedLines = append(redactedLines, line)
		}
	}

	return strings.Join(redactedLines, "\n")
}
