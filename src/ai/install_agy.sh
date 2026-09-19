#!/bin/bash
# Installs and configures AI CLI tools and assistant runtime
#
# Run: ./src/ai/install_agy.sh
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

echo -e "${fgcolor_white_bold}[AI Installer]: - Installing / Updating AI CLI assistant...${fgcolor_reset}"

mkdir -p "$HOME/.local/bin"

if [[ "$(command -v agy)" != "" ]]; then
	current_agy_version="$(agy --version 2>/dev/null || echo "installed")"
	echo -e "${fgcolor_green_bold}[AI Installer]: AI CLI assistant is already installed (${current_agy_version}).${fgcolor_reset}"
else
	echo -e "${fgcolor_white_bold}[AI Installer]: Downloading AI CLI assistant installer...${fgcolor_reset}"
	curl -fsSL https://antigravity.google/cli/install.sh | bash
fi

# Ensure commands are available in PATH
if [[ -f "$HOME/.local/bin/agy" ]]; then
	ln -sf "$HOME/.local/bin/agy" "$HOME/.local/bin/agy-cli"
elif [[ "$(command -v agy)" != "" ]]; then
	ln -sf "$(command -v agy)" "$HOME/.local/bin/agy-cli"
fi

echo -e "${fgcolor_white_bold}[AI Installer]: ${fgcolor_green_bold}✔️ AI CLI assistant ready!${fgcolor_reset}"
