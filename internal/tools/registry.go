package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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
type versionCacheEntry struct {
	Version string `json:"version"`
	ModTime int64  `json:"mod_time"`
	Size    int64  `json:"size"`
}

type Registry struct {
	mu           sync.RWMutex
	tools        map[string]domain.ToolManifest
	lookPath     LookPathFunc
	plat         *platform.Info
	homeDir      string
	cacheFile    string
	versionCache map[string]versionCacheEntry
	vMu          sync.RWMutex
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

	cacheFile := ""
	if cfg.plat != nil && cfg.plat.CacheDir() != "" {
		cacheFile = filepath.Join(cfg.plat.CacheDir(), "tool_versions.json")
	} else if home != "" {
		cacheFile = filepath.Join(home, ".cache", "anx", "tool_versions.json")
	}

	r := &Registry{
		tools:        make(map[string]domain.ToolManifest, 16),
		lookPath:     cfg.lookPath,
		plat:         cfg.plat,
		homeDir:      home,
		cacheFile:    cacheFile,
		versionCache: make(map[string]versionCacheEntry, 16),
	}
	r.loadVersionCache()

	// 1. Load embedded tools.yaml (a broken embedded manifest is a build defect: always fatal)
	if data, err := fs.ReadFile(manifests.EmbeddedFS, "tools.yaml"); err == nil {
		if err := r.mergeYAML("embedded:tools.yaml", data); err != nil {
			return nil, err
		}
	}

	// Overlays are fail-closed: an invalid manifest is reported with its file name instead of being
	// silently dropped, which would leave the operator running a different tool set than declared.
	var overlayErrs []error

	// 2. Overlay repository manifests/tools.yaml and manifests/tools/*.yaml. Only these paths are
	// read: manifests/ also holds packages, configs and hacks that are not tool manifests.
	if cfg.repoToolsDir != "" {
		overlayErrs = append(overlayErrs, r.loadRepoOverlay(cfg.repoToolsDir)...)
	}

	// 3. Overlay user drop-ins from ~/.config/anx/tools.d/
	if cfg.userToolsDir != "" {
		if info, err := os.Stat(cfg.userToolsDir); err == nil && info.IsDir() {
			overlayErrs = append(overlayErrs, r.loadFromDir(cfg.userToolsDir)...)
		}
	}

	if err := errors.Join(overlayErrs...); err != nil {
		return nil, fmt.Errorf("loading tool overlays: %w", err)
	}
	return r, nil
}

// loadRepoOverlay loads the tool manifests a repository may ship: tools.yaml and tools/*.yaml.
// A file path (instead of a directory) is loaded as a single manifest.
func (r *Registry) loadRepoOverlay(path string) []error {
	info, err := os.Stat(path)
	if err != nil {
		return nil // an absent overlay is normal, not a failure
	}
	if !info.IsDir() {
		return r.loadFile(path)
	}

	var errs []error
	single := filepath.Join(path, "tools.yaml")
	if fi, err := os.Stat(single); err == nil && !fi.IsDir() {
		errs = append(errs, r.loadFile(single)...)
	}
	if sub := filepath.Join(path, "tools"); dirExists(sub) {
		errs = append(errs, r.loadFromDir(sub)...)
	}
	return errs
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// loadFromDir merges every *.yaml/*.yml file below root in sorted order, collecting all failures.
func (r *Registry) loadFromDir(root string) []error {
	paths := []string{}
	var errs []error

	walkErr := filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			errs = append(errs, fmt.Errorf("walking %q: %w", path, walkErr))
			return nil
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
	if walkErr != nil {
		errs = append(errs, walkErr)
	}
	slices.Sort(paths)

	for _, p := range paths {
		errs = append(errs, r.loadFile(p)...)
	}
	return errs
}

func (r *Registry) loadFile(path string) []error {
	data, err := os.ReadFile(path)
	if err != nil {
		return []error{fmt.Errorf("reading tools YAML %q: %w", path, err)}
	}
	if err := r.mergeYAML(path, data); err != nil {
		return []error{err}
	}
	return nil
}

// mergeYAML decodes strictly (unknown keys are typos, not extensions), validates every manifest
// and merges them atomically: either the whole document is accepted or none of it is.
func (r *Registry) mergeYAML(source string, data []byte) error {
	var doc toolsDocument
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(&doc); err != nil && !errors.Is(err, io.EOF) {
		return fmt.Errorf("parsing tools YAML %q: %w", source, err)
	}

	seen := make(map[string]struct{}, len(doc.Tools))
	for _, t := range doc.Tools {
		if err := ValidateManifest(t); err != nil {
			return fmt.Errorf("%q: %w", source, err)
		}
		if _, dup := seen[t.ID]; dup {
			return fmt.Errorf("%q: %w: tool %q is defined twice", source, ErrInvalidManifest, t.ID)
		}
		seen[t.ID] = struct{}{}
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	for _, t := range doc.Tools {
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

func (r *Registry) loadVersionCache() {
	if r.cacheFile == "" {
		return
	}
	data, err := os.ReadFile(r.cacheFile)
	if err != nil {
		return
	}
	var cached map[string]versionCacheEntry
	if err := json.Unmarshal(data, &cached); err == nil && cached != nil {
		r.vMu.Lock()
		r.versionCache = cached
		r.vMu.Unlock()
	}
}

func (r *Registry) saveVersionCache() {
	if r.cacheFile == "" {
		return
	}
	r.vMu.RLock()
	data, err := json.Marshal(r.versionCache)
	r.vMu.RUnlock()
	if err != nil {
		return
	}
	_ = os.MkdirAll(filepath.Dir(r.cacheFile), 0o755)
	_ = os.WriteFile(r.cacheFile, data, 0o644)
}

func (r *Registry) detectVersion(ctx context.Context, execPath string, args []string) string {
	var modTime, size int64
	if fi, err := os.Stat(execPath); err == nil {
		modTime = fi.ModTime().UnixNano()
		size = fi.Size()
		r.vMu.RLock()
		if r.versionCache != nil {
			if entry, ok := r.versionCache[execPath]; ok && entry.ModTime == modTime && entry.Size == size && entry.Version != "" {
				r.vMu.RUnlock()
				return entry.Version
			}
		}
		r.vMu.RUnlock()
	}

	if len(args) == 0 {
		args = []string{"--version"}
	}
	vCtx, cancel := context.WithTimeout(ctx, 600*time.Millisecond)
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

	if line != "" {
		r.vMu.Lock()
		if r.versionCache == nil {
			r.versionCache = make(map[string]versionCacheEntry)
		}
		r.versionCache[execPath] = versionCacheEntry{
			Version: line,
			ModTime: modTime,
			Size:    size,
		}
		r.vMu.Unlock()
		r.saveVersionCache()
	}

	return line
}

// CheckStatusFast inspects the tool availability purely via filesystem and PATH checks
// without spawning any subprocesses or executing version detection commands.
func (r *Registry) CheckStatusFast(ctx context.Context, id string) (domain.ToolStatus, error) {
	m, ok := r.Get(id)
	if !ok {
		return domain.ToolStatus{}, fmt.Errorf("unknown tool %q", id)
	}

	status := domain.ToolStatus{
		ID:             m.ID,
		Name:           m.Name,
		Category:       m.Category,
		Language:       m.Language,
		HasBuiltin:     m.FallbackBuiltin,
		RepoURL:        m.RepoURL,
		UpdateStrategy: m.UpdateStrategy,
	}

	binName := m.Executable
	if binName == "" {
		binName = m.ID
	}

	var extPath string
	home := r.homeDir
	if home == "" && r.plat != nil {
		home = r.plat.HomeDir()
	}

	if r.plat != nil {
		toolsBin := r.plat.ToolsBinDir()
		if toolsBin != "" {
			cand := filepath.Join(toolsBin, binName)
			if r.plat.OS() == "windows" && !strings.HasSuffix(cand, ".exe") {
				cand += ".exe"
			}
			if info, err := os.Stat(cand); err == nil && !info.IsDir() {
				extPath = cand
			}
		}
	}

	if extPath == "" && r.plat != nil {
		userConfigBin := filepath.Join(r.plat.ConfigDir(), "tools", "bin", binName)
		if r.plat.OS() == "windows" && !strings.HasSuffix(userConfigBin, ".exe") {
			userConfigBin += ".exe"
		}
		if info, err := os.Stat(userConfigBin); err == nil && !info.IsDir() {
			extPath = userConfigBin
		}
	}

	if extPath == "" && m.Path != "" {
		expanded := m.Path
		if strings.HasPrefix(expanded, "~/") {
			expanded = filepath.Join(home, expanded[2:])
		}
		if info, err := os.Stat(expanded); err == nil && !info.IsDir() {
			extPath = expanded
		}
	}

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
		// Check cached version without spawning subprocess
		if fi, err := os.Stat(extPath); err == nil {
			r.vMu.RLock()
			if r.versionCache != nil {
				if entry, ok := r.versionCache[extPath]; ok && entry.ModTime == fi.ModTime().UnixNano() && entry.Size == fi.Size() && entry.Version != "" {
					status.Version = entry.Version
				}
			}
			r.vMu.RUnlock()
		}
		if status.Version == "" {
			status.Version = "available"
		}
		return status, nil
	}

	if m.FallbackBuiltin {
		status.Available = true
		status.IsExternal = false
		status.Kind = domain.ToolKindBuiltin
		status.ActiveTarget = "builtin (anx engine)"
		status.Version = "built-in"
		return status, nil
	}

	status.Available = false
	status.Kind = domain.ToolKindUnknown
	status.ActiveTarget = "missing"
	status.Version = "not installed"
	return status, nil
}

// ListFastStatuses returns the live operational status for every registered tool without executing subprocesses.
func (r *Registry) ListFastStatuses(ctx context.Context) []domain.ToolStatus {
	all := r.List()
	statuses := make([]domain.ToolStatus, len(all))
	valid := make([]bool, len(all))

	var wg sync.WaitGroup
	wg.Add(len(all))
	for i, m := range all {
		go func(idx int, toolID string) {
			defer wg.Done()
			st, err := r.CheckStatusFast(ctx, toolID)
			if err == nil {
				statuses[idx] = st
				valid[idx] = true
			}
		}(i, m.ID)
	}
	wg.Wait()

	out := make([]domain.ToolStatus, 0, len(all))
	for i := range statuses {
		if valid[i] {
			out = append(out, statuses[i])
		}
	}
	return out
}

// ListStatuses returns the live operational status for every registered tool concurrently.
func (r *Registry) ListStatuses(ctx context.Context) []domain.ToolStatus {
	all := r.List()
	statuses := make([]domain.ToolStatus, len(all))
	valid := make([]bool, len(all))

	var wg sync.WaitGroup
	wg.Add(len(all))
	for i, m := range all {
		go func(idx int, toolID string) {
			defer wg.Done()
			st, err := r.CheckStatus(ctx, toolID)
			if err == nil {
				statuses[idx] = st
				valid[idx] = true
			}
		}(i, m.ID)
	}
	wg.Wait()

	out := make([]domain.ToolStatus, 0, len(all))
	for i := range statuses {
		if valid[i] {
			out = append(out, statuses[i])
		}
	}
	return out
}
