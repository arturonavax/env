package security

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestSudoCleanEnv(t *testing.T) {
	env := []string{
		"PATH=/usr/bin:/bin",
		"SUDO_ASKPASS=/usr/lib/ssh/gnome-ssh-askpass",
		"SSH_ASKPASS=/usr/bin/ksshaskpass",
		"SSH_ASKPASS_REQUIRE=force",
		"USER=testuser",
	}

	cleaned := SudoCleanEnv(env)
	for _, kv := range cleaned {
		if strings.HasPrefix(kv, "SUDO_ASKPASS=/") ||
			strings.HasPrefix(kv, "SSH_ASKPASS=/") ||
			strings.HasPrefix(kv, "SSH_ASKPASS_REQUIRE=") {
			t.Fatalf("expected askpass variables to be stripped, got %s", kv)
		}
	}

	hasEmptySudoAskpass := false
	hasEmptySSHAskpass := false
	for _, kv := range cleaned {
		if kv == "SUDO_ASKPASS=" {
			hasEmptySudoAskpass = true
		}
		if kv == "SSH_ASKPASS=" {
			hasEmptySSHAskpass = true
		}
	}
	if !hasEmptySudoAskpass || !hasEmptySSHAskpass {
		t.Fatalf("expected empty SUDO_ASKPASS= and SSH_ASKPASS= to be explicitly present")
	}
}

func TestNewSudoAuthCmd(t *testing.T) {
	cmd := NewSudoAuthCmd()
	if cmd == nil {
		t.Fatal("expected non-nil cmd")
	}
	joined := strings.Join(cmd.Args, " ")
	if !strings.Contains(joined, "sudo -p") || !strings.Contains(joined, "sudo -k") || !strings.Contains(joined, "repair_macos_sudo_pam_if_needed") {
		t.Fatalf("expected sudo auth with PAM self-heal and password-retry fallback (sudo -k), got %v", cmd.Args)
	}
	// Verify askpass stripped
	for _, kv := range cmd.Env {
		if strings.HasPrefix(kv, "SSH_ASKPASS_REQUIRE=") {
			t.Fatalf("found forbidden askpass env: %s", kv)
		}
	}
}

func TestStartSudoKeepaliveLifecycle(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stop := StartSudoKeepalive(ctx)
	if stop == nil {
		t.Fatal("expected non-nil stop func")
	}
	// Calling stop multiple times must be safe and idempotent
	stop()
	stop()

	// Another instance cancelled via context
	ctx2, cancel2 := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel2()
	stop2 := StartSudoKeepalive(ctx2)
	<-ctx2.Done()
	stop2()
}

func TestBrokenSudoLocalModulesDetectsMissingPamReattach(t *testing.T) {
	dir := t.TempDir()
	present := dir + "/pam_present.so"
	if err := os.WriteFile(present, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	content := "# sudo_local\nauth       optional       " + dir + "/pam_reattach.so\nauth       optional       " + present + "\nauth       sufficient     pam_tid.so\n"
	missing := brokenSudoLocalModules(content)
	if len(missing) != 1 || missing[0] != dir+"/pam_reattach.so" {
		t.Fatalf("expected only missing pam_reattach.so, got %v", missing)
	}
}

func TestSudoPAMRecoveryHint(t *testing.T) {
	dir := t.TempDir()
	prev := SudoLocalPath
	SudoLocalPath = dir + "/sudo_local"
	t.Cleanup(func() { SudoLocalPath = prev })
	if err := os.WriteFile(SudoLocalPath, []byte("auth optional "+dir+"/pam_reattach.so\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	hint := SudoPAMRecoveryHint()
	if !strings.Contains(hint, "pam_reattach.so") || !strings.Contains(hint, "pam_permit.so.2") {
		t.Fatalf("expected actionable hint, got %q", hint)
	}
}

func TestEnsureSudoPAMHealthyAndSanitizeUnbricksInterruptedDestroyer(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX PAM test skipped on Windows")
	}

	dir := t.TempDir()
	sudoLocal := filepath.Join(dir, "sudo_local")
	sysPermit := filepath.Join(dir, "pam_permit.so.2")
	if err := os.WriteFile(sysPermit, []byte("apple-pam-permit"), 0o644); err != nil {
		t.Fatal(err)
	}

	prevPath := SudoLocalPath
	prevFallbacks := SystemPAMFallbackModules
	SudoLocalPath = sudoLocal
	SystemPAMFallbackModules = []string{sysPermit}
	t.Cleanup(func() {
		SudoLocalPath = prevPath
		SystemPAMFallbackModules = prevFallbacks
	})

	// Simulate interrupted Destroyer: sudo_local points to a deleted pam_reattach.so (or dangling symlink).
	missingReattach := filepath.Join(dir, "opt", "homebrew", "lib", "pam", "pam_reattach.so")
	if err := os.MkdirAll(filepath.Dir(missingReattach), 0o755); err != nil {
		t.Fatal(err)
	}
	// Create a dangling symlink to ensure EnsureSudoPAMHealthy replaces broken symlinks too.
	if err := os.Symlink(filepath.Join(dir, "nonexistent-cellar", "pam_reattach.so"), missingReattach); err != nil {
		t.Fatal(err)
	}

	initialSudoLocal := "# sudo_local: managed by anx\nauth       optional       " + missingReattach + "\nauth       sufficient     pam_tid.so\n"
	if err := os.WriteFile(sudoLocal, []byte(initialSudoLocal), 0o644); err != nil {
		t.Fatal(err)
	}

	// Stub sudo binary that emulates macOS OpenPAM: fails with "sudo: unable to initialize PAM: No such file or directory"
	// if any absolute .so path in sudo_local does not exist on disk.
	binDir := t.TempDir()
	sudoStub := `#!/bin/sh
if [ ! -f "` + missingReattach + `" ] && grep -q "pam_reattach.so" "` + sudoLocal + `"; then
  echo "sudo: unable to initialize PAM: No such file or directory" >&2
  exit 1
fi
if [ "$1" = "-n" ] && [ "$2" = "install" ]; then
  cp "$5" "$6"
  exit 0
fi
exit 0
`
	if err := os.WriteFile(filepath.Join(binDir, "sudo"), []byte(sudoStub), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir+":"+os.Getenv("PATH"))

	// 1. EnsureSudoPAMHealthy must repair the missing path in user-space without calling sudo.
	repaired, err := EnsureSudoPAMHealthy()
	if err != nil {
		t.Fatalf("EnsureSudoPAMHealthy failed: %v", err)
	}
	if len(repaired) != 1 || repaired[0] != missingReattach {
		t.Fatalf("expected repaired=[%s], got %v", missingReattach, repaired)
	}
	if _, err := os.Stat(missingReattach); err != nil {
		t.Fatalf("expected shimmed %s to exist on disk: %v", missingReattach, err)
	}
	if !isTemporaryPAMShim(missingReattach) {
		t.Fatalf("expected %s to be recognized as temporary PAM shim", missingReattach)
	}

	// 2. SanitizeSudoLocalPAM must rewrite sudo_local without the shimmed line AND delete the temporary shim symlink.
	changed, err := SanitizeSudoLocalPAM(context.Background())
	if err != nil || !changed {
		t.Fatalf("expected SanitizeSudoLocalPAM to rewrite sudo_local, got changed=%v err=%v", changed, err)
	}
	updatedRaw, err := os.ReadFile(sudoLocal)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(updatedRaw), "pam_reattach.so") {
		t.Fatalf("expected pam_reattach.so stripped from sudo_local, got:\n%s", string(updatedRaw))
	}
	if !strings.Contains(string(updatedRaw), "pam_tid.so") {
		t.Fatalf("expected pam_tid.so preserved in sudo_local, got:\n%s", string(updatedRaw))
	}
	if _, err := os.Lstat(missingReattach); !os.IsNotExist(err) {
		t.Fatalf("expected temporary shim %s to be removed after sudo_local rewrite, lstat err=%v", missingReattach, err)
	}

	// 3. PrepareSudoLocalForPurge must strip external pam_reattach.so even when the real file currently exists on disk.
	if err := os.WriteFile(missingReattach, []byte("real-binary"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(sudoLocal, []byte(initialSudoLocal), 0o644); err != nil {
		t.Fatal(err)
	}
	if !SudoLocalReferencesModule("pam_reattach.so") {
		t.Fatal("expected SudoLocalReferencesModule(pam_reattach.so) = true before purge preparation")
	}
	purged, err := PrepareSudoLocalForPurge(context.Background())
	if err != nil || !purged {
		t.Fatalf("expected PrepareSudoLocalForPurge to strip external module before package uninstall, got purged=%v err=%v", purged, err)
	}
	if SudoLocalReferencesModule("pam_reattach.so") {
		t.Fatal("expected SudoLocalReferencesModule(pam_reattach.so) = false after PrepareSudoLocalForPurge")
	}
}
