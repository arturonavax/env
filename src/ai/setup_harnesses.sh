#!/bin/bash
# Prepares directories and hooks for agent harnesses, evaluation runners, and orchestrators.
#
# Run: ./src/ai/setup_harnesses.sh
source ./src/remotes/_vars_colors.sh

echo -e "${fgcolor_white_bold}[AI Installer]: - Setting up AI harnesses and orchestrator environment...${fgcolor_reset}"

# Create directories for local agent harnesses and orchestrator states
mkdir -p "$HOME/.local/share/ai/harnesses"
mkdir -p "$HOME/.local/share/ai/orchestrators"

# Synchronize local harness configs from repository if present
if [[ -d ./files/ai/harnesses ]]; then
	for harness_file in ./files/ai/harnesses/*; do
		if [[ -f "$harness_file" && "$(basename "$harness_file")" != ".gitkeep" && "$(basename "$harness_file")" != "README.md" ]]; then
			cp "$harness_file" "$HOME/.local/share/ai/harnesses/."
		fi
	done
fi

echo -e "${fgcolor_white_bold}[AI Installer]: ${fgcolor_green_bold}✔️ AI harnesses and orchestrators environment ready!${fgcolor_reset}"
