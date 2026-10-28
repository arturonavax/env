#!/usr/bin/env bash
# ==============================================================================
# src/lib/transaction.sh
# Production Transactional Execution, Audit Journal & Graceful Shutdown Engine.
# Tracks every filesystem and environment mutation in a reversible journal.
# On interruption (SIGINT/Ctrl-C) or fatal failure, triggers automated or
# operator-guided rollback to leave the host in a pristine, uncorrupted state.
# Part of arturonavax/env development environment.
# ==============================================================================

# Prevent double sourcing
[[ -n "${_TRANSACTION_SH_LOADED:-}" ]] && return 0
_TRANSACTION_SH_LOADED=1

_LIB_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=/dev/null
source "${_LIB_DIR}/colors.sh" 2>/dev/null || true
# shellcheck source=/dev/null
source "${_LIB_DIR}/logging.sh" 2>/dev/null || true

TX_ACTIVE=0
TX_ID=""
TX_JOURNAL_FILE=""
TX_INITIAL_SNAPSHOT=""
TX_LOG_DIR="${HOME}/.arturonavax-env-backups/transactions"

# ------------------------------------------------------------------------------
# Transaction Lifecycle
# ------------------------------------------------------------------------------
tx_begin() {
	local tx_title="${1:-Installation}"
	local initial_snapshot="${2:-}"

	TX_ID="tx_$(date +'%Y%m%d_%H%M%S')_$$"
	mkdir -p "$TX_LOG_DIR"
	TX_JOURNAL_FILE="${TX_LOG_DIR}/${TX_ID}.journal"
	TX_INITIAL_SNAPSHOT="$initial_snapshot"
	TX_ACTIVE=1

	{
		echo "BEGIN: ${tx_title}"
		echo "TX_ID: ${TX_ID}"
		echo "TIMESTAMP: $(date +'%Y-%m-%d %H:%M:%S')"
		echo "SNAPSHOT: ${initial_snapshot}"
		echo "----------------------------------------------------"
	} >"$TX_JOURNAL_FILE"

	log_debug "Transaction ${TX_ID} initialized (${TX_JOURNAL_FILE})."

	# Arm graceful traps for interrupt signals and errors
	trap 'tx_on_signal SIGINT 130' SIGINT
	trap 'tx_on_signal SIGTERM 143' SIGTERM
}

tx_commit() {
	if ((TX_ACTIVE == 0)); then
		return 0
	fi

	echo "COMMIT: $(date +'%Y-%m-%d %H:%M:%S')" >>"$TX_JOURNAL_FILE"
	TX_ACTIVE=0
	log_debug "Transaction ${TX_ID} committed cleanly."

	# Disarm traps
	trap - SIGINT SIGTERM
}

# ------------------------------------------------------------------------------
# Mutation Recorders
# ------------------------------------------------------------------------------
tx_record_file_created() {
	local path="$1"
	if ((TX_ACTIVE == 1)) && [[ -f "$TX_JOURNAL_FILE" ]]; then
		echo "FILE_CREATED:${path}" >>"$TX_JOURNAL_FILE"
	fi
}

tx_record_symlink_created() {
	local link_path="$1"
	if ((TX_ACTIVE == 1)) && [[ -f "$TX_JOURNAL_FILE" ]]; then
		echo "SYMLINK_CREATED:${link_path}" >>"$TX_JOURNAL_FILE"
	fi
}

tx_record_dir_created() {
	local dir_path="$1"
	if ((TX_ACTIVE == 1)) && [[ -f "$TX_JOURNAL_FILE" ]]; then
		echo "DIR_CREATED:${dir_path}" >>"$TX_JOURNAL_FILE"
	fi
}

tx_record_marker_block() {
	local target_file="$1"
	local block_id="$2"
	if ((TX_ACTIVE == 1)) && [[ -f "$TX_JOURNAL_FILE" ]]; then
		echo "MARKER_BLOCK:${target_file}:${block_id}" >>"$TX_JOURNAL_FILE"
	fi
}

tx_record_action() {
	local desc="$1"
	if ((TX_ACTIVE == 1)) && [[ -f "$TX_JOURNAL_FILE" ]]; then
		echo "ACTION:${desc}" >>"$TX_JOURNAL_FILE"
	fi
}

# Read lines in reverse order (LIFO)
_reverse_file() {
	local file="$1"
	if command -v tac >/dev/null 2>&1; then
		tac "$file"
	elif command -v tail >/dev/null 2>&1 && tail -r /dev/null >/dev/null 2>&1; then
		tail -r "$file"
	else
		sed '1!G;h;$!d' "$file"
	fi
}

# ------------------------------------------------------------------------------
# Rollback Engine (LIFO Reversal)
# ------------------------------------------------------------------------------
tx_rollback() {
	if [[ ! -f "${TX_JOURNAL_FILE:-}" ]]; then
		log_warn "No active transaction journal found to rollback."
		return 0
	fi

	log_box "Transaction Rollback Triggered" \
		"Transaction ID: ${TX_ID}" \
		"Reversing mutations in reverse chronological order (LIFO)..."

	local reverted_count=0

	while IFS= read -r line; do
		case "$line" in
		FILE_CREATED:*)
			local file="${line#FILE_CREATED:}"
			if [[ -f "$file" ]]; then
				rm -f "$file"
				log_info "Reverted created file: ${file}"
				reverted_count=$((reverted_count + 1))
			fi
			;;
		SYMLINK_CREATED:*)
			local symlink="${line#SYMLINK_CREATED:}"
			if [[ -L "$symlink" ]]; then
				rm -f "$symlink"
				log_info "Removed created symlink: ${symlink}"
				reverted_count=$((reverted_count + 1))
			fi
			;;
		DIR_CREATED:*)
			local dir="${line#DIR_CREATED:}"
			if [[ -d "$dir" ]]; then
				rmdir "$dir" 2>/dev/null || true
				reverted_count=$((reverted_count + 1))
			fi
			;;
		MARKER_BLOCK:*)
			local payload="${line#MARKER_BLOCK:}"
			local target_file="${payload%%:*}"
			local block_id="${payload##*:}"
			if [[ -f "$target_file" ]]; then
				local start_m="# >>> arturonavax/env:${block_id} >>>"
				local end_m="# <<< arturonavax/env:${block_id} <<<"
				local tmp_f
				tmp_f="$(mktemp "${TMPDIR:-/tmp}/rb_block.XXXXXX")"
				awk -v start="$start_m" -v end="$end_m" '
					$0 == start { skip = 1; next }
					$0 == end { skip = 0; next }
					!skip { print }
				' "$target_file" >"$tmp_f"
				cat "$tmp_f" >"$target_file"
				rm -f "$tmp_f"
				log_info "Reverted marker block '${block_id}' from: ${target_file}"
				reverted_count=$((reverted_count + 1))
			fi
			;;
		esac
	done < <(_reverse_file "$TX_JOURNAL_FILE" 2>/dev/null || true)

	echo "ROLLBACK_COMPLETED: $(date +'%Y-%m-%d %H:%M:%S')" >>"$TX_JOURNAL_FILE"
	TX_ACTIVE=0

	# Restore snapshot if available and requested
	if [[ -n "${TX_INITIAL_SNAPSHOT:-}" ]] && [[ -d "$TX_INITIAL_SNAPSHOT" ]]; then
		log_step "Restoring initial configuration snapshot (${TX_INITIAL_SNAPSHOT})..."
		# Sourcing backup lib if available
		if command -v backup_restore >/dev/null 2>&1; then
			backup_restore "$(basename "$TX_INITIAL_SNAPSHOT")"
		fi
	fi

	log_success "Rollback completed. Reverted ${reverted_count} mutations. Host restored cleanly."
}

# ------------------------------------------------------------------------------
# Graceful Trap Signal Handler
# ------------------------------------------------------------------------------
tx_on_signal() {
	local sig_name="$1"
	local exit_code="$2"

	# Prevent recursive trap triggers
	trap - SIGINT SIGTERM

	printf '\n\n%s%s [INTERRUPTED]: Installation interrupted by %s (Ctrl+C)%s\n' \
		"$BOLD_RED" "$SYM_CROSS" "$sig_name" "$RESET" >&2
	printf '  %sTransaction ID: %s%s\n' "$YELLOW" "${TX_ID:-unknown}" "$RESET" >&2
	printf '  %sAudit journal:  %s%s\n\n' "$DIM" "${TX_JOURNAL_FILE:-none}" "$RESET" >&2

	# In interactive mode, ask operator before rolling back
	if [[ -t 0 ]] && [[ -t 1 ]] && ((TX_ACTIVE == 1)); then
		printf '  %s? Would you like to automatically rollback all changes made in this session? [Y/n]: %s' \
			"$BOLD_YELLOW" "$RESET" >&2
		local reply=""
		read -r reply || true
		reply="${reply:-y}"

		if [[ "$reply" =~ ^[Yy] ]]; then
			tx_rollback
		else
			log_warn "Rollback skipped by operator decision. Modified files kept."
		fi
	elif ((TX_ACTIVE == 1)); then
		# Non-interactive mode: always perform safe automatic rollback
		log_warn "Non-interactive interrupt caught. Performing automatic rollback..."
		tx_rollback
	fi

	exit "$exit_code"
}
