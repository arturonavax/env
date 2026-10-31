package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"

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

// GlobalFlagState holds parsed persistent flags.
type GlobalFlagState struct {
	Interactive  bool
	Yes          bool
	DryRun       bool
	Format       string
	Quiet        bool
	Verbose      bool
	NoColor      bool
	Theme        string
	Offline      bool
	Profile      string
	IncludeHacks bool
	ScriptMode   bool
}

// AppContext holds shared domain services injected into CLI commands.
type AppContext struct {
	Platform   *platform.Info
	Catalog    *catalog.Registry
	Journal    *journal.FileJournal
	Snapshots  *snapshot.Manager
	Store      *state.Store
	Runner     *security.Runner
	Provider   providers.Provider
	InstallSvc *app.InstallService
	SyncSvc    *app.SyncService
	DoctorSvc  *app.DoctorService
	AuditSvc   *app.AuditService
	HacksSvc   *app.HacksService
	OpsSvc     *app.OpsService
	ToolsSvc     *app.ToolsService
	ControlPlane *app.ControlPlaneService
	RepoRoot     string
	Viper      *viper.Viper
	Flags      GlobalFlagState
	Version    string
	Commit     string
	Date       string
}

// NewRootCommand creates the top-level anx Cobra command tree.
func NewRootCommand(appCtx *AppContext) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:           "anx",
		Short:         "Unified declarative environment, dotfiles and personal tooling manager",
		Version:       appCtx.Version,
		SilenceUsage:  true,
		SilenceErrors: true,
		Args:          cobra.NoArgs,
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			return applyConfigLayering(cmd, appCtx)
		},
		RunE: func(cmd *cobra.Command, _ []string) error {
			if appCtx.Flags.Yes {
				return runApply(cmd, appCtx, appCtx.Flags.Profile, nil)
			}
			// TTY guard: never launch a full-screen program into a pipe or when machine output was requested.
			if appCtx.Flags.Format != "text" || !isTerminal(cmd) {
				return runStatus(cmd, appCtx)
			}
			return runDashboard(cmd, appCtx)
		},
	}
	rootCmd.SetFlagErrorFunc(func(_ *cobra.Command, err error) error {
		return domain.NewExitError(domain.ExitUsageError, err)
	})

	pf := rootCmd.PersistentFlags()
	pf.BoolVarP(&appCtx.Flags.Interactive, "interactive", "i", false, "Force interactive prompts")
	pf.BoolVarP(&appCtx.Flags.Yes, "yes", "y", false, "Automatically answer yes to confirmations")
	pf.BoolVar(&appCtx.Flags.DryRun, "dry-run", false, "Preview actions without mutating the system")
	pf.StringVar(&appCtx.Flags.Format, "format", "text", "Output format (text|json|yaml)")
	pf.BoolVarP(&appCtx.Flags.Quiet, "quiet", "q", false, "Suppress non-essential output")
	pf.BoolVarP(&appCtx.Flags.Verbose, "verbose", "v", false, "Verbose execution logging")
	pf.BoolVar(&appCtx.Flags.NoColor, "no-color", false, "Disable ANSI color rendering")
	pf.StringVar(&appCtx.Flags.Theme, "theme", "tokyo-night", "UI theme (see 'anx theme list')")
	pf.BoolVar(&appCtx.Flags.Offline, "offline", false, "Disable network queries")
	pf.StringVar(&appCtx.Flags.Profile, "profile", "developer", "Target profile name")
	pf.BoolVar(&appCtx.Flags.IncludeHacks, "include-hacks", false, "Include matched hardware workstation hacks during apply")
	pf.BoolVar(&appCtx.Flags.ScriptMode, "script", false, "Output shell script preview of planned actions")
	_ = rootCmd.RegisterFlagCompletionFunc("format", staticCompletion("text", "json", "yaml"))
	_ = rootCmd.RegisterFlagCompletionFunc("profile", profileCompletion(appCtx))
	_ = rootCmd.RegisterFlagCompletionFunc("theme", staticCompletion(themeNames()...))

	rootCmd.AddGroup(
		&cobra.Group{ID: "packages", Title: "Packages & Profiles:"},
		&cobra.Group{ID: "config", Title: "Dotfiles & Backups:"},
		&cobra.Group{ID: "system", Title: "System, Hacks & Maintenance:"},
		&cobra.Group{ID: "tools", Title: "Personal Tools & AI Suite:"},
		&cobra.Group{ID: "meta", Title: "Diagnostics & Meta:"},
	)

	rootCmd.AddCommand(packageCommands(appCtx)...)
	rootCmd.AddCommand(configCommands(appCtx)...)
	rootCmd.AddCommand(systemCommands(appCtx)...)
	rootCmd.AddCommand(toolsCommands(appCtx)...)
	rootCmd.AddCommand(interactiveCommands(appCtx)...)
	rootCmd.AddCommand(metaCommands(appCtx)...)
	rootCmd.AddCommand(newLsCmd(appCtx))
	rootCmd.AddCommand(newWhatCmd(appCtx))
	rootCmd.AddCommand(newSelfCmd(appCtx))
	rootCmd.AddCommand(newCompletionCmd(rootCmd, appCtx))
	rootCmd.AddCommand(newDocCmd(rootCmd, appCtx))

	return rootCmd
}

// applyConfigLayering resolves flag > env (ANX_*) > config.yaml > default for unchanged persistent flags.
func applyConfigLayering(cmd *cobra.Command, appCtx *AppContext) error {
	v := appCtx.Viper
	if v == nil {
		return nil
	}
	root := cmd.Root()
	root.PersistentFlags().VisitAll(func(f *pflag.Flag) {
		if f.Changed {
			return
		}
		key := strings.ReplaceAll(f.Name, "-", "_")
		if !v.IsSet(key) {
			return
		}
		_ = root.PersistentFlags().Set(f.Name, v.GetString(key))
	})

	switch appCtx.Flags.Format {
	case "text", "json", "yaml":
	default:
		return domain.NewExitError(
			domain.ExitUsageError,
			fmt.Errorf("invalid --format %q (expected text, json or yaml)", appCtx.Flags.Format),
		)
	}
	if os.Getenv("NO_COLOR") != "" {
		appCtx.Flags.NoColor = true
	}
	return nil
}

func isLazyCommand(args []string) bool {
	if len(args) == 0 {
		return false
	}
	first := args[0]
	if first == "version" || first == "completion" || first == "--version" || first == "-v" || first == "help" {
		return true
	}
	if len(args) == 1 && (first == "--help" || first == "-h") {
		return true
	}
	return false
}

// ExecuteCLI runs the root Cobra command with signal handling and maps errors to spec exit codes.
func ExecuteCLI(args []string, out, errOut io.Writer, version, commit, date string) int {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	var appCtx *AppContext
	var err error
	if isLazyCommand(args) {
		appCtx, err = BuildLazyAppContext(version, commit, date)
	} else {
		appCtx, err = BuildAppContext(version, commit, date)
	}
	if err != nil {
		fmt.Fprintf(errOut, "Initialization error: %v\n", err)
		return domain.ExitPreflightFailed
	}

	rootCmd := NewRootCommand(appCtx)
	rootCmd.SetArgs(args)
	rootCmd.SetOut(out)
	rootCmd.SetErr(errOut)

	if err := rootCmd.ExecuteContext(ctx); err != nil {
		if errors.Is(err, context.Canceled) || ctx.Err() != nil {
			fmt.Fprintln(errOut, "\nOperation cancelled by signal.")
			return domain.ExitInterrupted
		}
		fmt.Fprintf(errOut, "Error: %v\n", err)
		return exitCodeFor(err)
	}
	return domain.ExitSuccess
}

func exitCodeFor(err error) int {
	var exitErr *domain.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.Code
	}
	msg := err.Error()
	for _, marker := range []string{"unknown command", "unknown flag", "unknown shorthand", "accepts ", "requires at least", "invalid argument"} {
		if strings.Contains(msg, marker) {
			return domain.ExitUsageError
		}
	}
	return domain.ResolveExitCode(err)
}
