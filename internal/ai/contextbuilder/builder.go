package contextbuilder

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	DefaultMaxTotalBytes = 500 * 1024 // 500 KiB total context cap
	DefaultMaxFileBytes  = 100 * 1024 // 100 KiB per-file cap
	DefaultMaxDirFiles   = 50
	binarySniffBytes     = 8 * 1024 // 8 KiB NUL-byte detection window
)

// ExcludedDirs mirrors files/nvim/lua/config/exclusions.lua directory exclusions.
var ExcludedDirs = map[string]struct{}{
	"node_modules":  {},
	".pnpm-store":   {},
	".yarn":         {},
	".npm":          {},
	"vendor":        {},
	".git":          {},
	".svn":          {},
	".hg":           {},
	".venv":         {},
	"venv":          {},
	"env":           {},
	"__pycache__":   {},
	".pytest_cache": {},
	".mypy_cache":   {},
	".ruff_cache":   {},
	"target":        {},
	"dist":          {},
	"build":         {},
	"out":           {},
	".next":         {},
	".nuxt":         {},
	".turbo":        {},
	".output":       {},
	".svelte-kit":   {},
	".angular":      {},
	".parcel-cache": {},
	"artifacts":     {},
	"cache":         {},
	"coverage":      {},
	".nyc_output":   {},
	".idea":         {},
	".vscode":       {},
	".cache":        {},
	".DS_Store":     {},
	".gemini":       {},
	".artifacts":    {},
}

// ExcludedFiles mirrors files/nvim/lua/config/exclusions.lua lockfile exclusions during directory walks.
var ExcludedFiles = map[string]struct{}{
	"package-lock.json": {},
	"pnpm-lock.yaml":    {},
	"yarn.lock":         {},
	"bun.lockb":         {},
	"composer.lock":     {},
	"Cargo.lock":        {},
	"poetry.lock":       {},
	"Pipfile.lock":      {},
}

// Options configures context attachment limits.
type Options struct {
	Files         []string
	Dirs          []string
	MaxTotalBytes int
	MaxFileBytes  int
	MaxDirFiles   int
}

// IsExcludedDir reports whether a directory base name is in the canonical exclusion set.
func IsExcludedDir(name string) bool {
	_, excluded := ExcludedDirs[name]
	return excluded
}

// IsExcludedFile reports whether a filename should be skipped during recursive directory traversal.
func IsExcludedFile(name string) bool {
	if _, excluded := ExcludedFiles[name]; excluded {
		return true
	}
	lower := strings.ToLower(name)
	return strings.HasSuffix(lower, ".min.js") ||
		strings.HasSuffix(lower, ".min.css") ||
		strings.HasSuffix(lower, ".map") ||
		strings.HasSuffix(lower, ".zwc")
}

// IsBinaryFile checks the first 8 KiB of path for a NUL byte.
func IsBinaryFile(path string) (bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer f.Close()

	buf := make([]byte, binarySniffBytes)
	n, err := f.Read(buf)
	if err != nil && err != io.EOF {
		return false, err
	}
	return bytes.IndexByte(buf[:n], 0) != -1, nil
}

// Build assembles context sections from requested files and directories while enforcing exclusions and byte caps.
func Build(opts Options) (string, error) {
	maxTotal := opts.MaxTotalBytes
	if maxTotal <= 0 {
		maxTotal = DefaultMaxTotalBytes
	}
	maxFile := opts.MaxFileBytes
	if maxFile <= 0 {
		maxFile = DefaultMaxFileBytes
	}
	maxDirFiles := opts.MaxDirFiles
	if maxDirFiles <= 0 {
		maxDirFiles = DefaultMaxDirFiles
	}

	var sb strings.Builder
	totalBytes := 0
	truncatedTotal := false

	appendSection := func(header, content string) bool {
		if totalBytes >= maxTotal {
			truncatedTotal = true
			return false
		}
		remaining := maxTotal - totalBytes
		fileTruncated := false
		if len(content) > maxFile {
			content = content[:maxFile]
			fileTruncated = true
		}
		if len(content) > remaining {
			content = content[:remaining]
			truncatedTotal = true
		}

		if sb.Len() > 0 {
			sb.WriteString("\n\n")
		}
		sb.WriteString(header)
		sb.WriteByte('\n')
		sb.WriteString(content)
		if fileTruncated && !truncatedTotal {
			fmt.Fprintf(&sb, "\n... [File truncated at %d bytes]", maxFile)
		}
		totalBytes += len(content)
		return !truncatedTotal
	}

	dirs := make([]string, 0, len(opts.Dirs))
	dirs = append(dirs, opts.Dirs...)

	for _, rawFile := range opts.Files {
		p := expandTilde(rawFile)
		info, err := os.Stat(p)
		if err != nil {
			if !appendSection(fmt.Sprintf("=== Attached File: %s (NOT FOUND) ===", rawFile), "[File does not exist]") {
				break
			}
			continue
		}
		if info.IsDir() {
			dirs = append(dirs, rawFile)
			continue
		}

		isBin, binErr := IsBinaryFile(p)
		if binErr != nil {
			if !appendSection(fmt.Sprintf("=== Attached File: %s (ERROR) ===", rawFile), fmt.Sprintf("[Error reading file: %v]", binErr)) {
				break
			}
			continue
		}
		if isBin {
			if !appendSection(fmt.Sprintf("=== Attached File: %s (BINARY) ===", rawFile), "[Binary file omitted]") {
				break
			}
			continue
		}

		content, readErr := readCappedFile(p, maxFile+1)
		if readErr != nil {
			if !appendSection(fmt.Sprintf("=== Attached File: %s (ERROR) ===", rawFile), fmt.Sprintf("[Error reading file: %v]", readErr)) {
				break
			}
			continue
		}
		if !appendSection(fmt.Sprintf("=== Attached File: %s ===", rawFile), content) {
			break
		}
	}

	if !truncatedTotal {
		for _, rawDir := range dirs {
			p := expandTilde(rawDir)
			info, err := os.Stat(p)
			if err != nil || !info.IsDir() {
				if !appendSection(fmt.Sprintf("=== Attached Directory: %s (NOT FOUND) ===", rawDir), "[Directory does not exist]") {
					break
				}
				continue
			}

			treeLines, candidateFiles, walkErr := collectDirEntries(p, maxDirFiles)
			if walkErr != nil {
				if !appendSection(fmt.Sprintf("=== Attached Directory: %s (ERROR) ===", rawDir), walkErr.Error()) {
					break
				}
				continue
			}

			if len(treeLines) > 100 {
				treeLines = append(treeLines[:100], "... [directory tree truncated]")
			}
			if !appendSection(fmt.Sprintf("=== Directory Tree: %s ===", rawDir), strings.Join(treeLines, "\n")) {
				break
			}

			for _, fPath := range candidateFiles {
				isBin, binErr := IsBinaryFile(fPath)
				if binErr != nil || isBin {
					continue
				}
				rel, relErr := filepath.Rel(p, fPath)
				if relErr != nil {
					rel = fPath
				}
				label := filepath.Join(rawDir, rel)
				content, readErr := readCappedFile(fPath, maxFile+1)
				if readErr != nil {
					continue
				}
				if !appendSection(fmt.Sprintf("=== File: %s ===", label), content) {
					break
				}
			}
			if truncatedTotal {
				break
			}
		}
	}

	if truncatedTotal {
		fmt.Fprintf(&sb, "\n\n... [Context limit of %d bytes reached, remaining attachments omitted]", maxTotal)
	}

	return strings.TrimSpace(sb.String()), nil
}

func collectDirEntries(rootDir string, maxFiles int) ([]string, []string, error) {
	treeLines := []string{}
	files := []string{}

	cleanRoot := filepath.Clean(rootDir)
	err := filepath.WalkDir(cleanRoot, func(curr string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		isRoot := curr == cleanRoot
		name := d.Name()

		if d.IsDir() {
			if !isRoot && (IsExcludedDir(name) || strings.HasPrefix(name, ".")) {
				return filepath.SkipDir
			}
			rel, _ := filepath.Rel(cleanRoot, curr)
			depth := 0
			if rel != "." {
				depth = strings.Count(rel, string(filepath.Separator)) + 1
			}
			base := name
			if isRoot {
				base = filepath.Base(cleanRoot)
			}
			treeLines = append(treeLines, fmt.Sprintf("%s[dir] %s/", strings.Repeat("  ", depth), base))
			return nil
		}

		if strings.HasPrefix(name, ".") || IsExcludedFile(name) {
			return nil
		}

		rel, _ := filepath.Rel(cleanRoot, curr)
		depth := strings.Count(rel, string(filepath.Separator)) + 1
		treeLines = append(treeLines, fmt.Sprintf("%s[file] %s", strings.Repeat("  ", depth), name))
		if len(files) < maxFiles {
			files = append(files, curr)
		}
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	sort.Strings(files)
	return treeLines, files, nil
}

func readCappedFile(path string, limit int) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	data, err := io.ReadAll(io.LimitReader(f, int64(limit)))
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func expandTilde(p string) string {
	if p == "~" || strings.HasPrefix(p, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			if p == "~" {
				return home
			}
			return filepath.Join(home, p[2:])
		}
	}
	return p
}
