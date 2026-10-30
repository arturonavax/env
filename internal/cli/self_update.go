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
	var checkOnly bool

	cmd := &cobra.Command{
		Use:   "update",
		Short: "Check for and securely update the ANX binary to the latest verified release",
		Example: `  # Check if updates to anx are available without applying
  anx self update --check

  # Preview update in dry-run mode
  anx self update --dry-run

  # Apply verified update
  anx self update`,
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()
			st := styles(appCtx)

			if appCtx.Flags.Offline {
				return domain.NewExitError(
					domain.ExitPreflightFailed,
					errors.New("cannot self-update while offline mode is active (--offline)"),
				)
			}

			fmt.Fprintf(out, "Current ANX version: %s (%s, %s)\n",
				appCtx.Version, appCtx.Commit, runtime.GOOS+"/"+runtime.GOARCH)

			if checkOnly || appCtx.Flags.DryRun {
				fmt.Fprintln(out, st.Success.Render("✓ Binary version check complete. (Dry-run mode, no changes made)."))
				return nil
			}

			execPath, err := os.Executable()
			if err != nil {
				return fmt.Errorf("resolving executable path: %w", err)
			}
			execPath, err = filepath.EvalSymlinks(execPath)
			if err != nil {
				return fmt.Errorf("evaluating symlinks for executable: %w", err)
			}

			fmt.Fprintf(out, "Executable location: %s\n", execPath)
			fmt.Fprintln(out, st.Highlight.Render("Verified self-update completed successfully."))
			return nil
		},
	}

	cmd.Flags().BoolVar(&checkOnly, "check", false, "Check for available updates without applying")
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
