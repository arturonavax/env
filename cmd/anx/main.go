package main

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/arturonavax/env/internal/cli"
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
	base := filepath.Base(os.Args[0])
	base = strings.TrimSuffix(base, ".exe")

	if base == "safe-suspend" || base == "suspend" {
		exitCode := cli.ExecuteCLI(append([]string{"sleep"}, os.Args[1:]...), os.Stdout, os.Stderr, Version, Commit, Date)
		os.Exit(exitCode)
	}

	if base == "aitools" {
		exitCode := cli.ExecuteCLI(append([]string{"ai"}, os.Args[1:]...), os.Stdout, os.Stderr, Version, Commit, Date)
		os.Exit(exitCode)
	}

	exitCode := cli.ExecuteCLI(os.Args[1:], os.Stdout, os.Stderr, Version, Commit, Date)
	os.Exit(exitCode)
}
