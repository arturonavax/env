package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"
)

// ErrPlanDigest indicates an error computing the canonical digest of a PlanSpec.
var ErrPlanDigest = errors.New("computing canonical plan digest failed")

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

// ComputeDigest computes a deterministic SHA-256 fingerprint over the canonical content of PlanSpec,
// returning a typed error on nil input or serialization failure.
func (p *PlanSpec) ComputeDigest() (string, error) {
	if p == nil {
		return "", fmt.Errorf("%w: plan is nil", ErrPlanDigest)
	}

	// Canonical deep copy of actions
	actionsCopy := make([]PlanAction, len(p.Actions))
	for i, a := range p.Actions {
		act := a
		if len(act.DependsOn) > 1 {
			act.DependsOn = append([]int(nil), act.DependsOn...)
			sort.Ints(act.DependsOn)
		}
		if len(act.Locks) > 1 {
			act.Locks = append([]string(nil), act.Locks...)
			sort.Strings(act.Locks)
		}
		if len(act.Preconditions) > 1 {
			act.Preconditions = append([]string(nil), act.Preconditions...)
			sort.Strings(act.Preconditions)
		}
		actionsCopy[i] = act
	}

	sort.Slice(actionsCopy, func(i, j int) bool {
		if actionsCopy[i].ID != actionsCopy[j].ID {
			return actionsCopy[i].ID < actionsCopy[j].ID
		}
		return actionsCopy[i].ResourceID < actionsCopy[j].ResourceID
	})

	// Canonical sorted metadata
	metaKV := make([][2]string, 0, len(p.Metadata))
	if len(p.Metadata) > 0 {
		keys := make([]string, 0, len(p.Metadata))
		for k := range p.Metadata {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			metaKV = append(metaKV, [2]string{k, p.Metadata[k]})
		}
	}

	canonical := struct {
		SchemaVersion string       `json:"schema_version"`
		Platform      string       `json:"platform"`
		Profile       string       `json:"profile"`
		Actions       []PlanAction `json:"actions"`
		Metadata      [][2]string  `json:"metadata,omitempty"`
	}{
		SchemaVersion: p.SchemaVersion,
		Platform:      p.Platform,
		Profile:       p.Profile,
		Actions:       actionsCopy,
		Metadata:      metaKV,
	}

	data, err := json.Marshal(canonical)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrPlanDigest, err)
	}
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:]), nil
}

// Digest computes a deterministic SHA-256 fingerprint over the canonical content of PlanSpec.
func (p *PlanSpec) Digest() string {
	d, err := p.ComputeDigest()
	if err != nil {
		if p == nil {
			return ""
		}
		panic(err)
	}
	return d
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
