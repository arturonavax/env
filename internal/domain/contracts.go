package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"time"
)

// ResourceID uniquely identifies a resource across kinds (e.g. "package.ghostty", "config.zshrc").
type ResourceID string

// ResourceKind categorizes managed artifacts.
type ResourceKind string

const (
	KindPackage       ResourceKind = "package"
	KindConfig        ResourceKind = "config"
	KindTool          ResourceKind = "tool"
	KindService       ResourceKind = "service"
	KindHack          ResourceKind = "hack"
	KindRuntime       ResourceKind = "runtime"
	KindRepository    ResourceKind = "repo"
	KindCapability    ResourceKind = "capability"
	KindSettingBundle ResourceKind = "setting_bundle"
	KindProfile       ResourceKind = "profile"
	KindExtension     ResourceKind = "extension"
)

// ResourceSpec declares the desired static configuration for a resource.
type ResourceSpec struct {
	ID          ResourceID     `json:"id"`
	Kind        ResourceKind   `json:"kind"`
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Version     string         `json:"version,omitempty"`
	Strategy    string         `json:"strategy,omitempty"`
	SourcePath  string         `json:"source_path,omitempty"`
	TargetPath  string         `json:"target_path,omitempty"`
	DependsOn   []ResourceID   `json:"depends_on,omitempty"`
	SupportedOS []string       `json:"supported_os,omitempty"`
	Properties  map[string]any `json:"properties,omitempty"`
}

// ObservedState captures the real-world live state of a managed resource.
type ObservedState struct {
	ResourceID       ResourceID   `json:"resource_id"`
	Kind             ResourceKind `json:"kind"`
	Exists           bool         `json:"exists"`
	InstalledVersion string       `json:"installed_version,omitempty"`
	Path             string       `json:"path,omitempty"`
	Hash             string       `json:"hash,omitempty"`
	Drift            string       `json:"drift"` // "clean", "drift", "conflict", "missing"
	Healthy          bool         `json:"healthy"`
	LastObserved     time.Time    `json:"last_observed"`
}

// DesiredState expresses the requested goal for a resource.
type DesiredState struct {
	ResourceID ResourceID   `json:"resource_id"`
	Kind       ResourceKind `json:"kind"`
	Version    string       `json:"version,omitempty"`
	State      string       `json:"state"` // "present", "absent", "updated"
}

// PlanAction defines an atomic, immutable planned mutation.
type PlanAction struct {
	ID            int          `json:"id"`
	ResourceID    ResourceID   `json:"resource_id"`
	Kind          ResourceKind `json:"kind"`
	Action        ActionType   `json:"action"`
	Description   string       `json:"description"`
	Provider      string       `json:"provider,omitempty"`
	Strategy      string       `json:"strategy,omitempty"`
	SourcePath    string       `json:"source_path,omitempty"`
	TargetPath    string       `json:"target_path,omitempty"`
	DependsOn     []int        `json:"depends_on,omitempty"`
	Locks         []string     `json:"locks,omitempty"`
	Preconditions []string     `json:"preconditions,omitempty"`
	IsPrivileged  bool         `json:"is_privileged"`
	IsDestructive bool         `json:"is_destructive"`
	IsExclusive   bool         `json:"is_exclusive"`
	Reversibility string       `json:"reversibility,omitempty"` // "full", "compensating", "none"
}

// PlanSpec represents an immutable, reproducible specification of actions to be applied.
// Invariant: Two plans with identical desired, observations, and actions produce the exact same Digest.
type PlanSpec struct {
	SchemaVersion string            `json:"schema_version"`
	Platform      string            `json:"platform"`
	Profile       string            `json:"profile"`
	Actions       []PlanAction      `json:"actions"`
	Metadata      map[string]string `json:"metadata,omitempty"`
}

// Digest computes a deterministic SHA-256 fingerprint over the canonical content of PlanSpec.
func (p *PlanSpec) Digest() string {
	if p == nil {
		return ""
	}
	// Copy to ensure canonical ordering and omit timestamps
	actionsCopy := make([]PlanAction, len(p.Actions))
	copy(actionsCopy, p.Actions)
	sort.Slice(actionsCopy, func(i, j int) bool {
		if actionsCopy[i].ID != actionsCopy[j].ID {
			return actionsCopy[i].ID < actionsCopy[j].ID
		}
		return actionsCopy[i].ResourceID < actionsCopy[j].ResourceID
	})

	canonical := struct {
		SchemaVersion string       `json:"schema_version"`
		Platform      string       `json:"platform"`
		Profile       string       `json:"profile"`
		Actions       []PlanAction `json:"actions"`
	}{
		SchemaVersion: p.SchemaVersion,
		Platform:      p.Platform,
		Profile:       p.Profile,
		Actions:       actionsCopy,
	}

	bytes, err := json.Marshal(canonical)
	if err != nil {
		return fmt.Sprintf("digest-error-%d", len(p.Actions))
	}
	h := sha256.Sum256(bytes)
	return hex.EncodeToString(h[:])
}

// Job represents an execution job tracking progress of an applied PlanSpec.
type Job struct {
	ID               string     `json:"id"`
	PlanDigest       string     `json:"plan_digest"`
	Status           StepStatus `json:"status"`
	CreatedAt        time.Time  `json:"created_at"`
	StartedAt        time.Time  `json:"started_at"`
	CompletedAt      time.Time  `json:"completed_at,omitempty"`
	TotalActions     int        `json:"total_actions"`
	CompletedActions int        `json:"completed_actions"`
	Error            string     `json:"error,omitempty"`
}

// Event represents a sequenced lifecycle notification emitted during execution.
type Event struct {
	Cursor     int64        `json:"cursor"`
	JobID      string       `json:"job_id"`
	Timestamp  time.Time    `json:"timestamp"`
	Type       string       `json:"type"` // "job_start", "action_start", "action_done", "job_done", "error"
	ActionID   int          `json:"action_id,omitempty"`
	ResourceID ResourceID   `json:"resource_id,omitempty"`
	Status     StepStatus   `json:"status"`
	Message    string       `json:"message,omitempty"`
}

// Provenance explains why a setting, package, or configuration was chosen and where it came from.
type Provenance struct {
	ResourceID         ResourceID   `json:"resource_id"`
	Kind               ResourceKind `json:"kind"`
	Origin             string       `json:"origin"`
	ActiveProfile      string       `json:"active_profile,omitempty"`
	SelectedBy         []string     `json:"selected_by,omitempty"`
	Conditions         []string     `json:"conditions,omitempty"`
	ExcludedCandidates []string     `json:"excluded_candidates,omitempty"`
	WinningPolicy      string       `json:"winning_policy,omitempty"`
	Dependencies       []ResourceID `json:"dependencies,omitempty"`
	Reason             string       `json:"reason"`
}
