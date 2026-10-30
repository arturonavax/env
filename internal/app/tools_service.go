package app

import (
	"context"
	"io"

	"github.com/arturonavax/env/internal/domain"
	"github.com/arturonavax/env/internal/tools"
)

// ToolsService orchestrates decoupled developer tool discovery, execution, and lifecycle updates.
type ToolsService struct {
	registry *tools.Registry
	runner   *tools.Runner
	updater  *tools.Updater
}

// NewToolsService constructs a ToolsService instance.
func NewToolsService(
	registry *tools.Registry,
	runner *tools.Runner,
	updater *tools.Updater,
) *ToolsService {
	return &ToolsService{
		registry: registry,
		runner:   runner,
		updater:  updater,
	}
}

// Registry returns the underlying tools registry.
func (s *ToolsService) Registry() *tools.Registry {
	return s.registry
}

// Runner returns the underlying tools runner.
func (s *ToolsService) Runner() *tools.Runner {
	return s.runner
}

// List returns the statuses of all registered and discovered tools.
func (s *ToolsService) List(ctx context.Context) []domain.ToolStatus {
	if s.registry == nil {
		return nil
	}
	return s.registry.ListStatuses(ctx)
}

// Get returns the operational status for a single tool by ID.
func (s *ToolsService) Get(ctx context.Context, id string) (domain.ToolStatus, error) {
	if s.registry == nil {
		return domain.ToolStatus{}, nil
	}
	return s.registry.CheckStatus(ctx, id)
}

// Run executes the tool with the given arguments and I/O streams.
func (s *ToolsService) Run(
	ctx context.Context,
	toolID string,
	args []string,
	in io.Reader,
	out, errOut io.Writer,
) error {
	if s.runner == nil {
		return nil
	}
	return s.runner.Execute(ctx, toolID, args, in, out, errOut)
}

// Update updates the specified tool.
func (s *ToolsService) Update(ctx context.Context, toolID string) (domain.ToolUpdateResult, error) {
	if s.updater == nil {
		return domain.ToolUpdateResult{ToolID: toolID, Status: "error", Message: "updater not configured"}, nil
	}
	return s.updater.Update(ctx, toolID)
}
