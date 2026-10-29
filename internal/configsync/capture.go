package configsync

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/arturonavax/env/internal/domain"
	"github.com/arturonavax/env/internal/security"
)

var defaultRuntimeCaptureExcludes = []string{
	".git",
	"*.zwc",
	"*.log",
	"session.json",
	"session-snapshots",
	".plugins.lock",
}

// CaptureToRepo ingests an active user dotfile or directory back into the repository cleanly,
// absorbing utils/copy_config.sh while preventing leakage of personal secrets outside marker blocks.
func CaptureToRepo(
	homeDir string,
	repoRoot string,
	cfg domain.ConfigurationResource,
	goos string,
) (int, error) {
	if strings.TrimSpace(homeDir) == "" || strings.TrimSpace(repoRoot) == "" {
		return 0, security.ErrInvalidPath
	}
	if goos == "" {
		goos = runtime.GOOS
	}
	if !cfg.SupportsOS(goos) {
		return 0, nil
	}

	rawTarget := strings.TrimSpace(cfg.TargetForOS(goos))
	if rawTarget == "" || strings.TrimSpace(cfg.Source) == "" {
		return 0, security.ErrInvalidPath
	}

	absHome, err := filepath.Abs(homeDir)
	if err != nil {
		return 0, fmt.Errorf("resolving home directory: %w", err)
	}
	absRepo, err := filepath.Abs(repoRoot)
	if err != nil {
		return 0, fmt.Errorf("resolving repository root: %w", err)
	}

	expandedTarget := expandHomeTarget(absHome, rawTarget)
	userTarget, err := security.SafePath(absHome, expandedTarget)
	if err != nil {
		return 0, fmt.Errorf("validating user target path: %w", err)
	}

	repoDest, err := security.SafePath(absRepo, cfg.Source)
	if err != nil {
		return 0, fmt.Errorf("validating repository destination path: %w", err)
	}

	targetInfo, err := os.Stat(userTarget)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, fmt.Errorf("stat active user target %q: %w", userTarget, err)
	}

	// If userTarget is already symlinked to repoDest, no ingestion copy is necessary.
	if realTarget, errT := filepath.EvalSymlinks(userTarget); errT == nil {
		if realDest, errD := filepath.EvalSymlinks(repoDest); errD == nil && realTarget == realDest {
			return 0, nil
		}
	}

	switch {
	case cfg.Strategy == domain.StrategyMarkerBlock || cfg.Strategy == domain.StrategySmartAppend:
		rawBytes, err := os.ReadFile(userTarget)
		if err != nil {
			return 0, fmt.Errorf("reading marker-managed file %q: %w", userTarget, err)
		}
		extracted, found := ExtractMarkerBlock(string(rawBytes), cfg.MarkerStart, cfg.MarkerEnd)
		if !found {
			// Do not overwrite repository template with un-marked personal user content.
			return 0, nil
		}
		if err := writeAtomicBytes(repoDest, []byte(extracted), 0o644); err != nil {
			return 0, fmt.Errorf("capturing marker block into %q: %w", repoDest, err)
		}
		return 1, nil

	case cfg.Strategy == domain.StrategyDirectorySync || targetInfo.IsDir():
		if !targetInfo.IsDir() {
			return 0, fmt.Errorf("expected directory for %q at %q", cfg.ID, userTarget)
		}
		combinedExcludes := make([]string, 0, len(defaultRuntimeCaptureExcludes)+len(cfg.Excludes))
		combinedExcludes = append(combinedExcludes, defaultRuntimeCaptureExcludes...)
		combinedExcludes = append(combinedExcludes, cfg.Excludes...)
		return SyncDirectory(userTarget, repoDest, combinedExcludes)

	default:
		deployer := NewDeployer()
		if err := deployer.deployAtomicFile(userTarget, repoDest); err != nil {
			return 0, fmt.Errorf("capturing file %q -> %q: %w", userTarget, repoDest, err)
		}
		return 1, nil
	}
}

func expandHomeTarget(homeDir, raw string) string {
	switch {
	case raw == "~" || raw == "$HOME":
		return homeDir
	case strings.HasPrefix(raw, "~/") || strings.HasPrefix(raw, "~\\"):
		return filepath.Join(homeDir, raw[2:])
	case strings.HasPrefix(raw, "$HOME/"):
		return filepath.Join(homeDir, strings.TrimPrefix(raw, "$HOME/"))
	default:
		return raw
	}
}
