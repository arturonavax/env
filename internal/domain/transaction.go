package domain

import "time"

// TxState represents the lifecycle state of a transaction.
type TxState int

const (
	TxPlanned TxState = iota + 1
	TxRunning
	TxCommitted
	TxRolledBack
	TxInterrupted
	TxFailed
)

func (s TxState) String() string {
	switch s {
	case TxPlanned:
		return "planned"
	case TxRunning:
		return "running"
	case TxCommitted:
		return "committed"
	case TxRolledBack:
		return "rolled_back"
	case TxInterrupted:
		return "interrupted"
	case TxFailed:
		return "failed"
	default:
		return "unknown"
	}
}

// JournalEntry represents an immutable event appended to journal.jsonl.
type JournalEntry struct {
	TxID       string     `json:"tx_id"`
	Timestamp  time.Time  `json:"timestamp"`
	Type       string     `json:"type"` // "begin", "step_start", "step_done", "commit", "rollback"
	StepID     int        `json:"step_id,omitempty"`
	Action     ActionType `json:"action,omitempty"`
	Target     string     `json:"target,omitempty"`
	BackupPath string     `json:"backup_path,omitempty"`
	Error      string     `json:"error,omitempty"`
	Details    string     `json:"details,omitempty"`
}

// Transaction represents a durable transaction tracked in local state.
type Transaction struct {
	ID                string     `json:"id"`
	PlanID            string     `json:"plan_id"`
	State             TxState    `json:"state"`
	StartedAt         time.Time  `json:"started_at"`
	FinishedAt        time.Time  `json:"finished_at,omitempty"`
	TotalSteps        int        `json:"total_steps"`
	CompletedSteps    int        `json:"completed_steps"`
	CompensatingSteps []PlanStep `json:"compensating_steps,omitempty"`
}
