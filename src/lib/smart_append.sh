#!/usr/bin/env bash
# ==============================================================================
# src/lib/smart_append.sh
# Non-destructive smart file preservation and idempotent marker-block management.
# Preserves user's local files, aliases, functions, and configs (smart-append-first).
# Part of arturonavax/env development environment.
# ==============================================================================

# Prevent double sourcing
[[ -n "${_SMART_APPEND_SH_LOADED:-}" ]] && return 0
_SMART_APPEND_SH_LOADED=1

_LIB_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
if [[ -f "${_LIB_DIR}/colors.sh" ]]; then
	# shellcheck source=/dev/null
	source "${_LIB_DIR}/colors.sh"
fi
if [[ -f "${_LIB_DIR}/logging.sh" ]]; then
	# shellcheck source=/dev/null
	source "${_LIB_DIR}/logging.sh"
fi

# Insert or replace a managed marker block inside a file
# Preserves ALL user content before and after the block.
# Usage:
#   smart_append_block <target_file> <block_id> <content_string_or_source_file>
smart_append_block() {
	local target_file="$1"
	local block_id="$2"
	local content_arg="$3"

	local start_marker="# >>> arturonavax/env:${block_id} >>>"
	local end_marker="# <<< arturonavax/env:${block_id} <<<"

	local block_body=""
	if [[ -f "$content_arg" ]]; then
		block_body="$(cat "$content_arg")"
	else
		block_body="$content_arg"
	fi

	mkdir -p "$(dirname "$target_file")"

	# If target file does not exist, create it with the block
	if [[ ! -f "$target_file" ]]; then
		{
			echo "$start_marker"
			echo "$block_body"
			echo "$end_marker"
		} >"$target_file"
		log_debug "Created ${target_file} with block '${block_id}'."
		return 0
	fi

	# Check if marker already exists
	if grep -qF "$start_marker" "$target_file" 2>/dev/null; then
		# Replace only the block between start_marker and end_marker
		local tmp_output
		tmp_output="$(mktemp "${TMPDIR:-/tmp}/smart_block.XXXXXX")"

		awk -v start="$start_marker" -v end="$end_marker" -v repl="$block_body" '
			$0 == start {
				print start
				print repl
				skip = 1
				next
			}
			$0 == end {
				print end
				skip = 0
				next
			}
			!skip { print }
		' "$target_file" >"$tmp_output"

		cat "$tmp_output" >"$target_file"
		rm -f "$tmp_output"
		log_debug "Updated block '${block_id}' in ${target_file}."
	else
		# Append to the end of user file, ensuring preceding newline
		{
			echo ""
			echo "$start_marker"
			echo "$block_body"
			echo "$end_marker"
		} >>"$target_file"
		log_debug "Appended block '${block_id}' to ${target_file} (user content preserved)."
	fi
}

# Preserve an existing user file; only install source file if target does not exist or is empty
# Usage:
#   smart_preserve_copy <source_file> <target_file>
smart_preserve_copy() {
	local source_file="$1"
	local target_file="$2"

	mkdir -p "$(dirname "$target_file")"

	if [[ -s "$target_file" ]]; then
		log_info "Preserving user file: ${BOLD_WHITE}${target_file}${RESET} (existing configuration kept)."
		return 0
	fi

	cp -p "$source_file" "$target_file"
	log_debug "Copied ${source_file} -> ${target_file}"
}

# Ensure a source line exists in a shell config file idempotently without duplicate lines
# Usage:
#   smart_ensure_source <shell_rc_file> <path_to_source>
smart_ensure_source() {
	local shell_rc="$1"
	local source_path="$2"
	local block_id
	block_id="$(basename "$source_path" | tr -d '.')"

	local snippet="[[ -f \"${source_path}\" ]] && source \"${source_path}\""
	smart_append_block "$shell_rc" "$block_id" "$snippet"
}
