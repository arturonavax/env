package cli

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"

	"github.com/arturonavax/env/internal/app"
	"github.com/arturonavax/env/internal/domain"
)

func configCommands(appCtx *AppContext) []*cobra.Command {
	cmds := []*cobra.Command{
		newDiffCmd(appCtx),
		newSyncCmd(appCtx),
		newCaptureCmd(appCtx),
		newBackupCmd(appCtx),
		newRollbackCmd(appCtx),
		newHistoryCmd(appCtx),
	}
	for _, c := range cmds {
		c.GroupID = "config"
	}
	return cmds
}

func groupCompletion(appCtx *AppContext) func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
		seen := map[string]bool{}
		out := []string{"all"}
		for _, c := range appCtx.Catalog.ListConfigurations() {
			for _, k := range []string{c.Group, c.ID} {
				if k != "" && !seen[k] {
					seen[k] = true
					out = append(out, k)
				}
			}
		}
		return out, cobra.ShellCompDirectiveNoFileComp
	}
}

func newDiffCmd(appCtx *AppContext) *cobra.Command {
	var patch bool
	cmd := &cobra.Command{
		Use:               "diff [group|config-id]",
		Short:             "Show real differences between repository and live dotfiles",
		Args:              cobra.MaximumNArgs(1),
		ValidArgsFunction: groupCompletion(appCtx),
		RunE: func(cmd *cobra.Command, args []string) error {
			group := ""
			if len(args) == 1 {
				group = args[0]
			}
			diffs, err := appCtx.SyncSvc.Diff(cmd.Context(), group)
			if err != nil {
				return err
			}
			return emit(cmd, appCtx, diffs, func(w io.Writer) {
				st := styles(appCtx)
				fmt.Fprintln(w, st.Header.Render("Dotfile Differences"))
				for _, d := range diffs {
					tag := st.Success.Render("[" + d.State + "]")
					switch d.State {
					case "drift", "missing":
						tag = st.Warning.Render("[" + d.State + "]")
					case "conflict":
						tag = st.Error.Render("[CONFLICT]")
					}
					summary := d.Summary
					if summary != "" {
						summary = " (" + summary + ")"
					}
					fmt.Fprintf(w, "  %-30s %s%s\n", d.ConfigID, tag, summary)
					if patch && d.Unified != "" {
						fmt.Fprintln(w, d.Unified)
					}
				}
			})
		},
	}
	cmd.Flags().BoolVarP(&patch, "patch", "p", false, "Print unified diffs")
	return cmd
}

func newSyncCmd(appCtx *AppContext) *cobra.Command {
	var (
		push  bool
		force bool
	)
	cmd := &cobra.Command{
		Use:   "sync [group|config-id]",
		Short: "Deploy dotfiles with true 3-way merge (or --push to capture them back into the repo)",
		Long: "Groups: terminal, editor, devtools, ai, osconfig, vscode, cursor, all.\n" +
			"vscode and cursor are only synced when named explicitly.",
		Args:              cobra.MaximumNArgs(1),
		ValidArgsFunction: groupCompletion(appCtx),
		RunE: func(cmd *cobra.Command, args []string) error {
			if appCtx.Flags.ScriptMode {
				return renderSyncScript(cmd.OutOrStdout(), appCtx, args, push)
			}
			opts := appSyncOptions(appCtx, args, push, force)
			results, err := appCtx.SyncSvc.Sync(cmd.Context(), opts)
			if err != nil {
				return err
			}
			if err := renderSync(cmd, appCtx, results); err != nil {
				return err
			}
			for _, r := range results {
				if r.Action == "conflict" {
					return domain.NewExitError(domain.ExitConflictOrAudit, errors.New("unresolved merge conflicts; inspect with 'anx diff -p' or re-run with --force"))
				}
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&push, "push", false, "Capture live dotfiles back into the repository")
	cmd.Flags().BoolVar(&force, "force", false, "Repository version wins on conflicts")
	return cmd
}

func renderSync(cmd *cobra.Command, appCtx *AppContext, results []app.SyncResult) error {
	return emit(cmd, appCtx, results, func(w io.Writer) {
		st := styles(appCtx)
		fmt.Fprintf(w, "Processed %d configurations (dry-run: %v)\n", len(results), appCtx.Flags.DryRun)
		for _, r := range results {
			tag := st.Success.Render("[" + r.Action + "]")
			if r.Action == "conflict" {
				tag = st.Error.Render("[conflict]")
			}
			fmt.Fprintf(w, "  %-26s %s %s\n", r.ConfigID, tag, r.Detail)
		}
	})
}

func appSyncOptions(appCtx *AppContext, args []string, push, force bool) app.SyncOptions {
	opts := app.SyncOptions{DryRun: appCtx.Flags.DryRun, Push: push, Force: force}
	if len(args) == 1 {
		opts.Group = args[0]
	}
	return opts
}

func newBackupCmd(appCtx *AppContext) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "backup",
		Aliases: []string{"snapshot"},
		Short:   "Create, list and restore configuration snapshots",
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "create [description]",
		Short: "Snapshot every managed dotfile (files and directories)",
		RunE: func(c *cobra.Command, args []string) error {
			desc := "manual snapshot"
			if len(args) > 0 {
				desc = strings.Join(args, " ")
			}
			paths := []string{}
			for _, cfg := range appCtx.Catalog.ListConfigurations() {
				if cfg.SupportsOS(appCtx.Platform.OS()) {
					paths = append(paths, appCtx.Platform.ExpandPath(cfg.TargetForOS(appCtx.Platform.OS())))
				}
			}
			snap, err := appCtx.Snapshots.CreateSnapshot(desc, paths)
			if err != nil {
				return err
			}
			return emit(c, appCtx, snap, func(w io.Writer) {
				fmt.Fprintf(w, "Created snapshot %s (%d files, %d bytes)\n", snap.ID, len(snap.Files), snap.SizeBytes)
			})
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List snapshots (newest first)",
		Args:  cobra.NoArgs,
		RunE: func(c *cobra.Command, _ []string) error {
			snaps, err := appCtx.Snapshots.ListSnapshots()
			if err != nil {
				return err
			}
			return emit(c, appCtx, snaps, func(w io.Writer) {
				if len(snaps) == 0 {
					fmt.Fprintln(w, "No snapshots found.")
				}
				for _, s := range snaps {
					fmt.Fprintf(w, "  %-24s %-26s (%d files, %d bytes)\n", s.ID, s.Description, len(s.Files), s.SizeBytes)
				}
			})
		},
	})

	var latest bool
	restore := &cobra.Command{
		Use:   "restore [snapshot-id]",
		Short: "Restore a snapshot after verifying its SHA-256 manifest (--latest for the newest)",
		Args:  cobra.MaximumNArgs(1),
		ValidArgsFunction: func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
			snaps, _ := appCtx.Snapshots.ListSnapshots()
			ids := make([]string, 0, len(snaps))
			for _, s := range snaps {
				ids = append(ids, s.ID)
			}
			return ids, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(c *cobra.Command, args []string) error {
			if latest {
				snap, err := appCtx.Snapshots.RestoreLatest()
				if err != nil {
					return err
				}
				fmt.Fprintf(c.OutOrStdout(), "Restored latest snapshot %s\n", snap.ID)
				return nil
			}
			if len(args) != 1 {
				return domain.NewExitError(domain.ExitUsageError, errors.New("provide a snapshot id or --latest"))
			}
			if err := appCtx.Snapshots.Restore(args[0]); err != nil {
				return err
			}
			fmt.Fprintf(c.OutOrStdout(), "Restored snapshot %s\n", args[0])
			return nil
		},
	}
	restore.Flags().BoolVar(&latest, "latest", false, "Restore the most recent snapshot")
	cmd.AddCommand(restore)
	return cmd
}

func newRollbackCmd(appCtx *AppContext) *cobra.Command {
	return &cobra.Command{
		Use:   "rollback",
		Short: "Restore the most recent pre-change configuration snapshot",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			snap, err := appCtx.InstallSvc.Rollback(cmd.Context())
			if err != nil {
				return domain.NewExitError(domain.ExitRollbackOrPartial, err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Rolled back to snapshot %s\n", snap.ID)
			return nil
		},
	}
}

func newHistoryCmd(appCtx *AppContext) *cobra.Command {
	return &cobra.Command{
		Use:   "history",
		Short: "Display the transaction journal",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			entries, err := appCtx.Journal.ReadEntries()
			if err != nil {
				return err
			}
			return emit(cmd, appCtx, entries, func(w io.Writer) {
				if len(entries) == 0 {
					fmt.Fprintln(w, "No transaction history recorded yet.")
				}
				for _, e := range entries {
					fmt.Fprintf(w, "  %s [%-11s] %s step:%d %s\n", e.Timestamp.Format("15:04:05"), e.Type, e.TxID, e.StepID, e.Target)
				}
			})
		},
	}
}

func renderSyncScript(w io.Writer, appCtx *AppContext, args []string, push bool) error {
	group := "all"
	if len(args) == 1 {
		group = args[0]
	}
	cfgs := appCtx.Catalog.ListConfigurationsByGroup(group)

	fmt.Fprintln(w, "#!/usr/bin/env bash")
	if push {
		fmt.Fprintf(w, "# ANX Dotfile Capture Script (Group: %s)\n", group)
	} else {
		fmt.Fprintf(w, "# ANX Dotfile Deployment Script (Group: %s)\n", group)
	}
	fmt.Fprintln(w, "set -euo pipefail")

	for _, c := range cfgs {
		target := c.TargetForOS(appCtx.Platform.OS())
		source := c.Source
		if push {
			fmt.Fprintf(w, "# Capture %s\n", c.ID)
			fmt.Fprintf(w, "mkdir -p $(dirname %s)\n", source)
			fmt.Fprintf(w, "cp -a %s %s\n\n", target, source)
		} else {
			fmt.Fprintf(w, "# Deploy %s\n", c.ID)
			fmt.Fprintf(w, "mkdir -p $(dirname %s)\n", target)
			fmt.Fprintf(w, "cp -a %s %s\n\n", source, target)
		}
	}
	return nil
}
