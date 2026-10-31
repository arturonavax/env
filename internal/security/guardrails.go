package security

import (
	"regexp"
	"strings"
)

// GuardrailViolation describes a blocked destructive or unsafe pattern inside a generated or custom shell script.
type GuardrailViolation struct {
	RuleID      string `json:"rule_id"`
	Description string `json:"description"`
	Line        int    `json:"line"`
	Snippet     string `json:"snippet"`
}

type guardrailRule struct {
	id          string
	description string
	pattern     *regexp.Regexp
}

// Compiled once at package level per golang-design-patterns (ported and expanded from utils/optimize_system.sh).
var scriptGuardrailRules = []guardrailRule{
	{
		id:          "GR-001",
		description: "recursive deletion targeting root or top-level system directories",
		pattern:     regexp.MustCompile(`(?i)\brm\s+(?:[^\n;|&]*\s+)?-(?:[a-zA-Z]*[rf][a-zA-Z]*|-recursive|-force)(?:\s+[^\n;|&]*)?\s+(/|/\*|/boot|/etc|/usr|/var|/home|--no-preserve-root)(?:\s|$)`),
	},
	{
		id:          "GR-002",
		description: "filesystem formatting command (mkfs/wipefs)",
		pattern:     regexp.MustCompile(`(?i)\b(mkfs(\.[a-z0-9]+)?|wipefs)\b`),
	},
	{
		id:          "GR-003",
		description: "raw block device overwrite via dd",
		pattern:     regexp.MustCompile(`(?i)\bdd\b[^\n]*\bof=/dev/(sd[a-z]|nvme\d+n\d+|vd[a-z]|hd[a-z]|mmcblk\d+|disk\d+)`),
	},
	{
		id:          "GR-004",
		description: "shell fork bomb signature",
		pattern:     regexp.MustCompile(`:\(\)\s*\{\s*:\s*\|\s*:\s*&\s*\}\s*;\s*:`),
	},
	{
		id:          "GR-005",
		description: "world-writable chmod 777 on system directories",
		pattern:     regexp.MustCompile(`(?i)\bchmod\s+(-R\s+)?0?777\s+(/|/etc|/usr|/bin|/sbin|/boot|/var)(?:\s|/|$)`),
	},
	{
		id:          "GR-006",
		description: "recursive chown across root or system hierarchies",
		pattern:     regexp.MustCompile(`(?i)\bchown\s+-R\s+[^\s]+\s+(/|/etc|/usr|/var|/boot)\s*$`),
	},
	{
		id:          "GR-007",
		description: "piping remote network content directly into privileged sudo shell",
		pattern:     regexp.MustCompile(`(?i)\b(curl|wget)\b[^\n|]*\|\s*sudo\s+(ba)?sh\b`),
	},
	{
		id:          "GR-008",
		description: "partition table mutation tool (fdisk/parted/sgdisk --zap-all)",
		pattern:     regexp.MustCompile(`(?i)\b(sgdisk\s+--zap-all|parted\s+-s\s+/dev/|sfdisk\s+--delete)\b`),
	},
	{
		id:          "GR-009",
		description: "disabling mandatory access control or host firewall globally",
		pattern:     regexp.MustCompile(`(?i)\b(setenforce\s+0|systemctl\s+(disable|mask)\s+(apparmor|ufw|firewalld))\b`),
	},
	{
		id:          "GR-010",
		description: "direct overwrite or redirection into /etc/passwd, /etc/shadow, or /etc/sudoers",
		pattern:     regexp.MustCompile(`(?i)>\s*/etc/(passwd|shadow|sudoers)\b`),
	},
	{
		id:          "GR-011",
		description: "raw redirection into physical block device node",
		pattern:     regexp.MustCompile(`(?i)>\s*/dev/(sd[a-z]|nvme\d+n\d+|vd[a-z])\b`),
	},
	{
		id:          "GR-012",
		description: "kernel panic or immediate ungraceful sysrq trigger",
		pattern:     regexp.MustCompile(`(?i)/proc/sysrq-trigger`),
	},
}

// ValidateScriptGuardrails scans a shell script line-by-line and returns any detected safety violations.
func ValidateScriptGuardrails(script string) []GuardrailViolation {
	violations := []GuardrailViolation{}
	lines := strings.Split(script, "\n")

	for idx, rawLine := range lines {
		trimmed := strings.TrimSpace(rawLine)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		for _, rule := range scriptGuardrailRules {
			if rule.pattern.MatchString(trimmed) {
				violations = append(violations, GuardrailViolation{
					RuleID:      rule.id,
					Description: rule.description,
					Line:        idx + 1,
					Snippet:     trimmed,
				})
			}
		}
	}

	return violations
}
