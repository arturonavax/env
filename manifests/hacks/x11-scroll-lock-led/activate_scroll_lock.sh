#!/usr/bin/env bash
# ==============================================================================
# hacks/x11-scroll-lock-led/activate_scroll_lock.sh
# Toggles the X11 Scroll Lock hardware LED indicator for keyboards whose
# backlight is wired to the Scroll Lock state under Linux X11 sessions.
# ==============================================================================
set -euo pipefail

SCRIPT_NAME="$(basename -- "$0")"
readonly SCRIPT_NAME

usage() {
	cat <<EOF
Usage: ${SCRIPT_NAME} [--toggle | --on | --off | --status | --help]

Options:
  --toggle    Toggle the X11 Scroll Lock LED state (default)
  --on        Enable the X11 Scroll Lock LED
  --off       Disable the X11 Scroll Lock LED
  --status    Print current Scroll Lock LED state ("on" or "off")
  -h, --help  Show this help message
EOF
}

get_scroll_lock_state() {
	local query_out state_raw
	query_out="$(xset -q 2>/dev/null || true)"
	state_raw="$(printf '%s\n' "${query_out}" | awk -F'Scroll Lock:' 'NF > 1 {print $2; exit}' | awk '{print $1}')"
	if [[ "${state_raw}" == "on" ]]; then
		printf 'on\n'
	else
		printf 'off\n'
	fi
}

main() {
	local action="toggle"
	if [[ "$#" -gt 0 ]]; then
		case "$1" in
		--toggle)
			action="toggle"
			;;
		--on)
			action="on"
			;;
		--off)
			action="off"
			;;
		--status)
			action="status"
			;;
		-h | --help)
			usage
			exit 0
			;;
		*)
			printf '[%s] Error: unknown option %q\n' "${SCRIPT_NAME}" "$1" >&2
			usage >&2
			exit 1
			;;
		esac
	fi

	if [[ "$(uname -s)" != "Linux" ]]; then
		printf '[%s] Skipping: X11 Scroll Lock LED hack applies to Linux hosts only.\n' "${SCRIPT_NAME}" >&2
		exit 0
	fi

	if [[ -z "${DISPLAY:-}" ]]; then
		printf '[%s] Skipping: X11 DISPLAY is not set (Wayland or headless session).\n' "${SCRIPT_NAME}" >&2
		exit 0
	fi

	if ! hash xset 2>/dev/null; then
		printf '[%s] Skipping: xset utility is not installed.\n' "${SCRIPT_NAME}" >&2
		exit 0
	fi

	case "${action}" in
	status)
		get_scroll_lock_state
		;;
	on)
		xset led named "Scroll Lock"
		;;
	off)
		xset -led named "Scroll Lock"
		;;
	toggle)
		if [[ "$(get_scroll_lock_state)" == "off" ]]; then
			xset led named "Scroll Lock"
		else
			xset -led named "Scroll Lock"
		fi
		;;
	esac
}

main "$@"
