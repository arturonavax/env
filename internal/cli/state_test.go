package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestCLIStateCommands(t *testing.T) {
	cases := []struct {
		name     string
		args     []string
		wantExit int
		wantText string
	}{
		{
			name:     "state status",
			args:     []string{"state", "status"},
			wantExit: 0,
			wantText: "ANX State Engine Status",
		},
		{
			name:     "state migrate plan",
			args:     []string{"state", "migrate", "--plan"},
			wantExit: 0,
			wantText: "state",
		},
		{
			name:     "state backup dry run",
			args:     []string{"state", "backup", "--dry-run"},
			wantExit: 0,
			wantText: "Would backup database",
		},
		{
			name:     "state verify",
			args:     []string{"state", "verify"},
			wantExit: 0,
			wantText: "Database integrity verification: PASS",
		},
		{
			name:     "state export",
			args:     []string{"state", "export"},
			wantExit: 0,
			wantText: `"schema_version"`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var out, errOut bytes.Buffer
			exitCode := ExecuteCLI(tc.args, &out, &errOut, "test", "head", "2026-10-31")
			if exitCode != tc.wantExit {
				t.Fatalf("args %v returned exit %d (want %d), errOut: %s", tc.args, exitCode, tc.wantExit, errOut.String())
			}
			combined := out.String() + errOut.String()
			if !strings.Contains(combined, tc.wantText) {
				t.Errorf("args %v output missing %q, got: %s", tc.args, tc.wantText, combined)
			}
		})
	}
}
