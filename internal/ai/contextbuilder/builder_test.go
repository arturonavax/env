package contextbuilder_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/arturonavax/env/internal/ai/contextbuilder"
)

func TestBuildExclusionsBinaryAndByteCap(t *testing.T) {
	t.Parallel()

	root := t.TempDir()

	// Valid source file
	srcDir := filepath.Join(root, "src")
	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		t.Fatal(err)
	}
	mainGo := filepath.Join(srcDir, "main.go")
	if err := os.WriteFile(mainGo, []byte("package main\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Excluded directories (node_modules, .git)
	for _, exDir := range []string{"node_modules/pkg", ".git/objects", "vendor/lib"} {
		fullDir := filepath.Join(root, exDir)
		if err := os.MkdirAll(fullDir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(fullDir, "secret_noise.js"), []byte("NOISE_TOKEN"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	// Excluded lockfile and minified bundle
	if err := os.WriteFile(filepath.Join(root, "package-lock.json"), []byte(`{"lock":true}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "app.min.js"), []byte(`minified()`), 0o644); err != nil {
		t.Fatal(err)
	}

	// Binary file with NUL byte inside first 8KB
	binPath := filepath.Join(srcDir, "compiled.bin")
	if err := os.WriteFile(binPath, []byte{'E', 'L', 'F', 0x00, 0x01, 0x02}, 0o644); err != nil {
		t.Fatal(err)
	}

	ctxOut, err := contextbuilder.Build(contextbuilder.Options{
		Files: []string{binPath},
		Dirs:  []string{root},
	})
	if err != nil {
		t.Fatalf("Build() error: %v", err)
	}

	if !strings.Contains(ctxOut, "package main") {
		t.Errorf("expected main.go content in output:\n%s", ctxOut)
	}
	if !strings.Contains(ctxOut, "(BINARY)") {
		t.Errorf("expected explicit binary omission notice:\n%s", ctxOut)
	}
	if strings.Contains(ctxOut, "NOISE_TOKEN") {
		t.Errorf("excluded directory file leaked into context:\n%s", ctxOut)
	}
	if strings.Contains(ctxOut, `"lock":true`) || strings.Contains(ctxOut, "minified()") {
		t.Errorf("excluded lockfile or minified asset leaked into context:\n%s", ctxOut)
	}

	// Test total byte cap truncation
	largeFile := filepath.Join(root, "large.txt")
	if err := os.WriteFile(largeFile, []byte(strings.Repeat("A", 2048)), 0o644); err != nil {
		t.Fatal(err)
	}

	cappedOut, err := contextbuilder.Build(contextbuilder.Options{
		Files:         []string{largeFile},
		MaxTotalBytes: 256,
		MaxFileBytes:  1024,
	})
	if err != nil {
		t.Fatalf("Build() capped error: %v", err)
	}
	if !strings.Contains(cappedOut, "Context limit of 256 bytes reached") {
		t.Errorf("expected byte cap truncation notice in:\n%s", cappedOut)
	}
}
