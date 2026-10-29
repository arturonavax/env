package domain

// ConflictType categorizes the drift detected between base, local, and incoming revisions.
type ConflictType int

const (
	ConflictNone ConflictType = iota
	ConflictLocalModified
	ConflictIncomingModified
	ConflictBothModified
	ConflictDeletedLocal
	ConflictDeletedIncoming
)

func (c ConflictType) String() string {
	switch c {
	case ConflictNone:
		return "clean"
	case ConflictLocalModified:
		return "local_modified"
	case ConflictIncomingModified:
		return "incoming_modified"
	case ConflictBothModified:
		return "both_modified"
	case ConflictDeletedLocal:
		return "deleted_local"
	case ConflictDeletedIncoming:
		return "deleted_incoming"
	default:
		return "unknown"
	}
}

// MergeResolution represents the chosen resolution for a conflict.
type MergeResolution int

const (
	ResolutionAutoMerged MergeResolution = iota + 1
	ResolutionKeepLocal
	ResolutionTakeIncoming
	ResolutionManual
	ResolutionSkipped
)

func (r MergeResolution) String() string {
	switch r {
	case ResolutionAutoMerged:
		return "auto_merged"
	case ResolutionKeepLocal:
		return "keep_local"
	case ResolutionTakeIncoming:
		return "take_incoming"
	case ResolutionManual:
		return "manual"
	case ResolutionSkipped:
		return "skipped"
	default:
		return "unknown"
	}
}

// ThreeWayDiff contains the diff result of a 3-way dotfile merge.
type ThreeWayDiff struct {
	TargetPath      string          `json:"target_path"`
	ConflictType    ConflictType    `json:"conflict_type"`
	HasConflict     bool            `json:"has_conflict"`
	BaseContent     string          `json:"base_content,omitempty"`
	LocalContent    string          `json:"local_content,omitempty"`
	IncomingContent string          `json:"incoming_content,omitempty"`
	MergedContent   string          `json:"merged_content,omitempty"`
	Resolution      MergeResolution `json:"resolution,omitempty"`
}
