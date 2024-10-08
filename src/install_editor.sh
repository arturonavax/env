#!/bin/bash
# Run: ./src/install_editor.sh
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

./src/requirements/editor.sh || exit 1

source ./src/remotes/_vars_colors.sh

editor="lvim"

# install_editor install editor
function install_editor() {
	if [[ -f ~/.lunarvim.lua ]]; then
		mv ~/.lunarvim.lua ~/.lunarvim.copy.lua
	fi

	set -o errexit
	trap exit-error-message ERR SIGINT

	rm -rf ./downloads/
	mkdir ./downloads/

	source ./src/remotes/_basics.sh

	./src/remotes/add_lines.sh basics

	echo -e "${fgcolor_white_bold}[Editor Installer]: Starting install_editor.sh script...${fgcolor_reset}"

	# npm or pnpm
	if [[ "$(command -v npm)" != "" ]]; then
		function node_package_module() {
			npm "$@"
		}
	fi

	if [[ "$(command -v pnpm)" != "" ]]; then
		echo -e "${fgcolor_white_bold}[Editor Installer]: - - pnpm is used instead of npm${fgcolor_reset}"

		function node_package_module() {
			pnpm "$@"
		}
	fi

	# python3 -m pip or pipx
	if python3 -m pip &>/dev/null; then
		function python_binary_installer() {
			python3 -m pip install --user --upgrade "$@"
		}
	fi

	if [[ "$(command -v pipx)" != "" ]]; then
		echo -e "${fgcolor_white_bold}[Editor Installer]: - - pipx is used instead of 'python3 -m pip'${fgcolor_reset}"

		function python_binary_installer() {
			pipx install "$@"
		}
	fi

	echo -e "${fgcolor_white_bold}[Editor Installer]: - Installing tools and dependencies...${fgcolor_reset}"

	if [[ "$(uname -s)" == "Linux" ]]; then
		source /etc/os-release

		if [[ "$ID_LIKE" == *"rhel"* || "$ID_LIKE" == *"centos"* ]]; then
			sudo dnf update -y
			sudo dnf install -y epel-release
		fi

		if [[ "$ID_LIKE" == *"debian"* || "$ID_LIKE" == *"ubuntu"* ]]; then
			# Dependencies and keys for vscode
			sudo apt-get -y install wget gpg
			wget -qO- https://packages.microsoft.com/keys/microsoft.asc | gpg --dearmor >packages.microsoft.gpg
			sudo install -D -o root -g root -m 644 packages.microsoft.gpg /etc/apt/keyrings/packages.microsoft.gpg
			sudo sh -c 'echo "deb [arch=amd64,arm64,armhf signed-by=/etc/apt/keyrings/packages.microsoft.gpg] https://packages.microsoft.com/repos/code stable main" > /etc/apt/sources.list.d/vscode.list'
			rm -f packages.microsoft.gpg

			sudo apt update -y

			sudo apt install -y apt-transport-https curl git xclip xsel ripgrep clang-format vim vim-nox

			# Dependencies with different names between APT and Homebrew
			sudo apt install -y silversearcher-ag fd-find \
				libncurses5-dev libncursesw5-dev libncurses-dev ncurses-term # ncurses

			# GNU/Linux only dependencies
			sudo apt install -y python3-neovim python3-pip python3-dev

			# Install vscode
			sudo apt install -y code

		elif [[ "$ID_LIKE" == *"rhel"* || "$ID_LIKE" == *"centos"* || "$ID_LIKE" == *"fedora"* || "$ID" == *"fedora"* ]]; then
			# Dependencies and keys for vscode
			sudo rpm --import https://packages.microsoft.com/keys/microsoft.asc
			sudo sh -c 'echo -e "[code]\nname=Visual Studio Code\nbaseurl=https://packages.microsoft.com/yumrepos/vscode\nenabled=1\ngpgcheck=1\ngpgkey=https://packages.microsoft.com/keys/microsoft.asc" > /etc/yum.repos.d/vscode.repo'

			sudo dnf update -y
			sudo dnf check-update

			sudo dnf install -y curl git xclip xsel ripgrep

			sudo dnf install -y the_silver_searcher \
				ncurses ncurses-term ncurses-devel

			sudo dnf install -y python3-pip

			if [[ "$ID" == *"fedora"* ]]; then
				sudo dnf install -y python3-neovim
			fi

			# Install vscode
			sudo dnf install -y code

		else
			echo "The operating system is not compatible with this installation." && exit 1
		fi

	elif [[ "$(uname -s)" == "Darwin" ]]; then
		brew install curl git xclip xsel ripgrep vim lua

		# Dependencies with different names between APT and Homebrew
		brew install the_silver_searcher fd \
			ncurses # ncurses

		# MacOS only dependencies
		brew install readline

		# Install vscode
		brew install --cask visual-studio-code || :
		xattr -d com.apple.quarantine "/Applications/Visual Studio Code.app" || :

	else
		echo "The operating system is not compatible with this installation." && exit 1
	fi

	if [[ "$(uname -s)" == "Linux" && "$(command -v fdfind)" != "" ]]; then
		sudo rm /usr/bin/fd || :
		sudo ln -s "$(command -v fdfind)" /usr/bin/fd
	fi

	# install yarn
	node_package_module install -g yarn

	# install eslint and typescript
	node_package_module install -g eslint typescript

	# install global node tools
	node_package_module install --global tree-sitter-cli prettier eslint typescript emmet-ls bash-language-server \
		markdownlint-cli @bufbuild/buf nginxbeautifier sql-formatter stylelint

	# check exists goenv
	export GOENV_ROOT="$HOME/.goenv"

	if [[ -d "$GOENV_ROOT" ]]; then
		export PATH="$GOENV_ROOT/bin:$PATH"
		eval "$(goenv init -)"
		export PATH="$GOROOT/bin:$PATH"
		export PATH="$PATH:$GOPATH/bin"
	fi

	if [[ "$(command -v go)" != "" ]]; then
		go install github.com/jesseduffield/lazygit@latest

		go install golang.org/x/tools/gopls@latest
		go install github.com/mgechev/revive@latest
		go install mvdan.cc/gofumpt@latest
		go install github.com/go-delve/delve/cmd/dlv@latest
		go install github.com/rhysd/actionlint/cmd/actionlint@latest
		go install github.com/bufbuild/buf-language-server/cmd/bufls@latest
		go install github.com/mrtazz/checkmake/cmd/checkmake@latest
		go install mvdan.cc/sh/v3/cmd/shfmt@latest
		go install github.com/sonatype-nexus-community/nancy@latest
		go install golang.org/x/vuln/cmd/govulncheck@latest

		curl -sfL https://raw.githubusercontent.com/securego/gosec/master/install.sh |
			sh -s -- -b "$(go env GOPATH)"/bin

		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh |
			sh -s -- -b "$(go env GOPATH)"/bin

		golangci-lint cache clean

	else
		echo -e "${fgcolor_yellow_bold}[Editor Installer]: The 'go' command was not found, the following tools will not be installed: ${fgcolor_white_bold}
        \tlazygit, gopls, revive, gofumpt, dlv, actionlint, bufls, checkmake, shfmt, nancy, govulncheck, gosec, golangci-lint${fgcolor_reset}"
		echo
	fi

	python_binary_installer black
	python_binary_installer pylint
	python_binary_installer beautysh
	python_binary_installer cmakelang

	if [[ "$(uname -s)" == "Linux" ]]; then
		cargo install stylua shellharden

	elif [[ "$(uname -s)" == "Darwin" ]]; then
		brew install shellharden stylua

	else
		echo "The operating system is not compatible with this installation." && exit 1
	fi

	echo

	# ---

	echo -e "${fgcolor_white_bold}[Editor Installer]: - Installing NeoVim...${fgcolor_reset}"

	cd ./downloads/
	bash <(curl -s https://raw.githubusercontent.com/lunarvim/lunarvim/master/utils/installer/install-neovim-from-release) || :
	cd .. # exit downloads/

	echo

	# ---

	echo -e "${fgcolor_white_bold}[Editor Installer]: - Installing Providers for NeoVim...${fgcolor_reset}"
	node_package_module install -g neovim
	python3 -m pip install --user --upgrade pynvim
	python3 -m pip install --user --upgrade msgpack-python
	python3 -m pip install --user --upgrade msgpack

	echo

	# ---

	echo -e "${fgcolor_white_bold}[Editor Installer]: - Uninstalling LunarVim (lvim)...${fgcolor_reset}"
	bash <(curl -s https://raw.githubusercontent.com/lunarvim/lunarvim/master/utils/installer/uninstall.sh) &>/dev/null || :
	rm -rf ~/.config/lvim || :

	echo

	# ---

	echo -e "${fgcolor_white_bold}[Editor Installer]: - Installing LunarVim (lvim)...${fgcolor_reset}"
	## Nightly
	# bash <(curl -s https://raw.githubusercontent.com/lunarvim/lunarvim/master/utils/installer/install.sh) \
	# 	--yes --install-dependencies

	## New release
	LV_BRANCH='release-1.3/neovim-0.9' bash <(curl -s https://raw.githubusercontent.com/LunarVim/LunarVim/release-1.3/neovim-0.9/utils/installer/install.sh) \
		--yes --install-dependencies

	if [[ "$(uname -s)" == "Linux" ]]; then
		if [[ "$(command -v desktop-file-install)" != "" ]]; then
			sudo desktop-file-install ./files/lvim/lvim.desktop
		fi

		if [[ "$(command -v update-desktop-database)" != "" ]]; then
			sudo update-desktop-database
		fi
	fi

	echo

	# ---

	echo -e "${fgcolor_white_bold}[Editor Installer]: - Pasting configuration...${fgcolor_reset}"
	bash ./utils/sync_config.sh editor devtools

	echo

	# ---

	echo -e "${fgcolor_white_bold}[Editor Installer]: - Installing LunarVim (lvim) plugins...${fgcolor_reset}"
	lvim --headless +LvimSyncCorePlugins +qa 2>/dev/null
	lvim --headless +LvimCacheReset +qa 2>/dev/null

	echo

	# ---

	echo -e "${fgcolor_white_bold}[Editor Installer]: - Configuring tools GIT with vscode...${fgcolor_reset}"
	git config --global color.ui auto

	# git config --global core.editor lvim
	git config --global core.editor 'code --wait'

	# git config --global diff.tool lvim
	# git config --global difftool.lvim.cmd 'lvim -d $LOCAL $REMOTE'
	git config --global diff.tool vscode
	# shellcheck disable=SC2016
	git config --global difftool.vscode.cmd 'code --wait --diff $LOCAL $REMOTE'

	# git config --global merge.tool lvim
	# git config --global mergetool.lvim.cmd 'lvim -d $LOCAL $REMOTE $MERGED'
	# git config --global mergetool.lvim.trustExitCode true
	git config --global merge.tool vscode
	# shellcheck disable=SC2016
	git config --global mergetool.vscode.cmd 'code --wait $MERGED'

	echo

	# ---

	echo -e "${fgcolor_white_bold}[Editor Installer]: ${fgcolor_green_bold}✔️ Editor ($editor) successfully installed!${fgcolor_reset}"
	echo -e "${fgcolor_white_bold}[Editor Installer]: (Read ~/.config/lvim/config.lua)${fgcolor_reset}"

	echo -en "$fgcolor_reset"

	if [[ -d ./downloads/ ]]; then
		rm -rf ./downloads/
	fi

	./src/remotes/fixer.sh

	if [[ -f ~/.lunarvim.copy.lua ]]; then
		mv ~/.lunarvim.copy.lua ~/.lunarvim.lua
	fi
}

function exit-error-message() {
	if [[ -f ~/.lunarvim.copy.lua ]]; then
		mv ~/.lunarvim.copy.lua ~/.lunarvim.lua
	fi

	echo -e "$(
		cat <<EOF

${fgcolor_white_bold}[Installer Editor Error]: ---
[Installer Editor Error]: ${fgcolor_red_bold}The installation had an error and was interrupted, the installation was not completed.${fgcolor_white_bold}
[Installer Editor Error]: ---${fgcolor_reset}
EOF
	)"

	exit 1
}

install_editor "$@"
