#!/usr/bin/env bash
# ==============================================================================
# tests/unit/test_strict.sh
# Unit tests for strict mode, cleanup handlers, and temp file registration.
# ==============================================================================

# shellcheck source=/dev/null
source "$(dirname "${BASH_SOURCE[0]}")/../test_helper.sh"

echo "  Running Strict Mode & Defensive Runtime Tests..."

setup_sandbox
trap teardown_sandbox EXIT

# Test 1: Subshell Temp File Auto-Purge on Exit
tmp_sentinel="${TEST_SANDBOX}/auto_cleanup_sentinel.txt"
touch "$tmp_sentinel"

(
	source "${REPO_ROOT}/src/lib/strict.sh"
	register_temp_file "$tmp_sentinel"
	exit 0
)

if [[ ! -e "$tmp_sentinel" ]]; then
	pass "register_temp_file automatically cleaned up temporary sentinel upon exit"
else
	fail "register_temp_file failed to remove registered temporary file"
fi

# Test 2: Subshell Cleanup Hook Execution
hook_marker="${TEST_SANDBOX}/cleanup_hook_ran.txt"

(
	source "${REPO_ROOT}/src/lib/strict.sh"
	register_cleanup_hook "touch '$hook_marker'"
	exit 0
)

if [[ -f "$hook_marker" ]]; then
	pass "register_cleanup_hook executed registered hook upon process termination"
else
	fail "register_cleanup_hook failed to execute registered hook"
fi
