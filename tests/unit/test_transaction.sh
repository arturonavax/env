#!/usr/bin/env bash
# ==============================================================================
# tests/unit/test_transaction.sh
# Unit tests for transactional execution, journal logging, and rollback engine.
# ==============================================================================

# shellcheck source=/dev/null
source "$(dirname "${BASH_SOURCE[0]}")/../test_helper.sh"
# shellcheck source=/dev/null
source "${REPO_ROOT}/src/lib/transaction.sh"

echo "  Running Transactional Execution & Rollback Tests..."

setup_sandbox
trap teardown_sandbox EXIT

export TX_LOG_DIR="${TEST_SANDBOX}/tx_logs"
mkdir -p "$TX_LOG_DIR"

# Test 1: Transaction Initialization
tx_begin "Test Session" ""
assert_equals "1" "$TX_ACTIVE" "Transaction is active" &&
	assert_file_exists "$TX_JOURNAL_FILE" "Journal file created" &&
	assert_file_contains "$TX_JOURNAL_FILE" "BEGIN: Test Session" "Journal contains begin marker" &&
	pass "tx_begin initialized transaction state and journal file"

# Test 2: Mutation Recording & LIFO Rollback
dummy_file="${TEST_SANDBOX}/created_file.txt"
dummy_link="${TEST_SANDBOX}/created_link.txt"
dummy_rc="${TEST_SANDBOX}/user_shell.rc"

# Create targets
echo "important content" >"$dummy_file"
ln -s "$dummy_file" "$dummy_link"
echo "alias my_custom=true" >"$dummy_rc"
{
	echo "# >>> arturonavax/env:test-block >>>"
	echo "managed line"
	echo "# <<< arturonavax/env:test-block <<<"
} >>"$dummy_rc"

# Record operations in transaction
tx_record_file_created "$dummy_file"
tx_record_symlink_created "$dummy_link"
tx_record_marker_block "$dummy_rc" "test-block"

# Trigger Rollback
tx_rollback

# Verify Rollback State
assert_file_contains "$dummy_rc" "alias my_custom=true" "User custom line preserved after rollback" &&
	pass "tx_rollback preserved custom user configuration"

if [[ -f "$dummy_file" ]]; then
	fail "dummy_file was not deleted during rollback"
else
	pass "tx_rollback reverted created file"
fi

if [[ -L "$dummy_link" ]]; then
	fail "dummy_link was not removed during rollback"
else
	pass "tx_rollback removed created symlink"
fi

if grep -qF "managed line" "$dummy_rc"; then
	fail "Marker block was not stripped during rollback"
else
	pass "tx_rollback stripped marker block cleanly"
fi

# Test 3: Transaction Commit
tx_begin "Commit Session" ""
tx_commit
assert_equals "0" "$TX_ACTIVE" "Transaction disarmed after commit" &&
	assert_file_contains "$TX_JOURNAL_FILE" "COMMIT:" "Journal contains commit entry" &&
	pass "tx_commit finalized transaction lifecycle cleanly"
