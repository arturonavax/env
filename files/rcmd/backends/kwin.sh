#!/usr/bin/env bash
# ==============================================================================
# rcmd backend: KDE KWin (Wayland / X11)
# Uses kdotool or KWin D-Bus interface.
# ==============================================================================

rcmd_backend_kwin() {
    local key="$1"
    local cmd="$2"
    local pattern="${3:-$cmd}"

    if command -v kdotool >/dev/null 2>&1; then
        local target="${pattern:-$key}"
        local wins
        wins=$(kdotool search --class "$target" 2>/dev/null)
        if [ -n "$wins" ]; then
            local first_win
            first_win=$(echo "$wins" | head -n1)
            kdotool windowactivate "$first_win" 2>/dev/null
            return 0
        fi
    fi

    # Fallback to launch
    if [ -n "$cmd" ]; then
        nohup bash -c "$cmd" >/dev/null 2>&1 &
    fi
}
