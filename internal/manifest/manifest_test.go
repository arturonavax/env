package manifest_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/arturonavax/env/internal/manifest"
	"github.com/arturonavax/env/internal/security"
	"gopkg.in/yaml.v3"
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

func TestMacOSOSConfigAssetsAndManifests(t *testing.T) {
	cat, err := manifest.LoadCatalog()
	if err != nil {
		t.Fatalf("LoadCatalog() failed: %v", err)
	}

	t.Run("macOS packages and configs are declared for darwin osconfig", func(t *testing.T) {
		for _, id := range []string{"amethyst", "karabiner-elements", "pam-reattach"} {
			spec, ok := cat.GetPackageSpec(id)
			if !ok {
				t.Fatalf("missing macOS package %q in catalog", id)
			}
			if !slices.Equal(spec.OS, []string{"darwin"}) {
				t.Errorf("%q OS = %v, want [darwin]", id, spec.OS)
			}
		}

		wsPkgs, err := cat.ResolveProfilePackages("workstation")
		if err != nil {
			t.Fatal(err)
		}
		for _, id := range []string{"amethyst", "karabiner-elements", "pam-reattach"} {
			if !slices.Contains(wsPkgs, id) {
				t.Errorf("workstation profile should include %q for darwin hosts", id)
			}
		}

		osConfigs := cat.ListConfigurationsByGroup("osconfig")
		found := map[string]bool{}
		for _, cfg := range osConfigs {
			found[cfg.ID] = true
			if strings.HasPrefix(cfg.ID, "macos-") || cfg.ID == "amethyst-config" || cfg.ID == "karabiner-config" {
				if !slices.Equal(cfg.SupportedOS, []string{"darwin"}) {
					t.Errorf("config %q SupportedOS = %v, want [darwin]", cfg.ID, cfg.SupportedOS)
				}
			}
		}
		for _, id := range []string{"amethyst-config", "karabiner-config", "macos-osconfig-script", "macos-shortcuts"} {
			if !found[id] {
				t.Errorf("osconfig group missing darwin configuration %q", id)
			}
		}
	})

	t.Run("amethyst config avoids keybind conflicts and uses object floating entries", func(t *testing.T) {
		raw, err := os.ReadFile(filepath.Join("..", "..", "files", "amethyst", ".amethyst.yml"))
		if err != nil {
			t.Fatal(err)
		}
		var doc struct {
			ToggleFloat struct {
				Mod string `yaml:"mod"`
				Key string `yaml:"key"`
			} `yaml:"toggle-float"`
			ToggleTiling struct {
				Mod string `yaml:"mod"`
				Key string `yaml:"key"`
			} `yaml:"toggle-tiling"`
			Floating []struct {
				ID           string   `yaml:"id"`
				WindowTitles []string `yaml:"window-titles"`
			} `yaml:"floating"`
		}
		if err := yaml.Unmarshal(raw, &doc); err != nil {
			t.Fatalf("invalid amethyst YAML or legacy floating format: %v", err)
		}
		if doc.ToggleFloat.Mod == doc.ToggleTiling.Mod && doc.ToggleFloat.Key == doc.ToggleTiling.Key {
			t.Errorf("duplicate amethyst shortcut on toggle-float and toggle-tiling: %+v", doc.ToggleFloat)
		}
		if len(doc.Floating) == 0 || doc.Floating[0].ID != "com.microsoft.VSCode" {
			t.Errorf("expected modern floating object entries starting with com.microsoft.VSCode, got %+v", doc.Floating)
		}
	})

	t.Run("karabiner config uses native software_function, new Teams bundle, and v15 keyboard schema", func(t *testing.T) {
		raw, err := os.ReadFile(filepath.Join("..", "..", "files", "karabiner", "karabiner.json"))
		if err != nil {
			t.Fatal(err)
		}
		text := string(raw)
		if strings.Contains(text, "Microsoft Teams (work or school)") || strings.Contains(text, "osascript") {
			t.Errorf("karabiner.json still contains legacy Teams app name or osascript shell invocation")
		}

		var doc struct {
			Profiles []struct {
				VirtualHIDKeyboard struct {
					KeyboardTypeV2 string `json:"keyboard_type_v2"`
				} `json:"virtual_hid_keyboard"`
				ComplexModifications struct {
					Rules []struct {
						Manipulators []struct {
							To []struct {
								KeyCode          string `json:"key_code"`
								SoftwareFunction *struct {
									OpenApplication *struct {
										BundleIdentifier string `json:"bundle_identifier"`
									} `json:"open_application"`
								} `json:"software_function"`
							} `json:"to"`
						} `json:"manipulators"`
					} `json:"rules"`
				} `json:"complex_modifications"`
			} `json:"profiles"`
		}
		if err := json.Unmarshal(raw, &doc); err != nil {
			t.Fatalf("karabiner.json unmarshal failed: %v", err)
		}
		if len(doc.Profiles) == 0 || doc.Profiles[0].VirtualHIDKeyboard.KeyboardTypeV2 != "ansi" {
			t.Fatalf("expected virtual_hid_keyboard.keyboard_type_v2 = ansi, got %+v", doc.Profiles)
		}
		bundles := map[string]bool{}
		for _, r := range doc.Profiles[0].ComplexModifications.Rules {
			for _, m := range r.Manipulators {
				for _, to := range m.To {
					if to.SoftwareFunction != nil && to.SoftwareFunction.OpenApplication != nil {
						bundles[to.SoftwareFunction.OpenApplication.BundleIdentifier] = true
					}
				}
			}
		}
		for _, wantBundle := range []string{"com.mitchellh.ghostty", "com.microsoft.teams2", "com.microsoft.VSCode", "com.apple.Safari"} {
			if !bundles[wantBundle] {
				t.Errorf("expected karabiner software_function open_application bundle %q, got %v", wantBundle, bundles)
			}
		}
	})

	t.Run("macos_osconfig.sh passes security guardrails and targets max macOS 27 with sudo_local", func(t *testing.T) {
		raw, err := os.ReadFile(filepath.Join("..", "..", "scripts", "macos_osconfig.sh"))
		if err != nil {
			t.Fatalf("reading scripts/macos_osconfig.sh: %v", err)
		}
		content := string(raw)
		if violations := security.ValidateScriptGuardrails(content); len(violations) > 0 {
			t.Errorf("scripts/macos_osconfig.sh triggered guardrails: %+v", violations)
		}
		for _, required := range []string{
			"/etc/pam.d/sudo_local",
			"pam_reattach.so",
			"pam_tid.so",
			"com.apple.WindowManager",
			"EnableTiledWindowMargins",
			"MACOS_MAX_VERSION=27",
			"DSDontWriteNetworkStores",
			`-string "PfHm"`,
			`defaults_curhost_write com.apple.controlcenter "NSStatusItem Visible Siri" -bool false`,
			"com.apple.Siri StatusMenuVisible -bool false",
			"NSScrollAnimationEnabled -bool false",
			"com.mitchellh.ghostty NSAppSleepDisabled -bool true",
			`com.apple.dock mineffect -string "scale"`,
			"pmset -a lowpowermode 0",
			"--power",
			"--reduce-ui-fx",
			"--reset-launchservices",
			"reduceTransparency",
			"reduceMotion",
			"AppleSymbolicHotKeys -dict-add",
			"defaults read com.apple.symbolichotkeys",
		} {
			if !strings.Contains(content, required) {
				t.Errorf("scripts/macos_osconfig.sh missing expected setting %q", required)
			}
		}
		for _, banned := range []string{
			"Resources/airport",
			"com.apple.dashboard",
			"LSQuarantine -bool false",
			"DSDontWriteUSBStores",
			"NSWindowResizeTime",
			"WebKit2BackspaceKeyNavigationEnabled",
			"NewWindowTargetPath",
			"defaults read com.apple.symbolichotkeys.plist",
			"NSGlobalDomain NSAppSleepDisabled",
			"powermode 2",
			"tcpkeepalive 0",
			"powernap 0",
			"csrutil",
			"sudo purge",
			"dynamic_pager",
		} {
			if strings.Contains(content, banned) {
				t.Errorf("scripts/macos_osconfig.sh still contains deprecated/unsafe command %q", banned)
			}
		}
	})
}

func TestMacOSOSConfigScriptDryRunAndVersionBounds(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX bash script test skipped on Windows")
	}

	scriptPath := filepath.Join("..", "..", "scripts", "macos_osconfig.sh")

	runWithStubOS := func(osName, productVersion string, args ...string) (string, error) {
		t.Helper()
		home := t.TempDir()
		binDir := t.TempDir()
		brewPrefix := t.TempDir()

		if err := os.MkdirAll(filepath.Join(brewPrefix, "lib", "pam"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(brewPrefix, "lib", "pam", "pam_reattach.so"), []byte("stub"), 0o644); err != nil {
			t.Fatal(err)
		}

		writeStub := func(name, body string) {
			t.Helper()
			if err := os.WriteFile(filepath.Join(binDir, name), []byte(body), 0o755); err != nil {
				t.Fatal(err)
			}
		}
		writeStub("uname", "#!/bin/sh\nif [ \"$1\" = \"-m\" ]; then echo arm64; else echo "+osName+"; fi\n")
		writeStub("sw_vers", "#!/bin/sh\necho "+productVersion+"\n")
		writeStub("brew", "#!/bin/sh\necho "+brewPrefix+"\n")

		cmdArgs := append([]string{scriptPath}, args...)
		cmd := exec.Command("bash", cmdArgs...)
		cmd.Env = []string{
			"HOME=" + home,
			"PATH=" + binDir + ":/usr/bin:/bin",
		}
		out, err := cmd.CombinedOutput()
		return string(out), err
	}

	t.Run("rejects non-darwin OS", func(t *testing.T) {
		out, err := runWithStubOS("Linux", "26.0", "--dry-run")
		if err == nil {
			t.Fatalf("expected non-zero exit on Linux, got nil (output: %s)", out)
		}
		if !strings.Contains(out, "only supports macOS") {
			t.Errorf("unexpected error output: %s", out)
		}
	})

	t.Run("macOS 27 dry-run default vs --all vs optional optimization flags", func(t *testing.T) {
		defaultOut, err := runWithStubOS("Darwin", "27.0", "--dry-run")
		if err != nil {
			t.Fatalf("expected macOS 27 default --dry-run to succeed, err=%v output:\n%s", err, defaultOut)
		}
		for _, want := range []string{
			"NSAppSleepDisabled",
			"NSScrollAnimationEnabled -bool false",
			"mineffect -string scale",
			"NewWindowTarget -string PfHm",
			"NSStatusItem\\ Visible\\ Siri -bool false",
			"com.apple.Siri StatusMenuVisible -bool false",
			"defaults read com.apple.symbolichotkeys",
			"activateSettings -u",
			"Open Safari once",
		} {
			if !strings.Contains(defaultOut, want) {
				t.Errorf("expected default --dry-run output to mention %q, got:\n%s", want, defaultOut)
			}
		}
		for _, id := range []string{"79", "80", "81", "82", "98", "118", "119", "120", "121", "122", "123", "124", "125", "126", "127"} {
			wantHotkey := "com.apple.symbolichotkeys AppleSymbolicHotKeys -dict-add " + id
			if !strings.Contains(defaultOut, wantHotkey) {
				t.Errorf("expected default --dry-run to mention missing hotkey %q, got:\n%s", wantHotkey, defaultOut)
			}
		}
		for _, unwanted := range []string{
			"pmset",
			"lsregister",
			"reduceTransparency",
			"reduceMotion",
			"defaults write com.apple.Safari",
		} {
			if strings.Contains(defaultOut, unwanted) {
				t.Errorf("default --dry-run should not include opt-in/unsafe %q, got:\n%s", unwanted, defaultOut)
			}
		}
		cfpIdx := strings.LastIndex(defaultOut, "killall cfprefsd")
		for _, daemon := range []string{"killall WindowManager", "killall ControlCenter", "killall SystemUIServer", "killall Dock", "killall Finder"} {
			idx := strings.Index(defaultOut, daemon)
			if cfpIdx == -1 || idx == -1 || cfpIdx > idx {
				t.Errorf("expected killall cfprefsd before %s in --dry-run, got:\n%s", daemon, defaultOut)
			}
		}

		allOut, err := runWithStubOS("Darwin", "27.0", "--all", "--dry-run")
		if err != nil {
			t.Fatalf("expected macOS 27 --all --dry-run to succeed, err=%v output:\n%s", err, allOut)
		}
		reattachIdx := strings.Index(allOut, "pam_reattach.so")
		tidIdx := strings.Index(allOut, "pam_tid.so")
		if reattachIdx == -1 || tidIdx == -1 || reattachIdx > tidIdx {
			t.Errorf("expected pam_reattach.so before pam_tid.so in sudo_local preview, got:\n%s", allOut)
		}
		for _, want := range []string{"EnableTiledWindowMargins", "BatteryShowPercentage", "ToggleTheme.shortcut", "pmset -a lowpowermode 0"} {
			if !strings.Contains(allOut, want) {
				t.Errorf("expected --all --dry-run output to mention %q, got:\n%s", want, allOut)
			}
		}
		for _, optInOnly := range []string{"reduceTransparency", "reduceMotion", "lsregister"} {
			if strings.Contains(allOut, optInOnly) {
				t.Errorf("expected --all --dry-run not to enable explicit opt-in flag %q, got:\n%s", optInOnly, allOut)
			}
		}

		powerOut, err := runWithStubOS("Darwin", "27.0", "--power", "--dry-run")
		if err != nil {
			t.Fatalf("expected --power --dry-run to succeed, err=%v output:\n%s", err, powerOut)
		}
		if !strings.Contains(powerOut, "sudo pmset -a lowpowermode 0") || strings.Contains(powerOut, "killall Dock") {
			t.Errorf("expected standalone --power --dry-run to only run pmset without UI restarts, got:\n%s", powerOut)
		}
		powerNoSudoOut, err := runWithStubOS("Darwin", "27.0", "--power", "--no-sudo", "--dry-run")
		if err != nil {
			t.Fatalf("expected --power --no-sudo --dry-run to succeed, err=%v output:\n%s", err, powerNoSudoOut)
		}
		if strings.Contains(powerNoSudoOut, "pmset -a") {
			t.Errorf("expected --power --no-sudo to skip pmset, got:\n%s", powerNoSudoOut)
		}

		optOut, err := runWithStubOS("Darwin", "27.0", "--reduce-ui-fx", "--reset-launchservices", "--dry-run")
		if err != nil {
			t.Fatalf("expected --reduce-ui-fx --reset-launchservices --dry-run to succeed, err=%v output:\n%s", err, optOut)
		}
		for _, want := range []string{"reduceTransparency -bool true", "reduceMotion -bool true", "lsregister -kill -r"} {
			if !strings.Contains(optOut, want) {
				t.Errorf("expected opt-in flags output to mention %q, got:\n%s", want, optOut)
			}
		}
	})

	t.Run("macOS 28 enforces max 27 guard unless --force is passed", func(t *testing.T) {
		out, err := runWithStubOS("Darwin", "28.0", "--dry-run")
		if err == nil {
			t.Fatalf("expected macOS 28 without --force to exit non-zero, output:\n%s", out)
		}
		if !strings.Contains(out, "exceeds supported maximum (macOS 27)") {
			t.Errorf("unexpected macOS 28 error message:\n%s", out)
		}

		forcedOut, forcedErr := runWithStubOS("Darwin", "28.0", "--force", "--dry-run")
		if forcedErr != nil {
			t.Fatalf("expected macOS 28 --force --dry-run to pass, err=%v output:\n%s", forcedErr, forcedOut)
		}
	})

	t.Run("tcc denial, safari container & FDA guard, PlistBuddy app-quit, spaces phases, power, and touchid idempotency", func(t *testing.T) {
		home := t.TempDir()
		binDir := t.TempDir()
		brewPrefix := t.TempDir()
		logFile := filepath.Join(t.TempDir(), "calls.log")

		if err := os.MkdirAll(filepath.Join(brewPrefix, "lib", "pam"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(brewPrefix, "lib", "pam", "pam_reattach.so"), []byte("stub"), 0o644); err != nil {
			t.Fatal(err)
		}
		prefsDir := filepath.Join(home, "Library", "Preferences")
		if err := os.MkdirAll(prefsDir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(prefsDir, "com.apple.finder.plist"), []byte("stub"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(prefsDir, "com.apple.Terminal.plist"), []byte("stub"), 0o644); err != nil {
			t.Fatal(err)
		}

		writeStub := func(name, body string) {
			t.Helper()
			if err := os.WriteFile(filepath.Join(binDir, name), []byte(body), 0o755); err != nil {
				t.Fatal(err)
			}
		}
		writeStub("uname", "#!/bin/sh\nif [ \"$1\" = \"-m\" ]; then echo arm64; else echo Darwin; fi\n")
		writeStub("sw_vers", "#!/bin/sh\necho 27.0\n")
		writeStub("brew", "#!/bin/sh\necho "+brewPrefix+"\n")
		writeStub("osascript", `#!/bin/sh
echo "osascript $*" >> "`+logFile+`"
exit 0
`)
		writeStub("chflags", "#!/bin/sh\nexit 0\n")
		writeStub("sleep", "#!/bin/sh\nexit 0\n")
		writeStub("activateSettings", `#!/bin/sh
echo "activateSettings $*" >> "`+logFile+`"
exit 0
`)
		writeStub("PlistBuddy", `#!/bin/sh
echo "PlistBuddy $*" >> "`+logFile+`"
case "$*" in
  *"Print :AppleSymbolicHotKeys:79:enabled"*) exit 0 ;;
  *"Print :AppleSymbolicHotKeys:79"*) exit 0 ;;
  *"Print :AppleSymbolicHotKeys:80:enabled"*) exit 1 ;;
  *"Print :AppleSymbolicHotKeys:80"*) exit 0 ;;
  *"Print :AppleSymbolicHotKeys:"*) exit 1 ;;
esac
exit 0
`)
		writeStub("defaults", `#!/bin/sh
echo "defaults $*" >> "`+logFile+`"
case "$*" in
  *com.apple.universalaccess*) exit 1 ;;
  *com.apple.Safari*)
    if [ "${ALLOW_SAFARI:-0}" = "1" ]; then exit 0; fi
    exit 1
    ;;
esac
exit 0
`)
		writeStub("killall", `#!/bin/sh
echo "killall $*" >> "`+logFile+`"
exit 0
`)
		// Require a template argument like macOS BSD mktemp.
		writeStub("mktemp", `#!/bin/sh
if [ $# -eq 0 ]; then
  echo "usage: mktemp [-d] template ..." >&2
  exit 1
fi
exec /usr/bin/mktemp "$1"
`)
		capturedSudoLocal := filepath.Join(t.TempDir(), "sudo_local.out")
		writeStub("sudo", `#!/bin/sh
if [ "$1" = "-v" ] || [ "$1" = "-n" ]; then
  exit 0
fi
if [ "$1" = "install" ]; then
  cp "$4" "`+capturedSudoLocal+`"
fi
echo "sudo $*" >> "`+logFile+`"
exit 0
`)

		env := []string{
			"HOME=" + home,
			"TMPDIR=" + t.TempDir(),
			"PATH=" + binDir + ":/usr/bin:/bin",
		}

		// 1. Clean Safari install warns and skips writing outside the container; Finder & Terminal quit before PlistBuddy; cfprefsd restarts before UI daemons.
		cmdDefs := exec.Command("bash", scriptPath, "--defaults", "--safari", "--no-sudo")
		cmdDefs.Env = env
		defsOut, err := cmdDefs.CombinedOutput()
		if err != nil {
			t.Fatalf("expected --defaults --safari --no-sudo to survive TCC & missing Safari container, err=%v out:\n%s", err, defsOut)
		}
		if !strings.Contains(string(defsOut), "Open Safari once") {
			t.Errorf("expected warning to open Safari once when container plist missing, got:\n%s", defsOut)
		}
		callsRaw, err := os.ReadFile(logFile)
		if err != nil {
			t.Fatal(err)
		}
		calls := string(callsRaw)
		if strings.Contains(calls, "defaults write com.apple.Safari") {
			t.Errorf("expected zero non-container Safari writes when container plist is missing, calls:\n%s", calls)
		}
		if !strings.Contains(calls, "com.apple.WindowManager EnableTiledWindowMargins") || !strings.Contains(calls, "com.apple.dock autohide") {
			t.Errorf("expected Finder/WindowManager/Dock writes after universalaccess refusal, calls:\n%s", calls)
		}
		quitFinderIdx := strings.Index(calls, `tell application "Finder" to quit`)
		pbFinderIdx := strings.Index(calls, "PlistBuddy -c Print :DesktopViewSettings")
		if quitFinderIdx == -1 || pbFinderIdx == -1 || quitFinderIdx > pbFinderIdx {
			t.Errorf("expected Finder quit before PlistBuddy finder edits, calls:\n%s", calls)
		}
		quitTermIdx := strings.Index(calls, `tell application "Terminal" to quit`)
		pbTermIdx := strings.Index(calls, `PlistBuddy -c Print :"Window Settings":Basic:shellExitAction`)
		if quitTermIdx == -1 || pbTermIdx == -1 || quitTermIdx > pbTermIdx {
			t.Errorf("expected Terminal quit before PlistBuddy Terminal edits, calls:\n%s", calls)
		}
		cfpIdx := strings.Index(calls, "killall cfprefsd")
		for _, daemon := range []string{"killall WindowManager", "killall ControlCenter", "killall SystemUIServer", "killall Dock", "killall Finder"} {
			idx := strings.Index(calls, daemon)
			if cfpIdx == -1 || idx == -1 || cfpIdx > idx {
				t.Errorf("expected killall cfprefsd before %s, calls:\n%s", daemon, calls)
			}
		}

		// 2. --reduce-ui-fx survives TCC denial cleanly.
		cmdUIFx := exec.Command("bash", scriptPath, "--reduce-ui-fx", "--no-restart")
		cmdUIFx.Env = env
		uiFxOut, err := cmdUIFx.CombinedOutput()
		if err != nil {
			t.Fatalf("expected --reduce-ui-fx to survive TCC denial, err=%v out:\n%s", err, uiFxOut)
		}
		if !strings.Contains(string(uiFxOut), "Grant Full Disk Access to Ghostty") {
			t.Errorf("expected Ghostty FDA warning when --reduce-ui-fx hits TCC refusal, got:\n%s", uiFxOut)
		}

		// 3. Safari container checks: chmod 000 Data dir (simulates macOS TCC EPERM) -> FDA warning; accessible plist -> writes inside container only.
		safariPlist := filepath.Join(home, "Library", "Containers", "com.apple.Safari", "Data", "Library", "Preferences", "com.apple.Safari.plist")
		safariDataDir := filepath.Join(home, "Library", "Containers", "com.apple.Safari", "Data")
		if err := os.MkdirAll(filepath.Dir(safariPlist), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(safariPlist, []byte("stub"), 0o644); err != nil {
			t.Fatal(err)
		}
		if os.Geteuid() != 0 {
			if err := os.Chmod(safariDataDir, 0o000); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = os.Chmod(safariDataDir, 0o755) })
			cmdSafariEperm := exec.Command("bash", scriptPath, "--safari", "--no-restart")
			cmdSafariEperm.Env = env
			epermOut, err := cmdSafariEperm.CombinedOutput()
			_ = os.Chmod(safariDataDir, 0o755)
			if err != nil {
				t.Fatalf("expected --safari to survive Data EPERM cleanly, err=%v out:\n%s", err, epermOut)
			}
			if !strings.Contains(string(epermOut), "Grant Full Disk Access to Ghostty") {
				t.Errorf("expected FDA guidance when container Data dir raises permission denied, got:\n%s", epermOut)
			}
		}

		// Accessible Safari container writes inside safariPlist and never bare com.apple.Safari.
		if err := os.WriteFile(logFile, []byte(""), 0o644); err != nil {
			t.Fatal(err)
		}
		cmdSafariOK := exec.Command("bash", scriptPath, "--safari", "--no-restart")
		cmdSafariOK.Env = append(append([]string{}, env...), "ALLOW_SAFARI=1")
		if out, err := cmdSafariOK.CombinedOutput(); err != nil {
			t.Fatalf("expected --safari with accessible container to succeed, err=%v out:\n%s", err, out)
		}
		safariCallsRaw, _ := os.ReadFile(logFile)
		safariCalls := string(safariCallsRaw)
		if !strings.Contains(safariCalls, "defaults write "+safariPlist+" IncludeDevelopMenu -bool true") {
			t.Errorf("expected Safari preferences written inside container plist, calls:\n%s", safariCalls)
		}
		if strings.Contains(safariCalls, "defaults write com.apple.Safari ") {
			t.Errorf("Safari writes must target container plist, not bare com.apple.Safari domain, calls:\n%s", safariCalls)
		}

		// 4. --spaces creates missing IDs BEFORE PlistBuddy Set/Add, then flushes cfprefsd BEFORE defaults read and activateSettings -u.
		if err := os.WriteFile(logFile, []byte(""), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(prefsDir, "com.apple.symbolichotkeys.plist"), []byte("stub"), 0o644); err != nil {
			t.Fatal(err)
		}
		cmdSpaces := exec.Command("bash", scriptPath, "--spaces")
		cmdSpaces.Env = env
		if out, err := cmdSpaces.CombinedOutput(); err != nil {
			t.Fatalf("expected --spaces to succeed, err=%v out:\n%s", err, out)
		}
		spacesCallsRaw, _ := os.ReadFile(logFile)
		spacesCalls := string(spacesCallsRaw)
		for _, id := range []string{"81", "82", "98", "118", "119", "120", "121", "122", "123", "124", "125", "126", "127"} {
			want := "defaults write com.apple.symbolichotkeys AppleSymbolicHotKeys -dict-add " + id
			if !strings.Contains(spacesCalls, want) {
				t.Errorf("expected missing hotkey %q in --spaces calls:\n%s", want, spacesCalls)
			}
		}
		dictAdd127Idx := strings.Index(spacesCalls, "AppleSymbolicHotKeys -dict-add 127")
		pb79Idx := strings.Index(spacesCalls, "PlistBuddy -c Set :AppleSymbolicHotKeys:79:enabled true")
		pb80Idx := strings.Index(spacesCalls, "PlistBuddy -c Add :AppleSymbolicHotKeys:80:enabled bool true")
		flushIdx := strings.Index(spacesCalls, "killall cfprefsd")
		readIdx := strings.Index(spacesCalls, "defaults read com.apple.symbolichotkeys\n")
		actIdx := strings.Index(spacesCalls, "activateSettings -u")
		if dictAdd127Idx == -1 || pb79Idx == -1 || pb80Idx == -1 || flushIdx == -1 || readIdx == -1 || actIdx == -1 {
			t.Fatalf("missing expected --spaces phase calls:\n%s", spacesCalls)
		}
		if !(dictAdd127Idx < pb79Idx && pb79Idx < pb80Idx && pb80Idx < flushIdx && flushIdx < readIdx && readIdx < actIdx) {
			t.Errorf("expected order dict-add -> PlistBuddy Set/Add -> killall cfprefsd -> defaults read -> activateSettings -u, calls:\n%s", spacesCalls)
		}

		// 5. --power live stub executes sudo pmset -a lowpowermode 0 without UI restarts.
		if err := os.WriteFile(logFile, []byte(""), 0o644); err != nil {
			t.Fatal(err)
		}
		cmdPower := exec.Command("bash", scriptPath, "--power")
		cmdPower.Env = env
		if out, err := cmdPower.CombinedOutput(); err != nil {
			t.Fatalf("expected --power to succeed, err=%v out:\n%s", err, out)
		}
		powerCallsRaw, _ := os.ReadFile(logFile)
		if !strings.Contains(string(powerCallsRaw), "sudo pmset -a lowpowermode 0") || strings.Contains(string(powerCallsRaw), "killall") {
			t.Errorf("unexpected --power calls:\n%s", powerCallsRaw)
		}

		// 6. --touchid succeeds idempotently with strict BSD mktemp and does NOT restart Dock/Finder.
		for pass := 1; pass <= 2; pass++ {
			if err := os.WriteFile(logFile, []byte(""), 0o644); err != nil {
				t.Fatal(err)
			}
			cmdTID := exec.Command("bash", scriptPath, "--touchid")
			cmdTID.Env = env
			if out, err := cmdTID.CombinedOutput(); err != nil {
				t.Fatalf("pass %d: expected --touchid with BSD mktemp stub to succeed, err=%v out:\n%s", pass, err, out)
			}
			tidCallsRaw, _ := os.ReadFile(logFile)
			if strings.Contains(string(tidCallsRaw), "killall") {
				t.Errorf("pass %d: --touchid alone should not restart UI daemons via killall, got:\n%s", pass, tidCallsRaw)
			}
		}
		sudoLocalBytes, err := os.ReadFile(capturedSudoLocal)
		if err != nil {
			t.Fatalf("expected sudo install to copy generated sudo_local: %v", err)
		}
		sudoLocal := string(sudoLocalBytes)
		if !strings.Contains(sudoLocal, "pam_reattach.so") || !strings.Contains(sudoLocal, "pam_tid.so") {
			t.Errorf("unexpected sudo_local content:\n%s", sudoLocal)
		}
	})
}

func TestSSHAgentAssetsAndAllProfilesCoverage(t *testing.T) {
	cat, err := manifest.LoadCatalog()
	if err != nil {
		t.Fatalf("LoadCatalog() failed: %v", err)
	}

	sshCfg, ok := cat.GetConfiguration("ssh-config")
	if !ok {
		t.Fatal("expected 'ssh-config' configuration in manifests/configs.yaml")
	}
	if sshCfg.DirMode != "0700" || sshCfg.FileMode != "0600" {
		t.Errorf("expected ssh-config dir_mode=0700 file_mode=0600, got %q / %q", sshCfg.DirMode, sshCfg.FileMode)
	}
	if !sshCfg.SkipCapture {
		t.Errorf("expected ssh-config to set skip_capture: true so user SSH hosts never leak")
	}

	requiredIDs := []string{
		"ssh-config",
		"ssh-agent-service",
		"ssh-agent-env",
		"ssh-agent-zshenv",
		"ssh-agent-profile",
	}
	for _, prof := range cat.ListProfiles() {
		cfgs, err := cat.ResolveProfileConfigurations(prof.Name)
		if err != nil {
			t.Fatalf("ResolveProfileConfigurations(%q): %v", prof.Name, err)
		}
		present := map[string]bool{}
		for _, c := range cfgs {
			present[c.ID] = true
		}
		for _, req := range requiredIDs {
			if !present[req] {
				t.Errorf("profile %q missing required SSH configuration %q", prof.Name, req)
			}
		}
	}

	sshConfigRaw, err := os.ReadFile(filepath.Join("..", "..", "files", "ssh", "config"))
	if err != nil {
		t.Fatalf("reading files/ssh/config: %v", err)
	}
	sshConfig := string(sshConfigRaw)
	for _, want := range []string{"Host *", "AddKeysToAgent yes", "IdentitiesOnly yes", "IgnoreUnknown UseKeychain", "UseKeychain yes"} {
		if !strings.Contains(sshConfig, want) {
			t.Errorf("files/ssh/config missing %q", want)
		}
	}
	if strings.Contains(sshConfig, "ForwardAgent yes") {
		t.Errorf("files/ssh/config must NEVER configure ForwardAgent yes")
	}

	unitRaw, err := os.ReadFile(filepath.Join("..", "..", "files", "ssh", "systemd", "ssh-agent.service"))
	if err != nil {
		t.Fatalf("reading files/ssh/systemd/ssh-agent.service: %v", err)
	}
	for _, want := range []string{"ExecStartPre=-/bin/rm -f %t/ssh-agent.socket", "ExecStart=/usr/bin/ssh-agent -D -a %t/ssh-agent.socket", "WantedBy=default.target"} {
		if !strings.Contains(string(unitRaw), want) {
			t.Errorf("ssh-agent.service missing %q", want)
		}
	}

	envRaw, err := os.ReadFile(filepath.Join("..", "..", "files", "ssh", "environment.d", "ssh-agent.conf"))
	if err != nil {
		t.Fatalf("reading files/ssh/environment.d/ssh-agent.conf: %v", err)
	}
	if !strings.Contains(string(envRaw), "SSH_AUTH_SOCK=${XDG_RUNTIME_DIR}/ssh-agent.socket") {
		t.Errorf("unexpected ssh-agent.conf content:\n%s", envRaw)
	}

	// Managed shell files must not manually spawn ssh-agent or run ssh-add at startup.
	shellFiles := []string{
		filepath.Join("..", "..", "files", "zsh", ".zshrc"),
		filepath.Join("..", "..", "files", "zsh", ".base.zsh"),
		filepath.Join("..", "..", "files", "zsh", ".tools.sh"),
		filepath.Join("..", "..", "files", "zsh", ".zshenv"),
		filepath.Join("..", "..", "files", "bash", ".bashrc"),
		filepath.Join("..", "..", "files", "bash", ".base.bash"),
		filepath.Join("..", "..", "files", "ssh", ".profile"),
	}
	for _, sf := range shellFiles {
		raw, err := os.ReadFile(sf)
		if err != nil {
			t.Fatalf("reading %s: %v", sf, err)
		}
		txt := string(raw)
		for _, banned := range []string{"ssh-agent -s", "ssh-agent -c", "eval \"$(ssh-agent", "eval '$(ssh-agent", "ssh-add "} {
			if strings.Contains(txt, banned) {
				t.Errorf("%s contains manual ssh-agent/ssh-add invocation %q", sf, banned)
			}
		}
	}

	if runtime.GOOS == "windows" {
		return
	}

	// Execute guarded ~/.zshenv and ~/.profile under stubbed Linux vs Darwin uname.
	for _, tc := range []struct {
		osName   string
		xdgDir   string
		wantSock string
	}{
		{osName: "Linux", xdgDir: "/run/user/1000", wantSock: "/run/user/1000/ssh-agent.socket"},
		{osName: "Linux", xdgDir: "", wantSock: "launchd-original"},
		{osName: "Darwin", xdgDir: "/tmp/should-ignore", wantSock: "launchd-original"},
	} {
		for _, relPath := range []string{filepath.Join("..", "..", "files", "zsh", ".zshenv"), filepath.Join("..", "..", "files", "ssh", ".profile")} {
			binDir := t.TempDir()
			if err := os.WriteFile(filepath.Join(binDir, "uname"), []byte("#!/bin/sh\necho "+tc.osName+"\n"), 0o755); err != nil {
				t.Fatal(err)
			}
			raw, err := os.ReadFile(relPath)
			if err != nil {
				t.Fatal(err)
			}
			cmd := exec.Command("sh", "-c", string(raw)+"\nprintf '%s' \"$SSH_AUTH_SOCK\"")
			env := []string{
				"PATH=" + binDir + ":/usr/bin:/bin",
				"SSH_AUTH_SOCK=launchd-original",
			}
			if tc.xdgDir != "" {
				env = append(env, "XDG_RUNTIME_DIR="+tc.xdgDir)
			}
			cmd.Env = env
			out, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("executing %s (%s): %v out=%s", relPath, tc.osName, err, out)
			}
			if string(out) != tc.wantSock {
				t.Errorf("%s (OS=%s XDG=%q): SSH_AUTH_SOCK = %q, want %q", relPath, tc.osName, tc.xdgDir, string(out), tc.wantSock)
			}
		}
	}
}
