#!/bin/bash
# Instalación headless de las CommandLineTools
# (Take this code from the Homebrew installer file)
#
# Run: curl -fsSL "https://env.arturonavax.dev/macos_install_clt.sh" | bash
clt_placeholder="/tmp/.com.apple.dt.CommandLineTools.installondemand.in-progress"

function install_clt() {
	[[ -e "/Library/Developer/CommandLineTools/usr/bin/git" ]] && echo "The CommandLineTools are already installed." && return 0

	sudo true || return 1

	echo "Searching for CommandLineTools label..."

	sudo touch "$clt_placeholder"

	label_clt="$(softwareupdate -l |
		grep -B 1 -E 'Command Line Tools' |
		awk -F'*' '/^ *\*/ {print $2}' |
		sed -e 's/^ *Label: //' -e 's/^ *//' |
		sort -V | tail -n1)" || return 1

	[[ "$label_clt" == "" ]] && echo "An error occurred while searching for the label" && return 1

	sudo softwareupdate -i "$label_clt" || return 1

	sudo rm -f "$clt_placeholder"

	sudo xcode-select --switch /Library/Developer/CommandLineTools &>/dev/null

	echo "CommandLineTools Installation completed."
}

install_clt || echo "Installation failed, retry or run the installation with GUI: xcode-select --install"
