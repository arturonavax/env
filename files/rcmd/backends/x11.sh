#!/usr/bin/env bash
# ==============================================================================
# rcmd backend: X11 (Universal EWMH: wmctrl + xdotool + xprop)
# Supports:
# - Matching by WM_CLASS or by window TITLE (e.g. for WhatsApp, web apps)
# - Cycle through open windows if app is currently focused
# - MRU focus + sister windows raising when coming from another app
# - Launch application if closed (when configured)
# - Dynamic fallback: acts on open app starting with key if not configured
# ==============================================================================

find_dynamic_pattern_x11() {
    local key="${1,,}"
    local stack
    mapfile -t stack < <(xprop -root _NET_CLIENT_LIST_STACKING 2>/dev/null | grep -o "0x[0-9a-fA-F]*" | tac)
    
    local wmlist
    wmlist=$(wmctrl -lxp)
    
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
        title=$(cut -d" " -f5- <<< "$line")
        
        local inst="${wmclass%%.*}"
        local cls="${wmclass##*.}"
        local comm=""
        [ -n "$pid" ] && [ "$pid" != "0" ] && comm=$(cat "/proc/$pid/comm" 2>/dev/null || echo "")
        
        # Strip notification prefix like (1) or (99+) from title for matching
        local clean_title
        clean_title=$(sed -E 's/^\([0-9]+\+?\)[[:space:]]*//' <<< "$title")
        
        local inst_l="${inst,,}"
        local cls_l="${cls,,}"
        local comm_l="${comm,,}"
        local clean_title_l="${clean_title,,}"
        
        if [[ "$inst_l" == "$key"* ]]; then
            echo "${inst}:class"
            return 0
        elif [[ "$cls_l" == "$key"* ]]; then
            echo "${cls}:class"
            return 0
        elif [[ "$comm_l" == "$key"* ]]; then
            echo "${comm}:class"
            return 0
        elif [[ "$clean_title_l" == "$key"* ]]; then
            local first_word="${clean_title%% *}"
            echo "${first_word:-$clean_title}:title"
            return 0
        fi
    done
    return 1
}

rcmd_backend_x11() {
    local key="$1"
    local cmd="$2"
    local pattern="$3"
    local match_mode="${4:-class}"

    # Dynamic fallback if no pattern or command was provided
    if [ -z "$pattern" ] && [ -z "$cmd" ]; then
        local dyn_res
        dyn_res=$(find_dynamic_pattern_x11 "$key" || true)
        # If no open window matches the requested key, nothing to do
        [ -z "$dyn_res" ] && exit 0
        if [[ "$dyn_res" == *":title" ]]; then
            pattern="${dyn_res%:title}"
            match_mode="title"
        elif [[ "$dyn_res" == *":class" ]]; then
            pattern="${dyn_res%:class}"
            match_mode="class"
        else
            pattern="$dyn_res"
        fi
    fi

    [ -z "$pattern" ] && pattern="$cmd"

    # 1. Obtener ventanas según modo de coincidencia
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

    # 2. Obtener el ID de la ventana activa en formato hex normalizado (0x00000000)
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

    # 3. Determinar ventana objetivo (TARGET_WIN)
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

    # 4. Enfocar y elevar a la cima absoluta la ventana objetivo
    xdotool windowraise "$((target_win))" 2>/dev/null
    xdotool windowactivate "$((target_win))" 2>/dev/null
    wmctrl -i -a "$target_win"
}
