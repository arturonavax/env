package tools_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/arturonavax/env/internal/domain"
	"github.com/arturonavax/env/internal/tools"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

const goodTool = `tools:
  - id: mytool
    name: "My Tool"
    executable: "mytool"
`

// TOOLS-01 regression: an invalid user overlay must fail loading, naming the offending file.
func TestOverlayFailsClosed(t *testing.T) {
	cases := map[string]string{
		"syntax error":     "tools: [unterminated",
		"unknown key":      "tools:\n  - id: x\n    name: X\n    executable: x\n    updat_cmd: [a]\n",
		"missing id":       "tools:\n  - name: X\n    executable: x\n",
		"missing name":     "tools:\n  - id: x\n    executable: x\n",
		"path executable":  "tools:\n  - id: x\n    name: X\n    executable: \"../../bin/sh\"\n",
		"unknown strategy": "tools:\n  - id: x\n    name: X\n    executable: x\n    update_strategy: magic\n",
		"strategy no cmd":  "tools:\n  - id: x\n    name: X\n    executable: x\n    update_strategy: self_update\n",
		"bad id":           "tools:\n  - id: \"Bad ID!\"\n    name: X\n    executable: x\n",
		"duplicate id":     "tools:\n  - id: x\n    name: X\n    executable: x\n  - id: x\n    name: Y\n    executable: y\n",
		"nothing to run":   "tools:\n  - id: x\n    name: X\n",
	}

	for name, content := range cases {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			writeFile(t, filepath.Join(dir, "broken.yaml"), content)

			r, err := tools.LoadRegistry(tools.WithUserToolsDir(dir), tools.WithHomeDir(t.TempDir()))
			if err == nil {
				t.Fatalf("expected error, got registry with %d tools", len(r.List()))
			}
			if r != nil {
				t.Fatal("registry must be nil when an overlay is invalid")
			}
			if !strings.Contains(err.Error(), "broken.yaml") {
				t.Fatalf("error must name the offending file: %v", err)
			}
		})
	}
}

func TestOverlayReportsEveryFailure(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "a.yaml"), "tools: [")
	writeFile(t, filepath.Join(dir, "b.yaml"), "tools:\n  - id: x\n")
	writeFile(t, filepath.Join(dir, "c.yaml"), goodTool)

	_, err := tools.LoadRegistry(tools.WithUserToolsDir(dir), tools.WithHomeDir(t.TempDir()))
	if err == nil {
		t.Fatal("expected error")
	}
	for _, f := range []string{"a.yaml", "b.yaml"} {
		if !strings.Contains(err.Error(), f) {
			t.Errorf("error should mention %s: %v", f, err)
		}
	}
	if !errors.Is(err, tools.ErrInvalidManifest) {
		t.Errorf("validation failures must wrap ErrInvalidManifest: %v", err)
	}
}

func TestValidOverlayLoadsAndOverridesEmbedded(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "ok.yaml"), goodTool)

	r, err := tools.LoadRegistry(tools.WithUserToolsDir(dir), tools.WithHomeDir(t.TempDir()))
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := r.Get("mytool"); !ok {
		t.Fatal("overlay tool not registered")
	}
	if _, ok := r.Get("ai"); !ok {
		t.Fatal("embedded tools must remain available alongside overlays")
	}
}

func TestEmptyOverlayFileIsAccepted(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "empty.yaml"), "")

	if _, err := tools.LoadRegistry(tools.WithUserToolsDir(dir), tools.WithHomeDir(t.TempDir())); err != nil {
		t.Fatalf("an empty drop-in is not an error: %v", err)
	}
}

func TestRepoOverlayOnlyReadsToolManifests(t *testing.T) {
	repo := t.TempDir()
	// Sibling manifests that are NOT tool manifests must never be parsed as such.
	writeFile(t, filepath.Join(repo, "packages", "x.yaml"), "name: not-a-tool-manifest\nlist: [")
	writeFile(t, filepath.Join(repo, "configs.yaml"), "unrelated: true\n")
	writeFile(t, filepath.Join(repo, "tools.yaml"), goodTool)
	writeFile(t, filepath.Join(repo, "tools", "extra.yaml"), "tools:\n  - id: extra\n    name: Extra\n    executable: extra\n")

	r, err := tools.LoadRegistry(tools.WithRepoToolsDir(repo), tools.WithHomeDir(t.TempDir()))
	if err != nil {
		t.Fatalf("unrelated manifests must be ignored: %v", err)
	}
	for _, id := range []string{"mytool", "extra"} {
		if _, ok := r.Get(id); !ok {
			t.Errorf("tool %q from repo overlay not loaded", id)
		}
	}
}

func TestRepoOverlayBrokenToolsYAMLFailsClosed(t *testing.T) {
	repo := t.TempDir()
	writeFile(t, filepath.Join(repo, "tools.yaml"), "tools: [")

	if _, err := tools.LoadRegistry(tools.WithRepoToolsDir(repo), tools.WithHomeDir(t.TempDir())); err == nil {
		t.Fatal("broken repo tools.yaml must fail closed")
	}
}

func TestMissingOverlayDirsAreNotErrors(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "does-not-exist")
	if _, err := tools.LoadRegistry(
		tools.WithRepoToolsDir(missing),
		tools.WithUserToolsDir(missing),
		tools.WithHomeDir(t.TempDir()),
	); err != nil {
		t.Fatalf("absent overlays are normal: %v", err)
	}
}

func TestAtomicDocumentMerge(t *testing.T) {
	dir := t.TempDir()
	// First tool is valid, second is not: nothing from this file may leak into the registry.
	writeFile(t, filepath.Join(dir, "mixed.yaml"),
		"tools:\n  - id: partial\n    name: P\n    executable: p\n  - id: bad\n")

	if _, err := tools.LoadRegistry(tools.WithUserToolsDir(dir), tools.WithHomeDir(t.TempDir())); err == nil {
		t.Fatal("expected failure")
	}
}

func TestValidateManifestAcceptsEmbeddedShapes(t *testing.T) {
	ok := []domain.ToolManifest{
		{ID: "ai", Name: "AI", Executable: "anx", FallbackBuiltin: true, UpdateStrategy: "anx_self"},
		{ID: "rtk", Name: "RTK", Executable: "rtk", UpdateStrategy: "self_update", UpdateCmd: []string{"rtk", "update"}},
		{ID: "pathy", Name: "Pathy", Path: "~/bin/pathy"},
	}
	for _, m := range ok {
		if err := tools.ValidateManifest(m); err != nil {
			t.Errorf("%s: %v", m.ID, err)
		}
	}
}
