#!/bin/bash
# Run: bash <(curl -fsSL "https://env.arturonavax.dev/install.sh")
# Parameter to install requirements: requirements
# Parameter to install terminal: terminal
# Parameter to install editor: editor
# Parameter to install OS config: osconfig
# Parameter to install all: all
# Parameter to help: help
bash <(curl -fsSL "https://env.arturonavax.dev/usage_install.sh") "$@" || exit 1

[[ -z "$installation_folder" ]] && installation_folder="$HOME/arturonavax-env"

echo "Installation folder: $installation_folder"
echo

if [[ "$(uname -s)" == "Linux" && "$(command -v git)" == "" ]]; then
	source /etc/os-release

	# install git
	if [[ "$ID_LIKE" == *"debian"* || "$ID_LIKE" == *"ubuntu"* ]]; then
		sudo apt install -y git

	elif [[ "$ID_LIKE" == *"rhel"* || "$ID_LIKE" == *"centos"* || "$ID_LIKE" == *"fedora"* || "$ID" == *"fedora"* ]]; then
		sudo dnf install -y git

	else
		echo "The operating system is not compatible with this installation." && exit 1
	fi

elif [[ "$(uname -s)" == "Darwin" && ! -e "/Library/Developer/CommandLineTools/usr/bin/git" ]]; then
	# install CommandLineTools (this contains GIT)
	curl -fsSL "https://env.arturonavax.dev/macos_install_clt.sh" | bash || exit 1

	[[ ! -e "/Library/Developer/CommandLineTools/usr/bin/git" ]] && echo "An error occurred with the CommandLineTools installation." && exit 1
fi

rm -rf "$installation_folder"
git clone --depth 1 https://github.com/arturonavax/env.git "$installation_folder"

cd "$installation_folder" && ./install.sh "$@"
