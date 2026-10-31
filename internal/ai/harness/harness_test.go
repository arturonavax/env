package harness_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/arturonavax/env/internal/ai/config"
	"github.com/arturonavax/env/internal/ai/harness"
)

type mockTestRunner struct{}

func (m *mockTestRunner) ExecuteProcess(ctx context.Context, spec harness.ProcessSpec) (harness.ProcessResult, error) {
	switch spec.Binary {
	case "fake-arg":
		return harness.ProcessResult{
			Stdout: fmt.Sprintf("\x1b[36m```text\nARG_OK:%s\n```\x1b[0m\n", strings.Join(spec.Args, " ")),
		}, nil
	case "fake-stdin":
		var inStr string
		if spec.Stdin != nil {
			b, _ := io.ReadAll(spec.Stdin)
			inStr = string(b)
		}
		return harness.ProcessResult{
			Stdout: fmt.Sprintf("STDIN_OK:%s\n", inStr),
		}, nil
	case "fake-file":
		if len(spec.Args) >= 2 {
			data, err := os.ReadFile(spec.Args[0])
			if err != nil {
				return harness.ProcessResult{}, err
			}
			if err := os.WriteFile(spec.Args[1], []byte(fmt.Sprintf("FILE_OK:%s", string(data))), 0o644); err != nil {
				return harness.ProcessResult{}, err
			}
		}
		return harness.ProcessResult{}, nil
	case "fake-json":
		return harness.ProcessResult{
			Stdout: "{\"choices\":[{\"message\":{\"content\":\"JSON_EXTRACTED_OK\"}}]}\n",
		}, nil
	case "claude":
		return harness.ProcessResult{
			Stdout: "CLAUDE_FALLBACK_OK",
		}, nil
	}
	return harness.ProcessResult{}, fmt.Errorf("unknown binary: %s", spec.Binary)
}

func TestCleanersAndJSONPath(t *testing.T) {
	t.Parallel()

	dirty := "\x1b[32m\x1b]0;spinner title\x07```markdown\nfeat(ai): port harness adapter\n```\x1b[0m\r\n"
	cleaned := harness.StripMarkdownFences(harness.StripANSI(dirty))
	if cleaned != "feat(ai): port harness adapter" {
		t.Errorf("cleaned = %q, want %q", cleaned, "feat(ai): port harness adapter")
	}

	payload := `{"choices":[{"message":{"content":"hello from dot path"}}]}`
	got, err := harness.ExtractJSONPath([]byte(payload), "choices.0.message.content")
	if err != nil {
		t.Fatalf("ExtractJSONPath() error: %v", err)
	}
	if got != "hello from dot path" {
		t.Errorf("ExtractJSONPath() = %q, want %q", got, "hello from dot path")
	}
}

func TestHarnessProtocolsWithFakeScripts(t *testing.T) {
	t.Parallel()

	cfg, err := config.DefaultConfig()
	if err != nil {
		t.Fatalf("DefaultConfig() error: %v", err)
	}
	cfg.Backends["test-arg"] = config.BackendConfig{
		Command:       []string{"fake-arg", "-p", "{prompt}"},
		InputMode:     "arg",
		OutputMode:    "stdout",
		ModelFlag:     "--model",
		EffortFlag:    "--effort",
		CleanANSI:     true,
		CleanMarkdown: true,
	}
	cfg.Backends["test-stdin"] = config.BackendConfig{
		Command:       []string{"fake-stdin", "{prompt}"},
		InputMode:     "stdin",
		OutputMode:    "stdout",
		CleanANSI:     true,
		CleanMarkdown: true,
	}
	cfg.Backends["test-file"] = config.BackendConfig{
		Command:       []string{"fake-file", "{prompt_file}", "{output_file}"},
		InputMode:     "temp_file",
		OutputMode:    "temp_file",
		CleanANSI:     true,
		CleanMarkdown: true,
	}
	cfg.Backends["test-json"] = config.BackendConfig{
		Command:       []string{"fake-json", "{prompt}"},
		InputMode:     "arg",
		OutputMode:    "json",
		JSONKey:       "choices.0.message.content",
		CleanANSI:     true,
		CleanMarkdown: true,
	}

	reg := harness.NewRegistry(
		cfg,
		harness.WithLookPath(func(bin string) (string, error) { return bin, nil }),
		harness.WithProcessRunner(&mockTestRunner{}),
	)
	ctx := context.Background()

	t.Run("arg input with model and effort flags", func(t *testing.T) {
		t.Parallel()
		ad, err := reg.Resolve("test-arg")
		if err != nil {
			t.Fatalf("Resolve(test-arg): %v", err)
		}
		out, err := ad.Execute(ctx, harness.Request{
			Prompt: "ping",
			Model:  "gemini-flash",
			Effort: "high",
			Mode:   "agent",
		})
		if err != nil {
			t.Fatalf("Execute(test-arg): %v", err)
		}
		if !strings.Contains(out, "ARG_OK:-p ping --model gemini-flash --effort high") {
			t.Errorf("unexpected arg output: %q", out)
		}
	})

	t.Run("stdin input with chat mode framing", func(t *testing.T) {
		t.Parallel()
		ad, err := reg.Resolve("test-stdin")
		if err != nil {
			t.Fatalf("Resolve(test-stdin): %v", err)
		}
		out, err := ad.Execute(ctx, harness.Request{
			Prompt: "explain mutex",
			Mode:   "chat",
		})
		if err != nil {
			t.Fatalf("Execute(test-stdin): %v", err)
		}
		if !strings.Contains(out, "Direct Single-Turn Chat Mode") || !strings.Contains(out, "explain mutex") {
			t.Errorf("unexpected stdin output: %q", out)
		}
	})

	t.Run("temp_file input and temp_file output", func(t *testing.T) {
		t.Parallel()
		ad, err := reg.Resolve("test-file")
		if err != nil {
			t.Fatalf("Resolve(test-file): %v", err)
		}
		out, err := ad.Execute(ctx, harness.Request{
			Prompt: "file-payload",
		})
		if err != nil {
			t.Fatalf("Execute(test-file): %v", err)
		}
		if out != "FILE_OK:file-payload" {
			t.Errorf("temp_file output = %q, want FILE_OK:file-payload", out)
		}
	})

	t.Run("json dot-path output extraction", func(t *testing.T) {
		t.Parallel()
		ad, err := reg.Resolve("test-json")
		if err != nil {
			t.Fatalf("Resolve(test-json): %v", err)
		}
		out, err := ad.Execute(ctx, harness.Request{
			Prompt: "json-req",
		})
		if err != nil {
			t.Fatalf("Execute(test-json): %v", err)
		}
		if out != "JSON_EXTRACTED_OK" {
			t.Errorf("json output = %q, want JSON_EXTRACTED_OK", out)
		}
	})
}

func TestFallbackDiscovery(t *testing.T) {
	t.Parallel()

	cfg, err := config.DefaultConfig()
	if err != nil {
		t.Fatalf("DefaultConfig(): %v", err)
	}

	lookPath := func(bin string) (string, error) {
		if bin == "claude" {
			return "claude", nil
		}
		return "", errors.New("not found")
	}

	// Point claude command in cfg to "claude" while agy is missing.
	claudeSpec := cfg.Backends["claude"]
	claudeSpec.Command = []string{"claude", "-p", "{prompt}"}
	cfg.Backends["claude"] = claudeSpec

	reg := harness.NewRegistry(cfg, harness.WithLookPath(lookPath), harness.WithProcessRunner(&mockTestRunner{}))

	adapter, usedFallback, err := reg.ResolveWithFallback("agy")
	if err != nil {
		t.Fatalf("ResolveWithFallback(agy) error: %v", err)
	}
	if !usedFallback {
		t.Error("expected usedFallback = true when agy is missing")
	}
	if adapter.Name() != "claude" {
		t.Errorf("fallback adapter = %q, want claude", adapter.Name())
	}

	resp, usedName, err := reg.Query(context.Background(), "agy", harness.Request{Prompt: "hello"})
	if err != nil {
		t.Fatalf("Query() error: %v", err)
	}
	if usedName != "claude" || resp != "CLAUDE_FALLBACK_OK" {
		t.Errorf("Query() = (%q, %q), want (CLAUDE_FALLBACK_OK, claude)", resp, usedName)
	}
}
