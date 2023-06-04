#!/bin/bash
# Run: curl -fsSL "https://env.arturonavax.dev/install_fonts.sh" | bash
repo_remote_files="https://env.arturonavax.dev"

# shellcheck disable=SC1090
source <(curl -fsSL "$repo_remote_files/_vars_colors.sh" | cat)

# shellcheck disable=SC2154
function install_fonts() {
	set -o errexit
	trap exit-error-message ERR SIGINT

	echo -e "${fgcolor_white_bold}[Fonts Installer]: Installing patched mono fonts...${fgcolor_reset}"

	tmp_dir="$(mktemp -d)"

	cd "$tmp_dir"

	# CaskaydiaCove Nerd Font Mono - regular
	curl -fsLO "https://github.com/ryanoasis/nerd-fonts/raw/HEAD/patched-fonts/CascadiaCode/Regular/CaskaydiaCoveNerdFontMono-Regular.ttf"
	curl -fsLO "https://github.com/ryanoasis/nerd-fonts/raw/HEAD/patched-fonts/CascadiaCode/Regular/CaskaydiaCoveNerdFontMono-Italic.ttf"

	# CaskaydiaCove Nerd Font Mono - bold
	curl -fsLO "https://github.com/ryanoasis/nerd-fonts/raw/HEAD/patched-fonts/CascadiaCode/Bold/CaskaydiaCoveNerdFontMono-Bold.ttf"
	curl -fsLO "https://github.com/ryanoasis/nerd-fonts/raw/HEAD/patched-fonts/CascadiaCode/Bold/CaskaydiaCoveNerdFontMono-BoldItalic.ttf"

	if [[ "$(uname -s)" == "Linux" ]]; then
		sudo mkdir -p /usr/local/share/fonts/patched-fonts

		sudo cp ./* /usr/local/share/fonts/patched-fonts/.

	elif [[ "$(uname -s)" == "Darwin" ]]; then
		# brew tap homebrew/cask-fonts
		# brew install --cask font-caskaydia-cove-nerd-font

		sudo mkdir -p /Library/Fonts/patched-fonts

		sudo cp ./* /Library/Fonts/patched-fonts/.

	else
		echo "The operating system is not compatible with this installation." && exit 1
	fi

	cd -

	sudo fc-cache -f &>/dev/null || :

	echo -e "${fgcolor_white_bold}[Fonts Installer]: ${fgcolor_green_bold}✔️ Successfully installed fonts!${fgcolor_reset}"
}

# shellcheck disable=SC2154
function exit-error-message() {
	echo -e "$(
		cat <<EOF

${fgcolor_white_bold}[Installer Fonts Error]: ---
[Installer Fonts Error]: ${fgcolor_red_bold}The installation had an error and was interrupted, the installation was not completed.${fgcolor_white_bold}
[Installer Fonts Error]: ---${fgcolor_reset}
EOF
	)"

	exit 1
}

install_fonts "$@"
