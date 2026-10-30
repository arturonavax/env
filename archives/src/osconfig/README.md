# Operating System & Desktop Environment Configurations (`src/osconfig`)

This directory centralizes operating system tweaks, desktop environment configurations, power-user keyboard shortcuts, and graphical environment enhancements.

---

## 1. Directory Structure

```text
src/osconfig/
├── linux_osconfig.sh       # Core Linux limits, sysctl tuning, font caches, and rcmd shortcuts
├── xfce/                   # XFCE Pro power-user desktop workflow & keyboard shortcuts
│   ├── setup_xfce_pro.sh       # Complete power-user XFCE setup (custom window switchers, OCR, cap)
│   ├── reset_base_setup_xfce_pro.sh # Reverts XFCE settings back to clean baseline
│   └── test_setup_xfce_pro.sh  # Automated verification suite for XFCE configuration
└── macos/                  # macOS developer optimizations and system tweaks
    ├── macos_osconfig.sh       # Master macOS configuration script
    ├── macos_safari.sh         # Safari developer menu & WebKit animation tweaks
    ├── macos_spaces.sh         # Mission Control spaces animation speedup
    └── macos_sudo_touchid.sh   # Enables Touch ID for sudo authentication
```

---

## 2. Linux Core (`linux_osconfig.sh`)

Applies baseline performance and stability configurations across Linux distributions:

- **System Limits (`limits.conf`)**: Increases open file descriptors (`nofile` to 65,536) and inotify watchers (`fs.inotify.max_user_watches = 524288`) for large codebases and LSP servers.
- **Font Cache Optimization**: Runs `fc-cache -fv` to register installed Nerd Fonts.
- **GTK Renderer Configuration**: Configures `GSK_RENDERER=gl` to prevent Wayland surface stalls on GTK 4.14 compositors.
- **Keyboard Repeat Rate**: Accelerates key repeat rates via `xset r rate 200 30` to eliminate editing latency.

---

## 3. XFCE Pro Workflow (`src/osconfig/xfce/`)

Transformative power-user setup for Linux workstations running XFCE:

- **Instant Application Switching**: Integrates `rcmd` and rofi window switcher.
- **Screen Capture & OCR Suite**:
  - `Super + Shift + 4` (or `cap-area`): Interactive rectangular screenshot.
  - `Super + Shift + 5` (or `cap-window`): Single window snapshot.
  - `Super + Shift + C` (or `cap-ocr`): Screen text extraction to clipboard via Tesseract OCR.
- **Session & Screen Lock Guard**: Hardened unlock script preventing black screens or frozen X11 sessions.
- **Baseline Reset (`reset_base_setup_xfce_pro.sh`)**: Cleanly restores factory XFCE settings, shortcuts, and color gamma.

---

## 4. macOS Developer Tweaks (`src/osconfig/macos/`)

Optimizes macOS for development velocity:

- **Touch ID for Sudo (`macos_sudo_touchid.sh`)**: Enables biometric authentication in terminal `sudo` prompts (`pam_tid.so`).
- **Mission Control Animation Speed (`macos_spaces.sh`)**: Eliminates the slow default slide delay when switching virtual desktops.
- **Safari Developer Tools (`macos_safari.sh`)**: Enables Developer menu, Web Inspector, and responsive design mode by default.
