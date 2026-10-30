package capability

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"

	"github.com/arturonavax/env/internal/domain"
)

// Definition describes an abstract capability, its preferred provider, and fallback.
type Definition struct {
	ID           string   `json:"id"`
	Alias        string   `json:"alias"`
	Description  string   `json:"description"`
	Primary      string   `json:"primary"`
	Fallback     string   `json:"fallback"`
	PrimaryArgs  []string `json:"primary_args,omitempty"`
	FallbackArgs []string `json:"fallback_args,omitempty"`
}

// Resolution describes how a capability was resolved given the current PATH.
type Resolution struct {
	Definition  Definition `json:"definition"`
	Selected    string     `json:"selected"`
	IsFallback  bool       `json:"is_fallback"`
	Explanation string     `json:"explanation"`
}

// Registry manages registered capabilities and resolves them against host environment.
type Registry struct {
	caps map[string]Definition
}

// NewRegistry initializes default capabilities.
func NewRegistry() *Registry {
	r := &Registry{caps: make(map[string]Definition)}
	r.Register(Definition{
		ID:          "files.list",
		Alias:       "ls",
		Description: "List directory contents with modern formatting",
		Primary:     "eza",
		Fallback:    "ls",
		PrimaryArgs: []string{"--group-directories-first"},
	})
	r.Register(Definition{
		ID:          "files.find",
		Alias:       "find",
		Description: "Fast filesystem search",
		Primary:     "fd",
		Fallback:    "find",
	})
	r.Register(Definition{
		ID:          "text.search",
		Alias:       "grep",
		Description: "Fast recursive regex text search",
		Primary:     "rg",
		Fallback:    "grep",
	})
	r.Register(Definition{
		ID:          "code.view",
		Alias:       "cat",
		Description: "Syntax-highlighted file viewing",
		Primary:     "bat",
		Fallback:    "cat",
	})
	return r
}

// Register registers or overrides a capability definition.
func (r *Registry) Register(def Definition) {
	r.caps[def.ID] = def
	if def.Alias != "" {
		r.caps[def.Alias] = def
	}
}

// Get finds a capability by ID or alias.
func (r *Registry) Get(idOrAlias string) (Definition, bool) {
	def, ok := r.caps[idOrAlias]
	return def, ok
}

// List returns all unique capabilities.
func (r *Registry) List() []Definition {
	seen := make(map[string]bool, len(r.caps))
	out := make([]Definition, 0, len(r.caps))
	for _, def := range r.caps {
		if !seen[def.ID] {
			seen[def.ID] = true
			out = append(out, def)
		}
	}
	return out
}

// Resolve inspects PATH to determine whether primary or fallback should execute.
func (r *Registry) Resolve(idOrAlias string) (*Resolution, error) {
	def, ok := r.Get(idOrAlias)
	if !ok {
		return nil, fmt.Errorf("unknown capability %q", idOrAlias)
	}

	if primaryPath, err := exec.LookPath(def.Primary); err == nil && primaryPath != "" {
		return &Resolution{
			Definition:  def,
			Selected:    def.Primary,
			IsFallback:  false,
			Explanation: fmt.Sprintf("Using primary provider %q found at %s", def.Primary, primaryPath),
		}, nil
	}

	if fallbackPath, err := exec.LookPath(def.Fallback); err == nil && fallbackPath != "" {
		return &Resolution{
			Definition:  def,
			Selected:    def.Fallback,
			IsFallback:  true,
			Explanation: fmt.Sprintf("Primary provider %q not found in PATH; falling back to %q at %s", def.Primary, def.Fallback, fallbackPath),
		}, nil
	}

	return nil, fmt.Errorf("neither primary provider %q nor fallback %q found in PATH for capability %s", def.Primary, def.Fallback, def.ID)
}

// Execute runs the resolved capability passing argv cleanly (no shell string concatenation).
func (r *Registry) Execute(ctx context.Context, idOrAlias string, userArgs []string, in io.Reader, out, errOut io.Writer) (int, error) {
	res, err := r.Resolve(idOrAlias)
	if err != nil {
		return 1, err
	}

	extra := res.Definition.PrimaryArgs
	if res.IsFallback {
		extra = res.Definition.FallbackArgs
	}

	args := make([]string, 0, len(extra)+len(userArgs))
	args = append(args, extra...)
	args = append(args, userArgs...)

	cmd := exec.CommandContext(ctx, res.Selected, args...)
	cmd.Stdin = in
	cmd.Stdout = out
	cmd.Stderr = errOut

	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return exitErr.ExitCode(), nil
		}
		return 1, err
	}
	return 0, nil
}

// Explain produces a domain.Provenance for a capability.
func (r *Registry) Explain(idOrAlias string) (*domain.Provenance, error) {
	def, ok := r.Get(idOrAlias)
	if !ok {
		return nil, fmt.Errorf("unknown capability %q", idOrAlias)
	}
	res, _ := r.Resolve(idOrAlias)
	selected := def.Primary
	reason := fmt.Sprintf("Primary: %s, Fallback: %s", def.Primary, def.Fallback)
	if res != nil {
		selected = res.Selected
		reason = res.Explanation
	}

	return &domain.Provenance{
		ResourceID:    domain.ResourceID(def.ID),
		Kind:          domain.KindCapability,
		Origin:        "internal/capability",
		WinningPolicy: fmt.Sprintf("selected:%s", selected),
		Reason:        reason,
		SelectedBy:    []string{def.Alias},
	}, nil
}
