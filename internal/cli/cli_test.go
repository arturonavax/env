package cli_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/arturonavax/env/internal/cli"
)

func TestCLIVersion(t *testing.T) {
	var outBuf, errBuf bytes.Buffer
	code := cli.ExecuteCLI([]string{"version"}, &outBuf, &errBuf, "1.0.0-test", "abcdef", "2026-10-29")
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d (err: %s)", code, errBuf.String())
	}
	if !strings.Contains(outBuf.String(), "anx version 1.0.0-test") {
		t.Fatalf("expected version output, got: %s", outBuf.String())
	}
}

func TestCLIStatusJSON(t *testing.T) {
	var outBuf, errBuf bytes.Buffer
	code := cli.ExecuteCLI([]string{"status", "--format", "json"}, &outBuf, &errBuf, "1.0.0-test", "", "")
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d (err: %s)", code, errBuf.String())
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(outBuf.Bytes(), &payload); err != nil {
		t.Fatalf("expected valid JSON output, got error: %v (raw: %s)", err, outBuf.String())
	}
	if payload["os"] == nil || payload["package_count"] == nil {
		t.Fatalf("expected os and package_count in json payload")
	}
}

func TestCLIPlan(t *testing.T) {
	var outBuf, errBuf bytes.Buffer
	code := cli.ExecuteCLI([]string{"plan", "--profile", "minimal"}, &outBuf, &errBuf, "1.0.0-test", "", "")
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d (err: %s)", code, errBuf.String())
	}
	if !strings.Contains(outBuf.String(), "Execution Plan") {
		t.Fatalf("expected execution plan output, got: %s", outBuf.String())
	}
}

func TestCLIDoctor(t *testing.T) {
	var outBuf, errBuf bytes.Buffer
	code := cli.ExecuteCLI([]string{"doctor"}, &outBuf, &errBuf, "1.0.0-test", "", "")
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d (err: %s)", code, errBuf.String())
	}
	if !strings.Contains(outBuf.String(), "ANX Health Doctor") {
		t.Fatalf("expected doctor header, got: %s", outBuf.String())
	}
}

func TestCLISyncDiff(t *testing.T) {
	var outBuf, errBuf bytes.Buffer
	code := cli.ExecuteCLI([]string{"diff"}, &outBuf, &errBuf, "1.0.0-test", "", "")
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d (err: %s)", code, errBuf.String())
	}
}

func TestCLIHacksList(t *testing.T) {
	var outBuf, errBuf bytes.Buffer
	code := cli.ExecuteCLI([]string{"hacks", "list"}, &outBuf, &errBuf, "1.0.0-test", "", "")
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d (err: %s)", code, errBuf.String())
	}
	if !strings.Contains(outBuf.String(), "dell-precision-m6800") && !strings.Contains(outBuf.String(), "Registered Hacks") {
		t.Fatalf("expected hacks output, got: %s", outBuf.String())
	}
}

func TestCLIThemeList(t *testing.T) {
	var outBuf, errBuf bytes.Buffer
	code := cli.ExecuteCLI([]string{"theme", "list"}, &outBuf, &errBuf, "1.0.0-test", "", "")
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d (err: %s)", code, errBuf.String())
	}
	if !strings.Contains(outBuf.String(), "tokyo-night") {
		t.Fatalf("expected tokyo-night in theme list, got: %s", outBuf.String())
	}
}

func TestCLIConfigShow(t *testing.T) {
	var outBuf, errBuf bytes.Buffer
	code := cli.ExecuteCLI([]string{"config", "show"}, &outBuf, &errBuf, "1.0.0-test", "", "")
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d (err: %s)", code, errBuf.String())
	}
	if !strings.Contains(outBuf.String(), "config_dir") {
		t.Fatalf("expected config_dir in config show output, got: %s", outBuf.String())
	}
}

func TestCLIAiHelp(t *testing.T) {
	var outBuf, errBuf bytes.Buffer
	code := cli.ExecuteCLI([]string{"ai", "--help"}, &outBuf, &errBuf, "1.0.0-test", "", "")
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d (err: %s)", code, errBuf.String())
	}
	if !strings.Contains(outBuf.String(), "commit") || !strings.Contains(outBuf.String(), "pr") {
		t.Fatalf("expected ai subcommands in help output, got: %s", outBuf.String())
	}
}
