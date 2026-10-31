package state

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/arturonavax/env/internal/storage/sqlite"
)

// InstalledRecord tracks ownership and metadata for a resource provisioned by anx.
type InstalledRecord struct {
	ResourceID    string    `json:"resource_id"`
	Version       string    `json:"version,omitempty"`
	Strategy      string    `json:"strategy"`
	Provider      string    `json:"provider"`
	ManagedByAnx  bool      `json:"managed_by_anx"`
	InstalledPath string    `json:"installed_path,omitempty"`
	InstalledAt   time.Time `json:"installed_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// ConfigBaseRecord stores the last deployed ancestor of a managed config file for true 3-way merges.
type ConfigBaseRecord struct {
	ConfigID   string    `json:"config_id"`
	SHA256     string    `json:"sha256"`
	Content    string    `json:"content"`
	RecordedAt time.Time `json:"recorded_at"`
}

// AppliedHackRecord tracks machine-specific optimization profiles applied on this workstation.
type AppliedHackRecord struct {
	HackID    string    `json:"hack_id"`
	Steps     []string  `json:"steps"`
	AppliedAt time.Time `json:"applied_at"`
}

// Store manages transactional atomic persistence of state inside the anx state directory
// backed by the embedded transactional SQLite engine (anx.db) in WAL mode.
type Store struct {
	mu       sync.RWMutex
	stateDir string
	db       *sqlite.DB
}

// OpenStore opens or initializes anx.db under stateDir, automatically migrating any
// existing legacy state.json without data loss.
func OpenStore(stateDir string) (*Store, error) {
	if err := os.MkdirAll(stateDir, 0o700); err != nil {
		return nil, fmt.Errorf("creating state directory: %w", err)
	}

	dbPath := filepath.Join(stateDir, "anx.db")
	db, err := sqlite.Open(context.Background(), sqlite.Options{
		Path: dbPath,
	})
	if err != nil {
		return nil, fmt.Errorf("opening state database: %w", err)
	}

	// Automatic zero-data-loss migration if legacy state.json exists
	if _, err := sqlite.ApplyMigration(context.Background(), stateDir, db); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("migrating legacy state: %w", err)
	}

	return &Store{
		stateDir: stateDir,
		db:       db,
	}, nil
}

// DB returns the underlying SQLite database handle.
func (s *Store) DB() *sqlite.DB {
	return s.db
}

// Close closes the underlying SQLite database cleanly.
func (s *Store) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// RecordResource upserts an InstalledRecord transactionally in the database.
func (s *Store) RecordResource(rec InstalledRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.db.UpsertResource(context.Background(), sqlite.ResourceRecord{
		ResourceID:    rec.ResourceID,
		Kind:          "package",
		Provider:      rec.Provider,
		Strategy:      rec.Strategy,
		Version:       rec.Version,
		ManagedByAnx:  rec.ManagedByAnx,
		InstalledPath: rec.InstalledPath,
		InstalledAt:   rec.InstalledAt,
		UpdatedAt:     rec.UpdatedAt,
	})
}

// RemoveResource deletes a resource record by ID.
func (s *Store) RemoveResource(resourceID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.db.DeleteResource(context.Background(), resourceID)
}

// GetResource returns the InstalledRecord for resourceID if tracked.
func (s *Store) GetResource(resourceID string) (InstalledRecord, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rec, ok, err := s.db.GetResource(context.Background(), resourceID)
	if err != nil || !ok {
		return InstalledRecord{}, false
	}

	return InstalledRecord{
		ResourceID:    rec.ResourceID,
		Version:       rec.Version,
		Strategy:      rec.Strategy,
		Provider:      rec.Provider,
		ManagedByAnx:  rec.ManagedByAnx,
		InstalledPath: rec.InstalledPath,
		InstalledAt:   rec.InstalledAt,
		UpdatedAt:     rec.UpdatedAt,
	}, true
}

// AllResources returns a copy of all tracked InstalledRecords.
func (s *Store) AllResources() map[string]InstalledRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()

	all, err := s.db.AllResources(context.Background())
	if err != nil {
		return make(map[string]InstalledRecord)
	}

	out := make(map[string]InstalledRecord, len(all))
	for id, rec := range all {
		out[id] = InstalledRecord{
			ResourceID:    rec.ResourceID,
			Version:       rec.Version,
			Strategy:      rec.Strategy,
			Provider:      rec.Provider,
			ManagedByAnx:  rec.ManagedByAnx,
			InstalledPath: rec.InstalledPath,
			InstalledAt:   rec.InstalledAt,
			UpdatedAt:     rec.UpdatedAt,
		}
	}
	return out
}

// DetectOrphaned compares tracked installed resources against active/desired resource IDs.
func (s *Store) DetectOrphaned(desiredResourceIDs []string) []InstalledRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()

	orphans, err := s.db.DetectOrphaned(context.Background(), desiredResourceIDs)
	if err != nil {
		return nil
	}

	out := make([]InstalledRecord, len(orphans))
	for i, o := range orphans {
		out[i] = InstalledRecord{
			ResourceID:    o.ResourceID,
			Version:       o.Version,
			Strategy:      o.Strategy,
			Provider:      o.Provider,
			ManagedByAnx:  o.ManagedByAnx,
			InstalledPath: o.InstalledPath,
			InstalledAt:   o.InstalledAt,
			UpdatedAt:     o.UpdatedAt,
		}
	}
	return out
}

// RecordConfigBase records the canonical ancestor content for 3-way dotfile merges.
func (s *Store) RecordConfigBase(configID string, content []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	sum := sha256.Sum256(content)
	return s.db.UpsertConfigBase(context.Background(), sqlite.ConfigBaseRecord{
		ConfigID:      configID,
		ContentDigest: hex.EncodeToString(sum[:]),
		Size:          int64(len(content)),
		EOL:           "lf",
		Content:       string(content),
	})
}

// GetConfigBase returns the persisted base content for configID if previously synced.
func (s *Store) GetConfigBase(configID string) ([]byte, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	cfg, ok, err := s.db.GetConfigBase(context.Background(), configID)
	if err != nil || !ok {
		return nil, false
	}
	return []byte(cfg.Content), true
}

// RecordHack persists applied hack steps for a workstation hack ID.
func (s *Store) RecordHack(hackID string, steps []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.db.RecordHack(context.Background(), sqlite.AppliedHackRecord{
		HackID: hackID,
		Steps:  steps,
	})
}

// IsHackApplied reports whether hackID has been applied on this machine.
func (s *Store) IsHackApplied(hackID string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	applied, err := s.db.IsHackApplied(context.Background(), hackID)
	if err != nil {
		return false
	}
	return applied
}

// SetActiveProfile updates the persisted active profile.
func (s *Store) SetActiveProfile(profile string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.db.SetMetadata(context.Background(), "active_profile", profile)
}

// ActiveProfile returns the persisted active profile name or "developer" default.
func (s *Store) ActiveProfile() string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	val, ok, err := s.db.GetMetadata(context.Background(), "active_profile")
	if err != nil || !ok || val == "" {
		return "developer"
	}
	return val
}

// SetActiveTheme updates the persisted active UI theme.
func (s *Store) SetActiveTheme(themeName string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.db.SetMetadata(context.Background(), "active_theme", themeName)
}

// ActiveTheme returns the persisted UI theme or empty string if unset.
func (s *Store) ActiveTheme() string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	val, ok, err := s.db.GetMetadata(context.Background(), "active_theme")
	if err != nil || !ok {
		return ""
	}
	return val
}

// SetRepoRoot updates the persisted repository root directory.
func (s *Store) SetRepoRoot(path string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.db.SetMetadata(context.Background(), "repo_root", strings.TrimSpace(path))
}

// RepoRoot returns the persisted repository root directory or empty string.
func (s *Store) RepoRoot() string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	val, ok, err := s.db.GetMetadata(context.Background(), "repo_root")
	if err != nil || !ok {
		return ""
	}
	return val
}

// Reset clears all installed resources, config bases, and applied hacks while preserving schema version.
func (s *Store) Reset() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.db.Reset(context.Background())
}
