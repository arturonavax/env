#!/usr/bin/env bash
# ==============================================================================
# src/lib/platform.sh
# Precision multi-platform OS, distribution, architecture and hardware discovery.
# Part of arturonavax/env development environment.
# ==============================================================================

# Prevent double sourcing
[[ -n "${_PLATFORM_SH_LOADED:-}" ]] && return 0
_PLATFORM_SH_LOADED=1

_LIB_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
if [[ -f "${_LIB_DIR}/colors.sh" ]]; then
	# shellcheck source=/dev/null
	source "${_LIB_DIR}/colors.sh"
fi
if [[ -f "${_LIB_DIR}/ui_box.sh" ]]; then
	# shellcheck source=/dev/null
	source "${_LIB_DIR}/ui_box.sh"
fi

# Detect Operating System Kernel
detect_os() {
	# 1. Android Termux detection
	if [[ -n "${TERMUX_VERSION:-}" ]] || [[ "${PREFIX:-}" == *"com.termux"* ]] || [[ -d "/data/data/com.termux" ]]; then
		echo "android"
		return 0
	fi

	local kernel
	kernel="$(uname -s 2>/dev/null || echo "Unknown")"
	case "$kernel" in
	Linux*)
		if grep -qi "microsoft" /proc/version 2>/dev/null; then
			echo "wsl"
		else
			echo "linux"
		fi
		;;
	Darwin*)
		echo "darwin"
		;;
	CYGWIN*)
		echo "windows_cygwin"
		;;
	MINGW* | MSYS*)
		echo "windows_msys"
		;;
	*)
		echo "unknown"
		;;
	esac
}

# Detect Hardware Architecture
detect_arch() {
	local machine
	machine="$(uname -m 2>/dev/null || echo "unknown")"
	case "$machine" in
	x86_64 | amd64)
		echo "x86_64"
		;;
	aarch64 | arm64)
		echo "arm64"
		;;
	armv7l | armv6l)
		echo "armv7"
		;;
	i386 | i686)
		echo "x86_32"
		;;
	*)
		echo "$machine"
		;;
	esac
}

# Detect Linux Distribution Family
detect_distro() {
	local os_type
	os_type="$(detect_os)"

	if [[ "$os_type" == "android" ]]; then
		echo "termux"
		return 0
	elif [[ "$os_type" == "darwin" ]]; then
		echo "darwin"
		return 0
	elif [[ "$os_type" == "windows" || "$os_type" == "windows_msys" || "$os_type" == "windows_cygwin" ]]; then
		echo "windows"
		return 0
	fi

	if [[ -f /etc/os-release ]]; then
		# shellcheck source=/dev/null
		local id="" id_like=""
		id="$(grep -E '^ID=' /etc/os-release | cut -d'=' -f2 | tr -d '"' | tr '[:upper:]' '[:lower:]')"
		id_like="$(grep -E '^ID_LIKE=' /etc/os-release | cut -d'=' -f2 | tr -d '"' | tr '[:upper:]' '[:lower:]')"

		case "$id" in
		ubuntu | pop | elementary | linuxmint | zorin)
			echo "ubuntu"
			return 0
			;;
		debian | raspbian | kali)
			echo "debian"
			return 0
			;;
		fedora)
			echo "fedora"
			return 0
			;;
		rhel | centos | rocky | almalinux)
			echo "rhel"
			return 0
			;;
		arch | endeavouros | manjaro | garuda | artix)
			echo "arch"
			return 0
			;;
		alpine)
			echo "alpine"
			return 0
			;;
		opensuse* | sles)
			echo "suse"
			return 0
			;;
		esac

		# Fallback to ID_LIKE
		if [[ "$id_like" == *"ubuntu"* ]]; then
			echo "ubuntu"
			return 0
		elif [[ "$id_like" == *"debian"* ]]; then
			echo "debian"
			return 0
		elif [[ "$id_like" == *"fedora"* ]]; then
			echo "fedora"
			return 0
		elif [[ "$id_like" == *"rhel"* || "$id_like" == *"centos"* ]]; then
			echo "rhel"
			return 0
		elif [[ "$id_like" == *"arch"* ]]; then
			echo "arch"
			return 0
		fi
	fi

	echo "unknown"
}

# Detect Native Package Manager
detect_pkg_mgr() {
	local distro
	distro="$(detect_distro)"

	case "$distro" in
	termux)
		echo "termux"
		;;
	ubuntu | debian)
		echo "apt"
		;;
	fedora | rhel)
		if command -v dnf >/dev/null 2>&1; then
			echo "dnf"
		else
			echo "yum"
		fi
		;;
	arch)
		echo "pacman"
		;;
	suse)
		echo "zypper"
		;;
	alpine)
		echo "apk"
		;;
	darwin)
		echo "brew"
		;;
	*)
		if command -v pkg >/dev/null 2>&1 && [[ -n "${TERMUX_VERSION:-}" || -d "/data/data/com.termux" ]]; then
			echo "termux"
		elif command -v apt-get >/dev/null 2>&1; then
			echo "apt"
		elif command -v dnf >/dev/null 2>&1; then
			echo "dnf"
		elif command -v pacman >/dev/null 2>&1; then
			echo "pacman"
		elif command -v zypper >/dev/null 2>&1; then
			echo "zypper"
		elif command -v brew >/dev/null 2>&1; then
			echo "brew"
		else
			echo "unknown"
		fi
		;;
	esac
}

# Determine if the platform supports/requires sudo elevation
has_sudo_capability() {
	if [[ "$(detect_os)" == "android" ]] || [[ "$(detect_distro)" == "termux" ]]; then
		return 1 # Termux runs single-user; no sudo required or available by default
	fi
	if [[ $EUID -eq 0 ]] || command -v sudo >/dev/null 2>&1; then
		return 0
	fi
	return 1
}

# Detect Init System
detect_init_system() {
	if [[ "$(detect_os)" == "darwin" ]]; then
		echo "launchd"
		return 0
	fi

	if command -v systemctl >/dev/null 2>&1 && systemctl is-system-running >/dev/null 2>&1; then
		echo "systemd"
	elif [[ -d /run/systemd/system ]]; then
		echo "systemd"
	elif command -v openrc >/dev/null 2>&1; then
		echo "openrc"
	else
		echo "unknown"
	fi
}

# Detect Display Server
detect_display_server() {
	if [[ -n "${WAYLAND_DISPLAY:-}" ]]; then
		echo "wayland"
	elif [[ -n "${DISPLAY:-}" ]]; then
		echo "x11"
	else
		echo "headless"
	fi
}

# Portable Free Disk Space in Megabytes
get_free_disk_space_mb() {
	local target_path="${1:-$HOME}"
	# df -k outputs 1024-byte blocks portably across GNU coreutils and BSD/macOS
	df -k "$target_path" 2>/dev/null | awk 'NR>1 {print $4}' | tail -n 1 | awk '{printf "%d\n", $1 / 1024}'
}

# Portable Total RAM in Megabytes
get_total_ram_mb() {
	if [[ -f /proc/meminfo ]]; then
		awk '/MemTotal/ {printf "%d\n", $2 / 1024}' /proc/meminfo
	elif [[ "$(detect_os)" == "darwin" ]]; then
		local bytes
		bytes="$(sysctl -n hw.memsize 2>/dev/null || echo 0)"
		echo "$((bytes / 1024 / 1024))"
	else
		echo "0"
	fi
}

# Print Beautiful Platform Summary
print_platform_summary() {
	local os distro arch pkg_mgr init_sys display free_disk ram
	os="$(detect_os)"
	distro="$(detect_distro)"
	arch="$(detect_arch)"
	pkg_mgr="$(detect_pkg_mgr)"
	init_sys="$(detect_init_system)"
	display="$(detect_display_server)"
	free_disk="$(get_free_disk_space_mb "$HOME")"
	ram="$(get_total_ram_mb)"

	local width
	width="$(ui_box_width 70)"

	echo
	ui_box_top "$width" "rounded" "$BOLD_BLUE"
	ui_box_line "System Hardware & Platform Profile" "$width" "rounded" "$BOLD_BLUE" "$BOLD_WHITE" "left"
	ui_box_mid "$width" "rounded" "$BOLD_BLUE"
	ui_box_line "• Operating System:  $os" "$width" "rounded" "$BOLD_BLUE" "$RESET" "left"
	ui_box_line "• Distribution:      $distro" "$width" "rounded" "$BOLD_BLUE" "$RESET" "left"
	ui_box_line "• Architecture:      $arch" "$width" "rounded" "$BOLD_BLUE" "$RESET" "left"
	ui_box_line "• Package Manager:   $pkg_mgr" "$width" "rounded" "$BOLD_BLUE" "$RESET" "left"
	ui_box_line "• Init System:       $init_sys" "$width" "rounded" "$BOLD_BLUE" "$RESET" "left"
	ui_box_line "• Display Server:    $display" "$width" "rounded" "$BOLD_BLUE" "$RESET" "left"
	ui_box_line "• Total Memory:      ${ram} MB" "$width" "rounded" "$BOLD_BLUE" "$RESET" "left"
	ui_box_line "• Home Free Space:   ${free_disk} MB" "$width" "rounded" "$BOLD_BLUE" "$RESET" "left"
	ui_box_bottom "$width" "rounded" "$BOLD_BLUE"
	echo
}
