package tools

import (
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/arturonavax/env/internal/domain"
)

// ErrInvalidManifest wraps every manifest validation failure so callers can match with errors.Is.
var ErrInvalidManifest = errors.New("invalid tool manifest")

var toolIDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,63}$`)

// knownUpdateStrategies mirrors the strategies implemented by Updater.Update.
var knownUpdateStrategies = map[string]struct{}{
	"":            {}, // no automatic update
	"self_update": {},
	"package":     {},
	"anx_self":    {},
	"command":     {},
}

// strategiesRequiringCommand lists strategies that are meaningless without an update command.
var strategiesRequiringCommand = map[string]struct{}{
	"self_update": {},
	"command":     {},
}

// ValidateManifest checks a manifest against the tool contract. The returned error wraps
// ErrInvalidManifest and names the offending field.
func ValidateManifest(m domain.ToolManifest) error {
	fail := func(format string, args ...any) error {
		return fmt.Errorf("%w: tool %q: %s", ErrInvalidManifest, m.ID, fmt.Sprintf(format, args...))
	}

	if !toolIDPattern.MatchString(m.ID) {
		return fail("id must match %s", toolIDPattern)
	}
	if strings.TrimSpace(m.Name) == "" {
		return fail("name is required")
	}

	// The executable is resolved inside isolated tool directories; a path would escape them.
	if m.Executable != "" && (m.Executable != filepath.Base(m.Executable) || strings.ContainsAny(m.Executable, `/\`)) {
		return fail("executable %q must be a bare file name, not a path", m.Executable)
	}
	if m.Executable == "" && m.Path == "" && !m.FallbackBuiltin {
		return fail("needs an executable, a path or fallback_builtin")
	}

	if _, ok := knownUpdateStrategies[m.UpdateStrategy]; !ok {
		return fail("unknown update_strategy %q", m.UpdateStrategy)
	}
	if _, needs := strategiesRequiringCommand[m.UpdateStrategy]; needs && len(m.UpdateCmd) == 0 {
		return fail("update_strategy %q requires update_cmd", m.UpdateStrategy)
	}
	return nil
}
