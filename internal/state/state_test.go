package state

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStorePersistenceAndConfigBases(t *testing.T) {
	dir := t.TempDir()
	store, err := OpenStore(dir)
	if err != nil {
		t.Fatalf("OpenStore failed: %v", err)
	}

	rec := InstalledRecord{
		ResourceID:    "ripgrep",
		Version:       "14.1.1",
		Strategy:      "native",
		Provider:      "apt",
		ManagedByAnx:  true,
		InstalledPath: "/usr/bin/rg",
	}
	if err := store.RecordResource(rec); err != nil {
		t.Fatalf("RecordResource failed: %v", err)
	}

	if err := store.RecordConfigBase("starship", []byte("[character]\nsuccess_symbol = '❯'\n")); err != nil {
		t.Fatalf("RecordConfigBase failed: %v", err)
	}

	if err := store.RecordHack("dell-precision-m6800", []string{"ghostty-a11y", "picom-glx"}); err != nil {
		t.Fatalf("RecordHack failed: %v", err)
	}

	// Reopen from disk to prove durability
	reopened, err := OpenStore(dir)
	if err != nil {
		t.Fatalf("reopening store failed: %v", err)
	}

	got, ok := reopened.GetResource("ripgrep")
	if !ok || got.Version != "14.1.1" || !got.ManagedByAnx {
		t.Fatalf("expected persisted ripgrep record, got ok=%v rec=%+v", ok, got)
	}

	base, ok := reopened.GetConfigBase("starship")
	if !ok || string(base) != "[character]\nsuccess_symbol = '❯'\n" {
		t.Fatalf("expected persisted base content, got ok=%v base=%q", ok, string(base))
	}

	if !reopened.IsHackApplied("dell-precision-m6800") {
		t.Fatal("expected dell-precision-m6800 hack persisted")
	}

	if err := reopened.RemoveResource("ripgrep"); err != nil {
		t.Fatalf("RemoveResource failed: %v", err)
	}
	if _, ok := reopened.GetResource("ripgrep"); ok {
		t.Fatal("expected ripgrep removed")
	}
}

func TestAdvisoryFileLockMutualExclusionAndStaleRecovery(t *testing.T) {
	dir := t.TempDir()

	lock1, err := AcquireLock(dir, "apply")
	if err != nil {
		t.Fatalf("AcquireLock 1 failed: %v", err)
	}

	// Second concurrent lock acquisition must fail
	if _, err := AcquireLock(dir, "sync"); err == nil {
		t.Fatal("expected concurrent AcquireLock to fail while lock1 is held")
	}

	if err := lock1.Release(); err != nil {
		t.Fatalf("Release failed: %v", err)
	}

	// Simulate a stale lock from non-existent PID
	lockPath := filepath.Join(dir, lockFileName)
	if err := os.WriteFile(lockPath, []byte(`{"pid":99999999,"operation":"crashed"}`), 0o600); err != nil {
		t.Fatalf("writing stale lock failed: %v", err)
	}

	lock2, err := AcquireLock(dir, "recover")
	if err != nil {
		t.Fatalf("expected stale PID lock recovery to succeed, got %v", err)
	}
	defer func() { _ = lock2.Release() }()
}
