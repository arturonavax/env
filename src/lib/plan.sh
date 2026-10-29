#!/usr/bin/env bash
# ==============================================================================
# src/lib/plan.sh
# Production Execution Plan & Manifest Inspector for arturonavax/env.
# Provides detailed inspection of packages, binaries, runtimes, and filesystem
# modifications before installation or uninstallation (dry-run preview).
# Part of arturonavax/env development environment.
# ==============================================================================

# Prevent double sourcing
[[ -n "${_PLAN_SH_LOADED:-}" ]] && return 0
_PLAN_SH_LOADED=1

_LIB_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=/dev/null
source "${_LIB_DIR}/colors.sh" 2>/dev/null || true
# shellcheck source=/dev/null
source "${_LIB_DIR}/ui_box.sh" 2>/dev/null || true
# shellcheck source=/dev/null
source "${_LIB_DIR}/logging.sh" 2>/dev/null || true
# shellcheck source=/dev/null
source "${_LIB_DIR}/platform.sh" 2>/dev/null || true
# shellcheck source=/dev/null
source "${_LIB_DIR}/packages.sh" 2>/dev/null || true

# Format section header
_plan_header() {
	local title="$1"
	local width
	width="$(ui_box_width 70)"
	printf '\n%s%s%s %s%s%s\n' "$BOLD_BLUE" "$SYM_ARROW" "$RESET" "$BOLD_WHITE" "$title" "$RESET"
	ui_box_separator "$width" "─" "$DIM"
}

_plan_item() {
	local tag="$1"
	local name="$2"
	local detail="${3:-}"
	printf '  %s%-16s%s %-32s %s%s%s\n' "$BOLD_CYAN" "[$tag]" "$RESET" "$name" "$DIM" "$detail" "$RESET"
}

# ------------------------------------------------------------------------------
# Installation Plan
# ------------------------------------------------------------------------------
show_install_plan() {
	local target_components=("$@")
	local os distro arch pkg_mgr
	os="$(detect_os)"
	distro="$(detect_distro)"
	arch="$(detect_arch)"
	pkg_mgr="$(detect_pkg_mgr)"

	local do_all=0
	local do_req=0
	local do_fonts=0
	local do_term=0
	local do_edit=0
	local do_ai=0
	local do_os=0

	if [[ ${#target_components[@]} -eq 0 ]]; then
		do_all=1
	else
		for comp in "${target_components[@]}"; do
			case "$comp" in
			all | a | --all) do_all=1 ;;
			requirements | r | req) do_req=1 ;;
			fonts | f) do_fonts=1 ;;
			terminal | t | term) do_term=1 ;;
			editor | e | edit) do_edit=1 ;;
			ai) do_ai=1 ;;
			osconfig | o | os) do_os=1 ;;
			esac
		done
	fi

	if ((do_all == 1)); then
		do_req=1
		do_fonts=1
		do_term=1
		do_edit=1
		do_ai=1
		do_os=1
	fi

	local width
	width="$(ui_box_width 70)"

	echo
	ui_box_top "$width" "rounded" "$BOLD_BLUE"
	ui_box_line "arturonavax/env · Installation Plan & Manifest Preview" "$width" "rounded" "$BOLD_BLUE" "$BOLD_WHITE" "left"
	ui_box_mid "$width" "rounded" "$BOLD_BLUE"
	ui_box_line "Platform:     ${os} (${distro}) - ${arch}" "$width" "rounded" "$BOLD_BLUE" "$RESET" "left"
	ui_box_line "Package Mgr:  $pkg_mgr" "$width" "rounded" "$BOLD_BLUE" "$RESET" "left"
	ui_box_line "Execution:    DRY RUN / PREVIEW (No files modified)" "$width" "rounded" "$BOLD_BLUE" "$RESET" "left"
	ui_box_bottom "$width" "rounded" "$BOLD_BLUE"

	# 1. Pre-installation Safeguards
	_plan_header "1. Pre-Installation Safeguards & Backups"
	_plan_item "BACKUP" "${HOME}/.arturonavax-env-backups/" "Automatic snapshot of all config files before any changes"
	_plan_item "MANIFEST" "manifest.txt" "Checksummed inventory generated inside backup folder"
	_plan_item "ROTATION" "50 MB Quota" "Oldest snapshots automatically rotated when quota exceeded"
	_plan_item "ROLLBACK" "utils/restore_backup.sh" "Single-command instant restoration to any snapshot"

	# 2. Package Manager Translations
	if ((do_req == 1 || do_term == 1 || do_edit == 1)); then
		_plan_header "2. Packages to be Installed via Native Manager ($pkg_mgr)"
		local base_pkgs=(
			"build-essential" "ssl-dev" "zlib-dev" "readline-dev" "sqlite-dev"
			"curl" "wget" "git" "unzip" "zstd" "make" "gcc" "fontconfig"
		)
		if [[ "$os" == "linux" ]]; then
			base_pkgs+=("wmctrl" "xdotool" "x11-utils" "keyd")
		fi
		if ((do_term == 1)); then
			base_pkgs+=("zsh" "fzf")
		fi
		if ((do_edit == 1)); then
			base_pkgs+=("neovim")
		fi

		for pkg in "${base_pkgs[@]}"; do
			local translated
			translated="$(pkg_translate "$pkg")"
			if [[ -n "$translated" ]]; then
				_plan_item "PKG ($pkg_mgr)" "$translated" "(canonical: $pkg)"
			else
				_plan_item "PKG (SKIP)" "$pkg" "Not applicable or unavailable on $os"
			fi
		done
	fi

	# 3. Compilers & Language Runtimes
	if ((do_req == 1)); then
		_plan_header "3. Language Runtimes & Toolchain Managers"
		_plan_item "RUNTIME" "Rustup / Cargo" "Latest stable Rust compiler & cargo toolchain"
		_plan_item "RUNTIME" "Fast Node Manager (fnm)" "Node.js version manager in ${HOME}/.local/share/fnm"
		_plan_item "RUNTIME" "Node.js (LTS/Latest)" "Configured via fnm with corepack isolation"
		_plan_item "RUNTIME" "pnpm" "Fast package manager in ${HOME}/.local/share/pnpm"
		_plan_item "RUNTIME" "GVM / Golang" "GVM (Go Version Manager) primary, with standalone Go SDK fallback"
		_plan_item "RUNTIME" "Python 3 / pyenv" "Configured with virtualenv & pip support"
	fi

	# 4. Standalone Binaries & Multiplexers
	if ((do_term == 1 || do_edit == 1 || do_ai == 1 || do_os == 1)); then
		_plan_header "4. Standalone Binaries & High-Performance CLI Tools"
		if ((do_term == 1)); then
			_plan_item "TERMINAL" "Ghostty" "Fast native GPU terminal (/usr/bin/ghostty or Applications)"
			_plan_item "MULTIPLEXER" "Herdr" "Native agent & terminal multiplexer (${HOME}/.local/bin/herdr)"
			_plan_item "HISTORY" "Atuin" "SQLite shell history engine (${HOME}/.local/bin/atuin)"
			_plan_item "PROMPT" "Starship" "Cross-shell ultra-lean prompt binary (/usr/local/bin/starship)"
			_plan_item "CLI TOOL" "difftastic (difft)" "AST Tree-sitter structural diffing (${HOME}/.local/bin/difft)"
			_plan_item "CLI TOOL" "git-delta (delta)" "Syntax-highlighting git pager (${HOME}/.local/bin/delta)"
			_plan_item "CLI TOOL" "tealdeer (tldr)" "Fast native Rust tldr client (${HOME}/.local/bin/tldr)"
			_plan_item "CLI TOOL" "usql" "Universal database CLI client (${HOME}/.local/bin/usql)"
			_plan_item "CLI TOOL" "fx" "Interactive terminal JSON viewer (${HOME}/.local/bin/fx)"
		fi
		if ((do_edit == 1)); then
			_plan_item "EDITOR" "Neovim (v0.12+)" "Modern modal editor binary (/usr/local/bin/nvim)"
			_plan_item "FORMATTER" "Stylua" "Opinionated Lua code formatter (${HOME}/.local/bin/stylua)"
			_plan_item "GIT TOOL" "Lazygit" "Terminal UI for git commands (${HOME}/.local/bin/lazygit)"
		fi
		if ((do_ai == 1)); then
			_plan_item "AI CLI" "Antigravity CLI (agy)" "Google Gemini CLI agent harness (${HOME}/.local/bin/agy)"
			_plan_item "AI PROXY" "RTK" "LLM token output optimizer & proxy (${HOME}/.local/bin/rtk)"
		fi
		if ((do_os == 1)) && [[ "$os" == "linux" ]]; then
			_plan_item "APP SWITCHER" "rcmd" "macOS-style Right-Alt app switcher (${HOME}/.local/bin/rcmd)"
		fi
	fi

	# 5. Files Interacted With
	_plan_header "5. File System Interactions (Created, Modified, Preserved)"

	# Smart Append
	_plan_item "SMART-APPEND" "${HOME}/.zshrc" "Preserves user content; injects # >>> arturonavax/env >>> loader"
	_plan_item "SMART-APPEND" "${HOME}/.bashrc" "Preserves user content; injects fallback loader marker block"

	# Managed Repo Dotfiles (always kept updated from repository)
	_plan_item "CONFIG" "${HOME}/.base.zsh" "Core shell configuration (maintained & updated by repository)"
	_plan_item "CONFIG" "${HOME}/.tools.sh" "Development toolchain PATHs (maintained & updated by repository)"

	# Preserved Files
	if [[ "$os" == "linux" ]]; then
		_plan_item "PRESERVED" "${HOME}/.config/rcmd/rcmd.conf" "User custom window switcher app bindings (kept intact)"
	fi

	# Deployed Configurations
	if ((do_term == 1)); then
		_plan_item "CONFIG" "${HOME}/.config/ghostty/config" "Terminal colors, fonts, keybindings, herdr launcher"
		_plan_item "CONFIG" "${HOME}/.config/herdr/config.toml" "Agent multiplexer workspaces, status bar, and hooks"
		_plan_item "CONFIG" "${HOME}/.config/atuin/config.toml" "SQLite offline history mode, directory scoping"
		_plan_item "CONFIG" "${HOME}/.config/starship.toml" "Lean 2-line prompt with agent workspace indicators"
		_plan_item "ZSH PLUGIN" "${HOME}/.zsh/plugins/zsh-autosuggestions" "Fish-like auto-suggestions for Zsh"
		_plan_item "ZSH PLUGIN" "${HOME}/.zsh/plugins/fast-syntax-highlighting" "Optimized syntax highlighting for Zsh"
		_plan_item "ZSH PLUGIN" "${HOME}/.zsh/plugins/fzf-tab" "Interactive FZF tab completion menu"
	fi

	if ((do_edit == 1)); then
		_plan_item "CONFIG" "${HOME}/.config/nvim/" "LazyVim modular architecture, Mason LSPs, exclusions"
		_plan_item "CONFIG" "${HOME}/.config/Code/User/settings.json" "VSCode settings & keybindings"
		_plan_item "CONFIG" "${HOME}/.config/Cursor/User/settings.json" "Cursor IDE settings & keybindings"
	fi

	if ((do_ai == 1)); then
		_plan_item "CONFIG" "${HOME}/.config/rtk/config.toml" "RTK compression rules and tool command wrappers"
		_plan_item "CONFIG" "${HOME}/.gemini/antigravity-cli/config.json" "Antigravity CLI configuration"
	fi

	if ((do_fonts == 1)); then
		if [[ "$os" == "darwin" ]]; then
			_plan_item "FONT" "${HOME}/Library/Fonts/JetBrainsMono*.ttf" "JetBrains Mono Nerd Font"
			_plan_item "FONT" "${HOME}/Library/Fonts/FiraCode*.ttf" "FiraCode Nerd Font"
		else
			_plan_item "FONT" "${HOME}/.local/share/fonts/JetBrainsMono*.ttf" "JetBrains Mono Nerd Font"
			_plan_item "FONT" "${HOME}/.local/share/fonts/FiraCode*.ttf" "FiraCode Nerd Font"
		fi
	fi

	printf '\n%s----------------------------------------------------------------------%s\n' "$DIM" "$RESET"
	printf '  %sTip:%s To execute this installation, run: %s./install.sh%s\n\n' \
		"$BOLD_WHITE" "$RESET" "$BOLD_GREEN" "$RESET"
}

# ------------------------------------------------------------------------------
# Uninstallation Plan
# ------------------------------------------------------------------------------
show_uninstall_plan() {
	local os distro arch pkg_mgr
	os="$(detect_os)"
	distro="$(detect_distro)"
	arch="$(detect_arch)"
	pkg_mgr="$(detect_pkg_mgr)"

	local width
	width="$(ui_box_width 70)"

	echo
	ui_box_top "$width" "rounded" "$BOLD_RED"
	ui_box_line "arturonavax/env · Uninstallation Plan & Teardown Preview" "$width" "rounded" "$BOLD_RED" "$BOLD_WHITE" "left"
	ui_box_mid "$width" "rounded" "$BOLD_RED"
	ui_box_line "Platform:     ${os} (${distro}) - ${arch}" "$width" "rounded" "$BOLD_RED" "$RESET" "left"
	ui_box_line "Package Mgr:  $pkg_mgr" "$width" "rounded" "$BOLD_RED" "$RESET" "left"
	ui_box_line "Execution:    DRY RUN / PREVIEW (No files modified or removed)" "$width" "rounded" "$BOLD_RED" "$RESET" "left"
	ui_box_bottom "$width" "rounded" "$BOLD_RED"

	# 1. Packages Removed
	_plan_header "1. Packages Removed via Native Manager ($pkg_mgr)"
	_plan_item "PKG PURGE" "neovim" "Removed via $pkg_mgr"
	_plan_item "PKG PURGE" "fzf" "Removed via $pkg_mgr"
	if command -v snap >/dev/null 2>&1; then
		_plan_item "SNAP PURGE" "ghostty, tmux, nvim" "Cleaned from snap if present"
	fi
	if [[ "$os" == "darwin" ]]; then
		_plan_item "BREW PURGE" "ghostty (cask), tmux, neovim, fzf" "Removed via Homebrew"
	fi

	# 2. System and Local Binaries Purged
	_plan_header "2. System & Local Binaries Purged"
	_plan_item "PURGE" "/usr/bin/ghostty, /usr/local/bin/ghostty" "Ghostty terminal binary"
	_plan_item "PURGE" "/usr/local/bin/nvim, ${HOME}/.local/bin/nvim" "Neovim editor binary"
	_plan_item "PURGE" "/opt/nvim-linux-*, /opt/zig-*" "Extracted compiler/editor release directories"
	_plan_item "PURGE" "/usr/local/bin/herdr, ${HOME}/.local/bin/herdr" "Herdr multiplexer binary"
	_plan_item "PURGE" "/usr/local/bin/atuin, ${HOME}/.local/bin/atuin" "Atuin shell history binary"
	_plan_item "PURGE" "/usr/local/bin/rtk, ${HOME}/.local/bin/rtk" "RTK token proxy binary"
	_plan_item "PURGE" "/usr/local/bin/zig" "Zig compiler binary"
	if [[ "$os" == "darwin" ]]; then
		_plan_item "PURGE" "/Applications/Ghostty.app" "Ghostty macOS application bundle"
	fi

	# 3. Desktop Entries, Icons & Systemd Units
	_plan_header "3. Desktop Entries, Icons & Systemd Services"
	_plan_item "PURGE" "/usr/share/applications/*ghostty*.desktop" "Ghostty application launcher entry"
	_plan_item "PURGE" "/usr/share/applications/nvim.desktop" "Neovim application launcher entry"
	_plan_item "PURGE" "${HOME}/.local/share/applications/nvim.desktop" "User Neovim application launcher entry"
	_plan_item "PURGE" "${HOME}/.config/systemd/user/*ghostty*" "Ghostty user systemd units"
	_plan_item "PURGE" "/usr/share/icons/hicolor/*/apps/*ghostty*" "Ghostty system icons"
	_plan_item "PURGE" "/usr/share/terminfo/g/ghostty*" "Ghostty terminfo databases"

	# 4. Configuration Directories & Caches Purged
	_plan_header "4. Configuration Directories, Local States & Caches Purged"
	_plan_item "PURGE" "${HOME}/.config/ghostty" "Ghostty terminal configuration"
	_plan_item "PURGE" "${HOME}/.config/herdr" "Herdr multiplexer configuration"
	_plan_item "PURGE" "${HOME}/.config/atuin, ${HOME}/.local/share/atuin" "Atuin history configuration & database"
	_plan_item "PURGE" "${HOME}/.config/nvim" "LazyVim editor configuration"
	_plan_item "PURGE" "${HOME}/.local/share/nvim" "Neovim Mason packages, plugins, and Treesitter parsers"
	_plan_item "PURGE" "${HOME}/.local/state/nvim" "Neovim log files and undo histories"
	_plan_item "PURGE" "${HOME}/.cache/nvim" "Neovim temporary bytecode and caches"
	_plan_item "PURGE" "${HOME}/.cache/tmux, ${HOME}/.tmux" "Legacy tmux plugins and runtime caches"

	# 5. Preserved Files (Safety)
	_plan_header "5. Files & Assets Preserved (NEVER DELETED)"
	_plan_item "PRESERVED" "${HOME}/.arturonavax-env-backups/" "All previous configuration snapshots and manifests"
	_plan_item "PRESERVED" "${HOME}/.zshrc, ${HOME}/.bashrc" "Custom user shell aliases, PATHs, functions (intact)"
	_plan_item "PRESERVED" "${HOME}/go/, ${HOME}/.cargo/" "Installed user packages, Go modules, and Cargo crates"
	_plan_item "PRESERVED" "${HOME}/.gvm/" "Go versions and pkgsets managed by GVM"
	_plan_item "PRESERVED" "${HOME}/.local/share/fnm/" "Node.js runtimes managed by fnm"

	printf '\n%s----------------------------------------------------------------------%s\n' "$DIM" "$RESET"
	printf '  %sWarning:%s To execute this uninstallation, run: %s./uninstall.sh%s\n\n' \
		"$BOLD_RED" "$RESET" "$BOLD_WHITE" "$RESET"
}
