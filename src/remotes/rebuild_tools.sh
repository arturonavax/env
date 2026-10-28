#!/usr/bin/env bash
# ==============================================================================
# src/remotes/rebuild_tools.sh
# Remote & local entry point for universal tool rebuilding & re-compilation.
# Run: curl -fsSL "https://env.arturonavax.dev/rebuild_tools.sh" | bash
# ==============================================================================
set -euo pipefail

_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
_LOCAL_SCRIPT="${_DIR}/../../utils/rebuild_tools.sh"

if [[ -f "$_LOCAL_SCRIPT" ]]; then
	exec bash "$_LOCAL_SCRIPT" "$@"
else
	# Standalone remote execution: fetch utils/rebuild_tools.sh directly
	_TMP="$(mktemp "${TMPDIR:-/tmp}/rebuild_tools.XXXXXX")"
	if curl -fsSL "https://raw.githubusercontent.com/arturonavax/env/main/utils/rebuild_tools.sh" -o "$_TMP"; then
		chmod +x "$_TMP"
		bash "$_TMP" "$@"
		rm -f "$_TMP"
	else
		echo "[Error]: Failed to download rebuild_tools.sh from repository." >&2
		rm -f "$_TMP"
		exit 1
	fi
fi
