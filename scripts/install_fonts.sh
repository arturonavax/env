#!/usr/bin/env bash
# install_fonts.sh - Idempotent Monaspace Nerd Font (Mono) installer for Linux and macOS.
# Installs user-level fonts without requiring sudo:
#   - Linux: ${XDG_DATA_HOME:-~/.local/share}/fonts/MonaspaceNerdFont
#   - macOS: ~/Library/Fonts/MonaspaceNerdFont

set -euo pipefail

FONT_VERSION="${NERD_FONTS_VERSION:-v3.5.1}"
DOWNLOAD_URL="${ANX_NERD_FONTS_URL:-https://github.com/ryanoasis/nerd-fonts/releases/download/${FONT_VERSION}/Monaspace.zip}"
READY_MARKER="${HOME}/.local/share/anx/fonts-monaspace.ready"

TMP_DIR=""
cleanup() {
	if [[ -n "$TMP_DIR" && -d "$TMP_DIR" ]]; then
		rm -rf "$TMP_DIR"
	fi
}
trap cleanup EXIT INT TERM

log_info() {
	printf '[fonts] %s\n' "$*"
}

log_err() {
	printf '[fonts] ERROR: %s\n' "$*" >&2
}

extract_zip() {
	local archive="$1"
	local dest="$2"
	if command -v unzip >/dev/null 2>&1; then
		unzip -q -o "$archive" -d "$dest"
	elif command -v ditto >/dev/null 2>&1; then
		ditto -x -k "$archive" "$dest"
	elif command -v python3 >/dev/null 2>&1; then
		python3 -m zipfile -e "$archive" "$dest"
	else
		log_err "No zip extractor found (install unzip)."
		return 1
	fi
}

main() {
	if ! command -v curl >/dev/null 2>&1; then
		log_err "curl is required to download Monaspace Nerd Fonts."
		return 1
	fi

	local os_name dest_dir
	os_name="$(uname -s)"
	case "$os_name" in
		Linux)
			dest_dir="${XDG_DATA_HOME:-${HOME}/.local/share}/fonts/MonaspaceNerdFont"
			;;
		Darwin)
			dest_dir="${HOME}/Library/Fonts/MonaspaceNerdFont"
			;;
		*)
			log_err "Unsupported OS: ${os_name} (use scripts/install_fonts.ps1 on Windows)."
			return 1
			;;
	esac

	TMP_DIR="$(mktemp -d)"
	local archive="${TMP_DIR}/Monaspace.zip"
	local extract_dir="${TMP_DIR}/extract"
	mkdir -p "$extract_dir"

	log_info "Downloading Monaspace Nerd Fonts (${FONT_VERSION})..."
	curl -fsSL -o "$archive" "$DOWNLOAD_URL"

	log_info "Extracting archive..."
	extract_zip "$archive" "$extract_dir"

	local font_files=()
	if [[ "${ANX_NERD_FONTS_ALL:-0}" != "1" ]]; then
		while IFS= read -r -d '' f; do
			font_files+=("$f")
		done < <(find "$extract_dir" -type f \( -iname '*NerdFontMono*.otf' -o -iname '*NerdFontMono*.ttf' \) -print0)
	fi

	if [[ ${#font_files[@]} -eq 0 ]]; then
		while IFS= read -r -d '' f; do
			font_files+=("$f")
		done < <(find "$extract_dir" -type f \( -iname '*.otf' -o -iname '*.ttf' \) -print0)
	fi

	if [[ ${#font_files[@]} -eq 0 ]]; then
		log_err "No .otf or .ttf font files found in ${DOWNLOAD_URL}."
		return 1
	fi

	mkdir -p "$dest_dir"
	local f
	for f in "${font_files[@]}"; do
		cp -f "$f" "$dest_dir/"
	done

	if [[ "$os_name" == "Linux" ]] && command -v fc-cache >/dev/null 2>&1; then
		log_info "Refreshing fontconfig cache..."
		fc-cache -f "$dest_dir" >/dev/null 2>&1 || true
	fi

	mkdir -p "$(dirname "$READY_MARKER")"
	printf 'version=%s\ncount=%d\ndir=%s\n' "$FONT_VERSION" "${#font_files[@]}" "$dest_dir" > "$READY_MARKER"
	log_info "Installed ${#font_files[@]} Monaspace Nerd Font Mono files into ${dest_dir}."
}

main "$@"
