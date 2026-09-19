#!/bin/bash
# Prepares the environment, directories and definitions for Model Context Protocol (MCP) servers.
#
# Run: ./src/ai/setup_mcp.sh
source ./src/remotes/_vars_colors.sh

echo -e "${fgcolor_white_bold}[AI Installer]: - Configuring Model Context Protocol (MCP) ecosystem...${fgcolor_reset}"

# Create directories for MCP configs and server repositories
mkdir -p "$HOME/.config/mcp"
mkdir -p "$HOME/.local/share/mcp"

# Synchronize transferable MCP configurations from repository if present
if [[ -d ./files/ai/mcp ]]; then
	for mcp_file in ./files/ai/mcp/*; do
		if [[ -f "$mcp_file" && "$(basename "$mcp_file")" != ".gitkeep" && "$(basename "$mcp_file")" != "README.md" ]]; then
			cp "$mcp_file" "$HOME/.config/mcp/."
		fi
	done
fi

echo -e "${fgcolor_white_bold}[AI Installer]: ${fgcolor_green_bold}✔️ MCP ecosystem directories and configurations ready!${fgcolor_reset}"
