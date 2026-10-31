package sqlite

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestRepositories_PlansJobsEvents(t *testing.T) {
	dir := t.TempDir()
	db, err := Open(context.Background(), Options{Path: filepath.Join(dir, "anx.db")})
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer func() { _ = db.Close() }()

	ctx := context.Background()

	// 1. Plan
	now := time.Now().UTC()
	authTime := now.Add(time.Minute)
	plan := PlanRecord{
		PlanID:                "plan-abc-123",
		SchemaVersion:         1,
		CanonicalIRDigest:     "sha256:ir-digest-123",
		BaseHostRevision:      "host-rev-1",
		ConfigCatalogRevision: "cat-rev-1",
		PolicyRevision:        "pol-rev-1",
		Target:                "linux/amd64",
		CreatedAt:             now,
		ExpiresAt:             now.Add(time.Hour),
		AuthorizedAt:          &authTime,
		AuthorizedBy:          "user",
		Status:                "authorized",
	}

	if err := db.SavePlan(ctx, plan); err != nil {
		t.Fatalf("SavePlan failed: %v", err)
	}

	gotPlan, ok, err := db.GetPlan(ctx, "plan-abc-123")
	if err != nil || !ok {
		t.Fatalf("GetPlan failed: ok=%v err=%v", ok, err)
	}
	if gotPlan.PlanID != plan.PlanID || gotPlan.CanonicalIRDigest != plan.CanonicalIRDigest || gotPlan.Status != "authorized" {
		t.Errorf("unexpected plan: %+v", gotPlan)
	}

	// 2. Job
	job := JobRecord{
		JobID:          "job-xyz-789",
		PlanID:         "plan-abc-123",
		IdempotencyKey: "idemp-key-1",
		State:          "queued",
		Cursor:         0,
		QueuedAt:       now,
		Attempt:        1,
	}

	if err := db.CreateJob(ctx, job); err != nil {
		t.Fatalf("CreateJob failed: %v", err)
	}

	gotJob, ok, err := db.GetJob(ctx, "job-xyz-789")
	if err != nil || !ok {
		t.Fatalf("GetJob failed: ok=%v err=%v", ok, err)
	}
	if gotJob.State != "queued" || gotJob.PlanID != "plan-abc-123" {
		t.Errorf("unexpected job: %+v", gotJob)
	}

	// Transition to running
	if err := db.UpdateJobState(ctx, "job-xyz-789", "running", 0); err != nil {
		t.Fatalf("UpdateJobState running failed: %v", err)
	}

	// Transition to succeeded
	if err := db.UpdateJobState(ctx, "job-xyz-789", "succeeded", 0); err != nil {
		t.Fatalf("UpdateJobState succeeded failed: %v", err)
	}

	gotJob2, ok, _ := db.GetJob(ctx, "job-xyz-789")
	if !ok || gotJob2.State != "succeeded" || gotJob2.FinishedAt == nil {
		t.Errorf("expected succeeded job with FinishedAt, got %+v", gotJob2)
	}

	// List Jobs
	jobs, err := db.ListJobs(ctx, 10)
	if err != nil || len(jobs) != 1 {
		t.Fatalf("ListJobs failed: count=%d err=%v", len(jobs), err)
	}

	// 3. Events
	ev := EventRecord{
		JobID:         "job-xyz-789",
		EventType:     "step.executed",
		SchemaVersion: 1,
		PayloadJSON:   `{"step":"package.ripgrep","duration_ms":125}`,
		Timestamp:     now,
		CorrelationID: "corr-1",
	}

	if err := db.AppendEvent(ctx, ev); err != nil {
		t.Fatalf("AppendEvent failed: %v", err)
	}

	events, err := db.GetEventsForJob(ctx, "job-xyz-789")
	if err != nil || len(events) != 1 {
		t.Fatalf("GetEventsForJob failed: count=%d err=%v", len(events), err)
	}
	if events[0].EventType != "step.executed" || events[0].Seq <= 0 {
		t.Errorf("unexpected event: %+v", events[0])
	}
}

func TestRepositories_ResourcesCRUD(t *testing.T) {
	dir := t.TempDir()
	db, err := Open(context.Background(), Options{Path: filepath.Join(dir, "anx.db")})
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer func() { _ = db.Close() }()

	ctx := context.Background()

	rec := ResourceRecord{
		ResourceID:    "ripgrep",
		Kind:          "package",
		Provider:      "apt",
		Strategy:      "native",
		Version:       "14.1.0",
		ManagedByAnx:  true,
		InstalledPath: "/usr/bin/rg",
		InstalledAt:   time.Now().UTC().Add(-time.Hour),
	}

	if err := db.UpsertResource(ctx, rec); err != nil {
		t.Fatalf("UpsertResource failed: %v", err)
	}

	got, ok, err := db.GetResource(ctx, "ripgrep")
	if err != nil || !ok {
		t.Fatalf("GetResource failed: ok=%v err=%v", ok, err)
	}
	if got.ResourceID != "ripgrep" || got.Version != "14.1.0" || !got.ManagedByAnx {
		t.Errorf("unexpected record: %+v", got)
	}

	all, err := db.AllResources(ctx)
	if err != nil {
		t.Fatalf("AllResources failed: %v", err)
	}
	if len(all) != 1 {
		t.Errorf("expected 1 resource, got %d", len(all))
	}

	// Test update (upsert preserves original InstalledAt, updates UpdatedAt and increments Generation)
	rec.Version = "14.1.1"
	if err := db.UpsertResource(ctx, rec); err != nil {
		t.Fatalf("UpsertResource update failed: %v", err)
	}

	updated, ok, err := db.GetResource(ctx, "ripgrep")
	if err != nil || !ok {
		t.Fatalf("GetResource after update failed: %v", err)
	}
	if updated.Version != "14.1.1" {
		t.Errorf("expected updated version 14.1.1, got %s", updated.Version)
	}
	if updated.Generation != 2 {
		t.Errorf("expected generation 2, got %d", updated.Generation)
	}

	// Delete
	if err := db.DeleteResource(ctx, "ripgrep"); err != nil {
		t.Fatalf("DeleteResource failed: %v", err)
	}
	_, ok, err = db.GetResource(ctx, "ripgrep")
	if err != nil || ok {
		t.Fatalf("expected resource deleted, got ok=%v err=%v", ok, err)
	}
}

func TestRepositories_ConfigBasesCRUD(t *testing.T) {
	dir := t.TempDir()
	db, err := Open(context.Background(), Options{Path: filepath.Join(dir, "anx.db")})
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer func() { _ = db.Close() }()

	ctx := context.Background()

	cfg := ConfigBaseRecord{
		ConfigID:      "starship",
		ContentDigest: "sha256:abc1234",
		Size:          42,
		EOL:           "lf",
		Content:       "[character]\nsuccess_symbol = '❯'\n",
	}

	if err := db.UpsertConfigBase(ctx, cfg); err != nil {
		t.Fatalf("UpsertConfigBase failed: %v", err)
	}

	got, ok, err := db.GetConfigBase(ctx, "starship")
	if err != nil || !ok {
		t.Fatalf("GetConfigBase failed: ok=%v err=%v", ok, err)
	}
	if got.Content != cfg.Content || got.Size != 42 {
		t.Errorf("unexpected config base: %+v", got)
	}

	all, err := db.AllConfigBases(ctx)
	if err != nil {
		t.Fatalf("AllConfigBases failed: %v", err)
	}
	if len(all) != 1 {
		t.Errorf("expected 1 config base, got %d", len(all))
	}
}

func TestRepositories_AppliedHacksCRUD(t *testing.T) {
	dir := t.TempDir()
	db, err := Open(context.Background(), Options{Path: filepath.Join(dir, "anx.db")})
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer func() { _ = db.Close() }()

	ctx := context.Background()

	hack := AppliedHackRecord{
		HackID:          "dell-m6800-fan",
		HostFingerprint: "host-12345",
		Steps:           []string{"step1", "step2"},
		Version:         "1.0.0",
	}

	if err := db.RecordHack(ctx, hack); err != nil {
		t.Fatalf("RecordHack failed: %v", err)
	}

	applied, err := db.IsHackApplied(ctx, "dell-m6800-fan")
	if err != nil || !applied {
		t.Fatalf("expected hack applied, got applied=%v err=%v", applied, err)
	}

	all, err := db.AllAppliedHacks(ctx)
	if err != nil {
		t.Fatalf("AllAppliedHacks failed: %v", err)
	}
	if len(all) != 1 || len(all["dell-m6800-fan"].Steps) != 2 {
		t.Errorf("unexpected all applied hacks: %+v", all)
	}
}

func TestRepositories_MetadataAndReset(t *testing.T) {
	dir := t.TempDir()
	db, err := Open(context.Background(), Options{Path: filepath.Join(dir, "anx.db")})
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer func() { _ = db.Close() }()

	ctx := context.Background()

	if err := db.SetMetadata(ctx, "active_profile", "workstation"); err != nil {
		t.Fatalf("SetMetadata failed: %v", err)
	}

	val, ok, err := db.GetMetadata(ctx, "active_profile")
	if err != nil || !ok || val != "workstation" {
		t.Fatalf("GetMetadata failed: val=%s ok=%v err=%v", val, ok, err)
	}

	// Add a resource and reset
	_ = db.UpsertResource(ctx, ResourceRecord{ResourceID: "git", Provider: "apt", Strategy: "native"})
	if err := db.Reset(ctx); err != nil {
		t.Fatalf("Reset failed: %v", err)
	}

	allRes, err := db.AllResources(ctx)
	if err != nil || len(allRes) != 0 {
		t.Errorf("expected 0 resources after reset, got %d", len(allRes))
	}

	_, ok, _ = db.GetMetadata(ctx, "active_profile")
	if ok {
		t.Error("expected active_profile cleared after reset")
	}
}

func TestRepositories_DetectOrphaned(t *testing.T) {
	dir := t.TempDir()
	db, err := Open(context.Background(), Options{Path: filepath.Join(dir, "anx.db")})
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer func() { _ = db.Close() }()

	ctx := context.Background()

	_ = db.UpsertResource(ctx, ResourceRecord{ResourceID: "pkg-a", Provider: "apt", Strategy: "native"})
	_ = db.UpsertResource(ctx, ResourceRecord{ResourceID: "pkg-b", Provider: "apt", Strategy: "native"})
	_ = db.UpsertResource(ctx, ResourceRecord{ResourceID: "pkg-c", Provider: "apt", Strategy: "native"})

	orphans, err := db.DetectOrphaned(ctx, []string{"pkg-b"})
	if err != nil {
		t.Fatalf("DetectOrphaned failed: %v", err)
	}
	if len(orphans) != 2 {
		t.Fatalf("expected 2 orphans, got %d", len(orphans))
	}
	if orphans[0].ResourceID != "pkg-a" || orphans[1].ResourceID != "pkg-c" {
		t.Errorf("unexpected orphans: %+v", orphans)
	}
}

func TestDB_CheckIntegrityAndBackup(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "anx.db")
	db, err := Open(context.Background(), Options{Path: dbPath})
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer func() { _ = db.Close() }()

	ctx := context.Background()
	if err := db.CheckIntegrity(ctx); err != nil {
		t.Fatalf("CheckIntegrity failed on fresh db: %v", err)
	}

	backupPath := filepath.Join(dir, "backup", "anx.backup.db")
	if err := db.BackupTo(ctx, backupPath); err != nil {
		t.Fatalf("BackupTo failed: %v", err)
	}

	// Open backup and verify
	backupDB, err := Open(ctx, Options{Path: backupPath})
	if err != nil {
		t.Fatalf("Open backup failed: %v", err)
	}
	defer func() { _ = backupDB.Close() }()

	if err := backupDB.CheckIntegrity(ctx); err != nil {
		t.Fatalf("CheckIntegrity on backup failed: %v", err)
	}
}

// TestDB_ConcurrentMultiProcessSimulation simulates multiple processes opening the database concurrently
// and writing records. In SQLite with WAL mode and serialized transactions, 0 records are lost.
func TestDB_ConcurrentMultiProcessSimulation(t *testing.T) {
	for _, instances := range []int{2, 4, 8} {
		t.Run(fmt.Sprintf("%d_processes", instances), func(t *testing.T) {
			dir := t.TempDir()
			dbPath := filepath.Join(dir, "anx.db")

			const writesPerInstance = 15

			var wg sync.WaitGroup
			errCh := make(chan error, instances*writesPerInstance)

			for inst := range instances {
				wg.Add(1)
				go func(instanceID int) {
					defer wg.Done()

					// Each instance opens its own DB connection handle to simulate a separate OS process
					db, err := Open(context.Background(), Options{Path: dbPath})
					if err != nil {
						errCh <- fmt.Errorf("instance %d open: %w", instanceID, err)
						return
					}
					defer func() { _ = db.Close() }()

					for i := range writesPerInstance {
						resID := fmt.Sprintf("multi-inst-%d-%d", instanceID, i)
						rec := ResourceRecord{
							ResourceID:   resID,
							Version:      "1.0",
							Provider:     "test",
							Strategy:     "native",
							ManagedByAnx: true,
						}
						if err := db.UpsertResource(context.Background(), rec); err != nil {
							errCh <- fmt.Errorf("instance %d record %s: %w", instanceID, resID, err)
							return
						}
					}
				}(inst)
			}

			wg.Wait()
			close(errCh)

			for err := range errCh {
				t.Errorf("concurrent write error: %v", err)
			}

			finalDB, err := Open(context.Background(), Options{Path: dbPath})
			if err != nil {
				t.Fatalf("final open failed: %v", err)
			}
			defer func() { _ = finalDB.Close() }()

			all, err := finalDB.AllResources(context.Background())
			if err != nil {
				t.Fatalf("AllResources failed: %v", err)
			}

			expected := instances * writesPerInstance
			if len(all) != expected {
				t.Fatalf("expected %d records across concurrent processes, got %d", expected, len(all))
			}
		})
	}
}
