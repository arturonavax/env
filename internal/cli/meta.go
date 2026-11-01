package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
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
		newExplainCmd(appCtx),
		newDoctorCmd(appCtx),
		newAuditCmd(appCtx),
		newThemeCmd(appCtx),
		newConfigShowCmd(appCtx),
		newAliasCmd(appCtx),
		newVersionCmd(appCtx),
	}
	for _, c := range cmds {
		c.GroupID = "meta"
	}
	return cmds
}

func interactiveCommands(appCtx *AppContext) []*cobra.Command {
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
	dash.GroupID, wiz.GroupID = "meta", "meta"
	return []*cobra.Command{dash, wiz}
}

func runDashboard(cmd *cobra.Command, appCtx *AppContext) error {
	if !isTerminal(cmd) {
		return domain.NewExitError(domain.ExitUsageError, errors.New("the dashboard requires an interactive terminal; use 'anx status --format json' in pipelines"))
	}
	pal := theme.ByName(appCtx.Flags.Theme)
	p := tea.NewProgram(tui.NewModel(tui.Deps{
		Platform: appCtx.Platform,
		Catalog:  appCtx.Catalog,
		Query:    appCtx.ControlPlane,
		Doctor:   appCtx.DoctorSvc,
		Audit:    appCtx.AuditSvc,
		Hacks:    appCtx.HacksSvc,
		Sync:     appCtx.SyncSvc,
		Store:    appCtx.Store,
		Theme:    &pal,
	}), tea.WithContext(cmd.Context()))
	_, err := p.Run()
	return err
}

func runWizard(cmd *cobra.Command, appCtx *AppContext) error {
	if appCtx.Flags.Yes || !isTerminal(cmd) {
		fmt.Fprintln(cmd.OutOrStdout(), "ANX Setup (autonomous mode)")
		return runApply(cmd, appCtx, appCtx.Flags.Profile, nil)
	}
	var matchedHacks []domain.HackProfile
	if appCtx.HacksSvc != nil {
		_, matchedHacks = appCtx.HacksSvc.Detect(cmd.Context())
	}
	var diffs []app.ConfigDiff
	if appCtx.SyncSvc != nil {
		diffs, _ = appCtx.SyncSvc.Diff(cmd.Context(), "all")
	}
	p := tea.NewProgram(tui.NewWizardModel(
		appCtx.Platform,
		appCtx.Catalog,
		appCtx.InstallSvc,
		tui.WithWizardMatchedHacks(matchedHacks),
		tui.WithWizardIncludeHacks(appCtx.Flags.IncludeHacks),
		tui.WithWizardConfigDiffs(diffs),
		tui.WithWizardSyncService(appCtx.SyncSvc),
		tui.WithWizardHacksService(appCtx.HacksSvc),
		tui.WithWizardDestroyerSvc(appCtx.DestroyerSvc),
		tui.WithWizardDestroyerMode(appCtx.Flags.Destroyer),
		tui.WithWizardSecuritySeal(appCtx.SecuritySeal),
		tui.WithWizardStore(appCtx.Store),
	), tea.WithContext(cmd.Context()))
	finalM, err := p.Run()
	if err == nil && appCtx.Store != nil {
		if wm, ok := finalM.(tui.WizardModel); ok && wm.IsCompleted() && wm.ChosenProfile() != "" {
			_ = appCtx.Store.SetActiveProfile(wm.ChosenProfile())
		}
	}
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
	SecuritySeal string   `json:"security_seal,omitempty" yaml:"security_seal,omitempty"`
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
	if appCtx.SecuritySeal != nil {
		payload.SecuritySeal = appCtx.SecuritySeal.ShortDigest()
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
		if appCtx.SecuritySeal != nil {
			sealBadge := st.Success.Render("[VERIFIED ✓]")
			if appCtx.SecuritySeal.InsecureOrigin {
				sealBadge = st.Warning.Render(fmt.Sprintf("[INSECURE: %s]", appCtx.SecuritySeal.InsecureReason))
			}
			fmt.Fprintf(w, "  Security Seal: %s %s (Manifests: %s, Dotfiles: %s)\n",
				appCtx.SecuritySeal.ShortDigest(), sealBadge,
				appCtx.SecuritySeal.ShortManifestDigest(), appCtx.SecuritySeal.ShortDotfilesDigest())
		}
		fmt.Fprintf(w, "\n  Packages: %d installed / %d missing\n", len(payload.Installed), len(payload.Missing))
		printRows(w, appCtx, rows)
	})
}

func newExplainCmd(appCtx *AppContext) *cobra.Command {
	return &cobra.Command{
		Use:   "explain [kind] <resource-id>",
		Short: "Explain origin, winning policy, conditions, and provenance of a resource or capability",
		Long: `Inspect and explain why a package, dotfile configuration, personal tool, hardware hack, or capability
was selected, including its origin manifest, active profile, dependencies, and winning policy.`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			st := styles(appCtx)
			rawID := args[0]
			if len(args) == 2 {
				rawID = args[1]
			}

			if appCtx.ControlPlane == nil {
				return errors.New("control plane service not available")
			}

			prov, err := appCtx.ControlPlane.Explain(ctx, domain.ResourceID(rawID))
			if err != nil {
				return domain.NewExitError(domain.ExitUsageError, err)
			}

			return emit(cmd, appCtx, prov, func(w io.Writer) {
				fmt.Fprintln(w, st.Header.Render(fmt.Sprintf("Resource Provenance: %s", prov.ResourceID)))
				fmt.Fprintf(w, "  • Kind:           %s\n", prov.Kind)
				fmt.Fprintf(w, "  • Origin:         %s\n", prov.Origin)
				if prov.ActiveProfile != "" {
					fmt.Fprintf(w, "  • Active Profile: %s\n", prov.ActiveProfile)
				}
				if len(prov.Conditions) > 0 {
					fmt.Fprintf(w, "  • Conditions:     %s\n", strings.Join(prov.Conditions, ", "))
				}
				if prov.WinningPolicy != "" {
					fmt.Fprintf(w, "  • Winning Policy: %s\n", st.Highlight.Render(prov.WinningPolicy))
				}
				if len(prov.Dependencies) > 0 {
					deps := make([]string, len(prov.Dependencies))
					for i, d := range prov.Dependencies {
						deps[i] = string(d)
					}
					fmt.Fprintf(w, "  • Dependencies:   %s\n", strings.Join(deps, ", "))
				}
				if prov.Reason != "" {
					fmt.Fprintf(w, "  • Explanation:    %s\n", prov.Reason)
				}
			})
		},
	}
}

func newDoctorCmd(appCtx *AppContext) *cobra.Command {
	var fix bool
	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Run system health checks and diagnostics",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			checks, err := appCtx.DoctorSvc.Diagnose(cmd.Context())
			if err != nil {
				return err
			}
			err = emit(cmd, appCtx, checks, func(w io.Writer) {
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
			if err != nil {
				return err
			}

			if fix {
				missingPkgs := make([]string, 0, len(checks))
				for _, c := range checks {
					if (c.Verdict == app.VerdictWarn || c.Verdict == app.VerdictFail) && c.PackageID != "" {
						missingPkgs = append(missingPkgs, c.PackageID)
					}
				}
				if len(missingPkgs) > 0 {
					fmt.Fprintf(cmd.OutOrStdout(), "\n[Remediation] Installing missing dependencies: %s\n", strings.Join(missingPkgs, ", "))
					if err := runApply(cmd, appCtx, "", missingPkgs); err != nil {
						return fmt.Errorf("failed to install missing dependencies: %w", err)
					}
				}
			} else {
				hasRemediations := false
				for _, c := range checks {
					if (c.Verdict == app.VerdictWarn || c.Verdict == app.VerdictFail) && c.PackageID != "" {
						hasRemediations = true
						break
					}
				}
				if hasRemediations && isTerminal(cmd) {
					fmt.Fprintln(cmd.OutOrStdout(), "\nTip: Run 'anx doctor --fix' to automatically install missing dependencies.")
				}
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&fix, "fix", false, "Automatically install missing dependencies and remediate diagnostic warnings")
	return cmd
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
	cmd.AddCommand(newAliasCmd(appCtx))
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

func newAliasCmd(appCtx *AppContext) *cobra.Command {
	return &cobra.Command{
		Use:   "alias <name>",
		Short: "Create a custom binary alias/symlink for anx (e.g. for forks or personal naming)",
		Long: `Create a custom command alias pointing to the anx binary in ~/.local/bin/<name>.
This allows forks or users desiring custom command names (e.g. 'myenv', 'envcli', 'dot')
to execute anx transparently under their preferred alias.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := strings.TrimSpace(args[0])
			if name == "" || name == "anx" {
				return domain.NewExitError(domain.ExitUsageError, fmt.Errorf("invalid alias name %q", name))
			}
			for _, r := range name {
				if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_') {
					return domain.NewExitError(domain.ExitUsageError, fmt.Errorf("alias %q contains invalid characters (allowed: a-z, 0-9, -, _)", name))
				}
			}

			execPath, err := os.Executable()
			if err != nil {
				return fmt.Errorf("failed to determine executable path: %w", err)
			}
			execPath, err = filepath.EvalSymlinks(execPath)
			if err != nil {
				return fmt.Errorf("failed to resolve executable symlinks: %w", err)
			}

			destDir := filepath.Dir(execPath)
			if destDir == "" || destDir == "." {
				destDir = filepath.Join(appCtx.Platform.HomeDir(), ".local", "bin")
			}
			aliasPath := filepath.Join(destDir, name)

			// Safely inspect destination using Lstat without following symlinks
			if fi, err := os.Lstat(aliasPath); err == nil {
				// Destination exists. Check if it's already a symlink pointing to our executable.
				if fi.Mode()&os.ModeSymlink != 0 {
					target, rErr := os.Readlink(aliasPath)
					if rErr == nil {
						absTarget := target
						if !filepath.IsAbs(absTarget) {
							absTarget = filepath.Join(destDir, target)
						}
						absTarget, _ = filepath.EvalSymlinks(absTarget)
						if absTarget == execPath {
							st := styles(appCtx)
							fmt.Fprintf(cmd.OutOrStdout(), "%s Alias %s already exists and points to %s (idempotent).\n", st.Success.Render("✓"), aliasPath, execPath)
							return nil
						}
					}
					// Existing symlink points elsewhere: remove only if verified as a symlink
					if err := os.Remove(aliasPath); err != nil {
						return fmt.Errorf("failed to replace existing alias symlink %s: %w", aliasPath, err)
					}
				} else {
					// Not a symlink: refuse to delete non-symlink arbitrary files!
					return domain.NewExitError(domain.ExitConflictOrAudit, fmt.Errorf("refusing to overwrite existing file %s: destination is not an anx symlink", aliasPath))
				}
			}

			// Create symlink with Windows fallback
			if err := os.Symlink(execPath, aliasPath); err != nil {
				if runtime.GOOS == "windows" {
					// Windows fallback: create .cmd wrapper script
					cmdScript := filepath.Join(destDir, name+".cmd")
					scriptContent := fmt.Sprintf("@echo off\r\n\"%s\" %%*\r\n", execPath)
					if wErr := os.WriteFile(cmdScript, []byte(scriptContent), 0o755); wErr != nil {
						return fmt.Errorf("failed to create Windows alias script %s: %w (symlink error: %v)", cmdScript, wErr, err)
					}
					aliasPath = cmdScript
				} else {
					return fmt.Errorf("failed to create symlink %s -> %s: %w", aliasPath, execPath, err)
				}
			}

			st := styles(appCtx)
			fmt.Fprintf(cmd.OutOrStdout(), "%s Created binary alias %s -> %s\n", st.Success.Render("✓"), aliasPath, execPath)
			fmt.Fprintf(cmd.OutOrStdout(), "  You can now execute '%s' anywhere in your terminal.\n", name)
			return nil
		},
	}
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

func newCompletionCmd(root *cobra.Command, appCtx *AppContext) *cobra.Command {
	var install bool

	cmd := &cobra.Command{
		Use:   "completion <bash|zsh|fish|powershell>",
		Short: "Generate or install shell completion script",
		Long: `Generate or install automatic completion scripts for bash, zsh, fish, or powershell.

Pass --install to write the completion file directly to your shell's user completion path.`,
		Example: `  # Load zsh completion in current shell
  source <(anx completion zsh)

  # Permanently install zsh completion for current user
  anx completion zsh --install

  # Install bash completion
  anx completion bash --install`,
		ValidArgs: []string{"bash", "zsh", "fish", "powershell"},
		Args:      cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		RunE: func(cmd *cobra.Command, args []string) error {
			shell := args[0]
			out := cmd.OutOrStdout()
			st := styles(appCtx)

			if !install {
				switch shell {
				case "bash":
					return root.GenBashCompletionV2(out, true)
				case "zsh":
					return root.GenZshCompletion(out)
				case "fish":
					return root.GenFishCompletion(out, true)
				default:
					return root.GenPowerShellCompletionWithDesc(out)
				}
			}

			// Install flow
			home := appCtx.Platform.HomeDir()
			var targetDir, targetFile string
			switch shell {
			case "bash":
				targetDir = filepath.Join(home, ".local", "share", "bash-completion", "completions")
				targetFile = filepath.Join(targetDir, "anx")
			case "zsh":
				targetDir = filepath.Join(home, ".zsh", "completions")
				targetFile = filepath.Join(targetDir, "_anx")
			case "fish":
				targetDir = filepath.Join(home, ".config", "fish", "completions")
				targetFile = filepath.Join(targetDir, "anx.fish")
			default:
				targetDir = filepath.Join(home, ".config", "powershell")
				targetFile = filepath.Join(targetDir, "anx.ps1")
			}

			if err := os.MkdirAll(targetDir, 0o755); err != nil {
				return fmt.Errorf("creating completion directory %q: %w", targetDir, err)
			}

			f, err := os.OpenFile(targetFile, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
			if err != nil {
				return fmt.Errorf("writing completion file %q: %w", targetFile, err)
			}
			defer f.Close()

			switch shell {
			case "bash":
				err = root.GenBashCompletionV2(f, true)
			case "zsh":
				err = root.GenZshCompletion(f)
			case "fish":
				err = root.GenFishCompletion(f, true)
			default:
				err = root.GenPowerShellCompletionWithDesc(f)
			}
			if err != nil {
				return fmt.Errorf("generating %s completion: %w", shell, err)
			}

			fmt.Fprintf(out, "%s Installed %s completion to %s\n", st.Success.Render("✓"), shell, targetFile)
			if shell == "zsh" {
				fmt.Fprintln(out, "  Run 'source ~/.zshrc' or ensure ~/.zsh/completions is in your $fpath.")
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&install, "install", false, "Install completion script into user shell completions directory")

	return cmd
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
