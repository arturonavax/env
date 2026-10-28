#!/usr/bin/env bash
# ==============================================================================
# tests/integration/test_cli_smoke.sh
# Smoke tests for top-level entry points and command-line utilities.
# ==============================================================================

# shellcheck source=/dev/null
source "$(dirname "${BASH_SOURCE[0]}")/../test_helper.sh"

echo "  Running CLI Smoke & Integration Tests..."

# Test 1: ./install.sh --check
check_output="$("${REPO_ROOT}/install.sh" --check 2>&1)"
check_code=$?
assert_exit_code 0 "$check_code" "./install.sh --check exit status" &&
	assert_contains "$check_output" "System Hardware & Platform Profile" "Profile header" &&
	pass "./install.sh --check executed cleanly with platform diagnosis"

# Test 2: ./install.sh --help
help_output="$("${REPO_ROOT}/install.sh" --help 2>&1)"
help_code=$?
assert_exit_code 0 "$help_code" "./install.sh --help exit status" &&
	assert_contains "$help_output" "Usage: ./install.sh" "Usage string" &&
	pass "./install.sh --help displayed usage documentation"

# Test 3: ./utils/restore_backup.sh --help
restore_help="$("${REPO_ROOT}/utils/restore_backup.sh" --help 2>&1)"
restore_help_code=$?
assert_exit_code 0 "$restore_help_code" "./utils/restore_backup.sh --help exit status" &&
	assert_contains "$restore_help" "Usage: ./utils/restore_backup.sh" "Usage string" &&
	pass "./utils/restore_backup.sh --help displayed usage documentation"

# Test 4: ./utils/restore_backup.sh -l
list_output="$("${REPO_ROOT}/utils/restore_backup.sh" -l 2>&1)"
list_code=$?
assert_exit_code 0 "$list_code" "./utils/restore_backup.sh -l exit status" &&
	assert_contains "$list_output" "Available Configuration Snapshots" "Snapshots header" &&
	pass "./utils/restore_backup.sh -l listed existing backup snapshots"

# Test 5: ./utils/sync_config.sh help
sync_help="$("${REPO_ROOT}/utils/sync_config.sh" help 2>&1)"
sync_code=$?
assert_exit_code 0 "$sync_code" "./utils/sync_config.sh help exit status" &&
	assert_contains "$sync_help" "[Sync Config]: List of Sync" "Sync config header" &&
	pass "./utils/sync_config.sh help displayed sync options"
