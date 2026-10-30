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
	DependsOn     []int      `json:"depends_on,omitempty"`
	Locks         []string   `json:"locks,omitempty"`
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

// ToSpec extracts an immutable PlanSpec from the execution plan.
func (p *ExecutionPlan) ToSpec() *PlanSpec {
	if p == nil {
		return nil
	}
	actions := make([]PlanAction, len(p.Steps))
	for i, s := range p.Steps {
		actions[i] = PlanAction{
			ID:            s.ID,
			ResourceID:    ResourceID(s.ResourceID),
			Action:        s.Action,
			Description:   s.Description,
			Provider:      s.Provider,
			Strategy:      s.Strategy,
			TargetPath:    s.TargetPath,
			SourcePath:    s.SourcePath,
			DependsOn:     s.DependsOn,
			Locks:         s.Locks,
			IsPrivileged:  s.IsPrivileged,
			IsDestructive: s.IsDestructive,
			IsExclusive:   s.IsExclusive,
		}
	}
	return &PlanSpec{
		SchemaVersion: "anx.dev/v1",
		Platform:      p.Platform,
		Profile:       p.Profile,
		Actions:       actions,
	}
}

// Digest computes a deterministic SHA-256 fingerprint over the plan's actions and target parameters,
// independent of execution ID or creation timestamp.
func (p *ExecutionPlan) Digest() string {
	if p == nil {
		return ""
	}
	return p.ToSpec().Digest()
}
