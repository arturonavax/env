#!/usr/bin/env bash
# ==============================================================================
# src/remotes/_basics.sh
# Core directory structure and unified PATH initialization.
# Servable from https://env.arturonavax.dev/_basics.sh
# ==============================================================================

mkdir -p "$HOME/.local/bin"

if [[ ! -d /usr/local/bin ]] && command -v sudo >/dev/null 2>&1; then
	sudo mkdir -m 755 -p /usr/local/bin 2>/dev/null || true
fi

# Ensure standard binary directories are in PATH
for bin_dir in "$HOME/.local/bin" "$HOME/.cargo/bin" "$HOME/.local/share/pnpm" "$HOME/.local/share/fnm" "$HOME/go/bin" "/usr/local/bin" "/usr/local/go/bin"; do
	if [[ -d "$bin_dir" ]] && [[ ":$PATH:" != *":${bin_dir}:"* ]]; then
		export PATH="${bin_dir}:${PATH}"
	fi
done
