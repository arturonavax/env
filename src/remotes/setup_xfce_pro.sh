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

echo "==> 1. Purgando dependencias obsoletas (xfdesktop, plank, betterlockscreen, alttab, skippy-xd)..."
# Proteger componentes esenciales de Xfce para que no sean eliminados por autoremove
sudo apt-mark manual xfce4-panel xfce4-pulseaudio-plugin xfce4-appfinder libgarcon-gtk3-1-0 2>/dev/null || true

sudo apt purge -y \
    xfdesktop4 plank feh xcape \
    remmina remmina-plugin-rdp remmina-plugin-vnc remmina-plugin-secret remmina-common \
    alttab skippy-xd \
    2>/dev/null || true
sudo apt autoremove -y 2>/dev/null || true

# Limpieza de binarios manuales anteriores y configs huérfanas
sudo rm -f /usr/local/bin/greenclip /usr/local/bin/i3lock-color /usr/local/bin/betterlockscreen /usr/local/bin/alttab /usr/local/bin/skippy-xd
rm -f "$HOME/.local/bin/greenclip" "$HOME/.local/bin/i3lock-color" "$HOME/.local/bin/betterlockscreen" "$HOME/.local/bin/alttab" "$HOME/.local/bin/alttab-daemon.sh" "$HOME/.local/bin/skippy-xd" "$HOME/.local/bin/skippy-xd.bin"
rm -rf "$HOME/.config/betterlockscreen" "$HOME/.config/plank" "$HOME/.config/skippy-xd" "$HOME/.cache/greenclip.history"
rm -f "$HOME/.config/autostart/plank.desktop" "$HOME/.config/autostart/xcape.desktop" \
      "$HOME/.config/autostart/greenclip.desktop" "$HOME/.config/autostart/touchpad-setup.desktop" \
      "$HOME/.config/autostart/alttab.desktop" "$HOME/.config/autostart/skippy-xd.desktop" "$HOME/.config/autostart/xfdashboard.desktop" \
      "$HOME/.config/systemd/user/skippy-xd.service" "$HOME/.config/systemd/user/xfdashboard.service"

echo "==> 2. Instalando stack base, Picom, Rofi y dependencias de sistema..."
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

echo "==> 7. Configurando Picom como compositor único y Rofi como selector de ventanas (Alt-Tab)..."
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
    "name = 'flameshot'",
    "class_g = 'vicinae'",
    "class_g = 'Xfce4-notifyd'",
    "class_g = 'xfce4-notifyd'",
    "window_type = 'notification'",
    "name = 'Notification'",
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

# Limpiar servicios residuales de Skippy-XD y Xfdashboard
systemctl --user stop xfdashboard.service skippy-xd.service 2>/dev/null || true
systemctl --user disable xfdashboard.service skippy-xd.service 2>/dev/null || true
rm -f "$HOME/.config/systemd/user/xfdashboard.service" "$HOME/.config/autostart/xfdashboard.desktop" \
      "$HOME/.config/systemd/user/skippy-xd.service" "$HOME/.config/autostart/skippy-xd.desktop" 2>/dev/null || true
killall -q xfdashboard xfdashboard.bin skippy-xd skippy-xd.bin 2>/dev/null || true

# Configuración de tema Rofi (Catppuccin Mocha / Orchis-Dark consistente con el sistema)
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
    width: 700px;
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

# Compilar watcher para liberar Alt y modo búsqueda ('/')
mkdir -p "$HOME/.local/src" "$HOME/.local/bin"
cat <<'EOF' > "$HOME/.local/src/rofi-alt-tab-watcher.c"
#include <X11/Xlib.h>
#include <X11/Xutil.h>
#include <X11/keysym.h>
#include <stdio.h>
#include <unistd.h>
#include <stdlib.h>
#include <string.h>
#include <sys/types.h>
#include <signal.h>

extern int XTestFakeKeyEvent(Display *dpy, unsigned int keycode, Bool is_press, unsigned long delay);

static int ignore_x_errors(Display *d, XErrorEvent *e) {
    (void)d; (void)e;
    return 0;
}

static int is_key_down(const char *keys, KeyCode code) {
    if (code == 0) return 0;
    return (keys[code / 8] & (1 << (code % 8))) != 0;
}

static Window find_rofi_window(Display *dpy, Window root) {
    Window parent, *children;
    unsigned int nchildren;
    Window result = 0;
    if (XQueryTree(dpy, root, &root, &parent, &children, &nchildren) == 0 || !children) {
        return 0;
    }
    for (unsigned int i = 0; i < nchildren; i++) {
        XClassHint hint;
        if (XGetClassHint(dpy, children[i], &hint)) {
            if ((hint.res_name && strcasecmp(hint.res_name, "rofi") == 0) ||
                (hint.res_class && strcasecmp(hint.res_class, "rofi") == 0)) {
                result = children[i];
                if (hint.res_name) XFree(hint.res_name);
                if (hint.res_class) XFree(hint.res_class);
                break;
            }
            if (hint.res_name) XFree(hint.res_name);
            if (hint.res_class) XFree(hint.res_class);
        }
    }
    if (children) XFree(children);
    return result;
}

static void send_return(Display *dpy, KeyCode ret_code) {
    XTestFakeKeyEvent(dpy, ret_code, True, CurrentTime);
    XTestFakeKeyEvent(dpy, ret_code, False, CurrentTime);
    XFlush(dpy);
}

int main(int argc, char *argv[]) {
    if (argc < 2) return 1;
    pid_t target_pid = (pid_t)atoi(argv[1]);

    Display *dpy = XOpenDisplay(NULL);
    if (!dpy) return 1;

    XSetErrorHandler(ignore_x_errors);

    Window root = DefaultRootWindow(dpy);
    KeyCode alt_l = XKeysymToKeycode(dpy, XK_Alt_L);
    KeyCode alt_r = XKeysymToKeycode(dpy, XK_Alt_R);
    KeyCode slash = XKeysymToKeycode(dpy, XK_slash);
    KeyCode question = XKeysymToKeycode(dpy, XK_question);
    KeyCode ret_code = XKeysymToKeycode(dpy, XK_Return);

    char keys[32];

    // 1. Esperar a que la ventana de Rofi se mapee
    Window rofi_win = 0;
    for (int i = 0; i < 300; i++) {
        if (kill(target_pid, 0) != 0) {
            XCloseDisplay(dpy);
            return 0;
        }

        XQueryKeymap(dpy, keys);
        if (is_key_down(keys, slash) || is_key_down(keys, question)) {
            // Modo búsqueda solicitado explícitamente con '/'
            XCloseDisplay(dpy);
            return 0;
        }

        rofi_win = find_rofi_window(dpy, root);
        if (rofi_win != 0) {
            break;
        }
        usleep(5000);
    }

    if (!rofi_win) {
        XCloseDisplay(dpy);
        return 0;
    }

    usleep(10000);

    // 2. Monitorear liberación de Alt o tecla '/'
    for (int i = 0; i < 4000; i++) {
        if (kill(target_pid, 0) != 0 || find_rofi_window(dpy, root) == 0) {
            break;
        }

        XQueryKeymap(dpy, keys);

        // Si se presiona '/' o '?', quedarse permanentemente en modo búsqueda
        if (is_key_down(keys, slash) || is_key_down(keys, question)) {
            break;
        }

        int alt_down = is_key_down(keys, alt_l) || is_key_down(keys, alt_r);
        if (!alt_down) {
            for (int p = 0; p < 4; p++) {
                if (kill(target_pid, 0) != 0 || find_rofi_window(dpy, root) == 0) {
                    break;
                }
                send_return(dpy, ret_code);
                usleep(20000);
            }
            break;
        }

        usleep(5000);
    }

    XCloseDisplay(dpy);
    return 0;
}
EOF
gcc -O2 "$HOME/.local/src/rofi-alt-tab-watcher.c" -lX11 -l:libXtst.so.6 -o "$HOME/.local/bin/rofi-alt-tab-watcher" 2>/dev/null || true
chmod +x "$HOME/.local/bin/rofi-alt-tab-watcher" 2>/dev/null || true

# Wrapper ejecutable para el selector de ventanas Rofi
mkdir -p "$HOME/.local/bin"
cat <<'EOF' > "$HOME/.local/bin/rofi-window"
#!/bin/sh
# Si se invoca con --search, -s o '/', permanece abierto en modo búsqueda difusa interactiva
if [ "${1:-}" = "--search" ] || [ "${1:-}" = "-s" ] || [ "${1:-}" = "/" ]; then
    shift
    exec rofi -show window \
        -matching fuzzy \
        -window-format "{w} · {c} · {t}" \
        -show-icons "$@"
fi

# Prevenir procesos residuales de watcher anteriores
killall -q rofi-alt-tab-watcher 2>/dev/null || true

# Lanzar selector de ventanas con fila 1 (MRU previa) preseleccionada
rofi -show window \
    -selected-row 1 \
    -matching fuzzy \
    -window-format "{w} · {c} · {t}" \
    -show-icons \
    -kb-element-next "Tab,Alt+Tab" \
    -kb-element-prev "ISO_Left_Tab,Alt+ISO_Left_Tab" "$@" &
ROFI_PID=$!

# Monitorear liberación de Alt o tecla '/' en segundo plano
if [ -x "$HOME/.local/bin/rofi-alt-tab-watcher" ]; then
    "$HOME/.local/bin/rofi-alt-tab-watcher" "$ROFI_PID" &
fi

wait "$ROFI_PID" 2>/dev/null || true
EOF
chmod +x "$HOME/.local/bin/rofi-window"
sudo ln -sf "$HOME/.local/bin/rofi-window" /usr/local/bin/rofi-window 2>/dev/null || true

# Comando estándar para Rofi Window Switcher
ROFI_WINDOW_CMD="$HOME/.local/bin/rofi-window"

# Desvincular switcher nativo de xfwm4 para ceder el control completo a Rofi
xfconf-query -c xfce4-keyboard-shortcuts -p "/xfwm4/custom/<Alt>Tab" -n -t string -s "none" 2>/dev/null || \
xfconf-query -c xfce4-keyboard-shortcuts -p "/xfwm4/custom/<Alt>Tab" -s "none" 2>/dev/null || true

xfconf-query -c xfce4-keyboard-shortcuts -p "/xfwm4/custom/<Alt><Shift>Tab" -n -t string -s "none" 2>/dev/null || \
xfconf-query -c xfce4-keyboard-shortcuts -p "/xfwm4/custom/<Alt><Shift>Tab" -s "none" 2>/dev/null || true

xfconf-query -c xfce4-keyboard-shortcuts -p "/xfwm4/custom/<Super>Tab" -n -t string -s "none" 2>/dev/null || \
xfconf-query -c xfce4-keyboard-shortcuts -p "/xfwm4/custom/<Super>Tab" -s "none" 2>/dev/null || true

# Configurar Rofi en atajos de teclado globales
xfconf-query -c xfce4-keyboard-shortcuts -p "/commands/custom/<Alt>Tab" -n -t string -s "$ROFI_WINDOW_CMD" 2>/dev/null || \
xfconf-query -c xfce4-keyboard-shortcuts -p "/commands/custom/<Alt>Tab" -s "$ROFI_WINDOW_CMD" 2>/dev/null || true

xfconf-query -c xfce4-keyboard-shortcuts -p "/commands/custom/<Alt><Shift>Tab" -n -t string -s "$ROFI_WINDOW_CMD" 2>/dev/null || \
xfconf-query -c xfce4-keyboard-shortcuts -p "/commands/custom/<Alt><Shift>Tab" -s "$ROFI_WINDOW_CMD" 2>/dev/null || true

xfconf-query -c xfce4-keyboard-shortcuts -p "/commands/custom/<Super>Tab" -n -t string -s "$ROFI_WINDOW_CMD" 2>/dev/null || \
xfconf-query -c xfce4-keyboard-shortcuts -p "/commands/custom/<Super>Tab" -s "$ROFI_WINDOW_CMD" 2>/dev/null || true

xfconf-query -c xfce4-keyboard-shortcuts -p "/commands/custom/<Alt>slash" -n -t string -s "$ROFI_WINDOW_CMD --search" 2>/dev/null || \
xfconf-query -c xfce4-keyboard-shortcuts -p "/commands/custom/<Alt>slash" -s "$ROFI_WINDOW_CMD --search" 2>/dev/null || true

# Iniciar o reiniciar compositor en sesión activa
killall -q picom 2>/dev/null || true
xfwm4 --replace >/dev/null 2>&1 &
sleep 1
(picom -b --config "$HOME/.config/picom/picom.conf" >/dev/null 2>&1 &) || true

echo "==> 8. Configurando xwallpaper y desacoplando xfdesktop de la sesión..."
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

# 1. Si se pasa un argumento, verificarlo y guardarlo como wallpaper preferido
if [ -n "${1:-}" ] && is_valid_image "$1"; then
    WALLPAPER="$(realpath "$1")"
    echo "$WALLPAPER" > "$CONFIG_FILE"
fi

# 2. Si hay wallpaper guardado previamente, usarlo
if [ -z "$WALLPAPER" ] && [ -f "$CONFIG_FILE" ]; then
    SAVED=$(cat "$CONFIG_FILE" 2>/dev/null || true)
    if is_valid_image "$SAVED"; then
        WALLPAPER="$SAVED"
    fi
fi

# 3. Buscar en fondos estándar del sistema si aún no hay wallpaper
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

# Distribución de teclado nativa (XKB: US/ES) y alternancia con Alt+Shift
xfconf-query -c keyboard-layout -p /Default/XkbDisable -n -t bool -s false 2>/dev/null || \
xfconf-query -c keyboard-layout -p /Default/XkbDisable -s false 2>/dev/null || true

xfconf-query -c keyboard-layout -p /Default/XkbLayout -n -t string -s "us,es" 2>/dev/null || \
xfconf-query -c keyboard-layout -p /Default/XkbLayout -s "us,es" 2>/dev/null || true

xfconf-query -c keyboard-layout -p /Default/XkbVariant -n -t string -s "," 2>/dev/null || \
xfconf-query -c keyboard-layout -p /Default/XkbVariant -s "," 2>/dev/null || true

xfconf-query -c keyboard-layout -p "/Default/XkbOptions/Group" -n -t string -s "grp:alt_shift_toggle" 2>/dev/null || \
xfconf-query -c keyboard-layout -p "/Default/XkbOptions/Group" -s "grp:alt_shift_toggle" 2>/dev/null || true

setxkbmap -layout "us,es" -variant "," -option "grp:alt_shift_toggle" 2>/dev/null || true

# Eliminar script residual previo de alternancia y atajo no estándar
rm -f "$HOME/.local/bin/toggle-layout.sh" 2>/dev/null || true
xfconf-query -c xfce4-keyboard-shortcuts -p "/commands/custom/<Super><Alt>space" -r 2>/dev/null || true

echo "==> 10. Integrando captura de pantalla (Flameshot) y OCR (Tesseract)..."
mkdir -p "$HOME/.local/bin"
mkdir -p "$HOME/.config/flameshot"

# Configuración de Flameshot: desactivar indicador de tamaño/geometría y lupa
# para evitar que tapen selecciones pequeñas de texto durante OCR
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
if [ -z "$INFO" ]; then
    exit 0
fi

X=$(echo "$INFO" | awk '/Absolute upper-left X:/ {print $4}')
Y=$(echo "$INFO" | awk '/Absolute upper-left Y:/ {print $4}')
W=$(echo "$INFO" | awk '/Width:/ {print $2}')
H=$(echo "$INFO" | awk '/Height:/ {print $2}')

if [ -z "$W" ] || [ -z "$H" ] || [ -z "$X" ] || [ -z "$Y" ]; then
    exit 0
fi

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

if ! flameshot gui --raw > "$TMP_IMG" 2>/dev/null; then
    exit 0
fi

if [ ! -s "$TMP_IMG" ]; then
    exit 0
fi

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

chmod +x "$HOME/.local/bin/cap-area" \
         "$HOME/.local/bin/cap-repeat" \
         "$HOME/.local/bin/cap-window" \
         "$HOME/.local/bin/cap-fullscreen" \
         "$HOME/.local/bin/cap-ocr"

sudo ln -sf "$HOME/.local/bin/cap-area" /usr/local/bin/cap-area 2>/dev/null || true
sudo ln -sf "$HOME/.local/bin/cap-repeat" /usr/local/bin/cap-repeat 2>/dev/null || true
sudo ln -sf "$HOME/.local/bin/cap-window" /usr/local/bin/cap-window 2>/dev/null || true
sudo ln -sf "$HOME/.local/bin/cap-fullscreen" /usr/local/bin/cap-fullscreen 2>/dev/null || true
sudo ln -sf "$HOME/.local/bin/cap-ocr" /usr/local/bin/cap-ocr 2>/dev/null || true

# Configurar atajos de captura y OCR en Xfce
xfconf-query -c xfce4-keyboard-shortcuts -p "/commands/custom/<Primary><Alt><Super>1" -n -t string -s "$HOME/.local/bin/cap-area" 2>/dev/null || \
xfconf-query -c xfce4-keyboard-shortcuts -p "/commands/custom/<Primary><Alt><Super>1" -s "$HOME/.local/bin/cap-area" 2>/dev/null || true

xfconf-query -c xfce4-keyboard-shortcuts -p "/commands/custom/<Primary><Alt><Super>2" -n -t string -s "$HOME/.local/bin/cap-repeat" 2>/dev/null || \
xfconf-query -c xfce4-keyboard-shortcuts -p "/commands/custom/<Primary><Alt><Super>2" -s "$HOME/.local/bin/cap-repeat" 2>/dev/null || true

xfconf-query -c xfce4-keyboard-shortcuts -p "/commands/custom/<Primary><Alt><Super>3" -n -t string -s "$HOME/.local/bin/cap-window" 2>/dev/null || \
xfconf-query -c xfce4-keyboard-shortcuts -p "/commands/custom/<Primary><Alt><Super>3" -s "$HOME/.local/bin/cap-window" 2>/dev/null || true

xfconf-query -c xfce4-keyboard-shortcuts -p "/commands/custom/<Primary><Alt><Super>4" -n -t string -s "$HOME/.local/bin/cap-fullscreen" 2>/dev/null || \
xfconf-query -c xfce4-keyboard-shortcuts -p "/commands/custom/<Primary><Alt><Super>4" -s "$HOME/.local/bin/cap-fullscreen" 2>/dev/null || true

xfconf-query -c xfce4-keyboard-shortcuts -p "/commands/custom/<Primary><Alt><Super>0" -n -t string -s "$HOME/.local/bin/cap-ocr" 2>/dev/null || \
xfconf-query -c xfce4-keyboard-shortcuts -p "/commands/custom/<Primary><Alt><Super>0" -s "$HOME/.local/bin/cap-ocr" 2>/dev/null || true

echo "==> 11. Integrando Nemo como gestor de archivos predeterminado..."
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
#XfceNotifyWindow progressbar {
    min-height: 6px;
    border-radius: 4px;
}
#XfceNotifyWindow progressbar progress {
    background-image: none;
    background-color: #89b4fa;
    border: none;
    border-radius: 4px;
}
#XfceNotifyWindow progressbar trough {
    background-image: none;
    background-color: #313244;
    border: 1px solid #45475a;
    border-radius: 4px;
}
EOF

# Sincronizar estilo en el directorio del tema a nivel del sistema
if [ -d "/usr/share/themes/Orchis-Dark" ]; then
    sudo mkdir -p /usr/share/themes/Orchis-Dark/xfce-notify-4.0 2>/dev/null || true
    sudo cp "$HOME/.themes/Orchis-Dark/xfce-notify-4.0/gtk.css" /usr/share/themes/Orchis-Dark/xfce-notify-4.0/gtk.css 2>/dev/null || true
fi

xfconf-query -c xfce4-notifyd -p /theme -s "Orchis-Dark" 2>/dev/null || \
    xfconf-query -c xfce4-notifyd -p /theme -n -t string -s "Orchis-Dark" 2>/dev/null || true

xfconf-query -c xfce4-notifyd -p /notify-location -s "top-right" 2>/dev/null || \
    xfconf-query -c xfce4-notifyd -p /notify-location -s 2 2>/dev/null || \
    xfconf-query -c xfce4-notifyd -p /notify-location -n -t string -s "top-right" 2>/dev/null || true

xfconf-query -c xfce4-notifyd -p /initial-opacity -s 0.95 2>/dev/null || \
    xfconf-query -c xfce4-notifyd -p /initial-opacity -n -t double -s 0.95 2>/dev/null || true

xfconf-query -c xfce4-notifyd -p /do-fadeout -s true 2>/dev/null || \
    xfconf-query -c xfce4-notifyd -p /do-fadeout -n -t bool -s true 2>/dev/null || true

xfconf-query -c xfce4-notifyd -p /do-not-disturb -s false 2>/dev/null || \
    xfconf-query -c xfce4-notifyd -p /do-not-disturb -n -t bool -s false 2>/dev/null || true

xfconf-query -c xfce4-notifyd -p /notification-log -s true 2>/dev/null || \
    xfconf-query -c xfce4-notifyd -p /notification-log -n -t bool -s true 2>/dev/null || true

xfconf-query -c xfce4-notifyd -p /log-level -s "always" 2>/dev/null || \
    xfconf-query -c xfce4-notifyd -p /log-level -n -t string -s "always" 2>/dev/null || true

xfconf-query -c xfce4-notifyd -p /log-level-apps -s "all" 2>/dev/null || \
    xfconf-query -c xfce4-notifyd -p /log-level-apps -n -t string -s "all" 2>/dev/null || true

xfconf-query -c xfce4-notifyd -p /log-max-size-enabled -s true 2>/dev/null || \
    xfconf-query -c xfce4-notifyd -p /log-max-size-enabled -n -t bool -s true 2>/dev/null || true

xfconf-query -c xfce4-notifyd -p /log-max-size -s 500 2>/dev/null || \
    xfconf-query -c xfce4-notifyd -p /log-max-size -n -t int -s 500 2>/dev/null || true

xfconf-query -c xfce4-notifyd -p /date-time-custom-format -s "%a %H:%M:%S" 2>/dev/null || \
    xfconf-query -c xfce4-notifyd -p /date-time-custom-format -n -t string -s "%a %H:%M:%S" 2>/dev/null || true

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

NOTIFY_PLUGIN=$(xfconf-query -c xfce4-panel -p /plugins -l 2>/dev/null | grep -E '^/plugins/plugin-[0-9]+$' | while read -r p; do
    [ "$(xfconf-query -c xfce4-panel -p "$p" 2>/dev/null || true)" = "notification-plugin" ] && echo "$p" && break
done)

# Si el plugin no existe en el panel, crearlo e incorporarlo a panel-1
if [ -z "$NOTIFY_PLUGIN" ]; then
    MAX_ID=$(xfconf-query -c xfce4-panel -p /plugins -l 2>/dev/null | grep -oE '[0-9]+' | sort -n | tail -1 || echo 0)
    NEW_ID=$(( MAX_ID + 1 ))
    NOTIFY_PLUGIN="/plugins/plugin-$NEW_ID"
    xfconf-query -c xfce4-panel -p "$NOTIFY_PLUGIN" -n -t string -s "notification-plugin" 2>/dev/null || true

    EXISTING_IDS=$(xfconf-query -c xfce4-panel -p /panels/panel-1/plugin-ids 2>/dev/null | grep -E '^[0-9]+$' || true)
    PANEL_ARGS=()
    for id in $EXISTING_IDS; do
        PANEL_ARGS+=(-t int -s "$id")
    done
    PANEL_ARGS+=(-t int -s "$NEW_ID")
    xfconf-query -c xfce4-panel -p /panels/panel-1/plugin-ids -n -a "${PANEL_ARGS[@]}" 2>/dev/null || \
        xfconf-query -c xfce4-panel -p /panels/panel-1/plugin-ids -a "${PANEL_ARGS[@]}" 2>/dev/null || true
fi

xfconf-query -c xfce4-panel -p "$NOTIFY_PLUGIN/show-in-menu" -n -t string -s "show-all" 2>/dev/null || \
    xfconf-query -c xfce4-panel -p "$NOTIFY_PLUGIN/show-in-menu" -s "show-all" 2>/dev/null || true
xfconf-query -c xfce4-panel -p "$NOTIFY_PLUGIN/hide-on-read" -n -t bool -s false 2>/dev/null || \
    xfconf-query -c xfce4-panel -p "$NOTIFY_PLUGIN/hide-on-read" -s false 2>/dev/null || true
xfconf-query -c xfce4-panel -p "$NOTIFY_PLUGIN/show-only-today" -n -t bool -s false 2>/dev/null || \
    xfconf-query -c xfce4-panel -p "$NOTIFY_PLUGIN/show-only-today" -s false 2>/dev/null || true
xfconf-query -c xfce4-panel -p "$NOTIFY_PLUGIN/log-display-limit" -n -t int -s 25 2>/dev/null || \
    xfconf-query -c xfce4-panel -p "$NOTIFY_PLUGIN/log-display-limit" -s 25 2>/dev/null || true

# Plugin de distribución de teclado (Keyboard Layout - xkb) en panel-1
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
    for id in $EXISTING_IDS; do
        PANEL_ARGS+=(-t int -s "$id")
    done
    PANEL_ARGS+=(-t int -s "$NEW_ID")
    xfconf-query -c xfce4-panel -p /panels/panel-1/plugin-ids -n -a "${PANEL_ARGS[@]}" 2>/dev/null || \
        xfconf-query -c xfce4-panel -p /panels/panel-1/plugin-ids -a "${PANEL_ARGS[@]}" 2>/dev/null || true
fi

xfconf-query -c xfce4-panel -p "$XKB_PLUGIN/display-type" -n -t int -s 0 2>/dev/null || \
    xfconf-query -c xfce4-panel -p "$XKB_PLUGIN/display-type" -s 0 2>/dev/null || true
xfconf-query -c xfce4-panel -p "$XKB_PLUGIN/display-name" -n -t int -s 0 2>/dev/null || \
    xfconf-query -c xfce4-panel -p "$XKB_PLUGIN/display-name" -s 0 2>/dev/null || true
xfconf-query -c xfce4-panel -p "$XKB_PLUGIN/display-scale" -n -t int -s 100 2>/dev/null || \
    xfconf-query -c xfce4-panel -p "$XKB_PLUGIN/display-scale" -s 100 2>/dev/null || true
xfconf-query -c xfce4-panel -p "$XKB_PLUGIN/show-notifications" -n -t bool -s false 2>/dev/null || \
    xfconf-query -c xfce4-panel -p "$XKB_PLUGIN/show-notifications" -s false 2>/dev/null || true

xfconf-query -c xfce4-panel -p /panels -a -t int -s 1 2>/dev/null || true
xfconf-query -c xfce4-panel -p /panels/panel-2 -r -R 2>/dev/null || true
xfce4-panel -r 2>/dev/null || true
killall -q xfce4-notifyd 2>/dev/null || true

echo "==> 13. Optimizaciones genéricas de red en arranque..."
sudo systemctl disable NetworkManager-wait-online.service 2>/dev/null || true

echo "==> Configuración completada. Reinicia el entorno para aplicar los cambios de sesión con: sudo reboot"
