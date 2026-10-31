package operation

import (
	"strings"
)

// EscapePOSIX returns a safely quoted argument for POSIX sh and Bash.
// It uses standard single-quote escaping: enclosing the argument in single quotes
// and escaping every literal single quote as '\”.
//
// Arguments consisting solely of unreserved safe characters [a-zA-Z0-9_./-] are returned bare.
func EscapePOSIX(s string) string {
	if s == "" {
		return "''"
	}

	if isSafePOSIX(s) {
		return s
	}

	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

func isSafePOSIX(s string) bool {
	for i := 0; i < len(s); i++ {
		c := s[i]
		if (c >= 'a' && c <= 'z') ||
			(c >= 'A' && c <= 'Z') ||
			(c >= '0' && c <= '9') ||
			c == '_' || c == '.' || c == '/' || c == '-' || c == '=' || c == ':' || c == '@' {
			continue
		}
		return false
	}
	return true
}

// EscapePowerShell returns a safely quoted argument for PowerShell.
// It encloses the string in single quotes and escapes single quotes as ”.
func EscapePowerShell(s string) string {
	if s == "" {
		return "''"
	}

	if isSafePOSIX(s) {
		return s
	}

	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}
