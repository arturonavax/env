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
		required-commands curl cargo
		required-sudo-commands snap apt

	elif [[ "$(uname -s)" == "Darwin" ]]; then
		required-commands brew curl cargo
	fi

	echo -en "$fgcolor_white_bold"
	echo "[Uninstaller]: - Uninstalling alacritty, vim, neovim and lunarvim..."

	echo -en "$fgcolor_yellow_bold"
	sudo true
	echo -e "${fgcolor_white_bold}[Uninstaller]: ${fgcolor_green_bold}Privileges for uninstallation obtained"
	echo -en "$fgcolor_reset"

	echo -en "$fgcolor_white_bold"
	echo "[Uninstaller]: - ... Phase 1/3"

	if [[ "$(uname -s)" == "Linux" ]]; then
		sudo apt remove -y --purge alacritty vim neovim fzf &>/dev/null
		sudo snap remove --purge zsh &>/dev/null
		sudo snap remove --purge tmux &>/dev/null
		sudo snap remove --purge alacritty &>/dev/null
		sudo snap remove --purge vim-editor &>/dev/null
		sudo snap remove --purge nvim &>/dev/null

	elif [[ "$(uname -s)" == "Darwin" ]]; then
		brew uninstall alacritty neovim fzf &>/dev/null
	fi

	echo "[Uninstaller]: - ... Phase 2/3"

	cargo uninstall alacritty &>/dev/null || :

	bash <(curl -s https://raw.githubusercontent.com/lunarvim/lunarvim/master/utils/installer/uninstall.sh) &>/dev/null || :
	rm -rf ~/.config/lvim || :

	echo "[Uninstaller]: - ... Phase 3/3"
	echo -e "[Uninstaller]: - ${fgcolor_green_bold}... Ready!${fgcolor_white_bold}"

	echo
	echo "[Uninstaller]: - Deleting configuration files and caches..."
	rm -rf ~/.config/lvim &>/dev/null
	rm -rf ~/.config/nvim &>/dev/null
	rm -rf ~/.local/share/nvim &>/dev/null
	rm -rf ~/.local/share/lunarvim &>/dev/null
	rm -rf ~/.local/share/lvim &>/dev/null
	rm -rf ~/.local/bin/lvim &>/dev/null
	rm -rf ~/.cache/nvim &>/dev/null
	rm -rf ~/.cache/tmux &>/dev/null
	rm -rf ~/.cache/nvim.bak &>/dev/null
	rm -rf ~/.config/alacritty &>/dev/null
	rm -rf ~/.tmux &>/dev/null
	golangci-lint cache clean &>/dev/null
	echo "[Uninstaller]: - ... Phase 1/1"
	echo -e "[Uninstaller]: - ${fgcolor_green_bold}... Ready!${fgcolor_white_bold}"

	echo
	echo -e "[Uninstaller]: ${fgcolor_green_bold}+ Uninstallation is complete${fgcolor_white_bold}"

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
