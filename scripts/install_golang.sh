#!/usr/bin/env bash
# ==============================================================================
# install_golang.sh
# Standalone Golang release installer, version checker, and source compiler.
# Part of arturonavax/env development environment.
#
# Primary Recommendation:
#   For multi-version Go management, GVM is the recommended primary manager.
#   This script serves as a standalone system-wide Go runtime provider.
#
# Usage:
#   ./src/remotes/install_golang.sh [-i] [-c]
# Remote:
#   bash <(curl -fsSL "https://env.arturonavax.dev/install_golang.sh") -i
#
# Flags:
#   -i   Install or update Go to the latest stable release.
#   -c   Compile Go from source (requires existing bootstrap Go or download).
#   -h   Show help documentation.
# ==============================================================================

set -euo pipefail

# ANSI colors
fcwb="\033[1;37m"
fcgb="\033[1;32m"
fcyb="\033[1;33m"
fcrb="\033[1;31m"
fccb="\033[1;36m"
fcr="\033[0m"

INSTALL_FLAG=0
COMPILE_FLAG=0
INSTALLATION_DIR="/usr/local"
BOOTSTRAP_DIR="${INSTALLATION_DIR}/go-bootstrap"
API_URL="https://go.dev/dl/?mode=json"
DL_URL="https://go.dev/dl/"

show_help() {
	cat <<EOF
${fcwb}Golang Standalone Installer & Updater${fcr}

Usage:
  install_golang.sh [options]

Options:
  -i          Install or update to latest official Go release
  -c          Compile from source repository (via git + all.bash)
  -h          Show this help message

Environment Notice:
  GVM (Go Version Manager) is the primary Go manager for arturonavax/env.
  This standalone installer installs official binaries into ${INSTALLATION_DIR}/go.
EOF
	exit 0
}

while getopts ':ich' flag; do
	case "$flag" in
	i) INSTALL_FLAG=1 ;;
	c) COMPILE_FLAG=1 ;;
	h) show_help ;;
	*)
		echo -e "${fcwb}[Golang Installer]: ${fcrb}Unknown flag: -${OPTARG}${fcr}. Use ${fcgb}'-i'${fcr} to install/update or ${fcgb}'-c'${fcr} to compile from source." >&2
		exit 1
		;;
	esac
done

# HTTP client helper (curl or wget)
fetch_url() {
	local url="$1"
	if command -v curl >/dev/null 2>&1; then
		curl -fsSL "$url"
	elif command -v wget >/dev/null 2>&1; then
		wget -qO- "$url"
	else
		echo -e "${fcwb}[Golang Installer]: ${fcrb}Error: curl or wget is required to query releases.${fcr}" >&2
		exit 1
	fi
}

echo -e "${fcwb}[Golang Installer]: 🔎 Resolving latest stable release of Golang ${fccb}🐹${fcwb}...${fcr}"

# 1. Query latest stable release via official Go JSON API
latest_version=""
json_response="$(fetch_url "$API_URL" 2>/dev/null || true)"
if [[ -n "$json_response" ]]; then
	latest_version="$(echo "$json_response" | grep -o '"version": "go[^"]*"' | head -n 1 | cut -d '"' -f 4 || true)"
fi

# Fallback to HTML scraping if JSON API fails
if [[ -z "$latest_version" ]]; then
	latest_version="$(fetch_url "$DL_URL" 2>/dev/null | grep -E "/go1(\.[0-9]+){0,2}" | grep -v "[0-9]rc" | grep -E -o "/go1(\.[0-9]+){0,2}" | sort --version-sort | tail -n 1 | cut -d "/" -f 2 || true)"
fi

if [[ -z "$latest_version" ]]; then
	echo -e "${fcwb}[Golang Installer]: ${fcrb}Failed to resolve latest Go version from go.dev.${fcr}" >&2
	exit 1
fi

# 2. Detect Operating System & CPU Architecture
raw_os="$(uname -s | tr '[:upper:]' '[:lower:]')"
raw_arch="$(uname -m)"

case "$raw_os" in
linux)
	target_os="linux"
	case "$raw_arch" in
	x86_64 | amd64) target_arch="amd64" ;;
	aarch64 | arm64) target_arch="arm64" ;;
	i386 | i686) target_arch="386" ;;
	armv6l | armv7l) target_arch="armv6l" ;;
	ppc64le) target_arch="ppc64le" ;;
	s390x) target_arch="s390x" ;;
	*)
		echo -e "${fcwb}[Golang Installer]: ${fcrb}Unsupported Linux architecture: $raw_arch${fcr}" >&2
		exit 1
		;;
	esac
	;;
darwin)
	target_os="darwin"
	if [[ "$raw_arch" == "arm64" ]] || [[ "$(sysctl -n machdep.cpu.brand_string 2>/dev/null || true)" == *"Apple"* ]]; then
		target_arch="arm64"
	else
		target_arch="amd64"
	fi
	;;
*)
	echo -e "${fcwb}[Golang Installer]: ${fcrb}Unsupported operating system: $raw_os${fcr}" >&2
	exit 1
	;;
esac

archive_filename="${latest_version}.${target_os}-${target_arch}.tar.gz"
archive_url="${DL_URL}${archive_filename}"

# Helper for binary release installation
install_binary_release() {
	echo -e "${fcwb}[Golang Installer]: ${fccb}⚡ Downloading ${archive_filename}...${fcr}"
	local tmp_dir
	tmp_dir="$(mktemp -d "${TMPDIR:-/tmp}/golang_install.XXXXXX")"
	local tmp_archive="${tmp_dir}/${archive_filename}"

	if command -v curl >/dev/null 2>&1; then
		curl --proto '=https' --tlsv1.2 -fsSL "$archive_url" -o "$tmp_archive"
	else
		wget -q "$archive_url" -O "$tmp_archive"
	fi

	echo -e "${fcwb}[Golang Installer]: Extracting into ${INSTALLATION_DIR}/go...${fcr}"
	sudo rm -rf "${INSTALLATION_DIR}/go"
	sudo mkdir -p "${INSTALLATION_DIR}"
	sudo tar -C "${INSTALLATION_DIR}" -xzf "$tmp_archive"
	sudo chmod -R a+rX "${INSTALLATION_DIR}/go"
	rm -rf "$tmp_dir"
}

# Helper for compiling from source
install_from_source() {
	echo -e "${fcwb}[Golang Installer]: Compiling Go from source repository...${fcr}"

	# Ensure bootstrap Go exists
	if [[ ! -x "${BOOTSTRAP_DIR}/go/bin/go" ]]; then
		echo -e "${fcwb}[Golang Installer]: Provisioning bootstrap compiler in ${BOOTSTRAP_DIR}...${fcr}"
		local b_tmp
		b_tmp="$(mktemp -d "${TMPDIR:-/tmp}/go_bootstrap.XXXXXX")"
		local b_archive="${b_tmp}/${archive_filename}"
		if command -v curl >/dev/null 2>&1; then
			curl --proto '=https' --tlsv1.2 -fsSL "$archive_url" -o "$b_archive"
		else
			wget -q "$archive_url" -O "$b_archive"
		fi
		sudo mkdir -p "$BOOTSTRAP_DIR"
		sudo tar -C "$BOOTSTRAP_DIR" -xzf "$b_archive"
		rm -rf "$b_tmp"
	fi

	local source_repo="${INSTALLATION_DIR}/go"
	if [[ -d "${source_repo}/.git" ]]; then
		echo -e "${fcwb}[Golang Installer]: Updating existing Go git clone...${fcr}"
		sudo git -C "$source_repo" fetch --tags
		sudo git -C "$source_repo" checkout "$latest_version"
	else
		echo -e "${fcwb}[Golang Installer]: Cloning Go source repository...${fcr}"
		sudo rm -rf "$source_repo"
		sudo git clone --depth 1 --branch "$latest_version" https://go.googlesource.com/go "$source_repo"
	fi

	echo -e "${fcwb}[Golang Installer]: Running ./all.bash compilation...${fcr}"
	sudo env GOROOT_BOOTSTRAP="${BOOTSTRAP_DIR}/go" bash -c "cd '${source_repo}/src' && ./all.bash"
}

# 3. Version Comparison & Action
current_version=""
if command -v go >/dev/null 2>&1; then
	current_version="$(go version 2>/dev/null | awk '{print $3}' || true)"
elif [[ -x "${INSTALLATION_DIR}/go/bin/go" ]]; then
	current_version="$("${INSTALLATION_DIR}/go/bin/go" version 2>/dev/null | awk '{print $3}' || true)"
fi

if [[ -z "$current_version" ]]; then
	echo -e "${fcwb}[Golang Installer]: ${fcrb}❌ Golang is not currently installed on this system.${fcr}"
	echo -e "${fcwb}[Golang Installer]: ${fcyb}ℹ️  Latest stable release: ${fcyb}${latest_version#go}${fcr}"
	echo

	if [[ "$INSTALL_FLAG" -eq 1 ]]; then
		if [[ "$COMPILE_FLAG" -eq 1 ]]; then
			install_from_source
		else
			install_binary_release
		fi
		echo -e "${fcwb}[Golang Installer]: ${fcgb}✔️ Installation completed successfully!${fcr}"
	else
		echo -e "${fcwb}[Golang Installer]: Run with ${fcgb}'-i'${fcr} to automatically install: ${fccb}./src/remotes/install_golang.sh -i${fcr}"
	fi
elif [[ "$current_version" != "$latest_version" ]]; then
	echo -e "${fcwb}[Golang Installer]: ${fcyb}⚠️ Installed version (${current_version#go}) differs from latest (${latest_version#go}).${fcr}"
	if [[ "$INSTALL_FLAG" -eq 1 ]]; then
		echo -e "${fcwb}[Golang Installer]: ${fccb}⚡ Updating Go to ${latest_version#go}...${fcr}"
		if [[ "$COMPILE_FLAG" -eq 1 ]]; then
			install_from_source
		else
			install_binary_release
		fi
		echo -e "${fcwb}[Golang Installer]: ${fcgb}✔️ Update completed successfully!${fcr}"
	else
		echo -e "${fcwb}[Golang Installer]: Run with ${fcgb}'-i'${fcr} to automatically update: ${fccb}./src/remotes/install_golang.sh -i${fcr}"
	fi
else
	echo -e "${fcwb}[Golang Installer]: ${fcgb}✔️ Golang is up to date: ${current_version#go}${fcr}"
fi

echo -e "\n${fcwb}[Golang Installer]: ${fcyb}Note:${fcr} To manage multiple Go versions dynamically per project, use ${fcwb}GVM${fcr}:"
echo -e "  ${fccb}gvm install ${latest_version}${fcr}"
echo -e "  ${fccb}gvm use ${latest_version} --default${fcr}"
