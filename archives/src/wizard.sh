#!/usr/bin/env bash
# ==============================================================================
# src/wizard.sh
# Production High-Performance Idempotent Installation Wizard.
# Guided step-by-step setup with pre-flight diagnostics, parallel task pool,
# smart non-destructive file preservation (smart-append-first), and rollback engine.
# Supports both Upfront Interactive Interview and Autonomous Non-Interactive modes.
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
source ./src/lib/toolchain.sh
source ./src/lib/wizard_engine.sh
source ./src/lib/transaction.sh
source ./src/remotes/_versions.sh

# Wizard Global State
ENV_LOG_FILE="${ENV_LOG_FILE:-$HOME/.arturonavax-env-install.log}"
DEST_BIN_DIR="${DEST_BIN_DIR:-$HOME/.local/bin}"
BACKUP_STORAGE_DIR="${BACKUP_STORAGE_DIR:-$HOME/.arturonavax-env-backups}"
SKIP_BACKUP=0
ROLLBACK_ON_ERROR=0
GO_MANAGER_STRATEGY="${GO_MANAGER_STRATEGY:-gvm}"
NODE_MANAGER="${NODE_MANAGER:-pnpm}"
DO_RCMD=1
DO_MACOS_TWEAKS=1
IS_INTERACTIVE=1
SHOW_PLAN_ONLY=0

# Component Flags (Default: all enabled)
DO_REQUIREMENTS=1
DO_FONTS=1
DO_TERMINAL=1
DO_EDITOR=1
DO_AI=1
DO_OSCONFIG=1

usage() {
	cat <<EOF
Usage: ./src/wizard.sh [OPTIONS]

Execution Modes:
  -i, --interactive             Run interactive wizard with upfront parameter interview (default in TTY).
  -y, --yes, --non-interactive,
  -n, --unattended, --batch     Run fully unattended non-interactive installation (zero prompts).

Component Selection:
  -c, --components=<list>       Comma-separated list of components to install:
                                requirements, fonts, terminal, editor, ai, osconfig, all, none
                                (Default in non-interactive mode: all)

Configuration & Paths:
  --dest-bin=<path>             Target directory for user binaries (default: ~/.local/bin).
  --backup-dir=<path>           Target directory for backup snapshots (default: ~/.arturonavax-env-backups).
  --skip-backup                 Bypass configuration snapshot creation.
  --go-manager=<strategy>       Golang manager strategy: gvm (default), standalone, package.
  --node-manager=<manager>      Node.js package manager: pnpm (default), npm.
  --rollback-on-error           Automatically trigger transactional rollback on any stage failure.

Planning & Diagnostics:
  -p, --plan, --dry-run,
  preview                       Display detailed execution plan and touched files without changes.
  -h, --help                    Show this help message.

Examples:
  ./src/wizard.sh                                     # Guided interactive setup
  ./src/wizard.sh -y                                  # Complete non-interactive install
  ./src/wizard.sh -y --components=terminal,editor     # Install only terminal and editor suites
  ./src/wizard.sh -y --go-manager=standalone          # Install with standalone Go strategy
  ./src/wizard.sh --plan                              # Preview installation manifest and touched files
EOF
}

# Parse comma-separated component string
parse_components() {
	local comp_str="$1"
	DO_REQUIREMENTS=0
	DO_FONTS=0
	DO_TERMINAL=0
	DO_EDITOR=0
	DO_AI=0
	DO_OSCONFIG=0

	local item
	local orig_ifs="$IFS"
	IFS=','
	read -r -a comp_arr <<<"$comp_str"
	IFS="$orig_ifs"

	for item in "${comp_arr[@]}"; do
		item="$(echo "$item" | tr -d '[:space:]')"
		case "$item" in
		all | a)
			DO_REQUIREMENTS=1
			DO_FONTS=1
			DO_TERMINAL=1
			DO_EDITOR=1
			DO_AI=1
			DO_OSCONFIG=1
			;;
		requirements | r) DO_REQUIREMENTS=1 ;;
		fonts | f) DO_FONTS=1 ;;
		terminal | t) DO_TERMINAL=1 ;;
		editor | e) DO_EDITOR=1 ;;
		ai) DO_AI=1 ;;
		osconfig | o) DO_OSCONFIG=1 ;;
		none) ;;
		*)
			log_error "Unrecognized component: '$item'"
			usage
			exit 1
			;;
		esac
	done
}

# Parse CLI options upfront
parse_args() {
	while [[ $# -gt 0 ]]; do
		case "$1" in
		-i | --interactive)
			IS_INTERACTIVE=1
			shift
			;;
		-y | --yes | -n | --non-interactive | --unattended | --batch)
			IS_INTERACTIVE=0
			shift
			;;
		-c | --components)
			parse_components "$2"
			shift 2
			;;
		--components=*)
			parse_components "${1#*=}"
			shift
			;;
		--dest-bin)
			DEST_BIN_DIR="$2"
			shift 2
			;;
		--dest-bin=*)
			DEST_BIN_DIR="${1#*=}"
			shift
			;;
		--backup-dir)
			BACKUP_STORAGE_DIR="$2"
			shift 2
			;;
		--backup-dir=*)
			BACKUP_STORAGE_DIR="${1#*=}"
			shift
			;;
		--skip-backup)
			SKIP_BACKUP=1
			shift
			;;
		--go-manager)
			GO_MANAGER_STRATEGY="$2"
			shift 2
			;;
		--go-manager=*)
			GO_MANAGER_STRATEGY="${1#*=}"
			shift
			;;
		--node-manager)
			NODE_MANAGER="$2"
			shift 2
			;;
		--node-manager=*)
			NODE_MANAGER="${1#*=}"
			shift
			;;
		--rollback-on-error)
			ROLLBACK_ON_ERROR=1
			shift
			;;
		-p | --plan | --dry-run | preview)
			SHOW_PLAN_ONLY=1
			shift
			;;
		-h | --help)
			usage
			exit 0
			;;
		*)
			log_error "Unrecognized flag or argument: $1"
			usage
			exit 1
			;;
		esac
	done

	# Detect if non-interactive environment (CI, pipe, background)
	if [[ ! -t 0 ]] || [[ ! -t 1 ]]; then
		IS_INTERACTIVE=0
	fi
}

# ==============================================================================
# Upfront Parameter Interview (Interactive Mode Only)
# Collects all user answers, configurations and consents BEFORE execution starts.
# ==============================================================================
interview_upfront() {
	wizard_init "arturonavax/env · Master Environment Wizard" 6 "Upfront parameter interview"
	wizard_banner

	# STEP 1: Pre-flight Diagnostics & Discovery
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

	if ((free_disk < 3000)); then
		warn "Available disk space is low (${free_disk} MB < 3000 MB recommended)."
		if ! confirm "Continue despite low disk space?" "n"; then
			log_fatal "Installation cancelled: Insufficient disk space." 1
		fi
	fi

	echo
	step "Verifying remote connectivity in parallel..."
	run_parallel_task_pool "Network Verification" \
		"GitHub Connection:::curl --proto '=https' --tlsv1.2 -fsSL --connect-timeout 4 'https://raw.githubusercontent.com' >/dev/null" \
		"CDN Endpoint:::curl --proto '=https' --tlsv1.2 -fsSL --connect-timeout 4 '${REMOTE_PRIMARY_URL}' >/dev/null 2>&1 || true"

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

	pause "Press Enter to proceed to path configuration..."

	# STEP 2: Paths & Destinations
	wizard_stage "Installation Directories & Safety Paths"
	say "Configure target paths for binaries and backup snapshots."
	echo

	ask DEST_BIN_DIR "Binary destination directory" "${DEST_BIN_DIR}"
	ask BACKUP_STORAGE_DIR "Backup snapshot directory" "${BACKUP_STORAGE_DIR}"
	export DEST_BIN_DIR BACKUP_STORAGE_DIR
	export PATH="${DEST_BIN_DIR}:${PATH}"
	mkdir -p "$DEST_BIN_DIR" "$BACKUP_STORAGE_DIR"

	pause "Press Enter to proceed to component selection..."

	# STEP 3: Subsystems Selection
	wizard_stage "Modular Subsystem Selection"
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

	pause "Press Enter to proceed to toolchain preferences..."

	# STEP 4: Toolchain & Runtime Preferences
	if ((DO_REQUIREMENTS == 1)); then
		wizard_stage "Toolchain & Runtime Preferences"
		say "Configure installation strategies for language runtimes and package managers."
		echo

		local go_choice=""
		select_option go_choice "Select Golang version management strategy:" \
			"GVM (Go Version Manager - Multi-version isolation - Recommended)" \
			"Standalone binary (Latest release directly from go.dev JSON API)" \
			"Native Package Manager (Distro repository package)"

		case "$go_choice" in
		*"GVM"*) GO_MANAGER_STRATEGY="gvm" ;;
		*"Standalone"*) GO_MANAGER_STRATEGY="standalone" ;;
		*"Package"*) GO_MANAGER_STRATEGY="package" ;;
		*) GO_MANAGER_STRATEGY="gvm" ;;
		esac
		export GO_MANAGER_STRATEGY

		local node_choice=""
		select_option node_choice "Select Node.js package manager preference:" \
			"pnpm (Fast, disk-space efficient package manager - Recommended)" \
			"npm (Standard Node package manager bundled with Node.js)"

		case "$node_choice" in
		*"pnpm"*) NODE_MANAGER="pnpm" ;;
		*"npm"*) NODE_MANAGER="npm" ;;
		*) NODE_MANAGER="pnpm" ;;
		esac
		export NODE_MANAGER

		pause "Press Enter to proceed to OS integration..."
	fi

	# STEP 5: OS Desktop Settings & Window Switcher
	if ((DO_OSCONFIG == 1)); then
		wizard_stage "Operating System & Desktop Preferences"
		say "Configure desktop environment integrations, window switchers, and shortcuts."
		echo

		if [[ "$os" == "linux" ]]; then
			confirm "Enable rcmd application launcher & window switcher (Right-Alt hotkeys)?" "y" && DO_RCMD=1 || DO_RCMD=0
			export DO_RCMD
		elif [[ "$os" == "darwin" ]]; then
			confirm "Apply macOS developer optimizations (Safari debug, spaces animation, sudo TouchID)?" "y" && DO_MACOS_TWEAKS=1 || DO_MACOS_TWEAKS=0
			export DO_MACOS_TWEAKS
		fi

		pause "Press Enter to review policy & confirmation..."
	fi

	# STEP 6: Safety Policy, Plan Preview & Final Confirmation Gate
	wizard_stage "Safety Policy & Execution Confirmation Gate"
	say "Review installation summary and safety policy before execution begins."
	echo

	step "Selected components:"
	if [[ "$DO_REQUIREMENTS" == "1" ]]; then say "  ${GREEN}${SYM_CHECK}${RESET} Base Compilers & Languages (${GO_MANAGER_STRATEGY})"; else say "  ${GRAY}${SYM_CROSS} Base Compilers & Languages (skipped)${RESET}"; fi
	if [[ "$DO_FONTS" == "1" ]]; then say "  ${GREEN}${SYM_CHECK}${RESET} Patched Nerd Fonts"; else say "  ${GRAY}${SYM_CROSS} Patched Nerd Fonts (skipped)${RESET}"; fi
	if [[ "$DO_TERMINAL" == "1" ]]; then say "  ${GREEN}${SYM_CHECK}${RESET} Modern Terminal Stack (Ghostty, Herdr, Zsh, Atuin)"; else say "  ${GRAY}${SYM_CROSS} Modern Terminal Stack (skipped)${RESET}"; fi
	if [[ "$DO_EDITOR" == "1" ]]; then say "  ${GREEN}${SYM_CHECK}${RESET} Code Editor Suite (Neovim v0.12+, LazyVim)"; else say "  ${GRAY}${SYM_CROSS} Code Editor Suite (skipped)${RESET}"; fi
	if [[ "$DO_AI" == "1" ]]; then say "  ${GREEN}${SYM_CHECK}${RESET} AI Tooling & Ecosystem (agy, RTK, MCP)"; else say "  ${GRAY}${SYM_CROSS} AI Tooling & Ecosystem (skipped)${RESET}"; fi
	if [[ "$DO_OSCONFIG" == "1" ]]; then say "  ${GREEN}${SYM_CHECK}${RESET} OS Integration & Shortcuts"; else say "  ${GRAY}${SYM_CROSS} OS Integration & Shortcuts (skipped)${RESET}"; fi

	step "Binary destination:    ${BOLD_WHITE}${DEST_BIN_DIR}${RESET}"
	step "Backup destination:    ${BOLD_WHITE}${BACKUP_STORAGE_DIR}${RESET}"
	echo

	if confirm "Enable automatic transactional rollback if a fatal error occurs?" "y"; then
		ROLLBACK_ON_ERROR=1
	else
		ROLLBACK_ON_ERROR=0
	fi

	echo
	if confirm "Preview detailed filesystem interaction manifest & plan before proceeding?" "n"; then
		local selected=()
		[[ "$DO_REQUIREMENTS" == "1" ]] && selected+=("requirements")
		[[ "$DO_FONTS" == "1" ]] && selected+=("fonts")
		[[ "$DO_TERMINAL" == "1" ]] && selected+=("terminal")
		[[ "$DO_EDITOR" == "1" ]] && selected+=("editor")
		[[ "$DO_AI" == "1" ]] && selected+=("ai")
		[[ "$DO_OSCONFIG" == "1" ]] && selected+=("osconfig")
		show_install_plan "${selected[@]}"
		echo
	fi

	if ! confirm "Proceed with autonomous installation now?" "y"; then
		log_info "Installation cancelled by operator. No files modified."
		exit 0
	fi
}

# ==============================================================================
# Stage Execution Wrapper with Transactional Failure Handling
# ==============================================================================
run_stage() {
	local stage_name="$1"
	local stage_cmd="$2"

	wizard_stage "$stage_name"
	log_step "Executing stage: ${stage_name}..."

	local status=0
	eval "$stage_cmd" || status=$?

	if ((status != 0)); then
		log_error "Stage '${stage_name}' failed with exit code ${status}."
		if ((ROLLBACK_ON_ERROR == 1)); then
			tx_rollback
			log_fatal "Installation aborted. Host restored cleanly via transactional rollback." "$status"
		elif ((IS_INTERACTIVE == 1)); then
			handle_stage_failure "$stage_name" "$status" "$ENV_LOG_FILE"
			local rec=$?
			if ((rec == 10)); then
				# User selected retry
				run_stage "$stage_name" "$stage_cmd"
			fi
		else
			log_fatal "Stage '${stage_name}' failed with exit code ${status} in non-interactive mode." "$status"
		fi
	else
		log_success "Stage '${stage_name}' completed successfully."
	fi
}

# ==============================================================================
# Autonomous Execution Phase (Zero Questions)
# ==============================================================================
execute_autonomous() {
	local mode_label="Interactive"
	((IS_INTERACTIVE == 0)) && mode_label="Non-Interactive"

	# Calculate dynamic stage count for graphical progress bar
	local total_exec_stages=2 # Safety snapshot + OS config/verification
	((DO_REQUIREMENTS == 1)) && total_exec_stages=$((total_exec_stages + 1))
	((DO_FONTS == 1)) && total_exec_stages=$((total_exec_stages + 1))
	((DO_TERMINAL == 1)) && total_exec_stages=$((total_exec_stages + 1))
	((DO_EDITOR == 1)) && total_exec_stages=$((total_exec_stages + 1))
	((DO_AI == 1)) && total_exec_stages=$((total_exec_stages + 1))

	wizard_init "arturonavax/env · Autonomous Execution" "$total_exec_stages" "Executing configured installation tasks (${mode_label})"

	# Export runtime paths and toolchain preferences
	export DEST_BIN_DIR BACKUP_STORAGE_DIR GO_MANAGER_STRATEGY NODE_MANAGER
	export PATH="${DEST_BIN_DIR}:${PATH}"

	# 1. Create Safety Snapshot
	local snapshot_path=""
	if ((SKIP_BACKUP == 0)); then
		wizard_stage "Configuration Snapshot & Safety Guard"
		say "Creating restorable configuration snapshot in ${BACKUP_STORAGE_DIR}..."
		snapshot_path="$(backup_create)"
		export WIZARD_INITIAL_BACKUP="$snapshot_path"
		step "Snapshot ready: ${BOLD_WHITE}$(basename "$snapshot_path")${RESET}"
	fi

	# 2. Initialize Transaction & Audit Journal
	tx_begin "arturonavax/env Installation (${mode_label})" "$snapshot_path"

	# 3. Base Requirements & Compilers
	if ((DO_REQUIREMENTS == 1)); then
		run_stage "Base Compilers & Runtimes" "bash ./src/_install_requirements.sh"
	fi

	# 4. Terminal Stack & CLI Tools
	if ((DO_TERMINAL == 1)); then
		wizard_stage "Terminal Stack & Plugin Synchronization"
		step "Synchronizing Zsh plugins in parallel..."
		mkdir -p "$HOME/.zsh"
		run_parallel_task_pool "Zsh Plugins Sync" \
			"fzf-tab:::if [[ -d '$HOME/.zsh/fzf-tab/.git' ]]; then git -C '$HOME/.zsh/fzf-tab' pull --rebase --quiet; else rm -rf '$HOME/.zsh/fzf-tab'; git clone --quiet https://github.com/Aloxaf/fzf-tab '$HOME/.zsh/fzf-tab'; fi" \
			"zsh-autosuggestions:::if [[ -d '$HOME/.zsh/zsh-autosuggestions/.git' ]]; then git -C '$HOME/.zsh/zsh-autosuggestions' pull --rebase --quiet; else rm -rf '$HOME/.zsh/zsh-autosuggestions'; git clone --quiet https://github.com/zsh-users/zsh-autosuggestions '$HOME/.zsh/zsh-autosuggestions'; fi" \
			"fast-syntax-highlighting:::if [[ -d '$HOME/.zsh/fast-syntax-highlighting/.git' ]]; then git -C '$HOME/.zsh/fast-syntax-highlighting' pull --rebase --quiet; else rm -rf '$HOME/.zsh/fast-syntax-highlighting'; git clone --quiet https://github.com/zdharma-continuum/fast-syntax-highlighting '$HOME/.zsh/fast-syntax-highlighting'; fi" \
			"zsh-completions:::if [[ -d '$HOME/.zsh/zsh-completions/.git' ]]; then git -C '$HOME/.zsh/zsh-completions' pull --rebase --quiet; else rm -rf '$HOME/.zsh/zsh-completions'; git clone --quiet https://github.com/zsh-users/zsh-completions.git '$HOME/.zsh/zsh-completions'; fi"

		run_stage "Deploying Terminal Suite" "bash ./src/install_terminal.sh"
	fi

	# 5. Code Editor Suite
	if ((DO_EDITOR == 1)); then
		run_stage "Code Editor Suite (Neovim + LazyVim)" "bash ./src/install_editor.sh"
	fi

	# 6. AI Tooling & Ecosystem
	if ((DO_AI == 1)); then
		run_stage "AI Tooling & Ecosystem" "bash ./src/install_ai.sh"
	fi

	# 7. Patched Fonts
	if ((DO_FONTS == 1)); then
		run_stage "Patched Nerd Fonts" "bash ./src/remotes/install_fonts.sh"
	fi

	# 8. OS Configuration, Dotfiles & Verification
	wizard_stage "OS Configuration & Dotfiles Integration"
	if ((DO_OSCONFIG == 1)); then
		local os
		os="$(detect_os)"
		if [[ "$os" == "linux" ]]; then
			step "Applying Linux OS configuration..."
			bash ./src/osconfig/linux_osconfig.sh || true
		elif [[ "$os" == "darwin" ]]; then
			step "Applying macOS configuration..."
			bash ./src/osconfig/macos/macos_osconfig.sh || true
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

	# Propagate dotfiles
	step "Propagating managed dotfiles into active environment..."
	bash ./utils/sync_config.sh all || true

	# End-to-end Binary Verification
	step "Running end-to-end binary verification in PATH:"
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

	# Finalize Transaction
	tx_commit

	echo
	if ((missing_count == 0)); then
		wizard_finish "All systems deployed and verified cleanly!"
	else
		wizard_finish "Installation finished with ${missing_count} binaries pending or not in PATH."
	fi
}

# ==============================================================================
# Main Entry Point
# ==============================================================================
main() {
	parse_args "$@"

	# Plan / Dry-run preview
	if ((SHOW_PLAN_ONLY == 1)); then
		local selected=()
		[[ "$DO_REQUIREMENTS" == "1" ]] && selected+=("requirements")
		[[ "$DO_FONTS" == "1" ]] && selected+=("fonts")
		[[ "$DO_TERMINAL" == "1" ]] && selected+=("terminal")
		[[ "$DO_EDITOR" == "1" ]] && selected+=("editor")
		[[ "$DO_AI" == "1" ]] && selected+=("ai")
		[[ "$DO_OSCONFIG" == "1" ]] && selected+=("osconfig")
		show_install_plan "${selected[@]}"
		exit 0
	fi

	if ((IS_INTERACTIVE == 1)); then
		# Phase 1: Upfront Interview (collects everything before execution)
		interview_upfront

		# Phase 2: Autonomous Execution (zero questions)
		execute_autonomous
	else
		# Non-Interactive Mode (completely autonomous)
		log_box "arturonavax/env Non-Interactive Installation" \
			"Platform: $(detect_os) ($(detect_distro) - $(detect_arch))" \
			"Autonomous execution mode active (zero prompts)."

		# Sudo validation in non-interactive mode
		local os
		os="$(detect_os)"
		if [[ "$os" != "android" ]] && [[ "$(detect_distro)" != "termux" ]]; then
			if ! sudo -n true 2>/dev/null && ! has_sudo_capability; then
				log_fatal "Non-interactive installation requires passwordless sudo or root privileges." 1
			fi
			ensure_sudo
		fi

		execute_autonomous
	fi
}

main "$@"
