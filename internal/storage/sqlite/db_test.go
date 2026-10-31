package sqlite

import (
	"context"
	"path/filepath"
	"testing"
)

func TestOpenAndPragmas(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	db, err := Open(context.Background(), Options{
		Path: dbPath,
	})
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			t.Errorf("Close failed: %v", err)
		}
	}()

	// Verify WAL mode
	var journalMode string
	err = db.writer.QueryRowContext(context.Background(), "PRAGMA journal_mode;").Scan(&journalMode)
	if err != nil {
		t.Fatalf("failed to query journal_mode: %v", err)
	}
	if journalMode != "wal" {
		t.Errorf("expected journal_mode=wal, got %s", journalMode)
	}

	// Verify foreign keys
	var fk int
	err = db.writer.QueryRowContext(context.Background(), "PRAGMA foreign_keys;").Scan(&fk)
	if err != nil {
		t.Fatalf("failed to query foreign_keys: %v", err)
	}
	if fk != 1 {
		t.Errorf("expected foreign_keys=1, got %d", fk)
	}
}
