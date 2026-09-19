#!/bin/bash
# Run: ./src/requirements/ai.sh
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

source ./src/remotes/_required_commands.sh
source ./src/remotes/_vars_colors.sh

required-commands git curl

if [[ "$(command -v node)" == "" ]]; then
	echo -e "${fgcolor_white_bold}[AI Warning]: ${fgcolor_yellow_bold}NodeJS is recommended for Model Context Protocol (MCP) servers.${fgcolor_reset}"
fi

if [[ "$(command -v python3)" == "" ]]; then
	echo -e "${fgcolor_white_bold}[AI Warning]: ${fgcolor_yellow_bold}Python3 is recommended for AI harnesses and orchestrators.${fgcolor_reset}"
fi
