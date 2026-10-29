package configsync

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/arturonavax/env/internal/security"
)

// DeployMode specifies how configuration files are placed on the target system.
type DeployMode string

const (
	ModeCopy    DeployMode = "copy"
	ModeSymlink DeployMode = "symlink"
)

// Deployer places configuration assets onto the target system safely.
type Deployer struct{}

// NewDeployer creates a new Deployer.
func NewDeployer() *Deployer {
	return &Deployer{}
}

// SyncDirectory recursively synchronizes srcDir into dstDir, honoring glob exclusions, preserving executable permissions, and enforcing SafePath boundaries.
func SyncDirectory(srcDir, dstDir string, excludes []string) (int, error) {
	return NewDeployer().SyncDirectory(srcDir, dstDir, excludes)
}

// Deploy deploys source to target using the requested mode.
func (d *Deployer) Deploy(source, target string, mode DeployMode) error {
	info, err := os.Stat(source)
	if err != nil {
		return fmt.Errorf("stat source: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return fmt.Errorf("creating parent directory: %w", err)
	}

	if mode == ModeSymlink {
		err := d.deploySymlink(source, target)
		if err == nil {
			return nil
		}
		if runtime.GOOS == "windows" {
			if info.IsDir() {
				_, syncErr := d.SyncDirectory(source, target, nil)
				return syncErr
			}
			return d.deployAtomicFile(source, target)
		}
		return err
	}

	if info.IsDir() {
		_, err := d.SyncDirectory(source, target, nil)
		return err
	}
	return d.deployAtomicFile(source, target)
}

// SyncDirectory recursively copies tracked files from srcDir into dstDir while skipping excluded globs and guarding against path traversal.
func (d *Deployer) SyncDirectory(srcDir, dstDir string, excludes []string) (int, error) {
	if strings.TrimSpace(srcDir) == "" || strings.TrimSpace(dstDir) == "" {
		return 0, security.ErrInvalidPath
	}

	absSrc, err := filepath.Abs(srcDir)
	if err != nil {
		return 0, fmt.Errorf("resolving source directory: %w", err)
	}
	absDst, err := filepath.Abs(dstDir)
	if err != nil {
		return 0, fmt.Errorf("resolving destination directory: %w", err)
	}

	srcInfo, err := os.Stat(absSrc)
	if err != nil {
		return 0, fmt.Errorf("stat source directory %q: %w", absSrc, err)
	}
	if !srcInfo.IsDir() {
		return 0, fmt.Errorf("source %q is not a directory", absSrc)
	}

	if err := os.MkdirAll(absDst, 0o755); err != nil {
		return 0, fmt.Errorf("creating destination directory %q: %w", absDst, err)
	}

	filesSynced := 0
	walkErr := filepath.WalkDir(absSrc, func(currPath string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		rel, err := filepath.Rel(absSrc, currPath)
		if err != nil {
			return fmt.Errorf("computing relative path: %w", err)
		}
		if rel == "." {
			return nil
		}

		if matchesAnyExclude(rel, entry.Name(), excludes) {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		safeSrc, err := security.SafePath(absSrc, rel)
		if err != nil {
			return err
		}
		safeDst, err := security.SafePath(absDst, rel)
		if err != nil {
			return err
		}

		// Prevent symlink escapes outside absSrc.
		if entry.Type()&os.ModeSymlink != 0 {
			resolved, err := filepath.EvalSymlinks(safeSrc)
			if err != nil {
				return fmt.Errorf("resolving symlink %q: %w", safeSrc, err)
			}
			if _, err := security.SafePath(absSrc, resolved); err != nil {
				return err
			}
			targetInfo, err := os.Stat(resolved)
			if err != nil {
				return err
			}
			if targetInfo.IsDir() {
				return nil
			}
			if err := d.deployAtomicFile(resolved, safeDst); err != nil {
				return err
			}
			filesSynced++
			return nil
		}

		if entry.IsDir() {
			return os.MkdirAll(safeDst, 0o755)
		}

		if !entry.Type().IsRegular() {
			return nil
		}

		if err := d.deployAtomicFile(safeSrc, safeDst); err != nil {
			return err
		}
		filesSynced++
		return nil
	})
	if walkErr != nil {
		return filesSynced, walkErr
	}

	return filesSynced, nil
}

func (d *Deployer) deploySymlink(source, target string) error {
	absSource, err := filepath.Abs(source)
	if err != nil {
		return err
	}

	if _, err := os.Lstat(target); err == nil {
		if rmErr := os.RemoveAll(target); rmErr != nil {
			return rmErr
		}
	}

	return os.Symlink(absSource, target)
}

func (d *Deployer) deployAtomicFile(source, target string) error {
	srcInfo, err := os.Stat(source)
	if err != nil {
		return fmt.Errorf("stat source file %q: %w", source, err)
	}

	perm := os.FileMode(0o644)
	if srcInfo.Mode()&0o111 != 0 {
		perm = 0o755
	}

	srcFile, err := os.Open(source)
	if err != nil {
		return fmt.Errorf("opening source: %w", err)
	}
	defer srcFile.Close()

	dir := filepath.Dir(target)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating target parent directory: %w", err)
	}

	tmpFile, err := os.CreateTemp(dir, ".anx-tmp-*")
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}
	tmpName := tmpFile.Name()
	renamed := false
	defer func() {
		if !renamed {
			_ = tmpFile.Close()
			_ = os.Remove(tmpName)
		}
	}()

	if _, err := io.Copy(tmpFile, srcFile); err != nil {
		return fmt.Errorf("writing to temp file: %w", err)
	}
	if err := tmpFile.Sync(); err != nil {
		return fmt.Errorf("syncing temp file: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmpName, perm); err != nil {
		return fmt.Errorf("applying file permissions: %w", err)
	}

	if err := os.Rename(tmpName, target); err != nil {
		return err
	}
	renamed = true
	return nil
}

func writeAtomicBytes(target string, content []byte, perm os.FileMode) error {
	dir := filepath.Dir(target)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating parent directory: %w", err)
	}

	tmpFile, err := os.CreateTemp(dir, ".anx-tmp-*")
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}
	tmpName := tmpFile.Name()
	renamed := false
	defer func() {
		if !renamed {
			_ = tmpFile.Close()
			_ = os.Remove(tmpName)
		}
	}()

	if _, err := tmpFile.Write(content); err != nil {
		return fmt.Errorf("writing temp content: %w", err)
	}
	if err := tmpFile.Sync(); err != nil {
		return fmt.Errorf("syncing temp content: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmpName, perm); err != nil {
		return fmt.Errorf("chmod temp file: %w", err)
	}
	if err := os.Rename(tmpName, target); err != nil {
		return err
	}
	renamed = true
	return nil
}

func matchesAnyExclude(relPath, baseName string, excludes []string) bool {
	if len(excludes) == 0 {
		return false
	}
	slashRel := filepath.ToSlash(relPath)
	segments := strings.Split(slashRel, "/")

	for _, rawPattern := range excludes {
		pattern := strings.TrimSpace(rawPattern)
		if pattern == "" {
			continue
		}
		normPattern := filepath.ToSlash(pattern)
		if matched, _ := path.Match(normPattern, baseName); matched {
			return true
		}
		if matched, _ := path.Match(normPattern, slashRel); matched {
			return true
		}
		for _, seg := range segments {
			if matched, _ := path.Match(normPattern, seg); matched {
				return true
			}
		}
	}
	return false
}
