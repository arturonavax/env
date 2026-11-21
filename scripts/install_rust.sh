#!/usr/bin/env bash
# install_rust.sh - Idempotent Rust toolchain installer/repair script for ANX.
# Ensures rustup configures a working `stable` default toolchain so `cargo` and
# `rustc` never fail on first invocation (e.g. Neovim plugins like avante.nvim).

set -euo pipefail

export CARGO_HOME="${CARGO_HOME:-${HOME}/.cargo}"
export RUSTUP_HOME="${RUSTUP_HOME:-${HOME}/.rustup}"
RUSTUP_INIT_URL="${ANX_RUSTUP_INIT_URL:-https://sh.rustup.rs}"
_ANX_RUSTUP_TMP_DIR=""

export PATH="${CARGO_HOME}/bin:/opt/homebrew/opt/rustup/bin:/usr/local/opt/rustup/bin:${HOME}/.local/bin:${PATH}"

log_info() {
	printf '[INFO] %s\n' "$*"
}

log_err() {
	printf '[ERROR] %s\n' "$*" >&2
}

cleanup_rustup_tmp() {
	if [[ -n "${_ANX_RUSTUP_TMP_DIR:-}" && -d "${_ANX_RUSTUP_TMP_DIR}" ]]; then
		rm -rf "${_ANX_RUSTUP_TMP_DIR}"
	fi
}

ensure_components() {
	if command -v rustup >/dev/null 2>&1; then
		rustup component add rust-analyzer clippy rustfmt >/dev/null 2>&1 || true
	fi
}

install_via_rustup_init() {
	local installer
	if ! command -v curl >/dev/null 2>&1; then
		log_err "curl is required to download rustup-init"
		return 1
	fi

	_ANX_RUSTUP_TMP_DIR="$(mktemp -d "${TMPDIR:-/tmp}/anx-rustup.XXXXXX")"
	trap cleanup_rustup_tmp EXIT INT TERM

	installer="${_ANX_RUSTUP_TMP_DIR}/rustup-init.sh"
	log_info "Downloading rustup installer from ${RUSTUP_INIT_URL}..."
	curl --proto '=https' --tlsv1.2 -fsSL "${RUSTUP_INIT_URL}" -o "${installer}"

	log_info "Installing Rust stable toolchain..."
	sh "${installer}" -y --no-modify-path --default-toolchain stable

	cleanup_rustup_tmp
	_ANX_RUSTUP_TMP_DIR=""
	trap - EXIT INT TERM
}

main() {
	if command -v rustup >/dev/null 2>&1; then
		if ! rustup show active-toolchain >/dev/null 2>&1 || ! cargo --version >/dev/null 2>&1 || ! rustc --version >/dev/null 2>&1; then
			log_info "Configuring default stable toolchain via existing rustup..."
			rustup toolchain install stable --profile default
			rustup default stable
		else
			log_info "Rust toolchain is already active."
		fi
		ensure_components
	elif command -v cargo >/dev/null 2>&1 && cargo --version >/dev/null 2>&1 && command -v rustc >/dev/null 2>&1 && rustc --version >/dev/null 2>&1; then
		log_info "System cargo and rustc are already installed and functional."
	else
		install_via_rustup_init
		if command -v rustup >/dev/null 2>&1 && ! rustup show active-toolchain >/dev/null 2>&1; then
			rustup default stable
		fi
		ensure_components
	fi

	if ! command -v cargo >/dev/null 2>&1 || ! cargo --version >/dev/null 2>&1; then
		log_err "cargo is still missing or non-functional after installation"
		return 1
	fi
	if ! command -v rustc >/dev/null 2>&1 || ! rustc --version >/dev/null 2>&1; then
		log_err "rustc is still missing or non-functional after installation"
		return 1
	fi

	log_info "Verified $(cargo --version) and $(rustc --version)"
}

main "$@"
