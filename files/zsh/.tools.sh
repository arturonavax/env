#!/bin/bash
# ==============================================================================
# files/zsh/.tools.sh
# Development runtimes, toolchain paths, and package manager environments.
# Maintained by arturonavax/env. Loaded by both Zsh (~/.zshrc) and Bash.
# ==============================================================================

# Fast in-memory PATH helpers avoiding subshell forks and duplicate entries
_tools_path_prepend() {
	if [[ -d "$1" && ":$PATH:" != *":$1:"* ]]; then
		PATH="$1:$PATH"
	fi
}

_tools_path_append() {
	if [[ -d "$1" && ":$PATH:" != *":$1:"* ]]; then
		PATH="$PATH:$1"
	fi
}

# 1. Base User & System Binary Paths
_tools_path_append "/usr/local/bin"
_tools_path_prepend "$HOME/.local/bin"
_tools_path_prepend "$HOME/.fzf/bin"

# 2. Go (Golang) Runtime & Tools (GVM primary, standalone & system Go fallback)
export GVM_ROOT="${GVM_ROOT:-$HOME/.gvm}"
_tools_path_prepend "$HOME/go/bin"
_tools_path_prepend "/usr/local/go/bin"
_tools_path_prepend "/opt/homebrew/opt/go/bin"
if [[ -s "$GVM_ROOT/scripts/gvm" ]]; then
	if ! command -v gvm >/dev/null 2>&1; then
		# shellcheck source=/dev/null
		source "$GVM_ROOT/scripts/gvm"
	fi
fi

# 3. Rust & Cargo Toolchain
_tools_path_prepend "$HOME/.cargo/bin"
if [[ -f "$HOME/.cargo/env" ]]; then
	# shellcheck source=/dev/null
	source "$HOME/.cargo/env"
fi

# 4. Python & pyenv
export PYENV_ROOT="${PYENV_ROOT:-$HOME/.pyenv}"
if [[ -d "$PYENV_ROOT" ]]; then
	_tools_path_prepend "$PYENV_ROOT/bin"
	_tools_path_prepend "$PYENV_ROOT/shims"
	if [[ -z "${PYENV_SHELL:-}" ]] && command -v pyenv >/dev/null 2>&1; then
		eval "$(pyenv init - 2>/dev/null)"
	fi
	if command -v pyenv-virtualenv-init >/dev/null 2>&1; then
		eval "$(pyenv virtualenv-init - 2>/dev/null)"
	fi
fi

# 5. Fast Node Manager (fnm)
if [[ "${OSTYPE:-}" == darwin* ]]; then
	_tools_path_prepend "$HOME/Library/Application Support/fnm"
else
	_tools_path_prepend "$HOME/.local/share/fnm"
fi
if [[ -z "${FNM_MULTISHELL_PATH:-}" ]] && command -v fnm >/dev/null 2>&1; then
	_fnm_sh="bash"
	[[ -n "${ZSH_VERSION:-}" ]] && _fnm_sh="zsh"
	eval "$(fnm env --use-on-cd --shell "$_fnm_sh" 2>/dev/null || fnm env 2>/dev/null)"
	unset _fnm_sh
fi

# 6. pnpm (Fast disk-efficient package manager)
if [[ "${OSTYPE:-}" == darwin* ]]; then
	export PNPM_HOME="$HOME/Library/pnpm"
else
	export PNPM_HOME="$HOME/.local/share/pnpm"
fi
_tools_path_prepend "$PNPM_HOME"
_tools_path_prepend "$PNPM_HOME/bin"

# 7. atuin (SQLite-backed shell history)
_tools_path_prepend "$HOME/.atuin/bin"
if [[ -f "$HOME/.atuin/bin/env" ]]; then
	# shellcheck source=/dev/null
	source "$HOME/.atuin/bin/env"
fi

# 8. Bun JavaScript Runtime
if [[ -d "$HOME/.bun" ]]; then
	export BUN_INSTALL="$HOME/.bun"
	_tools_path_prepend "$BUN_INSTALL/bin"
fi

# 9. Web3 & Developer Utilities (Foundry, Console Ninja)
_tools_path_prepend "$HOME/.foundry/bin"
_tools_path_prepend "$HOME/.console-ninja/.bin"

# 10. Local Private Environment Secrets (API keys, tokens — never committed to git)
if [[ -f "$HOME/.env.secret" ]]; then
	# shellcheck source=/dev/null
	source "$HOME/.env.secret"
fi
if [[ -f "$HOME/.env.local" ]]; then
	# shellcheck source=/dev/null
	source "$HOME/.env.local"
fi

# 11. Multi-Platform Clipboard Manager (Wayland, X11, macOS, WSL, Termux)
_detect_clipboard_backend() {
	if [[ -n "${WAYLAND_DISPLAY:-}" ]] && command -v wl-copy >/dev/null 2>&1; then
		echo "wayland"
		return 0
	fi
	if [[ "${OSTYPE:-}" == darwin* ]] && command -v pbcopy >/dev/null 2>&1; then
		echo "macos"
		return 0
	fi
	if command -v clip.exe >/dev/null 2>&1 && { grep -qi "microsoft" /proc/version 2>/dev/null || [[ -n "${WSL_DISTRO_NAME:-}" ]]; }; then
		echo "wsl"
		return 0
	fi
	if command -v termux-clipboard-set >/dev/null 2>&1; then
		echo "termux"
		return 0
	fi
	if [[ -n "${DISPLAY:-}" ]] && command -v xclip >/dev/null 2>&1; then
		echo "xclip"
		return 0
	fi
	if [[ -n "${DISPLAY:-}" ]] && command -v xsel >/dev/null 2>&1; then
		echo "xsel"
		return 0
	fi
	if command -v wl-copy >/dev/null 2>&1; then
		echo "wayland"
		return 0
	fi
	if command -v xclip >/dev/null 2>&1; then
		echo "xclip"
		return 0
	fi
	if command -v pbcopy >/dev/null 2>&1; then
		echo "macos"
		return 0
	fi
	if command -v clip.exe >/dev/null 2>&1; then
		echo "wsl"
		return 0
	fi
	echo "none"
	return 1
}

clip() {
	local backend
	backend="$(_detect_clipboard_backend 2>/dev/null || echo "none")"

	if [[ "$backend" == "none" || -z "$backend" ]]; then
		echo "clip: [ERROR] No supported clipboard tool found (requires wl-copy, xclip, xsel, pbcopy, clip.exe, or termux-clipboard-set)." >&2
		return 1
	fi

	local -a copy_cmd=()
	case "$backend" in
		wayland) copy_cmd=(wl-copy) ;;
		macos)   copy_cmd=(pbcopy) ;;
		wsl)     copy_cmd=(clip.exe) ;;
		termux)  copy_cmd=(termux-clipboard-set) ;;
		xclip)   copy_cmd=(xclip -selection clipboard) ;;
		xsel)    copy_cmd=(xsel --clipboard --input) ;;
	esac

	# Mode 1: Arguments provided
	if (( $# > 0 )); then
		if [[ "$1" == "-p" || "$1" == "--paste" ]]; then
			shift
			clip-paste "$@"
			return $?
		fi
		if [[ "$1" == "-h" || "$1" == "--help" ]]; then
			cat <<'EOF'
clip - Universal Multi-Platform Clipboard Manager

Usage:
  clip <file...>       Copy file content(s) to clipboard
  command | clip       Copy stdout from pipe/stream to clipboard
  clip "text..."       Copy argument string to clipboard
  clip -p, --paste     Paste clipboard content to stdout
  clip -h, --help      Show this help message

Supported backends:
  Wayland (wl-copy), X11 (xclip/xsel), macOS (pbcopy), WSL (clip.exe), Termux
EOF
			return 0
		fi

		# Check if any argument is an existing file
		local any_file=0
		local f
		for f in "$@"; do
			if [[ -f "$f" ]]; then
				any_file=1
				break
			fi
		done

		if (( any_file == 1 )); then
			for f in "$@"; do
				if ! [[ -r "$f" ]]; then
					echo "clip: [ERROR] File cannot be read: $f" >&2
					return 1
				fi
			done
			cat "$@" | "${copy_cmd[@]}"
			local ret=$?
			if (( ret == 0 )); then
				echo "✓ Copied file content ($# file(s)) to clipboard." >&2
			fi
			return $ret
		else
			printf '%s' "$*" | "${copy_cmd[@]}"
			return $?
		fi
	fi

	# Mode 2: Piped from stdin
	if ! [[ -t 0 ]]; then
		"${copy_cmd[@]}"
		return $?
	fi

	echo "clip: Nothing to copy. Pipe standard input (e.g. 'echo text | clip') or pass file path(s) ('clip file.txt')." >&2
	echo "      Run 'clip --help' for usage options." >&2
	return 1
}

clip-paste() {
	local backend
	backend="$(_detect_clipboard_backend 2>/dev/null || echo "none")"

	case "$backend" in
		wayland) wl-paste ;;
		macos)   pbpaste ;;
		wsl)     powershell.exe -NoProfile -Command "Get-Clipboard" 2>/dev/null | tr -d '\r' ;;
		termux)  termux-clipboard-get ;;
		xclip)   xclip -selection clipboard -o ;;
		xsel)    xsel --clipboard --output ;;
		*)
			echo "clip-paste: [ERROR] No supported clipboard tool found." >&2
			return 1
			;;
	esac
}

# Export normalized PATH and clean in-memory helper footprint
export PATH
unset -f _tools_path_prepend _tools_path_append
:
