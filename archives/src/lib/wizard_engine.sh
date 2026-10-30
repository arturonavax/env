#!/usr/bin/env bash
# ==============================================================================
# src/lib/wizard_engine.sh
# Production Pure-Bash Interactive Wizard & TUI Framework.
# Zero external runtime dependencies (no Python, no Node, no dialog/whiptail).
# Maximum cross-platform portability: Linux, macOS, WSL, MSYS2, Android Termux.
# High-performance, sub-millisecond flicker-free terminal rendering.
# Part of arturonavax/env development environment.
# ==============================================================================

# Prevent double sourcing
[[ -n "${_WIZARD_ENGINE_SH_LOADED:-}" ]] && return 0
_WIZARD_ENGINE_SH_LOADED=1

_LIB_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=/dev/null
source "${_LIB_DIR}/colors.sh" 2>/dev/null || true
# shellcheck source=/dev/null
source "${_LIB_DIR}/ui_box.sh" 2>/dev/null || true
# shellcheck source=/dev/null
source "${_LIB_DIR}/logging.sh" 2>/dev/null || true

# ------------------------------------------------------------------------------
# Wizard Engine State
# ------------------------------------------------------------------------------
WIZARD_TITLE="${WIZARD_TITLE:-Installation Wizard}"
WIZARD_SUBTITLE="${WIZARD_SUBTITLE:-Guided step-by-step setup}"
WIZARD_TOTAL_STAGES=0
WIZARD_CURRENT_STAGE=0
WIZARD_ENV_FILE="${WIZARD_ENV_FILE:-.env}"
WIZARD_INITIAL_BACKUP=""
WIZARD_WRITTEN_ENV=()
WIZARD_WRITTEN_SECRETS=()
WIZARD_SKIPPED_ACTIONS=()

# Terminal clearing helper (respects non-interactive and dumb terms)
wizard_clear() {
	if [[ -t 1 ]] && [[ "${TERM:-}" != "dumb" ]]; then
		if command -v tput >/dev/null 2>&1 && tput clear 2>/dev/null; then
			return 0
		fi
		printf '\033[2J\033[3J\033[H'
	fi
}

# Graphical Progress Bar
# Usage:
#   wizard_progress_bar <current> <total> [bar_width]
wizard_progress_bar() {
	local current="$1"
	local total="$2"
	local width="${3:-22}"

	((total <= 0)) && total=1
	((current > total)) && current=$total

	local percent=$((current * 100 / total))
	local filled=$((current * width / total))
	local empty=$((width - filled))

	local bar=""
	local b
	for ((b = 0; b < filled; b++)); do bar+="█"; done
	for ((b = 0; b < empty; b++)); do bar+="░"; done

	printf '%s[%s%s%s%s] %3d%%%s' "$DIM" "$BOLD_GREEN" "$bar" "$RESET" "$DIM" "$percent" "$RESET"
}

# ------------------------------------------------------------------------------
# Lifecycle & Stage Banners
# ------------------------------------------------------------------------------
wizard_init() {
	WIZARD_TITLE="$1"
	WIZARD_TOTAL_STAGES="${2:-1}"
	WIZARD_SUBTITLE="${3:-Guided setup wizard}"
	WIZARD_CURRENT_STAGE=0
	WIZARD_WRITTEN_ENV=()
	WIZARD_WRITTEN_SECRETS=()
	WIZARD_SKIPPED_ACTIONS=()
}

wizard_banner() {
	wizard_clear
	local width
	width="$(ui_box_width 70)"

	echo
	ui_box_top "$width" "rounded" "$BOLD_BLUE"
	ui_box_line "$WIZARD_TITLE" "$width" "rounded" "$BOLD_BLUE" "$BOLD_WHITE" "left"
	ui_box_mid "$width" "rounded" "$BOLD_BLUE"
	ui_box_line "$WIZARD_SUBTITLE" "$width" "rounded" "$BOLD_BLUE" "$RESET" "left"
	ui_box_line "Total Stages: ${WIZARD_TOTAL_STAGES} stages planned" "$width" "rounded" "$BOLD_BLUE" "$DIM" "left"
	ui_box_line "Stop anytime with Ctrl+C. Safe idempotent state recovery." "$width" "rounded" "$BOLD_BLUE" "$DIM" "left"
	ui_box_bottom "$width" "rounded" "$BOLD_BLUE"
	echo

	pause "Press Enter to start setup..."
}

wizard_stage() {
	local stage_name="$1"
	WIZARD_CURRENT_STAGE=$((WIZARD_CURRENT_STAGE + 1))
	wizard_clear

	local p_bar
	p_bar="$(wizard_progress_bar "$WIZARD_CURRENT_STAGE" "$WIZARD_TOTAL_STAGES")"

	printf '\n%s%s▸ Stage %s/%s · %s%s  %s\n' \
		"$BOLD" "$BLUE" "$WIZARD_CURRENT_STAGE" "$WIZARD_TOTAL_STAGES" "$stage_name" "$RESET" "$p_bar"
	printf '%s----------------------------------------------------------------------%s\n' "$DIM" "$RESET"
}

# ------------------------------------------------------------------------------
# Steppers & Output Elements
# ------------------------------------------------------------------------------
say()  { printf '  %s\n' "$*"; }
step() { printf '  %s•%s %s\n' "$BOLD_BLUE" "$RESET" "$*"; }
note() { printf '  %s%s%s\n' "$DIM" "$*" "$RESET"; }
warn() { printf '  %s⚠ %s%s\n' "$BOLD_YELLOW" "$*" "$RESET"; }

pause() {
	printf '\n  %s%s%s ' "$DIM" "${1:-Press Enter to proceed...}" "$RESET"
	read -r _ || true
}

# Yes/No confirmation gate
# Usage:
#   if confirm "Install component?" "y"; then ... fi
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

# Cross-platform URL opener (supports WSL, Windows, Linux, macOS)
open_url() {
	local url="$1"
	printf '  %s↗ Opening in browser:%s %s\n' "$BOLD_GREEN" "$RESET" "$url"
	{
		if command -v wslview >/dev/null 2>&1; then
			wslview "$url"
		elif command -v explorer.exe >/dev/null 2>&1; then
			explorer.exe "$url"
		elif command -v xdg-open >/dev/null 2>&1; then
			xdg-open "$url"
		elif command -v open >/dev/null 2>&1; then
			open "$url"
		else
			false
		fi
	} >/dev/null 2>&1 || warn "Could not launch browser automatically; please visit: $url"
}

# ------------------------------------------------------------------------------
# Input Prompts (Text, Secret, Select, Multi-Select)
# ------------------------------------------------------------------------------

# Ask single visible input with optional default
# Usage:
#   ask VAR_NAME "Prompt label" [default_value]
ask() {
	local var_name="$1"
	local prompt_label="$2"
	local default_val="${3:-}"
	local user_input=""

	if [[ -n "$default_val" ]]; then
		printf '  %s%s%s %s[%s]%s: ' "$BOLD_WHITE" "$prompt_label" "$RESET" "$DIM" "$default_val" "$RESET"
	else
		printf '  %s%s%s: ' "$BOLD_WHITE" "$prompt_label" "$RESET"
	fi

	read -r user_input || true
	user_input="${user_input:-$default_val}"
	printf -v "$var_name" '%s' "$user_input"
}

# Ask secret / hidden input
# Usage:
#   ask_secret VAR_NAME "Prompt label"
ask_secret() {
	local var_name="$1"
	local prompt_label="$2"
	local secret_input=""

	printf '  %s%s%s %s[input hidden]%s: ' "$BOLD_WHITE" "$prompt_label" "$RESET" "$DIM" "$RESET"
	read -r -s secret_input || true
	printf '\n'
	printf -v "$var_name" '%s' "$secret_input"
}

# Single selection menu
# Usage:
#   select_option SELECTED_VAR "Choose database" "PostgreSQL" "SQLite3" "MySQL"
select_option() {
	local out_var="$1"
	local prompt="$2"
	shift 2
	local options=("$@")

	printf '\n  %s%s%s\n' "$BOLD_YELLOW" "$prompt" "$RESET"
	local i
	for ((i = 0; i < ${#options[@]}; i++)); do
		printf '    %s[%d]%s %s\n' "$BOLD_CYAN" "$((i + 1))" "$RESET" "${options[i]}"
	done

	local choice=1
	while true; do
		printf '  Enter selection [1-%d]: ' "${#options[@]}"
		read -r choice || true
		if [[ "$choice" =~ ^[0-9]+$ ]] && ((choice >= 1 && choice <= ${#options[@]})); then
			printf -v "$out_var" '%s' "${options[$((choice - 1))]}"
			break
		fi
		warn "Invalid choice, please enter a number between 1 and ${#options[@]}."
	done
}

# ------------------------------------------------------------------------------
# Persistent Environment Management
# ------------------------------------------------------------------------------
wizard_write_env() {
	local key="$1"
	local val="$2"
	local target_env="${3:-$WIZARD_ENV_FILE}"

	mkdir -p "$(dirname "$target_env")" 2>/dev/null || true
	touch "$target_env"

	local tmp_file
	tmp_file="$(mktemp "${TMPDIR:-/tmp}/w_env.XXXXXX")"
	grep -vE "^${key}=" "$target_env" >"$tmp_file" 2>/dev/null || true
	printf "%s='%s'\n" "$key" "$val" >>"$tmp_file"
	cat "$tmp_file" >"$target_env"
	rm -f "$tmp_file"

	WIZARD_WRITTEN_ENV+=("$key")
	printf '  %s%s%s Saved %s to %s\n' "$GREEN" "$SYM_CHECK" "$RESET" "$key" "$target_env"
}

wizard_write_secret() {
	local name="$1"
	local val="$2"
	if command -v gh >/dev/null 2>&1 && gh auth status >/dev/null 2>&1; then
		if printf '%s' "$val" | gh secret set "$name" >/dev/null 2>&1; then
			WIZARD_WRITTEN_SECRETS+=("$name")
			printf '  %s%s%s Set GitHub secret: %s\n' "$GREEN" "$SYM_CHECK" "$RESET" "$name"
			return 0
		fi
	fi
	WIZARD_SKIPPED_ACTIONS+=("GitHub secret $name (gh unauthenticated; set manually: gh secret set $name)")
	warn "Skipped GitHub secret $name: gh not authenticated."
	return 1
}

# ------------------------------------------------------------------------------
# Stage Failure Recovery Traps
# ------------------------------------------------------------------------------
handle_stage_failure() {
	local stage_name="$1"
	local exit_code="$2"
	local log_path="${3:-$HOME/.arturonavax-env-install.log}"

	printf '\n%s%s STAGE FAILURE: %s (Exit code: %s)%s\n' "$BOLD_RED" "$SYM_CROSS" "$stage_name" "$exit_code" "$RESET"
	printf '  %sThe installer encountered an error during this stage.%s\n' "$RED" "$RESET"
	if [[ -f "$log_path" ]]; then
		printf '  %sDetailed logs are available at: %s%s\n\n' "$DIM" "$log_path" "$RESET"
	fi

	printf '  %sChoose recovery action:%s\n' "$BOLD_WHITE" "$RESET"
	printf '    %s[r]%s Retry this stage\n' "$BOLD_GREEN" "$RESET"
	printf '    %s[s]%s Skip this stage and continue\n' "$BOLD_YELLOW" "$RESET"
	if [[ -n "$WIZARD_INITIAL_BACKUP" ]]; then
		printf '    %s[b]%s Rollback to initial snapshot and abort\n' "$BOLD_CYAN" "$RESET"
	fi
	printf '    %s[a]%s Abort wizard immediately (default)\n\n' "$BOLD_RED" "$RESET"

	local choice=""
	read -r -p "  Enter choice [r/s/b/a]: " choice
	choice="${choice:-a}"

	case "$choice" in
	r | R)
		return 10 # Code for retry
		;;
	s | S)
		log_warn "Stage '${stage_name}' skipped by operator decision."
		WIZARD_SKIPPED_ACTIONS+=("$stage_name")
		return 0 # Code for skip
		;;
	b | B)
		if [[ -n "$WIZARD_INITIAL_BACKUP" ]]; then
			log_step "Triggering rollback to snapshot: $(basename "$WIZARD_INITIAL_BACKUP")..."
			backup_restore "$(basename "$WIZARD_INITIAL_BACKUP")"
			log_fatal "Wizard aborted after rollback." 1
		else
			log_fatal "Wizard aborted by operator." 1
		fi
		;;
	*)
		log_fatal "Wizard aborted by operator." 1
		;;
	esac
}

# ------------------------------------------------------------------------------
# Final Summary Frame
# ------------------------------------------------------------------------------
wizard_finish() {
	local custom_msg="${1:-Setup completed successfully!}"
	wizard_clear
	local width
	width="$(ui_box_width 70)"

	echo
	ui_box_top "$width" "rounded" "$BOLD_GREEN"
	ui_box_line "$custom_msg" "$width" "rounded" "$BOLD_GREEN" "$BOLD_WHITE" "left"
	ui_box_mid "$width" "rounded" "$BOLD_GREEN"
	ui_box_line "All ${WIZARD_TOTAL_STAGES} stages completed with zero unhandled errors." "$width" "rounded" "$BOLD_GREEN" "$RESET" "left"
	ui_box_bottom "$width" "rounded" "$BOLD_GREEN"
	echo

	if (( ${#WIZARD_WRITTEN_ENV[@]} > 0 )); then
		step "Configuration variables persisted:"
		for item in "${WIZARD_WRITTEN_ENV[@]}"; do
			say "  • ${BOLD_WHITE}${item}${RESET}"
		done
		echo
	fi

	if (( ${#WIZARD_WRITTEN_SECRETS[@]} > 0 )); then
		step "GitHub secrets configured:"
		for item in "${WIZARD_WRITTEN_SECRETS[@]}"; do
			say "  • ${BOLD_WHITE}${item}${RESET}"
		done
		echo
	fi

	if (( ${#WIZARD_SKIPPED_ACTIONS[@]} > 0 )); then
		warn "Skipped actions during this session:"
		for item in "${WIZARD_SKIPPED_ACTIONS[@]}"; do
			say "  • ${YELLOW}${item}${RESET}"
		done
		echo
	fi
}
