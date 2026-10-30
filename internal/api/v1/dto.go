package v1

import (
	"time"

	"github.com/arturonavax/env/internal/domain"
)

// CurrentSchemaVersion defines the v1 API schema version string.
const CurrentSchemaVersion = "anx.dev/api/v1"

// Envelope wraps all v1 API responses with transport-agnostic traceability metadata.
type Envelope[T any] struct {
	SchemaVersion string    `json:"schema_version"`
	HostID        string    `json:"host_id"`
	HostRevision  string    `json:"host_revision"`
	Timestamp     time.Time `json:"timestamp"`
	Data          T         `json:"data"`
}

// NewEnvelope creates a standard Envelope with timestamp and schema version.
func NewEnvelope[T any](hostID, hostRevision string, data T) Envelope[T] {
	return Envelope[T]{
		SchemaVersion: CurrentSchemaVersion,
		HostID:        hostID,
		HostRevision:  hostRevision,
		Timestamp:     time.Now().UTC(),
		Data:          data,
	}
}

// PlanRequestDTO represents a request to calculate a reproducible PlanSpec.
type PlanRequestDTO struct {
	Profile        string   `json:"profile"`
	RequestedIDs   []string `json:"requested_ids,omitempty"`
	IncludeHacks   bool     `json:"include_hacks"`
	DryRun         bool     `json:"dry_run"`
	IdempotencyKey string   `json:"idempotency_key,omitempty"`
}

// PlanResponseDTO represents the computed plan and its cryptographic digest.
type PlanResponseDTO struct {
	PlanDigest string          `json:"plan_digest"`
	Profile    string          `json:"profile"`
	Spec       domain.PlanSpec `json:"spec"`
	StepCount  int             `json:"step_count"`
}

// ApplyRequestDTO authorizes applying a precomputed PlanSpec.
type ApplyRequestDTO struct {
	PlanDigest     string `json:"plan_digest"`
	ExpectedRev    string `json:"expected_revision,omitempty"`
	DryRun         bool   `json:"dry_run"`
	IdempotencyKey string `json:"idempotency_key"`
}

// ApplyResponseDTO returns tracking metadata for the launched job.
type ApplyResponseDTO struct {
	JobID      string `json:"job_id"`
	PlanDigest string `json:"plan_digest"`
	Status     string `json:"status"`
}

// JobEventDTO represents a streaming progress event emitted during plan execution.
type JobEventDTO struct {
	JobID         string    `json:"job_id"`
	EventSequence int64     `json:"event_sequence"`
	StepID        int       `json:"step_id"`
	ResourceID    string    `json:"resource_id"`
	Action        string    `json:"action"`
	Status        string    `json:"status"`
	Timestamp     time.Time `json:"timestamp"`
	Error         string    `json:"error,omitempty"`
}
