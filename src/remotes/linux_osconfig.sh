#!/bin/bash
# Run: curl -fsSL "https://env.arturonavax.dev/linux_osconfig.sh" | bash
if [[ "$(command -v gsettings)" != "" ]]; then
	gsettings set org.gnome.desktop.peripherals.keyboard repeat true
	gsettings set org.gnome.desktop.peripherals.keyboard delay 200
	gsettings set org.gnome.desktop.peripherals.keyboard repeat-interval 30
fi

if [[ "$(command -v xset)" != "" ]]; then
	xset r rate 200 30
fi
