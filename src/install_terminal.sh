#!/bin/bash
# Run: ./src/install_terminal.sh
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

./src/requirements/terminal.sh || exit 1

source ./src/remotes/_vars_colors.sh
source ./src/remotes/_versions.sh

# install_terminal install terminal and tools
function install_terminal() {
	command -v corepack &>/dev/null && corepack disable 2>/dev/null || :
	set -o errexit
	trap exit-error-message ERR SIGINT

	rm -rf ./downloads/
	mkdir ./downloads/

	source ./src/remotes/_basics.sh



	echo -e "${fgcolor_white_bold}[Terminal Installer]: Starting install_terminal.sh script...${fgcolor_reset}"

	# npm or pnpm
	if [[ "$(command -v npm)" != "" ]]; then
		function node_package_module() {
			npm "$@"
		}
	fi

	if [[ "$(command -v pnpm)" != "" ]]; then
		echo -e "${fgcolor_white_bold}[Terminal Installer]: - - pnpm is used instead of npm${fgcolor_reset}"

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
		echo -e "${fgcolor_white_bold}[Terminal Installer]: - - pipx is used instead of 'python3 -m pip'${fgcolor_reset}"

		function python_binary_installer() {
			pipx install "$@"
		}
	fi

	echo -e "${fgcolor_white_bold}[Terminal Installer]: - Installing tools and dependencies...${fgcolor_reset}"

	if [[ "$(uname -s)" == "Linux" ]]; then
		source /etc/os-release

		if [[ "$ID_LIKE" == *"rhel"* || "$ID_LIKE" == *"centos"* ]]; then
			sudo dnf update -y
			sudo dnf install -y epel-release
		fi

		if [[ "$ID_LIKE" == *"debian"* || "$ID_LIKE" == *"ubuntu"* ]]; then
			sudo apt update -y

			# en_US.UTF-8
			sudo apt install -y language-pack-en
			sudo locale-gen en_US.UTF-8

			# GNU/Linux only dependencies
			## CLI/TUI dependencies
			sudo apt install -y libncurses5-dev libncursesw5-dev libncurses-dev ncurses-term

			## Async dependencies
			sudo apt install -y libevent-dev

			## Fonts dependencies
			sudo apt install -y pkg-config fontconfig

			# GNU/Linux base tools
			sudo apt install -y ca-certificates gnupg bash zsh vim nano less grep screen ed watch zip unzip gzip gcc make autoconf \
				automake cmake python3 git mercurial curl wget m4 byacc swig bison flex ffmpeg pkg-config llvm \
				jq htop wdiff tcpdump iftop rsync openssl openvpn gdb nasm binutils coreutils diffutils findutils util-linux

			# GNU tools for GNU/Linux only
			sudo apt install -y net-tools command-not-found strace

			# GNU tools with different names between APT and Homebrew
			sudo apt install -y netcat-traditional ssh

			# Tools
			sudo apt install -y clang-format rar mtr exiftool git-flow tree eza bat ripgrep xclip xsel tor \
				shellcheck nmap arp-scan aircrack-ng sqlmap direnv

			sudo apt install -y wireshark tshark

			# Tools with different names between APT and Homebrew
			sudo apt install -y tidy protobuf-compiler john fd-find
			sudo snap install ngrok

			./src/remotes/add_lines.sh ngrok
			ngrok config upgrade

			sudo apt install -y clang

		elif [[ "$ID_LIKE" == *"rhel"* || "$ID_LIKE" == *"centos"* || "$ID_LIKE" == *"fedora"* || "$ID" == *"fedora"* ]]; then
			sudo dnf update -y

			sudo dnf install -y ncurses ncurses-term ncurses-devel

			sudo dnf install -y libevent

			sudo dnf install -y fontconfig
			sudo dnf group install -y "Development Tools"

			sudo dnf install -y ca-certificates bash zsh vim-enhanced nano less grep screen ed zip unzip gzip gcc make \
				autoconf automake cmake python3 git mercurial curl wget m4 byacc bison flex llvm jq htop wdiff \
				tcpdump iftop rsync openssl openvpn gdb binutils coreutils diffutils findutils util-linux

			sudo dnf install -y net-tools strace

			sudo dnf install -y netcat openssh

			sudo dnf install -y mtr tree eza bat ripgrep xclip xsel tor \
				shellcheck nmap arp-scan

			sudo dnf install -y wireshark # includes tshark

			sudo dnf install -y tidy protobuf

			sudo dnf install -y clang

			if [[ "$ID" == *"fedora"* ]]; then
				sudo dnf install -y nasm swig john protobuf-compiler aircrack-ng util-linux-user
			fi

		else
			echo "The operating system is not compatible with this installation." && exit 1
		fi

	elif [[ "$(uname -s)" == "Darwin" ]]; then
		brew tap --force homebrew/cask
		brew update

		# MacOS only dependencies
		## CLI/TUI depedencies
		brew install ncurses

		## Async dependencies
		brew install libevent

		## Fonts dependencies
		brew install freetype fontconfig

		# GNU/Linux base tools
		brew install ca-certificates gnupg bash zsh vim nano less grep screen ed watch zip unzip gzip gcc make autoconf \
			automake cmake python@3.12 git mercurial curl wget m4 byacc swig bison flex ffmpeg pkg-config llvm \
			jq htop wdiff tcpdump iftop rsync openssl@3 openvpn nasm binutils coreutils diffutils findutils util-linux
		# jq htop wdiff tcpdump iftop rsync openssl@3 openvpn gdb nasm binutils coreutils diffutils findutils util-linux

		# GNU tools for MacOS only
		brew install gnu-indent gnu-sed gnu-tar gnu-which gnutls gnu-getopt gawk gpatch lsusb

		# GNU tools with different names between APT and Homebrew
		brew install netcat openssh

		# Tools
		brew install clang-format rar mtr exiftool git-flow tmux tree eza bat ripgrep xclip xsel tor \
			shellcheck nmap arp-scan aircrack-ng sqlmap direnv

		sudo ln -s /opt/homebrew/bin/zsh /usr/local/bin/zsh || :
		sudo ln -s /opt/homebrew/bin/tmux /usr/local/bin/tmux || :

		brew install --cask wireshark # includes tshark

		xattr -d com.apple.quarantine /Applications/Wireshark.app || :

		# Tools with different names between APT and Homebrew
		brew install tidy-html5 protobuf john-jumbo fd ngrok/ngrok/ngrok

		./src/remotes/add_lines.sh ngrok
		ngrok config upgrade

		# clang is installed with xcode-select --install

	else
		echo "The operating system is not compatible with this installation." && exit 1
	fi

	# install zoxide
	curl -sS https://raw.githubusercontent.com/ajeetdsouza/zoxide/main/install.sh | bash

	if [[ "$(command -v node_package_module)" != "" ]]; then
		# install yarn
		node_package_module install -g yarn

		# install serve and tldr
		node_package_module install -g serve tldr

	else
		echo -e "${fgcolor_yellow_bold}[Terminal Installer]: The 'node package module' command was not found, the following tools will not be installed: ${fgcolor_white_bold}
        \tyarn, serve, tldr${fgcolor_reset}"
	fi

	# check exists goenv
	export GOENV_ROOT="$HOME/.goenv"

	if [[ -d "$GOENV_ROOT" ]]; then
		export PATH="$GOENV_ROOT/bin:$PATH"
		eval "$(goenv init -)"
		export PATH="$GOROOT/bin:$PATH"
		export PATH="$PATH:$GOPATH/bin"
	fi

	if [[ "$(command -v go)" != "" ]]; then
		# install fx (JSON Viewer) - compiled locally
		go install -ldflags="-s -w" github.com/antonmedv/fx@${FX_VERSION}

		# install protobuf golang plugins - compiled locally
		go install -ldflags="-s -w" google.golang.org/protobuf/cmd/protoc-gen-go@latest
		go install -ldflags="-s -w" google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

		# install sec tools - compiled locally
		go install -ldflags="-s -w" github.com/sonatype-nexus-community/nancy@latest
		go install -ldflags="-s -w" golang.org/x/vuln/cmd/govulncheck@latest
		curl -sfL https://raw.githubusercontent.com/securego/gosec/master/install.sh |
			sh -s -- -b "$(go env GOPATH)"/bin "${GOSEC_VERSION}"

		# install air
		curl -sSfL https://raw.githubusercontent.com/cosmtrek/air/master/install.sh |
			sh -s -- -b "$(go env GOPATH)"/bin "${AIR_VERSION}"

		# install golangci-lint with pinned version
		curl -sSfL https://golangci-lint.run/install.sh |
			sh -s -- -b "$(go env GOPATH)"/bin "${GOLANGCI_LINT_VERSION}"

		# install delve - compiled locally
		go install -ldflags="-s -w" github.com/go-delve/delve/cmd/dlv@${DLV_VERSION}

		golangci-lint cache clean || :

	else
		echo -e "${fgcolor_yellow_bold}[Terminal Installer]: The 'go' command was not found, the following tools will not be installed: ${fgcolor_white_bold}
        \tfx, protoc-gen-go, protoc-gen-go-grpc, nancy, govulncheck, gosec, air, golangci-lint, delve${fgcolor_reset}"
		echo
	fi

	if [[ "$(command -v python_binary_installer)" != "" ]]; then
		python_binary_installer speedtest-cli
		# python_binary_installer pgcli
		python_binary_installer litecli

	else
		echo -e "${fgcolor_yellow_bold}[Terminal Installer]: The 'python binary installer' command was not found, the following tools will not be installed: ${fgcolor_white_bold}
        \tspeedtest-cli, litecli${fgcolor_reset}"
		echo
	fi

	# install just
	if [[ "$(uname -s)" == "Linux" ]]; then
		curl --proto '=https' --tlsv1.2 -sSf https://just.systems/install.sh | bash -s -- --to ~/.local/bin || :

	elif [[ "$(uname -s)" == "Darwin" ]]; then
		brew install just
	fi

	echo

	# ---
	# Install Ghostty
	echo -e "${fgcolor_white_bold}[Terminal Installer]: - Installing Ghostty...${fgcolor_reset}"

	if [[ "$(uname -s)" == "Linux" ]]; then
		if [[ "$(command -v snap)" != "" ]]; then
			sudo snap install ghostty --classic || sudo snap refresh ghostty --classic
		elif [[ "$ID" == *"fedora"* ]]; then
			sudo dnf copr enable -y pgdev/ghostty || :
			sudo dnf install -y ghostty || :
		elif [[ "$(command -v pacman)" != "" ]]; then
			sudo pacman -S --noconfirm ghostty || :
		else
			echo -e "${fgcolor_yellow_bold}[Terminal Installer]: Please install Ghostty manually or via snap: sudo snap install ghostty --classic${fgcolor_reset}"
		fi

		# Ensure handy symlinks for Linux
		mkdir -p "$HOME/.local/bin"
		if [[ "$(command -v batcat)" != "" ]]; then
			sudo ln -sf "$(command -v batcat)" /usr/local/bin/bat 2>/dev/null || ln -sf "$(command -v batcat)" "$HOME/.local/bin/bat"
		fi
		if [[ "$(command -v fdfind)" != "" ]]; then
			sudo ln -sf "$(command -v fdfind)" /usr/local/bin/fd 2>/dev/null || ln -sf "$(command -v fdfind)" "$HOME/.local/bin/fd"
		fi

	elif [[ "$(uname -s)" == "Darwin" ]]; then
		brew tap --force homebrew/cask
		brew install --cask ghostty || brew upgrade --cask ghostty || :
		xattr -d com.apple.quarantine /Applications/Ghostty.app 2>/dev/null || :

	else
		echo "The operating system is not compatible with this installation." && exit 1
	fi

	echo

	# ---
	# Install Tmux (Compiled from source for native CPU performance)
	echo -e "${fgcolor_white_bold}[Terminal Installer]: - Installing/Compiling Tmux (${TMUX_VERSION}) from source...${fgcolor_reset}"

	if [[ "$(uname -s)" == "Linux" ]]; then
		if [[ "$(tmux -V 2>/dev/null)" == "tmux ${TMUX_VERSION}" && -x /usr/local/bin/tmux ]]; then
			echo -e "${fgcolor_green_bold}[Terminal Installer]: Tmux ${TMUX_VERSION} is already compiled and installed at /usr/local/bin/tmux.${fgcolor_reset}"
		else
			cd ./downloads/
			if curl -sSfL "https://github.com/tmux/tmux/releases/download/${TMUX_VERSION}/tmux-${TMUX_VERSION}.tar.gz" -o "tmux-${TMUX_VERSION}.tar.gz"; then
				rm -rf "tmux-${TMUX_VERSION}"
				tar -xzf "tmux-${TMUX_VERSION}.tar.gz"
				cd "tmux-${TMUX_VERSION}"
				CFLAGS="-O3 -march=native" ./configure --prefix=/usr/local
				make -j"$(nproc)"
				sudo make install
				cd ..
			else
				echo -e "${fgcolor_yellow_bold}[Terminal Installer]: Fallback installing tmux from package manager...${fgcolor_reset}"
				if [[ "$(command -v snap)" != "" ]]; then
					sudo snap install tmux --classic || sudo snap refresh tmux --classic || :
				elif [[ "$ID_LIKE" == *"debian"* || "$ID_LIKE" == *"ubuntu"* ]]; then
					sudo apt install -y tmux
				elif [[ "$ID_LIKE" == *"rhel"* || "$ID_LIKE" == *"centos"* || "$ID_LIKE" == *"fedora"* || "$ID" == *"fedora"* ]]; then
					sudo dnf install -y tmux
				fi
			fi
			cd .. # exit downloads/
		fi

	elif [[ "$(uname -s)" == "Darwin" ]]; then
		brew install tmux || brew upgrade tmux || :
	fi

	echo

	# ---

	echo -e "${fgcolor_white_bold}[Terminal Installer]: - Installing tmux-256color info...${fgcolor_reset}"

	cd ./downloads/
	curl -LO https://invisible-island.net/datafiles/current/terminfo.src.gz && gunzip -f terminfo.src.gz
	sudo tic -xe tmux-256color terminfo.src || :
	cd .. # exit downloads/

	echo

	# ---

	if [[ "$SHELL" != *"zsh" ]]; then
		echo -e "${fgcolor_white_bold}[Terminal Installer]: - ${fgcolor_yellow_bold}Configuring Zsh... enter your sudo or root password${fgcolor_reset}"

		if [[ "$(command -v zsh)" != "" ]]; then
			chsh -s "$(command -v zsh)"
		fi

		touch ~/.zshrc

		[[ "$(wc -l ~/.zshrc | awk '{print $1}')" == 0 ]] && cp ./files/zsh/.zshrc ~/.

		echo
	fi

	echo -e "${fgcolor_white_bold}[Terminal Installer]: - Installing Starship...${fgcolor_reset}"
	curl -sS https://starship.rs/install.sh | sh -s -- --yes

	echo

	# ---

	echo -e "${fgcolor_white_bold}[Terminal Installer]: - Installing Zsh Plugins...${fgcolor_reset}"
	mkdir -p ~/.zsh
	for plugin_spec in \
		"https://github.com/zsh-users/zsh-autosuggestions ~/.zsh/zsh-autosuggestions" \
		"https://github.com/zdharma-continuum/fast-syntax-highlighting ~/.zsh/fast-syntax-highlighting" \
		"https://github.com/zsh-users/zsh-completions.git ~/.zsh/zsh-completions" \
		"https://github.com/Aloxaf/fzf-tab ~/.zsh/fzf-tab"; do
		plugin_url="${plugin_spec% *}"
		plugin_dest="${plugin_spec#* }"
		if [[ -d "$plugin_dest/.git" ]]; then
			git -C "$plugin_dest" pull --quiet || :
		else
			rm -rf "$plugin_dest"
			git clone --quiet "$plugin_url" "$plugin_dest" || :
		fi
	done
	rm -f ~/.zcompdump
	if [[ "$(command -v compinit)" != "" ]]; then
		compinit || :
	fi

	echo

	# ---

	echo -e "${fgcolor_white_bold}[Terminal Installer]: - Downloading Tmux Plugin Manager...${fgcolor_reset}"
	if [[ ! -d ~/.tmux/plugins/tpm/.git ]]; then
		mkdir -p ~/.tmux/plugins
		git clone https://github.com/tmux-plugins/tpm ~/.tmux/plugins/tpm
	else
		git -C ~/.tmux/plugins/tpm pull --quiet || :
	fi

	echo

	# ---

	echo -e "${fgcolor_white_bold}[Terminal Installer]: - Synchronizing configuration...${fgcolor_reset}"
	bash ./utils/sync_config.sh terminal

	echo

	# ---

	echo -e "${fgcolor_white_bold}[Terminal Installer]: - Installing Tmux Plugins...${fgcolor_reset}"
	bash ~/.tmux/plugins/tpm/scripts/install_plugins.sh

	echo

	# ---

	echo -e "${fgcolor_white_bold}[Terminal Installer]: - Add ZSH Shell sources...${fgcolor_reset}"

	[[ ! -f ~/.zshrc ]] && touch ~/.zshrc

	# add 'source ~/.tools.sh' command to the begin of the file
	# add 'source ~/.base.zsh' command to the begin of the file
	./src/remotes/add_lines.sh tools base

	[[ "$(wc -l ~/.zshrc | awk '{print $1}')" == "$(wc -l ./files/zsh/.zshrc | awk '{print $1}')" ]] && cp ./files/zsh/.zshrc ~/.zshrc

	# ---

	echo -e "${fgcolor_white_bold}[Terminal Installer]: - Installing fzf...${fgcolor_reset}"
	if [[ ! -d ~/.fzf/.git ]]; then
		rm -rf ~/.fzf
		git clone --depth 1 https://github.com/junegunn/fzf.git ~/.fzf
	else
		git -C ~/.fzf pull --quiet || :
	fi
	~/.fzf/install --all --no-update-rc

	echo

	# ---

	echo -e "${fgcolor_white_bold}[Terminal Installer]: ${fgcolor_green_bold}✔️ Terminal successfully installed!${fgcolor_reset}"
	echo -e "${fgcolor_white_bold}[Terminal Installer]: (Restarting the computer to use Zsh for the first time)${fgcolor_reset}"

	echo -en "$fgcolor_reset"

	if [[ -d ./downloads/ ]]; then
		rm -rf ./downloads/
	fi

	./src/remotes/fixer.sh
}

function exit-error-message() {
	echo -e "$(
		cat <<EOF

${fgcolor_white_bold}[Installer Terminal Error]: ---
[Installer Terminal Error]: ${fgcolor_red_bold}The installation had an error and was interrupted, the installation was not completed.${fgcolor_white_bold}
[Installer Terminal Error]: ---${fgcolor_reset}
EOF
	)"

	exit 1
}

install_terminal "$@"
