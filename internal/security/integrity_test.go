package security

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/arturonavax/env/internal/domain"
)

func TestCreateOriginSeal_Embedded(t *testing.T) {
	seal, err := CreateOriginSeal()
	if err != nil {
		t.Fatalf("failed to create embedded origin seal: %v", err)
	}
	if seal == nil {
		t.Fatal("expected non-nil seal")
	}
	if !seal.IsEmbedded {
		t.Errorf("expected IsEmbedded to be true, got false")
	}
	if seal.ProvenanceOrigin() != "Embedded Catalog (Immutable Binary)" {
		t.Errorf("unexpected provenance origin: %s", seal.ProvenanceOrigin())
	}
	if len(seal.CompositeDigest) != 64 {
		t.Errorf("expected 64-char composite digest, got %d", len(seal.CompositeDigest))
	}
	if len(seal.ShortDigest()) != 12 {
		t.Errorf("expected 12-char short digest, got %s", seal.ShortDigest())
	}
	if len(seal.ManifestFingerprint) != 64 {
		t.Errorf("expected 64-char manifest fingerprint, got %s", seal.ManifestFingerprint)
	}
	if seal.AssetCount == 0 {
		t.Errorf("expected embedded asset count > 0, got %d", seal.AssetCount)
	}

	// Verify embedded seal
	ctx := context.Background()
	if err := seal.Verify(ctx); err != nil {
		t.Errorf("expected embedded seal verification to pass, got: %v", err)
	}
}

func TestCreateOriginSeal_WithRepoAndTamper(t *testing.T) {
	tmpDir := t.TempDir()
	manDir := filepath.Join(tmpDir, "manifests")
	filesDir := filepath.Join(tmpDir, "files")
	if err := os.MkdirAll(manDir, 0755); err != nil {
		t.Fatalf("mkdir manifests: %v", err)
	}
	if err := os.MkdirAll(filesDir, 0755); err != nil {
		t.Fatalf("mkdir files: %v", err)
	}

	pkgFile := filepath.Join(manDir, "custom.yaml")
	if err := os.WriteFile(pkgFile, []byte("packages:\n  - id: custom\n    name: Custom\n"), 0644); err != nil {
		t.Fatalf("write custom.yaml: %v", err)
	}

	dotFile := filepath.Join(filesDir, "customrc")
	if err := os.WriteFile(dotFile, []byte("export CUSTOM=1\n"), 0644); err != nil {
		t.Fatalf("write customrc: %v", err)
	}

	seal, err := CreateOriginSeal(WithSealRepoRoot(tmpDir))
	if err != nil {
		t.Fatalf("CreateOriginSeal with repo failed: %v", err)
	}

	if seal.IsEmbedded {
		t.Errorf("expected IsEmbedded=false when repo has manifests")
	}
	if seal.ProvenanceOrigin() != tmpDir {
		t.Errorf("expected provenance origin %s, got %s", tmpDir, seal.ProvenanceOrigin())
	}

	ctx := context.Background()
	// Verification should succeed initially
	if err := seal.Verify(ctx); err != nil {
		t.Fatalf("initial seal verification failed: %v", err)
	}

	// Tamper 1: modify dotfile
	if err := os.WriteFile(dotFile, []byte("export CUSTOM=2\n"), 0644); err != nil {
		t.Fatalf("tamper write: %v", err)
	}
	tErr := seal.Verify(ctx)
	if tErr == nil {
		t.Fatal("expected tamper detection after dotfile modification, but got nil")
	}
	var exitErr *domain.ExitError
	if !errors.As(tErr, &exitErr) || exitErr.Code != domain.ExitSecurityViolation {
		t.Errorf("expected ExitSecurityViolation (%d), got: %v", domain.ExitSecurityViolation, tErr)
	}

	// Restore dotfile
	_ = os.WriteFile(dotFile, []byte("export CUSTOM=1\n"), 0644)
	if err := seal.Verify(ctx); err != nil {
		t.Fatalf("restored dotfile verification failed: %v", err)
	}

	// Tamper 2: inject new manifest file
	injectedFile := filepath.Join(manDir, "injected.yaml")
	if err := os.WriteFile(injectedFile, []byte("packages: []\n"), 0644); err != nil {
		t.Fatalf("inject manifest: %v", err)
	}
	tErr = seal.Verify(ctx)
	if tErr == nil {
		t.Fatal("expected tamper detection after injecting manifest, but got nil")
	}
	if !errors.As(tErr, &exitErr) || exitErr.Code != domain.ExitSecurityViolation {
		t.Errorf("expected ExitSecurityViolation for injected manifest, got: %v", tErr)
	}

	// Remove injected file
	_ = os.Remove(injectedFile)
	if err := seal.Verify(ctx); err != nil {
		t.Fatalf("verification after removing injected file failed: %v", err)
	}

	// Tamper 3: delete original manifest file
	_ = os.Remove(pkgFile)
	tErr = seal.Verify(ctx)
	if tErr == nil {
		t.Fatal("expected tamper detection after deleting manifest, but got nil")
	}
	if !errors.As(tErr, &exitErr) || exitErr.Code != domain.ExitSecurityViolation {
		t.Errorf("expected ExitSecurityViolation for deleted manifest, got: %v", tErr)
	}
}

func TestOriginSeal_UntrustedOrigin(t *testing.T) {
	tmpDir := t.TempDir()
	manDir := filepath.Join(tmpDir, "manifests")
	if err := os.MkdirAll(manDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	// Make directory world-writable (mode 0777)
	if err := os.Chmod(tmpDir, 0777); err != nil {
		t.Fatalf("chmod: %v", err)
	}

	seal, err := CreateOriginSeal(WithSealRepoRoot(tmpDir))
	if err != nil {
		t.Fatalf("CreateOriginSeal failed: %v", err)
	}

	if !seal.InsecureOrigin {
		t.Errorf("expected InsecureOrigin to be true for world-writable dir")
	}

	ctx := context.Background()
	vErr := seal.Verify(ctx)
	if vErr == nil {
		t.Fatal("expected untrusted origin error during verify, got nil")
	}
	var exitErr *domain.ExitError
	if !errors.As(vErr, &exitErr) || exitErr.Code != domain.ExitSecurityViolation {
		t.Errorf("expected ExitSecurityViolation for untrusted origin, got: %v", vErr)
	}

	// Fix permissions
	_ = os.Chmod(tmpDir, 0755)
}

func TestOriginSeal_ConcurrentVerify(t *testing.T) {
	seal, err := CreateOriginSeal()
	if err != nil {
		t.Fatalf("CreateOriginSeal failed: %v", err)
	}

	const goroutines = 20
	var wg sync.WaitGroup
	wg.Add(goroutines)

	ctx := context.Background()
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < 5; j++ {
				_ = seal.ShortDigest()
				_ = seal.ShortManifestDigest()
				_ = seal.ShortDotfilesDigest()
				_ = seal.ProvenanceOrigin()
				if vErr := seal.Verify(ctx); vErr != nil {
					t.Errorf("concurrent verify error: %v", vErr)
				}
			}
		}()
	}

	wg.Wait()
}
