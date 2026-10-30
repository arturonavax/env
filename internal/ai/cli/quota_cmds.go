package cli

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/arturonavax/env/internal/ai/daemon"
	"github.com/arturonavax/env/internal/ai/pipeline"
	"github.com/arturonavax/env/internal/ai/quota"
	"github.com/arturonavax/env/internal/security"
	"github.com/spf13/cobra"
)

func pipelineOpts(rt *runtimeEnv, cmd *cobra.Command, args []string) pipeline.RunOptions {
	return pipeline.RunOptions{
		Name:       args[0],
		Args:       args[1:],
		WorkDir:    rt.workDir(),
		Stdin:      cmd.InOrStdin(),
		StdinIsTTY: rt.isStdinTTY(cmd),
		Out:        cmd.OutOrStdout(),
		ErrOut:     cmd.ErrOrStderr(),
	}
}

func newQuotaCmd(rt *runtimeEnv) *cobra.Command {
	var (
		asJSON bool
		target string
	)
	cmd := &cobra.Command{
		Use:     "quota",
		Short:   "Inspect model quota windows, remaining percentage, and reset times",
		GroupID: "quota",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			evals, err := rt.quotaSvc.EvaluateAll(cmd.Context(), target)
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			if asJSON {
				s, err := quota.FormatJSON(evals)
				if err != nil {
					return err
				}
				fmt.Fprintln(out, s)
				return nil
			}
			fmt.Fprintln(out, quota.FormatTable(evals))
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "Output telemetry in JSON format")
	cmd.Flags().StringVarP(&target, "target", "t", "", "Filter by target ID")
	return cmd
}

func newWatchCmd(rt *runtimeEnv) *cobra.Command {
	var (
		interval int
		target   string
	)
	cmd := &cobra.Command{
		Use:     "watch",
		Short:   "Live-updating TUI dashboard for AI quotas and timers",
		GroupID: "quota",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if interval <= 0 {
				return fmt.Errorf("--interval must be a positive number of seconds")
			}
			return quota.RunWatch(
				cmd.Context(),
				cmd.OutOrStdout(),
				rt.quotaSvc,
				time.Duration(interval)*time.Second,
				target,
				rt.isInteractive(cmd),
			)
		},
	}
	cmd.Flags().IntVar(&interval, "interval", 30, "Refresh interval in seconds")
	cmd.Flags().StringVarP(&target, "target", "t", "", "Filter by target ID")
	return cmd
}

func newWakeCmd(rt *runtimeEnv) *cobra.Command {
	var (
		force  bool
		target string
	)
	cmd := &cobra.Command{
		Use:     "wake",
		Short:   "Evaluate models and trigger wakeup probes for expired/idle quotas",
		GroupID: "quota",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			rep, err := rt.quotaSvc.WakeWithOptions(cmd.Context(), quota.WakeOptions{
				Force:        force,
				TargetFilter: target,
				Reason:       "manual",
				Out:          cmd.OutOrStdout(),
			})
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "woken=%v skipped=%v failed=%d\n", rep.Woken, rep.Skipped, len(rep.Failed))
			if !rep.AllSucceeded() {
				return fmt.Errorf("%d wake probe(s) failed", len(rep.Failed))
			}
			return nil
		},
	}
	cmd.Flags().BoolVarP(&force, "force", "f", false, "Force wake probe even if the window is active")
	cmd.Flags().StringVarP(&target, "target", "t", "", "Filter by target ID")
	return cmd
}

func newDaemonCmd(rt *runtimeEnv) *cobra.Command {
	actions := []string{"status", "install", "uninstall", "run-wake-cycle", "auto-wake", "run-monitor", "log", "logs"}
	cmd := &cobra.Command{
		Use:       "daemon <action> [arg]",
		Short:     "Manage background wakeup daemon, RTC alarms, and safe suspension",
		GroupID:   "quota",
		Args:      cobra.MatchAll(cobra.RangeArgs(1, 2), cobra.OnlyValidArgs),
		ValidArgs: actions,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDaemonAction(rt, cmd, args)
		},
	}
	return cmd
}

func runDaemonAction(rt *runtimeEnv, cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	out := cmd.OutOrStdout()
	extra := ""
	if len(args) > 1 {
		extra = args[1]
	}

	switch args[0] {
	case "status":
		rep, err := rt.daemonMgr.Status(ctx)
		if err != nil {
			return err
		}
		fmt.Fprintln(out, daemon.FormatStatus(rep))
	case "install":
		res, err := rt.daemonMgr.Install(ctx)
		if err != nil {
			return err
		}
		for _, f := range res.Files {
			fmt.Fprintln(out, "installed:", f)
		}
	case "uninstall":
		return rt.daemonMgr.Uninstall(ctx)
	case "auto-wake":
		switch extra {
		case "on", "off":
			return rt.daemonMgr.SetAutoWake(ctx, extra == "on")
		default:
			return fmt.Errorf("auto-wake requires 'on' or 'off'")
		}
	case "run-monitor":
		return rt.daemonMgr.RunMonitor(ctx, out)
	case "run-wake-cycle":
		res, err := rt.daemonMgr.RunWakeCycle(ctx, out)
		if err != nil {
			return err
		}
		if res.AbortedSuspend {
			return fmt.Errorf("suspend aborted: %s", res.AbortReason)
		}
	case "log", "logs":
		n := 50
		if extra != "" {
			v, err := strconv.Atoi(extra)
			if err != nil || v <= 0 {
				return fmt.Errorf("invalid line count %q", extra)
			}
			n = v
		}
		lines, err := rt.daemonMgr.Logs(n)
		if err != nil {
			return err
		}
		for _, l := range lines {
			fmt.Fprintln(out, l)
		}
	}
	return nil
}

func newOptimizeCmd(rt *runtimeEnv) *cobra.Command {
	var p OptimizeParams
	cmd := &cobra.Command{
		Use:     "optimize",
		Aliases: []string{"sysopt"},
		Short:   "Audit hardware, kernel and security, then generate a guardrail-validated optimization script",
		GroupID: "system",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			p.Out = cmd.OutOrStdout()
			p.ErrOut = cmd.ErrOrStderr()

			if p.ValidateFile != "" {
				return validateScriptFile(p)
			}
			if rt.opts.optimizeFunc == nil {
				return fmt.Errorf("optimizer engine is not wired in this binary; use 'anx optimize' or pass --validate <script>")
			}
			return rt.opts.optimizeFunc(cmd.Context(), p)
		},
	}
	f := cmd.Flags()
	f.BoolVarP(&p.DryRun, "dry-run", "d", false, "Audit and validate without applying (default)")
	f.BoolVar(&p.Apply, "apply", false, "Execute the validated optimization script")
	f.BoolVarP(&p.AuditOnly, "audit-only", "a", false, "Print the actionable audit report and exit")
	f.BoolVar(&p.SkipLynis, "skip-lynis", false, "Bypass the Lynis audit scan")
	f.BoolVarP(&p.Wizard, "wizard", "w", false, "Launch interactive step-by-step wizard")
	f.StringVarP(&p.OutputFile, "output", "o", "", "Save generated script to file")
	f.StringVarP(&p.Backend, "backend", "b", "", "AI CLI backend override")
	f.StringVarP(&p.Model, "model", "m", "", "AI model override")
	f.StringVar(&p.ValidateFile, "validate", "", "Validate an existing script against the security guardrails")
	return cmd
}

func validateScriptFile(p OptimizeParams) error {
	data, err := os.ReadFile(p.ValidateFile)
	if err != nil {
		return fmt.Errorf("reading script: %w", err)
	}
	violations := security.ValidateScriptGuardrails(string(data))
	if len(violations) == 0 {
		fmt.Fprintf(p.Out, "OK: %s passed all %d security guardrails\n", p.ValidateFile, 12)
		return nil
	}
	for _, v := range violations {
		fmt.Fprintf(p.ErrOut, "BLOCKED %s line %d: %s\n", v.RuleID, v.Line, v.Description)
	}
	return fmt.Errorf("%d guardrail violation(s) detected", len(violations))
}
