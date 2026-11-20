#!/usr/bin/env bash
# ==============================================================================
# scripts/macos_osconfig.sh
# Modern, idempotent macOS system configuration (Sonoma 14 through macOS 27 max).
#
# Replaces archived legacy scripts (archives/src/osconfig/macos/*):
#   - Touch ID sudo via update-safe /etc/pam.d/sudo_local + pam_reattach (Ghostty + Herdr)
#   - macOS Sonoma (14) / Sequoia (15) / Tahoe (26) / Golden Gate (27) WindowManager, Control Center, Dock, Finder, Trackpad
#   - Containerized Safari developer/privacy preferences (with Full Disk Access check)
#   - Mission Control Spaces symbolic hotkeys
#   - Optional Apple Silicon power & WindowServer tuning (--power, --reduce-ui-fx, --reset-launchservices)
#   - Optional interactive Apple Shortcuts import (files/macos-shortcuts/*.shortcut)
#
# Usage:
#   ./scripts/macos_osconfig.sh [options]
# ==============================================================================

set -euo pipefail

readonly MACOS_MIN_VERSION=14
readonly MACOS_MAX_VERSION=27

DO_DEFAULTS=0
DO_SAFARI=0
DO_SPACES=0
DO_TOUCHID=0
DO_POWER=0
DO_REDUCE_UI_FX=0
DO_RESET_LAUNCHSERVICES=0
DO_SHORTCUTS=0
DO_RESTART=1
USE_SUDO=1
EXPLICIT_NO_SUDO=0
FORCE_VERSION=0
DRY_RUN=0
ANY_SECTION=0

SUDO_KEEPALIVE_PID=""
TMP_FILES=()

cleanup() {
	if [[ -n "$SUDO_KEEPALIVE_PID" ]]; then
		kill "$SUDO_KEEPALIVE_PID" >/dev/null 2>&1 || true
		wait "$SUDO_KEEPALIVE_PID" >/dev/null 2>&1 || true
	fi
	if [[ ${#TMP_FILES[@]} -gt 0 ]]; then
		rm -f "${TMP_FILES[@]}" >/dev/null 2>&1 || true
	fi
}
trap cleanup EXIT INT TERM

log_info() {
	printf '[macos-osconfig] %s\n' "$*"
}

log_warn() {
	printf '[macos-osconfig] WARNING: %s\n' "$*" >&2
}

log_err() {
	printf '[macos-osconfig] ERROR: %s\n' "$*" >&2
}

usage() {
	cat <<'EOF'
Usage: macos_osconfig.sh [options]

Configures macOS system preferences, Safari, Spaces hotkeys, and Touch ID sudo
for macOS 14 (Sonoma) up to macOS 27 maximum, tuned for Ghostty, Herdr, and Amethyst.

Sections (default when none specified: --defaults --safari --spaces --touchid):
  --all                   Apply all core sections plus --power and interactive --shortcuts import
  --defaults              Apply UI/UX, WindowManager, Keyboard, Trackpad, Finder, Dock, apps
  --safari                Apply containerized Safari developer & privacy settings
  --spaces                Enable Mission Control desktop switching symbolic hotkeys
  --touchid               Configure /etc/pam.d/sudo_local with pam_reattach + pam_tid (Ghostty + Herdr)
  --power                 Configure safe Apple Silicon AC/battery performance (pmset lowpowermode 0)
  --reduce-ui-fx          Enable reduceTransparency + reduceMotion in Accessibility (opt-in)
  --reset-launchservices  Rebuild LaunchServices database to deduplicate Open With entries (opt-in)
  --shortcuts             Open bundled signed .shortcut files in Shortcuts.app for import

Modifiers:
  --no-sudo               Skip commands requiring sudo privileges (e.g. sudo_local, pmset, loginwindow)
  --no-restart            Do not restart cfprefsd, WindowManager, ControlCenter, SystemUIServer, Dock, Finder, Safari, Activity Monitor
  --force                 Proceed even if detected macOS major version exceeds 27
  --dry-run               Print actions without mutating system state
  -h, --help              Show this help message
EOF
}

while [[ $# -gt 0 ]]; do
	case "$1" in
	--all)
		DO_DEFAULTS=1
		DO_SAFARI=1
		DO_SPACES=1
		DO_TOUCHID=1
		DO_POWER=1
		DO_SHORTCUTS=1
		ANY_SECTION=1
		;;
	--defaults)
		DO_DEFAULTS=1
		ANY_SECTION=1
		;;
	--safari)
		DO_SAFARI=1
		ANY_SECTION=1
		;;
	--spaces)
		DO_SPACES=1
		ANY_SECTION=1
		;;
	--touchid)
		DO_TOUCHID=1
		ANY_SECTION=1
		;;
	--power)
		DO_POWER=1
		ANY_SECTION=1
		;;
	--reduce-ui-fx)
		DO_REDUCE_UI_FX=1
		ANY_SECTION=1
		;;
	--reset-launchservices)
		DO_RESET_LAUNCHSERVICES=1
		ANY_SECTION=1
		;;
	--shortcuts)
		DO_SHORTCUTS=1
		ANY_SECTION=1
		;;
	--no-sudo)
		USE_SUDO=0
		EXPLICIT_NO_SUDO=1
		;;
	--no-restart)
		DO_RESTART=0
		;;
	--force)
		FORCE_VERSION=1
		;;
	--dry-run)
		DRY_RUN=1
		;;
	-h | --help)
		usage
		exit 0
		;;
	*)
		log_err "Unknown option: $1"
		usage >&2
		exit 2
		;;
	esac
	shift
done

if [[ "$ANY_SECTION" -eq 0 ]]; then
	DO_DEFAULTS=1
	DO_SAFARI=1
	DO_SPACES=1
	DO_TOUCHID=1
fi

run_cmd() {
	if [[ "$DRY_RUN" -eq 1 ]]; then
		printf '[dry-run]'
		printf ' %q' "$@"
		printf '\n'
		return 0
	fi
	"$@"
}

run_sudo() {
	if [[ "$USE_SUDO" -eq 0 ]]; then
		log_info "Skipping privileged command (--no-sudo): $*"
		return 0
	fi
	if [[ "$DRY_RUN" -eq 1 ]]; then
		printf '[dry-run] sudo'
		printf ' %q' "$@"
		printf '\n'
		return 0
	fi
	sudo "$@"
}

defaults_write() {
	run_cmd defaults write "$@"
}

defaults_curhost_write() {
	run_cmd defaults -currentHost write "$@"
}

FDA_WARNED=0
defaults_write_tcc() {
	if [[ "$DRY_RUN" -eq 1 ]]; then
		run_cmd defaults write "$@"
		return 0
	fi
	if ! defaults write "$@" >/dev/null 2>&1; then
		if [[ "$FDA_WARNED" -eq 0 ]]; then
			log_warn "Skipped TCC-protected domain ($1). Grant Full Disk Access to Ghostty (System Settings > Privacy & Security > Full Disk Access) and re-run."
			FDA_WARNED=1
		fi
	fi
}

OPTIONAL_WARNED_DOMAINS="|"
defaults_write_optional() {
	if [[ "$DRY_RUN" -eq 1 ]]; then
		run_cmd defaults write "$@"
		return 0
	fi
	if ! defaults write "$@" >/dev/null 2>&1; then
		case "$OPTIONAL_WARNED_DOMAINS" in
			*"|$1|"*) ;;
			*)
				OPTIONAL_WARNED_DOMAINS="${OPTIONAL_WARNED_DOMAINS}$1|"
				log_warn "Skipped sandboxed/unavailable defaults domain ($1)."
				;;
		esac
	fi
}

quit_app_if_running() {
	local app_name="$1"
	if [[ "$app_name" == "Terminal" && "${TERM_PROGRAM:-}" == "Apple_Terminal" ]]; then
		log_warn "Skipping quit for Terminal while running inside active Apple Terminal session."
		return 0
	fi
	if [[ "$DRY_RUN" -eq 1 ]]; then
		run_cmd osascript -e "if application \"${app_name}\" is running then tell application \"${app_name}\" to quit"
		return 0
	fi
	osascript -e "if application \"${app_name}\" is running then tell application \"${app_name}\" to quit" >/dev/null 2>&1 || true
}

resolve_plist_buddy() {
	if command -v PlistBuddy >/dev/null 2>&1; then
		command -v PlistBuddy
		return 0
	fi
	printf '%s\n' "/usr/libexec/PlistBuddy"
}

resolve_activate_settings() {
	if command -v activateSettings >/dev/null 2>&1; then
		command -v activateSettings
		return 0
	fi
	printf '%s\n' "/System/Library/PrivateFrameworks/SystemAdministration.framework/Resources/activateSettings"
}

flush_cfprefsd() {
	if [[ "$DO_RESTART" -eq 0 ]]; then
		return 0
	fi
	if [[ "$DRY_RUN" -eq 1 ]]; then
		run_cmd killall "cfprefsd"
		return 0
	fi
	killall "cfprefsd" >/dev/null 2>&1 || true
	sleep 1
}

plist_set() {
	local plist_path="$1"
	local key_path="$2"
	local value_type="$3"
	local value="$4"
	local pb
	pb="$(resolve_plist_buddy)"

	if [[ "$DRY_RUN" -eq 1 ]]; then
		printf '[dry-run] %s %s=%s (%s) in %s\n' "$pb" "$key_path" "$value" "$value_type" "$plist_path"
		return 0
	fi
	if [[ ! -f "$plist_path" ]]; then
		return 0
	fi
	if "$pb" -c "Print ${key_path}" "$plist_path" >/dev/null 2>&1; then
		"$pb" -c "Set ${key_path} ${value}" "$plist_path" >/dev/null 2>&1 || log_warn "Could not set ${key_path} in ${plist_path}."
	else
		local parent_path="${key_path%:*}"
		if [[ -n "$parent_path" && "$parent_path" != "$key_path" ]]; then
			if ! "$pb" -c "Print ${parent_path}" "$plist_path" >/dev/null 2>&1; then
				local grandparent_path="${parent_path%:*}"
				if [[ -n "$grandparent_path" && "$grandparent_path" != "$parent_path" ]] && ! "$pb" -c "Print ${grandparent_path}" "$plist_path" >/dev/null 2>&1; then
					"$pb" -c "Add ${grandparent_path} dict" "$plist_path" >/dev/null 2>&1 || true
				fi
				"$pb" -c "Add ${parent_path} dict" "$plist_path" >/dev/null 2>&1 || true
			fi
		fi
		"$pb" -c "Add ${key_path} ${value_type} ${value}" "$plist_path" >/dev/null 2>&1 || log_warn "Could not add ${key_path} in ${plist_path}."
	fi
}

check_platform() {
	local os_name version major
	os_name="$(uname -s)"
	if [[ "$os_name" != "Darwin" ]]; then
		log_err "This script only supports macOS (detected OS: $os_name)."
		exit 1
	fi

	version="$(sw_vers -productVersion)"
	major="${version%%.*}"
	log_info "Detected macOS $version ($(uname -m))"

	if ! [[ "$major" =~ ^[0-9]+$ ]]; then
		log_err "Could not parse macOS major version from '$version'."
		exit 1
	fi
	if [[ "$major" -lt "$MACOS_MIN_VERSION" ]]; then
		log_warn "macOS $version is older than tested minimum (macOS $MACOS_MIN_VERSION); some modern keys (e.g. sudo_local, WindowManager) may not apply."
	fi
	if [[ "$major" -gt "$MACOS_MAX_VERSION" ]]; then
		if [[ "$FORCE_VERSION" -eq 1 ]]; then
			log_warn "macOS $version exceeds validated maximum (macOS $MACOS_MAX_VERSION); continuing due to --force."
		else
			log_err "macOS $version exceeds supported maximum (macOS $MACOS_MAX_VERSION). Audit new defaults or re-run with --force."
			exit 1
		fi
	fi
}

ensure_homebrew_env() {
	if [[ -x "/opt/homebrew/bin/brew" ]]; then
		eval "$(/opt/homebrew/bin/brew shellenv)"
	elif [[ -x "/usr/local/bin/brew" ]]; then
		eval "$(/usr/local/bin/brew shellenv)"
	fi
}

init_sudo_keepalive() {
	if [[ "$USE_SUDO" -eq 0 || "$DRY_RUN" -eq 1 ]]; then
		return 0
	fi
	if ! sudo -v; then
		log_err "sudo authentication failed; re-run with cached sudo or pass --no-sudo to skip privileged steps."
		return 1
	fi
	(
		sleep_pid=""
		trap '[[ -n "${sleep_pid:-}" ]] && kill "${sleep_pid}" >/dev/null 2>&1 || true; exit 0' EXIT INT TERM
		while true; do
			sudo -n true >/dev/null 2>&1 || exit 0
			sleep 60 &
			sleep_pid="$!"
			wait "${sleep_pid}" >/dev/null 2>&1 || exit 0
			sleep_pid=""
			kill -0 "$$" >/dev/null 2>&1 || exit 0
		done
	) </dev/null >/dev/null 2>&1 &
	SUDO_KEEPALIVE_PID="$!"
}

find_pam_reattach() {
	local arch prefix
	arch="$(uname -m)"
	if command -v brew >/dev/null 2>&1; then
		prefix="$(brew --prefix pam-reattach 2>/dev/null || brew --prefix 2>/dev/null || true)"
		if [[ -n "$prefix" && -f "${prefix}/lib/pam/pam_reattach.so" ]]; then
			printf '%s\n' "${prefix}/lib/pam/pam_reattach.so"
			return 0
		fi
	fi
	if [[ "$arch" == "arm64" && -f "/opt/homebrew/lib/pam/pam_reattach.so" ]]; then
		printf '%s\n' "/opt/homebrew/lib/pam/pam_reattach.so"
		return 0
	fi
	if [[ -f "/usr/local/lib/pam/pam_reattach.so" ]]; then
		printf '%s\n' "/usr/local/lib/pam/pam_reattach.so"
		return 0
	fi
	return 1
}

configure_touchid_sudo() {
	log_info "Configuring Touch ID for sudo via /etc/pam.d/sudo_local (Ghostty + Herdr compatible)..."
	ensure_homebrew_env

	if [[ "$USE_SUDO" -eq 0 ]]; then
		log_warn "Skipping Touch ID sudo setup because --no-sudo was passed."
		return 0
	fi

	local pam_reattach_path=""
	if ! pam_reattach_path="$(find_pam_reattach 2>/dev/null)"; then
		if [[ "${ANX_OSCONFIG_NO_BREW:-0}" != "1" ]] && command -v brew >/dev/null 2>&1; then
			log_info "Installing pam-reattach via Homebrew (bridges Herdr/tmux daemon sessions to Aqua Touch ID)..."
			if ! run_cmd brew install pam-reattach; then
				log_warn "Failed to install pam-reattach via Homebrew; Touch ID inside detached Herdr/tmux sessions may require pam_reattach.so."
			fi
			pam_reattach_path="$(find_pam_reattach 2>/dev/null || true)"
		else
			log_warn "Homebrew not found; configuring native pam_tid.so (Touch ID works directly in Ghostty, install pam-reattach for detached Herdr/tmux sessions)."
		fi
	fi

	local desired_content tmp_sudo_local extra_pam_lines="" line
	desired_content="# sudo_local: local config file which survives system updates (managed by anx macos_osconfig)"$'\n'
	# pam_reattach must precede pam_tid.so with 'optional' so SSH/root logins never fail while Ghostty+Herdr sessions re-attach to Aqua GUI.
	if [[ -n "$pam_reattach_path" ]]; then
		desired_content+="auth       optional       ${pam_reattach_path}"$'\n'
	fi
	desired_content+="auth       sufficient     pam_tid.so"$'\n'
	if [[ -r /etc/pam.d/sudo_local ]]; then
		while IFS= read -r line || [[ -n "$line" ]]; do
			case "$line" in
			""|"# sudo_local:"*|*pam_reattach.so*|*pam_tid.so*) ;;
			*) extra_pam_lines+="${line}"$'\n' ;;
			esac
		done < /etc/pam.d/sudo_local
		desired_content+="$extra_pam_lines"
	fi

	if [[ "$DRY_RUN" -eq 1 ]]; then
		log_info "[dry-run] Would ensure /etc/pam.d/sudo_local contains:"
		printf '%s' "$desired_content"
		return 0
	fi

	if [[ -r /etc/pam.d/sudo_local ]] && [[ "$(cat /etc/pam.d/sudo_local)"$'\n' == "$desired_content" || "$(cat /etc/pam.d/sudo_local)" == "${desired_content%$'\n'}" ]]; then
		log_info "/etc/pam.d/sudo_local is already up to date."
		return 0
	fi

	if [[ -f /etc/pam.d/sudo_local ]]; then
		local backup_path="/etc/pam.d/.sudo_local.anx.bak"
		if [[ ! -f "$backup_path" ]]; then
			run_sudo cp /etc/pam.d/sudo_local "$backup_path"
		fi
	fi

	tmp_sudo_local="$(mktemp "${TMPDIR:-/tmp}/anx-sudo_local.XXXXXX")"
	TMP_FILES+=("$tmp_sudo_local")
	printf '%s' "$desired_content" >"$tmp_sudo_local"
	run_sudo install -m 0644 "$tmp_sudo_local" /etc/pam.d/sudo_local
	log_info "Touch ID sudo configured in /etc/pam.d/sudo_local."
}

configure_defaults() {
	log_info "Applying macOS UI/UX, WindowManager, Dock, Finder, Trackpad & Keyboard defaults..."

	# Quit System Settings if running so cached GUI panes do not overwrite written preferences.
	quit_app_if_running "System Settings"

	# General UI/UX & Power
	defaults_write NSGlobalDomain NSTableViewDefaultSizeMode -int 2
	defaults_write NSGlobalDomain AppleSpacesSwitchOnActivate -bool true
	defaults_write NSGlobalDomain NSCloseAlwaysConfirmsChanges -bool true
	defaults_write NSGlobalDomain AppleShowScrollBars -string "Always"
	defaults_write NSGlobalDomain NSUseAnimatedFocusRing -bool false
	defaults_write NSGlobalDomain NSScrollAnimationEnabled -bool false
	defaults_write NSGlobalDomain NSNavPanelExpandedStateForSaveMode -bool true
	defaults_write NSGlobalDomain NSNavPanelExpandedStateForSaveMode2 -bool true
	defaults_write NSGlobalDomain PMPrintingExpandedStateForPrint -bool true
	defaults_write NSGlobalDomain PMPrintingExpandedStateForPrint2 -bool true
	defaults_write NSGlobalDomain NSDocumentSaveNewDocumentsToCloud -bool false
	defaults_write NSGlobalDomain NSTextShowsControlCharacters -bool true
	defaults_write NSGlobalDomain NSQuitAlwaysKeepsWindows -bool false
	defaults_write NSGlobalDomain NSDisableAutomaticTermination -bool true
	defaults_write NSGlobalDomain com.apple.keyboard.fnState -bool false

	# Keep Gatekeeper quarantine enabled for downloaded apps; only skip Cocoa persistence & printer queue linger.
	defaults_write_optional com.apple.print.PrintingPrefs "Quit When Finished" -bool true
	defaults_write_optional com.apple.helpviewer DevMode -bool true

	# Disable typing autocorrections globally (better for terminal/code editing).
	defaults_write NSGlobalDomain NSAutomaticCapitalizationEnabled -bool false
	defaults_write NSGlobalDomain NSAutomaticDashSubstitutionEnabled -bool false
	defaults_write NSGlobalDomain NSAutomaticPeriodSubstitutionEnabled -bool false
	defaults_write NSGlobalDomain NSAutomaticQuoteSubstitutionEnabled -bool false
	defaults_write NSGlobalDomain NSAutomaticSpellingCorrectionEnabled -bool false

	# Keyboard & Full Keyboard Access
	defaults_write NSGlobalDomain KeyRepeat -int 2
	defaults_write NSGlobalDomain InitialKeyRepeat -int 15
	defaults_write NSGlobalDomain ApplePressAndHoldEnabled -bool false
	defaults_write NSGlobalDomain AppleKeyboardUIMode -int 2

	# Locale & Units
	defaults_write NSGlobalDomain AppleLanguages -array "en" "es"
	defaults_write NSGlobalDomain AppleLocale -string "en_US@currency=USD"
	defaults_write NSGlobalDomain AppleMeasurementUnits -string "Centimeters"
	defaults_write NSGlobalDomain AppleMetricUnits -bool true

	# WindowManager (macOS 14/15/26/27): prevent native tiled window gaps & accidental click-to-reveal-desktop when using Amethyst.
	defaults_write com.apple.WindowManager EnableTiledWindowMargins -bool false
	defaults_write com.apple.WindowManager EnableStandardClickToShowDesktop -bool false
	defaults_write com.apple.WindowManager GloballyEnabled -bool false

	# Control Center & Menu Bar Clock
	defaults_curhost_write com.apple.controlcenter BatteryShowPercentage -bool true
	defaults_curhost_write com.apple.controlcenter "NSStatusItem Visible Battery" -bool true
	defaults_curhost_write com.apple.controlcenter "NSStatusItem Visible WiFi" -bool true
	defaults_write com.apple.menuextra.clock IsAnalog -bool false
	defaults_write com.apple.menuextra.clock Show24Hour -bool false
	defaults_write com.apple.menuextra.clock ShowDayOfMonth -bool false
	defaults_write com.apple.menuextra.clock ShowDayOfWeek -bool false
	defaults_write com.apple.menuextra.clock FlashDateSeparators -bool false
	defaults_write com.apple.menuextra.clock ShowSeconds -bool true
	defaults_curhost_write com.apple.controlcenter "NSStatusItem Visible Siri" -bool false
	defaults_write com.apple.Siri StatusMenuVisible -bool false

	# Trackpad (Three-Finger Drag + non-natural scroll + Ctrl+scroll zoom)
	defaults_write com.apple.AppleMultitouchTrackpad Dragging -bool false
	defaults_write com.apple.AppleMultitouchTrackpad DragLock -bool false
	defaults_write com.apple.AppleMultitouchTrackpad TrackpadThreeFingerDrag -bool true
	defaults_write com.apple.AppleMultitouchTrackpad FirstClickThreshold -int 1
	defaults_write com.apple.driver.AppleBluetoothMultitouch.trackpad TrackpadThreeFingerDrag -bool true
	defaults_write com.apple.driver.AppleBluetoothMultitouch.trackpad Clicking -bool false
	defaults_curhost_write NSGlobalDomain com.apple.mouse.tapBehavior -int 0
	defaults_write NSGlobalDomain com.apple.mouse.tapBehavior -int 0
	defaults_write NSGlobalDomain com.apple.swipescrolldirection -bool false
	defaults_write_tcc com.apple.universalaccess closeViewScrollWheelToggle -bool true
	defaults_write_tcc com.apple.universalaccess HIDScrollZoomModifierMask -int 262144
	defaults_write_tcc com.apple.universalaccess closeViewZoomFollowsFocus -bool true
	defaults_write_tcc com.apple.universalaccess showWindowTitlebarIcons -bool true
	defaults_write NSGlobalDomain NSToolbarTitleViewRolloverDelay -float 0

	# Finder
	defaults_write NSGlobalDomain AppleShowAllExtensions -bool true
	defaults_write com.apple.finder CreateDesktop -bool true
	defaults_write com.apple.finder AppleShowAllFiles -bool true
	defaults_write com.apple.finder ShowPathbar -bool true
	defaults_write com.apple.finder ShowStatusBar -bool true
	defaults_write com.apple.finder FXDefaultSearchScope -string "SCcf"
	defaults_write com.apple.finder _FXSortFoldersFirst -bool true
	defaults_write com.apple.finder _FXSortFoldersFirstOnDesktop -bool true
	defaults_write com.apple.finder FXEnableExtensionChangeWarning -bool false
	defaults_write com.apple.finder FXPreferredViewStyle -string "Nlsv"
	defaults_write com.apple.finder FXPreferredGroupBy -string "Kind"
	defaults_write com.apple.finder NewWindowTarget -string "PfHm"
	defaults_write com.apple.finder ShowExternalHardDrivesOnDesktop -bool true
	defaults_write com.apple.finder ShowHardDrivesOnDesktop -bool true
	defaults_write com.apple.finder ShowMountedServersOnDesktop -bool true
	defaults_write com.apple.finder ShowRemovableMediaOnDesktop -bool true
	defaults_write com.apple.finder "FXRemoveOldTrashItems" -bool false
	defaults_write com.apple.finder FXInfoPanesExpanded -dict General -bool true OpenWith -bool true Privileges -bool true
	defaults_write NSGlobalDomain com.apple.springing.enabled -bool true
	defaults_write NSGlobalDomain com.apple.springing.delay -float 0
	defaults_write com.apple.desktopservices DSDontWriteNetworkStores -bool true
	run_cmd chflags nohidden "${HOME}/Library" 2>/dev/null || true

	# Close Finder before direct PlistBuddy mutations so cached preferences do not overwrite disk writes.
	quit_app_if_running "Finder"
	flush_cfprefsd
	local finder_plist="${HOME}/Library/Preferences/com.apple.finder.plist"
	local view_group
	for view_group in DesktopViewSettings FK_StandardViewSettings StandardViewSettings; do
		plist_set "$finder_plist" ":${view_group}:IconViewSettings:showItemInfo" bool true
		plist_set "$finder_plist" ":${view_group}:IconViewSettings:arrangeBy" string grid
		plist_set "$finder_plist" ":${view_group}:IconViewSettings:gridSpacing" integer 65
		plist_set "$finder_plist" ":${view_group}:IconViewSettings:iconSize" integer 80
	done
	plist_set "$finder_plist" ":DesktopViewSettings:IconViewSettings:labelOnBottom" bool false
	plist_set "$finder_plist" ":StandardViewSettings:IconViewSettings:arrangeBy" string kind
	plist_set "$finder_plist" ":StandardViewSettings:ExtendedListViewSettingsV2:sortColumn" string dateLastOpened

	# Dock & Mission Control (including mru-spaces=false required by Amethyst)
	defaults_write com.apple.dock orientation -string "left"
	defaults_write com.apple.dock tilesize -int 45
	defaults_write com.apple.dock magnification -bool true
	defaults_write com.apple.dock largesize -int 55
	defaults_write com.apple.dock mouse-over-hilite-stack -bool true
	defaults_write com.apple.dock autohide -bool true
	defaults_write com.apple.dock autohide-delay -float 0
	defaults_write com.apple.dock autohide-time-modifier -float 0.8
	defaults_write com.apple.dock expose-animation-duration -float 0.05
	defaults_write com.apple.dock show-recents -bool false
	defaults_write com.apple.dock minimize-to-application -bool true
	defaults_write com.apple.dock mineffect -string "scale"
	defaults_write com.apple.dock show-process-indicators -bool true
	defaults_write com.apple.dock enable-spring-load-actions-on-all-items -bool true
	defaults_write com.apple.dock expose-group-by-app -bool false
	defaults_write com.apple.dock mru-spaces -bool false
	defaults_write com.apple.dock static-only -bool false
	defaults_write com.apple.dock showhidden -bool true
	defaults_write com.apple.spaces spans-displays -bool false
	defaults_write com.apple.dock wvous-tl-corner -int 2
	defaults_write com.apple.dock wvous-tl-modifier -int 0
	defaults_write com.apple.dock wvous-tr-corner -int 12
	defaults_write com.apple.dock wvous-tr-modifier -int 0

	# Screenshots
	run_cmd mkdir -p "${HOME}/Pictures/Screenshots"
	defaults_write com.apple.screencapture type -string "png"
	defaults_write com.apple.screencapture location -string "${HOME}/Pictures/Screenshots"

	# Ghostty, Terminal.app, Activity Monitor, TextEdit, Disk Utility, Photos, Time Machine, SoftwareUpdate, Chrome
	# Keep Ghostty/Herdr/Neovim background builds on performance cores; leave App Nap enabled globally for battery life.
	defaults_write com.mitchellh.ghostty NSQuitAlwaysKeepsWindows -bool false
	defaults_write com.mitchellh.ghostty NSAppSleepDisabled -bool true
	defaults_write_optional com.apple.terminal StringEncodings -array 4
	defaults_write_optional com.apple.terminal SecureKeyboardEntry -bool true
	defaults_write_optional com.apple.terminal FocusFollowsMouse -bool false
	if [[ "${TERM_PROGRAM:-}" != "Apple_Terminal" ]]; then
		quit_app_if_running "Terminal"
		flush_cfprefsd
		plist_set "${HOME}/Library/Preferences/com.apple.Terminal.plist" ':"Window Settings":Basic:shellExitAction' integer 0
	fi

	defaults_write_optional com.apple.ActivityMonitor OpenMainWindow -bool true
	defaults_write_optional com.apple.ActivityMonitor IconType -int 5
	defaults_write_optional com.apple.ActivityMonitor ShowCategory -int 0
	defaults_write_optional com.apple.ActivityMonitor SortColumn -string "CPUUsage"
	defaults_write_optional com.apple.ActivityMonitor SortDirection -int 0
	defaults_write_optional com.apple.ActivityMonitor UpdatePeriod -int 5

	defaults_write_optional com.apple.TextEdit RichText -int 0
	defaults_write_optional com.apple.TextEdit PlainTextEncoding -int 4
	defaults_write_optional com.apple.TextEdit PlainTextEncodingForWrite -int 4
	defaults_write_optional com.apple.TextEdit SmartQuotes -bool false

	defaults_write_optional com.apple.DiskUtility DUDebugMenuEnabled -bool true
	defaults_write_optional com.apple.DiskUtility advanced-image-options -bool true
	defaults_write_optional com.apple.QuickTimePlayerX MGPlayMovieOnOpen -bool true
	defaults_curhost_write com.apple.ImageCapture disableHotPlug -bool true
	defaults_write_optional com.apple.TimeMachine DoNotOfferNewDisksForBackup -bool false
	defaults_write_optional com.apple.Music userWantsPlaybackNotifications -bool true
	defaults_write_optional com.apple.SoftwareUpdate AutomaticCheckEnabled -bool true
	defaults_write_optional com.apple.SoftwareUpdate AutomaticDownload -bool true
	defaults_write_optional com.apple.SoftwareUpdate CriticalUpdateInstall -bool true

	# Browser & Xcode tweaks
	defaults_write_optional com.google.Chrome AppleEnableSwipeNavigateWithScrolls -bool false
	defaults_write_optional com.google.Chrome AppleEnableMouseSwipeNavigateWithScrolls -bool false
	defaults_write_optional com.google.Chrome DisablePrintPreview -bool true
	defaults_write_optional com.google.Chrome PMPrintingExpandedStateForPrint2 -bool true
	defaults_write_optional com.apple.dt.Xcode ShowBuildOperationDuration -bool true

	# System-wide login window & volumes (privileged)
	run_sudo defaults write /Library/Preferences/com.apple.loginwindow AdminHostInfo HostName || true
	run_sudo defaults write /Library/Preferences/com.apple.loginwindow SHOWFULLNAME -int 0 || true
	run_sudo defaults write /Library/Preferences/com.apple.loginwindow TALLogoutSavesState -bool false || true
	run_sudo defaults write /Library/Preferences/com.apple.loginwindow showInputMenu -bool true || true
	run_sudo chflags nohidden /Volumes >/dev/null 2>&1 || true
}

configure_safari() {
	log_info "Configuring Safari preferences (with sandbox container & Full Disk Access check)..."
	local safari_container_data="${HOME}/Library/Containers/com.apple.Safari/Data"
	local safari_plist="${safari_container_data}/Library/Preferences/com.apple.Safari.plist"
	local probe_err=""

	if [[ "$DRY_RUN" -eq 0 ]]; then
		quit_app_if_running "Safari"
		if [[ -d "${HOME}/Library/Containers/com.apple.Safari" ]]; then
			if ! probe_err="$(ls "$safari_container_data" 2>&1 >/dev/null)"; then
				if [[ "$probe_err" == *"Operation not permitted"* || "$probe_err" == *"Permission denied"* ]]; then
					log_warn "Cannot access Safari sandbox container (${safari_plist}). Grant Full Disk Access to Ghostty (System Settings > Privacy & Security > Full Disk Access) and re-run --safari."
					return 0
				fi
			fi
		fi
	fi

	if [[ ! -e "$safari_plist" ]]; then
		log_warn "Safari container plist not found (${safari_plist}). Open Safari once and re-run --safari instead of writing outside the sandbox container."
		return 0
	fi

	if [[ "$DRY_RUN" -eq 0 ]] && ! defaults read "$safari_plist" >/dev/null 2>&1; then
		log_warn "Cannot access Safari sandbox container (${safari_plist}). Grant Full Disk Access to Ghostty (System Settings > Privacy & Security > Full Disk Access) and re-run --safari."
		return 0
	fi

	defaults_write_tcc "$safari_plist" UniversalSearchEnabled -bool false
	defaults_write_tcc "$safari_plist" SuppressSearchSuggestions -bool true
	defaults_write_tcc "$safari_plist" WebKitTabToLinksPreferenceKey -bool true
	defaults_write_tcc "$safari_plist" com.apple.Safari.ContentPageGroupIdentifier.WebKit2TabsToLinks -bool true
	defaults_write_tcc "$safari_plist" ShowFullURLInSmartSearchField -bool true
	defaults_write_tcc "$safari_plist" HomePage -string "about:blank"
	defaults_write_tcc "$safari_plist" AutoOpenSafeDownloads -bool false
	defaults_write_tcc "$safari_plist" ShowFavoritesBar -bool true
	defaults_write_tcc "$safari_plist" FindOnPageMatchesWordStartsOnly -bool false
	defaults_write_tcc "$safari_plist" IncludeDevelopMenu -bool true
	defaults_write_tcc "$safari_plist" WebKitDeveloperExtrasEnabledPreferenceKey -bool true
	defaults_write_tcc "$safari_plist" com.apple.Safari.ContentPageGroupIdentifier.WebKit2DeveloperExtrasEnabled -bool true
	defaults_write_tcc com.apple.Safari.SandboxBroker ShowDevelopMenu -bool true
	defaults_write NSGlobalDomain WebKitDeveloperExtras -bool true
	defaults_write_tcc "$safari_plist" WebContinuousSpellCheckingEnabled -bool true
	defaults_write_tcc "$safari_plist" WebAutomaticSpellingCorrectionEnabled -bool false
	defaults_write_tcc "$safari_plist" AutoFillFromAddressBook -bool false
	defaults_write_tcc "$safari_plist" AutoFillPasswords -bool false
	defaults_write_tcc "$safari_plist" AutoFillCreditCardData -bool false
	defaults_write_tcc "$safari_plist" AutoFillMiscellaneousForms -bool false
	defaults_write_tcc "$safari_plist" WarnAboutFraudulentWebsites -bool true
	defaults_write_tcc "$safari_plist" InstallExtensionUpdatesAutomatically -bool true
}

configure_spaces() {
	log_info "Enabling Mission Control Spaces symbolic hotkeys..."
	local hotkeys_plist="${HOME}/Library/Preferences/com.apple.symbolichotkeys.plist"
	local hotkey_id
	local pb activate
	pb="$(resolve_plist_buddy)"
	activate="$(resolve_activate_settings)"
	local existing_ids=()
	local added_missing_hotkeys=0

	# Phase 1: create missing entries via cfprefsd first so cached writes never clobber subsequent PlistBuddy disk edits.
	for hotkey_id in 79 80 81 82 98 118 119 120 121 122 123 124 125 126 127; do
		if [[ -f "$hotkeys_plist" ]] && "$pb" -c "Print :AppleSymbolicHotKeys:${hotkey_id}" "$hotkeys_plist" >/dev/null 2>&1; then
			existing_ids+=("$hotkey_id")
		elif [[ "$DRY_RUN" -eq 1 ]]; then
			run_cmd defaults write com.apple.symbolichotkeys AppleSymbolicHotKeys -dict-add "$hotkey_id" "<dict><key>enabled</key><true/></dict>"
		else
			defaults write com.apple.symbolichotkeys AppleSymbolicHotKeys -dict-add "$hotkey_id" "<dict><key>enabled</key><true/></dict>" >/dev/null 2>&1 || true
			added_missing_hotkeys=1
		fi
	done

	if [[ "$added_missing_hotkeys" -eq 1 && ${#existing_ids[@]} -gt 0 ]]; then
		flush_cfprefsd
	fi

	# Phase 2: enable pre-existing hotkey dicts (preserving keybindings, adding :enabled if absent).
	if [[ ${#existing_ids[@]} -gt 0 ]]; then
		for hotkey_id in "${existing_ids[@]}"; do
			plist_set "$hotkeys_plist" ":AppleSymbolicHotKeys:${hotkey_id}:enabled" bool true
		done
	fi

	# Phase 3: reload cfprefsd before defaults read and activateSettings -u so fresh hotkeys take effect.
	flush_cfprefsd
	if [[ "$DRY_RUN" -eq 1 ]]; then
		run_cmd defaults read com.apple.symbolichotkeys
		run_cmd "$activate" -u
	else
		defaults read com.apple.symbolichotkeys >/dev/null 2>&1 || true
		if [[ -x "$activate" ]]; then
			"$activate" -u >/dev/null 2>&1 || true
		fi
	fi
}

configure_power() {
	log_info "Configuring safe Apple Silicon power settings (--power)..."
	if [[ "$USE_SUDO" -eq 0 ]]; then
		log_warn "Skipping power configuration because --no-sudo was passed."
		return 0
	fi
	# Allow full CPU/GPU clock scaling on AC and battery on MacBook Air (High Power Mode does not exist here).
	# Intentionally leave tcpkeepalive, powernap, swap, and thermal protections untouched for Slack, Tailscale, and Syncthing.
	run_sudo pmset -a lowpowermode 0 || log_warn "Could not update pmset lowpowermode (continuing)."
}

configure_reduce_ui_fx() {
	log_info "Applying optional Accessibility UI effects reduction (--reduce-ui-fx)..."
	defaults_write_tcc com.apple.universalaccess reduceTransparency -bool true
	defaults_write_tcc com.apple.universalaccess reduceMotion -bool true
}

reset_launchservices() {
	log_info "Rebuilding LaunchServices database (--reset-launchservices)..."
	local lsreg="/System/Library/Frameworks/CoreServices.framework/Frameworks/LaunchServices.framework/Support/lsregister"
	if [[ "$DRY_RUN" -eq 1 ]]; then
		run_cmd "$lsreg" -kill -r -domain local -domain system -domain user
		return 0
	fi
	if [[ -x "$lsreg" ]]; then
		"$lsreg" -kill -r -domain local -domain system -domain user >/dev/null 2>&1 || true
	else
		log_warn "lsregister binary not found at $lsreg."
	fi
}

import_shortcuts() {
	log_info "Importing bundled macOS Shortcuts (requires interactive GUI approval per shortcut)..."
	local script_dir repo_dir shortcuts_dir
	script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
	repo_dir="$(cd "${script_dir}/.." && pwd)"

	if [[ -d "${repo_dir}/files/macos-shortcuts" ]]; then
		shortcuts_dir="${repo_dir}/files/macos-shortcuts"
	elif [[ -d "${HOME}/.local/share/anx/macos-shortcuts" ]]; then
		shortcuts_dir="${HOME}/.local/share/anx/macos-shortcuts"
	else
		log_warn "No macos-shortcuts directory found (expected files/macos-shortcuts or ~/.local/share/anx/macos-shortcuts)."
		return 0
	fi

	local sc_file
	shopt -s nullglob
	for sc_file in "${shortcuts_dir}"/*.shortcut; do
		log_info "Opening shortcut for import: $(basename "$sc_file")"
		run_cmd open "$sc_file" || log_warn "Could not open shortcut $(basename "$sc_file")."
	done
	shopt -u nullglob
}

restart_affected_services() {
	if [[ "$DO_RESTART" -eq 0 ]]; then
		log_info "Skipping service restart (--no-restart)."
		return 0
	fi
	log_info "Restarting affected macOS UI daemons (cfprefsd first)..."
	flush_cfprefsd
	local app
	if [[ "$DRY_RUN" -eq 1 ]]; then
		for app in "WindowManager" "ControlCenter" "SystemUIServer" "Dock" "Finder" "Safari" "Activity Monitor"; do
			run_cmd killall "$app"
		done
		return 0
	fi
	for app in "WindowManager" "ControlCenter" "SystemUIServer" "Dock" "Finder" "Safari" "Activity Monitor"; do
		killall "$app" >/dev/null 2>&1 || true
	done
}

main() {
	check_platform
	if [[ "$DO_DEFAULTS" -eq 1 || "$DO_TOUCHID" -eq 1 || "$DO_POWER" -eq 1 ]]; then
		init_sudo_keepalive
	fi
	if [[ "$DO_DEFAULTS" -eq 1 ]]; then
		configure_defaults
	fi
	if [[ "$DO_REDUCE_UI_FX" -eq 1 ]]; then
		configure_reduce_ui_fx
	fi
	if [[ "$DO_SAFARI" -eq 1 ]]; then
		configure_safari
	fi
	if [[ "$DO_SPACES" -eq 1 ]]; then
		configure_spaces
	fi
	if [[ "$DO_TOUCHID" -eq 1 ]]; then
		configure_touchid_sudo
	fi
	if [[ "$DO_POWER" -eq 1 ]]; then
		configure_power
	fi
	if [[ "$DO_RESET_LAUNCHSERVICES" -eq 1 ]]; then
		reset_launchservices
	fi
	if [[ "$DO_SHORTCUTS" -eq 1 ]]; then
		import_shortcuts
	fi
	if [[ "$DO_DEFAULTS" -eq 1 || "$DO_REDUCE_UI_FX" -eq 1 || "$DO_SAFARI" -eq 1 || "$DO_SPACES" -eq 1 ]]; then
		restart_affected_services
	fi
	if [[ "$EXPLICIT_NO_SUDO" -eq 1 ]]; then
		log_warn "Completed user-level configuration; privileged steps (Touch ID sudo_local / pmset / loginwindow) were skipped (--no-sudo)."
	fi
	log_info "Done. Re-login or reboot is recommended if WindowManager/Space hotkeys need full refresh."
}

main "$@"
