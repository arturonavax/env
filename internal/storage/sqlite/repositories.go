package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"
)

// UpsertResource inserts or updates an installed resource record.
func (db *DB) UpsertResource(ctx context.Context, rec ResourceRecord) error {
	now := time.Now().UTC().Unix()
	var installedAt int64
	if rec.InstalledAt.IsZero() {
		installedAt = now
	} else {
		installedAt = rec.InstalledAt.UTC().Unix()
	}

	managed := 0
	if rec.ManagedByAnx {
		managed = 1
	}

	query := `
INSERT INTO resources (
	resource_id, kind, provider, strategy, version, managed_by_anx,
	install_path, origin, installed_at, updated_at, generation
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 1)
ON CONFLICT(resource_id) DO UPDATE SET
	kind = excluded.kind,
	provider = excluded.provider,
	strategy = excluded.strategy,
	version = excluded.version,
	managed_by_anx = excluded.managed_by_anx,
	install_path = excluded.install_path,
	origin = excluded.origin,
	updated_at = excluded.updated_at,
	generation = resources.generation + 1;`

	kind := rec.Kind
	if kind == "" {
		kind = "package"
	}

	return db.WriteTx(ctx, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, query,
			rec.ResourceID,
			kind,
			rec.Provider,
			rec.Strategy,
			rec.Version,
			managed,
			rec.InstalledPath,
			rec.Origin,
			installedAt,
			now,
		)
		return err
	})
}

// GetResource fetches a single installed resource record by ID.
func (db *DB) GetResource(ctx context.Context, resourceID string) (ResourceRecord, bool, error) {
	query := `
SELECT resource_id, kind, provider, strategy, version, managed_by_anx,
       install_path, origin, installed_at, updated_at, generation
FROM resources WHERE resource_id = ?;`

	var rec ResourceRecord
	var managed int
	var installedAt, updatedAt int64
	var version, installPath, origin sql.NullString

	err := db.reader.QueryRowContext(ctx, query, resourceID).Scan(
		&rec.ResourceID,
		&rec.Kind,
		&rec.Provider,
		&rec.Strategy,
		&version,
		&managed,
		&installPath,
		&origin,
		&installedAt,
		&updatedAt,
		&rec.Generation,
	)
	if err == sql.ErrNoRows {
		return ResourceRecord{}, false, nil
	} else if err != nil {
		return ResourceRecord{}, false, fmt.Errorf("querying resource %s: %w", resourceID, err)
	}

	rec.Version = version.String
	rec.InstalledPath = installPath.String
	rec.Origin = origin.String
	rec.ManagedByAnx = managed == 1
	rec.InstalledAt = time.Unix(installedAt, 0).UTC()
	rec.UpdatedAt = time.Unix(updatedAt, 0).UTC()

	return rec, true, nil
}

// DeleteResource deletes an installed resource record by ID.
func (db *DB) DeleteResource(ctx context.Context, resourceID string) error {
	return db.WriteTx(ctx, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, "DELETE FROM resources WHERE resource_id = ?;", resourceID)
		return err
	})
}

// AllResources returns all tracked installed resources keyed by resource ID.
func (db *DB) AllResources(ctx context.Context) (map[string]ResourceRecord, error) {
	query := `
SELECT resource_id, kind, provider, strategy, version, managed_by_anx,
       install_path, origin, installed_at, updated_at, generation
FROM resources;`

	rows, err := db.reader.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("querying all resources: %w", err)
	}
	defer func() { _ = rows.Close() }()

	out := make(map[string]ResourceRecord)
	for rows.Next() {
		var rec ResourceRecord
		var managed int
		var installedAt, updatedAt int64
		var version, installPath, origin sql.NullString

		if err := rows.Scan(
			&rec.ResourceID,
			&rec.Kind,
			&rec.Provider,
			&rec.Strategy,
			&version,
			&managed,
			&installPath,
			&origin,
			&installedAt,
			&updatedAt,
			&rec.Generation,
		); err != nil {
			return nil, fmt.Errorf("scanning resource row: %w", err)
		}

		rec.Version = version.String
		rec.InstalledPath = installPath.String
		rec.Origin = origin.String
		rec.ManagedByAnx = managed == 1
		rec.InstalledAt = time.Unix(installedAt, 0).UTC()
		rec.UpdatedAt = time.Unix(updatedAt, 0).UTC()

		out[rec.ResourceID] = rec
	}

	return out, rows.Err()
}

// DetectOrphaned returns all installed resources that are not present in desiredResourceIDs.
func (db *DB) DetectOrphaned(ctx context.Context, desiredResourceIDs []string) ([]ResourceRecord, error) {
	all, err := db.AllResources(ctx)
	if err != nil {
		return nil, err
	}

	desiredMap := make(map[string]bool, len(desiredResourceIDs))
	for _, id := range desiredResourceIDs {
		desiredMap[id] = true
	}

	orphans := make([]ResourceRecord, 0)
	for id, rec := range all {
		if !desiredMap[id] {
			orphans = append(orphans, rec)
		}
	}

	slices.SortFunc(orphans, func(a, b ResourceRecord) int {
		return strings.Compare(a.ResourceID, b.ResourceID)
	})

	return orphans, nil
}

// UpsertConfigBase stores or updates the ancestor configuration content.
func (db *DB) UpsertConfigBase(ctx context.Context, rec ConfigBaseRecord) error {
	now := time.Now().UTC().Unix()
	query := `
INSERT INTO config_bases (
	config_id, content_digest, size, eol, applied_revision, updated_at, content
) VALUES (?, ?, ?, ?, 1, ?, ?)
ON CONFLICT(config_id) DO UPDATE SET
	content_digest = excluded.content_digest,
	size = excluded.size,
	eol = excluded.eol,
	applied_revision = config_bases.applied_revision + 1,
	updated_at = excluded.updated_at,
	content = excluded.content;`

	eol := rec.EOL
	if eol == "" {
		eol = "lf"
	}

	return db.WriteTx(ctx, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, query,
			rec.ConfigID,
			rec.ContentDigest,
			rec.Size,
			eol,
			now,
			rec.Content,
		)
		return err
	})
}

// GetConfigBase retrieves the ancestor configuration content by config ID.
func (db *DB) GetConfigBase(ctx context.Context, configID string) (ConfigBaseRecord, bool, error) {
	query := `
SELECT config_id, content_digest, size, eol, applied_revision, updated_at, content
FROM config_bases WHERE config_id = ?;`

	var rec ConfigBaseRecord
	var updatedAt int64

	err := db.reader.QueryRowContext(ctx, query, configID).Scan(
		&rec.ConfigID,
		&rec.ContentDigest,
		&rec.Size,
		&rec.EOL,
		&rec.AppliedRevision,
		&updatedAt,
		&rec.Content,
	)
	if err == sql.ErrNoRows {
		return ConfigBaseRecord{}, false, nil
	} else if err != nil {
		return ConfigBaseRecord{}, false, fmt.Errorf("querying config base %s: %w", configID, err)
	}

	rec.UpdatedAt = time.Unix(updatedAt, 0).UTC()
	return rec, true, nil
}

// AllConfigBases returns all ancestor configuration records.
func (db *DB) AllConfigBases(ctx context.Context) (map[string]ConfigBaseRecord, error) {
	query := `
SELECT config_id, content_digest, size, eol, applied_revision, updated_at, content
FROM config_bases;`

	rows, err := db.reader.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("querying all config bases: %w", err)
	}
	defer func() { _ = rows.Close() }()

	out := make(map[string]ConfigBaseRecord)
	for rows.Next() {
		var rec ConfigBaseRecord
		var updatedAt int64

		if err := rows.Scan(
			&rec.ConfigID,
			&rec.ContentDigest,
			&rec.Size,
			&rec.EOL,
			&rec.AppliedRevision,
			&updatedAt,
			&rec.Content,
		); err != nil {
			return nil, fmt.Errorf("scanning config base row: %w", err)
		}

		rec.UpdatedAt = time.Unix(updatedAt, 0).UTC()
		out[rec.ConfigID] = rec
	}

	return out, rows.Err()
}

// RecordHack persists applied hack steps.
func (db *DB) RecordHack(ctx context.Context, rec AppliedHackRecord) error {
	stepsJSON, err := json.Marshal(rec.Steps)
	if err != nil {
		return fmt.Errorf("marshaling steps: %w", err)
	}

	now := time.Now().UTC().Unix()
	query := `
INSERT INTO applied_hacks (hack_id, host_fingerprint, steps_json, applied_at, version)
VALUES (?, ?, ?, ?, ?)
ON CONFLICT(hack_id) DO UPDATE SET
	host_fingerprint = excluded.host_fingerprint,
	steps_json = excluded.steps_json,
	applied_at = excluded.applied_at,
	version = excluded.version;`

	return db.WriteTx(ctx, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, query,
			rec.HackID,
			rec.HostFingerprint,
			string(stepsJSON),
			now,
			rec.Version,
		)
		return err
	})
}

// IsHackApplied reports whether hackID is recorded as applied.
func (db *DB) IsHackApplied(ctx context.Context, hackID string) (bool, error) {
	var count int
	err := db.reader.QueryRowContext(ctx, "SELECT COUNT(*) FROM applied_hacks WHERE hack_id = ?;", hackID).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("checking hack %s: %w", hackID, err)
	}
	return count > 0, nil
}

// AllAppliedHacks returns all applied hacks.
func (db *DB) AllAppliedHacks(ctx context.Context) (map[string]AppliedHackRecord, error) {
	query := `SELECT hack_id, host_fingerprint, steps_json, applied_at, version FROM applied_hacks;`
	rows, err := db.reader.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("querying applied hacks: %w", err)
	}
	defer func() { _ = rows.Close() }()

	out := make(map[string]AppliedHackRecord)
	for rows.Next() {
		var rec AppliedHackRecord
		var stepsJSON string
		var appliedAt int64
		var hostFp, ver sql.NullString

		if err := rows.Scan(
			&rec.HackID,
			&hostFp,
			&stepsJSON,
			&appliedAt,
			&ver,
		); err != nil {
			return nil, fmt.Errorf("scanning hack row: %w", err)
		}

		_ = json.Unmarshal([]byte(stepsJSON), &rec.Steps)
		rec.HostFingerprint = hostFp.String
		rec.Version = ver.String
		rec.AppliedAt = time.Unix(appliedAt, 0).UTC()

		out[rec.HackID] = rec
	}

	return out, rows.Err()
}

// SetMetadata sets or updates a key-value pair in metadata.
func (db *DB) SetMetadata(ctx context.Context, key, value string) error {
	now := time.Now().UTC().Unix()
	query := `
INSERT INTO metadata (key, value, updated_at, revision)
VALUES (?, ?, ?, 1)
ON CONFLICT(key) DO UPDATE SET
	value = excluded.value,
	updated_at = excluded.updated_at,
	revision = metadata.revision + 1;`

	return db.WriteTx(ctx, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, query, key, value, now)
		return err
	})
}

// GetMetadata fetches a key from metadata.
func (db *DB) GetMetadata(ctx context.Context, key string) (string, bool, error) {
	var val string
	err := db.reader.QueryRowContext(ctx, "SELECT value FROM metadata WHERE key = ?;", key).Scan(&val)
	if err == sql.ErrNoRows {
		return "", false, nil
	} else if err != nil {
		return "", false, fmt.Errorf("getting metadata %s: %w", key, err)
	}
	return val, true, nil
}

// Reset clears all resources, config bases, applied hacks, and active profile in a single atomic transaction.
func (db *DB) Reset(ctx context.Context) error {
	return db.WriteTx(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, "DELETE FROM resources;"); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, "DELETE FROM config_bases;"); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, "DELETE FROM applied_hacks;"); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, "DELETE FROM metadata WHERE key = 'active_profile';"); err != nil {
			return err
		}
		return nil
	})
}

// GetMigrationMarker returns the migration marker for sourceSHA if previously migrated.
func (db *DB) GetMigrationMarker(ctx context.Context, sourceSHA string) (MigrationMarker, bool, error) {
	var marker MigrationMarker
	var importedAt int64
	err := db.reader.QueryRowContext(ctx, "SELECT source_sha256, schema_version, imported_at FROM migration_marker WHERE source_sha256 = ?;", sourceSHA).Scan(
		&marker.SourceSHA256,
		&marker.SchemaVersion,
		&importedAt,
	)
	if err == sql.ErrNoRows {
		return MigrationMarker{}, false, nil
	} else if err != nil {
		return MigrationMarker{}, false, err
	}
	marker.ImportedAt = time.Unix(importedAt, 0).UTC()
	return marker, true, nil
}

// RecordMigrationMarker records provenance for a completed migration.
func (db *DB) RecordMigrationMarker(ctx context.Context, marker MigrationMarker) error {
	now := marker.ImportedAt.Unix()
	if now == 0 {
		now = time.Now().UTC().Unix()
	}
	query := `INSERT OR REPLACE INTO migration_marker (source_sha256, schema_version, imported_at) VALUES (?, ?, ?);`
	return db.WriteTx(ctx, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, query, marker.SourceSHA256, marker.SchemaVersion, now)
		return err
	})
}

// SavePlan stores an immutable PlanRecord.
func (db *DB) SavePlan(ctx context.Context, plan PlanRecord) error {
	query := `
INSERT OR REPLACE INTO plans (
	plan_id, schema_version, canonical_ir_digest, base_host_revision,
	config_catalog_revision, policy_revision, target, created_at, expires_at,
	authorized_at, authorized_by, status
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);`

	var authAt *int64
	if plan.AuthorizedAt != nil {
		t := plan.AuthorizedAt.Unix()
		authAt = &t
	}

	return db.WriteTx(ctx, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, query,
			plan.PlanID,
			plan.SchemaVersion,
			plan.CanonicalIRDigest,
			plan.BaseHostRevision,
			plan.ConfigCatalogRevision,
			plan.PolicyRevision,
			plan.Target,
			plan.CreatedAt.Unix(),
			plan.ExpiresAt.Unix(),
			authAt,
			plan.AuthorizedBy,
			plan.Status,
		)
		return err
	})
}

// GetPlan retrieves a PlanRecord by ID.
func (db *DB) GetPlan(ctx context.Context, planID string) (PlanRecord, bool, error) {
	query := `
SELECT plan_id, schema_version, canonical_ir_digest, base_host_revision,
       config_catalog_revision, policy_revision, target, created_at, expires_at,
       authorized_at, authorized_by, status
FROM plans WHERE plan_id = ?;`

	var p PlanRecord
	var baseHost, cfgCat, polRev, target, authBy sql.NullString
	var authAt sql.NullInt64
	var createdAt, expiresAt int64

	err := db.reader.QueryRowContext(ctx, query, planID).Scan(
		&p.PlanID,
		&p.SchemaVersion,
		&p.CanonicalIRDigest,
		&baseHost,
		&cfgCat,
		&polRev,
		&target,
		&createdAt,
		&expiresAt,
		&authAt,
		&authBy,
		&p.Status,
	)
	if err == sql.ErrNoRows {
		return PlanRecord{}, false, nil
	} else if err != nil {
		return PlanRecord{}, false, err
	}

	p.BaseHostRevision = baseHost.String
	p.ConfigCatalogRevision = cfgCat.String
	p.PolicyRevision = polRev.String
	p.Target = target.String
	p.AuthorizedBy = authBy.String
	p.CreatedAt = time.Unix(createdAt, 0).UTC()
	p.ExpiresAt = time.Unix(expiresAt, 0).UTC()
	if authAt.Valid {
		t := time.Unix(authAt.Int64, 0).UTC()
		p.AuthorizedAt = &t
	}

	return p, true, nil
}

// CreateJob persists a new durable JobRecord.
func (db *DB) CreateJob(ctx context.Context, job JobRecord) error {
	query := `
INSERT OR REPLACE INTO jobs (
	job_id, plan_id, idempotency_key, state, cursor, queued_at,
	started_at, finished_at, attempt, exit_error_code, resume_token
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);`

	var startAt, finAt *int64
	if job.StartedAt != nil {
		t := job.StartedAt.Unix()
		startAt = &t
	}
	if job.FinishedAt != nil {
		t := job.FinishedAt.Unix()
		finAt = &t
	}

	return db.WriteTx(ctx, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, query,
			job.JobID,
			job.PlanID,
			job.IdempotencyKey,
			job.State,
			job.Cursor,
			job.QueuedAt.Unix(),
			startAt,
			finAt,
			job.Attempt,
			job.ExitErrorCode,
			job.ResumeToken,
		)
		return err
	})
}

// UpdateJobState transitions the job state and records completion time and error code when finished.
func (db *DB) UpdateJobState(ctx context.Context, jobID, state string, exitCode int) error {
	now := time.Now().UTC().Unix()
	var query string
	if state == "running" {
		query = `UPDATE jobs SET state = ?, started_at = ? WHERE job_id = ?;`
		return db.WriteTx(ctx, func(tx *sql.Tx) error {
			_, err := tx.ExecContext(ctx, query, state, now, jobID)
			return err
		})
	}
	query = `UPDATE jobs SET state = ?, finished_at = ?, exit_error_code = ? WHERE job_id = ?;`
	return db.WriteTx(ctx, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, query, state, now, exitCode, jobID)
		return err
	})
}

// GetJob fetches a JobRecord by ID.
func (db *DB) GetJob(ctx context.Context, jobID string) (JobRecord, bool, error) {
	query := `
SELECT job_id, plan_id, idempotency_key, state, cursor, queued_at,
       started_at, finished_at, attempt, exit_error_code, resume_token
FROM jobs WHERE job_id = ?;`

	var j JobRecord
	var planID, idempKey, resumeToken sql.NullString
	var startAt, finAt sql.NullInt64
	var queuedAt int64

	err := db.reader.QueryRowContext(ctx, query, jobID).Scan(
		&j.JobID,
		&planID,
		&idempKey,
		&j.State,
		&j.Cursor,
		&queuedAt,
		&startAt,
		&finAt,
		&j.Attempt,
		&j.ExitErrorCode,
		&resumeToken,
	)
	if err == sql.ErrNoRows {
		return JobRecord{}, false, nil
	} else if err != nil {
		return JobRecord{}, false, err
	}

	j.PlanID = planID.String
	j.IdempotencyKey = idempKey.String
	j.ResumeToken = resumeToken.String
	j.QueuedAt = time.Unix(queuedAt, 0).UTC()
	if startAt.Valid {
		t := time.Unix(startAt.Int64, 0).UTC()
		j.StartedAt = &t
	}
	if finAt.Valid {
		t := time.Unix(finAt.Int64, 0).UTC()
		j.FinishedAt = &t
	}

	return j, true, nil
}

// ListJobs retrieves the most recent jobs ordered by queued_at desc.
func (db *DB) ListJobs(ctx context.Context, limit int) ([]JobRecord, error) {
	if limit <= 0 {
		limit = 50
	}
	query := `
SELECT job_id, plan_id, idempotency_key, state, cursor, queued_at,
       started_at, finished_at, attempt, exit_error_code, resume_token
FROM jobs ORDER BY queued_at DESC LIMIT ?;`

	rows, err := db.reader.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var jobs []JobRecord
	for rows.Next() {
		var j JobRecord
		var planID, idempKey, resumeToken sql.NullString
		var startAt, finAt sql.NullInt64
		var queuedAt int64

		if err := rows.Scan(
			&j.JobID,
			&planID,
			&idempKey,
			&j.State,
			&j.Cursor,
			&queuedAt,
			&startAt,
			&finAt,
			&j.Attempt,
			&j.ExitErrorCode,
			&resumeToken,
		); err != nil {
			return nil, err
		}

		j.PlanID = planID.String
		j.IdempotencyKey = idempKey.String
		j.ResumeToken = resumeToken.String
		j.QueuedAt = time.Unix(queuedAt, 0).UTC()
		if startAt.Valid {
			t := time.Unix(startAt.Int64, 0).UTC()
			j.StartedAt = &t
		}
		if finAt.Valid {
			t := time.Unix(finAt.Int64, 0).UTC()
			j.FinishedAt = &t
		}
		jobs = append(jobs, j)
	}

	return jobs, rows.Err()
}

// AppendEvent appends an audit event to the append-only events log.
func (db *DB) AppendEvent(ctx context.Context, ev EventRecord) error {
	now := ev.Timestamp.Unix()
	if now == 0 {
		now = time.Now().UTC().Unix()
	}
	query := `
INSERT INTO events (job_id, event_type, schema_version, payload_json, timestamp, correlation_id, digest)
VALUES (?, ?, ?, ?, ?, ?, ?);`

	return db.WriteTx(ctx, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, query,
			ev.JobID,
			ev.EventType,
			ev.SchemaVersion,
			ev.PayloadJSON,
			now,
			ev.CorrelationID,
			ev.Digest,
		)
		return err
	})
}

// GetEventsForJob returns all events associated with a job in chronological order.
func (db *DB) GetEventsForJob(ctx context.Context, jobID string) ([]EventRecord, error) {
	query := `
SELECT seq, job_id, event_type, schema_version, payload_json, timestamp, correlation_id, digest
FROM events WHERE job_id = ? ORDER BY seq ASC;`

	rows, err := db.reader.QueryContext(ctx, query, jobID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var events []EventRecord
	for rows.Next() {
		var ev EventRecord
		var jID, corrID, digest sql.NullString
		var ts int64

		if err := rows.Scan(
			&ev.Seq,
			&jID,
			&ev.EventType,
			&ev.SchemaVersion,
			&ev.PayloadJSON,
			&ts,
			&corrID,
			&digest,
		); err != nil {
			return nil, err
		}

		ev.JobID = jID.String
		ev.CorrelationID = corrID.String
		ev.Digest = digest.String
		ev.Timestamp = time.Unix(ts, 0).UTC()
		events = append(events, ev)
	}

	return events, rows.Err()
}
