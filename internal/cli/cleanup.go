package cli

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

type cleanupItem struct {
	ID          string   `json:"id" yaml:"id"`
	Kind        string   `json:"kind" yaml:"kind"` // "orphan", "deprecated", "archive"
	Name        string   `json:"name" yaml:"name"`
	Reason      string   `json:"reason" yaml:"reason"`
	Replacement string   `json:"replacement,omitempty" yaml:"replacement,omitempty"`
	Paths       []string `json:"paths,omitempty" yaml:"paths,omitempty"`
}

type cleanupReport struct {
	TotalFound  int           `json:"total_found" yaml:"total_found"`
	TotalPurged int           `json:"total_purged" yaml:"total_purged"`
	DryRun      bool          `json:"dry_run" yaml:"dry_run"`
	Items       []cleanupItem `json:"items" yaml:"items"`
}

func newCleanupCmd(appCtx *AppContext) *cobra.Command {
	var (
		orphansOnly    bool
		deprecatedOnly bool
		archivesOnly   bool
	)

	cmd := &cobra.Command{
		Use:     "cleanup",
		Aliases: []string{"prune"},
		Short:   "Safely detect and prune orphaned, deprecated, and archived components",
		Long: `Detect and safely remove components that are no longer maintained, declared in active manifests,
or have been archived.

Tracks historical environment memory across:
  1. Orphaned packages and configs tracked in state.json but removed from manifests or active profile.
  2. Deprecated tools and packages recommended for safe purging.
  3. Legacy archived components (e.g. tmux, wdiff, litecli) with lingering host filesystem paths.

Use --dry-run to preview removals without touching the host filesystem or state.`,
		Example: `  # Preview all cleanup candidates without removing anything
  anx cleanup --dry-run

  # Interactively review and remove stale components
  anx cleanup

  # Automatically confirm and prune all orphaned components
  anx cleanup --yes --orphans-only`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runCleanup(cmd, appCtx, orphansOnly, deprecatedOnly, archivesOnly)
		},
	}

	cmd.Flags().BoolVar(&orphansOnly, "orphans-only", false, "Only prune orphaned packages tracked in state")
	cmd.Flags().BoolVar(&deprecatedOnly, "deprecated-only", false, "Only prune deprecated components")
	cmd.Flags().BoolVar(&archivesOnly, "archives-only", false, "Only prune decommissioned archive remnants from filesystem")

	return cmd
}

func runCleanup(cmd *cobra.Command, appCtx *AppContext, orphansOnly, deprecatedOnly, archivesOnly bool) error {
	ctx := cmd.Context()
	out := cmd.OutOrStdout()
	items := collectCleanupCandidates(ctx, appCtx, orphansOnly, deprecatedOnly, archivesOnly)

	report := cleanupReport{
		TotalFound: len(items),
		DryRun:     appCtx.Flags.DryRun,
		Items:      items,
	}

	if len(items) == 0 {
		if appCtx.Flags.Format != "text" {
			return emit(cmd, appCtx, report, func(io.Writer) {})
		}
		st := styles(appCtx)
		fmt.Fprintln(out, st.Success.Render("✓ No orphaned, deprecated, or archived components detected. Environment is clean."))
		return nil
	}

	if appCtx.Flags.Format != "text" && appCtx.Flags.DryRun {
		return emit(cmd, appCtx, report, func(io.Writer) {})
	}

	st := styles(appCtx)
	fmt.Fprintf(out, "%s\n", st.Header.Render(fmt.Sprintf("Detected %d cleanup candidate(s):", len(items))))
	for _, it := range items {
		var tag string
		switch it.Kind {
		case "orphan":
			tag = st.Error.Render("[Orphaned]")
		case "deprecated":
			tag = st.Warning.Render("[Deprecated]")
		case "archive":
			tag = st.Muted.Render("[Archived]")
		}
		fmt.Fprintf(out, "  %s %-16s %s\n", tag, it.ID, it.Reason)
		if it.Replacement != "" {
			fmt.Fprintf(out, "    Recommended replacement: %s\n", st.Highlight.Render(it.Replacement))
		}
		if len(it.Paths) > 0 {
			fmt.Fprintf(out, "    Target paths: %s\n", strings.Join(it.Paths, ", "))
		}
	}

	if appCtx.Flags.DryRun {
		fmt.Fprintf(out, "\n%s\n", st.Muted.Render("Dry-run preview: 0 changes made. Run 'anx cleanup' without --dry-run to apply."))
		return nil
	}

	// Confirm unless --yes is passed
	if !appCtx.Flags.Yes && isTerminal(cmd) {
		fmt.Fprintf(out, "\nProceed with pruning these %d components? [y/N]: ", len(items))
		reader := bufio.NewReader(cmd.InOrStdin())
		ans, _ := reader.ReadString('\n')
		ans = strings.TrimSpace(strings.ToLower(ans))
		if ans != "y" && ans != "yes" {
			fmt.Fprintln(out, "Cleanup aborted by user.")
			return nil
		}
	}

	// Execute pruning
	purged := 0
	for _, it := range items {
		switch it.Kind {
		case "orphan":
			_ = appCtx.InstallSvc.Uninstall(ctx, []string{it.ID})
			if err := appCtx.Store.RemoveResource(it.ID); err == nil {
				purged++
				if !appCtx.Flags.Quiet {
					fmt.Fprintf(out, "  %s Removed orphaned package state: %s\n", st.Success.Render("✓"), it.ID)
				}
			}
		case "deprecated":
			_ = appCtx.InstallSvc.Uninstall(ctx, []string{it.ID})
			_ = appCtx.Store.RemoveResource(it.ID)
			purged++
			if !appCtx.Flags.Quiet {
				fmt.Fprintf(out, "  %s Purged deprecated package: %s\n", st.Success.Render("✓"), it.ID)
			}
		case "archive":
			cleanedPaths := 0
			for _, p := range it.Paths {
				if err := os.RemoveAll(p); err == nil {
					cleanedPaths++
					if !appCtx.Flags.Quiet {
						fmt.Fprintf(out, "  %s Removed archive path: %s\n", st.Success.Render("✓"), p)
					}
				}
			}
			if cleanedPaths > 0 {
				purged++
			}
		}
	}

	report.TotalPurged = purged
	if appCtx.Flags.Format != "text" {
		return emit(cmd, appCtx, report, func(io.Writer) {})
	}

	fmt.Fprintf(out, "\n%s\n", st.Success.Render(fmt.Sprintf("✓ Cleanup complete: %d component(s) successfully pruned.", purged)))
	return nil
}

func collectCleanupCandidates(ctx context.Context, appCtx *AppContext, orphansOnly, deprecatedOnly, archivesOnly bool) []cleanupItem {
	items := make([]cleanupItem, 0, 16)
	includeAll := !orphansOnly && !deprecatedOnly && !archivesOnly

	// 1. Orphaned components: tracked in state.json but missing from catalog/profile
	if includeAll || orphansOnly {
		if appCtx.Store != nil && appCtx.Catalog != nil {
			allPkgs := appCtx.Catalog.ListPackages()
			desiredIDs := make([]string, 0, len(allPkgs))
			for _, p := range allPkgs {
				desiredIDs = append(desiredIDs, p.ID)
			}
			for _, c := range appCtx.Catalog.ListConfigurations() {
				desiredIDs = append(desiredIDs, c.ID)
			}
			orphans := appCtx.Store.DetectOrphaned(desiredIDs)
			for _, o := range orphans {
				var paths []string
				if o.InstalledPath != "" {
					paths = append(paths, o.InstalledPath)
				}
				items = append(items, cleanupItem{
					ID:     o.ResourceID,
					Kind:   "orphan",
					Name:   o.ResourceID,
					Reason: fmt.Sprintf("Installed at %s, but no longer declared in manifests or active profile", o.InstalledAt.Format("2006-01-02")),
					Paths:  paths,
				})
			}
		}
	}

	// 2. Deprecated / Purge-recommended components
	if includeAll || deprecatedOnly {
		if appCtx.Catalog != nil {
			for _, pkg := range appCtx.Catalog.ListPackages() {
				if pkg.Deprecated && pkg.RecommendPurge {
					installed, _, _ := appCtx.Provider.IsInstalled(ctx, pkg)
					if installed {
						items = append(items, cleanupItem{
							ID:          pkg.ID,
							Kind:        "deprecated",
							Name:        pkg.Name,
							Reason:      pkg.DeprecatedReason,
							Replacement: pkg.Replacement,
						})
					}
				}
			}
		}
	}

	// 3. Decommissioned archive remnants on host filesystem
	if includeAll || archivesOnly {
		if appCtx.Catalog != nil && appCtx.Platform != nil {
			home := appCtx.Platform.HomeDir()
			for _, arc := range appCtx.Catalog.ListArchives() {
				existingPaths := make([]string, 0, len(arc.CleanupPaths))
				for _, cleanPath := range arc.CleanupPaths {
					expanded := cleanPath
					if strings.HasPrefix(expanded, "~/") {
						expanded = filepath.Join(home, expanded[2:])
					}
					if _, err := os.Stat(expanded); err == nil {
						existingPaths = append(existingPaths, expanded)
					}
				}
				if len(existingPaths) > 0 {
					items = append(items, cleanupItem{
						ID:          arc.ID,
						Kind:        "archive",
						Name:        arc.Name,
						Reason:      arc.Reason,
						Replacement: arc.Replacement,
						Paths:       existingPaths,
					})
				}
			}
		}
	}

	return items
}
