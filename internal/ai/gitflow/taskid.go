package gitflow

import (
	"regexp"
	"strings"
)

// Compiled once at package level per golang-design-patterns.
var (
	explicitHashIssueRe = regexp.MustCompile(`#([0-9]{1,7})(?:$|[^0-9])`)
	prefixedIssueRe     = regexp.MustCompile(`(?i)(?:^|[/._-])(?:issue|gh|gl|pr|mr|ticket|task|feat|feature|fix|bugfix|hotfix|chore)[/-]#?([0-9]{1,7})(?:$|[/._-])`)
	leadingNumIssueRe   = regexp.MustCompile(`(?:^|/)([0-9]{1,7})[-_]`)
	jiraLinearKeyRe     = regexp.MustCompile(`(?:^|[/._-])([A-Za-z]{2,10})-([0-9]{1,7})(?:$|[/._-])`)

	nonProjectPrefixes = map[string]struct{}{
		"issue":    {},
		"gh":       {},
		"gl":       {},
		"pr":       {},
		"mr":       {},
		"ticket":   {},
		"task":     {},
		"feat":     {},
		"feature":  {},
		"fix":      {},
		"bugfix":   {},
		"hotfix":   {},
		"chore":    {},
		"docs":     {},
		"refactor": {},
		"test":     {},
		"ci":       {},
		"build":    {},
		"perf":     {},
	}
)

// ExtractTaskID extracts a tracker-agnostic Jira/Linear key (e.g. "PROJ-1234", "ENG-42")
// or GitHub/GitLab issue reference (e.g. "issue-42", "gh-42", "42-fix-bug" -> "#42") from a Git branch name.
func ExtractTaskID(branch string) string {
	b := strings.TrimSpace(branch)
	if b == "" {
		return ""
	}

	// 1. Explicit #42 token
	if m := explicitHashIssueRe.FindStringSubmatch(b); len(m) == 2 {
		return "#" + m[1]
	}

	// 2. Jira/Linear project key (excluding conventional branch keywords like issue-42, gh-42, fix-42)
	allKeys := jiraLinearKeyRe.FindAllStringSubmatch(b, -1)
	for _, m := range allKeys {
		if len(m) == 3 {
			prefixLower := strings.ToLower(m[1])
			if _, isKeyword := nonProjectPrefixes[prefixLower]; !isKeyword {
				return strings.ToUpper(m[1]) + "-" + m[2]
			}
		}
	}

	// 3. GitHub/GitLab prefixed issue numbers (issue-42, gh-42, fix-42, issue/42)
	if m := prefixedIssueRe.FindStringSubmatch(b); len(m) == 2 {
		return "#" + m[1]
	}

	// 4. Leading numeric issue branch (42-fix-auth, fix/42-crash)
	if m := leadingNumIssueRe.FindStringSubmatch(b); len(m) == 2 {
		return "#" + m[1]
	}

	return ""
}
