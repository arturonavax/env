#!/bin/bash
# Run: source <(curl -fsSL "https://env.arturonavax.dev/_required_commands.sh" | cat)
repo_remote_files="https://env.arturonavax.dev"

# shellcheck disable=SC1090
source <(curl -fsSL "$repo_remote_files/_vars_colors.sh" | cat)

# required-commands terminates the execution if there are no commands
# shellcheck disable=SC2154
function required-commands() {
	concat_missing_commands=""

	for arg in "$@"; do
		[[ "$(command -v "$arg")" == "" ]] && concat_missing_commands+="\t$arg\n"
	done

	if [[ "$concat_missing_commands" != "" ]]; then
		echo -e "${fgcolor_white_bold}[Installer Error]: ${fgcolor_red_bold}Commands are required:${fgcolor_white_bold}"
		echo -e "$concat_missing_commands${fgcolor_reset}"

		return 1
	fi
}

# required-sudo-commands terminates the execution if no 'sudo' commands exist
function required-sudo-commands() {
	concat_missing_commands=""

	for arg in "$@"; do
		[[ "$(sudo which "$arg")" == "" ]] && concat_missing_commands+="\tsudo $arg\n"
	done

	if [[ "$concat_missing_commands" != "" ]]; then
		echo -e "${fgcolor_white_bold}[Installer Error]: ${fgcolor_red_bold}Commands with SUDO are required:${fgcolor_white_bold}"
		echo -e "$concat_missing_commands${fgcolor_reset}"

		return 1
	fi
}
