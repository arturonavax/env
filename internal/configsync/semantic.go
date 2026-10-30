package configsync

import (
	"encoding/json"
	"path/filepath"
	"reflect"
	"strings"

	toml "github.com/pelletier/go-toml/v2"
)

// StripJSONCComments removes single-line (//) and multi-line (/* */) comments from JSONC,
// correctly ignoring comment markers inside quoted strings.
// Operates on UTF-8 bytes with zero intermediate slice allocations.
func StripJSONCComments(src string) string {
	// Fast-path: return immediately if no comment indicators exist
	if !strings.Contains(src, "//") && !strings.Contains(src, "/*") {
		return src
	}

	var sb strings.Builder
	sb.Grow(len(src))

	inString := false
	escaped := false
	inSingleLineComment := false
	inMultiLineComment := false
	n := len(src)

	for i := 0; i < n; i++ {
		b := src[i]

		if inSingleLineComment {
			if b == '\n' {
				inSingleLineComment = false
				sb.WriteByte(b)
			}
			continue
		}

		if inMultiLineComment {
			if b == '*' && i+1 < n && src[i+1] == '/' {
				inMultiLineComment = false
				i++ // skip '/'
			}
			continue
		}

		if inString {
			sb.WriteByte(b)
			if escaped {
				escaped = false
			} else if b == '\\' {
				escaped = true
			} else if b == '"' {
				inString = false
			}
			continue
		}

		// Outside comments and strings
		if b == '"' {
			inString = true
			sb.WriteByte(b)
			continue
		}

		if b == '/' && i+1 < n {
			if src[i+1] == '/' {
				inSingleLineComment = true
				i++
				continue
			}
			if src[i+1] == '*' {
				inMultiLineComment = true
				i++
				continue
			}
		}

		sb.WriteByte(b)
	}

	return sb.String()
}

// IsSemanticEqual checks if two files are semantically equivalent based on their file extension (.json, .jsonc, .toml).
// Returns (true, nil) if both contents represent the exact same data structure.
// Returns (false, nil) if they differ semantically.
// Returns (false, err) if either content is unparseable according to its format.
func IsSemanticEqual(path, a, b string) (bool, error) {
	ext := strings.ToLower(filepath.Ext(path))

	switch ext {
	case ".json", ".jsonc":
		cleanA := StripJSONCComments(a)
		cleanB := StripJSONCComments(b)

		var valA, valB any
		if err := json.Unmarshal([]byte(cleanA), &valA); err != nil {
			return false, err
		}
		if err := json.Unmarshal([]byte(cleanB), &valB); err != nil {
			return false, err
		}
		return reflect.DeepEqual(valA, valB), nil

	case ".toml":
		var valA, valB map[string]any
		if err := toml.Unmarshal([]byte(a), &valA); err != nil {
			return false, err
		}
		if err := toml.Unmarshal([]byte(b), &valB); err != nil {
			return false, err
		}
		return reflect.DeepEqual(valA, valB), nil

	default:
		// Plain text comparison
		return strings.TrimSpace(a) == strings.TrimSpace(b), nil
	}
}
