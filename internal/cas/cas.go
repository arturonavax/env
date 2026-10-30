package cas

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// ErrCorruptedBlob indicates data on disk does not match its SHA-256 key.
var ErrCorruptedBlob = errors.New("CAS blob integrity check failed: SHA-256 mismatch")

// ErrBlobNotFound indicates requested SHA-256 blob is missing from storage.
var ErrBlobNotFound = errors.New("CAS blob not found")

// BlobStore provides content-addressed storage for file snapshots and state assets.
type BlobStore struct {
	rootDir string
	mu      sync.RWMutex
}

// NewBlobStore creates or opens a content-addressed storage root at dir/cas.
func NewBlobStore(dir string) (*BlobStore, error) {
	casDir := filepath.Join(dir, "cas")
	if err := os.MkdirAll(casDir, 0o700); err != nil {
		return nil, fmt.Errorf("creating CAS root directory: %w", err)
	}
	return &BlobStore{rootDir: casDir}, nil
}

// Put writes data content-addressed by its SHA-256 digest with atomic fsync.
// Returns the hex-encoded SHA-256 digest.
func (s *BlobStore) Put(data []byte) (string, error) {
	h := sha256.Sum256(data)
	digest := hex.EncodeToString(h[:])

	prefix := digest[:2]
	rest := digest[2:]
	bucketDir := filepath.Join(s.rootDir, prefix)
	finalPath := filepath.Join(bucketDir, rest)

	// Fast-path: check if blob is already present and matches expected size
	if info, err := os.Stat(finalPath); err == nil && info.Size() == int64(len(data)) {
		return digest, nil
	}

	if err := os.MkdirAll(bucketDir, 0o700); err != nil {
		return "", fmt.Errorf("creating CAS bucket dir: %w", err)
	}

	tempFile, err := os.CreateTemp(bucketDir, "blob-*")
	if err != nil {
		return "", fmt.Errorf("creating CAS staging file: %w", err)
	}
	tempName := tempFile.Name()

	if _, err := tempFile.Write(data); err != nil {
		_ = tempFile.Close()
		_ = os.Remove(tempName)
		return "", fmt.Errorf("writing CAS blob: %w", err)
	}

	if err := tempFile.Sync(); err != nil {
		_ = tempFile.Close()
		_ = os.Remove(tempName)
		return "", fmt.Errorf("syncing CAS blob: %w", err)
	}

	if err := tempFile.Close(); err != nil {
		_ = os.Remove(tempName)
		return "", fmt.Errorf("closing CAS staging file: %w", err)
	}

	// Short critical section: only synchronize atomic rename into final CAS location
	s.mu.Lock()
	err = os.Rename(tempName, finalPath)
	s.mu.Unlock()

	if err != nil {
		_ = os.Remove(tempName)
		return "", fmt.Errorf("committing CAS blob: %w", err)
	}

	return digest, nil
}

// Get retrieves data by its SHA-256 digest, verifying content integrity.
func (s *BlobStore) Get(digest string) ([]byte, error) {
	if len(digest) != 64 {
		return nil, fmt.Errorf("invalid SHA-256 digest length: %s", digest)
	}

	prefix := digest[:2]
	rest := digest[2:]
	blobPath := filepath.Join(s.rootDir, prefix, rest)

	s.mu.RLock()
	defer s.mu.RUnlock()

	data, err := os.ReadFile(blobPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrBlobNotFound
		}
		return nil, fmt.Errorf("reading CAS blob: %w", err)
	}

	h := sha256.Sum256(data)
	computed := hex.EncodeToString(h[:])
	if computed != digest {
		return nil, fmt.Errorf("%w: expected %s, got %s", ErrCorruptedBlob, digest, computed)
	}

	return data, nil
}

// Has checks whether a blob exists in the store.
func (s *BlobStore) Has(digest string) bool {
	if len(digest) != 64 {
		return false
	}
	prefix := digest[:2]
	rest := digest[2:]
	blobPath := filepath.Join(s.rootDir, prefix, rest)

	s.mu.RLock()
	defer s.mu.RUnlock()

	_, err := os.Stat(blobPath)
	return err == nil
}
