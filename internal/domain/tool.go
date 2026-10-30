package domain

// ToolKind represents the runtime nature of a tool (built-in, external binary, or script).
type ToolKind string

const (
	ToolKindUnknown  ToolKind = "unknown"
	ToolKindBuiltin  ToolKind = "builtin"
	ToolKindExternal ToolKind = "external"
	ToolTypeScript   ToolKind = "script"
)

// ToolStatus represents the detected presence, runtime target, and health of a tool.
type ToolStatus struct {
	ID             string   `json:"id" yaml:"id"`
	Name           string   `json:"name" yaml:"name"`
	Category       string   `json:"category" yaml:"category"`
	Description    string   `json:"description" yaml:"description"`
	Kind           ToolKind `json:"kind" yaml:"kind"`
	Language       string   `json:"language,omitempty" yaml:"language,omitempty"`
	Executable     string   `json:"executable" yaml:"executable"`
	Path           string   `json:"path,omitempty" yaml:"path,omitempty"`
	Version        string   `json:"version,omitempty" yaml:"version,omitempty"`
	Available      bool     `json:"available" yaml:"available"`
	IsExternal     bool     `json:"is_external" yaml:"is_external"`
	ExternalPath   string   `json:"external_path,omitempty" yaml:"external_path,omitempty"`
	HasBuiltin     bool     `json:"has_builtin" yaml:"has_builtin"`
	ActiveTarget   string   `json:"active_target" yaml:"active_target"`
	RepoURL        string   `json:"repo_url,omitempty" yaml:"repo_url,omitempty"`
	UpdateStrategy string   `json:"update_strategy,omitempty" yaml:"update_strategy,omitempty"`
	Deprecated     bool     `json:"deprecated,omitempty" yaml:"deprecated,omitempty"`
	DeprecatedReason string `json:"deprecated_reason,omitempty" yaml:"deprecated_reason,omitempty"`
	RecommendPurge bool     `json:"recommend_purge,omitempty" yaml:"recommend_purge,omitempty"`
	Replacement    string   `json:"replacement,omitempty" yaml:"replacement,omitempty"`
	Archived       bool     `json:"archived,omitempty" yaml:"archived,omitempty"`
}

// ToolManifest defines the declarative configuration of a tool in YAML.
type ToolManifest struct {
	ID              string   `json:"id" yaml:"id"`
	Name            string   `json:"name" yaml:"name"`
	Category        string   `json:"category" yaml:"category"`
	Description     string   `json:"description" yaml:"description"`
	Executable      string   `json:"executable" yaml:"executable"`
	Path            string   `json:"path,omitempty" yaml:"path,omitempty"`
	FallbackBuiltin bool     `json:"fallback_builtin,omitempty" yaml:"fallback_builtin,omitempty"`
	AllowGlobalPATH bool     `json:"allow_global_path,omitempty" yaml:"allow_global_path,omitempty"`
	Language        string   `json:"language,omitempty" yaml:"language,omitempty"`
	VersionArgs     []string `json:"version_args,omitempty" yaml:"version_args,omitempty"`
	RepoURL         string   `json:"repo_url,omitempty" yaml:"repo_url,omitempty"`
	UpdateStrategy  string   `json:"update_strategy,omitempty" yaml:"update_strategy,omitempty"`
	UpdateCmd       []string `json:"update_cmd,omitempty" yaml:"update_cmd,omitempty"`
	Deprecated      bool     `json:"deprecated,omitempty" yaml:"deprecated,omitempty"`
	DeprecatedReason string  `json:"deprecated_reason,omitempty" yaml:"deprecated_reason,omitempty"`
	RecommendPurge  bool     `json:"recommend_purge,omitempty" yaml:"recommend_purge,omitempty"`
	Replacement     string   `json:"replacement,omitempty" yaml:"replacement,omitempty"`
	Archived        bool     `json:"archived,omitempty" yaml:"archived,omitempty"`
}

// ToolUpdateResult holds the outcome of updating a tool.
type ToolUpdateResult struct {
	ToolID  string `json:"tool_id" yaml:"tool_id"`
	Status  string `json:"status" yaml:"status"`
	Message string `json:"message" yaml:"message"`
	Updated bool   `json:"updated" yaml:"updated"`
}
