package config_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/arturonavax/env/internal/ai/config"
)

func TestStripJSONC(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		wantKeys map[string]string
	}{
		{
			name: "strips line and block comments and trailing commas",
			input: `{
				// Top-level line comment
				"version": "1.0.0", /* inline block comment */
				# Hash comment supported for Python aitools parity
				"default_commit_backend": "claude",
				"nested": {
					"a": "ok", // trailing comma before comment
				},
			}`,
			wantKeys: map[string]string{
				"version":                "1.0.0",
				"default_commit_backend": "claude",
			},
		},
		{
			name: "preserves URLs block tokens and escaped quotes inside string literals",
			input: `{
				"webhook": "https://api.telegram.org/bot123/sendMessage?note=/*not-a-comment*/#frag",
				"quote": "say \"hello // still inside string,}\"",
				"arr": ["x,}", "y",]
			}`,
			wantKeys: map[string]string{
				"webhook": "https://api.telegram.org/bot123/sendMessage?note=/*not-a-comment*/#frag",
				"quote":   `say "hello // still inside string,}"`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			clean := config.StripJSONC([]byte(tt.input))
			var parsed map[string]any
			if err := json.Unmarshal(clean, &parsed); err != nil {
				t.Fatalf("json.Unmarshal failed on StripJSONC output: %v\noutput:\n%s", err, string(clean))
			}

			for k, want := range tt.wantKeys {
				got, _ := parsed[k].(string)
				if got != want {
					t.Errorf("key %q = %q, want %q", k, got, want)
				}
			}
		})
	}
}

func TestLoadDeepMergeAndSave(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.json")

	overrideJSONC := `{
		// Override only commit backend and add one custom backend + command
		"default_commit_backend": "claude",
		"commit": {
			"default_backend": "claude",
			"effort": "high",
			"templates": {
				"org-custom": "[{task_id}] {diff}",
			},
		},
		"backends": {
			"my-custom": {
				"command": ["my-bin", "{prompt}"],
				"input_mode": "arg",
				"output_mode": "stdout",
			},
		},
	}`

	if err := os.WriteFile(cfgPath, []byte(overrideJSONC), 0o600); err != nil {
		t.Fatalf("writing test config: %v", err)
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		t.Fatalf("Load() unexpected error: %v", err)
	}

	if cfg.DefaultCommitBackend != "claude" {
		t.Errorf("DefaultCommitBackend = %q, want claude", cfg.DefaultCommitBackend)
	}
	if cfg.Commit.Effort != "high" {
		t.Errorf("Commit.Effort = %q, want high", cfg.Commit.Effort)
	}
	// Built-in backends must survive deep merge.
	if _, ok := cfg.Backends["agy"]; !ok {
		t.Error("expected built-in backend 'agy' to remain present after deep merge")
	}
	if _, ok := cfg.Backends["my-custom"]; !ok {
		t.Error("expected user backend 'my-custom' to be merged")
	}
	// Built-in templates and user custom template must coexist.
	tpls := cfg.EffectiveCommitTemplates()
	if _, ok := tpls["conventional"]; !ok {
		t.Error("expected built-in commit template 'conventional'")
	}
	if got := tpls["org-custom"]; got != "[{task_id}] {diff}" {
		t.Errorf("org-custom template = %q, want [{task_id}] {diff}", got)
	}
	// Built-in declarative commands must be loaded.
	for _, cmdName := range []string{"review", "explain", "refactor", "fix", "doc", "test", "changelog", "regex", "cmd"} {
		if _, ok := cfg.Commands[cmdName]; !ok {
			t.Errorf("expected built-in declarative command %q", cmdName)
		}
	}

	// SaveConfig round-trip
	savedPath := filepath.Join(tmpDir, "saved", "config.json")
	if err := config.SaveConfig(savedPath, cfg); err != nil {
		t.Fatalf("SaveConfig() error: %v", err)
	}
	reloaded, err := config.Load(savedPath)
	if err != nil {
		t.Fatalf("reloading saved config: %v", err)
	}
	if reloaded.DefaultCommitBackend != "claude" {
		t.Errorf("reloaded DefaultCommitBackend = %q, want claude", reloaded.DefaultCommitBackend)
	}

	// InitConfig creates file without overwriting unless requested
	initPath := filepath.Join(tmpDir, "init", "config.json")
	if err := config.InitConfig(initPath, false); err != nil {
		t.Fatalf("InitConfig() error: %v", err)
	}
	if _, err := os.Stat(initPath); err != nil {
		t.Fatalf("expected initialized config at %s: %v", initPath, err)
	}
}

func TestStateStoreAndAuditLogConcurrency(t *testing.T) {
	t.Parallel()

	store := config.NewStateStore(t.TempDir())

	st, err := store.Load()
	if err != nil {
		t.Fatalf("initial Load() error: %v", err)
	}

	now := time.Date(2026, 10, 29, 4, 0, 0, 0, time.UTC)
	st.RecordWakeSuccess("gemini", now, 98.5)
	st.RecordWakeFailure("claude", "timeout")

	if err := store.Save(st); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	loaded, err := store.Load()
	if err != nil {
		t.Fatalf("re-Load() error: %v", err)
	}
	if !loaded.LastWakeTime("gemini").Equal(now) {
		t.Errorf("LastWakeTime(gemini) = %v, want %v", loaded.LastWakeTime("gemini"), now)
	}
	if loaded.Targets["claude"].ConsecutiveFailures != 1 {
		t.Errorf("ConsecutiveFailures = %d, want 1", loaded.Targets["claude"].ConsecutiveFailures)
	}

	var wg sync.WaitGroup
	for i := range 12 {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			_ = store.AppendLog("INFO", "PROBE_OK", "concurrent entry", map[string]any{"worker": idx})
		}(i)
	}
	wg.Wait()

	lines, err := store.TailLog(5)
	if err != nil {
		t.Fatalf("TailLog() error: %v", err)
	}
	if len(lines) != 5 {
		t.Fatalf("len(TailLog(5)) = %d, want 5", len(lines))
	}
	for _, line := range lines {
		if !strings.Contains(line, "[PROBE_OK]") {
			t.Errorf("unexpected log line format: %q", line)
		}
	}
}

func TestStateStore_LegacyPythonFloatUnmarshal(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	store := config.NewStateStore(tmpDir)

	legacyJSON := `{
  "scheduled_wake_epoch": 1793350800.123,
  "expecting_auto_wake": true,
  "last_wake_epoch": 1793278668.9726114,
  "last_auto_wake_epoch": "1793270000",
  "targets_last_wake": {
    "gemini": 1793278668.55
  },
  "notified_resets": {
    "gemini": 1793278600.0
  }
}`
	if err := os.WriteFile(store.StateFilePath(), []byte(legacyJSON), 0o600); err != nil {
		t.Fatal(err)
	}

	st, err := store.Load()
	if err != nil {
		t.Fatalf("expected state store to decode legacy Python float timestamps without error: %v", err)
	}

	if st.LastWakeEpoch != 1793278668 {
		t.Errorf("LastWakeEpoch = %d, want 1793278668", st.LastWakeEpoch)
	}
	if st.ScheduledWakeEpoch != 1793350800 {
		t.Errorf("ScheduledWakeEpoch = %d, want 1793350800", st.ScheduledWakeEpoch)
	}
	if st.LastAutoWakeEpoch != 1793270000 {
		t.Errorf("LastAutoWakeEpoch = %d, want 1793270000", st.LastAutoWakeEpoch)
	}
	if st.TargetsLastWake["gemini"] != 1793278668 {
		t.Errorf("TargetsLastWake[gemini] = %d, want 1793278668", st.TargetsLastWake["gemini"])
	}
}
