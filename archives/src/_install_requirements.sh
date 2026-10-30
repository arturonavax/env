#!/usr/bin/env bash
# ==============================================================================
# src/_install_requirements.sh
# Multi-platform development dependencies, compilers, language runtimes (Rust,
# Node/fnm, pnpm, Go, Python) and base tools installer.
# Part of arturonavax/env development environment.
# ==============================================================================

# Anchoring
while [[ ! -d ./.git/ && ! -d ./files/ && ! -d ./src/ ]]; do
	if [[ "$PWD" == "/" ]]; then
		echo "[Requirements Error]: Repository root not found." >&2
		exit 1
	fi
	if ! cd ..; then
		echo "[Requirements Error]: Failed to change directory." >&2
		exit 1
	fi
done

source ./src/lib/strict.sh
source ./src/lib/colors.sh
source ./src/lib/logging.sh
source ./src/lib/platform.sh
source ./src/lib/packages.sh
source ./src/lib/download.sh
source ./src/lib/toolchain.sh
source ./src/lib/remote_env.sh
source ./src/remotes/_versions.sh

log_box "Requirements Installer" \
	"Platform: $(detect_os) ($(detect_distro) - $(detect_arch))" \
	"Installing build toolchains, system headers, and language runtimes."

# 1. macOS Command Line Tools & Homebrew bootstrap
if [[ "$(detect_os)" == "darwin" ]]; then
	if [[ ! -e "/Library/Developer/CommandLineTools/usr/bin/git" ]]; then
		log_step "Installing macOS Command Line Tools..."
		bash ./src/remotes/macos_install_clt.sh
	fi

	# Rosetta 2 on Apple Silicon
	if [[ "$(detect_arch)" == "arm64" ]]; then
		softwareupdate --install-rosetta --agree-to-license 2>/dev/null || true
	fi

	# Homebrew bootstrap if missing
	if ! command -v brew >/dev/null 2>&1; then
		log_step "Installing Homebrew..."
		/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
		brewbin="/opt/homebrew/bin/brew"
		[[ ! -f "$brewbin" ]] && brewbin="/usr/local/bin/brew"
		[[ -f "$brewbin" ]] && eval "$("$brewbin" shellenv)"
	fi
fi

# 2. Update Package Index & Install Development Packages
pkg_update_cache

log_step "Installing development toolchains and library headers..."
pkg_install \
	build-essential \
	ssl-dev \
	zlib-dev \
	bz2-dev \
	readline-dev \
	sqlite-dev \
	ffi-dev \
	ncurses-dev \
	ncurses-term \
	xz-dev \
	xml2-dev \
	xslt-dev \
	tk-dev \
	curl \
	wget \
	git \
	unzip \
	zstd \
	make \
	gcc \
	fontconfig \
	pkg-config \
	wmctrl \
	xdotool \
	x11-utils \
	keyd

# 3. Rust Toolchain (rustup & cargo)
if ! command -v cargo >/dev/null 2>&1; then
	log_step "Installing Rust toolchain (rustup)..."
	curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y --no-modify-path
	export PATH="$HOME/.cargo/bin:$PATH"
	log_success "Rust toolchain installed."
fi

# 4. Node.js & Fast Node Manager (fnm)
log_step "Configuring Node.js runtime via Toolchain..."
toolchain_install node

# 5. pnpm (Fast, disk space efficient package manager)
log_step "Configuring pnpm package manager via Toolchain..."
toolchain_install pnpm

# 6. Go (Golang) via Multi-Tier Toolchain (GVM primary, standalone & package fallbacks)
log_step "Configuring Go language runtime via Toolchain..."
toolchain_install golang
if [[ -s "$HOME/.gvm/scripts/gvm" ]]; then
	# shellcheck source=/dev/null
	source "$HOME/.gvm/scripts/gvm"
fi
export PATH="/usr/local/go/bin:$HOME/go/bin:$PATH"

# 7. Python (pyenv / system python)
log_step "Configuring Python environment via Toolchain..."
toolchain_install python

# 8. Load Development Tools Paths
if [[ -f ./files/zsh/.tools.sh ]]; then
	# shellcheck source=/dev/null
	source ./files/zsh/.tools.sh || true
fi

# 10. Synchronize ~/.tools.sh into user home
cp -f ./files/zsh/.tools.sh "$HOME/.tools.sh"
bash ./src/remotes/add_lines.sh tools

log_success "All development requirements and languages verified successfully!"
