# `rcmd` — Instant Window Switcher & App Launcher for Linux

A lightweight, ultra-responsive macOS `rcmd`-style instant application switcher for Linux desktop environments.

Allows you to switch to or launch applications instantly using dedicated **Right-Alt** key combinations (e.g. `Right-Alt + C` focuses Google Chrome), eliminating Alt-Tab cognitive friction.

---

## 1. How It Works

`rcmd` features an agnostic multi-backend architecture that detects the running display server and desktop compositor automatically:

```
                  ┌────────────────────────┐
                  │      rcmd Dispatcher   │
                  │   (Right-Alt + Key)    │
                  └───────────┬────────────┘
                              │
          ┌───────────────────┼───────────────────┐
          │                   │                   │
┌─────────▼────────┐ ┌────────▼────────┐ ┌────────▼────────┐
│   GNOME Shell    │ │ Hyprland / Sway │ │  X11 / KWin /   │
│  (D-Bus Native   │ │   (Wayland IPC  │ │   Openbox /     │
│    Extension)    │ │   dispatchers)  │ │  XFCE (wmctrl)  │
└──────────────────┘ └─────────────────┘ └─────────────────┘
```

### Supported Desktop Backends

1. **GNOME Shell (Wayland & X11)**:
   - Primary backend: Direct D-Bus IPC with the bundled native GNOME Shell extension [`rcmd-shell@arturonavax.dev`](../gnome/extensions/rcmd-shell@arturonavax.dev/extension.js) exporting `dev.arturonavax.rcmd`.
   - Secondary fallback: `org.gnome.Shell.Extensions.RunOrRaise`.
2. **Hyprland (Wayland)**:
   - Dispatches window focus via `hyprctl dispatch focuswindow` and `hyprctl clients -j`.
3. **Sway (Wayland)**:
   - Dispatches window focus via `swaymsg [app_id=...] focus` and IPC trees.
4. **KDE KWin (Wayland & X11)**:
   - Dispatches via KWin scripting over D-Bus (`org.kde.KWin`).
5. **Universal X11 Fallback (XFCE, i3, bspwm, Openbox)**:
   - Control via `wmctrl` and `xdotool`.

---

## 2. Default Keybindings

Configured via `keyd` or your window manager keyboard shortcuts:

| Shortcut | Target Application | Matching Mode |
| :--- | :--- | :--- |
| `Right-Alt + C` | Google Chrome / Chromium / Brave | WM_CLASS / app_id |
| `Right-Alt + E` | Ghostty Terminal / Herdr Workspace | WM_CLASS / app_id |
| `Right-Alt + V` | Neovim / VSCode / Cursor IDE | WM_CLASS / app_id |
| `Right-Alt + S` | Spotify / Slack | WM_CLASS / app_id |
| `Right-Alt + W` | WhatsApp Web / Web App | Window title regex |
| `Right-Alt + F` | File Manager (Thunar, Nautilus) | WM_CLASS / app_id |

---

## 3. Configuration & Customization (`~/.config/rcmd/rcmd.conf`)

`rcmd` preserves your local configuration file (`smart-preserve`). To add or customize shortcuts, edit `~/.config/rcmd/rcmd.conf`:

```ini
# Format: KEY=APP_IDENTIFIER [MODE]
# Modes:
#   class: Matches window WM_CLASS or Wayland app_id (default)
#   title: Matches regex pattern against window title
#   launch: Command to execute if application is not currently running

c=google-chrome-stable class launch:google-chrome-stable
e=com.mitchellh.ghostty class launch:ghostty
v=cursor class launch:cursor
w=WhatsApp title launch:google-chrome-stable --app=https://web.whatsapp.com
s=spotify class launch:spotify
f=org.gnome.Nautilus class launch:nautilus
```

### Installation

`rcmd` is automatically installed when selecting the `osconfig` subsystem:

```sh
./install.sh osconfig
```

Or manually deployed via sync:

```sh
anx sync osconfig
```
