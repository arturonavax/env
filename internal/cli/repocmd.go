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

	"github.com/spf13/cobra"

	"github.com/arturonavax/env/internal/domain"
)

type repoStatusInfo struct {
	RepoRoot     string `json:"repo_root"`
	Exists       bool   `json:"exists"`
	HasFiles     bool   `json:"has_files"`
	HasManifests bool   `json:"has_manifests"`
	IsGit        bool   `json:"is_git"`
	RemoteURL    string `json:"remote_url,omitempty"`
	Branch       string `json:"branch,omitempty"`
	Clean        bool   `json:"clean"`
}

func newRepoCmd(appCtx *AppContext) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "repo",
		Short: "Manage the linked dotfiles repository and configuration assets",
		Long: `Inspect, clone, or configure the repository containing dotfiles, configurations, and assets.
Supports decoupled dotfiles repositories independent of the anx CLI binary engine.`,
	}

	cmd.AddCommand(
		newRepoStatusCmd(appCtx),
		newRepoCloneCmd(appCtx),
		newRepoSetCmd(appCtx),
	)

	return cmd
}

func newRepoStatusCmd(appCtx *AppContext) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Inspect the status of the currently linked dotfiles repository",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			root := appCtx.RepoRoot
			info := inspectRepoStatus(cmd.Context(), root)

			return emit(cmd, appCtx, info, func(w io.Writer) {
				st := styles(appCtx)
				fmt.Fprintln(w, st.Header.Render("ANX Dotfiles Repository Status"))
				fmt.Fprintf(w, "  • Location:      %s\n", info.RepoRoot)
				if !info.Exists {
					fmt.Fprintf(w, "  • State:         %s (directory does not exist)\n", st.Error.Render("[NOT FOUND]"))
					fmt.Fprintln(w, "                   Run 'anx repo clone' to initialize the dotfiles repository.")
					return
				}

				filesBadge := st.Success.Render("[YES]")
				if !info.HasFiles {
					filesBadge = st.Warning.Render("[NO] (files/ directory missing)")
				}
				fmt.Fprintf(w, "  • Has Dotfiles:  %s\n", filesBadge)

				manBadge := st.Success.Render("[YES]")
				if !info.HasManifests {
					manBadge = st.Muted.Render("[NO] (using embedded manifests)")
				}
				fmt.Fprintf(w, "  • Has Manifests: %s\n", manBadge)

				if info.IsGit {
					fmt.Fprintf(w, "  • Git Remote:    %s\n", info.RemoteURL)
					fmt.Fprintf(w, "  • Git Branch:    %s\n", info.Branch)
					cleanBadge := st.Success.Render("clean")
					if !info.Clean {
						cleanBadge = st.Warning.Render("uncommitted changes")
					}
					fmt.Fprintf(w, "  • Working Tree:  %s\n", cleanBadge)
				} else {
					fmt.Fprintf(w, "  • Version Control: %s\n", st.Muted.Render("Not a git repository"))
				}
			})
		},
	}
}

func newRepoCloneCmd(appCtx *AppContext) *cobra.Command {
	var (
		targetDir string
		depth     int
	)
	cmd := &cobra.Command{
		Use:   "clone [url]",
		Short: "Clone a remote dotfiles repository and link it to anx",
		Long: `Clones a remote git repository containing dotfiles and configuration templates.
Defaults to 'https://github.com/arturonavax/env.git' (or ANX_DOTFILES_URL) into '~/.local/share/anx/repo'.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			repoURL := "https://github.com/arturonavax/env.git"
			if envURL := os.Getenv("ANX_DOTFILES_URL"); envURL != "" {
				repoURL = envURL
			}
			if len(args) == 1 && strings.TrimSpace(args[0]) != "" {
				repoURL = strings.TrimSpace(args[0])
			}

			dest := targetDir
			if dest == "" {
				dest = filepath.Join(appCtx.Platform.HomeDir(), ".local", "share", "anx", "repo")
			}
			dest = appCtx.Platform.ExpandPath(dest)

			if isDir(filepath.Join(dest, "files")) {
				fmt.Fprintf(cmd.OutOrStdout(), "Repository already exists at %s.\n", dest)
				return linkRepoRoot(appCtx, dest, cmd.OutOrStdout())
			}

			if appCtx.Flags.DryRun {
				fmt.Fprintf(cmd.OutOrStdout(), "dry-run: would clone %s into %s (depth: %d)\n", repoURL, dest, depth)
				return nil
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Cloning dotfiles repository from %s into %s...\n", repoURL, dest)
			if err := cloneDotfilesRepo(cmd.Context(), repoURL, dest, depth); err != nil {
				return err
			}

			return linkRepoRoot(appCtx, dest, cmd.OutOrStdout())
		},
	}

	cmd.Flags().StringVar(&targetDir, "dir", "", "Target destination directory for cloned repository (default: ~/.local/share/anx/repo)")
	cmd.Flags().IntVar(&depth, "depth", 1, "Create a shallow clone with a history truncated to the specified number of commits")
	return cmd
}

func newRepoSetCmd(appCtx *AppContext) *cobra.Command {
	var force bool
	cmd := &cobra.Command{
		Use:   "set <path>",
		Short: "Link an existing local directory as the active dotfiles repository",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			target := appCtx.Platform.ExpandPath(strings.TrimSpace(args[0]))
			absPath, err := filepath.Abs(target)
			if err != nil {
				return fmt.Errorf("resolving path %q: %w", target, err)
			}

			if !isDir(absPath) {
				return domain.NewExitError(domain.ExitUsageError, fmt.Errorf("directory %s does not exist", absPath))
			}

			hasFiles := isDir(filepath.Join(absPath, "files"))
			if !hasFiles && !force {
				return domain.NewExitError(
					domain.ExitUsageError,
					fmt.Errorf("%s does not contain a 'files/' directory; use --force to link anyway", absPath),
				)
			}

			return linkRepoRoot(appCtx, absPath, cmd.OutOrStdout())
		},
	}
	cmd.Flags().BoolVarP(&force, "force", "f", false, "Link directory even if 'files/' directory is not detected")
	return cmd
}

func linkRepoRoot(appCtx *AppContext, path string, out io.Writer) error {
	appCtx.RepoRoot = path
	if appCtx.Store != nil {
		if err := appCtx.Store.SetRepoRoot(path); err != nil {
			return fmt.Errorf("persisting repo root to state: %w", err)
		}
	}
	if appCtx.SyncSvc != nil {
		appCtx.SyncSvc.SetRepoRoot(path)
	}

	fmt.Fprintf(out, "Linked dotfiles repository: %s\n", path)
	return nil
}

func inspectRepoStatus(ctx context.Context, root string) repoStatusInfo {
	info := repoStatusInfo{
		RepoRoot:     root,
		Exists:       isDir(root),
		HasFiles:     isDir(filepath.Join(root, "files")),
		HasManifests: isDir(filepath.Join(root, "manifests")),
		IsGit:        isDir(filepath.Join(root, ".git")),
	}

	if info.IsGit {
		if out, err := exec.CommandContext(ctx, "git", "-C", root, "remote", "get-url", "origin").Output(); err == nil {
			info.RemoteURL = strings.TrimSpace(string(out))
		}
		if out, err := exec.CommandContext(ctx, "git", "-C", root, "branch", "--show-current").Output(); err == nil {
			info.Branch = strings.TrimSpace(string(out))
		}
		if out, err := exec.CommandContext(ctx, "git", "-C", root, "status", "--porcelain").Output(); err == nil {
			info.Clean = len(strings.TrimSpace(string(out))) == 0
		}
	}

	return info
}

func cloneDotfilesRepo(ctx context.Context, repoURL, dest string, depth int) error {
	if _, err := exec.LookPath("git"); err != nil {
		return fmt.Errorf("git is required to clone dotfiles repository: %w (install git with 'anx install git')", err)
	}

	if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
		return fmt.Errorf("creating parent directory %s: %w", filepath.Dir(dest), err)
	}

	args := []string{"clone"}
	if depth > 0 {
		args = append(args, "--depth", fmt.Sprintf("%d", depth))
	}
	args = append(args, repoURL, dest)

	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git clone failed: %w", err)
	}

	return nil
}

func ensureDotfilesRepo(cmd *cobra.Command, appCtx *AppContext) error {
	if isDir(filepath.Join(appCtx.RepoRoot, "files")) {
		return nil
	}

	targetDir := filepath.Join(appCtx.Platform.HomeDir(), ".local", "share", "anx", "repo")
	defaultURL := os.Getenv("ANX_DOTFILES_URL")
	if defaultURL == "" {
		defaultURL = os.Getenv("ANX_GIT_URL")
	}
	if defaultURL == "" {
		defaultURL = "https://github.com/arturonavax/env.git"
	}

	if appCtx.Flags.Offline {
		return domain.NewExitError(
			domain.ExitPreflightFailed,
			fmt.Errorf("dotfiles repository missing at %q and offline mode is enabled; run 'anx repo clone' when online", appCtx.RepoRoot),
		)
	}

	shouldClone := appCtx.Flags.Yes
	if !shouldClone && isTerminal(cmd) {
		fmt.Fprintf(cmd.OutOrStdout(), "\nNo local dotfiles repository found for synchronization.\n")
		fmt.Fprintf(cmd.OutOrStdout(), "  Target directory: %s\n", targetDir)
		fmt.Fprintf(cmd.OutOrStdout(), "  Remote URL:       %s\n", defaultURL)
		fmt.Print("Would you like to clone the dotfiles repository now? [Y/n]: ")

		var response string
		_, _ = fmt.Scanln(&response)
		response = strings.TrimSpace(strings.ToLower(response))
		shouldClone = response == "" || response == "y" || response == "yes"
	}

	if !shouldClone {
		return domain.NewExitError(
			domain.ExitUsageError,
			errors.New("dotfiles repository is required for synchronization; configure it with 'anx repo clone' or set ANX_REPO"),
		)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Cloning dotfiles repository into %s...\n", targetDir)
	if err := cloneDotfilesRepo(cmd.Context(), defaultURL, targetDir, 1); err != nil {
		return fmt.Errorf("failed to clone dotfiles repository: %w", err)
	}

	return linkRepoRoot(appCtx, targetDir, cmd.OutOrStdout())
}
