#!/bin/bash
# This script must be executed with the "source" command.
#
# Run: source <(curl -fsSL "https://env.arturonavax.dev/_cd.sh" | cat)
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
