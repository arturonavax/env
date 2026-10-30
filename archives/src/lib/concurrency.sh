#!/usr/bin/env bash
# ==============================================================================
# src/lib/concurrency.sh
# Production Parallel Task Pool & Background Worker Engine.
# Runs independent tasks concurrently with real-time visual progress & error isolation.
# Part of arturonavax/env development environment.
# ==============================================================================

# Prevent double sourcing
[[ -n "${_CONCURRENCY_SH_LOADED:-}" ]] && return 0
_CONCURRENCY_SH_LOADED=1

_LIB_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
if [[ -f "${_LIB_DIR}/colors.sh" ]]; then
	# shellcheck source=/dev/null
	source "${_LIB_DIR}/colors.sh"
fi
if [[ -f "${_LIB_DIR}/logging.sh" ]]; then
	# shellcheck source=/dev/null
	source "${_LIB_DIR}/logging.sh"
fi

# Detect available CPU cores portably
get_cpu_cores() {
	if command -v nproc >/dev/null 2>&1; then
		nproc
	elif command -v getconf >/dev/null 2>&1; then
		getconf _NPROCESSORS_ONLN 2>/dev/null || echo 4
	elif command -v sysctl >/dev/null 2>&1; then
		sysctl -n hw.ncpu 2>/dev/null || echo 4
	else
		echo 4
	fi
}

# Concurrency limit (default: cores capped between 2 and 6 to avoid network choke)
_DEFAULT_CONCURRENCY=$(get_cpu_cores)
if ((_DEFAULT_CONCURRENCY > 6)); then
	_DEFAULT_CONCURRENCY=6
elif ((_DEFAULT_CONCURRENCY < 2)); then
	_DEFAULT_CONCURRENCY=2
fi
export MAX_PARALLEL_JOBS="${MAX_PARALLEL_JOBS:-$_DEFAULT_CONCURRENCY}"

# Run a list of tasks in parallel
# Usage:
#   run_parallel_task_pool "Task Pool Name" "job1_label:::job1_cmd" "job2_label:::job2_cmd" ...
run_parallel_task_pool() {
	local pool_name="$1"
	shift
	local total_tasks=$#

	if ((total_tasks == 0)); then
		return 0
	fi

	log_step "${BOLD_WHITE}${pool_name}${RESET} (Running ${total_tasks} tasks concurrently, max parallel: ${MAX_PARALLEL_JOBS})..."

	local tmp_dir
	tmp_dir="$(mktemp -d "${TMPDIR:-/tmp}/parallel_pool.XXXXXX")"

	local -a task_labels=()
	local -a task_cmds=()
	local -a task_pids=()
	local -a task_logs=()
	local -a task_statuses=() # PENDING, RUNNING, DONE, FAILED

	local idx=0
	for spec in "$@"; do
		local label="${spec%%:::*}"
		local cmd="${spec#*:::}"
		task_labels[idx]="$label"
		task_cmds[idx]="$cmd"
		task_pids[idx]=0
		task_logs[idx]="${tmp_dir}/task_${idx}.log"
		task_statuses[idx]="PENDING"
		idx=$((idx + 1))
	done

	local active_count=0
	local finished_count=0
	local failed_count=0
	local next_to_launch=0

	# Spinner characters
	local -a spin_chars=("⠋" "⠙" "⠹" "⠸" "⠼" "⠴" "⠦" "⠧" "⠇" "⠏")
	local spin_idx=0

	# Loop until all tasks are finished
	while ((finished_count < total_tasks)); do
		# 1. Launch new tasks up to MAX_PARALLEL_JOBS
		while ((active_count < MAX_PARALLEL_JOBS && next_to_launch < total_tasks)); do
			local current_task=$next_to_launch
			local cmd="${task_cmds[current_task]}"
			local log_file="${task_logs[current_task]}"

			# Launch in background
			(
				eval "$cmd" >"$log_file" 2>&1
			) &
			task_pids[current_task]=$!
			task_statuses[current_task]="RUNNING"

			active_count=$((active_count + 1))
			next_to_launch=$((next_to_launch + 1))
		done

		# 2. Check active tasks
		for ((i = 0; i < next_to_launch; i++)); do
			if [[ "${task_statuses[i]}" == "RUNNING" ]]; then
				local pid="${task_pids[i]}"
				if ! kill -0 "$pid" 2>/dev/null; then
					# Task completed
					wait "$pid"
					local exit_code=$?
					if ((exit_code == 0)); then
						task_statuses[i]="DONE"
						printf '  %s%s%s %-45s %s[DONE]%s\n' "$GREEN" "$SYM_CHECK" "$RESET" "${task_labels[i]}" "$BOLD_GREEN" "$RESET"
					else
						task_statuses[i]="FAILED"
						failed_count=$((failed_count + 1))
						printf '  %s%s%s %-45s %s[FAILED: %s]%s\n' "$RED" "$SYM_CROSS" "$RESET" "${task_labels[i]}" "$BOLD_RED" "$exit_code" "$RESET"
					fi
					active_count=$((active_count - 1))
					finished_count=$((finished_count + 1))
				fi
			fi
		done

		# 3. Small sleep and spinner update if in interactive terminal
		if [[ -t 1 ]] && ((finished_count < total_tasks)); then
			spin_idx=$(((spin_idx + 1) % ${#spin_chars[@]}))
			printf '\r  %s%s%s [%s/%s completed, %s running]... ' \
				"$CYAN" "${spin_chars[spin_idx]}" "$RESET" "$finished_count" "$total_tasks" "$active_count"
		fi

		sleep 0.1
	done

	if [[ -t 1 ]]; then
		printf '\r%s\r' "                                                                      "
	fi

	# 4. Report Failures with Exact Captured Output
	if ((failed_count > 0)); then
		echo
		log_error "The following ${failed_count} parallel task(s) failed:"
		for ((i = 0; i < total_tasks; i++)); do
			if [[ "${task_statuses[i]}" == "FAILED" ]]; then
				echo -e "\n  ${BOLD_RED}------------------------------------------------------------${RESET}"
				echo -e "  ${BOLD_WHITE}Failed Task:${RESET} ${task_labels[i]}"
				echo -e "  ${BOLD_WHITE}Command:${RESET}     ${task_cmds[i]}"
				echo -e "  ${BOLD_WHITE}Captured Log Output:${RESET}"
				if [[ -f "${task_logs[i]}" ]]; then
					sed 's/^/    /' "${task_logs[i]}" | tail -n 20
				else
					echo "    (no log captured)"
				fi
				echo -e "  ${BOLD_RED}------------------------------------------------------------${RESET}"
			fi
		done
		rm -rf "$tmp_dir"
		return "$failed_count"
	fi

	rm -rf "$tmp_dir"
	log_success "All ${total_tasks} parallel tasks in '${pool_name}' finished successfully."
	return 0
}
