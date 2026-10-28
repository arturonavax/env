#!/usr/bin/env bash
# ==============================================================================
# src/lib/logging.sh
# Structured, production-grade logging, status spinners and diagnostic reports.
# Part of arturonavax/env development environment.
# ==============================================================================

# Prevent double sourcing
[[ -n "${_LOGGING_SH_LOADED:-}" ]] && return 0
_LOGGING_SH_LOADED=1

# Resolve colors
_LIB_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
if [[ -f "${_LIB_DIR}/colors.sh" ]]; then
	# shellcheck source=/dev/null
	source "${_LIB_DIR}/colors.sh"
fi

# Configuration Defaults
ENV_LOG_FILE="${ENV_LOG_FILE:-$HOME/.arturonavax-env-install.log}"
ENV_DEBUG="${ENV_DEBUG:-0}"

# Internal helper to format log entry with timestamp
_log_timestamp() {
	date '+%Y-%m-%d %H:%M:%S'
}

_write_logfile() {
	local level="$1"
	shift
	if [[ -n "${ENV_LOG_FILE:-}" ]]; then
		mkdir -p "$(dirname "$ENV_LOG_FILE")" 2>/dev/null || true
		echo "[$(_log_timestamp)] [$level] $*" >>"$ENV_LOG_FILE" 2>/dev/null || true
	fi
}

log_info() {
	local msg="$*"
	printf '  %s%s%s %s\n' "$CYAN" "$SYM_INFO" "$RESET" "$msg" >&2
	_write_logfile "INFO" "$msg"
}

log_step() {
	local msg="$*"
	printf '  %s%s%s %s\n' "$BOLD_BLUE" "$SYM_ARROW" "$RESET" "$msg" >&2
	_write_logfile "STEP" "$msg"
}

log_success() {
	local msg="$*"
	printf '  %s%s%s %s%s%s\n' "$BOLD_GREEN" "$SYM_CHECK" "$RESET" "$BOLD_WHITE" "$msg" "$RESET" >&2
	_write_logfile "SUCCESS" "$msg"
}

log_warn() {
	local msg="$*"
	printf '  %s%s [WARNING]:%s %s%s%s\n' "$BOLD_YELLOW" "$SYM_WARN" "$RESET" "$YELLOW" "$msg" "$RESET" >&2
	_write_logfile "WARN" "$msg"
}

log_error() {
	local msg="$*"
	printf '  %s%s [ERROR]:%s %s%s%s\n' "$BOLD_RED" "$SYM_CROSS" "$RESET" "$BOLD_RED" "$msg" "$RESET" >&2
	_write_logfile "ERROR" "$msg"
}

log_fatal() {
	local msg="$1"
	local code="${2:-1}"
	printf '\n%s%s [FATAL]:%s %s%s%s\n' "$BOLD_RED" "$SYM_CROSS" "$RESET" "$BOLD_WHITE" "$msg" "$RESET" >&2
	_write_logfile "FATAL" "$msg (exit code: $code)"
	exit "$code"
}

log_debug() {
	if [[ "$ENV_DEBUG" == "1" ]] || [[ "${DEBUG:-0}" == "1" ]]; then
		local msg="$*"
		printf '  %s[DEBUG]: %s%s\n' "$GRAY" "$msg" "$RESET" >&2
		_write_logfile "DEBUG" "$msg"
	fi
}

log_separator() {
	printf '%s----------------------------------------------------------------------%s\n' "$DIM" "$RESET" >&2
}

log_box() {
	local title="$1"
	shift
	printf '\n%s%s╭────────────────────────────────────────────────────────────────────╮%s\n' "$BOLD" "$BLUE" "$RESET" >&2
	printf '%s│%s  %-64s  %s%s│%s\n' "$BOLD_BLUE" "$BOLD_WHITE" "$title" "$RESET" "$BOLD_BLUE" "$RESET" >&2
	printf '%s├────────────────────────────────────────────────────────────────────┤%s\n' "$BOLD_BLUE" "$RESET" >&2
	while (("$#")); do
		printf '%s│%s  %-64s  %s│%s\n' "$BOLD_BLUE" "$RESET" "$1" "$BOLD_BLUE" "$RESET" >&2
		shift
	done
	printf '%s╰────────────────────────────────────────────────────────────────────╯%s\n\n' "$BOLD_BLUE" "$RESET" >&2
}
