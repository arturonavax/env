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
	set -o errexit
	trap exit-error-message ERR SIGINT

	rm -rf ./downloads/
	mkdir ./downloads/

	source ./src/remotes/_basics.sh

	./src/remotes/add_lines.sh basics

	echo
	echo -e "${fgcolor_white_bold}[Editor Installer]: Installing Editor ($editor)...${fgcolor_reset}"

	echo
	echo -e "${fgcolor_white_bold}[Editor Installer]: - Installing dependencies...${fgcolor_reset}"

	if [[ "$(uname -s)" == "Linux" ]]; then
		sudo apt install -y curl git xclip xsel ripgrep clang-format

		# Dependencies with different names between APT and Homebrew
		sudo apt install -y silversearcher-ag fd-find \
			libncurses5-dev libncursesw5-dev libncurses5 ncurses-term # ncurses

		# GNU/Linux only dependencies
		sudo apt install -y python3-neovim python3-pip python3-dev

	elif [[ "$(uname -s)" == "Darwin" ]]; then
		brew install curl git xclip xsel ripgrep

		# Dependencies with different names between APT and Homebrew
		brew install the_silver_searcher fd \
			ncurses # ncurses

		# MacOS only dependencies
		brew install readline
	fi

	go install github.com/jesseduffield/lazygit@latest

	if [[ "$(uname -s)" == "Linux" && "$(command -v fdfind)" != "" ]]; then
		sudo rm /usr/bin/fd || :
		sudo ln -s "$(command -v fdfind)" /usr/bin/fd
	fi

	echo

	echo -e "${fgcolor_white_bold}[Editor Installer]: Installing Vim and NeoVim...${fgcolor_reset}"
	if [[ "$(uname -s)" == "Linux" ]]; then
		sudo apt install -y vim vim-nox

		cd ./downloads/
		bash <(curl -s https://raw.githubusercontent.com/lunarvim/lunarvim/master/utils/installer/install-neovim-from-release) || :
		cd .. # exit downloads/

	elif [[ "$(uname -s)" == "Darwin" ]]; then
		brew install vim lua neovim
	fi
	echo

	echo -e "${fgcolor_white_bold}[Editor Installer]: - Installing corepack, yarn...${fgcolor_reset}"
	pnpm install -g corepack
	pnpm install -g yarn
	pnpm install -g eslint typescript
	corepack prepare yarn@stable --activate || :
	echo

	echo -e "${fgcolor_white_bold}[Editor Installer]: Installing Providers for NeoVim...${fgcolor_reset}"
	pnpm install -g neovim
	python3 -m pip install --user --upgrade pynvim
	python3 -m pip install --user --upgrade pynvim
	python3 -m pip install --user --upgrade msgpack-python
	python3 -m pip install --user --upgrade msgpack
	echo

	echo -e "${fgcolor_white_bold}[Editor Installer]: Uninstalling LunarVim (lvim)...${fgcolor_reset}"
	bash <(curl -s https://raw.githubusercontent.com/lunarvim/lunarvim/master/utils/installer/uninstall.sh) &>/dev/null || :
	rm -rf ~/.config/lvim || :
	echo

	echo -e "${fgcolor_white_bold}[Editor Installer]: Installing LunarVim (lvim)...${fgcolor_reset}"
	## Nightly
	# bash <(curl -s https://raw.githubusercontent.com/lunarvim/lunarvim/master/utils/installer/install.sh) \
	# 	--yes --install-dependencies

	## New release
	LV_BRANCH='release-1.3/neovim-0.9' bash <(curl -s https://raw.githubusercontent.com/LunarVim/LunarVim/release-1.3/neovim-0.9/utils/installer/install.sh) \
		--yes --install-dependencies

	echo

	if [[ "$(uname -s)" == "Linux" ]]; then
		if [[ "$(command -v desktop-file-install)" != "" ]]; then
			sudo desktop-file-install ./files/lvim/lvim.desktop
		fi

		if [[ "$(command -v update-desktop-database)" != "" ]]; then
			sudo update-desktop-database
		fi
	fi

	echo -e "${fgcolor_white_bold}[Editor Installer]: Configuring tools GIT with lvim and vim...${fgcolor_reset}"
	git config --global core.editor lvim

	git config --global diff.tool lvim
	# shellcheck disable=SC2016
	git config --global difftool.lvim.cmd 'lvim -d $LOCAL $REMOTE'

	git config --global merge.tool lvim
	# shellcheck disable=SC2016
	git config --global mergetool.lvim.cmd 'lvim -d $LOCAL $REMOTE $MERGED'
	git config --global mergetool.lvim.trustExitCode true

	echo

	echo -e "${fgcolor_white_bold}[Editor Installer]: - Pasting configuration...${fgcolor_reset}"
	bash ./utils/sync_config.sh editor devtools
	echo

	echo -e "${fgcolor_white_bold}[Editor Installer]: - Installing dependency commands...${fgcolor_reset}"

	pipx install black
	pipx install pylint
	pipx install beautysh
	pipx install cmakelang
	echo

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
	echo

	curl -sfL https://raw.githubusercontent.com/securego/gosec/master/install.sh |
		sh -s -- -b "$(go env GOPATH)"/bin
	echo

	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh |
		sh -s -- -b "$(go env GOPATH)"/bin

	golangci-lint cache clean
	echo

	if [[ "$(uname -s)" == "Linux" ]]; then
		cargo install stylua shellharden

	elif [[ "$(uname -s)" == "Darwin" ]]; then
		brew install shellharden stylua
	fi
	echo

	# install global node tools
	pnpm install --global tree-sitter-cli prettier rome eslint typescript emmet-ls bash-language-server \
		markdownlint-cli @bufbuild/buf nginxbeautifier sql-formatter stylelint

	# install node tools plugins in $HOME
	cd ~
	pnpm install eslint typescript postcss stylelint eslint-plugin-react eslint-config-standard-with-typescript \
		@typescript-eslint/parser @typescript-eslint/eslint-plugin eslint-plugin-import eslint-plugin-n eslint-plugin-promise \
		stylelint-config-standard stylelint-config-standard-scss
	cd -
	echo

	echo -e "${fgcolor_white_bold}[Editor Installer]: - Installing LunarVim (lvim) plugins...${fgcolor_reset}"
	lvim --headless +LvimSyncCorePlugins +qa 2>/dev/null
	lvim --headless +LvimCacheReset +qa 2>/dev/null
	echo

	echo -e "${fgcolor_white_bold}[Editor Installer]: ${fgcolor_green_bold}+ Editor ($editor) successfully installed! +${fgcolor_reset}"
	echo -e "${fgcolor_white_bold}[Editor Installer]: (Read ~/.config/lvim/config.lua)${fgcolor_reset}"
	echo

	echo -en "$fgcolor_reset"

	[[ -d ./downloads/ ]] && rm -rf ./downloads/

	./src/remotes/fixer.sh
}

function exit-error-message() {
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
