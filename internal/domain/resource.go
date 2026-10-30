package domain

import "strings"

// ResourceStatus represents the operational state of a managed resource.
type ResourceStatus int

const (
	StatusUnknown ResourceStatus = iota
	StatusInstalled
	StatusMissing
	StatusOutdated
	StatusFailed
	StatusUnsupported
	StatusUnmanaged
	StatusDeprecated
	StatusOrphaned
	StatusArchived
)

func (s ResourceStatus) String() string {
	switch s {
	case StatusInstalled:
		return "installed"
	case StatusMissing:
		return "missing"
	case StatusOutdated:
		return "outdated"
	case StatusFailed:
		return "failed"
	case StatusUnsupported:
		return "unsupported"
	case StatusUnmanaged:
		return "unmanaged"
	case StatusDeprecated:
		return "deprecated"
	case StatusOrphaned:
		return "orphaned"
	case StatusArchived:
		return "archived"
	default:
		return "unknown"
	}
}

// HealthStatus represents the health check verdict for a tool.
type HealthStatus int

const (
	HealthUnknown HealthStatus = iota
	HealthHealthy
	HealthWarning
	HealthFailed
)

func (h HealthStatus) String() string {
	switch h {
	case HealthHealthy:
		return "healthy"
	case HealthWarning:
		return "warning"
	case HealthFailed:
		return "failed"
	default:
		return "unknown"
	}
}

// StrategyType indicates how a resource is installed or synchronized.
type StrategyType string

const (
	StrategyNative        StrategyType = "native"
	StrategyGitHubRelease StrategyType = "github_release"
	StrategyBinary        StrategyType = "binary"
	StrategyScript        StrategyType = "script"
	StrategySource        StrategyType = "source"
	StrategyLanguagePkg   StrategyType = "language_pkg"
	StrategyGit           StrategyType = "git"
	StrategySymlink       StrategyType = "symlink"
	StrategyCopy          StrategyType = "copy"
	StrategyReplace       StrategyType = "replace"
	StrategyThreeWay      StrategyType = "three_way"
	StrategySmartAppend   StrategyType = "smart-append"
	StrategyMarkerBlock   StrategyType = "marker_block"
	StrategyDirectorySync StrategyType = "directory_sync"
)

// ManagedResource represents any package, tool, or configuration managed by anx.
type ManagedResource struct {
	ID               string         `json:"id" yaml:"id"`
	Name             string         `json:"name" yaml:"name"`
	Category         string         `json:"category" yaml:"category"`
	Description      string         `json:"description" yaml:"description"`
	BinaryName       string         `json:"binary_name,omitempty" yaml:"binary_name,omitempty"`
	Ecosystem        string         `json:"ecosystem,omitempty" yaml:"ecosystem,omitempty"`
	OSVPackage       string         `json:"osv_package,omitempty" yaml:"osv_package,omitempty"`
	Tags             []string       `json:"tags,omitempty" yaml:"tags,omitempty"`
	DependsOn        []string       `json:"depends_on,omitempty" yaml:"depends_on,omitempty"`
	SupportedOS      []string       `json:"supported_os,omitempty" yaml:"os,omitempty"`
	Status           ResourceStatus `json:"status" yaml:"-"`
	InstalledVersion string         `json:"installed_version,omitempty" yaml:"-"`
	DesiredVersion   string         `json:"desired_version,omitempty" yaml:"desired_version,omitempty"`
	LatestVersion    string         `json:"latest_version,omitempty" yaml:"-"`
	Strategy         StrategyType   `json:"strategy" yaml:"strategy,omitempty"`
	ProviderName     string         `json:"provider_name" yaml:"provider_name,omitempty"`
	IsManaged        bool           `json:"is_managed" yaml:"-"`
	Health           HealthStatus   `json:"health" yaml:"-"`
	AdvisoriesCount  int            `json:"advisories_count" yaml:"-"`
	TargetPath       string         `json:"target_path,omitempty" yaml:"target_path,omitempty"`
	SourcePath       string         `json:"source_path,omitempty" yaml:"source_path,omitempty"`
	Deprecated       bool           `json:"deprecated,omitempty" yaml:"deprecated,omitempty"`
	DeprecatedReason string         `json:"deprecated_reason,omitempty" yaml:"deprecated_reason,omitempty"`
	RecommendPurge   bool           `json:"recommend_purge,omitempty" yaml:"recommend_purge,omitempty"`
	Replacement      string         `json:"replacement,omitempty" yaml:"replacement,omitempty"`
	Archived         bool           `json:"archived,omitempty" yaml:"archived,omitempty"`
}

// ConfigurationResource represents a managed dotfile, directory tree, or shell marker block.
type ConfigurationResource struct {
	ID                 string              `json:"id" yaml:"id"`
	Name               string              `json:"name" yaml:"name"`
	Group              string              `json:"group,omitempty" yaml:"group,omitempty"`
	Description        string              `json:"description,omitempty" yaml:"description,omitempty"`
	Source             string              `json:"source" yaml:"source"`
	Target             string              `json:"target" yaml:"target"`
	TargetsByOS        map[string]string   `json:"targets_by_os,omitempty" yaml:"targets_by_os,omitempty"`
	Strategy           StrategyType        `json:"strategy" yaml:"strategy"`
	MarkerStart        string              `json:"marker_start,omitempty" yaml:"marker_start,omitempty"`
	MarkerEnd          string              `json:"marker_end,omitempty" yaml:"marker_end,omitempty"`
	Excludes           []string            `json:"excludes,omitempty" yaml:"excludes,omitempty"`
	SupportedOS        []string            `json:"supported_os,omitempty" yaml:"os,omitempty"`
	SupportedDesktops  []string            `json:"supported_desktops,omitempty" yaml:"desktops,omitempty"`
	SupportedShells    []string            `json:"supported_shells,omitempty" yaml:"shells,omitempty"`
	SupportedTerminals []string            `json:"supported_terminals,omitempty" yaml:"terminals,omitempty"`
	PostSyncCmds       [][]string          `json:"post_sync_cmds,omitempty" yaml:"post_sync_cmds,omitempty"`
	PostSyncByDesktop  map[string][][]string `json:"post_sync_by_desktop,omitempty" yaml:"post_sync_by_desktop,omitempty"`
	Backup             bool                `json:"backup" yaml:"backup"`
	Deprecated         bool                `json:"deprecated,omitempty" yaml:"deprecated,omitempty"`
	DeprecatedReason   string              `json:"deprecated_reason,omitempty" yaml:"deprecated_reason,omitempty"`
	RecommendPurge     bool                `json:"recommend_purge,omitempty" yaml:"recommend_purge,omitempty"`
	Replacement        string              `json:"replacement,omitempty" yaml:"replacement,omitempty"`
	Archived           bool                `json:"archived,omitempty" yaml:"archived,omitempty"`
}

// TargetForOS resolves the OS-specific target path, falling back to Target.
func (c ConfigurationResource) TargetForOS(goos string) string {
	if len(c.TargetsByOS) > 0 {
		if t, ok := c.TargetsByOS[goos]; ok && t != "" {
			return t
		}
	}
	return c.Target
}

// SupportsOS reports whether the configuration applies to goos.
func (c ConfigurationResource) SupportsOS(goos string) bool {
	if len(c.SupportedOS) == 0 {
		return true
	}
	for _, allowed := range c.SupportedOS {
		if allowed == goos || allowed == "all" {
			return true
		}
	}
	return false
}

// SupportsDesktop reports whether the configuration applies to the active desktop environment.
func (c ConfigurationResource) SupportsDesktop(de string) bool {
	if len(c.SupportedDesktops) == 0 {
		return true
	}
	if de == "" {
		return false
	}
	for _, allowed := range c.SupportedDesktops {
		if strings.EqualFold(allowed, de) || allowed == "all" {
			return true
		}
	}
	return false
}

// SupportsShell reports whether the configuration applies to the specified shell.
func (c ConfigurationResource) SupportsShell(sh string) bool {
	if len(c.SupportedShells) == 0 {
		return true
	}
	if sh == "" {
		return false
	}
	for _, allowed := range c.SupportedShells {
		if strings.EqualFold(allowed, sh) || allowed == "all" {
			return true
		}
	}
	return false
}

// SupportsTerminal reports whether the configuration applies to the active terminal.
func (c ConfigurationResource) SupportsTerminal(term string) bool {
	if len(c.SupportedTerminals) == 0 {
		return true
	}
	if term == "" {
		return false
	}
	for _, allowed := range c.SupportedTerminals {
		if strings.EqualFold(allowed, term) || allowed == "all" {
			return true
		}
	}
	return false
}

// Profile represents a named bundle of packages, configurations, and optional workstation hacks.
type Profile struct {
	Name           string   `json:"name" yaml:"name"`
	Description    string   `json:"description" yaml:"description"`
	Extends        string   `json:"extends,omitempty" yaml:"extends,omitempty"`
	Packages       []string `json:"packages" yaml:"packages"`
	ConfigGroups   []string `json:"config_groups,omitempty" yaml:"config_groups,omitempty"`
	Configurations []string `json:"configurations" yaml:"configurations"`
	Hacks          []string `json:"hacks,omitempty" yaml:"hacks,omitempty"`
}
