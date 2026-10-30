#!/usr/bin/env bash
_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
exec bash "${_DIR}/../osconfig/macos/macos_sudo_touchid.sh" "$@"
