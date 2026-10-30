package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/x/term"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"

	aicli "github.com/arturonavax/env/internal/ai/cli"
	"github.com/arturonavax/env/internal/app"
	"github.com/arturonavax/env/internal/catalog"
	"github.com/arturonavax/env/internal/hacks"
	"github.com/arturonavax/env/internal/journal"
	"github.com/arturonavax/env/internal/platform"
	"github.com/arturonavax/env/internal/providers"
	"github.com/arturonavax/env/internal/security"
	"github.com/arturonavax/env/internal/snapshot"
	"github.com/arturonavax/env/internal/state"
	"github.com/arturonavax/env/internal/theme"
	"github.com/arturonavax/env/internal/tools"
)

// BuildAppContext constructs the runtime AppContext (services, declarative catalog, state).
func BuildAppContext(version, commit, date string) (*AppContext, error) {
	plat := platform.Detect()

	v := viper.New()
	v.SetEnvPrefix("ANX")
	v.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	v.AutomaticEnv()
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(plat.ConfigDir())
	if err := v.ReadInConfig(); err != nil {
		var notFound viper.ConfigFileNotFoundError
		if !errors.As(err, &notFound) {
			return nil, fmt.Errorf("reading %s/config.yaml: %w", plat.ConfigDir(), err)
		}
	}

	store, err := state.OpenStore(plat.StateDir())
	if err != nil {
		return nil, err
	}

	repoRoot := resolveRepoRoot(v.GetString("repo"), plat.HomeDir(), store.RepoRoot())
	if isDir(filepath.Join(repoRoot, "files")) && isDir(filepath.Join(repoRoot, "manifests")) {
		_ = store.SetRepoRoot(repoRoot)
	}

	cat, err := catalog.NewRegistryWithOverlays(
		filepath.Join(repoRoot, "manifests"),
		filepath.Join(plat.ConfigDir(), "manifests.d"),
	)
	if err != nil {
		return nil, err
	}

	j, err := journal.NewFileJournal(plat.StateDir())
	if err != nil {
		return nil, err
	}
	snap, err := snapshot.NewManager(plat.StateDir())
	if err != nil {
		return nil, err
	}

	runner := security.NewRunner(60 * time.Second)
	installRunner := security.NewRunner(30 * time.Minute)
	prov := providers.NewDispatcher(plat, cat.Catalog(), installRunner, repoRoot)

	syncSvc := app.NewSyncService(plat, cat, snap, store, repoRoot)
	installSvc := app.NewInstallService(app.InstallDeps{
		Platform: plat, Catalog: cat, Journal: j, Snapshots: snap, Provider: prov, Store: store, Sync: syncSvc,
	})

	osvClient := security.NewOSVClient(
		10*time.Second,
		security.WithOSVCacheDir(filepath.Join(plat.CacheDir(), "osv"), 6*time.Hour),
	)

	repoHacksDir := filepath.Join(repoRoot, "manifests", "hacks")
	if _, err := os.Stat(repoHacksDir); err != nil {
		repoHacksDir = filepath.Join(repoRoot, "hacks")
	}

	hackReg, err := hacks.LoadRegistry(
		hacks.WithRepoHacksDir(repoHacksDir),
		hacks.WithUserHacksDir(filepath.Join(plat.ConfigDir(), "hacks.d")),
	)
	if err != nil {
		return nil, err
	}

	toolReg, err := tools.LoadRegistry(
		tools.WithRepoToolsDir(filepath.Join(repoRoot, "manifests")),
		tools.WithUserToolsDir(filepath.Join(plat.ConfigDir(), "tools.d")),
		tools.WithPlatform(plat),
	)
	if err != nil {
		return nil, err
	}
	toolRunner := tools.NewRunner(toolReg)
	toolUpdater := tools.NewUpdater(toolReg, func(ctx context.Context, packageID string) error {
		plan, err := installSvc.Plan(ctx, "all", []string{packageID})
		if err != nil {
			return err
		}
		return installSvc.Apply(ctx, plan)
	})
	toolsSvc := app.NewToolsService(toolReg, toolRunner, toolUpdater)

	appCtx := &AppContext{
		Platform:   plat,
		Catalog:    cat,
		Journal:    j,
		Snapshots:  snap,
		Store:      store,
		Runner:     runner,
		Provider:   prov,
		InstallSvc: installSvc,
		SyncSvc:    syncSvc,
		DoctorSvc:  app.NewDoctorService(plat, cat, app.WithDoctorStore(store)),
		AuditSvc:   app.NewAuditService(cat, prov, osvClient),
		HacksSvc:   app.NewHacksService(plat, hackReg, installRunner, snap, store, repoRoot),
		OpsSvc:     app.NewOpsService(plat, installRunner),
		ToolsSvc:   toolsSvc,
		RepoRoot:   repoRoot,
		Viper:      v,
		Version:    version,
		Commit:     commit,
		Date:       date,
	}

	toolRunner.RegisterBuiltin("ai", func(ctx context.Context, _ string, args []string, in io.Reader, out, errOut io.Writer) error {
		cmd := aicli.NewCommand(
			aicli.WithUse("ai"),
			aicli.WithOptimizeFunc(aiOptimizeFunc(appCtx)),
			aicli.WithPlatform(plat),
		)
		cmd.SetArgs(args)
		cmd.SetIn(in)
		cmd.SetOut(out)
		cmd.SetErr(errOut)
		return cmd.ExecuteContext(ctx)
	})

	return appCtx, nil
}

// resolveRepoRoot finds the env repository: configured path, walking up from cwd, persisted store root, or well-known user paths.
func resolveRepoRoot(configured, home, stored string) string {
	if configured != "" && isDir(filepath.Join(configured, "manifests")) {
		return configured
	}
	cwd, err := os.Getwd()
	if err == nil {
		for dir := cwd; ; dir = filepath.Dir(dir) {
			if isDir(filepath.Join(dir, "files")) && isDir(filepath.Join(dir, "manifests")) {
				return dir
			}
			if dir == filepath.Dir(dir) {
				break
			}
		}
	}
	if stored != "" && isDir(filepath.Join(stored, "files")) && isDir(filepath.Join(stored, "manifests")) {
		return stored
	}
	candidates := []string{
		filepath.Join(home, "repos", "github.com", "arturonavax", "env"),
		filepath.Join(home, ".local", "share", "anx", "repo"),
		filepath.Join(home, "env"),
		filepath.Join(home, ".dotfiles"),
	}
	for _, cand := range candidates {
		if isDir(filepath.Join(cand, "files")) && isDir(filepath.Join(cand, "manifests")) {
			return cand
		}
	}
	if cwd != "" {
		return cwd
	}
	return "."
}

func isDir(p string) bool {
	info, err := os.Stat(p)
	return err == nil && info.IsDir()
}

func isTerminal(cmd *cobra.Command) bool {
	in, inOK := cmd.InOrStdin().(*os.File)
	out, outOK := cmd.OutOrStdout().(*os.File)
	return inOK && outOK && term.IsTerminal(in.Fd()) && term.IsTerminal(out.Fd())
}

// emit renders v as JSON/YAML when requested, otherwise calls text.
func emit(cmd *cobra.Command, appCtx *AppContext, v any, text func(w io.Writer)) error {
	out := cmd.OutOrStdout()
	switch appCtx.Flags.Format {
	case "json":
		data, err := json.MarshalIndent(v, "", "  ")
		if err != nil {
			return err
		}
		fmt.Fprintln(out, string(data))
	case "yaml":
		data, err := yaml.Marshal(v)
		if err != nil {
			return err
		}
		fmt.Fprint(out, string(data))
	default:
		text(out)
	}
	return nil
}

func styles(appCtx *AppContext) *theme.Styles {
	if appCtx.Flags.NoColor {
		return theme.NewStyles(theme.NoColorPalette)
	}
	return theme.NewStyles(theme.ByName(appCtx.Flags.Theme))
}

func themeNames() []string { return theme.Names() }

func staticCompletion(values ...string) func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
		return values, cobra.ShellCompDirectiveNoFileComp
	}
}

func profileCompletion(appCtx *AppContext) func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
		names := []string{}
		for _, p := range appCtx.Catalog.ListProfiles() {
			names = append(names, p.Name)
		}
		return names, cobra.ShellCompDirectiveNoFileComp
	}
}

func packageCompletion(appCtx *AppContext) func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return func(_ *cobra.Command, _ []string, prefix string) ([]string, cobra.ShellCompDirective) {
		ids := []string{}
		for _, p := range appCtx.Catalog.ListPackages() {
			if strings.HasPrefix(p.ID, prefix) {
				ids = append(ids, p.ID+"\t"+p.Description)
			}
		}
		return ids, cobra.ShellCompDirectiveNoFileComp
	}
}
