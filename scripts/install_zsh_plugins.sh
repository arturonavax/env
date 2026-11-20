#!/usr/bin/env bash
# install_zsh_plugins.sh - Idempotent Zsh plugin installer for ANX.
# Installs fast-syntax-highlighting, zsh-autosuggestions, fzf-tab, and
# zsh-completions into ~/.zsh/ so syntax highlighting and completions work
# immediately on both macOS and Linux.

set -euo pipefail

ZSH_PLUGIN_DIR="${HOME}/.zsh"
READY_MARKER="${ZSH_PLUGIN_DIR}/.anx-plugins-ready"

FSH_URL="${ANX_ZSH_FSH_URL:-https://github.com/zdharma-continuum/fast-syntax-highlighting.git}"
AUTOSUGGEST_URL="${ANX_ZSH_AUTOSUGGEST_URL:-https://github.com/zsh-users/zsh-autosuggestions.git}"
FZFTAB_URL="${ANX_ZSH_FZFTAB_URL:-https://github.com/Aloxaf/fzf-tab.git}"
COMPLETIONS_URL="${ANX_ZSH_COMPLETIONS_URL:-https://github.com/zsh-users/zsh-completions.git}"

log_info() {
	printf '[INFO] %s\n' "$*"
}

log_err() {
	printf '[ERROR] %s\n' "$*" >&2
}

sync_plugin() {
	local name="$1"
	local url="$2"
	local entrypoint="$3"
	local dest="${ZSH_PLUGIN_DIR}/${name}"

	if [[ -d "${dest}/.git" ]] && [[ -e "${dest}/${entrypoint}" ]]; then
		log_info "Updating ${name}..."
		GIT_TERMINAL_PROMPT=0 git -C "${dest}" pull --ff-only --quiet >/dev/null 2>&1 || true
	else
		if [[ -e "${dest}" ]]; then
			rm -rf "${dest}"
		fi
		log_info "Cloning ${name}..."
		GIT_TERMINAL_PROMPT=0 git clone --depth=1 --quiet "${url}" "${dest}"
	fi

	if [[ ! -e "${dest}/${entrypoint}" ]]; then
		log_err "Plugin ${name} missing expected entrypoint ${dest}/${entrypoint}"
		return 1
	fi
}

main() {
	if ! command -v git >/dev/null 2>&1; then
		log_err "git is required to install Zsh plugins"
		return 1
	fi

	mkdir -p "${ZSH_PLUGIN_DIR}"
	rm -f "${READY_MARKER}"

	sync_plugin "zsh-completions" "${COMPLETIONS_URL}" "zsh-completions.plugin.zsh"
	sync_plugin "zsh-autosuggestions" "${AUTOSUGGEST_URL}" "zsh-autosuggestions.zsh"
	sync_plugin "fzf-tab" "${FZFTAB_URL}" "fzf-tab.plugin.zsh"
	sync_plugin "fast-syntax-highlighting" "${FSH_URL}" "fast-syntax-highlighting.plugin.zsh"

	rm -f "${HOME}/.zcompdump"* 2>/dev/null || true
	touch "${READY_MARKER}"
	log_info "Verified Zsh plugins in ${ZSH_PLUGIN_DIR}"
}

main "$@"
