package app_test

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/arturonavax/env/internal/app"
	"github.com/arturonavax/env/internal/catalog"
	"github.com/arturonavax/env/internal/domain"
	"github.com/arturonavax/env/internal/journal"
	"github.com/arturonavax/env/internal/platform"
	"github.com/arturonavax/env/internal/providers"
	"github.com/arturonavax/env/internal/security"
	"github.com/arturonavax/env/internal/snapshot"
	"github.com/arturonavax/env/internal/state"
)

type fixture struct {
	plat  *platform.Info
	cat   *catalog.Registry
	snap  *snapshot.Manager
	store *state.Store
	repo  string
	home  string
}

func newFixture(t *testing.T) fixture {
	t.Helper()
	home := t.TempDir()
	repo := t.TempDir()
	plat := platform.NewTestInfo(platform.TestInfoParams{OS: "linux", HomeDir: home, StateDir: filepath.Join(home, "state")})
	snap, err := snapshot.NewManager(plat.StateDir())
	if err != nil {
		t.Fatal(err)
	}
	store, err := state.OpenStore(plat.StateDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = store.Close()
	})
	return fixture{plat: plat, cat: catalog.NewDefaultRegistry(), snap: snap, store: store, repo: repo, home: home}
}

func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestInstallServicePlanResolvesProfileAndOrdersDeps(t *testing.T) {
	fx := newFixture(t)
	j, _ := journal.NewFileJournal(fx.plat.StateDir())
	prov := providers.NewBinaryProvider(security.NewRunner(0))
	sync := app.NewSyncService(fx.plat, fx.cat, fx.snap, fx.store, fx.repo)
	svc := app.NewInstallService(app.InstallDeps{
		Platform: fx.plat, Catalog: fx.cat, Journal: j, Snapshots: fx.snap, Provider: prov, Store: fx.store, Sync: sync,
	})

	plan, err := svc.Plan(context.Background(), "minimal", nil)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Profile != "minimal" || len(plan.Steps) == 0 {
		t.Fatalf("unexpected plan: %+v", plan)
	}
	for _, s := range plan.Steps {
		if s.Action == domain.ActionInstall && len(s.Locks) == 0 {
			t.Fatalf("install step %s must declare locks for fine-grained concurrency", s.ResourceID)
		}
	}
}

func TestInstallServicePrioritizePackageIDs(t *testing.T) {
	fx := newFixture(t)
	svc := app.NewInstallService(app.InstallDeps{
		Platform: fx.plat, Catalog: fx.cat, Snapshots: fx.snap, Store: fx.store,
	})

	input := []string{"ripgrep", "unlinked-pkg", "ghostty", "neovim", "zsh", "herdr", "git"}
	prioritized := svc.PrioritizePackageIDs(input)

	// Base tools must come first in priority order: git, ghostty, zsh, herdr, neovim
	expectedBasePrefix := []string{"git", "ghostty", "zsh", "herdr", "neovim"}
	if len(prioritized) != len(input) {
		t.Fatalf("expected %d packages, got %d", len(input), len(prioritized))
	}
	for i, expected := range expectedBasePrefix {
		if prioritized[i] != expected {
			t.Errorf("pos %d: expected %s, got %s", i, expected, prioritized[i])
		}
	}

	// ripgrep (Tier 1 direct tool) must come before unlinked-pkg (Tier 3)
	rgIdx := -1
	unlinkedIdx := -1
	for i, id := range prioritized {
		if id == "ripgrep" {
			rgIdx = i
		}
		if id == "unlinked-pkg" {
			unlinkedIdx = i
		}
	}
	if rgIdx == -1 || unlinkedIdx == -1 {
		t.Fatalf("missing ripgrep or unlinked-pkg in prioritized list: %v", prioritized)
	}
	if rgIdx >= unlinkedIdx {
		t.Errorf("expected ripgrep (idx %d) before unlinked-pkg (idx %d)", rgIdx, unlinkedIdx)
	}
}

func TestSyncThreeWayUsesRecordedBase(t *testing.T) {
	fx := newFixture(t)
	svc := app.NewSyncService(fx.plat, fx.cat, fx.snap, fx.store, fx.repo)
	src := filepath.Join(fx.repo, "files/starship/starship.toml")
	dst := filepath.Join(fx.home, ".config/starship.toml")
	ctx := context.Background()

	write(t, src, "a=1\nb=2\nc=3\n")
	if _, err := svc.Sync(ctx, app.SyncOptions{Group: "starship-config"}); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(dst); string(got) != "a=1\nb=2\nc=3\n" {
		t.Fatalf("initial deploy mismatch: %q", got)
	}

	// Local edits the top, repo edits the bottom: non-overlapping -> clean auto-merge.
	write(t, dst, "a=LOCAL\nb=2\nc=3\n")
	write(t, src, "a=1\nb=2\nc=REPO\n")
	res, err := svc.Sync(ctx, app.SyncOptions{Group: "starship-config"})
	if err != nil {
		t.Fatal(err)
	}
	if res[0].Action == "conflict" {
		t.Fatalf("expected auto-merge, got conflict")
	}
	if got, _ := os.ReadFile(dst); string(got) != "a=LOCAL\nb=2\nc=REPO\n" {
		t.Fatalf("merge result mismatch: %q", got)
	}
}

func TestSyncDirectoryAndMarkerBlockPreserveUserContent(t *testing.T) {
	fx := newFixture(t)
	svc := app.NewSyncService(fx.plat, fx.cat, fx.snap, fx.store, fx.repo)
	ctx := context.Background()

	write(t, filepath.Join(fx.repo, "files/nvim/init.lua"), "-- init")
	write(t, filepath.Join(fx.repo, "files/nvim/lua/a.lua"), "-- a")
	write(t, filepath.Join(fx.repo, "files/zsh/.zshrc"), "source ~/.base.zsh")
	write(t, filepath.Join(fx.home, ".zshrc"), "export USER_SECRET=1\n")

	if _, err := svc.Sync(ctx, app.SyncOptions{Group: "nvim-config"}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(fx.home, ".config/nvim/lua/a.lua")); err != nil {
		t.Fatalf("recursive directory deploy failed: %v", err)
	}
	if _, err := svc.Sync(ctx, app.SyncOptions{Group: "zshrc-loader"}); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(filepath.Join(fx.home, ".zshrc"))
	if string(got) == "" || !contains(string(got), "USER_SECRET") || !contains(string(got), "source ~/.base.zsh") {
		t.Fatalf("marker block must preserve user content, got %q", got)
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (func() bool {
		for i := 0; i+len(sub) <= len(s); i++ {
			if s[i:i+len(sub)] == sub {
				return true
			}
		}
		return false
	})()
}

func TestDoctorService(t *testing.T) {
	fx := newFixture(t)
	checks, err := app.NewDoctorService(fx.plat, fx.cat).Diagnose(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(checks) == 0 {
		t.Fatal("expected diagnostic checks")
	}
}

func TestControlPlaneServiceExplainAndInventory(t *testing.T) {
	fx := newFixture(t)
	sync := app.NewSyncService(fx.plat, fx.cat, fx.snap, fx.store, fx.repo)
	cp := app.NewControlPlaneService(fx.plat, fx.cat, nil, sync, nil, nil, fx.store)
	ctx := context.Background()

	// 1. Inventory query
	inv, err := cp.Inventory(ctx, app.InventoryFilter{})
	if err != nil {
		t.Fatalf("inventory query failed: %v", err)
	}
	if len(inv.Packages) == 0 {
		t.Errorf("expected packages in inventory, got 0")
	}
	if len(inv.Configurations) == 0 {
		t.Errorf("expected configurations in inventory, got 0")
	}

	// 2. Explain Package
	provPkg, err := cp.Explain(ctx, domain.ResourceID("ripgrep"))
	if err != nil {
		t.Fatalf("explain ripgrep failed: %v", err)
	}
	if provPkg.Kind != domain.KindPackage {
		t.Errorf("expected kind %s, got %s", domain.KindPackage, provPkg.Kind)
	}
	if provPkg.Origin == "" {
		t.Errorf("expected non-empty origin for ripgrep")
	}

	// 3. Explain Config
	provCfg, err := cp.Explain(ctx, domain.ResourceID("starship-config"))
	if err != nil {
		t.Fatalf("explain starship-config failed: %v", err)
	}
	if provCfg.Kind != domain.KindConfig {
		t.Errorf("expected kind %s, got %s", domain.KindConfig, provCfg.Kind)
	}

	// 4. Explain Unknown
	_, err = cp.Explain(ctx, domain.ResourceID("non-existent-artifact"))
	if err == nil {
		t.Errorf("expected error explaining non-existent artifact")
	}
}

func TestSyncPostSyncErrorSurfacesAsPartial(t *testing.T) {
	fx := newFixture(t)
	// Register a test config with a failing post-sync command
	testCfg := domain.ConfigurationResource{
		ID:           "test-post-sync-fail",
		Source:       "files/test/fail.txt",
		Target:       "~/.config/test/fail.txt",
		PostSyncCmds: [][]string{{"sh", "-c", "echo 'critical post-sync hook failure' >&2; exit 1"}},
	}
	fx.cat.AddConfiguration(testCfg)

	svc := app.NewSyncService(fx.plat, fx.cat, fx.snap, fx.store, fx.repo)
	src := filepath.Join(fx.repo, "files/test/fail.txt")
	write(t, src, "hello world")

	_, err := svc.SyncOne(context.Background(), "test-post-sync-fail", app.SyncOptions{})
	if err == nil {
		t.Fatalf("expected non-nil error when post-sync fails, got nil")
	}
	if !contains(err.Error(), "critical post-sync hook failure") {
		t.Fatalf("expected error to capture stderr, got: %v", err)
	}
}

func TestInstallServicePlanRequiresSudo(t *testing.T) {
	home := t.TempDir()
	platLinuxSudo := platform.NewTestInfo(platform.TestInfoParams{
		OS:              "linux",
		HasSudo:         true,
		IsRoot:          false,
		PackageManagers: []string{"apt"},
		HomeDir:         home,
		StateDir:        filepath.Join(home, "state"),
	})
	platRoot := platform.NewTestInfo(platform.TestInfoParams{
		OS:              "linux",
		HasSudo:         true,
		IsRoot:          true,
		PackageManagers: []string{"apt"},
		HomeDir:         home,
		StateDir:        filepath.Join(home, "state"),
	})
	platNoSudo := platform.NewTestInfo(platform.TestInfoParams{
		OS:              "linux",
		HasSudo:         false,
		IsRoot:          false,
		PackageManagers: []string{"apt"},
		HomeDir:         home,
		StateDir:        filepath.Join(home, "state"),
	})

	cat := catalog.NewDefaultRegistry()
	svcLinuxSudo := app.NewInstallService(app.InstallDeps{Platform: platLinuxSudo, Catalog: cat})
	svcRoot := app.NewInstallService(app.InstallDeps{Platform: platRoot, Catalog: cat})
	svcNoSudo := app.NewInstallService(app.InstallDeps{Platform: platNoSudo, Catalog: cat})

	planWithGit := &domain.ExecutionPlan{
		Steps: []domain.PlanStep{
			{ID: 1, ResourceID: "git", Action: domain.ActionInstall},
		},
	}
	emptyPlan := &domain.ExecutionPlan{}

	if !svcLinuxSudo.PlanRequiresSudo(planWithGit) {
		t.Errorf("expected PlanRequiresSudo(git) on Linux with sudo and apt to be true")
	}
	if svcRoot.PlanRequiresSudo(planWithGit) {
		t.Errorf("expected PlanRequiresSudo for root user to be false")
	}
	if svcNoSudo.PlanRequiresSudo(planWithGit) {
		t.Errorf("expected PlanRequiresSudo for host without sudo to be false")
	}
	if svcLinuxSudo.PlanRequiresSudo(emptyPlan) {
		t.Errorf("expected PlanRequiresSudo for empty plan to be false")
	}

	dispLinux := providers.NewDispatcher(platLinuxSudo, cat.Catalog(), security.NewRunner(0), "")
	svcWithDispatcher := app.NewInstallService(app.InstallDeps{Platform: platLinuxSudo, Catalog: cat, Provider: dispLinux})
	planWithGitHubRelease := &domain.ExecutionPlan{
		Steps: []domain.PlanStep{
			{ID: 1, ResourceID: "rtk", Action: domain.ActionInstall},
		},
	}
	if svcWithDispatcher.PlanRequiresSudo(planWithGitHubRelease) {
		t.Errorf("expected PlanRequiresSudo(rtk) to be false when winning strategy is github_release on linux/apt")
	}
	if !svcWithDispatcher.PlanRequiresSudo(planWithGit) {
		t.Errorf("expected PlanRequiresSudo(git) to be true when winning strategy is native apt")
	}
}

func TestInstallServicePlanFiltersByOSDisplayServerAndStrategy(t *testing.T) {
	cat := catalog.NewDefaultRegistry()
	runner := security.NewRunner(0)

	hasStep := func(plan *domain.ExecutionPlan, id string) bool {
		for _, st := range plan.Steps {
			if st.Action == domain.ActionInstall && st.ResourceID == id {
				return true
			}
		}
		return false
	}

	t.Run("macos developer avoids linux build deps and pyenv dev headers", func(t *testing.T) {
		home := t.TempDir()
		plat := platform.NewTestInfo(platform.TestInfoParams{
			OS:              "darwin",
			Arch:            "arm64",
			PackageManagers: []string{"brew"},
			DisplayServer:   "quartz",
			HomeDir:         home,
			StateDir:        filepath.Join(home, "state"),
		})
		disp := providers.NewDispatcher(plat, cat.Catalog(), runner, "")
		svc := app.NewInstallService(app.InstallDeps{Platform: plat, Catalog: cat, Provider: disp})

		plan, err := svc.PlanWithReinstall(context.Background(), "workstation", nil, true)
		if err != nil {
			t.Fatalf("Plan failed: %v", err)
		}

		forbidden := []string{
			"zig", "wget", "cmake", "pkg-config",
			"build-essential", "make", "gcc", "binutils",
			"bz2-dev", "ssl-dev", "zlib-dev", "readline-dev", "sqlite-dev", "ffi-dev", "xz-dev", "ncurses-dev",
			"wmctrl", "xdotool", "xclip",
		}
		for _, id := range forbidden {
			if hasStep(plan, id) {
				t.Errorf("macOS workstation plan unexpectedly scheduled %q", id)
			}
		}
	})

	t.Run("windows toolchains and ai tools avoid curl, pnpm, and linux build deps", func(t *testing.T) {
		home := t.TempDir()
		plat := platform.NewTestInfo(platform.TestInfoParams{
			OS:              "windows",
			Arch:            "amd64",
			PackageManagers: []string{"winget"},
			DisplayServer:   "win32",
			HomeDir:         home,
			StateDir:        filepath.Join(home, "state"),
		})
		disp := providers.NewDispatcher(plat, cat.Catalog(), runner, "")
		svc := app.NewInstallService(app.InstallDeps{Platform: plat, Catalog: cat, Provider: disp})

		plan, err := svc.PlanWithReinstall(context.Background(), "", []string{"rust", "golang", "zig", "herdr", "fnm", "rtk", "opencode", "claude-code", "codex-cli"}, true)
		if err != nil {
			t.Fatalf("Plan failed: %v", err)
		}

		for _, id := range []string{"curl", "pnpm", "build-essential", "make", "gcc", "binutils"} {
			if hasStep(plan, id) {
				t.Errorf("Windows plan unexpectedly scheduled %q", id)
			}
		}
	})

	t.Run("linux wayland workstation skips x11 tools and terminal skips zig", func(t *testing.T) {
		home := t.TempDir()
		plat := platform.NewTestInfo(platform.TestInfoParams{
			OS:              "linux",
			Arch:            "amd64",
			Distribution:    "ubuntu",
			PackageManagers: []string{"apt"},
			DisplayServer:   "wayland",
			HomeDir:         home,
			StateDir:        filepath.Join(home, "state"),
		})
		disp := providers.NewDispatcher(plat, cat.Catalog(), runner, "")
		svc := app.NewInstallService(app.InstallDeps{Platform: plat, Catalog: cat, Provider: disp})

		plan, err := svc.PlanWithReinstall(context.Background(), "workstation", nil, true)
		if err != nil {
			t.Fatalf("Plan failed: %v", err)
		}

		for _, id := range []string{"zig", "wmctrl", "xdotool", "xclip", "wget", "cmake", "pkg-config"} {
			if hasStep(plan, id) {
				t.Errorf("Linux Wayland workstation plan unexpectedly scheduled %q", id)
			}
		}
		if !hasStep(plan, "build-essential") || !hasStep(plan, "ssl-dev") {
			t.Errorf("Linux workstation plan should include build-essential and ssl-dev for rust/pyenv")
		}
		if hasStep(plan, "make") || hasStep(plan, "gcc") || hasStep(plan, "binutils") {
			t.Errorf("Linux plan should not duplicate make/gcc/binutils alongside build-essential")
		}
	})

	t.Run("linux x11 workstation keeps x11 tools", func(t *testing.T) {
		home := t.TempDir()
		plat := platform.NewTestInfo(platform.TestInfoParams{
			OS:              "linux",
			Arch:            "amd64",
			Distribution:    "ubuntu",
			PackageManagers: []string{"apt"},
			DisplayServer:   "x11",
			HomeDir:         home,
			StateDir:        filepath.Join(home, "state"),
		})
		disp := providers.NewDispatcher(plat, cat.Catalog(), runner, "")
		svc := app.NewInstallService(app.InstallDeps{Platform: plat, Catalog: cat, Provider: disp})

		plan, err := svc.PlanWithReinstall(context.Background(), "workstation", nil, true)
		if err != nil {
			t.Fatalf("Plan failed: %v", err)
		}

		for _, id := range []string{"wmctrl", "xdotool", "xclip"} {
			if !hasStep(plan, id) {
				t.Errorf("Linux X11 workstation plan missing %q", id)
			}
		}
	})

	t.Run("minimal profile preserves core user tools", func(t *testing.T) {
		home := t.TempDir()
		plat := platform.NewTestInfo(platform.TestInfoParams{
			OS:              "linux",
			Arch:            "amd64",
			Distribution:    "ubuntu",
			PackageManagers: []string{"apt"},
			DisplayServer:   "x11",
			HomeDir:         home,
			StateDir:        filepath.Join(home, "state"),
		})
		disp := providers.NewDispatcher(plat, cat.Catalog(), runner, "")
		svc := app.NewInstallService(app.InstallDeps{Platform: plat, Catalog: cat, Provider: disp})

		plan, err := svc.PlanWithReinstall(context.Background(), "minimal", nil, true)
		if err != nil {
			t.Fatalf("Plan failed: %v", err)
		}
		for _, id := range []string{"git", "curl", "htop", "jq", "fzf", "ripgrep", "fd", "bat", "eza", "tree", "starship"} {
			if !hasStep(plan, id) {
				t.Errorf("minimal plan missing core user tool %q", id)
			}
		}
	})
}

func TestControlPlaneServiceExplainAndInventoryEffectiveDeps(t *testing.T) {
	cat := catalog.NewDefaultRegistry()
	runner := security.NewRunner(0)
	ctx := context.Background()

	buildControlPlane := func(params platform.TestInfoParams) *app.ControlPlaneService {
		home := t.TempDir()
		params.HomeDir = home
		params.StateDir = filepath.Join(home, "state")
		plat := platform.NewTestInfo(params)
		disp := providers.NewDispatcher(plat, cat.Catalog(), runner, "")
		installSvc := app.NewInstallService(app.InstallDeps{Platform: plat, Catalog: cat, Provider: disp})
		return app.NewControlPlaneService(plat, cat, installSvc, nil, nil, nil, nil)
	}

	cpLinux := buildControlPlane(platform.TestInfoParams{
		OS:              "linux",
		Arch:            "amd64",
		Distribution:    "ubuntu",
		PackageManagers: []string{"apt"},
	})
	provLinux, err := cpLinux.Explain(ctx, domain.ResourceID("pyenv"))
	if err != nil {
		t.Fatalf("Explain(pyenv) on linux failed: %v", err)
	}
	if provLinux.WinningPolicy != "strategy:script" {
		t.Errorf("expected linux pyenv WinningPolicy strategy:script, got %q", provLinux.WinningPolicy)
	}
	if !slices.Contains(provLinux.Dependencies, domain.ResourceID("ssl-dev")) ||
		!slices.Contains(provLinux.Dependencies, domain.ResourceID("build-essential")) {
		t.Errorf("expected linux pyenv provenance dependencies to include ssl-dev and build-essential, got %v", provLinux.Dependencies)
	}

	cpDarwin := buildControlPlane(platform.TestInfoParams{
		OS:              "darwin",
		Arch:            "arm64",
		PackageManagers: []string{"brew"},
	})
	provDarwin, err := cpDarwin.Explain(ctx, domain.ResourceID("pyenv"))
	if err != nil {
		t.Fatalf("Explain(pyenv) on darwin failed: %v", err)
	}
	if provDarwin.WinningPolicy != "strategy:native" {
		t.Errorf("expected darwin brew pyenv WinningPolicy strategy:native, got %q", provDarwin.WinningPolicy)
	}
	if len(provDarwin.Dependencies) != 0 {
		t.Errorf("expected darwin brew pyenv provenance dependencies to be empty, got %v", provDarwin.Dependencies)
	}

	invDarwin, err := cpDarwin.Inventory(ctx, app.InventoryFilter{Kind: domain.KindPackage})
	if err != nil {
		t.Fatalf("Inventory on darwin failed: %v", err)
	}
	for _, pkg := range invDarwin.Packages {
		if pkg.ID == "pyenv" && len(pkg.DependsOn) != 0 {
			t.Errorf("expected darwin Inventory pyenv DependsOn to be empty, got %v", pkg.DependsOn)
		}
	}

	cpWayland := buildControlPlane(platform.TestInfoParams{
		OS:              "linux",
		Arch:            "amd64",
		Distribution:    "ubuntu",
		PackageManagers: []string{"apt"},
		DisplayServer:   "wayland",
	})
	provXclipWayland, err := cpWayland.Explain(ctx, domain.ResourceID("xclip"))
	if err != nil {
		t.Fatalf("Explain(xclip) on wayland failed: %v", err)
	}
	if provXclipWayland.WinningPolicy != "unsupported" {
		t.Errorf("expected wayland xclip WinningPolicy unsupported, got %q", provXclipWayland.WinningPolicy)
	}
	if !slices.Contains(provXclipWayland.Conditions, "display:x11") {
		t.Errorf("expected xclip Conditions to include display:x11, got %v", provXclipWayland.Conditions)
	}
}

func TestInstallServicePlanHeadlessAndStepEdges(t *testing.T) {
	cat := catalog.NewDefaultRegistry()
	runner := security.NewRunner(0)

	findStep := func(plan *domain.ExecutionPlan, id string) (domain.PlanStep, bool) {
		for _, s := range plan.Steps {
			if s.ResourceID == id {
				return s, true
			}
		}
		return domain.PlanStep{}, false
	}

	t.Run("headless omits x11 and desktop packages", func(t *testing.T) {
		home := t.TempDir()
		plat := platform.NewTestInfo(platform.TestInfoParams{
			OS:              "linux",
			Arch:            "amd64",
			PackageManagers: []string{"apt"},
			DisplayServer:   "headless",
			HomeDir:         home,
			StateDir:        filepath.Join(home, "state"),
		})
		disp := providers.NewDispatcher(plat, cat.Catalog(), runner, "")
		svc := app.NewInstallService(app.InstallDeps{Platform: plat, Catalog: cat, Provider: disp})

		plan, err := svc.PlanWithReinstall(context.Background(), "headless", nil, true)
		if err != nil {
			t.Fatalf("Plan(headless) failed: %v", err)
		}
		for _, unwanted := range []string{"xclip", "wmctrl", "xdotool", "ghostty", "wget"} {
			if _, ok := findStep(plan, unwanted); ok {
				t.Errorf("headless plan unexpectedly scheduled %q", unwanted)
			}
		}
	})

	t.Run("opensuse zypper developer links rust -> tree-sitter-cli edge while alpine does not", func(t *testing.T) {
		t.Setenv("PATH", t.TempDir())
		homeZypper := t.TempDir()
		platZypper := platform.NewTestInfo(platform.TestInfoParams{
			OS:              "linux",
			Arch:            "amd64",
			Distribution:    "opensuse-tumbleweed",
			PackageManagers: []string{"zypper"},
			HomeDir:         homeZypper,
			StateDir:        filepath.Join(homeZypper, "state"),
		})
		svcZypper := app.NewInstallService(app.InstallDeps{
			Platform: platZypper,
			Catalog:  cat,
			Provider: providers.NewDispatcher(platZypper, cat.Catalog(), runner, ""),
		})
		planZypper, err := svcZypper.PlanWithReinstall(context.Background(), "developer", nil, true)
		if err != nil {
			t.Fatalf("Plan(developer zypper) failed: %v", err)
		}
		rustStep, hasRust := findStep(planZypper, "rust")
		tsStep, hasTS := findStep(planZypper, "tree-sitter-cli")
		if !hasRust || !hasTS {
			t.Fatalf("expected rust and tree-sitter-cli in zypper developer plan (rust=%v, ts=%v)", hasRust, hasTS)
		}
		if !slices.Contains(tsStep.DependsOn, rustStep.ID) {
			t.Errorf("expected zypper tree-sitter-cli DependsOn to include rust step %d, got %v", rustStep.ID, tsStep.DependsOn)
		}

		homeApk := t.TempDir()
		platApk := platform.NewTestInfo(platform.TestInfoParams{
			OS:              "linux",
			Arch:            "amd64",
			Distribution:    "alpine",
			PackageManagers: []string{"apk"},
			HomeDir:         homeApk,
			StateDir:        filepath.Join(homeApk, "state"),
		})
		svcApk := app.NewInstallService(app.InstallDeps{
			Platform: platApk,
			Catalog:  cat,
			Provider: providers.NewDispatcher(platApk, cat.Catalog(), runner, ""),
		})
		planApk, err := svcApk.PlanWithReinstall(context.Background(), "developer", nil, true)
		if err != nil {
			t.Fatalf("Plan(developer apk) failed: %v", err)
		}
		rustApk, _ := findStep(planApk, "rust")
		tsApk, hasTSApk := findStep(planApk, "tree-sitter-cli")
		if !hasTSApk {
			t.Fatal("expected tree-sitter-cli in apk developer plan")
		}
		if slices.Contains(tsApk.DependsOn, rustApk.ID) {
			t.Errorf("expected apk native tree-sitter-cli NOT to depend on rust step %d, got %v", rustApk.ID, tsApk.DependsOn)
		}
	})

	t.Run("minimal linux orders curl before starship script", func(t *testing.T) {
		home := t.TempDir()
		plat := platform.NewTestInfo(platform.TestInfoParams{
			OS:              "linux",
			Arch:            "amd64",
			PackageManagers: []string{"apt"},
			HomeDir:         home,
			StateDir:        filepath.Join(home, "state"),
		})
		svc := app.NewInstallService(app.InstallDeps{
			Platform: plat,
			Catalog:  cat,
			Provider: providers.NewDispatcher(plat, cat.Catalog(), runner, ""),
		})
		plan, err := svc.PlanWithReinstall(context.Background(), "minimal", nil, true)
		if err != nil {
			t.Fatalf("Plan(minimal) failed: %v", err)
		}
		curlStep, hasCurl := findStep(plan, "curl")
		starStep, hasStar := findStep(plan, "starship")
		if !hasCurl || !hasStar {
			t.Fatalf("expected curl and starship in minimal plan (curl=%v, starship=%v)", hasCurl, hasStar)
		}
		if !slices.Contains(starStep.DependsOn, curlStep.ID) {
			t.Errorf("expected linux starship DependsOn to include curl step %d, got %v", curlStep.ID, starStep.DependsOn)
		}
	})

}
