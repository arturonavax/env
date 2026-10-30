#!/usr/bin/env bash
# ==============================================================================
# utils/backup_config.sh
# Creates a snapshot of user dotfiles into ~/.arturonavax-env-backups/
# Part of arturonavax/env development environment.
# ==============================================================================

# Anchoring
while [[ ! -d ./.git/ && ! -d ./files/ && ! -d ./src/ ]]; do
	if [[ "$PWD" == "/" ]]; then
		echo "[Backup Error]: Repository root not found." >&2
		exit 1
	fi
	if ! cd ..; then
		echo "[Backup Error]: Failed to change directory." >&2
		exit 1
	fi
done

source ./src/lib/strict.sh
source ./src/lib/colors.sh
source ./src/lib/logging.sh
source ./src/lib/platform.sh
source ./src/lib/backup.sh

backup_create
