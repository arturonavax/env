#!/bin/bash
# Run: ./src/requirements/terminal.sh
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

if [[ "$(uname -s)" == "Linux" ]]; then
	required-sudo-commands apt snap

elif [[ "$(uname -s)" == "Darwin" ]]; then
	required-commands brew
fi

required-commands git
