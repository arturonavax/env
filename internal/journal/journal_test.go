package journal_test

import (
	"os"
	"testing"
	"time"

	"github.com/arturonavax/env/internal/domain"
	"github.com/arturonavax/env/internal/journal"
)

func TestJournalAppendAndInterruptedDetection(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "anx-journal-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	j, err := journal.NewFileJournal(tempDir)
	if err != nil {
		t.Fatal(err)
	}

	txID := "tx-test-01"
	err = j.Append(domain.JournalEntry{
		TxID:      txID,
		Timestamp: time.Now(),
		Type:      "begin",
	})
	if err != nil {
		t.Fatal(err)
	}

	err = j.Append(domain.JournalEntry{
		TxID:      txID,
		Timestamp: time.Now(),
		Type:      "step_done",
		StepID:    1,
		Action:    domain.ActionSync,
		Target:    "/path/to/test",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Because we haven't committed, it should be detected as Interrupted
	pending, err := j.GetLastPendingTx()
	if err != nil {
		t.Fatal(err)
	}
	if pending == nil {
		t.Fatalf("expected pending interrupted transaction, got nil")
	}
	if pending.State != domain.TxInterrupted {
		t.Fatalf("expected state Interrupted, got %s", pending.State.String())
	}

	// Now append commit
	err = j.Append(domain.JournalEntry{
		TxID:      txID,
		Timestamp: time.Now(),
		Type:      "commit",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Now it shouldn't be pending
	pendingAfterCommit, err := j.GetLastPendingTx()
	if err != nil {
		t.Fatal(err)
	}
	if pendingAfterCommit != nil {
		t.Fatalf("expected nil pending after commit, got %v", pendingAfterCommit)
	}
}

func TestJournalCorruptedEntryDetection(t *testing.T) {
	tempDir := t.TempDir()
	j, err := journal.NewFileJournal(tempDir)
	if err != nil {
		t.Fatal(err)
	}

	// Append valid entry
	err = j.Append(domain.JournalEntry{
		TxID:      "tx-valid",
		Timestamp: time.Now(),
		Type:      "begin",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Corrupt journal by appending raw invalid JSON
	journalFile := tempDir + "/journal.jsonl"
	f, err := os.OpenFile(journalFile, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = f.WriteString("{truncated_json: true, \n")
	_ = f.Close()

	// Reading entries must fail closed with ErrCorruptedJournal
	_, err = j.ReadEntries()
	if err == nil {
		t.Fatalf("expected error on corrupted journal, got nil")
	}
}
