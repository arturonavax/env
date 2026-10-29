package domain

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
}

// ConfigurationResource represents a managed dotfile, directory tree, or shell marker block.
type ConfigurationResource struct {
	ID           string            `json:"id" yaml:"id"`
	Name         string            `json:"name" yaml:"name"`
	Group        string            `json:"group,omitempty" yaml:"group,omitempty"`
	Description  string            `json:"description,omitempty" yaml:"description,omitempty"`
	Source       string            `json:"source" yaml:"source"`
	Target       string            `json:"target" yaml:"target"`
	TargetsByOS  map[string]string `json:"targets_by_os,omitempty" yaml:"targets_by_os,omitempty"`
	Strategy     StrategyType      `json:"strategy" yaml:"strategy"`
	MarkerStart  string            `json:"marker_start,omitempty" yaml:"marker_start,omitempty"`
	MarkerEnd    string            `json:"marker_end,omitempty" yaml:"marker_end,omitempty"`
	Excludes     []string          `json:"excludes,omitempty" yaml:"excludes,omitempty"`
	SupportedOS  []string          `json:"supported_os,omitempty" yaml:"os,omitempty"`
	PostSyncCmds [][]string        `json:"post_sync_cmds,omitempty" yaml:"post_sync_cmds,omitempty"`
	Backup       bool              `json:"backup" yaml:"backup"`
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
