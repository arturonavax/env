//go:build !windows

package security

import (
	"os"
	"path/filepath"
	"testing"
)

// privateDir creates a 0700 directory under a fresh temp root so the verdict does not depend on
// the permissions of the checkout or of $TMPDIR.
func privateDir(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "origin")
	if err := os.Mkdir(dir, 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	return dir
}

func TestUnixTrust_PrivateDirectoryIsTrusted(t *testing.T) {
	report := DefaultTrustVerifier().VerifyDirectory(privateDir(t))
	if report.Level != TrustTrusted {
		t.Fatalf("expected trusted, got %s: %s", report.Level, report.Reason())
	}
}

func TestUnixTrust_WorldWritableLeafIsUntrusted(t *testing.T) {
	dir := privateDir(t)
	if err := os.Chmod(dir, 0o777); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	report := DefaultTrustVerifier().VerifyDirectory(dir)
	if report.Level != TrustUntrusted {
		t.Fatalf("expected untrusted, got %s: %s", report.Level, report.Reason())
	}
}

func TestUnixTrust_WorldWritableStickyLeafIsStillUntrusted(t *testing.T) {
	dir := privateDir(t)
	if err := os.Chmod(dir, 0o777|os.ModeSticky); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	report := DefaultTrustVerifier().VerifyDirectory(dir)
	if report.Level != TrustUntrusted {
		t.Fatalf("a sticky bit must not legitimise a world-writable origin, got %s", report.Level)
	}
}

func TestUnixTrust_WorldWritableAncestorWithoutStickyIsUntrusted(t *testing.T) {
	root := t.TempDir()
	parent := filepath.Join(root, "shared")
	if err := os.Mkdir(parent, 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	child := filepath.Join(parent, "origin")
	if err := os.Mkdir(child, 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.Chmod(parent, 0o777); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	report := DefaultTrustVerifier().VerifyDirectory(child)
	if report.Level != TrustUntrusted {
		t.Fatalf("a replaceable ancestor must make the origin untrusted, got %s: %s", report.Level, report.Reason())
	}
}

func TestUnixTrust_WorldWritableAncestorWithStickyIsAccepted(t *testing.T) {
	root := t.TempDir()
	parent := filepath.Join(root, "sticky")
	if err := os.Mkdir(parent, 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	child := filepath.Join(parent, "origin")
	if err := os.Mkdir(child, 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.Chmod(parent, 0o777|os.ModeSticky); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	report := DefaultTrustVerifier().VerifyDirectory(child)
	if report.Level.Blocks() {
		t.Fatalf("sticky ancestor (like /tmp) must be accepted, got %s: %s", report.Level, report.Reason())
	}
}

func TestUnixTrust_FileIsNotADirectory(t *testing.T) {
	file := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(file, []byte("x"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	if report := DefaultTrustVerifier().VerifyDirectory(file); report.Level != TrustUntrusted {
		t.Fatalf("expected untrusted for a regular file, got %s", report.Level)
	}
}

func TestUnixTrust_MissingPathIsUnknownNotTrusted(t *testing.T) {
	report := DefaultTrustVerifier().VerifyDirectory(filepath.Join(t.TempDir(), "does-not-exist"))
	if report.Level != TrustUnknown || !report.Level.Blocks() {
		t.Fatalf("an unverifiable path must be unknown and blocking, got %s", report.Level)
	}
}

func TestUnixTrust_SymlinkIsResolvedBeforeEvaluation(t *testing.T) {
	root := t.TempDir()
	real := filepath.Join(root, "real")
	if err := os.Mkdir(real, 0o777); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.Chmod(real, 0o777); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	link := filepath.Join(root, "link")
	if err := os.Symlink(real, link); err != nil {
		t.Fatalf("symlink: %v", err)
	}
	if report := DefaultTrustVerifier().VerifyDirectory(link); report.Level != TrustUntrusted {
		t.Fatalf("a symlink to an unsafe directory must be untrusted, got %s", report.Level)
	}
}
