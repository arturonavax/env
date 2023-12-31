#!/bin/bash
# Run: curl -fsSL "https://env.arturonavax.dev/macos_desktops.sh" | bash

desirable_desktops=5
count_desktops=$(defaults read com.apple.spaces SpacesDisplayConfiguration | command grep "uuid" | command grep -v 'uuid = "";' | wc -l)

if ((desirable_desktops - count_desktops >= 1)); then
	osascript <<EOF
do shell script "open -b com.apple.exposelauncher"
delay 0.5
tell application id "com.apple.systemevents"
	repeat (($desirable_desktops - $count_desktops)) times
		tell (every application process ¬
			whose bundle identifier = "com.apple.dock") to ¬
			click (button 1 of group 2 of group 1 of group 1)
		delay 0.5
	end repeat

	key code 53 -- esc key
end tell
EOF
fi

/usr/libexec/PlistBuddy -c "Set :AppleSymbolicHotKeys:79:enabled 1" ~/Library/Preferences/com.apple.symbolichotkeys.plist
/usr/libexec/PlistBuddy -c "Set :AppleSymbolicHotKeys:80:enabled 1" ~/Library/Preferences/com.apple.symbolichotkeys.plist
/usr/libexec/PlistBuddy -c "Set :AppleSymbolicHotKeys:81:enabled 1" ~/Library/Preferences/com.apple.symbolichotkeys.plist
/usr/libexec/PlistBuddy -c "Set :AppleSymbolicHotKeys:82:enabled 1" ~/Library/Preferences/com.apple.symbolichotkeys.plist
/usr/libexec/PlistBuddy -c "Set :AppleSymbolicHotKeys:98:enabled 1" ~/Library/Preferences/com.apple.symbolichotkeys.plist
/usr/libexec/PlistBuddy -c "Set :AppleSymbolicHotKeys:118:enabled 1" ~/Library/Preferences/com.apple.symbolichotkeys.plist
/usr/libexec/PlistBuddy -c "Set :AppleSymbolicHotKeys:119:enabled 1" ~/Library/Preferences/com.apple.symbolichotkeys.plist
/usr/libexec/PlistBuddy -c "Set :AppleSymbolicHotKeys:120:enabled 1" ~/Library/Preferences/com.apple.symbolichotkeys.plist
/usr/libexec/PlistBuddy -c "Set :AppleSymbolicHotKeys:121:enabled 1" ~/Library/Preferences/com.apple.symbolichotkeys.plist
/usr/libexec/PlistBuddy -c "Set :AppleSymbolicHotKeys:122:enabled 1" ~/Library/Preferences/com.apple.symbolichotkeys.plist
/usr/libexec/PlistBuddy -c "Set :AppleSymbolicHotKeys:123:enabled 1" ~/Library/Preferences/com.apple.symbolichotkeys.plist
/usr/libexec/PlistBuddy -c "Set :AppleSymbolicHotKeys:124:enabled 1" ~/Library/Preferences/com.apple.symbolichotkeys.plist
/usr/libexec/PlistBuddy -c "Set :AppleSymbolicHotKeys:125:enabled 1" ~/Library/Preferences/com.apple.symbolichotkeys.plist
/usr/libexec/PlistBuddy -c "Set :AppleSymbolicHotKeys:126:enabled 1" ~/Library/Preferences/com.apple.symbolichotkeys.plist
/usr/libexec/PlistBuddy -c "Set :AppleSymbolicHotKeys:127:enabled 1" ~/Library/Preferences/com.apple.symbolichotkeys.plist

# Ask the system to read the hotkey plist file and ignore the output. Likely updates an in-memory cache with the new plist values.
defaults read com.apple.symbolichotkeys.plist >/dev/null

# Run reactivateSettings to apply the updated settings.
/System/Library/PrivateFrameworks/SystemAdministration.framework/Resources/activateSettings -u
