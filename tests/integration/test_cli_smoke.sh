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

# Test 6: ./install.sh --plan
plan_smoke="$("${REPO_ROOT}/install.sh" --plan 2>&1)"
plan_smoke_code=$?
assert_exit_code 0 "$plan_smoke_code" "./install.sh --plan exit status" &&
	assert_contains "$plan_smoke" "Installation Plan & Manifest Preview" "Plan header" &&
	pass "./install.sh --plan previewed installation manifest"

# Test 7: ./install.sh --uninstall-plan
uninst_plan_smoke="$("${REPO_ROOT}/install.sh" --uninstall-plan 2>&1)"
uninst_plan_smoke_code=$?
assert_exit_code 0 "$uninst_plan_smoke_code" "./install.sh --uninstall-plan exit status" &&
	assert_contains "$uninst_plan_smoke" "Uninstallation Plan & Teardown Preview" "Uninstall plan header" &&
	pass "./install.sh --uninstall-plan previewed uninstallation manifest"

# Test 8: ./uninstall.sh --plan and --help
uninst_smoke="$("${REPO_ROOT}/uninstall.sh" --plan 2>&1)"
uninst_smoke_code=$?
uninst_help="$("${REPO_ROOT}/uninstall.sh" --help 2>&1)"
uninst_help_code=$?
assert_exit_code 0 "$uninst_smoke_code" "./uninstall.sh --plan exit status" &&
	assert_exit_code 0 "$uninst_help_code" "./uninstall.sh --help exit status" &&
	assert_contains "$uninst_smoke" "Uninstallation Plan & Teardown Preview" "Uninstall plan header" &&
	assert_contains "$uninst_help" "Usage: ./uninstall.sh" "Uninstall usage string" &&
	pass "./uninstall.sh --plan and --help executed cleanly without side effects"

# Test 9: ./src/wizard.sh --help
wizard_help="$("${REPO_ROOT}/src/wizard.sh" --help 2>&1)"
wizard_help_code=$?
assert_exit_code 0 "$wizard_help_code" "./src/wizard.sh --help exit status" &&
	assert_contains "$wizard_help" "Usage: ./src/wizard.sh" "Wizard usage documentation" &&
	assert_contains "$wizard_help" "--non-interactive" "Non-interactive flag documentation" &&
	assert_contains "$wizard_help" "--rollback-on-error" "Rollback on error flag documentation" &&
	pass "./src/wizard.sh --help displayed comprehensive execution options"

# Test 10: ./src/wizard.sh --plan
wizard_plan="$("${REPO_ROOT}/src/wizard.sh" --plan --components=terminal,editor 2>&1)"
wizard_plan_code=$?
assert_exit_code 0 "$wizard_plan_code" "./src/wizard.sh --plan exit status" &&
	assert_contains "$wizard_plan" "Installation Plan & Manifest Preview" "Wizard plan header" &&
	assert_contains "$wizard_plan" "Ghostty" "Selected terminal component in plan" &&
	pass "./src/wizard.sh --plan previewed component manifest cleanly"

# Test 11: ./install.sh --wizard --help forwarding
install_wizard_help="$("${REPO_ROOT}/install.sh" --wizard --help 2>&1)"
install_wizard_help_code=$?
assert_exit_code 0 "$install_wizard_help_code" "./install.sh --wizard --help exit status" &&
	assert_contains "$install_wizard_help" "Usage: ./src/wizard.sh" "Forwarded wizard usage" &&
	pass "./install.sh --wizard forwarded CLI arguments directly to wizard runner"
