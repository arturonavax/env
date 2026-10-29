package app

import "github.com/arturonavax/env/internal/domain"

// ProgressEvent communicates execution status from services to CLI/TUI renderers.
type ProgressEvent struct {
	StepID      int               `json:"step_id"`
	TotalSteps  int               `json:"total_steps"`
	ResourceID  string            `json:"resource_id"`
	Action      domain.ActionType `json:"action"`
	Description string            `json:"description"`
	Status      domain.StepStatus `json:"status"`
	Error       string            `json:"error,omitempty"`
}

// EventHandler receives progress events.
type EventHandler func(ProgressEvent)
