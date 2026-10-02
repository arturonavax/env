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
# 2. SELECTOR DE VENTANAS (Skippy-XD y Atajos)
# ==============================================================================
header "2. Gestor Alt-Tab (Skippy-XD)"
echo -n "  • Fuente de skippy-xd: "
inspect_binary "skippy-xd"

if pgrep -f "skippy-xd.*daemon" >/dev/null; then
    pass "Daemon de Skippy-XD activo en segundo plano[cite: 1]"
else
    fail "Daemon de Skippy-XD NO se está ejecutando (revisa ~/.config/autostart/skippy-xd.desktop)[cite: 1]"
fi

# Atajos nativos de xfwm4 desacoplados
XFWM_ALTTAB=$(xfconf-query -c xfce4-keyboard-shortcuts -p "/xfwm4/custom/<Alt>Tab" 2>/dev/null || echo "none")
if [ "$XFWM_ALTTAB" = "none" ] || [ -z "$XFWM_ALTTAB" ]; then
    pass "Atajo nativo xfwm4 <Alt>Tab anulado correctamente[cite: 1]"
else
    fail "Atajo nativo xfwm4 <Alt>Tab activo con acción: '$XFWM_ALTTAB'"
fi

# Atajos personalizados hacia Skippy-XD
CMD_ALTTAB=$(xfconf-query -c xfce4-keyboard-shortcuts -p "/commands/custom/<Alt>Tab" 2>/dev/null || echo "none")
if [[ "$CMD_ALTTAB" == *"skippy-xd"* ]]; then
    pass "<Alt>Tab asignado a: $CMD_ALTTAB[cite: 1]"
else
    fail "<Alt>Tab no apunta a skippy-xd (actual: '$CMD_ALTTAB')"
fi

CMD_SUPERTAB=$(xfconf-query -c xfce4-keyboard-shortcuts -p "/commands/custom/<Super>Tab" 2>/dev/null || echo "none")
if [[ "$CMD_SUPERTAB" == *"skippy-xd"* ]]; then
    pass "<Super>Tab asignado a: $CMD_SUPERTAB[cite: 1]"
else
    warn "<Super>Tab no apunta a skippy-xd (actual: '$CMD_SUPERTAB')"
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

OBSOLETE_PKGS=("xfdesktop4" "plank" "rofi" "feh" "imagemagick" "xcape" "alttab" "betterlockscreen")
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
    "$HOME/.local/bin/greenclip"
    "$HOME/.local/bin/alttab"
    "$HOME/.config/rofi"
    "$HOME/.config/betterlockscreen"
    "$HOME/.config/plank"
    "$HOME/.config/autostart/alttab.desktop"
    "$HOME/.config/autostart/plank.desktop"
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

echo ""

