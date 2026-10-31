package cli

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/arturonavax/env/internal/app"
	"github.com/arturonavax/env/internal/domain"
	"github.com/arturonavax/env/internal/scriptgen"
	"github.com/arturonavax/env/internal/security"
)

func newAuditCmd(appCtx *AppContext) *cobra.Command {
	var (
		profile      string
		showReport   bool
		showScript   bool
		useAI        bool
		apply        bool
		scanPackages bool
	)

	cmd := &cobra.Command{
		Use:   "audit",
		Short: "Multi-level system security audit with profile hardening and remediation scripts",
		Long: `Run comprehensive, profile-aware system security audits across network/firewall,
authentication, permissions, kernel sysctl parameters, and package vulnerabilities.

Profiles:
  workstation : Developer workstation/laptop (balanced security and usability)
  server      : Hardened production server (strict firewall, no root SSH, strict sysctl)
  minimal     : Essential baseline checks

Supports generating reproducible, idempotent Bash remediation scripts with static
security guardrails, and optional AI synthesis via the configured aitools harness.`,
		Example: `  # Run default workstation security audit
  anx audit

  # Run strict server security audit
  anx audit --profile server

  # Output inspectable shell remediation script
  anx audit --script

  # Synthesize remediation script using configured AI harness and review
  anx audit --ai --script

  # Apply verified remediations
  anx audit --apply

  # Scan installed package binaries against OSV.dev vulnerabilities
  anx audit --packages`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			out := cmd.OutOrStdout()

			// Package CVE scan mode
			if scanPackages {
				if appCtx.Flags.Offline {
					return domain.NewExitError(domain.ExitPreflightFailed, fmt.Errorf("package audit requires network access; remove --offline"))
				}
				findings, err := appCtx.AuditSvc.Audit(ctx)
				if err != nil {
					return err
				}
				return emit(cmd, appCtx, findings, func(w io.Writer) {
					if len(findings) == 0 {
						fmt.Fprintln(w, "No known vulnerabilities detected on installed tools.")
						return
					}
					for _, f := range findings {
						fmt.Fprintf(w, "  [%s] %s %s: %s\n         %s\n", f.Severity, f.ToolName, f.DetectedVersion, f.Summary, f.AdvisoryURL)
					}
				})
			}

			// Host security audit mode
			req := app.SecurityAuditRequest{
				Profile: profile,
				UseAI:   useAI,
				Apply:   apply && !appCtx.Flags.DryRun,
			}

			if useAI {
				fn, err := aiPromptFunc(appCtx)
				if err != nil {
					return fmt.Errorf("initializing AI harness: %w", err)
				}
				req.PromptFunc = security.SecurityPromptFunc(fn)
			}

			res, err := appCtx.AuditSvc.SystemAudit(ctx, req, appCtx.Flags.DryRun)
			if err != nil {
				return domain.NewExitError(domain.ExitConflictOrAudit, err)
			}

			// Script mode: output shell script directly
			if showScript || appCtx.Flags.ScriptMode {
				if res.Script == "" {
					b := scriptgen.NewBuilder("audit", appCtx.Platform)
					b.AddSection("Security Remediation", "System is already fully hardened. No remediations required.")
					b.AddCommand("echo '==> No security remediations required.'", "Clean audit")
					fmt.Fprint(out, b.Render())
					return nil
				}
				if useAI {
					fmt.Fprintln(out, res.Script)
					return nil
				}
				script := scriptgen.TranslateSecurityAudit(res.Report, appCtx.Platform)
				fmt.Fprint(out, script)
				return nil
			}

			return emit(cmd, appCtx, res, func(w io.Writer) {
				fmt.Fprintln(w, res.Report.MarkdownSummary())

				if res.Script != "" && !apply {
					fmt.Fprintln(w, "---")
					fmt.Fprintln(w, "💡 Run 'anx audit --script' to inspect the shell remediation script, or 'anx audit --apply' to execute.")
				}
				if res.Output != "" {
					fmt.Fprintln(w, "\n--- Remediation Execution Output ---")
					fmt.Fprintln(w, res.Output)
				}
			})
		},
	}

	f := cmd.Flags()
	f.StringVarP(&profile, "profile", "P", security.ProfileWorkstation, "Audit profile (workstation|server|minimal)")
	f.BoolVarP(&showReport, "report", "r", false, "Print detailed markdown audit report")
	f.BoolVarP(&showScript, "script", "s", false, "Print generated shell remediation script")
	f.BoolVar(&useAI, "ai", false, "Synthesize remediation script using configured AI harness")
	f.BoolVar(&apply, "apply", false, "Execute the guardrail-validated remediation script")
	f.BoolVarP(&scanPackages, "packages", "p", false, "Scan managed package tools for known OSV vulnerabilities")

	_ = cmd.RegisterFlagCompletionFunc("profile", staticCompletion(security.ProfileWorkstation, security.ProfileServer, security.ProfileMinimal))

	return cmd
}
