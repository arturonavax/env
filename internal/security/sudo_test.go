package security

import (
	"context"
	"os"
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
	if !strings.Contains(joined, "sudo -p") || !strings.Contains(joined, "sudo -k") {
		t.Fatalf("expected sudo auth with password-retry fallback (sudo -k), got %v", cmd.Args)
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

func TestSudoPAMRecoveryHintOnDarwinOnly(t *testing.T) {
	dir := t.TempDir()
	prev := SudoLocalPath
	SudoLocalPath = dir + "/sudo_local"
	t.Cleanup(func() { SudoLocalPath = prev })
	if err := os.WriteFile(SudoLocalPath, []byte("auth optional "+dir+"/pam_reattach.so\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	hint := SudoPAMRecoveryHint()
	if runtime.GOOS == "darwin" {
		if !strings.Contains(hint, "pam_reattach.so") || !strings.Contains(hint, "Password login") {
			t.Fatalf("expected actionable hint on darwin, got %q", hint)
		}
	} else if hint != "" {
		t.Fatalf("expected empty hint off darwin, got %q", hint)
	}
}
