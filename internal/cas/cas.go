package cas

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
)

// digestLen is the hex-encoded length of a SHA-256 digest.
const digestLen = sha256.Size * 2

// ErrCorruptedBlob indicates data on disk does not match its SHA-256 key.
var ErrCorruptedBlob = errors.New("CAS blob integrity check failed: SHA-256 mismatch")

// ErrBlobNotFound indicates requested SHA-256 blob is missing from storage.
var ErrBlobNotFound = errors.New("CAS blob not found")

// ErrInvalidDigest indicates a digest is not a lowercase hex-encoded SHA-256 value.
var ErrInvalidDigest = errors.New("invalid SHA-256 digest")

// BlobStore provides content-addressed storage for file snapshots and state assets.
//
// Concurrency model: blobs are immutable once committed, so readers only need to be
// excluded from the short atomic-rename critical section. Hashing and I/O run outside
// the write lock.
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

// validateDigest rejects anything that is not a lowercase hex SHA-256. This is a security
// boundary: the digest becomes a filesystem path, so path separators and ".." must never pass.
func validateDigest(digest string) error {
	if len(digest) != digestLen {
		return fmt.Errorf("%w: length %d, want %d", ErrInvalidDigest, len(digest), digestLen)
	}
	for i := range len(digest) {
		c := digest[i]
		isDigit := c >= '0' && c <= '9'
		isLowerHex := c >= 'a' && c <= 'f'
		if !isDigit && !isLowerHex {
			return fmt.Errorf("%w: non-hex character at offset %d", ErrInvalidDigest, i)
		}
	}
	return nil
}

func (s *BlobStore) blobPath(digest string) (bucketDir, finalPath string) {
	bucketDir = filepath.Join(s.rootDir, digest[:2])
	return bucketDir, filepath.Join(bucketDir, digest[2:])
}

// Put writes data content-addressed by its SHA-256 digest with atomic fsync.
// An existing blob is only reused after its content hash has been re-verified;
// a corrupted blob is transparently replaced. Returns the hex-encoded SHA-256 digest.
func (s *BlobStore) Put(data []byte) (string, error) {
	sum := sha256.Sum256(data)
	digest := hex.EncodeToString(sum[:])
	bucketDir, finalPath := s.blobPath(digest)

	// Fast-path: reuse only if the on-disk content really hashes to the digest.
	// Size equality is NOT proof of integrity (bit-rot, truncation+padding, tampering).
	if s.matchesDigest(finalPath, sum) {
		return digest, nil
	}

	if err := os.MkdirAll(bucketDir, 0o700); err != nil {
		return "", fmt.Errorf("creating CAS bucket dir: %w", err)
	}

	tempName, err := stageBlob(bucketDir, data)
	if err != nil {
		return "", err
	}

	// Short critical section: only synchronize the atomic rename into the final location.
	// Rename replaces any corrupted predecessor atomically on POSIX and Windows.
	s.mu.Lock()
	err = os.Rename(tempName, finalPath)
	s.mu.Unlock()
	if err != nil {
		_ = os.Remove(tempName)
		return "", fmt.Errorf("committing CAS blob: %w", err)
	}

	syncDir(bucketDir)
	return digest, nil
}

// stageBlob writes data to a fsynced temporary file inside dir and returns its path.
func stageBlob(dir string, data []byte) (string, error) {
	tempFile, err := os.CreateTemp(dir, "blob-*")
	if err != nil {
		return "", fmt.Errorf("creating CAS staging file: %w", err)
	}
	tempName := tempFile.Name()

	fail := func(op string, err error) (string, error) {
		_ = tempFile.Close()
		_ = os.Remove(tempName)
		return "", fmt.Errorf("%s: %w", op, err)
	}

	if _, err := tempFile.Write(data); err != nil {
		return fail("writing CAS blob", err)
	}
	if err := tempFile.Sync(); err != nil {
		return fail("syncing CAS blob", err)
	}
	if err := tempFile.Close(); err != nil {
		_ = os.Remove(tempName)
		return "", fmt.Errorf("closing CAS staging file: %w", err)
	}
	return tempName, nil
}

// matchesDigest streams the file through SHA-256 (constant memory) and compares to want.
func (s *BlobStore) matchesDigest(path string, want [sha256.Size]byte) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return false
	}

	var got [sha256.Size]byte
	copy(got[:], h.Sum(nil))
	return got == want
}

// syncDir best-effort fsyncs a directory so the rename survives power loss.
// Unsupported on Windows; failures are non-fatal because the blob is already durable.
func syncDir(dir string) {
	d, err := os.Open(dir)
	if err != nil {
		return
	}
	_ = d.Sync()
	_ = d.Close()
}

// Get retrieves data by its SHA-256 digest, verifying content integrity.
func (s *BlobStore) Get(digest string) ([]byte, error) {
	if err := validateDigest(digest); err != nil {
		return nil, err
	}
	_, blobPath := s.blobPath(digest)

	s.mu.RLock()
	data, err := os.ReadFile(blobPath)
	s.mu.RUnlock()
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrBlobNotFound
		}
		return nil, fmt.Errorf("reading CAS blob: %w", err)
	}

	h := sha256.Sum256(data)
	if computed := hex.EncodeToString(h[:]); computed != digest {
		return nil, fmt.Errorf("%w: expected %s, got %s", ErrCorruptedBlob, digest, computed)
	}
	return data, nil
}

// Has reports whether a blob file exists for digest. It does NOT verify content;
// use Verify when integrity matters.
func (s *BlobStore) Has(digest string) bool {
	if validateDigest(digest) != nil {
		return false
	}
	_, blobPath := s.blobPath(digest)

	s.mu.RLock()
	defer s.mu.RUnlock()

	_, err := os.Stat(blobPath)
	return err == nil
}

// Verify streams the stored blob and confirms it hashes to digest without loading it in memory.
func (s *BlobStore) Verify(digest string) error {
	if err := validateDigest(digest); err != nil {
		return err
	}
	_, blobPath := s.blobPath(digest)

	raw, _ := hex.DecodeString(digest)
	var want [sha256.Size]byte
	copy(want[:], raw)

	if _, err := os.Stat(blobPath); err != nil {
		if os.IsNotExist(err) {
			return ErrBlobNotFound
		}
		return fmt.Errorf("stat CAS blob: %w", err)
	}
	if !s.matchesDigest(blobPath, want) {
		return fmt.Errorf("%w: %s", ErrCorruptedBlob, digest)
	}
	return nil
}
