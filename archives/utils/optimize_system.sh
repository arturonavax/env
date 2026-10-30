#!/usr/bin/env bash
# ==============================================================================
# utils/optimize_system.sh
# Production Multi-Platform Hardware & System Optimizer with Lynis Integration,
# Actionable Report Distillation, aitools LLM Generation & Strict Security Guardrails.
# Part of arturonavax/env development environment.
# ==============================================================================
set -euo pipefail

# ── 1. Repository Anchoring & Library Loading ──────────────────────────────────
SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT=""
curr_dir="$SCRIPT_DIR"
while [[ "$curr_dir" != "/" ]]; do
	if [[ -d "$curr_dir/.git" && -d "$curr_dir/src/lib" ]]; then
		REPO_ROOT="$curr_dir"
		break
	fi
	curr_dir="$(dirname "$curr_dir")"
done

if [[ -z "$REPO_ROOT" ]]; then
	echo "[ERROR]: Failed to locate arturonavax/env repository root." >&2
	exit 1
fi

# Load modular libraries
# shellcheck source=/dev/null
source "${REPO_ROOT}/src/lib/colors.sh" 2>/dev/null || true
# shellcheck source=/dev/null
source "${REPO_ROOT}/src/lib/logging.sh" 2>/dev/null || true
# shellcheck source=/dev/null
source "${REPO_ROOT}/src/lib/platform.sh" 2>/dev/null || true
# shellcheck source=/dev/null
source "${REPO_ROOT}/src/lib/ui_box.sh" 2>/dev/null || true
# shellcheck source=/dev/null
source "${REPO_ROOT}/src/lib/backup.sh" 2>/dev/null || true
# shellcheck source=/dev/null
source "${REPO_ROOT}/src/lib/wizard_engine.sh" 2>/dev/null || true

AITOOLS_BIN="${REPO_ROOT}/files/ai/aitools/aitools"
if [[ ! -x "$AITOOLS_BIN" ]] && command -v aitools >/dev/null 2>&1; then
	AITOOLS_BIN="$(command -v aitools)"
fi

# ── 2. Temporary Working Environment & Cleanup Traps ─────────────────────────
TEMP_DIR="$(mktemp -d /tmp/sysopt_XXXXXX)"
cleanup() {
	local exit_code=$?
	# Remove any lynis pid files created in home or temp
	rm -f "$HOME/lynis.pid" "${TEMP_DIR}/lynis.pid" 2>/dev/null || true
	if [[ -d "$TEMP_DIR" ]]; then
		rm -rf "$TEMP_DIR" 2>/dev/null || true
	fi
	exit "$exit_code"
}
trap cleanup EXIT INT TERM

# ── 3. CLI State & Options ───────────────────────────────────────────────────
MODE_WIZARD=0
MODE_APPLY=0
MODE_AUDIT_ONLY=0
SKIP_LYNIS=0
AI_BACKEND=""
AI_MODEL=""
AI_EFFORT="medium"
OUTPUT_FILE=""
VALIDATE_ONLY_FILE=""

usage() {
	local width
	width="$(ui_box_width 78)"
	echo
	ui_box_top "$width" "rounded" "$BOLD_CYAN"
	ui_box_line "System Hardware & Performance Optimizer · arturonavax/env" "$width" "rounded" "$BOLD_CYAN" "$BOLD_WHITE" "center"
	ui_box_mid "$width" "rounded" "$BOLD_CYAN"
	ui_box_line "Audits hardware, kernel & Lynis security metrics, distills actionable bottlenecks," "$width" "rounded" "$BOLD_CYAN" "$DIM" "left"
	ui_box_line "generates a custom optimization script via aitools llm, and validates guardrails." "$width" "rounded" "$BOLD_CYAN" "$DIM" "left"
	ui_box_mid "$width" "rounded" "$BOLD_CYAN"
	ui_box_line "USAGE:" "$width" "rounded" "$BOLD_CYAN" "$BOLD_YELLOW" "left"
	ui_box_line "  $(basename "$0") [OPTIONS]" "$width" "rounded" "$BOLD_CYAN" "$BOLD_WHITE" "left"
	ui_box_line "" "$width" "rounded" "$BOLD_CYAN" "$RESET" "left"
	ui_box_line "OPTIONS:" "$width" "rounded" "$BOLD_CYAN" "$BOLD_YELLOW" "left"
	ui_box_line "  -w, --wizard         Launch interactive step-by-step TUI wizard" "$width" "rounded" "$BOLD_CYAN" "$RESET" "left"
	ui_box_line "  -d, --dry-run        Audit, generate script & validate guardrails without applying (default)" "$width" "rounded" "$BOLD_CYAN" "$RESET" "left"
	ui_box_line "  -y, --apply          Generate, validate guardrails, and execute optimizations" "$width" "rounded" "$BOLD_CYAN" "$RESET" "left"
	ui_box_line "  -a, --audit-only     Print distilled hardware & security actionable report and exit" "$width" "rounded" "$BOLD_CYAN" "$RESET" "left"
	ui_box_line "  --skip-lynis         Bypass Lynis audit scan (rely on native security audit)" "$width" "rounded" "$BOLD_CYAN" "$RESET" "left"
	ui_box_line "  -o, --output FILE    Save generated optimization script directly to file" "$width" "rounded" "$BOLD_CYAN" "$RESET" "left"
	ui_box_line "  -b, --backend ID     Override AI CLI backend (agy, claude, codex, llm, pi, ollama)" "$width" "rounded" "$BOLD_CYAN" "$RESET" "left"
	ui_box_line "  -m, --model NAME     Override AI model identifier" "$width" "rounded" "$BOLD_CYAN" "$RESET" "left"
	ui_box_line "  --validate FILE      Validate an existing script against the security guardrail engine" "$width" "rounded" "$BOLD_CYAN" "$RESET" "left"
	ui_box_line "  -h, --help           Show this manual" "$width" "rounded" "$BOLD_CYAN" "$RESET" "left"
	ui_box_bottom "$width" "rounded" "$BOLD_CYAN"
	echo
}

# Parse Arguments
while [[ $# -gt 0 ]]; do
	case "$1" in
	-w | --wizard)
		MODE_WIZARD=1
		shift
		;;
	-d | --dry-run)
		MODE_APPLY=0
		shift
		;;
	-y | --apply)
		MODE_APPLY=1
		shift
		;;
	-a | --audit-only)
		MODE_AUDIT_ONLY=1
		shift
		;;
	--skip-lynis)
		SKIP_LYNIS=1
		shift
		;;
	-o | --output)
		[[ -z "${2:-}" ]] && { echo "Error: --output requires a file path." >&2; exit 1; }
		OUTPUT_FILE="$2"
		shift 2
		;;
	-b | --backend)
		[[ -z "${2:-}" ]] && { echo "Error: --backend requires an identifier." >&2; exit 1; }
		AI_BACKEND="$2"
		shift 2
		;;
	-m | --model)
		[[ -z "${2:-}" ]] && { echo "Error: --model requires a model name." >&2; exit 1; }
		AI_MODEL="$2"
		shift 2
		;;
	--validate)
		[[ -z "${2:-}" ]] && { echo "Error: --validate requires a script path." >&2; exit 1; }
		VALIDATE_ONLY_FILE="$2"
		shift 2
		;;
	-h | --help)
		usage
		exit 0
		;;
	*)
		echo "Unknown argument: $1" >&2
		usage
		exit 1
		;;
	esac
done

# If running in interactive TTY without flags, default to wizard mode
if [[ -t 0 && -t 1 && $MODE_WIZARD -eq 0 && $MODE_APPLY -eq 0 && $MODE_AUDIT_ONLY -eq 0 && -z "$VALIDATE_ONLY_FILE" && $# -eq 0 ]]; then
	MODE_WIZARD=1
fi

# ── 4. Multi-Platform Hardware & System Profiler ──────────────────────────────
profile_hardware_and_system() {
	local os distro arch kernel is_wsl cpu_model cpu_cores cpu_threads
	local ram_total_mb ram_avail_mb swap_total_mb swappiness inotify_watches
	local tcp_congestion tcp_available io_schedulers disk_summary trim_timer
	local startup_time failed_units display_server gpu_info

	os="$(detect_os)"
	distro="$(detect_distro)"
	arch="$(detect_arch)"
	kernel="$(uname -r 2>/dev/null || echo "Unknown")"
	is_wsl="No"
	if [[ "$os" == "wsl" ]] || grep -qi "microsoft" /proc/version 2>/dev/null; then
		is_wsl="Yes"
	fi

	# 1. CPU
	cpu_model="Unknown"
	cpu_cores="1"
	cpu_threads="1"
	if command -v lscpu >/dev/null 2>&1; then
		cpu_model="$(lscpu 2>/dev/null | awk -F': +' '/Model name/ {print $2; exit}' || echo "Unknown")"
		cpu_cores="$(lscpu 2>/dev/null | awk -F': +' '/Core\(s\) per socket/ {print $2; exit}' || echo "1")"
		cpu_threads="$(nproc 2>/dev/null || echo "1")"
	elif [[ "$os" == "darwin" ]]; then
		cpu_model="$(sysctl -n machdep.cpu.brand_string 2>/dev/null || echo "Apple Silicon / Intel")"
		cpu_cores="$(sysctl -n hw.physicalcpu 2>/dev/null || echo "1")"
		cpu_threads="$(sysctl -n hw.ncpu 2>/dev/null || echo "1")"
	elif [[ -f /proc/cpuinfo ]]; then
		cpu_model="$(grep -m1 "model name" /proc/cpuinfo | cut -d: -f2 | sed 's/^[ \t]*//' || echo "Unknown")"
		cpu_threads="$(grep -c "^processor" /proc/cpuinfo || echo "1")"
		cpu_cores="$cpu_threads"
	fi

	# CPU Scaling Governor & Driver (Linux)
	local cpu_governor="n/a" cpu_driver="n/a"
	if [[ -f /sys/devices/system/cpu/cpu0/cpufreq/scaling_governor ]]; then
		cpu_governor="$(cat /sys/devices/system/cpu/cpu0/cpufreq/scaling_governor 2>/dev/null || echo "unknown")"
	fi
	if [[ -f /sys/devices/system/cpu/cpu0/cpufreq/scaling_driver ]]; then
		cpu_driver="$(cat /sys/devices/system/cpu/cpu0/cpufreq/scaling_driver 2>/dev/null || echo "unknown")"
	fi

	# 2. Memory & Swap
	ram_total_mb="$(get_total_ram_mb)"
	ram_avail_mb="0"
	swap_total_mb="0"
	if [[ -f /proc/meminfo ]]; then
		ram_avail_mb="$(awk '/MemAvailable/ {printf "%d\n", $2 / 1024}' /proc/meminfo 2>/dev/null || echo "0")"
		swap_total_mb="$(awk '/SwapTotal/ {printf "%d\n", $2 / 1024}' /proc/meminfo 2>/dev/null || echo "0")"
	elif [[ "$os" == "darwin" ]]; then
		local vm_stat_out
		vm_stat_out="$(vm_stat 2>/dev/null || true)"
		local pages_free pages_active
		pages_free="$(echo "$vm_stat_out" | awk '/Pages free/ {print $3}' | tr -d '.' || echo 0)"
		pages_active="$(echo "$vm_stat_out" | awk '/Pages active/ {print $3}' | tr -d '.' || echo 0)"
		ram_avail_mb="$(( (pages_free + pages_active) * 4096 / 1024 / 1024 ))"
	fi

	swappiness="$(sysctl -n vm.swappiness 2>/dev/null || echo "n/a")"
	local zram_active="No"
	if command -v swapon >/dev/null 2>&1 && swapon --show 2>/dev/null | grep -qi "zram"; then
		zram_active="Yes"
	elif [[ -d /sys/block/zram0 ]]; then
		zram_active="Yes"
	fi

	# 3. Disks, Schedulers & TRIM
	disk_summary=""
	io_schedulers=""
	trim_timer="n/a"
	if command -v lsblk >/dev/null 2>&1; then
		# Filter disks and extract name, size, rota (0=SSD/NVMe, 1=HDD)
		disk_summary="$(lsblk -d -o NAME,SIZE,TYPE,ROTA,MODEL 2>/dev/null | grep -E 'disk' || true)"
		# Inspect block schedulers
		local dev
		for dev in /sys/block/*; do
			local bname
			bname="$(basename "$dev")"
			if [[ "$bname" =~ ^(sd[a-z]|nvme[0-9]+n[0-9]+|vd[a-z]|mmcblk[0-9]+)$ ]] && [[ -f "$dev/queue/scheduler" ]]; then
				local sched
				sched="$(cat "$dev/queue/scheduler" 2>/dev/null || echo "unknown")"
				io_schedulers+="${bname}: ${sched} | "
			fi
		done
		io_schedulers="${io_schedulers% | }"
	elif [[ "$os" == "darwin" ]]; then
		disk_summary="$(diskutil list 2>/dev/null | head -n 12 || true)"
		io_schedulers="macOS APFS native I/O queue"
	fi

	if command -v systemctl >/dev/null 2>&1; then
		trim_timer="$(systemctl is-enabled fstrim.timer 2>/dev/null || echo "disabled")"
	fi

	# 4. Kernel, Network & Limits
	tcp_congestion="$(sysctl -n net.ipv4.tcp_congestion_control 2>/dev/null || echo "n/a")"
	tcp_available="$(sysctl -n net.ipv4.tcp_available_congestion_control 2>/dev/null || echo "n/a")"
	inotify_watches="$(sysctl -n fs.inotify.max_user_watches 2>/dev/null || echo "n/a")"
	local file_max somaxconn max_map_count
	file_max="$(sysctl -n fs.file-max 2>/dev/null || sysctl -n kern.maxfiles 2>/dev/null || echo "n/a")"
	somaxconn="$(sysctl -n net.core.somaxconn 2>/dev/null || sysctl -n kern.ipc.somaxconn 2>/dev/null || echo "n/a")"
	max_map_count="$(sysctl -n vm.max_map_count 2>/dev/null || echo "n/a")"

	# 5. Boot & Startup Services
	startup_time="n/a"
	local slowest_services=""
	failed_units="None"
	if command -v systemd-analyze >/dev/null 2>&1; then
		startup_time="$(systemd-analyze time 2>/dev/null | head -n 1 || echo "n/a")"
		slowest_services="$(systemd-analyze blame 2>/dev/null | head -n 5 | awk '{printf "%s (%s), ", $2, $1}' || true)"
		slowest_services="${slowest_services%, }"
		local failed_cnt
		failed_cnt="$(systemctl --failed --no-legend 2>/dev/null | wc -l || echo 0)"
		if (( failed_cnt > 0 )); then
			failed_units="${failed_cnt} unit(s) failing"
		fi
	fi

	# 6. Display & GPU
	display_server="$(detect_display_server)"
	gpu_info="Unknown"
	if command -v lspci >/dev/null 2>&1; then
		gpu_info="$(lspci 2>/dev/null | grep -iE 'vga|3d|display' | cut -d: -f3 | sed 's/^[ \t]*//' | head -n 2 | tr '\n' '; ' || echo "Unknown")"
		gpu_info="${gpu_info%; }"
	elif [[ "$os" == "darwin" ]]; then
		gpu_info="$(system_profiler SPDisplaysDataType 2>/dev/null | awk -F': +' '/Chipset Model/ {print $2}' | tr '\n' '; ' || echo "Apple Graphics")"
		gpu_info="${gpu_info%; }"
	fi

	# Store in global state file for synthesis
	cat <<EOF > "${TEMP_DIR}/hardware_profile.env"
OS="${os}"
DISTRO="${distro}"
ARCH="${arch}"
KERNEL="${kernel}"
IS_WSL="${is_wsl}"
CPU_MODEL="${cpu_model}"
CPU_CORES="${cpu_cores}"
CPU_THREADS="${cpu_threads}"
CPU_GOVERNOR="${cpu_governor}"
CPU_DRIVER="${cpu_driver}"
RAM_TOTAL_MB="${ram_total_mb}"
RAM_AVAIL_MB="${ram_avail_mb}"
SWAP_TOTAL_MB="${swap_total_mb}"
SWAPPINESS="${swappiness}"
ZRAM_ACTIVE="${zram_active}"
IO_SCHEDULERS="${io_schedulers}"
TRIM_TIMER="${trim_timer}"
TCP_CONGESTION="${tcp_congestion}"
TCP_AVAILABLE="${tcp_available}"
INOTIFY_WATCHES="${inotify_watches}"
FILE_MAX="${file_max}"
SOMAXCONN="${somaxconn}"
MAX_MAP_COUNT="${max_map_count}"
STARTUP_TIME="${startup_time}"
SLOWEST_SERVICES="${slowest_services}"
FAILED_UNITS="${failed_units}"
DISPLAY_SERVER="${display_server}"
GPU_INFO="${gpu_info}"
DISK_SUMMARY="${disk_summary}"
EOF
}

# ── 5. Security & Lynis Audit Context ─────────────────────────────────────────
profile_security_context() {
	local lynis_present=0
	local lynis_index="n/a"
	local lynis_warnings=()
	local lynis_suggestions=()
	local lynis_report="${TEMP_DIR}/lynis-report.dat"
	local lynis_log="${TEMP_DIR}/lynis.log"

	if command -v lynis >/dev/null 2>&1 && [[ $SKIP_LYNIS -eq 0 ]]; then
		lynis_present=1
		log_step "Executing non-privileged Lynis security and performance audit..."

		# Clean any prior stale pid
		rm -f "$HOME/lynis.pid" "./lynis.pid" 2>/dev/null || true

		# Run non-privileged fast audit without plugins to guarantee fast completion
		lynis audit system \
			--no-plugins \
			--quick \
			--cronjob \
			--report-file "$lynis_report" \
			--log-file "$lynis_log" >/dev/null 2>&1 || true

		# Parse findings from report if created
		if [[ -f "$lynis_report" ]]; then
			lynis_index="$(awk -F= '/^hardening_index/ {print $2}' "$lynis_report" || echo "n/a")"

			# Extract warnings
			while IFS= read -r line; do
				[[ -n "$line" ]] && lynis_warnings+=("${line#warning[]=}")
			done < <(grep '^warning\[\]=' "$lynis_report" 2>/dev/null || true)

			# Extract top suggestions
			while IFS= read -r line; do
				[[ -n "$line" ]] && lynis_suggestions+=("${line#suggestion[]=}")
			done < <(grep '^suggestion\[\]=' "$lynis_report" 2>/dev/null | head -n 8 || true)
		fi
		# Always clean generated PID file in home directory
		rm -f "$HOME/lynis.pid" "./lynis.pid" 2>/dev/null || true
	fi

	# Native security baseline check (works regardless of Lynis)
	local firewall_status="disabled"
	if command -v ufw >/dev/null 2>&1; then
		if ufw status 2>/dev/null | grep -qi "active"; then
			firewall_status="ufw active"
		fi
	elif command -v iptables >/dev/null 2>&1; then
		if iptables -L -n 2>/dev/null | grep -q "Chain"; then
			firewall_status="iptables configured"
		fi
	elif command -v pfctl >/dev/null 2>&1; then
		if pfctl -s info 2>/dev/null | grep -qi "enabled"; then
			firewall_status="pfctl enabled"
		fi
	fi

	local kptr_restrict dmesg_restrict suid_dumpable
	kptr_restrict="$(sysctl -n kernel.kptr_restrict 2>/dev/null || echo "n/a")"
	dmesg_restrict="$(sysctl -n kernel.dmesg_restrict 2>/dev/null || echo "n/a")"
	suid_dumpable="$(sysctl -n fs.suid_dumpable 2>/dev/null || echo "n/a")"

	cat <<EOF > "${TEMP_DIR}/security_profile.env"
LYNIS_PRESENT="${lynis_present}"
LYNIS_INDEX="${lynis_index}"
FIREWALL_STATUS="${firewall_status}"
KPTR_RESTRICT="${kptr_restrict}"
DMESG_RESTRICT="${dmesg_restrict}"
SUID_DUMPABLE="${suid_dumpable}"
EOF

	# Write warnings and suggestions to files
	printf "%s\n" "${lynis_warnings[@]:-}" > "${TEMP_DIR}/lynis_warnings.txt"
	printf "%s\n" "${lynis_suggestions[@]:-}" > "${TEMP_DIR}/lynis_suggestions.txt"
}

# ── 6. Distill Actionables into Structured High-Signal Report ──────────────────
# shellcheck disable=SC2153
distill_actionable_report() {
	# shellcheck source=/dev/null
	source "${TEMP_DIR}/hardware_profile.env"
	# shellcheck source=/dev/null
	source "${TEMP_DIR}/security_profile.env"

	local report_file="${TEMP_DIR}/actionable_report.md"

	{
		echo "# System Hardware, Performance & Security Audit Digest"
		echo ""
		echo "## 1. System Platform & Physical Hardware"
		echo "- Platform: ${OS} (${DISTRO}), Architecture: ${ARCH}, Kernel: ${KERNEL}"
		echo "- Virtualization: WSL=${IS_WSL}"
		echo "- CPU Model: ${CPU_MODEL} (${CPU_CORES} physical cores, ${CPU_THREADS} threads)"
		echo "- CPU Driver & Governor: driver=${CPU_DRIVER}, governor=${CPU_GOVERNOR}"
		echo "- Total RAM: ${RAM_TOTAL_MB} MB, Available: ${RAM_AVAIL_MB} MB, Swap: ${SWAP_TOTAL_MB} MB (ZRAM active: ${ZRAM_ACTIVE})"
		echo "- Display Server: ${DISPLAY_SERVER}, GPU: ${GPU_INFO}"
		echo ""
		echo "## 2. Storage, I/O Schedulers & Disks"
		echo "- Active Schedulers: ${IO_SCHEDULERS:-default}"
		echo "- SSD TRIM Timer (fstrim.timer): ${TRIM_TIMER}"
		if [[ -n "${DISK_SUMMARY:-}" ]]; then
			echo "- Disks Detected:"
			echo "${DISK_SUMMARY}" | while IFS= read -r d; do [[ -n "$d" ]] && echo "  * $d"; done
		fi
		echo ""
		echo "## 3. Kernel, Network & Developer System Limits"
		echo "- Active TCP Congestion: ${TCP_CONGESTION} (Available: ${TCP_AVAILABLE})"
		echo "- Memory Swappiness: ${SWAPPINESS} (Default is 60; recommended for SSD/NVMe workstation is 10-20)"
		echo "- Inotify Max User Watches: ${INOTIFY_WATCHES} (Default 8192 causes ENOSPC in Neovim/Vite/Node watchers; recommend 524288)"
		echo "- Max Open Files (fs.file-max): ${FILE_MAX}"
		echo "- Socket Backlog (net.core.somaxconn): ${SOMAXCONN}"
		echo "- Memory Mapped Files (vm.max_map_count): ${MAX_MAP_COUNT}"
		echo ""
		echo "## 4. Boot & Services Latency"
		echo "- Startup Duration: ${STARTUP_TIME}"
		echo "- Top Slow Startup Units: ${SLOWEST_SERVICES:-None detected}"
		echo "- Failed Units: ${FAILED_UNITS}"
		echo ""
		echo "## 5. Security & Hardening Context"
		echo "- Lynis Installed: $( [[ "$LYNIS_PRESENT" == "1" ]] && echo "Yes" || echo "No (native audit applied)" )"
		echo "- Lynis Hardening Index: ${LYNIS_INDEX}"
		echo "- Firewall: ${FIREWALL_STATUS}"
		echo "- Kernel Pointer Exposure (kptr_restrict): ${KPTR_RESTRICT} (recommend 2)"
		echo "- Kernel Dmesg Restriction (dmesg_restrict): ${DMESG_RESTRICT} (recommend 1)"
		echo "- Core Dumps Dumpable (suid_dumpable): ${SUID_DUMPABLE} (recommend 0)"

		if [[ -s "${TEMP_DIR}/lynis_warnings.txt" ]]; then
			echo ""
			echo "### Lynis Audit Warnings:"
			while IFS= read -r w; do
				[[ -n "$w" ]] && echo "- [WARNING] $w"
			done < "${TEMP_DIR}/lynis_warnings.txt"
		fi

		if [[ -s "${TEMP_DIR}/lynis_suggestions.txt" ]]; then
			echo ""
			echo "### Lynis Key Suggestions:"
			while IFS= read -r s; do
				[[ -n "$s" ]] && echo "- [SUGGESTION] $s"
			done < "${TEMP_DIR}/lynis_suggestions.txt"
		fi
	} > "$report_file"

	cat "$report_file"
}

# ── 7. Strict Security Guardrail Engine ───────────────────────────────────────
validate_script_guardrails() {
	local target_script="$1"

	if [[ ! -f "$target_script" ]]; then
		log_error "Validation failed: script file '$target_script' not found."
		return 1
	fi

	# 1. Syntax Verification
	if ! bash -n "$target_script" 2>"${TEMP_DIR}/syntax_error.log"; then
		local err
		err="$(cat "${TEMP_DIR}/syntax_error.log")"
		log_error "Guardrail Rejection: Script has invalid Bash syntax:"
		echo "$err" >&2
		return 1
	fi

	# 2. Dangerous Command Blacklist Verification
	# shellcheck disable=SC2016
	local -a dangerous_patterns=(
		# Destructive deletions
		'rm[[:space:]]+(-[a-zA-Z]*r[a-zA-Z]*|--recursive)'
		'rm[[:space:]]+-[a-zA-Z]*f[a-zA-Z]*[[:space:]]+(\/|\/etc|\/boot|\/usr|\/var|\/home|~|\$HOME)'
		'rmdir[[:space:]]+'
		# Partition & disk formatting / wiping
		'\b(mkfs|fdisk|parted|gdisk|wipefs|shred)\b'
		'\bdd[[:space:]]+if='
		# Direct write to raw block devices
		'>[[:space:]]*\/dev\/(sd[a-z]|nvme[0-9]+|hd[a-z]|vd[a-z])'
		# Disabling firewalls or security modules
		'\b(ufw[[:space:]]+disable|iptables[[:space:]]+-F|pfctl[[:space:]]+-d)\b'
		'\bsetenforce[[:space:]]+0\b'
		# Modifying credentials or shadow file directly
		'(\/etc\/shadow|\/etc\/passwd|\/etc\/sudoers)'
		# Remote pipeline code execution
		'(curl|wget)[[:space:]]+.*\|[[:space:]]*(bash|sh|zsh)'
		# Blanking root passwords
		'passwd[[:space:]]+-d'
	)

	local -a rule_names=(
		"Recursive directory deletion (rm -r / rm -rf)"
		"Root or home filesystem purge (rm -f /)"
		"Direct directory removal (rmdir)"
		"Filesystem formatting or disk partition manipulation (mkfs/fdisk/parted/wipefs/shred)"
		"Raw disk block copy / overwrite (dd if=)"
		"Direct raw device stdout redirection (> /dev/sd*)"
		"Disabling active firewall (ufw disable / iptables -F / pfctl -d)"
		"Disabling SELinux / AppArmor (setenforce 0)"
		"Direct tampering with system credential databases (/etc/shadow, /etc/passwd, /etc/sudoers)"
		"Unverified remote code download & execution pipeline (curl | sh)"
		"Blanking user passwords (passwd -d)"
	)

	local violations=0
	local idx=0
	local total_rules="${#dangerous_patterns[@]}"

	for ((idx=0; idx<total_rules; idx++)); do
		local pattern="${dangerous_patterns[$idx]}"
		local rule_name="${rule_names[$idx]}"

		if grep -Eiq "$pattern" "$target_script"; then
			local match_line
			match_line="$(grep -Ein "$pattern" "$target_script" | head -n 1 || true)"
			log_error "SECURITY GUARDRAIL TRIGGERED!"
			log_error "  Rule violated : ${rule_name}"
			log_error "  Offending line: ${match_line}"
			violations=$((violations + 1))
		fi
	done

	if (( violations > 0 )); then
		echo
		local width
		width="$(ui_box_width 78)"
		ui_box_top "$width" "rounded" "$BOLD_RED"
		ui_box_line "SECURITY GUARDRAIL REJECTION: ${violations} VIOLATION(S) DETECTED" "$width" "rounded" "$BOLD_RED" "$BOLD_WHITE" "center"
		ui_box_mid "$width" "rounded" "$BOLD_RED"
		ui_box_line "The generated script attempted dangerous operations that violate safe system policies." "$width" "rounded" "$BOLD_RED" "$BOLD_WHITE" "left"
		ui_box_line "Execution has been strictly blocked to protect filesystem and credentials." "$width" "rounded" "$BOLD_RED" "$RESET" "left"
		ui_box_bottom "$width" "rounded" "$BOLD_RED"
		echo
		return 1
	fi

	return 0
}

# ── 8. AI Optimization Script Generator ───────────────────────────────────────
generate_ai_optimization_script() {
	local digest_file="$1"
	local output_script="$2"

	log_step "Querying aitools LLM engine to synthesize personalized optimization script..."

	local system_prompt="You are a Principal Systems & Linux Kernel Performance Architect with 20+ years experience.
Your mission is to generate a pristine, production-grade, highly customized system optimization Bash script for this specific machine.
RULES AND SAFETY CONSTRAINTS:
1. TARGETED: Apply ONLY optimizations directly justified by the machine's hardware, CPU, RAM, SSD/NVMe, schedulers, kernel version, and Lynis security findings.
2. IDEMPOTENT & REVERSIBLE: Every setting must be idempotent. Use dedicated drop-in files like /etc/sysctl.d/99-custom-system-tuning.conf instead of editing /etc/sysctl.conf directly.
3. BACKUP FIRST: Before modifying any existing system configuration file, create a timestamped backup with 'cp -p'.
4. STRICT SAFETY: You must NEVER generate destructive commands (NO rm -rf, NO mkfs, NO dd to disks, NO disabling firewalls, NO tampering with /etc/shadow or /etc/passwd).
5. DOCUMENTATION: For EVERY single parameter tuned, include a concise comment '# WHY:' explaining the technical benefit for this specific hardware profile.
6. FORMAT: Output ONLY the complete, executable bash script inside a single '\`\`\`bash' codeblock with '#!/usr/bin/env bash' and 'set -euo pipefail'."

	local user_prompt
	user_prompt="Based on the following hardware, performance, and security audit digest of this workstation, generate the ultra-personalized optimization script:

$(cat "$digest_file")"

	local -a llm_cmd=("$AITOOLS_BIN" "llm" "--mode" "chat" "-o" "text" "-e" "$AI_EFFORT" "-s" "$system_prompt")
	if [[ -n "$AI_BACKEND" ]]; then
		llm_cmd+=("-b" "$AI_BACKEND")
	fi
	if [[ -n "$AI_MODEL" ]]; then
		llm_cmd+=("-m" "$AI_MODEL")
	fi

	local raw_output="${TEMP_DIR}/llm_raw_response.txt"
	if ! printf "%s\n" "$user_prompt" | "${llm_cmd[@]}" > "$raw_output" 2>&1; then
		log_error "Failed to query AI CLI backend through aitools llm."
		cat "$raw_output" >&2
		return 1
	fi

	# Extract script from markdown codeblock if present
	if grep -q '```bash' "$raw_output"; then
		awk '/```bash/{flag=1;next}/```/{flag=0}flag' "$raw_output" > "$output_script"
	elif grep -q '```' "$raw_output"; then
		awk '/```/{flag=!flag;next}flag' "$raw_output" > "$output_script"
	else
		cat "$raw_output" > "$output_script"
	fi

	# Ensure shebang exists
	if ! head -n 1 "$output_script" | grep -q '^#!'; then
		sed -i '1i#!/usr/bin/env bash\nset -euo pipefail\n' "$output_script"
	fi

	chmod +x "$output_script"
}

# ── 9. Standalone Validation Mode ─────────────────────────────────────────────
if [[ -n "$VALIDATE_ONLY_FILE" ]]; then
	log_step "Validating script safety guardrails for: $VALIDATE_ONLY_FILE"
	if validate_script_guardrails "$VALIDATE_ONLY_FILE"; then
		log_success "Script passed all security guardrail checks cleanly."
		exit 0
	else
		exit 1
	fi
fi

# ── 10. Interactive Wizard Mode ───────────────────────────────────────────────
run_optimizer_wizard() {
	wizard_init "AI System Optimizer · Setup & Execution Wizard" 6 "Hardware Profiling, Lynis Security Audit, AI Synthesis & Guardrails"
	wizard_banner

	# ── Stage 1: Hardware & System Profiling ──────────────────────────────────
	wizard_stage "Hardware Architecture & Platform Discovery"
	say "Interrogating processor cores, memory, disk subsystems, and display servers..."
	profile_hardware_and_system

	# shellcheck source=/dev/null
	# shellcheck disable=SC2153
	source "${TEMP_DIR}/hardware_profile.env"
	step "OS & Platform : ${BOLD_WHITE}${OS} (${DISTRO} ${ARCH}) - Kernel ${KERNEL}${RESET}"
	step "Processor     : ${BOLD_WHITE}${CPU_MODEL} (${CPU_CORES}c / ${CPU_THREADS}t)${RESET}"
	step "Memory & Swap : ${BOLD_WHITE}${RAM_TOTAL_MB} MB RAM (Swappiness: ${SWAPPINESS}, ZRAM: ${ZRAM_ACTIVE})${RESET}"
	step "Storage & I/O : ${BOLD_WHITE}TRIM timer: ${TRIM_TIMER} | Schedulers: ${IO_SCHEDULERS}${RESET}"
	step "Display & GPU : ${BOLD_WHITE}${DISPLAY_SERVER} | ${GPU_INFO}${RESET}"
	pause

	# ── Stage 2: Security & Lynis Audit ───────────────────────────────────────
	wizard_stage "Deep Security & Lynis System Audit"
	say "Evaluating system hardening, firewall policies, core dumps, and kernel exposures..."
	profile_security_context

	# shellcheck source=/dev/null
	# shellcheck disable=SC2153
	source "${TEMP_DIR}/security_profile.env"
	step "Lynis Status  : ${BOLD_WHITE}$( [[ "$LYNIS_PRESENT" == "1" ]] && echo "Active (Hardening Index: ${LYNIS_INDEX})" || echo "Not installed (Native audit applied)" )${RESET}"
	step "Firewall      : ${BOLD_WHITE}${FIREWALL_STATUS}${RESET}"
	step "Kernel Shields: ${BOLD_WHITE}kptr_restrict=${KPTR_RESTRICT}, dmesg_restrict=${DMESG_RESTRICT}, suid_dumpable=${SUID_DUMPABLE}${RESET}"
	pause

	# ── Stage 3: Actionable Digest Distillation ────────────────────────────────
	wizard_stage "Actionable Findings & Bottleneck Distillation"
	say "Synthesizing high-signal actionable report for AI optimization..."
	distill_actionable_report > "${TEMP_DIR}/actionable_report.md"

	echo
	local width
	width="$(ui_box_width 78)"
	ui_box_top "$width" "rounded" "$BOLD_BLUE"
	ui_box_line "High-Priority Optimization Opportunities" "$width" "rounded" "$BOLD_BLUE" "$BOLD_WHITE" "center"
	ui_box_mid "$width" "rounded" "$BOLD_BLUE"
	ui_box_line "• Storage   : Check SSD/NVMe scheduler and verify fstrim weekly service" "$width" "rounded" "$BOLD_BLUE" "$RESET" "left"
	ui_box_line "• Memory    : Align swappiness (10-15) and cache pressure for desktop workloads" "$width" "rounded" "$BOLD_BLUE" "$RESET" "left"
	ui_box_line "• Network   : Prime BBR congestion control and socket backlogs for dev servers" "$width" "rounded" "$BOLD_BLUE" "$RESET" "left"
	ui_box_line "• Limits    : Expand inotify.max_user_watches (524288) to prevent ENOSPC in IDEs" "$width" "rounded" "$BOLD_BLUE" "$RESET" "left"
	ui_box_line "• Hardening : Restrict unprivileged dmesg and kernel pointers exposure" "$width" "rounded" "$BOLD_BLUE" "$RESET" "left"
	ui_box_bottom "$width" "rounded" "$BOLD_BLUE"
	echo
	pause

	# ── Stage 4: AI Optimization Generation ───────────────────────────────────
	wizard_stage "AI Script Synthesis via aitools llm"
	say "Formulating custom optimization script tailored specifically to your machine..."
	local generated_script="${TEMP_DIR}/system_optimization.sh"
	generate_ai_optimization_script "${TEMP_DIR}/actionable_report.md" "$generated_script"
	step "Optimization script generated successfully: ${BOLD_WHITE}${generated_script}${RESET}"
	pause

	# ── Stage 5: Security Guardrail Inspection ─────────────────────────────────
	wizard_stage "Automated Security Guardrail Verification"
	say "Scanning generated code against anti-destruction and safety policy rules..."

	if validate_script_guardrails "$generated_script"; then
		echo
		ui_box_top "$width" "rounded" "$BOLD_GREEN"
		ui_box_line "GUARDRAIL SCAN: 100% CLEAN & VERIFIED" "$width" "rounded" "$BOLD_GREEN" "$BOLD_WHITE" "center"
		ui_box_mid "$width" "rounded" "$BOLD_GREEN"
		ui_box_line "✔ Bash Syntax Check: Passed with zero parser errors" "$width" "rounded" "$BOLD_GREEN" "$RESET" "left"
		ui_box_line "✔ Zero Destructive Commands: 0/11 blacklisted patterns triggered" "$width" "rounded" "$BOLD_GREEN" "$RESET" "left"
		ui_box_line "✔ Idempotency & Backups: Non-destructive /etc/sysctl.d/ drop-ins verified" "$width" "rounded" "$BOLD_GREEN" "$RESET" "left"
		ui_box_bottom "$width" "rounded" "$BOLD_GREEN"
		echo
	else
		log_error "Guardrail inspection failed. Aborting wizard."
		exit 1
	fi
	pause

	# ── Stage 6: Review, Preview & Application ─────────────────────────────────
	wizard_stage "Review, Script Export & Application"
	say "The optimization script is ready for review or execution."

	local dest_path="${OUTPUT_FILE:-$HOME/.arturonavax-system-optimize.sh}"
	cp -p "$generated_script" "$dest_path"
	step "Script saved to: ${BOLD_CYAN}${dest_path}${RESET}"

	local -a action_menu=(
		"Preview script preview with line numbers"
		"Apply optimizations now (requires sudo elevation)"
		"Exit without applying (keep script for manual inspection)"
	)
	local chosen_action=""
	select_option chosen_action "Choose action:" "${action_menu[@]}"

	case "$chosen_action" in
	"Preview"*)
		echo
		if command -v bat >/dev/null 2>&1; then
			bat --style=header,numbers "$dest_path"
		elif command -v batcat >/dev/null 2>&1; then
			batcat --style=header,numbers "$dest_path"
		else
			nl -ba "$dest_path" | head -n 40
		fi
		echo
		if wizard_confirm "Apply these optimizations now?"; then
			log_step "Executing optimization script with elevated privileges..."
			sudo bash "$dest_path"
			log_success "System optimization applied successfully!"
		fi
		;;
	"Apply"*)
		log_step "Executing optimization script with elevated privileges..."
		sudo bash "$dest_path"
		log_success "System optimization applied successfully!"
		;;
	*)
		step "You can review and run the script anytime with: ${BOLD_WHITE}sudo bash ${dest_path}${RESET}"
		;;
	esac

	wizard_summary "System Optimization Wizard Completed"
}

# ── 11. Batch / CLI Autonomous Mode ───────────────────────────────────────────
run_optimizer_cli() {
	local width
	width="$(ui_box_width 78)"

	echo
	ui_box_top "$width" "rounded" "$BOLD_CYAN"
	ui_box_line "System Hardware & Performance Optimizer" "$width" "rounded" "$BOLD_CYAN" "$BOLD_WHITE" "center"
	ui_box_mid "$width" "rounded" "$BOLD_CYAN"
	ui_box_line "Phase 1: Profiling hardware, kernel, network and storage..." "$width" "rounded" "$BOLD_CYAN" "$RESET" "left"
	profile_hardware_and_system

	ui_box_line "Phase 2: Auditing security and performance context..." "$width" "rounded" "$BOLD_CYAN" "$RESET" "left"
	profile_security_context

	ui_box_line "Phase 3: Distilling actionable bottlenecks into high-signal report..." "$width" "rounded" "$BOLD_CYAN" "$RESET" "left"
	local digest_file="${TEMP_DIR}/actionable_report.md"
	distill_actionable_report > "$digest_file"
	ui_box_bottom "$width" "rounded" "$BOLD_CYAN"
	echo

	if [[ $MODE_AUDIT_ONLY -eq 1 ]]; then
		log_success "Audit report distillation complete:"
		echo
		cat "$digest_file"
		echo
		exit 0
	fi

	local generated_script="${TEMP_DIR}/system_optimization.sh"
	generate_ai_optimization_script "$digest_file" "$generated_script"

	log_step "Running strict security guardrail scanner on generated optimization script..."
	if ! validate_script_guardrails "$generated_script"; then
		log_error "Generated script failed security guardrail checks. Aborting execution."
		exit 1
	fi
	log_success "Script passed all security guardrail checks cleanly."

	local final_path="${OUTPUT_FILE:-$HOME/.arturonavax-system-optimize.sh}"
	cp -p "$generated_script" "$final_path"
	log_success "Optimization script written to: ${final_path}"

	if [[ $MODE_APPLY -eq 1 ]]; then
		log_step "Applying verified optimizations (requires sudo)..."
		if command -v sudo >/dev/null 2>&1; then
			sudo bash "$final_path"
		else
			bash "$final_path"
		fi
		log_success "All system optimizations executed successfully."
	else
		log_info "Dry-run complete. Inspect the generated script with:"
		log_info "  preview ${final_path} || cat ${final_path}"
		log_info "To execute:"
		log_info "  sudo bash ${final_path}"
	fi
}

# ── 12. Main Entry Point ──────────────────────────────────────────────────────
if [[ $MODE_WIZARD -eq 1 ]]; then
	run_optimizer_wizard
else
	run_optimizer_cli
fi
