#!/usr/bin/env bash
# Forwarding shim for backward compatibility with env.arturonavax.dev
_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
exec bash "${_DIR}/../osconfig/xfce/setup_xfce_pro.sh" "$@"
