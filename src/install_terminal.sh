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

# install_terminal install terminal and tools
function install_terminal() {
	set -o errexit
	trap exit-error-message ERR SIGINT

	rm -rf ./downloads/
	mkdir ./downloads/

	source ./src/remotes/_basics.sh

	if [[ "$(command -v alacritty)" != "" ]]; then
		alacritty_version_minor="$(alacritty --version | cut -d' ' -f2 | tr -d '\-dev' | cut -d. -f2)"
	fi

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
			## CLI/TUI depedencies
			sudo apt install -y libncurses5-dev libncursesw5-dev libncurses-dev ncurses-term

			## Async dependencies
			sudo apt install -y libevent-dev

			## GUI and Qt dependencies
			sudo apt install -y libxrandr2 libxrandr2 libxss1 \
				libxcursor1 libxcomposite1 libasound2 libxi6 libxtst6 libxcb-xfixes0-dev

			## Fonts dependencies
			sudo apt install -y pkg-config fontconfig libfreetype6-dev libfontconfig1-dev libxcb-xfixes0-dev libxkbcommon-dev

			# GNU/Linux base tools
			sudo apt install -y ca-certificates gnupg bash zsh vim nano less grep screen ed watch zip unzip gzip gcc make autoconf \
				automake cmake python3 git mercurial curl wget m4 byacc swig bison flex ffmpeg pkg-config llvm \
				jq htop wdiff tcpdump iftop rsync openssl openvpn gdb nasm binutils coreutils diffutils findutils util-linux

			# GNU tools for GNU/Linux only
			sudo apt install -y net-tools command-not-found strace

			# GNU tools with different names between APT and Homebrew
			sudo apt install -y netcat-traditional ssh

			# Tools
			sudo apt install -y clang-format rar mtr exiftool git-flow tmux tree exa bat ripgrep xclip xsel tor \
				shellcheck nmap arp-scan aircrack-ng sqlmap

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

			sudo dnf install -y fontconfig freetype-devel fontconfig-devel libxcb-devel libxkbcommon-devel
			sudo dnf group install -y "Development Tools"

			sudo dnf install -y ca-certificates bash zsh vim-enhanced nano less grep screen ed zip unzip gzip gcc make \
				autoconf automake cmake python3 git mercurial curl wget m4 byacc bison flex llvm jq htop wdiff \
				tcpdump iftop rsync openssl openvpn gdb binutils coreutils diffutils findutils util-linux

			sudo dnf install -y net-tools strace

			sudo dnf install -y netcat openssh

			sudo dnf install -y mtr tmux tree exa bat ripgrep xclip xsel tor \
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
			automake cmake python3 git mercurial curl wget m4 byacc swig bison flex ffmpeg pkg-config llvm \
			jq htop wdiff tcpdump iftop rsync openssl openvpn gdb nasm binutils coreutils diffutils findutils util-linux

		# GNU tools for MacOS only
		brew install gnu-indent gnu-sed gnu-tar gnu-which gnutls gnu-getopt gawk gpatch lsusb

		# GNU tools with different names between APT and Homebrew
		brew install netcat openssh

		# Tools
		brew install clang-format rar mtr exiftool git-flow tmux tree exa bat ripgrep xclip xsel tor \
			shellcheck nmap arp-scan aircrack-ng sqlmap

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

	if [[ "$(command -v go)" != "" ]]; then
		# install fx (JSON Viewer)
		go install github.com/antonmedv/fx@latest

		# install protobuf golang plugins
		go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
		go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

		# install sec tools
		go install github.com/sonatype-nexus-community/nancy@latest
		go install golang.org/x/vuln/cmd/govulncheck@latest
		curl -sfL https://raw.githubusercontent.com/securego/gosec/master/install.sh |
			sh -s -- -b "$(go env GOPATH)"/bin

		# install air
		curl -sSfL https://raw.githubusercontent.com/cosmtrek/air/master/install.sh |
			sh -s -- -b "$(go env GOPATH)"/bin

		# install golangci-lint
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh |
			sh -s -- -b "$(go env GOPATH)"/bin

		# install delve
		go install github.com/go-delve/delve/cmd/dlv@latest

		golangci-lint cache clean

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

	if ((alacritty_version_minor <= 11)); then
		echo -e "${fgcolor_white_bold}[Terminal Installer]: - Installing Alacritty...${fgcolor_reset}"

		if [[ "$(uname -s)" == "Linux" ]]; then
			if [[ "$ID_LIKE" == *"debian"* || "$ID_LIKE" == *"ubuntu"* && "$(command -v add-apt-repository)" != "" ]]; then
				# sudo add-apt-repository ppa:aslatter/ppa -y

				sudo apt install alacritty

			elif [[ "$ID_LIKE" == *"rhel"* || "$ID_LIKE" == *"centos"* || "$ID_LIKE" == *"fedora"* || "$ID" == *"fedora"* ]]; then
				if [[ "$ID" == *"fedora"* ]]; then
					sudo dnf install -y alacritty

				else
					cd ./downloads/
					git clone --depth 1 https://github.com/alacritty/alacritty
					cd ./alacritty/

					cargo build --release
					infocmp alacritty &>/dev/null || :
					sudo tic -xe alacritty,alacritty-direct extra/alacritty.info || :

					sudo cp target/release/alacritty /usr/local/bin
					sudo cp extra/logo/alacritty-term.svg /usr/share/pixmaps/Alacritty.svg
					sudo desktop-file-install extra/linux/Alacritty.desktop
					sudo update-desktop-database

					cd ../.. # exit downloads/
				fi

			else
				echo "The operating system is not compatible with this installation." && exit 1
			fi

		elif [[ "$(uname -s)" == "Darwin" ]]; then
			brew tap --force homebrew/cask

			brew reinstall --cask alacritty

			xattr -d com.apple.quarantine /Applications/Alacritty.app || :

			## Compiling Alacritty on MacOS
			# cd ./downloads/
			# git clone --depth 1 https://github.com/alacritty/alacritty
			# cd ./alacritty/
			# make app
			# cp -r target/release/osx/Alacritty.app /Applications/
			# infocmp alacritty &>/dev/null || :
			# sudo tic -xe alacritty,alacritty-direct extra/alacritty.info || :
			# cd ../.. # exit downloads/

		else
			echo "The operating system is not compatible with this installation." && exit 1
		fi

		echo
	fi

	# ---

	echo -e "${fgcolor_white_bold}[Terminal Installer]: - Installing alacritty color and tmux-256color info...${fgcolor_reset}"

	cd ./downloads/

	curl -LO https://invisible-island.net/datafiles/current/terminfo.src.gz && gunzip terminfo.src.gz

	sudo tic -xe tmux-256color terminfo.src

	curl -LO https://raw.githubusercontent.com/alacritty/alacritty/master/extra/alacritty.info

	sudo tic -xe alacritty,alacritty-direct alacritty.info

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
	git clone https://github.com/zsh-users/zsh-autosuggestions ~/.zsh/zsh-autosuggestions || :
	git clone https://github.com/zdharma-continuum/fast-syntax-highlighting ~/.zsh/fast-syntax-highlighting || :
	git clone https://github.com/zsh-users/zsh-completions.git ~/.zsh/zsh-completions || :
	git clone https://github.com/Aloxaf/fzf-tab ~/.zsh/fzf-tab || :
	rm -f ~/.zcompdump
	if [[ "$(command -v compinit)" != "" ]]; then
		compinit || :
	fi

	echo

	# ---

	echo -e "${fgcolor_white_bold}[Terminal Installer]: - Downloading Tmux Plugin Manager...${fgcolor_reset}"
	rm -rf ~/.tmux/plugins
	git clone https://github.com/tmux-plugins/tpm ~/.tmux/plugins/tpm

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

	# add 'source ~/.base.zsh' command to the end of the file
	# add 'source ~/.tools.sh' command to the end of the file
	./src/remotes/add_lines.sh base tools

	[[ "$(wc -l ~/.zshrc | awk '{print $1}')" == "$(wc -l ./files/zsh/.zshrc | awk '{print $1}')" ]] && cp ./files/zsh/.zshrc ~/.zshrc

	# ---

	echo -e "${fgcolor_white_bold}[Terminal Installer]: - Installing fzf...${fgcolor_reset}"
	rm -rf ~/.fzf
	git clone --depth 1 https://github.com/junegunn/fzf.git ~/.fzf
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
