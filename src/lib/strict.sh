#!/usr/bin/env bash
# ==============================================================================
# src/lib/strict.sh
# Core defensive runtime, repository anchoring, strict error traps and sudo keep-alive.
# Part of arturonavax/env development environment.
# ==============================================================================

# Prevent double sourcing
[[ -n "${_STRICT_SH_LOADED:-}" ]] && return 0
_STRICT_SH_LOADED=1

# 1. Repository-Root Anchoring Loop
# Traverses upward until .git, files/, and src/ are found, ensuring execution from repo root.
while [[ ! -d ./.git/ && ! -d ./files/ && ! -d ./src/ ]]; do
	if [[ "$PWD" == "/" ]]; then
		echo "[Strict Error]: Repository root not found." >&2
		exit 1
	fi
	if ! cd ..; then
		echo "[Strict Error]: Failed to change directory to parent." >&2
		exit 1
	fi
done

# 2. Strict Mode
set -euo pipefail
IFS=$'\n\t'

# 3. Global Cleanup Hooks and Temp Files Array
declare -a _CLEANUP_HOOKS=()
declare -a _TEMP_FILES=()
_SUDO_KEEP_ALIVE_PID=""

# Register a temporary file or directory to be purged on exit
register_temp_file() {
	local path="$1"
	if [[ -n "$path" ]]; then
		_TEMP_FILES+=("$path")
	fi
}

# Register a cleanup callback command
register_cleanup_hook() {
	local hook="$1"
	if [[ -n "$hook" ]]; then
		_CLEANUP_HOOKS+=("$hook")
	fi
}

# 4. Exit and Error Handlers
on_exit() {
	local exit_code=$?

	# Terminate sudo keep-alive if running
	if [[ -n "$_SUDO_KEEP_ALIVE_PID" ]]; then
		kill "$_SUDO_KEEP_ALIVE_PID" 2>/dev/null || true
		wait "$_SUDO_KEEP_ALIVE_PID" 2>/dev/null || true
		_SUDO_KEEP_ALIVE_PID=""
	fi

	# Execute custom cleanup hooks in reverse order
	local i
	for ((i = ${#_CLEANUP_HOOKS[@]} - 1; i >= 0; i--)); do
		eval "${_CLEANUP_HOOKS[i]}" 2>/dev/null || true
	done

	# Remove registered temporary files
	for temp_path in "${_TEMP_FILES[@]}"; do
		if [[ -e "$temp_path" ]]; then
			rm -rf "$temp_path" 2>/dev/null || true
		fi
	done

	return "$exit_code"
}

on_error() {
	local exit_code=$?
	local failed_command="${BASH_COMMAND:-unknown}"
	local failed_line="${BASH_LINENO[0]:-unknown}"
	local failed_source="${BASH_SOURCE[1]:-${BASH_SOURCE[0]}}"

	# Disable error trap inside the handler to prevent loops
	trap - ERR

	# Use ANSI colors if available
	local red="\033[1;31m"
	local yellow="\033[1;33m"
	local white="\033[1;37m"
	local cyan="\033[1;36m"
	local dim="\033[2m"
	local reset="\033[0m"

	if [[ ! -t 2 ]] || [[ "${TERM:-}" == "dumb" ]]; then
		red="" yellow="" white="" cyan="" dim="" reset=""
	fi

	echo -e "\n${red}======================================================================${reset}" >&2
	echo -e "${white}[FATAL ERROR]: Command execution failed!${reset}" >&2
	echo -e "${red}======================================================================${reset}" >&2
	echo -e "  ${yellow}• Script:${reset}    ${failed_source}" >&2
	echo -e "  ${yellow}• Line:${reset}      ${failed_line}" >&2
	echo -e "  ${yellow}• Command:${reset}   ${white}${failed_command}${reset}" >&2
	echo -e "  ${yellow}• Exit Code:${reset} ${red}${exit_code}${reset}" >&2

	# Print callstack/traceback
	local frame_count=${#BASH_SOURCE[@]}
	if ((frame_count > 2)); then
		echo -e "\n  ${cyan}Call Stack / Traceback:${reset}" >&2
		local i
		for ((i = 1; i < frame_count; i++)); do
			local func="${FUNCNAME[$i]:-main}"
			local file="${BASH_SOURCE[$i]:-unknown}"
			local line="${BASH_LINENO[$((i - 1))]:-unknown}"
			echo -e "    ${dim}[$((i))]${reset} ${file}:${line} ${dim}in${reset} ${func}()" >&2
		done
	fi

	echo -e "${red}----------------------------------------------------------------------${reset}" >&2
	echo -e "${white}Execution halted to prevent silent corruption. Inspect the log above.${reset}" >&2
	echo -e "${red}======================================================================${reset}\n" >&2

	exit "$exit_code"
}

on_signal() {
	local signal_name="$1"
	echo -e "\n[Strict Alert]: Process interrupted by ${signal_name}. Exiting cleanly..." >&2
	exit 130
}

# Attach Traps
trap on_exit EXIT
trap on_error ERR
trap 'on_signal SIGINT' SIGINT
trap 'on_signal SIGTERM' SIGTERM

# 5. Sudo Privileges & Keep-Alive Manager
ensure_sudo() {
	# If running on Android Termux, single-user environment; no sudo required
	if [[ -n "${TERMUX_VERSION:-}" ]] || [[ "${PREFIX:-}" == *"com.termux"* ]] || [[ -d "/data/data/com.termux" ]]; then
		return 0
	fi

	# If running as root or sudo already active, proceed
	if [[ $EUID -eq 0 ]]; then
		return 0
	fi

	if ! command -v sudo >/dev/null 2>&1; then
		echo "[Strict Error]: 'sudo' command is required but not installed." >&2
		return 1
	fi

	# Prompt for password upfront
	if ! sudo -v; then
		echo "[Strict Error]: Sudo credentials verification failed." >&2
		return 1
	fi

	# Kill previous keep-alive if any
	if [[ -n "$_SUDO_KEEP_ALIVE_PID" ]]; then
		kill "$_SUDO_KEEP_ALIVE_PID" 2>/dev/null || true
	fi

	# Start background keep-alive loop (updates timestamp every 50 seconds)
	(
		while true; do
			sudo -n true 2>/dev/null || exit 0
			sleep 50
			kill -0 "$$" 2>/dev/null || exit 0
		done
	) &
	_SUDO_KEEP_ALIVE_PID=$!
	disown "$_SUDO_KEEP_ALIVE_PID" 2>/dev/null || true

	return 0
}
