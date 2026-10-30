package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)

func newDocCmd(rootCmd *cobra.Command, appCtx *AppContext) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "doc",
		Short: "Generate documentation, man pages, and shell reference manuals",
		Long: `Generate comprehensive offline documentation for anx commands and flags.

Supports standard Unix troff man pages (man1) and structured Markdown CLI references.`,
	}

	cmd.AddCommand(newDocManCmd(rootCmd, appCtx))
	cmd.AddCommand(newDocMarkdownCmd(rootCmd, appCtx))
	cmd.GroupID = "meta"

	return cmd
}

func newDocManCmd(rootCmd *cobra.Command, appCtx *AppContext) *cobra.Command {
	var (
		targetDir string
		install   bool
	)

	cmd := &cobra.Command{
		Use:   "man",
		Short: "Generate standard Unix troff man pages (section 1)",
		Long: `Generate section 1 man pages for anx and all subcommands using cobra/doc.

Pass --install to automatically install generated man pages into ~/.local/share/man/man1
so they are immediately accessible via 'man anx'.`,
		Example: `  # Generate man pages in ./man/man1
  anx doc man --dir ./man/man1

  # Automatically install man pages for the current user
  anx doc man --install`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			out := cmd.OutOrStdout()
			st := styles(appCtx)

			dir := targetDir
			if install || dir == "" {
				home := appCtx.Platform.HomeDir()
				dir = filepath.Join(home, ".local", "share", "man", "man1")
			}

			if err := os.MkdirAll(dir, 0o755); err != nil {
				return fmt.Errorf("creating man directory %q: %w", dir, err)
			}

			header := &doc.GenManHeader{
				Title:   "ANX",
				Section: "1",
				Manual:  "Anx User Manual",
				Source:  "anx " + appCtx.Version,
			}

			if err := doc.GenManTree(rootCmd, header, dir); err != nil {
				return fmt.Errorf("generating man pages: %w", err)
			}

			// Attempt updating mandb cache if available
			if mandbPath, err := exec.LookPath("mandb"); err == nil {
				_ = exec.Command(mandbPath, "-q").Run()
			}

			fmt.Fprintf(out, "%s Generated troff man pages in %s\n", st.Success.Render("✓"), dir)
			if install {
				fmt.Fprintf(out, "  You can now view the manual directly using: %s\n", st.Highlight.Render("man anx"))
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&targetDir, "dir", "", "Target directory for man pages (defaults to ./man or ~/.local/share/man/man1 with --install)")
	cmd.Flags().BoolVar(&install, "install", false, "Install man pages to user's ~/.local/share/man/man1 directory")

	return cmd
}

func newDocMarkdownCmd(rootCmd *cobra.Command, appCtx *AppContext) *cobra.Command {
	var targetDir string

	cmd := &cobra.Command{
		Use:   "markdown",
		Short: "Generate complete Markdown reference documentation for all commands",
		Long:  "Generate structured GitHub-flavored Markdown files for the entire anx CLI command hierarchy.",
		Example: `  # Generate markdown docs into docs/cli/
  anx doc markdown --dir docs/cli/`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			out := cmd.OutOrStdout()
			st := styles(appCtx)

			dir := targetDir
			if dir == "" {
				dir = filepath.Join(appCtx.RepoRoot, "docs", "cli")
			}

			if err := os.MkdirAll(dir, 0o755); err != nil {
				return fmt.Errorf("creating markdown doc directory %q: %w", dir, err)
			}

			if err := doc.GenMarkdownTree(rootCmd, dir); err != nil {
				return fmt.Errorf("generating markdown documentation: %w", err)
			}

			fmt.Fprintf(out, "%s Generated markdown CLI documentation in %s\n", st.Success.Render("✓"), dir)
			return nil
		},
	}

	cmd.Flags().StringVar(&targetDir, "dir", "", "Target directory for markdown docs (defaults to docs/cli/)")

	return cmd
}
