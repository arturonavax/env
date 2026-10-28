# Workstation-Specific Hacks & Workarounds (`src/hacks`)

This directory centralizes specialized hardware, desktop environment, and quota automation scripts that apply to specific machines, GPU profiles, or situational workflows. They are kept separated from the general development environment installer.

---

## 1. `install_quota_daemon_xfce.sh` / `uninstall_quota_daemon_xfce.sh`
- **Purpose**: RTC-backed intelligent primer and auto-wake quota manager for rolling quota windows (e.g. Google Gemini / Antigravity CLI 5-hour cycles).
- **Target Setup**: Linux systems with `systemd` and RTC wakealarm capability (e.g. `/sys/class/rtc/rtc0/wakealarm`).
- **Features**:
  - Live usage monitoring (`usage_data`).
  - Automated timed suspend & resume cycles.
  - Non-destructive uninstaller with `--purge` flag.

## 2. `activate_scroll_lock.sh`
- **Purpose**: Hardware LED controller for keyboards where Scroll Lock key controls the backlight (e.g., Cooler Master Devastator and similar gaming keyboards under Linux).
- **Target Setup**: Linux X11 systems via `xset led named "Scroll Lock"`.
