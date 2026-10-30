package capability_test

import (
	"bytes"
	"context"
	"testing"

	"github.com/arturonavax/env/internal/capability"
	"github.com/arturonavax/env/internal/domain"
)

func TestCapabilityRegistryAndResolve(t *testing.T) {
	reg := capability.NewRegistry()

	// 1. Get existing
	def, ok := reg.Get("files.list")
	if !ok || def.Alias != "ls" {
		t.Fatalf("expected capability files.list with alias ls")
	}

	// 2. Resolve ls
	res, err := reg.Resolve("ls")
	if err != nil {
		t.Fatalf("failed resolving capability ls: %v", err)
	}
	if res.Selected == "" {
		t.Fatalf("expected resolved executable, got empty")
	}

	// 3. Explain
	prov, err := reg.Explain("files.list")
	if err != nil {
		t.Fatalf("failed explaining capability: %v", err)
	}
	if prov.Kind != domain.KindCapability {
		t.Errorf("expected kind %s, got %s", domain.KindCapability, prov.Kind)
	}
	if prov.ResourceID != "files.list" {
		t.Errorf("expected resource_id files.list, got %s", prov.ResourceID)
	}

	// 4. Execute ls on TempDir
	tempDir := t.TempDir()
	var outBuf, errBuf bytes.Buffer
	code, err := reg.Execute(context.Background(), "ls", []string{tempDir}, nil, &outBuf, &errBuf)
	if err != nil || code != 0 {
		t.Fatalf("execution failed with code %d: %v (err output: %s)", code, err, errBuf.String())
	}
}
