#!/usr/bin/env bash
# ==============================================================================
# tests/unit/test_wizard_engine.sh
# Unit tests for pure-bash wizard engine, progress bars, and input prompts.
# ==============================================================================

# shellcheck source=/dev/null
source "$(dirname "${BASH_SOURCE[0]}")/../test_helper.sh"
# shellcheck source=/dev/null
source "${REPO_ROOT}/src/lib/wizard_engine.sh"

echo "  Running Pure-Bash Wizard Engine Tests..."

setup_sandbox
trap teardown_sandbox EXIT

# Test 1: Wizard Initialization
wizard_init "Test Wizard Title" 5 "Test Subtitle"
assert_equals "Test Wizard Title" "$WIZARD_TITLE" "Wizard title set" &&
	assert_equals "5" "$WIZARD_TOTAL_STAGES" "Total stages set" &&
	assert_equals "Test Subtitle" "$WIZARD_SUBTITLE" "Subtitle set" &&
	pass "wizard_init initialized framework state cleanly"

# Test 2: Graphical Progress Bar
bar_0="$(wizard_progress_bar 0 10 10)"
bar_50="$(wizard_progress_bar 5 10 10)"
bar_100="$(wizard_progress_bar 10 10 10)"

assert_contains "$bar_0" "0%" "0% progress text" &&
	assert_contains "$bar_50" "50%" "50% progress text" &&
	assert_contains "$bar_100" "100%" "100% progress text" &&
	pass "wizard_progress_bar calculated visual indicators and percentages correctly"

# Test 3: Environment Variable Upsert (wizard_write_env)
test_env="${TEST_SANDBOX}/.env.test"
wizard_write_env "APP_NAME" "EnvTest" "$test_env"
wizard_write_env "APP_PORT" "8080" "$test_env"

assert_file_exists "$test_env" "Generated dotenv file exists" &&
	assert_file_contains "$test_env" "APP_NAME='EnvTest'" "Contains APP_NAME" &&
	assert_file_contains "$test_env" "APP_PORT='8080'" "Contains APP_PORT" &&
	pass "wizard_write_env persisted configuration variables idempotently"

# Test 4: Environment Variable In-Place Update
wizard_write_env "APP_PORT" "9090" "$test_env"
app_port_count="$(grep -c "^APP_PORT=" "$test_env")"
assert_equals "1" "$app_port_count" "APP_PORT updated in-place without duplicating key" &&
	assert_file_contains "$test_env" "APP_PORT='9090'" "Contains updated APP_PORT" &&
	pass "wizard_write_env updated existing key in-place"

# Test 5: Ask helper with default value
(
	# Emulate pressing enter (empty input) to accept default
	echo "" | {
		ask TEST_VAL "Enter value" "my_default"
		assert_equals "my_default" "$TEST_VAL" "Default value accepted on empty input"
	}
) && pass "ask helper successfully accepted default value"
