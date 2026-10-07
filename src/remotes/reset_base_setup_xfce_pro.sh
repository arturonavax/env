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

echo "==> 1. Protegiendo binarios esenciales de la sesión Xfce..."
sudo apt-mark manual \
    xfce4-session xfwm4 xfce4-panel xfce4-terminal xfce4-settings \
    xfce4-power-manager xfce4-appfinder libgarcon-gtk3-1-0 2>/dev/null || true

echo "==> 2. Deteniendo procesos y daemons conflictivos..."
killall -q picom light-locker xfce4-screensaver xscreensaver xwallpaper ibus-daemon 2>/dev/null || true

echo "==> 3. Purgando servicios systemd de usuario residuales..."
systemctl --user stop picom.service skippy-xd.service xfdashboard.service 2>/dev/null || true
systemctl --user disable picom.service skippy-xd.service xfdashboard.service 2>/dev/null || true
rm -f "$HOME/.config/systemd/user/picom.service" \
      "$HOME/.config/systemd/user/skippy-xd.service" \
      "$HOME/.config/systemd/user/xfdashboard.service"
systemctl --user daemon-reload 2>/dev/null || true

echo "==> 4. Limpiando autostarts obsoletos..."
rm -f "$HOME/.config/autostart/wallpaper.desktop" \
      "$HOME/.config/autostart/xwallpaper.desktop" \
      "$HOME/.config/autostart/nitrogen.desktop" \
      "$HOME/.config/autostart/xfce4-screensaver.desktop" \
      "$HOME/.config/autostart/xscreensaver.desktop" \
      "$HOME/.config/autostart/plank.desktop" \
      "$HOME/.config/autostart/xcape.desktop" \
      "$HOME/.config/autostart/greenclip.desktop" \
      "$HOME/.config/autostart/alttab.desktop" \
      "$HOME/.config/autostart/skippy-xd.desktop" \
      "$HOME/.config/autostart/xfdashboard.desktop"

echo "==> 5. Purgando paquetes obsoletos..."
sudo apt purge -y \
    xfce4-screensaver xscreensaver xscreensaver-data xscreensaver-gl \
    xwallpaper xfdesktop4 plank xcape alttab skippy-xd \
    2>/dev/null || true

echo "==> 6. Eliminando scripts y configuraciones residuales..."
sudo rm -f /usr/local/bin/greenclip /usr/local/bin/i3lock-color /usr/local/bin/betterlockscreen \
           /usr/local/bin/alttab /usr/local/bin/skippy-xd /usr/local/bin/rofi-window \
           /usr/local/bin/wallpaper.sh /usr/local/bin/wallpaper-picker
rm -f "$HOME/.local/bin/greenclip" "$HOME/.local/bin/i3lock-color" "$HOME/.local/bin/betterlockscreen" \
      "$HOME/.local/bin/alttab" "$HOME/.local/bin/alttab-daemon.sh" "$HOME/.local/bin/skippy-xd" \
      "$HOME/.local/bin/skippy-xd.bin" "$HOME/.local/bin/rofi-window" "$HOME/.local/bin/rofi-alt-tab-watcher" \
      "$HOME/.local/src/rofi-alt-tab-watcher.c" "$HOME/.local/bin/toggle-layout.sh" \
      "$HOME/.local/bin/wallpaper.sh" "$HOME/.config/wallpaper"
rm -rf "$HOME/.config/nitrogen"

echo "==> 7. Restableciendo flags de energía..."
xfconf-query -c xfce4-power-manager -p /xfce4-power-manager/lock-screen-suspend-hibernate -s false 2>/dev/null || true

echo "==> Sanitización completada."
