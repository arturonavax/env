#!/bin/bash
# Run: ./install.sh help
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

function install() {
	echo -e "${fgcolor_white_bold}[Installer]: Running Backup script (~/.arturonavax-env-backups/)...${fgcolor_reset}"
	./utils/backup_config.sh &>/dev/null || :
	echo -e "${fgcolor_white_bold}[Installer]: Backup ready${fgcolor_reset}"

	set -o errexit
	trap exit-error-message ERR SIGINT

	# check arguments
	./src/remotes/usage_install.sh "$@"

	# get arguments
	for arg in "$@"; do
		case "$arg" in
		requirements | r) install_requirements=1 ;;
		fonts | f) install_fonts=1 ;;
		terminal | t) install_terminal=1 ;;
		editor | e) install_editor=1 ;;
		osconfig | o) install_osconfig=1 ;;
		all | a) install_all=1 ;;
		esac
	done

	[[ "$#" == 0 ]] && install_editor=1

	if [[ "$install_all" == 1 ]]; then
		install_requirements=1
		install_fonts=1
		install_terminal=1
		install_editor=1
		install_osconfig=1
	fi

	# ---

	echo -en "$fgcolor_yellow_bold"
	sudo true
	echo -e "${fgcolor_white_bold}[Installer]: ${fgcolor_green_bold}Installation privileges obtained"
	echo -en "$fgcolor_reset"

	# installs requirements
	if [[ "$install_requirements" == 1 ]]; then
		./src/requirements/requirements.sh

		source ./src/_install_requirements.sh
		echo
	fi

	source ./src/remotes/_required_commands.sh

	required-commands git

	echo -e "${fgcolor_white_bold}[Installer]: Downloading the latest version of the repository...${fgcolor_reset}"
	git pull origin main
	echo

	# check requireds install_fonts
	if [[ "$install_fonts" == 1 ]]; then
		required-commands curl
		required-sudo-commands fc-cache
	fi

	# check requireds install_terminal
	if [[ "$install_terminal" == 1 ]]; then
		./src/requirements/terminal.sh
	fi

	# check requireds install_editor
	if [[ "$install_editor" == 1 ]]; then
		./src/requirements/editor.sh
	fi

	# ---

	# installs
	if [[ "$install_fonts" == 1 ]]; then
		./src/remotes/install_fonts.sh
		echo
	fi

	if [[ "$install_terminal" == 1 ]]; then
		./src/install_terminal.sh
		echo
	fi

	if [[ "$install_editor" == 1 ]]; then
		./src/install_editor.sh
	fi

	if [[ "$install_osconfig" == 1 ]]; then
		echo
		echo -e "${fgcolor_white_bold}[Installer]: Applying system settings...${fgcolor_reset}"

		if [[ "$(uname -s)" == "Linux" ]]; then
			./src/remotes/linux_osconfig.sh

		elif [[ "$(uname -s)" == "Darwin" ]]; then
			./src/remotes/macos_osconfig.sh
		fi

		echo -e "${fgcolor_white_bold}[Installer]: ${fgcolor_green_bold}✔️ Applied configurations!${fgcolor_reset}"
	fi

	if [[ "$(uname -s)" == "Darwin" ]]; then
		echo
		echo -e "${fgcolor_white_bold}[Installer]: Homebrew cleanup...${fgcolor_reset}"

		brew cleanup

		echo -e "${fgcolor_white_bold}[Installer]: ${fgcolor_green_bold}✔️ Homebrew cleaned!${fgcolor_reset}"
	fi
}

function exit-error-message() {
	echo -e "$(
		cat <<EOF

${fgcolor_white_bold}[Installer Error]: ---
[Installer Error]: ${fgcolor_red_bold}The installation had an error and was interrupted, the installation was not completed.${fgcolor_white_bold}
[Installer Error]: ---${fgcolor_reset}
EOF
	)"

	exit 1
}

install "$@"
