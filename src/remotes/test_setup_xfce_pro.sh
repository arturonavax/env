#!/usr/bin/env bash
set -u

# --- Formato y colores ---
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

# --- Detector de origen de binarios ---
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

    # 1. Comprobación Snap
    if echo "$real_path" | grep -q "^/snap/"; then
        echo -e "${C_YELLOW}Snap ($real_path) [Aislamiento/Sobrecarga de inicio]${C_RESET}"
        return 0
    fi

    # 2. Comprobación Flatpak
    if echo "$real_path" | grep -E -q "(flatpak|/var/lib/flatpak|~/.local/share/flatpak)"; then
        echo -e "${C_YELLOW}Flatpak ($real_path) [Sandboxed]${C_RESET}"
        return 0
    fi

    # 3. Comprobación AppImage
    if echo "$real_path" | grep -qi "\.appimage"; then
        echo -e "${C_YELLOW}AppImage ($real_path)${C_RESET}"
        return 0
    fi

    # 4. Comprobación DPKG / APT
    local dpkg_owner
    dpkg_owner=$(dpkg -S "$real_path" 2>/dev/null | cut -d':' -f1 || true)
    if [ -n "$dpkg_owner" ]; then
        echo -e "${C_GREEN}APT / Nativo DPKG ($dpkg_owner -> $real_path)${C_RESET}"
        return 0
    fi

    # 5. Binario local / compilado manualmente
    if [[ "$real_path" == /usr/local/* ]] || [[ "$real_path" == "$HOME/.local/"* ]]; then
        echo -e "${C_CYAN}Compilado manual / Binario directo ($real_path)${C_RESET}"
        return 0
    fi

    echo -e "${C_CYAN}Binario del sistema ($real_path)${C_RESET}"
}

# ==============================================================================
# 1. COMPOSITOR (Picom vs xfwm4)
# ==============================================================================
header "1. Compositor y Renderizado"
echo -n "  • Fuente de picom: "
inspect_binary "picom"

XFWM_COMPOSITING=$(xfconf-query -c xfwm4 -p /general/use_compositing 2>/dev/null || echo "not_found")
if [ "$XFWM_COMPOSITING" = "false" ]; then
    pass "Compositor interno de xfwm4 desactivado (/general/use_compositing = false)[cite: 1]"
else
    fail "Compositor de xfwm4 aún ACTIVO ($XFWM_COMPOSITING). Causa conflicto con Picom."
fi

if pgrep -x "picom" >/dev/null; then
    PICOM_PID=$(pgrep -x "picom" | head -n1)
    pass "Picom en ejecución (PID: $PICOM_PID)"
    
    # Comprobar backend activo en configuración
    if [ -f "$HOME/.config/picom/picom.conf" ]; then
        BACKEND=$(grep -E '^[[:space:]]*backend' "$HOME/.config/picom/picom.conf" | tr -d '"; ' | cut -d'=' -f2)
        VSYNC=$(grep -E '^[[:space:]]*vsync' "$HOME/.config/picom/picom.conf" | tr -d '; ' | cut -d'=' -f2)
        info "Configuración Picom: backend = ${BACKEND:-desconocido}, vsync = ${VSYNC:-desconocido}[cite: 1]"
    fi
else
    fail "Picom NO está corriendo como proceso activo"
fi

# ==============================================================================
# 2. SELECTOR DE VENTANAS (Rofi y Atajos)
# ==============================================================================
header "2. Gestor de Ventanas y Selector (Rofi)"
echo -n "  • Fuente de rofi: "
inspect_binary "rofi"

if command -v rofi >/dev/null 2>&1; then
    ROFI_VER=$(rofi -v 2>&1 | head -n1)
    pass "Binario de Rofi disponible y funcional ($ROFI_VER)"
else
    fail "Rofi NO está instalado o no se encuentra en \$PATH"
fi

if [ -x "$HOME/.local/bin/rofi-window" ]; then
    pass "Wrapper ~/.local/bin/rofi-window presente y ejecutable"
else
    fail "Wrapper ~/.local/bin/rofi-window no existe o no tiene permisos de ejecución"
fi

# Atajos nativos de xfwm4 desacoplados
XFWM_ALTTAB=$(xfconf-query -c xfce4-keyboard-shortcuts -p "/xfwm4/custom/<Alt>Tab" 2>/dev/null || echo "none")
if [ "$XFWM_ALTTAB" = "none" ] || [ -z "$XFWM_ALTTAB" ]; then
    pass "Atajo nativo xfwm4 <Alt>Tab anulado correctamente"
else
    fail "Atajo nativo xfwm4 <Alt>Tab activo con acción: '$XFWM_ALTTAB'"
fi

# Atajos personalizados hacia Rofi
CMD_ALTTAB=$(xfconf-query -c xfce4-keyboard-shortcuts -p "/commands/custom/<Alt>Tab" 2>/dev/null || echo "none")
if [[ "$CMD_ALTTAB" == *"rofi"* ]]; then
    pass "<Alt>Tab asignado a: $CMD_ALTTAB"
else
    fail "<Alt>Tab no apunta a rofi (actual: '$CMD_ALTTAB')"
fi

CMD_SUPERTAB=$(xfconf-query -c xfce4-keyboard-shortcuts -p "/commands/custom/<Super>Tab" 2>/dev/null || echo "none")
if [[ "$CMD_SUPERTAB" == *"rofi"* ]]; then
    pass "<Super>Tab asignado a: $CMD_SUPERTAB"
else
    warn "<Super>Tab no apunta a rofi (actual: '$CMD_SUPERTAB')"
fi

if [ -x "$HOME/.local/bin/rofi-alt-tab-watcher" ]; then
    pass "Watcher C X11 (~/.local/bin/rofi-alt-tab-watcher) compilado y ejecutable"
else
    fail "Watcher C X11 (~/.local/bin/rofi-alt-tab-watcher) no existe o no tiene permisos de ejecución"
fi

CMD_ALTSLASH=$(xfconf-query -c xfce4-keyboard-shortcuts -p "/commands/custom/<Alt>slash" 2>/dev/null || echo "none")
if [[ "$CMD_ALTSLASH" == *"rofi-window --search"* ]]; then
    pass "<Alt>slash asignado a modo búsqueda modal ($CMD_ALTSLASH)"
else
    warn "<Alt>slash no apunta a rofi-window --search (actual: '$CMD_ALTSLASH')"
fi

# ==============================================================================
# 3. DISPLAY MANAGER & PANTALLA DE BLOQUEO
# ==============================================================================
header "3. Display Manager y Lock Screen"
echo -n "  • Fuente de light-locker: "
inspect_binary "light-locker"

# Display Manager activo
DEFAULT_DM=$(cat /etc/X11/default-display-manager 2>/dev/null || echo "No definido")
info "Display Manager por defecto en /etc/X11: $DEFAULT_DM[cite: 1]"

ACTIVE_DM_UNIT=$(systemctl is-active display-manager.service 2>/dev/null || echo "inactivo")
if [ "$ACTIVE_DM_UNIT" = "active" ]; then
    DM_EXEC=$(systemctl show -p Id display-manager.service | cut -d'=' -f2)
    pass "display-manager.service activo vía systemd ($DM_EXEC)[cite: 1]"
else
    warn "display-manager.service no reporta estado 'active'"
fi

# Lock screen wrapper
if [ -x "$HOME/.local/bin/screenlock" ]; then
    pass "Script de bloqueo presente y ejecutable en ~/.local/bin/screenlock[cite: 1]"
else
    fail "Script ~/.local/bin/screenlock no existe o no tiene permisos de ejecución"
fi

if pgrep -x "light-locker" >/dev/null; then
    pass "Daemon light-locker activo en la sesión"
else
    info "Daemon light-locker no detectado en procesos (se invocará bajo demanda o en reposo)"
fi

# ==============================================================================
# 4. GESTIÓN DE ESCRITORIO Y FONDO
# ==============================================================================
header "4. Daemon de Escritorio y Fondo de Pantalla"
echo -n "  • Fuente de xwallpaper: "
inspect_binary "xwallpaper"

if pgrep -x "xfdesktop" >/dev/null; then
    fail "xfdesktop continúa EJECUTÁNDOSE en memoria consumiendo recursos."
else
    pass "xfdesktop no se está ejecutando (comportamiento óptimo sin iconos de escritorio)"
fi

FAILSAFE_CLIENT=$(xfconf-query -c xfce4-session -p /sessions/Failsafe/Client4_Command 2>/dev/null || echo "empty")
if [ "$FAILSAFE_CLIENT" = "empty" ] || [[ "$FAILSAFE_CLIENT" != *"xfdesktop"* ]]; then
    pass "xfdesktop desacoplado de la sesión Failsafe de Xfce[cite: 1]"
else
    warn "xfdesktop sigue referenciado en la sesión de arranque Failsafe"
fi

# ==============================================================================
# 5. LANZADOR, CLIPS Y AUDIO (Vicinae / PipeWire)
# ==============================================================================
header "5. Servicios Auxiliares (Vicinae & Audio)"
echo -n "  • Fuente de vicinae: "
inspect_binary "vicinae"

if pgrep -x "vicinae" >/dev/null || pgrep -x "vicinae-server" >/dev/null || pgrep -f "vicinae-server" >/dev/null; then
    pass "Servidor Vicinae en ejecución[cite: 1]"
else
    fail "Vicinae no está corriendo en segundo plano"
fi

if systemctl --user is-active --quiet pipewire wireplumber 2>/dev/null; then
    pass "PipeWire + WirePlumber activos bajo systemd --user[cite: 1]"
else
    warn "PipeWire o WirePlumber no se encuentran activos en la sesión de usuario"
fi

# ==============================================================================
# 6. AUDITORÍA DE PURGA (Software Antiguo / Conflictos)
# ==============================================================================
header "6. Auditoría de Eliminación de Componentes Antiguos"

OBSOLETE_PKGS=("xfdesktop4" "plank" "feh" "xcape" "alttab" "betterlockscreen" "skippy-xd")
RESIDUAL_PACKAGES=()

for pkg in "${OBSOLETE_PKGS[@]}"; do
    if dpkg -l "$pkg" 2>/dev/null | grep -q "^ii"; then
        RESIDUAL_PACKAGES+=("$pkg")
    fi
done

if [ ${#RESIDUAL_PACKAGES[@]} -eq 0 ]; then
    pass "Todos los paquetes conflictivos fueron purgados de APT[cite: 1]"
else
    fail "Paquetes todavía instalados en el sistema: ${RESIDUAL_PACKAGES[*]}"
fi

# Verificar si quedan binarios manuales o configs huérfanas
RESIDUAL_PATHS=(
    "/usr/local/bin/greenclip"
    "/usr/local/bin/i3lock-color"
    "/usr/local/bin/betterlockscreen"
    "/usr/local/bin/alttab"
    "/usr/local/bin/skippy-xd"
    "$HOME/.local/bin/greenclip"
    "$HOME/.local/bin/alttab"
    "$HOME/.local/bin/skippy-xd"
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
    if [ -e "$path" ]; then
        FOUND_ORPHANS+=("$path")
    fi
done

if [ ${#FOUND_ORPHANS[@]} -eq 0 ]; then
    pass "No existen binarios manuales ni directorios huérfanos de herramientas antiguas[cite: 1]"
else
    warn "Archivos/carpetas residuales encontrados:"
    for item in "${FOUND_ORPHANS[@]}"; do
        echo "       - $item"
    done
fi

# ==============================================================================
# 7. GESTOR DE ARCHIVOS PREDETERMINADO (Nemo)
# ==============================================================================
header "7. Gestor de Archivos Predeterminado (Nemo)"
echo -n "  • Fuente de nemo: "
inspect_binary "nemo"

MIME_DIR=$(xdg-mime query default inode/directory 2>/dev/null || echo "none")
if [ "$MIME_DIR" = "nemo.desktop" ]; then
    pass "Asociación MIME inode/directory configurada en: $MIME_DIR"
else
    fail "Asociación MIME inode/directory incorrecta (actual: '$MIME_DIR', esperado: 'nemo.desktop')"
fi

MIME_SEARCH=$(xdg-mime query default application/x-gnome-saved-search 2>/dev/null || echo "none")
if [ "$MIME_SEARCH" = "nemo.desktop" ]; then
    pass "Asociación MIME application/x-gnome-saved-search configurada en: $MIME_SEARCH"
else
    warn "Asociación MIME application/x-gnome-saved-search (actual: '$MIME_SEARCH')"
fi

XFCE_HELPER=$(grep -E '^FileManager=' "$HOME/.config/xfce4/helpers.rc" 2>/dev/null | cut -d'=' -f2 || echo "none")
if [ "$XFCE_HELPER" = "nemo" ]; then
    pass "Aplicación preferida en Xfce (helpers.rc) apunta a: $XFCE_HELPER"
else
    warn "helpers.rc no tiene FileManager=nemo (actual: '$XFCE_HELPER')"
fi

NEMO_DESKTOP_ICONS=$(gsettings get org.nemo.desktop show-desktop-icons 2>/dev/null || echo "unknown")
if [ "$NEMO_DESKTOP_ICONS" = "false" ]; then
    pass "Gestión de iconos de escritorio en Nemo desactivada (sin conflicto con xwallpaper)"
else
    warn "Gestión de iconos de escritorio en Nemo no está desactivada (actual: '$NEMO_DESKTOP_ICONS')"
fi

# ==============================================================================
# 8. SISTEMA DE NOTIFICACIONES (xfce4-notifyd y notification-plugin)
# ==============================================================================
header "8. Sistema de Notificaciones e Historial (Campanita)"
echo -n "  • Fuente de xfce4-notifyd: "
inspect_binary "xfce4-notifyd-config"
echo -n "  • Fuente de notify-send: "
inspect_binary "notify-send"

NOTIFY_THEME=$(xfconf-query -c xfce4-notifyd -p /theme 2>/dev/null || echo "none")
if [ "$NOTIFY_THEME" = "Orchis-Dark" ]; then
    pass "Tema de notificaciones configurado en: $NOTIFY_THEME"
else
    fail "Tema de notificaciones no es Orchis-Dark (actual: '$NOTIFY_THEME')"
fi

NOTIFY_LOG=$(xfconf-query -c xfce4-notifyd -p /notification-log 2>/dev/null || echo "false")
NOTIFY_LOG_LVL=$(xfconf-query -c xfce4-notifyd -p /log-level 2>/dev/null || echo "none")
NOTIFY_LOG_APPS=$(xfconf-query -c xfce4-notifyd -p /log-level-apps 2>/dev/null || echo "none")
if [ "$NOTIFY_LOG" = "true" ] && [ "$NOTIFY_LOG_LVL" = "always" ] && [ "$NOTIFY_LOG_APPS" = "all" ]; then
    pass "Historial y log de notificaciones activo y registrando todas las aplicaciones (always/all)"
else
    fail "Historial de notificaciones no configurado de forma óptima (log: $NOTIFY_LOG, nivel: $NOTIFY_LOG_LVL, apps: $NOTIFY_LOG_APPS)"
fi

if [ -f "$HOME/.themes/Orchis-Dark/xfce-notify-4.0/gtk.css" ]; then
    pass "Estilos Catppuccin Mocha para notificaciones presentes en ~/.themes"
else
    fail "Estilos de notificación en ~/.themes/Orchis-Dark/xfce-notify-4.0/gtk.css no encontrados"
fi

NOTIFY_PLUGIN_FOUND=$(xfconf-query -c xfce4-panel -p /plugins -l 2>/dev/null | grep -E '^/plugins/plugin-[0-9]+$' | while read -r p; do
    [ "$(xfconf-query -c xfce4-panel -p "$p" 2>/dev/null || true)" = "notification-plugin" ] && echo "$p" && break
done)

if [ -n "$NOTIFY_PLUGIN_FOUND" ]; then
    pass "Plugin de notificaciones (campanita) presente en el panel ($NOTIFY_PLUGIN_FOUND)"
    SHOW_MENU=$(xfconf-query -c xfce4-panel -p "$NOTIFY_PLUGIN_FOUND/show-in-menu" 2>/dev/null || echo "none")
    if [ "$SHOW_MENU" = "show-all" ]; then
        pass "Historial en la campanita configurado para mostrar todas las notificaciones (show-all)"
    else
        warn "show-in-menu en campanita no está en 'show-all' (actual: '$SHOW_MENU')"
    fi
else
    fail "Plugin de notificaciones (campanita) NO encontrado en el panel de Xfce"
fi

# ==============================================================================
# 9. DISTRIBUCIÓN DE TECLADO Y PANEL (XKB & Alt+Shift)
# ==============================================================================
header "9. Distribución de Teclado (XKB & Panel)"
if dpkg -l xfce4-xkb-plugin 2>/dev/null | grep -q "^ii"; then
    pass "Paquete xfce4-xkb-plugin instalado vía APT"
else
    fail "Paquete xfce4-xkb-plugin no está instalado"
fi

XKB_LAYOUTS=$(xfconf-query -c keyboard-layout -p /Default/XkbLayout 2>/dev/null || echo "none")
XKB_GRP=$(xfconf-query -c keyboard-layout -p /Default/XkbOptions/Group 2>/dev/null || echo "none")
if [[ "$XKB_LAYOUTS" == *"us"* ]] && [[ "$XKB_LAYOUTS" == *"es"* ]] && [ "$XKB_GRP" = "grp:alt_shift_toggle" ]; then
    pass "Distribución XKB configurada en: $XKB_LAYOUTS con atajo Alt+Shift ($XKB_GRP)"
else
    fail "Configuración XKB no óptima (layouts: '$XKB_LAYOUTS', options: '$XKB_GRP')"
fi

XKB_PLUGIN_FOUND=$(xfconf-query -c xfce4-panel -p /plugins -l 2>/dev/null | grep -E '^/plugins/plugin-[0-9]+$' | while read -r p; do
    [ "$(xfconf-query -c xfce4-panel -p "$p" 2>/dev/null || true)" = "xkb" ] && echo "$p" && break
done)

if [ -n "$XKB_PLUGIN_FOUND" ]; then
    pass "Item Keyboard Layout (xkb) presente en el panel de Xfce ($XKB_PLUGIN_FOUND)"
else
    fail "Plugin xkb no encontrado en xfce4-panel"
fi

SUPER_ALT_SPACE=$(xfconf-query -c xfce4-keyboard-shortcuts -p "/commands/custom/<Super><Alt>space" 2>/dev/null || echo "none")
if [ "$SUPER_ALT_SPACE" = "none" ] || [ -z "$SUPER_ALT_SPACE" ]; then
    pass "Atajo legacy <Super><Alt>space eliminado correctamente"
else
    fail "Atajo legacy <Super><Alt>space sigue activo: $SUPER_ALT_SPACE"
fi

# ==============================================================================
# 10. CAPTURAS DE PANTALLA Y OCR (Flameshot & Tesseract)
# ==============================================================================
header "10. Capturas de Pantalla y OCR (Flameshot & Tesseract)"
echo -n "  • Fuente de flameshot: "
inspect_binary "flameshot"
echo -n "  • Fuente de tesseract: "
inspect_binary "tesseract"
echo -n "  • Fuente de xclip: "
inspect_binary "xclip"
echo -n "  • Fuente de xwininfo: "
inspect_binary "xwininfo"

if command -v flameshot >/dev/null 2>&1; then
    FLAMESHOT_VER=$(flameshot --version 2>&1 | grep -i flameshot | head -n1)
    pass "Flameshot disponible y funcional ($FLAMESHOT_VER)"
else
    fail "Flameshot no está disponible en \$PATH"
fi

if command -v tesseract >/dev/null 2>&1; then
    TESS_LANGS=$(tesseract --list-langs 2>/dev/null | grep -E "(spa|eng)" | tr '\n' ' ' || echo "none")
    pass "Tesseract OCR disponible con lenguajes: $TESS_LANGS"
else
    fail "Tesseract OCR no está disponible"
fi

# Verificación de exclusión en Picom para evitar artefactos
if [ -f "$HOME/.config/picom/picom.conf" ]; then
    if grep -q "class_g = 'flameshot'" "$HOME/.config/picom/picom.conf" && \
       grep -q "blur-background-exclude" "$HOME/.config/picom/picom.conf"; then
        pass "Reglas de exclusión para Flameshot presentes en picom.conf (sin sombras ni desenfoque)"
    else
        fail "Faltan reglas de exclusión para Flameshot en picom.conf"
    fi
else
    fail "Archivo de configuración picom.conf no encontrado"
fi

# Verificación de configuración de Flameshot (desactivar indicador de tamaño y lupa)
if [ -f "$HOME/.config/flameshot/flameshot.ini" ]; then
    GEOM=$(grep -E '^showSelectionGeometry=' "$HOME/.config/flameshot/flameshot.ini" 2>/dev/null | cut -d'=' -f2 || echo "none")
    MAGN=$(grep -E '^showMagnifier=' "$HOME/.config/flameshot/flameshot.ini" 2>/dev/null | cut -d'=' -f2 || echo "none")
    if [ "$GEOM" = "0" ] && [ "$MAGN" = "false" ]; then
        pass "Configuración de Flameshot óptima (indicador de geometría desactivado y lupa desactivada)"
    else
        warn "Flameshot no tiene geometría desactivada o lupa desactivada (geom: $GEOM, lupa: $MAGN)"
    fi
else
    fail "Archivo ~/.config/flameshot/flameshot.ini no encontrado"
fi

# Verificación de scripts ejecutables en ~/.local/bin/
CAP_SCRIPTS=("cap-area" "cap-repeat" "cap-window" "cap-fullscreen" "cap-ocr")
ALL_SCRIPTS_OK=true
for scr in "${CAP_SCRIPTS[@]}"; do
    if [ ! -x "$HOME/.local/bin/$scr" ]; then
        fail "Script ~/.local/bin/$scr no existe o no es ejecutable"
        ALL_SCRIPTS_OK=false
    fi
done
if [ "$ALL_SCRIPTS_OK" = true ]; then
    pass "Todos los scripts cap-* presentes y ejecutables en ~/.local/bin"
fi

# Verificación de atajos de teclado en Xfce
SHORTCUTS=(
    "/commands/custom/<Primary><Alt><Super>1:cap-area"
    "/commands/custom/<Primary><Alt><Super>2:cap-repeat"
    "/commands/custom/<Primary><Alt><Super>3:cap-window"
    "/commands/custom/<Primary><Alt><Super>4:cap-fullscreen"
    "/commands/custom/<Primary><Alt><Super>0:cap-ocr"
)
ALL_SHORTCUTS_OK=true
for sc in "${SHORTCUTS[@]}"; do
    prop="${sc%%:*}"
    expected="${sc##*:}"
    val=$(xfconf-query -c xfce4-keyboard-shortcuts -p "$prop" 2>/dev/null || echo "none")
    if [[ "$val" == *"$expected"* ]]; then
        pass "Atajo $prop -> $val"
    else
        fail "Atajo $prop incorrecto o no configurado (actual: '$val', esperado: '$expected')"
        ALL_SHORTCUTS_OK=false
    fi
done

# Prueba de tubería Tesseract OCR
OCR_TEST_TXT=$(echo "ANTIGRAVITY_OCR_OK" | convert -background white -fill black -pointsize 20 label:@- png:- 2>/dev/null | tesseract stdin stdout -l eng --psm 6 2>/dev/null || echo "")
if [[ "$OCR_TEST_TXT" == *"ANTIGRAVITY_OCR_OK"* ]]; then
    pass "Prueba de tubería Tesseract OCR procesada exitosamente"
else
    # Fallback sin convert label:@-
    TMP_T_PNG=$(mktemp --suffix=.png)
    convert -background white -fill black -pointsize 20 label:"ANTIGRAVITY_OCR_OK" "$TMP_T_PNG" 2>/dev/null || true
    OCR_TEST_TXT=$(tesseract "$TMP_T_PNG" stdout -l eng --psm 6 2>/dev/null || echo "")
    rm -f "$TMP_T_PNG"
    if [[ "$OCR_TEST_TXT" == *"ANTIGRAVITY_OCR_OK"* ]]; then
        pass "Prueba de tubería Tesseract OCR procesada exitosamente"
    else
        warn "Prueba de tubería Tesseract OCR no pudo ser confirmada automáticamente (salida: '$OCR_TEST_TXT')"
    fi
fi

echo ""


