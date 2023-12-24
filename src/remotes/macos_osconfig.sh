#!/bin/bash
# Run: curl -fsSL "https://env.arturonavax.dev/macos_osconfig.sh" | bash

# Close any open System Preferences panes, to prevent them from overriding
# settings we’re about to change
osascript -e 'tell application "System Preferences" to quit'

# Ask for the administrator password upfront
sudo -v

# Keep-alive: update existing `sudo` time stamp until `.macos` has finished
while true; do
	sudo -n true
	sleep 60
	kill -0 "$$" || exit
done 2>/dev/null &

# Set computer name (as done via System Preferences → Sharing)
# sudo scutil --set ComputerName "MacBook Arturo"
# sudo scutil --set HostName "MacBook Arturo"
# sudo scutil --set LocalHostName "MacBook Arturo"
# sudo defaults write /Library/Preferences/SystemConfiguration/com.apple.smb.server NetBIOSName -string "MacBook Arturo"

# Disable the sound effects on boot
sudo nvram SystemAudioVolume=" "
sudo nvram StartupMute=%01

# Set sidebar icon size to medium
defaults write NSGlobalDomain NSTableViewDefaultSizeMode -int 2

# Increase window resize speed for Cocoa applications
defaults write NSGlobalDomain NSWindowResizeTime -float 0.001

# Expand save panel by default
defaults write NSGlobalDomain NSNavPanelExpandedStateForSaveMode -bool true
defaults write NSGlobalDomain NSNavPanelExpandedStateForSaveMode2 -bool true

# Expand print panel by default
defaults write NSGlobalDomain PMPrintingExpandedStateForPrint -bool true
defaults write NSGlobalDomain PMPrintingExpandedStateForPrint2 -bool true

# Save to disk (not to iCloud) by default
defaults write NSGlobalDomain NSDocumentSaveNewDocumentsToCloud -bool false

# Automatically quit printer app once the print jobs complete
defaults write com.apple.print.PrintingPrefs "Quit When Finished" -bool true

# Disable the “Are you sure you want to open this application?” dialog
defaults write com.apple.LaunchServices LSQuarantine -bool false

# Remove duplicates in the “Open With” menu (also see `lscleanup` alias)
/System/Library/Frameworks/CoreServices.framework/Frameworks/LaunchServices.framework/Support/lsregister -kill -r -domain local -domain system -domain user

# Display ASCII control characters using caret notation in standard text views
# Try e.g. `cd /tmp; unidecode "\x{0000}" > cc.txt; open -e cc.txt`
defaults write NSGlobalDomain NSTextShowsControlCharacters -bool true

# Disable Resume system-wide
defaults write com.apple.systempreferences NSQuitAlwaysKeepsWindows -bool false

# Disable automatic termination of inactive apps
defaults write NSGlobalDomain NSDisableAutomaticTermination -bool true

# Reveal IP address, hostname, OS version, etc. when clicking the clock
# in the login window
sudo defaults write /Library/Preferences/com.apple.loginwindow AdminHostInfo HostName

# Show list of users in the login window instead of having to enter their names
sudo defaults write /Library/Preferences/com.apple.loginwindow SHOWFULLNAME -int 0

# Show language menu in the top right corner of the boot screen
sudo defaults write /Library/Preferences/com.apple.loginwindow showInputMenu -bool true

# Disable screen saver
defaults -currentHost write com.apple.screensaver idleTime -int 0

# Show clock in screen saver
defaults -currentHost write com.apple.screensaver showClock -int 1

# creates symbolic link for "airport" command
[[ "$(command -v airport)" == "" ]] && sudo ln -s /System/Library/PrivateFrameworks/Apple80211.framework/Versions/Current/Resources/airport /usr/local/bin/airport

# show hidden folders
chflags nohidden ~/
chflags nohidden ~/Library && xattr -d com.apple.FinderInfo ~/Library

# Show the /Volumes folder
sudo chflags nohidden /Volumes

# show extensions of all files
defaults write NSGlobalDomain AppleShowAllExtensions -bool true

# Trackpad: three finger drag
defaults write com.apple.AppleMultitouchTrackpad "TrackpadThreeFingerDrag" -bool "true"

# disable mouse natural scroll
defaults write NSGlobalDomain com.apple.swipescrolldirection -bool false

# Trackpad: disable tap to click for this user and for the login screen
defaults write com.apple.driver.AppleBluetoothMultitouch.trackpad Clicking -bool false
defaults -currentHost write NSGlobalDomain com.apple.mouse.tapBehavior -int 0
defaults write NSGlobalDomain com.apple.mouse.tapBehavior -int 0

# Show battery percentage
defaults write com.apple.menuextra.battery ShowPercent -string "YES"
defaults -currentHost write com.apple.controlcenter.plist BatteryShowPercentage -bool true

# disable .DS_Store files
# Avoid creating .DS_Store files on network or USB volumes
defaults write com.apple.desktopservices DSDontWriteNetworkStores -bool true
defaults write com.apple.desktopservices DSDontWriteUSBStores -bool true

# Finder: allow quitting via ⌘ + Q; doing so will also hide desktop icons
defaults write com.apple.finder QuitMenuItem -bool true

# finder: show hidden files
defaults write com.apple.finder AppleShowAllFiles -bool true

# show scroll bars always
defaults write NSGlobalDomain "AppleShowScrollBars" -string "Always"

# finder: show path bar
defaults write com.apple.finder ShowPathbar -bool true

# finder: show status bar
defaults write com.apple.finder ShowStatusBar -bool true

# When performing a search, search the current folder by default
defaults write com.apple.finder FXDefaultSearchScope -string "SCcf"

# Keep folders on top when sorting by name
defaults write com.apple.finder _FXSortFoldersFirst -bool true

# Disable the warning when changing a file extension
defaults write com.apple.finder FXEnableExtensionChangeWarning -bool false

# Enable spring loading for directories
defaults write NSGlobalDomain com.apple.springing.enabled -bool true

# Remove the spring loading delay for directories
defaults write NSGlobalDomain com.apple.springing.delay -float 0

# finder: show item info
# Show item info near icons on the desktop and in other icon views
/usr/libexec/PlistBuddy -c "Set :DesktopViewSettings:IconViewSettings:showItemInfo true" ~/Library/Preferences/com.apple.finder.plist
/usr/libexec/PlistBuddy -c "Set :FK_StandardViewSettings:IconViewSettings:showItemInfo true" ~/Library/Preferences/com.apple.finder.plist
/usr/libexec/PlistBuddy -c "Set :StandardViewSettings:IconViewSettings:showItemInfo true" ~/Library/Preferences/com.apple.finder.plist

# Show item info to the right of the icons on the desktop
/usr/libexec/PlistBuddy -c "Set DesktopViewSettings:IconViewSettings:labelOnBottom false" ~/Library/Preferences/com.apple.finder.plist

# Enable snap-to-grid for icons on the desktop and in other icon views
/usr/libexec/PlistBuddy -c "Set :DesktopViewSettings:IconViewSettings:arrangeBy grid" ~/Library/Preferences/com.apple.finder.plist
/usr/libexec/PlistBuddy -c "Set :FK_StandardViewSettings:IconViewSettings:arrangeBy grid" ~/Library/Preferences/com.apple.finder.plist
/usr/libexec/PlistBuddy -c "Set :StandardViewSettings:IconViewSettings:arrangeBy grid" ~/Library/Preferences/com.apple.finder.plist

# Increase grid spacing for icons on the desktop and in other icon views
/usr/libexec/PlistBuddy -c "Set :DesktopViewSettings:IconViewSettings:gridSpacing 65" ~/Library/Preferences/com.apple.finder.plist
/usr/libexec/PlistBuddy -c "Set :FK_StandardViewSettings:IconViewSettings:gridSpacing 65" ~/Library/Preferences/com.apple.finder.plist
/usr/libexec/PlistBuddy -c "Set :StandardViewSettings:IconViewSettings:gridSpacing 65" ~/Library/Preferences/com.apple.finder.plist

# Increase the size of icons on the desktop and in other icon views
/usr/libexec/PlistBuddy -c "Set :DesktopViewSettings:IconViewSettings:iconSize 80" ~/Library/Preferences/com.apple.finder.plist
/usr/libexec/PlistBuddy -c "Set :FK_StandardViewSettings:IconViewSettings:iconSize 80" ~/Library/Preferences/com.apple.finder.plist
/usr/libexec/PlistBuddy -c "Set :StandardViewSettings:IconViewSettings:iconSize 80" ~/Library/Preferences/com.apple.finder.plist

# Use list view in all Finder windows by default
# Four-letter codes for the other view modes: `icnv`, `clmv`, `glyv`
defaults write com.apple.finder FXPreferredViewStyle -string "Nlsv"

# finder: order by kind
defaults write com.apple.finder FXPreferredGroupBy kind
/usr/libexec/PlistBuddy -c 'set :StandardViewSettings:IconViewSettings:arrangeBy kind' ~/Library/Preferences/com.apple.finder.plist
/usr/libexec/PlistBuddy -c 'set :StandardViewSettings:ExtendedListViewSettingsV2:sortColumn dateLastOpened' ~/Library/Preferences/com.apple.finder.plist
/usr/libexec/PlistBuddy -c 'set :"FK_StandardViewSettings":IconViewSettings:arrangeBy kind' ~/Library/Preferences/com.apple.finder.plist
/usr/libexec/PlistBuddy -c 'set :"FK_StandardViewSettings":ExtendedListViewSettingsV2:sortColumn dateLastOpened' ~/Library/Preferences/com.apple.finder.plist

# Show icons for hard drives, servers, and removable media on the desktop
defaults write com.apple.finder ShowExternalHardDrivesOnDesktop -bool true
defaults write com.apple.finder ShowHardDrivesOnDesktop -bool true
defaults write com.apple.finder ShowMountedServersOnDesktop -bool true
defaults write com.apple.finder ShowRemovableMediaOnDesktop -bool true

# finder: show the path in the title in POSIX style
#defaults write com.apple.finder _FXShowPosixPathInTitle -bool true

# Set $HOME as the default location for new Finder windows
defaults write com.apple.finder NewWindowTarget -string "PfLo"
defaults write com.apple.finder NewWindowTargetPath -string "file://${HOME}"

# Expand the following File Info panes:
# “General”, “Open with”, and “Sharing & Permissions”
defaults write com.apple.finder FXInfoPanesExpanded -dict \
	General -bool true \
	OpenWith -bool true \
	Privileges -bool true

# dock
defaults write com.apple.dock orientation -string left
defaults write com.apple.dock autohide -bool true
defaults write com.apple.dock autohide-delay -int 0
defaults write com.apple.dock autohide-time-modifier -float 0.8
defaults write com.apple.dock magnification -int 1
defaults write com.apple.dock largesize -int 65
defaults write com.apple.dock tilesize -int 55
defaults write com.apple.dock expose-animation-duration -float 0.05
defaults write com.apple.dock show-recents -bool false

# default terminal exit
/usr/libexec/PlistBuddy -c 'add :"Window Settings":Basic:shellExitAction integer 0' ~/Library/Preferences/com.apple.Terminal.plist
/usr/libexec/PlistBuddy -c 'set :"Window Settings":Basic:shellExitAction 0' ~/Library/Preferences/com.apple.Terminal.plist

# Disable automatic capitalization as it’s annoying when typing code
defaults write NSGlobalDomain NSAutomaticCapitalizationEnabled -bool false

# Disable smart dashes as they’re annoying when typing code
defaults write NSGlobalDomain NSAutomaticDashSubstitutionEnabled -bool false

# Disable automatic period substitution as it’s annoying when typing code
defaults write NSGlobalDomain NSAutomaticPeriodSubstitutionEnabled -bool false

# Disable smart quotes as they’re annoying when typing code
defaults write NSGlobalDomain NSAutomaticQuoteSubstitutionEnabled -bool false

# Disable auto-correct
defaults write NSGlobalDomain NSAutomaticSpellingCorrectionEnabled -bool false

# speed keyboard
defaults write NSGlobalDomain ApplePressAndHoldEnabled -bool false
defaults write NSGlobalDomain KeyRepeat -int 2
defaults write NSGlobalDomain InitialKeyRepeat -int 15

# Require password immediately after sleep or screen saver begins
defaults write com.apple.screensaver askForPassword -int 1
defaults write com.apple.screensaver askForPasswordDelay -int 0

# Enable subpixel font rendering on non-Apple LCDs
# Reference: https://github.com/kevinSuttle/macOS-Defaults/issues/17#issuecomment-266633501
defaults write NSGlobalDomain AppleFontSmoothing -int 1

# Enable HiDPI display modes (requires restart)
sudo defaults write /Library/Preferences/com.apple.windowserver DisplayResolutionEnabled -bool true

# disable saving of window status on power off
sudo defaults write com.apple.loginwindow TALLogoutSavesState -bool false
defaults write NSGlobalDomain NSQuitAlwaysKeepsWindows -bool false
defaults write NSGlobalDomain ApplePersistence -bool no

# disable save state of Alacritty
mkdir -p "$HOME/Library/Saved Application State/org.alacritty.savedState/"
rm -rf "$HOME/Library/Saved Application State/org.alacritty.savedState/*"
chmod -R a-w "$HOME/Library/Saved Application State/org.alacritty.savedState/"

# screenshots in PNG format
defaults write com.apple.screencapture type -string "png"

# screenshots location
mkdir -p ~/Pictures/Screenshots
defaults write com.apple.screencapture location ~/Pictures/Screenshots

# Add iOS & Watch Simulator to Launchpad
sudo ln -sf "/Applications/Xcode.app/Contents/Developer/Applications/Simulator.app" "/Applications/Simulator.app"
sudo ln -sf "/Applications/Xcode.app/Contents/Developer/Applications/Simulator (Watch).app" "/Applications/Simulator (Watch).app"

# Hot corners
# Possible values:
#  0: no-op
#  2: Mission Control
#  3: Show application windows
#  4: Desktop
#  5: Start screen saver
#  6: Disable screen saver
#  7: Dashboard
# 10: Put display to sleep
# 11: Launchpad
# 12: Notification Center
# 13: Lock Screen
# Top left screen corner → Mission Control
defaults write com.apple.dock wvous-tl-corner -int 2
defaults write com.apple.dock wvous-tl-modifier -int 0

# Bottom left screen corner → Lock Screen
defaults write com.apple.dock wvous-bl-corner -int 13
defaults write com.apple.dock wvous-bl-modifier -int 0

# Bottom right screen corner → Desktop
defaults write com.apple.dock wvous-br-corner -int 4
defaults write com.apple.dock wvous-br-modifier -int 0

# Only use UTF-8 in Terminal.app
defaults write com.apple.terminal StringEncodings -array 4

# Enable Secure Keyboard Entry in Terminal.app
# See: https://security.stackexchange.com/a/47786/8918
defaults write com.apple.terminal SecureKeyboardEntry -bool true

###############################################################################
# Activity Monitor                                                            #
###############################################################################

# Show the main window when launching Activity Monitor
defaults write com.apple.ActivityMonitor OpenMainWindow -bool true

# Visualize CPU usage in the Activity Monitor Dock icon
defaults write com.apple.ActivityMonitor IconType -int 5

# Show all processes in Activity Monitor
defaults write com.apple.ActivityMonitor ShowCategory -int 0

# Sort Activity Monitor results by CPU usage
defaults write com.apple.ActivityMonitor SortColumn -string "CPUUsage"
defaults write com.apple.ActivityMonitor SortDirection -int 0

###############################################################################
# Mac App Store                                                               #
###############################################################################

# Enable the WebKit Developer Tools in the Mac App Store
defaults write com.apple.appstore WebKitDeveloperExtras -bool true

# Enable Debug Menu in the Mac App Store
defaults write com.apple.appstore ShowDebugMenu -bool true

# Enable the automatic update check
defaults write com.apple.SoftwareUpdate AutomaticCheckEnabled -bool true

# Check for software updates daily, not just once per week
defaults write com.apple.SoftwareUpdate ScheduleFrequency -int 1

# Download newly available updates in background
defaults write com.apple.SoftwareUpdate AutomaticDownload -int 1

# Install System data files & security updates
defaults write com.apple.SoftwareUpdate CriticalUpdateInstall -int 1

# Automatically download apps purchased on other Macs
defaults write com.apple.SoftwareUpdate ConfigDataInstall -int 1

# Turn on app auto-update
defaults write com.apple.commerce AutoUpdate -bool true

# Allow the App Store to reboot machine on macOS updates
defaults write com.apple.commerce AutoUpdateRestartRequired -bool true

###############################################################################
# Photos                                                                      #
###############################################################################

# Prevent Photos from opening automatically when devices are plugged in
defaults -currentHost write com.apple.ImageCapture disableHotPlug -bool true

###############################################################################
# Google Chrome & Google Chrome Canary                                        #
###############################################################################

# Disable the all too sensitive backswipe on trackpads
defaults write com.google.Chrome AppleEnableSwipeNavigateWithScrolls -bool false
defaults write com.google.Chrome.canary AppleEnableSwipeNavigateWithScrolls -bool false

# Disable the all too sensitive backswipe on Magic Mouse
defaults write com.google.Chrome AppleEnableMouseSwipeNavigateWithScrolls -bool false
defaults write com.google.Chrome.canary AppleEnableMouseSwipeNavigateWithScrolls -bool false

# Use the system-native print preview dialog
defaults write com.google.Chrome DisablePrintPreview -bool true
defaults write com.google.Chrome.canary DisablePrintPreview -bool true

# Expand the print dialog by default
defaults write com.google.Chrome PMPrintingExpandedStateForPrint2 -bool true
defaults write com.google.Chrome.canary PMPrintingExpandedStateForPrint2 -bool true

###############################################################################
# Opera & Opera Developer                                                     #
###############################################################################

# Expand the print dialog by default
defaults write com.operasoftware.Opera PMPrintingExpandedStateForPrint2 -boolean true
defaults write com.operasoftware.OperaDeveloper PMPrintingExpandedStateForPrint2 -boolean true

###############################################################################
# Transmission.app                                                            #
###############################################################################

# Use `~/Documents/Torrents` to store incomplete downloads
defaults write org.m0k.transmission UseIncompleteDownloadFolder -bool true
defaults write org.m0k.transmission IncompleteDownloadFolder -string "${HOME}/Documents/Torrents"

# Use `~/Downloads` to store completed downloads
defaults write org.m0k.transmission DownloadLocationConstant -bool true

# Don’t prompt for confirmation before downloading
defaults write org.m0k.transmission DownloadAsk -bool false
defaults write org.m0k.transmission MagnetOpenAsk -bool false

# Don’t prompt for confirmation before removing non-downloading active transfers
defaults write org.m0k.transmission CheckRemoveDownloading -bool true

# Trash original torrent files
defaults write org.m0k.transmission DeleteOriginalTorrent -bool true

# Hide the donate message
defaults write org.m0k.transmission WarningDonate -bool false
# Hide the legal disclaimer
defaults write org.m0k.transmission WarningLegal -bool false

# IP block list.
# Source: https://giuliomac.wordpress.com/2014/02/19/best-blocklist-for-transmission/
defaults write org.m0k.transmission BlocklistNew -bool true
defaults write org.m0k.transmission BlocklistURL -string "http://john.bitsurge.net/public/biglist.p2p.gz"
defaults write org.m0k.transmission BlocklistAutoUpdate -bool true

# Randomize port on launch
defaults write org.m0k.transmission RandomPort -bool true

###############################################################################
# Mail                                                                        #
###############################################################################

# Disable send and reply animations in Mail.app
defaults write com.apple.mail DisableReplyAnimations -bool true
defaults write com.apple.mail DisableSendAnimations -bool true

# Copy email addresses as `foo@example.com` instead of `Foo Bar <foo@example.com>` in Mail.app
defaults write com.apple.mail AddressesIncludeNameOnPasteboard -bool false

# Add the keyboard shortcut ⌘ + Enter to send an email in Mail.app
defaults write com.apple.mail NSUserKeyEquivalents -dict-add "Send" "@\U21a9"

# Display emails in threaded mode, sorted by date (oldest at the top)
defaults write com.apple.mail DraftsViewerAttributes -dict-add "DisplayInThreadedMode" -string "yes"
defaults write com.apple.mail DraftsViewerAttributes -dict-add "SortedDescending" -string "yes"
defaults write com.apple.mail DraftsViewerAttributes -dict-add "SortOrder" -string "received-date"

# Disable inline attachments (just show the icons)
defaults write com.apple.mail DisableInlineAttachmentViewing -bool true

# Disable automatic spell checking
defaults write com.apple.mail SpellCheckingBehavior -string "NoSpellCheckingEnabled"

###############################################################################
# Energy saving                                                               #
###############################################################################

# Enable lid wakeup
sudo pmset -a lidwake 1

# Sleep the display after 15 minutes
sudo pmset -a displaysleep 30

# Disable machine sleep while charging
sudo pmset -c sleep 0

# Set machine sleep to 5 minutes on battery
sudo pmset -b sleep 0

# Set standby delay to 24 hours (default is 1 hour)
sudo pmset -a standbydelay 86400

# Never go into computer sleep mode
sudo systemsetup -setcomputersleep Off >/dev/null

# Hibernation mode
# 0: Disable hibernation (speeds up entering sleep mode)
# 3: Copy RAM to disk so the system state can still be restored in case of a
#    power failure.
sudo pmset -a hibernatemode 0

###############################################################################
# Kill affected applications                                                  #
###############################################################################

for app in "Activity Monitor" \
	"Address Book" \
	"Calendar" \
	"cfprefsd" \
	"Contacts" \
	"Dock" \
	"Finder" \
	"Google Chrome Canary" \
	"Google Chrome" \
	"Mail" \
	"Messages" \
	"Opera" \
	"Photos" \
	"Safari" \
	"SizeUp" \
	"Spectacle" \
	"SystemUIServer" \
	"Terminal" \
	"Transmission" \
	"Tweetbot" \
	"Twitter" \
	"iCal"; do
	killall "${app}" &>/dev/null
done
echo "Done. Note that some of these changes require a logout/restart to take effect."
