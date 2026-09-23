#!/usr/bin/env bash
# ==============================================================================
# rcmd backend: GNOME Shell (Wayland / X11 via D-Bus)
# ==============================================================================
# Communicates with rcmd-shell extension (or run-or-raise fallback) to focus,
# cycle, or launch applications natively in Mutter / GNOME Shell.
#
# Supports:
# 1. Configured mode (cmd, pattern, match_mode).
# 2. Dynamic mode (unconfigured letter shortcut: focuses open app starting with that letter).
# ==============================================================================

rcmd_backend_gnome() {
    local key="${1,,}"
    local cmd="$2"
    local pattern="${3:-$cmd}"
    local match_mode="${4:-class}"

    # --------------------------------------------------------------------------
    # 1. Primary: rcmd-shell Extension (dev.arturonavax.rcmd)
    # --------------------------------------------------------------------------
    if command -v busctl >/dev/null 2>&1; then
        # Try well-known bus name
        if busctl --user call dev.arturonavax.rcmd \
            /dev/arturonavax/rcmd \
            dev.arturonavax.rcmd \
            Trigger ssss "$key" "$cmd" "$pattern" "$match_mode" >/dev/null 2>&1; then
            return 0
        fi

        # Also try on org.gnome.Shell connection
        if busctl --user call org.gnome.Shell \
            /dev/arturonavax/rcmd \
            dev.arturonavax.rcmd \
            Trigger ssss "$key" "$cmd" "$pattern" "$match_mode" >/dev/null 2>&1; then
            return 0
        fi
    elif command -v gdbus >/dev/null 2>&1; then
        if gdbus call --session \
            --dest dev.arturonavax.rcmd \
            --object-path /dev/arturonavax/rcmd \
            --method dev.arturonavax.rcmd.Trigger "$key" "$cmd" "$pattern" "$match_mode" >/dev/null 2>&1; then
            return 0
        fi

        if gdbus call --session \
            --dest org.gnome.Shell \
            --object-path /dev/arturonavax/rcmd \
            --method dev.arturonavax.rcmd.Trigger "$key" "$cmd" "$pattern" "$match_mode" >/dev/null 2>&1; then
            return 0
        fi
    fi

    # --------------------------------------------------------------------------
    # 2. Secondary Fallback: run-or-raise Extension (org.gnome.Shell.Extensions.RunOrRaise)
    # --------------------------------------------------------------------------
    local ror_line=""
    if [ -n "$pattern" ] || [ -n "$cmd" ]; then
        if [ "$match_mode" = "title" ]; then
            ror_line=",$cmd,,$pattern"
        else
            ror_line=",$cmd,$pattern,"
        fi

        if command -v busctl >/dev/null 2>&1; then
            if busctl --user call org.gnome.Shell \
                /org/gnome/Shell/Extensions/RunOrRaise \
                org.gnome.Shell.Extensions.RunOrRaise \
                Call s "$ror_line" >/dev/null 2>&1; then
                return 0
            fi

            # Legacy GNOME 3/40 run-or-raise path
            local target="${pattern:-$cmd}"
            if busctl --user call org.gnome.Shell \
                /cz/edvard/run_or_raise \
                cz.edvard.run_or_raise \
                Trigger s "$target" >/dev/null 2>&1; then
                return 0
            fi
        elif command -v gdbus >/dev/null 2>&1; then
            if gdbus call --session \
                --dest org.gnome.Shell \
                --object-path /org/gnome/Shell/Extensions/RunOrRaise \
                --method org.gnome.Shell.Extensions.RunOrRaise.Call "$ror_line" >/dev/null 2>&1; then
                return 0
            fi
        fi
    fi

    # --------------------------------------------------------------------------
    # 3. Offline / Pre-Reload Fallback
    # --------------------------------------------------------------------------
    # If the GNOME extension is not active yet (e.g. before logging out and in),
    # launch the app if not running, or warn the user without spamming new windows.
    if [ -n "$cmd" ]; then
        local is_running=0
        local check_target="${pattern:-$cmd}"
        # Extract binary name from check_target
        local bin_target="${check_target%% *}"
        bin_target="${bin_target##*/}"

        if [ -n "$bin_target" ]; then
            if pgrep -f -i "$bin_target" >/dev/null 2>&1 || pgrep -i "$bin_target" >/dev/null 2>&1; then
                is_running=1
            fi
        fi

        if [ "$is_running" -eq 0 ]; then
            nohup bash -c "$cmd" >/dev/null 2>&1 &
        else
            if command -v notify-send >/dev/null 2>&1; then
                notify-send "rcmd (GNOME)" "Extensión instalada. Cierra e inicia sesión una vez en GNOME para activar el enfoque nativo en Wayland." -u low 2>/dev/null || true
            fi
        fi
    fi
}
