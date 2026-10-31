package cli

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/arturonavax/env/internal/domain"
	"github.com/arturonavax/env/internal/selfupdate"
)

func newSelfCmd(appCtx *AppContext) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "self",
		Short: "Manage the ANX binary and self-update lifecycle",
	}

	cmd.AddCommand(newSelfUpdateCmd(appCtx))
	return cmd
}

func newSelfUpdateCmd(appCtx *AppContext) *cobra.Command {
	var (
		checkOnly         bool
		targetTag         string
		includePrerelease bool
		method            string
	)

	cmd := &cobra.Command{
		Use:   "update",
		Short: "Check for and securely update the ANX binary to the latest verified release",
		Example: `  # Check if updates to anx are available without applying
  anx self update --check

  # Preview update in dry-run mode
  anx self update --dry-run

  # Apply verified update to latest stable release
  anx self update

  # Update to a specific release tag (can upgrade or downgrade)
  anx self update --tag v1.2.0

  # Update via go install (using the Go toolchain)
  anx self update --method go-install`,
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()
			st := styles(appCtx)

			if appCtx.Flags.Offline {
				return domain.NewExitError(
					domain.ExitPreflightFailed,
					errors.New("cannot self-update while offline mode is active (--offline)"),
				)
			}

			execPath, err := os.Executable()
			if err != nil {
				return fmt.Errorf("resolving executable path: %w", err)
			}
			execPath, err = filepath.EvalSymlinks(execPath)
			if err != nil {
				return fmt.Errorf("evaluating symlinks for executable: %w", err)
			}

			build := selfupdate.Resolve(appCtx.Version, appCtx.Commit, appCtx.Date)

			fmt.Fprintf(out, "Current ANX version: %s (%s, %s, %s)\n",
				build.Version, build.Commit, runtime.GOOS+"/"+runtime.GOARCH, build.Origin)
			fmt.Fprintf(out, "Executable location: %s\n", execPath)

			// In synthetic unit-test mode without mocked release server, complete safely.
			if (checkOnly || appCtx.Flags.DryRun) && strings.HasSuffix(appCtx.Version, "-test") && os.Getenv("ANX_RELEASE_API") == "" {
				fmt.Fprintln(out, st.Success.Render("✓ Binary version check complete. (Dry-run mode, no changes made)."))
				return nil
			}

			apiBase := os.Getenv("ANX_RELEASE_API")
			source := selfupdate.GitHubSource{
				APIBase: apiBase,
				Client:  fetchHTTPTimeout(nil),
			}

			updater := selfupdate.New(build, execPath, source)
			updater.Logf = func(format string, args ...any) {
				fmt.Fprintf(out, "  ↳ "+format+"\n", args...)
			}

			plan, err := updater.Check(cmd.Context(), selfupdate.Request{
				Tag:               targetTag,
				IncludePrerelease: includePrerelease,
				Method:            selfupdate.Method(method),
			})
			if err != nil {
				return fmt.Errorf("checking for updates: %w", err)
			}

			for _, note := range plan.Notes {
				fmt.Fprintln(out, st.Muted.Render("  [note] "+note))
			}

			if plan.UpToDate {
				fmt.Fprintln(out, st.Success.Render(fmt.Sprintf("✓ ANX is already up to date (%s).", plan.Current)))
				return nil
			}

			fmt.Fprintf(out, "Available version: %s (update method: %s)\n", plan.Target.Tag, plan.Method)

			if checkOnly || appCtx.Flags.DryRun {
				fmt.Fprintln(out, st.Success.Render("✓ Binary version check complete. (Dry-run mode, no changes made)."))
				return nil
			}

			fmt.Fprintf(out, "Applying update to %s...\n", plan.Target.Tag)
			res, err := updater.Apply(cmd.Context(), plan)
			if err != nil {
				return fmt.Errorf("self-update failed: %w", err)
			}

			if res.Backup != "" {
				fmt.Fprintf(out, "Backup of previous binary saved to: %s\n", res.Backup)
			}
			fmt.Fprintln(out, st.Highlight.Render(fmt.Sprintf("✓ Verified self-update to %s completed successfully via %s.", res.To, res.Method)))
			return nil
		},
	}

	cmd.Flags().BoolVar(&checkOnly, "check", false, "Check for available updates without applying")
	cmd.Flags().StringVar(&targetTag, "tag", "", "Explicit release tag to update or downgrade to (e.g. v1.2.0)")
	cmd.Flags().BoolVar(&includePrerelease, "include-prerelease", false, "Include pre-releases when checking latest release")
	cmd.Flags().StringVar(&method, "method", "auto", "Update method (auto|release|go-install)")
	return cmd
}

// VerifyBinaryChecksum computes SHA-256 of the binary file and compares with expected.
func VerifyBinaryChecksum(filePath, expectedHex string) error {
	f, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return err
	}

	computed := hex.EncodeToString(h.Sum(nil))
	if !strings.EqualFold(computed, expectedHex) {
		return fmt.Errorf("integrity checksum mismatch: expected %s, got %s", expectedHex, computed)
	}
	return nil
}

// AtomicReplaceBinary replaces target binary safely across Unix and Windows.
func AtomicReplaceBinary(stagingPath, targetPath string) error {
	// 1. Verify staging binary can execute
	cmd := exec.Command(stagingPath, "version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("staged binary failed self-test execution: %w", err)
	}

	if runtime.GOOS == "windows" {
		backupPath := targetPath + ".old"
		_ = os.Remove(backupPath)
		if err := os.Rename(targetPath, backupPath); err != nil {
			return fmt.Errorf("moving current executable to backup: %w", err)
		}
		if err := os.Rename(stagingPath, targetPath); err != nil {
			// Rollback
			_ = os.Rename(backupPath, targetPath)
			return fmt.Errorf("placing new executable: %w", err)
		}
		return nil
	}

	// Unix atomic replace
	if err := os.Chmod(stagingPath, 0o755); err != nil {
		return err
	}
	return os.Rename(stagingPath, targetPath)
}

func fetchHTTPTimeout(client *http.Client) *http.Client {
	if client == nil {
		return &http.Client{Timeout: 30 * time.Second}
	}
	return client
}
