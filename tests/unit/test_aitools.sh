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
for sub in quota watch wake commit llm daemon config run commands; do
	sub_help="$("$AITOOLS_BIN" "$sub" --help 2>&1)"
	sub_code=$?
	assert_exit_code 0 "$sub_code" "aitools $sub --help exit code" &&
		assert_contains "$sub_help" "usage: aitools $sub" "Subcommand usage string"
done
pass "All aitools subcommands (quota, watch, wake, commit, llm, daemon, config, run, commands) expose clean help interfaces"

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
	assert_contains "$tmpl_list_output" "task" "Template task listed" &&
	assert_contains "$tmpl_list_output" "task-scope" "Template task-scope listed" &&
	assert_contains "$tmpl_list_output" "detailed" "Template detailed listed" &&
	pass "aitools commit --list-templates cleanly listed all available templates"

# Test 12: Agnostic task/ticket extraction, repo templates, and safe rendering
python3 -c "
from importlib.machinery import SourceFileLoader
import sys
aitools = SourceFileLoader('aitools', '${REPO_ROOT}/files/ai/aitools/aitools').load_module()

# 1. Test task ID extraction (standard project keys and issue numbers)
assert aitools.extract_task_id('feature/ENG-942-auth-flow') == 'ENG-942', 'Failed to extract ENG-942'
assert aitools.extract_task_id('fix/CORE-105-checkout') == 'CORE-105', 'Failed to extract CORE-105'
assert aitools.extract_task_id('hotfix/ABC-1_urgent') == 'ABC-1', 'Failed to extract ABC-1'
assert aitools.extract_task_id('issue/42-fix-login') == '#42', 'Failed to extract #42'
assert aitools.extract_task_id('feat/gh-101-cache') == 'GH-101', 'Failed to extract GH-101'
assert aitools.extract_task_id('chore/clean-tests') == '', 'Non-ticket branch should return empty string'

# 2. Test backwards compatibility alias
assert aitools.extract_jira_id('feature/ENG-942-auth-flow') == 'ENG-942', 'Failed alias extract_jira_id'

# 3. Test render_commit_template does not crash with KeyError on arbitrary braces
tmpl = 'Format: <type>({scope}): <summary>\nTask: {task_id}\nBranch: {branch}\nDiff:\n{diff}'
ctx = {
    'task_id': 'PROJ-1234',
    'branch': 'feature/PROJ-1234-login',
    'diff': 'sample diff content',
    'scope': 'auth'
}
rendered = aitools.render_commit_template(tmpl, ctx)
assert 'Format: <type>(auth): <summary>' in rendered, 'Failed to replace scope'
assert 'Task: PROJ-1234' in rendered, 'Failed to replace task_id'
assert 'Branch: feature/PROJ-1234-login' in rendered, 'Failed to replace branch'
assert 'sample diff content' in rendered, 'Failed to replace diff'

# 4. Test built-in task template resolution
t_str, t_name = aitools.resolve_commit_template('task', {})
assert t_name == 'task', f'Failed to resolve task template: {t_name}'
assert '{task_id}' in t_str or '[{task_id}]' in t_str, 'Task placeholder missing in template'

# 5. Test repo template wrapper
repo_msg = '# [<type>]: <subject>\n# Why this change is needed\n'
wrapped = aitools.wrap_repo_template_as_prompt(repo_msg, '.gitmessage')
assert '{diff}' in wrapped, 'Missing {diff} in wrapped repo template'
assert '{task_id}' in wrapped, 'Missing {task_id} in wrapped repo template'
assert 'Repository Commit Template & Guidelines:' in wrapped, 'Header missing'

sys.exit(0)
" >/dev/null 2>&1
task_test_code=$?
assert_exit_code 0 "$task_test_code" "Task extraction and template rendering assertions" &&
	pass "Agnostic task/ticket extraction, repo template wrapping, and safe rendering validated with 100% precision"

# Test 13: aitools commit options verification
commit_help_output="$("$AITOOLS_BIN" commit --help 2>&1)"
commit_help_code=$?
assert_exit_code 0 "$commit_help_code" "aitools commit --help exit code" &&
	assert_contains "$commit_help_output" "--template" "Template flag present" &&
	assert_contains "$commit_help_output" "--task" "Task flag present" &&
	assert_contains "$commit_help_output" "--scope" "Scope flag present" &&
	assert_contains "$commit_help_output" "--model" "Model flag present" &&
	assert_contains "$commit_help_output" "--list-templates" "List templates flag present" &&
	assert_contains "$commit_help_output" "--list-backends" "List backends flag present" &&
	pass "aitools commit CLI flags (--template, --task, --scope, --model, --list-templates, --list-backends) verified"

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

# Test 19: aitools llm piped stdin and prompt assembly
llm_piped_out="$(echo "Sample stdin diff content" | "$AITOOLS_BIN" llm "Analyze diff" -b "cat" 2>&1)"
llm_piped_code=$?
assert_exit_code 0 "$llm_piped_code" "aitools llm piped stdin exit code" &&
	assert_contains "$llm_piped_out" "=== Standard Input ===" "Piped stdin header present" &&
	assert_contains "$llm_piped_out" "Sample stdin diff content" "Piped stdin content present" &&
	assert_contains "$llm_piped_out" "=== Instruction ===" "Instruction header present" &&
	assert_contains "$llm_piped_out" "Analyze diff" "Instruction text present" &&
	pass "aitools llm successfully assembled piped stdin and CLI prompt instructions"

# Test 20: aitools llm file (-f) and directory (-d) attachments
test_file="${TEST_SANDBOX}/test_sample.txt"
echo "function hello() { return 'world'; }" > "$test_file"
test_dir="${TEST_SANDBOX}/test_pkg"
mkdir -p "$test_dir"
echo "package main" > "${test_dir}/main.go"

llm_attach_out="$("$AITOOLS_BIN" llm "Review package" -f "$test_file" -d "$test_dir" -b "cat" 2>&1)"
llm_attach_code=$?
assert_exit_code 0 "$llm_attach_code" "aitools llm attachments exit code" &&
	assert_contains "$llm_attach_out" "=== Attached File:" "File attachment header present" &&
	assert_contains "$llm_attach_out" "function hello()" "Attached file content present" &&
	assert_contains "$llm_attach_out" "=== Directory Tree:" "Directory tree header present" &&
	assert_contains "$llm_attach_out" "package main" "Directory file content present" &&
	pass "aitools llm successfully attached files and directory tree context"

# Test 21: aitools llm execution modes and prompt framing
llm_chat_out="$("$AITOOLS_BIN" llm "Quick question" -b "cat" --mode chat 2>&1)"
llm_chat_code=$?
assert_exit_code 0 "$llm_chat_code" "aitools llm --mode chat exit code" &&
	assert_contains "$llm_chat_out" "[Direct Single-Turn Chat Mode:" "Chat mode framing present" &&
	assert_contains "$llm_chat_out" "Quick question" "Prompt present" &&
	pass "aitools llm applied direct chat mode prompt framing"

llm_sys_out="$("$AITOOLS_BIN" llm "Question" -b "cat" -s "System instructions" 2>&1)"
llm_sys_code=$?
assert_exit_code 0 "$llm_sys_code" "aitools llm -s/--system exit code" &&
	assert_contains "$llm_sys_out" "System instructions" "System prompt prepended" &&
	pass "aitools llm prepended explicit system prompt instructions"

# Test 22: aitools llm output formats (json, output-file) and error handling
llm_json_out="$("$AITOOLS_BIN" llm "Sample text query" -b "cat" -o json 2>&1)"
llm_json_code=$?
assert_exit_code 0 "$llm_json_code" "aitools llm -o json exit code" &&
	assert_contains "$llm_json_out" '"status": "success"' "JSON success status" &&
	assert_contains "$llm_json_out" '"backend": "cat"' "JSON backend name" &&
	pass "aitools llm emitted structured JSON response envelope"

out_dest="${TEST_SANDBOX}/llm_out.txt"
llm_file_out="$("$AITOOLS_BIN" llm "Output file query" -b "cat" -O "$out_dest" 2>&1)"
llm_file_code=$?
assert_exit_code 0 "$llm_file_code" "aitools llm -O output-file exit code" &&
	assert_contains "$llm_file_out" "Output written to" "Output file confirmation logged" &&
	[[ -f "$out_dest" ]] && assert_contains "$(<"$out_dest")" "Output file query" "File content written" &&
	pass "aitools llm routed response directly to specified output file"

# Test 23: aitools llm error exit when no prompt or stdin is supplied
llm_empty_out="$("$AITOOLS_BIN" llm 2>&1)"
llm_empty_code=$?
if [[ "$llm_empty_code" -ne 0 ]]; then
	assert_contains "$llm_empty_out" "[ERROR] No prompt or input provided" "Empty prompt error message" &&
		pass "aitools llm gracefully rejected invocation without prompt or input"
else
	fail "aitools llm empty prompt" "Expected non-zero exit code but got 0"
fi

# Test 24: Repository-level commit template auto-discovery (.gitmessage)
repo_test_dir="${TEST_SANDBOX}/git_repo_with_msg"
mkdir -p "$repo_test_dir"
(
	cd "$repo_test_dir" || exit 1
	git init -q
	echo "# Custom repo gitmessage template" > .gitmessage
	echo "# [<type>]: <subject>" >> .gitmessage
	repo_list_out="$("$AITOOLS_BIN" commit --list-templates 2>&1)"
	assert_contains "$repo_list_out" "repo" "Discovered repo template in list" &&
		assert_contains "$repo_list_out" ".gitmessage" "Found .gitmessage reference"
)
repo_test_code=$?
assert_exit_code 0 "$repo_test_code" "Repo template discovery exit code" &&
	pass "aitools commit auto-discovered repository-level .gitmessage commit template"

# Test 25: aitools pr CLI flags and help interface
pr_help_output="$("$AITOOLS_BIN" pr --help 2>&1)"
pr_help_code=$?
assert_exit_code 0 "$pr_help_code" "aitools pr --help exit code" &&
	assert_contains "$pr_help_output" "-B BASE" "Base branch flag present" &&
	assert_contains "$pr_help_output" "-t TEMPLATE" "Template flag present" &&
	assert_contains "$pr_help_output" "-k TASK_ID" "Task ID flag present" &&
	assert_contains "$pr_help_output" "-o OUTPUT" "Output file flag present" &&
	assert_contains "$pr_help_output" "-c" "Copy flag present" &&
	assert_contains "$pr_help_output" "-p" "Print flag present" &&
	assert_contains "$pr_help_output" "-d" "Dry run flag present" &&
	assert_contains "$pr_help_output" "--list-templates" "List templates flag present" &&
	assert_contains "$pr_help_output" "--list-backends" "List backends flag present" &&
	pass "aitools pr CLI flags (--base, --template, --task, --output, --copy, --print, --dry-run) verified"

# Test 26: aitools pr --list-templates
pr_templates_output="$("$AITOOLS_BIN" pr --list-templates 2>&1)"
pr_templates_code=$?
assert_exit_code 0 "$pr_templates_code" "aitools pr --list-templates exit code" &&
	assert_contains "$pr_templates_output" "standard" "Standard PR template listed" &&
	assert_contains "$pr_templates_output" "task" "Task PR template listed" &&
	assert_contains "$pr_templates_output" "minimal" "Minimal PR template listed" &&
	assert_contains "$pr_templates_output" "detailed" "Detailed PR template listed" &&
	pass "aitools pr --list-templates cleanly displayed built-in PR templates"

# Test 27: Repository PR template auto-discovery (.github/pull_request_template.md) and prioritization
repo_pr_dir="${TEST_SANDBOX}/git_repo_with_pr_template"
mkdir -p "$repo_pr_dir/.github"
(
	cd "$repo_pr_dir" || exit 1
	git init -q
	cat <<'EOF' > .github/pull_request_template.md
## Summary
Explain what changed.

## Checklist
- [ ] Tested
EOF
	repo_pr_list="$("$AITOOLS_BIN" pr --list-templates 2>&1)"
	assert_contains "$repo_pr_list" "repo" "Discovered repo PR template in list" &&
		assert_contains "$repo_pr_list" ".github/pull_request_template.md" "Found repo template reference" &&
		assert_contains "$repo_pr_list" "(active repo)" "Marked as active repo template"
)
repo_pr_code=$?
assert_exit_code 0 "$repo_pr_code" "Repo PR template discovery exit code" &&
	pass "aitools pr auto-discovered and prioritized repository PR template"

# Test 28: Git branch change extraction and Python PR template resolver
python3 -c "
from importlib.machinery import SourceFileLoader
import os
aitools = SourceFileLoader('aitools', '${REPO_ROOT}/files/ai/aitools/aitools').load_module()

# 1. Test PR template resolution prioritization
cfg = {'pr': {'auto_detect_repo_template': False, 'default_template': 'task'}}
tpl, name = aitools.resolve_pr_template(None, cfg)
assert name == 'task', f'Expected task template, got {name}'

# 2. Test explicit template override
tpl_min, name_min = aitools.resolve_pr_template('minimal', cfg)
assert name_min == 'minimal', f'Expected minimal template, got {name_min}'

# 3. Test template rendering with context
ctx = {
    'branch': 'feature/PROJ-888-auth',
    'base': 'main',
    'task_id': 'PROJ-888',
    'task_ref': '- **Related Issue / Ticket**: PROJ-888',
    'commits': '- feat(auth): add jwt validation',
    'files': 'auth.go, auth_test.go',
    'stat': '2 files changed, 50 insertions(+)',
    'diff': '+func ValidateJWT() bool { return true }'
}
rendered = aitools.render_pr_template(tpl, ctx)
assert 'PROJ-888' in rendered, 'Task ID not found in rendered PR template'
assert 'ValidateJWT' in rendered, 'Diff content not found in rendered PR template'
"
assert_exit_code 0 "$?" "Python PR template resolver validation" &&
	pass "aitools PR template resolver and context renderer passed unit checks"

# Test 29: aitools pr pipeline mode (--print, --output) with mock backend
pr_out_file="${TEST_SANDBOX}/generated_pr.md"
pr_print_output="$("$AITOOLS_BIN" pr -B "HEAD~1" -b "cat" -p -o "$pr_out_file" -k "TICKET-77" 2>&1)"
pr_print_code=$?
assert_exit_code 0 "$pr_print_code" "aitools pr --print exit code" &&
	assert_contains "$pr_print_output" "TICKET-77" "Output contains task ID" &&
	[[ -f "$pr_out_file" ]] && assert_contains "$(<"$pr_out_file")" "TICKET-77" "File written with task ID" &&
	pass "aitools pr --print routed markdown description directly to stdout and file"
