package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const (
	EnvConfigPath = "AITOOLS_CONFIG"
	EnvStateDir   = "AITOOLS_STATE_DIR"
)

// DefaultConfigPath resolves the user's configuration file path (~/.config/aitools/config.json or AITOOLS_CONFIG).
func DefaultConfigPath(homeDir string) string {
	if envPath := os.Getenv(EnvConfigPath); envPath != "" {
		return envPath
	}
	if homeDir == "" {
		if h, err := os.UserHomeDir(); err == nil {
			homeDir = h
		}
	}
	return filepath.Join(homeDir, ".config", "aitools", "config.json")
}

// DefaultStateDir resolves the state directory (~/.local/state/aitools or AITOOLS_STATE_DIR).
func DefaultStateDir(homeDir string) string {
	if envDir := os.Getenv(EnvStateDir); envDir != "" {
		return envDir
	}
	if homeDir == "" {
		if h, err := os.UserHomeDir(); err == nil {
			homeDir = h
		}
	}
	return filepath.Join(homeDir, ".local", "state", "aitools")
}

// DeepMergeMaps recursively merges override into base without mutating either input map.
func DeepMergeMaps(base, override map[string]any) map[string]any {
	result := make(map[string]any, len(base)+len(override))
	for k, v := range base {
		result[k] = cloneValue(v)
	}
	for k, v := range override {
		baseMap, baseIsMap := result[k].(map[string]any)
		overMap, overIsMap := v.(map[string]any)
		if baseIsMap && overIsMap {
			result[k] = DeepMergeMaps(baseMap, overMap)
			continue
		}
		result[k] = cloneValue(v)
	}
	return result
}

func cloneValue(v any) any {
	switch typed := v.(type) {
	case map[string]any:
		cp := make(map[string]any, len(typed))
		for k, val := range typed {
			cp[k] = cloneValue(val)
		}
		return cp
	case []any:
		cp := make([]any, len(typed))
		for i, val := range typed {
			cp[i] = cloneValue(val)
		}
		return cp
	default:
		return v
	}
}

// Load reads the JSON/JSONC file at configPath (falling back to DefaultConfigPath("")) and deep-merges it over defaults.
// If the file does not exist, DefaultConfig() is returned without error.
func Load(configPath string) (*Config, error) {
	if configPath == "" {
		configPath = DefaultConfigPath("")
	}

	rawUser, err := os.ReadFile(configPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return DefaultConfig()
		}
		return nil, fmt.Errorf("reading config %q: %w", configPath, err)
	}

	return LoadBytes(rawUser)
}

// LoadBytes parses JSONC bytes and deep-merges them over embedded defaults.
func LoadBytes(rawJSONC []byte) (*Config, error) {
	var baseMap map[string]any
	if err := json.Unmarshal(embeddedDefaultConfig, &baseMap); err != nil {
		return nil, fmt.Errorf("unmarshaling default config map: %w", err)
	}

	sanitized := StripJSONC(rawJSONC)
	var userMap map[string]any
	if err := json.Unmarshal(sanitized, &userMap); err != nil {
		return nil, fmt.Errorf("parsing JSONC config: %w", err)
	}

	mergedMap := DeepMergeMaps(baseMap, userMap)
	mergedBytes, err := json.Marshal(mergedMap)
	if err != nil {
		return nil, fmt.Errorf("marshaling merged config: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(mergedBytes, &cfg); err != nil {
		return nil, fmt.Errorf("decoding merged config struct: %w", err)
	}
	cfg.normalize()
	return &cfg, nil
}

// SaveConfig atomically serializes and writes cfg to configPath.
func SaveConfig(configPath string, cfg *Config) error {
	if configPath == "" {
		configPath = DefaultConfigPath("")
	}
	if cfg == nil {
		return errors.New("config cannot be nil")
	}

	// Avoid persisting built-in template text clutter unless overridden; clone first.
	toWrite := *cfg
	if toWrite.Commit.Templates != nil {
		cleanCommitTpls := map[string]string{}
		builtins := BuiltInCommitTemplates()
		for k, v := range toWrite.Commit.Templates {
			if defV, isBuiltin := builtins[k]; !isBuiltin || defV != v {
				cleanCommitTpls[k] = v
			}
		}
		toWrite.Commit.Templates = cleanCommitTpls
	}
	if toWrite.PR.Templates != nil {
		cleanPRTpls := map[string]string{}
		builtins := BuiltInPRTemplates()
		for k, v := range toWrite.PR.Templates {
			if defV, isBuiltin := builtins[k]; !isBuiltin || defV != v {
				cleanPRTpls[k] = v
			}
		}
		toWrite.PR.Templates = cleanPRTpls
	}

	encoded, err := json.MarshalIndent(&toWrite, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding config: %w", err)
	}
	encoded = append(encoded, '\n')

	return atomicWriteFile(configPath, encoded, 0o600)
}

// InitConfig initializes configPath with the embedded default configuration if absent or overwrite is true.
func InitConfig(configPath string, overwrite bool) error {
	if configPath == "" {
		configPath = DefaultConfigPath("")
	}
	if !overwrite {
		if _, err := os.Stat(configPath); err == nil {
			return nil
		}
	}
	data := append(EmbeddedDefaultBytes(), '\n')
	return atomicWriteFile(configPath, data, 0o600)
}

func atomicWriteFile(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating directory %q: %w", dir, err)
	}

	tmpFile, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return fmt.Errorf("creating temp file in %q: %w", dir, err)
	}
	tmpPath := tmpFile.Name()
	cleanupTmp := true
	defer func() {
		if cleanupTmp {
			_ = os.Remove(tmpPath)
		}
	}()

	if _, err := tmpFile.Write(data); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("writing temp file %q: %w", tmpPath, err)
	}
	if err := tmpFile.Chmod(perm); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("setting permissions on %q: %w", tmpPath, err)
	}
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("closing temp file %q: %w", tmpPath, err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("renaming %q to %q: %w", tmpPath, path, err)
	}
	cleanupTmp = false
	return nil
}
