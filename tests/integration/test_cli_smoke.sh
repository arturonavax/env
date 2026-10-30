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
	assert_contains "$check_output" "ANX Health Doctor" "Profile header" &&
	pass "./install.sh --check executed cleanly with platform diagnosis"

# Test 2: ./install.sh --help
help_output="$("${REPO_ROOT}/install.sh" --help 2>&1)"
help_code=$?
assert_exit_code 0 "$help_code" "./install.sh --help exit status" &&
	assert_contains "$help_output" "Usage: ./install.sh" "Usage string" &&
	pass "./install.sh --help displayed usage documentation"

# Test 3: ./install.sh --plan
plan_smoke="$("${REPO_ROOT}/install.sh" --plan 2>&1)"
plan_smoke_code=$?
assert_exit_code 0 "$plan_smoke_code" "./install.sh --plan exit status" &&
	assert_contains "$plan_smoke" "Execution Plan:" "Plan header" &&
	pass "./install.sh --plan previewed installation manifest"

# Test 4: ./install.sh --uninstall-plan
uninst_plan_smoke="$("${REPO_ROOT}/install.sh" --uninstall-plan 2>&1)"
uninst_plan_smoke_code=$?
assert_exit_code 0 "$uninst_plan_smoke_code" "./install.sh --uninstall-plan exit status" &&
	assert_contains "$uninst_plan_smoke" "Uninstallation Plan" "Uninstall plan header" &&
	pass "./install.sh --uninstall-plan previewed uninstallation manifest"

# Test 5: ./uninstall.sh --plan and --help
uninst_smoke="$("${REPO_ROOT}/uninstall.sh" --plan 2>&1)"
uninst_smoke_code=$?
uninst_help="$("${REPO_ROOT}/uninstall.sh" --help 2>&1)"
uninst_help_code=$?
assert_exit_code 0 "$uninst_smoke_code" "./uninstall.sh --plan exit status" &&
	assert_exit_code 0 "$uninst_help_code" "./uninstall.sh --help exit status" &&
	assert_contains "$uninst_smoke" "Uninstallation Plan & Teardown Preview" "Uninstall plan header" &&
	assert_contains "$uninst_help" "Usage: ./uninstall.sh" "Uninstall usage string" &&
	pass "./uninstall.sh --plan and --help executed cleanly without side effects"

# Test 6: ANX binary commands smoke test (plan, doctor, status, backup, sync, tools)
anx_bin="${REPO_ROOT}/bin/anx"
if [[ ! -x "$anx_bin" ]]; then
	go build -o "$anx_bin" "${REPO_ROOT}/cmd/anx"
fi

anx_plan_out="$("$anx_bin" plan 2>&1)"
anx_plan_code=$?
assert_exit_code 0 "$anx_plan_code" "anx plan exit status" &&
	assert_contains "$anx_plan_out" "Execution Plan:" "ANX plan header" &&
	pass "anx plan executed cleanly"

anx_status_out="$("$anx_bin" status 2>&1)"
anx_status_code=$?
assert_exit_code 0 "$anx_status_code" "anx status exit status" &&
	assert_contains "$anx_status_out" "OS:" "ANX status output" &&
	pass "anx status executed cleanly"

anx_tools_out="$("$anx_bin" tools list 2>&1)"
anx_tools_code=$?
assert_exit_code 0 "$anx_tools_code" "anx tools list exit status" &&
	assert_contains "$anx_tools_out" "Tool" "ANX tools list table" &&
	pass "anx tools list displayed personal tools registry"

anx_ai_help="$("$anx_bin" tools ai --help 2>&1)"
anx_ai_code=$?
assert_exit_code 0 "$anx_ai_code" "anx tools ai --help exit status" &&
	assert_contains "$anx_ai_help" "Multi-harness AI CLI suite" "AI suite description" &&
	pass "anx tools ai displayed AI command suite"

# Test 7: aitools standalone binary
aitools_bin="${REPO_ROOT}/bin/aitools"
if [[ ! -x "$aitools_bin" ]]; then
	go build -o "$aitools_bin" "${REPO_ROOT}/cmd/aitools"
fi

aitools_smoke="$("$aitools_bin" --help 2>&1)"
aitools_smoke_code=$?
assert_exit_code 0 "$aitools_smoke_code" "aitools --help exit status" &&
	assert_contains "$aitools_smoke" "Multi-harness AI CLI suite" "aitools description" &&
	assert_contains "$aitools_smoke" "commit" "aitools commit command" &&
	assert_contains "$aitools_smoke" "quota" "aitools quota command" &&
	pass "aitools CLI verified with full subcommands suite"
