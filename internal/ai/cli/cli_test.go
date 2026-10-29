package cli_test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/arturonavax/env/internal/ai/cli"
	"github.com/arturonavax/env/internal/platform"
)

func newTestCmd(t *testing.T, args ...string) (*bytes.Buffer, *bytes.Buffer, error) {
	t.Helper()
	home := t.TempDir()
	plat := platform.NewTestInfo(platform.TestInfoParams{OS: "linux", HomeDir: home})
	cmd := cli.NewCommand(
		cli.WithPlatform(plat),
		cli.WithConfigPath(filepath.Join(home, "config.json")),
		cli.WithStateDir(filepath.Join(home, "state")),
		cli.WithWorkDir(home),
		cli.WithIsTTY(false),
	)
	var out, errOut bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errOut)
	cmd.SetArgs(args)
	err := cmd.ExecuteContext(context.Background())
	return &out, &errOut, err
}

func TestSubcommandsRunAgainstIsolatedHome(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "backends lists agy", args: []string{"backends"}, want: "agy"},
		{name: "commands lists catalog", args: []string{"commands"}, want: "COMMAND"},
		{name: "status", args: []string{"status"}, want: "Diagnostics"},
		{name: "daemon status", args: []string{"daemon", "status"}, want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, _, err := newTestCmd(t, tt.args...)
			if err != nil {
				t.Fatalf("%v failed: %v", tt.args, err)
			}
			if !strings.Contains(out.String(), tt.want) {
				t.Fatalf("output %q missing %q", out.String(), tt.want)
			}
		})
	}
}

func TestDaemonRejectsUnknownActionAndBadAutoWakeArg(t *testing.T) {
	if _, _, err := newTestCmd(t, "daemon", "bogus"); err == nil {
		t.Fatal("expected error for unknown daemon action")
	}
	if _, _, err := newTestCmd(t, "daemon", "auto-wake", "maybe"); err == nil {
		t.Fatal("expected error for invalid auto-wake argument")
	}
}

func TestOptimizeValidateBlocksDestructiveScript(t *testing.T) {
	dir := t.TempDir()
	bad := filepath.Join(dir, "bad.sh")
	good := filepath.Join(dir, "good.sh")
	if err := os.WriteFile(bad, []byte("#!/bin/sh\nmkfs.ext4 /dev/sda1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(good, []byte("#!/bin/sh\necho ok\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := newTestCmd(t, "optimize", "--validate", bad); err == nil {
		t.Fatal("expected guardrail violation")
	}
	if _, _, err := newTestCmd(t, "optimize", "--validate", good); err != nil {
		t.Fatalf("safe script rejected: %v", err)
	}
}
