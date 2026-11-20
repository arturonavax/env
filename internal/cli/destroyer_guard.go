package cli

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/arturonavax/env/internal/domain"
)

// allowActiveTerminalEnv lets advanced users bypass the active terminal/multiplexer guard.
const allowActiveTerminalEnv = "ANX_ALLOW_ACTIVE_TERMINAL"

// detectActiveTerminalSession reports the managed terminal emulators and multiplexers the
// current process is running inside. Destroyer modes uninstall these, which would kill the
// session mid-purge.
func detectActiveTerminalSession(getenv func(string) string) []string {
	var found []string
	add := func(name string) {
		for _, f := range found {
			if f == name {
				return
			}
		}
		found = append(found, name)
	}
	tp := strings.ToLower(getenv("TERM_PROGRAM"))
	switch {
	case strings.Contains(tp, "ghostty") || getenv("GHOSTTY_RESOURCES_DIR") != "":
		add("Ghostty")
	}
	if strings.Contains(tp, "kitty") || getenv("KITTY_WINDOW_ID") != "" {
		add("Kitty")
	}
	if strings.Contains(tp, "wezterm") || getenv("WEZTERM_PANE") != "" {
		add("WezTerm")
	}
	if strings.Contains(tp, "alacritty") || getenv("ALACRITTY_SOCKET") != "" || getenv("ALACRITTY_LOG") != "" {
		add("Alacritty")
	}
	if strings.Contains(tp, "warp") || getenv("WARP_IS_LOCAL_SHELL_SESSION") != "" {
		add("Warp")
	}
	if getenv("HERDR_PANE_ID") != "" || getenv("HERDR_ENV") != "" {
		add("Herdr")
	}
	if getenv("TMUX") != "" {
		add("tmux")
	}
	if getenv("ZELLIJ") != "" {
		add("Zellij")
	}
	if getenv("STY") != "" {
		add("GNU screen")
	}
	return found
}

// guardDestroyerSession refuses to run a destroyer inside a managed terminal or multiplexer.
func guardDestroyerSession(dryRun bool) error {
	if dryRun || os.Getenv(allowActiveTerminalEnv) == "1" {
		return nil
	}
	found := detectActiveTerminalSession(os.Getenv)
	if len(found) == 0 {
		return nil
	}
	msg := fmt.Sprintf(
		"destroyer mode cannot run inside %s: it uninstalls the terminal/multiplexer you are using and would be cut off halfway, leaving a broken system.\n"+
			"Open a different terminal WITHOUT a multiplexer (e.g. macOS Terminal.app, or a plain TTY/SSH session) and run the command again.\n"+
			"(Advanced: set %s=1 to override at your own risk.)",
		strings.Join(found, " + "), allowActiveTerminalEnv)
	return domain.NewExitError(domain.ExitPreflightFailed, errors.New(msg))
}
