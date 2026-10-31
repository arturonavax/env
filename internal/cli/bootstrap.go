package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/charmbracelet/x/term"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	aicli "github.com/arturonavax/env/internal/ai/cli"
	"github.com/arturonavax/env/internal/bootstrap"
	"github.com/arturonavax/env/internal/theme"
)

// BuildLazyAppContext constructs a lightweight AppContext for read-only or meta commands
// without touching the filesystem, initializing state, journals, or requiring a repository root.
func BuildLazyAppContext(version, commit, date string) (*AppContext, error) {
	container, err := bootstrap.NewLazyContainer(bootstrap.WithVersion(version, commit, date))
	if err != nil {
		return nil, err
	}

	return &AppContext{
		Platform:     container.Platform,
		Catalog:      container.Catalog,
		SecuritySeal: container.SecuritySeal,
		Viper:        container.Viper,
		Version:      version,
		Commit:       commit,
		Date:         date,
	}, nil
}

// BuildAppContext constructs the runtime AppContext by delegating to the reusable bootstrap container.
func BuildAppContext(version, commit, date string) (*AppContext, error) {
	container, err := bootstrap.NewContainer(bootstrap.WithVersion(version, commit, date))
	if err != nil {
		return nil, err
	}

	appCtx := &AppContext{
		Platform:     container.Platform,
		Catalog:      container.Catalog,
		Journal:      container.Journal,
		Snapshots:    container.Snapshots,
		Store:        container.Store,
		Runner:       container.Runner,
		Provider:     container.Provider,
		InstallSvc:   container.InstallSvc,
		SyncSvc:      container.SyncSvc,
		DoctorSvc:    container.DoctorSvc,
		AuditSvc:     container.AuditSvc,
		HacksSvc:     container.HacksSvc,
		OpsSvc:       container.OpsSvc,
		ToolsSvc:     container.ToolsSvc,
		ControlPlane: container.ControlPlane,
		DestroyerSvc: container.DestroyerSvc,
		SecuritySeal: container.SecuritySeal,
		RepoRoot:     container.RepoRoot,
		Viper:        container.Viper,
		Version:      version,
		Commit:       commit,
		Date:         date,
	}

	container.ToolRunner.RegisterBuiltin("ai", func(ctx context.Context, _ string, args []string, in io.Reader, out, errOut io.Writer) error {
		cmd := aicli.NewCommand(
			aicli.WithUse("ai"),
			aicli.WithOptimizeFunc(aiOptimizeFunc(appCtx)),
			aicli.WithPlatform(container.Platform),
		)
		cmd.SetArgs(args)
		cmd.SetIn(in)
		cmd.SetOut(out)
		cmd.SetErr(errOut)
		return cmd.ExecuteContext(ctx)
	})

	return appCtx, nil
}

// resolveRepoRoot finds the env repository by delegating to bootstrap.ResolveRepoRoot.
func resolveRepoRoot(configured, home, stored string) string {
	return bootstrap.ResolveRepoRoot(configured, home, stored)
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
