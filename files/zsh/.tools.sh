#!/bin/bash
# Basics
[[ ":$PATH:" != *":/usr/local/bin:"* ]] && export PATH="$PATH:/usr/local/bin"
[[ ":$PATH:" != *":$HOME/.local/bin:"* ]] && export PATH="$HOME/.local/bin:$PATH"

# go
[[ -d /usr/local/go/bin/ ]] && export PATH="/usr/local/go/bin:$HOME/go/bin:$PATH"

# rust
[[ -f ~/.cargo/env ]] && source "$HOME/.cargo/env"

# python - pyenv
if [[ -d ~/.pyenv/ ]]; then
	export PYENV_ROOT="$HOME/.pyenv"
	export PATH="$PYENV_ROOT/bin:$PATH"

	[[ "$(command -v pyenv)" != "" ]] && eval "$(pyenv init -)"
fi

# node - fnm
if [[ "$(uname -s)" == "Linux" ]]; then
	[[ -d ~/.local/share/fnm/ ]] && export PATH="$HOME/.local/share/fnm:$PATH"

elif [[ "$(uname -s)" == "Darwin" ]]; then
	[[ -d "$HOME/Library/Application Support/fnm" ]] && export PATH="$HOME/Library/Application Support/fnm:$PATH"
fi

[[ "$(command -v fnm)" != "" ]] && eval "$(fnm env)"

# node - pnpm
if [[ "$(uname -s)" == "Linux" ]]; then
	export PNPM_HOME="$HOME/.local/share/pnpm"

elif [[ "$(uname -s)" == "Darwin" ]]; then
	export PNPM_HOME="$HOME/Library/pnpm"
fi

[[ -d "$PNPM_HOME" ]] && export PATH="$PNPM_HOME:$PATH"

# deno
export DENO_INSTALL="$HOME/.deno"
[[ -d "$DENO_INSTALL/bin" ]] && export PATH="$DENO_INSTALL/bin:$PATH"
:
