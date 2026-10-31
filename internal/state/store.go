package state

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"
)

const stateFileName = "state.json"

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

type persistedState struct {
	SchemaVersion int                          `json:"schema_version"`
	ActiveProfile string                       `json:"active_profile,omitempty"`
	ActiveTheme   string                       `json:"active_theme,omitempty"`
	RepoRoot      string                       `json:"repo_root,omitempty"`
	Resources     map[string]InstalledRecord   `json:"resources"`
	ConfigBases   map[string]ConfigBaseRecord  `json:"config_bases"`
	AppliedHacks  map[string]AppliedHackRecord `json:"applied_hacks"`
}

// Store manages transactional atomic persistence of state.json inside the anx state directory.
// Concurrency model:
// 1. In-process: sync.RWMutex is held across the in-memory mutation and disk persistence,
//    guaranteeing strict linearizability and eliminating out-of-order write clobbers.
// 2. Cross-process: An exclusive file-lock on the state directory serializes mutations across
//    concurrent processes and reloads fresh on-disk state before mutating.
type Store struct {
	mu          sync.RWMutex
	filePath    string
	lastModTime time.Time
	data        persistedState
}

// OpenStore opens or initializes state.json under stateDir.
func OpenStore(stateDir string) (*Store, error) {
	if err := os.MkdirAll(stateDir, 0o700); err != nil {
		return nil, fmt.Errorf("creating state directory: %w", err)
	}

	s := &Store{
		filePath: filepath.Join(stateDir, stateFileName),
		data: persistedState{
			SchemaVersion: 1,
			Resources:     map[string]InstalledRecord{},
			ConfigBases:   map[string]ConfigBaseRecord{},
			AppliedHacks:  map[string]AppliedHackRecord{},
		},
	}

	if err := s.reloadFromDisk(); err != nil {
		return nil, err
	}
	return s, nil
}

// reloadFromDisk re-reads state.json from disk if modified, initializing empty maps if needed.
// Must be called with s.mu held or before concurrent exposure.
func (s *Store) reloadFromDisk() error {
	fi, err := os.Stat(s.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			s.lastModTime = time.Time{}
			return nil
		}
		return fmt.Errorf("stat state file: %w", err)
	}

	// Fast path: file has not changed since last load
	if !s.lastModTime.IsZero() && fi.ModTime().Equal(s.lastModTime) {
		return nil
	}

	raw, err := os.ReadFile(s.filePath)
	if err != nil {
		return fmt.Errorf("reading state file: %w", err)
	}
	if len(raw) == 0 {
		return nil
	}

	var d persistedState
	if err := json.Unmarshal(raw, &d); err != nil {
		return fmt.Errorf("parsing state file: %w", err)
	}
	if d.Resources == nil {
		d.Resources = map[string]InstalledRecord{}
	}
	if d.ConfigBases == nil {
		d.ConfigBases = map[string]ConfigBaseRecord{}
	}
	if d.AppliedHacks == nil {
		d.AppliedHacks = map[string]AppliedHackRecord{}
	}

	s.data = d
	s.lastModTime = fi.ModTime()
	return nil
}

// withWriteLock acquires an in-process write lock, reloads if disk state changed,
// runs the mutation function fn, marshals and atomically persists before releasing lock.
func (s *Store) withWriteLock(fn func() error) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Cross-process synchronization: retryable file lock around mutation and write
	lockDir := filepath.Dir(s.filePath)
	var fl *FileLock
	var err error
	for range 50 {
		fl, err = AcquireLock(lockDir, "store_write")
		if err == nil {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if fl != nil {
		defer func() { _ = fl.Release() }()
	}

	// Always sync with latest on-disk state before mutating
	_ = s.reloadFromDisk()

	if err := fn(); err != nil {
		return err
	}

	snapshot, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling state: %w", err)
	}

	if err := s.atomicWrite(snapshot); err != nil {
		return err
	}

	if fi, err := os.Stat(s.filePath); err == nil {
		s.lastModTime = fi.ModTime()
	}
	return nil
}

// RecordResource upserts an InstalledRecord and flushes atomically to disk.
func (s *Store) RecordResource(rec InstalledRecord) error {
	return s.withWriteLock(func() error {
		now := time.Now().UTC()
		existing, exists := s.data.Resources[rec.ResourceID]
		if exists && !existing.InstalledAt.IsZero() {
			rec.InstalledAt = existing.InstalledAt
		} else if rec.InstalledAt.IsZero() {
			rec.InstalledAt = now
		}
		rec.UpdatedAt = now
		s.data.Resources[rec.ResourceID] = rec
		return nil
	})
}

// RemoveResource deletes a resource record from state.json.
func (s *Store) RemoveResource(resourceID string) error {
	return s.withWriteLock(func() error {
		delete(s.data.Resources, resourceID)
		return nil
	})
}

// GetResource returns the InstalledRecord for resourceID if tracked.
func (s *Store) GetResource(resourceID string) (InstalledRecord, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rec, ok := s.data.Resources[resourceID]
	return rec, ok
}

// AllResources returns a copy of all tracked InstalledRecords.
func (s *Store) AllResources() map[string]InstalledRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make(map[string]InstalledRecord, len(s.data.Resources))
	for k, v := range s.data.Resources {
		out[k] = v
	}
	return out
}

// DetectOrphaned compares tracked installed resources against a slice of active/desired resource IDs.
// It returns all InstalledRecords present in state.json that are no longer part of active/desired manifests.
func (s *Store) DetectOrphaned(desiredResourceIDs []string) []InstalledRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()

	desiredMap := make(map[string]bool, len(desiredResourceIDs))
	for _, id := range desiredResourceIDs {
		desiredMap[id] = true
	}

	orphans := make([]InstalledRecord, 0)
	for id, rec := range s.data.Resources {
		if !desiredMap[id] {
			orphans = append(orphans, rec)
		}
	}
	slices.SortFunc(orphans, func(a, b InstalledRecord) int {
		return strings.Compare(a.ResourceID, b.ResourceID)
	})
	return orphans
}

// RecordConfigBase records the canonical ancestor content for 3-way dotfile merges.
func (s *Store) RecordConfigBase(configID string, content []byte) error {
	sum := sha256.Sum256(content)
	return s.withWriteLock(func() error {
		s.data.ConfigBases[configID] = ConfigBaseRecord{
			ConfigID:   configID,
			SHA256:     hex.EncodeToString(sum[:]),
			Content:    string(content),
			RecordedAt: time.Now().UTC(),
		}
		return nil
	})
}

// GetConfigBase returns the persisted base content for configID if previously synced.
func (s *Store) GetConfigBase(configID string) ([]byte, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rec, ok := s.data.ConfigBases[configID]
	if !ok {
		return nil, false
	}
	return []byte(rec.Content), true
}

// RecordHack persists applied hack steps for a workstation hack ID.
func (s *Store) RecordHack(hackID string, steps []string) error {
	copied := make([]string, len(steps))
	copy(copied, steps)

	return s.withWriteLock(func() error {
		s.data.AppliedHacks[hackID] = AppliedHackRecord{
			HackID:    hackID,
			Steps:     copied,
			AppliedAt: time.Now().UTC(),
		}
		return nil
	})
}

// IsHackApplied reports whether hackID has been applied on this machine.
func (s *Store) IsHackApplied(hackID string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.data.AppliedHacks[hackID]
	return ok
}

// SetActiveProfile updates the persisted active profile.
func (s *Store) SetActiveProfile(profile string) error {
	return s.withWriteLock(func() error {
		s.data.ActiveProfile = profile
		return nil
	})
}

// ActiveProfile returns the persisted active profile name or "developer" default.
func (s *Store) ActiveProfile() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.data.ActiveProfile == "" {
		return "developer"
	}
	return s.data.ActiveProfile
}

// SetActiveTheme updates the persisted active UI theme.
func (s *Store) SetActiveTheme(themeName string) error {
	return s.withWriteLock(func() error {
		s.data.ActiveTheme = themeName
		return nil
	})
}

// ActiveTheme returns the persisted UI theme or empty string if unset.
func (s *Store) ActiveTheme() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.data.ActiveTheme
}

// SetRepoRoot updates the persisted repository root directory.
func (s *Store) SetRepoRoot(path string) error {
	return s.withWriteLock(func() error {
		s.data.RepoRoot = strings.TrimSpace(path)
		return nil
	})
}

// RepoRoot returns the persisted repository root directory or empty string.
func (s *Store) RepoRoot() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.data.RepoRoot
}

// Reset clears all installed resources, config bases, and applied hacks while preserving schema version.
func (s *Store) Reset() error {
	return s.withWriteLock(func() error {
		s.data.Resources = make(map[string]InstalledRecord)
		s.data.ConfigBases = make(map[string]ConfigBaseRecord)
		s.data.AppliedHacks = make(map[string]AppliedHackRecord)
		s.data.ActiveProfile = ""
		return nil
	})
}

func (s *Store) atomicWrite(payload []byte) error {
	dir := filepath.Dir(s.filePath)
	tmpFile, err := os.CreateTemp(dir, "state-*.json.tmp")
	if err != nil {
		return fmt.Errorf("creating temp state file: %w", err)
	}
	tmpPath := tmpFile.Name()

	if _, err := tmpFile.Write(payload); err != nil {
		_ = tmpFile.Close()
		_ = os.Remove(tmpPath)
		return fmt.Errorf("writing temp state file: %w", err)
	}
	if err := tmpFile.Sync(); err != nil {
		_ = tmpFile.Close()
		_ = os.Remove(tmpPath)
		return fmt.Errorf("syncing temp state file: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("closing temp state file: %w", err)
	}
	if err := os.Chmod(tmpPath, 0o600); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("chmod temp state file: %w", err)
	}
	if err := os.Rename(tmpPath, s.filePath); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("renaming state file: %w", err)
	}

	// Durable parent directory sync
	if d, err := os.Open(dir); err == nil {
		_ = d.Sync()
		_ = d.Close()
	}
	return nil
}
