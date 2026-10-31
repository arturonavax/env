package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestCLIJobsCommands(t *testing.T) {
	var out, errOut bytes.Buffer
	exitCode := ExecuteCLI([]string{"jobs", "list"}, &out, &errOut, "test", "head", "2026-10-31")
	if exitCode != 0 {
		t.Fatalf("jobs list returned exit %d, err: %s", exitCode, errOut.String())
	}
	combined := out.String() + errOut.String()
	if !strings.Contains(combined, "Jobs") && !strings.Contains(combined, "jobs") {
		t.Errorf("expected jobs list output, got: %s", combined)
	}
}
