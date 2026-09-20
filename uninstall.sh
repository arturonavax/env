#!/bin/bash
# Run: ./uninstall.sh
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

# Go clean: golangci-lint cache clean ; sudo rm -rf "$(go env GOPATH)/pkg" ; go clean -x -cache -modcache -testcache -fuzzcache

function uninstall() {
	set -o errexit
	trap exit-error-message ERR SIGINT

	source ./src/remotes/_required_commands.sh

	if [[ "$(uname -s)" == "Linux" ]]; then
		required-commands curl
		required-sudo-commands apt

	elif [[ "$(uname -s)" == "Darwin" ]]; then
		required-commands brew curl
	fi

	echo -en "$fgcolor_white_bold"
	echo "[Uninstaller]: - Uninstalling ghostty, tmux, and neovim..."

	echo -en "$fgcolor_yellow_bold"

	# Ask for the administrator password upfront
	sudo -v

	# Keep-alive: update existing `sudo` time stamp until `install.sh` has finished
	while true; do
		sudo -n true
		sleep 60
		kill -0 "$$" || exit
	done 2>/dev/null &

	echo -e "${fgcolor_white_bold}[Uninstaller]: ${fgcolor_green_bold}Privileges for uninstallation obtained"
	echo -en "$fgcolor_reset"

	echo -en "$fgcolor_white_bold"
	echo "[Uninstaller]: - ... Phase 1/2"

	if [[ "$(uname -s)" == "Linux" ]]; then
		sudo apt remove -y --purge neovim fzf &>/dev/null || :
		sudo snap remove --purge ghostty &>/dev/null || :
		sudo snap remove --purge tmux &>/dev/null || :
		sudo snap remove --purge nvim &>/dev/null || :
		sudo rm -rf /opt/nvim-linux-* /usr/local/bin/nvim "$HOME/.local/bin/nvim" /usr/local/bin/tmux &>/dev/null || :

	elif [[ "$(uname -s)" == "Darwin" ]]; then
		brew uninstall --cask ghostty &>/dev/null || :
		brew uninstall tmux neovim fzf &>/dev/null || :
	fi

	echo "[Uninstaller]: - ... Phase 2/2"

	rm -rf ~/.config/ghostty ~/.config/nvim || :

	echo -e "[Uninstaller]: - ${fgcolor_green_bold}... Ready!${fgcolor_white_bold}"

	echo
	echo "[Uninstaller]: - Deleting configuration files and caches..."
	rm -rf ~/.config/ghostty &>/dev/null || :
	rm -rf ~/.config/nvim &>/dev/null || :
	rm -rf ~/.local/share/nvim &>/dev/null || :
	rm -rf ~/.local/state/nvim &>/dev/null || :
	rm -rf ~/.cache/nvim &>/dev/null || :
	rm -rf ~/.cache/tmux &>/dev/null || :
	rm -rf ~/.tmux &>/dev/null || :
	golangci-lint cache clean &>/dev/null || :
	echo -e "[Uninstaller]: - ${fgcolor_green_bold}... Ready!${fgcolor_white_bold}"

	echo
	echo -e "[Uninstaller]: ${fgcolor_green_bold}✔️ Uninstallation is complete${fgcolor_white_bold}"

	echo -en "$fgcolor_reset"
	echo
}

function exit-error-message() {
	echo -e "$(
		cat <<EOF

${fgcolor_white_bold}[Uninstaller Error]: ---
[Uninstaller Error]: ${fgcolor_red_bold}The installation had an error and was interrupted, the installation was not completed.${fgcolor_white_bold}
[Uninstaller Error]: ---${fgcolor_reset}
EOF
	)"

	exit 1
}

uninstall "$@"
