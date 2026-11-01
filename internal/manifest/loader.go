package manifest

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"github.com/arturonavax/env/internal/domain"
	"github.com/arturonavax/env/manifests"
	"gopkg.in/yaml.v3"
)

var validStrategyTypes = map[string]bool{
	string(domain.StrategyNative):        true,
	string(domain.StrategyGitHubRelease): true,
	string(domain.StrategyScript):        true,
	string(domain.StrategySource):        true,
	string(domain.StrategyLanguagePkg):   true,
	string(domain.StrategyBinary):        true,
}

// Option configures Catalog loading sources.
type Option func(*loadConfig)

type loadConfig struct {
	repoManifestDir string
	userDropInDir   string
}

// WithRepoManifestDir overlays YAML manifests from a live repository manifests/ directory.
func WithRepoManifestDir(dir string) Option {
	return func(c *loadConfig) {
		c.repoManifestDir = strings.TrimSpace(dir)
	}
}

// WithUserDropInDir overlays user-defined YAML manifests from ~/.config/anx/manifests.d.
func WithUserDropInDir(dir string) Option {
	return func(c *loadConfig) {
		c.userDropInDir = strings.TrimSpace(dir)
	}
}

// Catalog holds the merged declarative definitions of packages, configurations, and profiles.
type Catalog struct {
	mu             sync.RWMutex
	packages       map[string]PackageSpec
	configurations map[string]domain.ConfigurationResource
	profiles       map[string]domain.Profile
	archives       map[string]ArchiveRecord
	hacks          map[string]domain.HackProfile
}

// LoadCatalog loads embedded YAML manifests and overlays optional repository and user drop-in YAML files.
func LoadCatalog(opts ...Option) (*Catalog, error) {
	cfg := &loadConfig{}
	for _, opt := range opts {
		if opt != nil {
			opt(cfg)
		}
	}

	c := &Catalog{
		packages:       make(map[string]PackageSpec, 64),
		configurations: make(map[string]domain.ConfigurationResource, 32),
		profiles:       make(map[string]domain.Profile, 12),
		archives:       make(map[string]ArchiveRecord, 16),
		hacks:          make(map[string]domain.HackProfile, 8),
	}

	if err := c.loadFromFS(manifests.EmbeddedFS, "."); err != nil {
		return nil, fmt.Errorf("loading embedded manifests: %w", err)
	}

	if cfg.repoManifestDir != "" {
		if info, err := os.Stat(cfg.repoManifestDir); err == nil && info.IsDir() {
			if err := c.loadFromOSDir(cfg.repoManifestDir); err != nil {
				return nil, fmt.Errorf("loading repo manifests from %q: %w", cfg.repoManifestDir, err)
			}
		}
	}

	if cfg.userDropInDir != "" {
		if info, err := os.Stat(cfg.userDropInDir); err == nil && info.IsDir() {
			if err := c.loadFromOSDir(cfg.userDropInDir); err != nil {
				return nil, fmt.Errorf("loading user drop-in manifests from %q: %w", cfg.userDropInDir, err)
			}
		}
	}

	return c, nil
}

func (c *Catalog) loadFromFS(fsys fs.FS, root string) error {
	paths := []string{}
	err := fs.WalkDir(fsys, root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		if isYAMLFile(path) {
			paths = append(paths, path)
		}
		return nil
	})
	if err != nil {
		return err
	}
	slices.Sort(paths)

	for _, p := range paths {
		data, err := fs.ReadFile(fsys, p)
		if err != nil {
			return fmt.Errorf("reading %q: %w", p, err)
		}
		if err := c.mergeYAML(p, data); err != nil {
			return err
		}
	}
	return nil
}

func (c *Catalog) loadFromOSDir(root string) error {
	paths := []string{}
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		if isYAMLFile(path) {
			paths = append(paths, path)
		}
		return nil
	})
	if err != nil {
		return err
	}
	slices.Sort(paths)

	for _, p := range paths {
		data, err := os.ReadFile(p)
		if err != nil {
			return fmt.Errorf("reading %q: %w", p, err)
		}
		if err := c.mergeYAML(p, data); err != nil {
			return err
		}
	}
	return nil
}

func (c *Catalog) mergeYAML(source string, data []byte) error {
	var doc CatalogDocument
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return fmt.Errorf("parsing YAML %q: %w", source, err)
	}

	for _, pkg := range doc.Packages {
		if err := validatePackageSpec(pkg); err != nil {
			return fmt.Errorf("invalid package in %q: %w", source, err)
		}
		if pkg.BinaryAliases == nil {
			pkg.BinaryAliases = []string{}
		}
		if pkg.DependsOn == nil {
			pkg.DependsOn = []string{}
		}
		if pkg.OS == nil {
			pkg.OS = []string{}
		}
		if pkg.Tags == nil {
			pkg.Tags = []string{}
		}
		c.packages[pkg.ID] = pkg
	}

	for _, cfg := range doc.Configurations {
		if strings.TrimSpace(cfg.ID) == "" {
			return fmt.Errorf("configuration in %q is missing id", source)
		}
		if strings.TrimSpace(cfg.Source) == "" || strings.TrimSpace(cfg.Target) == "" {
			return fmt.Errorf("configuration %q in %q requires source and target", cfg.ID, source)
		}
		if cfg.Excludes == nil {
			cfg.Excludes = []string{}
		}
		if cfg.SupportedOS == nil {
			cfg.SupportedOS = []string{}
		}
		c.configurations[cfg.ID] = cfg
	}

	for _, prof := range doc.Profiles {
		if strings.TrimSpace(prof.Name) == "" {
			return fmt.Errorf("profile in %q is missing name", source)
		}
		if prof.Packages == nil {
			prof.Packages = []string{}
		}
		if prof.Configurations == nil {
			prof.Configurations = []string{}
		}
		if prof.ConfigGroups == nil {
			prof.ConfigGroups = []string{}
		}
		if prof.Hacks == nil {
			prof.Hacks = []string{}
		}
		c.profiles[prof.Name] = prof
	}

	for _, arc := range doc.Archives {
		if strings.TrimSpace(arc.ID) == "" {
			continue
		}
		if arc.CleanupPaths == nil {
			arc.CleanupPaths = []string{}
		}
		c.archives[arc.ID] = arc
	}

	for _, hk := range doc.Hacks {
		if strings.TrimSpace(hk.ID) == "" {
			continue
		}
		if hk.BackupPaths == nil {
			hk.BackupPaths = []string{}
		}
		if hk.Steps == nil {
			hk.Steps = []domain.HackStep{}
		}
		c.hacks[hk.ID] = hk
	}

	return nil
}

func validatePackageSpec(pkg PackageSpec) error {
	if strings.TrimSpace(pkg.ID) == "" {
		return errors.New("package id cannot be empty")
	}
	if strings.TrimSpace(pkg.Name) == "" {
		return fmt.Errorf("package %q requires name", pkg.ID)
	}
	if len(pkg.Strategies) == 0 {
		return fmt.Errorf("package %q must declare at least one installation strategy", pkg.ID)
	}
	for i, st := range pkg.Strategies {
		if !validStrategyTypes[st.Type] {
			return fmt.Errorf("package %q strategy[%d] has unsupported type %q", pkg.ID, i, st.Type)
		}
	}
	return nil
}

func isYAMLFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".yaml" || ext == ".yml"
}

// GetPackageSpec returns the full declarative PackageSpec by ID.
func (c *Catalog) GetPackageSpec(id string) (PackageSpec, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	spec, ok := c.packages[id]
	return spec, ok
}

// FindPackageIDByBinary finds a package whose BinaryName or BinaryAliases matches name.
func (c *Catalog) FindPackageIDByBinary(name string) (string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	for id, spec := range c.packages {
		if spec.BinaryName == name {
			return id, true
		}
		for _, alias := range spec.BinaryAliases {
			if alias == name {
				return id, true
			}
		}
	}
	return "", false
}

// ListPackageSpecs returns all PackageSpecs sorted deterministically by ID.
func (c *Catalog) ListPackageSpecs() []PackageSpec {
	c.mu.RLock()
	defer c.mu.RUnlock()

	keys := make([]string, 0, len(c.packages))
	for k := range c.packages {
		keys = append(keys, k)
	}
	slices.Sort(keys)

	result := make([]PackageSpec, 0, len(keys))
	for _, k := range keys {
		result = append(result, c.packages[k])
	}
	return result
}

// GetConfiguration returns a declarative ConfigurationResource by ID.
func (c *Catalog) GetConfiguration(id string) (domain.ConfigurationResource, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	cfg, ok := c.configurations[id]
	return cfg, ok
}

// AddConfiguration registers or updates a configuration in the catalog.
func (c *Catalog) AddConfiguration(cfg domain.ConfigurationResource) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.configurations[cfg.ID] = cfg
}

// ListConfigurations returns all configurations sorted deterministically by ID.
func (c *Catalog) ListConfigurations() []domain.ConfigurationResource {
	c.mu.RLock()
	defer c.mu.RUnlock()

	keys := make([]string, 0, len(c.configurations))
	for k := range c.configurations {
		keys = append(keys, k)
	}
	slices.Sort(keys)

	result := make([]domain.ConfigurationResource, 0, len(keys))
	for _, k := range keys {
		result = append(result, c.configurations[k])
	}
	return result
}

// ListConfigurationsByGroup filters configurations matching group (or default "all" groups excluding opt-in vscode/cursor unless requested).
func (c *Catalog) ListConfigurationsByGroup(group string) []domain.ConfigurationResource {
	all := c.ListConfigurations()
	normalized := strings.ToLower(strings.TrimSpace(group))
	if normalized == "" || normalized == "all" {
		result := make([]domain.ConfigurationResource, 0, len(all))
		for _, cfg := range all {
			// Mirror utils/sync_config.sh: vscode and cursor are synced only when explicitly targeted.
			if cfg.Group == "vscode" || cfg.Group == "cursor" {
				continue
			}
			result = append(result, cfg)
		}
		return result
	}

	result := []domain.ConfigurationResource{}
	for _, cfg := range all {
		if strings.EqualFold(cfg.Group, normalized) || strings.EqualFold(cfg.ID, normalized) {
			result = append(result, cfg)
		}
	}
	return result
}

// GetProfile returns a declarative Profile by name.
func (c *Catalog) GetProfile(name string) (domain.Profile, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	prof, ok := c.profiles[name]
	return prof, ok
}

// ListProfiles returns all profiles sorted deterministically by name.
func (c *Catalog) ListProfiles() []domain.Profile {
	c.mu.RLock()
	defer c.mu.RUnlock()

	keys := make([]string, 0, len(c.profiles))
	for k := range c.profiles {
		keys = append(keys, k)
	}
	slices.Sort(keys)

	result := make([]domain.Profile, 0, len(keys))
	for _, k := range keys {
		result = append(result, c.profiles[k])
	}
	return result
}

// ResolveProfilePackages recursively resolves inherited packages across multi-level profile extends chains.
func (c *Catalog) ResolveProfilePackages(name string) ([]string, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	chain, err := c.resolveProfileChainLocked(name)
	if err != nil {
		return nil, err
	}

	seen := make(map[string]bool)
	result := []string{}
	for _, prof := range chain {
		for _, pkgID := range prof.Packages {
			if !seen[pkgID] {
				seen[pkgID] = true
				result = append(result, pkgID)
			}
		}
	}
	return result, nil
}

// ResolveProfileConfigurations recursively resolves inherited configurations and config_groups for a profile.
func (c *Catalog) ResolveProfileConfigurations(name string) ([]domain.ConfigurationResource, error) {
	c.mu.RLock()
	chain, err := c.resolveProfileChainLocked(name)
	c.mu.RUnlock()
	if err != nil {
		return nil, err
	}

	seen := make(map[string]bool)
	result := []domain.ConfigurationResource{}

	for _, prof := range chain {
		for _, group := range prof.ConfigGroups {
			for _, cfg := range c.ListConfigurationsByGroup(group) {
				if !seen[cfg.ID] {
					seen[cfg.ID] = true
					result = append(result, cfg)
				}
			}
		}
		for _, cfgID := range prof.Configurations {
			if cfg, ok := c.GetConfiguration(cfgID); ok && !seen[cfg.ID] {
				seen[cfg.ID] = true
				result = append(result, cfg)
			}
		}
	}

	return result, nil
}

func (c *Catalog) resolveProfileChainLocked(name string) ([]domain.Profile, error) {
	visited := make(map[string]bool)
	reversed := []domain.Profile{}
	curr := name

	for curr != "" {
		if visited[curr] {
			return nil, fmt.Errorf("cyclic profile inheritance detected involving %q", curr)
		}
		visited[curr] = true

		prof, ok := c.profiles[curr]
		if !ok {
			return nil, fmt.Errorf("profile %q not found", curr)
		}
		reversed = append(reversed, prof)
		curr = strings.TrimSpace(prof.Extends)
	}

	slices.Reverse(reversed)
	return reversed, nil
}

// GetArchive returns a retired or decommissioned component archive record by ID.
func (c *Catalog) GetArchive(id string) (ArchiveRecord, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	arc, ok := c.archives[id]
	return arc, ok
}

// ListArchives returns all archive records sorted deterministically by ID.
func (c *Catalog) ListArchives() []ArchiveRecord {
	c.mu.RLock()
	defer c.mu.RUnlock()

	keys := make([]string, 0, len(c.archives))
	for k := range c.archives {
		keys = append(keys, k)
	}
	slices.Sort(keys)

	result := make([]ArchiveRecord, 0, len(keys))
	for _, k := range keys {
		result = append(result, c.archives[k])
	}
	return result
}

// GetHack returns a declarative HackProfile by ID.
func (c *Catalog) GetHack(id string) (domain.HackProfile, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	hk, ok := c.hacks[id]
	return hk, ok
}

// ListHacks returns all declarative hack profiles sorted deterministically by ID.
func (c *Catalog) ListHacks() []domain.HackProfile {
	c.mu.RLock()
	defer c.mu.RUnlock()

	keys := make([]string, 0, len(c.hacks))
	for k := range c.hacks {
		keys = append(keys, k)
	}
	slices.Sort(keys)

	result := make([]domain.HackProfile, 0, len(keys))
	for _, k := range keys {
		result = append(result, c.hacks[k])
	}
	return result
}
