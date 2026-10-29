package manifest_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/arturonavax/env/internal/manifest"
)

func TestEmbeddedManifestsLoadAndValidate(t *testing.T) {
	cat, err := manifest.LoadCatalog()
	if err != nil {
		t.Fatalf("LoadCatalog() failed: %v", err)
	}

	if len(cat.ListPackageSpecs()) < 45 {
		t.Fatalf("expected >=45 declarative packages from embedded YAML manifests, got %d", len(cat.ListPackageSpecs()))
	}

	// Verify multi-distro package name matrix for fd
	fdSpec, ok := cat.GetPackageSpec("fd")
	if !ok {
		t.Fatal("expected 'fd' package spec in embedded catalog")
	}
	foundApt := false
	foundPacman := false
	foundWinget := false
	for _, st := range fdSpec.Strategies {
		if st.Type == "native" {
			if pkgs := st.Packages["apt"]; len(pkgs) == 1 && pkgs[0] == "fd-find" {
				foundApt = true
			}
			if pkgs := st.Packages["pacman"]; len(pkgs) == 1 && pkgs[0] == "fd" {
				foundPacman = true
			}
			if pkgs := st.Packages["winget"]; len(pkgs) == 1 && pkgs[0] == "sharkdp.fd" {
				foundWinget = true
			}
		}
	}
	if !foundApt || !foundPacman || !foundWinget {
		t.Fatalf("expected multi-PM mappings for fd (apt=%v pacman=%v winget=%v)", foundApt, foundPacman, foundWinget)
	}

	// Verify Ghostty has declarative from-source strategy
	ghostty, ok := cat.GetPackageSpec("ghostty")
	if !ok {
		t.Fatal("expected 'ghostty' in embedded catalog")
	}
	hasSource := false
	for _, st := range ghostty.Strategies {
		if st.Type == "source" && st.Source != nil && len(st.Source.Steps) > 0 {
			hasSource = true
		}
	}
	if !hasSource {
		t.Fatal("expected ghostty to declare a from-source build strategy")
	}

	// Verify Herdr has curl/script strategy
	herdr, ok := cat.GetPackageSpec("herdr")
	if !ok {
		t.Fatal("expected 'herdr' in embedded catalog")
	}
	hasScript := false
	for _, st := range herdr.Strategies {
		if st.Type == "script" && st.Script != nil && st.Script.URL != "" {
			hasScript = true
		}
	}
	if !hasScript {
		t.Fatal("expected herdr to declare a script/curl strategy")
	}

	// Verify configs and profiles
	if len(cat.ListConfigurations()) < 12 {
		t.Fatalf("expected >=12 declarative configurations, got %d", len(cat.ListConfigurations()))
	}
	if _, ok := cat.GetProfile("developer"); !ok {
		t.Fatal("expected 'developer' profile in embedded catalog")
	}

	// Referential integrity check: every package and config referenced across all profiles must exist
	for _, prof := range cat.ListProfiles() {
		pkgIDs, err := cat.ResolveProfilePackages(prof.Name)
		if err != nil {
			t.Fatalf("ResolveProfilePackages(%q) failed: %v", prof.Name, err)
		}
		if len(pkgIDs) == 0 {
			t.Fatalf("expected profile %q to resolve non-empty packages", prof.Name)
		}
		if _, err := cat.ResolveInstallOrder(pkgIDs); err != nil {
			t.Fatalf("profile %q failed DAG resolution: %v", prof.Name, err)
		}
		cfgs, err := cat.ResolveProfileConfigurations(prof.Name)
		if err != nil {
			t.Fatalf("ResolveProfileConfigurations(%q) failed: %v", prof.Name, err)
		}
		if len(cfgs) == 0 {
			t.Fatalf("expected profile %q to resolve non-empty configurations", prof.Name)
		}
	}
}

func TestUserDropInOverlayWithoutGoChanges(t *testing.T) {
	userDropInDir := t.TempDir()
	customYAML := `packages:
  - id: custom-internal-cli
    name: Custom Internal CLI
    category: custom
    description: User-declared tool without modifying Go source
    binary_name: custom-cli
    strategies:
      - type: script
        script:
          url: https://internal.example.com/install.sh
          interpreter: sh
  - id: ripgrep
    name: ripgrep-overridden
    category: cli-tools
    description: Overridden description via user drop-in
    binary_name: rg
    strategies:
      - type: native
        packages:
          apt: [ripgrep]
`
	if err := os.WriteFile(filepath.Join(userDropInDir, "99-custom.yaml"), []byte(customYAML), 0o600); err != nil {
		t.Fatalf("writing custom drop-in yaml: %v", err)
	}

	cat, err := manifest.LoadCatalog(manifest.WithUserDropInDir(userDropInDir))
	if err != nil {
		t.Fatalf("LoadCatalog with user overlay failed: %v", err)
	}

	if _, ok := cat.GetPackageSpec("custom-internal-cli"); !ok {
		t.Fatal("expected custom-internal-cli loaded from user drop-in YAML")
	}
	rg, ok := cat.GetPackageSpec("ripgrep")
	if !ok || rg.Name != "ripgrep-overridden" {
		t.Fatalf("expected ripgrep overridden by drop-in, got %+v", rg)
	}
}

func TestDAGTopologicalOrderAndCycleDetection(t *testing.T) {
	dropIn := t.TempDir()
	dagYAML := `packages:
  - id: tool-c
    name: Tool C
    category: test
    depends_on: [tool-b]
    strategies:
      - type: native
        packages: { apt: [tool-c] }
  - id: tool-b
    name: Tool B
    category: test
    depends_on: [tool-a]
    strategies:
      - type: native
        packages: { apt: [tool-b] }
  - id: tool-a
    name: Tool A
    category: test
    strategies:
      - type: native
        packages: { apt: [tool-a] }
`
	if err := os.WriteFile(filepath.Join(dropIn, "dag.yaml"), []byte(dagYAML), 0o600); err != nil {
		t.Fatal(err)
	}

	cat, err := manifest.LoadCatalog(manifest.WithUserDropInDir(dropIn))
	if err != nil {
		t.Fatal(err)
	}

	ordered, err := cat.ResolveInstallOrder([]string{"tool-c"})
	if err != nil {
		t.Fatalf("ResolveInstallOrder failed: %v", err)
	}
	if len(ordered) != 3 || ordered[0].ID != "tool-a" || ordered[1].ID != "tool-b" || ordered[2].ID != "tool-c" {
		t.Fatalf("expected topological order [tool-a, tool-b, tool-c], got %+v", ordered)
	}

	cycleDir := t.TempDir()
	cycleYAML := `packages:
  - id: cyc-1
    name: Cyc 1
    category: test
    depends_on: [cyc-2]
    strategies:
      - type: native
        packages: { apt: [cyc-1] }
  - id: cyc-2
    name: Cyc 2
    category: test
    depends_on: [cyc-1]
    strategies:
      - type: native
        packages: { apt: [cyc-2] }
`
	if err := os.WriteFile(filepath.Join(cycleDir, "cycle.yaml"), []byte(cycleYAML), 0o600); err != nil {
		t.Fatal(err)
	}
	cycleCat, err := manifest.LoadCatalog(manifest.WithUserDropInDir(cycleDir))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := cycleCat.ResolveInstallOrder([]string{"cyc-1"}); err == nil {
		t.Fatal("expected cycle error for cyc-1 <-> cyc-2, got nil")
	}
}
