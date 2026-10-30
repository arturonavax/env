package tools

import (
	"bytes"
	"context"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/arturonavax/env/internal/domain"
	"github.com/arturonavax/env/internal/platform"
	"github.com/arturonavax/env/manifests"
	"gopkg.in/yaml.v3"
)

type toolsDocument struct {
	Tools []domain.ToolManifest `json:"tools" yaml:"tools"`
}

// LookPathFunc signature for executable resolution.
type LookPathFunc func(string) (string, error)

type config struct {
	repoToolsDir string
	userToolsDir string
	homeDir      string
	lookPath     LookPathFunc
	plat         *platform.Info
}

// Option customizes Registry behavior.
type Option func(*config)

// WithHomeDir overrides the home directory for testing.
func WithHomeDir(dir string) Option {
	return func(c *config) {
		c.homeDir = strings.TrimSpace(dir)
	}
}

// WithRepoToolsDir configures an overlay tools manifest directory from the active repository.
func WithRepoToolsDir(dir string) Option {
	return func(c *config) {
		c.repoToolsDir = strings.TrimSpace(dir)
	}
}

// WithUserToolsDir configures an overlay tools manifest directory from user config (~/.config/anx/tools.d).
func WithUserToolsDir(dir string) Option {
	return func(c *config) {
		c.userToolsDir = strings.TrimSpace(dir)
	}
}

// WithLookPath overrides the path lookup function for testing.
func WithLookPath(fn LookPathFunc) Option {
	return func(c *config) {
		if fn != nil {
			c.lookPath = fn
		}
	}
}

// WithPlatform injects a custom platform info.
func WithPlatform(plat *platform.Info) Option {
	return func(c *config) {
		if plat != nil {
			c.plat = plat
		}
	}
}

// Registry manages declarative tool definitions, dynamic discovery, and status resolution.
type Registry struct {
	mu       sync.RWMutex
	tools    map[string]domain.ToolManifest
	lookPath LookPathFunc
	plat     *platform.Info
	homeDir  string
}

// LoadRegistry instantiates a Registry loading embedded manifests and user overlays.
func LoadRegistry(opts ...Option) (*Registry, error) {
	cfg := config{
		lookPath: exec.LookPath,
		plat:     platform.Detect(),
	}
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}
	home := cfg.homeDir
	if home == "" && cfg.plat != nil {
		home = cfg.plat.HomeDir()
	}

	r := &Registry{
		tools:    make(map[string]domain.ToolManifest, 16),
		lookPath: cfg.lookPath,
		plat:     cfg.plat,
		homeDir:  home,
	}

	// 1. Load embedded tools.yaml
	if data, err := fs.ReadFile(manifests.EmbeddedFS, "tools.yaml"); err == nil {
		if err := r.mergeYAML("embedded:tools.yaml", data); err != nil {
			return nil, err
		}
	}

	// 2. Overlay repository manifests/tools.yaml or manifests/tools/*.yaml
	if cfg.repoToolsDir != "" {
		if info, err := os.Stat(cfg.repoToolsDir); err == nil {
			if info.IsDir() {
				_ = r.loadFromDir(cfg.repoToolsDir)
			} else {
				if data, err := os.ReadFile(cfg.repoToolsDir); err == nil {
					_ = r.mergeYAML(cfg.repoToolsDir, data)
				}
			}
		}
	}

	// 3. Overlay user drop-ins from ~/.config/anx/tools.d/
	if cfg.userToolsDir != "" {
		if info, err := os.Stat(cfg.userToolsDir); err == nil && info.IsDir() {
			_ = r.loadFromDir(cfg.userToolsDir)
		}
	}

	return r, nil
}

func (r *Registry) loadFromDir(root string) error {
	paths := []string{}
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if ext == ".yaml" || ext == ".yml" {
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
			continue
		}
		_ = r.mergeYAML(p, data)
	}
	return nil
}

func (r *Registry) mergeYAML(source string, data []byte) error {
	var doc toolsDocument
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return fmt.Errorf("parsing tools YAML %q: %w", source, err)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	for _, t := range doc.Tools {
		if strings.TrimSpace(t.ID) == "" {
			continue
		}
		r.tools[t.ID] = t
	}
	return nil
}

// List returns a sorted slice of all registered tool manifests.
func (r *Registry) List() []domain.ToolManifest {
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make([]domain.ToolManifest, 0, len(r.tools))
	for _, t := range r.tools {
		out = append(out, t)
	}
	slices.SortFunc(out, func(a, b domain.ToolManifest) int {
		return strings.Compare(a.ID, b.ID)
	})
	return out
}

// Get returns the manifest for a specific tool ID.
func (r *Registry) Get(id string) (domain.ToolManifest, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	t, ok := r.tools[id]
	return t, ok
}

// Register adds or updates a tool manifest programmatically.
func (r *Registry) Register(m domain.ToolManifest) {
	if m.ID == "" {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tools[m.ID] = m
}

// CheckStatus probes the host system to determine whether the tool is available as an
// external executable, script, or built-in fallback.
func (r *Registry) CheckStatus(ctx context.Context, id string) (domain.ToolStatus, error) {
	m, ok := r.Get(id)
	if !ok {
		return domain.ToolStatus{}, fmt.Errorf("unknown tool %q", id)
	}

	status := domain.ToolStatus{
		ID:             m.ID,
		Name:           m.Name,
		Category:       m.Category,
		Description:    m.Description,
		Language:       m.Language,
		Executable:     m.Executable,
		HasBuiltin:     m.FallbackBuiltin,
		RepoURL:        m.RepoURL,
		UpdateStrategy: m.UpdateStrategy,
	}

	// 1. Check for standalone external executable in isolated toolsBinDir, ~/.config/anx/tools/bin, or explicit Path.
	// Personal tools MUST NOT search generic ~/.local/bin/ to prevent collisions with other system binaries.
	binName := m.Executable
	if binName == "" {
		binName = m.ID
	}

	var extPath string
	home := r.homeDir
	if home == "" && r.plat != nil {
		home = r.plat.HomeDir()
	}

	// Priority 1: Isolated tools directory (~/.local/share/anx/bin or ANX_TOOLS_BIN)
	toolsBin := ""
	if r.plat != nil {
		toolsBin = r.plat.ToolsBinDir()
	}
	if toolsBin != "" {
		cand := filepath.Join(toolsBin, binName)
		if r.plat != nil && r.plat.OS() == "windows" && !strings.HasSuffix(cand, ".exe") {
			cand += ".exe"
		}
		if info, err := os.Stat(cand); err == nil && !info.IsDir() {
			extPath = cand
		}
	}

	// Priority 2: User config tools directory (~/.config/anx/tools/bin)
	if extPath == "" && r.plat != nil {
		userConfigBin := filepath.Join(r.plat.ConfigDir(), "tools", "bin", binName)
		if r.plat.OS() == "windows" && !strings.HasSuffix(userConfigBin, ".exe") {
			userConfigBin += ".exe"
		}
		if info, err := os.Stat(userConfigBin); err == nil && !info.IsDir() {
			extPath = userConfigBin
		}
	}

	// Priority 3: Explicit path defined in manifest
	if extPath == "" && m.Path != "" {
		expanded := m.Path
		if strings.HasPrefix(expanded, "~/") {
			expanded = filepath.Join(home, expanded[2:])
		}
		if info, err := os.Stat(expanded); err == nil && !info.IsDir() {
			extPath = expanded
		}
	}

	// Priority 4: System PATH lookup ONLY if AllowGlobalPATH is explicitly enabled in manifest
	if extPath == "" && m.AllowGlobalPATH {
		if p, err := r.lookPath(binName); err == nil {
			extPath = p
		}
	}

	if extPath != "" {
		status.Available = true
		status.IsExternal = true
		status.ExternalPath = extPath
		status.Path = extPath
		status.Kind = domain.ToolKindExternal
		if m.Language == "python" || m.Language == "bash" || m.Language == "node" {
			status.Kind = domain.ToolTypeScript
		}
		status.ActiveTarget = fmt.Sprintf("external (%s)", extPath)
		status.Version = r.detectVersion(ctx, extPath, m.VersionArgs)
		return status, nil
	}

	// 2. If missing externally but has built-in Go engine in anx
	if m.FallbackBuiltin {
		status.Available = true
		status.IsExternal = false
		status.Kind = domain.ToolKindBuiltin
		status.ActiveTarget = "builtin (anx engine)"
		status.Version = "built-in"
		return status, nil
	}

	// 3. Not installed
	status.Available = false
	status.Kind = domain.ToolKindUnknown
	status.ActiveTarget = "missing"
	return status, nil
}

func (r *Registry) detectVersion(ctx context.Context, execPath string, args []string) string {
	if len(args) == 0 {
		args = []string{"--version"}
	}
	vCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	cmd := exec.CommandContext(vCtx, execPath, args...)
	var outBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &outBuf

	if err := cmd.Run(); err != nil {
		return "detected"
	}
	line := strings.TrimSpace(outBuf.String())
	if first := strings.Split(line, "\n"); len(first) > 0 {
		line = strings.TrimSpace(first[0])
	}
	if len(line) > 40 {
		line = line[:40]
	}
	return line
}

// ListStatuses returns the live operational status for every registered tool.
func (r *Registry) ListStatuses(ctx context.Context) []domain.ToolStatus {
	all := r.List()
	out := make([]domain.ToolStatus, 0, len(all))
	for _, m := range all {
		st, err := r.CheckStatus(ctx, m.ID)
		if err == nil {
			out = append(out, st)
		}
	}
	return out
}
