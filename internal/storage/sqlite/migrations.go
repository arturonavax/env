package sqlite

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"
)

type migration struct {
	id  int
	sql string
}

var migrations = []migration{
	{
		id: 1,
		sql: `
CREATE TABLE IF NOT EXISTS schema_migrations (
	id INTEGER PRIMARY KEY,
	checksum TEXT NOT NULL,
	applied_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS metadata (
	key TEXT PRIMARY KEY,
	value TEXT NOT NULL,
	updated_at INTEGER NOT NULL,
	revision INTEGER NOT NULL DEFAULT 1
);

CREATE TABLE IF NOT EXISTS resources (
	resource_id TEXT PRIMARY KEY,
	kind TEXT NOT NULL DEFAULT 'package',
	provider TEXT NOT NULL,
	strategy TEXT NOT NULL,
	version TEXT,
	managed_by_anx INTEGER NOT NULL DEFAULT 1,
	install_path TEXT,
	origin TEXT,
	installed_at INTEGER NOT NULL,
	updated_at INTEGER NOT NULL,
	generation INTEGER NOT NULL DEFAULT 1
);

CREATE INDEX IF NOT EXISTS idx_resources_kind_provider ON resources(kind, provider, managed_by_anx);

CREATE TABLE IF NOT EXISTS config_bases (
	config_id TEXT PRIMARY KEY,
	content_digest TEXT NOT NULL,
	size INTEGER NOT NULL,
	eol TEXT NOT NULL DEFAULT 'lf',
	applied_revision INTEGER NOT NULL DEFAULT 1,
	updated_at INTEGER NOT NULL,
	content TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS applied_hacks (
	hack_id TEXT PRIMARY KEY,
	host_fingerprint TEXT,
	steps_json TEXT NOT NULL,
	applied_at INTEGER NOT NULL,
	version TEXT
);

CREATE INDEX IF NOT EXISTS idx_applied_hacks_host ON applied_hacks(host_fingerprint);

CREATE TABLE IF NOT EXISTS host_facts (
	host_id TEXT PRIMARY KEY,
	facts_json TEXT NOT NULL,
	fingerprint TEXT NOT NULL,
	observed_at INTEGER NOT NULL,
	expires_at INTEGER NOT NULL,
	confidence TEXT
);

CREATE TABLE IF NOT EXISTS plans (
	plan_id TEXT PRIMARY KEY,
	schema_version INTEGER NOT NULL,
	canonical_ir_digest TEXT NOT NULL,
	base_host_revision TEXT,
	config_catalog_revision TEXT,
	policy_revision TEXT,
	target TEXT,
	created_at INTEGER NOT NULL,
	expires_at INTEGER NOT NULL,
	authorized_at INTEGER,
	authorized_by TEXT,
	status TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_plans_status_created ON plans(status, created_at);

CREATE TABLE IF NOT EXISTS plan_operations (
	plan_id TEXT NOT NULL,
	op_index INTEGER NOT NULL,
	op_kind TEXT NOT NULL,
	target TEXT NOT NULL,
	dependencies_json TEXT,
	expected_hashes_json TEXT,
	owned_resource TEXT,
	compensation_json TEXT,
	redacted_args_json TEXT,
	side_effect_class TEXT,
	PRIMARY KEY (plan_id, op_index),
	FOREIGN KEY(plan_id) REFERENCES plans(plan_id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS jobs (
	job_id TEXT PRIMARY KEY,
	plan_id TEXT,
	idempotency_key TEXT UNIQUE,
	state TEXT NOT NULL,
	cursor INTEGER NOT NULL DEFAULT 0,
	queued_at INTEGER NOT NULL,
	started_at INTEGER,
	finished_at INTEGER,
	attempt INTEGER NOT NULL DEFAULT 1,
	exit_error_code INTEGER,
	resume_token TEXT,
	FOREIGN KEY(plan_id) REFERENCES plans(plan_id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_jobs_state_queued ON jobs(state, queued_at);

CREATE TABLE IF NOT EXISTS job_steps (
	job_id TEXT NOT NULL,
	step_id TEXT NOT NULL,
	state TEXT NOT NULL,
	attempts INTEGER NOT NULL DEFAULT 1,
	started_at INTEGER,
	ended_at INTEGER,
	preconditions_json TEXT,
	postconditions_json TEXT,
	result_digest TEXT,
	error_class TEXT,
	PRIMARY KEY(job_id, step_id),
	FOREIGN KEY(job_id) REFERENCES jobs(job_id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS events (
	seq INTEGER PRIMARY KEY AUTOINCREMENT,
	job_id TEXT,
	event_type TEXT NOT NULL,
	schema_version INTEGER NOT NULL,
	payload_json TEXT NOT NULL,
	timestamp INTEGER NOT NULL,
	correlation_id TEXT,
	digest TEXT,
	FOREIGN KEY(job_id) REFERENCES jobs(job_id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_events_job_seq ON events(job_id, seq);
CREATE INDEX IF NOT EXISTS idx_events_type_ts ON events(event_type, timestamp);

CREATE TABLE IF NOT EXISTS transactions (
	tx_id TEXT PRIMARY KEY,
	job_id TEXT,
	lifecycle TEXT NOT NULL,
	started_at INTEGER NOT NULL,
	committed_at INTEGER,
	compensated_at INTEGER,
	policy TEXT,
	outcome TEXT,
	FOREIGN KEY(job_id) REFERENCES jobs(job_id) ON DELETE SET NULL
);

CREATE TABLE IF NOT EXISTS snapshots (
	snapshot_id TEXT PRIMARY KEY,
	created_at INTEGER NOT NULL,
	root_digest TEXT NOT NULL,
	file_count INTEGER NOT NULL,
	total_bytes INTEGER NOT NULL,
	ownership TEXT,
	expiry INTEGER,
	pinned INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS snapshot_items (
	snapshot_id TEXT NOT NULL,
	canonical_path TEXT NOT NULL,
	blob_digest TEXT NOT NULL,
	mode INTEGER NOT NULL,
	uid INTEGER,
	gid INTEGER,
	size INTEGER NOT NULL,
	target_kind TEXT NOT NULL DEFAULT 'file',
	PRIMARY KEY(snapshot_id, canonical_path),
	FOREIGN KEY(snapshot_id) REFERENCES snapshots(snapshot_id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS cas_blobs (
	sha256 TEXT PRIMARY KEY,
	size INTEGER NOT NULL,
	created_at INTEGER NOT NULL,
	verified_at INTEGER NOT NULL,
	pin INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS migration_marker (
	source_sha256 TEXT PRIMARY KEY,
	schema_version INTEGER NOT NULL,
	imported_at INTEGER NOT NULL
);
`,
	},
}

func applyMigrations(ctx context.Context, db *sql.DB) error {
	const maxRetries = 20
	backoff := 10 * time.Millisecond

	for attempt := 0; attempt < maxRetries; attempt++ {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		err := func() error {
			// 1. Ensure migrations tracking table exists
			_, err := db.ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS schema_migrations (
	id INTEGER PRIMARY KEY,
	checksum TEXT NOT NULL,
	applied_at INTEGER NOT NULL
);`)
			if err != nil {
				return err
			}

			for _, m := range migrations {
				sum := sha256.Sum256([]byte(m.sql))
				hexSum := hex.EncodeToString(sum[:])

				var existingChecksum string
				rowErr := db.QueryRowContext(ctx, "SELECT checksum FROM schema_migrations WHERE id = ?", m.id).Scan(&existingChecksum)
				if rowErr == nil {
					if existingChecksum != hexSum {
						return fmt.Errorf("migration %d checksum mismatch: database has %s, code has %s", m.id, existingChecksum, hexSum)
					}
					continue
				} else if rowErr != sql.ErrNoRows {
					return fmt.Errorf("checking migration %d: %w", m.id, rowErr)
				}

				// Apply in transaction
				tx, err := db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
				if err != nil {
					return fmt.Errorf("starting tx for migration %d: %w", m.id, err)
				}

				if _, err := tx.ExecContext(ctx, m.sql); err != nil {
					_ = tx.Rollback()
					return fmt.Errorf("executing migration %d: %w", m.id, err)
				}

				if _, err := tx.ExecContext(ctx, "INSERT OR IGNORE INTO schema_migrations (id, checksum, applied_at) VALUES (?, ?, ?)", m.id, hexSum, time.Now().UTC().Unix()); err != nil {
					_ = tx.Rollback()
					return fmt.Errorf("recording migration %d: %w", m.id, err)
				}

				if err := tx.Commit(); err != nil {
					return fmt.Errorf("committing migration %d: %w", m.id, err)
				}
			}
			return nil
		}()

		if err == nil {
			return nil
		}

		if isBusy(err) {
			select {
			case <-time.After(backoff):
				backoff = min(backoff*2, 250*time.Millisecond)
				continue
			case <-ctx.Done():
				return ctx.Err()
			}
		}

		return err
	}

	return fmt.Errorf("migrations failed: database is locked after retries")
}
