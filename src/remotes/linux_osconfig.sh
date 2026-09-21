#!/bin/bash
# Run: curl -fsSL "https://env.arturonavax.dev/linux_osconfig.sh" | bash
# GNOME Keyboard configuration
if [[ "$(command -v gsettings)" != "" ]]; then
	gsettings set org.gnome.desktop.peripherals.keyboard repeat true
	gsettings set org.gnome.desktop.peripherals.keyboard delay 200
	gsettings set org.gnome.desktop.peripherals.keyboard repeat-interval 30
fi

# Xfce Keyboard configuration
if [[ "$(command -v xfconf-query)" != "" ]]; then
	xfconf-query -c keyboards -p /Default/KeyRepeat -n -t bool -s true
	xfconf-query -c keyboards -p /Default/KeyRepeat/Delay -n -t int -s 200
	xfconf-query -c keyboards -p /Default/KeyRepeat/Rate -n -t int -s 30
	xfconf-query -c keyboards -p /Default/RestoreNumlock -n -t bool -s true
	xfconf-query -c keyboards -p /Default/XkbDisable -n -t bool -s false
fi

# X11 Key Repeat (immediate effect for active session)
if [[ "$(command -v xset)" != "" ]]; then
	xset r rate 200 30
fi

