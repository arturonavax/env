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

# 4. Fast Node Manager (fnm)
if ! command -v fnm >/dev/null 2>&1; then
	log_step "Installing Fast Node Manager (fnm)..."
	curl -fsSL https://fnm.vercel.app/install | bash -s -- --skip-shell
	export PATH="$HOME/.local/share/fnm:$PATH"
	if command -v fnm >/dev/null 2>&1; then
		eval "$(fnm env --use-on-cd 2>/dev/null || true)"
	fi
	log_success "fnm installed."
fi

# 5. pnpm (Fast, disk space efficient package manager)
if ! command -v pnpm >/dev/null 2>&1; then
	log_step "Installing pnpm (${PNPM_VERSION})..."
	tmp_shell="${SHELL:-/bin/bash}"
	export SHELL="bash"
	curl -fsSL https://get.pnpm.io/install.sh | env PNPM_VERSION="${PNPM_VERSION}" sh -
	export SHELL="$tmp_shell"
	export PNPM_HOME="$HOME/.local/share/pnpm"
	export PATH="$PNPM_HOME:$PATH"
	log_success "pnpm installed."
fi

# 6. Go (Golang)
if ! command -v go >/dev/null 2>&1; then
	log_step "Installing Go language runtime..."
	if [[ -f ./src/remotes/install_golang.sh ]]; then
		bash ./src/remotes/install_golang.sh -i
	else
		curl -fsSL "${REMOTE_PRIMARY_URL}/install_golang.sh" | bash -s -- -i
	fi
	export PATH="/usr/local/go/bin:$HOME/go/bin:$PATH"
	log_success "Go language runtime installed."
fi

# 7. Python (pyenv / system python)
log_step "Configuring Python environment..."
bash ./src/remotes/install_python.sh -i

# 8. Load Development Tools Paths
if [[ -f ./files/zsh/.tools.sh ]]; then
	# shellcheck source=/dev/null
	source ./files/zsh/.tools.sh || true
fi

# 9. Node.js via fnm
if command -v fnm >/dev/null 2>&1; then
	if ! command -v node >/dev/null 2>&1 || ! command -v node | grep -q "fnm"; then
		log_step "Installing latest Node.js via fnm..."
		fnm install --latest
		latest_node="$(fnm list 2>/dev/null | sort --version-sort | awk '{print $2}' | tail -n 1)"
		if [[ -n "$latest_node" ]]; then
			fnm use "$latest_node"
			fnm default "$latest_node"
			log_success "Node.js ${latest_node} configured as default."
		fi
	fi
fi

# Decouple and disable Corepack to prevent npm/pnpm interference
if command -v corepack >/dev/null 2>&1; then
	corepack disable 2>/dev/null || true
fi

# 10. Synchronize ~/.tools.sh into user home
cp -f ./files/zsh/.tools.sh "$HOME/.tools.sh"
bash ./src/remotes/add_lines.sh tools

log_success "All development requirements and languages verified successfully!"
