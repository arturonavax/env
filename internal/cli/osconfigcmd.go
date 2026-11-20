package cli

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/arturonavax/env/internal/domain"
	"github.com/arturonavax/env/internal/osconfig"
	"github.com/arturonavax/env/internal/security"
)

func osconfigManager(appCtx *AppContext) (*osconfig.Manager, error) {
	if appCtx.Platform == nil || appCtx.Platform.OS() != "darwin" {
		return nil, domain.NewExitError(domain.ExitPreflightFailed, errors.New("osconfig backups are only available on macOS"))
	}
	return osconfig.NewManager(OSConfigBackupRoot(appCtx.Platform.StateDir()), appCtx.Platform.HomeDir()), nil
}

// OSConfigBackupRoot returns the directory holding osconfig snapshots.
func OSConfigBackupRoot(stateDir string) string {
	return filepath.Join(stateDir, "osconfig-backups")
}

func newOSConfigCmd(appCtx *AppContext) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "osconfig",
		Short: "Back up and restore the macOS settings changed by the osconfig script",
		Long: "The macOS osconfig script is preceded by an automatic snapshot of every preference domain,\n" +
			"/etc/pam.d/sudo_local and pmset lowpowermode it alters. At most 5 snapshots are kept; the first\n" +
			"one (your state before anx) is never deleted.",
		Args: cobra.NoArgs,
	}
	cmd.AddCommand(newOSConfigBackupCmd(appCtx), newOSConfigListCmd(appCtx), newOSConfigRestoreCmd(appCtx))
	return cmd
}

func newOSConfigBackupCmd(appCtx *AppContext) *cobra.Command {
	return &cobra.Command{
		Use:   "backup",
		Short: "Create a snapshot of the current macOS settings managed by osconfig",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			m, err := osconfigManager(appCtx)
			if err != nil {
				return err
			}
			man, created, err := m.Create(cmd.Context(), "manual", true)
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			if !created {
				fmt.Fprintf(out, "State unchanged since backup %s; no new snapshot stored.\n", man.ID)
				return nil
			}
			fmt.Fprintf(out, "Backup %s created (%d domains", man.ID, len(man.Domains))
			if len(man.Skipped) > 0 {
				fmt.Fprintf(out, ", %d skipped: %s", len(man.Skipped), strings.Join(man.Skipped, ", "))
			}
			fmt.Fprintln(out, ").")
			return nil
		},
	}
}

func newOSConfigListCmd(appCtx *AppContext) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List osconfig snapshots (the first is the original, pre-anx state)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			m, err := osconfigManager(appCtx)
			if err != nil {
				return err
			}
			all, err := m.List()
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			if len(all) == 0 {
				fmt.Fprintln(out, "No osconfig backups yet.")
				return nil
			}
			for _, man := range all {
				tag := ""
				if man.First {
					tag = "  [original - never deleted]"
				}
				fmt.Fprintf(out, "%s  %-12s macOS %-6s %2d domains%s\n", man.ID, man.Label, man.MacOSVersion, len(man.Domains), tag)
			}
			return nil
		},
	}
}

func newOSConfigRestoreCmd(appCtx *AppContext) *cobra.Command {
	var noRestart bool
	cmd := &cobra.Command{
		Use:   "restore [id|first|latest]",
		Short: "Restore macOS settings from a snapshot (default: latest; 'first' = state before anx)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			m, err := osconfigManager(appCtx)
			if err != nil {
				return err
			}
			ref := "latest"
			if len(args) == 1 {
				ref = args[0]
			}
			man, err := m.Resolve(ref)
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			if !appCtx.Flags.Yes && !appCtx.Flags.Force {
				if !isTerminal(cmd) {
					return domain.NewExitError(domain.ExitUsageError, errors.New("restore requires confirmation; pass --yes in non-interactive environments"))
				}
				fmt.Fprintf(out, "Restore snapshot %s (%s)?\n", man.ID, man.Label)
				fmt.Fprintln(out, "  Every managed preference domain is reverted to that snapshot; changes made to those domains since will be lost.")
				fmt.Fprintln(out, "  Dock, Finder, and menu bar processes will restart. A safety copy of the current state is stored first.")
				fmt.Fprint(out, "  Continue? [y/N]: ")
				var answer string
				_, _ = fmt.Scanln(&answer)
				if a := strings.ToLower(strings.TrimSpace(answer)); a != "y" && a != "yes" {
					fmt.Fprintln(out, "  Aborted.")
					return nil
				}
			}
			if appCtx.Platform.HasSudo() && !appCtx.Platform.IsRoot() && !appCtx.Flags.DryRun {
				if !security.IsSudoAuthenticated() {
					if !isTerminal(cmd) {
						return domain.NewExitError(domain.ExitPreflightFailed, errors.New("administrator privileges (sudo) required; run 'sudo -v' first"))
					}
					fmt.Fprintln(out, "Administrator privileges (sudo) required to restore system-level settings.")
					if err := security.AuthenticateSudoCLI(); err != nil {
						return fmt.Errorf("sudo authentication failed: %w", err)
					}
				}
			}
			if appCtx.Flags.DryRun {
				fmt.Fprintf(out, "Dry-run: would restore %d domains from snapshot %s.\n", len(man.Domains), man.ID)
				return nil
			}
			rep, err := m.Restore(cmd.Context(), ref, false, !noRestart)
			if err != nil {
				return err
			}
			fmt.Fprintf(out, "Restored %d items from snapshot %s.\n", len(rep.Restored), rep.Snapshot.ID)
			if rep.SafetyCopy != "" {
				fmt.Fprintf(out, "Safety copy of the previous state: %s (restore it with 'anx osconfig restore %s').\n", rep.SafetyCopy, rep.SafetyCopy)
			}
			for _, w := range rep.Warnings {
				fmt.Fprintf(out, "WARNING: %s\n", w)
			}
			fmt.Fprintln(out, "Log out and back in (or reboot) for trackpad and window-manager settings to fully apply.")
			if len(rep.Warnings) > 0 {
				return domain.NewExitError(domain.ExitRollbackOrPartial, fmt.Errorf("restore completed with %d warning(s)", len(rep.Warnings)))
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&noRestart, "no-restart", false, "Do not restart cfprefsd, Dock, Finder, and SystemUIServer after restoring")
	return cmd
}
