#!/bin/bash
# Run: curl -fsSL "https://env.arturonavax.dev/linux_osconfig.sh" | bash

repo_remote_files="https://env.arturonavax.dev"

# ------------------------------------------------------------------------------
# 1. Keyboard Repeat Configurations
# ------------------------------------------------------------------------------
# GNOME Keyboard configuration
if [[ "$(command -v gsettings)" != "" ]]; then
	gsettings set org.gnome.desktop.peripherals.keyboard repeat true
	gsettings set org.gnome.desktop.peripherals.keyboard delay 200
	gsettings set org.gnome.desktop.peripherals.keyboard repeat-interval 30
fi

# Xfce Keyboard configuration
if [[ "$(command -v xfconf-query)" != "" ]]; then
	xfconf-query -c keyboards -p /Default/KeyRepeat -n -t bool -s true
	xfconf-query -c keyboards -p /Default/KeyRepeat/Delay -n -t int -s 200
	xfconf-query -c keyboards -p /Default/KeyRepeat/Rate -n -t int -s 30
	xfconf-query -c keyboards -p /Default/RestoreNumlock -n -t bool -s true
	xfconf-query -c keyboards -p /Default/XkbDisable -n -t bool -s false
fi

# X11 Key Repeat (immediate effect for active session)
if [[ "$(command -v xset)" != "" ]]; then
	xset r rate 200 30
fi

# ------------------------------------------------------------------------------
# 2. rcmd Framework Installation (macOS rcmd behavior for Linux)
# ------------------------------------------------------------------------------
mkdir -p "$HOME/.local/bin" "$HOME/.local/lib/rcmd/backends" "$HOME/.config/rcmd"

# Find local repo root if executing locally
repo_root=""
curr_dir="$(pwd)"
while [[ "$curr_dir" != "/" ]]; do
	if [[ -d "$curr_dir/.git" && -d "$curr_dir/files/rcmd" ]]; then
		repo_root="$curr_dir"
		break
	fi
	curr_dir="$(dirname "$curr_dir")"
done

if [[ -n "$repo_root" ]]; then
	# Local install from repository
	cp "$repo_root/files/rcmd/rcmd" "$HOME/.local/bin/rcmd"
	cp "$repo_root/files/rcmd/backends/"*.sh "$HOME/.local/lib/rcmd/backends/" 2>/dev/null || :
	if [[ ! -f "$HOME/.config/rcmd/rcmd.conf" ]]; then
		cp "$repo_root/files/rcmd/rcmd.conf" "$HOME/.config/rcmd/rcmd.conf"
	fi
else
	# Remote install fallback via curl
	curl -fsSL "$repo_remote_files/files/rcmd/rcmd" -o "$HOME/.local/bin/rcmd" 2>/dev/null || :
	for backend in x11 gnome hyprland sway kwin; do
		curl -fsSL "$repo_remote_files/files/rcmd/backends/${backend}.sh" -o "$HOME/.local/lib/rcmd/backends/${backend}.sh" 2>/dev/null || :
	done
	if [[ ! -f "$HOME/.config/rcmd/rcmd.conf" ]]; then
		curl -fsSL "$repo_remote_files/files/rcmd/rcmd.conf" -o "$HOME/.config/rcmd/rcmd.conf" 2>/dev/null || :
	fi
fi

chmod +x "$HOME/.local/bin/rcmd" "$HOME/.local/lib/rcmd/backends/"*.sh 2>/dev/null || :

# Compatibility wrapper for run-or-raise
cat << 'EOF' > "$HOME/.local/bin/run-or-raise"
#!/usr/bin/env bash
exec "$HOME/.local/bin/rcmd" "$@"
EOF
chmod +x "$HOME/.local/bin/run-or-raise"

# ------------------------------------------------------------------------------
# 3. Desktop Shortcuts Synchronization (a-z)
# ------------------------------------------------------------------------------
if [[ "$(command -v xfconf-query)" != "" ]]; then
	"$HOME/.local/bin/rcmd" --sync-xfce >/dev/null 2>&1 || :
fi

if [[ "$(command -v gsettings)" != "" && "$XDG_CURRENT_DESKTOP" =~ "GNOME" ]]; then
	"$HOME/.local/bin/rcmd" --sync-gnome >/dev/null 2>&1 || :
fi

# ------------------------------------------------------------------------------
# 4. keyd Configuration (Right Alt -> rcmd layer)
# ------------------------------------------------------------------------------
if [[ -d "/etc/keyd" && -n "$repo_root" && -f "$repo_root/files/keyd/default.conf" ]]; then
	if sudo -n true 2>/dev/null; then
		sudo cp "$repo_root/files/keyd/default.conf" "/etc/keyd/default.conf" 2>/dev/null || :
		sudo keyd reload 2>/dev/null || :
	fi
fi
