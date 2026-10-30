#!/usr/bin/env bash
# ==============================================================================
# src/requirements/ai.sh
# Verifies prerequisites for AI tooling, MCP ecosystem, and RTK.
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

for cmd in git curl; do
	if ! command -v "$cmd" >/dev/null 2>&1; then
		log_fatal "Command '$cmd' is required for AI ecosystem installation."
	fi
done

if ! command -v node >/dev/null 2>&1; then
	log_warn "Node.js is recommended for Model Context Protocol (MCP) servers."
fi

if ! command -v python3 >/dev/null 2>&1; then
	log_warn "Python3 is recommended for AI harnesses and orchestrators."
fi
