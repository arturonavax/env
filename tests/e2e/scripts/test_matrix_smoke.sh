#!/bin/sh
# Portable, no-network, non-destructive smoke test in a real distro runtime.
set -eu
mkdir -p "$HOME" "$XDG_CONFIG_HOME" "$XDG_STATE_HOME" "$XDG_CACHE_HOME"
if [ ! -r /etc/os-release ]; then echo "Missing /etc/os-release" >&2; exit 1; fi
. /etc/os-release
printf 'E2E distro=%s version=%s arch=%s\n' "${ID:-unknown}" "${VERSION_ID:-unknown}" "$(uname -m)"
case "${EXPECTED_DISTRO:-}" in
  debian|ubuntu|fedora|alpine|arch) [ "${ID:-}" = "$EXPECTED_DISTRO" ] || { echo "Unexpected distro: ${ID:-unknown}, expected $EXPECTED_DISTRO" >&2; exit 1; } ;;
  *) echo "Missing/unknown EXPECTED_DISTRO" >&2; exit 1 ;;
esac
anx version
anx --help >/dev/null
anx plan --profile minimal --format json >/tmp/anx-plan.json
[ -s /tmp/anx-plan.json ] || { echo "Empty plan" >&2; exit 1; }
anx status --format json >/tmp/anx-status.json
[ -s /tmp/anx-status.json ] || { echo "Empty status" >&2; exit 1; }
anx sync --dry-run --format json >/tmp/anx-dry-run.json
[ -s /tmp/anx-dry-run.json ] || { echo "Empty dry-run" >&2; exit 1; }
printf 'PASS distro=%s arch=%s\n' "$ID" "$(uname -m)"
