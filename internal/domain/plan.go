package domain

import "time"

// ActionType defines the operation to be performed in a step.
type ActionType string

const (
	ActionInstall   ActionType = "install"
	ActionUpgrade   ActionType = "upgrade"
	ActionUninstall ActionType = "uninstall"
	ActionSync      ActionType = "sync"
	ActionCapture   ActionType = "capture"
	ActionBackup    ActionType = "backup"
	ActionRestore   ActionType = "restore"
	ActionRebuild   ActionType = "rebuild"
	ActionOptimize  ActionType = "optimize"
	ActionHack      ActionType = "hack"
)

// StepStatus tracks the execution phase of a single plan step.
// Zero value is StepUnknown so uninitialized steps are never mistaken for pending or succeeded.
type StepStatus int

const (
	StepUnknown StepStatus = iota
	StepPending
	StepRunning
	StepSuccess
	StepFailed
	StepSkipped
	StepCompensated
)

func (s StepStatus) String() string {
	switch s {
	case StepPending:
		return "pending"
	case StepRunning:
		return "running"
	case StepSuccess:
		return "success"
	case StepFailed:
		return "failed"
	case StepSkipped:
		return "skipped"
	case StepCompensated:
		return "compensated"
	default:
		return "unknown"
	}
}

// PlanStep represents a single discrete atomic mutation in an execution plan.
type PlanStep struct {
	ID            int        `json:"id"`
	ResourceID    string     `json:"resource_id"`
	Action        ActionType `json:"action"`
	Description   string     `json:"description"`
	Provider      string     `json:"provider,omitempty"`
	Strategy      string     `json:"strategy,omitempty"`
	TargetPath    string     `json:"target_path,omitempty"`
	SourcePath    string     `json:"source_path,omitempty"`
	IsPrivileged  bool       `json:"is_privileged"`
	IsDestructive bool       `json:"is_destructive"`
	IsExclusive   bool       `json:"is_exclusive"`
	Status        StepStatus `json:"status"`
	Error         string     `json:"error,omitempty"`
	BackupPath    string     `json:"backup_path,omitempty"`
}

// ExecutionPlan represents the ordered list of steps to transition to desired state.
type ExecutionPlan struct {
	ID        string     `json:"id"`
	CreatedAt time.Time  `json:"created_at"`
	Platform  string     `json:"platform"`
	Profile   string     `json:"profile"`
	Steps     []PlanStep `json:"steps"`
	DryRun    bool       `json:"dry_run"`
}

// StepCount returns the total number of steps.
func (p *ExecutionPlan) StepCount() int {
	if p == nil {
		return 0
	}
	return len(p.Steps)
}

// HasDestructiveSteps checks if any steps in the plan perform deletions or rewrites.
func (p *ExecutionPlan) HasDestructiveSteps() bool {
	if p == nil {
		return false
	}
	for _, s := range p.Steps {
		if s.IsDestructive {
			return true
		}
	}
	return false
}
