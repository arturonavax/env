#!/usr/bin/env bash
# ==============================================================================
# install_ghostty.sh
# Ghostty terminal emulator installer for Debian/Ubuntu and Linux.
# Part of arturonavax/env development environment.
# ==============================================================================

set -euo pipefail

if command -v ghostty >/dev/null 2>&1; then
	echo "✓ ghostty already installed at $(command -v ghostty)"
	exit 0
fi

# 1. Native apt install if available (Ubuntu 26.04+ has official ghostty in universe)
if command -v apt-get >/dev/null 2>&1; then
	if sudo DEBIAN_FRONTEND=noninteractive apt-get update -qq && sudo DEBIAN_FRONTEND=noninteractive apt-get install -y ghostty; then
		echo "✓ ghostty installed successfully via apt"
		exit 0
	fi
fi

# 2. Prebuilt debian/ubuntu package from mkasberg/ghostty-ubuntu
ARCH="$(uname -m)"
case "$ARCH" in
x86_64 | amd64) DEB_ARCH="amd64" ;;
aarch64 | arm64) DEB_ARCH="arm64" ;;
*) DEB_ARCH="" ;;
esac

if [ -n "$DEB_ARCH" ] && command -v dpkg >/dev/null 2>&1; then
	echo "Attempting installation via mkasberg/ghostty-ubuntu release..."
	TMP_DIR="$(mktemp -d -t anx-ghostty-XXXXXX)"
	trap 'rm -rf "${TMP_DIR}"' EXIT

	# Detect Ubuntu version codename/id
	UBUNTU_VER="26.04"
	if [ -f /etc/os-release ]; then
		# shellcheck source=/dev/null
		. /etc/os-release
		if [ -n "${VERSION_ID:-}" ]; then
			UBUNTU_VER="${VERSION_ID}"
		fi
	fi

	DEB_URL="https://github.com/mkasberg/ghostty-ubuntu/releases/download/1.3.1-0.ppa2/ghostty_1.3.1-0.ppa2_${DEB_ARCH}_${UBUNTU_VER}.deb"
	if curl -fsSL -o "${TMP_DIR}/ghostty.deb" "${DEB_URL}" 2>/dev/null; then
		if sudo DEBIAN_FRONTEND=noninteractive apt-get install -y "${TMP_DIR}/ghostty.deb"; then
			echo "✓ ghostty installed successfully via mkasberg/ghostty-ubuntu .deb"
			exit 0
		fi
	fi
fi

# 3. Snap fallback
if command -v snap >/dev/null 2>&1; then
	echo "Installing ghostty via snap..."
	if sudo snap install ghostty --classic; then
		echo "✓ ghostty installed successfully via snap"
		exit 0
	fi
fi

echo "Error: Unable to install ghostty via apt, prebuilt deb, or snap." >&2
exit 1
