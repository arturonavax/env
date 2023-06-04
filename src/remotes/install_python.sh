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
fgcolor_red='\033[0;31m'
fgcolor_red_bold='\033[1;31m'
fgcolor_green='\033[0;32m'
fgcolor_green_bold='\033[1;32m'
fgcolor_cyan_bold='\033[1;36m'

function exit-error-message() {
	echo
	echo -e "${fgcolor_white_bold}[Python Installer]: ${fgcolor_red_bold}The installation had an error and was interrupted, the installation was not completed.${fgcolor_reset}"

	exit 1
}

if [[ "$(command -v curl)" == "" ]]; then
	echo "${fgcolor_white_bold}[Python Installer]: ${fgcolor_red_bold}The curl command is needed to execute this script.${fgcolor_reset}"

	exit 1
fi

while getopts ':i' flag; do
	case "$flag" in
	i)
		install_flag=1
		;;
	*)
		echo -e "${fgcolor_white_bold}[Python Installer]: ${fgcolor_red_bold}Unknown flag${fgcolor_reset}, only ${fgcolor_green_bold}'-i'${fgcolor_reset} is accepted to install."
		exit 1
		;;
	esac
done

if [[ "$(uname -s)" == "Darwin" ]]; then
	if [[ ! -e "/Library/Developer/CommandLineTools/usr/bin/git" ]]; then
		# install CommandLineTools (this contains GIT)
		curl -fsSL "https://env.arturonavax.dev/macos_install_clt.sh" | bash || exit 1
	fi

	# Homebrew
	if [[ "$(command -v brew)" == "" ]]; then
		/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
	fi

	brewbin="/usr/local/bin/brew"
	[[ ! -f "$brewbin" ]] && brewbin="/opt/homebrew/bin/brew" # For Apple Silicon
	[[ -f "$brewbin" ]] && eval "$("$brewbin" shellenv)"

	if [[ "$(command -v brew)" == "" ]]; then
		echo "${fgcolor_white_bold}[Python Installer]: ${fgcolor_red}Homebrew installation error occurred." && exit 1
	fi
fi

echo -e "${fgcolor_white_bold}[Python Installer]: 🔎 Finding latest version of Python ${fgcolor_yellow_bold}🐍${fgcolor_white_bold}...${fgcolor_reset}"

export PYENV_ROOT="$HOME/.pyenv"

if [[ ":$PATH:" != *":$PYENV_ROOT/bin:"* ]]; then
	export PATH="$PYENV_ROOT/bin:$PATH"
fi

if [[ ! -d ~/.pyenv/ ]]; then
	echo -e "${fgcolor_white_bold}[Python Installer]: - Installing pyenv (python version manager)...${fgcolor_reset}"

	if [[ "$(uname -s)" == "Linux" ]]; then
		# download pyenv
		curl -fsSL https://pyenv.run | bash

	elif [[ "$(uname -s)" == "Darwin" ]]; then
		brew reinstall pyenv
	fi

	echo -e "${fgcolor_white_bold}[Python Installer]: ${fgcolor_green_bold}✔️ pyenv installation completed...${fgcolor_reset}"
	echo -e "${fgcolor_white_bold}[Python Installer]: ${fgcolor_reset}# Add pyenv to PATH: ${fgcolor_cyan}export PATH=\"\$HOME/.pyenv/bin:\$PATH\" && eval \"\$(pyenv init -)\"${fgcolor_reset}"
fi

eval "$(pyenv init -)"

latest_version="$(pyenv install -l | command grep -E -o '^[[:space:]]*[0-9]+(\.[0-9]+){1,2}$' |
	sort --version-sort | tail -n 1 | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')"

if [[ "$(command -v python3)" == "" ]]; then
	echo -e "${fgcolor_white_bold}[Python Installer]: ${fgcolor_red}❌ Python is not installed on this system..${fgcolor_reset}"

else
	current_version="$(python3 --version | awk '{print $2}')"

	if ! command -v python3 | grep -q ".pyenv"; then
		echo -e "${fgcolor_white_bold}[Python Installer]: ${fgcolor_yellow_bold}⚠️ The installed Python was not installed with pyenv.${fgcolor_reset}"
	fi

	if [[ "$latest_version" != "$current_version" ]]; then
		echo -e "${fgcolor_white_bold}[Python Installer]: ${fgcolor_yellow_bold}⚠️ The version of Python installed is not the latest one.${fgcolor_reset}"
		echo -e "${fgcolor_white_bold}[Python Installer]: ${fgcolor_yellow_bold}ℹ️ The latest available version of Python is: $latest_version${fgcolor_reset}"

	else
		echo -e "${fgcolor_white_bold}[Python Installer]: ${fgcolor_green}✔️ You have the latest version of Python: $latest_version${fgcolor_reset}"
	fi
fi

if [[ "$install_flag" == 1 ]]; then
	echo
	echo -e "${fgcolor_white_bold}[Python Installer]: ${fgcolor_cyan_bold}⚡ Running installation...${fgcolor_reset}"

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

			sudo apt install -y build-essential gcc llvm make git wget curl findutils libssl-dev zlib1g-dev \
				libbz2-dev libffi-dev liblzma-dev libxml2 libxml2-dev libxslt1-dev \
				libreadline-dev libsqlite3-dev libncurses5-dev libncursesw5-dev ncurses-term \
				libpq-dev xz-utils tk tk-dev python3-dev python3-venv python3-wheel python3-setuptools python3-tk python3-openssl

		elif [[ "$ID_LIKE" == *"rhel"* || "$ID_LIKE" == *"centos"* || "$ID_LIKE" == *"fedora"* || "$ID" == *"fedora"* ]]; then
			sudo dnf update -y

			sudo dnf group install -y "Development Tools"
			sudo dnf install -y gcc make git wget curl libffi libffi-devel qt5-qtbase-devel bzip2 bzip2-devel findutils \
				ncurses ncurses-devel ncurses-term sqlite sqlite-devel readline readline-devel zlib zlib-devel openssl openssl-devel \
				libxml2 libxml2-devel libpq-devel xz xz-devel tk tk-devel python3-devel python3-wheel python3-setuptools
		else
			echo "${fgcolor_white_bold}[Python Installer]: ${fgcolor_red_bold}❌ The operating system is not compatible with this installation."

			exit 1
		fi

	elif [[ "$(uname -s)" == "Darwin" ]]; then
		brew install curl wget git make gcc ncurses findutils sqlite3 openssl readline zlib xz tcl-tk python3 python-tk

	else
		echo "${fgcolor_white_bold}[Python Installer]: ${fgcolor_red_bold}❌ The operating system is not compatible with this installation."

		exit 1
	fi

	echo

	# Install python if the current python is not from pyenv
	echo -e "${fgcolor_white_bold}[Python Installer]: - Installing Python $latest_version...${fgcolor_reset}"

	if ! command -v python3 | grep -q ".pyenv" || [[ "$latest_version" != "$current_version" ]]; then
		# install python
		if [[ "$(uname -s)" == "Linux" ]]; then
			pyenv install -s "$latest_version"

		elif [[ "$(uname -s)" == "Darwin" ]]; then
			# env \
			# 	PATH="$(brew --prefix tcl-tk)/bin:$PATH" \
			# 	LDFLAGS="-L$(brew --prefix tcl-tk)/lib" \
			# 	CPPFLAGS="-I$(brew --prefix tcl-tk)/include" \
			# 	PKG_CONFIG_PATH="$(brew --prefix tcl-tk)/lib/pkgconfig" \
			# 	CFLAGS="-I$(brew --prefix tcl-tk)/include" \
			# 	PYTHON_CONFIGURE_OPTS="--enable-framework --with-tcltk-includes='-I$(brew --prefix tcl-tk)/include' --with-tcltk-libs='-L$(brew --prefix tcl-tk)/lib -ltcl8.6 -ltk8.6' " \
			pyenv install -s "$latest_version"

		else
			echo "${fgcolor_white_bold}[Python Installer]: ${fgcolor_red_bold}❌ The operating system is not compatible with this installation."

			exit 1
		fi

		pyenv global "$latest_version"

		eval "$(pyenv init -)"

		python3 -m pip install --upgrade pip
		python3 -m pip install --upgrade tk

		echo -e "${fgcolor_white_bold}[Python Installer]: ${fgcolor_green_bold}✔️ Python $latest_version installation completed...${fgcolor_reset}"
		echo -e "${fgcolor_white_bold}[Python Installer]: ${fgcolor_reset}# Add pyenv to PATH: ${fgcolor_cyan}export PATH=\"\$HOME/.pyenv/bin:\$PATH\" && eval \"\$(pyenv init -)\"${fgcolor_reset}"
		echo

	else
		echo -e "${fgcolor_white_bold}[Python Installer]: ${fgcolor_green_bold}✔️ Python $latest_version is already installed...${fgcolor_reset}"
		echo
	fi

	echo -e "${fgcolor_white_bold}[Python Installer]: - Installing pipx (python binary installer)...${fgcolor_reset}"

	if [[ "$(command -v pipx)" == "" ]] && command -v python3 | grep -q ".pyenv"; then
		# install pipx
		python3 -m pip install --upgrade pip
		python3 -m pip install --force --upgrade --user pipx
		python3 -m pipx ensurepath

		echo -e "${fgcolor_white_bold}[Python Installer]: ${fgcolor_green_bold}✔️ pipx installation completed (add ~/.local/bin to PATH)...${fgcolor_reset}"
		echo -e "${fgcolor_white_bold}[Python Installer]: ${fgcolor_reset}# Add ~/.local/bin to PATH: ${fgcolor_cyan}export PATH=\"\$HOME/.local/bin:\$PATH\"${fgcolor_reset}"

	else
		echo -e "${fgcolor_white_bold}[Python Installer]: ${fgcolor_green_bold}✔️ pip is already installed...${fgcolor_reset}"
	fi
fi
