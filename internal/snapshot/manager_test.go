package snapshot_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/arturonavax/env/internal/snapshot"
)

func TestSnapshotCreateAndRestore(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "anx-snapshot-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	mgr, err := snapshot.NewManager(tempDir)
	if err != nil {
		t.Fatal(err)
	}

	testFile := filepath.Join(tempDir, "config.txt")
	err = os.WriteFile(testFile, []byte("original content"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	snap, err := mgr.CreateSnapshot("pre-test", []string{testFile})
	if err != nil {
		t.Fatal(err)
	}
	if len(snap.Files) != 1 {
		t.Fatalf("expected 1 file in snapshot, got %d", len(snap.Files))
	}

	// Overwrite file
	err = os.WriteFile(testFile, []byte("mutated content"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// Restore
	err = mgr.Restore(snap.ID)
	if err != nil {
		t.Fatal(err)
	}

	restored, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(restored) != "original content" {
		t.Fatalf("expected 'original content', got '%s'", string(restored))
	}
}
