package state

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
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

	_ = store.Close()

	// Reopen from disk to prove durability
	reopened, err := OpenStore(dir)
	if err != nil {
		t.Fatalf("reopening store failed: %v", err)
	}
	t.Cleanup(func() { _ = reopened.Close() })

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

// STATE-01: In-process concurrent writes must be linearizable and race-free.
func TestStoreConcurrentInProcessWrites(t *testing.T) {
	dir := t.TempDir()
	store, err := OpenStore(dir)
	if err != nil {
		t.Fatal(err)
	}

	const workers = 16
	const writesPerWorker = 20

	var wg sync.WaitGroup
	for w := range workers {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for i := range writesPerWorker {
				resID := fmt.Sprintf("pkg-w%d-%d", workerID, i)
				rec := InstalledRecord{
					ResourceID:   resID,
					Version:      fmt.Sprintf("%d.0.0", i),
					Provider:     "test",
					ManagedByAnx: true,
				}
				if err := store.RecordResource(rec); err != nil {
					t.Errorf("worker %d failed recording %s: %v", workerID, resID, err)
					return
				}
			}
		}(w)
	}
	wg.Wait()

	// Verify all records exist
	all := store.AllResources()
	expectedCount := workers * writesPerWorker
	if len(all) != expectedCount {
		t.Fatalf("expected %d records in store, got %d", expectedCount, len(all))
	}
	_ = store.Close()

	// Verify durability after reopening
	reopened, err := OpenStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = reopened.Close() })
	reopenedAll := reopened.AllResources()
	if len(reopenedAll) != expectedCount {
		t.Fatalf("reopened store has %d records, want %d", len(reopenedAll), expectedCount)
	}
}

// STATE-01: Multi-process simulation (multiple Store instances targeting the same directory).
func TestStoreMultiProcessSimulation(t *testing.T) {
	for _, instances := range []int{2, 4, 8} {
		t.Run(fmt.Sprintf("%d_instances", instances), func(t *testing.T) {
			dir := t.TempDir()
			const writesPerInstance = 10

			var wg sync.WaitGroup
			for inst := range instances {
				wg.Add(1)
				go func(instanceID int) {
					defer wg.Done()
					// Each instance opens its own Store object simulating a separate process
					s, err := OpenStore(dir)
					if err != nil {
						t.Errorf("instance %d OpenStore: %v", instanceID, err)
						return
					}
					defer func() { _ = s.Close() }()

					for i := range writesPerInstance {
						resID := fmt.Sprintf("multi-inst-%d-%d", instanceID, i)
						rec := InstalledRecord{
							ResourceID:   resID,
							Version:      "1.0",
							ManagedByAnx: true,
						}
						if err := s.RecordResource(rec); err != nil {
							t.Errorf("instance %d record %s: %v", instanceID, resID, err)
							return
						}
					}
				}(inst)
			}
			wg.Wait()

			finalStore, err := OpenStore(dir)
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = finalStore.Close() }()

			all := finalStore.AllResources()
			expected := instances * writesPerInstance
			if len(all) != expected {
				t.Fatalf("multi-instance concurrent writes: expected %d records, got %d", expected, len(all))
			}
		})
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
