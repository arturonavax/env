#!/usr/bin/env bash
# ==============================================================================
# src/lib/ui_box.sh
# Dynamic, Terminal-Aware Box Drawing, Panel & Banner Normalization Engine.
# Part of arturonavax/env development environment.
# ==============================================================================

# Prevent double sourcing
[[ -n "${_UI_BOX_SH_LOADED:-}" ]] && return 0
_UI_BOX_SH_LOADED=1

# Resolve script directory and source colors if not already loaded
_UI_BOX_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=/dev/null
source "${_UI_BOX_DIR}/colors.sh" 2>/dev/null || true

# ------------------------------------------------------------------------------
# Terminal Width Detection & Clamping
# ------------------------------------------------------------------------------
ui_box_width() {
	local default_w="${1:-70}"
	local min_w="${2:-40}"
	local max_w="${3:-100}"
	local cols=0

	if [[ -t 1 ]] && command -v tput >/dev/null 2>&1; then
		cols=$(tput cols 2>/dev/null || echo 0)
	elif [[ -n "${COLUMNS:-}" ]]; then
		cols="$COLUMNS"
	fi

	if ! [[ "$cols" =~ ^[0-9]+$ ]] || ((cols <= 0)); then
		cols="$default_w"
	fi

	# Allow a 2-column margin for clean terminal wrapping
	if ((cols > 2)); then
		cols=$((cols - 2))
	fi

	if ((cols < min_w)); then
		cols="$min_w"
	elif ((cols > max_w)); then
		cols="$max_w"
	fi

	echo "$cols"
}

# ------------------------------------------------------------------------------
# String Metrics (Strips ANSI codes for true terminal visual width)
# ------------------------------------------------------------------------------
ui_strip_ansi() {
	local input="$1"
	# Remove ANSI escape sequences
	printf '%s' "$input" | sed -E 's/\x1B\[[0-9;]*[a-zA-Z]//g' | tr -d '\r'
}

ui_visible_length() {
	local clean
	clean="$(ui_strip_ansi "$1")"
	printf '%d' "${#clean}"
}

ui_repeat_char() {
	local char="${1:-─}"
	local count="${2:-0}"
	if ((count <= 0)); then
		return 0
	fi
	local spaces
	printf -v spaces "%${count}s" ""
	printf '%s' "${spaces// /$char}"
}

# ------------------------------------------------------------------------------
# Box Border Glyphs
# ------------------------------------------------------------------------------
_ui_box_chars() {
	local style="${1:-rounded}"
	case "$style" in
		double)
			UI_CHAR_TOP_L="╔" UI_CHAR_TOP_R="╗"
			UI_CHAR_BOT_L="╚" UI_CHAR_BOT_R="╝"
			UI_CHAR_H="═" UI_CHAR_V="║"
			UI_CHAR_MID_L="╠" UI_CHAR_MID_R="╣"
			;;
		single)
			UI_CHAR_TOP_L="┌" UI_CHAR_TOP_R="┐"
			UI_CHAR_BOT_L="└" UI_CHAR_BOT_R="┘"
			UI_CHAR_H="─" UI_CHAR_V="│"
			UI_CHAR_MID_L="├" UI_CHAR_MID_R="┤"
			;;
		ascii)
			UI_CHAR_TOP_L="+" UI_CHAR_TOP_R="+"
			UI_CHAR_BOT_L="+" UI_CHAR_BOT_R="+"
			UI_CHAR_H="-" UI_CHAR_V="|"
			UI_CHAR_MID_L="+" UI_CHAR_MID_R="+"
			;;
		rounded|*)
			UI_CHAR_TOP_L="╭" UI_CHAR_TOP_R="╮"
			UI_CHAR_BOT_L="╰" UI_CHAR_BOT_R="╯"
			UI_CHAR_H="─" UI_CHAR_V="│"
			UI_CHAR_MID_L="├" UI_CHAR_MID_R="┤"
			;;
	esac
}

# ------------------------------------------------------------------------------
# Border & Line Primitives
# ------------------------------------------------------------------------------
ui_box_top() {
	local width="${1:-70}"
	local style="${2:-rounded}"
	local color="${3:-${BOLD_BLUE:-}}"
	_ui_box_chars "$style"

	local inner_len=$((width - 2))
	local line
	line="$(ui_repeat_char "$UI_CHAR_H" "$inner_len")"
	printf '%s%s%s%s%s\n' "$color" "$UI_CHAR_TOP_L" "$line" "$UI_CHAR_TOP_R" "${RESET:-}"
}

ui_box_mid() {
	local width="${1:-70}"
	local style="${2:-rounded}"
	local color="${3:-${BOLD_BLUE:-}}"
	_ui_box_chars "$style"

	local inner_len=$((width - 2))
	local line
	line="$(ui_repeat_char "$UI_CHAR_H" "$inner_len")"
	printf '%s%s%s%s%s\n' "$color" "$UI_CHAR_MID_L" "$line" "$UI_CHAR_MID_R" "${RESET:-}"
}

ui_box_bottom() {
	local width="${1:-70}"
	local style="${2:-rounded}"
	local color="${3:-${BOLD_BLUE:-}}"
	_ui_box_chars "$style"

	local inner_len=$((width - 2))
	local line
	line="$(ui_repeat_char "$UI_CHAR_H" "$inner_len")"
	printf '%s%s%s%s%s\n' "$color" "$UI_CHAR_BOT_L" "$line" "$UI_CHAR_BOT_R" "${RESET:-}"
}

ui_box_line() {
	local text="$1"
	local width="${2:-70}"
	local style="${3:-rounded}"
	local border_color="${4:-${BOLD_BLUE:-}}"
	local text_color="${5:-${RESET:-}}"
	local align="${6:-left}"
	_ui_box_chars "$style"

	local inner_width=$((width - 4))
	local vis_len
	vis_len="$(ui_visible_length "$text")"

	local pad_left=0
	local pad_right=0

	if ((vis_len >= inner_width)); then
		pad_left=0
		pad_right=0
	elif [[ "$align" == "center" ]]; then
		local total_pad=$((inner_width - vis_len))
		pad_left=$((total_pad / 2))
		pad_right=$((total_pad - pad_left))
	elif [[ "$align" == "right" ]]; then
		pad_left=$((inner_width - vis_len))
		pad_right=0
	else
		pad_left=0
		pad_right=$((inner_width - vis_len))
	fi

	local spaces_l=""
	local spaces_r=""
	((pad_left > 0)) && spaces_l="$(printf "%${pad_left}s" "")"
	((pad_right > 0)) && spaces_r="$(printf "%${pad_right}s" "")"

	printf '%s%s%s  %s%s%s%s  %s%s%s\n' \
		"$border_color" "$UI_CHAR_V" "${RESET:-}" \
		"$spaces_l" "$text_color" "$text" "$spaces_r" \
		"$border_color" "$UI_CHAR_V" "${RESET:-}"
}

ui_box_separator() {
	local width="${1:-70}"
	local char="${2:-─}"
	local color="${3:-${DIM:-}}"
	local line
	line="$(ui_repeat_char "$char" "$width")"
	printf '%s%s%s\n' "$color" "$line" "${RESET:-}"
}

# ------------------------------------------------------------------------------
# High-Level Dynamic Box & Banner Renderers
# ------------------------------------------------------------------------------
ui_box_banner() {
	local title="$1"
	local subtitle="${2:-}"
	local color="${3:-${BOLD_BLUE:-}}"
	local width="${4:-}"
	local style="${5:-rounded}"

	if [[ -z "$width" ]]; then
		width="$(ui_box_width 70)"
	fi

	echo
	ui_box_top "$width" "$style" "$color"
	ui_box_line "$title" "$width" "$style" "$color" "${BOLD_WHITE:-}" "left"
	if [[ -n "$subtitle" ]]; then
		ui_box_mid "$width" "$style" "$color"
		ui_box_line "$subtitle" "$width" "$style" "$color" "${RESET:-}" "left"
	fi
	ui_box_bottom "$width" "$style" "$color"
	echo
}
