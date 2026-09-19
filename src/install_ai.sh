#!/bin/bash
# Installs AI tooling, CLI assistants, MCP environment, harnesses and prompts.
#
# Run: ./src/install_ai.sh
while [[ ! -d ./.git/ && ! -d ./files/ && ! -d ./src/ ]]; do
	if [[ "$PWD" == "/" ]]; then
		echo "Repository root not found"

		exit 1
	fi

	if ! cd ..; then
		echo "An error occurred while doing 'cd ..'"

		exit 1
	fi
done

./src/requirements/ai.sh || exit 1

source ./src/remotes/_vars_colors.sh
source ./src/remotes/_versions.sh

function install_ai() {
	set -o errexit
	trap exit-error-message ERR SIGINT

	echo -e "${fgcolor_white_bold}[AI Installer]: Starting install_ai.sh script...${fgcolor_reset}"

	# 1. AI CLI Assistant
	bash ./src/ai/install_agy.sh

	# 2. Model Context Protocol (MCP) Ecosystem
	bash ./src/ai/setup_mcp.sh

	# 3. Agent Harnesses & Orchestration
	bash ./src/ai/setup_harnesses.sh

	# 4. Prompting, Rules & System Instructions
	bash ./src/ai/setup_prompts.sh

	echo
	echo -e "${fgcolor_white_bold}[AI Installer]: ${fgcolor_green_bold}✔️ AI environment successfully installed and configured!${fgcolor_reset}"
	echo -en "$fgcolor_reset"
}

function exit-error-message() {
	echo -e "$(
		cat <<EOF

${fgcolor_white_bold}[Installer AI Error]: ---
[Installer AI Error]: ${fgcolor_red_bold}The installation had an error and was interrupted, the installation was not completed.${fgcolor_white_bold}
[Installer AI Error]: ---${fgcolor_reset}
EOF
	)"

	exit 1
}

install_ai "$@"
