package providers

import (
	"archive/tar"
	"archive/zip"
	"compress/bzip2"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/arturonavax/env/internal/manifest"
	"github.com/arturonavax/env/internal/security"
)

const maxExtractedFileBytes = 512 * 1024 * 1024 // 512 MB per archive entry cap

type githubReleaseAPIResponse struct {
	TagName string `json:"tag_name"`
	Assets  []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

func (d *Dispatcher) installGitHubRelease(
	ctx context.Context,
	spec manifest.PackageSpec,
	rel manifest.GitHubReleaseSpec,
) error {
	targetKey := d.platform.OS() + "/" + d.platform.Arch()

	downloadURL, assetName, err := d.resolveGitHubAssetURL(ctx, rel, targetKey)
	if err != nil {
		return err
	}

	expectedSHA := strings.TrimSpace(rel.SHA256)
	if byTarget, ok := rel.SHA256ByTarget[targetKey]; ok && strings.TrimSpace(byTarget) != "" {
		expectedSHA = strings.TrimSpace(byTarget)
	}

	tmpDir, err := os.MkdirTemp("", "anx-ghrel-*")
	if err != nil {
		return fmt.Errorf("creating temp release directory: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	archivePath := filepath.Join(tmpDir, assetName)
	if err := d.downloader.DownloadToFile(ctx, downloadURL, archivePath, expectedSHA); err != nil {
		return err
	}

	extractDir := filepath.Join(tmpDir, "extracted")
	if err := os.MkdirAll(extractDir, 0o755); err != nil {
		return err
	}

	lowerName := strings.ToLower(assetName)
	switch {
	case strings.HasSuffix(lowerName, ".tar.gz") || strings.HasSuffix(lowerName, ".tgz"):
		if err := extractTarArchive(archivePath, extractDir, "gzip"); err != nil {
			return err
		}
	case strings.HasSuffix(lowerName, ".tar.bz2") || strings.HasSuffix(lowerName, ".tbz2"):
		if err := extractTarArchive(archivePath, extractDir, "bzip2"); err != nil {
			return err
		}
	case strings.HasSuffix(lowerName, ".zip"):
		if err := extractZipArchive(archivePath, extractDir); err != nil {
			return err
		}
	default:
		// Treat download as a standalone binary asset.
		rawDest := filepath.Join(extractDir, assetName)
		if err := copyFileAtomic(archivePath, rawDest, 0o755); err != nil {
			return err
		}
	}

	if rel.ExtractDirTo != "" {
		return d.installExtractedDirectory(extractDir, rel)
	}

	binName := rel.BinaryInArchive
	if binName == "" {
		binName = spec.BinaryName
	}
	if binName == "" {
		binName = spec.ID
	}

	foundBin, err := findBinaryInTree(extractDir, binName, d.platform.OS() == "windows")
	if err != nil {
		return err
	}

	installTo := rel.InstallTo
	if installTo == "" {
		targetBin := spec.BinaryName
		if targetBin == "" {
			targetBin = filepath.Base(foundBin)
		}
		installTo = filepath.Join(d.platform.LocalBinDir(), targetBin)
	} else {
		installTo = expandTemplateString(installTo, d.platform, d.repoDir)
	}
	if d.platform.OS() == "windows" && !strings.HasSuffix(strings.ToLower(installTo), ".exe") {
		installTo += ".exe"
	}

	return copyFileAtomic(foundBin, installTo, 0o755)
}

func (d *Dispatcher) resolveGitHubAssetURL(
	ctx context.Context,
	rel manifest.GitHubReleaseSpec,
	targetKey string,
) (string, string, error) {
	if direct, ok := rel.DirectURLs[targetKey]; ok && strings.TrimSpace(direct) != "" {
		u := strings.TrimSpace(direct)
		base := filepath.Base(u)
		if idx := strings.Index(base, "?"); idx >= 0 {
			base = base[:idx]
		}
		if base == "" || base == "." || base == "/" {
			base = "artifact.bin"
		}
		return u, base, nil
	}

	pattern, ok := rel.AssetPatterns[targetKey]
	if !ok || strings.TrimSpace(pattern) == "" {
		return "", "", fmt.Errorf("no github_release asset pattern declared for %s", targetKey)
	}
	if strings.TrimSpace(rel.Repo) == "" {
		return "", "", errors.New("github_release repo is required when direct_urls is not set")
	}

	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", rel.Repo)
	if strings.TrimSpace(rel.Version) != "" && rel.Version != "latest" {
		apiURL = fmt.Sprintf("https://api.github.com/repos/%s/releases/tags/%s", rel.Repo, rel.Version)
	}

	reqCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, apiURL, nil)
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "anx-env-manager/2.0")
	if tok := os.Getenv("GITHUB_TOKEN"); tok != "" {
		req.Header.Set("Authorization", "Bearer "+tok)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("querying GitHub release metadata for %s: %w", rel.Repo, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("GitHub release metadata for %s returned HTTP %d", rel.Repo, resp.StatusCode)
	}

	var meta githubReleaseAPIResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, 4*1024*1024)).Decode(&meta); err != nil {
		return "", "", fmt.Errorf("decoding GitHub release metadata: %w", err)
	}

	cleanVer := strings.TrimPrefix(meta.TagName, "v")
	expectedAsset := strings.ReplaceAll(pattern, "{version}", cleanVer)
	expectedAsset = strings.ReplaceAll(expectedAsset, "{tag}", meta.TagName)

	for _, asset := range meta.Assets {
		if asset.Name == expectedAsset {
			return asset.BrowserDownloadURL, asset.Name, nil
		}
	}

	return "", "", fmt.Errorf("asset %q not found in %s release %s", expectedAsset, rel.Repo, meta.TagName)
}

func (d *Dispatcher) installExtractedDirectory(extractDir string, rel manifest.GitHubReleaseSpec) error {
	destDir := expandTemplateString(rel.ExtractDirTo, d.platform, d.repoDir)
	sourceRoot := unwrapSingleTopDir(extractDir)

	if err := os.MkdirAll(filepath.Dir(destDir), 0o755); err != nil {
		return err
	}
	_ = os.RemoveAll(destDir)
	if err := copyDirectoryRecursive(sourceRoot, destDir); err != nil {
		return err
	}

	if rel.SymlinkBinary != "" && rel.InstallTo != "" {
		linkTarget := filepath.Join(destDir, rel.SymlinkBinary)
		linkPath := expandTemplateString(rel.InstallTo, d.platform, d.repoDir)
		if err := os.MkdirAll(filepath.Dir(linkPath), 0o755); err != nil {
			return err
		}
		_ = os.Remove(linkPath)
		if err := os.Symlink(linkTarget, linkPath); err != nil {
			return copyFileAtomic(linkTarget, linkPath, 0o755)
		}
	}
	return nil
}

func extractTarArchive(archivePath, destDir, compression string) error {
	f, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer f.Close()

	var stream io.Reader = f
	switch compression {
	case "gzip":
		gzr, gzErr := gzip.NewReader(f)
		if gzErr != nil {
			return fmt.Errorf("opening gzip archive: %w", gzErr)
		}
		defer gzr.Close()
		stream = gzr
	case "bzip2":
		stream = bzip2.NewReader(f)
	}

	tr := tar.NewReader(stream)
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return fmt.Errorf("reading tar entry: %w", err)
		}

		cleanRel := strings.TrimPrefix(filepath.Clean(hdr.Name), string(filepath.Separator))
		if cleanRel == "" || cleanRel == "." {
			continue
		}

		destPath, err := security.SafePath(destDir, cleanRel)
		if err != nil {
			return fmt.Errorf("tar entry blocked: %w", err)
		}

		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(destPath, 0o755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
				return err
			}
			mode := os.FileMode(hdr.Mode) & 0o777
			if mode == 0 {
				mode = 0o644
			}
			if err := writeLimitedFile(destPath, tr, mode); err != nil {
				return err
			}
		case tar.TypeSymlink:
			linkDest := filepath.Join(filepath.Dir(destPath), hdr.Linkname)
			if _, err := security.SafePath(destDir, linkDest); err != nil {
				return fmt.Errorf("unsafe tar symlink target %q: %w", hdr.Linkname, err)
			}
			if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
				return err
			}
			_ = os.Remove(destPath)
			if err := os.Symlink(hdr.Linkname, destPath); err != nil {
				return err
			}
		}
	}
	return nil
}

func extractZipArchive(archivePath, destDir string) error {
	zr, err := zip.OpenReader(archivePath)
	if err != nil {
		return fmt.Errorf("opening zip archive: %w", err)
	}
	defer zr.Close()

	for _, zf := range zr.File {
		cleanRel := strings.TrimPrefix(filepath.Clean(zf.Name), string(filepath.Separator))
		if cleanRel == "" || cleanRel == "." {
			continue
		}

		destPath, err := security.SafePath(destDir, cleanRel)
		if err != nil {
			return fmt.Errorf("zip entry blocked: %w", err)
		}

		if zf.FileInfo().IsDir() {
			if err := os.MkdirAll(destPath, 0o755); err != nil {
				return err
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
			return err
		}

		rc, err := zf.Open()
		if err != nil {
			return err
		}
		mode := zf.Mode() & 0o777
		if mode == 0 {
			mode = 0o755
		}
		writeErr := writeLimitedFile(destPath, rc, mode)
		_ = rc.Close()
		if writeErr != nil {
			return writeErr
		}
	}
	return nil
}

func writeLimitedFile(destPath string, r io.Reader, mode os.FileMode) error {
	out, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	written, copyErr := io.Copy(out, io.LimitReader(r, maxExtractedFileBytes+1))
	closeErr := out.Close()
	if copyErr != nil || closeErr != nil {
		return errors.Join(copyErr, closeErr)
	}
	if written > maxExtractedFileBytes {
		_ = os.Remove(destPath)
		return fmt.Errorf("extracted entry %q exceeded %d bytes cap", destPath, maxExtractedFileBytes)
	}
	return nil
}

func findBinaryInTree(root, binaryName string, isWindows bool) (string, error) {
	candidates := []string{binaryName}
	if isWindows && !strings.HasSuffix(strings.ToLower(binaryName), ".exe") {
		candidates = append([]string{binaryName + ".exe"}, candidates...)
	}

	var found string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil || d.IsDir() {
			return walkErr
		}
		base := filepath.Base(path)
		for _, want := range candidates {
			if base == want || strings.EqualFold(base, want) {
				found = path
				return io.EOF
			}
		}
		return nil
	})
	if err != nil && !errors.Is(err, io.EOF) {
		return "", err
	}
	if found == "" {
		return "", fmt.Errorf("binary %q not found inside extracted archive", binaryName)
	}
	return found, nil
}

func unwrapSingleTopDir(extractDir string) string {
	entries, err := os.ReadDir(extractDir)
	if err != nil || len(entries) != 1 || !entries[0].IsDir() {
		return extractDir
	}
	return filepath.Join(extractDir, entries[0].Name())
}

func copyFileAtomic(srcPath, destPath string, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
		return err
	}
	in, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer in.Close()

	tmp, err := os.CreateTemp(filepath.Dir(destPath), ".anx-bin-*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()

	_, copyErr := io.Copy(tmp, in)
	chmodErr := tmp.Chmod(mode)
	closeErr := tmp.Close()
	if copyErr != nil || chmodErr != nil || closeErr != nil {
		_ = os.Remove(tmpPath)
		return errors.Join(copyErr, chmodErr, closeErr)
	}

	if err := os.Rename(tmpPath, destPath); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	return nil
}

func copyDirectoryRecursive(srcRoot, dstRoot string) error {
	return filepath.WalkDir(srcRoot, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(srcRoot, path)
		if err != nil {
			return err
		}
		target, err := security.SafePath(dstRoot, rel)
		if err != nil {
			return err
		}
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			linkDest, err := os.Readlink(path)
			if err != nil {
				return err
			}
			_ = os.Remove(target)
			return os.Symlink(linkDest, target)
		}
		mode := info.Mode() & 0o777
		if mode == 0 {
			mode = 0o644
		}
		return copyFileAtomic(path, target, mode)
	})
}
