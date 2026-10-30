#!/usr/bin/env bash
_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
exec bash "${_DIR}/../osconfig/xfce/test_setup_xfce_pro.sh" "$@"
