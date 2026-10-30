package cli

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/arturonavax/env/internal/app"
	"github.com/arturonavax/env/internal/domain"
	"github.com/arturonavax/env/internal/hacks"
)

func systemCommands(appCtx *AppContext) []*cobra.Command {
	cmds := []*cobra.Command{
		newUpdateCmd(appCtx),
		newUpgradeCmd(appCtx),
		newHacksCmd(appCtx),
		newRebuildCmd(appCtx),
		newOptimizeCmd(appCtx),
		newCleanupCmd(appCtx),
		newSleepCmd(appCtx),
	}
	for _, c := range cmds {
		c.GroupID = "system"
	}
	return cmds
}

func newSleepCmd(appCtx *AppContext) *cobra.Command {
	return &cobra.Command{
		Use:       "sleep [lock|s2idle|deep|hibernate|status]",
		Aliases:   []string{"suspend", "safe-suspend"},
		Short:     "Execute hardware-safe suspend, screen standby, or hibernation",
		ValidArgs: []string{"lock", "s2idle", "deep", "hibernate", "status"},
		Args:      cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			mode := ""
			if len(args) > 0 {
				mode = args[0]
			}
			scriptPath := filepath.Join(appCtx.RepoRoot, "scripts", "safe_suspend.sh")
			if !isFile(scriptPath) {
				binPath, err := exec.LookPath("safe-suspend")
				if err != nil {
					return fmt.Errorf("safe-suspend script not found in repo or PATH: %w", err)
				}
				scriptPath = binPath
			}
			c := exec.CommandContext(cmd.Context(), "bash", scriptPath, mode)
			c.Stdout = cmd.OutOrStdout()
			c.Stderr = cmd.ErrOrStderr()
			c.Stdin = os.Stdin
			return c.Run()
		},
	}
}

func hackCompletion(appCtx *AppContext) func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return func(_ *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
		if len(args) > 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		ids := []string{}
		for _, h := range appCtx.HacksSvc.List() {
			ids = append(ids, h.ID+"\t"+h.Name)
		}
		return ids, cobra.ShellCompDirectiveNoFileComp
	}
}

func newHacksCmd(appCtx *AppContext) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "hacks",
		Short: "Machine-specific hardware/OS optimizations (opt-in, never applied implicitly)",
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List every registered hack",
		Args:  cobra.NoArgs,
		RunE: func(c *cobra.Command, _ []string) error {
			list := appCtx.HacksSvc.List()
			return emit(c, appCtx, list, func(w io.Writer) {
				for _, h := range list {
					applied := ""
					if appCtx.HacksSvc.Applied(h.ID) {
						applied = " (applied)"
					}
					fmt.Fprintf(w, "  %-26s %-14s %s%s\n", h.ID, h.Category, h.Name, applied)
				}
			})
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "detect",
		Short: "Probe this machine and list the hacks that match its hardware",
		Args:  cobra.NoArgs,
		RunE: func(c *cobra.Command, _ []string) error {
			hw, matches := appCtx.HacksSvc.Detect(c.Context())
			payload := map[string]any{"hardware": hw, "matches": matches}
			return emit(c, appCtx, payload, func(w io.Writer) {
				fmt.Fprintf(w, "Host: %s %s | GPU: %s | DE: %s/%s\n",
					hw.DMIVendor, hw.DMIProduct, strings.Join(hw.GPUModels, ", "), hw.DesktopEnv, hw.DisplayServer)
				if len(matches) == 0 {
					fmt.Fprintln(w, "No hacks match this machine.")
				}
				for _, m := range matches {
					fmt.Fprintf(w, "  match: %-26s %s\n", m.ID, m.Name)
				}
			})
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:               "info <hack-id>",
		Short:             "Show the full declarative definition of a hack",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: hackCompletion(appCtx),
		RunE: func(c *cobra.Command, args []string) error {
			plan, err := appCtx.HacksSvc.Plan(args[0])
			if err != nil {
				return domain.NewExitError(domain.ExitUsageError, err)
			}
			return emit(c, appCtx, plan, func(w io.Writer) { printHackPlan(w, plan.Name, plan.BackupPaths, plan.Steps) })
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:               "plan <hack-id>",
		Short:             "Preview the steps a hack would run (no changes)",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: hackCompletion(appCtx),
		RunE: func(c *cobra.Command, args []string) error {
			plan, err := appCtx.HacksSvc.Plan(args[0])
			if err != nil {
				return domain.NewExitError(domain.ExitUsageError, err)
			}
			return emit(c, appCtx, plan, func(w io.Writer) { printHackPlan(w, plan.Name, plan.BackupPaths, plan.Steps) })
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:               "apply <hack-id>",
		Short:             "Snapshot touched files, validate scripts against guardrails, then apply the hack",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: hackCompletion(appCtx),
		RunE: func(c *cobra.Command, args []string) error {
			if appCtx.Flags.DryRun {
				plan, err := appCtx.HacksSvc.Plan(args[0])
				if err != nil {
					return err
				}
				printHackPlan(c.OutOrStdout(), plan.Name, plan.BackupPaths, plan.Steps)
				return nil
			}
			res, err := appCtx.HacksSvc.Apply(c.Context(), args[0])
			if err != nil {
				return err
			}
			return emit(c, appCtx, res, func(w io.Writer) {
				fmt.Fprintf(w, "Applied %s (snapshot %s): %s\n", res.HackID, res.SnapshotID, strings.Join(res.AppliedSteps, ", "))
			})
		},
	})
	return cmd
}

func newRebuildCmd(appCtx *AppContext) *cobra.Command {
	return &cobra.Command{
		Use:       "rebuild [all|go|rust|node|python|nvim|zsh]...",
		Short:     "Rebuild global tools after toolchain upgrades",
		ValidArgs: []string{"all", "go", "rust", "node", "python", "nvim", "zsh"},
		Args:      cobra.OnlyValidArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			results, err := appCtx.OpsSvc.Rebuild(cmd.Context(), args, appCtx.Flags.DryRun)
			if err != nil {
				return err
			}
			return emit(cmd, appCtx, results, func(w io.Writer) {
				for _, r := range results {
					state := fmt.Sprintf("%d rebuilt", r.Rebuilt)
					if r.Skipped {
						state = "skipped"
					}
					if r.Error != "" {
						state = "error: " + r.Error
					}
					fmt.Fprintf(w, "  %-8s %s %s\n", r.Scope, state, r.Message)
				}
			})
		},
	}
}

func newOptimizeCmd(appCtx *AppContext) *cobra.Command {
	var (
		auditOnly bool
		skipLynis bool
		useAI     bool
		apply     bool
	)
	cmd := &cobra.Command{
		Use:   "optimize",
		Short: "Audit hardware/kernel/security and produce a guardrail-validated optimization script",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			req := app.OptimizeRequest{AuditOnly: auditOnly, SkipLynis: skipLynis, Apply: apply && !appCtx.Flags.DryRun}
			if useAI {
				fn, err := aiPromptFunc(appCtx)
				if err != nil {
					return err
				}
				req.Prompt = fn
			}
			res, err := appCtx.OpsSvc.Optimize(cmd.Context(), req)
			if err != nil {
				return domain.NewExitError(domain.ExitConflictOrAudit, err)
			}
			return emit(cmd, appCtx, res, func(w io.Writer) {
				fmt.Fprintln(w, res.Report.MarkdownSummary())
				if res.Script != "" {
					fmt.Fprintln(w, "\n--- generated script (guardrail-validated) ---")
					fmt.Fprintln(w, res.Script)
					fmt.Fprintln(w, res.Output)
				}
			})
		},
	}
	f := cmd.Flags()
	f.BoolVarP(&auditOnly, "audit-only", "a", false, "Print the audit report and exit")
	f.BoolVar(&skipLynis, "skip-lynis", false, "Skip the Lynis security scan")
	f.BoolVar(&useAI, "ai", false, "Synthesize the script with the configured aitools harness")
	f.BoolVar(&apply, "apply", false, "Execute the validated script (default is dry-run validation)")
	return cmd
}

func printHackPlan(w io.Writer, name string, backups []string, steps []hacks.HackPlanStep) {
	fmt.Fprintf(w, "%s\n  backups: %s\n", name, strings.Join(backups, ", "))
	for i, st := range steps {
		priv := ""
		if st.Privileged {
			priv = " [sudo]"
		}
		fmt.Fprintf(w, "  %d. %s%s - %s\n     $ %s\n", i+1, st.Name, priv, st.Description, st.CommandLine)
	}
}
