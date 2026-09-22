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

	sudo rm -rf ./downloads/ 2>/dev/null || rm -rf ./downloads/ 2>/dev/null || :
	mkdir -p ./downloads/

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

			## Ghostty build dependencies (Debian and Ubuntu)
			sudo apt install -y libgtk-4-dev libadwaita-1-dev gettext libxml2-utils patchelf xz-utils
			sudo apt install -y libgtk4-layer-shell-dev 2>/dev/null || :
			sudo apt install -y gcc-multilib 2>/dev/null || :

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

			sudo apt install -y clang

		elif [[ "$ID_LIKE" == *"rhel"* || "$ID_LIKE" == *"centos"* || "$ID_LIKE" == *"fedora"* || "$ID" == *"fedora"* ]]; then
			sudo dnf update -y

			sudo dnf install -y ncurses ncurses-term ncurses-devel

			sudo dnf install -y libevent

			sudo dnf install -y fontconfig

			## Ghostty build dependencies (Fedora / RHEL)
			sudo dnf install -y gtk4-devel libadwaita-devel gettext pkgconf patchelf xz
			sudo dnf install -y gtk4-layer-shell-devel 2>/dev/null || :
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

		## Ghostty build dependencies (macOS)
		brew install gettext

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
		brew install tidy-html5 protobuf john-jumbo fd

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
	# Install Ghostty (Compiled from source with snap as fallback)
	echo -e "${fgcolor_white_bold}[Terminal Installer]: - Installing/Compiling Ghostty (${GHOSTTY_VERSION}) from source...${fgcolor_reset}"

	if [[ "$(uname -s)" == "Linux" ]]; then
		ghostty_already_compiled=false

		if [[ -f /usr/bin/ghostty ]]; then
			sudo chmod 755 /usr/bin/ghostty 2>/dev/null || :
			sudo chmod 755 /usr/lib/libghostty* /usr/lib/libgtk4-layer-shell* /usr/lib64/libghostty* /usr/lib64/libgtk4-layer-shell* 2>/dev/null || :
			sudo chmod 644 /usr/include/ghostty* /usr/include/gtk4-layer-shell* 2>/dev/null || :
			sudo chmod -R a+rX /usr/share/ghostty /usr/share/applications/*ghostty* /usr/share/icons/hicolor/*/apps/*ghostty* /usr/share/terminfo/*/*ghostty* 2>/dev/null || :
			sudo ldconfig 2>/dev/null || :
			installed_ver="$(/usr/bin/ghostty +version 2>/dev/null || /usr/bin/ghostty --version 2>/dev/null)"
			if [[ "$installed_ver" == *"${GHOSTTY_VERSION}"* ]]; then
				echo -e "${fgcolor_green_bold}[Terminal Installer]: Ghostty ${GHOSTTY_VERSION} is already compiled from source at /usr/bin/ghostty.${fgcolor_reset}"
				ghostty_already_compiled=true
				sudo mkdir -p /usr/share/ghostty 2>/dev/null || :
				sudo touch /usr/share/ghostty/.compiled_from_source 2>/dev/null || :
				sudo chmod a+r /usr/share/ghostty/.compiled_from_source 2>/dev/null || :
			fi
		fi

		ghostty_installed=false
		if [[ "$ghostty_already_compiled" == true ]]; then
			ghostty_installed=true
			if snap list ghostty &>/dev/null; then
				echo -e "${fgcolor_white_bold}[Terminal Installer]: Removing uncompiled snap Ghostty...${fgcolor_reset}"
				sudo snap remove ghostty 2>/dev/null || sudo snap remove --purge ghostty 2>/dev/null || :
			fi
		else
			if snap list ghostty &>/dev/null || [[ "$(command -v ghostty 2>/dev/null)" != "" ]]; then
				echo -e "${fgcolor_yellow_bold}[Terminal Installer]: Ghostty is currently installed without source compilation. Replacing with compiled version...${fgcolor_reset}"
			fi

			# Install Arch Linux build dependencies if pacman is present
			if [[ "$(command -v pacman)" != "" ]]; then
				sudo pacman -S --noconfirm --needed gtk4 pkgconf libadwaita gettext patchelf xz 2>/dev/null || :
				sudo pacman -S --noconfirm --needed gtk4-layer-shell 2>/dev/null || :
			fi

			zig_cmd=""
			if [[ -x "/opt/zig-${ZIG_VERSION}/zig" ]]; then
				zig_cmd="/opt/zig-${ZIG_VERSION}/zig"
			elif [[ "$(command -v zig)" != "" && "$(zig version 2>/dev/null)" == "${ZIG_VERSION}"* ]]; then
				zig_cmd="$(command -v zig)"
			else
				echo -e "${fgcolor_white_bold}[Terminal Installer]: - - Downloading Zig ${ZIG_VERSION}...${fgcolor_reset}"
				zig_arch=""
				case "$(uname -m)" in
				x86_64) zig_arch="x86_64" ;;
				aarch64 | arm64) zig_arch="aarch64" ;;
				esac

				if [[ -n "$zig_arch" ]]; then
					zig_tar="zig-${zig_arch}-linux-${ZIG_VERSION}.tar.xz"
					zig_url="https://ziglang.org/download/${ZIG_VERSION}/${zig_tar}"
					if curl -fsSL "$zig_url" -o "./downloads/${zig_tar}"; then
						sudo mkdir -p /opt
						sudo rm -rf "/opt/zig-${ZIG_VERSION}"
						sudo tar -C /opt -xf "./downloads/${zig_tar}"
						sudo mv "/opt/zig-${zig_arch}-linux-${ZIG_VERSION}" "/opt/zig-${ZIG_VERSION}" 2>/dev/null || :
						sudo chmod -R a+rX "/opt/zig-${ZIG_VERSION}" 2>/dev/null || :
						if [[ -x "/opt/zig-${ZIG_VERSION}/zig" ]]; then
							zig_cmd="/opt/zig-${ZIG_VERSION}/zig"
							sudo ln -sf "/opt/zig-${ZIG_VERSION}/zig" /usr/local/bin/zig 2>/dev/null || :
						fi
					fi
				fi
			fi

			if [[ -n "$zig_cmd" && -x "$zig_cmd" ]]; then
				ghostty_tar="ghostty-${GHOSTTY_VERSION}.tar.gz"
				ghostty_url="https://release.files.ghostty.org/${GHOSTTY_VERSION}/${ghostty_tar}"
				echo -e "${fgcolor_white_bold}[Terminal Installer]: - - Downloading Ghostty (${GHOSTTY_VERSION}) source tarball...${fgcolor_reset}"
				if curl -fsSL "$ghostty_url" -o "./downloads/${ghostty_tar}"; then
					# Verify signature with minisign if available
					if command -v minisign &>/dev/null; then
						minisign_pubkey="RWQlAjJC23149WL2sEpT/l0QKy7hMIFhYdQOFy0Z7z7PbneUgvlsnYcV"
						if curl -fsSL "${ghostty_url}.minisig" -o "./downloads/${ghostty_tar}.minisig" 2>/dev/null; then
							if minisign -Vm "./downloads/${ghostty_tar}" -P "$minisign_pubkey" -x "./downloads/${ghostty_tar}.minisig" &>/dev/null; then
								echo -e "${fgcolor_green_bold}[Terminal Installer]: Ghostty source tarball verified with minisign.${fgcolor_reset}"
							fi
						fi
					fi

					sudo rm -rf "./downloads/ghostty-${GHOSTTY_VERSION}" 2>/dev/null || rm -rf "./downloads/ghostty-${GHOSTTY_VERSION}" 2>/dev/null || :
					tar -C ./downloads -xzf "./downloads/${ghostty_tar}"

					build_flags=(-p /usr -Doptimize=ReleaseFast --cache-dir /tmp/ghostty-zig-cache --global-cache-dir /tmp/ghostty-zig-global-cache)
					if ! pkg-config --exists gtk4-layer-shell-0 2>/dev/null; then
						build_flags+=(-fno-sys=gtk4-layer-shell)
					fi

					build_success=false
					for attempt in 1 2 3; do
						echo -e "${fgcolor_white_bold}[Terminal Installer]: - - Compiling Ghostty with Zig (${zig_cmd}) [attempt ${attempt}/3]...${fgcolor_reset}"
						if (
							cd "./downloads/ghostty-${GHOSTTY_VERSION}" || exit 1
							sudo bash -c 'umask 022 && "$@"' -- "$zig_cmd" build "${build_flags[@]}"
						); then
							build_success=true
							break
						else
							echo -e "${fgcolor_yellow_bold}[Terminal Installer]: Build attempt ${attempt} failed. Retrying in 2 seconds...${fgcolor_reset}"
							sleep 2
						fi
					done

					sudo chmod 755 /usr/bin/ghostty 2>/dev/null || :
					sudo chmod 755 /usr/lib/libghostty* /usr/lib/libgtk4-layer-shell* /usr/lib64/libghostty* /usr/lib64/libgtk4-layer-shell* 2>/dev/null || :
					sudo chmod 644 /usr/include/ghostty* /usr/include/gtk4-layer-shell* 2>/dev/null || :
					sudo chmod -R a+rX /usr/share/ghostty 2>/dev/null || :
					sudo chmod a+rX /usr/share/applications/*ghostty*.desktop 2>/dev/null || :
					sudo chmod -R a+rX /usr/share/icons/hicolor/*/apps/*ghostty* 2>/dev/null || :
					sudo chmod -R a+rX /usr/share/terminfo/*/xterm-ghostty* /usr/share/terminfo/*/ghostty* 2>/dev/null || :
					sudo ldconfig 2>/dev/null || :

					if [[ "$build_success" == true && -f /usr/bin/ghostty ]]; then
						if [[ "${build_flags[*]}" == *"-fno-sys=gtk4-layer-shell"* ]] && command -v patchelf &>/dev/null; then
							sudo patchelf --set-rpath '$ORIGIN/../lib' /usr/bin/ghostty 2>/dev/null || :
						fi
						if command -v update-desktop-database &>/dev/null; then
							sudo update-desktop-database /usr/share/applications 2>/dev/null || :
						fi
						if command -v gtk-update-icon-cache &>/dev/null; then
							sudo gtk-update-icon-cache -f -t /usr/share/icons/hicolor 2>/dev/null || :
						fi
						if command -v glib-compile-schemas &>/dev/null; then
							sudo glib-compile-schemas /usr/share/glib-2.0/schemas 2>/dev/null || :
						fi
						if command -v update-mime-database &>/dev/null; then
							sudo update-mime-database /usr/share/mime 2>/dev/null || :
						fi
						sudo ldconfig 2>/dev/null || :

						# Mark as compiled from source
						sudo mkdir -p /usr/share/ghostty
						sudo touch /usr/share/ghostty/.compiled_from_source
						sudo chmod a+r /usr/share/ghostty/.compiled_from_source

						# Remove previous uncompiled installations to complete replacement
						if snap list ghostty &>/dev/null; then
							echo -e "${fgcolor_white_bold}[Terminal Installer]: Removing uncompiled snap Ghostty...${fgcolor_reset}"
							sudo snap remove ghostty 2>/dev/null || sudo snap remove --purge ghostty 2>/dev/null || :
						fi
						if [[ "$ID_LIKE" == *"debian"* || "$ID_LIKE" == *"ubuntu"* ]]; then
							sudo apt remove -y ghostty 2>/dev/null || :
						elif [[ "$ID_LIKE" == *"rhel"* || "$ID_LIKE" == *"centos"* || "$ID_LIKE" == *"fedora"* || "$ID" == *"fedora"* ]]; then
							sudo dnf remove -y ghostty 2>/dev/null || :
						elif [[ "$(command -v pacman)" != "" ]]; then
							sudo pacman -R --noconfirm ghostty 2>/dev/null || :
						fi

						ghostty_installed=true
						echo -e "${fgcolor_green_bold}[Terminal Installer]: Ghostty ${GHOSTTY_VERSION} compiled and installed successfully at /usr/bin/ghostty.${fgcolor_reset}"
					fi

					sudo rm -rf /tmp/ghostty-zig-* 2>/dev/null || :
					sudo rm -rf "./downloads/ghostty-${GHOSTTY_VERSION}" 2>/dev/null || rm -rf "./downloads/ghostty-${GHOSTTY_VERSION}" 2>/dev/null || :
				fi
			fi
		fi

		# Fallback to snap or distribution package manager
		if [[ "$ghostty_installed" == false ]]; then
			echo -e "${fgcolor_yellow_bold}[Terminal Installer]: Ghostty source compilation failed or was skipped. Falling back to snap / package manager...${fgcolor_reset}"
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
		ghostty_already_compiled=false

		if [[ -f /Applications/Ghostty.app/.compiled_from_source && -x /Applications/Ghostty.app/Contents/MacOS/ghostty ]]; then
			installed_ver="$(/Applications/Ghostty.app/Contents/MacOS/ghostty +version 2>/dev/null || /Applications/Ghostty.app/Contents/MacOS/ghostty --version 2>/dev/null)"
			if [[ "$installed_ver" == *"${GHOSTTY_VERSION}"* ]]; then
				echo -e "${fgcolor_green_bold}[Terminal Installer]: Ghostty ${GHOSTTY_VERSION} is already compiled from source at /Applications/Ghostty.app.${fgcolor_reset}"
				ghostty_already_compiled=true
			fi
		fi

		ghostty_installed=false
		if [[ "$ghostty_already_compiled" == true ]]; then
			ghostty_installed=true
		else
			if [[ -d /Applications/Ghostty.app || "$(brew list --cask ghostty 2>/dev/null)" != "" || "$(command -v ghostty 2>/dev/null)" != "" ]]; then
				echo -e "${fgcolor_yellow_bold}[Terminal Installer]: Ghostty is currently installed without source compilation. Replacing with compiled version...${fgcolor_reset}"
			fi

			# Check Xcode developer directory
			if [[ -d "/Applications/Xcode.app/Contents/Developer" && "$(xcode-select -p 2>/dev/null)" != "/Applications/Xcode.app/Contents/Developer" ]]; then
				sudo xcode-select --switch /Applications/Xcode.app/Contents/Developer 2>/dev/null || :
			fi

			zig_cmd=""
			if [[ -x "/opt/zig-${ZIG_VERSION}/zig" ]]; then
				zig_cmd="/opt/zig-${ZIG_VERSION}/zig"
			elif [[ "$(command -v zig)" != "" && "$(zig version 2>/dev/null)" == "${ZIG_VERSION}"* ]]; then
				zig_cmd="$(command -v zig)"
			else
				echo -e "${fgcolor_white_bold}[Terminal Installer]: - - Downloading Zig ${ZIG_VERSION}...${fgcolor_reset}"
				zig_arch=""
				case "$(uname -m)" in
				arm64 | aarch64) zig_arch="aarch64" ;;
				x86_64) zig_arch="x86_64" ;;
				esac

				if [[ -n "$zig_arch" ]]; then
					zig_tar="zig-${zig_arch}-macos-${ZIG_VERSION}.tar.xz"
					zig_url="https://ziglang.org/download/${ZIG_VERSION}/${zig_tar}"
					if curl -fsSL "$zig_url" -o "./downloads/${zig_tar}"; then
						tar -C ./downloads -xf "./downloads/${zig_tar}"
						if [[ -x "./downloads/zig-${zig_arch}-macos-${ZIG_VERSION}/zig" ]]; then
							zig_cmd="$PWD/downloads/zig-${zig_arch}-macos-${ZIG_VERSION}/zig"
							sudo mkdir -p /opt 2>/dev/null || :
							sudo rm -rf "/opt/zig-${ZIG_VERSION}" 2>/dev/null || :
							if sudo mv "./downloads/zig-${zig_arch}-macos-${ZIG_VERSION}" "/opt/zig-${ZIG_VERSION}" 2>/dev/null; then
								zig_cmd="/opt/zig-${ZIG_VERSION}/zig"
							fi
							sudo ln -sf "$zig_cmd" /usr/local/bin/zig 2>/dev/null || :
						fi
					fi
				fi
			fi

			if [[ -n "$zig_cmd" && -x "$zig_cmd" ]]; then
				ghostty_tar="ghostty-${GHOSTTY_VERSION}.tar.gz"
				ghostty_url="https://release.files.ghostty.org/${GHOSTTY_VERSION}/${ghostty_tar}"
				echo -e "${fgcolor_white_bold}[Terminal Installer]: - - Downloading Ghostty (${GHOSTTY_VERSION}) source tarball...${fgcolor_reset}"
				if curl -fsSL "$ghostty_url" -o "./downloads/${ghostty_tar}"; then
					# Verify signature with minisign if available
					if command -v minisign &>/dev/null; then
						minisign_pubkey="RWQlAjJC23149WL2sEpT/l0QKy7hMIFhYdQOFy0Z7z7PbneUgvlsnYcV"
						if curl -fsSL "${ghostty_url}.minisig" -o "./downloads/${ghostty_tar}.minisig" 2>/dev/null; then
							if minisign -Vm "./downloads/${ghostty_tar}" -P "$minisign_pubkey" -x "./downloads/${ghostty_tar}.minisig" &>/dev/null; then
								echo -e "${fgcolor_green_bold}[Terminal Installer]: Ghostty source tarball verified with minisign.${fgcolor_reset}"
							fi
						fi
					fi

					rm -rf "./downloads/ghostty-${GHOSTTY_VERSION}" 2>/dev/null || :
					tar -C ./downloads -xzf "./downloads/${ghostty_tar}"

					build_success=false
					for attempt in 1 2 3; do
						echo -e "${fgcolor_white_bold}[Terminal Installer]: - - Compiling Ghostty with Zig (${zig_cmd}) [attempt ${attempt}/3]...${fgcolor_reset}"
						if (
							cd "./downloads/ghostty-${GHOSTTY_VERSION}" || exit 1
							"$zig_cmd" build -Doptimize=ReleaseFast --cache-dir /tmp/ghostty-zig-cache --global-cache-dir /tmp/ghostty-zig-global-cache
						); then
							build_success=true
							break
						else
							echo -e "${fgcolor_yellow_bold}[Terminal Installer]: Build attempt ${attempt} failed. Retrying in 2 seconds...${fgcolor_reset}"
							sleep 2
						fi
					done

					if [[ "$build_success" == true && -d "./downloads/ghostty-${GHOSTTY_VERSION}/zig-out/Ghostty.app" ]]; then
						# Remove uncompiled brew cask if present to complete replacement
						if brew list --cask ghostty &>/dev/null; then
							echo -e "${fgcolor_white_bold}[Terminal Installer]: Removing uncompiled brew cask Ghostty...${fgcolor_reset}"
							brew uninstall --cask ghostty 2>/dev/null || :
						fi

						rm -rf /Applications/Ghostty.app
						cp -R "./downloads/ghostty-${GHOSTTY_VERSION}/zig-out/Ghostty.app" /Applications/
						xattr -d com.apple.quarantine /Applications/Ghostty.app 2>/dev/null || :
						touch /Applications/Ghostty.app/.compiled_from_source

						mkdir -p "$HOME/.local/bin"
						sudo ln -sf /Applications/Ghostty.app/Contents/MacOS/ghostty /usr/local/bin/ghostty 2>/dev/null || \
							ln -sf /Applications/Ghostty.app/Contents/MacOS/ghostty "$HOME/.local/bin/ghostty"

						ghostty_installed=true
						echo -e "${fgcolor_green_bold}[Terminal Installer]: Ghostty ${GHOSTTY_VERSION} compiled and installed successfully at /Applications/Ghostty.app.${fgcolor_reset}"
					fi

					rm -rf /tmp/ghostty-zig-* 2>/dev/null || :
					rm -rf "./downloads/ghostty-${GHOSTTY_VERSION}" 2>/dev/null || :
				fi
			fi
		fi

		# Fallback to Homebrew cask
		if [[ "$ghostty_installed" == false ]]; then
			echo -e "${fgcolor_yellow_bold}[Terminal Installer]: Ghostty source compilation failed or was skipped. Falling back to Homebrew cask...${fgcolor_reset}"
			brew tap --force homebrew/cask
			brew install --cask ghostty || brew upgrade --cask ghostty || :
			xattr -d com.apple.quarantine /Applications/Ghostty.app 2>/dev/null || :
		fi

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
				make -j"$(nproc 2>/dev/null || sysctl -n hw.ncpu 2>/dev/null || echo 2)"
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
	if curl -fsSL -o terminfo.src.gz https://invisible-island.net/datafiles/current/terminfo.src.gz 2>/dev/null; then
		gunzip -f terminfo.src.gz 2>/dev/null && sudo tic -xe tmux-256color terminfo.src 2>/dev/null || :
	fi
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
	mkdir -p "$HOME/.zsh"
	for plugin_spec in \
		"https://github.com/zsh-users/zsh-autosuggestions $HOME/.zsh/zsh-autosuggestions" \
		"https://github.com/zdharma-continuum/fast-syntax-highlighting $HOME/.zsh/fast-syntax-highlighting" \
		"https://github.com/zsh-users/zsh-completions.git $HOME/.zsh/zsh-completions" \
		"https://github.com/Aloxaf/fzf-tab $HOME/.zsh/fzf-tab"; do
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
	if [[ -f ~/.tmux/plugins/tpm/scripts/install_plugins.sh ]]; then
		tmux start-server 2>/dev/null || :
		tmux set-environment -g TMUX_PLUGIN_MANAGER_PATH "$HOME/.tmux/plugins/" 2>/dev/null || :
		tmux source-file ~/.tmux.conf 2>/dev/null || :
		bash ~/.tmux/plugins/tpm/scripts/install_plugins.sh || :
	fi

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
	if [[ -f ~/.fzf/install ]]; then
		~/.fzf/install --all --no-update-rc || :
	fi

	echo

	# ---

	echo -e "${fgcolor_white_bold}[Terminal Installer]: ${fgcolor_green_bold}✔️ Terminal successfully installed!${fgcolor_reset}"
	echo -e "${fgcolor_white_bold}[Terminal Installer]: (Restarting the computer to use Zsh for the first time)${fgcolor_reset}"

	echo -en "$fgcolor_reset"

	if [[ -d ./downloads/ ]]; then
		sudo rm -rf ./downloads/ 2>/dev/null || rm -rf ./downloads/ 2>/dev/null || :
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
