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

pgrep -x xfce4-panel >/dev/null || (xfce4-panel >/dev/null 2>&1 & sleep 1)

echo "==> 1. Purgando dependencias obsoletas y scripts huérfanos..."
sudo apt-mark manual xfce4-panel xfce4-pulseaudio-plugin xfce4-appfinder libgarcon-gtk3-1-0 2>/dev/null || true

sudo apt purge -y \
    xfdesktop4 plank feh xcape \
    remmina remmina-plugin-rdp remmina-plugin-vnc remmina-plugin-secret remmina-common \
    alttab skippy-xd \
    2>/dev/null || true
sudo apt autoremove -y 2>/dev/null || true

sudo rm -f /usr/local/bin/greenclip /usr/local/bin/i3lock-color /usr/local/bin/betterlockscreen \
           /usr/local/bin/alttab /usr/local/bin/skippy-xd /usr/local/bin/rofi-window
rm -f "$HOME/.local/bin/greenclip" "$HOME/.local/bin/i3lock-color" "$HOME/.local/bin/betterlockscreen" \
      "$HOME/.local/bin/alttab" "$HOME/.local/bin/alttab-daemon.sh" "$HOME/.local/bin/skippy-xd" \
      "$HOME/.local/bin/skippy-xd.bin" "$HOME/.local/bin/rofi-window" "$HOME/.local/bin/rofi-alt-tab-watcher" \
      "$HOME/.local/src/rofi-alt-tab-watcher.c" "$HOME/.local/bin/toggle-layout.sh"
rm -rf "$HOME/.config/betterlockscreen" "$HOME/.config/plank" "$HOME/.config/skippy-xd" "$HOME/.cache/greenclip.history"
rm -f "$HOME/.config/autostart/plank.desktop" "$HOME/.config/autostart/xcape.desktop" \
      "$HOME/.config/autostart/greenclip.desktop" "$HOME/.config/autostart/touchpad-setup.desktop" \
      "$HOME/.config/autostart/alttab.desktop" "$HOME/.config/autostart/skippy-xd.desktop" \
      "$HOME/.config/autostart/xfdashboard.desktop" \
      "$HOME/.config/systemd/user/skippy-xd.service" "$HOME/.config/systemd/user/xfdashboard.service"

echo "==> 2. Instalando stack base, Picom, Rofi y dependencias..."
sudo apt update
sudo apt install -y \
    lightdm lightdm-gtk-greeter lightdm-gtk-greeter-settings light-locker \
    xfce4-panel xfce4-pulseaudio-plugin xfce4-appfinder mugshot \
    xfce4-goodies xfce4-whiskermenu-plugin xfce4-notifyd xfce4-power-manager xfce4-screenshooter xfce4-xkb-plugin \
    xdotool brightnessctl pavucontrol network-manager-gnome \
    pipewire pipewire-pulse wireplumber \
    xdg-desktop-portal xdg-desktop-portal-gtk \
    tumbler ffmpegthumbnailer poppler-data tumbler-plugins-extra webp-pixbuf-loader \
    gvfs-backends gvfs-fuse policykit-1-gnome \
    fonts-inter fonts-jetbrains-mono \
    dconf-cli libglib2.0-bin libglib2.0-dev-bin libnotify-bin \
    xwallpaper libxcb-xrm0 \
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

echo "==> 4. Configurando Vicinae con icono oficial transparente y persistente..."
# 1. Asegurar binario de Vicinae
if ! command -v vicinae >/dev/null 2>&1; then
    curl -fsSL --connect-timeout 5 -m 30 https://vicinae.com/install | bash -s -- --prefix "$HOME/.local"
fi

if [ -f "$HOME/.local/bin/vicinae" ]; then
    sudo ln -sf "$HOME/.local/bin/vicinae" /usr/local/bin/vicinae 2>/dev/null || true
fi

# 2. Desactivar permanentemente daemons de indicadores de Ubuntu que bloquean el StatusNotifierWatcher
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

# 3. Purgar cualquier copia residual del SVG sintético generado anteriormente
find /usr/share/icons /usr/share/pixmaps "$HOME/.local/share/icons" \
     -type f \( -name "*vicinae*.svg" -o -name "*vicinae*.png" \) 2>/dev/null | while read -r f; do
    if grep -q "vicinae-grad" "$f" 2>/dev/null; then
        sudo rm -f "$f"
    fi
done

# 4. Obtener el asset oficial auténtico de forma no bloqueante
TMP_VIC=$(mktemp -d /tmp/vic_official.XXXXXX)
trap 'rm -rf "$TMP_VIC"' EXIT

OFFICIAL_ICON=""

# Comprobar primero en disco si ya existe un SVG oficial libre de gradientes sintéticos
for local_p in "/usr/share/icons/hicolor/scalable/apps/vicinae.svg" \
               "/usr/share/pixmaps/vicinae.svg" \
               "$HOME/.local/share/icons/hicolor/scalable/apps/vicinae.svg" \
               "$HOME/.local/share/vicinae/icons/vicinae.svg"; do
    if [ -f "$local_p" ] && ! grep -q "vicinae-grad" "$local_p" 2>/dev/null; then
        OFFICIAL_ICON="$local_p"
        break
    fi
done

# Si no está en disco, descargar directamente el SVG oficial de los repositorios sin ejecutar scripts externos
if [ -z "$OFFICIAL_ICON" ]; then
    CANDIDATE_URLS=(
        "https://raw.githubusercontent.com/vicinaehq/vicinae/main/crates/vicinae/assets/icon.svg"
        "https://raw.githubusercontent.com/vicinaehq/vicinae/main/assets/vicinae.svg"
        "https://raw.githubusercontent.com/vicinaehq/vicinae/main/data/icons/hicolor/scalable/apps/vicinae.svg"
        "https://vicinae.com/icon.svg"
        "https://vicinae.com/logo.svg"
    )
    for url in "${CANDIDATE_URLS[@]}"; do
        if curl -fsSL --connect-timeout 4 -m 8 "$url" -o "$TMP_VIC/vicinae.svg" 2>/dev/null; then
            if [ -s "$TMP_VIC/vicinae.svg" ] && ! grep -qi "404" "$TMP_VIC/vicinae.svg"; then
                OFFICIAL_ICON="$TMP_VIC/vicinae.svg"
                break
            fi
        fi
    done
fi

# 5. Desplegar vectoriales y rasterizados en el sistema
if [ -n "$OFFICIAL_ICON" ] && [ -f "$OFFICIAL_ICON" ]; then
    EXT="${OFFICIAL_ICON##*.}"
    sudo mkdir -p /usr/share/pixmaps \
                  /usr/share/icons/hicolor/scalable/apps \
                  /usr/share/icons/hicolor/scalable/status \
                  /usr/share/icons/Tela-circle-dark/scalable/apps \
                  /usr/share/icons/Tela-circle-dark/scalable/panel

    for name in vicinae vicinae-tray vicinae-indicator vicinae-status com.vicinae.Vicinae; do
        sudo cp -f "$OFFICIAL_ICON" "/usr/share/pixmaps/${name}.${EXT}"
        sudo cp -f "$OFFICIAL_ICON" "/usr/share/icons/hicolor/scalable/apps/${name}.${EXT}"
        sudo cp -f "$OFFICIAL_ICON" "/usr/share/icons/hicolor/scalable/status/${name}.${EXT}"
        sudo cp -f "$OFFICIAL_ICON" "/usr/share/icons/Tela-circle-dark/scalable/apps/${name}.${EXT}"
        sudo cp -f "$OFFICIAL_ICON" "/usr/share/icons/Tela-circle-dark/scalable/panel/${name}.${EXT}"
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

            $CONVERT_BIN "$OFFICIAL_ICON" -background none -resize "${sz}x${sz}" "$TMP_VIC/icon_${sz}.png"

            for name in vicinae vicinae-tray vicinae-indicator vicinae-status com.vicinae.Vicinae; do
                sudo cp -f "$TMP_VIC/icon_${sz}.png" "/usr/share/icons/hicolor/${sz}x${sz}/status/${name}.png"
                sudo cp -f "$TMP_VIC/icon_${sz}.png" "/usr/share/icons/hicolor/${sz}x${sz}/apps/${name}.png"
                sudo cp -f "$TMP_VIC/icon_${sz}.png" "/usr/share/icons/Tela-circle-dark/${sz}x${sz}/panel/${name}.png"
                sudo cp -f "$TMP_VIC/icon_${sz}.png" "/usr/share/icons/Tela-circle-dark/${sz}x${sz}/apps/${name}.png"
                [ "$sz" -eq 22 ] && sudo cp -f "$TMP_VIC/icon_${sz}.png" "/usr/share/pixmaps/${name}.png"
            done
        done
    fi

    sudo gtk-update-icon-cache -f -q /usr/share/icons/hicolor 2>/dev/null || true
    sudo gtk-update-icon-cache -f -q /usr/share/icons/Tela-circle-dark 2>/dev/null || true
    gtk-update-icon-cache -f -q "$HOME/.local/share/icons/hicolor" 2>/dev/null || true
fi
rm -rf "$TMP_VIC"

# 6. Autostart con retardo sincronizado (evita la caída a XEmbed al reiniciar el sistema)
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

# Reiniciar procesos y levantar en segundo plano de forma desasociada
killall -9 vicinae vicinae-server 2>/dev/null || true
killall -q ayatana-indicator-application-service indicator-application-service 2>/dev/null || true
(sleep 1 && /usr/local/bin/vicinae server >/dev/null 2>&1 &)

echo "==> 5. Configurando LightDM y Light-Locker..."
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

echo "==> 7. Configurando Picom, temas Rofi y Switchers de ventana..."
xfconf-query -c xfwm4 -p /general/use_compositing -s false 2>/dev/null || true
xfconf-query -c xfwm4 -p /general/show_frame_shadow -s false 2>/dev/null || true
xfconf-query -c xfwm4 -p /general/show_popup_shadow -s false 2>/dev/null || true
xfconf-query -c xfwm4 -p /general/show_dock_shadow -s false 2>/dev/null || true
xfconf-query -c xfwm4 -p /general/raise_on_focus -s true 2>/dev/null || true

xfconf-query -c xfwm4 -p /general/prevent_focus_stealing -s false 2>/dev/null || true
xfconf-query -c xfwm4 -p /general/focus_new -s true 2>/dev/null || true

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
    "class_g = 'Rofi'",
    "class_g = 'flameshot'",
    "class_g = 'Flameshot'",
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
    "window_type = 'notification'",
    "_NET_WM_STATE *= '_NET_WM_STATE_HIDDEN'"
];

blur-background-exclude = [
    "class_g = 'flameshot'",
    "class_g = 'Flameshot'",
    "window_type = 'dock'",
    "window_type = 'desktop'"
];

focus-exclude = [];

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

echo "==> 8. Configurando xwallpaper..."
cat <<'EOF' > "$HOME/.local/bin/wallpaper.sh"
#!/bin/bash
set -e

CONFIG_FILE="$HOME/.config/wallpaper"

is_valid_image() {
    local f="$1"
    [ -f "$f" ] || return 1
    local mime
    mime=$(file -b -L --mime-type "$f" 2>/dev/null || true)
    [[ "$mime" =~ ^image/(png|jpeg) ]]
}

WALLPAPER=""

if [ -n "${1:-}" ] && is_valid_image "$1"; then
    WALLPAPER="$(realpath "$1")"
    echo "$WALLPAPER" > "$CONFIG_FILE"
fi

if [ -z "$WALLPAPER" ] && [ -f "$CONFIG_FILE" ]; then
    SAVED=$(cat "$CONFIG_FILE" 2>/dev/null || true)
    if is_valid_image "$SAVED"; then
        WALLPAPER="$SAVED"
    fi
fi

if [ -z "$WALLPAPER" ]; then
    CANDIDATES=(
        "$HOME/Pictures/Wallpapers"/*
        "/usr/share/backgrounds/Resolute_Raccoon_Wallpaper_Dimmed_3840x2160.png"
        "/usr/share/backgrounds/warty-final-ubuntu.png"
        "/usr/share/backgrounds/cnusr25-Simple_Raccoon_Dark.png"
        "/usr/share/backgrounds/endycal-Flying_Boxes_Dark.png"
        "/usr/share/backgrounds/ezspain-Ubuntu_Coffee_Mug_Dark.png"
        "/usr/share/backgrounds"/*.png
        "/usr/share/backgrounds"/*.jpg
        "/usr/share/xfce4/backdrops"/*.png
        "/usr/share/xfce4/backdrops"/*.jpg
    )
    for w in "${CANDIDATES[@]}"; do
        if is_valid_image "$w"; then
            WALLPAPER="$w"
            echo "$WALLPAPER" > "$CONFIG_FILE"
            break
        fi
    done
fi

if command -v xwallpaper >/dev/null 2>&1 && [ -n "$WALLPAPER" ]; then
    killall -q xwallpaper 2>/dev/null || true
    xwallpaper --daemon --zoom "$WALLPAPER"
fi
EOF
chmod +x "$HOME/.local/bin/wallpaper.sh"
("$HOME/.local/bin/wallpaper.sh" >/dev/null 2>&1 &) || true

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

xfconf-query -c xfce4-session -p /sessions/Failsafe/Client4_Command -s "/bin/true" -t string 2>/dev/null || true
xfconf-query -c xfce4-session -p /sessions/Failsafe/Count -s 4 2>/dev/null || true

echo "==> 9. Configurando atajos globales y XKB nativo (desactivando IBus)..."
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

echo "==> Configuración completada exitosamente."
