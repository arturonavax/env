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

source ./src/lib/strict.sh
source ./src/lib/colors.sh
source ./src/lib/logging.sh
source ./src/lib/platform.sh
source ./src/lib/packages.sh
source ./src/remotes/_vars_colors.sh
source ./src/remotes/_versions.sh

editor="nvim"

# install_editor install editor
function install_editor() {
	command -v corepack &>/dev/null && corepack disable 2>/dev/null || :

	set -o errexit
	trap exit-error-message ERR SIGINT

	sudo rm -rf ./downloads/ 2>/dev/null || rm -rf ./downloads/ 2>/dev/null || :
	mkdir -p ./downloads/

	source ./src/remotes/_basics.sh

	./src/remotes/add_lines.sh basics

	echo -e "${fgcolor_white_bold}[Editor Installer]: Starting install_editor.sh script...${fgcolor_reset}"

	# Load toolchain paths if available
	if [[ -f ./files/zsh/.tools.sh ]]; then
		# shellcheck source=/dev/null
		source ./files/zsh/.tools.sh 2>/dev/null || true
	elif [[ -f "$HOME/.tools.sh" ]]; then
		# shellcheck source=/dev/null
		source "$HOME/.tools.sh" 2>/dev/null || true
	fi

	# Node Package Installer Helpers (pnpm preferred, npm fallback)
	node_pkg_add() {
		if command -v pnpm >/dev/null 2>&1; then
			pnpm add -g "$@"
		elif command -v npm >/dev/null 2>&1; then
			npm install -g "$@"
		else
			echo -e "${fgcolor_yellow_bold}[Editor Installer]: Neither pnpm nor npm found; skipping global packages: $*${fgcolor_reset}"
		fi
	}

	node_pkg_remove() {
		if command -v pnpm >/dev/null 2>&1; then
			pnpm remove -g "$@" 2>/dev/null || true
		elif command -v npm >/dev/null 2>&1; then
			npm uninstall -g "$@" 2>/dev/null || true
		fi
	}

	# Python Binary / CLI Application Installer (pipx preferred, pip user fallback)
	python_cli_add() {
		if command -v pipx >/dev/null 2>&1; then
			for pkg in "$@"; do
				pipx install "$pkg" 2>/dev/null || pipx upgrade "$pkg" 2>/dev/null || true
			done
		elif command -v python3 >/dev/null 2>&1 && python3 -m pip --version >/dev/null 2>&1; then
			python3 -m pip install --user --upgrade "$@"
		fi
	}

	echo -e "${fgcolor_white_bold}[Editor Installer]: - Installing tools and dependencies...${fgcolor_reset}"

	# Update package indexes and install editor prerequisites across all supported distributions
	pkg_update_cache
	pkg_install curl git xclip xsel ripgrep clang-format vim python3-pip fzf

	if [[ "$(uname -s)" == "Linux" && "$(command -v fdfind)" != "" ]]; then
		sudo rm -f /usr/bin/fd || :
		sudo ln -sf "$(command -v fdfind)" /usr/bin/fd 2>/dev/null || :
	fi

	# clean legacy npm/pnpm packages migrated to cargo or go
	node_pkg_remove tree-sitter-cli @bufbuild/buf
	rm -f "$HOME/.local/share/pnpm/tree-sitter" "$HOME/.local/share/pnpm/buf" 2>/dev/null || :

	# install global node tools
	node_pkg_add yarn eslint typescript prettier emmet-ls bash-language-server \
		markdownlint-cli nginxbeautifier sql-formatter stylelint

	# GVM (Go Version Manager) environment integration
	export GVM_ROOT="${GVM_ROOT:-$HOME/.gvm}"
	if [[ -s "$GVM_ROOT/scripts/gvm" ]]; then
		# shellcheck source=/dev/null
		source "$GVM_ROOT/scripts/gvm"
	fi

	mkdir -p "$HOME/.local/bin"

	# 1. lazygit (official precompiled release binary prioritized)
	if [[ "$(command -v lazygit)" == "" ]]; then
		if [[ "$(uname -s)" == "Darwin" ]]; then
			brew install lazygit 2>/dev/null || :
		else
			echo -e "${fgcolor_white_bold}[Editor Installer]: Downloading pre-built lazygit binary...${fgcolor_reset}"
			arch="$(uname -m)"
			lg_arch="x86_64"
			[[ "$arch" == "aarch64" || "$arch" == "arm64" ]] && lg_arch="arm64"
			if curl -fsSL "https://github.com/jesseduffield/lazygit/releases/download/${LAZYGIT_VERSION}/lazygit_${LAZYGIT_VERSION#v}_linux_${lg_arch}.tar.gz" | tar -xz -C "$HOME/.local/bin/" lazygit 2>/dev/null; then
				chmod +x "$HOME/.local/bin/lazygit"
			else
				if [[ "$(command -v go)" != "" ]]; then
					go install -ldflags="-s -w" github.com/jesseduffield/lazygit@"${LAZYGIT_VERSION}" || :
				fi
			fi
		fi
	fi

	# 2. shfmt (official precompiled release binary prioritized)
	if [[ "$(command -v shfmt)" == "" ]]; then
		if [[ "$(uname -s)" == "Darwin" ]]; then
			brew install shfmt 2>/dev/null || :
		else
			echo -e "${fgcolor_white_bold}[Editor Installer]: Downloading pre-built shfmt binary...${fgcolor_reset}"
			arch="$(uname -m)"
			shfmt_arch="amd64"
			[[ "$arch" == "aarch64" || "$arch" == "arm64" ]] && shfmt_arch="arm64"
			if curl -fsSL "https://github.com/mvdan/sh/releases/download/${SHFMT_VERSION}/shfmt_${SHFMT_VERSION}_linux_${shfmt_arch}" -o "$HOME/.local/bin/shfmt" 2>/dev/null; then
				chmod +x "$HOME/.local/bin/shfmt"
			else
				if [[ "$(command -v go)" != "" ]]; then
					go install -ldflags="-s -w" mvdan.cc/sh/v3/cmd/shfmt@"${SHFMT_VERSION}" || :
				fi
			fi
		fi
	fi

	# 3. actionlint (official precompiled release binary prioritized)
	if [[ "$(command -v actionlint)" == "" ]]; then
		if [[ "$(uname -s)" == "Darwin" ]]; then
			brew install actionlint 2>/dev/null || :
		else
			echo -e "${fgcolor_white_bold}[Editor Installer]: Downloading pre-built actionlint binary...${fgcolor_reset}"
			bash <(curl -s https://raw.githubusercontent.com/rhysd/actionlint/main/scripts/download-actionlint.bash) "$HOME/.local/bin" 2>/dev/null || {
				if [[ "$(command -v go)" != "" ]]; then
					go install -ldflags="-s -w" github.com/rhysd/actionlint/cmd/actionlint@"${ACTIONLINT_VERSION}" || :
				fi
			}
		fi
	fi

	if [[ "$(command -v go)" != "" ]]; then
		# Go language servers and development tools
		go install -ldflags="-s -w" golang.org/x/tools/gopls@"${GOPLS_VERSION}" || :
		go install -ldflags="-s -w" github.com/mgechev/revive@latest || :
		go install -ldflags="-s -w" mvdan.cc/gofumpt@"${GOFUMPT_VERSION}" || :
		go install -ldflags="-s -w" github.com/go-delve/delve/cmd/dlv@"${DLV_VERSION}" || :
		go install -ldflags="-s -w" github.com/bufbuild/buf-language-server/cmd/bufls@latest || :
		go install -ldflags="-s -w" github.com/bufbuild/buf/cmd/buf@"${BUF_VERSION}" || :
		go install -ldflags="-s -w" github.com/checkmake/checkmake/cmd/checkmake@"${CHECKMAKE_VERSION}" || :
		go install -ldflags="-s -w" github.com/sonatype-nexus-community/nancy@latest || :
		go install -ldflags="-s -w" golang.org/x/vuln/cmd/govulncheck@latest || :

		curl -sfL https://raw.githubusercontent.com/securego/gosec/master/install.sh |
			sh -s -- -b "$(go env GOPATH)"/bin "${GOSEC_VERSION}" || :

		curl -sSfL https://golangci-lint.run/install.sh |
			sh -s -- -b "$(go env GOPATH)"/bin "${GOLANGCI_LINT_VERSION}" || :

		golangci-lint cache clean || :
	fi

	# Python language server & formatting tools via pipx
	python_cli_add black pylint beautysh cmakelang

	# 4. stylua (official precompiled release binary prioritized)
	if [[ "$(command -v stylua)" == "" ]]; then
		if [[ "$(uname -s)" == "Darwin" ]]; then
			brew install stylua 2>/dev/null || :
		else
			echo -e "${fgcolor_white_bold}[Editor Installer]: Downloading pre-built stylua binary...${fgcolor_reset}"
			arch="$(uname -m)"
			stylua_zip="stylua-linux-x86_64.zip"
			[[ "$arch" == "aarch64" || "$arch" == "arm64" ]] && stylua_zip="stylua-linux-aarch64.zip"
			if curl -fsSL "https://github.com/JohnnyMorganz/StyLua/releases/download/v${STYLUA_VERSION}/${stylua_zip}" -o /tmp/stylua.zip 2>/dev/null && \
				unzip -o /tmp/stylua.zip -d "$HOME/.local/bin/" 2>/dev/null; then
				chmod +x "$HOME/.local/bin/stylua"
				rm -f /tmp/stylua.zip
			else
				rm -f /tmp/stylua.zip
				if [[ "$(command -v cargo)" != "" ]]; then
					RUSTFLAGS="-C target-cpu=native" cargo install --locked stylua --version "${STYLUA_VERSION}" || :
				fi
			fi
		fi
	fi

	# 5. tree-sitter CLI (official precompiled release binary prioritized)
	if [[ "$(command -v tree-sitter)" == "" ]]; then
		if [[ "$(uname -s)" == "Darwin" ]]; then
			brew install tree-sitter 2>/dev/null || :
		else
			echo -e "${fgcolor_white_bold}[Editor Installer]: Downloading pre-built tree-sitter binary...${fgcolor_reset}"
			arch="$(uname -m)"
			ts_bin="tree-sitter-linux-x64.gz"
			[[ "$arch" == "aarch64" || "$arch" == "arm64" ]] && ts_bin="tree-sitter-linux-arm64.gz"
			if curl -fsSL "https://github.com/tree-sitter/tree-sitter/releases/download/v${TREE_SITTER_VERSION}/${ts_bin}" | gunzip -c > "$HOME/.local/bin/tree-sitter" 2>/dev/null; then
				chmod +x "$HOME/.local/bin/tree-sitter"
			else
				if [[ "$(command -v cargo)" != "" ]]; then
					RUSTFLAGS="-C target-cpu=native" cargo install --locked tree-sitter-cli --version "${TREE_SITTER_VERSION}" || :
				fi
			fi
		fi
	fi

	# 6. shellharden
	if [[ "$(command -v shellharden)" == "" ]]; then
		if [[ "$(uname -s)" == "Darwin" ]]; then
			brew install shellharden 2>/dev/null || :
		elif [[ "$(command -v cargo)" != "" ]]; then
			RUSTFLAGS="-C target-cpu=native" cargo install --locked shellharden --version "${SHELLHARDEN_VERSION}" || :
		fi
	fi

	# Clean stale tree-sitter locks and ensure native binary takes precedence in ~/.local/bin
	rm -rf "$HOME/.cache/tree-sitter/lock" 2>/dev/null || :
	if [[ -f "$HOME/.cargo/bin/tree-sitter" ]]; then
		mkdir -p "$HOME/.local/bin"
		ln -sf "$HOME/.cargo/bin/tree-sitter" "$HOME/.local/bin/tree-sitter"
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
			if [[ -d "/opt/${nvim_dir}" ]]; then
				sudo chmod -R a+rX "/opt/${nvim_dir}" 2>/dev/null || :
			fi

			if [[ "$(command -v nvim)" != "" && "$(nvim --version 2>/dev/null | head -n 1)" == *"${NVIM_VERSION}"* && -x /usr/local/bin/nvim ]]; then
				echo -e "${fgcolor_green_bold}[Editor Installer]: Neovim (${NVIM_VERSION}) is already installed at /usr/local/bin/nvim.${fgcolor_reset}"
			else
				cd ./downloads/
				if curl -fsSL -O "https://github.com/neovim/neovim/releases/download/${NVIM_VERSION}/${nvim_tar}" ||
					curl -fsSL -O "https://github.com/neovim/neovim/releases/latest/download/${nvim_tar}"; then
					if [[ -f "${nvim_tar}" ]]; then
						sudo rm -rf "/opt/${nvim_dir}"
						sudo tar -C /opt -xzf "${nvim_tar}"
						sudo chmod -R a+rX "/opt/${nvim_dir}" 2>/dev/null || :
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
		pipx install pynvim || pipx upgrade pynvim || :
	fi

	echo

	# ---

	echo -e "${fgcolor_white_bold}[Editor Installer]: - Pasting LazyVim configuration...${fgcolor_reset}"
	bash ./utils/sync_config.sh editor devtools

	echo

	# ---

	echo -e "${fgcolor_white_bold}[Editor Installer]: - Restoring/Syncing LazyVim plugins and Treesitter parsers...${fgcolor_reset}"
	if [[ "$(command -v nvim)" != "" ]]; then
		rm -rf "$HOME/.cache/tree-sitter/lock" 2>/dev/null || :
		nvim --headless "+Lazy! restore" +qa </dev/null &>/dev/null || nvim --headless "+Lazy! sync" +qa </dev/null &>/dev/null || :
		# Pre-compile LazyVim treesitter parsers headless so the first interactive launch is clean and instant
		nvim --headless -c "Lazy load nvim-treesitter" -c "TSUpdate" -c "sleep 12" -c "qa" </dev/null &>/dev/null || :
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

	if [[ "$(uname -s)" == "Linux" ]]; then
		echo -e "${fgcolor_white_bold}[Editor Installer]: - Configuring Desktop entries and default MIME associations...${fgcolor_reset}"

		# Remove legacy LunarVim (lvim) desktop entries
		sudo rm -f /usr/share/applications/lvim.desktop /usr/local/share/applications/lvim.desktop 2>/dev/null || :
		rm -f "$HOME/.local/share/applications/lvim.desktop" 2>/dev/null || :

		# Install Neovim icons if present
		if [[ -n "$nvim_dir" && -d "/opt/${nvim_dir}/share/icons/hicolor" ]]; then
			sudo cp -rn "/opt/${nvim_dir}/share/icons/hicolor" /usr/share/icons/ 2>/dev/null || :
		fi

		# Install Neovim desktop file
		if [[ -f "./files/nvim/nvim.desktop" ]]; then
			if [[ "$(command -v desktop-file-install)" != "" ]]; then
				sudo desktop-file-install ./files/nvim/nvim.desktop 2>/dev/null || :
			else
				sudo cp ./files/nvim/nvim.desktop /usr/share/applications/ 2>/dev/null || :
			fi

			mkdir -p "$HOME/.local/share/applications"
			cp ./files/nvim/nvim.desktop "$HOME/.local/share/applications/" 2>/dev/null || :
		fi

		if [[ "$(command -v update-desktop-database)" != "" ]]; then
			sudo bash -c 'umask 022 && update-desktop-database /usr/share/applications && chmod a+r /usr/share/applications/mimeinfo.cache' 2>/dev/null || :
			update-desktop-database "$HOME/.local/share/applications" 2>/dev/null || :
		fi

		# Set nvim as default text editor for common formats
		if [[ "$(command -v xdg-mime)" != "" ]]; then
			for mime in text/plain text/english text/markdown text/x-c text/x-c++ text/x-c++hdr text/x-c++src \
				text/x-chdr text/x-csrc text/x-java text/x-makefile text/x-python text/rust text/x-go \
				application/json application/javascript application/x-yaml application/x-shellscript; do
				xdg-mime default nvim.desktop "$mime" 2>/dev/null || :
			done
		fi

		echo
	fi

	# ---

	echo -e "${fgcolor_white_bold}[Editor Installer]: ${fgcolor_green_bold}✔️ Editor ($editor - LazyVim) successfully installed!${fgcolor_reset}"
	echo -e "${fgcolor_white_bold}[Editor Installer]: (Read ~/.config/nvim/init.lua and ~/.config/nvim/lua/)${fgcolor_reset}"

	echo -en "$fgcolor_reset"

	if [[ -d ./downloads/ ]]; then
		sudo rm -rf ./downloads/ 2>/dev/null || rm -rf ./downloads/ 2>/dev/null || :
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
