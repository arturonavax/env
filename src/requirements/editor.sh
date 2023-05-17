#!/bin/bash
# Run: ./src/requirements/editor.sh
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

source ./src/remotes/_vars_colors.sh
source ./src/remotes/_required_commands.sh

# check requireds
if [[ "$(uname -s)" == "Linux" ]]; then
	source /etc/os-release

	if [[ "$ID_LIKE" == *"debian"* || "$ID_LIKE" == *"ubuntu"* ]]; then
		required-sudo-commands apt

	elif [[ "$ID_LIKE" == *"rhel"* || "$ID_LIKE" == *"centos"* || "$ID_LIKE" == *"fedora"* ]]; then
		required-sudo-commands dnf
	fi

elif [[ "$(uname -s)" == "Darwin" ]]; then
	required-commands brew
fi

required-commands git make python3 pip cargo node npm

if ! python3 -m pip &>/dev/null; then
	echo -e "${fgcolor_white_bold}[Editor Error]: ${fgcolor_red_bold}python3 must have the 'pip' module.${fgcolor_reset}"

	exit 1
fi

node_version_major="$(node --version | cut -d'.' -f1 | tr -d 'v')"

## check node version
if ((node_version_major < 12)); then
	echo -e "${fgcolor_white_bold}[Editor Error]: ${fgcolor_red_bold}NodeJs version must be equal to v12.0.0 or higher.${fgcolor_reset}"

	exit 1
fi
