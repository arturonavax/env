#!/usr/bin/env bash
# ==============================================================================
# rcmd backend: GNOME Shell (Wayland / D-Bus)
# Uses cz.edvard.run_or_raise GNOME extension D-Bus interface, or launches app.
# ==============================================================================

rcmd_backend_gnome() {
    local key="$1"
    local cmd="$2"
    local pattern="${3:-$cmd}"

    # Target pattern: configured pattern or key
    local target="${pattern:-$key}"

    # Try GNOME Shell run_or_raise extension via D-Bus
    if command -v busctl >/dev/null 2>&1; then
        if busctl --user call org.gnome.Shell \
            /cz/edvard/run_or_raise \
            cz.edvard.run_or_raise \
            Trigger s "$target" >/dev/null 2>&1; then
            return 0
        fi
    elif command -v gdbus >/dev/null 2>&1; then
        if gdbus call --session \
            --dest org.gnome.Shell \
            --object-path /cz/edvard/run_or_raise \
            --method cz.edvard.run_or_raise.Trigger "$target" >/dev/null 2>&1; then
            return 0
        fi
    fi

    # Fallback: if not running and cmd is defined, launch it
    if [ -n "$cmd" ]; then
        nohup bash -c "$cmd" >/dev/null 2>&1 &
    fi
}
