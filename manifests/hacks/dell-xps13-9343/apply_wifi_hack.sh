#!/usr/bin/env bash
# ==============================================================================
# Dell XPS 13 9343 Broadcom BCM4352 (Dell Wireless 1560) Driver Optimization
# ==============================================================================
# Resolves:
#   1. Missing kernel module 'wl' after system updates or package purges.
#   2. Conflicting 'bcma' driver preventing 802.11ac hardware initialization.
#   3. NetworkManager interface registration and automated reconnect.
# ==============================================================================

set -euo pipefail

echo "==> [Dell XPS 13 9343] Verifying Broadcom BCM4352 Wi-Fi Driver..."

# 0. Unmask packagekit if masked to prevent D-Bus GDBus noise
sudo systemctl unmask packagekit.service 2>/dev/null || true

# 1. Check if Broadcom 4352 PCI hardware is present
if ! lspci -d 14e4:43b1 >/dev/null 2>&1 && ! lspci 2>/dev/null | grep -qi "BCM4352"; then
    echo "    ℹ No Broadcom BCM4352 hardware detected on PCI bus. Skipping."
    exit 0
fi

# 2. Check and install broadcom-sta-dkms
if ! dpkg -s broadcom-sta-dkms >/dev/null 2>&1 || ! dpkg -s broadcom-sta-dkms 2>/dev/null | grep -q "Status: install ok installed"; then
    echo "    --> Installing broadcom-sta-dkms..."
    sudo apt-get install -y broadcom-sta-dkms
fi

# 3. Ensure conflicting 'bcma' kernel driver is unloaded
if lsmod | grep -q "^bcma "; then
    echo "    --> Unloading conflicting 'bcma' module..."
    sudo modprobe -r bcma 2>/dev/null || true
fi

# 4. Load the proprietary 'wl' module if not currently loaded
if ! lsmod | grep -q "^wl "; then
    echo "    --> Loading Broadcom 'wl' kernel module..."
    sudo modprobe wl
fi

# 5. Restart NetworkManager to discover the new wireless interface
echo "    --> Refreshing NetworkManager service..."
sudo systemctl restart NetworkManager

# 6. Verify wireless device detection
if command -v nmcli >/dev/null 2>&1; then
    echo "    ✓ Wireless device status:"
    nmcli dev | grep -E "wifi|wlan" || echo "    ℹ Wi-Fi interface is initializing..."
fi

echo "==> [Dell XPS 13 9343] Broadcom Wi-Fi driver suite verified and active."
