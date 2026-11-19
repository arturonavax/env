package manifest_test

import (
	"os"
	"path/filepath"
	"slices"
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
		hasTree := false
		for _, p := range pkgIDs {
			if p == "tree" {
				hasTree = true
				break
			}
		}
		if !hasTree {
			t.Fatalf("expected profile %q to include 'tree' package", prof.Name)
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

func TestManifestDependencyHygieneAndDistroMappings(t *testing.T) {
	cat, err := manifest.LoadCatalog()
	if err != nil {
		t.Fatalf("LoadCatalog() failed: %v", err)
	}

	t.Run("pure-go github_release and native packages avoid unconditional curl/zig/pnpm", func(t *testing.T) {
		noTopDeps := []string{"ghostty", "herdr", "fnm", "rtk", "opencode", "zig", "claude-code", "codex-cli", "build-essential", "pyenv", "rust"}
		for _, id := range noTopDeps {
			spec, ok := cat.GetPackageSpec(id)
			if !ok {
				t.Fatalf("expected package %q in catalog", id)
			}
			if len(spec.DependsOn) > 0 {
				t.Errorf("package %q should not declare unconditional top-level depends_on, got %v", id, spec.DependsOn)
			}
		}
	})

	t.Run("unused dead core packages are removed", func(t *testing.T) {
		removed := []string{"bison", "zstd", "clang", "fontconfig", "gnupg", "gzip"}
		for _, id := range removed {
			if _, ok := cat.GetPackageSpec(id); ok {
				t.Errorf("expected unused package %q to be removed from catalog", id)
			}
		}
	})

	t.Run("distro package mappings match OS packaging", func(t *testing.T) {
		be, ok := cat.GetPackageSpec("build-essential")
		if !ok {
			t.Fatal("missing build-essential")
		}
		if got := be.Strategies[0].Packages["dnf"]; len(got) == 0 || got[0] == "@development-tools" {
			t.Errorf("build-essential dnf should not use @development-tools, got %v", got)
		}
		if got := be.Strategies[0].Packages["zypper"]; len(got) == 0 || got[0] == "devel_basis" {
			t.Errorf("build-essential zypper should not use pattern name devel_basis without -t pattern, got %v", got)
		}

		nc, ok := cat.GetPackageSpec("ncurses-dev")
		if !ok {
			t.Fatal("missing ncurses-dev")
		}
		if got := nc.Strategies[0].Packages["apt"]; len(got) != 1 || got[0] != "libncurses-dev" {
			t.Errorf("ncurses-dev apt = %v, want [libncurses-dev]", got)
		}

		delta, ok := cat.GetPackageSpec("delta")
		if !ok {
			t.Fatal("missing delta")
		}
		var deltaApk []string
		for _, st := range delta.Strategies {
			if st.Type == "native" {
				deltaApk = st.Packages["apk"]
			}
		}
		if len(deltaApk) != 1 || deltaApk[0] != "delta" {
			t.Errorf("delta apk = %v, want [delta]", deltaApk)
		}

		ts, ok := cat.GetPackageSpec("tree-sitter-cli")
		if !ok {
			t.Fatal("missing tree-sitter-cli")
		}
		if got := ts.Strategies[0].Packages["brew"]; len(got) != 1 || got[0] != "tree-sitter-cli" {
			t.Errorf("tree-sitter-cli brew = %v, want [tree-sitter-cli]", got)
		}
		if got := ts.Strategies[0].Packages["apk"]; len(got) != 1 || got[0] != "tree-sitter-cli" {
			t.Errorf("tree-sitter-cli apk = %v, want [tree-sitter-cli]", got)
		}
		if len(ts.Strategies) < 3 || ts.Strategies[2].LanguagePkg == nil || ts.Strategies[2].LanguagePkg.Manager != "cargo" || !slices.Equal(ts.Strategies[2].DependsOn, []string{"rust"}) {
			t.Errorf("tree-sitter-cli fallback cargo strategy should declare depends_on: [rust], got %+v", ts.Strategies)
		}

		codex, ok := cat.GetPackageSpec("codex-cli")
		if !ok {
			t.Fatal("missing codex-cli")
		}
		if codex.Strategies[0].GitHubRelease == nil || codex.Strategies[0].GitHubRelease.AssetPatterns["windows/amd64"] != "codex-x86_64-pc-windows-msvc.exe.zip" {
			t.Errorf("codex-cli windows/amd64 asset pattern mismatch: %+v", codex.Strategies[0].GitHubRelease)
		}
		if len(codex.Strategies) < 2 || len(codex.Strategies[1].Packages["winget"]) == 0 || codex.Strategies[1].Packages["winget"][0] != "OpenAI.Codex" {
			t.Errorf("codex-cli should declare winget OpenAI.Codex fallback, got %+v", codex.Strategies)
		}
	})

	t.Run("profiles omit unnecessary wget and cmake", func(t *testing.T) {
		minPkgs, err := cat.ResolveProfilePackages("minimal")
		if err != nil {
			t.Fatal(err)
		}
		for _, id := range minPkgs {
			if id == "wget" {
				t.Errorf("minimal profile should not include redundant wget: %v", minPkgs)
			}
		}

		devPkgs, err := cat.ResolveProfilePackages("developer")
		if err != nil {
			t.Fatal(err)
		}
		for _, id := range devPkgs {
			if id == "cmake" || id == "pkg-config" {
				t.Errorf("developer profile should not include build-only %q: %v", id, devPkgs)
			}
		}
	})
}

func TestPackageSpecSupportsDisplayServer(t *testing.T) {
	tests := []struct {
		name           string
		displayServers []string
		hostDS         string
		want           bool
	}{
		{name: "unrestricted on x11", displayServers: nil, hostDS: "x11", want: true},
		{name: "unrestricted on wayland", displayServers: nil, hostDS: "wayland", want: true},
		{name: "unrestricted on headless", displayServers: nil, hostDS: "", want: true},
		{name: "x11 matches exact", displayServers: []string{"x11"}, hostDS: "x11", want: true},
		{name: "x11 matches case-insensitively", displayServers: []string{"x11"}, hostDS: "X11", want: true},
		{name: "x11 rejects wayland", displayServers: []string{"x11"}, hostDS: "wayland", want: false},
		{name: "x11 rejects headless", displayServers: []string{"x11"}, hostDS: "", want: false},
		{name: "all matches wayland", displayServers: []string{"all"}, hostDS: "wayland", want: true},
		{name: "all rejects headless", displayServers: []string{"all"}, hostDS: "", want: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			spec := manifest.PackageSpec{ID: "xclip", DisplayServers: tc.displayServers}
			if got := spec.SupportsDisplayServer(tc.hostDS); got != tc.want {
				t.Errorf("SupportsDisplayServer(%q) with %v = %v, want %v", tc.hostDS, tc.displayServers, got, tc.want)
			}
		})
	}
}

func TestStrategySpecDependsOnAndResolveInstallOrderWithDeps(t *testing.T) {
	dropIn := t.TempDir()
	yamlSpec := `packages:
  - id: dep-a
    name: Dep A
    category: test
    strategies:
      - type: native
        packages:
          brew: [dep-a]
  - id: dep-b
    name: Dep B
    category: test
    strategies:
      - type: native
        packages:
          apt: [dep-b]
  - id: linux-dep
    name: Linux Dep
    category: test
    os: [linux]
    strategies:
      - type: native
        packages:
          apt: [linux-dep]
  - id: skipped-tool
    name: Skipped Tool
    category: test
    os: [linux]
    depends_on: [linux-dep]
    strategies:
      - type: native
        packages:
          apt: [skipped-tool]
  - id: multi-strat
    name: Multi Strategy
    category: test
    strategies:
      - type: native
        depends_on: [dep-a]
        packages:
          brew: [multi-strat]
      - type: script
        depends_on: [dep-b]
        script:
          url: https://example.com/install.sh
          interpreter: sh
`
	if err := os.WriteFile(filepath.Join(dropIn, "strategy_deps.yaml"), []byte(yamlSpec), 0o600); err != nil {
		t.Fatal(err)
	}

	cat, err := manifest.LoadCatalog(manifest.WithUserDropInDir(dropIn))
	if err != nil {
		t.Fatalf("LoadCatalog() failed: %v", err)
	}

	spec, ok := cat.GetPackageSpec("multi-strat")
	if !ok {
		t.Fatal("expected multi-strat in catalog")
	}
	if len(spec.Strategies) != 2 {
		t.Fatalf("expected 2 strategies, got %d", len(spec.Strategies))
	}
	if !slices.Equal(spec.Strategies[0].DependsOn, []string{"dep-a"}) {
		t.Errorf("strategy[0].DependsOn = %v, want [dep-a]", spec.Strategies[0].DependsOn)
	}
	if !slices.Equal(spec.Strategies[1].DependsOn, []string{"dep-b"}) {
		t.Errorf("strategy[1].DependsOn = %v, want [dep-b]", spec.Strategies[1].DependsOn)
	}

	allowDarwin := func(p manifest.PackageSpec) bool {
		return p.SupportsOS("darwin")
	}
	depsForFirstStrategy := func(p manifest.PackageSpec) []string {
		if p.ID == "multi-strat" {
			return p.Strategies[0].DependsOn
		}
		return p.DependsOn
	}

	ordered, err := cat.ResolveInstallOrderWithDeps(
		[]string{"skipped-tool", "multi-strat"},
		allowDarwin,
		depsForFirstStrategy,
	)
	if err != nil {
		t.Fatalf("ResolveInstallOrderWithDeps failed: %v", err)
	}

	gotIDs := make([]string, 0, len(ordered))
	for _, p := range ordered {
		gotIDs = append(gotIDs, p.ID)
	}
	wantIDs := []string{"dep-a", "multi-strat"}
	if !slices.Equal(gotIDs, wantIDs) {
		t.Errorf("ResolveInstallOrderWithDeps IDs = %v, want %v (must skip disallowed package and unused strategy deps)", gotIDs, wantIDs)
	}
}

