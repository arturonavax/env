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
# ==============================================================================

find_dynamic_app_x11() {
    local key="${1,,}"
    local stack
    mapfile -t stack < <(xprop -root _NET_CLIENT_LIST_STACKING 2>/dev/null | grep -o "0x[0-9a-fA-F]*" | tac)
    [ ${#stack[@]} -eq 0 ] && return 1

    local wmlist
    wmlist=$(wmctrl -lxp)
    [ -z "$wmlist" ] && return 1

    for wid_raw in "${stack[@]}"; do
        local wid_hex
        wid_hex=$(printf "0x%08x" "$((wid_raw))" 2>/dev/null)
        local line
        line=$(grep -i "^$wid_hex" <<< "$wmlist" | head -n1)
        [ -z "$line" ] && continue

        local desk pid wmclass title
        desk=$(awk '{print $2}' <<< "$line")
        [ "$desk" = "-1" ] && continue

        pid=$(awk '{print $3}' <<< "$line")
        wmclass=$(awk '{print $4}' <<< "$line")

        # Skip system desktop elements, panels, docks, trays
        local wmclass_l="${wmclass,,}"
        if [[ "$wmclass_l" =~ (panel|desktop|dock|tray) ]]; then
            continue
        fi

        # In wmctrl -lxp: 1=wid, 2=desk, 3=pid, 4=wmclass, 5=client_machine, 6+=title
        title=$(awk '{$1=$2=$3=$4=$5=""; print $0}' <<< "$line" | sed 's/^[[:space:]]*//')

        # Process name from /proc/$pid/comm
        local comm=""
        if [ -n "$pid" ] && [ "$pid" != "0" ]; then
            comm=$(cat "/proc/$pid/comm" 2>/dev/null || echo "")
        fi

        # Convert to lowercase and clean suffixes
        local comm_l="${comm,,}"
        comm_l="${comm_l%-bin}"
        comm_l="${comm_l%.real}"
        comm_l="${comm_l%-wrapped}"

        local inst="${wmclass%%.*}"
        local inst_l="${inst,,}"

        local cls="${wmclass##*.}"
        local cls_l="${cls,,}"

        # Clean desktop prefixes: gnome-, xfce4-, xfce-, kde-
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

        # Extract app name from title
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
                local clean_title
                clean_title=$(sed -E 's/^\([0-9]+\+?\)[[:space:]]*//' <<< "$title_l")
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
        active_hex=$(printf "0x%08x" "$active_dec")

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
        # Busca en el título (columna 4 en adelante de wmctrl -l, tolera badges como "(1)")
        mapfile -t WINS < <(wmctrl -l | awk -v pat="$pattern" 'tolower($0) ~ tolower(pat) {print tolower($1)}')
    else
        # Busca en WM_CLASS (columna 3 de wmctrl -lx)
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
    active_hex=$(printf "0x%08x" "$active_dec")

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
        local stack
        stack=$(xprop -root _NET_CLIENT_LIST_STACKING 2>/dev/null | grep -o '0x[0-9a-fA-F]*' | tac)
        for s_id in $stack; do
            local s_hex
            s_hex=$(printf "0x%08x" "$((s_id))" 2>/dev/null)
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
