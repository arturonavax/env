package pipeline_test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/arturonavax/env/internal/ai/config"
	"github.com/arturonavax/env/internal/ai/harness"
	"github.com/arturonavax/env/internal/ai/pipeline"
)

type captureProc struct {
	args []string
}

func (c *captureProc) ExecuteProcess(_ context.Context, spec harness.ProcessSpec) (harness.ProcessResult, error) {
	c.args = append([]string{}, spec.Args...)
	return harness.ProcessResult{Stdout: "PIPELINE_RESPONSE_OK"}, nil
}

func TestDeclarativePipelineInputTypes(t *testing.T) {
	t.Parallel()

	cfg, err := config.DefaultConfig()
	if err != nil {
		t.Fatal(err)
	}
	cfg.Commands["summarize-file"] = config.CommandConfig{
		Description:    "Summarize file",
		Backend:        "agy",
		InputType:      "file",
		PromptTemplate: "File {file}:\n{input}",
	}

	proc := &captureProc{}
	reg := harness.NewRegistry(
		cfg,
		harness.WithProcessRunner(proc),
		harness.WithLookPath(func(bin string) (string, error) { return "/usr/bin/" + bin, nil }),
	)
	r := pipeline.NewRunner(cfg, reg, nil)

	tmpFile := filepath.Join(t.TempDir(), "sample.go")
	if err := os.WriteFile(tmpFile, []byte("package sample\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var out, errOut bytes.Buffer
	resp, err := r.Execute(context.Background(), pipeline.RunOptions{
		Name:   "summarize-file",
		Args:   []string{tmpFile},
		Out:    &out,
		ErrOut: &errOut,
	})
	if err != nil {
		t.Fatalf("Execute(summarize-file) error: %v", err)
	}
	if resp != "PIPELINE_RESPONSE_OK" {
		t.Errorf("resp = %q, want PIPELINE_RESPONSE_OK", resp)
	}
	joined := strings.Join(proc.args, " ")
	if !strings.Contains(joined, "package sample") || !strings.Contains(joined, tmpFile) {
		t.Errorf("expected file path and content interpolated in prompt: %q", joined)
	}

	// Test stdin_or_args via piped stdin on built-in 'explain'
	out.Reset()
	_, err = r.Execute(context.Background(), pipeline.RunOptions{
		Name:       "explain",
		Stdin:      strings.NewReader("goroutine leak trace"),
		StdinIsTTY: false,
		Out:        &out,
		ErrOut:     &errOut,
	})
	if err != nil {
		t.Fatalf("Execute(explain) stdin error: %v", err)
	}
	if !strings.Contains(strings.Join(proc.args, " "), "goroutine leak trace") {
		t.Errorf("expected stdin content in prompt: %v", proc.args)
	}
}
