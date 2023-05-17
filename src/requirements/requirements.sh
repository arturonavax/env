#!/bin/bash
# Run: ./src/requirements/requirements.sh
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
	source /etc/os-release

	if [[ "$ID_LIKE" == *"debian"* || "$ID_LIKE" == *"ubuntu"* ]]; then
		required-sudo-commands apt

	elif [[ "$ID_LIKE" == *"rhel"* || "$ID_LIKE" == *"centos"* ]]; then
		required-sudo-commands yum
	fi
fi
