#!/usr/bin/env bash
# ==============================================================================
# Dell Precision M6800 (Quadro K3100M Kepler / Nouveau / Xfce 4.20) System Hacks
# ==============================================================================
# Resolves:
#   1. Ghostty + Herdr input lag and DRI3 buffer leaks on Nouveau Kepler.
#   2. Brave orange drop shadows, GPU process lockups, and 60 FPS video playback.
#   3. Xfce windows rendered behind the top panel upon resume / unlock.
#   4. Night Shift (Redshift) double-toggle requirement and display sleep drops.
#   5. Unwanted 04:00 AM RTC hardware wakeups from S3 suspend.
# ==============================================================================

set -o errexit
set -o pipefail
set -o nounset

exit-error-message() {
    echo -e "\n\033[0;31m[ERROR] Dell Precision M6800 hacks failed at line $1\033[0m" >&2
}
trap 'exit-error-message $LINENO' ERR SIGINT

# 0. Repository root anchoring loop
while [[ ! -d ./.git/ && ! -d ./files/ && ! -d ./src/ ]]; do
    if [[ "$PWD" == "/" ]]; then
        echo "Repository root not found"
        exit 1
    fi
    if ! cd ..; then
        echo "An error occurred while doing 'cd ..'"
        exit 1
    fi
done

echo "==> Applying hardware hacks for Dell Precision M6800 (Nouveau Kepler / Xfce)..."

# ------------------------------------------------------------------------------
# 1. Ghostty Terminal Latency, Memory & Opacity Optimization
# ------------------------------------------------------------------------------
echo "--> [1/6] Configuring Ghostty (opaque visual, zero-latency vsync & bounded scrollback)..."
for cfg in "$HOME/.config/ghostty/config" "./files/ghostty/config"; do
    if [ -f "$cfg" ]; then
        sed -i 's/^background-opacity = .*/background-opacity = 1.0/' "$cfg"
        sed -i 's/^window-vsync = .*/window-vsync = false/' "$cfg"
        grep -q "^scrollback-limit" "$cfg" && sed -i 's/^scrollback-limit = .*/scrollback-limit = 10000/' "$cfg" || echo "scrollback-limit = 10000" >> "$cfg"
        grep -q "^image-storage-limit" "$cfg" && sed -i 's/^image-storage-limit = .*/image-storage-limit = 50000000/' "$cfg" || echo "image-storage-limit = 50000000" >> "$cfg"
    fi
done

# Wrapper de alto rendimiento: neutraliza la fuga del 90% de CPU por AT-SPI / ATK-Bridge
cat <<'EOF' > "$HOME/.local/bin/ghostty"
#!/usr/bin/env bash
export GTK_A11Y=none
export NO_AT_BRIDGE=1
export GTK_MODULES=""
export GSK_RENDERER=gl
exec /usr/bin/ghostty "$@"
EOF
chmod +x "$HOME/.local/bin/ghostty"

if [ -f "/usr/share/applications/com.mitchellh.ghostty.desktop" ]; then
    mkdir -p "$HOME/.local/share/applications"
    cp -f "/usr/share/applications/com.mitchellh.ghostty.desktop" "$HOME/.local/share/applications/"
    sed -i 's|Exec=.*ghostty.*|Exec=/home/arthurnavah/.local/bin/ghostty|g' "$HOME/.local/share/applications/com.mitchellh.ghostty.desktop"
fi

# ------------------------------------------------------------------------------
# 2. Picom Compositor Leak & Shader Bypass
# ------------------------------------------------------------------------------
echo "--> [2/6] Configuring Picom for Mesa Nouveau DRI3 stability..."
mkdir -p "$HOME/.config/picom"
cat <<'EOF' > "$HOME/.config/picom/picom.conf"
backend = "glx";
vsync = true;
use-damage = true;
unredir-if-possible = false;

# En Nouveau / Mesa Kepler, la sincronización por valla evita la acumulación de retraso (lag)
xrender-sync-fence = true;
glx-no-stencil = true;
glx-no-rebind-pixmap = false;

shadow = false;
fading = false;

corner-radius = 10;
rounded-corners-exclude = [
    "window_type = 'dock'",
    "window_type = 'desktop'",
    "window_type = 'toolbar'",
    "window_type = 'menu'",
    "window_type = 'dropdown_menu'",
    "window_type = 'popup_menu'",
    "window_type = 'tooltip'",
    "window_type = 'utility'",
    "class_g = 'Xfce4-panel'",
    "class_g = 'Rofi'",
    "class_g = 'flameshot'",
    "class_g = 'Flameshot'",
    "class_g = 'xfce4-screensaver'",
    "class_g = 'Xfce4-screensaver'",
    "class_g = 'com.mitchellh.ghostty'",
    "class_g = 'ghostty'",
    "class_g = 'Brave-browser'",
    "class_g = 'brave-browser'",
    "fullscreen"
];

detect-rounded-corners = true;
detect-client-opacity = true;
detect-transient = true;
use-ewmh-active-win = true;

shadow-exclude = [
    "class_g = 'flameshot'",
    "class_g = 'Flameshot'",
    "class_g = 'vicinae'",
    "class_g = 'Xfce4-notifyd'",
    "class_g = 'xfce4-screensaver'",
    "class_g = 'Xfce4-screensaver'",
    "window_type = 'notification'",
    "_NET_WM_STATE *= '_NET_WM_STATE_HIDDEN'"
];

blur-background-exclude = [
    "class_g = 'flameshot'",
    "class_g = 'Flameshot'",
    "class_g = 'xfce4-screensaver'",
    "class_g = 'Xfce4-screensaver'",
    "class_g = 'com.mitchellh.ghostty'",
    "class_g = 'ghostty'",
    "class_g = 'Brave-browser'",
    "window_type = 'dock'",
    "window_type = 'desktop'"
];

focus-exclude = [
    "class_g = 'xfce4-screensaver'",
    "class_g = 'Xfce4-screensaver'"
];

wintypes:
{
  tooltip = { fade = false; shadow = false; opacity = 1.0; focus = true; full-shadow = false; };
  dock = { shadow = false; clip-shadow-above = true; };
  dnd = { shadow = false; };
  popup_menu = { opacity = 1.0; shadow = false; };
  dropdown_menu = { opacity = 1.0; shadow = false; };
  notification = { shadow = false; };
};
EOF

if pgrep -x picom >/dev/null 2>&1; then
    killall -q picom || true
    sleep 0.2
    (picom -b --config "$HOME/.config/picom/picom.conf" >/dev/null 2>&1 &) || true
fi

# ------------------------------------------------------------------------------
# 3. Brave Browser Flags (Fix Orange Shadows, 60 FPS Video & Canvas Stability)
# ------------------------------------------------------------------------------
echo "--> [3/6] Configuring Brave flags for CPU 2D canvas, Skia rasterization & GPU compositing..."
cat <<'EOF' > "$HOME/.config/brave-flags.conf"
# Optimizaciones de estabilidad y rendimiento para NVIDIA Quadro Kepler (GK104) con driver Nouveau en X11

# 1. Desactivar aceleración de Canvas 2D (CRÍTICO: previene cuelgues del kernel Nouveau con animaciones de fondo como Matrix)
--disable-accelerated-2d-canvas

# 2. Desactivar rasterización por GPU para Skia y Canvas (elimina sombras anaranjadas y cuelgues del proceso GPU)
# Al calcular sombras y glifos en CPU (Haswell i7-4910MQ de 8 hilos), los colores son 100% exactos y no saturan la VRAM
--disable-gpu-rasterization
--canvas-oop-rasterization=false

# 3. Composición de ventanas por GPU activa con memoria compartida estándar (MIT-SHM para 60 FPS sin saturar socket)
--enable-gpu-compositing

# 4. Plataforma nativa X11
--ozone-platform=x11

# 5. Desactivar decodificadores y APIs inestables en Nouveau Kepler
--disable-features=Vulkan,UseChromeOSDirectVideoDecoder,AcceleratedVideoDecodeLinux

# 6. Desplazamiento fluido
--enable-smooth-scrolling
EOF

# ------------------------------------------------------------------------------
# 4. Multi-Monitor Geometry Alignment (Fix Windows Behind Top Panel)
# ------------------------------------------------------------------------------
echo "--> [4/6] Aligning multi-monitor layout at Y=0 (DP-2 4K + eDP-1)..."
if command -v xfconf-query >/dev/null 2>&1; then
    xfconf-query -c displays -p /Default/DP-2/Position/Y -s 0 2>/dev/null || true
fi

if command -v xrandr >/dev/null 2>&1 && [ -n "${DISPLAY:-}" ]; then
    if xrandr --query | grep -q "^DP-2 connected" && xrandr --query | grep -q "^eDP-1 connected"; then
        xrandr --output DP-2 --pos 0x0 --output eDP-1 --pos 3840x0 2>/dev/null || true
    fi
fi

if command -v xfce4-panel >/dev/null 2>&1 && [ -n "${DISPLAY:-}" ]; then
    xfce4-panel -r 2>/dev/null || true
fi

if command -v xfwm4 >/dev/null 2>&1 && [ -n "${DISPLAY:-}" ]; then
    (xfwm4 --replace >/dev/null 2>&1 &) || true
fi

# ------------------------------------------------------------------------------
# 5. Night Shift / Redshift Robust Toggle & Session Guard (Zero Transitions)
# ------------------------------------------------------------------------------
echo "--> [5/6] Deploying instantaneous binary toggle-redshift and xfce-session-guard..."
mkdir -p "$HOME/.local/bin" "$HOME/.config/systemd/user" "$HOME/.local/share/applications" "$HOME/.local/state"

# Purgar cualquier demonio de transición automática en autostart
rm -f "$HOME/.config/autostart/redshift.desktop" "$HOME/.config/redshift/redshift.conf" 2>/dev/null || true
pkill -9 -x redshift 2>/dev/null || true

# Configuración declarativa de Luz Nocturna (editable por el usuario)
if [ ! -f "$HOME/.config/night-shift.conf" ]; then
cat <<'EOF' > "$HOME/.config/night-shift.conf"
# ==============================================================================
# Configuración de Luz Nocturna (Night Shift) - Dell Precision M6800 / Xfce
# ==============================================================================

# Horario automático (formato 24h HH:MM)
# Activo por defecto desde las 16:00 (4:00 PM) hasta las 06:30 (6:30 AM)
START_TIME="16:00"
END_TIME="06:30"

# Perfil de temperatura (Kelvin) y corrección Gamma
# 2400K: Protección visual máxima sin estados intermedios
TEMP_NIGHT=2400
GAMMA_NIGHT=0.85

# Programación automática activa (1 = habilitado, 0 = solo manual vía atajo)
AUTO_SCHEDULE=1
EOF
fi

cat <<'EOF' > "$HOME/.local/bin/toggle-redshift"
#!/usr/bin/env bash
# ==============================================================================
# toggle-redshift: Interruptor binario y programador automático para Luz Nocturna
# Optimizado para Dell Precision M6800 (Nouveau Kepler / Dual Display DP-2 + eDP-1)
# ==============================================================================
# Características:
#   1. Cero transiciones progresivas, cero curvas solares, cero estados intermedios.
#   2. Detección directa por hardware en RandR (LUT CRTC) para ambas pantallas.
#   3. Soporte de horario configurable con cruce de medianoche (ej. 16:00 - 06:30).
#   4. Idempotencia total: si ya está encendido, no re-aplica ni reinicia gamma.
#   5. Respeto absoluto a anulaciones manuales (toggle diurno/nocturno).
#   6. Autocorrección ante desincronización de pantallas tras DPMS o suspensión.
# ==============================================================================
set -uo pipefail

export DISPLAY="${DISPLAY:-:0}"
export XAUTHORITY="${XAUTHORITY:-$HOME/.Xauthority}"
if [ -z "${DBUS_SESSION_BUS_ADDRESS:-}" ] && [ -f "/run/user/$(id -u)/bus" ]; then
    export DBUS_SESSION_BUS_ADDRESS="unix:path=/run/user/$(id -u)/bus"
fi

CONFIG_FILE="$HOME/.config/night-shift.conf"
if [ -f "$CONFIG_FILE" ]; then
    # shellcheck disable=SC1090
    source "$CONFIG_FILE"
fi

# Parámetros con valores por defecto resilientes
START_TIME="${START_TIME:-16:00}"
END_TIME="${END_TIME:-06:30}"
TEMP_NIGHT="${TEMP_NIGHT:-2400}"
GAMMA_NIGHT="${GAMMA_NIGHT:-0.85}"
AUTO_SCHEDULE="${AUTO_SCHEDULE:-1}"

STATE_DIR="$HOME/.local/state"
mkdir -p "$STATE_DIR"
STATE_FILE="$STATE_DIR/night-shift-state"
OVERRIDE_FILE="$STATE_DIR/night-shift-override"

time_to_sec() {
    local t="${1:-00:00}"
    local h="${t%%:*}"
    local m="${t#*:}"
    h=$((10#$h))
    m=$((10#$m))
    echo $(( h * 3600 + m * 60 ))
}

is_in_schedule() {
    [ "${AUTO_SCHEDULE:-1}" = "1" ] || return 1
    local now_h now_m now_sec start_sec end_sec
    now_h=$(date +%H)
    now_m=$(date +%M)
    now_sec=$(( 10#$now_h * 3600 + 10#$now_m * 60 ))
    start_sec=$(time_to_sec "$START_TIME")
    end_sec=$(time_to_sec "$END_TIME")

    if [ "$start_sec" -le "$end_sec" ]; then
        if [ "$now_sec" -ge "$start_sec" ] && [ "$now_sec" -lt "$end_sec" ]; then
            return 0
        fi
    else
        if [ "$now_sec" -ge "$start_sec" ] || [ "$now_sec" -lt "$end_sec" ]; then
            return 0
        fi
    fi
    return 1
}

is_hardware_night_shift_active() {
    local gammas
    gammas=$(xrandr --verbose 2>/dev/null | grep -E "^\s*Gamma:" | awk '{print $2}' || true)
    if [ -n "$gammas" ] && echo "$gammas" | grep -qv "1.0:1.0:1.0"; then
        return 0
    fi
    return 1
}

are_all_outputs_night_shift() {
    local gammas
    gammas=$(xrandr --verbose 2>/dev/null | grep -E "^\s*Gamma:" | awk '{print $2}' || true)
    [ -n "$gammas" ] || return 1
    if echo "$gammas" | grep -q "1.0:1.0:1.0"; then
        return 1
    fi
    return 0
}

get_override_type() {
    [ -f "$OVERRIDE_FILE" ] || return 1
    local type ts
    read -r type ts < "$OVERRIDE_FILE" 2>/dev/null || return 1
    [[ "${ts:-}" =~ ^[0-9]+$ ]] || return 1

    local now_epoch
    now_epoch=$(date +%s)
    local age=$(( now_epoch - ts ))

    # La anulación manual vence pasadas 14 horas
    if [ "$age" -gt 50400 ] || [ "$age" -lt 0 ]; then
        rm -f "$OVERRIDE_FILE" 2>/dev/null || true
        return 1
    fi
    echo "$type"
    return 0
}

apply_night_shift() {
    pkill -x redshift 2>/dev/null || true
    redshift -m randr -P -O "$TEMP_NIGHT" -g "$GAMMA_NIGHT" 2>/dev/null || true
    echo "on" > "$STATE_FILE"
}

disable_night_shift() {
    pkill -x redshift 2>/dev/null || true
    redshift -m randr -x 2>/dev/null || true
    echo "off" > "$STATE_FILE"
}

MODE="${1:-toggle}"

case "$MODE" in
    on)
        apply_night_shift
        rm -f "$OVERRIDE_FILE" 2>/dev/null || true
        notify-send -t 1200 -i display "Luz Nocturna" "Protección Máxima activada (${TEMP_NIGHT}K - Gamma ${GAMMA_NIGHT})" 2>/dev/null || true
        ;;
    off)
        disable_night_shift
        if is_in_schedule; then
            echo "manual_off $(date +%s)" > "$OVERRIDE_FILE"
        else
            rm -f "$OVERRIDE_FILE" 2>/dev/null || true
        fi
        notify-send -t 1200 -i display "Luz Nocturna" "Filtro desactivado (Luz neutral 6500K)" 2>/dev/null || true
        ;;
    toggle)
        if is_hardware_night_shift_active; then
            disable_night_shift
            if is_in_schedule; then
                echo "manual_off $(date +%s)" > "$OVERRIDE_FILE"
            else
                rm -f "$OVERRIDE_FILE" 2>/dev/null || true
            fi
            notify-send -t 1200 -i display "Luz Nocturna" "Filtro desactivado (Luz neutral 6500K)" 2>/dev/null || true
        else
            apply_night_shift
            if ! is_in_schedule; then
                echo "manual_on $(date +%s)" > "$OVERRIDE_FILE"
            else
                rm -f "$OVERRIDE_FILE" 2>/dev/null || true
            fi
            notify-send -t 1200 -i display "Luz Nocturna" "Protección Máxima activada (${TEMP_NIGHT}K - Gamma ${GAMMA_NIGHT})" 2>/dev/null || true
        fi
        ;;
    auto)
        [ "${AUTO_SCHEDULE:-1}" = "1" ] || exit 0

        ov_type=$(get_override_type || true)

        if is_in_schedule; then
            [ "$ov_type" = "manual_on" ] && rm -f "$OVERRIDE_FILE" 2>/dev/null || true

            if [ "$ov_type" = "manual_off" ]; then
                exit 0
            fi

            # Si ya está activo en todas las pantallas, no hacer nada (idempotente)
            if are_all_outputs_night_shift; then
                exit 0
            fi

            apply_night_shift
        else
            [ "$ov_type" = "manual_off" ] && rm -f "$OVERRIDE_FILE" 2>/dev/null || true

            if [ "$ov_type" = "manual_on" ]; then
                if ! are_all_outputs_night_shift; then
                    apply_night_shift
                fi
                exit 0
            fi

            if is_hardware_night_shift_active; then
                disable_night_shift
            fi
        fi
        ;;
    restore)
        ov_type=$(get_override_type || true)

        if [ "${AUTO_SCHEDULE:-1}" = "1" ]; then
            if is_in_schedule; then
                if [ "$ov_type" = "manual_off" ]; then
                    disable_night_shift
                else
                    apply_night_shift
                fi
            else
                if [ "$ov_type" = "manual_on" ]; then
                    apply_night_shift
                else
                    disable_night_shift
                fi
            fi
        else
            target_state="off"
            [ -f "$STATE_FILE" ] && target_state=$(cat "$STATE_FILE" 2>/dev/null || echo "off")
            if [ "$target_state" = "on" ]; then
                apply_night_shift
            else
                disable_night_shift
            fi
        fi
        ;;
    status)
        if is_hardware_night_shift_active; then
            echo "on"
        else
            echo "off"
        fi
        ;;
    *)
        echo "Uso: $0 {on|off|toggle|auto|restore|status}" >&2
        exit 1
        ;;
esac
EOF
chmod +x "$HOME/.local/bin/toggle-redshift"

cat <<'EOF' > "$HOME/.local/bin/xfce-session-guard"
#!/usr/bin/env bash
# ==============================================================================
# xfce-session-guard: Centinela de sesión, geometría y luz nocturna para Xfce
# Optimizado para Dell Precision M6800 (Quadro K3100M / DP-2 4K + eDP-1)
# ==============================================================================
set -euo pipefail

export DISPLAY="${DISPLAY:-:0}"
export XAUTHORITY="${XAUTHORITY:-$HOME/.Xauthority}"
if [ -z "${DBUS_SESSION_BUS_ADDRESS:-}" ] && [ -f "/run/user/$(id -u)/bus" ]; then
    export DBUS_SESSION_BUS_ADDRESS="unix:path=/run/user/$(id -u)/bus"
fi

on_resume_or_unlock() {
    # 1. Espera inicial breve para el panel interno eDP-1
    sleep 0.6
    if [ -x "$HOME/.local/bin/toggle-redshift" ]; then
        "$HOME/.local/bin/toggle-redshift" restore >/dev/null 2>&1 || true
    fi

    # 2. Espera secundaria para permitir entrenamiento de enlace DisplayPort (DP-2 4K)
    sleep 1.2
    if [ -x "$HOME/.local/bin/toggle-redshift" ]; then
        "$HOME/.local/bin/toggle-redshift" restore >/dev/null 2>&1 || true
    fi

    # 3. Re-afirmar geometría de ventanas y bordes bajo el panel superior
    if pgrep -x xfwm4 >/dev/null 2>&1; then
        xfwm4 --replace >/dev/null 2>&1 &
    fi
}

# 1. Escuchar evento de desbloqueo de salvapantallas de Xfce
gdbus monitor --session --dest org.xfce.ScreenSaver 2>/dev/null | while read -r line; do
    if echo "$line" | grep -q "ActiveChanged: <false>"; then
        on_resume_or_unlock "screensaver-unlock"
    fi
done &
PID_SS=$!

# 2. Escuchar evento de reanudación del sistema desde suspensión (systemd-logind)
gdbus monitor --system --dest org.freedesktop.login1 2>/dev/null | while read -r line; do
    if echo "$line" | grep -q "PrepareForSleep: <false>"; then
        on_resume_or_unlock "system-resume"
    fi
done &
PID_LOGIN=$!

# 3. Bucle periódico watchdog (cada 30s) para programación automática e idempotencia
while true; do
    sleep 30
    if [ -x "$HOME/.local/bin/toggle-redshift" ]; then
        "$HOME/.local/bin/toggle-redshift" auto >/dev/null 2>&1 || true
    fi
done &
PID_WATCHDOG=$!

cleanup() {
    kill "$PID_SS" "$PID_LOGIN" "$PID_WATCHDOG" 2>/dev/null || true
}
trap cleanup EXIT INT TERM

wait
EOF
chmod +x "$HOME/.local/bin/xfce-session-guard"

cat <<'EOF' > "$HOME/.config/systemd/user/xfce-session-guard.service"
[Unit]
Description=Xfce Session Guard (Night Shift and Strut Keeper on Unlock & Resume)
After=graphical-session.target

[Service]
Type=simple
ExecStart=%h/.local/bin/xfce-session-guard
Restart=always
RestartSec=3
Environment=DISPLAY=:0
Environment=XAUTHORITY=%h/.Xauthority

[Install]
WantedBy=graphical-session.target default.target
EOF

systemctl --user daemon-reload 2>/dev/null || true
systemctl --user enable --now xfce-session-guard.service 2>/dev/null || true

if command -v xfconf-query >/dev/null 2>&1; then
    xfconf-query -c xfce4-keyboard-shortcuts -p "/commands/custom/<Primary><Alt>n" -s "$HOME/.local/bin/toggle-redshift" 2>/dev/null || true
fi

cat <<'EOF' > "$HOME/.local/share/applications/toggle-redshift.desktop"
[Desktop Entry]
Type=Application
Name=Luz Nocturna (Night Shift)
GenericName=Filtro de Luz Azul
Comment=Alternar filtro de luz azul y protección de pantalla
Exec=/home/arthurnavah/.local/bin/toggle-redshift
Icon=display
Terminal=false
Categories=Settings;Utility;
Keywords=night;shift;redshift;blue;light;filtro;luz;nocturna;
EOF

# ------------------------------------------------------------------------------
# 6. Disable Unwanted Suspend Wakeups
# ------------------------------------------------------------------------------
echo "--> [6/6] Disabling 04:00 AM RTC auto-wakeups and quota-primer timers..."
if [ -x "$HOME/.local/bin/quota-ctl" ]; then
    "$HOME/.local/bin/quota-ctl" auto-4am off 2>/dev/null || true
fi

systemctl --user stop quota-primer.timer quota-primer.service 2>/dev/null || true
systemctl --user disable quota-primer.timer quota-primer.service 2>/dev/null || true

echo "==> Dell Precision M6800 hacks successfully applied!"
