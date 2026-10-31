package providers

import (
	"bytes"
	"context"
	"errors"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"text/template"
	"time"

	"github.com/arturonavax/env/internal/domain"
	"github.com/arturonavax/env/internal/platform"
	"github.com/arturonavax/env/internal/security"
)

// Provider abstracts package detection, installation, and removal operations.
type Provider interface {
	Name() string
	IsAvailable(ctx context.Context) bool
	IsInstalled(ctx context.Context, pkg domain.ManagedResource) (bool, string, error)
	Install(ctx context.Context, pkg domain.ManagedResource) error
	Uninstall(ctx context.Context, pkg domain.ManagedResource) error
}

// BinaryProvider provides read-only binary lookup in PATH (useful for lightweight tests and unmanaged binary checks).
type BinaryProvider struct {
	runner *security.Runner
}

var _ Provider = (*BinaryProvider)(nil)

// NewBinaryProvider constructs a read-only BinaryProvider.
func NewBinaryProvider(runner *security.Runner) *BinaryProvider {
	if runner == nil {
		runner = security.NewRunner(10 * time.Second)
	}
	return &BinaryProvider{runner: runner}
}

func (b *BinaryProvider) Name() string { return "binary" }

func (b *BinaryProvider) IsAvailable(_ context.Context) bool { return true }

func (b *BinaryProvider) IsInstalled(ctx context.Context, pkg domain.ManagedResource) (bool, string, error) {
	bin := pkg.BinaryName
	if bin == "" {
		bin = pkg.ID
	}

	path, err := exec.LookPath(bin)
	if err != nil {
		return false, "", nil
	}

	ver := probeBinaryVersion(ctx, b.runner, path, nil)
	return true, ver, nil
}

func (b *BinaryProvider) Install(_ context.Context, _ domain.ManagedResource) error {
	return errors.New("read-only BinaryProvider cannot install packages; use Dispatcher")
}

func (b *BinaryProvider) Uninstall(_ context.Context, _ domain.ManagedResource) error {
	return errors.New("read-only BinaryProvider cannot uninstall packages; use Dispatcher")
}

func probeBinaryVersion(
	ctx context.Context,
	runner *security.Runner,
	binPath string,
	customArgs []string,
) string {
	attempts := [][]string{{"--version"}, {"version"}, {"-V"}}
	if len(customArgs) > 0 {
		attempts = [][]string{customArgs}
	}

	for _, args := range attempts {
		command, commandArgs := binPath, args
		if runtime.GOOS == "windows" {
			ext := strings.ToLower(filepath.Ext(binPath))
			if ext == ".cmd" || ext == ".bat" {
				command = "cmd.exe"
				commandArgs = append([]string{"/d", "/c", binPath}, args...)
			}
		}
		out, err := runner.RunWithInput(
			ctx,
			security.RunOptions{Timeout: 4 * time.Second},
			command,
			commandArgs...,
		)
		trimmed := strings.TrimSpace(out)
		if trimmed == "" && err != nil {
			continue
		}
		if semver := security.ExtractSemver(trimmed); semver != "" {
			return semver
		}
		if err == nil && trimmed != "" {
			firstLine, _, _ := strings.Cut(trimmed, "\n")
			return strings.TrimSpace(firstLine)
		}
	}

	return "installed"
}

type templateVars struct {
	Home     string
	LocalBin string
	OS       string
	Arch     string
	Distro   string
	RepoDir  string
}

func expandTemplateString(raw string, plat *platform.Info, repoDir string) string {
	if !strings.Contains(raw, "{{") {
		return plat.ExpandPath(raw)
	}
	tpl, err := template.New("spec").Option("missingkey=zero").Parse(raw)
	if err != nil {
		return plat.ExpandPath(raw)
	}
	var buf bytes.Buffer
	err = tpl.Execute(&buf, templateVars{
		Home:     plat.HomeDir(),
		LocalBin: plat.LocalBinDir(),
		OS:       plat.OS(),
		Arch:     plat.Arch(),
		Distro:   plat.Distribution(),
		RepoDir:  repoDir,
	})
	if err != nil {
		return plat.ExpandPath(raw)
	}
	return plat.ExpandPath(buf.String())
}
