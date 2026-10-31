package sqlite

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// LegacyState represents the JSON structure of legacy state.json.
type LegacyState struct {
	SchemaVersion int    `json:"schema_version"`
	ActiveProfile string `json:"active_profile,omitempty"`
	ActiveTheme   string `json:"active_theme,omitempty"`
	RepoRoot      string `json:"repo_root,omitempty"`
	Resources     map[string]struct {
		ResourceID    string    `json:"resource_id"`
		Version       string    `json:"version,omitempty"`
		Strategy      string    `json:"strategy"`
		Provider      string    `json:"provider"`
		ManagedByAnx  bool      `json:"managed_by_anx"`
		InstalledPath string    `json:"installed_path,omitempty"`
		InstalledAt   time.Time `json:"installed_at"`
		UpdatedAt     time.Time `json:"updated_at"`
	} `json:"resources"`
	ConfigBases map[string]struct {
		ConfigID   string    `json:"config_id"`
		SHA256     string    `json:"sha256"`
		Content    string    `json:"content"`
		RecordedAt time.Time `json:"recorded_at"`
	} `json:"config_bases"`
	AppliedHacks map[string]struct {
		HackID    string    `json:"hack_id"`
		Steps     []string  `json:"steps"`
		AppliedAt time.Time `json:"applied_at"`
	} `json:"applied_hacks"`
}

// MigrationPlan represents the preview or completed summary of a state.json -> SQLite migration.
type MigrationPlan struct {
	SourcePath       string `json:"source_path"`
	SourceSHA256     string `json:"source_sha256"`
	ResourceCount    int    `json:"resource_count"`
	ConfigBaseCount  int    `json:"config_base_count"`
	AppliedHackCount int    `json:"applied_hack_count"`
	ActiveProfile    string `json:"active_profile,omitempty"`
	ActiveTheme      string `json:"active_theme,omitempty"`
	RepoRoot         string `json:"repo_root,omitempty"`
	BackupPath       string `json:"backup_path,omitempty"`
	AlreadyMigrated  bool   `json:"already_migrated"`
}

// PlanMigration reads state.json without modifying disk or writing to database.
func PlanMigration(stateDir string) (*MigrationPlan, *LegacyState, error) {
	jsonPath := filepath.Join(stateDir, "state.json")
	raw, err := os.ReadFile(jsonPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil, nil
		}
		return nil, nil, fmt.Errorf("reading legacy state.json: %w", err)
	}

	if len(raw) == 0 {
		return nil, nil, nil
	}

	var legacy LegacyState
	if err := json.Unmarshal(raw, &legacy); err != nil {
		return nil, nil, fmt.Errorf("parsing legacy state.json: %w", err)
	}

	sum := sha256.Sum256(raw)
	shaHex := hex.EncodeToString(sum[:])

	plan := &MigrationPlan{
		SourcePath:       jsonPath,
		SourceSHA256:     shaHex,
		ResourceCount:    len(legacy.Resources),
		ConfigBaseCount:  len(legacy.ConfigBases),
		AppliedHackCount: len(legacy.AppliedHacks),
		ActiveProfile:    legacy.ActiveProfile,
		ActiveTheme:      legacy.ActiveTheme,
		RepoRoot:         legacy.RepoRoot,
	}

	return plan, &legacy, nil
}

// ApplyMigration imports legacy state.json into SQLite within a single atomic transaction.
// It creates a recoverable backup before import and never deletes the original file.
func ApplyMigration(ctx context.Context, stateDir string, db *DB) (*MigrationPlan, error) {
	plan, legacy, err := PlanMigration(stateDir)
	if err != nil {
		return nil, err
	}
	if plan == nil {
		return nil, nil // Nothing to migrate
	}

	// Check if already migrated
	if _, ok, err := db.GetMigrationMarker(ctx, plan.SourceSHA256); err == nil && ok {
		plan.AlreadyMigrated = true
		return plan, nil
	}

	// Create timestamped backup: state.json.pre-sqlite.<timestamp>.bak
	ts := time.Now().UTC().Format("20060102-150405")
	backupName := fmt.Sprintf("state.json.pre-sqlite.%s.bak", ts)
	backupPath := filepath.Join(stateDir, backupName)

	raw, err := os.ReadFile(plan.SourcePath)
	if err != nil {
		return nil, fmt.Errorf("reading source for backup: %w", err)
	}

	if err := os.WriteFile(backupPath, raw, 0600); err != nil {
		return nil, fmt.Errorf("creating migration backup %s: %w", backupPath, err)
	}
	plan.BackupPath = backupPath

	// Atomic import transaction
	err = db.WriteTx(ctx, func(tx *sql.Tx) error {
		// 1. Resources
		resStmt, err := tx.PrepareContext(ctx, `
INSERT OR REPLACE INTO resources (
	resource_id, kind, provider, strategy, version, managed_by_anx,
	install_path, origin, installed_at, updated_at, generation
) VALUES (?, 'package', ?, ?, ?, ?, ?, '', ?, ?, 1);`)
		if err != nil {
			return fmt.Errorf("preparing resource insert: %w", err)
		}
		defer func() { _ = resStmt.Close() }()

		for _, r := range legacy.Resources {
			managed := 0
			if r.ManagedByAnx {
				managed = 1
			}
			instAt := r.InstalledAt.UTC().Unix()
			if instAt == 0 {
				instAt = time.Now().UTC().Unix()
			}
			updAt := r.UpdatedAt.UTC().Unix()
			if updAt == 0 {
				updAt = instAt
			}

			if _, err := resStmt.ExecContext(ctx,
				r.ResourceID,
				r.Provider,
				r.Strategy,
				r.Version,
				managed,
				r.InstalledPath,
				instAt,
				updAt,
			); err != nil {
				return fmt.Errorf("migrating resource %s: %w", r.ResourceID, err)
			}
		}

		// 2. Config Bases
		cfgStmt, err := tx.PrepareContext(ctx, `
INSERT OR REPLACE INTO config_bases (
	config_id, content_digest, size, eol, applied_revision, updated_at, content
) VALUES (?, ?, ?, 'lf', 1, ?, ?);`)
		if err != nil {
			return fmt.Errorf("preparing config_base insert: %w", err)
		}
		defer func() { _ = cfgStmt.Close() }()

		for _, c := range legacy.ConfigBases {
			updAt := c.RecordedAt.UTC().Unix()
			if updAt == 0 {
				updAt = time.Now().UTC().Unix()
			}
			if _, err := cfgStmt.ExecContext(ctx,
				c.ConfigID,
				c.SHA256,
				int64(len(c.Content)),
				updAt,
				c.Content,
			); err != nil {
				return fmt.Errorf("migrating config base %s: %w", c.ConfigID, err)
			}
		}

		// 3. Applied Hacks
		hackStmt, err := tx.PrepareContext(ctx, `
INSERT OR REPLACE INTO applied_hacks (
	hack_id, host_fingerprint, steps_json, applied_at, version
) VALUES (?, '', ?, ?, '');`)
		if err != nil {
			return fmt.Errorf("preparing applied_hack insert: %w", err)
		}
		defer func() { _ = hackStmt.Close() }()

		for _, h := range legacy.AppliedHacks {
			stepsJSON, _ := json.Marshal(h.Steps)
			appAt := h.AppliedAt.UTC().Unix()
			if appAt == 0 {
				appAt = time.Now().UTC().Unix()
			}
			if _, err := hackStmt.ExecContext(ctx,
				h.HackID,
				string(stepsJSON),
				appAt,
			); err != nil {
				return fmt.Errorf("migrating hack %s: %w", h.HackID, err)
			}
		}

		// 4. Metadata
		now := time.Now().UTC().Unix()
		metaStmt, err := tx.PrepareContext(ctx, `
INSERT OR REPLACE INTO metadata (key, value, updated_at, revision)
VALUES (?, ?, ?, 1);`)
		if err != nil {
			return fmt.Errorf("preparing metadata insert: %w", err)
		}
		defer func() { _ = metaStmt.Close() }()

		if legacy.ActiveProfile != "" {
			if _, err := metaStmt.ExecContext(ctx, "active_profile", legacy.ActiveProfile, now); err != nil {
				return err
			}
		}
		if legacy.ActiveTheme != "" {
			if _, err := metaStmt.ExecContext(ctx, "active_theme", legacy.ActiveTheme, now); err != nil {
				return err
			}
		}
		if legacy.RepoRoot != "" {
			if _, err := metaStmt.ExecContext(ctx, "repo_root", legacy.RepoRoot, now); err != nil {
				return err
			}
		}

		// 5. Migration Marker
		if _, err := tx.ExecContext(ctx, `
INSERT OR REPLACE INTO migration_marker (source_sha256, schema_version, imported_at)
VALUES (?, ?, ?);`,
			plan.SourceSHA256,
			legacy.SchemaVersion,
			now,
		); err != nil {
			return fmt.Errorf("recording migration marker: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("migration transaction failed: %w", err)
	}

	// Verify imported counts
	allRes, err := db.AllResources(ctx)
	if err != nil || len(allRes) < plan.ResourceCount {
		return nil, fmt.Errorf("verification failed: expected at least %d resources, got %d", plan.ResourceCount, len(allRes))
	}

	allCfg, err := db.AllConfigBases(ctx)
	if err != nil || len(allCfg) < plan.ConfigBaseCount {
		return nil, fmt.Errorf("verification failed: expected at least %d config bases, got %d", plan.ConfigBaseCount, len(allCfg))
	}

	return plan, nil
}
