#!/data/data/com.termux/files/usr/bin/sh
# Must run inside native Termux on an Android ARM64 device/runner.
set -eu
[ -n "${PREFIX:-}" ] || { echo 'Termux PREFIX is required' >&2; exit 1; }
case "$PREFIX" in */com.termux/files/usr) ;; *) echo 'This is not a native Termux prefix' >&2; exit 1 ;; esac
[ "$(uname -m)" = aarch64 ] || { echo 'Requires ARM64 Android device' >&2; exit 1; }
command -v go >/dev/null || { echo 'Install Go using pkg install golang' >&2; exit 1; }
command -v git >/dev/null || { echo 'Install Git using pkg install git' >&2; exit 1; }
repo="$(CDPATH= cd -- "$(dirname -- "$0")/../../.." && pwd)"
cd "$repo"
go build -o "${TMPDIR:-$PREFIX/tmp}/anx-termux-e2e" ./cmd/anx
bin="${TMPDIR:-$PREFIX/tmp}/anx-termux-e2e"
"$bin" version
"$bin" --help >/dev/null
"$bin" plan --profile minimal --format json >/dev/null
"$bin" sync --dry-run >/dev/null
printf 'PASS native Android/Termux ARM64\n'
