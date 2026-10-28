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
for sub in quota watch wake commit daemon config run commands; do
	sub_help="$("$AITOOLS_BIN" "$sub" --help 2>&1)"
	sub_code=$?
	assert_exit_code 0 "$sub_code" "aitools $sub --help exit code" &&
		assert_contains "$sub_help" "usage: aitools $sub" "Subcommand usage string"
done
pass "All aitools subcommands (quota, watch, wake, commit, daemon, config, run, commands) expose clean help interfaces"

# Test 7: install_aitools.sh script syntax
bash -n "${REPO_ROOT}/src/ai/install_aitools.sh"
inst_syntax_code=$?
assert_exit_code 0 "$inst_syntax_code" "install_aitools.sh syntax validation" &&
	pass "install_aitools.sh validated cleanly with bash -n"

# Test 8: aitools commands inspector displays built-in and declarative pipelines
commands_output="$("$AITOOLS_BIN" commands 2>&1)"
commands_code=$?
assert_exit_code 0 "$commands_code" "aitools commands exit code" &&
	assert_contains "$commands_output" "Available Commands & Declarative AI Pipelines" "Header output" &&
	assert_contains "$commands_output" "review" "Review declarative command listed" &&
	assert_contains "$commands_output" "explain" "Explain declarative command listed" &&
	assert_contains "$commands_output" "ask" "Ask declarative command listed" &&
	assert_contains "$commands_output" "optimize" "Optimize declarative command listed" &&
	pass "aitools commands displayed built-in and declarative custom pipelines"

# Test 9: JSONC parsing of config.template.jsonc with comments and trailing commas
python3 -c "
from importlib.machinery import SourceFileLoader
import json, sys
aitools = SourceFileLoader('aitools', '${REPO_ROOT}/files/ai/aitools/aitools').load_module()
with open('${REPO_ROOT}/files/ai/aitools/config.template.jsonc', 'r', encoding='utf-8') as f:
    clean = aitools.strip_jsonc_comments(f.read())
data = json.loads(clean)
assert 'commit' in data, 'Missing commit section'
assert 'commands' in data, 'Missing commands section'
assert 'review' in data['commands'], 'Missing review command'
assert len(data['targets']) >= 5, 'Insufficient targets'
sys.exit(0)
" >/dev/null 2>&1
jsonc_exit_code=$?
assert_exit_code 0 "$jsonc_exit_code" "JSONC parsing of config.template.jsonc" &&
	pass "JSONC comment stripper successfully parsed config.template.jsonc"

# Test 10: Declarative command execution error handling
unknown_cmd_output="$("$AITOOLS_BIN" run non_existent_cmd 2>&1)"
unknown_cmd_code=$?
assert_exit_code 1 "$unknown_cmd_code" "Unknown command should fail with exit code 1" &&
	assert_contains "$unknown_cmd_output" "Unknown custom command 'non_existent_cmd'" "Error message check" &&
	pass "Unknown custom command cleanly caught and reported"

ask_missing_args="$("$AITOOLS_BIN" run ask 2>&1)"
ask_missing_code=$?
assert_exit_code 1 "$ask_missing_code" "ask without args should fail with exit code 1" &&
	assert_contains "$ask_missing_args" "requires arguments" "Usage hint check" &&
	pass "Custom command argument requirements properly enforced"

direct_missing_args="$("$AITOOLS_BIN" ask 2>&1)"
direct_missing_code=$?
assert_exit_code 1 "$direct_missing_code" "direct ask without args should fail with exit code 1" &&
	assert_contains "$direct_missing_args" "requires arguments" "Direct usage hint check" &&
	pass "Direct custom command dispatch without 'run' prefix properly verified"
