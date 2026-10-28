#!/usr/bin/env bash
# ==============================================================================
# src/lib/toolchain.sh
# Multi-Tier Resilient Toolchain & Declarative Dependency Provisioner.
# Implements Strategy Pattern & Chain of Responsibility for tool provisioning:
#   Tier 1: Pre-compiled static binary (fastest, zero compile overhead)
#   Tier 2: Native Package Manager (apt, dnf, pacman, brew, zypper, pkg)
#   Tier 3: Verified curl-to-shell installer script (GVM, rustup, fnm, pnpm, atuin)
#   Tier 4: Build from source compilation (cargo, go, make, zig fallback)
# Part of arturonavax/env development environment.
# ==============================================================================

# Prevent double sourcing
[[ -n "${_TOOLCHAIN_SH_LOADED:-}" ]] && return 0
_TOOLCHAIN_SH_LOADED=1

_LIB_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=/dev/null
source "${_LIB_DIR}/strict.sh" 2>/dev/null || true
# shellcheck source=/dev/null
source "${_LIB_DIR}/colors.sh" 2>/dev/null || true
# shellcheck source=/dev/null
source "${_LIB_DIR}/logging.sh" 2>/dev/null || true
# shellcheck source=/dev/null
source "${_LIB_DIR}/platform.sh" 2>/dev/null || true
# shellcheck source=/dev/null
source "${_LIB_DIR}/packages.sh" 2>/dev/null || true
# shellcheck source=/dev/null
source "${_LIB_DIR}/download.sh" 2>/dev/null || true
# shellcheck source=/dev/null
source "${_LIB_DIR}/remote_env.sh" 2>/dev/null || true
# shellcheck source=/dev/null
source "${_LIB_DIR}/../remotes/_versions.sh" 2>/dev/null || true

# Execution Strategy Constants
export STRATEGY_BINARY="binary"
export STRATEGY_PACKAGE="package"
export STRATEGY_SCRIPT="script"
export STRATEGY_SOURCE="source"

# ------------------------------------------------------------------------------
# Dependency Resolver
# ------------------------------------------------------------------------------
# Ensures all prerequisite system packages for a tool are installed via packages.sh
toolchain_ensure_dependencies() {
	local tool_name="$1"
	shift
	local deps=("$@")

	if [[ ${#deps[@]} -eq 0 ]]; then
		return 0
	fi

	log_step "Resolving prerequisite dependencies for ${BOLD_WHITE}${tool_name}${RESET}: ${deps[*]}"
	pkg_install "${deps[@]}"
}

# ------------------------------------------------------------------------------
# Tool Providers & Fallback Handlers
# ------------------------------------------------------------------------------

# 1. Golang / GVM Provider
# GVM is the primary Go version manager; install_golang.sh is the standalone fallback
_install_golang_script_gvm() {
	log_step "Installing Go Version Manager (GVM) as primary Golang manager..."

	# GVM prerequisite packages
	toolchain_ensure_dependencies "gvm" bison make gcc binutils curl git

	# Run official GVM installer script
	bash < <(curl -s -S -L https://raw.githubusercontent.com/moovweb/gvm/master/binscripts/gvm-installer) || return 1

	if [[ -s "$HOME/.gvm/scripts/gvm" ]]; then
		# shellcheck source=/dev/null
		source "$HOME/.gvm/scripts/gvm"
		log_success "GVM installed successfully in ~/.gvm."

		# Attempt to install or configure binary Go release via GVM if network permits
		if command -v gvm >/dev/null 2>&1; then
			log_step "Configuring default Go environment via GVM..."
			gvm install go1.4 -B 2>/dev/null || true
		fi
		return 0
	fi
	return 1
}

_install_golang_script_standalone() {
	log_step "Installing standalone Golang runtime via install_golang.sh..."
	local script_path="${REPO_ROOT}/src/remotes/install_golang.sh"
	if [[ -f "$script_path" ]]; then
		bash "$script_path" -i || return 1
	else
		curl -fsSL "${REMOTE_PRIMARY_URL}/install_golang.sh" | bash -s -- -i || return 1
	fi
	return 0
}

_install_golang_package() {
	log_step "Installing Go via native package manager..."
	pkg_install golang
}

# 2. Herdr Multiplexer Provider
_install_herdr_script() {
	log_step "Installing Herdr terminal & agent multiplexer via official installer..."
	curl -fsSL https://herdr.dev/install.sh | bash || return 1
	mkdir -p "$HOME/.local/bin"
	if [[ -f "$HOME/.local/bin/herdr" ]]; then
		sudo ln -sf "$HOME/.local/bin/herdr" /usr/local/bin/herdr 2>/dev/null || true
		return 0
	fi
	return 1
}

# 3. Atuin Shell History Provider
_install_atuin_script() {
	log_step "Installing Atuin SQLite shell history via official binary installer..."
	curl --proto '=https' --tlsv1.2 -LsSf https://setup.atuin.sh | sh || return 1
	return 0
}

_install_atuin_package() {
	log_step "Installing Atuin via native package manager..."
	pkg_install atuin
}

_install_atuin_source() {
	log_step "Compiling Atuin from source via cargo..."
	command -v cargo >/dev/null 2>&1 || return 1
	cargo install atuin --locked || return 1
	return 0
}

# 4. RTK Token Compressor Provider
_install_rtk_script() {
	log_step "Installing RTK via official pre-compiled release script..."
	curl -fsSL https://raw.githubusercontent.com/rtk-ai/rtk/refs/heads/master/install.sh | sh || return 1
	return 0
}

_install_rtk_source() {
	log_step "Compiling RTK from source via cargo..."
	command -v cargo >/dev/null 2>&1 || return 1
	cargo install rtk || return 1
	return 0
}

# 5. Neovim Provider
_install_neovim_binary() {
	local os arch
	os="$(detect_os)"
	arch="$(detect_arch)"

	if [[ "$os" == "linux" && "$arch" == "x86_64" ]]; then
		log_step "Downloading official Neovim x86_64 pre-compiled release tarball..."
		local tmp_dir
		tmp_dir="$(mktemp -d "${TMPDIR:-/tmp}/nvim_bin.XXXXXX")"
		local tar_file="${tmp_dir}/nvim-linux-x86_64.tar.gz"

		if curl --proto '=https' --tlsv1.2 -fsSL "https://github.com/neovim/neovim/releases/download/v0.10.4/nvim-linux-x86_64.tar.gz" -o "$tar_file"; then
			sudo rm -rf /opt/nvim-linux-x86_64
			sudo tar -C /opt -xzf "$tar_file"
			sudo ln -sf /opt/nvim-linux-x86_64/bin/nvim /usr/local/bin/nvim
			mkdir -p "$HOME/.local/bin"
			ln -sf /opt/nvim-linux-x86_64/bin/nvim "$HOME/.local/bin/nvim"
			rm -rf "$tmp_dir"
			return 0
		fi
		rm -rf "$tmp_dir"
	fi
	return 1
}

_install_neovim_package() {
	log_step "Installing Neovim via native package manager..."
	pkg_install neovim
}

_install_neovim_source() {
	log_step "Compiling Neovim from source repository..."
	toolchain_ensure_dependencies "neovim-source" ninja-build cmake gettext curl git make gcc
	local build_dir
	build_dir="$(mktemp -d "${TMPDIR:-/tmp}/neovim_src.XXXXXX")"
	if git clone --depth 1 --branch stable https://github.com/neovim/neovim.git "$build_dir"; then
		(
			cd "$build_dir" || exit 1
			make CMAKE_BUILD_TYPE=RelWithDebInfo
			sudo make install
		)
		local res=$?
		rm -rf "$build_dir"
		return "$res"
	fi
	rm -rf "$build_dir"
	return 1
}

# 6. Python & pyenv Provider
_install_python_pyenv() {
	log_step "Installing Python via pyenv (${PYTHON_VERSION:-3.12.9})..."
	local script_path="${REPO_ROOT}/src/remotes/install_python.sh"
	if [[ -f "$script_path" ]]; then
		bash "$script_path" -i || return 1
	else
		curl -fsSL "${REMOTE_PRIMARY_URL}/install_python.sh" | bash -s -- -i || return 1
	fi
	return 0
}

_install_python_package() {
	log_step "Installing Python 3, pip, and pipx via native package manager..."
	pkg_install python3 python3-pip python3-venv pipx
}

# 7. Node.js & Fast Node Manager (fnm) Provider
_install_node_fnm() {
	log_step "Installing Fast Node Manager (fnm) & Node.js..."
	if ! command -v fnm >/dev/null 2>&1; then
		curl -fsSL https://fnm.vercel.app/install | bash -s -- --skip-shell || return 1
	fi

	local fnm_bin_dir="$HOME/.local/share/fnm"
	[[ "$(detect_os)" == "darwin" ]] && fnm_bin_dir="$HOME/Library/Application Support/fnm"

	if [[ -f "${fnm_bin_dir}/fnm" ]]; then
		export PATH="${fnm_bin_dir}:$PATH"
	fi

	if command -v fnm >/dev/null 2>&1; then
		local shell_type="bash"
		[[ -n "${ZSH_VERSION:-}" ]] && shell_type="zsh"
		eval "$(fnm env --use-on-cd --shell "$shell_type" 2>/dev/null || fnm env 2>/dev/null)"

		local target_node="${NODE_VERSION:-lts}"
		if [[ "$target_node" == "lts" ]]; then
			fnm install --lts || return 1
			fnm default lts-latest 2>/dev/null || true
			fnm use lts-latest 2>/dev/null || true
		else
			fnm install "$target_node" || return 1
			fnm default "$target_node" 2>/dev/null || true
			fnm use "$target_node" 2>/dev/null || true
		fi

		# Disable corepack prompts/interferences
		if command -v corepack >/dev/null 2>&1; then
			corepack disable 2>/dev/null || true
		fi
		return 0
	fi
	return 1
}

_install_node_package() {
	log_step "Installing Node.js via native package manager..."
	pkg_install nodejs npm
}

# 8. pnpm Provider
_install_pnpm_script() {
	local pnpm_ver="${PNPM_VERSION:-12.4.2}"
	log_step "Installing pnpm (${pnpm_ver}) via official installer..."
	local tmp_shell="${SHELL:-/bin/bash}"
	export SHELL="bash"
	curl -fsSL https://get.pnpm.io/install.sh | env PNPM_VERSION="${pnpm_ver}" sh - || {
		export SHELL="$tmp_shell"
		return 1
	}
	export SHELL="$tmp_shell"

	local pnpm_home="$HOME/.local/share/pnpm"
	[[ "$(detect_os)" == "darwin" ]] && pnpm_home="$HOME/Library/pnpm"
	export PNPM_HOME="$pnpm_home"
	export PATH="$PNPM_HOME:$PATH"

	if command -v pnpm >/dev/null 2>&1; then
		return 0
	fi
	return 1
}

_install_pnpm_npm() {
	log_step "Installing pnpm via npm fallback..."
	command -v npm >/dev/null 2>&1 || return 1
	npm install -g "pnpm@${PNPM_VERSION:-12.4.2}" || return 1
	return 0
}

_install_pnpm_package() {
	log_step "Installing pnpm via native package manager..."
	pkg_install pnpm
}

# ------------------------------------------------------------------------------
# Unified Toolchain Dispatcher & Fallback Chain
# ------------------------------------------------------------------------------
# Executes multi-tier installation with graceful fallback logging
# Usage:
#   toolchain_install <tool_name> [--force]
toolchain_install() {
	local tool="$1"
	local force=0
	[[ "${2:-}" == "--force" ]] && force=1

	case "$tool" in
	golang | go | gvm)
		if ((force == 0)) && (command -v go >/dev/null 2>&1 || [[ -s "$HOME/.gvm/scripts/gvm" ]]); then
			log_info "Golang is already installed: $(command -v go 2>/dev/null || echo "${HOME}/.gvm/scripts/gvm")"
			return 0
		fi

		local preferred="${GO_MANAGER_STRATEGY:-gvm}"

		log_box "Toolchain: Provisioning Golang" \
			"Selected Preference: ${preferred}" \
			"Supported Strategies: GVM, Standalone (install_golang.sh), Package Manager"

		if [[ "$preferred" == "standalone" ]]; then
			if _install_golang_script_standalone; then
				log_success "Golang provisioned via standalone release installer."
				return 0
			fi
			log_warn "Standalone installer failed. Falling back to GVM..."
			if _install_golang_script_gvm; then
				log_success "Golang provisioned via GVM."
				return 0
			fi
		elif [[ "$preferred" == "package" ]]; then
			if _install_golang_package; then
				log_success "Golang provisioned via native package manager."
				return 0
			fi
			log_warn "Package installation failed. Falling back to GVM..."
			if _install_golang_script_gvm; then
				log_success "Golang provisioned via GVM."
				return 0
			fi
		else
			# Default: GVM first
			if _install_golang_script_gvm; then
				log_success "Golang provisioned via GVM (Go Version Manager)."
				return 0
			fi
			log_warn "GVM installation strategy did not complete. Falling back to standalone installer script..."
			if _install_golang_script_standalone; then
				log_success "Golang provisioned via standalone release installer."
				return 0
			fi
		fi

		log_warn "Primary strategies exhausted. Falling back to native package manager..."
		if _install_golang_package; then
			log_success "Golang provisioned via native package manager."
			return 0
		fi

		log_error "All installation strategies failed for Golang."
		return 1
		;;

	herdr)
		if ((force == 0)) && command -v herdr >/dev/null 2>&1; then
			log_info "Herdr multiplexer is already installed: $(command -v herdr)"
			return 0
		fi

		log_box "Toolchain: Provisioning Herdr" \
			"Primary Strategy: Official static release binary script"

		if _install_herdr_script; then
			log_success "Herdr provisioned successfully."
			return 0
		fi

		log_error "Failed to provision Herdr multiplexer."
		return 1
		;;

	atuin)
		if ((force == 0)) && command -v atuin >/dev/null 2>&1; then
			log_info "Atuin is already installed: $(command -v atuin)"
			return 0
		fi

		log_box "Toolchain: Provisioning Atuin" \
			"Primary Strategy: Official standalone binary installer" \
			"Fallback 1: Native package manager" \
			"Fallback 2: Compile from source via Cargo"

		if _install_atuin_script; then
			log_success "Atuin provisioned via standalone release installer."
			return 0
		fi

		log_warn "Atuin standalone script failed. Falling back to native package manager..."
		if _install_atuin_package; then
			log_success "Atuin provisioned via native package manager."
			return 0
		fi

		log_warn "Package manager failed. Falling back to Cargo source compilation..."
		if _install_atuin_source; then
			log_success "Atuin provisioned via Cargo compilation from source."
			return 0
		fi

		log_error "All installation strategies failed for Atuin."
		return 1
		;;

	rtk)
		if ((force == 0)) && command -v rtk >/dev/null 2>&1; then
			log_info "RTK token compressor is already installed: $(command -v rtk)"
			return 0
		fi

		log_box "Toolchain: Provisioning RTK" \
			"Primary Strategy: Official pre-compiled release binary script" \
			"Fallback: Compile from source via Cargo"

		if _install_rtk_script; then
			log_success "RTK provisioned via official release script."
			return 0
		fi

		log_warn "RTK release script failed. Falling back to Cargo source compilation..."
		if _install_rtk_source; then
			log_success "RTK provisioned via Cargo compilation from source."
			return 0
		fi

		log_error "All installation strategies failed for RTK."
		return 1
		;;

	neovim | nvim)
		if ((force == 0)) && command -v nvim >/dev/null 2>&1; then
			log_info "Neovim is already installed: $(command -v nvim)"
			return 0
		fi

		log_box "Toolchain: Provisioning Neovim" \
			"Primary Strategy: Pre-compiled binary release tarball" \
			"Fallback 1: Native package manager" \
			"Fallback 2: Compile from source via CMake / Make"

		if _install_neovim_binary; then
			log_success "Neovim provisioned via pre-compiled release tarball."
			return 0
		fi

		log_warn "Pre-compiled binary failed. Falling back to native package manager..."
		if _install_neovim_package; then
			log_success "Neovim provisioned via native package manager."
			return 0
		fi

		log_warn "Package manager failed. Falling back to compilation from source..."
		if _install_neovim_source; then
			log_success "Neovim provisioned via compilation from source."
			return 0
		fi

		log_error "All installation strategies failed for Neovim."
		return 1
		;;

	python | pyenv | python3)
		if ((force == 0)) && (command -v pyenv >/dev/null 2>&1 && command -v python3 >/dev/null 2>&1); then
			log_info "Python is already configured: $(command -v python3) ($(python3 --version 2>&1))"
			return 0
		fi

		log_box "Toolchain: Provisioning Python" \
			"Primary Strategy: pyenv with Python ${PYTHON_VERSION:-3.12.9} and pipx" \
			"Fallback: Native distribution package manager"

		if _install_python_pyenv; then
			log_success "Python provisioned via pyenv."
			return 0
		fi

		log_warn "pyenv strategy failed. Falling back to native package manager..."
		if _install_python_package; then
			log_success "Python provisioned via native package manager."
			return 0
		fi

		log_error "All installation strategies failed for Python."
		return 1
		;;

	node | nodejs | fnm)
		if ((force == 0)) && command -v node >/dev/null 2>&1; then
			log_info "Node.js is already installed: $(command -v node) ($(node --version 2>&1))"
			return 0
		fi

		log_box "Toolchain: Provisioning Node.js" \
			"Primary Strategy: Fast Node Manager (fnm) with Node ${NODE_VERSION:-LTS}" \
			"Fallback: Native distribution package manager"

		if _install_node_fnm; then
			log_success "Node.js provisioned via Fast Node Manager (fnm)."
			return 0
		fi

		log_warn "fnm strategy failed. Falling back to native package manager..."
		if _install_node_package; then
			log_success "Node.js provisioned via native package manager."
			return 0
		fi

		log_error "All installation strategies failed for Node.js."
		return 1
		;;

	pnpm)
		if ((force == 0)) && command -v pnpm >/dev/null 2>&1; then
			log_info "pnpm is already installed: $(command -v pnpm) ($(pnpm --version 2>&1))"
			return 0
		fi

		log_box "Toolchain: Provisioning pnpm" \
			"Primary Strategy: Official standalone installer (${PNPM_VERSION:-12.4.2})" \
			"Fallback 1: Global npm install" \
			"Fallback 2: Native package manager"

		if _install_pnpm_script; then
			log_success "pnpm provisioned via official standalone script."
			return 0
		fi

		log_warn "Standalone pnpm script failed. Falling back to npm..."
		if _install_pnpm_npm; then
			log_success "pnpm provisioned via npm."
			return 0
		fi

		log_warn "npm fallback failed. Falling back to native package manager..."
		if _install_pnpm_package; then
			log_success "pnpm provisioned via native package manager."
			return 0
		fi

		log_error "All installation strategies failed for pnpm."
		return 1
		;;

	*)
		log_error "Unknown toolchain tool: '$tool'"
		return 1
		;;
	esac
}
