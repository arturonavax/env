package snapshot

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/arturonavax/env/internal/domain"
	"github.com/arturonavax/env/internal/security"
)

// DefaultMaxQuotaBytes defines the 50 MB snapshot storage rotation quota.
const DefaultMaxQuotaBytes int64 = 50 * 1024 * 1024

// MinRetainedSnapshots defines the minimum number of snapshots preserved regardless of quota.
const MinRetainedSnapshots = 3

// ErrNoSnapshots indicates no snapshots exist in the backup directory.
var ErrNoSnapshots = errors.New("no snapshots available")

// FileDigestEntry records per-file backup location, permissions, size, and SHA-256 integrity digest.
type FileDigestEntry struct {
	BackupPath string      `json:"backup_path"`
	RelPayload string      `json:"rel_payload"`
	SHA256     string      `json:"sha256"`
	SizeBytes  int64       `json:"size_bytes"`
	Mode       os.FileMode `json:"mode"`
}

// snapshotManifest extends domain.Snapshot on disk while remaining wire-compatible with domain.Snapshot JSON readers.
type snapshotManifest struct {
	domain.Snapshot
	Entries     map[string]FileDigestEntry `json:"entries,omitempty"`
	TrackedDirs []string                   `json:"tracked_dirs,omitempty"`
	EmptyDirs   []string                   `json:"empty_dirs,omitempty"`
}

// Manager handles hierarchical backup snapshot creation, SHA-256 verification, quota rotation, and restoration.
type Manager struct {
	backupDir  string
	quotaBytes int64
	mu         sync.Mutex
}

// NewManager creates a snapshot manager anchored in stateDir/backups.
func NewManager(stateDir string) (*Manager, error) {
	if strings.TrimSpace(stateDir) == "" {
		return nil, security.ErrInvalidPath
	}
	return NewManagerAtDir(filepath.Join(stateDir, "backups"))
}

// NewManagerAtDir creates a snapshot manager directly inside backupDir.
func NewManagerAtDir(backupDir string) (*Manager, error) {
	if strings.TrimSpace(backupDir) == "" {
		return nil, security.ErrInvalidPath
	}
	absDir, err := filepath.Abs(backupDir)
	if err != nil {
		return nil, fmt.Errorf("resolving backup directory: %w", err)
	}
	if err := os.MkdirAll(absDir, 0o700); err != nil {
		return nil, fmt.Errorf("creating backups directory: %w", err)
	}
	return &Manager{
		backupDir:  absDir,
		quotaBytes: DefaultMaxQuotaBytes,
	}, nil
}

// BackupDir returns the root directory storing snapshots.
func (m *Manager) BackupDir() string {
	return m.backupDir
}

// SetQuotaBytes updates the maximum total snapshot quota in bytes (used for custom policies and tests).
func (m *Manager) SetQuotaBytes(maxBytes int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if maxBytes > 0 {
		m.quotaBytes = maxBytes
	}
}

// CreateSnapshot captures files and directories recursively under <snapDir>/payload/... with SHA-256 digests.
func (m *Manager) CreateSnapshot(desc string, filePaths []string) (*domain.Snapshot, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	ts := time.Now().UTC()
	snapID, snapDir, err := m.allocateSnapshotDirLocked(ts)
	if err != nil {
		return nil, err
	}

	payloadRoot := filepath.Join(snapDir, "payload")
	if err := os.MkdirAll(payloadRoot, 0o700); err != nil {
		_ = os.RemoveAll(snapDir)
		return nil, fmt.Errorf("creating snapshot payload directory: %w", err)
	}

	manifest := snapshotManifest{
		Snapshot: domain.Snapshot{
			ID:          snapID,
			Timestamp:   ts,
			Description: desc,
			Files:       map[string]string{},
		},
		Entries:     map[string]FileDigestEntry{},
		TrackedDirs: []string{},
		EmptyDirs:   []string{},
	}

	seenDirs := map[string]bool{}
	captureSingleFile := func(cleanSrc string, info os.FileInfo) error {
		if _, already := manifest.Entries[cleanSrc]; already {
			return nil
		}

		relPayload := deterministicPayloadSubpath(cleanSrc)
		destPath, err := security.SafePath(payloadRoot, relPayload)
		if err != nil {
			return fmt.Errorf("computing safe payload path for %q: %w", cleanSrc, err)
		}

		digest, written, err := copyFileWithSHA256(cleanSrc, destPath, info.Mode().Perm())
		if err != nil {
			return fmt.Errorf("copying %q into snapshot: %w", cleanSrc, err)
		}

		manifest.Files[cleanSrc] = destPath
		manifest.Entries[cleanSrc] = FileDigestEntry{
			BackupPath: destPath,
			RelPayload: filepath.ToSlash(relPayload),
			SHA256:     digest,
			SizeBytes:  written,
			Mode:       info.Mode().Perm(),
		}
		manifest.SizeBytes += written
		return nil
	}

	for _, rawSrc := range filePaths {
		if strings.TrimSpace(rawSrc) == "" {
			continue
		}
		absSrc, err := filepath.Abs(rawSrc)
		if err != nil {
			_ = os.RemoveAll(snapDir)
			return nil, fmt.Errorf("resolving path %q: %w", rawSrc, err)
		}
		cleanSrc := filepath.Clean(absSrc)

		info, err := os.Stat(cleanSrc)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			_ = os.RemoveAll(snapDir)
			return nil, fmt.Errorf("stat %q: %w", cleanSrc, err)
		}

		if !info.IsDir() {
			if err := captureSingleFile(cleanSrc, info); err != nil {
				_ = os.RemoveAll(snapDir)
				return nil, err
			}
			continue
		}

		if !seenDirs[cleanSrc] {
			seenDirs[cleanSrc] = true
			manifest.TrackedDirs = append(manifest.TrackedDirs, cleanSrc)
		}

		walkErr := filepath.WalkDir(cleanSrc, func(curr string, d fs.DirEntry, wErr error) error {
			if wErr != nil {
				return wErr
			}
			if d.IsDir() {
				entries, readErr := os.ReadDir(curr)
				if readErr == nil && len(entries) == 0 {
					manifest.EmptyDirs = append(manifest.EmptyDirs, curr)
				}
				return nil
			}
			if !d.Type().IsRegular() {
				return nil
			}
			fInfo, statErr := d.Info()
			if statErr != nil {
				return statErr
			}
			return captureSingleFile(curr, fInfo)
		})
		if walkErr != nil {
			_ = os.RemoveAll(snapDir)
			return nil, fmt.Errorf("walking directory %q for snapshot: %w", cleanSrc, walkErr)
		}
	}

	slices.Sort(manifest.TrackedDirs)
	slices.Sort(manifest.EmptyDirs)

	manifestPath := filepath.Join(snapDir, "manifest.json")
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		_ = os.RemoveAll(snapDir)
		return nil, fmt.Errorf("marshaling snapshot manifest: %w", err)
	}
	if err := writeAtomicManifest(manifestPath, data); err != nil {
		_ = os.RemoveAll(snapDir)
		return nil, fmt.Errorf("writing snapshot manifest: %w", err)
	}

	_ = m.enforceQuotaLocked()

	snapCopy := manifest.Snapshot
	return &snapCopy, nil
}

// ListSnapshots returns all available snapshots ordered from newest to oldest.
func (m *Manager) ListSnapshots() ([]domain.Snapshot, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.listSnapshotsLocked()
}

// VerifySnapshot validates the manifest and verifies SHA-256 digests for every backed-up file in snapID.
func (m *Manager) VerifySnapshot(snapID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	_, err := m.verifyAndLoadLocked(snapID)
	return err
}

// Restore verifies SHA-256 integrity of snapID and restores all files and directories to their original locations.
func (m *Manager) Restore(snapID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.restoreLocked(snapID)
}

// RestoreLatest verifies and restores the most recent snapshot, returning its metadata.
func (m *Manager) RestoreLatest() (*domain.Snapshot, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	snaps, err := m.listSnapshotsLocked()
	if err != nil {
		return nil, err
	}
	if len(snaps) == 0 {
		return nil, ErrNoSnapshots
	}

	latest := snaps[0]
	if err := m.restoreLocked(latest.ID); err != nil {
		return nil, err
	}
	return &latest, nil
}

func (m *Manager) allocateSnapshotDirLocked(ts time.Time) (string, string, error) {
	baseID := ts.Format("2006-01-02_15-04-05")
	snapID := baseID

	for suffix := 0; ; suffix++ {
		if suffix > 0 {
			snapID = fmt.Sprintf("%s-%d", baseID, suffix)
		}
		snapDir, err := security.SafePath(m.backupDir, snapID)
		if err != nil {
			return "", "", err
		}
		_, statErr := os.Stat(snapDir)
		if os.IsNotExist(statErr) {
			if mkErr := os.MkdirAll(snapDir, 0o700); mkErr != nil {
				return "", "", fmt.Errorf("creating snapshot directory: %w", mkErr)
			}
			return snapID, snapDir, nil
		}
		if statErr != nil {
			return "", "", fmt.Errorf("checking snapshot directory %q: %w", snapDir, statErr)
		}
	}
}

func (m *Manager) verifyAndLoadLocked(snapID string) (*snapshotManifest, error) {
	if strings.TrimSpace(snapID) == "" {
		return nil, security.ErrInvalidPath
	}
	snapDir, err := security.SafePath(m.backupDir, snapID)
	if err != nil {
		return nil, fmt.Errorf("invalid snapshot id %q: %w", snapID, err)
	}

	manifestPath := filepath.Join(snapDir, "manifest.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("reading snapshot %q: %w", snapID, err)
	}

	var manifest snapshotManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("parsing snapshot manifest %q: %w", snapID, err)
	}

	if len(manifest.Entries) > 0 {
		for origPath, entry := range manifest.Entries {
			safeBackup, err := security.SafePath(snapDir, entry.BackupPath)
			if err != nil {
				return nil, fmt.Errorf("snapshot entry %q escapes snapshot dir: %w", origPath, err)
			}
			if err := security.VerifyFileSHA256(safeBackup, entry.SHA256); err != nil {
				return nil, fmt.Errorf("integrity check failed for %q in snapshot %q: %w", origPath, snapID, err)
			}
		}
		return &manifest, nil
	}

	// Backward compatibility with legacy manifests lacking Entries map.
	for origPath, backupPath := range manifest.Files {
		safeBackup, err := security.SafePath(snapDir, backupPath)
		if err != nil {
			return nil, fmt.Errorf("snapshot file %q escapes snapshot dir: %w", origPath, err)
		}
		if _, err := os.Stat(safeBackup); err != nil {
			return nil, fmt.Errorf("missing payload for %q in snapshot %q: %w", origPath, snapID, err)
		}
	}

	return &manifest, nil
}

func (m *Manager) restoreLocked(snapID string) error {
	manifest, err := m.verifyAndLoadLocked(snapID)
	if err != nil {
		return err
	}

	// Remove stray files created after snapshot inside explicitly tracked directories.
	for _, trackedDir := range manifest.TrackedDirs {
		if info, statErr := os.Stat(trackedDir); statErr == nil && info.IsDir() {
			if cleanErr := pruneUntrackedFiles(trackedDir, manifest.Files); cleanErr != nil {
				return fmt.Errorf("cleaning untracked files in %q: %w", trackedDir, cleanErr)
			}
		}
	}

	for _, emptyDir := range manifest.EmptyDirs {
		if err := os.MkdirAll(emptyDir, 0o755); err != nil {
			return fmt.Errorf("restoring directory %q: %w", emptyDir, err)
		}
	}

	origPaths := make([]string, 0, len(manifest.Files))
	for origPath := range manifest.Files {
		origPaths = append(origPaths, origPath)
	}
	slices.Sort(origPaths)

	for _, origPath := range origPaths {
		backupPath := manifest.Files[origPath]
		mode := os.FileMode(0o644)
		if entry, ok := manifest.Entries[origPath]; ok && entry.Mode != 0 {
			mode = entry.Mode
		}
		if err := atomicRestoreFile(backupPath, origPath, mode); err != nil {
			return fmt.Errorf("restoring %q: %w", origPath, err)
		}
	}

	return nil
}

func (m *Manager) listSnapshotsLocked() ([]domain.Snapshot, error) {
	entries, err := os.ReadDir(m.backupDir)
	if err != nil {
		return nil, err
	}

	snapshots := make([]domain.Snapshot, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		manifestPath := filepath.Join(m.backupDir, e.Name(), "manifest.json")
		data, err := os.ReadFile(manifestPath)
		if err != nil {
			continue
		}
		var s domain.Snapshot
		if err := json.Unmarshal(data, &s); err == nil {
			if s.Files == nil {
				s.Files = map[string]string{}
			}
			snapshots = append(snapshots, s)
		}
	}

	sort.Slice(snapshots, func(i, j int) bool {
		if !snapshots[i].Timestamp.Equal(snapshots[j].Timestamp) {
			return snapshots[i].Timestamp.After(snapshots[j].Timestamp)
		}
		return compareSnapshotIDsDesc(snapshots[i].ID, snapshots[j].ID)
	})

	return snapshots, nil
}

func (m *Manager) enforceQuotaLocked() error {
	snaps, err := m.listSnapshotsLocked()
	if err != nil || len(snaps) <= MinRetainedSnapshots {
		return nil
	}

	var totalBytes int64
	for _, s := range snaps {
		totalBytes += s.SizeBytes
	}

	for i := len(snaps) - 1; i >= MinRetainedSnapshots && totalBytes > m.quotaBytes; i-- {
		oldestDir, err := security.SafePath(m.backupDir, snaps[i].ID)
		if err != nil {
			continue
		}
		if err := os.RemoveAll(oldestDir); err == nil {
			totalBytes -= snaps[i].SizeBytes
		}
	}

	return nil
}

func deterministicPayloadSubpath(cleanAbs string) string {
	vol := filepath.VolumeName(cleanAbs)
	rest := strings.TrimPrefix(cleanAbs, vol)
	rest = strings.TrimLeft(rest, `/\`)
	if vol != "" {
		cleanVol := strings.TrimSuffix(vol, ":")
		return filepath.Join(cleanVol, rest)
	}
	return rest
}

func copyFileWithSHA256(src, dst string, mode os.FileMode) (string, int64, error) {
	if mode == 0 {
		mode = 0o644
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o700); err != nil {
		return "", 0, err
	}

	in, err := os.Open(src)
	if err != nil {
		return "", 0, err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return "", 0, err
	}

	hasher := sha256.New()
	written, copyErr := io.Copy(io.MultiWriter(out, hasher), in)
	syncErr := out.Sync()
	closeErr := out.Close()
	if copyErr != nil || syncErr != nil || closeErr != nil {
		_ = os.Remove(dst)
		return "", 0, errors.Join(copyErr, syncErr, closeErr)
	}

	return hex.EncodeToString(hasher.Sum(nil)), written, nil
}

func atomicRestoreFile(srcBackup, dstOrig string, mode os.FileMode) error {
	if mode == 0 {
		mode = 0o644
	}
	parent := filepath.Dir(dstOrig)
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return err
	}

	in, err := os.Open(srcBackup)
	if err != nil {
		return err
	}
	defer in.Close()

	tmp, err := os.CreateTemp(parent, ".anx-restore-*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	renamed := false
	defer func() {
		if !renamed {
			_ = tmp.Close()
			_ = os.Remove(tmpPath)
		}
	}()

	if _, err := io.Copy(tmp, in); err != nil {
		return err
	}
	if err := tmp.Sync(); err != nil {
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmpPath, mode); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, dstOrig); err != nil {
		return err
	}
	renamed = true
	return nil
}

func writeAtomicManifest(manifestPath string, payload []byte) error {
	dir := filepath.Dir(manifestPath)
	tmp, err := os.CreateTemp(dir, ".manifest-*.json.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	renamed := false
	defer func() {
		if !renamed {
			_ = tmp.Close()
			_ = os.Remove(tmpPath)
		}
	}()

	if _, err := tmp.Write(payload); err != nil {
		return err
	}
	if err := tmp.Sync(); err != nil {
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmpPath, 0o600); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, manifestPath); err != nil {
		return err
	}
	renamed = true
	return nil
}

func pruneUntrackedFiles(trackedDir string, snapshotFiles map[string]string) error {
	return filepath.WalkDir(trackedDir, func(curr string, d fs.DirEntry, wErr error) error {
		if wErr != nil {
			return wErr
		}
		if d.IsDir() {
			return nil
		}
		if _, tracked := snapshotFiles[curr]; !tracked {
			if err := os.Remove(curr); err != nil && !os.IsNotExist(err) {
				return err
			}
		}
		return nil
	})
}

func compareSnapshotIDsDesc(idA, idB string) bool {
	baseA, seqA := splitSnapshotID(idA)
	baseB, seqB := splitSnapshotID(idB)
	if baseA != baseB {
		return baseA > baseB
	}
	return seqA > seqB
}

func splitSnapshotID(id string) (string, int) {
	// Format is YYYY-MM-DD_HH-MM-SS (19 chars) optionally followed by -N.
	if len(id) > 20 && id[19] == '-' {
		if n, err := strconv.Atoi(id[20:]); err == nil {
			return id[:19], n
		}
	}
	return id, 0
}
