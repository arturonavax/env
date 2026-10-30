package app_test

import (
	"context"
	"os"
	"path/filepath"
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
		if s.Action == domain.ActionInstall && !s.IsExclusive {
			t.Fatalf("install step %s must be exclusive", s.ResourceID)
		}
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
