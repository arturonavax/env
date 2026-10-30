#!/usr/bin/env bash
# ==============================================================================
# rcmd backend: X11 (Universal EWMH: wmctrl + xdotool + xprop)
# Supports:
# - Configured shortcuts: launch if closed, focus/cycle if open, title/class mode
# - Unconfigured shortcuts: ONLY focus the last active open app starting with
#   that letter (lowercase), considering ONLY currently open apps.
# - Cleans desktop prefixes: gnome-, xfce4-, xfce-, kde- (e.g. gnome-calculator -> calculator)
# - Filters system panels, docks, desktop elements (e.g. xfce4-panel Clock)
# - MRU focus + sister windows raising when coming from another app
# - Optimized for near-instant execution (hash maps & pure bash builtins)
# ==============================================================================

find_dynamic_app_x11() {
    local key="${1,,}"

    # 1. Obtain stacking order (MRU: top to bottom)
    local stack=()
    mapfile -t stack < <(xprop -root _NET_CLIENT_LIST_STACKING 2>/dev/null | grep -o "0x[0-9a-fA-F]*" | tac)
    if [ ${#stack[@]} -eq 0 ]; then
        mapfile -t stack < <(xprop -root _NET_CLIENT_LIST 2>/dev/null | grep -o "0x[0-9a-fA-F]*" | tac)
    fi
    [ ${#stack[@]} -eq 0 ] && return 1

    # 2. Pre-parse open client windows into associative array (O(1) lookups)
    declare -A win_data
    local w d p c t norm_w
    while read -r w d p c _ t; do
        [ -z "$w" ] && continue
        printf -v norm_w "0x%08x" "$((w))" 2>/dev/null || norm_w="${w,,}"
        win_data["$norm_w"]="$d|$p|$c|$t"
    done < <(wmctrl -lxp 2>/dev/null)

    # 3. Search in stacking order (MRU)
    for wid_raw in "${stack[@]}"; do
        local wid_hex
        printf -v wid_hex "0x%08x" "$((wid_raw))" 2>/dev/null || continue
        local info="${win_data[$wid_hex]:-}"
        [ -z "$info" ] && continue

        local desk pid wmclass title
        IFS='|' read -r desk pid wmclass title <<< "$info"
        [ "$desk" = "-1" ] && continue

        # Skip system desktop elements, panels, docks, trays
        local wmclass_l="${wmclass,,}"
        if [[ "$wmclass_l" =~ (panel|desktop|dock|tray) ]]; then
            continue
        fi

        # Process name from /proc/$pid/comm (pure bash redirection, 0 forks)
        local comm=""
        if [ -n "$pid" ] && [ "$pid" != "0" ] && [ -r "/proc/$pid/comm" ]; then
            read -r comm < "/proc/$pid/comm" 2>/dev/null || comm=""
        fi

        # Convert to lowercase and clean runtime suffixes
        local comm_l="${comm,,}"
        comm_l="${comm_l%-bin}"
        comm_l="${comm_l%.real}"
        comm_l="${comm_l%-wrapped}"

        local inst="${wmclass%%.*}"
        local inst_l="${inst,,}"

        local cls="${wmclass##*.}"
        local cls_l="${cls,,}"

        # Clean desktop environment prefixes
        local comm_clean="${comm_l#gnome-}"
        comm_clean="${comm_clean#xfce4-}"
        comm_clean="${comm_clean#xfce-}"
        comm_clean="${comm_clean#kde-}"

        local inst_clean="${inst_l#gnome-}"
        inst_clean="${inst_clean#xfce4-}"
        inst_clean="${inst_clean#xfce-}"
        inst_clean="${inst_clean#kde-}"

        local cls_clean="${cls_l#gnome-}"
        cls_clean="${cls_clean#xfce4-}"
        cls_clean="${cls_clean#xfce-}"
        cls_clean="${cls_clean#kde-}"

        local candidates=("$inst_l" "$inst_clean" "$cls_l" "$cls_clean" "$comm_l" "$comm_clean")

        # Extract app name from title if available
        if [ -n "$title" ]; then
            local title_l="${title,,}"
            # If title has separator like ' — ' or ' - ', the app name is at the end (e.g. "... — Mozilla Firefox")
            if [[ "$title_l" =~ [[:space:]][—\-][[:space:]](.+)$ ]]; then
                local app_suffix="${BASH_REMATCH[1]}"
                candidates+=("$app_suffix")
                for w in $app_suffix; do
                    candidates+=("$w")
                done
            else
                # Standalone title (e.g. "Calculator", "Ghostty")
                local clean_title="${title_l#\(*\)[[:space:]]}"
                candidates+=("$clean_title")
                for w in $clean_title; do
                    candidates+=("$w")
                done
            fi
        fi

        # Check if any candidate starts with the requested key
        for cand in "${candidates[@]}"; do
            [ -z "$cand" ] && continue
            if [[ "$cand" == "$key"* ]]; then
                echo "$wid_hex|$wmclass"
                return 0
            fi
        done
    done
    return 1
}

rcmd_backend_x11() {
    local key="${1,,}"
    local cmd="$2"
    local pattern="$3"
    local match_mode="${4:-class}"

    # Defensive check: ensure required X11 tools exist
    if ! command -v wmctrl >/dev/null 2>&1 || ! command -v xdotool >/dev/null 2>&1; then
        echo "Error: rcmd requires 'wmctrl' and 'xdotool' on X11." >&2
        if [ -n "$cmd" ]; then
            nohup bash -c "$cmd" >/dev/null 2>&1 &
        fi
        exit 1
    fi

    # --------------------------------------------------------------------------
    # 1. Dynamic Mode (Unconfigured letter shortcut: only focus currently open apps)
    # --------------------------------------------------------------------------
    if [ -z "$pattern" ] && [ -z "$cmd" ]; then
        local target_info
        target_info=$(find_dynamic_app_x11 "$key" || true)

        # If no currently open app starts with this letter: DO NOTHING
        [ -z "$target_info" ] && exit 0

        local target_win app_wmclass
        IFS='|' read -r target_win app_wmclass <<< "$target_info"

        # Find all open windows belonging to this app
        local WINS=()
        mapfile -t WINS < <(wmctrl -lx | awk -v pat="$app_wmclass" 'tolower($3) == tolower(pat) {print tolower($1)}')
        [ ${#WINS[@]} -eq 0 ] && WINS=("$target_win")

        # Active window check
        local active_dec active_hex
        active_dec=$(xdotool getactivewindow 2>/dev/null || echo 0)
        printf -v active_hex "0x%08x" "$active_dec" 2>/dev/null || active_hex="0x00000000"

        local is_active_in_app=0
        local current_index=-1
        for i in "${!WINS[@]}"; do
            if [ "${WINS[$i]}" = "$active_hex" ]; then
                is_active_in_app=1
                current_index=$i
                break
            fi
        done

        if [ "$is_active_in_app" -eq 1 ]; then
            # Already active in this app: cycle to next window if multiple exist
            local next_index=$(( (current_index + 1) % ${#WINS[@]} ))
            target_win="${WINS[$next_index]}"
        else
            # Coming from another app: raise sister windows behind target
            for w in "${WINS[@]}"; do
                if [ "$w" != "$target_win" ]; then
                    xdotool windowraise "$((w))" 2>/dev/null
                fi
            done
        fi

        # Raise and activate target window
        xdotool windowraise "$((target_win))" 2>/dev/null
        xdotool windowactivate "$((target_win))" 2>/dev/null
        wmctrl -i -a "$target_win"
        exit 0
    fi

    # --------------------------------------------------------------------------
    # 2. Configured Mode (cmd or pattern specified)
    # --------------------------------------------------------------------------
    [ -z "$pattern" ] && pattern="$cmd"

    local WINS=()
    if [ "$match_mode" = "title" ]; then
        mapfile -t WINS < <(wmctrl -l | awk -v pat="$pattern" 'tolower($0) ~ tolower(pat) {print tolower($1)}')
    else
        mapfile -t WINS < <(wmctrl -lx | awk -v pat="$pattern" 'tolower($3) ~ tolower(pat) {print tolower($1)}')
    fi

    # Si no hay ventanas abiertas
    if [ ${#WINS[@]} -eq 0 ]; then
        if [ -n "$cmd" ]; then
            nohup bash -c "$cmd" >/dev/null 2>&1 &
        fi
        exit 0
    fi

    # Obtener el ID de la ventana activa en formato hex normalizado (0x00000000)
    local active_dec active_hex
    active_dec=$(xdotool getactivewindow 2>/dev/null || echo 0)
    printf -v active_hex "0x%08x" "$active_dec" 2>/dev/null || active_hex="0x00000000"

    local is_active_in_app=0
    local current_index=-1
    for i in "${!WINS[@]}"; do
        if [ "${WINS[$i]}" = "$active_hex" ]; then
            is_active_in_app=1
            current_index=$i
            break
        fi
    done

    # Determinar ventana objetivo (TARGET_WIN)
    local target_win=""
    if [ "$is_active_in_app" -eq 1 ]; then
        # Ya estás dentro de la app: ciclar a la siguiente ventana
        local next_index=$(( (current_index + 1) % ${#WINS[@]} ))
        target_win="${WINS[$next_index]}"
    else
        # Vienes de otra app: buscar en el stack X11 la ventana más reciente (MRU)
        local stack=()
        mapfile -t stack < <(xprop -root _NET_CLIENT_LIST_STACKING 2>/dev/null | grep -o '0x[0-9a-fA-F]*' | tac)
        if [ ${#stack[@]} -eq 0 ]; then
            mapfile -t stack < <(xprop -root _NET_CLIENT_LIST 2>/dev/null | grep -o '0x[0-9a-fA-F]*' | tac)
        fi
        for s_id in "${stack[@]}"; do
            local s_hex
            printf -v s_hex "0x%08x" "$((s_id))" 2>/dev/null || continue
            for w in "${WINS[@]}"; do
                if [ "$w" = "$s_hex" ]; then
                    target_win="$w"
                    break 2
                fi
            done
        done
        [ -z "$target_win" ] && target_win="${WINS[0]}"

        # Elevar las ventanas hermanas por detrás (estilo macOS)
        for w in "${WINS[@]}"; do
            if [ "$w" != "$target_win" ]; then
                xdotool windowraise "$((w))" 2>/dev/null
            fi
        done
    fi

    # Enfocar y elevar a la cima absoluta la ventana objetivo
    xdotool windowraise "$((target_win))" 2>/dev/null
    xdotool windowactivate "$((target_win))" 2>/dev/null
    wmctrl -i -a "$target_win"
}
