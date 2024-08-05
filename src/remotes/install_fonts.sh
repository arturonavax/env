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

	original_folder="$(pwd)"

	tmp_dir="$(mktemp -d)"

	cd "$tmp_dir"

	# Download Monaspace nerd font
	repo_font="ryanoasis/nerd-fonts"
	asset_name="Monaspace"

	latest_release=$(curl -s "https://api.github.com/repos/$repo_font/releases/latest")
	tag_name=$(echo "$latest_release" | grep '"tag_name":' | awk -F'"' '{print $4}')

	asset_folder="$asset_name$tag_name"

	if [ -z "$tag_name" ]; then
		echo "The latest version could not be obtained. Please check the GitHub API response." && exit 1
	fi

	download_url="https://github.com/$repo_font/releases/download/$tag_name/${asset_name}.zip"

	curl -sL -o "Nerd${asset_folder}.zip" "$download_url"

	unzip -q "Nerd${asset_folder}.zip" -d "Nerd${asset_folder}"

	cd "Nerd$asset_folder"

	if [[ "$(uname -s)" == "Linux" ]]; then
		sudo mkdir -p /usr/local/share/fonts/patched-fonts

		sudo cp ./* /usr/local/share/fonts/patched-fonts/.

	elif [[ "$(uname -s)" == "Darwin" ]]; then
		sudo mkdir -p /Library/Fonts/patched-fonts

		sudo cp ./* /Library/Fonts/patched-fonts/.

	else
		echo "The operating system is not compatible with this installation." && exit 1
	fi

	cd ..

	# Download Monaspace font
	repo_font="githubnext/monaspace"
	asset_name="monaspace-"

	latest_release=$(curl -s "https://api.github.com/repos/$repo_font/releases/latest")
	tag_name=$(echo "$latest_release" | grep '"tag_name":' | awk -F'"' '{print $4}')

	asset_folder="$asset_name$tag_name"

	if [ -z "$tag_name" ]; then
		echo "The latest version could not be obtained. Please check the GitHub API response." && exit 1
	fi

	download_url="https://github.com/$repo_font/releases/download/$tag_name/${asset_folder}.zip"

	curl -sL -o "${asset_folder}.zip" "$download_url"

	unzip -q "${asset_folder}.zip"

	cd "$asset_folder"

	if [[ "$(uname -s)" == "Linux" ]]; then
		./util/install_linux.sh

	elif [[ "$(uname -s)" == "Darwin" ]]; then
		./util/install_macos.sh

	else
		echo "The operating system is not compatible with this installation." && exit 1
	fi

	cd "$original_folder"

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
