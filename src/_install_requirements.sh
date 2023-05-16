#!/bin/bash
# Installs the tools and languages needed to run the installation script.
# Automatically add to the PATH of the current session everything to be able to run the installation script in the same session.
# This script should not be executed with the "source" command because it requires a modification of the PATH in the current SHELL session.
#
# Run: source ./src/_install_requirements.sh
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

repo_remote_files="https://env.arturonavax.dev"

source ./src/remotes/_basics.sh
source ./src/remotes/_vars_colors.sh

echo -e "${fgcolor_white_bold}[Requirements Installer]: Starting _install_requirements.sh script...${fgcolor_reset}"

echo -e "${fgcolor_white_bold}[Requirements Installer]: - Installing basic dependencies...${fgcolor_reset}"

# dependencies
if [[ "$(uname -s)" == "Linux" ]]; then
	sudo apt update

	sudo apt install -y make build-essential libssl-dev zlib1g-dev \
		libbz2-dev libreadline-dev libsqlite3-dev wget curl llvm libncurses5-dev \
		libncursesw5-dev ncurses-term xz-utils tk-dev libffi-dev liblzma-dev

	# for Python GUIs
	sudo apt install -y python3-dev python3-tk tk-dev

	# tools
	sudo apt install -y curl wget git make unzip fontconfig snapd

elif [[ "$(uname -s)" == "Darwin" ]]; then
	# CommandLineTools
	./src/remotes/macos_install_clt.sh

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

	# for Python GUIs
	brew install python-tk

	# tools
	brew install curl wget git make unzip fontconfig ncurses openssl readline sqlite3 xz zlib
fi

echo

# ---

echo -e "${fgcolor_white_bold}[Requirements Installer]: - Installing pyenv (python version manager)...${fgcolor_reset}"

# downloads
## download pyenv
curl https://pyenv.run | bash || :

echo

# ---

echo -e "${fgcolor_white_bold}[Requirements Installer]: - Installing rust...${fgcolor_reset}"

## download rust
curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y --no-modify-path

echo

# ---

echo -e "${fgcolor_white_bold}[Requirements Installer]: - Installing fnm (node version manager)...${fgcolor_reset}"

## download fnm
curl -fsSL https://fnm.vercel.app/install | bash -s -- --skip-shell

echo

# ---

echo -e "${fgcolor_white_bold}[Requirements Installer]: - Installing pnpm (node package module)...${fgcolor_reset}"

## download pnpm
tmp_shell="$SHELL"
export SHELL="bash" # To prevent the following installer, write to the ~/.zshrc file
curl -fsSL https://get.pnpm.io/install.sh | sh -
export SHELL="$tmp_shell"

echo

# ---

echo -e "${fgcolor_white_bold}[Requirements Installer]: - Installing deno...${fgcolor_reset}"

## download deno
curl -fsSL https://deno.land/x/install/install.sh | sh

echo

# ---

echo -e "${fgcolor_white_bold}[Requirements Installer]: - Installing golang...${fgcolor_reset}"

## download go
curl -fsSL "$repo_remote_files/install_golang.sh" | bash -s -- -i

echo

# ---

echo -e "${fgcolor_white_bold}[Requirements Installer]: - Loading installations (source command)...${fgcolor_reset}"

# loads
function rehash() {
	hash -r "$@"
}

source ./files/zsh/.tools.sh || :

echo

# ---

echo -e "${fgcolor_white_bold}[Requirements Installer]: - Installing python 3...${fgcolor_reset}"

# installs
## install python
pyenv install -s 3
pyenv global 3

eval "$(pyenv init -)"

echo

# ---

echo -e "${fgcolor_white_bold}[Requirements Installer]: - Installing pipx (python binary installer)...${fgcolor_reset}"

## install pipx
python3 -m pip install --upgrade pip
python3 -m pip install --user pipx --force
python3 -m pipx ensurepath

echo

# ---

echo -e "${fgcolor_white_bold}[Requirements Installer]: - Installing nodejs...${fgcolor_reset}"

## install node
fnm install --latest

node_latest_label="$(fnm list | sort --version-sort | awk '{print $2}' | tail -n 1)"

fnm use "$node_latest_label"
fnm default "$node_latest_label"

echo

# ---

case "$SHELL" in
*zsh) shell_file=".zshrc" ;;
*tcsh) shell_file=".tcshrc" ;;
*ksh) shell_file=".kshrc" ;;
*csh) shell_file=".cshrc" ;;
*) shell_file=".bashrc" ;;
esac

echo -e "${fgcolor_white_bold}[Requirements Installer]: - Adding lines (~/$shell_file)...${fgcolor_reset}"

cp ./files/zsh/.tools.sh ~/.

touch ~/"$shell_file"

# add 'source ~/.tools.sh' command to the end of the file
./src/remotes/add_lines.sh tools

echo -e "${fgcolor_white_bold}[Requirements Installer]: ${fgcolor_green_bold}+ Requirements successfully installed!${fgcolor_reset}"
