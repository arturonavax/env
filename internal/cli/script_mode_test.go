package cli_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/arturonavax/env/internal/cli"
)

func TestUniversalScriptModeAcrossCommands(t *testing.T) {
	t.Run("anx audit --script", func(t *testing.T) {
		var outBuf, errBuf bytes.Buffer
		code := cli.ExecuteCLI([]string{"audit", "--script", "--profile", "minimal"}, &outBuf, &errBuf, "1.0.0-test", "dev", "2026-10-31")
		if code != 0 {
			t.Fatalf("audit --script failed with code %d, err: %s", code, errBuf.String())
		}
		out := outBuf.String()
		if !strings.Contains(out, "#!/usr/bin/env bash") {
			t.Errorf("expected bash shebang, got:\n%s", out)
		}
		if !strings.Contains(out, "ANX Security Audit Remediation Script") {
			t.Errorf("expected security remediation plan banner, got:\n%s", out)
		}
	})

	t.Run("anx sync --script", func(t *testing.T) {
		var outBuf, errBuf bytes.Buffer
		code := cli.ExecuteCLI([]string{"sync", "--script", "terminal"}, &outBuf, &errBuf, "1.0.0-test", "dev", "2026-10-31")
		if code != 0 {
			t.Fatalf("sync --script failed with code %d, err: %s", code, errBuf.String())
		}
		out := outBuf.String()
		if !strings.Contains(out, "#!/usr/bin/env bash") {
			t.Errorf("expected bash script, got:\n%s", out)
		}
	})

	t.Run("anx cleanup --script", func(t *testing.T) {
		var outBuf, errBuf bytes.Buffer
		code := cli.ExecuteCLI([]string{"cleanup", "--script"}, &outBuf, &errBuf, "1.0.0-test", "dev", "2026-10-31")
		if code != 0 {
			t.Fatalf("cleanup --script failed with code %d, err: %s", code, errBuf.String())
		}
		out := outBuf.String()
		if !strings.Contains(out, "#!/usr/bin/env bash") {
			t.Errorf("expected bash script, got:\n%s", out)
		}
	})

	t.Run("anx optimize --script", func(t *testing.T) {
		var outBuf, errBuf bytes.Buffer
		code := cli.ExecuteCLI([]string{"optimize", "--script", "--skip-lynis"}, &outBuf, &errBuf, "1.0.0-test", "dev", "2026-10-31")
		if code != 0 {
			t.Fatalf("optimize --script failed with code %d, err: %s", code, errBuf.String())
		}
		out := outBuf.String()
		if !strings.Contains(out, "#!/usr/bin/env bash") {
			t.Errorf("expected bash script, got:\n%s", out)
		}
	})
}
