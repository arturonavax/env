#!/usr/bin/env bash
set -euo pipefail

if [ "${BASH_SOURCE[0]}" != "$0" ]; then
    echo "ERROR: No ejecutes este script con 'source' o '.'. Ejecútalo con: bash $0 o ./${0##*/}" >&2
    return 1 2>/dev/null || exit 1
fi

if [ "$EUID" -eq 0 ]; then
    echo "Ejecuta este script como usuario normal (solicitará sudo cuando sea necesario)." >&2
    exit 1
fi

echo "==> Solicitando credenciales sudo..."
sudo -v
while true; do sudo -n true; sleep 60; kill -0 "$$" || exit; done 2>/dev/null &
SUDO_PID=$!

cleanup() {
    [ -n "${SUDO_PID:-}" ] && kill "$SUDO_PID" 2>/dev/null || true
}
trap cleanup EXIT INT TERM

echo "==> 1. Protegiendo paquetes esenciales del stack base de Xfce..."
sudo apt-mark manual \
    xfce4-session xfwm4 xfce4-panel xfce4-terminal xfce4-settings \
    xfce4-power-manager xfce4-pulseaudio-plugin xfce4-notifyd \
    xfce4-appfinder libgarcon-gtk3-1-0 2>/dev/null || true

echo "==> 2. Deteniendo procesos y daemons conflictivos o residuales..."
killall -q picom light-locker xscreensaver xwallpaper ibus-daemon greenclip alttab skippy-xd 2>/dev/null || true
pkill -9 -x redshift 2>/dev/null || true
pkill -9 -f light-locker 2>/dev/null || true
pkill -f xfce-unlock-fixer 2>/dev/null || true
pkill -f alttab-daemon 2>/dev/null || true
pkill -f rofi-alt-tab-watcher 2>/dev/null || true

echo "==> 3. Purgando servicios systemd de usuario residuales..."
systemctl --user stop \
    picom.service skippy-xd.service xfdashboard.service \
    xfce-session-guard.service quota-primer.timer quota-primer.service \
    ayatana-indicator-application.service 2>/dev/null || true

systemctl --user disable \
    picom.service skippy-xd.service xfdashboard.service \
    xfce-session-guard.service quota-primer.timer quota-primer.service 2>/dev/null || true

rm -f "$HOME/.config/systemd/user/picom.service" \
      "$HOME/.config/systemd/user/skippy-xd.service" \
      "$HOME/.config/systemd/user/xfdashboard.service" \
      "$HOME/.config/systemd/user/xfce-session-guard.service" \
      "$HOME/.config/systemd/user/quota-primer.service" \
      "$HOME/.config/systemd/user/quota-primer.timer"
systemctl --user daemon-reload 2>/dev/null || true

echo "==> 4. Limpiando autostarts obsoletos y problemáticos..."
sudo rm -f /etc/xdg/autostart/light-locker.desktop 2>/dev/null || true

rm -f "$HOME/.config/autostart/light-locker.desktop" \
      "$HOME/.config/autostart/wallpaper.desktop" \
      "$HOME/.config/autostart/xwallpaper.desktop" \
      "$HOME/.config/autostart/nitrogen.desktop" \
      "$HOME/.config/autostart/xscreensaver.desktop" \
      "$HOME/.config/autostart/plank.desktop" \
      "$HOME/.config/autostart/xcape.desktop" \
      "$HOME/.config/autostart/greenclip.desktop" \
      "$HOME/.config/autostart/alttab.desktop" \
      "$HOME/.config/autostart/skippy-xd.desktop" \
      "$HOME/.config/autostart/xfdashboard.desktop" \
      "$HOME/.config/autostart/redshift.desktop" \
      "$HOME/.config/autostart/touchpad-setup.desktop" \
      "$HOME/.config/autostart/xfce-unlock-fixer.desktop"

echo "==> 5. Purgando paquetes obsoletos y conflictivos del sistema..."
sudo apt purge -y \
    light-locker xscreensaver xscreensaver-data xscreensaver-gl \
    xwallpaper xfdesktop4 plank xcape alttab skippy-xd \
    remmina remmina-plugin-rdp remmina-plugin-vnc remmina-plugin-secret remmina-common \
    2>/dev/null || true
sudo apt autoremove -y 2>/dev/null || true

echo "==> 6. Eliminando binarios manuales, scripts y carpetas huérfanas..."
sudo rm -f /usr/local/bin/greenclip /usr/local/bin/i3lock-color /usr/local/bin/betterlockscreen \
           /usr/local/bin/alttab /usr/local/bin/skippy-xd /usr/local/bin/rofi-window \
           /usr/local/bin/wallpaper.sh /usr/local/bin/wallpaper-picker \
           /usr/local/bin/xfce-unlock-fixer /usr/local/bin/toggle-redshift

rm -f "$HOME/.local/bin/greenclip" "$HOME/.local/bin/i3lock-color" "$HOME/.local/bin/betterlockscreen" \
      "$HOME/.local/bin/alttab" "$HOME/.local/bin/alttab-daemon.sh" "$HOME/.local/bin/skippy-xd" \
      "$HOME/.local/bin/skippy-xd.bin" "$HOME/.local/bin/rofi-window" "$HOME/.local/bin/rofi-alt-tab-watcher" \
      "$HOME/.local/src/rofi-alt-tab-watcher.c" "$HOME/.local/bin/toggle-layout.sh" \
      "$HOME/.local/bin/wallpaper.sh" "$HOME/.config/wallpaper" \
      "$HOME/.local/bin/xfce-unlock-fixer" "$HOME/.local/bin/toggle-redshift" \
      "$HOME/.local/bin/xfce-session-guard" \
      "$HOME/.local/share/applications/toggle-redshift.desktop"

rm -rf "$HOME/.config/nitrogen" "$HOME/.config/betterlockscreen" "$HOME/.config/plank" \
       "$HOME/.config/skippy-xd" "$HOME/.cache/greenclip.history" "$HOME/.config/redshift"
rm -f /tmp/redshift_* /tmp/xwallpaper* 2>/dev/null || true

echo "==> 7. Restableciendo configuraciones de hardware, energía y pantalla al estado base..."
# Limpiar configuraciones TLP y suspend hooks obsoletos
sudo rm -f /etc/tlp.d/99-performance.conf \
           /etc/systemd/system-sleep/99-performance-resume.sh \
           /etc/udev/rules.d/50-bluetooth-power.rules \
           /lib/systemd/system-sleep/99-quota-primer

# Desbloquear demonios de gestión de energía y temperatura
sudo systemctl unmask thermald 2>/dev/null || true
sudo systemctl unmask power-profiles-daemon 2>/dev/null || true
sudo udevadm control --reload-rules 2>/dev/null || true

# Desactivar auto-wake RTC 04:00 AM si existe
if [ -x "$HOME/.local/bin/quota-ctl" ]; then
    "$HOME/.local/bin/quota-ctl" auto-4am off 2>/dev/null || true
fi
sudo rtcwake -m disable 2>/dev/null || true

# Restablecer tablas de color de hardware (RandR Gamma) a neutral 6500K
if command -v redshift >/dev/null 2>&1; then
    redshift -m randr -x 2>/dev/null || true
elif command -v xrandr >/dev/null 2>&1 && [ -n "${DISPLAY:-}" ]; then
    for output in $(xrandr --query 2>/dev/null | grep " connected" | awk '{print $1}'); do
        xrandr --output "$output" --gamma 1.0:1.0:1.0 2>/dev/null || true
    done
fi

# Restablecer flags de energía en Xfce (prevenir bloqueos de pantalla corruptos en suspend/resume)
if command -v xfconf-query >/dev/null 2>&1; then
    xfconf-query -c xfce4-power-manager -p /xfce4-power-manager/lock-screen-suspend-hibernate -s false 2>/dev/null || true
fi

echo "==> Sanitización y restablecimiento a estado base completado exitosamente."
