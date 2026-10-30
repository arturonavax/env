#!/bin/bash
# Prepares directories and templates for AI prompting, rules, and system instructions.
#
# Run: ./src/ai/setup_prompts.sh
if [[ -f ./src/remotes/_vars_colors.sh ]]; then
	source ./src/remotes/_vars_colors.sh
elif [[ -f "$(dirname "$0")/../remotes/_vars_colors.sh" ]]; then
	source "$(dirname "$0")/../remotes/_vars_colors.sh"
fi

echo -e "${fgcolor_white_bold}[AI Installer]: - Setting up AI prompting and rules environment...${fgcolor_reset}"

# Create directories for prompt templates, system instructions, and agent rule sets
mkdir -p "$HOME/.config/ai/prompts"
mkdir -p "$HOME/.config/ai/rules"

# Synchronize prompting files from repository if present
if [[ -d ./files/ai/prompts ]]; then
	for prompt_file in ./files/ai/prompts/*; do
		if [[ -f "$prompt_file" && "$(basename "$prompt_file")" != ".gitkeep" && "$(basename "$prompt_file")" != "README.md" ]]; then
			cp "$prompt_file" "$HOME/.config/ai/prompts/."
		fi
	done
fi

echo -e "${fgcolor_white_bold}[AI Installer]: ${fgcolor_green_bold}✔️ AI prompts and rules environment ready!${fgcolor_reset}"
