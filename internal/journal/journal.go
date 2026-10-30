package journal

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/arturonavax/env/internal/domain"
)

// ErrCorruptedJournal indicates a corrupted or truncated entry was encountered in the journal file.
var ErrCorruptedJournal = errors.New("corrupted journal entry")

// FileJournal manages persistent append-only logging of transaction steps.
type FileJournal struct {
	path string
	mu   sync.RWMutex
}

// NewFileJournal creates or opens a FileJournal at stateDir/journal.jsonl.
func NewFileJournal(stateDir string) (*FileJournal, error) {
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		return nil, fmt.Errorf("creating state directory: %w", err)
	}
	path := filepath.Join(stateDir, "journal.jsonl")
	return &FileJournal{path: path}, nil
}

// Append writes a new entry to the journal file.
func (j *FileJournal) Append(entry domain.JournalEntry) error {
	j.mu.Lock()
	defer j.mu.Unlock()

	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now()
	}

	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("marshaling journal entry: %w", err)
	}

	f, err := os.OpenFile(j.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("opening journal file: %w", err)
	}
	defer f.Close()

	if _, err := f.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("writing journal entry: %w", err)
	}

	if err := f.Sync(); err != nil {
		return fmt.Errorf("syncing journal entry: %w", err)
	}

	return nil
}

// ReadEntries reads all entries from the journal.
func (j *FileJournal) ReadEntries() ([]domain.JournalEntry, error) {
	j.mu.RLock()
	defer j.mu.RUnlock()

	f, err := os.Open(j.path)
	if err != nil {
		if os.IsNotExist(err) {
			return []domain.JournalEntry{}, nil
		}
		return nil, fmt.Errorf("opening journal: %w", err)
	}
	defer f.Close()

	entries := make([]domain.JournalEntry, 0, 64)
	scanner := bufio.NewScanner(f)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var entry domain.JournalEntry
		if err := json.Unmarshal(line, &entry); err != nil {
			return nil, fmt.Errorf("%w at line %d: %v", ErrCorruptedJournal, lineNum, err)
		}
		entries = append(entries, entry)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scanning journal file: %w", err)
	}

	return entries, nil
}

// GetLastPendingTx scans the journal to check if the last transaction was interrupted before commit/rollback.
func (j *FileJournal) GetLastPendingTx() (*domain.Transaction, error) {
	entries, err := j.ReadEntries()
	if err != nil {
		return nil, err
	}
	if len(entries) == 0 {
		return nil, nil
	}

	// Map transactions by ID
	txMap := make(map[string]*domain.Transaction)
	var lastTxID string

	for _, e := range entries {
		lastTxID = e.TxID
		tx, exists := txMap[e.TxID]
		if !exists {
			tx = &domain.Transaction{
				ID:        e.TxID,
				StartedAt: e.Timestamp,
				State:     domain.TxRunning,
			}
			txMap[e.TxID] = tx
		}

		switch e.Type {
		case "begin":
			tx.State = domain.TxRunning
		case "step_done":
			tx.CompletedSteps++
		case "commit":
			tx.State = domain.TxCommitted
			tx.FinishedAt = e.Timestamp
		case "rollback":
			tx.State = domain.TxRolledBack
			tx.FinishedAt = e.Timestamp
		}
	}

	if lastTxID == "" {
		return nil, nil
	}

	lastTx := txMap[lastTxID]
	if lastTx.State == domain.TxRunning {
		lastTx.State = domain.TxInterrupted
		return lastTx, nil
	}

	return nil, nil
}
