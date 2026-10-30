#!/usr/bin/env bash
_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
exec bash "${_DIR}/../osconfig/xfce/reset_base_setup_xfce_pro.sh" "$@"
