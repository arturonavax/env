package configsync

import (
	"bytes"
	"fmt"
	"io/fs"
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
	"*.sock",
	"*.db",
	"*.sqlite",
	"*.sqlite-wal",
	"*.sqlite-shm",
}

// CaptureToRepo ingests an active user dotfile or directory back into the repository cleanly,
// absorbing utils/copy_config.sh while preventing leakage of personal secrets outside marker blocks.
func CaptureToRepo(
	homeDir string,
	repoRoot string,
	cfg domain.ConfigurationResource,
	goos string,
) (int, error) {
	paths, err := CaptureToRepoDetailed(homeDir, repoRoot, cfg, goos, false)
	return len(paths), err
}

// CaptureToRepoDetailed captures configurations and returns the exact list of modified relative paths.
func CaptureToRepoDetailed(
	homeDir string,
	repoRoot string,
	cfg domain.ConfigurationResource,
	goos string,
	dryRun bool,
) ([]string, error) {
	if strings.TrimSpace(homeDir) == "" || strings.TrimSpace(repoRoot) == "" {
		return nil, security.ErrInvalidPath
	}
	if goos == "" {
		goos = runtime.GOOS
	}
	if !cfg.SupportsOS(goos) {
		return nil, nil
	}

	rawTarget := strings.TrimSpace(cfg.TargetForOS(goos))
	if rawTarget == "" || strings.TrimSpace(cfg.Source) == "" {
		return nil, security.ErrInvalidPath
	}

	absHome, err := filepath.Abs(homeDir)
	if err != nil {
		return nil, fmt.Errorf("resolving home directory: %w", err)
	}
	absRepo, err := filepath.Abs(repoRoot)
	if err != nil {
		return nil, fmt.Errorf("resolving repository root: %w", err)
	}

	expandedTarget := expandHomeTarget(absHome, rawTarget)
	userTarget, err := security.SafePath(absHome, expandedTarget)
	if err != nil {
		return nil, fmt.Errorf("validating user target path: %w", err)
	}

	repoDest, err := security.SafePath(absRepo, cfg.Source)
	if err != nil {
		return nil, fmt.Errorf("validating repository destination path: %w", err)
	}

	targetInfo, err := os.Stat(userTarget)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("stat active user target %q: %w", userTarget, err)
	}

	// Reject socket, device, FIFO files (not suitable for git or dotfiles)
	if targetInfo.Mode()&(os.ModeSocket|os.ModeNamedPipe|os.ModeDevice) != 0 {
		return nil, nil
	}

	// If userTarget is already symlinked to repoDest, no ingestion copy is necessary.
	if realTarget, errT := filepath.EvalSymlinks(userTarget); errT == nil {
		if realDest, errD := filepath.EvalSymlinks(repoDest); errD == nil && realTarget == realDest {
			return nil, nil
		}
	}

	relRepoDest, err := filepath.Rel(absRepo, repoDest)
	if err != nil {
		relRepoDest = cfg.Source
	}
	relRepoDest = filepath.ToSlash(relRepoDest)

	switch {
	case cfg.Strategy == domain.StrategyMarkerBlock || cfg.Strategy == domain.StrategySmartAppend:
		rawBytes, err := os.ReadFile(userTarget)
		if err != nil {
			return nil, fmt.Errorf("reading marker-managed file %q: %w", userTarget, err)
		}
		extracted, found := ExtractMarkerBlock(string(rawBytes), cfg.MarkerStart, cfg.MarkerEnd)
		if !found {
			// Do not overwrite repository template with un-marked personal user content.
			return nil, nil
		}

		// Check if destination already has identical content
		destBytes, destErr := os.ReadFile(repoDest)
		if destErr == nil && bytes.Equal(destBytes, []byte(extracted)) {
			return nil, nil
		}

		if dryRun {
			return []string{relRepoDest}, nil
		}

		if err := writeAtomicBytes(repoDest, []byte(extracted), 0o644); err != nil {
			return nil, fmt.Errorf("capturing marker block into %q: %w", repoDest, err)
		}
		return []string{relRepoDest}, nil

	case cfg.Strategy == domain.StrategyDirectorySync || targetInfo.IsDir():
		if !targetInfo.IsDir() {
			return nil, fmt.Errorf("expected directory for %q at %q", cfg.ID, userTarget)
		}
		combinedExcludes := make([]string, 0, len(defaultRuntimeCaptureExcludes)+len(cfg.Excludes))
		combinedExcludes = append(combinedExcludes, defaultRuntimeCaptureExcludes...)
		combinedExcludes = append(combinedExcludes, cfg.Excludes...)

		var captured []string
		err := filepath.WalkDir(userTarget, func(currPath string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			rel, err := filepath.Rel(userTarget, currPath)
			if err != nil || rel == "." {
				return nil
			}

			// Reject sockets, pipes, devices
			if entry.Type()&(os.ModeSocket|os.ModeNamedPipe|os.ModeDevice) != 0 {
				return nil
			}

			if matchesAnyExclude(rel, entry.Name(), combinedExcludes) {
				if entry.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}

			if entry.IsDir() {
				return nil
			}

			destPath := filepath.Join(repoDest, rel)
			srcData, err := os.ReadFile(currPath)
			if err != nil {
				return nil
			}
			dstData, err := os.ReadFile(destPath)
			if err == nil && bytes.Equal(srcData, dstData) {
				return nil
			}

			relInRepo, _ := filepath.Rel(absRepo, destPath)
			relInRepo = filepath.ToSlash(relInRepo)
			captured = append(captured, relInRepo)

			if !dryRun {
				if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
					return err
				}
				info, _ := entry.Info()
				perm := os.FileMode(0o644)
				if info != nil && info.Mode()&0o111 != 0 {
					perm = 0o755
				}
				if err := writeAtomicBytes(destPath, srcData, perm); err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
		return captured, nil

	default:
		// File sync
		srcData, err := os.ReadFile(userTarget)
		if err != nil {
			return nil, fmt.Errorf("reading user target %q: %w", userTarget, err)
		}
		dstData, err := os.ReadFile(repoDest)
		if err == nil && bytes.Equal(srcData, dstData) {
			return nil, nil
		}

		if dryRun {
			return []string{relRepoDest}, nil
		}

		deployer := NewDeployer()
		if err := deployer.deployAtomicFile(userTarget, repoDest); err != nil {
			return nil, fmt.Errorf("capturing file %q -> %q: %w", userTarget, repoDest, err)
		}
		return []string{relRepoDest}, nil
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
