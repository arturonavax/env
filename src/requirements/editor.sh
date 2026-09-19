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

	elif [[ "$ID_LIKE" == *"rhel"* || "$ID_LIKE" == *"centos"* || "$ID_LIKE" == *"fedora"* || "$ID" == *"fedora"* ]]; then
		required-sudo-commands dnf
	fi

elif [[ "$(uname -s)" == "Darwin" ]]; then
	required-commands brew
fi

required-commands git make curl tar python3 cargo node npm

node_version_major="$(node --version | cut -d'.' -f1 | tr -d 'v')"

## check node version
if ((node_version_major < 18)); then
	echo -e "${fgcolor_white_bold}[Editor Warning]: ${fgcolor_yellow_bold}NodeJs version is recommended to be v18.0.0 or higher.${fgcolor_reset}"
fi
