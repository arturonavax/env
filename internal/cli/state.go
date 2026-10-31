package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/arturonavax/env/internal/storage/sqlite"
)

func newStateCmd(appCtx *AppContext) *cobra.Command {
	stateCmd := &cobra.Command{
		Use:     "state",
		Short:   "Manage transactional SQLite database and state migrations",
		GroupID: "meta",
	}

	stateCmd.AddCommand(newStateStatusCmd(appCtx))
	stateCmd.AddCommand(newStateMigrateCmd(appCtx))
	stateCmd.AddCommand(newStateBackupCmd(appCtx))
	stateCmd.AddCommand(newStateVerifyCmd(appCtx))
	stateCmd.AddCommand(newStateExportCmd(appCtx))

	return stateCmd
}

func newStateStatusCmd(appCtx *AppContext) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Display status of state database, WAL mode, and migrations",
		RunE: func(cmd *cobra.Command, _ []string) error {
			stateDir := appCtx.Platform.StateDir()
			dbPath := filepath.Join(stateDir, "anx.db")
			jsonPath := filepath.Join(stateDir, "state.json")

			type statusInfo struct {
				DatabasePath     string `json:"database_path"`
				DatabaseExists   bool   `json:"database_exists"`
				LegacyJSONExists bool   `json:"legacy_json_exists"`
				ResourceCount    int    `json:"resource_count"`
				ConfigBaseCount  int    `json:"config_base_count"`
				AppliedHackCount int    `json:"applied_hack_count"`
				ActiveProfile    string `json:"active_profile"`
				ActiveTheme      string `json:"active_theme"`
			}

			info := statusInfo{
				DatabasePath:  dbPath,
				ActiveProfile: "developer",
			}

			if _, err := os.Stat(dbPath); err == nil {
				info.DatabaseExists = true
			}
			if _, err := os.Stat(jsonPath); err == nil {
				info.LegacyJSONExists = true
			}

			if appCtx.Store != nil {
				allRes := appCtx.Store.AllResources()
				info.ResourceCount = len(allRes)
				info.ActiveProfile = appCtx.Store.ActiveProfile()
				info.ActiveTheme = appCtx.Store.ActiveTheme()

				if db := appCtx.Store.DB(); db != nil {
					allCfg, _ := db.AllConfigBases(cmd.Context())
					info.ConfigBaseCount = len(allCfg)
					allHacks, _ := db.AllAppliedHacks(cmd.Context())
					info.AppliedHackCount = len(allHacks)
				}
			}

			return emit(cmd, appCtx, info, func(w io.Writer) {
				fmt.Fprintln(w, "ANX State Engine Status")
				fmt.Fprintln(w, "=======================")
				fmt.Fprintf(w, "Database Path:       %s\n", info.DatabasePath)
				fmt.Fprintf(w, "Database Active:     %v\n", info.DatabaseExists)
				fmt.Fprintf(w, "Legacy state.json:   %v\n", info.LegacyJSONExists)
				fmt.Fprintf(w, "Tracked Resources:   %d\n", info.ResourceCount)
				fmt.Fprintf(w, "Config Bases (3-way): %d\n", info.ConfigBaseCount)
				fmt.Fprintf(w, "Applied Hacks:       %d\n", info.AppliedHackCount)
				fmt.Fprintf(w, "Active Profile:      %s\n", info.ActiveProfile)
				if info.ActiveTheme != "" {
					fmt.Fprintf(w, "Active Theme:        %s\n", info.ActiveTheme)
				}
			})
		},
	}
}

func newStateMigrateCmd(appCtx *AppContext) *cobra.Command {
	var planOnly, apply bool

	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Plan or apply migration from legacy state.json to transactional SQLite",
		RunE: func(cmd *cobra.Command, _ []string) error {
			stateDir := appCtx.Platform.StateDir()

			plan, _, err := sqlite.PlanMigration(stateDir)
			if err != nil {
				return fmt.Errorf("planning migration: %w", err)
			}
			if plan == nil {
				return emit(cmd, appCtx, map[string]string{"message": "No legacy state.json found; nothing to migrate"}, func(w io.Writer) {
					fmt.Fprintln(w, "No legacy state.json found; state is already native SQLite.")
				})
			}

			// If dry-run or plan-only was requested, output plan without applying
			if appCtx.Flags.DryRun || planOnly || !apply {
				return emit(cmd, appCtx, plan, func(w io.Writer) {
					fmt.Fprintln(w, "ANX State Migration Plan (Dry-Run)")
					fmt.Fprintln(w, "=================================")
					fmt.Fprintf(w, "Source File:       %s\n", plan.SourcePath)
					fmt.Fprintf(w, "Source SHA256:     %s\n", plan.SourceSHA256)
					fmt.Fprintf(w, "Resources to copy: %d\n", plan.ResourceCount)
					fmt.Fprintf(w, "Config Bases:      %d\n", plan.ConfigBaseCount)
					fmt.Fprintf(w, "Applied Hacks:     %d\n", plan.AppliedHackCount)
					fmt.Fprintln(w, "\nTo execute migration, run: anx state migrate --apply")
				})
			}

			// Apply migration
			if appCtx.Store == nil || appCtx.Store.DB() == nil {
				return fmt.Errorf("state store database is unavailable")
			}

			result, err := sqlite.ApplyMigration(cmd.Context(), stateDir, appCtx.Store.DB())
			if err != nil {
				return fmt.Errorf("applying migration: %w", err)
			}

			return emit(cmd, appCtx, result, func(w io.Writer) {
				fmt.Fprintln(w, "State Migration Completed Successfully")
				fmt.Fprintln(w, "=====================================")
				fmt.Fprintf(w, "Source:    %s\n", result.SourcePath)
				fmt.Fprintf(w, "Backup:    %s\n", result.BackupPath)
				fmt.Fprintf(w, "Resources: %d\n", result.ResourceCount)
				fmt.Fprintf(w, "Bases:     %d\n", result.ConfigBaseCount)
				fmt.Fprintf(w, "Hacks:     %d\n", result.AppliedHackCount)
			})
		},
	}

	cmd.Flags().BoolVar(&planOnly, "plan", false, "Preview migration without applying changes")
	cmd.Flags().BoolVar(&apply, "apply", false, "Execute atomic migration from state.json to SQLite")

	return cmd
}

func newStateBackupCmd(appCtx *AppContext) *cobra.Command {
	var outPath string

	cmd := &cobra.Command{
		Use:   "backup",
		Short: "Create an online, crash-consistent backup of anx.db using VACUUM INTO",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if appCtx.Store == nil || appCtx.Store.DB() == nil {
				return fmt.Errorf("state database is not open")
			}

			if outPath == "" {
				ts := time.Now().UTC().Format("20060102-150405")
				outPath = filepath.Join(appCtx.Platform.StateDir(), "backups", fmt.Sprintf("anx.%s.db", ts))
			}

			if appCtx.Flags.DryRun {
				return emit(cmd, appCtx, map[string]string{
					"action":  "backup",
					"target":  outPath,
					"dry_run": "true",
				}, func(w io.Writer) {
					fmt.Fprintf(w, "[dry-run] Would backup database to: %s\n", outPath)
				})
			}

			if err := appCtx.Store.DB().BackupTo(cmd.Context(), outPath); err != nil {
				return fmt.Errorf("backing up database: %w", err)
			}

			return emit(cmd, appCtx, map[string]string{
				"status":      "success",
				"backup_path": outPath,
			}, func(w io.Writer) {
				fmt.Fprintf(w, "Backup created successfully at: %s\n", outPath)
			})
		},
	}

	cmd.Flags().StringVar(&outPath, "out", "", "Destination path for backup database")
	return cmd
}

func newStateVerifyCmd(appCtx *AppContext) *cobra.Command {
	return &cobra.Command{
		Use:   "verify",
		Short: "Verify database integrity and foreign key constraints",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if appCtx.Store == nil || appCtx.Store.DB() == nil {
				return fmt.Errorf("state database is not open")
			}

			if err := appCtx.Store.DB().CheckIntegrity(cmd.Context()); err != nil {
				return fmt.Errorf("integrity check failed: %w", err)
			}

			return emit(cmd, appCtx, map[string]string{
				"status":    "ok",
				"integrity": "pass",
			}, func(w io.Writer) {
				fmt.Fprintln(w, "Database integrity verification: PASS (quick_check and foreign_key_check OK)")
			})
		},
	}
}

func newStateExportCmd(appCtx *AppContext) *cobra.Command {
	return &cobra.Command{
		Use:   "export",
		Short: "Export all persisted state to formatted JSON",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if appCtx.Store == nil {
				return fmt.Errorf("state store is not available")
			}

			type exportedState struct {
				SchemaVersion int            `json:"schema_version"`
				ActiveProfile string         `json:"active_profile"`
				ActiveTheme   string         `json:"active_theme,omitempty"`
				RepoRoot      string         `json:"repo_root,omitempty"`
				Resources     map[string]any `json:"resources"`
				ConfigBases   map[string]any `json:"config_bases"`
				AppliedHacks  map[string]any `json:"applied_hacks"`
			}

			exp := exportedState{
				SchemaVersion: 1,
				ActiveProfile: appCtx.Store.ActiveProfile(),
				ActiveTheme:   appCtx.Store.ActiveTheme(),
				RepoRoot:      appCtx.Store.RepoRoot(),
				Resources:     make(map[string]any),
				ConfigBases:   make(map[string]any),
				AppliedHacks:  make(map[string]any),
			}

			for id, rec := range appCtx.Store.AllResources() {
				exp.Resources[id] = rec
			}

			if db := appCtx.Store.DB(); db != nil {
				bases, _ := db.AllConfigBases(cmd.Context())
				for id, b := range bases {
					exp.ConfigBases[id] = b
				}
				hacks, _ := db.AllAppliedHacks(cmd.Context())
				for id, h := range hacks {
					exp.AppliedHacks[id] = h
				}
			}

			data, err := json.MarshalIndent(exp, "", "  ")
			if err != nil {
				return fmt.Errorf("marshaling state export: %w", err)
			}

			fmt.Fprintln(cmd.OutOrStdout(), string(data))
			return nil
		},
	}
}
