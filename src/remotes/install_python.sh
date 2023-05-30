#!/bin/bash
# Run: bash <(curl -fsSL "https://env.arturonavax.dev/install_python.sh")
# Flag to install update available: -i
set -o errexit
trap exit-error-message ERR SIGINT

# Reset Text Color
fgcolor_reset='\033[0m'

fgcolor_cyan='\033[0;36m'

# Bold Text Colors
fgcolor_white_bold='\033[1;37m'
fgcolor_yellow_bold='\033[1;33m'
fgcolor_red_bold='\033[1;31m'
fgcolor_green_bold='\033[1;32m'

function exit-error-message() {
	echo
	echo -e "${fgcolor_red_bold}The installation had an error and was interrupted, the installation was not completed.${fgcolor_reset}"

	exit 1
}

[[ "$(command -v curl)" == "" ]] && echo "The curl command is needed to execute this installation." && exit 1

while getopts ':i' flag; do
	case "$flag" in
	i)
		install_flag=1
		;;
	*)
		echo -e "${fgcolor_red_bold}Unknown flag${fgcolor_reset}, only ${fgcolor_green_bold}'-i'${fgcolor_reset} is accepted to install."
		exit 1
		;;
	esac
done

echo -e "${fgcolor_white_bold}[Python Installer]: Starting install_python.sh script...${fgcolor_reset}"

if [[ "$(command -v pyenv)" == "" ]]; then
	echo -e "${fgcolor_white_bold}[Python Installer]: - Installing pyenv (python version manager)...${fgcolor_reset}"

	if [[ "$(uname -s)" == "Darwin" ]]; then
		# install CommandLineTools (this contains GIT)
		curl -fsSL "https://env.arturonavax.dev/macos_install_clt.sh" | bash || exit 1
	fi

	# download pyenv
	curl -fsSL https://pyenv.run | bash || :

	export PYENV_ROOT="$HOME/.pyenv"
	export PATH="$PYENV_ROOT/bin:$PATH"

	if [[ -d ~/.pyenv ]]; then
		eval "$(pyenv init -)"
	fi

	echo -e "${fgcolor_white_bold}[Python Installer]: pyenv installation completed (add Pyenv to PATH)...${fgcolor_reset}"

	echo
fi

latest_version="$(pyenv install -l | command grep -E -o '^[[:space:]]*[0-9]+(\.[0-9]+){1,2}$' |
	sort --version-sort | tail -n 1 | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')"

current_version="$(python3 --version | awk '{print $2}')"

if [[ "$(command -v python3)" == "" ]]; then
	echo -e "${fgcolor_white_bold}[Python Installer]: ${fgcolor_yellow_bold}!!! Python is not installed.${fgcolor_reset}"

elif ! command -v python3 | grep -q ".pyenv"; then
	echo -e "${fgcolor_white_bold}[Python Installer]: ${fgcolor_yellow_bold}!!! The installed Python was not installed with Pyenv.${fgcolor_reset}"
fi

if [[ "$latest_version" != "$current_version" ]]; then
	echo -e "${fgcolor_white_bold}[Python Installer]: ${fgcolor_yellow_bold}!!! The version of Python installed is not the latest one ($latest_version).${fgcolor_reset}"

else
	echo -e "${fgcolor_white_bold}[Python Installer]: ${fgcolor_green_bold}You already have the latest version of Python ($latest_version).${fgcolor_reset}"
fi

if [[ "$install_flag" == 1 ]]; then
	echo -e "${fgcolor_white_bold}[Python Installer]: - Installing dependencies...${fgcolor_reset}"

	# dependencies
	if [[ "$(uname -s)" == "Linux" ]]; then
		source /etc/os-release

		if [[ "$ID_LIKE" == *"rhel"* || "$ID_LIKE" == *"centos"* ]]; then
			sudo dnf update -y
			sudo dnf install -y epel-release
		fi

		if [[ "$ID_LIKE" == *"debian"* || "$ID_LIKE" == *"ubuntu"* ]]; then
			sudo apt update -y

			sudo apt install -y build-essential gcc make git wget curl libssl-dev zlib1g-dev \
				libbz2-dev tk tk-dev libffi-dev liblzma-dev libxml libxml2-dev libxslt1-dev \
				libreadline-dev libsqlite3-dev libncurses5-dev libncursesw5-dev ncurses-term \
				libpq-dev python3-dev python3-venv python3-wheel python3-setuptools python3-tk

		elif [[ "$ID_LIKE" == *"rhel"* || "$ID_LIKE" == *"centos"* || "$ID_LIKE" == *"fedora"* || "$ID" == *"fedora"* ]]; then
			sudo dnf update -y

			sudo dnf group install -y "Development Tools"
			sudo dnf install -y gcc make git wget curl libffi llibffi-devel qt5-qtbase-devel \
				ncurses ncurses-devel ncurses-term sqlite sqlite-devel readline readline-devel \
				libxml2 libxml2-devel libpq-devel python3-devel python3-wheel python3-setuptools tk tk-devel

		else
			echo "The operating system is not compatible with this installation." && exit 1
		fi

	elif [[ "$(uname -s)" == "Darwin" ]]; then
		# install CommandLineTools (this contains GIT)
		curl -fsSL "https://env.arturonavax.dev/macos_install_clt.sh" | bash || exit 1

		# rosetta2
		softwareupdate --install-rosetta --agree-to-license || :

		# Homebrew
		if [[ "$(command -v brew)" == "" ]]; then
			/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
		fi

		brewbin="/usr/local/bin/brew"
		[[ ! -f "$brewbin" ]] && brewbin="/opt/homebrew/bin/brew" # For Apple Silicon
		[[ -f "$brewbin" ]] && eval "$("$brewbin" shellenv)"

		if [[ "$(command -v brew)" == "" ]]; then
			echo "!!! Homebrew installation error occurred." && exit 1
		fi

		brew install curl wget git make gcc ncurses sqlite3 openssl readline zlib xz tcl-tk python3 python-tk

	else
		echo "The operating system is not compatible with this installation." && exit 1
	fi

	echo

	# Install python if the current python is not from pyenv
	echo -e "${fgcolor_white_bold}[Python Installer]: - Installing Python $latest_version...${fgcolor_reset}"
	if ! command -v python3 | grep -q ".pyenv" || ! pyenv doctor &>/dev/null || [[ "$latest_version" != "$current_version" ]]; then

		# install python
		if [[ "$(uname -s)" == "Darwin" ]]; then
			env \
				PATH="$(brew --prefix tcl-tk)/bin:$PATH" \
				LDFLAGS="-L$(brew --prefix tcl-tk)/lib" \
				CPPFLAGS="-I$(brew --prefix tcl-tk)/include" \
				PKG_CONFIG_PATH="$(brew --prefix tcl-tk)/lib/pkgconfig" \
				CFLAGS="-I$(brew --prefix tcl-tk)/include" \
				PYTHON_CONFIGURE_OPTS="--enable-framework --with-tcltk-includes='-I$(brew --prefix tcl-tk)/include' --with-tcltk-libs='-L$(brew --prefix tcl-tk)/lib -ltcl8.6 -ltk8.6' " \
				pyenv install -s "$latest_version"

		else
			pyenv install -s "$latest_version"
		fi

		pyenv global "$latest_version"

		eval "$(pyenv init -)"

		python3 -m pip install --upgrade pip
		python3 -m pip install --upgrade tk

		echo -e "${fgcolor_white_bold}[Python Installer]: Python $latest_version installation completed...${fgcolor_reset}"
		echo -e "${fgcolor_white_bold}[Python Installer]: Add Pyenv to PATH: ${fgcolor_cyan}export PATH=\"\$HOME/.pyenv/bin:\$PATH\" && eval \"\$(pyenv init -)\"${fgcolor_reset}"

		echo
	fi

	if [[ "$(command -v pipx)" == "" ]] && command -v python3 | grep -q ".pyenv"; then
		echo -e "${fgcolor_white_bold}[Python Installer]: - Installing pipx (python binary installer)...${fgcolor_reset}"

		# install pipx
		python3 -m pip install --upgrade pip
		python3 -m pip install --force --upgrade --user pipx
		python3 -m pipx ensurepath

		echo -e "${fgcolor_white_bold}[Python Installer]: pipx installation completed (add ~/.local/bin to PATH)...${fgcolor_reset}"
		echo -e "${fgcolor_white_bold}[Python Installer]: Add ~/.local/bin to PATH: ${fgcolor_cyan}export PATH=\"\$HOME/.local/bin:\$PATH\"${fgcolor_reset}"

		echo
	fi
fi
