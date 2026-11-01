package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/arturonavax/env/internal/app"
	"github.com/arturonavax/env/internal/domain"
	"github.com/arturonavax/env/internal/security"
)

// UpdateCheckSummary holds the status of updates across repo, tools, and configs.
type UpdateCheckSummary struct {
	RepoBehind    int      `json:"repo_behind"`
	RepoBranch    string   `json:"repo_branch"`
	RepoClean     bool     `json:"repo_clean"`
	ToolsTotal    int      `json:"tools_total"`
	ConfigDrifts  int      `json:"config_drifts"`
	DriftedFiles  []string `json:"drifted_files,omitempty"`
	MatchedHacks  int      `json:"matched_hacks"`
	CheckDuration string   `json:"check_duration"`
}

func newUpdateCmd(appCtx *AppContext) *cobra.Command {
	var fetchRemote bool

	cmd := &cobra.Command{
		Use:   "update",
		Short: "Check for updates across repository, personal tools, and dotfile configurations",
		Long: `Inspect repository commits, personal tools, and dotfile configurations for available updates.

This command is safe and non-destructive: it inspects available updates without modifying the system.
Works seamlessly from any working directory (CWD-independent) by resolving the canonical repository root.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			start := time.Now()
			ctx := cmd.Context()
			st := styles(appCtx)
			repo := appCtx.RepoRoot

			summary := UpdateCheckSummary{
				RepoClean: true,
			}

			renderSecuritySealBanner(cmd.OutOrStdout(), appCtx)

			// 1. Check Git Repository
			if isGitRepo(repo) {
				if curBranch, err := gitCurrentBranch(ctx, repo); err == nil {
					summary.RepoBranch = curBranch
				}
				summary.RepoClean = isGitClean(ctx, repo)

				if fetchRemote && !appCtx.Flags.Offline {
					fetchCtx, cancel := context.WithTimeout(ctx, 6*time.Second)
					_ = exec.CommandContext(fetchCtx, "git", "-C", repo, "fetch", "--quiet").Run()
					cancel()
				}

				if behind, err := gitBehindCount(ctx, repo); err == nil {
					summary.RepoBehind = behind
				}
			}

			// 2. Check Personal Tools
			if appCtx.ToolsSvc != nil {
				tools := appCtx.ToolsSvc.List(ctx)
				summary.ToolsTotal = len(tools)
			}

			// 3. Check Dotfile Drifts
			if appCtx.SyncSvc != nil {
				diffs, err := appCtx.SyncSvc.Diff(ctx, "all")
				if err == nil {
					for _, d := range diffs {
						// d.State is "clean" when live and repository are synchronized.
						// Only count as drift if it differs, conflicts, or is missing.
						if d.State != "clean" && d.State != "synced" {
							summary.ConfigDrifts++
							sum := d.Summary
							if sum == "" {
								sum = d.State
							}
							summary.DriftedFiles = append(summary.DriftedFiles, fmt.Sprintf("%s (%s)", d.ConfigID, sum))
						}
					}
				}
			}

			// 4. Check Matched Hacks
			if appCtx.HacksSvc != nil {
				_, matches := appCtx.HacksSvc.Detect(ctx)
				summary.MatchedHacks = len(matches)
			}

			summary.CheckDuration = time.Since(start).Round(time.Millisecond).String()

			return emit(cmd, appCtx, summary, func(w io.Writer) {
				fmt.Fprintln(w, st.Header.Render("ANX Update Check"))
				fmt.Fprintf(w, "  • Repository Root:   %s\n", repo)
				if summary.RepoBranch != "" {
					behindMsg := st.Success.Render("up to date")
					if summary.RepoBehind > 0 {
						behindMsg = st.Warning.Render(fmt.Sprintf("%d commits behind remote", summary.RepoBehind))
					}
					cleanMsg := "clean"
					if !summary.RepoClean {
						cleanMsg = st.Warning.Render("uncommitted local changes")
					}
					fmt.Fprintf(w, "  • Git Branch:        %s [%s, %s]\n", summary.RepoBranch, behindMsg, cleanMsg)
				}
				fmt.Fprintf(w, "  • Personal Tools:    %d decoupled tools managed in ~/.local/share/anx/bin\n", summary.ToolsTotal)
				if summary.ConfigDrifts == 0 {
					fmt.Fprintf(w, "  • Dotfiles Status:   %s\n", st.Success.Render("all synchronized"))
				} else {
					fmt.Fprintf(w, "  • Dotfiles Status:   %s (%s)\n",
						st.Warning.Render(fmt.Sprintf("%d configs have drifts", summary.ConfigDrifts)),
						strings.Join(summary.DriftedFiles, ", "))
				}
				if summary.MatchedHacks > 0 {
					fmt.Fprintf(w, "  • Workstation Hacks: %s\n", st.Highlight.Render(fmt.Sprintf("%d hardware match(es) detected", summary.MatchedHacks)))
				}
				fmt.Fprintf(w, "\nChecked in %s. Run 'anx upgrade' to apply available updates.\n", summary.CheckDuration)
			})
		},
	}

	cmd.Flags().BoolVar(&fetchRemote, "fetch", true, "Fetch remote git references before checking")
	return cmd
}

func newUpgradeCmd(appCtx *AppContext) *cobra.Command {
	var (
		skipPull     bool
		skipTools    bool
		skipPackages bool
		skipSync     bool
		includeHacks bool
	)

	cmd := &cobra.Command{
		Use:   "upgrade",
		Short: "Upgrade environment: pull repo, update tools, apply package plans, and sync dotfiles",
		Long: `Perform an end-to-end upgrade of the workstation environment:

1. Fast-forward pull latest repository changes (if working tree is clean).
2. Upgrade personal decoupled tools in ~/.local/share/anx/bin.
3. Compute and apply package installation plan for the active profile.
4. Reconcile and synchronize dotfiles using 3-way merge.
5. (Optional) Apply matched workstation hardware hacks if --include-hacks is provided.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			out := cmd.OutOrStdout()
			st := styles(appCtx)
			repo := appCtx.RepoRoot

			isDryRun := appCtx.Flags.DryRun
			var stepErrors []error

			if isDryRun {
				fmt.Fprintln(out, st.Highlight.Render("Previewing ANX Environment Upgrade (Dry-Run Mode)..."))
			} else {
				fmt.Fprintln(out, st.Header.Render("Starting ANX Environment Upgrade..."))
			}

			if !appCtx.Flags.ScriptMode && !appCtx.Flags.Quiet {
				renderSecuritySealBanner(out, appCtx)
			}
			if err := verifySecuritySeal(cmd, appCtx); err != nil {
				return err
			}

			// Step 1: Git Pull (if in git and clean)
			if !skipPull && isGitRepo(repo) && !appCtx.Flags.Offline {
				if isDryRun {
					fmt.Fprintf(out, "  [1/4] %s Dry-run: would fast-forward pull repository (%s)\n", st.Highlight.Render("ℹ"), repo)
				} else if isGitClean(ctx, repo) {
					fmt.Fprintf(out, "  [1/4] Pulling latest repository changes (%s)...\n", repo)
					curBranch, _ := gitCurrentBranch(ctx, repo)
					hasUpstream := false
					if curBranch != "" {
						uCmd := exec.CommandContext(ctx, "git", "-C", repo, "rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{u}")
						if err := uCmd.Run(); err == nil {
							hasUpstream = true
						}
					}

					if !hasUpstream {
						checkRemote := exec.CommandContext(ctx, "git", "-C", repo, "rev-parse", "--verify", "origin/"+curBranch)
						if err := checkRemote.Run(); err == nil {
							pullCmd := exec.CommandContext(ctx, "git", "-C", repo, "pull", "--ff-only", "origin", curBranch)
							if pOut, err := pullCmd.CombinedOutput(); err != nil {
								fmt.Fprintf(out, "    %s git pull note: %s\n", st.Warning.Render("!"), strings.TrimSpace(string(pOut)))
							} else {
								fmt.Fprintf(out, "    %s Repository updated: %s\n", st.Success.Render("✓"), strings.TrimSpace(string(pOut)))
							}
						} else {
							fmt.Fprintf(out, "    %s Branch %q has no remote tracking upstream; skipping git pull.\n", st.Highlight.Render("ℹ"), curBranch)
						}
					} else {
						pullCmd := exec.CommandContext(ctx, "git", "-C", repo, "pull", "--ff-only")
						if pOut, err := pullCmd.CombinedOutput(); err != nil {
							outStr := strings.TrimSpace(string(pOut))
							if strings.Contains(outStr, "no tracking information") {
								fmt.Fprintf(out, "    %s Branch %q has no remote tracking upstream; skipping git pull.\n", st.Highlight.Render("ℹ"), curBranch)
							} else {
								errMsg := fmt.Errorf("git pull: %s", outStr)
								stepErrors = append(stepErrors, errMsg)
								fmt.Fprintf(out, "    %s git pull warning: %s\n", st.Warning.Render("!"), outStr)
							}
						} else {
							fmt.Fprintf(out, "    %s Repository updated: %s\n", st.Success.Render("✓"), strings.TrimSpace(string(pOut)))
						}
					}
				} else {
					fmt.Fprintf(out, "  [1/4] %s Repository has uncommitted local changes; skipping git pull to prevent conflicts.\n", st.Warning.Render("!"))
				}
			} else {
				fmt.Fprintln(out, "  [1/4] Skipping repository pull.")
			}

			// Step 2: Personal Tools Upgrade
			if !skipTools && appCtx.ToolsSvc != nil && appCtx.ToolsSvc.Registry() != nil {
				if isDryRun {
					fmt.Fprintln(out, "  [2/4] Dry-run: previewing personal tools update...")
					for _, t := range appCtx.ToolsSvc.Registry().List() {
						tStat, _ := appCtx.ToolsSvc.Get(ctx, t.ID)
						fmt.Fprintf(out, "    • %-16s [%s] current target: %s\n", t.ID, st.Highlight.Render("dry-run"), tStat.ActiveTarget)
					}
				} else {
					fmt.Fprintln(out, "  [2/4] Updating decoupled personal tools...")
					for _, t := range appCtx.ToolsSvc.Registry().List() {
						r, err := appCtx.ToolsSvc.Update(ctx, t.ID)
						if err != nil {
							stepErrors = append(stepErrors, fmt.Errorf("tool %s update: %w", t.ID, err))
							fmt.Fprintf(out, "    • %-16s [%s] %v\n", t.ID, st.Error.Render("failed"), err)
						} else {
							status := st.Success.Render(r.Status)
							if r.Status == "error" || r.Status == "failed" {
								stepErrors = append(stepErrors, fmt.Errorf("tool %s update reported: %s", r.ToolID, r.Message))
								status = st.Error.Render(r.Status)
							} else if r.Status == "skipped" {
								status = st.Muted.Render(r.Status)
							}
							fmt.Fprintf(out, "    • %-16s [%s] %s\n", r.ToolID, status, r.Message)
						}
					}
				}
			} else {
				fmt.Fprintln(out, "  [2/4] Skipping personal tools update.")
			}

			// Step 3: Package Plan and Apply
			if !skipPackages && appCtx.InstallSvc != nil {
				fmt.Fprintf(out, "  [3/4] Reconciling packages for profile '%s'...\n", appCtx.Flags.Profile)
				plan, err := appCtx.InstallSvc.Plan(ctx, appCtx.Flags.Profile, nil)
				if err != nil {
					return fmt.Errorf("failed to plan package installation: %w", err)
				}
				if len(plan.Steps) == 0 {
					fmt.Fprintf(out, "    %s All packages are already up to date (0 pending actions).\n", st.Success.Render("✓"))
				} else {
					fmt.Fprintf(out, "    Applying %d package actions...\n", len(plan.Steps))
					if !appCtx.Flags.DryRun {
						if appCtx.InstallSvc.PlanRequiresSudo(plan) {
							if !security.IsSudoAuthenticated() {
								if !isTerminal(cmd) {
									return domain.NewExitError(
										domain.ExitPreflightFailed,
										errors.New("administrator privileges (sudo) required; run 'sudo -v' prior to non-interactive execution"),
									)
								}
								fmt.Fprintln(out, "  Administrator privileges (sudo) required for package upgrade.")
								if err := security.AuthenticateSudoCLI(); err != nil {
									return fmt.Errorf("sudo authentication failed: %w", err)
								}
							}
							stopKeepalive := security.StartSudoKeepalive(ctx)
							defer stopKeepalive()
						}
						if err := appCtx.InstallSvc.Apply(ctx, plan); err != nil {
							return fmt.Errorf("package installation failed: %w", err)
						}
						fmt.Fprintf(out, "    %s Packages applied successfully.\n", st.Success.Render("✓"))
					} else {
						fmt.Fprintf(out, "    %s Dry-run: skipped applying %d actions.\n", st.Highlight.Render("ℹ"), len(plan.Steps))
					}
				}

				// Apply hacks if requested
				if (includeHacks || appCtx.Flags.IncludeHacks) && appCtx.HacksSvc != nil {
					_, matches := appCtx.HacksSvc.Detect(ctx)
					for _, hk := range matches {
						fmt.Fprintf(out, "    Applying matched hardware hack: %s...\n", hk.ID)
						if !appCtx.Flags.DryRun {
							if _, err := appCtx.HacksSvc.Apply(ctx, hk.ID); err != nil {
								fmt.Fprintf(out, "      %s Failed to apply hack %s: %v\n", st.Warning.Render("!"), hk.ID, err)
							} else {
								fmt.Fprintf(out, "      %s Applied %s\n", st.Success.Render("✓"), hk.ID)
							}
						}
					}
				}
			} else {
				fmt.Fprintln(out, "  [3/4] Skipping package reconciliation.")
			}

			// Step 4: Dotfiles Sync
			if !skipSync && appCtx.SyncSvc != nil {
				fmt.Fprintln(out, "  [4/4] Synchronizing dotfiles with 3-way merge...")
				syncOpts := app.SyncOptions{
					Group:  "all",
					DryRun: appCtx.Flags.DryRun,
					Force:  false,
				}
				results, err := appCtx.SyncSvc.Sync(ctx, syncOpts)
				if err != nil {
					return fmt.Errorf("dotfile sync failed: %w", err)
				}
				for _, r := range results {
					if r.Action != "unchanged" {
						fmt.Fprintf(out, "    • %-24s [%s]\n", r.ConfigID, st.Highlight.Render(r.Action))
					}
				}
				fmt.Fprintf(out, "    %s Dotfiles synchronized successfully.\n", st.Success.Render("✓"))
			} else {
				fmt.Fprintln(out, "  [4/4] Skipping dotfiles synchronization.")
			}

			if len(stepErrors) > 0 {
				fmt.Fprintln(out, st.Warning.Render(fmt.Sprintf("\n⚠ ANX Upgrade Finished with %d partial error(s):", len(stepErrors))))
				for _, err := range stepErrors {
					fmt.Fprintf(out, "  • %v\n", err)
				}
				return domain.NewExitError(domain.ExitRollbackOrPartial, fmt.Errorf("upgrade completed with %d errors", len(stepErrors)))
			}

			if isDryRun {
				fmt.Fprintln(out, st.Highlight.Render("\n✓ Dry-Run Upgrade Preview Completed (No changes applied)."))
				return nil
			}

			fmt.Fprintln(out, st.Success.Render("\n✓ ANX Environment Upgrade Completed Successfully!"))
			return nil
		},
	}

	cmd.Flags().BoolVar(&skipPull, "skip-pull", false, "Skip git repository pull")
	cmd.Flags().BoolVar(&skipTools, "skip-tools", false, "Skip decoupled personal tools update")
	cmd.Flags().BoolVar(&skipPackages, "skip-packages", false, "Skip package installations")
	cmd.Flags().BoolVar(&skipSync, "skip-sync", false, "Skip dotfiles synchronization")
	cmd.Flags().BoolVar(&includeHacks, "include-hacks", false, "Include matched hardware workstation hacks")

	return cmd
}

func isGitRepo(dir string) bool {
	if dir == "" {
		return false
	}
	st, err := os.Stat(filepath.Join(dir, ".git"))
	return err == nil && st != nil
}

func isGitClean(ctx context.Context, dir string) bool {
	cmd := exec.CommandContext(ctx, "git", "-C", dir, "diff-index", "--quiet", "HEAD", "--")
	return cmd.Run() == nil
}

func gitCurrentBranch(ctx context.Context, dir string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "-C", dir, "rev-parse", "--abbrev-ref", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func gitBehindCount(ctx context.Context, dir string) (int, error) {
	cmd := exec.CommandContext(ctx, "git", "-C", dir, "rev-list", "--count", "HEAD..@{u}")
	out, err := cmd.Output()
	if err != nil {
		return 0, err
	}
	var count int
	_, err = fmt.Sscanf(strings.TrimSpace(string(out)), "%d", &count)
	return count, err
}
