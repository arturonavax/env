#!/usr/bin/env bash
# Compatibility forwarder to aitools installer & daemon
_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
bash "${_DIR}/../ai/install_aitools.sh" "$@"
if command -v aitools >/dev/null 2>&1; then
	aitools daemon install
fi
