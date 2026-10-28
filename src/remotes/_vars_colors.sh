#!/usr/bin/env bash
# ==============================================================================
# src/remotes/_vars_colors.sh
# Terminal color palette and formatting variables.
# Servable from https://env.arturonavax.dev/_vars_colors.sh
# ==============================================================================

# If running inside repo, source modern library
if [[ -f ./src/lib/colors.sh ]]; then
	# shellcheck source=/dev/null
	source ./src/lib/colors.sh
	return 0
fi

# Reset
export fgcolor_reset='\033[0m'
export RESET='\033[0m'
export BOLD='\033[1m'
export DIM='\033[2m'

# Regular Colors
export fgcolor_black='\033[0;30m'
export fgcolor_white='\033[0;37m'
export fgcolor_blue='\033[0;34m'
export fgcolor_red='\033[0;31m'
export fgcolor_green='\033[0;32m'
export fgcolor_yellow='\033[0;33m'
export fgcolor_purple='\033[0;35m'
export fgcolor_cyan='\033[0;36m'
export fgcolor_gray='\033[0;90m'

# Bold Colors
export fgcolor_black_bold='\033[1;30m'
export fgcolor_white_bold='\033[1;37m'
export fgcolor_blue_bold='\033[1;34m'
export fgcolor_red_bold='\033[1;31m'
export fgcolor_green_bold='\033[1;32m'
export fgcolor_yellow_bold='\033[1;33m'
export fgcolor_purple_bold='\033[1;35m'
export fgcolor_cyan_bold='\033[1;36m'

# High-Intensity
export fgcolor_bright_black='\033[0;90m'
export fgcolor_bright_red='\033[0;91m'
export fgcolor_bright_green='\033[0;92m'
export fgcolor_bright_yellow='\033[0;93m'
export fgcolor_bright_blue='\033[0;94m'
export fgcolor_bright_purple='\033[0;95m'
export fgcolor_bright_cyan='\033[0;96m'
export fgcolor_bright_white='\033[0;97m'

# Backgrounds
export bgcolor_black='\033[40m'
export bgcolor_red='\033[41m'
export bgcolor_green='\033[42m'
export bgcolor_yellow='\033[43m'
export bgcolor_blue='\033[44m'
export bgcolor_purple='\033[45m'
export bgcolor_cyan='\033[46m'
export bgcolor_white='\033[47m'

# Aliases
export fcr="${fgcolor_reset}"
export fcwb="${fgcolor_white_bold}"
export fcgb="${fgcolor_green_bold}"
export fcgreenb="${fgcolor_green_bold}"
export fcyb="${fgcolor_yellow_bold}"
export fcyellowb="${fgcolor_yellow_bold}"
export fcrb="${fgcolor_red_bold}"
export fcredb="${fgcolor_red_bold}"
export fcbb="${fgcolor_blue_bold}"
export fcblueb="${fgcolor_blue_bold}"
export fccb="${fgcolor_cyan_bold}"
export fccyanb="${fgcolor_cyan_bold}"
export fcpurpleb="${fgcolor_purple_bold}"
