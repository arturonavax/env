#!/usr/bin/env bash
# ==============================================================================
# src/requirements/requirements.sh
# Verifies pre-flight commands and package managers for requirements installation.
# Part of arturonavax/env development environment.
# ==============================================================================

while [[ ! -d ./.git/ && ! -d ./files/ && ! -d ./src/ ]]; do
	if [[ "$PWD" == "/" ]]; then
		echo "Repository root not found" >&2
		exit 1
	fi
	if ! cd ..; then
		echo "An error occurred while doing 'cd ..'" >&2
		exit 1
	fi
done

source ./src/lib/strict.sh
source ./src/lib/platform.sh
source ./src/lib/logging.sh

pkg_mgr="$(detect_pkg_mgr)"
if [[ "$pkg_mgr" == "unknown" ]]; then
	log_fatal "No supported package manager found (apt, dnf, pacman, brew, apk)."
fi

log_debug "Package manager detected: ${pkg_mgr}"
