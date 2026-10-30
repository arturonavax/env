#!/bin/bash
# Installs and configures RTK (Rust Token Killer) for transparent token compression
#
# Run: ./src/ai/install_rtk.sh
# shellcheck disable=SC2154
if [[ -f ./src/remotes/_vars_colors.sh ]]; then
	source ./src/remotes/_vars_colors.sh
elif [[ -f "$(dirname "$0")/../remotes/_vars_colors.sh" ]]; then
	source "$(dirname "$0")/../remotes/_vars_colors.sh"
fi

if [[ -f ./src/remotes/_versions.sh ]]; then
	source ./src/remotes/_versions.sh
elif [[ -f "$(dirname "$0")/../remotes/_versions.sh" ]]; then
	source "$(dirname "$0")/../remotes/_versions.sh"
fi

echo -e "${fgcolor_white_bold}[RTK Installer]: - Installing and configuring RTK (Rust Token Killer)...${fgcolor_reset}"

mkdir -p "$HOME/.local/bin" "$HOME/.cargo/bin"

if [[ "$(command -v rtk)" != "" ]]; then
	current_rtk_version="$(rtk --version 2>/dev/null || echo "installed")"
	echo -e "${fgcolor_green_bold}[RTK Installer]: RTK is already installed (${current_rtk_version}).${fgcolor_reset}"
else
	echo -e "${fgcolor_white_bold}[RTK Installer]: Downloading official pre-built RTK binary...${fgcolor_reset}"
	curl -fsSL https://raw.githubusercontent.com/rtk-ai/rtk/refs/heads/master/install.sh | sh || {
		if [[ "$(command -v cargo)" != "" ]]; then
			echo -e "${fgcolor_yellow_bold}[RTK Installer]: Binary download failed, compiling RTK from source using cargo...${fgcolor_reset}"
			cargo install --locked --git https://github.com/rtk-ai/rtk || :
		fi
	}
fi

# Ensure rtk binary is symlinked in ~/.local/bin and /usr/local/bin
if [[ -f "$HOME/.cargo/bin/rtk" && ! -f "$HOME/.local/bin/rtk" ]]; then
	ln -sf "$HOME/.cargo/bin/rtk" "$HOME/.local/bin/rtk"
fi
if [[ -f "$HOME/.local/bin/rtk" && ! -f /usr/local/bin/rtk ]]; then
	sudo ln -sf "$HOME/.local/bin/rtk" /usr/local/bin/rtk 2>/dev/null || :
fi

# Configure agent hooks (Google Antigravity, Claude Code, etc.)
if [[ "$(command -v rtk)" != "" ]]; then
	echo -e "${fgcolor_white_bold}[RTK Installer]: Configuring agent hooks and instructions...${fgcolor_reset}"
	export RTK_TELEMETRY_DISABLED=1
	rtk init --agent antigravity --auto-patch 2>/dev/null || :
	rtk init -g --auto-patch </dev/null 2>/dev/null || :
	echo -e "${fgcolor_green_bold}[RTK Installer]: ✔️ RTK hooks and token compression rules configured!${fgcolor_reset}"
fi

echo -e "${fgcolor_white_bold}[RTK Installer]: ${fgcolor_green_bold}✔️ RTK environment ready!${fgcolor_reset}"
