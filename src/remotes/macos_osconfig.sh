#!/bin/bash
# Run: curl -fsSL "https://env.arturonavax.dev/macos_osconfig.sh" | bash

# creates symbolic link for "airport" command
[[ "$(command -v airport)" == "" ]] && sudo ln -s /System/Library/PrivateFrameworks/Apple80211.framework/Versions/Current/Resources/airport /usr/local/bin/airport

# show hidden folders
chflags nohidden ~/
chflags nohidden ~/Library

# show extensions of all files
defaults write -g AppleShowAllExtensions -bool true

# enable mouse natural scroll
defaults write -g com.apple.swipescrolldirection -bool true && killall cfprefsd

# disable .DS_Store files
defaults write com.apple.desktopservices DSDontWriteNetworkStores true

# finder: show hidden files
defaults write com.apple.finder AppleShowAllFiles -bool true

# finder: show path bar
defaults write com.apple.finder ShowPathbar -bool true

# finder: show status bar
defaults write com.apple.finder ShowStatusBar -bool true

# finder: search in the current folder
defaults write com.apple.finder FXDefaultSearchScope -string "SCcf"

# finder: show folders first
defaults write com.apple.finder _FXSortFoldersFirst -int 1

# finder: show item info
/usr/libexec/PlistBuddy -c 'set :StandardViewSettings:IconViewSettings:showItemInfo 1' ~/Library/Preferences/com.apple.finder.plist
/usr/libexec/PlistBuddy -c 'set :"FK_StandardViewSettings":IconViewSettings:showItemInfo 1' ~/Library/Preferences/com.apple.finder.plist
/usr/libexec/PlistBuddy -c 'set :DesktopViewSettings:IconViewSettings:showItemInfo integer 1' ~/Library/Preferences/com.apple.finder.plist
/usr/libexec/PlistBuddy -c 'add :SearchRecentsViewSettings:IconViewSettings:showItemInfo integer 1' ~/Library/Preferences/com.apple.finder.plist
/usr/libexec/PlistBuddy -c 'set :SearchRecentsViewSettings:IconViewSettings:showItemInfo 1' ~/Library/Preferences/com.apple.finder.plist

# finder: order by kind
defaults write com.apple.finder FXPreferredGroupBy kind
/usr/libexec/PlistBuddy -c 'set :StandardViewSettings:IconViewSettings:arrangeBy kind' ~/Library/Preferences/com.apple.finder.plist
/usr/libexec/PlistBuddy -c 'set :StandardViewSettings:ExtendedListViewSettingsV2:sortColumn dateLastOpened' ~/Library/Preferences/com.apple.finder.plist
/usr/libexec/PlistBuddy -c 'set :"FK_StandardViewSettings":IconViewSettings:arrangeBy kind' ~/Library/Preferences/com.apple.finder.plist
/usr/libexec/PlistBuddy -c 'set :"FK_StandardViewSettings":ExtendedListViewSettingsV2:sortColumn dateLastOpened' ~/Library/Preferences/com.apple.finder.plist

# finder: show the path in the title in POSIX style
#defaults write com.apple.finder _FXShowPosixPathInTitle -bool true

# dock
defaults write com.apple.dock orientation -string left
defaults write com.apple.dock autohide -bool true
defaults write com.apple.dock magnification -int 1
defaults write com.apple.dock largesize -int 65
defaults write com.apple.dock tilesize -int 55
defaults write com.apple.dock "show-recents" -int 0
defaults write com.apple.dock expose-animation-duration -float 0.05

# default terminal exit
/usr/libexec/PlistBuddy -c 'add :"Window Settings":Basic:shellExitAction integer 0' ~/Library/Preferences/com.apple.Terminal.plist
/usr/libexec/PlistBuddy -c 'set :"Window Settings":Basic:shellExitAction 0' ~/Library/Preferences/com.apple.Terminal.plist

# disable correctors
defaults write -g NSAutomaticCapitalizationEnabled -bool false
defaults write -g NSAutomaticSpellingCorrectionEnabled -bool false
defaults write -g NSAutomaticDashSubstitutionEnabled -bool false
defaults write -g NSAutomaticPeriodSubstitutionEnabled -bool false
defaults write -g NSAutomaticQuoteSubstitutionEnabled -bool false

# speed keyboard
defaults write -g ApplePressAndHoldEnabled -bool false
defaults write -g KeyRepeat -int 2
defaults write -g InitialKeyRepeat -int 15

# font smoothing
#defaults write -g CGFontRenderingFontSmoothingDisabled -bool NO
#defaults -currentHost write -globalDomain AppleFontSmoothing -int 2

# disable saving of window status on power off
sudo defaults write com.apple.loginwindow TALLogoutSavesState -bool false
defaults write -g NSQuitAlwaysKeepsWindows -bool false
defaults write -g ApplePersistence -bool no

# disable save state of Alacritty
mkdir -p "$HOME/Library/Saved Application State/org.alacritty.savedState/"
rm -rf "$HOME/Library/Saved Application State/org.alacritty.savedState/*"
chmod -R a-w "$HOME/Library/Saved Application State/org.alacritty.savedState/"

# screenshots in PNG format
defaults write com.apple.screencapture type png

# screenshots location
mkdir -p ~/Pictures/Screenshots
defaults write com.apple.screencapture location ~/Pictures/Screenshots

# reload process
killall Dock Finder cfprefsd
echo "Run: killall Terminal"
echo "Run: sudo killall loginwindow"
