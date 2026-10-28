#!/usr/bin/env bash
# ==============================================================================
# src/lib/download.sh
# Resilient, retry-capable downloader with checksum verification and extraction.
# Part of arturonavax/env development environment.
# ==============================================================================

# Prevent double sourcing
[[ -n "${_DOWNLOAD_SH_LOADED:-}" ]] && return 0
_DOWNLOAD_SH_LOADED=1

_LIB_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
if [[ -f "${_LIB_DIR}/logging.sh" ]]; then
	# shellcheck source=/dev/null
	source "${_LIB_DIR}/logging.sh"
fi

# Download a file from URL to destination path with retries
download_file() {
	local url="$1"
	local output_path="$2"
	local expected_sha256="${3:-}"

	mkdir -p "$(dirname "$output_path")"

	log_debug "Downloading ${url} -> ${output_path}..."

	local downloaded=0
	if command -v curl >/dev/null 2>&1; then
		if curl --proto '=https' --tlsv1.2 -fsSL --retry 3 --retry-delay 2 "$url" -o "$output_path"; then
			downloaded=1
		fi
	elif command -v wget >/dev/null 2>&1; then
		if wget -q --tries=3 --waitretry=2 "$url" -O "$output_path"; then
			downloaded=1
		fi
	else
		log_fatal "Neither curl nor wget is available to perform downloads."
	fi

	if ((downloaded == 0)); then
		log_error "Failed to download: $url"
		return 1
	fi

	# Checksum verification if hash provided
	if [[ -n "$expected_sha256" ]]; then
		log_debug "Verifying SHA-256 checksum..."
		local actual_sha256=""
		if command -v sha256sum >/dev/null 2>&1; then
			actual_sha256="$(sha256sum "$output_path" | awk '{print $1}')"
		elif command -v shasum >/dev/null 2>&1; then
			actual_sha256="$(shasum -a 256 "$output_path" | awk '{print $1}')"
		fi

		if [[ -n "$actual_sha256" && "$actual_sha256" != "$expected_sha256" ]]; then
			log_error "Checksum verification mismatch for ${output_path}!"
			log_error "Expected: ${expected_sha256}"
			log_error "Actual:   ${actual_sha256}"
			rm -f "$output_path"
			return 1
		fi
	fi

	return 0
}

# Download an archive and extract to target directory safely
download_and_extract() {
	local url="$1"
	local dest_dir="$2"
	local strip_components="${3:-0}"

	mkdir -p "$dest_dir"
	local tmp_archive
	tmp_archive="$(mktemp "${TMPDIR:-/tmp}/archive.XXXXXX")"

	if ! download_file "$url" "$tmp_archive"; then
		rm -f "$tmp_archive"
		return 1
	fi

	log_debug "Extracting ${url} to ${dest_dir}..."

	case "$url" in
	*.tar.gz | *.tgz)
		tar -xzf "$tmp_archive" -C "$dest_dir" --strip-components="$strip_components"
		;;
	*.tar.xz)
		tar -xJf "$tmp_archive" -C "$dest_dir" --strip-components="$strip_components"
		;;
	*.tar.bz2)
		tar -xjf "$tmp_archive" -C "$dest_dir" --strip-components="$strip_components"
		;;
	*.zip)
		unzip -q -o "$tmp_archive" -d "$dest_dir"
		;;
	*)
		log_fatal "Unsupported archive format for ${url}"
		;;
	esac

	rm -f "$tmp_archive"
	return 0
}

# Install a standalone executable binary directly into ~/.local/bin/
install_binary() {
	local src_binary="$1"
	local binary_name="$2"
	local target_user_bin="$HOME/.local/bin"
	local target_sys_bin="/usr/local/bin"

	mkdir -p "$target_user_bin"
	chmod +x "$src_binary"
	cp -f "$src_binary" "${target_user_bin}/${binary_name}"

	# Try to symlink into /usr/local/bin for all users if sudo privileges available
	if [[ -d "$target_sys_bin" ]] && [[ -w "$target_sys_bin" ]]; then
		ln -sf "${target_user_bin}/${binary_name}" "${target_sys_bin}/${binary_name}" 2>/dev/null || true
	elif command -v sudo >/dev/null 2>&1 && sudo -n true 2>/dev/null; then
		sudo ln -sf "${target_user_bin}/${binary_name}" "${target_sys_bin}/${binary_name}" 2>/dev/null || true
	fi

	log_success "Binary installed: ${BOLD_WHITE}${binary_name}${RESET} -> ${target_user_bin}/${binary_name}"
}

# Download multiple files in parallel using concurrency task pool
# Each argument is: "URL:::DEST_PATH" or "URL:::DEST_PATH:::EXPECTED_SHA256"
parallel_download_files() {
	local pool_name="$1"
	shift

	if [[ -f "${_LIB_DIR}/concurrency.sh" ]]; then
		# shellcheck source=/dev/null
		source "${_LIB_DIR}/concurrency.sh"
	fi

	local -a tasks=()
	for item in "$@"; do
		local url="${item%%:::*}"
		local rest="${item#*:::}"
		local dest="${rest%%:::*}"
		local sha=""
		if [[ "$rest" == *":::"* ]]; then
			sha="${rest#*:::}"
		fi

		local filename
		filename="$(basename "$dest")"
		local task_cmd
		if [[ -n "$sha" ]]; then
			task_cmd="source '${_LIB_DIR}/download.sh' && download_file '${url}' '${dest}' '${sha}'"
		else
			task_cmd="source '${_LIB_DIR}/download.sh' && download_file '${url}' '${dest}'"
		fi
		tasks+=("Download ${filename}:::${task_cmd}")
	done

	run_parallel_task_pool "$pool_name" "${tasks[@]}"
}
