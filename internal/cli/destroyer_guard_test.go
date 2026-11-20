package cli

import "testing"

func TestDetectActiveTerminalSession(t *testing.T) {
	env := map[string]string{"TERM_PROGRAM": "ghostty", "HERDR_ENV": "1"}
	got := detectActiveTerminalSession(func(k string) string { return env[k] })
	if len(got) != 2 || got[0] != "Ghostty" || got[1] != "Herdr TUI" {
		t.Fatalf("unexpected detection: %v", got)
	}
	appEnv := map[string]string{"HC_TERMINAL_ID": "t-1", "HC_TERMINAL_SOCKET": "/tmp/hc.sock"}
	got = detectActiveTerminalSession(func(k string) string { return appEnv[k] })
	if len(got) != 1 || got[0] != "Herdr Core / App" {
		t.Fatalf("expected Herdr Core / App, got %v", got)
	}
	if got := detectActiveTerminalSession(func(string) string { return "" }); len(got) != 0 {
		t.Fatalf("expected none, got %v", got)
	}
}

func TestGuardDestroyerSession(t *testing.T) {
	t.Setenv("TMUX", "/tmp/tmux-1/default,1,0")
	t.Setenv(allowActiveTerminalEnv, "")
	if err := guardDestroyerSession(false); err == nil {
		t.Fatal("expected guard to block inside tmux")
	}
	if err := guardDestroyerSession(true); err != nil {
		t.Fatalf("dry-run must not be blocked: %v", err)
	}
	t.Setenv(allowActiveTerminalEnv, "1")
	if err := guardDestroyerSession(false); err != nil {
		t.Fatalf("override must allow: %v", err)
	}
}
