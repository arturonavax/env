#!/bin/bash
# Installs and configures Antigravity CLI (agy / agy-cli)
#
# Run: ./src/ai/install_agy.sh
source ./src/remotes/_vars_colors.sh
source ./src/remotes/_versions.sh

echo -e "${fgcolor_white_bold}[AI Installer]: - Installing / Updating Antigravity CLI (agy)...${fgcolor_reset}"

mkdir -p "$HOME/.local/bin"

if [[ "$(command -v agy)" != "" ]]; then
	current_agy_version="$(agy --version 2>/dev/null || echo "installed")"
	echo -e "${fgcolor_green_bold}[AI Installer]: agy is already installed (${current_agy_version}).${fgcolor_reset}"
else
	echo -e "${fgcolor_white_bold}[AI Installer]: Downloading Antigravity CLI installer...${fgcolor_reset}"
	curl -fsSL https://antigravity.google/cli/install.sh | bash
fi

# Ensure both agy and agy-cli commands are available in PATH
if [[ -f "$HOME/.local/bin/agy" ]]; then
	ln -sf "$HOME/.local/bin/agy" "$HOME/.local/bin/agy-cli"
elif [[ "$(command -v agy)" != "" ]]; then
	ln -sf "$(command -v agy)" "$HOME/.local/bin/agy-cli"
fi

echo -e "${fgcolor_white_bold}[AI Installer]: ${fgcolor_green_bold}✔️ Antigravity CLI (agy / agy-cli) ready!${fgcolor_reset}"
