#!/usr/bin/env bash
# ==============================================================================
# scripts/safe_suspend.sh
# Production Safe Suspend, Screen Standby & Power Management Utility.
#
# Optimized for Linux workstations with problematic GPU sleep/resume
# (e.g., NVIDIA Kepler GK104 / Nouveau with nouveau.pstate=1, DisplayPort 4K,
# and Xfce 4.20 on Ubuntu).
#
# Usage:
#   safe-suspend [mode]
#
# Modes:
#   lock        - Lock session and enter DPMS display standby (0% risk, instant wake)
#   s2idle      - Suspend-to-Idle (CPU low power, keeps GPU PLL alive)
#   deep / vt   - Deep S3 sleep with VT switch to prevent Nouveau channel lockups
#   hibernate   - Suspend to disk (requires swap >= RAM)
#   status      - Diagnostic health check of power states, GPU driver, and swap
# ==============================================================================

set -euo pipefail

# ANSI Colors
BOLD="\033[1m"
GREEN="\033[0;32m"
YELLOW="\033[0;33m"
CYAN="\033[0;36m"
RED="\033[0;31m"
RESET="\033[0m"

log_info()    { echo -e "  [${CYAN}INFO${RESET}] $*"; }
log_step()    { echo -e "  [${BOLD}STEP${RESET}] $*"; }
log_success() { echo -e "  [${GREEN}OK${RESET}] $*"; }
log_warn()    { echo -e "  [${YELLOW}WARN${RESET}] $*"; }
log_error()   { echo -e "  [${RED}ERROR${RESET}] $*" >&2; }

detect_gpu_driver() {
	if lspci -nnk 2>/dev/null | grep -A2 -i "vga\|3d" | grep -qi "nouveau"; then
		echo "nouveau"
	elif lspci -nnk 2>/dev/null | grep -A2 -i "vga\|3d" | grep -qi "nvidia"; then
		echo "nvidia"
	elif lspci -nnk 2>/dev/null | grep -A2 -i "vga\|3d" | grep -qi "i915\|iris\|xe"; then
		echo "intel"
	elif lspci -nnk 2>/dev/null | grep -A2 -i "vga\|3d" | grep -qi "amdgpu\|radeon"; then
		echo "amd"
	else
		echo "unknown"
	fi
}

has_nouveau_pstate_quirk() {
	if grep -qi "nouveau.pstate=1" /proc/cmdline 2>/dev/null; then
		return 0
	fi
	return 1
}

lock_desktop_session() {
	log_step "Locking graphical desktop session..."
	if command -v xflock4 >/dev/null 2>&1; then
		xflock4 2>/dev/null &
	elif command -v light-locker-command >/dev/null 2>&1; then
		light-locker-command -l 2>/dev/null &
	elif command -v loginctl >/dev/null 2>&1; then
		loginctl lock-session 2>/dev/null || true
	elif command -v xdg-screensaver >/dev/null 2>&1; then
		xdg-screensaver lock 2>/dev/null || true
	fi
}

get_active_vt() {
	if [[ -r /sys/devices/virtual/tty/tty0/active ]]; then
		local vt
		vt="$(cat /sys/devices/virtual/tty/tty0/active)"
		echo "${vt#tty}"
	else
		echo "7"
	fi
}

# ------------------------------------------------------------------------------
# Action: Diagnostics & Status
# ------------------------------------------------------------------------------
do_status() {
	echo -e "${BOLD}==============================================================================${RESET}"
	echo -e "${BOLD}  ANX Safe Power & Suspend Diagnostics${RESET}"
	echo -e "${BOLD}==============================================================================${RESET}"

	local driver
	driver="$(detect_gpu_driver)"
	echo -e "  • GPU Driver in use  : ${BOLD}${driver}${RESET}"

	if has_nouveau_pstate_quirk; then
		echo -e "  • Kepler P-State     : ${RED}nouveau.pstate=1 detected${RESET} (ACPI S3 deep sleep causes VRAM PLL deadlock)"
	else
		echo -e "  • Kepler P-State     : Standard (no manual pstate override detected)"
	fi

	local mem_sleep="unknown"
	if [[ -f /sys/power/mem_sleep ]]; then
		mem_sleep="$(cat /sys/power/mem_sleep)"
	fi
	echo -e "  • Memory Sleep Modes : ${mem_sleep}"

	local active_vt
	active_vt="$(get_active_vt)"
	echo -e "  • Active X11 Console : tty${active_vt}"

	# Check memory and swap
	local ram_total_mb swap_total_mb
	ram_total_mb=$(free -m | awk '/^Mem:/{print $2}')
	swap_total_mb=$(free -m | awk '/^Swap:/{print $2}')
	echo -e "  • Physical RAM       : ${ram_total_mb} MB"
	echo -e "  • Swap Allocated     : ${swap_total_mb} MB"

	if [[ "$swap_total_mb" -ge "$ram_total_mb" ]]; then
		echo -e "  • Hibernation Ready  : ${GREEN}YES${RESET} (Swap >= RAM)"
	else
		echo -e "  • Hibernation Ready  : ${YELLOW}NO${RESET} (Swap ${swap_total_mb}MB < RAM ${ram_total_mb}MB)"
	fi

	echo ""
	echo -e "${BOLD}Recommended Safe Sleep Strategy:${RESET}"
	if [[ "$driver" == "nouveau" ]] && has_nouveau_pstate_quirk; then
		echo -e "  ${YELLOW}⚠ S3 Deep sleep (systemctl suspend) fails on this GPU with 'nv50cal_space: -110'.${RESET}"
		echo -e "  ${GREEN}✔ Option 1 (Recommended):${RESET} 'safe-suspend lock' (locks Xfce & powers off displays via DPMS; 0W screen, 0% crash risk)."
		echo -e "  ${GREEN}✔ Option 2:${RESET} 'safe-suspend s2idle' (suspends CPU to idle states without cutting GPU PLL power)."
		echo -e "  ${GREEN}✔ Option 3:${RESET} Configure 24GB swapfile for full zero-power disk hibernation."
	else
		echo -e "  ${GREEN}✔ Standard systemctl suspend is supported.${RESET}"
	fi
}

# ------------------------------------------------------------------------------
# Action: Screen Lock & DPMS Standby (100% Reliable, Zero Video Loss)
# ------------------------------------------------------------------------------
do_lock_screen() {
	echo -e "${BOLD}Entering Safe Screen Standby...${RESET}"
	log_step "Flushing filesystem buffers to disk..."
	sync

	lock_desktop_session
	sleep 0.4

	if command -v xset >/dev/null 2>&1 && [[ -n "${DISPLAY:-}" ]]; then
		log_step "Powering off all display outputs (DPMS force off)..."
		xset dpms force off 2>/dev/null || true
	fi

	log_success "Desktop session locked and monitors placed in low-power standby."
	log_info "Touch any key or move mouse to wake displays instantly."
}

# ------------------------------------------------------------------------------
# Action: Suspend-To-Idle (s2idle)
# ------------------------------------------------------------------------------
do_s2idle() {
	echo -e "${BOLD}Executing Safe Suspend-to-Idle (s2idle)...${RESET}"
	if [[ ! -f /sys/power/mem_sleep ]] || ! grep -q "s2idle" /sys/power/mem_sleep; then
		log_warn "s2idle is not exposed by current kernel. Falling back to safe screen standby..."
		do_lock_screen
		return 0
	fi

	log_step "Flushing filesystem caches..."
	sync
	lock_desktop_session

	log_step "Configuring mem_sleep to s2idle..."
	if [[ -w /sys/power/mem_sleep ]]; then
		echo "s2idle" > /sys/power/mem_sleep
	else
		echo "s2idle" | sudo tee /sys/power/mem_sleep >/dev/null
	fi

	log_step "Suspending system..."
	systemctl suspend || sudo systemctl suspend

	log_success "System resumed from s2idle."
}

# ------------------------------------------------------------------------------
# Action: Deep S3 Suspend with Virtual Terminal Switch (VT unbind)
# ------------------------------------------------------------------------------
do_vt_suspend() {
	echo -e "${BOLD}Executing Deep S3 Suspend with VT Switch...${RESET}"
	local orig_vt
	orig_vt="$(get_active_vt)"
	local target_vt="1"
	if [[ "$orig_vt" == "1" ]]; then
		target_vt="2"
	fi

	log_step "Flushing disk caches..."
	sync
	lock_desktop_session

	log_step "Switching to console tty${target_vt} to release X11 DRM master..."
	sudo chvt "$target_vt"
	sleep 0.5

	log_step "Calling systemctl suspend..."
	systemctl suspend || sudo systemctl suspend || true

	log_step "Resumed from sleep. Waiting for DisplayPort link training..."
	sleep 1.0

	log_step "Restoring X11 session on tty${orig_vt}..."
	sudo chvt "$orig_vt"
	sleep 0.5

	if command -v xrandr >/dev/null 2>&1 && [[ -n "${DISPLAY:-}" ]]; then
		xrandr --auto 2>/dev/null || true
	fi

	log_success "Resumed successfully."
}

# ------------------------------------------------------------------------------
# Action: Hibernate (Suspend to Disk)
# ------------------------------------------------------------------------------
do_hibernate() {
	echo -e "${BOLD}Checking Hibernation Prerequisites...${RESET}"
	local ram_total_mb swap_total_mb
	ram_total_mb=$(free -m | awk '/^Mem:/{print $2}')
	swap_total_mb=$(free -m | awk '/^Swap:/{print $2}')

	if [[ "$swap_total_mb" -lt "$ram_total_mb" ]]; then
		log_error "Cannot hibernate: swap space (${swap_total_mb} MB) is smaller than RAM (${ram_total_mb} MB)."
		echo ""
		echo -e "${YELLOW}To enable hibernation on this 22GB system:${RESET}"
		echo "  1. sudo swapoff /swapfile"
		echo "  2. sudo dd if=/dev/zero of=/swapfile bs=1G count=24 status=progress"
		echo "  3. sudo chmod 600 /swapfile && sudo mkswap /swapfile && sudo swapon /swapfile"
		echo "  4. Update /etc/initramfs-tools/conf.d/resume with swap UUID and update-initramfs -u"
		return 1
	fi

	log_step "Flushing disks and initiating hibernation..."
	sync
	systemctl hibernate || sudo systemctl hibernate
}

# ------------------------------------------------------------------------------
# Main Dispatch
# ------------------------------------------------------------------------------
main() {
	local mode="${1:-}"

	case "$mode" in
		status|check|info|-h|--help)
			do_status
			;;
		lock|screen|blank)
			do_lock_screen
			;;
		s2idle|idle)
			do_s2idle
			;;
		deep|vt|s3)
			do_vt_suspend
			;;
		hibernate|disk)
			do_hibernate
			;;
		"")
			# Default intelligent auto selection based on hardware
			local driver
			driver="$(detect_gpu_driver)"
			if [[ "$driver" == "nouveau" ]] && has_nouveau_pstate_quirk; then
				log_warn "Dell M6800 / Nouveau Kepler detected with 'nouveau.pstate=1'."
				log_info "Standard S3 sleep locks up video. Activating safe screen standby & lock..."
				do_lock_screen
			else
				do_lock_screen
			fi
			;;
		*)
			log_error "Unknown mode: '$mode'"
			echo "Usage: safe-suspend [lock|s2idle|deep|hibernate|status]"
			exit 1
			;;
	esac
}

main "$@"
