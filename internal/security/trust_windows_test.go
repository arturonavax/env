//go:build windows

package security

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// These tests exercise the real NTFS ACL verifier. They shell out to icacls (present on every
// supported Windows) to build directories with known DACLs, and never rely on the ACLs of the
// checkout or of %TEMP%.

func icacls(t *testing.T, args ...string) {
	t.Helper()
	if out, err := exec.Command("icacls", args...).CombinedOutput(); err != nil {
		t.Fatalf("icacls %v: %v\n%s", args, err, out)
	}
}

// lockedDir creates a directory whose inheritance is removed and whose only ACEs belong to the
// current user, Administrators and SYSTEM.
func lockedDir(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "origin")
	if err := os.Mkdir(dir, 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	user := os.Getenv("USERNAME")
	icacls(t, dir, "/inheritance:r")
	icacls(t, dir, "/grant:r", user+":(OI)(CI)F", "*S-1-5-32-544:(OI)(CI)F", "*S-1-5-18:(OI)(CI)F")
	return dir
}

func TestWindowsTrust_LockedDirectoryIsNotBlocked(t *testing.T) {
	report := DefaultTrustVerifier().VerifyDirectory(lockedDir(t))
	if report.Level.Blocks() {
		t.Fatalf("a directory restricted to owner/admins/SYSTEM must not block, got %s: %s", report.Level, report.Reason())
	}
}

func TestWindowsTrust_EveryoneWriteIsUntrusted(t *testing.T) {
	dir := lockedDir(t)
	icacls(t, dir, "/grant", "*S-1-1-0:(OI)(CI)M")
	report := DefaultTrustVerifier().VerifyDirectory(dir)
	if report.Level != TrustUntrusted {
		t.Fatalf("Everyone:Modify must be untrusted, got %s: %s", report.Level, report.Reason())
	}
}

func TestWindowsTrust_EveryoneReadOnlyDoesNotBlock(t *testing.T) {
	dir := lockedDir(t)
	icacls(t, dir, "/grant", "*S-1-1-0:(OI)(CI)R")
	report := DefaultTrustVerifier().VerifyDirectory(dir)
	if report.Level.Blocks() {
		t.Fatalf("read-only access for Everyone must not block, got %s: %s", report.Level, report.Reason())
	}
}

func TestWindowsTrust_AuthenticatedUsersWriteIsDegradedNotBlocking(t *testing.T) {
	dir := lockedDir(t)
	icacls(t, dir, "/grant", "*S-1-5-11:(OI)(CI)M")
	report := DefaultTrustVerifier().VerifyDirectory(dir)
	if report.Level != TrustDegraded {
		t.Fatalf("Authenticated Users:Modify must be degraded, got %s: %s", report.Level, report.Reason())
	}
}

func TestWindowsTrust_DenyACEMasksBroadAllow(t *testing.T) {
	dir := lockedDir(t)
	icacls(t, dir, "/grant", "*S-1-1-0:(OI)(CI)M")
	icacls(t, dir, "/deny", "*S-1-1-0:(OI)(CI)(W,D,DC,AD,WD)")
	// A deny ACE for Everyone also blocks the test's own TempDir cleanup, so lift it first.
	t.Cleanup(func() { _ = exec.Command("icacls", dir, "/remove:d", "*S-1-1-0").Run() })
	report := DefaultTrustVerifier().VerifyDirectory(dir)
	if report.Level == TrustUntrusted {
		t.Fatalf("an explicit deny for the same SID must neutralise the allow, got %s: %s", report.Level, report.Reason())
	}
}

func TestWindowsTrust_MissingPathIsUnknownNotTrusted(t *testing.T) {
	report := DefaultTrustVerifier().VerifyDirectory(filepath.Join(t.TempDir(), "does-not-exist"))
	if report.Level != TrustUnknown || !report.Level.Blocks() {
		t.Fatalf("an unverifiable path must be unknown and blocking, got %s", report.Level)
	}
}

func TestWindowsTrust_FileIsNotADirectory(t *testing.T) {
	file := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(file, []byte("x"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	if report := DefaultTrustVerifier().VerifyDirectory(file); report.Level != TrustUntrusted {
		t.Fatalf("expected untrusted for a regular file, got %s", report.Level)
	}
}
