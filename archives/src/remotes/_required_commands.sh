#!/usr/bin/env bash
# ==============================================================================
# src/remotes/_required_commands.sh
# Verifies required commands exist before proceeding with installation.
# Servable from https://env.arturonavax.dev/_required_commands.sh
# ==============================================================================

# Prevent double sourcing
[[ -n "${_REQUIRED_COMMANDS_SH_LOADED:-}" ]] && return 0
_REQUIRED_COMMANDS_SH_LOADED=1

repo_remote_files="${REMOTE_PRIMARY_URL:-https://env.arturonavax.dev}"

# Resolve colors locally first, fallback to remote
if [[ -f ./src/lib/colors.sh ]]; then
	# shellcheck source=/dev/null
	source ./src/lib/colors.sh
elif [[ -f "$(dirname "${BASH_SOURCE[0]}")/_vars_colors.sh" ]]; then
	# shellcheck source=/dev/null
	source "$(dirname "${BASH_SOURCE[0]}")/_vars_colors.sh"
elif [[ -f ./src/remotes/_vars_colors.sh ]]; then
	# shellcheck source=/dev/null
	source ./src/remotes/_vars_colors.sh
else
	# shellcheck disable=SC1090
	source <(curl -fsSL "$repo_remote_files/_vars_colors.sh" 2>/dev/null || true)
fi

# required-commands terminates execution if any command is missing
function required-commands() {
	local concat_missing_commands=""

	for arg in "$@"; do
		if ! command -v "$arg" >/dev/null 2>&1; then
			concat_missing_commands+="\t• $arg\n"
		fi
	done

	if [[ -n "$concat_missing_commands" ]]; then
		echo -e "${BOLD_WHITE:-}[Installer Error]: ${BOLD_RED:-}The following required commands are missing:${RESET:-}" >&2
		echo -e "${concat_missing_commands}${RESET:-}" >&2
		return 1
	fi
}

# required-sudo-commands terminates execution if any sudo command is missing
function required-sudo-commands() {
	local concat_missing_commands=""

	for arg in "$@"; do
		if ! sudo which "$arg" >/dev/null 2>&1; then
			concat_missing_commands+="\t• sudo $arg\n"
		fi
	done

	if [[ -n "$concat_missing_commands" ]]; then
		echo -e "${BOLD_WHITE:-}[Installer Error]: ${BOLD_RED:-}The following required sudo commands are missing:${RESET:-}" >&2
		echo -e "${concat_missing_commands}${RESET:-}" >&2
		return 1
	fi
}
