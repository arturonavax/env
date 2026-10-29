package catalog_test

import (
	"testing"

	"github.com/arturonavax/env/internal/catalog"
)

func TestCatalogRegistry(t *testing.T) {
	reg := catalog.NewDefaultRegistry()

	packages := reg.ListPackages()
	if len(packages) == 0 {
		t.Fatalf("expected packages in catalog")
	}

	rg, ok := reg.GetPackage("ripgrep")
	if !ok {
		t.Fatalf("expected ripgrep package in catalog")
	}
	if rg.BinaryName != "rg" {
		t.Fatalf("expected binary name 'rg', got '%s'", rg.BinaryName)
	}

	configs := reg.ListConfigurations()
	if len(configs) == 0 {
		t.Fatalf("expected configurations in catalog")
	}

	// Test profile extends resolution
	workstationPkgs, err := reg.ResolveProfilePackages("workstation")
	if err != nil {
		t.Fatal(err)
	}
	if len(workstationPkgs) < 10 {
		t.Fatalf("expected at least 10 packages in workstation profile, got %d", len(workstationPkgs))
	}
}
