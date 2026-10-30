#!/usr/bin/env bash
# ==============================================================================
# src/remotes/usage_install.sh
# Validates installation arguments and provides usage documentation.
# Servable from https://env.arturonavax.dev/usage_install.sh
# ==============================================================================

if [[ -f ./src/lib/colors.sh ]]; then
	# shellcheck source=/dev/null
	source ./src/lib/colors.sh
elif [[ -f "$(dirname "${BASH_SOURCE[0]}")/_vars_colors.sh" ]]; then
	# shellcheck source=/dev/null
	source "$(dirname "${BASH_SOURCE[0]}")/_vars_colors.sh"
fi

editor="nvim (LazyVim)"

function usage() {
	cat <<EOF
[Installer]: List of supported installation targets:
  wizard / -w       - Interactive step-by-step installation wizard (Recommended).
  requirements / r  - Install development tools, languages (Go, Node, Rust, Python) and headers.
  fonts / f         - Install patched mono Nerd Fonts.
  terminal / t      - Install Ghostty, Herdr multiplexer, lean Zsh, Starship, and modern CLI tools.
  editor / e        - Install $editor, Mason language servers, and Treesitter parsers.
  ai                - Install AI CLI assistants (agy), RTK token killer, MCP servers, and harnesses.
  osconfig / o      - Configure operating system desktop shortcuts, window switcher (rcmd), and tweaks.
  all / a           - Install and integrate all of the above subsystems.
  check / --check   - Run non-destructive platform diagnostics and hardware profile.
  help / -h         - Show this help message.
EOF
}

# Check arguments
if [[ "$#" -eq 0 ]]; then
	usage
	exit 1
fi

for arg in "$@"; do
	case "$arg" in
	wizard | -w | --wizard) ;;
	requirements | r) ;;
	fonts | f) ;;
	terminal | t) ;;
	editor | e) ;;
	ai) ;;
	osconfig | o) ;;
	all | a) ;;
	check | --check) ;;
	h | help | -h | --help)
		usage
		exit 0
		;;
	*)
		echo "[Installer Error]: Unrecognized argument: '$arg'" >&2
		usage
		exit 1
		;;
	esac
done
