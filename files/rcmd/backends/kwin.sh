#!/usr/bin/env bash
# ==============================================================================
# rcmd backend: KDE KWin (Wayland / X11)
# Uses kdotool or KWin D-Bus interface with title and class support.
# ==============================================================================

rcmd_backend_kwin() {
    local key="$1"
    local cmd="$2"
    local pattern="${3:-$cmd}"
    local match_mode="${4:-class}"

    if command -v kdotool >/dev/null 2>&1; then
        local target="${pattern:-$key}"
        local wins
        if [ "$match_mode" = "title" ]; then
            wins=$(kdotool search --name "$target" 2>/dev/null)
        else
            wins=$(kdotool search --class "$target" 2>/dev/null)
        fi
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
