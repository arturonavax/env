package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"runtime"
	"sort"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/spf13/cobra"

	"github.com/arturonavax/env/internal/ai/cli"
	aiconfig "github.com/arturonavax/env/internal/ai/config"
	"github.com/arturonavax/env/internal/ai/harness"
	"github.com/arturonavax/env/internal/app"
	"github.com/arturonavax/env/internal/domain"
	"github.com/arturonavax/env/internal/ops"
	"github.com/arturonavax/env/internal/theme"
	"github.com/arturonavax/env/internal/tui"
)

func metaCommands(appCtx *AppContext) []*cobra.Command {
	cmds := []*cobra.Command{
		newStatusCmd(appCtx),
		newDoctorCmd(appCtx),
		newAuditCmd(appCtx),
		newThemeCmd(appCtx),
		newConfigShowCmd(appCtx),
		newVersionCmd(appCtx),
	}
	for _, c := range cmds {
		c.GroupID = "meta"
	}
	return cmds
}

func aiCommands(appCtx *AppContext) []*cobra.Command {
	aiRoot := cli.NewCommand(cli.WithUse("ai"), cli.WithOptimizeFunc(aiOptimizeFunc(appCtx)))
	aiRoot.GroupID = "ai"

	dash := &cobra.Command{
		Use:   "dashboard",
		Short: "Launch the full-screen interactive dashboard",
		Args:  cobra.NoArgs,
		RunE:  func(cmd *cobra.Command, _ []string) error { return runDashboard(cmd, appCtx) },
	}
	wiz := &cobra.Command{
		Use:   "wizard",
		Short: "Interactive environment setup wizard (use --yes for autonomous mode)",
		Args:  cobra.NoArgs,
		RunE:  func(cmd *cobra.Command, _ []string) error { return runWizard(cmd, appCtx) },
	}
	dash.GroupID, wiz.GroupID = "ai", "ai"
	return []*cobra.Command{aiRoot, dash, wiz}
}

func runDashboard(cmd *cobra.Command, appCtx *AppContext) error {
	if !isTerminal(cmd) {
		return domain.NewExitError(domain.ExitUsageError, errors.New("the dashboard requires an interactive terminal; use 'anx status --format json' in pipelines"))
	}
	pal := theme.ByName(appCtx.Flags.Theme)
	p := tea.NewProgram(tui.NewModel(tui.Deps{
		Platform: appCtx.Platform, Catalog: appCtx.Catalog, Provider: appCtx.Provider,
		Sync: appCtx.SyncSvc, Doctor: appCtx.DoctorSvc, Audit: appCtx.AuditSvc, Hacks: appCtx.HacksSvc,
		Journal: appCtx.Journal, Theme: &pal,
	}), tea.WithContext(cmd.Context()))
	_, err := p.Run()
	return err
}

func runWizard(cmd *cobra.Command, appCtx *AppContext) error {
	if appCtx.Flags.Yes || !isTerminal(cmd) {
		fmt.Fprintln(cmd.OutOrStdout(), "ANX Setup (autonomous mode)")
		return runApply(cmd, appCtx, appCtx.Flags.Profile, nil)
	}
	p := tea.NewProgram(tui.NewWizardModel(appCtx.Platform, appCtx.Catalog, appCtx.InstallSvc), tea.WithContext(cmd.Context()))
	_, err := p.Run()
	return err
}

func newStatusCmd(appCtx *AppContext) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Inspect platform, packages and sync state",
		Args:  cobra.NoArgs,
		RunE:  func(cmd *cobra.Command, _ []string) error { return runStatus(cmd, appCtx) },
	}
}

type statusPayload struct {
	OS           string   `json:"os" yaml:"os"`
	Arch         string   `json:"arch" yaml:"arch"`
	Distribution string   `json:"distribution" yaml:"distribution"`
	IsWSL        bool     `json:"is_wsl" yaml:"is_wsl"`
	IsTermux     bool     `json:"is_termux" yaml:"is_termux"`
	HomeDir      string   `json:"home_dir" yaml:"home_dir"`
	StateDir     string   `json:"state_dir" yaml:"state_dir"`
	RepoRoot     string   `json:"repo_root" yaml:"repo_root"`
	Profile      string   `json:"profile" yaml:"profile"`
	PackageCount int      `json:"package_count" yaml:"package_count"`
	Installed    []string `json:"installed" yaml:"installed"`
	Missing      []string `json:"missing" yaml:"missing"`
}

func runStatus(cmd *cobra.Command, appCtx *AppContext) error {
	p := appCtx.Platform
	rows := collectPackages(cmd, appCtx, nil)
	payload := statusPayload{
		OS: p.OS(), Arch: p.Arch(), Distribution: p.Distribution(), IsWSL: p.IsWSL(), IsTermux: p.IsTermux(),
		HomeDir: p.HomeDir(), StateDir: p.StateDir(), RepoRoot: appCtx.RepoRoot,
		Profile: appCtx.Store.ActiveProfile(), PackageCount: len(rows),
		Installed: []string{}, Missing: []string{},
	}
	for _, r := range rows {
		if r.Installed {
			payload.Installed = append(payload.Installed, r.ID)
		} else {
			payload.Missing = append(payload.Missing, r.ID)
		}
	}
	return emit(cmd, appCtx, payload, func(w io.Writer) {
		st := styles(appCtx)
		fmt.Fprintln(w, st.Header.Render("ANX Environment Status"))
		fmt.Fprintf(w, "  OS: %s/%s (%s)  WSL: %v  Termux: %v\n", p.OS(), p.Arch(), p.Distribution(), p.IsWSL(), p.IsTermux())
		fmt.Fprintf(w, "  Home: %s\n  State: %s\n  Repo: %s\n  Profile: %s\n", p.HomeDir(), p.StateDir(), appCtx.RepoRoot, payload.Profile)
		fmt.Fprintf(w, "\n  Packages: %d installed / %d missing\n", len(payload.Installed), len(payload.Missing))
		printRows(w, appCtx, rows)
	})
}

func newDoctorCmd(appCtx *AppContext) *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Run system health checks and diagnostics",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			checks, err := appCtx.DoctorSvc.Diagnose(cmd.Context())
			if err != nil {
				return err
			}
			return emit(cmd, appCtx, checks, func(w io.Writer) {
				st := styles(appCtx)
				fmt.Fprintln(w, st.Header.Render("ANX Health Doctor"))
				for _, c := range checks {
					badge := st.Success.Render("[PASS]")
					switch c.Verdict {
					case app.VerdictWarn:
						badge = st.Warning.Render("[WARN]")
					case app.VerdictFail:
						badge = st.Error.Render("[FAIL]")
					}
					fmt.Fprintf(w, "  %s %-25s : %s\n", badge, c.Name, c.Message)
					if c.Remediation != "" {
						fmt.Fprintf(w, "         Remediation: %s\n", c.Remediation)
					}
				}
			})
		},
	}
}

func newAuditCmd(appCtx *AppContext) *cobra.Command {
	return &cobra.Command{
		Use:   "audit",
		Short: "Scan installed tools for known vulnerabilities (OSV)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if appCtx.Flags.Offline {
				return domain.NewExitError(domain.ExitPreflightFailed, errors.New("audit requires network access; remove --offline"))
			}
			findings, err := appCtx.AuditSvc.Audit(cmd.Context())
			if err != nil {
				return err
			}
			if err := emit(cmd, appCtx, findings, func(w io.Writer) {
				if len(findings) == 0 {
					fmt.Fprintln(w, "No known vulnerabilities detected on installed tools.")
				}
				for _, f := range findings {
					fmt.Fprintf(w, "  [%s] %s %s: %s\n         %s\n", f.Severity, f.ToolName, f.DetectedVersion, f.Summary, f.AdvisoryURL)
				}
			}); err != nil {
				return err
			}
			if len(findings) > 0 {
				return domain.NewExitError(domain.ExitConflictOrAudit, fmt.Errorf("%d vulnerabilities found", len(findings)))
			}
			return nil
		},
	}
}

func newThemeCmd(appCtx *AppContext) *cobra.Command {
	cmd := &cobra.Command{Use: "theme", Short: "List and persist UI themes"}
	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List supported themes",
		Args:  cobra.NoArgs,
		RunE: func(c *cobra.Command, _ []string) error {
			names := theme.Names()
			return emit(c, appCtx, names, func(w io.Writer) {
				for _, n := range names {
					fmt.Fprintf(w, "  • %s\n", n)
				}
			})
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:               "use <name>",
		Short:             "Persist the active theme",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: staticCompletion(theme.Names()...),
		RunE: func(c *cobra.Command, args []string) error {
			for _, n := range theme.Names() {
				if n == args[0] {
					return appCtx.Store.SetActiveTheme(n)
				}
			}
			return domain.NewExitError(domain.ExitUsageError, fmt.Errorf("unknown theme %q", args[0]))
		},
	})
	return cmd
}

func newConfigShowCmd(appCtx *AppContext) *cobra.Command {
	cmd := &cobra.Command{Use: "config", Short: "Show effective anx configuration and paths"}
	cmd.AddCommand(&cobra.Command{
		Use:   "show",
		Short: "Print the effective settings (flags > ANX_* env > config.yaml > defaults)",
		Args:  cobra.NoArgs,
		RunE: func(c *cobra.Command, _ []string) error {
			settings := map[string]any{
				"config_dir": appCtx.Platform.ConfigDir(), "state_dir": appCtx.Platform.StateDir(),
				"repo_root": appCtx.RepoRoot, "flags": appCtx.Flags,
				"manifests_overlay": appCtx.Platform.ConfigDir() + "/manifests.d",
				"hacks_overlay":     appCtx.Platform.ConfigDir() + "/hacks.d",
			}
			return emit(c, appCtx, settings, func(w io.Writer) {
				keys := make([]string, 0, len(settings))
				for k := range settings {
					keys = append(keys, k)
				}
				sort.Strings(keys)
				for _, k := range keys {
					fmt.Fprintf(w, "  %-18s %v\n", k, settings[k])
				}
			})
		},
	})
	return cmd
}

func newVersionCmd(appCtx *AppContext) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			info := map[string]string{
				"version": appCtx.Version, "commit": appCtx.Commit, "date": appCtx.Date,
				"go": runtime.Version(), "platform": runtime.GOOS + "/" + runtime.GOARCH,
			}
			return emit(cmd, appCtx, info, func(w io.Writer) {
				fmt.Fprintf(w, "anx version %s (commit %s, built %s, %s %s)\n",
					appCtx.Version, appCtx.Commit, appCtx.Date, runtime.Version(), info["platform"])
			})
		},
	}
}

func newCompletionCmd(root *cobra.Command) *cobra.Command {
	return &cobra.Command{
		Use:       "completion <bash|zsh|fish|powershell>",
		Short:     "Generate a shell completion script",
		ValidArgs: []string{"bash", "zsh", "fish", "powershell"},
		Args:      cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()
			switch args[0] {
			case "bash":
				return root.GenBashCompletionV2(out, true)
			case "zsh":
				return root.GenZshCompletion(out)
			case "fish":
				return root.GenFishCompletion(out, true)
			default:
				return root.GenPowerShellCompletionWithDesc(out)
			}
		},
	}
}

// aiPromptFunc builds an ops.PromptFunc backed by the aitools harness registry.
func aiPromptFunc(appCtx *AppContext) (ops.PromptFunc, error) {
	cfg, err := aiconfig.Load(aiconfig.DefaultConfigPath(appCtx.Platform.HomeDir()))
	if err != nil {
		return nil, err
	}
	reg := harness.NewRegistry(cfg)
	return func(ctx context.Context, prompt string) (string, error) {
		resp, _, err := reg.Query(ctx, "", harness.Request{Prompt: prompt, Mode: "chat"})
		return resp, err
	}, nil
}

func aiOptimizeFunc(appCtx *AppContext) cli.OptimizeFunc {
	return func(ctx context.Context, p cli.OptimizeParams) error {
		req := app.OptimizeRequest{AuditOnly: p.AuditOnly, SkipLynis: p.SkipLynis, Apply: p.Apply && !p.DryRun}
		fn, err := aiPromptFunc(appCtx)
		if err == nil {
			req.Prompt = fn
		}
		res, err := appCtx.OpsSvc.Optimize(ctx, req)
		if err != nil {
			return err
		}
		fmt.Fprintln(p.Out, res.Report.MarkdownSummary())
		if res.Script != "" {
			if p.OutputFile != "" {
				return os.WriteFile(p.OutputFile, []byte(res.Script), 0o700)
			}
			fmt.Fprintln(p.Out, res.Script)
			fmt.Fprintln(p.Out, res.Output)
		}
		return nil
	}
}

var _ = strings.TrimSpace
