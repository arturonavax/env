package cas_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/arturonavax/env/internal/cas"
)

func newStore(t *testing.T) (*cas.BlobStore, string) {
	t.Helper()
	dir := t.TempDir()
	store, err := cas.NewBlobStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	return store, dir
}

func blobFile(dir, digest string) string {
	return filepath.Join(dir, "cas", digest[:2], digest[2:])
}

// CAS-01 regression: same-size corrupted blob must NOT be reused by Put.
func TestPutRepairsSameSizeCorruptedBlob(t *testing.T) {
	store, dir := newStore(t)
	content := []byte("original uncorrupted data")

	digest, err := store.Put(content)
	if err != nil {
		t.Fatal(err)
	}

	tampered := bytes.Repeat([]byte("X"), len(content)) // identical size, different bytes
	if err := os.WriteFile(blobFile(dir, digest), tampered, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Get(digest); !errors.Is(err, cas.ErrCorruptedBlob) {
		t.Fatalf("precondition: expected corruption, got %v", err)
	}

	if _, err := store.Put(content); err != nil {
		t.Fatalf("Put must repair corrupted blob: %v", err)
	}
	got, err := store.Get(digest)
	if err != nil {
		t.Fatalf("blob still corrupted after Put: %v", err)
	}
	if !bytes.Equal(got, content) {
		t.Fatalf("repaired content mismatch: %q", got)
	}
}

func TestPutRepairsTruncatedBlob(t *testing.T) {
	store, dir := newStore(t)
	content := []byte("some longer content to truncate")
	digest, _ := store.Put(content)

	if err := os.WriteFile(blobFile(dir, digest), content[:5], 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Put(content); err != nil {
		t.Fatal(err)
	}
	if err := store.Verify(digest); err != nil {
		t.Fatalf("Verify after repair: %v", err)
	}
}

// Security: digest is a filesystem path; traversal must be rejected.
func TestDigestValidationRejectsTraversal(t *testing.T) {
	store, _ := newStore(t)

	bad := []string{
		"../../../../../../../../../../../../etc/passwd" + strings.Repeat("a", 17), // 64 chars, traversal
		strings.Repeat("A", 64), // uppercase
		strings.Repeat("g", 64), // non-hex
		strings.Repeat("a", 63),
		"",
		"aa/" + strings.Repeat("a", 61),
	}
	for _, d := range bad {
		if _, err := store.Get(d); !errors.Is(err, cas.ErrInvalidDigest) {
			t.Errorf("Get(%q): want ErrInvalidDigest, got %v", d, err)
		}
		if store.Has(d) {
			t.Errorf("Has(%q) must be false", d)
		}
		if err := store.Verify(d); !errors.Is(err, cas.ErrInvalidDigest) {
			t.Errorf("Verify(%q): want ErrInvalidDigest, got %v", d, err)
		}
	}
}

func TestVerifyStates(t *testing.T) {
	store, dir := newStore(t)
	digest, _ := store.Put([]byte("verify me"))

	if err := store.Verify(digest); err != nil {
		t.Fatalf("healthy blob: %v", err)
	}
	if err := store.Verify(strings.Repeat("0", 64)); !errors.Is(err, cas.ErrBlobNotFound) {
		t.Fatalf("missing blob: got %v", err)
	}
	if err := os.WriteFile(blobFile(dir, digest), []byte("nope"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := store.Verify(digest); !errors.Is(err, cas.ErrCorruptedBlob) {
		t.Fatalf("corrupt blob: got %v", err)
	}
}

// Real concurrency: many writers on the same and distinct blobs plus readers. Run under -race.
func TestConcurrentPutGet(t *testing.T) {
	store, _ := newStore(t)
	shared := []byte("shared payload written by every goroutine")

	var wg sync.WaitGroup
	for i := range 32 {
		wg.Go(func() {
			own := bytes.Repeat([]byte{byte(i)}, 1024+i)
			for range 20 {
				d1, err := store.Put(shared)
				if err != nil {
					t.Errorf("Put shared: %v", err)
					return
				}
				d2, err := store.Put(own)
				if err != nil {
					t.Errorf("Put own: %v", err)
					return
				}
				if got, err := store.Get(d1); err != nil || !bytes.Equal(got, shared) {
					t.Errorf("Get shared: %v", err)
					return
				}
				if got, err := store.Get(d2); err != nil || !bytes.Equal(got, own) {
					t.Errorf("Get own: %v", err)
					return
				}
			}
		})
	}
	wg.Wait()
}

func TestNoStagingLeftovers(t *testing.T) {
	store, dir := newStore(t)
	digest, _ := store.Put([]byte("leftover check"))

	entries, err := os.ReadDir(filepath.Join(dir, "cas", digest[:2]))
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "blob-") {
			t.Fatalf("staging file leaked: %s", e.Name())
		}
	}
}

func TestBlobStorePutAndGet(t *testing.T) {
	store, _ := newStore(t)

	content := []byte("hello content addressed storage")
	digest, err := store.Put(content)
	if err != nil {
		t.Fatalf("failed putting blob: %v", err)
	}
	if !store.Has(digest) {
		t.Fatalf("expected store.Has(%s) to be true", digest)
	}
	retrieved, err := store.Get(digest)
	if err != nil {
		t.Fatalf("failed getting blob: %v", err)
	}
	if !bytes.Equal(content, retrieved) {
		t.Fatalf("retrieved bytes mismatch: expected %q, got %q", content, retrieved)
	}
}

func TestBlobStoreCorruptedDetection(t *testing.T) {
	store, dir := newStore(t)
	digest, err := store.Put([]byte("original uncorrupted data"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(blobFile(dir, digest), []byte("tampered content"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err = store.Get(digest); !errors.Is(err, cas.ErrCorruptedBlob) {
		t.Fatalf("expected ErrCorruptedBlob, got: %v", err)
	}
}

func TestBlobStoreNotFound(t *testing.T) {
	store, _ := newStore(t)
	_, err := store.Get("e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855")
	if !errors.Is(err, cas.ErrBlobNotFound) {
		t.Fatalf("expected ErrBlobNotFound, got: %v", err)
	}
}
