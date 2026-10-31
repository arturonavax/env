package gitflow

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/arturonavax/env/internal/security"
)

// ClipboardOptions configures clipboard detection and OSC-52 fallback behavior.
type ClipboardOptions struct {
	FallbackWriter io.Writer
	LookPath       func(string) (string, error)
	Getenv         func(string) string
	CommandRunner  func(ctx context.Context, bin string, args []string, stdin io.Reader) error
}

// CopyToClipboard copies text to the system clipboard using the active platform utility
// (wl-copy, xclip, xsel, pbcopy, clip.exe) or falls back to OSC-52 terminal escape sequences.
func CopyToClipboard(ctx context.Context, runner *security.Runner, text string) error {
	_, err := CopyToClipboardWithOptions(ctx, runner, text, ClipboardOptions{
		FallbackWriter: io.Discard,
	})
	return err
}

// CopyToClipboardWithOptions copies text to the clipboard and returns the selected backend name.
func CopyToClipboardWithOptions(
	ctx context.Context,
	runner *security.Runner,
	text string,
	opts ClipboardOptions,
) (string, error) {
	if text == "" {
		return "", errors.New("cannot copy empty text to clipboard")
	}
	if runner == nil {
		runner = security.NewRunner(5 * time.Second)
	}
	lookPath := opts.LookPath
	if lookPath == nil {
		lookPath = exec.LookPath
	}
	getenv := opts.Getenv
	if getenv == nil {
		getenv = os.Getenv
	}

	type candidate struct {
		name string
		bin  string
		args []string
		cond bool
	}

	hasWayland := getenv("WAYLAND_DISPLAY") != ""
	hasX11 := getenv("DISPLAY") != ""

	candidates := []candidate{
		{name: "wl-copy", bin: "wl-copy", args: nil, cond: hasWayland},
		{name: "xclip", bin: "xclip", args: []string{"-selection", "clipboard"}, cond: hasX11},
		{name: "xsel", bin: "xsel", args: []string{"--clipboard", "--input"}, cond: hasX11},
		{name: "pbcopy", bin: "pbcopy", args: nil, cond: true},
		{name: "clip.exe", bin: "clip.exe", args: nil, cond: true},
		{name: "wl-copy", bin: "wl-copy", args: nil, cond: !hasWayland},
		{name: "xclip", bin: "xclip", args: []string{"-selection", "clipboard"}, cond: !hasX11},
		{name: "xsel", bin: "xsel", args: []string{"--clipboard", "--input"}, cond: !hasX11},
	}

	for _, c := range candidates {
		if !c.cond {
			continue
		}
		binPath, err := lookPath(c.bin)
		if err != nil {
			continue
		}
		var runErr error
		if opts.CommandRunner != nil {
			runErr = opts.CommandRunner(ctx, binPath, c.args, strings.NewReader(text))
		} else {
			_, runErr = runner.RunWithInput(
				ctx,
				security.RunOptions{
					Stdin:   strings.NewReader(text),
					Timeout: 5 * time.Second,
				},
				binPath,
				c.args...,
			)
		}
		if runErr == nil {
			return c.name, nil
		}
	}

	if opts.FallbackWriter != nil && opts.FallbackWriter != io.Discard {
		encoded := base64.StdEncoding.EncodeToString([]byte(text))
		if _, err := fmt.Fprintf(opts.FallbackWriter, "\x1b]52;c;%s\x07", encoded); err == nil {
			return "osc52", nil
		}
	}

	return "", errors.New("no supported clipboard utility (wl-copy, xclip, xsel, pbcopy, clip.exe) available")
}
