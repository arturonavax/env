#!/usr/bin/env bash
# install_node.sh - Idempotent Node.js LTS installer via fnm for ANX.
# Installs Node LTS through fnm, sets it as the default alias, and links
# node/npm/npx/corepack into ~/.local/bin so non-interactive tools (e.g.
# Neovim Mason LSP installers) find them immediately without shell reload.

set -euo pipefail

LOCAL_BIN_DIR="${HOME}/.local/bin"
export PATH="${LOCAL_BIN_DIR}:${HOME}/.local/share/fnm:${HOME}/Library/Application Support/fnm:/opt/homebrew/bin:/usr/local/bin:/home/linuxbrew/.linuxbrew/bin:${PATH}"

log_info() {
	printf '[INFO] %s\n' "$*"
}

log_err() {
	printf '[ERROR] %s\n' "$*" >&2
}

link_node_binaries() {
	local node_bin_dir="$1"
	local tool src dst

	mkdir -p "${LOCAL_BIN_DIR}"
	for tool in node npm npx corepack; do
		src="${node_bin_dir}/${tool}"
		dst="${LOCAL_BIN_DIR}/${tool}"
		if [[ -e "${src}" || -L "${src}" ]]; then
			ln -sfn "${src}" "${dst}"
		fi
	done
}

resolve_fnm_node_bin_dir() {
	local node_exec=""

	node_exec="$(fnm exec --using=default node -p 'process.execPath' 2>/dev/null || true)"
	if [[ -z "${node_exec}" ]]; then
		node_exec="$(fnm exec --using=lts-latest node -p 'process.execPath' 2>/dev/null || true)"
	fi
	if [[ -n "${node_exec}" && -x "${node_exec}" ]]; then
		dirname "${node_exec}"
		return 0
	fi
	return 1
}

main() {
	local node_bin_dir=""

	if command -v fnm >/dev/null 2>&1; then
		log_info "Ensuring Node.js LTS is installed via fnm..."
		fnm install --lts
		fnm default lts-latest >/dev/null 2>&1 || true

		if node_bin_dir="$(resolve_fnm_node_bin_dir)"; then
			link_node_binaries "${node_bin_dir}"
		else
			log_err "fnm installed Node.js LTS but executable path could not be resolved"
			return 1
		fi
	elif command -v node >/dev/null 2>&1 && node --version >/dev/null 2>&1 && command -v npm >/dev/null 2>&1 && npm --version >/dev/null 2>&1; then
		log_info "System node and npm are already installed; linking into ${LOCAL_BIN_DIR}..."
		node_bin_dir="$(dirname "$(command -v node)")"
		if [[ "${node_bin_dir}" != "${LOCAL_BIN_DIR}" ]]; then
			link_node_binaries "${node_bin_dir}"
		fi
	else
		log_err "fnm is required to install Node.js LTS (or install node/npm first)"
		return 1
	fi

	export PATH="${LOCAL_BIN_DIR}:${PATH}"
	if ! command -v node >/dev/null 2>&1 || ! node --version >/dev/null 2>&1; then
		log_err "node is missing or non-functional after installation"
		return 1
	fi
	if ! command -v npm >/dev/null 2>&1 || ! npm --version >/dev/null 2>&1; then
		log_err "npm is missing or non-functional after installation"
		return 1
	fi

	log_info "Verified node $(node --version) and npm $(npm --version)"
}

main "$@"
