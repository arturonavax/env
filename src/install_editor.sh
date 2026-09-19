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
source ./src/remotes/_versions.sh

editor="nvim"

# install_editor install editor
function install_editor() {
	command -v corepack &>/dev/null && corepack disable 2>/dev/null || :

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
			sudo apt update -y
			sudo apt install -y apt-transport-https curl git xclip xsel ripgrep clang-format vim vim-nox fd-find pipx python3-pip python3-dev

		elif [[ "$ID_LIKE" == *"rhel"* || "$ID_LIKE" == *"centos"* || "$ID_LIKE" == *"fedora"* || "$ID" == *"fedora"* ]]; then
			sudo dnf update -y
			sudo dnf install -y curl git xclip xsel ripgrep pipx python3-pip

		else
			echo "The operating system is not compatible with this installation." && exit 1
		fi

	elif [[ "$(uname -s)" == "Darwin" ]]; then
		brew install curl git xclip xsel ripgrep vim lua fd readline

	else
		echo "The operating system is not compatible with this installation." && exit 1
	fi

	if [[ "$(uname -s)" == "Linux" && "$(command -v fdfind)" != "" ]]; then
		sudo rm -f /usr/bin/fd || :
		sudo ln -sf "$(command -v fdfind)" /usr/bin/fd 2>/dev/null || :
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
		# Compiling tools locally from source with stripped symbols for maximum performance
		go install -ldflags="-s -w" github.com/jesseduffield/lazygit@${LAZYGIT_VERSION}
		go install -ldflags="-s -w" golang.org/x/tools/gopls@${GOPLS_VERSION}
		go install -ldflags="-s -w" github.com/mgechev/revive@latest
		go install -ldflags="-s -w" mvdan.cc/gofumpt@${GOFUMPT_VERSION}
		go install -ldflags="-s -w" github.com/go-delve/delve/cmd/dlv@${DLV_VERSION}
		go install -ldflags="-s -w" github.com/rhysd/actionlint/cmd/actionlint@${ACTIONLINT_VERSION}
		go install -ldflags="-s -w" github.com/bufbuild/buf-language-server/cmd/bufls@latest
		go install -ldflags="-s -w" github.com/mrtazz/checkmake/cmd/checkmake@latest
		go install -ldflags="-s -w" mvdan.cc/sh/v3/cmd/shfmt@${SHFMT_VERSION}
		go install -ldflags="-s -w" github.com/sonatype-nexus-community/nancy@latest
		go install -ldflags="-s -w" golang.org/x/vuln/cmd/govulncheck@latest

		curl -sfL https://raw.githubusercontent.com/securego/gosec/master/install.sh |
			sh -s -- -b "$(go env GOPATH)"/bin "${GOSEC_VERSION}"

		curl -sSfL https://golangci-lint.run/install.sh |
			sh -s -- -b "$(go env GOPATH)"/bin "${GOLANGCI_LINT_VERSION}"

		golangci-lint cache clean || :

	else
		echo -e "${fgcolor_yellow_bold}[Editor Installer]: The 'go' command was not found, the following tools will not be installed: ${fgcolor_white_bold}
        \tlazygit, gopls, revive, gofumpt, dlv, actionlint, bufls, checkmake, shfmt, nancy, govulncheck, gosec, golangci-lint${fgcolor_reset}"
		echo
	fi

	if [[ "$(command -v pipx)" != "" ]]; then
		pipx install black || :
		pipx install pylint || :
		pipx install beautysh || :
		pipx install cmakelang || :
	elif [[ "$(command -v python_binary_installer)" != "" ]]; then
		python_binary_installer black || :
		python_binary_installer pylint || :
		python_binary_installer beautysh || :
		python_binary_installer cmakelang || :
	fi

	# Compiling Rust tools locally with native CPU optimizations
	if [[ "$(uname -s)" == "Linux" ]]; then
		RUSTFLAGS="-C target-cpu=native" cargo install --locked stylua --version "${STYLUA_VERSION}" || cargo install stylua || :
		RUSTFLAGS="-C target-cpu=native" cargo install --locked shellharden --version "${SHELLHARDEN_VERSION}" || cargo install shellharden || :

	elif [[ "$(uname -s)" == "Darwin" ]]; then
		brew install shellharden stylua || :

	else
		echo "The operating system is not compatible with this installation." && exit 1
	fi

	echo

	# ---

	echo -e "${fgcolor_white_bold}[Editor Installer]: - Installing Neovim (${NVIM_VERSION})...${fgcolor_reset}"

	if [[ "$(uname -s)" == "Linux" ]]; then
		arch="$(uname -m)"
		nvim_tar=""
		nvim_dir=""
		if [[ "$arch" == "x86_64" ]]; then
			nvim_tar="nvim-linux-x86_64.tar.gz"
			nvim_dir="nvim-linux-x86_64"
		elif [[ "$arch" == "aarch64" || "$arch" == "arm64" ]]; then
			nvim_tar="nvim-linux-arm64.tar.gz"
			nvim_dir="nvim-linux-arm64"
		fi

		if [[ -n "$nvim_tar" ]]; then
			if [[ "$(command -v nvim)" != "" && "$(nvim --version 2>/dev/null | head -n 1)" == *"${NVIM_VERSION}"* && -x /usr/local/bin/nvim ]]; then
				echo -e "${fgcolor_green_bold}[Editor Installer]: Neovim (${NVIM_VERSION}) is already installed at /usr/local/bin/nvim.${fgcolor_reset}"
			else
				cd ./downloads/
				if curl -fsSL -O "https://github.com/neovim/neovim/releases/download/${NVIM_VERSION}/${nvim_tar}" ||
					curl -fsSL -O "https://github.com/neovim/neovim/releases/latest/download/${nvim_tar}"; then
					if [[ -f "${nvim_tar}" ]]; then
						sudo rm -rf "/opt/${nvim_dir}"
						sudo tar -C /opt -xzf "${nvim_tar}"
						sudo ln -sf "/opt/${nvim_dir}/bin/nvim" /usr/local/bin/nvim
						mkdir -p "$HOME/.local/bin"
						ln -sf "/opt/${nvim_dir}/bin/nvim" "$HOME/.local/bin/nvim"
					fi
				fi
				cd .. # exit downloads/
			fi
		fi

	elif [[ "$(uname -s)" == "Darwin" ]]; then
		brew install neovim || brew upgrade neovim || :
	fi

	echo

	# ---

	echo -e "${fgcolor_white_bold}[Editor Installer]: - Installing Providers for NeoVim...${fgcolor_reset}"
	node_package_module install -g neovim || :
	if [[ "$(command -v pipx)" != "" ]]; then
		pipx install pynvim || :
	fi

	echo

	# ---

	echo -e "${fgcolor_white_bold}[Editor Installer]: - Pasting LazyVim configuration...${fgcolor_reset}"
	bash ./utils/sync_config.sh editor devtools

	echo

	# ---

	echo -e "${fgcolor_white_bold}[Editor Installer]: - Restoring/Syncing LazyVim plugins from lockfile...${fgcolor_reset}"
	if [[ "$(command -v nvim)" != "" ]]; then
		nvim --headless "+Lazy! restore" +qa 2>/dev/null || nvim --headless "+Lazy! sync" +qa 2>/dev/null || :
	fi

	echo

	# ---

	echo -e "${fgcolor_white_bold}[Editor Installer]: - Configuring GIT with nvim...${fgcolor_reset}"
	git config --global color.ui auto

	if [[ "$(command -v nvim)" != "" ]]; then
		git config --global core.editor nvim
		git config --global diff.tool nvimdiff
		git config --global merge.tool nvimdiff
	fi

	echo

	# ---

	echo -e "${fgcolor_white_bold}[Editor Installer]: ${fgcolor_green_bold}✔️ Editor ($editor - LazyVim) successfully installed!${fgcolor_reset}"
	echo -e "${fgcolor_white_bold}[Editor Installer]: (Read ~/.config/nvim/init.lua and ~/.config/nvim/lua/)${fgcolor_reset}"

	echo -en "$fgcolor_reset"

	if [[ -d ./downloads/ ]]; then
		rm -rf ./downloads/
	fi

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
