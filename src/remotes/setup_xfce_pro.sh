#!/usr/bin/env bash
set -euo pipefail

if [ "$EUID" -eq 0 ]; then
    echo "Ejecuta este script como usuario normal (solicitará sudo cuando sea necesario)."
    exit 1
fi

echo "==> Solicitando credenciales sudo..."
sudo -v
while true; do sudo -n true; sleep 60; kill -0 "$$" || exit; done 2>/dev/null &
SUDO_PID=$!
trap 'kill "$SUDO_PID" 2>/dev/null || true' EXIT

# Asegurar panel activo para evitar fallos de D-Bus / Xfconf
pgrep -x xfce4-panel >/dev/null || (xfce4-panel >/dev/null 2>&1 & sleep 1)

echo "==> 1. Purgando dependencias obsoletas (xfdesktop, rofi, plank, betterlockscreen, alttab)..."
sudo apt purge -y \
    xfdesktop4 plank rofi feh imagemagick xcape \
    remmina remmina-plugin-rdp remmina-plugin-vnc remmina-plugin-secret remmina-common \
    alttab \
    2>/dev/null || true
sudo apt autoremove -y 2>/dev/null || true

# Limpieza de binarios manuales anteriores y configs huérfanas
sudo rm -f /usr/local/bin/greenclip /usr/local/bin/i3lock-color /usr/local/bin/betterlockscreen /usr/local/bin/alttab
rm -f "$HOME/.local/bin/greenclip" "$HOME/.local/bin/i3lock-color" "$HOME/.local/bin/betterlockscreen" "$HOME/.local/bin/alttab" "$HOME/.local/bin/alttab-daemon.sh"
rm -rf "$HOME/.config/rofi" "$HOME/.config/betterlockscreen" "$HOME/.config/plank" "$HOME/.cache/greenclip.history"
rm -f "$HOME/.config/autostart/plank.desktop" "$HOME/.config/autostart/xcape.desktop" \
      "$HOME/.config/autostart/greenclip.desktop" "$HOME/.config/autostart/touchpad-setup.desktop" \
      "$HOME/.config/autostart/alttab.desktop"

echo "==> 2. Instalando stack base, Picom y dependencias de sistema..."
sudo apt update
sudo apt install -y \
    lightdm lightdm-gtk-greeter lightdm-gtk-greeter-settings light-locker \
    xfce4-goodies xfce4-whiskermenu-plugin xfce4-notifyd xfce4-power-manager xfce4-screenshooter \
    xdotool brightnessctl pavucontrol network-manager-gnome \
    pipewire pipewire-pulse wireplumber \
    xdg-desktop-portal xdg-desktop-portal-gtk \
    tumbler ffmpegthumbnailer poppler-data tumbler-plugins-extra webp-pixbuf-loader \
    gvfs-backends gvfs-fuse policykit-1-gnome \
    fonts-inter fonts-jetbrains-mono \
    dconf-cli libglib2.0-bin libglib2.0-dev-bin libnotify-bin \
    xwallpaper libxcb-xrm0 \
    picom libchipmunk7 libgif7 libpng16-16t64 libxcomposite1 libxdamage1 libxft2 libxinerama1 libjpeg62 \
    nemo nemo-fileroller \
    curl wget git jq unzip

echo "==> 3. Instalando Vicinae..."
if ! command -v vicinae >/dev/null 2>&1 || ! vicinae version >/dev/null 2>&1; then
    curl -fsSL https://vicinae.com/install | bash -s -- --prefix "$HOME/.local"
fi

# Corregir permisos en caso de instalaciones previas a nivel de sistema (/usr/local)
if [ -d "/usr/local/lib/vicinae" ]; then
    sudo chmod -R a+rX /usr/local/lib/vicinae 2>/dev/null || true
fi

# Asegurar symlink global en /usr/local/bin accesible por cualquier entorno y display manager
if [ -f "$HOME/.local/bin/vicinae" ]; then
    sudo ln -sf "$HOME/.local/bin/vicinae" /usr/local/bin/vicinae 2>/dev/null || true
elif [ -f "/usr/local/lib/vicinae/usr/bin/vicinae" ]; then
    sudo ln -sf /usr/local/lib/vicinae/usr/bin/vicinae /usr/local/bin/vicinae 2>/dev/null || true
fi

# Asegurar que el usuario pertenezca al grupo input para captura evdev de atajos y portapapeles
if ! id -nG "$USER" | grep -qw "input"; then
    sudo usermod -aG input "$USER" 2>/dev/null || true
fi

# Asegurar directorios de configuración bajo UID del usuario
sudo rm -rf /tmp/vicinae* /run/user/"$(id -u)"/vicinae* 2>/dev/null || true
mkdir -p "$HOME/.config/vicinae" "$HOME/.local/share/vicinae"
sudo chown -R "$USER:$USER" "$HOME/.config/vicinae" "$HOME/.local/share/vicinae"

# Habilitar e iniciar servicio systemd de usuario si está disponible
systemctl --user daemon-reload 2>/dev/null || true
systemctl --user enable vicinae.service 2>/dev/null || true
systemctl --user restart vicinae.service 2>/dev/null || true

mkdir -p "$HOME/.config/autostart"
cat <<EOF > "$HOME/.config/autostart/vicinae.desktop"
[Desktop Entry]
Type=Application
Exec=sh -c "systemctl --user is-active --quiet vicinae || vicinae server --replace"
Hidden=false
NoDisplay=false
X-GNOME-Autostart-enabled=true
Name=Vicinae Daemon
Comment=Vicinae Background Service
EOF

if ! pgrep -x vicinae >/dev/null; then
    (vicinae server --replace >/dev/null 2>&1 &) || true
fi

echo "==> 4. Configurando LightDM y Light-Locker..."
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

mkdir -p "$HOME/.local/bin"
cat <<'EOF' > "$HOME/.local/bin/screenlock"
#!/bin/bash
if command -v light-locker-command >/dev/null 2>&1; then
    light-locker-command -l
elif command -v dm-tool >/dev/null 2>&1; then
    dm-tool lock
else
    xflock4
fi
EOF
chmod +x "$HOME/.local/bin/screenlock"

echo "==> 5. Instalando temas Orchis-Dark y Tela-circle-dark..."
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

echo "==> 6. Configuración de hardware (Touchpad en Xorg, PipeWire, Polkit)..."
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

echo "==> 7. Configurando Picom como compositor único y Skippy-XD para selector Alt-Tab..."
# Asegurar que alttab no interfiera
killall -q alttab 2>/dev/null || true
rm -f "$HOME/.config/autostart/alttab.desktop" "$HOME/.local/bin/alttab-daemon.sh" 2>/dev/null || true

# Desactivar de forma total y definitiva el compositor nativo de xfwm4
xfconf-query -c xfwm4 -p /general/use_compositing -n -t bool -s false 2>/dev/null || \
xfconf-query -c xfwm4 -p /general/use_compositing -s false 2>/dev/null || true

xfconf-query -c xfwm4 -p /general/show_frame_shadow -s false 2>/dev/null || true
xfconf-query -c xfwm4 -p /general/show_popup_shadow -s false 2>/dev/null || true
xfconf-query -c xfwm4 -p /general/show_dock_shadow -s false 2>/dev/null || true
xfconf-query -c xfwm4 -p /general/raise_on_focus -s true 2>/dev/null || true

# Configuración de Picom (GLX backend, vsync, aceleración por hardware y bordes redondeados limpios)
mkdir -p "$HOME/.config/picom"
cat <<'EOF' > "$HOME/.config/picom/picom.conf"
backend = "glx";
vsync = true;
use-damage = true;
unredir-if-possible = false;

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
    "class_g = 'skippy-xd'",
    "fullscreen"
];

detect-rounded-corners = true;
detect-client-opacity = true;
detect-transient = true;
use-ewmh-active-win = true;

shadow-exclude = [
    "class_g = 'skippy-xd'",
    "class_g = 'vicinae'",
    "name = 'Notification'",
    "_NET_WM_STATE *= '_NET_WM_STATE_HIDDEN'"
];

focus-exclude = [
    "class_g = 'skippy-xd'"
];

wintypes:
{
  tooltip = { fade = false; shadow = false; opacity = 1.0; focus = true; full-shadow = false; };
  dock = { shadow = false; clip-shadow-above = true; };
  dnd = { shadow = false; };
  popup_menu = { opacity = 1.0; shadow = false; };
  dropdown_menu = { opacity = 1.0; shadow = false; };
};
EOF

# Servicio systemd de usuario para Picom (persistencia resiliente a reinicios)
mkdir -p "$HOME/.config/systemd/user"
cat <<'EOF' > "$HOME/.config/systemd/user/picom.service"
[Unit]
Description=Picom X11 Compositor
Documentation=man:picom(1)
After=graphical-session.target
PartOf=graphical-session.target

[Service]
Type=simple
ExecStart=/usr/bin/picom
Restart=always
RestartSec=3
Environment=DISPLAY=:0

[Install]
WantedBy=graphical-session.target default.target
EOF

mkdir -p "$HOME/.config/autostart"
cat <<'EOF' > "$HOME/.config/autostart/picom.desktop"
[Desktop Entry]
Type=Application
Exec=sh -c "systemctl --user is-active --quiet picom || picom -b"
Hidden=false
NoDisplay=false
X-GNOME-Autostart-enabled=true
Name=Picom
Comment=Lightweight X11 Compositor
EOF

systemctl --user daemon-reload 2>/dev/null || true
systemctl --user enable picom.service 2>/dev/null || true
systemctl --user restart picom.service 2>/dev/null || true

# Instalación de Skippy-XD mediante APT (con resolución de dependencias universales)
if ! command -v skippy-xd >/dev/null 2>&1 || ! dpkg -l skippy-xd 2>/dev/null | grep -q "^ii"; then
    echo "==> Instalando paquete Skippy-XD mediante APT..."
    TEMP_SKIPPY=$(mktemp -d)
    ARCH=$(dpkg --print-architecture 2>/dev/null || echo "amd64")
    SKIPPY_URL="https://github.com/felixfung/skippy-xd/releases/download/v2026.09.26/skippy-xd_2026.09.26-1_${ARCH}.deb"
    if curl -fsSL "$SKIPPY_URL" -o "$TEMP_SKIPPY/skippy-xd.deb"; then
        dpkg-deb -R "$TEMP_SKIPPY/skippy-xd.deb" "$TEMP_SKIPPY/pkg"
        # Ajustar control para compatibilidad universal con Debian y Ubuntu
        sed -i 's/libjpeg62-turbo/libjpeg62 | libjpeg62-turbo | libjpeg-turbo8/g' "$TEMP_SKIPPY/pkg/DEBIAN/control"
        dpkg-deb -b "$TEMP_SKIPPY/pkg" "$TEMP_SKIPPY/skippy-xd-compatible.deb" >/dev/null 2>&1
        sudo apt install -y libjpeg62 2>/dev/null || true
        sudo apt install -y "$TEMP_SKIPPY/skippy-xd-compatible.deb" 2>/dev/null || \
            (sudo dpkg -i "$TEMP_SKIPPY/skippy-xd-compatible.deb" 2>/dev/null && sudo apt install -f -y 2>/dev/null) || true
    fi
    rm -rf "$TEMP_SKIPPY"
fi

# Configuración de Skippy-XD (Exposé / Alt-Tab switcher con miniaturas en vivo)
mkdir -p "$HOME/.config/skippy-xd"
cat <<'EOF' > "$HOME/.config/skippy-xd/skippy-xd.rc"
# Skippy-XD Configuration for Xfce + Picom
[system]
daemonPath = /tmp/skippy-xd-fifo
clientPath = /tmp/skippy-xd-fofi
clientList = _NET_CLIENT_LIST
pseudoTrans = false

[multimonitor]
showOnlyCurrentMonitor = false
showOnlyCurrentScreen = true
horizontalPanelAlignment = mid
verticalPanelAlignment = mid

[layout]
switchLayout = compactrect
exposeLayout = cosmos
switchWaitDuration = 50
switchCycleDuringWait = false
switchCycleDesktops = false
exposeCycleDesktops = false
distance = 32
upscaleWindows = false

[appearance]
animationDuration = 120
animationRefresh = 60
background = #1e1e2eb0
preservePages = true
includeFrame = true
leftFrameBorder = 0
topFrameBorder = 0
cornerRadius = 10

[filler]
opacity = 200
color = #1e1e2e
iconPlace = top left
iconSize = 48

[livepreview]
opacity = 255
icon = true
iconPlace = top left
iconSize = 48

[highlight]
tint = #89b4fa
tintOpacity = 96
tintWindow = true
tintBorder = 4

[multiselect]
tint = #a6e3a1
tintOpacity = 160

[panel]
show = true
backgroundTinting = true
reserveSpace = true

[desktop]
show = false
backgroundTinting = false

[label]
show = true
option = windowTitle
offsetX = 0
offsetY = -8
width = 0.85
border = #313244
background = #181825
backgroundHighlight = #89b4fa
opacity = 230
text = #cdd6f4
textOutline = #11111b
font = Inter 10:weight=bold

[bindings]
enforceFocus = true
pivotLockingTime = 0
moveMouse = false

keysUp = Up
keysDown = Down
keysLeft = Left
keysRight = Right

keysSelect = Return space
keysCancel = Escape
keysNext = Tab n
keysPrev = ISO_Left_Tab p

keysIconify = 1
keysShade = 2
keysClose = 3

miwMouse1 = focus
miwMouse2 = close-ewmh
miwMouse3 = iconify
miwMouse4 = keysNext
miwMouse5 = keysPrev
EOF

# Wrapper universal con normalización de DISPLAY y symlinks de compatibilidad FIFO
mkdir -p "$HOME/.local/bin"
cat <<'EOF' > "$HOME/.local/bin/skippy-xd"
#!/bin/sh
export LD_LIBRARY_PATH="$HOME/.local/lib/skippy-xd:${LD_LIBRARY_PATH:-}"

# Normalizar DISPLAY para garantizar correspondencia entre daemon y cliente (:0.0 <-> :0)
if [ -n "$DISPLAY" ]; then
    export DISPLAY="${DISPLAY%.0}"
fi

# Asegurar symlinks bidireccionales en el pipe FIFO de /tmp
DISP_BASE="${DISPLAY:-:0}"
DISP_BASE="${DISP_BASE%.0}"
if [ -e "/tmp/skippy-xd-fifo${DISP_BASE}" ] && [ ! -e "/tmp/skippy-xd-fifo${DISP_BASE}.0" ]; then
    ln -sf "/tmp/skippy-xd-fifo${DISP_BASE}" "/tmp/skippy-xd-fifo${DISP_BASE}.0" 2>/dev/null || true
elif [ -e "/tmp/skippy-xd-fifo${DISP_BASE}.0" ] && [ ! -e "/tmp/skippy-xd-fifo${DISP_BASE}" ]; then
    ln -sf "/tmp/skippy-xd-fifo${DISP_BASE}.0" "/tmp/skippy-xd-fifo${DISP_BASE}" 2>/dev/null || true
fi

if [ -x "$HOME/.local/bin/skippy-xd.bin" ]; then
    exec "$HOME/.local/bin/skippy-xd.bin" "$@"
elif [ -x "/usr/bin/skippy-xd" ]; then
    exec /usr/bin/skippy-xd "$@"
else
    exec skippy-xd "$@"
fi
EOF
chmod +x "$HOME/.local/bin/skippy-xd"
sudo ln -sf "$HOME/.local/bin/skippy-xd" /usr/local/bin/skippy-xd 2>/dev/null || true

cat <<'EOF' > "$HOME/.config/systemd/user/skippy-xd.service"
[Unit]
Description=Skippy-XD Window Switcher Daemon
Documentation=man:skippy-xd(1)
After=graphical-session.target picom.service
PartOf=graphical-session.target

[Service]
Type=simple
Environment="PATH=%h/.local/bin:/usr/local/bin:/usr/bin:/bin"
ExecStart=skippy-xd --start-daemon
Restart=always
RestartSec=3
Environment=DISPLAY=:0

[Install]
WantedBy=graphical-session.target default.target
EOF

cat <<'EOF' > "$HOME/.config/autostart/skippy-xd.desktop"
[Desktop Entry]
Type=Application
Exec=sh -c "systemctl --user is-active --quiet skippy-xd || skippy-xd --start-daemon"
Hidden=false
NoDisplay=false
X-GNOME-Autostart-enabled=true
Name=Skippy-XD Daemon
Comment=Window Switcher with Live Thumbnails
EOF

systemctl --user daemon-reload 2>/dev/null || true
systemctl --user enable skippy-xd.service 2>/dev/null || true
systemctl --user restart skippy-xd.service 2>/dev/null || true

# Desvincular switcher nativo de xfwm4 para ceder el control completo a Skippy-XD
xfconf-query -c xfce4-keyboard-shortcuts -p "/xfwm4/custom/<Alt>Tab" -n -t string -s "none" 2>/dev/null || \
xfconf-query -c xfce4-keyboard-shortcuts -p "/xfwm4/custom/<Alt>Tab" -s "none" 2>/dev/null || true

xfconf-query -c xfce4-keyboard-shortcuts -p "/xfwm4/custom/<Alt><Shift>Tab" -n -t string -s "none" 2>/dev/null || \
xfconf-query -c xfce4-keyboard-shortcuts -p "/xfwm4/custom/<Alt><Shift>Tab" -s "none" 2>/dev/null || true

xfconf-query -c xfce4-keyboard-shortcuts -p "/xfwm4/custom/<Super>Tab" -n -t string -s "none" 2>/dev/null || \
xfconf-query -c xfce4-keyboard-shortcuts -p "/xfwm4/custom/<Super>Tab" -s "none" 2>/dev/null || true

# Configurar Skippy-XD en atajos de teclado globales
xfconf-query -c xfce4-keyboard-shortcuts -p "/commands/custom/<Alt>Tab" -n -t string -s "skippy-xd --switch --next" 2>/dev/null || \
xfconf-query -c xfce4-keyboard-shortcuts -p "/commands/custom/<Alt>Tab" -s "skippy-xd --switch --next" 2>/dev/null || true

xfconf-query -c xfce4-keyboard-shortcuts -p "/commands/custom/<Alt><Shift>Tab" -n -t string -s "skippy-xd --switch --prev" 2>/dev/null || \
xfconf-query -c xfce4-keyboard-shortcuts -p "/commands/custom/<Alt><Shift>Tab" -s "skippy-xd --switch --prev" 2>/dev/null || true

xfconf-query -c xfce4-keyboard-shortcuts -p "/commands/custom/<Super>Tab" -n -t string -s "skippy-xd --expose" 2>/dev/null || \
xfconf-query -c xfce4-keyboard-shortcuts -p "/commands/custom/<Super>Tab" -s "skippy-xd --expose" 2>/dev/null || true

# Iniciar o reiniciar daemons en sesión activa
killall -q picom skippy-xd skippy-xd.bin 2>/dev/null || true
xfwm4 --replace >/dev/null 2>&1 &
sleep 1
(picom -b --config "$HOME/.config/picom/picom.conf" >/dev/null 2>&1 &) || true
(nohup skippy-xd --start-daemon >/dev/null 2>&1 &) || true

echo "==> 8. Configurando xwallpaper y desacoplando xfdesktop de la sesión..."
cat <<'EOF' > "$HOME/.local/bin/wallpaper.sh"
#!/bin/bash
WALLPAPER=""
for w in \
    "/usr/share/xfce4/backdrops/xubuntu-wallpaper.png" \
    "/usr/share/xfce4/backdrops/xubuntu-plucky.png" \
    $(ls -1 /usr/share/xfce4/backdrops/*.png 2>/dev/null) \
    $(ls -1 /usr/share/backgrounds/*.{png,jpg} 2>/dev/null) \
    "/usr/share/images/desktop-base/default"; do
    if [ -f "$w" ]; then
        WALLPAPER="$w"
        break
    fi
done

if command -v xwallpaper >/dev/null 2>&1 && [ -n "$WALLPAPER" ]; then
    xwallpaper --zoom "$WALLPAPER"
fi
EOF
chmod +x "$HOME/.local/bin/wallpaper.sh"

cat <<EOF > "$HOME/.config/autostart/wallpaper.desktop"
[Desktop Entry]
Type=Application
Exec=$HOME/.local/bin/wallpaper.sh
Hidden=false
NoDisplay=false
X-GNOME-Autostart-enabled=true
Name=Wallpaper Setter
Comment=Lightweight desktop wallpaper loader via xwallpaper
EOF

# Desacoplar xfdesktop de las aplicaciones de arranque de xfce4-session
xfconf-query -c xfce4-session -p /sessions/Failsafe/Client4_Command -s "/bin/true" -t string 2>/dev/null || true
xfconf-query -c xfce4-session -p /sessions/Failsafe/Count -s 4 2>/dev/null || true

echo "==> 9. Configurando atajos de teclado globales y comportamiento de ventanas..."
xfconf-query -c xsettings -p /Net/ThemeName -s "Orchis-Dark" 2>/dev/null || true
xfconf-query -c xsettings -p /Net/IconThemeName -s "Tela-circle-dark" 2>/dev/null || true
xfconf-query -c xsettings -p /Gtk/FontName -s "Inter 10" 2>/dev/null || true
xfconf-query -c xsettings -p /Gtk/MonospaceFontName -s "JetBrains Mono 10" 2>/dev/null || true

xfconf-query -c xfwm4 -p /general/theme -s "Orchis-Dark" 2>/dev/null || true
xfconf-query -c xfwm4 -p /general/title_font -s "Inter Bold 10" 2>/dev/null || true
xfconf-query -c xfwm4 -p /general/button_layout -s "CHM|T" 2>/dev/null || true
xfconf-query -c xfwm4 -p /general/borderless_maximize -s true 2>/dev/null || true

# Tecla Windows / Super abre Whisker Menu
xfconf-query -c xfce4-keyboard-shortcuts -p "/commands/custom/Super_L" -n -t string -s "xfce4-popup-whiskermenu" 2>/dev/null || \
xfconf-query -c xfce4-keyboard-shortcuts -p "/commands/custom/Super_L" -s "xfce4-popup-whiskermenu" 2>/dev/null || true

xfconf-query -c xfce4-keyboard-shortcuts -p "/commands/custom/Super_R" -n -t string -s "xfce4-popup-whiskermenu" 2>/dev/null || \
xfconf-query -c xfce4-keyboard-shortcuts -p "/commands/custom/Super_R" -s "xfce4-popup-whiskermenu" 2>/dev/null || true

xfconf-query -c xfce4-keyboard-shortcuts -p "/commands/custom/<Alt>F1" -n -t string -s "xfce4-popup-whiskermenu" 2>/dev/null || \
xfconf-query -c xfce4-keyboard-shortcuts -p "/commands/custom/<Alt>F1" -s "xfce4-popup-whiskermenu" 2>/dev/null || true

# Atajo Spotlight macOS en teclado Windows: Super + Espacio -> Vicinae
xfconf-query -c xfce4-keyboard-shortcuts -p "/commands/custom/<Super>space" -n -t string -s "vicinae toggle" 2>/dev/null || \
xfconf-query -c xfce4-keyboard-shortcuts -p "/commands/custom/<Super>space" -s "vicinae toggle" 2>/dev/null || true

# Portapapeles con anclado / favoritos: Super + V
xfconf-query -c xfce4-keyboard-shortcuts -p "/commands/custom/<Super>v" -n -t string -s "vicinae cmd launch clipboard:history" 2>/dev/null || \
xfconf-query -c xfce4-keyboard-shortcuts -p "/commands/custom/<Super>v" -s "vicinae cmd launch clipboard:history" 2>/dev/null || true

# Selector de Emojis: Super + .
xfconf-query -c xfce4-keyboard-shortcuts -p "/commands/custom/<Super>period" -n -t string -s "vicinae cmd launch core:search-emojis" 2>/dev/null || \
xfconf-query -c xfce4-keyboard-shortcuts -p "/commands/custom/<Super>period" -s "vicinae cmd launch core:search-emojis" 2>/dev/null || true

# Bloqueo de pantalla nativo
xfconf-query -c xfce4-keyboard-shortcuts -p "/commands/custom/<Super>l" -n -t string -s "$HOME/.local/bin/screenlock" 2>/dev/null || \
xfconf-query -c xfce4-keyboard-shortcuts -p "/commands/custom/<Super>l" -s "$HOME/.local/bin/screenlock" 2>/dev/null || true

xfconf-query -c xfce4-keyboard-shortcuts -p "/commands/custom/<Primary><Alt>l" -n -t string -s "$HOME/.local/bin/screenlock" 2>/dev/null || \
xfconf-query -c xfce4-keyboard-shortcuts -p "/commands/custom/<Primary><Alt>l" -s "$HOME/.local/bin/screenlock" 2>/dev/null || true

# Alternancia de teclado en Super + Alt + Espacio
cat <<'EOF' > "$HOME/.local/bin/toggle-layout.sh"
#!/bin/bash
CURRENT=$(setxkbmap -query 2>/dev/null | awk '/layout:/ {print $2}' | cut -d',' -f1)
if [ "$CURRENT" = "us" ]; then
    setxkbmap -layout es
    notify-send -t 1200 -i input-keyboard -h string:synchronous:keyboard-layout "Distribución de Teclado" "Español (ES)"
else
    setxkbmap -layout us
    notify-send -t 1200 -i input-keyboard -h string:synchronous:keyboard-layout "Distribución de Teclado" "Inglés (US)"
fi
EOF
chmod +x "$HOME/.local/bin/toggle-layout.sh"

xfconf-query -c xfce4-keyboard-shortcuts -p "/commands/custom/<Super><Alt>space" -n -t string -s "$HOME/.local/bin/toggle-layout.sh" 2>/dev/null || \
xfconf-query -c xfce4-keyboard-shortcuts -p "/commands/custom/<Super><Alt>space" -s "$HOME/.local/bin/toggle-layout.sh" 2>/dev/null || true

echo "==> 10. Integrando Nemo como gestor de archivos predeterminado..."
# 1. Asociar tipos MIME de carpetas y búsquedas en el sistema
xdg-mime default nemo.desktop inode/directory
xdg-mime default nemo.desktop application/x-gnome-saved-search
gio mime inode/directory nemo.desktop 2>/dev/null || true
gio mime application/x-gnome-saved-search nemo.desktop 2>/dev/null || true

# 2. Configurar manejador en ~/.config/mimeapps.list
mkdir -p "$HOME/.config"
touch "$HOME/.config/mimeapps.list"
if ! grep -q "^inode/directory=nemo.desktop" "$HOME/.config/mimeapps.list"; then
    if grep -q "\[Default Applications\]" "$HOME/.config/mimeapps.list"; then
        sed -i '/\[Default Applications\]/a inode/directory=nemo.desktop\napplication/x-gnome-saved-search=nemo.desktop' "$HOME/.config/mimeapps.list"
    else
        echo -e "[Default Applications]\ninode/directory=nemo.desktop\napplication/x-gnome-saved-search=nemo.desktop" >> "$HOME/.config/mimeapps.list"
    fi
fi

# 3. Integrar Nemo como Preferred Application en el subsistema Xfce (exo-open)
mkdir -p "$HOME/.local/share/xfce4/helpers"
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

mkdir -p "$HOME/.config/xfce4"
touch "$HOME/.config/xfce4/helpers.rc"
if grep -q "^FileManager=" "$HOME/.config/xfce4/helpers.rc"; then
    sed -i 's/^FileManager=.*/FileManager=nemo/' "$HOME/.config/xfce4/helpers.rc"
else
    echo "FileManager=nemo" >> "$HOME/.config/xfce4/helpers.rc"
fi

# 4. Desactivar gestión de escritorio en Nemo (evita interferencias con xwallpaper)
gsettings set org.nemo.desktop show-desktop-icons false 2>/dev/null || true
gsettings set org.nemo.preferences show-image-thumbnails 'always' 2>/dev/null || true

echo "==> 11. Configurando Panel Xfce y Notificaciones..."
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
EOF

xfconf-query -c xfce4-notifyd -p /theme -s "Orchis-Dark" 2>/dev/null || true
xfconf-query -c xfce4-notifyd -p /notify-location -s "top-right" 2>/dev/null || true
xfconf-query -c xfce4-notifyd -p /notification-log -s true 2>/dev/null || true

xfconf-query -c xfce4-panel -p /plugins/plugin-1 -s "whiskermenu" 2>/dev/null || true
xfconf-query -c xfce4-panel -p /plugins/plugin-1/button-icon -s "view-app-grid-symbolic" 2>/dev/null || true
xfconf-query -c xfce4-panel -p /plugins/plugin-1/show-button-icon -s true 2>/dev/null || true
xfconf-query -c xfce4-panel -p /plugins/plugin-1/show-button-title -s false 2>/dev/null || true

CLOCK_PLUGIN=$(xfconf-query -c xfce4-panel -p /plugins -l 2>/dev/null | grep -E '^/plugins/plugin-[0-9]+$' | while read -r p; do
    [ "$(xfconf-query -c xfce4-panel -p "$p" 2>/dev/null || true)" = "clock" ] && echo "$p" && break
done)

if [ -n "$CLOCK_PLUGIN" ]; then
    xfconf-query -c xfce4-panel -p "$CLOCK_PLUGIN/mode" -s 2 2>/dev/null || true
    xfconf-query -c xfce4-panel -p "$CLOCK_PLUGIN/digital-layout" -s 3 2>/dev/null || true
    xfconf-query -c xfce4-panel -p "$CLOCK_PLUGIN/digital-time-format" -s "%I:%M:%S %p" 2>/dev/null || true
    xfconf-query -c xfce4-panel -p "$CLOCK_PLUGIN/show-seconds" -s true 2>/dev/null || true
fi

xfconf-query -c xfce4-panel -p /panels -a -t int -s 1 2>/dev/null || true
xfconf-query -c xfce4-panel -p /panels/panel-2 -r -R 2>/dev/null || true
xfce4-panel -r 2>/dev/null || true

echo "==> 12. Optimizaciones genéricas de red en arranque..."
sudo systemctl disable NetworkManager-wait-online.service 2>/dev/null || true

echo "==> Configuración completada. Reinicia el entorno para aplicar los cambios de sesión con: sudo reboot"
