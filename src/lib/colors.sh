#!/usr/bin/env bash
# ==============================================================================
# src/lib/colors.sh
# Terminal styling, color detection, semantic palettes and legacy compatibility.
# Part of arturonavax/env development environment.
# ==============================================================================

# Prevent double sourcing
[[ -n "${_COLORS_SH_LOADED:-}" ]] && return 0
_COLORS_SH_LOADED=1

# Color capability detection (respects NO_COLOR standard: https://no-color.org)
_HAS_COLOR=0
if [[ -z "${NO_COLOR:-}" ]]; then
	if [[ -t 1 ]] || [[ -n "${CLICOLOR_FORCE:-}" ]]; then
		if command -v tput >/dev/null 2>&1; then
			_NUM_COLORS=$(tput colors 2>/dev/null || echo 0)
			if ((_NUM_COLORS >= 8)); then
				_HAS_COLOR=1
			fi
		elif [[ "${TERM:-}" != "dumb" ]]; then
			_HAS_COLOR=1
		fi
	fi
fi

if ((_HAS_COLOR == 1)); then
	# Standard ANSI Formatting
	export BOLD='\033[1m'
	export DIM='\033[2m'
	export UNDERLINE='\033[4m'
	export RESET='\033[0m'

	# Foreground Colors
	export BLACK='\033[0;30m'
	export RED='\033[0;31m'
	export GREEN='\033[0;32m'
	export YELLOW='\033[0;33m'
	export BLUE='\033[0;34m'
	export PURPLE='\033[0;35m'
	export CYAN='\033[0;36m'
	export WHITE='\033[0;37m'
	export GRAY='\033[0;90m'

	# Bold Foreground Colors
	export BOLD_BLACK='\033[1;30m'
	export BOLD_RED='\033[1;31m'
	export BOLD_GREEN='\033[1;32m'
	export BOLD_YELLOW='\033[1;33m'
	export BOLD_BLUE='\033[1;34m'
	export BOLD_PURPLE='\033[1;35m'
	export BOLD_CYAN='\033[1;36m'
	export BOLD_WHITE='\033[1;37m'

	# Bright Foreground Colors
	export BRIGHT_BLACK='\033[0;90m'
	export BRIGHT_RED='\033[0;91m'
	export BRIGHT_GREEN='\033[0;92m'
	export BRIGHT_YELLOW='\033[0;93m'
	export BRIGHT_BLUE='\033[0;94m'
	export BRIGHT_PURPLE='\033[0;95m'
	export BRIGHT_CYAN='\033[0;96m'
	export BRIGHT_WHITE='\033[0;97m'

	# Background Colors
	export BG_BLACK='\033[40m'
	export BG_RED='\033[41m'
	export BG_GREEN='\033[42m'
	export BG_YELLOW='\033[43m'
	export BG_BLUE='\033[44m'
	export BG_PURPLE='\033[45m'
	export BG_CYAN='\033[46m'
	export BG_WHITE='\033[47m'
else
	export BOLD="" DIM="" UNDERLINE="" RESET=""
	export BLACK="" RED="" GREEN="" YELLOW="" BLUE="" PURPLE="" CYAN="" WHITE="" GRAY=""
	export BOLD_BLACK="" BOLD_RED="" BOLD_GREEN="" BOLD_YELLOW="" BOLD_BLUE="" BOLD_PURPLE="" BOLD_CYAN="" BOLD_WHITE=""
	export BRIGHT_BLACK="" BRIGHT_RED="" BRIGHT_GREEN="" BRIGHT_YELLOW="" BRIGHT_BLUE="" BRIGHT_PURPLE="" BRIGHT_CYAN="" BRIGHT_WHITE=""
	export BG_BLACK="" BG_RED="" BG_GREEN="" BG_YELLOW="" BG_BLUE="" BG_PURPLE="" BG_CYAN="" BG_WHITE=""
fi

# Semantic Status Symbols
export SYM_CHECK="✔"
export SYM_CROSS="✖"
export SYM_WARN="⚠"
export SYM_ARROW="▸"
export SYM_INFO="ℹ"
export SYM_BULLET="•"
export SYM_STAR="★"
export SYM_GEAR="⚙"

# ==============================================================================
# Backward Compatibility Layer for existing repository scripts
# ==============================================================================
export fgcolor_reset="${RESET}"
export fgcolor_black="${BLACK}"
export fgcolor_white="${WHITE}"
export fgcolor_blue="${BLUE}"
export fgcolor_red="${RED}"
export fgcolor_green="${GREEN}"
export fgcolor_yellow="${YELLOW}"
export fgcolor_purple="${PURPLE}"
export fgcolor_cyan="${CYAN}"
export fgcolor_gray="${GRAY}"

export fgcolor_black_bold="${BOLD_BLACK}"
export fgcolor_white_bold="${BOLD_WHITE}"
export fgcolor_blue_bold="${BOLD_BLUE}"
export fgcolor_red_bold="${BOLD_RED}"
export fgcolor_green_bold="${BOLD_GREEN}"
export fgcolor_yellow_bold="${BOLD_YELLOW}"
export fgcolor_purple_bold="${BOLD_PURPLE}"
export fgcolor_cyan_bold="${BOLD_CYAN}"

export fgcolor_bright_black="${BRIGHT_BLACK}"
export fgcolor_bright_red="${BRIGHT_RED}"
export fgcolor_bright_green="${BRIGHT_GREEN}"
export fgcolor_bright_yellow="${BRIGHT_YELLOW}"
export fgcolor_bright_blue="${BRIGHT_BLUE}"
export fgcolor_bright_purple="${BRIGHT_PURPLE}"
export fgcolor_bright_cyan="${BRIGHT_CYAN}"
export fgcolor_bright_white="${BRIGHT_WHITE}"

export bgcolor_black="${BG_BLACK}"
export bgcolor_red="${BG_RED}"
export bgcolor_green="${BG_GREEN}"
export bgcolor_yellow="${BG_YELLOW}"
export bgcolor_blue="${BG_BLUE}"
export bgcolor_purple="${BG_PURPLE}"
export bgcolor_cyan="${BG_CYAN}"
export bgcolor_white="${BG_WHITE}"

export fcr="${RESET}"
export fcwb="${BOLD_WHITE}"
export fcgreenb="${BOLD_GREEN}"
export fcyellowb="${BOLD_YELLOW}"
export fcredb="${BOLD_RED}"
export fcblueb="${BOLD_BLUE}"
export fccyanb="${BOLD_CYAN}"
export fcpurpleb="${BOLD_PURPLE}"
