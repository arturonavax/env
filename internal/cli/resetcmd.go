package cli

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"

	"github.com/arturonavax/env/internal/domain"
	"github.com/arturonavax/env/internal/security"
)

func newResetCmd(appCtx *AppContext) *cobra.Command {
	var (
		uninstallOnly bool
		targetProfile string
	)

	cmd := &cobra.Command{
		Use:     "reset",
		Aliases: []string{"reinstall", "destroyer-reinstall"},
		Short:   "Destroyer mode: aggressively purge and reinstall the environment from scratch",
		Long: `Destroyer Reinstall completely purges all managed packages, personal tools,
managed dotfiles, state databases, and caches, and rebuilds the environment fresh from declarations.
Use --uninstall-only to perform a complete Destroyer Uninstall without re-installing.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if targetProfile == "" {
				targetProfile = appCtx.Flags.Profile
			}
			if targetProfile == "" {
				targetProfile = "developer"
			}

			if uninstallOnly {
				return runDestroyerUninstall(cmd, appCtx, targetProfile)
			}

			return runDestroyerReinstall(cmd, appCtx, targetProfile)
		},
	}

	cmd.Flags().BoolVar(&uninstallOnly, "uninstall-only", false, "Execute Destroyer Uninstall without reinstalling")
	cmd.Flags().StringVar(&targetProfile, "profile", "", "Profile to reinstall after purge (default: developer)")
	_ = cmd.RegisterFlagCompletionFunc("profile", profileCompletion(appCtx))

	return cmd
}

func runDestroyerReinstall(cmd *cobra.Command, appCtx *AppContext, profile string) error {
	desc := fmt.Sprintf("PURGE all packages, tools, configurations, and state, then REINSTALL profile %q", profile)
	confirmed, err := confirmDestroyerAction(cmd, appCtx, desc)
	if err != nil {
		return err
	}
	if !confirmed {
		return nil
	}

	if !appCtx.Flags.ScriptMode && !appCtx.Flags.Quiet {
		renderSecuritySealBanner(cmd.OutOrStdout(), appCtx)
	}
	if err := verifySecuritySeal(cmd, appCtx); err != nil {
		return err
	}

	st := styles(appCtx)
	out := cmd.OutOrStdout()

	if appCtx.Flags.DryRun {
		fmt.Fprintln(out, st.Header.Render("Destroyer Reinstall Plan (Dry-Run):"))
	} else {
		if appCtx.Platform != nil && appCtx.Platform.HasSudo() && !appCtx.Platform.IsRoot() {
			if !security.IsSudoAuthenticated() {
				if !isTerminal(cmd) {
					return domain.NewExitError(
						domain.ExitPreflightFailed,
						errors.New("administrator privileges (sudo) required; run 'sudo -v' prior to non-interactive execution"),
					)
				}
				fmt.Fprintln(out, "==> Administrator privileges (sudo) required for system package operations.")
				if err := security.AuthenticateSudoCLI(); err != nil {
					return fmt.Errorf("sudo authentication failed: %w", err)
				}
			}
			stopKeepalive := security.StartSudoKeepalive(cmd.Context())
			defer stopKeepalive()
		}
		fmt.Fprintln(out, st.Header.Render("Executing Destroyer Reinstall..."))
	}

	res, err := appCtx.DestroyerSvc.DestroyerReinstall(cmd.Context(), profile, appCtx.Flags.DryRun)
	if err != nil {
		return err
	}
	if !appCtx.Flags.DryRun && profile != "" && appCtx.Store != nil {
		_ = appCtx.Store.SetActiveProfile(profile)
	}

	return emit(cmd, appCtx, res, func(w io.Writer) {
		mode := "Executed"
		if appCtx.Flags.DryRun {
			mode = "Planned (Dry-Run)"
		}
		fmt.Fprintf(w, "\n%s Destroyer Reinstall:\n", mode)
		fmt.Fprintf(w, "  • Uninstalled Packages: %d\n", res.UninstalledPackages)
		fmt.Fprintf(w, "  • Purged Configurations: %d\n", res.PurgedConfigs)
		if len(res.PurgedPaths) > 0 {
			fmt.Fprintln(w, "  • Purged Paths:")
			for _, p := range res.PurgedPaths {
				fmt.Fprintf(w, "      - %s\n", p)
			}
		}
		if res.Reinstalled {
			fmt.Fprintf(w, "  • Fresh Installation:   %s (Plan ID: %s)\n", st.Success.Render("[COMPLETED]"), res.ReinstallPlanID)
			fmt.Fprintf(w, "  • Dotfile Sync:         %s\n", st.Success.Render("[COMPLETED]"))
		}
	})
}

func runDestroyerUninstall(cmd *cobra.Command, appCtx *AppContext, profile string) error {
	desc := fmt.Sprintf("PURGE packages, tools, configurations, state, caches, and anx binaries (profile: %s)", profile)
	confirmed, err := confirmDestroyerAction(cmd, appCtx, desc)
	if err != nil {
		return err
	}
	if !confirmed {
		return nil
	}

	if !appCtx.Flags.ScriptMode && !appCtx.Flags.Quiet {
		renderSecuritySealBanner(cmd.OutOrStdout(), appCtx)
	}
	if err := verifySecuritySeal(cmd, appCtx); err != nil {
		return err
	}

	st := styles(appCtx)
	out := cmd.OutOrStdout()

	if appCtx.Flags.DryRun {
		fmt.Fprintln(out, st.Header.Render("Destroyer Uninstall Plan (Dry-Run):"))
	} else {
		if appCtx.Platform != nil && appCtx.Platform.HasSudo() && !appCtx.Platform.IsRoot() {
			if !security.IsSudoAuthenticated() {
				if !isTerminal(cmd) {
					return domain.NewExitError(
						domain.ExitPreflightFailed,
						errors.New("administrator privileges (sudo) required; run 'sudo -v' prior to non-interactive execution"),
					)
				}
				fmt.Fprintln(out, "==> Administrator privileges (sudo) required for system package operations.")
				if err := security.AuthenticateSudoCLI(); err != nil {
					return fmt.Errorf("sudo authentication failed: %w", err)
				}
			}
			stopKeepalive := security.StartSudoKeepalive(cmd.Context())
			defer stopKeepalive()
		}
		fmt.Fprintln(out, st.Header.Render("Executing Destroyer Uninstall..."))
	}

	res, err := appCtx.DestroyerSvc.DestroyerUninstall(cmd.Context(), profile, appCtx.Flags.DryRun, true)
	if err != nil {
		return err
	}

	return emit(cmd, appCtx, res, func(w io.Writer) {
		mode := "Executed"
		if appCtx.Flags.DryRun {
			mode = "Planned (Dry-Run)"
		}
		fmt.Fprintf(w, "\n%s Destroyer Uninstall:\n", mode)
		fmt.Fprintf(w, "  • Uninstalled Packages: %d\n", res.UninstalledPackages)
		fmt.Fprintf(w, "  • Purged Configurations: %d\n", res.PurgedConfigs)
		if len(res.PurgedPaths) > 0 {
			fmt.Fprintln(w, "  • Purged Paths:")
			for _, p := range res.PurgedPaths {
				fmt.Fprintf(w, "      - %s\n", p)
			}
		}
		fmt.Fprintf(w, "\n%s System completely purged.\n", st.Success.Render("✓"))
	})
}

func confirmDestroyerAction(cmd *cobra.Command, appCtx *AppContext, actionDescription string) (bool, error) {
	if appCtx.Flags.Yes || appCtx.Flags.Force {
		return true, nil
	}

	if !isTerminal(cmd) {
		return false, domain.NewExitError(
			domain.ExitUsageError,
			errors.New("destroyer action requires confirmation; pass --yes (-y) or --force in non-interactive environments"),
		)
	}

	st := styles(appCtx)
	out := cmd.OutOrStdout()
	fmt.Fprintln(out)
	fmt.Fprintln(out, st.Error.Render("  ╔════════════════════════════════════════════════════════════════════╗"))
	fmt.Fprintln(out, st.Error.Render("  ║              ⚠️   D E S T R O Y E R   M O D E   ⚠️                  ║"))
	fmt.Fprintln(out, st.Error.Render("  ╚════════════════════════════════════════════════════════════════════╝"))
	fmt.Fprintf(out, "\n  %s %s\n\n", st.Warning.Render("CRITICAL:"), actionDescription)
	fmt.Fprintln(out, "  This will irrevocably DELETE and PURGE:")
	fmt.Fprintln(out, "    • Installed managed packages and developer toolchains")
	fmt.Fprintln(out, "    • Personal tools in ~/.local/share/anx/bin")
	fmt.Fprintln(out, "    • Managed dotfiles and configurations (~/.config/nvim, ghostty, herdr, etc.)")
	fmt.Fprintln(out, "    • Local state database, history, and caches")
	fmt.Fprintln(out)
	fmt.Fprint(out, "  Type 'destroy' or 'yes' to proceed: ")

	var confirm string
	_, _ = fmt.Scanln(&confirm)
	confirm = strings.TrimSpace(strings.ToLower(confirm))
	if confirm != "destroy" && confirm != "yes" && confirm != "y" {
		fmt.Fprintln(out, "\n  Operation aborted by user.")
		return false, nil
	}

	return true, nil
}
