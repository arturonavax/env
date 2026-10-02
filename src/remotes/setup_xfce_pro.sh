#!/usr/bin/env bash
set -euo pipefail

if [ "$EUID" -eq 0 ]; then
    echo "Ejecuta este script como usuario normal (solicitará sudo cuando sea necesario)."
    exit 1
fi

echo "==> Solicitando credenciales sudo..."
sudo -v
# Mantener vivo el token de sudo en segundo plano durante la ejecución
while true; do sudo -n true; sleep 60; kill -0 "$$" || exit; done 2>/dev/null &
SUDO_PID=$!
trap 'kill "$SUDO_PID" 2>/dev/null || true' EXIT

# 0. Asegurar que el panel esté levantado para evitar fallos de D-Bus / Xfconf
pgrep -x xfce4-panel >/dev/null || (xfce4-panel >/dev/null 2>&1 & sleep 1)

echo "==> 1. Instalando dependencias de Xfce Pro, herramientas y compatibilidad..."
sudo apt update
sudo apt install -y \
    lightdm lightdm-gtk-greeter lightdm-gtk-greeter-settings \
    xfce4-goodies xfce4-whiskermenu-plugin xfce4-notifyd xfce4-power-manager xfce4-screenshooter \
    xcape xdotool brightnessctl pavucontrol network-manager-gnome \
    pipewire pipewire-pulse wireplumber \
    xdg-desktop-portal xdg-desktop-portal-gtk \
    tumbler ffmpegthumbnailer poppler-data tumbler-plugins-extra webp-pixbuf-loader \
    gvfs-backends gvfs-fuse policykit-1-gnome \
    fonts-inter fonts-jetbrains-mono \
    plank dconf-cli libglib2.0-bin libglib2.0-dev-bin libnotify-bin \
    picom rofi feh imagemagick bc libxcb-xrm0 \
    curl wget git jq unzip

echo "==> Instalando utilidades Pro (Greenclip, i3lock-color, Betterlockscreen)..."
# 1. Greenclip (portapapeles searchable)
if [ ! -f "/usr/local/bin/greenclip" ]; then
    echo "    - Descargando Greenclip (daemon de portapapeles)..."
    sudo curl -fsSL "https://github.com/erebe/greenclip/releases/download/v4.2/greenclip" -o /usr/local/bin/greenclip
    sudo chmod +x /usr/local/bin/greenclip
fi
mkdir -p "$HOME/.local/bin"
ln -sf /usr/local/bin/greenclip "$HOME/.local/bin/greenclip" 2>/dev/null || true

# 2. i3lock-color (binario precompilado oficial de Raymo111)
if [ ! -f "/usr/local/bin/i3lock-color" ]; then
    echo "    - Descargando i3lock-color..."
    sudo curl -fsSL "https://github.com/Raymo111/i3lock-color/releases/download/2.13.c.5/i3lock" -o /usr/local/bin/i3lock-color
    sudo chmod +x /usr/local/bin/i3lock-color
    sudo ln -sf /usr/local/bin/i3lock-color /usr/local/bin/i3lock 2>/dev/null || true
fi
ln -sf /usr/local/bin/i3lock-color "$HOME/.local/bin/i3lock-color" 2>/dev/null || true

# 3. Betterlockscreen
if [ ! -f "/usr/local/bin/betterlockscreen" ]; then
    echo "    - Descargando Betterlockscreen..."
    sudo curl -fsSL "https://raw.githubusercontent.com/betterlockscreen/betterlockscreen/main/betterlockscreen" -o /usr/local/bin/betterlockscreen
    sudo chmod +x /usr/local/bin/betterlockscreen
fi
ln -sf /usr/local/bin/betterlockscreen "$HOME/.local/bin/betterlockscreen" 2>/dev/null || true

echo "==> Limpiando paquetes y herramientas obsoletas..."
sudo apt purge -y \
    remmina remmina-plugin-rdp remmina-plugin-vnc remmina-plugin-secret remmina-common \
    2>/dev/null || true
sudo apt autoremove -y 2>/dev/null || true
rm -f ~/.config/autostart/remmina*.desktop 2>/dev/null || true
rm -f ~/.config/plank/dock1/launchers/remmina*.dockitem 2>/dev/null || true

echo "==> 2. Definiendo e integrando LightDM como gestor de sesión inicial..."
# Configuración debconf
echo "lightdm shared/default-x-display-manager select lightdm" | sudo debconf-set-selections
echo "gdm3 shared/default-x-display-manager select lightdm" | sudo debconf-set-selections 2>/dev/null || true
sudo dpkg-reconfigure -f noninteractive lightdm 2>/dev/null || true

# Forzar archivo /etc/X11/default-display-manager
echo "/usr/sbin/lightdm" | sudo tee /etc/X11/default-display-manager >/dev/null

# Desactivar GDM y activar LightDM en systemd
sudo systemctl disable gdm3 gdm 2>/dev/null || true
sudo systemctl enable lightdm.service --force 2>/dev/null || true
sudo ln -sf /lib/systemd/system/lightdm.service /etc/systemd/system/display-manager.service

# Asegurar sesión Xfce por defecto en LightDM
sudo mkdir -p /etc/lightdm/lightdm.conf.d
sudo bash -c "cat <<'EOF' > /etc/lightdm/lightdm.conf.d/50-xfce-desktop.conf
[Seat:*]
user-session=xfce
greeter-session=lightdm-gtk-greeter
greeter-hide-users=false
EOF"

# Configurar LightDM GTK Greeter con hora AM/PM y soporte de fondo de usuario
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

sudo rm -f /usr/share/backgrounds/modern/minimal.png 2>/dev/null || true

echo "==> 3. Verificando e instalando temas globales en /usr/share..."
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
else
    echo "    Temas Orchis-Dark y Tela-circle-dark ya instalados."
fi

echo "==> 4. Configurando compatibilidad (Touchpad, Compositor xfwm4 GLX, PipeWire, Polkit, Xcape)..."
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

# Aplicar método de click buttonareas (botón derecho físico) y persistir para el usuario
mkdir -p "$HOME/.local/bin"
cat <<'EOF' > "$HOME/.local/bin/touchpad-setup.sh"
#!/bin/bash
for id in $(xinput list --id-only 2>/dev/null); do
    if xinput list-props "$id" 2>/dev/null | grep -q "libinput Click Method Enabled"; then
        xinput set-prop "$id" "libinput Click Method Enabled" 1 0 2>/dev/null || true
    fi
done
EOF
chmod +x "$HOME/.local/bin/touchpad-setup.sh"
"$HOME/.local/bin/touchpad-setup.sh" 2>/dev/null || true

mkdir -p "$HOME/.config/autostart"
cat <<EOF > "$HOME/.config/autostart/touchpad-setup.desktop"
[Desktop Entry]
Type=Application
Exec=$HOME/.local/bin/touchpad-setup.sh
Hidden=false
NoDisplay=false
X-GNOME-Autostart-enabled=true
Name=Touchpad Setup
Comment=Enable right-click button area on touchpad
EOF

systemctl --user enable --now pipewire pipewire-pulse wireplumber 2>/dev/null || true

# Configuración del compositor acelerado GLX de xfwm4 para soporte nativo de miniaturas en Alt-Tab sin tearing
xfconf-query -c xfwm4 -p /general/use_compositing -n -t bool -s true 2>/dev/null || \
xfconf-query -c xfwm4 -p /general/use_compositing -s true 2>/dev/null || true

xfconf-query -c xfwm4 -p /general/vblank_mode -n -t string -s "glx" 2>/dev/null || \
xfconf-query -c xfwm4 -p /general/vblank_mode -s "glx" 2>/dev/null || true

# Configuración de previsualización en vivo en Alt-Tab (Grid Thumbnails)
xfconf-query -c xfwm4 -p /general/cycle_tabwin_mode -n -t int -s 1 2>/dev/null || \
xfconf-query -c xfwm4 -p /general/cycle_tabwin_mode -s 1 2>/dev/null || true

xfconf-query -c xfwm4 -p /general/cycle_preview -n -t bool -s true 2>/dev/null || \
xfconf-query -c xfwm4 -p /general/cycle_preview -s true 2>/dev/null || true

xfconf-query -c xfwm4 -p /general/cycle_draw_frame -n -t bool -s true 2>/dev/null || \
xfconf-query -c xfwm4 -p /general/cycle_draw_frame -s true 2>/dev/null || true

xfconf-query -c xfwm4 -p /general/cycle_minimum -n -t bool -s false 2>/dev/null || \
xfconf-query -c xfwm4 -p /general/cycle_minimum -s false 2>/dev/null || true

# Desactivar autostart de Picom para que no colisione con el compositor GLX de xfwm4
rm -f "$HOME/.config/autostart/picom.desktop" 2>/dev/null || true
killall picom 2>/dev/null || true

# Autostart: Plank
cat <<'EOF' > "$HOME/.config/autostart/plank.desktop"
[Desktop Entry]
Type=Application
Exec=plank
Hidden=false
NoDisplay=false
X-GNOME-Autostart-enabled=true
Name=Plank
EOF

# Autostart: Polkit GNOME Agent
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

# Autostart: Xcape (mapea tecla Windows / Super a Rofi vía Alt+F1 sin colisiones)
cat <<'EOF' > "$HOME/.config/autostart/xcape.desktop"
[Desktop Entry]
Type=Application
Exec=sh -c "killall xcape 2>/dev/null; sleep 1; xcape -e 'Super_L=Alt_L|F1;Super_R=Alt_L|F1'"
Hidden=false
NoDisplay=false
X-GNOME-Autostart-enabled=true
Name=Xcape Super Key Mapper
Comment=Mapea la tecla Windows / Super al menu Rofi via Alt+F1
EOF

killall xcape 2>/dev/null || true
(xcape -e 'Super_L=Alt_L|F1;Super_R=Alt_L|F1' >/dev/null 2>&1 &) || true

echo "==> Configurando Greenclip y Rofi (Clipboard searchable + App Launcher)..."
cat <<EOF > "$HOME/.config/greenclip.toml"
[greenclip]
  blacklisted_applications = []
  enable_image_support = true
  history_file = "$HOME/.cache/greenclip.history"
  image_cache_directory = "/tmp/greenclip"
  max_history_length = 50
  max_selection_size_bytes = 0
  static_history = []
  trim_space_from_selection = true
  use_primary_selection_as_input = false
EOF

cat <<EOF > "$HOME/.config/autostart/greenclip.desktop"
[Desktop Entry]
Type=Application
Exec=$HOME/.local/bin/greenclip daemon
Hidden=false
NoDisplay=false
X-GNOME-Autostart-enabled=true
Name=Greenclip Daemon
Comment=Clipboard manager daemon
EOF

killall greenclip 2>/dev/null || true
(greenclip daemon >/dev/null 2>&1 &) || true

mkdir -p "$HOME/.config/rofi"
cat <<'EOF' > "$HOME/.config/rofi/config.rasi"
configuration {
    modi: "drun,run,window,clipboard:greenclip print";
    font: "Inter 10";
    show-icons: true;
    icon-theme: "Tela-circle-dark";
    terminal: "ghostty";
    drun-display-format: "{name}";
    disable-history: false;
    hide-scrollbar: true;
    display-drun: " 󰀻  Apps ";
    display-run: " 󰌆  Run ";
    display-window: " 󰕰  Window ";
    display-clipboard: " 󱉥  Clipboard ";
    kb-cancel: "Escape,Control+g,Control+bracketleft,Alt+F1,Super_L,Super_R";
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
    width: 600px;
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
    placeholder: "Buscar...";
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
    spacing: 8px;
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

cat <<'EOF' > "$HOME/.local/bin/rofi-clipboard"
#!/bin/bash
TIMESTAMP_FILE="/tmp/rofi_toggle_time"
NOW=$(date +%s%3N)

if pgrep -x rofi >/dev/null 2>&1; then
    killall -q rofi 2>/dev/null || true
    echo "$NOW" > "$TIMESTAMP_FILE"
    exit 0
fi

if [ -f "$TIMESTAMP_FILE" ]; then
    LAST_TIME=$(cat "$TIMESTAMP_FILE" 2>/dev/null || echo 0)
    DIFF=$(( NOW - LAST_TIME ))
    if [ "$DIFF" -ge 0 ] && [ "$DIFF" -lt 450 ]; then
        exit 0
    fi
fi

rofi -modi "clipboard:greenclip print" -show clipboard -run-command '{cmd}'
echo "$(date +%s%3N)" > "$TIMESTAMP_FILE"
EOF
chmod +x "$HOME/.local/bin/rofi-clipboard"

cat <<'EOF' > "$HOME/.local/bin/rofi-launcher"
#!/bin/bash
TIMESTAMP_FILE="/tmp/rofi_toggle_time"
NOW=$(date +%s%3N)

if pgrep -x rofi >/dev/null 2>&1; then
    killall -q rofi 2>/dev/null || true
    echo "$NOW" > "$TIMESTAMP_FILE"
    exit 0
fi

if [ -f "$TIMESTAMP_FILE" ]; then
    LAST_TIME=$(cat "$TIMESTAMP_FILE" 2>/dev/null || echo 0)
    DIFF=$(( NOW - LAST_TIME ))
    if [ "$DIFF" -ge 0 ] && [ "$DIFF" -lt 450 ]; then
        exit 0
    fi
fi

rofi -show drun -show-icons
echo "$(date +%s%3N)" > "$TIMESTAMP_FILE"
EOF
chmod +x "$HOME/.local/bin/rofi-launcher"

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

echo "==> Configurando xfce4-notifyd (con historial, barra visual y tema Orchis-Dark)..."
# Desactivar Dunst si existiese
killall dunst 2>/dev/null || true
rm -f "$HOME/.config/autostart/dunst.desktop" 2>/dev/null || true
rm -f "$HOME/.local/share/dbus-1/services/org.freedesktop.Notifications.service" 2>/dev/null || true

# Tema visual moderno para xfce4-notifyd acorde a Orchis-Dark
mkdir -p "$HOME/.themes/Orchis-Dark/xfce-notify-4.0"
cat <<'EOF' > "$HOME/.themes/Orchis-Dark/xfce-notify-4.0/gtk.css"
#XfceNotifyWindow {
    background-color: #1e1e2e;
    color: #cdd6f4;
    border: 2px solid #45475a;
    border-radius: 10px;
    padding: 12px;
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
    background-color: #89b4fa;
    color: #11111b;
}

#XfceNotifyWindow label#summary {
    font-weight: bold;
    color: #89b4fa;
    font-size: 11pt;
}

#XfceNotifyWindow label#body {
    color: #cdd6f4;
}

#XfceNotifyWindow progressbar {
    min-height: 8px;
    border-radius: 4px;
}

#XfceNotifyWindow progressbar progress {
    background-image: none;
    background-color: #89b4fa;
    border: none;
    border-radius: 4px;
}

#XfceNotifyWindow progressbar trough {
    background-color: #313244;
    border: 1px solid #45475a;
    border-radius: 4px;
}
EOF

# Asegurar servicio systemd activo para xfce4-notifyd
systemctl --user unmask xfce4-notifyd.service 2>/dev/null || true
systemctl --user daemon-reload 2>/dev/null || true
systemctl --user start xfce4-notifyd.service 2>/dev/null || true

# Configuración de propiedades en xfconf
xfconf-query -c xfce4-notifyd -p /theme -n -t string -s "Orchis-Dark" 2>/dev/null || \
xfconf-query -c xfce4-notifyd -p /theme -s "Orchis-Dark" 2>/dev/null || true

xfconf-query -c xfce4-notifyd -p /notify-location -n -t string -s "top-right" 2>/dev/null || \
xfconf-query -c xfce4-notifyd -p /notify-location -s "top-right" 2>/dev/null || true

xfconf-query -c xfce4-notifyd -p /initial-opacity -n -t double -s 0.95 2>/dev/null || \
xfconf-query -c xfce4-notifyd -p /initial-opacity -s 0.95 2>/dev/null || true

xfconf-query -c xfce4-notifyd -p /do-fadeout -n -t bool -s true 2>/dev/null || \
xfconf-query -c xfce4-notifyd -p /do-fadeout -s true 2>/dev/null || true

xfconf-query -c xfce4-notifyd -p /notification-log -n -t bool -s true 2>/dev/null || \
xfconf-query -c xfce4-notifyd -p /notification-log -s true 2>/dev/null || true

xfconf-query -c xfce4-notifyd -p /log-max-size-enabled -n -t bool -s true 2>/dev/null || \
xfconf-query -c xfce4-notifyd -p /log-max-size-enabled -s true 2>/dev/null || true

xfconf-query -c xfce4-notifyd -p /log-max-size -n -t int -s 100 2>/dev/null || \
xfconf-query -c xfce4-notifyd -p /log-max-size -s 100 2>/dev/null || true

echo "==> Configurando Betterlockscreen..."
mkdir -p "$HOME/.config/betterlockscreen"
cat <<'EOF' > "$HOME/.config/betterlockscreen/betterlockscreenrc"
# ==============================================================================
# Betterlockscreen Configuration
# ==============================================================================
display_on=0
span_image=false
lock_timeout=300
fx_list=(dim blur dimblur pixel dimpixel color)
dim_level=40
blur_level=1
pixel_scale=10,1000
solid_color=1e1e2e
wallpaper_cmd="feh --no-fehbg --bg-fill"
quiet=false
EOF

cat <<'EOF' > "$HOME/.local/bin/screenlock"
#!/bin/bash
if command -v betterlockscreen >/dev/null 2>&1 && [ -f ~/.cache/betterlockscreen/current/wall_blur.png ]; then
    betterlockscreen -l dimblur
elif command -v betterlockscreen >/dev/null 2>&1; then
    betterlockscreen -l
elif command -v i3lock >/dev/null 2>&1; then
    i3lock -c 1e1e2e
else
    xflock4
fi
EOF
chmod +x "$HOME/.local/bin/screenlock"

echo "==> Desacoplando xfdesktop y configurando Feh para fondos ultra-rápidos..."
cat <<'EOF' > "$HOME/.local/bin/wallpaper.sh"
#!/bin/bash
# Desacoplar xfdesktop si estuviese en ejecucion
killall xfdesktop 2>/dev/null || true

# Imagen de fondo predeterminada
WALLPAPER="/usr/share/xfce4/backdrops/xubuntu-wallpaper.png"
[ ! -f "$WALLPAPER" ] && WALLPAPER="/usr/share/xfce4/backdrops/xubuntu-plucky.png"
[ ! -f "$WALLPAPER" ] && WALLPAPER="$(ls -1 /usr/share/xfce4/backdrops/*.png 2>/dev/null | head -n 1)"
[ ! -f "$WALLPAPER" ] && WALLPAPER="/usr/share/backgrounds/warty-final-ubuntu.png"

# Pintar fondo con feh en milisegundos (o nitrogen)
if command -v feh >/dev/null 2>&1; then
    feh --no-fehbg --bg-fill "$WALLPAPER"
elif command -v nitrogen >/dev/null 2>&1; then
    nitrogen --set-zoom-fill "$WALLPAPER" --save
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
Name=Wallpaper Daemon
Comment=Set desktop background via feh and decouple xfdesktop
EOF

# Desacoplar xfdesktop de los clientes automáticos de xfce4-session
xfconf-query -c xfce4-session -p /sessions/Failsafe/Client4_Command -r -R 2>/dev/null || true
xfconf-query -c xfce4-session -p /sessions/Failsafe/Count -s 4 2>/dev/null || true
killall xfdesktop 2>/dev/null || true
("$HOME/.local/bin/wallpaper.sh" >/dev/null 2>&1 &) || true

# Generar cache inicial de betterlockscreen si el binario esta disponible
if command -v betterlockscreen >/dev/null 2>&1; then
    WALL="/usr/share/xfce4/backdrops/xubuntu-wallpaper.png"
    [ ! -f "$WALL" ] && WALL="/usr/share/xfce4/backdrops/xubuntu-plucky.png"
    [ -f "$WALL" ] && (betterlockscreen -u "$WALL" >/dev/null 2>&1 &) || true
fi

echo "==> 5. Aplicando estilos del entorno y atajos de teclado..."
xfconf-query -c xsettings -p /Net/ThemeName -s "Orchis-Dark" 2>/dev/null || true
xfconf-query -c xsettings -p /Net/IconThemeName -s "Tela-circle-dark" 2>/dev/null || true
xfconf-query -c xsettings -p /Gtk/CursorThemeName -s "DMZ-Black" 2>/dev/null || true
xfconf-query -c xsettings -p /Gtk/FontName -s "Inter 10" 2>/dev/null || true
xfconf-query -c xsettings -p /Gtk/MonospaceFontName -s "JetBrains Mono 10" 2>/dev/null || true

xfconf-query -c xfwm4 -p /general/theme -s "Orchis-Dark" 2>/dev/null || true
xfconf-query -c xfwm4 -p /general/title_font -s "Inter Bold 10" 2>/dev/null || true
xfconf-query -c xfwm4 -p /general/button_layout -s "CHM|T" 2>/dev/null || true
xfconf-query -c xfwm4 -p /general/borderless_maximize -s true 2>/dev/null || true

# Atajos para abrir Rofi Launcher con tecla Windows / Super (vía Alt+F1 y xcape) y Ctrl+Escape
xfconf-query -c xfce4-keyboard-shortcuts -p "/commands/custom/<Alt>F1" -n -t string -s "$HOME/.local/bin/rofi-launcher" 2>/dev/null || \
xfconf-query -c xfce4-keyboard-shortcuts -p "/commands/custom/<Alt>F1" -s "$HOME/.local/bin/rofi-launcher" 2>/dev/null || true

xfconf-query -c xfce4-keyboard-shortcuts -p "/commands/custom/<Primary>Escape" -n -t string -s "$HOME/.local/bin/rofi-launcher" 2>/dev/null || \
xfconf-query -c xfce4-keyboard-shortcuts -p "/commands/custom/<Primary>Escape" -s "$HOME/.local/bin/rofi-launcher" 2>/dev/null || true

# Limpiar bindings directos de Super_L y Super_R en custom shortcuts para que xcape traduzca el tap sin colisiones
xfconf-query -c xfce4-keyboard-shortcuts -p "/commands/custom/Super_L" -r 2>/dev/null || true
xfconf-query -c xfce4-keyboard-shortcuts -p "/commands/custom/Super_R" -r 2>/dev/null || true
xfconf-query -c xfce4-keyboard-shortcuts -p "/commands/custom/xfce4-popup-whiskermenu" -r 2>/dev/null || true

# Atajos para portapapeles (Greenclip + Rofi), lanzador rápido (Rofi) y bloqueo (Betterlockscreen)
xfconf-query -c xfce4-keyboard-shortcuts -p "/commands/custom/<Super>v" -n -t string -s "$HOME/.local/bin/rofi-clipboard" 2>/dev/null || \
xfconf-query -c xfce4-keyboard-shortcuts -p "/commands/custom/<Super>v" -s "$HOME/.local/bin/rofi-clipboard" 2>/dev/null || true

# Alternar distribución de teclado con Win + Space (US / ES)
xfconf-query -c xfce4-keyboard-shortcuts -p "/commands/custom/<Super>space" -n -t string -s "$HOME/.local/bin/toggle-layout.sh" 2>/dev/null || \
xfconf-query -c xfce4-keyboard-shortcuts -p "/commands/custom/<Super>space" -s "$HOME/.local/bin/toggle-layout.sh" 2>/dev/null || true

# Configuración de layouts en el canal keyboards de Xfce
xfconf-query -c keyboards -p /Default/XkbLayout -n -t string -s "us,es" 2>/dev/null || \
xfconf-query -c keyboards -p /Default/XkbLayout -s "us,es" 2>/dev/null || true
xfconf-query -c keyboards -p /Default/XkbVariant -n -t string -s "," 2>/dev/null || \
xfconf-query -c keyboards -p /Default/XkbVariant -s "," 2>/dev/null || true

xfconf-query -c xfce4-keyboard-shortcuts -p "/commands/custom/<Super>l" -n -t string -s "$HOME/.local/bin/screenlock" 2>/dev/null || \
xfconf-query -c xfce4-keyboard-shortcuts -p "/commands/custom/<Super>l" -s "$HOME/.local/bin/screenlock" 2>/dev/null || true

xfconf-query -c xfce4-keyboard-shortcuts -p "/commands/custom/<Primary><Alt>l" -n -t string -s "$HOME/.local/bin/screenlock" 2>/dev/null || \
xfconf-query -c xfce4-keyboard-shortcuts -p "/commands/custom/<Primary><Alt>l" -s "$HOME/.local/bin/screenlock" 2>/dev/null || true

echo "==> 6. Configurando Xfce Panel (Whisker Menu sobrio, Notificaciones, Reloj AM/PM)..."

# Configurar plugin-1 como Whisker Menu con icono sobrio y moderno (rejilla 9-dot minimalista, sin animales)
xfconf-query -c xfce4-panel -p /plugins/plugin-1 -n -t string -s "whiskermenu" 2>/dev/null || \
xfconf-query -c xfce4-panel -p /plugins/plugin-1 -s "whiskermenu" 2>/dev/null || true

xfconf-query -c xfce4-panel -p /plugins/plugin-1/button-icon -n -t string -s "view-app-grid-symbolic" 2>/dev/null || \
xfconf-query -c xfce4-panel -p /plugins/plugin-1/button-icon -s "view-app-grid-symbolic" 2>/dev/null || true

xfconf-query -c xfce4-panel -p /plugins/plugin-1/button-title -n -t string -s "" 2>/dev/null || \
xfconf-query -c xfce4-panel -p /plugins/plugin-1/button-title -s "" 2>/dev/null || true

xfconf-query -c xfce4-panel -p /plugins/plugin-1/show-button-icon -n -t bool -s true 2>/dev/null || \
xfconf-query -c xfce4-panel -p /plugins/plugin-1/show-button-icon -s true 2>/dev/null || true

xfconf-query -c xfce4-panel -p /plugins/plugin-1/show-button-title -n -t bool -s false 2>/dev/null || \
xfconf-query -c xfce4-panel -p /plugins/plugin-1/show-button-title -s false 2>/dev/null || true

# Configurar plugin-6 como notification-plugin para historial visual de notificaciones y no-molestar
xfconf-query -c xfce4-panel -p /plugins/plugin-6 -n -t string -s "notification-plugin" 2>/dev/null || \
xfconf-query -c xfce4-panel -p /plugins/plugin-6 -s "notification-plugin" 2>/dev/null || true

# Configurar la lista canónica de plugins en panel-1
xfconf-query -c xfce4-panel -p /panels/panel-1/plugin-ids -a \
    -t int -s 1 -t int -s 2 -t int -s 3 -t int -s 4 -t int -s 5 -t int -s 6 -t int -s 7 -t int -s 8 -t int -s 9 -t int -s 10 2>/dev/null || true

# Configurar reloj en formato Digital, 12 horas AM/PM con segundos
CLOCK_PLUGIN=$(xfconf-query -c xfce4-panel -p /plugins -l 2>/dev/null | grep -E '^/plugins/plugin-[0-9]+$' | while read -r p; do
    name=$(xfconf-query -c xfce4-panel -p "$p" 2>/dev/null || true)
    if [ "$name" = "clock" ]; then
        echo "$p"
        break
    fi
done)

if [ -n "$CLOCK_PLUGIN" ]; then
    xfconf-query -c xfce4-panel -p "$CLOCK_PLUGIN/mode" -n -t uint -s 2 2>/dev/null || \
    xfconf-query -c xfce4-panel -p "$CLOCK_PLUGIN/mode" -s 2 2>/dev/null || true

    xfconf-query -c xfce4-panel -p "$CLOCK_PLUGIN/digital-layout" -n -t uint -s 3 2>/dev/null || \
    xfconf-query -c xfce4-panel -p "$CLOCK_PLUGIN/digital-layout" -s 3 2>/dev/null || true

    xfconf-query -c xfce4-panel -p "$CLOCK_PLUGIN/digital-time-format" -n -t string -s "%I:%M:%S %p" 2>/dev/null || \
    xfconf-query -c xfce4-panel -p "$CLOCK_PLUGIN/digital-time-format" -s "%I:%M:%S %p" 2>/dev/null || true

    xfconf-query -c xfce4-panel -p "$CLOCK_PLUGIN/show-seconds" -n -t bool -s true 2>/dev/null || \
    xfconf-query -c xfce4-panel -p "$CLOCK_PLUGIN/show-seconds" -s true 2>/dev/null || true

    xfconf-query -c xfce4-panel -p "$CLOCK_PLUGIN/tooltip-format" -n -t string -s "%A, %d de %B de %Y" 2>/dev/null || \
    xfconf-query -c xfce4-panel -p "$CLOCK_PLUGIN/tooltip-format" -s "%A, %d de %B de %Y" 2>/dev/null || true
fi

# Eliminar panel 2 de la configuración de panels
xfconf-query -c xfce4-panel -p /panels -a -t int -s 1 2>/dev/null || true
xfconf-query -c xfce4-panel -p /panels/panel-2 -r -R 2>/dev/null || true

# Configurar dimensiones del panel 1
xfconf-query -c xfce4-panel -p /panels/panel-1/size -s 32 2>/dev/null || true
xfconf-query -c xfce4-panel -p /panels/panel-1/icon-size -s 16 2>/dev/null || true
xfconf-query -c xfce4-panel -p /panels/panel-1/autohide-behavior -s 0 2>/dev/null || true

# Reiniciar panel para aplicar cambios en vivo
xfce4-panel -r 2>/dev/null || true

echo "==> 7. Configurando Plank Dock (sin Remmina y preservando launchers actuales)..."
dconf write /net/launchpad/plank/docks/dock1/theme "'Transparent'" 2>/dev/null || true
dconf write /net/launchpad/plank/docks/dock1/icon-size 44 2>/dev/null || true
dconf write /net/launchpad/plank/docks/dock1/hide-mode "'auto'" 2>/dev/null || true

# Asegurar que no existan accesos a Remmina en el dock
rm -f ~/.config/plank/dock1/launchers/remmina*.dockitem 2>/dev/null || true

# Si la carpeta de launchers no existe o está vacía, crear estructura inicial limpia
mkdir -p ~/.config/plank/dock1/launchers
if [ -z "$(ls -A ~/.config/plank/dock1/launchers 2>/dev/null)" ]; then
    cat <<'EOF' > ~/.config/plank/dock1/launchers/home.dockitem
[PlankDockItemPreferences]
Launcher=file://$HOME
EOF
    # Terminal
    if [ -f "/usr/share/applications/com.mitchellh.ghostty.desktop" ]; then
        echo -e "[PlankDockItemPreferences]\nLauncher=file:///usr/share/applications/com.mitchellh.ghostty.desktop" > ~/.config/plank/dock1/launchers/ghostty.dockitem
    fi
    # Navegador
    if [ -f "/usr/share/applications/brave-browser.desktop" ]; then
        echo -e "[PlankDockItemPreferences]\nLauncher=file:///usr/share/applications/brave-browser.desktop" > ~/.config/plank/dock1/launchers/brave.dockitem
    elif [ -f "/usr/share/applications/google-chrome.desktop" ]; then
        echo -e "[PlankDockItemPreferences]\nLauncher=file:///usr/share/applications/google-chrome.desktop" > ~/.config/plank/dock1/launchers/chrome.dockitem
    fi
fi

# Reiniciar plank si está en ejecución para reflejar cambios
killall plank 2>/dev/null || true
(plank >/dev/null 2>&1 &)

echo "==> 8. Optimizando el arranque del sistema (mitigación de hardware, red y bases de datos a demanda)..."
# 1. Hardware TPM inaccesible (elimina timeout crítico de 90s en Dell XPS 13)
sudo systemctl mask tpm2.target 2>/dev/null || true
sudo mkdir -p /etc/modprobe.d
echo "blacklist tpm_crb" | sudo tee /etc/modprobe.d/blacklist-tpm.conf >/dev/null

# 2. NetworkManager-wait-online (elimina bloqueo de ~10s esperando conectividad)
sudo systemctl disable NetworkManager-wait-online.service 2>/dev/null || true

# 3. Bases de datos: Configuración bajo demanda (ahorro de RAM y ~9s de tiempo de inicio)
# PostgreSQL: Desactivar inicio automático global
sudo systemctl disable postgresql.service 2>/dev/null || true
if command -v pg_lsclusters >/dev/null 2>&1; then
    LATEST_PG=$(pg_lsclusters -h 2>/dev/null | awk '{print $1}' | sort -V | tail -n 1)
    for ver in $(pg_lsclusters -h 2>/dev/null | awk '{print $1}'); do
        if [ "$ver" = "$LATEST_PG" ]; then
            # Mantener última versión en inicio manual (a demanda con: pg_ctlcluster 18 main start o systemctl start postgresql@18-main)
            if [ -f "/etc/postgresql/$ver/main/start.conf" ]; then
                echo "manual" | sudo tee "/etc/postgresql/$ver/main/start.conf" >/dev/null
            fi
            sudo pg_ctlcluster "$ver" main stop 2>/dev/null || true
            sudo systemctl stop "postgresql@$ver-main" 2>/dev/null || true
            sudo systemctl disable "postgresql@$ver-main" 2>/dev/null || true
        else
            # Detener y deshabilitar versiones obsoletas
            if [ -f "/etc/postgresql/$ver/main/start.conf" ]; then
                echo "manual" | sudo tee "/etc/postgresql/$ver/main/start.conf" >/dev/null
            fi
            sudo pg_ctlcluster "$ver" main stop 2>/dev/null || true
            sudo systemctl stop "postgresql@$ver-main" 2>/dev/null || true
            sudo systemctl disable "postgresql@$ver-main" 2>/dev/null || true
        fi
    done
fi

# MySQL / MariaDB (arranque manual si existiesen)
sudo systemctl disable mysql.service mariadb.service 2>/dev/null || true
sudo systemctl stop mysql.service mariadb.service 2>/dev/null || true

echo "==> Configuración de Xfce Pro completada exitosamente."
echo "    - LightDM integrado como gestor de sesión inicial por defecto."
echo "    - Whisker Menu integrado en panel superior izquierdo con icono sobrio 'view-app-grid-symbolic'."
echo "    - xfce4-notifyd activo con tema Orchis-Dark, barra visual de progreso y widget de historial en el panel."
echo "    - Rofi como lanzador rápido asignado a la tecla Windows / Super (tap sin colisiones)."
echo "    - Selector Alt-Tab con previsualización en vivo (Grid Thumbnails) vía compositor GLX nativo de xfwm4."
echo "    - Greenclip + Rofi configurado: Super+V para portapapeles searchable, Super+Space para Rofi."
echo "    - Betterlockscreen e i3lock-color listos: Super+L para bloqueo con efecto dimblur."
echo "    - xfdesktop desacoplado: fondos de pantalla aplicados al vuelo en milisegundos con Feh."
echo "    - Touchpad con botón derecho físico funcional (método buttonareas)."
echo "    - Reloj configurado en formato 12 horas (AM/PM) con segundos."
echo "    - Optimización de arranque: tpm2.target enmascarado (ahorro 90s), wait-online apagado, bases de datos a demanda."
echo "    Para aplicar todos los cambios de sesión y display manager por completo, reinicia con: sudo reboot"
