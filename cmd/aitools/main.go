// Command aitools is the standalone entrypoint of the AI productivity suite.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/arturonavax/env/internal/ai/cli"
	"github.com/arturonavax/env/internal/domain"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cmd := cli.NewCommand(cli.WithUse("aitools"))
	if err := cmd.ExecuteContext(ctx); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		stop()
		os.Exit(domain.ResolveExitCode(err))
	}
}
