#!/usr/bin/env bash
# ==============================================================================
# src/requirements/editor.sh
# Verifies prerequisites for Neovim (LazyVim) installation and plugins.
# Part of arturonavax/env development environment.
# ==============================================================================

while [[ ! -d ./.git/ && ! -d ./files/ && ! -d ./src/ ]]; do
	if [[ "$PWD" == "/" ]]; then
		echo "Repository root not found" >&2
		exit 1
	fi
	if ! cd ..; then
		echo "An error occurred while doing 'cd ..'" >&2
		exit 1
	fi
done

source ./src/lib/strict.sh
source ./src/lib/platform.sh
source ./src/lib/logging.sh

pkg_mgr="$(detect_pkg_mgr)"
if [[ "$pkg_mgr" == "unknown" ]]; then
	log_fatal "No supported package manager found (apt, dnf, pacman, brew, apk)."
fi

# Essential tooling checks
local_missing=()
for cmd in git make curl tar; do
	if ! command -v "$cmd" >/dev/null 2>&1; then
		local_missing+=("$cmd")
	fi
done

if ((${#local_missing[@]} > 0)); then
	log_fatal "Required commands missing for editor installation: ${local_missing[*]}"
fi

# Node.js check (Mason/LSPs recommend >= 18)
if command -v node >/dev/null 2>&1; then
	node_version_major="$(node --version | cut -d'.' -f1 | tr -d 'v')"
	if ((node_version_major < 18)); then
		log_warn "Node.js version is recommended to be v18.0.0 or higher (found: $(node --version))."
	fi
fi
