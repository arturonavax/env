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
    xfce4-goodies xfce4-whiskermenu-plugin xfce4-power-manager xfce4-screenshooter \
    xcape xdotool brightnessctl pavucontrol network-manager-gnome \
    pipewire pipewire-pulse wireplumber \
    xdg-desktop-portal xdg-desktop-portal-gtk \
    tumbler ffmpegthumbnailer poppler-data tumbler-plugins-extra webp-pixbuf-loader \
    gvfs-backends gvfs-fuse policykit-1-gnome \
    fonts-inter fonts-jetbrains-mono \
    plank dconf-cli libglib2.0-bin libglib2.0-dev-bin libnotify-bin \
    curl wget git jq unzip

echo "==> Removiendo Remmina (paquetes, applet de inicio y accesos en dock)..."
sudo apt purge -y remmina remmina-plugin-rdp remmina-plugin-vnc remmina-plugin-secret remmina-common 2>/dev/null || true
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

# Configurar LightDM GTK Greeter (sin descarga de fondos rotos, con hora AM/PM y soporte de fondo de usuario)
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

# Limpieza del fondo de pantalla anterior roto si existiese
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

systemctl --user enable --now pipewire pipewire-pulse wireplumber 2>/dev/null || true

# Configurar compositor nativo de xfwm4 con directivas visuales completas estándar
xfconf-query -c xfwm4 -p /general/use_compositing -n -t bool -s true 2>/dev/null || xfconf-query -c xfwm4 -p /general/use_compositing -s true 2>/dev/null || true
xfconf-query -c xfwm4 -p /general/show_frame_shadow -n -t bool -s true 2>/dev/null || xfconf-query -c xfwm4 -p /general/show_frame_shadow -s true 2>/dev/null || true
xfconf-query -c xfwm4 -p /general/show_popup_shadow -n -t bool -s true 2>/dev/null || xfconf-query -c xfwm4 -p /general/show_popup_shadow -s true 2>/dev/null || true
xfconf-query -c xfwm4 -p /general/show_dock_shadow -n -t bool -s true 2>/dev/null || xfconf-query -c xfwm4 -p /general/show_dock_shadow -s true 2>/dev/null || true
xfconf-query -c xfwm4 -p /general/cycle_preview -n -t bool -s true 2>/dev/null || xfconf-query -c xfwm4 -p /general/cycle_preview -s true 2>/dev/null || true
xfconf-query -c xfwm4 -p /general/cycle_tabwin_mode -n -t int -s 1 2>/dev/null || xfconf-query -c xfwm4 -p /general/cycle_tabwin_mode -s 1 2>/dev/null || true
xfconf-query -c xfwm4 -p /general/vblank_mode -n -t string -s "auto" 2>/dev/null || xfconf-query -c xfwm4 -p /general/vblank_mode -s "auto" 2>/dev/null || true

# Asegurar que picom no interfiera con el compositor de xfwm4
rm -f ~/.config/autostart/picom.desktop 2>/dev/null || true
killall picom 2>/dev/null || true

# Autostart: Plank
cat <<'EOF' > ~/.config/autostart/plank.desktop
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
cat <<EOF > ~/.config/autostart/polkit-gnome.desktop
[Desktop Entry]
Type=Application
Exec=$POLKIT_BIN
Hidden=false
NoDisplay=false
X-GNOME-Autostart-enabled=true
Name=PolicyKit Authentication Agent
EOF

# Autostart: Xcape (mapea tecla Windows / Super a Whisker Menu vía Alt+F1 sin colisiones)
cat <<'EOF' > ~/.config/autostart/xcape.desktop
[Desktop Entry]
Type=Application
Exec=sh -c "killall xcape 2>/dev/null; sleep 1; xcape -e 'Super_L=Alt_L|F1;Super_R=Alt_L|F1'"
Hidden=false
NoDisplay=false
X-GNOME-Autostart-enabled=true
Name=Xcape Super Key Mapper
Comment=Mapea la tecla Windows / Super al menu Whisker
EOF

# Iniciar xcape de inmediato si estamos en sesión gráfica
killall xcape 2>/dev/null || true
(xcape -e 'Super_L=Alt_L|F1;Super_R=Alt_L|F1' >/dev/null 2>&1 &) || true

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

# Atajos para abrir Whisker Menu con tecla Windows / Super y Alt+F1
xfconf-query -c xfce4-keyboard-shortcuts -p "/commands/custom/<Alt>F1" -n -t string -s "xfce4-popup-whiskermenu" 2>/dev/null || \
xfconf-query -c xfce4-keyboard-shortcuts -p "/commands/custom/<Alt>F1" -s "xfce4-popup-whiskermenu" 2>/dev/null || true

# Limpiar bindings directos de Super_L y Super_R en custom shortcuts para que xcape traduzca el tap sin doble disparo
xfconf-query -c xfce4-keyboard-shortcuts -p "/commands/custom/Super_L" -r 2>/dev/null || true
xfconf-query -c xfce4-keyboard-shortcuts -p "/commands/custom/Super_R" -r 2>/dev/null || true

echo "==> 6. Configurando Xfce Panel (Whisker Menu, Reloj AM/PM con segundos)..."

# Identificar o configurar Whisker Menu en el panel
WHISKER_PLUGIN=$(xfconf-query -c xfce4-panel -p /plugins -l 2>/dev/null | grep -E '^/plugins/plugin-[0-9]+$' | while read -r p; do
    name=$(xfconf-query -c xfce4-panel -p "$p" 2>/dev/null || true)
    if [ "$name" = "whiskermenu" ]; then
        echo "$p"
        break
    fi
done)

if [ -z "$WHISKER_PLUGIN" ]; then
    # Si applicationsmenu está en el panel, reemplazarlo por whiskermenu
    APP_PLUGIN=$(xfconf-query -c xfce4-panel -p /plugins -l 2>/dev/null | grep -E '^/plugins/plugin-[0-9]+$' | while read -r p; do
        name=$(xfconf-query -c xfce4-panel -p "$p" 2>/dev/null || true)
        if [ "$name" = "applicationsmenu" ]; then
            echo "$p"
            break
        fi
    done)
    if [ -n "$APP_PLUGIN" ]; then
        xfconf-query -c xfce4-panel -p "$APP_PLUGIN" -s "whiskermenu" 2>/dev/null || true
        WHISKER_PLUGIN="$APP_PLUGIN"
    else
        # Si no existe, configurar plugin-1 como whiskermenu
        xfconf-query -c xfce4-panel -p /plugins/plugin-1 -n -t string -s "whiskermenu" 2>/dev/null || \
        xfconf-query -c xfce4-panel -p /plugins/plugin-1 -s "whiskermenu" 2>/dev/null || true
        WHISKER_PLUGIN="/plugins/plugin-1"
    fi
fi

# Configurar apariencia del botón de Whisker Menu
xfconf-query -c xfce4-panel -p "$WHISKER_PLUGIN/button-icon" -n -t string -s "org.xfce.panel.whiskermenu" 2>/dev/null || \
xfconf-query -c xfce4-panel -p "$WHISKER_PLUGIN/button-icon" -s "org.xfce.panel.whiskermenu" 2>/dev/null || true

xfconf-query -c xfce4-panel -p "$WHISKER_PLUGIN/button-title" -n -t string -s "" 2>/dev/null || \
xfconf-query -c xfce4-panel -p "$WHISKER_PLUGIN/button-title" -s "" 2>/dev/null || true

xfconf-query -c xfce4-panel -p "$WHISKER_PLUGIN/show-button-icon" -n -t bool -s true 2>/dev/null || \
xfconf-query -c xfce4-panel -p "$WHISKER_PLUGIN/show-button-icon" -s true 2>/dev/null || true

xfconf-query -c xfce4-panel -p "$WHISKER_PLUGIN/show-button-title" -n -t bool -s false 2>/dev/null || \
xfconf-query -c xfce4-panel -p "$WHISKER_PLUGIN/show-button-title" -s false 2>/dev/null || true

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
echo "    - Whisker Menu activado y asignado a la tecla Windows / Super."
echo "    - Reloj configurado en formato 12 horas (AM/PM) con segundos."
echo "    - Fondos de pantalla del usuario preservados sin descargas externas."
echo "    - Remmina removido completamente del sistema y del dock."
echo "    Para aplicar el cambio de gestor de inicio LightDM por completo, reinicia con: sudo reboot"
