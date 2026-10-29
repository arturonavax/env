package config

// StripJSONC sanitizes JSON with Comments (JSONC) into strict RFC 8259 JSON in a single pass.
// It removes // and # single-line comments, /* ... */ block comments, and trailing commas
// before '}' or ']', while preserving string literals (including escaped quotes and URLs)
// and newline offsets for accurate error line numbers.
func StripJSONC(input []byte) []byte {
	if len(input) == 0 {
		return []byte{}
	}

	out := make([]byte, 0, len(input))
	inString := false
	escaped := false

	n := len(input)
	for i := 0; i < n; {
		c := input[i]

		if inString {
			out = append(out, c)
			switch {
			case escaped:
				escaped = false
			case c == '\\':
				escaped = true
			case c == '"':
				inString = false
			}
			i++
			continue
		}

		switch {
		case c == '"':
			inString = true
			out = append(out, c)
			i++

		case c == '/' && i+1 < n && input[i+1] == '/':
			i += 2
			for i < n && input[i] != '\n' {
				i++
			}

		case c == '#':
			i++
			for i < n && input[i] != '\n' {
				i++
			}

		case c == '/' && i+1 < n && input[i+1] == '*':
			i += 2
			for i < n {
				if input[i] == '\n' {
					out = append(out, '\n')
				}
				if i+1 < n && input[i] == '*' && input[i+1] == '/' {
					i += 2
					break
				}
				i++
			}

		case c == '}' || c == ']':
			stripTrailingComma(out)
			out = append(out, c)
			i++

		default:
			out = append(out, c)
			i++
		}
	}

	return out
}

// stripTrailingComma replaces a structural trailing comma at the end of out (ignoring trailing whitespace)
// with a space byte so slice length and newline counts remain unchanged.
func stripTrailingComma(out []byte) {
	for idx := len(out) - 1; idx >= 0; idx-- {
		switch out[idx] {
		case ' ', '\t', '\r', '\n':
			continue
		case ',':
			out[idx] = ' '
			return
		default:
			return
		}
	}
}
