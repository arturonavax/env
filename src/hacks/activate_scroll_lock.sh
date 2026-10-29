#!/usr/bin/env bash
# ==============================================================================
# src/hacks/activate_scroll_lock.sh
# Backward-compatible delegating shim for X11 Scroll Lock LED control.
# Resolves hacks/x11-scroll-lock-led/activate_scroll_lock.sh inside a repository
# checkout, or executes a standalone safe xset fallback when piped via curl.
#
# Run: curl -fsSL "https://env.arturonavax.dev/activate_scroll_lock.sh" | bash
# ==============================================================================
set -euo pipefail

if [[ -n "${BASH_SOURCE[0]:-}" && -f "${BASH_SOURCE[0]}" ]]; then
	SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
	CANONICAL_HACK="${SCRIPT_DIR}/../../hacks/x11-scroll-lock-led/activate_scroll_lock.sh"
	if [[ -f "${CANONICAL_HACK}" ]]; then
		exec bash "${CANONICAL_HACK}" "$@"
	fi
fi

toggle_scroll_lock_fallback() {
	if [[ "$(uname -s)" != "Linux" ]]; then
		printf '[activate_scroll_lock] Skipping: supported on Linux X11 only.\n' >&2
		exit 0
	fi
	if [[ -z "${DISPLAY:-}" ]]; then
		printf '[activate_scroll_lock] Skipping: X11 DISPLAY is not set.\n' >&2
		exit 0
	fi
	if ! hash xset 2>/dev/null; then
		printf '[activate_scroll_lock] Skipping: xset utility is not installed.\n' >&2
		exit 0
	fi

	local query_out current_state
	query_out="$(xset -q 2>/dev/null || true)"
	current_state="$(printf '%s\n' "${query_out}" | awk -F'Scroll Lock:' 'NF > 1 {print $2; exit}' | awk '{print $1}')"

	if [[ "${current_state}" == "off" || -z "${current_state}" ]]; then
		xset led named "Scroll Lock"
	else
		xset -led named "Scroll Lock"
	fi
}

toggle_scroll_lock_fallback "$@"
