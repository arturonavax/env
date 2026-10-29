package domain

import "time"

// Snapshot represents a point-in-time configuration snapshot.
type Snapshot struct {
	ID          string            `json:"id"`
	Timestamp   time.Time         `json:"timestamp"`
	Description string            `json:"description"`
	SizeBytes   int64             `json:"size_bytes"`
	Files       map[string]string `json:"files"` // relative path -> backup path
}

// BackupMetadata tracks backup file locations and checksums.
type BackupMetadata struct {
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	Reason    string    `json:"reason"`
	FileCount int       `json:"file_count"`
	SizeBytes int64     `json:"size_bytes"`
}
