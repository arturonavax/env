package app_test

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"strings"
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

func TestSyncPostSyncUsesPlatformHomeDir(t *testing.T) {
	fx := newFixture(t)
	testCfg := domain.ConfigurationResource{
		ID:       "test-post-sync-home",
		Source:   "files/test/run.sh",
		Target:   "~/.local/bin/test-run",
		Strategy: domain.StrategyReplace,
		FileMode: "0755",
		PostSyncCmds: [][]string{
			{"sh", "-c", `printf '%s' "$HOME" > "${HOME}/post_sync_home.txt"`},
		},
	}
	fx.cat.AddConfiguration(testCfg)

	svc := app.NewSyncService(fx.plat, fx.cat, fx.snap, fx.store, fx.repo)
	write(t, filepath.Join(fx.repo, "files/test/run.sh"), "#!/bin/sh\necho ok\n")

	if _, err := svc.SyncOne(context.Background(), "test-post-sync-home", app.SyncOptions{}); err != nil {
		t.Fatalf("SyncOne failed: %v", err)
	}
	targetInfo, err := os.Stat(filepath.Join(fx.plat.HomeDir(), ".local/bin/test-run"))
	if err != nil {
		t.Fatalf("expected synced target to exist: %v", err)
	}
	if targetInfo.Mode().Perm()&0o111 == 0 {
		t.Errorf("expected FileMode 0755 executable bits on target, got %v", targetInfo.Mode().Perm())
	}
	raw, err := os.ReadFile(filepath.Join(fx.plat.HomeDir(), "post_sync_home.txt"))
	if err != nil {
		t.Fatalf("expected post-sync hook to write inside platform HomeDir: %v", err)
	}
	if string(raw) != fx.plat.HomeDir() {
		t.Errorf("post-sync HOME = %q, want %q", string(raw), fx.plat.HomeDir())
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

	t.Run("darwin workstation schedules amethyst karabiner and pam-reattach while linux skips them", func(t *testing.T) {
		homeDarwin := t.TempDir()
		platDarwin := platform.NewTestInfo(platform.TestInfoParams{
			OS:              "darwin",
			Arch:            "arm64",
			PackageManagers: []string{"brew"},
			DisplayServer:   "quartz",
			HomeDir:         homeDarwin,
			StateDir:        filepath.Join(homeDarwin, "state"),
		})
		svcDarwin := app.NewInstallService(app.InstallDeps{
			Platform: platDarwin,
			Catalog:  cat,
			Provider: providers.NewDispatcher(platDarwin, cat.Catalog(), runner, ""),
		})
		planDarwin, err := svcDarwin.PlanWithReinstall(context.Background(), "workstation", nil, true)
		if err != nil {
			t.Fatalf("Plan(darwin workstation) failed: %v", err)
		}
		for _, id := range []string{"amethyst", "karabiner-elements", "pam-reattach"} {
			if _, ok := findStep(planDarwin, id); !ok {
				t.Errorf("darwin workstation plan missing %q", id)
			}
		}

		homeLinux := t.TempDir()
		platLinux := platform.NewTestInfo(platform.TestInfoParams{
			OS:              "linux",
			Arch:            "amd64",
			PackageManagers: []string{"apt"},
			DisplayServer:   "x11",
			HomeDir:         homeLinux,
			StateDir:        filepath.Join(homeLinux, "state"),
		})
		svcLinux := app.NewInstallService(app.InstallDeps{
			Platform: platLinux,
			Catalog:  cat,
			Provider: providers.NewDispatcher(platLinux, cat.Catalog(), runner, ""),
		})
		planLinux, err := svcLinux.PlanWithReinstall(context.Background(), "workstation", nil, true)
		if err != nil {
			t.Fatalf("Plan(linux workstation) failed: %v", err)
		}
		for _, id := range []string{"amethyst", "karabiner-elements", "pam-reattach"} {
			if _, ok := findStep(planLinux, id); ok {
				t.Errorf("linux workstation plan unexpectedly scheduled darwin-only %q", id)
			}
		}
	})
}

type stubSSHRunner struct {
	calls     [][]string
	lookPaths map[string]string
	env       map[string]string
	handler   func(name string, args ...string) (int, string, string, error)
}

func (r *stubSSHRunner) Run(_ context.Context, name string, args ...string) (int, string, string, error) {
	call := append([]string{name}, args...)
	r.calls = append(r.calls, call)
	if r.handler != nil {
		return r.handler(name, args...)
	}
	return 0, "", "", nil
}

func (r *stubSSHRunner) LookPath(name string) (string, error) {
	if p, ok := r.lookPaths[name]; ok {
		return p, nil
	}
	return "", os.ErrNotExist
}

func (r *stubSSHRunner) Getenv(k string) string {
	return r.env[k]
}

func TestSyncServiceSSHAgentLinuxAndDarwin(t *testing.T) {
	repoRoot := filepath.Join("..", "..")

	t.Run("enforces 0700/0600 on ssh-config, preserves user hosts, respects dry-run and capture guard, and enables systemd user agent", func(t *testing.T) {
		home := t.TempDir()
		plat := platform.NewTestInfo(platform.TestInfoParams{
			OS:           "linux",
			Arch:         "amd64",
			CurrentShell: "zsh",
			HomeDir:  home,
			StateDir: filepath.Join(home, "state"),
		})
		snap, err := snapshot.NewManager(plat.StateDir())
		if err != nil {
			t.Fatal(err)
		}
		store, err := state.OpenStore(plat.StateDir())
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = store.Close() })

		sshDir := filepath.Join(home, ".ssh")
		if err := os.MkdirAll(sshDir, 0o755); err != nil {
			t.Fatal(err)
		}
		sshCfgPath := filepath.Join(sshDir, "config")
		existingUserHost := "Host prod-bastion\n    HostName 10.0.0.10\n    User deploy\n"
		if err := os.WriteFile(sshCfgPath, []byte(existingUserHost), 0o644); err != nil {
			t.Fatal(err)
		}

		runner := &stubSSHRunner{
			lookPaths: map[string]string{"systemctl": "/usr/bin/systemctl"},
			env:       map[string]string{"XDG_RUNTIME_DIR": "/run/user/1000"},
		}
		cat := catalog.NewDefaultRegistry()
		svc := app.NewSyncService(plat, cat, snap, store, repoRoot, app.WithSyncSSHRunner(runner))

		// 1. Dry-run leaves permissions and contents untouched.
		dryRes, err := svc.SyncOne(context.Background(), "ssh-config", app.SyncOptions{DryRun: true})
		if err != nil {
			t.Fatalf("dry-run SyncOne(ssh-config) failed: %v", err)
		}
		if dryRes.Action == "unchanged" {
			t.Errorf("expected dry-run over missing marker block not to report unchanged")
		}
		info, _ := os.Stat(sshCfgPath)
		if info.Mode().Perm() != 0o644 {
			t.Errorf("dry-run mutated file permissions to %04o", info.Mode().Perm())
		}

		// 2. Live sync creates backup, merges Host *, preserves user host, tightens 0700 & 0600.
		res, err := svc.SyncOne(context.Background(), "ssh-config", app.SyncOptions{})
		if err != nil {
			t.Fatalf("SyncOne(ssh-config) failed: %v", err)
		}
		if res.SnapshotID == "" {
			t.Errorf("expected backup snapshot before updating ~/.ssh/config")
		}
		dirInfo, _ := os.Stat(sshDir)
		cfgInfo, _ := os.Stat(sshCfgPath)
		if dirInfo.Mode().Perm() != 0o700 {
			t.Errorf("expected ~/.ssh mode 0700, got %04o", dirInfo.Mode().Perm())
		}
		if cfgInfo.Mode().Perm() != 0o600 {
			t.Errorf("expected ~/.ssh/config mode 0600, got %04o", cfgInfo.Mode().Perm())
		}
		mergedBytes, _ := os.ReadFile(sshCfgPath)
		merged := string(mergedBytes)
		if !strings.Contains(merged, "Host prod-bastion") || !strings.Contains(merged, "AddKeysToAgent yes") || !strings.Contains(merged, "UseKeychain yes") {
			t.Errorf("unexpected merged ~/.ssh/config:\n%s", merged)
		}

		// Capture skips ssh-config because skip_capture: true.
		cfgObj, _ := cat.GetConfiguration("ssh-config")
		if !cfgObj.IsHostSpecific() {
			t.Errorf("expected ssh-config.IsHostSpecific() == true to block capture leakage")
		}

		// 3. Diff detects permission drift and re-syncing tightens modes back to 0700/0600 even when file text matches.
		_ = os.Chmod(sshDir, 0o755)
		_ = os.Chmod(sshCfgPath, 0o644)
		diffs, err := svc.Diff(context.Background(), "ssh-config")
		if err != nil {
			t.Fatalf("Diff(ssh-config) failed: %v", err)
		}
		foundPermDrift := false
		for _, d := range diffs {
			if d.ConfigID == "ssh-config" && d.State == "drift" && strings.Contains(d.Summary, "permissions") {
				foundPermDrift = true
			}
		}
		if !foundPermDrift {
			t.Errorf("expected Diff to detect permission drift on ssh-config, got %+v", diffs)
		}
		res2, err := svc.SyncOne(context.Background(), "ssh-config", app.SyncOptions{})
		if err != nil {
			t.Fatalf("re-sync ssh-config failed: %v", err)
		}
		if res2.Action != "updated" {
			t.Errorf("expected Action=updated when fixing drifted perms, got %q", res2.Action)
		}
		dirInfo, _ = os.Stat(sshDir)
		cfgInfo, _ = os.Stat(sshCfgPath)
		if dirInfo.Mode().Perm() != 0o700 || cfgInfo.Mode().Perm() != 0o600 {
			t.Errorf("expected re-tightened modes 0700/0600, got %04o/%04o", dirInfo.Mode().Perm(), cfgInfo.Mode().Perm())
		}

		// 4. ssh-agent-service invokes daemon-reload & enable --now when systemd --user session is active.
		runner.calls = nil
		svcRes, err := svc.SyncOne(context.Background(), "ssh-agent-service", app.SyncOptions{})
		if err != nil {
			t.Fatalf("SyncOne(ssh-agent-service) failed: %v", err)
		}
		var sawReload, sawEnable bool
		for _, c := range runner.calls {
			joined := strings.Join(c, " ")
			if strings.Contains(joined, "systemctl --user daemon-reload") {
				sawReload = true
			}
			if strings.Contains(joined, "systemctl --user enable --now ssh-agent.service") {
				sawEnable = true
			}
		}
		if !sawReload || !sawEnable {
			t.Errorf("expected daemon-reload & enable --now calls (detail=%q), got calls: %v", svcRes.Detail, runner.calls)
		}
	})

	t.Run("linux degrades safely without systemd user session", func(t *testing.T) {
		home := t.TempDir()
		plat := platform.NewTestInfo(platform.TestInfoParams{
			OS:           "linux",
			CurrentShell: "zsh",
			HomeDir:  home,
			StateDir: filepath.Join(home, "state"),
		})
		runner := &stubSSHRunner{
			lookPaths: map[string]string{"systemctl": "/usr/bin/systemctl"},
			handler: func(name string, args ...string) (int, string, string, error) {
				if name == "systemctl" && slices.Contains(args, "show-environment") {
					return 1, "", "Failed to connect to bus: No medium found", nil
				}
				return 0, "", "", nil
			},
		}
		svc := app.NewSyncService(plat, catalog.NewDefaultRegistry(), nil, nil, repoRoot, app.WithSyncSSHRunner(runner))
		res, err := svc.SyncOne(context.Background(), "ssh-agent-service", app.SyncOptions{})
		if err != nil {
			t.Fatalf("expected safe degradation without systemd --user bus, got err=%v", err)
		}
		if !strings.Contains(strings.ToLower(res.Detail), "systemd --user") {
			t.Errorf("expected informative degradation detail, got %q", res.Detail)
		}
	})

	t.Run("darwin respects launchd and emits non-blocking keychain hint on exit code 1", func(t *testing.T) {
		home := t.TempDir()
		plat := platform.NewTestInfo(platform.TestInfoParams{
			OS:           "darwin",
			Arch:         "arm64",
			CurrentShell: "zsh",
			HomeDir:  home,
			StateDir: filepath.Join(home, "state"),
		})
		_ = os.MkdirAll(filepath.Join(home, ".ssh"), 0o700)
		_ = os.WriteFile(filepath.Join(home, ".ssh", "id_ed25519"), []byte("private"), 0o600)

		runner := &stubSSHRunner{
			lookPaths: map[string]string{"ssh-add": "/usr/bin/ssh-add"},
			env:       map[string]string{"SSH_AUTH_SOCK": "/private/tmp/com.apple.launchd.123/Listeners"},
			handler: func(name string, args ...string) (int, string, string, error) {
				if name == "ssh-add" && len(args) == 1 && args[0] == "-l" {
					return 1, "The agent has no identities.", "", nil
				}
				return 0, "", "", nil
			},
		}
		svc := app.NewSyncService(plat, catalog.NewDefaultRegistry(), nil, nil, repoRoot, app.WithSyncSSHRunner(runner))
		res, err := svc.SyncOne(context.Background(), "ssh-config", app.SyncOptions{})
		if err != nil {
			t.Fatalf("SyncOne(ssh-config) on darwin failed: %v", err)
		}
		if !strings.Contains(res.Detail, "ssh-add --apple-use-keychain") {
			t.Errorf("expected non-blocking keychain hint in Detail, got %q", res.Detail)
		}
		for _, c := range runner.calls {
			if c[0] == "systemctl" {
				t.Errorf("darwin must never invoke systemctl: %v", c)
			}
		}

		// InstallService.Apply forwards the keychain hint through StepSuccess ProgressEvent.OutputLine.
		instSvc := app.NewInstallService(app.InstallDeps{
			Platform: plat,
			Catalog:  catalog.NewDefaultRegistry(),
			Sync:     svc,
		})
		var sawHint bool
		instSvc.SetOnProgress(func(ev app.ProgressEvent) {
			if ev.ResourceID == "ssh-config" && ev.Status == domain.StepSuccess && strings.Contains(ev.OutputLine, "ssh-add --apple-use-keychain") {
				sawHint = true
			}
		})
		if err := instSvc.Apply(context.Background(), &domain.ExecutionPlan{
			ID:      "plan-darwin-ssh",
			Profile: "minimal",
			Steps: []domain.PlanStep{
				{ID: 1, ResourceID: "ssh-config", Action: domain.ActionSync, Description: "Synchronize OpenSSH client config", Locks: []string{"file:" + filepath.Join(home, ".ssh", "config")}},
			},
		}); err != nil {
			t.Fatalf("InstallService.Apply(ssh-config) failed: %v", err)
		}
		if !sawHint {
			t.Errorf("expected InstallService.Apply to emit ProgressEvent.OutputLine with keychain hint")
		}
	})

	t.Run("all profiles schedule OS-appropriate SSH agent configs cleanly", func(t *testing.T) {
		cat := catalog.NewDefaultRegistry()
		runner := security.NewRunner(0)
		for _, goos := range []string{"linux", "darwin"} {
			for _, prof := range cat.ListProfiles() {
				home := t.TempDir()
				plat := platform.NewTestInfo(platform.TestInfoParams{
					OS:       goos,
					Arch:     "amd64",
					HomeDir:  home,
					StateDir: filepath.Join(home, "state"),
				})
				syncSvc := app.NewSyncService(plat, cat, nil, nil, repoRoot)
				instSvc := app.NewInstallService(app.InstallDeps{
					Platform: plat,
					Catalog:  cat,
					Provider: providers.NewDispatcher(plat, cat.Catalog(), runner, ""),
					Sync:     syncSvc,
				})
				plan, err := instSvc.PlanWithReinstall(context.Background(), prof.Name, nil, true)
				if err != nil {
					t.Fatalf("Plan(%s/%s): %v", goos, prof.Name, err)
				}
				hasStep := func(id string) bool {
					for _, st := range plan.Steps {
						if st.ResourceID == id {
							return true
						}
					}
					return false
				}
				if !hasStep("ssh-config") || !hasStep("ssh-agent-zshenv") || !hasStep("ssh-agent-profile") {
					t.Errorf("%s profile %q missing shared ssh configs", goos, prof.Name)
				}
				if goos == "linux" && (!hasStep("ssh-agent-service") || !hasStep("ssh-agent-env")) {
					t.Errorf("%s profile %q missing systemd/environment.d configs", goos, prof.Name)
				}
				if goos == "darwin" && (hasStep("ssh-agent-service") || hasStep("ssh-agent-env")) {
					t.Errorf("%s profile %q should not schedule linux systemd/environment.d configs", goos, prof.Name)
				}
			}
		}
	})
}
