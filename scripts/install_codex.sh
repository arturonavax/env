#!/usr/bin/env bash
# install_codex.sh - Idempotent OpenAI Codex CLI standalone installer for ANX.
# Replaces broken single-binary ~/.local/bin/codex installs with the full
# standalone Codex package layout (~/.codex/packages/standalone/current/).

set -euo pipefail

LOCAL_BIN_DIR="${HOME}/.local/bin"
CODEX_BIN="${LOCAL_BIN_DIR}/codex"
PACKAGE_MARKER="${HOME}/.codex/packages/standalone/current/codex-package.json"
CODEX_INSTALL_URL="${ANX_CODEX_INSTALL_URL:-https://chatgpt.com/codex/install.sh}"
_ANX_CODEX_TMP_DIR=""

export PATH="${LOCAL_BIN_DIR}:/opt/homebrew/bin:/usr/local/bin:/home/linuxbrew/.linuxbrew/bin:${PATH}"

log_info() {
	printf '[INFO] %s\n' "$*"
}

log_err() {
	printf '[ERROR] %s\n' "$*" >&2
}

cleanup_codex_tmp() {
	if [[ -n "${_ANX_CODEX_TMP_DIR:-}" && -d "${_ANX_CODEX_TMP_DIR}" ]]; then
		rm -rf "${_ANX_CODEX_TMP_DIR}"
	fi
}

has_valid_standalone_package() {
	[[ -f "${PACKAGE_MARKER}" ]] && [[ -x "${CODEX_BIN}" ]] && "${CODEX_BIN}" --version >/dev/null 2>&1
}

remove_unpackaged_local_binary() {
	if [[ -e "${CODEX_BIN}" || -L "${CODEX_BIN}" ]] && [[ ! -f "${PACKAGE_MARKER}" ]]; then
		log_info "Removing incomplete single-binary Codex at ${CODEX_BIN}..."
		rm -f "${CODEX_BIN}"
	fi
}

install_via_official_script() {
	local installer
	if ! command -v curl >/dev/null 2>&1; then
		log_err "curl is required to download Codex CLI installer"
		return 1
	fi

	mkdir -p "${LOCAL_BIN_DIR}"
	_ANX_CODEX_TMP_DIR="$(mktemp -d "${TMPDIR:-/tmp}/anx-codex.XXXXXX")"
	trap cleanup_codex_tmp EXIT INT TERM

	installer="${_ANX_CODEX_TMP_DIR}/codex-install.sh"
	log_info "Downloading official Codex CLI installer from ${CODEX_INSTALL_URL}..."
	curl --proto '=https' --tlsv1.2 -fsSL "${CODEX_INSTALL_URL}" -o "${installer}"

	log_info "Installing standalone Codex CLI package into ${LOCAL_BIN_DIR}..."
	CODEX_NON_INTERACTIVE=true CODEX_INSTALL_DIR="${LOCAL_BIN_DIR}" sh "${installer}"

	cleanup_codex_tmp
	_ANX_CODEX_TMP_DIR=""
	trap - EXIT INT TERM
}

main() {
	if has_valid_standalone_package; then
		log_info "Codex CLI standalone package is already installed: $("${CODEX_BIN}" --version)"
		return 0
	fi

	remove_unpackaged_local_binary
	install_via_official_script

	export PATH="${LOCAL_BIN_DIR}:${PATH}"
	if [[ ! -x "${CODEX_BIN}" ]] || ! "${CODEX_BIN}" --version >/dev/null 2>&1; then
		log_err "codex binary at ${CODEX_BIN} is missing or non-functional after installation"
		return 1
	fi

	if [[ ! -f "${PACKAGE_MARKER}" ]]; then
		log_err "codex installed at ${CODEX_BIN} without standalone package metadata (${PACKAGE_MARKER})"
		return 1
	fi

	log_info "Verified $("${CODEX_BIN}" --version)"
}

main "$@"
