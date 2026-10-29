package snapshot_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"

	"github.com/arturonavax/env/internal/security"
	"github.com/arturonavax/env/internal/snapshot"
)

func TestSnapshotRecursiveDirsAndCollidingBasenames(t *testing.T) {
	home := t.TempDir()
	backupDir := filepath.Join(t.TempDir(), "custom-backups")

	herdrCfg := filepath.Join(home, ".config/herdr/config.toml")
	atuinCfg := filepath.Join(home, ".config/atuin/config.toml")
	nvimDir := filepath.Join(home, ".config/nvim")
	nvimInit := filepath.Join(nvimDir, "init.lua")
	nvimLsp := filepath.Join(nvimDir, "lua/plugins/lsp.lua")
	nvimHelper := filepath.Join(nvimDir, "bin/helper.sh")

	seed := map[string]struct {
		content string
		mode    os.FileMode
	}{
		herdrCfg:   {content: "prefix = 'ctrl-a'\n", mode: 0o644},
		atuinCfg:   {content: "auto_sync = false\n", mode: 0o600},
		nvimInit:   {content: "require('config.lazy')\n", mode: 0o644},
		nvimLsp:    {content: "return { 'neovim/nvim-lspconfig' }\n", mode: 0o644},
		nvimHelper: {content: "#!/usr/bin/env bash\necho helper\n", mode: 0o755},
	}

	for p, entry := range seed {
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(entry.content), entry.mode); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(p, entry.mode); err != nil {
			t.Fatal(err)
		}
	}

	mgr, err := snapshot.NewManagerAtDir(backupDir)
	if err != nil {
		t.Fatalf("NewManagerAtDir: %v", err)
	}

	snap, err := mgr.CreateSnapshot("pre-upgrade", []string{herdrCfg, atuinCfg, nvimDir})
	if err != nil {
		t.Fatalf("CreateSnapshot: %v", err)
	}
	if len(snap.Files) != 5 {
		t.Fatalf("len(snap.Files) = %d, want 5", len(snap.Files))
	}

	// Ensure colliding basenames live under distinct payload subpaths.
	if snap.Files[herdrCfg] == snap.Files[atuinCfg] {
		t.Fatalf("colliding payload path for herdr and atuin config.toml: %s", snap.Files[herdrCfg])
	}
	if !strings.Contains(filepath.ToSlash(snap.Files[herdrCfg]), "/payload/") {
		t.Fatalf("expected payload hierarchy in %q", snap.Files[herdrCfg])
	}

	if err := mgr.VerifySnapshot(snap.ID); err != nil {
		t.Fatalf("VerifySnapshot on clean snapshot failed: %v", err)
	}

	// Mutate both colliding files, delete a file inside nvimDir, and add an untracked stray file.
	if err := os.WriteFile(herdrCfg, []byte("corrupted herdr"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(atuinCfg, []byte("corrupted atuin"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(nvimLsp); err != nil {
		t.Fatal(err)
	}
	strayFile := filepath.Join(nvimDir, "lua/plugins/stray.lua")
	if err := os.WriteFile(strayFile, []byte("rogue"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := mgr.Restore(snap.ID); err != nil {
		t.Fatalf("Restore failed: %v", err)
	}

	for p, entry := range seed {
		got, err := os.ReadFile(p)
		if err != nil {
			t.Fatalf("reading restored %q: %v", p, err)
		}
		if string(got) != entry.content {
			t.Errorf("restored %q = %q, want %q", p, string(got), entry.content)
		}
		if runtime.GOOS != "windows" {
			info, err := os.Stat(p)
			if err != nil {
				t.Fatal(err)
			}
			if info.Mode().Perm() != entry.mode {
				t.Errorf("restored %q mode = %o, want %o", p, info.Mode().Perm(), entry.mode)
			}
		}
	}

	if _, err := os.Stat(strayFile); !os.IsNotExist(err) {
		t.Errorf("expected stray file %q inside tracked directory to be pruned on restore", strayFile)
	}
}

func TestSnapshotTamperDetectionBlocksRestore(t *testing.T) {
	workDir := t.TempDir()
	mgr, err := snapshot.NewManager(workDir)
	if err != nil {
		t.Fatal(err)
	}

	target := filepath.Join(workDir, "starship.toml")
	if err := os.WriteFile(target, []byte("add_newline = false\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	snap, err := mgr.CreateSnapshot("tamper-test", []string{target})
	if err != nil {
		t.Fatal(err)
	}

	// Mutate live target to verify failed restore does not overwrite it with tampered payload.
	if err := os.WriteFile(target, []byte("add_newline = true\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Tamper directly with the payload file inside snapshot storage.
	payloadPath := snap.Files[target]
	if err := os.WriteFile(payloadPath, []byte("TAMPERED PAYLOAD\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := mgr.VerifySnapshot(snap.ID); !errors.Is(err, security.ErrChecksumMismatch) {
		t.Fatalf("expected ErrChecksumMismatch from VerifySnapshot, got: %v", err)
	}

	if err := mgr.Restore(snap.ID); !errors.Is(err, security.ErrChecksumMismatch) {
		t.Fatalf("expected ErrChecksumMismatch from Restore, got: %v", err)
	}

	liveContent, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(liveContent) != "add_newline = true\n" {
		t.Fatalf("tampered payload must not overwrite live target, got: %q", string(liveContent))
	}
}

func TestSnapshotSubSecondCollisionsAndRestoreLatest(t *testing.T) {
	workDir := t.TempDir()
	mgr, err := snapshot.NewManager(workDir)
	if err != nil {
		t.Fatal(err)
	}

	// RestoreLatest on empty store must return ErrNoSnapshots.
	if _, err := mgr.RestoreLatest(); !errors.Is(err, snapshot.ErrNoSnapshots) {
		t.Fatalf("expected ErrNoSnapshots, got: %v", err)
	}

	target := filepath.Join(workDir, "rapid.conf")

	// Rapid concurrent + sequential sub-second creation without ID collisions.
	var wg sync.WaitGroup
	errs := make(chan error, 4)
	for i := range 4 {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			f := filepath.Join(workDir, "concurrent-"+string(rune('a'+idx))+".txt")
			_ = os.WriteFile(f, []byte("payload"), 0o644)
			_, cErr := mgr.CreateSnapshot("rapid-concurrent", []string{f})
			if cErr != nil {
				errs <- cErr
			}
		}(i)
	}
	wg.Wait()
	close(errs)
	for cErr := range errs {
		t.Fatalf("concurrent CreateSnapshot failed: %v", cErr)
	}

	// Create deterministic latest snapshot for target.
	if err := os.WriteFile(target, []byte("v1"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err = mgr.CreateSnapshot("snap-v1", []string{target})
	if err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(target, []byte("v2-latest"), 0o644); err != nil {
		t.Fatal(err)
	}
	latestCreated, err := mgr.CreateSnapshot("snap-v2", []string{target})
	if err != nil {
		t.Fatal(err)
	}

	// Mutate live target to v3, then RestoreLatest -> must restore v2-latest.
	if err := os.WriteFile(target, []byte("v3-dirty"), 0o644); err != nil {
		t.Fatal(err)
	}

	restoredSnap, err := mgr.RestoreLatest()
	if err != nil {
		t.Fatalf("RestoreLatest failed: %v", err)
	}
	if restoredSnap.ID != latestCreated.ID {
		t.Fatalf("RestoreLatest picked %q, want %q", restoredSnap.ID, latestCreated.ID)
	}

	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "v2-latest" {
		t.Fatalf("restored target = %q, want 'v2-latest'", string(got))
	}
}

func TestSnapshotQuotaRotationRetainsMinimumThree(t *testing.T) {
	workDir := t.TempDir()
	mgr, err := snapshot.NewManager(workDir)
	if err != nil {
		t.Fatal(err)
	}
	// Configure 4 KB quota so five 2 KB snapshots trigger rotation down to the minimum 3 retained snapshots.
	mgr.SetQuotaBytes(4 * 1024)

	payloadFile := filepath.Join(workDir, "blob.bin")
	chunk := bytes.Repeat([]byte("A"), 2048)
	if err := os.WriteFile(payloadFile, chunk, 0o644); err != nil {
		t.Fatal(err)
	}

	createdIDs := make([]string, 0, 5)
	for range 5 {
		s, err := mgr.CreateSnapshot("quota-step", []string{payloadFile})
		if err != nil {
			t.Fatalf("CreateSnapshot during quota rotation: %v", err)
		}
		createdIDs = append(createdIDs, s.ID)
	}

	snaps, err := mgr.ListSnapshots()
	if err != nil {
		t.Fatalf("ListSnapshots: %v", err)
	}
	if len(snaps) != snapshot.MinRetainedSnapshots {
		t.Fatalf("retained snapshots = %d, want %d", len(snaps), snapshot.MinRetainedSnapshots)
	}

	// Verify the newest 3 snapshots were retained and the oldest 2 were pruned.
	if snaps[0].ID != createdIDs[4] || snaps[1].ID != createdIDs[3] || snaps[2].ID != createdIDs[2] {
		t.Fatalf("unexpected retained order: got [%s, %s, %s], created %v",
			snaps[0].ID, snaps[1].ID, snaps[2].ID, createdIDs)
	}
}
