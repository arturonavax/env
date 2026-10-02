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
    xfce4-goodies xfce4-power-manager xfce4-screenshooter \
    xcape xdotool brightnessctl pavucontrol network-manager-gnome \
    pipewire pipewire-pulse wireplumber \
    xdg-desktop-portal xdg-desktop-portal-gtk \
    tumbler ffmpegthumbnailer poppler-data tumbler-plugins-extra webp-pixbuf-loader \
    gvfs-backends gvfs-fuse policykit-1-gnome \
    fonts-inter fonts-jetbrains-mono \
    plank dconf-cli libglib2.0-bin libglib2.0-dev-bin libnotify-bin \
    picom rofi dunst feh imagemagick bc libxcb-xrm0 \
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

echo "==> Limpiando paquetes y herramientas obsoletas o reemplazadas..."
sudo apt purge -y \
    remmina remmina-plugin-rdp remmina-plugin-vnc remmina-plugin-secret remmina-common \
    xfce4-whiskermenu-plugin xfce4-notifyd 2>/dev/null || true
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

echo "==> 4. Configurando compatibilidad (Touchpad, Picom, PipeWire, Polkit, Xcape)..."
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

# Configuración óptima y ultra ligera de Picom (GLX + vsync + unredir + sin blur pesado ni sombras)
mkdir -p "$HOME/.config/picom"
cat <<'EOF' > "$HOME/.config/picom/picom.conf"
# ==============================================================================
# Picom Configuration - Optimized, High-Performance, Tear-Free
# ==============================================================================
backend = "glx";
vsync = true;

# Evita procesar aplicaciones a pantalla completa (juegos, video)
unredir-if-possible = true;

# Desactivar sombras pesadas para maximo rendimiento y evitar artefactos
shadow = false;

# Fading ligero y suave
fading = true;
fade-in-step = 0.08;
fade-out-step = 0.08;
fade-delta = 10;

# Desactivar blur pesado
blur-background = false;

# Bordes redondeados sutiles (sin lag)
corner-radius = 8;
rounded-corners-exclude = [
  "window_type = 'dock'",
  "window_type = 'desktop'",
  "class_g = 'xfce4-panel'",
  "class_g = 'Plank'"
];

# Optimizaciones de renderizado y sincronizacion
mark-wmwin-focused = true;
mark-ovredir-focused = true;
detect-rounded-corners = true;
detect-client-opacity = true;
detect-transient = true;
use-damage = true;
glx-no-stencil = true;
glx-no-rebind-pixmap = true;
EOF

# Desactivar compositor integrado de xfwm4 para cederle el control exclusivo a Picom
xfconf-query -c xfwm4 -p /general/use_compositing -n -t bool -s false 2>/dev/null || \
xfconf-query -c xfwm4 -p /general/use_compositing -s false 2>/dev/null || true

# Autostart: Picom
cat <<EOF > "$HOME/.config/autostart/picom.desktop"
[Desktop Entry]
Type=Application
Exec=picom -b --config $HOME/.config/picom/picom.conf
Hidden=false
NoDisplay=false
X-GNOME-Autostart-enabled=true
Name=Picom Compositor
Comment=Optimized X11 OpenGL compositor
EOF

# Iniciar Picom en caliente si estamos en sesión gráfica
killall picom 2>/dev/null || true
(picom -b --config "$HOME/.config/picom/picom.conf" >/dev/null 2>&1 &) || true

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
rofi -modi "clipboard:greenclip print" -show clipboard -run-command '{cmd}'
EOF
chmod +x "$HOME/.local/bin/rofi-clipboard"

cat <<'EOF' > "$HOME/.local/bin/rofi-launcher"
#!/bin/bash
rofi -show drun -show-icons
EOF
chmod +x "$HOME/.local/bin/rofi-launcher"

echo "==> Configurando Dunst (reemplazo ultra-ligero de xfce4-notifyd)..."
mkdir -p "$HOME/.config/dunst" "$HOME/.local/share/dbus-1/services"
cat <<'EOF' > "$HOME/.config/dunst/dunstrc"
[global]
    monitor = 0
    follow = mouse
    width = (300, 480)
    height = (50, 160)
    origin = top-right
    offset = (20, 48)
    scale = 0
    notification_limit = 5
    progress_bar = true
    progress_bar_height = 8
    progress_bar_frame_width = 1
    progress_bar_min_width = 150
    progress_bar_max_width = 320
    progress_bar_corner_radius = 4
    indicate_hidden = yes
    transparency = 10
    separator_height = 2
    padding = 12
    horizontal_padding = 14
    text_icon_padding = 12
    frame_width = 2
    frame_color = "#383f4a"
    gap_size = 6
    separator_color = frame
    sort = yes
    font = Inter 10
    line_height = 0
    markup = full
    format = "<b>%s</b>\n%b"
    alignment = left
    vertical_alignment = center
    show_age_threshold = 60
    ellipsize = middle
    ignore_newline = no
    stack_duplicates = true
    hide_duplicate_count = false
    show_indicators = yes
    enable_recursive_icon_lookup = true
    icon_theme = "Tela-circle-dark, elementary-xfce-dark, Adwaita"
    icon_position = left
    min_icon_size = 24
    max_icon_size = 48
    sticky_history = yes
    history_length = 20
    browser = /usr/bin/xdg-open
    always_run_script = true
    title = Dunst
    class = Dunst
    corner_radius = 8
    ignore_dbusclose = false
    mouse_left_click = close_current
    mouse_middle_click = do_action, close_current
    mouse_right_click = close_all

[urgency_low]
    background = "#1e1e2e"
    foreground = "#cdd6f4"
    frame_color = "#313244"
    timeout = 4

[urgency_normal]
    background = "#1e1e2e"
    foreground = "#cdd6f4"
    frame_color = "#89b4fa"
    timeout = 6

[urgency_critical]
    background = "#1e1e2e"
    foreground = "#f38ba8"
    frame_color = "#f38ba8"
    timeout = 0
EOF

cat <<'EOF' > "$HOME/.local/share/dbus-1/services/org.freedesktop.Notifications.service"
[D-BUS Service]
Name=org.freedesktop.Notifications
Exec=/usr/bin/dunst
EOF

cat <<'EOF' > "$HOME/.config/autostart/dunst.desktop"
[Desktop Entry]
Type=Application
Exec=dunst
Hidden=false
NoDisplay=false
X-GNOME-Autostart-enabled=true
Name=Dunst
Comment=Lightweight notification daemon
EOF

killall xfce4-notifyd 2>/dev/null || true
systemctl --user mask xfce4-notifyd.service 2>/dev/null || true
systemctl --user stop xfce4-notifyd.service 2>/dev/null || true
killall dunst 2>/dev/null || true
(dunst >/dev/null 2>&1 &) || true

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

xfconf-query -c xfce4-keyboard-shortcuts -p "/commands/custom/<Super>space" -n -t string -s "$HOME/.local/bin/rofi-launcher" 2>/dev/null || \
xfconf-query -c xfce4-keyboard-shortcuts -p "/commands/custom/<Super>space" -s "$HOME/.local/bin/rofi-launcher" 2>/dev/null || true

xfconf-query -c xfce4-keyboard-shortcuts -p "/commands/custom/<Super>l" -n -t string -s "$HOME/.local/bin/screenlock" 2>/dev/null || \
xfconf-query -c xfce4-keyboard-shortcuts -p "/commands/custom/<Super>l" -s "$HOME/.local/bin/screenlock" 2>/dev/null || true

xfconf-query -c xfce4-keyboard-shortcuts -p "/commands/custom/<Primary><Alt>l" -n -t string -s "$HOME/.local/bin/screenlock" 2>/dev/null || \
xfconf-query -c xfce4-keyboard-shortcuts -p "/commands/custom/<Primary><Alt>l" -s "$HOME/.local/bin/screenlock" 2>/dev/null || true

echo "==> 6. Configurando Xfce Panel (Reloj AM/PM con segundos, limpieza de menús)..."

# Remover cualquier plugin de menú (whiskermenu, applicationsmenu) para una barra limpia y moderna
for p in $(xfconf-query -c xfce4-panel -p /plugins -l 2>/dev/null | grep -E '^/plugins/plugin-[0-9]+$'); do
    pname=$(xfconf-query -c xfce4-panel -p "$p" 2>/dev/null || true)
    if [ "$pname" = "whiskermenu" ] || [ "$pname" = "applicationsmenu" ]; then
        pid=$(echo "$p" | sed 's|/plugins/plugin-||')
        current_ids=$(xfconf-query -c xfce4-panel -p /panels/panel-1/plugin-ids 2>/dev/null | grep -E '^[0-9]+$' | grep -v "^$pid$" || true)
        if [ -n "$current_ids" ]; then
            cmd="xfconf-query -c xfce4-panel -p /panels/panel-1/plugin-ids"
            for id in $current_ids; do
                cmd="$cmd -t int -s $id"
            done
            eval "$cmd" 2>/dev/null || true
        fi
        xfconf-query -c xfce4-panel -p "$p" -r -R 2>/dev/null || true
    fi
done

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

echo "==> Configuración de Xfce Pro completada exitosamente."
echo "    - LightDM integrado como gestor de sesión inicial por defecto."
echo "    - Rofi como lanzador principal asignado a la tecla Windows / Super (tap sin colisiones)."
echo "    - Picom optimizado con backend GLX, vsync y bypass fullscreen activo (sin sombras pesadas)."
echo "    - Greenclip + Rofi configurado: Super+V para portapapeles searchable, Super+Space para Rofi."
echo "    - Dunst configurado como daemon de notificaciones moderno y ligero (reemplazando a xfce4-notifyd)."
echo "    - Betterlockscreen e i3lock-color listos: Super+L para bloqueo con efecto dimblur."
echo "    - xfdesktop desacoplado: fondos de pantalla aplicados al vuelo en milisegundos con Feh."
echo "    - Touchpad con botón derecho físico funcional (método buttonareas)."
echo "    - Reloj configurado en formato 12 horas (AM/PM) con segundos."
echo "    - Remmina, Whisker Menu y xfce4-notifyd purgados del sistema."
echo "    Para aplicar todos los cambios de sesión y display manager por completo, reinicia con: sudo reboot"
