#!/bin/bash
# Run: curl -fsSL "https://env.arturonavax.dev/install_fonts.sh" | bash
repo_remote_files="https://env.arturonavax.dev"

# shellcheck disable=SC1090
if [[ -f ./src/remotes/_vars_colors.sh ]]; then
	source ./src/remotes/_vars_colors.sh
elif [[ -f "$(dirname "$0")/_vars_colors.sh" ]]; then
	source "$(dirname "$0")/_vars_colors.sh"
else
	source <(curl -fsSL "$repo_remote_files/_vars_colors.sh" | cat) || :
fi

if [[ -f ./src/remotes/_versions.sh ]]; then
	source ./src/remotes/_versions.sh
elif [[ -f "$(dirname "$0")/_versions.sh" ]]; then
	source "$(dirname "$0")/_versions.sh"
fi

# shellcheck disable=SC2154
function install_fonts() {
	set -o errexit
	trap exit-error-message ERR SIGINT

	echo -e "${fgcolor_white_bold}[Fonts Installer]: Installing patched mono fonts...${fgcolor_reset}"

	original_folder="$(pwd)"

	tmp_dir="$(mktemp -d)"

	cd "$tmp_dir"

	# Download Monaspace nerd font
	repo_font="ryanoasis/nerd-fonts"
	font_version="${NERD_FONTS_VERSION:-v3.5.1}"

	download_url="https://github.com/$repo_font/releases/download/${font_version}/Monaspace.tar.xz"

	mkdir -p "NerdMonaspace"
	curl -fsSL -o "Monaspace.tar.xz" "$download_url"

	tar -xJf "Monaspace.tar.xz" -C "NerdMonaspace"

	if [[ "$(uname -s)" == "Linux" ]]; then
		sudo mkdir -p /usr/local/share/fonts/patched-fonts
		sudo cp ./NerdMonaspace/*.otf /usr/local/share/fonts/patched-fonts/ 2>/dev/null || sudo cp ./NerdMonaspace/* /usr/local/share/fonts/patched-fonts/ 2>/dev/null || :
	elif [[ "$(uname -s)" == "Darwin" ]]; then
		sudo mkdir -p /Library/Fonts/patched-fonts
		sudo cp ./NerdMonaspace/*.otf /Library/Fonts/patched-fonts/ 2>/dev/null || sudo cp ./NerdMonaspace/* /Library/Fonts/patched-fonts/ 2>/dev/null || :
	else
		echo "The operating system is not compatible with this installation." && exit 1
	fi

	cd "$original_folder"
	rm -rf "$tmp_dir"

	if [[ "$(uname -s)" == "Linux" ]]; then
		sudo fc-cache -f &>/dev/null || :
	fi

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
