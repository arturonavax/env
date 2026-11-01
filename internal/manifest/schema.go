package manifest

import (
	"strings"

	"github.com/arturonavax/env/internal/domain"
)

// CatalogDocument represents any declarative YAML file containing packages, configurations, and/or profiles.
type CatalogDocument struct {
	Packages       []PackageSpec                  `json:"packages,omitempty" yaml:"packages,omitempty"`
	Configurations []domain.ConfigurationResource `json:"configurations,omitempty" yaml:"configurations,omitempty"`
	Profiles       []domain.Profile               `json:"profiles,omitempty" yaml:"profiles,omitempty"`
	Archives       []ArchiveRecord                `json:"archives,omitempty" yaml:"archives,omitempty"`
	Hacks          []domain.HackProfile           `json:"hacks,omitempty" yaml:"hacks,omitempty"`
}

// ArchiveRecord represents a retired, decommissioned, or deprecated component tracked historically.
type ArchiveRecord struct {
	ID             string   `json:"id" yaml:"id"`
	Name           string   `json:"name" yaml:"name"`
	Category       string   `json:"category" yaml:"category"`
	RetiredAt      string   `json:"retired_at,omitempty" yaml:"retired_at,omitempty"`
	Reason         string   `json:"reason" yaml:"reason"`
	RecommendPurge bool     `json:"recommend_purge" yaml:"recommend_purge"`
	Replacement    string   `json:"replacement,omitempty" yaml:"replacement,omitempty"`
	CleanupPaths   []string `json:"cleanup_paths,omitempty" yaml:"cleanup_paths,omitempty"`
}

// PackageSpec defines a declarative multi-platform package or tool and its ordered installation strategies.
type PackageSpec struct {
	ID               string         `json:"id" yaml:"id"`
	Name             string         `json:"name" yaml:"name"`
	Category         string         `json:"category" yaml:"category"`
	Description      string         `json:"description" yaml:"description"`
	BinaryName       string         `json:"binary_name,omitempty" yaml:"binary_name,omitempty"`
	BinaryAliases    []string       `json:"binary_aliases,omitempty" yaml:"binary_aliases,omitempty"`
	VersionArgs      []string       `json:"version_args,omitempty" yaml:"version_args,omitempty"`
	Ecosystem        string         `json:"ecosystem,omitempty" yaml:"ecosystem,omitempty"`
	OSVPackage       string         `json:"osv_package,omitempty" yaml:"osv_package,omitempty"`
	Tags             []string       `json:"tags,omitempty" yaml:"tags,omitempty"`
	DependsOn        []string       `json:"depends_on,omitempty" yaml:"depends_on,omitempty"`
	OS               []string       `json:"os,omitempty" yaml:"os,omitempty"`
	DesiredVersion   string         `json:"desired_version,omitempty" yaml:"desired_version,omitempty"`
	Deprecated       bool           `json:"deprecated,omitempty" yaml:"deprecated,omitempty"`
	DeprecatedReason string         `json:"deprecated_reason,omitempty" yaml:"deprecated_reason,omitempty"`
	RecommendPurge   bool           `json:"recommend_purge,omitempty" yaml:"recommend_purge,omitempty"`
	Replacement      string         `json:"replacement,omitempty" yaml:"replacement,omitempty"`
	Archived         bool           `json:"archived,omitempty" yaml:"archived,omitempty"`
	Strategies       []StrategySpec `json:"strategies" yaml:"strategies"`
}

// SupportsOS reports whether the package supports the given runtime.GOOS value.
func (p PackageSpec) SupportsOS(goos string) bool {
	if len(p.OS) == 0 {
		return true
	}
	for _, allowed := range p.OS {
		if allowed == goos || allowed == "all" {
			return true
		}
	}
	return false
}

// IsDirectTool reports whether this package is a direct interactive user tool as defined in domain.DirectToolsSet.
func (p PackageSpec) IsDirectTool(_ map[string][]string) bool {
	_, ok := domain.DirectToolsSet[strings.ToLower(p.ID)]
	return ok
}

// PrimaryStrategy returns the first declared strategy type, or StrategyNative if none is set.
func (p PackageSpec) PrimaryStrategy() domain.StrategyType {
	if len(p.Strategies) == 0 {
		return domain.StrategyNative
	}
	return domain.StrategyType(p.Strategies[0].Type)
}

// ToManagedResource converts the declarative PackageSpec into a domain.ManagedResource value.
func (p PackageSpec) ToManagedResource() domain.ManagedResource {
	tags := make([]string, len(p.Tags))
	copy(tags, p.Tags)

	deps := make([]string, len(p.DependsOn))
	copy(deps, p.DependsOn)

	supportedOS := make([]string, len(p.OS))
	copy(supportedOS, p.OS)

	return domain.ManagedResource{
		ID:               p.ID,
		Name:             p.Name,
		Category:         p.Category,
		Description:      p.Description,
		BinaryName:       p.BinaryName,
		Ecosystem:        p.Ecosystem,
		OSVPackage:       p.OSVPackage,
		Tags:             tags,
		DependsOn:        deps,
		SupportedOS:      supportedOS,
		DesiredVersion:   p.DesiredVersion,
		Strategy:         p.PrimaryStrategy(),
		Deprecated:       p.Deprecated,
		DeprecatedReason: p.DeprecatedReason,
		RecommendPurge:   p.RecommendPurge,
		Replacement:      p.Replacement,
		Archived:         p.Archived,
	}
}

// StrategySpec defines one tier in a package's ordered installation fallback chain.
type StrategySpec struct {
	Type          string              `json:"type" yaml:"type"`
	When          *StrategyCondition  `json:"when,omitempty" yaml:"when,omitempty"`
	Packages      map[string][]string `json:"packages,omitempty" yaml:"packages,omitempty"`
	Repositories  map[string][]string `json:"repositories,omitempty" yaml:"repositories,omitempty"`
	GitHubRelease *GitHubReleaseSpec  `json:"github_release,omitempty" yaml:"github_release,omitempty"`
	Script        *ScriptSpec         `json:"script,omitempty" yaml:"script,omitempty"`
	Source        *SourceSpec         `json:"source,omitempty" yaml:"source,omitempty"`
	LanguagePkg   *LanguagePkgSpec    `json:"language_pkg,omitempty" yaml:"language_pkg,omitempty"`
}

// StrategyCondition restricts when a strategy tier is eligible on the current host.
type StrategyCondition struct {
	OS             []string `json:"os,omitempty" yaml:"os,omitempty"`
	Arch           []string `json:"arch,omitempty" yaml:"arch,omitempty"`
	Distros        []string `json:"distros,omitempty" yaml:"distros,omitempty"`
	Desktops       []string `json:"desktops,omitempty" yaml:"desktops,omitempty"`
	DisplayServers []string `json:"display_servers,omitempty" yaml:"display_servers,omitempty"`
	RequiresBinary string   `json:"requires_binary,omitempty" yaml:"requires_binary,omitempty"`
}

// GitHubReleaseSpec configures pre-compiled release archive or binary installation.
type GitHubReleaseSpec struct {
	Repo            string            `json:"repo,omitempty" yaml:"repo,omitempty"`
	Version         string            `json:"version,omitempty" yaml:"version,omitempty"`
	AssetPatterns   map[string]string `json:"asset_patterns,omitempty" yaml:"asset_patterns,omitempty"`
	DirectURLs      map[string]string `json:"direct_urls,omitempty" yaml:"direct_urls,omitempty"`
	SHA256          string            `json:"sha256,omitempty" yaml:"sha256,omitempty"`
	SHA256ByTarget  map[string]string `json:"sha256_by_target,omitempty" yaml:"sha256_by_target,omitempty"`
	BinaryInArchive string            `json:"binary_in_archive,omitempty" yaml:"binary_in_archive,omitempty"`
	ExtractDirTo    string            `json:"extract_dir_to,omitempty" yaml:"extract_dir_to,omitempty"`
	SymlinkBinary   string            `json:"symlink_binary,omitempty" yaml:"symlink_binary,omitempty"`
	InstallTo       string            `json:"install_to,omitempty" yaml:"install_to,omitempty"`
}

// ScriptSpec configures local or remote (curl) installer script execution.
type ScriptSpec struct {
	URL         string            `json:"url,omitempty" yaml:"url,omitempty"`
	LocalScript string            `json:"local_script,omitempty" yaml:"local_script,omitempty"`
	SHA256      string            `json:"sha256,omitempty" yaml:"sha256,omitempty"`
	Interpreter string            `json:"interpreter,omitempty" yaml:"interpreter,omitempty"`
	Args        []string          `json:"args,omitempty" yaml:"args,omitempty"`
	Env         map[string]string `json:"env,omitempty" yaml:"env,omitempty"`
	Creates     string            `json:"creates,omitempty" yaml:"creates,omitempty"`
}

// SourceSpec configures a multi-step from-source git build recipe.
type SourceSpec struct {
	GitURL         string            `json:"git_url" yaml:"git_url"`
	Ref            string            `json:"ref,omitempty" yaml:"ref,omitempty"`
	BuildDeps      []string          `json:"build_deps,omitempty" yaml:"build_deps,omitempty"`
	Steps          [][]string        `json:"steps" yaml:"steps"`
	Env            map[string]string `json:"env,omitempty" yaml:"env,omitempty"`
	UninstallPaths []string          `json:"uninstall_paths,omitempty" yaml:"uninstall_paths,omitempty"`
}

// LanguagePkgSpec configures installation via a language ecosystem package manager (go, cargo, pnpm, npm, pipx).
type LanguagePkgSpec struct {
	Manager string   `json:"manager" yaml:"manager"`
	Package string   `json:"package" yaml:"package"`
	Args    []string `json:"args,omitempty" yaml:"args,omitempty"`
}
