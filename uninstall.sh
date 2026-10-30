#!/usr/bin/env bash
# ==============================================================================
# uninstall.sh
# Multi-platform systematic uninstallation and cache purging entry point.
# Removes binaries, services, desktop entries, configurations, and caches cleanly.
# Part of arturonavax/env development environment.
# ==============================================================================

set -euo pipefail

# ANSI Colors
BOLD_WHITE="\033[1;37m"
GREEN="\033[0;32m"
YELLOW="\033[0;33m"
CYAN="\033[0;36m"
RED="\033[0;31m"
RESET="\033[0m"

log_info()    { echo -e "  [${CYAN}INFO${RESET}] $*"; }
log_step()    { echo -e "  [${BOLD_WHITE}STEP${RESET}] $*"; }
log_success() { echo -e "  [${GREEN}OK${RESET}] $*"; }
log_warn()    { echo -e "  [${YELLOW}WARN${RESET}] $*"; }
log_error()   { echo -e "  [${RED}ERROR${RESET}] $*" >&2; }

detect_os() {
	local u
	u="$(uname -s | tr '[:upper:]' '[:lower:]')"
	case "$u" in
		linux*) echo "linux" ;;
		darwin*) echo "darwin" ;;
		*) echo "$u" ;;
	esac
}

detect_arch() {
	uname -m
}

detect_distro() {
	if [[ -f /etc/os-release ]]; then
		# shellcheck source=/dev/null
		. /etc/os-release
		echo "${ID:-unknown}"
	else
		echo "unknown"
	fi
}

pkg_remove() {
	local pkgs=("$@")
	if command -v apt-get >/dev/null 2>&1; then
		sudo apt-get remove -y "${pkgs[@]}" 2>/dev/null || true
	elif command -v pacman >/dev/null 2>&1; then
		sudo pacman -R --noconfirm "${pkgs[@]}" 2>/dev/null || true
	elif command -v dnf >/dev/null 2>&1; then
		sudo dnf remove -y "${pkgs[@]}" 2>/dev/null || true
	elif command -v brew >/dev/null 2>&1; then
		brew uninstall "${pkgs[@]}" 2>/dev/null || true
	fi
}

ensure_sudo() {
	if [[ $EUID -ne 0 ]]; then
		if command -v sudo >/dev/null 2>&1; then
			sudo -v || { log_error "Failed to obtain sudo privileges."; exit 1; }
		else
			log_error "sudo is required for system uninstallation."
			exit 1
		fi
	fi
}

show_uninstall_plan() {
	echo -e "${BOLD_WHITE}Uninstallation Plan & Teardown Preview${RESET}"
	echo "  • Binaries:       /usr/local/bin/{anx,aitools,herdr,atuin,nvim,ghostty}, ~/.local/bin/{anx,aitools}"
	echo "  • Configurations: ~/.config/{anx,aitools,ghostty,herdr,atuin,nvim}"
	echo "  • Data & State:   ~/.local/share/{anx,atuin,nvim}, ~/.local/state/{anx,nvim}"
	echo "  • Caches:         ~/.cache/{anx,nvim}"
}

usage() {
	cat <<EOF
Usage: ./uninstall.sh [OPTIONS]

Options:
  -p, --plan, plan    Display uninstallation plan, purged binaries, and caches without modifying system.
  --dry-run, preview  Alias for --plan (safe preview).
  -h, --help          Show this help message.

Examples:
  ./uninstall.sh --plan     # Preview uninstallation plan (safe dry-run)
  ./uninstall.sh            # Execute full system uninstallation
EOF
}

# Early check for plan or help
for arg in "$@"; do
	case "$arg" in
	-p | --plan | plan | --dry-run | dry-run | --preview | preview)
		show_uninstall_plan
		exit 0
		;;
	-h | --help | help)
		usage
		exit 0
		;;
	esac
done

echo -e "${BOLD_WHITE}==============================================================================${RESET}"
echo -e "${BOLD_WHITE}  arturonavax/env Uninstaller${RESET}"
echo -e "${BOLD_WHITE}  Platform: $(detect_os) ($(detect_distro) - $(detect_arch))${RESET}"
echo -e "${BOLD_WHITE}==============================================================================${RESET}"

log_step "Acquiring administrator credentials for uninstallation..."
ensure_sudo
log_success "Privileges confirmed."

# ------------------------------------------------------------------------------
# Phase 1: Packages, Binaries, Desktop Entries & Services
# ------------------------------------------------------------------------------
log_step "Phase 1/2: Removing installed packages and binaries..."

# 1. Native package removal
pkg_remove neovim fzf

# 2. Linux snap cleanups if snap is present
if command -v snap >/dev/null 2>&1; then
	sudo snap remove --purge ghostty 2>/dev/null || true
	sudo snap remove --purge tmux 2>/dev/null || true
	sudo snap remove --purge nvim 2>/dev/null || true
fi

# 3. Purge system and local binaries
sudo rm -rf \
	/opt/nvim-linux-* /opt/zig-* \
	/usr/local/bin/zig /usr/local/bin/nvim "$HOME/.local/bin/nvim" \
	/usr/local/bin/tmux /usr/local/bin/herdr "$HOME/.local/bin/herdr" \
	/usr/local/bin/atuin "$HOME/.local/bin/atuin" "$HOME/.cargo/bin/atuin" \
	/usr/local/bin/rtk "$HOME/.local/bin/rtk" \
	/usr/bin/ghostty /usr/local/bin/ghostty /usr/share/ghostty \
	/usr/lib/libghostty* /usr/lib/libgtk4-layer-shell* /usr/lib64/libghostty* /usr/lib64/libgtk4-layer-shell* \
	/usr/include/ghostty* /usr/include/gtk4-layer-shell* \
	/usr/share/applications/*ghostty*.desktop /usr/share/applications/nvim.desktop /usr/share/applications/lvim.desktop \
	"$HOME/.local/share/applications/nvim.desktop" "$HOME/.local/share/applications/lvim.desktop" \
	/usr/share/terminfo/g/ghostty* /usr/share/terminfo/x/xterm-ghostty* \
	/usr/share/icons/hicolor/*/apps/*ghostty* 2>/dev/null || true

# 4. macOS Cask and App cleanups
if [[ "$(detect_os)" == "darwin" ]]; then
	if command -v brew >/dev/null 2>&1; then
		brew uninstall --cask ghostty 2>/dev/null || true
		brew uninstall tmux neovim fzf 2>/dev/null || true
	fi
	rm -rf /Applications/Ghostty.app 2>/dev/null || true
fi

# 5. Systemd user services cleanup
if command -v systemctl >/dev/null 2>&1; then
	systemctl --user stop app-com.mitchellh.ghostty.service ghostty.service 2>/dev/null || true
	systemctl --user disable app-com.mitchellh.ghostty.service ghostty.service 2>/dev/null || true
	rm -f "$HOME/.config/systemd/user/"*ghostty* 2>/dev/null || true
	if [[ -d /usr/lib/systemd/user ]]; then
		sudo rm -f /usr/lib/systemd/user/*ghostty* 2>/dev/null || true
	fi
	systemctl --user daemon-reload 2>/dev/null || true
fi

# 6. Desktop database and linker cache refresh
if [[ "$(detect_os)" == "linux" ]]; then
	sudo ldconfig 2>/dev/null || true
	if command -v update-desktop-database >/dev/null 2>&1; then
		sudo update-desktop-database /usr/share/applications 2>/dev/null || true
		update-desktop-database "$HOME/.local/share/applications" 2>/dev/null || true
	fi
fi

# ------------------------------------------------------------------------------
# Phase 2: Configuration Directories, Local States & Caches
# ------------------------------------------------------------------------------
log_step "Phase 2/2: Purging configuration files, local states, and caches..."

rm -rf \
	"$HOME/.local/bin/anx" \
	"$HOME/.local/bin/aitools" \
	"$HOME/.config/anx" \
	"$HOME/.local/state/anx" \
	"$HOME/.cache/anx" \
	"$HOME/.config/aitools" \
	"$HOME/.config/ghostty" \
	"$HOME/.config/herdr" \
	"$HOME/.config/atuin" "$HOME/.local/share/atuin" \
	"$HOME/.config/nvim" \
	"$HOME/.local/share/nvim" \
	"$HOME/.local/state/nvim" \
	"$HOME/.cache/nvim" 2>/dev/null || true

if command -v golangci-lint >/dev/null 2>&1; then
	golangci-lint cache clean 2>/dev/null || true
fi

log_success "Uninstallation completed successfully."
log_info "Binaries, desktop integrations, configs, and caches have been removed."
