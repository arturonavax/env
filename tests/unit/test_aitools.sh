#!/usr/bin/env bash
# ==============================================================================
# tests/unit/test_aitools.sh
# Unit tests for aitools AI Productivity, Quota Management & Commit Suite.
# ==============================================================================

# shellcheck source=/dev/null
source "$(dirname "${BASH_SOURCE[0]}")/../test_helper.sh"

echo "  Running aitools AI Suite & CLI Tests..."

setup_sandbox
trap teardown_sandbox EXIT

AITOOLS_BIN="${REPO_ROOT}/files/ai/aitools/aitools"

# Test 1: aitools binary exists and is executable
if [[ ! -x "$AITOOLS_BIN" ]]; then
	fail "aitools binary not found or not executable at $AITOOLS_BIN"
else
	pass "aitools executable verified at $AITOOLS_BIN"
fi

# Test 2: Python syntax compilation
python3 -m py_compile "$AITOOLS_BIN" >/dev/null 2>&1
compile_code=$?
assert_exit_code 0 "$compile_code" "Python compilation of aitools" &&
	pass "aitools compiled cleanly with zero Python syntax errors"

# Test 3: aitools --help
help_output="$("$AITOOLS_BIN" --help 2>&1)"
help_code=$?
assert_exit_code 0 "$help_code" "aitools --help exit code" &&
	assert_contains "$help_output" "aitools: AI Productivity" "Help description header" &&
	assert_contains "$help_output" "quota" "Subcommand quota listed" &&
	assert_contains "$help_output" "watch" "Subcommand watch listed" &&
	assert_contains "$help_output" "wake" "Subcommand wake listed" &&
	assert_contains "$help_output" "commit" "Subcommand commit listed" &&
	assert_contains "$help_output" "daemon" "Subcommand daemon listed" &&
	assert_contains "$help_output" "status" "Subcommand status listed" &&
	pass "aitools --help displayed comprehensive CLI manual"

# Test 4: aitools status
status_output="$("$AITOOLS_BIN" status 2>&1)"
status_code=$?
assert_exit_code 0 "$status_code" "aitools status exit code" &&
	assert_contains "$status_output" "AI Ecosystem & CLI Status Diagnostics" "Status header" &&
	assert_contains "$status_output" "agy" "Status lists agy" &&
	assert_contains "$status_output" "claude" "Status lists claude" &&
	pass "aitools status verified local CLI tools and integrations"

# Test 5: aitools config path
config_path_output="$("$AITOOLS_BIN" config path 2>&1)"
config_path_code=$?
assert_exit_code 0 "$config_path_code" "aitools config path exit code" &&
	assert_contains "$config_path_output" ".config/aitools/config.json" "Config file path" &&
	pass "aitools config path returned expected configuration location"

# Test 6: Subcommand --help flags
for sub in quota watch wake commit daemon config; do
	sub_help="$("$AITOOLS_BIN" "$sub" --help 2>&1)"
	sub_code=$?
	assert_exit_code 0 "$sub_code" "aitools $sub --help exit code" &&
		assert_contains "$sub_help" "usage: aitools $sub" "Subcommand usage string"
done
pass "All aitools subcommands (quota, watch, wake, commit, daemon, config) expose clean help interfaces"

# Test 7: install_aitools.sh script syntax
bash -n "${REPO_ROOT}/src/ai/install_aitools.sh"
inst_syntax_code=$?
assert_exit_code 0 "$inst_syntax_code" "install_aitools.sh syntax validation" &&
	pass "install_aitools.sh validated cleanly with bash -n"
