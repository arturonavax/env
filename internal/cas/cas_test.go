package cas_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/arturonavax/env/internal/cas"
)

func TestBlobStorePutAndGet(t *testing.T) {
	tempDir := t.TempDir()
	store, err := cas.NewBlobStore(tempDir)
	if err != nil {
		t.Fatal(err)
	}

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
	tempDir := t.TempDir()
	store, err := cas.NewBlobStore(tempDir)
	if err != nil {
		t.Fatal(err)
	}

	content := []byte("original uncorrupted data")
	digest, err := store.Put(content)
	if err != nil {
		t.Fatal(err)
	}

	// Corrupt file on disk
	prefix := digest[:2]
	rest := digest[2:]
	blobPath := filepath.Join(tempDir, "cas", prefix, rest)
	if err := os.WriteFile(blobPath, []byte("tampered content"), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err = store.Get(digest)
	if err == nil {
		t.Fatal("expected error on corrupted blob, got nil")
	}
	if !errors.Is(err, cas.ErrCorruptedBlob) {
		t.Fatalf("expected ErrCorruptedBlob, got: %v", err)
	}
}

func TestBlobStoreNotFound(t *testing.T) {
	tempDir := t.TempDir()
	store, err := cas.NewBlobStore(tempDir)
	if err != nil {
		t.Fatal(err)
	}

	_, err = store.Get("e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855")
	if !errors.Is(err, cas.ErrBlobNotFound) {
		t.Fatalf("expected ErrBlobNotFound, got: %v", err)
	}
}
