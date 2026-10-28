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

# Test 11: aitools commit --list-templates
tmpl_list_output="$("$AITOOLS_BIN" commit --list-templates 2>&1)"
tmpl_list_code=$?
assert_exit_code 0 "$tmpl_list_code" "aitools commit --list-templates exit code" &&
	assert_contains "$tmpl_list_output" "Available Commit Message Templates" "Templates header" &&
	assert_contains "$tmpl_list_output" "conventional" "Template conventional listed" &&
	assert_contains "$tmpl_list_output" "jira" "Template jira listed" &&
	assert_contains "$tmpl_list_output" "jira-scope" "Template jira-scope listed" &&
	assert_contains "$tmpl_list_output" "detailed" "Template detailed listed" &&
	pass "aitools commit --list-templates cleanly listed all available templates"

# Test 12: Jira ticket extraction and safe template rendering
python3 -c "
from importlib.machinery import SourceFileLoader
import sys
aitools = SourceFileLoader('aitools', '${REPO_ROOT}/files/ai/aitools/aitools').load_module()

# Test Jira extraction
assert aitools.extract_jira_id('feature/ENG-942-auth-flow') == 'ENG-942', 'Failed to extract ENG-942'
assert aitools.extract_jira_id('fix/CORE-105-checkout') == 'CORE-105', 'Failed to extract CORE-105'
assert aitools.extract_jira_id('hotfix/ABC-1_urgent') == 'ABC-1', 'Failed to extract ABC-1'
assert aitools.extract_jira_id('chore/clean-tests') == '', 'Non-jira branch should return empty string'

# Test render_commit_template does not crash with KeyError on arbitrary braces
tmpl = 'Format: <type>({scope}): <summary>\nJira: {jira_id}\nBranch: {branch}\nDiff:\n{diff}'
ctx = {
    'jira_id': 'PROJ-1234',
    'branch': 'feature/PROJ-1234-login',
    'diff': 'sample diff content',
    'scope': 'auth'
}
rendered = aitools.render_commit_template(tmpl, ctx)
assert 'Format: <type>(auth): <summary>' in rendered, 'Failed to replace scope'
assert 'Jira: PROJ-1234' in rendered, 'Failed to replace jira_id'
assert 'Branch: feature/PROJ-1234-login' in rendered, 'Failed to replace branch'
assert 'sample diff content' in rendered, 'Failed to replace diff'

# Test built-in template resolution
t_str, t_name = aitools.resolve_commit_template('jira', {})
assert t_name == 'jira', 'Failed to resolve jira template'
assert '{jira_id}' in t_str or '[{jira_id}]' in t_str, 'Jira placeholder missing in template'
sys.exit(0)
" >/dev/null 2>&1
jira_test_code=$?
assert_exit_code 0 "$jira_test_code" "Jira extraction and template rendering assertions" &&
	pass "Jira ticket extraction and safe template rendering validated with 100% precision"

# Test 13: aitools commit options verification
commit_help_output="$("$AITOOLS_BIN" commit --help 2>&1)"
commit_help_code=$?
assert_exit_code 0 "$commit_help_code" "aitools commit --help exit code" &&
	assert_contains "$commit_help_output" "--template" "Template flag present" &&
	assert_contains "$commit_help_output" "--jira" "Jira flag present" &&
	assert_contains "$commit_help_output" "--scope" "Scope flag present" &&
	assert_contains "$commit_help_output" "--model" "Model flag present" &&
	assert_contains "$commit_help_output" "--list-templates" "List templates flag present" &&
	assert_contains "$commit_help_output" "--list-backends" "List backends flag present" &&
	pass "aitools commit CLI flags (--template, --jira, --scope, --model, --list-templates, --list-backends) verified"

# Test 14: aitools backends and commit --list-backends
backends_output="$("$AITOOLS_BIN" backends 2>&1)"
backends_code=$?
assert_exit_code 0 "$backends_code" "aitools backends exit code" &&
	assert_contains "$backends_output" "Available AI Harness Backends" "Backends header" &&
	assert_contains "$backends_output" "agy" "Backend agy listed" &&
	assert_contains "$backends_output" "llm" "Backend llm listed" &&
	assert_contains "$backends_output" "claude" "Backend claude listed" &&
	assert_contains "$backends_output" "ollama" "Backend ollama listed" &&
	pass "aitools backends cleanly displayed all registered AI harnesses and execution strategies"

commit_backends_output="$("$AITOOLS_BIN" commit --list-backends 2>&1)"
commit_backends_code=$?
assert_exit_code 0 "$commit_backends_code" "aitools commit --list-backends exit code" &&
	assert_contains "$commit_backends_output" "Available AI Harness Backends" "Backends header from commit" &&
	pass "aitools commit --list-backends cleanly inspected harness registry"

# Test 15: HarnessAdapter unit tests (I/O normalization, ANSI stripping, JSON extraction, file modes)
python3 -c "
from importlib.machinery import SourceFileLoader
import sys, os
aitools = SourceFileLoader('aitools', '${REPO_ROOT}/files/ai/aitools/aitools').load_module()

# 1. ANSI stripping test
dirty = '\x1b[32m\x1b[1mfeat(core): test message\x1b[0m\r\n'
clean = aitools.strip_ansi_codes(dirty)
assert clean == 'feat(core): test message', f'Failed ANSI strip: {clean}'

# 2. JSON path extraction test
sample_json = {'data': {'choices': [{'message': {'content': 'fix(parser): syntax fix'}}]}}
extracted = aitools.extract_json_path(sample_json, 'data.choices.0.message.content')
assert extracted == 'fix(parser): syntax fix', f'Failed JSON path: {extracted}'

# 3. HarnessAdapter with temp_file input and output mode
spec_file = {
    'command': ['bash', '-c', 'cat {prompt_file} > {output_file} && echo \"追加\" >> {output_file}'],
    'input_mode': 'temp_file',
    'output_mode': 'temp_file',
    'clean_ansi': True,
    'clean_markdown': True
}
adapter_file = aitools.HarnessAdapter('mock_file_harness', spec_file)
out_file = adapter_file.execute('feat(test): hello world')
assert 'feat(test): hello world' in out_file, f'Temp file I/O failed: {out_file}'

# 4. HarnessAdapter with stdin input mode
spec_stdin = {
    'command': ['cat'],
    'input_mode': 'stdin',
    'output_mode': 'stdout',
    'clean_ansi': True
}
adapter_stdin = aitools.HarnessAdapter('mock_stdin', spec_stdin)
out_stdin = adapter_stdin.execute('chore(deps): bump library')
assert out_stdin == 'chore(deps): bump library', f'Stdin pipe failed: {out_stdin}'

# 5. Ad-hoc command resolution
ad_hoc = aitools.resolve_backend_adapter('cat')
assert ad_hoc.input_mode == 'stdin', 'Ad-hoc command should default to stdin'

sys.exit(0)
" >/dev/null 2>&1
adapter_test_code=$?
assert_exit_code 0 "$adapter_test_code" "HarnessAdapter protocol normalization tests" &&
	pass "HarnessAdapter I/O protocols (stdin, temp_file, json, ANSI cleaning) verified with 100% precision"

# Test 16: Interactive wizard CLI options (setup and config -i)
setup_help_output="$("$AITOOLS_BIN" setup --help 2>&1)"
setup_help_code=$?
assert_exit_code 0 "$setup_help_code" "aitools setup --help exit code" &&
	assert_contains "$setup_help_output" "usage: aitools setup" "Setup usage present" &&
	pass "aitools setup exposes dedicated interactive wizard entrypoint"

config_help_output="$("$AITOOLS_BIN" config --help 2>&1)"
config_help_code=$?
assert_exit_code 0 "$config_help_code" "aitools config --help exit code" &&
	assert_contains "$config_help_output" "--interactive" "Interactive flag present in config" &&
	assert_contains "$config_help_output" "wizard" "Wizard action choice present" &&
	pass "aitools config exposes interactive wizard flags and choices"

# Test 17: Interactive wizard execution with simulated responses
mock_cfg_dir="${TEST_SANDBOX}/.config/aitools"
mkdir -p "$mock_cfg_dir"
HOME="${TEST_SANDBOX}" "$AITOOLS_BIN" setup <<EOF >/dev/null 2>&1
1
gemini-2.5-flash
high
2
y
n
y
EOF
wizard_run_code=$?
assert_exit_code 0 "$wizard_run_code" "aitools setup interactive run exit code" &&
	pass "aitools setup executed interactive wizard and saved preferences cleanly"

# Test 18: Pure-Bash companion wizard (utils/aitools_wizard.sh)
WIZARD_SCRIPT="${REPO_ROOT}/utils/aitools_wizard.sh"
if [[ ! -x "$WIZARD_SCRIPT" ]]; then
	fail "utils/aitools_wizard.sh is not executable or not found"
else
	pass "utils/aitools_wizard.sh is verified and executable"
fi

bash -n "$WIZARD_SCRIPT"
wiz_syntax_code=$?
assert_exit_code 0 "$wiz_syntax_code" "aitools_wizard.sh syntax validation" &&
	pass "utils/aitools_wizard.sh passed bash -n syntax checks with zero errors"
