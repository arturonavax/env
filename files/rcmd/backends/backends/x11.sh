#!/usr/bin/env bash
# ==============================================================================
# rcmd backend: X11 (Universal EWMH)
# Aislamiento estricto entre ventanas de tipo CLASS y tipo TITLE
# ==============================================================================

# Normaliza cualquier representación de ID de ventana a 0x%08x
normalize_wid() {
    local raw="$1"
    [ -z "$raw" ] && { echo ""; return; }
    printf "0x%08x" "$((raw))" 2>/dev/null || echo "${raw,,}"
}

# Obtiene todos los patrones configurados con match_mode 'title'
get_configured_title_patterns() {
    local conf="${CONFIG_FILE:-}"
    if [ -z "$conf" ] && declare -f find_config >/dev/null 2>&1; then
        conf="$(find_config 2>/dev/null || echo "")"
    fi
    [ -z "$conf" ] || [ ! -f "$conf" ] && return 0

    while IFS='|' read -r k cmd pat mode rest || [ -n "$k" ]; do
        k="${k//$'\r'/}"
        cmd="${cmd//$'\r'/}"
        pat="${pat//$'\r'/}"
        mode="${mode//$'\r'/}"

        k="${k#"${k%%[![:space:]]*}"}"
        k="${k%"${k##*[![:space:]]}"}"
        case "$k" in '#'*|'') continue ;; esac

        mode="${mode#"${mode%%[![:space:]]*}"}"
        mode="${mode%"${mode##*[![:space:]]}"}"

        if [ "${mode,,}" = "title" ]; then
            pat="${pat#"${pat%%[![:space:]]*}"}"
            pat="${pat%"${pat##*[![:space:]]}"}"
            cmd="${cmd#"${cmd%%[![:space:]]*}"}"
            cmd="${cmd%"${cmd##*[![:space:]]}"}"
            [ -z "$pat" ] && pat="$cmd"
            [ -n "$pat" ] && echo "$pat"
        fi
    done < "$conf"
}

# Verifica si una cadena contiene alguno de los patrones excluidos
matches_any_title_pattern() {
    local str="${1,,}"
    shift
    local p
    for p in "$@"; do
        [ -z "$p" ] && continue
        local p_l="${p,,}"
        if [[ "$str" == *"$p_l"* ]]; then
            return 0
        fi
    done
    return 1
}

find_dynamic_app_x11() {
    local key="${1,,}"
    local title_excludes=()
    mapfile -t title_excludes < <(get_configured_title_patterns)

    # 1. Obtener orden de apilamiento MRU (cima a fondo)
    local stack=()
    mapfile -t stack < <(xprop -root _NET_CLIENT_LIST_STACKING 2>/dev/null | grep -o "0x[0-9a-fA-F]*" | tac)
    if [ ${#stack[@]} -eq 0 ]; then
        mapfile -t stack < <(xprop -root _NET_CLIENT_LIST 2>/dev/null | grep -o "0x[0-9a-fA-F]*" | tac)
    fi
    [ ${#stack[@]} -eq 0 ] && return 1

    # 2. Mapear ventanas abiertas en hash map O(1) con IDs normalizados
    declare -A win_data
    local w d p c _ t norm_w
    while read -r w d p c _ t || [ -n "$w" ]; do
        [ -z "$w" ] && continue
        norm_w="$(normalize_wid "$w")"
        win_data["$norm_w"]="$d|$p|$c|$t"
    done < <(wmctrl -lxp 2>/dev/null)

    # 3. Buscar la ventana candidata más reciente en el stack
    for wid_raw in "${stack[@]}"; do
        local wid_hex
        wid_hex="$(normalize_wid "$wid_raw")"
        [ -z "$wid_hex" ] && continue

        local info="${win_data[$wid_hex]:-}"
        [ -z "$info" ] && continue

        local desk pid wmclass title
        IFS='|' read -r desk pid wmclass title <<< "$info"
        [ "$desk" = "-1" ] && continue

        # Ignorar paneles, escritorios y docks
        local wmclass_l="${wmclass,,}"
        case "$wmclass_l" in
            *panel*|*desktop*|*dock*|*tray*) continue ;;
        esac

        # Ignorar ventanas que pertenezcan a atajos dedicados de título
        if [ ${#title_excludes[@]} -gt 0 ] && [ -n "$title" ]; then
            if matches_any_title_pattern "$title" "${title_excludes[@]}"; then
                continue
            fi
        fi

        local comm=""
        if [ -n "$pid" ] && [ "$pid" != "0" ] && [ -r "/proc/$pid/comm" ]; then
            read -r comm < "/proc/$pid/comm" 2>/dev/null || comm=""
        fi

        local comm_l="${comm,,}"
        comm_l="${comm_l%-bin}"
        comm_l="${comm_l%.real}"
        comm_l="${comm_l%-wrapped}"

        local inst="${wmclass%%.*}"
        local inst_l="${inst,,}"
        local cls="${wmclass##*.}"
        local cls_l="${cls,,}"

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

        # Extracción limpia de sufijos de títulos sin regex frágil
        if [ -n "$title" ]; then
            local title_l="${title,,}"
            if [[ "$title_l" == *" - "* ]]; then
                local app_suffix="${title_l##* - }"
                candidates+=("$app_suffix")
                for w_item in $app_suffix; do candidates+=("$w_item"); done
            elif [[ "$title_l" == *"   "* ]]; then
                local app_suffix="${title_l##*   }"
                candidates+=("$app_suffix")
                for w_item in $app_suffix; do candidates+=("$w_item"); done
            else
                local clean_title="${title_l#\(*\)[[:space:]]}"
                clean_title="${clean_title#\([0-9]*\)[[:space:]]}"
                candidates+=("$clean_title")
                for w_item in $clean_title; do candidates+=("$w_item"); done
            fi
        fi

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

    if ! command -v wmctrl >/dev/null 2>&1 || ! command -v xdotool >/dev/null 2>&1; then
        echo "Error: rcmd requiere 'wmctrl' y 'xdotool' en X11." >&2
        [ -n "$cmd" ] && rcmd_launch "$cmd"
        exit 1
    fi

    local title_excludes=()
    if [ "$match_mode" = "class" ]; then
        mapfile -t title_excludes < <(get_configured_title_patterns)
    fi

    # --------------------------------------------------------------------------
    # 1. Modo Dinámico (Sin configurar en rcmd.conf)
    # --------------------------------------------------------------------------
    if [ -z "$pattern" ] && [ -z "$cmd" ]; then
        local target_info
        target_info=$(find_dynamic_app_x11 "$key" 2>/dev/null || true)
        [ -z "$target_info" ] && exit 0

        local target_win app_wmclass
        IFS='|' read -r target_win app_wmclass <<< "$target_info"

        local WINS=()
        local app_wmclass_l="${app_wmclass,,}"
        while read -r w d c host title || [ -n "$w" ]; do
            [ -z "$w" ] && continue
            local c_l="${c,,}"
            if [[ "$c_l" == *"$app_wmclass_l"* ]]; then
                if [ ${#title_excludes[@]} -gt 0 ] && matches_any_title_pattern "$title" "${title_excludes[@]}"; then
                    continue
                fi
                WINS+=("$(normalize_wid "$w")")
            fi
        done < <(wmctrl -lx 2>/dev/null)

        [ ${#WINS[@]} -eq 0 ] && WINS=("$target_win")

        local active_dec active_hex
        active_dec=$(xdotool getactivewindow 2>/dev/null || echo 0)
        active_hex="$(normalize_wid "$active_dec")"

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
            local next_index=$(( (current_index + 1) % ${#WINS[@]} ))
            target_win="${WINS[$next_index]}"
        fi

        wmctrl -i -a "$target_win" 2>/dev/null || xdotool windowactivate "$((target_win))" 2>/dev/null
        exit 0
    fi

    # --------------------------------------------------------------------------
    # 2. Modo Configurado (cmd o pattern especificado)
    # --------------------------------------------------------------------------
    [ -z "$pattern" ] && pattern="$cmd"
    local pat_l="${pattern,,}"

    local WINS=()
    declare -A WINS_MAP

    while read -r w d c host title || [ -n "$w" ]; do
        [ -z "$w" ] && continue
        local c_l="${c,,}"
        local t_l="${title,,}"
        local nw
        nw="$(normalize_wid "$w")"

        if [ "$match_mode" = "title" ]; then
            if [[ "$t_l" == *"$pat_l"* ]]; then
                WINS+=("$nw")
                WINS_MAP["$nw"]=1
            fi
        else
            if [[ "$c_l" == *"$pat_l"* ]]; then
                if [ ${#title_excludes[@]} -gt 0 ] && matches_any_title_pattern "$title" "${title_excludes[@]}"; then
                    continue
                fi
                WINS+=("$nw")
                WINS_MAP["$nw"]=1
            fi
        fi
    done < <(wmctrl -lx 2>/dev/null)

    if [ ${#WINS[@]} -eq 0 ]; then
        if [ -n "$cmd" ]; then
            rcmd_launch "$cmd"
        fi
        exit 0
    fi

    local active_dec active_hex
    active_dec=$(xdotool getactivewindow 2>/dev/null || echo 0)
    active_hex="$(normalize_wid "$active_dec")"

    local is_active_in_app=0
    local current_index=-1
    for i in "${!WINS[@]}"; do
        if [ "${WINS[$i]}" = "$active_hex" ]; then
            is_active_in_app=1
            current_index=$i
            break
        fi
    done

    local target_win=""
    if [ "$is_active_in_app" -eq 1 ]; then
        local next_index=$(( (current_index + 1) % ${#WINS[@]} ))
        target_win="${WINS[$next_index]}"
    else
        local stack=()
        mapfile -t stack < <(xprop -root _NET_CLIENT_LIST_STACKING 2>/dev/null | grep -o '0x[0-9a-fA-F]*' | tac)
        if [ ${#stack[@]} -eq 0 ]; then
            mapfile -t stack < <(xprop -root _NET_CLIENT_LIST 2>/dev/null | grep -o '0x[0-9a-fA-F]*' | tac)
        fi

        for s_id in "${stack[@]}"; do
            local s_hex
            s_hex="$(normalize_wid "$s_id")"
            [ -z "$s_hex" ] && continue
            if [ "${WINS_MAP[$s_hex]:-0}" -eq 1 ]; then
                target_win="$s_hex"
                break
            fi
        done

        [ -z "$target_win" ] && target_win="${WINS[0]}"
    fi

    wmctrl -i -a "$target_win" 2>/dev/null || xdotool windowactivate "$((target_win))" 2>/dev/null
}
