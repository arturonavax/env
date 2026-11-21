#!/usr/bin/env bash
# ==============================================================================
# rcmd backend: XFCE (xfwm4 / EWMH via xdotool & wmctrl)
# Native window focus and cycling for XFCE4 desktops
# ==============================================================================

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Ensure x11 base backend is loaded for EWMH primitives
if ! declare -f rcmd_backend_x11 >/dev/null 2>&1; then
    if [ -f "$SCRIPT_DIR/x11.sh" ]; then
        # shellcheck source=/dev/null
        source "$SCRIPT_DIR/x11.sh"
    elif [ -f "$HOME/.local/lib/rcmd/backends/x11.sh" ]; then
        # shellcheck source=/dev/null
        source "$HOME/.local/lib/rcmd/backends/x11.sh"
    fi
fi

rcmd_backend_xfce() {
    local key="${1,,}"
    local cmd="$2"
    local pattern="${3:-$cmd}"
    local match_mode="${4:-class}"

    # Delegate to x11 backend with xfwm4-specific adjustments
    if declare -f rcmd_backend_x11 >/dev/null 2>&1; then
        rcmd_backend_x11 "$key" "$cmd" "$pattern" "$match_mode"
        return $?
    fi

    # Fallback to direct launch if x11 backend is unavailable
    if [ -n "$cmd" ]; then
        nohup bash -c "$cmd" >/dev/null 2>&1 &
    fi
}
