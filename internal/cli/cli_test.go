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

func TestCLIToolsList(t *testing.T) {
	var outBuf, errBuf bytes.Buffer
	code := cli.ExecuteCLI([]string{"tools", "list"}, &outBuf, &errBuf, "1.0.0-test", "", "")
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d (err: %s)", code, errBuf.String())
	}
	out := outBuf.String()
	if !strings.Contains(out, "Decoupled Tools") || !strings.Contains(out, "ai") {
		t.Fatalf("expected tools list output containing 'ai', got: %s", out)
	}
}

func TestCLIToolsInfo(t *testing.T) {
	var outBuf, errBuf bytes.Buffer
	code := cli.ExecuteCLI([]string{"tools", "info", "ai"}, &outBuf, &errBuf, "1.0.0-test", "", "")
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d (err: %s)", code, errBuf.String())
	}
	out := outBuf.String()
	if !strings.Contains(out, "Tool Info: AI Productivity Suite") {
		t.Fatalf("expected tool info header, got: %s", out)
	}
}

func TestCLIToolsAISubcommands(t *testing.T) {
	// Test 'anx tools ai commit --help'
	var outBuf, errBuf bytes.Buffer
	code := cli.ExecuteCLI([]string{"tools", "ai", "commit", "--help"}, &outBuf, &errBuf, "1.0.0-test", "", "")
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d (err: %s)", code, errBuf.String())
	}
	if !strings.Contains(strings.ToLower(outBuf.String()), "conventional") {
		t.Fatalf("expected commit help, got: %s", outBuf.String())
	}

	// Test 'anx tools ai pr --help'
	outBuf.Reset()
	errBuf.Reset()
	code = cli.ExecuteCLI([]string{"tools", "ai", "pr", "--help"}, &outBuf, &errBuf, "1.0.0-test", "", "")
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d (err: %s)", code, errBuf.String())
	}
	if !strings.Contains(outBuf.String(), "Pull Request") {
		t.Fatalf("expected pr help, got: %s", outBuf.String())
	}

	// Test 'anx tools ai quota --help'
	outBuf.Reset()
	errBuf.Reset()
	code = cli.ExecuteCLI([]string{"tools", "ai", "quota", "--help"}, &outBuf, &errBuf, "1.0.0-test", "", "")
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d (err: %s)", code, errBuf.String())
	}
	if !strings.Contains(outBuf.String(), "quota") {
		t.Fatalf("expected quota help, got: %s", outBuf.String())
	}
}

func TestCLIToolsShortcuts(t *testing.T) {
	// Test shortcut: 'anx tools llm --help'
	var outBuf, errBuf bytes.Buffer
	code := cli.ExecuteCLI([]string{"tools", "llm", "--help"}, &outBuf, &errBuf, "1.0.0-test", "", "")
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d (err: %s)", code, errBuf.String())
	}

	// Test shortcut: 'anx tools commit --help'
	outBuf.Reset()
	errBuf.Reset()
	code = cli.ExecuteCLI([]string{"tools", "commit", "--help"}, &outBuf, &errBuf, "1.0.0-test", "", "")
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d (err: %s)", code, errBuf.String())
	}

	// Test shortcut: 'anx tools pr --help'
	outBuf.Reset()
	errBuf.Reset()
	code = cli.ExecuteCLI([]string{"tools", "pr", "--help"}, &outBuf, &errBuf, "1.0.0-test", "", "")
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d (err: %s)", code, errBuf.String())
	}

	// Test top-level alias: 'anx llm --help'
	outBuf.Reset()
	errBuf.Reset()
	code = cli.ExecuteCLI([]string{"llm", "--help"}, &outBuf, &errBuf, "1.0.0-test", "", "")
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d (err: %s)", code, errBuf.String())
	}
}

func TestCLICleanupDryRun(t *testing.T) {
	var outBuf, errBuf bytes.Buffer
	code := cli.ExecuteCLI([]string{"cleanup", "--dry-run"}, &outBuf, &errBuf, "1.0.0-test", "", "")
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d (err: %s)", code, errBuf.String())
	}
}

func TestCLIDocManAndMarkdown(t *testing.T) {
	tempDir := t.TempDir()

	// Test 'anx doc man --dir <tempDir>'
	var outBuf, errBuf bytes.Buffer
	code := cli.ExecuteCLI([]string{"doc", "man", "--dir", tempDir}, &outBuf, &errBuf, "1.0.0-test", "", "")
	if code != 0 {
		t.Fatalf("expected doc man exit code 0, got %d (err: %s)", code, errBuf.String())
	}
	if !strings.Contains(outBuf.String(), "Generated troff man pages") {
		t.Fatalf("expected troff man generation output, got: %s", outBuf.String())
	}

	// Test 'anx doc markdown --dir <tempDir>'
	outBuf.Reset()
	errBuf.Reset()
	code = cli.ExecuteCLI([]string{"doc", "markdown", "--dir", tempDir}, &outBuf, &errBuf, "1.0.0-test", "", "")
	if code != 0 {
		t.Fatalf("expected doc markdown exit code 0, got %d (err: %s)", code, errBuf.String())
	}
	if !strings.Contains(outBuf.String(), "Generated markdown CLI documentation") {
		t.Fatalf("expected markdown generation output, got: %s", outBuf.String())
	}
}

func TestCLICompletion(t *testing.T) {
	var outBuf, errBuf bytes.Buffer
	code := cli.ExecuteCLI([]string{"completion", "zsh"}, &outBuf, &errBuf, "1.0.0-test", "", "")
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d (err: %s)", code, errBuf.String())
	}
	if !strings.Contains(outBuf.String(), "#compdef anx") {
		t.Fatalf("expected zsh compdef in completion output")
	}
}

func TestCLIListFilters(t *testing.T) {
	var outBuf, errBuf bytes.Buffer
	code := cli.ExecuteCLI([]string{"list", "--missing"}, &outBuf, &errBuf, "1.0.0-test", "", "")
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d (err: %s)", code, errBuf.String())
	}

	outBuf.Reset()
	errBuf.Reset()
	code = cli.ExecuteCLI([]string{"list", "--deprecated"}, &outBuf, &errBuf, "1.0.0-test", "", "")
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d (err: %s)", code, errBuf.String())
	}
}

func TestCLIUpdate(t *testing.T) {
	var outBuf, errBuf bytes.Buffer
	code := cli.ExecuteCLI([]string{"update", "--fetch=false"}, &outBuf, &errBuf, "1.0.0-test", "", "")
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d (err: %s)", code, errBuf.String())
	}
	out := outBuf.String()
	if !strings.Contains(out, "ANX Update Check") {
		t.Fatalf("expected update check header, got: %s", out)
	}

	// JSON format output test
	var jsonOut, jsonErr bytes.Buffer
	code = cli.ExecuteCLI([]string{"update", "--fetch=false", "--format", "json"}, &jsonOut, &jsonErr, "1.0.0-test", "", "")
	if code != 0 {
		t.Fatalf("expected exit code 0 for json update, got %d (err: %s)", code, jsonErr.String())
	}
	var summary cli.UpdateCheckSummary
	if err := json.Unmarshal(jsonOut.Bytes(), &summary); err != nil {
		t.Fatalf("expected valid JSON UpdateCheckSummary, got error %v (output: %s)", err, jsonOut.String())
	}
	// Verify that ConfigDrifts matches the number of DriftedFiles
	if summary.ConfigDrifts != len(summary.DriftedFiles) {
		t.Errorf("expected ConfigDrifts (%d) to match DriftedFiles count (%d)", summary.ConfigDrifts, len(summary.DriftedFiles))
	}
}

func TestCLIUpgradeDryRun(t *testing.T) {
	var outBuf, errBuf bytes.Buffer
	code := cli.ExecuteCLI([]string{"upgrade", "--dry-run", "--skip-pull"}, &outBuf, &errBuf, "1.0.0-test", "", "")
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d (err: %s)", code, errBuf.String())
	}
	out := outBuf.String()
	if !strings.Contains(out, "ANX Environment Upgrade") {
		t.Fatalf("expected upgrade header, got: %s", out)
	}
}

func TestCLICaptureDryRun(t *testing.T) {
	var outBuf, errBuf bytes.Buffer
	code := cli.ExecuteCLI([]string{"capture", "--all", "--dry-run"}, &outBuf, &errBuf, "1.0.0-test", "", "")
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d (err: %s)", code, errBuf.String())
	}
	out := outBuf.String()
	if !strings.Contains(out, "Processed") && !strings.Contains(out, "captured") {
		t.Fatalf("expected capture output, got: %s", out)
	}
}

func TestCLIAlias(t *testing.T) {
	var outBuf, errBuf bytes.Buffer
	// Validation error for empty or invalid alias
	code := cli.ExecuteCLI([]string{"alias", "invalid name!"}, &outBuf, &errBuf, "1.0.0-test", "", "")
	if code == 0 {
		t.Fatalf("expected non-zero exit code for invalid alias name")
	}
}

func TestCLIIdempotence(t *testing.T) {
	// 1. Verify plan idempotence: running plan consecutive times produces consistent actions
	var outBuf1, errBuf1 bytes.Buffer
	code1 := cli.ExecuteCLI([]string{"plan", "--profile", "minimal"}, &outBuf1, &errBuf1, "1.0.0-test", "", "")
	if code1 != 0 {
		t.Fatalf("first plan run failed: %s", errBuf1.String())
	}

	var outBuf2, errBuf2 bytes.Buffer
	code2 := cli.ExecuteCLI([]string{"plan", "--profile", "minimal"}, &outBuf2, &errBuf2, "1.0.0-test", "", "")
	if code2 != 0 {
		t.Fatalf("second plan run failed: %s", errBuf2.String())
	}

	stripPlanID := func(s string) string {
		lines := strings.Split(s, "\n")
		filtered := make([]string, 0, len(lines))
		for _, l := range lines {
			if !strings.Contains(l, "Execution Plan:") && !strings.Contains(l, "Plan ID:") {
				filtered = append(filtered, l)
			}
		}
		return strings.Join(filtered, "\n")
	}

	if stripPlanID(outBuf1.String()) != stripPlanID(outBuf2.String()) {
		t.Fatalf("plan output changed between runs (non-idempotent plan generation)")
	}

	// 2. Verify cleanup dry-run idempotence
	var outBuf3, errBuf3 bytes.Buffer
	code3 := cli.ExecuteCLI([]string{"cleanup", "--dry-run"}, &outBuf3, &errBuf3, "1.0.0-test", "", "")
	if code3 != 0 {
		t.Fatalf("first cleanup run failed: %s", errBuf3.String())
	}

	var outBuf4, errBuf4 bytes.Buffer
	code4 := cli.ExecuteCLI([]string{"cleanup", "--dry-run"}, &outBuf4, &errBuf4, "1.0.0-test", "", "")
	if code4 != 0 {
		t.Fatalf("second cleanup run failed: %s", errBuf4.String())
	}

	if outBuf3.String() != outBuf4.String() {
		t.Fatalf("cleanup output changed between consecutive dry runs")
	}
}

func TestCLIExplain(t *testing.T) {
	// Text format
	var outBuf, errBuf bytes.Buffer
	code := cli.ExecuteCLI([]string{"explain", "ripgrep"}, &outBuf, &errBuf, "1.0.0-test", "", "")
	if code != 0 {
		t.Fatalf("expected exit code 0 for explain, got %d (err: %s)", code, errBuf.String())
	}
	out := outBuf.String()
	if !strings.Contains(out, "Resource Provenance: ripgrep") {
		t.Fatalf("expected provenance header, got: %s", out)
	}
	if !strings.Contains(out, "Kind:           package") {
		t.Fatalf("expected kind package, got: %s", out)
	}

	// JSON format
	var jsonOut, jsonErr bytes.Buffer
	code = cli.ExecuteCLI([]string{"explain", "ripgrep", "--format", "json"}, &jsonOut, &jsonErr, "1.0.0-test", "", "")
	if code != 0 {
		t.Fatalf("expected exit code 0 for json explain, got %d (err: %s)", code, jsonErr.String())
	}
	var prov map[string]interface{}
	if err := json.Unmarshal(jsonOut.Bytes(), &prov); err != nil {
		t.Fatalf("expected valid JSON provenance, got error %v (output: %s)", err, jsonOut.String())
	}
	if prov["resource_id"] != "ripgrep" || prov["kind"] != "package" {
		t.Fatalf("unexpected json fields: %v", prov)
	}
}
