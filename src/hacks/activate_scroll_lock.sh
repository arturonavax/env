#!/bin/bash
# This script is to make the "Scroll Lock" or "Scroll Lock" key work in Linux,
# you must add the execution of this script to the corresponding physical key.
# In the Gnome shortcuts for example.
#
# Run: curl -fsSL "https://env.arturonavax.dev/activate_scroll_lock.sh" | bash
if [[ "$(uname -s)" == "Linux" ]]; then
	on="$(xset -q | grep 'Scroll Lock:' | cut -d ':' -f 7)"

	if [[ "$on" == " off" ]]; then
		xset led named "Scroll Lock"
	else
		xset -led named "Scroll Lock"
	fi

elif [[ "$(uname -s)" == "Darwin" ]]; then
	echo "!!! Under construction for MacOS"
fi
