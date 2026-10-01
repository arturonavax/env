#!/bin/bash
# This script must be executed with the "source" command.
#
# Run: source <(curl -fsSL "https://env.arturonavax.dev/_vars_colors.sh" | cat)

# Reset Text Color
export fgcolor_reset='\033[0m'

# Regular Text Colors
export \
	fgcolor_black='\033[0;30m' \
	fgcolor_white='\033[0;37m' \
	fgcolor_blue='\033[0;34m' \
	fgcolor_red='\033[0;31m' \
	fgcolor_green='\033[0;32m' \
	fgcolor_yellow='\033[0;33m' \
	fgcolor_purple='\033[0;35m' \
	fgcolor_cyan='\033[0;36m'

# Bold Text Colors
export \
	fgcolor_black_bold='\033[1;30m' \
	fgcolor_white_bold='\033[1;37m' \
	fgcolor_blue_bold='\033[1;34m' \
	fgcolor_red_bold='\033[1;31m' \
	fgcolor_green_bold='\033[1;32m' \
	fgcolor_yellow_bold='\033[1;33m' \
	fgcolor_purple_bold='\033[1;35m' \
	fgcolor_cyan_bold='\033[1;36m'

# Underline Text Colors
export \
	fgcolor_black_underline='\033[4;30m' \
	fgcolor_white_underline='\033[4;37m' \
	fgcolor_blue_underline='\033[4;34m' \
	fgcolor_red_underline='\033[4;31m' \
	fgcolor_green_underline='\033[4;32m' \
	fgcolor_yellow_underline='\033[4;33m' \
	fgcolor_purple_underline='\033[4;35m' \
	fgcolor_cyan_underline='\033[4;36m'

# Background Colors
export \
	bgcolor_black='\033[40m' \
	bgcolor_red='\033[41m' \
	bgcolor_green='\033[42m' \
	bgcolor_yellow='\033[43m' \
	bgcolor_blue='\033[44m' \
	bgcolor_purple='\033[45m' \
	bgcolor_cyan='\033[46m' \
	bgcolor_white='\033[47m'

# High-Intensity (Bright) Foreground Colors
export \
	fgcolor_gray='\033[0;90m' \
	fgcolor_bright_black='\033[0;90m' \
	fgcolor_bright_red='\033[0;91m' \
	fgcolor_bright_green='\033[0;92m' \
	fgcolor_bright_yellow='\033[0;93m' \
	fgcolor_bright_blue='\033[0;94m' \
	fgcolor_bright_purple='\033[0;95m' \
	fgcolor_bright_cyan='\033[0;96m' \
	fgcolor_bright_white='\033[0;97m'

# Short Convenience Aliases (for concise prints and scripts)
export \
	fcr="${fgcolor_reset}" \
	fcwb="${fgcolor_white_bold}" \
	fcgreenb="${fgcolor_green_bold}" \
	fcyellowb="${fgcolor_yellow_bold}" \
	fcredb="${fgcolor_red_bold}" \
	fcblueb="${fgcolor_blue_bold}" \
	fccyanb="${fgcolor_cyan_bold}" \
	fcpurpleb="${fgcolor_purple_bold}"
