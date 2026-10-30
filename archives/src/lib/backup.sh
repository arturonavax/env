#!/usr/bin/env bash
# ==============================================================================
# src/lib/backup.sh
# Production Cross-Platform Configuration Backup, Rotation and Rollback Engine.
# Works uniformly on GNU Linux and BSD/macOS with zero GNU-specific flags.
# Part of arturonavax/env development environment.
# ==============================================================================

# Prevent double sourcing
[[ -n "${_BACKUP_SH_LOADED:-}" ]] && return 0
_BACKUP_SH_LOADED=1

_LIB_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
if [[ -f "${_LIB_DIR}/colors.sh" ]]; then
	# shellcheck source=/dev/null
	source "${_LIB_DIR}/colors.sh"
fi
if [[ -f "${_LIB_DIR}/logging.sh" ]]; then
	# shellcheck source=/dev/null
	source "${_LIB_DIR}/logging.sh"
fi
if [[ -f "${_LIB_DIR}/platform.sh" ]]; then
	# shellcheck source=/dev/null
	source "${_LIB_DIR}/platform.sh"
fi

BACKUP_STORAGE_DIR="${BACKUP_STORAGE_DIR:-$HOME/.arturonavax-env-backups}"
BACKUP_MAX_SIZE_KB="${BACKUP_MAX_SIZE_KB:-$((50 * 1024))}" # 50 Megabytes in Kilobytes

# Portable Directory Size in Kilobytes
_get_dir_size_kb() {
	local dir_path="$1"
	if [[ ! -d "$dir_path" ]]; then
		echo 0
		return 0
	fi
	# du -k is standard across GNU coreutils and BSD/macOS
	du -k -s "$dir_path" 2>/dev/null | awk '{print $1}' || echo 0
}

# Purge oldest backups when total size exceeds quota
_rotate_backups() {
	local total_kb
	total_kb=$(_get_dir_size_kb "$BACKUP_STORAGE_DIR")

	while ((total_kb > BACKUP_MAX_SIZE_KB)); do
		# Because backup folders are named YYYY-MM-DD_HH-MM-SS, alphabetical sort is chronological
		local oldest
		oldest=$(find "$BACKUP_STORAGE_DIR" -mindepth 1 -maxdepth 1 -type d | sort | head -n 1)

		if [[ -z "$oldest" || ! -d "$oldest" ]]; then
			break
		fi

		log_warn "Backup quota exceeded ($((total_kb / 1024))MB > $((BACKUP_MAX_SIZE_KB / 1024))MB). Purging oldest snapshot: $(basename "$oldest")"
		rm -rf "$oldest"

		total_kb=$(_get_dir_size_kb "$BACKUP_STORAGE_DIR")
	done
}

# Create a complete snapshot
backup_create() {
	local timestamp
	timestamp="$(date +'%Y-%m-%d_%H-%M-%S')"
	local target_dir="${BACKUP_STORAGE_DIR}/${timestamp}"

	mkdir -p "$target_dir"
	log_step "Creating configuration snapshot in ${BOLD_WHITE}${target_dir}${RESET}..."

	# Safe copy helper
	_copy_file() {
		local src="$1"
		local dest_dir="$2"
		if [[ -f "$src" ]]; then
			mkdir -p "$dest_dir"
			cp -p "$src" "$dest_dir/" 2>/dev/null || true
		fi
	}

	_copy_dir() {
		local src="$1"
		local dest_dir="$2"
		if [[ -d "$src" ]]; then
			mkdir -p "$dest_dir"
			cp -Rp "$src" "$dest_dir/" 2>/dev/null || true
		fi
	}

	# 1. Terminal & Multiplexers
	_copy_dir "$HOME/.config/ghostty" "$target_dir/config"
	_copy_file "$HOME/.config/herdr/config.toml" "$target_dir/config/herdr"
	_copy_file "$HOME/.config/atuin/config.toml" "$target_dir/config/atuin"
	_copy_file "$HOME/.config/starship.toml" "$target_dir/config"

	# 2. Shell Configuration
	_copy_file "$HOME/.zshrc" "$target_dir"
	_copy_file "$HOME/.base.zsh" "$target_dir"
	_copy_file "$HOME/.tools.sh" "$target_dir"
	_copy_file "$HOME/.lscolors.sh" "$target_dir"
	_copy_file "$HOME/.bashrc" "$target_dir"

	# 3. Editor (Neovim)
	_copy_dir "$HOME/.config/nvim" "$target_dir/config"

	# 4. GUI Editors (VSCode / Cursor)
	local os_type
	os_type="$(detect_os)"
	if [[ "$os_type" == "darwin" ]]; then
		_copy_file "$HOME/Library/Application Support/Code/User/settings.json" "$target_dir/vscode"
		_copy_file "$HOME/Library/Application Support/Code/User/keybindings.json" "$target_dir/vscode"
		_copy_file "$HOME/Library/Application Support/Cursor/User/settings.json" "$target_dir/cursor"
		_copy_file "$HOME/Library/Application Support/Cursor/User/keybindings.json" "$target_dir/cursor"
		_copy_file "$HOME/.amethyst.yml" "$target_dir"
		_copy_file "$HOME/.config/karabiner/karabiner.json" "$target_dir/config/karabiner"
	else
		_copy_file "$HOME/.config/Code/User/settings.json" "$target_dir/vscode"
		_copy_file "$HOME/.config/Code/User/keybindings.json" "$target_dir/vscode"
		_copy_file "$HOME/.config/Cursor/User/settings.json" "$target_dir/cursor"
		_copy_file "$HOME/.config/Cursor/User/keybindings.json" "$target_dir/cursor"
	fi

	# 5. Window Switcher & Keyd
	_copy_file "$HOME/.config/rcmd/rcmd.conf" "$target_dir/config/rcmd"
	_copy_file "$HOME/.config/rcmd.conf" "$target_dir/config"
	_copy_file "$HOME/.config/keyd/default.conf" "$target_dir/config/keyd"

	# 6. AI Tooling & Assistants
	_copy_file "$HOME/.config/rtk/config.toml" "$target_dir/config/rtk"
	_copy_file "$HOME/.gemini/antigravity-cli/config.json" "$target_dir/ai"

	# 7. Code Quality & Formatters
	_copy_file "$HOME/.golangci.yml" "$target_dir"
	_copy_file "$HOME/.eslintrc.json" "$target_dir"
	_copy_file "$HOME/.prettierrc.json" "$target_dir"
	_copy_file "$HOME/.stylelintrc.json" "$target_dir"
	_copy_file "$HOME/.sql-formatter.json" "$target_dir"

	# 8. Generate Manifest
	local manifest_file="${target_dir}/manifest.txt"
	{
		echo "# arturonavax/env Backup Manifest"
		echo "Timestamp: $timestamp"
		echo "Hostname: $(hostname 2>/dev/null || echo 'localhost')"
		echo "OS: $(detect_os) ($(detect_distro))"
		echo "----------------------------------------------------"
		(cd "$target_dir" && find . -type f ! -name "manifest.txt" | sort)
	} >"$manifest_file"

	# Enforce size rotation
	_rotate_backups

	log_success "Snapshot created: ${BOLD_WHITE}${timestamp}${RESET} ($(_get_dir_size_kb "$target_dir") KB)"
	echo "$target_dir"
}

# List all existing backups
backup_list() {
	if [[ ! -d "$BACKUP_STORAGE_DIR" ]]; then
		log_info "No backup directory exists yet (${BACKUP_STORAGE_DIR})."
		return 0
	fi

	local count
	count=$(find "$BACKUP_STORAGE_DIR" -mindepth 1 -maxdepth 1 -type d | wc -l)
	if ((count == 0)); then
		log_info "No backups found in ${BACKUP_STORAGE_DIR}."
		return 0
	fi

	log_box "Available Configuration Snapshots" \
		"Storage path: ${BACKUP_STORAGE_DIR}" \
		"Total size:   $(( $(_get_dir_size_kb "$BACKUP_STORAGE_DIR") / 1024 )) MB / $(( BACKUP_MAX_SIZE_KB / 1024 )) MB limit"

	printf '%s%-22s  %-10s  %s%s\n' "$BOLD_WHITE" "SNAPSHOT" "SIZE" "PATH" "$RESET"
	printf '%s----------------------------------------------------------------------%s\n' "$DIM" "$RESET"

	while IFS= read -r snap_dir; do
		if [[ -d "$snap_dir" ]]; then
			local name size_kb
			name="$(basename "$snap_dir")"
			size_kb="$(_get_dir_size_kb "$snap_dir")"
			printf '%-22s  %-10s  %s\n' "$name" "${size_kb} KB" "$snap_dir"
		fi
	done < <(find "$BACKUP_STORAGE_DIR" -mindepth 1 -maxdepth 1 -type d | sort -r)
}

# Restore from a backup snapshot
backup_restore() {
	local snapshot_target="$1"
	local snapshot_path=""

	if [[ -d "$snapshot_target" ]]; then
		snapshot_path="$snapshot_target"
	elif [[ -d "${BACKUP_STORAGE_DIR}/${snapshot_target}" ]]; then
		snapshot_path="${BACKUP_STORAGE_DIR}/${snapshot_target}"
	else
		log_fatal "Snapshot '${snapshot_target}' not found in ${BACKUP_STORAGE_DIR}."
	fi

	log_step "Restoring configuration from snapshot: ${BOLD_WHITE}$(basename "$snapshot_path")${RESET}..."

	# Restore files
	if [[ -d "${snapshot_path}/config" ]]; then
		mkdir -p "$HOME/.config"
		cp -Rp "${snapshot_path}/config/"* "$HOME/.config/" 2>/dev/null || true
	fi

	for dotfile in .zshrc .base.zsh .tools.sh .lscolors.sh .bashrc .golangci.yml .eslintrc.json .prettierrc.json .stylelintrc.json .sql-formatter.json; do
		if [[ -f "${snapshot_path}/${dotfile}" ]]; then
			cp -p "${snapshot_path}/${dotfile}" "$HOME/" 2>/dev/null || true
		fi
	done

	# Restore IDE configs
	if [[ "$(detect_os)" == "darwin" ]]; then
		if [[ -d "${snapshot_path}/vscode" ]]; then
			mkdir -p "$HOME/Library/Application Support/Code/User"
			cp -Rp "${snapshot_path}/vscode/"* "$HOME/Library/Application Support/Code/User/" 2>/dev/null || true
		fi
		if [[ -d "${snapshot_path}/cursor" ]]; then
			mkdir -p "$HOME/Library/Application Support/Cursor/User"
			cp -Rp "${snapshot_path}/cursor/"* "$HOME/Library/Application Support/Cursor/User/" 2>/dev/null || true
		fi
	else
		if [[ -d "${snapshot_path}/vscode" ]]; then
			mkdir -p "$HOME/.config/Code/User"
			cp -Rp "${snapshot_path}/vscode/"* "$HOME/.config/Code/User/" 2>/dev/null || true
		fi
		if [[ -d "${snapshot_path}/cursor" ]]; then
			mkdir -p "$HOME/.config/Cursor/User"
			cp -Rp "${snapshot_path}/cursor/"* "$HOME/.config/Cursor/User/" 2>/dev/null || true
		fi
	fi

	log_success "Configuration restored successfully from $(basename "$snapshot_path")!"
}
