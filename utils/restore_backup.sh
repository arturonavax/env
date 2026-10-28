#!/usr/bin/env bash
# ==============================================================================
# utils/restore_backup.sh
# Restores configurations from a selected snapshot in ~/.arturonavax-env-backups/
# Part of arturonavax/env development environment.
# ==============================================================================

while [[ ! -d ./.git/ && ! -d ./files/ && ! -d ./src/ ]]; do
	if [[ "$PWD" == "/" ]]; then
		echo "[Restore Error]: Repository root not found." >&2
		exit 1
	fi
	if ! cd ..; then
		echo "[Restore Error]: Failed to change directory." >&2
		exit 1
	fi
done

source ./src/lib/strict.sh
source ./src/lib/colors.sh
source ./src/lib/logging.sh
source ./src/lib/platform.sh
source ./src/lib/backup.sh

usage() {
	cat <<EOF
Usage: ./utils/restore_backup.sh [OPTIONS] [SNAPSHOT_TIMESTAMP]

Options:
    -l, --list        List available snapshots
    -latest           Restore the most recent snapshot automatically
    -h, --help        Show this help message

Examples:
    ./utils/restore_backup.sh -l
    ./utils/restore_backup.sh -latest
    ./utils/restore_backup.sh 2026-10-01_20-34
EOF
}

target="${1:-}"

case "$target" in
-l | --list | list)
	backup_list
	exit 0
	;;
-latest | --latest | latest)
	latest_snap=$(find "$BACKUP_STORAGE_DIR" -mindepth 1 -maxdepth 1 -type d | sort | tail -n 1)
	if [[ -z "$latest_snap" ]]; then
		log_fatal "No snapshots available to restore."
	fi
	log_info "Targeting most recent snapshot: $(basename "$latest_snap")"
	backup_restore "$latest_snap"
	exit 0
	;;
-h | --help | help)
	usage
	exit 0
	;;
"")
	backup_list
	echo
	read -r -p "Enter snapshot name or path to restore (or Ctrl+C to cancel): " user_input
	if [[ -n "$user_input" ]]; then
		backup_restore "$user_input"
	fi
	;;
*)
	backup_restore "$target"
	;;
esac
