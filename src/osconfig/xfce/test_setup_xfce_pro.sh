#!/usr/bin/env bash
set -u

C_RESET="\033[0m"
C_BOLD="\033[1m"
C_GREEN="\033[32m"
C_RED="\033[31m"
C_YELLOW="\033[33m"
C_CYAN="\033[36m"

pass() { echo -e "  [${C_GREEN}OK${C_RESET}] $1"; }
fail() { echo -e "  [${C_RED}FAIL${C_RESET}] $1"; }
warn() { echo -e "  [${C_YELLOW}WARN${C_RESET}] $1"; }
info() { echo -e "  [${C_CYAN}INFO${C_RESET}] $1"; }
header() { echo -e "\n${C_BOLD}${C_CYAN}=== $1 ===${C_RESET}"; }

inspect_binary() {
    local bin_name="$1"
    local bin_path
    bin_path=$(command -v "$bin_name" 2>/dev/null || true)
    if [ -z "$bin_path" ]; then
        echo -e "${C_RED}No instalado / no está en \$PATH${C_RESET}"
        return 1
    fi
    local real_path
    real_path=$(readlink -f "$bin_path" 2>/dev/null || echo "$bin_path")
    echo -e "${C_CYAN}Ruta: $real_path${C_RESET}"
}

# ==============================================================================
# 1. COMPOSITOR (Picom vs xfwm4)
# ==============================================================================
header "1. Compositor y Renderizado"
echo -n "  • Binario picom: "
inspect_binary "picom"

XFWM_COMPOSITING=$(xfconf-query -c xfwm4 -p /general/use_compositing 2>/dev/null || echo "not_found")
if [ "$XFWM_COMPOSITING" = "false" ]; then
    pass "Compositor interno de xfwm4 desactivado"[cite: 11, 15, 17]
else
    fail "Compositor de xfwm4 aún activo ($XFWM_COMPOSITING)"[cite: 11, 15, 17]
fi

if pgrep -x "picom" >/dev/null; then
    PICOM_PID=$(pgrep -x "picom" | head -n1)
    pass "Picom en ejecución (PID: $PICOM_PID)"[cite: 11, 15, 17]
    if [ -f "$HOME/.config/picom/picom.conf" ]; then
        BACKEND=$(grep -E '^[[:space:]]*backend' "$HOME/.config/picom/picom.conf" | tr -d '"; ' | cut -d'=' -f2)
        pass "Configuración picom.conf detectada (backend: ${BACKEND:-glx})"[cite: 11, 15, 17]
    else
        fail "Archivo ~/.config/picom/picom.conf no existe"[cite: 11, 15, 17]
    fi
else
    fail "Picom NO está corriendo como proceso activo"[cite: 11, 15, 17]
fi

# ==============================================================================
# 2. SELECTOR DE VENTANAS (Alt+Tab Nativo y Rofi Búsqueda)
# ==============================================================================
header "2. Gestor de Ventanas y Conmutador (Alt+Tab & Rofi)"
echo -n "  • Binario rofi: "
inspect_binary "rofi"

XFWM_ALTTAB=$(xfconf-query -c xfce4-keyboard-shortcuts -p "/xfwm4/custom/<Alt>Tab" 2>/dev/null || echo "none")
if [ "$XFWM_ALTTAB" = "cycle_windows_key" ]; then
    pass "Atajo nativo xfwm4 <Alt>Tab configurado en: cycle_windows_key"[cite: 11, 15, 17]
else
    fail "Atajo nativo xfwm4 <Alt>Tab incorrecto (actual: '$XFWM_ALTTAB')"[cite: 11, 15, 17]
fi

XFWM_ALTSHIFTTAB=$(xfconf-query -c xfce4-keyboard-shortcuts -p "/xfwm4/custom/<Alt><Shift>Tab" 2>/dev/null || echo "none")
if [ "$XFWM_ALTSHIFTTAB" = "cycle_reverse_windows_key" ]; then
    pass "Atajo nativo xfwm4 <Alt><Shift>Tab configurado en: cycle_reverse_windows_key"[cite: 11, 15, 17]
else
    fail "Atajo nativo xfwm4 <Alt><Shift>Tab incorrecto (actual: '$XFWM_ALTSHIFTTAB')"[cite: 11, 15, 17]
fi

CMD_ALTTAB=$(xfconf-query -c xfce4-keyboard-shortcuts -p "/commands/custom/<Alt>Tab" 2>/dev/null || echo "none")
if [ "$CMD_ALTTAB" = "none" ] || [ -z "$CMD_ALTTAB" ]; then
    pass "Sin comandos custom interceptando <Alt>Tab (control delegado a xfwm4)"[cite: 11, 15, 17]
else
    fail "<Alt>Tab interceptado por comando personalizado: '$CMD_ALTTAB'"[cite: 11, 15, 17]
fi

CMD_ALTSLASH=$(xfconf-query -c xfce4-keyboard-shortcuts -p "/commands/custom/<Alt>slash" 2>/dev/null || echo "none")
if [[ "$CMD_ALTSLASH" == *"rofi"* && "$CMD_ALTSLASH" == *"window"* ]]; then
    pass "<Alt>slash asignado a modo búsqueda modal ($CMD_ALTSLASH)"[cite: 11, 15, 17]
else
    fail "<Alt>slash no apunta a 'rofi -show window' (actual: '$CMD_ALTSLASH')"[cite: 11, 15, 17]
fi

# ==============================================================================
# 3. VICINAE (Autostart, Proceso, Icono Oficial de Bandeja y Foco)
# ==============================================================================
header "3. Vicinae (Lanzador, Icono Oficial de Bandeja y Foco)"
echo -n "  • Binario vicinae: "
inspect_binary "vicinae"

if [ -f "$HOME/.config/autostart/vicinae.desktop" ]; then
    if grep -q "sleep" "$HOME/.config/autostart/vicinae.desktop" && grep -q "vicinae.*server" "$HOME/.config/autostart/vicinae.desktop"; then
        pass "Autostart presente con arranque diferido (sleep) para sincronización con el panel"
    else
        warn "Autostart no contiene sleep previo a 'vicinae server' (riesgo de degradación XEmbed al reiniciar)"
    fi
else
    fail "Archivo ~/.config/autostart/vicinae.desktop no existe"[cite: 11, 15, 17]
fi

# Verificar neutralización de daemons conflictivos de Ayatana
if systemctl --user is-active --quiet ayatana-indicator-application.service 2>/dev/null; then
    fail "ayatana-indicator-application.service activo (secuestrará el icono en el próximo reinicio)"[cite: 15, 17]
elif [ -f "$HOME/.config/autostart/ayatana-indicator-application.desktop" ]; then
    pass "ayatana-indicator-application neutralizado correctamente en systemd y autostart"
else
    warn "ayatana-indicator-application.desktop no encontrado en ~/.config/autostart"
fi

VICINAE_PROC_PIDS=$(pgrep -f "vicinae.*server|[v]icinae-server" 2>/dev/null | grep -v "^$$$" | tr '\n' ' ' || true)
[ -z "$VICINAE_PROC_PIDS" ] && VICINAE_PROC_PIDS=$(pgrep -x "vicinae" 2>/dev/null | grep -v "^$$$" | tr '\n' ' ' || true)

if [ -n "$VICINAE_PROC_PIDS" ]; then
    pass "Daemon Vicinae activo en memoria (PID: $VICINAE_PROC_PIDS)"[cite: 11, 15, 17]
elif command -v vicinae >/dev/null 2>&1 && vicinae ping >/dev/null 2>&1; then
    pass "Daemon Vicinae respondiendo activamente a peticiones IPC"[cite: 11, 15, 17]
else
    fail "Vicinae no se encuentra en ejecución"[cite: 11, 15, 17]
fi

SYNTHETIC_EXISTS=$(grep -rn "vicinae-grad" /usr/share/icons /usr/share/pixmaps "$HOME/.local/share/icons" 2>/dev/null | head -n1 || true)
if [ -n "$SYNTHETIC_EXISTS" ]; then
    fail "Icono sintético anterior aún presente en el sistema"[cite: 15, 17]
else
    pass "Sin rastros del icono sintético anterior"[cite: 15, 17]
fi

CHECK_ICON_PATHS=(
    "/usr/share/icons/Tela-circle-dark/22x22/panel/vicinae.png"
    "/usr/share/icons/Tela-circle-dark/scalable/panel/vicinae.svg"
    "/usr/share/icons/hicolor/22x22/status/vicinae.png"
    "/usr/share/icons/hicolor/scalable/status/vicinae.svg"
    "/usr/share/pixmaps/vicinae.svg"
    "/usr/share/pixmaps/vicinae.png"
)
FOUND_ICON=""
for ip in "${CHECK_ICON_PATHS[@]}"; do
    if [ -f "$ip" ]; then
        FOUND_ICON="$ip"
        break
    fi
done

if [ -n "$FOUND_ICON" ]; then
    pass "Icono oficial de Vicinae registrado para panel y bandeja ($FOUND_ICON)"[cite: 11, 15, 17]
else
    fail "Icono oficial de Vicinae no encontrado en las rutas de resolución del panel"[cite: 15, 17]
fi

FOCUS_STEAL=$(xfconf-query -c xfwm4 -p /general/prevent_focus_stealing 2>/dev/null || echo "true")
FOCUS_NEW=$(xfconf-query -c xfwm4 -p /general/focus_new 2>/dev/null || echo "false")
if [ "$FOCUS_STEAL" = "false" ] && [ "$FOCUS_NEW" = "true" ]; then
    pass "Reglas de foco en xfwm4 óptimas para captura inmediata de teclado"[cite: 11, 15, 17]
else
    fail "Reglas de foco restringen teclado (prevent_focus_stealing: $FOCUS_STEAL, focus_new: $FOCUS_NEW)"[cite: 11, 15, 17]
fi

VICINAE_KEY=$(xfconf-query -c xfce4-keyboard-shortcuts -p "/commands/custom/<Super>space" 2>/dev/null || echo "none")
if [[ "$VICINAE_KEY" == *"vicinae toggle"* ]]; then
    pass "Atajo <Super>space asignado a 'vicinae toggle'"[cite: 11, 15, 17]
else
    fail "Atajo <Super>space no asignado (actual: '$VICINAE_KEY')"[cite: 11, 15, 17]
fi

# ==============================================================================
# 4. AUDITORÍA DE PURGA (Software Antiguo)
# ==============================================================================
header "4. Auditoría de Eliminación de Componentes Antiguos"

OBSOLETE_PKGS=("xfdesktop4" "plank" "feh" "xcape" "alttab" "betterlockscreen" "skippy-xd")
RESIDUAL_PACKAGES=()
for pkg in "${OBSOLETE_PKGS[@]}"; do
    if dpkg -l "$pkg" 2>/dev/null | grep -q "^ii"; then
        RESIDUAL_PACKAGES+=("$pkg")
    fi
done

if [ ${#RESIDUAL_PACKAGES[@]} -eq 0 ]; then
    pass "Todos los paquetes obsoletos purgados de APT"[cite: 11, 15, 17]
else
    fail "Paquetes obsoletos aún presentes: ${RESIDUAL_PACKAGES[*]}"[cite: 11, 15, 17]
fi

RESIDUAL_PATHS=(
    "/usr/local/bin/greenclip"
    "/usr/local/bin/i3lock-color"
    "/usr/local/bin/betterlockscreen"
    "/usr/local/bin/alttab"
    "/usr/local/bin/skippy-xd"
    "/usr/local/bin/rofi-window"
    "$HOME/.local/bin/greenclip"
    "$HOME/.local/bin/alttab"
    "$HOME/.local/bin/skippy-xd"
    "$HOME/.local/bin/rofi-window"
    "$HOME/.local/bin/rofi-alt-tab-watcher"
    "$HOME/.local/src/rofi-alt-tab-watcher.c"
    "$HOME/.config/betterlockscreen"
    "$HOME/.config/plank"
    "$HOME/.config/skippy-xd"
    "$HOME/.config/autostart/alttab.desktop"
    "$HOME/.config/autostart/plank.desktop"
    "$HOME/.config/autostart/skippy-xd.desktop"
    "$HOME/.local/bin/toggle-layout.sh"
)

FOUND_ORPHANS=()
for path in "${RESIDUAL_PATHS[@]}"; do
    [ -e "$path" ] && FOUND_ORPHANS+=("$path")
done

if [ ${#FOUND_ORPHANS[@]} -eq 0 ]; then
    pass "Sin binarios manuales ni scripts huérfanos"[cite: 11, 15, 17]
else
    fail "Rutas residuales detectadas:"[cite: 11, 15, 17]
    for item in "${FOUND_ORPHANS[@]}"; do
        echo "       - $item"
    done
fi

# ==============================================================================
# 5. DISTRIBUCIÓN DE TECLADO (XKB Nativo)
# ==============================================================================
header "5. Distribución de Teclado (XKB Nativo)"

XINPUTRC_CONTENT=$(cat "$HOME/.xinputrc" 2>/dev/null || echo "none")
if [[ "$XINPUTRC_CONTENT" == *"run_im none"* ]]; then
    pass "~/.xinputrc configurado en 'run_im none' (Input method framework desactivado)"[cite: 11, 15, 17]
else
    warn "~/.xinputrc no contiene 'run_im none' (actual: '$XINPUTRC_CONTENT')"[cite: 11, 15, 17]
fi

XKB_LAYOUTS=$(xfconf-query -c keyboard-layout -p /Default/XkbLayout 2>/dev/null || echo "none")
XKB_GRP=$(xfconf-query -c keyboard-layout -p /Default/XkbOptions/Group 2>/dev/null || echo "none")
if [[ "$XKB_LAYOUTS" == *"us"* ]] && [[ "$XKB_LAYOUTS" == *"es"* ]] && [ "$XKB_GRP" = "grp:alt_shift_toggle" ]; then
    pass "XKB configurado con layouts '$XKB_LAYOUTS' y alternancia 'Alt+Shift'"[cite: 11, 15, 17]
else
    fail "Configuración XKB no óptima (layouts: '$XKB_LAYOUTS', options: '$XKB_GRP')"[cite: 11, 15, 17]
fi

XKB_PLUGIN_FOUND=$(xfconf-query -c xfce4-panel -p /plugins -l 2>/dev/null | grep -E '^/plugins/plugin-[0-9]+$' | while read -r p; do
    [ "$(xfconf-query -c xfce4-panel -p "$p" 2>/dev/null || true)" = "xkb" ] && echo "$p" && break
done)

if [ -n "$XKB_PLUGIN_FOUND" ]; then
    pass "Plugin xkb presente y activo en el panel ($XKB_PLUGIN_FOUND)"[cite: 11, 15, 17]
else
    fail "Plugin xkb no encontrado en el panel de Xfce"[cite: 11, 15, 17]
fi

# ==============================================================================
# 6. CAPTURAS DE PANTALLA Y OCR
# ==============================================================================
header "6. Capturas de Pantalla y OCR"
echo -n "  • Binario flameshot: "
inspect_binary "flameshot"
echo -n "  • Binario tesseract: "
inspect_binary "tesseract"

CAP_SCRIPTS=("cap-area" "cap-repeat" "cap-window" "cap-fullscreen" "cap-ocr")
ALL_SCRIPTS_OK=true
for scr in "${CAP_SCRIPTS[@]}"; do
    if [ ! -x "$HOME/.local/bin/$scr" ]; then
        fail "Script ~/.local/bin/$scr no existe o no es ejecutable"[cite: 11, 15, 17]
        ALL_SCRIPTS_OK=false
    fi
done
if [ "$ALL_SCRIPTS_OK" = true ]; then
    pass "Scripts de captura y OCR presentes y ejecutables en ~/.local/bin"[cite: 11, 15, 17]
fi

echo ""
