package selfupdate

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// Release artifact naming contract, shared with the release pipeline and the installers.
const (
	// ChecksumsName lists "<sha256>  <asset>" for every archive in a release.
	ChecksumsName = "SHA256SUMS"
	// SignatureName holds the base64 ed25519 signature of ChecksumsName.
	SignatureName = "SHA256SUMS.sig"
	// BinaryName is the executable inside every archive.
	BinaryName = "anx"

	maxAPIResponseBytes = 4 << 20
)

// ErrNoRelease indicates the repository has no published (non-draft) release.
var ErrNoRelease = errors.New("no published release found")

// ErrUnsupportedPlatform indicates ANX publishes no archive for the target OS/architecture.
var ErrUnsupportedPlatform = errors.New("unsupported platform for release archives")

// Asset is a downloadable release file.
type Asset struct {
	Name string
	URL  string
	Size int64
}

// Release is a published version and its assets.
type Release struct {
	Tag        string
	Prerelease bool
	Assets     []Asset
}

// Asset returns the asset with the given name.
func (r Release) Asset(name string) (Asset, bool) {
	for _, a := range r.Assets {
		if a.Name == name {
			return a, true
		}
	}
	return Asset{}, false
}

// Source discovers releases. It is an interface so the updater can be tested against a fake and
// so alternative distribution channels (mirrors, enterprise registries) can plug in later.
type Source interface {
	// Latest returns the newest stable release, or the newest including pre-releases.
	Latest(ctx context.Context, includePrerelease bool) (Release, error)
	// ByTag returns a specific release.
	ByTag(ctx context.Context, tag string) (Release, error)
}

// ArchiveName returns the archive file name for a platform: anx_<os>_<arch>.tar.gz (.zip on Windows).
func ArchiveName(goos, goarch string) (string, error) {
	switch goos + "/" + goarch {
	case "linux/amd64", "linux/arm64", "darwin/amd64", "darwin/arm64":
		return fmt.Sprintf("anx_%s_%s.tar.gz", goos, goarch), nil
	case "windows/amd64", "windows/arm64":
		return fmt.Sprintf("anx_%s_%s.zip", goos, goarch), nil
	}
	return "", fmt.Errorf("%w: %s/%s", ErrUnsupportedPlatform, goos, goarch)
}

// GitHubSource reads releases from the GitHub REST API.
type GitHubSource struct {
	// APIBase defaults to https://api.github.com; override for tests, mirrors or GHES.
	APIBase string
	// Repo is "owner/name"; defaults to Repository.
	Repo   string
	Client *http.Client
}

type ghRelease struct {
	Tag        string `json:"tag_name"`
	Draft      bool   `json:"draft"`
	Prerelease bool   `json:"prerelease"`
	Assets     []struct {
		Name string `json:"name"`
		URL  string `json:"browser_download_url"`
		Size int64  `json:"size"`
	} `json:"assets"`
}

func (g GitHubSource) base() string {
	if g.APIBase != "" {
		return strings.TrimRight(g.APIBase, "/")
	}
	return "https://api.github.com"
}

func (g GitHubSource) repo() string {
	if g.Repo != "" {
		return g.Repo
	}
	return Repository
}

func (g GitHubSource) client() *http.Client {
	if g.Client != nil {
		return g.Client
	}
	return http.DefaultClient
}

// Latest implements Source.
func (g GitHubSource) Latest(ctx context.Context, includePrerelease bool) (Release, error) {
	if !includePrerelease {
		var r ghRelease
		if err := g.get(ctx, fmt.Sprintf("/repos/%s/releases/latest", g.repo()), &r); err != nil {
			return Release{}, err
		}
		return g.convert(r)
	}

	var list []ghRelease
	if err := g.get(ctx, fmt.Sprintf("/repos/%s/releases?per_page=20", g.repo()), &list); err != nil {
		return Release{}, err
	}
	var best *Release
	var bestVer Version
	for _, r := range list {
		if r.Draft {
			continue
		}
		v, err := ParseVersion(r.Tag)
		if err != nil {
			continue
		}
		if best == nil || v.Compare(bestVer) > 0 {
			rel, err := g.convert(r)
			if err != nil {
				continue
			}
			best, bestVer = &rel, v
		}
	}
	if best == nil {
		return Release{}, ErrNoRelease
	}
	return *best, nil
}

// ByTag implements Source.
func (g GitHubSource) ByTag(ctx context.Context, tag string) (Release, error) {
	var r ghRelease
	path := fmt.Sprintf("/repos/%s/releases/tags/%s", g.repo(), url.PathEscape(tag))
	if err := g.get(ctx, path, &r); err != nil {
		return Release{}, err
	}
	return g.convert(r)
}

func (g GitHubSource) convert(r ghRelease) (Release, error) {
	if r.Draft || r.Tag == "" {
		return Release{}, ErrNoRelease
	}
	rel := Release{Tag: r.Tag, Prerelease: r.Prerelease, Assets: make([]Asset, 0, len(r.Assets))}
	for _, a := range r.Assets {
		rel.Assets = append(rel.Assets, Asset{Name: a.Name, URL: a.URL, Size: a.Size})
	}
	return rel, nil
}

func (g GitHubSource) get(ctx context.Context, path string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, g.base()+path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "anx-self-update")

	resp, err := g.client().Do(req)
	if err != nil {
		return fmt.Errorf("querying release API: %w", err)
	}
	defer resp.Body.Close()

	switch {
	case resp.StatusCode == http.StatusNotFound:
		return ErrNoRelease
	case resp.StatusCode != http.StatusOK:
		return fmt.Errorf("release API returned %s", resp.Status)
	}

	dec := json.NewDecoder(io.LimitReader(resp.Body, maxAPIResponseBytes))
	if err := dec.Decode(out); err != nil {
		return fmt.Errorf("decoding release API response: %w", err)
	}
	return nil
}
