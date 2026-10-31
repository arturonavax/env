package security

import (
	"regexp"
	"strings"
)

type SecretFinding struct {
	RuleID      string `json:"rule_id"`
	Description string `json:"description"`
	Line        int    `json:"line"`
}

var secretPatterns = []struct {
	ruleID string
	desc   string
	regex  *regexp.Regexp
}{
	{
		ruleID: "SEC-PRIVKEY",
		desc:   "Private encryption key header",
		regex:  regexp.MustCompile(`(?i)-----BEGIN\s+([A-Z0-9\s_-]+)?PRIVATE\s+KEY-----`),
	},
	{
		ruleID: "SEC-AWSKEY",
		desc:   "AWS access key ID",
		regex:  regexp.MustCompile(`\b(AKIA|ABIA|ACCA|ASIA)[0-9A-Z]{16}\b`),
	},
	{
		ruleID: "SEC-GHTOKEN",
		desc:   "GitHub personal access / OAuth token",
		regex:  regexp.MustCompile(`\bgh[pousr]_[A-Za-z0-9_]{36,}\b`),
	},
	{
		ruleID: "SEC-SLACKTOKEN",
		desc:   "Slack API token",
		regex:  regexp.MustCompile(`\bxox[baprs]-[0-9A-Za-z]{10,}\b`),
	},
	{
		ruleID: "SEC-JWT",
		desc:   "JSON Web Token (JWT) secret token",
		regex:  regexp.MustCompile(`\beyJ[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}\b`),
	},
	{
		ruleID: "SEC-GENERIC",
		desc:   "Explicit secret or API key assignment",
		regex:  regexp.MustCompile(`(?i)\b(api[_-]?key|access[_-]?token|secret[_-]?key|auth[_-]?token)\b[\s]*[=:][\s]*["'][A-Za-z0-9_\-+=/]{16,}["']`),
	},
}

// ScanForSecrets inspects text content for known credentials, private keys, or API tokens.
// Returns a slice of findings. If empty, the content is clean.
func ScanForSecrets(content string) []SecretFinding {
	var findings []SecretFinding
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "//") {
			// Skip pure comment lines unless they match a private key header
			if !strings.Contains(trimmed, "PRIVATE KEY") {
				continue
			}
		}
		for _, sp := range secretPatterns {
			if sp.regex.MatchString(line) {
				findings = append(findings, SecretFinding{
					RuleID:      sp.ruleID,
					Description: sp.desc,
					Line:        i + 1,
				})
			}
		}
	}
	return findings
}
