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

# 2. Go (Golang) Runtime & Tools
_tools_path_prepend "/usr/local/go/bin"
_tools_path_prepend "/opt/homebrew/opt/go/bin"
_tools_path_prepend "$HOME/go/bin"
if ! command -v gvm >/dev/null 2>&1 && [[ -s "$HOME/.gvm/scripts/gvm" ]]; then
	# shellcheck source=/dev/null
	source "$HOME/.gvm/scripts/gvm"
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

# Export normalized PATH and clean in-memory helper footprint
export PATH
unset -f _tools_path_prepend _tools_path_append
:
