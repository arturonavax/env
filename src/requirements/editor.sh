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
required-commands go python3 pipx cargo node npm pnpm

go_version_major="$(go version | cut -d' ' -f3 | cut -d'.' -f1 | tr -d 'go')"
go_version_minor="$(go version | cut -d' ' -f3 | cut -d'.' -f2)"
node_version_major="$(node --version | cut -d'.' -f1 | tr -d 'v')"

## check node version
if ((node_version_major < 12)); then
	echo -e "${fgcolor_white_bold}[Editor Error]: ${fgcolor_red_bold}NodeJs version must be equal to v12.0.0 or higher.${fgcolor_reset}"

	exit 1
fi

## check go version
if ((go_version_major < 1)) && ((go_version_minor < 15)); then
	echo -e "${fgcolor_white_bold}[Editor Error]: ${fgcolor_red_bold}Go version must be equal to v15.0.0 or higher.${fgcolor_reset}"

	exit 1
fi

## check go env
if [[ "$(go env GOROOT)" == "" ]]; then
	echo -e "${fgcolor_white_bold}[Editor Error]: ${fgcolor_red_bold}The GOROOT environment variable is not set.${fgcolor_reset}"

	exit 1
fi

if [[ "$(go env GOPATH)" == "" ]]; then
	echo "${fgcolor_white_bold}[Editor Error]: ${fgcolor_red_bold}The GOPATH environment variable is not set.${fgcolor_reset}"

	exit 1
fi
