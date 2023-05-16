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
