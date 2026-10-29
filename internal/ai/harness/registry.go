package harness

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/arturonavax/env/internal/ai/config"
)

// FallbackPriority defines the canonical fallback discovery order when a configured default harness is unavailable.
var FallbackPriority = []string{"agy", "claude", "codex", "opencode", "pi", "llm", "ollama"}

// BackendStatus summarizes availability and I/O protocol metadata for a registered harness.
type BackendStatus struct {
	Name        string `json:"name"`
	Binary      string `json:"binary"`
	Path        string `json:"path,omitempty"`
	Available   bool   `json:"available"`
	IsDefault   bool   `json:"is_default"`
	InputMode   string `json:"input_mode"`
	OutputMode  string `json:"output_mode"`
	Description string `json:"description"`
}

// Registry resolves, inspects, and executes AI CLI harnesses with fallback discovery.
type Registry struct {
	cfg      *config.Config
	runner   ProcessRunner
	lookPath func(string) (string, error)
}

// RegistryOption customizes a Registry for production or testing.
type RegistryOption func(*Registry)

// WithProcessRunner injects a custom ProcessRunner.
func WithProcessRunner(r ProcessRunner) RegistryOption {
	return func(reg *Registry) {
		if r != nil {
			reg.runner = r
		}
	}
}

// WithLookPath injects a custom executable lookup function.
func WithLookPath(fn func(string) (string, error)) RegistryOption {
	return func(reg *Registry) {
		if fn != nil {
			reg.lookPath = fn
		}
	}
}

// NewRegistry creates a Registry backed by cfg.
func NewRegistry(cfg *config.Config, opts ...RegistryOption) *Registry {
	if cfg == nil {
		cfg, _ = config.DefaultConfig()
	}
	reg := &Registry{
		cfg:      cfg,
		runner:   NewDefaultProcessRunner(),
		lookPath: exec.LookPath,
	}
	for _, opt := range opts {
		opt(reg)
	}
	return reg
}

// Resolve looks up backendName in cfg.Backends, built-in defaults, or ad-hoc binaries in PATH.
func (r *Registry) Resolve(backendName string) (*Adapter, error) {
	name := strings.TrimSpace(backendName)
	if name == "" {
		name = r.defaultBackendName()
	}

	if spec, ok := r.cfg.Backends[name]; ok {
		return NewAdapter(name, spec, r.runner, r.lookPath), nil
	}

	defCfg, err := config.DefaultConfig()
	if err == nil {
		if spec, ok := defCfg.Backends[name]; ok {
			return NewAdapter(name, spec, r.runner, r.lookPath), nil
		}
	}

	// Ad-hoc executable path or binary found in PATH
	fields := strings.Fields(name)
	if len(fields) > 0 {
		bin := fields[0]
		if _, err := r.lookPath(bin); err == nil || fileIsExecutable(bin) {
			adHoc := config.BackendConfig{
				Name:          name,
				Description:   "Ad-hoc CLI harness",
				Command:       fields,
				InputMode:     "stdin",
				OutputMode:    "stdout",
				CleanANSI:     true,
				CleanMarkdown: true,
				TimeoutSec:    60,
			}
			return NewAdapter(name, adHoc, r.runner, r.lookPath), nil
		}
	}

	names := r.AvailableNames()
	return nil, fmt.Errorf("unknown AI backend %q (configured backends: %s)", backendName, strings.Join(names, ", "))
}

// ResolveWithFallback resolves preferred and, if its binary is missing, probes FallbackPriority.
// The returned boolean indicates whether a fallback harness was substituted.
func (r *Registry) ResolveWithFallback(preferred string) (*Adapter, bool, error) {
	primary, err := r.Resolve(preferred)
	if err != nil {
		return nil, false, err
	}
	if primary.IsAvailable() {
		return primary, false, nil
	}

	candidates := make([]string, 0, len(FallbackPriority)+len(r.cfg.Backends))
	candidates = append(candidates, FallbackPriority...)
	for k := range r.cfg.Backends {
		if !slices.Contains(candidates, k) {
			candidates = append(candidates, k)
		}
	}

	for _, cand := range candidates {
		if cand == primary.Name() {
			continue
		}
		fb, fbErr := r.Resolve(cand)
		if fbErr == nil && fb.IsAvailable() {
			return fb, true, nil
		}
	}

	return nil, false, fmt.Errorf(
		"AI harness %q binary (%q) not found in PATH and no fallback CLI harnesses were detected",
		primary.Name(),
		primary.Binary(),
	)
}

// Query resolves the target backend (with fallback discovery) and executes req.
func (r *Registry) Query(ctx context.Context, preferredBackend string, req Request) (string, string, error) {
	if req.Effort == "" {
		req.Effort = r.cfg.LLM.DefaultEffort
	}
	if req.Model == "" {
		req.Model = r.cfg.LLM.DefaultModel
	}
	if req.Mode == "" {
		req.Mode = r.cfg.LLM.DefaultMode
	}
	if req.OutputFormat == "" {
		req.OutputFormat = r.cfg.LLM.DefaultOutputFormat
	}
	if req.Timeout <= 0 && r.cfg.LLM.TimeoutSec > 0 {
		req.Timeout = time.Duration(r.cfg.LLM.TimeoutSec) * time.Second
	}

	target := strings.TrimSpace(preferredBackend)
	if target == "" {
		target = r.cfg.LLM.DefaultBackend
	}
	if target == "" {
		target = r.defaultBackendName()
	}

	adapter, _, err := r.ResolveWithFallback(target)
	if err != nil {
		return "", "", err
	}

	resp, err := adapter.Execute(ctx, req)
	return resp, adapter.Name(), err
}

// AvailableNames returns all sorted backend names known to this registry.
func (r *Registry) AvailableNames() []string {
	set := map[string]struct{}{}
	for k := range r.cfg.Backends {
		set[k] = struct{}{}
	}
	if defCfg, err := config.DefaultConfig(); err == nil {
		for k := range defCfg.Backends {
			set[k] = struct{}{}
		}
	}
	names := make([]string, 0, len(set))
	for k := range set {
		names = append(names, k)
	}
	sort.Strings(names)
	return names
}

// ListStatuses returns sorted BackendStatus entries for all registered backends.
func (r *Registry) ListStatuses() []BackendStatus {
	names := r.AvailableNames()
	defName := r.defaultBackendName()
	statuses := make([]BackendStatus, 0, len(names))

	for _, name := range names {
		adapter, err := r.Resolve(name)
		if err != nil {
			continue
		}
		spec := adapter.Spec()
		bin := adapter.Binary()
		resolvedPath, lookErr := r.lookPath(bin)
		avail := lookErr == nil
		if !avail && fileIsExecutable(bin) {
			avail = true
			resolvedPath = bin
		}
		desc := spec.Description
		if desc == "" {
			desc = "AI CLI harness " + name
		}
		statuses = append(statuses, BackendStatus{
			Name:        name,
			Binary:      bin,
			Path:        resolvedPath,
			Available:   avail,
			IsDefault:   name == defName,
			InputMode:   spec.InputMode,
			OutputMode:  spec.OutputMode,
			Description: desc,
		})
	}
	return statuses
}

func (r *Registry) defaultBackendName() string {
	if r.cfg.Commit.DefaultBackend != "" {
		return r.cfg.Commit.DefaultBackend
	}
	if r.cfg.DefaultCommitBackend != "" {
		return r.cfg.DefaultCommitBackend
	}
	return "agy"
}

func fileIsExecutable(p string) bool {
	info, err := os.Stat(p)
	if err != nil || info.IsDir() {
		return false
	}
	return info.Mode()&0o111 != 0
}
