package cli

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/arturonavax/env/internal/app"
	"github.com/arturonavax/env/internal/security"
)

// AuthCheckResult details the authorization outcome for git push operations.
type AuthCheckResult struct {
	ActiveUser   string   `json:"active_user"`
	RepoOwner    string   `json:"repo_owner"`
	AllowedUsers []string `json:"allowed_users"`
	IsAuthorized bool     `json:"is_authorized"`
	Reason       string   `json:"reason"`
}

func newCaptureCmd(appCtx *AppContext) *cobra.Command {
	var (
		wizardMode bool
		doCommit   bool
		doPush     bool
		branchName string
		commitMsg  string
		allConfigs bool
	)

	cmd := &cobra.Command{
		Use:   "capture [group|config-id]",
		Short: "Ingest live dotfiles back into the repository with optional git commit & push",
		Long: `Capture live user dotfiles from $HOME back into the repository files/ directory.

Supports both rapid non-interactive execution and interactive interview wizard mode.
Optionally stages, commits to a feature branch, and pushes to git origin with strict
ownership verification (only allowed repository owners/members can push).`,
		Example: `  # Rapidly capture all modified dotfiles back into the repository
  anx capture --all

  # Interactive wizard: review diffs and select files to capture
  anx capture --wizard

  # Capture, create a git commit in a branch, and push (if authorized)
  anx capture --all --commit --push

  # Capture specific group to custom branch
  anx capture terminal --branch sync/terminal-update --commit`,
		Args:              cobra.MaximumNArgs(1),
		ValidArgsFunction: groupCompletion(appCtx),
		RunE: func(cmd *cobra.Command, args []string) error {
			group := "all"
			if len(args) == 1 {
				group = args[0]
			} else if !allConfigs && !wizardMode {
				group = "all"
			}

			if wizardMode {
				return runCaptureWizard(cmd, appCtx, group)
			}

			return runCaptureNonInteractive(cmd, appCtx, group, doCommit, doPush, branchName, commitMsg)
		},
	}

	cmd.Flags().BoolVarP(&wizardMode, "wizard", "w", false, "Interactive step-by-step review wizard")
	cmd.Flags().BoolVarP(&allConfigs, "all", "a", true, "Capture all supported configurations")
	cmd.Flags().BoolVarP(&doCommit, "commit", "c", false, "Create a git commit on a feature branch")
	cmd.Flags().BoolVarP(&doPush, "push", "p", false, "Push the feature branch to remote origin (guard-protected)")
	cmd.Flags().StringVarP(&branchName, "branch", "b", "", "Target git branch name (defaults to sync/dotfiles-<timestamp>)")
	cmd.Flags().StringVarP(&commitMsg, "message", "m", "", "Commit message override")

	return cmd
}

func runCaptureNonInteractive(
	cmd *cobra.Command,
	appCtx *AppContext,
	group string,
	doCommit, doPush bool,
	branchName, commitMsg string,
) error {
	ctx := cmd.Context()
	out := cmd.OutOrStdout()
	st := styles(appCtx)

	opts := app.SyncOptions{
		Group:  group,
		Push:   true,
		DryRun: appCtx.Flags.DryRun,
	}

	results, err := appCtx.SyncSvc.Sync(ctx, opts)
	if err != nil {
		return err
	}

	if err := renderSync(cmd, appCtx, results); err != nil {
		return err
	}

	capturedCount := 0
	var allCapturedPaths []string
	for _, r := range results {
		if r.Action == "captured" || r.Action == "drift" {
			capturedCount += len(r.CapturedPaths)
			allCapturedPaths = append(allCapturedPaths, r.CapturedPaths...)
		}
	}

	if capturedCount == 0 {
		if !appCtx.Flags.Quiet {
			fmt.Fprintln(out, st.Muted.Render("No dotfiles were modified; repository is already up to date."))
		}
		return nil
	}

	if appCtx.Flags.ScriptMode {
		fmt.Fprintln(out, "#!/usr/bin/env bash\n# ANX Capture Script\nset -euo pipefail")
		bName := branchName
		if bName == "" {
			bName = "sync/dotfiles-update"
		}
		cMsg := commitMsg
		if cMsg == "" {
			cMsg = "feat(config): capture live dotfiles into repository"
		}
		fmt.Fprintf(out, "git checkout -b %s\n", bName)
		fmt.Fprintf(out, "git add -- %s\n", strings.Join(allCapturedPaths, " "))
		fmt.Fprintf(out, "git commit -m %q\n", cMsg)
		if doPush {
			fmt.Fprintf(out, "git push -u origin %s\n", bName)
		}
		return nil
	}

	if doCommit || doPush {
		return executeGitCommitAndPush(cmd, appCtx, branchName, commitMsg, allCapturedPaths, doPush)
	}

	return nil
}

func runCaptureWizard(cmd *cobra.Command, appCtx *AppContext, group string) error {
	ctx := cmd.Context()
	out := cmd.OutOrStdout()
	st := styles(appCtx)
	reader := bufio.NewReader(cmd.InOrStdin())

	fmt.Fprintln(out, st.Header.Render("ANX Dotfile Ingestion Wizard"))
	fmt.Fprintln(out, st.Muted.Render("Reviewing differences between live host files and repository templates..."))

	diffs, err := appCtx.SyncSvc.Diff(ctx, group)
	if err != nil {
		return err
	}

	var driftConfigs []app.ConfigDiff
	for _, d := range diffs {
		if d.State == "drift" || d.State == "conflict" {
			driftConfigs = append(driftConfigs, d)
		}
	}

	if len(driftConfigs) == 0 {
		fmt.Fprintln(out, st.Success.Render("✓ All live dotfiles already match repository templates. Nothing to capture."))
		return nil
	}

	fmt.Fprintf(out, "\nDetected %d configuration(s) with live local modifications:\n", len(driftConfigs))
	var toCapture []string

	for i, d := range driftConfigs {
		summary := d.Summary
		if summary == "" {
			summary = d.State
		}
		fmt.Fprintf(out, "\n[%d/%d] %s (%s)\n", i+1, len(driftConfigs), st.Highlight.Render(d.ConfigID), summary)
		fmt.Fprintf(out, "      Live Target: %s\n", d.Target)

		fmt.Print("      Capture this configuration into repository? [Y/n/q]: ")
		ans, _ := reader.ReadString('\n')
		ans = strings.TrimSpace(strings.ToLower(ans))

		if ans == "q" {
			fmt.Fprintln(out, "Wizard aborted by user.")
			return nil
		}
		if ans == "" || ans == "y" || ans == "yes" {
			toCapture = append(toCapture, d.ConfigID)
		}
	}

	if len(toCapture) == 0 {
		fmt.Fprintln(out, "\nNo configurations selected for capture.")
		return nil
	}

	fmt.Fprintf(out, "\nCapturing %d configuration(s) into %s/files/...\n", len(toCapture), appCtx.RepoRoot)
	var wizardCapturedPaths []string
	for _, id := range toCapture {
		opts := app.SyncOptions{Group: id, Push: true}
		res, err := appCtx.SyncSvc.Sync(ctx, opts)
		if err != nil {
			fmt.Fprintf(out, "  %s Failed capturing %s: %v\n", st.Error.Render("✗"), id, err)
		} else {
			for _, r := range res {
				wizardCapturedPaths = append(wizardCapturedPaths, r.CapturedPaths...)
				fmt.Fprintf(out, "  %s %s: captured %d file(s)\n", st.Success.Render("✓"), r.ConfigID, r.Files)
			}
		}
	}

	// Offer Git commit
	fmt.Print("\nDo you want to create a git commit in a new branch? [Y/n]: ")
	ans, _ := reader.ReadString('\n')
	ans = strings.TrimSpace(strings.ToLower(ans))
	if ans != "" && ans != "y" && ans != "yes" {
		fmt.Fprintln(out, "Finished. Changes are saved in working tree.")
		return nil
	}

	defaultBranch := fmt.Sprintf("sync/dotfiles-%s", time.Now().Format("20060102-150405"))
	fmt.Fprintf(out, "Branch name [%s]: ", defaultBranch)
	branch, _ := reader.ReadString('\n')
	branch = strings.TrimSpace(branch)
	if branch == "" {
		branch = defaultBranch
	}

	// Authorization Check for Push
	auth := checkGitPushAuthorization(ctx, appCtx)
	fmt.Fprintf(out, "\nGit Push Authorization Status: ")
	if auth.IsAuthorized {
		fmt.Fprintf(out, "%s (user: %s, repo owner: %s)\n", st.Success.Render("Authorized"), auth.ActiveUser, auth.RepoOwner)
	} else {
		fmt.Fprintf(out, "%s (%s)\n", st.Warning.Render("Restricted"), auth.Reason)
	}

	doPush := false
	if auth.IsAuthorized {
		fmt.Printf("Push branch '%s' to remote origin? [y/N]: ", branch)
		pAns, _ := reader.ReadString('\n')
		pAns = strings.TrimSpace(strings.ToLower(pAns))
		if pAns == "y" || pAns == "yes" {
			doPush = true
		}
	} else {
		fmt.Fprintln(out, st.Muted.Render("Remote push is disabled because you are not listed as the repository owner/member."))
	}

	return executeGitCommitAndPush(cmd, appCtx, branch, "", wizardCapturedPaths, doPush)
}

func executeGitCommitAndPush(
	cmd *cobra.Command,
	appCtx *AppContext,
	branchName, commitMsg string,
	capturedPaths []string,
	doPush bool,
) error {
	ctx := cmd.Context()
	out := cmd.OutOrStdout()
	st := styles(appCtx)
	repo := appCtx.RepoRoot

	if branchName == "" {
		branchName = fmt.Sprintf("sync/dotfiles-%s", time.Now().Format("20060102-150405"))
	}
	if commitMsg == "" {
		commitMsg = "feat(config): capture live dotfiles into repository"
	}

	if appCtx.Flags.DryRun {
		fmt.Fprintf(out, "[dry-run] Would create branch %s and stage %d captured files:\n", branchName, len(capturedPaths))
		for _, p := range capturedPaths {
			fmt.Fprintf(out, "  [dry-run] git add -- %s\n", p)
		}
		if doPush {
			fmt.Fprintf(out, "  [dry-run] Would push branch %s to origin\n", branchName)
		}
		return nil
	}

	if len(capturedPaths) == 0 {
		return nil
	}

	// Secret scan every captured file before staging
	for _, relPath := range capturedPaths {
		fullPath := filepath.Join(repo, relPath)
		data, err := os.ReadFile(fullPath)
		if err == nil {
			findings := security.ScanForSecrets(string(data))
			if len(findings) > 0 {
				return fmt.Errorf("security guardrail: secret detected in %s (%s, line %d); commit and push aborted to prevent credentials leakage",
					relPath, findings[0].RuleID, findings[0].Line)
			}
		}
	}

	// 1. Create and switch to feature branch
	if bErr := exec.CommandContext(ctx, "git", "-C", repo, "checkout", "-b", branchName).Run(); bErr != nil {
		// If branch exists, just checkout
		_ = exec.CommandContext(ctx, "git", "-C", repo, "checkout", branchName).Run()
	}

	// 2. Stage ONLY the exact allowlisted captured files
	addArgs := append([]string{"-C", repo, "add", "--"}, capturedPaths...)
	if aErr := exec.CommandContext(ctx, "git", addArgs...).Run(); aErr != nil {
		return fmt.Errorf("staging captured files: %w", aErr)
	}

	// 3. Commit
	cCmd := exec.CommandContext(ctx, "git", "-C", repo, "commit", "-m", commitMsg)
	cOut, cErr := cCmd.CombinedOutput()
	if cErr != nil && !strings.Contains(string(cOut), "nothing to commit") {
		return fmt.Errorf("git commit failed: %s (%w)", string(cOut), cErr)
	}

	fmt.Fprintf(out, "%s Created git commit on branch %s\n", st.Success.Render("✓"), st.Highlight.Render(branchName))

	// 4. Push if requested
	if doPush {
		auth := checkGitPushAuthorization(ctx, appCtx)
		if !auth.IsAuthorized {
			fmt.Fprintf(out, "%s %s\n", st.Warning.Render("[Push Guard]"), auth.Reason)
			fmt.Fprintf(out, "  Commit remains safely stored in local branch '%s'. Push was omitted.\n", branchName)
			return nil
		}

		fmt.Fprintf(out, "Pushing branch %s to origin...\n", branchName)
		pCmd := exec.CommandContext(ctx, "git", "-C", repo, "push", "-u", "origin", branchName)
		pOut, pErr := pCmd.CombinedOutput()
		if pErr != nil {
			return fmt.Errorf("git push failed: %s (%w)", string(pOut), pErr)
		}
		fmt.Fprintf(out, "%s Successfully pushed branch %s to origin\n", st.Success.Render("✓"), branchName)
	}

	return nil
}

func checkGitPushAuthorization(ctx context.Context, appCtx *AppContext) AuthCheckResult {
	repo := appCtx.RepoRoot
	v := appCtx.Viper

	// 1. Determine origin owner
	cmd := exec.CommandContext(ctx, "git", "-C", repo, "remote", "get-url", "origin")
	out, err := cmd.Output()
	originURL := strings.TrimSpace(string(out))

	owner := v.GetString("repo_owner")
	if owner == "" && err == nil && originURL != "" {
		owner = parseGitRepoOwner(originURL)
	}

	allowedConfig := v.GetStringSlice("allowed_push_users")
	allowed := make([]string, 0, len(allowedConfig)+1)
	if owner != "" {
		allowed = append(allowed, owner)
	}
	for _, u := range allowedConfig {
		u = strings.TrimSpace(u)
		if u != "" && !slices.Contains(allowed, u) {
			allowed = append(allowed, u)
		}
	}

	// 2. Identify active user
	var activeUser string
	if ghPath, err := exec.LookPath("gh"); err == nil {
		ghCmd := exec.CommandContext(ctx, ghPath, "api", "user", "-q", ".login")
		if ghOut, err := ghCmd.Output(); err == nil {
			activeUser = strings.TrimSpace(string(ghOut))
		}
	}
	if activeUser == "" {
		gitUserCmd := exec.CommandContext(ctx, "git", "-C", repo, "config", "user.name")
		if gitOut, err := gitUserCmd.Output(); err == nil {
			activeUser = strings.TrimSpace(string(gitOut))
		}
	}

	res := AuthCheckResult{
		ActiveUser:   activeUser,
		RepoOwner:    owner,
		AllowedUsers: allowed,
	}

	if len(allowed) == 0 {
		res.IsAuthorized = true
		res.Reason = "no ownership restrictions configured"
		return res
	}

	for _, a := range allowed {
		if strings.EqualFold(activeUser, a) {
			res.IsAuthorized = true
			res.Reason = fmt.Sprintf("active user %q is an authorized owner/member", activeUser)
			return res
		}
	}

	res.IsAuthorized = false
	res.Reason = fmt.Sprintf("active user %q is not authorized to push to origin (allowed: %s)", activeUser, strings.Join(allowed, ", "))
	return res
}

func parseGitRepoOwner(remoteURL string) string {
	remoteURL = strings.TrimSpace(remoteURL)
	remoteURL = strings.TrimSuffix(remoteURL, ".git")

	// HTTP/HTTPS format: https://github.com/owner/repo
	if strings.HasPrefix(remoteURL, "http://") || strings.HasPrefix(remoteURL, "https://") {
		trimmed := strings.TrimPrefix(remoteURL, "https://")
		trimmed = strings.TrimPrefix(trimmed, "http://")
		parts := strings.Split(strings.Trim(trimmed, "/"), "/")
		if len(parts) >= 2 {
			return parts[1]
		}
		return ""
	}

	// SSH format: git@github.com:owner/repo
	if strings.Contains(remoteURL, ":") {
		parts := strings.Split(remoteURL, ":")
		if len(parts) == 2 {
			pathParts := strings.Split(strings.Trim(parts[1], "/"), "/")
			if len(pathParts) >= 2 {
				return pathParts[0]
			}
		}
	}

	return ""
}
