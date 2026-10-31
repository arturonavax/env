package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

// DB manages SQLite database connections with strict separation between a serialized
// single-writer pool and a concurrent read pool operating in WAL mode.
type DB struct {
	path   string
	opts   Options
	writer *sql.DB
	reader *sql.DB
	wMu    sync.Mutex // Serializes writer transaction callers within the process
}

// Open initializes or opens the SQLite database at opts.Path.
func Open(ctx context.Context, opts Options) (*DB, error) {
	opts.setDefaults()

	if opts.Path == "" {
		return nil, fmt.Errorf("database path cannot be empty")
	}

	dir := filepath.Dir(opts.Path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, fmt.Errorf("creating database directory: %w", err)
	}

	// Format DSN with pragmas for modernc.org/sqlite
	dsn := fmt.Sprintf("file:%s?_txlock=immediate&_pragma=busy_timeout(%d)&_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)&_pragma=synchronous(%s)&_pragma=cache_size(-2048)",
		url.PathEscape(opts.Path),
		opts.BusyTimeout.Milliseconds(),
		opts.Synchronous,
	)

	// Writer connection pool: MaxOpenConns = 1 strictly prevents internal SQLITE_BUSY deadlocks
	wDB, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("opening write connection: %w", err)
	}
	wDB.SetMaxOpenConns(1)
	wDB.SetMaxIdleConns(1)

	// Execute explicit PRAGMAs to ensure active state on new database
	if err := initPragmas(ctx, wDB, opts); err != nil {
		_ = wDB.Close()
		return nil, fmt.Errorf("initializing writer pragmas: %w", err)
	}

	// Apply migrations on writer
	if err := applyMigrations(ctx, wDB); err != nil {
		_ = wDB.Close()
		return nil, fmt.Errorf("applying migrations: %w", err)
	}

	// Reader connection pool
	rDB, err := sql.Open("sqlite", dsn)
	if err != nil {
		_ = wDB.Close()
		return nil, fmt.Errorf("opening read connection: %w", err)
	}
	rDB.SetMaxOpenConns(opts.MaxReadConns)
	rDB.SetMaxIdleConns(opts.MaxReadConns)

	// POSIX 0600 security permissions on database file
	_ = os.Chmod(opts.Path, 0600)

	return &DB{
		path:   opts.Path,
		opts:   opts,
		writer: wDB,
		reader: rDB,
	}, nil
}

func initPragmas(ctx context.Context, db *sql.DB, opts Options) error {
	pragmas := []string{
		"PRAGMA journal_mode=WAL;",
		fmt.Sprintf("PRAGMA synchronous=%s;", opts.Synchronous),
		fmt.Sprintf("PRAGMA busy_timeout=%d;", opts.BusyTimeout.Milliseconds()),
		"PRAGMA foreign_keys=ON;",
		"PRAGMA cache_size=-2048;",
		"PRAGMA temp_store=MEMORY;",
	}

	for _, p := range pragmas {
		var err error
		backoff := 5 * time.Millisecond
		for attempt := 0; attempt < 15; attempt++ {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			_, err = db.ExecContext(ctx, p)
			if err == nil || !isBusy(err) {
				break
			}
			time.Sleep(backoff)
			backoff = min(backoff*2, 100*time.Millisecond)
		}
		if err != nil && !isBusy(err) {
			return fmt.Errorf("executing %q: %w", p, err)
		}
	}
	return nil
}

// Path returns the filesystem path of the database.
func (db *DB) Path() string {
	return db.path
}

// Writer returns the serialized single-connection writer *sql.DB.
func (db *DB) Writer() *sql.DB {
	return db.writer
}

// Reader returns the reader pool *sql.DB.
func (db *DB) Reader() *sql.DB {
	return db.reader
}

// Close closes all writer and reader database handles. It is safe and idempotent.
func (db *DB) Close() error {
	if db == nil {
		return nil
	}
	var firstErr error
	if db.reader != nil {
		if err := db.reader.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
		db.reader = nil
	}
	if db.writer != nil {
		if err := db.writer.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
		db.writer = nil
	}
	return firstErr
}

// isBusy checks if an error represents SQLite lock contention (SQLITE_BUSY or database is locked).
func isBusy(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "SQLITE_BUSY") ||
		strings.Contains(msg, "database is locked") ||
		strings.Contains(msg, "database table is locked")
}

// WriteTx executes fn within an exclusive serialized transaction with BEGIN IMMEDIATE and bounded backoff.
func (db *DB) WriteTx(ctx context.Context, fn func(tx *sql.Tx) error) error {
	db.wMu.Lock()
	defer db.wMu.Unlock()

	const maxRetries = 15
	var lastErr error
	backoff := 5 * time.Millisecond

	for attempt := 0; attempt < maxRetries; attempt++ {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		err := func() error {
			tx, err := db.writer.BeginTx(ctx, &sql.TxOptions{
				Isolation: sql.LevelSerializable,
			})
			if err != nil {
				return err
			}

			if err := fn(tx); err != nil {
				_ = tx.Rollback()
				return err
			}

			if err := tx.Commit(); err != nil {
				return err
			}
			return nil
		}()

		if err == nil {
			return nil
		}

		if isBusy(err) {
			lastErr = err
			select {
			case <-time.After(backoff):
				backoff = min(backoff*2, 200*time.Millisecond)
				continue
			case <-ctx.Done():
				return ctx.Err()
			}
		}

		return err
	}

	return fmt.Errorf("transaction failed after retries (lock contention): %w", lastErr)
}

// ReadTx executes fn within a read-only transaction against the reader pool.
func (db *DB) ReadTx(ctx context.Context, fn func(tx *sql.Tx) error) error {
	tx, err := db.reader.BeginTx(ctx, &sql.TxOptions{
		ReadOnly: true,
	})
	if err != nil {
		return fmt.Errorf("starting read tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	return fn(tx)
}

// CheckIntegrity runs SQLite's quick_check and foreign_key_check.
func (db *DB) CheckIntegrity(ctx context.Context) error {
	var result string
	err := db.reader.QueryRowContext(ctx, "PRAGMA quick_check;").Scan(&result)
	if err != nil {
		return fmt.Errorf("quick_check failed: %w", err)
	}
	if result != "ok" {
		return fmt.Errorf("quick_check returned: %s", result)
	}

	rows, err := db.reader.QueryContext(ctx, "PRAGMA foreign_key_check;")
	if err != nil {
		return fmt.Errorf("foreign_key_check failed: %w", err)
	}
	defer func() { _ = rows.Close() }()

	if rows.Next() {
		var table, fkid, target string
		var rowid int64
		_ = rows.Scan(&table, &rowid, &target, &fkid)
		return fmt.Errorf("foreign key violation in table %s (rowid: %d, target: %s)", table, rowid, target)
	}

	return nil
}

// BackupTo creates an online, crash-consistent copy of the database at destPath using VACUUM INTO.
func (db *DB) BackupTo(ctx context.Context, destPath string) error {
	db.wMu.Lock()
	defer db.wMu.Unlock()

	// Ensure destination directory exists with private permissions
	if err := os.MkdirAll(filepath.Dir(destPath), 0700); err != nil {
		return fmt.Errorf("creating backup dir: %w", err)
	}

	// Remove destination file if exists, since VACUUM INTO requires the target file to not exist
	_ = os.Remove(destPath)

	escapedPath := strings.ReplaceAll(destPath, "'", "''")
	query := fmt.Sprintf("VACUUM INTO '%s';", escapedPath)
	if _, err := db.writer.ExecContext(ctx, query); err != nil {
		return fmt.Errorf("VACUUM INTO failed: %w", err)
	}

	_ = os.Chmod(destPath, 0600)
	return nil
}
