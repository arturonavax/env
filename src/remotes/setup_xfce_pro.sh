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

TMP_VIC=""
cleanup() {
    [ -n "${SUDO_PID:-}" ] && kill "$SUDO_PID" 2>/dev/null || true
    [ -n "${TMP_VIC:-}" ] && rm -rf "$TMP_VIC" 2>/dev/null || true
}
trap cleanup EXIT INT TERM

pgrep -x xfce4-panel >/dev/null || (xfce4-panel >/dev/null 2>&1 & sleep 1)

echo "==> 1. Asegurando repositorios (Universe) y protegiendo stack Xfce..."
sudo apt update
sudo apt install -y software-properties-common
sudo add-apt-repository -y universe
sudo apt update

sudo apt-mark manual \
    xfce4-session xfwm4 xfce4-panel xfce4-terminal xfce4-settings \
    xfce4-power-manager xfce4-pulseaudio-plugin xfce4-appfinder libgarcon-gtk3-1-0 2>/dev/null || true

# Purgar light-locker y servicios obsoletos que causan congelamientos de VT
sudo apt purge -y \
    light-locker xscreensaver xscreensaver-data xscreensaver-gl \
    xfdesktop4 plank xcape xwallpaper \
    remmina remmina-plugin-rdp remmina-plugin-vnc remmina-plugin-secret remmina-common \
    alttab skippy-xd \
    2>/dev/null || true
sudo apt autoremove -y 2>/dev/null || true

sudo rm -f /etc/xdg/autostart/light-locker.desktop 2>/dev/null || true
killall -9 light-locker 2>/dev/null || true
pkill -9 -f light-locker 2>/dev/null || true
sudo rm -f /usr/local/bin/greenclip /usr/local/bin/i3lock-color /usr/local/bin/betterlockscreen \
           /usr/local/bin/alttab /usr/local/bin/skippy-xd /usr/local/bin/rofi-window \
           /usr/local/bin/wallpaper.sh
rm -f "$HOME/.local/bin/greenclip" "$HOME/.local/bin/i3lock-color" "$HOME/.local/bin/betterlockscreen" \
      "$HOME/.local/bin/alttab" "$HOME/.local/bin/alttab-daemon.sh" "$HOME/.local/bin/skippy-xd" \
      "$HOME/.local/bin/skippy-xd.bin" "$HOME/.local/bin/rofi-window" "$HOME/.local/bin/rofi-alt-tab-watcher" \
      "$HOME/.local/src/rofi-alt-tab-watcher.c" "$HOME/.local/bin/toggle-layout.sh" \
      "$HOME/.local/bin/wallpaper.sh" "$HOME/.config/wallpaper"
rm -rf "$HOME/.config/betterlockscreen" "$HOME/.config/plank" "$HOME/.config/skippy-xd" "$HOME/.cache/greenclip.history"
rm -f "$HOME/.config/autostart/plank.desktop" "$HOME/.config/autostart/xcape.desktop" \
      "$HOME/.config/autostart/greenclip.desktop" "$HOME/.config/autostart/touchpad-setup.desktop" \
      "$HOME/.config/autostart/alttab.desktop" "$HOME/.config/autostart/skippy-xd.desktop" \
      "$HOME/.config/autostart/xfdashboard.desktop" "$HOME/.config/autostart/wallpaper.desktop" \
      "$HOME/.config/autostart/nitrogen.desktop" "$HOME/.config/autostart/light-locker.desktop" \
      "$HOME/.config/systemd/user/skippy-xd.service" "$HOME/.config/systemd/user/xfdashboard.service" \
      "$HOME/.config/systemd/user/picom.service"

echo "==> 2. Instalando stack base, Redshift, xfce4-screensaver, Feh, Zenity, Picom y Rofi..."
sudo apt install -y \
    lightdm lightdm-gtk-greeter lightdm-gtk-greeter-settings \
    xfce4-screensaver \
    redshift \
    feh zenity \
    xfce4-panel xfce4-pulseaudio-plugin xfce4-appfinder mugshot \
    xfce4-goodies xfce4-whiskermenu-plugin xfce4-notifyd xfce4-power-manager xfce4-screenshooter xfce4-xkb-plugin \
    xdotool brightnessctl pavucontrol network-manager-gnome \
    pipewire pipewire-pulse wireplumber \
    xdg-desktop-portal xdg-desktop-portal-gtk \
    tumbler ffmpegthumbnailer poppler-data tumbler-plugins-extra webp-pixbuf-loader \
    gvfs-backends gvfs-fuse policykit-1-gnome \
    fonts-inter fonts-jetbrains-mono \
    dconf-cli libglib2.0-bin libglib2.0-dev-bin libnotify-bin \
    libxcb-xrm0 \
    picom libchipmunk7 libgif7 libpng16-16t64 libxcomposite1 libxdamage1 libxft2 libxinerama1 libjpeg62 \
    rofi \
    flameshot tesseract-ocr tesseract-ocr-spa tesseract-ocr-eng xclip x11-utils imagemagick \
    nemo nemo-fileroller \
    curl wget git jq unzip

echo "==> 3. Instalando temas Orchis-Dark y Tela-circle-dark..."
if [ ! -d "/usr/share/themes/Orchis-Dark" ] || [ ! -d "/usr/share/icons/Tela-circle-dark" ]; then
    TEMP_DIR=$(mktemp -d)
    if [ ! -d "/usr/share/themes/Orchis-Dark" ]; then
        git clone --depth=1 https://github.com/vinceliuice/Orchis-theme.git "$TEMP_DIR/orchis"
        sudo "$TEMP_DIR/orchis/install.sh" -d "/usr/share/themes" -c dark -t default --round 10px
    fi
    if [ ! -d "/usr/share/icons/Tela-circle-dark" ]; then
        git clone --depth=1 https://github.com/vinceliuice/Tela-circle-icon-theme.git "$TEMP_DIR/tela"
        sudo "$TEMP_DIR/tela/install.sh" -d "/usr/share/icons"
    fi
    rm -rf "$TEMP_DIR"
fi

echo "==> 4. Configurando Vicinae e iconos..."
if ! command -v vicinae >/dev/null 2>&1; then
    curl -fsSL --connect-timeout 5 -m 30 https://vicinae.com/install | bash -s -- --prefix "$HOME/.local"
fi

if [ -f "$HOME/.local/bin/vicinae" ]; then
    sudo ln -sf "$HOME/.local/bin/vicinae" /usr/local/bin/vicinae 2>/dev/null || true
fi

systemctl --user stop ayatana-indicator-application.service indicator-application.service 2>/dev/null || true
systemctl --user mask ayatana-indicator-application.service indicator-application.service 2>/dev/null || true

mkdir -p "$HOME/.config/autostart"
for indicator_desktop in ayatana-indicator-application.desktop indicator-application.desktop; do
    cat <<EOF > "$HOME/.config/autostart/${indicator_desktop}"
[Desktop Entry]
Type=Application
Name=${indicator_desktop}
Exec=/bin/true
Hidden=true
X-GNOME-Autostart-enabled=false
EOF
done

find /usr/share/icons /usr/share/pixmaps "$HOME/.local/share/icons" \
     -type f \( -name "*vicinae*.svg" -o -name "*vicinae*.png" \) 2>/dev/null | while read -r f; do
    if grep -q "vicinae-grad" "$f" 2>/dev/null; then
        sudo rm -f "$f"
    fi
done

TMP_VIC=$(mktemp -d /tmp/vic_official.XXXXXX)
OFFICIAL_ICON=""

for local_p in "/usr/share/icons/hicolor/scalable/apps/vicinae.svg" \
               "/usr/share/pixmaps/vicinae.svg" \
               "$HOME/.local/share/icons/hicolor/scalable/apps/vicinae.svg" \
               "$HOME/.local/share/vicinae/icons/vicinae.svg"; do
    if [ -f "$local_p" ] && ! grep -q "vicinae-grad" "$local_p" 2>/dev/null; then
        OFFICIAL_ICON="$local_p"
        break
    fi
done

if [ -z "$OFFICIAL_ICON" ]; then
    CANDIDATE_URLS=(
        "https://raw.githubusercontent.com/vicinaehq/vicinae/main/crates/vicinae/assets/icon.svg"
        "https://raw.githubusercontent.com/vicinaehq/vicinae/main/assets/vicinae.svg"
        "https://raw.githubusercontent.com/vicinaehq/vicinae/main/data/icons/hicolor/scalable/apps/vicinae.svg"
        "https://vicinae.com/icon.svg"
        "https://vicinae.com/logo.svg"
    )
    for url in "${CANDIDATE_URLS[@]}"; do
        if curl -fsSL --connect-timeout 4 -m 8 "$url" -o "$TMP_VIC/downloaded.svg" 2>/dev/null; then
            if [ -s "$TMP_VIC/downloaded.svg" ] && ! grep -qi "404" "$TMP_VIC/downloaded.svg"; then
                OFFICIAL_ICON="$TMP_VIC/downloaded.svg"
                break
            fi
        fi
    done
fi

copy_safe() {
    local src="$1"
    local dst="$2"
    if [ -f "$dst" ] && [ "$src" -ef "$dst" ]; then
        return 0
    fi
    sudo cp -f "$src" "$dst"
}

if [ -n "$OFFICIAL_ICON" ] && [ -f "$OFFICIAL_ICON" ]; then
    EXT="${OFFICIAL_ICON##*.}"
    MASTER_ICON="$TMP_VIC/master.${EXT}"

    if [ ! "$OFFICIAL_ICON" -ef "$MASTER_ICON" ]; then
        cp -f "$OFFICIAL_ICON" "$MASTER_ICON"
    fi

    sudo mkdir -p /usr/share/pixmaps \
                  /usr/share/icons/hicolor/scalable/apps \
                  /usr/share/icons/hicolor/scalable/status \
                  /usr/share/icons/Tela-circle-dark/scalable/apps \
                  /usr/share/icons/Tela-circle-dark/scalable/panel

    for name in vicinae vicinae-tray vicinae-indicator vicinae-status com.vicinae.Vicinae; do
        copy_safe "$MASTER_ICON" "/usr/share/pixmaps/${name}.${EXT}"
        copy_safe "$MASTER_ICON" "/usr/share/icons/hicolor/scalable/apps/${name}.${EXT}"
        copy_safe "$MASTER_ICON" "/usr/share/icons/hicolor/scalable/status/${name}.${EXT}"
        copy_safe "$MASTER_ICON" "/usr/share/icons/Tela-circle-dark/scalable/apps/${name}.${EXT}"
        copy_safe "$MASTER_ICON" "/usr/share/icons/Tela-circle-dark/scalable/panel/${name}.${EXT}"
    done

    CONVERT_BIN=""
    command -v magick >/dev/null 2>&1 && CONVERT_BIN="magick"
    [ -z "$CONVERT_BIN" ] && command -v convert >/dev/null 2>&1 && CONVERT_BIN="convert"

    if [ -n "$CONVERT_BIN" ]; then
        for sz in 16 22 24 32 48; do
            sudo mkdir -p "/usr/share/icons/hicolor/${sz}x${sz}/status" \
                          "/usr/share/icons/hicolor/${sz}x${sz}/apps" \
                          "/usr/share/icons/Tela-circle-dark/${sz}x${sz}/panel" \
                          "/usr/share/icons/Tela-circle-dark/${sz}x${sz}/apps"

            $CONVERT_BIN "$MASTER_ICON" -background none -resize "${sz}x${sz}" "$TMP_VIC/icon_${sz}.png"

            for name in vicinae vicinae-tray vicinae-indicator vicinae-status com.vicinae.Vicinae; do
                copy_safe "$TMP_VIC/icon_${sz}.png" "/usr/share/icons/hicolor/${sz}x${sz}/status/${name}.png"
                copy_safe "$TMP_VIC/icon_${sz}.png" "/usr/share/icons/hicolor/${sz}x${sz}/apps/${name}.png"
                copy_safe "$TMP_VIC/icon_${sz}.png" "/usr/share/icons/Tela-circle-dark/${sz}x${sz}/panel/${name}.png"
                copy_safe "$TMP_VIC/icon_${sz}.png" "/usr/share/icons/Tela-circle-dark/${sz}x${sz}/apps/${name}.png"
                [ "$sz" -eq 22 ] && copy_safe "$TMP_VIC/icon_${sz}.png" "/usr/share/pixmaps/${name}.png"
            done
        done
    fi

    sudo gtk-update-icon-cache -f -q /usr/share/icons/hicolor 2>/dev/null || true
    sudo gtk-update-icon-cache -f -q /usr/share/icons/Tela-circle-dark 2>/dev/null || true
    gtk-update-icon-cache -f -q "$HOME/.local/share/icons/hicolor" 2>/dev/null || true
fi

rm -rf "$TMP_VIC"
TMP_VIC=""

cat <<'EOF' > "$HOME/.config/autostart/vicinae.desktop"
[Desktop Entry]
Type=Application
Name=Vicinae Daemon
Comment=Vicinae Background Service
Exec=sh -c "sleep 3 && exec /usr/local/bin/vicinae server"
Icon=vicinae
Terminal=false
Hidden=false
NoDisplay=false
X-GNOME-Autostart-enabled=true
X-GNOME-Autostart-Delay=3
EOF

killall -9 vicinae vicinae-server 2>/dev/null || true
killall -q ayatana-indicator-application-service indicator-application-service 2>/dev/null || true
(sleep 1 && /usr/local/bin/vicinae server >/dev/null 2>&1 &)

echo "==> 5. Configurando LightDM (Inicio) y xfce4-screensaver (Sin VT Switch)..."
echo "lightdm shared/default-x-display-manager select lightdm" | sudo debconf-set-selections
echo "/usr/sbin/lightdm" | sudo tee /etc/X11/default-display-manager >/dev/null
sudo systemctl disable gdm3 gdm 2>/dev/null || true
sudo systemctl enable lightdm.service --force 2>/dev/null || true
sudo ln -sf /lib/systemd/system/lightdm.service /etc/systemd/system/display-manager.service

sudo mkdir -p /etc/lightdm/lightdm.conf.d
sudo bash -c "cat <<'EOF' > /etc/lightdm/lightdm.conf.d/50-xfce-desktop.conf
[Seat:*]
user-session=xfce
greeter-session=lightdm-gtk-greeter
greeter-hide-users=false
EOF"

sudo bash -c "cat <<'EOF' > /etc/lightdm/lightdm-gtk-greeter.conf
[greeter]
theme-name = Orchis-Dark
icon-theme-name = Tela-circle-dark
font-name = Inter 10
background = #1e1e2e
user-background = true
indicators = ~host;~spacer;~clock;~spacer;~session;~power
clock-format = %A, %d de %B — %I:%M:%S %p
position = 50%,center 50%,center
default-user-image = #avatar-default
screensaver-timeout = 60
EOF"

# Configuración óptima de xfce4-screensaver (pantalla negra, 0% CPU y sin cambio de VT)
xfconf-query -c xfce4-screensaver -p /saver/enabled -n -t bool -s true 2>/dev/null || xfconf-query -c xfce4-screensaver -p /saver/enabled -s true 2>/dev/null || true
xfconf-query -c xfce4-screensaver -p /saver/mode -n -t int -s 0 2>/dev/null || xfconf-query -c xfce4-screensaver -p /saver/mode -s 0 2>/dev/null || true
xfconf-query -c xfce4-screensaver -p /lock/enabled -n -t bool -s true 2>/dev/null || xfconf-query -c xfce4-screensaver -p /lock/enabled -s true 2>/dev/null || true
xfconf-query -c xfce4-screensaver -p /lock/saver-activation/enabled -n -t bool -s true 2>/dev/null || xfconf-query -c xfce4-screensaver -p /lock/saver-activation/enabled -s true 2>/dev/null || true
xfconf-query -c xfce4-screensaver -p /lock/saver-activation/delay -n -t int -s 0 2>/dev/null || xfconf-query -c xfce4-screensaver -p /lock/saver-activation/delay -s 0 2>/dev/null || true

xfconf-query -c xfce4-power-manager -p /xfce4-power-manager/lock-screen-suspend-hibernate -n -t bool -s true 2>/dev/null || \
xfconf-query -c xfce4-power-manager -p /xfce4-power-manager/lock-screen-suspend-hibernate -s true 2>/dev/null || true

mkdir -p "$HOME/.config/autostart"
cat <<'EOF' > "$HOME/.config/autostart/xfce4-screensaver.desktop"
[Desktop Entry]
Type=Application
Name=Xfce Screensaver
Comment=Launch Screensaver and Locker
Exec=xfce4-screensaver
Icon=preferences-desktop-screensaver
Hidden=false
NoDisplay=false
X-GNOME-Autostart-enabled=true
EOF

mkdir -p "$HOME/.local/bin"
cat <<'EOF' > "$HOME/.local/bin/screenlock"
#!/usr/bin/env bash
set -euo pipefail
if command -v xfce4-screensaver-command >/dev/null 2>&1; then
    xfce4-screensaver-command --lock
else
    xflock4
fi
EOF
chmod +x "$HOME/.local/bin/screenlock"

# Solución definitiva al bug de pérdida de foco en X11/Xfwm4 tras desbloquear xfce4-screensaver
cat <<'EOF_FIXER' > "$HOME/.local/bin/xfce-unlock-fixer"
#!/usr/bin/env bash
# xfce-unlock-fixer v3: Monitorea D-Bus (sesión y sistema) para reactivar foco X11 y protocolo WM_TAKE_FOCUS (Ghostty, terminales, apps)
set -uo pipefail

reactivate_window() {
    # 1. Extraer ID hexadecimal limpio sin caracteres residuales ni comas (ej: 0x5400004)
    local win_id
    win_id=$(xprop -root _NET_ACTIVE_WINDOW 2>/dev/null | grep -oE '0x[0-9a-fA-F]+' | head -n 1 || true)

    # 2. Si no hay ventana activa reportada o es 0x0, buscar ventana de Ghostty visible
    if [ -z "$win_id" ] || [ "$win_id" = "0x0" ]; then
        win_id=$(xdotool search --onlyvisible --class "ghostty" 2>/dev/null | head -n 1 || true)
    fi

    # 3. Fallback: buscar cualquier ventana visible en la sesión
    if [ -z "$win_id" ] || [ "$win_id" = "0x0" ]; then
        win_id=$(xdotool search --onlyvisible --class ".*" 2>/dev/null | tail -n 1 || true)
    fi

    # 4. Activar y enfocar explícitamente para que Xfwm4 envíe WM_TAKE_FOCUS y FocusIn
    if [ -n "$win_id" ] && [ "$win_id" != "0x0" ]; then
        xdotool windowactivate "$win_id" 2>/dev/null || true
        xdotool windowfocus "$win_id" 2>/dev/null || true
    fi

    # 5. Señal de refresco al compositor Picom
    pkill -USR1 picom 2>/dev/null || true
}

# Subproceso 1: Escuchar desbloqueo de screensaver (bus de sesión)
(
    dbus-monitor --session "type='signal',interface='org.xfce.ScreenSaver',member='ActiveChanged'"                            "type='signal',interface='org.freedesktop.ScreenSaver',member='ActiveChanged'" 2>/dev/null |     while read -r line; do
        if echo "$line" | grep -q "boolean false"; then
            sleep 0.12
            reactivate_window
        fi
    done
) &

# Subproceso 2: Escuchar reanudación tras suspensión del sistema (bus del sistema)
(
    dbus-monitor --system "type='signal',interface='org.freedesktop.login1.Manager',member='PrepareForSleep'" 2>/dev/null |     while read -r line; do
        if echo "$line" | grep -q "boolean false"; then
            sleep 0.4
            reactivate_window
        fi
    done
) &

wait
EOF_FIXER
chmod +x "$HOME/.local/bin/xfce-unlock-fixer"
sudo ln -sf "$HOME/.local/bin/xfce-unlock-fixer" /usr/local/bin/xfce-unlock-fixer 2>/dev/null || true

cat <<'EOF_FIXER_DESKTOP' > "$HOME/.config/autostart/xfce-unlock-fixer.desktop"
[Desktop Entry]
Type=Application
Name=Xfce Unlock Focus Fixer
Comment=Restaura automáticamente el foco del teclado y ratón al desbloquear
Exec=sh -c "pkill -f xfce-unlock-fixer; exec $HOME/.local/bin/xfce-unlock-fixer"
Hidden=false
NoDisplay=true
X-GNOME-Autostart-enabled=true
EOF_FIXER_DESKTOP

pkill -f xfce-unlock-fixer 2>/dev/null || true
("$HOME/.local/bin/xfce-unlock-fixer" >/dev/null 2>&1 &) || true



# Ajuste de DPMS en xfce4-power-manager para evitar apagados prematuros y pérdida de teclas
xfconf-query -c xfce4-power-manager -p /xfce4-power-manager/dpms-enabled -n -t bool -s true 2>/dev/null || xfconf-query -c xfce4-power-manager -p /xfce4-power-manager/dpms-enabled -s true 2>/dev/null || true

xfconf-query -c xfce4-power-manager -p /xfce4-power-manager/blank-on-ac -n -t int -s 0 2>/dev/null || xfconf-query -c xfce4-power-manager -p /xfce4-power-manager/blank-on-ac -s 0 2>/dev/null || true

xfconf-query -c xfce4-power-manager -p /xfce4-power-manager/dpms-on-ac-sleep -n -t int -s 20 2>/dev/null || xfconf-query -c xfce4-power-manager -p /xfce4-power-manager/dpms-on-ac-sleep -s 20 2>/dev/null || true

xfconf-query -c xfce4-power-manager -p /xfce4-power-manager/dpms-on-ac-off -n -t int -s 30 2>/dev/null || xfconf-query -c xfce4-power-manager -p /xfce4-power-manager/dpms-on-ac-off -s 30 2>/dev/null || true

pgrep -f xfce4-screensaver >/dev/null || (xfce4-screensaver >/dev/null 2>&1 &)

echo "==> 6. Configuración de hardware (Touchpad, PipeWire, Polkit)..."
sudo mkdir -p /etc/X11/xorg.conf.d
sudo bash -c "cat <<'EOF' > /etc/X11/xorg.conf.d/40-libinput.conf
Section \"InputClass\"
    Identifier \"libinput touchpad catchall\"
    MatchIsTouchpad \"on\"
    MatchDevicePath \"/dev/input/event*\"
    Driver \"libinput\"
    Option \"Tapping\" \"on\"
    Option \"NaturalScrolling\" \"true\"
    Option \"ClickMethod\" \"buttonareas\"
    Option \"DisableWhileTyping\" \"true\"
EndSection
EOF"

systemctl --user enable --now pipewire pipewire-pulse wireplumber 2>/dev/null || true

echo "==> 6.1. Optimizando rendimiento de CPU y eliminando estrangulamiento (TLP + Thermald)..."
# Desactivar power-profiles-daemon para que no entre en conflicto con TLP ni estrangule la CPU
if systemctl is-active power-profiles-daemon >/dev/null 2>&1 || systemctl is-enabled power-profiles-daemon >/dev/null 2>&1; then
    sudo systemctl stop power-profiles-daemon 2>/dev/null || true
    sudo systemctl disable power-profiles-daemon 2>/dev/null || true
    sudo systemctl mask power-profiles-daemon 2>/dev/null || true
fi
sudo systemctl enable --now tlp 2>/dev/null || true

# Desactivar thermald para evitar el bug crítico de Haswell/Dell que traba la CPU en 800MHz tras suspensión
if systemctl is-enabled thermald >/dev/null 2>&1 || systemctl is-active thermald >/dev/null 2>&1; then
    sudo systemctl stop thermald 2>/dev/null || true
    sudo systemctl disable thermald 2>/dev/null || true
    sudo systemctl mask thermald 2>/dev/null || true
fi

# Configuración de alto rendimiento en TLP para garantizar máximo clock en AC y batería
sudo mkdir -p /etc/tlp.d
sudo bash -c "cat <<'EOF_TLP' > /etc/tlp.d/99-performance.conf
# Perfil forzado de máximo rendimiento sin reducción de potencia
CPU_SCALING_GOVERNOR_ON_AC=performance
CPU_SCALING_GOVERNOR_ON_BAT=performance
CPU_ENERGY_PERF_POLICY_ON_AC=performance
CPU_ENERGY_PERF_POLICY_ON_BAT=performance
CPU_MIN_PERF_ON_AC=0
CPU_MAX_PERF_ON_AC=100
CPU_MIN_PERF_ON_BAT=0
CPU_MAX_PERF_ON_BAT=100
CPU_BOOST_ON_AC=1
CPU_BOOST_ON_BAT=1

# Desactivar USB autosuspend global y proteger adaptador Intel Bluetooth 8087:07dc
USB_AUTOSUSPEND=0
USB_DENYLIST="8087:07dc"

# Mantener PCIe y GPU Nouveau en rendimiento óptimo
PCIE_ASPM_ON_AC=performance
PCIE_ASPM_ON_BAT=performance
RADEON_DPM_PERFLVL_ON_AC=high
RADEON_DPM_PERFLVL_ON_BAT=high
EOF_TLP"

if command -v tlp >/dev/null 2>&1; then
    sudo tlp start 2>/dev/null || true
fi

# Aplicar gobernador 'performance' inmediatamente a todos los núcleos activos
for gov in /sys/devices/system/cpu/cpu*/cpufreq/scaling_governor; do
    [ -f "$gov" ] && echo "performance" | sudo tee "$gov" >/dev/null 2>&1 || true
done
for epb in /sys/devices/system/cpu/cpu*/power/energy_perf_bias; do
    [ -f "$epb" ] && echo "0" | sudo tee "$epb" >/dev/null 2>&1 || true
done

echo "==> 6.2. Asegurando persistencia de Bluetooth y reactivación sin fallos tras suspensión..."
# Regla udev para evitar que el kernel ponga en autosuspend el chip Bluetooth USB (8087:07dc)
sudo bash -c "cat <<'EOF_BT_UDEV' > /etc/udev/rules.d/50-bluetooth-power.rules
# Desactivar autosuspend para chip Intel Bluetooth (8087:07dc) y controladores btusb
ACTION=="add", SUBSYSTEM=="usb", ATTR{idVendor}=="8087", ATTR{idProduct}=="07dc", ATTR{power/control}="on"
ACTION=="add", SUBSYSTEM=="usb", ATTR{bInterfaceClass}=="e0", ATTR{bInterfaceSubClass}=="01", ATTR{power/control}="on"
ACTION=="add", SUBSYSTEM=="bluetooth", ATTR{power/control}="on"
EOF_BT_UDEV"
sudo udevadm control --reload-rules 2>/dev/null || true
sudo udevadm trigger --subsystem-match=usb 2>/dev/null || true

# Configuración de BlueZ para reconexión inmediata y modo siempre activo
if [ -f /etc/bluetooth/main.conf ]; then
    sudo sed -i 's/^#*AutoEnable=.*/AutoEnable=true/' /etc/bluetooth/main.conf 2>/dev/null || true
    sudo sed -i 's/^#*FastConnectable=.*/FastConnectable=true/' /etc/bluetooth/main.conf 2>/dev/null || true
    grep -q "^AutoEnable=" /etc/bluetooth/main.conf || sudo bash -c 'echo "AutoEnable=true" >> /etc/bluetooth/main.conf'
    grep -q "^FastConnectable=" /etc/bluetooth/main.conf || sudo bash -c 'echo "FastConnectable=true" >> /etc/bluetooth/main.conf'
fi

# Hook de sistema systemd-sleep para forzar máximo rendimiento y despertar limpio de Bluetooth y Picom al salir de suspensión
sudo mkdir -p /etc/systemd/system-sleep
sudo bash -c "cat <<'EOF_SLEEP' > /etc/systemd/system-sleep/99-performance-resume.sh
#!/bin/sh
case "\$1" in
    post)
        # 1. Restaurar frecuencia máxima y gobernador de CPU
        for g in /sys/devices/system/cpu/cpu*/cpufreq/scaling_governor; do
            [ -f "\$g" ] && echo performance > "\$g" 2>/dev/null || true
        done
        for p in /sys/devices/system/cpu/cpu*/power/energy_perf_bias; do
            [ -f "\$p" ] && echo 0 > "\$p" 2>/dev/null || true
        done

        # 2. Desbloquear rfkill y asegurar que hci0 esté activo
        rfkill unblock bluetooth 2>/dev/null || true
        command -v hciconfig >/dev/null 2>&1 && hciconfig hci0 up 2>/dev/null || true

        # 3. Forzar reinicio de TLP para re-aplicar perfiles limpios
        command -v tlp >/dev/null 2>&1 && tlp start 2>/dev/null || true

        # 4. Señal de refresco al compositor Picom en sesiones X11 activas
        pkill -USR1 picom 2>/dev/null || true
        ;;
esac
EOF_SLEEP"
sudo chmod +x /etc/systemd/system-sleep/99-performance-resume.sh


POLKIT_BIN="/usr/lib/policykit-1-gnome/polkit-gnome-authentication-agent-1"
[ -f "/usr/libexec/polkit-gnome-authentication-agent-1" ] && POLKIT_BIN="/usr/libexec/polkit-gnome-authentication-agent-1"
cat <<EOF > "$HOME/.config/autostart/polkit-gnome.desktop"
[Desktop Entry]
Type=Application
Exec=$POLKIT_BIN
Hidden=false
NoDisplay=false
X-GNOME-Autostart-enabled=true
Name=PolicyKit Authentication Agent
EOF

echo "==> 7. Configurando Picom, foco de Xfwm4 y Rofi..."
xfconf-query -c xfwm4 -p /general/use_compositing -s false 2>/dev/null || true
xfconf-query -c xfwm4 -p /general/show_frame_shadow -s false 2>/dev/null || true
xfconf-query -c xfwm4 -p /general/show_popup_shadow -s false 2>/dev/null || true
xfconf-query -c xfwm4 -p /general/show_dock_shadow -s false 2>/dev/null || true
xfconf-query -c xfwm4 -p /general/raise_on_focus -n -t bool -s true 2>/dev/null || xfconf-query -c xfwm4 -p /general/raise_on_focus -s true 2>/dev/null || true
xfconf-query -c xfwm4 -p /general/activate_action -n -t string -s "bring" 2>/dev/null || xfconf-query -c xfwm4 -p /general/activate_action -s "bring" 2>/dev/null || true
xfconf-query -c xfwm4 -p /general/focus_hint -n -t bool -s true 2>/dev/null || xfconf-query -c xfwm4 -p /general/focus_hint -s true 2>/dev/null || true
xfconf-query -c xfwm4 -p /general/click_to_focus -n -t bool -s true 2>/dev/null || xfconf-query -c xfwm4 -p /general/click_to_focus -s true 2>/dev/null || true
xfconf-query -c xfwm4 -p /general/prevent_focus_stealing -n -t bool -s false 2>/dev/null || xfconf-query -c xfwm4 -p /general/prevent_focus_stealing -s false 2>/dev/null || true
xfconf-query -c xfwm4 -p /general/focus_new -n -t bool -s true 2>/dev/null || xfconf-query -c xfwm4 -p /general/focus_new -s true 2>/dev/null || true

mkdir -p "$HOME/.config/picom"
cat <<'EOF' > "$HOME/.config/picom/picom.conf"
backend = "glx";
vsync = true;
use-damage = true;
unredir-if-possible = false;

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

systemctl --user stop picom.service 2>/dev/null || true
systemctl --user disable picom.service 2>/dev/null || true
rm -f "$HOME/.config/systemd/user/picom.service"

mkdir -p "$HOME/.config/autostart"
cat <<'EOF' > "$HOME/.config/autostart/picom.desktop"
[Desktop Entry]
Type=Application
Exec=sh -c "killall -q picom; exec picom -b --config $HOME/.config/picom/picom.conf"
Hidden=false
NoDisplay=false
X-GNOME-Autostart-enabled=true
Name=Picom
Comment=Lightweight X11 Compositor
EOF

killall -q picom 2>/dev/null || true
(picom -b --config "$HOME/.config/picom/picom.conf" >/dev/null 2>&1 &) || true

mkdir -p "$HOME/.config/rofi"
cat <<'EOF' > "$HOME/.config/rofi/config.rasi"
configuration {
    modes: "window,drun,run";
    font: "Inter 10";
    show-icons: true;
    icon-theme: "Tela-circle-dark";
    terminal: "ghostty";
    disable-history: false;
    display-window: " 󰕰  Ventanas ";
    display-drun: " 󰀻  Apps ";
    display-run: " 󰌆  Run ";
}

* {
    bg: #1e1e2e;
    bg-alt: #313244;
    fg: #cdd6f4;
    fg-alt: #a6adc8;
    accent: #89b4fa;
    border-col: #45475a;
    background-color: transparent;
    text-color: @fg;
    margin: 0;
    padding: 0;
    spacing: 0;
}

window {
    background-color: @bg;
    border: 2px;
    border-color: @border-col;
    border-radius: 10px;
    width: 650px;
    padding: 12px;
}

mainbox {
    children: [inputbar, listview];
    spacing: 8px;
}

inputbar {
    children: [prompt, entry];
    background-color: @bg-alt;
    border-radius: 8px;
    padding: 8px 12px;
    spacing: 8px;
}

prompt {
    text-color: @accent;
}

entry {
    placeholder: "Filtrar ventanas...";
    placeholder-color: @fg-alt;
}

listview {
    lines: 8;
    columns: 1;
    fixed-height: false;
    scrollbar: false;
}

element {
    padding: 8px 12px;
    border-radius: 6px;
    spacing: 10px;
}

element selected {
    background-color: @accent;
    text-color: #11111b;
}

element-icon {
    size: 24px;
}

element-text {
    vertical-align: 0.5;
    text-color: inherit;
}
EOF

xfconf-query -c xfce4-keyboard-shortcuts -p "/commands/custom/<Alt>Tab" -r 2>/dev/null || true
xfconf-query -c xfce4-keyboard-shortcuts -p "/commands/custom/<Alt><Shift>Tab" -r 2>/dev/null || true
xfconf-query -c xfce4-keyboard-shortcuts -p "/commands/custom/<Super>Tab" -r 2>/dev/null || true

xfconf-query -c xfce4-keyboard-shortcuts -p "/xfwm4/custom/<Alt>Tab" -n -t string -s "cycle_windows_key" 2>/dev/null || \
xfconf-query -c xfce4-keyboard-shortcuts -p "/xfwm4/custom/<Alt>Tab" -s "cycle_windows_key" 2>/dev/null || true

xfconf-query -c xfce4-keyboard-shortcuts -p "/xfwm4/custom/<Alt><Shift>Tab" -n -t string -s "cycle_reverse_windows_key" 2>/dev/null || \
xfconf-query -c xfce4-keyboard-shortcuts -p "/xfwm4/custom/<Alt><Shift>Tab" -s "cycle_reverse_windows_key" 2>/dev/null || true

xfconf-query -c xfwm4 -p /general/cycle_preview -s true 2>/dev/null || true
xfconf-query -c xfwm4 -p /general/cycle_tabwin_mode -s 1 2>/dev/null || true
xfconf-query -c xfwm4 -p /general/cycle_minimum -s true 2>/dev/null || true
xfconf-query -c xfwm4 -p /general/cycle_hidden -s true 2>/dev/null || true

xfconf-query -c xfce4-keyboard-shortcuts -p "/commands/custom/<Alt>slash" -n -t string -s "rofi -show window" 2>/dev/null || \
xfconf-query -c xfce4-keyboard-shortcuts -p "/commands/custom/<Alt>slash" -s "rofi -show window" 2>/dev/null || true

echo "==> 8. Configurando motor Feh y selector gráfico de Wallpapers..."
mkdir -p "$HOME/Pictures"

sudo bash -c 'cat <<'"'"'EOF'"'"' > /usr/local/bin/wallpaper-picker
#!/usr/bin/env bash
set -euo pipefail

INITIAL_DIR="$HOME/Pictures"
[ ! -d "$INITIAL_DIR" ] && INITIAL_DIR="/usr/share/backgrounds"

FILE=$(zenity --file-selection \
    --title="Seleccionar Fondo de Pantalla" \
    --filename="${INITIAL_DIR}/" \
    --file-filter="Imágenes (*.png, *.jpg, *.jpeg, *.webp) | *.png *.jpg *.jpeg *.webp *.PNG *.JPG *.JPEG *.WEBP" \
    --file-filter="Todos los archivos | *" 2>/dev/null || true)

if [ -n "$FILE" ] && [ -f "$FILE" ]; then
    feh --bg-fill "$FILE"
    notify-send -t 2000 -i preferences-desktop-wallpaper "Fondo de Pantalla" "Fondo actualizado correctamente" 2>/dev/null || true
fi
EOF'
sudo chmod +x /usr/local/bin/wallpaper-picker

mkdir -p "$HOME/.local/share/applications"
cat <<'EOF' > "$HOME/.local/share/applications/wallpaper-picker.desktop"
[Desktop Entry]
Type=Application
Name=Cambiar Fondo de Pantalla
Comment=Selector gráfico y ligero de fondos de pantalla
Exec=/usr/local/bin/wallpaper-picker
Icon=preferences-desktop-wallpaper
Terminal=false
Categories=Settings;DesktopSettings;
EOF

if [ ! -f "$HOME/.fehbg" ]; then
    FALLBACK_IMG=""
    for cand in /usr/share/backgrounds/*.png /usr/share/backgrounds/*.jpg /usr/share/xfce4/backdrops/*.png; do
        if [ -f "$cand" ]; then
            FALLBACK_IMG="$cand"
            break
        fi
    done

    if [ -n "$FALLBACK_IMG" ]; then
        feh --bg-fill "$FALLBACK_IMG" 2>/dev/null || true
    fi
else
    sh "$HOME/.fehbg" 2>/dev/null || true
fi

cat <<'EOF' > "$HOME/.config/autostart/wallpaper-restore.desktop"
[Desktop Entry]
Type=Application
Exec=sh -c "test -f $HOME/.fehbg && exec $HOME/.fehbg"
Hidden=false
NoDisplay=false
X-GNOME-Autostart-enabled=true
Name=Wallpaper Restore
Comment=Restaura el fondo de pantalla guardado usando feh
EOF

xfconf-query -c xfce4-session -p /sessions/Failsafe/Client4_Command -s "/bin/true" -t string 2>/dev/null || true
xfconf-query -c xfce4-session -p /sessions/Failsafe/Count -s 4 2>/dev/null || true

echo "==> 9. Configurando atajos globales y XKB nativo..."
xfconf-query -c xsettings -p /Net/ThemeName -s "Orchis-Dark" 2>/dev/null || true
xfconf-query -c xsettings -p /Net/IconThemeName -s "Tela-circle-dark" 2>/dev/null || true
xfconf-query -c xsettings -p /Gtk/FontName -s "Inter 10" 2>/dev/null || true
xfconf-query -c xsettings -p /Gtk/MonospaceFontName -s "JetBrains Mono 10" 2>/dev/null || true

xfconf-query -c xfwm4 -p /general/theme -s "Orchis-Dark" 2>/dev/null || true
xfconf-query -c xfwm4 -p /general/title_font -s "Inter Bold 10" 2>/dev/null || true
xfconf-query -c xfwm4 -p /general/button_layout -s "CHM|T" 2>/dev/null || true
xfconf-query -c xfwm4 -p /general/borderless_maximize -s true 2>/dev/null || true

xfconf-query -c xfce4-keyboard-shortcuts -p "/commands/custom/Super_L" -n -t string -s "xfce4-popup-whiskermenu" 2>/dev/null || \
xfconf-query -c xfce4-keyboard-shortcuts -p "/commands/custom/Super_L" -s "xfce4-popup-whiskermenu" 2>/dev/null || true

xfconf-query -c xfce4-keyboard-shortcuts -p "/commands/custom/<Alt>F1" -n -t string -s "xfce4-popup-whiskermenu" 2>/dev/null || \
xfconf-query -c xfce4-keyboard-shortcuts -p "/commands/custom/<Alt>F1" -s "xfce4-popup-whiskermenu" 2>/dev/null || true

xfconf-query -c xfce4-keyboard-shortcuts -p "/commands/custom/<Super>space" -n -t string -s "vicinae toggle" 2>/dev/null || \
xfconf-query -c xfce4-keyboard-shortcuts -p "/commands/custom/<Super>space" -s "vicinae toggle" 2>/dev/null || true

xfconf-query -c xfce4-keyboard-shortcuts -p "/commands/custom/<Super>v" -n -t string -s "vicinae cmd launch clipboard:history" 2>/dev/null || \
xfconf-query -c xfce4-keyboard-shortcuts -p "/commands/custom/<Super>v" -s "vicinae cmd launch clipboard:history" 2>/dev/null || true

xfconf-query -c xfce4-keyboard-shortcuts -p "/commands/custom/<Super>period" -n -t string -s "vicinae cmd launch core:search-emojis" 2>/dev/null || \
xfconf-query -c xfce4-keyboard-shortcuts -p "/commands/custom/<Super>period" -s "vicinae cmd launch core:search-emojis" 2>/dev/null || true

xfconf-query -c xfce4-keyboard-shortcuts -p "/commands/custom/<Super>l" -n -t string -s "$HOME/.local/bin/screenlock" 2>/dev/null || \
xfconf-query -c xfce4-keyboard-shortcuts -p "/commands/custom/<Super>l" -s "$HOME/.local/bin/screenlock" 2>/dev/null || true

im-config -n none 2>/dev/null || true
echo "run_im none" > "$HOME/.xinputrc"
systemctl --user mask ibus.service 2>/dev/null || true
gsettings set org.freedesktop.ibus.panel show 0 2>/dev/null || true
gsettings set org.freedesktop.ibus.general.panels show 0 2>/dev/null || true
killall -q ibus-daemon 2>/dev/null || true

xfconf-query -c keyboard-layout -p /Default/XkbDisable -n -t bool -s false 2>/dev/null || \
xfconf-query -c keyboard-layout -p /Default/XkbDisable -s false 2>/dev/null || true

xfconf-query -c keyboard-layout -p /Default/XkbLayout -n -t string -s "us,es" 2>/dev/null || \
xfconf-query -c keyboard-layout -p /Default/XkbLayout -s "us,es" 2>/dev/null || true

xfconf-query -c keyboard-layout -p /Default/XkbVariant -n -t string -s "," 2>/dev/null || \
xfconf-query -c keyboard-layout -p /Default/XkbVariant -s "," 2>/dev/null || true

xfconf-query -c keyboard-layout -p "/Default/XkbOptions/Group" -n -t string -s "grp:alt_shift_toggle" 2>/dev/null || \
xfconf-query -c keyboard-layout -p "/Default/XkbOptions/Group" -s "grp:alt_shift_toggle" 2>/dev/null || true

setxkbmap -layout "us,es" -variant "," -option "grp:alt_shift_toggle" 2>/dev/null || true

xfconf-query -c xfce4-keyboard-shortcuts -p "/commands/custom/<Primary><Alt>n" -n -t string -s "/usr/local/bin/toggle-redshift" 2>/dev/null || xfconf-query -c xfce4-keyboard-shortcuts -p "/commands/custom/<Primary><Alt>n" -s "/usr/local/bin/toggle-redshift" 2>/dev/null || true

echo "==> 10. Integrando capturas de pantalla (Flameshot) y OCR..."
mkdir -p "$HOME/.local/bin" "$HOME/.config/flameshot"
cat <<'EOF' > "$HOME/.config/flameshot/flameshot.ini"
[General]
showSelectionGeometry=0
showSelectionGeometryHideTime=0
showMagnifier=false
showHelp=false
showSidePanelButton=false
disabledTrayIcon=true
autoCloseIdleDaemon=true
contrastUiColor=#89b4fa
uiColor=#1e1e2e
drawColor=#89b4fa
EOF

killall -q flameshot 2>/dev/null || true

cat <<'EOF' > "$HOME/.local/bin/cap-area"
#!/usr/bin/env bash
set -euo pipefail
flameshot gui
EOF

cat <<'EOF' > "$HOME/.local/bin/cap-repeat"
#!/usr/bin/env bash
set -euo pipefail
flameshot gui --last-region -c 2>/dev/null || true
notify-send -t 1500 -i camera-photo "Captura de Pantalla" "Última región copiada al portapapeles" 2>/dev/null || true
EOF

cat <<'EOF' > "$HOME/.local/bin/cap-window"
#!/usr/bin/env bash
set -euo pipefail
INFO=$(xwininfo -frame 2>/dev/null || true)
[ -z "$INFO" ] && exit 0
X=$(echo "$INFO" | awk '/Absolute upper-left X:/ {print $4}')
Y=$(echo "$INFO" | awk '/Absolute upper-left Y:/ {print $4}')
W=$(echo "$INFO" | awk '/Width:/ {print $2}')
H=$(echo "$INFO" | awk '/Height:/ {print $2}')
[ -z "$W" ] || [ -z "$H" ] || [ -z "$X" ] || [ -z "$Y" ] && exit 0
REGION="${W}x${H}+${X}+${Y}"
flameshot screen --region "$REGION" --raw 2>/dev/null | xclip -selection clipboard -t image/png || \
    flameshot screen --region "$REGION" -c 2>/dev/null || true
notify-send -t 1500 -i camera-photo "Captura de Ventana" "Ventana copiada al portapapeles (${REGION})" 2>/dev/null || true
EOF

cat <<'EOF' > "$HOME/.local/bin/cap-fullscreen"
#!/usr/bin/env bash
set -euo pipefail
flameshot full --raw 2>/dev/null | xclip -selection clipboard -t image/png || flameshot full -c 2>/dev/null || true
notify-send -t 1500 -i camera-photo "Captura Completa" "Pantalla completa copiada al portapapeles" 2>/dev/null || true
EOF

cat <<'EOF' > "$HOME/.local/bin/cap-ocr"
#!/usr/bin/env bash
set -uo pipefail
TMP_IMG=$(mktemp --suffix=.png)
trap 'rm -f "$TMP_IMG"' EXIT
if ! flameshot gui --raw > "$TMP_IMG" 2>/dev/null; then exit 0; fi
[ ! -s "$TMP_IMG" ] && exit 0
if command -v magick >/dev/null 2>&1; then
    magick "$TMP_IMG" -colorspace Gray -sharpen 0x1 -contrast-stretch 0.15%x0.05% "$TMP_IMG" 2>/dev/null || true
elif command -v convert >/dev/null 2>&1; then
    convert "$TMP_IMG" -colorspace Gray -sharpen 0x1 -contrast-stretch 0.15%x0.05% "$TMP_IMG" 2>/dev/null || true
fi
TEXT=$(tesseract "$TMP_IMG" stdout -l spa+eng --psm 6 2>/dev/null || true)
CLEAN_TEXT=$(echo "$TEXT" | sed -e 's/^[[:space:]]*//' -e 's/[[:space:]]*$//')
if [ -n "$CLEAN_TEXT" ]; then
    printf "%s" "$CLEAN_TEXT" | xclip -selection clipboard
    PREVIEW=$(echo "$CLEAN_TEXT" | tr '\n' ' ' | head -c 120)
    notify-send -t 3000 -i accessories-character-map "OCR Copiado al Portapapeles" "$PREVIEW"
else
    notify-send -t 2000 -i dialog-warning "OCR" "No se detectó texto en la selección"
fi
EOF

chmod +x "$HOME/.local/bin/cap-area" "$HOME/.local/bin/cap-repeat" "$HOME/.local/bin/cap-window" \
         "$HOME/.local/bin/cap-fullscreen" "$HOME/.local/bin/cap-ocr"
sudo ln -sf "$HOME/.local/bin/cap-area" /usr/local/bin/cap-area 2>/dev/null || true
sudo ln -sf "$HOME/.local/bin/cap-repeat" /usr/local/bin/cap-repeat 2>/dev/null || true
sudo ln -sf "$HOME/.local/bin/cap-window" /usr/local/bin/cap-window 2>/dev/null || true
sudo ln -sf "$HOME/.local/bin/cap-fullscreen" /usr/local/bin/cap-fullscreen 2>/dev/null || true
sudo ln -sf "$HOME/.local/bin/cap-ocr" /usr/local/bin/cap-ocr 2>/dev/null || true

xfconf-query -c xfce4-keyboard-shortcuts -p "/commands/custom/<Primary><Alt><Super>1" -n -t string -s "$HOME/.local/bin/cap-area" 2>/dev/null || true
xfconf-query -c xfce4-keyboard-shortcuts -p "/commands/custom/<Primary><Alt><Super>2" -n -t string -s "$HOME/.local/bin/cap-repeat" 2>/dev/null || true
xfconf-query -c xfce4-keyboard-shortcuts -p "/commands/custom/<Primary><Alt><Super>3" -n -t string -s "$HOME/.local/bin/cap-window" 2>/dev/null || true
xfconf-query -c xfce4-keyboard-shortcuts -p "/commands/custom/<Primary><Alt><Super>4" -n -t string -s "$HOME/.local/bin/cap-fullscreen" 2>/dev/null || true
xfconf-query -c xfce4-keyboard-shortcuts -p "/commands/custom/<Primary><Alt><Super>0" -n -t string -s "$HOME/.local/bin/cap-ocr" 2>/dev/null || true

echo "==> 11. Integrando Nemo..."
xdg-mime default nemo.desktop inode/directory
xdg-mime default nemo.desktop application/x-gnome-saved-search
gio mime inode/directory nemo.desktop 2>/dev/null || true

mkdir -p "$HOME/.local/share/xfce4/helpers" "$HOME/.config/xfce4"
cat <<'EOF' > "$HOME/.local/share/xfce4/helpers/nemo.desktop"
[Desktop Entry]
Version=1.0
Icon=system-file-manager
Type=X-XFCE-Helper
Name=Nemo
StartupNotify=true
X-XFCE-Binaries=nemo;
X-XFCE-Category=FileManager
X-XFCE-Commands=%B;
X-XFCE-CommandsWithParameter=%B "%s";
EOF
echo "FileManager=nemo" > "$HOME/.config/xfce4/helpers.rc"
gsettings set org.nemo.desktop show-desktop-icons false 2>/dev/null || true

echo "==> 12. Configurando Panel Xfce y Notificaciones..."
mkdir -p "$HOME/.themes/Orchis-Dark/xfce-notify-4.0"
cat <<'EOF' > "$HOME/.themes/Orchis-Dark/xfce-notify-4.0/gtk.css"
#XfceNotifyWindow {
    background-color: #1e1e2e;
    color: #cdd6f4;
    border: 2px solid #45475a;
    border-radius: 10px;
    padding: 12px;
}
#XfceNotifyWindow label#summary {
    font-weight: bold;
    color: #89b4fa;
    font-size: 11pt;
}
#XfceNotifyWindow label#body {
    color: #cdd6f4;
}
#XfceNotifyWindow button {
    background-image: none;
    background-color: #313244;
    color: #cdd6f4;
    border: 1px solid #45475a;
    border-radius: 6px;
    padding: 4px 8px;
}
#XfceNotifyWindow button:hover {
    background-color: #45475a;
    color: #ffffff;
}
EOF

xfconf-query -c xfce4-notifyd -p /theme -n -t string -s "Orchis-Dark" 2>/dev/null || \
xfconf-query -c xfce4-notifyd -p /theme -s "Orchis-Dark" 2>/dev/null || true

xfconf-query -c xfce4-notifyd -p /notification-log -n -t bool -s true 2>/dev/null || \
xfconf-query -c xfce4-notifyd -p /notification-log -s true 2>/dev/null || true

xfconf-query -c xfce4-panel -p /plugins/plugin-1 -s "whiskermenu" 2>/dev/null || true

XKB_PLUGIN=$(xfconf-query -c xfce4-panel -p /plugins -l 2>/dev/null | grep -E '^/plugins/plugin-[0-9]+$' | while read -r p; do
    [ "$(xfconf-query -c xfce4-panel -p "$p" 2>/dev/null || true)" = "xkb" ] && echo "$p" && break
done)

if [ -z "$XKB_PLUGIN" ]; then
    MAX_ID=$(xfconf-query -c xfce4-panel -p /plugins -l 2>/dev/null | grep -oE '[0-9]+' | sort -n | tail -1 || echo 0)
    NEW_ID=$(( MAX_ID + 1 ))
    XKB_PLUGIN="/plugins/plugin-$NEW_ID"
    xfconf-query -c xfce4-panel -p "$XKB_PLUGIN" -n -t string -s "xkb" 2>/dev/null || true

    EXISTING_IDS=$(xfconf-query -c xfce4-panel -p /panels/panel-1/plugin-ids 2>/dev/null | grep -E '^[0-9]+$' || true)
    PANEL_ARGS=()
    for id in $EXISTING_IDS; do PANEL_ARGS+=(-t int -s "$id"); done
    PANEL_ARGS+=(-t int -s "$NEW_ID")
    xfconf-query -c xfce4-panel -p /panels/panel-1/plugin-ids -n -a "${PANEL_ARGS[@]}" 2>/dev/null || \
    xfconf-query -c xfce4-panel -p /panels/panel-1/plugin-ids -a "${PANEL_ARGS[@]}" 2>/dev/null || true
fi

xfconf-query -c xfce4-panel -p "$XKB_PLUGIN/display-type" -n -t int -s 0 2>/dev/null || \
xfconf-query -c xfce4-panel -p "$XKB_PLUGIN/display-type" -s 0 2>/dev/null || true

xfce4-panel -r 2>/dev/null || true

echo "==> 13. Configurando Luz Nocturna / Redshift (Binario Instantáneo, Horario Configurable)..."
mkdir -p "$HOME/.config" "$HOME/.local/bin" "$HOME/.local/state" "$HOME/.local/share/applications" "$HOME/.config/systemd/user"

# 1. Purgar cualquier autostart o demonio con curvas de transición progresiva
rm -f "$HOME/.config/autostart/redshift.desktop" "$HOME/.config/redshift/redshift.conf" 2>/dev/null || true
pkill -9 -x redshift 2>/dev/null || true

# 2. Configuración declarativa de Luz Nocturna (editable por el usuario)
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

# 3. Crear script de control binario e idempotente con soporte de horario y RandR LUT
cat <<'EOF' > "$HOME/.local/bin/toggle-redshift"
#!/usr/bin/env bash
# ==============================================================================
# toggle-redshift: Interruptor binario y programador automático para Luz Nocturna
# Optimizado para Dell Precision M6800 (Nouveau Kepler / Dual Display DP-2 + eDP-1)
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
sudo cp -f "$HOME/.local/bin/toggle-redshift" /usr/local/bin/toggle-redshift 2>/dev/null || true

# 4. Registrar acceso de escritorio (.desktop) para indexación inmediata en Vicinae
mkdir -p "$HOME/.local/share/applications"
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

# 5. Configurar monitor de desbloqueo, reanudación y vigilancia de horario (Xfce Session Guard)
mkdir -p "$HOME/.local/bin" "$HOME/.config/systemd/user"
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

# Restablecer estado visual neutral en RandR
pkill -x redshift 2>/dev/null || true
redshift -m randr -x 2>/dev/null || true

echo "==> Configuración completada exitosamente."
