package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/arturonavax/env/internal/domain"
)

func packageCommands(appCtx *AppContext) []*cobra.Command {
	cmds := []*cobra.Command{
		newPlanCmd(appCtx),
		newApplyCmd(appCtx),
		newInstallCmd(appCtx),
		newUninstallCmd(appCtx),
		newResumeCmd(appCtx),
		newListCmd(appCtx),
		newSearchCmd(appCtx),
		newInfoCmd(appCtx),
		newProfileCmd(appCtx),
	}
	for _, c := range cmds {
		c.GroupID = "packages"
	}
	return cmds
}

func newPlanCmd(appCtx *AppContext) *cobra.Command {
	return &cobra.Command{
		Use:               "plan [packages...]",
		Short:             "Display the dry-run execution plan for the desired state",
		ValidArgsFunction: packageCompletion(appCtx),
		RunE: func(cmd *cobra.Command, args []string) error {
			plan, err := appCtx.InstallSvc.Plan(cmd.Context(), appCtx.Flags.Profile, args)
			if err != nil {
				return err
			}
			return emit(cmd, appCtx, plan, func(w io.Writer) { renderPlan(w, appCtx, plan) })
		},
	}
}

func renderPlan(w io.Writer, appCtx *AppContext, plan *domain.ExecutionPlan) {
	st := styles(appCtx)
	fmt.Fprintln(w, st.Header.Render(fmt.Sprintf("Execution Plan: %s (Profile: %s)", plan.ID, plan.Profile)))
	if len(plan.Steps) == 0 {
		fmt.Fprintln(w, "  Nothing to do: the machine already matches the desired state.")
		return
	}
	for _, step := range plan.Steps {
		tag := st.Highlight.Render(fmt.Sprintf("[%s]", step.Action))
		fmt.Fprintf(w, "  Step %02d: %s %s\n", step.ID, tag, step.Description)
	}
}

func newApplyCmd(appCtx *AppContext) *cobra.Command {
	return &cobra.Command{
		Use:               "apply [packages...]",
		Short:             "Execute the plan to bring this machine to the desired state",
		ValidArgsFunction: packageCompletion(appCtx),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runApply(cmd, appCtx, appCtx.Flags.Profile, args)
		},
	}
}

func runApply(cmd *cobra.Command, appCtx *AppContext, profile string, pkgs []string) error {
	plan, err := appCtx.InstallSvc.Plan(cmd.Context(), profile, pkgs)
	if err != nil {
		return err
	}
	if appCtx.Flags.ScriptMode {
		return renderPlanScript(cmd.OutOrStdout(), appCtx, plan)
	}
	plan.DryRun = appCtx.Flags.DryRun
	if appCtx.Flags.DryRun {
		return emit(cmd, appCtx, plan, func(w io.Writer) { renderPlan(w, appCtx, plan) })
	}
	if !appCtx.Flags.Quiet {
		fmt.Fprintf(cmd.OutOrStdout(), "Applying plan %s (%d steps)...\n", plan.ID, len(plan.Steps))
	}
	if err := appCtx.InstallSvc.Apply(cmd.Context(), plan); err != nil {
		return err
	}
	if len(pkgs) == 0 {
		_ = appCtx.Store.SetActiveProfile(plan.Profile)
	}

	if appCtx.Flags.IncludeHacks && appCtx.HacksSvc != nil {
		_, matched := appCtx.HacksSvc.Detect(cmd.Context())
		for _, hk := range matched {
			if !appCtx.Flags.Quiet {
				fmt.Fprintf(cmd.OutOrStdout(), "Applying matched hardware workstation hack: %s (%s)...\n", hk.ID, hk.Name)
			}
			_, _ = appCtx.HacksSvc.Apply(cmd.Context(), hk.ID)
		}
	}
	return nil
}

func newInstallCmd(appCtx *AppContext) *cobra.Command {
	return &cobra.Command{
		Use:               "install <package...>",
		Short:             "Install packages (and their declared dependencies)",
		Args:              cobra.MinimumNArgs(1),
		ValidArgsFunction: packageCompletion(appCtx),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runApply(cmd, appCtx, "", args)
		},
	}
}

func newUninstallCmd(appCtx *AppContext) *cobra.Command {
	var (
		systemUninstall bool
		purgeConfigs    bool
	)
	cmd := &cobra.Command{
		Use:               "uninstall [package...]",
		Aliases:           []string{"remove"},
		Short:             "Remove managed packages (or --system to teardown the anx platform)",
		Args: func(cmd *cobra.Command, args []string) error {
			if systemUninstall {
				return nil
			}
			if len(args) == 0 {
				return errors.New("requires at least 1 arg(s), or --system for full teardown")
			}
			return nil
		},
		ValidArgsFunction: packageCompletion(appCtx),
		RunE: func(cmd *cobra.Command, args []string) error {
			if systemUninstall {
				return runSystemUninstall(cmd, appCtx, purgeConfigs)
			}
			if appCtx.Flags.DryRun {
				fmt.Fprintf(cmd.OutOrStdout(), "dry-run: would uninstall %s\n", strings.Join(args, ", "))
				return nil
			}
			if err := appCtx.InstallSvc.Uninstall(cmd.Context(), args); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Uninstalled %s\n", strings.Join(args, ", "))
			return nil
		},
	}
	cmd.Flags().BoolVar(&systemUninstall, "system", false, "Teardown and uninstall the anx platform and binaries from this machine")
	cmd.Flags().BoolVar(&purgeConfigs, "purge", false, "Purge configuration directories during system uninstallation")
	return cmd
}

func runSystemUninstall(cmd *cobra.Command, appCtx *AppContext, purge bool) error {
	ctx := cmd.Context()
	out := cmd.OutOrStdout()
	st := styles(appCtx)

	if appCtx.Flags.DryRun {
		fmt.Fprintln(out, st.Header.Render("ANX System Uninstallation Plan (Dry-Run):"))
		fmt.Fprintf(out, "  • Binaries:       %s (anx, safe-suspend, aliases)\n", appCtx.Platform.LocalBinDir())
		fmt.Fprintf(out, "  • State & tools:  %s\n", appCtx.Platform.StateDir())
		if purge {
			fmt.Fprintf(out, "  • Configurations: %s (purge requested)\n", appCtx.Platform.ConfigDir())
		} else {
			fmt.Fprintf(out, "  • Configurations: %s (preserved)\n", appCtx.Platform.ConfigDir())
		}
		return nil
	}

	if !appCtx.Flags.Yes && isTerminal(cmd) {
		fmt.Fprintf(out, "%s Are you sure you want to completely uninstall ANX from this system? (y/N): ", st.Warning.Render("!"))
		var confirm string
		_, _ = fmt.Scanln(&confirm)
		confirm = strings.TrimSpace(strings.ToLower(confirm))
		if confirm != "y" && confirm != "yes" {
			fmt.Fprintln(out, "Uninstallation cancelled.")
			return nil
		}
	}

	repo := appCtx.RepoRoot
	if appCtx.Platform.OS() == "windows" {
		psScript := filepath.Join(repo, "scripts", "uninstall_anx.ps1")
		if isFile(psScript) {
			args := []string{"-NoProfile", "-ExecutionPolicy", "Bypass", "-File", psScript, "-Yes"}
			if purge {
				args = append(args, "-PurgeConfigs")
			}
			c := exec.CommandContext(ctx, "powershell.exe", args...)
			c.Stdout, c.Stderr = out, cmd.ErrOrStderr()
			return c.Run()
		}
	} else {
		shScript := filepath.Join(repo, "uninstall.sh")
		if isFile(shScript) {
			c := exec.CommandContext(ctx, "bash", shScript)
			c.Stdout, c.Stderr = out, cmd.ErrOrStderr()
			return c.Run()
		}
	}

	binDir := appCtx.Platform.LocalBinDir()
	for _, b := range []string{"anx", "aitools"} {
		target := filepath.Join(binDir, b)
		_ = os.Remove(target)
	}
	_ = os.RemoveAll(appCtx.Platform.StateDir())
	if purge {
		_ = os.RemoveAll(appCtx.Platform.ConfigDir())
	}

	fmt.Fprintf(out, "%s Successfully uninstalled ANX from this system.\n", st.Success.Render("✓"))
	return nil
}

func isFile(path string) bool {
	st, err := os.Stat(path)
	return err == nil && !st.IsDir()
}

func newResumeCmd(appCtx *AppContext) *cobra.Command {
	return &cobra.Command{
		Use:   "resume",
		Short: "Resume an interrupted apply (completed steps are skipped idempotently)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			pending, err := appCtx.InstallSvc.PendingTransaction()
			if err != nil {
				return err
			}
			if pending == nil {
				fmt.Fprintln(cmd.OutOrStdout(), "No interrupted transaction to resume.")
				return nil
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Resuming after transaction %s (%d steps completed)...\n", pending.ID, pending.CompletedSteps)
			_, err = appCtx.InstallSvc.Resume(cmd.Context())
			return err
		},
	}
}

type packageRow struct {
	ID          string `json:"id" yaml:"id"`
	Category    string `json:"category" yaml:"category"`
	Installed   bool   `json:"installed" yaml:"installed"`
	Version     string `json:"version,omitempty" yaml:"version,omitempty"`
	Strategy    string `json:"strategy" yaml:"strategy"`
	Description string `json:"description" yaml:"description"`
	Deprecated  bool   `json:"deprecated,omitempty" yaml:"deprecated,omitempty"`
	Orphaned    bool   `json:"orphaned,omitempty" yaml:"orphaned,omitempty"`
	Replacement string `json:"replacement,omitempty" yaml:"replacement,omitempty"`
}

func collectPackages(cmd *cobra.Command, appCtx *AppContext, filter func(string, string, string) bool) []packageRow {
	rows := []packageRow{}
	catalogIDs := make(map[string]bool)

	for _, p := range appCtx.Catalog.ListPackages() {
		catalogIDs[p.ID] = true
		if filter != nil && !filter(p.ID, p.Name, p.Description+" "+p.Category) {
			continue
		}
		installed, ver, _ := appCtx.Provider.IsInstalled(cmd.Context(), p)
		rows = append(rows, packageRow{
			ID: p.ID, Category: p.Category, Installed: installed, Version: ver,
			Strategy: string(p.Strategy), Description: p.Description,
			Deprecated: p.Deprecated, Replacement: p.Replacement,
		})
	}

	// Also surface orphaned packages tracked in state.json
	if appCtx.Store != nil {
		desiredIDs := make([]string, 0, len(catalogIDs))
		for id := range catalogIDs {
			desiredIDs = append(desiredIDs, id)
		}
		orphans := appCtx.Store.DetectOrphaned(desiredIDs)
		for _, o := range orphans {
			if filter != nil && !filter(o.ResourceID, o.ResourceID, "orphaned") {
				continue
			}
			rows = append(rows, packageRow{
				ID: o.ResourceID, Category: "orphaned", Installed: true, Version: o.Version,
				Strategy: o.Strategy, Description: "Orphaned package tracked in state.json",
				Orphaned: true,
			})
		}
	}

	sort.Slice(rows, func(i, j int) bool { return rows[i].ID < rows[j].ID })
	return rows
}

func printRows(w io.Writer, appCtx *AppContext, rows []packageRow) {
	st := styles(appCtx)
	for _, r := range rows {
		state := st.Error.Render("missing")
		if r.Installed {
			state = st.Success.Render("installed " + r.Version)
		}
		badge := ""
		if r.Orphaned {
			badge = " " + st.Error.Render("[orphaned]")
		} else if r.Deprecated {
			badge = " " + st.Warning.Render("[deprecated]")
		}
		fmt.Fprintf(w, "  %-18s %-14s %s%s\n", r.ID, r.Category, state, badge)
	}
}

func newListCmd(appCtx *AppContext) *cobra.Command {
	var (
		onlyMissing    bool
		onlyDeprecated bool
		onlyOrphaned   bool
	)
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List catalog packages with their installation state",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			rows := collectPackages(cmd, appCtx, nil)
			if onlyMissing || onlyDeprecated || onlyOrphaned {
				filtered := rows[:0]
				for _, r := range rows {
					if onlyMissing && r.Installed {
						continue
					}
					if onlyDeprecated && !r.Deprecated {
						continue
					}
					if onlyOrphaned && !r.Orphaned {
						continue
					}
					filtered = append(filtered, r)
				}
				rows = filtered
			}
			return emit(cmd, appCtx, rows, func(w io.Writer) { printRows(w, appCtx, rows) })
		},
	}
	cmd.Flags().BoolVar(&onlyMissing, "missing", false, "Show only packages that are not installed")
	cmd.Flags().BoolVar(&onlyDeprecated, "deprecated", false, "Show only deprecated packages")
	cmd.Flags().BoolVar(&onlyOrphaned, "orphaned", false, "Show only orphaned packages tracked in state")
	return cmd
}

func newSearchCmd(appCtx *AppContext) *cobra.Command {
	return &cobra.Command{
		Use:   "search <term>",
		Short: "Search the declarative catalog by id, name, category or description",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			term := strings.ToLower(args[0])
			rows := collectPackages(cmd, appCtx, func(id, name, desc string) bool {
				return strings.Contains(strings.ToLower(id+" "+name+" "+desc), term)
			})
			return emit(cmd, appCtx, rows, func(w io.Writer) { printRows(w, appCtx, rows) })
		},
	}
}

func newInfoCmd(appCtx *AppContext) *cobra.Command {
	return &cobra.Command{
		Use:               "info <package>",
		Short:             "Show a package definition, strategies and resolved install source",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: packageCompletion(appCtx),
		RunE: func(cmd *cobra.Command, args []string) error {
			spec, ok := appCtx.Catalog.GetPackageSpec(args[0])
			if !ok {
				return domain.NewExitError(domain.ExitUsageError, fmt.Errorf("unknown package %q", args[0]))
			}
			return emit(cmd, appCtx, spec, func(w io.Writer) {
				st := styles(appCtx)
				fmt.Fprintln(w, st.Header.Render(spec.Name+" ("+spec.ID+")"))
				fmt.Fprintf(w, "  %s\n  category: %s  binary: %s  depends_on: %v\n", spec.Description, spec.Category, spec.BinaryName, spec.DependsOn)
				for i, s := range spec.Strategies {
					fmt.Fprintf(w, "  tier %d: %s\n", i+1, s.Type)
				}
			})
		},
	}
}

func newProfileCmd(appCtx *AppContext) *cobra.Command {
	cmd := &cobra.Command{Use: "profile", Short: "Inspect and select configuration profiles"}

	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List available profiles",
		Args:  cobra.NoArgs,
		RunE: func(c *cobra.Command, _ []string) error {
			profiles := appCtx.Catalog.ListProfiles()
			return emit(c, appCtx, profiles, func(w io.Writer) {
				active := appCtx.Store.ActiveProfile()
				for _, p := range profiles {
					mark := " "
					if p.Name == active {
						mark = "*"
					}
					fmt.Fprintf(w, " %s %-15s : %s\n", mark, p.Name, p.Description)
				}
			})
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:               "show <name>",
		Short:             "Show the fully resolved packages of a profile (including inherited ones)",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: profileCompletion(appCtx),
		RunE: func(c *cobra.Command, args []string) error {
			pkgs, err := appCtx.Catalog.ResolveProfilePackages(args[0])
			if err != nil {
				return domain.NewExitError(domain.ExitUsageError, err)
			}
			return emit(c, appCtx, pkgs, func(w io.Writer) { fmt.Fprintln(w, strings.Join(pkgs, "\n")) })
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:               "use <name>",
		Short:             "Persist the active profile",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: profileCompletion(appCtx),
		RunE: func(c *cobra.Command, args []string) error {
			if _, ok := appCtx.Catalog.GetProfile(args[0]); !ok {
				return domain.NewExitError(domain.ExitUsageError, errors.New("unknown profile "+args[0]))
			}
			return appCtx.Store.SetActiveProfile(args[0])
		},
	})
	return cmd
}

func renderPlanScript(w io.Writer, _ *AppContext, plan *domain.ExecutionPlan) error {
	fmt.Fprintln(w, "#!/usr/bin/env bash")
	fmt.Fprintf(w, "# ANX Apply Plan: %s (Profile: %s, Total Steps: %d)\n", plan.ID, plan.Profile, len(plan.Steps))
	fmt.Fprintln(w, "set -euo pipefail")

	if len(plan.Steps) == 0 {
		fmt.Fprintln(w, "echo '==> System is already in desired state. No actions needed.'")
		return nil
	}

	for i, st := range plan.Steps {
		fmt.Fprintf(w, "# Step %d: %s %s [%s]\n", i+1, st.Action, st.ResourceID, st.Provider)
		if st.Strategy == "github_release" || st.Strategy == "script" {
			fmt.Fprintf(w, "echo '--> Installing %s via %s...'\n", st.ResourceID, st.Strategy)
			fmt.Fprintf(w, "anx install %s\n\n", st.ResourceID)
			continue
		}
		switch {
		case strings.HasPrefix(st.Provider, "native:apt"):
			pkg := strings.TrimPrefix(st.Provider, "native:apt (")
			pkg = strings.TrimSuffix(pkg, ")")
			if pkg == "" {
				pkg = st.ResourceID
			}
			fmt.Fprintf(w, "sudo apt-get update -y && sudo apt-get install -y %s\n\n", pkg)
		case strings.HasPrefix(st.Provider, "native:pacman"):
			pkg := strings.TrimPrefix(st.Provider, "native:pacman (")
			pkg = strings.TrimSuffix(pkg, ")")
			if pkg == "" {
				pkg = st.ResourceID
			}
			fmt.Fprintf(w, "sudo pacman -S --noconfirm --needed %s\n\n", pkg)
		case strings.HasPrefix(st.Provider, "native:dnf"):
			pkg := strings.TrimPrefix(st.Provider, "native:dnf (")
			pkg = strings.TrimSuffix(pkg, ")")
			if pkg == "" {
				pkg = st.ResourceID
			}
			fmt.Fprintf(w, "sudo dnf install -y %s\n\n", pkg)
		case strings.HasPrefix(st.Provider, "native:brew"):
			pkg := strings.TrimPrefix(st.Provider, "native:brew (")
			pkg = strings.TrimSuffix(pkg, ")")
			if pkg == "" {
				pkg = st.ResourceID
			}
			fmt.Fprintf(w, "brew install %s\n\n", pkg)
		default:
			fmt.Fprintf(w, "anx install %s\n\n", st.ResourceID)
		}
	}
	return nil
}
