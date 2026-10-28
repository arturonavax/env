#!/usr/bin/env bash
# ==============================================================================
# Quota Primer & Auto-Wake Daemon Uninstaller for Linux & Xfce
# ==============================================================================
# Run:
#   ./src/remotes/uninstall_quota_daemon_xfce.sh [--purge]
# Or remotely:
#   curl -fsSL https://raw.githubusercontent.com/arturonavax/env/main/src/remotes/uninstall_quota_daemon_xfce.sh | bash
# ==============================================================================

set -euo pipefail

if [ "${BASH_SOURCE[0]}" != "$0" ]; then
    echo "ERROR: Do not source this script. Run: bash $0 or ./${0##*/}" >&2
    return 1 2>/dev/null || exit 1
fi

if [ "$EUID" -eq 0 ]; then
    echo "ERROR: Run this script as a normal user (it will prompt for sudo when necessary)." >&2
    exit 1
fi

BIN_DIR="$HOME/.local/bin"
CONFIG_DIR="$HOME/.config/quota-primer"
STATE_DIR="$HOME/.local/state/quota_primer"
SYSTEMD_USER_DIR="$HOME/.config/systemd/user"
DAEMON_SCRIPT="$BIN_DIR/quota-ctl"
SUDOERS_FILE="/etc/sudoers.d/quota-primer-rtcwake"

PURGE=false
for arg in "$@"; do
    case "$arg" in
        -p|--purge)
            PURGE=true
            ;;
        -h|--help)
            echo "Usage: $0 [--purge]"
            echo "  --purge   Remove configuration (~/.config/quota-primer) and state files as well."
            exit 0
            ;;
    esac
done

echo "==> 1. Stopping and disabling systemd user services..."
if command -v systemctl >/dev/null 2>&1; then
    systemctl --user stop quota-monitor.service quota-primer.timer quota-primer.service 2>/dev/null || true
    systemctl --user disable quota-monitor.service quota-primer.timer quota-primer.service 2>/dev/null || true
    rm -f "$SYSTEMD_USER_DIR/quota-monitor.service" \
          "$SYSTEMD_USER_DIR/quota-primer.service" \
          "$SYSTEMD_USER_DIR/quota-primer.timer"
    systemctl --user daemon-reload 2>/dev/null || true
fi

echo "==> 2. Disabling any active hardware RTC alarms..."
if command -v rtcwake >/dev/null 2>&1; then
    sudo -n rtcwake -m disable 2>/dev/null || sudo rtcwake -m disable 2>/dev/null || true
fi

echo "==> 3. Removing sudoers rule for rtcwake..."
if [ -f "$SUDOERS_FILE" ]; then
    sudo -n rm -f "$SUDOERS_FILE" 2>/dev/null || sudo rm -f "$SUDOERS_FILE" 2>/dev/null || true
fi

echo "==> 4. Removing daemon binary..."
rm -f "$DAEMON_SCRIPT"

if [ "$PURGE" = true ]; then
    echo "==> 5. Purging configuration and history directories..."
    rm -rf "$CONFIG_DIR" "$STATE_DIR"
else
    echo "==> 5. Preserving configuration in $CONFIG_DIR and state in $STATE_DIR."
    echo "    (Run with --purge to remove configuration completely)."
fi

echo "==> Quota Primer successfully uninstalled."
