#!/bin/bash
# This is a script made to run individually, it looks for the latest version of Go and
# compares it with the one currently installed.
#
# Run: bash <(curl -fsSL "https://env.arturonavax.dev/install_golang.sh")
# Flag to install update available: -i
# Flag to compiling from sources: -c
set -o errexit
trap exit-error-message ERR SIGINT

url_webscraping="https://go.dev/dl/"

installation_dirpath="/usr/local/"
bootstrap_dirpath="${installation_dirpath}go-bootstrap/"

# Reset Text Color
fgcolor_reset='\033[0m'

# Regular Text Colors
fgcolor_green='\033[0;32m'
fgcolor_red='\033[0;31m'
fgcolor_yellow='\033[0;33m'
fgcolor_cyan='\033[0;36m'

# Bold Text Colors
fgcolor_white_bold='\033[1;37m'
fgcolor_yellow_bold='\033[1;33m'
fgcolor_cyan_bold='\033[1;36m'
fgcolor_red_bold='\033[1;31m'
fgcolor_green_bold='\033[1;32m'

function exit-error-message() {
	echo
	echo -e "${fgcolor_white_bold}[Golang Installer]: ${fgcolor_red_bold}The installation had an error and was interrupted, the installation was not completed.${fgcolor_reset}"

	exit 1
}

if [[ "$(command -v wget)" == "" ]]; then
	echo "${fgcolor_white_bold}[Golang Installer]: ${fgcolor_red_bold}The wget command is needed to run this script.${fgcolor_reset}"

	exit 1
fi

while getopts ':ic' flag; do
	case "$flag" in
	i)
		install_flag=1
		;;
	c)
		compile_flag=1
		;;
	*)
		echo -e "${fgcolor_white_bold}[Golang Installer]: ${fgcolor_red_bold}Unknown flag${fgcolor_reset}, only ${fgcolor_green_bold}'-i'${fgcolor_reset} is accepted to install in case you don't have Golang installed on your system or find an update available, and ${fgcolor_green_bold}'-c'${fgcolor_reset} to perform the process by compiling from sources"
		exit 1
		;;
	esac
done

echo -e "${fgcolor_white_bold}[Golang Installer]: 🔎 Finding latest version of Golang ${fgcolor_cyan_bold}🐹${fgcolor_white_bold}...${fgcolor_reset}"

latest_version="$(wget -qO- "$url_webscraping" | command grep -E "/go1(\.[0-9]+){0,2}" | command grep -v "[0-9]rc" | command grep -E -o "/go1(\.[0-9]+){0,2}" | sort --version-sort | tail -n 1 | cut -d "/" -f 2)"

os=$(uname -s | tr '[:upper:]' '[:lower:]')

if [[ "$(uname -s)" == "Linux" ]]; then
	if [[ "$(uname -p)" == "x86_64" || "$(uname -p)" == "unknown" ]]; then
		arch="amd64"

	elif [[ "$(uname -p)" == "i386" ]]; then
		arch="386"

	elif [[ "$(uname -p)" == "arm64" ]]; then
		arch="arm64"

	elif [[ "$(uname -p)" == "armv6l" ]]; then
		arch="armv6l"

	elif [[ "$(uname -p)" == "ppc64le" ]]; then
		arch="ppc64le"

	elif [[ "$(uname -p)" == "s390x" ]]; then
		arch="s390x"

	else
		echo "${fgcolor_white_bold}[Golang Installer]: ${fgcolor_red_bold}❌ The operating system is not compatible with this installation."

		exit 1
	fi

elif [[ "$(uname -s)" == "Darwin" ]]; then
	if [[ "$(sysctl -n machdep.cpu.brand_string)" == *"Apple"* ]]; then
		arch="arm64"
	else
		arch="amd64"
	fi

else
	echo "${fgcolor_white_bold}[Golang Installer]: ${fgcolor_red_bold}❌ The operating system is not compatible with this installation."

	exit 1
fi

go_latest_filename="${latest_version}.${os}-${arch}.tar.gz"
go_latest_url="${url_webscraping}${go_latest_filename}"

case "$SHELL" in
*zsh) shell_file=".zshrc" ;;
*tcsh) shell_file=".tcshrc" ;;
*ksh) shell_file=".kshrc" ;;
*csh) shell_file=".cshrc" ;;
*) shell_file=".bashrc" ;;
esac

# eval_installation Print the installation line to be executed with "eval".
#  $1 = Installation path
function eval_installation() {
	[[ "$1" == "" ]] && return

	echo -n "rm -f ${go_latest_filename} ; wget ${go_latest_url} && sudo rm -rf ${1}go && sudo tar -C ${1} -xzf ${go_latest_filename} && rm -f ${go_latest_filename}"
}

line_installation=$(
	cat <<EOF
 ${fgcolor_white_bold}[Golang Installer]: ${fgcolor_yellow}⚡ Installation command (You still have to set up your PATH):${fgcolor_reset}
  ${fgcolor_yellow}$ ${fgcolor_cyan}$(eval_installation "$installation_dirpath")${fgcolor_reset}
EOF
)

line_example=$(
	cat <<EOF
 ${fgcolor_white_bold}[Golang Installer]: ${fgcolor_yellow}# Example for adding "go" and "GOPATH/bin" to your PATH:
  ${fgcolor_yellow}$ ${fgcolor_cyan}echo 'export PATH="${installation_dirpath}go/bin:\$HOME/go/bin:\$PATH"' >> ~/${shell_file}${fgcolor_reset}
EOF
)

function compiled_installation() {
	echo -e "${fgcolor_white_bold}[Golang Installer]: Getting bootstrap...${fgcolor_reset}"
	echo

	sudo mkdir -p "$bootstrap_dirpath"

	eval "$(eval_installation "$bootstrap_dirpath")" || exit 1

	if [[ -d "${installation_dirpath}go/.git/" ]]; then
		cd "${installation_dirpath}go/"

		sudo git pull

		sudo git checkout "$latest_version"

		sudo rm -f "${installation_dirpath}go/bin/*"

		cd "${installation_dirpath}go/src/"

		sudo su root --login --command "cd ${installation_dirpath}go/src/; GOROOT_BOOTSTRAP=${bootstrap_dirpath}go; ./all.bash"
	else
		sudo rm -rf "${installation_dirpath}go/"

		echo -e "${fgcolor_white_bold}[Golang Installer]: Cloning Golang sources...${fgcolor_reset}"
		echo

		sudo git clone https://go.googlesource.com/go "${installation_dirpath}go"

		cd "${installation_dirpath}go/"

		sudo git checkout "$latest_version"

		sudo rm -f "${installation_dirpath}go/bin/*"

		cd "${installation_dirpath}go/src/"

		sudo su root --login --command "cd ${installation_dirpath}go/src/; export GOROOT_BOOTSTRAP=${bootstrap_dirpath}go; ./all.bash"
	fi
}

if [[ "$(command -v go)" == "" ]]; then
	echo -e "${fgcolor_white_bold}[Golang Installer]: ${fgcolor_red}❌ Golang is not installed on this system.${fgcolor_reset}"
	echo -e "${fgcolor_white_bold}[Golang Installer]: ${fgcolor_yellow}ℹ️ The latest available version of Golang is: ${fgcolor_yellow_bold}${latest_version#go}${fgcolor_reset}"
	echo

	if [[ "$install_flag" == 1 ]]; then
		echo -e "${fgcolor_white_bold}[Golang Installer]: ${fgcolor_cyan_bold}⚡ Running installation... (in ${fgcolor_white_bold}${installation_dirpath}${fgcolor_cyan_bold})${fgcolor_reset}"
		echo

		if [[ "$compile_flag" == 1 ]]; then
			echo -e "${fgcolor_white_bold}(!!! Compiling from sources)${fgcolor_reset}"
			echo

			compiled_installation
		else
			sudo mkdir -p "$installation_dirpath"

			eval "$(eval_installation "$installation_dirpath")"
		fi

		echo -e "${fgcolor_white_bold}[Golang Installer]: ${fgcolor_green_bold}✔️ Installation completed.${fgcolor_reset}"
	else
		echo -e "$line_installation"
	fi

	echo
	echo -e "$line_example"

	exit 0
fi

current_version="$("$installation_dirpath/go/bin/go" version | cut -d " " -f 3)"

if [[ "$latest_version" != "$current_version" ]]; then
	echo -e "${fgcolor_white_bold}[Golang Installer]: ${fgcolor_yellow}⚠️ There is a different version of Golang: ${current_version}"
	echo -e "${fgcolor_white_bold}[Golang Installer]: ${fgcolor_yellow_bold}ℹ️ Latest new version available: ${latest_version#go}${fgcolor_reset}"
	echo

	if [[ "$install_flag" == 1 ]]; then
		echo -e "${fgcolor_white_bold}[Golang Installer]: ${fgcolor_cyan_bold}⚡ Running updating... (in ${fgcolor_white_bold}${installation_dirpath}${fgcolor_cyan_bold})${fgcolor_reset}"
		echo

		if [[ "$compile_flag" == 1 ]]; then
			echo -e "${fgcolor_white_bold}(!!! Compiling from sources)${fgcolor_reset}"
			echo

			compiled_installation
		else
			sudo mkdir -p "$installation_dirpath"

			eval "$(eval_installation "$installation_dirpath")"
		fi

		echo -e "${fgcolor_white_bold}[Golang Installer]: ${fgcolor_green_bold}✔️ Updating completed.${fgcolor_reset}"
	else
		echo -e "$line_installation"
	fi
else
	echo -e "${fgcolor_white_bold}[Golang Installer]: ${fgcolor_green}✔️ You have the latest version of Golang: ${current_version#go}${fgcolor_reset}"
fi
