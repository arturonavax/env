#!/usr/bin/env bash
_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
exec bash "${_DIR}/../hacks/install_quota_daemon_xfce.sh" "$@"
