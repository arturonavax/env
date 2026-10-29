package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	aicli "github.com/arturonavax/env/internal/ai/cli"
	"github.com/arturonavax/env/internal/cli"
	"github.com/arturonavax/env/internal/domain"
)

var (
	// Version is injected at build time via -ldflags "-X main.Version=...".
	Version = "1.0.0"
	// Commit is injected at build time via -ldflags "-X main.Commit=...".
	Commit = "dev"
	// Date is injected at build time via -ldflags "-X main.Date=...".
	Date = "2026-10-29"
)

func main() {
	// Multicall dispatch: if binary is invoked as "aitools" (symlink or hardlink),
	// route execution directly to the AI productivity suite CLI tree.
	base := filepath.Base(os.Args[0])
	base = strings.TrimSuffix(base, ".exe")
	if base == "aitools" {
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()

		cmd := aicli.NewCommand(aicli.WithUse("aitools"))
		cmd.SetArgs(os.Args[1:])
		if err := cmd.ExecuteContext(ctx); err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			stop()
			os.Exit(domain.ResolveExitCode(err))
		}
		return
	}

	exitCode := cli.ExecuteCLI(os.Args[1:], os.Stdout, os.Stderr, Version, Commit, Date)
	os.Exit(exitCode)
}
