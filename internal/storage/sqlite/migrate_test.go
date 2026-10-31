package sqlite

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestMigrateJSONToSQLite(t *testing.T) {
	stateDir := t.TempDir()

	sampleJSON := `{
  "schema_version": 1,
  "active_profile": "workstation",
  "active_theme": "tokyonight",
  "repo_root": "/home/user/repos/env",
  "resources": {
    "ripgrep": {
      "resource_id": "ripgrep",
      "version": "14.1.0",
      "strategy": "native",
      "provider": "apt",
      "managed_by_anx": true,
      "installed_path": "/usr/bin/rg"
    },
    "fd": {
      "resource_id": "fd",
      "version": "9.0.0",
      "strategy": "native",
      "provider": "apt",
      "managed_by_anx": true,
      "installed_path": "/usr/bin/fdfind"
    }
  },
  "config_bases": {
    "starship": {
      "config_id": "starship",
      "sha256": "abc123",
      "content": "[character]\nsymbol = '>'\n"
    }
  },
  "applied_hacks": {
    "dell-fan": {
      "hack_id": "dell-fan",
      "steps": ["stepA", "stepB"]
    }
  }
}`

	jsonPath := filepath.Join(stateDir, "state.json")
	if err := os.WriteFile(jsonPath, []byte(sampleJSON), 0600); err != nil {
		t.Fatalf("writing sample state.json: %v", err)
	}

	// 1. Plan Migration
	plan, _, err := PlanMigration(stateDir)
	if err != nil {
		t.Fatalf("PlanMigration failed: %v", err)
	}
	if plan.ResourceCount != 2 || plan.ConfigBaseCount != 1 || plan.AppliedHackCount != 1 {
		t.Fatalf("unexpected plan counts: %+v", plan)
	}

	// 2. Open DB and Apply
	dbPath := filepath.Join(stateDir, "anx.db")
	db, err := Open(context.Background(), Options{Path: dbPath})
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer func() { _ = db.Close() }()

	resPlan, err := ApplyMigration(context.Background(), stateDir, db)
	if err != nil {
		t.Fatalf("ApplyMigration failed: %v", err)
	}
	if resPlan == nil || resPlan.BackupPath == "" {
		t.Fatalf("expected completed migration plan with backup, got %+v", resPlan)
	}

	// Verify backup file exists
	if _, err := os.Stat(resPlan.BackupPath); err != nil {
		t.Fatalf("backup file not found at %s: %v", resPlan.BackupPath, err)
	}

	// Verify imported resources
	rg, ok, err := db.GetResource(context.Background(), "ripgrep")
	if err != nil || !ok || rg.Version != "14.1.0" {
		t.Fatalf("ripgrep record not found or wrong version: ok=%v err=%v rg=%+v", ok, err, rg)
	}

	// Verify metadata
	prof, ok, err := db.GetMetadata(context.Background(), "active_profile")
	if err != nil || !ok || prof != "workstation" {
		t.Fatalf("active_profile not imported: ok=%v err=%v prof=%s", ok, err, prof)
	}

	// 3. Idempotent re-run
	secondPlan, err := ApplyMigration(context.Background(), stateDir, db)
	if err != nil {
		t.Fatalf("second ApplyMigration failed: %v", err)
	}
	if !secondPlan.AlreadyMigrated {
		t.Fatalf("expected AlreadyMigrated=true on second run, got %+v", secondPlan)
	}
}
