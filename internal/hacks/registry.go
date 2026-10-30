package hacks

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

// RegistryOption configures hack Registry loading sources.
type RegistryOption func(*registryConfig)

type registryConfig struct {
	repoHacksDir string
	userHacksDir string
	skipEmbedded bool
}

type multiHackDocument struct {
	Hacks []domain.HackProfile `yaml:"hacks"`
}

// WithRepoHacksDir overlays hack profiles from a repository hacks/ directory.
func WithRepoHacksDir(dir string) RegistryOption {
	return func(c *registryConfig) {
		c.repoHacksDir = strings.TrimSpace(dir)
	}
}

// WithUserHacksDir overlays user-defined hack profiles from ~/.config/anx/hacks.d.
func WithUserHacksDir(dir string) RegistryOption {
	return func(c *registryConfig) {
		c.userHacksDir = strings.TrimSpace(dir)
	}
}

// WithSkipEmbedded disables embedded hack loading for isolated unit tests.
func WithSkipEmbedded(skip bool) RegistryOption {
	return func(c *registryConfig) {
		c.skipEmbedded = skip
	}
}

// Registry stores loaded declarative HackProfiles safely for concurrent reads.
type Registry struct {
	mu       sync.RWMutex
	profiles map[string]domain.HackProfile
}

// LoadRegistry loads embedded hack profiles overlaid with optional repo and user drop-in directories.
func LoadRegistry(opts ...RegistryOption) (*Registry, error) {
	cfg := &registryConfig{}
	for _, opt := range opts {
		if opt != nil {
			opt(cfg)
		}
	}

	reg := &Registry{
		profiles: make(map[string]domain.HackProfile, 8),
	}

	if !cfg.skipEmbedded {
		if sub, err := fs.Sub(manifests.EmbeddedFS, "hacks"); err == nil {
			if err := reg.loadFromFS(sub, "."); err != nil {
				return nil, fmt.Errorf("loading embedded hack profiles: %w", err)
			}
		}
		if data, err := fs.ReadFile(manifests.EmbeddedFS, "hacks.yaml"); err == nil {
			_ = reg.parseAndUpsert("embedded:manifests/hacks.yaml", "", data)
		}
	}

	if cfg.repoHacksDir != "" {
		if info, err := os.Stat(cfg.repoHacksDir); err == nil && info.IsDir() {
			if err := reg.loadFromOSDir(cfg.repoHacksDir); err != nil {
				return nil, fmt.Errorf("loading repo hacks from %q: %w", cfg.repoHacksDir, err)
			}
		}
	}

	if cfg.userHacksDir != "" {
		if info, err := os.Stat(cfg.userHacksDir); err == nil && info.IsDir() {
			if err := reg.loadFromOSDir(cfg.userHacksDir); err != nil {
				return nil, fmt.Errorf("loading user hacks from %q: %w", cfg.userHacksDir, err)
			}
		}
	}

	return reg, nil
}

// List returns all loaded HackProfiles sorted deterministically by ID.
func (r *Registry) List() []domain.HackProfile {
	r.mu.RLock()
	defer r.mu.RUnlock()

	ids := make([]string, 0, len(r.profiles))
	for id := range r.profiles {
		ids = append(ids, id)
	}
	slices.Sort(ids)

	out := make([]domain.HackProfile, 0, len(ids))
	for _, id := range ids {
		out = append(out, r.profiles[id])
	}
	return out
}

// Get retrieves a HackProfile by ID.
func (r *Registry) Get(id string) (domain.HackProfile, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.profiles[id]
	return p, ok
}

func (r *Registry) loadFromFS(fsys fs.FS, root string) error {
	paths := []string{}
	err := fs.WalkDir(fsys, root, func(p string, d fs.DirEntry, wErr error) error {
		if wErr != nil {
			return wErr
		}
		if !d.IsDir() && isYAMLPath(p) {
			paths = append(paths, p)
		}
		return nil
	})
	if err != nil {
		return err
	}
	slices.Sort(paths)

	for _, p := range paths {
		raw, err := fs.ReadFile(fsys, p)
		if err != nil {
			return fmt.Errorf("reading embedded %q: %w", p, err)
		}
		if err := r.parseAndUpsert(p, "", raw); err != nil {
			return err
		}
	}
	return nil
}

func (r *Registry) loadFromOSDir(root string) error {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return err
	}

	paths := []string{}
	err = filepath.WalkDir(absRoot, func(p string, d fs.DirEntry, wErr error) error {
		if wErr != nil {
			return wErr
		}
		if !d.IsDir() && isYAMLPath(p) {
			paths = append(paths, p)
		}
		return nil
	})
	if err != nil {
		return err
	}
	slices.Sort(paths)

	for _, p := range paths {
		raw, err := os.ReadFile(p)
		if err != nil {
			return fmt.Errorf("reading %q: %w", p, err)
		}
		if err := r.parseAndUpsert(p, filepath.Dir(p), raw); err != nil {
			return err
		}
	}
	return nil
}

func (r *Registry) parseAndUpsert(sourcePath, baseDir string, raw []byte) error {
	var multi multiHackDocument
	if err := yaml.Unmarshal(raw, &multi); err == nil && len(multi.Hacks) > 0 {
		for _, prof := range multi.Hacks {
			norm, vErr := normalizeAndValidateHackProfile(prof, baseDir)
			if vErr != nil {
				return fmt.Errorf("invalid hack profile in %q: %w", sourcePath, vErr)
			}
			r.profiles[norm.ID] = norm
		}
		return nil
	}

	var single domain.HackProfile
	if err := yaml.Unmarshal(raw, &single); err != nil {
		return fmt.Errorf("parsing YAML %q: %w", sourcePath, err)
	}
	norm, err := normalizeAndValidateHackProfile(single, baseDir)
	if err != nil {
		return fmt.Errorf("invalid hack profile in %q: %w", sourcePath, err)
	}
	r.profiles[norm.ID] = norm
	return nil
}

func normalizeAndValidateHackProfile(p domain.HackProfile, baseDir string) (domain.HackProfile, error) {
	p.ID = strings.TrimSpace(p.ID)
	p.Name = strings.TrimSpace(p.Name)
	if p.ID == "" {
		return domain.HackProfile{}, errors.New("hack profile id is required")
	}
	if p.Name == "" {
		return domain.HackProfile{}, fmt.Errorf("hack profile %q requires name", p.ID)
	}
	if len(p.Steps) == 0 {
		return domain.HackProfile{}, fmt.Errorf("hack profile %q must define at least one step", p.ID)
	}

	p.Dir = baseDir
	if p.BackupPaths == nil {
		p.BackupPaths = []string{}
	}
	if p.Match.OS == nil {
		p.Match.OS = []string{}
	}
	if p.Match.Distros == nil {
		p.Match.Distros = []string{}
	}
	if p.Match.DesktopEnv == nil {
		p.Match.DesktopEnv = []string{}
	}
	if p.Match.DisplayServer == nil {
		p.Match.DisplayServer = []string{}
	}

	for i := range p.Steps {
		if strings.TrimSpace(p.Steps[i].ID) == "" {
			return domain.HackProfile{}, fmt.Errorf("hack profile %q step[%d] requires id", p.ID, i)
		}
		if strings.TrimSpace(p.Steps[i].Script) == "" && len(p.Steps[i].Command) == 0 {
			return domain.HackProfile{}, fmt.Errorf("hack profile %q step %q requires script or command", p.ID, p.Steps[i].ID)
		}
		if p.Steps[i].Command == nil {
			p.Steps[i].Command = []string{}
		}
		if p.Steps[i].Args == nil {
			p.Steps[i].Args = []string{}
		}
		if p.Steps[i].Touches == nil {
			p.Steps[i].Touches = []string{}
		}
	}

	return p, nil
}

func isYAMLPath(p string) bool {
	ext := strings.ToLower(filepath.Ext(p))
	return ext == ".yaml" || ext == ".yml"
}
