package domain

import "time"

// Severity indicates the threat level of a security vulnerability.
type Severity int

const (
	SeverityUnknown Severity = iota
	SeverityLow
	SeverityMedium
	SeverityHigh
	SeverityCritical
)

func (s Severity) String() string {
	switch s {
	case SeverityLow:
		return "low"
	case SeverityMedium:
		return "medium"
	case SeverityHigh:
		return "high"
	case SeverityCritical:
		return "critical"
	default:
		return "unknown"
	}
}

// AuditFinding represents a verified vulnerability discovered on a managed tool.
type AuditFinding struct {
	ID              string    `json:"id"` // CVE or GHSA ID
	ResourceID      string    `json:"resource_id"`
	ToolName        string    `json:"tool_name"`
	DetectedVersion string    `json:"detected_version"`
	FixedVersion    string    `json:"fixed_version,omitempty"`
	Severity        Severity  `json:"severity"`
	Summary         string    `json:"summary"`
	AdvisoryURL     string    `json:"advisory_url"`
	DiscoveredAt    time.Time `json:"discovered_at"`
}
