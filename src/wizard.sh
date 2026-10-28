#!/usr/bin/env bash
# ==============================================================================
# src/wizard.sh
# Production High-Performance Idempotent Installation Wizard.
# Guided step-by-step setup with pre-flight diagnostics, parallel task pool,
# smart non-destructive file preservation (smart-append-first), and rollback engine.
# Supports GNU/Linux (all distros), macOS, Windows (WSL/MSYS2), and Android (Termux).
# Part of arturonavax/env development environment.
# ==============================================================================

# Anchoring
while [[ ! -d ./.git/ && ! -d ./files/ && ! -d ./src/ ]]; do
	if [[ "$PWD" == "/" ]]; then
		echo "[Wizard Error]: Repository root not found." >&2
		exit 1
	fi
	if ! cd ..; then
		echo "[Wizard Error]: Failed to change directory." >&2
		exit 1
	fi
done

source ./src/lib/strict.sh
source ./src/lib/colors.sh
source ./src/lib/logging.sh
source ./src/lib/platform.sh
source ./src/lib/packages.sh
source ./src/lib/backup.sh
source ./src/lib/download.sh
source ./src/lib/concurrency.sh
source ./src/lib/smart_append.sh
source ./src/lib/remote_env.sh
source ./src/lib/plan.sh
source ./src/remotes/_versions.sh

# Wizard State
TOTAL_STAGES=8
_STAGE_INDEX=0
_INITIAL_BACKUP=""
ENV_LOG_FILE="${ENV_LOG_FILE:-$HOME/.arturonavax-env-install.log}"

# Component Flags (Default: all enabled)
DO_REQUIREMENTS=1
DO_FONTS=1
DO_TERMINAL=1
DO_EDITOR=1
DO_AI=1
DO_OSCONFIG=1

# Terminal clearing helper (respects non-interactive and dumb terms)
_clear_screen() {
	if [[ -t 1 ]] && [[ "${TERM:-}" != "dumb" ]]; then
		if command -v tput >/dev/null 2>&1 && tput clear 2>/dev/null; then
			return 0
		fi
		printf '\033[2J\033[3J\033[H'
	fi
}

# Graphical Progress Bar
render_progress_bar() {
	local current="$1" total="$2"
	local width=22
	local percent=$((current * 100 / total))
	local filled=$((current * width / total))
	local empty=$((width - filled))

	local bar=""
	for ((b = 0; b < filled; b++)); do bar+="█"; done
	for ((b = 0; b < empty; b++)); do bar+="░"; done

	printf '%s[%s%s%s%s] %3d%%%s' "$DIM" "$BOLD_GREEN" "$bar" "$RESET" "$DIM" "$percent" "$RESET"
}

# Wizard UI Helpers
wizard_banner() {
	_clear_screen
	printf '\n%s%s╭────────────────────────────────────────────────────────────────────╮%s\n' "$BOLD" "$BLUE" "$RESET"
	printf '%s│%s  %-64s  %s%s│%s\n' "$BOLD_BLUE" "$BOLD_WHITE" "arturonavax/env · Master Environment Wizard" "$RESET" "$BOLD_BLUE" "$RESET"
	printf '%s├────────────────────────────────────────────────────────────────────┤%s\n' "$BOLD_BLUE" "$RESET"
	printf '%s│%s  %-64s  %s│%s\n' "$BOLD_BLUE" "$RESET" "High-performance modular development environment." "$BOLD_BLUE" "$RESET"
	printf '%s│%s  %-64s  %s│%s\n' "$BOLD_BLUE" "$RESET" "Multi-platform: Linux (all distros), macOS, Windows, Android." "$BOLD_BLUE" "$RESET"
	printf '%s│%s  %-64s  %s│%s\n' "$BOLD_BLUE" "$DIM" "Ghostty, Herdr, LazyVim, AI Suite, concurrent task downloads." "$BOLD_BLUE" "$RESET"
	printf '%s│%s  %-64s  %s│%s\n' "$BOLD_BLUE" "$DIM" "Smart file preservation (smart-append-first) & instant rollback." "$BOLD_BLUE" "$RESET"
	printf '%s╰────────────────────────────────────────────────────────────────────╯%s\n\n' "$BOLD_BLUE" "$RESET"
}

wizard_stage() {
	_clear_screen
	_STAGE_INDEX=$((_STAGE_INDEX + 1))
	local progress_bar
	progress_bar="$(render_progress_bar "$_STAGE_INDEX" "$TOTAL_STAGES")"

	printf '\n%s%s▸ Stage %s/%s · %s%s  %s\n' \
		"$BOLD" "$BLUE" "$_STAGE_INDEX" "$TOTAL_STAGES" "$1" "$RESET" "$progress_bar"
	printf '%s----------------------------------------------------------------------%s\n' "$DIM" "$RESET"
}

say()  { printf '  %s\n' "$*"; }
step() { printf '  %s•%s %s\n' "$BOLD_BLUE" "$RESET" "$*"; }
note() { printf '  %s%s%s\n' "$DIM" "$*" "$RESET"; }
warn() { printf '  %s⚠ %s%s\n' "$BOLD_YELLOW" "$*" "$RESET"; }

pause() {
	printf '\n  %s%s%s ' "$DIM" "${1:-Press Enter to proceed...}" "$RESET"
	read -r _ || true
}

confirm() {
	local prompt="$1"
	local default="${2:-y}"
	local reply=""

	if [[ "$default" == "y" ]]; then
		printf '  %s? %s [Y/n]: %s' "$BOLD_YELLOW" "$prompt" "$RESET"
	else
		printf '  %s? %s [y/N]: %s' "$BOLD_YELLOW" "$prompt" "$RESET"
	fi

	read -r reply || true
	reply="${reply:-$default}"
	[[ "$reply" =~ ^[Yy] ]]
}

# Failure Recovery Prompt
handle_stage_failure() {
	local stage_name="$1"
	local exit_code="$2"

	printf '\n%s%s  STAGE FAILURE: %s (Exit code: %s)%s\n' "$BOLD_RED" "$SYM_CROSS" "$stage_name" "$exit_code" "$RESET"
	printf '  %sThe installer encountered an error during this stage.%s\n' "$RED" "$RESET"
	printf '  %sDetailed logs are available at: %s%s\n\n' "$DIM" "$ENV_LOG_FILE" "$RESET"

	printf '  %sChoose recovery action:%s\n' "$BOLD_WHITE" "$RESET"
	printf '    %s[r]%s Retry this stage\n' "$BOLD_GREEN" "$RESET"
	printf '    %s[s]%s Skip this stage and continue\n' "$BOLD_YELLOW" "$RESET"
	if [[ -n "$_INITIAL_BACKUP" ]]; then
		printf '    %s[b]%s Rollback to initial snapshot and abort\n' "$BOLD_CYAN" "$RESET"
	fi
	printf '    %s[a]%s Abort installer immediately (default)\n\n' "$BOLD_RED" "$RESET"

	local choice=""
	read -r -p "  Enter choice [r/s/b/a]: " choice
	choice="${choice:-a}"

	case "$choice" in
	r | R)
		return 10 # Code for retry
		;;
	s | S)
		log_warn "Stage '${stage_name}' skipped by user choice."
		return 0 # Code for skip
		;;
	b | B)
		if [[ -n "$_INITIAL_BACKUP" ]]; then
			log_step "Rolling back to ${_INITIAL_BACKUP}..."
			backup_restore "$_INITIAL_BACKUP"
		fi
		log_fatal "Installation aborted and rolled back by user." 1
		;;
	*)
		log_fatal "Installation aborted by user due to error in stage '${stage_name}'." "$exit_code"
		;;
	esac
}

# ==============================================================================
# Wizard Execution Stages
# ==============================================================================

# STAGE 1: Pre-flight Verification & Platform Discovery
stage_preflight() {
	wizard_stage "Pre-flight Diagnostics & Multi-Platform Discovery"

	say "Verifying system compatibility, hardware resources, and privileges..."
	echo

	local os distro arch free_disk ram cores
	os="$(detect_os)"
	distro="$(detect_distro)"
	arch="$(detect_arch)"
	free_disk="$(get_free_disk_space_mb "$HOME")"
	ram="$(get_total_ram_mb)"
	cores="$(get_cpu_cores)"

	step "Operating System:   ${BOLD_WHITE}${os} (${distro})${RESET}"
	step "Architecture:       ${BOLD_WHITE}${arch}${RESET}"
	step "CPU Cores:          ${BOLD_WHITE}${cores} cores${RESET} (Parallel pool limit: ${MAX_PARALLEL_JOBS})"
	step "Total RAM:          ${BOLD_WHITE}${ram} MB${RESET}"
	step "Free Disk Space:    ${BOLD_WHITE}${free_disk} MB${RESET}"

	# Validate Disk Space (Recommend at least 3000 MB)
	if ((free_disk < 3000)); then
		warn "Available disk space is low (${free_disk} MB < 3000 MB recommended)."
		if ! confirm "Continue despite low disk space?" "n"; then
			log_fatal "Installation cancelled: Insufficient disk space." 1
		fi
	fi

	# Parallel Network Connectivity Probes
	echo
	step "Verifying remote connectivity in parallel..."
	run_parallel_task_pool "Network Verification" \
		"GitHub Connection:::curl --proto '=https' --tlsv1.2 -fsSL --connect-timeout 4 'https://raw.githubusercontent.com' >/dev/null" \
		"CDN Endpoint:::curl --proto '=https' --tlsv1.2 -fsSL --connect-timeout 4 '${REMOTE_PRIMARY_URL}' >/dev/null 2>&1 || true"

	# Privileges: check Android Termux sandbox vs standard sudo
	echo
	if [[ "$os" == "android" ]] || [[ "$distro" == "termux" ]]; then
		step "Environment: Android Termux sandbox."
		say "  ${GREEN}${SYM_CHECK} Sudo elevation skipped (Termux single-user architecture).${RESET}"
	elif has_sudo_capability; then
		step "Acquiring administrator privileges (sudo)..."
		ensure_sudo
		say "  ${GREEN}${SYM_CHECK} Sudo credentials validated and keep-alive active.${RESET}"
	else
		warn "No sudo privileges detected. Package installations may require manual intervention."
	fi

	pause
}

# STAGE 2: Safety Snapshot & Smart Preservation
stage_backup() {
	wizard_stage "Configuration Snapshot & Safety Guard"

	say "Creating an automatic, restorable snapshot of all existing configuration files..."
	say "Destination directory: ${BOLD_WHITE}~/.arturonavax-env-backups/${RESET}"
	echo

	_INITIAL_BACKUP="$(backup_create)"
	echo
	step "Snapshot created: ${BOLD_WHITE}$(basename "$_INITIAL_BACKUP")${RESET}"
	note "Smart Preservation Policy (smart-append-first):"
	note "  • Your custom shell aliases, PATHs and functions in ~/.zshrc or ~/.bashrc are preserved."
	note "  • Environment loaders are isolated between dedicated marker blocks."
	note "  • To rollback at any time, run: ./utils/restore_backup.sh $(basename "$_INITIAL_BACKUP")"

	pause
}

# STAGE 3: Component Selection
stage_select_components() {
	wizard_stage "Modular Component Selection"

	say "Select the subsystems you wish to install in this environment."
	say "Default: All components enabled for a complete, optimal setup."
	echo

	if confirm "Install complete environment with ALL components?" "y"; then
		DO_REQUIREMENTS=1
		DO_FONTS=1
		DO_TERMINAL=1
		DO_EDITOR=1
		DO_AI=1
		DO_OSCONFIG=1
	else
		echo
		say "Configure components individually:"
		confirm "1. Base Languages & Compilers (GCC/Clang, Rust, Go, Node, Python)?" "y" && DO_REQUIREMENTS=1 || DO_REQUIREMENTS=0
		confirm "2. Patched Nerd Fonts (JetBrains Mono, FiraCode)?" "y" && DO_FONTS=1 || DO_FONTS=0
		confirm "3. Modern Terminal Stack (Ghostty/Zsh/Starship/Atuin & CLI tools)?" "y" && DO_TERMINAL=1 || DO_TERMINAL=0
		confirm "4. Code Editor Suite (Neovim v0.12+, LazyVim, Mason LSPs, Treesitter)?" "y" && DO_EDITOR=1 || DO_EDITOR=0
		confirm "5. AI Tooling & Ecosystem (Antigravity CLI / agy, RTK, MCP servers)?" "y" && DO_AI=1 || DO_AI=0
		confirm "6. OS Integration & Window Switcher (rcmd, shortcuts, desktop settings)?" "y" && DO_OSCONFIG=1 || DO_OSCONFIG=0
	fi

	echo
	step "Selected components summary:"
	if [[ "$DO_REQUIREMENTS" == "1" ]]; then say "  ${GREEN}${SYM_CHECK}${RESET} Base Compilers & Languages"; else say "  ${GRAY}${SYM_CROSS} Base Compilers & Languages (skipped)${RESET}"; fi
	if [[ "$DO_FONTS" == "1" ]]; then say "  ${GREEN}${SYM_CHECK}${RESET} Patched Nerd Fonts"; else say "  ${GRAY}${SYM_CROSS} Patched Nerd Fonts (skipped)${RESET}"; fi
	if [[ "$DO_TERMINAL" == "1" ]]; then say "  ${GREEN}${SYM_CHECK}${RESET} Modern Terminal Stack"; else say "  ${GRAY}${SYM_CROSS} Modern Terminal Stack (skipped)${RESET}"; fi
	if [[ "$DO_EDITOR" == "1" ]]; then say "  ${GREEN}${SYM_CHECK}${RESET} Code Editor Suite"; else say "  ${GRAY}${SYM_CROSS} Code Editor Suite (skipped)${RESET}"; fi
	if [[ "$DO_AI" == "1" ]]; then say "  ${GREEN}${SYM_CHECK}${RESET} AI Tooling & Ecosystem"; else say "  ${GRAY}${SYM_CROSS} AI Tooling & Ecosystem (skipped)${RESET}"; fi
	if [[ "$DO_OSCONFIG" == "1" ]]; then say "  ${GREEN}${SYM_CHECK}${RESET} OS Integration & Shortcuts"; else say "  ${GRAY}${SYM_CROSS} OS Integration & Shortcuts (skipped)${RESET}"; fi

	echo
	if confirm "Preview detailed installation plan & touched files before proceeding?" "n"; then
		local selected=()
		[[ "$DO_REQUIREMENTS" == "1" ]] && selected+=("requirements")
		[[ "$DO_FONTS" == "1" ]] && selected+=("fonts")
		[[ "$DO_TERMINAL" == "1" ]] && selected+=("terminal")
		[[ "$DO_EDITOR" == "1" ]] && selected+=("editor")
		[[ "$DO_AI" == "1" ]] && selected+=("ai")
		[[ "$DO_OSCONFIG" == "1" ]] && selected+=("osconfig")
		show_install_plan "${selected[@]}"
	fi

	pause
}

# STAGE 4: Base Compilers & Requirements
stage_install_requirements() {
	if [[ "$DO_REQUIREMENTS" != "1" ]]; then
		wizard_stage "Base Compilers & Languages"
		say "Skipped by user selection."
		sleep 1
		return 0
	fi

	while true; do
		wizard_stage "Installing Base Compilers, Runtimes & Dependencies"
		say "Provisioning native build tools, SSL/SQLite headers, Rustup, fnm, pnpm, Go, and Python..."
		echo

		local status=0
		bash ./src/_install_requirements.sh || status=$?

		if ((status == 0)); then
			log_success "Base requirements and compilers installed successfully."
			pause
			break
		else
			handle_stage_failure "Base Requirements" "$status"
			local recovery=$?
			if ((recovery == 0)); then break; fi
		fi
	done
}

# STAGE 5: Terminal & CLI Tools (with Parallel Downloads)
stage_install_terminal() {
	if [[ "$DO_TERMINAL" != "1" ]]; then
		wizard_stage "Terminal & CLI Tools Stack"
		say "Skipped by user selection."
		sleep 1
		return 0
	fi

	while true; do
		wizard_stage "Installing High-Performance Terminal & CLI Stack"
		say "Deploying Ghostty, Herdr multiplexer, lean Zsh, Starship, Atuin, and modern CLI tools..."
		echo

		# Parallel Pre-cloning of Zsh Plugins
		step "Synchronizing Zsh plugins in parallel..."
		mkdir -p "$HOME/.zsh"
		run_parallel_task_pool "Zsh Plugins Sync" \
			"fzf-tab:::if [[ -d '$HOME/.zsh/fzf-tab/.git' ]]; then git -C '$HOME/.zsh/fzf-tab' pull --rebase --quiet; else rm -rf '$HOME/.zsh/fzf-tab'; git clone --quiet https://github.com/Aloxaf/fzf-tab '$HOME/.zsh/fzf-tab'; fi" \
			"zsh-autosuggestions:::if [[ -d '$HOME/.zsh/zsh-autosuggestions/.git' ]]; then git -C '$HOME/.zsh/zsh-autosuggestions' pull --rebase --quiet; else rm -rf '$HOME/.zsh/zsh-autosuggestions'; git clone --quiet https://github.com/zsh-users/zsh-autosuggestions '$HOME/.zsh/zsh-autosuggestions'; fi" \
			"fast-syntax-highlighting:::if [[ -d '$HOME/.zsh/fast-syntax-highlighting/.git' ]]; then git -C '$HOME/.zsh/fast-syntax-highlighting' pull --rebase --quiet; else rm -rf '$HOME/.zsh/fast-syntax-highlighting'; git clone --quiet https://github.com/zdharma-continuum/fast-syntax-highlighting '$HOME/.zsh/fast-syntax-highlighting'; fi" \
			"zsh-completions:::if [[ -d '$HOME/.zsh/zsh-completions/.git' ]]; then git -C '$HOME/.zsh/zsh-completions' pull --rebase --quiet; else rm -rf '$HOME/.zsh/zsh-completions'; git clone --quiet https://github.com/zsh-users/zsh-completions.git '$HOME/.zsh/zsh-completions'; fi"

		echo
		local status=0
		bash ./src/install_terminal.sh || status=$?

		if ((status == 0)); then
			log_success "Terminal stack deployed successfully."
			pause
			break
		else
			handle_stage_failure "Terminal Stack" "$status"
			local recovery=$?
			if ((recovery == 0)); then break; fi
		fi
	done
}

# STAGE 6: Editor Suite
stage_install_editor() {
	if [[ "$DO_EDITOR" != "1" ]]; then
		wizard_stage "Code Editor Suite"
		say "Skipped by user selection."
		sleep 1
		return 0
	fi

	while true; do
		wizard_stage "Installing Code Editor Suite (Neovim + LazyVim)"
		say "Deploying Neovim (${NVIM_VERSION}), LazyVim, Mason LSPs, Treesitter parsers, and Git integration..."
		echo

		local status=0
		bash ./src/install_editor.sh || status=$?

		if ((status == 0)); then
			log_success "Editor suite installed successfully."
			pause
			break
		else
			handle_stage_failure "Editor Suite" "$status"
			local recovery=$?
			if ((recovery == 0)); then break; fi
		fi
	done
}

# STAGE 7: AI Tooling
stage_install_ai() {
	if [[ "$DO_AI" != "1" ]]; then
		wizard_stage "AI Tooling & Ecosystem"
		say "Skipped by user selection."
		sleep 1
		return 0
	fi

	while true; do
		wizard_stage "Installing AI Tooling, MCP Ecosystem & RTK"
		say "Deploying Antigravity CLI (agy), Model Context Protocol servers, RTK token killer, and harnesses..."
		echo

		local status=0
		bash ./src/install_ai.sh || status=$?

		if ((status == 0)); then
			log_success "AI tooling environment configured successfully."
			pause
			break
		else
			handle_stage_failure "AI Tooling" "$status"
			local recovery=$?
			if ((recovery == 0)); then break; fi
		fi
	done
}

# STAGE 8: OS Configuration, Smart File Management & Verification
stage_finalize() {
	wizard_stage "OS Configuration, Smart File Management & Verification"

	# Install Fonts
	if [[ "$DO_FONTS" == "1" ]]; then
		step "Installing patched Nerd Fonts..."
		bash ./src/remotes/install_fonts.sh || true
	fi

	# System OS Config
	if [[ "$DO_OSCONFIG" == "1" ]]; then
		step "Applying operating system configurations and window switcher..."
		local os
		os="$(detect_os)"
		if [[ "$os" == "linux" ]]; then
			bash ./src/remotes/linux_osconfig.sh || true
		elif [[ "$os" == "darwin" ]]; then
			bash ./src/remotes/macos_osconfig.sh || true
		fi
	fi

	# Smart Non-Destructive Shell Sync (smart-append-first)
	step "Applying smart-append-first shell integration..."
	local shell_entry="[[ -f ~/.tools.sh ]] && source ~/.tools.sh\n[[ -f ~/.base.zsh ]] && source ~/.base.zsh"
	if [[ -f "$HOME/.zshrc" ]]; then
		smart_append_block "$HOME/.zshrc" "env-loader" "$shell_entry"
	fi
	if [[ -f "$HOME/.bashrc" ]]; then
		smart_append_block "$HOME/.bashrc" "env-loader" "[[ -f ~/.tools.sh ]] && source ~/.tools.sh"
	fi

	# Synchronize Configurations
	step "Propagating dotfiles into active environment..."
	bash ./utils/sync_config.sh all || true

	# Health Check & Verification
	echo
	step "Running end-to-end binary and tool verification:"
	local -a binaries=(git zsh starship ghostty herdr atuin difft delta tldr usql fx nvim rtk agy)
	local missing_count=0

	for bin in "${binaries[@]}"; do
		if command -v "$bin" >/dev/null 2>&1; then
			printf '  %s%s%s %-12s %s\n' "$GREEN" "$SYM_CHECK" "$RESET" "$bin" "${DIM}($(command -v "$bin"))${RESET}"
		else
			printf '  %s%s%s %-12s %s\n' "$YELLOW" "$SYM_WARN" "$RESET" "$bin" "${YELLOW}(not in current PATH)${RESET}"
			missing_count=$((missing_count + 1))
		fi
	done

	echo
	if ((missing_count == 0)); then
		log_box "Installation Completed Successfully!" \
			"All requested subsystems have been cleanly installed and verified." \
			"Your development environment is ready to use." \
			"Restart your terminal or run: source ~/.zshrc"
	else
		log_box "Installation Finished with Warnings" \
			"Completed with ${missing_count} binaries pending or not in current PATH." \
			"Ensure ~/.local/bin is in your PATH, or run: source ~/.tools.sh"
	fi
}

# ==============================================================================
# Main Entry Point
# ==============================================================================
main() {
	wizard_banner
	pause "Ready to begin setup? Press Enter to start..."

	stage_preflight
	stage_backup
	stage_select_components
	stage_install_requirements
	stage_install_terminal
	stage_install_editor
	stage_install_ai
	stage_finalize
}

main "$@"
