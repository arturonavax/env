package security

import (
	"context"
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
		"USER=arthurnavah",
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
	if cmd.Path == "" && cmd.Args[0] != "sudo" {
		t.Fatalf("expected sudo command, got %v", cmd.Args)
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
