package sqlite

import (
	"time"
)

// ResourceRecord matches the installed resource metadata persisted in the database.
type ResourceRecord struct {
	ResourceID    string    `json:"resource_id"`
	Kind          string    `json:"kind"`
	Provider      string    `json:"provider"`
	Strategy      string    `json:"strategy"`
	Version       string    `json:"version,omitempty"`
	ManagedByAnx  bool      `json:"managed_by_anx"`
	InstalledPath string    `json:"installed_path,omitempty"`
	Origin        string    `json:"origin,omitempty"`
	InstalledAt   time.Time `json:"installed_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	Generation    int       `json:"generation"`
}

// ConfigBaseRecord stores the last deployed ancestor of a managed config file for 3-way merges.
type ConfigBaseRecord struct {
	ConfigID        string    `json:"config_id"`
	ContentDigest   string    `json:"content_digest"`
	Size            int64     `json:"size"`
	EOL             string    `json:"eol"`
	AppliedRevision int       `json:"applied_revision"`
	UpdatedAt       time.Time `json:"updated_at"`
	Content         string    `json:"content"`
}

// AppliedHackRecord tracks machine-specific optimization profiles applied on this workstation.
type AppliedHackRecord struct {
	HackID          string    `json:"hack_id"`
	HostFingerprint string    `json:"host_fingerprint,omitempty"`
	Steps           []string  `json:"steps"`
	AppliedAt       time.Time `json:"applied_at"`
	Version         string    `json:"version,omitempty"`
}

// MigrationMarker records provenance when migrating legacy state.json into SQLite.
type MigrationMarker struct {
	SourceSHA256  string    `json:"source_sha256"`
	SchemaVersion int       `json:"schema_version"`
	ImportedAt    time.Time `json:"imported_at"`
}

// PlanRecord persists immutable plan authorization and metadata.
type PlanRecord struct {
	PlanID                string     `json:"plan_id"`
	SchemaVersion         int        `json:"schema_version"`
	CanonicalIRDigest     string     `json:"canonical_ir_digest"`
	BaseHostRevision      string     `json:"base_host_revision,omitempty"`
	ConfigCatalogRevision string     `json:"config_catalog_revision,omitempty"`
	PolicyRevision        string     `json:"policy_revision,omitempty"`
	Target                string     `json:"target,omitempty"`
	CreatedAt             time.Time  `json:"created_at"`
	ExpiresAt             time.Time  `json:"expires_at"`
	AuthorizedAt          *time.Time `json:"authorized_at,omitempty"`
	AuthorizedBy          string     `json:"authorized_by,omitempty"`
	Status                string     `json:"status"`
}

// JobRecord represents durable background and foreground execution state.
type JobRecord struct {
	JobID          string     `json:"job_id"`
	PlanID         string     `json:"plan_id,omitempty"`
	IdempotencyKey string     `json:"idempotency_key,omitempty"`
	State          string     `json:"state"` // "queued", "running", "recovering", "partial", "succeeded", "failed", "cancelled"
	Cursor         int        `json:"cursor"`
	QueuedAt       time.Time  `json:"queued_at"`
	StartedAt      *time.Time `json:"started_at,omitempty"`
	FinishedAt     *time.Time `json:"finished_at,omitempty"`
	Attempt        int        `json:"attempt"`
	ExitErrorCode  int        `json:"exit_error_code,omitempty"`
	ResumeToken    string     `json:"resume_token,omitempty"`
}

// EventRecord represents append-only audit and execution telemetry events.
type EventRecord struct {
	Seq           int64     `json:"seq"`
	JobID         string    `json:"job_id,omitempty"`
	EventType     string    `json:"event_type"`
	SchemaVersion int       `json:"schema_version"`
	PayloadJSON   string    `json:"payload_json"`
	Timestamp     time.Time `json:"timestamp"`
	CorrelationID string    `json:"correlation_id,omitempty"`
	Digest        string    `json:"digest,omitempty"`
}
