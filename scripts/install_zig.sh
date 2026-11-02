#!/usr/bin/env bash
# ==============================================================================
# install_zig.sh
# Standalone Zig compiler installer for Linux and macOS.
# Part of arturonavax/env development environment.
#
# Usage:
#   ./scripts/install_zig.sh [-v <version>] [-d <dir>]
#
# Flags:
#   -v <version>  Zig version to install (default: 0.14.1)
#   -d <dir>      Installation directory (default: $HOME/.local/opt/zig)
#   -h            Show help documentation
# ==============================================================================

set -euo pipefail

ZIG_VERSION="${ZIG_VERSION:-0.14.1}"
INSTALL_BASE="${HOME}/.local/opt"
BIN_DIR="${HOME}/.local/bin"

show_help() {
	cat <<EOF
Zig Standalone Installer

Usage:
  install_zig.sh [options]

Options:
  -v <version>  Zig version to install (default: ${ZIG_VERSION})
  -d <dir>      Installation base directory (default: ${INSTALL_BASE})
  -h            Show this help message
EOF
	exit 0
}

while getopts ':v:d:h' flag; do
	case "$flag" in
	v) ZIG_VERSION="$OPTARG" ;;
	d) INSTALL_BASE="$OPTARG" ;;
	h) show_help ;;
	*)
		echo "Unknown flag: -${OPTARG}" >&2
		exit 1
		;;
	esac
done

# Detect operating system
OS_NAME="$(uname -s)"
case "${OS_NAME}" in
Linux) OS="linux" ;;
Darwin) OS="macos" ;;
*)
	echo "Unsupported operating system: ${OS_NAME}" >&2
	exit 1
	;;
esac

# Detect architecture
ARCH_NAME="$(uname -m)"
case "${ARCH_NAME}" in
x86_64 | amd64) ARCH="x86_64" ;;
aarch64 | arm64) ARCH="aarch64" ;;
*)
	echo "Unsupported architecture: ${ARCH_NAME}" >&2
	exit 1
	;;
esac

TARGET_NAME="zig-${ARCH}-${OS}-${ZIG_VERSION}"
ARCHIVE_NAME="${TARGET_NAME}.tar.xz"
DOWNLOAD_URL="https://ziglang.org/download/${ZIG_VERSION}/${ARCHIVE_NAME}"
TARGET_DIR="${INSTALL_BASE}/${TARGET_NAME}"
SYMLINK_DIR="${INSTALL_BASE}/zig"

echo "Installing Zig ${ZIG_VERSION} (${OS}/${ARCH})..."

mkdir -p "${INSTALL_BASE}" "${BIN_DIR}"
TMP_DIR="$(mktemp -d -t anx-zig-XXXXXX)"
trap 'rm -rf "${TMP_DIR}"' EXIT

echo "Downloading ${DOWNLOAD_URL}..."
if command -v curl >/dev/null 2>&1; then
	curl -fsSL --retry 3 --connect-timeout 15 -o "${TMP_DIR}/${ARCHIVE_NAME}" "${DOWNLOAD_URL}"
elif command -v wget >/dev/null 2>&1; then
	wget -q -O "${TMP_DIR}/${ARCHIVE_NAME}" "${DOWNLOAD_URL}"
else
	echo "Error: Neither curl nor wget is available." >&2
	exit 1
fi

echo "Extracting ${ARCHIVE_NAME}..."
tar -xf "${TMP_DIR}/${ARCHIVE_NAME}" -C "${TMP_DIR}"

if [ ! -d "${TMP_DIR}/${TARGET_NAME}" ]; then
	echo "Error: Expected directory ${TARGET_NAME} not found in archive." >&2
	exit 1
fi

rm -rf "${TARGET_DIR}"
mv "${TMP_DIR}/${TARGET_NAME}" "${TARGET_DIR}"

# Create/update opt symlink
rm -f "${SYMLINK_DIR}"
ln -sfn "${TARGET_DIR}" "${SYMLINK_DIR}"

# Create/update bin symlink
ln -sf "${SYMLINK_DIR}/zig" "${BIN_DIR}/zig"

if [ -x "${BIN_DIR}/zig" ]; then
	INSTALLED_VER="$("${BIN_DIR}/zig" version)"
	echo "✓ Zig ${INSTALLED_VER} installed successfully to ${BIN_DIR}/zig"
else
	echo "Error: Zig binary at ${BIN_DIR}/zig is not executable." >&2
	exit 1
fi
