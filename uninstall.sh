#!/usr/bin/env bash
# ==============================================================================
# uninstall.sh
# Multi-platform systematic uninstallation and cache purging entry point.
# Removes binaries, services, desktop entries, configurations, and caches cleanly.
# Part of arturonavax/env development environment.
# ==============================================================================

while [[ ! -d ./.git/ && ! -d ./files/ && ! -d ./src/ ]]; do
	if [[ "$PWD" == "/" ]]; then
		echo "[Uninstaller Error]: Repository root not found." >&2
		exit 1
	fi
	if ! cd ..; then
		echo "[Uninstaller Error]: Failed to change directory." >&2
		exit 1
	fi
done

source ./src/lib/strict.sh
source ./src/lib/colors.sh
source ./src/lib/logging.sh
source ./src/lib/platform.sh
source ./src/lib/packages.sh

log_box "arturonavax/env Uninstaller" \
	"Platform: $(detect_os) ($(detect_distro) - $(detect_arch))" \
	"Phase 1: Package removal, systemd units, binaries & desktop entries." \
	"Phase 2: Purging configuration files, local states, and caches."

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
	"$HOME/.config/ghostty" \
	"$HOME/.config/herdr" \
	"$HOME/.config/atuin" "$HOME/.local/share/atuin" \
	"$HOME/.config/nvim" \
	"$HOME/.local/share/nvim" \
	"$HOME/.local/state/nvim" \
	"$HOME/.cache/nvim" \
	"$HOME/.cache/tmux" \
	"$HOME/.tmux" 2>/dev/null || true

if command -v golangci-lint >/dev/null 2>&1; then
	golangci-lint cache clean 2>/dev/null || true
fi

log_box "Uninstallation Completed Successfully" \
	"Binaries, desktop integrations, configs, and caches have been removed."
